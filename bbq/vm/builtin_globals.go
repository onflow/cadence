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
	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

const (
	ReceiverIndex                   = 0
	TypeBoundFunctionArgumentOffset = 1
)

type BuiltinGlobalsProvider func(location common.Location) *activations.Activation[Variable]

var defaultBuiltinGlobals = activations.NewActivation[Variable](nil, nil)

func DefaultBuiltinGlobals() *activations.Activation[Variable] {
	return defaultBuiltinGlobals
}

func RegisterBuiltinFunction(functionValue *NativeFunctionValue) {
	registerGlobalFunction(
		functionValue.Name,
		functionValue,
		defaultBuiltinGlobals,
	)
}

func registerGlobalFunction(
	functionName string,
	functionValue *NativeFunctionValue,
	activation *activations.Activation[Variable],
) {
	existing := activation.Find(functionName)
	if existing != nil {
		panic(errors.NewUnexpectedError("function already exists: %s", functionName))
	}
	variable := &interpreter.SimpleVariable{}
	variable.InitializeWithValue(functionValue)
	activation.Set(functionName, variable)
}

func RegisterBuiltinTypeBoundFunction(typeName string, functionValue *NativeFunctionValue) {
	// Update the name of the function to be type-qualified
	qualifiedName := commons.QualifiedName(typeName, functionValue.Name)
	functionValue.Name = qualifiedName

	RegisterBuiltinFunction(functionValue)
}

func RegisterBuiltinTypeBoundCommonFunction(typeName string, functionValue *NativeFunctionValue) {
	// Here the function value is common for many types.
	// Hence, do not update the function name to be type-qualified.
	// Only the key in the map is type-qualified.
	qualifiedName := commons.QualifiedName(typeName, functionValue.Name)
	registerGlobalFunction(
		qualifiedName,
		functionValue,
		defaultBuiltinGlobals,
	)
}

