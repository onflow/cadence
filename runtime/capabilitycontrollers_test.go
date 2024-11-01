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
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestRuntimeCapabilityControllers(t *testing.T) {
	t.Parallel()

	testWithSignerCount := func(t *testing.T, tx string, signerCount int) (
		err error,
		storage *Storage,
		events []cadence.Event,
	) {

		rt := NewTestInterpreterRuntime()

		accountCodes := map[Location][]byte{}

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
                          self.account.storage.save(
                              <-create R(id: id),
                              to: storagePath
                          )
                      }

                      access(all) fun createAndSaveS(storagePath: StoragePath) {
                          self.account.storage.save(
                              <-create S(),
                              to: storagePath
                          )
                      }

                      /// quickSort is qsort from "The C Programming Language".
                      ///
                      /// > Our version of quicksort is not the fastest possible,
                      /// > but it's one of the simplest.
                      ///
                      access(all) fun quickSort(_ items: auth(Mutate) &[AnyStruct], isLess: fun(Int, Int): Bool) {

                          fun quickSortPart(leftIndex: Int, rightIndex: Int) {

                              if leftIndex >= rightIndex {
                                  return
                              }

                              let pivotIndex = (leftIndex + rightIndex) / 2

                              items[pivotIndex] <-> items[leftIndex]
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

		if signerCount < 1 {
			signerCount = 1
		}

		testSigners := make([]Address, signerCount)
		for signerIndex := 0; signerIndex < signerCount; signerIndex++ {
			binary.BigEndian.PutUint64(
				testSigners[signerIndex][:],
				uint64(signerIndex+1),
			)
		}

		signers := []Address{testSigners[0]}

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnProgramLog: func(message string) {
				// NO-OP
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
			OnGetSigningAccounts: func() ([]Address, error) {
				return signers, nil
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
		}

		nextTransactionLocation := NewTransactionLocationGenerator()

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

		signers = testSigners

		err = rt.ExecuteTransaction(
			Script{
				Source: []byte(tx),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextTransactionLocation(),
			},
		)

		var storageErr error
		storage, _, storageErr = rt.Storage(Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, storageErr)

		return
	}

	test := func(t *testing.T, tx string) (
		err error,
		storage *Storage,
		events []cadence.Event,
	) {
		return testWithSignerCount(t, tx, 1)
	}

	authAccountType := sema.FullyEntitledAccountReferenceType
	publicAccountType := sema.AccountReferenceType

	testAccount := func(accountType sema.Type, accountExpression string) {

		testName := fmt.Sprintf(
			"%s.Capabilities",
			accountType.String(),
		)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			t.Run("get non-existing", func(t *testing.T) {

				t.Parallel()

				err, _, events := test(
					t,
					fmt.Sprintf(
						// language=cadence
						`
                            transaction {
                                prepare(signer: auth(Capabilities) &Account) {
                                    let path = /public/x

                                    // Act
                                    let gotCap: Capability<&AnyStruct> =
                                        %[1]s.capabilities.get<&AnyStruct>(path)

                                    // Assert
                                    assert(!%[1]s.capabilities.exists(path))
                                    assert(gotCap.id == 0)
                                    assert(gotCap.borrow() == nil)
                                    assert(gotCap.check() == false)
                                    assert(gotCap.address == 0x1)
                                }
                            }
                        `,
						accountExpression,
					),
				)
				require.NoError(t, err)

				require.Equal(t,
					[]string{},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("get and check existing, with valid type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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
                                          %[1]s.capabilities.get<&Test.R>(publicPath)

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
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

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&Account> =
                                          %[1]s.capabilities.get<&Account>(publicPath)

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
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

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events))
				})
			})

			t.Run("get, borrow, and check existing, with valid type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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
                                          %[1]s.capabilities.get<&Test.R>(publicPath)
                                      let ref: &Test.R = gotCap.borrow()!

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
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

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events))
				})

				t.Run("account capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&Account> =
                                          %[1]s.capabilities.get<&Account>(publicPath)
                                      let ref: &Account = gotCap.borrow()!

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
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

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})
			})

			t.Run("get existing, with subtype", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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
                                      let gotCap: Capability<auth(Test.X) &Test.R> =
                                          %[1]s.capabilities.get<auth(Test.X) &Test.R>(publicPath)

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.id == 0)
                                      assert(gotCap.borrow() == nil)
                                      assert(gotCap.check() == false)
                                      assert(gotCap.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&Test.R> =
                                          %[1]s.capabilities.get<&Test.R>(publicPath)

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.id == 0)
                                      assert(gotCap.borrow() == nil)
                                      assert(gotCap.check() == false)
                                      assert(gotCap.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})
			})

			t.Run("get existing, with different type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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
                                      let gotCap: Capability<&Test.S> =
                                          %[1]s.capabilities.get<&Test.S>(publicPath)

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.id == 0)
                                      assert(gotCap.borrow() == nil)
                                      assert(gotCap.check() == false)
                                      assert(gotCap.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let gotCap: Capability<&AnyResource> =
                                          %[1]s.capabilities.get<&AnyResource>(publicPath)

                                      // Assert
                                      assert(%[1]s.capabilities.exists(publicPath))
                                      assert(issuedCap.id == expectedCapID)
                                      assert(gotCap.id == 0)
                                      assert(gotCap.borrow() == nil)
                                      assert(gotCap.check() == false)
                                      assert(gotCap.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})
			})

			t.Run("get unpublished", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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
                                      let gotCap: Capability<&Test.R> =
                                          %[1]s.capabilities.get<&Test.R>(publicPath)

                                      // Assert
                                      assert(!%[1]s.capabilities.exists(publicPath))
                                      assert(issuedCap.id == expectedCapID)
                                      assert(unpublishedcap!.id == expectedCapID)
                                      assert(gotCap.id == 0)
                                      assert(gotCap.borrow() == nil)
                                      assert(gotCap.check() == false)
                                      assert(gotCap.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
							`flow.CapabilityUnpublished(address: 0x0000000000000001, path: /public/r)`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)
                                      let unpublishedcap = signer.capabilities.unpublish(publicPath)

                                      // Act
                                      let gotCap: Capability<&Account> =
                                          %[1]s.capabilities.get<&Account>(publicPath)

                                      // Assert
                                      assert(!%[1]s.capabilities.exists(publicPath))
                                      assert(issuedCap.id == expectedCapID)
                                      assert(unpublishedcap!.id == expectedCapID)
                                      assert(gotCap.id == 0)
                                      assert(gotCap.borrow() == nil)
                                      assert(gotCap.check() == false)
                                      assert(gotCap.address == 0x1)
                                  }
                              }
                            `,
							accountExpression,
						),
					)
					require.NoError(t, err)

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
							`flow.CapabilityUnpublished(address: 0x0000000000000001, path: /public/acct)`,
						},
						nonDeploymentEventStrings(events),
					)
				})

			})

			t.Run("borrow non-existing", func(t *testing.T) {

				t.Parallel()

				err, _, events := test(
					t,
					fmt.Sprintf(
						// language=cadence
						`
                        transaction {
                            prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("borrow existing, with valid type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let ref: &Account =
                                          %s.capabilities.borrow<&Account>(publicPath)!

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

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})
			})

			t.Run("borrow existing, with subtype", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)

                                      // Act
                                      let ref: auth(Test.X) &Account? =
                                          %s.capabilities.borrow<auth(Test.X) &Account>(publicPath)

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

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})
			})

			t.Run("borrow existing, with different type", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {

					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
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

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
						},
						nonDeploymentEventStrings(events),
					)
				})
			})

			t.Run("borrow unpublished", func(t *testing.T) {

				t.Parallel()

				t.Run("storage capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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

					require.Equal(t,
						[]string{
							`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
							`flow.CapabilityUnpublished(address: 0x0000000000000001, path: /public/r)`,
						},
						nonDeploymentEventStrings(events),
					)
				})

				t.Run("account capability", func(t *testing.T) {
					err, _, events := test(
						t,
						fmt.Sprintf(
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
                                      signer.capabilities.publish(issuedCap, at: publicPath)
                                      let unpublishedcap = signer.capabilities.unpublish(publicPath)

                                      // Act
                                      let ref: &Account? =
                                          %s.capabilities.borrow<&Account>(publicPath)

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

					require.Equal(t,
						[]string{
							`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
							`flow.CapabilityUnpublished(address: 0x0000000000000001, path: /public/acct)`,
						},
						nonDeploymentEventStrings(events),
					)
				})
			})

			if accountType == authAccountType {

				t.Run("publish, existing published", func(t *testing.T) {

					t.Parallel()

					t.Run("storage capability", func(t *testing.T) {
						err, _, events := test(
							t,
							// language=cadence
							`
                              import Test from 0x1

                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
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

						require.Equal(t,
							[]string{
								`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
								`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/r, capability: Capability<&A.0000000000000001.Test.R>(address: 0x0000000000000001, id: 1))`,
							},
							nonDeploymentEventStrings(events),
						)
					})

					t.Run("account capability", func(t *testing.T) {
						err, _, events := test(
							t,
							// language=cadence
							`
                              transaction {
                                  prepare(signer: auth(Capabilities) &Account) {
                                      let publicPath = /public/acct
                                      let expectedCapID: UInt64 = 1

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer.capabilities.account.issue<&Account>()
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

						require.Equal(t,
							[]string{
								`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
								`flow.CapabilityPublished(address: 0x0000000000000001, path: /public/acct, capability: Capability<&Account>(address: 0x0000000000000001, id: 1))`,
							},
							nonDeploymentEventStrings(events),
						)
					})
				})

				t.Run("publish different account", func(t *testing.T) {

					t.Parallel()

					t.Run("storage capability", func(t *testing.T) {

						err, _, events := testWithSignerCount(
							t,
							// language=cadence
							`
                               transaction {
                                   prepare(
                                       signer1: auth(Capabilities) &Account,
                                       signer2: auth(Capabilities) &Account
                                   ) {
                                       let publicPath = /public/r
                                       let storagePath = /storage/r

                                       // Arrange
                                       let issuedCap: Capability<&AnyStruct> =
                                           signer1.capabilities.storage.issue<&AnyStruct>(storagePath)

                                       // Act
                                       signer2.capabilities.publish(issuedCap, at: publicPath)
                                   }
                               }
                             `,
							2,
						)
						RequireError(t, err)

						var publishingError interpreter.CapabilityAddressPublishingError
						require.ErrorAs(t, err, &publishingError)
						assert.Equal(t,
							interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x2}),
							publishingError.AccountAddress,
						)
						assert.Equal(t,
							interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
							publishingError.CapabilityAddress,
						)

						require.Equal(
							t,
							[]string{
								`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&AnyStruct>(), path: /storage/r)`,
							},
							nonDeploymentEventStrings(events),
						)
					})

					t.Run("account capability", func(t *testing.T) {

						err, _, events := testWithSignerCount(
							t,
							// language=cadence
							`
                              transaction {
                                  prepare(
                                      signer1: auth(Capabilities) &Account,
                                      signer2: auth(Capabilities) &Account
                                  ) {
                                      let publicPath = /public/r

                                      // Arrange
                                      let issuedCap: Capability<&Account> =
                                          signer1.capabilities.account.issue<&Account>()

                                      // Act
                                      signer2.capabilities.publish(issuedCap, at: publicPath)
                                  }
                              }
                            `,
							2,
						)
						RequireError(t, err)

						var publishingError interpreter.CapabilityAddressPublishingError
						require.ErrorAs(t, err, &publishingError)
						assert.Equal(t,
							interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x2}),
							publishingError.AccountAddress,
						)
						assert.Equal(t,
							interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
							publishingError.CapabilityAddress,
						)

						require.Equal(t,
							[]string{
								`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
							},
							nonDeploymentEventStrings(events),
						)
					})
				})

				t.Run("unpublish non-existing", func(t *testing.T) {

					t.Parallel()

					err, _, events := test(
						t,
						// language=cadence
						`
                          transaction {
                              prepare(signer: auth(Capabilities) &Account) {
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

					require.Equal(t,
						[]string{},
						nonDeploymentEventStrings(events),
					)
				})
			}
		})
	}

	for accountType, accountExpression := range map[sema.Type]string{
		authAccountType:   "signer",
		publicAccountType: "getAccount(0x1)",
	} {
		testAccount(accountType, accountExpression)
	}

	t.Run("Account.StorageCapabilities", func(t *testing.T) {

		t.Parallel()

		t.Run("issue, multiple controllers to various paths, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Act
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap4: Capability<&Test.R> =
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

			require.Equal(
				t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 4, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r2)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("issue, multiple controllers to various paths, with same or different type, type value", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Act
                          let issuedCap1: Capability<&Test.R> = signer.capabilities.storage
                              .issueWithType(storagePath1, type: Type<&Test.R>()) as! Capability<&Test.R>
                          let issuedCap2: Capability<&Test.R> = signer.capabilities.storage
                              .issueWithType(storagePath1, type: Type<&Test.R>()) as! Capability<&Test.R>
                          let issuedCap3: Capability<&Test.R> = signer.capabilities.storage
                              .issueWithType(storagePath1, type: Type<&Test.R>()) as! Capability<&Test.R>
                          let issuedCap4: Capability<&Test.R> = signer.capabilities.storage
                              .issueWithType(storagePath2, type: Type<&Test.R>()) as! Capability<&Test.R>

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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 4, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r2)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("issue with type value, invalid type", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          signer.capabilities.storage.issueWithType(/storage/test, type: Type<Int>())
                      }
                  }
                `,
			)
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.InvalidCapabilityIssueTypeError{})

			require.Equal(t,
				[]string{},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getController, non-existing", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getController, account capability controller", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          let issuedCap: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()

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

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getController, multiple controllers to various paths, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
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
                          assert(controller3!.borrowType == Type<&Test.R>())
                          assert(controller3!.target() == storagePath1)

                          assert(controller4!.capabilityID == 4)
                          assert(controller4!.borrowType == Type<&Test.R>())
                          assert(controller4!.target() == storagePath2)
                      }
                  }
                `,
			)
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 4, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r2)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getControllers", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
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
                              &controllers1 as auth(Mutate) &[AnyStruct],
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 4, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r2)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, no controllers", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, all", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          let storagePath1 = /storage/r
                          let storagePath2 = /storage/r2

                          // Arrange
                          let issuedCap1: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap2: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
                          let issuedCap3: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath1)
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
                              &controllers1 as auth(Mutate) &[AnyStruct],
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 4, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r2)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, stop immediately", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (issue), stop", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (issue), continue", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (delete), stop", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (delete), continue", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
					`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, box and convert argument", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          let storagePath = /storage/r

                          // Arrange
						  signer.capabilities.storage.issue<&Test.R>(storagePath)

                          // Act
                          var res: String? = nil
                          signer.capabilities.storage.forEachController(
                              forPath: storagePath,
                              // NOTE: The function has a parameter of type &StorageCapabilityController?
                              // instead of just &StorageCapabilityController
                              fun (controller: &StorageCapabilityController?): Bool {
                                  // The map should call Optional.map, not fail,
                                  // because path is PublicPath?, not PublicPath
                                  res = controller.map(fun(string: AnyStruct): String {
                                      return "Optional.map"
                                  })
                                  return true
                              }
                          )

                          // Assert
                          assert(res == "Optional.map")
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})
	})

	t.Run("Account.AccountCapabilities", func(t *testing.T) {

		t.Parallel()

		t.Run("issue, multiple controllers, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Act
                          let issuedCap1: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap2: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap3: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()

                          // Assert
                          assert(issuedCap1.id == 1)
                          assert(issuedCap2.id == 2)
                          assert(issuedCap3.id == 3)
                      }
                  }
                `,
			)
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("issue, multiple controllers, with same or different type, type value", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Act
                          let issuedCap1: Capability<&Account> = signer.capabilities.account
                              .issueWithType(Type<&Account>()) as! Capability<&Account>
                          let issuedCap2: Capability<&Account> = signer.capabilities.account
                              .issueWithType(Type<&Account>()) as! Capability<&Account>
                          let issuedCap3: Capability<&Account> = signer.capabilities.account
                              .issueWithType(Type<&Account>()) as! Capability<&Account>

                          // Assert
                          assert(issuedCap1.id == 1)
                          assert(issuedCap2.id == 2)
                          assert(issuedCap3.id == 3)
                      }
                  }
                `,
			)
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("issue with type value, invalid type", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          signer.capabilities.account.issueWithType(Type<Int>())
                      }
                  }
                `,
			)
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.InvalidCapabilityIssueTypeError{})

			require.Equal(t,
				[]string{},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getController, non-existing", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getController, storage capability controller", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&AnyStruct>(), path: /storage/x)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getController, multiple controllers to various paths, with same or different type", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          let issuedCap1: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap2: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap3: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()

                          // Act
                          let controller1: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap1.id)
                          let controller2: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap2.id)
                          let controller3: &AccountCapabilityController? =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap3.id)

                          // Assert
                          assert(controller1!.capabilityID == 1)
                          assert(controller1!.borrowType == Type<&Account>())

                          assert(controller2!.capabilityID == 2)
                          assert(controller2!.borrowType == Type<&Account>())

                          assert(controller3!.capabilityID == 3)
                          assert(controller3!.borrowType == Type<&Account>())
                      }
                  }
                `,
			)
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("getControllers", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {

                          // Arrange
                          let issuedCap1: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap2: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap3: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()

                          // Act
                          let controllers: [&AccountCapabilityController] =
                              signer.capabilities.account.getControllers()

                          // Assert
                          assert(controllers.length == 3)

                          Test.quickSort(
                              &controllers as auth(Mutate) &[AnyStruct],
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

			require.Equal(
				t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, no controllers", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {

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

			require.Equal(t,
				[]string{},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, all", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          let issuedCap1: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap2: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap3: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()

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
                              &controllers as auth(Mutate) &[AnyStruct],
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

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, stop immediately", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          let issuedCap1: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let issuedCap2: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()

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

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (issue), continue", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          signer.capabilities.account.issue<&Account>()

                          // Act
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {

                                  signer.capabilities.account.issue<&Account>()

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

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (issue), stop", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          signer.capabilities.account.issue<&Account>()

                          // Act
                          signer.capabilities.account.forEachController(
                              fun (controller: &AccountCapabilityController): Bool {

                                  signer.capabilities.account.issue<&Account>()

                                  return false
                              }
                          )
                      }
                  }
                `,
			)
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (delete), continue", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          signer.capabilities.account.issue<&Account>()

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

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, mutation (delete), stop", func(t *testing.T) {

			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          signer.capabilities.account.issue<&Account>()

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

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
					`flow.AccountCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("forEachController, box and convert argument", func(t *testing.T) {

			t.Parallel()

			err, _, _ := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
						  signer.capabilities.account.issue<&Account>()

                          // Act
                          var res: String? = nil
                          signer.capabilities.account.forEachController(
                              // NOTE: The function has a parameter of type &AccountCapabilityController?
                              // instead of just &AccountCapabilityController
                              fun (controller: &AccountCapabilityController?): Bool {
                                  // The map should call Optional.map, not fail,
                                  // because path is PublicPath?, not PublicPath
                                  res = controller.map(fun(string: AnyStruct): String {
                                      return "Optional.map"
                                  })
                                  return true
                              }
                          )

                          // Assert
                          assert(res == "Optional.map")
                      }
                  }
                `,
			)
			require.NoError(t, err)
		})
	})

	t.Run("StorageCapabilityController", func(t *testing.T) {

		t.Parallel()

		t.Run("capability", func(t *testing.T) {
			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Storage, Capabilities) &Account) {
                          let storagePath = /storage/r
                          let resourceID = 42

						  // Arrange
						  Test.createAndSaveR(id: resourceID, storagePath: storagePath)

                          let issuedCap: Capability<&Test.R> =
                              signer.capabilities.storage.issue<&Test.R>(storagePath)
                          let controller1: &StorageCapabilityController =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)!
                          let controller2: &StorageCapabilityController =
                              signer.capabilities.storage.getController(byCapabilityID: issuedCap.id)!

                          // Act
                          let controller1Cap = controller1.capability
                          let controller2Cap = controller2.capability

                          // Assert
                          assert(controller1Cap.borrow<&Test.R>() != nil)
                          assert(controller2Cap.borrow<&Test.R>() != nil)
                      }
                  }
                `,
			)
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("tag", func(t *testing.T) {
			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
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

			require.Equal(t,
				[]string{
					`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("retarget", func(t *testing.T) {

			t.Parallel()

			t.Run("target, getControllers", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

                              let issuedCap3: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath1)
                              let controller3: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap3.id)

                              let issuedCap4: Capability<&Test.R> =
                                  signer.capabilities.storage.issue<&Test.R>(storagePath2)
                              let controller4: &StorageCapabilityController? =
                                  signer.capabilities.storage.getController(byCapabilityID: issuedCap4.id)

                              let controllers1Before = signer.capabilities.storage.getControllers(forPath: storagePath1)
                              Test.quickSort(
                                  &controllers1Before as auth(Mutate) &[AnyStruct],
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
                                  &controllers2Before as auth(Mutate) &[AnyStruct],
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
                                  &controllers1After as auth(Mutate) &[AnyStruct],
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
                                  &controllers2After as auth(Mutate) &[AnyStruct],
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerIssued(id: 3, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerIssued(id: 4, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r2)`,
						`flow.StorageCapabilityControllerTargetChanged(id: 1, address: 0x0000000000000001, path: /storage/r2)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("retarget empty, borrow", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerTargetChanged(id: 1, address: 0x0000000000000001, path: /storage/empty)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("retarget to value with same type, borrow", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerTargetChanged(id: 1, address: 0x0000000000000001, path: /storage/r2)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("retarget to value with different type, borrow", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerTargetChanged(id: 1, address: 0x0000000000000001, path: /storage/s)`,
					},
					nonDeploymentEventStrings(events),
				)
			})
		})

		t.Run("delete", func(t *testing.T) {

			t.Parallel()

			t.Run("getController, getControllers", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("target", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("retarget", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("capability set cleared from storage", func(t *testing.T) {
				t.Parallel()

				err, storage, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)

			})

			t.Run("check, borrow", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      import Test from 0x1

                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
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

				require.Equal(t,
					[]string{
						`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&A.0000000000000001.Test.R>(), path: /storage/r)`,
						`flow.StorageCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

		})
	})

	t.Run("AccountCapabilityController", func(t *testing.T) {

		t.Parallel()

		t.Run("capability", func(t *testing.T) {
			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  import Test from 0x1

                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {

						  // Arrange
                          let issuedCap: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
                          let controller1: &AccountCapabilityController =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap.id)!
                          let controller2: &AccountCapabilityController =
                              signer.capabilities.account.getController(byCapabilityID: issuedCap.id)!

                          // Act
                          let controller1Cap = controller1.capability
                          let controller2Cap = controller2.capability

                          // Assert
                          assert(controller1Cap.borrow<&Account>() != nil)
                          assert(controller2Cap.borrow<&Account>() != nil)
                      }
                  }
                `,
			)
			require.NoError(t, err)

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("tag", func(t *testing.T) {
			t.Parallel()

			err, _, events := test(
				t,
				// language=cadence
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          // Arrange
                          let issuedCap: Capability<&Account> =
                              signer.capabilities.account.issue<&Account>()
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

			require.Equal(t,
				[]string{
					`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
				},
				nonDeploymentEventStrings(events),
			)
		})

		t.Run("delete", func(t *testing.T) {

			t.Parallel()

			t.Run("getController, getControllers", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
                              // Arrange
                              let issuedCap: Capability<&Account> =
                                  signer.capabilities.account.issue<&Account>()
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

				require.Equal(t,
					[]string{
						`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
						`flow.AccountCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("delete", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
                              // Arrange
                              let issuedCap: Capability<&Account> =
                                  signer.capabilities.account.issue<&Account>()
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

				require.Equal(t,
					[]string{
						`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
						`flow.AccountCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})

			t.Run("check, borrow", func(t *testing.T) {
				t.Parallel()

				err, _, events := test(
					t,
					// language=cadence
					`
                      transaction {
                          prepare(signer: auth(Capabilities) &Account) {
                              // Arrange
                              let issuedCap: Capability<&Account> =
                                  signer.capabilities.account.issue<&Account>()
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

				require.Equal(t,
					[]string{
						`flow.AccountCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&Account>())`,
						`flow.AccountCapabilityControllerDeleted(id: 1, address: 0x0000000000000001)`,
					},
					nonDeploymentEventStrings(events),
				)
			})
		})
	})

}

func nonDeploymentEventStrings(events []cadence.Event) []string {
	accountContractAddedEventTypeID := stdlib.AccountContractAddedEventType.ID()

	strings := make([]string, 0, len(events))
	for _, event := range events {
		// Skip deployment events, i.e. contract added to account
		if common.TypeID(event.Type().ID()) == accountContractAddedEventTypeID {
			continue
		}
		strings = append(strings, event.String())
	}
	return strings
}

func TestRuntimeCapabilityBorrowAsInheritedInterface(t *testing.T) {

	t.Parallel()

	runtime := NewTestInterpreterRuntime()

	contract := []byte(`
        access(all) contract Test {

            access(all) resource interface Balance {}

            access(all) resource interface Vault:Balance {}

            access(all) resource VaultImpl: Vault {}

            access(all) fun createVaultImpl(): @VaultImpl {
                return <- create VaultImpl()
            }
        }
    `)

	script := []byte(`
        import Test from 0x01

        transaction {
            prepare(acct: auth(Storage, Capabilities) &Account) {
                acct.storage.save(<-Test.createVaultImpl(), to: /storage/r)

                let cap = acct.capabilities.storage.issue<&{Test.Balance}>(/storage/r)
                acct.capabilities.publish(cap, at: /public/r)

                let vaultRef = acct.capabilities.borrow<&{Test.Balance}>(/public/r)
                    ?? panic("Could not borrow Balance reference to the Vault")
            }
        }
    `)

	deploy := DeploymentTransaction("Test", contract)

	address := common.MustBytesToAddress([]byte{0x1})

	var accountCode []byte

	runtimeInterface := &TestRuntimeInterface{
		OnGetCode: func(_ Location) (bytes []byte, err error) {
			return accountCode, nil
		},
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]Address, error) {
			return []Address{address}, nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(_ common.AddressLocation) (code []byte, err error) {
			return accountCode, nil
		},
		OnUpdateAccountContractCode: func(_ common.AddressLocation, code []byte) error {
			accountCode = code
			return nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
	}

	nextTransactionLocation := NewTransactionLocationGenerator()

	// Deploy

	err := runtime.ExecuteTransaction(
		Script{
			Source: deploy,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	// Test

	err = runtime.ExecuteTransaction(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)

	require.NoError(t, err)
}

func TestRuntimeCapabilityControllerOperationAfterDeletion(t *testing.T) {

	t.Parallel()

	type operation struct {
		name string
		code string
	}

	type testCase struct {
		name       string
		setup      string
		operations []operation
	}

	test := func(testCase testCase, operation operation) {

		testName := fmt.Sprintf("%s: %s", testCase.name, operation.name)

		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			rt := NewTestInterpreterRuntime()

			tx := []byte(fmt.Sprintf(
				`
                  transaction {
                      prepare(signer: auth(Capabilities) &Account) {
                          %s
                          %s
                      }
                  }
                `,
				testCase.setup,
				operation.code,
			))

			address := common.MustBytesToAddress([]byte{0x1})
			accountIDs := map[common.Address]uint64{}

			runtimeInterface := &TestRuntimeInterface{
				Storage: NewTestLedger(nil, nil),
				OnGetSigningAccounts: func() ([]Address, error) {
					return []Address{address}, nil
				},
				OnEmitEvent: func(event cadence.Event) error {
					return nil
				},
				OnGenerateAccountID: func(address common.Address) (uint64, error) {
					accountID := accountIDs[address] + 1
					accountIDs[address] = accountID
					return accountID, nil
				},
			}

			nextTransactionLocation := NewTransactionLocationGenerator()

			// Test

			err := rt.ExecuteTransaction(
				Script{
					Source: tx,
				},
				Context{
					Interface: runtimeInterface,
					Location:  nextTransactionLocation(),
				},
			)

			require.ErrorContains(t, err, "controller is deleted")
		})
	}

	testCases := []testCase{
		{
			name: "Storage capability controller",
			setup: `
              // Issue capability and get controller
              let storageCapabilities = signer.capabilities.storage
              let capability = storageCapabilities.issue<&AnyStruct>(/storage/test1)
              let controller = storageCapabilities.getController(byCapabilityID: capability.id)!

              // Prepare bound functions
              let delete = controller.delete
              let tag = controller.tag
              let setTag = controller.setTag
              let target = controller.target
              let retarget = controller.retarget

              // Delete
              controller.delete()
            `,
			operations: []operation{
				// Read
				{
					name: "get capability",
					code: `controller.capability`,
				},
				{
					name: "get tag",
					code: `controller.tag`,
				},
				{
					name: "get borrow type",
					code: `controller.borrowType`,
				},
				{
					name: "get ID",
					code: `controller.capabilityID`,
				},
				// Mutate
				{
					name: "delete",
					code: `delete()`,
				},
				{
					name: "set tag",
					code: `setTag("test")`,
				},
				{
					name: "target",
					code: `target()`,
				},
				{
					name: "retarget",
					code: `retarget(/storage/test2)`,
				},
			},
		},
		{
			name: "Account capability controller",
			setup: `
              // Issue capability and get controller
              let accountCapabilities = signer.capabilities.account
              let capability = accountCapabilities.issue<auth(Storage) &Account>()
              let controller = accountCapabilities.getController(byCapabilityID: capability.id)!

              // Prepare bound functions
              let delete = controller.delete
              let tag = controller.tag
              let setTag = controller.setTag

              // Delete
              controller.delete()
            `,
			operations: []operation{
				// Read
				{
					name: "get capability",
					code: `controller.capability`,
				},
				{
					name: "get tag",
					code: `controller.tag`,
				},
				{
					name: "get borrow type",
					code: `controller.borrowType`,
				},
				{
					name: "get ID",
					code: `controller.capabilityID`,
				},
				// Mutate
				{
					name: "delete",
					code: `delete()`,
				},
				{
					name: "set tag",
					code: `setTag("test")`,
				},
			},
		},
	}

	for _, testCase := range testCases {
		for _, operation := range testCase.operations {
			test(testCase, operation)
		}
	}
}

func TestRuntimeCapabilitiesGetBackwardCompatibility(t *testing.T) {
	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	test := func(t *testing.T, value interpreter.Value) {

		rt := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
		}

		storage, inter, err := rt.Storage(Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, err)

		publicStorageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainPublic.Identifier(),
			true,
		)

		publicStorageMap.SetValue(
			inter,
			interpreter.StringStorageMapKey("test"),
			value,
		)

		err = storage.Commit(inter, false)
		require.NoError(t, err)

		nextScriptLocation := NewScriptLocationGenerator()

		_, err = rt.ExecuteScript(
			Script{
				Source: []byte(`
                  access(all) fun main() {
                      let capabilities = getAccount(0x1).capabilities
                      let path = /public/test
                      assert(capabilities.get<&AnyStruct>(path).id == 0)
                      assert(capabilities.borrow<&AnyStruct>(path) == nil)
                  }
                `),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
			},
		)
		require.NoError(t, err)

	}

	t.Run("path capability, typed", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
			&interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.AddressValue(testAddress),
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		))
	})

	t.Run("path capability, untyped", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
			// NOTE: no borrow type
			nil,
			interpreter.AddressValue(testAddress),
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		))
	})

	t.Run("path link", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.PathLinkValue{ //nolint:staticcheck
			Type: &interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
			TargetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		})
	})
}

func TestRuntimeCapabilitiesPublishBackwardCompatibility(t *testing.T) {
	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	test := func(t *testing.T, value interpreter.Value) {

		rt := NewTestInterpreterRuntime()

		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		storage, inter, err := rt.Storage(Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, err)

		publicStorageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			true,
		)

		publicStorageMap.SetValue(
			inter,
			interpreter.StringStorageMapKey("cap"),
			value,
		)

		err = storage.Commit(inter, false)
		require.NoError(t, err)

		nextScriptLocation := NewScriptLocationGenerator()

		_, err = rt.ExecuteScript(
			Script{
				Source: []byte(`
                  access(all) fun main() {
                      let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)
                      let capability = account.storage.load<Capability>(from: /storage/cap)!
                      account.capabilities.publish(capability, at: /public/test)
                  }
                `),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
			},
		)
		require.NoError(t, err)
	}

	t.Run("path capability, untyped", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
			// NOTE: no borrow type
			nil,
			interpreter.AddressValue(testAddress),
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		))
	})

	t.Run("path capability, typed", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
			&interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.AddressValue(testAddress),
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		))
	})
}

func TestRuntimeCapabilitiesUnpublishBackwardCompatibility(t *testing.T) {
	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	test := func(t *testing.T, value interpreter.Value) {

		rt := NewTestInterpreterRuntime()

		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
			},
		}

		storage, inter, err := rt.Storage(Context{
			Interface: runtimeInterface,
		})
		require.NoError(t, err)

		publicStorageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainPublic.Identifier(),
			true,
		)

		publicStorageMap.SetValue(
			inter,
			interpreter.StringStorageMapKey("test"),
			value,
		)

		err = storage.Commit(inter, false)
		require.NoError(t, err)

		nextScriptLocation := NewScriptLocationGenerator()

		_, err = rt.ExecuteScript(
			Script{
				Source: []byte(`
                  access(all) fun main() {
                      let account = getAuthAccount<auth(Storage, Capabilities) &Account>(0x1)
                      let capability = account.capabilities.unpublish(/public/test)!
                      assert(capability.id == 0)
                  }
                `),
			},
			Context{
				Interface: runtimeInterface,
				Location:  nextScriptLocation(),
			},
		)
		require.NoError(t, err)

	}

	t.Run("path capability, untyped", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
			// NOTE: no borrow type
			nil,
			interpreter.AddressValue(testAddress),
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		))
	})

	t.Run("path capability, typed", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
			&interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.AddressValue(testAddress),
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		))
	})

	t.Run("path link", func(t *testing.T) {
		t.Parallel()

		test(t, interpreter.PathLinkValue{ //nolint:staticcheck
			Type: &interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
			TargetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "test",
			},
		})
	})
}
