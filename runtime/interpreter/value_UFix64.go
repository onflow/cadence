package interpreter

import (
	"encoding/binary"
	"fmt"
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

// UFix64Value
type UFix64Value uint64

const UFix64MaxValue = math.MaxUint64

const ufix64Size = int(unsafe.Sizeof(UFix64Value(0)))

var ufix64MemoryUsage = common.NewNumberMemoryUsage(ufix64Size)

func NewUFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() uint64, locationRange LocationRange) UFix64Value {
	common.UseMemory(gauge, ufix64MemoryUsage)
	return NewUnmeteredUFix64ValueWithInteger(constructor(), locationRange)
}

func NewUnmeteredUFix64ValueWithInteger(integer uint64, locationRange LocationRange) UFix64Value {
	if integer > sema.UFix64TypeMaxInt {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUnmeteredUFix64Value(integer * sema.Fix64Factor)
}

func NewUFix64Value(gauge common.MemoryGauge, constructor func() uint64) UFix64Value {
	common.UseMemory(gauge, ufix64MemoryUsage)
	return NewUnmeteredUFix64Value(constructor())
}

func NewUnmeteredUFix64Value(integer uint64) UFix64Value {
	return UFix64Value(integer)
}

var _ Value = UFix64Value(0)
var _ atree.Storable = UFix64Value(0)
var _ NumberValue = UFix64Value(0)
var _ FixedPointValue = Fix64Value(0)
var _ EquatableValue = UFix64Value(0)
var _ ComparableValue = UFix64Value(0)
var _ HashableValue = UFix64Value(0)
var _ MemberAccessibleValue = UFix64Value(0)

func (UFix64Value) isValue() {}

func (v UFix64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUFix64Value(interpreter, v)
}

func (UFix64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UFix64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUFix64)
}

func (UFix64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UFix64Value) String() string {
	return format.UFix64(uint64(v))
}

func (v UFix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UFix64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UFix64Value) ToInt(_ LocationRange) int {
	return int(v / sema.Fix64Factor)
}

func (v UFix64Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return safeAddUint64(uint64(v), uint64(o), locationRange)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		sum := v + o
		// INT30-C
		if sum < v {
			return math.MaxUint64
		}
		return uint64(sum)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		diff := v - o

		// INT30-C
		if diff > v {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return uint64(diff)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		diff := v - o

		// INT30-C
		if diff > v {
			return 0
		}
		return uint64(diff)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	valueGetter := func() uint64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if !result.IsUint64() {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return result.Uint64()
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	valueGetter := func() uint64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if !result.IsUint64() {
			return math.MaxUint64
		}

		return result.Uint64()
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	valueGetter := func() uint64 {
		result := new(big.Int).Mul(a, sema.Fix64FactorBig)
		result.Div(result, b)

		return result.Uint64()
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UFix64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// v - int(v/o) * o
	quotient, ok := v.Div(interpreter, o, locationRange).(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	truncatedQuotient := NewUFix64Value(
		interpreter,
		func() uint64 {
			return (uint64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
		},
	)

	return v.Minus(
		interpreter,
		truncatedQuotient.Mul(interpreter, o, locationRange),
		locationRange,
	)
}

func (v UFix64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
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

func (v UFix64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
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

func (v UFix64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
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

func (v UFix64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
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

func (v UFix64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUFix64, ok := other.(UFix64Value)
	if !ok {
		return false
	}
	return v == otherUFix64
}

// HashInput returns a byte slice containing:
// - HashInputTypeUFix64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v UFix64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertUFix64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UFix64Value {
	switch value := value.(type) {
	case UFix64Value:
		return value

	case Fix64Value:
		if value < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return NewUFix64Value(
			memoryGauge,
			func() uint64 {
				return uint64(value)
			},
		)

	case BigNumberValue:
		converter := func() uint64 {
			v := value.ToBigInt(memoryGauge)

			if v.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}

			// First, check if the value is at least in the uint64 range.
			// The integer range for UFix64 is smaller, but this test at least
			// allows us to call `v.UInt64()` safely.

			if !v.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}

			return v.Uint64()
		}

		// Now check that the integer value fits the range of UFix64
		return NewUFix64ValueWithInteger(memoryGauge, converter, locationRange)

	case NumberValue:
		converter := func() uint64 {
			v := value.ToInt(locationRange)
			if v < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}

			return uint64(v)
		}

		// Check that the integer value fits the range of UFix64
		return NewUFix64ValueWithInteger(memoryGauge, converter, locationRange)

	default:
		panic(fmt.Sprintf("can't convert to UFix64: %s", value))
	}
}

func (v UFix64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UFix64Type, locationRange)
}

func (UFix64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UFix64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UFix64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UFix64Value) IsStorable() bool {
	return true
}

func (v UFix64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UFix64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UFix64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UFix64Value) Transfer(
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

func (v UFix64Value) Clone(_ *Interpreter) Value {
	return v
}

func (UFix64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UFix64Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UFix64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UFix64Value) ChildStorables() []atree.Storable {
	return nil
}

func (v UFix64Value) IntegerPart() NumberValue {
	return UInt64Value(v / sema.Fix64Factor)
}

func (UFix64Value) Scale() int {
	return sema.Fix64Scale
}
