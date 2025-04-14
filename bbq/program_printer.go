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
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/onflow/cadence/bbq/opcode"
)

type ProgramPrinter[E, T any] struct {
	stringBuilder strings.Builder
	codePrinter   func(builder *strings.Builder, code []E) error
	typeDecoder   func(bytes T) (StaticType, error)
}

func NewBytecodeProgramPrinter() *ProgramPrinter[byte, []byte] {
	return &ProgramPrinter[byte, []byte]{
		codePrinter: opcode.PrintBytecode,
		typeDecoder: StaticTypeFromBytes,
	}
}

func NewInstructionsProgramPrinter() *ProgramPrinter[opcode.Instruction, StaticType] {
	return &ProgramPrinter[opcode.Instruction, StaticType]{
		codePrinter: opcode.PrintInstructions,
		typeDecoder: func(typ StaticType) (StaticType, error) {
			return typ, nil
		},
	}
}

func (p *ProgramPrinter[E, T]) PrintProgram(program *Program[E, T]) string {
	p.printImports(program.Imports)
	p.printConstantPool(program.Constants)
	p.printTypePool(program.Types)

	for _, function := range program.Functions {
		p.printFunction(function)
		p.stringBuilder.WriteRune('\n')
	}

	return p.stringBuilder.String()
}

func (p *ProgramPrinter[E, T]) printFunction(function Function[E]) {
	p.stringBuilder.WriteString("-- " + function.Name + " --\n")
	err := p.codePrinter(&p.stringBuilder, function.Code)
	if err != nil {
		// TODO: propagate error
		panic(err)
	}
}

func (p *ProgramPrinter[_, T]) printConstantPool(constants []Constant) {
	p.stringBuilder.WriteString("-- Constant Pool --\n")

	tabWriter := tabwriter.NewWriter(&p.stringBuilder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for index, constant := range constants {
		_, _ = fmt.Fprintf(
			tabWriter,
			"%d |\t%s |\t %s\n",
			index,
			constant.Kind,
			constant,
		)
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(&p.stringBuilder)
}

func (p *ProgramPrinter[_, T]) printTypePool(types []T) {
	p.stringBuilder.WriteString("-- Type Pool --\n")

	tabWriter := tabwriter.NewWriter(&p.stringBuilder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for index, typ := range types {
		staticType, err := p.typeDecoder(typ)
		if err != nil {
			panic(err)
		}

		_, _ = fmt.Fprintf(tabWriter, "%d |\t %s\n", index, staticType.ID())
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(&p.stringBuilder)
}

func (p *ProgramPrinter[_, _]) printImports(imports []Import) {
	p.stringBuilder.WriteString("-- Imports --\n")

	tabWriter := tabwriter.NewWriter(&p.stringBuilder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for index, impt := range imports {

		name := impt.Name
		if impt.Location != nil {
			name = string(impt.Location.TypeID(nil, impt.Name))
		}

		_, _ = fmt.Fprintf(tabWriter, "%d |\t %s\n", index, name)
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(&p.stringBuilder)
}
