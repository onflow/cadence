package interpreter

import (
	"encoding/binary"
	"math"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// Int64Value

type Int64Value int64

var Int64MemoryUsage = common.NewNumberMemoryUsage(int64Size)

func NewInt64Value(gauge common.MemoryGauge, valueGetter func() int64) Int64Value {
	common.UseMemory(gauge, Int64MemoryUsage)

	return NewUnmeteredInt64Value(valueGetter())
}

func NewUnmeteredInt64Value(value int64) Int64Value {
	return Int64Value(value)
}

var _ Value = Int64Value(0)
var _ atree.Storable = Int64Value(0)
var _ NumberValue = Int64Value(0)
var _ IntegerValue = Int64Value(0)
var _ EquatableValue = Int64Value(0)
var _ ComparableValue = Int64Value(0)
var _ HashableValue = Int64Value(0)
var _ MemberAccessibleValue = Int64Value(0)

func (Int64Value) isValue() {}

func (v Int64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt64Value(interpreter, v)
}

func (Int64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt64)
}

func (Int64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int64Value) String() string {
	return format.Int(int64(v))
}

func (v Int64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int64Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Int64Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(-v)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func safeAddInt64(a, b int64, locationRange LocationRange) int64 {
	// INT32-C
	if (b > 0) && (a > (math.MaxInt64 - b)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (b < 0) && (a < (math.MinInt64 - b)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	return a + b
}

func (v Int64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return safeAddInt64(int64(v), int64(o), locationRange)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt64 - o)) {
			return math.MaxInt64
		} else if (o < 0) && (v < (math.MinInt64 - o)) {
			return math.MinInt64
		}
		return int64(v + o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v - o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt64 + o)) {
			return math.MinInt64
		} else if (o < 0) && (v > (math.MaxInt64 + o)) {
			return math.MaxInt64
		}
		return int64(v - o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
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

	valueGetter := func() int64 {
		return int64(v % o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
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
			if v > (math.MaxInt64 / o) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt64 / v) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt64 / o) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt64 / v)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}

	valueGetter := func() int64 {
		return int64(v * o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if v > 0 {
			if o > 0 {
				// positive * positive = positive. overflow?
				if v > (math.MaxInt64 / o) {
					return math.MaxInt64
				}
			} else {
				// positive * negative = negative. underflow?
				if o < (math.MinInt64 / v) {
					return math.MinInt64
				}
			}
		} else {
			if o > 0 {
				// negative * positive = negative. underflow?
				if v < (math.MinInt64 / o) {
					return math.MinInt64
				}
			} else {
				// negative * negative = positive. overflow?
				if (v != 0) && (o < (math.MaxInt64 / v)) {
					return math.MaxInt64
				}
			}
		}
		return int64(v * o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
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
	} else if (v == math.MinInt64) && (o == -1) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v / o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT33-C
		// https://golang.org/ref/spec#Integer_operators
		if o == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		} else if (v == math.MinInt64) && (o == -1) {
			return math.MaxInt64
		}
		return int64(v / o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
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

func (v Int64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
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

func (v Int64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
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

func (v Int64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
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

func (v Int64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt64, ok := other.(Int64Value)
	if !ok {
		return false
	}
	return v == otherInt64
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt64 (1 byte)
// - int64 value encoded in big-endian (8 bytes)
func (v Int64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertInt64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int64Value {
	converter := func() int64 {
		switch value := value.(type) {
		case BigNumberValue:
			v := value.ToBigInt(memoryGauge)
			if v.Cmp(sema.Int64TypeMaxInt) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Cmp(sema.Int64TypeMinInt) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return v.Int64()

		case NumberValue:
			v := value.ToInt(locationRange)
			return int64(v)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return NewInt64Value(memoryGauge, converter)
}

func (v Int64Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v | o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v ^ o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v & o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v << o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v >> o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int64Type, locationRange)
}

func (Int64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Int64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int64Value) Transfer(
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

func (v Int64Value) Clone(_ *Interpreter) Value {
	return v
}

func (Int64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int64Value) ByteSize() uint32 {
	return cborTagSize + getIntCBORSize(int64(v))
}

func (v Int64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int64Value) ChildStorables() []atree.Storable {
	return nil
}
