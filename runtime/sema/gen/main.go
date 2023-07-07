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
	"github.com/dave/dst/decorator/resolver/guess"

	"github.com/dave/dst"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/pretty"
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
	equatable          bool
	exportable         bool
	comparable         bool
	importable         bool
	memberAccessible   bool
	memberDeclarations []ast.Declaration
	nestedTypes        []*typeDecl
}

type generator struct {
	typeStack []*typeDecl
	decls     []dst.Decl
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

func (*generator) VisitSpecialFunctionDeclaration(_ *ast.SpecialFunctionDeclaration) struct{} {
	panic("special function declarations are not supported")
}

func (g *generator) VisitCompositeDeclaration(decl *ast.CompositeDeclaration) (_ struct{}) {

	compositeKind := decl.CompositeKind
	switch compositeKind {
	case common.CompositeKindStructure,
		common.CompositeKindResource:
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

	// We can generate a SimpleType declaration,
	// if this is a top-level type,
	// and this declaration has no nested type declarations.
	// Otherwise, we have to generate a CompositeType

	canGenerateSimpleType := len(g.typeStack) == 1

	for _, memberDeclaration := range decl.Members.Declarations() {
		ast.AcceptDeclaration[struct{}](memberDeclaration, g)

		// Visiting unsupported declarations panics,
		// so only supported member declarations are added
		typeDecl.memberDeclarations = append(
			typeDecl.memberDeclarations,
			memberDeclaration,
		)

		switch memberDeclaration.(type) {
		case *ast.FieldDeclaration,
			*ast.FunctionDeclaration:
			break

		default:
			canGenerateSimpleType = false
		}
	}

	for _, conformance := range decl.Conformances {
		switch conformance.Identifier.Identifier {
		case "Storable":
			if !canGenerateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as storable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.storable = true

		case "Equatable":
			if !canGenerateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as equatable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.equatable = true

		case "Comparable":
			if !canGenerateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as comparable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.comparable = true

		case "Exportable":
			if !canGenerateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as exportable: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.exportable = true

		case "Importable":
			typeDecl.importable = true

		case "ContainFields":
			if !canGenerateSimpleType {
				panic(fmt.Errorf(
					"composite types cannot be explicitly marked as having fields: %s",
					g.currentTypeID(),
				))
			}
			typeDecl.memberAccessible = true
		}
	}

	var typeVarDecl dst.Expr
	if canGenerateSimpleType {
		typeVarDecl = simpleTypeLiteral(typeDecl)
	} else {
		typeVarDecl = compositeTypeExpr(typeDecl)
	}

	tyVarName := typeVarName(typeDecl.fullTypeName)

	g.addDecls(
		goConstDecl(
			typeNameVarName(typeDecl.fullTypeName),
			goStringLit(typeName),
		),
		goVarDecl(
			tyVarName,
			typeVarDecl,
		),
	)

	memberDeclarations := typeDecl.memberDeclarations

	if len(memberDeclarations) > 0 {

		if canGenerateSimpleType {

			// func init() {
			//   t.Members = func(t *SimpleType) map[string]MemberResolver {
			//     return MembersAsResolvers(...)
			//   }
			// }

			memberResolversFunc := simpleTypeMemberResolversFunc(typeDecl.fullTypeName, memberDeclarations)

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
			// }

			members := membersExpr(typeDecl.fullTypeName, tyVarName, memberDeclarations)

			const membersVariableIdentifier = "members"

			g.addDecls(
				&dst.FuncDecl{
					Name: dst.NewIdent("init"),
					Type: &dst.FuncType{},
					Body: &dst.BlockStmt{
						List: []dst.Stmt{
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
										Fun: dst.NewIdent("MembersAsMap"),
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
										Fun: dst.NewIdent("MembersFieldNames"),
										Args: []dst.Expr{
											dst.NewIdent(membersVariableIdentifier),
										},
									},
								},
							},
						},
					},
				},
			)
		}
	}

	return
}

