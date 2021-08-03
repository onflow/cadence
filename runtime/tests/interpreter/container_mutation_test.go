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
	"fmt"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArrayMutation(t *testing.T) {

	t.Parallel()

	t.Run("simple array valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [String] = ["foo", "bar"]
                names[0] = "baz"
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("simple array invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names[0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("nested array invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [[AnyStruct]] = [["foo", "bar"]] as [[String]]
                names[0][0] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array append valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.append("baz")
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array append invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.append(5)
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array appendAll invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.appendAll(["baz", 5] as [AnyStruct])
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array insert valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.insert(at: 1, "baz")
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("array insert invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                names.insert(at: 1, 4)
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("array concat invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): [AnyStruct] {
                let names: [AnyStruct] = ["foo", "bar"] as [String]
                return names.concat(["baz", 5] as [AnyStruct])
            }
        `)

		_, err := inter.Invoke("test")

		// This should not give errors, since resulting array is a new array.
		// It doesn't mutate the existing array.
		require.NoError(t, err)
	})
}

func TestDictionaryMutation(t *testing.T) {

	t.Parallel()

	t.Run("simple dictionary valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: String} = {"foo": "bar"}
                names["foo"] = "baz"
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("simple dictionary invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names["foo"] = 5
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("optional dictionary valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: String?} = {"foo": nil}
                names["foo"] = nil
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("dictionary insert valid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names.insert(key: "foo", "baz")
            }
        `)

		_, err := inter.Invoke("test")
		require.NoError(t, err)
	})

	t.Run("dictionary insert invalid", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {String: AnyStruct} = {"foo": "bar"} as {String: String}
                names.insert(key: "foo", 5)
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.StringType, mutationError.ExpectedType)
	})

	t.Run("dictionary insert invalid key", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test() {
                let names: {Path: AnyStruct} = {/public/path: "foo"} as {PublicPath: String}
                names.insert(key: /private/path, "bar")
            }
        `)

		_, err := inter.Invoke("test")
		require.Error(t, err)

		mutationError := &interpreter.ContainerMutationError{}
		require.ErrorAs(t, err, mutationError)

		assert.Equal(t, sema.PublicPathType, mutationError.ExpectedType)
	})

	t.Run("dictionary insert invalid key", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
			pub resource R {

				pub var value: Int
		
				init(_ value: Int) {
					self.value = value
				}
		
				pub fun increment() {
					self.value = self.value + 1
				}
		
				destroy() {
				}
			}
		
			pub fun createR(_ value: Int): @R {
				return <-create R(value)
			}

			pub fun foo():Int? {
				let rs: @{String:R} <- {}
			
				let existing <- rs["foo"] <- createR(4)
			
				var val: Int = rs.length
			
				destroy existing
				destroy rs
			
				return val
			}
			
			pub fun bar(): Int? {
				let rs: @[R] <- []
			
				let existing <- rs[0] <- createR(4)
			
				var val: Int = rs.length
			
				destroy existing
				destroy rs
			
				return val
			}
        `)

		value, err := inter.Invoke("foo")
		//require.Error(t, err)

		fmt.Println(err)
		fmt.Println(value)
	})
}
