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

package stdlib

import (
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

func NewUnifiedStandardLibraryStaticFunction(
	name string,
	functionType *sema.FunctionType,
	docString string,
	function interpreter.UnifiedNativeFunction,
	isVM bool,
) StandardLibraryValue {
	parameters := functionType.Parameters

	argumentLabels := make([]string, len(parameters))

	for i, parameter := range parameters {
		argumentLabels[i] = parameter.EffectiveArgumentLabel()
	}

	var functionValue interpreter.Value
	if isVM {
		functionValue = vm.NewUnifiedNativeFunctionValue(name, functionType, function)
	} else {
		functionValue = interpreter.NewUnmeteredUnifiedStaticHostFunctionValue(functionType, function)
	}

	return StandardLibraryValue{
		Name:           name,
		Type:           functionType,
		DocString:      docString,
		Value:          functionValue,
		ArgumentLabels: argumentLabels,
		Kind:           common.DeclarationKindFunction,
	}
}

// These functions are helpers for testing.
// NewInterpreterStandardLibraryStaticFunction should only be used for creating static functions.
func NewInterpreterStandardLibraryStaticFunction(
	name string,
	functionType *sema.FunctionType,
	docString string,
	function interpreter.HostFunction,
) StandardLibraryValue {

	parameters := functionType.Parameters

	argumentLabels := make([]string, len(parameters))

	for i, parameter := range parameters {
		argumentLabels[i] = parameter.EffectiveArgumentLabel()
	}

	functionValue := interpreter.NewUnmeteredStaticHostFunctionValue(functionType, function)

	return StandardLibraryValue{
		Name:           name,
		Type:           functionType,
		DocString:      docString,
		Value:          functionValue,
		ArgumentLabels: argumentLabels,
		Kind:           common.DeclarationKindFunction,
	}
}

// NewVMStandardLibraryStaticFunction should only be used for creating static functions.
func NewVMStandardLibraryStaticFunction(
	name string,
	functionType *sema.FunctionType,
	docString string,
	function vm.NativeFunction,
) StandardLibraryValue {

	parameters := functionType.Parameters

	argumentLabels := make([]string, len(parameters))

	for i, parameter := range parameters {
		argumentLabels[i] = parameter.EffectiveArgumentLabel()
	}

	functionValue := vm.NewNativeFunctionValue(name, functionType, function)

	return StandardLibraryValue{
		Name:           name,
		Type:           functionType,
		DocString:      docString,
		Value:          functionValue,
		ArgumentLabels: argumentLabels,
		Kind:           common.DeclarationKindFunction,
	}
}
