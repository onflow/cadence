// Code generated by gen/main.go from instructions.yml. DO NOT EDIT.

package opcode

import (
	"strings"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

// InstructionUnknown
//
// An unknown instruction.
type InstructionUnknown struct {
}

var _ Instruction = InstructionUnknown{}

func (InstructionUnknown) Opcode() Opcode {
	return Unknown
}

func (i InstructionUnknown) String() string {
	return i.Opcode().String()
}

func (i InstructionUnknown) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionGetLocal
//
// Pushes the value of the local at the given index onto the stack.
type InstructionGetLocal struct {
	LocalIndex uint16
}

var _ Instruction = InstructionGetLocal{}

func (InstructionGetLocal) Opcode() Opcode {
	return GetLocal
}

func (i InstructionGetLocal) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "localIndex", i.LocalIndex)
	return sb.String()
}

func (i InstructionGetLocal) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.LocalIndex)
}

func DecodeGetLocal(ip *uint16, code []byte) (i InstructionGetLocal) {
	i.LocalIndex = decodeUint16(ip, code)
	return i
}

// InstructionSetLocal
//
// Pops a value off the stack and then sets the local at the given index to that value.
type InstructionSetLocal struct {
	LocalIndex uint16
}

var _ Instruction = InstructionSetLocal{}

func (InstructionSetLocal) Opcode() Opcode {
	return SetLocal
}

func (i InstructionSetLocal) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "localIndex", i.LocalIndex)
	return sb.String()
}

func (i InstructionSetLocal) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.LocalIndex)
}

func DecodeSetLocal(ip *uint16, code []byte) (i InstructionSetLocal) {
	i.LocalIndex = decodeUint16(ip, code)
	return i
}

// InstructionGetGlobal
//
// Pushes the value of the global at the given index onto the stack.
type InstructionGetGlobal struct {
	GlobalIndex uint16
}

var _ Instruction = InstructionGetGlobal{}

func (InstructionGetGlobal) Opcode() Opcode {
	return GetGlobal
}

func (i InstructionGetGlobal) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "globalIndex", i.GlobalIndex)
	return sb.String()
}

func (i InstructionGetGlobal) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.GlobalIndex)
}

func DecodeGetGlobal(ip *uint16, code []byte) (i InstructionGetGlobal) {
	i.GlobalIndex = decodeUint16(ip, code)
	return i
}

// InstructionSetGlobal
//
// Pops a value off the stack and then sets the global at the given index to that value.
type InstructionSetGlobal struct {
	GlobalIndex uint16
}

var _ Instruction = InstructionSetGlobal{}

func (InstructionSetGlobal) Opcode() Opcode {
	return SetGlobal
}

func (i InstructionSetGlobal) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "globalIndex", i.GlobalIndex)
	return sb.String()
}

func (i InstructionSetGlobal) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.GlobalIndex)
}

func DecodeSetGlobal(ip *uint16, code []byte) (i InstructionSetGlobal) {
	i.GlobalIndex = decodeUint16(ip, code)
	return i
}

// InstructionGetField
//
// Pops a value off the stack, the target, and then pushes the value of the field at the given index onto the stack.
type InstructionGetField struct {
	FieldNameIndex uint16
}

var _ Instruction = InstructionGetField{}

func (InstructionGetField) Opcode() Opcode {
	return GetField
}

func (i InstructionGetField) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "fieldNameIndex", i.FieldNameIndex)
	return sb.String()
}

func (i InstructionGetField) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.FieldNameIndex)
}

func DecodeGetField(ip *uint16, code []byte) (i InstructionGetField) {
	i.FieldNameIndex = decodeUint16(ip, code)
	return i
}

// InstructionSetField
//
// Pops two values off the stack, the target and the value, and then sets the field at the given index of the target to the value.
type InstructionSetField struct {
	FieldNameIndex uint16
}

var _ Instruction = InstructionSetField{}

func (InstructionSetField) Opcode() Opcode {
	return SetField
}

func (i InstructionSetField) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "fieldNameIndex", i.FieldNameIndex)
	return sb.String()
}

