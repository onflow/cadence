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

package opcode_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/interpreter"
)

func TestPrettyInstructionWithResolvableOperands(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{
		Constants: []constant.DecodedConstant{
			{Data: "myField", Kind: constant.String},
			{Data: "myOtherField", Kind: constant.String},
		},
		Types: []interpreter.StaticType{
			interpreter.PrimitiveStaticTypeInt,
			interpreter.PrimitiveStaticTypeString,
		},
		Functions: []bbq.Function[opcode.Instruction]{
			{
				Name:          "myFunction",
				QualifiedName: "MyContract.myFunction",
			},
			{
				Name: "anonFunc",
			},
		},
	}

	t.Run("constant and type", func(t *testing.T) {
		t.Parallel()

		instruction := opcode.InstructionGetField{
			FieldName:    1,
			AccessedType: 1,
		}

		assert.Equal(t,
			opcode.PrettyInstructionGetField{
				FieldName: constant.DecodedConstant{
					Data: "myOtherField",
					Kind: constant.String,
				},
				AccessedType: interpreter.PrimitiveStaticTypeString,
			},
			instruction.Pretty(program),
		)
	})

	t.Run("function", func(t *testing.T) {
		t.Parallel()

		instruction := opcode.InstructionNewClosure{
			Function: 0,
			Upvalues: []opcode.Upvalue{
				{TargetIndex: 1, IsLocal: true},
			},
		}

		assert.Equal(t,
			opcode.PrettyInstructionNewClosure{
				Function: "MyContract.myFunction",
				Upvalues: []opcode.Upvalue{
					{TargetIndex: 1, IsLocal: true},
				},
			},
			instruction.Pretty(program),
		)
	})

	t.Run("types", func(t *testing.T) {
		t.Parallel()

		instruction := opcode.InstructionInvoke{
			TypeArgs:   []uint16{0, 1},
			ArgCount:   2,
			ReturnType: 1,
		}

		assert.Equal(t,
			opcode.PrettyInstructionInvoke{
				TypeArgs: []interpreter.StaticType{
					interpreter.PrimitiveStaticTypeInt,
					interpreter.PrimitiveStaticTypeString,
				},
				ArgCount:   2,
				ReturnType: interpreter.PrimitiveStaticTypeString,
			},
			instruction.Pretty(program),
		)
	})
}

func TestPrettyInstructionWithoutResolvableOperands(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{}

	instruction := opcode.InstructionGetLocal{
		Local: 5,
	}

	prettyIns := instruction.Pretty(program).(opcode.PrettyInstructionGetLocal)

	require.Equal(t, instruction.Local, prettyIns.Local)
}

func TestPrettyInstructionOpcode(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{
		Constants: []constant.DecodedConstant{
			{Data: "test", Kind: constant.String},
		},
		Types: []interpreter.StaticType{
			interpreter.PrimitiveStaticTypeInt,
		},
	}

	instruction := opcode.InstructionGetField{
		FieldName:    0,
		AccessedType: 0,
	}

	prettyIns := instruction.Pretty(program)

	require.Equal(t, opcode.GetField, prettyIns.Opcode())
}

func TestPrettyInstructionString(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{
		Constants: []constant.DecodedConstant{
			{Data: "test", Kind: constant.String},
		},
		Types: []interpreter.StaticType{
			interpreter.PrimitiveStaticTypeInt,
		},
	}

	instruction := opcode.InstructionGetField{
		FieldName:    0,
		AccessedType: 0,
	}

	prettyIns := instruction.Pretty(program)

	require.Equal(t, `GetField fieldName:test accessedType:"Int"`, prettyIns.String())
}

func TestPrettyInstructionAnonymousFunction(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{
		Functions: []bbq.Function[opcode.Instruction]{
			{
				// NOTE: no Name or QualifiedName
			},
		},
	}

	instruction := opcode.InstructionNewClosure{
		Function: 0,
		Upvalues: nil,
	}

	prettyIns := instruction.Pretty(program).(opcode.PrettyInstructionNewClosure)

	require.Equal(t, "<anonymous>", prettyIns.Function)
}

func TestPrettyFunctionNameFallback(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{
		Functions: []bbq.Function[opcode.Instruction]{
			{
				Name: "simpleName",
				// NOTE: no QualifiedName
			},
		},
	}

	instruction := opcode.InstructionNewClosure{
		Function: 0,
		Upvalues: nil,
	}

	prettyIns := instruction.Pretty(program).(opcode.PrettyInstructionNewClosure)

	require.Equal(t, "simpleName", prettyIns.Function)
}

