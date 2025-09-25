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

	"github.com/onflow/cadence/activations"
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
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

// assertGlobalsEqual compares GlobalInfo of globals
func assertGlobalsEqual(t *testing.T, expected map[string]bbq.GlobalInfo, actual map[string]bbq.Global) {
	// Check that both maps have the same keys
	assert.Equal(t, len(expected), len(actual), "globals maps have different lengths")

	for key, expectedGlobal := range expected {
		actualGlobal, exists := actual[key]
		if !assert.True(t, exists, "expected global %s not found in actual", key) {
			continue
		}

		assert.Equal(t, expectedGlobal, actualGlobal.GetGlobalInfo())
	}
}

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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// functionTypeIndex is the index of the function type, which is the first type
		functionTypeIndex = iota //nolint:unused
		// intTypeIndex is the index of the Int type, which is the second type
		intTypeIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// if n < 2
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 9},
			// then return n
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},

			// return ...
			opcode.InstructionStatement{},
			// fib(n - 1)
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionSubtract{},
			opcode.InstructionTransferAndConvert{Type: intTypeIndex},
			opcode.InstructionInvoke{ArgCount: 1},
			// fib(n - 2)
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionSubtract{},
			opcode.InstructionTransferAndConvert{Type: intTypeIndex},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionAdd{},
			// return
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// functionTypeIndex is the index of the function type, which is the first type
		functionTypeIndex = iota //nolint:unused
		// intTypeIndex is the index of the Int type, which is the second type
		intTypeIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var fib1 = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: fib1Index},

			// var fib2 = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: fib2Index},

			// var fibonacci = fib1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: fib1Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: fibonacciIndex},

			// var i = 2
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// while i < n
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetLocal{Local: nIndex},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 43},

			opcode.InstructionLoop{},

			// fibonacci = fib1 + fib2
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: fib1Index},
			opcode.InstructionGetLocal{Local: fib2Index},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: fibonacciIndex},

			// fib1 = fib2
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: fib2Index},
			opcode.InstructionTransferAndConvert{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: fib1Index},

			// fib2 = fibonacci
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: fibonacciIndex},
			opcode.InstructionTransferAndConvert{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: fib2Index},

			// i = i + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: iIndex},

			// continue loop
			opcode.InstructionJump{Target: 17},

			// return fibonacci
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: fibonacciIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// while true
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 22},

			opcode.InstructionLoop{},

			// if i > 3
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionGreater{},
			opcode.InstructionJumpIfFalse{Target: 15},

			// break
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 22},

			// i = i + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// repeat
			opcode.InstructionJump{Target: 5},

			// return i
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// iIndex is the index of the local variable `i`, which is the first local variable
	const iIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// var i = 0
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// while true
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 24},

			opcode.InstructionLoop{},

			// i = i + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// if i < 3
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 21},

			// continue
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 5},

			// break
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 24},

			// repeat
			opcode.InstructionJump{Target: 5},

			// return i
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileVoid(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test() {
          return ()
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// return nil
			opcode.InstructionStatement{},
			opcode.InstructionVoid{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileTrue(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Bool {
          return true
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{

			// return true
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileFalse(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Bool {
          return false
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// return false
			opcode.InstructionStatement{},
			opcode.InstructionFalse{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileNil(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      fun test(): Bool? {
          return nil
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// return nil
			opcode.InstructionStatement{},
			opcode.InstructionNil{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// xsIndex is the index of the local variable `xs`, which is the first local variable
	const xsIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{

			opcode.InstructionStatement{},

			// [1, 2, 3]
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionNewArray{
				Type:       1,
				Size:       3,
				IsResource: false,
			},

			// let xs =
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xsIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// xsIndex is the index of the local variable `xs`, which is the first local variable
	const xsIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// {"a": 1, "b": 2, "c": 3}
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 3},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionGetConstant{Constant: 4},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 5},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionNewDictionary{
				Type:       1,
				Size:       3,
				IsResource: false,
			},
			// let xs =
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xsIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredStringValue("a"),
				Kind: constant.String,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredStringValue("b"),
				Kind: constant.String,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredStringValue("c"),
				Kind: constant.String,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		xIndex     = iota
		tempYIndex // index for the temp var to hold the value of the expression
		yIndex
	)
	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// let y' = x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: tempYIndex},

			// if nil
			opcode.InstructionGetLocal{Local: tempYIndex},
			opcode.InstructionJumpIfNil{Target: 14},

			// let y = y'
			opcode.InstructionGetLocal{Local: tempYIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionSetLocal{Local: yIndex},

			// then { return y }
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},
			opcode.InstructionJump{Target: 18},

			// else { return 2 }
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
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

	functions := program.Functions
	require.Len(t, functions, 1)

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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x1Index},

			// var z = 0
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: zIndex},

			// if let x = y
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: tempIfLetIndex},

			opcode.InstructionGetLocal{Local: tempIfLetIndex},
			opcode.InstructionJumpIfNil{Target: 22},

			// then
			opcode.InstructionGetLocal{Local: tempIfLetIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionSetLocal{Local: x2Index},

			// z = x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: x2Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: zIndex},
			opcode.InstructionJump{Target: 26},

			// else { z = x }
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: zIndex},

			// return x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
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

	functions := program.Functions
	require.Len(t, functions, 1)

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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// switch x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 16},

			// a = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// jump to end
			opcode.InstructionJump{Target: 29},

			// case 2:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 25},

			// a = 2
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// jump to end
			opcode.InstructionJump{Target: 29},

			// default:
			// a = 3
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 3},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: aIndex},

			// return a
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: aIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// switch x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 10},
			// break
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 19},
			// end of case
			opcode.InstructionJump{Target: 19},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 17},
			// break
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 19},
			// end of case
			opcode.InstructionJump{Target: 19},

			// default:
			// break
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 19},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// xIndex is the index of the local variable `x`, which is the first local variable
		xIndex = iota
		// switchIndex is the index of the local variable used to store the value of the switch expression
		switchIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var x = 0
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			// while true
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 25},

			opcode.InstructionLoop{},

			// switch x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: switchIndex},

			// case 1:
			opcode.InstructionGetLocal{Local: switchIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionEqual{},
			opcode.InstructionJumpIfFalse{Target: 18},

			// break
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 18},
			// end of case
			opcode.InstructionJump{Target: 18},

			// x = x + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			// repeat
			opcode.InstructionJump{Target: 5},

			// return x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			// x
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// x as Int?
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSimpleCast{Type: 1},

			// return
			opcode.InstructionTransferAndConvert{Type: 2},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// x as! Int
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionForceCast{Type: 1},

			// return
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// xIndex is the index of the parameter `x`, which is the first parameter
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// x as? Int
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionFailableCast{Type: 1},

			// return
			opcode.InstructionTransferAndConvert{Type: 2},
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

	functions := program.Functions
	require.Len(t, functions, 1)

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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: zeroIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// while i < 10
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: tenIndex},
			opcode.InstructionLess{},

			opcode.InstructionJumpIfFalse{Target: 45},

			opcode.InstructionLoop{},

			// var j = 0
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: zeroIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: jIndex},

			// while j < 10
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: jIndex},
			opcode.InstructionGetConstant{Constant: tenIndex},
			opcode.InstructionLess{},
			opcode.InstructionJumpIfFalse{Target: 36},

			opcode.InstructionLoop{},

			// if i == j
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetLocal{Local: jIndex},
			opcode.InstructionEqual{},

			opcode.InstructionJumpIfFalse{Target: 27},

			// break
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 36},

			// j = j + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: jIndex},
			opcode.InstructionGetConstant{Constant: oneIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: jIndex},

			// continue
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 15},

			// repeat
			opcode.InstructionJump{Target: 15},

			// i = i + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: oneIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: iIndex},

			// continue
			opcode.InstructionStatement{},
			opcode.InstructionJump{Target: 5},

			// repeat
			opcode.InstructionJump{Target: 5},

			// return i
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(10),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// xIndex is the index of the local variable `x`, which is the first local variable
	const xIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// var x = 0
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			// x = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetGlobal{Global: xIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	// global var `x` initializer
	variables := program.Variables
	require.Len(t, variables, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// return 0
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		variables[xIndex].Getter.Code,
	)

	// Constants
	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// arrayIndex is the index of the parameter `array`, which is the first parameter
		arrayIndex = iota
		// indexIndex is the index of the parameter `index`, which is the second parameter
		indexIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// array[index]
			opcode.InstructionGetLocal{Local: arrayIndex},
			opcode.InstructionGetLocal{Local: indexIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionGetIndex{},

			// return
			opcode.InstructionTransferAndConvert{Type: 2},
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

	functions := program.Functions
	require.Len(t, functions, 1)

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
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: arrayIndex},
			opcode.InstructionGetLocal{Local: indexIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionTransferAndConvert{Type: 2},
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

	functions := program.Functions
	require.Len(t, functions, 4)

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
		// Initializers do not have a $_result variable
		const localsOffset = parameterCount

		const (
			// selfIndex is the index of the local variable `self`, which is the first local variable
			selfIndex = localsOffset + iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let self = Test()
				opcode.InstructionNewComposite{
					Kind: common.CompositeKindStructure,
					Type: 1,
				},
				opcode.InstructionSetLocal{Local: selfIndex},

				// self.foo = value
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetLocal{Local: valueIndex},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetField{FieldName: 0, AccessedType: 1},

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
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetField{FieldName: 0, AccessedType: 1},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionReturnValue{},
			},
			functions[getValueFuncIndex].Code,
		)
	}

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: "foo",
				Kind: constant.RawString,
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

	functions := program.Functions
	require.Len(t, functions, 2)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// f()
			opcode.InstructionStatement{},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// yesIndex is the index of the local variable `yes`, which is the first local variable
		yesIndex = iota
		// noIndex is the index of the local variable `no`, which is the second local variable
		noIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let yes = true
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: yesIndex},

			// let no = false
			opcode.InstructionStatement{},
			opcode.InstructionFalse{},
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// return "Hello, world!"
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredStringValue("Hello, world!"),
				Kind: constant.String,
			},
		},
		program.Constants,
	)
}