func (i InstructionSetField) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.FieldNameIndex)
}

func DecodeSetField(ip *uint16, code []byte) (i InstructionSetField) {
	i.FieldNameIndex = decodeUint16(ip, code)
	return i
}

// InstructionGetIndex
//
// Pops two values off the stack, the array and the index, and then pushes the value at the given index of the array onto the stack.
type InstructionGetIndex struct {
}

var _ Instruction = InstructionGetIndex{}

func (InstructionGetIndex) Opcode() Opcode {
	return GetIndex
}

func (i InstructionGetIndex) String() string {
	return i.Opcode().String()
}

func (i InstructionGetIndex) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionSetIndex
//
// Pops three values off the stack, the array, the index, and the value, and then sets the value at the given index of the array to the value.
type InstructionSetIndex struct {
}

var _ Instruction = InstructionSetIndex{}

func (InstructionSetIndex) Opcode() Opcode {
	return SetIndex
}

func (i InstructionSetIndex) String() string {
	return i.Opcode().String()
}

func (i InstructionSetIndex) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionTrue
//
// Pushes the boolean value `true` onto the stack.
type InstructionTrue struct {
}

var _ Instruction = InstructionTrue{}

func (InstructionTrue) Opcode() Opcode {
	return True
}

func (i InstructionTrue) String() string {
	return i.Opcode().String()
}

func (i InstructionTrue) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionFalse
//
// Pushes the boolean value `false` onto the stack.
type InstructionFalse struct {
}

var _ Instruction = InstructionFalse{}

func (InstructionFalse) Opcode() Opcode {
	return False
}

func (i InstructionFalse) String() string {
	return i.Opcode().String()
}

func (i InstructionFalse) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionNil
//
// Pushes the value `nil` onto the stack.
type InstructionNil struct {
}

var _ Instruction = InstructionNil{}

func (InstructionNil) Opcode() Opcode {
	return Nil
}

func (i InstructionNil) String() string {
	return i.Opcode().String()
}

func (i InstructionNil) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionPath
//
// Creates a new path with the given domain and identifier and then pushes it onto the stack.
type InstructionPath struct {
	Domain          common.PathDomain
	IdentifierIndex uint16
}

var _ Instruction = InstructionPath{}

func (InstructionPath) Opcode() Opcode {
	return Path
}

func (i InstructionPath) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "domain", i.Domain)
	printfArgument(&sb, "identifierIndex", i.IdentifierIndex)
	return sb.String()
}

func (i InstructionPath) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitPathDomain(code, i.Domain)
	emitUint16(code, i.IdentifierIndex)
}

func DecodePath(ip *uint16, code []byte) (i InstructionPath) {
	i.Domain = decodePathDomain(ip, code)
	i.IdentifierIndex = decodeUint16(ip, code)
	return i
}

// InstructionNew
//
// Creates a new instance of the given kind and type and then pushes it onto the stack.
type InstructionNew struct {
	Kind      common.CompositeKind
	TypeIndex uint16
}

var _ Instruction = InstructionNew{}

func (InstructionNew) Opcode() Opcode {
	return New
}

func (i InstructionNew) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "kind", i.Kind)
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	return sb.String()
}

func (i InstructionNew) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitCompositeKind(code, i.Kind)
	emitUint16(code, i.TypeIndex)
}

func DecodeNew(ip *uint16, code []byte) (i InstructionNew) {
	i.Kind = decodeCompositeKind(ip, code)
	i.TypeIndex = decodeUint16(ip, code)
	return i
}

// InstructionNewArray
//
// Pops the given number of elements off the stack, creates a new array with the given type, size, and elements, and then pushes it onto the stack.
type InstructionNewArray struct {
	TypeIndex  uint16
	Size       uint16
	IsResource bool
}

var _ Instruction = InstructionNewArray{}

func (InstructionNewArray) Opcode() Opcode {
	return NewArray
}

func (i InstructionNewArray) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	printfArgument(&sb, "size", i.Size)
	printfArgument(&sb, "isResource", i.IsResource)
	return sb.String()
}

