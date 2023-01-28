/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"fmt"
	"go/token"
	"os"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/dave/dst/decorator"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"

	"github.com/dave/dst"
)

const headerTemplate = `// Code generated from {{ . }}. DO NOT EDIT.
/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

`

var parsedHeaderTemplate = template.Must(template.New("header").Parse(headerTemplate))

var parserConfig = parser.Config{
	StaticModifierEnabled: true,
	NativeModifierEnabled: true,
}

func initialUpper(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}

func trimLineSpaces(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		b.WriteString(strings.TrimSpace(line))
		b.WriteByte('\n')
	}
	return b.String()
}

type generator struct {
	containerTypeNames []string
	decls              []dst.Decl
}

var _ ast.DeclarationVisitor[struct{}] = &generator{}

func (g *generator) addDecls(decls ...*dst.GenDecl) {
	for _, decl := range decls {
		g.decls = append(g.decls, decl)
	}
}

func (*generator) VisitVariableDeclaration(_ *ast.VariableDeclaration) struct{} {
	panic("variable declarations are not supported")
}

func (g *generator) VisitFunctionDeclaration(decl *ast.FunctionDeclaration) (_ struct{}) {
	if len(g.containerTypeNames) == 0 {
		panic("global function declarations are not supported")
	}

	if !decl.IsNative() {
		panic("non-native function declarations are not supported")
	}

	if decl.IsStatic() {
		panic("static function declarations are not supported")
	}

	functionName := decl.Identifier.Identifier
	fullTypeName := g.fullTypeName()
	docString := g.declarationDocString(decl)

	parameters := decl.ParameterList.Parameters

	parameterTypeAnnotations := make([]*ast.TypeAnnotation, 0, len(parameters))
	for _, parameter := range parameters {
		parameterTypeAnnotations = append(
			parameterTypeAnnotations,
			parameter.TypeAnnotation,
		)
	}

	g.addDecls(
		goConstDecl(
			functionNameVarName(fullTypeName, functionName),
			goStringLit(functionName),
		),
		goVarDecl(
			functionTypeVarName(fullTypeName, functionName),
			typeExpr(&ast.FunctionType{
				ReturnTypeAnnotation:     decl.ReturnTypeAnnotation,
				ParameterTypeAnnotations: parameterTypeAnnotations,
			}),
		),
		goConstDecl(
			functionDocStringVarName(fullTypeName, functionName),
			goRawLit(docString),
		),
	)

	return
}

func (g *generator) declarationDocString(decl ast.Declaration) string {
	identifier := decl.DeclarationIdentifier().Identifier
	docString := strings.TrimSpace(decl.DeclarationDocString())

	if len(docString) == 0 {
		panic(fmt.Errorf(
			"missing doc string for %s",
			g.memberID(identifier),
		))
	}

	// TODO: allow by splitting and wrapping in double quotes
	if strings.ContainsRune(docString, '`') {
		panic(fmt.Errorf("invalid ` in doc string for field %s", g.memberID(identifier)))
	}

	return trimLineSpaces(docString)
}

func (*generator) VisitSpecialFunctionDeclaration(_ *ast.SpecialFunctionDeclaration) struct{} {
	panic("special function declarations are not supported")
}

func (g *generator) VisitCompositeDeclaration(decl *ast.CompositeDeclaration) (_ struct{}) {
	var isResource bool

	compositeKind := decl.CompositeKind
	switch compositeKind {
	case common.CompositeKindStructure:
		break
	case common.CompositeKindResource:
		isResource = true
	default:
		panic(fmt.Sprintf("%s declarations are not supported", compositeKind.Name()))
	}

	typeName := decl.Identifier.Identifier

	g.containerTypeNames = append(g.containerTypeNames, typeName)
	defer func() {
		g.containerTypeNames = g.containerTypeNames[:len(g.containerTypeNames)-1]
	}()

	var memberDeclarations []ast.Declaration

	for _, memberDeclaration := range decl.Members.Declarations() {
		ast.AcceptDeclaration[struct{}](memberDeclaration, g)

		memberDeclarationKind := memberDeclaration.DeclarationKind()
		switch memberDeclarationKind {
		case common.DeclarationKindField,
			common.DeclarationKindFunction:

			memberDeclarations = append(memberDeclarations, memberDeclaration)

		default:
			panic(fmt.Errorf(
				"%s members are not supported",
				memberDeclarationKind.Name(),
			))
		}
	}

	var (
		equatable,
		storable,
		exportable,
		importable bool
	)

	for _, conformance := range decl.Conformances {
		switch conformance.Identifier.Identifier {
		case "Storable":
			storable = true
		case "Equatable":
			equatable = true
		case "Exportable":
			exportable = true
		case "Importable":
			importable = true
		}
	}

	g.addDecls(
		goConstDecl(
			typeNameVarName(typeName),
			goStringLit(typeName),
		),
		goVarDecl(
			fmt.Sprintf("%sType", typeName),
			simpleTypeLiteral(simpleType{
				typeName:           typeName,
				fullTypeName:       g.fullTypeName(),
				isResource:         isResource,
				Storable:           storable,
				Equatable:          equatable,
				Exportable:         exportable,
				Importable:         importable,
				memberDeclarations: memberDeclarations,
			}),
		),
	)

	return
}

