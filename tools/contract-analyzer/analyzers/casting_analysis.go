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

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/tools/analysis"
)

type CheckCastVisitor struct {
	exprInferredType sema.Type
	targetType       sema.Type
}

var _ ast.ExpressionVisitor = &CheckCastVisitor{}

func (d *CheckCastVisitor) IsRedundantCast(expr ast.Expression, exprInferredType, targetType sema.Type) bool {
	prevInferredType := d.exprInferredType
	prevTargetType := d.targetType

	defer func() {
		d.exprInferredType = prevInferredType
		d.targetType = prevTargetType
	}()

	d.exprInferredType = exprInferredType
	d.targetType = targetType

	result := expr.AcceptExp(d)
	return result.(bool)
}

func (d *CheckCastVisitor) VisitBoolExpression(_ *ast.BoolExpression) ast.Repr {
	return d.isTypeRedundant(sema.BoolType, d.targetType)
}

func (d *CheckCastVisitor) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	return d.isTypeRedundant(sema.NilType, d.targetType)
}

func (d *CheckCastVisitor) VisitIntegerExpression(_ *ast.IntegerExpression) ast.Repr {
	// For integer expressions, default inferred type is `Int`.
	// So, if the target type is not `Int`, then the cast is not redundant.
	return d.isTypeRedundant(sema.IntType, d.targetType)
}

func (d *CheckCastVisitor) VisitFixedPointExpression(expr *ast.FixedPointExpression) ast.Repr {
	if expr.Negative {
		// Default inferred type for fixed-point expressions with sign is `Fix64Type`.
		return d.isTypeRedundant(sema.Fix64Type, d.targetType)
	}

	// Default inferred type for fixed-point expressions without sign is `UFix64Type`.
	return d.isTypeRedundant(sema.UFix64Type, d.targetType)
}

func (d *CheckCastVisitor) VisitArrayExpression(expr *ast.ArrayExpression) ast.Repr {
	// If the target type is `ConstantSizedType`, then it is not redundant.
	// Because array literals are always inferred to be `VariableSizedType`,
	// unless specified.
	targetArrayType, ok := d.targetType.(*sema.VariableSizedType)
	if !ok {
		return false
	}

	inferredArrayType, ok := d.exprInferredType.(sema.ArrayType)
	if !ok {
		return false
	}

	for _, element := range expr.Values {
		// If at-least one element uses the target-type to infer the expression type,
		// then the casting is not redundant.
		if !d.IsRedundantCast(
			element,
			inferredArrayType.ElementType(false),
			targetArrayType.ElementType(false),
		) {
			return false
		}
	}

	return true
}

func (d *CheckCastVisitor) VisitDictionaryExpression(expr *ast.DictionaryExpression) ast.Repr {
	targetDictionaryType, ok := d.targetType.(*sema.DictionaryType)
	if !ok {
		return false
	}

	inferredDictionaryType, ok := d.exprInferredType.(*sema.DictionaryType)
	if !ok {
		return false
	}

	for _, entry := range expr.Entries {
		// If at-least one key or value uses the target-type to infer the expression type,
		// then the casting is not redundant.
		if !d.IsRedundantCast(
			entry.Key,
			inferredDictionaryType.KeyType,
			targetDictionaryType.KeyType,
		) {
			return false
		}

		if !d.IsRedundantCast(
			entry.Value,
			inferredDictionaryType.ValueType,
			targetDictionaryType.ValueType,
		) {
			return false
		}
	}

	return true
}

