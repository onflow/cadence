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

func RegisterFunction(functionName string, functionValue NativeFunctionValue) {
	functionValue.Name = functionName

	_, ok := nativeFunctions[functionName]
	if ok {
		panic(errors.NewUnexpectedError("function already exists: %s", functionName))
	}

	nativeFunctions[functionName] = functionValue
}

func RegisterTypeBoundFunction(typeName, functionName string, functionValue NativeFunctionValue) {
	// +1 is for the receiver
	functionValue.ParameterCount++
	qualifiedName := commons.TypeQualifiedName(typeName, functionName)
	RegisterFunction(qualifiedName, functionValue)
}

func init() {
	RegisterFunction(commons.LogFunctionName, NativeFunctionValue{
		ParameterCount: len(stdlib.LogFunctionType.Parameters),
		Function: func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
			return stdlib.Log(
				config,
				config,
				arguments[0],
				EmptyLocationRange,
			)
		},
	})

	RegisterFunction(commons.PanicFunctionName, NativeFunctionValue{
		ParameterCount: len(stdlib.PanicFunctionType.Parameters),
		Function: func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
			return stdlib.PanicWithError(
				arguments[0],
				EmptyLocationRange,
			)
		},
	})

	RegisterFunction(commons.GetAccountFunctionName, NativeFunctionValue{
		ParameterCount: len(stdlib.GetAccountFunctionType.Parameters),
		Function: func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
			address := arguments[0].(interpreter.AddressValue)
			return NewAccountReferenceValue(
				config,
				config.GetAccountHandler(),
				common.Address(address),
			)
		},
	})

	// Type constructors
	// TODO: add the remaining type constructor functions

	RegisterFunction(sema.MetaTypeName, NativeFunctionValue{
		ParameterCount: len(sema.MetaTypeFunctionType.Parameters),
		Function: func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
			return interpreter.NewTypeValue(
				config.MemoryGauge,
				typeArguments[0],
			)
		},
	})

	// Value conversion functions
	for _, declaration := range interpreter.ConverterDeclarations {
		// NOTE: declare in loop, as captured in closure below
		convert := declaration.Convert

		RegisterFunction(declaration.Name, NativeFunctionValue{
			ParameterCount: len(declaration.FunctionType.Parameters),
			Function: func(config *Config, typeArguments []bbq.StaticType, arguments ...Value) Value {
				return convert(
					config.MemoryGauge,
					arguments[0],
					EmptyLocationRange,
				)
			},
		})
	}
}
