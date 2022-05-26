package runtime

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestResourceDuplicate(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	accountCodes := map[common.LocationID][]byte{}

	var events []cadence.Event

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location.ID()], nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location.ID()], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// ---------------- Deploy Fungible Token contract ----------------

	err := runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"FungibleToken",
				[]byte(realFungibleTokenContractInterface),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// ---------------- Deploy Flow Token contract ----------------

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(fmt.Sprintf(
				`
                  transaction {

                      prepare(signer: AuthAccount) {
                          signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
                      }
                  }
                `,
				hex.EncodeToString([]byte(realFlowContract)),
			)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// --------------- Deploy Holder contract ----------------

	signerAccount = common.MustBytesToAddress([]byte{0x2})

	const holderContract = `
		import FlowToken from 0x1

		access(all) contract Holder {
			pub (set) var content: @FlowToken.Vault?
			init() {
				self.content <- nil
			}
		}
	`
	err = runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"Holder",
				[]byte(holderContract),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// --------------------------------

	code := `
		import FlowToken from 0x1
		import Holder from 0x2

		transaction {

		  prepare(acct: AuthAccount) {

			  //get current vault
			  var vault <- FlowToken.createEmptyVault() as! @FlowToken.Vault?

			  //put it to contract
			  Holder.content <-! vault

			  //save to storage
			  acct.save(Holder as AnyStruct, to:/storage/dnz)

			  //remove vault
			  var exvault <- Holder.content <- nil
			  var unwrappedExVault <- exvault!

			  //abracadabra
			  var dupe = acct.load<AnyStruct>(from:/storage/dnz)!
			  var dupeContract = dupe as! Holder
			  var dupeVault <- dupeContract.content <- nil

			  unwrappedExVault.deposit(from: <- dupeVault!)

			  //put out vault back
			  acct.save(<-unwrappedExVault, to:/storage/flowTokenVault)
		  }

		  execute {
		  }
		}
		`

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(code),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.Error(t, err)

	var nonTransferableValueError interpreter.NonTransferableValueError
	require.ErrorAs(t, err, &nonTransferableValueError)
}
