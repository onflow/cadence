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

import (
	"bytes"
	"fmt"
	"strings"
)

func PrintInstructions(builder *strings.Builder, reader *bytes.Reader) error {
	for reader.Len() > 0 {
		offset := reader.Size() - int64(reader.Len())
		err := PrintInstruction(builder, reader)
		if err != nil {
			return fmt.Errorf("failed to print instruction at offset %d: %w", offset, err)
		}
		builder.WriteByte('\n')
	}
	return nil
}

func PrintInstruction(builder *strings.Builder, reader *bytes.Reader) error {

	rawOpcode, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read opcode: %w", err)
	}
	opcode := Opcode(rawOpcode)

	builder.WriteString(opcode.String())

	switch opcode {

	// opcodes with one operand
	case GetConstant,
		GetLocal,
		SetLocal,
		GetGlobal,
		SetGlobal,
		Jump,
		JumpIfFalse,
		Transfer:

		operand, err := readIntOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read operand: %w", err)
		}

		builder.WriteByte(' ')
		_, _ = fmt.Fprint(builder, operand)

	case New:
		kind, err := readIntOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read kind operand: %w", err)
		}

		typeIndex, err := readIntOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read type index operand: %w", err)
		}

		_, _ = fmt.Fprintf(builder, " kind:%d typeIndex:%d", kind, typeIndex)

	case Cast:
		typeIndex, err := readIntOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read type index operand: %w", err)
		}

		castKind, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read cast kind operand: %w", err)
		}

		_, _ = fmt.Fprintf(builder, " typeIndex:%d castKind:%d", typeIndex, castKind)

	case Path:
		domain, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read domain operand: %w", err)
		}

		identifier, err := readStringOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read identifier operand: %w", err)
		}

		_, _ = fmt.Fprintf(builder, " domain:%d identifier:%q", domain, identifier)

	case InvokeDynamic:
		// Function name
		funcName, err := readStringOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read function name operand: %w", err)
		}
		_, _ = fmt.Fprintf(builder, " funcName:%q", funcName)

		// Type parameters
		err = printTypeParameters(builder, reader)
		if err != nil {
			return fmt.Errorf("failed to read type parameters: %w", err)
		}

		// Argument count
		argsCount, err := readIntOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read argument count operand: %w", err)
		}
		_, _ = fmt.Fprintf(builder, " argsCount:%d", argsCount)

	case Invoke:
		err := printTypeParameters(builder, reader)
		if err != nil {
			return fmt.Errorf("failed to read type parameters: %w", err)
		}

	// opcodes with no operands
	case Unknown,
		Return,
		ReturnValue,
		IntAdd,
		IntSubtract,
		IntMultiply,
		IntDivide,
		IntMod,
		IntLess,
		IntGreater,
		IntLessOrEqual,
		IntGreaterOrEqual,
		Equal,
		NotEqual,
		Unwrap,
		Destroy,
		True,
		False,
		Nil,
		NewArray,
		NewDictionary,
		NewRef,
		GetField,
		SetField,
		SetIndex,
		GetIndex,
		Drop,
		Dup:
		// no operands
	}

	return nil
}

func readIntOperand(reader *bytes.Reader) (operand int, err error) {
	first, err := reader.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("failed to read first byte of int operand: %w", err)
	}

	second, err := reader.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("failed to read second byte of int operand: %w", err)
	}

	operand = int(uint16(first)<<8 | uint16(second))
	return operand, nil
}

func readStringOperand(reader *bytes.Reader) (operand string, err error) {
	stringLength, err := readIntOperand(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read string length of string operand: %w", err)
	}

	stringBytes := make([]byte, stringLength)
	readLength, err := reader.Read(stringBytes)
	if err != nil {
		return "", fmt.Errorf("failed to read string bytes of string operand: %w", err)
	}
	if readLength != stringLength {
		return "", fmt.Errorf(
			"failed to read all bytes of string operand: expected %d, got %d",
			stringLength,
			readLength,
		)
	}

	return string(stringBytes), nil
}

func printTypeParameters(builder *strings.Builder, reader *bytes.Reader) error {
	typeParamCount, err := readIntOperand(reader)
	if err != nil {
		return fmt.Errorf("failed to read type parameter count operand: %w", err)
	}
	_, _ = fmt.Fprintf(builder, " typeParamCount:%d typeParams:[", typeParamCount)

	for i := 0; i < typeParamCount; i++ {
		if i > 0 {
			builder.WriteString(", ")
		}

		typeIndex, err := readIntOperand(reader)
		if err != nil {
			return fmt.Errorf("failed to read type index operand: %w", err)
		}

		_, _ = fmt.Fprint(builder, typeIndex)
	}
	builder.WriteByte(']')

	return nil
}
