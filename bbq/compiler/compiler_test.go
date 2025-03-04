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

package compiler_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/constantkind"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// if n < 2
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 6},
			// then return n
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionReturnValue{},
			// fib(n - 1)
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x1},
			opcode.InstructionSubtract{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionGetGlobal{GlobalIndex: 0x0},
			opcode.InstructionInvoke{},
			// fib(n - 2)
			opcode.InstructionGetLocal{LocalIndex: 0x0},
			opcode.InstructionGetConstant{ConstantIndex: 0x0},
			opcode.InstructionSubtract{},
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionGetGlobal{GlobalIndex: 0x0},
			opcode.InstructionInvoke{},
			opcode.InstructionAdd{},
			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0x0},
			opcode.InstructionSetLocal{LocalIndex: 0x1},
			// return $result
			opcode.InstructionGetLocal{LocalIndex: 0x1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	const parameterCount = 1

	// nIndex is the index of the parameter `n`, which is the first parameter
	const nIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// fib1Index is the index of the local variable `fib1`, which is the first local variable
		fib1Index = localsOffset + iota
		// fib2Index is the index of the local variable `fib2`, which is the second local variable
		fib2Index
		// fibonacciIndex is the index of the local variable `fibonacci`, which is the third local variable
		fibonacciIndex
		// iIndex is the index of the local variable `i`, which is the fourth local variable
		iIndex
	)

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// var fib1 = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: fib1Index},

			// var fib2 = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: fib2Index},

			// var fibonacci = fib1
			opcode.InstructionGetLocal{LocalIndex: fib1Index},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: fibonacciIndex},

			// var i = 2
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// while i < n
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetLocal{LocalIndex: nIndex},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 33},

			// fibonacci = fib1 + fib2
			opcode.InstructionGetLocal{LocalIndex: fib1Index},
			opcode.InstructionGetLocal{LocalIndex: fib2Index},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: fibonacciIndex},

			// fib1 = fib2
			opcode.InstructionGetLocal{LocalIndex: fib2Index},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: fib1Index},

			// fib2 = fibonacci
			opcode.InstructionGetLocal{LocalIndex: fibonacciIndex},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: fib2Index},

			// i = i + 1
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// continue loop
			opcode.InstructionJump{Target: 12},

			// return fibonacci
			opcode.InstructionGetLocal{LocalIndex: fibonacciIndex},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	// iIndex is the index of the local variable `i`, which is the first local variable
	const iIndex = localsOffset

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 16},

			// if i > 3
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionGreater{},
			opcode.InstructionJumpIfFalse{Target: 10},

			// break
			opcode.InstructionJump{Target: 16},

			// i = i + 1
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// repeat
			opcode.InstructionJump{Target: 3},

			// return i
			opcode.InstructionGetLocal{LocalIndex: iIndex},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	// iIndex is the index of the local variable `i`, which is the first local variable
	const iIndex = localsOffset

	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 17},

			// i = i + 1
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// if i < 3
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 15},

			// continue
			opcode.InstructionJump{Target: 3},

			// break
			opcode.InstructionJump{Target: 17},

			// repeat
			opcode.InstructionJump{Target: 3},

			// return i
			opcode.InstructionGetLocal{LocalIndex: iIndex},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	// xsIndex is the index of the local variable `xs`, which is the first local variable
	const xsIndex = localsOffset

	assert.Equal(t,
		[]opcode.Instruction{
			// [1, 2, 3]
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionNewArray{
				TypeIndex:  0,
				Size:       3,
				IsResource: false,
			},

			// let xs =
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: xsIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	// xsIndex is the index of the local variable `xs`, which is the first local variable
	const xsIndex = localsOffset

	assert.Equal(t,
		[]opcode.Instruction{
			// {"a": 1, "b": 2, "c": 3}
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionGetConstant{ConstantIndex: 3},
			opcode.InstructionGetConstant{ConstantIndex: 4},
			opcode.InstructionGetConstant{ConstantIndex: 5},
			opcode.InstructionNewDictionary{
				TypeIndex:  0,
				Size:       3,
				IsResource: false,
			},
			// let xs =
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: xsIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		xIndex     = iota
		_          // result index (unused)
		tempYIndex // index for the temp var to hold the value of the expression
		yIndex
	)
	assert.Equal(t,
		[]opcode.Instruction{
			// let y' = x
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionSetLocal{LocalIndex: tempYIndex},

			// if nil
			opcode.InstructionGetLocal{LocalIndex: tempYIndex},
			opcode.InstructionJumpIfNil{Target: 11},

			// let y = y'
			opcode.InstructionGetLocal{LocalIndex: tempYIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: yIndex},

			// then { return y }
			opcode.InstructionGetLocal{LocalIndex: yIndex},
			opcode.InstructionReturnValue{},
			opcode.InstructionJump{Target: 13},

			// else { return 2 }
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionReturnValue{},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{0x2},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileIfLetScope(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(y: Int?): Int {
            let x = 1
            var z = 0
            if let x = y {
                z = x
            } else {
                z = x
            }
            return x
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const parameterCount = 1

	// yIndex is the index of the parameter `y`, which is the first parameter
	const yIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// x1Index is the index of the local variable `x` in the first block, which is the first local variable
		x1Index = localsOffset + iota
		// zIndex is the index of the local variable `z`, which is the second local variable
		zIndex
		// tempIfLetIndex is the index of the temporary variable
		tempIfLetIndex
		// x2Index is the index of the local variable `x` in the second block, which is the third local variable
		x2Index
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x1Index},

			// var z = 0
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: zIndex},

			// if let x = y
			opcode.InstructionGetLocal{LocalIndex: yIndex},
			opcode.InstructionSetLocal{LocalIndex: tempIfLetIndex},

			opcode.InstructionGetLocal{LocalIndex: tempIfLetIndex},
			opcode.InstructionJumpIfNil{Target: 18},

			// then
			opcode.InstructionGetLocal{LocalIndex: tempIfLetIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x2Index},

			// z = x
			opcode.InstructionGetLocal{LocalIndex: x2Index},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: zIndex},
			opcode.InstructionJump{Target: 21},

			// else { z = x }
			opcode.InstructionGetLocal{LocalIndex: x1Index},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: zIndex},

			// return x
			opcode.InstructionGetLocal{LocalIndex: x1Index},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0},
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

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// aIndex is the index of the local variable `a`, which is the first local variable
		aIndex = localsOffset + iota
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var a = 0
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: aIndex},

			// switch x
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionSetLocal{LocalIndex: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{LocalIndex: switchIndex},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 13},

			// a = 1
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: aIndex},

			// jump to end
			opcode.InstructionJump{Target: 24},

			// case 2:
			opcode.InstructionGetLocal{LocalIndex: switchIndex},
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 21},

			// a = 2
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: aIndex},

			// jump to end
			opcode.InstructionJump{Target: 24},

			// default:
			// a = 3
			opcode.InstructionGetConstant{ConstantIndex: 3},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: aIndex},

			// return a
			opcode.InstructionGetLocal{LocalIndex: aIndex},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

