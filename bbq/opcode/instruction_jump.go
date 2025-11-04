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

package opcode

import "github.com/onflow/cadence/errors"

// JumpTarget returns the target of a jump instruction
func JumpTarget(instruction Instruction) uint16 {
	switch ins := instruction.(type) {
	case InstructionJump:
		return ins.Target
	case InstructionJumpIfFalse:
		return ins.Target
	case InstructionJumpIfTrue:
		return ins.Target
	case InstructionJumpIfNil:
		return ins.Target
	default:
		panic(errors.NewUnreachableError())
	}
}

func IsJump(instruction Instruction) bool {
	switch instruction.(type) {
	case InstructionJump:
		return true
	case InstructionJumpIfFalse:
		return true
	case InstructionJumpIfTrue:
		return true
	case InstructionJumpIfNil:
		return true
	default:
		return false
	}
}
