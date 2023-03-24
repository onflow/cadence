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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestContractUpdateWithDependencies(t *testing.T) {
	t.Parallel()

	runtime := newTestInterpreterRuntime()
	accountCodes := map[common.Location][]byte{}
	signerAccount := common.MustBytesToAddress([]byte{0x1})
	fooLocation := common.AddressLocation{
		Address: signerAccount,
		Name:    "Foo",
	}
	var checkGetAndSetProgram, getProgramCalled bool

	programs := map[Location]*interpreter.Program{}
	clearPrograms := func() {
		for l := range programs {
			delete(programs, l)
		}
	}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
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
			return nil
		},
		decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
		getAndSetProgram: func(
			location Location,
			load func() (*interpreter.Program, error),
		) (
			program *interpreter.Program,
			err error,
		) {
			_, isTransactionLocation := location.(common.TransactionLocation)
			if checkGetAndSetProgram && !isTransactionLocation {
				require.Equal(t, location, fooLocation)
				require.False(t, getProgramCalled)
			}

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
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	const fooContractV1 = `
        pub contract Foo {
            init() {}
            pub fun hello() {}
        }
    `
	const barContractV1 = `
        import Foo from 0x01

        pub contract Bar {
            init() {
                Foo.hello()
            }
        }
    `

	const fooContractV2 = `
        pub contract Foo {
            init() {}
            pub fun hello(_ a: Int) {}
        }
    `

	const barContractV2 = `
        import Foo from 0x01

        pub contract Bar {
            init() {
                Foo.hello(5)
            }
        }
    `

	// Deploy 'Foo' contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"Foo",
				[]byte(fooContractV1),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Programs are only valid during the transaction
	clearPrograms()

	// Deploy 'Bar' contract

	signerAccount = common.MustBytesToAddress([]byte{0x2})

	err = runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"Bar",
				[]byte(barContractV1),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Programs are only valid during the transaction
	clearPrograms()

	// Update 'Foo' contract to change function signature

	signerAccount = common.MustBytesToAddress([]byte{0x1})
	err = runtime.ExecuteTransaction(
		Script{
			Source: utils.UpdateTransaction("Foo", []byte(fooContractV2)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Programs are only valid during the transaction
	clearPrograms()

	// Update 'Bar' contract to change match the
	// function signature change in 'Foo'.

	signerAccount = common.MustBytesToAddress([]byte{0x2})

	checkGetAndSetProgram = true

	err = runtime.ExecuteTransaction(
		Script{
			Source: utils.UpdateTransaction("Bar", []byte(barContractV2)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestContractUpdateWithPrecedingIdentifiers(t *testing.T) {
	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	fooLocation := common.AddressLocation{
		Address: signerAccount,
		Name:    "Foo",
	}

	const fooContractV1 = `
        pub contract Foo {
            // NOTE: invalid preceding identifier in member declaration
            bar pub let foo: Int

            init() {
                self.foo = 1
            }
        }
    `

	const fooContractV2 = `
        pub contract Foo {
            pub let foo: Int

            init() {
                self.foo = 1
            }
        }
    `

	// Assume contract with deprecated syntax is already deployed

	accountCodes := map[common.Location][]byte{
		fooLocation: []byte(fooContractV1),
	}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
		decodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Update contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: utils.UpdateTransaction("Foo", []byte(fooContractV2)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

}