func TestSwitchBreak(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(x: Int) {
          switch x {
              case 1:
                  break
              case 2:
                  break
              default:
                  break
          }
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex = localsOffset + iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// switch x
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionSetLocal{LocalIndex: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{LocalIndex: switchIndex},
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 8},
			// break
			opcode.InstructionJump{Target: 15},
			// end of case
			opcode.InstructionJump{Target: 15},

			// case 1:
			opcode.InstructionGetLocal{LocalIndex: switchIndex},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 14},
			// break
			opcode.InstructionJump{Target: 15},
			// end of case
			opcode.InstructionJump{Target: 15},

			// default:
			// break
			opcode.InstructionJump{Target: 15},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

func TestWhileSwitchBreak(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Int {
          var x = 0
          while true {
              switch x {
                  case 1:
                      break
              }
              x = x + 1
          }
          return x
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// xIndex is the index of the local variable `x`, which is the first local variable
		xIndex = localsOffset + iota
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var x = 0
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: xIndex},

			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 19},

			// switch x
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionSetLocal{LocalIndex: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{LocalIndex: switchIndex},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 13},

			// break
			opcode.InstructionJump{Target: 13},
			// end of case
			opcode.InstructionJump{Target: 13},

			// x = x + 1
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: xIndex},

			// repeat
			opcode.InstructionJump{Target: 3},

			// assign to temp $result
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{0x0},
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

func TestCompileEmit(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      event Inc(val: Int)

      fun test(x: Int) {
          emit Inc(val: x)
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	var testFunction bbq.Function[opcode.Instruction]
	for _, f := range functions {
		if f.Name == "test" {
			testFunction = f
		}
	}
	require.NotNil(t, testFunction.Code)

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// Inc(val: x)
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionGetGlobal{GlobalIndex: 1},
			opcode.InstructionInvoke{},
			// emit
			opcode.InstructionEmitEvent{TypeIndex: 1},

			opcode.InstructionReturn{},
		},
		testFunction.Code,
	)

	assert.Empty(t, program.Constants)
}

