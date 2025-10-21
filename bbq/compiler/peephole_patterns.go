package compiler

import "github.com/onflow/cadence/bbq/opcode"

type PeepholePattern struct {
	Name        string
	Opcodes     []opcode.Opcode
	Replacement func(instructions []opcode.Instruction) []opcode.Instruction
}

var VariableInitializationPattern = PeepholePattern{
	Name:    "VariableInitialization",
	Opcodes: []opcode.Opcode{opcode.GetConstant, opcode.TransferAndConvert, opcode.SetLocal},
	Replacement: func(instructions []opcode.Instruction) []opcode.Instruction {
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
