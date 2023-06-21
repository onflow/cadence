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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
)

func TestAccountInboxPublishUnpublish(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.unpublish<&[Int]>("foo")!
				log(cap.borrow()![0])
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
			"flow.InboxValueUnpublished(provider: 0x0000000000000001, name: \"foo\")",
		},
		events,
	)
}

func TestAccountInboxUnpublishWrongType(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				signer.inbox.publish(cap, name: "foo", recipient: 0x2)
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.unpublish<&[String]>("foo")!
				log(cap.borrow()![0])
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
		},
		events,
	)
}

func TestAccountInboxUnpublishAbsent(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.unpublish<&[Int]>("bar")
				log(cap)
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
		},
		events,
	)
}

func TestAccountInboxUnpublishRemove(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.unpublish<&[Int]>("foo")!
				log(cap.borrow()![0])
				let cap2 = signer.inbox.unpublish<&[Int]>("foo")
				log(cap2)
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()
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

			// unpublish successfully removes the value
			"nil",
		},
		logs,
	)

	require.Equal(t,
		[]string{
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
			"flow.InboxValueUnpublished(provider: 0x0000000000000001, name: \"foo\")",
		},
		events,
	)
}

func TestAccountInboxUnpublishWrongAccount(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction1point5 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.unpublish<&[Int]>("foo")
				log(cap)
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.unpublish<&[Int]>("foo")!
				log(cap.borrow()![0])
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
			"flow.InboxValueUnpublished(provider: 0x0000000000000001, name: \"foo\")",
		},
		events,
	)
}

func TestAccountInboxPublishClaim(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
			"flow.InboxValueClaimed(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\")",
		},
		events,
	)
}

func TestAccountInboxPublishClaimWrongType(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&[String]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
		},
		events,
	)
}

func TestAccountInboxPublishClaimWrongName(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&[String]>("bar", provider: 0x1)
				log(cap)
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
		},
		events,
	)
}

func TestAccountInboxPublishClaimRemove(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
			}
		}
	`)

	transaction3 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)
				log(cap)
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
			"flow.InboxValueClaimed(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\")",
		},
		events,
	)
}

func TestAccountInboxPublishClaimWrongAccount(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	var logs []string
	var events []string

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.capabilities.storage.issue<&[Int]>(/storage/foo)
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	transaction2 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)
				log(cap)
			}
		}
	`)

	transaction3 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				let cap = signer.inbox.claim<&[Int]>("foo", provider: 0x1)!
				log(cap.borrow()![0])
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	runtimeInterface3 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		emitEvent: func(event cadence.Event) error {
			events = append(events, event.String())
			return nil
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 3}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

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
			"flow.InboxValuePublished(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\", type: Type<Capability<&[Int]>>())",
			"flow.InboxValueClaimed(provider: 0x0000000000000001, recipient: 0x0000000000000002, name: \"foo\")",
		},
		events,
	)
}
