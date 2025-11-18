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
	. "github.com/onflow/cadence/test_utils/runtime_utils"
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

	contractCode2 := `
        access(all) contract Test2 {
            access(all) struct A2 {}
            access(all) resource B2 {}
            access(all) event C2()

            init() {}
        }
    `

	tx := `
        transaction {
            prepare(signer: &Account) {
                let deployedContract = signer.contracts.get(name: "Test")
                assert(deployedContract!.name == "Test")

                let expected: {String: Void} = {
                    "A.2a00000000000000.Test.A": (),
                    "A.2a00000000000000.Test.B": (),
                    "A.2a00000000000000.Test.C": ()
                }
                let types = deployedContract!.publicTypes()
                assert(types.length == 3)

                for type in types {
                    assert(
                        expected[type.identifier] != nil,
                        message: "type \(type.identifier) missing"
                    )
                }
            }
        }
    `

	tx2 := `
        transaction {
            prepare(signer: &Account) {
                let deployedContract = signer.contracts.get(name: "Test2")
                assert(deployedContract!.name == "Test2")

                let expected: {String: Void} = {
                    "A.2a00000000000000.Test2.A2": (),
                    "A.2a00000000000000.Test2.B2": (),
                    "A.2a00000000000000.Test2.C2": ()
                }
                let types = deployedContract!.publicTypes()
                assert(types.length == 3)

                for type in types {
                    assert(
                        expected[type.identifier] != nil,
                        message: "type \(type.identifier) missing"
                    )
                }
            }
        }
    `

	rt := NewTestRuntime()
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

	// Deploy contracts
	for name, code := range map[string]string{
		"Test":  contractCode,
		"Test2": contractCode2,
	} {

		err := rt.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction(name, []byte(code)),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
				UseVM:     *compile,
			},
		)
		require.NoError(t, err)
	}

	// Run test transactions
	for _, script := range []string{tx, tx2} {

		err := rt.ExecuteTransaction(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
				UseVM:     *compile,
			},
		)
		require.NoError(t, err)
	}

}
