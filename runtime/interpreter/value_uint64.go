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

package interpreter

import (
	"math"
	"math/big"
	"unsafe"

	"encoding/binary"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// UInt64Value

type UInt64Value uint64

var _ Value = UInt64Value(0)
var _ atree.Storable = UInt64Value(0)
var _ NumberValue = UInt64Value(0)
var _ IntegerValue = UInt64Value(0)
var _ EquatableValue = UInt64Value(0)
var _ ComparableValue = UInt64Value(0)
var _ HashableValue = UInt64Value(0)
var _ MemberAccessibleValue = UInt64Value(0)

// NOTE: important, do *NOT* remove:
// UInt64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
var _ BigNumberValue = UInt64Value(0)

var UInt64MemoryUsage = common.NewNumberMemoryUsage(int(unsafe.Sizeof(UInt64Value(0))))

func NewUInt64Value(gauge common.MemoryGauge, uint64Constructor func() uint64) UInt64Value {
	common.UseMemory(gauge, UInt64MemoryUsage)

	return NewUnmeteredUInt64Value(uint64Constructor())
}

func NewUnmeteredUInt64Value(value uint64) UInt64Value {
	return UInt64Value(value)
}

func (UInt64Value) isValue() {}

func (v UInt64Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitUInt64Value(interpreter, v)
}

func (UInt64Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (UInt64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt64)
}

func (UInt64Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v UInt64Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt64Value) MeteredString(interpreter *Interpreter, _ SeenReferences, locationRange LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
		),
	)
	return v.String()
}

func (v UInt64Value) ToInt(locationRange LocationRange) int {
	if v > math.MaxInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v)
}

func (v UInt64Value) ByteLength() int {
	return 8
}

// ToBigInt
//
// NOTE: important, do *NOT* remove:
// UInt64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
func (v UInt64Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).SetUint64(uint64(v))
}

func (v UInt64Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func safeAddUint64(a, b uint64, locationRange LocationRange) uint64 {
	sum := a + b
	// INT30-C
	if sum < a {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return sum
}

func (v UInt64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return safeAddUint64(uint64(v), uint64(o), locationRange)
		},
	)
}

func (v UInt64Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			sum := v + o
			// INT30-C
			if sum < v {
				return math.MaxUint64
			}
			return uint64(sum)
		},
	)
}

func (v UInt64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			diff := v - o
			// INT30-C
			if diff > v {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return uint64(diff)
		},
	)
}

func (v UInt64Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			diff := v - o
			// INT30-C
			if diff > v {
				return 0
			}
			return uint64(diff)
		},
	)
}

func (v UInt64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint64(v % o)
		},
	)
}

func (v UInt64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint64(v * o)
		},
	)
}

func (v UInt64Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
				return math.MaxUint64
			}
			return uint64(v * o)
		},
	)
}

func (v UInt64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint64(v / o)
		},
	)
}

func (v UInt64Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
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

func (v UInt64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
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

func (v UInt64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
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

func (v UInt64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
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

func (v UInt64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
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

func (v UInt64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt64, ok := other.(UInt64Value)
	if !ok {
		return false
	}
	return v == otherUInt64
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v UInt64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertUInt64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt64Value {
	return NewUInt64Value(
		memoryGauge,
		func() uint64 {
			return ConvertUnsigned[uint64](
				memoryGauge,
				value,
				sema.UInt64TypeMaxInt,
				-1,
				locationRange,
			)
		},
	)
}

func (v UInt64Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v | o)
		},
	)
}

func (v UInt64Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v ^ o)
		},
	)
}

func (v UInt64Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v & o)
		},
	)
}

func (v UInt64Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v << o)
		},
	)
}

func (v UInt64Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v >> o)
		},
	)
}

func (v UInt64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt64Type, locationRange)
}

func (UInt64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UInt64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt64Value) IsStorable() bool {
	return true
}

func (v UInt64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt64Value) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v UInt64Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt64Value) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v UInt64Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UInt64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt64Value) ChildStorables() []atree.Storable {
	return nil
}
