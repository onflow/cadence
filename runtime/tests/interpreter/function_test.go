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
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretResultVariable(t *testing.T) {

	t.Parallel()

	t.Run("resource type, resource value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource R {
                pub let id: UInt64
                init() {
                    self.id = 1
                }
            }

            pub fun main(): @R  {
                post {
                    result.id == 1: "invalid id"
                }
                return <- create R()
            }`,
		)

		result, err := inter.Invoke("main")
		require.NoError(t, err)

		require.IsType(t, &interpreter.CompositeValue{}, result)
		resource := result.(*interpreter.CompositeValue)
		assert.Equal(t, common.CompositeKindResource, resource.Kind)
		utils.AssertValuesEqual(
			t,
			inter,
			interpreter.UInt64Value(1),
			resource.GetField(inter, interpreter.EmptyLocationRange, "id"),
		)
	})

	t.Run("optional resource type, resource value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource R {
                pub let id: UInt64
                init() {
                    self.id = 1
                }
            }

            pub fun main(): @R?  {
                post {
                    result!.id == 1: "invalid id"
                }
                return <- create R()
            }`,
		)

		result, err := inter.Invoke("main")
		require.NoError(t, err)

		require.IsType(t, &interpreter.SomeValue{}, result)
		someValue := result.(*interpreter.SomeValue)

		innerValue := someValue.InnerValue(inter, interpreter.EmptyLocationRange)
		require.IsType(t, &interpreter.CompositeValue{}, innerValue)

		resource := innerValue.(*interpreter.CompositeValue)
		assert.Equal(t, common.CompositeKindResource, resource.Kind)
		utils.AssertValuesEqual(
			t,
			inter,
			interpreter.UInt64Value(1),
			resource.GetField(inter, interpreter.EmptyLocationRange, "id"),
		)
	})

	t.Run("optional resource type, nil value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource R {
                pub let id: UInt64
                init() {
                    self.id = 1
                }
            }

            pub fun main(): @R?  {
                post {
                    result == nil: "invalid result"
                }
                return nil
            }`,
		)

		result, err := inter.Invoke("main")
		require.NoError(t, err)
		require.Equal(t, interpreter.NilValue{}, result)
	})

	t.Run("any resource type, optional value", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource R {
                pub let id: UInt64
                init() {
                    self.id = 1
                }
            }

            pub fun main(): @AnyResource  {
                post {
                    result != nil: "invalid value"
                }

                var value: @R? <- create R()
                return <- value
            }`,
		)

		result, err := inter.Invoke("main")
		require.NoError(t, err)

		require.IsType(t, &interpreter.SomeValue{}, result)
		someValue := result.(*interpreter.SomeValue)

		innerValue := someValue.InnerValue(inter, interpreter.EmptyLocationRange)
		require.IsType(t, &interpreter.CompositeValue{}, innerValue)

		resource := innerValue.(*interpreter.CompositeValue)
		assert.Equal(t, common.CompositeKindResource, resource.Kind)
		utils.AssertValuesEqual(
			t,
			inter,
			interpreter.UInt64Value(1),
			resource.GetField(inter, interpreter.EmptyLocationRange, "id"),
		)
	})

	t.Run("reference invalidation, optional type", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource R {
                pub(set) var id: UInt64
                init() {
                    self.id = 1
                }
            }

            var ref: &R? = nil

            pub fun main(): @R? {
                var r <- createAndStoreRef()
                if var r <- r {
                    r.id = 2
                    return <- r
                }

                return <- r
            }

            pub fun createAndStoreRef(): @R? {
                post {
                    storeRef(result)
                }
                return <- create R()
            }

            pub fun storeRef(_ r: &R?): Bool {
                ref = r
                return r != nil
            }

            pub fun getRef(): &R {
                return ref!
            }`,
		)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		result, err := inter.Invoke("getRef")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, result)
		referenceValue := result.(*interpreter.EphemeralReferenceValue)

		// Get the field via the reference. Must have the updated field value.
		utils.AssertValuesEqual(
			t,
			inter,
			interpreter.UInt64Value(2),
			referenceValue.GetMember(inter, interpreter.EmptyLocationRange, "id"),
		)
	})

	t.Run("reference invalidation, non optional", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            pub resource R {
                pub(set) var id: UInt64
                init() {
                    self.id = 1
                }
            }

            var ref: &R? = nil

            pub fun main(): @R {
                var r <- createAndStoreRef()
                r.id = 2
                return <- r
            }

            pub fun createAndStoreRef(): @R {
                post {
                    storeRef(result)
                }
                return <- create R()
            }

            pub fun storeRef(_ r: &R): Bool {
                ref = r
                return r != nil
            }

            pub fun getRef(): &R {
                return ref!
            }`,
		)

		_, err := inter.Invoke("main")
		require.NoError(t, err)

		result, err := inter.Invoke("getRef")
		require.NoError(t, err)

		require.IsType(t, &interpreter.EphemeralReferenceValue{}, result)
		referenceValue := result.(*interpreter.EphemeralReferenceValue)

		// Get the field via the reference. Must have the updated field value.
		utils.AssertValuesEqual(
			t,
			inter,
			interpreter.UInt64Value(2),
			referenceValue.GetMember(inter, interpreter.EmptyLocationRange, "id"),
		)
	})
}
