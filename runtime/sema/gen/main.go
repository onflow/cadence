/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"flag"
	"fmt"
	"go/token"
	"os"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/dave/dst/decorator"
	"github.com/dave/dst/decorator/resolver/guess"

	"github.com/dave/dst"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"
	"github.com/onflow/cadence/runtime/sema"
)

const semaPath = "github.com/onflow/cadence/runtime/sema"
const astPath = "github.com/onflow/cadence/runtime/ast"

var packagePathFlag = flag.String("p", semaPath, "package path")

const headerTemplate = `// Code generated from {{ . }}. DO NOT EDIT.
/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	TypeParametersEnabled: true,
}

func initialUpper(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(unicode.ToUpper(rune(s[0]))) + s[1:]
}

// turn a non-empty docstring into a formatted raw go string literal, with surrounding backticks.
// inline backticks for code literals are turned into separate strings that are
func renderDocString(s string) dst.Expr {

	var docstringChunks []dst.Expr

	lines := strings.Split(s, "\n")

	var b strings.Builder
	b.WriteByte('\n') // start the very first chunk with a newline

	writeAccumulated := func() {
		if b.Len() == 0 {
			return
		}

		docstringChunks = append(docstringChunks, goRawLit(b.String()))

		b.Reset()
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		chunks := strings.Split(line, "`")
		if len(chunks) == 1 {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}

		// handle inline backticked expressions by splitting them into regular string literals
		inChunk := false
		for _, chunk := range chunks {
			if inChunk {

				writeAccumulated()

				if len(chunk) > 0 {
					surrounded := fmt.Sprintf("`%s`", chunk)
					docstringChunks = append(docstringChunks, goStringLit(surrounded))
				}

			} else {
				b.WriteString(chunk)
			}

			inChunk = !inChunk // splitting by backticks means each chunk is an alternate state
		}
		b.WriteByte('\n')
	}

	writeAccumulated()

	result, rest := docstringChunks[0], docstringChunks[1:]

	// perform a left-associative fold over chunks, joining them as `x + y`
	// the `+` token is left-associative in go
	for _, chunk := range rest {
		result = &dst.BinaryExpr{
			X:  result,
			Op: token.ADD,
			Y:  chunk,
		}
	}

	return result
}

type typeDecl struct {
	typeName           string
	fullTypeName       string
	compositeKind      common.CompositeKind
	storable           bool
	primitive          bool
	equatable          bool
	exportable         bool
	comparable         bool
	importable         bool
	memberAccessible   bool
	memberDeclarations []ast.Declaration
	nestedTypes        []*typeDecl
	hasConstructor     bool

	// used in simpleType generation
	conformances []string
}

type generator struct {
	typeStack     []*typeDecl
	decls         []dst.Decl
	leadingPragma map[string]struct{}
}

var _ ast.DeclarationVisitor[struct{}] = &generator{}

func (g *generator) addDecls(decls ...dst.Decl) {
	g.decls = append(g.decls, decls...)
}

func (*generator) VisitVariableDeclaration(_ *ast.VariableDeclaration) struct{} {
	panic("variable declarations are not supported")
}

func (g *generator) VisitFunctionDeclaration(decl *ast.FunctionDeclaration) (_ struct{}) {
	if len(g.typeStack) == 0 {
		panic("global function declarations are not supported")
	}

	if decl.IsStatic() {
		panic("static function declarations are not supported")
	}

	functionName := decl.Identifier.Identifier
	fullTypeName := g.currentFullTypeName()

	g.addFunctionNameDeclaration(fullTypeName, functionName)

	var typeParams map[string]string

	if decl.TypeParameterList != nil {
		typeParams = g.addFunctionTypeParameterDeclarations(decl, fullTypeName, functionName)
	}

	g.addFunctionTypeDeclaration(decl, fullTypeName, functionName, typeParams)

	g.addFunctionDocStringDeclaration(decl, fullTypeName, functionName)

	return
}

func (g *generator) addFunctionNameDeclaration(
	fullTypeName string,
	functionName string,
) {
	g.addDecls(
		goConstDecl(
			functionNameVarName(fullTypeName, functionName),
			goStringLit(functionName),
		),
	)
}

func (g *generator) addFunctionTypeParameterDeclarations(
	decl *ast.FunctionDeclaration,
	fullTypeName string,
	functionName string,
) (typeParams map[string]string) {
	typeParameters := decl.TypeParameterList.TypeParameters
	typeParams = make(map[string]string, len(typeParameters))

	for _, typeParameter := range typeParameters {
		typeParameterName := typeParameter.Identifier.Identifier

		var typeBound dst.Expr
		if typeParameter.TypeBound != nil {
			typeBound = typeExpr(
				typeParameter.TypeBound.Type,
				typeParams,
			)
		}

		typeParams[typeParameterName] = functionTypeParameterVarName(
			fullTypeName,
			functionName,
			typeParameterName,
		)

		g.addDecls(
			goVarDecl(
				functionTypeParameterVarName(
					fullTypeName,
					functionName,
					typeParameterName,
				),
				typeParameterExpr(
					typeParameterName,
					typeBound,
				),
			),
		)
	}

	return
}

func (g *generator) addFunctionTypeDeclaration(
	decl *ast.FunctionDeclaration,
	fullTypeName string,
	functionName string,
	typeParams map[string]string,
) {
	parameters := decl.ParameterList.Parameters

	parameterTypeAnnotations := make([]*ast.TypeAnnotation, 0, len(parameters))
	for _, parameter := range parameters {
		parameterTypeAnnotations = append(
			parameterTypeAnnotations,
			parameter.TypeAnnotation,
		)
	}

	g.addDecls(
		goVarDecl(
			functionTypeVarName(fullTypeName, functionName),
			functionTypeExpr(
				&ast.FunctionType{
					PurityAnnotation:         decl.Purity,
					ReturnTypeAnnotation:     decl.ReturnTypeAnnotation,
					ParameterTypeAnnotations: parameterTypeAnnotations,
				},
				decl.ParameterList,
				decl.TypeParameterList,
				typeParams,
				false,
			),
		),
	)
}

func (g *generator) addFunctionDocStringDeclaration(
	decl *ast.FunctionDeclaration,
	fullTypeName string,
	functionName string,
) {
	docString := g.declarationDocString(decl)

	g.addDecls(
		goConstDecl(
			functionDocStringVarName(fullTypeName, functionName),
			docString,
		),
	)
}

func (g *generator) declarationDocString(decl ast.Declaration) dst.Expr {
	identifier := decl.DeclarationIdentifier().Identifier
	docString := strings.TrimSpace(decl.DeclarationDocString())

	if len(docString) == 0 {
		panic(fmt.Errorf(
			"missing doc string for %s",
			g.currentMemberID(identifier),
		))
	}

	return renderDocString(docString)
}

