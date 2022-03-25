/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/tests/utils"
	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
)

type storageKey struct {
	address common.Address
	domain  string
	key     string
}

func testAccount(
	t *testing.T,
	address interpreter.AddressValue,
	auth bool,
	code string,
) (
	*interpreter.Interpreter,
	func() map[storageKey]interpreter.Value,
) {

	var valueDeclarations stdlib.StandardLibraryValues

	// `authAccount`

	authAccountValueDeclaration := stdlib.StandardLibraryValue{
		Name: "authAccount",
		Type: sema.AuthAccountType,
		ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
			return newTestAuthAccountValue(inter, address)
		},
		Kind: common.DeclarationKindConstant,
	}
	valueDeclarations = append(valueDeclarations, authAccountValueDeclaration)

	// `pubAccount`

	pubAccountValueDeclaration := stdlib.StandardLibraryValue{
		Name: "pubAccount",
		Type: sema.PublicAccountType,
		ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
			return newTestPublicAccountValue(inter, address)
		},
		Kind: common.DeclarationKindConstant,
	}
	valueDeclarations = append(valueDeclarations, pubAccountValueDeclaration)

	// `account`

	var accountValueDeclaration stdlib.StandardLibraryValue

	if auth {
		accountValueDeclaration = authAccountValueDeclaration
	} else {
		accountValueDeclaration = pubAccountValueDeclaration
	}
	accountValueDeclaration.Name = "account"
	valueDeclarations = append(valueDeclarations, accountValueDeclaration)

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(valueDeclarations.ToSemaValueDeclarations()),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(valueDeclarations.ToInterpreterValueDeclarations()),
			},
		},
	)
	require.NoError(t, err)

	getAccountValues := func() map[storageKey]interpreter.Value {
		accountValues := make(map[storageKey]interpreter.Value)

		for storageMapKey, accountStorage := range inter.Storage.(interpreter.InMemoryStorage).StorageMaps {
			iterator := accountStorage.Iterator(inter)
			for {
				key, value := iterator.Next()
				if key == "" {
					break
				}
				storageKey := storageKey{
					address: storageMapKey.Address,
					domain:  storageMapKey.Key,
					key:     key,
				}
				accountValues[storageKey] = value
			}
		}

		return accountValues
	}
	return inter, getAccountValues
}

func returnZeroUInt64(_ *interpreter.Interpreter) interpreter.UInt64Value {
	return interpreter.NewUnmeteredUInt64Value(0)
}

func returnZeroUFix64() interpreter.UFix64Value {
	return interpreter.NewUnmeteredUFix64Value(0)
}

func TestInterpretAuthAccount_save(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			`
              resource R {}

              fun test() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)
              }
            `,
		)

		// Save first value

		t.Run("initial save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			accountValues := getAccountValues()
			require.Len(t, accountValues, 1)
			for _, value := range accountValues {
				assert.IsType(t, &interpreter.CompositeValue{}, value)
			}
		})

		// Attempt to save again, overwriting should fail

		t.Run("second save", func(t *testing.T) {

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.ErrorAs(t, err, &interpreter.OverwriteError{})
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			`
              struct S {}

              fun test() {
                  let s = S()
                  account.save(s, to: /storage/s)
              }
            `,
		)

		// Save first value

		t.Run("initial save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			accountValues := getAccountValues()
			require.Len(t, accountValues, 1)
			for _, value := range accountValues {
				assert.IsType(t, &interpreter.CompositeValue{}, value)
			}

		})

		// Attempt to save again, overwriting should fail

		t.Run("second save", func(t *testing.T) {

			_, err := inter.Invoke("test")

			require.Error(t, err)

			require.ErrorAs(t, err, &interpreter.OverwriteError{})
		})
	})
}

