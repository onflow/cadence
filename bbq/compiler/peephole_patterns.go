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
	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type PeepholePattern struct {
	Name string
	// never match a jump instruction
	Opcodes     []opcode.Opcode
	Replacement func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, interpreter.StaticType]) []opcode.Instruction
}

var InvokeTransferAndConvertPattern = PeepholePattern{
	Name:    "InvokeTransferAndConvert",
	Opcodes: []opcode.Opcode{opcode.Invoke, opcode.TransferAndConvert},
	Replacement: func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, interpreter.StaticType]) []opcode.Instruction {
		invoke := instructions[0].(opcode.InstructionInvoke)
		transferAndConvert := instructions[1].(opcode.InstructionTransferAndConvert)

		return []opcode.Instruction{opcode.InstructionInvokeTransferAndConvert{
			TypeArgs: invoke.TypeArgs,
			ArgCount: invoke.ArgCount,
			Type:     transferAndConvert.Type,
		}}
	},
}

var GetFieldLocalPattern = PeepholePattern{
	Name:    "GetFieldLocal",
	Opcodes: []opcode.Opcode{opcode.GetLocal, opcode.GetField},
	Replacement: func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, interpreter.StaticType]) []opcode.Instruction {
		getLocal := instructions[0].(opcode.InstructionGetLocal)
		getField := instructions[1].(opcode.InstructionGetField)

		return []opcode.Instruction{opcode.InstructionGetFieldLocal{
			FieldName:    getField.FieldName,
			AccessedType: getField.AccessedType,
			Local:        getLocal.Local,
		}}
	},
}

// Optimizations ported from compiler `mustEmitTransferAndConvert`
// TODO: check correctness in case of branching
var ConstantTransferAndConvertPattern = PeepholePattern{
	Name:    "ConstantTransferAndConvert",
	Opcodes: []opcode.Opcode{opcode.GetConstant, opcode.TransferAndConvert},
	Replacement: func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, interpreter.StaticType]) []opcode.Instruction {
		getConstant := instructions[0].(opcode.InstructionGetConstant)
		transferAndConvert := instructions[1].(opcode.InstructionTransferAndConvert)

		// safety check
		constantKind := compiler.constants[getConstant.Constant].kind
		targetType := compiler.types[transferAndConvert.Type]
		if constantKind == constant.FromSemaType(targetType) {
			return []opcode.Instruction{getConstant}
		}

		return instructions
	},
}

var PathTransferAndConvertPattern = PeepholePattern{
	Name:    "PathTransferAndConvert",
	Opcodes: []opcode.Opcode{opcode.NewPath, opcode.TransferAndConvert},
	Replacement: func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, interpreter.StaticType]) []opcode.Instruction {
		getPath := instructions[0].(opcode.InstructionNewPath)
		transferAndConvert := instructions[1].(opcode.InstructionTransferAndConvert)

		semaType := compiler.types[transferAndConvert.Type]

		// check if optimization is applicable
		if getPath.Domain != common.PathDomainStorage {
			switch getPath.Domain {
			case common.PathDomainPublic:
				if semaType == sema.PublicPathType {
					return []opcode.Instruction{getPath}
				}
			case common.PathDomainPrivate:
				if semaType == sema.PrivatePathType {
					return []opcode.Instruction{getPath}
				}
			}
		}

		return instructions
	},
}

var NilTransferAndConvertPattern = PeepholePattern{
	Name:    "NilTransferAndConvert",
	Opcodes: []opcode.Opcode{opcode.Nil, opcode.TransferAndConvert},
	Replacement: func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, interpreter.StaticType]) []opcode.Instruction {
		nil := instructions[0].(opcode.InstructionNil)
		return []opcode.Instruction{nil}
	},
}

func (p *PeepholePattern) Match(instructions []opcode.Instruction) bool {
	for i, opcode := range p.Opcodes {
		if instructions[i].Opcode() != opcode {
			return false
		}
	}
	return true
}

var AllPatterns = []PeepholePattern{
	InvokeTransferAndConvertPattern,
	GetFieldLocalPattern,
	ConstantTransferAndConvertPattern,
	PathTransferAndConvertPattern,
	NilTransferAndConvertPattern,
}
