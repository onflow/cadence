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

// UFix64Value
type UFix64Value fix.UFix64

const ufix64Size = int(unsafe.Sizeof(UFix64Value(0)))

var UFix64MemoryUsage = common.NewNumberMemoryUsage(ufix64Size)

// Note that this function uses the default scaling of 8.
func NewUnmeteredUFix64ValueWithInteger(integer uint64) UFix64Value {
	bigInt := new(big.Int).SetUint64(integer)
	bigInt = new(big.Int).Mul(
		bigInt,
		sema.UFix64FactorIntBig,
	)

	return NewUFix64ValueFromBigIntWithRangeCheck(nil, bigInt)
}

func NewUnmeteredUFix64ValueWithIntegerAndScale(integer uint64, scale int64) UFix64Value {
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

	return NewUFix64ValueFromBigInt(nil, bigInt)
}

func NewUFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() uint64) UFix64Value {
	common.UseMemory(gauge, UFix64MemoryUsage)
	return NewUnmeteredUFix64ValueWithInteger(constructor())
}

func NewUFix64Value(gauge common.MemoryGauge, valueGetter func() fix.UFix64) UFix64Value {
	common.UseMemory(gauge, UFix64MemoryUsage)
	return NewUnmeteredUFix64Value(valueGetter())
}

func NewUnmeteredUFix64Value(ufix64 fix.UFix64) UFix64Value {
	return UFix64Value(ufix64)
}

func NewUFix64ValueFromBigEndianBytes(gauge common.MemoryGauge, b []byte) Value {
	return NewUFix64Value(
		gauge,
		func() fix.UFix64 {
			bytes := padWithZeroes(b, 8)
			val := binary.BigEndian.Uint64(bytes)
			return fix.UFix64(val)
		},
	)
}

func NewUFix64ValueFromBigInt(gauge common.MemoryGauge, v *big.Int) UFix64Value {
	return NewUFix64Value(
		gauge,
		func() fix.UFix64 {
			return fixedpoint.UFix64FromBigInt(v)
		},
	)
}

func NewUFix64ValueFromBigIntWithRangeCheck(gauge common.MemoryGauge, v *big.Int) UFix64Value {
	if v.Sign() < 0 {
		panic(&UnderflowError{})
	}

	if v.Cmp(fixedpoint.UFix64TypeMaxIntBig) > 0 {
		panic(&OverflowError{})
	}

	return NewUFix64ValueFromBigInt(gauge, v)
}

func ConvertUFix64(memoryGauge common.MemoryGauge, value Value) UFix64Value {
	scaledInt := new(big.Int)

	switch value := value.(type) {

	case Fix64Value:
		bigInt := value.ToBigInt()
		scaledInt = bigInt

	case UFix64Value:
		return value

	case Fix128Value:
		scaledInt = value.ToBigInt()

	case UFix128Value:
		scaledInt = value.ToBigInt()

	case BigNumberValue:
		bigInt := value.ToBigInt(memoryGauge)
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.UFix64FactorAsBigInt,
		)

	case NumberValue:
		bigInt := new(big.Int).SetInt64(int64(value.ToInt()))
		scaledInt = scaledInt.Mul(
			bigInt,
			fixedpoint.UFix64FactorAsBigInt,
		)

	default:
		panic(fmt.Sprintf("can't convert to UFix64: %s", value))
	}

	return NewUFix64ValueFromBigIntWithRangeCheck(memoryGauge, scaledInt)
}

var _ Value = UFix64Value(0)
var _ atree.Storable = UFix64Value(0)
var _ NumberValue = UFix64Value(0)
var _ FixedPointValue = UFix64Value(0)
var _ EquatableValue = UFix64Value(0)
var _ ComparableValue = UFix64Value(0)
var _ HashableValue = UFix64Value(0)
var _ MemberAccessibleValue = UFix64Value(0)

func (UFix64Value) IsValue() {}

func (v UFix64Value) Accept(context ValueVisitContext, visitor Visitor) {
	visitor.VisitUFix64Value(context, v)
}

func (UFix64Value) Walk(_ ValueWalkContext, _ func(Value)) {
	// NO-OP
}

func (UFix64Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeUFix64)
}

func (UFix64Value) IsImportable(_ ValueImportableContext) bool {
	return true
}

func (v UFix64Value) String() string {
	return format.UFix64(uint64(v))
}

