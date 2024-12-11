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

// encodeUint16 encodes the given uint16 in big-endian representation
func encodeUint16(v uint16) (byte, byte) {
	return byte((v >> 8) & 0xff),
		byte(v & 0xff)
}

func EmitTrue(code *[]byte) {
	emit(code, True)
}

func EmitFalse(code *[]byte) {
	emit(code, False)
}

func EmitNil(code *[]byte) {
	emit(code, Nil)
}

func EmitDup(code *[]byte) {
	emit(code, Dup)
}

func EmitDrop(code *[]byte) {
	emit(code, Drop)
}

func EmitGetConstant(code *[]byte, index uint16) int {
	first, second := encodeUint16(index)
	return emit(code, GetConstant, first, second)
}

func EmitJump(code *[]byte, target uint16) int {
	first, second := encodeUint16(target)
	return emit(code, Jump, first, second)
}

func EmitJumpIfFalse(code *[]byte, target uint16) int {
	first, second := encodeUint16(target)
	return emit(code, JumpIfFalse, first, second)
}

func PatchJump(code *[]byte, opcodeOffset int, target uint16) {
	first, second := encodeUint16(target)
	(*code)[opcodeOffset+1] = first
	(*code)[opcodeOffset+2] = second
}

func EmitReturn(code *[]byte) int {
	return emit(code, Return)
}

func EmitReturnValue(code *[]byte) int {
	return emit(code, ReturnValue)
}

func EmitGetLocal(code *[]byte, index uint16) {
	first, second := encodeUint16(index)
	emit(code, GetLocal, first, second)
}

func EmitSetLocal(code *[]byte, index uint16) {
	first, second := encodeUint16(index)
	emit(code, SetLocal, first, second)
}

func EmitGetGlobal(code *[]byte, index uint16) {
	first, second := encodeUint16(index)
	emit(code, GetGlobal, first, second)
}
func EmitSetGlobal(code *[]byte, index uint16) {
	first, second := encodeUint16(index)
	emit(code, SetGlobal, first, second)
}

func EmitGetField(code *[]byte) {
	emit(code, GetField)
}

func EmitSetField(code *[]byte) {
	emit(code, SetField)
}

func EmitGetIndex(code *[]byte) {
	emit(code, GetIndex)
}

func EmitSetIndex(code *[]byte) {
	emit(code, SetIndex)
}

func EmitNewArray(code *[]byte, index uint16, size uint16, isResource bool) {
	indexFirst, indexSecond := encodeUint16(index)
	sizeFirst, sizeSecond := encodeUint16(size)
	var isResourceFlag byte
	if isResource {
		isResourceFlag = 1
	}
	emit(
		code,
		NewArray,
		indexFirst, indexSecond,
		sizeFirst, sizeSecond,
		isResourceFlag,
	)
}

func EmitIntAdd(code *[]byte) {
	emit(code, IntAdd)
}

func EmitIntSubtract(code *[]byte) {
	emit(code, IntSubtract)
}

func EmitIntMultiply(code *[]byte) {
	emit(code, IntMultiply)
}

func EmitIntDivide(code *[]byte) {
	emit(code, IntDivide)
}

func EmitIntMod(code *[]byte) {
	emit(code, IntMod)
}

func EmitEqual(code *[]byte) {
	emit(code, Equal)
}

func EmitNotEqual(code *[]byte) {
	emit(code, NotEqual)
}

func EmitIntLess(code *[]byte) {
	emit(code, IntLess)
}

func EmitIntLessOrEqual(code *[]byte) {
	emit(code, IntLessOrEqual)
}

func EmitIntGreater(code *[]byte) {
	emit(code, IntGreater)
}

func EmitIntGreaterOrEqual(code *[]byte) {
	emit(code, IntGreaterOrEqual)
}

func EmitUnwrap(code *[]byte) {
	emit(code, Unwrap)
}

func EmitCast(code *[]byte, index uint16, kind CastKind) {
	first, second := encodeUint16(index)
	emit(code, Cast, first, second, byte(kind))
}

func EmitDestroy(code *[]byte) {
	emit(code, Destroy)
}

func EmitTransfer(code *[]byte, index uint16) {
	first, second := encodeUint16(index)
	emit(code, Transfer, first, second)
}

func EmitNewRef(code *[]byte, index uint16) {
	first, second := encodeUint16(index)
	emit(code, NewRef, first, second)
}

func EmitPath(code *[]byte, domain common.PathDomain, identifier string) {
	emit(code, Path)

	*code = append(*code, byte(domain))

	identifierLength := len(identifier)

	identifierSizeFirst, identifierSizeSecond := encodeUint16(uint16(identifierLength))
	*code = append(*code, identifierSizeFirst, identifierSizeSecond)

	*code = append(*code, identifier...)
}

func emitTypeArgs(code *[]byte, typeArgs []uint16) {
	first, second := encodeUint16(uint16(len(typeArgs)))
	*code = append(*code, first, second)
	for _, typeArg := range typeArgs {
		first, second := encodeUint16(typeArg)
		*code = append(*code, first, second)
	}
}

func EmitInvoke(code *[]byte, typeArgs []uint16) {
	emit(code, Invoke)
	emitTypeArgs(code, typeArgs)
}

func EmitNew(code *[]byte, kind uint16, index uint16) {
	firstKind, secondKind := encodeUint16(kind)
	firstIndex, secondIndex := encodeUint16(index)
	emit(
		code,
		New,
		firstKind, secondKind,
		firstIndex, secondIndex,
	)
}

func EmitInvokeDynamic(code *[]byte, name string, typeArgs []uint16, argCount uint16) {
	emit(code, InvokeDynamic)

	funcNameSizeFirst, funcNameSizeSecond := encodeUint16(uint16(len(name)))
	*code = append(*code, funcNameSizeFirst, funcNameSizeSecond)

	*code = append(*code, []byte(name)...)

	emitTypeArgs(code, typeArgs)

	argsCountFirst, argsCountSecond := encodeUint16(argCount)
	*code = append(*code, argsCountFirst, argsCountSecond)
}