func (*generator) VisitInterfaceDeclaration(_ *ast.InterfaceDeclaration) struct{} {
	panic("interface declarations are not supported")
}

func (*generator) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) struct{} {
	panic("transaction declarations are not supported")
}

func (g *generator) VisitFieldDeclaration(decl *ast.FieldDeclaration) (_ struct{}) {
	fieldName := decl.Identifier.Identifier
	fullTypeName := g.fullTypeName()
	docString := g.declarationDocString(decl)

	g.addDecls(
		goConstDecl(
			fieldNameVarName(fullTypeName, fieldName),
			goStringLit(fieldName),
		),
		goVarDecl(
			fieldTypeVarName(fullTypeName, fieldName),
			typeExpr(decl.TypeAnnotation.Type),
		),
		goConstDecl(
			fieldDocStringVarName(fullTypeName, fieldName),
			goRawLit(docString),
		),
	)

	return
}

func typeExpr(t ast.Type) dst.Expr {
	switch t := t.(type) {
	case *ast.NominalType:
		return typeVarIdent(t.Identifier.Identifier)

	case *ast.OptionalType:
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("OptionalType"),
				Elts: []dst.Expr{
					goKeyValue("Type", typeExpr(t.Type)),
				},
			},
		}

	case *ast.ReferenceType:
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("ReferenceType"),
				Elts: []dst.Expr{
					goKeyValue("Type", typeExpr(t.Type)),
				},
			},
		}

	case *ast.VariableSizedType:
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("VariableSizedType"),
				Elts: []dst.Expr{
					goKeyValue("Type", typeExpr(t.Type)),
				},
			},
		}

	case *ast.ConstantSizedType:
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("ConstantSizedType"),
				Elts: []dst.Expr{
					goKeyValue("Type", typeExpr(t.Type)),
					goKeyValue(
						"Size",
						&dst.BasicLit{
							Kind:  token.INT,
							Value: t.Size.String(),
						},
					),
				},
			},
		}
	case *ast.FunctionType:
		// TODO: type parameters

		parameterExprs := make([]dst.Expr, 0, len(t.ParameterTypeAnnotations))

		for _, parameterTypeAnnotation := range t.ParameterTypeAnnotations {

			parameterExpr := &dst.CompositeLit{
				Elts: []dst.Expr{
					goKeyValue(
						"TypeAnnotation",
						typeAnnotationCallExpr(typeExpr(parameterTypeAnnotation.Type)),
					),
				},
			}
			parameterExpr.Decorations().Before = dst.NewLine
			parameterExpr.Decorations().After = dst.NewLine

			parameterExprs = append(
				parameterExprs,
				parameterExpr,
			)
		}

		parametersExpr := &dst.CompositeLit{
			Type: &dst.ArrayType{
				Elt: dst.NewIdent("Parameter"),
			},
			Elts: parameterExprs,
		}

		returnTypeExpr := typeExpr(t.ReturnTypeAnnotation.Type)
		returnTypeExpr.Decorations().Before = dst.NewLine
		returnTypeExpr.Decorations().After = dst.NewLine

		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("FunctionType"),
				Elts: []dst.Expr{
					goKeyValue(
						"Parameters",
						parametersExpr,
					),
					goKeyValue(
						"ReturnTypeAnnotation",
						typeAnnotationCallExpr(returnTypeExpr),
					),
				},
			},
		}

	default:
		panic(fmt.Errorf("%T types are not supported", t))
	}
}

func (*generator) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) struct{} {
	panic("enum case declarations are not supported")
}

func (*generator) VisitPragmaDeclaration(_ *ast.PragmaDeclaration) struct{} {
	panic("pragma declarations are not supported")
}

