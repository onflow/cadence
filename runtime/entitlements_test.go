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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestRuntimeAccountEntitlementSaveAndLoadSuccess(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement X
            access(all) entitlement Y
        }
    `))

	transaction1 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: auth(Storage, Capabilities) &Account) {
                signer.storage.save(3, to: /storage/foo)
                let cap = signer.capabilities.storage.issue<auth(Test.X, Test.Y) &Int>(/storage/foo)
                signer.capabilities.publish(cap, at: /public/foo)
            }
        }
     `)

	transaction2 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: &Account) {
                let ref = signer.capabilities.borrow<auth(Test.X, Test.Y) &Int>(/public/foo)!
                let downcastRef = ref as! auth(Test.X, Test.Y) &Int
            }
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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

}

func TestRuntimeAccountEntitlementSaveAndLoadFail(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement X
            access(all) entitlement Y
        }
    `))

	transaction1 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: auth(Storage, Capabilities) &Account) {
                signer.storage.save(3, to: /storage/foo)
                let cap = signer.capabilities.storage.issue<auth(Test.X, Test.Y) &Int>(/storage/foo)
                signer.capabilities.publish(cap, at: /public/foo)
            }
        }
     `)

	transaction2 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: &Account) {
                let ref = signer.capabilities.borrow<auth(Test.X) &Int>(/public/foo)!
                let downcastRef = ref as! auth(Test.X, Test.Y) &Int
            }
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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

	require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
}

func TestRuntimeAccountEntitlementAttachment(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement Y

            access(all) resource R {
				access(Y) fun foo() {}
			}

            access(all) attachment A for R {
                access(Y) fun foo() {}
            }

            access(all) fun createRWithA(): @R {
                return <-attach A() to <-create R()
            }
        }
    `))

	transaction1 := []byte(`
        import Test from 0x1

        transaction {
            prepare(signer: auth(Storage, Capabilities) &Account) {
                let r <- Test.createRWithA()
                signer.storage.save(<-r, to: /storage/foo)
                let cap = signer.capabilities.storage.issue<auth(Test.Y) &Test.R>(/storage/foo)
                signer.capabilities.publish(cap, at: /public/foo)
            }
        }
     `)

	transaction2 := []byte(`
        import Test from 0x1

        transaction {
            prepare(signer: &Account) {
                let ref = signer.capabilities.borrow<auth(Test.Y) &Test.R>(/public/foo)!
                ref[Test.A]!.foo()
            }
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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
}

func TestRuntimeAccountExportEntitledRef(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement X

            access(all) resource R {}

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `))

	script := []byte(`
        import Test from 0x1
        access(all) fun main(): &Test.R {
            let r <- Test.createR()
            let authAccount = getAuthAccount<auth(Storage) &Account>(0x1)
            authAccount.storage.save(<-r, to: /storage/foo)
            let ref = authAccount.storage.borrow<auth(Test.X) &Test.R>(from: /storage/foo)!
            return ref
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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

	value, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)
	require.Equal(t, "A.0000000000000001.Test.R(uuid: 1)", value.String())
}