func init() {

	// Type constructors

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.MetaTypeName,
			sema.MetaTypeFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				return interpreter.NewTypeValue(
					context.MemoryGauge,
					typeArguments[0],
				)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.OptionalTypeFunctionName,
			sema.OptionalTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				typeValue := arguments[0].(interpreter.TypeValue)

				return interpreter.ConstructOptionalTypeValue(context, typeValue)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.VariableSizedArrayTypeFunctionName,
			sema.VariableSizedArrayTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				typeValue := arguments[0].(interpreter.TypeValue)

				return interpreter.ConstructVariableSizedArrayTypeValue(
					context,
					typeValue,
				)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.ConstantSizedArrayTypeFunctionName,
			sema.ConstantSizedArrayTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				typeValue := arguments[0].(interpreter.TypeValue)
				sizeValue := arguments[1].(interpreter.IntValue)

				return interpreter.ConstructConstantSizedArrayTypeValue(
					context,
					EmptyLocationRange,
					typeValue,
					sizeValue,
				)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.DictionaryTypeFunctionName,
			sema.DictionaryTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				keyTypeValue := arguments[0].(interpreter.TypeValue)
				valueTypeValue := arguments[1].(interpreter.TypeValue)

				return interpreter.ConstructDictionaryTypeValue(
					context,
					keyTypeValue,
					valueTypeValue,
				)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.CompositeTypeFunctionName,
			sema.CompositeTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				typeIDValue := arguments[0].(*interpreter.StringValue)

				return interpreter.ConstructCompositeTypeValue(context, typeIDValue)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.FunctionTypeFunctionName,
			sema.FunctionTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				parameterTypeValues := arguments[0].(*interpreter.ArrayValue)
				returnTypeValue := arguments[1].(interpreter.TypeValue)

				return interpreter.ConstructFunctionTypeValue(
					context,
					EmptyLocationRange,
					parameterTypeValues,
					returnTypeValue,
				)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.ReferenceTypeFunctionName,
			sema.ReferenceTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				entitlementValues := arguments[0].(*interpreter.ArrayValue)
				typeValue := arguments[1].(interpreter.TypeValue)

				return interpreter.ConstructReferenceTypeValue(
					context,
					EmptyLocationRange,
					entitlementValues,
					typeValue,
				)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.IntersectionTypeFunctionName,
			sema.IntersectionTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				intersectionIDs := arguments[0].(*interpreter.ArrayValue)

				return interpreter.ConstructIntersectionTypeValue(
					context,
					EmptyLocationRange,
					intersectionIDs,
				)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.CapabilityTypeFunctionName,
			sema.CapabilityTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				typeValue := arguments[0].(interpreter.TypeValue)

				return interpreter.ConstructCapabilityTypeValue(context, typeValue)
			},
		),
	)

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.InclusiveRangeTypeFunctionName,
			sema.InclusiveRangeTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

				typeValue := arguments[0].(interpreter.TypeValue)

				return interpreter.ConstructInclusiveRangeTypeValue(context, typeValue)
			},
		),
	)

	// Value conversion functions
	for _, declaration := range interpreter.ConverterDeclarations {
		// NOTE: declare in loop, as captured in closure below
		convert := declaration.Convert

		functionType := sema.BaseValueActivation.Find(declaration.Name).Type.(*sema.FunctionType)

		function := NewNativeFunctionValue(
			declaration.Name,
			functionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				return convert(
					context.MemoryGauge,
					arguments[0],
					EmptyLocationRange,
				)
			},
		)
		RegisterBuiltinFunction(function)

		addMember := func(name string, value interpreter.Value) {
			if function.fields == nil {
				function.fields = make(map[string]interpreter.Value)
			}
			if _, exists := function.fields[name]; exists {
				panic(errors.NewUnexpectedError("member already exists: %s", name))
			}
			function.fields[name] = value
		}

		if declaration.Min != nil {
			addMember(sema.NumberTypeMinFieldName, declaration.Min)
		}

		if declaration.Max != nil {
			addMember(sema.NumberTypeMaxFieldName, declaration.Max)
		}

		if stringValueParser, ok := interpreter.StringValueParsers[declaration.Name]; ok {
			RegisterBuiltinTypeBoundFunction(
				commons.TypeQualifier(stringValueParser.ReceiverType),
				newFromStringFunction(stringValueParser),
			)
		}

		if bigEndianBytesConverter, ok := interpreter.BigEndianBytesConverters[declaration.Name]; ok {
			RegisterBuiltinTypeBoundFunction(
				commons.TypeQualifier(bigEndianBytesConverter.ReceiverType),
				newFromBigEndianBytesFunction(bigEndianBytesConverter),
			)
		}
	}

	// Value constructors

	RegisterBuiltinFunction(
		NewNativeFunctionValue(
			sema.StringType.String(),
			sema.StringFunctionType,
			func(_ *Context, _ []bbq.StaticType, _ ...Value) Value {
				return interpreter.EmptyString
			},
		),
	)

	// Register type-bound functions that are common to many types.
	registerBuiltinCommonTypeBoundFunctions()

	registerBuiltinSaturatingArithmeticFunctions()
}

func registerBuiltinCommonTypeBoundFunctions() {
	for _, builtinType := range commons.BuiltinTypes {
		typeQualifier := commons.TypeQualifier(builtinType)
		registerBuiltinTypeBoundFunctions(typeQualifier)
	}

	for _, function := range commonBuiltinTypeBoundFunctions {
		IndexedCommonBuiltinTypeBoundFunctions[function.Name] = function
	}
}

func registerBuiltinTypeBoundFunctions(
	typeQualifier string,
) {
	for _, boundFunction := range commonBuiltinTypeBoundFunctions {
		RegisterBuiltinTypeBoundCommonFunction(
			typeQualifier,
			boundFunction,
		)
	}
}

// Built-in functions that are common to all the types.
var commonBuiltinTypeBoundFunctions = []*NativeFunctionValue{

	// `isInstance` function
	NewNativeFunctionValue(
		sema.IsInstanceFunctionName,
		sema.IsInstanceFunctionType,
		func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
			value := arguments[ReceiverIndex]

			typeValue, ok := arguments[TypeBoundFunctionArgumentOffset].(interpreter.TypeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return interpreter.IsInstance(context, value, typeValue)
		},
	),

	// `getType` function
	NewNativeFunctionValue(
		sema.GetTypeFunctionName,
		sema.GetTypeFunctionType,
		func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
			value := arguments[ReceiverIndex]
			return interpreter.ValueGetType(context, value)
		},
	),

	// TODO: add remaining functions
}

