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

// UInt16Value

type UInt16Value uint16

var _ Value = UInt16Value(0)
var _ atree.Storable = UInt16Value(0)
var _ NumberValue = UInt16Value(0)
var _ IntegerValue = UInt16Value(0)
var _ EquatableValue = UInt16Value(0)
var _ ComparableValue = UInt16Value(0)
var _ HashableValue = UInt16Value(0)
var _ MemberAccessibleValue = UInt16Value(0)

var UInt16MemoryUsage = common.NewNumberMemoryUsage(int(unsafe.Sizeof(UInt16Value(0))))

func NewUInt16Value(gauge common.MemoryGauge, uint16Constructor func() uint16) UInt16Value {
	common.UseMemory(gauge, UInt16MemoryUsage)

	return NewUnmeteredUInt16Value(uint16Constructor())
}

func NewUnmeteredUInt16Value(value uint16) UInt16Value {
	return UInt16Value(value)
}

func (UInt16Value) isValue() {}

func (v UInt16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt16Value(interpreter, v)
}

func (UInt16Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UInt16Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt16)
}

func (UInt16Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UInt16Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt16Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UInt16Value) ToInt(_ LocationRange) int {
	return int(v)
}
func (v UInt16Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			sum := v + o
			// INT30-C
			if sum < v {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint16(sum)
		},
	)
}

func (v UInt16Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			sum := v + o
			// INT30-C
			if sum < v {
				return math.MaxUint16
			}
			return uint16(sum)
		},
	)
}

func (v UInt16Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			diff := v - o
			// INT30-C
			if diff > v {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return uint16(diff)
		},
	)
}

func (v UInt16Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			diff := v - o
			// INT30-C
			if diff > v {
				return 0
			}
			return uint16(diff)
		},
	)
}

func (v UInt16Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint16(v % o)
		},
	)
}

func (v UInt16Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint16(v * o)
		},
	)
}

func (v UInt16Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
				return math.MaxUint16
			}
			return uint16(v * o)
		},
	)
}

func (v UInt16Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint16(v / o)
		},
	)
}

func (v UInt16Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
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

func (v UInt16Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
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

func (v UInt16Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
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

func (v UInt16Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
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

func (v UInt16Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
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

func (v UInt16Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt16, ok := other.(UInt16Value)
	if !ok {
		return false
	}
	return v == otherUInt16
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt16 (1 byte)
// - uint16 value encoded in big-endian (2 bytes)
func (v UInt16Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertUInt16(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt16Value {
	return NewUInt16Value(
		memoryGauge,
		func() uint16 {
			return ConvertUnsigned[uint16](
				memoryGauge,
				value,
				sema.UInt16TypeMaxInt,
				math.MaxUint16,
				locationRange,
			)
		},
	)
}

func (v UInt16Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v | o)
		},
	)
}

func (v UInt16Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v ^ o)
		},
	)
}

func (v UInt16Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v & o)
		},
	)
}

func (v UInt16Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v << o)
		},
	)
}

func (v UInt16Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v >> o)
		},
	)
}

func (v UInt16Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt16Type, locationRange)
}

func (UInt16Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt16Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v UInt16Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt16Value) IsStorable() bool {
	return true
}

func (v UInt16Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt16Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt16Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt16Value) Transfer(
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

func (v UInt16Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt16Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt16Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UInt16Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt16Value) ChildStorables() []atree.Storable {
	return nil
}
