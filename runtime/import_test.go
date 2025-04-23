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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestRuntimeCyclicImport(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	imported1 := []byte(`
      import p2
    `)

	imported2 := []byte(`
      import p1
    `)

	script := []byte(`
      import p1

      access(all) fun main() {}
    `)

	var checkCount int

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.IdentifierLocation("p1"):
				return imported1, nil
			case common.IdentifierLocation("p2"):
				return imported2, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		OnProgramChecked: func(location Location, duration time.Duration) {
			checkCount += 1
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

	require.Contains(t, err.Error(), "cyclic import of `p1`")

	// Script

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := RequireCheckerErrors(t, checkerErr, 1)

	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr)

	// P1

	var checkerErr2 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr.Err, &checkerErr2)

	errs = RequireCheckerErrors(t, checkerErr2, 1)

	var importedProgramErr2 *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr2)

	// P2

	var checkerErr3 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr2.Err, &checkerErr3)

	errs = RequireCheckerErrors(t, checkerErr3, 1)

	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}

func TestRuntimeCheckCyclicImportsAfterUpdate(t *testing.T) {

	runtime := NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}

	signerAccount := contractsAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) ([]byte, error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
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

	environment := NewBaseInterpreterEnvironment(Config{})

	nextTransactionLocation := NewTransactionLocationGenerator()

	deploy := func(name string, contract string, update bool) error {
		var txSource = DeploymentTransaction
		if update {
			txSource = UpdateTransaction
		}

		return runtime.ExecuteTransaction(
			Script{
				Source: txSource(
					name,
					[]byte(contract),
				),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
	}

	const fooContract = `
        access(all) contract Foo {}
    `

	const barContract = `
        import Foo from 0x0000000000000001
        access(all) contract Bar {}
    `

	const updatedFooContract = `
        import Bar from 0x0000000000000001
        access(all) contract Foo {}
    `

	err := deploy("Foo", fooContract, false)
	require.NoError(t, err)

	err = deploy("Bar", barContract, false)
	require.NoError(t, err)

	// Update `Foo` contract creating a cycle.
	err = deploy("Foo", updatedFooContract, true)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := RequireCheckerErrors(t, checkerErr, 1)

	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr)

	var nestedCheckerErr *sema.CheckerError
	require.ErrorAs(t, importedProgramErr.Err, &nestedCheckerErr)

	errs = RequireCheckerErrors(t, nestedCheckerErr, 1)
	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}

func TestRuntimeCheckCyclicImportAddress(t *testing.T) {

	runtime := NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}

	signerAccount := contractsAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) ([]byte, error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
			if len(identifiers) == 0 {
				require.IsType(t, common.AddressLocation{}, location)
				addressLocation := location.(common.AddressLocation)

				require.Equal(t, contractsAddress, addressLocation.Address)

				// `Foo` and `Bar` are already deployed at the address.
				identifiers = append(
					identifiers,
					Identifier{
						Identifier: "Foo",
					},
					Identifier{
						Identifier: "Bar",
					},
				)
			}

			return MultipleIdentifierLocationResolver(identifiers, location)
		},
		OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
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

	environment := NewBaseInterpreterEnvironment(Config{})

	nextTransactionLocation := NewTransactionLocationGenerator()

	deploy := func(name string, contract string, update bool) error {
		var txSource = DeploymentTransaction
		if update {
			txSource = UpdateTransaction
		}

		return runtime.ExecuteTransaction(
			Script{
				Source: txSource(
					name,
					[]byte(contract),
				),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
	}

	const fooContract = `
        access(all) contract Foo {}
    `

	const barContract = `
        import Foo from 0x0000000000000001
        access(all) contract Bar {}
    `

	const updatedFooContract = `
        import 0x0000000000000001
        access(all) contract Foo {}
    `

	err := deploy("Foo", fooContract, false)
	require.NoError(t, err)

	err = deploy("Bar", barContract, false)
	require.NoError(t, err)

	// Update `Foo` contract creating a cycle.
	err = deploy("Foo", updatedFooContract, true)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := RequireCheckerErrors(t, checkerErr, 1)

	// Direct cycle, by importing `Foo` in `Foo`
	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}

func TestRuntimeCheckCyclicImportToSelfDuringDeploy(t *testing.T) {

	runtime := NewTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}

	signerAccount := contractsAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) ([]byte, error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		OnResolveLocation: func(identifiers []Identifier, location Location) ([]ResolvedLocation, error) {
			if len(identifiers) == 0 {
				require.IsType(t, common.AddressLocation{}, location)
				addressLocation := location.(common.AddressLocation)

				require.Equal(t, contractsAddress, addressLocation.Address)
			}

			// There are no contracts in the account, so the identifiers are empty.
			return MultipleIdentifierLocationResolver(identifiers, location)
		},
		OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
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

	environment := NewBaseInterpreterEnvironment(Config{})

	nextTransactionLocation := NewTransactionLocationGenerator()

	deploy := func(name string, contract string, update bool) error {
		var txSource = DeploymentTransaction
		if update {
			txSource = UpdateTransaction
		}

		return runtime.ExecuteTransaction(
			Script{
				Source: txSource(
					name,
					[]byte(contract),
				),
			},
			Context{
				Interface:   runtimeInterface,
				Location:    nextTransactionLocation(),
				Environment: environment,
			},
		)
	}

	const fooContract = `
        import 0x0000000000000001
        access(all) contract Foo {}
    `

	err := deploy("Foo", fooContract, false)

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := RequireCheckerErrors(t, checkerErr, 1)
	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}

func TestRuntimeContractImport(t *testing.T) {

	t.Parallel()

	addressValue := Address{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	runtime := NewTestInterpreterRuntime()

	contract := []byte(`
        access(all) contract Foo {
            access(all) let x: [Int]

            access(all) fun answer(): Int {
                return 42
            }

            access(all) struct Bar {}

            init() {
                self.x = []
            }
        }`,
	)

	deploy := DeploymentTransaction("Foo", contract)

	script := []byte(`
        import Foo from 0x01

        access(all) fun main() {
            var foo: &Foo = Foo
            var x: &[Int] = Foo.x
            var bar: Foo.Bar = Foo.Bar()
        }
    `)

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{addressValue}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnCreateAccount: func(payer Address) (address Address, err error) {
			return addressValue, nil
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

	nextScriptLocation := NewScriptLocationGenerator()

	_, err = runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)
}
