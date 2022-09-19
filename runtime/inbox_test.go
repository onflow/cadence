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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountInboxPublishUnpublish(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// correct value returned from unpublish
	require.Equal(t, logs[1], "3")
}

func TestAccountInboxPublishWithoutPermission(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
			}
		}
	`)

	runtimeInterface1 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
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

	// unsuccessful publish
	require.Equal(t, logs[0], "false")
}

func TestAccountInboxUnpublishWrongType(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
				log(signer.inbox.publish(cap, name: "foo", recipient: 0x2))
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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
	assert.Contains(t, err.Error(), "unexpectedly found non-`Capability<&[String]>` while force-casting value")

	// successful publish
	require.Equal(t, logs[0], "true")
}

func TestAccountInboxUnpublishAbsent(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// correct value returned from unpublish
	require.Equal(t, logs[1], "nil")
}

func TestAccountInboxUnpublishRemove(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// correct value returned from unpublish
	require.Equal(t, logs[1], "3")

	// unpublish successfully removes the value
	require.Equal(t, logs[2], "nil")
}

func TestAccountInboxUnpublishWrongAccount(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// unpublish not successful from wrong account
	require.Equal(t, logs[1], "nil")

	// correct value returned from unpublish
	require.Equal(t, logs[2], "3")
}

func TestAccountInboxPublishClaim(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// correct value returned from claim
	require.Equal(t, logs[1], "3")
}

func TestAccountInboxPublishClaimWrongType(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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
	assert.Contains(t, err.Error(), "unexpectedly found non-`Capability<&[String]>` while force-casting value")

	// successful publish
	require.Equal(t, logs[0], "true")
}

func TestAccountInboxPublishClaimWrongPath(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// no value claimed
	require.Equal(t, logs[1], "nil")
}

func TestAccountInboxPublishClaimRemove(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
		},
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 2}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// correct value returned from claim
	require.Equal(t, logs[1], "3")

	// claimed value properly removed
	require.Equal(t, logs[2], "nil")
}

func TestAccountInboxPublishClaimWrongAccount(t *testing.T) {
	t.Parallel()

	storage := newTestLedger(nil, nil)
	rt := newTestInterpreterRuntime()

	logs := make([]string, 0)

	transaction0 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.inbox.permit(0x1)
			}
		}
	`)

	transaction1 := []byte(`
		transaction {
			prepare(signer: AuthAccount) {
				signer.save([3], to: /storage/foo)
				let cap = signer.link<&[Int]>(/public/foo, target: /storage/foo)!
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 1}}, nil
		},
	}

	runtimeInterface2 := &testRuntimeInterface{
		storage: storage,
		log: func(message string) {
			logs = append(logs, message)
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
		getSigningAccounts: func() ([]Address, error) {
			return []Address{[8]byte{0, 0, 0, 0, 0, 0, 0, 3}}, nil
		},
	}

	nextTransactionLocation := newTransactionLocationGenerator()

	// permit 1 to publish to 2
	err := rt.ExecuteTransaction(
		Script{
			Source: transaction0,
		},
		Context{
			Interface: runtimeInterface2,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)

	// publish from 1 to 2
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

	// successful publish
	require.Equal(t, logs[0], "true")

	// value is not claimed by 3
	require.Equal(t, logs[1], "nil")

	// value is claimed by 2
	require.Equal(t, logs[2], "3")
}
