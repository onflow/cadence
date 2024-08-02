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
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func withWritesToStorage(
	tb testing.TB,
	count int,
	random *rand.Rand,
	onWrite func(owner, key, value []byte),
	handler func(*Storage, *interpreter.Interpreter),
) {
	ledger := NewTestLedger(nil, onWrite)
	storage := NewStorage(ledger, nil)

	inter := NewTestInterpreter(tb)

	address := common.MustBytesToAddress([]byte{0x1})

	for i := 0; i < count; i++ {

		randomIndex := random.Uint32()

		storageKey := interpreter.StorageKey{
			Address: address,
			Key:     fmt.Sprintf("%d", randomIndex),
		}

		var slabIndex atree.SlabIndex
		binary.BigEndian.PutUint32(slabIndex[:], randomIndex)

		if storage.NewStorageMaps == nil {
			storage.NewStorageMaps = &orderedmap.OrderedMap[interpreter.StorageKey, atree.SlabIndex]{}
		}
		storage.NewStorageMaps.Set(storageKey, slabIndex)
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

	var previousWrites []ownerKeyPair

	// verify for 10 times and check the writes are always deterministic
	for i := 0; i < 10; i++ {

		var writes []ownerKeyPair

		onWrite := func(owner, key, _ []byte) {
			writes = append(writes, ownerKeyPair{
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

	runtime := NewTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	tx := []byte(`
      transaction {
          prepare(signer: auth(Storage) &Account) {
              signer.storage.save(1, to: /storage/one)
          }
       }
    `)

	var writes []ownerKeyPair

	onWrite := func(owner, key, _ []byte) {
		writes = append(writes, ownerKeyPair{
			owner,
			key,
		})
	}

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, onWrite),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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
		[]ownerKeyPair{
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

	runtime := NewTestInterpreterRuntime()

	script := []byte(`
      transaction {
        prepare(signer: auth(Storage) &Account) {
           let before = signer.storage.used
           signer.storage.save(42, to: /storage/answer)
           let after = signer.storage.used
           log(after != before)
        }
      }
    `)

	var loggedMessages []string

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{{42}}, nil
		},
		OnGetStorageUsed: func(_ Address) (uint64, error) {
			var amount uint64 = 0

			for _, data := range storage.StoredValues {
				amount += uint64(len(data))
			}

			return amount, nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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

	runtime := NewTestInterpreterRuntime()

	addressString, err := hex.DecodeString("aad3e26e406987c2")
	require.NoError(t, err)

	signingAddress := common.MustBytesToAddress(addressString)

	const testContract = `
      access(all) contract TestContract{
        access(all) struct fake{
          access(all) var balance: UFix64

          init(){
            self.balance = 0.0
          }

          access(all) fun setBalance(_ balance: UFix64) {
            self.balance = balance
          }
        }
        access(all) resource resourceConverter{
          access(all) fun convert(b: fake): AnyStruct {
            b.setBalance(100.0)
            return b
          }
        }
        access(all) resource resourceConverter2{
          access(all) fun convert(b: @AnyResource): AnyStruct {
            destroy b
            return ""
          }
        }
        access(all) fun createConverter():  @resourceConverter{
            return <- create resourceConverter();
        }
      }
    `

	deployTestContractTx := DeploymentTransaction("TestContract", []byte(testContract))

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signingAddress}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract

	err = runtime.ExecuteTransaction(
		Script{
			Source: deployTestContractTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Run test transaction

	const testTx = `
      import TestContract from 0xaad3e26e406987c2

      transaction {
        prepare(signer: auth(Storage, Capabilities) &Account) {

          let rc <- TestContract.createConverter()
          signer.storage.save(<-rc, to: /storage/rc)

          let cap = signer.capabilities.storage.issue<&TestContract.resourceConverter2>(/storage/rc)
          signer.capabilities.publish(cap, at: /public/rc)

          let ref = getAccount(0xaad3e26e406987c2)
              .capabilities
              .borrow<&TestContract.resourceConverter2>(/public/rc)
          assert(ref == nil)
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

	require.NoError(t, err)
}

func TestRuntimeStorageReadAndBorrow(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	storage := NewTestLedger(nil, nil)

	signer := common.MustBytesToAddress([]byte{0x42})

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Store a value and link a capability

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction {
                 prepare(signer: auth(Storage, Capabilities) &Account) {
                     signer.storage.save(42, to: /storage/test)
                     let cap = signer.capabilities.storage.issue<&Int>(/storage/test)
                     signer.capabilities.publish(cap, at: /public/test)
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

	t.Run("read stored, storage, existing", func(t *testing.T) {

		value, err := runtime.ReadStored(
			signer,
			cadence.Path{
				Domain:     common.PathDomainStorage,
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

	t.Run("read stored, storage, non-existing", func(t *testing.T) {

		value, err := runtime.ReadStored(
			signer,
			cadence.Path{
				Domain:     common.PathDomainStorage,
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

	t.Run("read stored, public, existing", func(t *testing.T) {

		value, err := runtime.ReadStored(
			signer,
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "test",
			},
			Context{
				Location:  TestLocation,
				Interface: runtimeInterface,
			},
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewCapability(
				1,
				cadence.Address(signer),
				cadence.NewReferenceType(
					cadence.Unauthorized{},
					cadence.IntType,
				),
			),
			value,
		)
	})

	t.Run("read stored, public, non-existing", func(t *testing.T) {

		value, err := runtime.ReadStored(
			signer,
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "other",
			},
			Context{
				Location:  TestLocation,
				Interface: runtimeInterface,
			},
		)
		require.NoError(t, err)
		require.Equal(t, nil, value)
	})
}

func TestRuntimeTopShotContractDeployment(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	testAddress, err := common.HexToAddress("0x0b2a3299cc857e29")
	require.NoError(t, err)

	nextTransactionLocation := NewTransactionLocationGenerator()

	nftAddress, err := common.HexToAddress("0x1d7e57aa55817448")
	require.NoError(t, err)

	accountCodes := map[common.Location]string{
		common.AddressLocation{
			Address: nftAddress,
			Name:    "NonFungibleToken",
		}: modifiedNonFungibleTokenInterface,
	}

	events := make([]cadence.Event, 0)

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{testAddress}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = []byte(accountCodes[location])
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	err = runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
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
			Source: DeploymentTransaction(
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
			Source: DeploymentTransaction(
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

	runtime := NewTestInterpreterRuntime()

	nftAddress, err := common.HexToAddress("0x1d7e57aa55817448")
	require.NoError(t, err)

	accountCodes := map[common.Location]string{
		common.AddressLocation{
			Address: nftAddress,
			Name:    "NonFungibleToken",
		}: modifiedNonFungibleTokenInterface,
	}

	deployTx := DeploymentTransaction("TopShot", []byte(realTopShotContract))

	topShotAddress, err := common.HexToAddress("0x0b2a3299cc857e29")
	require.NoError(t, err)

	var events []cadence.Event
	var loggedMessages []string

	var signerAddress common.Address

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = []byte(accountCodes[location])
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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

                  prepare(signer: auth(Storage) &Account) {
                      let adminRef = signer.storage.borrow<&TopShot.Admin>(from: /storage/TopShotAdmin)!

                      let playID = adminRef.createPlay(metadata: {"name": "Test"})
                      let setID = TopShot.nextSetID
                      adminRef.createSet(name: "Test")
                      let setRef = adminRef.borrowSet(setID: setID)
                      setRef.addPlay(playID: playID)

                      let moments <- setRef.batchMintMoment(playID: playID, quantity: 2)

                      signer.storage.borrow<&TopShot.Collection>(from: /storage/MomentCollection)!
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

          prepare(signer: auth(Storage, Capabilities) &Account) {
              signer.storage.save(
                 <-TopShot.createEmptyCollection(),
                 to: /storage/MomentCollection
              )
              let cap = signer.capabilities.storage.issue<&TopShot.Collection>(/storage/MomentCollection)
              signer.capabilities.publish(cap, at: /public/MomentCollection)
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
          let transferTokens: @{NonFungibleToken.Collection}

          prepare(signer: auth(Storage) &Account) {
              let ref = signer.storage.borrow<&TopShot.Collection>(from: /storage/MomentCollection)!
              self.transferTokens <- ref.batchWithdraw(ids: momentIDs)
          }

          execute {
              // get the recipient's public account object
              let recipient = getAccount(0x42)

              // get the Collection reference for the receiver
              let receiverRef = recipient.capabilities
                  .borrow<&{TopShot.MomentCollectionPublic}>(/public/MomentCollection)!

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

	runtime := NewTestInterpreterRuntime()

	const contract = `
      access(all) contract Test {

          access(all) resource interface INFT {}

          access(all) resource NFT: INFT {}

          access(all) resource Collection {

              access(all) var ownedNFTs: @{UInt64: NFT}

              init() {
                  self.ownedNFTs <- {}
              }

              access(all) fun withdraw(id: UInt64): @NFT {
                  let token <- self.ownedNFTs.remove(key: id)
                      ?? panic("Cannot withdraw: NFT does not exist in the collection")

                  return <-token
              }

              access(all) fun deposit(token: @NFT) {
                  let oldToken <- self.ownedNFTs[token.uuid] <- token
                  destroy oldToken
              }

              access(all) fun batchDeposit(collection: @Collection) {
                  let ids = collection.getIDs()

                  for id in ids {
                      self.deposit(token: <-collection.withdraw(id: id))
                  }

                  destroy collection
              }

              access(all) fun batchWithdraw(ids: [UInt64]): @Collection {
                  let collection <- create Collection()

                  for id in ids {
                      collection.deposit(token: <-self.withdraw(id: id))
                  }

                  return <-collection
              }

              access(all) fun getIDs(): [UInt64] {
                  return self.ownedNFTs.keys
              }
          }

          init() {
              self.account.storage.save(
                 <-Test.createEmptyCollection(),
                 to: /storage/MainCollection
              )
          }

          access(all) fun mint(): @NFT {
              return <- create NFT()
          }

          access(all) fun createEmptyCollection(): @Collection {
              return <- create Collection()
          }

          access(all) fun batchMint(count: UInt64): @Collection {
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

	deployTx := DeploymentTransaction("Test", []byte(contract))

	contractAddress := common.MustBytesToAddress([]byte{0x1})

	var events []cadence.Event
	var loggedMessages []string

	var signerAddress common.Address

	accountCodes := map[Location]string{}

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = []byte(accountCodes[location])
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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

                  prepare(signer: auth(Storage) &Account) {
                      let collection <- Test.batchMint(count: 1000)

                      log(collection.getIDs())

                      signer.storage.borrow<&Test.Collection>(from: /storage/MainCollection)!
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

          prepare(signer: auth(Storage, Capabilities) &Account) {
              signer.storage.save(
                 <-Test.createEmptyCollection(),
                 to: /storage/TestCollection
              )
              let cap = signer.capabilities.storage.issue<&Test.Collection>(/storage/TestCollection)
              signer.capabilities.publish(cap, at: /public/TestCollection)
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

          prepare(signer: auth(Storage) &Account) {
              self.collection <- signer.storage
                  .borrow<&Test.Collection>(from: /storage/MainCollection)!
                  .batchWithdraw(ids: ids)
          }

          execute {
              getAccount(0x2)
                  .capabilities
                  .borrow<&Test.Collection>(/public/TestCollection)!
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

func TestRuntimeStoragePublishAndUnpublish(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	storage := NewTestLedger(nil, nil)

	signer := common.MustBytesToAddress([]byte{0x42})

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Store a value and publish a capability

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction {
                  prepare(signer: auth(Storage, Capabilities) &Account) {
                      signer.storage.save(42, to: /storage/test)

                      let cap = signer.capabilities.storage.issue<&Int>(/storage/test)
                      signer.capabilities.publish(cap, at: /public/test)

                      assert(signer.capabilities.borrow<&Int>(/public/test) != nil)
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

	// Unpublish the capability

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
            transaction {
                prepare(signer: auth(Capabilities) &Account) {
                    signer.capabilities.unpublish(/public/test)

                    assert(signer.capabilities.borrow<&Int>(/public/test) == nil)
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

	// Get the capability after unpublish

	err = runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction {
                  prepare(signer: auth(Capabilities) &Account) {
                      assert(signer.capabilities.borrow<&Int>(/public/test) == nil)
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

	runtime := NewTestInterpreterRuntime()

	storage := NewTestLedger(nil, nil)

	signer := common.MustBytesToAddress([]byte{0x42})

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	ty := &cadence.ReferenceType{
		Authorization: cadence.UnauthorizedAccess,
		Type:          cadence.IntType,
	}

	var storagePathCounter int
	newStoragePath := func() cadence.Path {
		storagePathCounter++
		return cadence.Path{
			Domain: common.PathDomainStorage,
			Identifier: fmt.Sprintf(
				"test%d",
				storagePathCounter,
			),
		}
	}

	storagePath1 := newStoragePath()
	storagePath2 := newStoragePath()

	context := Context{
		Interface: runtimeInterface,
		Location:  nextTransactionLocation(),
	}

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: auth(Storage, Capabilities) &Account) {
                          let cap = signer.capabilities.storage.issue<%[1]s>(/storage/test)!
                          signer.capabilities.publish(cap, at: /public/test)
                          signer.storage.save(cap, to: %[2]s)

                          let cap2 = signer.capabilities.get<%[1]s>(/public/test)
                          signer.storage.save(cap2, to: %[3]s)
                      }
                  }
                `,
				ty.ID(),
				storagePath1,
				storagePath2,
			)),
		},
		context,
	)
	require.NoError(t, err)

	value, err := runtime.ReadStored(signer, storagePath1, context)
	require.NoError(t, err)

	expected := cadence.NewCapability(
		cadence.UInt64(1),
		cadence.Address(signer),
		ty,
	)

	require.Equal(t, expected, value)
}

func TestRuntimeStorageReferenceCast(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	signerAddress := common.MustBytesToAddress([]byte{0x42})

	deployTx := DeploymentTransaction("Test", []byte(`
      access(all) contract Test {

          access(all) resource interface RI {}

          access(all) resource R: RI {}

          access(all) fun createR(): @R {
              return <-create R()
          }
      }
    `))

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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
          prepare(signer: auth(Storage, Capabilities) &Account) {
              signer.storage.save(<-Test.createR(), to: /storage/r)

              let cap = signer.capabilities.storage
                  .issue<&Test.R>(/storage/r)
              signer.capabilities.publish(cap, at: /public/r)

              let ref = signer.capabilities.borrow<&Test.R>(/public/r)!

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

	require.NoError(t, err)
}

func TestRuntimeStorageReferenceDowncast(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	signerAddress := common.MustBytesToAddress([]byte{0x42})

	deployTx := DeploymentTransaction("Test", []byte(`
      access(all) contract Test {

          access(all) resource interface RI {}

          access(all) resource R: RI {}

          access(all) entitlement E

          access(all) fun createR(): @R {
              return <-create R()
          }
      }
    `))

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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
          prepare(signer: auth(Storage, Capabilities) &Account) {
              signer.storage.save(<-Test.createR(), to: /storage/r)

              let cap = signer.capabilities.storage.issue<&Test.R>(/storage/r)
              signer.capabilities.publish(cap, at: /public/r)

              let ref = signer.capabilities.borrow<&Test.R>(/public/r)!

              let casted = (ref as AnyStruct) as! auth(Test.E) &Test.R
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

	require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
}

func TestRuntimeStorageNonStorable(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	for name, code := range map[string]string{
		"ephemeral reference": `
            let value = &1 as &Int
        `,
		"storage reference": `
            signer.storage.save("test", to: /storage/string)
            let value = signer.storage.borrow<&String>(from: /storage/string)!
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
                          prepare(signer: auth(Storage) &Account) {
                              %s
                              signer.storage.save((value as AnyStruct), to: /storage/value)
                          }
                       }
                    `,
					code,
				),
			)

			runtimeInterface := &TestRuntimeInterface{
				Storage: NewTestLedger(nil, nil),
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
			}

			nextTransactionLocation := NewTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: tx,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			RequireError(t, err)

			require.Contains(t, err.Error(), "cannot store non-storable value")
		})
	}
}

func TestRuntimeStorageRecursiveReference(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	const code = `
      transaction {
          prepare(signer: auth(Storage) &Account) {
              let refs: [AnyStruct] = []
              refs.insert(at: 0, &refs as &AnyStruct)
              signer.storage.save(refs, to: /storage/refs)
          }
      }
    `

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(code),
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	RequireError(t, err)

	require.Contains(t, err.Error(), "cannot store non-storable value")
}

func TestRuntimeStorageTransfer(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	ledger := NewTestLedger(nil, nil)

	var signers []Address

	runtimeInterface := &TestRuntimeInterface{
		Storage: ledger,
		OnGetSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Store

	signers = []Address{address1}

	storeTx := []byte(`
      transaction {
          prepare(signer: auth(Storage) &Account) {
              signer.storage.save([1], to: /storage/test)
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
          prepare(
              signer1: auth(Storage) &Account,
              signer2: auth(Storage) &Account
          ) {
              let value = signer1.storage.load<[Int]>(from: /storage/test)!
              signer2.storage.save(value, to: /storage/test)
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
	for _, data := range ledger.StoredValues {
		if len(data) > 0 {
			nonEmptyKeys++
		}
	}

	// TODO: maybe retrieve and compare stored values from 2 accounts

	// 4:
	// NOTE: with atree inlining, array is inlined inside storage map
	// - 2x storage index for storage domain storage map
	// - 2x storage domain storage map
	assert.Equal(t, 4, nonEmptyKeys)
}

func TestRuntimeResourceOwnerChange(t *testing.T) {

	t.Parallel()

	config := DefaultTestInterpreterConfig
	config.ResourceOwnerChangeHandlerEnabled = true
	runtime := NewTestInterpreterRuntimeWithConfig(config)

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	ledger := NewTestLedger(nil, nil)

	var signers []Address

	deployTx := DeploymentTransaction("Test", []byte(`
      access(all) contract Test {

          access(all) resource R {}

          access(all) fun createR(): @R {
              return <-create R()
          }
      }
    `))

	type resourceOwnerChange struct {
		uuid       *interpreter.UInt64Value
		typeID     common.TypeID
		oldAddress common.Address
		newAddress common.Address
	}

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string
	var resourceOwnerChanges []resourceOwnerChange

	runtimeInterface := &TestRuntimeInterface{
		Storage: ledger,
		OnGetSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
		OnResourceOwnerChanged: func(
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
					uuid:       resource.ResourceUUID(inter, interpreter.EmptyLocationRange),
					oldAddress: oldAddress,
					newAddress: newAddress,
				},
			)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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
          prepare(signer: auth(Storage) &Account) {
              signer.storage.save(<-Test.createR(), to: /storage/test)
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
          prepare(
              signer1: auth(Storage) &Account,
              signer2: auth(Storage) &Account
          ) {
              let value <- signer1.storage.load<@Test.R>(from: /storage/test)!
              signer2.storage.save(<-value, to: /storage/test)
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
	for key, data := range ledger.StoredValues {
		if len(data) > 0 {
			nonEmptyKeys = append(nonEmptyKeys, key)
		}
	}

	sort.Strings(nonEmptyKeys)

	assert.Equal(t,
		[]string{
			// account 0x1:
			// NOTE: with atree inlining, contract is inlined in contract map
			//     storage map (domain key + map slab)
			//   + contract map (domain key + map slab)
			"\x00\x00\x00\x00\x00\x00\x00\x01|$\x00\x00\x00\x00\x00\x00\x00\x02",
			"\x00\x00\x00\x00\x00\x00\x00\x01|$\x00\x00\x00\x00\x00\x00\x00\x04",
			"\x00\x00\x00\x00\x00\x00\x00\x01|contract",
			"\x00\x00\x00\x00\x00\x00\x00\x01|storage",
			// account 0x2
			// NOTE: with atree inlining, resource is inlined in storage map
			//     storage map (domain key + map slab)
			"\x00\x00\x00\x00\x00\x00\x00\x02|$\x00\x00\x00\x00\x00\x00\x00\x02",
			"\x00\x00\x00\x00\x00\x00\x00\x02|storage",
		},
		nonEmptyKeys,
	)

	expectedUUID := interpreter.NewUnmeteredUInt64Value(1)
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

	runtime := NewTestInterpreterRuntime()

	ledger := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		Storage: ledger,
		OnGetStorageUsed: func(_ Address) (uint64, error) {
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
       access(all) fun main() {
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
                var x = account.storage.used
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

func TestRuntimeSortContractUpdates(t *testing.T) {

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
access(all) contract Test {
    access(all) enum Role: UInt8 {
        access(all) case aaa
        access(all) case bbb
    }

    access(all) resource AAA {
        access(all) fun callA(): String {
            return "AAA"
        }
    }

    access(all) resource BBB {
        access(all) fun callB(): String {
            return "BBB"
        }
    }

    access(all) resource interface Receiver {
        access(all) fun receive(asRole: Role, capability: Capability)
    }

    access(all) resource Holder: Receiver {
        access(self) let roles: { Role: Capability }
        access(all) fun receive(asRole: Role, capability: Capability) {
            self.roles[asRole] = capability
        }

        access(all) fun borrowA(): &AAA {
            let role = self.roles[Role.aaa]!
            return role.borrow<&AAA>()!
        }

        access(all) fun borrowB(): &BBB {
            let role = self.roles[Role.bbb]!
            return role.borrow<&BBB>()!
        }

        access(contract) init() {
            self.roles = {}
        }
    }

    access(self) let capabilities: { Role: Capability }

    access(all) fun createHolder(): @Holder {
        return <- create Holder()
    }

    access(all) fun attach(asRole: Role, receiver: &{Receiver}) {
        // TODO: Now verify that the owner is valid.

        let capability = self.capabilities[asRole]!
        receiver.receive(asRole: asRole, capability: capability)
    }

    init() {
        self.account.storage.save<@AAA>(<- create AAA(), to: /storage/TestAAA)
        self.account.storage.save<@BBB>(<- create BBB(), to: /storage/TestBBB)

        self.capabilities = {}
        self.capabilities[Role.aaa] = self.account.capabilities.storage.issue<&AAA>(/storage/TestAAA)!
        self.capabilities[Role.bbb] = self.account.capabilities.storage.issue<&BBB>(/storage/TestBBB)!
    }
}

`

	const tx = `
import Test from 0x1

transaction {
    prepare(signer: &Account) {}

    execute {
        let holder <- Test.createHolder()
        Test.attach(asRole: Test.Role.aaa, receiver: &holder as &{Test.Receiver})
        destroy holder
    }
}
`

	runtime := NewTestInterpreterRuntime()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}

	var events []cadence.Event

	signerAccount := testAddress

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(location Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		Storage: NewTestLedger(nil, nil),
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

	// Deploy contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
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
          access(all) contract TestContract {
              access(all) resource TestResource {}

              access(all) fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(
                  accountA: auth(Storage, Capabilities) &Account,
                  accountB: auth(Storage, Capabilities) &Account
              ) {
                  let testResource <- TestContract.makeTestResource()
                  let ref1 = &testResource as &TestContract.TestResource

                  // At this point the resource is not in storage
                  log(ref1.owner?.address)

                  accountA.storage.save(<-testResource, to: /storage/test)

                  // At this point the resource is in storage A
                  let cap = accountA.capabilities.storage.issue<&TestContract.TestResource>(/storage/test)
                  accountA.capabilities.publish(cap, at: /public/test)

                  let ref2 = accountA.capabilities.borrow<&TestContract.TestResource>(/public/test)!
                  log(ref2.owner?.address)

                  let testResource2 <- accountA.storage.load<@TestContract.TestResource>(from: /storage/test)!

                  let ref3 = &testResource2 as &TestContract.TestResource

                   // At this point the resource is not in storage
                  log(ref3.owner?.address)

                  accountB.storage.save(<-testResource2, to: /storage/test)

                  let cap2 = accountB.capabilities.storage.issue<&TestContract.TestResource>(/storage/test)
                  accountB.capabilities.publish(cap2, at: /public/test)

                  let ref4 = accountB.capabilities.borrow<&TestContract.TestResource>(/public/test)!

                  // At this point the resource is in storage B
                  log(ref4.owner?.address)
              }
          }
        `

		runtime := NewTestInterpreterRuntime()

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		var loggedMessages []string

		signers := []Address{
			common.MustBytesToAddress([]byte{0x1}),
		}

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return signers, nil
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
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction(
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
				"0x0000000000000002",
			},
			loggedMessages,
		)
	})

	t.Run("resource (array element)", func(t *testing.T) {

		t.Parallel()

		const contract = `
          access(all) contract TestContract {
              access(all) resource TestResource {}

              access(all) fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: auth(Storage, Capabilities) &Account) {

                  let testResources <- [<-TestContract.makeTestResource()]
                  let ref1 = &testResources[0] as &TestContract.TestResource

                  // At this point the resource is not in storage
                  log(ref1.owner?.address)

                  account.storage.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  let cap = account.capabilities.storage.issue<&[TestContract.TestResource]>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  let ref2 = account.capabilities.borrow<&[TestContract.TestResource]>(/public/test)!
                  let ref3 = ref2[0]
                  log(ref3.owner?.address)
              }
          }
        `

		runtime := NewTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
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
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction(
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
          access(all) contract TestContract {
              access(all) resource TestNestedResource {}

              access(all) resource TestNestingResource {
                  access(all) let nestedResources: @[TestNestedResource]

                  init () {
                      self.nestedResources <- [<- create TestNestedResource()]
                  }
              }

              access(all) fun makeTestNestingResource(): @TestNestingResource {
                  return <- create TestNestingResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: auth(Storage, Capabilities) &Account) {

                  let nestingResource <- TestContract.makeTestNestingResource()
                  var nestingResourceRef = &nestingResource as &TestContract.TestNestingResource
                  var nestedElementResourceRef = &nestingResource.nestedResources[0] as &TestContract.TestNestedResource

                  // At this point the nesting and nested resources are not in storage
                  log(nestingResourceRef.owner?.address)
                  log(nestedElementResourceRef.owner?.address)

                  account.storage.save(<-nestingResource, to: /storage/test)

                  // At this point the nesting and nested resources are both in storage
                  let cap = account.capabilities.storage.issue<&TestContract.TestNestingResource>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  nestingResourceRef = account.capabilities.borrow<&TestContract.TestNestingResource>(/public/test)!
                  nestedElementResourceRef = nestingResourceRef.nestedResources[0]

                  log(nestingResourceRef.owner?.address)
                  log(nestedElementResourceRef.owner?.address)
              }
          }
        `

		runtime := NewTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
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
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction(
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
          access(all) contract TestContract {
              access(all) resource TestResource {}

              access(all) fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: auth(Storage, Capabilities) &Account) {

                  let testResources <- [<-[<-TestContract.makeTestResource()]]
                  var ref = &testResources[0] as &[TestContract.TestResource]

                  // At this point the resource is not in storage
                  log(ref[0].owner?.address)

                  account.storage.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  let cap = account.capabilities.storage.issue<&[[TestContract.TestResource]]>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  let testResourcesRef = account.capabilities.borrow<&[[TestContract.TestResource]]>(/public/test)!
                  ref = testResourcesRef[0]
                  log(ref[0].owner?.address)
              }
          }
        `

		runtime := NewTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
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
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (cadence.Value, error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction(
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
          access(all) contract TestContract {
              access(all) resource TestResource {}

              access(all) fun makeTestResource(): @TestResource {
                  return <- create TestResource()
              }
          }
        `

		const tx = `
          import TestContract from 0x1

          transaction {

              prepare(account: auth(Storage, Capabilities) &Account) {

                  let testResources <- [<-{0: <-TestContract.makeTestResource()}]
                  var ref = &testResources[0] as &{Int: TestContract.TestResource}

                  // At this point the resource is not in storage
                  log(ref[0]?.owner?.address)

                  account.storage.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  let cap = account.capabilities.storage.issue<&[{Int: TestContract.TestResource}]>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  let testResourcesRef = account.capabilities.borrow<&[{Int: TestContract.TestResource}]>(/public/test)!

                  ref = testResourcesRef[0]
                  log(ref[0]?.owner?.address)
              }
          }
        `

		runtime := NewTestInterpreterRuntime()

		testAddress := common.MustBytesToAddress([]byte{0x1})

		accountCodes := map[Location][]byte{}

		var events []cadence.Event

		signerAccount := testAddress

		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			OnGetCode: func(location Location) (bytes []byte, err error) {
				return accountCodes[location], nil
			},
			Storage: NewTestLedger(nil, nil),
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
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: DeploymentTransaction(
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

			runtime := NewTestInterpreterRuntime()

			address := common.MustBytesToAddress([]byte{0x1})

			const code = `
              transaction {
                  prepare(signer: auth(Storage) &Account) {
                      let refs: [AnyStruct] = []
                      refs.append(&refs as &AnyStruct)
                      signer.storage.save(refs, to: /storage/refs)
                  }
              }
            `

			runtimeInterface := &TestRuntimeInterface{
				Storage: NewTestLedger(nil, nil),
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
			}

			nextTransactionLocation := NewTransactionLocationGenerator()

			err := runtime.ExecuteTransaction(
				Script{
					Source: []byte(code),
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			RequireError(t, err)

			require.Contains(t, err.Error(), "cannot store non-storable value")
		}
	})
}

// TestRuntimeStorageEnumCase tests the writing an enum case to storage,
// reading it back from storage, as well as using it to index into a dictionary.
func TestRuntimeStorageEnumCase(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	accountCodes := map[Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
		OnProgramLog: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy contract

	err := runtime.ExecuteTransaction(
		Script{
			Source: DeploymentTransaction(
				"C",
				[]byte(`
                  access(all) contract C {

                    access(all) enum E: UInt8 {
                        access(all) case A
                        access(all) case B
                    }

                    access(all) resource R {
                        access(all) let id: UInt64
                        access(all) let e: E

                        init(id: UInt64, e: E) {
                            self.id = id
                            self.e = e
                        }
                    }

                    access(all) fun createR(id: UInt64, e: E): @R {
                        return <- create R(id: id, e: e)
                    }

                    access(all) resource Collection {
                        access(all) var rs: @{UInt64: R}

                        init () {
                            self.rs <- {}
                        }

                        access(all) fun withdraw(id: UInt64): @R {
                            return <- self.rs.remove(key: id)!
                        }

                        access(all) fun deposit(_ r: @R) {

                            let counts: {E: UInt64} = {}
                            log(r.e)
                            counts[r.e] = 42 // test indexing expression is transferred properly
                            log(r.e)

                            let oldR <- self.rs[r.id] <-! r
                            destroy oldR
                        }
                    }

                    access(all) fun createEmptyCollection(): @Collection {
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
                  prepare(signer: auth(Storage) &Account) {
                      signer.storage.save(<-C.createEmptyCollection(), to: /storage/collection)
                      let collection = signer.storage.borrow<&C.Collection>(from: /storage/collection)!
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
                  prepare(signer: auth(Storage) &Account) {
                      let collection = signer.storage.borrow<&C.Collection>(from: /storage/collection)!
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

func TestRuntimeStorageReadNoImplicitWrite(t *testing.T) {

	t.Parallel()

	rt := NewTestInterpreterRuntime()

	address, err := common.HexToAddress("0x1")
	require.NoError(t, err)

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, func(_, _, _ []byte) {
			assert.FailNow(t, "unexpected write")
		}),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
	}

	err = rt.ExecuteTransaction(
		Script{
			Source: []byte((`
              transaction {
                prepare(signer: &Account) {
                    let ref = getAccount(0x2).capabilities.borrow<&AnyStruct>(/public/test)
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

	runtime := NewTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	deployTx := DeploymentTransaction("Test", []byte(`
     access(all) contract Test {

         access(all) resource interface RI {}

         access(all) resource R: RI {}

         access(all) fun createR(): @R {
             return <-create R()
         }
     }
   `))

	accountCodes := map[common.Location][]byte{}
	var events []cadence.Event
	var loggedMessages []string

	ledger := NewTestLedger(nil, nil)

	newRuntimeInterface := func() Interface {
		return &TestRuntimeInterface{
			Storage: ledger,
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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
                 prepare(signer: auth(Storage) &Account) {
                     signer.storage.save("Hello, World!", to: /storage/first)
                     signer.storage.save(["one", "two", "three"], to: /storage/second)
                     signer.storage.save(<-Test.createR(), to: /storage/r)
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

	firstValue := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("first"))
	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("Hello, World!"),
		firstValue,
	)

	// Read second

	secondValue := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("second"))
	require.IsType(t, &interpreter.ArrayValue{}, secondValue)

	arrayValue := secondValue.(*interpreter.ArrayValue)

	element := arrayValue.Get(inter, interpreter.EmptyLocationRange, 2)
	RequireValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("three"),
		element,
	)

	// Read r

	rValue := storageMap.ReadValue(nil, interpreter.StringStorageMapKey("r"))
	require.IsType(t, &interpreter.CompositeValue{}, rValue)

	_, err = ExportValue(rValue, inter, interpreter.EmptyLocationRange)
	require.NoError(t, err)
}

func TestRuntimeStorageIteration(t *testing.T) {

	t.Parallel()

	t.Run("non existing type", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := NewTestLedger(nil, nil)
		nextTransactionLocation := NewTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() (Interface, *[]Location) {

			var programStack []Location

			runtimeInterface := &TestRuntimeInterface{
				Storage: ledger,
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract no longer has the type
						return []byte(`access(all) contract Test {}`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					return nil
				},
			}

			return runtimeInterface, &programStack
		}

		// Deploy contract

		runtimeInterface, _ := newRuntimeInterface()

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

		runtimeInterface, _ = newRuntimeInterface()

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Test from 0x1

                    transaction {
                        prepare(signer: auth(Storage) &Account) {
                            signer.storage.save("Hello, World!", to: /storage/first)
                            signer.storage.save(["one", "two", "three"], to: /storage/second)
                            signer.storage.save(Test.Foo(), to: /storage/third)
                            signer.storage.save(1, to: /storage/fourth)
                            signer.storage.save(Test.Foo(), to: /storage/fifth)
                            signer.storage.save("two", to: /storage/sixth)
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

		// Make the `Test` contract broken. i.e: `Test.Foo` type is broken
		contractIsBroken = true

		var programStack *[]Location

		runtimeInterface, programStack = newRuntimeInterface()

		// Read value
		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    transaction {
                        prepare(account: auth(Storage) &Account) {
                            var total = 0
                            account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                                account.storage.borrow<&AnyStruct>(from: path)!
                                total = total + 1
                                return true
                            })

                            // Total values iterated should be 4.
                            // The two broken values must be skipped.
                            assert(total == 4)
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

		require.Empty(t, *programStack)
	})

	t.Run("broken contract, parsing problem", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := NewTestLedger(nil, nil)
		nextTransactionLocation := NewTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() Interface {
			return &TestRuntimeInterface{
				Storage: ledger,
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract has a syntax problem
						return []byte(`BROKEN`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					return nil
				},
			}

		}

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

		// Store values

		runtimeInterface = newRuntimeInterface()

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Test from 0x1

                    transaction {
                        prepare(signer: auth(Storage, Capabilities) &Account) {
                            signer.storage.save("Hello, World!", to: /storage/first)
                            signer.storage.save(["one", "two", "three"], to: /storage/second)
                            signer.storage.save(Test.Foo(), to: /storage/third)
                            signer.storage.save(1, to: /storage/fourth)
                            signer.storage.save(Test.Foo(), to: /storage/fifth)
                            signer.storage.save("two", to: /storage/sixth)

                            let capA = signer.capabilities.storage.issue<&String>(/storage/first)
                            signer.capabilities.publish(capA, at: /public/a)
                            let capB = signer.capabilities.storage.issue<&[String]>(/storage/second)
                            signer.capabilities.publish(capB, at: /public/b)
                            let capC = signer.capabilities.storage.issue<&Test.Foo>(/storage/third)
                            signer.capabilities.publish(capC, at: /public/c)
                            let capD = signer.capabilities.storage.issue<&Int>(/storage/fourth)
                            signer.capabilities.publish(capD, at: /public/d)
                            let capE = signer.capabilities.storage.issue<&Test.Foo>(/storage/fifth)
                            signer.capabilities.publish(capE, at: /public/e)
                            let capF = signer.capabilities.storage.issue<&String>(/storage/sixth)
                            signer.capabilities.publish(capF, at: /public/f)
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

		// Make the `Test` contract broken. i.e: `Test.Foo` type is broken
		contractIsBroken = true

		runtimeInterface = newRuntimeInterface()

		// Read value
		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    transaction {
                        prepare(account: auth(Storage) &Account) {
                            var total = 0
                            account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                                account.capabilities.borrow<&AnyStruct>(path)!
                                total = total + 1
                                return true
                            })

                            // Total values iterated should be 4.
                            // The two broken values must be skipped.
                            assert(total == 4)
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
	})

	t.Run("broken contract, type checking problem", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := NewTestLedger(nil, nil)
		nextTransactionLocation := NewTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() Interface {
			return &TestRuntimeInterface{
				Storage: ledger,
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract has a semantic error. i.e: cannot find `Bar`
						return []byte(`access(all) contract Test {
                            access(all) struct Foo: Bar {}
                        }`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					return nil
				},
			}
		}

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

		// Store values

		runtimeInterface = newRuntimeInterface()

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Test from 0x1
                    transaction {
                        prepare(signer: auth(Storage, Capabilities) &Account) {
                            signer.storage.save("Hello, World!", to: /storage/first)
                            signer.storage.save(["one", "two", "three"], to: /storage/second)
                            signer.storage.save(Test.Foo(), to: /storage/third)
                            signer.storage.save(1, to: /storage/fourth)
                            signer.storage.save(Test.Foo(), to: /storage/fifth)
                            signer.storage.save("two", to: /storage/sixth)

                            let capA = signer.capabilities.storage.issue<&String>(/storage/first)
                            signer.capabilities.publish(capA, at: /public/a)
                            let capB = signer.capabilities.storage.issue<&[String]>(/storage/second)
                            signer.capabilities.publish(capB, at: /public/b)
                            let capC = signer.capabilities.storage.issue<&Test.Foo>(/storage/third)
                            signer.capabilities.publish(capC, at: /public/c)
                            let capD = signer.capabilities.storage.issue<&Int>(/storage/fourth)
                            signer.capabilities.publish(capD, at: /public/d)
                            let capE = signer.capabilities.storage.issue<&Test.Foo>(/storage/fifth)
                            signer.capabilities.publish(capE, at: /public/e)
                            let capF = signer.capabilities.storage.issue<&String>(/storage/sixth)
                            signer.capabilities.publish(capF, at: /public/f)
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

		// Make the `Test` contract broken. i.e: `Test.Foo` type is broken
		contractIsBroken = true

		runtimeInterface = newRuntimeInterface()

		// Read value
		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    transaction {
                        prepare(account: &Account) {
                            var total = 0
                            account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                                account.capabilities.borrow<&AnyStruct>(path)!
                                total = total + 1
                                return true
                            })
                            // Total values iterated should be 4.
                            // The two broken values must be skipped.
                            assert(total == 4)
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
	})

	t.Run("type checking problem, wrapped error", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := NewTestLedger(nil, nil)
		nextTransactionLocation := NewTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() *TestRuntimeInterface {
			return &TestRuntimeInterface{
				Storage: ledger,
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract has a semantic error. i.e: cannot find `Bar`
						return []byte(`access(all) contract Test {
                            access(all) struct Foo: Bar {}
                        }`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					return nil
				},
			}
		}

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

		// Store values

		runtimeInterface = newRuntimeInterface()

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Test from 0x1
                    transaction {
                        prepare(signer: auth(Storage, Capabilities) &Account) {
                            signer.storage.save("Hello, World!", to: /storage/first)
                            signer.storage.save(["one", "two", "three"], to: /storage/second)
                            signer.storage.save(Test.Foo(), to: /storage/third)
                            signer.storage.save(1, to: /storage/fourth)
                            signer.storage.save(Test.Foo(), to: /storage/fifth)
                            signer.storage.save("two", to: /storage/sixth)

                            let capA = signer.capabilities.storage.issue<&String>(/storage/first)
                            signer.capabilities.publish(capA, at: /public/a)
                            let capB = signer.capabilities.storage.issue<&[String]>(/storage/second)
                            signer.capabilities.publish(capB, at: /public/b)
                            let capC = signer.capabilities.storage.issue<&Test.Foo>(/storage/third)
                            signer.capabilities.publish(capC, at: /public/c)
                            let capD = signer.capabilities.storage.issue<&Int>(/storage/fourth)
                            signer.capabilities.publish(capD, at: /public/d)
                            let capE = signer.capabilities.storage.issue<&Test.Foo>(/storage/fifth)
                            signer.capabilities.publish(capE, at: /public/e)
                            let capF = signer.capabilities.storage.issue<&String>(/storage/sixth)
                            signer.capabilities.publish(capF, at: /public/f)
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

		// Make the `Test` contract broken. i.e: `Test.Foo` type is broken
		contractIsBroken = true

		runtimeInterface = newRuntimeInterface()

		runtimeInterface.OnGetAndSetProgram = func(
			location Location,
			load func() (*interpreter.Program, error),
		) (*interpreter.Program, error) {
			program, err := load()
			if err != nil {
				// Return a wrapped error
				return nil, fmt.Errorf("failed to load program: %w", err)
			}
			return program, nil
		}

		// Read value
		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    transaction {
                        prepare(account: &Account) {
                            var total = 0
                            account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                                account.capabilities.borrow<&AnyStruct>(path)!
                                total = total + 1
                                return true
                            })

                            // Total values iterated should be 4.
                            // The two broken values must be skipped.
                            assert(total == 4)
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
	})

	t.Run("broken impl, stored with interface", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := NewTestLedger(nil, nil)
		nextTransactionLocation := NewTransactionLocationGenerator()
		contractIsBroken := false

		deployFoo := DeploymentTransaction("Foo", []byte(`
            access(all) contract Foo {
                access(all) struct interface Collection {}
            }
        `))

		deployBar := DeploymentTransaction("Bar", []byte(`
            import Foo from 0x1

            access(all) contract Bar {
                access(all) struct CollectionImpl: Foo.Collection {}
            }
        `))

		newRuntimeInterface := func() Interface {
			return &TestRuntimeInterface{
				Storage: ledger,
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken && location.Name == "Bar" {
						// Contract has a semantic error. i.e: Mismatched types at `bar` function
						return []byte(`
                        import Foo from 0x1

                        access(all) contract Bar {
                            access(all) struct CollectionImpl: Foo.Collection {
                                access(all) var mismatch: Int

                                init() {
                                    self.mismatch = "hello"
                                }
                            }
                        }`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					return nil
				},
			}
		}

		// Deploy `Foo` contract

		runtimeInterface := newRuntimeInterface()

		err := runtime.ExecuteTransaction(
			Script{
				Source: deployFoo,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Deploy `Bar` contract

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployBar,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Store values

		runtimeInterface = newRuntimeInterface()

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Bar from 0x1
                    import Foo from 0x1

                    transaction {
                        prepare(signer: auth(Storage, Capabilities) &Account) {
                            signer.storage.save("Hello, World!", to: /storage/first)

                            var structArray: [{Foo.Collection}] = [Bar.CollectionImpl()]
                            signer.storage.save(structArray, to: /storage/second)

                            let capA = signer.capabilities.storage.issue<&String>(/storage/first)
                            signer.capabilities.publish(capA, at: /public/a)

                            let capB = signer.capabilities.storage.issue<&[{Foo.Collection}]>(/storage/second)
                            signer.capabilities.publish(capB, at: /public/b)
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

		// Make the `Bar` contract broken. i.e: `Bar.CollectionImpl` type is broken.
		contractIsBroken = true

		runtimeInterface = newRuntimeInterface()

		// 1) Iterate through public paths

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Foo from 0x1

                    transaction {
                        prepare(account: &Account) {
                            var total = 0
                            var capTaken = false

                            account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                                total = total + 1

                                var cap = account.capabilities.get<&[{Foo.Collection}]>(path)
								if cap.id != 0 {
									cap.check()
									var refArray = cap.borrow()!
									capTaken = true
								}
                                
                                return true
                            })

                            assert(total == 2)
                            assert(capTaken)
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

		// 2) Iterate through storage paths

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Foo from 0x1

                    transaction {
                        prepare(account: &Account) {
                            var total = 0

                            account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                                account.storage.check<[{Foo.Collection}]>(from: path)
                                total = total + 1
                                return true
                            })

                            assert(total == 2)
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
	})

	t.Run("broken impl, published with interface", func(t *testing.T) {

		t.Parallel()

		runtime := NewTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := NewTestLedger(nil, nil)
		nextTransactionLocation := NewTransactionLocationGenerator()
		contractIsBroken := false

		deployFoo := DeploymentTransaction("Foo", []byte(`
            access(all) contract Foo {
                access(all) resource interface Collection {}
            }
        `))

		deployBar := DeploymentTransaction("Bar", []byte(`
            import Foo from 0x1

            access(all) contract Bar {
                access(all) resource CollectionImpl: Foo.Collection {}

                access(all) fun getCollection(): @Bar.CollectionImpl {
                    return <- create Bar.CollectionImpl()
                }
            }
        `))

		newRuntimeInterface := func() Interface {
			return &TestRuntimeInterface{
				Storage: ledger,
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnResolveLocation: NewSingleIdentifierLocationResolver(t),
				OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken && location.Name == "Bar" {
						// Contract has a semantic error. i.e: Mismatched types at `bar` function
						return []byte(`
                        import Foo from 0x1

                        access(all) contract Bar {
                            access(all) resource CollectionImpl: Foo.Collection {
                                access(all) var mismatch: Int

                                init() {
                                    self.mismatch = "hello"
                                }
                            }
                        }`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					return nil
				},
			}
		}

		// Deploy ``Foo` contract

		runtimeInterface := newRuntimeInterface()

		err := runtime.ExecuteTransaction(
			Script{
				Source: deployFoo,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Deploy `Bar` contract

		err = runtime.ExecuteTransaction(
			Script{
				Source: deployBar,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Store values

		runtimeInterface = newRuntimeInterface()

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Bar from 0x1
                    import Foo from 0x1

                    transaction {
                        prepare(signer: auth(Storage, Capabilities) &Account) {
                            signer.storage.save("Hello, World!", to: /storage/first)
                            signer.storage.save(<- Bar.getCollection(), to: /storage/second)

                            let capA = signer.capabilities.storage.issue<&String>(/storage/first)
                            signer.capabilities.publish(capA, at: /public/a)

                            let capB = signer.capabilities.storage.issue<&{Foo.Collection}>(/storage/second)
                            signer.capabilities.publish(capB, at: /public/b)
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

		// Make the `Bar` contract broken. i.e: `Bar.CollectionImpl` type is broken.
		contractIsBroken = true

		runtimeInterface = newRuntimeInterface()

		// 1) Iterate through public paths

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Foo from 0x1

                    transaction {
                        prepare(account: &Account) {
                            var total = 0
                            var capTaken = false

                            account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                                total = total + 1

                                var cap = account.capabilities.get<&{Foo.Collection}>(path)
								if cap.id != 0 {
									cap.check()
									capTaken = true
								}

                                return true
                            })

                            // Total values iterated should be 1.
                            // The broken value must be skipped.
                            assert(total == 1)

                            // Should not reach this path, because the iteration skip the value altogether.
                            assert(!capTaken)
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

		// 2) Iterate through storage paths

		err = runtime.ExecuteTransaction(
			Script{
				Source: []byte(`
                    import Foo from 0x1

                    transaction {
                        prepare(account: &Account) {
                            var total = 0
                            var capTaken = false

                            account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                                account.storage.check<@{Foo.Collection}>(from: path)
                                total = total + 1
                                return true
                            })

                            // Total values iterated should be 1.
                            // The broken value must be skipped.
                            assert(total == 1)
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
	})

	t.Run("published with wrong type", func(t *testing.T) {

		t.Parallel()

		test := func(brokenType bool, t *testing.T) {

			runtime := NewTestInterpreterRuntime()
			address := common.MustBytesToAddress([]byte{0x1})
			accountCodes := map[common.Location][]byte{}
			ledger := NewTestLedger(nil, nil)
			nextTransactionLocation := NewTransactionLocationGenerator()
			contractIsBroken := false

			deployFoo := DeploymentTransaction("Foo", []byte(`
              access(all) contract Foo {
                  access(all) resource interface Collection {}
              }
            `))

			deployBar := DeploymentTransaction("Bar", []byte(`
              import Foo from 0x1

              access(all) contract Bar {
                  access(all) resource CollectionImpl: Foo.Collection {}

                  access(all) fun getCollection(): @Bar.CollectionImpl {
                      return <- create Bar.CollectionImpl()
                  }
              }
            `))

			newRuntimeInterface := func() Interface {
				return &TestRuntimeInterface{
					Storage: ledger,
					OnGetSigningAccounts: func() ([]Address, error) {
						return []Address{address}, nil
					},
					OnResolveLocation: NewSingleIdentifierLocationResolver(t),
					OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
						accountCodes[location] = code
						return nil
					},
					OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
						if contractIsBroken && location.Name == "Bar" {
							// Contract has a semantic error. i.e: Mismatched types at `bar` function
							return []byte(`
                              import Foo from 0x1

                              access(all) contract Bar {
                                  access(all) resource CollectionImpl: Foo.Collection {
                                      access(all) var mismatch: Int

                                      init() {
                                          self.mismatch = "hello"
                                      }
                                  }
                              }
                            `), nil
						}

						code = accountCodes[location]
						return code, nil
					},
					OnEmitEvent: func(event cadence.Event) error {
						return nil
					},
				}
			}

			// Deploy ``Foo` contract

			runtimeInterface := newRuntimeInterface()

			err := runtime.ExecuteTransaction(
				Script{
					Source: deployFoo,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			// Deploy `Bar` contract

			err = runtime.ExecuteTransaction(
				Script{
					Source: deployBar,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)

			// Store values

			runtimeInterface = newRuntimeInterface()

			err = runtime.ExecuteTransaction(
				Script{
					Source: []byte(`
                      import Bar from 0x1
                      import Foo from 0x1

                      transaction {
                          prepare(signer: auth(Storage, Capabilities) &Account) {
                              signer.storage.save("Hello, World!", to: /storage/first)
                              signer.storage.save(<- Bar.getCollection(), to: /storage/second)

                              let capA = signer.capabilities.storage.issue<&String>(/storage/first)
                              signer.capabilities.publish(capA, at: /public/a)

                              let capB = signer.capabilities.storage.issue<&String>(/storage/second)
                              signer.capabilities.publish(capB, at: /public/b)
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

			// Make the `Bar` contract broken. i.e: `Bar.CollectionImpl` type is broken.
			contractIsBroken = brokenType

			runtimeInterface = newRuntimeInterface()

			// Iterate through public paths

			// If the type is broken, iterator should only find 1 value.
			// Otherwise, it should find all values (2).
			count := 2
			if brokenType {
				count = 1
			}

			err = runtime.ExecuteTransaction(
				Script{
					Source: []byte(fmt.Sprintf(`
                          import Foo from 0x1

                          transaction {
                              prepare(account: &Account) {
                                  var total = 0
                                  account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                                      var cap = account.capabilities.get<&String>(path)
                                      cap.check()
                                      total = total + 1
                                      return true
                                  })

                                  // The broken value must be skipped.
                                  assert(total == %d)
                              }
                          }
                        `,
						count,
					)),
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)
			require.NoError(t, err)
		}

		t.Run("broken type in storage", func(t *testing.T) {
			test(true, t)
		})

		t.Run("valid type in storage", func(t *testing.T) {
			test(false, t)
		})
	})
}

func TestRuntimeStorageIteration2(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	newRuntime := func() (TestInterpreterRuntime, *TestRuntimeInterface) {
		runtime := NewTestInterpreterRuntime()
		accountCodes := map[common.Location][]byte{}

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				return nil
			},
		}
		return runtime, runtimeInterface
	}

	t.Run("paths field", func(t *testing.T) {

		t.Parallel()

		const testContract = `
          access(all)
          contract Test {
              access(all)
              fun saveStorage() {
                  self.account.storage.save(0, to:/storage/foo)
              }

              access(all)
              fun saveOtherStorage() {
                  self.account.storage.save(0, to:/storage/bar)
              }

              access(all)
              fun loadStorage() {
                  self.account.storage.load<Int>(from:/storage/foo)
              }

              access(all)
              fun publish() {
                  let cap = self.account.capabilities.storage.issue<&Int>(/storage/foo)
                  self.account.capabilities.publish(cap, at: /public/foo)
              }

              access(all)
              fun unpublish() {
                  self.account.capabilities.unpublish(/public/foo)
              }

              access(all)
              fun getStoragePaths(): &[StoragePath] {
                  return self.account.storage.storagePaths
              }

              access(all)
              fun getPublicPaths(): &[PublicPath] {
                  return getAccount(self.account.address).storage.publicPaths
              }
          }
        `

		contractLocation := common.NewAddressLocation(nil, address, "Test")

		deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		err := runtime.ExecuteTransaction(
			Script{
				Source: deployTestContractTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		invoke := func(name string) (cadence.Value, error) {
			return runtime.InvokeContractFunction(
				contractLocation,
				name,
				nil,
				nil,
				Context{Interface: runtimeInterface},
			)
		}

		t.Run("before any save", func(t *testing.T) {

			value, err := invoke("getStoragePaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths := value.(cadence.Array).Values
			require.Equal(t, 0, len(paths))

			value, err = invoke("getPublicPaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths = value.(cadence.Array).Values
			require.Equal(t, 0, len(paths))
		})

		t.Run("storage save", func(t *testing.T) {
			_, err := invoke("saveStorage")
			require.NoError(t, err)

			value, err := invoke("getStoragePaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths := value.(cadence.Array).Values
			require.Equal(t, 1, len(paths))
			expectedPath, err := cadence.NewPath(common.PathDomainStorage, "foo")
			require.NoError(t, err)
			require.Equal(t, expectedPath, paths[0])

			value, err = invoke("getPublicPaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths = value.(cadence.Array).Values
			require.Equal(t, 0, len(paths))
		})

		t.Run("publish", func(t *testing.T) {
			_, err := invoke("publish")
			require.NoError(t, err)

			value, err := invoke("getStoragePaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths := value.(cadence.Array).Values
			require.Equal(t, 1, len(paths))
			require.Equal(t, cadence.MustNewPath(common.PathDomainStorage, "foo"), paths[0])

			value, err = invoke("getPublicPaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths = value.(cadence.Array).Values
			require.Equal(t, 1, len(paths))
			require.Equal(t, cadence.MustNewPath(common.PathDomainPublic, "foo"), paths[0])
		})

		t.Run("save storage bar", func(t *testing.T) {
			_, err := invoke("saveOtherStorage")
			require.NoError(t, err)

			value, err := invoke("getStoragePaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths := value.(cadence.Array).Values
			require.Equal(t, 2, len(paths))
			require.Equal(t, cadence.MustNewPath(common.PathDomainStorage, "bar"), paths[0])
			require.Equal(t, cadence.MustNewPath(common.PathDomainStorage, "foo"), paths[1])

			value, err = invoke("getPublicPaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths = value.(cadence.Array).Values
			require.Equal(t, 1, len(paths))
			require.Equal(t, cadence.MustNewPath(common.PathDomainPublic, "foo"), paths[0])
		})

		t.Run("load storage", func(t *testing.T) {
			_, err := invoke("loadStorage")
			require.NoError(t, err)

			value, err := invoke("getStoragePaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths := value.(cadence.Array).Values
			require.Equal(t, 1, len(paths))
			require.Equal(t, cadence.MustNewPath(common.PathDomainStorage, "bar"), paths[0])

			value, err = invoke("getPublicPaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths = value.(cadence.Array).Values
			require.Equal(t, 1, len(paths))
			require.Equal(t, cadence.MustNewPath(common.PathDomainPublic, "foo"), paths[0])
		})

		t.Run("unpublish", func(t *testing.T) {
			_, err := invoke("unpublish")
			require.NoError(t, err)

			value, err := invoke("getStoragePaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths := value.(cadence.Array).Values
			require.Equal(t, 1, len(paths))
			require.Equal(t, cadence.MustNewPath(common.PathDomainStorage, "bar"), paths[0])

			value, err = invoke("getPublicPaths")
			require.NoError(t, err)
			require.IsType(t, cadence.Array{}, value)
			paths = value.(cadence.Array).Values
			require.Equal(t, 0, len(paths))
		})
	})

	t.Run("forEachPublic PublicAccount", func(t *testing.T) {

		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)
              let pubAccount = getAccount(0x1)

              account.storage.save(S(value: 2), to: /storage/foo)
              account.storage.save("", to: /storage/bar)
              let capA = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capA, at: /public/a)
              let capB = account.capabilities.storage.issue<&String>(/storage/bar)
              account.capabilities.publish(capB, at: /public/b)
              let capC = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capC, at: /public/c)
              let capD = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capD, at: /public/d)
              let capE = account.capabilities.storage.issue<&String>(/storage/bar)
              account.capabilities.publish(capE, at: /public/e)

              var total = 0
              pubAccount.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                  if type == Type<Capability<&S>>() {
                      total = total + pubAccount.capabilities.borrow<&S>(path)!.value
                  }
                  return true
              })

              return total
          }
        `

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(6),
			result,
		)
	})

	t.Run("forEachPublic PublicAccount number", func(t *testing.T) {

		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)
              let pubAccount = getAccount(0x1)

              account.storage.save(S(value: 2), to: /storage/foo)
              account.storage.save("", to: /storage/bar)
              let capA = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capA, at: /public/a)
              let capB = account.capabilities.storage.issue<&String>(/storage/bar)
              account.capabilities.publish(capB, at: /public/b)
              let capC = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capC, at: /public/c)
              let capD = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capD, at: /public/d)
              let capE = account.capabilities.storage.issue<&String>(/storage/bar)
              account.capabilities.publish(capE, at: /public/e)

              var total = 0
              pubAccount.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                  total = total + 1
                  return true
              })

              return total
          }
        `

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(5),
			result,
		)
	})

	t.Run("forEachPublic AuthAccount", func(t *testing.T) {
		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)
              let pubAccount = getAccount(0x1)

              account.storage.save(S(value: 2), to: /storage/foo)
              account.storage.save("", to: /storage/bar)
              let capA = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capA, at: /public/a)
              let capB = account.capabilities.storage.issue<&String>(/storage/bar)
              account.capabilities.publish(capB, at: /public/b)
              let capC = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capC, at: /public/c)
              let capD = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capD, at: /public/d)
              let capE = account.capabilities.storage.issue<&String>(/storage/bar)
              account.capabilities.publish(capE, at: /public/e)

              var total = 0
              account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                  if type == Type<Capability<&S>>() {
                      total = total + account.capabilities.borrow<&S>(path)!.value
                  }
                  return true
              })

              return total
           }
        `

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(6),
			result,
		)
	})

	t.Run("forEachStored", func(t *testing.T) {
		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              account.storage.save(S(value: 1), to: /storage/foo1)
              account.storage.save(S(value: 2), to: /storage/foo2)
              account.storage.save(S(value: 5), to: /storage/foo3)
              account.storage.save("", to: /storage/bar1)
              account.storage.save(4, to: /storage/bar2)

              var total = 0
              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      total = total + account.storage.borrow<&S>(from: path)!.value
                  }
                  return true
              })

              return total
          }
        `

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(8),
			result,
		)
	})

	t.Run("forEachStored after empty", func(t *testing.T) {
		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              let value: Int

              init(value: Int) {
                  self.value = value
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage) &Account>(0x1)

              var total = 0
              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  total = total + 1
                  return true
              })

              account.storage.save(S(value: 1), to: /storage/foo1)
              account.storage.save(S(value: 2), to: /storage/foo2)
              account.storage.save(S(value: 5), to: /storage/foo3)

              return total
          }
        `

		nextScriptLocation := NewScriptLocationGenerator()

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(0),
			result,
		)

		const script2 = `
           access(all)
           fun main(): Int {
              let account = getAuthAccount<auth(Storage) &Account>(0x1)

              var total = 0
              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  total = total + 1
                  return true
              })
              return total
          }
        `

		result, err = runtime.ExecuteScript(
			Script{
				Source: []byte(script2),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(3),
			result,
		)
	})

	t.Run("forEachStored with update", func(t *testing.T) {
		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              var value: Int

              init(value: Int) {
                  self.value = value
              }

              access(all)
              fun increment() {
                  self.value = self.value + 1
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage) &Account>(0x1)

              account.storage.save(S(value: 1), to: /storage/foo1)
              account.storage.save(S(value: 2), to: /storage/foo2)
              account.storage.save(S(value: 5), to: /storage/foo3)
              account.storage.save("", to: /storage/bar1)
              account.storage.save(4, to: /storage/bar2)

              var total = 0
              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      account.storage.borrow<&S>(from: path)!.increment()
                  }
                  return true
              })
              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      total = total + account.storage.borrow<&S>(from: path)!.value
                  }
                  return true
              })

              return total
          }
        `

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(11),
			result,
		)
	})

	t.Run("forEachStored with mutation", func(t *testing.T) {
		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              var value: Int

              init(value: Int) {
                  self.value = value
              }

              access(all)
              fun increment() {
                  self.value = self.value + 1
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage) &Account>(0x1)

              account.storage.save(S(value: 1), to: /storage/foo1)
              account.storage.save(S(value: 2), to: /storage/foo2)
              account.storage.save(S(value: 5), to: /storage/foo3)
              account.storage.save("qux", to: /storage/bar1)
              account.storage.save(4, to: /storage/bar2)

              var total = 0
              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      total = total + account.storage.borrow<&S>(from: path)!.value
                  }
                  if type == Type<String>() {
                      let id = account.storage.load<String>(from: path)!
                      account.storage.save(S(value:3), to: StoragePath(identifier: id)!)
                  }
                  return true
              })

              return total
          }
        `

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
	})

	t.Run("forEachStored with early termination", func(t *testing.T) {
		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {
              access(all)
              var value: Int

              init(value: Int) {
                  self.value = value
              }

              access(all)
              fun increment() {
                  self.value = self.value + 1
              }
          }

          access(all)
          fun main(): Int {
              let account = getAuthAccount<auth(Storage) &Account>(0x1)

              account.storage.save(1, to: /storage/foo1)
              account.storage.save(2, to: /storage/foo2)
              account.storage.save(3, to: /storage/foo3)
              account.storage.save(4, to: /storage/bar1)
              account.storage.save(5, to: /storage/bar2)

              var seen = 0
              var stuff: [&AnyStruct] = []
              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if seen >= 3 {
                      return false
                  }
                  stuff.append(account.storage.borrow<&AnyStruct>(from: path)!)
                  seen = seen + 1
                  return true
              })

              return stuff.length
          }
        `

		result, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		assert.Equal(
			t,
			cadence.NewInt(3),
			result,
		)
	})
}

func TestRuntimeAccountIterationMutation(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	newRuntime := func() (TestInterpreterRuntime, *TestRuntimeInterface) {
		runtime := NewTestInterpreterRuntime()
		accountCodes := map[common.Location][]byte{}

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				return nil
			},
		}
		return runtime, runtimeInterface
	}

	test := func(continueAfterMutation bool) {

		t.Run(fmt.Sprintf("forEachStored, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			script := fmt.Sprintf(
				`
                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)
                      account.storage.save(2, to: /storage/foo2)
                      account.storage.save(3, to: /storage/foo3)
                      account.storage.save("qux", to: /storage/foo4)

                      account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                          if type == Type<String>() {
                              account.storage.save("bar", to: /storage/foo5)
                              return %t
                          }
                          return true
                      })
                  }
                `,
				continueAfterMutation,
			)

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)

			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("forEachPublic, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			script := fmt.Sprintf(
				`
                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)

                      let capA = account.capabilities.storage.issue<&Int>(/storage/foo1)
                      account.capabilities.publish(capA, at: /public/foo1)

                      account.storage.save("", to: /storage/foo2)

                      let capB = account.capabilities.storage.issue<&String>(/storage/foo2)
                      account.capabilities.publish(capB, at: /public/foo2)

                      account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                          if type == Type<Capability<&String>>() {
                              account.storage.save("bar", to: /storage/foo3)
                              return %t
                          }
                          return true
                      })
                  }
                `,
				continueAfterMutation,
			)

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)

			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("with function call, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			script := fmt.Sprintf(
				`
                  access(all)
                  fun foo() {
                      let account = getAuthAccount<auth(Storage) &Account>(0x1)

                      account.storage.save("bar", to: /storage/foo5)
                  }

                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)
                      account.storage.save(2, to: /storage/foo2)
                      account.storage.save(3, to: /storage/foo3)
                      account.storage.save("qux", to: /storage/foo4)

                      account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                          if type == Type<String>() {
                              foo()
                              return %t
                          }
                          return true
                      })
                  }
                `,
				continueAfterMutation,
			)

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)

			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("with function call and nested iteration, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			script := fmt.Sprintf(
				`
                  access(all)
                  fun foo() {
                      let account = getAuthAccount<auth(Storage) &Account>(0x1)

                      account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                          return true
                      })
                      account.storage.save("bar", to: /storage/foo5)
                  }

                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)
                      account.storage.save(2, to: /storage/foo2)
                      account.storage.save(3, to: /storage/foo3)
                      account.storage.save("qux", to: /storage/foo4)

                      account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                          if type == Type<String>() {
                              foo()
                              return %t
                          }
                          return true
                      })
                  }
                `,
				continueAfterMutation,
			)

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)

			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("load, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			script := fmt.Sprintf(
				`
                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)
                      account.storage.save(2, to: /storage/foo2)
                      account.storage.save(3, to: /storage/foo3)
                      account.storage.save("qux", to: /storage/foo4)

                      account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                          if type == Type<String>() {
                              account.storage.load<Int>(from: /storage/foo1)
                              return %t
                          }
                          return true
                      })
                   }
                `,
				continueAfterMutation,
			)

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)
			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("publish, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			script := fmt.Sprintf(
				`
                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)
                      account.storage.save("", to: /storage/foo2)
                      let capA = account.capabilities.storage.issue<&Int>(/storage/foo1)
                      account.capabilities.publish(capA, at: /public/foo1)
                      let capB = account.capabilities.storage.issue<&String>(/storage/foo2)
                      account.capabilities.publish(capB, at: /public/foo2)

                      account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                          if type == Type<Capability<&String>>() {
                              account.capabilities.storage.issue<&Int>(/storage/foo1)
                              return %t
                          }
                          return true
                      })
                  }
                `,
				continueAfterMutation,
			)

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)
			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("unpublish, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			script := fmt.Sprintf(
				`
                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)
                      account.storage.save("", to: /storage/foo2)
                      let capA = account.capabilities.storage.issue<&Int>(/storage/foo1)
                      account.capabilities.publish(capA, at: /public/foo1)
                      let capB = account.capabilities.storage.issue<&String>(/storage/foo2)
                      account.capabilities.publish(capB, at: /public/foo2)

                      account.storage.forEachPublic(fun (path: PublicPath, type: Type): Bool {
                          if type == Type<Capability<&String>>() {
                              account.capabilities.unpublish(/public/foo1)
                              return %t
                          }
                          return true
                      })
                  }
                `,
				continueAfterMutation,
			)

			_, err := runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)
			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})

		t.Run(fmt.Sprintf("with imported function call, continue: %t", continueAfterMutation), func(t *testing.T) {
			t.Parallel()

			runtime, runtimeInterface := newRuntime()

			// Deploy contract

			const testContract = `
              access(all)
              contract Test {

                  access(all)
                  fun foo() {
                      self.account.storage.save("bar", to: /storage/foo5)
                  }
              }
            `

			deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

			err := runtime.ExecuteTransaction(
				Script{
					Source: deployTestContractTx,
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.TransactionLocation{},
				},
			)
			require.NoError(t, err)

			// Run test script

			script := fmt.Sprintf(`
                  import Test from 0x1

                  access(all)
                  fun main() {
                      let account = getAuthAccount<auth(Storage) &Account>(0x1)

                      account.storage.save(1, to: /storage/foo1)
                      account.storage.save(2, to: /storage/foo2)
                      account.storage.save(3, to: /storage/foo3)
                      account.storage.save("qux", to: /storage/foo4)

                      account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                          if type == Type<String>() {
                              Test.foo()
                              return %t
                          }
                          return true
                      })
                  }
                `,
				continueAfterMutation,
			)

			_, err = runtime.ExecuteScript(
				Script{
					Source: []byte(script),
				},
				Context{
					Interface: runtimeInterface,
					Location:  common.ScriptLocation{},
				},
			)
			if continueAfterMutation {
				RequireError(t, err)

				require.ErrorAs(t, err, &interpreter.StorageMutatedDuringIterationError{})
			} else {
				require.NoError(t, err)
			}
		})
	}

	test(true)
	test(false)

	t.Run("state properly cleared on iteration end", func(t *testing.T) {
		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage) &Account>(0x1)

              account.storage.save(1, to: /storage/foo1)
              account.storage.save(2, to: /storage/foo2)
              account.storage.save(3, to: /storage/foo3)
              account.storage.save("qux", to: /storage/foo4)

              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  return true
              })
              account.storage.save("bar", to: /storage/foo5)

              account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  account.storage.forEachStored(fun (path: StoragePath, type: Type): Bool {
                      return true
                  })
                  return true
              })
              account.storage.save("baz", to: /storage/foo6)
          }
        `

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)
	})

	t.Run("non-lambda", func(t *testing.T) {
		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          fun foo (path: StoragePath, type: Type): Bool {
              return true
          }

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage) &Account>(0x1)

              account.storage.forEachStored(foo)
          }
        `

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)
	})

	t.Run("method", func(t *testing.T) {
		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          struct S {

              access(all)
              fun foo(path: StoragePath, type: Type): Bool {
                  return true
              }
          }

          access(all)
          fun main() {

              let account = getAuthAccount<auth(Storage) &Account>(0x1)
              let s = S()
              account.storage.forEachStored(s.foo)
          }
        `

		_, err := runtime.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)
	})
}