func TestInterpretAuthAccount_type(t *testing.T) {

	t.Parallel()

	t.Run("type", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountStorables := testAccount(
			t,
			address,
			true,
			`
              struct S {}

              resource R {}

              fun saveR() {
				let r <- create R()
				account.save(<-r, to: /storage/x)
              }

			  fun saveS() {
				let s = S()
				destroy account.load<@R>(from: /storage/x)
			 	account.save(s, to: /storage/x)
			  }

              fun typeAt(): AnyStruct {
				return account.type(at: /storage/x)
              }
            `,
		)

		// type empty path is nil

		value, err := inter.Invoke("typeAt")
		require.NoError(t, err)
		require.Len(t, getAccountStorables(), 0)
		require.Equal(t, interpreter.NilValue{}, value)

		// save R

		_, err = inter.Invoke("saveR")
		require.NoError(t, err)
		require.Len(t, getAccountStorables(), 1)

		// type is now type of R

		value, err = inter.Invoke("typeAt")
		require.NoError(t, err)
		require.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.CompositeStaticType{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "R",
						TypeID:              "S.test.R",
					},
				},
			),
			value,
		)

		// save S

		_, err = inter.Invoke("saveS")
		require.NoError(t, err)
		require.Len(t, getAccountStorables(), 1)

		// type is now type of S

		value, err = inter.Invoke("typeAt")
		require.NoError(t, err)
		require.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.CompositeStaticType{
						Location:            utils.TestLocation,
						QualifiedIdentifier: "S",
						TypeID:              "S.test.S",
					},
				},
			),
			value,
		)
	})
}

func TestInterpretAuthAccount_load(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			`
              resource R {}

              resource R2 {}

              fun save() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)
              }

              fun loadR(): @R? {
                  return <-account.load<@R>(from: /storage/r)
              }

              fun loadR2(): @R2? {
                  return <-account.load<@R2>(from: /storage/r)
              }
            `,
		)

		t.Run("save R and load R ", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// first load

			value, err := inter.Invoke("loadR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was removed from storage
			require.Len(t, getAccountValues(), 0)

			// second load

			value, err = inter.Invoke("loadR")
			require.NoError(t, err)

			require.IsType(t, interpreter.NilValue{}, value)
		})

		t.Run("save R and load R2", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// load

			_, err = inter.Invoke("loadR2")

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			`
              struct S {}

              struct S2 {}

              fun save() {
                  let s = S()
                  account.save(s, to: /storage/s)
              }

              fun loadS(): S? {
                  return account.load<S>(from: /storage/s)
              }

              fun loadS2(): S2? {
                  return account.load<S2>(from: /storage/s)
              }
            `,
		)

		t.Run("save S and load S", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// first load

			value, err := inter.Invoke("loadS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was removed from storage
			require.Len(t, getAccountValues(), 0)

			// second load

			value, err = inter.Invoke("loadS")
			require.NoError(t, err)

			require.IsType(t, interpreter.NilValue{}, value)
		})

		t.Run("save S and load S2", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// load

			_, err = inter.Invoke("loadS2")
			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})
	})
}

func TestInterpretAuthAccount_copy(t *testing.T) {

	t.Parallel()

	const code = `
      struct S {}

      struct S2 {}

      fun save() {
          let s = S()
          account.save(s, to: /storage/s)
      }

      fun copyS(): S? {
          return account.copy<S>(from: /storage/s)
      }

      fun copyS2(): S2? {
          return account.copy<S2>(from: /storage/s)
      }
    `

	t.Run("save S and copy S ", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			code,
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		testCopyS := func() {

			value, err := inter.Invoke("copyS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		}

		testCopyS()

		testCopyS()
	})

	t.Run("save S and copy S2", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			code,
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		// load

		_, err = inter.Invoke("copyS2")
		require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

		// NOTE: check loaded value was *not* removed from storage
		require.Len(t, getAccountValues(), 1)
	})
}

