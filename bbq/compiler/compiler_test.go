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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/constantkind"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
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

	compiler := NewInstructionCompiler(checker).
		WithConfig(&Config{
			ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
				if location == checker.Location {
					return checker.Elaboration, nil
				}

				return nil, fmt.Errorf("cannot find elaboration for: %s", location)
			},
		})

	program := compiler.Compile()

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
	assert.Equal(t, concreteTypeConstructorIndex, compiler.globals[concreteTypeConstructorName].index)

	// `Test` type's `test` function.

	const concreteTypeTestFuncName = "Test.test"
	concreteTypeTestFunc := program.Functions[concreteTypeFunctionIndex]
	require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.Name)

	// Also check if the globals are linked properly.
	assert.Equal(t, concreteTypeFunctionIndex, compiler.globals[concreteTypeTestFuncName].index)

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
	assert.Equal(t, interfaceFunctionIndex, compiler.globals[interfaceTypeTestFuncName].index)

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

		compiler := NewInstructionCompiler(checker)
		program := compiler.Compile()

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

		compiler := NewInstructionCompiler(checker)
		program := compiler.Compile()

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

		compiler := NewInstructionCompiler(checker)
		program := compiler.Compile()

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

		compiler := NewInstructionCompiler(checker).
			WithConfig(&Config{
				ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
					if location == checker.Location {
						return checker.Elaboration, nil
					}

					return nil, fmt.Errorf("cannot find elaboration for: %s", location)
				},
			})

		program := compiler.Compile()
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
		assert.Equal(t, concreteTypeConstructorIndex, compiler.globals[concreteTypeConstructorName].index)

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
		assert.Equal(t, concreteTypeFunctionIndex, compiler.globals[concreteTypeTestFuncName].index)

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

		compiler := NewInstructionCompiler(checker).
			WithConfig(&Config{
				ElaborationResolver: func(location common.Location) (*sema.Elaboration, error) {
					if location == checker.Location {
						return checker.Elaboration, nil
					}

					return nil, fmt.Errorf("cannot find elaboration for: %s", location)
				},
			})

		program := compiler.Compile()
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
		assert.Equal(t, concreteTypeConstructorIndex, compiler.globals[concreteTypeConstructorName].index)

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
		assert.Equal(t, concreteTypeFunctionIndex, compiler.globals[concreteTypeTestFuncName].index)

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
