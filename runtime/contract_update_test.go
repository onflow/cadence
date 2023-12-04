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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeContractUpdateWithDependencies(t *testing.T) {
	t.Parallel()

	runtime := NewTestInterpreterRuntime()
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
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
		OnGetAndSetProgram: func(
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

	nextTransactionLocation := NewTransactionLocationGenerator()

	const fooContractV1 = `
        access(all) contract Foo {
            init() {}
            access(all) fun hello() {}
        }
    `
	const barContractV1 = `
        import Foo from 0x01

        access(all) contract Bar {
            init() {
                Foo.hello()
            }
        }
    `

	const fooContractV2 = `
        access(all) contract Foo {
            init() {}
            access(all) fun hello(_ a: Int) {}
        }
    `

	const barContractV2 = `
        import Foo from 0x01

        access(all) contract Bar {
            init() {
                Foo.hello(5)
            }
        }
    `

	// Deploy 'Foo' contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
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
			Source: DeploymentTransaction(
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
			Source: UpdateTransaction("Foo", []byte(fooContractV2)),
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
			Source: UpdateTransaction("Bar", []byte(barContractV2)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeContractUpdateWithPrecedingIdentifiers(t *testing.T) {
	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	fooLocation := common.AddressLocation{
		Address: signerAccount,
		Name:    "Foo",
	}

	const fooContractV1 = `
        access(all) contract Foo {
            // NOTE: invalid preceding identifier in member declaration
            bar access(all) let foo: Int

            init() {
                self.foo = 1
            }
        }
    `

	const fooContractV2 = `
        access(all) contract Foo {
            access(all) let foo: Int

            init() {
                self.foo = 1
            }
        }
    `

	// Assume contract with deprecated syntax is already deployed

	accountCodes := map[common.Location][]byte{
		fooLocation: []byte(fooContractV1),
	}

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
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Update contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: UpdateTransaction("Foo", []byte(fooContractV2)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

}

func TestRuntimeContractRedeployInSameTransaction(t *testing.T) {

	t.Parallel()

	t.Run("two additions", func(t *testing.T) {

		foo1 := []byte(`
            access(all)
            contract Foo {

                access(all)
                resource R {

                    access(all)
                    var x: Int

                    init() {
                        self.x = 0
                    }
                }

                access(all)
                fun createR(): @R {
                    return <-create R()
                }
            }
        `)

		foo2 := []byte(`
            access(all)
            contract Foo {

                access(all)
                struct R {
                    access(all)
                    var x: Int

                    init() {
                        self.x = 0
                    }
                }
            }
        `)

		tx := []byte(`
            transaction(foo1: String, foo2: String) {
                prepare(signer: auth(Contracts) &Account) {
                    signer.contracts.add(name: "Foo", code: foo1.utf8)
                    signer.contracts.add(name: "Foo", code: foo2.utf8)
                }
            }
        `)

		runtime := NewTestInterpreterRuntimeWithConfig(Config{
			AtreeValidationEnabled: false,
		})

		address := common.MustBytesToAddress([]byte{0x1})

		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
				return nil, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				// "delay"
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

			// Deploy

		err := runtime.ExecuteTransaction(
			Script{
				Source: tx,
				Arguments: encodeArgs([]cadence.Value{
					cadence.String(foo1),
					cadence.String(foo2),
				}),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)
		require.ErrorContains(t, err, "cannot overwrite existing contract")
	})
}

func TestRuntimeNestedContractDeployment(t *testing.T) {

	t.Parallel()

	t.Run("add while adding", func(t *testing.T) {

		t.Parallel()

		contract := []byte(`
            access(all) contract Foo {
 
                access(all) resource Bar {}

                init(){
                    self.account.contracts.add(
                        name: "Foo",
                        code: "access(all) contract Foo { access(all) struct Bar {} }".utf8
                    )
                }
            }
        `)

		runtime := NewTestInterpreterRuntimeWithConfig(Config{
			AtreeValidationEnabled: false,
		})

		address := common.MustBytesToAddress([]byte{0x1})

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
				return nil, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				// "delay"
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy

		deploymentTx := DeploymentTransaction("Foo", contract)

		err := runtime.ExecuteTransaction(
			Script{
				Source: deploymentTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)
		require.ErrorContains(t, err, "cannot overwrite existing contract")
	})

	t.Run("update while adding", func(t *testing.T) {

		t.Parallel()

		contract := []byte(`
            access(all) contract Foo {
 
                access(all) resource Bar {}

                init(){
                   self.account.contracts.update__experimental(
                        name: "Foo",
                        code: "access(all) contract Foo { access(all) struct Bar {} }".utf8
                    )
                }
            }
        `)

		runtime := NewTestInterpreterRuntimeWithConfig(Config{
			AtreeValidationEnabled: false,
		})

		address := common.MustBytesToAddress([]byte{0x1})

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
				return nil, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				// "delay"
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy

		deploymentTx := DeploymentTransaction("Foo", contract)

		err := runtime.ExecuteTransaction(
			Script{
				Source: deploymentTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		RequireError(t, err)
		require.ErrorContains(t, err, "cannot update non-existing contract")
	})

	t.Run("update while updating", func(t *testing.T) {

		t.Parallel()

		deployedContract := []byte(`
            access(all) contract Foo {
 
                access(all) resource Bar {}

                init() {}
            }
        `)

		contract := []byte(`
            access(all) contract Foo {
 
                access(all) resource Bar {}

                init(){
                   self.account.contracts.update__experimental(
                        name: "Foo",
                        code: "access(all) contract Foo { access(all) struct Bar {} }".utf8
                    )
                }
            }
        `)

		runtime := NewTestInterpreterRuntimeWithConfig(Config{
			AtreeValidationEnabled: false,
		})

		address := common.MustBytesToAddress([]byte{0x1})

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
				return deployedContract, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				// "delay"
				deployedContract = code
				return nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Update

		updateTx := UpdateTransaction("Foo", contract)

		err := runtime.ExecuteTransaction(
			Script{
				Source: updateTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		// OK: since the initializer never runs.
		require.NoError(t, err)
	})
}

func TestRuntimeContractRedeploymentInSeparateTransactions(t *testing.T) {

	t.Parallel()

	contract := []byte(`
            access(all) contract Foo {
                access(all) resource Bar {}
            }
        `)

	runtime := NewTestInterpreterRuntimeWithConfig(Config{
		AtreeValidationEnabled: false,
	})

	address := common.MustBytesToAddress([]byte{0x1})

	var contractCode []byte

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
			return contractCode, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			contractCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy

	deploymentTx := DeploymentTransaction("Foo", contract)
	err := runtime.ExecuteTransaction(
		Script{
			Source: deploymentTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Update
	// Updating in a separate transaction is OK, and should not abort.

	updateTx := UpdateTransaction("Foo", contract)
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
}
