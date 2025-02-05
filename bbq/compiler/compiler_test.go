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

package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/constantkind"
	"github.com/onflow/cadence/bbq/opcode"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCompileRecursionFib(t *testing.T) {

	t.Parallel()

	t.SkipNow()

	checker, err := ParseAndCheck(t, `
      fun fib(_ n: Int): Int {
          if n < 2 {
             return n
          }
          return fib(n - 1) + fib(n - 2)
      }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	assert.Equal(t,
		[]opcode.Instruction{
			// if n < 2
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionIntLess{},
			opcode.InstructionJumpIfFalse{Target: 6},
			// then return n
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionReturnValue{},
			// fib(n - 1)
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionIntSubtract{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionGetGlobal{GlobalIndex: 0x0},
			opcode.InstructionInvoke{},
			// fib(n - 2)
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionIntSubtract{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionGetGlobal{GlobalIndex: 0x0},
			opcode.InstructionInvoke{},
			opcode.InstructionIntAdd{},
			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},
			// return $result
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionReturnValue{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
		[]*bbq.Constant{
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

func TestCompileImperativeFib(t *testing.T) {

	t.Parallel()

	t.SkipNow()

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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	assert.Equal(t,
		[]opcode.Instruction{
			// var fib1 = 1
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},
			// var fib2 = 1
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x2},
			// var fibonacci = fib1
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x3},
			// var i = 2
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x4},
			// while i < n
			opcode.InstructionGetLocal{LocalIndex: 0x4},
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionIntLess{},
			opcode.InstructionJumpIfFalse{Target: 33},
			// fibonacci = fib1 + fib2
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionGetLocal{LocalIndex: 0x2},
			opcode.InstructionIntAdd{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x3},
			// fib1 = fib2
			opcode.InstructionGetLocal{LocalIndex: 0x2},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},
			// fib2 = fibonacci
			opcode.InstructionGetLocal{LocalIndex: 0x3},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x2},
			// i = i + 1
			opcode.InstructionGetLocal{LocalIndex: 0x4},
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionIntAdd{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x4},
			// continue loop
			opcode.InstructionJump{Target: 12},
			// assign to temp $result
			opcode.InstructionGetLocal{LocalIndex: 0x3},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x5},
			// return $result
			opcode.InstructionGetLocal{LocalIndex: 0x5},
			opcode.InstructionReturnValue{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
		[]*bbq.Constant{
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

func TestCompileBreak(t *testing.T) {

	t.Parallel()

	t.SkipNow()

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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x0},
			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 16},
			// if i > 3
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionIntGreater{},
			opcode.InstructionJumpIfFalse{Target: 10},
			// break
			opcode.InstructionJump{Target: 16},
			// i = i + 1
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x2},
			opcode.InstructionIntAdd{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x0},
			// repeat
			opcode.InstructionJump{Target: 3},
			// assign i to temp $result
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},
			// return $result
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionReturnValue{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
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

	t.SkipNow()

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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)
	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x0},
			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 17},
			// i = i + 1
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionIntAdd{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x0},
			// if i < 3
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x2},
			opcode.InstructionIntLess{},
			opcode.InstructionJumpIfFalse{Target: 15},
			// continue
			opcode.InstructionJump{Target: 3},
			// break
			opcode.InstructionJump{Target: 17},
			// repeat
			opcode.InstructionJump{Target: 3},
			// assign i to temp $result
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},
			// return $result
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionReturnValue{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
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

func TestCompileArray(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test() {
          let xs: [Int] = [1, 2, 3]
      }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// [1, 2, 3]
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionGetConstant{ConstantIndex: 0x2},
			opcode.InstructionNewArray{
				TypeIndex:  0x0,
				Size:       0x3,
				IsResource: false,
			},

			// let xs =
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x0},

			opcode.InstructionReturn{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
		[]*bbq.Constant{
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0x2},
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

func TestCompileDictionary(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test() {
          let xs: {String: Int} = {"a": 1, "b": 2, "c": 3}
      }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// {"a": 1, "b": 2, "c": 3}
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionGetConstant{ConstantIndex: 0x2},
			opcode.InstructionGetConstant{ConstantIndex: 0x3},
			opcode.InstructionGetConstant{ConstantIndex: 0x4},
			opcode.InstructionGetConstant{ConstantIndex: 0x5},
			opcode.InstructionNewDictionary{
				TypeIndex:  0x0,
				Size:       0x3,
				IsResource: false,
			},
			// let xs =
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x0},

			opcode.InstructionReturn{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
		[]*bbq.Constant{
			{
				Data: []byte{'a'},
				Kind: constantkind.String,
			},
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{'b'},
				Kind: constantkind.String,
			},
			{
				Data: []byte{0x2},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{'c'},
				Kind: constantkind.String,
			},
			{
				Data: []byte{0x3},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileIfLet(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(x: Int?): Int {
          if let y = x {
             return y
          } else {
             return 2
          }
      }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// let y = x
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},

			// if
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionJumpIfNil{Target: 11},

			// let y = x
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionUnwrap{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x2},

			// then { return y }
			opcode.InstructionGetLocal{LocalIndex: 0x2},
			opcode.InstructionReturnValue{},
			opcode.InstructionJump{Target: 13},

			// else { return 2 }
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionReturnValue{},

			opcode.InstructionReturn{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
		[]*bbq.Constant{
			{
				Data: []byte{0x2},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileSwitch(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(x: Int): Int {
          var a = 0
          switch x {
              case 1:
                  a = 1
              case 2:
                  a = 2
              default:
                  a = 3
          }
          return a
      }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// var a = 0
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},

			// switch x
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x2},

			// case 1:
			opcode.InstructionGetLocal{LocalIndex: 0x2},
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 13},

			// a = 1
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},

			// jump to end
			opcode.InstructionJump{Target: 24},

			// case 2:
			opcode.InstructionGetLocal{LocalIndex: 0x2},
			opcode.InstructionGetConstant{ConstantIndex: 0x2},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 21},

			// a = 2
			opcode.InstructionGetConstant{ConstantIndex: 0x2},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},

			// jump to end
			opcode.InstructionJump{Target: 24},

			// default:
			// a = 3
			opcode.InstructionGetConstant{ConstantIndex: 0x3},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},

			// return a
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x3},
			opcode.InstructionGetLocal{LocalIndex: 0x3},
			opcode.InstructionReturnValue{},
		},
		compiler.ExportFunctions()[0].Code,
	)

	assert.Equal(t,
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
				Data: []byte{0x2},
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
