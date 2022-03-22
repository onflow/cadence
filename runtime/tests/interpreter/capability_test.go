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
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/interpreter"
)

func TestInterpretCapability_borrow(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
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

              struct S {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun saveAndLink() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)

                  account.link<&R>(/public/single, target: /storage/r)

                  account.link<&R>(/public/double, target: /public/single)

                  account.link<&R>(/public/nonExistent, target: /storage/nonExistent)

                  account.link<&R>(/public/loop1, target: /public/loop2)
                  account.link<&R>(/public/loop2, target: /public/loop1)

                  account.link<&R2>(/public/r2, target: /storage/r)
              }

              fun foo(_ path: CapabilityPath): Int {
                  return account.getCapability(path).borrow<&R>()!.foo
              }

              fun single(): Int {
                  return foo(/public/single)
              }

              fun singleAuth(): auth &R? {
                  return account.getCapability(/public/single).borrow<auth &R>()
              }

              fun singleR2(): &R2? {
                  return account.getCapability(/public/single).borrow<&R2>()
              }

              fun singleS(): &S? {
                  return account.getCapability(/public/single).borrow<&S>()
              }

              fun double(): Int {
                  return foo(/public/double)
              }

              fun nonExistent(): Int {
                  return foo(/public/nonExistent)
              }

              fun loop(): Int {
                  return foo(/public/loop1)
              }

              fun singleTyped(): Int {
                  return account.getCapability<&R>(/public/single)!.borrow()!.foo
              }

              fun r2(): Int {
                  return account.getCapability<&R2>(/public/r2).borrow()!.foo
              }

              fun singleChangeAfterBorrow(): Int {
                 let ref = account.getCapability(/public/single).borrow<&R>()!

                 let r <- account.load<@R>(from: /storage/r)
                 destroy r

                 let r2 <- create R2()
                 account.save(<-r2, to: /storage/r)

                 return ref.foo
              }
            `,
		)

		// save

		_, err := inter.Invoke("saveAndLink")
		require.NoError(t, err)

		t.Run("single", func(t *testing.T) {

			value, err := inter.Invoke("single")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})

		t.Run("single R2", func(t *testing.T) {

			value, err := inter.Invoke("singleR2")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NilValue{},
				value,
			)
		})

		t.Run("single S", func(t *testing.T) {

			value, err := inter.Invoke("singleS")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NilValue{},
				value,
			)
		})

		t.Run("single auth", func(t *testing.T) {

			value, err := inter.Invoke("singleAuth")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NilValue{},
				value,
			)
		})

		t.Run("double", func(t *testing.T) {

			value, err := inter.Invoke("double")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})

		t.Run("nonExistent", func(t *testing.T) {

			_, err := inter.Invoke("nonExistent")
			require.Error(t, err)

			require.ErrorAs(t, err, &interpreter.ForceNilError{})
		})

		t.Run("loop", func(t *testing.T) {

			_, err := inter.Invoke("loop")
			require.Error(t, err)

			var cyclicLinkErr interpreter.CyclicLinkError
			require.ErrorAs(t, err, &cyclicLinkErr)

			require.Equal(t,
				cyclicLinkErr.Error(),
				"cyclic link in account 0x2a: /public/loop1 -> /public/loop2 -> /public/loop1",
			)
		})

		t.Run("singleTyped", func(t *testing.T) {

			value, err := inter.Invoke("singleTyped")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})

		t.Run("r2", func(t *testing.T) {

			_, err := inter.Invoke("r2")
			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
		})

		t.Run("single change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("singleChangeAfterBorrow")
			require.Error(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
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

              resource R {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun saveAndLink() {
                  let s = S()
                  account.save(s, to: /storage/s)

                  account.link<&S>(/public/single, target: /storage/s)

                  account.link<&S>(/public/double, target: /public/single)

                  account.link<&S>(/public/nonExistent, target: /storage/nonExistent)

                  account.link<&S>(/public/loop1, target: /public/loop2)
                  account.link<&S>(/public/loop2, target: /public/loop1)

                  account.link<&S2>(/public/s2, target: /storage/s)
              }

              fun foo(_ path: CapabilityPath): Int {
                  return account.getCapability(path).borrow<&S>()!.foo
              }

              fun single(): Int {
                  return foo(/public/single)
              }

              fun singleAuth(): auth &S? {
                  return account.getCapability(/public/single).borrow<auth &S>()
              }

              fun singleS2(): &S2? {
                  return account.getCapability(/public/single).borrow<&S2>()
              }

              fun singleR(): &R? {
                  return account.getCapability(/public/single).borrow<&R>()
              }

              fun double(): Int {
                  return foo(/public/double)
              }

              fun nonExistent(): Int {
                  return foo(/public/nonExistent)
              }

              fun loop(): Int {
                  return foo(/public/loop1)
              }

              fun singleTyped(): Int {
                  return account.getCapability<&S>(/public/single)!.borrow()!.foo
              }

              fun s2(): Int {
                  return account.getCapability<&S2>(/public/s2).borrow()!.foo
              }

              fun singleChangeAfterBorrow(): Int {
                 let ref = account.getCapability(/public/single).borrow<&S>()!

                 // remove stored value
                 account.load<S>(from: /storage/s)

                 let s2 = S2()
                 account.save(s2, to: /storage/s)

                 return ref.foo
              }
            `,
		)

		// save

		_, err := inter.Invoke("saveAndLink")
		require.NoError(t, err)

		t.Run("single", func(t *testing.T) {

			value, err := inter.Invoke("single")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})

		t.Run("single S2", func(t *testing.T) {

			value, err := inter.Invoke("singleS2")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NilValue{},
				value,
			)
		})

		t.Run("single R", func(t *testing.T) {

			value, err := inter.Invoke("singleR")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NilValue{},
				value,
			)
		})

		t.Run("single auth", func(t *testing.T) {

			value, err := inter.Invoke("singleAuth")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NilValue{},
				value,
			)
		})

		t.Run("double", func(t *testing.T) {

			value, err := inter.Invoke("double")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})

		t.Run("nonExistent", func(t *testing.T) {

			_, err := inter.Invoke("nonExistent")
			require.Error(t, err)

			require.ErrorAs(t, err, &interpreter.ForceNilError{})
		})

		t.Run("loop", func(t *testing.T) {

			_, err := inter.Invoke("loop")
			require.Error(t, err)

			var cyclicLinkErr interpreter.CyclicLinkError
			require.ErrorAs(t, err, &cyclicLinkErr)

			require.Equal(t,
				cyclicLinkErr.Error(),
				"cyclic link in account 0x2a: /public/loop1 -> /public/loop2 -> /public/loop1",
			)
		})

		t.Run("singleTyped", func(t *testing.T) {

			value, err := inter.Invoke("singleTyped")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)
		})

		t.Run("s2", func(t *testing.T) {

			_, err := inter.Invoke("s2")
			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
		})

		t.Run("single change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("singleChangeAfterBorrow")
			require.Error(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})
	})
}