func (g *generator) VisitSpecialFunctionDeclaration(decl *ast.SpecialFunctionDeclaration) (_ struct{}) {
	if decl.Kind != common.DeclarationKindInitializer {
		panic(fmt.Errorf(
			"%s special function declarations are not supported",
			decl.Kind.Name(),
		))
	}

	typeDecl := g.currentTypeDecl()

	fullTypeName := typeDecl.fullTypeName

	if typeDecl.hasConstructor {
		panic(fmt.Errorf("invalid second initializer for type %s", fullTypeName))
	}
	typeDecl.hasConstructor = true

	isResource := typeDecl.compositeKind == common.CompositeKindResource

	typeNames := make([]string, 0, len(g.typeStack))
	for i := 0; i < len(g.typeStack); i++ {
		typeNames = append(typeNames, g.typeStack[i].typeName)
	}

	g.addConstructorTypeDeclaration(decl, fullTypeName, typeNames, isResource)

	g.addConstructorDocStringDeclaration(decl, fullTypeName)

	return
}

func (g *generator) addConstructorTypeDeclaration(
	initDecl *ast.SpecialFunctionDeclaration,
	fullTypeName string,
	typeNames []string,
	isResource bool,
) {
	decl := initDecl.FunctionDeclaration

	parameters := decl.ParameterList.Parameters

	parameterTypeAnnotations := make([]*ast.TypeAnnotation, 0, len(parameters))
	for _, parameter := range parameters {
		parameterTypeAnnotations = append(
			parameterTypeAnnotations,
			parameter.TypeAnnotation,
		)
	}

	nestedIdentifiers := make([]ast.Identifier, 0, len(typeNames)-1)
	for i := 1; i < len(typeNames); i++ {
		typeName := typeNames[i]
		nestedIdentifiers = append(
			nestedIdentifiers,
			ast.Identifier{
				Identifier: typeName,
			},
		)
	}

	returnType := &ast.NominalType{
		NestedIdentifiers: nestedIdentifiers,
		Identifier: ast.Identifier{
			Identifier: typeNames[0],
		},
	}

	g.addDecls(
		goVarDecl(
			constructorTypeVarName(fullTypeName),
			functionTypeExpr(
				&ast.FunctionType{
					PurityAnnotation: decl.Purity,
					ReturnTypeAnnotation: &ast.TypeAnnotation{
						Type:       returnType,
						IsResource: isResource,
					},
					ParameterTypeAnnotations: parameterTypeAnnotations,
				},
				decl.ParameterList,
				nil,
				nil,
				true,
			),
		),
	)
}

func (g *generator) addConstructorDocStringDeclaration(
	decl *ast.SpecialFunctionDeclaration,
	fullTypeName string,
) {
	docString := g.declarationDocString(decl)

	g.addDecls(
		goConstDecl(
			constructorDocStringVarName(fullTypeName),
			docString,
		),
	)
}

func (g *generator) VisitCompositeDeclaration(decl *ast.CompositeDeclaration) (_ struct{}) {

	compositeKind := decl.CompositeKind
	switch compositeKind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindContract:
		break
	default:
		panic(fmt.Sprintf("%s declarations are not supported", compositeKind.Name()))
	}

	typeName := decl.Identifier.Identifier

	typeDecl := &typeDecl{
		typeName:      typeName,
		fullTypeName:  g.newFullTypeName(typeName),
		compositeKind: compositeKind,
	}

	if len(g.typeStack) > 0 {
		parentType := g.typeStack[len(g.typeStack)-1]
		parentType.nestedTypes = append(
			parentType.nestedTypes,
			typeDecl,
		)
	}

	g.typeStack = append(
		g.typeStack,
		typeDecl,
	)
	defer func() {
		// Pop
		lastIndex := len(g.typeStack) - 1
		g.typeStack[lastIndex] = nil
		g.typeStack = g.typeStack[:lastIndex]
	}()

	var generateSimpleType bool

	// Check if the declaration is explicitly marked to be generated as a composite type.
	if _, ok := g.leadingPragma["compositeType"]; ok {
		generateSimpleType = false
	} else {
		// If not, decide what to generate depending on the type.

		// We can generate a SimpleType declaration,
		// if this is a top-level type,
		// and this declaration has no nested type declarations.
		// Otherwise, we have to generate a CompositeType
		generateSimpleType = len(g.typeStack) == 1
		if generateSimpleType {
			switch compositeKind {
			case common.CompositeKindStructure,
				common.CompositeKindResource:
				break
			default:
				generateSimpleType = false
			}
		}
	}

	for _, memberDeclaration := range decl.Members.Declarations() {
		generateDeclaration(g, memberDeclaration)

		// Visiting unsupported declarations panics,
		// so only supported member declarations are added
		typeDecl.memberDeclarations = append(
			typeDecl.memberDeclarations,
			memberDeclaration,
		)

		if generateSimpleType {
			switch memberDeclaration.(type) {
			case *ast.FieldDeclaration,
				*ast.FunctionDeclaration:
				break

			default:
				generateSimpleType = false
			}
		}
	}

	for _, conformance := range decl.Conformances {
		switch conformance.Identifier.Identifier {
		case "Storable":
			if !generateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as storable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.storable = true

		case "Primitive":
			if !generateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as primitive: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.primitive = true

		case "Equatable":
			if !generateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as equatable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.equatable = true

		case "Comparable":
			if !generateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as comparable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.comparable = true

		case "Exportable":
			if !generateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as exportable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.exportable = true

		case "Importable":
			typeDecl.importable = true

		case "ContainFields":
			if !generateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as having fields: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.memberAccessible = true
		case "StructStringer":
			typeDecl.conformances = append(typeDecl.conformances, "StructStringerType")
		}
	}

	var typeVarDecl dst.Expr
	if generateSimpleType {
		typeVarDecl = simpleTypeLiteral(typeDecl)
	} else {
		typeVarDecl = compositeTypeExpr(typeDecl)
	}

	fullTypeName := typeDecl.fullTypeName

	tyVarName := typeVarName(fullTypeName)

	g.addDecls(
		goConstDecl(
			typeNameVarName(fullTypeName),
			goStringLit(typeName),
		),
		goVarDecl(
			tyVarName,
			typeVarDecl,
		),
	)

	memberDeclarations := typeDecl.memberDeclarations

	if len(memberDeclarations) > 0 {

		if generateSimpleType {

			// func init() {
			//   t.Members = func(t *SimpleType) map[string]MemberResolver {
			//     return MembersAsResolvers(...)
			//   }
			// }

			memberResolversFunc := simpleTypeMemberResolversFunc(fullTypeName, memberDeclarations)

			g.addDecls(
				&dst.FuncDecl{
					Name: dst.NewIdent("init"),
					Type: &dst.FuncType{},
					Body: &dst.BlockStmt{
						List: []dst.Stmt{
							&dst.AssignStmt{
								Lhs: []dst.Expr{
									&dst.SelectorExpr{
										X:   dst.NewIdent(tyVarName),
										Sel: dst.NewIdent("Members"),
									},
								},
								Tok: token.ASSIGN,
								Rhs: []dst.Expr{
									memberResolversFunc,
								},
							},
						},
					},
				},
			)

		} else {

			// func init() {
			//   members := []*Member{...}
			//   t.Members = MembersAsMap(members)
			//   t.Fields = MembersFieldNames(members)
			//   t.ConstructorParameters = ...
			// }

			members := membersExpr(
				fullTypeName,
				tyVarName,
				memberDeclarations,
			)

			const membersVariableIdentifier = "members"

			stmts := []dst.Stmt{
				&dst.DeclStmt{
					Decl: goVarDecl(
						membersVariableIdentifier,
						members,
					),
				},
				&dst.AssignStmt{
					Lhs: []dst.Expr{
						&dst.SelectorExpr{
							X:   dst.NewIdent(tyVarName),
							Sel: dst.NewIdent("Members"),
						},
					},
					Tok: token.ASSIGN,
					Rhs: []dst.Expr{
						&dst.CallExpr{
							Fun: &dst.Ident{
								Name: "MembersAsMap",
								Path: semaPath,
							},
							Args: []dst.Expr{
								dst.NewIdent(membersVariableIdentifier),
							},
						},
					},
				},
				&dst.AssignStmt{
					Lhs: []dst.Expr{
						&dst.SelectorExpr{
							X:   dst.NewIdent(tyVarName),
							Sel: dst.NewIdent("Fields"),
						},
					},
					Tok: token.ASSIGN,
					Rhs: []dst.Expr{
						&dst.CallExpr{
							Fun: &dst.Ident{
								Name: "MembersFieldNames",
								Path: semaPath,
							},
							Args: []dst.Expr{
								dst.NewIdent(membersVariableIdentifier),
							},
						},
					},
				},
			}

			if typeDecl.hasConstructor {
				stmts = append(
					stmts,
					&dst.AssignStmt{
						Lhs: []dst.Expr{
							&dst.SelectorExpr{
								X:   dst.NewIdent(tyVarName),
								Sel: dst.NewIdent("ConstructorParameters"),
							},
						},
						Tok: token.ASSIGN,
						Rhs: []dst.Expr{
							&dst.SelectorExpr{
								X:   dst.NewIdent(constructorTypeVarName(fullTypeName)),
								Sel: dst.NewIdent("Parameters"),
							},
						},
					},
				)
			}

			g.addDecls(
				&dst.FuncDecl{
					Name: dst.NewIdent("init"),
					Type: &dst.FuncType{},
					Body: &dst.BlockStmt{
						List: stmts,
					},
				},
			)
		}
	}

	return
}

