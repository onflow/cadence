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

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/compiler"
	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/bbq/opcode"
	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// functionTypeIndex is the index of the function type, which is the first type
		functionTypeIndex = iota //nolint:unused
		// intTypeIndex is the index of the Int type, which is the second type
		intTypeIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// if n < 2
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 7},
			// then return n
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
			// fib(n - 1)
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionSubtract{},
			opcode.InstructionTransfer{Type: intTypeIndex},
			opcode.InstructionInvoke{ArgCount: 1},
			// fib(n - 2)
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionSubtract{},
			opcode.InstructionTransfer{Type: intTypeIndex},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionAdd{},
			// return
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
		},
		program.Constants,
	)

	assert.Equal(t,
		[]bbq.StaticType{
			interpreter.FunctionStaticType{
				Type: sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					[]sema.Parameter{
						{
							Label:          sema.ArgumentLabelNotRequired,
							Identifier:     "n",
							TypeAnnotation: sema.IntTypeAnnotation,
						},
					},
					sema.IntTypeAnnotation,
				),
			},
			interpreter.PrimitiveStaticTypeInt,
		},
		program.Types,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	const parameterCount = 0

	// nIndex is the index of the parameter `n`, which is the first parameter
	const nIndex = 0

	// localsOffset is the offset of the first local variable
	const localsOffset = parameterCount + 1

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

	const (
		// functionTypeIndex is the index of the function type, which is the first type
		functionTypeIndex = iota //nolint:unused
		// intTypeIndex is the index of the Int type, which is the second type
		intTypeIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var fib1 = 1
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: fib1Index},

			// var fib2 = 1
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: fib2Index},

			// var fibonacci = fib1
			opcode.InstructionGetLocal{Local: fib1Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: fibonacciIndex},

			// var i = 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// while i < n
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetLocal{Local: nIndex},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 33},

			// fibonacci = fib1 + fib2
			opcode.InstructionGetLocal{Local: fib1Index},
			opcode.InstructionGetLocal{Local: fib2Index},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: fibonacciIndex},

			// fib1 = fib2
			opcode.InstructionGetLocal{Local: fib2Index},
			opcode.InstructionTransfer{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: fib1Index},

			// fib2 = fibonacci
			opcode.InstructionGetLocal{Local: fibonacciIndex},
			opcode.InstructionTransfer{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: fib2Index},

			// i = i + 1
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: iIndex},

			// continue loop
			opcode.InstructionJump{Target: 12},

			// return fibonacci
			opcode.InstructionGetLocal{Local: fibonacciIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
		},
		program.Constants,
	)

	assert.Equal(t,
		[]bbq.StaticType{
			interpreter.FunctionStaticType{
				Type: sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					[]sema.Parameter{
						{
							Label:          sema.ArgumentLabelNotRequired,
							Identifier:     "n",
							TypeAnnotation: sema.IntTypeAnnotation,
						},
					},
					sema.IntTypeAnnotation,
				),
			},
			interpreter.PrimitiveStaticTypeInt,
		},
		program.Types,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	// iIndex is the index of the local variable `i`, which is the first local variable
	const iIndex = 0

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 16},

			// if i > 3
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionGreater{},
			opcode.InstructionJumpIfFalse{Target: 10},

			// break
			opcode.InstructionJump{Target: 16},

			// i = i + 1
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// repeat
			opcode.InstructionJump{Target: 3},

			// return i
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// iIndex is the index of the local variable `i`, which is the first local variable
	const iIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 17},

			// i = i + 1
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// if i < 3
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 15},

			// continue
			opcode.InstructionJump{Target: 3},

			// break
			opcode.InstructionJump{Target: 17},

			// repeat
			opcode.InstructionJump{Target: 3},

			// return i
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// xsIndex is the index of the local variable `xs`, which is the first local variable
	const xsIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// [1, 2, 3]
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionNewArray{
				Type:       1,
				Size:       3,
				IsResource: false,
			},

			// let xs =
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xsIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// xsIndex is the index of the local variable `xs`, which is the first local variable
	const xsIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// {"a": 1, "b": 2, "c": 3}
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 3},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionGetConstant{Constant: 3},
			opcode.InstructionTransfer{Type: 3},
			opcode.InstructionGetConstant{Constant: 4},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionGetConstant{Constant: 5},
			opcode.InstructionTransfer{Type: 3},
			opcode.InstructionNewDictionary{
				Type:       1,
				Size:       3,
				IsResource: false,
			},
			// let xs =
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xsIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{'a'},
				Kind: constant.String,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{'b'},
				Kind: constant.String,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
			{
				Data: []byte{'c'},
				Kind: constant.String,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		xIndex     = iota
		tempYIndex // index for the temp var to hold the value of the expression
		yIndex
	)
	assert.Equal(t,
		[]opcode.Instruction{
			// let y' = x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: tempYIndex},

			// if nil
			opcode.InstructionGetLocal{Local: tempYIndex},
			opcode.InstructionJumpIfNil{Target: 12},

			// let y = y'
			opcode.InstructionGetLocal{Local: tempYIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// then { return y }
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
			opcode.InstructionJump{Target: 15},

			// else { return 2 }
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x2},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const (
		// yIndex is the index of the parameter `y`, which is the first parameter
		yIndex = iota
		// x1Index is the index of the local variable `x` in the first block, which is the first local variable
		x1Index
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
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x1Index},

			// var z = 0
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: zIndex},

			// if let x = y
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionSetLocal{Local: tempIfLetIndex},

			opcode.InstructionGetLocal{Local: tempIfLetIndex},
			opcode.InstructionJumpIfNil{Target: 18},

			// then
			opcode.InstructionGetLocal{Local: tempIfLetIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x2Index},

			// z = x
			opcode.InstructionGetLocal{Local: x2Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: zIndex},
			opcode.InstructionJump{Target: 21},

			// else { z = x }
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: zIndex},

			// return x
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// aIndex is the index of the local variable `a`, which is the first local variable
		aIndex
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var a = 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// switch x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 13},

			// a = 1
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// jump to end
			opcode.InstructionJump{Target: 24},

			// case 2:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 21},

			// a = 2
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// jump to end
			opcode.InstructionJump{Target: 24},

			// default:
			// a = 3
			opcode.InstructionGetConstant{Constant: 3},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// return a
			opcode.InstructionGetLocal{Local: aIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// switch x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 8},
			// break
			opcode.InstructionJump{Target: 15},
			// end of case
			opcode.InstructionJump{Target: 15},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 1},
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
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// xIndex is the index of the local variable `x`, which is the first local variable
		xIndex = iota
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var x = 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			// while true
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 19},

			// switch x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 13},

			// break
			opcode.InstructionJump{Target: 13},
			// end of case
			opcode.InstructionJump{Target: 13},

			// x = x + 1
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			// repeat
			opcode.InstructionJump{Target: 3},

			// return x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransfer{Type: 1},
			// emit
			opcode.InstructionEmitEvent{
				Type:     2,
				ArgCount: 1,
			},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// x as Int?
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSimpleCast{Type: 1},

			// return
			opcode.InstructionTransfer{Type: 2},
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// x as! Int
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionForceCast{Type: 1},

			// return
			opcode.InstructionTransfer{Type: 1},
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// x as? Int
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionFailableCast{Type: 1},

			// return
			opcode.InstructionTransfer{Type: 2},
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// iIndex is the index of the local variable `i`, which is the first local variable
		iIndex = iota
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
			opcode.InstructionGetConstant{Constant: zeroIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// i < 10
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: tenIndex},
			opcode.InstructionLess{},

			opcode.InstructionJumpIfFalse{Target: 33},

			// var j = 0
			opcode.InstructionGetConstant{Constant: zeroIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: jIndex},

			// j < 10
			opcode.InstructionGetLocal{Local: jIndex},
			opcode.InstructionGetConstant{Constant: tenIndex},
			opcode.InstructionLess{},

			opcode.InstructionJumpIfFalse{Target: 26},

			// i == j
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetLocal{Local: jIndex},
			opcode.InstructionEqual{},

			opcode.InstructionJumpIfFalse{Target: 19},

			// break
			opcode.InstructionJump{Target: 26},

			// j = j + 1
			opcode.InstructionGetLocal{Local: jIndex},
			opcode.InstructionGetConstant{Constant: oneIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: jIndex},

			// continue
			opcode.InstructionJump{Target: 10},

			// repeat
			opcode.InstructionJump{Target: 10},

			// i = i + 1
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: oneIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// continue
			opcode.InstructionJump{Target: 3},

			// repeat
			opcode.InstructionJump{Target: 3},

			// return i
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0xa},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// xIndex is the index of the local variable `x`, which is the first local variable
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// var x = 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			// x = 1
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
		},
		program.Constants,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions

	const (
		xIndex = iota
	)

	// `test` function

	require.Len(t, functions, 1)
	require.Equal(t, len(functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// x = 1
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetGlobal{Global: xIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	// global var `x` initializer
	variables := program.Variables
	require.Len(t, variables, 1)
	require.Equal(t, len(variables), len(variables))

	assert.Equal(t,
		[]opcode.Instruction{
			// return 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		variables[xIndex].Getter.Code,
	)

	// Constants
	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// arrayIndex is the index of the parameter `array`, which is the first parameter
		arrayIndex = iota
		// indexIndex is the index of the parameter `index`, which is the second parameter
		indexIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// array[index]
			opcode.InstructionGetLocal{Local: arrayIndex},
			opcode.InstructionGetLocal{Local: indexIndex},
			opcode.InstructionGetIndex{},

			// return
			opcode.InstructionTransfer{Type: 1},
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
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
			opcode.InstructionGetLocal{Local: arrayIndex},
			opcode.InstructionGetLocal{Local: indexIndex},
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionTransfer{Type: 1},
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 4)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		initFuncIndex = iota
		// Next two indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
		getValueFuncIndex
	)

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
					Kind: common.CompositeKindStructure,
					Type: 1,
				},
				opcode.InstructionSetLocal{Local: selfIndex},

				// self.x = value
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetLocal{Local: valueIndex},
				opcode.InstructionTransfer{Type: 2},
				opcode.InstructionSetField{FieldName: 0},

				// return self
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionReturnValue{},
			},
			functions[initFuncIndex].Code,
		)
	}

	{
		// nIndex is the index of the parameter `self`, which is the first parameter
		const selfIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetField{FieldName: 0},
				opcode.InstructionTransfer{Type: 2},
				opcode.InstructionReturnValue{},
			},
			functions[getValueFuncIndex].Code,
		)
	}

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte("foo"),
				Kind: constant.String,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
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
			opcode.InstructionGetGlobal{Global: 0},
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// yesIndex is the index of the local variable `yes`, which is the first local variable
		yesIndex = iota
		// noIndex is the index of the local variable `no`, which is the second local variable
		noIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let yes = true
			opcode.InstructionTrue{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yesIndex},

			// let no = false
			opcode.InstructionFalse{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: noIndex},

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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// return "Hello, world!"
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte("Hello, world!"),
				Kind: constant.String,
			},
		},
		program.Constants,
	)
}

