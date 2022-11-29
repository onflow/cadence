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

package compiler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/bbq"
	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/bbq/registers"
	. "github.com/onflow/cadence/runtime/tests/checker"
)

func TestCompileRecursionFib(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun fib(_ n: Int): Int {
          if n < 2 {
             return n
          }
          return fib(n - 1) + fib(n - 2)
      }
  `)
	require.NoError(t, err)

	compiler := NewCompiler(checker.Program, checker.Elaboration)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	require.Equal(t,
		[]opcode.Opcode{
			// if n < 2
			//byte(opcode.GetLocal), 0, 0,
			opcode.IntConstantLoad{0, 1},
			opcode.IntLess{0, 1, 0},
			opcode.JumpIfFalse{0, 4},
			// then return n
			//opcode.GetLocal{}, 0, 0,
			opcode.ReturnValue{0},
			// fib(n - 1)
			//opcode.GetLocal{}, 0, 0,
			opcode.IntConstantLoad{1, 2},
			opcode.IntSubtract{0, 2, 3},
			opcode.GlobalFuncLoad{0, 0},
			opcode.Call{0, []opcode.Argument{{registers.Int, 3}}, 4},
			// fib(n - 2{}
			//opcode.GetLocal{}, 0, 0,
			opcode.IntConstantLoad{2, 5},
			opcode.IntSubtract{0, 5, 6},
			opcode.GlobalFuncLoad{0, 1},
			opcode.Call{1, []opcode.Argument{{registers.Int, 6}}, 7},
			// return sum
			opcode.IntAdd{4, 7, 8},
			opcode.ReturnValue{8},
		},
		compiler.functions[0].code,
	)

	require.Equal(t,
		[]*bbq.Constant{
			{
				Data: []byte{0x2},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileImperativeFib(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
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
  `)
	require.NoError(t, err)

	compiler := NewCompiler(checker.Program, checker.Elaboration)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	require.Equal(t,
		[]opcode.Opcode{
			// var fib1 = 1
			opcode.IntConstantLoad{}, 0, 0,
			opcode.MoveInt{}, 0, 1,
			// var fib2 = 1
			opcode.IntConstantLoad{}, 0, 1,
			opcode.MoveInt{}, 0, 2,
			// var fibonacci = fib1
			opcode.MoveInt{}, 0, 3,
			// var i = 2
			opcode.IntConstantLoad{}, 0, 2,
			opcode.MoveInt{}, 0, 4,
			// while i < n
			opcode.IntLess{},
			opcode.JumpIfFalse{}, 0, 69,
			// fibonacci = fib1 + fib2
			opcode.IntAdd{},
			opcode.MoveInt{}, 0, 3,
			// fib1 = fib2
			opcode.MoveInt{}, 0, 1,
			// fib2 = fibonacci
			opcode.MoveInt{}, 0, 2,
			// i = i + 1
			opcode.IntConstantLoad{}, 0, 3,
			opcode.IntAdd{},
			opcode.MoveInt{}, 0, 4,
			// continue loop
			opcode.Jump{}, 0, 24,
			// return fibonacci
			opcode.ReturnValue{},
		},
		compiler.functions[0].code,
	)

	require.Equal(t,
		[]*bbq.Constant{
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileBreak(t *testing.T) {

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

	compiler := NewCompiler(checker.Program, checker.Elaboration)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	require.Equal(t,
		[]opcode.Opcode{
			// var i = 0
			opcode.IntConstantLoad{}, 0, 0,
			opcode.MoveInt{}, 0, 0,
			// while true
			opcode.True{},
			opcode.JumpIfFalse{}, 0, 36,
			// if i > 3
			opcode.IntConstantLoad{}, 0, 1,
			opcode.IntGreater{},
			opcode.JumpIfFalse{}, 0, 23,
			// break
			opcode.Jump{}, 0, 36,
			// i = i + 1
			opcode.IntConstantLoad{}, 0, 2,
			opcode.IntAdd{},
			opcode.MoveInt{}, 0, 0,
			// repeat
			opcode.Jump{}, 0, 6,
			// return i
			opcode.ReturnValue{},
		},
		compiler.functions[0].code,
	)

	require.Equal(t,
		[]*bbq.Constant{
			{
				Data: []byte{0x0},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileContinue(t *testing.T) {

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

	compiler := NewCompiler(checker.Program, checker.Elaboration)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	require.Equal(t,
		[]opcode.Opcode{
			// var i = 0
			opcode.IntConstantLoad{}, 0, 0,
			opcode.MoveInt{}, 0, 0,
			// while true
			opcode.True{},
			opcode.JumpIfFalse{}, 0, 39,
			// i = i + 1
			opcode.IntConstantLoad{}, 0, 1,
			opcode.IntAdd{},
			opcode.MoveInt{}, 0, 0,
			// if i < 3
			opcode.IntConstantLoad{}, 0, 2,
			opcode.IntLess{},
			opcode.JumpIfFalse{}, 0, 33,
			// continue
			opcode.Jump{}, 0, 6,
			// break
			opcode.Jump{}, 0, 39,
			// repeat
			opcode.Jump{}, 0, 6,
			// return i
			opcode.ReturnValue{},
		},
		compiler.functions[0].code,
	)

	require.Equal(t,
		[]*bbq.Constant{
			{
				Data: []byte{0x0},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}
