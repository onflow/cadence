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
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	runtimeErrors "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeImport(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	importedScript := []byte(`
      access(all) fun answer(): Int {
          return 42
      }
    `)

	script := []byte(`
      import "imported"

      access(all) fun main(): Int {
          let answer = answer()
          if answer != 42 {
            panic("?!")
          }
          return answer
        }
    `)

	var checkCount int

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		OnProgramChecked: func(location Location, duration time.Duration) {
			checkCount += 1
		},
	}

	nextScriptLocation := NewScriptLocationGenerator()

	const transactionCount = 10
	for i := 0; i < transactionCount; i++ {

		value, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, cadence.NewInt(42), value)
	}
	require.Equal(t, transactionCount+1, checkCount)
}

func TestRuntimeConcurrentImport(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	importedScript := []byte(`
      access(all) fun answer(): Int {
          return 42
      }
    `)

	script := []byte(`
      import "imported"

      access(all) fun main(): Int {
          let answer = answer()
          if answer != 42 {
            panic("?!")
          }
          return answer
        }
    `)

	var checkCount uint64

	var programs sync.Map

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		OnProgramChecked: func(location Location, duration time.Duration) {
			atomic.AddUint64(&checkCount, 1)
		},
		OnGetAndSetProgram: func(
			location Location,
			load func() (*interpreter.Program, error),
		) (
			program *interpreter.Program,
			err error,
		) {
			item, ok := programs.Load(location)
			if ok {
				program = item.(*interpreter.Program)
				return
			}

			program, err = load()

			// NOTE: important: still set empty program,
			// even if error occurred

			programs.Store(location, program)

			return
		},
	}

	nextScriptLocation := NewScriptLocationGenerator()

	var wg sync.WaitGroup
	const concurrency uint64 = 10
	for i := uint64(0); i < concurrency; i++ {

		location := nextScriptLocation()

		wg.Add(1)
		go func() {
			defer wg.Done()

			value, err := runtime.ExecuteScript(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  location,
				},
			)
			require.NoError(t, err)

			assert.Equal(t, cadence.NewInt(42), value)
		}()
	}
	wg.Wait()

	// TODO:
	//   Ideally we would expect the imported program only be checked once
	//   (`concurrency` transactions + 1 for the imported program),
	//   however, currently the imported program gets re-checked if it is currently being checked.
	//   This can probably be optimized by synchronizing the checking of a program using `sync`.
	//
	// require.Equal(t, concurrency+1, checkCount)
}

func TestRuntimeProgramSetAndGet(t *testing.T) {

	t.Parallel()

	programs := map[Location]*interpreter.Program{}
	programsHits := make(map[Location]bool)

	importedScript := []byte(`
      transaction {
          prepare() {}
          execute {}
      }
    `)
	importedScriptLocation := common.StringLocation("imported")
	scriptLocation := common.StringLocation("placeholder")

	runtime := NewTestInterpreterRuntime()
	runtimeInterface := &TestRuntimeInterface{
		OnGetAndSetProgram: func(
			location Location,
			load func() (*interpreter.Program, error),
		) (
			program *interpreter.Program,
			err error,
		) {
			var ok bool
			program, ok = programs[location]

			programsHits[location] = ok

			if ok {
				return
			}

			program, err = load()

			// NOTE: important: still set empty program,
			// even if error occurred

			programs[location] = program

			return
		},
		OnGetCode: func(location Location) ([]byte, error) {
			switch location {
			case importedScriptLocation:
				return importedScript, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
	}

	t.Run("empty programs, miss", func(t *testing.T) {

		script := []byte(`
          import "imported"

          transaction {
              prepare() {}
              execute {}
          }
        `)

		// Initial call, should parse script, store program.
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		// Program was added to stored programs.
		storedProgram, exists := programs[scriptLocation]
		assert.True(t, exists)
		assert.NotNil(t, storedProgram)

		// Script was not in stored programs.
		assert.False(t, programsHits[scriptLocation])
	})

	t.Run("program previously parsed, hit", func(t *testing.T) {

		script := []byte(`
          import "imported"

          transaction {
              prepare() {}
              execute {}
          }
        `)

		// Call a second time to hit stored programs.
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		assert.True(t, programsHits[scriptLocation])
	})

	t.Run("imported program previously parsed, hit", func(t *testing.T) {

		script := []byte(`
          import "imported"

          transaction {
              prepare() {}
              execute {}
          }
        `)

		// Call a second time to hit the stored programs
		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  scriptLocation,
			},
		)
		assert.NoError(t, err)

		assert.True(t, programsHits[scriptLocation])
		assert.False(t, programsHits[importedScriptLocation])
	})
}

func TestRuntimeInvalidTransactionArgumentAccount(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {}
        execute {}
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

}

func TestRuntimeTransactionWithAccount(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: &Account) {
          log(signer.address)
        }
      }
    `)

	var loggedMessage string

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{
				common.MustBytesToAddress([]byte{42}),
			}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, "0x000000000000002a", loggedMessage)
}

func TestRuntimeTransactionWithArguments(t *testing.T) {

	t.Parallel()

	type testCase struct {
		check        func(t *testing.T, err error)
		label        string
		script       string
		args         [][]byte
		authorizers  []Address
		expectedLogs []string
		contracts    map[common.AddressLocation][]byte
	}

	var tests = []testCase{
		{
			label: "Single argument",
			script: `
              transaction(x: Int) {
                execute {
                  log(x)
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewInt(42),
			}),
			expectedLogs: []string{"42"},
		},
		{
			label: "Single argument with authorizer",
			script: `
              transaction(x: Int) {
                prepare(signer: &Account) {
                  log(signer.address)
                }

                execute {
                  log(x)
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewInt(42),
			}),
			authorizers:  []Address{common.MustBytesToAddress([]byte{42})},
			expectedLogs: []string{"0x000000000000002a", "42"},
		},
		{
			label: "Multiple arguments",
			script: `
              transaction(x: Int, y: String) {
                execute {
                  log(x)
                  log(y)
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}),
			expectedLogs: []string{"42", `"foo"`},
		},
		{
			label: "Invalid bytes",
			script: `
              transaction(x: Int) { execute {} }
            `,
			args: [][]byte{
				{1, 2, 3, 4}, // not valid JSON-CDC
			},
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
			},
		},
		{
			label: "Type mismatch",
			script: `
              transaction(x: Int) {
                execute {
                  log(x)
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.String("foo"),
			}),
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidValueTypeError{}, errors.Unwrap(errors.Unwrap(err)))
			},
		},
		{
			label: "Address",
			script: `
              transaction(x: Address) {
                execute {
                  let acct = getAccount(x)
                  log(acct.address)
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.BytesToAddress(
					[]byte{
						0x0, 0x0, 0x0, 0x0,
						0x0, 0x0, 0x0, 0x1,
					},
				),
			}),
			expectedLogs: []string{"0x0000000000000001"},
		},
		{
			label: "Array",
			script: `
              transaction(x: [Int]) {
                execute {
                  log(x)
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewArray(
					[]cadence.Value{
						cadence.NewInt(1),
						cadence.NewInt(2),
						cadence.NewInt(3),
					},
				),
			}),
			expectedLogs: []string{"[1, 2, 3]"},
		},
		{
			label: "Dictionary",
			script: `
              transaction(x: {String:Int}) {
                execute {
                  log(x["y"])
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.String("y"),
							Value: cadence.NewInt(42),
						},
					},
				),
			}),
			expectedLogs: []string{"42"},
		},
		{
			label: "Invalid dictionary",
			script: `
              transaction(x: {String:String}) {
                execute {
                  log(x["y"])
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.String("y"),
							Value: cadence.NewInt(42),
						},
					},
				),
			}),
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				var argErr interpreter.ContainerMutationError
				require.ErrorAs(t, err, &argErr)
			},
		},
		{
			label: "Struct",
			contracts: map[common.AddressLocation][]byte{
				{
					Address: common.MustBytesToAddress([]byte{0x1}),
					Name:    "C",
				}: []byte(`
                  access(all) contract C {
                      access(all) struct Foo {
                           access(all) var y: String

                           init() {
                               self.y = "initial string"
                           }
                      }
                  }
                `),
			},
			script: `
              import C from 0x1

              transaction(x: C.Foo) {
                  execute {
                      log(x.y)
                  }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.
					NewStruct([]cadence.Value{cadence.String("bar")}).
					WithType(&cadence.StructType{
						Location: common.AddressLocation{
							Address: common.MustBytesToAddress([]byte{0x1}),
							Name:    "C",
						},
						QualifiedIdentifier: "C.Foo",
						Fields: []cadence.Field{
							{
								Identifier: "y",
								Type:       cadence.StringType,
							},
						},
					}),
			}),
			expectedLogs: []string{`"bar"`},
		},
		{
			label: "Struct in array",
			contracts: map[common.AddressLocation][]byte{
				{
					Address: common.MustBytesToAddress([]byte{0x1}),
					Name:    "C",
				}: []byte(`
                  access(all) contract C {
                      access(all) struct Foo {
                           access(all) var y: String

                           init() {
                               self.y = "initial string"
                           }
                      }
                  }
                `),
			},
			script: `
              import C from 0x1

              transaction(f: [C.Foo]) {
                execute {
                  let x = f[0]
                  log(x.y)
                }
              }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewArray([]cadence.Value{
					cadence.
						NewStruct([]cadence.Value{cadence.String("bar")}).
						WithType(&cadence.StructType{
							Location: common.AddressLocation{
								Address: common.MustBytesToAddress([]byte{0x1}),
								Name:    "C",
							},
							QualifiedIdentifier: "C.Foo",
							Fields: []cadence.Field{
								{
									Identifier: "y",
									Type:       cadence.StringType,
								},
							},
						}),
				}),
			}),
			expectedLogs: []string{`"bar"`},
		},
	}

	test := func(tc testCase) {

		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()
			rt := NewTestInterpreterRuntime()

			var loggedMessages []string

			storage := NewTestLedger(nil, nil)

			runtimeInterface := &TestRuntimeInterface{
				Storage: storage,
				OnGetSigningAccounts: func() ([]Address, error) {
					return tc.authorizers, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					return tc.contracts[location], nil
				},
				OnProgramLog: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
				OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
					return json.Decode(nil, b)
				},
			}

			err := rt.ExecuteTransaction(
				Script{
					Source:    []byte(tc.script),
					Arguments: tc.args,
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.TransactionLocation{},
				},
			)

			if tc.check != nil {
				tc.check(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tc.expectedLogs, loggedMessages)
			}
		})
	}

	for _, tt := range tests {
		test(tt)
	}
}

