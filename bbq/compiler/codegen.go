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

type CodeGen interface {
	Offset() int
	Code() interface{}
	Emit(instruction opcode.Instruction)
	PatchJump(offset int, newTarget uint16)
}

// ByteCodeGen is a CodeGen implementation that emits bytecode
type ByteCodeGen struct {
	code []byte
}

var _ CodeGen = &ByteCodeGen{}

func (g *ByteCodeGen) Offset() int {
	return len(g.code)
}

func (g *ByteCodeGen) Code() interface{} {
	return g.code
}

func (g *ByteCodeGen) Emit(instruction opcode.Instruction) {
	instruction.Encode(&g.code)
}

func (g *ByteCodeGen) PatchJump(offset int, newTarget uint16) {
	opcode.PatchJump(&g.code, offset, newTarget)
}

// InstructionCodeGen is a CodeGen implementation that emits opcode.Instruction
type InstructionCodeGen struct {
	code []opcode.Instruction
}

var _ CodeGen = &InstructionCodeGen{}

func (g *InstructionCodeGen) Offset() int {
	return len(g.code)
}

func (g *InstructionCodeGen) Code() interface{} {
	return g.code
}

func (g *InstructionCodeGen) Emit(instruction opcode.Instruction) {
	g.code = append(g.code, instruction)
}

func (g *InstructionCodeGen) PatchJump(offset int, newTarget uint16) {
	switch ins := g.code[offset].(type) {
	case opcode.InstructionJump:
		ins.Target = newTarget
		g.code[offset] = ins

	case opcode.InstructionJumpIfFalse:
		ins.Target = newTarget
		g.code[offset] = ins
	}

	panic(errors.NewUnreachableError())
}