func TestInterpretAuthAccount_borrow(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			`
              resource R {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              resource R2 {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun save() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)
              }

              fun borrowR(): &R? {
                  return account.borrow<&R>(from: /storage/r)
              }

              fun foo(): Int {
                  return account.borrow<&R>(from: /storage/r)!.foo
              }

              fun borrowR2(): &R2? {
                  return account.borrow<&R2>(from: /storage/r)
              }

              fun changeAfterBorrow(): Int {
                 let ref = account.borrow<&R>(from: /storage/r)!

                 let r <- account.load<@R>(from: /storage/r)
                 destroy r

                 let r2 <- create R2()
                 account.save(<-r2, to: /storage/r)

                 return ref.foo
              }
            `,
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		t.Run("borrow R ", func(t *testing.T) {

			// first borrow

			value, err := inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// foo

			value, err = inter.Invoke("foo")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// TODO: should fail, i.e. return nil

			// second borrow

			value, err = inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue = value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("borrow R2", func(t *testing.T) {

			_, err := inter.Invoke("borrowR2")
			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("changeAfterBorrow")

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(
			t,
			address,
			true,
			`
              struct S {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              struct S2 {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun save() {
                  let s = S()
                  account.save(s, to: /storage/s)
              }

              fun borrowS(): &S? {
                  return account.borrow<&S>(from: /storage/s)
              }

              fun foo(): Int {
                  return account.borrow<&S>(from: /storage/s)!.foo
              }

              fun borrowS2(): &S2? {
                  return account.borrow<&S2>(from: /storage/s)
              }

              fun changeAfterBorrow(): Int {
                 let ref = account.borrow<&S>(from: /storage/s)!

                 // remove stored value
                 account.load<S>(from: /storage/s)

                 let s2 = S2()
                 account.save(s2, to: /storage/s)

                 return ref.foo
              }

              fun invalidBorrowS(): &S2? {
                  let s = S()
                  account.save(s, to: /storage/another_s)
                  let borrowedS = account.borrow<auth &AnyStruct>(from: /storage/another_s)
                  return borrowedS as! auth &S2?
              }
            `,
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		t.Run("borrow S", func(t *testing.T) {

			// first borrow

			value, err := inter.Invoke("borrowS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// foo

			value, err = inter.Invoke("foo")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// TODO: should fail, i.e. return nil

			// second borrow

			value, err = inter.Invoke("borrowS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue = value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("borrow S2", func(t *testing.T) {

			_, err = inter.Invoke("borrowS2")
			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("changeAfterBorrow")

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})

		t.Run("borrow as invalid type", func(t *testing.T) {
			_, err = inter.Invoke("invalidBorrowS")
			require.Error(t, err)
			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
		})
	})
}

func TestInterpretAuthAccount_link(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		test := func(capabilityDomain common.PathDomain) {

			t.Run(capabilityDomain.Identifier(), func(t *testing.T) {

				t.Parallel()

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

				inter, getAccountValues := testAccount(
					t,
					address,
					true,
					fmt.Sprintf(
						`
	                      resource R {}

	                      resource R2 {}

	                      fun save() {
	                          let r <- create R()
	                          account.save(<-r, to: /storage/r)
	                      }

	                      fun linkR(): Capability? {
	                          return account.link<&R>(/%[1]s/rCap, target: /storage/r)
	                      }

	                      fun linkR2(): Capability? {
	                          return account.link<&R2>(/%[1]s/rCap2, target: /storage/r)
	                      }
	                    `,
						capabilityDomain.Identifier(),
					),
				)

				// save

				_, err := inter.Invoke("save")
				require.NoError(t, err)

				require.Len(t, getAccountValues(), 1)

				t.Run("link R", func(t *testing.T) {

					// first link

					value, err := inter.Invoke("linkR")
					require.NoError(t, err)

					require.IsType(t, &interpreter.SomeValue{}, value)

					capability := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

					rType := checker.RequireGlobalType(t, inter.Program.Elaboration, "R")

					expectedBorrowType := interpreter.ConvertSemaToStaticType(
						&sema.ReferenceType{
							Authorized: false,
							Type:       rType,
						},
					)

					RequireValuesEqual(
						t,
						inter,
						&interpreter.CapabilityValue{
							Address: address,
							Path: interpreter.PathValue{
								Domain:     capabilityDomain,
								Identifier: "rCap",
							},
							BorrowType: expectedBorrowType,
						},
						capability,
					)

					// stored value + link
					require.Len(t, getAccountValues(), 2)

					// second link

					value, err = inter.Invoke("linkR")
					require.NoError(t, err)

					require.IsType(t, interpreter.NilValue{}, value)

					// NOTE: check loaded value was *not* removed from storage
					require.Len(t, getAccountValues(), 2)
				})

				t.Run("link R2", func(t *testing.T) {

					// first link

					value, err := inter.Invoke("linkR2")
					require.NoError(t, err)

					require.IsType(t, &interpreter.SomeValue{}, value)

					capability := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

					r2Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "R2")

					expectedBorrowType := interpreter.ConvertSemaToStaticType(
						&sema.ReferenceType{
							Authorized: false,
							Type:       r2Type,
						},
					)

					RequireValuesEqual(
						t,
						inter,
						&interpreter.CapabilityValue{
							Address: address,
							Path: interpreter.PathValue{
								Domain:     capabilityDomain,
								Identifier: "rCap2",
							},
							BorrowType: expectedBorrowType,
						},
						capability,
					)

					// stored value + link
					require.Len(t, getAccountValues(), 3)

					// second link

					value, err = inter.Invoke("linkR2")
					require.NoError(t, err)

					require.IsType(t, interpreter.NilValue{}, value)

					// NOTE: check loaded value was *not* removed from storage
					require.Len(t, getAccountValues(), 3)
				})
			})
		}

		for _, capabilityDomain := range []common.PathDomain{
			common.PathDomainPrivate,
			common.PathDomainPublic,
		} {
			test(capabilityDomain)
		}
	})

	t.Run("struct", func(t *testing.T) {

		test := func(capabilityDomain common.PathDomain) {

			t.Run(capabilityDomain.Identifier(), func(t *testing.T) {

				t.Parallel()

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

				inter, getAccountValues := testAccount(
					t,
					address,
					true,
					fmt.Sprintf(
						`
	                      struct S {}

	                      struct S2 {}

	                      fun save() {
	                          let s = S()
	                          account.save(s, to: /storage/s)
	                      }

	                      fun linkS(): Capability? {
	                          return account.link<&S>(/%[1]s/sCap, target: /storage/s)
	                      }

	                      fun linkS2(): Capability? {
	                          return account.link<&S2>(/%[1]s/sCap2, target: /storage/s)
	                      }
	                    `,
						capabilityDomain.Identifier(),
					),
				)

				// save

				_, err := inter.Invoke("save")
				require.NoError(t, err)

				require.Len(t, getAccountValues(), 1)

				t.Run("link S", func(t *testing.T) {

					// first link

					value, err := inter.Invoke("linkS")
					require.NoError(t, err)

					require.IsType(t, &interpreter.SomeValue{}, value)

					capability := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

					sType := checker.RequireGlobalType(t, inter.Program.Elaboration, "S")

					expectedBorrowType := interpreter.ConvertSemaToStaticType(
						&sema.ReferenceType{
							Authorized: false,
							Type:       sType,
						},
					)

					RequireValuesEqual(
						t,
						inter,
						&interpreter.CapabilityValue{
							Address: address,
							Path: interpreter.PathValue{
								Domain:     capabilityDomain,
								Identifier: "sCap",
							},
							BorrowType: expectedBorrowType,
						},
						capability,
					)

					// stored value + link
					require.Len(t, getAccountValues(), 2)

					// second link

					value, err = inter.Invoke("linkS")
					require.NoError(t, err)

					require.IsType(t, interpreter.NilValue{}, value)

					// NOTE: check loaded value was *not* removed from storage
					require.Len(t, getAccountValues(), 2)
				})

				t.Run("link S2", func(t *testing.T) {

					// first link

					value, err := inter.Invoke("linkS2")
					require.NoError(t, err)

					require.IsType(t, &interpreter.SomeValue{}, value)

					capability := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)
					require.IsType(t, &interpreter.CapabilityValue{}, capability)

					s2Type := checker.RequireGlobalType(t, inter.Program.Elaboration, "S2")

					expectedBorrowType := interpreter.ConvertSemaToStaticType(
						&sema.ReferenceType{
							Authorized: false,
							Type:       s2Type,
						},
					)

					RequireValuesEqual(
						t,
						inter,
						&interpreter.CapabilityValue{
							Address: address,
							Path: interpreter.PathValue{
								Domain:     capabilityDomain,
								Identifier: "sCap2",
							},
							BorrowType: expectedBorrowType,
						},
						capability,
					)

					// stored value + link
					require.Len(t, getAccountValues(), 3)

					// second link

					value, err = inter.Invoke("linkS2")
					require.NoError(t, err)

					require.IsType(t, interpreter.NilValue{}, value)

					// NOTE: check loaded value was *not* removed from storage
					require.Len(t, getAccountValues(), 3)
				})
			})
		}

		for _, capabilityDomain := range []common.PathDomain{
			common.PathDomainPrivate,
			common.PathDomainPublic,
		} {
			test(capabilityDomain)
		}
	})

	t.Run("link to same path", func(t *testing.T) {
		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		test := func(capabilityDomain common.PathDomain) {
			inter, getAccountValues := testAccount(
				t,
				address,
				true,
				fmt.Sprintf(`
                    struct S1 {}

                    struct S2 {}

                    fun save() {
                        let s1 = S1()
                        account.save(s1, to: /storage/s1)

                        let s2 = S2()
                        account.save(s2, to: /storage/s2)
                    }

                    fun linkToSamePath(): Capability? {
                        account.link<&S1>(/%[1]s/sCap, target: /storage/s1)

                        // link a different storage value to the same path.
                        return account.link<&S2>(/%[1]s/sCap, target: /storage/s2)
                    }

                    fun getCapability(): Capability? {
                        return account.getCapability<&S1>(/%[1]s/sCap)
                    }`,

					capabilityDomain.Identifier(),
				),
			)

			// Save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 2)

			t.Run(capabilityDomain.Identifier(), func(t *testing.T) {
				value, err := inter.Invoke("linkToSamePath")
				require.NoError(t, err)
				require.IsType(t, interpreter.NilValue{}, value)

				// Only one link must have been created.
				// i.e: 2 values + 1 link
				require.Len(t, getAccountValues(), 3)

				value, err = inter.Invoke("getCapability")
				require.NoError(t, err)
				require.IsType(t, &interpreter.SomeValue{}, value)

				capability := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

				sType := checker.RequireGlobalType(t, inter.Program.Elaboration, "S1")
				expectedBorrowType := interpreter.ConvertSemaToStaticType(
					&sema.ReferenceType{
						Authorized: false,
						Type:       sType,
					},
				)

				RequireValuesEqual(
					t,
					inter,
					&interpreter.CapabilityValue{
						Address: address,
						Path: interpreter.PathValue{
							Domain:     capabilityDomain,
							Identifier: "sCap",
						},
						BorrowType: expectedBorrowType,
					},
					capability,
				)
			})
		}

		for _, capabilityDomain := range []common.PathDomain{
			common.PathDomainPrivate,
			common.PathDomainPublic,
		} {
			test(capabilityDomain)
		}
	})

	t.Run("link same storage", func(t *testing.T) {
		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		test := func(capabilityDomain common.PathDomain) {
			inter, getAccountValues := testAccount(
				t,
				address,
				true,
				fmt.Sprintf(`
                    struct S {}

                    fun save() {
                        let s = S()
                        account.save(s, to: /storage/s)
                    }

                    fun linkSameStorage(): Capability? {
                        account.link<&S>(/%[1]s/s1Cap, target: /storage/s)

                        // link an already linked storage value to a different path.
                        return account.link<&S>(/%[1]s/s2Cap, target: /storage/s)
                    }

                    fun getFirstCapability(): Capability? {
                        return account.getCapability<&S>(/%[1]s/s1Cap)
                    }`,

					capabilityDomain.Identifier(),
				),
			)

			// Save
			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			t.Run(capabilityDomain.Identifier(), func(t *testing.T) {
				value, err := inter.Invoke("linkSameStorage")
				require.NoError(t, err)
				require.IsType(t, &interpreter.SomeValue{}, value)

				// 1 value + 2 links
				require.Len(t, getAccountValues(), 3)

				capability := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

				sType := checker.RequireGlobalType(t, inter.Program.Elaboration, "S")
				expectedBorrowType := interpreter.ConvertSemaToStaticType(
					&sema.ReferenceType{
						Authorized: false,
						Type:       sType,
					},
				)

				RequireValuesEqual(
					t,
					inter,
					&interpreter.CapabilityValue{
						Address: address,
						Path: interpreter.PathValue{
							Domain:     capabilityDomain,
							Identifier: "s2Cap",
						},
						BorrowType: expectedBorrowType,
					},
					capability,
				)

				value, err = inter.Invoke("getFirstCapability")
				require.NoError(t, err)
				require.IsType(t, &interpreter.SomeValue{}, value)

				capability = value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

				sType = checker.RequireGlobalType(t, inter.Program.Elaboration, "S")
				expectedBorrowType = interpreter.ConvertSemaToStaticType(
					&sema.ReferenceType{
						Authorized: false,
						Type:       sType,
					},
				)

				RequireValuesEqual(
					t,
					inter,
					&interpreter.CapabilityValue{
						Address: address,
						Path: interpreter.PathValue{
							Domain:     capabilityDomain,
							Identifier: "s1Cap",
						},
						BorrowType: expectedBorrowType,
					},
					capability,
				)
			})
		}

		for _, capabilityDomain := range []common.PathDomain{
			common.PathDomainPrivate,
			common.PathDomainPublic,
		} {
			test(capabilityDomain)
		}
	})
}

func TestInterpretAuthAccount_unlink(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		test := func(capabilityDomain common.PathDomain) {

			t.Run(capabilityDomain.Identifier(), func(t *testing.T) {

				t.Parallel()

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

				inter, getAccountValues := testAccount(
					t,
					address,
					true,
					fmt.Sprintf(
						`
	                      resource R {}

	                      resource R2 {}

	                      fun saveAndLinkR() {
	                          let r <- create R()
	                          account.save(<-r, to: /storage/r)
	                          account.link<&R>(/%[1]s/r, target: /storage/r)
	                      }

	                      fun unlinkR() {
	                          account.unlink(/%[1]s/r)
	                      }

                          fun unlinkR2() {
	                          account.unlink(/%[1]s/r2)
	                      }
	                    `,
						capabilityDomain.Identifier(),
					),
				)

				// save and link

				_, err := inter.Invoke("saveAndLinkR")
				require.NoError(t, err)

				require.Len(t, getAccountValues(), 2)

				t.Run("unlink R", func(t *testing.T) {
					_, err := inter.Invoke("unlinkR")
					require.NoError(t, err)

					require.Len(t, getAccountValues(), 1)
				})

				t.Run("unlink R2", func(t *testing.T) {

					_, err := inter.Invoke("unlinkR2")
					require.NoError(t, err)

					require.Len(t, getAccountValues(), 1)
				})
			})
		}

		for _, capabilityDomain := range []common.PathDomain{
			common.PathDomainPrivate,
			common.PathDomainPublic,
		} {

			test(capabilityDomain)
		}
	})

	t.Run("struct", func(t *testing.T) {

		test := func(capabilityDomain common.PathDomain) {

			t.Run(capabilityDomain.Identifier(), func(t *testing.T) {

				t.Parallel()

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

				inter, getAccountValues := testAccount(
					t,
					address,
					true,
					fmt.Sprintf(
						`
	                      struct S {}

	                      struct S2 {}

	                      fun saveAndLinkS() {
	                          let s = S()
	                          account.save(s, to: /storage/s)
	                          account.link<&S>(/%[1]s/s, target: /storage/s)
	                      }

	                      fun unlinkS() {
	                          account.unlink(/%[1]s/s)
	                      }

                          fun unlinkS2() {
	                          account.unlink(/%[1]s/s2)
	                      }
	                    `,
						capabilityDomain.Identifier(),
					),
				)

				// save and link

				_, err := inter.Invoke("saveAndLinkS")
				require.NoError(t, err)

				require.Len(t, getAccountValues(), 2)

				t.Run("unlink S", func(t *testing.T) {
					_, err := inter.Invoke("unlinkS")
					require.NoError(t, err)

					require.Len(t, getAccountValues(), 1)
				})

				t.Run("unlink S2", func(t *testing.T) {

					_, err := inter.Invoke("unlinkS2")
					require.NoError(t, err)

					require.Len(t, getAccountValues(), 1)
				})
			})
		}

		for _, capabilityDomain := range []common.PathDomain{
			common.PathDomainPrivate,
			common.PathDomainPublic,
		} {

			test(capabilityDomain)
		}
	})
}

