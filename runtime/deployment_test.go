/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"strings"
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
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

		expectedCodeHash := sha3.Sum256(accountCode)

		fields := cadence.FieldsMappedByName(event)
		codeHashValue := fields["codeHash"]

		inter := NewTestInterpreter(t)

		require.Equal(t,
			ImportType(inter, codeHashValue.Type()),
			interpreter.ConvertSemaToStaticType(inter, stdlib.AccountEventCodeHashParameter.TypeAnnotation.Type),
		)

		codeHash, err := ImportValue(
			inter,
			nil,
			nil,
			codeHashValue,
			stdlib.HashType,
		)
		require.NoError(t, err)

		actualCodeHash, err := interpreter.ByteArrayValueToByteSlice(
			inter,
			codeHash,
		)
		require.NoError(t, err)

		require.Equal(t, expectedCodeHash[:], actualCodeHash)
	}

	expectFailure := func(expectedErrorMessage string, codesCount, programsCount int) expectation {
		return func(t *testing.T, err error, accountCode []byte, events []cadence.Event, _ cadence.Type) {
			RequireError(t, err)

			var runtimeErr Error
			require.ErrorAs(t, err, &runtimeErr)

			assert.ErrorContains(t, runtimeErr, expectedErrorMessage)

			assert.Len(t, runtimeErr.Codes, codesCount)
			assert.Len(t, runtimeErr.Programs, programsCount)

			assert.Nil(t, accountCode)
			assert.Len(t, events, 0)
		}
	}

	type checkFunc = func(
		t *testing.T,
		err error,
		accountCode []byte,
		events []cadence.Event,
		expectedEventType cadence.Type,
	)

	type testCase struct {
		check         checkFunc
		contract      string
		arguments     []string
		declaredValue stdlib.StandardLibraryValue
	}

	test := func(t *testing.T, test testCase) {

		t.Parallel()

		contractArrayCode := fmt.Sprintf(
			`"%s".decodeHex()`,
			hex.EncodeToString([]byte(test.contract)),
		)

		argumentCode := strings.Join(test.arguments, ", ")
		if len(test.arguments) > 0 {
			argumentCode = ", " + argumentCode
		}

		script := []byte(fmt.Sprintf(
			`
              transaction {

                  prepare(signer: auth(AddContract) &Account) {
                      signer.contracts.add(name: "Test", code: %s%s)
                  }
              }
            `,
			contractArrayCode,
			argumentCode,
		))

		runtime := NewTestRuntime()

		var accountCode []byte
		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{{42}}, nil
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

		environment := newTransactionEnvironment()

		if test.declaredValue.Value != nil {
			environment.DeclareValue(
				test.declaredValue,
				nil,
			)
		}

		err := runtime.ExecuteTransaction(
			Script{
				Source: script,
			},
			Context{
				Interface:   runtimeInterface,
				Environment: environment,
				Location:    common.TransactionLocation{},
				UseVM:       *compile,
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
              access(all) contract Test {}
            `,
			arguments: []string{},
			check:     expectSuccess,
		})
	})

	t.Run("with argument", func(t *testing.T) {
		test(t, testCase{
			contract: `
              access(all) contract Test {
                  init(_ x: Int) {}
              }
            `,
			arguments: []string{
				`1`,
			},
			check: expectSuccess,
		})
	})

	t.Run("with incorrect argument", func(t *testing.T) {

		expectedErrorMessage := "Execution failed:\n" +
			"error: invalid argument at index 0: expected type `Int`, got `Bool`\n" +
			" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
			"  |\n" +
			"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202061636365737328616c6c2920636f6e74726163742054657374207b0a202020202020202020202020202020202020696e6974285f20783a20496e7429207b7d0a20202020202020202020202020207d0a202020202020202020202020\".decodeHex(), true)\n" +
			"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"

		test(t, testCase{
			contract: `
              access(all) contract Test {
                  init(_ x: Int) {}
              }
            `,
			arguments: []string{
				`true`,
			},
			check: expectFailure(
				expectedErrorMessage,
				2,
				2,
			),
		})
	})

	t.Run("additional argument", func(t *testing.T) {

		expectedErrorMessage := "Execution failed:\n" +
			"error: invalid argument count, too many arguments: expected 0, got 1\n" +
			" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
			"  |\n" +
			"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202061636365737328616c6c2920636f6e74726163742054657374207b7d0a202020202020202020202020\".decodeHex(), 1)\n" +
			"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"

		test(t, testCase{
			contract: `
              access(all) contract Test {}
            `,
			arguments: []string{
				`1`,
			},
			check: expectFailure(
				expectedErrorMessage,
				2,
				2,
			),
		})
	})

	t.Run("additional code which is invalid at top-level", func(t *testing.T) {

		expectedErrorMessage := "Execution failed:\n" +
			"error: cannot deploy invalid contract\n" +
			" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
			"  |\n" +
			"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202061636365737328616c6c2920636f6e74726163742054657374207b7d0a0a202020202020202020202020202066756e2074657374436173652829207b7d0a202020202020202020202020\".decodeHex())\n" +
			"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
			"\n" +
			"error: function declarations are not valid at the top-level\n" +
			" --> 2a00000000000000.Test:4:18\n" +
			"  |\n" +
			"4 |               fun testCase() {}\n" +
			"  |                   ^^^^^^^^ move this declaration into a contract\n" +
			"\n" +
			"error: missing access modifier for function\n" +
			" --> 2a00000000000000.Test:4:14\n" +
			"  |\n" +
			"4 |               fun testCase() {}\n" +
			"  |               ^ an access modifier is required for this declaration; add an access modifier, like e.g. `access(all)` or `access(self)`\n" +
			"\n" +
			"  See documentation at: https://cadence-lang.org/docs/language/access-control\n" +
			"\n"

		test(t, testCase{
			contract: `
              access(all) contract Test {}

              fun testCase() {}
            `,
			arguments: []string{},
			check: expectFailure(
				expectedErrorMessage,
				2,
				2,
			),
		})
	})

	t.Run("invalid contract, parsing error", func(t *testing.T) {

		expectedErrorMessage := "Execution failed:\n" +
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
			"  |               ^ check for extra characters, missing semicolons, or incomplete statements\n"

		test(t, testCase{
			contract: `
              X
            `,
			arguments: []string{},
			check: expectFailure(
				expectedErrorMessage,
				2,
				1,
			),
		})
	})

	t.Run("invalid contract, checking error", func(t *testing.T) {
		expectedErrorMessage := "Execution failed:\n" +
			"error: cannot deploy invalid contract\n" +
			" --> 0000000000000000000000000000000000000000000000000000000000000000:5:22\n" +
			"  |\n" +
			"5 |                       signer.contracts.add(name: \"Test\", code: \"0a202020202020202020202020202061636365737328616c6c2920636f6e74726163742054657374207b0a20202020202020202020202020202020202061636365737328616c6c292066756e20746573742829207b2058207d0a20202020202020202020202020207d0a202020202020202020202020\".decodeHex())\n" +
			"  |                       ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
			"\n" +
			"error: cannot find variable in this scope: `X`\n" +
			" --> 2a00000000000000.Test:3:43\n" +
			"  |\n" +
			"3 |                   access(all) fun test() { X }\n" +
			"  |                                            ^ not found in this scope; check for typos or declare it\n"

		test(t, testCase{
			contract: `
              access(all) contract Test {
                  access(all) fun test() { X }
              }
            `,
			arguments: []string{},
			check: expectFailure(
				expectedErrorMessage,
				2,
				2,
			),
		})
	})

	t.Run("Path subtype", func(t *testing.T) {
		test(t, testCase{
			contract: `
              access(all) contract Test {
                  init(_ path: StoragePath) {}
              }
            `,
			arguments: []string{
				`/storage/test`,
			},
			check: expectSuccess,
		})
	})

	t.Run("Type confusion", func(t *testing.T) {

		var check checkFunc
		if *compile {
			check = expectFailure("invalid argument at index 0: expected type `Bool`, got `Int`",
				2,
				2,
			)
		} else {
			check = expectFailure(
				"invalid transfer of value: expected `Int`, got `Bool`",
				1,
				1,
			)
		}

		const declaredValueName = `injectedValue`
		test(t, testCase{
			contract: `
              access(all) contract Test {
                  init(_ bool: Bool) {}
              }
            `,
			arguments: []string{
				declaredValueName,
			},
			declaredValue: stdlib.StandardLibraryValue{
				Name:  declaredValueName,
				Type:  sema.IntType,
				Kind:  common.DeclarationKindValue,
				Value: interpreter.TrueValue,
			},
			check: check,
		})
	})
}

func TestRuntimeContractDeploymentInitializerArgument(t *testing.T) {

	t.Parallel()

	runtime := NewTestRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
      access(all) contract Test {
          init(arg: {Int: Int}) {
              check(arg)
          }
      }
    `)

	deploy := fmt.Sprintf(
		`
          transaction {
              prepare(signer: auth(Contracts) &Account) {
                  let arg: {Int: Int} = {}
                  signer.contracts.add(name: "Test", code: "%s".decodeHex(), arg: arg)
              }
          }
        `,
		hex.EncodeToString(contract),
	)

	var accountCode []byte

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
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
	}

	check := func(value interpreter.Value) {
		dictionaryValue, ok := value.(*interpreter.DictionaryValue)
		require.True(t, ok)

		assert.Equal(t,
			atree.ValueID{
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6,
			},
			dictionaryValue.ValueID(),
		)
	}

	transactionEnvironment := newTransactionEnvironment()

	functionType := sema.NewSimpleFunctionType(
		sema.FunctionPurityView,
		[]sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "arg",
				TypeAnnotation: sema.NewTypeAnnotation(
					sema.NewDictionaryType(nil, sema.IntType, sema.IntType),
				),
			},
		},
		sema.VoidTypeAnnotation,
	)

	var function interpreter.FunctionValue

	const functionName = "check"
	if *compile {
		function = vm.NewNativeFunctionValue(
			functionName,
			functionType,
			func(
				_ interpreter.NativeFunctionContext,
				_ interpreter.TypeArgumentsIterator,
				_ interpreter.ArgumentTypesIterator,
				_ interpreter.Value,
				args []interpreter.Value,
			) interpreter.Value {
				check(args[0])
				return interpreter.Void
			},
		)
	} else {
		function = interpreter.NewStaticHostFunctionValue(
			nil,
			functionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				check(invocation.Arguments[0])
				return interpreter.Void
			},
		)
	}

	transactionEnvironment.DeclareValue(
		stdlib.StandardLibraryValue{
			Name:  functionName,
			Type:  functionType,
			Kind:  common.DeclarationKindFunction,
			Value: function,
		},
		nil,
	)

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(deploy),
		},
		Context{
			Interface:   runtimeInterface,
			Environment: transactionEnvironment,
			Location:    nextTransactionLocation(),
			UseVM:       *compile,
		},
	)
	require.NoError(t, err)
}
