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
              fun saveAndPublish() {
                  let r <- create R()
                  self.account.save(<-r, to: /storage/r)

                  let rCap = self.account.capabilities.storage.issue<&R>(/storage/r)
                  self.account.capabilities.publish(rCap, at: /public/r)

                  let r2Cap = self.account.capabilities.storage.issue<&R2>(/storage/r)
                  self.account.capabilities.publish(r2Cap, at: /public/r2)

                  let noCap = self.account.capabilities.storage.issue<&R>(/storage/nonExistent)
                  self.account.capabilities.publish(noCap, at: /public/nonExistent)
              }

              access(all)
              fun foo(_ path: PublicPath): Int {
                  return self.account.capabilities.borrow<&R>(path)!.foo
              }

              access(all)
              fun single(): Int {
                  return self.foo(/public/r)
              }

              access(all)
              fun singleAuth(): auth(X) &R? {
                  return self.account.capabilities.borrow<auth(X) &R>(/public/r)
              }

              access(all)
              fun singleR2(): &R2? {
                  return self.account.capabilities.borrow<&R2>(/public/r)
              }

              access(all)
              fun singleS(): &S? {
                  return self.account.capabilities.borrow<&S>(/public/r)
              }

              access(all)
              fun nonExistent(): Int {
                  return self.foo(/public/nonExistent)
              }

              access(all)
              fun singleTyped(): Int {
                  return self.account.capabilities.borrow<&R>(/public/r)!.foo
              }

              access(all)
              fun r2(): Int {
                  return self.account.capabilities.borrow<&R2>(/public/r2)!.foo
              }

              access(all)
              fun singleChangeAfterBorrow(): Int {
                 let ref = self.account.capabilities.borrow<&R>(/public/r)!

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

		// Run test scripts

		// save

		_, err = invoke("saveAndPublish")
		require.NoError(t, err)

		t.Run("single", func(t *testing.T) {

			value, err := invoke("single")
			require.NoError(t, err)

			require.Equal(
				t,
				cadence.NewInt(42),
				value,
			)
		})

		t.Run("single R2", func(t *testing.T) {

			value, err := invoke("singleR2")
			require.NoError(t, err)

			require.Equal(t, cadence.Optional{}, value)
		})

		t.Run("single S", func(t *testing.T) {

			value, err := invoke("singleS")
			require.NoError(t, err)

			require.Equal(t, cadence.Optional{}, value)
		})

		t.Run("single auth", func(t *testing.T) {

			value, err := invoke("singleAuth")
			require.NoError(t, err)

			require.Equal(t, cadence.Optional{}, value)
		})

		t.Run("nonExistent", func(t *testing.T) {

			_, err := invoke("nonExistent")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceNilError{})
		})

		t.Run("singleTyped", func(t *testing.T) {

			value, err := invoke("singleTyped")
			require.NoError(t, err)

			require.Equal(
				t,
				cadence.NewInt(42),
				value,
			)
		})

		t.Run("r2", func(t *testing.T) {

			_, err := invoke("r2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceNilError{})
		})

		t.Run("single change after borrow", func(t *testing.T) {

			_, err := invoke("singleChangeAfterBorrow")
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
	          fun saveAndPublish() {
	              let s = S()
	              self.account.save(s, to: /storage/s)

                  let sCap = self.account.capabilities.storage.issue<&S>(/storage/s)
                  self.account.capabilities.publish(sCap, at: /public/s)

                  let s2Cap = self.account.capabilities.storage.issue<&S2>(/storage/s)
                  self.account.capabilities.publish(s2Cap, at: /public/s2)

                  let noCap = self.account.capabilities.storage.issue<&S>(/storage/nonExistent)
                  self.account.capabilities.publish(noCap, at: /public/nonExistent)
	          }

              access(all)
	          fun foo(_ path: PublicPath): Int {
	              return self.account.capabilities.borrow<&S>(path)!.foo
	          }

              access(all)
	          fun single(): Int {
	              return self.foo(/public/s)
	          }

              access(all)
	          fun singleAuth(): auth(X) &S? {
	              return self.account.capabilities.borrow<auth(X) &S>(/public/s)
	          }

              access(all)
	          fun singleS2(): &S2? {
	              return self.account.capabilities.borrow<&S2>(/public/s)
	          }

              access(all)
	          fun singleR(): &R? {
	              return self.account.capabilities.borrow<&R>(/public/s)
	          }

              access(all)
	          fun nonExistent(): Int {
	              return self.foo(/public/nonExistent)
	          }

              access(all)
	          fun singleTyped(): Int {
	              return self.account.capabilities.borrow<&S>(/public/s)!.foo
	          }

              access(all)
	          fun s2(): Int {
	              return self.account.capabilities.borrow<&S2>(/public/s2)!.foo
	          }

              access(all)
	          fun singleChangeAfterBorrow(): Int {
	              let ref = self.account.capabilities.borrow<&S>(/public/s)!

	              // remove stored value
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

		// Run test scripts

		// save

		_, err = invoke("saveAndPublish")
		require.NoError(t, err)

		t.Run("single", func(t *testing.T) {

			value, err := invoke("single")
			require.NoError(t, err)

			require.Equal(
				t,
				cadence.NewInt(42),
				value,
			)
		})

		t.Run("single S2", func(t *testing.T) {

			value, err := invoke("singleS2")
			require.NoError(t, err)

			require.Equal(t, cadence.Optional{}, value)
		})

		t.Run("single R", func(t *testing.T) {

			value, err := invoke("singleR")
			require.NoError(t, err)

			require.Equal(t, cadence.Optional{}, value)
		})

		t.Run("single auth", func(t *testing.T) {

			value, err := invoke("singleAuth")
			require.NoError(t, err)

			require.Equal(t, cadence.Optional{}, value)
		})

		t.Run("nonExistent", func(t *testing.T) {

			_, err := invoke("nonExistent")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceNilError{})
		})

		t.Run("singleTyped", func(t *testing.T) {

			value, err := invoke("singleTyped")
			require.NoError(t, err)

			require.Equal(
				t,
				cadence.NewInt(42),
				value,
			)
		})

		t.Run("s2", func(t *testing.T) {

			_, err := invoke("s2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceNilError{})
		})

		t.Run("single change after borrow", func(t *testing.T) {

			_, err := invoke("singleChangeAfterBorrow")
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
	          fun address(): Address {
	              return self.cap.borrow<&AuthAccount>()!.address
	          }

              access(all)
	          fun borrowAuth(): auth(X) &AuthAccount? {
	              return self.cap.borrow<auth(X) &AuthAccount>()
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

		// Run test scripts

		t.Run("address", func(t *testing.T) {

			value, err := invoke("address")
			require.NoError(t, err)

			require.Equal(
				t,
				cadence.Address(address),
				value,
			)
		})

		t.Run("borrowAuth", func(t *testing.T) {

			value, err := invoke("borrowAuth")
			require.NoError(t, err)

			require.Equal(t, cadence.Optional{}, value)
		})
	})
}