func TestCompilePositiveIntegers(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type, expectedData []byte) {

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

			comp := compiler.NewInstructionCompiler(
				interpreter.ProgramFromChecker(checker),
				checker.Location,
			)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let yes = true
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransfer{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(integerType)

			assert.Equal(t,
				[]constant.Constant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type][]byte{
		sema.IntType:   {0x2},
		sema.Int8Type:  {0x2},
		sema.Int16Type: {0x0, 0x2},
		sema.Int32Type: {0x0, 0x0, 0x0, 0x2},
		sema.Int64Type: {0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
		sema.Int128Type: {
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
		},
		sema.Int256Type: {
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
		},
		sema.UIntType:   {0x2},
		sema.UInt8Type:  {0x2},
		sema.UInt16Type: {0x0, 0x2},
		sema.UInt32Type: {0x0, 0x0, 0x0, 0x2},
		sema.UInt64Type: {0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
		sema.UInt128Type: {
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
		},
		sema.UInt256Type: {
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
		},
		sema.Word8Type:  {0x2},
		sema.Word16Type: {0x0, 0x2},
		sema.Word32Type: {0x0, 0x0, 0x0, 0x2},
		sema.Word64Type: {0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
		sema.Word128Type: {
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
		},
		sema.Word256Type: {
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
		},
	}

	for _, integerType := range common.Concat(
		sema.AllUnsignedIntegerTypes,
		sema.AllSignedIntegerTypes,
	) {
		if _, ok := tests[integerType]; !ok {
			panic(fmt.Errorf("missing test for type %s", integerType))
		}
	}

	for ty, expectedData := range tests {
		test(ty, expectedData)
	}
}

func TestCompileNegativeIntegers(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type, expectedData []byte) {

		t.Run(integerType.String(), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(`
                        fun test() {
                            let v: %s = -3
                        }
                    `,
					integerType,
				),
			)
			require.NoError(t, err)

			comp := compiler.NewInstructionCompiler(
				interpreter.ProgramFromChecker(checker),
				checker.Location,
			)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let yes = true
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransfer{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(integerType)

			assert.Equal(t,
				[]constant.Constant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type][]byte{
		sema.IntType:   {0xfd},
		sema.Int8Type:  {0xfd},
		sema.Int16Type: {0xff, 0xfd},
		sema.Int32Type: {0xff, 0xff, 0xff, 0xfd},
		sema.Int64Type: {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd},
		sema.Int128Type: {
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
		},
		sema.Int256Type: {
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xfd,
		},
	}

	for _, integerType := range sema.AllSignedIntegerTypes {
		if _, ok := tests[integerType]; !ok {
			panic(fmt.Errorf("missing test for type %s", integerType))
		}
	}

	for ty, expectedData := range tests {
		test(ty, expectedData)
	}
}

func TestCompileAddress(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            let v: Address = 0x1
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex = iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let yes = true
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: vIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Address,
			},
		},
		program.Constants,
	)
}

func TestCompilePositiveFixedPoint(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type, expectedData []byte) {

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

			comp := compiler.NewInstructionCompiler(
				interpreter.ProgramFromChecker(checker),
				checker.Location,
			)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let yes = true
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransfer{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(fixedPointType)

			assert.Equal(t,
				[]constant.Constant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type][]byte{
		sema.Fix64Type:  {0x0, 0x0, 0x0, 0x0, 0x0d, 0xb5, 0x85, 0x80},
		sema.UFix64Type: {0x0, 0x0, 0x0, 0x0, 0x0d, 0xb5, 0x85, 0x80},
	}

	for _, fixedPointType := range common.Concat(
		sema.AllUnsignedFixedPointTypes,
		sema.AllSignedFixedPointTypes,
	) {
		if _, ok := tests[fixedPointType]; !ok {
			panic(fmt.Errorf("missing test for type %s", fixedPointType))
		}
	}

	for ty, expectedData := range tests {
		test(ty, expectedData)
	}
}

func TestCompileNegativeFixedPoint(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type, expectedData []byte) {

		t.Run(fixedPointType.String(), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(`
                        fun test() {
                            let v: %s = -2.3
                        }
                    `,
					fixedPointType,
				),
			)
			require.NoError(t, err)

			comp := compiler.NewInstructionCompiler(
				interpreter.ProgramFromChecker(checker),
				checker.Location,
			)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let yes = true
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransfer{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(fixedPointType)

			assert.Equal(t,
				[]constant.Constant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type][]byte{
		sema.Fix64Type: {0xff, 0xff, 0xff, 0xff, 0xf2, 0x4a, 0x7a, 0x80},
	}

	for _, fixedPointType := range sema.AllSignedFixedPointTypes {
		if _, ok := tests[fixedPointType]; !ok {
			panic(fmt.Errorf("missing test for type %s", fixedPointType))
		}
	}

	for ty, expectedData := range tests {
		test(ty, expectedData)
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// noIndex is the index of the local variable `no`, which is the first local variable
		noIndex = iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let no = !true
			opcode.InstructionTrue{},
			opcode.InstructionNot{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: noIndex},

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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let v = -x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionNegate{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: vIndex},

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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// refIndex is the index of the parameter `ref`, which is the first parameter
		refIndex = iota
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let v = *ref
			opcode.InstructionGetLocal{Local: refIndex},
			opcode.InstructionDeref{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: vIndex},

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

			comp := compiler.NewInstructionCompiler(
				interpreter.ProgramFromChecker(checker),
				checker.Location,
			)
			program := comp.Compile()

			require.Len(t, program.Functions, 1)

			functions := comp.ExportFunctions()
			require.Equal(t, len(program.Functions), len(functions))

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let v = 6 ... 3
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionGetConstant{Constant: 1},
					instruction,
					opcode.InstructionTransfer{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			assert.Equal(t,
				[]constant.Constant{
					{
						Data: []byte{0x6},
						Kind: constant.Int,
					},
					{
						Data: []byte{0x3},
						Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// valueIndex is the index of the parameter `value`, which is the first parameter
	const valueIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// value ??
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionDup{},
			opcode.InstructionJumpIfNil{Target: 5},

			// value
			opcode.InstructionUnwrap{},
			opcode.InstructionJump{Target: 7},

			// 0
			opcode.InstructionDrop{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},

			// return
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 5)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		testFuncIndex = iota
		initFuncIndex
		// Next two indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
		fFuncIndex
	)

	{
		const (
			// fooIndex is the index of the local variable `foo`, which is the first local variable
			fooIndex = iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let foo = Foo()
				opcode.InstructionGetGlobal{Global: initFuncIndex},
				opcode.InstructionInvoke{ArgCount: 0, TypeArgs: nil},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: fooIndex},

				// foo.f(true)
				opcode.InstructionGetGlobal{Global: fFuncIndex},
				opcode.InstructionGetLocal{Local: fooIndex},
				opcode.InstructionTrue{},
				opcode.InstructionTransfer{Type: 2},
				opcode.InstructionInvokeMethodStatic{
					TypeArgs: nil,
					ArgCount: 2,
				},
				opcode.InstructionDrop{},

				opcode.InstructionReturn{},
			},
			functions[testFuncIndex].Code,
		)
	}

	{
		const parameterCount = 0

		const selfIndex = parameterCount

		assert.Equal(t,
			[]opcode.Instruction{
				// Foo()
				opcode.InstructionNew{
					Kind: common.CompositeKindStructure,
					Type: 1,
				},

				// assign to self
				opcode.InstructionSetLocal{Local: selfIndex},

				// return self
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionReturnValue{},
			},
			functions[initFuncIndex].Code,
		)
	}

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionReturn{},
		},
		functions[fFuncIndex].Code,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 4)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const (
		testFuncIndex = iota
		initFuncIndex
		// Next two indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
	)

	{
		const (
			// fooIndex is the index of the local variable `foo`, which is the first local variable
			fooIndex = iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let foo <- create Foo()
				opcode.InstructionGetGlobal{Global: initFuncIndex},
				opcode.InstructionInvoke{TypeArgs: nil},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: fooIndex},

				// destroy foo
				opcode.InstructionGetLocal{Local: fooIndex},
				opcode.InstructionDestroy{},

				opcode.InstructionReturn{},
			},
			functions[testFuncIndex].Code,
		)
	}

	{
		const parameterCount = 0

		const selfIndex = parameterCount

		assert.Equal(t,
			[]opcode.Instruction{
				// Foo()
				opcode.InstructionNew{
					Kind: common.CompositeKindResource,
					Type: 1,
				},

				// assign to self
				opcode.InstructionSetLocal{Local: selfIndex},

				// return self
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionReturnValue{},
			},
			functions[initFuncIndex].Code,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionNewPath{
				Domain:     common.PathDomainStorage,
				Identifier: 0,
			},

			// return
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte("foo"),
				Kind: constant.String,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const (
		// yIndex is the index of the parameter `y`, which is the first parameter
		yIndex = iota
		// x1Index is the index of the local variable `x` in the first block, which is the first local variable
		x1Index
		// x2Index is the index of the local variable `x` in the second block, which is the second local variable
		x2Index
		// x3Index is the index of the local variable `x` in the third block, which is the third local variable
		x3Index
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x1Index},

			// if y
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 9},

			// { let x = 2 }
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x2Index},

			opcode.InstructionJump{Target: 12},

			// else { let x = 3 }
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x3Index},

			// return x
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{1},
				Kind: constant.Int,
			},
			{
				Data: []byte{2},
				Kind: constant.Int,
			},
			{
				Data: []byte{3},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(functions), len(program.Functions))

	const (
		// yIndex is the index of the parameter `y`, which is the first parameter
		yIndex = iota
		// x1Index is the index of the local variable `x` in the first block, which is the first local variable
		x1Index
		// x2Index is the index of the local variable `x` in the second block, which is the second local variable
		x2Index
		// x3Index is the index of the local variable `x` in the third block, which is the third local variable
		x3Index
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x1Index},

			// if y
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 12},

			// var x = x
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x2Index},

			// x = 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x2Index},

			opcode.InstructionJump{Target: 18},

			// var x = x
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x3Index},

			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: x3Index},

			// return x
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{1},
				Kind: constant.Int,
			},
			{
				Data: []byte{2},
				Kind: constant.Int,
			},
			{
				Data: []byte{3},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&compiler.Config{
			ElaborationResolver: func(location common.Location) (*compiler.DesugaredElaboration, error) {
				if location == checker.Location {
					return compiler.NewDesugaredElaboration(checker.Elaboration), nil
				}

				return nil, fmt.Errorf("cannot find elaboration for: %s", location)
			},
		},
	)

	program := comp.Compile()

	require.Len(t, program.Functions, 7)

	const (
		concreteTypeConstructorIndex uint16 = iota
		// Next two indexes are for builtin methods (i.e: getType, isInstance) for concrete type
		_
		_
		concreteTypeFunctionIndex
		// Next two indexes are for builtin methods (i.e: getType, isInstance) for interface type
		_
		_
		interfaceFunctionIndex
	)

	// 	`Test` type's constructor
	// Not interested in the content of the constructor.
	const concreteTypeConstructorName = "Test"
	constructor := program.Functions[concreteTypeConstructorIndex]
	require.Equal(t, concreteTypeConstructorName, constructor.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, concreteTypeConstructorIndex, comp.Globals[concreteTypeConstructorName].Index)

	// `Test` type's `test` function.

	const concreteTypeTestFuncName = "Test.test"
	concreteTypeTestFunc := program.Functions[concreteTypeFunctionIndex]
	require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, concreteTypeFunctionIndex, comp.Globals[concreteTypeTestFuncName].Index)

	// Should be calling into interface's default function.
	// ```
	//     fun test(): Int {
	//        return self.test()
	//    }
	// ```

	const (
		selfIndex = iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// self.test()
			opcode.InstructionGetGlobal{Global: interfaceFunctionIndex}, // must be interface method's index
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionInvokeMethodStatic{
				TypeArgs: nil,
				ArgCount: 1,
			},

			// return
			opcode.InstructionTransfer{Type: 5},
			opcode.InstructionReturnValue{},
		},
		concreteTypeTestFunc.Code,
	)

	// 	`IA` type's `test` function

	const interfaceTypeTestFuncName = "IA.test"
	interfaceTypeTestFunc := program.Functions[interfaceFunctionIndex]
	require.Equal(t, interfaceTypeTestFuncName, interfaceTypeTestFunc.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, interfaceFunctionIndex, comp.Globals[interfaceTypeTestFuncName].Index)

	// Should contain the implementation.
	// ```
	//    fun test(): Int {
	//        return 42
	//    }
	// ```

	assert.Equal(t,
		[]opcode.Instruction{
			// 42
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 5},

			// return
			opcode.InstructionReturnValue{},
		},
		interfaceTypeTestFunc.Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{42},
				Kind: constant.Int,
			},
		},
		program.Constants,
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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		const (
			xIndex = iota
		)

		// Would be equivalent to:
		// fun test(x: Int): Int {
		//    if !(x > 0) {
		//        panic("pre/post condition failed")
		//    }
		//    return 5
		// }
		assert.Equal(t,
			[]opcode.Instruction{
				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 10},

				// panic("pre/post condition failed")
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 1}, // error message
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransfer{Type: 2},
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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
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
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 3},

				// let result = $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: resultIndex},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 16},

				// panic("pre/post condition failed")
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 2}, // error message
				opcode.InstructionTransfer{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 1},
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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
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
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 3},

				// Get the reference and assign to `result`.
				// i.e: `let result = &$_result`
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionNewRef{Type: 1},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: resultIndex},

				// result != nil
				opcode.InstructionGetLocal{Local: resultIndex},
				opcode.InstructionNil{},
				opcode.InstructionNotEqual{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 17},

				// panic("pre/post condition failed")
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 0}, // error message
				opcode.InstructionTransfer{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 3},
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

		comp := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			&compiler.Config{
				ElaborationResolver: func(location common.Location) (*compiler.DesugaredElaboration, error) {
					if location == checker.Location {
						return compiler.NewDesugaredElaboration(checker.Elaboration), nil
					}

					return nil, fmt.Errorf("cannot find elaboration for: %s", location)
				},
			},
		)

		program := comp.Compile()
		require.Len(t, program.Functions, 6)

		// Function indexes
		const (
			concreteTypeConstructorIndex uint16 = iota
			// Next two indexes are for builtin methods (i.e: getType, isInstance) for concrete type
			_
			_
			concreteTypeFunctionIndex
			// Next two indexes are for builtin methods (i.e: getType, isInstance) for interface type
			_
			_
			panicFunctionIndex
		)

		// 	`Test` type's constructor
		// Not interested in the content of the constructor.
		const concreteTypeConstructorName = "Test"
		constructor := program.Functions[concreteTypeConstructorIndex]
		require.Equal(t, concreteTypeConstructorName, constructor.QualifiedName)

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
		require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.QualifiedName)

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
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: const0Index},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 10},

				// panic("pre/post condition failed")
				opcode.InstructionGetGlobal{Global: panicFunctionIndex},
				opcode.InstructionGetConstant{Constant: constPanicMessageIndex},
				opcode.InstructionTransfer{Type: 5},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// Function body

				// $_result = 42
				opcode.InstructionGetConstant{Constant: const42Index},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 13},

				// let result = $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 6},
				opcode.InstructionSetLocal{Local: resultIndex},

				// Inherited post condition

				// y > 0
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionGetConstant{Constant: const0Index},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 26},

				// panic("pre/post condition failed")
				opcode.InstructionGetGlobal{Global: panicFunctionIndex},
				opcode.InstructionGetConstant{Constant: constPanicMessageIndex},
				opcode.InstructionTransfer{Type: 5},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 6},
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

		comp := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			&compiler.Config{
				ElaborationResolver: func(location common.Location) (*compiler.DesugaredElaboration, error) {
					if location == checker.Location {
						return compiler.NewDesugaredElaboration(checker.Elaboration), nil
					}

					return nil, fmt.Errorf("cannot find elaboration for: %s", location)
				},
			},
		)

		program := comp.Compile()
		require.Len(t, program.Functions, 6)

		// Function indexes
		const (
			concreteTypeConstructorIndex uint16 = iota
			// Next two indexes are for builtin methods (i.e: getType, isInstance) for concrete type
			_
			_
			concreteTypeFunctionIndex
			// Next two indexes are for builtin methods (i.e: getType, isInstance) for interface type
			_
			_
			panicFunctionIndex
		)

		// 	`Test` type's constructor
		// Not interested in the content of the constructor.
		const concreteTypeConstructorName = "Test"
		constructor := program.Functions[concreteTypeConstructorIndex]
		require.Equal(t, concreteTypeConstructorName, constructor.QualifiedName)

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
		require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.QualifiedName)

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
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransfer{Type: 5},
				opcode.InstructionSetLocal{Local: beforeVarIndex},

				// Function body

				// $_result = 42
				opcode.InstructionGetConstant{Constant: const42Index},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 6},

				// let result = $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 5},
				opcode.InstructionSetLocal{Local: resultIndex},

				// Inherited post condition

				// $before_0 < x
				opcode.InstructionGetLocal{Local: beforeVarIndex},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionLess{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 19},

				// panic("pre/post condition failed")
				opcode.InstructionGetGlobal{Global: panicFunctionIndex},
				opcode.InstructionGetConstant{Constant: constPanicMessageIndex},
				opcode.InstructionTransfer{Type: 6},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 5},
				opcode.InstructionReturnValue{},
			},
			concreteTypeTestFunc.Code,
		)
	})

	t.Run("inherited condition with transitive dependency", func(t *testing.T) {

		// Deploy contract with a type

		aContract := `
            contract A {
                struct TestStruct {
                    view fun test(): Bool {
                        log("invoked TestStruct.test()")
                        return true
                    }
                }
            }
        `

		logFunction := stdlib.NewStandardLibraryStaticFunction(
			commons.LogFunctionName,
			&sema.FunctionType{
				Purity: sema.FunctionPurityView,
				Parameters: []sema.Parameter{
					{
						Label:          sema.ArgumentLabelNotRequired,
						Identifier:     "value",
						TypeAnnotation: sema.NewTypeAnnotation(sema.AnyStructType),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					sema.VoidType,
				),
			},
			``,
			nil,
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(logFunction)

		programs := CompiledPrograms{}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		aLocation := common.NewAddressLocation(nil, contractsAddress, "A")
		bLocation := common.NewAddressLocation(nil, contractsAddress, "B")
		cLocation := common.NewAddressLocation(nil, contractsAddress, "C")
		dLocation := common.NewAddressLocation(nil, contractsAddress, "D")

		// Only need to compile
		ParseCheckAndCompileCodeWithOptions(
			t,
			aContract,
			aLocation,
			ParseCheckAndCompileOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					Config: &sema.Config{
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
			},
			programs,
		)

		// Deploy contract interface

		bContract := fmt.Sprintf(`
          import A from %[1]s

          contract interface B {

              resource interface VaultInterface {
                  var balance: Int

                  fun getBalance(): Int {
                      // Call 'A.TestStruct()' which is only available to this contract interface.
                      pre { A.TestStruct().test() }
                  }
              }
          }
        `,
			contractsAddress.HexWithPrefix(),
		)

		// Only need to compile
		ParseCheckAndCompile(t, bContract, bLocation, programs)

		// Deploy another intermediate contract interface

		cContract := fmt.Sprintf(`
          import B from %[1]s

          contract interface C: B {
              resource interface VaultIntermediateInterface: B.VaultInterface {}
          }
        `,
			contractsAddress.HexWithPrefix(),
		)

		// Only need to compile
		ParseCheckAndCompile(t, cContract, cLocation, programs)

		// Deploy contract with the implementation

		dContract := fmt.Sprintf(
			`
              import C from %[1]s

              contract D: C {

                  resource Vault: C.VaultIntermediateInterface {
                      var balance: Int

                      init(balance: Int) {
                          self.balance = balance
                      }

                      fun getBalance(): Int {
                          // Inherits a function call 'A.TestStruct()' from the grand-parent 'B',
                          // But 'A' is NOT available to this contract (as an import).
                          return self.balance
                      }
                  }

                  fun createVault(balance: Int): @Vault {
                      return <- create Vault(balance: balance)
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		dProgram := ParseCheckAndCompile(t, dContract, dLocation, programs)
		require.Len(t, dProgram.Functions, 8)

		// Function indexes
		const (
			concreteTypeFunctionIndex = 7
			panicFunctionIndex        = 11
		)

		// `D.Vault` type's `getBalance` function.

		const (
			selfIndex         = 0
			panicMessageIndex = 1
		)

		const concreteTypeTestFuncName = "D.Vault.getBalance"
		concreteTypeTestFunc := dProgram.Functions[concreteTypeFunctionIndex]
		require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.QualifiedName)

		// Would be equivalent to:
		// ```
		//  fun getBalance(): Int {
		//	  if !A.TestStruct().test() {
		//	    panic("pre/post condition failed")
		//    }
		//	  return self.balance
		//  }
		// ```

		assert.Equal(t,
			[]opcode.Instruction{
				// Get function value `A.TestStruct.test()`
				opcode.InstructionGetGlobal{Global: 9},

				// Load receiver `A.TestStruct()`
				opcode.InstructionGetGlobal{Global: 10},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionInvokeMethodStatic{
					ArgCount: 1,
				},

				// if !<condition>
				// panic("pre/post condition failed")
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 11},

				opcode.InstructionGetGlobal{Global: panicFunctionIndex},
				opcode.InstructionGetConstant{Constant: panicMessageIndex},
				opcode.InstructionTransfer{Type: 9},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return self.balance
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetField{FieldName: 0},
				opcode.InstructionTransfer{Type: 5},
				opcode.InstructionReturnValue{},
			},
			concreteTypeTestFunc.Code,
		)

		// Check whether the transitive dependency `A.TestStruct`
		// has been added as imports.

		assert.Equal(
			t,
			[]bbq.Import{
				{
					Location: aLocation,
					Name:     "A.TestStruct.test",
				},
				{
					Location: aLocation,
					Name:     "A.TestStruct",
				},
				{
					Location: nil,
					Name:     "panic",
				},
			},
			dProgram.Imports,
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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()
		require.Len(t, program.Functions, 1)

		const (
			arrayValueIndex = iota
			iteratorVarIndex
			elementVarIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionGetLocal{Local: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{Local: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 11},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: elementVarIndex},

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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()
		require.Len(t, program.Functions, 1)

		const (
			arrayValueIndex = iota
			iteratorVarIndex
			indexVarIndex
			elementVarIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionGetLocal{Local: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{Local: iteratorVarIndex},

				// Initialize index.
				// `var i = -1`
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionSetLocal{Local: indexVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 17},

				// If true:

				// Increment the index
				opcode.InstructionGetLocal{Local: indexVarIndex},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionAdd{},
				opcode.InstructionSetLocal{Local: indexVarIndex},

				// Get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: elementVarIndex},

				// Jump to the beginning (condition) of the loop.
				opcode.InstructionJump{Target: 5},

				// Return
				opcode.InstructionReturn{},
			},
			program.Functions[0].Code,
		)

		assert.Equal(t,
			[]constant.Constant{
				{
					Data: []byte{0xff},
					Kind: constant.Int,
				},
				{
					Data: []byte{0x1},
					Kind: constant.Int,
				},
			},
			program.Constants,
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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()
		require.Len(t, program.Functions, 1)

		const (
			arrayValueIndex = iota
			x1Index
			iteratorVarIndex
			e1Index
			e2Index
			x2Index
		)

		assert.Equal(t,
			[]opcode.Instruction{

				// x = 5
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: x1Index},

				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionGetLocal{Local: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{Local: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 20},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: e1Index},

				// var e = e
				opcode.InstructionGetLocal{Local: e1Index},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: e2Index},

				// var x = 8
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: x2Index},

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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// yIndex is the index of the local variable `y`, which is the first local variable
		yIndex
	)

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// var y = 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// if x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfFalse{Target: 9},

			// then { y = 1 }
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			opcode.InstructionJump{Target: 12},

			// else { y = 2 }
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return y
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x0},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// return x ? 1 : 2
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfFalse{Target: 4},

			// then: 1
			opcode.InstructionGetConstant{Constant: 0},

			opcode.InstructionJump{Target: 5},

			// else: 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},

			// return
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// yIndex is the index of the parameter `y`, which is the second parameter
		yIndex
	)

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// return x || y
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfTrue{Target: 4},

			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 6},

			opcode.InstructionTrue{},
			opcode.InstructionJump{Target: 7},

			opcode.InstructionFalse{},

			// return
			opcode.InstructionTransfer{Type: 1},
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

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// yIndex is the index of the parameter `y`, which is the second parameter
		yIndex
	)

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// return x && y
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfFalse{Target: 6},

			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 6},

			opcode.InstructionTrue{},
			opcode.InstructionJump{Target: 7},

			opcode.InstructionFalse{},

			// return
			opcode.InstructionTransfer{Type: 1},
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

	comp := compiler.NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&compiler.Config{
			ElaborationResolver: func(location common.Location) (*compiler.DesugaredElaboration, error) {
				if location == checker.Location {
					return compiler.NewDesugaredElaboration(checker.Elaboration), nil
				}

				return nil, fmt.Errorf("cannot find elaboration for: %s", location)
			},
		},
	)

	program := comp.Compile()
	require.Len(t, program.Functions, 5)

	// Function indexes
	const (
		transactionInitFunctionIndex uint16 = iota
		// Next two indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
		prepareFunctionIndex
		executeFunctionIndex
		panicFunctionIndex
	)

	// Transaction constructor
	// Not interested in the content of the constructor.
	constructor := program.Functions[transactionInitFunctionIndex]
	require.Equal(t, commons.TransactionWrapperCompositeName, constructor.QualifiedName)

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
	require.Equal(t, commons.TransactionPrepareFunctionName, prepareFunction.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, prepareFunctionIndex, comp.Globals[commons.TransactionPrepareFunctionName].Index)

	assert.Equal(t,
		[]opcode.Instruction{
			// self.count = 2
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetConstant{Constant: const2Index},
			opcode.InstructionTransfer{Type: 4},
			opcode.InstructionSetField{FieldName: constFieldNameIndex},

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
	require.Equal(t, commons.TransactionExecuteFunctionName, executeFunction.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, executeFunctionIndex, comp.Globals[commons.TransactionExecuteFunctionName].Index)

	assert.Equal(t,
		[]opcode.Instruction{
			// Pre condition
			// `self.count == 2`
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetField{FieldName: constFieldNameIndex},
			opcode.InstructionGetConstant{Constant: const2Index},
			opcode.InstructionEqual{},

			// if !<condition>
			opcode.InstructionNot{},
			opcode.InstructionJumpIfFalse{Target: 11},

			// panic("pre/post condition failed")
			opcode.InstructionGetGlobal{Global: panicFunctionIndex},
			opcode.InstructionGetConstant{Constant: constErrorMsgIndex},
			opcode.InstructionTransfer{Type: 5},
			opcode.InstructionInvoke{ArgCount: 1},

			// Drop since it's a statement-expression
			opcode.InstructionDrop{},

			// self.count = 10
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetConstant{Constant: const10Index},
			opcode.InstructionTransfer{Type: 4},
			opcode.InstructionSetField{FieldName: constFieldNameIndex},

			// Post condition
			// `self.count == 10`
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetField{FieldName: constFieldNameIndex},
			opcode.InstructionGetConstant{Constant: const10Index},
			opcode.InstructionEqual{},

			// if !<condition>
			opcode.InstructionNot{},
			opcode.InstructionJumpIfFalse{Target: 26},

			// panic("pre/post condition failed")
			opcode.InstructionGetGlobal{Global: panicFunctionIndex},
			opcode.InstructionGetConstant{Constant: constErrorMsgIndex},
			opcode.InstructionTransfer{Type: 5},
			opcode.InstructionInvoke{ArgCount: 1},

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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// return x!
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionUnwrap{},
				opcode.InstructionTransfer{Type: 1},
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

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// return x!
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionUnwrap{},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	})
}

func TestCompileReturns(t *testing.T) {

	t.Parallel()

	t.Run("empty return", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test() {
                return
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		assert.Equal(t,
			[]opcode.Instruction{
				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	})

	t.Run("value return", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(x: Int): Int {
                return x
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// return x
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	})

	t.Run("empty return with post condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test() {
                post {true}
                return
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		assert.Equal(t,
			[]opcode.Instruction{
				// Jump to post conditions
				opcode.InstructionJump{Target: 1},

				// Post condition
				opcode.InstructionTrue{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 9},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionInvoke{ArgCount: 1},
				opcode.InstructionDrop{},

				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	})

	t.Run("value return with post condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(): Int {
                post {true}
                var a = 5
                return a
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		const (
			tempResultIndex = iota
			aIndex
			resultIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// var a = 5
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: aIndex},

				// $_result = a
				opcode.InstructionGetLocal{Local: aIndex},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// Jump to post conditions
				opcode.InstructionJump{Target: 6},

				// result = $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: resultIndex},

				// Post condition
				opcode.InstructionTrue{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 17},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransfer{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	})

	t.Run("void value return with post condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test() {
                post {true}
                return voidReturnFunc()
            }

            fun voidReturnFunc() {}
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 2)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		assert.Equal(t,
			[]opcode.Instruction{
				// invoke `voidReturnFunc()`
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},

				// Drop the returning void value
				opcode.InstructionDrop{},

				// Jump to post conditions
				opcode.InstructionJump{Target: 4},

				// Post condition
				opcode.InstructionTrue{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 12},
				opcode.InstructionGetGlobal{Global: 2},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionInvoke{ArgCount: 1},
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	})
}

func TestCompileFunctionExpression(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Int {
          let addOne = fun(_ x: Int): Int {
              return x + 1
          }
          let x = 2
          return x + addOne(3)
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		addOneIndex = iota
		xIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let addOne = fun ...
			opcode.InstructionNewClosure{Function: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: addOneIndex},

			// let x = 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionSetLocal{Local: xIndex},

			// return x + addOne(3)
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionGetLocal{Local: addOneIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x + 1
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileInnerFunction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Int {
          fun addOne(_ x: Int): Int {
              return x + 1
          }
          let x = 2
          return x + addOne(3)
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		addOneIndex = iota
		xIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// fun addOne(...
			opcode.InstructionNewClosure{Function: 1},
			opcode.InstructionSetLocal{Local: addOneIndex},

			// let x = 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionSetLocal{Local: xIndex},

			// return x + addOne(3)
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionGetLocal{Local: addOneIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x + 1
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileFunctionExpressionOuterVariableUse(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            let x = 1
            let inner = fun(): Int {
                let y = 2
                return x + y
            }
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// xLocalIndex is the local index of the variable `x`, which is the first local variable
		xLocalIndex = iota
		// innerLocalIndex is the local index of the variable `inner`, which is the second local variable
		innerLocalIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xLocalIndex},

			// let inner = fun(): Int { ...
			opcode.InstructionNewClosure{
				Function: 1,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.InstructionTransfer{Type: 2},
			opcode.InstructionSetLocal{Local: innerLocalIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	// xUpvalueIndex is the upvalue index of the variable `x`, which is the first upvalue
	const xUpvalueIndex = 0

	// yIndex is the index of the local variable `y`, which is the first local variable
	const yIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// let y = 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.InstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)
}

func TestCompileInnerFunctionOuterVariableUse(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            let x = 1
            fun inner(): Int {
                let y = 2
                return x + y
            }
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// xLocalIndex is the local index of the variable `x`, which is the first local variable
		xLocalIndex = iota
		// innerLocalIndex is the local index of the variable `inner`, which is the second local variable
		innerLocalIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xLocalIndex},

			// fun inner(): Int { ...
			opcode.InstructionNewClosure{
				Function: 1,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.InstructionSetLocal{Local: innerLocalIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	// xUpvalueIndex is the upvalue index of the variable `x`, which is the first upvalue
	const xUpvalueIndex = 0

	// yIndex is the index of the local variable `y`, which is the first local variable
	const yIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// let y = 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.InstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileInnerFunctionOuterOuterVariableUse(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            let x = 1
            fun middle() {
                fun inner(): Int {
                    let y = 2
                    return x + y
                }
            }
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 3)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	const (
		// xLocalIndex is the local index of the variable `x`, which is the first local variable
		xLocalIndex = iota
		// middleLocalIndex is the local index of the variable `middle`, which is the second local variable
		middleLocalIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: xLocalIndex},

			// fun middle(): Int { ...
			opcode.InstructionNewClosure{
				Function: 1,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.InstructionSetLocal{Local: middleLocalIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	// innerLocalIndex is the local index of the variable `inner`, which is the first local variable
	const innerLocalIndex = 0

	// xUpvalueIndex is the upvalue index of the variable `x`, which is the first upvalue
	const xUpvalueIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// fun inner(): Int { ...
			opcode.InstructionNewClosure{
				Function: 2,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xUpvalueIndex,
						IsLocal:     false,
					},
				},
			},
			opcode.InstructionSetLocal{Local: innerLocalIndex},

			opcode.InstructionReturn{},
		},
		functions[1].Code,
	)

	// yIndex is the index of the local variable `y`, which is the first local variable
	const yIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// let y = 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.InstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[2].Code,
	)

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileRecursiveInnerFunction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            fun inner() {
                inner()
            }
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	// innerLocalIndex is the local index of the variable `inner`, which is the first local variable
	const innerLocalIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// fun inner() { ...
			opcode.InstructionNewClosure{
				Function: 1,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: innerLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.InstructionSetLocal{Local: innerLocalIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	// innerUpvalueIndex is the upvalue index of the variable `inner`, which is the first upvalue
	const innerUpvalueIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetUpvalue{
				Upvalue: innerUpvalueIndex,
			},
			opcode.InstructionInvoke{ArgCount: 0},
			opcode.InstructionDrop{},
			opcode.InstructionReturn{},
		},
		functions[1].Code,
	)
}

func TestCompileFunctionExpressionOuterOuterVariableUse(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test() {
          let a = 1
          let b = 2
          fun middle() {
              let c = 3
              let d = 4
              fun inner(): Int {
                  let e = 5
                  let f = 6
                  return f + e + d + b + c + a
              }
          }
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 3)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	{
		const (
			// aLocalIndex is the local index of the variable `a`, which is the first local variable
			aLocalIndex = iota
			// bLocalIndex is the local index of the variable `b`, which is the second local variable
			bLocalIndex
			// middleLocalIndex is the local index of the variable `middle`, which is the third local variable
			middleLocalIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let a = 1
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: aLocalIndex},

				// let b = 2
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: bLocalIndex},

				// fun middle(): Int { ...
				opcode.InstructionNewClosure{
					Function: 1,
					Upvalues: []opcode.Upvalue{
						{

							TargetIndex: bLocalIndex,
							IsLocal:     true,
						},
						{
							TargetIndex: aLocalIndex,
							IsLocal:     true,
						},
					},
				},
				opcode.InstructionSetLocal{Local: middleLocalIndex},

				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	}

	{
		const (
			// cLocalIndex is the local index of the variable `c`, which is the first local variable
			cLocalIndex = iota
			// dLocalIndex is the local index of the variable `d`, which is the second local variable
			dLocalIndex
			// innerLocalIndex is the local index of the variable `inner`, which is the third local variable
			innerLocalIndex
		)

		const (
			// bUpvalueIndex is the upvalue index of the variable `b`, which is the first upvalue
			bUpvalueIndex = iota
			// aUpvalueIndex is the upvalue index of the variable `a`, which is the second upvalue
			aUpvalueIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let c = 3
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: cLocalIndex},

				// let d = 4
				opcode.InstructionGetConstant{Constant: 3},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: dLocalIndex},

				// fun inner(): Int { ...
				opcode.InstructionNewClosure{
					Function: 2,
					Upvalues: []opcode.Upvalue{
						// inner uses d, b, c, a
						{
							TargetIndex: dLocalIndex,
							IsLocal:     true,
						},
						{
							TargetIndex: bUpvalueIndex,
							IsLocal:     false,
						},
						{
							TargetIndex: cLocalIndex,
							IsLocal:     true,
						},
						{
							TargetIndex: aUpvalueIndex,
							IsLocal:     false,
						},
					},
				},
				opcode.InstructionSetLocal{Local: innerLocalIndex},

				opcode.InstructionReturn{},
			},
			functions[1].Code,
		)
	}

	{
		const (
			// eLocalIndex is the local index of the variable `e`, which is the first local variable
			eLocalIndex = iota
			// fLocalIndex is the local index of the variable `f`, which is the second local variable
			fLocalIndex
		)

		const (
			// dUpvalueIndex is the upvalue index of the variable `d`, which is the first upvalue
			dUpvalueIndex = iota
			// bUpvalueIndex is the upvalue index of the variable `b`, which is the second upvalue
			bUpvalueIndex
			// cUpvalueIndex is the upvalue index of the variable `c`, which is the third upvalue
			cUpvalueIndex
			// aUpvalueIndex is the upvalue index of the variable `a`, which is the fourth upvalue
			aUpvalueIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let e = 5
				opcode.InstructionGetConstant{Constant: 4},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: eLocalIndex},

				// let f = 6
				opcode.InstructionGetConstant{Constant: 5},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: fLocalIndex},

				// return f + e + d + b + c + a
				opcode.InstructionGetLocal{Local: fLocalIndex},
				opcode.InstructionGetLocal{Local: eLocalIndex},
				opcode.InstructionAdd{},

				opcode.InstructionGetUpvalue{Upvalue: dUpvalueIndex},
				opcode.InstructionAdd{},

				opcode.InstructionGetUpvalue{Upvalue: bUpvalueIndex},
				opcode.InstructionAdd{},

				opcode.InstructionGetUpvalue{Upvalue: cUpvalueIndex},
				opcode.InstructionAdd{},

				opcode.InstructionGetUpvalue{Upvalue: aUpvalueIndex},
				opcode.InstructionAdd{},

				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionReturnValue{},
			},
			functions[2].Code,
		)
	}

	assert.Equal(t,
		[]constant.Constant{
			{
				Data: []byte{0x1},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x2},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x3},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x4},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x5},
				Kind: constant.Int,
			},
			{
				Data: []byte{0x6},
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileTransferConstant(t *testing.T) {

	t.Parallel()

	t.Run("optimized", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun test() {
              let x = 1
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = 1
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.Constant{
				{
					Data: []byte{0x1},
					Kind: constant.Int,
				},
			},
			program.Constants,
		)
	})

	t.Run("unoptimized", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun test() {
              let x: Int? = 1
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = 1
				opcode.InstructionGetConstant{Constant: 0},
				// NOTE: transfer
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.Constant{
				{
					Data: []byte{0x1},
					Kind: constant.Int,
				},
			},
			program.Constants,
		)
	})

}