func (*generator) VisitImportDeclaration(_ *ast.ImportDeclaration) struct{} {
	panic("import declarations are not supported")
}

func (g *generator) fullTypeName() string {
	return strings.Join(g.containerTypeNames, "")
}

func (g *generator) memberID(fieldName string) string {
	return fmt.Sprintf("%s.%s",
		strings.Join(g.containerTypeNames, "."),
		fieldName,
	)
}

func goField(name string, ty dst.Expr) *dst.Field {
	return &dst.Field{
		Names: []*dst.Ident{
			dst.NewIdent(name),
		},
		Type: ty,
	}
}

func goVarConstDecl(isConst bool, name string, value dst.Expr) *dst.GenDecl {
	tok := token.VAR
	if isConst {
		tok = token.CONST
	}
	decl := &dst.GenDecl{
		Tok: tok,
		Specs: []dst.Spec{
			&dst.ValueSpec{
				Names: []*dst.Ident{
					dst.NewIdent(name),
				},
				Values: []dst.Expr{
					value,
				},
			},
		},
	}
	decl.Decorations().After = dst.EmptyLine
	return decl
}

func goConstDecl(name string, value dst.Expr) *dst.GenDecl {
	return goVarConstDecl(true, name, value)
}

func goVarDecl(name string, value dst.Expr) *dst.GenDecl {
	return goVarConstDecl(false, name, value)
}

func goKeyValue(name string, value dst.Expr) *dst.KeyValueExpr {
	expr := &dst.KeyValueExpr{
		Key:   dst.NewIdent(name),
		Value: value,
	}
	expr.Decorations().Before = dst.NewLine
	expr.Decorations().After = dst.NewLine
	return expr
}

func goStringLit(s string) dst.Expr {
	return &dst.BasicLit{
		Kind:  token.STRING,
		Value: strconv.Quote(s),
	}
}

func goRawLit(s string) dst.Expr {
	return &dst.BasicLit{
		Kind:  token.STRING,
		Value: fmt.Sprintf("`%s`", s),
	}
}

func goBoolLit(b bool) dst.Expr {
	if b {
		return dst.NewIdent(strconv.FormatBool(true))
	}
	return dst.NewIdent(strconv.FormatBool(false))
}

func declarationKindExpr(kind string) *dst.SelectorExpr {
	return &dst.SelectorExpr{
		X:   dst.NewIdent("common"),
		Sel: dst.NewIdent(fmt.Sprintf("DeclarationKind%s", kind)),
	}
}

func typeVarName(typeName string) string {
	return fmt.Sprintf("%sType", typeName)
}

func typeVarIdent(typeName string) *dst.Ident {
	return dst.NewIdent(typeVarName(typeName))
}

func typeNameVarName(typeName string) string {
	return fmt.Sprintf("%sTypeName", typeName)
}

func typeNameVarIdent(typeName string) *dst.Ident {
	return dst.NewIdent(typeNameVarName(typeName))
}

func typeTagVarIdent(typeName string) *dst.Ident {
	return dst.NewIdent(fmt.Sprintf("%sTypeTag", typeName))
}

func memberVarName(fullTypeName, fieldName, kind, part string) string {
	return fmt.Sprintf(
		"%sType%s%s%s",
		fullTypeName,
		initialUpper(fieldName),
		kind,
		part,
	)
}

func fieldNameVarName(fullTypeName, fieldName string) string {
	return memberVarName(fullTypeName, fieldName, "Field", "Name")
}

func functionNameVarName(fullTypeName, functionName string) string {
	return memberVarName(fullTypeName, functionName, "Function", "Name")
}

func fieldTypeVarName(fullTypeName, fieldName string) string {
	return memberVarName(fullTypeName, fieldName, "Field", "Type")
}

func functionTypeVarName(fullTypeName, functionName string) string {
	return memberVarName(fullTypeName, functionName, "Function", "Type")
}

func fieldDocStringVarName(fullTypeName, fieldName string) string {
	return memberVarName(fullTypeName, fieldName, "Field", "DocString")
}

func functionDocStringVarName(fullTypeName, functionName string) string {
	return memberVarName(fullTypeName, functionName, "Function", "DocString")
}

type simpleType struct {
	typeName           string
	fullTypeName       string
	isResource         bool
	Storable           bool
	Equatable          bool
	Exportable         bool
	Importable         bool
	memberDeclarations []ast.Declaration
}

