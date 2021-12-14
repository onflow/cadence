/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

// VisitReferenceExpression checks a reference expression `&t as T`,
// where `t` is the referenced expression, and `T` is the result type.
//
func (checker *Checker) VisitReferenceExpression(referenceExpression *ast.ReferenceExpression) ast.Repr {

	// Check the result type and ensure it is a reference type

	resultType := checker.ConvertType(referenceExpression.Type)
	checker.checkInvalidInterfaceAsType(resultType, referenceExpression.Type)

	var referenceType *ReferenceType
	var targetType, referencedType Type
	var returnType Type

	if !resultType.IsInvalidType() {
		var ok bool
		// indexed access to dictionaries require an optional reference as their
		// target type, so allow one level of optionals
		optType, optOk := resultType.(*OptionalType)
		if optOk {
			resultType = optType.Type
		}
		referenceType, ok = resultType.(*ReferenceType)
		if !ok {
			checker.report(
				&NonReferenceTypeReferenceError{
					ActualType: resultType,
					Range:      ast.NewRangeFromPositioned(referenceExpression.Type),
				},
			)
		} else {
			targetType = referenceType.Type
			if optOk {
				targetType = &OptionalType{Type: targetType}
			}
		}
	}
	returnType = referenceType

	// Type-check the referenced expression

	referencedExpression := referenceExpression.Expression

	_, referencedType = checker.visitExpression(referencedExpression, targetType)

	// Unwrap the optional one level, but not infinitely
	if optionalReferencedType, ok := referencedType.(*OptionalType); ok {
		// if referenced type is optional, require that target type is also optional
		referencedType = optionalReferencedType.Type
		returnType = &OptionalType{Type: referenceType}
	}

	if _, ok := referencedType.(*OptionalType); ok {
		checker.report(
			&OptionalTypeReferenceError{
				ActualType: referencedType,
				Range:      expressionRange(referencedExpression),
			},
		)
	}

	if referenceType == nil {
		return InvalidType
	}

	checker.Elaboration.ReferenceExpressionBorrowTypes[referenceExpression] = referenceType

	return returnType
}