func TestCompileTransferNewPath(t *testing.T) {

	t.Parallel()

	t.Run("storage", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun test() {
              let x = /storage/foo
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = /storage/foo
				opcode.InstructionNewPath{
					Domain:     common.PathDomainStorage,
					Identifier: 0,
				},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.Constant{
				{
					Data: []byte("foo"),
					Kind: constant.String,
				},
			},
			program.Constants,
		)
	})

	t.Run("public", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun test() {
              let x = /public/foo
          }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		require.Len(t, program.Functions, 1)

		functions := comp.ExportFunctions()
		require.Equal(t, len(program.Functions), len(functions))

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = /public/foo
				opcode.InstructionNewPath{
					Domain:     common.PathDomainPublic,
					Identifier: 0,
				},
				opcode.InstructionTransfer{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.Constant{
				{
					Data: []byte("foo"),
					Kind: constant.String,
				},
			},
			program.Constants,
		)
	})
}

func TestCompileTransferClosure(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test() {
          let x = fun() {}
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 2)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = fun() {}
			opcode.InstructionNewClosure{
				Function: 1,
			},
			opcode.InstructionTransfer{Type: 0},
			opcode.InstructionSetLocal{Local: 0},
			// return
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileTransferNil(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test() {
          let x: Int? = nil
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	assert.Equal(t,
		[]opcode.Instruction{
			// let x: Int? = nil
			opcode.InstructionNil{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetLocal{Local: 0},
			// return
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileArgument(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun f(_ x: Int?) {}

      fun test() {
          let x = 1
          f(x)
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
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

	const (
		// fTypeIndex is the index of the type of function `f`, which is the first type
		fTypeIndex = iota //nolint:unused
		// testTypeIndex is the index of the type of function `test`, which is the second type
		testTypeIndex //nolint:unused
		// intTypeIndex is the index of the type int, which is the third type
		intTypeIndex
		// xParameterTypeIndex is the index of the type of parameter `x`, which is the fourth type
		xParameterTypeIndex
	)

	// xIndex is the index of the local variable `x`, which is the first local variable
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetConstant{},
			opcode.InstructionTransfer{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: xIndex},
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransfer{Type: xParameterTypeIndex},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionDrop{},
			opcode.InstructionReturn{},
		},
		functions[1].Code,
	)

	assert.Equal(t,
		[]bbq.StaticType{
			interpreter.FunctionStaticType{
				Type: sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					[]sema.Parameter{
						{
							Label:      sema.ArgumentLabelNotRequired,
							Identifier: "x",
							TypeAnnotation: sema.NewTypeAnnotation(
								sema.NewOptionalType(nil, sema.IntType),
							),
						},
					},
					sema.VoidTypeAnnotation,
				),
			},
			interpreter.FunctionStaticType{
				Type: sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					[]sema.Parameter{},
					sema.VoidTypeAnnotation,
				),
			},
			interpreter.PrimitiveStaticTypeInt,
			interpreter.NewOptionalStaticType(nil, interpreter.PrimitiveStaticTypeInt),
		},
		program.Types,
	)
}

func TestCompileLineNumberInfo(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(array: [Int], index: Int, value: Int) {
          array[index] = value + value
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	require.Len(t, program.Functions, 1)

	functions := comp.ExportFunctions()
	require.Equal(t, len(program.Functions), len(functions))

	testFunction := functions[0]

	const (
		arrayIndex = iota
		indexIndex
		valueIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetLocal{Local: arrayIndex},
			opcode.InstructionGetLocal{Local: indexIndex},
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransfer{Type: 1},
			opcode.InstructionSetIndex{},
			opcode.InstructionReturn{},
		},
		testFunction.Code,
	)

	assert.Equal(t,
		[]bbq.PositionInfo{
			// Load variable `array`.
			// Opcodes:
			//   opcode.InstructionGetLocal{Local: arrayIndex}
			{
				InstructionIndex: 0,
				Position: bbq.Position{
					StartPos: ast.Position{
						Offset: 66,
						Line:   3,
						Column: 10,
					},
					EndPos: ast.Position{
						Offset: 70,
						Line:   3,
						Column: 14,
					},
				},
			},

			// Load variable `index`.
			// Opcodes:
			//  opcode.InstructionGetLocal{Local: indexIndex}
			{
				InstructionIndex: 1,
				Position: bbq.Position{
					StartPos: ast.Position{
						Offset: 72,
						Line:   3,
						Column: 16,
					},
					EndPos: ast.Position{
						Offset: 76,
						Line:   3,
						Column: 20,
					},
				},
			},

			// Load variable `value`.
			// Opcodes:
			//   opcode.InstructionGetLocal{Local: valueIndex}
			{
				InstructionIndex: 2,
				Position: bbq.Position{
					StartPos: ast.Position{
						Offset: 81,
						Line:   3,
						Column: 25,
					},
					EndPos: ast.Position{
						Offset: 85,
						Line:   3,
						Column: 29,
					},
				},
			},

			// Load variable `value`.
			// Opcodes:
			//   opcode.InstructionGetLocal{Local: valueIndex}
			{
				InstructionIndex: 3,
				Position: bbq.Position{
					StartPos: ast.Position{
						Offset: 89,
						Line:   3,
						Column: 33,
					},
					EndPos: ast.Position{
						Offset: 93,
						Line:   3,
						Column: 37,
					},
				},
			},

			// Addition `value + value`.
			// Opcodes:
			//   opcode.InstructionAdd{},
			{
				InstructionIndex: 4,
				Position: bbq.Position{
					StartPos: ast.Position{
						Offset: 81,
						Line:   3,
						Column: 25,
					},
					EndPos: ast.Position{
						Offset: 93,
						Line:   3,
						Column: 37,
					},
				},
			},

			// Assignment to array index: `array[index] = value + value`.
			// Opcodes:
			//   opcode.InstructionTransfer{Type: 1}
			//   opcode.InstructionSetIndex{}
			{
				InstructionIndex: 5,
				Position: bbq.Position{
					StartPos: ast.Position{
						Offset: 66,
						Line:   3,
						Column: 10,
					},
					EndPos: ast.Position{
						Offset: 93,
						Line:   3,
						Column: 37,
					},
				},
			},

			// This has a position same as the function declaration,
			// since this is an injected return.
			// opcode.InstructionReturn{}
			{
				InstructionIndex: 7,
				Position: bbq.Position{
					StartPos: ast.Position{
						Offset: 7,
						Line:   2,
						Column: 6,
					},
					EndPos: ast.Position{
						Offset: 101,
						Line:   4,
						Column: 6,
					},
				},
			},
		},
		testFunction.LineNumbers.Positions,
	)

	// Get position for `opcode.InstructionSetIndex{}`
	// Position must start at the start of LHS.
	pos := testFunction.LineNumbers.GetSourcePosition(6)
	assert.Equal(
		t,
		bbq.Position{
			StartPos: ast.Position{
				Offset: 66,
				Line:   3,
				Column: 10,
			},
			EndPos: ast.Position{
				Offset: 93,
				Line:   3,
				Column: 37,
			},
		},
		pos,
	)
}
