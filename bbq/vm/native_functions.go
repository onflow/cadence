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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

const (
	receiverIndex                   = 0
	typeBoundFunctionArgumentOffset = 1
)

var nativeFunctions = map[string]Value{}

// BuiltInLocation is the location of built-in constructs.
// It's always nil.
var BuiltInLocation common.Location = nil

type NativeFunctionsProvider func() map[string]*Variable

func NativeFunctions() map[string]*Variable {
	funcs := make(map[string]*Variable, len(nativeFunctions))
	for name, value := range nativeFunctions { //nolint:maprange
		variable := &interpreter.SimpleVariable{}
		variable.InitializeWithValue(value)
		funcs[name] = variable
	}
	return funcs
}

func RegisterFunction(functionValue NativeFunctionValue) {
	functionName := functionValue.Name
	registerFunction(functionName, functionValue)
}

func registerFunction(functionName string, functionValue NativeFunctionValue) {
	_, ok := nativeFunctions[functionName]
	if ok {
		panic(errors.NewUnexpectedError("function already exists: %s", functionName))
	}

	nativeFunctions[functionName] = functionValue
}

func RegisterTypeBoundFunction(typeName string, functionValue NativeFunctionValue) {
	// Update the name of the function to be type-qualified
	qualifiedName := commons.QualifiedName(typeName, functionValue.Name)
	functionValue.Name = qualifiedName

	RegisterFunction(functionValue)
}

func RegisterTypeBoundCommonFunction(typeName string, functionValue NativeFunctionValue) {
	// Here the function value is common for many types.
	// Hence, do not update the function name to be type-qualified.
	// Only the key in the map is type-qualified.
	qualifiedName := commons.QualifiedName(typeName, functionValue.Name)
	registerFunction(qualifiedName, functionValue)
}

func init() {
	RegisterFunction(
		NewNativeFunctionValue(
			commons.LogFunctionName,
			stdlib.LogFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				return stdlib.Log(
					context,
					context,
					arguments[0],
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterFunction(
		NewNativeFunctionValue(
			commons.AssertFunctionName,
			stdlib.AssertFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				result, ok := arguments[0].(interpreter.BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				var message string
				if len(arguments) > 1 {
					messageValue, ok := arguments[1].(*interpreter.StringValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}
					message = messageValue.Str
				}

				return stdlib.Assert(
					result,
					message,
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterFunction(
		NewNativeFunctionValue(
			commons.PanicFunctionName,
			stdlib.PanicFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				return stdlib.PanicWithError(
					arguments[0],
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterFunction(
		NewNativeFunctionValue(
			commons.GetAccountFunctionName,
			stdlib.GetAccountFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				address := arguments[0].(interpreter.AddressValue)
				return NewAccountReferenceValue(
					context,
					context.GetAccountHandler(),
					common.Address(address),
				)
			},
		),
	)

	// Type constructors
	// TODO: add the remaining type constructor functions

	RegisterFunction(
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

	RegisterFunction(
		NewNativeFunctionValue(
			sema.ReferenceTypeFunctionName,
			sema.ReferenceTypeFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
				entitlementValues := arguments[0].(*interpreter.ArrayValue)
				typeValue := arguments[1].(interpreter.TypeValue)
				return interpreter.ConstructReferenceStaticType(
					context,
					entitlementValues,
					EmptyLocationRange,
					typeValue,
				)
			},
		),
	)

	// Value conversion functions
	for _, declaration := range interpreter.ConverterDeclarations {
		// NOTE: declare in loop, as captured in closure below
		convert := declaration.Convert

		RegisterFunction(
			NewNativeFunctionValue(
				declaration.Name,
				declaration.FunctionType,
				func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
					return convert(
						context.MemoryGauge,
						arguments[0],
						EmptyLocationRange,
					)
				},
			),
		)
	}

	// Value constructors

	RegisterFunction(
		NewNativeFunctionValue(
			sema.StringType.String(),
			sema.StringFunctionType,
			func(_ *Context, _ []bbq.StaticType, _ ...Value) Value {
				return interpreter.EmptyString
			},
		),
	)

	// Register type-bound functions that are common to many types.
	registerCommonBuiltinTypeBoundFunctions()

	registerAllSaturatingArithmeticFunctions()
}

func registerCommonBuiltinTypeBoundFunctions() {
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
		RegisterTypeBoundCommonFunction(
			typeQualifier,
			boundFunction,
		)
	}
}

// Built-in functions that are common to all the types.
var commonBuiltinTypeBoundFunctions = []NativeFunctionValue{

	// `isInstance` function
	NewNativeFunctionValue(
		sema.IsInstanceFunctionName,
		sema.IsInstanceFunctionType,
		func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
			value := arguments[receiverIndex]

			typeValue, ok := arguments[typeBoundFunctionArgumentOffset].(interpreter.TypeValue)
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
		func(context *Context, typeArguments []bbq.StaticType, arguments ...Value) Value {
			value := arguments[receiverIndex]
			return interpreter.ValueGetType(context, value)
		},
	),

	// TODO: add remaining functions
}

var IndexedCommonBuiltinTypeBoundFunctions = map[string]NativeFunctionValue{}

func registerAllSaturatingArithmeticFunctions() {
	for _, ty := range common.Concat(
		sema.AllUnsignedFixedPointTypes,
		sema.AllSignedFixedPointTypes,
		sema.AllUnsignedIntegerTypes,
		sema.AllSignedIntegerTypes,
	) {
		registerSaturatingArithmeticFunctions(ty.(sema.SaturatingArithmeticType))
	}
}

func registerSaturatingArithmeticFunctions(t sema.SaturatingArithmeticType) {

	register := func(
		functionName string,
		op func(context *Context, v, other interpreter.NumberValue) interpreter.NumberValue,
	) {
		RegisterTypeBoundFunction(
			commons.TypeQualifier(t),
			NewNativeFunctionValue(
				functionName,
				sema.SaturatingArithmeticTypeFunctionTypes[t],
				func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
					v, ok := args[receiverIndex].(interpreter.NumberValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					other, ok := args[typeBoundFunctionArgumentOffset].(interpreter.NumberValue)
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
