package compiler

import "github.com/onflow/cadence/bbq/opcode"

type PeepholeOptimizer[E any] struct {
	PatternsByOpcode map[opcode.Opcode][]PeepholePattern
}

func NewPeepholeOptimizer[E any]() *PeepholeOptimizer[E] {
	patternsByOpcode := make(map[opcode.Opcode][]PeepholePattern)
	for _, pattern := range AllPatterns {
		if patternsByOpcode[pattern.Opcodes[0]] == nil {
			patternsByOpcode[pattern.Opcodes[0]] = make([]PeepholePattern, 0)
		}
		patternsByOpcode[pattern.Opcodes[0]] = append(patternsByOpcode[pattern.Opcodes[0]], pattern)
	}
	return &PeepholeOptimizer[E]{
		PatternsByOpcode: patternsByOpcode,
	}
}

func (o *PeepholeOptimizer[E]) OptimizeInstructions(instructions []opcode.Instruction) []opcode.Instruction {
	optimized := make([]opcode.Instruction, 0, len(instructions))
	for i := 0; i < len(instructions); i++ {
		candidates := o.PatternsByOpcode[instructions[i].Opcode()]
		var matched bool
		for _, candidate := range candidates {
			window := instructions[i : i+len(candidate.Opcodes)]
			if candidate.Match(window) {
				optimized = append(optimized, candidate.Replacement(window)...)
				i += len(candidate.Opcodes) - 1
				matched = true
				break
			}
		}
		if !matched {
			optimized = append(optimized, instructions[i])
		}
	}
	return optimized
}

func (o *PeepholeOptimizer[E]) Optimize(code []E) []E {
	if instructions, ok := any(code).([]opcode.Instruction); ok {
		return any(o.OptimizeInstructions(instructions)).([]E)
	}
	return code
}