func (g *generator) VisitInterfaceDeclaration(decl *ast.InterfaceDeclaration) (_ struct{}) {
	compositeKind := decl.CompositeKind
	switch compositeKind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindContract:
		break
	default:
		panic(fmt.Sprintf("%s declarations are not supported", compositeKind.Name()))
	}

	typeName := decl.Identifier.Identifier

	typeDecl := &typeDecl{
		typeName:      typeName,
		fullTypeName:  g.newFullTypeName(typeName),
		compositeKind: compositeKind,
	}

	if len(g.typeStack) > 0 {
		parentType := g.typeStack[len(g.typeStack)-1]
		parentType.nestedTypes = append(
			parentType.nestedTypes,
			typeDecl,
		)
	}

	g.typeStack = append(
		g.typeStack,
		typeDecl,
	)
	defer func() {
		// Pop
		lastIndex := len(g.typeStack) - 1
		g.typeStack[lastIndex] = nil
		g.typeStack = g.typeStack[:lastIndex]
	}()

	for _, memberDeclaration := range decl.Members.Declarations() {
		generateDeclaration(g, memberDeclaration)

		// Visiting unsupported declarations panics,
		// so only supported member declarations are added
		typeDecl.memberDeclarations = append(
			typeDecl.memberDeclarations,
			memberDeclaration,
		)
	}

	var typeVarDecl = interfaceTypeExpr(typeDecl)

	fullTypeName := typeDecl.fullTypeName

	tyVarName := typeVarName(fullTypeName)

	g.addDecls(
		goConstDecl(
			typeNameVarName(fullTypeName),
			goStringLit(typeName),
		),
		goVarDecl(
			tyVarName,
			typeVarDecl,
		),
	)

	memberDeclarations := typeDecl.memberDeclarations

	if len(memberDeclarations) > 0 {

		// func init() {
		//   members := []*Member{...}
		//   t.Members = MembersAsMap(members)
		//   t.Fields = MembersFieldNames(members)
		//   t.ConstructorParameters = ...
		// }

		members := membersExpr(
			fullTypeName,
			tyVarName,
			memberDeclarations,
		)

		const membersVariableIdentifier = "members"

		stmts := []dst.Stmt{
			&dst.DeclStmt{
				Decl: goVarDecl(
					membersVariableIdentifier,
					members,
				),
			},
			&dst.AssignStmt{
				Lhs: []dst.Expr{
					&dst.SelectorExpr{
						X:   dst.NewIdent(tyVarName),
						Sel: dst.NewIdent("Members"),
					},
				},
				Tok: token.ASSIGN,
				Rhs: []dst.Expr{
					&dst.CallExpr{
						Fun: &dst.Ident{
							Name: "MembersAsMap",
							Path: semaPath,
						},
						Args: []dst.Expr{
							dst.NewIdent(membersVariableIdentifier),
						},
					},
				},
			},
			&dst.AssignStmt{
				Lhs: []dst.Expr{
					&dst.SelectorExpr{
						X:   dst.NewIdent(tyVarName),
						Sel: dst.NewIdent("Fields"),
					},
				},
				Tok: token.ASSIGN,
				Rhs: []dst.Expr{
					&dst.CallExpr{
						Fun: &dst.Ident{
							Name: "MembersFieldNames",
							Path: semaPath,
						},
						Args: []dst.Expr{
							dst.NewIdent(membersVariableIdentifier),
						},
					},
				},
			},
		}

		g.addDecls(
			&dst.FuncDecl{
				Name: dst.NewIdent("init"),
				Type: &dst.FuncType{},
				Body: &dst.BlockStmt{
					List: stmts,
				},
			},
		)
	}

	return
}

func (*generator) VisitAttachmentDeclaration(_ *ast.AttachmentDeclaration) struct{} {
	panic("attachment declarations are not supported")
}

func (*generator) VisitTransactionDeclaration(_ *ast.TransactionDeclaration) struct{} {
	panic("transaction declarations are not supported")
}

func (g *generator) VisitEntitlementDeclaration(decl *ast.EntitlementDeclaration) (_ struct{}) {
	entitlementName := decl.Identifier.Identifier
	typeVarName := typeVarName(entitlementName)
	typeVarDecl := entitlementTypeLiteral(entitlementName)

	g.addDecls(
		goVarDecl(
			typeVarName,
			typeVarDecl,
		),
	)

	return
}

func (g *generator) VisitEntitlementMappingDeclaration(decl *ast.EntitlementMappingDeclaration) (_ struct{}) {

	entitlementMappingName := decl.Identifier.Identifier
	typeVarName := typeVarName(entitlementMappingName)
	typeVarDecl := entitlementMapTypeLiteral(entitlementMappingName, decl.Elements)

	g.addDecls(
		goVarDecl(
			typeVarName,
			typeVarDecl,
		),
	)

	return
}

