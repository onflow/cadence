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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/activations"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

type storageKey struct {
	address common.Address
	domain  string
	key     atree.Value
}

func testAccount(
	t *testing.T,
	address interpreter.AddressValue,
	auth bool,
	code string,
	checkerConfig sema.Config,
) (
	*interpreter.Interpreter,
	func() map[storageKey]interpreter.Value,
) {
	return testAccountWithErrorHandler(
		t,
		address,
		auth,
		code,
		checkerConfig,
		nil,
	)
}

func testAccountWithErrorHandler(
	t *testing.T,
	address interpreter.AddressValue,
	auth bool,
	code string,
	checkerConfig sema.Config,
	checkerErrorHandler func(error),
) (
	*interpreter.Interpreter,
	func() map[storageKey]interpreter.Value,
) {

	var valueDeclarations []stdlib.StandardLibraryValue

	// `authAccount`

	authAccountValueDeclaration := stdlib.StandardLibraryValue{
		Name:  "authAccount",
		Type:  sema.AuthAccountType,
		Value: newTestAuthAccountValue(nil, address),
		Kind:  common.DeclarationKindConstant,
	}
	valueDeclarations = append(valueDeclarations, authAccountValueDeclaration)

	// `pubAccount`

	pubAccountValueDeclaration := stdlib.StandardLibraryValue{
		Name:  "pubAccount",
		Type:  sema.PublicAccountType,
		Value: newTestPublicAccountValue(nil, address),
		Kind:  common.DeclarationKindConstant,
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

	if checkerConfig.BaseValueActivation == nil {
		checkerConfig.BaseValueActivation = sema.BaseValueActivation
	}
	baseValueActivation := sema.NewVariableActivation(checkerConfig.BaseValueActivation)
	for _, valueDeclaration := range valueDeclarations {
		baseValueActivation.DeclareValue(valueDeclaration)
	}
	checkerConfig.BaseValueActivation = baseValueActivation

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	for _, valueDeclaration := range valueDeclarations {
		interpreter.Declare(baseActivation, valueDeclaration)
	}

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &checkerConfig,
			Config: &interpreter.Config{
				BaseActivation:                       baseActivation,
				ContractValueHandler:                 makeContractValueHandler(nil, nil, nil),
				InvalidatedResourceValidationEnabled: true,
				AuthAccountHandler: func(address interpreter.AddressValue) interpreter.Value {
					return newTestAuthAccountValue(nil, address)
				},
			},
			HandleCheckerError: checkerErrorHandler,
		},
	)
	require.NoError(t, err)

	getAccountValues := func() map[storageKey]interpreter.Value {
		accountValues := make(map[storageKey]interpreter.Value)

		for storageMapKey, accountStorage := range inter.Storage().(interpreter.InMemoryStorage).StorageMaps {
			iterator := accountStorage.Iterator(inter)
			for {
				key, value := iterator.Next()
				if key == nil {
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
			sema.Config{},
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
			RequireError(t, err)

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
			sema.Config{},
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
			RequireError(t, err)

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
			sema.Config{},
		)

		// type empty path is nil

		value, err := inter.Invoke("typeAt")
		require.NoError(t, err)
		require.Len(t, getAccountStorables(), 0)
		require.Equal(t, interpreter.Nil, value)

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
						Location:            TestLocation,
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
						Location:            TestLocation,
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
			sema.Config{},
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

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.EmptyLocationRange)

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was removed from storage
			require.Len(t, getAccountValues(), 0)

			// second load

			value, err = inter.Invoke("loadR")
			require.NoError(t, err)

			require.IsType(t, interpreter.Nil, value)
		})

		t.Run("save R and load R2", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// load

			_, err = inter.Invoke("loadR2")
			RequireError(t, err)

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
			sema.Config{},
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

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.EmptyLocationRange)

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was removed from storage
			require.Len(t, getAccountValues(), 0)

			// second load

			value, err = inter.Invoke("loadS")
			require.NoError(t, err)

			require.IsType(t, interpreter.Nil, value)
		})

		t.Run("save S and load S2", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// load

			_, err = inter.Invoke("loadS2")
			RequireError(t, err)

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
			sema.Config{},
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		testCopyS := func() {

			value, err := inter.Invoke("copyS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.EmptyLocationRange)

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
			sema.Config{},
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		// load

		_, err = inter.Invoke("copyS2")
		RequireError(t, err)

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

			  fun checkR(): Bool {
				  return account.check<@R>(from: /storage/r)
			  }

              fun borrowR(): &R? {
                  return account.borrow<&R>(from: /storage/r)
              }

              fun foo(): Int {
                  return account.borrow<&R>(from: /storage/r)!.foo
              }

			  fun checkR2(): Bool {
				  return account.check<@R2>(from: /storage/r)
			  }

              fun borrowR2(): &R2? {
                  return account.borrow<&R2>(from: /storage/r)
              }

			  fun checkR2WithInvalidPath(): Bool {
				  return account.check<@R2>(from: /storage/wrongpath)
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
			sema.Config{},
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		t.Run("borrow R ", func(t *testing.T) {

			// first check & borrow
			checkRes, err := inter.Invoke("checkR")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.AsBoolValue(true),
				checkRes,
			)

			value, err := inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.EmptyLocationRange)

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

			// second check & borrow
			checkRes, err = inter.Invoke("checkR")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.AsBoolValue(true),
				checkRes,
			)

			value, err = inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue = value.(*interpreter.SomeValue).InnerValue(inter, interpreter.EmptyLocationRange)

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("borrow R2", func(t *testing.T) {
			checkRes, err := inter.Invoke("checkR2")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.AsBoolValue(false),
				checkRes,
			)

			_, err = inter.Invoke("borrowR2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("changeAfterBorrow")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})

		t.Run("check R2 with wrong path", func(t *testing.T) {
			checkRes, err := inter.Invoke("checkR2WithInvalidPath")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.AsBoolValue(false),
				checkRes,
			)
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

			  fun checkS(): Bool {
				  return account.check<S>(from: /storage/s)
			  }

              fun borrowS(): &S? {
                  return account.borrow<&S>(from: /storage/s)
              }

              fun foo(): Int {
                  return account.borrow<&S>(from: /storage/s)!.foo
              }
			 
			  fun checkS2(): Bool {
				  return account.check<S2>(from: /storage/s)
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
			sema.Config{},
		)

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		t.Run("borrow S", func(t *testing.T) {

			// first check & borrow
			checkRes, err := inter.Invoke("checkS")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.AsBoolValue(true),
				checkRes,
			)

			value, err := inter.Invoke("borrowS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue(inter, interpreter.EmptyLocationRange)

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

			// second check & borrow
			checkRes, err = inter.Invoke("checkS")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.AsBoolValue(true),
				checkRes,
			)

			value, err = inter.Invoke("borrowS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue = value.(*interpreter.SomeValue).InnerValue(inter, interpreter.EmptyLocationRange)

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("borrow S2", func(t *testing.T) {
			checkRes, err := inter.Invoke("checkS2")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.AsBoolValue(false),
				checkRes,
			)

			_, err = inter.Invoke("borrowS2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("changeAfterBorrow")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})

		t.Run("borrow as invalid type", func(t *testing.T) {
			_, err = inter.Invoke("invalidBorrowS")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
		})
	})
}

func TestInterpretAccountBalanceFields(t *testing.T) {
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
					sema.Config{},
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
					sema.Config{},
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
