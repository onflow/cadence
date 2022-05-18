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

package sema

import "github.com/onflow/cadence/runtime/ast"

type CheckCastVisitor struct {
	exprInferredType Type
	targetType       Type
}

var _ ast.ExpressionVisitor = &CheckCastVisitor{}

func (d *CheckCastVisitor) IsRedundantCast(expr ast.Expression, exprInferredType, targetType Type) bool {
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
	return d.isTypeRedundant(BoolType, d.targetType)
}

func (d *CheckCastVisitor) VisitNilExpression(_ *ast.NilExpression) ast.Repr {
	return d.isTypeRedundant(NilType, d.targetType)
}

func (d *CheckCastVisitor) VisitIntegerExpression(_ *ast.IntegerExpression) ast.Repr {
	// For integer expressions, default inferred type is `Int`.
	// So, if the target type is not `Int`, then the cast is not redundant.
	return d.isTypeRedundant(IntType, d.targetType)
}

func (d *CheckCastVisitor) VisitFixedPointExpression(expr *ast.FixedPointExpression) ast.Repr {
	if expr.Negative {
		// Default inferred type for fixed-point expressions with sign is `Fix64Type`.
		return d.isTypeRedundant(Fix64Type, d.targetType)
	}

	// Default inferred type for fixed-point expressions without sign is `UFix64Type`.
	return d.isTypeRedundant(UFix64Type, d.targetType)
}

func (d *CheckCastVisitor) VisitArrayExpression(expr *ast.ArrayExpression) ast.Repr {
	// If the target type is `ConstantSizedType`, then it is not redundant.
	// Because array literals are always inferred to be `VariableSizedType`,
	// unless specified.
	targetArrayType, ok := d.targetType.(*VariableSizedType)
	if !ok {
		return false
	}

	inferredArrayType, ok := d.exprInferredType.(ArrayType)
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
	targetDictionaryType, ok := d.targetType.(*DictionaryType)
	if !ok {
		return false
	}

	inferredDictionaryType, ok := d.exprInferredType.(*DictionaryType)
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
	return d.isTypeRedundant(StringType, d.targetType)
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

func (d *CheckCastVisitor) isTypeRedundant(exprType, targetType Type) bool {
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
