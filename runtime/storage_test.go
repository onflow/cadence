/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func withWritesToStorage(
	tb testing.TB,
	count int,
	random *rand.Rand,
	onWrite func(owner, key, value []byte),
	handler func(*Storage, *interpreter.Interpreter),
) {
	ledger := newTestLedger(nil, onWrite)
	storage := NewStorage(ledger, nil)

	inter := newTestInterpreter(tb)

	address := common.MustBytesToAddress([]byte{0x1})

	for i := 0; i < count; i++ {

		randomIndex := random.Uint32()

		storageKey := interpreter.StorageKey{
			Address: address,
			Key:     fmt.Sprintf("%d", randomIndex),
		}

		var storageIndex atree.StorageIndex
		binary.BigEndian.PutUint32(storageIndex[:], randomIndex)

		storage.writes[storageKey] = storageIndex
	}

	handler(storage, inter)
}

func TestRuntimeStorageWriteCached(t *testing.T) {

	t.Parallel()

	random := rand.New(rand.NewSource(42))

	var writes int

	onWrite := func(owner, key, value []byte) {
		writes++
	}

	const count = 100

	withWritesToStorage(
		t,
		count,
		random,
		onWrite,
		func(storage *Storage, inter *interpreter.Interpreter) {
			const commitContractUpdates = true
			err := storage.Commit(inter, commitContractUpdates)
			require.NoError(t, err)

			require.Equal(t, count, writes)
		},
	)
}

func TestRuntimeStorageWriteCachedIsDeterministic(t *testing.T) {

	t.Parallel()

	var previousWrites []testWrite

	// verify for 10 times and check the writes are always deterministic
	for i := 0; i < 10; i++ {

		var writes []testWrite

		onWrite := func(owner, key, _ []byte) {
			writes = append(writes, testWrite{
				owner: owner,
				key:   key,
			})
		}

		const count = 100
		withWritesToStorage(
			t,
			count,
			rand.New(rand.NewSource(42)),
			onWrite,
			func(storage *Storage, inter *interpreter.Interpreter) {
				const commitContractUpdates = true
				err := storage.Commit(inter, commitContractUpdates)
				require.NoError(t, err)
			},
		)

		if previousWrites != nil {
			// no additional items
			require.Len(t, writes, len(previousWrites))

			for i, previousWrite := range previousWrites {
				// compare the new write with the old write
				require.Equal(t, previousWrite, writes[i])
			}
		}

		previousWrites = writes
	}
}

func TestRuntimeStorageWrite(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	tx := []byte(`
      transaction {
          prepare(signer: AuthAccount) {
              signer.save(1, to: /storage/one)
          }
       }
    `)

	var writes []testWrite

	onWrite := func(owner, key, _ []byte) {
		writes = append(writes, testWrite{
			owner,
			key,
		})
	}

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, onWrite),
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
			// storage index to storage domain storage map
			{
				[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
				[]byte("storage"),
			},
			// storage domain storage map
			{
				[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
				[]byte{'$', 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			},
		},
		writes,
	)
}

func TestRuntimeAccountStorage(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	storage := newTestLedger(nil, nil)

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

	runtime := newTestInterpreterRuntime()

	addressString, err := hex.DecodeString("aad3e26e406987c2")
	require.NoError(t, err)

	signingAddress := common.MustBytesToAddress(addressString)

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

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signingAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = accountCodes[location]
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

	require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
}

func TestRuntimeStorageReadAndBorrow(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	storage := newTestLedger(nil, nil)

	signer := common.MustBytesToAddress([]byte{0x42})

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
		require.Equal(t, cadence.NewInt(42), value)
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
		require.Equal(t, nil, value)
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
		require.Equal(t, cadence.NewInt(42), value)
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
		require.Equal(t, nil, value)
	})
}

