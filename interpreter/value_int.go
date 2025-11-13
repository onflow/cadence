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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// Int

type IntValue struct {
	values.IntValue
}

func NewIntValueFromInt64(memoryGauge common.MemoryGauge, value int64) IntValue {
	return IntValue{
		IntValue: values.NewIntValueFromInt64(memoryGauge, value),
	}
}

func NewUnmeteredIntValueFromInt64(value int64) IntValue {
	return NewUnmeteredIntValueFromBigInt(big.NewInt(value))
}

func NewIntValueFromBigInt(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) IntValue {
	return IntValue{
		IntValue: values.NewIntValueFromBigInt(
			memoryGauge,
			memoryUsage,
			bigIntConstructor,
		),
	}
}

func NewUnmeteredIntValueFromBigInt(value *big.Int) IntValue {
	return IntValue{
		IntValue: values.NewUnmeteredIntValueFromBigInt(value),
	}
}

func NewIntValueFromBigEndianBytes(gauge common.MemoryGauge, b []byte) Value {
	bi := values.BigEndianBytesToSignedBigInt(b)
	memoryUsage := common.NewBigIntMemoryUsage(
		common.BigIntByteLength(bi),
	)
	return NewIntValueFromBigInt(gauge, memoryUsage, func() *big.Int { return bi })
}

func ConvertInt(memoryGauge common.MemoryGauge, value Value) IntValue {
	switch value := value.(type) {
	case BigNumberValue:
		return NewUnmeteredIntValueFromBigInt(
			value.ToBigInt(memoryGauge),
		)

	case NumberValue:
		return NewIntValueFromInt64(
			memoryGauge,
			int64(value.ToInt()),
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

var _ Value = IntValue{}
var _ atree.Storable = IntValue{}
var _ NumberValue = IntValue{}
var _ IntegerValue = IntValue{}
var _ EquatableValue = IntValue{}
var _ ComparableValue = IntValue{}
var _ HashableValue = IntValue{}
var _ MemberAccessibleValue = IntValue{}

func (IntValue) IsValue() {}

func (v IntValue) Accept(context ValueVisitContext, visitor Visitor) {
	visitor.VisitIntValue(context, v)
}

func (IntValue) Walk(_ ValueWalkContext, _ func(Value)) {
	// NO-OP
}

func (IntValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeInt)
}

func (IntValue) IsImportable(_ ValueImportableContext) bool {
	return true
}

func (v IntValue) ToInt() int {
	result, err := v.IntValue.ToInt()
	if _, ok := err.(values.OverflowError); ok {
		panic(&OverflowError{})
	}
	return result
}

func (v IntValue) ToUint32() uint32 {
	if !v.BigInt.IsUint64() {
		panic(&OverflowError{})
	}

	result := v.BigInt.Uint64()

	if result > math.MaxUint32 {
		panic(&OverflowError{})
	}

	return uint32(result)
}

func (v IntValue) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v IntValue) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v IntValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v IntValue) MeteredString(
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

func (v IntValue) Negate(context NumberValueArithmeticContext) NumberValue {
	return IntValue{
		IntValue: v.IntValue.Negate(context),
	}
}

func (v IntValue) Plus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}
	result, err := v.IntValue.Plus(context, o.IntValue)
	if err != nil {
		panic(err)
	}
	return IntValue{IntValue: result}
}

func (v IntValue) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(*InvalidOperandsError); ok {
			panic(&InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingAddFunctionName,
				LeftType:     v.StaticType(context),
				RightType:    other.StaticType(context),
			})
		}
	}()

	return v.Plus(context, other)
}

func (v IntValue) Minus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.Minus(context, o.IntValue)
	if err != nil {
		panic(err)
	}
	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(*InvalidOperandsError); ok {
			panic(&InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
				LeftType:     v.StaticType(context),
				RightType:    other.StaticType(context),
			})
		}
	}()

	return v.Minus(context, other)
}

func (v IntValue) Mod(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.Mod(context, o.IntValue)
	if err != nil {
		panic(err)
	}

	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) Mul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.Mul(context, o.IntValue)
	if err != nil {
		panic(err)
	}
	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) SaturatingMul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(*InvalidOperandsError); ok {
			panic(&InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
				LeftType:     v.StaticType(context),
				RightType:    other.StaticType(context),
			})
		}
	}()

	return v.Mul(context, other)
}

func (v IntValue) Div(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.Div(context, o.IntValue)
	if err != nil {
		panic(err)
	}

	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(*InvalidOperandsError); ok {
			panic(&InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(context),
				RightType:    other.StaticType(context),
			})
		}
	}()

	return v.Div(context, other)
}

func (v IntValue) Less(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return BoolValue(v.IntValue.Less(context, o.IntValue))
}

func (v IntValue) LessEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return BoolValue(v.IntValue.LessEqual(context, o.IntValue))
}

func (v IntValue) Greater(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return BoolValue(v.IntValue.Greater(context, o.IntValue))
}

func (v IntValue) GreaterEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	return BoolValue(v.IntValue.GreaterEqual(context, o.IntValue))
}

func (v IntValue) Equal(context ValueComparisonContext, other Value) bool {
	otherInt, ok := other.(IntValue)
	if !ok {
		return false
	}

	return v.IntValue.Equal(context, otherInt.IntValue)
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt (1 byte)
// - big int encoded in big-endian (n bytes)
func (v IntValue) HashInput(_ common.Gauge, scratch []byte) []byte {
	b := values.SignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeInt)
	copy(buffer[1:], b)
	return buffer
}

func (v IntValue) BitwiseOr(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.BitwiseOr(context, o.IntValue)
	if err != nil {
		panic(err)
	}

	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) BitwiseXor(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.BitwiseXor(context, o.IntValue)
	if err != nil {
		panic(err)
	}

	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.BitwiseAnd(context, o.IntValue)
	if err != nil {
		panic(err)
	}

	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.BitwiseLeftShift(context, o.IntValue)
	if err != nil {
		panic(err)
	}

	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	result, err := v.IntValue.BitwiseRightShift(context, o.IntValue)
	if err != nil {
		panic(err)
	}

	return IntValue{
		IntValue: result,
	}
}

func (v IntValue) GetMember(context MemberAccessibleContext, name string) Value {
	return context.GetMethod(v, name)
}

func (v IntValue) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.IntType)
}

func (IntValue) RemoveMember(_ ValueTransferContext, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (IntValue) SetMember(_ ValueTransferContext, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v IntValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ TypeConformanceResults,
) bool {
	return true
}

func (IntValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (IntValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v IntValue) Transfer(
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

func (v IntValue) Clone(_ ValueCloneContext) Value {
	return NewUnmeteredIntValueFromBigInt(v.BigInt)
}

func (IntValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}
