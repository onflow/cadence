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
                  let unentitledRI1 = self.account.capabilities.get<&{RI}>(/public/unentitledRI).borrow()!
                  let entitledRI1 = self.account.capabilities.get<auth(X) &{RI}>(/public/unentitledRI).borrow()
                  assert(entitledRI1 == nil)

                  let unentitledRI2 = self.account.capabilities.get<&{RI}>(/public/entitledRI).borrow()!
                  let entitledRI2 = self.account.capabilities.get<auth(X) &{RI}>(/public/entitledRI).borrow()!
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
                  let unentitledSI1 = self.account.capabilities.get<&{SI}>(/public/unentitledSI).borrow()!
                  let entitledSI1 = self.account.capabilities.get<auth(X) &{SI}>(/public/unentitledSI).borrow()
                  assert(entitledSI1 == nil)

                  let unentitledSI2 = self.account.capabilities.get<&{SI}>(/public/entitledSI).borrow()!
                  let entitledSI2 = self.account.capabilities.get<auth(X) &{SI}>(/public/entitledSI).borrow()!
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
}
