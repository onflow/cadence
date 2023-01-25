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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestDeployedContracts(t *testing.T) {
	t.Parallel()

	contractCode := `
		pub contract Test {
			pub struct A {}
			pub resource B {}
			pub event C()

			init() {}
		}
	`

	script :=
		`
		transaction {
			prepare(signer: AuthAccount) {
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

	rt := newTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		getAccountContractCode: func(address Address, name string) ([]byte, error) {
			location := common.AddressLocation{
				Address: address, Name: name,
			}
			return accountCodes[location], nil
		},
		getAccountContractNames: func(_ Address) ([]string, error) {
			names := make([]string, 0, len(accountCodes))
			for location := range accountCodes {
				names = append(names, location.String())
			}
			return names, nil
		},
		emitEvent: func(_ cadence.Event) error {
			return nil
		},
		updateAccountContractCode: func(address common.Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address, Name: name,
			}
			accountCodes[location] = code
			return nil
		},
		log:     func(msg string) {},
		storage: newTestLedger(nil, nil),
	}

	nextTransactionLocation := newTransactionLocationGenerator()
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
