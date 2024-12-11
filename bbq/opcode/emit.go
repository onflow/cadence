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

func emit(code *[]byte, opcode Opcode, args ...byte) int {
	offset := len(*code)
	*code = append(*code, byte(opcode))
	*code = append(*code, args...)
	return offset
}

// uint16

// encodeUint16 encodes the given uint16 in big-endian representation
func encodeUint16(v uint16) (byte, byte) {
	return byte((v >> 8) & 0xff),
		byte(v & 0xff)
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

// Bool

func decodeBool(ip *uint16, code []byte) bool {
	return decodeByte(ip, code) == 1
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
	sizeFirst, sizeSecond := encodeUint16(uint16(len(str)))
	*code = append(*code, sizeFirst, sizeSecond)
	*code = append(*code, []byte(str)...)
}

// True

func EmitTrue(code *[]byte) {
	emit(code, True)
}

// False

func EmitFalse(code *[]byte) {
	emit(code, False)
}

// Nil

func EmitNil(code *[]byte) {
	emit(code, Nil)
}

// Dup

func EmitDup(code *[]byte) {
	emit(code, Dup)
}

// Drop

func EmitDrop(code *[]byte) {
	emit(code, Drop)
}

// GetConstant

func EmitGetConstant(code *[]byte, constantIndex uint16) int {
	first, second := encodeUint16(constantIndex)
	return emit(code, GetConstant, first, second)
}

func DecodeGetConstant(ip *uint16, code []byte) (constantIndex uint16) {
	return decodeUint16(ip, code)
}

// Jump

func EmitJump(code *[]byte, target uint16) int {
	first, second := encodeUint16(target)
	return emit(code, Jump, first, second)
}

func DecodeJump(ip *uint16, code []byte) (target uint16) {
	return decodeUint16(ip, code)
}

// JumpIfFalse

func EmitJumpIfFalse(code *[]byte, target uint16) int {
	first, second := encodeUint16(target)
	return emit(code, JumpIfFalse, first, second)
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
	return emit(code, Return)
}

// ReturnValue

func EmitReturnValue(code *[]byte) int {
	return emit(code, ReturnValue)
}

// GetLocal

func EmitGetLocal(code *[]byte, localIndex uint16) {
	first, second := encodeUint16(localIndex)
	emit(code, GetLocal, first, second)
}

func DecodeGetLocal(ip *uint16, code []byte) (localIndex uint16) {
	return decodeUint16(ip, code)
}

// SetLocal

func EmitSetLocal(code *[]byte, localIndex uint16) {
	first, second := encodeUint16(localIndex)
	emit(code, SetLocal, first, second)
}

func DecodeSetLocal(ip *uint16, code []byte) (localIndex uint16) {
	return decodeUint16(ip, code)
}

// GetGlobal

func EmitGetGlobal(code *[]byte, globalIndex uint16) {
	first, second := encodeUint16(globalIndex)
	emit(code, GetGlobal, first, second)
}

func DecodeGetGlobal(ip *uint16, code []byte) (globalIndex uint16) {
	return decodeUint16(ip, code)
}

// SetGlobal

func EmitSetGlobal(code *[]byte, globalIndex uint16) {
	first, second := encodeUint16(globalIndex)
	emit(code, SetGlobal, first, second)
}

func DecodeSetGlobal(ip *uint16, code []byte) (globalIndex uint16) {
	return decodeUint16(ip, code)
}

// GetField

func EmitGetField(code *[]byte) {
	emit(code, GetField)
}

// SetField

func EmitSetField(code *[]byte) {
	emit(code, SetField)
}

// GetIndex

func EmitGetIndex(code *[]byte) {
	emit(code, GetIndex)
}

// SetIndex

func EmitSetIndex(code *[]byte) {
	emit(code, SetIndex)
}

// NewArray

func EmitNewArray(code *[]byte, typeIndex uint16, size uint16, isResource bool) {
	typeIndexFirst, typeIndexSecond := encodeUint16(typeIndex)
	sizeFirst, sizeSecond := encodeUint16(size)
	var isResourceFlag byte
	if isResource {
		isResourceFlag = 1
	}
	emit(
		code,
		NewArray,
		typeIndexFirst, typeIndexSecond,
		sizeFirst, sizeSecond,
		isResourceFlag,
	)
}

func DecodeNewArray(ip *uint16, code []byte) (typeIndex uint16, size uint16, isResource bool) {
	typeIndex = decodeUint16(ip, code)
	size = decodeUint16(ip, code)
	isResource = decodeBool(ip, code)
	return typeIndex, size, isResource
}

// IntAdd

func EmitIntAdd(code *[]byte) {
	emit(code, IntAdd)
}

// IntSubtract

func EmitIntSubtract(code *[]byte) {
	emit(code, IntSubtract)
}

// IntMultiply

func EmitIntMultiply(code *[]byte) {
	emit(code, IntMultiply)
}

// IntDivide

func EmitIntDivide(code *[]byte) {
	emit(code, IntDivide)
}

// IntMod

func EmitIntMod(code *[]byte) {
	emit(code, IntMod)
}

// IntEqual

func EmitEqual(code *[]byte) {
	emit(code, Equal)
}

// IntNotEqual

func EmitNotEqual(code *[]byte) {
	emit(code, NotEqual)
}

// IntLess

func EmitIntLess(code *[]byte) {
	emit(code, IntLess)
}

// IntLessOrEqual

func EmitIntLessOrEqual(code *[]byte) {
	emit(code, IntLessOrEqual)
}

// IntGreater

func EmitIntGreater(code *[]byte) {
	emit(code, IntGreater)
}

// IntGreaterOrEqual

func EmitIntGreaterOrEqual(code *[]byte) {
	emit(code, IntGreaterOrEqual)
}

// Unwrap

func EmitUnwrap(code *[]byte) {
	emit(code, Unwrap)
}

// Cast

func EmitCast(code *[]byte, typeIndex uint16, kind CastKind) {
	first, second := encodeUint16(typeIndex)
	emit(code, Cast, first, second, byte(kind))
}

func DecodeCast(ip *uint16, code []byte) (typeIndex uint16, kind byte) {
	typeIndex = decodeUint16(ip, code)
	kind = decodeByte(ip, code)
	return typeIndex, kind
}

// Destroy

func EmitDestroy(code *[]byte) {
	emit(code, Destroy)
}

// Transfer

func EmitTransfer(code *[]byte, typeIndex uint16) {
	first, second := encodeUint16(typeIndex)
	emit(code, Transfer, first, second)
}

func DecodeTransfer(ip *uint16, code []byte) (typeIndex uint16) {
	return decodeUint16(ip, code)
}

// NewRef

func EmitNewRef(code *[]byte, typeIndex uint16) {
	first, second := encodeUint16(typeIndex)
	emit(code, NewRef, first, second)
}

func DecodeNewRef(ip *uint16, code []byte) (typeIndex uint16) {
	return decodeUint16(ip, code)
}

// Path

func EmitPath(code *[]byte, domain common.PathDomain, identifier string) {
	emit(code, Path)

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
	first, second := encodeUint16(uint16(len(typeArgs)))
	*code = append(*code, first, second)
	for _, typeArg := range typeArgs {
		first, second := encodeUint16(typeArg)
		*code = append(*code, first, second)
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
	emit(code, Invoke)
	emitTypeArgs(code, typeArgs)
}

func DecodeInvoke(ip *uint16, code []byte) (typeArgs []uint16) {
	return decodeTypeArgs(ip, code)
}

// New

func EmitNew(code *[]byte, kind uint16, typeIndex uint16) {
	firstKind, secondKind := encodeUint16(kind)
	firstTypeIndex, secondTypeIndex := encodeUint16(typeIndex)
	emit(
		code,
		New,
		firstKind, secondKind,
		firstTypeIndex, secondTypeIndex,
	)
}

func DecodeNew(ip *uint16, code []byte) (kind uint16, typeIndex uint16) {
	kind = decodeUint16(ip, code)
	typeIndex = decodeUint16(ip, code)
	return kind, typeIndex
}

// InvokeDynamic

func EmitInvokeDynamic(code *[]byte, name string, typeArgs []uint16, argCount uint16) {
	emit(code, InvokeDynamic)
	emitString(code, name)
	emitTypeArgs(code, typeArgs)
	argsCountFirst, argsCountSecond := encodeUint16(argCount)
	*code = append(*code, argsCountFirst, argsCountSecond)
}

func DecodeInvokeDynamic(ip *uint16, code []byte) (name string, typeArgs []uint16, argCount uint16) {
	name = decodeString(ip, code)
	typeArgs = decodeTypeArgs(ip, code)
	argCount = decodeUint16(ip, code)
	return name, typeArgs, argCount
}
