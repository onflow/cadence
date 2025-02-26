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
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/errors"
)

type CodeGen[E any] interface {
	Offset() int
	SetTarget(code *[]E)
	Emit(instruction opcode.Instruction)
	PatchJump(offset int, newTarget uint16)
}

// ByteCodeGen is a CodeGen implementation that emits bytecode
type ByteCodeGen struct {
	target *[]byte
}

var _ CodeGen[byte] = &ByteCodeGen{}

func (g *ByteCodeGen) Offset() int {
	return len(*g.target)
}

func (g *ByteCodeGen) SetTarget(target *[]byte) {
	g.target = target
}

func (g *ByteCodeGen) Emit(instruction opcode.Instruction) {
	instruction.Encode(g.target)
}

func (g *ByteCodeGen) PatchJump(offset int, newTarget uint16) {
	opcode.PatchJump(g.target, offset, newTarget)
}

// InstructionCodeGen is a CodeGen implementation that emits opcode.Instruction
type InstructionCodeGen struct {
	target *[]opcode.Instruction
}

var _ CodeGen[opcode.Instruction] = &InstructionCodeGen{}

func (g *InstructionCodeGen) Offset() int {
	return len(*g.target)
}

func (g *InstructionCodeGen) SetTarget(target *[]opcode.Instruction) {
	g.target = target
}

func (g *InstructionCodeGen) Emit(instruction opcode.Instruction) {
	*g.target = append(*g.target, instruction)
}

func (g *InstructionCodeGen) PatchJump(offset int, newTarget uint16) {
	switch ins := (*g.target)[offset].(type) {
	case opcode.InstructionJump:
		ins.Target = newTarget
		(*g.target)[offset] = ins

	case opcode.InstructionJumpIfFalse:
		ins.Target = newTarget
		(*g.target)[offset] = ins

	case opcode.InstructionJumpIfTrue:
		ins.Target = newTarget
		(*g.target)[offset] = ins

	case opcode.InstructionJumpIfNil:
		ins.Target = newTarget
		(*g.target)[offset] = ins

	default:
		panic(errors.NewUnreachableError())
	}
}
