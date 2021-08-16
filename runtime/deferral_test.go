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

// TODO:
//import (
//	"encoding/hex"
//	"fmt"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/require"
//
//	"github.com/onflow/cadence"
//	"github.com/onflow/cadence/runtime/common"
//	"github.com/onflow/cadence/runtime/tests/utils"
//)
//
//const simpleDeferralContract = `
//  pub contract Test {
//
//      pub resource R {
//
//          pub var value: Int
//
//          init(_ value: Int) {
//              self.value = value
//          }
//
//          pub fun increment() {
//              self.value = self.value + 1
//          }
//
//          destroy() {
//              log("destroying R")
//              log(self.value)
//          }
//      }
//
//      pub fun createR(_ value: Int): @R {
//          return <-create R(value)
//      }
//
//      pub resource C {
//
//          pub let rs: @{String: R}
//
//          init() {
//              self.rs <- {}
//          }
//
//          pub fun remove(_ id: String): @R {
//              let r <- self.rs.remove(key: id) ?? panic("missing")
//              return <-r
//          }
//
//          pub fun insert(_ id: String, _ r: @R): @R? {
//              let old <- self.rs.insert(key: id, <-r)
//              return <- old
//          }
//
//          destroy() {
//              destroy self.rs
//          }
//      }
//
//      pub fun createC(): @C {
//          return <-create C()
//      }
//  }
//`
//
//func TestRuntimeStorageDeferredResourceDictionaryValues(t *testing.T) {
//
//	runtime := NewInterpreterRuntime()
//
//	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})
//
//	contract := []byte(simpleDeferralContract)
//
//	deploy := utils.DeploymentTransaction("Test", contract)
//
//	setupTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              signer.save(<-Test.createC(), to: /storage/c)
//          }
//       }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//	var loggedMessages []string
//	var reads []testRead
//	var writes []testWrite
//
//	onRead := func(owner, key, value []byte) {
//		reads = append(reads, testRead{
//			owner,
//			key,
//		})
//	}
//
//	onWrite := func(owner, key, value []byte) {
//		writes = append(writes, testWrite{
//			owner,
//			key,
//			value,
//		})
//	}
//
//	clearReadsAndWrites := func() {
//		writes = nil
//		reads = nil
//	}
//
//	runtimeInterface := &testRuntimeInterface{
//		resolveLocation: singleIdentifierLocationResolver(t),
//		getAccountContractCode: func(_ Address, _ string) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: newTestStorage(onRead, onWrite),
//		getSigningAccounts: func() ([]Address, error) {
//			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//		log: func(message string) {
//			loggedMessages = append(loggedMessages, message)
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	clearReadsAndWrites()
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deploy,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.NotNil(t, accountCode)
//
//	assert.Len(t, writes, 1)
//
//	clearReadsAndWrites()
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Len(t, writes, 1)
//
//	// Dictionary keys should be written to separate storage keys
//
//	insertTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c = signer.borrow<&Test.C>(from: /storage/c)!
//              c.rs["a"] <-! Test.createR(1)
//              c.rs["b"] <-! Test.createR(2)
//         }
//      }
//    `)
//
//	clearReadsAndWrites()
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: insertTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	cStorageKey := []byte("storage\x1fc")
//	aStorageKey := []byte("storage\x1fc\x1frs\x1fv\x1fa")
//	bStorageKey := []byte("storage\x1fc\x1frs\x1fv\x1fb")
//
//	// writes can be out of order
//	assert.ElementsMatch(t,
//		[][]byte{
//			cStorageKey,
//			aStorageKey,
//			bStorageKey,
//		},
//		[][]byte{
//			writes[0].key,
//			writes[1].key,
//			writes[2].key,
//		},
//	)
//
//	// Reading a single key should only load that key once
//
//	readTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c = signer.borrow<&Test.C>(from: /storage/c)!
//              log(c.rs["b"]?.value)
//              log(c.rs["b"]?.value)
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: readTx},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t, []string{"2", "2"}, loggedMessages)
//
//	assert.Len(t, writes, 0)
//	require.Len(t, reads, 2)
//	assert.Equal(t,
//		cStorageKey,
//		reads[0].key,
//	)
//	assert.Equal(t,
//		bStorageKey,
//		reads[1].key,
//	)
//
//	// Updating a value of a single key should only update
//	// the single, associated storage key
//
//	updateTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c = signer.borrow<&Test.C>(from: /storage/c)!
//              c.rs["b"]?.increment()
//
//              log(c.rs["b"]?.value)
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: updateTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t, []string{"3"}, loggedMessages)
//
//	// TODO: optimize: only value has changed, dictionary itself did not
//
//	// writes can be out of order
//	assert.ElementsMatch(t,
//		[][]byte{
//			cStorageKey,
//			bStorageKey,
//		},
//		[][]byte{
//			writes[0].key,
//			writes[1].key,
//		},
//	)
//
//	require.Len(t, reads, 2)
//	assert.Equal(t,
//		cStorageKey,
//		reads[0].key,
//	)
//	assert.Equal(t,
//		bStorageKey,
//		reads[1].key,
//	)
//
//	// Replace the key with a different resource
//
//	replaceTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c = signer.borrow<&Test.C>(from: /storage/c)!
//              log(c.rs["b"]?.value)
//              let existing <- c.rs["b"] <- Test.createR(4)
//              destroy existing
//              log(c.rs["b"]?.value)
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: replaceTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t,
//		[]string{
//			"3",
//			`"destroying R"`,
//			"3",
//			"4",
//		},
//		loggedMessages,
//	)
//
//	// TODO: optimize: only value has changed, dictionary itself did not
//
//	// writes can be out of order
//	assert.ElementsMatch(t,
//		[][]byte{
//			cStorageKey,
//			bStorageKey,
//		},
//		[][]byte{
//			writes[0].key,
//			writes[1].key,
//		},
//	)
//
//	require.Len(t, reads, 3)
//	assert.Equal(t,
//		cStorageKey,
//		reads[0].key,
//	)
//	assert.Equal(t,
//		bStorageKey,
//		reads[1].key,
//	)
//	assert.Equal(t,
//		[]byte(formatContractKey("Test")),
//		reads[2].key,
//	)
//
//	// Remove the key
//
//	removeTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c = signer.borrow<&Test.C>(from: /storage/c)!
//              log(c.rs["b"]?.value)
//              let existing <- c.rs["b"] <- nil
//              destroy existing
//              log(c.rs["b"]?.value)
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: removeTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t,
//		[]string{
//			"4",
//			`"destroying R"`,
//			"4",
//			"nil",
//		},
//		loggedMessages,
//	)
//
//	// TODO: optimize: only value has changed, dictionary itself did not
//
//	// writes can be out of order
//	assert.ElementsMatch(t,
//		[][]byte{
//			cStorageKey,
//			bStorageKey,
//		},
//		[][]byte{
//			writes[0].key,
//			writes[1].key,
//		},
//	)
//
//	require.Len(t, reads, 2)
//	assert.Equal(t,
//		cStorageKey,
//		reads[0].key,
//	)
//	assert.Equal(t,
//		bStorageKey,
//		reads[1].key,
//	)
//
//	// Read the deleted key
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: readTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t, []string{"nil", "nil"}, loggedMessages)
//
//	assert.Len(t, writes, 0)
//	require.Len(t, reads, 1)
//	assert.Equal(t,
//		cStorageKey,
//		reads[0].key,
//	)
//
//	// Replace the collection
//
//	destroyTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              if let c <- signer.load<@Test.C>(from: /storage/c) {
//                  // important: read "a", so the value is in the dictionary value,
//                  // but the deferred storage key must still be removed
//                  log(c.rs["a"]?.value)
//                  destroy c
//              }
//
//              let c2 <- Test.createC()
//              c2.rs["x"] <-! Test.createR(10)
//              signer.save(<-c2, to: /storage/c)
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: destroyTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t,
//		[]string{
//			"1",
//			`"destroying R"`,
//			"1",
//		},
//		loggedMessages,
//	)
//
//	xStorageKey := []byte("storage\x1fc\x1frs\x1fv\x1fx")
//
//	// writes can be out of order
//	assert.ElementsMatch(t,
//		[][]byte{
//			cStorageKey,
//			aStorageKey,
//			xStorageKey,
//		},
//		[][]byte{
//			writes[0].key,
//			writes[1].key,
//			writes[2].key,
//		},
//	)
//
//	require.Len(t, reads, 3)
//	assert.Equal(t,
//		cStorageKey,
//		reads[0].key,
//	)
//	assert.Equal(t,
//		aStorageKey,
//		reads[1].key,
//	)
//	assert.Equal(t,
//		[]byte(formatContractKey("Test")),
//		reads[2].key,
//	)
//}
//
//func TestRuntimeStorageDeferredResourceDictionaryValues_Nested(t *testing.T) {
//
//	runtime := NewInterpreterRuntime()
//
//	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})
//
//	contract := []byte(`
//      pub contract Test {
//
//          pub resource R {
//
//              pub var value: Int
//
//              init(_ value: Int) {
//                  self.value = value
//              }
//
//              pub fun increment() {
//                  self.value = self.value + 1
//              }
//          }
//
//          pub fun createR(_ value: Int): @R {
//              return <-create R(value)
//          }
//
//          pub resource C2 {
//
//              pub let rs: @{String: R}
//
//              init() {
//                  self.rs <- {}
//              }
//
//              pub fun value(key: String): Int? {
//                  return self.rs[key]?.value
//              }
//
//              destroy() {
//                  destroy self.rs
//              }
//          }
//
//          pub fun createC2(): @C2 {
//              return <-create C2()
//          }
//
//          pub resource C {
//
//              pub let c2s: @{String: C2}
//
//              init() {
//                  self.c2s <- {}
//              }
//
//              destroy() {
//                  destroy self.c2s
//              }
//          }
//
//          pub fun createC(): @C {
//              return <-create C()
//          }
//      }
//    `)
//
//	deploy := utils.DeploymentTransaction("Test", contract)
//
//	setupTx := []byte(`
//      import Test from 0xCADE
//
//       transaction {
//
//           prepare(signer: AuthAccount) {
//               signer.save(<-Test.createC(), to: /storage/c)
//           }
//       }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//	var loggedMessages []string
//	var reads []testRead
//	var writes []testWrite
//
//	onRead := func(owner, key, value []byte) {
//		reads = append(reads, testRead{
//			owner,
//			key,
//		})
//	}
//
//	onWrite := func(owner, key, value []byte) {
//		writes = append(writes, testWrite{
//			owner,
//			key,
//			value,
//		})
//	}
//
//	clearReadsAndWrites := func() {
//		writes = nil
//		reads = nil
//	}
//
//	runtimeInterface := &testRuntimeInterface{
//		resolveLocation: singleIdentifierLocationResolver(t),
//		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
//			return accountCode, nil
//		},
//		getCode: func(_ Location) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: newTestStorage(onRead, onWrite),
//		getSigningAccounts: func() ([]Address, error) {
//			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//		log: func(message string) {
//			loggedMessages = append(loggedMessages, message)
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	clearReadsAndWrites()
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deploy,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.NotNil(t, accountCode)
//
//	assert.Len(t, writes, 1)
//
//	clearReadsAndWrites()
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Len(t, writes, 1)
//
//	// Dictionary keys should be written to separate storage keys
//
//	insertTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c = signer.borrow<&Test.C>(from: /storage/c)!
//              let c2 <- Test.createC2()
//              c2.rs["a"] <-! Test.createR(1)
//              c2.rs["b"] <-! Test.createR(2)
//              c.c2s["x"] <-! c2
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: insertTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	cStorageKey := []byte("storage\x1fc")
//	xStorageKey := []byte("storage\x1fc\x1fc2s\x1fv\x1fx")
//	aStorageKey := []byte("storage\x1fc\x1fc2s\x1fv\x1fx\x1frs\x1fv\x1fa")
//	bStorageKey := []byte("storage\x1fc\x1fc2s\x1fv\x1fx\x1frs\x1fv\x1fb")
//
//	// writes can be out of order
//	assert.ElementsMatch(t,
//		[][]byte{
//			cStorageKey,
//			xStorageKey,
//			aStorageKey,
//			bStorageKey,
//		},
//		[][]byte{
//			writes[0].key,
//			writes[1].key,
//			writes[2].key,
//			writes[3].key,
//		},
//	)
//
//	// Reading a single key should only load that key once
//
//	readTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c = signer.borrow<&Test.C>(from: /storage/c)!
//              // TODO: use nested optional chaining
//              log(c.c2s["x"]?.value(key: "b"))
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: readTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t, []string{"2"}, loggedMessages)
//
//	assert.Len(t, writes, 0)
//	require.Len(t, reads, 3)
//	assert.Equal(t,
//		cStorageKey,
//		reads[0].key,
//	)
//	assert.Equal(t,
//		xStorageKey,
//		reads[1].key,
//	)
//	assert.Equal(t,
//		bStorageKey,
//		reads[2].key,
//	)
//}
//
//func TestRuntimeStorageDeferredResourceDictionaryValues_DictionaryTransfer(t *testing.T) {
//
//	signer1 := common.BytesToAddress([]byte{0x1})
//	signer2 := common.BytesToAddress([]byte{0x2})
//
//	runtime := NewInterpreterRuntime()
//
//	contract := []byte(`
//      pub contract Test {
//
//          pub resource R {
//
//              pub var value: Int
//
//              init(_ value: Int) {
//                  self.value = value
//              }
//
//              pub fun increment() {
//                  self.value = self.value + 1
//              }
//          }
//
//          pub fun createR(_ value: Int): @R {
//              return <-create R(value)
//          }
//
//          pub resource C {
//
//              pub let rs: @{String: R}
//
//              init() {
//                  self.rs <- {}
//              }
//
//              destroy() {
//                  destroy self.rs
//              }
//          }
//
//          pub fun createC(): @C {
//              return <-create C()
//          }
//      }
//    `)
//
//	deploy := []byte(fmt.Sprintf(
//		`
//          transaction {
//
//              prepare(signer1: AuthAccount, signer2: AuthAccount) {
//                  signer1.contracts.add(name: "Test", code: "%s".decodeHex())
//              }
//          }
//        `,
//		hex.EncodeToString(contract),
//	))
//
//	setupTx := []byte(`
//      import Test from 0x1
//
//       transaction {
//
//           prepare(signer1: AuthAccount, signer2: AuthAccount) {
//               let c <- Test.createC()
//               c.rs["a"] <-! Test.createR(1)
//               c.rs["b"] <-! Test.createR(2)
//               signer1.save(<-c, to: /storage/c)
//           }
//       }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//	var loggedMessages []string
//	var reads []testRead
//	var writes []testWrite
//
//	onRead := func(owner, key, value []byte) {
//		reads = append(reads, testRead{
//			owner,
//			key,
//		})
//	}
//
//	onWrite := func(owner, key, value []byte) {
//		writes = append(writes, testWrite{
//			owner,
//			key,
//			value,
//		})
//	}
//
//	clearReadsAndWrites := func() {
//		writes = nil
//		reads = nil
//	}
//
//	indexWrites := func() (indexedWrites map[string]map[string][]byte) {
//		indexedWrites = map[string]map[string][]byte{}
//		for _, write := range writes {
//			values, ok := indexedWrites[string(write.owner)]
//			if !ok {
//				values = map[string][]byte{}
//				indexedWrites[string(write.owner)] = values
//			}
//			values[string(write.key)] = write.value
//		}
//		return
//	}
//
//	runtimeInterface := &testRuntimeInterface{
//		getCode: func(_ Location) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: newTestStorage(onRead, onWrite),
//		getSigningAccounts: func() ([]Address, error) {
//			return []Address{
//				signer1,
//				signer2,
//			}, nil
//		},
//		resolveLocation: singleIdentifierLocationResolver(t),
//		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
//			return accountCode, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) (err error) {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//		log: func(message string) {
//			loggedMessages = append(loggedMessages, message)
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	clearReadsAndWrites()
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deploy,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.NotNil(t, accountCode)
//
//	assert.Len(t, writes, 1)
//
//	clearReadsAndWrites()
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	cStorageKey := []byte("storage\x1fc")
//	aStorageKey := []byte("storage\x1fc\x1frs\x1fv\x1fa")
//	bStorageKey := []byte("storage\x1fc\x1frs\x1fv\x1fb")
//
//	// writes can be out of order
//	assert.ElementsMatch(t,
//		[][]byte{
//			cStorageKey,
//			aStorageKey,
//			bStorageKey,
//		},
//		[][]byte{
//			writes[0].key,
//			writes[1].key,
//			writes[2].key,
//		},
//	)
//
//	// Transfer
//
//	transferTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//          prepare(signer1: AuthAccount, signer2: AuthAccount) {
//              let c <- signer1.load<@Test.C>(from: /storage/c) ?? panic("missing C")
//              c.rs["x"] <-! Test.createR(42)
//              signer2.save(<-c, to: /storage/c2)
//          }
//      }
//    `)
//
//	clearReadsAndWrites()
//	loggedMessages = nil
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: transferTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	require.Len(t, writes, 7)
//
//	indexedWrites := indexWrites()
//
//	cStorageKey2 := []byte("storage\x1fc2")
//	aStorageKey2 := []byte("storage\x1fc2\x1frs\x1fv\x1fa")
//	bStorageKey2 := []byte("storage\x1fc2\x1frs\x1fv\x1fb")
//	xStorageKey2 := []byte("storage\x1fc2\x1frs\x1fv\x1fx")
//
//	assert.Empty(t, indexedWrites[string(signer1[:])][string(cStorageKey)])
//	assert.Empty(t, indexedWrites[string(signer1[:])][string(aStorageKey)])
//	assert.Empty(t, indexedWrites[string(signer1[:])][string(bStorageKey)])
//
//	assert.NotEmpty(t, indexedWrites[string(signer2[:])][string(cStorageKey2)])
//	assert.NotEmpty(t, indexedWrites[string(signer2[:])][string(aStorageKey2)])
//	assert.NotEmpty(t, indexedWrites[string(signer2[:])][string(bStorageKey2)])
//	assert.NotEmpty(t, indexedWrites[string(signer2[:])][string(xStorageKey2)])
//}
//
//func TestRuntimeStorageDeferredResourceDictionaryValues_Removal(t *testing.T) {
//
//	// Test that the `remove` function correctly loads the potentially deferred value
//
//	runtime := NewInterpreterRuntime()
//
//	contract := []byte(simpleDeferralContract)
//
//	deployTx := utils.DeploymentTransaction("Test", contract)
//
//	setupTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c <- Test.createC()
//              c.rs["a"] <-! Test.createR(1)
//              c.rs["b"] <-! Test.createR(2)
//              signer.save(<-c, to: /storage/c)
//          }
//      }
//    `)
//
//	borrowTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer: AuthAccount) {
//             let c = signer.borrow<&Test.C>(from: /storage/c)!
//             let r <- c.remove("a")
//             destroy r
//         }
//      }
//    `)
//
//	loadTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer: AuthAccount) {
//             let c <- signer.load<@Test.C>(from: /storage/c)!
//             let r <- c.remove("b")
//             destroy r
//             destroy c
//         }
//      }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//	var loggedMessages []string
//
//	signer := common.BytesToAddress([]byte{0x1})
//
//	runtimeInterface := &testRuntimeInterface{
//		getCode: func(_ Location) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: newTestStorage(nil, nil),
//		getSigningAccounts: func() ([]Address, error) {
//			return []Address{signer}, nil
//		},
//		resolveLocation: singleIdentifierLocationResolver(t),
//		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
//			return accountCode, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) (err error) {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//		log: func(message string) {
//			loggedMessages = append(loggedMessages, message)
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deployTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: borrowTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: loadTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//}
//
//func TestRuntimeStorageDeferredResourceDictionaryValues_Destruction(t *testing.T) {
//
//	// Test that the destructor is called correctly for potentially deferred value
//
//	runtime := NewInterpreterRuntime()
//
//	contract := []byte(simpleDeferralContract)
//
//	deployTx := utils.DeploymentTransaction("Test", contract)
//
//	setupTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c <- Test.createC()
//              c.rs["a"] <-! Test.createR(1)
//              c.rs["b"] <-! Test.createR(2)
//              signer.save(<-c, to: /storage/c)
//          }
//      }
//    `)
//
//	testTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer: AuthAccount) {
//             let c <- signer.load<@Test.C>(from: /storage/c)
//             destroy c
//         }
//      }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//	var loggedMessages []string
//
//	signer := common.BytesToAddress([]byte{0x1})
//
//	runtimeInterface := &testRuntimeInterface{
//		getCode: func(_ Location) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: newTestStorage(nil, nil),
//		getSigningAccounts: func() ([]Address, error) {
//			return []Address{signer}, nil
//		},
//		resolveLocation: singleIdentifierLocationResolver(t),
//		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
//			return accountCode, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//		log: func(message string) {
//			loggedMessages = append(loggedMessages, message)
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deployTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: testTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	assert.Equal(t,
//		[]string{
//			`"destroying R"`,
//			"1",
//			`"destroying R"`,
//			"2",
//		},
//		loggedMessages,
//	)
//}
//
//func TestRuntimeStorageDeferredResourceDictionaryValues_Insertion(t *testing.T) {
//
//	// Test that the `insert` function correctly loads the potentially deferred value
//
//	runtime := NewInterpreterRuntime()
//
//	contract := []byte(simpleDeferralContract)
//
//	deployTx := utils.DeploymentTransaction("Test", contract)
//
//	setupTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c <- Test.createC()
//              c.rs["a"] <-! Test.createR(1)
//              c.rs["b"] <-! Test.createR(2)
//              signer.save(<-c, to: /storage/c)
//          }
//      }
//    `)
//
//	borrowTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer: AuthAccount) {
//             let c = signer.borrow<&Test.C>(from: /storage/c)!
//
//             let e1 <- c.insert("c", <-Test.createR(3))
//             assert(e1 == nil)
//             destroy e1
//
//             let e2 <- c.insert("a", <-Test.createR(1))
//             assert(e2 != nil)
//             destroy e2
//         }
//      }
//    `)
//
//	loadTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer: AuthAccount) {
//             let c <- signer.load<@Test.C>(from: /storage/c)!
//             let e1 <- c.insert("d", <-Test.createR(4))
//             assert(e1 == nil)
//             destroy e1
//
//             let e2 <- c.insert("b", <-Test.createR(2))
//             assert(e2 != nil)
//             destroy e2
//
//             destroy c
//         }
//      }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//	var loggedMessages []string
//
//	signer := common.BytesToAddress([]byte{0x1})
//
//	runtimeInterface := &testRuntimeInterface{
//		getCode: func(_ Location) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: newTestStorage(nil, nil),
//		getSigningAccounts: func() ([]Address, error) {
//			return []Address{signer}, nil
//		},
//		resolveLocation: singleIdentifierLocationResolver(t),
//		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
//			return accountCode, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//		log: func(message string) {
//			loggedMessages = append(loggedMessages, message)
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deployTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: borrowTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: loadTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//}
//
//func TestRuntimeStorageDeferredResourceDictionaryValues_ValueTransferAndDestroy(t *testing.T) {
//
//	runtime := NewInterpreterRuntime()
//
//	contract := []byte(simpleDeferralContract)
//
//	deployTx := utils.DeploymentTransaction("Test", contract)
//
//	setupTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let c <- Test.createC()
//              signer.save(<-c, to: /storage/c)
//          }
//      }
//    `)
//
//	mintTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer: AuthAccount) {
//             let c = signer.borrow<&Test.C>(from: /storage/c)!
//
//             let existing <- c.insert("1", <-Test.createR(1))
//             assert(existing == nil)
//             destroy existing
//         }
//      }
//    `)
//
//	transferTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer1: AuthAccount, signer2: AuthAccount) {
//             let c1 = signer1.borrow<&Test.C>(from: /storage/c)!
//             let c2 = signer2.borrow<&Test.C>(from: /storage/c)!
//
//             let r <- c1.remove("1")
//
//             let existing <- c2.insert("1", <-r)
//             assert(existing == nil)
//             destroy existing
//         }
//      }
//    `)
//
//	destroyTx := []byte(`
//      import Test from 0x1
//
//      transaction {
//
//         prepare(signer: AuthAccount) {
//             let c = signer.borrow<&Test.C>(from: /storage/c)!
//
//             let r <- c.remove("1")
//             destroy r
//         }
//      }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//	var loggedMessages []string
//
//	signer1 := common.BytesToAddress([]byte{0x1})
//	signer2 := common.BytesToAddress([]byte{0x2})
//	signer3 := common.BytesToAddress([]byte{0x3})
//
//	var signers []Address
//
//	testStorage := newTestStorage(nil, nil)
//
//	runtimeInterface := &testRuntimeInterface{
//		getCode: func(_ Location) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: testStorage,
//		getSigningAccounts: func() ([]Address, error) {
//			return signers, nil
//		},
//		resolveLocation: singleIdentifierLocationResolver(t),
//		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
//			return accountCode, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//		log: func(message string) {
//			loggedMessages = append(loggedMessages, message)
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	signers = []Address{signer1}
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deployTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	signers = []Address{signer2}
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	signers = []Address{signer3}
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	signers = []Address{signer2}
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: mintTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	signers = []Address{signer2, signer3}
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: transferTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//
//	signers = []Address{signer3}
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: destroyTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(t, err)
//}
//
//func BenchmarkRuntimeStorageDeferredResourceDictionaryValues(b *testing.B) {
//
//	runtime := NewInterpreterRuntime()
//
//	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})
//
//	contract := []byte(`
//	  pub contract Test {
//
//          pub resource R {}
//
//          pub fun createR(): @R {
//              return <-create R()
//          }
//      }
//    `)
//
//	deploy := utils.DeploymentTransaction("Test", contract)
//
//	setupTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let data: @{Int: Test.R} <- {}
//              var i = 0
//              while i < 1000 {
//                  data[i] <-! Test.createR()
//                  i = i + 1
//              }
//              signer.save(<-data, to: /storage/data)
//          }
//       }
//    `)
//
//	var accountCode []byte
//	var events []cadence.Event
//
//	storage := newTestStorage(nil, nil)
//
//	runtimeInterface := &testRuntimeInterface{
//		resolveLocation: singleIdentifierLocationResolver(b),
//		getAccountContractCode: func(_ Address, _ string) (bytes []byte, err error) {
//			return accountCode, nil
//		},
//		storage: storage,
//		getSigningAccounts: func() ([]Address, error) {
//			return []Address{common.BytesToAddress(addressValue.Bytes())}, nil
//		},
//		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
//			accountCode = code
//			return nil
//		},
//		emitEvent: func(event cadence.Event) error {
//			events = append(events, event)
//			return nil
//		},
//	}
//
//	nextTransactionLocation := newTransactionLocationGenerator()
//
//	err := runtime.ExecuteTransaction(
//		Script{
//			Source: deploy,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(b, err)
//
//	assert.NotNil(b, accountCode)
//
//	err = runtime.ExecuteTransaction(
//		Script{
//			Source: setupTx,
//		},
//		Context{
//			Interface: runtimeInterface,
//			Location:  nextTransactionLocation(),
//		},
//	)
//	require.NoError(b, err)
//
//	readTx := []byte(`
//      import Test from 0xCADE
//
//      transaction {
//
//          prepare(signer: AuthAccount) {
//              let ref = signer.borrow<&{Int: Test.R}>(from: /storage/data)!
//              assert(ref[50] != nil)
//         }
//      }
//    `)
//
//	b.ReportAllocs()
//	b.ResetTimer()
//
//	for i := 0; i < b.N; i++ {
//
//		err = runtime.ExecuteTransaction(
//			Script{
//				Source: readTx,
//			},
//			Context{
//				Interface: runtimeInterface,
//				Location:  nextTransactionLocation(),
//			},
//		)
//		require.NoError(b, err)
//	}
//}
