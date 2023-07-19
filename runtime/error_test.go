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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
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
            access(all) fun main() {
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

	t.Run("execution error with position", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
			access(all) fun main() {
				let x: AnyStruct? = nil
				let y = x!
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
				"error: unexpectedly found nil while forcing an Optional value\n"+
				" --> 0100000000000000000000000000000000000000000000000000000000000000:4:12\n"+
				"  |\n"+
				"4 | 				let y = x!\n"+
				"  | 				        ^^\n",
		)
	})

	t.Run("execution multiline nested error", func(t *testing.T) {

		t.Parallel()

		runtime := newTestInterpreterRuntime()

		script := []byte(`
			access(all) resource Resource {
				init(s:String){
					panic("42")
				}
			}
		
			access(all) fun createResource(): @Resource{
				return <- create Resource(
					s: "argument"
				)
			}
			
			access(all) fun main() {
				destroy createResource()
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

		require.EqualError(t, err,
			"Execution failed:\n"+
				"  --> 0100000000000000000000000000000000000000000000000000000000000000:15:12\n"+
				"   |\n"+
				"15 | 				destroy createResource()\n"+
				"   | 				        ^^^^^^^^^^^^^^^^\n"+
				"\n"+
				"  --> 0100000000000000000000000000000000000000000000000000000000000000:9:21\n"+
				"   |\n"+
				" 9 | 				return <- create Resource(\n"+
				"10 | 					s: \"argument\"\n"+
				"11 | 				)\n"+
				"   | 				^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"+
				"\n"+
				"error: panic: 42\n"+
				" --> 0100000000000000000000000000000000000000000000000000000000000000:4:5\n"+
				"  |\n"+
				"4 | 					panic(\"42\")\n"+
				"  | 					^^^^^^^^^^^\n",
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
            access(all) fun add() {
                let a: UInt8 = 255
                let b: UInt8 = 1
                // overflow
                a + b
            }
        `)

		script := []byte(`
            import add from "imported"

            access(all) fun main() {
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

		codes := map[Location]string{
			location: `
              // import program that has errors
              import A from 0x1
            `,
			common.AddressLocation{
				Address: common.MustBytesToAddress([]byte{0x1}),
				Name:    "A",
			}: `
              // import program that has errors
              import B from 0x2

              // program itself has more errors:

              // invalid top-level declaration
              access(all) fun foo() {
                  // invalid reference to undeclared variable
                  Y
              }
            `,
			common.AddressLocation{
				Address: common.MustBytesToAddress([]byte{0x2}),
				Name:    "B",
			}: `
              // invalid top-level declaration
              access(all) fun bar() {
                  // invalid reference to undeclared variable
                  X
              }
            `,
		}

		runtimeInterface := &testRuntimeInterface{
			resolveLocation: multipleIdentifierLocationResolver,
			getAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
				code := codes[location]
				return []byte(code), nil
			},
		}

		rt := newTestInterpreterRuntime()
		err = rt.ExecuteTransaction(
			Script{
				Source: []byte(codes[location]),
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.EqualError(t, err,
			"Execution failed:\n"+
				"error: function declarations are not valid at the top-level\n"+
				" --> 0000000000000002.B:3:30\n"+
				"  |\n"+
				"3 |               access(all) fun bar() {\n"+
				"  |                               ^^^\n"+
				"\n"+
				"error: cannot find variable in this scope: `X`\n"+
				" --> 0000000000000002.B:5:18\n"+
				"  |\n"+
				"5 |                   X\n"+
				"  |                   ^ not found in this scope\n"+
				"\n"+
				"error: function declarations are not valid at the top-level\n"+
				" --> 0000000000000001.A:8:30\n"+
				"  |\n"+
				"8 |               access(all) fun foo() {\n"+
				"  |                               ^^^\n"+
				"\n"+
				"error: cannot find variable in this scope: `Y`\n"+
				"  --> 0000000000000001.A:10:18\n"+
				"   |\n"+
				"10 |                   Y\n"+
				"   |                   ^ not found in this scope\n",
		)

	})
}

func TestRuntimeDefaultFunctionConflictPrintingError(t *testing.T) {
	t.Parallel()

	runtime := newTestInterpreterRuntime()

	makeDeployTransaction := func(name, code string) []byte {
		return []byte(fmt.Sprintf(
			`
              transaction {
                prepare(signer: AuthAccount) {
                  let acct = AuthAccount(payer: signer)
                  acct.contracts.add(name: "%s", code: "%s".decodeHex())
                }
              }
            `,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	contractInterfaceCode := `
      access(all) contract TestInterfaces {

          access(all) resource interface A {
              access(all) fun foo() {
                  let x = 3
              }
          }

		  access(all) resource interface B {
			access(all) fun foo() 
		}
      }
    `

	contractCode := `
      import TestInterfaces from 0x2
      access(all) contract TestContract {
          access(all) resource R: TestInterfaces.A, TestInterfaces.B {}
		  // fill space
		  // fill space
		  // fill space
		  // fill space
		  // fill space
		  // fill space
		  // filling lots of space
		  // filling lots of space
		  // filling lots of space
      }
    `

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	var nextAccount byte = 0x2

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		createAccount: func(payer Address) (address Address, err error) {
			result := interpreter.NewUnmeteredAddressValueFromBytes([]byte{nextAccount})
			nextAccount++
			return result.ToAddress(), nil
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{0x1}}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	deployTransaction := makeDeployTransaction("TestInterfaces", contractInterfaceCode)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	deployTransaction = makeDeployTransaction("TestContract", contractCode)
	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}
