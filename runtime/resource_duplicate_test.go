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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeResourceDuplicationUsingDestructorIteration(t *testing.T) {
	t.Parallel()

	t.Run("Reported error", func(t *testing.T) {

		t.Parallel()

		script := `
	// This Vault class is from Flow docs, used as our "victim" in this example
	access(all) resource Vault {
		// Balance of a user's Vault
		// we use unsigned fixed point numbers for balances
		// because they can represent decimals and do not allow negative values
		access(all) var balance: UFix64
	
		init(balance: UFix64) {
			self.balance = balance
		}
	
		access(all) fun withdraw(amount: UFix64): @Vault {
			self.balance = self.balance - amount
			return <-create Vault(balance: amount)
		}
	
		access(all) fun deposit(from: @Vault) {
			self.balance = self.balance + from.balance
			destroy from
		}
	}
	
	// --- this code actually makes use of the vuln ---
	access(all) resource DummyResource {
		access(all) var dictRef: &{Bool: AnyResource};
		access(all) var arrRef: &[Vault];
		access(all) var victim: @Vault;
		init(dictRef: &{Bool: AnyResource}, arrRef: &[Vault], victim: @Vault) {
			self.dictRef = dictRef;
			self.arrRef = arrRef;
			self.victim <- victim;
		}
	
		destroy() {
			self.arrRef.append(<- self.victim)
			self.dictRef[false] <-> self.dictRef[true]; // This screws up the destruction order
		}
	}
	
	access(all) fun duplicateResource(victim1: @Vault, victim2: @Vault): @[Vault]{
		let arr : @[Vault] <- [];
		let dict: @{Bool: DummyResource} <- { }
		let ref = &dict as &{Bool: AnyResource};
		let arrRef = &arr as &[Vault];
	
		var v1: @DummyResource? <- create DummyResource(dictRef: ref, arrRef: arrRef, victim: <- victim1);
		dict[false] <-> v1;
		destroy v1;
	
		var v2: @DummyResource? <- create DummyResource(dictRef: ref, arrRef: arrRef, victim: <- victim2);
		dict[true] <-> v2;
		destroy v2;
	
		destroy dict // Trigger the destruction chain where dict[false] will be destructed twice
		return <- arr;
	}
	
	// --- end of vuln code ---
	
	access(all) fun main() {
	
		var v1 <- create Vault(balance: 1000.0); // This will be duplicated
		var v2 <- create Vault(balance: 1.0); // This will be lost
		var v3 <- create Vault(balance: 0.0); // We'll collect the spoils here
	
		// The call will return an array of [v1, v1]
		var res <- duplicateResource(victim1: <- v1, victim2: <-v2)
	
		v3.deposit(from: <- res.removeLast());
		v3.deposit(from: <- res.removeLast());
		destroy res;
	
		log(v3.balance);
		destroy v3;
	}`

		runtime := newTestInterpreterRuntime()

		accountCodes := map[common.Location][]byte{}

		var events []cadence.Event

		signerAccount := common.MustBytesToAddress([]byte{0x1})

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			log: func(s string) {
				assert.Fail(t, "we should not reach this point")
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		var checkerErr *sema.CheckerError
		require.ErrorAs(t, err, &checkerErr)

		errs := checker.RequireCheckerErrors(t, checkerErr, 2)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("simplified", func(t *testing.T) {

		t.Parallel()

		script := `
		access(all) resource Vault {
            access(all) var balance: UFix64
            access(all) var dictRef: &{Bool: Vault};

            init(balance: UFix64, _ dictRef: &{Bool: Vault}) {
                self.balance = balance
                self.dictRef = dictRef;
            }

            access(all) fun withdraw(amount: UFix64): @Vault {
                self.balance = self.balance - amount
                return <-create Vault(balance: amount, self.dictRef)
            }

            access(all) fun deposit(from: @Vault) {
                self.balance = self.balance + from.balance
                destroy from
            }

            destroy() {
                self.dictRef[false] <-> self.dictRef[true]; // This screws up the destruction order
            }
        }

        access(all) fun main(): UFix64 {

            let dict: @{Bool: Vault} <- { }
            let dictRef = &dict as &{Bool: Vault};

            var v1 <- create Vault(balance: 1000.0, dictRef); // This will be duplicated
            var v2 <- create Vault(balance: 1.0, dictRef); // This will be lost

            var v1Ref = &v1 as &Vault

			destroy dict.insert(key: false, <- v1)
		    destroy dict.insert(key: true, <- v2)

            destroy dict;

            // v1 is not destroyed!
            return v1Ref.balance
        }`

		runtime := newTestInterpreterRuntime()

		accountCodes := map[common.Location][]byte{}

		var events []cadence.Event

		signerAccount := common.MustBytesToAddress([]byte{0x1})

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		var checkerErr *sema.CheckerError
		require.ErrorAs(t, err, &checkerErr)

		errs := checker.RequireCheckerErrors(t, checkerErr, 3)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		assert.IsType(t, &sema.InvalidatedResourceReferenceError{}, errs[2])
	})

	t.Run("forEachKey", func(t *testing.T) {

		t.Parallel()

		script := `
	access(all) resource R{}

	access(all) fun main() {
		var dict: @{Int: R} <- {}

		var r1: @R? <- create R()
		var r2: @R? <- create R()
		var r3: @R? <- create R()

		dict[0] <-> r1
		dict[1] <-> r2
		dict[2] <-> r3

		destroy r1 
		destroy r2 
		destroy r3

		let acc = getAuthAccount(0x1)
		acc.save(<-dict, to: /storage/foo)

		let ref = acc.borrow<&{Int: R}>(from: /storage/foo)!

		ref.forEachKey(fun(i: Int): Bool {
			var r4: @R? <- create R()
			ref[i+1] <-> r4
			destroy r4
			return true
		})
	}`

		runtime := newTestInterpreterRuntime()

		accountCodes := map[common.Location][]byte{}

		var events []cadence.Event

		signerAccount := common.MustBytesToAddress([]byte{0x1})

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		errs := checker.RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		script := `
		access(all) resource Vault {
            access(all) var balance: UFix64
            access(all) var arrRef: &[Vault]

            init(balance: UFix64, _ arrRef: &[Vault]) {
                self.balance = balance
                self.arrRef = arrRef;
            }

            access(all) fun withdraw(amount: UFix64): @Vault {
                self.balance = self.balance - amount
                return <-create Vault(balance: amount, self.arrRef)
            }

            access(all) fun deposit(from: @Vault) {
                self.balance = self.balance + from.balance
                destroy from
            }

            destroy() {
                self.arrRef.append(<-create Vault(balance: 0.0, self.arrRef))
            }
        }

        access(all) fun main(): UFix64 {

            let arr: @[Vault] <- []
            let arrRef = &arr as &[Vault];

            var v1 <- create Vault(balance: 1000.0, arrRef); // This will be duplicated
            var v2 <- create Vault(balance: 1.0, arrRef); // This will be lost

            var v1Ref = &v1 as &Vault

			arr.append(<- v1)
		    arr.append(<- v2)

            destroy arr

            // v1 is not destroyed!
            return v1Ref.balance
        }`

		runtime := newTestInterpreterRuntime()

		accountCodes := map[common.Location][]byte{}

		var events []cadence.Event

		signerAccount := common.MustBytesToAddress([]byte{0x1})

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: storage,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		}

		_, err := runtime.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		RequireError(t, err)

		var checkerErr *sema.CheckerError
		require.ErrorAs(t, err, &checkerErr)

		errs := checker.RequireCheckerErrors(t, checkerErr, 1)

		assert.IsType(t, &sema.InvalidatedResourceReferenceError{}, errs[0])
	})
}

func TestRuntimeResourceDuplicationWithContractTransfer(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	var events []cadence.Event

	signerAccount := common.MustBytesToAddress([]byte{0x1})

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(nil, b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy Fungible Token contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
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

          prepare(acct: AuthAccount) {

              // Create vault
              let vault <- FlowToken.createEmptyVault() as! @FlowToken.Vault?

              // Move vault into the contract
              Holder.setContent(<-vault)

              // Save the contract into storage (invalid, even if same account)
              acct.save(Holder as AnyStruct, to: /storage/holder)

              // Move vault back out of the contract
              let vault2 <- Holder.swapContent(nil)
              let unwrappedVault2 <- vault2!

              // Load the contract back from storage
              let dupeContract = acct.load<AnyStruct>(from: /storage/holder)! as! Holder

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

	var nonTransferableValueError interpreter.NonTransferableValueError
	require.ErrorAs(t, err, &nonTransferableValueError)
}
