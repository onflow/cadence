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

import "github.com/onflow/cadence/runtime/ast"

func (checker *Checker) VisitArrayExpression(expression *ast.ArrayExpression) ast.Repr {

	inferType := false

	// visit all elements, ensure they are all the same type

	expectedType := UnwrapOptionalType(checker.expectedType)

	var elementType Type
	if expectedType != nil {

		switch typ := expectedType.(type) {

		case *ConstantSizedType:
			elementType = typ.ElementType(true)

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
			elementType = typ.ElementType(true)

		default:
			// If the expected type is not an array type, then it could either be an invalid type, or a super type
			// (like: AnyStruct, etc.). Either case, infer the type from the expression.
			inferType = true
		}
	} else {
		inferType = true
	}

	argumentTypes := make([]Type, len(expression.Values))

	for i, value := range expression.Values {
		valueType := checker.VisitExpression(value, elementType)

		argumentTypes[i] = valueType

		checker.checkVariableMove(value)
		checker.checkResourceMoveOperation(value, valueType)

		// infer element type from first element
		// TODO: find common super type?
		if elementType == nil {
			elementType = valueType
		}
	}

	checker.Elaboration.ArrayExpressionArgumentTypes[expression] = argumentTypes

	if elementType == nil {
		// i.e: contextually expected type is not available and array has zero elements.
		elementType = NeverType
	}

	checker.Elaboration.ArrayExpressionElementType[expression] = elementType

	if inferType {
		return &VariableSizedType{
			Type: elementType,
		}
	}

	return expectedType
}
