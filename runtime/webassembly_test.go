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

package runtime_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestRuntimeWebAssemblyAdd(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		WebAssemblyEnabled: true,
	})

	// A simple program which exports a function `add` with type `(i32, i32) -> i32`,
	// which sums the arguments and returns the result:
	//
	//  (module
	//    (type (;0;) (func (param i32 i32) (result i32)))
	//    (func (;0;) (type 0) (param i32 i32) (result i32)
	//      local.get 0
	//      local.get 1
	//      i32.add)
	//    (export "add" (func 0)))
	//
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
          return add(a, b) + add(a, b)
      }
    `)

	var webAssemblyFuelComputationMeterings []uint

	runtimeInterface := &TestRuntimeInterface{
		Storage:              NewTestLedger(nil, nil),
		OnCompileWebAssembly: NewWasmtimeWebAssemblyModule,
		OnDecodeArgument: func(b []byte, _ cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
		OnMeterComputation: func(compKind common.ComputationKind, intensity uint) error {
			if compKind != common.ComputationKindWebAssemblyFuel {
				return nil
			}

			webAssemblyFuelComputationMeterings = append(
				webAssemblyFuelComputationMeterings,
				intensity,
			)

			return nil
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
		cadence.Int32(6),
		result,
	)
	assert.Equal(t,
		[]uint{4, 4},
		webAssemblyFuelComputationMeterings,
	)
}

func TestRuntimeWebAssemblyDisabled(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		WebAssemblyEnabled: false,
	})

	// language=cadence
	script := []byte(`
      access(all)
      fun main() {
          WebAssembly
      }
    `)

	runtimeInterface := &TestRuntimeInterface{}

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)
}

func TestRuntimeWebAssemblyLoop(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		WebAssemblyEnabled: true,
	})

	// A simple program which exports a function that loops forever:
	//
	//  (module
	//    (func (loop $loop (br 0)))
	//    (export "loop" (func 0)))
	//
	addProgram := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x04, 0x01, 0x60,
		0x00, 0x00, 0x03, 0x02, 0x01, 0x00, 0x07, 0x08, 0x01, 0x04, 0x6c, 0x6f,
		0x6f, 0x70, 0x00, 0x00, 0x0a, 0x09, 0x01, 0x07, 0x00, 0x03, 0x40, 0x0c,
		0x00, 0x0b, 0x0b,
	}

	// language=cadence
	script := []byte(`
      access(all)
      fun main(program: [UInt8]) {
          let instance = WebAssembly.compileAndInstantiate(bytes: program).instance
          let loop = instance.getExport<fun(): Void>(name: "loop")
          loop()
      }
    `)

	var webAssemblyFuelComputationMeterings []uint

	runtimeInterface := &TestRuntimeInterface{
		Storage:              NewTestLedger(nil, nil),
		OnCompileWebAssembly: NewWasmtimeWebAssemblyModule,
		OnDecodeArgument: func(b []byte, _ cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
		OnMeterComputation: func(compKind common.ComputationKind, intensity uint) error {
			if compKind != common.ComputationKindWebAssemblyFuel {
				return nil
			}

			webAssemblyFuelComputationMeterings = append(
				webAssemblyFuelComputationMeterings,
				intensity,
			)

			return nil
		},
		OnComputationRemaining: func(kind common.ComputationKind) uint {
			if kind != common.ComputationKindWebAssemblyFuel {
				return 0
			}

			return 1000
		},
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
			Arguments: encodeArgs(
				newBytesValue(addProgram),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	require.Error(t, err)
	require.ErrorAs(t, err, &stdlib.WebAssemblyTrapError{})

	assert.Equal(t,
		// TODO: adjust, currently todoAvailableFuel
		[]uint{1000},
		webAssemblyFuelComputationMeterings,
	)
}

func TestRuntimeWebAssemblyInfiniteLoopAtStart(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		WebAssemblyEnabled: true,
	})

	// A simple program which loops forever, on start/load:
	//
	//  (module
	//    (func (loop $loop (br 0)))
	//    (start 0))
	//
	addProgram := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x04, 0x01, 0x60,
		0x00, 0x00, 0x03, 0x02, 0x01, 0x00, 0x08, 0x01, 0x00, 0x0a, 0x09, 0x01,
		0x07, 0x00, 0x03, 0x40, 0x0c, 0x00, 0x0b, 0x0b,
	}

	// language=cadence
	script := []byte(`
      access(all)
      fun main(program: [UInt8]) {
          let instance = WebAssembly.compileAndInstantiate(bytes: program).instance
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		Storage:              NewTestLedger(nil, nil),
		OnCompileWebAssembly: NewWasmtimeWebAssemblyModule,
		OnDecodeArgument: func(b []byte, _ cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
		OnComputationRemaining: func(kind common.ComputationKind) uint {
			if kind != common.ComputationKindWebAssemblyFuel {
				return 0
			}

			return 1000
		},
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
			Arguments: encodeArgs(
				newBytesValue(addProgram),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)
	require.ErrorAs(t, err, &stdlib.WebAssemblyTrapError{})
}

func TestRuntimeWebAssemblyNonFunctionExport(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		WebAssemblyEnabled: true,
	})

	// A simple program which exports a memory `add`.
	//
	//  (module
	//    (memory (export "add") 1 4)
	//    (data (i32.const 0x1) "\01\02\03")
	//  )
	//
	addProgram := []byte{
		0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x05, 0x04, 0x01, 0x01,
		0x01, 0x04, 0x07, 0x07, 0x01, 0x03, 0x61, 0x64, 0x64, 0x02, 0x00, 0x0b,
		0x09, 0x01, 0x00, 0x41, 0x01, 0x0b, 0x03, 0x01, 0x02, 0x03,
	}

	// language=cadence
	script := []byte(`
      access(all)
      fun main(program: [UInt8]) {
          let instance = WebAssembly.compileAndInstantiate(bytes: program).instance
          instance.getExport<fun(): Int32>(name: "add")
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		Storage:              NewTestLedger(nil, nil),
		OnCompileWebAssembly: NewWasmtimeWebAssemblyModule,
		OnDecodeArgument: func(b []byte, _ cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
			Arguments: encodeArgs(
				newBytesValue(addProgram),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)
	require.ErrorAs(t, err, &stdlib.WebAssemblyNonFunctionExportError{})
}

func TestRuntimeWebAssemblyInvalidModule(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		WebAssemblyEnabled: true,
	})

	program := []byte{0xFF}

	// language=cadence
	script := []byte(`
      access(all)
      fun main(program: [UInt8]) {
          WebAssembly.compileAndInstantiate(bytes: program).instance
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		Storage:              NewTestLedger(nil, nil),
		OnCompileWebAssembly: NewWasmtimeWebAssemblyModule,
		OnDecodeArgument: func(b []byte, _ cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
			Arguments: encodeArgs(
				newBytesValue(program),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)
	require.ErrorAs(t, err, &stdlib.WebAssemblyCompilationError{})
}
