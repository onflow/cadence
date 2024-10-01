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

package sema

import (
	"github.com/onflow/cadence/runtime/ast"
)

// VisitReferenceExpression checks a reference expression
func (checker *Checker) VisitReferenceExpression(referenceExpression *ast.ReferenceExpression) Type {

	resultType := checker.expectedType
	if resultType == nil {
		checker.report(
			&TypeAnnotationRequiredError{
				Cause: "cannot infer type from reference expression:",
				Pos:   referenceExpression.Expression.StartPosition(),
			},
		)
		return InvalidType
	}

	// Check the result type and ensure it is a reference type
	var referenceType *ReferenceType
	var expectedLeftType, returnType Type

	if !resultType.IsInvalidType() {
		// Reference expressions may reference a value which has an optional type.
		// For example, the result of indexing into a dictionary is an optional:
		//
		// let ints: {Int: String} = {0: "zero"}
		// let ref: &T? = &ints[0] as &T?   // read as (&T)?
		//
		// In this case the reference expression's type is an optional type.
		// Unwrap it (recursively) to get the actual reference type
		expectedLeftType, returnType, referenceType =
			checker.expectedTypeForReferencedExpr(resultType, referenceExpression)
	}

	// Type-check the referenced expression

	referencedExpression := referenceExpression.Expression

	beforeErrors := len(checker.errors)

	referencedType, actualType := checker.visitExpression(referencedExpression, referenceExpression, expectedLeftType)

	// check that the type of the referenced value is not itself a reference
	var requireNoReferenceNesting func(actualType Type)
	requireNoReferenceNesting = func(actualType Type) {
		switch nestedReference := actualType.(type) {
		case *ReferenceType:
			checker.report(&NestedReferenceError{
				Type:  nestedReference,
				Range: checker.expressionRange(referenceExpression),
			})
		case *OptionalType:
			requireNoReferenceNesting(nestedReference.Type)
		}
	}
	requireNoReferenceNesting(actualType)

	hasErrors := len(checker.errors) > beforeErrors
	if !hasErrors {
		checker.checkOptionalityMatch(
			expectedLeftType,
			actualType,
			referencedExpression,
			referenceExpression,
		)
	}

	if referenceType == nil {
		return InvalidType
	}

	checker.checkUnusedExpressionResourceLoss(referencedType, referencedExpression)

	checker.Elaboration.SetReferenceExpressionBorrowType(referenceExpression, returnType)

	return returnType
}

func (checker *Checker) expectedTypeForReferencedExpr(
	expectedType Type,
	hasPosition ast.HasPosition,
) (expectedLeftType, returnType Type, referenceType *ReferenceType) {
	// Reference expressions may reference a value which has an optional type.
	// For example, the result of indexing into a dictionary is an optional:
	//
	// let ints: {Int: String} = {0: "zero"}
	// let ref: &T? = &ints[0] as &T?   // read as (&T)?
	//
	// In this case the reference expression's type is an optional type.
	// Unwrap it to get the actual reference type

	switch expectedType := expectedType.(type) {
	case *OptionalType:
		expectedLeftType, returnType, referenceType =
			checker.expectedTypeForReferencedExpr(expectedType.Type, hasPosition)

		// Re-wrap with an optional
		expectedLeftType = &OptionalType{Type: expectedLeftType}
		returnType = &OptionalType{Type: returnType}

	case *ReferenceType:
		referencedType := expectedType.Type
		if referencedOptionalType, referenceToOptional := referencedType.(*OptionalType); referenceToOptional {
			checker.report(
				&ReferenceToAnOptionalError{
					ReferencedOptionalType: referencedOptionalType,
					Range:                  ast.NewRangeFromPositioned(checker.memoryGauge, hasPosition),
				},
			)
		}

		return expectedType.Type, expectedType, expectedType

	default:
		checker.report(
			&NonReferenceTypeReferenceError{
				ActualType: expectedType,
				Range:      ast.NewRangeFromPositioned(checker.memoryGauge, hasPosition),
			},
		)

		return InvalidType, InvalidType, nil
	}

	return
}

func (checker *Checker) checkOptionalityMatch(
	expectedType, actualType Type,
	referencedExpression ast.Expression,
	referenceExpression ast.Expression,
) {

	// Do not report an error if the `expectedType` is unknown
	if expectedType == nil || expectedType.IsInvalidType() {
		return
	}

	// If the reference type was an optional type,
	// we proposed an optional type to the referenced expression.
	//
	// Check that it actually has an optional type

	// If the reference type was a non-optional type,
	// check that the referenced expression does not have an optional type

	expectedOptional, expectedIsOptional := expectedType.(*OptionalType)
	actualOptional, actualIsOptional := actualType.(*OptionalType)

	if expectedIsOptional && actualIsOptional {
		checker.checkOptionalityMatch(
			expectedOptional.Type,
			actualOptional.Type,
			referencedExpression,
			referenceExpression,
		)
		return
	}

	if expectedIsOptional != actualIsOptional {
		checker.report(&TypeMismatchError{
			ExpectedType: expectedType,
			ActualType:   actualType,
			Expression:   referencedExpression,
			Range:        checker.expressionRange(referenceExpression),
		})
	}
}
