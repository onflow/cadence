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
	"fmt"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
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

func (InstructionTrue) Opcode() Opcode {
	return True
}

func (ins InstructionTrue) String() string {
	return ins.Opcode().String()
}

func (ins InstructionTrue) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// False

type InstructionFalse struct{}

func (InstructionFalse) Opcode() Opcode {
	return False
}

func (ins InstructionFalse) String() string {
	return ins.Opcode().String()
}

func (ins InstructionFalse) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// Nil

type InstructionNil struct{}

func (InstructionNil) Opcode() Opcode {
	return Nil
}

func (ins InstructionNil) String() string {
	return ins.Opcode().String()
}

func (ins InstructionNil) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// Dup

type InstructionDup struct{}

func (InstructionDup) Opcode() Opcode {
	return Dup
}

func (ins InstructionDup) String() string {
	return ins.Opcode().String()
}

func (ins InstructionDup) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// Drop

type InstructionDrop struct{}

func (InstructionDrop) Opcode() Opcode {
	return Drop
}

func (ins InstructionDrop) String() string {
	return ins.Opcode().String()
}

func (ins InstructionDrop) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// GetConstant

type InstructionGetConstant struct {
	ConstantIndex uint16
}

func (InstructionGetConstant) Opcode() Opcode {
	return GetConstant
}

func (ins InstructionGetConstant) String() string {
	return fmt.Sprintf(
		"%s constantIndex:%d",
		ins.Opcode(),
		ins.ConstantIndex,
	)
}

func (ins InstructionGetConstant) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionJump) Opcode() Opcode {
	return Jump
}

func (ins InstructionJump) String() string {
	return fmt.Sprintf(
		"%s target:%d",
		ins.Opcode(),
		ins.Target,
	)
}

func (ins InstructionJump) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionJumpIfFalse) Opcode() Opcode {
	return JumpIfFalse
}

func (ins InstructionJumpIfFalse) String() string {
	return fmt.Sprintf(
		"%s target:%d",
		ins.Opcode(),
		ins.Target,
	)
}

func (ins InstructionJumpIfFalse) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionReturn) Opcode() Opcode {
	return Return
}

func (ins InstructionReturn) String() string {
	return ins.Opcode().String()
}

func (ins InstructionReturn) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// ReturnValue

type InstructionReturnValue struct{}

func (InstructionReturnValue) Opcode() Opcode {
	return ReturnValue
}

func (ins InstructionReturnValue) String() string {
	return ins.Opcode().String()
}

func (ins InstructionReturnValue) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// GetLocal

type InstructionGetLocal struct {
	LocalIndex uint16
}

func (InstructionGetLocal) Opcode() Opcode {
	return GetLocal
}

func (ins InstructionGetLocal) String() string {
	return fmt.Sprintf(
		"%s localIndex:%d",
		ins.Opcode(),
		ins.LocalIndex,
	)
}

func (ins InstructionGetLocal) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionSetLocal) Opcode() Opcode {
	return SetLocal
}

func (ins InstructionSetLocal) String() string {
	return fmt.Sprintf(
		"%s localIndex:%d",
		ins.Opcode(),
		ins.LocalIndex,
	)
}

func (ins InstructionSetLocal) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionGetGlobal) Opcode() Opcode {
	return GetGlobal
}

func (ins InstructionGetGlobal) String() string {
	return fmt.Sprintf(
		"%s globalIndex:%d",
		ins.Opcode(),
		ins.GlobalIndex,
	)
}

func (ins InstructionGetGlobal) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionSetGlobal) Opcode() Opcode {
	return SetGlobal
}

func (ins InstructionSetGlobal) String() string {
	return fmt.Sprintf(
		"%s globalIndex:%d",
		ins.Opcode(),
		ins.GlobalIndex,
	)
}

func (ins InstructionSetGlobal) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
	emitUint16(code, ins.GlobalIndex)
}

func DecodeSetGlobal(ip *uint16, code []byte) (ins InstructionSetGlobal) {
	ins.GlobalIndex = decodeUint16(ip, code)
	return ins
}

// GetField

type InstructionGetField struct{}

func (InstructionGetField) Opcode() Opcode {
	return GetField
}

func (ins InstructionGetField) String() string {
	return ins.Opcode().String()
}

func (ins InstructionGetField) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// SetField

type InstructionSetField struct{}

func (InstructionSetField) Opcode() Opcode {
	return SetField
}

func (ins InstructionSetField) String() string {
	return ins.Opcode().String()
}

func (ins InstructionSetField) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// GetIndex

