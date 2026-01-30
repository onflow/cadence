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
func assertGlobalsEqual(t *testing.T, expected []bbq.GlobalInfo, actual []bbq.Global) {
	// Check that both maps have the same keys
	assert.Equal(t, len(expected), len(actual), "globals have different lengths")

	for index, actualGlobal := range actual {
		actualGlobalInfo := actualGlobal.GetGlobalInfo()
		expectedGlobalInfo := expected[index]
		assert.Equal(t, expectedGlobalInfo, actualGlobalInfo)
	}
}

func assertTypesEqual(t *testing.T, expectedTypes, actualTypes []interpreter.StaticType) {
	require.Equal(t, len(expectedTypes), len(actualTypes))
	for i, expectedType := range expectedTypes {
		actualType := actualTypes[i]
		assert.True(t, expectedType.Equal(actualType))
	}
}

func prettyInstructions(
	instructions []opcode.Instruction,
	program *bbq.InstructionProgram,
) []opcode.PrettyInstruction {
	pretty := make([]opcode.PrettyInstruction, len(instructions))
	for i, instr := range instructions {
		pretty[i] = instr.Pretty(program)
	}
	return pretty
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

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// if n < 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionLess{},
			opcode.PrettyInstructionJumpIfFalse{Target: 9},
			// then return n
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},

			// return ...
			opcode.PrettyInstructionStatement{},
			// fib(n - 1)
			opcode.PrettyInstructionGetGlobal{Global: 0},
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionSubtract{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeInt,
			},
			// fib(n - 2)
			opcode.PrettyInstructionGetGlobal{Global: 0},
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionSubtract{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionAdd{},
			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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

	assertTypesEqual(
		t,
		[]bbq.StaticType{
			interpreter.FunctionStaticType{
				FunctionType: sema.NewSimpleFunctionType(
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

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// var fib1 = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: fib1Index},

			// var fib2 = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: fib2Index},

			// var fibonacci = fib1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: fib1Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: fibonacciIndex},

			// var i = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// while i < n
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetLocal{Local: nIndex},
			opcode.PrettyInstructionLess{},
			opcode.PrettyInstructionJumpIfFalse{Target: 43},

			opcode.PrettyInstructionLoop{},

			// fibonacci = fib1 + fib2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: fib1Index},
			opcode.PrettyInstructionGetLocal{Local: fib2Index},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: fibonacciIndex},

			// fib1 = fib2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: fib2Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: fib1Index},

			// fib2 = fibonacci
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: fibonacciIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: fib2Index},

			// i = i + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// continue loop
			opcode.PrettyInstructionJump{Target: 17},

			// return fibonacci
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: fibonacciIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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

	assertTypesEqual(
		t,
		[]bbq.StaticType{
			interpreter.FunctionStaticType{
				FunctionType: sema.NewSimpleFunctionType(
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
		[]opcode.PrettyInstruction{
			// var i = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// while true
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionJumpIfFalse{Target: 22},

			opcode.PrettyInstructionLoop{},

			// if i > 3
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionGreater{},
			opcode.PrettyInstructionJumpIfFalse{Target: 15},

			// break
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 22},

			// i = i + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// repeat
			opcode.PrettyInstructionJump{Target: 5},

			// return i
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// var i = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// while true
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionJumpIfFalse{Target: 24},

			opcode.PrettyInstructionLoop{},

			// i = i + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// if i < 3
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionLess{},
			opcode.PrettyInstructionJumpIfFalse{Target: 21},

			// continue
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 5},

			// break
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 24},

			// repeat
			opcode.PrettyInstructionJump{Target: 5},

			// return i
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// return nil
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionVoid{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeVoid,
				TargetType: interpreter.PrimitiveStaticTypeVoid,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{

			// return true
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeBool,
				TargetType: interpreter.PrimitiveStaticTypeBool,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// return false
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionFalse{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeBool,
				TargetType: interpreter.PrimitiveStaticTypeBool,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// return nil
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNil{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: interpreter.NilStaticType,
				TargetType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeBool,
				},
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{

			opcode.PrettyInstructionStatement{},

			// [1, 2, 3]
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionNewArray{
				Type: &interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				Size:       3,
				IsResource: false,
			},

			// let xs =
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: &interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				TargetType: &interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			opcode.PrettyInstructionSetLocal{Local: xsIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// {"a": 1, "b": 2, "c": 3}
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredStringValue("a"),
					Kind: constant.String,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredStringValue("b"),
					Kind: constant.String,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredStringValue("c"),
					Kind: constant.String,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionNewDictionary{
				Type: &interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeInt,
				},
				Size:       3,
				IsResource: false,
			},
			// let xs =
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: &interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeInt,
				},
				TargetType: &interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeInt,
				},
			},
			opcode.PrettyInstructionSetLocal{Local: xsIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// let y' = x
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				TargetType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			opcode.PrettyInstructionSetLocal{
				Local:     tempYIndex,
				IsTempVar: true,
			},

			// if nil
			opcode.PrettyInstructionGetLocal{Local: tempYIndex},
			opcode.PrettyInstructionJumpIfNil{Target: 14},

			// let y = y'
			opcode.PrettyInstructionGetLocal{Local: tempYIndex},
			opcode.PrettyInstructionUnwrap{},
			opcode.PrettyInstructionSetLocal{Local: yIndex},

			// then { return y }
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
			opcode.PrettyInstructionJump{Target: 18},

			// else { return 2 }
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// let x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x1Index},

			// var z = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: zIndex},

			// if let x = y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				TargetType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			opcode.PrettyInstructionSetLocal{
				Local:     tempIfLetIndex,
				IsTempVar: true,
			},

			opcode.PrettyInstructionGetLocal{Local: tempIfLetIndex},
			opcode.PrettyInstructionJumpIfNil{Target: 22},

			// then
			opcode.PrettyInstructionGetLocal{Local: tempIfLetIndex},
			opcode.PrettyInstructionUnwrap{},
			opcode.PrettyInstructionSetLocal{Local: x2Index},

			// z = x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: x2Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: zIndex},
			opcode.PrettyInstructionJump{Target: 26},

			// else { z = x }
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: x1Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: zIndex},

			// return x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: x1Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		// tempSwitchValueIndex is the index of the local variable used to store the value of the switch expression
		tempSwitchValueIndex
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// var a = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: aIndex},

			// switch x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionSetLocal{
				Local:     tempSwitchValueIndex,
				IsTempVar: true,
			},

			// case 1:
			opcode.PrettyInstructionGetLocal{Local: tempSwitchValueIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionEqual{},
			opcode.PrettyInstructionJumpIfFalse{Target: 16},

			// a = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: aIndex},

			// jump to end
			opcode.PrettyInstructionJump{Target: 29},

			// case 2:
			opcode.PrettyInstructionGetLocal{Local: tempSwitchValueIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionEqual{},
			opcode.PrettyInstructionJumpIfFalse{Target: 25},

			// a = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: aIndex},

			// jump to end
			opcode.PrettyInstructionJump{Target: 29},

			// default:
			// a = 3
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: aIndex},

			// return a
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: aIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		// tempSwitchValueIndex is the index of the local variable used to store the value of the switch expression
		tempSwitchValueIndex
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// switch x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionSetLocal{
				Local:     tempSwitchValueIndex,
				IsTempVar: true,
			},

			// case 1:
			opcode.PrettyInstructionGetLocal{Local: tempSwitchValueIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionEqual{},
			opcode.PrettyInstructionJumpIfFalse{Target: 10},
			// break
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 19},
			// end of case
			opcode.PrettyInstructionJump{Target: 19},

			// case 1:
			opcode.PrettyInstructionGetLocal{Local: tempSwitchValueIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionEqual{},
			opcode.PrettyInstructionJumpIfFalse{Target: 17},
			// break
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 19},
			// end of case
			opcode.PrettyInstructionJump{Target: 19},

			// default:
			// break
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 19},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		// tempSwitchValueIndex is the index of the local variable used to store the value of the switch expression
		tempSwitchValueIndex
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// var x = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xIndex},

			// while true
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionJumpIfFalse{Target: 25},

			opcode.PrettyInstructionLoop{},

			// switch x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionSetLocal{
				Local:     tempSwitchValueIndex,
				IsTempVar: true,
			},

			// case 1:
			opcode.PrettyInstructionGetLocal{Local: tempSwitchValueIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionEqual{},
			opcode.PrettyInstructionJumpIfFalse{Target: 18},

			// break
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 18},
			// end of case
			opcode.PrettyInstructionJump{Target: 18},

			// x = x + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xIndex},

			// repeat
			opcode.PrettyInstructionJump{Target: 5},

			// return x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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

	incType := interpreter.CompositeStaticType{
		Location:            checker.Location,
		QualifiedIdentifier: "Inc",
		TypeID:              checker.Location.TypeID(nil, "Inc"),
	}
	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},
			// x
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			// emit
			opcode.PrettyInstructionEmitEvent{
				Type:     &incType,
				ArgCount: 1,
			},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// x as Int?
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionSimpleCast{
				TargetType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},

			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				TargetType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// x as! Int
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionForceCast{
				TargetType: interpreter.PrimitiveStaticTypeInt,
				ValueType:  interpreter.PrimitiveStaticTypeAnyStruct,
			},

			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// x as? Int
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionFailableCast{
				TargetType: interpreter.PrimitiveStaticTypeInt,
				ValueType:  interpreter.PrimitiveStaticTypeAnyStruct,
			},

			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
				TargetType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// var i = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// while i < 10
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(10),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionLess{},

			opcode.PrettyInstructionJumpIfFalse{Target: 45},

			opcode.PrettyInstructionLoop{},

			// var j = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: jIndex},

			// while j < 10
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: jIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(10),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionLess{},
			opcode.PrettyInstructionJumpIfFalse{Target: 36},

			opcode.PrettyInstructionLoop{},

			// if i == j
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetLocal{Local: jIndex},
			opcode.PrettyInstructionEqual{},

			opcode.PrettyInstructionJumpIfFalse{Target: 27},

			// break
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 36},

			// j = j + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: jIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: jIndex},

			// continue
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 15},

			// repeat
			opcode.PrettyInstructionJump{Target: 15},

			// i = i + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: iIndex},

			// continue
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionJump{Target: 5},

			// repeat
			opcode.PrettyInstructionJump{Target: 5},

			// return i
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: iIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// var x = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xIndex},

			// x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetGlobal{Global: xIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	// global var `x` initializer
	variables := program.Variables
	require.Len(t, variables, 1)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// return 0
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(variables[xIndex].Getter.Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// array[index]
			opcode.PrettyInstructionGetLocal{Local: arrayIndex},
			opcode.PrettyInstructionGetLocal{Local: indexIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInteger,
			},
			opcode.PrettyInstructionGetIndex{},

			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: arrayIndex},
			opcode.PrettyInstructionGetLocal{Local: indexIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInteger,
			},
			opcode.PrettyInstructionGetLocal{Local: valueIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetIndex{},
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
	require.Len(t, functions, 5)

	const (
		initFuncIndex = iota
		// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment)
		_
		_
		_
		getValueFuncIndex
	)

	testType := &interpreter.CompositeStaticType{
		Location:            checker.Location,
		QualifiedIdentifier: "Test",
		TypeID:              checker.Location.TypeID(nil, "Test"),
	}

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
			[]opcode.PrettyInstruction{
				// let self = Test()
				opcode.PrettyInstructionNewComposite{
					Kind: common.CompositeKindStructure,
					Type: testType,
				},
				opcode.PrettyInstructionSetLocal{Local: selfIndex},

				// self.foo = value
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: selfIndex},
				opcode.PrettyInstructionGetLocal{Local: valueIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetField{
					FieldName: constant.DecodedConstant{
						Data: "foo",
						Kind: constant.RawString,
					},
					AccessedType: testType,
				},

				// return self
				opcode.PrettyInstructionGetLocal{Local: selfIndex},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[initFuncIndex].Code, program),
		)
	}

	{
		// nIndex is the index of the parameter `self`, which is the first parameter
		const selfIndex = 0

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: selfIndex},
				opcode.PrettyInstructionGetField{
					FieldName: constant.DecodedConstant{
						Data: "foo",
						Kind: constant.RawString,
					},
					AccessedType: testType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[getValueFuncIndex].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// f()
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetGlobal{Global: 0},
			opcode.PrettyInstructionInvoke{
				ReturnType: interpreter.PrimitiveStaticTypeVoid,
			},
			opcode.PrettyInstructionDrop{},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[1].Code, program),
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
		[]opcode.PrettyInstruction{
			// let yes = true
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeBool,
				TargetType: interpreter.PrimitiveStaticTypeBool,
			},
			opcode.PrettyInstructionSetLocal{Local: yesIndex},

			// let no = false
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionFalse{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeBool,
				TargetType: interpreter.PrimitiveStaticTypeBool,
			},
			opcode.PrettyInstructionSetLocal{Local: noIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// return "Hello, world!"
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredStringValue("Hello, world!"),
					Kind: constant.String,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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

			expectedConstantKind := constant.FromSemaType(integerType)
			targetType := program.Types[1]

			assert.Equal(t,
				[]opcode.PrettyInstruction{
					// let v: ... = 2
					opcode.PrettyInstructionStatement{},
					opcode.PrettyInstructionGetConstant{
						Constant: constant.DecodedConstant{
							Data: expectedData,
							Kind: expectedConstantKind,
						},
					},
					opcode.PrettyInstructionTransferAndConvert{
						ValueType:  targetType,
						TargetType: targetType,
					},
					opcode.PrettyInstructionSetLocal{Local: vIndex},

					opcode.PrettyInstructionReturn{},
				},
				prettyInstructions(functions[0].Code, program),
			)

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

			expectedConstantKind := constant.FromSemaType(integerType)
			targetType := program.Types[1]

			assert.Equal(t,
				[]opcode.PrettyInstruction{
					// let v: ... = -3
					opcode.PrettyInstructionStatement{},
					opcode.PrettyInstructionGetConstant{
						Constant: constant.DecodedConstant{
							Data: expectedData,
							Kind: expectedConstantKind,
						},
					},
					opcode.PrettyInstructionTransferAndConvert{
						ValueType:  targetType,
						TargetType: targetType,
					},
					opcode.PrettyInstructionSetLocal{Local: vIndex},

					opcode.PrettyInstructionReturn{},
				},
				prettyInstructions(functions[0].Code, program),
			)

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
		[]opcode.PrettyInstruction{
			// let v: Address = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{1}),
					Kind: constant.Address,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeAddress,
				TargetType: interpreter.PrimitiveStaticTypeAddress,
			},
			opcode.PrettyInstructionSetLocal{Local: vIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{1}),
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

			expectedConstantKind := constant.FromSemaType(fixedPointType)
			targetType := program.Types[1]

			assert.Equal(t,
				[]opcode.PrettyInstruction{
					// let v: ... = 2.3
					opcode.PrettyInstructionStatement{},
					opcode.PrettyInstructionGetConstant{
						Constant: constant.DecodedConstant{
							Data: expectedData,
							Kind: expectedConstantKind,
						},
					},
					opcode.PrettyInstructionTransferAndConvert{
						ValueType:  targetType,
						TargetType: targetType,
					},
					opcode.PrettyInstructionSetLocal{Local: vIndex},

					opcode.PrettyInstructionReturn{},
				},
				prettyInstructions(functions[0].Code, program),
			)

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

			expectedConstantKind := constant.FromSemaType(fixedPointType)
			targetType := program.Types[1]

			assert.Equal(t,
				[]opcode.PrettyInstruction{
					// let v: ... = -2.3
					opcode.PrettyInstructionStatement{},
					opcode.PrettyInstructionGetConstant{
						Constant: constant.DecodedConstant{
							Data: expectedData,
							Kind: expectedConstantKind,
						},
					},
					opcode.PrettyInstructionTransferAndConvert{
						ValueType:  targetType,
						TargetType: targetType,
					},
					opcode.PrettyInstructionSetLocal{Local: vIndex},

					opcode.PrettyInstructionReturn{},
				},
				prettyInstructions(functions[0].Code, program),
			)

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
		[]opcode.PrettyInstruction{
			// let no = !true
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionNot{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeBool,
				TargetType: interpreter.PrimitiveStaticTypeBool,
			},
			opcode.PrettyInstructionSetLocal{Local: noIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// let v = -x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionNegate{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: vIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// let v = *ref
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: refIndex},
			opcode.PrettyInstructionDeref{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: vIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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

			prettyInstruction := instruction.Pretty(program)
			resultType := program.Types[1]

			assert.Equal(t,
				[]opcode.PrettyInstruction{
					// let v = 6 ... 3
					opcode.PrettyInstructionStatement{},
					opcode.PrettyInstructionGetConstant{
						Constant: constant.DecodedConstant{
							Data: interpreter.NewUnmeteredIntValueFromInt64(6),
							Kind: constant.Int,
						},
					},
					opcode.PrettyInstructionGetConstant{
						Constant: constant.DecodedConstant{
							Data: interpreter.NewUnmeteredIntValueFromInt64(3),
							Kind: constant.Int,
						},
					},
					prettyInstruction,
					opcode.PrettyInstructionTransferAndConvert{
						ValueType:  resultType,
						TargetType: resultType,
					},
					opcode.PrettyInstructionSetLocal{Local: vIndex},

					opcode.PrettyInstructionReturn{},
				},
				prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// value ??
			opcode.PrettyInstructionGetLocal{Local: valueIndex},
			opcode.PrettyInstructionDup{},
			opcode.PrettyInstructionJumpIfNil{Target: 7},

			// value
			opcode.PrettyInstructionUnwrap{},
			// The Value type should be the unwrapped `Int`.
			opcode.PrettyInstructionConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionJump{Target: 10},

			// 0
			opcode.PrettyInstructionDrop{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			// The Value type should be the unwrapped `Int`.
			opcode.PrettyInstructionConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},

			// return
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	assertTypesEqual(
		t,
		[]bbq.StaticType{
			interpreter.FunctionStaticType{
				FunctionType: sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					[]sema.Parameter{
						{
							TypeAnnotation: sema.NewTypeAnnotation(
								sema.NewOptionalType(nil, sema.IntType),
							),
							Label:      sema.ArgumentLabelNotRequired,
							Identifier: "value",
						},
					},
					sema.NewTypeAnnotation(sema.IntType),
				),
			},
			interpreter.PrimitiveStaticTypeInt,
		},
		program.Types,
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
	require.Len(t, functions, 6)

	const (
		testFuncIndex = iota
		initFuncIndex
		// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment)
		_
		_
		_
		fFuncIndex
	)

	fooType := &interpreter.CompositeStaticType{
		Location:            checker.Location,
		QualifiedIdentifier: "Foo",
		TypeID:              checker.Location.TypeID(nil, "Foo"),
	}

	{
		const (
			// fooIndex is the index of the local variable `foo`, which is the first local variable
			fooIndex = iota
		)

		fooType := fooType

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let foo = Foo()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: initFuncIndex},
				opcode.PrettyInstructionInvoke{
					ReturnType: fooType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  fooType,
					TargetType: fooType,
				},
				opcode.PrettyInstructionSetLocal{Local: fooIndex},

				// foo.f(true)
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: fooIndex},
				opcode.PrettyInstructionGetMethod{
					Method:       fFuncIndex,
					ReceiverType: fooType,
				},
				opcode.PrettyInstructionTrue{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeBool,
					TargetType: interpreter.PrimitiveStaticTypeBool,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeVoid,
				},
				opcode.PrettyInstructionDrop{},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[testFuncIndex].Code, program),
		)
	}

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// Foo()
			opcode.PrettyInstructionNewComposite{
				Kind: common.CompositeKindStructure,
				Type: fooType,
			},

			// NOTE: no redundant set-local / get-local for self in struct init

			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[initFuncIndex].Code, program),
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[fFuncIndex].Code, program),
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
	require.Len(t, functions, 5)

	const (
		testFuncIndex = iota
		initFuncIndex
		// Next three indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
		_
	)

	fooType := &interpreter.CompositeStaticType{
		Location:            checker.Location,
		QualifiedIdentifier: "Foo",
		TypeID:              checker.Location.TypeID(nil, "Foo"),
	}

	{
		const (
			// fooIndex is the index of the local variable `foo`, which is the first local variable
			fooIndex = iota
		)

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let foo <- create Foo()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: initFuncIndex},
				opcode.PrettyInstructionInvoke{
					ReturnType: fooType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  fooType,
					TargetType: fooType,
				},
				opcode.PrettyInstructionSetLocal{Local: fooIndex},

				// destroy foo
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: fooIndex},
				opcode.PrettyInstructionDestroy{},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[testFuncIndex].Code, program),
		)
	}

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// Foo()
			opcode.PrettyInstructionNewComposite{
				Kind: common.CompositeKindResource,
				Type: fooType,
			},

			// NOTE: no redundant set-local / get-local for self in resource init

			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[initFuncIndex].Code, program),
	)
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// /storage/foo
			opcode.PrettyInstructionNewPath{
				Domain: common.PathDomainStorage,
				Identifier: constant.DecodedConstant{
					Data: "foo",
					Kind: constant.RawString,
				},
			},

			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeStoragePath,
				TargetType: interpreter.PrimitiveStaticTypePath,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// let x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x1Index},

			// if y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionJumpIfFalse{Target: 12},

			// { let x = 2 }
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x2Index},

			opcode.PrettyInstructionJump{Target: 16},

			// else { let x = 3 }
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x3Index},

			// return x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: x1Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// let x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x1Index},

			// if y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionJumpIfFalse{Target: 16},

			// var x = x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: x1Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x2Index},

			// x = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x2Index},

			opcode.PrettyInstructionJump{Target: 24},

			// var x = x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: x1Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x3Index},

			// x = 3
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: x3Index},

			// return x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: x1Index},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
            fun test(x: Int): Int {
                return x
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
	require.Len(t, functions, 9)

	const (
		concreteTypeConstructorIndex uint16 = iota
		// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment) for concrete type
		_
		_
		_
		concreteTypeFunctionIndex
		// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment) for interface type
		_
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
	//     fun test(x: Int): Int {
	//        return self.test(x: x)
	//    }
	// ```

	const (
		selfIndex = iota
		xIndex
	)

	iaType := &interpreter.InterfaceStaticType{
		Location:            checker.Location,
		QualifiedIdentifier: "IA",
		TypeID:              checker.Location.TypeID(nil, "IA"),
	}

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// self.test(x: x)
			opcode.PrettyInstructionGetLocal{Local: selfIndex},
			opcode.PrettyInstructionGetMethod{
				Method:       interfaceFunctionIndex,
				ReceiverType: iaType,
			},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			// NOTE: no transfer or convert of argument
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeInt,
			},

			// return
			// NOTE: no transfer or convert of value
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(concreteTypeTestFunc.Code, program),
	)

	// 	`IA` type's `test` function

	const interfaceTypeTestFuncName = "IA.test"
	interfaceTypeTestFunc := program.Functions[interfaceFunctionIndex]
	require.Equal(t, interfaceTypeTestFuncName, interfaceTypeTestFunc.QualifiedName)

	// Also check if the globals are linked properly.
	assert.Equal(t, interfaceFunctionIndex, comp.Globals[interfaceTypeTestFuncName].GetGlobalInfo().Index)

	// Should contain the implementation.
	// ```
	//    fun test(x: Int): Int {
	//        return x
	//    }
	// ```

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// x
			opcode.PrettyInstructionGetLocal{Local: 1},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},

			// return
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(interfaceTypeTestFunc.Code, program),
	)

	assert.Equal(t,
		[]constant.DecodedConstant(nil),
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
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// x > 0
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(0),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionGreater{},

				// if !<condition>
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 12},

				// $failPreCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1}, // global index 1 is 'panic' function
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},

				// Drop since it's a statement-expression
				opcode.PrettyInstructionDrop{},

				// return 5
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(5),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// $_result = 5
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(5),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.PrettyInstructionJump{Target: 6},

				// let result $noTransfer $_result
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.PrettyInstructionSetLocal{Local: resultIndex},

				opcode.PrettyInstructionStatement{},

				// x > 0
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(0),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionGreater{},

				// if !<condition>
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 21},

				// $failPostCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1}, // global index 1 is 'panic' function
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},

				// Drop since it's a statement-expression
				opcode.PrettyInstructionDrop{},

				// return $_result
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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

		optionalAnyResourceType := &interpreter.OptionalStaticType{
			Type: interpreter.PrimitiveStaticTypeAnyResource,
		}

		optionalRefAnyResourceType := &interpreter.OptionalStaticType{
			Type: &interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeAnyResource,
			},
		}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// $_result <- x
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransfer{},
				opcode.PrettyInstructionConvert{
					ValueType:  optionalAnyResourceType,
					TargetType: optionalAnyResourceType,
				},
				opcode.PrettyInstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.PrettyInstructionJump{Target: 7},

				// Get the reference and assign to `result`.
				// i.e: `let result $noTransfer &$_result`
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				opcode.PrettyInstructionNewRef{Type: optionalRefAnyResourceType},
				// NOTE: Explicitly no transferAndConvert
				opcode.PrettyInstructionSetLocal{Local: resultIndex},

				opcode.PrettyInstructionStatement{},

				// result != nil
				opcode.PrettyInstructionGetLocal{Local: resultIndex},
				opcode.PrettyInstructionNil{},
				opcode.PrettyInstructionNotEqual{},

				// if !<condition>
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 23},

				// $failPostCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1}, // global index 1 is 'panic' function
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},

				// Drop since it's a statement-expression
				opcode.PrettyInstructionDrop{},

				// return $_result
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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
		require.Len(t, functions, 8)

		// Function indexes
		const (
			concreteTypeConstructorIndex uint16 = iota
			// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment) for concrete type
			_
			_
			_
			concreteTypeFunctionIndex
			// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment) for interface type
			_
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
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// Inherited pre-condition
				// x > 0
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(0),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionGreater{},

				// if !<condition>
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 12},

				// $failPreCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: failPreConditionFunctionIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},

				// Drop since it's a statement-expression
				opcode.PrettyInstructionDrop{},

				// Function body

				opcode.PrettyInstructionStatement{},

				// $_result = 42
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(42),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.PrettyInstructionJump{Target: 18},

				// let result $noTransfer $_result
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.PrettyInstructionSetLocal{Local: resultIndex},

				// Inherited post condition

				opcode.PrettyInstructionStatement{},

				// y > 0
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(0),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionGreater{},

				// if !<condition>
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 33},

				// $failPostCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: failPostConditionFunctionIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},

				// Drop since it's a statement-expression
				opcode.PrettyInstructionDrop{},

				// return $_result
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(concreteTypeTestFunc.Code, program),
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
		require.Len(t, functions, 8)

		// Function indexes
		const (
			concreteTypeConstructorIndex uint16 = iota
			// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment) for concrete type
			_
			_
			_
			concreteTypeFunctionIndex
			// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment) for interface type
			_
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
			[]opcode.PrettyInstruction{
				// Inherited before function

				// var $before_0 = x
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: beforeVarIndex},

				// Function body

				opcode.PrettyInstructionStatement{},

				// $_result = 42
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(42),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.PrettyInstructionJump{Target: 10},

				// let result $noTransfer $_result
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.PrettyInstructionSetLocal{Local: resultIndex},

				// Inherited post condition

				opcode.PrettyInstructionStatement{},

				// $before_0 < x
				opcode.PrettyInstructionGetLocal{Local: beforeVarIndex},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionLess{},

				// if !<condition>
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 25},

				// $failPostCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: failPostConditionFunctionIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},

				// Drop since it's a statement-expression
				opcode.PrettyInstructionDrop{},

				// return $_result
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(concreteTypeTestFunc.Code, program),
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

		contractsAddress := common.MustBytesToAddress([]byte{1})

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
		require.Len(t, dProgram.Functions, 9)

		// Function indexes
		const (
			concreteTypeFunctionIndex     = 8
			failPreConditionFunctionIndex = 12
		)

		// `D.Vault` type's `getBalance` function.

		// Local var indexes
		const (
			selfIndex = iota
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

		aTestStructType := &interpreter.CompositeStaticType{
			Location:            aLocation,
			QualifiedIdentifier: "A.TestStruct",
			TypeID:              aLocation.TypeID(nil, "A.TestStruct"),
		}

		dVaultType := &interpreter.CompositeStaticType{
			Location:            dLocation,
			QualifiedIdentifier: "D.Vault",
			TypeID:              dLocation.TypeID(nil, "D.Vault"),
		}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// Load receiver `A.TestStruct()`
				opcode.PrettyInstructionGetGlobal{Global: 10},
				opcode.PrettyInstructionInvoke{ReturnType: aTestStructType},

				// Get function value `A.TestStruct.test()`
				opcode.PrettyInstructionGetMethod{
					Method:       11,
					ReceiverType: aTestStructType,
				},
				opcode.PrettyInstructionInvoke{ReturnType: interpreter.PrimitiveStaticTypeBool},

				// if !<condition>
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 13},

				// $failPreCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: failPreConditionFunctionIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},

				// Drop since it's a statement-expression
				opcode.PrettyInstructionDrop{},

				// return self.balance
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: selfIndex},
				opcode.PrettyInstructionGetField{
					FieldName: constant.DecodedConstant{
						Data: "balance",
						Kind: constant.RawString,
					},
					AccessedType: dVaultType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(concreteTypeTestFunc.Code, dProgram),
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
			[]opcode.PrettyInstruction{

				// Before-statements

				// var exp_0 = x
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: beforeExprValueIndex},

				// Pre conditions

				// if !(x > 0)
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(0),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionGreater{},
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 16},

				// $failPreCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},
				opcode.PrettyInstructionDrop{},

				// Function body
				opcode.PrettyInstructionStatement{},

				// $_result = 5
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(5),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.PrettyInstructionJump{Target: 22},

				// let result $noTransfer $_result
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.PrettyInstructionSetLocal{Local: resultIndex},

				// Post conditions

				// if !(exp_0 < x)
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: beforeExprValueIndex},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionLess{},
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 37},

				// $failPostCondition("")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 2},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},
				opcode.PrettyInstructionDrop{},

				// return $_result
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.PrettyInstructionGetLocal{Local: arrayValueIndex},
				opcode.PrettyInstructionIterator{},
				opcode.PrettyInstructionSetLocal{Local: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.PrettyInstructionJumpIfFalse{Target: 13},

				opcode.PrettyInstructionLoop{},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorNext{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: elementVarIndex},

				// Jump to the beginning (condition) of the loop.
				opcode.PrettyInstructionJump{Target: 4},

				// End of the loop, end the iterator.
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorEnd{},

				// Return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: arrayValueIndex},
				opcode.PrettyInstructionIterator{},
				opcode.PrettyInstructionSetLocal{Local: iteratorVarIndex},

				// Initialize index.
				// `var i = -1`
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(-1),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionSetLocal{Local: indexVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.PrettyInstructionJumpIfFalse{Target: 19},

				opcode.PrettyInstructionLoop{},

				// If true:

				// Increment the index
				opcode.PrettyInstructionGetLocal{Local: indexVarIndex},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(1),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionAdd{},
				opcode.PrettyInstructionSetLocal{Local: indexVarIndex},

				// Get the next element and store in local var.
				// var e = iterator.next()
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorNext{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: elementVarIndex},

				// Jump to the beginning (condition) of the loop.
				opcode.PrettyInstructionJump{Target: 6},

				// End of the loop, end the iterator.
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorEnd{},

				// Return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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
			[]opcode.PrettyInstruction{

				// var x = 5
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(5),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: x1Index},

				// Get the iterator and store in local var.
				// `var <iterator> = array.Iterator`
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: arrayValueIndex},
				opcode.PrettyInstructionIterator{},
				opcode.PrettyInstructionSetLocal{Local: iteratorVarIndex},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.PrettyInstructionJumpIfFalse{Target: 25},

				opcode.PrettyInstructionLoop{},

				// If true, get the next element and store in local var.
				// var e = iterator.next()
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorNext{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: e1Index},

				// var e = e
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: e1Index},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: e2Index},

				// var x = 8
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(8),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: x2Index},

				// Jump to the beginning (condition) of the loop.
				opcode.PrettyInstructionJump{Target: 8},

				// End of the loop, end the iterator.
				opcode.PrettyInstructionGetLocal{Local: iteratorVarIndex},
				opcode.PrettyInstructionIteratorEnd{},

				// Return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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
			[]opcode.PrettyInstruction{

				// Get the iterator and store in local var.
				// `var <iterator> = a.Iterator`
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: aIndex},
				opcode.PrettyInstructionIterator{},
				opcode.PrettyInstructionSetLocal{Local: iter1Index},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.PrettyInstructionGetLocal{Local: iter1Index},
				opcode.PrettyInstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.PrettyInstructionJumpIfFalse{Target: 44},

				opcode.PrettyInstructionLoop{},

				// If true, get the next element and store in local var.
				// var x = iterator.next()
				opcode.PrettyInstructionGetLocal{Local: iter1Index},
				opcode.PrettyInstructionIteratorNext{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: xIndex},

				// Get the iterator and store in local var.
				// `var <iterator> = b.Iterator`
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: bIndex},
				opcode.PrettyInstructionIterator{},
				opcode.PrettyInstructionSetLocal{Local: iter2Index},

				// Loop condition: Check whether `iterator.hasNext()`
				opcode.PrettyInstructionGetLocal{Local: iter2Index},
				opcode.PrettyInstructionIteratorHasNext{},

				// If false, then jump to the end of the loop
				opcode.PrettyInstructionJumpIfFalse{Target: 35},

				opcode.PrettyInstructionLoop{},

				// If true, get the next element and store in local var.
				// var y = iterator.next()
				opcode.PrettyInstructionGetLocal{Local: iter2Index},
				opcode.PrettyInstructionIteratorNext{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: yIndex},

				// return x + y
				// Also, end all active iterators (inner and outer).
				opcode.PrettyInstructionStatement{},

				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionAdd{},

				opcode.PrettyInstructionGetLocal{Local: iter1Index},
				opcode.PrettyInstructionIteratorEnd{},
				opcode.PrettyInstructionGetLocal{Local: iter2Index},
				opcode.PrettyInstructionIteratorEnd{},

				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},

				// Jump to the beginning (condition) of the inner loop.
				opcode.PrettyInstructionJump{Target: 16},

				// End of the loop, end the inner iterator.
				opcode.PrettyInstructionGetLocal{Local: iter2Index},
				opcode.PrettyInstructionIteratorEnd{},

				// return x
				// Also, end all active iterators (outer).
				opcode.PrettyInstructionStatement{},

				opcode.PrettyInstructionGetLocal{Local: xIndex},

				opcode.PrettyInstructionGetLocal{Local: iter1Index},
				opcode.PrettyInstructionIteratorEnd{},

				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},

				// Jump to the beginning (condition) of the outer loop.
				opcode.PrettyInstructionJump{Target: 4},

				// End of the loop, end the outer iterator.
				opcode.PrettyInstructionGetLocal{Local: iter1Index},
				opcode.PrettyInstructionIteratorEnd{},

				// return 0
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(0),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(program.Functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// var y = 0
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(0),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: yIndex},

			// if x
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionJumpIfFalse{Target: 12},

			// then { y = 1 }
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: yIndex},

			opcode.PrettyInstructionJump{Target: 16},

			// else { y = 2 }
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: yIndex},

			// return y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},

			// return x ? 1 : 2
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionJumpIfFalse{Target: 5},

			// then: 1
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},

			opcode.PrettyInstructionJump{Target: 6},

			// else: 2
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},

			// return
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// return x || y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionJumpIfTrue{Target: 5},

			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionJumpIfFalse{Target: 7},

			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionJump{Target: 8},

			opcode.PrettyInstructionFalse{},

			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeBool,
				TargetType: interpreter.PrimitiveStaticTypeBool,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// return x && y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionJumpIfFalse{Target: 7},

			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionJumpIfFalse{Target: 7},

			opcode.PrettyInstructionTrue{},
			opcode.PrettyInstructionJump{Target: 8},

			opcode.PrettyInstructionFalse{},

			// return
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeBool,
				TargetType: interpreter.PrimitiveStaticTypeBool,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
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
	require.Len(t, functions, 7)

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
		// Next three indexes are for builtin methods (i.e: getType, isInstance, forEachAttachment)
		_
		_
		_
		prepareFunctionIndex
		executeFunctionIndex
		programInitFunctionIndex
	)

	const transactionParameterCount = 1

	const (
		nGlobalIndex = iota
		// Next 7 indexes are for functions, see above
		_
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

	transactionType := &interpreter.CompositeStaticType{
		Location:            checker.Location,
		QualifiedIdentifier: commons.TransactionWrapperCompositeName,
		TypeID:              checker.Location.TypeID(nil, commons.TransactionWrapperCompositeName),
	}

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionNewSimpleComposite{
				Kind: common.CompositeKindStructure,
				Type: transactionType,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(constructor.Code, program),
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
		[]opcode.PrettyInstruction{
			// self.count = 1 + n
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: selfIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionGetGlobal{Global: nGlobalIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetField{
				FieldName: constant.DecodedConstant{
					Data: "count",
					Kind: constant.RawString,
				},
				AccessedType: transactionType,
			},

			// return
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(prepareFunction.Code, program),
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
		[]opcode.PrettyInstruction{
			// Pre condition
			// `self.count == 2 + n: "pre failed"`
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: selfIndex},
			opcode.PrettyInstructionGetField{
				FieldName: constant.DecodedConstant{
					Data: "count",
					Kind: constant.RawString,
				},
				AccessedType: transactionType,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionGetGlobal{Global: nGlobalIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionEqual{},

			// if !<condition>
			opcode.PrettyInstructionNot{},
			opcode.PrettyInstructionJumpIfFalse{Target: 15},

			// $failPreCondition("pre failed")
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetGlobal{Global: failPreConditionGlobalIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredStringValue("pre failed"),
					Kind: constant.String,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeNever,
			},

			// Drop since it's a statement-expression
			opcode.PrettyInstructionDrop{},

			// self.count = 3 + n
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: selfIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionGetGlobal{Global: nGlobalIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetField{
				FieldName: constant.DecodedConstant{
					Data: "count",
					Kind: constant.RawString,
				},
				AccessedType: transactionType,
			},

			// Post condition
			// `self.count == 4 + n: "post failed"`
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: selfIndex},
			opcode.PrettyInstructionGetField{
				FieldName: constant.DecodedConstant{
					Data: "count",
					Kind: constant.RawString,
				},
				AccessedType: transactionType,
			},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(4),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionGetGlobal{Global: nGlobalIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionEqual{},

			// if !<condition>
			opcode.PrettyInstructionNot{},
			opcode.PrettyInstructionJumpIfFalse{Target: 37},

			// $failPostCondition("post failed")
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetGlobal{Global: failPostConditionGlobalIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredStringValue("post failed"),
					Kind: constant.String,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeNever,
			},

			// Drop since it's a statement-expression
			opcode.PrettyInstructionDrop{},

			// return
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(executeFunction.Code, program),
	)

	// Program init function
	initFunction := program.Functions[programInitFunctionIndex]
	require.Equal(t,
		commons.ProgramInitFunctionName,
		initFunction.QualifiedName,
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// n = $_param_n
			opcode.PrettyInstructionGetLocal{Local: 0},
			// NOTE: no transfer, intentional to avoid copy
			opcode.PrettyInstructionSetGlobal{Global: nGlobalIndex},
		},
		prettyInstructions(initFunction.Code, program),
	)
}

func TestCompileForce(t *testing.T) {

	t.Parallel()

	t.Run("optional", func(t *testing.T) {

		t.Parallel()

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
			[]opcode.PrettyInstruction{
				// return x!
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionUnwrap{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[0].Code, program),
		)
	})

	t.Run("non-optional", func(t *testing.T) {
		t.Parallel()

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
			[]opcode.PrettyInstruction{
				// return x!
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionUnwrap{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// return
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// return x
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// return <- x
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				// There should be only one transfer
				opcode.PrettyInstructionTransfer{},
				opcode.PrettyInstructionConvert{
					ValueType:  interpreter.PrimitiveStaticTypeAnyResource,
					TargetType: interpreter.PrimitiveStaticTypeAnyResource,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// Jump to post conditions
				opcode.PrettyInstructionJump{Target: 2},

				// Post condition
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionTrue{},
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 12},

				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},
				opcode.PrettyInstructionDrop{},

				// return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// var a = 5
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(5),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: aIndex},

				// $_result = a
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: aIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: tempResultIndex},

				// Jump to post conditions
				opcode.PrettyInstructionJump{Target: 10},

				// let result $noTransfer $_result
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				// NOTE: Explicitly no transferAndConvert
				opcode.PrettyInstructionSetLocal{Local: resultIndex},

				// Post condition
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionTrue{},
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 23},

				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},
				opcode.PrettyInstructionDrop{},

				// return $_result
				// Note: no transfer/convert, since the value is already
				// transferred/converted when assigning to `$_result`.
				opcode.PrettyInstructionGetLocal{Local: tempResultIndex},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// invoke `voidReturnFunc()`
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{ReturnType: interpreter.PrimitiveStaticTypeVoid},

				// Drop the returning void value
				opcode.PrettyInstructionDrop{},

				// Jump to post conditions
				opcode.PrettyInstructionJump{Target: 5},

				// Post condition
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionTrue{},
				opcode.PrettyInstructionNot{},
				opcode.PrettyInstructionJumpIfFalse{Target: 15},

				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 2},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(""),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount:   1,
					ReturnType: interpreter.PrimitiveStaticTypeNever,
				},
				opcode.PrettyInstructionDrop{},

				// return $_result
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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

	// Ideally we would assert a concrete function type here,
	// but that would require a custom assertion function,
	// as function types are not directly comparable.
	functionType := program.Types[1]

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let addOne = fun ...
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{Function: "<anonymous>"},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  functionType,
				TargetType: functionType,
			},
			opcode.PrettyInstructionSetLocal{Local: addOneIndex},

			// let x = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xIndex},

			// return x + addOne(3)
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionGetLocal{Local: addOneIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// return x + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[1].Code, program),
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
		[]opcode.PrettyInstruction{
			// fun addOne(...
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{Function: "addOne"},
			opcode.PrettyInstructionSetLocal{Local: addOneIndex},

			// let x = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xIndex},

			// return x + addOne(3)
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionGetLocal{Local: addOneIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(3),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// return x + 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[1].Code, program),
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

	// Ideally we would assert a concrete function type here,
	// but that would require a custom assertion function,
	// as function types are not directly comparable.
	functionType := program.Types[2]

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xLocalIndex},

			// let inner = fun(): Int { ...
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{
				Function: "<anonymous>",
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  functionType,
				TargetType: functionType,
			},
			opcode.PrettyInstructionSetLocal{Local: innerLocalIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	// xUpvalueIndex is the upvalue index of the variable `x`, which is the first upvalue
	const xUpvalueIndex = 0

	// yIndex is the index of the local variable `y`, which is the first local variable
	const yIndex = 0

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let y = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[1].Code, program),
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
		[]opcode.PrettyInstruction{
			// let x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xLocalIndex},

			// fun inner(): Int { ...
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{
				Function: "inner",
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.PrettyInstructionSetLocal{Local: innerLocalIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	// xUpvalueIndex is the upvalue index of the variable `x`, which is the first upvalue
	const xUpvalueIndex = 0

	// yIndex is the index of the local variable `y`, which is the first local variable
	const yIndex = 0

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let y = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[1].Code, program),
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
		[]opcode.PrettyInstruction{
			// let x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xLocalIndex},

			// fun middle(): Int { ...
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{
				Function: "middle",
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.PrettyInstructionSetLocal{Local: middleLocalIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	// innerLocalIndex is the local index of the variable `inner`, which is the first local variable
	const innerLocalIndex = 0

	// xUpvalueIndex is the upvalue index of the variable `x`, which is the first upvalue
	const xUpvalueIndex = 0

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// fun inner(): Int { ...
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{
				Function: "inner",
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: xUpvalueIndex,
						IsLocal:     false,
					},
				},
			},
			opcode.PrettyInstructionSetLocal{Local: innerLocalIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[1].Code, program),
	)

	// yIndex is the index of the local variable `y`, which is the first local variable
	const yIndex = 0

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let y = 2
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(2),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: yIndex},

			// return x + y
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetUpvalue{Upvalue: xUpvalueIndex},
			opcode.PrettyInstructionGetLocal{Local: yIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[2].Code, program),
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
		[]opcode.PrettyInstruction{
			// fun inner() { ...
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{
				Function: "inner",
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: innerLocalIndex,
						IsLocal:     true,
					},
				},
			},
			opcode.PrettyInstructionSetLocal{Local: innerLocalIndex},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	// innerUpvalueIndex is the upvalue index of the variable `inner`, which is the first upvalue
	const innerUpvalueIndex = 0

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetUpvalue{
				Upvalue: innerUpvalueIndex,
			},
			opcode.PrettyInstructionInvoke{
				ReturnType: interpreter.PrimitiveStaticTypeVoid,
			},
			opcode.PrettyInstructionDrop{},
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[1].Code, program),
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
			[]opcode.PrettyInstruction{
				// let a = 1
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(1),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: aLocalIndex},

				// let b = 2
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(2),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: bLocalIndex},

				// fun middle(): Int { ...
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionNewClosure{
					Function: "middle",
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
				opcode.PrettyInstructionSetLocal{Local: middleLocalIndex},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// let c = 3
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(3),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: cLocalIndex},

				// let d = 4
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(4),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: dLocalIndex},

				// fun inner(): Int { ...
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionNewClosure{
					Function: "inner",
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
				opcode.PrettyInstructionSetLocal{Local: innerLocalIndex},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[1].Code, program),
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
			[]opcode.PrettyInstruction{
				// let e = 5
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(5),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: eLocalIndex},

				// let f = 6
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(6),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: fLocalIndex},

				// return f + e + d + b + c + a
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: fLocalIndex},
				opcode.PrettyInstructionGetLocal{Local: eLocalIndex},
				opcode.PrettyInstructionAdd{},

				opcode.PrettyInstructionGetUpvalue{Upvalue: dUpvalueIndex},
				opcode.PrettyInstructionAdd{},

				opcode.PrettyInstructionGetUpvalue{Upvalue: bUpvalueIndex},
				opcode.PrettyInstructionAdd{},

				opcode.PrettyInstructionGetUpvalue{Upvalue: cUpvalueIndex},
				opcode.PrettyInstructionAdd{},

				opcode.PrettyInstructionGetUpvalue{Upvalue: aUpvalueIndex},
				opcode.PrettyInstructionAdd{},

				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[2].Code, program),
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
			[]opcode.PrettyInstruction{
				// let x = 1
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(1),
						Kind: constant.Int,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeInt,
					TargetType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionSetLocal{Local: 0},

				// return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// let x = 1
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredIntValueFromInt64(1),
						Kind: constant.Int,
					},
				},
				// NOTE: transfer
				opcode.PrettyInstructionTransferAndConvert{
					ValueType: interpreter.PrimitiveStaticTypeInt,
					TargetType: &interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeInt,
					},
				},
				opcode.PrettyInstructionSetLocal{Local: 0},
				// return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// let x = /storage/foo
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionNewPath{
					Domain: common.PathDomainStorage,
					Identifier: constant.DecodedConstant{
						Data: "foo",
						Kind: constant.RawString,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeStoragePath,
					TargetType: interpreter.PrimitiveStaticTypeStoragePath,
				},
				opcode.PrettyInstructionSetLocal{Local: 0},

				// return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
			[]opcode.PrettyInstruction{
				// let x = /public/foo
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionNewPath{
					Domain: common.PathDomainPublic,
					Identifier: constant.DecodedConstant{
						Data: "foo",
						Kind: constant.RawString,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypePublicPath,
					TargetType: interpreter.PrimitiveStaticTypePublicPath,
				},
				opcode.PrettyInstructionSetLocal{Local: 0},

				// return
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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

	// Ideally we would assert a concrete function type here,
	// but that would require a custom assertion function,
	// as function types are not directly comparable.
	functionType := program.Types[0]

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let x = fun() {}
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNewClosure{
				Function: "<anonymous>",
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  functionType,
				TargetType: functionType,
			},
			opcode.PrettyInstructionSetLocal{Local: 0},

			// return
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			// let x: Int? = nil
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionNil{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeNever,
				},
				TargetType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			opcode.PrettyInstructionSetLocal{Local: 0},

			// return
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	const (
		// fTypeIndex is the index of the type of function `f`, which is the first type
		fTypeIndex = iota //nolint:unused
		// testTypeIndex is the index of the type of function `test`, which is the second type
		testTypeIndex //nolint:unused
		// intTypeIndex is the index of the type int, which is the third type
		intTypeIndex
		voidTypeIndex
		// xParameterTypeIndex is the index of the type of parameter `x`, which is the fourth type
		xParameterTypeIndex
	)

	// xIndex is the index of the local variable `x`, which is the first local variable
	const xIndex = 0

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetLocal{Local: xIndex},

			// f(x)
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetGlobal{Global: 0},
			opcode.PrettyInstructionGetLocal{Local: xIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType: interpreter.PrimitiveStaticTypeInt,
				TargetType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: interpreter.PrimitiveStaticTypeVoid,
			},
			opcode.PrettyInstructionDrop{},

			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(functions[1].Code, program),
	)

	assertTypesEqual(
		t,
		[]bbq.StaticType{
			interpreter.FunctionStaticType{
				FunctionType: sema.NewSimpleFunctionType(
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
				FunctionType: sema.NewSimpleFunctionType(
					sema.FunctionPurityImpure,
					[]sema.Parameter{},
					sema.VoidTypeAnnotation,
				),
			},
			interpreter.PrimitiveStaticTypeInt,
			interpreter.PrimitiveStaticTypeVoid,
			&interpreter.OptionalStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
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
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionStatement{},
			// array[index]
			opcode.PrettyInstructionGetLocal{Local: arrayIndex},
			opcode.PrettyInstructionGetLocal{Local: indexIndex},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInteger,
			},
			// value + value
			opcode.PrettyInstructionGetLocal{Local: valueIndex},
			opcode.PrettyInstructionGetLocal{Local: valueIndex},
			opcode.PrettyInstructionAdd{},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetIndex{},

			// return
			opcode.PrettyInstructionReturn{},
		},
		prettyInstructions(testFunction.Code, program),
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
			//   opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1}
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
			//   opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1}
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

		contractsAddress := common.MustBytesToAddress([]byte{1})

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

		contractsAddress := common.MustBytesToAddress([]byte{1})

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
		require.Len(t, functions, 5)

		const (
			fooIndex = iota
			tempIndex
		)

		fooType := &interpreter.CompositeStaticType{
			Location:            checker.Location,
			QualifiedIdentifier: "Foo",
			TypeID:              checker.Location.TypeID(nil, "Foo"),
		}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let foo: Foo? = nil
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionNil{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType: &interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeNever,
					},
					TargetType: &interpreter.OptionalStaticType{
						Type: fooType,
					},
				},
				opcode.PrettyInstructionSetLocal{Local: fooIndex},

				opcode.PrettyInstructionStatement{},

				// Store the value in a temp index for the nil check.
				opcode.PrettyInstructionGetLocal{Local: fooIndex},
				opcode.PrettyInstructionSetLocal{
					Local:     tempIndex,
					IsTempVar: true,
				},

				// Nil check
				opcode.PrettyInstructionGetLocal{Local: tempIndex},
				opcode.PrettyInstructionJumpIfNil{Target: 13},

				// If `foo != nil`
				// Unwrap optional
				opcode.PrettyInstructionGetLocal{Local: tempIndex},
				opcode.PrettyInstructionUnwrap{},

				// foo.bar
				opcode.PrettyInstructionGetField{
					FieldName: constant.DecodedConstant{
						Data: "bar",
						Kind: constant.RawString,
					},
					AccessedType: fooType,
				},
				opcode.PrettyInstructionJump{Target: 14},

				// If `foo == nil`
				opcode.PrettyInstructionNil{},

				// Return value
				opcode.PrettyInstructionTransferAndConvert{
					ValueType: &interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeInt,
					},
					TargetType: &interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeInt,
					},
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[0].Code, program),
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
		require.Len(t, functions, 6)

		const (
			fooIndex = iota
			optionalValueTempIndex
		)

		fooType := &interpreter.CompositeStaticType{
			Location:            checker.Location,
			QualifiedIdentifier: "Foo",
			TypeID:              checker.Location.TypeID(nil, "Foo"),
		}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let foo: Foo? = nil
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionNil{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType: &interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeNever,
					},
					TargetType: &interpreter.OptionalStaticType{
						Type: fooType,
					},
				},
				opcode.PrettyInstructionSetLocal{Local: fooIndex},

				opcode.PrettyInstructionStatement{},

				// Store the receiver in a temp index for the nil check.
				opcode.PrettyInstructionGetLocal{Local: fooIndex},
				opcode.PrettyInstructionSetLocal{
					Local:     optionalValueTempIndex,
					IsTempVar: true,
				},

				// Nil check
				opcode.PrettyInstructionGetLocal{Local: optionalValueTempIndex},
				opcode.PrettyInstructionJumpIfNil{Target: 15},

				// If `foo != nil`
				// Unwrap the optional. (Loads receiver)
				opcode.PrettyInstructionGetLocal{Local: optionalValueTempIndex},
				opcode.PrettyInstructionUnwrap{},

				// Load `Foo.bar` function
				opcode.PrettyInstructionGetMethod{
					Method:       5,
					ReceiverType: fooType,
				},
				opcode.PrettyInstructionInvoke{
					ReturnType: interpreter.PrimitiveStaticTypeInt,
				},
				opcode.PrettyInstructionWrap{},
				opcode.PrettyInstructionJump{Target: 16},

				// If `foo == nil`
				opcode.PrettyInstructionNil{},

				// Return value
				opcode.PrettyInstructionTransferAndConvert{
					ValueType: &interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeInt,
					},
					TargetType: &interpreter.OptionalStaticType{
						Type: interpreter.PrimitiveStaticTypeInt,
					},
				},
				opcode.PrettyInstructionReturnValue{},
			},
			prettyInstructions(functions[0].Code, program),
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
		require.Len(t, functions, 5)

		const (
			xIndex = iota
			yIndex
			zIndex
		)

		rType := &interpreter.CompositeStaticType{
			Location:            checker.Location,
			QualifiedIdentifier: "R",
			TypeID:              checker.Location.TypeID(nil, "R"),
		}
		optionalRType := &interpreter.OptionalStaticType{Type: rType}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let x: @R <- create R()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{
					ReturnType: rType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: rType,
				},
				opcode.PrettyInstructionSetLocal{Local: xIndex},

				// var y: @R? <- create R()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{
					ReturnType: rType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: optionalRType,
				},
				opcode.PrettyInstructionSetLocal{Local: yIndex},

				opcode.PrettyInstructionStatement{},

				// Load `y` onto the stack.
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  optionalRType,
					TargetType: optionalRType,
				},

				// Second value assignment.
				// y <- x
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: optionalRType,
				},
				opcode.PrettyInstructionSetLocal{Local: yIndex},

				// Transfer and store the loaded y-value above, to z.
				// z <- y
				opcode.PrettyInstructionSetLocal{Local: zIndex},

				// destroy y
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionDestroy{},

				// destroy z
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: zIndex},
				opcode.PrettyInstructionDestroy{},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
		require.Len(t, functions, 5)

		const (
			xIndex = iota
			yIndex
			tempYIndex
			tempIndexingValueIndex
			zIndex
		)

		rType := &interpreter.CompositeStaticType{
			Location:            checker.Location,
			QualifiedIdentifier: "R",
			TypeID:              checker.Location.TypeID(nil, "R"),
		}
		stringType := interpreter.PrimitiveStaticTypeString
		dictionaryType := &interpreter.DictionaryStaticType{
			KeyType:   stringType,
			ValueType: rType,
		}
		optionalRType := &interpreter.OptionalStaticType{Type: rType}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let x: @R <- create R()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{
					ReturnType: rType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: rType,
				},
				opcode.PrettyInstructionSetLocal{Local: xIndex},

				// var y <- {"r" : <- create R()}
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue("r"),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  stringType,
					TargetType: stringType,
				},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{
					ReturnType: rType,
				},
				opcode.PrettyInstructionTransfer{},
				opcode.PrettyInstructionConvert{
					ValueType:  rType,
					TargetType: rType,
				},
				opcode.PrettyInstructionNewDictionary{
					Type:       dictionaryType,
					Size:       1,
					IsResource: true,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  dictionaryType,
					TargetType: dictionaryType,
				},
				opcode.PrettyInstructionSetLocal{Local: yIndex},

				opcode.PrettyInstructionStatement{},

				// <- y["r"]

				// Evaluate `y` and store in a temp local.
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionSetLocal{
					Local:     tempYIndex,
					IsTempVar: true,
				},

				// evaluate "r", and store in a temp local.
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue("r"),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionSetLocal{
					Local:     tempIndexingValueIndex,
					IsTempVar: true,
				},

				// Evaluate the index expression, `y["r"]`, using temp locals.
				opcode.PrettyInstructionGetLocal{Local: tempYIndex},
				opcode.PrettyInstructionGetLocal{Local: tempIndexingValueIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  stringType,
					TargetType: stringType,
				},
				opcode.PrettyInstructionRemoveIndex{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  optionalRType,
					TargetType: optionalRType,
				},

				// Second value assignment.
				// y["r"] <- x
				// `y` and "r" must be loaded from temp locals.
				opcode.PrettyInstructionGetLocal{Local: tempYIndex},
				opcode.PrettyInstructionGetLocal{Local: tempIndexingValueIndex},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: optionalRType,
				},
				opcode.PrettyInstructionSetIndex{},

				// Store the transferred y-value above (already on stack), to z.
				// z <- y["r"]
				opcode.PrettyInstructionSetLocal{Local: zIndex},

				// destroy y
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionDestroy{},

				// destroy z
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: zIndex},
				opcode.PrettyInstructionDestroy{},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
		require.Len(t, functions, 9)

		const (
			xIndex = iota
			yIndex
			tempYIndex
			zIndex
		)

		barType := &interpreter.CompositeStaticType{
			Location:            checker.Location,
			QualifiedIdentifier: "Bar",
			TypeID:              checker.Location.TypeID(nil, "Bar"),
		}
		fooType := &interpreter.CompositeStaticType{
			Location:            checker.Location,
			QualifiedIdentifier: "Foo",
			TypeID:              checker.Location.TypeID(nil, "Foo"),
		}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let x: @R <- create R()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 5},
				opcode.PrettyInstructionInvoke{
					ReturnType: barType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  barType,
					TargetType: barType,
				},
				opcode.PrettyInstructionSetLocal{Local: xIndex},

				// var y <- {"r" : <- create R()}
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{
					ReturnType: fooType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  fooType,
					TargetType: fooType,
				},
				opcode.PrettyInstructionSetLocal{Local: yIndex},

				opcode.PrettyInstructionStatement{},

				// <- y.bar

				// Evaluate `y` and store in a temp local.
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionSetLocal{
					Local:     tempYIndex,
					IsTempVar: true,
				},

				// Evaluate the member access, `y.bar`, using temp local.
				opcode.PrettyInstructionGetLocal{Local: tempYIndex},
				opcode.PrettyInstructionRemoveField{
					FieldName: constant.DecodedConstant{
						Data: "bar",
						Kind: constant.RawString,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  barType,
					TargetType: barType,
				},

				// Second value assignment.
				//  `y.bar <- x`
				// `y` must be loaded from the temp local.
				opcode.PrettyInstructionGetLocal{Local: tempYIndex},
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  barType,
					TargetType: barType,
				},
				opcode.PrettyInstructionSetField{
					FieldName: constant.DecodedConstant{
						Data: "bar",
						Kind: constant.RawString,
					},
					AccessedType: fooType,
				},

				// Store the transferred y-value above (already on stack), to z.
				// z <- y.bar
				opcode.PrettyInstructionSetLocal{Local: zIndex},

				// destroy y
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionDestroy{},

				// destroy z
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: zIndex},
				opcode.PrettyInstructionDestroy{},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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
		require.Len(t, functions, 5)

		const (
			xIndex = iota
			yIndex
			tempIndex
			zIndex
			resIndex
		)

		rType := &interpreter.CompositeStaticType{
			Location:            checker.Location,
			QualifiedIdentifier: "R",
			TypeID:              checker.Location.TypeID(nil, "R"),
		}
		optionalRType := &interpreter.OptionalStaticType{Type: rType}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				// let x: @R <- create R()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{
					ReturnType: rType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: rType,
				},
				opcode.PrettyInstructionSetLocal{Local: xIndex},

				// var y: @R? <- create R()
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionInvoke{
					ReturnType: rType,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: optionalRType,
				},
				opcode.PrettyInstructionSetLocal{Local: yIndex},

				opcode.PrettyInstructionStatement{},

				// store y in temp index for nil check
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  optionalRType,
					TargetType: optionalRType,
				},

				// Second value assignment. Store `x` in `y`.
				// y <- x
				opcode.PrettyInstructionGetLocal{Local: xIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: optionalRType,
				},
				opcode.PrettyInstructionSetLocal{Local: yIndex},

				// Store the previously loaded `y`s old value on the temp local.
				opcode.PrettyInstructionSetLocal{
					Local:     tempIndex,
					IsTempVar: true,
				},

				// nil check on temp y.
				opcode.PrettyInstructionGetLocal{Local: tempIndex},
				opcode.PrettyInstructionJumpIfNil{Target: 29},

				// If not-nil, transfer the temp `y` and store in `z` (i.e: y <- y)
				opcode.PrettyInstructionGetLocal{Local: tempIndex},
				opcode.PrettyInstructionUnwrap{},
				opcode.PrettyInstructionSetLocal{Local: zIndex},

				// let res: @R <- z
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: zIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  rType,
					TargetType: rType,
				},
				opcode.PrettyInstructionSetLocal{Local: resIndex},

				// destroy res
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: resIndex},
				opcode.PrettyInstructionDestroy{},

				// destroy y
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetLocal{Local: yIndex},
				opcode.PrettyInstructionDestroy{},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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

	const rawValueTypeIndex = 4

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
					Type: 1,
				},
				opcode.InstructionSetLocal{Local: selfIndex},

				// self.rawValue = rawValue
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: selfIndex},
				opcode.InstructionGetLocal{Local: rawValueIndex},
				opcode.InstructionTransferAndConvert{ValueType: rawValueTypeIndex, TargetType: rawValueTypeIndex},
				opcode.InstructionSetField{FieldName: 3, AccessedType: 1},

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
				opcode.InstructionSetLocal{
					Local:     tempIndex,
					IsTempVar: true,
				},

				// switch temp

				// case 1:
				opcode.InstructionGetLocal{Local: tempIndex},
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionEqual{},
				opcode.InstructionJumpIfFalse{Target: 12},

				// return Test.a
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: testAGlobalIndex},
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 2},
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
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 2},
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
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 2},
				opcode.InstructionReturnValue{},
				opcode.InstructionJump{Target: 34},

				// default:
				// return nil
				opcode.InstructionStatement{},
				opcode.InstructionNil{},
				opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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
			opcode.InstructionGetField{FieldName: 3, AccessedType: 1},
			opcode.InstructionTransferAndConvert{ValueType: rawValueTypeIndex, TargetType: rawValueTypeIndex},
			opcode.InstructionReturnValue{},
		},
		functions[testFuncIndex].Code,
	)

	testType := &interpreter.CompositeStaticType{
		Location:            checker.Location,
		QualifiedIdentifier: "Test",
		TypeID:              checker.Location.TypeID(nil, "Test"),
	}

	{
		// rawValueIndex is the index of the parameter `rawValue`, which is the first parameter
		const rawValueIndex = iota

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: testLookupGlobalIndex},
				opcode.PrettyInstructionGetLocal{Local: rawValueIndex},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeUInt8,
					TargetType: interpreter.PrimitiveStaticTypeUInt8,
				},
				opcode.PrettyInstructionInvoke{
					ArgCount: 1,
					ReturnType: &interpreter.OptionalStaticType{
						Type: testType,
					},
				},
				opcode.PrettyInstructionDrop{},
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[test2FuncIndex].Code, program),
		)
	}

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionGetGlobal{Global: testConstructorGlobalIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredUInt8Value(0),
					Kind: constant.UInt8,
				},
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: testType,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(variables[testAVarIndex].Getter.Code, program),
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionGetGlobal{Global: testConstructorGlobalIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredUInt8Value(1),
					Kind: constant.UInt8,
				},
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: testType,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(variables[testBVarIndex].Getter.Code, program),
	)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			opcode.PrettyInstructionGetGlobal{Global: testConstructorGlobalIndex},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredUInt8Value(2),
					Kind: constant.UInt8,
				},
			},
			opcode.PrettyInstructionInvoke{
				ArgCount:   1,
				ReturnType: testType,
			},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(variables[testCVarIndex].Getter.Code, program),
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
			[]opcode.PrettyInstruction{
				// assert(true, message: "hello")
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionTrue{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeBool,
					TargetType: interpreter.PrimitiveStaticTypeBool,
				},
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue("hello"),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionInvokeTyped{
					ArgTypes: []interpreter.StaticType{
						interpreter.PrimitiveStaticTypeBool,
						interpreter.PrimitiveStaticTypeString,
					},
					ReturnType: interpreter.PrimitiveStaticTypeVoid,
				},
				opcode.PrettyInstructionDrop{},

				// assert(false)
				opcode.PrettyInstructionStatement{},
				opcode.PrettyInstructionGetGlobal{Global: 1},
				opcode.PrettyInstructionFalse{},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeBool,
					TargetType: interpreter.PrimitiveStaticTypeBool,
				},
				opcode.PrettyInstructionInvokeTyped{
					ArgTypes: []interpreter.StaticType{
						interpreter.PrimitiveStaticTypeBool,
					},
					ReturnType: interpreter.PrimitiveStaticTypeVoid,
				},
				opcode.PrettyInstructionDrop{},
				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[0].Code, program),
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

		contractsAddress := common.MustBytesToAddress([]byte{1})

		aLocation := common.NewAddressLocation(nil, contractsAddress, "A")

		program := ParseCheckAndCompile(
			t,
			aContract,
			aLocation,
			programs,
		)

		functions := program.Functions
		require.Len(t, functions, 4)

		aType := &interpreter.CompositeStaticType{
			Location:            aLocation,
			QualifiedIdentifier: "A",
			TypeID:              aLocation.TypeID(nil, "A"),
		}

		// Ideally we would assert a concrete type here,
		// but constructing it manually is non-trivial
		accountContractsReferenceType := program.Types[7]

		uint8ArrayType := &interpreter.VariableSizedStaticType{
			Type: interpreter.PrimitiveStaticTypeUInt8,
		}

		assert.Equal(t,
			[]opcode.PrettyInstruction{
				opcode.PrettyInstructionStatement{},

				// Load receiver `self.account.contracts`.
				opcode.PrettyInstructionGetLocal{Local: 0},
				opcode.PrettyInstructionGetField{
					FieldName: constant.DecodedConstant{
						Data: "account",
						Kind: constant.RawString,
					},
					AccessedType: aType,
				},
				opcode.PrettyInstructionGetField{
					FieldName: constant.DecodedConstant{
						Data: "contracts",
						Kind: constant.RawString,
					},
					AccessedType: interpreter.NewReferenceStaticType(
						nil,
						interpreter.FullyEntitledAccountAccess,
						interpreter.PrimitiveStaticTypeAccount,
					),
				},
				opcode.PrettyInstructionNewRef{
					Type:       accountContractsReferenceType,
					IsImplicit: true,
				},

				// Load function value `add()`
				opcode.PrettyInstructionGetMethod{
					Method:       5,
					ReceiverType: accountContractsReferenceType,
				},

				// Load arguments.

				// Name: "Foo",
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue("Foo"),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  interpreter.PrimitiveStaticTypeString,
					TargetType: interpreter.PrimitiveStaticTypeString,
				},

				// Contract code
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue(
							" contract Foo { let message: String\n init(message:String) {self.message = message}\nfun test(): String {return self.message}}",
						),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionGetField{
					FieldName: constant.DecodedConstant{
						Data: "utf8",
						Kind: constant.RawString,
					},
					AccessedType: interpreter.PrimitiveStaticTypeString,
				},
				opcode.PrettyInstructionTransferAndConvert{
					ValueType:  uint8ArrayType,
					TargetType: uint8ArrayType,
				},

				// Message: "Optional arg"
				opcode.PrettyInstructionGetConstant{
					Constant: constant.DecodedConstant{
						Data: interpreter.NewUnmeteredStringValue("Optional arg"),
						Kind: constant.String,
					},
				},
				opcode.PrettyInstructionTransfer{},

				opcode.PrettyInstructionInvokeTyped{
					ArgTypes: []interpreter.StaticType{
						interpreter.PrimitiveStaticTypeString,
						uint8ArrayType,
						interpreter.PrimitiveStaticTypeString,
					},
					ReturnType: interpreter.PrimitiveStaticTypeDeployedContract,
				},
				opcode.PrettyInstructionDrop{},

				opcode.PrettyInstructionReturn{},
			},
			prettyInstructions(functions[3].Code, program),
		)

		assert.Equal(t,
			[]constant.DecodedConstant{
				{
					Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{1}),
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

	contractsAddress := common.MustBytesToAddress([]byte{1})

	aLocation := common.NewAddressLocation(nil, contractsAddress, "A")

	program := ParseCheckAndCompile(
		t,
		aContract,
		aLocation,
		programs,
	)

	functions := program.Functions
	require.Len(t, functions, 3)

	assert.Equal(t,
		[]opcode.PrettyInstruction{
			// let self = A()
			opcode.PrettyInstructionNewCompositeAt{
				Kind: common.CompositeKindContract,
				Type: &interpreter.CompositeStaticType{
					Location:            aLocation,
					QualifiedIdentifier: "A",
					TypeID:              aLocation.TypeID(nil, "A"),
				},
				Address: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{1}),
					Kind: constant.Address,
				},
			},
			opcode.PrettyInstructionDup{},
			opcode.PrettyInstructionSetGlobal{Global: 0},
			opcode.PrettyInstructionSetLocal{Local: 0},

			// self.x = 1
			opcode.PrettyInstructionStatement{},
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: interpreter.NewUnmeteredIntValueFromInt64(1),
					Kind: constant.Int,
				},
			},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeInt,
				TargetType: interpreter.PrimitiveStaticTypeInt,
			},
			opcode.PrettyInstructionSetField{
				FieldName: constant.DecodedConstant{
					Data: "x",
					Kind: constant.RawString,
				},
				AccessedType: &interpreter.CompositeStaticType{
					Location:            aLocation,
					QualifiedIdentifier: "A",
					TypeID:              aLocation.TypeID(nil, "A"),
				},
			},

			// return self
			opcode.PrettyInstructionGetLocal{Local: 0},
			opcode.PrettyInstructionReturnValue{},
		},
		prettyInstructions(functions[0].Code, program),
	)

	assert.Equal(t,
		[]constant.DecodedConstant{
			{
				Data: interpreter.NewUnmeteredAddressValueFromBytes([]byte{1}),
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
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: xIndex},

			// var y = 2
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: yIndex},

			// x <-> y
			opcode.InstructionStatement{},

			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionSetLocal{
				Local:     tempIndex1,
				IsTempVar: true,
			},

			opcode.InstructionGetLocal{Local: yIndex},
			opcode.InstructionSetLocal{
				Local:     tempIndex2,
				IsTempVar: true,
			},

			// get left (x)
			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionSetLocal{
				Local:     tempIndex3,
				IsTempVar: true,
			},

			// get right (y)
			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionSetLocal{
				Local:     tempIndex4,
				IsTempVar: true,
			},

			// convert right value to left type
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: tempIndex4},

			// convert left value to right type
			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: tempIndex3},

			// set left (x) with right value
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionSetLocal{Local: xIndex},

			// set right (y) with left value
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
	require.Len(t, functions, 5)

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
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: sIndex},

			// s.x <-> s.y
			opcode.InstructionStatement{},

			opcode.InstructionGetLocal{Local: sIndex},
			opcode.InstructionSetLocal{
				Local:     tempIndex1,
				IsTempVar: true,
			},

			opcode.InstructionGetLocal{Local: sIndex},
			opcode.InstructionSetLocal{
				Local:     tempIndex2,
				IsTempVar: true,
			},

			// get left (s.x)
			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetField{FieldName: 0, AccessedType: 1},
			opcode.InstructionSetLocal{
				Local:     tempIndex3,
				IsTempVar: true,
			},

			// get right (s.y)
			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionGetField{FieldName: 1, AccessedType: 1},
			opcode.InstructionSetLocal{
				Local:     tempIndex4,
				IsTempVar: true,
			},

			// convert right value to left type
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionSetLocal{Local: tempIndex4},

			// convert left value to right type
			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionSetLocal{Local: tempIndex3},

			// set left (s.x) with right value
			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionSetField{FieldName: 0, AccessedType: 1},

			// set right (s.y) with left value
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

