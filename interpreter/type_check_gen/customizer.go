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
	"go/token"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

const (
	typedSuperTypeVarName              = "typedSuperType"
	typedSemaSuperTypeVarName          = "typedSemaSuperType"
	typedSubTypeVarName                = "typedSubType"
	typedSemaSubTypeVarName            = "typedSemaSubType"
	typeConverter                      = "typeConverter"
	staticToSemaTypeConversionFuncName = "SemaTypeFromStaticType"
)

func Update(decls []dst.Decl) []dst.Decl {
	for i, decl := range decls {
		for _, updater := range updaters {
			decls[i] = updater(decl)
		}
	}

	return decls
}

var updaters = []CodeUpdater{
	IntersectionTypeCheckUpdater,
	FunctionParametersCheckUpdater,
}

type CodeUpdater func(decl dst.Decl) dst.Decl

// IntersectionTypeCheckUpdater Updates the intersection type's subtype checking
// to use `sema.Type`s, since `StaticType`s doesn't preserve conformance info.
func IntersectionTypeCheckUpdater(decl dst.Decl) dst.Decl {
	var intersectionTypeRuleNode, nestedCaseClause dst.Node
	return dstutil.Apply(
		decl,

		// Pre-order traversal: called before visiting children
		func(cursor *dstutil.Cursor) bool {
			currentNode := cursor.Node()

			switch currentNode := currentNode.(type) {
			case *dst.CaseClause:
				caseExpr := currentNode.List[0]
				starExpr, ok := caseExpr.(*dst.StarExpr)
				if !ok {
					break
				}
				identifier, ok := starExpr.X.(*dst.Ident)
				if !ok {
					break
				}

				// This is the case-clause for `*InterfaceStaticType`, in the outer type-switch.
				if intersectionTypeRuleNode == nil && identifier.Name == "InterfaceStaticType" {
					intersectionTypeRuleNode = currentNode
					return true
				}

				// This is a nested case-clause inside `intersectionTypeRuleNode`.
				if intersectionTypeRuleNode != nil {
					nestedCaseClause = currentNode
				}

			case *dst.Ident:
				if nestedCaseClause != nil {
					switch currentNode.Name {
					case typedSuperTypeVarName:
						cursor.Replace(dst.NewIdent(typedSemaSuperTypeVarName))
					case typedSubTypeVarName:
						cursor.Replace(dst.NewIdent(typedSemaSubTypeVarName))
					}
				}
			}

			// Return true to continue visiting children
			return true
		},

		// Post-order traversal: called after visiting children
		func(cursor *dstutil.Cursor) bool {
			node := cursor.Node()
			if node == nil {
				return true
			}

			// Add the new variables after visiting the clause (rather than before visiting),
			// so that renaming the variables won't affect this newly added one.
			switch node {
			case intersectionTypeRuleNode:
				intersectionTypeRuleNode = nil

			case nestedCaseClause:
				caseClause := node.(*dst.CaseClause)
				caseExpr := caseClause.List[0]
				starExpr := caseExpr.(*dst.StarExpr)
				identifier := starExpr.X.(*dst.Ident)

				semaTypeName := strings.ReplaceAll(identifier.Name, "StaticType", "Type")

				superTypeSemaConversion := &dst.AssignStmt{
					Lhs: []dst.Expr{
						dst.NewIdent(typedSemaSuperTypeVarName),
					},
					Tok: token.DEFINE,
					Rhs: []dst.Expr{
						&dst.TypeAssertExpr{
							X: &dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   dst.NewIdent(typeConverter),
									Sel: dst.NewIdent(staticToSemaTypeConversionFuncName),
								},
								Args: []dst.Expr{
									dst.NewIdent(typedSuperTypeVarName),
								},
							},
							Type: &dst.StarExpr{
								X: &dst.Ident{
									Name: "InterfaceType",
									Path: semaPkgPath,
								},
							},
						},
					},
				}

				subtypeSemaConversion := &dst.AssignStmt{
					Lhs: []dst.Expr{
						dst.NewIdent(typedSemaSubTypeVarName),
					},
					Tok: token.DEFINE,
					Rhs: []dst.Expr{
						&dst.TypeAssertExpr{
							X: &dst.CallExpr{
								Fun: &dst.SelectorExpr{
									X:   dst.NewIdent(typeConverter),
									Sel: dst.NewIdent(staticToSemaTypeConversionFuncName),
								},
								Args: []dst.Expr{
									dst.NewIdent(typedSubTypeVarName),
								},
							},
							Type: &dst.StarExpr{
								X: &dst.Ident{
									Name: semaTypeName,
									Path: semaPkgPath,
								},
							},
						},
					},
				}

				stmts := []dst.Stmt{
					superTypeSemaConversion,
					subtypeSemaConversion,
				}

				stmts = append(stmts, caseClause.Body...)
				caseClause.Body = stmts

				nestedCaseClause = nil
			}

			// Return true to continue
			return true
		},
	).(dst.Decl)
}

// FunctionParametersCheckUpdater updates the function parameter check to
// use the `IsSubType` function from the `sema` package.
func FunctionParametersCheckUpdater(decl dst.Decl) dst.Decl {
	var functionTypeRuleNode dst.Node
	var isFunctionParamsLoop bool

	return dstutil.Apply(
		decl,

		// Pre-order traversal: called before visiting children
		func(cursor *dstutil.Cursor) bool {
			currentNode := cursor.Node()

			switch currentNode := currentNode.(type) {
			case *dst.CaseClause:
				caseExpr := currentNode.List[0]
				identifier, ok := caseExpr.(*dst.Ident)
				if !ok {
					break
				}

				// This is the case-clause for `FunctionStaticType`, in the outer type-switch.
				if functionTypeRuleNode == nil && identifier.Name == "FunctionStaticType" {
					functionTypeRuleNode = currentNode
				}

			case *dst.RangeStmt:
				if functionTypeRuleNode != nil {
					identifier, ok := currentNode.X.(*dst.Ident)
					if ok && identifier.Name == "typedSubTypeParameters" {
						isFunctionParamsLoop = true
					}
				}

			case *dst.CallExpr:
				if isFunctionParamsLoop {
					identifier, ok := currentNode.Fun.(*dst.Ident)
					if ok && identifier.Name == "IsSubType" {
						// Update the package of the function.
						identifier.Path = semaPkgPath
						// Drop the "typeConverter" argument, since `sema.IsSubType` method don't need it.
						currentNode.Args = currentNode.Args[1:]
					}
				}
			}

			// Return true to continue visiting children
			return true
		},
		nil,
	).(dst.Decl)
}
