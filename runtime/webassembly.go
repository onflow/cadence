/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package runtime

import (
	"fmt"

	"github.com/bytecodealliance/wasmtime-go/v12"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

type WasmtimeWebAssemblyModule struct {
	Module *wasmtime.Module
	Store  *wasmtime.Store
}

func NewWasmtimeWebAssemblyModule(bytes []byte) (stdlib.WebAssemblyModule, error) {
	config := wasmtime.NewConfig()
	config.SetConsumeFuel(true)

	engine := wasmtime.NewEngineWithConfig(config)
	store := wasmtime.NewStore(engine)

	module, err := wasmtime.NewModule(engine, bytes)
	if err != nil {
		return nil, interpreter.WebAssemblyNewModuleError{}
	}

	return WasmtimeWebAssemblyModule{
		Store:  store,
		Module: module,
	}, nil
}

var _ stdlib.WebAssemblyModule = WasmtimeWebAssemblyModule{}

func (m WasmtimeWebAssemblyModule) InstantiateWebAssemblyModule(_ common.MemoryGauge) (stdlib.WebAssemblyInstance, error) {
	instance, err := wasmtime.NewInstance(m.Store, m.Module, nil)
	if err != nil {
		return nil, interpreter.WebAssemblyNewInstanceError{}
	}
	return WasmtimeWebAssemblyInstance{
		Instance: instance,
		Store:    m.Store,
	}, nil
}

type WasmtimeWebAssemblyInstance struct {
	Instance *wasmtime.Instance
	Store    *wasmtime.Store
}

var _ stdlib.WebAssemblyInstance = WasmtimeWebAssemblyInstance{}

func wasmtimeValKindToSemaType(valKind wasmtime.ValKind) sema.Type {
	switch valKind {
	case wasmtime.KindI32:
		return sema.Int32Type

	case wasmtime.KindI64:
		return sema.Int64Type

	default:
		return nil
	}
}

func (i WasmtimeWebAssemblyInstance) GetExport(gauge common.MemoryGauge, name string) (*stdlib.WebAssemblyExport, error) {
	extern := i.Instance.GetExport(i.Store, name)
	if extern == nil {
		return nil, nil
	}

	if function := extern.Func(); function != nil {
		return newWasmtimeFunctionWebAssemblyExport(gauge, function, i.Store)
	}

	return nil, fmt.Errorf("unsupported export")
}

func newWasmtimeFunctionWebAssemblyExport(
	gauge common.MemoryGauge,
	function *wasmtime.Func,
	store *wasmtime.Store,
) (
	*stdlib.WebAssemblyExport,
	error,
) {
	funcType := function.Type(store)

	functionType := &sema.FunctionType{}

	// Parameters

	for _, paramType := range funcType.Params() {
		paramKind := paramType.Kind()
		parameterType := wasmtimeValKindToSemaType(paramKind)
		if parameterType == nil {
			return nil, fmt.Errorf(
				"unsupported export: function with unsupported parameter type '%s'",
				paramKind,
			)
		}
		functionType.Parameters = append(
			functionType.Parameters,
			sema.Parameter{
				TypeAnnotation: sema.NewTypeAnnotation(parameterType),
			},
		)
	}

	// Result

	results := funcType.Results()
	switch len(results) {
	case 0:
		functionType.ReturnTypeAnnotation = sema.VoidTypeAnnotation

	case 1:
		result := results[0]
		resultKind := result.Kind()
		returnType := wasmtimeValKindToSemaType(resultKind)
		if returnType == nil {
			return nil, fmt.Errorf(
				"unsupported export: function with unsupported result type '%s'",
				resultKind,
			)
		}
		functionType.ReturnTypeAnnotation = sema.NewTypeAnnotation(returnType)

	default:
		return nil, fmt.Errorf("unsupported export: function has more than one result")
	}

	hostFunctionValue := interpreter.NewHostFunctionValue(
		gauge,
		functionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			arguments := invocation.Arguments
			inter := invocation.Interpreter

			// Convert the arguments

			convertedArguments := make([]any, 0, len(arguments))

			for i, argument := range arguments {
				ty := functionType.Parameters[i].TypeAnnotation.Type

				var convertedArgument any

				switch ty {
				case sema.Int32Type:
					convertedArgument = int32(argument.(interpreter.Int32Value))

				case sema.Int64Type:
					convertedArgument = int64(argument.(interpreter.Int64Value))

				default:
					panic(errors.NewUnreachableError())
				}

				convertedArguments = append(convertedArguments, convertedArgument)
			}

			// Call the function, with metering

			fuelConsumedBefore, _ := store.FuelConsumed()

			// TODO: get remaining computation and convert to fuel.
			//   needs e.g. invocation.Interpreter.RemainingComputation()
			var todoAvailableFuel uint64 = 1000
			err := store.AddFuel(todoAvailableFuel)
			if err != nil {
				panic(interpreter.WebAssemblyStoreAddFuelError{
					TodoAvailableFuel: todoAvailableFuel,
				})
			}

			result, err := function.Call(store, convertedArguments...)
			if err != nil {

				panic(interpreter.WebAssemblyfunctionCallError{})
			}

			fuelConsumedAfter, _ := store.FuelConsumed()
			fuelDelta := fuelConsumedAfter - fuelConsumedBefore

			inter.ReportComputation(common.ComputationKindWebAssemblyFuel, uint(fuelDelta))

			remainingFuel, err := store.ConsumeFuel(0)
			if err != nil {
				panic(interpreter.WebAssemblystoreConsumeFuel{
					RemainingFuel: remainingFuel,
				})
			}

			remainingFuel, err = store.ConsumeFuel(remainingFuel)
			if err != nil {
				panic(interpreter.WebAssemblystoreConsumeFuel{
					RemainingFuel: remainingFuel,
				})
			}

			if remainingFuel != 0 {
				panic(errors.NewUnreachableError())
			}

			// Return the result

			switch result := result.(type) {
			case int32:
				return interpreter.Int32Value(result)

			case int64:
				return interpreter.Int64Value(result)

			default:
				panic(errors.NewUnreachableError())

			}
		},
	)

	return &stdlib.WebAssemblyExport{
		Type:  functionType,
		Value: hostFunctionValue,
	}, nil
}
