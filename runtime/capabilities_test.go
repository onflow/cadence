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
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCapability_borrow(t *testing.T) {

	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	newRuntime := func() (testInterpreterRuntime, *testRuntimeInterface) {
		runtime := newTestInterpreterRuntime()
		accountCodes := map[common.Location][]byte{}

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{address}, nil
			},
			resolveLocation: singleIdentifierLocationResolver(t),
			updateAccountContractCode: func(location common.AddressLocation, code []byte) error {
				accountCodes[location] = code
				return nil
			},
			getAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
				code = accountCodes[location]
				return code, nil
			},
			emitEvent: func(event cadence.Event) error {
				return nil
			},
		}
		return runtime, runtimeInterface
	}

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              entitlement X

              access(all)
              resource R {

                  access(all)
                  let foo: Int

                  init() {
                      self.foo = 42
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
                  self.account.save(<-r, to: /storage/r)

                  let rCap = self.account.capabilities.storage.issue<&R>(/storage/r)
                  self.account.capabilities.publish(rCap, at: /public/r)

                  let rAsR2Cap = self.account.capabilities.storage.issue<&R2>(/storage/r)
                  self.account.capabilities.publish(rAsR2Cap, at: /public/rAsR2)

                  let rAsSCap = self.account.capabilities.storage.issue<&S>(/storage/r)
                  self.account.capabilities.publish(rAsSCap, at: /public/rAsS)

                  let noCap = self.account.capabilities.storage.issue<&R>(/storage/nonExistent)
                  self.account.capabilities.publish(noCap, at: /public/nonExistent)
              }

              access(all)
              fun testBorrowR() {
                  let ref = self.account.capabilities.get<&R>(/public/r)!.borrow()!
                  assert(ref.foo == 42)
              }

              access(all)
              fun testBorrowRAsR2() {
                  assert(self.account.capabilities.get<&R2>(/public/rAsR2)!.borrow() == nil)
              }

              access(all)
              fun testBorrowRAsS() {
                  assert(self.account.capabilities.get<&S>(/public/rAsS)!.borrow() == nil)
              }

              access(all)
              fun testNonExistent() {
                  assert(self.account.capabilities.get<&R>(/public/nonExistent)!.borrow() == nil)
              }

              access(all)
              fun testSwap(): Int {
                 let ref = self.account.capabilities.get<&R>(/public/r)!.borrow()!

                 let r <- self.account.load<@R>(from: /storage/r)
                 destroy r

                 let r2 <- create R2()
                 self.account.save(<-r2, to: /storage/r)

                 return ref.foo
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

		t.Run("testBorrowR", func(t *testing.T) {
			_, err := invoke("testBorrowR")
			require.NoError(t, err)
		})

		t.Run("testBorrowRAsR2", func(t *testing.T) {
			_, err := invoke("testBorrowRAsR2")
			require.NoError(t, err)
		})

		t.Run("testBorrowRAsS", func(t *testing.T) {
			_, err := invoke("testBorrowRAsS")
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
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              entitlement X

              access(all)
              struct S {

                  access(all)
                  let foo: Int

                  init() {
                      self.foo = 42
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
                  self.account.save(s, to: /storage/s)

                  let sCap = self.account.capabilities.storage.issue<&S>(/storage/s)
                  self.account.capabilities.publish(sCap, at: /public/s)

                  let sAsS2Cap = self.account.capabilities.storage.issue<&S2>(/storage/s)
                  self.account.capabilities.publish(sAsS2Cap, at: /public/sAsS2)

                  let sAsRCap = self.account.capabilities.storage.issue<&R>(/storage/s)
                  self.account.capabilities.publish(sAsRCap, at: /public/sAsR)

                  let noCap = self.account.capabilities.storage.issue<&S>(/storage/nonExistent)
                  self.account.capabilities.publish(noCap, at: /public/nonExistent)
              }

              access(all)
              fun testBorrowS() {
                  let ref = self.account.capabilities.get<&S>(/public/s)!.borrow()!
                  assert(ref.foo == 42)
              }

              access(all)
              fun testBorrowSAsS2() {
                  assert(self.account.capabilities.get<&S2>(/public/sAsS2)!.borrow() == nil)
              }

              access(all)
              fun testBorrowSAsR() {
                  assert(self.account.capabilities.get<&R>(/public/sAsR)!.borrow() == nil)
              }

              access(all)
              fun testNonExistent() {
                  assert(self.account.capabilities.get<&S>(/public/nonExistent)!.borrow() == nil)
              }

              access(all)
              fun testSwap(): Int {
                 let ref = self.account.capabilities.get<&S>(/public/s)!.borrow()!

                 self.account.load<S>(from: /storage/s)

                 let s2 = S2()
                 self.account.save(s2, to: /storage/s)

                 return ref.foo
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

		t.Run("testBorrowS", func(t *testing.T) {
			_, err := invoke("testBorrowS")
			require.NoError(t, err)
		})

		t.Run("testBorrowSAsS2", func(t *testing.T) {
			_, err := invoke("testBorrowSAsS2")
			require.NoError(t, err)
		})

		t.Run("testBorrowSAsR", func(t *testing.T) {
			_, err := invoke("testBorrowSAsR")
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
	})

	t.Run("account", func(t *testing.T) {

		t.Parallel()

		runtime, runtimeInterface := newRuntime()

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		const testContract = `
          access(all)
          contract Test {

              access(all)
              var cap: Capability

              access(all)
              entitlement X

              init() {
	              self.cap = self.account.capabilities.account.issue<&AuthAccount>()
	          }

              access(all)
	          fun testBorrow() {
                  let ref = self.cap.borrow<&AuthAccount>()!
	              assert(ref.address == 0x1)
	          }

              access(all)
	          fun testBorrowAuth() {
	              assert(self.cap.borrow<auth(X) &AuthAccount>() == nil)
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

		t.Run("address", func(t *testing.T) {
			_, err := invoke("testBorrow")
			require.NoError(t, err)
		})

		t.Run("borrowAuth", func(t *testing.T) {
			_, err := invoke("testBorrowAuth")
			require.NoError(t, err)
		})
	})
}
