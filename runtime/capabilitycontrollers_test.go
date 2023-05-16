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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCapabilityControllers(t *testing.T) {
	t.Parallel()

	test := func(tx string) (
		err error,
		logs []string,
		events []string,
	) {

		rt := newTestInterpreterRuntime()

		accountCodes := map[Location][]byte{}
		accountIDs := map[common.Address]uint64{}

		deployTx := DeploymentTransaction(
			"Test",
			// language=cadence
			[]byte(`
                  pub contract Test {

                      pub resource R {

                          pub let id: Int

                          init(id: Int) {
                              self.id = id
                          }
                      }

                      pub resource S {}

                      pub fun createAndSaveR(id: Int, storagePath: StoragePath) {
                              self.account.save(
                              <-create R(id: id),
                              to: storagePath
                          )
                      }

                      /// quickSort is qsort from "The C Programming Language".
                      ///
                      /// > Our version of quicksort is not the fastest possible,
                      /// > but it's one of the simplest.
                      ///
                      pub fun quickSort(_ items: &[AnyStruct], isLess: ((Int, Int): Bool)) {

                          fun quickSortPart(leftIndex: Int, rightIndex: Int) {

                              if leftIndex >= rightIndex {
                                  return
                              }

                              let pivotIndex = (leftIndex + rightIndex) / 2

                              items[pivotIndex] <-> items[leftIndex]

                              var lastIndex = leftIndex
                              var index = leftIndex + 1
                              while index <= rightIndex {
                                  if isLess(index, leftIndex) {
                                      lastIndex = lastIndex + 1
                                      items[lastIndex] <-> items[index]
                                  }
                                  index = index + 1
                              }

                              items[leftIndex] <-> items[lastIndex]

                              quickSortPart(leftIndex: leftIndex, rightIndex: lastIndex - 1)
                              quickSortPart(leftIndex: lastIndex + 1, rightIndex: rightIndex)
                          }

                          quickSortPart(
                              leftIndex: 0,
                              rightIndex: items.length - 1
                          )
                      }
                  }
                `),
		)

		signer := common.MustBytesToAddress([]byte{0x1})

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			log: func(message string) {
				logs = append(logs, message)
			},
			emitEvent: func(event cadence.Event) error {
				events = append(events, event.String())
				return nil
			},
			getSigningAccounts: func() ([]Address, error) {
				return []Address{signer}, nil
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
			generateAccountID: func(address common.Address) (uint64, error) {
				accountID := accountIDs[address] + 1
				accountIDs[address] = accountID
				return accountID, nil
			},
		}

		nextTransactionLocation := newTransactionLocationGenerator()

		// Deploy contract

		err = rt.ExecuteTransaction(
			Script{
				Source: deployTx,
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)
		require.NoError(t, err)

		// Call contract

		err = rt.ExecuteTransaction(
			Script{
				Source: []byte(tx),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		return
	}

	// TODO: account capability
	testAccount := func(accountType sema.Type, accountExpression string) {

		testName := fmt.Sprintf(
			"%s.Capabilities, storage capability",
			accountType.String(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("get non-existing", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                            transaction {
                                prepare(signer: AuthAccount) {
                                    // Act
                                    let gotCap: Capability<&AnyStruct>? =
                                        %s.capabilities.get<&AnyStruct>(/public/r)

                                    // Assert
                                    assert(gotCap == nil)
                                }
                            }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("get and borrow existing, with valid type", func(t *testing.T) {

				// TODO:
				t.Skip("TODO")

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R> =
                                      signer.capabilities.storage.issue<&Test.R>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)

                                  // Act
                                  let gotCap: Capability<&Test.R> = %s.capabilities.get<&Test.R>(publicPath)!
                                  let ref: &Test.R = gotCap.borrow()!

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(gotCap.id == expectedCapID)
                                  assert(ref.id == resourceID)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("get existing, with subtype", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R{}> =
                                      signer.capabilities.storage.issue<&Test.R{}>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)

                                  // Act
                                  let gotCap: Capability<&Test.R>? = %s.capabilities.get<&Test.R>(publicPath)

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(gotCap == nil)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("get existing, with different type", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R> =
                                      signer.capabilities.storage.issue<&Test.R>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)

                                  // Act
                                  let gotCap: Capability<&Test.S>? = %s.capabilities.get<&Test.S>(publicPath)

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(gotCap == nil)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("get unpublished", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R> =
                                      signer.capabilities.storage.issue<&Test.R>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)
                                  let unpublishedcap = signer.capabilities.unpublish(publicPath)

                                  // Act
                                  let gotCap: Capability<&Test.R>? = %s.capabilities.get<&Test.R>(publicPath)

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(unpublishedcap!.id == expectedCapID)
                                  assert(gotCap == nil)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("borrow non-existing", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                        transaction {
                            prepare(signer: AuthAccount) {
                                // Act
                                let ref: &AnyStruct? =
                                    %s.capabilities.borrow<&AnyStruct>(/public/r)

                                // Assert
                                assert(ref == nil)
                            }
                        }
                    `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("borrow existing, with valid type", func(t *testing.T) {

				// TODO:
				t.Skip("TODO")

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R> =
                                      signer.capabilities.storage.issue<&Test.R>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)

                                  // Act
                                  let ref: &Test.R = %s.capabilities.borrow<&Test.R>(publicPath)!

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(ref.id == resourceID)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("borrow existing, with subtype", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R{}> =
                                      signer.capabilities.storage.issue<&Test.R{}>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)

                                  // Act
                                  let ref: &Test.R? = %s.capabilities.borrow<&Test.R>(publicPath)

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(ref == nil)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("borrow existing, with different type", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R> =
                                      signer.capabilities.storage.issue<&Test.R>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)

                                  // Act
                                  let ref: &Test.S? = %s.capabilities.borrow<&Test.S>(publicPath)

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(ref == nil)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			t.Run("borrow unpublished", func(t *testing.T) {

				t.Parallel()

				err, _, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1
                                  let resourceID = 42

                                  // Arrange
                                  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                                  let issuedCap: Capability<&Test.R> =
                                      signer.capabilities.storage.issue<&Test.R>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)
                                  let unpublishedcap = signer.capabilities.unpublish(publicPath)

                                  // Act
                                  let ref: &Test.R? = %s.capabilities.borrow<&Test.R>(publicPath)

                                  // Assert
                                  assert(issuedCap.id == expectedCapID)
                                  assert(unpublishedcap!.id == expectedCapID)
                                  assert(ref == nil)
                              }
                          }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)
			})

			if accountType == sema.AuthAccountType {

				t.Run("publish, existing published", func(t *testing.T) {

					t.Parallel()

					err, _, _ := test(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let storagePath = /storage/r
                                  let publicPath = /public/r
                                  let expectedCapID: UInt64 = 1

                                  // Arrange
                                  let issuedCap: Capability<&Test.R> =
                                      signer.capabilities.storage.issue<&Test.R>(storagePath)
                                  signer.capabilities.publish(issuedCap, at: publicPath)

                                  // Act
                                  signer.capabilities.publish(issuedCap, at: publicPath)
                              }
                          }
                        `,
					)
					RequireError(t, err)

					var overwriteErr interpreter.OverwriteError
					require.ErrorAs(t, err, &overwriteErr)
				})

				t.Run("unpublish non-existing", func(t *testing.T) {

					t.Parallel()

					err, _, _ := test(
						// language=cadence
						`
                          import Test from 0x1

                          transaction {
                              prepare(signer: AuthAccount) {
                                  let publicPath = /public/r

                                  // Act
                                  let cap = signer.capabilities.unpublish(publicPath)

                                  // Assert
                                  assert(cap == nil)
                              }
                          }
                        `,
					)
					require.NoError(t, err)
				})
			}
		})
	}

	for accountType, accountExpression := range map[sema.Type]string{
		sema.AuthAccountType:   "signer",
		sema.PublicAccountType: "getAccount(0x1)",
	} {
		testAccount(accountType, accountExpression)
	}

	t.Run("AuthAccount.StorageCapabilities", func(t *testing.T) {

		t.Parallel()

		t.Run("issue, multiple controllers to various paths, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Act
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R{}> =
                              signer.capabilities.storage.issue<&Test.R{}>(storagePath1)
                          let issuedCap4: Capability<&Test.R{}> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath2)

                          // Assert
                          assert(issuedCap1.id == 1)
                          assert(issuedCap2.id == 2)
                          assert(issuedCap3.id == 3)
                          assert(issuedCap4.id == 4)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("getController, non-existing", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          // Act
                          let controller1: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: 0)
                          let controller2: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: 1)

                          // Assert
                          assert(controller1 == nil)
                          assert(controller2 == nil)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		// TODO: getController, non-storage capability controller

		t.Run("getController, multiple controllers to various paths, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R{}> =
                              signer.capabilities.storage.issue<&Test.R{}>(storagePath1)
                          let issuedCap4: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath2)

                          // Act
                          let controller1: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap1.id)
                          let controller2: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap2.id)
                          let controller3: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap3.id)
                          let controller4: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap4.id)

                          // Assert
                          assert(controller1!.capabilityID == 1)
                          assert(controller1!.borrowType == Type<&Test.R>())
                          assert(controller1!.target() == storagePath1)

                          assert(controller2!.capabilityID == 2)
                          assert(controller2!.borrowType == Type<&Test.R>())
                          assert(controller2!.target() == storagePath1)

                          assert(controller3!.capabilityID == 3)
                          assert(controller3!.borrowType == Type<&Test.R{}>())
                          assert(controller3!.target() == storagePath1)

                          assert(controller4!.capabilityID == 4)
                          assert(controller4!.borrowType == Type<&Test.R>())
                          assert(controller4!.target() == storagePath2)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("getControllers", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R{}> =
                              signer.capabilities.storage.issue<&Test.R{}>(storagePath1)
                          let issuedCap4: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath2)

                          // Act
                          let controllers1: [&StorageCapabilityController] =
                              signer.capabilities.storage.getControllers(forPath: storagePath1)
                          let controllers2: [&StorageCapabilityController] =
                              signer.capabilities.storage.getControllers(forPath: storagePath2)

                          // Assert
                          assert(controllers1.length == 3)

                          Test.quickSort(
                              &controllers1 as &[AnyStruct],
                              isLess: fun(i: Int, j: Int): Bool {
                                  let a = controllers1[i]
                                  let b = controllers1[j]
                                  return a.capabilityID < b.capabilityID
                              }
                          )

                          assert(controllers1[0].capabilityID == 1)
                          assert(controllers1[1].capabilityID == 2)
                          assert(controllers1[2].capabilityID == 3)

                          assert(controllers2.length == 1)
                          assert(controllers2[0].capabilityID == 4)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, all", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R{}> =
                              signer.capabilities.storage.issue<&Test.R{}>(storagePath1)
                          let issuedCap4: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath2)

                          // Act
                          let controllers1: [&StorageCapabilityController] = []
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath1,
                              fun (controller: &StorageCapabilityController): Bool {
                                  controllers1.append(controller)
                                  return true
                              }
                          )

                          let controllers2: [&StorageCapabilityController] = []
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath2,
                              fun (controller: &StorageCapabilityController): Bool {
                                  controllers2.append(controller)
                                  return true
                              }
                          )

                          // Assert
                          assert(controllers1.length == 3)

                          Test.quickSort(
                              &controllers1 as &[AnyStruct],
                              isLess: fun(i: Int, j: Int): Bool {
                                  let a = controllers1[i]
                                  let b = controllers1[j]
                                  return a.capabilityID < b.capabilityID
                              }
                          )

                          assert(controllers1[0].capabilityID == 1)
                          assert(controllers1[1].capabilityID == 2)
                          assert(controllers1[2].capabilityID == 3)

                          assert(controllers2.length == 1)
                          assert(controllers2[0].capabilityID == 4)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, stop immediately", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath = /storage/r

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath)

                          // Act
                          var stopped = false
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath,
                              fun (controller: &StorageCapabilityController): Bool {
                                  assert(!stopped)
                                  stopped = true
                                  return false
                              }
                          )

                          // Assert
                          assert(stopped)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})
	})

	// TODO: AuthAccount.AccountCapabilities

	t.Run("StorageCapabilityController", func(t *testing.T) {

		t.Parallel()

		t.Run("retarget", func(t *testing.T) {

			t.Parallel()

			// TODO: assert borrow, getControllers, and forEachController after retarget

			err, _, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let controller1: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap1.id)

                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let controller2: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap2.id)

                          let issuedCap3: Capability<&Test.R{}> =
                              signer.capabilities.storage.issue<&Test.R{}>(storagePath1)
                          let controller3: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap3.id)

                          let issuedCap4: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath2)
                          let controller4: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap4.id)

                          // Act
                          controller1!.retarget(storagePath2)

                          // Assert
                          assert(controller1!.target() == storagePath2)
                          let controller1After: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap1.id)
                          assert(controller1After!.target() == storagePath2)

                          assert(controller2!.target() == storagePath1)

                          assert(controller3!.target() == storagePath1)

                          assert(controller4!.target() == storagePath2)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})
	})

	// TODO: AccountCapabilityController
}
