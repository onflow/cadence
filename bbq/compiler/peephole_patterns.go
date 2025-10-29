package compiler

import (
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/interpreter"
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
}
