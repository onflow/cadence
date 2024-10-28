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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/tests/runtime_utils"
	"github.com/onflow/cadence/tests/utils"
)

func TestRuntimeDeployedContracts(t *testing.T) {
	t.Parallel()

	contractCode := `
		access(all) contract Test {
			access(all) struct A {}
			access(all) resource B {}
			access(all) event C()

			init() {}
		}
	`

	script :=
		`
		transaction {
			prepare(signer: &Account) {
				let deployedContract = signer.contracts.get(name: "Test")
				assert(deployedContract!.name == "Test")

				let expected: {String: Void} =
					{ "A.2a00000000000000.Test.A": ()
					, "A.2a00000000000000.Test.B": ()
					, "A.2a00000000000000.Test.C": ()
					}
				let types = deployedContract!.publicTypes()
				assert(types.length == 3)

				for type in types {
					assert(expected[type.identifier] != nil, message: type.identifier)
				}
			}
		}
		`

	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) ([]byte, error) {
			return accountCodes[location], nil
		},
		OnGetAccountContractNames: func(_ Address) ([]string, error) {
			names := make([]string, 0, len(accountCodes))
			for location := range accountCodes {
				names = append(names, location.String())
			}
			return names, nil
		},
		OnEmitEvent: func(_ cadence.Event) error {
			return nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnProgramLog: func(msg string) {},
		Storage:      NewTestLedger(nil, nil),
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	newContext := func() Context {
		return Context{Interface: runtimeInterface, Location: nextTransactionLocation()}
	}

	// deploy the contract
	err := rt.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction("Test", []byte(contractCode)),
		},
		newContext(),
	)
	require.NoError(t, err)

	// grab the public types from the deployed contract
	err = rt.ExecuteTransaction(
		Script{
			Source: []byte(script),
		},
		newContext(),
	)

	require.NoError(t, err)
}
