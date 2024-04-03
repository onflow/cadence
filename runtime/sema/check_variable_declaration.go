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

	checker.checkDeclarationAccessModifier(
		checker.accessFromAstAccess(declaration.Access),
		declaration.DeclarationKind(),
		valueType,
		nil,
		declaration.StartPos,
		declaration.IsConstant,
	)

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

	// The value transfer need to be marked as a resource move (when applicable),
	// even if there is no second value, because in destructors,
	// nested resource moves are allowed.
	valueIsResource := valueType != nil && valueType.IsResourceType()
	if valueIsResource {
		checker.elaborateNestedResourceMoveExpression(declaration.Value)
	}

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

			// Given a variable declaration with a second value transfer of the form
			// 				`let x <- e1 <- e2`
			//
			// the interpreter will evaluate this in the following order:
			//  - evaluating `e1` to some value `v1` (which is necessarily either a variable, an index expression, or a member expression)
			//  - moving the value stored in `v1` into the newly declared variable `x`
			//  - evaluating `e2` to some value `v2`
			//  - evaluate `e1` again to some value `v1'` (as `e1` may produce a different value after `e2`'s execution)
			//  - lastly moving the value `v2` into the "variable" denoted by `v1'`
			//
			// This means that while at the end of the execution of the entire statement,
			// `v1` is still a valid resource (having had the result of `e2` moved into it),
			// specifically during the execution of `e2` it is invalid.
			// In order to reflect this, we temporarily invalidate `v1` before evaluating `e2` and checking the assignment,
			// and then re-validate `v1` afterwards.
			// We also re-check `e1` again as well, since it is evaluated a second time by the interpreter.
			// Note that while typechecking the same expression twice is not safe in general,
			// in this case it is permissible for a specific reason:
			// `e1` is necessarily an identifier, index or member expression, which may not have side effects

			recordedResourceInvalidation := checker.recordResourceInvalidation(
				declaration.Value,
				declarationType,
				ResourceInvalidationKindMoveTemporary,
			)

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

			if recordedResourceInvalidation != nil {
				checker.resources.RemoveTemporaryMoveInvalidation(recordedResourceInvalidation.resource, recordedResourceInvalidation.invalidation)
			}
			checker.VisitExpression(declaration.Value, expectedValueType)
		}
	}

	checker.Elaboration.SetVariableDeclarationTypes(
		declaration,
		VariableDeclarationTypes{
			TargetType:      declarationType,
			ValueType:       valueType,
			SecondValueType: secondValueType,
		},
	)

	return declarationType
}

func (checker *Checker) declareVariableDeclaration(declaration *ast.VariableDeclaration, declarationType Type) {
	// Finally, declare the variable in the current value activation

	identifier := declaration.Identifier.Identifier

	variable, err := checker.valueActivations.declare(variableDeclaration{
		identifier:               identifier,
		ty:                       declarationType,
		docString:                declaration.DocString,
		access:                   checker.accessFromAstAccess(declaration.Access),
		kind:                     declaration.DeclarationKind(),
		pos:                      declaration.Identifier.Pos,
		isConstant:               declaration.IsConstant,
		argumentLabels:           nil,
		allowOuterScopeShadowing: true,
	})
	checker.report(err)

	if checker.PositionInfo != nil && variable != nil {
		checker.recordVariableDeclarationOccurrence(identifier, variable)
		checker.recordVariableDeclarationRange(declaration, identifier, declarationType)
	}

	checker.recordReference(variable, declaration.Value)
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
		checker.Elaboration.SetIsNestedResourceMoveExpression(expression)
	}
}

func (checker *Checker) recordReferenceCreation(target, expr ast.Expression) {
	switch target := target.(type) {
	case *ast.IdentifierExpression:
		targetVariable := checker.valueActivations.Find(target.Identifier.Identifier)
		checker.recordReference(targetVariable, expr)
	default:
		// Currently it's not possible to track the references
		// assigned to member-expressions/index-expressions.
		return
	}
}

func (checker *Checker) recordReference(targetVariable *Variable, expr ast.Expression) {
	if targetVariable == nil {
		return
	}

	if _, isReference := MaybeReferenceType(targetVariable.Type); !isReference {
		return
	}

	targetVariable.referencedResourceVariables = checker.referencedVariables(expr)
}