func simpleTypeLiteral(ty simpleType) dst.Expr {
	elements := []dst.Expr{
		goKeyValue("Name", typeNameVarIdent(ty.typeName)),
		goKeyValue("QualifiedName", typeNameVarIdent(ty.typeName)),
		goKeyValue("TypeID", typeNameVarIdent(ty.typeName)),
		goKeyValue("tag", typeTagVarIdent(ty.typeName)),
		goKeyValue("IsResource", goBoolLit(ty.isResource)),
		goKeyValue("Storable", goBoolLit(ty.Storable)),
		goKeyValue("Equatable", goBoolLit(ty.Equatable)),
		goKeyValue("Exportable", goBoolLit(ty.Exportable)),
		goKeyValue("Importable", goBoolLit(ty.Importable)),
	}

	if len(ty.memberDeclarations) > 0 {
		members := simpleTypeMembers(ty.fullTypeName, ty.memberDeclarations)

		elements = append(
			elements,
			goKeyValue("Members", members),
		)
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: dst.NewIdent("SimpleType"),
			Elts: elements,
		},
	}
}

func simpleTypeMembers(fullTypeName string, declarations []ast.Declaration) dst.Expr {

	elements := make([]dst.Expr, 0, len(declarations))

	for _, declaration := range declarations {
		resolve := simpleTypeMemberResolver(fullTypeName, declaration)

		var memberName string
		var kind dst.Expr

		declarationKind := declaration.DeclarationKind()

		memberName = declaration.DeclarationIdentifier().Identifier

		switch declarationKind {
		case common.DeclarationKindField:
			memberName = fieldNameVarName(
				fullTypeName,
				memberName,
			)
			kind = declarationKindExpr("Field")

		case common.DeclarationKindFunction:
			memberName = functionNameVarName(
				fullTypeName,
				memberName,
			)
			kind = declarationKindExpr("Function")

		default:
			panic(fmt.Errorf(
				"%s members are not supported",
				declarationKind.Name(),
			))
		}

		elements = append(
			elements,
			goKeyValue(
				memberName,
				&dst.CompositeLit{
					Elts: []dst.Expr{
						goKeyValue("Kind", kind),
						goKeyValue("Resolve", resolve),
					},
				},
			),
		)
	}

	// func(t *SimpleType) map[string]MemberResolver {
	//   return map[string]MemberResolver{
	//     ...
	//   }
	// }

	returnStatement := &dst.ReturnStmt{
		Results: []dst.Expr{
			&dst.CompositeLit{
				Type: stringMemberResolverMapType(),
				Elts: elements,
			},
		},
	}
	returnStatement.Decorations().Before = dst.NewLine
	returnStatement.Decorations().After = dst.NewLine

	return &dst.FuncLit{
		Type: &dst.FuncType{
			Func: true,
			Params: &dst.FieldList{
				List: []*dst.Field{
					goField("t", &dst.StarExpr{X: dst.NewIdent("SimpleType")}),
				},
			},
			Results: &dst.FieldList{
				List: []*dst.Field{
					{
						Type: stringMemberResolverMapType(),
					},
				},
			},
		},
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				returnStatement,
			},
		},
	}
}