func TestCompileSimpleCast(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(x: Int): AnyStruct {
           return x as Int?
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			// x as Int?
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionSimpleCast{TypeIndex: 0},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 1},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileForceCast(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(x: AnyStruct): Int {
            return x as! Int
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			// x as! Int
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionForceCast{TypeIndex: 0},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileFailableCast(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(x: AnyStruct): Int? {
            return x as? Int
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			// x as? Int
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionFailableCast{TypeIndex: 0},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 1},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileNestedLoop(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Int {
          var i = 0
          while i < 10 {
              var j = 0
              while j < 10 {
                  if i == j {
                      break
                  }
                  j = j + 1
                  continue
              }
              i = i + 1
              continue
          }
          return i
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// iIndex is the index of the local variable `i`, which is the first local variable
		iIndex = localsOffset + iota
		// jIndex is the index of the local variable `j`, which is the second local variable
		jIndex
	)
	const (
		// zeroIndex is the index of the constant `0`, which is the first constant
		zeroIndex = iota
		// tenIndex is the index of the constant `10`, which is the second constant
		tenIndex
		// oneIndex is the index of the constant `1`, which is the third constant
		oneIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionGetConstant{ConstantIndex: zeroIndex},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// i < 10
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetConstant{ConstantIndex: tenIndex},
			opcode.InstructionLess{},

			opcode.InstructionJumpIfFalse{Target: 33},

			// var j = 0
			opcode.InstructionGetConstant{ConstantIndex: zeroIndex},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: jIndex},

			// j < 10
			opcode.InstructionGetLocal{LocalIndex: jIndex},
			opcode.InstructionGetConstant{ConstantIndex: tenIndex},
			opcode.InstructionLess{},

			opcode.InstructionJumpIfFalse{Target: 26},

			// i == j
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetLocal{LocalIndex: jIndex},
			opcode.InstructionEqual{},

			opcode.InstructionJumpIfFalse{Target: 19},

			// break
			opcode.InstructionJump{Target: 26},

			// j = j + 1
			opcode.InstructionGetLocal{LocalIndex: jIndex},
			opcode.InstructionGetConstant{ConstantIndex: oneIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: jIndex},

			// continue
			opcode.InstructionJump{Target: 10},

			// repeat
			opcode.InstructionJump{Target: 10},

			// i = i + 1
			opcode.InstructionGetLocal{LocalIndex: iIndex},
			opcode.InstructionGetConstant{ConstantIndex: oneIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: iIndex},

			// continue
			opcode.InstructionJump{Target: 3},

			// repeat
			opcode.InstructionJump{Target: 3},

			// return i
			opcode.InstructionGetLocal{LocalIndex: iIndex},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{0x0},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{0xa},
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

func TestCompileAssignLocal(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            var x = 0
            x = 1
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	// xIndex is the index of the local variable `x`, which is the first local variable
	const xIndex = localsOffset

	assert.Equal(t,
		[]opcode.Instruction{
			// var x = 0
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: xIndex},

			// x = 1
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: xIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{0x0},
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

func TestCompileAssignGlobal(t *testing.T) {

	t.Parallel()

	// TODO: compile global variables

	checker, err := ParseAndCheck(t, `
        var x = 0

        fun test() {
            x = 1
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// x = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetGlobal{GlobalIndex: 0},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{0x1},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileIndex(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(array: [Int], index: Int): Int {
            return array[index]
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 2

	const (
		// arrayIndex is the index of the parameter `array`, which is the first parameter
		arrayIndex = iota
		// indexIndex is the index of the parameter `index`, which is the second parameter
		indexIndex
	)

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			// array[index]
			opcode.InstructionGetLocal{LocalIndex: arrayIndex},
			opcode.InstructionGetLocal{LocalIndex: indexIndex},
			opcode.InstructionGetIndex{},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileAssignIndex(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(array: [Int], index: Int, value: Int) {
            array[index] = value
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// arrayIndex is the index of the parameter `array`, which is the first parameter
		arrayIndex = iota
		// indexIndex is the index of the parameter `index`, which is the second parameter
		indexIndex
		// valueIndex is the index of the parameter `value`, which is the third parameter
		valueIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetLocal{LocalIndex: arrayIndex},
			opcode.InstructionGetLocal{LocalIndex: indexIndex},
			opcode.InstructionGetLocal{LocalIndex: valueIndex},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetIndex{},
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileMember(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        struct Test {
            var foo: Int

            init(value: Int) {
                self.foo = value
            }

            fun getValue(): Int {
                return self.foo
            }
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	{
		const parameterCount = 1

		// valueIndex is the index of the parameter `value`, which is the first parameter
		const valueIndex = iota

		// localsOffset is the offset of the first local variable.
		// Initializers do not have a $result variable
		const localsOffset = parameterCount

		const (
			// selfIndex is the index of the local variable `self`, which is the first local variable
			selfIndex = localsOffset + iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let self = Test()
				opcode.InstructionNew{
					Kind:      common.CompositeKindStructure,
					TypeIndex: 0,
				},
				opcode.InstructionSetLocal{LocalIndex: selfIndex},

				// self.x = value
				opcode.InstructionGetLocal{LocalIndex: selfIndex},
				opcode.InstructionGetLocal{LocalIndex: valueIndex},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionSetField{FieldNameIndex: 0},

				// return self
				opcode.InstructionGetLocal{LocalIndex: selfIndex},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	}

	{
		const parameterCount = 1

		// nIndex is the index of the parameter `self`, which is the first parameter
		const selfIndex = 0

		// resultIndex is the index of the $result variable
		const resultIndex = parameterCount

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionGetLocal{LocalIndex: selfIndex},
				opcode.InstructionGetField{FieldNameIndex: 0},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},
				opcode.InstructionGetLocal{LocalIndex: resultIndex},
				opcode.InstructionReturnValue{},
			},
			functions[1].Code,
		)
	}

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte("foo"),
				Kind: constantkind.String,
			},
		},
		program.Constants,
	)
}

func TestCompileExpressionStatement(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun f() {}

        fun test() {
            f()
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// f()
			opcode.InstructionGetGlobal{GlobalIndex: 0},
			opcode.InstructionInvoke{TypeArgs: nil},
			opcode.InstructionDrop{},

			opcode.InstructionReturn{},
		},
		functions[1].Code,
	)
}

func TestCompileBool(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `

        fun test() {
            let yes = true
            let no = false
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// yesIndex is the index of the local variable `yes`, which is the first local variable
		yesIndex = localsOffset + iota
		// noIndex is the index of the local variable `no`, which is the second local variable
		noIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let yes = true
			opcode.InstructionTrue{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: yesIndex},

			// let no = false
			opcode.InstructionFalse{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: noIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileString(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `

        fun test(): String {
            return "Hello, world!"
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			// return "Hello, world!"
			opcode.InstructionGetConstant{ConstantIndex: 0},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte("Hello, world!"),
				Kind: constantkind.String,
			},
		},
		program.Constants,
	)
}

func TestCompileIntegers(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type) {

		t.Run(integerType.String(), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(`
                        fun test() {
                            let v: %s = 2
                        }
                    `,
					integerType,
				),
			)
			require.NoError(t, err)

			comp := compiler.NewInstructionCompiler(checker)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const parameterCount = 0

			// resultIndex is the index of the $result variable
			const resultIndex = parameterCount

			// localsOffset is the offset of the first local variable
			const localsOffset = resultIndex + 1

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = localsOffset + iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let yes = true
					opcode.InstructionGetConstant{ConstantIndex: 0},
					opcode.InstructionTransfer{TypeIndex: 0},
					opcode.InstructionSetLocal{LocalIndex: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constantkind.FromSemaType(integerType)

			assert.Equal(t,
				[]bbq.Constant{
					{
						Data: []byte{0x2},
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	for _, integerType := range common.Concat(
		sema.AllUnsignedIntegerTypes,
		sema.AllSignedIntegerTypes,
	) {
		test(integerType)
	}
}

func TestCompileFixedPoint(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type, isSigned bool) {

		t.Run(fixedPointType.String(), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(`
                        fun test() {
                            let v: %s = 2.3
                        }
                    `,
					fixedPointType,
				),
			)
			require.NoError(t, err)

			comp := compiler.NewInstructionCompiler(checker)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const parameterCount = 0

			// resultIndex is the index of the $result variable
			const resultIndex = parameterCount

			// localsOffset is the offset of the first local variable
			const localsOffset = resultIndex + 1

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = localsOffset + iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let yes = true
					opcode.InstructionGetConstant{ConstantIndex: 0},
					opcode.InstructionTransfer{TypeIndex: 0},
					opcode.InstructionSetLocal{LocalIndex: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constantkind.FromSemaType(fixedPointType)

			var expectedData []byte
			if isSigned {
				expectedData = []byte{0x80, 0x8b, 0xd6, 0xed, 0x0}
			} else {
				expectedData = []byte{0x80, 0x8b, 0xd6, 0x6d}
			}

			assert.Equal(t,
				[]bbq.Constant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	for _, fixedPointType := range sema.AllUnsignedFixedPointTypes {
		test(fixedPointType, false)
	}

	for _, fixedPointType := range sema.AllSignedFixedPointTypes {
		test(fixedPointType, true)
	}
}

func TestCompileUnaryNot(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            let no = !true
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// noIndex is the index of the local variable `no`, which is the first local variable
		noIndex = localsOffset + iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let no = !true
			opcode.InstructionTrue{},
			opcode.InstructionNot{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: noIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileUnaryNegate(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(x: Int) {
            let v = -x
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex = localsOffset + iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let v = -x
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionNegate{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: vIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileUnaryDeref(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(ref: &Int) {
            let v = *ref
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// refIndex is the index of the parameter `ref`, which is the first parameter
	const refIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex = localsOffset + iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let v = *ref
			opcode.InstructionGetLocal{LocalIndex: refIndex},
			opcode.InstructionDeref{},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: vIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileBinary(t *testing.T) {

	t.Parallel()

	test := func(op string, instruction opcode.Instruction) {

		t.Run(op, func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                        fun test() {
                            let v = 6 %s 3
                        }
                    `,
					op,
				),
			)
			require.NoError(t, err)

			comp := compiler.NewInstructionCompiler(checker)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const parameterCount = 0

			// resultIndex is the index of the $result variable
			const resultIndex = parameterCount

			// localsOffset is the offset of the first local variable
			const localsOffset = resultIndex + 1

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = localsOffset + iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let v = 6 ... 3
					opcode.InstructionGetConstant{ConstantIndex: 0},
					opcode.InstructionGetConstant{ConstantIndex: 1},
					instruction,
					opcode.InstructionTransfer{TypeIndex: 0},
					opcode.InstructionSetLocal{LocalIndex: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			assert.Equal(t,
				[]bbq.Constant{
					{
						Data: []byte{0x6},
						Kind: constantkind.Int,
					},
					{
						Data: []byte{0x3},
						Kind: constantkind.Int,
					},
				},
				program.Constants,
			)
		})
	}

	binaryInstructions := map[string]opcode.Instruction{
		"+": opcode.InstructionAdd{},
		"-": opcode.InstructionSubtract{},
		"*": opcode.InstructionMultiply{},
		"/": opcode.InstructionDivide{},
		"%": opcode.InstructionMod{},

		"<":  opcode.InstructionLess{},
		"<=": opcode.InstructionLessOrEqual{},
		">":  opcode.InstructionGreater{},
		">=": opcode.InstructionGreaterOrEqual{},

		"==": opcode.InstructionEqual{},
		"!=": opcode.InstructionNotEqual{},

		"&":  opcode.InstructionBitwiseAnd{},
		"|":  opcode.InstructionBitwiseOr{},
		"^":  opcode.InstructionBitwiseXor{},
		"<<": opcode.InstructionBitwiseLeftShift{},
		">>": opcode.InstructionBitwiseRightShift{},
	}

	for op, instruction := range binaryInstructions {
		test(op, instruction)
	}
}

func TestCompileNilCoalesce(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `

        fun test(_ value: Int?): Int {
            return value ?? 0
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 1

	// valueIndex is the index of the parameter `value`, which is the first parameter
	const valueIndex = 0

	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			// value ??
			opcode.InstructionGetLocal{LocalIndex: valueIndex},
			opcode.InstructionDup{},
			opcode.InstructionJumpIfNil{Target: 5},

			// value
			opcode.InstructionUnwrap{},
			opcode.InstructionJump{Target: 7},

			// 0
			opcode.InstructionDrop{},
			opcode.InstructionGetConstant{ConstantIndex: 0},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{0},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileMethodInvocation(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        struct Foo {
            fun f(_ x: Bool) {}
        }

        fun test() {
            let foo = Foo()
            foo.f(true)
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 3)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	{
		const parameterCount = 0

		const resultIndex = parameterCount

		const localsOffset = resultIndex + 1

		const (
			// fooIndex is the index of the local variable `foo`, which is the first local variable
			fooIndex = localsOffset + iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let foo = Foo()
				opcode.InstructionGetGlobal{GlobalIndex: 1},
				opcode.InstructionInvoke{TypeArgs: nil},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: fooIndex},

				// foo.f(true)
				opcode.InstructionGetLocal{LocalIndex: fooIndex},
				opcode.InstructionTrue{},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionGetGlobal{GlobalIndex: 2},
				opcode.InstructionInvoke{TypeArgs: nil},
				opcode.InstructionDrop{},

				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	}

	{
		const parameterCount = 0

		const resultIndex = parameterCount

		assert.Equal(t,
			[]opcode.Instruction{
				// Foo()
				opcode.InstructionNew{
					Kind:      common.CompositeKindStructure,
					TypeIndex: 0,
				},

				// assign to temp $result
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// return $result
				opcode.InstructionGetLocal{LocalIndex: resultIndex},
				opcode.InstructionReturnValue{},
			},
			functions[1].Code,
		)
	}

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionReturn{},
		},
		functions[2].Code,
	)
}

func TestCompileResourceCreateAndDestroy(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        resource Foo {}

        fun test() {
            let foo <- create Foo()
            destroy foo
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	{
		const parameterCount = 0

		const resultIndex = parameterCount

		const localsOffset = resultIndex + 1

		const (
			// fooIndex is the index of the local variable `foo`, which is the first local variable
			fooIndex = localsOffset + iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let foo <- create Foo()
				opcode.InstructionGetGlobal{GlobalIndex: 1},
				opcode.InstructionInvoke{TypeArgs: nil},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: fooIndex},

				// destroy foo
				opcode.InstructionGetLocal{LocalIndex: fooIndex},
				opcode.InstructionDestroy{},

				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	}

	{
		const parameterCount = 0

		const resultIndex = parameterCount

		assert.Equal(t,
			[]opcode.Instruction{
				// Foo()
				opcode.InstructionNew{
					Kind:      common.CompositeKindResource,
					TypeIndex: 0,
				},

				// assign to temp $result
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// return $result
				opcode.InstructionGetLocal{LocalIndex: resultIndex},
				opcode.InstructionReturnValue{},
			},
			functions[1].Code,
		)
	}
}

func TestCompilePath(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(): Path {
            return /storage/foo
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionPath{
				Domain:          common.PathDomainStorage,
				IdentifierIndex: 0,
			},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte("foo"),
				Kind: constantkind.String,
			},
		},
		program.Constants,
	)
}

func TestCompileBlockScope(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(y: Bool): Int {
            let x = 1
            if y {
                let x = 2
            } else {
                let x = 3
            }
            return x
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const parameterCount = 1

	// yIndex is the index of the parameter `y`, which is the first parameter
	const yIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// x1Index is the index of the local variable `x` in the first block, which is the first local variable
		x1Index = localsOffset + iota
		// x2Index is the index of the local variable `x` in the second block, which is the second local variable
		x2Index
		// x3Index is the index of the local variable `x` in the third block, which is the third local variable
		x3Index
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x1Index},

			// if y
			opcode.InstructionGetLocal{LocalIndex: yIndex},
			opcode.InstructionJumpIfFalse{Target: 9},

			// { let x = 2 }
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x2Index},

			opcode.InstructionJump{Target: 12},

			// else { let x = 3 }
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x3Index},

			// return x
			opcode.InstructionGetLocal{LocalIndex: x1Index},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{2},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{3},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileBlockScope2(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test(y: Bool): Int {
            let x = 1
            if y {
                var x = x
                x = 2
            } else {
                var x = x
                x = 3
            }
            return x
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const parameterCount = 1

	// yIndex is the index of the parameter `y`, which is the first parameter
	const yIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// x1Index is the index of the local variable `x` in the first block, which is the first local variable
		x1Index = localsOffset + iota
		// x2Index is the index of the local variable `x` in the second block, which is the second local variable
		x2Index
		// x3Index is the index of the local variable `x` in the third block, which is the third local variable
		x3Index
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x1Index},

			// if y
			opcode.InstructionGetLocal{LocalIndex: yIndex},
			opcode.InstructionJumpIfFalse{Target: 12},

			// var x = x
			opcode.InstructionGetLocal{LocalIndex: x1Index},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x2Index},

			// x = 2
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x2Index},

			opcode.InstructionJump{Target: 18},

			// var x = x
			opcode.InstructionGetLocal{LocalIndex: x1Index},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x3Index},

			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: x3Index},

			// return x
			opcode.InstructionGetLocal{LocalIndex: x1Index},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
			{
				Data: []byte{1},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{2},
				Kind: constantkind.Int,
			},
			{
				Data: []byte{3},
				Kind: constantkind.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileDefaultFunction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        struct interface IA {
            fun test(): Int {
                return 42
            }
        }

        struct Test: IA {}
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker).
		WithConfig(&compiler.Config{
			ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
				if location == checker.Location {
					return checker.Elaboration, nil
				}

				return nil, fmt.Errorf("cannot find elaboration for: %s", location)
			},
		})

	program := comp.Compile()

	require.Len(t, program.Functions, 3)

	const (
		concreteTypeConstructorIndex uint16 = iota
		concreteTypeFunctionIndex
		interfaceFunctionIndex
	)

	// 	`Test` type's constructor
	// Not interested in the content of the constructor.
	const concreteTypeConstructorName = "Test"
	constructor := program.Functions[concreteTypeConstructorIndex]
	require.Equal(t, concreteTypeConstructorName, constructor.Name)

	// Also check if the globals are linked properly.
	assert.Equal(t, concreteTypeConstructorIndex, comp.Globals[concreteTypeConstructorName].Index)

	// `Test` type's `test` function.

	const concreteTypeTestFuncName = "Test.test"
	concreteTypeTestFunc := program.Functions[concreteTypeFunctionIndex]
	require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.Name)

	// Also check if the globals are linked properly.
	assert.Equal(t, concreteTypeFunctionIndex, comp.Globals[concreteTypeTestFuncName].Index)

	// Should be calling into interface's default function.
	// ```
	//     fun test(): Int {
	//        var $_result: Int
	//        $_result = self.test()
	//        return $_result
	//    }
	// ```

	const (
		selfIndex = iota
		tempResultIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// self.test()
			opcode.InstructionGetLocal{LocalIndex: selfIndex},
			opcode.InstructionGetGlobal{GlobalIndex: interfaceFunctionIndex}, // must be interface method's index
			opcode.InstructionInvoke{},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 1},
			opcode.InstructionSetLocal{LocalIndex: tempResultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
			opcode.InstructionReturnValue{},
		},
		concreteTypeTestFunc.Code,
	)

	// 	`IA` type's `test` function

	const interfaceTypeTestFuncName = "IA.test"
	interfaceTypeTestFunc := program.Functions[interfaceFunctionIndex]
	require.Equal(t, interfaceTypeTestFuncName, interfaceTypeTestFunc.Name)

	// Also check if the globals are linked properly.
	assert.Equal(t, interfaceFunctionIndex, comp.Globals[interfaceTypeTestFuncName].Index)

	// Should contain the implementation.
	// ```
	//    fun test(): Int {
	//        var $_result: Int
	//        $_result = 42
	//        return $_result
	//    }
	// ```

	// Since the function is an object-method, receiver becomes the first parameter.
	const parameterCount = 1

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			// 42
			opcode.InstructionGetConstant{ConstantIndex: 0},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 1},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		interfaceTypeTestFunc.Code,
	)
}

func TestCompileFunctionConditions(t *testing.T) {

	t.Parallel()

	t.Run("pre condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
        fun test(x: Int): Int {
            pre {x > 0}
            return 5
        }
    `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		const (
			xIndex = iota
			tempResultIndex
		)

		// Would be equivalent to:
		// fun test(x: Int): Int {
		//    if !(x > 0) {
		//        panic("pre/post condition failed")
		//    }
		//    $_result = 5
		//    return $_result
		// }
		assert.Equal(t,
			[]opcode.Instruction{
				// x > 0
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionGetConstant{ConstantIndex: 0},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 10},

				// panic("pre/post condition failed")
				opcode.InstructionGetConstant{ConstantIndex: 1}, // error message
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionGetGlobal{GlobalIndex: 1}, // global index 1 is 'panic' function
				opcode.InstructionInvoke{},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// $_result = 5
				opcode.InstructionGetConstant{ConstantIndex: 2},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionSetLocal{LocalIndex: tempResultIndex},

				// return $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionReturnValue{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("post condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
        fun test(x: Int): Int {
            post {x > 0}
            return 5
        }
    `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const (
			xIndex = iota
			tempResultIndex
			resultIndex
		)

		// Would be equivalent to:
		// fun test(x: Int): Int {
		//    $_result = 5
		//    let result = $_result
		//    if !(x > 0) {
		//        panic("pre/post condition failed")
		//    }
		//    return $_result
		// }
		assert.Equal(t,
			[]opcode.Instruction{
				// $_result = 5
				opcode.InstructionGetConstant{ConstantIndex: 0},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: tempResultIndex},

				// let result = $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// x > 0
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionGetConstant{ConstantIndex: 1},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 16},

				// panic("pre/post condition failed")
				opcode.InstructionGetConstant{ConstantIndex: 2}, // error message
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionGetGlobal{GlobalIndex: 1}, // global index 1 is 'panic' function
				opcode.InstructionInvoke{},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionReturnValue{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("resource typed result var", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
        fun test(x: @AnyResource?): @AnyResource? {
            post {result != nil}
            return <- x
        }
    `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const (
			xIndex = iota
			tempResultIndex
			resultIndex
		)

		// Would be equivalent to:
		// fun test(x: @AnyResource?): @AnyResource? {
		//    var $_result <-x
		//    let result = &$_result
		//    if !(result != nil) {
		//        panic("pre/post condition failed")
		//    }
		//    return <-$_result
		//}
		assert.Equal(t,
			[]opcode.Instruction{
				// $_result = x
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: tempResultIndex},

				// Get the reference and assign to `result`.
				// i.e: `let result = &$_result`
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionNewRef{TypeIndex: 1},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// result != nil
				opcode.InstructionGetLocal{LocalIndex: resultIndex},
				opcode.InstructionNil{},
				opcode.InstructionNotEqual{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 17},

				// panic("pre/post condition failed")
				opcode.InstructionGetConstant{ConstantIndex: 0}, // error message
				opcode.InstructionTransfer{TypeIndex: 2},
				opcode.InstructionGetGlobal{GlobalIndex: 1}, // global index 1 is 'panic' function
				opcode.InstructionInvoke{},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionReturnValue{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("inherited conditions", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
            struct interface IA {
                fun test(x: Int, y: Int): Int {
                    pre {x > 0}
                    post {y > 0}
                }
            }

            struct Test: IA {
                fun test(x: Int, y: Int): Int {
                    return 42
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker).
			WithConfig(&compiler.Config{
				ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
					if location == checker.Location {
						return checker.Elaboration, nil
					}

					return nil, fmt.Errorf("cannot find elaboration for: %s", location)
				},
			})

		program := comp.Compile()
		require.Len(t, program.Functions, 2)

		// Function indexes
		const (
			concreteTypeConstructorIndex uint16 = iota
			concreteTypeFunctionIndex
			panicFunctionIndex
		)

		// 	`Test` type's constructor
		// Not interested in the content of the constructor.
		const concreteTypeConstructorName = "Test"
		constructor := program.Functions[concreteTypeConstructorIndex]
		require.Equal(t, concreteTypeConstructorName, constructor.Name)

		// Also check if the globals are linked properly.
		assert.Equal(t, concreteTypeConstructorIndex, comp.Globals[concreteTypeConstructorName].Index)

		// `Test` type's `test` function.

		// local var indexes
		const (
			xIndex = iota + 1
			yIndex
			tempResultIndex
			resultIndex
		)

		// const indexes var indexes
		const (
			const0Index = iota
			constPanicMessageIndex
			const42Index
		)

		const concreteTypeTestFuncName = "Test.test"
		concreteTypeTestFunc := program.Functions[concreteTypeFunctionIndex]
		require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.Name)

		// Also check if the globals are linked properly.
		assert.Equal(t, concreteTypeFunctionIndex, comp.Globals[concreteTypeTestFuncName].Index)

		// Would be equivalent to:
		// ```
		//     fun test(x: Int, y: Int): Int {
		//        if !(x > 0) {
		//            panic("pre/post condition failed")
		//        }
		//
		//        var $_result = 42
		//        let result = $_result
		//
		//        if !(y > 0) {
		//            panic("pre/post condition failed")
		//        }
		//
		//        return $_result
		//    }
		// ```

		assert.Equal(t,
			[]opcode.Instruction{
				// Inherited pre-condition
				// x > 0
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionGetConstant{ConstantIndex: const0Index},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 10},

				// panic("pre/post condition failed")
				opcode.InstructionGetConstant{ConstantIndex: constPanicMessageIndex},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionGetGlobal{GlobalIndex: panicFunctionIndex},
				opcode.InstructionInvoke{},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// Function body

				// $_result = 42
				opcode.InstructionGetConstant{ConstantIndex: const42Index},
				opcode.InstructionTransfer{TypeIndex: 2},
				opcode.InstructionSetLocal{LocalIndex: tempResultIndex},

				// let result = $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionTransfer{TypeIndex: 2},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// Inherited post condition

				// y > 0
				opcode.InstructionGetLocal{LocalIndex: yIndex},
				opcode.InstructionGetConstant{ConstantIndex: const0Index},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 26},

				// panic("pre/post condition failed")
				opcode.InstructionGetConstant{ConstantIndex: constPanicMessageIndex},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionGetGlobal{GlobalIndex: panicFunctionIndex},
				opcode.InstructionInvoke{},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionReturnValue{},
			},
			concreteTypeTestFunc.Code,
		)
	})

	t.Run("inherited before function", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
            struct interface IA {
                fun test(x: Int): Int {
                    post {before(x) < x}
                }
            }

            struct Test: IA {
                fun test(x: Int): Int {
                    return 42
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker).
			WithConfig(&compiler.Config{
				ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
					if location == checker.Location {
						return checker.Elaboration, nil
					}

					return nil, fmt.Errorf("cannot find elaboration for: %s", location)
				},
			})

		program := comp.Compile()
		require.Len(t, program.Functions, 2)

		// Function indexes
		const (
			concreteTypeConstructorIndex uint16 = iota
			concreteTypeFunctionIndex
			panicFunctionIndex
		)

		// 	`Test` type's constructor
		// Not interested in the content of the constructor.
		const concreteTypeConstructorName = "Test"
		constructor := program.Functions[concreteTypeConstructorIndex]
		require.Equal(t, concreteTypeConstructorName, constructor.Name)

		// Also check if the globals are linked properly.
		assert.Equal(t, concreteTypeConstructorIndex, comp.Globals[concreteTypeConstructorName].Index)

		// `Test` type's `test` function.

		// local var indexes
		const (
			// Since the function is an object-method, receiver becomes the first parameter.
			xIndex = iota + 1
			beforeVarIndex
			tempResultIndex
			resultIndex
		)

		// const indexes var indexes
		const (
			const42Index = iota
			constPanicMessageIndex
		)

		const concreteTypeTestFuncName = "Test.test"
		concreteTypeTestFunc := program.Functions[concreteTypeFunctionIndex]
		require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.Name)

		// Also check if the globals are linked properly.
		assert.Equal(t, concreteTypeFunctionIndex, comp.Globals[concreteTypeTestFuncName].Index)

		// Would be equivalent to:
		// ```
		// struct Test: IA {
		//    fun test(x: Int): Int {
		//        var $before_0 = x
		//        var $_result = 42
		//        let result = $_result
		//        if !($before_0 < x) {
		//            panic("pre/post condition failed")
		//        }
		//        return $_result
		//    }
		//}
		// ```

		assert.Equal(t,
			[]opcode.Instruction{
				// Inherited before function
				// var $before_0 = x
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionSetLocal{LocalIndex: beforeVarIndex},

				// Function body

				// $_result = 42
				opcode.InstructionGetConstant{ConstantIndex: const42Index},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionSetLocal{LocalIndex: tempResultIndex},

				// let result = $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionTransfer{TypeIndex: 1},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// Inherited post condition

				// $before_0 < x
				opcode.InstructionGetLocal{LocalIndex: beforeVarIndex},
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionLess{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 19},

				// panic("pre/post condition failed")
				opcode.InstructionGetConstant{ConstantIndex: constPanicMessageIndex},
				opcode.InstructionTransfer{TypeIndex: 2},
				opcode.InstructionGetGlobal{GlobalIndex: panicFunctionIndex},
				opcode.InstructionInvoke{},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{LocalIndex: tempResultIndex},
				opcode.InstructionReturnValue{},
			},
			concreteTypeTestFunc.Code,
		)
	})
}

func TestForLoop(t *testing.T) {

	t.Parallel()

	t.Run("array", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(array: [Int]) {
                for e in array {
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()
		require.Len(t, program.Functions, 1)

		const (
			arrayValueIndex = iota
			_               // result index (unused)
			iteratorVarIndex
			elementVarIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionGetLocal{LocalIndex: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{LocalIndex: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{LocalIndex: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 10},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{LocalIndex: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionSetLocal{LocalIndex: elementVarIndex},

				// Jump to the beginning (condition) of the loop.
				opcode.InstructionJump{Target: 3},

				// Return
				opcode.InstructionReturn{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("array with index", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(array: [Int]) {
                for i, e in array {
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()
		require.Len(t, program.Functions, 1)

		const (
			arrayValueIndex = iota
			_               // result index (unused)
			iteratorVarIndex
			indexVarIndex
			elementVarIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionGetLocal{LocalIndex: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{LocalIndex: iteratorVarIndex},

				// Initialize index.
				// `var i = -1`
				opcode.InstructionGetConstant{ConstantIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: indexVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{LocalIndex: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 16},

				// If true:

				// Increment the index
				opcode.InstructionGetLocal{LocalIndex: indexVarIndex},
				opcode.InstructionGetConstant{ConstantIndex: 1},
				opcode.InstructionAdd{},
				opcode.InstructionSetLocal{LocalIndex: indexVarIndex},

				// Get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{LocalIndex: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionSetLocal{LocalIndex: elementVarIndex},

				// Jump to the beginning (condition) of the loop.
				opcode.InstructionJump{Target: 5},

				// Return
				opcode.InstructionReturn{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("array scope", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(array: [Int]) {
                var x = 5
                for e in array {
                    var e = e
                    var x = 8
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()
		require.Len(t, program.Functions, 1)

		const (
			arrayValueIndex = iota
			_               // result index (unused)
			x1Index
			iteratorVarIndex
			e1Index
			e2Index
			x2Index
		)

		assert.Equal(t,
			[]opcode.Instruction{

				// x = 5
				opcode.InstructionGetConstant{ConstantIndex: 0},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: x1Index},

				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionGetLocal{LocalIndex: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{LocalIndex: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{LocalIndex: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 19},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{LocalIndex: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionSetLocal{LocalIndex: e1Index},

				// var e = e
				opcode.InstructionGetLocal{LocalIndex: e1Index},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: e2Index},

				// var x = 8
				opcode.InstructionGetConstant{ConstantIndex: 1},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: x2Index},

				// Jump to the beginning (condition) of the loop.
				opcode.InstructionJump{Target: 6},

				// Return
				opcode.InstructionReturn{},
			},
			program.Functions[0].Code,
		)
	})
}

func TestCompileIf(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(x: Bool): Int {
          var y = 0
          if x {
             y = 1
          } else {
             y = 2
          }
          return y
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// yIndex is the index of the local variable `y`, which is the first local variable
		yIndex = localsOffset + iota
	)

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// var y = 0
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: yIndex},

			// if x
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionJumpIfFalse{Target: 9},

			// then { y = 1 }
			opcode.InstructionGetConstant{ConstantIndex: 1},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: yIndex},

			opcode.InstructionJump{Target: 12},

			// else { y = 2 }
			opcode.InstructionGetConstant{ConstantIndex: 2},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: yIndex},

			// return y
			opcode.InstructionGetLocal{LocalIndex: yIndex},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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
		},
		program.Constants,
	)
}

func TestCompileConditional(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(x: Bool): Int {
          return x ? 1 : 2
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	const parameterCount = 1

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// return x ? 1 : 2
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionJumpIfFalse{Target: 4},

			// then: 1
			opcode.InstructionGetConstant{ConstantIndex: 0},

			opcode.InstructionJump{Target: 5},

			// else: 2
			opcode.InstructionGetConstant{ConstantIndex: 1},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]bbq.Constant{
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

func TestCompileOr(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(x: Bool, y: Bool): Bool {
          return x || y
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	const parameterCount = 2

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// yIndex is the index of the parameter `y`, which is the second parameter
		yIndex
	)

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// return x || y
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionJumpIfTrue{Target: 4},

			opcode.InstructionGetLocal{LocalIndex: yIndex},
			opcode.InstructionJumpIfFalse{Target: 6},

			opcode.InstructionTrue{},
			opcode.InstructionJump{Target: 7},

			opcode.InstructionFalse{},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileAnd(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(x: Bool, y: Bool): Bool {
          return x && y
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	const parameterCount = 2

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// yIndex is the index of the parameter `y`, which is the second parameter
		yIndex
	)

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// return x && y
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionJumpIfFalse{Target: 6},

			opcode.InstructionGetLocal{LocalIndex: yIndex},
			opcode.InstructionJumpIfFalse{Target: 6},

			opcode.InstructionTrue{},
			opcode.InstructionJump{Target: 7},

			opcode.InstructionFalse{},

			// assign to temp $result
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: resultIndex},

			// return $result
			opcode.InstructionGetLocal{LocalIndex: resultIndex},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileTransaction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        transaction {
            var count: Int

            prepare() {
                self.count = 2
            }

            pre {
                self.count == 2
            }

            execute {
                self.count = 10
            }

            post {
                self.count == 10
            }
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker).
		WithConfig(&compiler.Config{
			ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
				if location == checker.Location {
					return checker.Elaboration, nil
				}

				return nil, fmt.Errorf("cannot find elaboration for: %s", location)
			},
		})

	program := comp.Compile()
	require.Len(t, program.Functions, 3)

	// Function indexes
	const (
		transactionInitFunctionIndex uint16 = iota
		prepareFunctionIndex
		executeFunctionIndex
	)

	// Transaction constructor
	// Not interested in the content of the constructor.
	constructor := program.Functions[transactionInitFunctionIndex]
	require.Equal(t, commons.TransactionWrapperCompositeName, constructor.Name)

	// Also check if the globals are linked properly.
	assert.Equal(t, transactionInitFunctionIndex, comp.Globals[commons.TransactionWrapperCompositeName].Index)

	// constant indexes
	const (
		const2Index = iota
		constFieldNameIndex
		constErrorMsgIndex
		const10Index
	)

	// Prepare function.
	// local var indexes
	const (
		selfIndex = iota
	)

	prepareFunction := program.Functions[prepareFunctionIndex]
	require.Equal(t, commons.TransactionPrepareFunctionName, prepareFunction.Name)

	// Also check if the globals are linked properly.
	assert.Equal(t, prepareFunctionIndex, comp.Globals[commons.TransactionPrepareFunctionName].Index)

	assert.Equal(t,
		[]opcode.Instruction{
			// self.count = 2
			opcode.InstructionGetLocal{LocalIndex: selfIndex},
			opcode.InstructionGetConstant{ConstantIndex: const2Index},
			opcode.InstructionTransfer{TypeIndex: 1},
			opcode.InstructionSetField{FieldNameIndex: constFieldNameIndex},

			// return
			opcode.InstructionReturn{},
		},
		prepareFunction.Code,
	)

	// Execute function.

	// Would be equivalent to:
	//    fun execute {
	//        if !(self.count == 2) {
	//            panic("pre/post condition failed")
	//        }
	//
	//        var $_result
	//        self.count = 10
	//
	//        if !(self.count == 10) {
	//            panic("pre/post condition failed")
	//        }
	//        return
	//    }

	executeFunction := program.Functions[executeFunctionIndex]
	require.Equal(t, commons.TransactionExecuteFunctionName, executeFunction.Name)

	// Also check if the globals are linked properly.
	assert.Equal(t, executeFunctionIndex, comp.Globals[commons.TransactionExecuteFunctionName].Index)

	assert.Equal(t,
		[]opcode.Instruction{
			// Pre condition
			// `self.count == 2`
			opcode.InstructionGetLocal{LocalIndex: selfIndex},
			opcode.InstructionGetField{FieldNameIndex: constFieldNameIndex},
			opcode.InstructionGetConstant{ConstantIndex: const2Index},
			opcode.InstructionEqual{},

			// if !<condition>
			opcode.InstructionNot{},
			opcode.InstructionJumpIfFalse{Target: 11},

			// panic("pre/post condition failed")
			opcode.InstructionGetConstant{ConstantIndex: constErrorMsgIndex},
			opcode.InstructionTransfer{TypeIndex: 2},
			opcode.InstructionGetGlobal{GlobalIndex: 3}, // global index 3 is 'panic' function
			opcode.InstructionInvoke{},

			// Drop since it's a statement-expression
			opcode.InstructionDrop{},

			// self.count = 10
			opcode.InstructionGetLocal{LocalIndex: selfIndex},
			opcode.InstructionGetConstant{ConstantIndex: const10Index},
			opcode.InstructionTransfer{TypeIndex: 1},
			opcode.InstructionSetField{FieldNameIndex: constFieldNameIndex},

			// Post condition
			// `self.count == 10`
			opcode.InstructionGetLocal{LocalIndex: selfIndex},
			opcode.InstructionGetField{FieldNameIndex: constFieldNameIndex},
			opcode.InstructionGetConstant{ConstantIndex: const10Index},
			opcode.InstructionEqual{},

			// if !<condition>
			opcode.InstructionNot{},
			opcode.InstructionJumpIfFalse{Target: 26},

			// panic("pre/post condition failed")
			opcode.InstructionGetConstant{ConstantIndex: constErrorMsgIndex},
			opcode.InstructionTransfer{TypeIndex: 2},
			opcode.InstructionGetGlobal{GlobalIndex: 3}, // global index 3 is 'panic' function
			opcode.InstructionInvoke{},

			// Drop since it's a statement-expression
			opcode.InstructionDrop{},

			// return
			opcode.InstructionReturn{},
		},
		executeFunction.Code,
	)
}

func TestCompileForce(t *testing.T) {

	t.Parallel()

	t.Run("optional", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
            fun test(x: Int?): Int {
                return x!
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		const parameterCount = 1

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		// resultIndex is the index of the $result variable
		const resultIndex = parameterCount

		assert.Equal(t,
			[]opcode.Instruction{
				// return x!
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionUnwrap{},

				// assign to temp $result
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// return $result
				opcode.InstructionGetLocal{LocalIndex: resultIndex},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	})

	t.Run("non-optional", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
            fun test(x: Int): Int {
                return x!
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(checker)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		const parameterCount = 1

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		// resultIndex is the index of the $result variable
		const resultIndex = parameterCount

		assert.Equal(t,
			[]opcode.Instruction{
				// return x!
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionUnwrap{},

				// assign to temp $result
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// return $result
				opcode.InstructionGetLocal{LocalIndex: resultIndex},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	})
}

func TestCompileInnerFunction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `

        fun test(): Int {
            let x = 1
            fun inner(): Int {
                let y = 2
                return y
            }
            return x
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	{
		const parameterCount = 0

		// resultIndex is the index of the $result variable
		const resultIndex = parameterCount

		// localsOffset is the offset of the first local variable
		const localsOffset = resultIndex + 1

		const (
			// xIndex is the index of the local variable `x`, which is the first local variable
			xIndex = localsOffset + iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = 1
				opcode.InstructionGetConstant{ConstantIndex: 0},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: xIndex},

				// return x
				opcode.InstructionGetLocal{LocalIndex: xIndex},

				// assign to temp $result
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: resultIndex},

				// return $result
				opcode.InstructionGetLocal{LocalIndex: resultIndex},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	}

	{
		// TODO: inner function should also have / use a result variable

		// xIndex is the index of the local variable `x`, which is the first local variable
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = 2
				opcode.InstructionGetConstant{ConstantIndex: 1},
				opcode.InstructionTransfer{TypeIndex: 0},
				opcode.InstructionSetLocal{LocalIndex: xIndex},

				// return x
				opcode.InstructionGetLocal{LocalIndex: xIndex},
				opcode.InstructionReturnValue{},
			},
			functions[1].Code,
		)
	}
}

func TestCompileInnerFunctionOuterVariableUse(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `

        fun test() {
            let x = 1
            fun inner(): Int {
                return x
            }
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(checker)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	const (
		// xIndex is the index of the local variable `x`, which is the first local variable
		xIndex = localsOffset + iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetLocal{LocalIndex: xIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x
			opcode.InstructionGetLocal{LocalIndex: xIndex},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)
}