func TestInterpretAccount_getLinkTarget(t *testing.T) {

	t.Parallel()

	testResource := func(capabilityDomain common.PathDomain, auth bool) {

		t.Run(capabilityDomain.Identifier(), func(t *testing.T) {

			t.Parallel()

			address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

			inter, getAccountValues := testAccount(
				t,
				address,
				auth,
				fmt.Sprintf(
					`
	                  resource R {}

	                  fun link() {
	                      authAccount.link<&R>(/%[1]s/r, target: /storage/r)
	                  }

	                  fun existing(): Path? {
	                      return account.getLinkTarget(/%[1]s/r)
	                  }

                      fun nonExisting(): Path? {
	                      return account.getLinkTarget(/%[1]s/r2)
	                  }
	                `,
					capabilityDomain.Identifier(),
				),
			)

			// link

			_, err := inter.Invoke("link")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			t.Run("existing", func(t *testing.T) {

				value, err := inter.Invoke("existing")
				require.NoError(t, err)

				require.IsType(t, &interpreter.SomeValue{}, value)

				innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

				AssertValuesEqual(
					t,
					inter,
					interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "r",
					},
					innerValue,
				)

				require.Len(t, getAccountValues(), 1)
			})

			t.Run("nonExisting", func(t *testing.T) {

				value, err := inter.Invoke("nonExisting")
				require.NoError(t, err)

				RequireValuesEqual(
					t,
					inter,
					interpreter.NilValue{},
					value,
				)

				require.Len(t, getAccountValues(), 1)
			})
		})
	}

	testStruct := func(capabilityDomain common.PathDomain, auth bool) {

		t.Run(capabilityDomain.Identifier(), func(t *testing.T) {

			t.Parallel()

			address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

			inter, getAccountValues := testAccount(
				t,
				address,
				auth,
				fmt.Sprintf(
					`
	                  struct S {}

	                  fun link() {
	                      authAccount.link<&S>(/%[1]s/s, target: /storage/s)
	                  }

	                  fun existing(): Path? {
	                      return account.getLinkTarget(/%[1]s/s)
	                  }

                      fun nonExisting(): Path? {
	                      return account.getLinkTarget(/%[1]s/s2)
	                  }
	                `,
					capabilityDomain.Identifier(),
				),
			)

			// link

			_, err := inter.Invoke("link")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			t.Run("existing", func(t *testing.T) {

				value, err := inter.Invoke("existing")
				require.NoError(t, err)

				require.IsType(t, &interpreter.SomeValue{}, value)

				innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.ReturnEmptyLocationRange)

				AssertValuesEqual(
					t,
					inter,
					interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "s",
					},
					innerValue,
				)

				require.Len(t, getAccountValues(), 1)
			})

			t.Run("nonExisting", func(t *testing.T) {

				value, err := inter.Invoke("nonExisting")
				require.NoError(t, err)

				RequireValuesEqual(
					t,
					inter,
					interpreter.NilValue{},
					value,
				)

				require.Len(t, getAccountValues(), 1)
			})
		})
	}

	for _, auth := range []bool{true, false} {

		t.Run(fmt.Sprintf("auth: %v", auth), func(t *testing.T) {

			t.Run("resource", func(t *testing.T) {

				for _, capabilityDomain := range []common.PathDomain{
					common.PathDomainPrivate,
					common.PathDomainPublic,
				} {

					testResource(capabilityDomain, auth)
				}
			})

			t.Run("struct", func(t *testing.T) {

				for _, capabilityDomain := range []common.PathDomain{
					common.PathDomainPrivate,
					common.PathDomainPublic,
				} {

					testStruct(capabilityDomain, auth)
				}
			})
		})
	}
}

