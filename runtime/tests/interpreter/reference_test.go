/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/stretchr/testify/require"
)

func TestInterpretResourceReferenceInstanceOf(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource R {}

        fun test(): Bool {
            let r <- create R()
            let ref = &r as &R
            let isInstance = ref.isInstance(Type<@R>())
            destroy r
            return isInstance
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(true),
		value,
	)
}

func TestInterpretResourceReferenceFieldComparison(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
        resource R {
            let n: Int
            init() {
                self.n = 1
            }
        }

        fun test(): Bool {
            let r <- create R()
            let ref = &r as &R
            let isOne = ref.n == 1
            destroy r
            return isOne
        }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.BoolValue(true),
		value,
	)
}

func TestInterpretContainerVariance(t *testing.T) {

	t.Parallel()

	t.Run("interpreted function", func(t *testing.T) {

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              pub fun getSecret(): Int {
                  return 0
              }
          }

          struct S2 {
              priv fun getSecret(): Int {
                  return 42
              }
          }

          fun test(): Int {
              let dict: {Int: &S1} = {}

              let s2 = S2()

              let dictRef = &dict as &{Int: &AnyStruct}
              dictRef[0] = &s2 as &AnyStruct

              return dict.values[0].getSecret()
          }
        `)

		_, err := inter.Invoke("test")

		var invocationReceiverTypeErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &invocationReceiverTypeErr)
	})

	t.Run("field read", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S1 {
              var value: Int

              init() {
                  self.value = 0
              }
          }

          struct S2 {
              priv var value: Int

              init() {
                  self.value = 1
              }
          }

          fun test(): Int {
              let dict: {Int: &S1} = {}

              let s2 = S2()

              let dictRef = &dict as &{Int: &AnyStruct}
              dictRef[0] = &s2 as &AnyStruct

              return dict.values[0].value
          }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		var typeMismatchErr interpreter.TypeMismatchError
		require.ErrorAs(t, err, &typeMismatchErr)
	})

	t.Run("field write", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
	     struct S1 {
	         var value: Int

	         init() {
	             self.value = 0
	         }
	     }

	     struct S2 {
	         // field is only publicly readable, not writeable
	         pub var value: Int

	         init() {
	             self.value = 0
	         }
	     }

	     fun test() {
	         let dict: {Int: &S1} = {}

	         let s2 = S2()

	         let dictRef = &dict as &{Int: &AnyStruct}
	         dictRef[0] = &s2 as &AnyStruct

	         dict.values[0].value = 1

             // NOTE: intentionally not reading,
             // the test checks writes
         }
	   `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		var typeMismatchErr interpreter.TypeMismatchError
		require.ErrorAs(t, err, &typeMismatchErr)
	})
}