func TestRuntimeScriptArguments(t *testing.T) {

	t.Parallel()

	type testCase struct {
		check        func(t *testing.T, err error)
		name         string
		script       string
		args         [][]byte
		expectedLogs []string
	}

	var tests = []testCase{
		{
			name: "No arguments",
			script: `
                access(all) fun main() {
                    log("t")
                }
            `,
			args:         nil,
			expectedLogs: []string{`"t"`},
		},
		{
			name: "Single argument",
			script: `
                access(all) fun main(x: Int) {
                    log(x)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewInt(42),
			}),
			expectedLogs: []string{"42"},
		},
		{
			name: "Multiple arguments",
			script: `
                access(all) fun main(x: Int, y: String) {
                    log(x)
                    log(y)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}),
			expectedLogs: []string{"42", `"foo"`},
		},
		{
			name: "Invalid bytes",
			script: `
                access(all) fun main(x: Int) { }
            `,
			args: [][]byte{
				{1, 2, 3, 4}, // not valid JSON-CDC
			},
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
			},
		},
		{
			name: "Type mismatch",
			script: `
                access(all) fun main(x: Int) {
                    log(x)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.String("foo"),
			}),
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				assert.IsType(t, &InvalidEntryPointArgumentError{}, errors.Unwrap(err))
				assert.IsType(t, &InvalidValueTypeError{}, errors.Unwrap(errors.Unwrap(err)))
			},
		},
		{
			name: "Address",
			script: `
                access(all) fun main(x: Address) {
                    log(x)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.BytesToAddress(
					[]byte{
						0x0, 0x0, 0x0, 0x0,
						0x0, 0x0, 0x0, 0x1,
					},
				),
			}),
			expectedLogs: []string{"0x0000000000000001"},
		},
		{
			name: "Array",
			script: `
                access(all) fun main(x: [Int]) {
                    log(x)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewArray(
					[]cadence.Value{
						cadence.NewInt(1),
						cadence.NewInt(2),
						cadence.NewInt(3),
					},
				),
			}),
			expectedLogs: []string{"[1, 2, 3]"},
		},
		{
			name: "Constant-sized array, too many elements",
			script: `
                access(all) fun main(x: [Int; 2]) {
                    log(x)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewArray(
					[]cadence.Value{
						cadence.NewInt(1),
						cadence.NewInt(2),
						cadence.NewInt(3),
					},
				),
			}),
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				var invalidEntryPointArgumentErr *InvalidEntryPointArgumentError
				assert.ErrorAs(t, err, &invalidEntryPointArgumentErr)
			},
		},
		{
			name: "Constant-sized array, too few elements",
			script: `
                access(all) fun main(x: [Int; 2]) {
                    log(x)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewArray(
					[]cadence.Value{
						cadence.NewInt(1),
					},
				),
			}),
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				var invalidEntryPointArgumentErr *InvalidEntryPointArgumentError
				assert.ErrorAs(t, err, &invalidEntryPointArgumentErr)
			},
		},
		{
			name: "Dictionary",
			script: `
                access(all) fun main(x: {String:Int}) {
                    log(x["y"])
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.String("y"),
							Value: cadence.NewInt(42),
						},
					},
				),
			}),
			expectedLogs: []string{"42"},
		},
		{
			name: "Invalid dictionary",
			script: `
                access(all) fun main(x: {String:String}) {
                    log(x["y"])
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewDictionary(
					[]cadence.KeyValuePair{
						{
							Key:   cadence.String("y"),
							Value: cadence.NewInt(42),
						},
					},
				),
			}),
			check: func(t *testing.T, err error) {
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				var argErr interpreter.ContainerMutationError
				require.ErrorAs(t, err, &argErr)
			},
		},
		{
			name: "Struct",
			script: `
                access(all) struct Foo {
                    access(all) var y: String

                    init() {
                        self.y = "initial string"
                    }
                }

                access(all) fun main(x: Foo) {
                    log(x.y)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.
					NewStruct([]cadence.Value{cadence.String("bar")}).
					WithType(&cadence.StructType{
						Location:            common.ScriptLocation{},
						QualifiedIdentifier: "Foo",
						Fields: []cadence.Field{
							{
								Identifier: "y",
								Type:       cadence.StringType,
							},
						},
					}),
			}),
			expectedLogs: []string{`"bar"`},
		},
		{
			name: "Struct in array",
			script: `
                access(all) struct Foo {
                    access(all) var y: String

                    init() {
                        self.y = "initial string"
                    }
                }

                access(all) fun main(f: [Foo]) {
                    let x = f[0]
                    log(x.y)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.NewArray([]cadence.Value{
					cadence.
						NewStruct([]cadence.Value{cadence.String("bar")}).
						WithType(&cadence.StructType{
							Location:            common.ScriptLocation{},
							QualifiedIdentifier: "Foo",
							Fields: []cadence.Field{
								{
									Identifier: "y",
									Type:       cadence.StringType,
								},
							},
						}),
				}),
			}),
			expectedLogs: []string{`"bar"`},
		},
		{
			name: "Path subtype",
			script: `
                access(all) fun main(x: StoragePath) {
                    log(x)
                }
            `,
			args: encodeArgs([]cadence.Value{
				cadence.Path{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
			}),
			expectedLogs: []string{
				"/storage/foo",
			},
		},
	}

	test := func(tt testCase) {

		t.Run(tt.name, func(t *testing.T) {

			t.Parallel()

			rt := NewTestInterpreterRuntime()

			var loggedMessages []string

			storage := NewTestLedger(nil, nil)

			runtimeInterface := &TestRuntimeInterface{
				Storage: storage,
				OnProgramLog: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
				OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
					return json.Decode(nil, b)
				},
			}

			_, err := rt.ExecuteScript(
				Script{
					Source:    []byte(tt.script),
					Arguments: tt.args,
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)

			if tt.check != nil {
				tt.check(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expectedLogs, loggedMessages)
			}
		})
	}

	for _, tt := range tests {
		test(tt)
	}
}

func TestRuntimeProgramWithNoTransaction(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) fun main() {}
    `)

	runtimeInterface := &TestRuntimeInterface{}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	require.ErrorAs(t, err, &InvalidTransactionCountError{})
}

func TestRuntimeProgramWithMultipleTransaction(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        execute {}
      }
      transaction {
        execute {}
      }
    `)

	runtimeInterface := &TestRuntimeInterface{}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	require.ErrorAs(t, err, &InvalidTransactionCountError{})
}

func TestRuntimeStorage(t *testing.T) {

	t.Parallel()

	tests := map[string]string{
		"resource": `
          let r <- signer.storage.load<@R>(from: /storage/r)
          log(r == nil)
          destroy r

          signer.storage.save(<-createR(), to: /storage/r)
          let r2 <- signer.storage.load<@R>(from: /storage/r)
          log(r2 != nil)
          destroy r2
        `,
		"struct": `
          let s = signer.storage.load<S>(from: /storage/s)
          log(s == nil)

          signer.storage.save(S(), to: /storage/s)
          let s2 = signer.storage.load<S>(from: /storage/s)
          log(s2 != nil)
        `,
		"resource array": `
          let rs <- signer.storage.load<@[R]>(from: /storage/rs)
          log(rs == nil)
          destroy rs

          signer.storage.save(<-[<-createR()], to: /storage/rs)
          let rs2 <- signer.storage.load<@[R]>(from: /storage/rs)
          log(rs2 != nil)
          destroy rs2
        `,
		"struct array": `
          let s = signer.storage.load<[S]>(from: /storage/s)
          log(s == nil)

          signer.storage.save([S()], to: /storage/s)
          let s2 = signer.storage.load<[S]>(from: /storage/s)
          log(s2 != nil)
        `,
		"resource dictionary": `
          let rs <- signer.storage.load<@{String: R}>(from: /storage/rs)
          log(rs == nil)
          destroy rs

          signer.storage.save(<-{"r": <-createR()}, to: /storage/rs)
          let rs2 <- signer.storage.load<@{String: R}>(from: /storage/rs)
          log(rs2 != nil)
          destroy rs2
        `,
		"struct dictionary": `
          let s = signer.storage.load<{String: S}>(from: /storage/s)
          log(s == nil)

          signer.storage.save({"s": S()}, to: /storage/s)
          let rs2 = signer.storage.load<{String: S}>(from: /storage/s)
          log(rs2 != nil)
        `,
	}

	for name, code := range tests {
		t.Run(name, func(t *testing.T) {
			runtime := NewTestInterpreterRuntime()

			imported := []byte(`
              access(all) resource R {}

              access(all) fun createR(): @R {
                return <-create R()
              }

              access(all) struct S {}
            `)

			script := []byte(fmt.Sprintf(`
                  import "imported"

                  transaction {
                    prepare(signer: auth (Storage) &Account) {
                      %s
                    }
                  }
                `,
				code,
			))

			var loggedMessages []string

			runtimeInterface := &TestRuntimeInterface{
				OnGetCode: func(location Location) ([]byte, error) {
					switch location {
					case common.StringLocation("imported"):
						return imported, nil
					default:
						return nil, fmt.Errorf("unknown import location: %s", location)
					}
				},
				Storage: NewTestLedger(nil, nil),
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{{42}}, nil
				},
				OnProgramLog: func(message string) {
					loggedMessages = append(loggedMessages, message)
				},
			}

			nextTransactionLocation := NewTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			assert.Equal(t, []string{"true", "true"}, loggedMessages)
		})
	}
}

func TestRuntimeStorageMultipleTransactionsResourceWithArray(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	container := []byte(`
      access(all) resource Container {
        access(all) var values: [Int]

        init() {
          self.values = []
        }

		access(all) fun appendValue(_ v: Int) {
			self.values.append(v)
		}
      }

      access(all) fun createContainer(): @Container {
        return <-create Container()
      }
    `)

	script1 := []byte(`
      import "container"

      transaction {

        prepare(signer: auth (Storage, Capabilities) &Account) {
          signer.storage.save(<-createContainer(), to: /storage/container)
          let cap = signer.capabilities.storage.issue<auth(Insert) &Container>(/storage/container)
          signer.capabilities.publish(cap, at: /public/container)
        }
      }
    `)

	script2 := []byte(`
      import "container"

      transaction {
        prepare(signer: &Account) {
          let publicAccount = getAccount(signer.address)
          let ref = publicAccount.capabilities.borrow<auth(Insert) &Container>(/public/container)!

          let length = ref.values.length
          ref.appendValue(1)
          let length2 = ref.values.length
        }
      }
    `)

	script3 := []byte(`
      import "container"

      transaction {
        prepare(signer: &Account) {
          let publicAccount = getAccount(signer.address)
          let ref = publicAccount.capabilities.borrow<auth(Insert) &Container>(/public/container)!

          let length = ref.values.length
          ref.appendValue(2)
          let length2 = ref.values.length
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("container"):
				return container, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script3,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

// TestRuntimeStorageMultipleTransactionsResourceFunction tests a function call
// of a stored resource declared in an imported program
func TestRuntimeStorageMultipleTransactionsResourceFunction(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	deepThought := []byte(`
      access(all) resource DeepThought {

        access(all) fun answer(): Int {
          return 42
        }
      }

      access(all) fun createDeepThought(): @DeepThought {
        return <-create DeepThought()
      }
    `)

	script1 := []byte(`
      import "deep-thought"

      transaction {

        prepare(signer: auth(Storage) &Account) {
          signer.storage.save(<-createDeepThought(), to: /storage/deepThought)
        }
      }
    `)

	script2 := []byte(`
      import "deep-thought"

      transaction {
        prepare(signer: auth(Storage) &Account) {
          let answer = signer.storage.borrow<&DeepThought>(from: /storage/deepThought)?.answer()
          log(answer ?? 0)
        }
      }
    `)

	var loggedMessages []string

	ledger := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("deep-thought"):
				return deepThought, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: ledger,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Contains(t, loggedMessages, "42")
}

// TestRuntimeStorageMultipleTransactionsResourceField tests reading a field
// of a stored resource declared in an imported program
func TestRuntimeStorageMultipleTransactionsResourceField(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported := []byte(`
      access(all) resource SomeNumber {
        access(all) var n: Int
        init(_ n: Int) {
          self.n = n
        }
      }

      access(all) fun createNumber(_ n: Int): @SomeNumber {
        return <-create SomeNumber(n)
      }
    `)

	script1 := []byte(`
      import "imported"

      transaction {
        prepare(signer: auth(Storage) &Account) {
          signer.storage.save(<-createNumber(42), to: /storage/number)
        }
      }
    `)

	script2 := []byte(`
      import "imported"

      transaction {
        prepare(signer: auth(Storage) &Account) {
          if let number <- signer.storage.load<@SomeNumber>(from: /storage/number) {
            log(number.n)
            destroy number
          }
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Contains(t, loggedMessages, "42")
}

// TestRuntimeCompositeFunctionInvocationFromImportingProgram checks
// that member functions of imported composites can be invoked from an importing program.
// See https://github.com/dapperlabs/flow-go/issues/838
func TestRuntimeCompositeFunctionInvocationFromImportingProgram(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported := []byte(`
      // function must have arguments
      access(all) fun x(x: Int) {}

      // invocation must be in composite
      access(all) resource Y {
        access(all) fun x() {
          x(x: 1)
        }
      }

      access(all) fun createY(): @Y {
        return <-create Y()
      }
    `)

	script1 := []byte(`
      import Y, createY from "imported"

      transaction {
        prepare(signer: auth(Storage) &Account) {
          signer.storage.save(<-createY(), to: /storage/y)
        }
      }
    `)

	script2 := []byte(`
      import Y from "imported"

      transaction {
        prepare(signer: auth(Storage) &Account) {
          let y <- signer.storage.load<@Y>(from: /storage/y)
          y?.x()
          destroy y
        }
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeResourceContractUseThroughReference(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported := []byte(`
      access(all) resource R {
        access(all) fun x() {
          log("x!")
        }
      }

      access(all) fun createR(): @R {
        return <- create R()
      }
    `)

	script1 := []byte(`
      import R, createR from "imported"

      transaction {

        prepare(signer: auth(Storage) &Account) {
          signer.storage.save(<-createR(), to: /storage/r)
        }
      }
    `)

	script2 := []byte(`
      import R from "imported"

      transaction {

        prepare(signer: auth(Storage) &Account) {
          let ref = signer.storage.borrow<&R>(from: /storage/r)!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestRuntimeResourceContractUseThroughLink(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported := []byte(`
      access(all) resource R {
        access(all) fun x() {
          log("x!")
        }
      }

      access(all) fun createR(): @R {
          return <- create R()
      }
    `)

	script1 := []byte(`
      import R, createR from "imported"

      transaction {

        prepare(signer: auth(Storage, Capabilities) &Account) {
          signer.storage.save(<-createR(), to: /storage/r)
          let cap = signer.capabilities.storage.issue<&R>(/storage/r)
          signer.capabilities.publish(cap, at: /public/r)
        }
      }
    `)

	script2 := []byte(`
      import R from "imported"

      transaction {
        prepare(signer: &Account) {
          let publicAccount = getAccount(signer.address)
          let ref = publicAccount.capabilities.borrow<&R>(/public/r)!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestRuntimeResourceContractWithInterface(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported1 := []byte(`
      access(all) resource interface RI {
        access(all) fun x()
      }
    `)

	imported2 := []byte(`
      import RI from "imported1"

      access(all) resource R: RI {
        access(all) fun x() {
          log("x!")
        }
      }

      access(all) fun createR(): @R {
        return <- create R()
      }
    `)

	script1 := []byte(`
      import RI from "imported1"
      import R, createR from "imported2"

      transaction {
        prepare(signer: auth(Storage, Capabilities) &Account) {
          signer.storage.save(<-createR(), to: /storage/r)
          let cap = signer.capabilities.storage.issue<&{RI}>(/storage/r)
          signer.capabilities.publish(cap, at: /public/r)
        }
      }
    `)

	// TODO: Get rid of the requirement that the underlying type must be imported.
	//   This requires properly initializing Interpreter.CompositeFunctions.
	//   Also initialize Interpreter.DestructorFunctions

	script2 := []byte(`
      import RI from "imported1"
      import R from "imported2"

      transaction {
        prepare(signer: &Account) {
          let ref = signer.capabilities.borrow<&{RI}>(/public/r)!
          ref.x()
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported1"):
				return imported1, nil
			case common.StringLocation("imported2"):
				return imported2, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"\"x!\""}, loggedMessages)
}

func TestRuntimeParseAndCheckProgram(t *testing.T) {

	t.Parallel()

	t.Run("ValidProgram", func(t *testing.T) {
		runtime := NewTestInterpreterRuntime()

		script := []byte("access(all) fun test(): Int { return 42 }")
		runtimeInterface := &TestRuntimeInterface{}

		nextTransactionLocation := NewTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		assert.NoError(t, err)
	})

	t.Run("InvalidSyntax", func(t *testing.T) {
		runtime := NewTestInterpreterRuntime()

		script := []byte("invalid syntax")
		runtimeInterface := &TestRuntimeInterface{}

		nextTransactionLocation := NewTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		assert.NotNil(t, err)
	})

	t.Run("InvalidSemantics", func(t *testing.T) {
		runtime := NewTestInterpreterRuntime()

		script := []byte(`access(all) let a: Int = "b"`)
		runtimeInterface := &TestRuntimeInterface{}

		nextTransactionLocation := NewTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		assert.NotNil(t, err)
	})
}

func TestRuntimeScriptReturnSpecial(t *testing.T) {

	t.Parallel()

	type testCase struct {
		expected cadence.Value
		code     string
		invalid  bool
	}

	test := func(t *testing.T, test testCase) {

		runtime := NewTestInterpreterRuntime()

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
		}

		actual, err := runtime.ExecuteScript(
			Script{
				Source: []byte(test.code),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		if test.invalid {
			RequireError(t, err)

			var subErr *InvalidScriptReturnTypeError
			require.ErrorAs(t, err, &subErr)
		} else {
			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		}
	}

	t.Run("interpreted function", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  access(all) fun main(): AnyStruct {
                      return fun (): Int {
                          return 0
                      }
                  }
                `,
				expected: cadence.Function{
					FunctionType: &cadence.FunctionType{
						ReturnType: cadence.IntType,
					},
				},
			},
		)
	})

	t.Run("host function", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  access(all) fun main(): AnyStruct {
                      return panic
                  }
                `,
				expected: cadence.Function{
					FunctionType: &cadence.FunctionType{
						Purity: sema.FunctionPurityView,
						Parameters: []cadence.Parameter{
							{
								Label:      sema.ArgumentLabelNotRequired,
								Identifier: "message",
								Type:       cadence.StringType,
							},
						},
						ReturnType: cadence.NeverType,
					},
				},
			},
		)
	})

	t.Run("bound function", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  access(all) struct S {
                      access(all) fun f() {}
                  }

                  access(all) fun main(): AnyStruct {
                      let s = S()
                      return s.f
                  }
                `,
				expected: cadence.Function{
					FunctionType: &cadence.FunctionType{
						ReturnType: cadence.VoidType,
					},
				},
			},
		)
	})

	t.Run("reference", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  access(all) fun main(): AnyStruct {
                      let a: Address = 0x1
                      return &a as &Address
                  }
                `,
				expected: cadence.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
		)
	})

	t.Run("recursive reference", func(t *testing.T) {

		t.Parallel()

		test(t,
			testCase{
				code: `
                  access(all) fun main(): AnyStruct {
                      let refs: [&AnyStruct] = []
                      refs.append(&refs as &AnyStruct)
                      return refs
                  }
                `,
				expected: cadence.NewArray([]cadence.Value{
					cadence.NewArray([]cadence.Value{
						nil,
					}).WithType(&cadence.VariableSizedArrayType{
						ElementType: &cadence.ReferenceType{
							Type:          cadence.AnyStructType,
							Authorization: cadence.UnauthorizedAccess,
						},
					}),
				}).WithType(&cadence.VariableSizedArrayType{
					ElementType: &cadence.ReferenceType{
						Type:          cadence.AnyStructType,
						Authorization: cadence.UnauthorizedAccess,
					},
				}),
			},
		)
	})
}

func TestRuntimeScriptParameterTypeNotImportableError(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) fun main(x: fun(): Int) {
        return
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

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

	var subErr *ScriptParameterTypeNotImportableError
	require.ErrorAs(t, err, &subErr)
}

func TestRuntimeSyntaxError(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) fun main(): String {
          return "Hello World!
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

}

func TestRuntimeStorageChanges(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported := []byte(`
      access(all) resource X {
        access(all) var x: Int

        init() {
          self.x = 0
        }

		access(all) fun setX(_ x: Int) {
			self.x = x
		}
      }

      access(all) fun createX(): @X {
          return <-create X()
      }
    `)

	script1 := []byte(`
      import X, createX from "imported"

      transaction {
        prepare(signer: auth(Storage) &Account) {
          signer.storage.save(<-createX(), to: /storage/x)

          let ref = signer.storage.borrow<&X>(from: /storage/x)!
          ref.setX(1)
        }
      }
    `)

	script2 := []byte(`
      import X from "imported"

      transaction {
        prepare(signer: auth(Storage) &Account) {
          let ref = signer.storage.borrow<&X>(from: /storage/x)!
          log(ref.x)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"1"}, loggedMessages)
}

func TestRuntimeAccountAddress(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: &Account) {
          log(signer.address)
        }
      }
    `)

	var loggedMessages []string

	address := common.MustBytesToAddress([]byte{42})

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"0x000000000000002a"}, loggedMessages)
}

func TestRuntimePublicAccountAddress(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          log(getAccount(0x42).address)
        }
      }
    `)

	var loggedMessages []string

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x42})

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			address.String(),
		},
		loggedMessages,
	)
}

