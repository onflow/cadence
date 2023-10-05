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

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretGenericFunctionDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("global", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          fun head<T: AnyStruct>(_ items: [T]): T? {
              if items.length < 1 {
                  return nil
              }
              return items[0]
          }

          fun test(): Int {
              return head([1, 2, 3])!
          }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		utils.RequireValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(nil, 1),
			result,
		)
	})

	t.Run("composite", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {
              fun head<T: AnyStruct>(_ items: [T]): T? {
                  if items.length < 1 {
                      return nil
                  }
                  return items[0]
              }
          }

          fun test(): Int {
              return  S().head([1, 2, 3])!
          }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)

		utils.RequireValuesEqual(
			t,
			inter,
			interpreter.NewIntValueFromInt64(nil, 1),
			result,
		)
	})
}