func simpleTypeMemberResolver(fullTypeName string, declaration ast.Declaration) dst.Expr {

	// func(
	//     memoryGauge common.MemoryGauge,
	//     identifier string,
	//     targetRange ast.Range,
	//     report func(error),
	// ) *Member

	parameters := []*dst.Field{
		goField(
			"memoryGauge",
			&dst.SelectorExpr{
				X:   dst.NewIdent("common"),
				Sel: dst.NewIdent("MemoryGauge"),
			},
		),
		goField("identifier", dst.NewIdent("string")),
		goField(
			"targetRange",
			&dst.SelectorExpr{
				X:   dst.NewIdent("ast"),
				Sel: dst.NewIdent("Range"),
			},
		),
		goField(
			"report",
			&dst.FuncType{
				Params: &dst.FieldList{
					List: []*dst.Field{
						{Type: dst.NewIdent("error")},
					},
				},
			},
		),
	}

	// TODO: bug: does not add newline before first and after last.
	//   Neither does setting decorations on the parameter field list
	//   or the function type work. Likely a problem in dst.
	for _, parameter := range parameters {
		parameter.Decorations().Before = dst.NewLine
		parameter.Decorations().After = dst.NewLine
	}

	functionType := &dst.FuncType{
		Func: true,
		Params: &dst.FieldList{
			List: parameters,
		},
		Results: &dst.FieldList{
			List: []*dst.Field{
				{
					Type: &dst.StarExpr{
						X: dst.NewIdent("Member"),
					},
				},
			},
		},
	}

	declarationKind := declaration.DeclarationKind()
	declarationName := declaration.DeclarationIdentifier().Identifier

	var result dst.Expr

	switch declarationKind {
	case common.DeclarationKindField:
		args := []dst.Expr{
			dst.NewIdent("memoryGauge"),
			dst.NewIdent("t"),
			dst.NewIdent("identifier"),
			dst.NewIdent(fieldTypeVarName(fullTypeName, declarationName)),
			dst.NewIdent(fieldDocStringVarName(fullTypeName, declarationName)),
		}

		for _, arg := range args {
			arg.Decorations().Before = dst.NewLine
			arg.Decorations().After = dst.NewLine
		}

		// TODO: add support for var
		result = &dst.CallExpr{
			Fun:  dst.NewIdent("NewPublicConstantFieldMember"),
			Args: args,
		}

	case common.DeclarationKindFunction:
		args := []dst.Expr{
			dst.NewIdent("memoryGauge"),
			dst.NewIdent("t"),
			dst.NewIdent("identifier"),
			dst.NewIdent(functionTypeVarName(fullTypeName, declarationName)),
			dst.NewIdent(functionDocStringVarName(fullTypeName, declarationName)),
		}

		for _, arg := range args {
			arg.Decorations().Before = dst.NewLine
			arg.Decorations().After = dst.NewLine
		}

		result = &dst.CallExpr{
			Fun:  dst.NewIdent("NewPublicFunctionMember"),
			Args: args,
		}

	default:
		panic(fmt.Errorf(
			"%s members are not supported",
			declarationKind.Name(),
		))
	}

	returnStatement := &dst.ReturnStmt{
		Results: []dst.Expr{
			result,
		},
	}
	returnStatement.Decorations().Before = dst.EmptyLine

	return &dst.FuncLit{
		Type: functionType,
		Body: &dst.BlockStmt{
			List: []dst.Stmt{
				returnStatement,
			},
		},
	}
}

func stringMemberResolverMapType() *dst.MapType {
	return &dst.MapType{
		Key:   dst.NewIdent("string"),
		Value: dst.NewIdent("MemberResolver"),
	}
}

func typeAnnotationCallExpr(returnTypeExpr dst.Expr) *dst.CallExpr {
	return &dst.CallExpr{
		Fun: dst.NewIdent("NewTypeAnnotation"),
		Args: []dst.Expr{
			returnTypeExpr,
		},
	}
}

func parseCadenceFile(path string) *ast.Program {
	program, code, err := parser.ParseProgramFromFile(nil, path, parserConfig)
	if err != nil {
		printer := pretty.NewErrorPrettyPrinter(os.Stderr, true)
		location := common.StringLocation(path)
		_ = printer.PrettyPrintError(err, location, map[common.Location][]byte{
			location: code,
		})
		os.Exit(1)
		return nil
	}
	return program
}

func gen(inPath string, outFile *os.File) {
	program := parseCadenceFile(inPath)

	var gen generator
	gen.addDecls(
		goImportDeclaration(
			"github.com/onflow/cadence/runtime/ast",
			"github.com/onflow/cadence/runtime/common",
		),
	)

	for _, declaration := range program.Declarations() {
		_ = ast.AcceptDeclaration[struct{}](declaration, &gen)
	}

	writeGoFile(inPath, outFile, gen.decls)
}

func goImportDeclaration(paths ...string) *dst.GenDecl {

	specs := make([]dst.Spec, 0, len(paths))

	for _, path := range paths {
		specs = append(
			specs,
			&dst.ImportSpec{
				Path: &dst.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(path),
				},
			},
		)
	}

	return &dst.GenDecl{
		Tok:   token.IMPORT,
		Specs: specs,
	}
}

func writeGoFile(inPath string, outFile *os.File, decls []dst.Decl) {
	err := parsedHeaderTemplate.Execute(outFile, inPath)
	if err != nil {
		panic(err)
	}

	err = decorator.Fprint(
		outFile,
		&dst.File{
			Name:  dst.NewIdent("sema"),
			Decls: decls,
		},
	)
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		panic("Missing path to input Cadence file")
	}
	if len(os.Args) < 3 {
		panic("Missing path to output Go file")
	}
	inPath := os.Args[1]
	outPath := os.Args[2]

	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	gen(inPath, outFile)
}