func TestRuntimeAccountPublishAndAccess(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported := []byte(`
      access(all) resource R {
        access(all) fun test(): Int {
          return 42
        }
      }

      access(all) fun createR(): @R {
        return <-create R()
      }
    `)

	script1 := []byte(`
      import "imported"

      transaction {
        prepare(signer: auth(Storage, Capabilities) &Account) {
          signer.storage.save(<-createR(), to: /storage/r)
          let cap = signer.capabilities.storage.issue<&R>(/storage/r)
          signer.capabilities.publish(cap, at: /public/r)
        }
      }
    `)

	address := common.MustBytesToAddress([]byte{42})

	script2 := []byte(
		fmt.Sprintf(
			`
              import "imported"

              transaction {

                prepare(signer: &Account) {
                  log(getAccount(0x%s).capabilities.borrow<&R>(/public/r)!.test())
                }
              }
            `,
			address,
		),
	)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) ([]byte, error) {
			switch location {
			case common.StringLocation("imported"):
				return imported, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"42"}, loggedMessages)
}

func TestRuntimeTransaction_CreateAccount(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: auth(Storage) &Account) {
          Account(payer: signer)
        }
      }
    `)

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnCreateAccount: func(payer Address) (address Address, err error) {
			return Address{42}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.EqualValues(
		t,
		stdlib.AccountCreatedEventType.ID(),
		events[0].Type().ID(),
	)
}

func TestRuntimeContractAccount(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
      access(all) contract Test {
          access(all) let address: Address

          init() {
              // field 'account' can be used, as it is considered initialized
              self.address = self.account.address
          }

          // test that both functions are linked back into restored composite values,
          // and also injected fields are injected back into restored composite values
          //
          access(all) fun test(): Address {
              return self.account.address
          }
      }
    `)

	script1 := []byte(`
      import Test from 0xCADE

      access(all) fun main(): Address {
          return Test.address
      }
    `)

	script2 := []byte(`
      import Test from 0xCADE

      access(all) fun main(): Address {
          return Test.test()
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	// Run script 1

	value, err := runtime.ExecuteScript(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, addressValue, value)

	// Run script 2

	value, err = runtime.ExecuteScript(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, addressValue, value)
}

func TestRuntimeInvokeContractFunction(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        access(all) contract Test {
            access(all) fun hello() {
                log("Hello World!")
            }
            access(all) fun helloArg(_ arg: String) {
                log("Hello ".concat(arg))
            }
            access(all) fun helloMultiArg(arg1: String, arg2: Int, arg3: Address) {
                log("Hello ".concat(arg1).concat(" ").concat(arg2.toString()).concat(" from ").concat(arg3.toString()))
            }
            access(all) fun helloReturn(_ arg: String): String {
                log("Hello return!")
                return arg
            }
            access(all) fun helloAuthAcc(account: &Account) {
                log("Hello ".concat(account.address.toString()))
            }
            access(all) fun helloPublicAcc(account: &Account) {
                log("Hello access(all) ".concat(account.address.toString()))
            }
        }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte
	var loggedMessage string

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	t.Run("simple function", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"hello",
			nil,
			nil,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, `"Hello World!"`, loggedMessage)
	})

	t.Run("function with parameter", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.String("there!"),
			},
			[]sema.Type{
				sema.StringType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, `"Hello there!"`, loggedMessage)
	})

	t.Run("function with return type", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloReturn",
			[]cadence.Value{
				cadence.String("there!"),
			},
			[]sema.Type{
				sema.StringType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, `"Hello return!"`, loggedMessage)
	})

	t.Run("function with multiple arguments", func(t *testing.T) {

		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloMultiArg",
			[]cadence.Value{
				cadence.String("number"),
				cadence.NewInt(42),
				cadence.BytesToAddress(addressValue.Bytes()),
			},
			[]sema.Type{
				sema.StringType,
				sema.IntType,
				sema.TheAddressType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, `"Hello number 42 from 0x0000000000000001"`, loggedMessage)
	})

	t.Run("function with not enough arguments panics", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloMultiArg",
			[]cadence.Value{
				cadence.String("number"),
				cadence.NewInt(42),
			},
			[]sema.Type{
				sema.StringType,
				sema.IntType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)

		assert.ErrorAs(t, err, &Error{})
	})

	t.Run("function with incorrect argument type errors", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.NewInt(42),
			},
			[]sema.Type{
				sema.IntType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ValueTransferTypeError{})
	})

	t.Run("function with un-importable argument errors and error propagates (ID capability)", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.NewCapability(
					1,
					cadence.Address{},
					cadence.AddressType, // this will error during `importValue`
				),
			},
			[]sema.Type{
				&sema.CapabilityType{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		require.ErrorContains(t, err, "cannot import capability")
	})

	t.Run("function with un-importable argument errors and error propagates (ID capability)", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloArg",
			[]cadence.Value{
				cadence.NewCapability(
					42,
					cadence.Address{},
					cadence.AddressType, // this will error during `importValue`
				),
			},
			[]sema.Type{
				&sema.CapabilityType{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		require.ErrorContains(t, err, "cannot import capability")
	})

	t.Run("function with auth account works", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloAuthAcc",
			[]cadence.Value{
				cadence.BytesToAddress(addressValue.Bytes()),
			},
			[]sema.Type{
				sema.FullyEntitledAccountReferenceType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, `"Hello 0x0000000000000001"`, loggedMessage)
	})
	t.Run("function with public account works", func(t *testing.T) {
		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"helloPublicAcc",
			[]cadence.Value{
				cadence.BytesToAddress(addressValue.Bytes()),
			},
			[]sema.Type{
				sema.AccountReferenceType,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(t, `"Hello access(all) 0x0000000000000001"`, loggedMessage)
	})
}

func TestRuntimeContractNestedResource(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        access(all) contract Test {
            access(all) resource R {
                // test that the hello function is linked back into the nested resource
                // after being loaded from storage
                access(all) fun hello(): String {
                    return "Hello World!"
                }
            }

            init() {
                // store nested resource in account on deployment
                self.account.storage.save(<-create R(), to: /storage/r)
            }
        }
    `)

	tx := []byte(`
        import Test from 0x01

        transaction {

            prepare(acct: auth(Storage) &Account) {
                log(acct.storage.borrow<&Test.R>(from: /storage/r)?.hello())
            }
        }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte
	var loggedMessage string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessage = message
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, `"Hello World!"`, loggedMessage)
}

func TestRuntimeStorageLoadedDestructionConcreteType(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        access(all) contract Test {

            access(all) resource R {}

            init() {
                // store nested resource in account on deployment
                self.account.storage.save(<-create R(), to: /storage/r)
            }
        }
    `)

	tx := []byte(`
        import Test from 0x01

        transaction {

            prepare(acct: auth(Storage) &Account) {
                let r <- acct.storage.load<@Test.R>(from: /storage/r)
                destroy r
            }
        }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error { return nil },
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		})
	require.NoError(t, err)

	// DestructorTODO: Assert default event is emitted here

}

func TestRuntimeStorageLoadedDestructionAnyResource(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        access(all) contract Test {
            access(all) resource R {}

            init() {
                // store nested resource in account on deployment
                self.account.storage.save(<-create R(), to: /storage/r)
            }
        }
    `)

	tx := []byte(`
        // NOTE: *not* importing concrete implementation.
        //   Should be imported automatically when loading the value from storage

        transaction {

            prepare(acct: auth(Storage) &Account) {
                let r <- acct.storage.load<@AnyResource>(from: /storage/r)
                destroy r
            }
        }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error { return nil },
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// DestructorTODO: Assert default event is emitted here
}

func TestRuntimeStorageLoadedDestructionAfterRemoval(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
        access(all) contract Test {
            access(all) resource R {}

            init() {
                // store nested resource in account on deployment
                self.account.storage.save(<-create R(), to: /storage/r)
            }
        }
    `)

	tx := []byte(`
        // NOTE: *not* importing concrete implementation.
        //   Should be imported automatically when loading the value from storage

        transaction {

            prepare(acct: auth(Storage) &Account) {
                let r <- acct.storage.load<@AnyResource>(from: /storage/r)
                destroy r
            }
        }
    `)

	deploy := DeploymentTransaction("Test", contract)
	removal := RemovalTransaction("Test")

	var accountCode []byte

	ledger := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: ledger,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnRemoveAccountContractCode: func(_ common.AddressLocation) (err error) {
			accountCode = nil
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error { return nil },
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy the contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	// Remove the contract

	err = runtime.ExecuteTransaction(
		Script{
			Source: removal,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Nil(t, accountCode)

	// Destroy

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	var typeLoadingErr interpreter.TypeLoadingError
	require.ErrorAs(t, err, &typeLoadingErr)

	require.Equal(t,
		common.AddressLocation{Address: addressValue}.TypeID(nil, "Test.R"),
		typeLoadingErr.TypeID,
	)
}

const basicFungibleTokenContract = `
access(all) contract FungibleToken {

    access(all) resource interface Provider {
        access(all) fun withdraw(amount: Int): @Vault {
            pre {
                amount > 0:
                    "Withdrawal amount must be positive"
            }
            post {
                result.balance == amount:
                    "Incorrect amount returned"
            }
        }
    }

    access(all) resource interface Receiver {
        access(all) balance: Int

        init(balance: Int) {
            pre {
                balance >= 0:
                    "Initial balance must be non-negative"
            }
            post {
                self.balance == balance:
                    "Balance must be initialized to the initial balance"
            }
        }

        access(all) fun deposit(from: @{Receiver}) {
            pre {
                from.balance > 0:
                    "Deposit balance needs to be positive!"
            }
            post {
                self.balance == before(self.balance) + before(from.balance):
                    "Incorrect amount removed"
            }
        }
    }

    access(all) resource Vault: Provider, Receiver {

        access(all) var balance: Int

        init(balance: Int) {
            self.balance = balance
        }

        access(all) fun withdraw(amount: Int): @Vault {
            self.balance = self.balance - amount
            return <-create Vault(balance: amount)
        }

        // transfer combines withdraw and deposit into one function call
        access(all) fun transfer(to: &{Receiver}, amount: Int) {
            pre {
                amount <= self.balance:
                    "Insufficient funds"
            }
            post {
                self.balance == before(self.balance) - amount:
                    "Incorrect amount removed"
            }
            to.deposit(from: <-self.withdraw(amount: amount))
        }

        access(all) fun deposit(from: @{Receiver}) {
            self.balance = self.balance + from.balance
            destroy from
        }

        access(all) fun createEmptyVault(): @Vault {
            return <-create Vault(balance: 0)
        }
    }

    access(all) fun createEmptyVault(): @Vault {
        return <-create Vault(balance: 0)
    }

    access(all) resource VaultMinter {
        access(all) fun mintTokens(amount: Int, recipient: &{Receiver}) {
            recipient.deposit(from: <-create Vault(balance: amount))
        }
    }

    init() {
        self.account.storage.save(<-create Vault(balance: 30), to: /storage/vault)
        self.account.storage.save(<-create VaultMinter(), to: /storage/minter)
    }
}
`

func TestRuntimeFungibleTokenUpdateAccountCode(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address1Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	address2Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
	}

	deploy := DeploymentTransaction("FungibleToken", []byte(basicFungibleTokenContract))

	setup1Transaction := []byte(`
      import FungibleToken from 0x01

      transaction {

          prepare(acct: auth(Capabilities) &Account) {

              let receiverCap = acct.capabilities.storage
                  .issue<&{FungibleToken.Receiver}>(/storage/vault)
              acct.capabilities.publish(receiverCap, at: /public/receiver)

              let vaultCap = acct.capabilities.storage.issue<&FungibleToken.Vault>(/storage/vault)
              acct.capabilities.publish(vaultCap, at: /public/vault)
          }
      }
    `)

	setup2Transaction := []byte(`
      // NOTE: import location not the same as in setup1Transaction
      import FungibleToken from 0x01

      transaction {

          prepare(acct: auth(Storage, Capabilities) &Account) {
              let vault <- FungibleToken.createEmptyVault()

              acct.storage.save(<-vault, to: /storage/vault)

              let receiverCap = acct.capabilities.storage
                  .issue<&{FungibleToken.Receiver}>(/storage/vault)
              acct.capabilities.publish(receiverCap, at: /public/receiver)

              let vaultCap = acct.capabilities.storage.issue<&FungibleToken.Vault>(/storage/vault)
              acct.capabilities.publish(vaultCap, at: /public/vault)
          }
      }
    `)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	signerAccount := address1Value

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup1Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	signerAccount = address2Value

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup2Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeFungibleTokenCreateAccount(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address1Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	address2Value := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
	}

	deploy := []byte(fmt.Sprintf(
		`
          transaction {
            prepare(signer: auth(Storage) &Account) {
                let acct = Account(payer: signer)
                acct.contracts.add(name: "FungibleToken", code: "%s".decodeHex())
            }
          }
        `,
		hex.EncodeToString([]byte(basicFungibleTokenContract)),
	))

	setup1Transaction := []byte(`
      import FungibleToken from 0x2

      transaction {

          prepare(acct: auth(Capabilities) &Account) {
              let receiverCap = acct.capabilities.storage
                  .issue<&{FungibleToken.Receiver}>(/storage/vault)
              acct.capabilities.publish(receiverCap, at: /public/receiver1)

              let vaultCap = acct.capabilities.storage.issue<&FungibleToken.Vault>(/storage/vault)
              acct.capabilities.publish(vaultCap, at: /public/vault1)
          }
      }
    `)

	setup2Transaction := []byte(`
      // NOTE: import location not the same as in setup1Transaction
      import FungibleToken from 0x02

      transaction {

          prepare(acct: auth(Storage, Capabilities) &Account) {
              let vault <- FungibleToken.createEmptyVault()

              acct.storage.save(<-vault, to: /storage/vault)

              let receiverCap = acct.capabilities.storage
                  .issue<&{FungibleToken.Receiver}>(/storage/vault)
              acct.capabilities.publish(receiverCap, at: /public/receiver2)

              let vaultCap = acct.capabilities.storage.issue<&FungibleToken.Vault>(/storage/vault)
              acct.capabilities.publish(vaultCap, at: /public/vault2)
          }
      }
    `)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	signerAccount := address1Value

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnCreateAccount: func(payer Address) (address Address, err error) {
			return address2Value, nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup1Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setup2Transaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeInvokeStoredInterfaceFunction(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	makeDeployTransaction := func(name, code string) []byte {
		return []byte(fmt.Sprintf(
			`
              transaction {
                prepare(signer: auth(Storage) &Account) {
                  let acct = Account(payer: signer)
                  acct.contracts.add(name: "%s", code: "%s".decodeHex())
                }
              }
            `,
			name,
			hex.EncodeToString([]byte(code)),
		))
	}

	contractInterfaceCode := `
      access(all) contract interface TestContractInterface {

          access(all) resource interface RInterface {

              access(all) fun check(a: Int, b: Int) {
                  pre { a > 1 }
                  post { b > 1 }
              }
          }
      }
    `

	contractCode := `
      import TestContractInterface from 0x2

      access(all) contract TestContract: TestContractInterface {

          access(all) resource R: TestContractInterface.RInterface {

              access(all) fun check(a: Int, b: Int) {
                  pre { a < 3 }
                  post { b < 3 }
              }
          }

          access(all) fun createR(): @R {
              return <-create R()
          }
       }
    `

	setupCode := []byte(`
      import TestContractInterface from 0x2
      import TestContract from 0x3

      transaction {
          prepare(signer: auth(Storage) &Account) {
              signer.storage.save(<-TestContract.createR(), to: /storage/r)
          }
      }
    `)

	makeUseCode := func(a int, b int) []byte {
		return []byte(
			fmt.Sprintf(
				`
                  import TestContractInterface from 0x2

                  // NOTE: *not* importing concrete implementation.
                  //   Should be imported automatically when loading the value from storage

                  // import TestContract from 0x3

                  transaction {
                      prepare(signer: auth(Storage) &Account) {
                          signer.storage.borrow<&{TestContractInterface.RInterface}>(from: /storage/r)
                            ?.check(a: %d, b: %d)
                      }
                  }
                `,
				a,
				b,
			),
		)
	}

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	var nextAccount byte = 0x2

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnCreateAccount: func(payer Address) (address Address, err error) {
			result := interpreter.NewUnmeteredAddressValueFromBytes([]byte{nextAccount})
			nextAccount++
			return result.ToAddress(), nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{0x1}}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	deployTransaction := makeDeployTransaction("TestContractInterface", contractInterfaceCode)
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

	err = runtime.ExecuteTransaction(
		Script{
			Source: setupCode,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	for a := 1; a <= 3; a++ {
		for b := 1; b <= 3; b++ {

			t.Run(fmt.Sprintf("%d/%d", a, b), func(t *testing.T) {

				err = runtime.ExecuteTransaction(
					Script{
						Source: makeUseCode(a, b),
					},
					Context{
						Interface: runtimeInterface,
						Location:  nextTransactionLocation(),
					},
				)

				if a == 2 && b == 2 {
					assert.NoError(t, err)
				} else {
					RequireError(t, err)

					assertRuntimeErrorIsUserError(t, err)

					require.ErrorAs(t, err, &interpreter.ConditionError{})
				}
			})
		}
	}
}

func TestRuntimeBlock(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          let block = getCurrentBlock()
          log(block)
          log(block.height)
          log(block.view)
          log(block.id)
          log(block.timestamp)

          let nextBlock = getBlock(at: block.height + UInt64(1))
          log(nextBlock)
          log(nextBlock?.height)
          log(nextBlock?.view)
          log(nextBlock?.id)
          log(nextBlock?.timestamp)
        }
      }
    `)

	var loggedMessages []string

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"Block(height: 1, view: 1, id: 0x0000000000000000000000000000000000000000000000000000000000000001, timestamp: 1.00000000)",
			"1",
			"1",
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1]",
			"1.00000000",
			"Block(height: 2, view: 2, id: 0x0000000000000000000000000000000000000000000000000000000000000002, timestamp: 2.00000000)",
			"2",
			"2",
			"[0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2]",
			"2.00000000",
		},
		loggedMessages,
	)
}

func TestRuntimeUnsafeRandom(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          let rand = unsafeRandom()
          log(rand)
        }
      }
    `)

	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnReadRandom: func(buffer []byte) error {
			binary.LittleEndian.PutUint64(buffer, 7558174677681708339)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"7558174677681708339",
		},
		loggedMessages,
	)
}

func TestRuntimeTransactionTopLevelDeclarations(t *testing.T) {

	t.Parallel()

	t.Run("transaction with function", func(t *testing.T) {
		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all) fun test() {}

          transaction {}
        `)

		runtimeInterface := &TestRuntimeInterface{
			OnGetSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	})

	t.Run("transaction with resource", func(t *testing.T) {
		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all) resource R {}

          transaction {}
        `)

		runtimeInterface := &TestRuntimeInterface{
			OnGetSigningAccounts: func() ([]Address, error) {
				return nil, nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		assertRuntimeErrorIsUserError(t, err)

		var checkerErr *sema.CheckerError
		require.ErrorAs(t, err, &checkerErr)

		errs := checker.RequireCheckerErrors(t, checkerErr, 1)

		assert.IsType(t, &sema.InvalidTopLevelDeclarationError{}, errs[0])
	})
}

func TestRuntimeStoreIntegerTypes(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := interpreter.AddressValue{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xCA, 0xDE,
	}

	for _, integerType := range sema.AllIntegerTypes {

		typeName := integerType.String()

		t.Run(typeName, func(t *testing.T) {

			contract := []byte(
				fmt.Sprintf(
					`
                      access(all) contract Test {

                          access(all) let n: %s

                          init() {
                              self.n = 42
                          }
                      }
                    `,
					typeName,
				),
			)

			deploy := DeploymentTransaction("Test", contract)

			var accountCode []byte
			var events []cadence.Event

			runtimeInterface := &TestRuntimeInterface{
				OnGetCode: func(_ Location) (bytes []byte, err error) {
					return accountCode, nil
				},
				Storage: NewTestLedger(nil, nil),
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{addressValue.ToAddress()}, nil
				},
				OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
					return accountCode, nil
				},
				OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
					accountCode = code
					return nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					events = append(events, event)
					return nil
				},
			}

			nextTransactionLocation := NewTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: deploy,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			assert.NotNil(t, accountCode)
		})
	}
}

