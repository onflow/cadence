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
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type BuiltinGlobalsProvider func(location common.Location) *activations.Activation[Variable]

var defaultBuiltinGlobals = activations.NewActivation[Variable](nil, nil)

func DefaultBuiltinGlobals() *activations.Activation[Variable] {
	return defaultBuiltinGlobals
}

func registerBuiltinFunction(functionValue *NativeFunctionValue) {
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

func registerBuiltinTypeBoundFunction(typeName string, functionValue *NativeFunctionValue) {
	// Update the name of the function to be type-qualified
	qualifiedName := commons.QualifiedName(typeName, functionValue.Name)
	functionValue.Name = qualifiedName

	registerBuiltinFunction(functionValue)
}

func registerBuiltinTypeBoundCommonFunction(typeName string, functionValue *NativeFunctionValue) {
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

var failConditionFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "message",
			TypeAnnotation: sema.StringTypeAnnotation,
		},
	},
	sema.NeverTypeAnnotation,
)

func init() {

	// Pre/post condition failure functions

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			commons.FailPreConditionFunctionName,
			failConditionFunctionType,
			func(_ interpreter.UnifiedFunctionContext, _ interpreter.LocationRange, _ interpreter.TypeParameterGetter, _ interpreter.Value, args ...interpreter.Value) interpreter.Value {
				messageValue := args[0].(*interpreter.StringValue)
				panic(&interpreter.ConditionError{
					Message:       messageValue.Str,
					ConditionKind: ast.ConditionKindPre,
				})
			},
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			commons.FailPostConditionFunctionName,
			failConditionFunctionType,
			func(_ interpreter.UnifiedFunctionContext, _ interpreter.LocationRange, _ interpreter.TypeParameterGetter, _ interpreter.Value, args ...interpreter.Value) interpreter.Value {
				messageValue := args[0].(*interpreter.StringValue)
				panic(&interpreter.ConditionError{
					Message:       messageValue.Str,
					ConditionKind: ast.ConditionKindPost,
				})
			},
		),
	)

	// Type constructors

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.MetaTypeName,
			sema.MetaTypeFunctionType,
			interpreter.UnifiedMetaTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.OptionalTypeFunctionName,
			sema.OptionalTypeFunctionType,
			interpreter.UnifiedOptionalTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.VariableSizedArrayTypeFunctionName,
			sema.VariableSizedArrayTypeFunctionType,
			interpreter.UnifiedVariableSizedArrayTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.ConstantSizedArrayTypeFunctionName,
			sema.ConstantSizedArrayTypeFunctionType,
			interpreter.UnifiedConstantSizedArrayTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.DictionaryTypeFunctionName,
			sema.DictionaryTypeFunctionType,
			interpreter.UnifiedDictionaryTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.CompositeTypeFunctionName,
			sema.CompositeTypeFunctionType,
			interpreter.UnifiedCompositeTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.FunctionTypeFunctionName,
			sema.FunctionTypeFunctionType,
			interpreter.UnifiedFunctionTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.ReferenceTypeFunctionName,
			sema.ReferenceTypeFunctionType,
			interpreter.UnifiedReferenceTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.IntersectionTypeFunctionName,
			sema.IntersectionTypeFunctionType,
			interpreter.UnifiedIntersectionTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.CapabilityTypeFunctionName,
			sema.CapabilityTypeFunctionType,
			interpreter.UnifiedCapabilityTypeFunction,
		),
	)

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.InclusiveRangeTypeFunctionName,
			sema.InclusiveRangeTypeFunctionType,
			interpreter.UnifiedInclusiveRangeTypeFunction,
		),
	)

	// Value conversion functions
	for _, declaration := range interpreter.ConverterDeclarations {
		// NOTE: declare in loop, as captured in closure below
		convert := declaration.Convert

		functionType := sema.BaseValueActivation.Find(declaration.Name).Type.(*sema.FunctionType)

		function := NewUnifiedNativeFunctionValue(
			declaration.Name,
			functionType,
			interpreter.UnifiedConverterFunction(convert),
		)
		registerBuiltinFunction(function)

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
			registerBuiltinTypeBoundFunction(
				commons.TypeQualifier(stringValueParser.ReceiverType),
				newFromStringFunction(stringValueParser),
			)
		}

		if bigEndianBytesConverter, ok := interpreter.BigEndianBytesConverters[declaration.Name]; ok {
			registerBuiltinTypeBoundFunction(
				commons.TypeQualifier(bigEndianBytesConverter.ReceiverType),
				newFromBigEndianBytesFunction(bigEndianBytesConverter),
			)
		}
	}

	// Value constructors

	registerBuiltinFunction(
		NewUnifiedNativeFunctionValue(
			sema.StringType.String(),
			sema.StringFunctionType,
			interpreter.UnifiedStringFunction,
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

	for _, function := range CommonBuiltinTypeBoundFunctions {
		IndexedCommonBuiltinTypeBoundFunctions[function.Name] = function
	}
}

func registerBuiltinTypeBoundFunctions(
	typeQualifier string,
) {
	for _, boundFunction := range CommonBuiltinTypeBoundFunctions {
		registerBuiltinTypeBoundCommonFunction(
			typeQualifier,
			boundFunction,
		)
	}
}

// CommonBuiltinTypeBoundFunctions are the built-in functions that are common to all the types.
var CommonBuiltinTypeBoundFunctions = []*NativeFunctionValue{

	// `isInstance` function
	NewUnifiedNativeFunctionValue(
		sema.IsInstanceFunctionName,
		sema.IsInstanceFunctionType,
		interpreter.UnifiedIsInstanceFunction,
	),

	// `getType` function
	NewUnifiedNativeFunctionValue(
		sema.GetTypeFunctionName,
		sema.GetTypeFunctionType,
		interpreter.UnifiedGetTypeFunction,
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
	functionType := sema.SaturatingArithmeticTypeFunctionTypes[t]

	if t.SupportsSaturatingAdd() {
		registerBuiltinTypeBoundFunction(
			commons.TypeQualifier(t),
			NewUnifiedNativeFunctionValue(
				sema.NumericTypeSaturatingAddFunctionName,
				functionType,
				interpreter.UnifiedNumberSaturatingAddFunction,
			),
		)
	}

	if t.SupportsSaturatingSubtract() {
		registerBuiltinTypeBoundFunction(
			commons.TypeQualifier(t),
			NewUnifiedNativeFunctionValue(
				sema.NumericTypeSaturatingSubtractFunctionName,
				functionType,
				interpreter.UnifiedNumberSaturatingSubtractFunction,
			),
		)
	}

	if t.SupportsSaturatingMultiply() {
		registerBuiltinTypeBoundFunction(
			commons.TypeQualifier(t),
			NewUnifiedNativeFunctionValue(
				sema.NumericTypeSaturatingMultiplyFunctionName,
				functionType,
				interpreter.UnifiedNumberSaturatingMultiplyFunction,
			),
		)
	}

	if t.SupportsSaturatingDivide() {
		registerBuiltinTypeBoundFunction(
			commons.TypeQualifier(t),
			NewUnifiedNativeFunctionValue(
				sema.NumericTypeSaturatingDivideFunctionName,
				functionType,
				interpreter.UnifiedNumberSaturatingDivideFunction,
			),
		)
	}
}

func newFromStringFunction(typedParser interpreter.TypedStringValueParser) *NativeFunctionValue {
	functionType := sema.FromStringFunctionType(typedParser.ReceiverType)
	parser := typedParser.Parser

	return NewUnifiedNativeFunctionValue(
		sema.FromStringFunctionName,
		functionType,
		interpreter.UnifiedFromStringFunction(parser),
	)
}

func newFromBigEndianBytesFunction(typedConverter interpreter.TypedBigEndianBytesConverter) *NativeFunctionValue {
	functionType := sema.FromBigEndianBytesFunctionType(typedConverter.ReceiverType)
	byteLength := typedConverter.ByteLength
	converter := typedConverter.Converter

	return NewUnifiedNativeFunctionValue(
		sema.FromBigEndianBytesFunctionName,
		functionType,
		interpreter.UnifiedFromBigEndianBytesFunction(byteLength, converter),
	)
}
