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
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// Int8Value

type Int8Value int8

const int8Size = int(unsafe.Sizeof(Int8Value(0)))

var Int8MemoryUsage = common.NewNumberMemoryUsage(int8Size)

func NewInt8Value(gauge common.MemoryGauge, valueGetter func() int8) Int8Value {
	common.UseMemory(gauge, Int8MemoryUsage)

	return NewUnmeteredInt8Value(valueGetter())
}

func NewUnmeteredInt8Value(value int8) Int8Value {
	return Int8Value(value)
}

func NewInt8ValueFromBigEndianBytes(gauge common.MemoryGauge, b []byte) Value {
	return NewInt8Value(
		gauge,
		func() int8 {
			bytes := padWithZeroes(b, 1)
			return int8(bytes[0])
		},
	)
}

var _ Value = Int8Value(0)
var _ atree.Storable = Int8Value(0)
var _ NumberValue = Int8Value(0)
var _ IntegerValue = Int8Value(0)
var _ EquatableValue = Int8Value(0)
var _ ComparableValue = Int8Value(0)
var _ HashableValue = Int8Value(0)

func (Int8Value) IsValue() {}

func (v Int8Value) Accept(context ValueVisitContext, visitor Visitor) {
	visitor.VisitInt8Value(context, v)
}

func (Int8Value) Walk(_ ValueWalkContext, _ func(Value)) {
	// NO-OP
}

func (Int8Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeInt8)
}

func (Int8Value) IsImportable(_ ValueImportableContext) bool {
	return true
}

func (v Int8Value) String() string {
	return format.Int(int64(v))
}

func (v Int8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int8Value) MeteredString(
	context ValueStringContext,
	_ SeenReferences,
) string {
	common.UseMemory(
		context,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(context, v),
		),
	)
	return v.String()
}

func (v Int8Value) ToInt() int {
	return int(v)
}

func (v Int8Value) Negate(context NumberValueArithmeticContext) NumberValue {
	// INT32-C
	if v == math.MinInt8 {
		panic(&OverflowError{})
	}

	valueGetter := func() int8 {
		return int8(-v)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) Plus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt8 - o)) {
		panic(&OverflowError{})
	} else if (o < 0) && (v < (math.MinInt8 - o)) {
		panic(&UnderflowError{})
	}

	valueGetter := func() int8 {
		return int8(v + o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() int8 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt8 - o)) {
			return math.MaxInt8
		} else if (o < 0) && (v < (math.MinInt8 - o)) {
			return math.MinInt8
		}
		return int8(v + o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) Minus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt8 + o)) {
		panic(&UnderflowError{})
	} else if (o < 0) && (v > (math.MaxInt8 + o)) {
		panic(&OverflowError{})
	}

	valueGetter := func() int8 {
		return int8(v - o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() int8 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt8 + o)) {
			return math.MinInt8
		} else if (o < 0) && (v > (math.MaxInt8 + o)) {
			return math.MaxInt8
		}
		return int8(v - o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) Mod(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	// INT33-C
	if o == 0 {
		panic(&DivisionByZeroError{})
	}

	valueGetter := func() int8 {
		return int8(v % o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) Mul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt8 / o) {
				panic(&OverflowError{})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt8 / v) {
				panic(&UnderflowError{})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt8 / o) {
				panic(&UnderflowError{})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt8 / v)) {
				panic(&OverflowError{})
			}
		}
	}

	valueGetter := func() int8 {
		return int8(v * o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() int8 {
		// INT32-C
		if v > 0 {
			if o > 0 {
				// positive * positive = positive. overflow?
				if v > (math.MaxInt8 / o) {
					return math.MaxInt8
				}
			} else {
				// positive * negative = negative. underflow?
				if o < (math.MinInt8 / v) {
					return math.MinInt8
				}
			}
		} else {
			if o > 0 {
				// negative * positive = negative. underflow?
				if v < (math.MinInt8 / o) {
					return math.MinInt8
				}
			} else {
				// negative * negative = positive. overflow?
				if (v != 0) && (o < (math.MaxInt8 / v)) {
					return math.MaxInt8
				}
			}
		}

		return int8(v * o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) Div(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(&DivisionByZeroError{})
	} else if (v == math.MinInt8) && (o == -1) {
		panic(&OverflowError{})
	}

	valueGetter := func() int8 {
		return int8(v / o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() int8 {
		// INT33-C
		// https://golang.org/ref/spec#Integer_operators
		if o == 0 {
			panic(&DivisionByZeroError{})
		} else if (v == math.MinInt8) && (o == -1) {
			return math.MaxInt8
		}
		return int8(v / o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) Less(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return v < o
}

func (v Int8Value) LessEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return v <= o
}

func (v Int8Value) Greater(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return v > o
}

func (v Int8Value) GreaterEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return v >= o
}

func (v Int8Value) Equal(_ ValueComparisonContext, other Value) bool {
	otherInt8, ok := other.(Int8Value)
	if !ok {
		return false
	}
	return v == otherInt8
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt8 (1 byte)
// - int8 value (1 byte)
func (v Int8Value) HashInput(_ common.MemoryGauge, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertInt8(memoryGauge common.MemoryGauge, value Value) Int8Value {
	converter := func() int8 {

		switch value := value.(type) {
		case BigNumberValue:
			v := value.ToBigInt(memoryGauge)
			if v.Cmp(sema.Int8TypeMaxInt) > 0 {
				panic(&OverflowError{})
			} else if v.Cmp(sema.Int8TypeMinInt) < 0 {
				panic(&UnderflowError{})
			}
			return int8(v.Int64())

		case NumberValue:
			v := value.ToInt()
			if v > math.MaxInt8 {
				panic(&OverflowError{})
			} else if v < math.MinInt8 {
				panic(&UnderflowError{})
			}
			return int8(v)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return NewInt8Value(memoryGauge, converter)
}

func (v Int8Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() int8 {
		return int8(v | o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() int8 {
		return int8(v ^ o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() int8 {
		return int8(v & o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	if o < 0 {
		panic(&NegativeShiftError{})
	}

	valueGetter := func() int8 {
		return int8(v << o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	if o < 0 {
		panic(&NegativeShiftError{})
	}

	valueGetter := func() int8 {
		return int8(v >> o)
	}

	return NewInt8Value(context, valueGetter)
}

func (v Int8Value) GetMember(context MemberAccessibleContext, name string) Value {
	return context.GetMethod(v, name)
}

func (v Int8Value) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.Int8Type)
}

func (Int8Value) RemoveMember(_ ValueTransferContext, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int8Value) SetMember(_ ValueTransferContext, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Int8Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int8Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int8Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int8Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v Int8Value) Transfer(
	context ValueTransferContext,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return v
}

func (v Int8Value) Clone(_ ValueCloneContext) Value {
	return v
}

func (Int8Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v Int8Value) ByteSize() uint32 {
	return values.CBORTagSize + values.GetIntCBORSize(int64(v))
}

func (v Int8Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int8Value) ChildStorables() []atree.Storable {
	return nil
}