func TestCompilePositiveIntegers(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type, expectedData interpreter.Value) {

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

			functions := program.Functions
			require.Len(t, functions, 1)

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let v: ... = 2
					opcode.InstructionStatement{},
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransferAndConvert{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(integerType)

			assert.Equal(t,
				[]constant.DecodedConstant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type]interpreter.Value{
		sema.IntType:    interpreter.NewUnmeteredIntValueFromInt64(2),
		sema.Int8Type:   interpreter.NewUnmeteredInt8Value(2),
		sema.Int16Type:  interpreter.NewUnmeteredInt16Value(2),
		sema.Int32Type:  interpreter.NewUnmeteredInt32Value(2),
		sema.Int64Type:  interpreter.NewUnmeteredInt64Value(2),
		sema.Int128Type: interpreter.NewUnmeteredInt128ValueFromInt64(2),
		sema.Int256Type: interpreter.NewUnmeteredInt256ValueFromInt64(2),

		sema.UIntType:    interpreter.NewUnmeteredUIntValueFromUint64(2),
		sema.UInt8Type:   interpreter.NewUnmeteredUInt8Value(2),
		sema.UInt16Type:  interpreter.NewUnmeteredUInt16Value(2),
		sema.UInt32Type:  interpreter.NewUnmeteredUInt32Value(2),
		sema.UInt64Type:  interpreter.NewUnmeteredUInt64Value(2),
		sema.UInt128Type: interpreter.NewUnmeteredUInt128ValueFromUint64(2),
		sema.UInt256Type: interpreter.NewUnmeteredUInt256ValueFromUint64(2),

		sema.Word8Type:   interpreter.NewUnmeteredWord8Value(2),
		sema.Word16Type:  interpreter.NewUnmeteredWord16Value(2),
		sema.Word32Type:  interpreter.NewUnmeteredWord32Value(2),
		sema.Word64Type:  interpreter.NewUnmeteredWord64Value(2),
		sema.Word128Type: interpreter.NewUnmeteredWord128ValueFromUint64(2),
		sema.Word256Type: interpreter.NewUnmeteredWord256ValueFromUint64(2),
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

	test := func(integerType sema.Type, expectedData interpreter.Value) {

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

			functions := program.Functions
			require.Len(t, functions, 1)

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let v: ... = -3
					opcode.InstructionStatement{},
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransferAndConvert{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(integerType)

			assert.Equal(t,
				[]constant.DecodedConstant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type]interpreter.Value{
		sema.IntType:    interpreter.NewUnmeteredIntValueFromInt64(-3),
		sema.Int8Type:   interpreter.NewUnmeteredInt8Value(-3),
		sema.Int16Type:  interpreter.NewUnmeteredInt16Value(-3),
		sema.Int32Type:  interpreter.NewUnmeteredInt32Value(-3),
		sema.Int64Type:  interpreter.NewUnmeteredInt64Value(-3),
		sema.Int128Type: interpreter.NewUnmeteredInt128ValueFromInt64(-3),
		sema.Int256Type: interpreter.NewUnmeteredInt256ValueFromInt64(-3),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex = iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let v: Address = 0x1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: vIndex},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				Kind: constant.Address,
			},
		},
		program.Constants,
	)
}

func TestCompilePositiveFixedPoint(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type, expectedData interpreter.Value) {

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

			functions := program.Functions
			require.Len(t, functions, 1)

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let v: ... = 2.3
					opcode.InstructionStatement{},
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransferAndConvert{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(fixedPointType)

			assert.Equal(t,
				[]constant.DecodedConstant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type]interpreter.Value{
		sema.Fix64Type:   interpreter.NewUnmeteredFix64Value(230000000),
		sema.Fix128Type:  interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(23, sema.Fix128Scale-1),
		sema.UFix64Type:  interpreter.NewUnmeteredUFix64Value(230000000),
		sema.UFix128Type: interpreter.NewUnmeteredUFix128ValueWithIntegerAndScale(23, sema.Fix128Scale-1),
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

	test := func(fixedPointType sema.Type, expectedData interpreter.Value) {

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

			functions := program.Functions
			require.Len(t, functions, 1)

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let v: ... = -2.3
					opcode.InstructionStatement{},
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionTransferAndConvert{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			expectedConstantKind := constant.FromSemaType(fixedPointType)

			assert.Equal(t,
				[]constant.DecodedConstant{
					{
						Data: expectedData,
						Kind: expectedConstantKind,
					},
				},
				program.Constants,
			)
		})
	}

	tests := map[sema.Type]interpreter.Value{
		sema.Fix64Type:  interpreter.NewUnmeteredFix64Value(-230000000),
		sema.Fix128Type: interpreter.NewUnmeteredFix128ValueWithIntegerAndScale(-23, sema.Fix128Scale-1),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// noIndex is the index of the local variable `no`, which is the first local variable
		noIndex = iota
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let no = !true
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionNot{},
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// xIndex is the index of the parameter `x`, which is the first parameter
		xIndex = iota
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let v = -x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionNegate{},
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		// refIndex is the index of the parameter `ref`, which is the first parameter
		refIndex = iota
		// vIndex is the index of the local variable `v`, which is the first local variable
		vIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let v = *ref
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: refIndex},
			opcode.InstructionDeref{},
			opcode.InstructionTransferAndConvert{Type: 1},
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

			functions := program.Functions
			require.Len(t, functions, 1)

			const (
				// vIndex is the index of the local variable `v`, which is the first local variable
				vIndex = iota
			)

			assert.Equal(t,
				[]opcode.Instruction{
					// let v = 6 ... 3
					opcode.InstructionStatement{},
					opcode.InstructionGetConstant{Constant: 0},
					opcode.InstructionGetConstant{Constant: 1},
					instruction,
					opcode.InstructionTransferAndConvert{Type: 1},
					opcode.InstructionSetLocal{Local: vIndex},

					opcode.InstructionReturn{},
				},
				functions[0].Code,
			)

			assert.Equal(t,
				[]constant.DecodedConstant{
					{
						Data: interpreter.NewUnmeteredIntValueFromInt64(6),
						Kind: constant.Int,
					},
					{
						Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	// valueIndex is the index of the parameter `value`, which is the first parameter
	const valueIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// value ??
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionDup{},
			opcode.InstructionJumpIfNil{Target: 6},

			// value
			opcode.InstructionUnwrap{},
			opcode.InstructionJump{Target: 8},

			// 0
			opcode.InstructionDrop{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},

			// return
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
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

	functions := program.Functions
	require.Len(t, functions, 5)

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
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: initFuncIndex},
				opcode.InstructionInvoke{ArgCount: 0, TypeArgs: nil},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: fooIndex},

				// foo.f(true)
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: fooIndex},
				opcode.InstructionGetMethod{Method: fFuncIndex},
				opcode.InstructionTrue{},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{
					TypeArgs: nil,
					ArgCount: 1,
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
				opcode.InstructionNewComposite{
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

	functions := program.Functions
	require.Len(t, functions, 4)

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
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: initFuncIndex},
				opcode.InstructionInvoke{TypeArgs: nil},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: fooIndex},

				// destroy foo
				opcode.InstructionStatement{},
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
				opcode.InstructionNewComposite{
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// /storage/foo
			opcode.InstructionNewPath{
				Domain:     common.PathDomainStorage,
				Identifier: 0,
			},

			// return
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: "foo",
				Kind: constant.RawString,
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

	functions := program.Functions
	require.Len(t, functions, 1)

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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x1Index},

			// if y
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 12},

			// { let x = 2 }
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x2Index},

			opcode.InstructionJump{Target: 16},

			// else { let x = 3 }
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x3Index},

			// return x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 1)

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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x1Index},

			// if y
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 16},

			// var x = x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x2Index},

			// x = 2
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x2Index},

			opcode.InstructionJump{Target: 24},

			// var x = x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x3Index},

			// x = 3
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: x3Index},

			// return x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: x1Index},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 7)

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
	assert.Equal(t, concreteTypeConstructorIndex, comp.Globals[concreteTypeConstructorName].GetGlobalInfo().Index)

	// `Test` type's `test` function.

	const concreteTypeTestFuncName = "Test.test"
	concreteTypeTestFunc := program.Functions[concreteTypeFunctionIndex]
	require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, concreteTypeFunctionIndex, comp.Globals[concreteTypeTestFuncName].GetGlobalInfo().Index)

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
			opcode.InstructionStatement{},

			// self.test()
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetMethod{Method: interfaceFunctionIndex}, // must be interface method's index
			opcode.InstructionInvoke{
				TypeArgs: nil,
				ArgCount: 0,
			},

			// return
			opcode.InstructionTransferAndConvert{Type: 5},
			opcode.InstructionReturnValue{},
		},
		concreteTypeTestFunc.Code,
	)

	// 	`IA` type's `test` function

	const interfaceTypeTestFuncName = "IA.test"
	interfaceTypeTestFunc := program.Functions[interfaceFunctionIndex]
	require.Equal(t, interfaceTypeTestFuncName, interfaceTypeTestFunc.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, interfaceFunctionIndex, comp.Globals[interfaceTypeTestFuncName].GetGlobalInfo().Index)

	// Should contain the implementation.
	// ```
	//    fun test(): Int {
	//        return 42
	//    }
	// ```

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// 42
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 5},

			// return
			opcode.InstructionReturnValue{},
		},
		interfaceTypeTestFunc.Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(42),
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
                pre { x > 0 }
                return 5
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 1)

		const (
			xIndex = iota
		)

		// Would be equivalent to:
		// fun test(x: Int): Int {
		//    if !(x > 0) {
		//        $failPreCondition("")
		//    }
		//    return 5
		// }
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 12},

				// $failPreCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 1}, // error message
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionReturnValue{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("post condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(x: Int): Int {
                post { x > 0 }
                return 5
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const (
			xIndex = iota
			tempResultIndex
			resultIndex
		)

		// Would be equivalent to:
		// fun test(x: Int): Int {
		//    let $_result
		//    $_result = 5
		//    let result $noTransfer $_result
		//    if !(x > 0) {
		//       $failPostCondition("")
		//    }
		//    return $_result
		// }
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// $_result = 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 5},

				// let result $noTransfer $_result
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				opcode.InstructionStatement{},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 20},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 2}, // error message
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionReturnValue{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("resource typed result var", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(x: @AnyResource?): @AnyResource? {
                post { result != nil }
                return <- x
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const (
			xIndex = iota
			tempResultIndex
			resultIndex
		)

		// Would be equivalent to:
		// fun test(x: @AnyResource?): @AnyResource? {
		//    var $_result <- x
		//    let result $noTransfer &$_result
		//    if !(result != nil) {
		//        $failPostCondition("")
		//    }
		//    return <-$_result
		//}
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// $_result <- x
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransfer{},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 6},

				// Get the reference and assign to `result`.
				// i.e: `let result $noTransfer &$_result`
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionNewRef{Type: 1},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				opcode.InstructionStatement{},

				// result != nil
				opcode.InstructionGetLocal{Local: resultIndex},
				opcode.InstructionNil{},
				opcode.InstructionNotEqual{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 22},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 0}, // error message
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionReturnValue{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("inherited conditions", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            struct interface IA {
                fun test(x: Int, y: Int): Int {
                    pre { x > 0 }
                    post { y > 0 }
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
		functions := program.Functions
		require.Len(t, functions, 6)

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
			failPreConditionFunctionIndex
			failPostConditionFunctionIndex
		)

		// 	`Test` type's constructor
		// Not interested in the content of the constructor.
		const concreteTypeConstructorName = "Test"
		constructor := program.Functions[concreteTypeConstructorIndex]
		require.Equal(t, concreteTypeConstructorName, constructor.QualifiedName)

		// Also check if the globals are linked properly.
		assert.Equal(t, concreteTypeConstructorIndex, comp.Globals[concreteTypeConstructorName].GetGlobalInfo().Index)

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
		assert.Equal(t, concreteTypeFunctionIndex, comp.Globals[concreteTypeTestFuncName].GetGlobalInfo().Index)

		// Would be equivalent to:
		// ```
		//     fun test(x: Int, y: Int): Int {
		//        if !(x > 0) {
		//            $failPreCondition("")
		//        }
		//
		//        var $_result = 42
		//        let result $noTransfer $_result
		//
		//        if !(y > 0) {
		//            $failPostCondition("")
		//        }
		//
		//        return $_result
		//    }
		// ```

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// Inherited pre-condition
				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: const0Index},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 12},

				// $failPreCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: failPreConditionFunctionIndex},
				opcode.InstructionGetConstant{Constant: constPanicMessageIndex},
				opcode.InstructionTransferAndConvert{Type: 5},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// Function body

				opcode.InstructionStatement{},

				// $_result = 42
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: const42Index},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 17},

				// let result $noTransfer $_result
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				// Inherited post condition

				opcode.InstructionStatement{},

				// y > 0
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionGetConstant{Constant: const0Index},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 32},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: failPostConditionFunctionIndex},
				opcode.InstructionGetConstant{Constant: constPanicMessageIndex},
				opcode.InstructionTransferAndConvert{Type: 5},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 6},
				opcode.InstructionReturnValue{},
			},
			concreteTypeTestFunc.Code,
		)
	})

	t.Run("inherited before function", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            struct interface IA {
                fun test(x: Int): Int {
                    post { before(x) < x }
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
		functions := program.Functions
		require.Len(t, functions, 6)

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
			failPostConditionFunctionIndex
		)

		// 	`Test` type's constructor
		// Not interested in the content of the constructor.
		const concreteTypeConstructorName = "Test"
		constructor := program.Functions[concreteTypeConstructorIndex]
		require.Equal(t, concreteTypeConstructorName, constructor.QualifiedName)

		// Also check if the globals are linked properly.
		assert.Equal(t, concreteTypeConstructorIndex, comp.Globals[concreteTypeConstructorName].GetGlobalInfo().Index)

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
		assert.Equal(t, concreteTypeFunctionIndex, comp.Globals[concreteTypeTestFuncName].GetGlobalInfo().Index)

		// Would be equivalent to:
		// ```
		// struct Test: IA {
		//    fun test(x: Int): Int {
		//        var $before_0 = x
		//        var $_result = 42
		//        let result $noTransfer $_result
		//        if !($before_0 < x) {
		//            $failPostCondition("")
		//        }
		//        return $_result
		//    }
		//}
		// ```

		assert.Equal(t,
			[]opcode.Instruction{
				// Inherited before function

				// var $before_0 = x
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransferAndConvert{Type: 5},
				opcode.InstructionSetLocal{Local: beforeVarIndex},

				// Function body

				opcode.InstructionStatement{},

				// $_result = 42
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: const42Index},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 9},

				// let result $noTransfer $_result
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				// Inherited post condition

				opcode.InstructionStatement{},

				// $before_0 < x
				opcode.InstructionGetLocal{Local: beforeVarIndex},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionLess{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 24},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: failPostConditionFunctionIndex},
				opcode.InstructionGetConstant{Constant: constPanicMessageIndex},
				opcode.InstructionTransferAndConvert{Type: 6},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 5},
				opcode.InstructionReturnValue{},
			},
			concreteTypeTestFunc.Code,
		)
	})

	t.Run("inherited condition with transitive dependency", func(t *testing.T) {

		t.Parallel()

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

		// like stdlib log function, but view/pure
		logFunction := stdlib.NewVMStandardLibraryStaticFunction(
			stdlib.LogFunctionName,
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
			// not needed for this test
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
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
					},
				},
				CompilerConfig: &compiler.Config{
					BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
						activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
						activation.Set(
							stdlib.LogFunctionName,
							compiler.NewGlobalImport(stdlib.LogFunctionName),
						)
						return activation
					},
				},
			},
			programs,
		)

		// Deploy contract interface

		bContract := fmt.Sprintf(
			`
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

		cContract := fmt.Sprintf(
			`
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
			concreteTypeFunctionIndex     = 7
			failPreConditionFunctionIndex = 11
		)

		// `D.Vault` type's `getBalance` function.

		// Local var indexes
		const (
			selfIndex = iota
		)

		// Constant indexes
		const (
			fieldNameIndex    = 1
			panicMessageIndex = 2
		)

		const concreteTypeTestFuncName = "D.Vault.getBalance"
		concreteTypeTestFunc := dProgram.Functions[concreteTypeFunctionIndex]
		require.Equal(t, concreteTypeTestFuncName, concreteTypeTestFunc.QualifiedName)

		// Would be equivalent to:
		// ```
		//  fun getBalance(): Int {
		//	  if !A.TestStruct().test() {
		//	    $failPreCondition("")
		//    }
		//	  return self.balance
		//  }
		// ```

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// Load receiver `A.TestStruct()`
				opcode.InstructionGetGlobal{Global: 9},
				opcode.InstructionInvoke{ArgCount: 0},

				// Get function value `A.TestStruct.test()`
				opcode.InstructionGetMethod{Method: 10},
				opcode.InstructionInvoke{
					ArgCount: 0,
				},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 13},

				// $failPreCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: failPreConditionFunctionIndex},
				opcode.InstructionGetConstant{Constant: panicMessageIndex},
				opcode.InstructionTransferAndConvert{Type: 9},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return self.balance
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetField{
					FieldName:    fieldNameIndex,
					AccessedType: 6,
				},
				opcode.InstructionTransferAndConvert{Type: 5},
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
					Name:     "A.TestStruct",
				},
				{
					Location: aLocation,
					Name:     "A.TestStruct.test",
				},
				{
					Location: nil,
					Name:     "$failPreCondition",
				},
			},
			dProgram.Imports,
		)
	})

	t.Run("conditions order", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(x: Int): Int {
                pre { x > 0 }
                post { before(x) < x }
                return 5
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
		functions := program.Functions
		require.Len(t, functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const (
			xIndex = iota
			beforeExprValueIndex
			tempResultIndex
			resultIndex
		)

		// Would be equivalent to:
		//
		// fun test(x: Int): Int {
		//     // before-statements comes first
		//     var exp_0 = x
		//
		//     // Pre-conditions
		//     if !(x > 0) {
		//         $failPreCondition("")
		//     }
		//
		//     $_result = 5
		//     let result $noTransfer $_result
		//
		//     // Post-conditions
		//     if !(exp_0 < x) {
		//         $failPostCondition("")
		//     }
		//
		//     return $_result
		// }
		assert.Equal(t,
			[]opcode.Instruction{

				// Before-statements

				// var exp_0 = x
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: beforeExprValueIndex},

				// Pre conditions

				// if !(x > 0)
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionGreater{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 16},

				// $failPreCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{
					ArgCount: 1,
				},
				opcode.InstructionDrop{},

				// Function body
				opcode.InstructionStatement{},

				// $_result = 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 21},

				// let result $noTransfer $_result
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				// Post conditions

				// if !(exp_0 < x)
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: beforeExprValueIndex},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionLess{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 36},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 2},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{
					ArgCount: 1,
				},
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},
			},
			program.Functions[0].Code,
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
		functions := program.Functions
		require.Len(t, functions, 1)

		const (
			arrayValueIndex = iota
			iteratorVarIndex
			elementVarIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionGetLocal{Local: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{Local: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 13},

				opcode.InstructionLoop{},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: elementVarIndex},

				// Jump to the beginning (condition) of the loop.
				opcode.InstructionJump{Target: 4},

				// End of the loop, end the iterator.
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorEnd{},

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
		functions := program.Functions
		require.Len(t, functions, 1)

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
				opcode.InstructionStatement{},
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
				opcode.InstructionJumpIfFalse{Target: 19},

				opcode.InstructionLoop{},

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
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: elementVarIndex},

				// Jump to the beginning (condition) of the loop.
				opcode.InstructionJump{Target: 6},

				// End of the loop, end the iterator.
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorEnd{},

				// Return
				opcode.InstructionReturn{},
			},
			program.Functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredIntValueFromInt64(-1),
					Kind: constant.Int,
				},
				{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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
		functions := program.Functions
		require.Len(t, functions, 1)

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

				// var x = 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: x1Index},

				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: arrayValueIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{Local: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 25},

				opcode.InstructionLoop{},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorNext{},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: e1Index},

				// var e = e
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: e1Index},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: e2Index},

				// var x = 8
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: x2Index},

				// Jump to the beginning (condition) of the loop.
				opcode.InstructionJump{Target: 8},

				// End of the loop, end the iterator.
				opcode.InstructionGetLocal{Local: iteratorVarIndex},
				opcode.InstructionIteratorEnd{},

				// Return
				opcode.InstructionReturn{},
			},
			program.Functions[0].Code,
		)
	})

	t.Run("nested, with return", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(a: [Int], b: [Int]): Int {
                for x in a {
                    for y in b {
                        return x + y
                    }
                    return x
                }
                return 0
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()
		functions := program.Functions
		require.Len(t, functions, 1)

		const (
			aIndex = iota
			bIndex
			iter1Index
			xIndex
			iter2Index
			yIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{

				// Get the iterator and store in local var.
				// `var <iterator> = a.Iterator`
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: aIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{Local: iter1Index},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{Local: iter1Index},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 44},

				opcode.InstructionLoop{},

				// If true, get the next element and store in local var.
				// var x = iterator.next()
				opcode.InstructionGetLocal{Local: iter1Index},
				opcode.InstructionIteratorNext{},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: xIndex},

				// Get the iterator and store in local var.
				// `var <iterator> = b.Iterator`
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: bIndex},
				opcode.InstructionIterator{},
				opcode.InstructionSetLocal{Local: iter2Index},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.InstructionGetLocal{Local: iter2Index},
				opcode.InstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.InstructionJumpIfFalse{Target: 35},

				opcode.InstructionLoop{},

				// If true, get the next element and store in local var.
				// var y = iterator.next()
				opcode.InstructionGetLocal{Local: iter2Index},
				opcode.InstructionIteratorNext{},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: yIndex},

				// return x + y
				// Also, end all active iterators (inner and outer).
				opcode.InstructionStatement{},

				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionAdd{},

				opcode.InstructionGetLocal{Local: iter1Index},
				opcode.InstructionIteratorEnd{},
				opcode.InstructionGetLocal{Local: iter2Index},
				opcode.InstructionIteratorEnd{},

				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},

				// Jump to the beginning (condition) of the inner loop.
				opcode.InstructionJump{Target: 16},

				// End of the loop, end the inner iterator.
				opcode.InstructionGetLocal{Local: iter2Index},
				opcode.InstructionIteratorEnd{},

				// return x
				// Also, end all active iterators (outer).
				opcode.InstructionStatement{},

				opcode.InstructionGetLocal{Local: xIndex},

				opcode.InstructionGetLocal{Local: iter1Index},
				opcode.InstructionIteratorEnd{},

				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},

				// Jump to the beginning (condition) of the outer loop.
				opcode.InstructionJump{Target: 4},

				// End of the loop, end the outer iterator.
				opcode.InstructionGetLocal{Local: iter1Index},
				opcode.InstructionIteratorEnd{},

				// return 0
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// var y = 0
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// if x
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfFalse{Target: 12},

			// then { y = 1 }
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			opcode.InstructionJump{Target: 16},

			// else { y = 2 }
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return y
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// return x ? 1 : 2
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfFalse{Target: 5},

			// then: 1
			opcode.InstructionGetConstant{Constant: 0},

			opcode.InstructionJump{Target: 6},

			// else: 2
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},

			// return
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x || y
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfTrue{Target: 5},

			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 7},

			opcode.InstructionTrue{},
			opcode.InstructionJump{Target: 8},

			opcode.InstructionFalse{},

			// return
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x && y
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionJumpIfFalse{Target: 7},

			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionJumpIfFalse{Target: 7},

			opcode.InstructionTrue{},
			opcode.InstructionJump{Target: 8},

			opcode.InstructionFalse{},

			// return
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)
}

