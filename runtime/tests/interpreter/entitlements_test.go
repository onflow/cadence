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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"

	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretEntitledReferenceRuntimeTypes(t *testing.T) {

	t.Parallel()

	t.Run("no entitlements", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
        resource R {}

        fun test(): Bool {
            return Type<&R>().isSubtype(of: Type<&R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("unentitled not <: auth", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
        resource R {}

        fun test(): Bool {
            return Type<&R>().isSubtype(of: Type<auth(X) &R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("auth <: unentitled", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
        resource R {}

        fun test(): Bool {
			return Type<auth(X) &R>().isSubtype(of: Type<&R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("auth <: auth", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
        resource R {}

        fun test(): Bool {
			return Type<auth(X) &R>().isSubtype(of: Type<auth(X) &R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("auth <: auth supertype", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        resource R {}

        fun test(): Bool {
			return Type<auth(X, Y) &R>().isSubtype(of: Type<auth(X) &R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})
	t.Run("created auth <: auth", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
        resource R {}

        fun test(): Bool {
			return ReferenceType(entitlements: ["S.test.X"], type: Type<@R>())!.isSubtype(of: Type<auth(X) &R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("created superset auth <: auth", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        resource R {}

        fun test(): Bool {
			return ReferenceType(entitlements: ["S.test.X", "S.test.Y"], type: Type<@R>())!.isSubtype(of: Type<auth(X) &R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("created different auth <: auth", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        resource R {}

        fun test(): Bool {
			return ReferenceType(entitlements: ["S.test.Y"], type: Type<@R>())!.isSubtype(of: Type<auth(X) &R>())
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})
}

func TestInterpretEntitledReferences(t *testing.T) {

	t.Parallel()

	t.Run("upcasting changes static entitlements", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y
			resource R {}
			fun test(): auth(X) &R {
				let r <- create R()
				account.save(<-r, to: /storage/foo)
				return account.borrow<auth(X) &R>(from: /storage/foo)!
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.X"},
				sema.Conjunction,
			).Equal(value.(*interpreter.StorageReferenceValue).Authorization),
		)
	})

	t.Run("upcasting and downcasting", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			access(all) fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let anyStruct = ref as AnyStruct
				let downRef = (anyStruct as? &Int)!
				let downDownRef = downRef as? auth(X) &Int
				return downDownRef == nil
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Equal(
			t,
			interpreter.TrueValue,
			value,
		)
	})
}

func TestInterpretEntitledReferenceCasting(t *testing.T) {
	t.Parallel()

	t.Run("subset downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X, Y) &Int
				let upRef = ref as auth(X) &Int
				let downRef = upRef as? auth(X, Y) &Int
				return downRef == nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("disjoint downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X, Y) &Int
				let upRef = ref as auth(X | Y) &Int
				let downRef = upRef as? auth(X) &Int
				return downRef == nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("wrong entitlement downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let upRef = ref as auth(X | Y) &Int
				let downRef = ref as? auth(Y) &Int
				return downRef != nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("correct entitlement downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let upRef = ref as auth(X | Y) &Int
				let downRef = upRef as? auth(X) &Int
				return downRef == nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("superset downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let downRef = ref as? auth(X, Y) &Int
				return downRef != nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("cast up to anystruct, cannot expand", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let anyStruct = ref as AnyStruct
				let downRef = anyStruct as? auth(X, Y) &Int
				return downRef != nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("cast up to anystruct, retains old type", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let anyStruct = ref as AnyStruct
				let downRef = anyStruct as? auth(X) &Int
				return downRef != nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("cast up to anystruct, then through reference, retains entitlement", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let anyStruct = ref as AnyStruct
				let downRef = anyStruct as! &Int
				let downDownRef = downRef as? auth(X) &Int
				return downDownRef != nil
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("entitled to nonentitled downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			resource interface RI {}

			resource R: RI {}

			entitlement E

			fun test(): Bool {
				let x <- create R()
				let r = &x as auth(E) &AnyResource{RI}
				let r2 = r as! &R{RI}
				let isSuccess = r2 != nil
				destroy x
				return isSuccess
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("order of entitlements doesn't matter", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement E
			entitlement F

			fun test(): Bool {
				let r = &1 as auth(E, F) &Int
				let r2 = r as!auth(F, E) &Int
				let isSuccess = r2 != nil
				return isSuccess
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("capability downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y

			fun test(): Bool {
				account.save(3, to: /storage/foo)
				let capX = account.getCapability<auth(X, Y) &Int>(/public/foo)
				let upCap = capX as Capability<auth(X) &Int>
				return upCap as? Capability<auth(X, Y) &Int> == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("unparameterized capability downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				account.save(3, to: /storage/foo)
				let capX = account.getCapability<auth(X) &Int>(/public/foo)
				let upCap = capX as Capability
				return upCap as? Capability<auth(X) &Int> == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("ref downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let arr: auth(X) &Int = &1
				let upArr = arr as &Int
				return upArr as? auth(X) &Int == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("optional ref downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let arr: auth(X) &Int? = &1
				let upArr = arr as &Int?
				return upArr as? auth(X) &Int? == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("ref array downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int] = [&1, &2]
				let upArr = arr as [&Int]
				return upArr as? [auth(X) &Int] == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("ref constant array downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int; 2] = [&1, &2]
				let upArr = arr as [&Int; 2]
				return upArr as? [auth(X) &Int; 2] == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("ref array element downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int] = [&1, &2]
				let upArr = arr as [&Int]
				return upArr[0] as? auth(X) &Int == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("ref constant array element downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int; 2] = [&1, &2]
				let upArr = arr as [&Int; 2]
				return upArr[0] as? auth(X) &Int == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("ref dict downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let dict: {String: auth(X) &Int} = {"foo": &3}
				let upDict = dict as {String: &Int}
				return upDict as? {String: auth(X) &Int} == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("ref dict element downcast forced", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X

			fun test(): Bool {
				let dict: {String: auth(X) &Int} = {"foo": &3}
				let upDict = dict as {String: &Int}
				return upDict["foo"]! as? auth(X) &Int == nil
			}
			`,
			sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

}

func TestInterpretCapabilityEntitlements(t *testing.T) {
	t.Parallel()

	t.Run("can borrow with supertype", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y
			resource R {}
			fun test(): &R {
				let r <- create R()
				account.save(<-r, to: /storage/foo)
				account.link<auth(X, Y) &R>(/public/foo, target: /storage/foo)
				let cap = account.getCapability(/public/foo)
				return cap.borrow<auth(X | Y) &R>()!
			}
			`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("cannot borrow with supertype then downcast", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y
			resource R {}
			fun test(): &R? {
				let r <- create R()
				account.save(<-r, to: /storage/foo)
				account.link<auth(X, Y) &R>(/public/foo, target: /storage/foo)
				let cap = account.getCapability(/public/foo)
				return cap.borrow<auth(X | Y) &R>()! as? auth(X, Y) &R
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NilOptionalValue,
			value,
		)
	})

	t.Run("can borrow with two types", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y
			resource R {}
			fun test(): &R {
				let r <- create R()
				account.save(<-r, to: /storage/foo)
				account.link<auth(X, Y) &R>(/public/foo, target: /storage/foo)
				let cap = account.getCapability(/public/foo)
				cap.borrow<auth(X | Y) &R>()! as? auth(X, Y) &R
				return cap.borrow<auth(X, Y) &R>()! as! auth(X, Y) &R
			}
			`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("upcast runtime entitlements", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			struct S {}
			fun test(): Bool {
				let s = S()
				account.save(s, to: /storage/foo)
				account.link<auth(X) &S>(/public/foo, target: /storage/foo)
				let cap: Capability<auth(X) &S> = account.getCapability<auth(X) &S>(/public/foo)
				let runtimeType = cap.getType() 
				let upcastCap = cap as Capability<&S> 
				let upcastRuntimeType = upcastCap.getType() 
				return runtimeType == upcastRuntimeType 
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("upcast runtime type", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			struct S {}
			fun test(): Bool {
				let s = S()
				account.save(s, to: /storage/foo)
				account.link<&S>(/public/foo, target: /storage/foo)
				let cap: Capability<&S> = account.getCapability<&S>(/public/foo)
				let runtimeType = cap.getType() 
				let upcastCap = cap as Capability<&AnyStruct> 
				let upcastRuntimeType = upcastCap.getType() 
				return runtimeType == upcastRuntimeType 
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("can check with supertype", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y
			resource R {}
			fun test(): Bool {
				let r <- create R()
				account.save(<-r, to: /storage/foo)
				account.link<auth(X, Y) &R>(/public/foo, target: /storage/foo)
				let cap = account.getCapability(/public/foo)
				return cap.check<auth(X | Y) &R>()
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("cannot borrow with subtype", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y
			resource R {}
			fun test(): &R {
				let r <- create R()
				account.save(<-r, to: /storage/foo)
				account.link<auth(X) &R>(/public/foo, target: /storage/foo)
				let cap = account.getCapability(/public/foo)
				return cap.borrow<auth(X, Y) &R>()!
			}
			`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		var nilErr interpreter.ForceNilError
		require.ErrorAs(t, err, &nilErr)
	})

	t.Run("cannot check with subtype", func(t *testing.T) {
		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t,
			address,
			true,
			`
			entitlement X
			entitlement Y
			resource R {}
			fun test(): Bool {
				let r <- create R()
				account.save(<-r, to: /storage/foo)
				account.link<auth(X) &R>(/public/foo, target: /storage/foo)
				let cap = account.getCapability(/public/foo)
				return cap.check<auth(X, Y) &R>()
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

}

func TestInterpretEntitledResult(t *testing.T) {
	t.Parallel()

	t.Run("valid upcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		resource R {
			view access(X, Y) fun foo(): Bool {
				return true
			}
		}
		fun bar(_ r: @R): @R {
			post {
				result as? auth(X | Y) &R != nil : "beep"
			}
			return <-r
		}
		fun test() {
			destroy bar(<-create R())
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.Void,
			value,
		)
	})

	t.Run("invalid downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		resource R {
			view access(X) fun foo(): Bool {
				return true
			}
		}
		fun bar(_ r: @R): @R {
			post {
				result as? auth(X, Y) &R != nil : "beep"
			}
			return <-r
		}
		fun test() {
			destroy bar(<-create R())
		}
		`)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		var conditionError interpreter.ConditionError
		require.ErrorAs(t, err, &conditionError)
	})
}

func TestInterpretEntitlementMappingFields(t *testing.T) {
	t.Parallel()
	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement mapping M {
			X -> Y
		}
		struct S {
			access(M) let foo: auth(M) &Int
			init() {
				self.foo = &2 as auth(Y) &Int
			}
		}
		fun test(): auth(Y) &Int {
			let s = S()
			let ref = &s as auth(X) &S
			let i = ref.foo
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		var refType *interpreter.EphemeralReferenceValue
		require.IsType(t, value, refType)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)

		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(2),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})

	t.Run("map applies to static types at runtime", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) let foo: auth(M) &Int
			init() {
				self.foo = &3 as auth(F, Y) &Int
			}
		}
		fun test(): auth(Y) &Int {
			let s = S()
			let ref = &s as auth(X, E) &S
			let upref = ref as auth(X) &S
			let i = upref.foo
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		var refType *interpreter.EphemeralReferenceValue
		require.IsType(t, value, refType)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)

		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})

	t.Run("does not generate types with no input", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) let foo: auth(M) &Int
			init() {
				self.foo = &3 as auth(F, Y) &Int
			}
		}
		fun test(): auth(Y) &Int {
			let s = S()
			let ref = &s as auth(X) &S
			let i = ref.foo
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		var refType *interpreter.EphemeralReferenceValue
		require.IsType(t, value, refType)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)

		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})

	t.Run("fully entitled", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) let foo: auth(M) &Int
			init() {
				self.foo = &3 as auth(F, Y) &Int
			}
		}
		fun test(): auth(F, Y) &Int {
			let s = S()
			let i = s.foo
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		var refType *interpreter.EphemeralReferenceValue
		require.IsType(t, value, refType)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y", "S.test.F"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)

		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})

	t.Run("fully entitled but less than initialized", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement Q
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) let foo: auth(M) &Int
			init() {
				self.foo = &3 as auth(F, Y, Q) &Int
			}
		}
		fun test(): auth(Y, F) &Int {
			let s = S()
			let i = s.foo
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		var refType *interpreter.EphemeralReferenceValue
		require.IsType(t, value, refType)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y", "S.test.F"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)

		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})

	t.Run("optional value", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) let foo: auth(M) &Int
			init() {
				self.foo = &3 as auth(F, Y) &Int
			}
		}
		fun test(): auth(Y) &Int {
			let s = S()
			let ref = &s as auth(X) &S?
			let i = ref?.foo
			return i!
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		var refType *interpreter.EphemeralReferenceValue
		require.IsType(t, value, refType)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)

		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})

	t.Run("storage reference value", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(self) let myFoo: Int
			access(M) fun foo(): auth(M) &Int {
				return &self.myFoo as auth(M) &Int
			}
			init() {
				self.myFoo = 3
			}
		}
		fun test(): auth(Y) &Int {
			let s = S()
			account.save(s, to: /storage/foo)
			let ref = account.borrow<auth(X) &S>(from: /storage/foo)
			let i = ref?.foo()
			return i!
		}
		`, sema.Config{
			AttachmentsEnabled: false,
		})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		var refType *interpreter.EphemeralReferenceValue
		require.IsType(t, value, refType)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)

		require.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value.(*interpreter.EphemeralReferenceValue).Value,
		)
	})
}

func TestInterpretEntitlementMappingAccessors(t *testing.T) {

	t.Parallel()
	t.Run("basic", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement mapping M {
			X -> Y
		}
		struct S {
			access(M) fun foo(): auth(M) &Int {
				return &1 as auth(M) &Int
			}
		}
		fun test(): auth(Y) &Int {
			let s = S()
			let ref = &s as auth(X) &S
			let i = ref.foo()
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic with subtype return", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement Z
		entitlement mapping M {
			X -> Y
		}
		struct S {
			access(M) fun foo(): auth(M) &Int {
				return &1 as auth(Y, Z) &Int
			}
		}
		fun test(): Bool {
			let s = S()
			let ref = &s as auth(X) &S
			let i = ref.foo()
			return (i as? auth(Y, Z) &Int) == nil
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Equal(
			t,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("basic owned", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement Z
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct S {
			access(M) fun foo(): auth(M) &Int {
				return &1 as auth(M) &Int
			}
		}
		fun test(): auth(Y, Z) &Int {
			let s = S()
			let i = s.foo()
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y", "S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("optional chain", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement Z
		entitlement E
		entitlement mapping M {
			X -> Y
			X -> Z
		}
		struct S {
			access(M) fun foo(): auth(M) &Int {
				return &1 as auth(M) &Int
			}
		}
		fun test(): auth(Y, Z) &Int {
			let s = S()
			let ref: auth(X, E) &S? = &s
			let i = ref?.foo()
			return i!
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y", "S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("downcasting", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) fun foo(): auth(M) &Int? {
				let ref = &1 as auth(F) &Int
				// here M is substituted for F, so this works
				if let r = ref as? auth(M) &Int {
					return r
				} else {
					return nil
				}
			}
		}
		fun test(): auth(F) &Int {
			let s = S()
			let ref = &s as auth(E) &S
			let i = ref.foo()!
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.F"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("downcasting fail", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) fun foo(): auth(M) &Int? {
				let ref = &1 as auth(F) &Int
				// here M is substituted for Y, so this fails
				if let r = ref as? auth(M) &Int {
					return r
				} else {
					return nil
				}
			}
		}
		fun test(): Bool {
			let s = S()
			let ref = &s as auth(X) &S
			let i = ref.foo()
			return i == nil
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Equal(
			t,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("downcasting nested success", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) fun foo(): auth(M) &Int? {
				let x = [&1 as auth(Y) &Int]
				let y = x as! [auth(M) &Int]
				return y[0]
			}
		}
		fun test(): Bool {
			let s = S()
			let refX = &s as auth(X) &S
			return refX.foo() != nil
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Equal(
			t,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("downcasting nested fail", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) fun foo(): auth(M) &Int? {
				let x = [&1 as auth(Y) &Int]
				let y = x as! [auth(M) &Int]
				return y[0]
			}
		}
		fun test(): Bool {
			let s = S()
			let refE = &s as auth(E) &S
			return refE.foo() != nil
		}
		`)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		var forceCastErr interpreter.ForceCastTypeMismatchError
		require.ErrorAs(t, err, &forceCastErr)
	})

	t.Run("downcasting fail", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
		entitlement E
		entitlement F
		entitlement mapping M {
			X -> Y
			E -> F
		}
		struct S {
			access(M) fun foo(): auth(M) &Int? {
				let ref = &1 as auth(F) &Int
				if let r = ref as? auth(M) &Int {
					return r
				} else {
					return nil
				}
			}
		}
		fun test(): &Int? {
			let s = S()
			let ref = &s as auth(X) &S
			let i = ref.foo()
			return i
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Equal(
			t,
			interpreter.Nil,
			value,
		)
	})

	t.Run("nested object access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y
			entitlement mapping M {
				X -> Y
			}
			struct T {
				access(Y) fun getRef(): auth(Y) &Int {
					return &1 as auth(Y) &Int
				}
			}
			struct S {
				access(M) let t: auth(M) &T
				access(M) fun foo(): auth(M) &Int {
					// success because we have self is fully entitled to the domain of M
					return self.t.getRef() 
				}
				init() {
					self.t = &T() as auth(Y) &T
				}
			}
			fun test(): auth(Y) &Int {
				let s = S()
				let ref = &s as auth(X) &S
				return ref.foo()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("nested mapping access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y
			entitlement Z
			entitlement mapping M {
				X -> Y
			}
			entitlement mapping N {
				Y -> Z
			}
			struct T {
				access(N) fun getRef(): auth(N) &Int {
					return &1 as auth(N) &Int
				}
			}
			struct S {
				access(M) let t: auth(M) &T
				access(X) fun foo(): auth(Z) &Int {
					return self.t.getRef() 
				}
				init() {
					self.t = &T() as auth(Y) &T
				}
			}
			fun test(): auth(Z) &Int {
				let s = S()
				let ref = &s as auth(X) &S
				return ref.foo()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("composing mapping access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y
			entitlement Z
			entitlement mapping M {
				X -> Y
			}
			entitlement mapping N {
				Y -> Z
			}
			entitlement mapping NM {
				X -> Z
			}
			struct T {
				access(N) fun getRef(): auth(N) &Int {
					return &1 as auth(N) &Int
				}
			}
			struct S {
				access(M) let t: auth(M) &T
				access(NM) fun foo(): auth(NM) &Int {
					return self.t.getRef() 
				}
				init() {
					self.t = &T() as auth(Y) &T
				}
			}
			fun test(): auth(Z) &Int {
				let s = S()
				let ref = &s as auth(X) &S
				return ref.foo()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("superset composing mapping access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y
			entitlement Z
			entitlement A 
			entitlement B
			entitlement mapping M {
				X -> Y
				A -> B
			}
			entitlement mapping N {
				Y -> Z
			}
			entitlement mapping NM {
				X -> Z
			}
			struct T {
				access(N) fun getRef(): auth(N) &Int {
					return &1 as auth(N) &Int
				}
			}
			struct S {
				access(M) let t: auth(M) &T
				access(NM) fun foo(): auth(NM) &Int {
					return self.t.getRef() 
				}
				init() {
					self.t = &T() as auth(Y, B) &T
				}
			}
			fun test(): auth(Z) &Int {
				let s = S()
				let ref = &s as auth(X, A) &S
				return ref.foo()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("composing mapping access with intermediate step", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y
			entitlement Z
			entitlement A 
			entitlement B
			entitlement mapping M {
				X -> Y
				A -> B
			}
			entitlement mapping N {
				Y -> Z
				B -> B
			}
			entitlement mapping NM {
				X -> Z
				A -> B
			}
			struct T {
				access(N) fun getRef(): auth(N) &Int {
					return &1 as auth(N) &Int
				}
			}
			struct S {
				access(M) let t: auth(M) &T
				access(NM) fun foo(): auth(NM) &Int {
					return self.t.getRef() 
				}
				init() {
					self.t = &T() as auth(Y, B) &T
				}
			}
			fun test(): auth(Z, B) &Int {
				let s = S()
				let ref = &s as auth(X, A) &S
				return ref.foo()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.B", "S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})
}

func TestInterpretEntitledAttachments(t *testing.T) {
	t.Parallel()

	t.Run("basic access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement mapping M {
				X -> Y
				X -> Z
			}
			struct S {}
			access(M) attachment A for S {}
			fun test(): auth(Y, Z) &A {
				let s = attach A() to S()
				return s[A]!
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y", "S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic call", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement mapping M {
				X -> Y
				X -> Z
			}
			struct S {}
			access(M) attachment A for S {
				access(Y | Z) fun entitled(): auth(Y, Z) &A {
					return self
				} 
			}
			fun test(): auth(Y, Z) &A {
				let s = attach A() to S()
				return s[A]!.entitled()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.Y", "S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic call return base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement mapping M {
				X -> Y
				X -> Z
			}
			struct S {}
			access(M) attachment A for S {
				access(Y | Z) fun entitled(): &S {
					return base
				} 
			}
			fun test(): &S {
				let s = attach A() to S()
				return s[A]!.entitled()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.UnauthorizedAccess.Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic call return authorized base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement mapping M {
				X -> Y
				X -> Z
			}
			struct S {}
			access(M) attachment A for S {
				require entitlement X
				access(Y | Z) fun entitled(): auth(X) &S {
					return base
				} 
			}
			fun test(): auth(X) &S {
				let s = attach A() to S() with (X)
				return s[A]!.entitled()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.X"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic ref access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			struct S {}
			access(M) attachment A for S {}
			fun test(): auth(F, G) &A {
				let s = attach A() to S()
				let ref = &s as auth(E) &S
				return ref[A]!
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.F", "S.test.G"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic ref call", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			struct S {}
			access(M) attachment A for S {
				access(F | Z) fun entitled(): auth(Y, Z, F, G) &A {
					return self
				} 
			}
			fun test(): auth(Y, Z, F, G) &A {
				let s = attach A() to S()
				let ref = &s as auth(E) &S
				return ref[A]!.entitled()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.F", "S.test.G", "S.test.Y", "S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic ref call return base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			struct S {}
			access(M) attachment A for S {
				access(F | Z) fun entitled(): &S {
					return base
				} 
			}
			fun test(): &S {
				let s = attach A() to S()
				let ref = &s as auth(E) &S
				return ref[A]!.entitled()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.UnauthorizedAccess.Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic ref call return entitled base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			struct S {}
			access(M) attachment A for S {
				require entitlement E
				access(F | Z) fun entitled(): auth(E) &S {
					return base
				} 
			}
			fun test(): auth(E) &S {
				let s = attach A() to S() with(E)
				let ref = &s as auth(E) &S
				return ref[A]!.entitled()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.E"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic intersection access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			struct S: I {}
			struct interface I {}
			access(M) attachment A for I {}
			fun test(): auth(F, G) &A {
				let s = attach A() to S()
				let ref = &s as auth(E) &{I}
				return ref[A]!
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.F", "S.test.G"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref access", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			resource R {}
			access(M) attachment A for R {}
			fun test(): auth(F, G) &A {
				let r <- attach A() to <-create R()
				account.save(<-r, to: /storage/foo)
				let ref = account.borrow<auth(E) &R>(from: /storage/foo)!
				return ref[A]!
			}
		`, sema.Config{
			AttachmentsEnabled: true,
		})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.F", "S.test.G"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref call", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			resource R {}
			access(M) attachment A for R {
				access(F | Z) fun entitled(): auth(F, G, Y, Z) &A {
					return self
				} 
			}
			fun test(): auth(F, G, Y, Z) &A {
				let r <- attach A() to <-create R()
				account.save(<-r, to: /storage/foo)
				let ref = account.borrow<auth(E) &R>(from: /storage/foo)!
				return ref[A]!.entitled()
			}
		`, sema.Config{
			AttachmentsEnabled: true,
		})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.F", "S.test.G", "S.test.Y", "S.test.Z"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref call base", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			resource R {}
			access(M) attachment A for R {
				require entitlement X
				access(F | Z) fun entitled(): auth(X) &R {
					return base
				} 
			}
			fun test(): auth(X) &R {
				let r <- attach A() to <-create R() with (X)
				account.save(<-r, to: /storage/foo)
				let ref = account.borrow<auth(E) &R>(from: /storage/foo)!
				return ref[A]!.entitled()
			}
		`, sema.Config{
			AttachmentsEnabled: true,
		})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.X"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("fully entitled in init and destroy", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z 
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				X -> Z
				E -> F
				X -> F
				E -> G
			}
			resource R {}
			access(M) attachment A for R {
				require entitlement E
				require entitlement X
				init() {
					let x = self as! auth(Y, Z, F, G) &A
					let y = base as! auth(X, E) &R
				}
				destroy() {
					let x = self as! auth(Y, Z, F, G) &A
					let y = base as! auth(X, E) &R
				}
			}
			fun test() {
				let r <- attach A() to <-create R() with (E, X)
				destroy r
			}
		`)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composed mapped attachment access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			entitlement Z
			entitlement E
			entitlement F
			entitlement G
			entitlement mapping M {
				X -> Y
				E -> F
			}
			entitlement mapping N {
				Z -> X
				G -> F
			}
			struct S {}
			struct T {}
			access(M) attachment A for S {
				access(self) let t: T
				init(t: T) {
					self.t = t
				}
				access(Y) fun getT(): auth(Z) &T {
					return &self.t as auth(Z) &T
				}
			}
			access(N) attachment B for T {}
			fun test(): auth(X) &B {
				let s = attach A(t: attach B() to T()) to S()
				let ref = &s as auth(X) &S
				return ref[A]!.getT()[B]!
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.X"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})
}

func TestInterpretEntitledReferenceCollections(t *testing.T) {
	t.Parallel()

	t.Run("arrays", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			fun test(): auth(X) &Int {
				let arr: [auth(X) &Int] = [&1 as auth(X) &Int]
				arr.append(&2 as auth(X, Y) &Int)
				arr.append(&3 as auth(X) &Int)
				return arr[1]
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.X"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("dict", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y 
			fun test(): auth(X) &Int {
				let dict: {String: auth(X) &Int} = {"one": &1 as auth(X) &Int}
				dict.insert(key: "two", &2 as auth(X, Y) &Int)
				dict.insert(key: "three", &3 as auth(X) &Int)
				return dict["two"]!
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.X"},
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})
}

func TestInterpretEntitlementSetEquality(t *testing.T) {
	t.Parallel()

	t.Run("different sigils", func(t *testing.T) {

		t.Parallel()

		conjunction := interpreter.NewEntitlementSetAuthorization(
			nil,
			[]common.TypeID{"S.test.X"},
			sema.Conjunction,
		)

		disjunction := interpreter.NewEntitlementSetAuthorization(
			nil,
			[]common.TypeID{"S.test.X"},
			sema.Disjunction,
		)

		require.False(t, conjunction.Equal(disjunction))
		require.False(t, disjunction.Equal(conjunction))
	})

	t.Run("different lengths", func(t *testing.T) {

		t.Parallel()

		one := interpreter.NewEntitlementSetAuthorization(
			nil,
			[]common.TypeID{"S.test.X"},
			sema.Conjunction,
		)

		two := interpreter.NewEntitlementSetAuthorization(
			nil,
			[]common.TypeID{"S.test.X", "S.test.Y"},
			sema.Conjunction,
		)

		require.False(t, one.Equal(two))
		require.False(t, two.Equal(one))
	})
}
