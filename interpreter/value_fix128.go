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

// Fix128Value
type Fix128Value fix.Fix128

const fix128Size = int(unsafe.Sizeof(Fix128Value{}))

var fix128MemoryUsage = common.NewNumberMemoryUsage(fix128Size)

// NewUnmeteredFix128ValueWithInteger construct a Fix128Value from an int64.
// Note that this function uses the default scaling of 24.
func NewUnmeteredFix128ValueWithInteger(integer int64, locationRange LocationRange) Fix128Value {
	bigInt := big.NewInt(integer)
	bigInt = new(big.Int).Mul(
		bigInt,
		sema.Fix128FactorIntBig,
	)

	return NewFix128ValueFromBigIntWithRangeCheck(nil, bigInt, locationRange)
}

func NewUnmeteredFix128ValueWithIntegerAndScale(integer int64, scale int64) Fix128Value {
	bigInt := big.NewInt(integer)

	bigInt = new(big.Int).Mul(
		bigInt,
		// To remove the fractional, multiply it by the given scale.
		new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(scale),
			nil,
		),
	)

	return NewFix128ValueFromBigInt(nil, bigInt)
}

func NewFix128Value(gauge common.MemoryGauge, valueGetter func() fix.Fix128) Fix128Value {
	common.UseMemory(gauge, fix128MemoryUsage)
	return NewUnmeteredFix128Value(valueGetter())
}

func NewUnmeteredFix128Value(fix128 fix.Fix128) Fix128Value {
	return Fix128Value(fix128)
}

func NewFix128ValueFromBigEndianBytes(gauge common.MemoryGauge, b []byte) Value {
	return NewFix128Value(
		gauge,
		func() fix.Fix128 {
			bytes := padWithZeroes(b, 16)
			high := new(big.Int).SetBytes(bytes[:8]).Uint64()
			low := new(big.Int).SetBytes(bytes[8:]).Uint64()
			return fix.NewFix128(high, low)
		},
	)
}

func NewFix128ValueFromBigInt(gauge common.MemoryGauge, v *big.Int) Fix128Value {
	return NewFix128Value(
		gauge,
		func() fix.Fix128 {
			return fixedpoint.Fix128FromBigInt(v)
		},
	)
}

func NewFix128ValueFromBigIntWithRangeCheck(gauge common.MemoryGauge, v *big.Int, locationRange LocationRange) Fix128Value {
	if v.Cmp(fixedpoint.Fix128TypeMinBig) == -1 {
		panic(&UnderflowError{
			LocationRange: locationRange,
		})
	}

	if v.Cmp(fixedpoint.Fix128TypeMaxBig) == 1 {
		panic(&OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewFix128ValueFromBigInt(gauge, v)
}

var _ Value = Fix128Value{}
var _ atree.Storable = Fix128Value{}
var _ NumberValue = Fix128Value{}
var _ FixedPointValue = Fix128Value{}
var _ EquatableValue = Fix128Value{}
var _ ComparableValue = Fix128Value{}
var _ HashableValue = Fix128Value{}
var _ MemberAccessibleValue = Fix128Value{}

func (Fix128Value) IsValue() {}

func (v Fix128Value) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitFix128Value(context, v)
}

func (Fix128Value) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Fix128Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeFix128)
}

func (Fix128Value) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return true
}

func (v Fix128Value) String() string {
	return format.Fix128(fix.Fix128(v))
}

func (v Fix128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Fix128Value) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		context,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(context, v),
		),
	)
	return v.String()
}

func (v Fix128Value) ToInt(locationRange LocationRange) int {
	// TODO: Maybe compute this without the use of `big.Int`
	fix128BigInt := v.ToBigInt()
	integerPart := fix128BigInt.Div(fix128BigInt, sema.Fix128FactorIntBig)

	if !integerPart.IsInt64() {
		panic(&OverflowError{
			LocationRange: locationRange,
		})
	}

	return int(integerPart.Int64())
}

