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
	"fmt"
	"math"
	"math/big"
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// Fix64Value
type Fix64Value int64

const Fix64MaxValue = math.MaxInt64

const fix64Size = int(unsafe.Sizeof(Fix64Value(0)))

var fix64MemoryUsage = common.NewNumberMemoryUsage(fix64Size)

func NewFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() int64, locationRange LocationRange) Fix64Value {
	common.UseMemory(gauge, fix64MemoryUsage)
	return NewUnmeteredFix64ValueWithInteger(constructor(), locationRange)
}

func NewUnmeteredFix64ValueWithInteger(integer int64, locationRange LocationRange) Fix64Value {

	if integer < sema.Fix64TypeMinInt {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	if integer > sema.Fix64TypeMaxInt {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUnmeteredFix64Value(integer * sema.Fix64Factor)
}

func NewFix64Value(gauge common.MemoryGauge, valueGetter func() int64) Fix64Value {
	common.UseMemory(gauge, fix64MemoryUsage)
	return NewUnmeteredFix64Value(valueGetter())
}

func NewUnmeteredFix64Value(integer int64) Fix64Value {
	return Fix64Value(integer)
}

var _ Value = Fix64Value(0)
var _ atree.Storable = Fix64Value(0)
var _ NumberValue = Fix64Value(0)
var _ FixedPointValue = Fix64Value(0)
var _ EquatableValue = Fix64Value(0)
var _ ComparableValue = Fix64Value(0)
var _ HashableValue = Fix64Value(0)
var _ MemberAccessibleValue = Fix64Value(0)

func (Fix64Value) isValue() {}

func (v Fix64Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitFix64Value(interpreter, v)
}

func (Fix64Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Fix64Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeFix64)
}

func (Fix64Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v Fix64Value) String() string {
	return format.Fix64(int64(v))
}

func (v Fix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Fix64Value) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
		),
	)
	return v.String()
}

func (v Fix64Value) ToInt(_ LocationRange) int {
	return int(v / sema.Fix64Factor)
}

func (v Fix64Value) Negate(context NumberValueArithmeticContext, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(-v)
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return safeAddInt64(int64(v), int64(o), locationRange)
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
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

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
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

		return int64(v - o)
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
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

	return NewFix64Value(context, valueGetter)
}

var minInt64Big = big.NewInt(math.MinInt64)
var maxInt64Big = big.NewInt(math.MaxInt64)

func (v Fix64Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if result.Cmp(minInt64Big) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if result.Cmp(maxInt64Big) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return result.Int64()
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if result.Cmp(minInt64Big) < 0 {
			return math.MinInt64
		} else if result.Cmp(maxInt64Big) > 0 {
			return math.MaxInt64
		}

		return result.Int64()
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, sema.Fix64FactorBig)
		result.Div(result, b)

		if result.Cmp(minInt64Big) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if result.Cmp(maxInt64Big) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return result.Int64()
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, sema.Fix64FactorBig)
		result.Div(result, b)

		if result.Cmp(minInt64Big) < 0 {
			return math.MinInt64
		} else if result.Cmp(maxInt64Big) > 0 {
			return math.MaxInt64
		}

		return result.Int64()
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	// v - int(v/o) * o
	quotient, ok := v.Div(context, o, locationRange).(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	truncatedQuotient := NewFix64Value(
		context,
		func() int64 {
			return (int64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
		},
	)

	return v.Minus(
		context,
		truncatedQuotient.Mul(context, o, locationRange),
		locationRange,
	)
}

func (v Fix64Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
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

func (v Fix64Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
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

func (v Fix64Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
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

func (v Fix64Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
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

func (v Fix64Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherFix64, ok := other.(Fix64Value)
	if !ok {
		return false
	}
	return v == otherFix64
}

// HashInput returns a byte slice containing:
// - HashInputTypeFix64 (1 byte)
// - int64 value encoded in big-endian (8 bytes)
func (v Fix64Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertFix64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Fix64Value {
	switch value := value.(type) {
	case Fix64Value:
		return value

	case UFix64Value:
		if value.UFix64Value > Fix64MaxValue {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		return NewFix64Value(
			memoryGauge,
			func() int64 {
				return int64(value.UFix64Value)
			},
		)

	case BigNumberValue:
		converter := func() int64 {
			v := value.ToBigInt(memoryGauge)

			// First, check if the value is at least in the int64 range.
			// The integer range for Fix64 is smaller, but this test at least
			// allows us to call `v.Int64()` safely.

			if !v.IsInt64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}

			return v.Int64()
		}

		// Now check that the integer value fits the range of Fix64
		return NewFix64ValueWithInteger(memoryGauge, converter, locationRange)

	case NumberValue:
		// Check that the integer value fits the range of Fix64
		return NewFix64ValueWithInteger(
			memoryGauge,
			func() int64 {
				return int64(value.ToInt(locationRange))
			},
			locationRange,
		)

	default:
		panic(fmt.Sprintf("can't convert Fix64: %s", value))
	}
}

func (v Fix64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Fix64Type, locationRange)
}

func (Fix64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Fix64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Fix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Fix64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Fix64Value) IsStorable() bool {
	return true
}

func (v Fix64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Fix64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Fix64Value) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v Fix64Value) Transfer(
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

func (v Fix64Value) Clone(_ *Interpreter) Value {
	return v
}

func (Fix64Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v Fix64Value) ByteSize() uint32 {
	return values.CBORTagSize + values.GetIntCBORSize(int64(v))
}

func (v Fix64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Fix64Value) ChildStorables() []atree.Storable {
	return nil
}

func (v Fix64Value) IntegerPart() NumberValue {
	return UInt64Value(v / sema.Fix64Factor)
}

func (Fix64Value) Scale() int {
	return sema.Fix64Scale
}