func TestCompileSwapIndexInStructs(t *testing.T) {

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
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionNewArray{Type: 1, Size: 2},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: charsIndex},

			// chars[0] <-> chars[1]
			opcode.InstructionStatement{},

			opcode.InstructionGetLocal{Local: charsIndex},
			opcode.InstructionSetLocal{
				Local:     tempIndex1,
				IsTempVar: true,
			},

			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{ValueType: 3, TargetType: 3},
			opcode.InstructionSetLocal{
				Local:     tempIndex2,
				IsTempVar: true,
			},

			opcode.InstructionGetLocal{Local: charsIndex},
			opcode.InstructionSetLocal{
				Local:     tempIndex3,
				IsTempVar: true,
			},

			opcode.InstructionGetConstant{Constant: 3},
			opcode.InstructionTransferAndConvert{ValueType: 3, TargetType: 3},
			opcode.InstructionSetLocal{
				Local:     tempIndex4,
				IsTempVar: true,
			},

			// get left value
			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionGetIndex{},
			opcode.InstructionSetLocal{
				Local:     tempIndex5,
				IsTempVar: true,
			},

			// get right value
			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionGetLocal{Local: tempIndex4},
			opcode.InstructionGetIndex{},
			opcode.InstructionSetLocal{
				Local:     tempIndex6,
				IsTempVar: true,
			},

			// convert right value to left type
			opcode.InstructionGetLocal{Local: tempIndex6},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionSetLocal{Local: tempIndex6},

			// convert left value to right type
			opcode.InstructionGetLocal{Local: tempIndex5},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionSetLocal{Local: tempIndex5},

			// set right index with left value
			opcode.InstructionGetLocal{Local: tempIndex1},
			opcode.InstructionGetLocal{Local: tempIndex2},
			opcode.InstructionGetLocal{Local: tempIndex6},
			opcode.InstructionSetIndex{},

			// set left index with right value
			opcode.InstructionGetLocal{Local: tempIndex3},
			opcode.InstructionGetLocal{Local: tempIndex4},
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

