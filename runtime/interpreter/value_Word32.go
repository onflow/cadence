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

// Word32Value

type Word32Value uint32

var _ Value = Word32Value(0)
var _ atree.Storable = Word32Value(0)
var _ NumberValue = Word32Value(0)
var _ IntegerValue = Word32Value(0)
var _ EquatableValue = Word32Value(0)
var _ ComparableValue = Word32Value(0)
var _ HashableValue = Word32Value(0)
var _ MemberAccessibleValue = Word32Value(0)

const word32Size = int(unsafe.Sizeof(Word32Value(0)))

var word32MemoryUsage = common.NewNumberMemoryUsage(word32Size)

func NewWord32Value(gauge common.MemoryGauge, valueGetter func() uint32) Word32Value {
	common.UseMemory(gauge, word32MemoryUsage)

	return NewUnmeteredWord32Value(valueGetter())
}

func NewUnmeteredWord32Value(value uint32) Word32Value {
	return Word32Value(value)
}

func (Word32Value) isValue() {}

func (v Word32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord32Value(interpreter, v)
}

func (Word32Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word32Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord32)
}

func (Word32Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word32Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word32Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word32Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Word32Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v + o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v - o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
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

	valueGetter := func() uint32 {
		return uint32(v % o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v * o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
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

	valueGetter := func() uint32 {
		return uint32(v / o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
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

func (v Word32Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
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

func (v Word32Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
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

func (v Word32Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
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

func (v Word32Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord32, ok := other.(Word32Value)
	if !ok {
		return false
	}
	return v == otherWord32
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord32 (1 byte)
// - uint32 value encoded in big-endian (4 bytes)
func (v Word32Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertWord32(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word32Value {
	return NewWord32Value(
		memoryGauge,
		func() uint32 {
			return ConvertWord[uint32](memoryGauge, value, locationRange)
		},
	)
}

func (v Word32Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v | o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v ^ o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v & o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v << o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v >> o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word32Type, locationRange)
}

func (Word32Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word32Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Word32Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word32Value) IsStorable() bool {
	return true
}

func (v Word32Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word32Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word32Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word32Value) Transfer(
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

func (v Word32Value) Clone(_ *Interpreter) Value {
	return v
}

func (Word32Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word32Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v Word32Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word32Value) ChildStorables() []atree.Storable {
	return nil
}