func (i InstructionNewArray) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
	emitUint16(code, i.Size)
	emitBool(code, i.IsResource)
}

func DecodeNewArray(ip *uint16, code []byte) (i InstructionNewArray) {
	i.TypeIndex = decodeUint16(ip, code)
	i.Size = decodeUint16(ip, code)
	i.IsResource = decodeBool(ip, code)
	return i
}

// InstructionNewDictionary
//
// Pops the given number of entries off the stack (twice the number of the given size), creates a new dictionary with the given type, size, and entries, and then pushes it onto the stack.
type InstructionNewDictionary struct {
	TypeIndex  uint16
	Size       uint16
	IsResource bool
}

var _ Instruction = InstructionNewDictionary{}

func (InstructionNewDictionary) Opcode() Opcode {
	return NewDictionary
}

func (i InstructionNewDictionary) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	printfArgument(&sb, "size", i.Size)
	printfArgument(&sb, "isResource", i.IsResource)
	return sb.String()
}

func (i InstructionNewDictionary) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
	emitUint16(code, i.Size)
	emitBool(code, i.IsResource)
}

func DecodeNewDictionary(ip *uint16, code []byte) (i InstructionNewDictionary) {
	i.TypeIndex = decodeUint16(ip, code)
	i.Size = decodeUint16(ip, code)
	i.IsResource = decodeBool(ip, code)
	return i
}

// InstructionNewRef
//
// Pops a value off the stack, creates a new reference with the given type, and then pushes it onto the stack.
type InstructionNewRef struct {
	TypeIndex uint16
}

var _ Instruction = InstructionNewRef{}

func (InstructionNewRef) Opcode() Opcode {
	return NewRef
}

func (i InstructionNewRef) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	return sb.String()
}

func (i InstructionNewRef) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
}

func DecodeNewRef(ip *uint16, code []byte) (i InstructionNewRef) {
	i.TypeIndex = decodeUint16(ip, code)
	return i
}

// InstructionGetConstant
//
// Pushes the constant at the given index onto the stack.
type InstructionGetConstant struct {
	ConstantIndex uint16
}

var _ Instruction = InstructionGetConstant{}

func (InstructionGetConstant) Opcode() Opcode {
	return GetConstant
}

func (i InstructionGetConstant) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "constantIndex", i.ConstantIndex)
	return sb.String()
}

func (i InstructionGetConstant) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.ConstantIndex)
}

func DecodeGetConstant(ip *uint16, code []byte) (i InstructionGetConstant) {
	i.ConstantIndex = decodeUint16(ip, code)
	return i
}

// InstructionInvoke
//
// Pops the function and arguments off the stack, invokes the function with the arguments, and then pushes the result back on to the stack.
type InstructionInvoke struct {
	TypeArgs []uint16
}

var _ Instruction = InstructionInvoke{}

func (InstructionInvoke) Opcode() Opcode {
	return Invoke
}

func (i InstructionInvoke) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfUInt16ArrayArgument(&sb, "typeArgs", i.TypeArgs)
	return sb.String()
}

func (i InstructionInvoke) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16Array(code, i.TypeArgs)
}

func DecodeInvoke(ip *uint16, code []byte) (i InstructionInvoke) {
	i.TypeArgs = decodeUint16Array(ip, code)
	return i
}

// InstructionInvokeDynamic
//
// Pops the arguments off the stack, invokes the function with the given name and argument count, and then pushes the result back on to the stack.
type InstructionInvokeDynamic struct {
	NameIndex uint16
	TypeArgs  []uint16
	ArgCount  uint16
}

var _ Instruction = InstructionInvokeDynamic{}

func (InstructionInvokeDynamic) Opcode() Opcode {
	return InvokeDynamic
}

func (i InstructionInvokeDynamic) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "nameIndex", i.NameIndex)
	printfUInt16ArrayArgument(&sb, "typeArgs", i.TypeArgs)
	printfArgument(&sb, "argCount", i.ArgCount)
	return sb.String()
}

func (i InstructionInvokeDynamic) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.NameIndex)
	emitUint16Array(code, i.TypeArgs)
	emitUint16(code, i.ArgCount)
}

