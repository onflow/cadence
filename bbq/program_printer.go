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

	"github.com/onflow/cadence/common"

	"github.com/onflow/cadence/bbq/constantkind"
	"github.com/onflow/cadence/bbq/leb128"
	"github.com/onflow/cadence/bbq/opcode"
	"github.com/onflow/cadence/errors"
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

		// TODO: duplicate of `VM.initializeConstant()`
		kind := constant.Kind
		data := constant.Data

		var (
			v   any
			err error
		)

		switch kind {
		case constantkind.String:
			v = string(data)

		case constantkind.Int:
			// TODO: support larger integers
			v, _, err = leb128.ReadInt64(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Int constant: %s", err))
			}

		case constantkind.Int8:
			v, _, err = leb128.ReadInt32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Int8 constant: %s", err))
			}

		case constantkind.Int16:
			v, _, err = leb128.ReadInt32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Int16 constant: %s", err))
			}

		case constantkind.Int32:
			v, _, err = leb128.ReadInt32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Int32 constant: %s", err))
			}

		case constantkind.Int64:
			v, _, err = leb128.ReadInt64(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Int64 constant: %s", err))
			}

		case constantkind.UInt:
			// TODO: support larger integers
			v, _, err = leb128.ReadUint64(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read UInt constant: %s", err))
			}

		case constantkind.UInt8:
			v, _, err = leb128.ReadUint32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read UInt8 constant: %s", err))
			}

		case constantkind.UInt16:
			v, _, err = leb128.ReadUint32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read UInt16 constant: %s", err))
			}

		case constantkind.UInt32:
			v, _, err = leb128.ReadUint32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read UInt32 constant: %s", err))
			}

		case constantkind.UInt64:
			v, _, err = leb128.ReadUint64(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read UInt64 constant: %s", err))
			}

		case constantkind.Word8:
			v, _, err = leb128.ReadUint32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Word8 constant: %s", err))
			}

		case constantkind.Word16:
			v, _, err = leb128.ReadUint32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Word16 constant: %s", err))
			}

		case constantkind.Word32:
			v, _, err = leb128.ReadUint32(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Word32 constant: %s", err))
			}

		case constantkind.Word64:
			v, _, err = leb128.ReadUint64(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Word64 constant: %s", err))
			}

		case constantkind.Fix64:
			v, _, err = leb128.ReadInt64(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read Fix64 constant: %s", err))
			}

		case constantkind.UFix64:
			v, _, err = leb128.ReadUint64(data)
			if err != nil {
				panic(errors.NewUnexpectedError("failed to read UFix64 constant: %s", err))
			}

		case constantkind.Address:
			v = common.MustBytesToAddress(data)

		// TODO:
		// case constantkind.Int128:
		// case constantkind.Int256:
		// case constantkind.UInt128:
		// case constantkind.UInt256:
		// case constantkind.Word128:
		// case constantkind.Word256:

		default:
			panic(errors.NewUnexpectedError("unsupported constant kind: %s", kind))
		}

		_, _ = fmt.Fprintf(tabWriter, "%d |\t%s |\t %v\n", index, constant.Kind, v)
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