func (d *CheckCastVisitor) VisitIdentifierExpression(_ *ast.IdentifierExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitInvocationExpression(_ *ast.InvocationExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitMemberExpression(_ *ast.MemberExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitIndexExpression(_ *ast.IndexExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitConditionalExpression(conditionalExpr *ast.ConditionalExpression) ast.Repr {
	return d.IsRedundantCast(conditionalExpr.Then, d.exprInferredType, d.targetType) &&
		d.IsRedundantCast(conditionalExpr.Else, d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitUnaryExpression(_ *ast.UnaryExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitBinaryExpression(_ *ast.BinaryExpression) ast.Repr {
	// Binary expressions are not straight-forward to check.
	// Hence skip checking redundant casts for now.
	return false
}

func (d *CheckCastVisitor) VisitFunctionExpression(_ *ast.FunctionExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitStringExpression(_ *ast.StringExpression) ast.Repr {
	return d.isTypeRedundant(sema.StringType, d.targetType)
}

func (d *CheckCastVisitor) VisitCastingExpression(_ *ast.CastingExpression) ast.Repr {
	// This is already covered under Case-I: where expected type is same as casted type.
	// So skip checking it here to avid duplicate errors.
	return false
}

func (d *CheckCastVisitor) VisitCreateExpression(_ *ast.CreateExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitDestroyExpression(_ *ast.DestroyExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitReferenceExpression(_ *ast.ReferenceExpression) ast.Repr {
	return false
}

func (d *CheckCastVisitor) VisitForceExpression(_ *ast.ForceExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) VisitPathExpression(_ *ast.PathExpression) ast.Repr {
	return d.isTypeRedundant(d.exprInferredType, d.targetType)
}

func (d *CheckCastVisitor) isTypeRedundant(exprType, targetType sema.Type) bool {
	// If there is no expected type (e.g: var-decl with no type annotation),
	// then the simple-cast might be used as a way of marking the type of the variable.
	// Therefore, it is ok for the target type to be a super-type.
	// But being the exact type as expression's type is redundant.
	// e.g:
	//   var x: Int8 = 5
	//   var y = x as Int8     // <-- not ok: `y` will be of type `Int8` with/without cast
	//   var y = x as Integer  // <-- ok	: `y` will be of type `Integer`
	return exprType != nil &&
		exprType.Equal(targetType)
}

// isRedundantCast checks whether a simple cast is redundant.
// Checks for two cases:
//    - Case I: Contextually expected type is same as the casted type (target type).
//    - Case II: Expression is self typed, and is same as the casted type (target type).
func isRedundantCast(expr ast.Expression, exprInferredType, targetType, expectedType sema.Type) bool {
	if expectedType != nil &&
		!expectedType.IsInvalidType() &&
		expectedType.Equal(targetType) {
		return true
	}

	checkCastVisitor := &CheckCastVisitor{}

	return checkCastVisitor.IsRedundantCast(expr, exprInferredType, targetType)
}

var RedundantCastAnalyzer = (func() *analysis.Analyzer {

	elementFilter := []ast.Element{
		(*ast.CastingExpression)(nil),
	}

	return &analysis.Analyzer{
		Description: "Detects unnecessary cast expressions",
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

					castingExpression, ok := element.(*ast.CastingExpression)
					if !ok {
						return
					}

					redundantType, ok := elaboration.StaticCastTypes[castingExpression]
					if ok && isRedundantCast(castingExpression.Expression,
						redundantType.ExprActualType,
						redundantType.TargetType,
						redundantType.ExpectedType,
					) {
						report(
							analysis.Diagnostic{
								Location: location,
								Range:    ast.NewRangeFromPositioned(nil, castingExpression.TypeAnnotation),
								Category: "lint",
								Message:  fmt.Sprintf("cast to `%s` is redundant", redundantType.TargetType),
							},
						)
						return
					}

					alwaysSucceedingTypes, ok := elaboration.RuntimeCastTypes[castingExpression]
					if ok && sema.IsSubType(alwaysSucceedingTypes.Left, alwaysSucceedingTypes.Right) {
						switch castingExpression.Operation {
						case ast.OperationFailableCast:
							report(
								analysis.Diagnostic{
									Location: location,
									Range:    ast.NewRangeFromPositioned(nil, castingExpression),
									Category: "lint",
									Message: fmt.Sprintf("failable cast ('%s') from `%s` to `%s` always succeeds",
										ast.OperationFailableCast.Symbol(),
										alwaysSucceedingTypes.Left,
										alwaysSucceedingTypes.Right),
								},
							)
						case ast.OperationForceCast:
							report(
								analysis.Diagnostic{
									Location: location,
									Range:    ast.NewRangeFromPositioned(nil, castingExpression),
									Category: "lint",
									Message: fmt.Sprintf("force cast ('%s') from `%s` to `%s` always succeeds",
										ast.OperationForceCast.Symbol(),
										alwaysSucceedingTypes.Left,
										alwaysSucceedingTypes.Right),
								},
							)
						default:
							panic(errors.NewUnreachableError())
						}
					}
				},
			)

			return nil
		},
	}
})()

func init() {
	registerAnalyzer(
		"cast-analysis",
		RedundantCastAnalyzer,
	)
}