func DecodeInvokeDynamic(ip *uint16, code []byte) (i InstructionInvokeDynamic) {
	i.NameIndex = decodeUint16(ip, code)
	i.TypeArgs = decodeUint16Array(ip, code)
	i.ArgCount = decodeUint16(ip, code)
	return i
}

// InstructionDup
//
// Pops a value off the stack, duplicates it, and then pushes the original and the copy back on to the stack.
type InstructionDup struct {
}

var _ Instruction = InstructionDup{}

func (InstructionDup) Opcode() Opcode {
	return Dup
}

func (i InstructionDup) String() string {
	return i.Opcode().String()
}

func (i InstructionDup) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionDrop
//
// Pops a value off the stack and discards it.
type InstructionDrop struct {
}

var _ Instruction = InstructionDrop{}

func (InstructionDrop) Opcode() Opcode {
	return Drop
}

func (i InstructionDrop) String() string {
	return i.Opcode().String()
}

func (i InstructionDrop) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionDestroy
//
// Pops a resource off the stack and then destroys it.
type InstructionDestroy struct {
}

var _ Instruction = InstructionDestroy{}

func (InstructionDestroy) Opcode() Opcode {
	return Destroy
}

func (i InstructionDestroy) String() string {
	return i.Opcode().String()
}

func (i InstructionDestroy) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionUnwrap
//
// Pops an optional value off the stack, unwraps it, and then pushes the value back on to the stack.
type InstructionUnwrap struct {
}

var _ Instruction = InstructionUnwrap{}

func (InstructionUnwrap) Opcode() Opcode {
	return Unwrap
}

func (i InstructionUnwrap) String() string {
	return i.Opcode().String()
}

func (i InstructionUnwrap) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionTransfer
//
// Pops a value off the stack, transfers it to the given type, and then pushes it back on to the stack.
type InstructionTransfer struct {
	TypeIndex uint16
}

var _ Instruction = InstructionTransfer{}

func (InstructionTransfer) Opcode() Opcode {
	return Transfer
}

func (i InstructionTransfer) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	return sb.String()
}

func (i InstructionTransfer) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
}

func DecodeTransfer(ip *uint16, code []byte) (i InstructionTransfer) {
	i.TypeIndex = decodeUint16(ip, code)
	return i
}

// InstructionSimpleCast
//
// Pops a value off the stack, casts it to the given type, and then pushes it back on to the stack.
type InstructionSimpleCast struct {
	TypeIndex uint16
}

var _ Instruction = InstructionSimpleCast{}

func (InstructionSimpleCast) Opcode() Opcode {
	return SimpleCast
}

func (i InstructionSimpleCast) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	return sb.String()
}

func (i InstructionSimpleCast) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
}

func DecodeSimpleCast(ip *uint16, code []byte) (i InstructionSimpleCast) {
	i.TypeIndex = decodeUint16(ip, code)
	return i
}

// InstructionFailableCast
//
// Pops a value off the stack and casts it to the given type. If the value is a subtype of the given type, then casted value is pushed back on to the stack. If the value is not a subtype of the given type, then a `nil` is pushed to the stack instead.
type InstructionFailableCast struct {
	TypeIndex uint16
}

var _ Instruction = InstructionFailableCast{}

func (InstructionFailableCast) Opcode() Opcode {
	return FailableCast
}

func (i InstructionFailableCast) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	return sb.String()
}

func (i InstructionFailableCast) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
}

func DecodeFailableCast(ip *uint16, code []byte) (i InstructionFailableCast) {
	i.TypeIndex = decodeUint16(ip, code)
	return i
}

// InstructionForceCast
//
// Pops a value off the stack, force-casts it to the given type, and then pushes it back on to the stack. Panics if the value is not a subtype of the given type.
type InstructionForceCast struct {
	TypeIndex uint16
}

var _ Instruction = InstructionForceCast{}

func (InstructionForceCast) Opcode() Opcode {
	return ForceCast
}

func (i InstructionForceCast) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	return sb.String()
}