func TestCompileTransaction(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithOptions(t,
		`
            transaction(n: Int) {

                var count: Int

                prepare() {
                    self.count = 1 + n
                }

                pre {
                    self.count == 2 + n: "pre failed"
                }

                execute {
                    self.count = 3 + n
                }

                post {
                    self.count == 4 + n: "post failed"
                }
            }
        `,
		ParseAndCheckOptions{
			Location: common.TransactionLocation{},
		},
	)
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
	functions := program.Functions
	require.Len(t, functions, 6)

	// constant indexes
	const (
		oneConstIndex = iota
		fieldNameConstIndex
		twoConstIndex
		preErrorMessageConstIndex
		threeConstIndex
		fourConstIndex
		postErrorMessageConstIndex
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: "count",
				Kind: constant.RawString,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredStringValue("pre failed"),
				Kind: constant.String,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(4),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredStringValue("post failed"),
				Kind: constant.String,
			},
		},
		program.Constants,
	)

	// Function indexes
	const (
		transactionInitFunctionIndex uint16 = iota
		// Next two indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
		prepareFunctionIndex
		executeFunctionIndex
		programInitFunctionIndex
	)

	const transactionParameterCount = 1

	const (
		nGlobalIndex = iota
		// Next 6 indexes are for functions, see above
		_
		_
		_
		_
		_
		_
		failPreConditionGlobalIndex
		failPostConditionGlobalIndex
	)

	// Transaction constructor
	// Not interested in the content of the constructor.
	constructor := program.Functions[transactionInitFunctionIndex]
	require.Equal(t,
		commons.TransactionWrapperCompositeName,
		constructor.QualifiedName,
	)

	// Also check if the globals are linked properly.
	assert.Equal(t,
		transactionParameterCount+transactionInitFunctionIndex,
		comp.Globals[commons.TransactionWrapperCompositeName].GetGlobalInfo().Index,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionNewSimpleComposite{
				Kind: common.CompositeKindStructure,
				Type: 1,
			},
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionReturnValue{},
		},
		constructor.Code,
	)

	// Prepare function.
	// local var indexes
	const (
		selfIndex = iota
	)

	prepareFunction := program.Functions[prepareFunctionIndex]
	require.Equal(t,
		commons.TransactionPrepareFunctionName,
		prepareFunction.QualifiedName,
	)

	// Also check if the globals are linked properly.
	assert.Equal(t,
		transactionParameterCount+prepareFunctionIndex,
		comp.Globals[commons.TransactionPrepareFunctionName].GetGlobalInfo().Index,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// self.count = 1 + n
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetConstant{Constant: oneConstIndex},
			opcode.InstructionGetGlobal{Global: nGlobalIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 4},
			opcode.InstructionSetField{
				FieldName:    fieldNameConstIndex,
				AccessedType: 1,
			},

			// return
			opcode.InstructionReturn{},
		},
		prepareFunction.Code,
	)

	// Execute function.

	// Would be equivalent to:
	//    fun execute {
	//        if !(self.count == 2 + n) {
	//            $failPreCondition("pre failed")
	//        }
	//
	//        var $_result
	//        self.count = 3 + n
	//
	//        if !(self.count == 4 + n) {
	//            $failPostCondition("post failed")
	//        }
	//        return
	//    }

	executeFunction := program.Functions[executeFunctionIndex]
	require.Equal(t, commons.TransactionExecuteFunctionName, executeFunction.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t,
		transactionParameterCount+executeFunctionIndex,
		comp.Globals[commons.TransactionExecuteFunctionName].GetGlobalInfo().Index,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// Pre condition
			// `self.count == 2 + n: "pre failed"`
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetField{
				FieldName:    fieldNameConstIndex,
				AccessedType: 1,
			},
			opcode.InstructionGetConstant{Constant: twoConstIndex},
			opcode.InstructionGetGlobal{Global: nGlobalIndex},
			opcode.InstructionAdd{},
			opcode.InstructionEqual{},

			// if !<condition>
			opcode.InstructionNot{},
			opcode.InstructionJumpIfFalse{Target: 15},

			// $failPreCondition("pre failed")
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: failPreConditionGlobalIndex},
			opcode.InstructionGetConstant{Constant: preErrorMessageConstIndex},
			opcode.InstructionTransferAndConvert{Type: 5},
			opcode.InstructionInvoke{ArgCount: 1},

			// Drop since it's a statement-expression
			opcode.InstructionDrop{},

			// self.count = 3 + n
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetConstant{Constant: threeConstIndex},
			opcode.InstructionGetGlobal{Global: nGlobalIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 4},
			opcode.InstructionSetField{
				FieldName:    fieldNameConstIndex,
				AccessedType: 1,
			},

			// Post condition
			// `self.count == 4 + n: "post failed"`
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetField{
				FieldName:    fieldNameConstIndex,
				AccessedType: 1,
			},
			opcode.InstructionGetConstant{Constant: fourConstIndex},
			opcode.InstructionGetGlobal{Global: nGlobalIndex},
			opcode.InstructionAdd{},
			opcode.InstructionEqual{},

			// if !<condition>
			opcode.InstructionNot{},
			opcode.InstructionJumpIfFalse{Target: 37},

			// $failPostCondition("post failed")
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: failPostConditionGlobalIndex},
			opcode.InstructionGetConstant{Constant: postErrorMessageConstIndex},
			opcode.InstructionTransferAndConvert{Type: 5},
			opcode.InstructionInvoke{ArgCount: 1},

			// Drop since it's a statement-expression
			opcode.InstructionDrop{},

			// return
			opcode.InstructionReturn{},
		},
		executeFunction.Code,
	)

	// Program init function
	initFunction := program.Functions[programInitFunctionIndex]
	require.Equal(t,
		commons.ProgramInitFunctionName,
		initFunction.QualifiedName,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// n = $_param_n
			opcode.InstructionGetLocal{Local: 0},
			// NOTE: no transfer, intentional to avoid copy
			opcode.InstructionSetGlobal{Global: nGlobalIndex},
		},
		initFunction.Code,
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

		functions := program.Functions
		require.Len(t, functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// return x!
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionUnwrap{},
				opcode.InstructionTransferAndConvert{Type: 1},
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

		functions := program.Functions
		require.Len(t, functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// return x!
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionUnwrap{},
				opcode.InstructionTransferAndConvert{Type: 1},
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

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// return
				opcode.InstructionStatement{},
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

		functions := program.Functions
		require.Len(t, functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// return x
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	})

	t.Run("resource value return", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test(x: @AnyResource): @AnyResource {
                return <- x
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 1)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const xIndex = 0

		assert.Equal(t,
			[]opcode.Instruction{
				// return <- x
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: xIndex},
				// There should be only one transfer
				opcode.InstructionTransfer{},
				opcode.InstructionConvert{Type: 1},
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

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// Jump to post conditions
				opcode.InstructionJump{Target: 2},

				// Post condition
				opcode.InstructionStatement{},
				opcode.InstructionTrue{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 12},

				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
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

		functions := program.Functions
		require.Len(t, functions, 1)

		const (
			tempResultIndex = iota
			aIndex
			resultIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// var a = 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: aIndex},

				// $_result = a
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: aIndex},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// Jump to post conditions
				opcode.InstructionJump{Target: 9},

				// let result $noTransfer $_result
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				// Post condition
				opcode.InstructionStatement{},
				opcode.InstructionTrue{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 22},

				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
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

		functions := program.Functions
		require.Len(t, functions, 2)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// invoke `voidReturnFunc()`
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},

				// Drop the returning void value
				opcode.InstructionDrop{},

				// Jump to post conditions
				opcode.InstructionJump{Target: 5},

				// Post condition
				opcode.InstructionStatement{},
				opcode.InstructionTrue{},
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 15},

				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 2},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 2)

	const (
		addOneIndex = iota
		xIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let addOne = fun ...
			opcode.InstructionStatement{},
			opcode.InstructionNewClosure{Function: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: addOneIndex},

			// let x = 2
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: xIndex},

			// return x + addOne(3)
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionGetLocal{Local: addOneIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 2)

	const (
		addOneIndex = iota
		xIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// fun addOne(...
			opcode.InstructionStatement{},
			opcode.InstructionNewClosure{Function: 1},
			opcode.InstructionSetLocal{Local: addOneIndex},

			// let x = 2
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: xIndex},

			// return x + addOne(3)
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionGetLocal{Local: addOneIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x + 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
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

	functions := program.Functions
	require.Len(t, functions, 2)

	const (
		// xLocalIndex is the local index of the variable `x`, which is the first local variable
		xLocalIndex = iota
		// innerLocalIndex is the local index of the variable `inner`, which is the second local variable
		innerLocalIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xLocalIndex},

			// let inner = fun(): Int { ...
			opcode.InstructionStatement{},
			opcode.InstructionNewClosure{
				Function: 1,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.InstructionTransferAndConvert{Type: 2},
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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.InstructionStatement{},
			opcode.InstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 2)

	const (
		// xLocalIndex is the local index of the variable `x`, which is the first local variable
		xLocalIndex = iota
		// innerLocalIndex is the local index of the variable `inner`, which is the second local variable
		innerLocalIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xLocalIndex},

			// fun inner(): Int { ...
			opcode.InstructionStatement{},
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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.InstructionStatement{},
			opcode.InstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
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

	functions := program.Functions
	require.Len(t, functions, 3)

	const (
		// xLocalIndex is the local index of the variable `x`, which is the first local variable
		xLocalIndex = iota
		// middleLocalIndex is the local index of the variable `middle`, which is the second local variable
		middleLocalIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: xLocalIndex},

			// fun middle(): Int { ...
			opcode.InstructionStatement{},
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
			opcode.InstructionStatement{},
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
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.InstructionStatement{},
			opcode.InstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionReturnValue{},
		},
		functions[2].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
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

	functions := program.Functions
	require.Len(t, functions, 2)

	// innerLocalIndex is the local index of the variable `inner`, which is the first local variable
	const innerLocalIndex = 0

	assert.Equal(t,
		[]opcode.Instruction{
			// fun inner() { ...
			opcode.InstructionStatement{},
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
			opcode.InstructionStatement{},
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

	functions := program.Functions
	require.Len(t, functions, 3)

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
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: aLocalIndex},

				// let b = 2
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: bLocalIndex},

				// fun middle(): Int { ...
				opcode.InstructionStatement{},
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
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: cLocalIndex},

				// let d = 4
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 3},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: dLocalIndex},

				// fun inner(): Int { ...
				opcode.InstructionStatement{},
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
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 4},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: eLocalIndex},

				// let f = 6
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 5},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: fLocalIndex},

				// return f + e + d + b + c + a
				opcode.InstructionStatement{},
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

				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},
			},
			functions[2].Code,
		)
	}

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(3),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(4),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(5),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(6),
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

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = 1
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},

				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = 1
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				// NOTE: transfer
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
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

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = /storage/foo
				opcode.InstructionStatement{},
				opcode.InstructionNewPath{
					Domain:     common.PathDomainStorage,
					Identifier: 0,
				},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},

				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: "foo",
					Kind: constant.RawString,
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

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x = /public/foo
				opcode.InstructionStatement{},
				opcode.InstructionNewPath{
					Domain:     common.PathDomainPublic,
					Identifier: 0,
				},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},

				// return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: "foo",
					Kind: constant.RawString,
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

	functions := program.Functions
	require.Len(t, functions, 2)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x = fun() {}
			opcode.InstructionStatement{},
			opcode.InstructionNewClosure{
				Function: 1,
			},
			opcode.InstructionTransferAndConvert{Type: 0},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	assert.Equal(t,
		[]opcode.Instruction{
			// let x: Int? = nil
			opcode.InstructionStatement{},
			opcode.InstructionNil{},
			opcode.InstructionTransferAndConvert{Type: 1},
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

	functions := program.Functions
	require.Len(t, functions, 2)

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
			// let x = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{},
			opcode.InstructionTransferAndConvert{Type: intTypeIndex},
			opcode.InstructionSetLocal{Local: xIndex},

			// f(x)
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransferAndConvert{Type: xParameterTypeIndex},
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

	functions := program.Functions
	require.Len(t, functions, 1)

	testFunction := functions[0]

	const (
		arrayIndex = iota
		indexIndex
		valueIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			// array[index]
			opcode.InstructionGetLocal{Local: arrayIndex},
			opcode.InstructionGetLocal{Local: indexIndex},
			opcode.InstructionTransferAndConvert{Type: 1},
			// value + value
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionGetLocal{Local: valueIndex},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetIndex{},

			// return
			opcode.InstructionReturn{},
		},
		testFunction.Code,
	)

	assert.Equal(t,
		[]bbq.PositionInfo{
			// Statement.
			// Opcodes:
			//   opcode.InstructionStatement{}
			{
				InstructionIndex: 0,
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

			// Load variable `array`.
			// Opcodes:
			//   opcode.InstructionGetLocal{Local: arrayIndex}
			{
				InstructionIndex: 1,
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
				InstructionIndex: 2,
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

			// Transfer and convert `index`.
			// Opcodes:
			//   opcode.InstructionTransferAndConvert{Type: 1}
			{
				InstructionIndex: 3,
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

			// Load variable `value`.
			// Opcodes:
			//   opcode.InstructionGetLocal{Local: valueIndex}
			{
				InstructionIndex: 4,
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
				InstructionIndex: 5,
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
				InstructionIndex: 6,
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
			//   opcode.InstructionTransferAndConvert{Type: 1}
			//   opcode.InstructionSetIndex{}
			{
				InstructionIndex: 7,
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
				InstructionIndex: 9,
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
	pos := testFunction.LineNumbers.GetSourcePosition(7)
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

func TestCompileImports(t *testing.T) {

	t.Parallel()

	t.Run("simple import", func(t *testing.T) {
		t.Parallel()

		aContract := `
            contract A {
                fun test() {}
            }
        `

		programs := CompiledPrograms{}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		aLocation := common.NewAddressLocation(nil, contractsAddress, "A")
		bLocation := common.NewAddressLocation(nil, contractsAddress, "B")

		aProgram := ParseCheckAndCompile(
			t,
			aContract,
			aLocation,
			programs,
		)

		// Should have no imports
		assert.Empty(t, aProgram.Imports)

		// Deploy a second contract.

		bContract := fmt.Sprintf(
			`
              import A from %[1]s

              contract B {
                  fun test() {
                      return A.test()
                  }
              }
            `,
			contractsAddress.HexWithPrefix(),
		)

		bProgram := ParseCheckAndCompile(
			t,
			bContract,
			bLocation,
			programs,
		)

		// Should have import for contract value `A` and the method `A.test`.
		assert.Equal(
			t,
			[]bbq.Import{
				{
					Location: aLocation,
					Name:     "A",
				},
				{
					Location: aLocation,
					Name:     "A.test",
				},
			},
			bProgram.Imports,
		)
	})

	t.Run("transitive import", func(t *testing.T) {
		t.Parallel()

		aContract := `
            contract A {
                struct Foo {
                    fun test() {}
                    fun unusedMethodOfFoo() {}
                }

                fun unusedMethodOfA() {}
            }
        `

		programs := CompiledPrograms{}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		aLocation := common.NewAddressLocation(nil, contractsAddress, "A")
		bLocation := common.NewAddressLocation(nil, contractsAddress, "B")
		cLocation := common.NewAddressLocation(nil, contractsAddress, "C")

		aProgram := ParseCheckAndCompile(
			t,
			aContract,
			aLocation,
			programs,
		)

		// Should have no imports
		assert.Empty(t, aProgram.Imports)

		// Deploy a second contract.

		bContract := fmt.Sprintf(`
            import A from %[1]s

            contract B {
                struct Bar {
                    fun getFoo(): A.Foo {
                        return A.Foo()
                    }

                    fun unusedMethodOfBar() {}
                }

                fun unusedMethodOfA() {}
            }
        `,
			contractsAddress.HexWithPrefix(),
		)

		bProgram := ParseCheckAndCompile(t, bContract, bLocation, programs)

		// Should have only one import for `A.Foo()` constructor.
		assert.Equal(
			t,
			[]bbq.Import{
				{
					Location: aLocation,
					Name:     "A.Foo",
				},
			},
			bProgram.Imports,
		)

		// Deploy third contract

		cContract := fmt.Sprintf(`
            import B from %[1]s

            contract C {
                struct Baz {
                    fun test() {
                        var foo = B.Bar().getFoo()

                        // Invokes a function of 'A.Foo', which is a transitive dependency.
                        foo.test()
                    }
                }
            }
        `,
			contractsAddress.HexWithPrefix(),
		)

		cProgram := ParseCheckAndCompile(t, cContract, cLocation, programs)

		// Should have 3 imports, including the transitive dependency `A.Foo.test`.
		// Should only have used imports.
		assert.Equal(
			t,
			[]bbq.Import{
				{
					Location: bLocation,
					Name:     "B.Bar",
				},
				{
					Location: bLocation,
					Name:     "B.Bar.getFoo",
				},
				{
					Location: aLocation,
					Name:     "A.Foo.test",
				},
			},
			cProgram.Imports,
		)
	})
}

func TestCompileOptionalChaining(t *testing.T) {

	t.Parallel()

	t.Run("field", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            struct Foo {
                var bar: Int
                init(value: Int) {
                    self.bar = value
                }
            }

            fun test(): Int? {
                let foo: Foo? = nil
                return foo?.bar
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 4)

		const (
			fooIndex = iota
			tempIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let foo: Foo? = nil
				opcode.InstructionStatement{},
				opcode.InstructionNil{},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: fooIndex},

				opcode.InstructionStatement{},

				// Store the value in a temp index for the nil check.
				opcode.InstructionGetLocal{Local: fooIndex},
				opcode.InstructionSetLocal{Local: tempIndex},

				// Nil check
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionJumpIfNil{Target: 13},

				// If `foo != nil`
				// Unwrap optional
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionUnwrap{},

				// foo.bar
				opcode.InstructionGetField{FieldName: 0, AccessedType: 2},
				opcode.InstructionJump{Target: 14},

				// If `foo == nil`
				opcode.InstructionNil{},

				// Return value
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: "bar",
					Kind: constant.RawString,
				},
			},
			program.Constants,
		)
	})

	t.Run("method", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            struct Foo {
                fun bar(): Int {
                    return 1
                }
            }

            fun test(): Int? {
                let foo: Foo? = nil
                return foo?.bar()
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 5)

		const (
			fooIndex = iota
			optionalValueTempIndex
			unwrappedValueTempIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let foo: Foo? = nil
				opcode.InstructionStatement{},
				opcode.InstructionNil{},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: fooIndex},

				opcode.InstructionStatement{},

				// Store the receiver in a temp index for the nil check.
				opcode.InstructionGetLocal{Local: fooIndex},
				opcode.InstructionSetLocal{Local: optionalValueTempIndex},

				// Nil check
				opcode.InstructionGetLocal{Local: optionalValueTempIndex},
				opcode.InstructionJumpIfNil{Target: 15},

				// If `foo != nil`
				// Unwrap the optional. (Loads receiver)
				opcode.InstructionGetLocal{Local: optionalValueTempIndex},
				opcode.InstructionUnwrap{},

				// Load `Foo.bar` function
				opcode.InstructionGetMethod{Method: 4},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionWrap{},
				opcode.InstructionJump{Target: 16},

				// If `foo == nil`
				opcode.InstructionNil{},

				// Return value
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)
	})
}

func TestCompileSecondValueAssignment(t *testing.T) {

	t.Parallel()

	t.Run("in variable declaration", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let x: @R <- create R()
                var y: @R? <- create R()

                let z: @R? <- y <- x

                destroy y
                destroy z
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 4)

		const (
			xIndex = iota
			yIndex
			zIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x: @R <- create R()
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: xIndex},

				// var y: @R? <- create R()
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetLocal{Local: yIndex},

				opcode.InstructionStatement{},

				// Load `y` onto the stack.
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionTransferAndConvert{Type: 2},

				// Second value assignment.
				// y <- x
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetLocal{Local: yIndex},

				// Transfer and store the loaded y-value above, to z.
				// z <- y
				opcode.InstructionSetLocal{Local: zIndex},

				// destroy y
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionDestroy{},

				// destroy z
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: zIndex},
				opcode.InstructionDestroy{},

				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	})

	t.Run("index expr in variable declaration", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let x: @R <- create R()
                var y <- {"r" : <- create R()}

                let z: @R? <- y["r"] <- x

                destroy y
                destroy z
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 4)

		const (
			xIndex = iota
			yIndex
			tempYIndex
			tempIndexingValueIndex
			zIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x: @R <- create R()
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: xIndex},

				// var y <- {"r" : <- create R()}
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{TypeArgs: []uint16(nil), ArgCount: 0},
				opcode.InstructionTransfer{},
				opcode.InstructionConvert{Type: 1},
				opcode.InstructionNewDictionary{Type: 2, Size: 1, IsResource: true},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetLocal{Local: yIndex},

				opcode.InstructionStatement{},

				// <- y["r"]

				// Evaluate `y` and store in a temp local.
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionSetLocal{Local: tempYIndex},

				// evaluate "r", and store in a temp local.
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionSetLocal{Local: tempIndexingValueIndex},

				// Evaluate the index expression, `y["r"]`, using temp locals.
				opcode.InstructionGetLocal{Local: tempYIndex},
				opcode.InstructionGetLocal{Local: tempIndexingValueIndex},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionRemoveIndex{},
				opcode.InstructionTransferAndConvert{Type: 4},

				// Second value assignment.
				// y["r"] <- x
				// `y` and "r" must be loaded from temp locals.
				opcode.InstructionGetLocal{Local: tempYIndex},
				opcode.InstructionGetLocal{Local: tempIndexingValueIndex},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransferAndConvert{Type: 4},
				opcode.InstructionSetIndex{},

				// Store the transferred y-value above (already on stack), to z.
				// z <- y["r"]
				opcode.InstructionSetLocal{Local: zIndex},

				// destroy y
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionDestroy{},

				// destroy z
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: zIndex},
				opcode.InstructionDestroy{},

				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	})

	t.Run("member expr in variable declaration", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            resource Foo {
                var bar: @Bar
                init() {
                    self.bar <- create Bar()
                }
            }

            resource Bar {}

            fun test() {
                let x: @Bar <- create Bar()
                var y <- create Foo()

                let z <- y.bar <- x

                destroy y
                destroy z
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 7)

		const (
			xIndex = iota
			yIndex
			tempYIndex
			zIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x: @R <- create R()
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 4},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: xIndex},

				// var y <- {"r" : <- create R()}
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetLocal{Local: yIndex},

				opcode.InstructionStatement{},

				// <- y.bar

				// Evaluate `y` and store in a temp local.
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionSetLocal{Local: tempYIndex},

				// Evaluate the member access, `y.bar`, using temp local.
				opcode.InstructionGetLocal{Local: tempYIndex},
				opcode.InstructionRemoveField{FieldName: 0},
				opcode.InstructionTransferAndConvert{Type: 1},

				// Second value assignment.
				//  `y.bar <- x`
				// `y` must be loaded from the temp local.
				opcode.InstructionGetLocal{Local: tempYIndex},
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetField{FieldName: 0, AccessedType: 2},

				// Store the transferred y-value above (already on stack), to z.
				// z <- y.bar
				opcode.InstructionSetLocal{Local: zIndex},

				// destroy y
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionDestroy{},

				// destroy z
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: zIndex},
				opcode.InstructionDestroy{},

				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	})

	t.Run("in if statement", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let x: @R <- create R()
                var y: @R? <- create R()

                if let z <- y <- x {
                    let res: @R <- z
                    destroy res
                }

                destroy y
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 4)

		const (
			xIndex = iota
			yIndex
			tempIndex
			zIndex
			resIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let x: @R <- create R()
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: xIndex},

				// var y: @R? <- create R()
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionInvoke{ArgCount: 0},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetLocal{Local: yIndex},

				opcode.InstructionStatement{},

				// store y in temp index for nil check
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionTransferAndConvert{Type: 2},

				// Second value assignment. Store `x` in `y`.
				// y <- x
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetLocal{Local: yIndex},

				// Store the previously loaded `y`s old value on the temp local.
				opcode.InstructionSetLocal{Local: tempIndex},

				// nil check on temp y.
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionJumpIfNil{Target: 29},

				// If not-nil, transfer the temp `y` and store in `z` (i.e: y <- y)
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionUnwrap{},
				opcode.InstructionSetLocal{Local: zIndex},

				// let res: @R <- z
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: zIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: resIndex},

				// destroy res
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: resIndex},
				opcode.InstructionDestroy{},

				// destroy y
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: yIndex},
				opcode.InstructionDestroy{},

				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)
	})
}

