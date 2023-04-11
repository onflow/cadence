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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type BytecodePrinter struct {
	stringBuilder strings.Builder
}

func (p *BytecodePrinter) PrintProgram(program *Program) string {
	p.printImports(program.Imports)
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
			opcode.SetGlobal,
			opcode.Jump,
			opcode.JumpIfFalse,
			opcode.Transfer:
			var operand int
			operand, i = p.getIntOperand(codes, i)
			p.stringBuilder.WriteString(" " + fmt.Sprint(operand))

		case opcode.New:
			var kind int
			kind, i = p.getIntOperand(codes, i)

			var location common.Location
			location, i = p.getLocation(codes, i)

			var typeName string
			typeName, i = p.getStringOperand(codes, i)

			if location != nil {
				typeName = string(location.TypeID(nil, typeName))
			}

			p.stringBuilder.WriteString(" " + fmt.Sprint(kind) + " " + typeName)

		case opcode.Cast:
			var typeIndex int
			var castType byte
			typeIndex, i = p.getIntOperand(codes, i)
			castType, i = p.getByteOperand(codes, i)
			p.stringBuilder.WriteString(" " + fmt.Sprint(typeIndex) + " " + fmt.Sprint(int8(castType)))

		case opcode.Path:
			var identifier string
			var domain byte
			domain, i = p.getByteOperand(codes, i)
			identifier, i = p.getStringOperand(codes, i)
			p.stringBuilder.WriteString(" " + fmt.Sprint(int8(domain)) + " " + identifier)

		case opcode.InvokeDynamic:
			var funcName string
			funcName, i = p.getStringOperand(codes, i)
			p.stringBuilder.WriteString(" " + " " + funcName)

		// opcodes with no operands
		default:
			// do nothing
		}

		p.stringBuilder.WriteRune('\n')
	}
}

func (*BytecodePrinter) getIntOperand(codes []byte, i int) (operand int, endIndex int) {
	first := codes[i+1]
	last := codes[i+2]
	operand = int(uint16(first)<<8 | uint16(last))
	return operand, i + 2
}

func (p *BytecodePrinter) getStringOperand(codes []byte, i int) (operand string, endIndex int) {
	strLen, i := p.getIntOperand(codes, i)
	operand = string(codes[i+1 : i+1+strLen])
	return operand, i + strLen
}

func (*BytecodePrinter) getByteOperand(codes []byte, i int) (operand byte, endIndex int) {
	byt := codes[i+1]
	return byt, i + 1
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

func (p *BytecodePrinter) getLocation(codes []byte, i int) (location common.Location, endIndex int) {
	locationLen, i := p.getIntOperand(codes, i)
	locationBytes := codes[i+1 : i+1+locationLen]

	dec := interpreter.CBORDecMode.NewByteStreamDecoder(locationBytes)
	locationDecoder := interpreter.NewLocationDecoder(dec, nil)
	location, err := locationDecoder.DecodeLocation()
	if err != nil {
		panic(err)
	}

	return location, i + locationLen
}

func (p *BytecodePrinter) printImports(imports []*Import) {
	p.stringBuilder.WriteString("-- Imports --\n")
	for _, impt := range imports {
		location := impt.Location
		if location != nil {
			p.stringBuilder.WriteString(location.String())
			p.stringBuilder.WriteRune('.')
		}
		p.stringBuilder.WriteString(impt.Name)
		p.stringBuilder.WriteRune('\n')
	}
	p.stringBuilder.WriteRune('\n')
}
