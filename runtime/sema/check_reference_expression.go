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
)

// VisitReferenceExpression checks a reference expression `&t as T`,
// where `t` is the referenced expression, and `T` is the result type.
//
func (checker *Checker) VisitReferenceExpression(referenceExpression *ast.ReferenceExpression) ast.Repr {

	// Check the result type and ensure it is a reference type

	resultType := checker.ConvertType(referenceExpression.Type)
	checker.checkInvalidInterfaceAsType(resultType, referenceExpression.Type)

	var referenceType *ReferenceType
	var targetType, returnType Type

	if !resultType.IsInvalidType() {
		var ok bool
		// Reference expressions may reference a value which has an optional type.
		// For example, the result of indexing into a dictionary is an optional:
		//
		// let ints: {Int: String} = {0: "zero"}
		// let ref: &T? = &ints[0] as &T?   // read as (&T)?
		//
		// In this case the reference expression's type is an optional type.
		// Unwrap it one level to get the actual reference type
		optType, optOk := resultType.(*OptionalType)
		if optOk {
			resultType = optType.Type
		}

		referenceType, ok = resultType.(*ReferenceType)
		if !ok {
			checker.report(
				&NonReferenceTypeReferenceError{
					ActualType: resultType,
					Range:      ast.NewRangeFromPositioned(checker.memoryGauge, referenceExpression.Type),
				},
			)
		} else {
			targetType = referenceType.Type
			returnType = referenceType
			if optOk {
				targetType = &OptionalType{Type: targetType}
				returnType = &OptionalType{Type: returnType}
			}
		}
	}

	// Type-check the referenced expression

	referencedExpression := referenceExpression.Expression

	_, _ = checker.visitExpression(referencedExpression, targetType)

	if referenceType == nil {
		return InvalidType
	}

	checker.Elaboration.ReferenceExpressionBorrowTypes[referenceExpression] = returnType

	return returnType
}