func TestRuntimeTopShotContractDeployment(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	testAddress, err := common.HexToAddress("0x0b2a3299cc857e29")
	require.NoError(t, err)

	nextTransactionLocation := newTransactionLocationGenerator()

	accountCodes := map[common.LocationID]string{
		"A.1d7e57aa55817448.NonFungibleToken": realNonFungibleTokenInterface,
	}

	events := make([]cadence.Event, 0)

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{testAddress}, nil
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
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	err = runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"TopShot",
				[]byte(realTopShotContract),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"TopShotShardedCollection",
				[]byte(realTopShotShardedCollectionContract),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"TopshotAdminReceiver",
				[]byte(realTopshotAdminReceiverContract),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeTopShotBatchTransfer(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	accountCodes := map[common.LocationID]string{
		"A.1d7e57aa55817448.NonFungibleToken": realNonFungibleTokenInterface,
	}

	deployTx := utils.DeploymentTransaction("TopShot", []byte(realTopShotContract))

	topShotAddress, err := common.HexToAddress("0x0b2a3299cc857e29")
	require.NoError(t, err)

	var events []cadence.Event
	var loggedMessages []string

	var signerAddress common.Address

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
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
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
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

	signerAddress = common.MustBytesToAddress([]byte{0x42})

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
}

func TestRuntimeBatchMintAndTransfer(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	const contract = `
      pub contract Test {

          pub resource interface INFT {}

          pub resource NFT: INFT {}

          pub resource Collection {

              pub var ownedNFTs: @{UInt64: NFT}

              init() {
                  self.ownedNFTs <- {}
              }

              pub fun withdraw(id: UInt64): @NFT {
                  let token <- self.ownedNFTs.remove(key: id)
                      ?? panic("Cannot withdraw: NFT does not exist in the collection")

                  return <-token
              }

              pub fun deposit(token: @NFT) {
                  let oldToken <- self.ownedNFTs[token.uuid] <- token
                  destroy oldToken
              }

              pub fun batchDeposit(collection: @Collection) {
                  let ids = collection.getIDs()

                  for id in ids {
                      self.deposit(token: <-collection.withdraw(id: id))
                  }

                  destroy collection
              }

              pub fun batchWithdraw(ids: [UInt64]): @Collection {
                  let collection <- create Collection()

                  for id in ids {
                      collection.deposit(token: <-self.withdraw(id: id))
                  }

                  return <-collection
              }

              pub fun getIDs(): [UInt64] {
                  return self.ownedNFTs.keys
              }

              destroy() {
                  destroy self.ownedNFTs
              }
          }

          init() {
              self.account.save(
                 <-Test.createEmptyCollection(),
                 to: /storage/MainCollection
              )
              self.account.link<&Collection>(
                 /public/MainCollection,
                 target: /storage/MainCollection
              )
          }

          pub fun mint(): @NFT {
              return <- create NFT()
          }

          pub fun createEmptyCollection(): @Collection {
              return <- create Collection()
          }

          pub fun batchMint(count: UInt64): @Collection {
              let collection <- create Collection()

              var i: UInt64 = 0
              while i < count {
                  collection.deposit(token: <-self.mint())
                  i = i + 1
              }
              return <-collection
          }
      }
    `

	deployTx := utils.DeploymentTransaction("Test", []byte(contract))

	contractAddress := common.MustBytesToAddress([]byte{0x1})

	var events []cadence.Event
	var loggedMessages []string

	var signerAddress common.Address

	accountCodes := map[Location]string{}

	var uuid uint64

	runtimeInterface := &testRuntimeInterface{
		generateUUID: func() (uint64, error) {
			uuid++
			return uuid, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = string(code)
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = []byte(accountCodes[location])
			return code, nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contract

	signerAddress = contractAddress

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

	// Mint moments

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              import Test from 0x1

              transaction {

                  prepare(signer: AuthAccount) {
                      let collection <- Test.batchMint(count: 1000)

                      log(collection.getIDs())

                      signer.borrow<&Test.Collection>(from: /storage/MainCollection)!
                          .batchDeposit(collection: <-collection)
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

	// Set up receiver

	const setupTx = `
      import Test from 0x1

      transaction {

          prepare(signer: AuthAccount) {
              signer.save(
                 <-Test.createEmptyCollection(),
                 to: /storage/TestCollection
              )
              signer.link<&Test.Collection>(
                 /public/TestCollection,
                 target: /storage/TestCollection
              )
          }
      }
    `

	signerAddress = common.MustBytesToAddress([]byte{0x2})

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

	// Transfer

	signerAddress = contractAddress

	const transferTx = `
      import Test from 0x1

      transaction(ids: [UInt64]) {
          let collection: @Test.Collection

          prepare(signer: AuthAccount) {
              self.collection <- signer.borrow<&Test.Collection>(from: /storage/MainCollection)!
                  .batchWithdraw(ids: ids)
          }

          execute {
              getAccount(0x2)
                  .getCapability(/public/TestCollection)
                  .borrow<&Test.Collection>()!
                  .batchDeposit(collection: <-self.collection)
          }
      }
    `

	var values []cadence.Value

	const startID uint64 = 10
	const count = 20

	for id := startID; id <= startID+count; id++ {
		values = append(values, cadence.NewUInt64(id))
	}

	encodedArg, err := json.Encode(cadence.NewArray(values))
	require.NoError(t, err)

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
}

func TestRuntimeStorageUnlink(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	storage := newTestLedger(nil, nil)

	signer := common.MustBytesToAddress([]byte{0x42})

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

	runtime := newTestInterpreterRuntime()

	storage := newTestLedger(nil, nil)

	signer := common.MustBytesToAddress([]byte{0x42})

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
					cadence.Capability{
						Path: cadence.Path{
							Domain:     domain.Identifier(),
							Identifier: "test",
						},
						Address:    cadence.Address(signer),
						BorrowType: ty,
					},
					value,
				)
			})
		}
	}
}

