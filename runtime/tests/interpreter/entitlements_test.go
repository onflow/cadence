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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/require"

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
}

func TestInterpretEntitledReferences(t *testing.T) {

	t.Parallel()

	t.Run("upcasting does not change static entitlements", func(t *testing.T) {

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
				return account.borrow<auth(X) &R>(from: /storage/foo)!
			}
			`,
			sema.Config{},
		)

		value, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Equal(
			t,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				[]common.TypeID{"S.test.X"},
			),
			value.(*interpreter.StorageReferenceValue).Authorization,
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
				let downRef = ref as? auth(X, Y) &Int
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

	t.Run("disjoint downcast", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			entitlement X
			entitlement Y

			fun test(): Bool {
				let ref = &1 as auth(X, Y) &Int
				let upRef = ref as auth(X | Y) &Int
				let downRef = ref as? auth(X) &Int
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
				let downRef = ref as? auth(X) &Int
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
}

func TestInterpretDisjointSetRuntimeCreation(t *testing.T) {

	t.Parallel()

	t.Run("cannot borrow with disjoint entitlement set", func(t *testing.T) {

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
				return account.borrow<auth(X | Y) &R>(from: /storage/foo)!
			}
			`,
			sema.Config{},
		)

		_, err := inter.Invoke("test")
		require.Error(t, err)
		var disjointErr interpreter.InvalidDisjointRuntimeEntitlementSetCreationError
		require.ErrorAs(t, err, &disjointErr)

	})
}
