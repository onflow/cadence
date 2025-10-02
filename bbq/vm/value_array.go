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

package vm

import (
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {

	// Methods available for both types of arrays (constant-sized and variable-sized).

	for _, typeQualifier := range []string{
		commons.TypeQualifierArrayVariableSized,
		commons.TypeQualifierArrayConstantSized,
	} {
		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeFirstIndexFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayFirstIndexFunctionType(elementType)
				},
				func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
					array := receiver.(*interpreter.ArrayValue)
					element := arguments[0]
					return array.FirstIndex(context, element)
				},
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeContainsFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayContainsFunctionType(elementType)
				},
				func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
					array := receiver.(*interpreter.ArrayValue)
					element := arguments[0]
					return array.Contains(context, element)
				},
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeReverseFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayReverseFunctionType(arrayType)
				},
				func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
					array := receiver.(*interpreter.ArrayValue)
					return array.Reverse(context)
				},
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeFilterFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayFilterFunctionType(context, elementType)
				},
				func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
					array := receiver.(*interpreter.ArrayValue)
					funcArgument := arguments[0].(FunctionValue)
					return array.Filter(context, funcArgument)
				},
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeMapFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayMapFunctionType(context, arrayType)
				},
				func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
					array := receiver.(*interpreter.ArrayValue)
					funcArgument := arguments[0].(FunctionValue)
					return array.Map(context, funcArgument)
				},
			),
		)
	}

	// Functions available only for variable-sized arrays.

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeAppendFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayAppendFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				element := arguments[0]
				array.Append(context, element)
				return interpreter.Void
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeAppendAllFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				arrayType := arrayTypeFromValue(receiver, context)
				return sema.ArrayAppendAllFunctionType(arrayType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				otherArray := arguments[0].(*interpreter.ArrayValue)

				array.AppendAll(
					context,
					otherArray,
				)
				return interpreter.Void
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeConcatFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				arrayType := arrayTypeFromValue(receiver, context)
				return sema.ArrayConcatFunctionType(arrayType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				otherArray := arguments[0].(*interpreter.ArrayValue)
				return array.Concat(context, otherArray)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeInsertFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayInsertFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				indexValue := arguments[0].(interpreter.NumberValue)
				element := arguments[1]

				index := indexValue.ToInt()

				array.Insert(
					context,
					index,
					element,
				)

				return interpreter.Void
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				indexValue := arguments[0].(interpreter.NumberValue)

				index := indexValue.ToInt()

				return array.Remove(
					context,
					index,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveFirstFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFirstFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				return array.RemoveFirst(context)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveLastFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveLastFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				return array.RemoveLast(context)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeSliceFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArraySliceFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				from := arguments[0].(interpreter.IntValue)
				to := arguments[1].(interpreter.IntValue)
				return array.Slice(
					context,
					from,
					to,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeToConstantSizedFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToConstantSizedFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				constantSizedArrayType := typeArguments[0].(*interpreter.ConstantSizedStaticType)
				return array.ToConstantSized(
					context,
					constantSizedArrayType.Size,
				)
			},
		),
	)

	// Methods available only for constant-sized arrays.

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayConstantSized,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeToVariableSizedFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToVariableSizedFunctionType(elementType)
			},
			func(context *Context, _ []bbq.StaticType, receiver Value, arguments ...Value) Value {
				array := receiver.(*interpreter.ArrayValue)
				return array.ToVariableSized(context)
			},
		),
	)
}

func arrayTypeFromValue(receiver Value, context interpreter.ValueStaticTypeContext) sema.ArrayType {
	return receiver.(*interpreter.ArrayValue).
		SemaType(context)
}

func arrayElementTypeFromValue(receiver Value, context interpreter.ValueStaticTypeContext) sema.Type {
	return arrayTypeFromValue(receiver, context).
		ElementType(false)
}