func TestRuntimeTypeOrderInsignificance(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	newRuntime := func() (TestInterpreterRuntime, *TestRuntimeInterface) {
		runtime := NewTestInterpreterRuntime()
		accountCodes := map[common.Location][]byte{}

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				return nil
			},
		}
		return runtime, runtimeInterface
	}

	t.Run("intersection types", func(t *testing.T) {
		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all)
            contract Test {

                access(all)
                struct interface A {}


                access(all)
                struct interface B {}
            }
        `))

		tx1 := []byte(`
          import Test from 0x1

          transaction {
              prepare(account: auth(Storage) &Account) {

                  let t1 = Type<&{Test.A, Test.B}>()
                  let t2 = Type<&{Test.B, Test.A}>()

                  let dict: {Type: Bool} = {}
                  dict[t1] = true

                  assert(dict[t1]!)
                  assert(dict[t2]!)

                  account.storage.save(dict, to: /storage/dict)
              }
          }
        `)

		tx2 := []byte(`
          import Test from 0x1

          transaction {
              prepare(account: auth(Storage) &Account) {

                  let t1 = Type<&{Test.A, Test.B}>()
                  let t2 = Type<&{Test.B, Test.A}>()

                  let dict = account.storage.load<{Type: Bool}>(from: /storage/dict)!

                  assert(dict[t1]!)
                  assert(dict[t2]!)
              }
          }
        `)

		nextTransactionLocation := NewTransactionLocationGenerator()

		for _, tx := range [][]byte{deployTx, tx1, tx2} {

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
		}
	})

	t.Run("entitlements", func(t *testing.T) {
		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all)
            contract Test {

                access(all)
                entitlement A


                access(all)
                entitlement B
            }
        `))

		tx1 := []byte(`
          import Test from 0x1

          transaction {
              prepare(account: auth(Storage) &Account) {

                  let t1 = Type<auth(Test.A, Test.B) &AnyStruct>()
                  let t2 = Type<auth(Test.B, Test.A) &AnyStruct>()

                  let dict: {Type: Bool} = {}
                  dict[t1] = true

                  assert(dict[t1]!)
                  assert(dict[t2]!)

                  account.storage.save(dict, to: /storage/dict)
              }
          }
        `)

		tx2 := []byte(`
          import Test from 0x1

          transaction {
              prepare(account: auth(Storage) &Account) {

                  let t1 = Type<auth(Test.A, Test.B) &AnyStruct>()
                  let t2 = Type<auth(Test.B, Test.A) &AnyStruct>()

                  let dict = account.storage.load<{Type: Bool}>(from: /storage/dict)!

                  assert(dict[t1]!)
                  assert(dict[t2]!)
              }
          }
        `)

		nextTransactionLocation := NewTransactionLocationGenerator()

		for _, tx := range [][]byte{deployTx, tx1, tx2} {

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
		}
	})
}