func TestRuntimeResourceOwnerFieldUseComposite(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
      access(all) contract Test {

          access(all) resource R {

              access(all) fun logOwnerAddress() {
                log(self.owner?.address)
              }
          }

          access(all) fun createR(): @R {
              return <-create R()
          }
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: auth(Storage, Capabilities) &Account) {

              let r <- Test.createR()
              log(r.owner?.address)
              r.logOwnerAddress()

              signer.storage.save(<-r, to: /storage/r)
              let cap = signer.capabilities.storage.issue<&Test.R>(/storage/r)
              signer.capabilities.publish(cap, at: /public/r)

              let ref1 = signer.storage.borrow<&Test.R>(from: /storage/r)!
              log(ref1.owner?.address)
              ref1.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.capabilities.borrow<&Test.R>(/public/r)!
              log(ref2.owner?.address)
              ref2.logOwnerAddress()
          }
      }
    `)

	tx2 := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: auth(Storage) &Account) {
              let ref1 = signer.storage.borrow<&Test.R>(from: /storage/r)!
              log(ref1.owner?.address)
              log(ref1.owner?.balance)
              log(ref1.owner?.availableBalance)
              log(ref1.owner?.storage?.used)
              log(ref1.owner?.storage?.capacity)
              ref1.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.capabilities.borrow<&Test.R>(/public/r)!
              log(ref2.owner?.address)
              log(ref2.owner?.balance)
              log(ref2.owner?.availableBalance)
              log(ref2.owner?.storage?.used)
              log(ref2.owner?.storage?.capacity)
              ref2.logOwnerAddress()
          }
      }
    `)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnGetAccountBalance: func(_ Address) (uint64, error) {
			// return a dummy value
			return 12300000000, nil
		},
		OnGetAccountAvailableBalance: func(_ Address) (uint64, error) {
			// return a dummy value
			return 152300000000, nil
		},
		OnGetStorageUsed: func(_ Address) (uint64, error) {
			// return a dummy value
			return 120, nil
		},
		OnGetStorageCapacity: func(_ Address) (uint64, error) {
			// return a dummy value
			return 1245, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"nil", "nil",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
		},
		loggedMessages,
	)

	loggedMessages = nil
	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"0x0000000000000001", // ref1.owner?.address
			"123.00000000",       // ref2.owner?.balance
			"1523.00000000",      // ref2.owner?.availableBalance
			"120",                // ref1.owner?.storage.used
			"1245",               // ref1.owner?.storage.capacity

			"0x0000000000000001",

			"0x0000000000000001", // ref2.owner?.address
			"123.00000000",       // ref2.owner?.balance
			"1523.00000000",      // ref2.owner?.availableBalance
			"120",                // ref2.owner?.storage.used
			"1245",               // ref2.owner?.storage.capacity

			"0x0000000000000001",
		},
		loggedMessages,
	)
}