func TestCompileSwapIndexInResources(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        resource R {}

        fun test() {
           let rs <- [
               <- create R()
           ]

           // We swap only '0'
           rs[0] <-> rs[0]

           destroy rs
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
		rsIndex = iota
		leftTargetIndex
		leftIndexIndex
		rightTargetIndex
		rightIndexIndex
		leftInsertedPlaceholderIndex
		leftValueIndex
		rightValueIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},

			// let rs <- [<- create R()]
			opcode.InstructionGetGlobal{Global: 1},
			opcode.InstructionInvoke{ReturnType: 2},
			opcode.InstructionTransfer{},
			opcode.InstructionConvert{ValueType: 2, TargetType: 2},

			opcode.InstructionNewArray{
				Type:       1,
				Size:       1,
				IsResource: true,
			},
			opcode.InstructionTransferAndConvert{
				ValueType:  1,
				TargetType: 1,
			},
			opcode.InstructionSetLocal{
				Local: rsIndex,
			},

			// rs[0] <-> rs[0]
			opcode.InstructionStatement{},

			opcode.InstructionGetLocal{Local: rsIndex},
			opcode.InstructionSetLocal{
				Local:     leftTargetIndex,
				IsTempVar: true,
			},

			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{
				ValueType:  3,
				TargetType: 3,
			},
			opcode.InstructionSetLocal{
				Local:     leftIndexIndex,
				IsTempVar: true,
			},

			opcode.InstructionGetLocal{Local: rsIndex},
			opcode.InstructionSetLocal{
				Local:     rightTargetIndex,
				IsTempVar: true,
			},

			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{
				ValueType:  3,
				TargetType: 3,
			},
			opcode.InstructionSetLocal{
				Local:     rightIndexIndex,
				IsTempVar: true,
			},

			// get left value
			opcode.InstructionGetLocal{Local: leftTargetIndex},
			opcode.InstructionGetLocal{Local: leftIndexIndex},
			opcode.InstructionRemoveIndex{PushPlaceholder: true},
			opcode.InstructionSetLocal{Local: leftInsertedPlaceholderIndex},
			opcode.InstructionSetLocal{
				Local:     leftValueIndex,
				IsTempVar: true,
			},

			// get right value
			opcode.InstructionGetLocal{Local: rightTargetIndex},
			opcode.InstructionGetLocal{Local: rightIndexIndex},
			opcode.InstructionRemoveIndex{PushPlaceholder: false},
			opcode.InstructionSetLocal{
				Local:     rightValueIndex,
				IsTempVar: true,
			},

			// compare right value and left inserted placeholder
			opcode.InstructionGetLocal{Local: rightValueIndex},
			opcode.InstructionGetLocal{Local: leftInsertedPlaceholderIndex},
			opcode.InstructionSame{},
			opcode.InstructionJumpIfFalse{Target: 37},

			// set left index back with left value
			opcode.InstructionGetLocal{Local: leftTargetIndex},
			opcode.InstructionGetLocal{Local: leftIndexIndex},
			opcode.InstructionGetLocal{Local: leftValueIndex},
			opcode.InstructionSetIndex{},

			// jump to the end
			opcode.InstructionJump{Target: 51},

			// convert right value to left type
			opcode.InstructionGetLocal{Local: rightValueIndex},
			opcode.InstructionTransferAndConvert{
				ValueType:  2,
				TargetType: 2,
			},
			opcode.InstructionSetLocal{Local: rightValueIndex},

			// convert left value to right type
			opcode.InstructionGetLocal{Local: leftValueIndex},
			opcode.InstructionTransferAndConvert{
				ValueType:  2,
				TargetType: 2,
			},
			opcode.InstructionSetLocal{Local: leftValueIndex},

			// set left index with right value
			opcode.InstructionGetLocal{Local: leftTargetIndex},
			opcode.InstructionGetLocal{Local: leftIndexIndex},
			opcode.InstructionGetLocal{Local: rightValueIndex},
			opcode.InstructionSetIndex{},

			// set right index with left value
			opcode.InstructionGetLocal{Local: rightTargetIndex},
			opcode.InstructionGetLocal{Local: rightIndexIndex},
			opcode.InstructionGetLocal{Local: leftValueIndex},
			opcode.InstructionSetIndex{},

			// destroy rs
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: rsIndex},
			opcode.InstructionDestroy{},

			// Return
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
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
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
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
				opcode.InstructionSetLocal{Local: 0},
				// let b = "B"
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 1},
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
				opcode.InstructionSetLocal{Local: 1},
				// let c = 4
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
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
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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
			opcode.InstructionTransferAndConvert{ValueType: 3, TargetType: 3},
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
			opcode.InstructionInvoke{ReturnType: 2},
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
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
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
				opcode.InstructionTransferAndConvert{ValueType: 3, TargetType: 3},
				opcode.InstructionInvoke{ArgCount: 1, ReturnType: 2},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
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
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
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
				opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 6},

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
				opcode.InstructionJumpIfFalse{Target: 21},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 2}, // error message
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
				opcode.InstructionInvoke{ArgCount: 1, ReturnType: 3},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
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
				opcode.InstructionTransferAndConvert{ValueType: 3, TargetType: 3},
				opcode.InstructionInvoke{ArgCount: 1, ReturnType: 2},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
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
				opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
				opcode.InstructionSetLocal{Local: tempResultIndex},

				// jump to post conditions
				opcode.InstructionJump{Target: 6},

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
				opcode.InstructionJumpIfFalse{Target: 21},

				// $failPostCondition("")
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: 1},     // global index 1 is 'panic' function
				opcode.InstructionGetConstant{Constant: 2}, // error message
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
				opcode.InstructionInvoke{ArgCount: 1, ReturnType: 3},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return $_result
				opcode.InstructionGetLocal{Local: tempResultIndex},
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
				opcode.InstructionTransferAndConvert{ValueType: 3, TargetType: 3},
				opcode.InstructionInvoke{ArgCount: 1, ReturnType: 2},

				// Drop since it's a statement-expression
				opcode.InstructionDrop{},

				// return 5
				opcode.InstructionStatement{},
				opcode.InstructionGetConstant{Constant: 2},
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
				opcode.InstructionReturnValue{},
			},
			functions[anonymousFunctionIndex].Code,
		)
	})

}

