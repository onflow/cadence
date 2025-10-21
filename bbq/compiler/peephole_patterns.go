package compiler

import (
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/interpreter"
)

type PeepholePattern struct {
	Name        string
	Opcodes     []opcode.Opcode
	Replacement func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, any]) []opcode.Instruction
}

var VariableInitializationPattern = PeepholePattern{
	Name:    "VariableInitialization",
	Opcodes: []opcode.Opcode{opcode.GetConstant, opcode.TransferAndConvert, opcode.SetLocal},
	Replacement: func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, any]) []opcode.Instruction {
		getConstant := instructions[0].(opcode.InstructionGetConstant)
		transferAndConvert := instructions[1].(opcode.InstructionTransferAndConvert)
		setLocal := instructions[2].(opcode.InstructionSetLocal)

		return []opcode.Instruction{opcode.InstructionInitLocalFromConst{
			Constant: getConstant.Constant,
			Type:     transferAndConvert.Type,
			Local:    setLocal.Local,
		}}
	},
}

var AddPattern = PeepholePattern{
	Name:    "ConstantFoldingAdd",
	Opcodes: []opcode.Opcode{opcode.GetConstant, opcode.GetConstant, opcode.Add},
	Replacement: func(instructions []opcode.Instruction, compiler *Compiler[opcode.Instruction, any]) []opcode.Instruction {
		getConstant1 := instructions[0].(opcode.InstructionGetConstant)
		getConstant2 := instructions[1].(opcode.InstructionGetConstant)

		c1 := compiler.constants[getConstant1.Constant].data.(interpreter.NumberValue)
		c2 := compiler.constants[getConstant2.Constant].data.(interpreter.NumberValue)

		// TODO: how can we run this arithmetic operation
		// code below is just a placeholder
		compiler.emitIntConst(int64(c1.Plus(nil, c2).ToInt()))

		return []opcode.Instruction{opcode.InstructionGetConstant{
			Constant: getConstant1.Constant + getConstant2.Constant,
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
	VariableInitializationPattern,
}
