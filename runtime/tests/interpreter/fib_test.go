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

const imperativeFib = `
  fun fib(_ n: Int): Int {
      var fib1 = 1
      var fib2 = 1
      var fibonacci = fib1
      var i = 2
      while i < n {
          fibonacci = fib1 + fib2
          fib1 = fib2
          fib2 = fibonacci
          i = i + 1
      }
      return fibonacci
  }
`

func TestImperativeFib(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, imperativeFib)

	var value interpreter.Value = interpreter.NewUnmeteredIntValueFromInt64(7)

	result, err := inter.Invoke("fib", value)
	require.NoError(t, err)
	require.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(13), result)
}

func BenchmarkImperativeFib(b *testing.B) {

	inter := parseCheckAndInterpret(b, imperativeFib)

	var value interpreter.Value = interpreter.NewUnmeteredIntValueFromInt64(14)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := inter.Invoke("fib", value)
		require.NoError(b, err)
	}
}
