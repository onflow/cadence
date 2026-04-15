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
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {

	// Methods available for both types of arrays (constant-sized and variable-sized),
	// and references to them.

	for _, typeQualifier := range []string{
		commons.TypeQualifierArrayVariableSized,
		commons.TypeQualifierArrayConstantSized,
		commons.TypeQualifierArrayVariableSizedRef,
		commons.TypeQualifierArrayConstantSizedRef,
	} {
		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeFirstIndexFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayFirstIndexFunctionType(elementType)
				},
				interpreter.NativeArrayFirstIndexFunction,
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
				interpreter.NativeArrayContainsFunction,
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
				interpreter.NativeArrayReverseFunction,
			),
		)
	}

	// Methods available for both types of arrays (constant-sized and variable-sized).

	for _, typeQualifier := range []string{
		commons.TypeQualifierArrayVariableSized,
		commons.TypeQualifierArrayConstantSized,
	} {
		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeFilterFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					elementType := arrayType.ElementType(false)
					return sema.ArrayFilterFunctionType(
						context,
						arrayType,
						elementType,
					)
				},
				interpreter.NativeArrayFilterFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeMapFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayMapFunctionType(
						context,
						arrayType,
						arrayType,
						func(err error) {
							// TODO:
							panic(err)
						},
					)
				},
				interpreter.NativeArrayMapFunction,
			),
		)
	}

	// Methods available for references to both types of arrays (constant-sized and variable-sized).

	for _, typeQualifier := range []string{
		commons.TypeQualifierArrayVariableSizedRef,
		commons.TypeQualifierArrayConstantSizedRef,
	} {
		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeFilterFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					arrayRefType := sema.NewReferenceType(
						context,
						sema.UnauthorizedAccess,
						arrayType,
					)
					elementType := arrayType.ElementType(false)

					return sema.ArrayFilterFunctionType(
						context,
						arrayRefType,
						elementType,
					)
				},
				func(
					context interpreter.NativeFunctionContext,
					_ interpreter.TypeArgumentsIterator,
					_ interpreter.ArgumentTypesIterator,
					receiver Value,
					args []Value,
				) Value {
					array := interpreter.AssertValueOfType[*interpreter.ArrayValue](receiver)
					funcValue := interpreter.AssertValueOfType[interpreter.FunctionValue](args[0])
					accessedType := interpreter.MustSemaTypeOfValue(receiver, context)

					return array.Filter(context, funcValue, accessedType)
				},
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeMapFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					arrayRefType := sema.NewReferenceType(
						context,
						sema.UnauthorizedAccess,
						arrayType,
					)

					return sema.ArrayMapFunctionType(
						context,
						arrayRefType,
						arrayType,
						func(err error) {
							// TODO:
							panic(err)
						},
					)
				},
				func(
					context interpreter.NativeFunctionContext,
					_ interpreter.TypeArgumentsIterator,
					_ interpreter.ArgumentTypesIterator,
					receiver Value,
					args []Value,
				) Value {
					array := interpreter.AssertValueOfType[*interpreter.ArrayValue](receiver)
					funcValue := interpreter.AssertValueOfType[interpreter.FunctionValue](args[0])

					accessedType := interpreter.MustSemaTypeOfValue(receiver, context)

					return array.Map(context, funcValue, accessedType)
				},
			),
		)
	}

	// Methods available only for variable-sized arrays,
	// and references to them.

	for _, typeQualifier := range []string{
		commons.TypeQualifierArrayVariableSized,
		commons.TypeQualifierArrayVariableSizedRef,
	} {
		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeAppendFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayAppendFunctionType(elementType)
				},
				interpreter.NativeArrayAppendFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeAppendAllFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayAppendAllFunctionType(arrayType)
				},
				interpreter.NativeArrayAppendAllFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeConcatFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayConcatFunctionType(arrayType)
				},
				interpreter.NativeArrayConcatFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeInsertFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayInsertFunctionType(elementType)
				},
				interpreter.NativeArrayInsertFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeRemoveFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayRemoveFunctionType(elementType)
				},
				interpreter.NativeArrayRemoveFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeRemoveFirstFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayRemoveFirstFunctionType(elementType)
				},
				interpreter.NativeArrayRemoveFirstFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeRemoveLastFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayRemoveLastFunctionType(elementType)
				},
				interpreter.NativeArrayRemoveLastFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeSliceFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArraySliceFunctionType(elementType)
				},
				interpreter.NativeArraySliceFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeToConstantSizedFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayToConstantSizedFunctionType(elementType)
				},
				interpreter.NativeArrayToConstantSizedFunction,
			),
		)
	}

	// Methods available only for constant-sized arrays.
	// and references to them.

	for _, typeQualifier := range []string{
		commons.TypeQualifierArrayConstantSized,
		commons.TypeQualifierArrayConstantSizedRef,
	} {
		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeToVariableSizedFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayToVariableSizedFunctionType(elementType)
				},
				interpreter.NativeArrayToVariableSizedFunction,
			),
		)
	}
}

func arrayTypeFromValue(receiver Value, context interpreter.ValueStaticTypeContext) sema.ArrayType {
	return receiver.(*interpreter.ArrayValue).
		SemaType(context)
}

func arrayElementTypeFromValue(receiver Value, context interpreter.ValueStaticTypeContext) sema.Type {
	return arrayTypeFromValue(receiver, context).
		ElementType(false)
}
