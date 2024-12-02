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
	"encoding/binary"
	"math"
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// UInt32Value

type UInt32Value uint32

var UInt32MemoryUsage = common.NewNumberMemoryUsage(int(unsafe.Sizeof(UInt32Value(0))))

func NewUInt32Value(gauge common.MemoryGauge, uint32Constructor func() uint32) UInt32Value {
	common.UseMemory(gauge, UInt32MemoryUsage)

	return NewUnmeteredUInt32Value(uint32Constructor())
}

func NewUnmeteredUInt32Value(value uint32) UInt32Value {
	return UInt32Value(value)
}

var _ Value = UInt32Value(0)
var _ atree.Storable = UInt32Value(0)
var _ NumberValue = UInt32Value(0)
var _ IntegerValue = UInt32Value(0)
var _ EquatableValue = UInt32Value(0)
var _ ComparableValue = UInt32Value(0)
var _ HashableValue = UInt32Value(0)
var _ MemberAccessibleValue = UInt32Value(0)

func (UInt32Value) isValue() {}

func (v UInt32Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitUInt32Value(interpreter, v)
}

func (UInt32Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (UInt32Value) StaticType(staticTypeGetter StaticTypeGetter) StaticType {
	return NewPrimitiveStaticType(staticTypeGetter, PrimitiveStaticTypeUInt32)
}

func (UInt32Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v UInt32Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt32Value) MeteredString(interpreter *Interpreter, _ SeenReferences, locationRange LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
		),
	)
	return v.String()
}

func (v UInt32Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v UInt32Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			sum := v + o
			// INT30-C
			if sum < v {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint32(sum)
		},
	)
}

func (v UInt32Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			sum := v + o
			// INT30-C
			if sum < v {
				return math.MaxUint32
			}
			return uint32(sum)
		},
	)
}

func (v UInt32Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			diff := v - o
			// INT30-C
			if diff > v {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return uint32(diff)
		},
	)
}

func (v UInt32Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			diff := v - o
			// INT30-C
			if diff > v {
				return 0
			}
			return uint32(diff)
		},
	)
}

func (v UInt32Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint32(v % o)
		},
	)
}

func (v UInt32Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint32(v * o)
		},
	)
}

func (v UInt32Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {

			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
				return math.MaxUint32
			}
			return uint32(v * o)
		},
	)
}

func (v UInt32Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint32(v / o)
		},
	)
}

func (v UInt32Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
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

func (v UInt32Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
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

func (v UInt32Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
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

func (v UInt32Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
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

func (v UInt32Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
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

func (v UInt32Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt32, ok := other.(UInt32Value)
	if !ok {
		return false
	}
	return v == otherUInt32
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt32 (1 byte)
// - uint32 value encoded in big-endian (4 bytes)
func (v UInt32Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertUInt32(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt32Value {
	return NewUInt32Value(
		memoryGauge,
		func() uint32 {
			return ConvertUnsigned[uint32](
				memoryGauge,
				value,
				sema.UInt32TypeMaxInt,
				math.MaxUint32,
				locationRange,
			)
		},
	)
}

func (v UInt32Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v | o)
		},
	)
}

func (v UInt32Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v ^ o)
		},
	)
}

func (v UInt32Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v & o)
		},
	)
}

func (v UInt32Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v << o)
		},
	)
}

func (v UInt32Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v >> o)
		},
	)
}

func (v UInt32Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt32Type, locationRange)
}

func (UInt32Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt32Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v UInt32Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt32Value) IsStorable() bool {
	return true
}

func (v UInt32Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt32Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt32Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt32Value) Transfer(
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

func (v UInt32Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt32Value) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v UInt32Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UInt32Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt32Value) ChildStorables() []atree.Storable {
	return nil
}