func TestRuntimeResourceOwnerFieldUseArray(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
      access(all) contract Test {

          access(all) resource R {

              access(all) fun logOwnerAddress() {
                log(self.owner?.address)
              }
          }

          access(all) fun createR(): @R {
              return <-create R()
          }
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: auth(Storage, Capabilities) &Account) {

              let rs <- [
                  <-Test.createR(),
                  <-Test.createR()
              ]
              log(rs[0].owner?.address)
              log(rs[1].owner?.address)
              rs[0].logOwnerAddress()
              rs[1].logOwnerAddress()

              signer.storage.save(<-rs, to: /storage/rs)
              let cap = signer.capabilities.storage.issue<&[Test.R]>(/storage/rs)
              signer.capabilities.publish(cap, at: /public/rs)

              let ref1 = signer.storage.borrow<&[Test.R]>(from: /storage/rs)!
              log(ref1[0].owner?.address)
              log(ref1[1].owner?.address)
              ref1[0].logOwnerAddress()
              ref1[1].logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.capabilities.borrow<&[Test.R]>(/public/rs)!
              log(ref2[0].owner?.address)
              log(ref2[1].owner?.address)
              ref2[0].logOwnerAddress()
              ref2[1].logOwnerAddress()
          }
      }
    `)

	tx2 := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: auth(Storage) &Account) {
              let ref1 = signer.storage.borrow<&[Test.R]>(from: /storage/rs)!
              log(ref1[0].owner?.address)
              log(ref1[1].owner?.address)
              ref1[0].logOwnerAddress()
              ref1[1].logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.capabilities.borrow<&[Test.R]>(/public/rs)!
              log(ref2[0].owner?.address)
              log(ref2[1].owner?.address)
              ref2[0].logOwnerAddress()
              ref2[1].logOwnerAddress()
          }
      }
    `)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"nil", "nil",
			"nil", "nil",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
		},
		loggedMessages,
	)

	loggedMessages = nil
	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
		},
		loggedMessages,
	)
}

func TestRuntimeResourceOwnerFieldUseDictionary(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	contract := []byte(`
      access(all) contract Test {

          access(all) resource R {

              access(all) fun logOwnerAddress() {
                log(self.owner?.address)
              }
          }

          access(all) fun createR(): @R {
              return <-create R()
          }
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	tx := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: auth(Storage, Capabilities) &Account) {

              let rs <- {
                  "a": <-Test.createR(),
                  "b": <-Test.createR()
              }
              log(rs["a"]?.owner?.address)
              log(rs["b"]?.owner?.address)
              rs["a"]?.logOwnerAddress()
              rs["b"]?.logOwnerAddress()

              signer.storage.save(<-rs, to: /storage/rs)
              let cap = signer.capabilities.storage.issue<&{String: Test.R}>(/storage/rs)
              signer.capabilities.publish(cap, at: /public/rs)

              let ref1 = signer.storage.borrow<&{String: Test.R}>(from: /storage/rs)!
              log(ref1["a"]?.owner?.address)
              log(ref1["b"]?.owner?.address)
              ref1["a"]?.logOwnerAddress()
              ref1["b"]?.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.capabilities.borrow<&{String: Test.R}>(/public/rs)!
              log(ref2["a"]?.owner?.address)
              log(ref2["b"]?.owner?.address)
              ref2["a"]?.logOwnerAddress()
              ref2["b"]?.logOwnerAddress()
          }
      }
    `)

	tx2 := []byte(`
      import Test from 0x1

      transaction {

          prepare(signer: auth(Storage) &Account) {
              let ref1 = signer.storage.borrow<&{String: Test.R}>(from: /storage/rs)!
              log(ref1["a"]?.owner?.address)
              log(ref1["b"]?.owner?.address)
              ref1["a"]?.logOwnerAddress()
              ref1["b"]?.logOwnerAddress()

              let publicAccount = getAccount(0x01)
              let ref2 = publicAccount.capabilities.borrow<&{String: Test.R}>(/public/rs)!
              log(ref2["a"]?.owner?.address)
              log(ref2["b"]?.owner?.address)
              ref2["a"]?.logOwnerAddress()
              ref2["b"]?.logOwnerAddress()
          }
      }
    `)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"nil", "nil",
			"nil", "nil",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
		},
		loggedMessages,
	)

	loggedMessages = nil
	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
			"0x0000000000000001", "0x0000000000000001",
		},
		loggedMessages,
	)
}

func TestRuntimeMetrics(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported1Location := common.StringLocation("imported1")

	importedScript1 := []byte(`
      access(all) fun generate(): [Int] {
        return [1, 2, 3]
      }
    `)

	imported2Location := common.StringLocation("imported2")

	importedScript2 := []byte(`
      access(all) fun getPath(): StoragePath {
        return /storage/foo
      }
    `)

	script1 := []byte(`
      import "imported1"

      transaction {
          prepare(signer: auth(Storage) &Account) {
              signer.storage.save(generate(), to: /storage/foo)
          }
          execute {}
      }
    `)

	script2 := []byte(`
      import "imported2"

      transaction {
          prepare(signer: auth(Storage) &Account) {
              signer.storage.load<[Int]>(from: getPath())
          }
          execute {}
      }
    `)

	storage := NewTestLedger(nil, nil)

	type reports struct {
		programParsed      map[Location]int
		programChecked     map[Location]int
		programInterpreted map[Location]int
	}

	newRuntimeInterface := func() (runtimeInterface Interface, r *reports) {

		r = &reports{
			programParsed:      map[common.Location]int{},
			programChecked:     map[common.Location]int{},
			programInterpreted: map[common.Location]int{},
		}

		runtimeInterface = &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			OnGetCode: func(location Location) (bytes []byte, err error) {
				switch location {
				case imported1Location:
					return importedScript1, nil
				case imported2Location:
					return importedScript2, nil
				default:
					return nil, fmt.Errorf("unknown import location: %s", location)
				}
			},
			OnProgramParsed: func(location common.Location, duration time.Duration) {
				r.programParsed[location]++
			},
			OnProgramChecked: func(location common.Location, duration time.Duration) {
				r.programChecked[location]++
			},
			OnProgramInterpreted: func(location common.Location, duration time.Duration) {
				r.programInterpreted[location]++
			},
		}

		return
	}

	i1, r1 := newRuntimeInterface()

	nextTransactionLocation := NewTransactionLocationGenerator()

	transactionLocation := nextTransactionLocation()
	err := runtime.ExecuteTransaction(
		Script{
			Source: script1,
		},
		Context{
			Interface: i1,
			Location:  transactionLocation,
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
			imported1Location:   1,
		},
		r1.programParsed,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
			imported1Location:   1,
		},
		r1.programChecked,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
		},
		r1.programInterpreted,
	)

	i2, r2 := newRuntimeInterface()

	transactionLocation = nextTransactionLocation()

	err = runtime.ExecuteTransaction(
		Script{
			Source: script2,
		},
		Context{
			Interface: i2,
			Location:  transactionLocation,
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
			imported2Location:   1,
		},
		r2.programParsed,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
			imported2Location:   1,
		},
		r2.programChecked,
	)
	assert.Equal(t,
		map[common.Location]int{
			transactionLocation: 1,
		},
		r2.programInterpreted,
	)
}

type ownerKeyPair struct {
	owner, key []byte
}

func (w ownerKeyPair) String() string {
	return string(w.key)
}

func TestRuntimeContractWriteback(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
      access(all) contract Test {

          access(all) var test: Int

          init() {
              self.test = 1
          }

		  access(all) fun setTest(_ test: Int) {
			self.test = test
		  }
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	readTx := []byte(`
      import Test from 0xCADE

       transaction {

          prepare(signer: &Account) {
              log(Test.test)
          }
       }
    `)

	writeTx := []byte(`
      import Test from 0xCADE

       transaction {

          prepare(signer: &Account) {
              Test.setTest(2)
          }
       }
    `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string
	var writes []ownerKeyPair

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, ownerKeyPair{
			owner,
			key,
		})
	}

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, onWrite),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) (err error) {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	assert.Equal(t,
		[]ownerKeyPair{
			// storage index to contract domain storage map
			{
				addressValue[:],
				[]byte("contract"),
			},
			// contract value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
			// contract domain storage map
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: readTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Empty(t, writes)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: writeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]ownerKeyPair{
			// contract value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
		},
		writes,
	)
}

func TestRuntimeStorageWriteback(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
      access(all) contract Test {

          access(all) resource R {

              access(all) var test: Int

              init() {
                  self.test = 1
              }

			  access(all) fun setTest(_ test: Int) {
				self.test = test
			  }
          }


          access(all) fun createR(): @R {
              return <-create R()
          }
      }
    `)

	deploy := DeploymentTransaction("Test", contract)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string
	var writes []ownerKeyPair

	onWrite := func(owner, key, _ []byte) {
		writes = append(writes, ownerKeyPair{
			owner,
			key,
		})
	}

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, onWrite),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	assert.Equal(t,
		[]ownerKeyPair{
			// storage index to contract domain storage map
			{
				addressValue[:],
				[]byte("contract"),
			},
			// contract value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
			// contract domain storage map
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              import Test from 0xCADE

               transaction {

                  prepare(signer: auth(Storage) &Account) {
                      signer.storage.save(<-Test.createR(), to: /storage/r)
                  }
               }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]ownerKeyPair{
			// storage index to storage domain storage map
			{
				addressValue[:],
				[]byte("storage"),
			},
			// resource value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3},
			},
			// storage domain storage map
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4},
			},
		},
		writes,
	)

	readTx := []byte(`
     import Test from 0xCADE

      transaction {

         prepare(signer: auth(Storage) &Account) {
             log(signer.storage.borrow<&Test.R>(from: /storage/r)!.test)
         }
      }
    `)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: readTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Empty(t, writes)

	writeTx := []byte(`
     import Test from 0xCADE

      transaction {

         prepare(signer: auth(Storage) &Account) {
             let r = signer.storage.borrow<&Test.R>(from: /storage/r)!
             r.setTest(2)
         }
      }
    `)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: writeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]ownerKeyPair{
			// resource value
			{
				addressValue[:],
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3},
			},
		},
		writes,
	)
}

type logPanicError struct{}

func (logPanicError) Error() string {
	return ""
}

var _ error = logPanicError{}

func TestRuntimeExternalError(t *testing.T) {

	t.Parallel()

	interpreterRuntime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          log("ok")
        }
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		OnProgramLog: func(message string) {
			panic(logPanicError{})
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := interpreterRuntime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)

	assertRuntimeErrorIsExternalError(t, err)
}

func TestRuntimeExternalNonError(t *testing.T) {

	t.Parallel()

	interpreterRuntime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare() {
          log("ok")
        }
      }
    `)

	type logPanic struct{}

	runtimeInterface := &TestRuntimeInterface{
		OnGetSigningAccounts: func() ([]Address, error) {
			return nil, nil
		},
		OnProgramLog: func(message string) {
			panic(logPanic{})
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := interpreterRuntime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)

	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.ErrorAs(t, innerError, &runtimeErrors.ExternalNonError{})
}

