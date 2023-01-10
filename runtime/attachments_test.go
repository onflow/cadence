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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestAccountAttachmentSaveAndLoad(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)
	events := make([]string, 0)
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		pub contract Test {
			pub resource R {
				pub fun foo(): Int {
					return 3
				}
			}
			pub attachment A for R {
				pub fun foo(): Int {
					return base.foo()
				}
			}
			pub fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	transaction1 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: AuthAccount) {
				let r <- Test.makeRWithA()
				signer.save(<-r, to: /storage/foo)
			}
		}
	 `)

	transaction2 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: AuthAccount) {
				let r <- signer.load<@Test.R>(from: /storage/foo)!
				let i = r[Test.A]!.foo()
				destroy r
				log(i)
			}
		}
	 `)

	runtimeInterface1 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
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
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction2,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t, logs[0], "3")
}

func TestAccountAttachmentExport(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)
	events := make([]string, 0)
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		pub contract Test {
			pub resource R {}
			pub attachment A for R {}
			pub fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	script := []byte(`
		import Test from 0x1
		pub fun main(): &Test.A? { 
			let r <- Test.makeRWithA()
			let a = r[Test.A]
			destroy r
			return a
		}
	 `)

	runtimeInterface1 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
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
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	v, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.IsType(t, cadence.Optional{}, v)
	require.IsType(t, cadence.Attachment{}, v.(cadence.Optional).Value)
	require.Equal(t, "A.0000000000000001.Test.A()", v.(cadence.Optional).Value.String())

	require.NoError(t, err)
}

func TestAccountAttachedExport(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)
	events := make([]string, 0)
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		pub contract Test {
			pub resource R {}
			pub attachment A for R {}
			pub fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	script := []byte(`
		import Test from 0x1
		pub fun main(): @Test.R { 
			return <-Test.makeRWithA()
		}
	 `)

	runtimeInterface1 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
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
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	v, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.IsType(t, cadence.Resource{}, v)
	require.Len(t, v.(cadence.Resource).Fields, 2)
	require.IsType(t, cadence.Attachment{}, v.(cadence.Resource).Fields[1])
	require.Equal(t, "A.0000000000000001.Test.A()", v.(cadence.Resource).Fields[1].String())

	require.NoError(t, err)
}

func TestAccountAttachmentSaveAndBorrow(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)
	events := make([]string, 0)
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		pub contract Test {
			pub resource interface I {
				pub fun foo(): Int
			}
			pub resource R: I {
				pub fun foo(): Int {
					return 3
				}
			}
			pub attachment A for I {
				pub fun foo(): Int {
					return base.foo()
				}
			}
			pub fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	transaction1 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: AuthAccount) {
				let r <- Test.makeRWithA()
				signer.save(<-r, to: /storage/foo)
			}
		}
	 `)

	transaction2 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: AuthAccount) {
				let r = signer.borrow<&{Test.I}>(from: /storage/foo)!
				let a: &Test.A = r[Test.A]!
				let i = a.foo()
				log(i)
			}
		}
	 `)

	runtimeInterface1 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
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
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction2,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t, logs[0], "3")
}

func TestAccountAttachmentCapability(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)
	events := make([]string, 0)
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		pub contract Test {
			pub resource interface I {
				pub fun foo(): Int
			}
			pub resource R: I {
				pub fun foo(): Int {
					return 3
				}
			}
			pub attachment A for I {
				pub fun foo(): Int {
					return base.foo()
				}
			}
			pub fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	transaction1 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: AuthAccount) {
				let r <- Test.makeRWithA()
				signer.save(<-r, to: /storage/foo)
				let cap = signer.link<&{Test.I}>(/public/foo, target: /storage/foo)!
				signer.inbox.publish(cap, name: "foo", recipient: 0x2)
			}
		}
	 `)

	transaction2 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&{Test.I}>("foo", provider: 0x1)!
				let ref = cap.borrow()!
				let i = ref[Test.A]!.foo()
				log(i)
			}
		}
	 `)

	runtimeInterface1 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
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
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		resolveLocation: singleIdentifierLocationResolver(t),
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
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
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		Script{
			Source: deployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: transaction2,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t, logs[0], "3")
}
