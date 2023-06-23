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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
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

		if storage.newStorageMaps == nil {
			storage.newStorageMaps = &orderedmap.OrderedMap[interpreter.StorageKey, atree.StorageIndex]{}
		}
		storage.newStorageMaps.Set(storageKey, storageIndex)
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

	runtime := newTestInterpreterRuntime()

	address := common.MustBytesToAddress([]byte{0x1})

	tx := []byte(`
      transaction {
          prepare(signer: AuthAccount) {
              signer.save(1, to: /storage/one)
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

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signingAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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
        prepare(acct: AuthAccount) {

          let rc <- TestContract.createConverter()
          acct.save(<-rc, to: /storage/rc)

          let cap = acct.capabilities.storage.issue<&TestContract.resourceConverter2>(/storage/rc)
          acct.capabilities.publish(cap, at: /public/rc)

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
			cadence.NewIDCapability(
				1,
				cadence.Address(signer),
				cadence.NewReferenceType(
					cadence.Unauthorized{},
					cadence.IntType{},
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

	runtime := newTestInterpreterRuntime()

	testAddress, err := common.HexToAddress("0x0b2a3299cc857e29")
	require.NoError(t, err)

	nextTransactionLocation := newTransactionLocationGenerator()

	nftAddress, err := common.HexToAddress("0x1d7e57aa55817448")
	require.NoError(t, err)

	accountCodes := map[common.Location]string{
		common.AddressLocation{
			Address: nftAddress,
			Name:    "NonFungibleToken",
		}: realNonFungibleTokenInterface,
	}

	events := make([]cadence.Event, 0)

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{testAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = []byte(accountCodes[location])
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

	runtime := newTestInterpreterRuntime()

	nftAddress, err := common.HexToAddress("0x1d7e57aa55817448")
	require.NoError(t, err)

	accountCodes := map[common.Location]string{
		common.AddressLocation{
			Address: nftAddress,
			Name:    "NonFungibleToken",
		}: realNonFungibleTokenInterface,
	}

	deployTx := DeploymentTransaction("TopShot", []byte(realTopShotContract))

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
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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
          let transferTokens: @NonFungibleToken.Collection

          prepare(acct: AuthAccount) {
              let ref = acct.borrow<&TopShot.Collection>(from: /storage/MomentCollection)!
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

	runtime := newTestInterpreterRuntime()

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

              destroy() {
                  destroy self.ownedNFTs
              }
          }

          init() {
              self.account.save(
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

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = string(code)
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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

          prepare(signer: AuthAccount) {
              self.collection <- signer.borrow<&Test.Collection>(from: /storage/MainCollection)!
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

	// Store a value and publish a capability

	err := runtime.ExecuteTransaction(
		Script{
			Source: []byte(`
              transaction {
                  prepare(signer: AuthAccount) {
                      signer.save(42, to: /storage/test)

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
                prepare(signer: AuthAccount) {
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
                  prepare(signer: AuthAccount) {
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

func TestRuntimeStorageSaveIDCapability(t *testing.T) {

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

	ty := &cadence.ReferenceType{
		Authorization: cadence.UnauthorizedAccess,
		Type:          cadence.IntType{},
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
                      prepare(signer: AuthAccount) {
                          let cap = signer.capabilities.storage.issue<%[1]s>(/storage/test)!
                          signer.capabilities.publish(cap, at: /public/test)
                          signer.save(cap, to: %[2]s)

                          let cap2 = signer.capabilities.get<%[1]s>(/public/test)!
                          signer.save(cap2, to: %[3]s)
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

	expected := cadence.NewIDCapability(
		cadence.UInt64(1),
		cadence.Address(signer),
		ty,
	)

	actual := cadence.ValueWithCachedTypeID(value)
	require.Equal(t, expected, actual)
}

func TestRuntimeStorageReferenceCast(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

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

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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

              let cap = signer.capabilities.storage.issue<&Test.R{Test.RI}>(/storage/r)
              signer.capabilities.publish(cap, at: /public/r)

              let ref = signer.capabilities.borrow<&Test.R{Test.RI}>(/public/r)!

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

	runtime := newTestInterpreterRuntime()

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

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signerAddress}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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

              let cap = signer.capabilities.storage.issue<&Test.R{Test.RI}>(/storage/r)
              signer.capabilities.publish(cap, at: /public/r)

              let ref = signer.capabilities.borrow<&Test.R{Test.RI}>(/public/r)!

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
			RequireError(t, err)

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
	RequireError(t, err)

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

	runtime := newTestInterpreterRuntime()
	runtime.defaultConfig.ResourceOwnerChangeHandlerEnabled = true

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	ledger := newTestLedger(nil, nil)

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

	runtimeInterface := &testRuntimeInterface{
		storage: ledger,
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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
					uuid:       resource.ResourceUUID(inter, interpreter.EmptyLocationRange),
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

    access(all) fun attach(asRole: Role, receiver: &AnyResource{Receiver}) {
        // TODO: Now verify that the owner is valid.

        let capability = self.capabilities[asRole]!
        receiver.receive(asRole: asRole, capability: capability)
    }

    init() {
        self.account.save<@AAA>(<- create AAA(), to: /storage/TestAAA)
        self.account.save<@BBB>(<- create BBB(), to: /storage/TestBBB)

        self.capabilities = {}
        self.capabilities[Role.aaa] = self.account.capabilities.storage.issue<&AAA>(/storage/TestAAA)!
        self.capabilities[Role.bbb] = self.account.capabilities.storage.issue<&BBB>(/storage/TestBBB)!
    }
}

`

	const tx = `
import Test from 0x1

transaction {
    prepare(acct: AuthAccount) {}
    execute {
        let holder <- Test.createHolder()
        Test.attach(asRole: Test.Role.aaa, receiver: &holder as &AnyResource{Test.Receiver})
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

              prepare(accountA: AuthAccount, accountB: AuthAccount) {

                  let testResource <- TestContract.makeTestResource()
                  let ref1 = &testResource as &TestContract.TestResource

                  // At this point the resource is not in storage
                  log(ref1.owner?.address)

                  accountA.save(<-testResource, to: /storage/test)

                  // At this point the resource is in storage A
                  let cap = accountA.capabilities.storage.issue<&TestContract.TestResource>(/storage/test)
                  accountA.capabilities.publish(cap, at: /public/test)

                  let ref2 = accountA.capabilities.borrow<&TestContract.TestResource>(/public/test)!
                  log(ref2.owner?.address)

                  let testResource2 <- accountA.load<@TestContract.TestResource>(from: /storage/test)!

                  let ref3 = &testResource2 as &TestContract.TestResource

                   // At this point the resource is not in storage
                  log(ref3.owner?.address)

                  accountB.save(<-testResource2, to: /storage/test)

                  let cap2 = accountB.capabilities.storage.issue<&TestContract.TestResource>(/storage/test)
                  accountB.capabilities.publish(cap2, at: /public/test)

                  let ref4 = accountB.capabilities.borrow<&TestContract.TestResource>(/public/test)!

                  // At this point the resource is in storage B
                  log(ref4.owner?.address)
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

              prepare(account: AuthAccount) {

                  let testResources <- [<-TestContract.makeTestResource()]
                  let ref1 = &testResources[0] as &TestContract.TestResource

                  // At this point the resource is not in storage
                  log(ref1.owner?.address)

                  account.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  let cap = account.capabilities.storage.issue<&[TestContract.TestResource]>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  let ref2 = account.capabilities.borrow<&[TestContract.TestResource]>(/public/test)!
                  let ref3 = &ref2[0] as &TestContract.TestResource
                  log(ref3.owner?.address)
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

                  destroy () {
                      destroy self.nestedResources
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

              prepare(account: AuthAccount) {

                  let nestingResource <- TestContract.makeTestNestingResource()
                  var nestingResourceRef = &nestingResource as &TestContract.TestNestingResource
                  var nestedElementResourceRef = &nestingResource.nestedResources[0] as &TestContract.TestNestedResource

                  // At this point the nesting and nested resources are not in storage
                  log(nestingResourceRef.owner?.address)
                  log(nestedElementResourceRef.owner?.address)

                  account.save(<-nestingResource, to: /storage/test)

                  // At this point the nesting and nested resources are both in storage
                  let cap = account.capabilities.storage.issue<&TestContract.TestNestingResource>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  nestingResourceRef = account.capabilities.borrow<&TestContract.TestNestingResource>(/public/test)!
                  nestedElementResourceRef = &nestingResourceRef.nestedResources[0] as &TestContract.TestNestedResource

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

              prepare(account: AuthAccount) {

                  let testResources <- [<-[<-TestContract.makeTestResource()]]
                  var ref = &testResources[0] as &[TestContract.TestResource]

                  // At this point the resource is not in storage
                  log(ref[0].owner?.address)

                  account.save(<-testResources, to: /storage/test)

                  // At this point the resource is in storage
                  let cap = account.capabilities.storage.issue<&[[TestContract.TestResource]]>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  let testResourcesRef = account.capabilities.borrow<&[[TestContract.TestResource]]>(/public/test)!
                  ref = &testResourcesRef[0] as &[TestContract.TestResource]
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

              prepare(account: AuthAccount) {

                  let testResources <- [<-{0: <-TestContract.makeTestResource()}]
                  var ref = &testResources[0] as &{Int: TestContract.TestResource}

                  // At this point the resource is not in storage
                  log(ref[0]?.owner?.address)

                  account.save(<-testResources, to: /storage/test)

                  let cap = account.capabilities.storage.issue<&[{Int: TestContract.TestResource}]>(/storage/test)
                  account.capabilities.publish(cap, at: /public/test)

                  let testResourcesRef = account.capabilities.borrow<&[{Int: TestContract.TestResource}]>(/public/test)!

                  ref = &testResourcesRef[0] as &{Int: TestContract.TestResource}
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
			RequireError(t, err)

			require.Contains(t, err.Error(), "cannot store non-storable value")
		}
	})
}

// TestRuntimeStorageEnumCase tests the writing an enum case to storage,
// reading it back from storage, as well as using it to index into a dictionary.
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
		updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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

                        destroy() {
                             destroy self.rs
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

func TestRuntimeStorageReadNoImplicitWrite(t *testing.T) {

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

	runtime := newTestInterpreterRuntime()

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

	ledger := newTestLedger(nil, nil)

	newRuntimeInterface := func() Interface {
		return &testRuntimeInterface{
			storage: ledger,
			getSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
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

		runtime := newTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := newTestLedger(nil, nil)
		nextTransactionLocation := newTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() (Interface, *[]Location) {

			var programStack []Location

			runtimeInterface := &testRuntimeInterface{
				storage: ledger,
				getSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				resolveLocation: singleIdentifierLocationResolver(t),
				updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract no longer has the type
						return []byte(`access(all) contract Test {}`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				emitEvent: func(event cadence.Event) error {
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
                        prepare(signer: AuthAccount) {
                            signer.save("Hello, World!", to: /storage/first)
                            signer.save(["one", "two", "three"], to: /storage/second)
                            signer.save(Test.Foo(), to: /storage/third)
                            signer.save(1, to: /storage/fourth)
                            signer.save(Test.Foo(), to: /storage/fifth)
                            signer.save("two", to: /storage/sixth)
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
                        prepare(account: AuthAccount) {
                            var total = 0
                            account.forEachStored(fun (path: StoragePath, type: Type): Bool {
                                account.borrow<&AnyStruct>(from: path)!
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

		runtime := newTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := newTestLedger(nil, nil)
		nextTransactionLocation := newTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() Interface {
			return &testRuntimeInterface{
				storage: ledger,
				getSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				resolveLocation: singleIdentifierLocationResolver(t),
				updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract has a syntax problem
						return []byte(`BROKEN`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				emitEvent: func(event cadence.Event) error {
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
                        prepare(signer: AuthAccount) {
                            signer.save("Hello, World!", to: /storage/first)
                            signer.save(["one", "two", "three"], to: /storage/second)
                            signer.save(Test.Foo(), to: /storage/third)
                            signer.save(1, to: /storage/fourth)
                            signer.save(Test.Foo(), to: /storage/fifth)
                            signer.save("two", to: /storage/sixth)

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
                        prepare(account: AuthAccount) {
                            var total = 0
                            account.forEachPublic(fun (path: PublicPath, type: Type): Bool {
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

		runtime := newTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := newTestLedger(nil, nil)
		nextTransactionLocation := newTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() Interface {
			return &testRuntimeInterface{
				storage: ledger,
				getSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				resolveLocation: singleIdentifierLocationResolver(t),
				updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract has a semantic error. i.e: cannot find `Bar`
						return []byte(`access(all) contract Test {
                            access(all) struct Foo: Bar {}
                        }`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				emitEvent: func(event cadence.Event) error {
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
                        prepare(signer: AuthAccount) {
                            signer.save("Hello, World!", to: /storage/first)
                            signer.save(["one", "two", "three"], to: /storage/second)
                            signer.save(Test.Foo(), to: /storage/third)
                            signer.save(1, to: /storage/fourth)
                            signer.save(Test.Foo(), to: /storage/fifth)
                            signer.save("two", to: /storage/sixth)

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
                        prepare(account: AuthAccount) {
                            var total = 0
                            account.forEachPublic(fun (path: PublicPath, type: Type): Bool {
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

		runtime := newTestInterpreterRuntime()
		address := common.MustBytesToAddress([]byte{0x1})
		accountCodes := map[common.Location][]byte{}
		ledger := newTestLedger(nil, nil)
		nextTransactionLocation := newTransactionLocationGenerator()
		contractIsBroken := false

		deployTx := DeploymentTransaction("Test", []byte(`
            access(all) contract Test {
                access(all) struct Foo {}
            }
        `))

		newRuntimeInterface := func() *testRuntimeInterface {
			return &testRuntimeInterface{
				storage: ledger,
				getSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				resolveLocation: singleIdentifierLocationResolver(t),
				updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
					accountCodes[location] = code
					return nil
				},
				getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
					if contractIsBroken {
						// Contract has a semantic error. i.e: cannot find `Bar`
						return []byte(`access(all) contract Test {
                            access(all) struct Foo: Bar {}
                        }`), nil
					}

					code = accountCodes[location]
					return code, nil
				},
				emitEvent: func(event cadence.Event) error {
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
                        prepare(signer: AuthAccount) {
                            signer.save("Hello, World!", to: /storage/first)
                            signer.save(["one", "two", "three"], to: /storage/second)
                            signer.save(Test.Foo(), to: /storage/third)
                            signer.save(1, to: /storage/fourth)
                            signer.save(Test.Foo(), to: /storage/fifth)
                            signer.save("two", to: /storage/sixth)

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

		runtimeInterface.getAndSetProgram = func(
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
                        prepare(account: AuthAccount) {
                            var total = 0
                            account.forEachPublic(fun (path: PublicPath, type: Type): Bool {
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
}

func TestRuntimeStorageIteration2(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	newRuntime := func() (testInterpreterRuntime, *testRuntimeInterface) {
		runtime := newTestInterpreterRuntime()
		accountCodes := map[common.Location][]byte{}

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			emitEvent: func(event cadence.Event) error {
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
                  self.account.save(0, to:/storage/foo)
              }

              access(all)
              fun saveOtherStorage() {
                  self.account.save(0, to:/storage/bar)
              }

              access(all)
              fun loadStorage() {
                  self.account.load<Int>(from:/storage/foo)
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
              fun getStoragePaths(): [StoragePath] {
                  return self.account.storagePaths
              }

              access(all)
              fun getPublicPaths(): [PublicPath] {
                  return getAccount(self.account.address).publicPaths
              }
          }
        `

		contractLocation := common.NewAddressLocation(nil, address, "Test")

		deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := newTransactionLocationGenerator()

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
              let account = getAuthAccount(0x1)
              let pubAccount = getAccount(0x1)

              account.save(S(value: 2), to: /storage/foo)
              account.save("", to: /storage/bar)
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
              pubAccount.forEachPublic(fun (path: PublicPath, type: Type): Bool {
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
              let account = getAuthAccount(0x1)
              let pubAccount = getAccount(0x1)

              account.save(S(value: 2), to: /storage/foo)
              account.save("", to: /storage/bar)
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
              pubAccount.forEachPublic(fun (path: PublicPath, type: Type): Bool {
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
              let account = getAuthAccount(0x1)
              let pubAccount = getAccount(0x1)

              account.save(S(value: 2), to: /storage/foo)
              account.save("", to: /storage/bar)
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
              account.forEachPublic(fun (path: PublicPath, type: Type): Bool {
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

	t.Run("forEachPrivate", func(t *testing.T) {

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
              let account = getAuthAccount(0x1)
              let pubAccount = getAccount(0x1)

              account.save(S(value: 2), to: /storage/foo)
              account.save("test", to: /storage/bar)
              let capA = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(capA, at: /public/a)

              var total = 0
              account.forEachPrivate(fun (path: PrivatePath, type: Type): Bool {
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
			cadence.NewInt(0),
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
              let account = getAuthAccount(0x1)

              account.save(S(value: 1), to: /storage/foo1)
              account.save(S(value: 2), to: /storage/foo2)
              account.save(S(value: 5), to: /storage/foo3)
              account.save("", to: /storage/bar1)
              account.save(4, to: /storage/bar2)

              var total = 0
              account.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      total = total + account.borrow<&S>(from: path)!.value
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
              let account = getAuthAccount(0x1)

              var total = 0
              account.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  total = total + 1
                  return true
              })

              account.save(S(value: 1), to: /storage/foo1)
              account.save(S(value: 2), to: /storage/foo2)
              account.save(S(value: 5), to: /storage/foo3)

              return total
          }
        `

		nextScriptLocation := newScriptLocationGenerator()

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
              let account = getAuthAccount(0x1)

              var total = 0
              account.forEachStored(fun (path: StoragePath, type: Type): Bool {
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
              let account = getAuthAccount(0x1)

              account.save(S(value: 1), to: /storage/foo1)
              account.save(S(value: 2), to: /storage/foo2)
              account.save(S(value: 5), to: /storage/foo3)
              account.save("", to: /storage/bar1)
              account.save(4, to: /storage/bar2)

              var total = 0
              account.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      account.borrow<&S>(from: path)!.increment()
                  }
                  return true
              })
              account.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      total = total + account.borrow<&S>(from: path)!.value
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
              let account = getAuthAccount(0x1)

              account.save(S(value: 1), to: /storage/foo1)
              account.save(S(value: 2), to: /storage/foo2)
              account.save(S(value: 5), to: /storage/foo3)
              account.save("qux", to: /storage/bar1)
              account.save(4, to: /storage/bar2)

              var total = 0
              account.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if type == Type<S>() {
                      total = total + account.borrow<&S>(from: path)!.value
                  }
                  if type == Type<String>() {
                      let id = account.load<String>(from: path)!
                      account.save(S(value:3), to: StoragePath(identifier: id)!)
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
              let account = getAuthAccount(0x1)

              account.save(1, to: /storage/foo1)
              account.save(2, to: /storage/foo2)
              account.save(3, to: /storage/foo3)
              account.save(4, to: /storage/bar1)
              account.save(5, to: /storage/bar2)

              var seen = 0
              var stuff: [&AnyStruct] = []
              account.forEachStored(fun (path: StoragePath, type: Type): Bool {
                  if seen >= 3 {
                      return false
                  }
                  stuff.append(account.borrow<&AnyStruct>(from: path)!)
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