func TestRuntimeStorageReferenceCast(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	signerAddress := common.MustBytesToAddress([]byte{0x42})

	deployTx := utils.DeploymentTransaction("Test", []byte(`
      pub contract Test {

          pub resource interface RI {}

          pub resource R: RI {}

          pub fun createR(): @R {
              return <-create R()
          }
      }
    `))

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = accountCodes[location]
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

	runtime := newTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

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
				storage: newTestLedger(nil, nil),
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

			require.Contains(t, err.Error(), "cannot store non-storable value")
		})
	}
}

func TestRuntimeStorageRecursiveReference(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

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
		storage: newTestLedger(nil, nil),
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

	require.Contains(t, err.Error(), "cannot store non-storable value")
}

func TestRuntimeStorageTransfer(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	ledger := newTestLedger(nil, nil)

	var signers []Address

	runtimeInterface := &testRuntimeInterface{
		storage: ledger,
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Store

	signers = []Address{address1}

	storeTx := []byte(`
      transaction {
          prepare(signer: AuthAccount) {
              signer.save([1], to: /storage/test)
          }
       }
    `)

	err := runtime.ExecuteTransaction(
		Script{
			Source: storeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Transfer

	signers = []Address{address1, address2}

	transferTx := []byte(`
      transaction {
          prepare(signer1: AuthAccount, signer2: AuthAccount) {
              let value = signer1.load<[Int]>(from: /storage/test)!
              signer2.save(value, to: /storage/test)
          }
       }
    `)

	err = runtime.ExecuteTransaction(
		Script{
			Source: transferTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	var nonEmptyKeys int
	for _, data := range ledger.storedValues {
		if len(data) > 0 {
			nonEmptyKeys++
		}
	}
	// 5:
	// - 2x storage index for storage domain storage map
	// - 2x storage domain storage map
	// - array (atree array)
	assert.Equal(t, 5, nonEmptyKeys)
}

func TestRuntimeResourceOwnerChange(t *testing.T) {

	t.Parallel()

	runtime := NewInterpreterRuntime(Config{
		ResourceOwnerChangeHandlerEnabled: true,
	})

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	ledger := newTestLedger(nil, nil)

	var signers []Address

	deployTx := utils.DeploymentTransaction("Test", []byte(`
      pub contract Test {

          pub resource R {}

          pub fun createR(): @R {
              return <-create R()
          }
      }
    `))

	type resourceOwnerChange struct {
		typeID     common.TypeID
		uuid       *interpreter.UInt64Value
		oldAddress common.Address
		newAddress common.Address
	}

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string
	var resourceOwnerChanges []resourceOwnerChange

	runtimeInterface := &testRuntimeInterface{
		storage: ledger,
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = accountCodes[location]
			return code, nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		resourceOwnerChanged: func(
			inter *interpreter.Interpreter,
			resource *interpreter.CompositeValue,
			oldAddress common.Address,
			newAddress common.Address,
		) {
			resourceOwnerChanges = append(
				resourceOwnerChanges,
				resourceOwnerChange{
					typeID: resource.TypeID(),
					// TODO: provide proper location range
					uuid:       resource.ResourceUUID(inter, interpreter.ReturnEmptyLocationRange),
					oldAddress: oldAddress,
					newAddress: newAddress,
				},
			)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contract

	signers = []Address{address1}

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

	// Store

	signers = []Address{address1}

	storeTx := []byte(`
      import Test from 0x1

      transaction {
          prepare(signer: AuthAccount) {
              signer.save(<-Test.createR(), to: /storage/test)
          }
      }
    `)

	err = runtime.ExecuteTransaction(
		Script{
			Source: storeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Transfer

	signers = []Address{address1, address2}

	transferTx := []byte(`
      import Test from 0x1

      transaction {
          prepare(signer1: AuthAccount, signer2: AuthAccount) {
              let value <- signer1.load<@Test.R>(from: /storage/test)!
              signer2.save(<-value, to: /storage/test)
          }
      }
    `)

	err = runtime.ExecuteTransaction(
		Script{
			Source: transferTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	var nonEmptyKeys []string
	for key, data := range ledger.storedValues {
		if len(data) > 0 {
			nonEmptyKeys = append(nonEmptyKeys, key)
		}
	}

	sort.Strings(nonEmptyKeys)

	assert.Equal(t,
		[]string{
			// account 0x1:
			//     storage map (domain key + map slab)
			//   + contract map (domain key + map slap)
			//   + contract
			"\x00\x00\x00\x00\x00\x00\x00\x01|$\x00\x00\x00\x00\x00\x00\x00\x01",
			"\x00\x00\x00\x00\x00\x00\x00\x01|$\x00\x00\x00\x00\x00\x00\x00\x02",
			"\x00\x00\x00\x00\x00\x00\x00\x01|$\x00\x00\x00\x00\x00\x00\x00\x04",
			"\x00\x00\x00\x00\x00\x00\x00\x01|contract",
			"\x00\x00\x00\x00\x00\x00\x00\x01|storage",
			// account 0x2
			//     storage map (domain key + map slab)
			//   + resource
			"\x00\x00\x00\x00\x00\x00\x00\x02|$\x00\x00\x00\x00\x00\x00\x00\x01",
			"\x00\x00\x00\x00\x00\x00\x00\x02|$\x00\x00\x00\x00\x00\x00\x00\x02",
			"\x00\x00\x00\x00\x00\x00\x00\x02|storage",
		},
		nonEmptyKeys,
	)

	expectedUUID := interpreter.NewUnmeteredUInt64Value(0)
	assert.Equal(t,
		[]resourceOwnerChange{
			{
				typeID: "A.0000000000000001.Test.R",
				uuid:   &expectedUUID,
				oldAddress: common.Address{
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
				newAddress: common.Address{
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
			},
			{
				typeID: "A.0000000000000001.Test.R",
				uuid:   &expectedUUID,
				oldAddress: common.Address{
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
				},
				newAddress: common.Address{
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
			},
			{
				typeID: "A.0000000000000001.Test.R",
				uuid:   &expectedUUID,
				oldAddress: common.Address{
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
				},
				newAddress: common.Address{
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
				},
			},
		},
		resourceOwnerChanges,
	)
}

func TestRuntimeStorageUsed(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	ledger := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		storage: ledger,
		getStorageUsed: func(_ Address) (uint64, error) {
			return 1, nil
		},
	}

	// NOTE: do NOT change the contents of this script,
	// it matters how the array is constructed,
	// ESPECIALLY the value of the addresses and the number of elements!
	//
	// Querying storageUsed commits storage, and this test asserts
	// that this should not clear temporary slabs

	script := []byte(`
       pub fun main() {
            var addresses: [Address]= [
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731,
                0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731, 0x2a3c4c2581cef731
            ]
            var count = 0
            for address in addresses {
                let account = getAccount(address)
                var x = account.storageUsed
            }
        }
    `)

	_, err := runtime.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)
	require.NoError(t, err)

}

func TestSortContractUpdates(t *testing.T) {

	t.Parallel()

	updates := []ContractUpdate{
		{
			Key: interpreter.StorageKey{
				Address: common.Address{2},
				Key:     "a",
			},
		},
		{
			Key: interpreter.StorageKey{
				Address: common.Address{1},
				Key:     "b",
			},
		},
		{
			Key: interpreter.StorageKey{
				Address: common.Address{1},
				Key:     "a",
			},
		},
		{
			Key: interpreter.StorageKey{
				Address: common.Address{0},
				Key:     "x",
			},
		},
	}

	SortContractUpdates(updates)

	require.Equal(t,
		[]ContractUpdate{
			{
				Key: interpreter.StorageKey{
					Address: common.Address{0},
					Key:     "x",
				},
			},
			{
				Key: interpreter.StorageKey{
					Address: common.Address{1},
					Key:     "a",
				},
			},
			{
				Key: interpreter.StorageKey{
					Address: common.Address{1},
					Key:     "b",
				},
			},
			{
				Key: interpreter.StorageKey{
					Address: common.Address{2},
					Key:     "a",
				},
			},
		},
		updates,
	)
}

func TestRuntimeMissingSlab1173(t *testing.T) {

	t.Parallel()

	const contract = `
pub contract Test {
    pub enum Role: UInt8 {
        pub case aaa
        pub case bbb
    }

    pub resource AAA {
        pub fun callA(): String {
            return "AAA"
        }
    }

    pub resource BBB {
        pub fun callB(): String {
            return "BBB"
        }
    }

    pub resource interface Receiver {
        pub fun receive(as: Role, capability: Capability)
    }

    pub resource Holder: Receiver {
        access(self) let roles: { Role: Capability }
        pub fun receive(as: Role, capability: Capability) {
            self.roles[as] = capability
        }

        pub fun borrowA(): &AAA {
            let role = self.roles[Role.aaa]!
            return role.borrow<&AAA>()!
        }

        pub fun borrowB(): &BBB {
            let role = self.roles[Role.bbb]!
            return role.borrow<&BBB>()!
        }

        access(contract) init() {
            self.roles = {}
        }
    }

    access(self) let capabilities: { Role: Capability }

    pub fun createHolder(): @Holder {
        return <- create Holder()
    }

    pub fun attach(as: Role, receiver: &AnyResource{Receiver}) {
        // TODO: Now verify that the owner is valid.

        let capability = self.capabilities[as]!
        receiver.receive(as: as, capability: capability)
    }

    init() {
        self.account.save<@AAA>(<- create AAA(), to: /storage/TestAAA)
        self.account.save<@BBB>(<- create BBB(), to: /storage/TestBBB)

        self.capabilities = {}
        self.capabilities[Role.aaa] = self.account.link<&AAA>(/private/TestAAA, target: /storage/TestAAA)!
        self.capabilities[Role.bbb] = self.account.link<&BBB>(/private/TestBBB, target: /storage/TestBBB)!
    }
}

`

	const tx = `
import Test from 0x1

transaction {
    prepare(acct: AuthAccount) {}
    execute {
        let holder <- Test.createHolder()
        Test.attach(as: Test.Role.aaa, receiver: &holder as &AnyResource{Test.Receiver})
        destroy holder
    }
}
`

	runtime := newTestInterpreterRuntime()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}

	var events []cadence.Event

	signerAccount := testAddress

	runtimeInterface := &testRuntimeInterface{
		getCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAccount}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			return accountCodes[location], nil
		},
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: utils.DeploymentTransaction(
				"Test",
				[]byte(contract),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Run transaction

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(tx),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeReferenceOwnerAccess(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		const contract = `
          pub contract TestContract {
              pub resource TestResource {}

              pub fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(accountA: AuthAccount, accountB: AuthAccount) {

                  let testResource <- TestContract.makeTestResource()
                  let ref = &testResource as &TestContract.TestResource

                  // At this point the resource is not in storage
                  log(ref.owner?.address)

                  accountA.save(<-testResource, to: /storage/test)

                  // At this point the resource is in storage A
                  log(ref.owner?.address)

                  let testResource2 <- accountA.load<@TestContract.TestResource>(from: /storage/test)!

                  let ref2 = &testResource2 as &TestContract.TestResource

                   // At this point the resource is not in storage
                  log(ref.owner?.address)
                  log(ref2.owner?.address)

                  accountB.save(<-testResource2, to: /storage/test)

                  // At this point the resource is in storage B
                  log(ref.owner?.address)
                  log(ref2.owner?.address)
              }
          }
        `

		runtime := newTestInterpreterRuntime()

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		var loggedMessages []string

		signers := []Address{
			common.MustBytesToAddress([]byte{0x1}),
		}

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return signers, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(address Address, name string, code []byte) error {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					"TestContract",
					[]byte(contract),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Run transaction

		signers = []Address{
			common.MustBytesToAddress([]byte{0x1}),
			common.MustBytesToAddress([]byte{0x2}),
		}

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(tx),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"nil",
				"0x0000000000000001",
				"nil",
				"nil",
				"0x0000000000000002",
				"0x0000000000000002",
			},
			loggedMessages,
		)
	})

	t.Run("resource (array element)", func(t *testing.T) {

		t.Parallel()

		const contract = `
          pub contract TestContract {
              pub resource TestResource {}

              pub fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: AuthAccount) {

                  let testResources <- [<-TestContract.makeTestResource()]
                  let ref = &testResources[0] as &TestContract.TestResource

                  // At this point the resource is not in storage
                  log(ref.owner?.address)

                  account.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  log(ref.owner?.address)
              }
          }
        `

		runtime := newTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(address Address, name string, code []byte) error {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					"TestContract",
					[]byte(contract),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Run transaction

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(tx),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"nil",
				"0x0000000000000001",
			},
			loggedMessages,
		)
	})

	t.Run("resource (nested field, array element)", func(t *testing.T) {

		t.Parallel()

		const contract = `
          pub contract TestContract {
              pub resource TestNestedResource {}

              pub resource TestNestingResource {
                  pub let nestedResources: @[TestNestedResource]

                  init () {
                      self.nestedResources <- [<- create TestNestedResource()]
                  }

                  destroy () {
                      destroy self.nestedResources
                  }
              }

              pub fun makeTestNestingResource(): @TestNestingResource {
                  return <- create TestNestingResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: AuthAccount) {

                  let nestingResource <- TestContract.makeTestNestingResource()
                  let nestingResourceRef = &nestingResource as &TestContract.TestNestingResource
                  let nestedElementResourceRef = &nestingResource.nestedResources[0] as &TestContract.TestNestedResource

                  // At this point the nesting and nested resources are not in storage
                  log(nestingResourceRef.owner?.address)
                  log(nestedElementResourceRef.owner?.address)

                  account.save(<-nestingResource, to: /storage/test)

                  // At this point the nesting and nested resources are both in storage
                  log(nestingResourceRef.owner?.address)
                  log(nestedElementResourceRef.owner?.address)
              }
          }
        `

		runtime := newTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(address Address, name string, code []byte) error {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					"TestContract",
					[]byte(contract),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Run transaction

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(tx),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"nil",
				"nil",
				"0x0000000000000001",
				"0x0000000000000001",
			},
			loggedMessages,
		)
	})

	t.Run("array", func(t *testing.T) {

		t.Parallel()

		const contract = `
          pub contract TestContract {
              pub resource TestResource {}

              pub fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: AuthAccount) {

                  let testResources <- [<-[<-TestContract.makeTestResource()]]
                  let ref = &testResources[0] as &[TestContract.TestResource]

                  // At this point the resource is not in storage
                  log(ref[0].owner?.address)

                  account.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  log(ref[0].owner?.address)
              }
          }
        `

		runtime := newTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(address Address, name string, code []byte) error {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					"TestContract",
					[]byte(contract),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Run transaction

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(tx),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"nil",
				"0x0000000000000001",
			},
			loggedMessages,
		)
	})

	t.Run("dictionary", func(t *testing.T) {

		t.Parallel()

		const contract = `
          pub contract TestContract {
              pub resource TestResource {}

              pub fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: AuthAccount) {

                  let testResources <- [<-{0: <-TestContract.makeTestResource()}]
                  let ref = &testResources[0] as &{Int: TestContract.TestResource}

                  // At this point the resource is not in storage
                  log(ref[0]?.owner?.address)

                  account.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  log(ref[0]?.owner?.address)
              }
          }
        `

		runtime := newTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &testRuntimeInterface{
			getCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signerAccount}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			getAccountContractCode: func(address Address, name string) (code []byte, err error) {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				return accountCodes[location], nil
			},
			updateAccountContractCode: func(address Address, name string, code []byte) error {
				location := common.AddressLocation{
					Address: address,
					Name:    name,
				}
				accountCodes[location] = code
				return nil
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			log: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: utils.DeploymentTransaction(
					"TestContract",
					[]byte(contract),
				),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Run transaction

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(tx),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		require.Equal(t,
			[]string{
				"nil",
				"0x0000000000000001",
			},
			loggedMessages,
		)
	})
}

func TestRuntimeNoAtreeSendOnClosedChannelDuringCommit(t *testing.T) {

	t.Parallel()

	assert.NotPanics(t, func() {

		for i := 0; i < 1000; i++ {

			runtime := newTestInterpreterRuntime()

			address := common.MustBytesToAddress([]byte{0x1})

			const code = `
              transaction {
                  prepare(signer: AuthAccount) {
                      let refs: [AnyStruct] = []
                      refs.append(&refs as &AnyStruct)
                      signer.save(refs, to: /storage/refs)
                  }
              }
            `

			runtimeInterface := &testRuntimeInterface{
				storage: newTestLedger(nil, nil),
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

			require.Contains(t, err.Error(), "cannot store non-storable value")
		}
	})
}

// TestRuntimeStorageEnumCase tests the writing an enum case to storage,
// reading it back from storage, as well as using it to index into a dictionary.
//
func TestRuntimeStorageEnumCase(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(address Address, name string, code []byte) error {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(address Address, name string) (code []byte, err error) {
			location := common.AddressLocation{
				Address: address,
				Name:    name,
			}
			code = accountCodes[location]
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
			Source: utils.DeploymentTransaction(
				"C",
				[]byte(`
                  pub contract C {

                    pub enum E: UInt8 {
                        pub case A
                        pub case B
                    }

                    pub resource R {
                        pub let id: UInt64
                        pub let e: E

                        init(id: UInt64, e: E) {
                            self.id = id
                            self.e = e
                        }
                    }

                    pub fun createR(id: UInt64, e: E): @R {
                        return <- create R(id: id, e: e)
                    }

                    pub resource Collection {
                        pub var rs: @{UInt64: R}

                        init () {
                            self.rs <- {}
                        }

                        pub fun withdraw(id: UInt64): @R {
                            return <- self.rs.remove(key: id)!
                        }

                        pub fun deposit(_ r: @R) {

                            let counts: {E: UInt64} = {}
                            log(r.e)
                            counts[r.e] = 42 // test indexing expression is transferred properly
                            log(r.e)

                            let oldR <- self.rs[r.id] <-! r
                            destroy oldR
                        }

                        destroy() {
                             destroy self.rs
                        }
                    }

                    pub fun createEmptyCollection(): @Collection {
                      return <- create Collection()
                    }
                  }
                `),
			),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Store enum case

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              import C from 0x1

              transaction {
                  prepare(signer: AuthAccount) {
                      signer.save(<-C.createEmptyCollection(), to: /storage/collection)
                      let collection = signer.borrow<&C.Collection>(from: /storage/collection)!
                      collection.deposit(<-C.createR(id: 0, e: C.E.B))
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

	// Load enum case

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              import C from 0x1

              transaction {
                  prepare(signer: AuthAccount) {
                      let collection = signer.borrow<&C.Collection>(from: /storage/collection)!
                      let r <- collection.withdraw(id: 0)
                      log(r.e)
                      destroy r
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

	require.Equal(t,
		[]string{
			"A.0000000000000001.C.E(rawValue: 1)",
			"A.0000000000000001.C.E(rawValue: 1)",
			"A.0000000000000001.C.E(rawValue: 1)",
		},
		loggedMessages,
	)
}

func TestStorageReadNoImplicitWrite(t *testing.T) {

	t.Parallel()

	rt := newTestInterpreterRuntime()

	address, err := common.HexToAddress("0x1")
	require.NoError(t, err)

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, func(_, _, _ []byte) {
			assert.FailNow(t, "unexpected write")
		}),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	err = rt.ExecuteTransaction(
		Script{
			Source: []byte((`
              transaction {
			    prepare(signer: AuthAccount) {
			        let ref = getAccount(0x2)
			            .getCapability(/public/test)
			            .borrow<&AnyStruct>()
                    assert(ref == nil)
			    }
              }
            `)),
		},
		Context{
			Interface: runtimeInterface,
			Location:  common.TransactionLocation{},
		},
	)
	require.NoError(t, err)
}

func TestRuntimeStorageInternalAccess(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

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

	ledger := newTestLedger(nil, nil)

	newRuntimeInterface := func() Interface {
		return &testRuntimeInterface{
			storage: ledger,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
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
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// Deploy contract

	runtimeInterface := newRuntimeInterface()

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

	// Store value

	runtimeInterface = newRuntimeInterface()

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              import Test from 0x1

              transaction {
                  prepare(signer: AuthAccount) {
                      signer.save("Hello, World!", to: /storage/first)
                      signer.save(["one", "two", "three"], to: /storage/second)
                      signer.save(<-Test.createR(), to: /storage/r)
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

	// Get storage map

	runtimeInterface = newRuntimeInterface()

	storage, inter, err := runtime.Storage(Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	storageMap := storage.GetStorageMap(address, common.PathDomainStorage.Identifier(), false)
	require.NotNil(t, storageMap)

	// Read first

	firstValue := storageMap.ReadValue(nil, "first")
	utils.RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("Hello, World!"),
		firstValue,
	)

	// Read second

	secondValue := storageMap.ReadValue(nil, "second")
	require.IsType(t, &interpreter.ArrayValue{}, secondValue)

	arrayValue := secondValue.(*interpreter.ArrayValue)

	element := arrayValue.Get(inter, interpreter.ReturnEmptyLocationRange, 2)
	utils.RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("three"),
		element,
	)

	// Read r

	rValue := storageMap.ReadValue(nil, "r")
	require.IsType(t, &interpreter.CompositeValue{}, rValue)

	_, err = ExportValue(rValue, inter, interpreter.ReturnEmptyLocationRange)
	require.NoError(t, err)
}