func TestRuntimeStorageReferenceBoundFunction(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		runtime := NewTestInterpreterRuntime()

		signerAddress := common.MustBytesToAddress([]byte{0x42})

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {

                access(all) resource R {
                    access(all) fun foo() {}
                }

                access(all) fun createR(): @R {
                    return <-create R()
                }
            }
        `))

		accountCodes := map[Location][]byte{}
		var events []cadence.Event
		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{signerAddress}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

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
                prepare(signer: auth(Storage) &Account) {
                    signer.storage.save(<-Test.createR(), to: /storage/r)

                    let ref = signer.storage.borrow<&Test.R>(from: /storage/r)!

                    var func = ref.foo

                    let r <- signer.storage.load<@Test.R>(from: /storage/r)!

                    // Should fail: Underlying value was removed from storage
                    func()

                    destroy r
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

		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.ReferencedValueChangedError{})
	})

	t.Run("struct", func(t *testing.T) {
		t.Parallel()

		runtime := NewTestInterpreterRuntimeWithAttachments()

		tx := []byte(`
            transaction {

               prepare(signer: auth(Storage, Capabilities) &Account) {

                  signer.storage.save([] as [AnyStruct], to: /storage/zombieArray)
                  var borrowed = signer.storage.borrow<auth(Mutate) &[AnyStruct]>(from: /storage/zombieArray)!

                  var x: [Int] = []

                  var appendFunc = borrowed.append

                  // If we were to call appendFunc() here, we wouldn't see a big effect as the
                  // next load() call  will remove the array from storage
                  var throwaway = signer.storage.load<[AnyStruct]>(from: /storage/zombieArray)

                  // Should be an error, since the value was moved out.
                  appendFunc(x)
               }
            }
        `)

		signer := common.MustBytesToAddress([]byte{0x1})

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{signer}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

		err := runtime.ExecuteTransaction(
			Script{
				Source: tx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			})

		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.ReferencedValueChangedError{})
	})

	t.Run("replace resource", func(t *testing.T) {

		runtime := NewTestInterpreterRuntime()

		signerAddress := common.MustBytesToAddress([]byte{0x42})

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {

                access(all) resource Foo {
                    access(all) fun hello() {}
                }

                access(all) fun createFoo(): @Foo {
                    return <-create Foo()
                }

                access(all) resource Bar {
                    access(all) fun hello() {}
                }

                access(all) fun createBar(): @Bar {
                    return <-create Bar()
                }
            }
        `))

		accountCodes := map[Location][]byte{}
		var events []cadence.Event
		var loggedMessages []string

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{signerAddress}, nil
			},
			OnResolveLocation: NewSingleIdentifierLocationResolver(t),
			OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			OnProgramLog: func(message string) {
				loggedMessages = append(loggedMessages, message)
			},
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

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
                prepare(signer: auth(Storage) &Account) {
                    signer.storage.save(<-Test.createFoo(), to: /storage/xyz)
                    let ref = signer.storage.borrow<&Test.Foo>(from: /storage/xyz)!

                    // Take a reference to 'Foo.hello'
                    var hello = ref.hello

                    // Remove 'Foo'
                    let foo <- signer.storage.load<@Test.Foo>(from: /storage/xyz)!

                    // Replace it with 'Bar' value
                    signer.storage.save(<-Test.createBar(), to: /storage/xyz)

                    // Should be an error
                    hello()

                    destroy foo
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

		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.DereferenceError{})
	})

}

