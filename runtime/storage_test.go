/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeHighLevelStorage(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := common.BytesToAddress([]byte{0xCA, 0xDE})
	contract := []byte(`
       pub contract Test {

           pub resource R {
               pub var i: Int

               init(_ i: Int) {
                   self.i = i
               }

               pub fun update(_ i: Int) {
                   self.i = i
               }
           }

           pub fun createR(_ i: Int): @R {
               return <-create R(i)
           }
       }
    `)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
	  import Test from 0xCADE

	  transaction {

	      prepare(signer: AuthAccount) {
	          let rs <- {
	             "r1": <- Test.createR(3),
	             "r2": <- Test.createR(4)
	          }
	          signer.save(<-rs, to: /storage/rs)
	      }
	   }
	`)

	changeTx := []byte(`
	  import Test from 0xCADE

	  transaction {

	      prepare(signer: AuthAccount) {
	          let rs = signer.borrow<&{String: Test.R}>(from: /storage/rs)!
              rs["r1"]?.update(5)
	      }
	   }
	`)

	var accountCode []byte
	var events []cadence.Event

	type write struct {
		owner common.Address
		key   string
		value cadence.Value
	}

	var writes []write

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) (err error) {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		setCadenceValue: func(owner Address, key string, value cadence.Value) (err error) {
			writes = append(writes, write{
				owner: owner,
				key:   key,
				value: value,
			})
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	writes = nil

	err := runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

	rType := &cadence.ResourceType{
		Location: common.AddressLocation{
			Address: common.BytesToAddress([]byte{0xca, 0xde}),
			Name:    "Test",
		},
		QualifiedIdentifier: "Test.R",
		Fields: []cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "i",
				Type:       cadence.IntType{},
			},
		},
	}

	assert.Equal(t,
		[]write{
			{
				address,
				"contract\x1fTest",
				cadence.NewContract([]cadence.Value{}).WithType(&cadence.ContractType{
					Location: common.AddressLocation{
						Address: common.BytesToAddress([]byte{0xca, 0xde}),
						Name:    "Test",
					},
					QualifiedIdentifier: "Test",
					Fields:              []cadence.Field{},
					Initializers:        nil,
				}),
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: setupTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]write{
			{
				address,
				"storage\x1frs",
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key: cadence.NewString("r1"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewUInt64(0),
							cadence.NewInt(3),
						}).WithType(rType),
					},
					{
						Key: cadence.NewString("r2"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewUInt64(0),
							cadence.NewInt(4),
						}).WithType(rType),
					},
				}),
			},
		},
		writes,
	)

	writes = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: changeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]write{
			{
				address,
				"storage\x1frs",
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key: cadence.NewString("r1"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewUInt64(0),
							cadence.NewInt(5),
						}).WithType(rType),
					},
					{
						Key: cadence.NewString("r2"),
						Value: cadence.NewResource([]cadence.Value{
							cadence.NewUInt64(0),
							cadence.NewInt(4),
						}).WithType(rType),
					},
				}),
			},
		},
		writes,
	)
}

func TestRuntimeMagic(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := common.BytesToAddress([]byte{0x1})

	tx := []byte(`
	  transaction {
	      prepare(signer: AuthAccount) {
	          signer.save(1, to: /storage/one)
	      }
	   }
	`)

	var writes []testWrite

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, testWrite{
			owner,
			key,
			value,
		})
	}

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, onWrite),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: tx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]testWrite{
			{
				[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
				[]byte("storage\x1fone"),
				[]byte{
					// magic
					0x0, 0xCA, 0xDE, 0x0, 0x3,
					// CBOR
					// - tag
					0xd8, 0x98,
					// - positive bignum
					0xc2,
					// - byte string, length 1
					0x41,
					0x1,
				},
			},
		},
		writes,
	)
}

func TestRuntimeAccountStorage(t *testing.T) {
	runtime := NewInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: AuthAccount) {
           let before = signer.storageUsed
           signer.save(42, to: /storage/answer)
           let after = signer.storageUsed
           log(after != before)
        }
      }
    `)

	var loggedMessages []string

	storage := newTestStorage(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		getStorageUsed: func(_ Address) (uint64, error) {
			var amount uint64 = 0

			for _, data := range storage.storedValues {
				amount += uint64(len(data))
			}

			return amount, nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		[]string{"true"},
		loggedMessages,
	)
}