func TestInterpretCapability_check(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
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

              struct S {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun saveAndLink() {
                  let r <- create R()
                  account.save(<-r, to: /storage/r)

                  account.link<&R>(/public/single, target: /storage/r)

                  account.link<&R>(/public/double, target: /public/single)

                  account.link<&R>(/public/nonExistent, target: /storage/nonExistent)

                  account.link<&R>(/public/loop1, target: /public/loop2)
                  account.link<&R>(/public/loop2, target: /public/loop1)

                  account.link<&R2>(/public/r2, target: /storage/r)
              }

              fun check(_ path: CapabilityPath): Bool {
                  return account.getCapability(path).check<&R>()
              }

              fun single(): Bool {
                  return check(/public/single)
              }

              fun singleAuth(): Bool {
                  return account.getCapability(/public/single).check<auth &R>()
              }

              fun singleR2(): Bool {
                  return account.getCapability(/public/single).check<&R2>()
              }

              fun singleS(): Bool {
                  return account.getCapability(/public/single).check<&S>()
              }

              fun double(): Bool {
                  return check(/public/double)
              }

              fun nonExistent(): Bool {
                  return check(/public/nonExistent)
              }

              fun loop(): Bool {
                  return check(/public/loop1)
              }

              fun singleTyped(): Bool {
                  return account.getCapability<&R>(/public/single)!.check()
              }

              fun r2(): Bool {
                  return account.getCapability<&R2>(/public/r2).check()
              }
            `,
		)

		// save

		_, err := inter.Invoke("saveAndLink")
		require.NoError(t, err)

		t.Run("single", func(t *testing.T) {

			value, err := inter.Invoke("single")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
		})

		t.Run("single auth", func(t *testing.T) {

			value, err := inter.Invoke("singleAuth")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("single R2", func(t *testing.T) {

			value, err := inter.Invoke("singleR2")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("single S", func(t *testing.T) {

			value, err := inter.Invoke("singleS")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("double", func(t *testing.T) {

			value, err := inter.Invoke("double")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
		})

		t.Run("nonExistent", func(t *testing.T) {

			value, err := inter.Invoke("nonExistent")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("loop", func(t *testing.T) {

			_, err := inter.Invoke("loop")
			require.Error(t, err)

			var cyclicLinkErr interpreter.CyclicLinkError
			require.ErrorAs(t, err, &cyclicLinkErr)

			require.Equal(t,
				cyclicLinkErr.Error(),
				"cyclic link in account 0x2a: /public/loop1 -> /public/loop2 -> /public/loop1",
			)
		})

		t.Run("singleTyped", func(t *testing.T) {

			value, err := inter.Invoke("singleTyped")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
		})

		t.Run("r2", func(t *testing.T) {

			value, err := inter.Invoke("r2")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(
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

              resource R {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun saveAndLink() {
                  let s = S()
                  account.save(s, to: /storage/s)

                  account.link<&S>(/public/single, target: /storage/s)

                  account.link<&S>(/public/double, target: /public/single)

                  account.link<&S>(/public/nonExistent, target: /storage/nonExistent)

                  account.link<&S>(/public/loop1, target: /public/loop2)
                  account.link<&S>(/public/loop2, target: /public/loop1)

                  account.link<&S2>(/public/s2, target: /storage/s)
              }

              fun check(_ path: CapabilityPath): Bool {
                  return account.getCapability(path).check<&S>()
              }

              fun single(): Bool {
                  return check(/public/single)
              }

              fun singleAuth(): Bool {
                  return account.getCapability(/public/single).check<auth &S>()
              }

              fun singleS2(): Bool {
                  return account.getCapability(/public/single).check<&S2>()
              }

              fun singleR(): Bool {
                  return account.getCapability(/public/single).check<&R>()
              }

              fun double(): Bool {
                  return check(/public/double)
              }

              fun nonExistent(): Bool {
                  return check(/public/nonExistent)
              }

              fun loop(): Bool {
                  return check(/public/loop1)
              }

              fun singleTyped(): Bool {
                  return account.getCapability<&S>(/public/single)!.check()
              }

              fun s2(): Bool {
                  return account.getCapability<&S2>(/public/s2).check()
              }
            `,
		)

		// save

		_, err := inter.Invoke("saveAndLink")
		require.NoError(t, err)

		t.Run("single", func(t *testing.T) {

			value, err := inter.Invoke("single")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
		})

		t.Run("single auth", func(t *testing.T) {

			value, err := inter.Invoke("singleAuth")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("single S2", func(t *testing.T) {

			value, err := inter.Invoke("singleS2")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("single R", func(t *testing.T) {

			value, err := inter.Invoke("singleR")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("double", func(t *testing.T) {

			value, err := inter.Invoke("double")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
		})

		t.Run("nonExistent", func(t *testing.T) {

			value, err := inter.Invoke("nonExistent")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})

		t.Run("loop", func(t *testing.T) {

			_, err := inter.Invoke("loop")
			require.Error(t, err)

			var cyclicLinkErr interpreter.CyclicLinkError
			require.ErrorAs(t, err, &cyclicLinkErr)

			require.Equal(t,
				cyclicLinkErr.Error(),
				"cyclic link in account 0x2a: /public/loop1 -> /public/loop2 -> /public/loop1",
			)
		})

		t.Run("singleTyped", func(t *testing.T) {

			value, err := inter.Invoke("singleTyped")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
		})

		t.Run("s2", func(t *testing.T) {

			value, err := inter.Invoke("s2")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
		})
	})
}

func TestInterpretCapability_address(t *testing.T) {

	t.Parallel()

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

	inter, _ := testAccount(
		t,
		address,
		true,
		`
			fun single(): Address {
				return account.getCapability(/public/single).address
			}

			fun double(): Address {
				return account.getCapability(/public/double).address
			}

			fun nonExistent() : Address {
				return account.getCapability(/public/nonExistent).address
			}				
		`,
	)

	t.Run("single", func(t *testing.T) {
		value, err := inter.Invoke("single")
		require.NoError(t, err)

		require.IsType(t, interpreter.AddressValue{}, value)
	})

	t.Run("double", func(t *testing.T) {
		value, err := inter.Invoke("double")
		require.NoError(t, err)

		require.IsType(t, interpreter.AddressValue{}, value)
	})

	t.Run("nonExistent", func(t *testing.T) {
		value, err := inter.Invoke("nonExistent")
		require.NoError(t, err)

		require.IsType(t, interpreter.AddressValue{}, value)
	})

}
