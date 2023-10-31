package interpreter

import (
	"encoding/binary"
	"math"
	"math/big"
	"unsafe"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// Word64Value

type Word64Value uint64

var _ Value = Word64Value(0)
var _ atree.Storable = Word64Value(0)
var _ NumberValue = Word64Value(0)
var _ IntegerValue = Word64Value(0)
var _ EquatableValue = Word64Value(0)
var _ ComparableValue = Word64Value(0)
var _ HashableValue = Word64Value(0)
var _ MemberAccessibleValue = Word64Value(0)

const word64Size = int(unsafe.Sizeof(Word64Value(0)))

var word64MemoryUsage = common.NewNumberMemoryUsage(word64Size)

func NewWord64Value(gauge common.MemoryGauge, valueGetter func() uint64) Word64Value {
	common.UseMemory(gauge, word64MemoryUsage)

	return NewUnmeteredWord64Value(valueGetter())
}

func NewUnmeteredWord64Value(value uint64) Word64Value {
	return Word64Value(value)
}

// NOTE: important, do *NOT* remove:
// Word64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
var _ BigNumberValue = Word64Value(0)

func (Word64Value) isValue() {}

func (v Word64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord64Value(interpreter, v)
}

func (Word64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord64)
}

func (Word64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word64Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word64Value) ToInt(locationRange LocationRange) int {
	if v > math.MaxInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v)
}

func (v Word64Value) ByteLength() int {
	return 8
}

// ToBigInt
//
// NOTE: important, do *NOT* remove:
// Word64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
func (v Word64Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).SetUint64(uint64(v))
}

func (v Word64Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v + o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v - o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
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

	valueGetter := func() uint64 {
		return uint64(v % o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v * o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
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

	valueGetter := func() uint64 {
		return uint64(v / o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
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

func (v Word64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
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

func (v Word64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
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

func (v Word64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
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

func (v Word64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord64, ok := other.(Word64Value)
	if !ok {
		return false
	}
	return v == otherWord64
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v Word64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertWord64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word64Value {
	return NewWord64Value(
		memoryGauge,
		func() uint64 {
			return ConvertWord[uint64](memoryGauge, value, locationRange)
		},
	)
}

func (v Word64Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v | o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v ^ o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v & o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v << o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v >> o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word64Type, locationRange)
}

func (Word64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Word64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word64Value) IsStorable() bool {
	return true
}

func (v Word64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word64Value) Transfer(
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

func (v Word64Value) Clone(_ *Interpreter) Value {
	return v
}

func (v Word64Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (Word64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word64Value) ChildStorables() []atree.Storable {
	return nil
}

// Word128Value

type Word128Value struct {
	BigInt *big.Int
}

func NewWord128ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Word128Value {
	return NewWord128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Word128MemoryUsage = common.NewBigIntMemoryUsage(16)

func NewWord128ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Word128Value {
	common.UseMemory(memoryGauge, Word128MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredWord128ValueFromBigInt(value)
}

func NewUnmeteredWord128ValueFromUint64(value uint64) Word128Value {
	return NewUnmeteredWord128ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUnmeteredWord128ValueFromBigInt(value *big.Int) Word128Value {
	return Word128Value{
		BigInt: value,
	}
}

var _ Value = Word128Value{}
var _ atree.Storable = Word128Value{}
var _ NumberValue = Word128Value{}
var _ IntegerValue = Word128Value{}
var _ EquatableValue = Word128Value{}
var _ ComparableValue = Word128Value{}
var _ HashableValue = Word128Value{}
var _ MemberAccessibleValue = Word128Value{}

func (Word128Value) isValue() {}

func (v Word128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord128Value(interpreter, v)
}

func (Word128Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word128Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord128)
}

func (Word128Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word128Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Word128Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Word128Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Word128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Word128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word128Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word128Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and wrap around in case of overflow.
			//
			// Note that since v and o are both in the range [0, 2**128 - 1),
			// their sum will be in range [0, 2*(2**128 - 1)).
			// Hence it is sufficient to subtract 2**128 to wrap around.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.Word128TypeMaxIntBig) > 0 {
				sum.Sub(sum, sema.Word128TypeMaxIntPlusOneBig)
			}
			return sum
		},
	)
}

func (v Word128Value) SaturatingPlus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and wrap around in case of underflow.
			//
			// Note that since v and o are both in the range [0, 2**128 - 1),
			// their difference will be in range [-(2**128 - 1), 2**128 - 1).
			// Hence it is sufficient to add 2**128 to wrap around.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Sign() < 0 {
				diff.Add(diff, sema.Word128TypeMaxIntPlusOneBig)
			}
			return diff
		},
	)
}

func (v Word128Value) SaturatingMinus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.Word128TypeMaxIntBig) > 0 {
				res.Mod(res, sema.Word128TypeMaxIntPlusOneBig)
			}
			return res
		},
	)
}

func (v Word128Value) SaturatingMul(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)

}

func (v Word128Value) SaturatingDiv(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Word128Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Word128Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Word128Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Word128Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Word128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord128 (1 byte)
// - big int encoded in big endian (n bytes)
func (v Word128Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeWord128)
	copy(buffer[1:], b)
	return buffer
}