func TestRuntimePublicCapabilityBorrowTypeConfusion(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	addressString, err := hex.DecodeString("aad3e26e406987c2")
	require.NoError(t, err)

	signingAddress := common.BytesToAddress(addressString)

	deployFTContractTx := utils.DeploymentTransaction("FungibleToken", []byte(realFungibleTokenContractInterface))

	const ducContract = `
      import FungibleToken from 0xaad3e26e406987c2

      pub contract DapperUtilityCoin: FungibleToken {

    // Total supply of DapperUtilityCoins in existence
    pub var totalSupply: UFix64

    // Event that is emitted when the contract is created
    pub event TokensInitialized(initialSupply: UFix64)

    // Event that is emitted when tokens are withdrawn from a Vault
    pub event TokensWithdrawn(amount: UFix64, from: Address?)

    // Event that is emitted when tokens are deposited to a Vault
    pub event TokensDeposited(amount: UFix64, to: Address?)

    // Event that is emitted when new tokens are minted
    pub event TokensMinted(amount: UFix64)

    // Event that is emitted when tokens are destroyed
    pub event TokensBurned(amount: UFix64)

    // Event that is emitted when a new minter resource is created
    pub event MinterCreated(allowedAmount: UFix64)

    // Event that is emitted when a new burner resource is created
    pub event BurnerCreated()

    // Vault
    //
    // Each user stores an instance of only the Vault in their storage
    // The functions in the Vault and governed by the pre and post conditions
    // in FungibleToken when they are called.
    // The checks happen at runtime whenever a function is called.
    //
    // Resources can only be created in the context of the contract that they
    // are defined in, so there is no way for a malicious user to create Vaults
    // out of thin air. A special Minter resource needs to be defined to mint
    // new tokens.
    //
    pub resource Vault: FungibleToken.Provider, FungibleToken.Receiver, FungibleToken.Balance {

        // holds the balance of a users tokens
        pub var balance: UFix64

        // initialize the balance at resource creation time
        init(balance: UFix64) {
            self.balance = balance
        }

        // withdraw
        //
        // Function that takes an integer amount as an argument
        // and withdraws that amount from the Vault.
        // It creates a new temporary Vault that is used to hold
        // the money that is being transferred. It returns the newly
        // created Vault to the context that called so it can be deposited
        // elsewhere.
        //
        pub fun withdraw(amount: UFix64): @FungibleToken.Vault {
            self.balance = self.balance - amount
            emit TokensWithdrawn(amount: amount, from: self.owner?.address)
            return <-create Vault(balance: amount)
        }

        // deposit
        //
        // Function that takes a Vault object as an argument and adds
        // its balance to the balance of the owners Vault.
        // It is allowed to destroy the sent Vault because the Vault
        // was a temporary holder of the tokens. The Vault's balance has
        // been consumed and therefore can be destroyed.
        pub fun deposit(from: @FungibleToken.Vault) {
            let vault <- from as! @DapperUtilityCoin.Vault
            self.balance = self.balance + vault.balance
            emit TokensDeposited(amount: vault.balance, to: self.owner?.address)
            vault.balance = 0.0
            destroy vault
        }

        destroy() {
            DapperUtilityCoin.totalSupply = DapperUtilityCoin.totalSupply - self.balance
        }
    }

    // createEmptyVault
    //
    // Function that creates a new Vault with a balance of zero
    // and returns it to the calling context. A user must call this function
    // and store the returned Vault in their storage in order to allow their
    // account to be able to receive deposits of this token type.
    //
    pub fun createEmptyVault(): @FungibleToken.Vault {
        return <-create Vault(balance: 0.0)
    }

    pub resource Administrator {
        // createNewMinter
        //
        // Function that creates and returns a new minter resource
        //
        pub fun createNewMinter(allowedAmount: UFix64): @Minter {
            emit MinterCreated(allowedAmount: allowedAmount)
            return <-create Minter(allowedAmount: allowedAmount)
        }

        // createNewBurner
        //
        // Function that creates and returns a new burner resource
        //
        pub fun createNewBurner(): @Burner {
            emit BurnerCreated()
            return <-create Burner()
        }
    }

    // Minter
    //
    // Resource object that token admin accounts can hold to mint new tokens.
    //
    pub resource Minter {

        // the amount of tokens that the minter is allowed to mint
        pub var allowedAmount: UFix64

        // mintTokens
        //
        // Function that mints new tokens, adds them to the total supply,
        // and returns them to the calling context.
        //
        pub fun mintTokens(amount: UFix64): @DapperUtilityCoin.Vault {
            pre {
                amount > UFix64(0): "Amount minted must be greater than zero"
                amount <= self.allowedAmount: "Amount minted must be less than the allowed amount"
            }
            DapperUtilityCoin.totalSupply = DapperUtilityCoin.totalSupply + amount
            self.allowedAmount = self.allowedAmount - amount
            emit TokensMinted(amount: amount)
            return <-create Vault(balance: amount)
        }

        init(allowedAmount: UFix64) {
            self.allowedAmount = allowedAmount
        }
    }

    // Burner
    //
    // Resource object that token admin accounts can hold to burn tokens.
    //
    pub resource Burner {

        // burnTokens
        //
        // Function that destroys a Vault instance, effectively burning the tokens.
        //
        // Note: the burned tokens are automatically subtracted from the
        // total supply in the Vault destructor.
        //
        pub fun burnTokens(from: @FungibleToken.Vault) {
            let vault <- from as! @DapperUtilityCoin.Vault
            let amount = vault.balance
            destroy vault
            emit TokensBurned(amount: amount)
        }
    }

    init() {
        // we're using a high value as the balance here to make it look like we've got a ton of money,
        // just in case some contract manually checks that our balance is sufficient to pay for stuff
        self.totalSupply = 999999999.0

        let admin <- create Administrator()
        let minter <- admin.createNewMinter(allowedAmount: self.totalSupply)
        self.account.save(<-admin, to: /storage/dapperUtilityCoinAdmin)

        // mint tokens
        let tokenVault <- minter.mintTokens(amount: self.totalSupply)
        self.account.save(<-tokenVault, to: /storage/dapperUtilityCoinVault)
        destroy minter

        // Create a public capability to the stored Vault that only exposes
        // the balance field through the Balance interface
        self.account.link<&DapperUtilityCoin.Vault{FungibleToken.Balance}>(
            /public/dapperUtilityCoinBalance,
            target: /storage/dapperUtilityCoinVault
        )

        // Create a public capability to the stored Vault that only exposes
        // the deposit method through the Receiver interface
        self.account.link<&{FungibleToken.Receiver}>(
            /public/dapperUtilityCoinReceiver,
            target: /storage/dapperUtilityCoinVault
        )

        // Emit an event that shows that the contract was initialized
        emit TokensInitialized(initialSupply: self.totalSupply)
    }
}

    `

	deployDucContractTx := utils.DeploymentTransaction("DapperUtilityCoin", []byte(ducContract))

	const testContract = `
      access(all) contract TestContract{
        pub struct fake{
          pub(set) var balance: UFix64

          init(){
            self.balance = 0.0
          }
        }
        pub resource resourceConverter{
          pub fun convert(b: fake): AnyStruct {
            b.balance = 100.0
            return b
          }
        }
        pub resource resourceConverter2{
          pub fun convert(b: @AnyResource): AnyStruct {
            destroy b
            return ""
          }
        }
        access(all) fun createConverter():  @resourceConverter{
            return <- create resourceConverter();
        }
      }
    `

	deployTestContractTx := utils.DeploymentTransaction("TestContract", []byte(testContract))

	accountCodes := map[common.LocationID][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signingAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = code
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = accountCodes[location.ID()]
			return code, nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contracts

	for _, deployTx := range [][]byte{
		deployFTContractTx,
		deployDucContractTx,
		deployTestContractTx,
	} {

		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

	}

	// Run test transaction

	const testTx = `
import TestContract from 0xaad3e26e406987c2
import DapperUtilityCoin from 0xaad3e26e406987c2

transaction {
  prepare(acct: AuthAccount) {

    var tokens <- DapperUtilityCoin.createEmptyVault()

    let rc <- TestContract.createConverter()
    acct.save(<-rc, to: /storage/rc)

    acct.link<&TestContract.resourceConverter2>(/public/rc, target: /storage/rc)

    var cap=getAccount(0xaad3e26e406987c2).getCapability(/public/rc).borrow<&TestContract.resourceConverter2>()!

    var vaultx = cap.convert(b: <-tokens)

    acct.save(vaultx, to: /storage/v1)

    acct.link<&DapperUtilityCoin.Vault>(/public/v1, target: /storage/v1)

    var cap3=getAccount(0xaad3e26e406987c2).getCapability(/public/v1).borrow<&DapperUtilityCoin.Vault>()!

    log(cap3.balance)
  }
}
`

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(testTx),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	require.Error(t, err)

	require.Contains(t, err.Error(), "unexpectedly found nil while forcing an Optional value")
}
