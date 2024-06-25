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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
)

func TestRuntimeAccountInboxPublishUnpublish(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.unpublish<&[Int]>("foo")!
				log(cap.borrow()![0])
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// unpublish from 1
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

	require.Equal(t,
		[]string{
			// successful publish
			"()",
			// correct value returned from unpublish
			"3",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
			`flow.InboxValueUnpublished(provider: 0x0000000000000001, name: "foo")`,
		},
		events,
	)
}

func TestRuntimeAccountInboxUnpublishWrongType(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				signer.inbox.publish(cap, name: "foo", recipient: 0x2)
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.unpublish<&[String]>("foo")!
				log(cap.borrow()![0])
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// unpublish from 1
	err = rt.ExecuteTransaction(
		Script{
			Source: transaction2,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to force-cast value: expected type `Capability<&[String]>`, got `Capability<&[Int]>`")

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
		},
		events,
	)
}

func TestRuntimeAccountInboxUnpublishAbsent(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.unpublish<&[Int]>("bar")
				log(cap)
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// unpublish from 1
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

	require.Equal(t,
		[]string{
			// successful publish
			"()",

			// correct value returned from unpublish
			"nil",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
		},
		events,
	)
}

func TestRuntimeAccountInboxUnpublishRemove(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction(name: String) {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: name, recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction(name: String) {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.unpublish<&[Int]>(name)!
				log(cap.borrow()![0])
				let cap2 = signer.inbox.unpublish<&[Int]>(name)
				log(cap2)
			}
		}
	`)

	address := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
	}

	// NOTE: generate a long name
	nameArgument, err := cadence.NewString(strings.Repeat("x", 10_000))
	require.NoError(t, err)

	args := encodeArgs(nameArgument)

	nextTransactionLocation := NewTransactionLocationGenerator()
	// publish from 1 to 2
	err = rt.ExecuteTransaction(
		Script{
			Source:    transaction1,
			Arguments: args,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// unpublish from 1

	err = rt.ExecuteTransaction(
		Script{
			Source:    transaction2,
			Arguments: args,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		[]string{
			// successful publish
			"()",

			// correct value returned from unpublish
			"3",

			// unpublish successfully removes the value
			"nil",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: ` +
				nameArgument.String() +
				`, type: Type<Capability<&[Int]>>())`,
			`flow.InboxValueUnpublished(provider: 0x0000000000000001, name: ` +
				nameArgument.String() +
				`)`,
		},
		events,
	)
}

func TestRuntimeAccountInboxUnpublishWrongAccount(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction1point5 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.unpublish<&[Int]>("foo")
				log(cap)
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.unpublish<&[Int]>("foo")!
				log(cap.borrow()![0])
			}
		}
	`)

	address1 := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address1}, nil
		},
	}

	address2 := common.MustBytesToAddress([]byte{0x2})

	runtimeInterface2 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address2}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// attempt to unpublish from 2
	err = rt.ExecuteTransaction(
		Script{
			Source: transaction1point5,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// unpublish from 1
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

	require.Equal(t,
		[]string{
			// successful publish
			"()",
			// unpublish not successful from wrong account
			"nil",
			// correct value returned from unpublish
			"3",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
			`flow.InboxValueUnpublished(provider: 0x0000000000000001, name: "foo")`,
		},
		events,
	)
}

func TestRuntimeAccountInboxPublishClaim(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
			}
		}
	`)

	address1 := common.MustBytesToAddress([]byte{0x1})

	runtimeInterface1 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address1}, nil
		},
	}

	address2 := common.MustBytesToAddress([]byte{0x2})

	runtimeInterface2 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {

			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address2}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// claim from 2
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

	require.Equal(t,
		[]string{
			// successful publish
			"()",

			// correct value returned from claim
			"3",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
			`flow.InboxValueClaimed(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo")`,
		},
		events,
	)
}

func TestRuntimeAccountInboxPublishClaimWrongType(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&[String]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// claim from 2
	err = rt.ExecuteTransaction(
		Script{
			Source: transaction2,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.Error(t, err)
	assert.ErrorContains(t, err, "failed to force-cast value: expected type `Capability<&[String]>`, got `Capability<&[Int]>`")

	require.Equal(t,
		[]string{
			// successful publish
			"()",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
		},
		events,
	)
}

func TestRuntimeAccountInboxPublishClaimWrongName(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&[String]>("bar", provider: 0x1)
				log(cap)
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// claim from 2
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

	require.Equal(t,
		[]string{
			// successful publish
			"()",
			// no value claimed
			"nil",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
		},
		events,
	)
}

func TestRuntimeAccountInboxPublishClaimRemove(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
			}
		}
	`)

	transaction3 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)
				log(cap)
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// claim from 2
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

	// claim from 2 again
	err = rt.ExecuteTransaction(
		Script{
			Source: transaction3,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	require.Equal(t,
		[]string{
			// successful publish
			"()",
			// correct value returned from claim
			"3",
			// claimed value properly removed
			"nil",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
			`flow.InboxValueClaimed(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo")`,
		},
		events,
	)
}

func TestRuntimeAccountInboxPublishClaimWrongAccount(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: auth(Storage, Capabilities, Inbox) &Account) {
				signer.storage.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)
				log(cap)
			}
		}
	`)

	transaction3 := []byte(`
		transaction {
			prepare(signer: auth(Inbox) &Account) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	runtimeInterface3 := &TestRuntimeInterface{
		Storage: storage,
		OnProgramLog: func(message string) {
			logs = append(logs, message)
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 3}}, nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// publish from 1 to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction1,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// claim from 3
	err = rt.ExecuteTransaction(
		Script{
			Source: transaction2,
		},
		Context{
			Interface: runtimeInterface3,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// claim from 2
	err = rt.ExecuteTransaction(
		Script{
			Source: transaction3,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	require.Equal(t,
		[]string{
			// successful publish
			"()",
			// value is not claimed by 3
			"nil",
			// value is claimed by 2
			"3",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			`flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo", type: Type<Capability<&[Int]>>())`,
			`flow.InboxValueClaimed(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: "foo")`,
		},
		events,
	)
}
