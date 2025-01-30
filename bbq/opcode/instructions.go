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
// Sets the value of the local at the given index to the top value on the stack.
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
// Sets the value of the global at the given index to the top value on the stack.
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
// Pushes the value of the field at the given index onto the stack.
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
// Sets the value of the field at the given index to the top value on the stack.
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
// Pushes the value at the given index onto the stack.
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
// Sets the value at the given index to the top value on the stack.
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
// Pushes the path with the given domain and identifier onto the stack.
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
// Creates a new instance of the given type.
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
// Creates a new array with the given type and size.
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
// Creates a new dictionary with the given type and size.
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
// Creates a new reference with the given type.
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
// Invokes the function with the given type arguments.
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
// Invokes the dynamic function with the given name, type arguments, and argument count.
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
// Duplicates the top value on the stack.
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
// Removes the top value from the stack.
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
// Destroys the top value on the stack.
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
// Unwraps the top value on the stack.
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
// Transfers the top value on the stack.
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

// InstructionCast
//
// Casts the top value on the stack to the given type.
type InstructionCast struct {
	TypeIndex uint16
	Kind      CastKind
}

var _ Instruction = InstructionCast{}

func (InstructionCast) Opcode() Opcode {
	return Cast
}

func (i InstructionCast) String() string {
	var sb strings.Builder
	sb.WriteString(i.Opcode().String())
	printfArgument(&sb, "typeIndex", i.TypeIndex)
	printfArgument(&sb, "kind", i.Kind)
	return sb.String()
}

func (i InstructionCast) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
	emitUint16(code, i.TypeIndex)
	emitCastKind(code, i.Kind)
}

func DecodeCast(ip *uint16, code []byte) (i InstructionCast) {
	i.TypeIndex = decodeUint16(ip, code)
	i.Kind = decodeCastKind(ip, code)
	return i
}

// InstructionJump
//
// Jumps to the given instruction.
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
// Jumps to the given instruction, if the top value on the stack is `false`.
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
// Returns from the current function, with the top value on the stack.
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
// Compares the top two values on the stack for equality.
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
// Compares the top two values on the stack for inequality.
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
// Negates the boolean value at the top of the stack.
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

// InstructionIntAdd
//
// Adds the top two values on the stack.
type InstructionIntAdd struct {
}

var _ Instruction = InstructionIntAdd{}

func (InstructionIntAdd) Opcode() Opcode {
	return IntAdd
}

func (i InstructionIntAdd) String() string {
	return i.Opcode().String()
}

func (i InstructionIntAdd) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntSubtract
//
// Subtracts the top two values on the stack.
type InstructionIntSubtract struct {
}

var _ Instruction = InstructionIntSubtract{}

func (InstructionIntSubtract) Opcode() Opcode {
	return IntSubtract
}

func (i InstructionIntSubtract) String() string {
	return i.Opcode().String()
}

func (i InstructionIntSubtract) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntMultiply
//
// Multiplies the top two values on the stack.
type InstructionIntMultiply struct {
}

var _ Instruction = InstructionIntMultiply{}

func (InstructionIntMultiply) Opcode() Opcode {
	return IntMultiply
}

func (i InstructionIntMultiply) String() string {
	return i.Opcode().String()
}

func (i InstructionIntMultiply) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntDivide
//
// Divides the top two values on the stack.
type InstructionIntDivide struct {
}

var _ Instruction = InstructionIntDivide{}

func (InstructionIntDivide) Opcode() Opcode {
	return IntDivide
}

func (i InstructionIntDivide) String() string {
	return i.Opcode().String()
}

func (i InstructionIntDivide) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntMod
//
// Calculates the modulo of the top two values on the stack.
type InstructionIntMod struct {
}

var _ Instruction = InstructionIntMod{}

func (InstructionIntMod) Opcode() Opcode {
	return IntMod
}

func (i InstructionIntMod) String() string {
	return i.Opcode().String()
}

func (i InstructionIntMod) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntLess
//
// Compares the top two values on the stack for less than.
type InstructionIntLess struct {
}

var _ Instruction = InstructionIntLess{}

func (InstructionIntLess) Opcode() Opcode {
	return IntLess
}

func (i InstructionIntLess) String() string {
	return i.Opcode().String()
}

func (i InstructionIntLess) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntLessOrEqual
//
// Compares the top two values on the stack for less than or equal.
type InstructionIntLessOrEqual struct {
}

var _ Instruction = InstructionIntLessOrEqual{}

func (InstructionIntLessOrEqual) Opcode() Opcode {
	return IntLessOrEqual
}

func (i InstructionIntLessOrEqual) String() string {
	return i.Opcode().String()
}

func (i InstructionIntLessOrEqual) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntGreater
//
// Compares the top two values on the stack for greater than.
type InstructionIntGreater struct {
}

var _ Instruction = InstructionIntGreater{}

func (InstructionIntGreater) Opcode() Opcode {
	return IntGreater
}

func (i InstructionIntGreater) String() string {
	return i.Opcode().String()
}

func (i InstructionIntGreater) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
}

// InstructionIntGreaterOrEqual
//
// Compares the top two values on the stack for greater than or equal.
type InstructionIntGreaterOrEqual struct {
}

var _ Instruction = InstructionIntGreaterOrEqual{}

func (InstructionIntGreaterOrEqual) Opcode() Opcode {
	return IntGreaterOrEqual
}

func (i InstructionIntGreaterOrEqual) String() string {
	return i.Opcode().String()
}

func (i InstructionIntGreaterOrEqual) Encode(code *[]byte) {
	emitOpcode(code, i.Opcode())
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
	case Cast:
		return DecodeCast(ip, code)
	case Jump:
		return DecodeJump(ip, code)
	case JumpIfFalse:
		return DecodeJumpIfFalse(ip, code)
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
	case IntLessOrEqual:
		return InstructionIntLessOrEqual{}
	case IntGreater:
		return InstructionIntGreater{}
	case IntGreaterOrEqual:
		return InstructionIntGreaterOrEqual{}
	}

	panic(errors.NewUnreachableError())
}
