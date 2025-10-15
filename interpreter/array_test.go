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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/interpreter"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

func TestInterpretArrayFunctionEntitlements(t *testing.T) {

	t.Parallel()

	t.Run("mutable reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar"]
                var arrayRef = &array as auth(Mutate) &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])

                // Insert functions
                arrayRef.append("baz")
                arrayRef.appendAll(["baz"])
                arrayRef.insert(at:0, "baz")

                // Remove functions
                arrayRef.remove(at: 1)
                arrayRef.removeFirst()
                arrayRef.removeLast()
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("non auth reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar"]
                var arrayRef = &array as &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])
            }
	        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("insert reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar"]
                var arrayRef = &array as auth(Insert) &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])

                // Insert functions
                arrayRef.append("baz")
                arrayRef.appendAll(["baz"])
                arrayRef.insert(at:0, "baz")
            }
	        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("remove reference", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test() {
                let array: [String] = ["foo", "bar", "baz"]
                var arrayRef = &array as auth(Remove) &[String]

                // Public functions
                arrayRef.contains("hello")
                arrayRef.firstIndex(of: "hello")
                arrayRef.slice(from: 1, upTo: 1)
                arrayRef.concat(["hello"])

                // Remove functions
                arrayRef.remove(at: 1)
                arrayRef.removeFirst()
                arrayRef.removeLast()
            }
	        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})
}

func TestCheckArrayReferenceTypeInferenceWithDowncasting(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
		entitlement E
		entitlement F 
		entitlement G

		fun test() {
			let ef = &1 as auth(E, F) &Int
			let eg = &1 as auth(E, G) &Int
			let arr = [ef, eg]
			let ref = arr[0]
            let downcastRef = ref as! auth(E, F) &Int
		}
	
	`)

	_, err := inter.Invoke("test")
	require.Error(t, err)
	var forceCastTypeMismatchError *interpreter.ForceCastTypeMismatchError
	require.ErrorAs(t, err, &forceCastTypeMismatchError)
}

func TestInterpretArrayReduce(t *testing.T) {
	t.Parallel()

	t.Run("with variable sized array - sum", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			let xs = [1, 2, 3, 4, 5]

			let sum =
				fun (acc: Int, x: Int): Int {
					return acc + x
				}

			fun reduce(): Int {
				return xs.reduce(initial: 0, sum)
			}
		`)

		val, err := inter.Invoke("reduce")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(15),
			val,
		)
	})

	t.Run("with variable sized array - product", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			let xs = [1, 2, 3, 4]

			let product =
				fun (acc: Int, x: Int): Int {
					return acc * x
				}

			fun reduce(): Int {
				return xs.reduce(initial: 1, product)
			}
		`)

		val, err := inter.Invoke("reduce")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(24),
			val,
		)
	})

	t.Run("with empty array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			let xs: [Int] = []

			let sum =
				fun (acc: Int, x: Int): Int {
					return acc + x
				}

			fun reduce(): Int {
				return xs.reduce(initial: 42, sum)
			}
		`)

		val, err := inter.Invoke("reduce")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(42),
			val,
		)
	})

	t.Run("with fixed sized array", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			let xs: [Int; 4] = [10, 20, 30, 40]

			let sum =
				fun (acc: Int, x: Int): Int {
					return acc + x
				}

			fun reduce(): Int {
				return xs.reduce(initial: 0, sum)
			}
		`)

		val, err := inter.Invoke("reduce")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredIntValueFromInt64(100),
			val,
		)
	})

	t.Run("with type conversion", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			let xs = [1, 2, 3]

			let concat =
				fun (acc: String, x: Int): String {
					return acc.concat(x.toString())
				}

			fun reduce(): String {
				return xs.reduce(initial: "", concat)
			}
		`)

		val, err := inter.Invoke("reduce")
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("123"),
			val,
		)
	})

	t.Run("mutation during reduce", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
			let xs = [1, 2, 3]
			let xRef = &xs as auth(Mutate) &[Int]
			let sum =
				fun (acc: Int, x: Int): Int {
					xRef.remove(at: 0)
					return acc + x
				}

			fun reduce(): Int {
				return xRef.reduce(initial: 0, sum)
			}
		`)

		_, err := inter.Invoke("reduce")
		var containerMutationError *interpreter.ContainerMutatedDuringIterationError
		require.ErrorAs(t, err, &containerMutationError)
	})
}