func TestRuntimeStorageReferenceAccess(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	deployTx := DeploymentTransaction("Test", []byte(`
      access(all)
      contract Test {

          access(all)
          resource R {

              access(all)
              var balance: Int

              init() {
                  self.balance = 10
              }
          }

          access(all)
          fun createR(): @R {
              return <-create R()
          }
      }
    `))

	accountCodes := map[Location][]byte{}
	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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

	t.Run("top-level reference", func(t *testing.T) {

		transferTx := []byte(`
          import Test from 0x1

          transaction {
              prepare(signer: auth(Storage) &Account) {
                  signer.storage.save(<-Test.createR(), to: /storage/test)
                  let ref = signer.storage.borrow<&Test.R>(from: /storage/test)!
                  let value <- signer.storage.load<@Test.R>(from: /storage/test)!
                  destroy value
                  ref.balance
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
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.DereferenceError{})
	})

	t.Run("optional reference", func(t *testing.T) {

		transferTx := []byte(`
          import Test from 0x1

          transaction {
              prepare(signer: auth(Storage) &Account) {
                  signer.storage.save(<-Test.createR(), to: /storage/test)
                  let ref = signer.storage.borrow<&Test.R>(from: /storage/test)
                  let value <- signer.storage.load<@Test.R>(from: /storage/test)!
                  destroy value
                  ref?.balance
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
		RequireError(t, err)
		require.ErrorAs(t, err, &interpreter.DereferenceError{})
	})
}