func TestPrettyInstructionMapping(t *testing.T) {
	t.Parallel()

	program := &bbq.InstructionProgram{
		Constants: []constant.DecodedConstant{
			{Data: "myField", Kind: constant.String},
			{Data: "myOtherField", Kind: constant.String},
		},
		Types: []interpreter.StaticType{
			interpreter.PrimitiveStaticTypeInt,
			interpreter.PrimitiveStaticTypeString,
		},
		Functions: []bbq.Function[opcode.Instruction]{
			{
				Name:          "myFunction",
				QualifiedName: "MyContract.myFunction",
			},
			{
				Name: "anonFunc",
			},
		},
	}

	type instructionMapping struct {
		Instruction       opcode.Instruction
		PrettyInstruction opcode.PrettyInstruction
	}

	instructions := []instructionMapping{
		// Instructions without operands
		{opcode.InstructionUnknown{}, opcode.PrettyInstructionUnknown{}},
		{opcode.InstructionVoid{}, opcode.PrettyInstructionVoid{}},
		{opcode.InstructionTrue{}, opcode.PrettyInstructionTrue{}},
		{opcode.InstructionFalse{}, opcode.PrettyInstructionFalse{}},
		{opcode.InstructionNil{}, opcode.PrettyInstructionNil{}},
		{opcode.InstructionDup{}, opcode.PrettyInstructionDup{}},
		{opcode.InstructionDrop{}, opcode.PrettyInstructionDrop{}},
		{opcode.InstructionDestroy{}, opcode.PrettyInstructionDestroy{}},
		{opcode.InstructionUnwrap{}, opcode.PrettyInstructionUnwrap{}},
		{opcode.InstructionWrap{}, opcode.PrettyInstructionWrap{}},
		{opcode.InstructionTransfer{}, opcode.PrettyInstructionTransfer{}},
		{opcode.InstructionDeref{}, opcode.PrettyInstructionDeref{}},
		{opcode.InstructionReturn{}, opcode.PrettyInstructionReturn{}},
		{opcode.InstructionReturnValue{}, opcode.PrettyInstructionReturnValue{}},
		{opcode.InstructionEqual{}, opcode.PrettyInstructionEqual{}},
		{opcode.InstructionNotEqual{}, opcode.PrettyInstructionNotEqual{}},
		{opcode.InstructionSame{}, opcode.PrettyInstructionSame{}},
		{opcode.InstructionNot{}, opcode.PrettyInstructionNot{}},
		{opcode.InstructionAdd{}, opcode.PrettyInstructionAdd{}},
		{opcode.InstructionSubtract{}, opcode.PrettyInstructionSubtract{}},
		{opcode.InstructionMultiply{}, opcode.PrettyInstructionMultiply{}},
		{opcode.InstructionDivide{}, opcode.PrettyInstructionDivide{}},
		{opcode.InstructionMod{}, opcode.PrettyInstructionMod{}},
		{opcode.InstructionNegate{}, opcode.PrettyInstructionNegate{}},
		{opcode.InstructionLess{}, opcode.PrettyInstructionLess{}},
		{opcode.InstructionLessOrEqual{}, opcode.PrettyInstructionLessOrEqual{}},
		{opcode.InstructionGreater{}, opcode.PrettyInstructionGreater{}},
		{opcode.InstructionGreaterOrEqual{}, opcode.PrettyInstructionGreaterOrEqual{}},
		{opcode.InstructionBitwiseOr{}, opcode.PrettyInstructionBitwiseOr{}},
		{opcode.InstructionBitwiseXor{}, opcode.PrettyInstructionBitwiseXor{}},
		{opcode.InstructionBitwiseAnd{}, opcode.PrettyInstructionBitwiseAnd{}},
		{opcode.InstructionBitwiseLeftShift{}, opcode.PrettyInstructionBitwiseLeftShift{}},
		{opcode.InstructionBitwiseRightShift{}, opcode.PrettyInstructionBitwiseRightShift{}},
		{opcode.InstructionIterator{}, opcode.PrettyInstructionIterator{}},
		{opcode.InstructionIteratorHasNext{}, opcode.PrettyInstructionIteratorHasNext{}},
		{opcode.InstructionIteratorNext{}, opcode.PrettyInstructionIteratorNext{}},
		{opcode.InstructionIteratorEnd{}, opcode.PrettyInstructionIteratorEnd{}},
		{opcode.InstructionLoop{}, opcode.PrettyInstructionLoop{}},
		{opcode.InstructionStatement{}, opcode.PrettyInstructionStatement{}},
		{opcode.InstructionGetIndex{}, opcode.PrettyInstructionGetIndex{}},
		{opcode.InstructionRemoveIndex{}, opcode.PrettyInstructionRemoveIndex{}},
		{opcode.InstructionSetIndex{}, opcode.PrettyInstructionSetIndex{}},
		{opcode.InstructionSetAttachmentBase{}, opcode.PrettyInstructionSetAttachmentBase{}},

		// Instructions with non-resolvable operands only
		{
			opcode.InstructionGetLocal{Local: 5},
			opcode.PrettyInstructionGetLocal{Local: 5},
		},
		{
			opcode.InstructionSetLocal{Local: 3, IsTempVar: true},
			opcode.PrettyInstructionSetLocal{Local: 3, IsTempVar: true},
		},
		{
			opcode.InstructionGetUpvalue{Upvalue: 2},
			opcode.PrettyInstructionGetUpvalue{Upvalue: 2},
		},
		{
			opcode.InstructionSetUpvalue{Upvalue: 1},
			opcode.PrettyInstructionSetUpvalue{Upvalue: 1},
		},
		{
			opcode.InstructionCloseUpvalue{Local: 4},
			opcode.PrettyInstructionCloseUpvalue{Local: 4},
		},
		{
			opcode.InstructionGetGlobal{Global: 10},
			opcode.PrettyInstructionGetGlobal{Global: 10},
		},
		{
			opcode.InstructionSetGlobal{Global: 11},
			opcode.PrettyInstructionSetGlobal{Global: 11},
		},
		{
			opcode.InstructionGetMethod{
				Method:       7,
				ReceiverType: 1,
			},
			opcode.PrettyInstructionGetMethod{
				Method:       7,
				ReceiverType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionJump{Target: 100},
			opcode.PrettyInstructionJump{Target: 100},
		},
		{
			opcode.InstructionJumpIfFalse{Target: 200},
			opcode.PrettyInstructionJumpIfFalse{Target: 200},
		},
		{
			opcode.InstructionJumpIfTrue{Target: 300},
			opcode.PrettyInstructionJumpIfTrue{Target: 300},
		},
		{
			opcode.InstructionJumpIfNil{Target: 400},
			opcode.PrettyInstructionJumpIfNil{Target: 400},
		},
		{
			opcode.InstructionTemplateString{ExprSize: 3},
			opcode.PrettyInstructionTemplateString{ExprSize: 3},
		},

		// Instructions with resolvable operands - constants
		{
			opcode.InstructionGetConstant{Constant: 1},
			opcode.PrettyInstructionGetConstant{
				Constant: constant.DecodedConstant{
					Data: "myOtherField",
					Kind: constant.String,
				},
			},
		},
		{
			opcode.InstructionRemoveField{FieldName: 1},
			opcode.PrettyInstructionRemoveField{
				FieldName: constant.DecodedConstant{
					Data: "myOtherField",
					Kind: constant.String,
				},
			},
		},

		// Instructions with resolvable operands - constants and types
		{
			opcode.InstructionGetField{
				FieldName:    1,
				AccessedType: 1,
			},
			opcode.PrettyInstructionGetField{
				FieldName: constant.DecodedConstant{
					Data: "myOtherField",
					Kind: constant.String,
				},
				AccessedType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionSetField{
				FieldName:    1,
				AccessedType: 1,
			},
			opcode.PrettyInstructionSetField{
				FieldName: constant.DecodedConstant{
					Data: "myOtherField",
					Kind: constant.String,
				},
				AccessedType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionGetFieldLocal{
				FieldName:    1,
				AccessedType: 1,
				Local:        2,
			},
			opcode.PrettyInstructionGetFieldLocal{
				FieldName: constant.DecodedConstant{
					Data: "myOtherField",
					Kind: constant.String,
				},
				AccessedType: interpreter.PrimitiveStaticTypeString,
				Local:        2,
			},
		},

		// Instructions with resolvable operands - types
		{
			opcode.InstructionTransferAndConvert{ValueType: 1, TargetType: 1},
			opcode.PrettyInstructionTransferAndConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionConvert{ValueType: 1, TargetType: 1},
			opcode.PrettyInstructionConvert{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionSimpleCast{ValueType: 1, TargetType: 1},
			opcode.PrettyInstructionSimpleCast{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionFailableCast{ValueType: 1, TargetType: 1},
			opcode.PrettyInstructionFailableCast{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionForceCast{ValueType: 1, TargetType: 1},
			opcode.PrettyInstructionForceCast{
				ValueType:  interpreter.PrimitiveStaticTypeString,
				TargetType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionGetTypeIndex{Type: 1},
			opcode.PrettyInstructionGetTypeIndex{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionRemoveTypeIndex{Type: 1},
			opcode.PrettyInstructionRemoveTypeIndex{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionSetTypeIndex{Type: 1},
			opcode.PrettyInstructionSetTypeIndex{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionNewRef{Type: 1, IsImplicit: true},
			opcode.PrettyInstructionNewRef{
				Type:       interpreter.PrimitiveStaticTypeString,
				IsImplicit: true,
			},
		},
		{
			opcode.InstructionNewArray{Type: 1, Size: 5, IsResource: false},
			opcode.PrettyInstructionNewArray{
				Type:       interpreter.PrimitiveStaticTypeString,
				Size:       5,
				IsResource: false,
			},
		},
		{
			opcode.InstructionNewDictionary{Type: 1, Size: 3, IsResource: true},
			opcode.PrettyInstructionNewDictionary{
				Type:       interpreter.PrimitiveStaticTypeString,
				Size:       3,
				IsResource: true,
			},
		},
		{
			opcode.InstructionEmitEvent{Type: 1, ArgCount: 2},
			opcode.PrettyInstructionEmitEvent{
				Type:     interpreter.PrimitiveStaticTypeString,
				ArgCount: 2,
			},
		},

		// Instructions with composite kind and type
		{
			opcode.InstructionNewSimpleComposite{
				Kind: 1,
				Type: 1,
			},
			opcode.PrettyInstructionNewSimpleComposite{
				Kind: 1,
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionNewComposite{
				Kind: 2,
				Type: 1,
			},
			opcode.PrettyInstructionNewComposite{
				Kind: 2,
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionNewCompositeAt{
				Kind:    1,
				Type:    1,
				Address: 1,
			},
			opcode.PrettyInstructionNewCompositeAt{
				Kind: 1,
				Type: interpreter.PrimitiveStaticTypeString,
				Address: constant.DecodedConstant{
					Data: "myOtherField",
					Kind: constant.String,
				},
			},
		},

		// Instructions with path domain and constant
		{
			opcode.InstructionNewPath{
				Domain:     1,
				Identifier: 0,
			},
			opcode.PrettyInstructionNewPath{
				Domain: 1,
				Identifier: constant.DecodedConstant{
					Data: "myField",
					Kind: constant.String,
				},
			},
		},

		// Instructions with function index
		{
			opcode.InstructionNewClosure{
				Function: 0,
				Upvalues: []opcode.Upvalue{
					{TargetIndex: 1, IsLocal: true},
				},
			},
			opcode.PrettyInstructionNewClosure{
				Function: "MyContract.myFunction",
				Upvalues: []opcode.Upvalue{
					{TargetIndex: 1, IsLocal: true},
				},
			},
		},

		// Instructions with type indices
		{
			opcode.InstructionInvoke{
				TypeArgs:   []uint16{0, 1},
				ArgCount:   2,
				ReturnType: 1,
			},
			opcode.PrettyInstructionInvoke{
				TypeArgs: []interpreter.StaticType{
					interpreter.PrimitiveStaticTypeInt,
					interpreter.PrimitiveStaticTypeString,
				},
				ArgCount:   2,
				ReturnType: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			opcode.InstructionInvokeTyped{
				TypeArgs:   []uint16{1},
				ArgTypes:   []uint16{0, 1},
				ReturnType: 1,
			},
			opcode.PrettyInstructionInvokeTyped{
				TypeArgs: []interpreter.StaticType{
					interpreter.PrimitiveStaticTypeString,
				},
				ArgTypes: []interpreter.StaticType{
					interpreter.PrimitiveStaticTypeInt,
					interpreter.PrimitiveStaticTypeString,
				},
				ReturnType: interpreter.PrimitiveStaticTypeString,
			},
		},
	}

	tested := map[opcode.Opcode]struct{}{}
	for _, ins := range instructions {
		tested[ins.Instruction.Opcode()] = struct{}{}
	}

	for op := range opcode.OpcodeMax {
		name := op.String()
		if !strings.HasPrefix(name, "Opcode(") {
			assert.Contains(t, tested, op, "missing test for instruction %s", name)
		}
	}

	for _, ins := range instructions {
		t.Run(ins.Instruction.Opcode().String(), func(t *testing.T) {
			assert.Equal(t,
				ins.PrettyInstruction,
				ins.Instruction.Pretty(program),
			)
		})
	}

}
