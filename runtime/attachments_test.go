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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/tests/runtime_utils"
	. "github.com/onflow/cadence/tests/utils"
)

func TestRuntimeAccountAttachmentSaveAndLoad(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntimeWithAttachments()

	var logs []string
	var events []string
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		access(all) contract Test {
			access(all) resource R {
				access(all) fun foo(): Int {
					return 3
				}
			}
			access(all) attachment A for R {
				access(all) fun foo(): Int {
					return base.foo()
				}
			}
			access(all) fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	transaction1 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: auth(Storage) &Account) {
				let r <- Test.makeRWithA()
				signer.storage.save(<-r, to: /storage/foo)
			}
		}
	 `)

	transaction2 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: auth(Storage) &Account) {
				let r <- signer.storage.load<@Test.R>(from: /storage/foo)!
				let i = r[Test.A]!.foo()
				destroy r
				log(i)
			}
		}
	 `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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

	require.Equal(t, []string{"3"}, logs)
}

func TestRuntimeAccountAttachmentExportFailure(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntimeWithAttachments()

	logs := make([]string, 0)
	events := make([]string, 0)
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		access(all) contract Test {
			access(all) resource R {}
			access(all) attachment A for R {}
			access(all) fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	script := []byte(`
		import Test from 0x1
		access(all) fun main(): &Test.A? {
			let r <- Test.makeRWithA()
			var a = r[Test.A]

			// Life span of attachments (references) are validated statically.
			// This indirection helps to trick the checker and causes to perform the validation at runtime,
			// which is the intention of this test.
			a = returnSameRef(a)

			destroy r
			return a
		}

		access(all) fun returnSameRef(_ ref: &Test.A?): &Test.A? {
		    return ref
		}
	 `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

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

	_, err = rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextScriptLocation(),
		},
	)
	require.Error(t, err)
	require.ErrorAs(t, err, &interpreter.InvalidatedResourceReferenceError{})
}

