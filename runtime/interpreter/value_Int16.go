package interpreter

import (
	"encoding/binary"
	"math"
	"unsafe"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// Int16Value

type Int16Value int16

const int16Size = int(unsafe.Sizeof(Int16Value(0)))

var Int16MemoryUsage = common.NewNumberMemoryUsage(int16Size)

func NewInt16Value(gauge common.MemoryGauge, valueGetter func() int16) Int16Value {
	common.UseMemory(gauge, Int16MemoryUsage)

	return NewUnmeteredInt16Value(valueGetter())
}

func NewUnmeteredInt16Value(value int16) Int16Value {
	return Int16Value(value)
}

var _ Value = Int16Value(0)
var _ atree.Storable = Int16Value(0)
var _ NumberValue = Int16Value(0)
var _ IntegerValue = Int16Value(0)
var _ EquatableValue = Int16Value(0)
var _ ComparableValue = Int16Value(0)
var _ HashableValue = Int16Value(0)
var _ MemberAccessibleValue = Int16Value(0)

func (Int16Value) isValue() {}

func (v Int16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt16Value(interpreter, v)
}

func (Int16Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int16Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt16)
}

func (Int16Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int16Value) String() string {
	return format.Int(int64(v))
}

func (v Int16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int16Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int16Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Int16Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt16 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(-v)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt16 - o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v < (math.MinInt16 - o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v + o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt16 - o)) {
			return math.MaxInt16
		} else if (o < 0) && (v < (math.MinInt16 - o)) {
			return math.MinInt16
		}
		return int16(v + o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt16 + o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v > (math.MaxInt16 + o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v - o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt16 + o)) {
			return math.MinInt16
		} else if (o < 0) && (v > (math.MaxInt16 + o)) {
			return math.MaxInt16
		}
		return int16(v - o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v % o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt16 / o) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt16 / v) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt16 / o) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt16 / v)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}

	valueGetter := func() int16 {
		return int16(v * o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT32-C
		if v > 0 {
			if o > 0 {
				// positive * positive = positive. overflow?
				if v > (math.MaxInt16 / o) {
					return math.MaxInt16
				}
			} else {
				// positive * negative = negative. underflow?
				if o < (math.MinInt16 / v) {
					return math.MinInt16
				}
			}
		} else {
			if o > 0 {
				// negative * positive = negative. underflow?
				if v < (math.MinInt16 / o) {
					return math.MinInt16
				}
			} else {
				// negative * negative = positive. overflow?
				if (v != 0) && (o < (math.MaxInt16 / v)) {
					return math.MaxInt16
				}
			}
		}
		return int16(v * o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	} else if (v == math.MinInt16) && (o == -1) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v / o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT33-C
		// https://golang.org/ref/spec#Integer_operators
		if o == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		} else if (v == math.MinInt16) && (o == -1) {
			return math.MaxInt16
		}
		return int16(v / o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
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

func (v Int16Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
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

func (v Int16Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
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

func (v Int16Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
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

func (v Int16Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt16, ok := other.(Int16Value)
	if !ok {
		return false
	}
	return v == otherInt16
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt16 (1 byte)
// - int16 value encoded in big-endian (2 bytes)
func (v Int16Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertInt16(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int16Value {
	converter := func() int16 {

		switch value := value.(type) {
		case BigNumberValue:
			v := value.ToBigInt(memoryGauge)
			if v.Cmp(sema.Int16TypeMaxInt) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Cmp(sema.Int16TypeMinInt) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int16(v.Int64())

		case NumberValue:
			v := value.ToInt(locationRange)
			if v > math.MaxInt16 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v < math.MinInt16 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int16(v)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return NewInt16Value(memoryGauge, converter)
}

func (v Int16Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v | o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v ^ o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v & o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v << o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v >> o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int16Type, locationRange)
}

func (Int16Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int16Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Int16Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int16Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int16Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int16Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int16Value) Transfer(
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

func (v Int16Value) Clone(_ *Interpreter) Value {
	return v
}

func (Int16Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int16Value) ByteSize() uint32 {
	return cborTagSize + getIntCBORSize(int64(v))
}

func (v Int16Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int16Value) ChildStorables() []atree.Storable {
	return nil
}