func TestCompileEnum(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        enum Test: UInt8 {
            case a
            case b
            case c
        }

        fun test(): UInt8 {
            return Test.b.rawValue
        }

        fun test2(rawValue: UInt8) {
            Test(rawValue: rawValue)
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	variables := program.Variables
	require.Len(t, variables, 3)

	const (
		testAVarIndex = iota
		testBVarIndex
		testCVarIndex
	)

	functions := program.Functions
	require.Len(t, functions, 6)

	const (
		testLookupFuncIndex = iota
		testFuncIndex
		test2FuncIndex
		testConstructorFuncIndex
		// Next two indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
	)

	const (
		testAGlobalIndex = iota
		testBGlobalIndex
		testCGlobalIndex
		testLookupGlobalIndex
		testGlobalIndex
		test2GlobalIndex
		testConstructorGlobalIndex
	)

	{
		const parameterCount = 1

		// rawValueIndex is the index of the parameter `rawValue`, which is the first parameter
		const rawValueIndex = iota

		// localsOffset is the offset of the first local variable.
		// Initializers do not have a $_result variable
		const localsOffset = parameterCount

		const (
			// selfIndex is the index of the local variable `self`, which is the first local variable
			selfIndex = localsOffset + iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let self = Test()
				opcode.InstructionNewComposite{
					Kind: common.CompositeKindEnum,
					Type: 3,
				},
				opcode.InstructionSetLocal{Local: selfIndex},

				// self.rawValue = rawValue
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetLocal{Local: rawValueIndex},
				opcode.InstructionTransferAndConvert{Type: 4},
				opcode.InstructionSetField{FieldName: 3, AccessedType: 3},

				// return self
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionReturnValue{},
			},
			functions[testConstructorFuncIndex].Code,
		)
	}

	{
		const (
			// rawValueIndex is the index of the parameter `rawValue`, which is the first parameter
			rawValueIndex = iota
			// tempIndex is the index of the temporary variable used for switch
			tempIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// let temp = rawValue
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: rawValueIndex},
				opcode.InstructionSetLocal{Local: tempIndex},

				// switch temp

				// case 1:
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionEqual{},
				opcode.InstructionJumpIfFalse{Target: 12},

				// return Test.a
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: testAGlobalIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},
				opcode.InstructionJump{Target: 34},

				// case 2:
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionEqual{},
				opcode.InstructionJumpIfFalse{Target: 21},

				// return Test.b
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: testBGlobalIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},
				opcode.InstructionJump{Target: 34},

				// case 3:
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionEqual{},
				opcode.InstructionJumpIfFalse{Target: 30},

				// return Test.c
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: testCGlobalIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},
				opcode.InstructionJump{Target: 34},

				// default:
				// return nil
				opcode.InstructionStatement{},
				opcode.InstructionNil{},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionReturnValue{},

				// return
				opcode.InstructionReturn{},
			},
			functions[testLookupFuncIndex].Code,
		)
	}

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: testBGlobalIndex},
			opcode.InstructionGetField{FieldName: 3, AccessedType: 3},
			opcode.InstructionTransferAndConvert{Type: 4},
			opcode.InstructionReturnValue{},
		},
		functions[testFuncIndex].Code,
	)

	{
		// rawValueIndex is the index of the parameter `rawValue`, which is the first parameter
		const rawValueIndex = iota

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: testLookupGlobalIndex},
				opcode.InstructionGetLocal{Local: rawValueIndex},
				opcode.InstructionTransferAndConvert{Type: 4},
				opcode.InstructionInvoke{ArgCount: 1},
				opcode.InstructionDrop{},
				opcode.InstructionReturn{},
			},
			functions[test2FuncIndex].Code,
		)
	}

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetGlobal{Global: testConstructorGlobalIndex},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionReturnValue{},
		},
		variables[testAVarIndex].Getter.Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetGlobal{Global: testConstructorGlobalIndex},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionReturnValue{},
		},
		variables[testBVarIndex].Getter.Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionGetGlobal{Global: testConstructorGlobalIndex},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionReturnValue{},
		},
		variables[testCVarIndex].Getter.Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredUInt8Value(0),
				Kind: constant.UInt8,
			},
			{
				Data: interpreter.NewUnmeteredUInt8Value(1),
				Kind: constant.UInt8,
			},
			{
				Data: interpreter.NewUnmeteredUInt8Value(2),
				Kind: constant.UInt8,
			},
			{
				Data: "rawValue",
				Kind: constant.RawString,
			},
		},
		program.Constants,
	)
}