func (v Fix128Value) Negate(context NumberValueArithmeticContext, locationRange LocationRange) NumberValue {
	valueGetter := func() fix.Fix128 {
		neg, err := fix.Fix128(v).Neg()
		handleFixedpointError(err, locationRange)
		return neg
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Add(fix.Fix128(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Add(fix.Fix128(o))
		return fix128SaturationArithmaticResult(result, err)
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Sub(fix.Fix128(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Sub(fix.Fix128(o))
		return fix128SaturationArithmaticResult(result, err)
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Mul(
			fix.Fix128(o),
			fix.RoundTruncate,
		)

		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Mul(
			fix.Fix128(o),
			fix.RoundTruncate,
		)

		return fix128SaturationArithmaticResult(result, err)
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Div(
			fix.Fix128(o),
			fix.RoundTruncate,
		)

		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Div(
			fix.Fix128(o),
			fix.RoundTruncate,
		)
		return fix128SaturationArithmaticResult(result, err)
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() fix.Fix128 {
		result, err := fix.Fix128(v).Mod(fix.Fix128(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err, locationRange)
		return result
	}

	return NewFix128Value(context, valueGetter)
}

func (v Fix128Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.Fix128(v)
	that := fix.Fix128(o)

	return BoolValue(this.Lt(that))
}

func (v Fix128Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.Fix128(v)
	that := fix.Fix128(o)

	return BoolValue(this.Lte(that))
}

func (v Fix128Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.Fix128(v)
	that := fix.Fix128(o)

	return BoolValue(this.Gt(that))
}

func (v Fix128Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix128Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	this := fix.Fix128(v)
	that := fix.Fix128(o)

	return BoolValue(this.Gte(that))
}

func (v Fix128Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherFix128, ok := other.(Fix128Value)
	if !ok {
		return false
	}
	return v == otherFix128
}

// HashInput returns a byte slice containing:
// - HashInputTypeFix128 (1 byte)
// - high 64 bits encoded in big-endian (8 bytes)
// - low 64 bits encoded in big-endian (8 bytes)
func (v Fix128Value) HashInput(_ common.Gauge, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeFix128)

	fix128 := fix.Fix128(v)
	binary.BigEndian.PutUint64(scratch[1:], uint64(fix128.Hi))
	binary.BigEndian.PutUint64(scratch[9:], uint64(fix128.Lo))
	return scratch[:17]
}

func ConvertFix128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Fix128Value {
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
		return value

	case UFix128Value:
		scaledInt = value.ToBigInt()

	case BigNumberValue:
		bigInt := value.ToBigInt(memoryGauge)
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.Fix128FactorAsBigInt,
		)

	case NumberValue:
		bigInt := new(big.Int).SetInt64(int64(value.ToInt(locationRange)))
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.Fix128FactorAsBigInt,
		)

	default:
		panic(fmt.Sprintf("can't convert to Fix128: %s", value))
	}

	return NewFix128ValueFromBigIntWithRangeCheck(memoryGauge, scaledInt, locationRange)
}

func (v Fix128Value) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return context.GetMethod(v, name, locationRange)
}

func (v Fix128Value) GetMethod(
	context MemberAccessibleContext,
	locationRange LocationRange,
	name string,
) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.Fix128Type, locationRange)
}

func (Fix128Value) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Fix128Value) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Fix128Value) ToBigEndianBytes() []byte {
	fix128 := fix.Fix128(v)
	return fixedpoint.Fix128ToBigEndianBytes(fix128)
}

func (v Fix128Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Fix128Value) IsStorable() bool {
	return true
}

func (v Fix128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Fix128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Fix128Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v Fix128Value) Transfer(
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

func (v Fix128Value) Clone(_ ValueCloneContext) Value {
	return v
}

func (Fix128Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v Fix128Value) ByteSize() uint32 {
	fix128 := fix.Fix128(v)

	// tag number (2 bytes) + array head (1 byte) + high-bits (CBOR uint) + low-bits (CBOR uint)
	return values.CBORTagSize +
		1 +
		values.GetUintCBORSize(uint64(fix128.Hi)) +
		values.GetUintCBORSize(uint64(fix128.Lo))
}

func (v Fix128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Fix128Value) ChildStorables() []atree.Storable {
	return nil
}

func (v Fix128Value) IntegerPart() NumberValue {
	// TODO: Maybe compute this without the use of `big.Int`.
	fix128BigInt := v.ToBigInt()

	integerPart := new(big.Int).Div(fix128BigInt, sema.Fix128FactorIntBig)

	// The max length of the integer part is 128-bits.
	// Therefore, return an `Int128`.
	return NewUnmeteredInt128ValueFromBigInt(integerPart)
}

func (Fix128Value) Scale() int {
	return sema.Fix128Scale
}

func (v Fix128Value) ToBigInt() *big.Int {
	return fixedpoint.Fix128ToBigInt(fix.Fix128(v))
}

func handleFixedpointError(err error, locationRange LocationRange) {
	switch err.(type) {
	// `fix.ErrUnderflow` happens when the value is within the range but is too small
	// to be represented using the current bit-length.
	// These should be treated as non-errors, and should return the truncated value
	// (assumes that the value returned is already the truncated value).
	case nil, fix.UnderflowError:
		return
	case fix.PositiveOverflowError:
		panic(&OverflowError{
			LocationRange: locationRange,
		})
	case fix.NegativeOverflowError:
		panic(&UnderflowError{
			LocationRange: locationRange,
		})
	default:
		panic(err)
	}
}

func fix128SaturationArithmaticResult(
	result fix.Fix128,
	err error,
) fix.Fix128 {
	// Should not panic on overflow/underflow.
	switch err.(type) {
	case nil:
		return result
	case fix.PositiveOverflowError:
		return fix.Fix128Max
	case fix.NegativeOverflowError:
		return fix.Fix128Min
	case fix.UnderflowError:
		return fix.Fix128Zero
	default:
		panic(err)
	}
}
