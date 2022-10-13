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

import (
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/errors"
)

func (checker *Checker) VisitVariableDeclaration(declaration *ast.VariableDeclaration) (_ struct{}) {
	declarationType := checker.visitVariableDeclarationValues(declaration, false)
	checker.declareVariableDeclaration(declaration, declarationType)

	return
}

func (checker *Checker) visitVariableDeclarationValues(declaration *ast.VariableDeclaration, isOptionalBinding bool) Type {

	checker.checkDeclarationAccessModifier(
		declaration.Access,
		declaration.DeclarationKind(),
		declaration.StartPos,
		declaration.IsConstant,
	)

	// Determine the type of the initial value of the variable declaration
	// and save it in the elaboration

	var declarationType Type
	var expectedValueType Type

	if declaration.TypeAnnotation != nil {
		typeAnnotation := checker.ConvertTypeAnnotation(declaration.TypeAnnotation)
		checker.checkTypeAnnotation(typeAnnotation, declaration.TypeAnnotation)
		declarationType = typeAnnotation.Type

		// If the variable declaration is an optional binding (`if let`),
		// then the value is expected to be an optional of the declaration's type.
		if isOptionalBinding {
			expectedValueType = &OptionalType{
				Type: declarationType,
			}
		} else {
			expectedValueType = declarationType
		}
	}

	valueType := checker.VisitExpression(declaration.Value, expectedValueType)

	if isOptionalBinding {
		optionalType, isOptional := valueType.(*OptionalType)

		if !isOptional || optionalType.Equal(declarationType) {
			if !valueType.IsInvalidType() {
				checker.report(
					&TypeMismatchError{
						ExpectedType: &OptionalType{},
						ActualType:   valueType,
						Range:        ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Value),
					},
				)
			}
		} else if declarationType == nil {
			declarationType = optionalType.Type
		}
	}

	if declarationType == nil {
		declarationType = valueType
	}

	checker.checkTransfer(declaration.Transfer, declarationType)

	// The variable declaration might have a second transfer and second expression.
	//
	// In that case the declaration transfers not only the value of the first expression
	// to the identifier (new variable), but the declaration also transfers the value
	// of the second expression to the first expression (which must be a target expression).
	//
	// This is only valid for resources, i.e. the declaration type, first value type,
	// and the second value type must be resource types, and all transfers must be moves.

	var secondValueType Type

	if declaration.SecondTransfer == nil {
		if declaration.SecondValue != nil {
			panic(errors.NewUnreachableError())
		}

		checker.checkVariableMove(declaration.Value)

		// If only one value expression is provided, it is invalidated (if it has a resource type)

		checker.recordResourceInvalidation(
			declaration.Value,
			declarationType,
			ResourceInvalidationKindMoveDefinite,
		)
	} else {

		// The first expression must be a target expression (e.g. identifier expression,
		// indexing expression, or member access expression)

		if !IsValidAssignmentTargetExpression(declaration.Value) {
			checker.report(
				&InvalidAssignmentTargetError{
					Range: ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Value),
				},
			)
		} else {
			// The assignment is valid (i.e. to a target expression)

			// NOTE: Check that the *first* value type is a resource type –
			// The assignment check below will also ensure that the second value type
			// is a resource.
			//
			// The first value is checked instead of the second value,
			// so that second value types that are standalone not considered resource typed
			// are still admitted if they are type compatible (e.g. `nil`).

			valueIsResource := valueType != nil && valueType.IsResourceType()

			if valueType != nil &&
				!valueType.IsInvalidType() &&
				!valueIsResource {

				checker.report(
					&NonResourceTypeError{
						ActualType: valueType,
						Range:      ast.NewRangeFromPositioned(checker.memoryGauge, declaration.Value),
					},
				)
			}

			// Check the assignment of the second value to the first expression

			// The check of the assignment of the second value to the first also:
			// - Invalidates the second resource
			// - Checks the second transfer
			// - Checks the second value type is a subtype of value type
			// etc.

			// NOTE: already performs resource invalidation

			_, secondValueType = checker.checkAssignment(
				declaration,
				declaration.Value,
				declaration.SecondValue,
				declaration.SecondTransfer,
				true,
			)

			if valueIsResource {
				checker.elaborateNestedResourceMoveExpression(declaration.Value)
			}
		}
	}

	checker.Elaboration.VariableDeclarationTypes[declaration] =
		VariableDeclarationTypes{
			TargetType:      declarationType,
			ValueType:       valueType,
			SecondValueType: secondValueType,
		}

	return declarationType
}

func (checker *Checker) declareVariableDeclaration(declaration *ast.VariableDeclaration, declarationType Type) {
	// Finally, declare the variable in the current value activation

	identifier := declaration.Identifier.Identifier

	variable, err := checker.valueActivations.declare(variableDeclaration{
		identifier:               identifier,
		ty:                       declarationType,
		docString:                declaration.DocString,
		access:                   declaration.Access,
		kind:                     declaration.DeclarationKind(),
		pos:                      declaration.Identifier.Pos,
		isConstant:               declaration.IsConstant,
		argumentLabels:           nil,
		allowOuterScopeShadowing: true,
	})
	checker.report(err)

	if checker.PositionInfo != nil {
		checker.recordVariableDeclarationOccurrence(identifier, variable)
		checker.recordVariableDeclarationRange(declaration, identifier, declarationType)
	}

	checker.recordReference(identifier, declaration.Value)
}