func TestCompileOptionalArgument(t *testing.T) {
	t.Parallel()

	t.Run("assert function", func(t *testing.T) {
		t.Parallel()

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.VMAssertFunction)

		checker, err := ParseAndCheckWithOptions(t,
			`
            fun test() {
                assert(true, message: "hello")
                assert(false)
            }
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
		require.NoError(t, err)

		config := &compiler.Config{
			BuiltinGlobalsProvider: func(_ common.Location) *activations.Activation[compiler.GlobalImport] {
				activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
				activation.Set(
					stdlib.AssertFunctionName,
					compiler.NewGlobalImport(stdlib.AssertFunctionName),
				)
				return activation
			},
		}

		comp := compiler.NewInstructionCompilerWithConfig(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
			config,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// assert(true, message: "hello")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},
				opcode.InstructionTrue{},
				opcode.InstructionTransferAndConvert{Type: 0x1},
				opcode.InstructionGetConstant{Constant: 0x0},
				opcode.InstructionTransferAndConvert{Type: 0x2},
				opcode.InstructionInvoke{TypeArgs: []uint16(nil), ArgCount: 0x2},
				opcode.InstructionDrop{},

				// assert(false)
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 0x1},
				opcode.InstructionFalse{},
				opcode.InstructionTransferAndConvert{Type: 0x1},
				opcode.InstructionInvoke{TypeArgs: []uint16(nil), ArgCount: 0x1},
				opcode.InstructionDrop{},
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredStringValue("hello"),
					Kind: constant.String,
				},
			},
			program.Constants,
		)
	})

	t.Run("account add optional args", func(t *testing.T) {
		t.Parallel()

		aContract := `
            contract A {
                fun test() {
                    self.account.contracts.add(
                        name: "Foo",
                        code: " contract Foo { let message: String\n init(message:String) {self.message = message}\nfun test(): String {return self.message}}".utf8,
                        message: "Optional arg",
                    )
                }
            }
        `

		programs := CompiledPrograms{}

		contractsAddress := common.MustBytesToAddress([]byte{0x1})

		aLocation := common.NewAddressLocation(nil, contractsAddress, "A")

		program := ParseCheckAndCompile(
			t,
			aContract,
			aLocation,
			programs,
		)

		functions := program.Functions
		require.Len(t, functions, 4)

		const (
			_ = iota
			accountFieldNameIndex
			contractsFieldNameIndex
			contractNameIndex
			contractCodeIndex
			utf8FieldNameIndex
			optionalArgIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// Load receiver `self.account.contracts`.
				opcode.InstructionGetLocal{Local: 0},
				opcode.InstructionGetField{
					FieldName:    accountFieldNameIndex,
					AccessedType: 1,
				},
				opcode.InstructionGetField{
					FieldName:    contractsFieldNameIndex,
					AccessedType: 4,
				},
				opcode.InstructionNewRef{Type: 5, IsImplicit: true},

				// Load function value `add()`
				opcode.InstructionGetMethod{Method: 5},

				// Load arguments.

				// Name: "Foo",
				opcode.InstructionGetConstant{Constant: contractNameIndex},
				opcode.InstructionTransferAndConvert{Type: 6},

				// Contract code
				opcode.InstructionGetConstant{Constant: contractCodeIndex},
				opcode.InstructionGetField{
					FieldName:    utf8FieldNameIndex,
					AccessedType: 6,
				},
				opcode.InstructionTransferAndConvert{Type: 7},

				// Message: "Optional arg"
				opcode.InstructionGetConstant{Constant: optionalArgIndex},
				opcode.InstructionTransfer{},

				opcode.InstructionInvoke{ArgCount: 3},
				opcode.InstructionDrop{},

				opcode.InstructionReturn{}},
			functions[3].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
					Kind: constant.Address,
				},
				{
					Data: "account",
					Kind: constant.RawString,
				},
				{
					Data: "contracts",
					Kind: constant.RawString,
				},
				{
					Data: interpreter.NewUnmeteredStringValue("Foo"),
					Kind: constant.String,
				},
				{
					Data: interpreter.NewUnmeteredStringValue(" contract Foo { let message: String\n init(message:String) {self.message = message}\nfun test(): String {return self.message}}"),
					Kind: constant.String,
				},
				{
					Data: "utf8",
					Kind: constant.RawString,
				},
				{
					Data: interpreter.NewUnmeteredStringValue("Optional arg"),
					Kind: constant.String,
				},
			},
			program.Constants,
		)
	})

}

func TestCompileContract(t *testing.T) {
	t.Parallel()

	aContract := `
        contract A {
            let x: Int

            init() {
                self.x = 1
            }
        }
    `

	programs := CompiledPrograms{}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	aLocation := common.NewAddressLocation(nil, contractsAddress, "A")

	program := ParseCheckAndCompile(
		t,
		aContract,
		aLocation,
		programs,
	)

	functions := program.Functions
	require.Len(t, functions, 3)

	const (
		addressIndex = iota
		oneIndex
		xFieldNameIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let self = A()
			opcode.InstructionNewCompositeAt{
				Kind:    common.CompositeKindContract,
				Type:    1,
				Address: addressIndex,
			},
			opcode.InstructionDup{},
			opcode.InstructionSetGlobal{Global: 0},
			opcode.InstructionSetLocal{Local: 0},

			// self.x = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: oneIndex},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetField{
				FieldName:    xFieldNameIndex,
				AccessedType: 1,
			},

			// return self
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				Kind: constant.Address,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: "x",
				Kind: constant.RawString,
			},
		},
		program.Constants,
	)
}

func TestCompileSwapIdentifiers(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            var x = 1
            var y = 2
            x <-> y
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		xIndex = iota
		yIndex
		tempIndex1
		tempIndex2
		tempIndex3
		tempIndex4
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// var x = 1
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 0x1},
			opcode.InstructionSetLocal{Local: xIndex},

			// var y = 2
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 0x1},
			opcode.InstructionSetLocal{Local: yIndex},

			// x <-> y
			opcode.InstructionStatement{},

			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{Local: tempIndex1},

			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionSetLocal{Local: tempIndex2},

			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionTransferAndConvert{Type: 0x1},
			opcode.InstructionSetLocal{Local: tempIndex3},

			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionTransferAndConvert{Type: 0x1},
			opcode.InstructionSetLocal{Local: tempIndex4},

			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionSetLocal{Local: xIndex},

			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionSetLocal{Local: yIndex},

			// Return
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileSwapMembers(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        struct S {
            var x: Int
            var y: Int

            init() {
                self.x = 1
                self.y = 2
            }
        }

        fun test() {
            let s = S()
            s.x <-> s.y
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 4)

	const (
		sIndex = iota
		tempIndex1
		tempIndex2
		tempIndex3
		tempIndex4
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let s = S()
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 1},
			opcode.InstructionInvoke{},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: sIndex},

			// s.x <-> s.y
			opcode.InstructionStatement{},

			opcode.InstructionGetLocal{Local: sIndex},
			opcode.InstructionSetLocal{Local: tempIndex1},

			opcode.InstructionGetLocal{Local: sIndex},
			opcode.InstructionSetLocal{Local: tempIndex2},

			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetField{FieldName: 0, AccessedType: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: tempIndex3},

			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionGetField{FieldName: 1, AccessedType: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: tempIndex4},

			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionSetField{FieldName: 0, AccessedType: 1},

			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionSetField{FieldName: 1, AccessedType: 1},

			// Return
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: "x",
				Kind: constant.RawString,
			},
			{
				Data: "y",
				Kind: constant.RawString,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(2),
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileSwapIndex(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
            let chars = ["a", "b"]
            chars[0] <-> chars[1]
        }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		charsIndex = iota
		tempIndex1
		tempIndex2
		tempIndex3
		tempIndex4
		tempIndex5
		tempIndex6
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// let chars = ["a", "b"]
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionNewArray{Type: 1, Size: 2},
			opcode.InstructionTransferAndConvert{Type: 1},
			opcode.InstructionSetLocal{Local: charsIndex},

			// chars[0] <-> chars[1]
			opcode.InstructionStatement{},

			opcode.InstructionGetLocal{Local: charsIndex},
			opcode.InstructionSetLocal{Local: tempIndex1},

			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionSetLocal{Local: tempIndex2},

			opcode.InstructionGetLocal{Local: charsIndex},
			opcode.InstructionSetLocal{Local: tempIndex3},

			opcode.InstructionGetConstant{Constant: 3},
			opcode.InstructionSetLocal{Local: tempIndex4},

			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionGetIndex{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: tempIndex5},

			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionGetIndex{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: tempIndex6},

			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionGetLocal{Local: tempIndex6},
			opcode.InstructionSetIndex{},

			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionGetLocal{Local: tempIndex5},
			opcode.InstructionSetIndex{},

			// Return
			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredStringValue("a"),
				Kind: constant.String,
			},
			{
				Data: interpreter.NewUnmeteredStringValue("b"),
				Kind: constant.String,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(0),
				Kind: constant.Int,
			},
			{
				Data: interpreter.NewUnmeteredIntValueFromInt64(1),
				Kind: constant.Int,
			},
		},
		program.Constants,
	)
}

func TestCompileStringTemplate(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test() {
                let str = "2+2=\(2+2)"
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// let str = "2+2=\(2+2)"
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionAdd{},
				opcode.InstructionTemplateString{ExprSize: 1},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},

				// Return
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredStringValue("2+2="),
					Kind: constant.String,
				},

				{
					Data: interpreter.NewUnmeteredStringValue(""),
					Kind: constant.String,
				},
				{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			program.Constants,
		)
	})

	t.Run("multiple exprs", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test() {
                let a = "A"
                let b = "B"
                let c = 4
                let str = "\(a) + \(b) = \(c)"
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 1)

		assert.Equal(t,
			[]opcode.Instruction{
				// let a = "A"
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				// let b = "B"
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 1},
				// let c = 4
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionSetLocal{Local: 2},
				// let str = "\(a) + \(b) = \(c)"
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 3},
				opcode.InstructionGetConstant{Constant: 4},
				opcode.InstructionGetConstant{Constant: 5},
				opcode.InstructionGetConstant{Constant: 3},
				opcode.InstructionGetLocal{Local: 0},
				opcode.InstructionGetLocal{Local: 1},
				opcode.InstructionGetLocal{Local: 2},
				opcode.InstructionTemplateString{ExprSize: 3},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 3},
				opcode.InstructionReturn{},
			},
			functions[0].Code,
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredStringValue("A"),
					Kind: constant.String,
				},
				{
					Data: interpreter.NewUnmeteredStringValue("B"),
					Kind: constant.String,
				},
				{
					Data: interpreter.NewUnmeteredIntValueFromInt64(4),
					Kind: constant.Int,
				},
				{
					Data: interpreter.NewUnmeteredStringValue(""),
					Kind: constant.String,
				},
				{
					Data: interpreter.NewUnmeteredStringValue(" + "),
					Kind: constant.String,
				},
				{
					Data: interpreter.NewUnmeteredStringValue(" = "),
					Kind: constant.String,
				},
			},
			program.Constants,
		)
	})
}

func TestForStatementCapturing(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        fun test() {
           for i, x in [1, 2, 3] {
               let f = fun (): Int {
                   return x + i
               }
               if x > 0 {
                   continue
               }
               f()
           }
       }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 2)

	const (
		iterIndex = iota
		iIndex
		xIndex
		fIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{

			// for i, x in [1, 2, 3]
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionNewArray{
				Type:       1,
				Size:       3,
				IsResource: false,
			},

			// get iterator
			opcode.InstructionIterator{},
			opcode.InstructionSetLocal{Local: iterIndex},

			// set i = -1
			opcode.InstructionGetConstant{Constant: 3},
			opcode.InstructionSetLocal{Local: iIndex},

			// check if iterator has more elements
			opcode.InstructionGetLocal{Local: iterIndex},
			opcode.InstructionIteratorHasNext{},
			opcode.InstructionJumpIfFalse{Target: 44},

			opcode.InstructionLoop{},
			// increment i
			opcode.InstructionGetLocal{Local: iIndex},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionAdd{},
			opcode.InstructionSetLocal{Local: iIndex},

			// get next iterator element
			opcode.InstructionGetLocal{Local: iterIndex},
			opcode.InstructionIteratorNext{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: xIndex},

			// let f = fun() ...
			opcode.InstructionStatement{},
			opcode.InstructionNewClosure{
				Function: 1,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xIndex,
						IsLocal:     true,
					},
					{
						TargetIndex: iIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.InstructionTransferAndConvert{Type: 3},
			opcode.InstructionSetLocal{Local: fIndex},

			// if x > 0
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionGetConstant{Constant: 4},
			opcode.InstructionGreater{},
			opcode.InstructionJumpIfFalse{Target: 37},

			// continue
			opcode.InstructionStatement{},
			opcode.InstructionCloseUpvalue{Local: iIndex},
			opcode.InstructionCloseUpvalue{Local: xIndex},
			opcode.InstructionJump{Target: 12},

			// f()
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 3},
			opcode.InstructionInvoke{ArgCount: 0},
			opcode.InstructionDrop{},

			// next iteration
			opcode.InstructionCloseUpvalue{Local: iIndex},
			opcode.InstructionCloseUpvalue{Local: xIndex},
			opcode.InstructionJump{Target: 12},

			// end of for loop
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionIteratorEnd{},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// return x + i
			opcode.InstructionStatement{},
			opcode.InstructionGetUpvalue{Upvalue: 0},
			opcode.InstructionGetUpvalue{Upvalue: 1},
			opcode.InstructionAdd{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[1].Code,
	)
}

func TestCompileFunctionExpressionConditions(t *testing.T) {

	t.Parallel()

	t.Run("pre condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
        fun test() {
            var foo = fun(x: Int): Int {
                pre { x > 0 }
                return 5
            }
        }
    `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 2)

		const (
			testFunctionIndex = iota
			anonymousFunctionIndex
		)

		// `test` function
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},
				opcode.InstructionNewClosure{Function: anonymousFunctionIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				opcode.InstructionReturn{},
			},
			functions[testFunctionIndex].Code,
		)

		// Function expression. Would be equivalent to:
		// fun foo(x: Int): Int {
		//    if !(x > 0) {
		//        $failPreCondition("")
		//    }
		//    return 5
		// }

		const (
			xIndex = iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 12},

				// $failPreCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 1}, // error message
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionReturnValue{},
			},
			functions[anonymousFunctionIndex].Code,
		)
	})

	t.Run("post condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
        fun test() {
            var foo = fun(x: Int): Int {
                post { x > 0 }
                return 5
            }
        }
    `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 2)

		const (
			testFunctionIndex = iota
			anonymousFunctionIndex
		)

		// `test` function
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},
				opcode.InstructionNewClosure{Function: anonymousFunctionIndex},
				opcode.InstructionTransferAndConvert{Type: 1},
				opcode.InstructionSetLocal{Local: 0},
				opcode.InstructionReturn{},
			},
			functions[testFunctionIndex].Code,
		)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const (
			xIndex = iota
			tempResultIndex
			resultIndex
		)

		// Would be equivalent to:
		// fun test(x: Int): Int {
		//    $_result = 5
		//    let result $noTransfer $_result
		//    if !(x > 0) {
		//        $failPostCondition("")
		//    }
		//    return $_result
		// }
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// $_result = 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 5},

				// let result $noTransfer $_result
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				opcode.InstructionStatement{},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 20},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 2}, // error message
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionReturnValue{},
			},
			functions[anonymousFunctionIndex].Code,
		)
	})
}

func TestCompileInnerFunctionConditions(t *testing.T) {

	t.Parallel()

	t.Run("pre condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test() {
                fun foo(x: Int): Int {
                    pre { x > 0 }
                    return 5
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 2)

		const (
			testFunctionIndex = iota
			anonymousFunctionIndex
		)

		// `test` function
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},
				opcode.InstructionNewClosure{Function: anonymousFunctionIndex},
				opcode.InstructionSetLocal{Local: 0},
				opcode.InstructionReturn{},
			},
			functions[testFunctionIndex].Code,
		)

		// Function expression. Would be equivalent to:
		// fun foo(x: Int): Int {
		//    if !(x > 0) {
		//        $failPreCondition("")
		//    }
		//    return 5
		// }

		const (
			xIndex = iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 12},

				// $failPreCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 1}, // error message
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionReturnValue{},
			},
			functions[anonymousFunctionIndex].Code,
		)
	})

	t.Run("post condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            fun test() {
                fun foo(x: Int): Int {
                    post { x > 0 }
                    return 5
                }
            }
        `)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 2)

		const (
			testFunctionIndex = iota
			anonymousFunctionIndex
		)

		// `test` function
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},
				opcode.InstructionNewClosure{Function: anonymousFunctionIndex},
				opcode.InstructionSetLocal{Local: 0},
				opcode.InstructionReturn{},
			},
			functions[testFunctionIndex].Code,
		)

		// xIndex is the index of the parameter `x`, which is the first parameter
		const (
			xIndex = iota
			tempResultIndex
			resultIndex
		)

		// Would be equivalent to:
		// fun test(x: Int): Int {
		//    $_result = 5
		//    let result $noTransfer $_result
		//    if !(x > 0) {
		//        $failPostCondition("")
		//    }
		//    return $_result
		// }
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// $_result = 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 5},

				// let result $noTransfer $_result
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.InstructionSetLocal{Local: resultIndex},

				opcode.InstructionStatement{},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 20},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 2}, // error message
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionReturnValue{},
			},
			functions[anonymousFunctionIndex].Code,
		)
	})

	t.Run("function nested inside statements", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
        fun test() {
            if true {
                if true {
                    fun foo(x: Int): Int {
                        pre { x > 0 }
                        return 5
                    }
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

		functions := program.Functions
		require.Len(t, functions, 2)

		const (
			testFunctionIndex = iota
			anonymousFunctionIndex
		)

		// `test` function
		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},
				opcode.InstructionTrue{},
				opcode.InstructionJumpIfFalse{Target: 9},
				opcode.InstructionStatement{},
				opcode.InstructionTrue{},
				opcode.InstructionJumpIfFalse{Target: 9},
				opcode.InstructionStatement{},
				opcode.InstructionNewClosure{Function: anonymousFunctionIndex},
				opcode.InstructionSetLocal{Local: 0},
				opcode.InstructionReturn{},
			},
			functions[testFunctionIndex].Code,
		)

		// Function expression. Would be equivalent to:
		// fun foo(x: Int): Int {
		//    if !(x > 0) {
		//        $failPreCondition("")
		//    }
		//    return 5
		// }

		const (
			xIndex = iota
		)

		assert.Equal(t,
			[]opcode.Instruction{
				opcode.InstructionStatement{},

				// x > 0
				opcode.InstructionGetLocal{Local: xIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionGreater{},

				// if !<condition>
				opcode.InstructionNot{},
				opcode.InstructionJumpIfFalse{Target: 12},

				// $failPreCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 1}, // error message
				opcode.InstructionTransferAndConvert{Type: 2},
				opcode.InstructionInvoke{ArgCount: 1},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{Type: 3},
				opcode.InstructionReturnValue{},
			},
			functions[anonymousFunctionIndex].Code,
		)
	})

}

