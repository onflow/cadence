package sema

import (
	"github.com/onflow/cadence/runtime/ast"
)

func (checker *Checker) VisitForceExpression(expression *ast.ForceExpression) ast.Repr {

	valueType := expression.Expression.Accept(checker).(Type)

	if valueType.IsInvalidType() {
		return valueType
	}

	checker.recordResourceInvalidation(
		expression.Expression,
		valueType,
		ResourceInvalidationKindMove,
	)

	optionalType, ok := valueType.(*OptionalType)
	if !ok {
		checker.report(
			&NonOptionalForceError{
				Type:  valueType,
				Range: ast.NewRangeFromPositioned(expression.Expression),
			},
		)

		return valueType
	}

	return optionalType.Type
}
