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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// UFix64Value

type UFix64Value struct {
	values.UFix64Value
}

const UFix64MaxValue = math.MaxUint64

func NewUFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() uint64, locationRange LocationRange) UFix64Value {
	ufix64Value, err := values.NewUFix64ValueWithInteger(gauge, func() (uint64, error) {
		return constructor(), nil
	})
	if err != nil {
		if _, ok := err.(values.OverflowError); ok {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		panic(err)
	}
	return UFix64Value{
		UFix64Value: ufix64Value,
	}
}

func NewUnmeteredUFix64ValueWithInteger(integer uint64, locationRange LocationRange) UFix64Value {
	ufix64Value, err := values.NewUnmeteredUFix64ValueWithInteger(integer)
	if err != nil {
		if _, ok := err.(values.OverflowError); ok {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		panic(err)
	}
	return UFix64Value{
		UFix64Value: ufix64Value,
	}
}

func NewUFix64Value(gauge common.MemoryGauge, constructor func() uint64) UFix64Value {
	ufix64Value, err := values.NewUFix64Value(gauge, func() (uint64, error) {
		return constructor(), nil
	})
	if err != nil {
		panic(err)
	}
	return UFix64Value{
		UFix64Value: ufix64Value,
	}
}

func NewUnmeteredUFix64Value(integer uint64) UFix64Value {
	return UFix64Value{
		UFix64Value: values.NewUnmeteredUFix64Value(integer),
	}
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

var _ Value = UFix64Value{}
var _ atree.Storable = UFix64Value{}
var _ NumberValue = UFix64Value{}
var _ FixedPointValue = UFix64Value{}
var _ EquatableValue = UFix64Value{}
var _ ComparableValue = UFix64Value{}
var _ HashableValue = UFix64Value{}
var _ MemberAccessibleValue = UFix64Value{}

func (UFix64Value) IsValue() {}

func (v UFix64Value) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitUFix64Value(context, v)
}

func (UFix64Value) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (UFix64Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeUFix64)
}

func (UFix64Value) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return true
}

func (v UFix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UFix64Value) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		context,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(context, v),
		),
	)
	return v.String()
}

func (v UFix64Value) ToInt(_ LocationRange) int {
	result, err := v.UFix64Value.ToInt()
	if err != nil {
		panic(err)
	}
	return result
}

func (v UFix64Value) Negate(NumberValueArithmeticContext, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}
	result, err := v.UFix64Value.Plus(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	result, err := v.UFix64Value.SaturatingPlus(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	result, err := v.UFix64Value.Minus(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	result, err := v.UFix64Value.SaturatingMinus(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	result, err := v.UFix64Value.Mul(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	result, err := v.UFix64Value.SaturatingMul(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	result, err := v.UFix64Value.Div(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
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

func (v UFix64Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	result, err := v.UFix64Value.Mod(context, o.UFix64Value)
	if err != nil {
		panic(err)
	}
	return UFix64Value{UFix64Value: result}
}

func (v UFix64Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.UFix64Value.Less(o.UFix64Value))
}

func (v UFix64Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.UFix64Value.LessEqual(o.UFix64Value))
}

func (v UFix64Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.UFix64Value.Greater(o.UFix64Value))
}

func (v UFix64Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.UFix64Value.GreaterEqual(o.UFix64Value))
}

func (v UFix64Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherUFix64, ok := other.(UFix64Value)
	if !ok {
		return false
	}

	return v.UFix64Value.Equal(otherUFix64.UFix64Value)
}

// HashInput returns a byte slice containing:
// - HashInputTypeUFix64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v UFix64Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v.UFix64Value))
	return scratch[:9]
}

func (v UFix64Value) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return context.GetMethod(v, name, locationRange)
}

func (v UFix64Value) GetMethod(
	context MemberAccessibleContext,
	locationRange LocationRange,
	name string,
) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.UFix64Type, locationRange)
}

func (UFix64Value) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UFix64Value) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UFix64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UFix64Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v UFix64Value) Transfer(
	context ValueTransferContext,
	_ LocationRange,
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

func (v UFix64Value) Clone(_ ValueCloneContext) Value {
	return v
}

func (UFix64Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v UFix64Value) IntegerPart() NumberValue {
	return UInt64Value(v.UFix64Value.IntegerPart())
}