func TestCompileImportEnumCase(t *testing.T) {

	t.Parallel()

	aContract := `
        contract A {
            enum E: UInt8 {
                case X
            }
        }
    `

	programs := CompiledPrograms{}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	aLocation := common.NewAddressLocation(nil, contractsAddress, "A")
	bLocation := common.NewAddressLocation(nil, contractsAddress, "B")

	aProgram := ParseCheckAndCompile(
		t,
		aContract,
		aLocation,
		programs,
	)

	// Should have no imports
	assert.Empty(t, aProgram.Imports)

	// Deploy a second contract.

	bContract := fmt.Sprintf(
		`
          import A from %[1]s

          contract B {
              fun test(): A.E {
                  return A.E.X
              }
          }
        `,
		contractsAddress.HexWithPrefix(),
	)

	bProgram := ParseCheckAndCompile(
		t,
		bContract,
		bLocation,
		programs,
	)

	// Should have import for the enum case `A.E.X`.
	assert.Equal(
		t,
		[]bbq.Import{
			{
				Location: aLocation,
				Name:     "A.E.X",
			},
		},
		bProgram.Imports,
	)
}

func TestDynamicMethodInvocationViaOptionalChaining(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      struct interface SI {
          fun answer(): Int
      }

      fun answer(_ si: {SI}?): Int? {
          return si?.answer()
      }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 3)

	const (
		siIndex = iota
		tempIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: siIndex},
			opcode.InstructionSetLocal{Local: tempIndex},
			opcode.InstructionGetLocal{Local: tempIndex},
			opcode.InstructionJumpIfNil{Target: 11},
			opcode.InstructionGetLocal{Local: tempIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionGetField{
				FieldName:    0,
				AccessedType: 1,
			},
			opcode.InstructionInvoke{
				ArgCount: 0,
			},
			opcode.InstructionWrap{},
			opcode.InstructionJump{Target: 12},
			opcode.InstructionNil{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionReturnValue{},
		},
		functions[0].Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: "answer",
				Kind: constant.RawString,
			},
		},
		program.Constants,
	)

}

