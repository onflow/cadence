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
		valid       bool
		isInterface bool
	}

	test := func(t *testing.T, tc testCase) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		var loggedMessages []string

		signerAddress := Address{0x1}

		testTx := []byte(
			fmt.Sprintf(
				`
	              transaction {
	                  prepare(signer: AuthAccount) {
                          let contract = signer.contracts.add(name: %q, code: "%s".decodeHex())
                          log(contract.name)
                          log(contract.code)
	                  }
	               }
	            `,
				tc.name,
				hex.EncodeToString([]byte(tc.code)),
			))

		codeUpdated := false

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
				codeUpdated = true

				assert.Equal(t, tc.name, name)
				assert.Equal(t, signerAddress, address)
				assert.Equal(t, tc.code, string(code))

				return nil
			},
			emitEvent: func(event cadence.Event) {
				events = append(events, event)
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(testTx, nil, runtimeInterface, nextTransactionLocation())

		if tc.valid {
			require.NoError(t, err)
			require.True(t, codeUpdated)
			require.Equal(t,
				[]string{
					`"Test"`,
					interpreter.ByteSliceToByteArrayValue([]byte(tc.code)).String(),
				},
				loggedMessages,
			)
			require.Len(t, events, 1)
			assert.EqualValues(t, stdlib.AccountContractAddedEventType.ID(), events[0].Type().ID())
		} else {
			require.Error(t, err)
			require.False(t, codeUpdated)
			require.Empty(t, events)
			require.Empty(t, loggedMessages)
		}

		contractKey := []byte("contract\x1fTest")

		exists, err := storage.valueExists(signerAddress[:], contractKey)
		require.NoError(t, err)

		if tc.valid && !tc.isInterface {
			require.True(t, exists)
		} else {
			require.False(t, exists)
		}
	}

	t.Run("valid contract, correct name", func(t *testing.T) {
		test(t, testCase{
			name:        "Test",
			code:        `pub contract Test {}`,
			valid:       true,
			isInterface: false,
		})
	})

	t.Run("valid contract interface, correct name", func(t *testing.T) {
		test(t, testCase{
			name:        "Test",
			code:        `pub contract interface Test {}`,
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