func (*generator) VisitInterfaceDeclaration(_ *ast.InterfaceDeclaration) struct{} {
	panic("interface declarations are not supported")
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
	typeVarDecl := entitlementMapTypeLiteral(entitlementMappingName, decl.Associations)

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
	return g.typeStack[len(g.typeStack)-1].fullTypeName
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
					Type: dst.NewIdent("GenericType"),
					Elts: []dst.Expr{
						goKeyValue("TypeParameter", dst.NewIdent(typeParamVarName)),
					},
				},
			}
		}

		switch identifier {
		case "":
			identifier = "Void"
		case "Address":
			identifier = "TheAddress"
		case "Type":
			identifier = "Meta"
		case "Capability":
			return &dst.UnaryExpr{
				Op: token.AND,
				X: &dst.CompositeLit{
					Type: dst.NewIdent("CapabilityType"),
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

		return typeVarIdent(identifier)

	case *ast.OptionalType:
		innerType := typeExpr(t.Type, typeParams)
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("OptionalType"),
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
				Type: dst.NewIdent("ReferenceType"),
				Elts: []dst.Expr{
					goKeyValue("Type", borrowType),
					// TODO: add support for parsing entitlements
					goKeyValue("Authorization", dst.NewIdent("UnauthorizedAccess")),
				},
			},
		}

	case *ast.VariableSizedType:
		elementType := typeExpr(t.Type, typeParams)
		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("VariableSizedType"),
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
				Type: dst.NewIdent("ConstantSizedType"),
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

	case *ast.FunctionType:
		return functionTypeExpr(t, nil, nil, typeParams)

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
			Fun:  dst.NewIdent("MustInstantiate"),
			Args: argumentExprs,
		}

	case *ast.RestrictedType:
		var elements []dst.Expr
		if t.Type != nil {
			restrictedType := typeExpr(t.Type, typeParams)
			elements = append(elements,
				goKeyValue("Type", restrictedType),
			)
		}

		if len(t.Restrictions) > 0 {
			restrictions := make([]dst.Expr, 0, len(t.Restrictions))
			for _, restriction := range t.Restrictions {
				restrictions = append(
					restrictions,
					typeExpr(restriction, typeParams),
				)
			}
			elements = append(
				elements,
				goKeyValue("Restrictions",
					&dst.CompositeLit{
						Type: &dst.ArrayType{
							Elt: &dst.StarExpr{
								X: dst.NewIdent("InterfaceType"),
							},
						},
						Elts: restrictions,
					},
				),
			)
		}

		return &dst.UnaryExpr{
			Op: token.AND,
			X: &dst.CompositeLit{
				Type: dst.NewIdent("RestrictedType"),
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
) dst.Expr {

	// Function purity

	var purityExpr dst.Expr
	if t.PurityAnnotation == ast.FunctionPurityView {
		purityExpr = dst.NewIdent("FunctionPurityView")
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
					X: dst.NewIdent("TypeParameter"),
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
				Elt: dst.NewIdent("Parameter"),
			},
			Elts: parameterExprs,
		}
	}

	// Return type

	var returnTypeExpr dst.Expr
	if t.ReturnTypeAnnotation != nil {
		returnTypeExpr = typeExpr(t.ReturnTypeAnnotation.Type, typeParams)
	} else {
		returnTypeExpr = typeVarIdent("Void")
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
			Type: dst.NewIdent("FunctionType"),
			Elts: compositeElements,
		},
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

const typeNameSeparator = '_'

func (g *generator) newFullTypeName(typeName string) string {
	if len(g.typeStack) == 0 {
		return typeName
	}
	parentFullTypeName := g.typeStack[len(g.typeStack)-1].fullTypeName
	return fmt.Sprintf(
		"%s%c%s",
		escapeTypeName(parentFullTypeName),
		typeNameSeparator,
		escapeTypeName(typeName),
	)
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
	//       addToBaseActivation(FooEntitlement)
	//
	//       ...
	//
	//       BuiltinEntitlements[BarEntitlementMapping.Identifier] = BarEntitlementMapping
	//       addToBaseActivation(BarEntitlementMapping)
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
				X: dst.NewIdent(mapName),
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

	typeRegisterStmt := &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: dst.NewIdent("addToBaseActivation"),
			Args: []dst.Expr{
				dst.NewIdent(varName),
			},
		},
	}

	return []dst.Stmt{
		mapUpdateStmt,
		typeRegisterStmt,
	}
}