func TestCompileInjectedContract(t *testing.T) {

	t.Parallel()

	cType := &sema.FunctionType{
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "n",
				TypeAnnotation: sema.IntTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
	}

	bType := &sema.CompositeType{
		Identifier: "B",
		Kind:       common.CompositeKindContract,
	}

	bType.Members = sema.MembersAsMap([]*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			bType,
			"c",
			cType,
			"",
		),
		sema.NewUnmeteredPublicConstantFieldMember(
			bType,
			"d",
			sema.IntType,
			"",
		),
	})

	bStaticType := interpreter.ConvertSemaCompositeTypeToStaticCompositeType(nil, bType)

	bValue := interpreter.NewSimpleCompositeValue(
		nil,
		bType.ID(),
		bStaticType,
		[]string{"d"},
		map[string]interpreter.Value{
			"d": interpreter.NewUnmeteredIntValueFromInt64(1),
		},
		nil,
		nil,
		nil,
		nil,
	)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.StandardLibraryValue{
		Name:  bType.Identifier,
		Type:  bType,
		Value: bValue,
		Kind:  common.DeclarationKindContract,
	})

	checker, err := ParseAndCheckWithOptions(t,
		`
          contract A {
              fun test(): Int {
                  return B.c(B.d)
              }
          }
        `,
		ParseAndCheckOptions{
			Location: TestLocation,
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(location common.Location) *sema.VariableActivation {
					assert.Equal(t, TestLocation, location)
					return baseValueActivation
				},
			},
		},
	)
	require.NoError(t, err)

	config := &compiler.Config{
		BuiltinGlobalsProvider: func(location common.Location) *activations.Activation[compiler.GlobalImport] {
			assert.Equal(t, TestLocation, location)
			activation := activations.NewActivation(nil, compiler.DefaultBuiltinGlobals())
			activation.Set(
				"B",
				compiler.NewGlobalImport("B"),
			)
			activation.Set(
				"B.c",
				compiler.NewGlobalImport("B.c"),
			)
			return activation
		},
	}

	comp := compiler.NewInstructionCompilerWithConfig(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		config,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 4)

	aTestFunction := functions[3]

	require.Equal(t, aTestFunction.Name, "A.test")

	assert.Equal(t,
		[]opcode.Instruction{
			// return B.c(B.d)
			opcode.InstructionStatement{},
			// B.c(...)
			opcode.InstructionGetGlobal{Global: 5},
			opcode.InstructionGetMethod{Method: 6},
			// B.d
			opcode.InstructionGetGlobal{Global: 5},
			opcode.InstructionGetField{
				FieldName:    0,
				AccessedType: 5,
			},
			opcode.InstructionTransferAndConvert{Type: 6},
			opcode.InstructionInvoke{ArgCount: 1},
			opcode.InstructionTransferAndConvert{Type: 6},
			// return
			opcode.InstructionReturnValue{},
		},
		aTestFunction.Code,
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: "d",
				Kind: constant.RawString,
			},
		},
		program.Constants,
	)

	assert.Equal(t,
		[]bbq.Import{
			{
				Name: "B",
			},
			{
				Name: "B.c",
			},
		},
		program.Imports,
	)
}

