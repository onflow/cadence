/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package bbq

import (
	"fmt"
	"strings"

	"github.com/onflow/cadence/runtime/bbq/constantkind"
	"github.com/onflow/cadence/runtime/bbq/leb128"
	"github.com/onflow/cadence/runtime/bbq/opcode"
	"github.com/onflow/cadence/runtime/errors"
)

type BytecodePrinter struct {
	stringBuilder strings.Builder
}

func (p *BytecodePrinter) PrintProgram(program *Program) string {
	p.printConstantPool(program.Constants)
	for _, function := range program.Functions {
		p.printFunction(function)
		p.stringBuilder.WriteRune('\n')
	}

	return p.stringBuilder.String()
}

func (p *BytecodePrinter) printFunction(function *Function) {
	p.stringBuilder.WriteString("-- " + function.Name + " --\n")
	p.printCode(function.Code)
}

func (p *BytecodePrinter) printCode(codes []byte) {
	for i := 0; i < len(codes); i++ {
		code := codes[i]
		opcodeString := opcode.Opcode(code).String()

		p.stringBuilder.WriteString(opcodeString)

		switch opcode.Opcode(code) {

		// opcodes with one operand
		case opcode.GetConstant,
			opcode.GetLocal,
			opcode.SetLocal,
			opcode.GetGlobal,
			opcode.Jump,
			opcode.JumpIfFalse,
			opcode.CheckType:

			first := codes[i+1]
			last := codes[i+2]
			i += 2

			operand := int(uint16(first)<<8 | uint16(last))
			p.stringBuilder.WriteString(" " + fmt.Sprint(operand))

		// opcodes with no operands
		default:
			// do nothing
		}

		p.stringBuilder.WriteRune('\n')
	}
}

func (p *BytecodePrinter) printConstantPool(constants []*Constant) {
	p.stringBuilder.WriteString("-- Constant Pool --\n")

	for index, constant := range constants {
		var constantStr string

		// TODO: duplicate of `VM.initializeConstant()`
		switch constant.Kind {
		case constantkind.Int:
			smallInt, _, _ := leb128.ReadInt64(constant.Data)
			constantStr = fmt.Sprint(smallInt)
		case constantkind.String:
			constantStr = string(constant.Data)
		default:
			panic(errors.NewUnreachableError())
		}

		p.stringBuilder.WriteString(fmt.Sprint(index))
		p.stringBuilder.WriteString(" | ")
		p.stringBuilder.WriteString(constant.Kind.String())
		p.stringBuilder.WriteString(" | ")
		p.stringBuilder.WriteString(constantStr)
		p.stringBuilder.WriteRune('\n')
	}

	p.stringBuilder.WriteRune('\n')
}
