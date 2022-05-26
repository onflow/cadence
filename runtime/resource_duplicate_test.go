/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

func TestRuntimeResourceDuplicationWithContractTransfer(t *testing.T) {

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

	// Deploy Fungible Token contract

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

	// Deploy Flow Token contract

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

	// Deploy Holder contract

	signerAccount = common.MustBytesToAddress([]byte{0x2})

	const holderContract = `
      import FlowToken from 0x1

      pub contract Holder {

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

	// Run transaction

	const code = `
        import FungibleToken from 0x1
        import FlowToken from 0x1
        import Holder from 0x2

        transaction {

          prepare(acct: AuthAccount) {

              // Create vault
              let vault <- FlowToken.createEmptyVault() as! @FlowToken.Vault?

              // Move vault into the contract
              Holder.content <-! vault

              // Save the contract into storage (invalid, even if same account)
              acct.save(Holder as AnyStruct, to: /storage/holder)

              // Move vault back out of the contract
              let vault2 <- Holder.content <- nil
              let unwrappedVault2 <- vault2!

              // Load the contract back from storage
              let dupeContract = acct.load<AnyStruct>(from: /storage/holder)! as! Holder

              // Move the vault of of the duplicated contract
              let dupeVault <- dupeContract.content <- nil
              let unwrappedDupeVault <- dupeVault!

              // Deposit the duplicated vault into the original vault
              unwrappedVault2.deposit(from: <- unwrappedDupeVault)

              destroy unwrappedVault2
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