func (g *generator) VisitFieldDeclaration(decl *ast.FieldDeclaration) (_ struct{}) {
	fieldName := decl.Identifier.Identifier
	fullTypeName := g.currentFullTypeName()
	docString := g.declarationDocString(decl)

	g.addDecls(
		goConstDecl(
			fieldNameVarName(fullTypeName, fieldName),
			goStringLit(fieldName),
		),
		goVarDecl(
			fieldTypeVarName(fullTypeName, fieldName),
			typeExpr(decl.TypeAnnotation.Type, nil),
		),
		goConstDecl(
			fieldDocStringVarName(fullTypeName, fieldName),
			docString,
		),
	)

	return
}

func (g *generator) currentFullTypeName() string {
	return g.currentTypeDecl().fullTypeName
}

func (g *generator) currentTypeDecl() *typeDecl {
	return g.typeStack[len(g.typeStack)-1]
}

func typeExpr(t ast.Type, typeParams map[string]string) dst.Expr {
	switch t := t.(type) {
	case *ast.NominalType:
		identifier := t.Identifier.Identifier

		typeParamVarName, ok := typeParams[identifier]
		if ok {
			return &dst.UnaryExpr{
				Op: token.AND,
				X: &dst.CompositeLit{
					Type: &dst.Ident{
						Name: "GenericType",
						Path: semaPath,
					},
					Elts: []dst.Expr{
						goKeyValue("TypeParameter", dst.NewIdent(typeParamVarName)),
					},
				},
			}
		}

		inSema := sema.BaseTypeActivation.Find(identifier) != nil

		switch identifier {
		case "":
			identifier = "Void"
			inSema = true
		case "Any":
			inSema = true
		case "Address":
			identifier = "TheAddress"
		case "Type":
			identifier = "Meta"
		case "Capability":
			return &dst.UnaryExpr{
				Op: token.AND,
				X: &dst.CompositeLit{
					Type: &dst.Ident{
						Name: "CapabilityType",
						Path: semaPath,
					},
				},
			}
		default:
			var fullIdentifier strings.Builder
			fullIdentifier.WriteString(escapeTypeName(identifier))

			for _, nestedIdentifier := range t.NestedIdentifiers {
				fullIdentifier.WriteByte(typeNameSeparator)
				fullIdentifier.WriteString(escapeTypeName(nestedIdentifier.Identifier))
			}

			identifier = fullIdentifier.String()
		}

		ident := typeVarIdent(identifier)
		if inSema {
			ident.Path = semaPath
		}
		return ident

	case *ast.OptionalType:
		innerType := typeExpr(t.Type, typeParams)
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: &dst.Ident{
					Name: "OptionalType",
					Path: semaPath,
				},
				Elts: []dst.Expr{
					goKeyValue("Type", innerType),
				},
			},
		}

	case *ast.ReferenceType:
		borrowType := typeExpr(t.Type, typeParams)
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: &dst.Ident{
					Name: "ReferenceType",
					Path: semaPath,
				},
				Elts: []dst.Expr{
					goKeyValue("Type", borrowType),
					// TODO: add support for parsing entitlements
					goKeyValue(
						"Authorization",
						&dst.Ident{
							Name: "UnauthorizedAccess",
							Path: semaPath,
						},
					),
				},
			},
		}

	case *ast.VariableSizedType:
		elementType := typeExpr(t.Type, typeParams)
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: &dst.Ident{
					Name: "VariableSizedType",
					Path: semaPath,
				},
				Elts: []dst.Expr{
					goKeyValue("Type", elementType),
				},
			},
		}

	case *ast.ConstantSizedType:
		elementType := typeExpr(t.Type, typeParams)
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: &dst.Ident{
					Name: "ConstantSizedType",
					Path: semaPath,
				},
				Elts: []dst.Expr{
					goKeyValue("Type", elementType),
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

	case *ast.DictionaryType:
		keyType := typeExpr(t.KeyType, typeParams)
		valueType := typeExpr(t.ValueType, typeParams)
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: &dst.Ident{
					Name: "DictionaryType",
					Path: semaPath,
				},
				Elts: []dst.Expr{
					goKeyValue("KeyType", keyType),
					goKeyValue("ValueType", valueType),
				},
			},
		}

	case *ast.FunctionType:
		return functionTypeExpr(
			t,
			nil,
			nil,
			typeParams,
			false,
		)

	case *ast.InstantiationType:
		typeArguments := t.TypeArguments
		argumentExprs := []dst.Expr{
			typeExpr(t.Type, typeParams),
		}

		for _, argument := range typeArguments {
			argumentExprs = append(
				argumentExprs,
				typeExpr(argument.Type, typeParams),
			)
		}

		for _, expr := range argumentExprs {
			expr.Decorations().Before = dst.NewLine
			expr.Decorations().After = dst.NewLine
		}

		return &dst.CallExpr{
			Fun: &dst.Ident{
				Name: "MustInstantiate",
				Path: semaPath,
			},
			Args: argumentExprs,
		}

	case *ast.IntersectionType:
		var elements []dst.Expr

		if len(t.Types) > 0 {
			intersectedTypes := make([]dst.Expr, 0, len(t.Types))
			for _, intersectedType := range t.Types {
				intersectedTypes = append(
					intersectedTypes,
					typeExpr(intersectedType, typeParams),
				)
			}
			elements = append(
				elements,
				goKeyValue("Types",
					&dst.CompositeLit{
						Type: &dst.ArrayType{
							Elt: &dst.StarExpr{
								X: &dst.Ident{
									Name: "InterfaceType",
									Path: semaPath,
								},
							},
						},
						Elts: intersectedTypes,
					},
				),
			)
		}

		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: &dst.Ident{
					Name: "IntersectionType",
					Path: semaPath,
				},
				Elts: elements,
			},
		}

	default:
		panic(fmt.Errorf("%T types are not supported", t))
	}
}

