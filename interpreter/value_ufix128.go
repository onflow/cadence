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
	"math/big"
	"unsafe"

	"github.com/onflow/atree"

	fix "github.com/onflow/fixed-point"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// UFix128Value
type UFix128Value fix.UFix128

const ufix128Size = int(unsafe.Sizeof(UFix128Value{}))

var UFix128MemoryUsage = common.NewNumberMemoryUsage(ufix128Size)

// NewUnmeteredUFix128ValueWithInteger construct a UFix128Value from an uint64.
// Note that this function uses the default scaling of 24.
func NewUnmeteredUFix128ValueWithInteger(integer uint64, locationRange LocationRange) UFix128Value {
	bigInt := new(big.Int).SetUint64(integer)
	bigInt = new(big.Int).Mul(
		bigInt,
		sema.UFix128FactorIntBig,
	)

	return NewUFix128ValueFromBigIntWithRangeCheck(nil, bigInt, locationRange)
}

func NewUnmeteredUFix128ValueWithIntegerAndScale(integer uint64, scale int64) UFix128Value {
	bigInt := new(big.Int).SetUint64(integer)

	bigInt = new(big.Int).Mul(
		bigInt,
		// To remove the fractional, multiply it by the given scale.
		new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(scale),
			nil,
		),
	)

	return NewUFix128ValueFromBigInt(nil, bigInt)
}

func NewUFix128Value(gauge common.MemoryGauge, valueGetter func() fix.UFix128) UFix128Value {
	common.UseMemory(gauge, UFix128MemoryUsage)
	return NewUnmeteredUFix128Value(valueGetter())
}

func NewUnmeteredUFix128Value(ufix128 fix.UFix128) UFix128Value {
	return UFix128Value(ufix128)
}

func NewUFix128ValueFromBigEndianBytes(gauge common.MemoryGauge, b []byte) Value {
	return NewUFix128Value(
		gauge,
		func() fix.UFix128 {
			bytes := padWithZeroes(b, 16)
			high := new(big.Int).SetBytes(bytes[:8]).Uint64()
			low := new(big.Int).SetBytes(bytes[8:]).Uint64()
			return fix.NewUFix128(high, low)
		},
	)
}

func NewUFix128ValueFromBigInt(gauge common.MemoryGauge, v *big.Int) UFix128Value {
	return NewUFix128Value(
		gauge,
		func() fix.UFix128 {
			return fixedpoint.UFix128FromBigInt(v)
		},
	)
}

func NewUFix128ValueFromBigIntWithRangeCheck(gauge common.MemoryGauge, v *big.Int, locationRange LocationRange) UFix128Value {
	if v.Cmp(fixedpoint.UFix128TypeMinBig) < 0 {
		panic(&UnderflowError{
			LocationRange: locationRange,
		})
	}

	if v.Cmp(fixedpoint.UFix128TypeMaxBig) > 0 {
		panic(&OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUFix128ValueFromBigInt(gauge, v)
}

var _ Value = UFix128Value{}
var _ atree.Storable = UFix128Value{}
var _ NumberValue = UFix128Value{}
var _ FixedPointValue = UFix128Value{}
var _ EquatableValue = UFix128Value{}
var _ ComparableValue = UFix128Value{}
var _ HashableValue = UFix128Value{}
var _ MemberAccessibleValue = UFix128Value{}

func (UFix128Value) IsValue() {}

func (v UFix128Value) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitUFix128Value(context, v)
}

func (UFix128Value) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (UFix128Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeUFix128)
}

func (UFix128Value) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return true
}

func (v UFix128Value) String() string {
	return format.UFix128(fix.UFix128(v))
}

func (v UFix128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UFix128Value) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		context,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(context, v),
		),
	)
	return v.String()
}

func (v UFix128Value) ToInt(locationRange LocationRange) int {
	// TODO: Maybe compute this without the use of `big.Int`
	UFix128BigInt := v.ToBigInt()
	integerPart := UFix128BigInt.Div(UFix128BigInt, sema.UFix128FactorIntBig)

	if !integerPart.IsInt64() {
		panic(&OverflowError{
			LocationRange: locationRange,
		})
	}

	return int(integerPart.Int64())
}

