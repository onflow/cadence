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
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

const resourceDictionaryContract = `
 pub contract Test {

     pub resource R {

         pub var value: Int

         init(_ value: Int) {
             self.value = value
         }

         pub fun increment() {
             self.value = self.value + 1
         }

         destroy() {
             log("destroying R")
             log(self.value)
         }
     }

     pub fun createR(_ value: Int): @R {
         return <-create R(value)
     }

     pub resource C {

         pub(set) var rs: @{String: R}

         init() {
             self.rs <- {}
         }

         pub fun remove(_ id: String): @R {
             let r <- self.rs.remove(key: id) ?? panic("missing")
             return <-r
         }

         pub fun insert(_ id: String, _ r: @R): @R? {
             let old <- self.rs.insert(key: id, <-r)
             return <- old
         }

         destroy() {
             destroy self.rs
         }
     }

     pub fun createC(): @C {
         return <-create C()
     }
 }
`

func TestRuntimeResourceDictionaryValues(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(resourceDictionaryContract)

	deploy := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: AuthAccount) {
             signer.save(<-Test.createC(), to: /storage/c)
         }
      }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
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

         prepare(signer: AuthAccount) {
             let c = signer.borrow<&Test.C>(from: /storage/c)!
             c.rs["a"] <-! Test.createR(1)
             c.rs["b"] <-! Test.createR(2)
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

         prepare(signer: AuthAccount) {
             let c = signer.borrow<&Test.C>(from: /storage/c)!
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

         prepare(signer: AuthAccount) {
             let c = signer.borrow<&Test.C>(from: /storage/c)!
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

         prepare(signer: AuthAccount) {
             let c = signer.borrow<&Test.C>(from: /storage/c)!
             log(c.rs["b"]?.value)
             let existing <- c.rs["b"] <- Test.createR(4)
             destroy existing
             log(c.rs["b"]?.value)
         }
     }
   `)

	loggedMessages = nil

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
			`"destroying R"`,
			"3",
			"4",
		},
		loggedMessages,
	)

	// Remove the key

	removeTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: AuthAccount) {
             let c = signer.borrow<&Test.C>(from: /storage/c)!
             log(c.rs["b"]?.value)
             let existing <- c.rs["b"] <- nil
             destroy existing
             log(c.rs["b"]?.value)
         }
     }
   `)

	loggedMessages = nil

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
			`"destroying R"`,
			"4",
			"nil",
		},
		loggedMessages,
	)

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

         prepare(signer: AuthAccount) {
             if let c <- signer.load<@Test.C>(from: /storage/c) {
                 log(c.rs["a"]?.value)
                 destroy c
             }

             let c2 <- Test.createC()
             c2.rs["x"] <-! Test.createR(10)
             signer.save(<-c2, to: /storage/c)
         }
     }
   `)

	loggedMessages = nil

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
			`"destroying R"`,
			"1",
		},
		loggedMessages,
	)
}

func TestRuntimeResourceDictionaryValues_Nested(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
     pub contract Test {

         pub resource R {

             pub var value: Int

             init(_ value: Int) {
                 self.value = value
             }

             pub fun increment() {
                 self.value = self.value + 1
             }
         }

         pub fun createR(_ value: Int): @R {
             return <-create R(value)
         }

         pub resource C2 {

             pub(set) var rs: @{String: R}

             init() {
                 self.rs <- {}
             }

             pub fun value(key: String): Int? {
                 return self.rs[key]?.value
             }

             destroy() {
                 destroy self.rs
             }
         }

         pub fun createC2(): @C2 {
             return <-create C2()
         }

         pub resource C {

             pub(set) var c2s: @{String: C2}

             init() {
                 self.c2s <- {}
             }

             destroy() {
                 destroy self.c2s
             }
         }

         pub fun createC(): @C {
             return <-create C()
         }
     }
   `)

	deploy := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0xCADE

      transaction {

          prepare(signer: AuthAccount) {
              signer.save(<-Test.createC(), to: /storage/c)
          }
      }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
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

         prepare(signer: AuthAccount) {
             let c = signer.borrow<&Test.C>(from: /storage/c)!
             let c2 <- Test.createC2()
             c2.rs["a"] <-! Test.createR(1)
             c2.rs["b"] <-! Test.createR(2)
             c.c2s["x"] <-! c2
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

         prepare(signer: AuthAccount) {
             let c = signer.borrow<&Test.C>(from: /storage/c)!
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

	runtime := newTestInterpreterRuntime()

	contract := []byte(`
     pub contract Test {

         pub resource R {

             pub var value: Int

             init(_ value: Int) {
                 self.value = value
             }

             pub fun increment() {
                 self.value = self.value + 1
             }
         }

         pub fun createR(_ value: Int): @R {
             return <-create R(value)
         }

         pub resource C {

             pub(set) var rs: @{String: R}

             init() {
                 self.rs <- {}
             }

             destroy() {
                 destroy self.rs
             }
         }

         pub fun createC(): @C {
             return <-create C()
         }
     }
   `)

	deploy := []byte(fmt.Sprintf(
		`
         transaction {

             prepare(signer1: AuthAccount, signer2: AuthAccount) {
                 signer1.contracts.add(name: "Test", code: "%s".decodeHex())
             }
         }
       `,
		hex.EncodeToString(contract),
	))

	setupTx := []byte(`
     import Test from 0x1

      transaction {

          prepare(signer1: AuthAccount, signer2: AuthAccount) {
              let c <- Test.createC()
              c.rs["a"] <-! Test.createR(1)
              c.rs["b"] <-! Test.createR(2)
              signer1.save(<-c, to: /storage/c)
          }
      }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{
				signer1,
				signer2,
			}, nil
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
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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

         prepare(signer1: AuthAccount, signer2: AuthAccount) {
             let c <- signer1.load<@Test.C>(from: /storage/c) ?? panic("missing C")
             c.rs["x"] <-! Test.createR(42)
             signer2.save(<-c, to: /storage/c2)
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

	runtime := newTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: AuthAccount) {
             let c <- Test.createC()
             c.rs["a"] <-! Test.createR(1)
             c.rs["b"] <-! Test.createR(2)
             signer.save(<-c, to: /storage/c)
         }
     }
   `)

	borrowTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: AuthAccount) {
            let c = signer.borrow<&Test.C>(from: /storage/c)!
            let r <- c.remove("a")
            destroy r
        }
     }
   `)

	loadTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: AuthAccount) {
            let c <- signer.load<@Test.C>(from: /storage/c)!
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

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
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
		log: func(message string) {
			loggedMessages = append(loggedMessages, message)
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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

func TestRuntimeSResourceDictionaryValues_Destruction(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: AuthAccount) {
             let c <- Test.createC()
             c.rs["a"] <-! Test.createR(1)
             c.rs["b"] <-! Test.createR(2)
             signer.save(<-c, to: /storage/c)
         }
     }
   `)

	testTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: AuthAccount) {
            let c <- signer.load<@Test.C>(from: /storage/c)
            destroy c
        }
     }
   `)

	var accountCode []byte
	var events []cadence.Event
	var loggedMessages []string

	signer := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
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

	assert.Equal(t,
		[]string{
			`"destroying R"`,
			"1",
			`"destroying R"`,
			"2",
		},
		loggedMessages,
	)
}

func TestRuntimeResourceDictionaryValues_Insertion(t *testing.T) {

	t.Parallel()

	runtime := newTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: AuthAccount) {
             let c <- Test.createC()
             c.rs["a"] <-! Test.createR(1)
             c.rs["b"] <-! Test.createR(2)
             signer.save(<-c, to: /storage/c)
         }
     }
   `)

	borrowTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: AuthAccount) {
            let c = signer.borrow<&Test.C>(from: /storage/c)!

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

        prepare(signer: AuthAccount) {
            let c <- signer.load<@Test.C>(from: /storage/c)!
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

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: newTestLedger(nil, nil),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{signer}, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
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

	runtime := newTestInterpreterRuntime()

	contract := []byte(resourceDictionaryContract)

	deployTx := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0x1

     transaction {

         prepare(signer: AuthAccount) {
             let c <- Test.createC()
             signer.save(<-c, to: /storage/c)
         }
     }
   `)

	mintTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer: AuthAccount) {
            let c = signer.borrow<&Test.C>(from: /storage/c)!

            let existing <- c.insert("1", <-Test.createR(1))
            assert(existing == nil)
            destroy existing
        }
     }
   `)

	transferTx := []byte(`
     import Test from 0x1

     transaction {

        prepare(signer1: AuthAccount, signer2: AuthAccount) {
            let c1 = signer1.borrow<&Test.C>(from: /storage/c)!
            let c2 = signer2.borrow<&Test.C>(from: /storage/c)!

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

        prepare(signer: AuthAccount) {
            let c = signer.borrow<&Test.C>(from: /storage/c)!

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

	testStorage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		getCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: testStorage,
		getSigningAccounts: func() ([]Address, error) {
			return signers, nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getAccountContractCode: func(_ Address, _ string) (code []byte, err error) {
			return accountCode, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
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

	runtime := newTestInterpreterRuntime()

	addressValue := cadence.BytesToAddress([]byte{0xCA, 0xDE})

	contract := []byte(`
	  pub contract Test {

         pub resource R {}

         pub fun createR(): @R {
             return <-create R()
         }
     }
   `)

	deploy := utils.DeploymentTransaction("Test", contract)

	setupTx := []byte(`
     import Test from 0xCADE

     transaction {

         prepare(signer: AuthAccount) {
             let data: @{Int: Test.R} <- {}
             var i = 0
             while i < 1000 {
                 data[i] <-! Test.createR()
                 i = i + 1
             }
             signer.save(<-data, to: /storage/data)
         }
      }
   `)

	var accountCode []byte
	var events []cadence.Event

	storage := newTestLedger(nil, nil)

	runtimeInterface := &testRuntimeInterface{
		resolveLocation: singleIdentifierLocationResolver(b),
		getAccountContractCode: func(_ Address, _ string) (bytes []byte, err error) {
			return accountCode, nil
		},
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{Address(addressValue)}, nil
		},
		updateAccountContractCode: func(_ Address, _ string, code []byte) error {
			accountCode = code
			return nil
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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

         prepare(signer: AuthAccount) {
             let ref = signer.borrow<&{Int: Test.R}>(from: /storage/data)!
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
