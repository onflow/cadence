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

type NativeFunctionsProvider func() map[string]Value

func NativeFunctions() map[string]Value {
	funcs := make(map[string]Value, len(nativeFunctions))
	for name, value := range nativeFunctions { //nolint:maprange
		funcs[name] = value
	}
	return funcs
}

func RegisterFunction(functionValue NativeFunctionValue) {
	functionName := functionValue.Name
	_, ok := nativeFunctions[functionName]
	if ok {
		panic(errors.NewUnexpectedError("function already exists: %s", functionName))
	}

	nativeFunctions[functionName] = functionValue
}

func RegisterTypeBoundFunction(typeName string, functionValue NativeFunctionValue) {
	// +1 is for the receiver
	functionValue.ParameterCount++

	// Update the name of the function to be type-qualified
	qualifiedName := commons.TypeQualifiedName(typeName, functionValue.Name)
	functionValue.Name = qualifiedName

	RegisterFunction(functionValue)
}

func init() {
	RegisterFunction(
		NewNativeFunctionValue(
			commons.LogFunctionName,
			stdlib.LogFunctionType,
			func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
				return stdlib.Log(
					config,
					config,
					arguments[0],
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterFunction(
		NewNativeFunctionValue(
			commons.PanicFunctionName,
			stdlib.PanicFunctionType,
			func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
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
			func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
				address := arguments[0].(interpreter.AddressValue)
				return NewAccountReferenceValue(
					config,
					config.GetAccountHandler(),
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
			func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
				return interpreter.NewTypeValue(
					config.MemoryGauge,
					typeArguments[0],
				)
			},
		),
	)

	RegisterFunction(
		NewNativeFunctionValue(
			sema.ReferenceTypeFunctionName,
			sema.ReferenceTypeFunctionType,
			func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
				entitlementValues := arguments[0].(*interpreter.ArrayValue)
				typeValue := arguments[1].(interpreter.TypeValue)
				return interpreter.ConstructReferenceStaticType(
					config,
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
				func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
					return convert(
						config.MemoryGauge,
						arguments[0],
						EmptyLocationRange,
					)
				},
			),
		)
	}
}