func TestRuntimeDeployCodeCaching(t *testing.T) {

	t.Parallel()

	const helloWorldContract = `
      access(all) contract HelloWorld {

          access(all) let greeting: String

          init() {
              self.greeting = "Hello, World!"
          }

          access(all) fun hello(): String {
              return self.greeting
          }
      }
    `

	const callHelloTxTemplate = `
        import HelloWorld from 0x%s

        transaction {
            prepare(signer: &Account) {
                assert(HelloWorld.hello() == "Hello, World!")
            }
        }
    `

	createAccountTx := []byte(`
        transaction {
            prepare(signer: auth(BorrowValue) &Account) {
                Account(payer: signer)
            }
        }
    `)

	deployTx := DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

	runtime := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	var accountCounter uint8 = 0

	var signerAddresses []Address

	runtimeInterface := &TestRuntimeInterface{
		OnCreateAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// create the account

	signerAddresses = []Address{{accountCounter}}

	err := runtime.ExecuteTransaction(
		Script{
			Source: createAccountTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// deploy the contract

	signerAddresses = []Address{{accountCounter}}

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// call the hello function

	callTx := []byte(fmt.Sprintf(callHelloTxTemplate, Address{accountCounter}))

	err = runtime.ExecuteTransaction(
		Script{
			Source: callTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeUpdateCodeCaching(t *testing.T) {

	t.Parallel()

	const helloWorldContract1 = `
      access(all) contract HelloWorld {

          access(all) fun hello(): String {
              return "1"
          }
      }
    `

	const helloWorldContract2 = `
      access(all) contract HelloWorld {

          access(all) fun hello(): String {
              return "2"
          }
      }
    `

	const callHelloScriptTemplate = `
        import HelloWorld from 0x%s

        access(all) fun main(): String {
            return HelloWorld.hello()
        }
    `

	const callHelloTransactionTemplate = `
        import HelloWorld from 0x%s

        transaction {
            prepare(signer: &Account) {
                log(HelloWorld.hello())
            }
        }
    `

	createAccountTx := []byte(`
        transaction {
            prepare(signer: auth(BorrowValue) &Account) {
                Account(payer: signer)
            }
        }
    `)

	deployTx := DeploymentTransaction("HelloWorld", []byte(helloWorldContract1))
	updateTx := UpdateTransaction("HelloWorld", []byte(helloWorldContract2))

	runtime := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programHits []string

	runtimeInterface := &TestRuntimeInterface{
		OnCreateAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	// create the account

	signerAddresses = []Address{{accountCounter}}

	err := runtime.ExecuteTransaction(
		Script{
			Source: createAccountTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// deploy the contract

	programHits = nil

	signerAddresses = []Address{{accountCounter}}

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Empty(t, programHits)

	location := common.AddressLocation{
		Address: signerAddresses[0],
		Name:    "HelloWorld",
	}

	require.NotContains(t, runtimeInterface.Programs, location)

	// call the initial hello function

	callScript := []byte(fmt.Sprintf(callHelloScriptTemplate, Address{accountCounter}))

	result1, err := runtime.ExecuteScript(
		Script{
			Source: callScript,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, cadence.String("1"), result1)

	// The deployed hello world contract was imported,
	// assert that it was stored in the program storage
	// after it was parsed and checked

	initialProgram := runtimeInterface.Programs[location]
	require.NotNil(t, initialProgram)

	// update the contract

	programHits = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: updateTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Empty(t, programHits)

	// Assert that the contract update did NOT change
	// the program in program storage

	require.Same(t,
		initialProgram,
		runtimeInterface.Programs[location],
	)
	require.NotNil(t, runtimeInterface.Programs[location])

	// call the new hello function from a script

	result2, err := runtime.ExecuteScript(
		Script{
			Source: callScript,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, cadence.String("2"), result2)

	// call the new hello function from a transaction

	callTransaction := []byte(fmt.Sprintf(callHelloTransactionTemplate, Address{accountCounter}))

	loggedMessages = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: callTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t,
		[]string{`"2"`},
		loggedMessages,
	)
}

func TestRuntimeProgramsHitForToplevelPrograms(t *testing.T) {

	// We do not want to hit the stored programs for toplevel programs
	// (scripts and transactions) until we have moved the caching layer to Cadence.

	t.Parallel()

	const helloWorldContract = `
      access(all) contract HelloWorld {

          access(all) let greeting: String

          init() {
              self.greeting = "Hello, World!"
          }

          access(all) fun hello(): String {
              return self.greeting
          }
      }
    `

	const callHelloTxTemplate = `
        import HelloWorld from 0x%s

        transaction {
            prepare(signer: &Account) {
                assert(HelloWorld.hello() == "Hello, World!")
            }
        }
    `

	createAccountTx := []byte(`
        transaction {
            prepare(signer: auth(BorrowValue) &Account) {
                Account(payer: signer)
            }
        }
    `)

	deployTx := DeploymentTransaction("HelloWorld", []byte(helloWorldContract))

	runtime := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event

	programs := map[common.Location]*interpreter.Program{}

	var accountCounter uint8 = 0

	var signerAddresses []Address

	var programsHits []Location

	runtimeInterface := &TestRuntimeInterface{
		OnCreateAccount: func(payer Address) (address Address, err error) {
			accountCounter++
			return Address{accountCounter}, nil
		},
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnGetAndSetProgram: func(
			location Location,
			load func() (*interpreter.Program, error),
		) (
			program *interpreter.Program,
			err error,
		) {
			programsHits = append(programsHits, location)

			var ok bool
			program, ok = programs[location]
			if ok {
				return
			}

			program, err = load()

			// NOTE: important: still set empty program,
			// even if error occurred

			programs[location] = program

			return
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return signerAddresses, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	signerAddresses = []Address{{accountCounter}}

	// create the account

	err := runtime.ExecuteTransaction(
		Script{
			Source: createAccountTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	signerAddresses = []Address{{accountCounter}}

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// call the function

	callTx := []byte(fmt.Sprintf(callHelloTxTemplate, Address{accountCounter}))

	err = runtime.ExecuteTransaction(
		Script{
			Source: callTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		[]common.Location{
			common.TransactionLocation{
				0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			common.TransactionLocation{
				0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			common.TransactionLocation{
				0x3, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			},
			common.AddressLocation{
				Address: Address{0x1},
				Name:    "HelloWorld",
			},
			common.AddressLocation{
				Address: Address{0x1},
				Name:    "HelloWorld",
			},
		},
		programsHits,
	)
}

func TestRuntimeTransaction_ContractUpdate(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	const contract1 = `
      access(all) contract Test {

          access(all) resource R {

              access(all) let name: String

              init(name: String) {
                  self.name = name
              }

              access(all) fun hello(): Int {
                  return 1
              }
          }

          access(all) var rs: @{String: R}

          access(all) fun hello(): Int {
              return 1
          }

          init() {
              self.rs <- {}
              self.rs["r1"] <-! create R(name: "1")
          }
      }
    `

	const contract2 = `
      access(all) contract Test {

          access(all) resource R {

              access(all) let name: String

              init(name: String) {
                  self.name = name
              }

              access(all) fun hello(): Int {
                  return 2
              }
          }

          access(all) var rs: @{String: R}

          access(all) fun hello(): Int {
              return 2
          }

          init() {
              self.rs <- {}
              panic("should never be executed")
          }
      }
    `

	var accountCode []byte
	var events []cadence.Event

	signerAddress := common.MustBytesToAddress([]byte{0x42})

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		OnResolveLocation: func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
			require.Empty(t, identifiers)
			require.IsType(t, common.AddressLocation{}, location)

			return []ResolvedLocation{
				{
					Location: common.AddressLocation{
						Address: location.(common.AddressLocation).Address,
						Name:    "Test",
					},
					Identifiers: []ast.Identifier{
						{
							Identifier: "Test",
						},
					},
				},
			}, nil
		},
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy the Test contract

	deployTx1 := DeploymentTransaction("Test", []byte(contract1))

	err := runtime.ExecuteTransaction(
		Script{
			Source: deployTx1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	location := common.AddressLocation{
		Address: signerAddress,
		Name:    "Test",
	}

	require.NotContains(t, runtimeInterface.Programs, location)

	// Use the Test contract

	script1 := []byte(`
      import 0x42

      access(all) fun main() {
          // Check stored data

          assert(Test.rs.length == 1)
          assert(Test.rs["r1"]?.name == "1")

          // Check functions

          assert(Test.rs["r1"]?.hello() == 1)
          assert(Test.hello() == 1)
      }
    `)

	nextScriptLocation := NewScriptLocationGenerator()

	_, err = runtime.ExecuteScript(
		Script{
			Source: script1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	// The deployed hello world contract was imported,
	// assert that it was stored in the program storage
	// after it was parsed and checked

	initialProgram := runtimeInterface.Programs[location]
	require.NotNil(t, initialProgram)

	// Update the Test contract

	deployTx2 := UpdateTransaction("Test", []byte(contract2))

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Assert that the contract update did NOT change
	// the program in program storage

	require.Same(t,
		initialProgram,
		runtimeInterface.Programs[location],
	)
	require.NotNil(t, runtimeInterface.Programs[location])

	// Use the new Test contract

	script2 := []byte(`
      import 0x42

      access(all) fun main() {
          // Existing data is still available and the same as before

          assert(Test.rs.length == 1)
          assert(Test.rs["r1"]?.name == "1")

          // New function code is executed.
          // Compare with script1 above, which checked 1.

          assert(Test.rs["r1"]?.hello() == 2)
          assert(Test.hello() == 2)
      }
    `)

	_, err = runtime.ExecuteScript(
		Script{
			Source: script2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeExecuteScriptArguments(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) fun main(num: Int) {}
    `)

	type testCase struct {
		name      string
		arguments [][]byte
		valid     bool
	}

	test := func(tc testCase) {
		t.Run(tc.name, func(t *testing.T) {

			// NOTE: to parallelize this sub-test,
			// access to `programs` must be made thread-safe first

			storage := NewTestLedger(nil, nil)

			runtimeInterface := &TestRuntimeInterface{
				Storage: storage,
				OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
					return json.Decode(nil, b)
				},
			}

			_, err := runtime.ExecuteScript(
				Script{
					Source:    script,
					Arguments: tc.arguments,
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{0x1},
				},
			)

			if tc.valid {
				require.NoError(t, err)
			} else {
				RequireError(t, err)

				assertRuntimeErrorIsUserError(t, err)

				require.ErrorAs(t, err, &InvalidEntryPointParameterCountError{})
			}
		})
	}

	for _, testCase := range []testCase{
		{
			name:      "too few arguments",
			arguments: [][]byte{},
			valid:     false,
		},
		{
			name: "correct number of arguments",
			arguments: encodeArgs([]cadence.Value{
				cadence.NewInt(1),
			}),
			valid: true,
		},
		{
			name: "too many arguments",
			arguments: encodeArgs([]cadence.Value{
				cadence.NewInt(1),
				cadence.NewInt(2),
			}),
			valid: false,
		},
	} {
		test(testCase)
	}
}

func TestRuntimePanics(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) fun main() {
        [1][1]
      }
    `)

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

}

func TestRuntimeAccountsInDictionary(t *testing.T) {

	t.Parallel()

	t.Run("store auth account reference", func(t *testing.T) {
		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all) fun main() {
              let dict: {Int: &Account} = {}
              let ref = &dict as auth(Mutate) &{Int: AnyStruct}
              ref[0] = getAuthAccount<auth(Storage) &Account>(0x01) as AnyStruct
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

		require.NoError(t, err)
	})

	t.Run("invalid: public account reference stored as auth account reference", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all) fun main() {
              let dict: {Int: auth(Storage) &Account} = {}
              let ref = &dict as auth(Mutate) &{Int: AnyStruct}
              ref[0] = getAccount(0x01) as AnyStruct
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

		assertRuntimeErrorIsUserError(t, err)

		var typeErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &typeErr)
	})

	t.Run("public account reference storage as public account reference", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all) fun main() {
              let dict: {Int: &Account} = {}
              let ref = &dict as auth(Mutate) &{Int: AnyStruct}
              ref[0] = getAccount(0x01) as AnyStruct
          }
        `)

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)
	})
}

func TestRuntimeStackOverflow(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	const contract = `

        access(all) contract Recurse {

            access(self) fun recurse() {
                self.recurse()
            }

            init() {
                self.recurse()
            }
        }
    `

	deployTx := DeploymentTransaction("Recurse", []byte(contract))

	var events []cadence.Event
	var loggedMessages []string
	var signerAddress common.Address
	accountCodes := map[common.Location]string{}

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = []byte(accountCodes[location])
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy

	err := runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	assertRuntimeErrorIsUserError(t, err)

	var callStackLimitExceededErr CallStackLimitExceededError
	require.ErrorAs(t, err, &callStackLimitExceededErr)
}

func TestRuntimeInternalErrors(t *testing.T) {

	t.Parallel()

	t.Run("script with go error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          access(all) fun main() {
              log("hello")
          }
        `)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnProgramLog: func(message string) {
				// panic due to go-error in cadence implementation
				var val any = message
				_ = val.(int)
			},
		}

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

		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("script with cadence error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          access(all) fun main() {
              log("hello")
          }
        `)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnProgramLog: func(message string) {
				// intentionally panic
				panic(fmt.Errorf("panic trying to log %s", message))
			},
		}

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

		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("transaction", func(t *testing.T) {

		t.Parallel()

		script := []byte(`
          transaction {
              prepare() {}
              execute {
                  log("hello")
              }
          }
        `)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnProgramLog: func(message string) {
				// panic due to Cadence implementation error
				var val any = message
				_ = val.(int)
			},
		}

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("contract function", func(t *testing.T) {

		t.Parallel()

		addressValue := Address{
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
		}

		contract := []byte(`
          access(all) contract Test {
              access(all) fun hello() {
                  log("Hello World!")
              }
          }
       `)

		var accountCode []byte

		storage := NewTestLedger(nil, nil)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{addressValue}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
				return accountCode, nil
			},
			OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
				accountCode = code
				return nil
			},
			OnEmitEvent: func(_ cadence.Event) error {
				return nil
			},
			OnProgramLog: func(message string) {
				// panic due to Cadence implementation error
				var val any = message
				_ = val.(int)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		deploy := DeploymentTransaction("Test", contract)
		err := runtime.ExecuteTransaction(
			Script{
				Source: deploy,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		assert.NotNil(t, accountCode)

		_, err = runtime.InvokeContractFunction(
			common.AddressLocation{
				Address: addressValue,
				Name:    "Test",
			},
			"hello",
			nil,
			nil,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("parse and check", func(t *testing.T) {

		t.Parallel()

		script := []byte("access(all) fun test() {}")

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnGetAndSetProgram: func(_ Location, _ func() (*interpreter.Program, error)) (*interpreter.Program, error) {
				panic(errors.New("crash while getting/setting program"))
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		_, err := runtime.ParseAndCheckProgram(
			script,
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("read stored", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: TestLedger{
				OnGetValue: func(owner, key []byte) (value []byte, err error) {
					panic(errors.New("crasher"))
				},
			},
		}

		address, err := common.BytesToAddress([]byte{0x42})
		require.NoError(t, err)

		_, err = runtime.ReadStored(
			address,
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
			Context{
				Interface: runtimeInterface,
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("read linked", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: TestLedger{
				OnGetValue: func(owner, key []byte) (value []byte, err error) {
					panic(errors.New("crasher"))
				},
			},
		}

		address, err := common.BytesToAddress([]byte{0x42})
		require.NoError(t, err)

		_, err = runtime.ReadStored(
			address,
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
			Context{
				Interface: runtimeInterface,
			},
		)

		RequireError(t, err)

		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("panic with non error", func(t *testing.T) {

		t.Parallel()

		script := []byte(`access(all) fun main() {}`)

		runtime := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnMeterMemory: func(usage common.MemoryUsage) error {
				// panic with a non-error type
				panic("crasher")
			},
		}

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

		assertRuntimeErrorIsInternalError(t, err)
	})

}

func TestRuntimeComputationMetring(t *testing.T) {
	t.Parallel()

	type test struct {
		name      string
		code      string
		ok        bool
		hits      uint
		intensity uint
	}

	compLimit := uint(6)

	tests := []test{
		{
			name: "Infinite while loop",
			code: `
          while true {}
        `,
			ok:        false,
			hits:      compLimit,
			intensity: 6,
		},
		{
			name: "Limited while loop",
			code: `
          var i = 0
          while i < 5 {
              i = i + 1
          }
        `,
			ok:        false,
			hits:      compLimit,
			intensity: 6,
		},
		{
			name: "statement + createArray + transferArray + too many for-in loop iterations",
			code: `
          for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] {}
        `,
			ok:        false,
			hits:      compLimit,
			intensity: 15,
		},
		{
			name: "statement + createArray + transferArray + two for-in loop iterations",
			code: `
          for i in [1, 2] {}
        `,
			ok:        true,
			hits:      5,
			intensity: 6,
		},
		{
			name: "statement + functionInvocation + encoding",
			code: `
          acc.storage.save("A quick brown fox jumps over the lazy dog", to:/storage/some_path)
        `,
			ok:        true,
			hits:      3,
			intensity: 88,
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			script := []byte(
				fmt.Sprintf(
					`
                  transaction {
                      prepare(acc: auth(Storage) &Account) {
                          %s
                      }
                  }
                `,
					test.code,
				),
			)

			runtime := NewTestInterpreterRuntime()

			compErr := errors.New("computation exceeded limit")
			var hits, totalIntensity uint
			meterComputationFunc := func(kind common.ComputationKind, intensity uint) error {
				hits++
				totalIntensity += intensity
				if hits >= compLimit {
					return compErr
				}
				return nil
			}

			address := common.MustBytesToAddress([]byte{0x1})

			runtimeInterface := &TestRuntimeInterface{
				Storage: NewTestLedger(nil, nil),
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnMeterComputation: meterComputationFunc,
			}

			nextTransactionLocation := NewTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: script,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			if test.ok {
				require.NoError(t, err)
			} else {
				RequireError(t, err)

				var executionErr Error
				require.ErrorAs(t, err, &executionErr)
				require.ErrorAs(t, err.(Error).Unwrap(), &compErr)
			}

			assert.Equal(t, test.hits, hits)
			assert.Equal(t, test.intensity, totalIntensity)
		})
	}
}

func TestRuntimeImportAnyStruct(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	var loggedMessages []string

	address := common.MustBytesToAddress([]byte{0x1})

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	err := rt.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction(args: [AnyStruct]) {
                prepare(signer: &Account) {}
              }
            `),
			Arguments: [][]byte{
				[]byte(`{"value":[{"value":"0xf8d6e0586b0a20c7","type":"Address"},{"value":{"domain":"private","identifier":"USDCAdminCap-ca258982-c98e-4ef0-adef-7ff80ee96b10"},"type":"Path"}],"type":"Array"}`),
			},
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.TransactionLocation{},
		},
	)
	require.NoError(t, err)
}

// Error needs to be `runtime.Error`, and the inner error should be `errors.UserError`.
func assertRuntimeErrorIsUserError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.True(
		t,
		runtimeErrors.IsUserError(innerError),
		"Expected `UserError`, found `%T`", innerError,
	)
}

// Error needs to be `runtime.Error`, and the inner error should be `errors.InternalError`.
func assertRuntimeErrorIsInternalError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.True(
		t,
		runtimeErrors.IsInternalError(innerError),
		"Expected `InternalError`, found `%T`", innerError,
	)
}

// Error needs to be `runtime.Error`, and the inner error should be `interpreter.ExternalError`.
func assertRuntimeErrorIsExternalError(t *testing.T, err error) {
	var runtimeError Error
	require.ErrorAs(t, err, &runtimeError)

	innerError := runtimeError.Unwrap()
	require.ErrorAs(t, innerError, &runtimeErrors.ExternalError{})
}

func BenchmarkRuntimeScriptNoop(b *testing.B) {

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
	}

	script := Script{
		Source: []byte("access(all) fun main() {}"),
	}

	environment := NewScriptInterpreterEnvironment(Config{})

	context := Context{
		Interface:   runtimeInterface,
		Location:    common.ScriptLocation{},
		Environment: environment,
	}

	require.NotNil(b, stdlib.CryptoChecker())

	runtime := NewTestInterpreterRuntime()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = runtime.ExecuteScript(script, context)
	}
}

func TestRuntimeImportTestStdlib(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	runtimeInterface := &TestRuntimeInterface{}

	_, err := rt.ExecuteScript(
		Script{
			Source: []byte(`
                import Test

                access(all) fun main() {
                    Test.assert(true)
                }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)

	errs := checker.RequireCheckerErrors(t, err, 1)

	notDeclaredErr := &sema.NotDeclaredError{}
	require.ErrorAs(t, errs[0], &notDeclaredErr)
	assert.Equal(t, "Test", notDeclaredErr.Name)
}

func TestRuntimeGetCurrentBlockScript(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	runtimeInterface := &TestRuntimeInterface{}

	_, err := rt.ExecuteScript(
		Script{
			Source: []byte(`
                access(all) fun main(): AnyStruct {
                    return getCurrentBlock()
                }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)

	var subErr *ValueNotExportableError
	require.ErrorAs(t, err, &subErr)
}

func TestRuntimeTypeMismatchErrorMessage(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	contract := []byte(`
      access(all) contract Foo {
         access(all) struct Bar {}
      }
    `)

	deploy := DeploymentTransaction("Foo", contract)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	signerAccount := address1

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	// Deploy same contract to two different accounts

	for _, address := range []Address{address1, address2} {
		signerAccount = address

		err := runtime.ExecuteTransaction(
			Script{
				Source: deploy,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	}

	// Set up account

	setupTransaction := []byte(`
      import Foo from 0x1

      transaction {

          prepare(acct: auth(Storage) &Account) {
              acct.storage.save(Foo.Bar(), to: /storage/bar)
          }
      }
    `)

	signerAccount = address1

	err := runtime.ExecuteTransaction(
		Script{
			Source: setupTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Use wrong type

	script := []byte(`
      import Foo from 0x2

      access(all) fun main() {
          getAuthAccount<auth(Storage) &Account>(0x1)
              .storage.borrow<&Foo.Bar>(from: /storage/bar)
      }
    `)

	_, err = runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	RequireError(t, err)

	require.ErrorContains(t, err, "expected type `A.0000000000000002.Foo.Bar`, got `A.0000000000000001.Foo.Bar`")

}

func TestRuntimeErrorExcerpts(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	script := []byte(`
    access(all) fun main(): Int {
        // fill lines so the error occurs on lines 9 and 10
        //
        //
        //
        //
        let a = [1,2,3,4]
        return a
            .firstIndex(of: 5)!
    }
    `)

	runtimeInterface := &TestRuntimeInterface{
		OnGetAccountBalance:          noopRuntimeUInt64Getter,
		OnGetAccountAvailableBalance: noopRuntimeUInt64Getter,
		OnGetStorageUsed:             noopRuntimeUInt64Getter,
		OnGetStorageCapacity:         noopRuntimeUInt64Getter,
		OnAccountKeysCount:           noopRuntimeUInt64Getter,
		Storage:                      NewTestLedger(nil, nil),
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)

	errorString := `Execution failed:
error: unexpectedly found nil while forcing an Optional value
  --> 0000000000000000000000000000000000000000000000000000000000000000:9:15
   |
 9 |         return a
10 |             .firstIndex(of: 5)!
   |                ^^^^^^^^^^^^^^^^
`

	require.Equal(t, errorString, err.Error())
}

func TestRuntimeErrorExcerptsMultiline(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	script := []byte(`
    access(all) fun main(): String {
        // fill lines so the error occurs on lines 9 and 10
        //
        //
        //
        //
        let a = [1,2,3,4]
        return a
            .firstIndex(of: 5)
                ?.toString()!
    }
    `)

	runtimeInterface := &TestRuntimeInterface{
		OnGetAccountBalance:          noopRuntimeUInt64Getter,
		OnGetAccountAvailableBalance: noopRuntimeUInt64Getter,
		OnGetStorageUsed:             noopRuntimeUInt64Getter,
		OnGetStorageCapacity:         noopRuntimeUInt64Getter,
		OnAccountKeysCount:           noopRuntimeUInt64Getter,
		Storage:                      NewTestLedger(nil, nil),
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	RequireError(t, err)

	errorString := `Execution failed:
error: unexpectedly found nil while forcing an Optional value
  --> 0000000000000000000000000000000000000000000000000000000000000000:9:15
   |
 9 |         return a
10 |             .firstIndex(of: 5)
11 |                 ?.toString()!
   |                ^^^^^^^^^^^^^^
`

	require.Equal(t, errorString, err.Error())
}

// https://github.com/onflow/cadence/issues/2464
func TestRuntimeAccountTypeEquality(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) fun main(address: Address): AnyStruct {
          let acct = getAuthAccount<auth(Capabilities) &Account>(address)
          let path = /public/tmp

          let cap = acct.capabilities.account.issue<&Account>()
          acct.capabilities.publish(cap, at: path)

          let capType = acct.capabilities.borrow<&Account>(path)!.getType()

          return Type<Account>() == capType
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
		OnEmitEvent: func(_ cadence.Event) error {
			return nil
		},
	}

	result, err := rt.ExecuteScript(
		Script{
			Source: script,
			Arguments: encodeArgs([]cadence.Value{
				cadence.Address(common.MustBytesToAddress([]byte{0x1})),
			}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	require.NoError(t, err)

	require.Equal(t, cadence.Bool(true), result)
}

func TestRuntimeUserPanicToError(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf(
		"wrapped: %w",
		runtimeErrors.NewDefaultUserError("user error"),
	)
	retErr := UserPanicToError(func() { panic(err) })
	require.Equal(t, retErr, err)
}

func TestRuntimeFlowEventTypes(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) fun main(): Type? {
          return CompositeType("flow.AccountContractAdded")
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
	}

	result, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	require.NoError(t, err)

	accountContractAddedType := ExportType(
		stdlib.AccountContractAddedEventType,
		map[sema.TypeID]cadence.Type{},
	)

	require.Equal(t,
		cadence.Optional{
			Value: cadence.TypeValue{
				StaticType: accountContractAddedType,
			},
		},
		result,
	)
}

func TestRuntimeInvalidatedResourceUse(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	signers := []Address{signerAccount}

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) (err error) {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	attacker := []byte(fmt.Sprintf(`
		import VictimContract from %s

		access(all) contract AttackerContract {

			access(all) resource AttackerResource {
				access(all) var vault: @VictimContract.Vault
				access(all) var firstCopy: @VictimContract.Vault

				init(vault: @VictimContract.Vault) {
					self.vault <- vault
					self.firstCopy <- self.vault.withdraw(amount: 0.0)
				}

				access(all) fun shenanigans(): UFix64{
					let fullBalance = self.vault.balance

					var withdrawn <- self.vault.withdraw(amount: 0.0)

					// "Rug pull" the vault from under the in-flight
					// withdrawal and deposit it into our "first copy" wallet
					self.vault <-> withdrawn
					self.firstCopy.deposit(from: <- withdrawn)

					// Return the pre-deposit balance for caller to withdraw
					return fullBalance
				}

				access(all) fun fetchfirstCopy(): @VictimContract.Vault {
					var withdrawn <- self.firstCopy.withdraw(amount: 0.0)
					self.firstCopy <-> withdrawn
					return <- withdrawn
				}
			}

			access(all) fun doubleBalanceOfVault(_ victim: @VictimContract.Vault): @VictimContract.Vault {
				var r <- create AttackerResource(vault: <- victim)

				// The magic happens during the execution of the following line of code
				// var withdrawAmmount = r.shenanigans()
				var secondCopy <- r.vault.withdraw(amount: r.shenanigans())

				// Deposit the second copy of the funds as retained by the AttackerResource instance
				secondCopy.deposit(from: <- r.fetchfirstCopy())

				destroy r
				return <- secondCopy
			}

			access(all) fun attack() {
				var v1 <- VictimContract.faucet()
				var v2<- AttackerContract.doubleBalanceOfVault(<- v1)
				destroy v2
		   }
		}`,
		signerAccount.HexWithPrefix(),
	))

	victim := []byte(`
        access(all) contract VictimContract {
            access(all) resource Vault {

                // Balance of a user's Vault
                // we use unsigned fixed point numbers for balances
                // because they can represent decimals and do not allow negative values
                access(all) var balance: UFix64

                init(balance: UFix64) {
                    self.balance = balance
                }

                access(all) fun withdraw(amount: UFix64): @Vault {
                    self.balance = self.balance - amount
                    return <-create Vault(balance: amount)
                }

                access(all) fun deposit(from: @Vault) {
                    self.balance = self.balance + from.balance
                    destroy from
                }
            }

            access(all) fun faucet(): @VictimContract.Vault {
                return <- create VictimContract.Vault(balance: 5.0)
            }
        }
    `)

	// Deploy Victim

	deployVictim := DeploymentTransaction("VictimContract", victim)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deployVictim,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Deploy Attacker

	deployAttacker := DeploymentTransaction("AttackerContract", attacker)

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployAttacker,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	attackTransaction := []byte(fmt.Sprintf(`
        import VictimContract from %s
        import AttackerContract from %s

        transaction {
            execute {
                AttackerContract.attack()
            }
        }`,
		signerAccount.HexWithPrefix(),
		signerAccount.HexWithPrefix(),
	))

	signers = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTransaction,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})

}

func TestRuntimeOptionalReferenceAttack(t *testing.T) {

	t.Parallel()

	script := `
      access(all) resource Vault {
          access(all) var balance: UFix64

          init(balance: UFix64) {
              self.balance = balance
          }

          access(all) fun withdraw(amount: UFix64): @Vault {
              self.balance = self.balance - amount
              return <-create Vault(balance: amount)
          }

          access(all) fun deposit(from: @Vault) {
              self.balance = self.balance + from.balance
              destroy from
          }
      }

      access(all) fun empty(): @Vault {
          return <- create Vault(balance: 0.0)
      }

      access(all) fun giveme(): @Vault {
          return <- create Vault(balance: 10.0)
      }

      access(all) fun main() {
          var vault <- giveme() //get 10 token
          var someDict:@{Int:Vault} <- {1:<-vault}
          var r = (&someDict[1] as &AnyResource) as! &Vault
          var double <- empty()
          double.deposit(from: <- someDict.remove(key:1)!)
          double.deposit(from: <- r.withdraw(amount:10.0))
          log(double.balance) // 20
          destroy double
          destroy someDict
      }
    `

	runtime := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(s string) {

		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	_, err := runtime.ExecuteScript(
		Script{
			Source:    []byte(script),
			Arguments: [][]byte{},
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	RequireError(t, err)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := checker.RequireCheckerErrors(t, checkerErr, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestRuntimeReturnDestroyedOptional(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      access(all) resource Foo {}

      access(all) fun main(): AnyStruct {
          let y: @Foo? <- create Foo()
          let z: @AnyResource <- y
          var ref = &z as &AnyResource
          ref = returnSameRef(ref)
          destroy z
          return ref
      }

      access(all) fun returnSameRef(_ ref: &AnyResource): &AnyResource {
          return ref
      }
    `)

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
	}

	// Test

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
	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
}

func TestRuntimeComputationMeteringError(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	t.Run("regular error returned", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            access(all) fun main() {
                foo()
            }
        `)

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnMeterComputation: func(compKind common.ComputationKind, intensity uint) error {
				return fmt.Errorf("computation limit exceeded")
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an external error.
		// It can NOT be an internal error.
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("regular error panicked", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            access(all) fun main() {
                foo()
            }
        `)

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnMeterComputation: func(compKind common.ComputationKind, intensity uint) error {
				panic(fmt.Errorf("computation limit exceeded"))
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an external error.
		// It can NOT be an internal error.
		assertRuntimeErrorIsExternalError(t, err)
	})

	t.Run("go runtime error panicked", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            access(all) fun main() {
                foo()
            }
        `)

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnMeterComputation: func(compKind common.ComputationKind, intensity uint) error {
				// Cause a runtime error
				var x any = "hello"
				_ = x.(int)
				return nil
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an internal error.
		assertRuntimeErrorIsInternalError(t, err)
	})

	t.Run("go runtime error returned", func(t *testing.T) {
		t.Parallel()

		script := []byte(`
            access(all) fun foo() {}

            access(all) fun main() {
                foo()
            }
        `)

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnMeterComputation: func(compKind common.ComputationKind, intensity uint) (err error) {
				// Cause a runtime error. Catch it and return.
				var x any = "hello"
				defer func() {
					if r := recover(); r != nil {
						if r, ok := r.(error); ok {
							err = r
						}
					}
				}()

				_ = x.(int)

				return
			},
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.Error(t, err)

		// Returned error MUST be an internal error.
		assertRuntimeErrorIsInternalError(t, err)
	})
}

func TestRuntimeWrappedErrorHandling(t *testing.T) {

	t.Parallel()

	foo := []byte(`
        access(all) contract Foo {
            access(all) resource R {
                access(all) var x: Int

                init() {
                    self.x = 0
                }
            }

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `)

	brokenFoo := []byte(`
        access(all) contract Foo {
            access(all) resource R {
                access(all) var x: Int

                init() {
                    self.x = "hello"
                }
            }

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `)

	tx1 := []byte(`
        import Foo from 0x1

        transaction {
            prepare(signer: auth (Storage, Capabilities) &Account) {
                signer.storage.save(<- Foo.createR(), to: /storage/r)
                let cap = signer.capabilities.storage.issue<&Foo.R>(/storage/r)
                signer.capabilities.publish(cap, at: /public/r)
            }
        }
    `)

	tx2 := []byte(`
        transaction {
            prepare(signer: &Account) {
                let cap = signer.capabilities.get<&AnyStruct>(/public/r)!
				cap.check()
            }
        }
    `)

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		AtreeValidationEnabled: false,
	})

	address := common.MustBytesToAddress([]byte{0x1})

	deploy := DeploymentTransaction("Foo", foo)

	var contractCode []byte
	var events []cadence.Event

	isContractBroken := false

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return contractCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			if isContractBroken && contractCode != nil {
				return brokenFoo, nil
			}
			return contractCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			contractCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnGetAndSetProgram: func(location Location, load func() (*interpreter.Program, error)) (*interpreter.Program, error) {
			program, err := load()
			if err == nil {
				return program, nil
			}
			return program, fmt.Errorf("wrapped error: %w", err)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Run Tx to save values

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx1,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Run Tx to load value.
	// Mark the contract is broken

	isContractBroken = true

	err = runtime.ExecuteTransaction(
		Script{
			Source: tx2,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	RequireError(t, err)

	// Returned error MUST be a user error.
	// It can NOT be an internal error.
	assertRuntimeErrorIsUserError(t, err)
}

func BenchmarkRuntimeResourceTracking(b *testing.B) {

	runtime := NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}

	signerAccount := contractsAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(b),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	environment := NewBaseInterpreterEnvironment(Config{})

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
				"Foo",
				[]byte(`
                    access(all) contract Foo {
                        access(all) resource R {}

                        access(all) fun getResourceArray(): @[R] {
                            var resourceArray: @[R] <- []
                            var i = 0
                            while i < 1000 {
                                resourceArray.append(<- create R())
                                i = i + 1
                            }
                            return <- resourceArray
                        }
                    }
                `),
			),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
                import Foo from 0x1

                transaction {
                    prepare(signer: &Account) {
                        signer.storage.save(<- Foo.getResourceArray(), to: /storage/r)
                    }
                }
            `),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
                import Foo from 0x1

                transaction {
                    prepare(signer: &Account) {
                        // When the array is loaded from storage, all elements are also loaded.
                        // So all moves of this resource will check for tracking of all elements aas well.

                        var array1 <- signer.storage.load<@[Foo.R]>(from: /storage/r)!
                        var array2 <- array1
                        var array3 <- array2
                        var array4 <- array3
                        var array5 <- array4
                        destroy array5
                    }
                }
            `),
		},
		Context{
			Interface:   runtimeInterface,
			Location:    nextTransactionLocation(),
			Environment: environment,
		},
	)
	require.NoError(b, err)
}

func TestRuntimeTypesAndConversions(t *testing.T) {
	t.Parallel()

	test := func(name string, semaType sema.Type) {
		t.Run(name, func(t *testing.T) {

			t.Parallel()

			var staticType interpreter.StaticType

			t.Run("sema -> static", func(t *testing.T) {
				staticType = interpreter.ConvertSemaToStaticType(nil, semaType)
				require.NotNil(t, staticType)
			})

			if staticType != nil {
				t.Run("static -> sema", func(t *testing.T) {

					t.Parallel()

					inter, err := interpreter.NewInterpreter(nil, nil, &interpreter.Config{})
					require.NoError(t, err)

					convertedSemaType, err := inter.ConvertStaticToSemaType(staticType)
					require.NoError(t, err)
					require.True(t, semaType.Equal(convertedSemaType))
				})
			}

			var cadenceType cadence.Type

			t.Run("sema -> cadence", func(t *testing.T) {

				cadenceType = ExportType(semaType, map[sema.TypeID]cadence.Type{})
				require.NotNil(t, cadenceType)
			})

			if cadenceType != nil {

				t.Run("cadence -> static", func(t *testing.T) {

					t.Parallel()

					convertedStaticType := ImportType(nil, cadenceType)
					require.True(t, staticType.Equal(convertedStaticType))
				})
			}
		})
	}

	for name, ty := range checker.AllBaseSemaTypes() {
		test(name, ty)
	}
}

func TestRuntimeEventEmission(t *testing.T) {

	t.Parallel()

	t.Run("primitive", func(t *testing.T) {
		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all)
          event TestEvent(ref: Int)

          access(all)
          fun main() {
              emit TestEvent(ref: 42)
          }
        `)

		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextScriptLocation := NewScriptLocationGenerator()

		location := nextScriptLocation()

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.NoError(t, err)

		require.Len(t, events, 1)
		event := events[0]

		assert.EqualValues(
			t,
			location.TypeID(nil, "TestEvent"),
			event.Type().ID(),
		)

		assert.Equal(
			t,
			[]cadence.Value{
				cadence.NewInt(42),
			},
			event.GetFieldValues(),
		)

	})

	t.Run("reference", func(t *testing.T) {
		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		script := []byte(`
          access(all)
          event TestEvent(ref: &Int)

          access(all)
          fun main() {
              emit TestEvent(ref: &42)
          }
        `)

		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextScriptLocation := NewScriptLocationGenerator()

		location := nextScriptLocation()

		_, err := runtime.ExecuteScript(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  location,
			},
		)
		require.NoError(t, err)

		require.Len(t, events, 1)
		event := events[0]

		assert.EqualValues(
			t,
			location.TypeID(nil, "TestEvent"),
			event.Type().ID(),
		)

		assert.Equal(
			t,
			[]cadence.Value{
				cadence.NewInt(42),
			},
			event.GetFieldValues(),
		)

	})
}

func TestRuntimeInvalidWrappedPrivateCapability(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		AtreeValidationEnabled: false,
	})

	address := common.MustBytesToAddress([]byte{0x1})

	helperContract := []byte(`
      access(all)
      contract Foo {

          access(all)
          struct Thing {

              access(all)
              let cap: Capability

              init(cap: Capability) {
                  self.cap = cap
              }
          }
      }
    `)

	getCapScript := []byte(`
      import Foo from 0x1

      access(all)
      fun main(addr: Address): AnyStruct {
          let acct = getAuthAccount<auth(Capabilities) &Account>(addr)
          let cap = acct.capabilities.account.issue<auth(Storage) &Account>()
          return Foo.Thing(cap: cap)
      }
	`)

	attackTx := []byte(`
      import Foo from 0x1

      transaction(thing: Foo.Thing) {
          prepare(_: &Account) {
 		      thing.cap.borrow<auth(Storage) &Account>()!
                  .storage.save("Hello, World", to: /storage/attack)
          }
      }
    `)

	deploy := DeploymentTransaction("Foo", helperContract)

	var accountCode []byte
	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	// Deploy helper contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Get Capability

	capability, err := runtime.ExecuteScript(
		Script{
			Source: getCapScript,
			Arguments: encodeArgs([]cadence.Value{
				cadence.BytesToAddress([]byte{0x1}),
			}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	// Attack

	err = runtime.ExecuteTransaction(
		Script{
			Source: attackTx,
			Arguments: encodeArgs([]cadence.Value{
				capability,
			}),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	var argumentNotImportableErr *ArgumentNotImportableError
	require.ErrorAs(t, err, &argumentNotImportableErr)
}