func (i InstructionForceCast) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
}

func DecodeForceCast(ip *uint16, code []byte) (i InstructionForceCast) {
	i.TypeIndex = decodeUint16(ip, code)
	return i
}

// InstructionJump
//
// Unconditionally jumps to the given instruction.
type InstructionJump struct {
	Target uint16
}

var _ Instruction = InstructionJump{}

func (InstructionJump) Opcode() Opcode {
	return Jump
}

func (i InstructionJump) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "target", i.Target)
	return sb.String()
}

func (i InstructionJump) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.Target)
}

func DecodeJump(ip *uint16, code []byte) (i InstructionJump) {
	i.Target = decodeUint16(ip, code)
	return i
}

// InstructionJumpIfFalse
//
// Pops a value off the stack. If it is `false`, jumps to the target instruction.
type InstructionJumpIfFalse struct {
	Target uint16
}

var _ Instruction = InstructionJumpIfFalse{}

func (InstructionJumpIfFalse) Opcode() Opcode {
	return JumpIfFalse
}

func (i InstructionJumpIfFalse) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "target", i.Target)
	return sb.String()
}

func (i InstructionJumpIfFalse) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.Target)
}

func DecodeJumpIfFalse(ip *uint16, code []byte) (i InstructionJumpIfFalse) {
	i.Target = decodeUint16(ip, code)
	return i
}

// InstructionJumpIfNil
//
// Pops a value off the stack. If it is `nil`, jumps to the target instruction.
type InstructionJumpIfNil struct {
	Target uint16
}

var _ Instruction = InstructionJumpIfNil{}

func (InstructionJumpIfNil) Opcode() Opcode {
	return JumpIfNil
}

func (i InstructionJumpIfNil) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "target", i.Target)
	return sb.String()
}

func (i InstructionJumpIfNil) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.Target)
}

func DecodeJumpIfNil(ip *uint16, code []byte) (i InstructionJumpIfNil) {
	i.Target = decodeUint16(ip, code)
	return i
}

// InstructionReturn
//
// Returns from the current function, without a value.
type InstructionReturn struct {
}

var _ Instruction = InstructionReturn{}

func (InstructionReturn) Opcode() Opcode {
	return Return
}

func (i InstructionReturn) String() string {
	return i.Opcode().String()
}

func (i InstructionReturn) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionReturnValue
//
// Pops a value off the stack and then returns from the current function with that value.
type InstructionReturnValue struct {
}

var _ Instruction = InstructionReturnValue{}

func (InstructionReturnValue) Opcode() Opcode {
	return ReturnValue
}

func (i InstructionReturnValue) String() string {
	return i.Opcode().String()
}

func (i InstructionReturnValue) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionEqual
//
// Pops two values off the stack, checks if the first value is equal to the second, and then pushes the result back on to the stack.
type InstructionEqual struct {
}

var _ Instruction = InstructionEqual{}

func (InstructionEqual) Opcode() Opcode {
	return Equal
}

func (i InstructionEqual) String() string {
	return i.Opcode().String()
}

func (i InstructionEqual) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionNotEqual
//
// Pops two values off the stack, checks if the first value is not equal to the second, and then pushes the result back on to the stack.
type InstructionNotEqual struct {
}

var _ Instruction = InstructionNotEqual{}

func (InstructionNotEqual) Opcode() Opcode {
	return NotEqual
}

func (i InstructionNotEqual) String() string {
	return i.Opcode().String()
}

func (i InstructionNotEqual) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionNot
//
// Pops a boolean value off the stack, negates it, and then pushes the result back on to the stack.
type InstructionNot struct {
}

var _ Instruction = InstructionNot{}

func (InstructionNot) Opcode() Opcode {
	return Not
}

func (i InstructionNot) String() string {
	return i.Opcode().String()
}

func (i InstructionNot) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionAdd
//
// Pops two number values off the stack, adds them together, and then pushes the result back on to the stack.
type InstructionAdd struct {
}

var _ Instruction = InstructionAdd{}

func (InstructionAdd) Opcode() Opcode {
	return Add
}

