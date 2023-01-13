package runtime

import (
	"fmt"
	"testing"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestDeployedContracts(t *testing.T) {
	t.Parallel()

	contractCode := `
		pub contract Test {
			pub struct A {}
			pub struct B {}
			pub struct C {}

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
		log: func(msg string) {
			fmt.Println(msg)
		},
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
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)
}
