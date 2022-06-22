/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestRuntimeError(t *testing.T) {

	t.Parallel()

	t.Run("parse error", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

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
			"Execution failed:\n"+
				"error: unexpected token: identifier\n"+
				" --> 0100000000000000000000000000000000000000000000000000000000000000:1:0\n"+
				"  |\n"+
				"1 | X\n"+
				"  | ^\n",
		)
	})

	t.Run("checking error", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

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
				" --> 0100000000000000000000000000000000000000000000000000000000000000:1:0\n"+
				"  |\n"+
				"1 | fun test() {}\n"+
				"  | ^\n",
		)
	})

	t.Run("execution error", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
            pub fun main() {
                let a: UInt8 = 255
                let b: UInt8 = 1
                // overflow
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
			"Execution failed:\n"+
				"error: overflow\n"+
				" --> 0100000000000000000000000000000000000000000000000000000000000000:6:16\n"+
				"  |\n"+
				"6 |                 a + b\n"+
				"  |                 ^^^^^\n",
		)
	})

	t.Run("parse error in import", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

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

		runtime := newTestInterpreterRuntime()

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

		runtime := newTestInterpreterRuntime()

		importedScript := []byte(`
            pub fun add() {
                let a: UInt8 = 255
                let b: UInt8 = 1
                // overflow
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
			"Execution failed:\n"+
				" --> 0100000000000000000000000000000000000000000000000000000000000000:5:16\n"+
				"  |\n"+
				"5 |                 add()\n"+
				"  |                 ^^^^^\n"+
				"\n"+
				"error: overflow\n"+
				" --> imported:6:16\n"+
				"  |\n"+
				"6 |                 a + b\n"+
				"  |                 ^^^^^\n"+
				"",
		)
	})

	t.Run("nested errors", func(t *testing.T) {

		// Test error pretty printing for the case where a program has errors,
		// but also imports a program that has errors.
		//
		// The location of the nested errors should not effect the location of the outer errors.

		id, err := hex.DecodeString("57717cc72f97494ac90441790352a07b999a39526819e638b5d367e62e43c37a")
		require.NoError(t, err)

		var location common.TransactionLocation
		copy(location[:], id)

		codes := map[common.LocationID]string{
			location.ID(): `
              // import program that has errors
              import A from 0x1
            `,
			common.AddressLocation{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "A",
			}.ID(): `
              // import program that has errors
              import B from 0x2

              // program itself has more errors:

              // invalid top-level declaration
              pub fun foo() {
                  // invalid reference to undeclared variable
                  Y
              }
            `,
			common.AddressLocation{
				Address: common.MustBytesToAddress([]byte{0x2}),
				Name:    "B",
			}.ID(): `
              // invalid top-level declaration
              pub fun bar() {
                  // invalid reference to undeclared variable
                  X
              }
            `,
		}

		runtimeInterface := &testRuntimeInterface{
			resolveLocation: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {
				for _, identifier := range identifiers {
					result = append(result, sema.ResolvedLocation{
						Location: common.AddressLocation{
							Address: location.(common.AddressLocation).Address,
							Name:    identifier.Identifier,
						},
						Identifiers: []ast.Identifier{
							identifier,
						},
					})
				}
				return
			},
			getAccountContractCode: func(address Address, name string) ([]byte, error) {
				location := common.AddressLocation{
					Name:    name,
					Address: address,
				}
				code := codes[location.ID()]
				return []byte(code), nil
			},
		}

		rt := newTestInterpreterRuntime()
		err = rt.ExecuteTransaction(
			Script{
				Source: []byte(codes[location.ID()]),
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(t, err,
			"Execution failed:\n"+
				"error: function declarations are not valid at the top-level\n"+
				" --> 0000000000000002.B:3:22\n"+
				"  |\n"+
				"3 |               pub fun bar() {\n"+
				"  |                       ^^^\n"+
				"\n"+
				"error: cannot find variable in this scope: `X`\n"+
				" --> 0000000000000002.B:5:18\n"+
				"  |\n"+
				"5 |                   X\n"+
				"  |                   ^ not found in this scope\n"+
				"\n"+
				"error: function declarations are not valid at the top-level\n"+
				" --> 0000000000000001.A:8:22\n"+
				"  |\n"+
				"8 |               pub fun foo() {\n"+
				"  |                       ^^^\n"+
				"\n"+
				"error: cannot find variable in this scope: `Y`\n"+
				"  --> 0000000000000001.A:10:18\n"+
				"   |\n"+
				"10 |                   Y\n"+
				"   |                   ^ not found in this scope\n",
		)

	})
}
