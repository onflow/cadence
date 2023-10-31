package interpreter

import (
	"encoding/binary"
	"unsafe"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// Word8Value

type Word8Value uint8

var _ Value = Word8Value(0)
var _ atree.Storable = Word8Value(0)
var _ NumberValue = Word8Value(0)
var _ IntegerValue = Word8Value(0)
var _ EquatableValue = Word8Value(0)
var _ ComparableValue = Word8Value(0)
var _ HashableValue = Word8Value(0)
var _ MemberAccessibleValue = Word8Value(0)

const word8Size = int(unsafe.Sizeof(Word8Value(0)))

var word8MemoryUsage = common.NewNumberMemoryUsage(word8Size)

func NewWord8Value(gauge common.MemoryGauge, valueGetter func() uint8) Word8Value {
	common.UseMemory(gauge, word8MemoryUsage)

	return NewUnmeteredWord8Value(valueGetter())
}

func NewUnmeteredWord8Value(value uint8) Word8Value {
	return Word8Value(value)
}

func (Word8Value) isValue() {}

func (v Word8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord8Value(interpreter, v)
}

func (Word8Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word8Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord8)
}

func (Word8Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word8Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word8Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word8Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Word8Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v + o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v - o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v % o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v * o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v / o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Word8Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Word8Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Word8Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Word8Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord8, ok := other.(Word8Value)
	if !ok {
		return false
	}
	return v == otherWord8
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord8 (1 byte)
// - uint8 value (1 byte)
func (v Word8Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertWord8(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word8Value {
	return NewWord8Value(
		memoryGauge,
		func() uint8 {
			return ConvertWord[uint8](memoryGauge, value, locationRange)
		},
	)
}

func (v Word8Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v | o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v ^ o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v & o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v << o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v >> o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word8Type, locationRange)
}

func (Word8Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word8Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Word8Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word8Value) IsStorable() bool {
	return true
}

func (v Word8Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word8Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word8Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word8Value) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v Word8Value) Clone(_ *Interpreter) Value {
	return v
}

func (Word8Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word8Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v Word8Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word8Value) ChildStorables() []atree.Storable {
	return nil
}

// Word16Value

type Word16Value uint16

var _ Value = Word16Value(0)
var _ atree.Storable = Word16Value(0)
var _ NumberValue = Word16Value(0)
var _ IntegerValue = Word16Value(0)
var _ EquatableValue = Word16Value(0)
var _ ComparableValue = Word16Value(0)
var _ HashableValue = Word16Value(0)
var _ MemberAccessibleValue = Word16Value(0)

const word16Size = int(unsafe.Sizeof(Word16Value(0)))

var word16MemoryUsage = common.NewNumberMemoryUsage(word16Size)

func NewWord16Value(gauge common.MemoryGauge, valueGetter func() uint16) Word16Value {
	common.UseMemory(gauge, word16MemoryUsage)

	return NewUnmeteredWord16Value(valueGetter())
}

func NewUnmeteredWord16Value(value uint16) Word16Value {
	return Word16Value(value)
}

func (Word16Value) isValue() {}

func (v Word16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord16Value(interpreter, v)
}

func (Word16Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word16Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord16)
}

func (Word16Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word16Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word16Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word16Value) ToInt(_ LocationRange) int {
	return int(v)
}
func (v Word16Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v + o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v - o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v % o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v * o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v / o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Word16Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Word16Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Word16Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Word16Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord16, ok := other.(Word16Value)
	if !ok {
		return false
	}
	return v == otherWord16
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord16 (1 byte)
// - uint16 value encoded in big-endian (2 bytes)
func (v Word16Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertWord16(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word16Value {
	return NewWord16Value(
		memoryGauge,
		func() uint16 {
			return ConvertWord[uint16](memoryGauge, value, locationRange)
		},
	)
}

func (v Word16Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v | o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v ^ o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v & o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v << o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v >> o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word16Type, locationRange)
}

func (Word16Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word16Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Word16Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word16Value) IsStorable() bool {
	return true
}

func (v Word16Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word16Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word16Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word16Value) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v Word16Value) Clone(_ *Interpreter) Value {
	return v
}

func (Word16Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word16Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v Word16Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word16Value) ChildStorables() []atree.Storable {
	return nil
}