func (i InstructionAdd) String() string {
	return i.Opcode().String()
}

func (i InstructionAdd) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionSubtract
//
// Pops two number values off the stack, subtracts the second from the first, and then pushes the result back on to the stack.
type InstructionSubtract struct {
}

var _ Instruction = InstructionSubtract{}

func (InstructionSubtract) Opcode() Opcode {
	return Subtract
}

func (i InstructionSubtract) String() string {
	return i.Opcode().String()
}

func (i InstructionSubtract) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionMultiply
//
// Pops two number values off the stack, multiplies them together, and then pushes the result back on to the stack.
type InstructionMultiply struct {
}

var _ Instruction = InstructionMultiply{}

func (InstructionMultiply) Opcode() Opcode {
	return Multiply
}

func (i InstructionMultiply) String() string {
	return i.Opcode().String()
}

func (i InstructionMultiply) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionDivide
//
// Pops two number values off the stack, divides the first by the second, and then pushes the result back on to the stack.
type InstructionDivide struct {
}

var _ Instruction = InstructionDivide{}

func (InstructionDivide) Opcode() Opcode {
	return Divide
}

func (i InstructionDivide) String() string {
	return i.Opcode().String()
}

func (i InstructionDivide) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionMod
//
// Pops two number values off the stack, calculates the modulus of the first by the second, and then pushes the result back on to the stack.
type InstructionMod struct {
}

var _ Instruction = InstructionMod{}

func (InstructionMod) Opcode() Opcode {
	return Mod
}

func (i InstructionMod) String() string {
	return i.Opcode().String()
}

func (i InstructionMod) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionLess
//
// Pops two values off the stack, checks if the first value is less than the second, and then pushes the result back on to the stack.
type InstructionLess struct {
}

var _ Instruction = InstructionLess{}

func (InstructionLess) Opcode() Opcode {
	return Less
}

func (i InstructionLess) String() string {
	return i.Opcode().String()
}

func (i InstructionLess) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionLessOrEqual
//
// Pops two values off the stack, checks if the first value is less than or equal to the second, and then pushes the result back on to the stack.
type InstructionLessOrEqual struct {
}

var _ Instruction = InstructionLessOrEqual{}

func (InstructionLessOrEqual) Opcode() Opcode {
	return LessOrEqual
}

func (i InstructionLessOrEqual) String() string {
	return i.Opcode().String()
}

func (i InstructionLessOrEqual) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionGreater
//
// Pops two values off the stack, checks if the first value is greater than the second, and then pushes the result back on to the stack.
type InstructionGreater struct {
}

var _ Instruction = InstructionGreater{}

func (InstructionGreater) Opcode() Opcode {
	return Greater
}

func (i InstructionGreater) String() string {
	return i.Opcode().String()
}

func (i InstructionGreater) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionGreaterOrEqual
//
// Pops two values off the stack, checks if the first value is greater than or equal to the second, and then pushes the result back on to the stack.
type InstructionGreaterOrEqual struct {
}

var _ Instruction = InstructionGreaterOrEqual{}

func (InstructionGreaterOrEqual) Opcode() Opcode {
	return GreaterOrEqual
}

func (i InstructionGreaterOrEqual) String() string {
	return i.Opcode().String()
}

func (i InstructionGreaterOrEqual) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIterator
//
// Pops a value from stack, get an iterator to it, and push the iterator back onto the stack.
type InstructionIterator struct {
}

var _ Instruction = InstructionIterator{}

func (InstructionIterator) Opcode() Opcode {
	return Iterator
}

func (i InstructionIterator) String() string {
	return i.Opcode().String()
}

func (i InstructionIterator) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIteratorHasNext
//
// Pops a value-iterator from stack, calls `hasNext()` method on it, and push the result back onto the stack.
type InstructionIteratorHasNext struct {
}

var _ Instruction = InstructionIteratorHasNext{}

func (InstructionIteratorHasNext) Opcode() Opcode {
	return IteratorHasNext
}

func (i InstructionIteratorHasNext) String() string {
	return i.Opcode().String()
}

