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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCyclicImport(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			switch location {
			case common.IdentifierLocation("p1"):
				return imported1, nil
			case common.IdentifierLocation("p2"):
				return imported2, nil
			default:
				return nil, fmt.Errorf("unknown import location: %s", location)
			}
		},
		programChecked: func(location Location, duration time.Duration) {
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

	errs := checker.RequireCheckerErrors(t, checkerErr, 1)

	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr)

	// P1

	var checkerErr2 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr.Err, &checkerErr2)

	errs = checker.RequireCheckerErrors(t, checkerErr2, 1)

	var importedProgramErr2 *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr2)

	// P2

	var checkerErr3 *sema.CheckerError
	require.ErrorAs(t, importedProgramErr2.Err, &checkerErr3)

	errs = checker.RequireCheckerErrors(t, checkerErr3, 1)

	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}

func TestCheckCyclicImports(t *testing.T) {

	runtime := newTestInterpreterRuntime()

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}

	signerAccount := contractsAddress

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) ([]byte, error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (cadence.Value, error) {
		return json.Decode(runtimeInterface, b)
	}

	environment := NewBaseInterpreterEnvironment(Config{})

	nextTransactionLocation := newTransactionLocationGenerator()

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

	errs := checker.RequireCheckerErrors(t, checkerErr, 1)

	var importedProgramErr *sema.ImportedProgramError
	require.ErrorAs(t, errs[0], &importedProgramErr)

	var nestedCheckerErr *sema.CheckerError
	require.ErrorAs(t, importedProgramErr.Err, &nestedCheckerErr)

	errs = checker.RequireCheckerErrors(t, nestedCheckerErr, 1)
	require.IsType(t, &sema.CyclicImportsError{}, errs[0])
}