func functionTypeExpr(
	t *ast.FunctionType,
	parameters *ast.ParameterList,
	typeParameterList *ast.TypeParameterList,
	typeParams map[string]string,
	isConstructor bool,
) dst.Expr {

	// Function purity

	var purityExpr dst.Expr
	if t.PurityAnnotation == ast.FunctionPurityView {
		purityExpr = &dst.Ident{
			Name: "FunctionPurityView",
			Path: semaPath,
		}
	}

	// Type parameters

	var typeParameterTypeAnnotations []*ast.TypeParameter
	if typeParameterList != nil {
		typeParameterTypeAnnotations = typeParameterList.TypeParameters
	}
	typeParameterCount := len(typeParameterTypeAnnotations)

	var typeParametersExpr dst.Expr

	if typeParameterCount > 0 {
		typeParameterExprs := make([]dst.Expr, 0, typeParameterCount)

		for _, typeParameterTypeAnnotation := range typeParameterTypeAnnotations {
			typeParameterName := typeParameterTypeAnnotation.Identifier.Identifier
			typeParameterExpr := dst.NewIdent(typeParams[typeParameterName])

			typeParameterExpr.Decorations().Before = dst.NewLine
			typeParameterExpr.Decorations().After = dst.NewLine

			typeParameterExprs = append(
				typeParameterExprs,
				typeParameterExpr,
			)
		}

		typeParametersExpr = &dst.CompositeLit{
			Type: &dst.ArrayType{
				Elt: &dst.StarExpr{
					X: &dst.Ident{
						Name: "TypeParameter",
						Path: semaPath,
					},
				},
			},
			Elts: typeParameterExprs,
		}
	}

	// Parameters

	parameterTypeAnnotations := t.ParameterTypeAnnotations
	parameterCount := len(parameterTypeAnnotations)

	var parametersExpr dst.Expr

	if parameterCount > 0 {
		parameterExprs := make([]dst.Expr, 0, parameterCount)

		for parameterIndex, parameterTypeAnnotation := range parameterTypeAnnotations {

			var parameterElements []dst.Expr

			if parameters != nil {
				parameter := parameters.Parameters[parameterIndex]

				if parameter.Label != "" {
					var lit dst.Expr
					// NOTE: avoid import of sema (ArgumentLabelNotRequired),
					// so sema can be in a non-buildable state
					// and code generation will still succeed
					if parameter.Label == "_" {
						lit = &dst.Ident{
							Name: "ArgumentLabelNotRequired",
							Path: semaPath,
						}
					} else {
						lit = goStringLit(parameter.Label)
					}

					parameterElements = append(
						parameterElements,
						goKeyValue("Label", lit),
					)
				}

				parameterElements = append(
					parameterElements,
					goKeyValue("Identifier", goStringLit(parameter.Identifier.Identifier)),
				)
			}

			parameterElements = append(
				parameterElements,
				goKeyValue(
					"TypeAnnotation",
					typeAnnotationCallExpr(typeExpr(parameterTypeAnnotation.Type, typeParams)),
				),
			)

			parameterExpr := &dst.CompositeLit{
				Elts: parameterElements,
			}

			parameterExpr.Decorations().Before = dst.NewLine
			parameterExpr.Decorations().After = dst.NewLine

			parameterExprs = append(
				parameterExprs,
				parameterExpr,
			)
		}

		parametersExpr = &dst.CompositeLit{
			Type: &dst.ArrayType{
				Elt: &dst.Ident{
					Name: "Parameter",
					Path: semaPath,
				},
			},
			Elts: parameterExprs,
		}
	}

	// Return type

	var returnTypeExpr dst.Expr
	if t.ReturnTypeAnnotation != nil {
		returnTypeExpr = typeExpr(t.ReturnTypeAnnotation.Type, typeParams)
	} else {
		returnTypeExpr = typeExpr(
			&ast.NominalType{
				Identifier: ast.Identifier{
					Identifier: "Void",
				},
			},
			nil,
		)
	}

	returnTypeExpr.Decorations().Before = dst.NewLine
	returnTypeExpr.Decorations().After = dst.NewLine

	// Composite literal elements

	var compositeElements []dst.Expr

	if purityExpr != nil {
		compositeElements = append(
			compositeElements,
			goKeyValue(
				"Purity",
				purityExpr,
			),
		)
	}

	if isConstructor {
		compositeElements = append(
			compositeElements,
			goKeyValue(
				"IsConstructor",
				goBoolLit(true),
			),
		)
	}

	if typeParametersExpr != nil {
		compositeElements = append(
			compositeElements,
			goKeyValue(
				"TypeParameters",
				typeParametersExpr,
			),
		)
	}

	if parametersExpr != nil {
		compositeElements = append(
			compositeElements,
			goKeyValue(
				"Parameters",
				parametersExpr,
			),
		)
	}

	compositeElements = append(
		compositeElements,
		goKeyValue(
			"ReturnTypeAnnotation",
			typeAnnotationCallExpr(returnTypeExpr),
		),
	)

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "FunctionType",
				Path: semaPath,
			},
			Elts: compositeElements,
		},
	}
}

func (*generator) VisitEnumCaseDeclaration(_ *ast.EnumCaseDeclaration) struct{} {
	panic("enum case declarations are not supported")
}

func (g *generator) VisitPragmaDeclaration(pragma *ast.PragmaDeclaration) (_ struct{}) {
	// Treat pragmas as part of the declaration to follow.

	identifierExpr, ok := pragma.Expression.(*ast.IdentifierExpression)
	if !ok {
		panic("only identifier pragmas are supported")
	}

	if g.leadingPragma == nil {
		g.leadingPragma = map[string]struct{}{}
	}
	g.leadingPragma[identifierExpr.Identifier.Identifier] = struct{}{}

	return
}

func (*generator) VisitImportDeclaration(_ *ast.ImportDeclaration) struct{} {
	panic("import declarations are not supported")
}

const typeNameSeparator = '_'

func joinTypeName(parentFullTypeName string, typeName string) string {
	return fmt.Sprintf(
		"%s%c%s",
		escapeTypeName(parentFullTypeName),
		typeNameSeparator,
		escapeTypeName(typeName),
	)
}

func (g *generator) newFullTypeName(typeName string) string {
	if len(g.typeStack) == 0 {
		return typeName
	}
	parentFullTypeName := g.typeStack[len(g.typeStack)-1].fullTypeName
	return joinTypeName(parentFullTypeName, typeName)
}

func escapeTypeName(typeName string) string {
	return strings.ReplaceAll(typeName, string(typeNameSeparator), "__")
}

func (g *generator) currentTypeID() string {
	var b strings.Builder
	for i := range g.typeStack {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(g.typeStack[i].typeName)
	}
	return b.String()
}

func (g *generator) currentMemberID(memberName string) string {
	var b strings.Builder
	for i := range g.typeStack {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(g.typeStack[i].typeName)
	}
	b.WriteByte('.')
	b.WriteString(memberName)
	return b.String()
}

func (g *generator) generateTypeInit(program *ast.Program) {

	// Currently this only generate registering of entitlements and entitlement mappings.
	// It is possible to extend this to register other types as well.
	// So they are not needed to be manually added to the base activation.
	//
	// Generates the following:
	//
	//   func init() {
	//       BuiltinEntitlements[FooEntitlement.Identifier] = FooEntitlement
	//
	//       ...
	//
	//       BuiltinEntitlements[BarEntitlementMapping.Identifier] = BarEntitlementMapping
	//
	//       ...
	//   }
	//

	var stmts []dst.Stmt

	for _, declaration := range program.EntitlementMappingDeclarations() {
		stmts = append(stmts, entitlementMapInitStatements(declaration)...)
	}

	for _, declaration := range program.EntitlementDeclarations() {
		stmts = append(stmts, entitlementInitStatements(declaration)...)
	}

	if len(stmts) == 0 {
		return
	}

	for _, stmt := range stmts {
		stmt.Decorations().Before = dst.NewLine
		stmt.Decorations().After = dst.NewLine
	}

	initDecl := &dst.FuncDecl{
		Name: dst.NewIdent("init"),
		Type: &dst.FuncType{},
		Body: &dst.BlockStmt{
			List: stmts,
		},
	}

	g.addDecls(initDecl)
}

func entitlementMapInitStatements(declaration *ast.EntitlementMappingDeclaration) []dst.Stmt {
	const mapName = "BuiltinEntitlementMappings"
	varName := typeVarName(declaration.Identifier.Identifier)

	mapUpdateStmt := &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.IndexExpr{
				X: &dst.Ident{
					Name: mapName,
					Path: semaPath,
				},
				Index: &dst.SelectorExpr{
					X:   dst.NewIdent(varName),
					Sel: dst.NewIdent("Identifier"),
				},
			},
		},
		Tok: token.ASSIGN,
		Rhs: []dst.Expr{
			dst.NewIdent(varName),
		},
	}

	return []dst.Stmt{
		mapUpdateStmt,
	}
}