func TestCompileAttachments(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			struct S {}
			attachment A for S {
				let x: Int
				init(x: Int) {
					self.x = x
				}
				fun foo(): Int { return self.x }
			}
			fun test(): Int {
				var s = S()
				s = attach A(x: 3) to s
				return s[A]?.foo()!
			}
		`)
		require.NoError(t, err)

		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		program := comp.Compile()

		functions := program.Functions
		require.Len(t, functions, 9)

		// global functions
		const (
			sConstructorGlobalIndex = 1
			aConstructorGlobalIndex = 5
		)

		// local variables
		const (
			sLocalIndex = iota
			sTmpLocalIndex
			sRefLocalIndex
			attachmentLocalIndex
		)

		assert.Equal(t,
			[]opcode.Instruction{
				// STATEMENT: var s = S()
				opcode.InstructionStatement{},
				opcode.InstructionGetGlobal{Global: sConstructorGlobalIndex},
				opcode.InstructionInvoke{ReturnType: 1},
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
				opcode.InstructionSetLocal{Local: sLocalIndex},

				// STATEMENT: s = attach A(x:3) to s
				opcode.InstructionStatement{},
				// get s on stack
				opcode.InstructionGetLocal{Local: sLocalIndex},
				// store s in a separate local, put on stack
				opcode.InstructionSetLocal{
					Local:     sTmpLocalIndex,
					IsTempVar: true,
				},
				opcode.InstructionGetLocal{Local: sTmpLocalIndex},
				// create a reference to s and store locally
				opcode.InstructionNewRef{Type: 2, IsImplicit: false},
				opcode.InstructionSetLocal{
					Local:     sRefLocalIndex,
					IsTempVar: true,
				},
				// get A constructor
				opcode.InstructionGetGlobal{Global: aConstructorGlobalIndex},
				// get 3
				opcode.InstructionGetConstant{Constant: 0},
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
				// get s reference
				opcode.InstructionGetLocal{Local: sRefLocalIndex},
				// invoke A constructor with &s as arg, puts A on stack
				opcode.InstructionInvoke{ArgCount: 2, ReturnType: 3},
				// get s back on stack
				opcode.InstructionGetLocal{Local: sTmpLocalIndex},
				// attachment operation, attach A to s-copy
				opcode.InstructionSetTypeIndex{Type: 3},
				// return value is s-copy
				opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
				// finish assignment of s
				opcode.InstructionSetLocal{Local: sLocalIndex},

				// STATEMENT: return s[A]?.foo()!
				opcode.InstructionStatement{},
				opcode.InstructionGetLocal{Local: sLocalIndex},
				// access A on s: s[A], returns attachment reference as optional
				opcode.InstructionGetTypeIndex{Type: 3},
				opcode.InstructionSetLocal{
					Local:     attachmentLocalIndex,
					IsTempVar: true,
				},
				opcode.InstructionGetLocal{Local: attachmentLocalIndex},
				opcode.InstructionJumpIfNil{Target: 32},
				opcode.InstructionGetLocal{Local: attachmentLocalIndex},
				opcode.InstructionUnwrap{},
				// call foo if not nil
				opcode.InstructionGetMethod{Method: 8, ReceiverType: 5},
				opcode.InstructionInvoke{ReturnType: 4},
				opcode.InstructionWrap{},
				opcode.InstructionJump{Target: 33},
				opcode.InstructionNil{},
				opcode.InstructionUnwrap{},
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
				opcode.InstructionReturnValue{},
			},
			functions[0].Code,
		)

		// local variables
		const (
			xLocalIndex = iota
			baseLocalIndex
			selfLocalIndex
			returnLocalIndex
		)

		// `A` init
		assert.Equal(t,
			[]opcode.Instruction{
				// create attachment
				opcode.InstructionNewComposite{Kind: 6, Type: 3},
				// set returnLocalIndex to attachment
				opcode.InstructionSetLocal{Local: returnLocalIndex},
				// set base to be the attachment
				opcode.InstructionGetLocal{Local: baseLocalIndex},
				opcode.InstructionGetLocal{Local: returnLocalIndex},
				opcode.InstructionSetAttachmentBase{},
				// get a reference to attachment
				opcode.InstructionGetLocal{Local: returnLocalIndex},
				// set self to be the reference
				opcode.InstructionNewRef{Type: 5, IsImplicit: false},
				opcode.InstructionSetLocal{Local: selfLocalIndex},

				// self.x = x
				opcode.InstructionStatement{},
				// get self
				opcode.InstructionGetLocal{Local: selfLocalIndex},
				// get x
				opcode.InstructionGetLocal{Local: xLocalIndex},
				opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
				// set self.x = x
				opcode.InstructionSetField{FieldName: 1, AccessedType: 5},
				// return created attachment (returnLocalIndex)
				opcode.InstructionGetLocal{Local: returnLocalIndex},
				opcode.InstructionReturnValue{},
			},
			functions[5].Code,
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

	contractsAddress := common.MustBytesToAddress([]byte{1})

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
	require.Len(t, functions, 4)

	const (
		siIndex = iota
		tempIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: siIndex},
			opcode.InstructionSetLocal{
				Local:     tempIndex,
				IsTempVar: true,
			},
			opcode.InstructionGetLocal{Local: tempIndex},
			opcode.InstructionJumpIfNil{Target: 11},
			opcode.InstructionGetLocal{Local: tempIndex},
			opcode.InstructionUnwrap{},
			opcode.InstructionGetField{
				FieldName:    0,
				AccessedType: 2,
			},
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionWrap{},
			opcode.InstructionJump{Target: 12},
			opcode.InstructionNil{},
			opcode.InstructionTransferAndConvert{ValueType: 3, TargetType: 3},
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

	require.Equal(t, aTestFunction.QualifiedName, "A.test")

	assert.Equal(t,
		[]opcode.Instruction{
			// return B.c(B.d)
			opcode.InstructionStatement{},
			// B.c(...)
			opcode.InstructionGetGlobal{Global: 5},
			opcode.InstructionGetMethod{Method: 6, ReceiverType: 6},
			// B.d
			opcode.InstructionGetGlobal{Global: 5},
			opcode.InstructionGetField{
				FieldName:    0,
				AccessedType: 6,
			},
			opcode.InstructionTransferAndConvert{ValueType: 5, TargetType: 5},
			opcode.InstructionInvoke{ArgCount: 1, ReturnType: 5},
			opcode.InstructionTransferAndConvert{ValueType: 5, TargetType: 5},
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
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionSetLocal{Local: xIndex},

			// for y in [1]
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
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

	contractsAddress := common.MustBytesToAddress([]byte{1})

	barLocation := common.NewAddressLocation(nil, contractsAddress, "Bar")
	fooLocation := common.NewAddressLocation(nil, contractsAddress, "Foo")

	barProgram := ParseCheckAndCompile(
		t,
		barContract,
		barLocation,
		programs,
	)

	functions := barProgram.Functions
	require.Len(t, functions, 8)

	defaultDestroyEventConstructor := functions[5]
	require.Equal(t, "Bar.XYZ.ResourceDestroyed", defaultDestroyEventConstructor.Name)

	const (
		xIndex = iota
		selfIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			// Create a `Bar.XYZ.ResourceDestroyed` event value.
			opcode.InstructionNewComposite{Kind: 4, Type: 4},
			opcode.InstructionSetLocal{Local: selfIndex},
			opcode.InstructionStatement{},

			// Set the parameter to the field.
			//  `self.x = x`
			opcode.InstructionGetLocal{Local: selfIndex},
			opcode.InstructionGetLocal{Local: xIndex},
			opcode.InstructionTransferAndConvert{ValueType: 5, TargetType: 5},
			opcode.InstructionSetField{
				FieldName:    0,
				AccessedType: 4,
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
	require.Len(t, functions, 12)

	defaultDestroyEventEmittingFunction := functions[8]
	require.Equal(t, "Foo.ABC.$ResourceDestroyed", defaultDestroyEventEmittingFunction.QualifiedName)

	const inheritedEventConstructorIndex = 10
	const selfDefinedABCEventConstructorIndex = 13

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
			opcode.InstructionTransferAndConvert{ValueType: 7, TargetType: 7},
			opcode.InstructionInvoke{ArgCount: 1, ReturnType: 11},

			// Construct the self defined event
			// Foo.ABC.ResourceDestroyed(self.x)
			opcode.InstructionGetGlobal{Global: selfDefinedABCEventConstructorIndex},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetField{FieldName: 2, AccessedType: 13},
			opcode.InstructionTransferAndConvert{ValueType: 7, TargetType: 7},
			opcode.InstructionInvoke{ArgCount: 1, ReturnType: 12},

			// Invoke `collectEvents` with the above event.
			// `collectEvents(...)`
			opcode.InstructionInvoke{ArgCount: 2, ReturnType: 10},
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

		t.Parallel()

		compiledPrograms := CompiledPrograms{}

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{1}),
			Name:    "Foo",
		}

		_ = ParseCheckAndCompile(t,
			`
				contract Foo {
					fun hello(): String {
						return "hello"
					}
				}
            `,
			importLocation,
			compiledPrograms,
		)

		program := ParseCheckAndCompile(t,
			`
				import Foo as Bar from 0x01

				fun test(): String {
					return Bar.hello()
				}
            `,
			TestLocation,
			compiledPrograms,
		)
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
			[]bbq.GlobalInfo{
				{
					Location:      nil,
					Name:          "test",
					QualifiedName: "test",
					Index:         0,
				},
				{
					Location:      importLocation,
					Name:          "Foo",
					QualifiedName: "A.0000000000000001.Foo",
					Index:         1,
				},
				{
					Location:      importLocation,
					Name:          "Foo.hello",
					QualifiedName: "A.0000000000000001.Foo.hello",
					Index:         2,
				},
			},
			program.Globals,
		)

	})

	t.Run("interface", func(t *testing.T) {

		t.Parallel()

		compiledPrograms := CompiledPrograms{}

		importLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{1}),
			Name:    "FooInterface",
		}

		_ = ParseCheckAndCompile(t,
			`
				struct interface FooInterface {
					fun hello(): String

					fun defaultHello(): String {
						return "hi"
					}
				}
            `,
			importLocation,
			compiledPrograms,
		)

		barLocation := common.AddressLocation{
			Address: common.MustBytesToAddress([]byte{1}),
			Name:    "Bar",
		}

		program := ParseCheckAndCompile(t,
			`
				import FooInterface as FI from 0x01

				struct Bar: FI {
					fun hello(): String {
						return "hello"
					}
				}
            `,
			barLocation,
			compiledPrograms,
		)

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
			[]bbq.GlobalInfo{
				{
					Location:      nil,
					Name:          "Bar",
					QualifiedName: "Bar",
					Index:         0,
				},
				{
					Location:      nil,
					Name:          "Bar.getType",
					QualifiedName: "Bar.getType",
					Index:         1,
				},
				{
					Location:      nil,
					Name:          "Bar.isInstance",
					QualifiedName: "Bar.isInstance",
					Index:         2,
				},
				{
					Location:      nil,
					Name:          "Bar.forEachAttachment",
					QualifiedName: "Bar.forEachAttachment",
					Index:         3,
				},
				{
					Location:      nil,
					Name:          "Bar.hello",
					QualifiedName: "Bar.hello",
					Index:         4,
				},
				{
					Location:      nil,
					Name:          "Bar.defaultHello",
					QualifiedName: "Bar.defaultHello",
					Index:         5,
				},
				{
					Location:      importLocation,
					Name:          "FooInterface.defaultHello",
					QualifiedName: "A.0000000000000001.FooInterface.defaultHello",
					Index:         6,
				},
			},
			program.Globals,
		)
	})
}

func TestPeepholeOptimizer(t *testing.T) {
	t.Parallel()

	t.Run("constant transfer and convert", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			fun test(): Int {
				let x = 1
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

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			// this transfer can be optimized out
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionReturnValue{},
		},
			functions[0].Code,
		)

		comp2 := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp2.Config.PeepholeOptimizationsEnabled = true
		program2 := comp2.Compile()

		functions2 := program2.Functions
		require.Len(t, functions2, 1)

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			// transfer gone
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionReturnValue{},
		}, functions2[0].Code)
	})

	t.Run("patch jumps", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			fun test2(): Int {
				return 32
			}
			fun test(): Int {
				var x = 0
				var y = test2()
				if x > 0 {
					y = test2()
				} else {
					y = 64
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

		functions := program.Functions
		require.Len(t, functions, 2)

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 1},
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionGreater{},
			opcode.InstructionJumpIfFalse{Target: 20},
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 1},
			opcode.InstructionJump{Target: 24},
			// 20
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 1},
			// 24
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionReturnValue{},
		},
			functions[1].Code,
		)

		comp2 := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp2.Config.PeepholeOptimizationsEnabled = true
		program2 := comp2.Compile()

		functions2 := program2.Functions
		require.Len(t, functions2, 2)

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 1},
			// transfers removed
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 0},
			// combined instrs
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 1},
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetConstant{Constant: 1},
			opcode.InstructionGreater{},
			opcode.InstructionJumpIfFalse{Target: 19},
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 0},
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 1},
			opcode.InstructionJump{Target: 22},
			// 17, jumps to correct statement after patching
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 2},
			opcode.InstructionSetLocal{Local: 1},
			// 20
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionReturnValue{},
		}, functions2[1].Code)
	})

	t.Run("getFieldLocal", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			struct Test {
				var x: Int

				init() {
					self.x = 32
				}
			}

			fun test(): Int {
				let test = Test()
				return test.x
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

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 1},
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			// common pattern
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetField{FieldName: 0, AccessedType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionReturnValue{},
		},
			functions[0].Code,
		)

		comp2 := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp2.Config.PeepholeOptimizationsEnabled = true
		program2 := comp2.Compile()

		functions2 := program2.Functions
		require.Len(t, functions2, 5)

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetGlobal{Global: 1},
			opcode.InstructionInvoke{ReturnType: 1},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			// combined instr
			opcode.InstructionGetFieldLocal{FieldName: 0, AccessedType: 1, Local: 0},
			opcode.InstructionTransferAndConvert{ValueType: 2, TargetType: 2},
			opcode.InstructionReturnValue{},
		}, functions2[0].Code)
	})

	t.Run("peephole avoid jump targets", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			fun test(): Int? {
				var x: Int? = true ? 123 : nil
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

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 5},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionJump{Target: 6},
			opcode.InstructionNil{},
			// this transfer after nil cannot be optimized out because it is a jump target
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionReturnValue{},
		},
			functions[0].Code,
		)

		comp2 := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp2.Config.PeepholeOptimizationsEnabled = true
		program2 := comp2.Compile()

		functions2 := program2.Functions
		require.Len(t, functions2, 1)

		assert.Equal(t, []opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionTrue{},
			opcode.InstructionJumpIfFalse{Target: 5},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionJump{Target: 6},
			opcode.InstructionNil{},
			// expect this transfer to still be here
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionSetLocal{Local: 0},
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.InstructionReturnValue{},
		}, functions2[0].Code)
	})
}

func BenchmarkCompileTime(b *testing.B) {
	checker, err := ParseAndCheck(b, `
	struct Foo {
		var id : Int

		init(_ id: Int) {
			self.id = id
		}
	}

	fun test(count: Int) {
		var i = 0
		while i < count {
			Foo(i)
			i = i + 1
		}
	}
	`)
	require.NoError(b, err)

	for b.Loop() {
		b.StopTimer()
		comp := compiler.NewInstructionCompiler(
			interpreter.ProgramFromChecker(checker),
			checker.Location,
		)
		comp.Config.PeepholeOptimizationsEnabled = true
		b.ReportAllocs()
		b.StartTimer()
		comp.Compile()
	}
}

func TestCompileBoundFunctionClosure(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      contract Test {
          fun getAnswer(): Int {
              return 42
          }

          fun getAnswerClosure(): fun(): Int {
              return fun(): Int {
                  return self.getAnswer()
              }
          }

          fun getAnswerDirect(): Int {
              return self.getAnswer()
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
	require.Len(t, functions, 7)

	const (
		initFuncIndex = iota
		// Next three indexes are for builtin methods (i.e: getType, isInstance)
		_
		_
		getAnswerFuncIndex
		getAnswerClosureFuncIndex
		getAnswerClosureInnerFuncIndex
		getAnswerDirectFuncIndex
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionNewComposite{Kind: common.CompositeKindContract, Type: 1},
			opcode.InstructionDup{},
			opcode.InstructionSetGlobal{Global: 0},
			opcode.InstructionReturnValue{},
		},
		functions[initFuncIndex].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetConstant{Constant: 0},
			opcode.InstructionTransferAndConvert{ValueType: 5, TargetType: 5},
			opcode.InstructionReturnValue{},
		},
		functions[getAnswerFuncIndex].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionNewClosure{
				Function: 5,
				Upvalues: []opcode.Upvalue{
					{
						TargetIndex: 0,
						IsLocal:     true,
					},
				},
			},
			opcode.InstructionTransferAndConvert{ValueType: 4, TargetType: 4},
			opcode.InstructionReturnValue{},
		},
		functions[getAnswerClosureFuncIndex].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetUpvalue{Upvalue: 0},
			opcode.InstructionGetMethod{Method: 4, ReceiverType: 1},
			opcode.InstructionInvoke{ReturnType: 5},
			opcode.InstructionTransferAndConvert{ValueType: 5, TargetType: 5},
			opcode.InstructionReturnValue{},
		},
		functions[getAnswerClosureInnerFuncIndex].Code,
	)

	assert.Equal(t,
		[]opcode.Instruction{
			opcode.InstructionStatement{},
			opcode.InstructionGetLocal{Local: 0},
			opcode.InstructionGetMethod{Method: 4, ReceiverType: 1},
			opcode.InstructionInvoke{ReturnType: 5},
			opcode.InstructionTransferAndConvert{ValueType: 5, TargetType: 5},
			opcode.InstructionReturnValue{},
		},
		functions[getAnswerDirectFuncIndex].Code,
	)
}
