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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"

	. "github.com/onflow/cadence/test_utils/interpreter_utils"
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

		inter, _ := testAccount(t, address, true, nil,
			`
              entitlement X
              entitlement Y

              resource R {}

              fun test(): auth(X) &R {
                  let r <- create R()
                  account.storage.save(<-r, to: /storage/foo)
                  return account.storage.borrow<auth(X) &R>(from: /storage/foo)!
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
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.StorageReferenceValue).Authorization),
		)
	})

	t.Run("upcasting and downcasting", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            access(all) fun test(): Bool {
                let ref = &1 as auth(X) &Int
                let anyStruct = ref as AnyStruct
                let downRef = (anyStruct as? &Int)!
                let downDownRef = downRef as? auth(X) &Int
                return downDownRef == nil
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

		inter := parseCheckAndInterpret(t, `
            entitlement X
            entitlement Y

            fun test(capXY: Capability<auth(X, Y) &Int>): Bool {
                let upCap = capXY as Capability<auth(X) &Int>
                return upCap as? Capability<auth(X, Y) &Int> == nil
            }
        `)

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

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(capX: Capability<auth(X) &Int>): Capability {
                let upCap = capX as Capability
                return (upCap as? Capability<auth(X) &Int>)!
            }
        `)

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

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let arr: auth(X) &Int = &1
                let upArr = arr as &Int
                return upArr as? auth(X) &Int == nil
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

	t.Run("optional ref downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let one: Int? = 1
                let arr: auth(X) &Int? = &one
                let upArr = arr as &Int?
                return upArr as? auth(X) &Int? == nil
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

	t.Run("ref array downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let arr: [auth(X) &Int] = [&1, &2]
                let upArr = arr as [&Int]
                return upArr as? [auth(X) &Int] == nil
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

	t.Run("ref constant array downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let arr: [auth(X) &Int; 2] = [&1, &2]
                let upArr = arr as [&Int; 2]
                return upArr as? [auth(X) &Int; 2] == nil
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

	t.Run("ref constant array downcast no change", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let arr: [auth(X) &Int; 2] = [&1, &2]
                let upArr = arr as [auth(X) &Int; 2]
                return upArr as? [auth(X) &Int; 2] == nil
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

	t.Run("ref array element downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let arr: [auth(X) &Int] = [&1, &2]
                let upArr = arr as [&Int]
                return upArr[0] as? auth(X) &Int == nil
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

	t.Run("ref constant array element downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let arr: [auth(X) &Int; 2] = [&1, &2]
                let upArr = arr as [&Int; 2]
                return upArr[0] as? auth(X) &Int == nil
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

	t.Run("ref dict downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let dict: {String: auth(X) &Int} = {"foo": &3}
                let upDict = dict as {String: &Int}
                return upDict as? {String: auth(X) &Int} == nil
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

	t.Run("ref dict element downcast forced", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X

            fun test(): Bool {
                let dict: {String: auth(X) &Int} = {"foo": &3}
                let upDict = dict as {String: &Int}
                return upDict["foo"]! as? auth(X) &Int == nil
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
		RequireError(t, err)

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
              access(mapping M) let foo: [Int]

              init() {
                  self.foo = []
              }
          }

          fun test(): auth(Y) &[Int] {
              let s = S()
              let ref = &s as auth(X) &S
              let i = ref.foo
              return i
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
		refValue := value.(*interpreter.EphemeralReferenceValue)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{"S.test.Y"}
				},
				1,
				sema.Conjunction,
			).Equal(refValue.Authorization),
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
              access(mapping M) let foo: [Int]

              init() {
                  self.foo = []
              }
          }

          fun test(): auth(Y) &[Int] {
              let s = S()
              let ref = &s as auth(X, E) &S
              let upref = ref as auth(X) &S
              let i = upref.foo
              return i
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
		refValue := value.(*interpreter.EphemeralReferenceValue)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{"S.test.Y"}
				},
				1,
				sema.Conjunction,
			).Equal(refValue.Authorization),
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
              access(mapping M) let foo: [Int]

              init() {
                  self.foo = []
              }
          }

          fun test(): auth(Y) &[Int] {
              let s = S()
              let ref = &s as auth(X) &S
              let i = ref.foo
              return i
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
		refValue := value.(*interpreter.EphemeralReferenceValue)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{"S.test.Y"}
				},
				1,
				sema.Conjunction,
			).Equal(refValue.Authorization),
		)
	})

	t.Run("owned access", func(t *testing.T) {

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
              access(mapping M) let foo: [Int]
              init() {
                  self.foo = []
              }
          }

          fun test(): [Int] {
              let s = S()
              let i = s.foo
              return i
          }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.ArrayValue{}, value)
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
            access(mapping M) let foo: [Int]

            init() {
                self.foo = []
            }
        }

        fun test(): auth(Y) &[Int] {
            let s: S? = S()
            let ref = &s as auth(X) &S?
            let i = ref?.foo
            return i!
        }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
		refValue := value.(*interpreter.EphemeralReferenceValue)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{"S.test.Y"}
				},
				1,
				sema.Conjunction,
			).Equal(refValue.Authorization),
		)
	})

	t.Run("storage reference value", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil,
			`
              entitlement X
              entitlement Y
              entitlement E
              entitlement F

              entitlement mapping M {
                  X -> Y
                  E -> F
              }

              struct S {
                  access(mapping M) let foo: [Int]

                  init() {
                      self.foo = []
                  }
              }

              fun test(): auth(Y) &[Int] {
                  let s = S()
                  account.storage.save(s, to: /storage/foo)
                  let ref = account.storage.borrow<auth(X) &S>(from: /storage/foo)
                  let i = ref?.foo
                  return i!
              }
            `,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, value)
		refValue := value.(*interpreter.EphemeralReferenceValue)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"S.test.Y"} },
				1,
				sema.Conjunction,
			).Equal(refValue.Authorization),
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
            entitlement Y
            entitlement Z

            struct S {
                access(Y | Z) fun foo() {}
            }

            access(all) attachment A for S {
                access(Y | Z) fun entitled(): auth(Y | Z) &S {
                    return base
                }
            }

            fun test(): auth(Y | Z) &S {
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

	t.Run("basic call unbound method", func(t *testing.T) {

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
                let foo = s[A]!.entitled
                return foo()
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

	t.Run("basic call unbound method base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement Y
            entitlement Z

            struct S {
                access(Y | Z) fun foo() {}
            }

            access(all) attachment A for S {
                access(Y | Z) fun entitled(): auth(Y | Z) &S {
                    return base
                }
            }

            fun test(): auth(Y | Z) &S {
                let s = attach A() to S()
                let foo = s[A]!.entitled
                return foo()
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

	t.Run("basic ref access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X
            entitlement E
            entitlement G

            struct S {
                access(X, E, G) fun foo() {}
            }

            access(all) attachment A for S {}

            fun test(): auth(E) &A {
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
				func() []common.TypeID { return []common.TypeID{"S.test.E"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic ref call", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X
            entitlement E
            entitlement G

            struct S {
                access(X, E, G) fun foo() {}
            }

            access(all) attachment A for S {
                access(E | G) fun entitled(): auth(E | G) &A {
                    return self
                }
            }

            fun test(): auth(E | G) &A {
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
				func() []common.TypeID { return []common.TypeID{"S.test.E", "S.test.G"} },
				2,
				sema.Disjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic ref call conjunction", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X
            entitlement E
            entitlement G

            struct S {
                access(X, E, G) fun foo() {}
            }

            access(all) attachment A for S {
                access(E) fun entitled(): auth(E) &A {
                    return self
                }
            }

            fun test(): auth(E) &A {
                let s = attach A() to S()
                let ref = &s as auth(E, G) &S
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

	t.Run("basic ref call return base", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X
            entitlement E
            entitlement G

            struct S {
                access(X, E, G) fun foo() {}
            }

            access(all) attachment A for S {
                access(X | E) fun entitled(): auth(X | E) &S {
                    return base
                }
            }

            fun test(): auth(X | E) &S {
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
				func() []common.TypeID { return []common.TypeID{"S.test.E", "S.test.X"} },
				2,
				sema.Disjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("basic intersection access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X
            entitlement E
            entitlement G

            struct S: I {
                access(X, E, G) fun foo() {}
            }

            struct interface I {
                access(X, E, G) fun foo()
            }

            access(all) attachment A for I {}

            fun test(): auth(G) &A {
                let s = attach A() to S()
                let ref = &s as auth(G) &{I}
                return ref[A]!
            }
        `)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.True(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID { return []common.TypeID{"S.test.G"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref access", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil,
			`
              entitlement X
              entitlement E
              entitlement G

              resource R {
                  access(X, E, G) fun foo() {}
              }

              access(all) attachment A for R {}

              fun test(): auth(E) &A {
                  let r <- attach A() to <-create R()
                  account.storage.save(<-r, to: /storage/foo)
                  let ref = account.storage.borrow<auth(E) &R>(from: /storage/foo)!
                  return ref[A]!
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
				func() []common.TypeID { return []common.TypeID{"S.test.E"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref call", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil,
			`
              entitlement X
              entitlement E
              entitlement G

              resource R {
                  access(X, E, G) fun foo() {}
              }

              access(all) attachment A for R {
                  access(X, E) fun entitled(): auth(X, E) &A {
                      return self
                  }
              }

              fun test(): auth(X, E) &A {
                  let r <- attach A() to <-create R()
                  account.storage.save(<-r, to: /storage/foo)
                  let ref = account.storage.borrow<auth(X, E, G) &R>(from: /storage/foo)!
                  return ref[A]!.entitled()
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
				func() []common.TypeID { return []common.TypeID{"S.test.X", "S.test.E"} },
				2,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("storage ref call base", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, _ := testAccount(t, address, true, nil,
			`
              entitlement X
              entitlement E
              entitlement G

              resource R {
                  access(X, E, G) fun foo() {}
              }

              access(all) attachment A for R {
                  access(X) fun entitled(): auth(X) &R {
                      return base
                  }
              }

              fun test(): auth(X) &R {
                  let r <- attach A() to <-create R()
                  account.storage.save(<-r, to: /storage/foo)
                  let ref = account.storage.borrow<auth(E, X, G) &R>(from: /storage/foo)!
                  return ref[A]!.entitled()
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
				func() []common.TypeID { return []common.TypeID{"S.test.X"} },
				1,
				sema.Conjunction,
			).Equal(value.(*interpreter.EphemeralReferenceValue).Authorization),
		)
	})

	t.Run("fully entitled in init", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement X
            entitlement E
            entitlement G

            resource R {
                access(X, E, G) fun foo() {}
            }

            access(all) attachment A for R {

                init() {
                    let x = self as! auth(X, E, G) &A
                    let y = base as! auth(X, E, G) &R
                }
            }

            fun test() {
                let r <- attach A() to <-create R()
                destroy r
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("composed attachment access", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            entitlement Z
            entitlement Y

            struct S {
                access(Y) fun foo() {}
            }

            struct T {
                access(Z) fun foo() {}
            }

            access(all) attachment A for S {

                access(self) let t: T

                init(t: T) {
                    self.t = t
                }

                access(Y) fun getT(): auth(Z) &T {
                    return &self.t as auth(Z) &T
                }
            }

            access(all) attachment B for T {}

            fun test(): auth(Z) &B {
                let s = attach A(t: attach B() to T()) to S()
                let ref = &s as auth(Y) &S
                return ref[A]!.getT()[B]!
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
