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
	"github.com/onflow/cadence/common"
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

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	// iIndex is the index of the local variable `i`, which is the first local variable
	const iIndex = localsOffset

	require.Len(t, program.Functions, 1)
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

	const parameterCount = 0

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	// localsOffset is the offset of the first local variable
	const localsOffset = resultIndex + 1

	// iIndex is the index of the local variable `i`, which is the first local variable
	const iIndex = localsOffset

	require.Len(t, program.Functions, 1)
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

	t.SkipNow()

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

func TestCompileEmit(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      event Inc(val: Int)

      fun test(x: Int) {
          emit Inc(val: x)
      }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 2)

	var testFunction *bbq.Function[opcode.Instruction]
	for _, f := range compiler.ExportFunctions() {
		if f.Name == "test" {
			testFunction = f
		}
	}
	require.NotNil(t, testFunction)

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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

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
		compiler.ExportFunctions()[0].Code,
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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

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
		compiler.ExportFunctions()[0].Code,
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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

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
		compiler.ExportFunctions()[0].Code,
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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

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
		compiler.ExportFunctions()[0].Code,
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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

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
		compiler.ExportFunctions()[0].Code,
	)
}

func TestCompileAssignGlobal(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        var x = 0

        fun test() {
            x = 1
        }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// x = 1
			opcode.InstructionGetConstant{ConstantIndex: 0},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetGlobal{GlobalIndex: 0},

			opcode.InstructionReturn{},
		},
		compiler.ExportFunctions()[0].Code,
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

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

	const parameterCount = 3

	const (
		// arrayIndex is the index of the parameter `array`, which is the first parameter
		arrayIndex = iota
		// indexIndex is the index of the parameter `index`, which is the second parameter
		indexIndex
		// valueIndex is the index of the parameter `value`, which is the third parameter
		valueIndex
	)

	// resultIndex is the index of the $result variable
	const resultIndex = parameterCount

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetLocal{LocalIndex: arrayIndex},
			opcode.InstructionGetLocal{LocalIndex: indexIndex},
			opcode.InstructionGetLocal{LocalIndex: valueIndex},
			opcode.InstructionTransfer{TypeIndex: 0},
			opcode.InstructionSetIndex{},
			opcode.InstructionReturn{},
		},
		compiler.ExportFunctions()[0].Code,
	)
}

func TestCompileAssignMember(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        struct Test {
            var x: Int

            init(value: Int) {
                self.x = value
            }
        }
    `)
	require.NoError(t, err)

	compiler := NewInstructionCompiler(checker)
	program := compiler.Compile()

	require.Len(t, program.Functions, 1)

	const (
		// valueIndex is the index of the parameter `value`, which is the first parameter
		valueIndex = iota
		// selfIndex is the index of the `self` variable
		selfIndex
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

			// return $result
			opcode.InstructionGetLocal{LocalIndex: selfIndex},
			opcode.InstructionReturnValue{},
		},
		compiler.ExportFunctions()[0].Code,
	)
}