func TestInterpretAccount_getCapability(t *testing.T) {

	t.Parallel()

	tests := map[bool][]common.PathDomain{
		true: {
			common.PathDomainPublic,
			common.PathDomainPrivate,
		},
		false: {
			common.PathDomainPublic,
		},
	}

	for auth, validDomains := range tests {

		for _, domain := range validDomains {

			for _, typed := range []bool{false, true} {

				var typeArguments string
				if typed {
					typeArguments = "<&Int>"
				}

				testName := fmt.Sprintf(
					"auth: %v, domain: %s, typed: %v",
					auth,
					domain.Identifier(),
					typed,
				)

				t.Run(testName, func(t *testing.T) {

					address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

					inter, _ := testAccount(
						t,
						address,
						auth,
						fmt.Sprintf(
							`
	                          fun test(): Capability%[1]s {
	                              return account.getCapability%[1]s(/%[2]s/r)
	                          }
	                        `,
							typeArguments,
							domain.Identifier(),
						),
					)

					value, err := inter.Invoke("test")

					require.NoError(t, err)

					require.IsType(t, &interpreter.CapabilityValue{}, value)

					actualBorrowType := value.(*interpreter.CapabilityValue).BorrowType

					if typed {
						expectedBorrowType := interpreter.ConvertSemaToStaticType(
							&sema.ReferenceType{
								Authorized: false,
								Type:       sema.IntType,
							},
						)
						require.Equal(t,
							expectedBorrowType,
							actualBorrowType,
						)

					} else {
						require.Nil(t, actualBorrowType)
					}
				})
			}
		}
	}
}

