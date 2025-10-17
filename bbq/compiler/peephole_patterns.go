package compiler

import "github.com/onflow/cadence/bbq/opcode"

type PeepholePattern struct {
	Name        string
	Opcodes     []opcode.Opcode
	Replacement []opcode.Instruction
}

var VariableInitializationPattern = PeepholePattern{
	Name:        "VariableInitialization",
	Opcodes:     []opcode.Opcode{opcode.GetConstant, opcode.TransferAndConvert, opcode.SetLocal},
	Replacement: []opcode.Instruction{}, // TODO: implement this
}

func (p *PeepholePattern) Match(instructions []opcode.Instruction) bool {
	for i, opcode := range p.Opcodes {
		if instructions[i].Opcode() == opcode {
			return true
		}
	}
	return false
}

var AllPatterns = []PeepholePattern{
	VariableInitializationPattern,
}
