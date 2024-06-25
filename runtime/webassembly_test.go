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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"

	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
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
