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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/checker"

	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/compiler"
	"github.com/onflow/cadence/runtime/bbq/vm/context"
	"github.com/onflow/cadence/runtime/bbq/vm/values"
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
		values.IntValue{SmallInt: 7},
	)
	require.NoError(t, err)
	require.Equal(t, values.IntValue{SmallInt: 13}, result)
}

func BenchmarkRecursionFib(b *testing.B) {

	checker, err := ParseAndCheck(b, recursiveFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	b.ReportAllocs()
	b.ResetTimer()

	expected := values.IntValue{SmallInt: 377}

	for i := 0; i < b.N; i++ {

		result, err := vm.Invoke(
			"fib",
			values.IntValue{SmallInt: 14},
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
		values.IntValue{SmallInt: 7},
	)
	require.NoError(t, err)
	require.Equal(t, values.IntValue{SmallInt: 13}, result)
}

func BenchmarkImperativeFib(b *testing.B) {

	checker, err := ParseAndCheck(b, imperativeFib)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	b.ReportAllocs()
	b.ResetTimer()

	var value values.Value = values.IntValue{SmallInt: 14}

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

	require.Equal(t, values.IntValue{SmallInt: 4}, result)
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

	require.Equal(t, values.IntValue{SmallInt: 3}, result)
}

func TestNewStruct(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct Foo {
          var id : Int

          init(_ id: Int) {
              self.id = id
          }
      }

      fun test(count: Int): Foo {
          var i = 0
          var r = Foo(0)
          while i < count {
              i = i + 1
              r = Foo(i)
              r.id = r.id + 2
          }
          return r
      }
  `)
	require.NoError(t, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	printProgram(program)

	vm := NewVM(program)

	result, err := vm.Invoke("test", values.IntValue{SmallInt: 10})
	require.NoError(t, err)

	require.IsType(t, &values.CompositeValue{}, result)
	structValue := result.(*values.CompositeValue)

	require.Equal(t, "Foo", structValue.QualifiedIdentifier)
	require.Equal(
		t,
		values.IntValue{SmallInt: 12},
		structValue.GetMember(vm.context, "id"),
	)
}

func BenchmarkNewStruct(b *testing.B) {

	checker, err := ParseAndCheck(b, `
      resource Foo {
          var id : Int

          init(_ id: Int) {
              self.id = id
          }
      }

      fun test(count: Int): @Foo {
          var i = 0
          var r <- create Foo(0)
          while i < count {
              i = i + 1
              destroy create Foo(i)
          }
          return <- r
      }
  `)
	require.NoError(b, err)

	comp := compiler.NewCompiler(checker.Program, checker.Elaboration)
	program := comp.Compile()

	vm := NewVM(program)

	b.ReportAllocs()
	b.ResetTimer()

	value := values.IntValue{SmallInt: 7}

	for i := 0; i < b.N; i++ {
		_, err := vm.Invoke("test", value)
		require.NoError(b, err)
	}
}

func BenchmarkNewStructRaw(b *testing.B) {

	storage := interpreter.NewInMemoryStorage(nil)
	ctx := &context.Context{
		Storage: storage,
	}

	fieldValue := values.IntValue{SmallInt: 7}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 8; j++ {
			structValue := values.NewCompositeValue(
				nil,
				"Foo",
				common.CompositeKindStructure,
				common.Address{},
				storage.BasicSlabStorage,
			)
			structValue.SetMember(ctx, "id", fieldValue)
		}
	}
}

func printProgram(program *bbq.Program) {
	byteCodePrinter := &bbq.BytecodePrinter{}
	fmt.Println(byteCodePrinter.PrintProgram(program))
}