var IndexedCommonBuiltinTypeBoundFunctions = map[string]*NativeFunctionValue{}

func registerBuiltinSaturatingArithmeticFunctions() {
	for _, ty := range common.Concat(
		sema.AllUnsignedFixedPointTypes,
		sema.AllSignedFixedPointTypes,
		sema.AllUnsignedIntegerTypes,
		sema.AllSignedIntegerTypes,
	) {
		registerBuiltinTypeSaturatingArithmeticFunctions(ty.(sema.SaturatingArithmeticType))
	}
}

func registerBuiltinTypeSaturatingArithmeticFunctions(t sema.SaturatingArithmeticType) {

	register := func(
		functionName string,
		op func(context *Context, v, other interpreter.NumberValue) interpreter.NumberValue,
	) {
		RegisterBuiltinTypeBoundFunction(
			commons.TypeQualifier(t),
			NewNativeFunctionValue(
				functionName,
				sema.SaturatingArithmeticTypeFunctionTypes[t],
				func(context *Context, _ []bbq.StaticType, args ...Value) Value {
					v, ok := args[ReceiverIndex].(interpreter.NumberValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					other, ok := args[TypeBoundFunctionArgumentOffset].(interpreter.NumberValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					return op(context, v, other)
				},
			),
		)
	}

	if t.SupportsSaturatingAdd() {
		register(
			sema.NumericTypeSaturatingAddFunctionName,
			func(context *Context, v, other interpreter.NumberValue) interpreter.NumberValue {
				return v.SaturatingPlus(
					context,
					other,
					EmptyLocationRange,
				)
			},
		)
	}

	if t.SupportsSaturatingSubtract() {
		register(
			sema.NumericTypeSaturatingSubtractFunctionName,
			func(context *Context, v, other interpreter.NumberValue) interpreter.NumberValue {
				return v.SaturatingMinus(
					context,
					other,
					EmptyLocationRange,
				)
			},
		)
	}

	if t.SupportsSaturatingMultiply() {
		register(
			sema.NumericTypeSaturatingMultiplyFunctionName,
			func(context *Context, v, other interpreter.NumberValue) interpreter.NumberValue {
				return v.SaturatingMul(
					context,
					other,
					EmptyLocationRange,
				)
			},
		)
	}

	if t.SupportsSaturatingDivide() {
		register(
			sema.NumericTypeSaturatingDivideFunctionName,
			func(context *Context, v, other interpreter.NumberValue) interpreter.NumberValue {
				return v.SaturatingDiv(
					context,
					other,
					EmptyLocationRange,
				)
			},
		)
	}
}

func newFromStringFunction(typedParser interpreter.TypedStringValueParser) *NativeFunctionValue {
	functionType := sema.FromStringFunctionType(typedParser.ReceiverType)
	parser := typedParser.Parser

	return NewNativeFunctionValue(
		sema.FromStringFunctionName,
		functionType,
		func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
			argument, ok := arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			return parser(context, argument.Str)
		},
	)
}

func newFromBigEndianBytesFunction(typedConverter interpreter.TypedBigEndianBytesConverter) *NativeFunctionValue {
	functionType := sema.FromBigEndianBytesFunctionType(typedConverter.ReceiverType)
	byteLength := typedConverter.ByteLength
	converter := typedConverter.Converter

	return NewNativeFunctionValue(
		sema.FromBigEndianBytesFunctionName,
		functionType,
		func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {

			argument, ok := arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			bytes, err := interpreter.ByteArrayValueToByteSlice(context, argument, EmptyLocationRange)
			if err != nil {
				return interpreter.Nil
			}

			// overflow
			if byteLength != 0 && uint(len(bytes)) > byteLength {
				return interpreter.Nil
			}

			return interpreter.NewSomeValueNonCopying(context, converter(context, bytes))
		},
	)
}
