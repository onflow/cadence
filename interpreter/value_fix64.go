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

// Fix64Value
type Fix64Value fix.Fix64

const fix64Size = int(unsafe.Sizeof(Fix64Value(0)))

var fix64MemoryUsage = common.NewNumberMemoryUsage(fix64Size)

// Note that this function uses the default scaling of 8.
func NewUnmeteredFix64ValueWithInteger(integer int64) Fix64Value {
	bigInt := big.NewInt(integer)
	bigInt = new(big.Int).Mul(
		bigInt,
		sema.Fix64FactorBig,
	)

	return NewFix64ValueFromBigIntWithRangeCheck(nil, bigInt)
}

func NewUnmeteredFix64ValueWithIntegerAndScale(integer int64, scale int64) Fix64Value {
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

	return NewFix64ValueFromBigInt(nil, bigInt)
}

func NewFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() int64) Fix64Value {
	common.UseMemory(gauge, fix64MemoryUsage)
	return NewUnmeteredFix64ValueWithInteger(constructor())
}

func NewFix64Value(gauge common.MemoryGauge, valueGetter func() fix.Fix64) Fix64Value {
	common.UseMemory(gauge, fix64MemoryUsage)
	return NewUnmeteredFix64Value(valueGetter())
}

func NewUnmeteredFix64Value(fix64 fix.Fix64) Fix64Value {
	return Fix64Value(fix64)
}

func NewFix64ValueFromBigEndianBytes(gauge common.MemoryGauge, b []byte) Value {
	return NewFix64Value(
		gauge,
		func() fix.Fix64 {
			bytes := padWithZeroes(b, 8)
			val := int64(binary.BigEndian.Uint64(bytes))
			return fix.Fix64(val)
		},
	)
}

func NewFix64ValueFromBigInt(gauge common.MemoryGauge, v *big.Int) Fix64Value {
	return NewFix64Value(
		gauge,
		func() fix.Fix64 {
			return fixedpoint.Fix64FromBigInt(v)
		},
	)
}

func NewFix64ValueFromBigIntWithRangeCheck(gauge common.MemoryGauge, v *big.Int) Fix64Value {
	if v.Cmp(fixedpoint.Fix64TypeMin) == -1 {
		panic(&UnderflowError{})
	}

	if v.Cmp(fixedpoint.Fix64TypeMax) == 1 {
		panic(&OverflowError{})
	}

	return NewFix64ValueFromBigInt(gauge, v)
}

var _ Value = Fix64Value(0)
var _ atree.Storable = Fix64Value(0)
var _ NumberValue = Fix64Value(0)
var _ FixedPointValue = Fix64Value(0)
var _ EquatableValue = Fix64Value(0)
var _ ComparableValue = Fix64Value(0)
var _ HashableValue = Fix64Value(0)
var _ MemberAccessibleValue = Fix64Value(0)

func (Fix64Value) IsValue() {}

func (v Fix64Value) Accept(context ValueVisitContext, visitor Visitor) {
	visitor.VisitFix64Value(context, v)
}

func (Fix64Value) Walk(_ ValueWalkContext, _ func(Value)) {
	// NO-OP
}

func (Fix64Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeFix64)
}

func (Fix64Value) IsImportable(_ ValueImportableContext) bool {
	return true
}

func (v Fix64Value) String() string {
	return format.Fix64(int64(v))
}

