package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

// Minimal schema structs sufficient to parse the YAML and generate Go.
type Spec struct {
	Version string `yaml:"version"`
	Name    string `yaml:"name"`
	Rules   []any  `yaml:"rules"`
	Default any    `yaml:"default"`
}

func readYAML(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Name == "" {
		return nil, errors.New("rules name is empty")
	}
	return &s, nil
}

func generateFunction(spec *Spec, pkg string) ([]byte, error) {
	// Render code blocks selected and ordered by rule names in spec.Rules
	var pre []string
	var typeCases []string
	var post []string
	openTypeSwitch := false

	emit := func(name string) {
		switch name {
		case "early-never":
			pre = append(pre, "\tif subType == NeverType {\n\t\treturn true\n\t}\n\n")
		case "super-singletons":
			pre = append(pre, "\tswitch superType {\n\tcase AnyType:\n\t\treturn true\n\n\tcase AnyStructType:\n\t\tif subType.IsResourceType() {\n\t\t\treturn false\n\t\t}\n\t\treturn subType != AnyType\n\n\tcase AnyResourceType:\n\t\treturn subType.IsResourceType()\n\n\tcase AnyResourceAttachmentType:\n\t\treturn subType.IsResourceType() && isAttachmentType(subType)\n\n\tcase AnyStructAttachmentType:\n\t\treturn !subType.IsResourceType() && isAttachmentType(subType)\n\n\tcase HashableStructType:\n\t\treturn IsHashableStructType(subType)\n\n\tcase PathType:\n\t\treturn IsSubType(subType, StoragePathType) ||\n\t\t\tIsSubType(subType, CapabilityPathType)\n\n\tcase StorableType:\n\t\tstorableResults := map[*Member]bool{}\n\t\treturn subType.IsStorable(storableResults)\n\n\tcase CapabilityPathType:\n\t\treturn IsSubType(subType, PrivatePathType) ||\n\t\t\tIsSubType(subType, PublicPathType)\n\n\tcase NumberType:\n\t\tswitch subType {\n\t\tcase NumberType, SignedNumberType:\n\t\t\treturn true\n\t\t}\n\n\t\treturn IsSubType(subType, IntegerType) ||\n\t\t\tIsSubType(subType, FixedPointType)\n\n\tcase SignedNumberType:\n\t\tif subType == SignedNumberType {\n\t\t\treturn true\n\t\t}\n\n\t\treturn IsSubType(subType, SignedIntegerType) ||\n\t\t\tIsSubType(subType, SignedFixedPointType)\n\n\tcase IntegerType:\n\t\tswitch subType {\n\t\tcase IntegerType, SignedIntegerType, FixedSizeUnsignedIntegerType,\n\t\t\tUIntType:\n\n\t\t\treturn true\n\n\t\tdefault:\n\t\t\treturn IsSubType(subType, SignedIntegerType) || IsSubType(subType, FixedSizeUnsignedIntegerType)\n\t\t}\n\n\tcase SignedIntegerType:\n\t\tswitch subType {\n\t\tcase SignedIntegerType,\n\t\t\tIntType,\n\t\t\tInt8Type, Int16Type, Int32Type, Int64Type, Int128Type, Int256Type:\n\n\t\t\treturn true\n\n\t\tdefault:\n\t\t\treturn false\n\t\t}\n\n\tcase FixedSizeUnsignedIntegerType:\n\t\tswitch subType {\n\t\tcase UInt8Type, UInt16Type, UInt32Type, UInt64Type, UInt128Type, UInt256Type,\n\t\t\tWord8Type, Word16Type, Word32Type, Word64Type, Word128Type, Word256Type:\n\n\t\t\treturn true\n\n\t\tdefault:\n\t\t\treturn false\n\t\t}\n\n\tcase FixedPointType:\n\t\tswitch subType {\n\t\tcase FixedPointType,\n\t\t\tSignedFixedPointType,\n\t\t\tUFix64Type,\n\t\t\tUFix128Type:\n\n\t\t\treturn true\n\n\t\tdefault:\n\t\t\treturn IsSubType(subType, SignedFixedPointType)\n\t\t}\n\n\tcase SignedFixedPointType:\n\t\tswitch subType {\n\t\tcase SignedFixedPointType,\n\t\t\tFix64Type,\n\t\t\tFix128Type:\n\t\t\treturn true\n\n\t\tdefault:\n\t\t\treturn false\n\t\t}\n\t}\n\n")
		case "optional", "dictionary", "variable-sized-array", "constant-sized-array", "reference", "function", "intersection-with-any-legacy", "composite-super", "interface-super", "parameterized":
			if !openTypeSwitch {
				pre = append(pre, "\tswitch typedSuperType := superType.(type) {\n")
				openTypeSwitch = true
			}
			// emit the specific case arm
			switch name {
			case "optional":
				typeCases = append(typeCases, "\tcase *OptionalType:\n\t\toptionalSubType, ok := subType.(*OptionalType)\n\t\tif !ok {\n\t\t\treturn IsSubType(subType, typedSuperType.Type)\n\t\t}\n\t\treturn IsSubType(optionalSubType.Type, typedSuperType.Type)\n\n")
			case "dictionary":
				typeCases = append(typeCases, "\tcase *DictionaryType:\n\t\ttypedSubType, ok := subType.(*DictionaryType)\n\t\tif !ok {\n\t\t\treturn false\n\t\t}\n\n\t\treturn IsSubType(typedSubType.KeyType, typedSuperType.KeyType) &&\n\t\t\tIsSubType(typedSubType.ValueType, typedSuperType.ValueType)\n\n")
			case "variable-sized-array":
				typeCases = append(typeCases, "\tcase *VariableSizedType:\n\t\ttypedSubType, ok := subType.(*VariableSizedType)\n\t\tif !ok {\n\t\t\treturn false\n\t\t}\n\n\t\treturn IsSubType(\n\t\t\ttypedSubType.ElementType(false),\n\t\t\ttypedSuperType.ElementType(false),\n\t\t)\n\n")
			case "constant-sized-array":
				typeCases = append(typeCases, "\tcase *ConstantSizedType:\n\t\ttypedSubType, ok := subType.(*ConstantSizedType)\n\t\tif !ok {\n\t\t\treturn false\n\t\t}\n\n\t\tif typedSubType.Size != typedSuperType.Size {\n\t\t\treturn false\n\t\t}\n\n\t\treturn IsSubType(\n\t\t\ttypedSubType.ElementType(false),\n\t\t\ttypedSuperType.ElementType(false),\n\t\t)\n\n")
			case "reference":
				typeCases = append(typeCases, "\tcase *ReferenceType:\n\t\ttypedSubType, ok := subType.(*ReferenceType)\n\t\tif !ok {\n\t\t\treturn false\n\t\t}\n\n\t\tif !typedSuperType.Authorization.PermitsAccess(typedSubType.Authorization) {\n\t\t\treturn false\n\t\t}\n\n\t\treturn IsSubType(typedSubType.Type, typedSuperType.Type)\n\n")
			case "function":
				typeCases = append(typeCases, "\tcase *FunctionType:\n\t\ttypedSubType, ok := subType.(*FunctionType)\n\t\tif !ok {\n\t\t\treturn false\n\t\t}\n\n\t\tif typedSubType.Purity != typedSuperType.Purity && typedSubType.Purity != FunctionPurityView {\n\t\t\treturn false\n\t\t}\n\n\t\tif len(typedSubType.TypeParameters) != len(typedSuperType.TypeParameters) {\n\t\t\treturn false\n\t\t}\n\n\t\tfor i, subTypeParameter := range typedSubType.TypeParameters {\n\t\t\tsuperTypeParameter := typedSuperType.TypeParameters[i]\n\t\t\tif !subTypeParameter.TypeBoundEqual(superTypeParameter.TypeBound) {\n\t\t\t\treturn false\n\t\t\t}\n\t\t}\n\n\t\tif len(typedSubType.Parameters) != len(typedSuperType.Parameters) {\n\t\t\treturn false\n\t\t}\n\n\t\tif !typedSubType.ArityEqual(typedSuperType.Arity) {\n\t\t\treturn false\n\t\t}\n\n\t\tfor i, subParameter := range typedSubType.Parameters {\n\t\t\tsuperParameter := typedSuperType.Parameters[i]\n\t\t\tif !IsSubType(\n\t\t\t\tsuperParameter.TypeAnnotation.Type,\n\t\t\t\tsubParameter.TypeAnnotation.Type,\n\t\t\t) {\n\t\t\t\treturn false\n\t\t\t}\n\t\t}\n\n\t\tif typedSubType.ReturnTypeAnnotation.Type != nil {\n\t\t\tif typedSuperType.ReturnTypeAnnotation.Type == nil {\n\t\t\t\treturn false\n\t\t\t}\n\n\t\t\tif !IsSubType(\n\t\t\t\ttypedSubType.ReturnTypeAnnotation.Type,\n\t\t\t\ttypedSuperType.ReturnTypeAnnotation.Type,\n\t\t\t) {\n\t\t\t\treturn false\n\t\t\t}\n\t\t} else if typedSuperType.ReturnTypeAnnotation.Type != nil {\n\t\t\treturn false\n\t\t}\n\n\t\tif typedSubType.IsConstructor != typedSuperType.IsConstructor {\n\t\t\treturn false\n\t\t}\n\n\t\treturn true\n\n")
			case "intersection-with-any-legacy":
				typeCases = append(typeCases, "\tcase *IntersectionType:\n\n\t\tintersectionSuperType := typedSuperType.LegacyType //nolint:staticcheck\n\n\t\tswitch intersectionSuperType {\n\t\tcase nil, AnyResourceType, AnyStructType, AnyType:\n\n\t\t\tswitch subType {\n\t\t\tcase AnyResourceType:\n\t\t\t\treturn false\n\n\t\t\tcase AnyStructType:\n\t\t\t\treturn false\n\n\t\t\tcase AnyType:\n\t\t\t\treturn false\n\t\t\t}\n\n\t\t\tswitch typedSubType := subType.(type) {\n\t\t\tcase *IntersectionType:\n\n\t\t\t\tintersectionSubtype := typedSubType.LegacyType //nolint:staticcheck\n\t\t\t\tswitch intersectionSubtype {\n\t\t\t\tcase nil:\n\t\t\t\t\treturn typedSuperType.EffectiveIntersectionSet().\n\t\t\t\t\t\tIsSubsetOf(typedSubType.EffectiveIntersectionSet())\n\n\t\t\t\tcase AnyResourceType, AnyStructType, AnyType:\n\t\t\t\t\tif intersectionSuperType != nil &&\n\t\t\t\t\t\t!IsSubType(intersectionSubtype, intersectionSuperType) {\n\n\t\t\t\t\t\treturn false\n\t\t\t\t\t}\n\n\t\t\t\t\treturn typedSuperType.EffectiveIntersectionSet().\n\t\t\t\t\t\tIsSubsetOf(typedSubType.EffectiveIntersectionSet())\n\t\t\t\t}\n\n\t\t\t\tif intersectionSubtype, ok := intersectionSubtype.(*CompositeType); ok {\n\t\t\t\t\tif intersectionSuperType != nil &&\n\t\t\t\t\t\t!IsSubType(intersectionSubtype, intersectionSuperType) {\n\n\t\t\t\t\t\treturn false\n\t\t\t\t\t}\n\n\t\t\t\t\treturn typedSuperType.EffectiveIntersectionSet().\n\t\t\t\t\t\tIsSubsetOf(intersectionSubtype.EffectiveInterfaceConformanceSet())\n\t\t\t\t}\n\n\t\t\tcase ConformingType:\n\t\t\t\tif intersectionSuperType != nil &&\n\t\t\t\t\t!IsSubType(typedSubType, intersectionSuperType) {\n\n\t\t\t\t\treturn false\n\t\t\t\t}\n\n\t\t\t\treturn typedSuperType.EffectiveIntersectionSet().\n\t\t\t\t\tIsSubsetOf(typedSubType.EffectiveInterfaceConformanceSet())\n\t\t\t}\n\n\t\tdefault:\n\t\t\t// Supertype (intersection) has a non-Any* legacy type\n\n\t\t\tswitch typedSubType := subType.(type) {\n\t\t\tcase *IntersectionType:\n\n\t\t\t\tintersectionSubType := typedSubType.LegacyType //nolint:staticcheck\n\t\t\t\tswitch intersectionSubType {\n\t\t\t\tcase nil, AnyResourceType, AnyStructType, AnyType:\n\t\t\t\t\treturn false\n\t\t\t\t}\n\n\t\t\t\tif intersectionSubType, ok := intersectionSubType.(*CompositeType); ok {\n\t\t\t\t\treturn intersectionSubType == intersectionSuperType\n\t\t\t\t}\n\n\t\t\tcase *CompositeType:\n\t\t\t\treturn IsSubType(typedSubType, intersectionSuperType)\n\t\t\t}\n\n\t\t\tswitch subType {\n\t\t\tcase AnyResourceType, AnyStructType, AnyType:\n\t\t\t\treturn false\n\t\t\t}\n\t\t}\n\n")
			case "composite-super":
				typeCases = append(typeCases, "\tcase *CompositeType:\n\n\t\tswitch typedSubType := subType.(type) {\n\t\tcase *IntersectionType:\n\n\t\t\tlegacyType := typedSubType.LegacyType\n\t\t\tswitch legacyType {\n\t\t\tcase nil, AnyResourceType, AnyStructType, AnyType:\n\t\t\t\treturn false\n\t\t\t}\n\n\t\t\tif intersectionSubType, ok := legacyType.(*CompositeType); ok {\n\t\t\t\treturn intersectionSubType == typedSuperType\n\t\t\t}\n\n\t\tcase *CompositeType:\n\t\t\treturn false\n\t\t}\n\n")
			case "interface-super":
				typeCases = append(typeCases, "\tcase *InterfaceType:\n\n\t\tswitch typedSubType := subType.(type) {\n\t\tcase *CompositeType:\n\n\t\t\tif typedSubType.Kind != typedSuperType.CompositeKind {\n\t\t\t\treturn false\n\t\t\t}\n\n\t\t\treturn typedSubType.EffectiveInterfaceConformanceSet().\n\t\t\t\tContains(typedSuperType)\n\n\t\tcase *IntersectionType:\n\t\t\treturn typedSubType.EffectiveIntersectionSet().Contains(typedSuperType)\n\n\t\tcase *InterfaceType:\n\t\t\treturn typedSubType.EffectiveInterfaceConformanceSet().\n\t\t\t\tContains(typedSuperType)\n\t\t}\n\n")
			case "parameterized":
				typeCases = append(typeCases, "\tcase ParameterizedType:\n\t\tif superTypeBaseType := typedSuperType.BaseType(); superTypeBaseType != nil {\n\n\t\t\tif typedSubType, ok := subType.(ParameterizedType); ok {\n\t\t\t\tif subTypeBaseType := typedSubType.BaseType(); subTypeBaseType != nil {\n\n\t\t\t\t\tif !IsSubType(subTypeBaseType, superTypeBaseType) {\n\t\t\t\t\t\treturn false\n\t\t\t\t\t}\n\n\t\t\t\t\tsubTypeTypeArguments := typedSubType.TypeArguments()\n\t\t\t\t\tsuperTypeTypeArguments := typedSuperType.TypeArguments()\n\n\t\t\t\t\tif len(subTypeTypeArguments) != len(superTypeTypeArguments) {\n\t\t\t\t\t\treturn false\n\t\t\t\t\t}\n\n\t\t\t\t\tfor i, superTypeTypeArgument := range superTypeTypeArguments {\n\t\t\t\t\t\tsubTypeTypeArgument := subTypeTypeArguments[i]\n\t\t\t\t\t\tif !IsSubType(subTypeTypeArgument, superTypeTypeArgument) {\n\t\t\t\t\t\t\treturn false\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\n\t\t\t\t\treturn true\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t}\n\n")
		}
	}

	for _, r := range spec.Rules {
		switch v := r.(type) {
		case map[string]any:
			if n, ok := v["name"].(string); ok {
				emit(n)
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString("package ")
	buf.WriteString(pkg)
	buf.WriteString("\n\n")
	buf.WriteString("// Code generated by tools/subtype-gen; DO NOT EDIT.\n")
	buf.WriteString("// Source: declarative rules (" + spec.Name + ")\n\n")
	buf.WriteString("func checkSubTypeWithoutEquality(subType Type, superType Type) bool {\n\n")
	for _, b := range pre {
		buf.WriteString(b)
	}
	if openTypeSwitch {
		for _, c := range typeCases {
			buf.WriteString(c)
		}
		buf.WriteString("\t}\n\n")
	}
	for _, p := range post {
		buf.WriteString(p)
	}
	buf.WriteString("\treturn false\n}\n")
	return buf.Bytes(), nil
}

func writeOutput(dst string, content []byte, stdout bool) error {
	if stdout || dst == "-" {
		_, err := io.Copy(os.Stdout, bytes.NewReader(content))
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0o644)
}

func main() {
	var (
		yamlPath string
		outPath  string
		pkgName  string
		toStdout bool
	)
	flag.StringVar(&yamlPath, "rules", "rules.yaml", "path to YAML rules")
	flag.StringVar(&outPath, "out", "-", "output file path or '-' for stdout")
	flag.StringVar(&pkgName, "pkg", "sema", "target Go package name")
	flag.BoolVar(&toStdout, "stdout", false, "write to stdout")
	flag.Parse()

	spec, err := readYAML(yamlPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	code, err := generateFunction(spec, pkgName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := writeOutput(outPath, code, toStdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
