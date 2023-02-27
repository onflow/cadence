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

package vm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/bbq/compiler"
	. "github.com/onflow/cadence/runtime/tests/checker"
)

const recursiveFib = `
  fun fib(_ n: Int): Int {
      if n < 2 {
         return n
      }
      return fib(n - 1) + fib(n - 2)
  }
`

func TestRecursionFib(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, recursiveFib)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	result, err := vm.Invoke(
		"fib",
		IntValue{7},
	)
	require.NoError(t, err)
	require.Equal(t, IntValue{13}, result)
}

func BenchmarkRecursionFib(b *testing.B) {

	checker, err := ParseAndCheck(b, recursiveFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	b.ReportAllocs()
	b.ResetTimer()

	expected := IntValue{377}

	for i := 0; i < b.N; i++ {

		result, err := vm.Invoke(
			"fib",
			IntValue{14},
		)
		require.NoError(b, err)
		require.Equal(b, expected, result)
	}
}

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

	checker, err := ParseAndCheck(t, imperativeFib)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	result, err := vm.Invoke(
		"fib",
		IntValue{7},
	)
	require.NoError(t, err)
	require.Equal(t, IntValue{13}, result)
}

func BenchmarkImperativeFib(b *testing.B) {

	checker, err := ParseAndCheck(b, imperativeFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	b.ReportAllocs()
	b.ResetTimer()

	var value Value = IntValue{14}

	for i := 0; i < b.N; i++ {
		_, err := vm.Invoke("fib", value)
		require.NoError(b, err)
	}
}

func TestBreak(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Int {
          var i = 0
          while true {
              if i > 3 {
                 break
              }
              i = i + 1
          }
          return i
      }
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	result, err := vm.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, IntValue{4}, result)
}

func TestContinue(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Int {
          var i = 0
          while true {
              i = i + 1
              if i < 3 {
                 continue
              }
              break
          }
          return i
      }
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	result, err := vm.Invoke("test")
	require.NoError(t, err)

	require.Equal(t, IntValue{3}, result)
}

func TestNewStruct(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Foo {
          init() {}
      }

      fun test(count: Int): Int {
          var i = 0
          while i < count {
              i = i + 1
              Foo()
          }
          return i
      }
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	result, err := vm.Invoke("test", IntValue{10})
	require.NoError(t, err)

	require.Equal(t, IntValue{10}, result)
}

func BenchmarkNewStruct(b *testing.B) {

	checker, err := ParseAndCheck(b, `
      struct Foo {
          init() {}
      }

      fun test(count: Int): Int {
          var i = 0
          while i < count {
              i = i + 1
              Foo()
          }
          return i
      }
  `)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	b.ReportAllocs()
	b.ResetTimer()

	var value Value = IntValue{7}

	for i := 0; i < b.N; i++ {
		_, err := vm.Invoke("test", value)
		require.NoError(b, err)
	}
}