func (v Fix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Fix64Value) MeteredString(
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

func (v Fix64Value) ToInt() int {
	// TODO: Maybe compute this without the use of `big.Int`
	fix64BigInt := v.ToBigInt()
	integerPart := fix64BigInt.Div(fix64BigInt, sema.Fix64FactorBig)

	if !integerPart.IsInt64() {
		panic(&OverflowError{})
	}

	return int(integerPart.Int64())
}

func (v Fix64Value) Negate(context NumberValueArithmeticContext) NumberValue {
	valueGetter := func() fix.Fix64 {
		neg, err := fix.Fix64(v).Neg()
		handleFixedpointError(err)
		return neg
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Plus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Add(fix.Fix64(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err)
		return result
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Add(fix.Fix64(o))
		return fix64SaturationArithmaticResult(result, err)
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Minus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Sub(fix.Fix64(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err)
		return result
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Sub(fix.Fix64(o))
		return fix64SaturationArithmaticResult(result, err)
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Mul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Mul(
			fix.Fix64(o),
			fix.RoundTruncate,
		)

		// Should panic on overflow/underflow
		handleFixedpointError(err)
		return result
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Mul(
			fix.Fix64(o),
			fix.RoundTruncate,
		)

		return fix64SaturationArithmaticResult(result, err)
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Div(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Div(
			fix.Fix64(o),
			fix.RoundTruncate,
		)
		// Should panic on overflow/underflow
		handleFixedpointError(err)
		return result
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Div(
			fix.Fix64(o),
			fix.RoundTruncate,
		)
		return fix64SaturationArithmaticResult(result, err)
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Mod(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.Fix64 {
		result, err := fix.Fix64(v).Mod(fix.Fix64(o))
		// Should panic on overflow/underflow
		handleFixedpointError(err)
		return result
	}

	return NewFix64Value(context, valueGetter)
}

func (v Fix64Value) Less(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.Fix64(v)
	that := fix.Fix64(o)

	return BoolValue(this.Lt(that))
}

func (v Fix64Value) LessEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.Fix64(v)
	that := fix.Fix64(o)

	return BoolValue(this.Lte(that))
}

func (v Fix64Value) Greater(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.Fix64(v)
	that := fix.Fix64(o)

	return BoolValue(this.Gt(that))
}

func (v Fix64Value) GreaterEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.Fix64(v)
	that := fix.Fix64(o)

	return BoolValue(this.Gte(that))
}

func (v Fix64Value) Equal(_ ValueComparisonContext, other Value) bool {
	otherFix64, ok := other.(Fix64Value)
	if !ok {
		return false
	}
	return v == otherFix64
}

// HashInput returns a byte slice containing:
// - HashInputTypeFix64 (1 byte)
// - int64 value encoded in big-endian (8 bytes)
func (v Fix64Value) HashInput(_ common.MemoryGauge, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertFix64(memoryGauge common.MemoryGauge, value Value) Fix64Value {
	scaledInt := new(big.Int)

	switch value := value.(type) {
	case Fix64Value:
		return value

	case UFix64Value:
		bigInt := UFix64ToBigInt(value)
		scaledInt = bigInt

	case Fix128Value:
		scaledInt = value.ToBigInt()

	case UFix128Value:
		scaledInt = value.ToBigInt()

	case BigNumberValue:
		bigInt := value.ToBigInt(memoryGauge)
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.Fix64FactorAsBigInt,
		)

	case NumberValue:
		bigInt := new(big.Int).SetInt64(int64(value.ToInt()))
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.Fix64FactorAsBigInt,
		)

	default:
		panic(fmt.Sprintf("can't convert to Fix64: %s", value))
	}

	return NewFix64ValueFromBigIntWithRangeCheck(memoryGauge, scaledInt)
}

func (v Fix64Value) GetMember(context MemberAccessibleContext, name string) Value {
	return context.GetMethod(v, name)
}

func (v Fix64Value) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.Fix64Type)
}

func (Fix64Value) RemoveMember(_ ValueTransferContext, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Fix64Value) SetMember(_ ValueTransferContext, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Fix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Fix64Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
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

func (Fix64Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v Fix64Value) Transfer(
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

func (v Fix64Value) Clone(_ ValueCloneContext) Value {
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
	// TODO: Maybe compute this without the use of `big.Int`
	fix64BigInt := v.ToBigInt()
	integerPart := fix64BigInt.Div(fix64BigInt, sema.Fix64FactorBig)

	if !integerPart.IsInt64() {
		panic(&OverflowError{})
	}

	return UInt64Value(integerPart.Uint64())
}

func (Fix64Value) Scale() int {
	return sema.Fix64Scale
}

func (v Fix64Value) ToBigInt() *big.Int {
	return fixedpoint.Fix64ToBigInt(fix.Fix64(v))
}

func UFix64ToBigInt(value UFix64Value) *big.Int {
	return fixedpoint.UFix64ToBigInt(fix.UFix64(uint64(value)))
}

func fix128BigIntToFix64(
	memoryGauge common.MemoryGauge,
	bigInt *big.Int,
) Fix64Value {

	if bigInt.Cmp(fixedpoint.Fix64TypeMaxScaledTo128) > 0 {
		panic(&OverflowError{})
	} else if bigInt.Cmp(fixedpoint.Fix64TypeMinScaledTo128) < 0 {
		panic(&UnderflowError{})
	}

	bigInt = bigInt.Div(bigInt, fixedpoint.Fix64ToFix128FactorAsBigInt)
	return NewFix64ValueFromBigInt(memoryGauge, bigInt)
}

func fix64SaturationArithmaticResult(result fix.Fix64, err error) fix.Fix64 {
	switch err.(type) {
	case nil:
		return result
	case fix.PositiveOverflowError:
		return fix.Fix64Max
	case fix.NegativeOverflowError:
		return fix.Fix64Min
	default:
		panic(err)
	}
}
