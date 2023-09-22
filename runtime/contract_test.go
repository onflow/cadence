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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeContract(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name        string // the name of the contract used in add/update calls
		code        string // the code we use to add the contract
		code2       string // the code we use to update the contract
		valid       bool
		isInterface bool
	}

	test := func(t *testing.T, tc testCase) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()

		var loggedMessages []string

		signerAddress := Address{0x1}

		var deployedCode []byte

		addTx := []byte(
			fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: auth(AddContract) &Account) {
                          let contract1 = signer.contracts.get(name: %[1]q)
                          log(contract1?.name)
                          log(contract1?.code)

                          let contract2 = signer.contracts.add(name: %[1]q, code: "%[2]s".decodeHex())
                          log(contract2.name)
                          log(contract2.code)

                          let contract3 = signer.contracts.get(name: %[1]q)
                          log(contract3?.name)
                          log(contract3?.code)

                          let contract4 = signer.contracts.get(name: "Unknown")
                          log(contract4)
                      }
                   }
                `,
				tc.name,
				hex.EncodeToString([]byte(tc.code)),
			),
		)

		updateTx := []byte(
			fmt.Sprintf(
				`
                 transaction {
                     prepare(signer: auth(UpdateContract) &Account) {

                         let contract1 = signer.contracts.get(name: %[1]q)
                         log(contract1?.name)
                         log(contract1?.code)

                         let contract2 = signer.contracts.update(name: %[1]q, code: "%[2]s".decodeHex())
                         log(contract2.name)
                         log(contract2.code)

                         let contract3 = signer.contracts.get(name: %[1]q)
                         log(contract3?.name)
                         log(contract3?.code)
                     }
                  }
               `,
				tc.name,
				hex.EncodeToString([]byte(tc.code2)),
			),
		)

		removeTx := []byte(
			fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: auth(RemoveContract) &Account) {
                          let contract1 = signer.contracts.get(name: %[1]q)
                          log(contract1?.name)
                          log(contract1?.code)

                          let contract2 = signer.contracts.remove(name: %[1]q)
                          log(contract2?.name)
                          log(contract2?.code)

                          let contract3 = signer.contracts.get(name: %[1]q)
                          log(contract3)
                      }
                   }
                `,
				tc.name,
			),
		)

		removeAndAddTx := []byte(
			fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: auth(Contracts) &Account) {
                          let contract1 = signer.contracts.get(name: %[1]q)
                          log(contract1?.name)
                          log(contract1?.code)

                          let contract2 = signer.contracts.remove(name: %[1]q)
                          log(contract2?.name)
                          log(contract2?.code)

                          let contract3 = signer.contracts.get(name: %[1]q)
                          log(contract3)

                          let contract4 = signer.contracts.add(name: %[1]q, code: "%[2]s".decodeHex())
                          log(contract4.name)
                          log(contract4.code)

                          let contract5 = signer.contracts.get(name: %[1]q)
                          log(contract5?.name)
                          log(contract5?.code)
                      }
                   }
                `,
				tc.name,
				hex.EncodeToString([]byte(tc.code2)),
			),
		)

		var events []cadence.Event

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{signerAddress}, nil
			},
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				require.Equal(t, tc.name, location.Name)
				assert.Equal(t, signerAddress, location.Address)

				deployedCode = code

				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				if location.Name == tc.name {
					return deployedCode, nil
				}

				return nil, nil
			},
			OnRemoveAccountContractCode: func(location common.AddressLocation) error {
				require.Equal(t, tc.name, location.Name)
				assert.Equal(t, signerAddress, location.Address)

				deployedCode = nil

				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		inter := NewTestInterpreter(t)
		codeArrayString := interpreter.ByteSliceToByteArrayValue(inter, []byte(tc.code)).String()
		code2ArrayString := interpreter.ByteSliceToByteArrayValue(inter, []byte(tc.code2)).String()

		// For each check, we always need to create a new runtime storage instance
		// and get the storage map (which is backed by an atree ordered map),
		// because we want to get the latest view / updates of the map –
		// the runtime creates storage maps internally and modifies them,
		// so getting the storage map here once upfront would result in outdated data

		getContractValueExists := func() bool {
			storageMap := NewStorage(storage, nil).
				GetStorageMap(signerAddress, StorageDomainContract, false)
			if storageMap == nil {
				return false
			}
			return storageMap.ValueExists(interpreter.StringStorageMapKey("Test"))
		}

		t.Run("add", func(t *testing.T) {

			err := runtime.ExecuteTransaction(
				Script{
					Source:    addTx,
					Arguments: nil,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)

			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, []byte(tc.code), deployedCode)

				contractValueExists := getContractValueExists()

				if tc.isInterface {
					require.False(t, contractValueExists)
				} else {
					require.True(t, contractValueExists)
				}

				require.Equal(t,
					[]string{
						`nil`,
						`nil`,
						`"Test"`,
						codeArrayString,
						`"Test"`,
						codeArrayString,
						`nil`,
					},
					loggedMessages,
				)

				require.Len(t, events, 1)
				assert.EqualValues(t, stdlib.AccountContractAddedEventType.ID(), events[0].Type().ID())

			} else {
				RequireError(t, err)

				require.Empty(t, deployedCode)
				require.Empty(t, events)
				require.Equal(t,
					[]string{
						`nil`,
						`nil`,
					},
					loggedMessages,
				)
				contractValueExists := getContractValueExists()
				require.False(t, contractValueExists)
			}
		})

		if !tc.valid {
			return
		}

		t.Run("re-add", func(t *testing.T) {

			// Re-run the addition transaction, ensure that overwriting is not possible

			loggedMessages = nil
			events = nil

			err := runtime.ExecuteTransaction(
				Script{
					Source: addTx,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			RequireError(t, err)

			// the deployed code should not have been updated,
			// and no events should have been emitted,
			// as the deployment should fail

			require.NotEmpty(t, deployedCode)
			require.Empty(t, events)

		})

		t.Run("update", func(t *testing.T) {

			// Run the update transaction

			loggedMessages = nil
			events = nil

			err := runtime.ExecuteTransaction(
				Script{
					Source: updateTx,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			require.Equal(t, []byte(tc.code2), deployedCode)

			contractValueExists := getContractValueExists()

			if tc.isInterface {
				require.False(t, contractValueExists)
			} else {
				require.True(t, contractValueExists)
			}

			require.Equal(t,
				[]string{
					`"Test"`,
					codeArrayString,
					`"Test"`,
					code2ArrayString,
					`"Test"`,
					code2ArrayString,
				},
				loggedMessages,
			)

			require.Len(t, events, 1)
			assert.EqualValues(t, stdlib.AccountContractUpdatedEventType.ID(), events[0].Type().ID())
		})

		t.Run("remove", func(t *testing.T) {

			// Run the removal transaction

			loggedMessages = nil
			events = nil

			err := runtime.ExecuteTransaction(
				Script{
					Source: removeTx,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			require.Empty(t, deployedCode)

			require.Equal(t,
				[]string{
					`"Test"`,
					code2ArrayString,
					`"Test"`,
					code2ArrayString,
					`nil`,
				},
				loggedMessages,
			)

			require.Len(t, events, 1)
			assert.EqualValues(t, stdlib.AccountContractRemovedEventType.ID(), events[0].Type().ID())

			contractValueExists := getContractValueExists()
			require.False(t, contractValueExists)

		})

		t.Run("add again", func(t *testing.T) {

			// Run the add transaction again

			loggedMessages = nil
			events = nil

			err := runtime.ExecuteTransaction(
				Script{
					Source:    addTx,
					Arguments: nil,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)

			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, []byte(tc.code), deployedCode)

				contractValueExists := getContractValueExists()

				if tc.isInterface {
					require.False(t, contractValueExists)
				} else {
					require.True(t, contractValueExists)
				}

				require.Equal(t,
					[]string{
						`nil`,
						`nil`,
						`"Test"`,
						codeArrayString,
						`"Test"`,
						codeArrayString,
						`nil`,
					},
					loggedMessages,
				)

				require.Len(t, events, 1)
				assert.EqualValues(t, stdlib.AccountContractAddedEventType.ID(), events[0].Type().ID())

			} else {
				RequireError(t, err)

				require.Empty(t, deployedCode)
				require.Empty(t, events)
				require.Empty(t, loggedMessages)

				contractValueExists := getContractValueExists()
				require.False(t, contractValueExists)
			}
		})

		t.Run("remove and add in same transaction", func(t *testing.T) {

			// Run the remove-and-add transaction

			loggedMessages = nil
			events = nil

			err := runtime.ExecuteTransaction(
				Script{
					Source: removeAndAddTx,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			require.Equal(t, []byte(tc.code2), deployedCode)

			require.Equal(t,
				[]string{
					`"Test"`,
					codeArrayString,
					`"Test"`,
					codeArrayString,
					`nil`,
					`"Test"`,
					code2ArrayString,
					`"Test"`,
					code2ArrayString,
				},
				loggedMessages,
			)

			require.Len(t, events, 2)
			assert.EqualValues(t, stdlib.AccountContractRemovedEventType.ID(), events[0].Type().ID())
			assert.EqualValues(t, stdlib.AccountContractAddedEventType.ID(), events[1].Type().ID())

			contractValueExists := getContractValueExists()

			if tc.isInterface {
				require.False(t, contractValueExists)
			} else {
				require.True(t, contractValueExists)
			}
		})

	}

	t.Run("valid contract, correct name", func(t *testing.T) {
		test(t, testCase{
			name:        "Test",
			code:        `access(all) contract Test {}`,
			code2:       `access(all) contract Test { access(all) fun test() {} }`,
			valid:       true,
			isInterface: false,
		})
	})

	t.Run("valid contract interface, correct name", func(t *testing.T) {
		test(t, testCase{
			name:        "Test",
			code:        `access(all) contract interface Test {}`,
			code2:       `access(all) contract interface Test { access(all) fun test() }`,
			valid:       true,
			isInterface: true,
		})
	})

	t.Run("valid contract, wrong name", func(t *testing.T) {
		test(t, testCase{
			name:        "XYZ",
			code:        `access(all) contract Test {}`,
			valid:       false,
			isInterface: false,
		})
	})

	t.Run("valid contract interface, wrong name", func(t *testing.T) {
		test(t, testCase{
			name:        "XYZ",
			code:        `access(all) contract interface Test {}`,
			valid:       false,
			isInterface: true,
		})
	})

	t.Run("invalid code", func(t *testing.T) {
		test(t, testCase{
			name:        "Test",
			code:        `foo`,
			valid:       false,
			isInterface: false,
		})
	})

	t.Run("missing contract or contract interface", func(t *testing.T) {
		test(t, testCase{
			name:        "Test",
			code:        ``,
			valid:       false,
			isInterface: false,
		})
	})

	t.Run("two contracts", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              access(all) contract Test {}

              access(all) contract Test2 {}
            `,
			valid:       false,
			isInterface: false,
		})
	})

	t.Run("two contract interfaces", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              access(all) contract interface Test {}

              access(all) contract interface Test2 {}
            `,
			valid:       false,
			isInterface: true,
		})
	})

	t.Run("contract and contract interface", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              access(all) contract Test {}

              access(all) contract interface Test2 {}
            `,
			valid:       false,
			isInterface: false,
		})
	})
}

