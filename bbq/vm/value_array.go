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
	"github.com/onflow/cadence/errors"
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
					accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
					arrayType := arrayTypeFromValue(receiver, context)
					return sema.ArrayReverseFunctionType(context, accessedType, arrayType)
				},
				interpreter.NativeArrayReverseFunction,
			).WithDereferenceReceiver(false),
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
					accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
					valueIndexableType := accessedType.(sema.ValueIndexableType)
					elementType := valueIndexableType.ElementType(false)

					return sema.ArrayFilterFunctionType(
						context,
						accessedType,
						elementType,
					)
				},
				interpreter.NativeArrayFilterFunction,
			).WithDereferenceReceiver(false),
		)

		registerBuiltinTypeBoundFunction(
			typeQualifier,
			NewNativeFunctionValueWithDerivedType(
				sema.ArrayTypeMapFunctionName,
				func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
					accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
					arrayType := arrayTypeFromSemaType(accessedType)

					return sema.ArrayMapFunctionType(
						context,
						accessedType,
						arrayType,
					)
				},
				interpreter.NativeArrayMapFunction,
			).WithDereferenceReceiver(false),
		)
	}

	// Methods available only for variable-sized arrays,
	// and references to them.
	typeQualifier := commons.TypeQualifierArrayVariableSized

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
				accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
				arrayType := arrayTypeFromValue(receiver, context)
				return sema.ArrayConcatFunctionType(context, accessedType, arrayType)
			},
			interpreter.NativeArrayConcatFunction,
		).WithDereferenceReceiver(false),
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
				accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFunctionType(context, accessedType, elementType)
			},
			interpreter.NativeArrayRemoveFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeQualifier,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveFirstFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveFirstFunctionType(context, accessedType, elementType)
			},
			interpreter.NativeArrayRemoveFirstFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeQualifier,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeRemoveLastFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayRemoveLastFunctionType(context, accessedType, elementType)
			},
			interpreter.NativeArrayRemoveLastFunction,
		),
	)

	registerBuiltinTypeBoundFunction(
		typeQualifier,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeSliceFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArraySliceFunctionType(context, accessedType, elementType)
			},
			interpreter.NativeArraySliceFunction,
		).WithDereferenceReceiver(false),
	)

	registerBuiltinTypeBoundFunction(
		typeQualifier,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeToConstantSizedFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToConstantSizedFunctionType(context, accessedType, elementType)
			},
			interpreter.NativeArrayToConstantSizedFunction,
		).WithDereferenceReceiver(false),
	)

	// Methods available only for constant-sized arrays
	// and references to them.
	typeQualifier = commons.TypeQualifierArrayConstantSized

	registerBuiltinTypeBoundFunction(
		typeQualifier,
		NewNativeFunctionValueWithDerivedType(
			sema.ArrayTypeToVariableSizedFunctionName,
			func(receiver Value, context interpreter.ValueStaticTypeContext) *sema.FunctionType {
				accessedType := context.SemaTypeFromStaticType(receiver.StaticType(context))
				elementType := arrayElementTypeFromValue(receiver, context)
				return sema.ArrayToVariableSizedFunctionType(
					context,
					accessedType,
					elementType,
				)
			},
			interpreter.NativeArrayToVariableSizedFunction,
		).WithDereferenceReceiver(false),
	)
}

func arrayTypeFromSemaType(accessedType sema.Type) sema.ArrayType {
	switch accessedType := accessedType.(type) {
	case sema.ArrayType:
		return accessedType
	case *sema.ReferenceType:
		return arrayTypeFromSemaType(accessedType.Type)
	default:
		panic(errors.NewUnreachableError())
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