func TestInterpretAccount_BalanceFields(t *testing.T) {
	t.Parallel()

	for accountType, auth := range map[string]bool{
		"AuthAccount":   true,
		"PublicAccount": false,
	} {

		for _, fieldName := range []string{
			"balance",
			"availableBalance",
		} {

			testName := fmt.Sprintf(
				"%s.%s",
				accountType,
				fieldName,
			)

			t.Run(testName, func(t *testing.T) {

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

				code := fmt.Sprintf(
					`
	                      fun test(): UFix64 {
	                          return account.%s
	                      }
	                    `,
					fieldName,
				)
				inter, _ := testAccount(
					t,
					address,
					auth,
					code,
				)

				value, err := inter.Invoke("test")
				require.NoError(t, err)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredUFix64Value(0),
					value,
				)
			})
		}
	}
}

func TestInterpretAccount_StorageFields(t *testing.T) {
	t.Parallel()

	for accountType, auth := range map[string]bool{
		"AuthAccount":   true,
		"PublicAccount": false,
	} {

		for _, fieldName := range []string{
			"storageUsed",
			"storageCapacity",
		} {

			testName := fmt.Sprintf(
				"%s.%s",
				accountType,
				fieldName,
			)

			t.Run(testName, func(t *testing.T) {

				code := fmt.Sprintf(
					`
	                      fun test(): UInt64 {
	                          return account.%s
	                      }
	                    `,
					fieldName,
				)

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

				inter, _ := testAccount(
					t,
					address,
					auth,
					code,
				)

				value, err := inter.Invoke("test")
				require.NoError(t, err)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredUInt64Value(0),
					value,
				)
			})
		}
	}
}