type InstructionGetIndex struct{}

func (InstructionGetIndex) Opcode() Opcode {
	return GetIndex
}

func (ins InstructionGetIndex) String() string {
	return ins.Opcode().String()
}

func (ins InstructionGetIndex) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// SetIndex

type InstructionSetIndex struct{}

func (InstructionSetIndex) Opcode() Opcode {
	return SetIndex
}

func (ins InstructionSetIndex) String() string {
	return ins.Opcode().String()
}

func (ins InstructionSetIndex) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// NewArray

type InstructionNewArray struct {
	TypeIndex  uint16
	Size       uint16
	IsResource bool
}

func (InstructionNewArray) Opcode() Opcode {
	return NewArray
}

func (ins InstructionNewArray) String() string {
	return fmt.Sprintf(
		"%s typeIndex:%d size:%d isResource:%t",
		ins.Opcode(),
		ins.TypeIndex,
		ins.Size,
		ins.IsResource,
	)
}

func (ins InstructionNewArray) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionIntAdd) Opcode() Opcode {
	return IntAdd
}

func (ins InstructionIntAdd) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntAdd) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntSubtract

type InstructionIntSubtract struct{}

func (InstructionIntSubtract) Opcode() Opcode {
	return IntSubtract
}

func (ins InstructionIntSubtract) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntSubtract) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntMultiply

type InstructionIntMultiply struct{}

func (InstructionIntMultiply) Opcode() Opcode {
	return IntMultiply
}

func (ins InstructionIntMultiply) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntMultiply) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntDivide

type InstructionIntDivide struct{}

func (InstructionIntDivide) Opcode() Opcode {
	return IntDivide
}

func (ins InstructionIntDivide) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntDivide) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntMod

type InstructionIntMod struct{}

func (InstructionIntMod) Opcode() Opcode {
	return IntMod
}

func (ins InstructionIntMod) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntMod) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// Equal

type InstructionEqual struct{}

func (ins InstructionEqual) Opcode() Opcode {
	return Equal
}

func (ins InstructionEqual) String() string {
	return ins.Opcode().String()
}

func (ins InstructionEqual) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// NotEqual

type InstructionNotEqual struct{}

func (InstructionNotEqual) Opcode() Opcode {
	return NotEqual
}

func (ins InstructionNotEqual) String() string {
	return ins.Opcode().String()
}

func (ins InstructionNotEqual) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntLess

type InstructionIntLess struct{}

func (InstructionIntLess) Opcode() Opcode {
	return IntLess
}

func (ins InstructionIntLess) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntLess) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntLessOrEqual

type InstructionIntLessOrEqual struct{}

func (ins InstructionIntLessOrEqual) Opcode() Opcode {
	return IntLessOrEqual
}

func (ins InstructionIntLessOrEqual) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntLessOrEqual) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntGreater

type InstructionIntGreater struct{}

func (InstructionIntGreater) Opcode() Opcode {
	return IntGreater
}

func (ins InstructionIntGreater) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntGreater) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// IntGreaterOrEqual

type InstructionIntGreaterOrEqual struct{}

func (ins InstructionIntGreaterOrEqual) Opcode() Opcode {
	return IntGreaterOrEqual
}

func (ins InstructionIntGreaterOrEqual) String() string {
	return ins.Opcode().String()
}

func (ins InstructionIntGreaterOrEqual) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// Unwrap

type InstructionUnwrap struct{}

func (InstructionUnwrap) Opcode() Opcode {
	return Unwrap
}

func (ins InstructionUnwrap) String() string {
	return ins.Opcode().String()
}

func (ins InstructionUnwrap) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// Cast

type InstructionCast struct {
	TypeIndex uint16
	Kind      CastKind
}

func (InstructionCast) Opcode() Opcode {
	return Cast
}

func (ins InstructionCast) String() string {
	return fmt.Sprintf(
		"%s typeIndex:%d kind:%d",
		ins.Opcode(),
		ins.TypeIndex,
		ins.Kind,
	)
}

func (ins InstructionCast) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionDestroy) Opcode() Opcode {
	return Destroy
}

func (ins InstructionDestroy) String() string {
	return ins.Opcode().String()
}

func (ins InstructionDestroy) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// Transfer

type InstructionTransfer struct {
	TypeIndex uint16
}

func (InstructionTransfer) Opcode() Opcode {
	return Transfer
}

func (ins InstructionTransfer) String() string {
	return fmt.Sprintf(
		"%s typeIndex:%d",
		ins.Opcode(),
		ins.TypeIndex,
	)
}

