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

//go:generate go run ./gen/main.go instructions.yml instructions.go

package opcode

import (
	"fmt"
	"math"
	"strings"

	"github.com/onflow/cadence/common"
)

type Instruction interface {
	Encode(code *[]byte)
	String() string
	Opcode() Opcode
}

func emitOpcode(code *[]byte, opcode Opcode) {
	*code = append(*code, byte(opcode))
}

// uint16

// encodeUint16 encodes the given uint16 in big-endian representation
func encodeUint16(v uint16) (byte, byte) {
	return byte((v >> 8) & 0xff),
		byte(v & 0xff)
}

// emitUint16 encodes the given uint16 in big-endian representation
func emitUint16(code *[]byte, v uint16) {
	first, last := encodeUint16(v)
	*code = append(*code, first, last)
}

func decodeUint16(ip *uint16, code []byte) uint16 {
	first := code[*ip]
	last := code[*ip+1]
	*ip += 2
	return uint16(first)<<8 | uint16(last)
}

// Byte

func decodeByte(ip *uint16, code []byte) byte {
	byt := code[*ip]
	*ip += 1
	return byt
}

func emitByte(code *[]byte, b byte) {
	*code = append(*code, b)
}

// Bool

func decodeBool(ip *uint16, code []byte) bool {
	return decodeByte(ip, code) == 1
}

func emitBool(code *[]byte, v bool) {
	var b byte
	if v {
		b = 1
	}
	*code = append(*code, b)
}

// PathDomain

func decodePathDomain(ip *uint16, code []byte) common.PathDomain {
	return common.PathDomain(decodeByte(ip, code))
}

func emitPathDomain(code *[]byte, domain common.PathDomain) {
	emitByte(code, byte(domain))
}

// CompositeKind

func decodeCompositeKind(ip *uint16, code []byte) common.CompositeKind {
	return common.CompositeKind(decodeUint16(ip, code))
}

func emitCompositeKind(code *[]byte, kind common.CompositeKind) {
	emitUint16(code, uint16(kind))
}

// Uint16Array

func emitUint16Array(code *[]byte, values []uint16) {
	count := len(values)
	if count > math.MaxUint16 {
		panic(fmt.Errorf("uint16 array too large: %d", count))
	}
	emitUint16(code, uint16(count))

	for _, value := range values {
		emitUint16(code, value)
	}
}

func decodeUint16Array(ip *uint16, code []byte) (values []uint16) {
	count := decodeUint16(ip, code)
	for i := 0; i < int(count); i++ {
		value := decodeUint16(ip, code)
		values = append(values, value)
	}
	return values
}

// Upvalue

func emitUpvalue(code *[]byte, upvalue Upvalue) {
	emitUint16(code, upvalue.TargetIndex)
	emitBool(code, upvalue.IsLocal)
}

func decodeUpvalue(ip *uint16, code []byte) Upvalue {
	targetIndex := decodeUint16(ip, code)
	isLocal := decodeBool(ip, code)
	return Upvalue{
		TargetIndex: targetIndex,
		IsLocal:     isLocal,
	}
}

// UpvalueArray

func emitUpvalueArray(code *[]byte, upvalues []Upvalue) {
	count := len(upvalues)
	if count > math.MaxUint16 {
		panic(fmt.Errorf("uint16 array too large: %d", count))
	}
	emitUint16(code, uint16(count))

	for _, upvalue := range upvalues {
		emitUpvalue(code, upvalue)
	}
}

func decodeUpvalueArray(ip *uint16, code []byte) (upvalues []Upvalue) {
	count := decodeUint16(ip, code)
	for i := 0; i < int(count); i++ {
		upvalue := decodeUpvalue(ip, code)
		upvalues = append(upvalues, upvalue)
	}
	return upvalues
}

// Jump

func PatchJump(code *[]byte, opcodeOffset int, newTarget uint16) {
	first, second := encodeUint16(newTarget)
	(*code)[opcodeOffset+1] = first
	(*code)[opcodeOffset+2] = second
}

func DecodeInstructions(code []byte) []Instruction {
	var instructions []Instruction
	var ip uint16
	for ip < uint16(len(code)) {
		instruction := DecodeInstruction(&ip, code)
		instructions = append(instructions, instruction)
	}
	return instructions
}

// Instruction pretty print

func printfUInt16ArrayArgument(sb *strings.Builder, argName string, values []uint16) {
	_, _ = fmt.Fprintf(sb, " %s:[", argName)
	for i, value := range values {
		if i > 0 {
			_, _ = fmt.Fprint(sb, ", ")
		}
		_, _ = fmt.Fprintf(sb, "%d", value)
	}

	sb.WriteByte(']')
}

func printfUpvalueArrayArgument(sb *strings.Builder, argName string, upvalues []Upvalue) {
	_, _ = fmt.Fprintf(sb, " %s:[", argName)
	for i, upvalue := range upvalues {
		if i > 0 {
			_, _ = fmt.Fprint(sb, ", ")
		}
		_, _ = fmt.Fprintf(sb, "targetIndex:%d isLocal:%v", upvalue.TargetIndex, upvalue.IsLocal)
	}

	sb.WriteByte(']')
}

func printfArgument(sb *strings.Builder, fieldName string, v any) {
	_, _ = fmt.Fprintf(sb, " %s:%v", fieldName, v)
}
