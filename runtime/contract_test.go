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

	"github.com/stretchr/testify/require"
)

func TestRuntimeContract(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name  string
		code  string
		valid bool
	}

	test := func(t *testing.T, tc testCase) {

		t.Parallel()

		runtime := NewInterpreterRuntime()

		var loggedMessages []string

		testTx := []byte(
			fmt.Sprintf(
				`
	              transaction {
	                  prepare() {
	                      let contract = Contract(name: %q, code: "%s".decodeHex())
                          log(contract.name)
                          log(contract.code)
	                  }
	               }
	            `,
				tc.name,
				hex.EncodeToString([]byte(tc.code)),
			))

		runtimeInterface := &testRuntimeInterface{
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(testTx, nil, runtimeInterface, nextTransactionLocation())

		if tc.valid {
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`"Test"`,
					`[112, 117, 98, 32, 99, 111, 110, 116, 114, 97, 99, 116, 32, 84, 101, 115, 116, 32, 123, 125]`,
				},
				loggedMessages,
			)
		} else {
			require.Error(t, err)
		}
	}

	t.Run("valid contract, correct name", func(t *testing.T) {
		test(t, testCase{
			name:  "Test",
			code:  `pub contract Test {}`,
			valid: true,
		})
	})

	t.Run("valid contract interface, correct name", func(t *testing.T) {
		test(t, testCase{
			name:  "Test",
			code:  `pub contract interface Test {}`,
			valid: true,
		})
	})

	t.Run("valid contract, wrong name", func(t *testing.T) {
		test(t, testCase{
			name:  "XYZ",
			code:  `pub contract Test {}`,
			valid: false,
		})
	})

	t.Run("valid contract interface, wrong name", func(t *testing.T) {
		test(t, testCase{
			name:  "XYZ",
			code:  `pub contract interface Test {}`,
			valid: false,
		})
	})

	t.Run("invalid code", func(t *testing.T) {
		test(t, testCase{
			name:  "Test",
			code:  `foo`,
			valid: false,
		})
	})

	t.Run("missing contract or contract interface", func(t *testing.T) {
		test(t, testCase{
			name:  "Test",
			code:  ``,
			valid: false,
		})
	})

	t.Run("two contracts", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              pub contract Test {}

              pub contract Test2 {}
            `,
			valid: false,
		})
	})

	t.Run("two contract interfaces", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              pub contract interface Test {}

              pub contract interface Test2 {}
            `,
			valid: false,
		})
	})

	t.Run("contract and contract interface", func(t *testing.T) {
		test(t, testCase{
			name: "Test",
			code: `
              pub contract Test {}

              pub contract interface Test2 {}
            `,
			valid: false,
		})
	})
}