func entitlementInitStatements(declaration *ast.EntitlementDeclaration) []dst.Stmt {
	const mapName = "BuiltinEntitlements"
	varName := typeVarName(declaration.Identifier.Identifier)

	mapUpdateStmt := &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.IndexExpr{
				X: &dst.Ident{
					Name: mapName,
					Path: semaPath,
				},
				Index: &dst.SelectorExpr{
					X:   dst.NewIdent(varName),
					Sel: dst.NewIdent("Identifier"),
				},
			},
		},
		Tok: token.ASSIGN,
		Rhs: []dst.Expr{
			dst.NewIdent(varName),
		},
	}

	return []dst.Stmt{
		mapUpdateStmt,
	}
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

func compositeKindExpr(compositeKind common.CompositeKind) *dst.Ident {
	return &dst.Ident{
		Path: "github.com/onflow/cadence/runtime/common",
		Name: compositeKind.String(),
	}
}

func typeVarName(fullTypeName string) string {
	return fmt.Sprintf("%sType", fullTypeName)
}

func typeVarIdent(fullTypeName string) *dst.Ident {
	return dst.NewIdent(typeVarName(fullTypeName))
}

func typeNameVarName(fullTypeName string) string {
	return fmt.Sprintf("%sTypeName", fullTypeName)
}

func typeNameVarIdent(fullTypeName string) *dst.Ident {
	return dst.NewIdent(typeNameVarName(fullTypeName))
}

