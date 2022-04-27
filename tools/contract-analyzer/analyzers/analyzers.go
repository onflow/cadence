/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package analyzers

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/tools/analysis"
)

// Reference to optional analyzer

var ReferenceToOptionalAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.ReferenceExpression)(nil),
	}

	return &analysis.Analyzer{
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			location := pass.Program.Location
			elaboration := pass.Program.Elaboration
			report := pass.Report

			inspector.Preorder(
				elementFilter,
				func(element ast.Element) {
					referenceExpression, ok := element.(*ast.ReferenceExpression)
					if !ok {
						return
					}

					indexExpression, ok := referenceExpression.Expression.(*ast.IndexExpression)
					if !ok {
						return
					}

					indexedType := elaboration.IndexExpressionIndexedTypes[indexExpression]
					resultType := indexedType.ElementType(false)
					_, ok = resultType.(*sema.OptionalType)
					if !ok {
						return
					}

					replacement := &ast.OptionalType{
						Type: referenceExpression.Type,
					}

					report(
						analysis.Diagnostic{
							Location:         location,
							Range:            ast.NewRangeFromPositioned(referenceExpression.Type),
							Category:         "update required",
							Message:          "reference to optional will return optional reference",
							SecondaryMessage: fmt.Sprintf("replace with '%s'", replacement.String()),
						},
					)
				},
			)

			return nil
		},
	}
})()

func init() {

	registerAnalyzer(
		"reference-to-optional",
		ReferenceToOptionalAnalyzer,
	)
}

// Deprecated key functions analyzer

var DeprecatedKeyFunctionsAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.InvocationExpression)(nil),
	}

	return &analysis.Analyzer{
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			location := pass.Program.Location
			elaboration := pass.Program.Elaboration
			report := pass.Report

			inspector.Preorder(
				elementFilter,
				func(element ast.Element) {
					invocationExpression, ok := element.(*ast.InvocationExpression)
					if !ok {
						return
					}

					memberExpression, ok := invocationExpression.InvokedExpression.(*ast.MemberExpression)
					if !ok {
						return
					}

					memberInfo := elaboration.MemberExpressionMemberInfos[memberExpression]
					member := memberInfo.Member
					if member == nil {
						return
					}

					if member.ContainerType != sema.AuthAccountType {
						return
					}

					var replacement string
					functionName := member.Identifier.Identifier
					switch functionName {
					case sema.AuthAccountAddPublicKeyField:
						replacement = "keys.add"
					case sema.AuthAccountRemovePublicKeyField:
						replacement = "keys.revoke"
					default:
						return
					}

					report(
						analysis.Diagnostic{
							Location: location,
							Range:    ast.NewRangeFromPositioned(memberExpression.Identifier),
							Category: "update recommended",
							Message: fmt.Sprintf(
								"deprecated function '%s' will get removed",
								functionName,
							),
							SecondaryMessage: fmt.Sprintf(
								"replace with '%s'",
								replacement,
							),
						},
					)
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"deprecated-key-functions",
		DeprecatedKeyFunctionsAnalyzer,
	)
}

// Number supertype arithmetic operations analyzer

var NumberSupertypeBinaryOperationsAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.BinaryExpression)(nil),
	}

	numberSupertypes := map[sema.Type]struct{}{
		sema.NumberType:           {},
		sema.SignedNumberType:     {},
		sema.IntegerType:          {},
		sema.SignedIntegerType:    {},
		sema.FixedPointType:       {},
		sema.SignedFixedPointType: {},
	}

	return &analysis.Analyzer{
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			location := pass.Program.Location
			elaboration := pass.Program.Elaboration
			report := pass.Report

			inspector.Preorder(
				elementFilter,
				func(element ast.Element) {
					binaryExpression, ok := element.(*ast.BinaryExpression)
					if !ok {
						return
					}

					leftType := elaboration.BinaryExpressionLeftTypes[binaryExpression]

					if _, ok := numberSupertypes[leftType]; !ok {
						return
					}

					report(
						analysis.Diagnostic{
							Location: location,
							Range:    ast.NewRangeFromPositioned(element),
							Category: "update required",
							Message: fmt.Sprintf(
								"%s operations on number supertypes will get removed",
								binaryExpression.Operation.Category(),
							),
						},
					)
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"number-supertype-binary-operations",
		NumberSupertypeBinaryOperationsAnalyzer,
	)
}

// Parameter list missing commas analyzer

var ParameterListMissingCommasAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.FunctionDeclaration)(nil),
		(*ast.FunctionExpression)(nil),
	}

	return &analysis.Analyzer{
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			location := pass.Program.Location
			code := pass.Program.Code
			report := pass.Report

			inspector.Preorder(
				elementFilter,
				func(element ast.Element) {

					var parameterList *ast.ParameterList
					switch element := element.(type) {
					case *ast.FunctionExpression:
						parameterList = element.ParameterList
					case *ast.FunctionDeclaration:
						parameterList = element.ParameterList
					default:
						return
					}

					parameters := parameterList.Parameters
					for i, parameter := range parameters {
						if i == 0 {
							continue
						}

						startOffset := parameter.StartPosition().Offset

						previousParameter := parameters[i-1]
						previousEndPos := previousParameter.EndPosition()
						previousEndOffset := previousEndPos.Offset

						if strings.ContainsRune(code[previousEndOffset:startOffset], ',') {
							continue
						}

						diagnosticPos := previousEndPos.Shifted(1)

						report(
							analysis.Diagnostic{
								Location: location,
								Range: ast.Range{
									StartPos: diagnosticPos,
									EndPos:   diagnosticPos,
								},
								Category:         "update required",
								Message:          "missing comma between parameters",
								SecondaryMessage: "insert missing comma here",
							},
						)
					}

				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"parameter-list-missing-commas",
		ParameterListMissingCommasAnalyzer,
	)
}

// Supertype inference analyzer

var SupertypeInferenceAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.ArrayExpression)(nil),
		(*ast.DictionaryExpression)(nil),
		(*ast.ConditionalExpression)(nil),
	}

	return &analysis.Analyzer{
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			location := pass.Program.Location
			elaboration := pass.Program.Elaboration
			report := pass.Report

			inspector.Preorder(
				elementFilter,
				func(element ast.Element) {

					type typeTuple struct {
						first, second sema.Type
					}
					var typeTuples []typeTuple

					var kind string

					switch element := element.(type) {
					case *ast.ArrayExpression:
						kind = "arrays"

						argumentTypes := elaboration.ArrayExpressionArgumentTypes[element]
						if len(argumentTypes) < 2 {
							return
						}
						typeTuples = append(
							typeTuples,
							typeTuple{
								first:  argumentTypes[0],
								second: argumentTypes[1],
							},
						)

					case *ast.DictionaryExpression:
						kind = "dictionaries"

						entryTypes := elaboration.DictionaryExpressionEntryTypes[element]
						if len(entryTypes) < 2 {
							return
						}
						typeTuples = append(
							typeTuples,
							typeTuple{
								first:  entryTypes[0].KeyType,
								second: entryTypes[1].KeyType,
							},
							typeTuple{
								first:  entryTypes[0].ValueType,
								second: entryTypes[1].ValueType,
							},
						)

					case *ast.ConditionalExpression:
						kind = "conditionals / ternary operations"

						typeTuples = append(
							typeTuples,
							typeTuple{
								first:  elaboration.ConditionalExpressionThenType[element],
								second: elaboration.ConditionalExpressionElseType[element],
							},
						)

					default:
						return
					}

					for _, typeTuple := range typeTuples {
						if typeTuple.first.Equal(typeTuple.second) {
							continue
						}

						report(
							analysis.Diagnostic{
								Location:         location,
								Range:            ast.NewRangeFromPositioned(element),
								Category:         "check required",
								Message:          fmt.Sprintf("type inference for %s will change", kind),
								SecondaryMessage: "ensure the newly inferred type is correct",
							},
						)

						// Only report one diagnostic for each expression
						return
					}

				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"supertype-inference",
		SupertypeInferenceAnalyzer,
	)
}

// External mutation analyzer

var ExternalMutationAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.AssignmentStatement)(nil),
		(*ast.MemberExpression)(nil),
	}

	outerCompositeTypes := func(
		elaboration *sema.Elaboration,
		stack []ast.Element,
	) []*sema.CompositeType {

		compositeTypes := make([]*sema.CompositeType, 0, len(stack))

		for _, element := range stack {
			compositeDeclaration, ok := element.(*ast.CompositeDeclaration)
			if !ok {
				continue
			}

			compositeType := elaboration.CompositeDeclarationTypes[compositeDeclaration]
			compositeTypes = append(compositeTypes, compositeType)
		}

		return compositeTypes
	}

	memberExpressionMutatedElement := func(
		elaboration *sema.Elaboration,
		memberExpression *ast.MemberExpression,
	) ast.Element {

		memberInfo := elaboration.MemberExpressionMemberInfos[memberExpression]
		member := memberInfo.Member
		if member == nil {
			return nil
		}

		switch memberInfo.AccessedType.(type) {
		case *sema.DictionaryType:
			switch member.Identifier.Identifier {
			case "insert", "remove":
				break
			default:
				return nil
			}

		case sema.ArrayType:
			switch member.Identifier.Identifier {
			case "append",
				"appendAll",
				"insert",
				"remove",
				"removeFirst",
				"removeLast":
				break
			default:
				return nil
			}

		default:
			return nil
		}

		return memberExpression.Expression
	}

	indexingAssignmentStatementMutatedElement := func(
		elaboration *sema.Elaboration,
		assignmentStatement *ast.AssignmentStatement,
	) ast.Element {
		indexExpression, ok := assignmentStatement.Target.(*ast.IndexExpression)
		if !ok {
			return nil
		}

		return indexExpression.TargetExpression
	}

	isExternallyMutated := func(
		elaboration *sema.Elaboration,
		element ast.Element,
		stack []ast.Element,
	) bool {
		innerMemberExpression, ok := element.(*ast.MemberExpression)
		if !ok {
			return false
		}

		innerMemberInfo := elaboration.MemberExpressionMemberInfos[innerMemberExpression]
		innerMember := innerMemberInfo.Member
		if innerMember == nil {
			return false
		}

		if innerMember.Access == ast.AccessPublicSettable {
			return false
		}

		compositeTypes := outerCompositeTypes(elaboration, stack)

		for _, compositeType := range compositeTypes {
			if innerMemberInfo.AccessedType == compositeType {
				return false
			}
		}

		return true
	}

	analyze := func(pass *analysis.Pass, element ast.Element, stack []ast.Element) {

		location := pass.Program.Location
		elaboration := pass.Program.Elaboration
		report := pass.Report

		var mutatedElement ast.Element
		switch element := element.(type) {
		case *ast.AssignmentStatement:
			mutatedElement = indexingAssignmentStatementMutatedElement(elaboration, element)

		case *ast.MemberExpression:
			mutatedElement = memberExpressionMutatedElement(elaboration, element)

		default:
			return
		}

		if !isExternallyMutated(elaboration, mutatedElement, stack) {
			return
		}

		report(
			analysis.Diagnostic{
				Location: location,
				Range:    ast.NewRangeFromPositioned(mutatedElement),
				Category: "update required",
				Message:  "external mutation of non-settable public container-typed field will get disallowed",
				SecondaryMessage: fmt.Sprintf(
					"add setter function for field, or change field access to %s",
					ast.AccessPublicSettable.Keyword(),
				),
			},
		)
	}

	return &analysis.Analyzer{
		Requires: []*analysis.Analyzer{
			analysis.InspectorAnalyzer,
		},
		Run: func(pass *analysis.Pass) interface{} {
			inspector := pass.ResultOf[analysis.InspectorAnalyzer].(*ast.Inspector)

			inspector.WithStack(
				elementFilter,
				func(element ast.Element, push bool, stack []ast.Element) (proceed bool) {
					if !push {
						return true
					}

					analyze(pass, element, stack)

					return true
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"external-mutation",
		ExternalMutationAnalyzer,
	)
}
