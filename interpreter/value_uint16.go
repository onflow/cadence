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
	"github.com/onflow/cadence/values"
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

func (v UInt16Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitUInt16Value(interpreter, v)
}

func (UInt16Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (UInt16Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeUInt16)
}

func (UInt16Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v UInt16Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt16Value) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
		),
	)
	return v.String()
}

func (v UInt16Value) ToInt(_ LocationRange) int {
	return int(v)
}
func (v UInt16Value) Negate(NumberValueArithmeticContext, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (v UInt16Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (v UInt16Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (v UInt16Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (v UInt16Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (v UInt16Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (v UInt16Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
		func() uint16 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
				return math.MaxUint16
			}
			return uint16(v * o)
		},
	)
}

func (v UInt16Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (v UInt16Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(context),
				RightType:     other.StaticType(context),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(context, other, locationRange)
}

func (v UInt16Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v < o
}

func (v UInt16Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v <= o
}

func (v UInt16Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v > o
}

func (v UInt16Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v >= o
}

func (v UInt16Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherUInt16, ok := other.(UInt16Value)
	if !ok {
		return false
	}
	return v == otherUInt16
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt16 (1 byte)
// - uint16 value encoded in big-endian (2 bytes)
func (v UInt16Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
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

func (v UInt16Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
		func() uint16 {
			return uint16(v | o)
		},
	)
}

func (v UInt16Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
		func() uint16 {
			return uint16(v ^ o)
		},
	)
}

func (v UInt16Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
		func() uint16 {
			return uint16(v & o)
		},
	)
}

func (v UInt16Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
		func() uint16 {
			return uint16(v << o)
		},
	)
}

func (v UInt16Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		context,
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

func (UInt16Value) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v UInt16Value) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		context.RemoveReferencedSlab(storable)
	}
	return v
}

func (v UInt16Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt16Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v UInt16Value) ByteSize() uint32 {
	return values.CBORTagSize + values.GetUintCBORSize(uint64(v))
}

func (v UInt16Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt16Value) ChildStorables() []atree.Storable {
	return nil
}