func TestRuntimeImportMultipleContracts(t *testing.T) {

	t.Parallel()

	contractA := `
      access(all) contract A {

          access(all) fun a(): Int {
              return 1
          }
      }
    `

	contractB := `
      access(all) contract B {

          access(all) fun b(): Int {
              return 2
          }
      }
    `

	contractC := `
      import A, B from 0x1

      access(all) contract C {

          access(all) fun c(): Int {
              return A.a() + B.b()
          }
      }
    `

	addTx := func(name, code string) []byte {
		return []byte(
			fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: auth(Contracts) &Account) {
                          signer.contracts.add(name: %[1]q, code: "%[2]s".decodeHex())
                      }
                   }
                `,
				name,
				hex.EncodeToString([]byte(code)),
			),
		)
	}

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnRemoveAccountContractCode: func(location common.AddressLocation) error {
			delete(accountCodes, location)
			return nil
		},
		OnResolveLocation: MultipleIdentifierLocationResolver,
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	runtime := NewTestInterpreterRuntime()

	nextTransactionLocation := NewTransactionLocationGenerator()

	for _, contract := range []struct{ name, code string }{
		{"A", contractA},
		{"B", contractB},
		{"C", contractC},
	} {
		tx := addTx(contract.name, contract.code)
		err := runtime.ExecuteTransaction(
			Script{
				Source: tx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			})
		require.NoError(t, err)
	}

	t.Run("use A", func(t *testing.T) {
		tx := []byte(`
          import A from 0x1

          transaction {
              prepare(signer: &Account) {
                  log(A.a())
              }
          }
        `)

		loggedMessages = nil

		err := runtime.ExecuteTransaction(
			Script{
				Source: tx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	})

	t.Run("use B", func(t *testing.T) {
		tx := []byte(`
         import B from 0x1

         transaction {
             prepare(signer: &Account) {
                 log(B.b())
             }
         }
       `)

		loggedMessages = nil

		err := runtime.ExecuteTransaction(
			Script{
				Source: tx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	})

	t.Run("use C", func(t *testing.T) {
		tx := []byte(`
          import C from 0x1

          transaction {
              prepare(signer: &Account) {
                  log(C.c())
              }
          }
        `)

		loggedMessages = nil

		err := runtime.ExecuteTransaction(
			Script{
				Source: tx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	})
}

func TestRuntimeContractInterfaceEventEmission(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployInterfaceTx := DeploymentTransaction("TestInterface", []byte(`
		access(all) contract interface TestInterface {
			access(all) event Foo(x: Int)

			access(all) fun foo() {
				emit Foo(x: 3)
			}
		}
	`))

	deployTx := DeploymentTransaction("TestContract", []byte(`
		import TestInterface from 0x1
		access(all) contract TestContract: TestInterface {
			access(all) event Foo(x: String, y: Int)

			access(all) fun bar() {
				emit Foo(x: "", y: 2)
			}
		}
	`))

	transaction1 := []byte(`
		import TestContract from 0x1
		transaction {
			prepare(signer: &Account) {
				TestContract.foo()
				TestContract.bar()
			}
		}
	 `)

	var actualEvents []cadence.Event

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
			actualEvents = append(actualEvents, event)
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: deployInterfaceTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// first two events are `AccountContractAdded`
	require.Len(t, actualEvents, 4)

	intfEvent := actualEvents[2]
	concreteEvent := actualEvents[3]

	require.Equal(t, intfEvent.EventType.QualifiedIdentifier, "TestInterface.Foo")
	require.Equal(t, concreteEvent.EventType.QualifiedIdentifier, "TestContract.Foo")

	require.Len(t, intfEvent.Fields, 1)
	require.Len(t, concreteEvent.Fields, 2)

	require.Equal(t, intfEvent.Fields[0], cadence.NewInt(3))
	require.Equal(t, concreteEvent.Fields[0], cadence.String(""))
	require.Equal(t, concreteEvent.Fields[1], cadence.NewInt(2))
}

func TestRuntimeContractInterfaceConditionEventEmission(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployInterfaceTx := DeploymentTransaction("TestInterface", []byte(`
		access(all)
        contract interface TestInterface {

			access(all)
            event Foo(x: Int)

			access(all) fun bar() {
				post {
					emit Foo(x: 3)
				}
			}
		}
	`))

	deployTx := DeploymentTransaction("TestContract", []byte(`
		import TestInterface from 0x1

		access(all)
        contract TestContract: TestInterface {

			access(all)
            event Foo(x: String, y: Int)

			access(all)
            fun bar() {
				emit Foo(x: "", y: 2)
			}
		}
	`))

	transaction1 := []byte(`
		import TestContract from 0x1

		transaction {
			prepare(signer: &Account) {
				TestContract.bar()
			}
		}
	 `)

	var actualEvents []cadence.Event

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
			actualEvents = append(actualEvents, event)
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: deployInterfaceTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// first two events are `AccountContractAdded`
	require.Len(t, actualEvents, 4)

	intfEvent := actualEvents[3]
	concreteEvent := actualEvents[2]

	require.Equal(t, intfEvent.EventType.QualifiedIdentifier, "TestInterface.Foo")
	require.Equal(t, concreteEvent.EventType.QualifiedIdentifier, "TestContract.Foo")

	require.Len(t, intfEvent.Fields, 1)
	require.Len(t, concreteEvent.Fields, 2)

	require.Equal(t, intfEvent.Fields[0], cadence.NewInt(3))
	require.Equal(t, concreteEvent.Fields[0], cadence.String(""))
	require.Equal(t, concreteEvent.Fields[1], cadence.NewInt(2))
}

func TestRuntimeContractTryUpdate(t *testing.T) {
	t.Parallel()

	newTestRuntimeInterface := func(onUpdate func()) *TestRuntimeInterface {
		var actualEvents []cadence.Event
		storage := NewTestLedger(nil, nil)
		accountCodes := map[Location][]byte{}

		return &TestRuntimeInterface{
			Storage:      storage,
			OnProgramLog: func(message string) {},
			OnEmitEvent: func(event cadence.Event) error {
				actualEvents = append(actualEvents, event)
				return nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
			},
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				onUpdate()
				accountCodes[location] = code
				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
		}
	}

	t.Run("tryUpdate simple", func(t *testing.T) {

		t.Parallel()

		rt := NewTestInterpreterRuntime()

		deployTx := DeploymentTransaction("Foo", []byte(`access(all) contract Foo {}`))

		updateTx := []byte(`
			transaction {
				prepare(signer: auth(UpdateContract) &Account) {
					let code = "access(all) contract Foo { access(all) fun sayHello(): String {return \"hello\"} }".utf8

					let deploymentResult = signer.contracts.tryUpdate(
						name: "Foo",
						code: code,
					)

					let deployedContract = deploymentResult.deployedContract!
					assert(deployedContract.name == "Foo")
					assert(deployedContract.address == 0x1)
					assert(deployedContract.code == code)
				}
			}
		`)

		invokeTx := []byte(`
			import Foo from 0x1

			transaction {
				prepare(signer: &Account) {
					assert(Foo.sayHello() == "hello")
				}
			}
		`)

		runtimeInterface := newTestRuntimeInterface(func() {})
		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy 'Foo'
		err := rt.ExecuteTransaction(
			Script{
				Source: deployTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Update 'Foo'
		err = rt.ExecuteTransaction(
			Script{
				Source: updateTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Test the updated 'Foo'
		err = rt.ExecuteTransaction(
			Script{
				Source: invokeTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
	})

	t.Run("tryUpdate non existing", func(t *testing.T) {

		t.Parallel()

		rt := NewTestInterpreterRuntime()

		updateTx := []byte(`
			transaction {
				prepare(signer: auth(UpdateContract) &Account) {
					let deploymentResult = signer.contracts.tryUpdate(
						name: "Foo",
						code: "access(all) contract Foo { access(all) fun sayHello(): String {return \"hello\"} }".utf8,
					)

					assert(deploymentResult.deployedContract == nil)
				}
			}
		`)

		invokeTx := []byte(`
			import Foo from 0x1

			transaction {
				prepare(signer: &Account) {
					assert(Foo.sayHello() == "hello")
				}
			}
		`)

		runtimeInterface := newTestRuntimeInterface(func() {})
		nextTransactionLocation := NewTransactionLocationGenerator()

		// Update non-existing 'Foo'. Should not panic.
		err := rt.ExecuteTransaction(
			Script{
				Source: updateTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Test the updated 'Foo'.
		// Foo must not be available.

		err = rt.ExecuteTransaction(
			Script{
				Source: invokeTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		RequireError(t, err)

		errs := checker.RequireCheckerErrors(t, err, 1)
		var notExportedError *sema.NotExportedError
		require.ErrorAs(t, errs[0], &notExportedError)
	})

	t.Run("tryUpdate with checking error", func(t *testing.T) {

		t.Parallel()

		rt := NewTestInterpreterRuntime()

		deployTx := DeploymentTransaction("Foo", []byte(`access(all) contract Foo {}`))

		updateTx := []byte(`
			transaction {
				prepare(signer: auth(UpdateContract) &Account) {
					let deploymentResult = signer.contracts.tryUpdate(
						name: "Foo",

						// Has a semantic error!
						code: "access(all) contract Foo { access(all) fun sayHello(): Int { return \"hello\" } }".utf8,
					)

					assert(deploymentResult.deployedContract == nil)
				}
			}
		`)

		runtimeInterface := newTestRuntimeInterface(func() {})

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy 'Foo'
		err := rt.ExecuteTransaction(
			Script{
				Source: deployTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Update 'Foo'.
		// User errors (parsing, checking and interpreting) should be handled gracefully.

		err = rt.ExecuteTransaction(
			Script{
				Source: updateTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		require.NoError(t, err)
	})

	t.Run("tryUpdate panic with internal error", func(t *testing.T) {

		t.Parallel()

		rt := NewTestInterpreterRuntime()

		deployTx := DeploymentTransaction("Foo", []byte(`access(all) contract Foo {}`))

		updateTx := []byte(`
			transaction {
				prepare(signer: auth(UpdateContract) &Account) {
					let deploymentResult = signer.contracts.tryUpdate(
						name: "Foo",
						code: "access(all) contract Foo { access(all) fun sayHello(): String {return \"hello\"} }".utf8,
					)

					assert(deploymentResult.deployedContract == nil)
				}
			}
		`)

		shouldPanic := false
		didPanic := false

		runtimeInterface := newTestRuntimeInterface(func() {
			if shouldPanic {
				didPanic = true
				panic("panic during update")
			}
		})

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy 'Foo'
		err := rt.ExecuteTransaction(
			Script{
				Source: deployTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)
		assert.False(t, didPanic)

		// Update 'Foo'.
		// Internal errors should NOT be handled gracefully.

		shouldPanic = true
		err = rt.ExecuteTransaction(
			Script{
				Source: updateTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)
		var unexpectedError errors.UnexpectedError
		require.ErrorAs(t, err, &unexpectedError)

		assert.True(t, didPanic)
	})
}
