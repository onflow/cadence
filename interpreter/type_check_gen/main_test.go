package main

import (
	"fmt"
	"os"
	"testing"

	subtypegen "github.com/onflow/cadence/tools/subtype-gen"
)

func TestCustomization(t *testing.T) {
	// Read and parse YAML rules
	rules, err := subtypegen.ParseRules()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error reading YAML rules: %v\n", err)
		os.Exit(1)
	}

	config := subtypegen.Config{
		SimpleTypePrefix:  "PrimitiveStaticType",
		ComplexTypeSuffix: "StaticType",
		ExtraParams: []subtypegen.ExtraParam{
			{
				Name:    typeConverterParamName,
				Type:    typeConverterTypeName,
				PkgPath: interpreterPkgPath,
			},
		},
		SkipTypes: map[string]struct{}{
			subtypegen.TypePlaceholderStorable:      {},
			subtypegen.TypePlaceholderParameterized: {},
		},
		NonPointerTypes: map[string]struct{}{
			subtypegen.TypePlaceholderFunction:   {},
			subtypegen.TypePlaceholderConforming: {},
		},
	}

	// Generate code using the comprehensive generator
	gen := subtypegen.NewSubTypeCheckGenerator(config)
	decls := gen.GenerateCheckSubTypeWithoutEqualityFunction(rules)

	decls = Update(decls)
}
