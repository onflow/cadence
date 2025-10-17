package compiler

import "github.com/onflow/cadence/bbq/opcode"

type PeepholeOptimizer struct {
	PatternsByOpcode map[opcode.Opcode][]PeepholePattern
}

func NewPeepholeOptimizer() *PeepholeOptimizer {
	patternsByOpcode := make(map[opcode.Opcode][]PeepholePattern)
	for _, pattern := range AllPatterns {
		if patternsByOpcode[pattern.Opcodes[0]] == nil {
			patternsByOpcode[pattern.Opcodes[0]] = make([]PeepholePattern, 0)
		}
		patternsByOpcode[pattern.Opcodes[0]] = append(patternsByOpcode[pattern.Opcodes[0]], pattern)
	}
	return &PeepholeOptimizer{
		PatternsByOpcode: patternsByOpcode,
	}
}

func (o *PeepholeOptimizer) Optimize(instructions []opcode.Instruction) []opcode.Instruction {
	optimized := make([]opcode.Instruction, 0, len(instructions))
	for i := 0; i < len(instructions); i++ {
		candidates := o.PatternsByOpcode[instructions[i].Opcode()]
		for _, candidate := range candidates {
			if candidate.Match(instructions[i : i+len(candidate.Opcodes)-1]) {
				optimized = append(optimized, candidate.Replacement...)
				i += len(candidate.Opcodes)
				break
			}
		}
		optimized = append(optimized, instructions[i])
	}
	return optimized
}
