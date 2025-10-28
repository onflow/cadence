package compiler

import (
	"sort"

	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/interpreter"
)

type BytecodeShiftPair struct {
	Offset int
	Shift  int
}
type PeepholeOptimizer[E, T any] struct {
	PatternsByOpcode map[opcode.Opcode][]PeepholePattern
	compiler         *Compiler[opcode.Instruction, interpreter.StaticType]
	// bytecodeShifts is a list of where bytecode has been changed
	// and how much the offset has changed. this is used to patch jumps
	// after the peephole optimizations have been applied
	bytecodeShifts []BytecodeShiftPair

	jumps []int
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
	if c, ok := any(compiler).(*Compiler[opcode.Instruction, interpreter.StaticType]); ok {
		return &PeepholeOptimizer[E, T]{
			PatternsByOpcode: patternsByOpcode,
			compiler:         c,
			bytecodeShifts:   make([]BytecodeShiftPair, 0),
			jumps:            make([]int, 0),
		}
	}
	return nil
}

// logic for patching jumps
// jumps are in order of their target offset (NOT the optimized index),
// bytecode shifts are also in order of their offset,
// so we can keep a counter for cumulative shift and patch jumps as we go
func (o *PeepholeOptimizer[E, T]) patchJumps(optimized []opcode.Instruction) {
	cumShift := 0
	currentShiftIndex := 0
	for _, jump := range o.jumps {
		var jumpTarget int
		switch ins := optimized[jump].(type) {
		case opcode.InstructionJump:
			jumpTarget = int(ins.Target)
		case opcode.InstructionJumpIfFalse:
			jumpTarget = int(ins.Target)
		case opcode.InstructionJumpIfTrue:
			jumpTarget = int(ins.Target)
		case opcode.InstructionJumpIfNil:
			jumpTarget = int(ins.Target)
		}

		// apply all shifts up to the current jump target
		for currentShiftIndex < len(o.bytecodeShifts) && o.bytecodeShifts[currentShiftIndex].Offset < jumpTarget {
			cumShift += o.bytecodeShifts[currentShiftIndex].Shift
			currentShiftIndex++
		}

		newJumpTarget := jumpTarget + cumShift
		switch optimized[jump].(type) {
		case opcode.InstructionJump:
			optimized[jump] = opcode.InstructionJump{Target: uint16(newJumpTarget)}
		case opcode.InstructionJumpIfFalse:
			optimized[jump] = opcode.InstructionJumpIfFalse{Target: uint16(newJumpTarget)}
		case opcode.InstructionJumpIfTrue:
			optimized[jump] = opcode.InstructionJumpIfTrue{Target: uint16(newJumpTarget)}
		case opcode.InstructionJumpIfNil:
			optimized[jump] = opcode.InstructionJumpIfNil{Target: uint16(newJumpTarget)}
		}
	}
}

func (o *PeepholeOptimizer[E, T]) OptimizeInstructions(instructions []opcode.Instruction) []opcode.Instruction {
	optimized := make([]opcode.Instruction, 0, len(instructions))

	for i := 0; i < len(instructions); i++ {
		candidates := o.PatternsByOpcode[instructions[i].Opcode()]
		var matched bool
		for _, candidate := range candidates {
			window := instructions[i : i+len(candidate.Opcodes)]
			if candidate.Match(window) {
				replacement := candidate.Replacement(window, o.compiler)
				optimized = append(optimized, replacement...)
				o.bytecodeShifts = append(o.bytecodeShifts, BytecodeShiftPair{
					Offset: i,
					// replace 2 with 1 -> shift is -1
					Shift: len(replacement) - len(candidate.Opcodes),
				})
				i += len(candidate.Opcodes) - 1
				matched = true
				break
			}
		}
		if !matched {
			optimized = append(optimized, instructions[i])
			if instructions[i].Opcode() == opcode.Jump ||
				instructions[i].Opcode() == opcode.JumpIfFalse ||
				instructions[i].Opcode() == opcode.JumpIfTrue ||
				instructions[i].Opcode() == opcode.JumpIfNil {
				o.jumps = append(o.jumps, len(optimized)-1)
			}
		}
	}

	// Sort jumps by their target addresses
	sort.Slice(o.jumps, func(a, b int) bool {
		var jumpATarget, jumpBTarget uint16
		switch ins := instructions[o.jumps[a]].(type) {
		case opcode.InstructionJump:
			jumpATarget = ins.Target
		case opcode.InstructionJumpIfFalse:
			jumpATarget = ins.Target
		case opcode.InstructionJumpIfTrue:
			jumpATarget = ins.Target
		case opcode.InstructionJumpIfNil:
			jumpATarget = ins.Target
		}
		// ascending order
		return jumpATarget < jumpBTarget
	})

	o.patchJumps(optimized)

	return optimized
}

func (o *PeepholeOptimizer[E, T]) Optimize(code []E) []E {
	if instructions, ok := any(code).([]opcode.Instruction); ok {
		return any(o.OptimizeInstructions(instructions)).([]E)
	}
	return code
}
