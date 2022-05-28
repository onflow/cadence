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

func (checker *Checker) VisitArrayExpression(expression *ast.ArrayExpression) ast.Repr {

	// visit all elements, ensure they are all the same type

	expectedType := UnwrapOptionalType(checker.expectedType)

	var elementType Type
	var resultType ArrayType

	switch typ := expectedType.(type) {

	case *ConstantSizedType:
		elementType = typ.ElementType(false)
		resultType = typ

		literalCount := int64(len(expression.Values))
		if typ.Size != literalCount {
			checker.report(
				&ConstantSizedArrayLiteralSizeError{
					ExpectedSize: typ.Size,
					ActualSize:   literalCount,
					Range:        expression.Range,
				},
			)
		}

	case *VariableSizedType:
		elementType = typ.ElementType(false)
		resultType = typ

	default:
		// If the expected type is AnyStruct or AnyResource, and the array is empty,
		// then expect the elements to also be of the same type.
		// Otherwise, infer the type from the expression.
		if len(expression.Values) == 0 {
			elementType = expectedType
			resultType = &VariableSizedType{
				Type: elementType,
			}
		}
	}

	argumentTypes := make([]Type, len(expression.Values))

	for i, value := range expression.Values {
		valueType := checker.VisitExpression(value, elementType)

		argumentTypes[i] = valueType

		checker.checkVariableMove(value)
		checker.checkResourceMoveOperation(value, valueType)
	}

	checker.Elaboration.ArrayExpressionArgumentTypes[expression] = argumentTypes

	if elementType == nil {
		// Contextually expected type is not available.
		// Therefore, find the least common supertype of the elements.
		elementType = LeastCommonSuperType(argumentTypes...)

		if elementType == InvalidType {
			checker.report(
				&TypeAnnotationRequiredError{
					Cause: "cannot infer type from array literal: ",
					Pos:   expression.StartPos,
				},
			)

			return InvalidType
		}

		resultType = &VariableSizedType{
			Type: elementType,
		}
	}

	checker.Elaboration.ArrayExpressionArrayType[expression] = resultType

	return resultType
}
