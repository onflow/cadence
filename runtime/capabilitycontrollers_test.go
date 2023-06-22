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
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeCapabilityControllers(t *testing.T) {
	t.Parallel()

	test := func(tx string) (
		err error,
		storage *Storage,
	) {

		rt := newTestInterpreterRuntime()

		accountCodes := map[Location][]byte{}
		accountIDs := map[common.Address]uint64{}

		deployTx := DeploymentTransaction(
			"Test",
			// language=cadence
			[]byte(`
                  access(all) contract Test {

                      access(all) entitlement X

                      access(all) resource R {

                          access(all) let id: Int

                          init(id: Int) {
                              self.id = id
                          }
                      }

                      access(all) resource S {}

                      access(all) fun createAndSaveR(id: Int, storagePath: StoragePath) {
                          self.account.save(
                              <-create R(id: id),
                              to: storagePath
                          )
                      }

                      access(all) fun createAndSaveS(storagePath: StoragePath) {
                          self.account.save(
                              <-create S(),
                              to: storagePath
                          )
                      }

                      /// quickSort is qsort from "The C Programming Language".
                      ///
                      /// > Our version of quicksort is not the fastest possible,
                      /// > but it's one of the simplest.
                      ///
                      access(all) fun quickSort(_ items: &[AnyStruct], isLess: fun(Int, Int): Bool) {

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
				// NO-OP
			},
			emitEvent: func(event cadence.Event) error {
				// NO-OP
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

		storage, _, _ = rt.Storage(Context{
			Interface: runtimeInterface,
		})

		return
	}

	testAccount := func(accountType sema.Type, accountExpression string) {

		testName := fmt.Sprintf(
			"%s.Capabilities",
			accountType.String(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("get non-existing", func(t *testing.T) {

				t.Parallel()

				err, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                            transaction {
                                prepare(signer: AuthAccount) {
                                    // Act
                                    let gotCap: Capability<&AnyStruct>? =
                                        %s.capabilities.get<&AnyStruct>(/public/x)

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

			t.Run("get and check existing, with valid type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _ := test(
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
                                      let gotCap: Capability<&Test.R> =
                                          %s.capabilities.get<&Test.R>(publicPath)!

                                      // Assert
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.check())
                                      assert(gotCap.id == expectedCapID)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)
				})

				t.Run("account capability", func(t *testing.T) {
					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&AuthAccount> =
                                          %s.capabilities.get<&AuthAccount>(publicPath)!

                                      // Assert
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.check())
                                      assert(gotCap.id == expectedCapID)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)
				})
			})

			t.Run("get, borrow, and check existing, with valid type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _ := test(
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
                                      let gotCap: Capability<&Test.R> =
                                          %s.capabilities.get<&Test.R>(publicPath)!
                                      let ref: &Test.R = gotCap.borrow()!

                                      // Assert
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.check())
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

				t.Run("account capability", func(t *testing.T) {

					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&AuthAccount> =
                                          %s.capabilities.get<&AuthAccount>(publicPath)!
                                      let ref: &AuthAccount = gotCap.borrow()!

                                      // Assert
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.check())
                                      assert(gotCap.id == expectedCapID)
                                      assert(ref.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)
				})
			})

			t.Run("get existing, with subtype", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _ := test(
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
                                      let gotCap: Capability<auth(Test.X) &Test.R>? =
                                          %s.capabilities.get<auth(Test.X) &Test.R>(publicPath)

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

				t.Run("account capability", func(t *testing.T) {

					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount{}> =
                                          signer.capabilities.account.issue<&AuthAccount{}>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&Test.R>? =
                                          %s.capabilities.get<&Test.R>(publicPath)

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
			})

			t.Run("get existing, with different type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _ := test(
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
                                      let gotCap: Capability<&Test.S>? =
                                          %s.capabilities.get<&Test.S>(publicPath)

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

				t.Run("account capability", func(t *testing.T) {

					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&AnyResource>? =
                                          %s.capabilities.get<&AnyResource>(publicPath)

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
			})

			t.Run("get unpublished", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _ := test(
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
                                      let gotCap: Capability<&Test.R>? =
                                          %s.capabilities.get<&Test.R>(publicPath)

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

				t.Run("account capability", func(t *testing.T) {
					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)
                                      let unpublishedcap = signer.capabilities.unpublish(publicPath)

                                      // Act
                                      let gotCap: Capability<&AuthAccount>? =
                                          %s.capabilities.get<&AuthAccount>(publicPath)

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

			})

			t.Run("borrow non-existing", func(t *testing.T) {

				t.Parallel()

				err, _ := test(
					fmt.Sprintf(
						// language=cadence
						`
                        transaction {
                            prepare(signer: AuthAccount) {
                                // Act
                                let ref: &AnyStruct? =
                                    %s.capabilities.borrow<&AnyStruct>(/public/x)

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

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _ := test(
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
                                      let ref: &Test.R =
                                          %s.capabilities.borrow<&Test.R>(publicPath)!

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

				t.Run("account capability", func(t *testing.T) {
					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let ref: &AuthAccount =
                                          %s.capabilities.borrow<&AuthAccount>(publicPath)!

                                      // Assert
                                      assert(issuedCap.id == expectedCapID)
                                      assert(ref.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)
				})
			})

			t.Run("borrow existing, with subtype", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _ := test(
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
                                      let ref: auth(Test.X) &Test.R? =
                                          %s.capabilities.borrow<auth(Test.X) &Test.R>(publicPath)

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

				t.Run("account capability", func(t *testing.T) {

					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1 

                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let ref: auth(Test.X) &AuthAccount? =
                                          %s.capabilities.borrow<auth(Test.X) &AuthAccount>(publicPath)

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
			})

			t.Run("borrow existing, with different type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _ := test(
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
                                      let ref: &Test.S? =
                                          %s.capabilities.borrow<&Test.S>(publicPath)

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

				t.Run("account capability", func(t *testing.T) {

					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let ref: &AnyResource? =
                                          %s.capabilities.borrow<&AnyResource>(publicPath)

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
			})

			t.Run("borrow unpublished", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _ := test(
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
                                      let ref: &Test.R? =
                                          %s.capabilities.borrow<&Test.R>(publicPath)

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

				t.Run("account capability", func(t *testing.T) {
					err, _ := test(
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)
                                      let unpublishedcap = signer.capabilities.unpublish(publicPath)

                                      // Act
                                      let ref: &AuthAccount? =
                                          %s.capabilities.borrow<&AuthAccount>(publicPath)

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
			})

			if accountType == sema.AuthAccountType {

				t.Run("publish, existing published", func(t *testing.T) {

					t.Parallel()

					t.Run("storage capability", func(t *testing.T) {
						err, _ := test(
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

					t.Run("storage capability", func(t *testing.T) {
						err, _ := test(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: AuthAccount) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&AuthAccount> =
                                          signer.capabilities.account.issue<&AuthAccount>()
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
				})

				t.Run("unpublish non-existing", func(t *testing.T) {

					t.Parallel()

					err, _ := test(
						// language=cadence
						`
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

			err, _ := test(
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

			err, _ := test(
				// language=cadence
				`
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

		t.Run("getController, account capability controller", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          let issuedCap: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()

                          // Act
                          let controller: &StorageCapabilityController? =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                          // Assert
                          assert(controller == nil)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("getController, multiple controllers to various paths, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
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

			err, _ := test(
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

		t.Run("forEachController, no controllers", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath = /storage/r

                          // Act
                          var called = false
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath,
                              fun (controller: &StorageCapabilityController): Bool {
                                  called = true
                                  return true
                              }
                          )

                          // Assert
                          assert(!called)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, all", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
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

			err, _ := test(
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

		t.Run("forEachController, mutation (issue), stop", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath = /storage/r

                          // Arrange
                          signer.capabilities.storage.issue<&Test.R>(storagePath)

                          // Act
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath,
                              fun (controller: &StorageCapabilityController): Bool {

                                  signer.capabilities.storage.issue<&Test.R>(storagePath)

                                  return false
                              }
                          )
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, mutation (issue), continue", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath = /storage/r

                          // Arrange
                          signer.capabilities.storage.issue<&Test.R>(storagePath)

                          // Act
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath,
                              fun (controller: &StorageCapabilityController): Bool {

                                  signer.capabilities.storage.issue<&Test.R>(storagePath)

                                  return true
                              }
                          )
                      }
                  }
                `,
			)
			RequireError(t, err)

			var mutationErr stdlib.CapabilityControllersMutatedDuringIterationError
			require.ErrorAs(t, err, &mutationErr)
		})

		t.Run("forEachController, mutation (delete), stop", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath = /storage/r

                          // Arrange
                          signer.capabilities.storage.issue<&Test.R>(storagePath)

                          // Act
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath,
                              fun (controller: &StorageCapabilityController): Bool {

                                  controller.delete()

                                  return false
                              }
                          )
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, mutation (delete), continue", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath = /storage/r

                          // Arrange
                          signer.capabilities.storage.issue<&Test.R>(storagePath)

                          // Act
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath,
                              fun (controller: &StorageCapabilityController): Bool {

                                  controller.delete()

                                  return true
                              }
                          )
                      }
                  }
                `,
			)
			RequireError(t, err)

			var mutationErr stdlib.CapabilityControllersMutatedDuringIterationError
			require.ErrorAs(t, err, &mutationErr)
		})
	})

	t.Run("AuthAccount.AccountCapabilities", func(t *testing.T) {

		t.Parallel()

		t.Run("issue, multiple controllers, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Act
                          let issuedCap1: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap2: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap3: Capability<&AuthAccount{}> =
                              signer.capabilities.account.issue<&AuthAccount{}>()

                          // Assert
                          assert(issuedCap1.id == 1)
                          assert(issuedCap2.id == 2)
                          assert(issuedCap3.id == 3)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("getController, non-existing", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Act
                          let controller1: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: 0)
                          let controller2: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: 1)

                          // Assert
                          assert(controller1 == nil)
                          assert(controller2 == nil)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("getController, storage capability controller", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          let issuedCap: Capability<&AnyStruct> =
                              signer.capabilities.storage.issue<&AnyStruct>(/storage/x)

                          // Act
                          let controller: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap.id)

                          // Assert
                          assert(controller == nil)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("getController, multiple controllers to various paths, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          let issuedCap1: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap2: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap3: Capability<&AuthAccount{}> =
                              signer.capabilities.account.issue<&AuthAccount{}>()

                          // Act
                          let controller1: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap1.id)
                          let controller2: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap2.id)
                          let controller3: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap3.id)

                          // Assert
                          assert(controller1!.capabilityID == 1)
                          assert(controller1!.borrowType == Type<&AuthAccount>())

                          assert(controller2!.capabilityID == 2)
                          assert(controller2!.borrowType == Type<&AuthAccount>())

                          assert(controller3!.capabilityID == 3)
                          assert(controller3!.borrowType == Type<&AuthAccount{}>())
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("getControllers", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {

                          // Arrange
                          let issuedCap1: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap2: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap3: Capability<&AuthAccount{}> =
                              signer.capabilities.account.issue<&AuthAccount{}>()

                          // Act
                          let controllers: [&AccountCapabilityController] =
                              signer.capabilities.account.getControllers()

                          // Assert
                          assert(controllers.length == 3)

                          Test.quickSort(
                              &controllers as &[AnyStruct],
                              isLess: fun(i: Int, j: Int): Bool {
                                  let a = controllers[i]
                                  let b = controllers[j]
                                  return a.capabilityID < b.capabilityID
                              }
                          )

                          assert(controllers[0].capabilityID == 1)
                          assert(controllers[1].capabilityID == 2)
                          assert(controllers[2].capabilityID == 3)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, no controllers", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {

                          // Act
                          var called = false
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {
                                  called = true
                                  return true
                              }
                          )

                          // Assert
                          assert(!called)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, all", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          let issuedCap1: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap2: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap3: Capability<&AuthAccount{}> =
                              signer.capabilities.account.issue<&AuthAccount{}>()

                          // Act
                          let controllers: [&AccountCapabilityController] = []
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {
                                  controllers.append(controller)
                                  return true
                              }
                          )

                          // Assert
                          assert(controllers.length == 3)

                          Test.quickSort(
                              &controllers as &[AnyStruct],
                              isLess: fun(i: Int, j: Int): Bool {
                                  let a = controllers[i]
                                  let b = controllers[j]
                                  return a.capabilityID < b.capabilityID
                              }
                          )

                          assert(controllers[0].capabilityID == 1)
                          assert(controllers[1].capabilityID == 2)
                          assert(controllers[2].capabilityID == 3)
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, stop immediately", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          let issuedCap1: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let issuedCap2: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()

                          // Act
                          var stopped = false
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {
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

		t.Run("forEachController, mutation (issue), continue", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          signer.capabilities.account.issue<&AuthAccount>()

                          // Act
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {

                                  signer.capabilities.account.issue<&AuthAccount>()

                                  return true
                              }
                          )
                      }
                  }
                `,
			)
			RequireError(t, err)

			var mutationErr stdlib.CapabilityControllersMutatedDuringIterationError
			require.ErrorAs(t, err, &mutationErr)
		})

		t.Run("forEachController, mutation (issue), stop", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          signer.capabilities.account.issue<&AuthAccount>()

                          // Act
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {

                                  signer.capabilities.account.issue<&AuthAccount>()

                                  return false
                              }
                          )
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("forEachController, mutation (delete), continue", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          signer.capabilities.account.issue<&AuthAccount>()

                          // Act
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {

                                  controller.delete()

                                  return true
                              }
                          )
                      }
                  }
                `,
			)
			RequireError(t, err)

			var mutationErr stdlib.CapabilityControllersMutatedDuringIterationError
			require.ErrorAs(t, err, &mutationErr)
		})

		t.Run("forEachController, mutation (delete), stop", func(t *testing.T) {

			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          signer.capabilities.account.issue<&AuthAccount>()

                          // Act
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {

                                  controller.delete()

                                  return false
                              }
                          )
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})
	})

	t.Run("StorageCapabilityController", func(t *testing.T) {

		t.Parallel()

		t.Run("tag", func(t *testing.T) {
			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: AuthAccount) {
                          let storagePath = /storage/r

                          // Arrange
                          let issuedCap: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath)
                          let controller1: &StorageCapabilityController =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)!
                          let controller2: &StorageCapabilityController =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)!

                          assert(controller1.tag == "")
                          assert(controller2.tag == "")

                          // Act
                          controller1.setTag("something")

                          // Assert
                          let controller3: &StorageCapabilityController =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)!

                          assert(controller1.tag == "something")
                          assert(controller2.tag == "something")
                          assert(controller3.tag == "something")
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("retarget", func(t *testing.T) {

			t.Parallel()

			t.Run("target, getControllers", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
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

                              let controllers1Before = signer.capabilities.storage.getControllers(forPath: storagePath1)
                              Test.quickSort(
                                  &controllers1Before as &[AnyStruct],
                                  isLess: fun(i: Int, j: Int): Bool {
                                      let a = controllers1Before[i]
                                      let b = controllers1Before[j]
                                      return a.capabilityID < b.capabilityID
                                  }
                              )
                              assert(controllers1Before.length == 3)
                              assert(controllers1Before[0].capabilityID == 1)
                              assert(controllers1Before[1].capabilityID == 2)
                              assert(controllers1Before[2].capabilityID == 3)

                              let controllers2Before = signer.capabilities.storage.getControllers(forPath: storagePath2)
                              Test.quickSort(
                                  &controllers2Before as &[AnyStruct],
                                  isLess: fun(i: Int, j: Int): Bool {
                                      let a = controllers2Before[i]
                                      let b = controllers2Before[j]
                                      return a.capabilityID < b.capabilityID
                                  }
                              )
                              assert(controllers2Before.length == 1)
                              assert(controllers2Before[0].capabilityID == 4)

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

                              let controllers1After = signer.capabilities.storage.getControllers(forPath: storagePath1)
                              Test.quickSort(
                                  &controllers1After as &[AnyStruct],
                                  isLess: fun(i: Int, j: Int): Bool {
                                      let a = controllers1After[i]
                                      let b = controllers1After[j]
                                      return a.capabilityID < b.capabilityID
                                  }
                              )
                              assert(controllers1After.length == 2)
                              assert(controllers1After[0].capabilityID == 2)
                              assert(controllers1After[1].capabilityID == 3)

                              let controllers2After = signer.capabilities.storage.getControllers(forPath: storagePath2)
                              Test.quickSort(
                                  &controllers2After as &[AnyStruct],
                                  isLess: fun(i: Int, j: Int): Bool {
                                      let a = controllers2After[i]
                                      let b = controllers2After[j]
                                      return a.capabilityID < b.capabilityID
                                  }
                              )
                              assert(controllers2After.length == 2)
                              assert(controllers2After[0].capabilityID == 1)
                              assert(controllers2After[1].capabilityID == 4)
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})

			t.Run("retarget empty, borrow", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath1 = /storage/r
                              let storagePath2 = /storage/empty
                              let resourceID = 42

                              // Arrange
                              Test.createAndSaveR(id: resourceID, storagePath: storagePath1)

                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath1)
                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              assert(issuedCap.borrow() != nil)
                              assert(issuedCap.check())
                              assert(issuedCap.borrow()!.id == resourceID)

                              // Act
                              controller!.retarget(storagePath2)

                              // Assert
                              assert(issuedCap.borrow() == nil)
                              assert(!issuedCap.check())
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})

			t.Run("retarget to value with same type, borrow", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath1 = /storage/r
                              let storagePath2 = /storage/r2
                              let resourceID1 = 42
                              let resourceID2 = 43

                              // Arrange
                              Test.createAndSaveR(id: resourceID1, storagePath: storagePath1)
                              Test.createAndSaveR(id: resourceID2, storagePath: storagePath2)

                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath1)
                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              assert(issuedCap.borrow() != nil)
                              assert(issuedCap.check())
                              assert(issuedCap.borrow()!.id == resourceID1)

                              // Act
                              controller!.retarget(storagePath2)

                              // Assert
                              assert(issuedCap.borrow() != nil)
                              assert(issuedCap.check())
                              assert(issuedCap.borrow()!.id == resourceID2)
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})

			t.Run("retarget to value with different type, borrow", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath1 = /storage/r
                              let storagePath2 = /storage/s
                              let resourceID = 42

                              // Arrange
                              Test.createAndSaveR(id: resourceID, storagePath: storagePath1)
                              Test.createAndSaveS(storagePath: storagePath2)

                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath1)
                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              assert(issuedCap.borrow() != nil)
                              assert(issuedCap.check())
                              assert(issuedCap.borrow()!.id == resourceID)

                              // Act
                              controller!.retarget(storagePath2)

                              // Assert
                              assert(issuedCap.borrow() == nil)
                              assert(!issuedCap.check())
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})
		})

		t.Run("delete", func(t *testing.T) {

			t.Parallel()

			t.Run("getController, getControllers", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath = /storage/r

                              // Arrange
                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath)
                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              let controllersBefore = signer.capabilities.storage.getControllers(forPath: storagePath)
                              assert(controllersBefore.length == 1)
                              assert(controllersBefore[0].capabilityID == 1)

                              // Act
                              controller!.delete()

                              // Assert
                              let controllerAfter: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)
                              assert(controllerAfter == nil)

                              let controllersAfter = signer.capabilities.storage.getControllers(forPath: storagePath)
                              assert(controllersAfter.length == 0)
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})

			t.Run("target", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath = /storage/r

                              // Arrange
                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath)
                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              // Act
                              controller!.delete()

                              // Assert
                              controller!.target()
                          }
                      }
                    `,
				)
				require.ErrorContains(t, err, "controller is deleted")
			})

			t.Run("retarget", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath = /storage/r

                              // Arrange
                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath)
                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              // Act
                              controller!.delete()

                              // Assert
                              controller!.retarget(/storage/r2)
                          }
                      }
                    `,
				)
				require.ErrorContains(t, err, "controller is deleted")
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath = /storage/r

                              // Arrange
                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath)
                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              // Act
                              controller!.delete()

                              // Assert
                              controller!.delete()
                          }
                      }
                    `,
				)
				require.ErrorContains(t, err, "controller is deleted")
			})

			t.Run("capability set cleared from storage", func(t *testing.T) {
				t.Parallel()

				err, storage := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath = /storage/r

                              // Arrange
                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath)
                              let controller: &StorageCapabilityController =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)!

                              // Act
                              controller.delete()
                          }
                      }
                    `,
				)
				require.NoError(t, err)

				storageMap := storage.GetStorageMap(
					common.MustBytesToAddress([]byte{0x1}),
					stdlib.PathCapabilityStorageDomain,
					false,
				)
				require.Zero(t, storageMap.Count())

			})

			t.Run("check, borrow", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: AuthAccount) {
                              let storagePath = /storage/r
							  let resourceID = 42

                              // Arrange
							  Test.createAndSaveR(id: resourceID, storagePath: storagePath)
                              let issuedCap: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath)
                              assert(issuedCap.check())
                              assert(issuedCap.borrow() != nil)

                              let controller: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)

                              // Act
                              controller!.delete()

                              // Assert
                              assert(!issuedCap.check())
                              assert(issuedCap.borrow() == nil)
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})

		})
	})

	t.Run("AccountCapabilityController", func(t *testing.T) {

		t.Parallel()

		t.Run("tag", func(t *testing.T) {
			t.Parallel()

			err, _ := test(
				// language=cadence
				`
                  transaction {
                      prepare(signer: AuthAccount) {
                          // Arrange
                          let issuedCap: Capability<&AuthAccount> =
                              signer.capabilities.account.issue<&AuthAccount>()
                          let controller1: &AccountCapabilityController =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap.id)!
                          let controller2: &AccountCapabilityController =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap.id)!

                          assert(controller1.tag == "")
                          assert(controller2.tag == "")

                          // Act
                          controller1.setTag("something")

                          // Assert
                          let controller3: &AccountCapabilityController =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap.id)!

                          assert(controller1.tag == "something")
                          assert(controller2.tag == "something")
                          assert(controller3.tag == "something")
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})

		t.Run("delete", func(t *testing.T) {

			t.Parallel()

			t.Run("getController, getControllers", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      transaction {
                          prepare(signer: AuthAccount) {
                              // Arrange
                              let issuedCap: Capability<&AuthAccount> =
                                  signer.capabilities.account.issue<&AuthAccount>()
                              let controller: &AccountCapabilityController? =
                                  signer.capabilities.account.getController(byCapabilityID: issuedCap.id)

                              let controllersBefore = signer.capabilities.account.getControllers()
                              assert(controllersBefore.length == 1)
                              assert(controllersBefore[0].capabilityID == 1)

                              // Act
                              controller!.delete()

                              // Assert
                              let controllerAfter: &AccountCapabilityController? =
                                  signer.capabilities.account.getController(byCapabilityID: issuedCap.id)
                              assert(controllerAfter == nil)

                              let controllersAfter = signer.capabilities.account.getControllers()
                              assert(controllersAfter.length == 0)
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      transaction {
                          prepare(signer: AuthAccount) {
                              // Arrange
                              let issuedCap: Capability<&AuthAccount> =
                                  signer.capabilities.account.issue<&AuthAccount>()
                              let controller: &AccountCapabilityController? =
                                  signer.capabilities.account.getController(byCapabilityID: issuedCap.id)

                              // Act
                              controller!.delete()

                              // Assert
                              controller!.delete()
                          }
                      }
                    `,
				)
				require.ErrorContains(t, err, "controller is deleted")
			})

			t.Run("check, borrow", func(t *testing.T) {
				t.Parallel()

				err, _ := test(
					// language=cadence
					`
                      transaction {
                          prepare(signer: AuthAccount) {
                              // Arrange
                              let issuedCap: Capability<&AuthAccount> =
                                  signer.capabilities.account.issue<&AuthAccount>()
                              assert(issuedCap.check())
                              assert(issuedCap.borrow() != nil)

                              let controller: &AccountCapabilityController? =
                                  signer.capabilities.account.getController(byCapabilityID: issuedCap.id)

                              // Act
                              controller!.delete()

                              // Assert
                              assert(!issuedCap.check())
                              assert(issuedCap.borrow() == nil)
                          }
                      }
                    `,
				)
				require.NoError(t, err)
			})
		})
	})

}
