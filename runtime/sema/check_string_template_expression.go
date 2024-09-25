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

import "github.com/onflow/cadence/runtime/ast"

func (checker *Checker) VisitStringTemplateExpression(stringTemplateExpression *ast.StringTemplateExpression) Type {

	// visit all elements

	var elementType Type

	elementCount := len(stringTemplateExpression.Expressions)

	var argumentTypes []Type
	if elementCount > 0 {
		argumentTypes = make([]Type, elementCount)

		for i, element := range stringTemplateExpression.Expressions {
			valueType := checker.VisitExpression(element, stringTemplateExpression, elementType)

			argumentTypes[i] = valueType

			// All number types, addresses, path types, bool and strings are supported in string template
			if IsSubType(valueType, NumberType) || IsSubType(valueType, TheAddressType) ||
				IsSubType(valueType, PathType) || IsSubType(valueType, StringType) || IsSubType(valueType, BoolType) {
				checker.checkResourceMoveOperation(element, valueType)
			} else {
				checker.report(
					&TypeMismatchWithDescriptionError{
						ActualType:              valueType,
						ExpectedTypeDescription: "a type with built-in toString() or bool",
						Range:                   ast.NewRangeFromPositioned(checker.memoryGauge, element),
					},
				)
			}
		}
	}

	checker.Elaboration.SetStringTemplateExpressionTypes(
		stringTemplateExpression,
		StringTemplateExpressionTypes{
			ArgumentTypes: argumentTypes,
		},
	)

	return StringType
}
