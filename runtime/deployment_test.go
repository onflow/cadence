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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeTransactionWithContractDeployment(t *testing.T) {

	t.Parallel()

	type expectation func(t *testing.T, err error, accountCode []byte, events []cadence.Event, expectedEventType cadence.Type)

	expectSuccess := func(t *testing.T, err error, accountCode []byte, events []cadence.Event, expectedEventType cadence.Type) {
		require.NoError(t, err)

		assert.NotNil(t, accountCode)

		require.Len(t, events, 1)

		event := events[0]

		require.Equal(t, event.Type(), expectedEventType)

		expectedEventCompositeType := expectedEventType.(*cadence.EventType)

		codeHashParameterIndex := -1

		for i, field := range expectedEventCompositeType.Fields {
			if field.Identifier != stdlib.AccountEventCodeHashParameter.Identifier {
				continue
			}
			codeHashParameterIndex = i
		}

		if codeHashParameterIndex < 0 {
			t.Error("couldn't find code hash parameter in event type")
		}

		expectedCodeHash := sha3.Sum256(accountCode)

		codeHashValue := event.Fields[codeHashParameterIndex]

		inter := newTestInterpreter(t)

		codeHash, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			codeHashValue,
			sema.ByteArrayType,
		)
		require.NoError(t, err)

		actualCodeHash, err := interpreter.ByteArrayValueToByteSlice(inter, codeHash)
		require.NoError(t, err)

		require.Equal(t, expectedCodeHash[:], actualCodeHash)
	}

	expectFailure := func(expectedErrorMessage string) expectation {
		return func(t *testing.T, err error, accountCode []byte, events []cadence.Event, _ cadence.Type) {
			RequireError(t, err)

			var runtimeErr Error
			require.ErrorAs(t, err, &runtimeErr)

			println(runtimeErr.Error())

			assert.EqualError(t, runtimeErr, expectedErrorMessage)

			assert.Len(t, runtimeErr.Codes, 2)
			assert.Len(t, runtimeErr.Programs, 1)

			assert.Nil(t, accountCode)
			assert.Len(t, events, 0)
		}
	}

	type argument interface {
		interpreter.Value
	}

	type testCase struct {
		contract  string
		arguments []argument
		check     func(t *testing.T, err error, accountCode []byte, events []cadence.Event, expectedEventType cadence.Type)
	}

	test := func(t *testing.T, test testCase) {

		t.Parallel()

		contractArrayCode := fmt.Sprintf(
			`"%s".decodeHex()`,
			hex.EncodeToString([]byte(test.contract)),
		)

		argumentCodes := make([]string, len(test.arguments))

		for i, argument := range test.arguments {
			argumentCodes[i] = argument.String()
		}

		argumentCode := strings.Join(argumentCodes, ", ")
		if len(test.arguments) > 0 {
			argumentCode = ", " + argumentCode
		}

		script := []byte(fmt.Sprintf(
			`
              transaction {

                  prepare(signer: AuthAccount) {
                      signer.contracts.add(name: "Test", code: %s%s)
                  }
              }
            `,
			contractArrayCode,
			argumentCode,
		))

		runtime := newTestInterpreterRuntime()

		var accountCode []byte
		var events []cadence.Event

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
			},
			getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
				return accountCode, nil
			},
			updateAccountContractCode: func(_ Address, _ string, code []byte) error {
				accountCode = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		exportedEventType := ExportType(
			stdlib.AccountContractAddedEventType,
			map[sema.TypeID]cadence.Type{},
		)
		test.check(t, err, accountCode, events, exportedEventType)
	}

	t.Run("no arguments", func(t *testing.T) {
		test(t, testCase{
			contract: `
              pub contract Test {}
            `,
			arguments: []argument{},
			check:     expectSuccess,
		})
	})

	t.Run("with argument", func(t *testing.T) {
		test(t, testCase{
			contract: `
              pub contract Test {
                  init(_ x: Int) {}
              }
            `,
			arguments: []argument{
				interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			check: expectSuccess,
		})
	})

	t.Run("with incorrect argument", func(t *testing.T) {
		test(t, testCase{
			contract: `
              pub contract Test {
                  init(_ x: Int) {}
              }
            `,
			arguments: []argument{
				interpreter.TrueValue,
			},
			check: expectFailure(
				"Execution failed:\n" +
					"error: invalid argument 0: expected type `Int`, got `Bool`\n" +
					" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
					"  |\n" +
					"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202070756220636f6e74726163742054657374207b0a202020202020202020202020202020202020696e6974285f20783a20496e7429207b7d0a20202020202020202020202020207d0a202020202020202020202020\".decodeHex(), true)\n" +
					"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n",
			),
		})
	})

	t.Run("additional argument", func(t *testing.T) {
		test(t, testCase{
			contract: `
              pub contract Test {}
            `,
			arguments: []argument{
				interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			check: expectFailure(
				"Execution failed:\n" +
					"error: invalid argument count, too many arguments: expected 0, got 1\n" +
					" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
					"  |\n" +
					"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202070756220636f6e74726163742054657374207b7d0a202020202020202020202020\".decodeHex(), 1)\n" +
					"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n",
			),
		})
	})

	t.Run("additional code which is invalid at top-level", func(t *testing.T) {
		test(t, testCase{
			contract: `
              pub contract Test {}

              fun testCase() {}
            `,
			arguments: []argument{},
			check: expectFailure(
				"Execution failed:\n" +
					"error: cannot deploy invalid contract\n" +
					" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
					"  |\n" +
					"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202070756220636f6e74726163742054657374207b7d0a0a202020202020202020202020202066756e2074657374436173652829207b7d0a202020202020202020202020\".decodeHex())\n" +
					"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
					"\n" +
					"error: function declarations are not valid at the top-level\n" +
					" --> 2a00000000000000.Test:4:18\n" +
					"  |\n" +
					"4 |               fun testCase() {}\n" +
					"  |                   ^^^^^^^^\n" +
					"\n" +
					"error: missing access modifier for function\n" +
					" --> 2a00000000000000.Test:4:14\n" +
					"  |\n" +
					"4 |               fun testCase() {}\n" +
					"  |               ^\n",
			),
		})
	})

	t.Run("invalid contract, parsing error", func(t *testing.T) {
		test(t, testCase{
			contract: `
              X
            `,
			arguments: []argument{},
			check: expectFailure(
				"Execution failed:\n" +
					"error: cannot deploy invalid contract\n" +
					" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
					"  |\n" +
					"5 |                       signer.contracts.add(name: \"Test\", code: \"0a2020202020202020202020202020580a202020202020202020202020\".decodeHex())\n" +
					"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
					"\n" +
					"error: unexpected token: identifier\n" +
					" --> 2a00000000000000.Test:2:14\n" +
					"  |\n" +
					"2 |               X\n" +
					"  |               ^\n",
			),
		})
	})

	t.Run("invalid contract, checking error", func(t *testing.T) {
		test(t, testCase{
			contract: `
              pub contract Test {
                  pub fun test() { X }
              }
            `,
			arguments: []argument{},
			check: expectFailure(
				"Execution failed:\n" +
					"error: cannot deploy invalid contract\n" +
					" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
					"  |\n" +
					"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202070756220636f6e74726163742054657374207b0a2020202020202020202020202020202020207075622066756e20746573742829207b2058207d0a20202020202020202020202020207d0a202020202020202020202020\".decodeHex())\n" +
					"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
					"\n" +
					"error: cannot find variable in this scope: `X`\n" +
					" --> 2a00000000000000.Test:3:35\n" +
					"  |\n" +
					"3 |                   pub fun test() { X }\n" +
					"  |                                    ^ not found in this scope\n",
			),
		})
	})

}
