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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

const resourceDictionaryContract = `
 access(all) contract Test {

     access(all) resource R {

		 access(all) event ResourceDestroyed(value: Int = self.value)

         access(all) var value: Int

         init(_ value: Int) {
             self.value = value
         }

         access(all) fun increment() {
             self.value = self.value + 1
         }
     }

     access(all) fun createR(_ value: Int): @R {
         return <-create R(value)
     }

     access(all) resource C {

         access(all) var rs: @{String: R}

         init() {
             self.rs <- {}
         }

         access(all) fun remove(_ id: String): @R {
             let r <- self.rs.remove(key: id) ?? panic("missing")
             return <-r
         }

         access(all) fun insert(_ id: String, _ r: @R): @R? {
             let old <- self.rs.insert(key: id, <-r)
             return <- old
         }

		 access(all) fun forceInsert(_ id: String, _ r: @R) {
			self.rs[id] <-! r
		 }
     }

     access(all) fun createC(): @C {
         return <-create C()
     }
 }
`

func TestRuntimeResourceDictionaryValues(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(resourceDictionaryContract)

	deploy := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             signer.storage.save(<-Test.createC(), to: /storage/c)
         }
      }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
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

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

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

	insertTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
             c.forceInsert("a", <- Test.createR(1))
             c.forceInsert("b", <- Test.createR(2))
        }
     }
   `)

	err = runtime.ExecuteTransaction(
		Script{
			Source: insertTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	readTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
             log(c.rs["b"]?.value)
             log(c.rs["b"]?.value)
         }
     }
   `)

	loggedMessages = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: readTx},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"2", "2"}, loggedMessages)

	updateTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
             c.rs["b"]?.increment()

             log(c.rs["b"]?.value)
         }
     }
   `)

	loggedMessages = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: updateTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"3"}, loggedMessages)

	// Replace the key with a different resource

	replaceTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
             log(c.rs["b"]?.value)
			 destroy c.remove("b")
			 c.forceInsert("b", <- Test.createR(4))
             log(c.rs["b"]?.value)
         }
     }
   `)

	loggedMessages = nil
	events = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: replaceTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"3",
			"4",
		},
		loggedMessages,
	)
	require.Len(t, events, 1)
	require.Equal(t, "A.000000000000cade.Test.R.ResourceDestroyed(value: 3)", events[0].String())

	// Remove the key

	removeTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
             log(c.rs["b"]?.value)
			 destroy c.remove("b")
             log(c.rs["b"]?.value)
         }
     }
   `)

	loggedMessages = nil
	events = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: removeTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"4",
			"nil",
		},
		loggedMessages,
	)
	require.Len(t, events, 1)
	require.Equal(t, "A.000000000000cade.Test.R.ResourceDestroyed(value: 4)", events[0].String())

	// Read the deleted key

	loggedMessages = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: readTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"nil", "nil"}, loggedMessages)

	// Replace the collection

	destroyTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             if let c <- signer.storage.load<@Test.C>(from: /storage/c) {
                 log(c.rs["a"]?.value)
                 destroy c
             }

             let c2 <- Test.createC()
			 c2.forceInsert("x", <-Test.createR(10))
             signer.storage.save(<-c2, to: /storage/c)
         }
     }
   `)

	loggedMessages = nil
	events = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: destroyTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]string{
			"1",
		},
		loggedMessages,
	)
	require.Len(t, events, 1)
	require.Equal(t, "A.000000000000cade.Test.R.ResourceDestroyed(value: 1)", events[0].String())

}