func ConvertWord128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
	return NewWord128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {

			var v *big.Int

			switch value := value.(type) {
			case BigNumberValue:
				v = value.ToBigInt(memoryGauge)

			case NumberValue:
				v = big.NewInt(int64(value.ToInt(locationRange)))

			default:
				panic(errors.NewUnreachableError())
			}

			if v.Cmp(sema.Word128TypeMaxIntBig) > 0 || v.Sign() < 0 {
				// When Sign() < 0, Mod will add sema.Word128TypeMaxIntPlusOneBig
				// to ensure the range is [0, sema.Word128TypeMaxIntPlusOneBig)
				v.Mod(v, sema.Word128TypeMaxIntPlusOneBig)
			}

			return v
		},
	)
}

func (v Word128Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)

}

func (v Word128Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word128Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word128Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word128Type, locationRange)
}

func (Word128Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word128Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word128Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Word128Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word128Value) IsStorable() bool {
	return true
}

func (v Word128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word128Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word128Value) Transfer(
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

func (v Word128Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredWord128ValueFromBigInt(v.BigInt)
}

func (Word128Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word128Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Word128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word128Value) ChildStorables() []atree.Storable {
	return nil
}

// Word256Value

type Word256Value struct {
	BigInt *big.Int
}

func NewWord256ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Word256Value {
	return NewWord256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Word256MemoryUsage = common.NewBigIntMemoryUsage(32)

func NewWord256ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Word256Value {
	common.UseMemory(memoryGauge, Word256MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredWord256ValueFromBigInt(value)
}

func NewUnmeteredWord256ValueFromUint64(value uint64) Word256Value {
	return NewUnmeteredWord256ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUnmeteredWord256ValueFromBigInt(value *big.Int) Word256Value {
	return Word256Value{
		BigInt: value,
	}
}

var _ Value = Word256Value{}
var _ atree.Storable = Word256Value{}
var _ NumberValue = Word256Value{}
var _ IntegerValue = Word256Value{}
var _ EquatableValue = Word256Value{}
var _ ComparableValue = Word256Value{}
var _ HashableValue = Word256Value{}
var _ MemberAccessibleValue = Word256Value{}

func (Word256Value) isValue() {}

func (v Word256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord256Value(interpreter, v)
}

func (Word256Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word256Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord256)
}

func (Word256Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word256Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Word256Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Word256Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Word256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Word256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word256Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word256Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and wrap around in case of overflow.
			//
			// Note that since v and o are both in the range [0, 2**256 - 1),
			// their sum will be in range [0, 2*(2**256 - 1)).
			// Hence it is sufficient to subtract 2**256 to wrap around.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.Word256TypeMaxIntBig) > 0 {
				sum.Sub(sum, sema.Word256TypeMaxIntPlusOneBig)
			}
			return sum
		},
	)
}

func (v Word256Value) SaturatingPlus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and wrap around in case of underflow.
			//
			// Note that since v and o are both in the range [0, 2**256 - 1),
			// their difference will be in range [-(2**256 - 1), 2**256 - 1).
			// Hence it is sufficient to add 2**256 to wrap around.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Sign() < 0 {
				diff.Add(diff, sema.Word256TypeMaxIntPlusOneBig)
			}
			return diff
		},
	)
}

func (v Word256Value) SaturatingMinus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.Word256TypeMaxIntBig) > 0 {
				res.Mod(res, sema.Word256TypeMaxIntPlusOneBig)
			}
			return res
		},
	)
}

func (v Word256Value) SaturatingMul(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)

}

func (v Word256Value) SaturatingDiv(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Word256Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Word256Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Word256Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Word256Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Word256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord256 (1 byte)
// - big int encoded in big endian (n bytes)
func (v Word256Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeWord256)
	copy(buffer[1:], b)
	return buffer
}

func ConvertWord256(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
	return NewWord256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {

			var v *big.Int

			switch value := value.(type) {
			case BigNumberValue:
				v = value.ToBigInt(memoryGauge)

			case NumberValue:
				v = big.NewInt(int64(value.ToInt(locationRange)))

			default:
				panic(errors.NewUnreachableError())
			}

			if v.Cmp(sema.Word256TypeMaxIntBig) > 0 || v.Sign() < 0 {
				// When Sign() < 0, Mod will add sema.Word256TypeMaxIntPlusOneBig
				// to ensure the range is [0, sema.Word256TypeMaxIntPlusOneBig)
				v.Mod(v, sema.Word256TypeMaxIntPlusOneBig)
			}

			return v
		},
	)
}

func (v Word256Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)

}

func (v Word256Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word256Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word256Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word256Type, locationRange)
}

func (Word256Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word256Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word256Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Word256Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word256Value) IsStorable() bool {
	return true
}

func (v Word256Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word256Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word256Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word256Value) Transfer(
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

func (v Word256Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredWord256ValueFromBigInt(v.BigInt)
}

func (Word256Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word256Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Word256Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word256Value) ChildStorables() []atree.Storable {
	return nil
}

// FixedPointValue is a fixed-point number value
type FixedPointValue interface {
	NumberValue
	IntegerPart() NumberValue
	Scale() int
}
