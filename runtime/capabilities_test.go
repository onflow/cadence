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
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCapability_borrowAndCheck(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	newRuntime := func() (TestInterpreterRuntime, *TestRuntimeInterface) {
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
		return runtime, runtimeInterface
	}

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              entitlement X

              access(all)
              resource interface RI {}

              access(all)
              resource R: RI {

                  access(all)
                  let foo: Int

                  access(X)
                  let bar: Int

                  init() {
                      self.foo = 42
                      self.bar = 21
                  }
              }

              access(all)
              resource R2 {

                  access(all)
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              access(all)
              struct S {

                  access(all)
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              access(all)
              fun setup() {
                  let r <- create R()
                  self.account.storage.save(<-r, to: /storage/r)

                  let rCap = self.account.capabilities.storage.issue<&R>(/storage/r)
                  self.account.capabilities.publish(rCap, at: /public/r)

                  let rAsR2Cap = self.account.capabilities.storage.issue<&R2>(/storage/r)
                  self.account.capabilities.publish(rAsR2Cap, at: /public/rAsR2)

                  let rAsSCap = self.account.capabilities.storage.issue<&S>(/storage/r)
                  self.account.capabilities.publish(rAsSCap, at: /public/rAsS)

                  let noCap = self.account.capabilities.storage.issue<&R>(/storage/nonExistentTarget)
                  self.account.capabilities.publish(noCap, at: /public/nonExistentTarget)

                  let unentitledRICap = self.account.capabilities.storage.issue<&{RI}>(/storage/r)
                  self.account.capabilities.publish(unentitledRICap, at: /public/unentitledRI)

                  let entitledRICap = self.account.capabilities.storage.issue<auth(X) &{RI}>(/storage/r)
                  self.account.capabilities.publish(entitledRICap, at: /public/entitledRI)
              }

              access(all)
              fun testR() {
                  let path = /public/r
                  let cap = self.account.capabilities.get<&R>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                      cap.check(),
                      message: "check failed"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  let ref = cap.borrow()
                  assert(
                      ref != nil,
                      message: "borrow failed"
                  )

                  assert(
                      ref?.foo == 42,
                      message: "invalid foo"
                  )
              }

              access(all)
              fun testRAsR2() {
                  let path = /public/rAsR2
                  let cap = self.account.capabilities.get<&R2>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                      !cap.check(),
                      message: "invalid check"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  assert(
                      cap.borrow() == nil,
                      message: "invalid borrow"
                  )
              }

              access(all)
              fun testRAsS() {
                  let path = /public/rAsS
                  let cap = self.account.capabilities.get<&S>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                      !cap.check(),
                      message: "invalid check"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  assert(
                      cap.borrow() == nil,
                      message: "invalid borrow"
                  )
              }

              access(all)
              fun testNonExistentTarget() {
                  let path = /public/nonExistentTarget
                  let cap = self.account.capabilities.get<&R>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                      !cap.check(),
                      message: "invalid check"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  assert(
                      cap.borrow() == nil,
                      message: "invalid borrow"
                  )
              }

              access(all)
              fun testNonExistent() {
                  let path = /public/nonExistent

                  let cap = self.account.capabilities.get<&R>(path)
                  assert(cap.id == 0)
                  assert(cap as? Capability<&R> != nil)
                  assert(cap as? Capability<&AnyResource> != nil)
                  assert(cap.borrow() == nil)
                  assert(cap.address == 0x1)
                  assert(cap.check() == false)

                  let cap2 = self.account.capabilities.get<&AnyResource>(path)
                  assert(cap2.id == 0)
                  assert(cap2 as? Capability<&AnyResource> != nil)
                  assert(cap2.borrow() == nil)
                  assert(cap2.address == 0x1)
                  assert(cap2.check() == false)

                  assert(!self.account.capabilities.exists(path))
              }

              access(all)
              fun testSwap(): Int {
                  let ref = self.account.capabilities.get<&R>(/public/r).borrow()!

                  let r <- self.account.storage.load<@R>(from: /storage/r)
                  destroy r

                  let r2 <- create R2()
                  self.account.storage.save(<-r2, to: /storage/r)

                  return ref.foo
              }

              access(all)
              fun testRI() {
                  // Borrow /public/unentitledRI.
                  // - All unentitled borrows should succeed (as &{RI} / as &R)
                  // - All entitled borrows should fail (as &{RI} / as &R)

                  let unentitledRI1 = self.account.capabilities.get<&{RI}>(/public/unentitledRI).borrow()
                  assert(unentitledRI1 != nil, message: "unentitledRI1 should not be nil")

                  let entitledRI1 = self.account.capabilities.get<auth(X) &{RI}>(/public/unentitledRI).borrow()
                  assert(entitledRI1 == nil, message: "entitledRI1 should be nil")

                  let unentitledR1 = self.account.capabilities.get<&R>(/public/unentitledRI).borrow()
                  assert(unentitledR1 != nil, message: "unentitledR1 should not be nil")

                  let entitledR1 = self.account.capabilities.get<auth(X) &R>(/public/unentitledRI).borrow()
                  assert(entitledR1 == nil, message: "entitledR1 should be nil")

                  // Borrow /public/entitledRI.
                  // All borrows should succeed:
                  // - As &{RI} / as &R
                  // - Unentitled / entitled

                  let unentitledRI2 = self.account.capabilities.get<&{RI}>(/public/entitledRI).borrow()
                  assert(unentitledRI2 != nil, message: "unentitledRI2 should not be nil")

                  let entitledRI2 = self.account.capabilities.get<auth(X) &{RI}>(/public/entitledRI).borrow()
                  assert(entitledRI2 != nil, message: "entitledRI2 should not be nil")

                  let unentitledR2 = self.account.capabilities.get<&R>(/public/entitledRI).borrow()
                  assert(unentitledR2 != nil, message: "unentitledR2 should not be nil")

                  let entitledR2 = self.account.capabilities.get<auth(X) &R>(/public/entitledRI).borrow()
                  assert(entitledR2 != nil, message: "entitledR2 should not be nil")
              }
          }
        `

		contractLocation := common.NewAddressLocation(nil, address, "Test")

		deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

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

		// Run tests

		_, err = invoke("setup")
		require.NoError(t, err)

		t.Run("testR", func(t *testing.T) {
			_, err := invoke("testR")
			require.NoError(t, err)
		})

		t.Run("testRAsR2", func(t *testing.T) {
			_, err := invoke("testRAsR2")
			require.NoError(t, err)
		})

		t.Run("testRAsS", func(t *testing.T) {
			_, err := invoke("testRAsS")
			require.NoError(t, err)
		})

		t.Run("testNonExistentTarget", func(t *testing.T) {
			_, err := invoke("testNonExistentTarget")
			require.NoError(t, err)
		})

		t.Run("testNonExistent", func(t *testing.T) {
			_, err := invoke("testNonExistent")
			require.NoError(t, err)
		})

		t.Run("testSwap", func(t *testing.T) {

			_, err := invoke("testSwap")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})

		t.Run("testRI", func(t *testing.T) {

			_, err := invoke("testRI")
			require.NoError(t, err)
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              entitlement X

              access(all)
              struct interface SI {}

              access(all)
              struct S: SI {

                  access(all)
                  let foo: Int

                  access(X)
                  let bar: Int

                  init() {
                      self.foo = 42
                      self.bar = 21
                  }
              }

              access(all)
              struct S2 {

                  access(all)
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              access(all)
              resource R {

                  access(all)
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              access(all)
              fun setup() {
                  let s = S()
                  self.account.storage.save(s, to: /storage/s)

                  let sCap = self.account.capabilities.storage.issue<&S>(/storage/s)
                  self.account.capabilities.publish(sCap, at: /public/s)

                  let sAsS2Cap = self.account.capabilities.storage.issue<&S2>(/storage/s)
                  self.account.capabilities.publish(sAsS2Cap, at: /public/sAsS2)

                  let sAsRCap = self.account.capabilities.storage.issue<&R>(/storage/s)
                  self.account.capabilities.publish(sAsRCap, at: /public/sAsR)

                  let noCap = self.account.capabilities.storage.issue<&S>(/storage/nonExistentTarget)
                  self.account.capabilities.publish(noCap, at: /public/nonExistentTarget)

                  let unentitledSICap = self.account.capabilities.storage.issue<&{SI}>(/storage/s)
                  self.account.capabilities.publish(unentitledSICap, at: /public/unentitledSI)

                  let entitledSICap = self.account.capabilities.storage.issue<auth(X) &{SI}>(/storage/s)
                  self.account.capabilities.publish(entitledSICap, at: /public/entitledSI)
              }

              access(all)
              fun testS() {
                  let path = /public/s
                  let cap = self.account.capabilities.get<&S>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                       cap.check(),
                       message: "check failed"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  let ref = cap.borrow()
                  assert(
                      ref != nil,
                      message: "borrow failed"
                  )

                  assert(
                      ref?.foo == 42,
                      message: "invalid foo"
                  )
              }

              access(all)
              fun testSAsS2() {
                  let path = /public/sAsS2
                  let cap = self.account.capabilities.get<&S2>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                      !cap.check(),
                      message: "invalid check"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  assert(
                      cap.borrow() == nil,
                      message: "invalid borrow"
                  )
              }

              access(all)
              fun testSAsR() {
                  let path = /public/sAsR
                  let cap = self.account.capabilities.get<&R>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                      !cap.check(),
                      message: "invalid check"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  assert(
                      cap.borrow() == nil,
                      message: "invalid borrow"
                  )
              }

              access(all)
              fun testNonExistentTarget() {
                  let path = /public/nonExistentTarget
                  let cap = self.account.capabilities.get<&S>(path)

                  assert(self.account.capabilities.exists(path))

                  assert(
                      !cap.check(),
                      message: "invalid check"
                  )

                  assert(
                      cap.address == 0x1,
                      message: "invalid address"
                  )

                  assert(
                      cap.borrow() == nil,
                      message: "invalid borrow"
                  )
              }

              access(all)
              fun testNonExistent() {
                  let path = /public/nonExistent

                  let cap = self.account.capabilities.get<&S>(path)
                  assert(cap.id == 0)
                  assert(cap as? Capability<&S> != nil)
                  assert(cap as? Capability<&AnyStruct> != nil)
                  assert(cap.borrow() == nil)
                  assert(cap.address == 0x1)
                  assert(cap.check() == false)

                  let cap2 = self.account.capabilities.get<&AnyStruct>(path)
                  assert(cap2.id == 0)
                  assert(cap2 as? Capability<&AnyStruct> != nil)
                  assert(cap2.borrow() == nil)
                  assert(cap2.address == 0x1)
                  assert(cap2.check() == false)

                  assert(!self.account.capabilities.exists(path))
              }

              access(all)
              fun testSwap(): Int {
                  let ref = self.account.capabilities.get<&S>(/public/s).borrow()!

                  self.account.storage.load<S>(from: /storage/s)

                  let s2 = S2()
                  self.account.storage.save(s2, to: /storage/s)

                  return ref.foo
              }

              access(all)
              fun testSI() {

                  // Borrow /public/unentitledSI.
                  // - All unentitled borrows should succeed (as &{SI} / as &S)
                  // - All entitled borrows should fail (as &{SI} / as &S)

                  let unentitledSI1 = self.account.capabilities.get<&{SI}>(/public/unentitledSI).borrow()
                  assert(unentitledSI1 != nil, message: "unentitledSI1 should not be nil")

                  let entitledSI1 = self.account.capabilities.get<auth(X) &{SI}>(/public/unentitledSI).borrow()
                  assert(entitledSI1 == nil, message: "entitledSI1 should be nil")

                  let unentitledS1 = self.account.capabilities.get<&S>(/public/unentitledSI).borrow()
                  assert(unentitledS1 != nil, message: "unentitledS1 should not be nil")

                  let entitledS1 = self.account.capabilities.get<auth(X) &S>(/public/unentitledSI).borrow()
                  assert(entitledS1 == nil, message: "entitledS1 should be nil")

                  // Borrow /public/entitledSI.
                  // All borrows should succeed:
                  // - As &{SI} / as &S
                  // - Unentitled / entitled

                  let unentitledSI2 = self.account.capabilities.get<&{SI}>(/public/entitledSI).borrow()
                  assert(unentitledSI2 != nil, message: "unentitledSI2 should not be nil")

                  let entitledSI2 = self.account.capabilities.get<auth(X) &{SI}>(/public/entitledSI).borrow()
                  assert(entitledSI2 != nil, message: "entitledSI2 should not be nil")

                  let unentitledS2 = self.account.capabilities.get<&S>(/public/entitledSI).borrow()
                  assert(unentitledS2 != nil, message: "unentitledS2 should not be nil")

                  let entitledS2 = self.account.capabilities.get<auth(X) &S>(/public/entitledSI).borrow()
                  assert(entitledS2 != nil, message: "entitledS2 should not be nil")
              }
          }
        `

		contractLocation := common.NewAddressLocation(nil, address, "Test")

		deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

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

		// Run tests

		_, err = invoke("setup")
		require.NoError(t, err)

		t.Run("testS", func(t *testing.T) {
			_, err := invoke("testS")
			require.NoError(t, err)
		})

		t.Run("testSAsS2", func(t *testing.T) {
			_, err := invoke("testSAsS2")
			require.NoError(t, err)
		})

		t.Run("testSAsR", func(t *testing.T) {
			_, err := invoke("testSAsR")
			require.NoError(t, err)
		})

		t.Run("testNonExistentTarget", func(t *testing.T) {
			_, err := invoke("testNonExistentTarget")
			require.NoError(t, err)
		})

		t.Run("testNonExistent", func(t *testing.T) {
			_, err := invoke("testNonExistent")
			require.NoError(t, err)
		})

		t.Run("testSwap", func(t *testing.T) {

			_, err := invoke("testSwap")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})

		t.Run("testSI", func(t *testing.T) {

			_, err := invoke("testSI")
			require.NoError(t, err)
		})
	})

	t.Run("account", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              var cap: Capability

              access(all)
              entitlement X

              init() {
                  self.cap = self.account.capabilities.account.issue<&Account>()
              }

              access(all)
              fun test() {

                  assert(
                      self.cap.check<&Account>(),
                      message: "check failed"
                  )

                  assert(
                      self.cap.address == 0x1,
                      message: "invalid cap address"
                  )

                  let ref = self.cap.borrow<&Account>()
                  assert(
                      ref != nil,
                      message: "borrow failed"
                  )

                  assert(
                      ref?.address == 0x1,
                      message: "invalid ref address"
                  )
              }

              access(all)
              fun testAuth() {
                  assert(
                      !self.cap.check<auth(X) &Account>(),
                      message: "invalid check"
                  )

                  assert(
                      self.cap.address == 0x1,
                      message: "invalid address"
                  )

                  assert(
                      self.cap.borrow<auth(X) &Account>() == nil,
                      message: "invalid borrow"
                  )
              }
          }
        `

		contractLocation := common.NewAddressLocation(nil, address, "Test")

		deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

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

		// Run tests

		t.Run("test", func(t *testing.T) {
			_, err := invoke("test")
			require.NoError(t, err)
		})

		t.Run("testAuth", func(t *testing.T) {
			_, err := invoke("testAuth")
			require.NoError(t, err)
		})
	})

	t.Run("multiple sibling interfaces", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              entitlement X

              access(all)
              resource interface I1: I2 {}

              access(all)
              resource interface I2 {}

              access(all)
              resource interface I3 {}

              access(all)
              resource R: I1, I2, I3 {

                  access(all)
                  let foo: Int

                  access(X)
                  let bar: Int

                  init() {
                      self.foo = 42
                      self.bar = 21
                  }
              }

              access(all)
              fun setup() {
                  let r <- create R()
                  self.account.storage.save(<-r, to: /storage/r)

                  let i2Cap = self.account.capabilities.storage.issue<&{I2}>(/storage/r)
                  self.account.capabilities.publish(i2Cap, at: /public/i2)

                  let i2I3Cap = self.account.capabilities.storage.issue<&{I2, I3}>(/storage/r)
                  self.account.capabilities.publish(i2I3Cap, at: /public/i2i3)
              }

              access(all)
              fun testI2AsI1() {
                  let i1 = self.account.capabilities.get<&{I1}>(/public/i2).borrow()
                  assert(i1 != nil, message: "i1 should not be nil")
              }

              access(all)
              fun testI2I3AsI1() {
                  let i1 = self.account.capabilities.get<&{I1}>(/public/i2i3).borrow()
                  assert(i1 != nil, message: "i1 should not be nil")
              }
          }
        `

		contractLocation := common.NewAddressLocation(nil, address, "Test")

		deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

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

		// Run tests

		_, err = invoke("setup")
		require.NoError(t, err)

		t.Run("testI2AsI1", func(t *testing.T) {
			_, err := invoke("testI2AsI1")
			require.NoError(t, err)
		})

		t.Run("testI2I3AsI1", func(t *testing.T) {
			_, err := invoke("testI2I3AsI1")
			require.NoError(t, err)
		})
	})

	t.Run("multiple sibling interfaces complex", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := NewTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              entitlement X

              access(all)
              resource interface NFTReceiver {}

              access(all)
              resource interface ResolverCollection {}

              access(all)
              resource interface someCollectionPublic {}

              access(all)
              resource interface NFTCollectionPublic {}

              access(all)
              resource interface NFTCollection: NFTReceiver, NFTCollectionPublic, ResolverCollection {}

              access(all)
              resource Collection: NFTCollection, someCollectionPublic {

                  access(all)
                  let foo: Int

                  access(X)
                  let bar: Int

                  init() {
                      self.foo = 42
                      self.bar = 21
                  }
              }

              access(all)
              fun setup() {
                  let r <- create Collection()
                  self.account.storage.save(<-r, to: /storage/collection)

                  let collectionCap = self.account.capabilities.storage.issue<&{NFTCollectionPublic, ResolverCollection, someCollectionPublic}>(/storage/collection)
                  self.account.capabilities.publish(collectionCap, at: /public/collectionCap)
              }

              access(all)
              fun testBorrowCollectionCapAsReceiver() {
                  let receiver = self.account.capabilities.get<&{NFTReceiver}>(/public/collectionCap).borrow()
                  assert(receiver != nil, message: "receiver should not be nil")
              }
          }
        `

		contractLocation := common.NewAddressLocation(nil, address, "Test")

		deployTestContractTx := DeploymentTransaction("Test", []byte(testContract))

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

		// Run tests

		_, err = invoke("setup")
		require.NoError(t, err)

		t.Run("testBorrowCollectionCapAsReceiver", func(t *testing.T) {
			_, err := invoke("testBorrowCollectionCapAsReceiver")
			require.NoError(t, err)
		})
	})

}
