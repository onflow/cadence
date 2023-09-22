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

package runtime_test

import (
	"fmt"
	"testing"

	"github.com/bytecodealliance/wasmtime-go/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
)

type WasmtimeModule struct {
	Module *wasmtime.Module
	Store  *wasmtime.Store
}

var _ stdlib.WebAssemblyModule = WasmtimeModule{}

func (m WasmtimeModule) InstantiateWebAssemblyModule(_ common.MemoryGauge) (stdlib.WebAssemblyInstance, error) {
	instance, err := wasmtime.NewInstance(m.Store, m.Module, nil)
	if err != nil {
		return nil, err
	}
	return WasmtimeInstance{
		Instance: instance,
		Store:    m.Store,
	}, nil
}

type WasmtimeInstance struct {
	Instance *wasmtime.Instance
	Store    *wasmtime.Store
}

var _ stdlib.WebAssemblyInstance = WasmtimeInstance{}

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

func (i WasmtimeInstance) GetExport(gauge common.MemoryGauge, name string) (*stdlib.WebAssemblyExport, error) {
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
		break

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

			result, err := function.Call(store, convertedArguments...)
			if err != nil {
				panic(err)
			}

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

func TestRuntimeWebAssembly(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	// A simple program which exports a function `add` with type `(i32, i32) -> i32`,
	// which sums the arguments and returns the result
	addProgram := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x07, 0x01, 0x60,
		0x02, 0x7f, 0x7f, 0x01, 0x7f, 0x03, 0x02, 0x01, 0x00, 0x07, 0x07, 0x01,
		0x03, 0x61, 0x64, 0x64, 0x00, 0x00, 0x0a, 0x09, 0x01, 0x07, 0x00, 0x20,
		0x00, 0x20, 0x01, 0x6a, 0x0b,
	}

	// language=cadence
	script := []byte(`
      access(all)
      fun main(program: [UInt8], a: Int32, b: Int32): Int32 {
          let instance = WebAssembly.compileAndInstantiate(bytes: program).instance
          let add = instance.getExport<fun(Int32, Int32): Int32>(name: "add")
          return add(a, b)
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnCompileWebAssembly: func(bytes []byte) (stdlib.WebAssemblyModule, error) {
			store := wasmtime.NewStore(wasmtime.NewEngine())
			module, err := wasmtime.NewModule(store.Engine, bytes)
			if err != nil {
				return nil, err
			}

			return WasmtimeModule{
				Store:  store,
				Module: module,
			}, nil
		},
		OnDecodeArgument: func(b []byte, _ cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	result, err := runtime.ExecuteScript(
		Script{
			Source: script,
			Arguments: encodeArgs(
				newBytesValue(addProgram),
				cadence.Int32(1),
				cadence.Int32(2),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	require.NoError(t, err)
	assert.Equal(t,
		cadence.Int32(3),
		result,
	)
}
