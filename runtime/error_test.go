/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
)

func TestRuntimeError(t *testing.T) {

	t.Parallel()

	t.Run("parse error", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		script := []byte(`X`)

		runtimeInterface := &testRuntimeInterface{}

		location := common.ScriptLocation{0x1}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(
			t,
			err,
			"Execution failed:\nerror: unexpected token: identifier\n"+
				" --> 01:1:0\n"+
				"  |\n"+
				"1 | X\n"+
				"  | ^\n",
		)
	})

	t.Run("checking error", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		script := []byte(`fun test() {}`)

		runtimeInterface := &testRuntimeInterface{}

		location := common.ScriptLocation{0x1}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(
			t,
			err,
			"Execution failed:\n"+
				"error: missing access modifier for function\n"+
				" --> 01:1:0\n"+
				"  |\n"+
				"1 | fun test() {}\n"+
				"  | ^\n",
		)
	})

	t.Run("execution error", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		script := []byte(`
            pub fun main() {
                let a: UInt8 = 255
                let b: UInt8 = 1
                a + b
            }
        `)

		runtimeInterface := &testRuntimeInterface{}

		location := common.ScriptLocation{0x1}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(
			t,
			err,
			"Execution failed:\nerror: overflow\n"+
				" --> 01:5:16\n"+
				"  |\n"+
				"5 |                 a + b\n"+
				"  |                 ^^^^^\n",
		)
	})

	t.Run("parse error in import", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		importedScript := []byte(`X`)

		script := []byte(`import "imported"`)

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				switch location {
				case common.StringLocation("imported"):
					return importedScript, nil
				default:
					return nil, fmt.Errorf("unknown import location: %s", location)
				}
			},
		}

		location := common.ScriptLocation{0x1}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(
			t,
			err,
			"Execution failed:\nerror: unexpected token: identifier\n"+
				" --> imported:1:0\n"+
				"  |\n"+
				"1 | X\n"+
				"  | ^\n",
		)
	})

	t.Run("checking error in import", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		importedScript := []byte(`fun test() {}`)

		script := []byte(`import "imported"`)

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				switch location {
				case common.StringLocation("imported"):
					return importedScript, nil
				default:
					return nil, fmt.Errorf("unknown import location: %s", location)
				}
			},
		}

		location := common.ScriptLocation{0x1}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(
			t,
			err,
			"Execution failed:\n"+
				"error: missing access modifier for function\n"+
				" --> imported:1:0\n"+
				"  |\n"+
				"1 | fun test() {}\n"+
				"  | ^\n",
		)
	})

	t.Run("execution error in import", func(t *testing.T) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		importedScript := []byte(`
            pub fun add() {
                let a: UInt8 = 255
                let b: UInt8 = 1
                a + b
            }
        `)

		script := []byte(`
            import add from "imported"

            pub fun main() {
                add()
            }
        `)

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				switch location {
				case common.StringLocation("imported"):
					return importedScript, nil
				default:
					return nil, fmt.Errorf("unknown import location: %s", location)
				}
			},
		}

		location := common.ScriptLocation{0x1}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(
			t,
			err,
			"Execution failed:\nerror: overflow\n"+
				" --> imported:5:16\n"+
				"  |\n"+
				"5 |                 a + b\n"+
				"  |                 ^^^^^\n",
		)
	})

}
