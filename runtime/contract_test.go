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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestRuntimeContract(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name        string
		code        string
		code2       string
		valid       bool
		isInterface bool
	}

	test := func(t *testing.T, tc testCase) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		var loggedMessages []string

		signerAddress := Address{0x1}

		var deployedCode []byte

		addTx := []byte(
			fmt.Sprintf(
				`
	              transaction {
	                  prepare(signer: AuthAccount) {
                          let contract1 = signer.contracts.add(name: %[1]q, code: "%[2]s".decodeHex())
                          log(contract1.name)
                          log(contract1.code)

                          let contract2 = signer.contracts.get(name: %[1]q)
                          log(contract2?.name)
                          log(contract2?.code)

                          let contract3 = signer.contracts.get(name: "Unknown")
                          log(contract3)
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

		var events []cadence.Event

		storage := newTestStorage(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			getSigningAccounts: func() []Address {
				return []Address{signerAddress}
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
			emitEvent: func(event cadence.Event) {
				events = append(events, event)
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		contractKey := []byte("contract\x1fTest")

		codeArrayString := interpreter.ByteSliceToByteArrayValue([]byte(tc.code)).String()
		code2ArrayString := interpreter.ByteSliceToByteArrayValue([]byte(tc.code2)).String()

		t.Run("add", func(t *testing.T) {

			err := runtime.ExecuteTransaction(addTx, nil, runtimeInterface, nextTransactionLocation())

			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, []byte(tc.code), deployedCode)

				contractValueExists, err := storage.valueExists(signerAddress[:], contractKey)
				require.NoError(t, err)

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

				contractValueExists, err := storage.valueExists(signerAddress[:], contractKey)
				require.NoError(t, err)

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

			err := runtime.ExecuteTransaction(addTx, nil, runtimeInterface, nextTransactionLocation())
			require.Error(t, err)

			require.NotEmpty(t, deployedCode)
			require.Empty(t, events)

		})

		t.Run("update", func(t *testing.T) {

			// Run the update transaction

			loggedMessages = nil
			events = nil

			err := runtime.ExecuteTransaction(updateTx, nil, runtimeInterface, nextTransactionLocation())
			require.NoError(t, err)

			require.Equal(t, []byte(tc.code2), deployedCode)

			contractValueExists, err := storage.valueExists(signerAddress[:], contractKey)
			require.NoError(t, err)

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

			err := runtime.ExecuteTransaction(removeTx, nil, runtimeInterface, nextTransactionLocation())
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

			contractValueExists, err := storage.valueExists(signerAddress[:], contractKey)
			require.NoError(t, err)

			require.False(t, contractValueExists)

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
