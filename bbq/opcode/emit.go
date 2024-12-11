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

type Instruction interface {
	Encode(code *[]byte)
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

type InstructionTrue struct{}

func (InstructionTrue) Encode(code *[]byte) {
	emitOpcode(code, True)
}

// False

type InstructionFalse struct{}

func (InstructionFalse) Encode(code *[]byte) {
	emitOpcode(code, False)
}

// Nil

type InstructionNil struct{}

func (InstructionNil) Encode(code *[]byte) {
	emitOpcode(code, Nil)
}

// Dup

type InstructionDup struct{}

func (InstructionDup) Encode(code *[]byte) {
	emitOpcode(code, Dup)
}

// Drop

type InstructionDrop struct{}

func (InstructionDrop) Encode(code *[]byte) {
	emitOpcode(code, Drop)
}

// GetConstant

type InstructionGetConstant struct {
	ConstantIndex uint16
}

func (ins InstructionGetConstant) Encode(code *[]byte) {
	emitOpcode(code, GetConstant)
	emitUint16(code, ins.ConstantIndex)
}

func DecodeGetConstant(ip *uint16, code []byte) (ins InstructionGetConstant) {
	ins.ConstantIndex = decodeUint16(ip, code)
	return ins
}

// Jump

type InstructionJump struct {
	Target uint16
}

func (ins InstructionJump) Encode(code *[]byte) {
	emitOpcode(code, Jump)
	emitUint16(code, ins.Target)
}

func DecodeJump(ip *uint16, code []byte) (ins InstructionJump) {
	ins.Target = decodeUint16(ip, code)
	return ins
}

// JumpIfFalse

type InstructionJumpIfFalse struct {
	Target uint16
}

func (ins InstructionJumpIfFalse) Encode(code *[]byte) {
	emitOpcode(code, JumpIfFalse)
	emitUint16(code, ins.Target)
}

func DecodeJumpIfFalse(ip *uint16, code []byte) (ins InstructionJumpIfFalse) {
	ins.Target = decodeUint16(ip, code)
	return ins
}

func PatchJump(code *[]byte, opcodeOffset int, newTarget uint16) {
	first, second := encodeUint16(newTarget)
	(*code)[opcodeOffset+1] = first
	(*code)[opcodeOffset+2] = second
}

// Return

type InstructionReturn struct{}

func (InstructionReturn) Encode(code *[]byte) {
	emitOpcode(code, Return)
}

// ReturnValue

type InstructionReturnValue struct{}

func (InstructionReturnValue) Encode(code *[]byte) {
	emitOpcode(code, ReturnValue)
}

// GetLocal

type InstructionGetLocal struct {
	LocalIndex uint16
}

func (ins InstructionGetLocal) Encode(code *[]byte) {
	emitOpcode(code, GetLocal)
	emitUint16(code, ins.LocalIndex)
}

func DecodeGetLocal(ip *uint16, code []byte) (ins InstructionGetLocal) {
	ins.LocalIndex = decodeUint16(ip, code)
	return ins
}

// SetLocal

type InstructionSetLocal struct {
	LocalIndex uint16
}

func (ins InstructionSetLocal) Encode(code *[]byte) {
	emitOpcode(code, SetLocal)
	emitUint16(code, ins.LocalIndex)
}

func DecodeSetLocal(ip *uint16, code []byte) (ins InstructionSetLocal) {
	ins.LocalIndex = decodeUint16(ip, code)
	return ins
}

// GetGlobal

type InstructionGetGlobal struct {
	GlobalIndex uint16
}

func (ins InstructionGetGlobal) Encode(code *[]byte) {
	emitOpcode(code, GetGlobal)
	emitUint16(code, ins.GlobalIndex)
}

func DecodeGetGlobal(ip *uint16, code []byte) (ins InstructionGetGlobal) {
	ins.GlobalIndex = decodeUint16(ip, code)
	return ins
}

// SetGlobal

type InstructionSetGlobal struct {
	GlobalIndex uint16
}

func (ins InstructionSetGlobal) Encode(code *[]byte) {
	emitOpcode(code, SetGlobal)
	emitUint16(code, ins.GlobalIndex)
}

func DecodeSetGlobal(ip *uint16, code []byte) (ins InstructionSetGlobal) {
	ins.GlobalIndex = decodeUint16(ip, code)
	return ins
}

// GetField

type InstructionGetField struct{}

func (InstructionGetField) Encode(code *[]byte) {
	emitOpcode(code, GetField)
}

// SetField

type InstructionSetField struct{}

func (InstructionSetField) Encode(code *[]byte) {
	emitOpcode(code, SetField)
}

// GetIndex

type InstructionGetIndex struct{}

func (InstructionGetIndex) Encode(code *[]byte) {
	emitOpcode(code, GetIndex)
}

// SetIndex

type InstructionSetIndex struct{}

func (InstructionSetIndex) Encode(code *[]byte) {
	emitOpcode(code, SetIndex)
}

// NewArray

type InstructionNewArray struct {
	TypeIndex  uint16
	Size       uint16
	IsResource bool
}

func (ins InstructionNewArray) Encode(code *[]byte) {
	emitOpcode(code, NewArray)
	emitUint16(code, ins.TypeIndex)
	emitUint16(code, ins.Size)
	emitBool(code, ins.IsResource)
}

func DecodeNewArray(ip *uint16, code []byte) (ins InstructionNewArray) {
	ins.TypeIndex = decodeUint16(ip, code)
	ins.Size = decodeUint16(ip, code)
	ins.IsResource = decodeBool(ip, code)
	return ins
}

// IntAdd

type InstructionIntAdd struct{}

func (InstructionIntAdd) Encode(code *[]byte) {
	emitOpcode(code, IntAdd)
}

// IntSubtract

type InstructionIntSubtract struct{}

func (InstructionIntSubtract) Encode(code *[]byte) {
	emitOpcode(code, IntSubtract)
}

// IntMultiply

type InstructionIntMultiply struct{}

func (InstructionIntMultiply) Encode(code *[]byte) {
	emitOpcode(code, IntMultiply)
}

// IntDivide

type InstructionIntDivide struct{}

func (InstructionIntDivide) Encode(code *[]byte) {
	emitOpcode(code, IntDivide)
}

// IntMod

type InstructionIntMod struct{}

func (InstructionIntMod) Encode(code *[]byte) {
	emitOpcode(code, IntMod)
}

// Equal

type InstructionEqual struct{}

func (InstructionEqual) Encode(code *[]byte) {
	emitOpcode(code, Equal)
}

// NotEqual

type InstructionNotEqual struct{}

func (InstructionNotEqual) Encode(code *[]byte) {
	emitOpcode(code, NotEqual)
}

// IntLess

type InstructionIntLess struct{}

func (InstructionIntLess) Encode(code *[]byte) {
	emitOpcode(code, IntLess)
}

// IntLessOrEqual

type InstructionIntLessOrEqual struct{}

func (InstructionIntLessOrEqual) Encode(code *[]byte) {
	emitOpcode(code, IntLessOrEqual)
}

// IntGreater

type InstructionIntGreater struct{}

func (InstructionIntGreater) Encode(code *[]byte) {
	emitOpcode(code, IntGreater)
}

// IntGreaterOrEqual

type InstructionIntGreaterOrEqual struct{}

func (InstructionIntGreaterOrEqual) Encode(code *[]byte) {
	emitOpcode(code, IntGreaterOrEqual)
}

// Unwrap

type InstructionUnwrap struct{}

func (InstructionUnwrap) Encode(code *[]byte) {
	emitOpcode(code, Unwrap)
}

// Cast

type InstructionCast struct {
	TypeIndex uint16
	Kind      CastKind
}

func (ins InstructionCast) Encode(code *[]byte) {
	emitOpcode(code, Cast)
	emitUint16(code, ins.TypeIndex)
	emitByte(code, byte(ins.Kind))
}

func DecodeCast(ip *uint16, code []byte) (ins InstructionCast) {
	ins.TypeIndex = decodeUint16(ip, code)
	ins.Kind = CastKind(decodeByte(ip, code))
	return ins
}

// Destroy

type InstructionDestroy struct{}

func (InstructionDestroy) Encode(code *[]byte) {
	emitOpcode(code, Destroy)
}

// Transfer

type InstructionTransfer struct {
	TypeIndex uint16
}

func (ins InstructionTransfer) Encode(code *[]byte) {
	emitOpcode(code, Transfer)
	emitUint16(code, ins.TypeIndex)
}

func DecodeTransfer(ip *uint16, code []byte) (ins InstructionTransfer) {
	ins.TypeIndex = decodeUint16(ip, code)
	return ins
}

// NewRef

type InstructionNewRef struct {
	TypeIndex uint16
}

func (ins InstructionNewRef) Encode(code *[]byte) {
	emitOpcode(code, NewRef)
	emitUint16(code, ins.TypeIndex)
}

func DecodeNewRef(ip *uint16, code []byte) (ins InstructionNewRef) {
	ins.TypeIndex = decodeUint16(ip, code)
	return ins
}

// Path

type InstructionPath struct {
	Domain     common.PathDomain
	Identifier string
}

func (ins InstructionPath) Encode(code *[]byte) {
	emitOpcode(code, Path)
	emitByte(code, byte(ins.Domain))
	emitString(code, ins.Identifier)
}

func DecodePath(ip *uint16, code []byte) (ins InstructionPath) {
	ins.Domain = common.PathDomain(decodeByte(ip, code))
	ins.Identifier = decodeString(ip, code)
	return ins
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

type InstructionInvoke struct {
	TypeArgs []uint16
}

func (ins InstructionInvoke) Encode(code *[]byte) {
	emitOpcode(code, Invoke)
	emitTypeArgs(code, ins.TypeArgs)
}

func DecodeInvoke(ip *uint16, code []byte) (ins InstructionInvoke) {
	ins.TypeArgs = decodeTypeArgs(ip, code)
	return ins
}

// New

type InstructionNew struct {
	Kind      uint16
	TypeIndex uint16
}

func (ins InstructionNew) Encode(code *[]byte) {
	emitOpcode(code, New)
	emitUint16(code, ins.Kind)
	emitUint16(code, ins.TypeIndex)
}

func DecodeNew(ip *uint16, code []byte) (ins InstructionNew) {
	ins.Kind = decodeUint16(ip, code)
	ins.TypeIndex = decodeUint16(ip, code)
	return ins
}

// InvokeDynamic

type InstructionInvokeDynamic struct {
	Name     string
	TypeArgs []uint16
	ArgCount uint16
}

func (ins InstructionInvokeDynamic) Encode(code *[]byte) {
	emitOpcode(code, InvokeDynamic)
	emitString(code, ins.Name)
	emitTypeArgs(code, ins.TypeArgs)
	emitUint16(code, ins.ArgCount)
}

func DecodeInvokeDynamic(ip *uint16, code []byte) (ins InstructionInvokeDynamic) {
	ins.Name = decodeString(ip, code)
	ins.TypeArgs = decodeTypeArgs(ip, code)
	ins.ArgCount = decodeUint16(ip, code)
	return
}