func typeTagVarIdent(fullTypeName string) *dst.Ident {
	return dst.NewIdent(fmt.Sprintf("%sTypeTag", fullTypeName))
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

func constructorTypeVarName(fullTypeName string) string {
	return memberVarName(fullTypeName, "", "Constructor", "Type")
}

func functionTypeParameterVarName(fullTypeName, functionName, typeParameterName string) string {
	return memberVarName(fullTypeName, functionName, "Function", "TypeParameter"+typeParameterName)
}

func fieldDocStringVarName(fullTypeName, fieldName string) string {
	return memberVarName(fullTypeName, fieldName, "Field", "DocString")
}

func functionDocStringVarName(fullTypeName, functionName string) string {
	return memberVarName(fullTypeName, functionName, "Function", "DocString")
}

func constructorDocStringVarName(fullTypeName string) string {
	return memberVarName(fullTypeName, "", "Constructor", "DocString")
}

func simpleTypeLiteral(ty *typeDecl) dst.Expr {

	// &SimpleType{
	//	Name:          TestTypeName,
	//	QualifiedName: TestTypeName,
	//	TypeID:        TestTypeName,
	//	tag:           TestTypeTag,
	//	IsResource:    true,
	//	Storable:      false,
	//	Primitive:     false,
	//	Equatable:     false,
	//	Comparable:    false,
	//	Exportable:    false,
	//	Importable:    false,
	//  comformances:  []*InterfaceType {
	//      StructStringer,
	//  }
	//}

	isResource := ty.compositeKind == common.CompositeKindResource
	elements := []dst.Expr{
		goKeyValue("Name", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("QualifiedName", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("TypeID", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("TypeTag", typeTagVarIdent(ty.fullTypeName)),
		goKeyValue("IsResource", goBoolLit(isResource)),
		goKeyValue("Storable", goBoolLit(ty.storable)),
		goKeyValue("Primitive", goBoolLit(ty.primitive)),
		goKeyValue("Equatable", goBoolLit(ty.equatable)),
		goKeyValue("Comparable", goBoolLit(ty.comparable)),
		goKeyValue("Exportable", goBoolLit(ty.exportable)),
		goKeyValue("Importable", goBoolLit(ty.importable)),
		goKeyValue("ContainFields", goBoolLit(ty.memberAccessible)),
	}

	if len(ty.conformances) > 0 {
		var elts = []dst.Expr{}
		for _, conformance := range ty.conformances {
			elts = append(elts, &dst.Ident{
				Name: conformance,
				Path: semaPath,
			})
		}
		elements = append(elements, goKeyValue("conformances", &dst.CompositeLit{
			Type: &dst.ArrayType{
				Elt: &dst.StarExpr{
					X: &dst.Ident{
						Name: "InterfaceType",
					},
				},
			},
			Elts: elts,
		}))
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "SimpleType",
				Path: semaPath,
			},
			Elts: elements,
		},
	}
}

func simpleTypeMemberResolversFunc(fullTypeName string, declarations []ast.Declaration) dst.Expr {
	// func(t *SimpleType) map[string]MemberResolver {
	//   return MembersAsResolvers(...)
	// }

	const typeVarName = "t"

	returnStatement := &dst.ReturnStmt{
		Results: []dst.Expr{
			&dst.CallExpr{
				Fun: &dst.Ident{
					Name: "MembersAsResolvers",
					Path: semaPath,
				},
				Args: []dst.Expr{
					membersExpr(
						fullTypeName,
						typeVarName,
						declarations,
					),
				},
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
					goField(typeVarName, simpleType()),
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

func membersExpr(
	fullTypeName string,
	typeVarName string,
	memberDeclarations []ast.Declaration,
) dst.Expr {

	// []*Member{
	//   ...
	// }

	elements := make([]dst.Expr, 0, len(memberDeclarations))

	for _, declaration := range memberDeclarations {
		var memberVarName string
		memberName := declaration.DeclarationIdentifier().Identifier

		declarationKind := declaration.DeclarationKind()
		switch declarationKind {
		case common.DeclarationKindField:
			memberVarName = fieldNameVarName(
				fullTypeName,
				memberName,
			)

		case common.DeclarationKindFunction:
			memberVarName = functionNameVarName(
				fullTypeName,
				memberName,
			)

		case common.DeclarationKindInitializer:
			// Generated as a member of the container
			continue

		case common.DeclarationKindStructureInterface,
			common.DeclarationKindStructure,
			common.DeclarationKindResource,
			common.DeclarationKindResourceInterface:

			initializers := declaration.DeclarationMembers().Initializers()
			if len(initializers) > 0 {
				initializer := initializers[0]

				typeName := declaration.DeclarationIdentifier().Identifier

				element := newDeclarationMember(
					joinTypeName(fullTypeName, typeName),
					typeVarName,
					// type name is used instead
					"",
					initializer,
				)
				element.Decorations().Before = dst.NewLine
				element.Decorations().After = dst.NewLine

				elements = append(elements, element)
			}

			continue

		default:
			panic(fmt.Errorf(
				"%s members are not supported",
				declarationKind.Name(),
			))
		}

		element := newDeclarationMember(
			fullTypeName,
			typeVarName,
			memberVarName,
			declaration,
		)
		element.Decorations().Before = dst.NewLine
		element.Decorations().After = dst.NewLine

		elements = append(elements, element)
	}

	return &dst.CompositeLit{
		Type: &dst.ArrayType{
			Elt: &dst.StarExpr{
				X: &dst.Ident{
					Name: "Member",
					Path: semaPath,
				},
			},
		},
		Elts: elements,
	}
}

func simpleType() *dst.StarExpr {
	return &dst.StarExpr{
		X: &dst.Ident{
			Name: "SimpleType",
			Path: semaPath,
		},
	}
}

func accessExpr(access ast.Access) dst.Expr {
	switch access := access.(type) {
	case ast.PrimitiveAccess:
		return &dst.CallExpr{
			Fun: &dst.Ident{
				Name: "PrimitiveAccess",
				Path: semaPath,
			},
			Args: []dst.Expr{
				&dst.Ident{
					Name: access.String(),
					Path: astPath,
				},
			},
		}

	case ast.EntitlementAccess:
		entitlements := access.EntitlementSet.Entitlements()

		entitlementExprs := make([]dst.Expr, 0, len(entitlements))

		for _, nominalType := range entitlements {
			entitlementExpr := typeExpr(nominalType, nil)
			entitlementExprs = append(entitlementExprs, entitlementExpr)
		}

		var setKind dst.Expr

		switch access.EntitlementSet.Separator() {
		case ast.Conjunction:
			setKind = &dst.Ident{
				Name: "Conjunction",
				Path: semaPath,
			}
		case ast.Disjunction:
			setKind = &dst.Ident{
				Name: "Disjunction",
				Path: semaPath,
			}
		default:
			panic(errors.NewUnreachableError())
		}

		args := []dst.Expr{
			&dst.CompositeLit{
				Type: &dst.ArrayType{
					Elt: &dst.Ident{
						Name: "Type",
						Path: semaPath,
					},
				},
				Elts: entitlementExprs,
			},
			setKind,
		}

		for _, arg := range args {
			arg.Decorations().Before = dst.NewLine
			arg.Decorations().After = dst.NewLine
		}

		return &dst.CallExpr{
			Fun: &dst.Ident{
				Name: "newEntitlementAccess",
				Path: semaPath,
			},
			Args: args,
		}

	default:
		panic(fmt.Errorf("unsupported access: %#+v\n", access))
	}
}

func variableKindIdent(variableKind ast.VariableKind) *dst.Ident {
	return &dst.Ident{
		Name: variableKind.String(),
		Path: astPath,
	}
}

func newDeclarationMember(
	fullTypeName string,
	containerTypeVariableIdentifier string,
	memberNameVariableIdentifier string,
	declaration ast.Declaration,
) dst.Expr {
	declarationName := declaration.DeclarationIdentifier().Identifier

	// Field

	access := declaration.DeclarationAccess()
	if access == ast.AccessNotSpecified {
		switch declaration.DeclarationKind() {
		case common.DeclarationKindInitializer:
			access = ast.AccessAll

		default:
			panic(fmt.Errorf(
				"member with unspecified access: %s.%s",
				fullTypeName,
				declarationName,
			))
		}
	}

	if fieldDeclaration, ok := declaration.(*ast.FieldDeclaration); ok {
		args := []dst.Expr{
			dst.NewIdent(containerTypeVariableIdentifier),
			accessExpr(access),
			variableKindIdent(fieldDeclaration.VariableKind),
			dst.NewIdent(memberNameVariableIdentifier),
			dst.NewIdent(fieldTypeVarName(fullTypeName, declarationName)),
			dst.NewIdent(fieldDocStringVarName(fullTypeName, declarationName)),
		}

		for _, arg := range args {
			arg.Decorations().Before = dst.NewLine
			arg.Decorations().After = dst.NewLine
		}

		return &dst.CallExpr{
			Fun: &dst.Ident{
				Name: "NewUnmeteredFieldMember",
				Path: semaPath,
			},
			Args: args,
		}
	}

	declarationKind := declaration.DeclarationKind()

	// Function or initializer

	switch declarationKind {
	case common.DeclarationKindFunction:
		args := []dst.Expr{
			dst.NewIdent(containerTypeVariableIdentifier),
			accessExpr(access),
			dst.NewIdent(memberNameVariableIdentifier),
			dst.NewIdent(functionTypeVarName(fullTypeName, declarationName)),
			dst.NewIdent(functionDocStringVarName(fullTypeName, declarationName)),
		}

		for _, arg := range args {
			arg.Decorations().Before = dst.NewLine
			arg.Decorations().After = dst.NewLine
		}

		return &dst.CallExpr{
			Fun: &dst.Ident{
				Name: "NewUnmeteredFunctionMember",
				Path: semaPath,
			},
			Args: args,
		}

	case common.DeclarationKindInitializer:
		args := []dst.Expr{
			dst.NewIdent(containerTypeVariableIdentifier),
			accessExpr(access),
			typeNameVarIdent(fullTypeName),
			dst.NewIdent(constructorTypeVarName(fullTypeName)),
			dst.NewIdent(constructorDocStringVarName(fullTypeName)),
		}

		for _, arg := range args {
			arg.Decorations().Before = dst.NewLine
			arg.Decorations().After = dst.NewLine
		}

		return &dst.CallExpr{
			Fun: &dst.Ident{
				Name: "NewUnmeteredConstructorMember",
				Path: semaPath,
			},
			Args: args,
		}
	}

	// Unsupported

	panic(fmt.Errorf(
		"%s members are not supported",
		declarationKind.Name(),
	))
}

func stringMemberResolverMapType() *dst.MapType {
	return &dst.MapType{
		Key: dst.NewIdent("string"),
		Value: &dst.Ident{
			Name: "MemberResolver",
			Path: semaPath,
		},
	}
}

func compositeTypeExpr(ty *typeDecl) dst.Expr {

	// func() *CompositeType {
	// 	var t = &CompositeType{
	// 		Identifier:         FooTypeName,
	// 		Kind:               common.CompositeKindStructure,
	// 		ImportableBuiltin:  false,
	// 		HasComputedMembers: true,
	// 	}
	//
	// 	t.SetNestedType(FooBarTypeName, FooBarType)
	// 	return t
	// }()

	const typeVarName = "t"

	statements := []dst.Stmt{
		&dst.DeclStmt{
			Decl: goVarDecl(
				typeVarName,
				compositeTypeLiteral(ty),
			),
		},
	}

	for _, nestedType := range ty.nestedTypes {
		statements = append(
			statements,
			&dst.ExprStmt{
				X: &dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   dst.NewIdent(typeVarName),
						Sel: dst.NewIdent("SetNestedType"),
					},
					Args: []dst.Expr{
						typeNameVarIdent(nestedType.fullTypeName),
						typeVarIdent(nestedType.fullTypeName),
					},
				},
			},
		)
	}

	statements = append(
		statements,
		&dst.ReturnStmt{
			Results: []dst.Expr{
				dst.NewIdent(typeVarName),
			},
		},
	)

	return &dst.CallExpr{
		Fun: &dst.FuncLit{
			Type: &dst.FuncType{
				Func: true,
				Results: &dst.FieldList{
					List: []*dst.Field{
						{
							Type: &dst.StarExpr{
								X: &dst.Ident{
									Name: "CompositeType",
									Path: semaPath,
								},
							},
						},
					},
				},
			},
			Body: &dst.BlockStmt{
				List: statements,
			},
		},
	}
}

func compositeTypeLiteral(ty *typeDecl) dst.Expr {
	kind := compositeKindExpr(ty.compositeKind)

	elements := []dst.Expr{
		goKeyValue("Identifier", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("Kind", kind),
		goKeyValue("ImportableBuiltin", goBoolLit(ty.importable)),
		goKeyValue("HasComputedMembers", goBoolLit(true)),
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "CompositeType",
				Path: semaPath,
			},
			Elts: elements,
		},
	}
}

func interfaceTypeExpr(ty *typeDecl) dst.Expr {

	// func() *InterfaceType {
	// 	var t = &InterfaceType{
	// 		Identifier:         FooTypeName,
	// 		CompositeKind:      common.CompositeKindStructure,
	// 	}
	//
	// 	t.SetNestedType(FooBarTypeName, FooBarType)
	// 	return t
	// }()

	const typeVarName = "t"

	statements := []dst.Stmt{
		&dst.DeclStmt{
			Decl: goVarDecl(
				typeVarName,
				interfaceTypeLiteral(ty),
			),
		},
	}

	for _, nestedType := range ty.nestedTypes {
		statements = append(
			statements,
			&dst.ExprStmt{
				X: &dst.CallExpr{
					Fun: &dst.SelectorExpr{
						X:   dst.NewIdent(typeVarName),
						Sel: dst.NewIdent("SetNestedType"),
					},
					Args: []dst.Expr{
						typeNameVarIdent(nestedType.fullTypeName),
						typeVarIdent(nestedType.fullTypeName),
					},
				},
			},
		)
	}

	statements = append(
		statements,
		&dst.ReturnStmt{
			Results: []dst.Expr{
				dst.NewIdent(typeVarName),
			},
		},
	)

	return &dst.CallExpr{
		Fun: &dst.FuncLit{
			Type: &dst.FuncType{
				Func: true,
				Results: &dst.FieldList{
					List: []*dst.Field{
						{
							Type: &dst.StarExpr{
								X: &dst.Ident{
									Name: "InterfaceType",
									Path: semaPath,
								},
							},
						},
					},
				},
			},
			Body: &dst.BlockStmt{
				List: statements,
			},
		},
	}
}

func interfaceTypeLiteral(ty *typeDecl) dst.Expr {
	kind := compositeKindExpr(ty.compositeKind)

	elements := []dst.Expr{
		goKeyValue("Identifier", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("CompositeKind", kind),
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "InterfaceType",
				Path: semaPath,
			},
			Elts: elements,
		},
	}
}

func typeAnnotationCallExpr(ty dst.Expr) *dst.CallExpr {
	return &dst.CallExpr{
		Fun: &dst.Ident{
			Name: "NewTypeAnnotation",
			Path: semaPath,
		},
		Args: []dst.Expr{
			ty,
		},
	}
}

func typeParameterExpr(name string, typeBound dst.Expr) dst.Expr {
	elements := []dst.Expr{
		goKeyValue("Name", goStringLit(name)),
	}
	if typeBound != nil {
		elements = append(
			elements,
			goKeyValue("TypeBound", typeBound),
		)
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "TypeParameter",
				Path: semaPath,
			},
			Elts: elements,
		},
	}
}

