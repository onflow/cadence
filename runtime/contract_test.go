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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
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

		runtime := newTestInterpreterRuntime()

		var loggedMessages []string

		signerAddress := Address{0x1}

		var deployedCode []byte

		addTx := []byte(
			fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: AuthAccount) {
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
                     prepare(signer: AuthAccount) {

                         let contract1 = signer.contracts.get(name: %[1]q)
                         log(contract1?.name)
                         log(contract1?.code)

                         let contract2 = signer.contracts.update__experimental(name: %[1]q, code: "%[2]s".decodeHex())
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
                      prepare(signer: AuthAccount) {
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
                      prepare(signer: AuthAccount) {
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

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAddress}, nil
			},
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			updateAccountContractCode: func(address Address, name string, code []byte) error {
				require.Equal(t, tc.name, name)
				assert.Equal(t, signerAddress, address)

				deployedCode = code

				return nil
			},
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				if name == tc.name {
					return deployedCode, nil
				}

				return nil, nil
			},
			removeAccountContractCode: func(address Address, name string) error {
				require.Equal(t, tc.name, name)
				assert.Equal(t, signerAddress, address)

				deployedCode = nil

				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		inter := newTestInterpreter(t)
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
			return storageMap.ValueExists("Test")
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
				require.Error(t, err)
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
			require.Error(t, err)

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
				require.Error(t, err)
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
			code:        `pub contract Test {}`,
			code2:       `pub contract Test { pub fun test() {} }`,
			valid:       true,
			isInterface: false,
		})
	})

	t.Run("valid contract interface, correct name", func(t *testing.T) {
		test(t, testCase{
			name:        "Test",
			code:        `pub contract interface Test {}`,
			code2:       `pub contract interface Test { pub fun test() }`,
			valid:       true,
			isInterface: true,
		})
	})

	t.Run("valid contract, wrong name", func(t *testing.T) {
		test(t, testCase{
			name:        "XYZ",
			code:        `pub contract Test {}`,
			valid:       false,
			isInterface: false,
		})
	})

	t.Run("valid contract interface, wrong name", func(t *testing.T) {
		test(t, testCase{
			name:        "XYZ",
			code:        `pub contract interface Test {}`,
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
              pub contract Test {}

              pub contract Test2 {}
            `,
			valid:       false,
			isInterface: false,
		})
	})

	t.Run("two contract interfaces", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              pub contract interface Test {}

              pub contract interface Test2 {}
            `,
			valid:       false,
			isInterface: true,
		})
	})

	t.Run("contract and contract interface", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              pub contract Test {}

              pub contract interface Test2 {}
            `,
			valid:       false,
			isInterface: false,
		})
	})
}

func TestRuntimeImportMultipleContracts(t *testing.T) {

	t.Parallel()

	contractA := `
      pub contract A {

          pub fun a(): Int {
              return 1
          }
      }
    `

	contractB := `
      pub contract B {

          pub fun b(): Int {
              return 2
          }
      }
    `

	contractC := `
      import A, B from 0x1

      pub contract C {

          pub fun c(): Int {
              return A.a() + B.b()
          }
      }
    `

	addTx := func(name, code string) []byte {
		return []byte(
			fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: AuthAccount) {
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

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{common.MustBytesToAddress([]byte{0x1})}, nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = accountCodes[location]
			return code, nil
		},
		removeAccountContractCode: func(address Address, name string) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			delete(accountCodes, location)
			return nil
		},
		resolveLocation: func(identifiers []ast.Identifier, location common.Location) (result []sema.ResolvedLocation, err error) {

			// Resolve each identifier as an address location

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
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	runtime := newTestInterpreterRuntime()

	nextTransactionLocation := newTransactionLocationGenerator()

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
              prepare(signer: AuthAccount) {
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
             prepare(signer: AuthAccount) {
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
              prepare(signer: AuthAccount) {
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