func TestNestedLoops(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
         fun test() {
             for x in [1, 2] {
                 for y in [1] {}
             }
         }
    `)
	require.NoError(t, err)

	comp := compiler.NewInstructionCompiler(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
	)
	program := comp.Compile()

	functions := program.Functions
	require.Len(t, functions, 1)

	const (
		outerIterIndex = iota
		xIndex
		innerIterIndex
		yIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// for x in [1, 2]
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionNewArray{
				Type: 1,
				Size: 2,
			},
			opcode.InstructionIterator{},
			opcode.InstructionSetLocal{Local: outerIterIndex},
			opcode.InstructionGetLocal{Local: outerIterIndex},
			opcode.InstructionIteratorHasNext{},
			opcode.InstructionJumpIfFalse{Target: 34},

			opcode.InstructionLoop{},
			opcode.InstructionGetLocal{Local: outerIterIndex},
			opcode.InstructionIteratorNext{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: xIndex},

			// for y in [1]
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionNewArray{
				Type: 1,
				Size: 1,
			},
			opcode.InstructionIterator{},
			opcode.InstructionSetLocal{Local: innerIterIndex},
			opcode.InstructionGetLocal{Local: innerIterIndex},
			opcode.InstructionIteratorHasNext{},
			opcode.InstructionJumpIfFalse{Target: 31},

			opcode.InstructionLoop{},
			opcode.InstructionGetLocal{Local: innerIterIndex},
			opcode.InstructionIteratorNext{},
			opcode.InstructionTransferAndConvert{Type: 2},
			opcode.InstructionSetLocal{Local: yIndex},

			opcode.InstructionJump{Target: 22},
			opcode.InstructionGetLocal{Local: innerIterIndex},
			opcode.InstructionIteratorEnd{},

			opcode.InstructionJump{Target: 8},
			opcode.InstructionGetLocal{Local: outerIterIndex},
			opcode.InstructionIteratorEnd{},

			opcode.InstructionReturn{},
		},
		functions[0].Code,
	)
}

func TestCompileInheritedDefaultDestroyEvent(t *testing.T) {

	t.Parallel()

	// Deploy contract interface

	barContract := `
        contract interface Bar {
            resource interface XYZ {
                var x: Int
                event ResourceDestroyed(x: Int = self.x)
            }
        }
    `

	programs := CompiledPrograms{}

	contractsAddress := common.MustBytesToAddress([]byte{0x1})

	barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
	fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

	barProgram := ParseCheckAndCompile(
		t,
		barContract,
		barLocation,
		programs,
	)

	functions := barProgram.Functions
	require.Len(t, functions, 7)

	defaultDestroyEventConstructor := functions[4]
	require.Equal(t, "Bar.XYZ.ResourceDestroyed", defaultDestroyEventConstructor.Name)

	const (
		xIndex = iota
		selfIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// Create a `Bar.XYZ.ResourceDestroyed` event value.
			opcode.InstructionNewComposite{Kind: 4, Type: 3},
			opcode.InstructionSetLocal{Local: selfIndex},
			opcode.InstructionStatement{},

			// Set the parameter to the field.
			//  `self.x = x`
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransferAndConvert{Type: 4},
			opcode.InstructionSetField{
				FieldName:    0,
				AccessedType: 3,
			},

			// Return the constructed event value.
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionReturnValue{},
		},
		defaultDestroyEventConstructor.Code,
	)

	// Deploy contract implementation

	fooContract := fmt.Sprintf(
		`
        import Bar from %[1]s

        contract Foo {

            resource ABC: Bar.XYZ {
                var x: Int

                event ResourceDestroyed(x: Int = self.x)

                init() {
                    self.x = 6
                }
            }

            fun createABC(): @ABC {
                return <- create ABC()
            }
        }
            `,
		contractsAddress.HexWithPrefix(),
	)

	fooProgram := ParseCheckAndCompile(t, fooContract, fooLocation, programs)

	functions = fooProgram.Functions
	require.Len(t, functions, 11)

	defaultDestroyEventEmittingFunction := functions[7]
	require.Equal(t, "Foo.ABC.$ResourceDestroyed", defaultDestroyEventEmittingFunction.Name)

	const inheritedEventConstructorIndex = 9
	const selfDefinedABCEventConstructorIndex = 12

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// Get the `collectEvents` parameter for invocation.
			opcode.InstructionGetLocal{Local: 1},

			// Construct the inherited event
			// Bar.XYZ.ResourceDestroyed(self.x)
			opcode.InstructionGetGlobal{Global: inheritedEventConstructorIndex},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetField{FieldName: 2, AccessedType: 5},
			opcode.InstructionTransferAndConvert{Type: 6},
			opcode.InstructionInvoke{ArgCount: 1},

			// Construct the self defined event
			// Foo.ABC.ResourceDestroyed(self.x)
			opcode.InstructionGetGlobal{Global: selfDefinedABCEventConstructorIndex},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetField{FieldName: 2, AccessedType: 8},
			opcode.InstructionTransferAndConvert{Type: 6},
			opcode.InstructionInvoke{ArgCount: 1},

			// Invoke `collectEvents` with the above event.
			// `collectEvents(...)`
			opcode.InstructionInvoke{ArgCount: 2},
			opcode.InstructionDrop{},

			// Return
			opcode.InstructionReturn{},
		},
		defaultDestroyEventEmittingFunction.Code,
	)
}

func TestCompileImportAlias(t *testing.T) {

	t.Parallel()

	t.Run("simple alias", func(t *testing.T) {

		importLocation := common.NewAddressLocation(nil, common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}, "")

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
				contract Foo {
					fun hello(): String {
						return "hello"
					}
				}
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		checker, err := ParseAndCheckWithOptions(t,
			`
				import Foo as Bar from 0x01

				fun test(): String {
					return Bar.hello()
				}
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}

		program := comp.Compile()

		assert.Equal(
			t,
			[]bbq.Import{
				{
					Location: importLocation,
					Name:     "Foo",
				},
				{
					Location: importLocation,
					Name:     "Foo.hello",
				},
			},
			program.Imports,
		)

		// Imported types are location qualified.
		assertGlobalsEqual(
			t,
			map[string]bbq.GlobalInfo{
				"test": {
					Location:      nil,
					Name:          "test",
					QualifiedName: "test",
					Index:         0,
				},
				"A.0000000000000001.Foo": {
					Location:      importLocation,
					Name:          "Foo",
					QualifiedName: "A.0000000000000001.Foo",
					Index:         1,
				},
				"A.0000000000000001.Foo.hello": {
					Location:      importLocation,
					Name:          "Foo.hello",
					QualifiedName: "A.0000000000000001.Foo.hello",
					Index:         2,
				},
			},
			comp.Globals,
		)

	})

	t.Run("interface", func(t *testing.T) {

		importLocation := common.NewAddressLocation(nil, common.Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1}, "")

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
				struct interface FooInterface {
					fun hello(): String

					fun defaultHello(): String {
						return "hi"
					}
				}
            `,
			ParseAndCheckOptions{
				Location: importLocation,
			},
		)
		require.NoError(t, err)

		importCompiler := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(importedChecker),
			importedChecker.Location,
		)
		importedProgram := importCompiler.Compile()

		checker, err := ParseAndCheckWithOptions(t,
			`
				import FooInterface as FI from 0x01

				struct Bar: FI {
					fun hello(): String {
						return "hello"
					}
				}
            `,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.ImportHandler = func(location common.Location) *bbq.InstructionProgram {
			return importedProgram
		}
		comp.Config.ElaborationResolver = func(location common.Location) (*compiler.DesugaredElaboration, error) {
			switch location {
			case importLocation:
				return compiler.NewDesugaredElaboration(importedChecker.Elaboration), nil
			default:
				return nil, fmt.Errorf("cannot find elaboration for %s", location)
			}
		}

		program := comp.Compile()

		assert.Equal(
			t,
			[]bbq.Import{
				{
					Location: importLocation,
					Name:     "FooInterface.defaultHello",
				},
			},
			program.Imports,
		)

		// only imported function is a location qualified global.
		assertGlobalsEqual(
			t,
			map[string]bbq.GlobalInfo{
				"Bar": {
					Location:      nil,
					Name:          "Bar",
					QualifiedName: "Bar",
					Index:         0,
				},
				"Bar.getType": {
					Location:      nil,
					Name:          "Bar.getType",
					QualifiedName: "Bar.getType",
					Index:         1,
				},
				"Bar.hello": {
					Location:      nil,
					Name:          "Bar.hello",
					QualifiedName: "Bar.hello",
					Index:         3,
				},
				"Bar.isInstance": {
					Location:      nil,
					Name:          "Bar.isInstance",
					QualifiedName: "Bar.isInstance",
					Index:         2,
				},
				"Bar.defaultHello": {
					Location:      nil,
					Name:          "Bar.defaultHello",
					QualifiedName: "Bar.defaultHello",
					Index:         4,
				},
				"A.0000000000000001.FooInterface.defaultHello": {
					Location:      importLocation,
					Name:          "FooInterface.defaultHello",
					QualifiedName: "A.0000000000000001.FooInterface.defaultHello",
					Index:         5,
				},
			},
			comp.Globals,
		)

	})
}