func entitlementTypeLiteral(name string) dst.Expr {
	// &sema.EntitlementType{
	//	Identifier: "Foo",
	//}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "EntitlementType",
				Path: semaPath,
			},
			Elts: []dst.Expr{
				goKeyValue("Identifier", goStringLit(name)),
			},
		},
	}
}

func entitlementMapTypeLiteral(name string, elements []ast.EntitlementMapElement) dst.Expr {
	// &sema.EntitlementMapType{
	//	Identifier: "Foo",
	//	Relations: []EntitlementRelation{
	//		{
	//			Input: BarType,
	//			Output: BazType,
	//		},
	//	}
	// }

	includesIdentity := false
	relationExprs := make([]dst.Expr, 0, len(elements))

	for _, element := range elements {

		relation, isRelation := element.(*ast.EntitlementMapRelation)
		include, isInclude := element.(*ast.NominalType)
		if !isRelation && !isInclude {
			panic(fmt.Errorf("invalid map element: expected relations or include, got '%s'", element))
		}
		if isInclude && include.Identifier.Identifier == "Identity" {
			includesIdentity = true
			continue
		} else if isInclude {
			panic(fmt.Errorf("non-Identity map include is not supported: %s", element))
		}

		relationExpr := &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "EntitlementRelation",
				Path: semaPath,
			},
			Elts: []dst.Expr{
				goKeyValue("Input", typeExpr(relation.Input, nil)),
				goKeyValue("Output", typeExpr(relation.Output, nil)),
			},
		}

		relationExpr.Decorations().Before = dst.NewLine
		relationExpr.Decorations().After = dst.NewLine

		relationExprs = append(relationExprs, relationExpr)
	}

	relationsExpr := &dst.CompositeLit{
		Type: &dst.ArrayType{
			Elt: &dst.Ident{
				Name: "EntitlementRelation",
				Path: semaPath,
			},
		},
		Elts: relationExprs,
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: &dst.Ident{
				Name: "EntitlementMapType",
				Path: semaPath,
			},
			Elts: []dst.Expr{
				goKeyValue("Identifier", goStringLit(name)),
				goKeyValue("IncludesIdentity", goBoolLit(includesIdentity)),
				goKeyValue("Relations", relationsExpr),
			},
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

func gen(inPath string, outFile *os.File, packagePath string) {
	program := parseCadenceFile(inPath)

	var gen generator

	for _, declaration := range program.Declarations() {
		generateDeclaration(&gen, declaration)
	}

	gen.generateTypeInit(program)

	writeGoFile(inPath, outFile, gen.decls, packagePath)
}

func generateDeclaration(gen *generator, declaration ast.Declaration) {
	// Treat leading pragmas as part of this declaration.
	// Reset them after finishing the current decl. This is to handle nested declarations.
	if declaration.DeclarationKind() != common.DeclarationKindPragma {
		prevLeadingPragma := gen.leadingPragma
		defer func() {
			gen.leadingPragma = prevLeadingPragma
		}()
	}

	_ = ast.AcceptDeclaration[struct{}](declaration, gen)
}

func writeGoFile(inPath string, outFile *os.File, decls []dst.Decl, packagePath string) {
	err := parsedHeaderTemplate.Execute(outFile, inPath)
	if err != nil {
		panic(err)
	}

	resolver := guess.New()
	restorer := decorator.NewRestorerWithImports(packagePath, resolver)

	packageName, err := resolver.ResolvePackage(packagePath)
	if err != nil {
		panic(err)
	}

	err = restorer.Fprint(
		outFile,
		&dst.File{
			Name:  dst.NewIdent(packageName),
			Decls: decls,
		},
	)
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Parse()
	argumentCount := flag.NArg()

	if argumentCount < 1 {
		panic("Missing path to input Cadence file")
	}
	if argumentCount < 2 {
		panic("Missing path to output Go file")
	}
	inPath := flag.Arg(0)
	outPath := flag.Arg(1)

	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	gen(inPath, outFile, *packagePathFlag)
}
