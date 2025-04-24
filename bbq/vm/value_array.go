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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeAppendFunctionName,
			// TODO:
			sema.ArrayAppendFunctionType(sema.AnyType),
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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeAppendAllFunctionName,
			// TODO:
			sema.ArrayAppendAllFunctionType(sema.AnyType),
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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeConcatFunctionName,
			// TODO:
			sema.ArrayConcatFunctionType(sema.AnyType),
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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeInsertFunctionName,
			// TODO:
			sema.ArrayInsertFunctionType(sema.AnyType),
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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeRemoveFunctionName,
			// TODO:
			sema.ArrayRemoveFunctionType(sema.AnyType),
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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeRemoveFirstFunctionName,
			// TODO:
			sema.ArrayRemoveFirstFunctionType(sema.AnyType),
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				return array.RemoveFirst(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewBoundNativeFunctionValue(
			sema.ArrayTypeRemoveLastFunctionName,
			// TODO:
			sema.ArrayRemoveLastFunctionType(sema.AnyType),
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				return array.RemoveLast(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewBoundNativeFunctionValue(
			sema.ArrayTypeFirstIndexFunctionName,
			// TODO:
			sema.ArrayFirstIndexFunctionType(sema.AnyType),
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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeContainsFunctionName,
			// TODO:
			sema.ArrayContainsFunctionType(sema.AnyType),
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
		NewBoundNativeFunctionValue(
			sema.ArrayTypeSliceFunctionName,
			// TODO:
			sema.ArraySliceFunctionType(sema.AnyType),
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

	//RegisterTypeBoundFunction(
	//	commons.TypeQualifierArray,
	//	NewBoundNativeFunctionValue(
	//		sema.ArrayTypeReverseFunctionName,
	//		// TODO:
	//		sema.ArrayReverseFunctionType(nil),
	//		func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
	//			value := arguments[receiverIndex]
	//			array := value.(*interpreter.ArrayValue)
	//			return array.Reverse(context, EmptyLocationRange)
	//		},
	//	),
	//)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewBoundNativeFunctionValue(
			sema.ArrayTypeFilterFunctionName,
			// TODO:
			sema.ArrayFilterFunctionType(nil, sema.AnyType),
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				funcArgument := arguments[1].(FunctionValue)
				return array.Filter(context, EmptyLocationRange, funcArgument)
			},
		),
	)

	//RegisterTypeBoundFunction(
	//	commons.TypeQualifierArray,
	//	NewBoundNativeFunctionValue(
	//		sema.ArrayTypeMapFunctionName,
	//		// TODO:
	//		sema.ArrayMapFunctionType(nil, nil),
	//		func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
	//			value := arguments[receiverIndex]
	//			array := value.(*interpreter.ArrayValue)
	//			funcArgument := arguments[1].(FunctionValue)
	//			return array.Map(context, EmptyLocationRange, funcArgument)
	//		},
	//	),
	//)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewBoundNativeFunctionValue(
			sema.ArrayTypeToVariableSizedFunctionName,
			// TODO:
			sema.ArrayToVariableSizedFunctionType(sema.AnyType),
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				value := arguments[receiverIndex]
				array := value.(*interpreter.ArrayValue)
				return array.ToVariableSized(context, EmptyLocationRange)
			},
		),
	)

	RegisterTypeBoundFunction(
		commons.TypeQualifierArray,
		NewBoundNativeFunctionValue(
			sema.ArrayTypeToConstantSizedFunctionName,
			// TODO:
			sema.ArrayToConstantSizedFunctionType(sema.AnyType),
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
