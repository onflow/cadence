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

	"github.com/stretchr/testify/assert"
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

	t.Run("subtype comparison order irrelevant", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        resource R {}

        fun test(): Bool {
			return ReferenceType(entitlements: ["S.test.X", "S.test.Y"], type: Type<@R>())!.isSubtype(of: 
				   ReferenceType(entitlements: ["S.test.Y", "S.test.X"], type: Type<@R>())!
			) && ReferenceType(entitlements: ["S.test.Y", "S.test.X"], type: Type<@R>())!.isSubtype(of: 
				 ReferenceType(entitlements: ["S.test.X", "S.test.Y"], type: Type<@R>())!
			)
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

	t.Run("equality", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        resource R {}

        fun test(): Bool {
			return ReferenceType(entitlements: ["S.test.X", "S.test.Y"], type: Type<@R>())! ==
				   ReferenceType(entitlements: ["S.test.Y", "S.test.X"], type: Type<@R>())! && 
				   Type<auth(X, Y) &R>() == Type<auth(Y, X) &R>()
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

	t.Run("order irrelevant as dictionary key", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        resource R {}

        fun test(): Int {
			let runtimeType1 = ReferenceType(entitlements: ["S.test.X", "S.test.Y"], type: Type<@R>())!
			let runtimeType2 = ReferenceType(entitlements: ["S.test.Y", "S.test.X"], type: Type<@R>())!

			let dict = {runtimeType1 : 3}
			return dict[runtimeType2]!
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value,
		)
	})

	t.Run("order irrelevant as dictionary key when obtained from Type<>", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        resource R {}

        fun test(): Int {
			let runtimeType1 = Type<auth(X, Y) &R>()
			let runtimeType2 = Type<auth(Y, X) &R>()

			let dict = {runtimeType1 : 3}
			return dict[runtimeType2]!
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			value,
		)
	})

	t.Run("order irrelevant as dictionary key when obtained from .getType()", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement X
		entitlement Y
        struct S {}

        fun test(): Int {
			let runtimeType1 = [&S() as auth(X, Y) &S].getType()
			let runtimeType2 = [&S() as auth(Y, X) &S].getType()

			let dict = {runtimeType1 : 3}
			return dict[runtimeType2]!
        }
    `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(3),
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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X
			entitlement Y
			resource R {}
			fun test(): auth(X) &R {
				let r <- create R()
				account.storage.save(<-r, to: /storage/foo)
				return account.storage.borrow<auth(X) &R>(from: /storage/foo)!
			}
			`, sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.StorageReferenceValue).Authorization),
		)
	})

	t.Run("upcasting and downcasting", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X
			access(all) fun test(): Bool {
				let ref = &1 as auth(X) &Int
				let anyStruct = ref as AnyStruct
				let downRef = (anyStruct as? &Int)!
				let downDownRef = downRef as? auth(X) &Int
				return downDownRef == nil
			}
			`, sema.Config{})

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
				let r = &x as auth(E) &{RI}
				let r2 = r as! &{RI}
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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X
			entitlement Y

			fun test(capXY: Capability<auth(X, Y) &Int>): Bool {
				let upCap = capXY as Capability<auth(X) &Int>
				return upCap as? Capability<auth(X, Y) &Int> == nil
			}
			`, sema.Config{})

		capXY := interpreter.NewCapabilityValue(
			nil,
			interpreter.NewUnmeteredUInt64Value(1),
			address,
			interpreter.NewReferenceStaticType(
				nil,
				interpreter.NewEntitlementSetAuthorization(
					nil,
					func() []common.TypeID { return []common.TypeID{"S.test.X", "S.test.Y"} },
					2,
					sema.Conjunction,
				),
				interpreter.PrimitiveStaticTypeInt,
			),
		)

		value, err := inter.Invoke("test", capXY)
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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(capX: Capability<auth(X) &Int>): Capability {
				let upCap = capX as Capability
				return (upCap as? Capability<auth(X) &Int>)!
			}
			`, sema.Config{})

		capX := interpreter.NewCapabilityValue(
			nil,
			interpreter.NewUnmeteredUInt64Value(1),
			address,
			interpreter.NewReferenceStaticType(
				nil,
				interpreter.NewEntitlementSetAuthorization(nil,
					func() []common.TypeID { return []common.TypeID{"S.test.X"} },
					1,
					sema.Conjunction),
				interpreter.PrimitiveStaticTypeInt,
			),
		)

		value, err := inter.Invoke("test", capX)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			capX,
			value,
		)
	})

	t.Run("ref downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let arr: auth(X) &Int = &1
				let upArr = arr as &Int
				return upArr as? auth(X) &Int == nil
			}
			`, sema.Config{})

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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let one: Int? = 1
				let arr: auth(X) &Int? = &one
				let upArr = arr as &Int?
				return upArr as? auth(X) &Int? == nil
			}
			`, sema.Config{})

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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int] = [&1, &2]
				let upArr = arr as [&Int]
				return upArr as? [auth(X) &Int] == nil
			}
			`, sema.Config{})

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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int; 2] = [&1, &2]
				let upArr = arr as [&Int; 2]
				return upArr as? [auth(X) &Int; 2] == nil
			}
			`, sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			value,
		)
	})

	t.Run("ref constant array downcast no change", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int; 2] = [&1, &2]
				let upArr = arr as [auth(X) &Int; 2]
				return upArr as? [auth(X) &Int; 2] == nil
			}
			`, sema.Config{})

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			value,
		)
	})

	t.Run("ref array element downcast", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int] = [&1, &2]
				let upArr = arr as [&Int]
				return upArr[0] as? auth(X) &Int == nil
			}
			`, sema.Config{})

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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let arr: [auth(X) &Int; 2] = [&1, &2]
				let upArr = arr as [&Int; 2]
				return upArr[0] as? auth(X) &Int == nil
			}
			`, sema.Config{})

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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let dict: {String: auth(X) &Int} = {"foo": &3}
				let upDict = dict as {String: &Int}
				return upDict as? {String: auth(X) &Int} == nil
			}
			`, sema.Config{})

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

		inter, _ := testAccount(t, address, true, nil, `
			entitlement X

			fun test(): Bool {
				let dict: {String: auth(X) &Int} = {"foo": &3}
				let upDict = dict as {String: &Int}
				return upDict["foo"]! as? auth(X) &Int == nil
			}
			`, sema.Config{})

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
			access(mapping M) let foo: auth(mapping M) &Int
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
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
			access(mapping M) let foo: auth(mapping M) &Int
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
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
			access(mapping M) let foo: auth(mapping M) &Int
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
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
			access(mapping M) let foo: auth(mapping M) &Int
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y", "S.test.F"} },
				2,
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
			access(mapping M) let foo: auth(mapping M) &Int
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y", "S.test.F"} },
				2,
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
			access(mapping M) let foo: auth(mapping M) &Int
			init() {
				self.foo = &3 as auth(F, Y) &Int
			}
		}
		fun test(): auth(Y) &Int {
			let s: S? = S()
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
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

		inter, _ := testAccount(t, address, true, nil, `
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
			access(mapping M) fun foo(): auth(mapping M) &Int {
				return &self.myFoo as auth(mapping M) &Int
			}
			init() {
				self.myFoo = 3
			}
		}
		fun test(): auth(Y) &Int {
			let s = S()
			account.storage.save(s, to: /storage/foo)
			let ref = account.storage.borrow<auth(X) &S>(from: /storage/foo)
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
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
			access(mapping M) fun foo(): auth(mapping M) &Int {
				return &1 as auth(mapping M) &Int
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
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
			access(mapping M) fun foo(): auth(mapping M) &Int {
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
			access(mapping M) fun foo(): auth(mapping M) &Int {
				return &1 as auth(mapping M) &Int
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y", "S.test.Z"} },
				2,
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
			access(mapping M) fun foo(): auth(mapping M) &Int {
				return &1 as auth(mapping M) &Int
			}
		}
		fun test(): auth(Y, Z) &Int {
			let s: S? = S()
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y", "S.test.Z"} },
				2,
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
			access(mapping M) fun foo(): auth(mapping M) &Int? {
				let ref = &1 as auth(F) &Int
				// here M is substituted for F, so this works
				if let r = ref as? auth(mapping M) &Int {
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
				func() []common.TypeID { return []common.TypeID{"S.test.F"} },
				1,
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
			access(mapping M) fun foo(): auth(mapping M) &Int? {
				let ref = &1 as auth(F) &Int
				// here M is substituted for Y, so this fails
				if let r = ref as? auth(mapping M) &Int {
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
			access(mapping M) fun foo(): auth(mapping M) &Int? {
				let x = [&1 as auth(Y) &Int]
				let y = x as! [auth(mapping M) &Int]
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
			access(mapping M) fun foo(): auth(mapping M) &Int? {
				let x = [&1 as auth(Y) &Int]
				let y = x as! [auth(mapping M) &Int]
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
			access(mapping M) fun foo(): auth(mapping M) &Int? {
				let ref = &1 as auth(F) &Int
				if let r = ref as? auth(mapping M) &Int {
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
				access(mapping M) let t: auth(mapping M) &T
				access(mapping M) fun foo(): auth(mapping M) &Int {
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
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
				access(mapping N) fun getRef(): auth(mapping N) &Int {
					return &1 as auth(mapping N) &Int
				}
			}
			struct S {
				access(mapping M) let t: auth(mapping M) &T
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
				func() []common.TypeID { return []common.TypeID{"S.test.Z"} },
				1,
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
				access(mapping N) fun getRef(): auth(mapping N) &Int {
					return &1 as auth(mapping N) &Int
				}
			}
			struct S {
				access(mapping M) let t: auth(mapping M) &T
				access(mapping NM) fun foo(): auth(mapping NM) &Int {
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
				func() []common.TypeID { return []common.TypeID{"S.test.Z"} },
				1,
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
				access(mapping N) fun getRef(): auth(mapping N) &Int {
					return &1 as auth(mapping N) &Int
				}
			}
			struct S {
				access(mapping M) let t: auth(mapping M) &T
				access(mapping NM) fun foo(): auth(mapping NM) &Int {
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
				func() []common.TypeID { return []common.TypeID{"S.test.Z"} },
				1,
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
				access(mapping N) fun getRef(): auth(mapping N) &Int {
					return &1 as auth(mapping N) &Int
				}
			}
			struct S {
				access(mapping M) let t: auth(mapping M) &T
				access(mapping NM) fun foo(): auth(mapping NM) &Int {
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
				func() []common.TypeID { return []common.TypeID{"S.test.B", "S.test.Z"} },
				2,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("accessor function with mapped ref arg", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement E
            entitlement F
            entitlement G
            entitlement H
            entitlement mapping M {
                E -> F
                G -> H
            }
            struct S {
                access(mapping M) fun foo(_ arg: auth(mapping M) &Int): auth(mapping M) &Int {
					return arg
				}
            }

            fun test(): auth(F) &Int {
				let s = S()
				let sRef = &s as auth(E) &S
                return sRef.foo(&1)
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"S.test.F"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("accessor function with full mapped ref arg", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement E
            entitlement F
            entitlement G
            entitlement H
            entitlement mapping M {
                E -> F
                G -> H
            }
            struct S {
                access(mapping M) fun foo(_ arg: auth(mapping M) &Int): auth(mapping M) &Int {
					return arg
				}
            }

            fun test(): auth(F, H) &Int {
				let s = S()
                return s.foo(&1)
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"S.test.F", "S.test.H"} },
				2,
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
			entitlement Y 
			entitlement Z 
			struct S {
				access(Y, Z) fun foo() {}
			}
			access(all) attachment A for S {}
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y", "S.test.Z"} },
				2,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic call", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement Y 
			entitlement Z 
			struct S {
				access(Y | Z) fun foo() {}
			}
			access(all) attachment A for S {
				access(Y | Z) fun entitled(): auth(Y | Z) &A {
					return self
				} 
			}
			fun test(): auth(Y | Z) &A {
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
				func() []common.TypeID { return []common.TypeID{"S.test.Y", "S.test.Z"} },
				2,
				sema.Disjunction,
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
			access(mapping M) attachment A for S {
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
			access(mapping M) attachment A for S {
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
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
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
			access(mapping M) attachment A for S {}
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
				func() []common.TypeID { return []common.TypeID{"S.test.F", "S.test.G"} },
				2,
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
			access(mapping M) attachment A for S {
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
				func() []common.TypeID { return []common.TypeID{"S.test.F", "S.test.G", "S.test.Y", "S.test.Z"} },
				4,
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
			access(mapping M) attachment A for S {
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
			access(mapping M) attachment A for S {
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
				func() []common.TypeID { return []common.TypeID{"S.test.E"} },
				1,
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
			access(mapping M) attachment A for I {}
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
				func() []common.TypeID { return []common.TypeID{"S.test.F", "S.test.G"} },
				2,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref access", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
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
			access(mapping M) attachment A for R {}
			fun test(): auth(F, G) &A {
				let r <- attach A() to <-create R()
				account.storage.save(<-r, to: /storage/foo)
				let ref = account.storage.borrow<auth(E) &R>(from: /storage/foo)!
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
				func() []common.TypeID { return []common.TypeID{"S.test.F", "S.test.G"} },
				2,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref call", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
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
			access(mapping M) attachment A for R {
				access(F | Z) fun entitled(): auth(F, G, Y, Z) &A {
					return self
				} 
			}
			fun test(): auth(F, G, Y, Z) &A {
				let r <- attach A() to <-create R()
				account.storage.save(<-r, to: /storage/foo)
				let ref = account.storage.borrow<auth(E) &R>(from: /storage/foo)!
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
				func() []common.TypeID { return []common.TypeID{"S.test.F", "S.test.G", "S.test.Y", "S.test.Z"} },
				4,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref call base", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil, `
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
			access(mapping M) attachment A for R {
				require entitlement X
				access(F | Z) fun entitled(): auth(X) &R {
					return base
				} 
			}
			fun test(): auth(X) &R {
				let r <- attach A() to <-create R() with (X)
				account.storage.save(<-r, to: /storage/foo)
				let ref = account.storage.borrow<auth(E) &R>(from: /storage/foo)!
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
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
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
			access(mapping M) attachment A for R {
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
			access(mapping M) attachment A for S {
				access(self) let t: T
				init(t: T) {
					self.t = t
				}
				access(Y) fun getT(): auth(Z) &T {
					return &self.t as auth(Z) &T
				}
			}
			access(mapping N) attachment B for T {}
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
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("empty output", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement A
			entitlement B

			entitlement mapping M {
			    A -> B
			}

			struct S {
				access(mapping M) fun foo(): auth(mapping M) &AnyStruct {
					let a: AnyStruct = "hello"
					return &a as auth(mapping M) &AnyStruct
				}
			}

			fun test(): &AnyStruct {
				let s = S()
				let ref = &s as &S

				// Must return an unauthorized ref
				return ref.foo()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Equal(
			t,
			interpreter.UnauthorizedAccess,
			value.(*interpreter.EphemeralReferenceValue).Authorization,
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
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
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
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
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
			func() []common.TypeID { return []common.TypeID{"S.test.X"} },
			1,
			sema.Conjunction,
		)

		disjunction := interpreter.NewEntitlementSetAuthorization(
			nil,
			func() []common.TypeID { return []common.TypeID{"S.test.X"} },
			1,
			sema.Disjunction,
		)

		require.False(t, conjunction.Equal(disjunction))
		require.False(t, disjunction.Equal(conjunction))
	})

	t.Run("different lengths", func(t *testing.T) {

		t.Parallel()

		one := interpreter.NewEntitlementSetAuthorization(
			nil,
			func() []common.TypeID { return []common.TypeID{"S.test.X"} },
			1,
			sema.Conjunction,
		)

		two := interpreter.NewEntitlementSetAuthorization(
			nil,
			func() []common.TypeID { return []common.TypeID{"S.test.X", "S.test.Y"} },
			2,
			sema.Conjunction,
		)

		require.False(t, one.Equal(two))
		require.False(t, two.Equal(one))
	})
}

func TestInterpretBuiltinEntitlements(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        struct S {
            access(Mutate) fun foo() {}
            access(Insert) fun bar() {}
            access(Remove) fun baz() {}
        }

        fun main() {
            let s = S()
            let mutableRef = &s as auth(Mutate) &S
            let insertableRef = &s as auth(Insert) &S
            let removableRef = &s as auth(Remove) &S
        }
    `)

	_, err := inter.Invoke("main")
	assert.NoError(t, err)
}

func TestInterpretIdentityMapping(t *testing.T) {

	t.Parallel()

	t.Run("owned value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                // OK: Must return an unauthorized ref
                let resultRef1: &AnyStruct = s.foo()
            }
        `)

		_, err := inter.Invoke("main")
		assert.NoError(t, err)
	})

	t.Run("unauthorized ref", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                let ref = &s as &S

                // OK: Must return an unauthorized ref
                let resultRef1: &AnyStruct = ref.foo()
            }
        `)

		_, err := inter.Invoke("main")
		assert.NoError(t, err)
	})

	t.Run("basic entitled ref", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                let mutableRef = &s as auth(Mutate) &S
                let ref1: auth(Mutate) &AnyStruct = mutableRef.foo()

                let insertableRef = &s as auth(Insert) &S
                let ref2: auth(Insert) &AnyStruct = insertableRef.foo()

                let removableRef = &s as auth(Remove) &S
                let ref3: auth(Remove) &AnyStruct = removableRef.foo()
            }
        `)

		_, err := inter.Invoke("main")
		assert.NoError(t, err)
	})

	t.Run("entitlement set ref", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            struct S {
                access(mapping Identity) fun foo(): auth(mapping Identity) &AnyStruct {
                    let a: AnyStruct = "hello"
                    return &a as auth(mapping Identity) &AnyStruct
                }
            }

            fun main() {
                let s = S()

                let ref1 = &s as auth(Insert | Remove) &S
                let resultRef1: auth(Insert | Remove) &AnyStruct = ref1.foo()

                let ref2 = &s as auth(Insert, Remove) &S
                let resultRef2: auth(Insert, Remove) &AnyStruct = ref2.foo()
            }
        `)

		_, err := inter.Invoke("main")
		assert.NoError(t, err)
	})

	t.Run("owned value, with entitlements", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement A
            entitlement B
            entitlement C

            struct X {
               access(A | B) var s: String
               init() {
                   self.s = "hello"
               }
               access(C) fun foo() {}
            }

            struct Y {

                // Reference
                access(mapping Identity) var x1: auth(mapping Identity) &X

                // Optional reference
                access(mapping Identity) var x2: auth(mapping Identity) &X?

                // Function returning a reference
                access(mapping Identity) fun getX(): auth(mapping Identity) &X {
                    let x = X()
                    return &x as auth(mapping Identity) &X
                }

                // Function returning an optional reference
                access(mapping Identity) fun getOptionalX(): auth(mapping Identity) &X? {
                    let x: X? = X()
                    return &x as auth(mapping Identity) &X?
                }

                init() {
                    let x = X()
                    self.x1 = &x as auth(A, B, C) &X
                    self.x2 = nil
                }
            }

            fun main() {
                let y = Y()

                let ref1: auth(A, B, C) &X = y.x1

                let ref2: auth(A, B, C) &X? = y.x2

                let ref3: auth(A, B, C) &X = y.getX()

                let ref4: auth(A, B, C) &X? = y.getOptionalX()
            }
        `)

		_, err := inter.Invoke("main")
		assert.NoError(t, err)
	})
}

func NoTestInterpretMappingInclude(t *testing.T) {

	t.Parallel()

	t.Run("included identity", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement E
		entitlement F
		entitlement G 

		entitlement mapping M {
			include Identity 
		}

		struct S {
			access(mapping M) fun foo(): auth(mapping M) &Int {
				return &3
			}
		}

		fun main(): auth(E, F) &Int {
			let s = &S() as  auth(E, F) &S
			return s.foo()
		}
        `)

		value, err := inter.Invoke("main")
		assert.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{
						"S.test.E",
						"S.test.F",
					}
				},
				2,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("included identity with additional", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement E
		entitlement F
		entitlement G 

		entitlement mapping M {
			include Identity 
			F -> G
		}

		struct S {
			access(mapping M) fun foo(): auth(mapping M) &Int {
				return &3
			}
		}

		fun main(): auth(E, F, G) &Int {
			let s = &S() as  auth(E, F) &S
			return s.foo()
		}
        `)

		value, err := inter.Invoke("main")
		assert.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{
						"S.test.E",
						"S.test.F",
						"S.test.G",
					}
				},
				3,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("included non-identity", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement E
			entitlement F
			entitlement X
			entitlement Y

			entitlement mapping M {
				include N
			}

			entitlement mapping N {
				E -> F 
				X -> Y
			}

			struct S {
				access(mapping M) fun foo(): auth(mapping M) &Int {
					return &3
				}
			}

			fun main(): auth(F, Y) &Int {
				let s = &S() as  auth(E, X) &S
				return s.foo()
			}
        `)

		value, err := inter.Invoke("main")
		assert.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{
						"S.test.F",
						"S.test.Y",
					}
				},
				2,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("overlapping includes", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement E
			entitlement F
			entitlement X
			entitlement Y

			entitlement mapping A {
				E -> F
				F -> X
				X -> Y
			}

			entitlement mapping B {
				X -> Y
			}

			entitlement mapping M {
				include A
				include B
				F -> X
			}

			struct S {
				access(mapping M) fun foo(): auth(mapping M) &Int {
					return &3
				}
			}

			fun main(): auth(F, X, Y) &Int {
				let s = &S() as  auth(E, X, F) &S
				return s.foo()
			}
        `)

		value, err := inter.Invoke("main")
		assert.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{
						"S.test.F",
						"S.test.X",
						"S.test.Y",
					}
				},
				3,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("diamond include", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement E
			entitlement F
			entitlement X
			entitlement Y

			entitlement mapping M {
				include B
				include C
			}

			entitlement mapping C {
				include A
				X -> Y
			}

			entitlement mapping B {
				F -> X
				include A
			}

			entitlement mapping A {
				E -> F
			}

			struct S {
				access(mapping M) fun foo(): auth(mapping M) &Int {
					return &3
				}
			}

			fun main(): auth(F, X, Y) &Int {
				let s = &S() as  auth(E, X, F) &S
				return s.foo()
			}
        `)

		value, err := inter.Invoke("main")
		assert.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{
						"S.test.F",
						"S.test.X",
						"S.test.Y",
					}
				},
				3,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})
}

func TestInterpretEntitlementMappingComplexFields(t *testing.T) {
	t.Parallel()

	t.Run("array field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement Inner1
			entitlement Inner2
			entitlement Outer1
			entitlement Outer2

			entitlement mapping MyMap {
				Outer1 -> Inner1
				Outer2 -> Inner2
			}
			struct InnerObj {
				access(Inner1) fun first(): Int{ return 9999 }
				access(Inner2) fun second(): Int{ return 8888 }
			}

			struct Carrier{
				access(mapping MyMap) let arr: [auth(mapping MyMap) &InnerObj]
				init() {
					self.arr = [&InnerObj()]
				}
			}    

			fun test(): Int {
				let x = Carrier().arr[0]
				return x.first() + x.second()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(9999+8888),
			value,
		)
	})

	t.Run("dictionary field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement Inner1
			entitlement Inner2
			entitlement Outer1
			entitlement Outer2

			entitlement mapping MyMap {
				Outer1 -> Inner1
				Outer2 -> Inner2
			}
			struct InnerObj {
				access(Inner1) fun first(): Int{ return 9999 }
				access(Inner2) fun second(): Int{ return 8888 }
			}

			struct Carrier{
				access(mapping MyMap) let dict: {String: auth(mapping MyMap) &InnerObj}
				init() {
                    self.dict = {"": &InnerObj()}
                }
			}    

			fun test(): Int {
				let x = Carrier().dict[""]!
				return x.first() + x.second()
			}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(9999+8888),
			value,
		)
	})

	t.Run("lambda array field", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		entitlement Inner1
		entitlement Inner2
		entitlement Outer1
		entitlement Outer2

		entitlement mapping MyMap {
			Outer1 -> Inner1
			Outer2 -> Inner2
		}
		struct InnerObj {
			access(Inner1) fun first(): Int{ return 9999 }
			access(Inner2) fun second(): Int{ return 8888 }
		}

		struct Carrier{
			access(mapping MyMap) let fnArr: [fun(auth(mapping MyMap) &InnerObj): auth(mapping MyMap) &InnerObj]
			init() {
				let innerObj = &InnerObj() as auth(Inner1, Inner2) &InnerObj
				self.fnArr = [fun(_ x: &InnerObj): auth(Inner1, Inner2) &InnerObj {
					return innerObj
				}]
			}
		 
		}    

		fun test(): Int {
			let carrier = Carrier()
			let ref1 = &carrier as auth(Outer1) &Carrier
			let ref2 = &carrier as auth(Outer2) &Carrier
			return ref1.fnArr[0](&InnerObj() as auth(Inner1) &InnerObj).first() + 
			ref2.fnArr[0](&InnerObj() as auth(Inner2) &InnerObj).second() 
		}
		`)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(9999+8888),
			value,
		)
	})
}
