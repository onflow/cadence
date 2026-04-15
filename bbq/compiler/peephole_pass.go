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
	"math"
	"sort"

	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type BytecodeShiftPair struct {
	Offset int
	Shift  int
}

type PeepholeOptimizer[E any] interface {
	Optimize(code []E) []E
}

type PeepholeInstructionStaticTypeOptimizer struct {
	compiler *Compiler[opcode.Instruction, interpreter.StaticType]
	// bytecodeShifts is a list of where bytecode has been changed
	// and how much the offset has changed. this is used to patch jumps
	// after the peephole optimizations have been applied
	bytecodeShifts []BytecodeShiftPair

	// jumps is a list of offsets in the optimized bytecode that need to be patched
	jumps []int

	// jumpTargets is a set of offsets in the original bytecode that are jump targets
	jumpTargets map[int]struct{}
}

var _ PeepholeOptimizer[opcode.Instruction] = &PeepholeInstructionStaticTypeOptimizer{}

// look up table to improve pattern matching performance
var PatternsByOpcode map[opcode.Opcode][]PeepholePattern

func init() {
	PatternsByOpcode = make(map[opcode.Opcode][]PeepholePattern)
	for _, pattern := range AllPatterns {
		firstOpcode := pattern.Opcodes[0]
		PatternsByOpcode[firstOpcode] = append(PatternsByOpcode[firstOpcode], pattern)
	}
}

func NewPeepholeOptimizer[E, T any](compiler *Compiler[E, T]) PeepholeOptimizer[E] {
	// restrict the compiler to opcode.Instruction and static type
	// so patterns can ignore generics
	if c, ok := any(compiler).(*Compiler[opcode.Instruction, interpreter.StaticType]); ok {
		optimizer := &PeepholeInstructionStaticTypeOptimizer{
			compiler: c,
			// TODO: allocate these lazily
			jumpTargets:    make(map[int]struct{}),
			bytecodeShifts: make([]BytecodeShiftPair, 0),
			jumps:          make([]int, 0),
		}
		return any(optimizer).(PeepholeOptimizer[E])
	}
	return PeepholeNoopOptimizer[E]{}
}

type PeepholeNoopOptimizer[E any] struct {
}

var _ PeepholeOptimizer[struct{}] = PeepholeNoopOptimizer[struct{}]{}

func (PeepholeNoopOptimizer[E]) Optimize(code []E) []E {
	return code
}

// logic for patching jumps
// jumps are in order of their target offset (NOT their position in the optimized bytecode),
// bytecode shifts are also in order of their offset,
// so we can keep a counter for cumulative shift and patch jumps in the target order
// ASSUMPTION: peephole patterns should never match a jump instruction
func (o *PeepholeInstructionStaticTypeOptimizer) patchJumps(optimized []opcode.Instruction) {
	cumShift := 0
	currentShiftIndex := 0
	for _, jump := range o.jumps {
		jumpTarget := int(opcode.JumpTarget(optimized[jump]))

		// apply all shifts up to the current jump target
		for currentShiftIndex < len(o.bytecodeShifts) && o.bytecodeShifts[currentShiftIndex].Offset < jumpTarget {
			cumShift += o.bytecodeShifts[currentShiftIndex].Shift
			currentShiftIndex++
		}

		// patch jump
		newJumpTarget := jumpTarget + cumShift
		if newJumpTarget > math.MaxUint16 {
			//TODO: abort optimization pass instead of panicking
			panic(errors.NewUnexpectedError("peephole shifted jump target past max uint16"))
		}
		opcode.PatchJumpInstruction(optimized, jump, uint16(newJumpTarget))
	}
}

func (o *PeepholeInstructionStaticTypeOptimizer) OptimizeInstructions(instructions []opcode.Instruction) []opcode.Instruction {
	// collect all jump targets
	for _, instruction := range instructions {
		if opcode.IsJump(instruction) {
			o.jumpTargets[int(opcode.JumpTarget(instruction))] = struct{}{}
		}
	}

	optimized := make([]opcode.Instruction, 0, len(instructions))

	for i := 0; i < len(instructions); i++ {
		currentInstruction := instructions[i]
		candidates := PatternsByOpcode[currentInstruction.Opcode()]

		var matched bool
		// check candidates for pattern match for each instruction
		for _, candidate := range candidates {
			candidateOpcodes := candidate.Opcodes
			window := instructions[i : i+len(candidateOpcodes)]

			if candidate.Match(window, i, o.jumpTargets) {
				replacement := candidate.Replacement(window, o.compiler)
				optimized = append(optimized, replacement...)

				o.bytecodeShifts = append(o.bytecodeShifts, BytecodeShiftPair{
					Offset: i,
					// e.g. replace 2 with 1 -> shift is -1
					Shift: len(replacement) - len(candidateOpcodes),
				})

				i += len(candidateOpcodes) - 1
				matched = true
				break
			}
		}
		if !matched {
			optimized = append(optimized, currentInstruction)

			// if instruction is a jump, add it to the jumps list
			// its important we do it in the optimized index, not the original index
			// because its hard to calculate the optimized index from the original index later
			if opcode.IsJump(currentInstruction) {
				o.jumps = append(o.jumps, len(optimized)-1)
			}
		}
	}

	// Sort jumps by their target offsets
	sort.Slice(o.jumps, func(a, b int) bool {
		jumpATarget := opcode.JumpTarget(optimized[o.jumps[a]])
		jumpBTarget := opcode.JumpTarget(optimized[o.jumps[b]])
		// ascending order
		return jumpATarget < jumpBTarget
	})

	o.patchJumps(optimized)

	return optimized
}
func (o *PeepholeInstructionStaticTypeOptimizer) Optimize(code []opcode.Instruction) []opcode.Instruction {
	o.bytecodeShifts = o.bytecodeShifts[:0]
	o.jumps = o.jumps[:0]
	clear(o.jumpTargets)
	return o.OptimizeInstructions(code)
}
