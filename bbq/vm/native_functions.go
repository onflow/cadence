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
	"fmt"

	"github.com/onflow/cadence/bbq/commons"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/stdlib"
)

var nativeFunctions = map[string]Value{}

// BuiltInLocation is the location of built-in constructs.
// It's always nil.
var BuiltInLocation common.Location = nil

func NativeFunctions() map[string]Value {
	funcs := make(map[string]Value, len(nativeFunctions))
	for name, value := range nativeFunctions {
		funcs[name] = value
	}
	return funcs
}

func RegisterFunction(functionName string, functionValue NativeFunctionValue) {
	functionValue.Name = functionName
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
		Function: func(config *Config, typeArguments []StaticType, arguments ...Value) Value {
			// TODO: Properly implement
			fmt.Println(arguments[0].String())
			return VoidValue{}
		},
	})

	RegisterFunction(commons.PanicFunctionName, NativeFunctionValue{
		ParameterCount: len(stdlib.PanicFunctionType.Parameters),
		Function: func(config *Config, typeArguments []StaticType, arguments ...Value) Value {
			// TODO: Properly implement
			messageValue, ok := arguments[0].(StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			panic(stdlib.PanicError{
				Message: string(messageValue.Str),
				// TODO: pass location
			})
		},
	})

	RegisterFunction(commons.GetAccountFunctionName, NativeFunctionValue{
		ParameterCount: len(stdlib.PanicFunctionType.Parameters),
		Function: func(config *Config, typeArguments []StaticType, arguments ...Value) Value {
			address := arguments[0].(AddressValue)
			return NewAccountReferenceValue(config, common.Address(address))
		},
	})
}