// referencedVariables return the referenced variables
func (checker *Checker) referencedVariables(expr ast.Expression) (variables []*Variable) {
	refExpressions := referenceExpressions(expr)

	for _, refExpr := range refExpressions {
		var variableRefExpr *ast.Identifier

		switch refExpr := refExpr.(type) {
		case *ast.ReferenceExpression:
			// If it is a reference expression, then find the "root variable".
			// As nested resources cannot be tracked, at least track the "root" if possible.
			// For example, for an expression `&a.b.c as &T`, the "root variable" is `a`.
			variableRefExpr = rootVariableOfExpression(refExpr.Expression)
		case *ast.IdentifierExpression:
			variableRefExpr = &refExpr.Identifier
		case *ast.IndexExpression:
			// If it is a reference expression, then find the "root variable".
			// As nested resources cannot be tracked, at least track the "root" if possible.
			// For example, for an expression `a[b][c]`, the "root variable" is `a`.
			variableRefExpr = rootVariableOfExpression(refExpr.TargetExpression)
		case *ast.MemberExpression:
			// If it is a reference expression, then find the "root variable".
			// As nested resources cannot be tracked, at least track the "root" if possible.
			// For example, for an expression `a.b.c`, the "root variable" is `a`.
			variableRefExpr = rootVariableOfExpression(refExpr.Expression)
		default:
			continue
		}

		if variableRefExpr == nil {
			continue
		}

		referencedVariable := checker.valueActivations.Find(variableRefExpr.Identifier)
		if referencedVariable == nil {
			continue
		}

		// If the referenced variable is again a reference,
		// then find the variable of the root of the reference chain.
		// e.g.:
		//     ref1 = &v
		//     ref2 = &ref1
		//     ref2.field = 3
		//
		// Here, `ref2` refers to `ref1`, which refers to `v`.
		// So `ref2` is actually referring to `v`

		referencedVars := nestedReferencedVariables(referencedVariable)
		if referencedVars != nil {
			variables = append(variables, referencedVars...)
		}
	}

	return
}

// referenceExpressions returns all sub-expressions that may produce a reference.
//
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
func referenceExpressions(expr ast.Expression) []ast.Expression {
	switch expr := expr.(type) {
	case *ast.ReferenceExpression:
		return []ast.Expression{expr}
	case *ast.ForceExpression:
		return referenceExpressions(expr.Expression)
	case *ast.CastingExpression:
		return referenceExpressions(expr.Expression)
	case *ast.BinaryExpression:
		if expr.Operation != ast.OperationNilCoalesce {
			return nil
		}

		refExpressions := make([]ast.Expression, 0)

		lhsRef := referenceExpressions(expr.Left)
		if lhsRef != nil {
			refExpressions = append(refExpressions, lhsRef...)
		}

		rhsRef := referenceExpressions(expr.Right)
		if rhsRef != nil {
			refExpressions = append(refExpressions, rhsRef...)
		}

		return refExpressions
	case *ast.ConditionalExpression:
		refExpressions := make([]ast.Expression, 0)

		thenRef := referenceExpressions(expr.Then)
		if thenRef != nil {
			refExpressions = append(refExpressions, thenRef...)
		}

		elseRef := referenceExpressions(expr.Else)
		if elseRef != nil {
			refExpressions = append(refExpressions, elseRef...)
		}

		return refExpressions
	case *ast.IdentifierExpression,
		*ast.IndexExpression,
		*ast.MemberExpression:
		// For all these expressions, we reach here only if the expression's type is a reference.
		// Hence, no need to check it here again.
		return []ast.Expression{expr}
	default:
		return nil
	}
}

// rootVariableOfExpression returns the identifier expression
// of a var-ref/member-access/index-access expression.
func rootVariableOfExpression(expr ast.Expression) *ast.Identifier {
	for {
		switch typedExpr := expr.(type) {
		case *ast.IdentifierExpression:
			return &typedExpr.Identifier
		case *ast.MemberExpression:
			expr = typedExpr.Expression
		case *ast.IndexExpression:
			expr = typedExpr.TargetExpression
		default:
			return nil
		}
	}
}

func nestedReferencedVariables(variable *Variable) []*Variable {
	// If there are no more referenced variables, then it is the root of the reference chain.
	if len(variable.referencedResourceVariables) == 0 {
		// Add it as the referenced variable, if it is a resource.
		if !variable.Type.IsResourceType() {
			return nil
		}

		return []*Variable{variable}
	}

	var referencedResourceVariables []*Variable
	for _, referencedVar := range variable.referencedResourceVariables {
		nestedReferencedVars := nestedReferencedVariables(referencedVar)
		if nestedReferencedVars != nil {
			referencedResourceVariables = append(referencedResourceVariables, nestedReferencedVars...)

		}
	}

	return referencedResourceVariables
}
