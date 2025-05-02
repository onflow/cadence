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

	"github.com/logrusorgru/aurora/v4"

	"github.com/onflow/cadence/bbq/constant"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/interpreter"
)

type CodePrinter[T, E any] func(
	builder *strings.Builder,
	code []E,
	resolve bool,
	constants []constant.Constant,
	types []T,
	functionNames []string,
	colorize bool,
) error

type TypeDecoder[T any] func(bytes T) (StaticType, error)

type ProgramPrinter[E, T any] struct {
	stringBuilder strings.Builder
	codePrinter   CodePrinter[T, E]
	typeDecoder   TypeDecoder[T]
	resolve       bool
	colorize      bool
}

func NewBytecodeProgramPrinter(resolve bool, colorize bool) *ProgramPrinter[byte, []byte] {
	return &ProgramPrinter[byte, []byte]{
		codePrinter: opcode.PrintBytecode,
		typeDecoder: interpreter.StaticTypeFromBytes,
		resolve:     resolve,
		colorize:    colorize,
	}
}

func NewInstructionsProgramPrinter(resolve bool, colorize bool) *ProgramPrinter[opcode.Instruction, StaticType] {
	return &ProgramPrinter[opcode.Instruction, StaticType]{
		codePrinter: opcode.PrintInstructions,
		typeDecoder: func(typ StaticType) (StaticType, error) {
			return typ, nil
		},
		resolve:  resolve,
		colorize: colorize,
	}
}

func (p *ProgramPrinter[E, T]) PrintProgram(program *Program[E, T]) string {
	p.printImports(program.Imports)
	p.printConstantPool(program.Constants)
	p.printTypePool(program.Types)

	for _, variable := range program.Variables {
		p.printFunction(
			variable.Getter,
			program.Constants,
			program.Types,
			nil,
		)
		p.stringBuilder.WriteRune('\n')
	}

	var functionNames []string
	if len(program.Functions) > 0 {
		functionNames = make([]string, 0, len(program.Functions))
		for _, function := range program.Functions {
			functionNames = append(functionNames, function.Name)
		}
	}

	for _, function := range program.Functions {
		p.printFunction(
			function,
			program.Constants,
			program.Types,
			functionNames,
		)
		p.stringBuilder.WriteRune('\n')
	}

	return p.stringBuilder.String()
}

func (p *ProgramPrinter[E, T]) printFunction(
	function Function[E],
	constants []constant.Constant,
	types []T,
	functionNames []string,
) {
	p.printHeader(function.QualifiedName)

	err := p.codePrinter(
		&p.stringBuilder,
		function.Code,
		p.resolve,
		constants,
		types,
		functionNames,
		p.colorize,
	)
	if err != nil {
		// TODO: propagate error
		panic(err)
	}
}

func (p *ProgramPrinter[_, T]) printConstantPool(constants []constant.Constant) {
	p.printHeader("Constant Pool")

	tabWriter := tabwriter.NewWriter(&p.stringBuilder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for index, constant := range constants {
		_, _ = fmt.Fprintf(
			tabWriter,
			"%s |\t%s |\t %s\n",
			p.colorizeIndex(index),
			p.colorizeConstantKind(constant.Kind),
			constant,
		)
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(&p.stringBuilder)
}

func (p *ProgramPrinter[_, T]) printTypePool(types []T) {
	p.printHeader("Type Pool")

	tabWriter := tabwriter.NewWriter(&p.stringBuilder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for index, typ := range types {
		staticType, err := p.typeDecoder(typ)
		if err != nil {
			panic(err)
		}

		_, _ = fmt.Fprintf(
			tabWriter,
			"%s |\t %s\n",
			p.colorizeIndex(index),
			staticType.ID(),
		)
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(&p.stringBuilder)
}

func (p *ProgramPrinter[_, _]) printImports(imports []Import) {
	p.printHeader("Imports")

	tabWriter := tabwriter.NewWriter(&p.stringBuilder, 0, 0, 1, ' ', tabwriter.AlignRight)

	for index, impt := range imports {

		name := impt.Name
		if impt.Location != nil {
			name = string(impt.Location.TypeID(nil, impt.Name))
		}

		_, _ = fmt.Fprintf(
			tabWriter,
			"%s |\t %s\n",
			p.colorizeIndex(index),
			name,
		)
	}

	_ = tabWriter.Flush()
	_, _ = fmt.Fprintln(&p.stringBuilder)
}

func (p *ProgramPrinter[_, _]) colorizeIndex(index int) string {
	if p.colorize {
		return opcode.ColorizeOffset(index)
	} else {
		return fmt.Sprint(index)
	}
}

func (p *ProgramPrinter[_, _]) colorizeConstantKind(kind constant.Kind) string {
	if p.colorize {
		return aurora.Blue(kind).String()
	} else {
		return kind.String()
	}
}

func (p *ProgramPrinter[_, _]) printHeader(title string) {
	title = fmt.Sprintf("-- %s --\n", title)
	if p.colorize {
		title = aurora.Bold(title).String()
	}
	p.stringBuilder.WriteString(title)
}