func (i InstructionIteratorHasNext) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIteratorNext
//
// Pops a value-iterator from stack, calls `next()` method on it, and push the result back onto the stack.
type InstructionIteratorNext struct {
}

var _ Instruction = InstructionIteratorNext{}

func (InstructionIteratorNext) Opcode() Opcode {
	return IteratorNext
}

func (i InstructionIteratorNext) String() string {
	return i.Opcode().String()
}

func (i InstructionIteratorNext) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionEmitEvent
//
// Pops an event off the stack and then emits it.
type InstructionEmitEvent struct {
	TypeIndex uint16
}

var _ Instruction = InstructionEmitEvent{}

func (InstructionEmitEvent) Opcode() Opcode {
	return EmitEvent
}

func (i InstructionEmitEvent) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	return sb.String()
}

func (i InstructionEmitEvent) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
}

func DecodeEmitEvent(ip *uint16, code []byte) (i InstructionEmitEvent) {
	i.TypeIndex = decodeUint16(ip, code)
	return i
}

func DecodeInstruction(ip *uint16, code []byte) Instruction {
	switch Opcode(decodeByte(ip, code)) {
	case Unknown:
		return InstructionUnknown{}
	case GetLocal:
		return DecodeGetLocal(ip, code)
	case SetLocal:
		return DecodeSetLocal(ip, code)
	case GetGlobal:
		return DecodeGetGlobal(ip, code)
	case SetGlobal:
		return DecodeSetGlobal(ip, code)
	case GetField:
		return DecodeGetField(ip, code)
	case SetField:
		return DecodeSetField(ip, code)
	case GetIndex:
		return InstructionGetIndex{}
	case SetIndex:
		return InstructionSetIndex{}
	case True:
		return InstructionTrue{}
	case False:
		return InstructionFalse{}
	case Nil:
		return InstructionNil{}
	case Path:
		return DecodePath(ip, code)
	case New:
		return DecodeNew(ip, code)
	case NewArray:
		return DecodeNewArray(ip, code)
	case NewDictionary:
		return DecodeNewDictionary(ip, code)
	case NewRef:
		return DecodeNewRef(ip, code)
	case GetConstant:
		return DecodeGetConstant(ip, code)
	case Invoke:
		return DecodeInvoke(ip, code)
	case InvokeDynamic:
		return DecodeInvokeDynamic(ip, code)
	case Dup:
		return InstructionDup{}
	case Drop:
		return InstructionDrop{}
	case Destroy:
		return InstructionDestroy{}
	case Unwrap:
		return InstructionUnwrap{}
	case Transfer:
		return DecodeTransfer(ip, code)
	case SimpleCast:
		return DecodeSimpleCast(ip, code)
	case FailableCast:
		return DecodeFailableCast(ip, code)
	case ForceCast:
		return DecodeForceCast(ip, code)
	case Jump:
		return DecodeJump(ip, code)
	case JumpIfFalse:
		return DecodeJumpIfFalse(ip, code)
	case JumpIfNil:
		return DecodeJumpIfNil(ip, code)
	case Return:
		return InstructionReturn{}
	case ReturnValue:
		return InstructionReturnValue{}
	case Equal:
		return InstructionEqual{}
	case NotEqual:
		return InstructionNotEqual{}
	case Not:
		return InstructionNot{}
	case Add:
		return InstructionAdd{}
	case Subtract:
		return InstructionSubtract{}
	case Multiply:
		return InstructionMultiply{}
	case Divide:
		return InstructionDivide{}
	case Mod:
		return InstructionMod{}
	case Less:
		return InstructionLess{}
	case LessOrEqual:
		return InstructionLessOrEqual{}
	case Greater:
		return InstructionGreater{}
	case GreaterOrEqual:
		return InstructionGreaterOrEqual{}
	case Iterator:
		return InstructionIterator{}
	case IteratorHasNext:
		return InstructionIteratorHasNext{}
	case IteratorNext:
		return InstructionIteratorNext{}
	case EmitEvent:
		return DecodeEmitEvent(ip, code)
	}

	panic(errors.NewUnreachableError())
}