func (ins InstructionTransfer) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionNewRef) Opcode() Opcode {
	return NewRef
}

func (ins InstructionNewRef) String() string {
	return fmt.Sprintf(
		"%s typeIndex:%d",
		ins.Opcode(),
		ins.TypeIndex,
	)
}

func (ins InstructionNewRef) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionPath) Opcode() Opcode {
	return Path
}

func (ins InstructionPath) String() string {
	return fmt.Sprintf(
		"%s domain:%d identifier:%q",
		ins.Opcode(),
		ins.Domain,
		ins.Identifier,
	)
}

func (ins InstructionPath) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionInvoke) Opcode() Opcode {
	return Invoke
}

func (ins InstructionInvoke) String() string {
	return fmt.Sprintf(
		"%s typeArgs:%v",
		ins.Opcode(),
		ins.TypeArgs,
	)
}

func (ins InstructionInvoke) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionNew) Opcode() Opcode {
	return New
}

func (ins InstructionNew) String() string {
	return fmt.Sprintf(
		"%s kind:%d typeIndex:%d",
		ins.Opcode(),
		ins.Kind,
		ins.TypeIndex,
	)
}

func (ins InstructionNew) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

func (InstructionInvokeDynamic) Opcode() Opcode {
	return InvokeDynamic
}

func (ins InstructionInvokeDynamic) String() string {
	return fmt.Sprintf(
		"%s name:%q typeArgs:%v argCount:%d",
		ins.Opcode(),
		ins.Name,
		ins.TypeArgs,
		ins.ArgCount,
	)
}

func (ins InstructionInvokeDynamic) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
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

// Unknown

type InstructionUnknown struct{}

func (InstructionUnknown) Opcode() Opcode {
	return Unknown
}

func (ins InstructionUnknown) String() string {
	return ins.Opcode().String()
}

func (ins InstructionUnknown) Encode(code *[]byte) {
	emitOpcode(code, ins.Opcode())
}

// DecodeInstruction

func DecodeInstruction(ip *uint16, code []byte) Instruction {

	switch Opcode(decodeByte(ip, code)) {

	case Return:
		return InstructionReturn{}
	case ReturnValue:
		return InstructionReturnValue{}
	case Jump:
		return DecodeJump(ip, code)
	case JumpIfFalse:
		return DecodeJumpIfFalse(ip, code)
	case IntAdd:
		return InstructionIntAdd{}
	case IntSubtract:
		return InstructionIntSubtract{}
	case IntMultiply:
		return InstructionIntMultiply{}
	case IntDivide:
		return InstructionIntDivide{}
	case IntMod:
		return InstructionIntMod{}
	case IntLess:
		return InstructionIntLess{}
	case IntGreater:
		return InstructionIntGreater{}
	case IntLessOrEqual:
		return InstructionIntLessOrEqual{}
	case IntGreaterOrEqual:
		return InstructionIntGreaterOrEqual{}
	case Equal:
		return InstructionEqual{}
	case NotEqual:
		return InstructionNotEqual{}
	case Unwrap:
		return InstructionUnwrap{}
	case Destroy:
		return InstructionDestroy{}
	case Transfer:
		return DecodeTransfer(ip, code)
	case Cast:
		return DecodeCast(ip, code)
	case True:
		return InstructionTrue{}
	case False:
		return InstructionFalse{}
	case New:
		return DecodeNew(ip, code)
	case Path:
		return DecodePath(ip, code)
	case Nil:
		return InstructionNil{}
	case NewArray:
		return DecodeNewArray(ip, code)
	case NewDictionary:
		// TODO:
		return nil
	case NewRef:
		return DecodeNewRef(ip, code)
	case GetConstant:
		return DecodeGetConstant(ip, code)
	case GetLocal:
		return DecodeGetLocal(ip, code)
	case SetLocal:
		return DecodeSetLocal(ip, code)
	case GetGlobal:
		return DecodeGetGlobal(ip, code)
	case SetGlobal:
		return DecodeSetGlobal(ip, code)
	case GetField:
		return InstructionGetField{}
	case SetField:
		return InstructionSetField{}
	case SetIndex:
		return InstructionSetIndex{}
	case GetIndex:
		return InstructionGetIndex{}
	case Invoke:
		return DecodeInvoke(ip, code)
	case InvokeDynamic:
		return DecodeInvokeDynamic(ip, code)
	case Drop:
		return InstructionDrop{}
	case Dup:
		return InstructionDup{}
	case Unknown:
		return InstructionUnknown{}
	}

	panic(errors.NewUnreachableError())
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
