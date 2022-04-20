/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
)

// TestInterpretIndexingExpressionTransfer tests if the indexing value
// (not the value that is indexed into) is properly transferred.
// If the indexing value is used for an assignment,
// it will be transferred into the indexed value,
// and as part of it, will get removed.
// Ensure the *copy* is removed, and *not the original*.
//
func TestInterpretIndexingExpressionTransfer(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t,
		`
          enum E: UInt8 {
              case First
              case Second
              case Third
          }

          resource R {
              let e: E
              init(e: E) {
                  self.e = e
              }
          }

          fun test(): UInt8 {
              let r <- create R(e: E.Third)
              let counts: {E: UInt64} = {}
              counts[r.e] = 42
              let res = r.e.rawValue
              destroy r
              return res
          }
        `,
	)

	result, err := inter.Invoke("test")
	require.NoError(t, err)

	require.Equal(t,
		// E.Third.rawValue
		interpreter.UInt8Value(2),
		result,
	)
}
