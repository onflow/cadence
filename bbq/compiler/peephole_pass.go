package compiler

import "github.com/onflow/cadence/bbq/opcode"

type PeepholeOptimizer[E, T any] struct {
	PatternsByOpcode map[opcode.Opcode][]PeepholePattern
	compiler         *Compiler[opcode.Instruction, any]
}

func NewPeepholeOptimizer[E, T any](compiler *Compiler[E, T]) *PeepholeOptimizer[E, T] {
	patternsByOpcode := make(map[opcode.Opcode][]PeepholePattern)
	for _, pattern := range AllPatterns {
		if patternsByOpcode[pattern.Opcodes[0]] == nil {
			patternsByOpcode[pattern.Opcodes[0]] = make([]PeepholePattern, 0)
		}
		patternsByOpcode[pattern.Opcodes[0]] = append(patternsByOpcode[pattern.Opcodes[0]], pattern)
	}
	// restrict the compiler to opcode.Instruction and any
	// so patterns can ignore generics
	if c, ok := any(compiler).(*Compiler[opcode.Instruction, any]); ok {
		return &PeepholeOptimizer[E, T]{
			PatternsByOpcode: patternsByOpcode,
			compiler:         c,
		}
	}
	return nil
}

func (o *PeepholeOptimizer[E, T]) OptimizeInstructions(instructions []opcode.Instruction) []opcode.Instruction {
	optimized := make([]opcode.Instruction, 0, len(instructions))
	for i := 0; i < len(instructions); i++ {
		candidates := o.PatternsByOpcode[instructions[i].Opcode()]
		var matched bool
		for _, candidate := range candidates {
			window := instructions[i : i+len(candidate.Opcodes)]
			if candidate.Match(window) {
				optimized = append(optimized, candidate.Replacement(window, o.compiler)...)
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

func (o *PeepholeOptimizer[E, T]) Optimize(code []E) []E {
	if instructions, ok := any(code).([]opcode.Instruction); ok {
		return any(o.OptimizeInstructions(instructions)).([]E)
	}
	return code
}
