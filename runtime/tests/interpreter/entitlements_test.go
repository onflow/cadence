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

	"github.com/onflow/cadence/runtime/interpreter"
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
            let r <- create R()
            let ref = &r as &R
            let isSub = ref.getType().isSubtype(of: Type<&R>())
            destroy r
            return isSub
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
            let r <- create R()
            let ref = &r as &R
            let isSub = ref.getType().isSubtype(of: Type<auth(X) &R>())
            destroy r
            return isSub
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
            let r <- create R()
            let ref = &r as auth(X) &R
            let isSub = ref.getType().isSubtype(of: Type<&R>())
            destroy r
            return isSub
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
            let r <- create R()
            let ref = &r as auth(X) &R
            let isSub = ref.getType().isSubtype(of: Type<auth(X) &R>())
            destroy r
            return isSub
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

	inter := parseCheckAndInterpret(t, `
	   entitlement X
	   resource R {}
       fun test(): &R {
	      let r <- create R()
		  let ref = &r as auth(X) &R
		  return ref 
       }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredIntValueFromInt64(10),
		value,
	)
}
