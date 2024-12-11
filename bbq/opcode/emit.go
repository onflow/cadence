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
	"github.com/onflow/cadence/common"
)

func emitOpcode(code *[]byte, opcode Opcode) int {
	offset := len(*code)
	*code = append(*code, byte(opcode))
	return offset
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

// String

func decodeString(ip *uint16, code []byte) string {
	strLen := decodeUint16(ip, code)
	start := *ip
	*ip += strLen
	end := *ip
	return string(code[start:end])
}

func emitString(code *[]byte, str string) {
	emitUint16(code, uint16(len(str)))
	*code = append(*code, []byte(str)...)
}

// True

func EmitTrue(code *[]byte) {
	emitOpcode(code, True)
}

// False

func EmitFalse(code *[]byte) {
	emitOpcode(code, False)
}

// Nil

func EmitNil(code *[]byte) {
	emitOpcode(code, Nil)
}

// Dup

func EmitDup(code *[]byte) {
	emitOpcode(code, Dup)
}

// Drop

func EmitDrop(code *[]byte) {
	emitOpcode(code, Drop)
}

// GetConstant

func EmitGetConstant(code *[]byte, constantIndex uint16) (offset int) {
	offset = emitOpcode(code, GetConstant)
	emitUint16(code, constantIndex)
	return offset
}

func DecodeGetConstant(ip *uint16, code []byte) (constantIndex uint16) {
	return decodeUint16(ip, code)
}

// Jump

func EmitJump(code *[]byte, target uint16) (offset int) {
	offset = emitOpcode(code, Jump)
	emitUint16(code, target)
	return offset
}

func DecodeJump(ip *uint16, code []byte) (target uint16) {
	return decodeUint16(ip, code)
}

// JumpIfFalse

func EmitJumpIfFalse(code *[]byte, target uint16) (offset int) {
	offset = emitOpcode(code, JumpIfFalse)
	emitUint16(code, target)
	return offset
}

func DecodeJumpIfFalse(ip *uint16, code []byte) (target uint16) {
	return decodeUint16(ip, code)
}

func PatchJump(code *[]byte, opcodeOffset int, target uint16) {
	first, second := encodeUint16(target)
	(*code)[opcodeOffset+1] = first
	(*code)[opcodeOffset+2] = second
}

// Return

func EmitReturn(code *[]byte) int {
	return emitOpcode(code, Return)
}

// ReturnValue

func EmitReturnValue(code *[]byte) int {
	return emitOpcode(code, ReturnValue)
}

// GetLocal

func EmitGetLocal(code *[]byte, localIndex uint16) {
	emitOpcode(code, GetLocal)
	emitUint16(code, localIndex)
}

func DecodeGetLocal(ip *uint16, code []byte) (localIndex uint16) {
	return decodeUint16(ip, code)
}

// SetLocal

func EmitSetLocal(code *[]byte, localIndex uint16) {
	emitOpcode(code, SetLocal)
	emitUint16(code, localIndex)
}

func DecodeSetLocal(ip *uint16, code []byte) (localIndex uint16) {
	return decodeUint16(ip, code)
}

// GetGlobal

func EmitGetGlobal(code *[]byte, globalIndex uint16) {
	emitOpcode(code, GetGlobal)
	emitUint16(code, globalIndex)
}

func DecodeGetGlobal(ip *uint16, code []byte) (globalIndex uint16) {
	return decodeUint16(ip, code)
}

// SetGlobal

func EmitSetGlobal(code *[]byte, globalIndex uint16) {
	emitOpcode(code, SetGlobal)
	emitUint16(code, globalIndex)
}

func DecodeSetGlobal(ip *uint16, code []byte) (globalIndex uint16) {
	return decodeUint16(ip, code)
}

// GetField

func EmitGetField(code *[]byte) {
	emitOpcode(code, GetField)
}

// SetField

func EmitSetField(code *[]byte) {
	emitOpcode(code, SetField)
}

// GetIndex

func EmitGetIndex(code *[]byte) {
	emitOpcode(code, GetIndex)
}

// SetIndex

func EmitSetIndex(code *[]byte) {
	emitOpcode(code, SetIndex)
}

// NewArray

func EmitNewArray(code *[]byte, typeIndex uint16, size uint16, isResource bool) {
	emitOpcode(code, NewArray)
	emitUint16(code, typeIndex)
	emitUint16(code, size)
	emitBool(code, isResource)
}

func DecodeNewArray(ip *uint16, code []byte) (typeIndex uint16, size uint16, isResource bool) {
	typeIndex = decodeUint16(ip, code)
	size = decodeUint16(ip, code)
	isResource = decodeBool(ip, code)
	return typeIndex, size, isResource
}

// IntAdd

func EmitIntAdd(code *[]byte) {
	emitOpcode(code, IntAdd)
}

// IntSubtract

func EmitIntSubtract(code *[]byte) {
	emitOpcode(code, IntSubtract)
}

// IntMultiply

func EmitIntMultiply(code *[]byte) {
	emitOpcode(code, IntMultiply)
}

// IntDivide

func EmitIntDivide(code *[]byte) {
	emitOpcode(code, IntDivide)
}

// IntMod

func EmitIntMod(code *[]byte) {
	emitOpcode(code, IntMod)
}

// IntEqual

func EmitEqual(code *[]byte) {
	emitOpcode(code, Equal)
}

// IntNotEqual

func EmitNotEqual(code *[]byte) {
	emitOpcode(code, NotEqual)
}

// IntLess

func EmitIntLess(code *[]byte) {
	emitOpcode(code, IntLess)
}

// IntLessOrEqual

func EmitIntLessOrEqual(code *[]byte) {
	emitOpcode(code, IntLessOrEqual)
}

// IntGreater

func EmitIntGreater(code *[]byte) {
	emitOpcode(code, IntGreater)
}

// IntGreaterOrEqual

func EmitIntGreaterOrEqual(code *[]byte) {
	emitOpcode(code, IntGreaterOrEqual)
}

// Unwrap

func EmitUnwrap(code *[]byte) {
	emitOpcode(code, Unwrap)
}

// Cast

func EmitCast(code *[]byte, typeIndex uint16, kind CastKind) {
	emitOpcode(code, Cast)
	emitUint16(code, typeIndex)
	emitByte(code, byte(kind))
}

func DecodeCast(ip *uint16, code []byte) (typeIndex uint16, kind CastKind) {
	typeIndex = decodeUint16(ip, code)
	kind = CastKind(decodeByte(ip, code))
	return typeIndex, kind
}

// Destroy

func EmitDestroy(code *[]byte) {
	emitOpcode(code, Destroy)
}

// Transfer

func EmitTransfer(code *[]byte, typeIndex uint16) {
	emitOpcode(code, Transfer)
	emitUint16(code, typeIndex)
}

func DecodeTransfer(ip *uint16, code []byte) (typeIndex uint16) {
	return decodeUint16(ip, code)
}

// NewRef

func EmitNewRef(code *[]byte, typeIndex uint16) {
	emitOpcode(code, NewRef)
	emitUint16(code, typeIndex)
}

func DecodeNewRef(ip *uint16, code []byte) (typeIndex uint16) {
	return decodeUint16(ip, code)
}

// Path

func EmitPath(code *[]byte, domain common.PathDomain, identifier string) {
	emitOpcode(code, Path)

	*code = append(*code, byte(domain))

	emitString(code, identifier)
}

func DecodePath(ip *uint16, code []byte) (domain byte, identifier string) {
	domain = decodeByte(ip, code)
	identifier = decodeString(ip, code)
	return domain, identifier
}

// Type arguments

func emitTypeArgs(code *[]byte, typeArgs []uint16) {
	emitUint16(code, uint16(len(typeArgs)))
	for _, typeArg := range typeArgs {
		emitUint16(code, typeArg)
	}
}

func decodeTypeArgs(ip *uint16, code []byte) (typeArgs []uint16) {
	typeArgCount := decodeUint16(ip, code)
	for i := 0; i < int(typeArgCount); i++ {
		typeIndex := decodeUint16(ip, code)
		typeArgs = append(typeArgs, typeIndex)
	}
	return typeArgs
}

// Invoke

func EmitInvoke(code *[]byte, typeArgs []uint16) {
	emitOpcode(code, Invoke)
	emitTypeArgs(code, typeArgs)
}

func DecodeInvoke(ip *uint16, code []byte) (typeArgs []uint16) {
	return decodeTypeArgs(ip, code)
}

// New

func EmitNew(code *[]byte, kind uint16, typeIndex uint16) {
	emitOpcode(code, New)
	emitUint16(code, kind)
	emitUint16(code, typeIndex)
}

func DecodeNew(ip *uint16, code []byte) (kind uint16, typeIndex uint16) {
	kind = decodeUint16(ip, code)
	typeIndex = decodeUint16(ip, code)
	return kind, typeIndex
}

// InvokeDynamic

func EmitInvokeDynamic(code *[]byte, name string, typeArgs []uint16, argCount uint16) {
	emitOpcode(code, InvokeDynamic)
	emitString(code, name)
	emitTypeArgs(code, typeArgs)
	emitUint16(code, argCount)
}

func DecodeInvokeDynamic(ip *uint16, code []byte) (name string, typeArgs []uint16, argCount uint16) {
	name = decodeString(ip, code)
	typeArgs = decodeTypeArgs(ip, code)
	argCount = decodeUint16(ip, code)
	return name, typeArgs, argCount
}