func entitlementInitStatements(declaration *ast.EntitlementDeclaration) []dst.Stmt {
	const mapName = "BuiltinEntitlements"
	varName := typeVarName(declaration.Identifier.Identifier)

	mapUpdateStmt := &dst.AssignStmt{
		Lhs: []dst.Expr{
			&dst.IndexExpr{
				X: dst.NewIdent(mapName),
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

	typeRegisterStmt := &dst.ExprStmt{
		X: &dst.CallExpr{
			Fun: dst.NewIdent("addToBaseActivation"),
			Args: []dst.Expr{
				dst.NewIdent(varName),
			},
		},
	}

	return []dst.Stmt{
		mapUpdateStmt,
		typeRegisterStmt,
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

func functionTypeParameterVarName(fullTypeName, functionName, typeParameterName string) string {
	return memberVarName(fullTypeName, functionName, "Function", "TypeParameter"+typeParameterName)
}

func fieldDocStringVarName(fullTypeName, fieldName string) string {
	return memberVarName(fullTypeName, fieldName, "Field", "DocString")
}

func functionDocStringVarName(fullTypeName, functionName string) string {
	return memberVarName(fullTypeName, functionName, "Function", "DocString")
}

func simpleTypeLiteral(ty *typeDecl) dst.Expr {

	// &SimpleType{
	//	Name:          TestTypeName,
	//	QualifiedName: TestTypeName,
	//	TypeID:        TestTypeName,
	//	tag:           TestTypeTag,
	//	IsResource:    true,
	//	Storable:      false,
	//	Equatable:     false,
	//	Comparable:    false,
	//	Exportable:    false,
	//	Importable:    false,
	//}

	isResource := ty.compositeKind == common.CompositeKindResource
	elements := []dst.Expr{
		goKeyValue("Name", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("QualifiedName", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("TypeID", typeNameVarIdent(ty.fullTypeName)),
		goKeyValue("tag", typeTagVarIdent(ty.fullTypeName)),
		goKeyValue("IsResource", goBoolLit(isResource)),
		goKeyValue("Storable", goBoolLit(ty.storable)),
		goKeyValue("Equatable", goBoolLit(ty.equatable)),
		goKeyValue("Comparable", goBoolLit(ty.comparable)),
		goKeyValue("Exportable", goBoolLit(ty.exportable)),
		goKeyValue("Importable", goBoolLit(ty.importable)),
		goKeyValue("ContainFields", goBoolLit(ty.memberAccessible)),
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: dst.NewIdent("SimpleType"),
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
				Fun: dst.NewIdent("MembersAsResolvers"),
				Args: []dst.Expr{
					membersExpr(fullTypeName, typeVarName, declarations),
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

		case common.DeclarationKindStructureInterface,
			common.DeclarationKindStructure,
			common.DeclarationKindResource,
			common.DeclarationKindResourceInterface:

			continue

		default:
			panic(fmt.Errorf(
				"%s members are not supported",
				declarationKind.Name(),
			))
		}

		element := newDeclarationMember(fullTypeName, typeVarName, memberVarName, declaration)
		element.Decorations().Before = dst.NewLine
		element.Decorations().After = dst.NewLine

		elements = append(elements, element)
	}

	return &dst.CompositeLit{
		Type: &dst.ArrayType{
			Elt: &dst.StarExpr{X: dst.NewIdent("Member")},
		},
		Elts: elements,
	}
}

func simpleType() *dst.StarExpr {
	return &dst.StarExpr{X: dst.NewIdent("SimpleType")}
}

func accessExpr(access ast.Access) dst.Expr {
	switch access := access.(type) {
	case ast.PrimitiveAccess:
		return &dst.CallExpr{
			Fun: dst.NewIdent("PrimitiveAccess"),
			Args: []dst.Expr{
				&dst.Ident{
					Name: access.String(),
					Path: "github.com/onflow/cadence/runtime/ast",
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
			setKind = dst.NewIdent("Conjunction")
		case ast.Disjunction:
			setKind = dst.NewIdent("Disjunction")
		default:
			panic(errors.NewUnreachableError())
		}

		args := []dst.Expr{
			&dst.CompositeLit{
				Type: &dst.ArrayType{
					Elt: dst.NewIdent("Type"),
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
			Fun:  dst.NewIdent("newEntitlementAccess"),
			Args: args,
		}

	default:
		panic(fmt.Errorf("unsupported access: %#+v\n", access))
	}
}

func variableKindIdent(variableKind ast.VariableKind) *dst.Ident {
	return &dst.Ident{
		Name: variableKind.String(),
		Path: "github.com/onflow/cadence/runtime/ast",
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
		panic(fmt.Errorf(
			"member with unspecified access: %s.%s",
			fullTypeName,
			declarationName,
		))
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
			Fun:  dst.NewIdent("NewUnmeteredFieldMember"),
			Args: args,
		}
	}

	declarationKind := declaration.DeclarationKind()

	// Function

	if declarationKind == common.DeclarationKindFunction {
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
			Fun:  dst.NewIdent("NewUnmeteredFunctionMember"),
			Args: args,
		}
	}

	panic(fmt.Errorf(
		"%s members are not supported",
		declarationKind.Name(),
	))
}

func stringMemberResolverMapType() *dst.MapType {
	return &dst.MapType{
		Key:   dst.NewIdent("string"),
		Value: dst.NewIdent("MemberResolver"),
	}
}

func compositeTypeExpr(ty *typeDecl) dst.Expr {

	// func() *CompositeType {
	// 	var t = &CompositeType{
	// 		Identifier:         FooTypeName,
	// 		Kind:               common.CompositeKindStructure,
	// 		importable:         false,
	// 		hasComputedMembers: true,
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
								X: dst.NewIdent("CompositeType"),
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
		goKeyValue("importable", goBoolLit(ty.importable)),
		goKeyValue("hasComputedMembers", goBoolLit(true)),
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: dst.NewIdent("CompositeType"),
			Elts: elements,
		},
	}
}

func typeAnnotationCallExpr(ty dst.Expr) *dst.CallExpr {
	return &dst.CallExpr{
		Fun: dst.NewIdent("NewTypeAnnotation"),
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
			Type: dst.NewIdent("TypeParameter"),
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
			Type: dst.NewIdent("EntitlementType"),
			Elts: []dst.Expr{
				goKeyValue("Identifier", goStringLit(name)),
			},
		},
	}
}

func entitlementMapTypeLiteral(name string, associations []*ast.EntitlementMapElement) dst.Expr {
	// &sema.EntitlementMapType{
	//	Identifier: "Foo",
	//	Relations: []EntitlementRelation{
	//		{
	//			Input: BarType,
	//			Output: BazType,
	//		},
	//	}
	// }

	relationExprs := make([]dst.Expr, 0, len(associations))

	for _, association := range associations {
		relationExpr := &dst.CompositeLit{
			Type: dst.NewIdent("EntitlementRelation"),
			Elts: []dst.Expr{
				goKeyValue("Input", typeExpr(association.Input, nil)),
				goKeyValue("Output", typeExpr(association.Output, nil)),
			},
		}

		relationExpr.Decorations().Before = dst.NewLine
		relationExpr.Decorations().After = dst.NewLine

		relationExprs = append(relationExprs, relationExpr)
	}

	relationsExpr := &dst.CompositeLit{
		Type: &dst.ArrayType{
			Elt: dst.NewIdent("EntitlementRelation"),
		},
		Elts: relationExprs,
	}

	return &dst.UnaryExpr{
		Op: token.AND,
		X: &dst.CompositeLit{
			Type: dst.NewIdent("EntitlementMapType"),
			Elts: []dst.Expr{
				goKeyValue("Identifier", goStringLit(name)),
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

func gen(inPath string, outFile *os.File) {
	program := parseCadenceFile(inPath)

	var gen generator

	for _, declaration := range program.Declarations() {
		_ = ast.AcceptDeclaration[struct{}](declaration, &gen)
	}

	gen.generateTypeInit(program)

	writeGoFile(inPath, outFile, gen.decls)
}

func writeGoFile(inPath string, outFile *os.File, decls []dst.Decl) {
	err := parsedHeaderTemplate.Execute(outFile, inPath)
	if err != nil {
		panic(err)
	}

	restorer := decorator.NewRestorerWithImports("sema", guess.RestorerResolver{})

	err = restorer.Fprint(
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