func (v UFix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UFix64Value) MeteredString(
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

func (v UFix64Value) ToInt() int {
	// TODO: Maybe compute this without the use of `big.Int`
	ufix64BigInt := v.ToBigInt()
	integerPart := ufix64BigInt.Div(ufix64BigInt, sema.Fix64FactorBig)

	if !integerPart.IsInt64() {
		panic(&OverflowError{})
	}

	return int(integerPart.Int64())
}

func (v UFix64Value) Negate(NumberValueArithmeticContext) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Plus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Add(fix.UFix64(o))
		handleFixedpointError(err)
		return result
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Add(fix.UFix64(o))
		return ufix64SaturationArithmaticResult(result, err)
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) Minus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Sub(fix.UFix64(o))
		handleFixedpointError(err)
		return result
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Sub(fix.UFix64(o))
		return ufix64SaturationArithmaticResult(result, err)
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) Mul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Mul(
			fix.UFix64(o),
			fix.RoundTruncate,
		)
		handleFixedpointError(err)
		return result
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(context),
			RightType:    other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Mul(
			fix.UFix64(o),
			fix.RoundTruncate,
		)
		return ufix64SaturationArithmaticResult(result, err)
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) Div(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Div(
			fix.UFix64(o),
			fix.RoundTruncate,
		)
		handleFixedpointError(err)
		return result
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) SaturatingDiv(_ NumberValueArithmeticContext, _ NumberValue) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Mod(context NumberValueArithmeticContext, other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	valueGetter := func() fix.UFix64 {
		result, err := fix.UFix64(v).Mod(fix.UFix64(o))
		handleFixedpointError(err)
		return result
	}

	return NewUFix64Value(context, valueGetter)
}

func (v UFix64Value) Less(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.UFix64(v)
	that := fix.UFix64(o)

	return BoolValue(this.Lt(that))
}

func (v UFix64Value) LessEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.UFix64(v)
	that := fix.UFix64(o)

	return BoolValue(this.Lte(that))
}

func (v UFix64Value) Greater(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.UFix64(v)
	that := fix.UFix64(o)

	return BoolValue(this.Gt(that))
}

func (v UFix64Value) GreaterEqual(context ValueComparisonContext, other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	this := fix.UFix64(v)
	that := fix.UFix64(o)

	return BoolValue(this.Gte(that))
}

func (v UFix64Value) Equal(_ ValueComparisonContext, other Value) bool {
	otherUFix64, ok := other.(UFix64Value)
	if !ok {
		return false
	}
	return v == otherUFix64
}

// HashInput returns a byte slice containing:
// - HashInputTypeUFix64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v UFix64Value) HashInput(_ common.MemoryGauge, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(uint64(v)))
	return scratch[:9]
}

func (v UFix64Value) GetMember(context MemberAccessibleContext, name string) Value {
	return context.GetMethod(v, name)
}

func (v UFix64Value) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.UFix64Type)
}

func (UFix64Value) RemoveMember(_ ValueTransferContext, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UFix64Value) SetMember(_ ValueTransferContext, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
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
	// TODO: Maybe compute this without the use of `big.Int`
	ufix64BigInt := v.ToBigInt()
	integerPart := ufix64BigInt.Div(ufix64BigInt, sema.Fix64FactorBig)

	if !integerPart.IsUint64() {
		panic(&OverflowError{})
	}

	return UInt64Value(integerPart.Uint64())
}

func (v UFix64Value) ToBigInt() *big.Int {
	return fixedpoint.UFix64ToBigInt(fix.UFix64(v))
}

func (v UFix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UFix64Value) ByteSize() uint32 {
	return values.CBORTagSize + values.GetUintCBORSize(uint64(v))
}

func (v UFix64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UFix64Value) ChildStorables() []atree.Storable {
	return nil
}

func (UFix64Value) IsStorable() bool {
	return true
}

func (v UFix64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UFix64Value) Scale() int {
	return sema.Fix64Scale
}

// Encode encodes UFix64Value as
//
//	cbor.Tag{
//			Number:  CBORTagUFix64Value,
//			Content: uint64(v),
//	}
func (v UFix64Value) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, values.CBORTagUFix64Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint64(uint64(v))
}

func fix128BigIntToUFix64(
	memoryGauge common.MemoryGauge,
	bigInt *big.Int,
) UFix64Value {

	if bigInt.Cmp(fixedpoint.UFix64TypeMaxScaledTo128) > 0 {
		panic(&OverflowError{})
	} else if bigInt.Cmp(fixedpoint.UFix64TypeMinScaledTo128) < 0 {
		panic(&UnderflowError{})
	}

	bigInt = bigInt.Div(bigInt, fixedpoint.Fix64ToFix128FactorAsBigInt)
	return NewUFix64ValueFromBigInt(memoryGauge, bigInt)
}

func ufix64SaturationArithmaticResult(result fix.UFix64, err error) fix.UFix64 {
	switch err.(type) {
	case nil:
		return result
	case fix.PositiveOverflowError:
		return fix.UFix64Max
	case fix.NegativeOverflowError:
		return fix.UFix64Zero
	default:
		panic(err)
	}
}