func TestRuntimeAccountEntitlementNamingConflict(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement X

            access(all) resource R {
                access(X) fun foo() {}
            }

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `))

	otherDeployTx := DeploymentTransaction("OtherTest", []byte(`
        access(all) contract OtherTest {
            access(all) entitlement X
        }
    `))

	script := []byte(`
        import Test from 0x1
        import OtherTest from 0x1

        access(all) fun main() {
            let r <- Test.createR()
            let ref = &r as auth(OtherTest.X) &Test.R
            ref.foo()
            destroy r
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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

	err = rt.ExecuteTransaction(
		Script{
			Source: otherDeployTx,
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

	var checkerErr *sema.CheckerError
	require.ErrorAs(t, err, &checkerErr)

	errs := RequireCheckerErrors(t, checkerErr, 1)

	var accessError *sema.InvalidAccessError
	require.ErrorAs(t, errs[0], &accessError)
}

func TestRuntimeAccountEntitlementCapabilityCasting(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement X
            access(all) entitlement Y

            access(all) resource R {}

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `))

	transaction1 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: auth(Storage, Capabilities) &Account) {
                let r <- Test.createR()
                signer.storage.save(<-r, to: /storage/foo)
                let cap = signer.capabilities.storage.issue<auth(Test.X) &Test.R>(/storage/foo)
                signer.capabilities.publish(cap, at: /public/foo)
            }
        }
     `)

	transaction2 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: &Account) {
                let capX = signer.capabilities.get<auth(Test.X) &Test.R>(/public/foo)
                let upCap = capX as Capability<&Test.R>
                let downCap = upCap as! Capability<auth(Test.X) &Test.R>
            }
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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

	require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
}

func TestRuntimeAccountEntitlementCapabilityDictionary(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement X
            access(all) entitlement Y

            access(all) resource R {}

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `))

	transaction1 := []byte(`
        import Test from 0x1

        transaction {
            prepare(signer: auth(Storage, Capabilities) &Account) {
                let r <- Test.createR()
                signer.storage.save(<-r, to: /storage/foo)

                let capFoo = signer.capabilities.storage.issue<auth(Test.X) &Test.R>(/storage/foo)
                signer.capabilities.publish(capFoo, at: /public/foo)

                let r2 <- Test.createR()
                signer.storage.save(<-r2, to: /storage/bar)

                let capBar = signer.capabilities.storage.issue<auth(Test.Y) &Test.R>(/storage/bar)
                signer.capabilities.publish(capBar, at: /public/bar)
            }
        }
     `)

	transaction2 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: &Account) {
                let capX = signer.capabilities.get<auth(Test.X) &Test.R>(/public/foo)
                let capY = signer.capabilities.get<auth(Test.Y) &Test.R>(/public/bar)

                let dict: {Type: Capability<&Test.R>} = {}
                dict[capX.getType()] = capX
                dict[capY.getType()] = capY

                let newCapX = dict[capX.getType()]!
                let ref = newCapX.borrow()!
                let downCast = ref as! auth(Test.X) &Test.R
            }
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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

	require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
}

func TestRuntimeAccountEntitlementGenericCapabilityDictionary(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	deployTx := DeploymentTransaction("Test", []byte(`
        access(all) contract Test {
            access(all) entitlement X
            access(all) entitlement Y

            access(all) resource R {}

            access(all) fun createR(): @R {
                return <-create R()
            }
        }
    `))

	transaction1 := []byte(`
        import Test from 0x1

        transaction {
            prepare(signer: auth(Storage, Capabilities) &Account) {
                let r <- Test.createR()
                signer.storage.save(<-r, to: /storage/foo)

                let capFoo = signer.capabilities.storage.issue<auth(Test.X) &Test.R>(/storage/foo)
                signer.capabilities.publish(capFoo, at: /public/foo)

                let r2 <- Test.createR()
                signer.storage.save(<-r2, to: /storage/bar)

                let capBar = signer.capabilities.storage.issue<auth(Test.Y) &Test.R>(/storage/bar)
                signer.capabilities.publish(capBar, at: /public/bar)
            }
        }
     `)

	transaction2 := []byte(`
        import Test from 0x1
        transaction {
            prepare(signer: &Account) {
                let capX = signer.capabilities.get<auth(Test.X) &Test.R>(/public/foo)
                let capY = signer.capabilities.get<auth(Test.Y) &Test.R>(/public/bar)

                let dict: {Type: Capability} = {}
                dict[capX.getType()] = capX
                dict[capY.getType()] = capY

                let newCapX = dict[capX.getType()]!
                let ref = newCapX.borrow<auth(Test.X) &Test.R>()!
                let downCast = ref as! auth(Test.X) &Test.R
            }
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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
}

func TestRuntimeCapabilityEntitlements(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	test := func(t *testing.T, script string) {
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
	}

	t.Run("can borrow with supertype", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          entitlement X

          access(all)
          entitlement Y

          access(all)
          resource R {}

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              let r <- create R()
              account.storage.save(<-r, to: /storage/foo)

              let issuedCap = account.capabilities.storage.issue<auth(X, Y) &R>(/storage/foo)
              account.capabilities.publish(issuedCap, at: /public/foo)

              let ref = account.capabilities.borrow<auth(X | Y) &R>(/public/foo)
              assert(ref != nil, message: "failed borrow")
          }
        `)
	})

	t.Run("cannot borrow with supertype then downcast", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          entitlement X

          access(all)
          entitlement Y

          access(all)
          resource R {}

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              let r <- create R()
              account.storage.save(<-r, to: /storage/foo)

              let issuedCap = account.capabilities.storage.issue<auth(X, Y) &R>(/storage/foo)
              account.capabilities.publish(issuedCap, at: /public/foo)

              let ref = account.capabilities.borrow<auth(X | Y) &R>(/public/foo)
              assert(ref != nil, message: "failed borrow")

              let ref2 = ref! as? auth(X, Y) &R
              assert(ref2 == nil, message: "invalid cast")
          }
        `)
	})

	t.Run("can borrow with two types", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          entitlement X

          access(all)
          entitlement Y

          access(all)
          resource R {}

          access(all)
          fun main() {
               let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

               let r <- create R()
               account.storage.save(<-r, to: /storage/foo)

               let issuedCap = account.capabilities.storage.issue<auth(X, Y) &R>(/storage/foo)
               account.capabilities.publish(issuedCap, at: /public/foo)

               let ref = account.capabilities.borrow<auth(X, Y) &R>(/public/foo)
               assert(ref != nil, message: "failed borrow")

               let ref2 = ref! as? auth(X, Y) &R
               assert(ref2 != nil, message: "failed cast")
          }
        `)
	})

	t.Run("upcast runtime entitlements", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          entitlement X

          access(all)
          struct S {}

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              let s = S()
              account.storage.save(s, to: /storage/foo)

              let issuedCap = account.capabilities.storage.issue<auth(X) &S>(/storage/foo)
              account.capabilities.publish(issuedCap, at: /public/foo)

              let cap: Capability<auth(X) &S> = account.capabilities.get<auth(X) &S>(/public/foo)

              let runtimeType = cap.getType()

              let upcastCap = cap as Capability<&S>
              let upcastRuntimeType = upcastCap.getType()

              assert(runtimeType != upcastRuntimeType)
          }
        `)
	})

	t.Run("upcast runtime type", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          struct S {}

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              let s = S()
              account.storage.save(s, to: /storage/foo)

              let issuedCap = account.capabilities.storage.issue<&S>(/storage/foo)
              account.capabilities.publish(issuedCap, at: /public/foo)

              let cap: Capability<&S> = account.capabilities.get<&S>(/public/foo)

              let runtimeType = cap.getType()
              let upcastCap = cap as Capability<&AnyStruct>
              let upcastRuntimeType = upcastCap.getType()
              assert(runtimeType == upcastRuntimeType)
           }
        `)
	})

	t.Run("can check with supertype", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          entitlement X

          access(all)
          entitlement Y

          access(all)
          resource R {}

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              let r <- create R()
              account.storage.save(<-r, to: /storage/foo)

              let issuedCap = account.capabilities.storage.issue<auth(X, Y) &R>(/storage/foo)
              account.capabilities.publish(issuedCap, at: /public/foo)

              let cap = account.capabilities.get<auth(X | Y) &R>(/public/foo)
              assert(cap.check())
          }
        `)
	})

	t.Run("cannot borrow with subtype", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          entitlement X

          access(all)
          entitlement Y

          access(all)
          resource R {}

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              let r <- create R()
              account.storage.save(<-r, to: /storage/foo)

              let issuedCap = account.capabilities.storage.issue<auth(X) &R>(/storage/foo)
              account.capabilities.publish(issuedCap, at: /public/foo)

              let ref = account.capabilities.borrow<auth(X, Y) &R>(/public/foo)
              assert(ref == nil)
          }
        `)
	})

	t.Run("cannot get with subtype", func(t *testing.T) {
		t.Parallel()

		test(t, `
          access(all)
          entitlement X

          access(all)
          entitlement Y

          access(all)
          resource R {}

          access(all)
          fun main() {
              let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)

              let r <- create R()
              account.storage.save(<-r, to: /storage/foo)

              let issuedCap = account.capabilities.storage.issue<auth(X) &R>(/storage/foo)
              account.capabilities.publish(issuedCap, at: /public/foo)

              let cap = account.capabilities.get<auth(X, Y) &R>(/public/foo)
              assert(!cap.check())
          }
        `)
	})
}

func TestRuntimeImportedEntitlementMapInclude(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	furtherUpstreamDeployTx := DeploymentTransaction("FurtherUpstream", []byte(`
        access(all) contract FurtherUpstream {
            access(all) entitlement X
            access(all) entitlement Y
            access(all) entitlement Z

            access(all) entitlement mapping M {
                X -> Y
                Y -> Z
            }
        }
    `))

	upstreamDeployTx := DeploymentTransaction("Upstream", []byte(`
        import FurtherUpstream from 0x1

        access(all) contract Upstream {
            access(all) entitlement A
            access(all) entitlement B
            access(all) entitlement C

            access(all) entitlement mapping M {
                include FurtherUpstream.M

                A -> FurtherUpstream.Y
                FurtherUpstream.X -> B
            }
        }
    `))

	testDeployTx := DeploymentTransaction("Test", []byte(`
        import FurtherUpstream from 0x1
        import Upstream from 0x1

        access(all) contract Test {
            access(all) entitlement E
            access(all) entitlement F
            access(all) entitlement G

            access(all) entitlement mapping M {
                include Upstream.M

                E -> FurtherUpstream.Z
                E -> G
                F -> Upstream.C
                Upstream.C -> FurtherUpstream.X
            }

            access(all) struct S {
                access(mapping M) let x: [Int]

                init() {
                    self.x = [1]
                }
            }
        }
    `))

	script := []byte(`
        import Test from 0x1
        import Upstream from 0x1
        import FurtherUpstream from 0x1

        access(all) fun main() {
            let ref1 = &Test.S() as auth(FurtherUpstream.X, Upstream.C, Test.E) &Test.S
            ref1.x
            let type1 = Type<
                auth(
                    // from map of FurtherUpstream.X
                    FurtherUpstream.Y,
                    Upstream.B,
                    // from map of Upstream.C
                    FurtherUpstream.X,
                    // from map of Test.E
                    FurtherUpstream.Z,
                    Test.G
                ) &Int
            >()
            assert(type1 != nil)
            assert(ref1.x[0].getType() == Type<Int>())

            let ref2 = &Test.S() as auth(FurtherUpstream.Y, Upstream.A, Test.F) &Test.S
            ref2.x
            let type2 = Type<
                auth(
                    // from map of FurtherUpstream.Y
                    FurtherUpstream.Z,
                    // from map of Upstream.A
                    FurtherUpstream.Y,
                    // from map of Test.F
                    Upstream.C
                ) &Int
            >()
            assert(type2 != nil)
            assert(ref2.x[0].getType() == Type<Int>())

            let ref3 = &Test.S() as auth(FurtherUpstream.Z, Upstream.B, Test.G) &Test.S
            ref3.x
            assert(ref3.x[0].getType() == Type<Int>())
        }
     `)

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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
			Source: furtherUpstreamDeployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: upstreamDeployTx,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	err = rt.ExecuteTransaction(
		Script{
			Source: testDeployTx,
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

	require.NoError(t, err)
}

func TestRuntimeEntitlementMapIncludeDeduped(t *testing.T) {
	t.Parallel()

	storage := NewTestLedger(nil, nil)
	rt := NewTestInterpreterRuntime()
	accountCodes := map[Location][]byte{}

	script := []byte(`
	access(all) entitlement E
	access(all) entitlement F
	
	access(all) entitlement X
	access(all) entitlement Y
	access(all) entitlement mapping N1 {
	  E -> F
	  E -> E
	  E -> X
	  E -> Y
	  F -> F
	  F -> E
	  F -> X
	  F -> Y
	  X -> F
	  X -> E
	  X -> X
	  X -> Y
	}
	access(all) entitlement mapping N2{ include N1 }
	access(all) entitlement mapping N3{ include N2 }
	access(all) entitlement mapping N4{ include N3 }
	access(all) entitlement mapping A {
	  include N1
	  include N2
	  include N3
	  include N4
	}
	access(all) entitlement mapping B {
	  include A
	  include N1
	  include N2
	}
	access(all) entitlement mapping C {
	  include A
	  include B
	  include N1
	  include N2
	}
	access(all) entitlement mapping D {
	  include A
	  include B
	  include C
	  include N1
	  include N2
	}
	access(all) entitlement mapping AA {
	  include A
	  include B
	  include C
	  include D
	}
	access(all) entitlement mapping BB {include AA}
	access(all) entitlement mapping CC {include AA}
	access(all) entitlement mapping DD {include AA}
	access(all) entitlement mapping AAA {
	  include AA
	  include BB
	  include CC
	  include DD
	}
	access(all) entitlement mapping BBB {
	  include AAA
	  include AA
	  include BB
	  include CC
	  include DD
	}
	access(all) entitlement mapping CCC {
	  include AAA
	  include BBB
	  include AA
	  include BB
	  include CC
	  include DD
	}
	access(all) entitlement mapping DDD {
	  include AAA
	  include BBB
	  include CCC
	  include AA
	  include BB
	  include CC
	  include DD
	}
	access(all) entitlement mapping AAAA {
	  include AAA
	  include BBB
	  include CCC
	  include DDD
	}
	access(all) entitlement mapping BBBB {
	  include AAAA
	  include AAA
	  include BBB
	  include CCC
	  include DDD
	}
	access(all) entitlement mapping CCCC {
	  include AAAA
	  include BBBB
	  include AAA
	  include BBB
	  include CCC
	  include DDD
	}
	access(all) entitlement mapping DDDD {
	  include AAAA
	  include BBBB
	  include CCCC
	  include AAA
	  include BBB
	  include CCC
	  include DDD
	}
	access(all) entitlement mapping AAAAA {
	  include AAAA
	  include BBBB
	  include CCCC
	  include DDDD
	}
	access(all) entitlement mapping BBBBB {
	  include AAAAA
	  include AAAA
	  include BBBB
	  include CCCC
	  include DDDD
	}
	access(all) entitlement mapping CCCCC {
	  include AAAAA
	  include BBBBB
	  include AAAA
	  include BBBB
	  include CCCC
	  include DDDD
	}
	access(all) entitlement mapping DDDDD {
	  include AAAAA
	  include BBBBB
	  include CCCCC
	  include AAAA
	  include BBBB
	  include CCCC
	  include DDDD
	}
	access(all) entitlement mapping AAAAAA {
	  include AAAAA
	  include BBBBB
	  include CCCCC
	  include DDDDD
	}
	access(all) entitlement mapping BBBBBB { include AAAAAA}
	access(all) entitlement mapping CCCCCC { include AAAAAA}
	access(all) entitlement mapping DDDDDD { include AAAAAA}
	access(all) entitlement mapping P1 {
	  include AAAAAA
	  include BBBBBB
	  include CCCCCC
	  include DDDDDD
	}
	access(all) entitlement mapping P2 { include P1 }
	access(all) entitlement mapping P3 {
	  include P1
	  include P2
	}
	access(all) entitlement mapping P4 {
	  include P1
	  include P2
	  include P3
	}
	
	access(all) fun main() {}
    `)

	nextScriptLocation := NewScriptLocationGenerator()

	var totalRelations uint
	var failed bool

	runtimeInterface1 := &TestRuntimeInterface{
		Storage:      storage,
		OnProgramLog: func(message string) {},
		OnEmitEvent: func(event cadence.Event) error {
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
		OnMeterMemory: func(usage common.MemoryUsage) error {
			if usage.Kind == common.MemoryKindEntitlementRelationSemaType {
				totalRelations++
			}
			if totalRelations > 1000 {
				failed = true
			}
			return nil
		},
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface1,
			Location:  nextScriptLocation(),
		},
	)

	require.NoError(t, err)
	require.False(t, failed)
}
