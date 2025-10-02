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

	// Methods available for both types of arrays (constant-sized and variable-sized).

	for _, typeQualifier := range []string{
		commons.TypeQualifierArrayVariableSized,
		commons.TypeQualifierArrayConstantSized,
	} {
		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewUnifiedNativeFunctionValueWithDerivedType(
				sema.ArrayTypeFirstIndexFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayFirstIndexFunctionType(elementType)
				},
				interpreter.UnifiedArrayFirstIndexFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewUnifiedNativeFunctionValueWithDerivedType(
				sema.ArrayTypeContainsFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayContainsFunctionType(elementType)
				},
				interpreter.UnifiedArrayContainsFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewUnifiedNativeFunctionValueWithDerivedType(
				sema.ArrayTypeReverseFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayReverseFunctionType(arrayType)
				},
				interpreter.UnifiedArrayReverseFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewUnifiedNativeFunctionValueWithDerivedType(
				sema.ArrayTypeFilterFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					elementType := arrayElementTypeFromValue(receiver, context)
					return sema.ArrayFilterFunctionType(context, elementType)
				},
				interpreter.UnifiedArrayFilterFunction,
			),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewUnifiedNativeFunctionValueWithDerivedType(
				sema.ArrayTypeMapFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayMapFunctionType(context, arrayType)
				},
				interpreter.UnifiedArrayMapFunction,
			),
		)
	}

	// Functions available only for variable-sized arrays.

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeAppendFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayAppendFunctionType(elementType)
			},
			interpreter.UnifiedArrayAppendFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeAppendAllFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				arrayType := arrayTypeFromValue(receiver, context)
				return sema.ArrayAppendAllFunctionType(arrayType)
			},
			interpreter.UnifiedArrayAppendAllFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeConcatFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				arrayType := arrayTypeFromValue(receiver, context)
				return sema.ArrayConcatFunctionType(arrayType)
			},
			interpreter.UnifiedArrayConcatFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeInsertFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayInsertFunctionType(elementType)
			},
			interpreter.UnifiedArrayInsertFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFunctionType(elementType)
			},
			interpreter.UnifiedArrayRemoveFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveFirstFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFirstFunctionType(elementType)
			},
			interpreter.UnifiedArrayRemoveFirstFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveLastFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveLastFunctionType(elementType)
			},
			interpreter.UnifiedArrayRemoveLastFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeSliceFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArraySliceFunctionType(elementType)
			},
			interpreter.UnifiedArraySliceFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayVariableSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeToConstantSizedFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToConstantSizedFunctionType(elementType)
			},
			interpreter.UnifiedArrayToConstantSizedFunction,
		),
	)

	// Methods available only for constant-sized arrays.

	registerBuiltinTypeBoundFunction(
		commons.TypeQualifierArrayConstantSized,
		NewUnifiedNativeFunctionValueWithDerivedType(
			sema.ArrayTypeToVariableSizedFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToVariableSizedFunctionType(elementType)
			},
			interpreter.UnifiedArrayToVariableSizedFunction,
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