func (v UFix128Value) Negate(context NumberValueArithmeticContext, locationRange LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix128Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Add(fix.UFix128(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Add(fix.UFix128(o))
		return ufix128SaturationArithmaticResult(result, err)
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Sub(fix.UFix128(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Sub(fix.UFix128(o))
		return ufix128SaturationArithmaticResult(result, err)
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Mul(
			fix.UFix128(o),
			fix.RoundTruncate,
		)
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Mul(
			fix.UFix128(o),
			fix.RoundTruncate,
		)
		return ufix128SaturationArithmaticResult(result, err)
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Div(
			fix.UFix128(o),
			fix.RoundTruncate,
		)
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Div(
			fix.UFix128(o),
			fix.RoundTruncate,
		)
		return ufix128SaturationArithmaticResult(result, err)
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.UFix128 {
		result, err := fix.UFix128(v).Mod(fix.UFix128(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewUFix128Value(context, valueGetter)
}

func (v UFix128Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.UFix128(v)
	that := fix.UFix128(o)

	return BoolValue(this.Lt(that))
}

func (v UFix128Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.UFix128(v)
	that := fix.UFix128(o)

	return BoolValue(this.Lte(that))
}

func (v UFix128Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.UFix128(v)
	that := fix.UFix128(o)

	return BoolValue(this.Gt(that))
}

func (v UFix128Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.UFix128(v)
	that := fix.UFix128(o)

	return BoolValue(this.Gte(that))
}

func (v UFix128Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherFix64, ok := other.(UFix128Value)
	if !ok {
		return false
	}
	return v == otherFix64
}

// HashInput returns a byte slice containing:
// - HashInputTypeFix64 (1 byte)
// - high 64 bits encoded in big-endian (8 bytes)
// - low 64 bits encoded in big-endian (8 bytes)
func (v UFix128Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUFix128)

	UFix128 := fix.UFix128(v)
	binary.BigEndian.PutUint64(scratch[1:], uint64(UFix128.Hi))
	binary.BigEndian.PutUint64(scratch[9:], uint64(UFix128.Lo))
	return scratch[:17]
}

func ConvertUFix128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UFix128Value {
	scaledInt := new(big.Int)

	switch value := value.(type) {

	case Fix64Value:
		bigInt := big.NewInt(int64(value))
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.Fix64ToFix128FactorAsBigInt,
		)

	case UFix64Value:
		bigInt := new(big.Int).SetUint64(uint64(value.UFix64Value))
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.Fix64ToFix128FactorAsBigInt,
		)

	case Fix128Value:
		scaledInt = value.ToBigInt()

	case UFix128Value:
		return value

	case BigNumberValue:
		bigInt := value.ToBigInt(memoryGauge)
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.UFix128FactorAsBigInt,
		)

	case NumberValue:
		bigInt := new(big.Int).SetInt64(int64(value.ToInt(locationRange)))
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.UFix128FactorAsBigInt,
		)

	default:
		panic(fmt.Sprintf("can't convert UFix64: %s", value))
	}

	return NewUFix128ValueFromBigIntWithRangeCheck(memoryGauge, scaledInt, locationRange)
}

func (v UFix128Value) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return context.GetMethod(v, name, locationRange)
}

func (v UFix128Value) GetMethod(
	context MemberAccessibleContext,
	locationRange LocationRange,
	name string,
) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.Fix64Type, locationRange)
}

func (UFix128Value) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UFix128Value) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UFix128Value) ToBigEndianBytes() []byte {
	UFix128 := fix.UFix128(v)
	return fixedpoint.Fix128ToBigEndianBytes(fix.Fix128(UFix128))
}

func (v UFix128Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UFix128Value) IsStorable() bool {
	return true
}

func (v UFix128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UFix128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UFix128Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v UFix128Value) Transfer(
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

func (v UFix128Value) Clone(_ ValueCloneContext) Value {
	return v
}

func (UFix128Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v UFix128Value) ByteSize() uint32 {
	UFix128 := fix.UFix128(v)

	// tag number (2 bytes) + array head (1 byte) + high-bits (CBOR uint) + low-bits (CBOR uint)
	return values.CBORTagSize +
		1 +
		values.GetUintCBORSize(uint64(UFix128.Hi)) +
		values.GetUintCBORSize(uint64(UFix128.Lo))
}

func (v UFix128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UFix128Value) ChildStorables() []atree.Storable {
	return nil
}

func (v UFix128Value) IntegerPart() NumberValue {
	// TODO: Maybe compute this without the use of `big.Int`.
	UFix128BigInt := v.ToBigInt()

	integerPart := new(big.Int).Div(UFix128BigInt, sema.UFix128FactorIntBig)

	// The max length of the integer part is 128-bits.
	// Therefore, return an `Int128`.
	return NewUnmeteredInt128ValueFromBigInt(integerPart)
}

func (UFix128Value) Scale() int {
	// same as Fix128Scale
	return sema.Fix128Scale
}

func (v UFix128Value) ToBigInt() *big.Int {
	return fixedpoint.UFix128ToBigInt(fix.UFix128(v))
}

func ufix128SaturationArithmaticResult(
	result fix.UFix128,
	err error,
) fix.UFix128 {
	if err == nil {
		return result
	}

	// Should not panic on overflow/underflow.

	// TODO: Switch on error type, rather than the value.
	// 	Need changes to the fixedpoint library.
	switch err {
	case fix.ErrOverflow:
		return fixedpoint.UFix128TypeMax
	case fix.ErrNegOverflow:
		return fixedpoint.UFix128TypeMin
	case fix.ErrUnderflow:
		return fix.UFix128Zero
	default:
		panic(err)
	}
}