func TestRuntimeAccountAttachmentExport(t *testing.T) {

	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntimeWithAttachments()

	var logs []string
	var events []string
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		access(all) contract Test {
			access(all) resource R {}
			access(all) attachment A for R {}
			access(all) fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	script := []byte(`
		import Test from 0x1
		access(all) fun main(): &Test.A? {
			let r <- Test.makeRWithA()
			let authAccount = getAuthAccount<auth(Storage) &Account>(0x1)
			authAccount.storage.save(<-r, to: /storage/foo)
			let ref = authAccount.storage.borrow<&Test.R>(from: /storage/foo)!
			let a = ref[Test.A]
			return a
		}
	 `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

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
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)
	require.IsType(t, cadence.Optional{}, v)
	require.IsType(t, cadence.Attachment{}, v.(cadence.Optional).Value)
	require.Equal(t, "A.0000000000000001.Test.A()", v.(cadence.Optional).Value.String())
}

func TestRuntimeAccountAttachedExport(t *testing.T) {

	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntimeWithAttachments()

	var logs []string
	var events []string
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		access(all) contract Test {
			access(all) resource R {}
			access(all) attachment A for R {}
			access(all) fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	script := []byte(`
		import Test from 0x1
		access(all) fun main(): @Test.R {
			return <-Test.makeRWithA()
		}
	 `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

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
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	require.IsType(t, cadence.Resource{}, v)
	fields := cadence.FieldsMappedByName(v.(cadence.Resource))
	require.Len(t, fields, 2)

	attachment := fields["$A.0000000000000001.Test.A"]
	require.IsType(t, cadence.Attachment{}, attachment)
	require.Equal(
		t,
		"A.0000000000000001.Test.A()",
		attachment.String(),
	)
}

func TestRuntimeAccountAttachmentSaveAndBorrow(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntimeWithAttachments()

	var logs []string
	var events []string
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		access(all) contract Test {
			access(all) resource interface I {
				access(all) fun foo(): Int
			}
			access(all) resource R: I {
				access(all) fun foo(): Int {
					return 3
				}
			}
			access(all) attachment A for I {
				access(all) fun foo(): Int {
					return base.foo()
				}
			}
			access(all) fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	transaction1 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: auth(Storage) &Account) {
				let r <- Test.makeRWithA()
				signer.storage.save(<-r, to: /storage/foo)
			}
		}
	 `)

	transaction2 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: auth(Storage) &Account) {
				let r = signer.storage.borrow<&{Test.I}>(from: /storage/foo)!
				let a: &Test.A = r[Test.A]!
				let i = a.foo()
				log(i)
			}
		}
	 `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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

	require.Equal(t, []string{"3"}, logs)
}

func TestRuntimeAccountAttachmentCapability(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntimeWithAttachments()

	var logs []string
	var events []string
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
		access(all) contract Test {
			access(all) resource interface I {
				access(all) fun foo(): Int
			}
			access(all) resource R: I {
				access(all) fun foo(): Int {
					return 3
				}
			}
			access(all) attachment A for I {
				access(all) fun foo(): Int {
					return base.foo()
				}
			}
			access(all) fun makeRWithA(): @R {
				return <- attach A() to <-create R()
			}
		}
	`))

	transaction1 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				let r <- Test.makeRWithA()
				signer.storage.save(<-r, to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&{Test.I}>(/storage/foo)!
				signer.inbox.publish(cap, name: "foo", recipient: 0x2)
			}
		}
	 `)

	transaction2 := []byte(`
		import Test from 0x1
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&{Test.I}>("foo", provider: 0x1)!
				let ref = cap.borrow()!
				let i = ref[Test.A]!.foo()
				log(i)
			}
		}
	 `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	runtimeInterface2 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

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

	require.Equal(t, []string{"3"}, logs)
}

func TestRuntimeAttachmentStorage(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	newRuntime := func() (TestInterpreterRuntime, *TestRuntimeInterface) {
		runtime := NewTestInterpreterRuntimeWithAttachments()

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

	t.Run("save and load", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		const script = `
          access(all)
          resource R {}

          access(all)
          attachment A for R {

              access(all)
              fun foo(): Int { return 3 }
          }

          access(all)
          fun main(): Int {
              let authAccount = getAuthAccount<auth(Storage) &Account>(0x1)

              let r <- create R()
              let r2 <- attach A() to <-r
              authAccount.storage.save(<-r2, to: /storage/foo)
              let r3 <- authAccount.storage.load<@R>(from: /storage/foo)!
              let i = r3[A]?.foo()!
              destroy r3
              return i
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

		assert.Equal(t, cadence.NewInt(3), result)
	})

	t.Run("save and borrow", func(t *testing.T) {
		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		const script = `
		  access(all)
	      resource R {}

		  access(all)
	      attachment A for R {

	          access(all)
	          fun foo(): Int { return 3 }
	      }

	      access(all)
          fun main(): Int {
              let authAccount = getAuthAccount<auth(Storage) &Account>(0x1)

	          let r <- create R()
	          let r2 <- attach A() to <-r
	          authAccount.storage.save(<-r2, to: /storage/foo)
	          let r3 = authAccount.storage.borrow<&R>(from: /storage/foo)!
	          return r3[A]?.foo()!
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

		assert.Equal(t, cadence.NewInt(3), result)
	})

	t.Run("capability", func(t *testing.T) {
		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		const script = `
		  access(all)
	      resource R {}

		  access(all)
	      attachment A for R {

	          access(all)
	          fun foo(): Int { return 3 }
	      }

	      access(all)
          fun main(): Int {
              let authAccount = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)
              let pubAccount = getAccount(0x1)

	          let r <- create R()
	          let r2 <- attach A() to <-r
	          authAccount.storage.save(<-r2, to: /storage/foo)
	          let cap = authAccount.capabilities.storage
                  .issue<&R>(/storage/foo)
              authAccount.capabilities.publish(cap, at: /public/foo)

	          let ref = pubAccount.capabilities.borrow<&R>(/public/foo)!
	          return ref[A]?.foo()!
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

		assert.Equal(t, cadence.NewInt(3), result)
	})

	t.Run("capability interface", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		const script = `
	      access(all)
	      resource R: I {}

	      access(all)
	      resource interface I {}

	      access(all)
	      attachment A for I {

	          access(all)
	          fun foo(): Int { return 3 }
	      }

	      access(all)
          fun main(): Int {
              let authAccount = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)
              let pubAccount = getAccount(0x1)

	          let r <- create R()
	          let r2 <- attach A() to <-r
	          authAccount.storage.save(<-r2, to: /storage/foo)
	          let cap = authAccount.capabilities.storage
                    .issue<&{I}>(/storage/foo)
              authAccount.capabilities.publish(cap, at: /public/foo)

	          let ref = pubAccount.capabilities.borrow<&{I}>(/public/foo)!
	          return ref[A]?.foo()!
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

		assert.Equal(t, cadence.NewInt(3), result)
	})
}
