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
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func withWritesToStorage(
	arrayElementCount int,
	storageItemCount int,
	onWrite func(owner, key, value []byte),
	handler func(runtimeStorage *runtimeStorage),
) {
	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, onWrite),
	}

	runtimeStorage := newRuntimeStorage(runtimeInterface)

	array := interpreter.NewArrayValueUnownedNonCopying(
		interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeInt,
		},
	)

	inter, _ := interpreter.NewInterpreter(nil, utils.TestLocation)

	for i := 0; i < arrayElementCount; i++ {
		array.Append(inter, nil, interpreter.NewIntValueFromInt64(int64(i)))
	}

	address := common.BytesToAddress([]byte{0x1})

	for i := 0; i < storageItemCount; i++ {
		runtimeStorage.cache[StorageKey{
			Address: address,
			Key:     strconv.Itoa(i),
		}] = CacheEntry{
			MustWrite: true,
			Value:     array,
		}
	}

	handler(runtimeStorage)
}

func TestRuntimeStorageWriteCached(t *testing.T) {

	t.Parallel()

	var writes []testWrite

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, testWrite{
			owner: owner,
			key:   key,
			value: value,
		})
	}

	const arrayElementCount = 100
	const storageItemCount = 100
	withWritesToStorage(arrayElementCount, storageItemCount, onWrite, func(runtimeStorage *runtimeStorage) {
		err := runtimeStorage.writeCached(nil)
		require.NoError(t, err)

		require.Len(t, writes, storageItemCount)
	})
}

func TestRuntimeStorageWriteCachedIsDeterministic(t *testing.T) {

	t.Parallel()

	var writes []testWrite

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, testWrite{
			owner: owner,
			key:   key,
			value: value,
		})
	}

	const arrayElementCount = 100
	const storageItemCount = 100
	withWritesToStorage(arrayElementCount, storageItemCount, onWrite, func(runtimeStorage *runtimeStorage) {
		err := runtimeStorage.writeCached(nil)
		require.NoError(t, err)

		previousWrites := make([]testWrite, len(writes))
		copy(previousWrites, writes)

		// verify for 10 times and check the writes are always deterministic
		for i := 0; i < 10; i++ {
			// test that writing again should produce the same result
			writes = nil
			err := runtimeStorage.writeCached(nil)
			require.NoError(t, err)

			for i, previousWrite := range previousWrites {
				// compare the new write with the old write
				require.Equal(t, previousWrite, writes[i])
			}

			// no additional items
			require.Len(t, writes, len(previousWrites))
		}
	})
}

func BenchmarkRuntimeStorageWriteCached(b *testing.B) {
	var writes []testWrite

	onWrite := func(owner, key, value []byte) {
		writes = append(writes, testWrite{
			owner: owner,
			key:   key,
			value: value,
		})
	}

	const arrayElementCount = 100
	const storageItemCount = 100
	withWritesToStorage(arrayElementCount, storageItemCount, onWrite, func(runtimeStorage *runtimeStorage) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			writes = nil
			err := runtimeStorage.writeCached(nil)
			require.NoError(b, err)

			require.Len(b, writes, storageItemCount)
		}
	})
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
					0x0, 0xCA, 0xDE, 0x0, 0x5,
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

	t.Parallel()

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

    let rc <- TestContract.createConverter()
    acct.save(<-rc, to: /storage/rc)

    acct.link<&TestContract.resourceConverter2>(/public/rc, target: /storage/rc)

    let optRef = getAccount(0xaad3e26e406987c2).getCapability(/public/rc).borrow<&TestContract.resourceConverter2>()

    if let ref = optRef {

      var tokens <- DapperUtilityCoin.createEmptyVault()

      var vaultx = ref.convert(b: <-tokens)

      acct.save(vaultx, to: /storage/v1)

      acct.link<&DapperUtilityCoin.Vault>(/public/v1, target: /storage/v1)

      var cap3 = getAccount(0xaad3e26e406987c2).getCapability(/public/v1).borrow<&DapperUtilityCoin.Vault>()!

      log(cap3.balance)
    } else {
      panic("failed to borrow resource converter")
    }
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

	require.Contains(t, err.Error(), "failed to borrow resource converter")
}

func TestRuntimeStorageReadAndBorrow(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	storage := newTestStorage(nil, nil)

	signer := common.BytesToAddress([]byte{0x42})

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Store a value and link a capability

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction {
                 prepare(signer: AuthAccount) {
                     signer.save(42, to: /storage/test)
                     signer.link<&Int>(
                         /private/test,
                         target: /storage/test
                     )
                 }
              }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	t.Run("read stored, existing", func(t *testing.T) {

		value, err := runtime.ReadStored(
			signer,
			cadence.Path{
				Domain:     "storage",
				Identifier: "test",
			},
			Context{
				// NOTE: no location
				Interface: runtimeInterface,
			},
		)
		require.NoError(t, err)
		require.Equal(t, cadence.NewOptional(cadence.NewInt(42)), value)
	})

	t.Run("read stored, non-existing", func(t *testing.T) {

		value, err := runtime.ReadStored(
			signer,
			cadence.Path{
				Domain:     "storage",
				Identifier: "other",
			},
			Context{
				// NOTE: no location
				Interface: runtimeInterface,
			},
		)
		require.NoError(t, err)
		require.Equal(t, cadence.NewOptional(nil), value)
	})

	t.Run("read linked, existing", func(t *testing.T) {

		value, err := runtime.ReadLinked(
			signer,
			cadence.Path{
				Domain:     "private",
				Identifier: "test",
			},
			Context{
				Location:  utils.TestLocation,
				Interface: runtimeInterface,
			},
		)
		require.NoError(t, err)
		require.Equal(t, cadence.NewOptional(cadence.NewInt(42)), value)
	})

	t.Run("read linked, non-existing", func(t *testing.T) {

		value, err := runtime.ReadLinked(
			signer,
			cadence.Path{
				Domain:     "private",
				Identifier: "other",
			},
			Context{
				Location:  utils.TestLocation,
				Interface: runtimeInterface,
			},
		)
		require.NoError(t, err)
		require.Equal(t, cadence.NewOptional(nil), value)
	})
}

