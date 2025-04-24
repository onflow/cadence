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
	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeAppendFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayAppendFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				element := arguments[1]
				array.Append(context, EmptyLocationRange, element)
				return interpreter.Void
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeAppendAllFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayAppendAllFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				otherArray := arguments[1].(*interpreter.ArrayValue)

				array.AppendAll(
					context,
					EmptyLocationRange,
					otherArray,
				)
				return interpreter.Void
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeConcatFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayConcatFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				otherArray := arguments[1].(*interpreter.ArrayValue)
				array.Concat(context, EmptyLocationRange, otherArray)
				return interpreter.Void
			},
		),
	)
	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeInsertFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayInsertFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				indexValue := arguments[1].(interpreter.NumberValue)
				element := arguments[2]

				locationRange := EmptyLocationRange
				index := indexValue.ToInt(locationRange)

				array.Insert(
					context,
					locationRange,
					index,
					element,
				)

				return interpreter.Void
			},
		),
	)
	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeRemoveFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				indexValue := arguments[1].(interpreter.NumberValue)

				locationRange := EmptyLocationRange
				index := indexValue.ToInt(locationRange)

				return array.Remove(
					context,
					locationRange,
					index,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeRemoveFirstFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFirstFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				return array.RemoveFirst(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeRemoveLastFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveLastFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				return array.RemoveLast(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeFirstIndexFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayFirstIndexFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				element := arguments[1]
				return array.FirstIndex(context, EmptyLocationRange, element)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeContainsFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayContainsFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				element := arguments[1]
				return array.Contains(context, EmptyLocationRange, element)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeSliceFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArraySliceFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				from := arguments[1].(interpreter.IntValue)
				to := arguments[2].(interpreter.IntValue)
				return array.Slice(
					context,
					from,
					to,
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeReverseFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				arrayType := arrayTypeFromValue(receiver, context)
				return sema.ArrayReverseFunctionType(arrayType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				return array.Reverse(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeFilterFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayFilterFunctionType(context, elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				funcArgument := arguments[1].(FunctionValue)
				return array.Filter(context, EmptyLocationRange, funcArgument)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeMapFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				arrayType := arrayTypeFromValue(receiver, context)
				return sema.ArrayMapFunctionType(context, arrayType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				funcArgument := arguments[1].(FunctionValue)
				return array.Map(context, EmptyLocationRange, funcArgument)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeToVariableSizedFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToVariableSizedFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				return array.ToVariableSized(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewDerivedTypeBoundNativeFunctionValue(
			sema.ArrayTypeToConstantSizedFunctionName,
			func(receiver Value, context interpreter.TypeConverter) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToConstantSizedFunctionType(elementType)
			},
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				constantSizedArrayType := typeArguments[1].(*interpreter.ConstantSizedStaticType)
				return array.ToConstantSized(
					context,
					EmptyLocationRange,
					constantSizedArrayType.Size,
				)
			},
		),
	)
}

func arrayTypeFromValue(receiver Value, context interpreter.TypeConverter) sema.ArrayType {
	return receiver.(*interpreter.ArrayValue).
		SemaType(context)
}

func arrayElementTypeFromValue(receiver Value, context interpreter.TypeConverter) sema.Type {
	return arrayTypeFromValue(receiver, context).
		ElementType(false)
}
