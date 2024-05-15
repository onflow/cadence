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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeResourceDuplicationWithContractTransfer(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: storage,
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
			events = append(events, event)
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy Fungible Token contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
				"FungibleToken",
				[]byte(modifiedFungibleTokenContractInterface),
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

                      prepare(signer: auth(Storage, Contracts, Capabilities) &Account) {
                          signer.contracts.add(name: "FlowToken", code: "%s".decodeHex(), signer)
                      }
                  }
                `,
				hex.EncodeToString([]byte(modifiedFlowContract)),
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

      access(all) contract Holder {

          access(all) var content: @FlowToken.Vault?

          init() {
              self.content <- nil
          }

          access(all) fun setContent(_ vault: @FlowToken.Vault?) {
            self.content <-! vault
          }

          access(all) fun swapContent(_ vault: @FlowToken.Vault?): @FlowToken.Vault? {
            let oldVault <- self.content <- vault
            return <-oldVault
          }

      }
    `
	err = runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
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

          prepare(acct: auth(Storage) &Account) {

              // Create vault
              let vault <- FlowToken.createEmptyVault() as! @FlowToken.Vault?

              // Move vault into the contract
              Holder.setContent(<-vault)

              // Save the contract into storage (invalid, even if same account)
              acct.storage.save(Holder as AnyStruct, to: /storage/holder)

              // Move vault back out of the contract
              let vault2 <- Holder.swapContent(nil)
              let unwrappedVault2 <- vault2!

              // Load the contract back from storage
              let dupeContract = acct.storage.load<AnyStruct>(from: /storage/holder)! as! Holder

              // Move the vault of of the duplicated contract
              let dupeVault <- dupeContract.swapContent(nil)
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
	RequireError(t, err)

	var invalidMoveError *sema.InvalidMoveError
	require.ErrorAs(t, err, &invalidMoveError)
}