func (checker *Checker) recordVariableDeclarationRange(
	declaration *ast.VariableDeclaration,
	identifier string,
	declarationType Type,
) {
	activation := checker.valueActivations.Current()
	activation.LeaveCallbacks = append(
		activation.LeaveCallbacks,
		func(getEndPosition EndPositionGetter) {
			if getEndPosition == nil || checker.PositionInfo == nil {
				return
			}

			memoryGauge := checker.memoryGauge

			endPosition := getEndPosition(memoryGauge)

			checker.PositionInfo.recordVariableDeclarationRange(
				memoryGauge,
				declaration,
				endPosition,
				identifier,
				declarationType,
			)
		},
	)
}

func (checker *Checker) elaborateNestedResourceMoveExpression(expression ast.Expression) {
	switch expression.(type) {
	case *ast.IndexExpression, *ast.MemberExpression:
		checker.Elaboration.IsNestedResourceMoveExpression[expression] = struct{}{}
	}
}

func (checker *Checker) recordReferenceCreation(target, expr ast.Expression) {
	switch target := target.(type) {
	case *ast.IdentifierExpression:
		checker.recordReference(target.Identifier.Identifier, expr)
	default:
		// Currently it's not possible to track the references
		// assigned to member-expressions/index-expressions.
		return
	}
}

func (checker *Checker) recordReference(targetVarName string, expr ast.Expression) {
	referencedVar := checker.referencedVariable(expr)

	if referencedVar != nil &&
		referencedVar.Type.IsResourceType() {
		checker.references.Set(targetVarName, referencedVar)
	}
}

// referencedVariable return the referenced variable
func (checker *Checker) referencedVariable(expr ast.Expression) *Variable {
	refExpression := referenceExpression(expr)

	var variableRefExpr *ast.Identifier

	switch refExpression := refExpression.(type) {
	case *ast.ReferenceExpression:
		// If it is a reference expression, then find the "root variable".
		// As nested resources cannot be tracked, at least track the "root" if possible.
		// For example, for an expression `&a.b.c as &T`, the "root variable" is `a`.
		variableRefExpr = variableReferenceExpression(refExpression.Expression)
	case *ast.IdentifierExpression:
		variableRefExpr = &refExpression.Identifier
	default:
		return nil
	}

	if variableRefExpr == nil {
		return nil
	}

	referencedVariableName := variableRefExpr.Identifier

	for {
		// If the referenced variable is again a reference,
		// then find the variable of the root of the reference chain.
		// e.g.:
		//     ref1 = &v
		//     ref2 = &ref1
		//     ref2.field = 3
		//
		// Here, `ref2` refers to `ref1`, which refers to `v`.
		// So `ref2` is actually referring to `v`

		referencedRef := checker.references.Find(referencedVariableName)
		if referencedRef == nil {
			break
		}

		referencedVariableName = referencedRef.Identifier
	}

	return checker.valueActivations.Find(referencedVariableName)
}

// referenceExpression returns the expression that return a reference.
// There could be two types of expressions that can result in a reference:
//  1. Expressions that create a new reference.
//     (i.e: reference-expression)
//  2. Expressions that return an existing reference.
//     (i.e: identifier-expression/member-expression/index-expression having a reference type)
//
// However, it is currently not possible to track member-expressions and index-expressions.
// So this method either returns a reference-expression or an identifier-expression.
//
// The expression could also be hidden inside some other expression.
// e.g(1): `&v as &T` is a casting expression, but has a hidden reference expression.
// e.g(2): `(&v as &T?)!
func referenceExpression(expr ast.Expression) ast.Expression {
	switch expr := expr.(type) {
	case *ast.ReferenceExpression:
		return expr
	case *ast.ForceExpression:
		return referenceExpression(expr.Expression)
	case *ast.CastingExpression:
		return referenceExpression(expr.Expression)
	case *ast.BinaryExpression:
		if expr.Operation != ast.OperationNilCoalesce {
			return nil
		}
		return referenceExpression(expr.Left)
	case *ast.IdentifierExpression:
		return expr
	default:
		return nil
	}
}

// variableReferenceExpression returns the identifier expression
// of a var-ref/member-access/index-access expression.
func variableReferenceExpression(expr ast.Expression) *ast.Identifier {
	switch expr := expr.(type) {
	case *ast.IdentifierExpression:
		return &expr.Identifier
	case *ast.MemberExpression:
		return variableReferenceExpression(expr.Expression)
	case *ast.IndexExpression:
		return variableReferenceExpression(expr.TargetExpression)
	default:
		return nil
	}
}
