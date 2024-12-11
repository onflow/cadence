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
)

type CodeGen interface {
	Offset() int
	Code() interface{}
	Emit(instruction opcode.Instruction)
	PatchJump(offset int, newTarget uint16)
}

type BytecodeGen struct {
	code []byte
}

var _ CodeGen = &BytecodeGen{}

func (g *BytecodeGen) Offset() int {
	return len(g.code)
}

func (g *BytecodeGen) Code() interface{} {
	return g.code
}

func (g *BytecodeGen) Emit(instruction opcode.Instruction) {
	instruction.Encode(&g.code)
}

func (g *BytecodeGen) PatchJump(offset int, newTarget uint16) {
	opcode.PatchJump(&g.code, offset, newTarget)
}