func TestRuntimeTopShotBatchTransfer(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	accountCodes := map[common.LocationID]string{

		"A.1d7e57aa55817448.NonFungibleToken": `

pub contract interface NonFungibleToken {

    // The total number of tokens of this type in existence
    pub var totalSupply: UInt64

    // Event that emitted when the NFT contract is initialized
    //
    pub event ContractInitialized()

    // Event that is emitted when a token is withdrawn,
    // indicating the owner of the collection that it was withdrawn from.
    //
    pub event Withdraw(id: UInt64, from: Address?)

    // Event that emitted when a token is deposited to a collection.
    //
    // It indicates the owner of the collection that it was deposited to.
    //
    pub event Deposit(id: UInt64, to: Address?)

    // Interface that the NFTs have to conform to
    //
    pub resource interface INFT {
        // The unique ID that each NFT has
        pub let id: UInt64
    }

    // Requirement that all conforming NFT smart contracts have
    // to define a resource called NFT that conforms to INFT
    pub resource NFT: INFT {
        pub let id: UInt64
    }

    // Interface to mediate withdraws from the Collection
    //
    pub resource interface Provider {
        // withdraw removes an NFT from the collection and moves it to the caller
        pub fun withdraw(withdrawID: UInt64): @NFT {
            post {
                result.id == withdrawID: "The ID of the withdrawn token must be the same as the requested ID"
            }
        }
    }

    // Interface to mediate deposits to the Collection
    //
    pub resource interface Receiver {

        // deposit takes an NFT as an argument and adds it to the Collection
        //
		pub fun deposit(token: @NFT)
    }

    // Interface that an account would commonly
    // publish for their collection
    pub resource interface CollectionPublic {
        pub fun deposit(token: @NFT)
        pub fun getIDs(): [UInt64]
        pub fun borrowNFT(id: UInt64): &NFT
    }

    // Requirement for the the concrete resource type
    // to be declared in the implementing contract
    //
    pub resource Collection: Provider, Receiver, CollectionPublic {

        // Dictionary to hold the NFTs in the Collection
        pub var ownedNFTs: @{UInt64: NFT}

        // withdraw removes an NFT from the collection and moves it to the caller
        pub fun withdraw(withdrawID: UInt64): @NFT

        // deposit takes a NFT and adds it to the collections dictionary
        // and adds the ID to the id array
        pub fun deposit(token: @NFT)

        // getIDs returns an array of the IDs that are in the collection
        pub fun getIDs(): [UInt64]

        // Returns a borrowed reference to an NFT in the collection
        // so that the caller can read data and call methods from it
        pub fun borrowNFT(id: UInt64): &NFT {
            pre {
                self.ownedNFTs[id] != nil: "NFT does not exist in the collection!"
            }
        }
    }

    // createEmptyCollection creates an empty Collection
    // and returns it to the caller so that they can own NFTs
    pub fun createEmptyCollection(): @Collection {
        post {
            result.getIDs().length == 0: "The created collection must be empty!"
        }
    }
}
`,
	}

	const topShotContract = `
import NonFungibleToken from 0x1d7e57aa55817448

pub contract TopShot: NonFungibleToken {

    // -----------------------------------------------------------------------
    // TopShot contract Event definitions
    // -----------------------------------------------------------------------

    // emitted when the TopShot contract is created
    pub event ContractInitialized()

    // emitted when a new Play struct is created
    pub event PlayCreated(id: UInt32, metadata: {String:String})
    // emitted when a new series has been triggered by an admin
    pub event NewSeriesStarted(newCurrentSeries: UInt32)

    // Events for Set-Related actions
    //
    // emitted when a new Set is created
    pub event SetCreated(setID: UInt32, series: UInt32)
    // emitted when a new play is added to a set
    pub event PlayAddedToSet(setID: UInt32, playID: UInt32)
    // emitted when a play is retired from a set and cannot be used to mint
    pub event PlayRetiredFromSet(setID: UInt32, playID: UInt32, numMoments: UInt32)
    // emitted when a set is locked, meaning plays cannot be added
    pub event SetLocked(setID: UInt32)
    // emitted when a moment is minted from a set
    pub event MomentMinted(momentID: UInt64, playID: UInt32, setID: UInt32, serialNumber: UInt32)

    // events for Collection-related actions
    //
    // emitted when a moment is withdrawn from a collection
    pub event Withdraw(id: UInt64, from: Address?)
    // emitted when a moment is deposited into a collection
    pub event Deposit(id: UInt64, to: Address?)

    // emitted when a moment is destroyed
    pub event MomentDestroyed(id: UInt64)

    // -----------------------------------------------------------------------
    // TopShot contract-level fields
    // These contain actual values that are stored in the smart contract
    // -----------------------------------------------------------------------

    // Series that this set belongs to
    // Series is a concept that indicates a group of sets through time
    // Many sets can exist at a time, but only one series
    pub var currentSeries: UInt32

    // variable size dictionary of Play structs
    access(self) var playDatas: {UInt32: Play}

    // variable size dictionary of SetData structs
    access(self) var setDatas: {UInt32: SetData}

    // variable size dictionary of Set resources
    access(self) var sets: @{UInt32: Set}

    // the ID that is used to create Plays.
    // Every time a Play is created, playID is assigned
    // to the new Play's ID and then is incremented by 1.
    pub var nextPlayID: UInt32

    // the ID that is used to create Sets. Every time a Set is created
    // setID is assigned to the new set's ID and then is incremented by 1.
    pub var nextSetID: UInt32

    // the total number of Top shot moment NFTs that have been created
    // Because NFTs can be destroyed, it doesn't necessarily mean that this
    // reflects the total number of NFTs in existence, just the number that
    // have been minted to date.
    // Is also used as global moment IDs for minting
    pub var totalSupply: UInt64

    // -----------------------------------------------------------------------
    // TopShot contract-level Composite Type DEFINITIONS
    // -----------------------------------------------------------------------
    // These are just definitions for types that this contract
    // and other accounts can use. These definitions do not contain
    // actual stored values, but an instance (or object) of one of these types
    // can be created by this contract that contains stored values
    // -----------------------------------------------------------------------

    // Play is a Struct that holds metadata associated
    // with a specific NBA play, like the legendary moment when
    // Ray Allen hit the 3 to tie the Heat and Spurs in the 2013 finals game 6
    // or when Lance Stephenson blew in the ear of Lebron James
    //
    // Moment NFTs will all reference a single Play as the owner of
    // its metadata. The Plays are publicly accessible, so anyone can
    // read the metadata associated with a specific play ID
    //
    pub struct Play {

        // the unique ID that the Play has
        pub let playID: UInt32

        // Stores all the metadata about the Play as a string mapping
        // This is not the long term way we will do metadata. Just a temporary
        // construct while we figure out a better way to do metadata
        //
        pub let metadata: {String: String}

        init(metadata: {String: String}) {
            pre {
                metadata.length != 0: "New Play Metadata cannot be empty"
            }
            self.playID = TopShot.nextPlayID
            self.metadata = metadata

            // increment the ID so that it isn't used again
            TopShot.nextPlayID = TopShot.nextPlayID + UInt32(1)

            emit PlayCreated(id: self.playID, metadata: metadata)
        }
    }

    // A Set is a grouping of plays that have occurred in the real world
    // that make up a related group of collectibles, like sets of baseball
    // or Magic cards.
    //
    // SetData is a struct that is stored in a public field of the contract.
    // This is to allow anyone to be able to query the constant information
    // about a set but not have the ability to modify any data in the
    // private set resource
    //
    pub struct SetData {

        // unique ID for the set
        pub let setID: UInt32

        // Name of the Set
        // ex. "Times when the Toronto Raptors choked in the playoffs"
        pub let name: String

        // Series that this set belongs to
        // Series is a concept that indicates a group of sets through time
        // Many sets can exist at a time, but only one series
        pub let series: UInt32

        init(name: String) {
            pre {
                name.length > 0: "New Set name cannot be empty"
            }
            self.setID = TopShot.nextSetID
            self.name = name
            self.series = TopShot.currentSeries

            // increment the setID so that it isn't used again
            TopShot.nextSetID = TopShot.nextSetID + UInt32(1)

            emit SetCreated(setID: self.setID, series: self.series)
        }
    }

    // Set is a resource type that contains the functions to add and remove
    // plays from a set and mint moments.
    //
    // It is stored in a private field in the contract so that
    // the admin resource can call its methods and that there can be
    // public getters for some of its fields
    //
    // The admin can add Plays to a set so that the set can mint moments
    // that reference that playdata.
    // The moments that are minted by a set will be listed as belonging to
    // the set that minted it, as well as the Play it references
    //
    // The admin can also retire plays from the set, meaning that the retired
    // play can no longer have moments minted from it.
    //
    // If the admin locks the Set, then no more plays can be added to it, but
    // moments can still be minted.
    //
    // If retireAll() and lock() are called back to back,
    // the Set is closed off forever and nothing more can be done with it
    pub resource Set {

        // unique ID for the set
        pub let setID: UInt32

        // Array of plays that are a part of this set
        // When a play is added to the set, its ID gets appended here
        // The ID does not get removed from this array when a play is retired
        pub var plays: [UInt32]

        // Indicates if a play in this set can be minted
        // A play is set to false when it is added to a set
        // to indicate that it is still active
        // When the play is retired, this is set to true and cannot be changed
        pub var retired: {UInt32: Bool}

        // Indicates if the set is currently locked
        // When a set is created, it is unlocked
        // and plays are allowed to be added to it
        // When a set is locked, plays cannot be added
        // A set can never be changed from locked to unlocked
        // The decision to lock it is final
        // If a set is locked, plays cannot be added, but
        // moments can still be minted from plays
        // that already had been added to it.
        pub var locked: Bool

        // Indicates the number of moments
        // that have been minted per play in this set
        // When a moment is minted, this value is stored in the moment to
        // show where in the play set it is so far. ex. 13 of 60
        pub var numberMintedPerPlay: {UInt32: UInt32}

        init(name: String) {
            self.setID = TopShot.nextSetID
            self.plays = []
            self.retired = {}
            self.locked = false
            self.numberMintedPerPlay = {}

            // Create a new SetData for this Set and store it in contract storage
            TopShot.setDatas[self.setID] = SetData(name: name)
        }

        // addPlay adds a play to the set
        //
        // Parameters: playID: The ID of the play that is being added
        //
        // Pre-Conditions:
        // The play needs to be an existing play
        // The set needs to be not locked
        // The play can't have already been added to the set
        //
        pub fun addPlay(playID: UInt32) {
            pre {
                TopShot.playDatas[playID] != nil: "Cannot add the Play to Set: Play doesn't exist"
                !self.locked: "Cannot add the play to the Set after the set has been locked"
                self.numberMintedPerPlay[playID] == nil: "The play has already beed added to the set"
            }

            // Add the play to the array of plays
            self.plays.append(playID)

            // Open the play up for minting
            self.retired[playID] = false

            // Initialize the moment count to zero
            self.numberMintedPerPlay[playID] = 0

            emit PlayAddedToSet(setID: self.setID, playID: playID)
        }

        // addPlays adds multiple plays to the set
        //
        // Parameters: playIDs: The IDs of the plays that are being added
        //                      as an array
        //
        pub fun addPlays(playIDs: [UInt32]) {
            for play in playIDs {
                self.addPlay(playID: play)
            }
        }

        // retirePlay retires a play from the set so that it can't mint new moments
        //
        // Parameters: playID: The ID of the play that is being retired
        //
        // Pre-Conditions:
        // The play needs to be an existing play that is currently open for minting
        //
        pub fun retirePlay(playID: UInt32) {
            pre {
                self.retired[playID] != nil: "Cannot retire the Play: Play doesn't exist in this set!"
            }

            if !self.retired[playID]! {
                self.retired[playID] = true

                emit PlayRetiredFromSet(setID: self.setID, playID: playID, numMoments: self.numberMintedPerPlay[playID]!)
            }
        }

        // retireAll retires all the plays in the set
        // Afterwards, none of the retired plays will be able to mint new moments
        //
        pub fun retireAll() {
            for play in self.plays {
                self.retirePlay(playID: play)
            }
        }

        // lock() locks the set so that no more plays can be added to it
        //
        // Pre-Conditions:
        // The set cannot already have been locked
        pub fun lock() {
            if !self.locked {
                self.locked = true
                emit SetLocked(setID: self.setID)
            }
        }

        // mintMoment mints a new moment and returns the newly minted moment
        //
        // Parameters: playID: The ID of the play that the moment references
        //
        // Pre-Conditions:
        // The play must exist in the set and be allowed to mint new moments
        //
        // Returns: The NFT that was minted
        //
        pub fun mintMoment(playID: UInt32): @NFT {
            pre {
                self.retired[playID] != nil: "Cannot mint the moment: This play doesn't exist"
                !self.retired[playID]!: "Cannot mint the moment from this play: This play has been retired"
            }

            // get the number of moments that have been minted for this play
            // to use as this moment's serial number
            let numInPlay = self.numberMintedPerPlay[playID]!

            // mint the new moment
            let newMoment: @NFT <- create NFT(serialNumber: numInPlay + UInt32(1),
                                              playID: playID,
                                              setID: self.setID)

            // Increment the count of moments minted for this play
            self.numberMintedPerPlay[playID] = numInPlay + UInt32(1)

            return <-newMoment
        }

        // batchMintMoment mints an arbitrary quantity of moments
        // and returns them as a Collection
        //
        // Parameters: playID: the ID of the play that the moments are minted for
        //             quantity: The quantity of moments to be minted
        //
        // Returns: Collection object that contains all the moments that were minted
        //
        pub fun batchMintMoment(playID: UInt32, quantity: UInt64): @Collection {
            let newCollection <- create Collection()

            var i: UInt64 = 0
            while i < quantity {
                newCollection.deposit(token: <-self.mintMoment(playID: playID))
                i = i + UInt64(1)
            }

            return <-newCollection
        }
    }

    pub struct MomentData {

        // the ID of the Set that the Moment comes from
        pub let setID: UInt32

        // the ID of the Play that the moment references
        pub let playID: UInt32

        // the place in the play that this moment was minted
        // Otherwise know as the serial number
        pub let serialNumber: UInt32

        init(setID: UInt32, playID: UInt32, serialNumber: UInt32) {
            self.setID = setID
            self.playID = playID
            self.serialNumber = serialNumber
        }

    }

    // The resource that represents the Moment NFTs
    //
    pub resource NFT: NonFungibleToken.INFT {

        // global unique moment ID
        pub let id: UInt64

        // struct of moment metadata
        pub let data: MomentData

        init(serialNumber: UInt32, playID: UInt32, setID: UInt32) {
            // Increment the global moment IDs
            TopShot.totalSupply = TopShot.totalSupply + UInt64(1)

            self.id = TopShot.totalSupply

            // set the metadata struct
            self.data = MomentData(setID: setID, playID: playID, serialNumber: serialNumber)

            emit MomentMinted(momentID: self.id, playID: playID, setID: self.data.setID, serialNumber: self.data.serialNumber)
        }

        destroy() {
            emit MomentDestroyed(id: self.id)
        }
    }

    // Admin is a special authorization resource that
    // allows the owner to perform important functions to modify the
    // various aspects of the plays, sets, and moments
    //
    pub resource Admin {

        // createPlay creates a new Play struct
        // and stores it in the plays dictionary in the TopShot smart contract
        //
        // Parameters: metadata: A dictionary mapping metadata titles to their data
        //                       example: {"Player Name": "Kevin Durant", "Height": "7 feet"}
        //                               (because we all know Kevin Durant is not 6'9")
        //
        // Returns: the ID of the new Play object
        pub fun createPlay(metadata: {String: String}): UInt32 {
            // Create the new Play
            var newPlay = Play(metadata: metadata)
            let newID = newPlay.playID

            // Store it in the contract storage
            TopShot.playDatas[newID] = newPlay

            return newID
        }

        // createSet creates a new Set resource and returns it
        // so that the caller can store it in their account
        //
        // Parameters: name: The name of the set
        //             series: The series that the set belongs to
        //
        pub fun createSet(name: String) {
            // Create the new Set
            var newSet <- create Set(name: name)

            TopShot.sets[newSet.setID] <-! newSet
        }

        // borrowSet returns a reference to a set in the TopShot
        // contract so that the admin can call methods on it
        //
        // Parameters: setID: The ID of the set that you want to
        // get a reference to
        //
        // Returns: A reference to the set with all of the fields
        // and methods exposed
        //
        pub fun borrowSet(setID: UInt32): &Set {
            pre {
                TopShot.sets[setID] != nil: "Cannot borrow Set: The Set doesn't exist"
            }
            return &TopShot.sets[setID] as &Set
        }

        // startNewSeries ends the current series by incrementing
        // the series number, meaning that moments will be using the
        // new series number from now on
        //
        // Returns: The new series number
        //
        pub fun startNewSeries(): UInt32 {
            // end the current series and start a new one
            // by incrementing the TopShot series number
            TopShot.currentSeries = TopShot.currentSeries + UInt32(1)

            emit NewSeriesStarted(newCurrentSeries: TopShot.currentSeries)

            return TopShot.currentSeries
        }

        // createNewAdmin creates a new Admin Resource
        //
        pub fun createNewAdmin(): @Admin {
            return <-create Admin()
        }
    }

    // This is the interface that users can cast their moment Collection as
    // to allow others to deposit moments into their collection
    pub resource interface MomentCollectionPublic {
        pub fun deposit(token: @NonFungibleToken.NFT)
        pub fun batchDeposit(tokens: @NonFungibleToken.Collection)
        pub fun getIDs(): [UInt64]
        pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT
        pub fun borrowMoment(id: UInt64): &TopShot.NFT? {
            // If the result isn't nil, the id of the returned reference
            // should be the same as the argument to the function
            post {
                (result == nil) || (result?.id == id):
                    "Cannot borrow Moment reference: The ID of the returned reference is incorrect"
            }
        }
    }

    // Collection is a resource that every user who owns NFTs
    // will store in their account to manage their NFTS
    //
    pub resource Collection: MomentCollectionPublic, NonFungibleToken.Provider, NonFungibleToken.Receiver, NonFungibleToken.CollectionPublic {
        // Dictionary of Moment conforming tokens
        // NFT is a resource type with a UInt64 ID field
        pub var ownedNFTs: @{UInt64: NonFungibleToken.NFT}

        init() {
            self.ownedNFTs <- {}
        }

        // withdraw removes an Moment from the collection and moves it to the caller
        pub fun withdraw(withdrawID: UInt64): @NonFungibleToken.NFT {
            let token <- self.ownedNFTs.remove(key: withdrawID)
                ?? panic("Cannot withdraw: Moment does not exist in the collection")

            emit Withdraw(id: token.id, from: self.owner?.address)

            return <-token
        }

        // batchWithdraw withdraws multiple tokens and returns them as a Collection
        pub fun batchWithdraw(ids: [UInt64]): @NonFungibleToken.Collection {
            var batchCollection <- create Collection()

            // iterate through the ids and withdraw them from the collection
            for id in ids {
                batchCollection.deposit(token: <-self.withdraw(withdrawID: id))
            }
            return <-batchCollection
        }

        // deposit takes a Moment and adds it to the collections dictionary
        pub fun deposit(token: @NonFungibleToken.NFT) {
            let token <- token as! @TopShot.NFT

            let id = token.id
            // add the new token to the dictionary
            let oldToken <- self.ownedNFTs[id] <- token

            if self.owner?.address != nil {
                emit Deposit(id: id, to: self.owner?.address)
            }

            destroy oldToken
        }

        // batchDeposit takes a Collection object as an argument
        // and deposits each contained NFT into this collection
        pub fun batchDeposit(tokens: @NonFungibleToken.Collection) {
            let keys = tokens.getIDs()

            // iterate through the keys in the collection and deposit each one
            for key in keys {
                self.deposit(token: <-tokens.withdraw(withdrawID: key))
            }
            destroy tokens
        }

        // getIDs returns an array of the IDs that are in the collection
        pub fun getIDs(): [UInt64] {
            return self.ownedNFTs.keys
        }

        // borrowNFT Returns a borrowed reference to a Moment in the collection
        // so that the caller can read its ID
        //
        // Parameters: id: The ID of the NFT to get the reference for
        //
        // Returns: A reference to the NFT
        pub fun borrowNFT(id: UInt64): &NonFungibleToken.NFT {
            return &self.ownedNFTs[id] as &NonFungibleToken.NFT
        }

        // borrowMoment Returns a borrowed reference to a Moment in the collection
        // so that the caller can read data and call methods from it
        // They can use this to read its setID, playID, serialNumber,
        // or any of the setData or Play Data associated with it by
        // getting the setID or playID and reading those fields from
        // the smart contract
        //
        // Parameters: id: The ID of the NFT to get the reference for
        //
        // Returns: A reference to the NFT
        pub fun borrowMoment(id: UInt64): &TopShot.NFT? {
            if self.ownedNFTs[id] != nil {
                let ref = &self.ownedNFTs[id] as auth &NonFungibleToken.NFT
                return ref as! &TopShot.NFT
            } else {
                return nil
            }
        }

        // If a transaction destroys the Collection object,
        // All the NFTs contained within are also destroyed
        // Kind of like when Damien Lillard destroys the hopes and
        // dreams of the entire city of Houston
        //
        destroy() {
            destroy self.ownedNFTs
        }
    }

    // -----------------------------------------------------------------------
    // TopShot contract-level function definitions
    // -----------------------------------------------------------------------

    // createEmptyCollection creates a new, empty Collection object so that
    // a user can store it in their account storage.
    // Once they have a Collection in their storage, they are able to receive
    // Moments in transactions
    //
    pub fun createEmptyCollection(): @NonFungibleToken.Collection {
        return <-create TopShot.Collection()
    }

    // getAllPlays returns all the plays in topshot
    //
    // Returns: An array of all the plays that have been created
    pub fun getAllPlays(): [TopShot.Play] {
        return TopShot.playDatas.values
    }

    // getPlayMetaData returns all the metadata associated with a specific play
    //
    // Parameters: playID: The id of the play that is being searched
    //
    // Returns: The metadata as a String to String mapping optional
    pub fun getPlayMetaData(playID: UInt32): {String: String}? {
        return self.playDatas[playID]?.metadata
    }

    // getPlayMetaDataByField returns the metadata associated with a
    //                        specific field of the metadata
    //                        Ex: field: "Team" will return something
    //                        like "Memphis Grizzlies"
    //
    // Parameters: playID: The id of the play that is being searched
    //             field: The field to search for
    //
    // Returns: The metadata field as a String Optional
    pub fun getPlayMetaDataByField(playID: UInt32, field: String): String? {
        // Don't force a revert if the playID or field is invalid
        if let play = TopShot.playDatas[playID] {
            return play.metadata[field]
        } else {
            return nil
        }
    }

    // getSetName returns the name that the specified set
    //            is associated with.
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: The name of the set
    pub fun getSetName(setID: UInt32): String? {
        // Don't force a revert if the setID is invalid
        return TopShot.setDatas[setID]?.name
    }

    // getSetSeries returns the series that the specified set
    //              is associated with.
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: The series that the set belongs to
    pub fun getSetSeries(setID: UInt32): UInt32? {
        // Don't force a revert if the setID is invalid
        return TopShot.setDatas[setID]?.series
    }

    // getSetIDsByName returns the IDs that the specified set name
    //                 is associated with.
    //
    // Parameters: setName: The name of the set that is being searched
    //
    // Returns: An array of the IDs of the set if it exists, or nil if doesn't
    pub fun getSetIDsByName(setName: String): [UInt32]? {
        var setIDs: [UInt32] = []

        // iterate through all the setDatas and search for the name
        for setData in TopShot.setDatas.values {
            if setName == setData.name {
                // if the name is found, return the ID
                setIDs.append(setData.setID)
            }
        }

        // If the name isn't found, return nil
        // Don't force a revert if the setName is invalid
        if setIDs.length == 0 {
            return nil
        } else {
            return setIDs
        }
    }

    // getPlaysInSet returns the list of play IDs that are in the set
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: An array of play IDs
    pub fun getPlaysInSet(setID: UInt32): [UInt32]? {
        // Don't force a revert if the setID is invalid
        return TopShot.sets[setID]?.plays
    }

    // isEditionRetired returns a boolean that indicates if a set/play combo
    //                  (otherwise known as an edition) is retired.
    //                  If an edition is retired, it still remains in the set,
    //                  but moments can no longer be minted from it.
    //
    // Parameters: setID: The id of the set that is being searched
    //             playID: The id of the play that is being searched
    //
    // Returns: Boolean indicating if the edition is retired or not
    pub fun isEditionRetired(setID: UInt32, playID: UInt32): Bool? {
        // Don't force a revert if the set or play ID is invalid
        // remove the set from the dictionary to ket its field
        if let setToRead <- TopShot.sets.remove(key: setID) {

            let retired = setToRead.retired[playID]

            TopShot.sets[setID] <-! setToRead

            return retired
        } else {
            return nil
        }
    }

    // isSetLocked returns a boolean that indicates if a set
    //             is locked. If an set is locked,
    //             new plays can no longer be added to it,
    //             but moments can still be minted from plays
    //             that are currently in it.
    //
    // Parameters: setID: The id of the set that is being searched
    //
    // Returns: Boolean indicating if the set is locked or not
    pub fun isSetLocked(setID: UInt32): Bool? {
        // Don't force a revert if the setID is invalid
        return TopShot.sets[setID]?.locked
    }

    // getNumMomentsInEdition return the number of moments that have been
    //                        minted from a certain edition.
    //
    // Parameters: setID: The id of the set that is being searched
    //             playID: The id of the play that is being searched
    //
    // Returns: The total number of moments
    //          that have been minted from an edition
    pub fun getNumMomentsInEdition(setID: UInt32, playID: UInt32): UInt32? {
        // Don't force a revert if the set or play ID is invalid
        // remove the set from the dictionary to get its field
        if let setToRead <- TopShot.sets.remove(key: setID) {

            // read the numMintedPerPlay
            let amount = setToRead.numberMintedPerPlay[playID]

            // put the set back
            TopShot.sets[setID] <-! setToRead

            return amount
        } else {
            return nil
        }
    }

    // -----------------------------------------------------------------------
    // TopShot initialization function
    // -----------------------------------------------------------------------
    //
    init() {
        // initialize the fields
        self.currentSeries = 0
        self.playDatas = {}
        self.setDatas = {}
        self.sets <- {}
        self.nextPlayID = 1
        self.nextSetID = 1
        self.totalSupply = 0

        // Put a new Collection in storage
        self.account.save<@Collection>(<- create Collection(), to: /storage/MomentCollection)

        // create a public capability for the collection
        self.account.link<&{MomentCollectionPublic}>(/public/MomentCollection, target: /storage/MomentCollection)

        // Put the Minter in storage
        self.account.save<@Admin>(<- create Admin(), to: /storage/TopShotAdmin)

        emit ContractInitialized()
    }
}
`

	deployTx := utils.DeploymentTransaction("TopShot", []byte(topShotContract))

	topShotAddress, err := common.HexToAddress("0x0b2a3299cc857e29")
	require.NoError(t, err)

	var events []cadence.Event
	var loggedMessages []string

	var signerAddress common.Address

	var contractValueReads = 0

	onRead := func(owner, key, value []byte) {
		if bytes.Equal(key, []byte(formatContractKey("TopShot"))) {
			contractValueReads++
		}
	}

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(onRead, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location.ID()] = string(code)
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = []byte(accountCodes[location.ID()])
			return code, nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		decodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
			return json.Decode(b)
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy TopShot contract

	signerAddress = topShotAddress

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Mint moments

	contractValueReads = 0

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              import TopShot from 0x0b2a3299cc857e29

              transaction {

                  prepare(signer: AuthAccount) {
	                  let adminRef = signer.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)!

                      let playID = adminRef.createPlay(metadata: {"name": "Test"})
                      let setID = TopShot.nextSetID
                      adminRef.createSet(name: "Test")
                      let setRef = adminRef.borrowSet(setID: setID)
                      setRef.addPlay(playID: playID)

	                  let moments <- setRef.batchMintMoment(playID: playID, quantity: 2)

                      signer.borrow<&TopShot.Collection>(from: /storage/MomentCollection)!
                          .batchDeposit(tokens: <-moments)
                  }
              }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, contractValueReads)

	// Set up receiver

	const setupTx = `
	  import NonFungibleToken from 0x1d7e57aa55817448
	  import TopShot from 0x0b2a3299cc857e29

	  transaction {

	      prepare(signer: AuthAccount) {
	          signer.save(
                 <-TopShot.createEmptyCollection(),
                 to: /storage/MomentCollection
              )
              signer.link<&TopShot.Collection>(
                 /public/MomentCollection,
                 target: /storage/MomentCollection
              )
	      }
	  }
	`

	signerAddress = common.BytesToAddress([]byte{0x42})

	contractValueReads = 0

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(setupTx),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, contractValueReads)

	// Transfer

	signerAddress = topShotAddress

	const transferTx = `
	  import NonFungibleToken from 0x1d7e57aa55817448
	  import TopShot from 0x0b2a3299cc857e29

	  transaction(momentIDs: [UInt64]) {
	      let transferTokens: @NonFungibleToken.Collection

	      prepare(acct: AuthAccount) {
	          let ref = acct.borrow<&TopShot.Collection>(from: /storage/MomentCollection)!
	          self.transferTokens <- ref.batchWithdraw(ids: momentIDs)
	      }

	      execute {
	          // get the recipient's public account object
	          let recipient = getAccount(0x42)

	          // get the Collection reference for the receiver
	          let receiverRef = recipient.getCapability(/public/MomentCollection)
	              .borrow<&{TopShot.MomentCollectionPublic}>()!

	          // deposit the NFT in the receivers collection
	          receiverRef.batchDeposit(tokens: <-self.transferTokens)
	      }
	  }
	`

	encodedArg, err := json.Encode(
		cadence.NewArray([]cadence.Value{
			cadence.NewUInt64(1),
		}),
	)
	require.NoError(t, err)

	contractValueReads = 0

	err = runtime.ExecuteTransaction(
		Script{
			Source:    []byte(transferTx),
			Arguments: [][]byte{encodedArg},
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	require.Equal(t, 0, contractValueReads)
}

func TestRuntimeStorageUnlink(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	storage := newTestStorage(nil, nil)

	signer := common.BytesToAddress([]byte{0x42})

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Store a value and link a capability

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction {
                  prepare(signer: AuthAccount) {
                      signer.save(42, to: /storage/test)

                      signer.link<&Int>(
                          /public/test,
                          target: /storage/test
                      )

                      assert(signer.getCapability<&Int>(/public/test).borrow() != nil)
                  }
              }
			`),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Unlink the capability

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
            transaction {
                prepare(signer: AuthAccount) {
                    signer.unlink(/public/test)

                    assert(signer.getCapability<&Int>(/public/test).borrow() == nil)
                }
            }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Get the capability after unlink

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction {
                  prepare(signer: AuthAccount) {
                      assert(signer.getCapability<&Int>(/public/test).borrow() == nil)
                  }
              }
            `),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeStorageSaveCapability(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	storage := newTestStorage(nil, nil)

	signer := common.BytesToAddress([]byte{0x42})

	runtimeInterface := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Store a capability

	for _, domain := range []common.PathDomain{
		common.PathDomainPrivate,
		common.PathDomainPublic,
	} {

		for typeDescription, ty := range map[string]cadence.Type{
			"Untyped": nil,
			"Typed":   cadence.ReferenceType{Authorized: false, Type: cadence.IntType{}},
		} {

			t.Run(fmt.Sprintf("%s %s", domain.Identifier(), typeDescription), func(t *testing.T) {

				storagePath := cadence.Path{
					Domain: "storage",
					Identifier: fmt.Sprintf(
						"test%s%s",
						typeDescription,
						domain.Identifier(),
					),
				}

				context := Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				}

				var typeArgument string
				if ty != nil {
					typeArgument = fmt.Sprintf("<%s>", ty.ID())
				}

				err := runtime.ExecuteTransaction(
					Script{
						Source: []byte(fmt.Sprintf(
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let cap = signer.getCapability%s(/%s/test)
                                      signer.save(cap, to: %s)
                                  }
                              }
			                `,
							typeArgument,
							domain.Identifier(),
							storagePath,
						)),
					},
					context,
				)
				require.NoError(t, err)

				value, err := runtime.ReadStored(signer, storagePath, context)
				require.NoError(t, err)

				require.Equal(t,
					cadence.Optional{
						Value: cadence.Capability{
							Path: cadence.Path{
								Domain:     domain.Identifier(),
								Identifier: "test",
							},
							Address:    cadence.Address(signer),
							BorrowType: ty,
						},
					},
					value,
				)
			})
		}
	}
}

func TestRuntimeStorageReferenceCast(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	signerAddress := common.BytesToAddress([]byte{0x42})

	deployTx := utils.DeploymentTransaction("Test", []byte(`
      pub contract Test {

          pub resource interface RI {}

          pub resource R: RI {}

          pub fun createR(): @R {
              return <-create R()
          }
      }
    `))

	accountCodes := map[common.LocationID][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
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

	// Deploy contract

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

	// Run test transaction

	const testTx = `
      import Test from 0x42

      transaction {
          prepare(signer: AuthAccount) {
              signer.save(<-Test.createR(), to: /storage/r)

              signer.link<&Test.R{Test.RI}>(
                 /public/r,
                 target: /storage/r
              )

              let ref = signer.getCapability<&Test.R{Test.RI}>(/public/r).borrow()!

              let casted = (ref as AnyStruct) as! &Test.R
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

	require.Contains(t, err.Error(), "unexpectedly found non-`&Test.R` while force-casting value")
}

func TestRuntimeStorageNonStorable(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := common.BytesToAddress([]byte{0x1})

	for name, code := range map[string]string{
		"ephemeral reference": `
            let value = &1 as &Int
        `,
		"storage reference": `
            signer.save("test", to: /storage/string)
            let value = signer.borrow<&String>(from: /storage/string)!
        `,
		"function": `
            let value = fun () {}
        `,
	} {

		t.Run(name, func(t *testing.T) {

			tx := []byte(
				fmt.Sprintf(
					`
	                  transaction {
	                      prepare(signer: AuthAccount) {
                              %s
	                          signer.save((value as AnyStruct), to: /storage/value)
	                      }
	                   }
	                `,
					code,
				),
			)

			runtimeInterface := &testRuntimeInterface{
				storage: newTestStorage(nil, nil),
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
			require.Error(t, err)

			require.Contains(t, err.Error(), "cannot write non-storable value")
		})
	}
}

func TestRuntimeStorageRecursiveReference(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime()

	address := common.BytesToAddress([]byte{0x1})

	const code = `
      transaction {
	      prepare(signer: AuthAccount) {
              let refs: [AnyStruct] = []
              refs.insert(at: 0, &refs as &AnyStruct)
              signer.save(refs, to: /storage/refs)
	      }
	  }
    `

	runtimeInterface := &testRuntimeInterface{
		storage: newTestStorage(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(code),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.Error(t, err)

	require.Contains(t, err.Error(), "cannot write non-storable value")
}