func TestRuntimeResourceDictionaryValues_Nested(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
     access(all) contract Test {

         access(all) resource R {

             access(all) var value: Int

             init(_ value: Int) {
                 self.value = value
             }

             access(all) fun increment() {
                 self.value = self.value + 1
             }
         }

         access(all) fun createR(_ value: Int): @R {
             return <-create R(value)
         }

         access(all) resource C2 {

             access(all) var rs: @{String: R}

             init() {
                 self.rs <- {}
             }

             access(all) fun value(key: String): Int? {
                 return self.rs[key]?.value
             }

			 access(all) fun forceInsert(_ id: String, _ r: @R) {
				self.rs[id] <-! r
			 }
         }

         access(all) fun createC2(): @C2 {
             return <-create C2()
         }

         access(all) resource C {

             access(all) var c2s: @{String: C2}

			 access(all) fun forceInsert(_ id: String, _ c: @C2) {
				self.c2s[id] <-! c
			 }

             init() {
                 self.c2s <- {}
             }
         }

         access(all) fun createC(): @C {
             return <-create C()
         }
     }
   `)

	deploy := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0xCADE

      transaction {

          prepare(signer: auth(Storage) &Account) {
              signer.storage.save(
                  <-Test.createC(),
                  to: /storage/c
              )
          }
      }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
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

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

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

	insertTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
             let c2 <- Test.createC2()
             c2.forceInsert("a", <- Test.createR(1))
			 c2.forceInsert("b", <- Test.createR(2))
			 c.forceInsert("x", <- c2)
         }
     }
   `)

	err = runtime.ExecuteTransaction(
		Script{
			Source: insertTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	readTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
             // TODO: use nested optional chaining
             log(c.c2s["x"]?.value(key: "b"))
         }
     }
   `)

	loggedMessages = nil

	err = runtime.ExecuteTransaction(
		Script{
			Source: readTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"2"}, loggedMessages)
}

func TestRuntimeResourceDictionaryValues_DictionaryTransfer(t *testing.T) {

	t.Parallel()

	signer1 := common.MustBytesToAddress([]byte{0x1})
	signer2 := common.MustBytesToAddress([]byte{0x2})

	runtime := NewTestInterpreterRuntime()

	contract := []byte(`
     access(all) contract Test {

         access(all) resource R {

             access(all) var value: Int

             init(_ value: Int) {
                 self.value = value
             }

             access(all) fun increment() {
                 self.value = self.value + 1
             }
         }

         access(all) fun createR(_ value: Int): @R {
             return <-create R(value)
         }

         access(all) resource C {

             access(all) var rs: @{String: R}

			 access(all) fun setRs(key s: String, r: @R) {
				self.rs[s] <-! r
			 }

             init() {
                 self.rs <- {}
             }
         }

         access(all) fun createC(): @C {
             return <-create C()
         }
     }
   `)

	deploy := []byte(fmt.Sprintf(
		`
         transaction {

             prepare(
                 signer1: auth(Contracts) &Account,
                 signer2: &Account
             ) {
                 signer1.contracts.add(name: "Test", code: "%s".decodeHex())
             }
         }
       `,
		hex.EncodeToString(contract),
	))

	setupTx := []byte(`
     import Test from 0x1

      transaction {

          prepare(
              signer1: auth(Storage) &Account,
              signer2: &Account
          ) {
              let c <- Test.createC()
              c.setRs(key: "a", r: <- Test.createR(1))
			  c.setRs(key: "b", r: <- Test.createR(2))
              signer1.storage.save(<-c, to: /storage/c)
          }
      }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{
				signer1,
				signer2,
			}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) (err error) {
			accountCode = code
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

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	assert.NotNil(t, accountCode)

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

	// Transfer

	transferTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(
             signer1: auth(Storage) &Account,
             signer2: auth(Storage) &Account
         ) {
             let c <- signer1.storage.load<@Test.C>(from: /storage/c) ?? panic("missing C")
             c.setRs(key: "x", r: <- Test.createR(42))
             signer2.storage.save(<-c, to: /storage/c2)
         }
     }
   `)

	loggedMessages = nil

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
}

func TestRuntimeResourceDictionaryValues_Removal(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c <- Test.createC()
			 c.forceInsert("a", <- Test.createR(1))
             c.forceInsert("b", <- Test.createR(2))
             signer.storage.save(<-c, to: /storage/c)
         }
     }
   `)

	borrowTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: auth(Storage) &Account) {
            let c = signer.storage.borrow<&Test.C>(from: /storage/c)!
            let r <- c.remove("a")
            destroy r
        }
     }
   `)

	loadTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: auth(Storage) &Account) {
            let c <- signer.storage.load<@Test.C>(from: /storage/c)!
            let r <- c.remove("b")
            destroy r
            destroy c
        }
     }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	signer := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) (err error) {
			accountCode = code
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

	err = runtime.ExecuteTransaction(
		Script{
			Source: borrowTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: loadTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeResourceDictionaryValues_Destruction(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c <- Test.createC()
             c.forceInsert("a", <- Test.createR(1))
             c.forceInsert("b", <- Test.createR(2))
             signer.storage.save(<-c, to: /storage/c)
         }
     }
   `)

	testTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: auth(Storage) &Account) {
            let c <- signer.storage.load<@Test.C>(from: /storage/c)
            destroy c
        }
     }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	signer := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
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

	err = runtime.ExecuteTransaction(
		Script{
			Source: testTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Len(t, events, 3)
	require.Equal(t, "flow.AccountContractAdded", events[0].EventType.ID())
	require.Equal(t, "A.0000000000000001.Test.R.ResourceDestroyed", events[1].EventType.ID())
	require.Equal(t, "A.0000000000000001.Test.R.ResourceDestroyed", events[2].EventType.ID())
	event2Fields := cadence.FieldsMappedByName(events[1])
	event3Fields := cadence.FieldsMappedByName(events[2])
	require.Equal(t, "2", event2Fields["value"].String())
	require.Equal(t, "1", event3Fields["value"].String())
}

func TestRuntimeResourceDictionaryValues_Insertion(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c <- Test.createC()
			 c.forceInsert("a", <- Test.createR(1))
             c.forceInsert("b", <- Test.createR(2))
             signer.storage.save(<-c, to: /storage/c)
         }
     }
   `)

	borrowTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: auth(Storage) &Account) {
            let c = signer.storage.borrow<&Test.C>(from: /storage/c)!

            let e1 <- c.insert("c", <-Test.createR(3))
            assert(e1 == nil)
            destroy e1

            let e2 <- c.insert("a", <-Test.createR(1))
            assert(e2 != nil)
            destroy e2
        }
     }
   `)

	loadTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: auth(Storage) &Account) {
            let c <- signer.storage.load<@Test.C>(from: /storage/c)!
            let e1 <- c.insert("d", <-Test.createR(4))
            assert(e1 == nil)
            destroy e1

            let e2 <- c.insert("b", <-Test.createR(2))
            assert(e2 != nil)
            destroy e2

            destroy c
        }
     }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	signer := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
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

	err = runtime.ExecuteTransaction(
		Script{
			Source: borrowTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = runtime.ExecuteTransaction(
		Script{
			Source: loadTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func TestRuntimeResourceDictionaryValues_ValueTransferAndDestroy(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: auth(Storage) &Account) {
             let c <- Test.createC()
             signer.storage.save(<-c, to: /storage/c)
         }
     }
   `)

	mintTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: auth(Storage) &Account) {
            let c = signer.storage.borrow<&Test.C>(from: /storage/c)!

            let existing <- c.insert("1", <-Test.createR(1))
            assert(existing == nil)
            destroy existing
        }
     }
   `)

	transferTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(
            signer1: auth(Storage) &Account,
            signer2: auth(Storage) &Account
        ) {
            let c1 = signer1.storage.borrow<&Test.C>(from: /storage/c)!
            let c2 = signer2.storage.borrow<&Test.C>(from: /storage/c)!

            let r <- c1.remove("1")

            let existing <- c2.insert("1", <-r)
            assert(existing == nil)
            destroy existing
        }
     }
   `)

	destroyTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: auth(Storage) &Account) {
            let c = signer.storage.borrow<&Test.C>(from: /storage/c)!

            let r <- c.remove("1")
            destroy r
        }
     }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	signer1 := common.MustBytesToAddress([]byte{0x1})
	signer2 := common.MustBytesToAddress([]byte{0x2})
	signer3 := common.MustBytesToAddress([]byte{0x3})

	var signers []Address

	testStorage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: testStorage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
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

	signers = []Address{signer1}
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

	signers = []Address{signer2}
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

	signers = []Address{signer3}
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

	signers = []Address{signer2}
	err = runtime.ExecuteTransaction(
		Script{
			Source: mintTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	signers = []Address{signer2, signer3}
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

	signers = []Address{signer3}
	err = runtime.ExecuteTransaction(
		Script{
			Source: destroyTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)
}

func BenchmarkRuntimeResourceDictionaryValues(b *testing.B) {

	runtime := NewTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
	  access(all) contract Test {

         access(all) resource R {}

         access(all) fun createR(): @R {
             return <-create R()
         }
     }
   `)

	deploy := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: &Account) {
             let data: @{Int: Test.R} <- {}
             var i = 0
             while i < 1000 {
                 data[i] <-! Test.createR()
                 i = i + 1
             }
             signer.storage.save(<-data, to: /storage/data)
         }
      }
   `)

	var accountCode []byte
	var events []cadence.Event

	storage := NewTestLedger(nil, nil)

	runtimeInterface := &TestRuntimeInterface{
		OnResolveLocation: NewSingleIdentifierLocationResolver(b),
		OnGetAccountContractCode: func(_ common.AddressLocation) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: storage,
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(b, err)

	assert.NotNil(b, accountCode)

	err = runtime.ExecuteTransaction(
		Script{
			Source: setupTx,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(b, err)

	readTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: &Account) {
             let ref = signer.storage.borrow<&{Int: Test.R}>(from: /storage/data)!
             assert(ref[50] != nil)
        }
     }
   `)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		err = runtime.ExecuteTransaction(
			Script{
				Source: readTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(b, err)
	}
}
