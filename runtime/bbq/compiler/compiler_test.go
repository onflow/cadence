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
			opcode.IntConstantLoad{0, 1},
			opcode.IntLess{0, 1, 0},
			opcode.JumpIfFalse{0, 4},
			// then return n
			opcode.ReturnValue{0},
			// fib(n - 1)
			opcode.IntConstantLoad{1, 2},
			opcode.IntSubtract{0, 2, 3},
			opcode.GlobalFuncLoad{0, 0},
			opcode.Call{0, []opcode.Argument{{registers.Int, 3}}, 4},
			// fib(n - 2)
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
			// intReg[0] is reserved for the param
			// Hence int reg indexes for local values start from 1.

			// var fib1 = 1
			opcode.IntConstantLoad{0, 1}, // load constant
			opcode.IntMove{1, 2},         // copy to local variable
			// these moves can be optimized later via a byte code optimizer.

			// var fib2 = 1
			opcode.IntConstantLoad{1, 3}, // load constant
			opcode.IntMove{3, 4},         // store to local variable
			// var fibonacci = fib1
			opcode.IntMove{2, 5}, // copy to local variable
			// var i = 2
			opcode.IntConstantLoad{2, 6}, // load constant
			opcode.IntMove{6, 7},         // store to local variable
			// while i < n
			opcode.IntLess{7, 0, 0},
			opcode.JumpIfFalse{0, 17},
			// fibonacci = fib1 + fib2
			opcode.IntAdd{2, 4, 8}, // add two numbers
			opcode.IntMove{8, 5},   // store result in local variable
			// fib1 = fib2
			opcode.IntMove{4, 2},
			// fib2 = fibonacci
			opcode.IntMove{5, 4},
			// i = i + 1
			opcode.IntConstantLoad{3, 9},
			opcode.IntAdd{7, 9, 10},
			opcode.IntMove{10, 7},
			// continue loop
			opcode.Jump{7},
			// return fibonacci
			opcode.ReturnValue{5},
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
			opcode.IntConstantLoad{0, 0},
			opcode.IntMove{0, 1},
			// while true
			opcode.True{0},
			opcode.JumpIfFalse{0, 12},
			// if i > 3
			opcode.IntConstantLoad{1, 2},
			opcode.IntGreater{1, 2, 1},
			opcode.JumpIfFalse{1, 8},
			// break
			opcode.Jump{12},
			// i = i + 1
			opcode.IntConstantLoad{2, 3},
			opcode.IntAdd{1, 3, 4},
			opcode.IntMove{4, 1},
			// repeat
			opcode.Jump{2},
			// return i
			opcode.ReturnValue{1},
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
			opcode.IntConstantLoad{0, 0},
			opcode.IntMove{0, 1},
			// while true
			opcode.True{0},
			opcode.JumpIfFalse{0, 13},
			// i = i + 1
			opcode.IntConstantLoad{1, 2},
			opcode.IntAdd{1, 2, 3},
			opcode.IntMove{3, 1},
			// if i < 3
			opcode.IntConstantLoad{2, 4},
			opcode.IntLess{1, 4, 1},
			opcode.JumpIfFalse{1, 11},
			// continue
			opcode.Jump{2},
			// break
			opcode.Jump{13},
			// repeat
			opcode.Jump{2},
			// return i
			opcode.ReturnValue{1},
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
