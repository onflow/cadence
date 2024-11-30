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

package bbq

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/onflow/cadence/bbq/constantkind"
	"github.com/onflow/cadence/bbq/leb128"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type BytecodePrinter struct {
	stringBuilder strings.Builder
}

func (p *BytecodePrinter) PrintProgram(program *Program) string {
	p.printImports(program.Imports)
	p.printConstantPool(program.Constants)
	p.printTypePool(program.Types)

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

func (p *BytecodePrinter) printCode(code []byte) {
	reader := bytes.NewReader(code)
	err := opcode.PrintInstructions(&p.stringBuilder, reader)
	if err != nil {
		// TODO: propagate error
		panic(err)
	}
}

func (*BytecodePrinter) getIntOperand(codes []byte, i int) (operand int, endIndex int) {
	first := codes[i+1]
	last := codes[i+2]
	operand = int(uint16(first)<<8 | uint16(last))
	return operand, i + 2
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

func (p *BytecodePrinter) printTypePool(types [][]byte) {
	p.stringBuilder.WriteString("-- Type Pool --\n")

	for index, typeBytes := range types {
		dec := interpreter.CBORDecMode.NewByteStreamDecoder(typeBytes)
		typeDecoder := interpreter.NewTypeDecoder(dec, nil)
		staticType, err := typeDecoder.DecodeStaticType()
		if err != nil {
			panic(err)
		}

		p.stringBuilder.WriteString(fmt.Sprint(index))
		p.stringBuilder.WriteString(" | ")
		p.stringBuilder.WriteString(string(staticType.ID()))
		p.stringBuilder.WriteRune('\n')
	}

	p.stringBuilder.WriteRune('\n')
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
