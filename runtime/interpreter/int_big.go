/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

type IntBigValue struct {
	_big *big.Int
}

const int64Size = int(unsafe.Sizeof(int64(0)))

var int64BigIntMemoryUsage = common.NewBigIntMemoryUsage(int64Size)

func NewIntValueFromInt64(memoryGauge common.MemoryGauge, value int64) IntegerValue {
	common.UseMemory(memoryGauge, int64BigIntMemoryUsage)
	return NewUnmeteredIntValueFromInt64(value)
}

func NewUnmeteredIntValueFromInt64(value int64) IntegerValue {
	if math.MinInt32 <= value && value <= math.MaxInt32 {
		return IntSmallValue{_small: value}
	}
	return IntBigValue{_big: big.NewInt(value)}
}

func NewIntValueFromBigInt(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) IntegerValue {
	common.UseMemory(memoryGauge, memoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredIntValueFromBigInt(value)
}

func NewUnmeteredIntValueFromBigInt(value *big.Int) IntegerValue {
	if value.IsInt64() {
		return NewUnmeteredIntValueFromInt64(value.Int64())
	}
	return IntBigValue{_big: value}
}

func ConvertInt(memoryGauge common.MemoryGauge, value Value) IntegerValue {
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

var _ Value = IntBigValue{}
var _ atree.Storable = IntBigValue{}
var _ NumberValue = IntBigValue{}
var _ IntegerValue = IntBigValue{}
var _ EquatableValue = IntBigValue{}
var _ HashableValue = IntBigValue{}
var _ MemberAccessibleValue = IntBigValue{}

func (IntBigValue) IsValue() {}

func (v IntBigValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitIntBigValue(interpreter, v)
}

func (IntBigValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (IntBigValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt)
}

func (IntBigValue) IsImportable(_ *Interpreter) bool {
	return true
}

func (v IntBigValue) ToInt() int {
	_big := v._big
	if !_big.IsInt64() {
		panic(OverflowError{})
	}
	return int(_big.Int64())
}

func (v IntBigValue) ByteLength() int {
	_big := v._big
	return common.BigIntByteLength(_big)
}

func (v IntBigValue) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	_big := v._big
	return new(big.Int).Set(_big)
}

func (v IntBigValue) String() string {
	_big := v._big
	return format.BigInt(_big)
}

func (v IntBigValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v IntBigValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v IntBigValue) Negate(interpreter *Interpreter) NumberValue {
	_big := v._big
	common.UseMemory(interpreter, common.NewNegateBigIntMemoryUsage(_big))
	return NewUnmeteredIntValueFromBigInt(new(big.Int).Neg(_big))
}

func (v IntBigValue) Plus(interpreter *Interpreter, other NumberValue) NumberValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewPlusBigIntMemoryUsage(vBig, oBig))
	return NewUnmeteredIntValueFromBigInt(
		new(big.Int).Add(vBig, oBig),
	)
}

func (v IntBigValue) SaturatingPlus(interpreter *Interpreter, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingAddFunctionName,
				LeftType:     v.StaticType(interpreter),
				RightType:    other.StaticType(interpreter),
			})
		}
	}()

	return v.Plus(interpreter, other)
}

func (v IntBigValue) Minus(interpreter *Interpreter, other NumberValue) NumberValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewMinusBigIntMemoryUsage(vBig, oBig))
	return NewUnmeteredIntValueFromBigInt(
		new(big.Int).Sub(vBig, oBig),
	)
}

func (v IntBigValue) SaturatingMinus(interpreter *Interpreter, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
				LeftType:     v.StaticType(interpreter),
				RightType:    other.StaticType(interpreter),
			})
		}
	}()

	return v.Minus(interpreter, other)
}

func (v IntBigValue) Mod(interpreter *Interpreter, other NumberValue) NumberValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewModBigIntMemoryUsage(vBig, oBig))
	res := new(big.Int)
	// INT33-C
	if oBig.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	return NewUnmeteredIntValueFromBigInt(
		res.Rem(vBig, oBig),
	)
}

func (v IntBigValue) Mul(interpreter *Interpreter, other NumberValue) NumberValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewMulBigIntMemoryUsage(vBig, oBig))
	return NewUnmeteredIntValueFromBigInt(
		new(big.Int).Mul(vBig, oBig),
	)
}

func (v IntBigValue) SaturatingMul(interpreter *Interpreter, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
				LeftType:     v.StaticType(interpreter),
				RightType:    other.StaticType(interpreter),
			})
		}
	}()

	return v.Mul(interpreter, other)
}

func (v IntBigValue) Div(interpreter *Interpreter, other NumberValue) NumberValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewDivBigIntMemoryUsage(vBig, oBig))
	res := new(big.Int)
	// INT33-C
	if oBig.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	return NewUnmeteredIntValueFromBigInt(
		res.Div(vBig, oBig),
	)
}

func (v IntBigValue) SaturatingDiv(interpreter *Interpreter, other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(interpreter),
				RightType:    other.StaticType(interpreter),
			})
		}
	}()

	return v.Div(interpreter, other)
}

func (v IntBigValue) Less(interpreter *Interpreter, other NumberValue) BoolValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	cmp := vBig.Cmp(oBig)
	return AsBoolValue(cmp == -1)
}

func (v IntBigValue) LessEqual(interpreter *Interpreter, other NumberValue) BoolValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	cmp := vBig.Cmp(oBig)
	return AsBoolValue(cmp <= 0)
}

func (v IntBigValue) Greater(interpreter *Interpreter, other NumberValue) BoolValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	cmp := vBig.Cmp(oBig)
	return AsBoolValue(cmp == 1)

}

func (v IntBigValue) GreaterEqual(interpreter *Interpreter, other NumberValue) BoolValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	cmp := vBig.Cmp(oBig)
	return AsBoolValue(cmp >= 0)
}

func (v IntBigValue) Equal(interpreter *Interpreter, _ LocationRange, other Value) bool {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		return false
	}

	cmp := vBig.Cmp(oBig)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt (1 byte)
// - big int encoded in big-endian (n bytes)
func (v IntBigValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	// TODO: maybe don't delegate for hash value implementation,
	// but duplicate behaviour to prevent accidental break
	b := v.ToBigEndianBytes()

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

func (v IntBigValue) BitwiseOr(interpreter *Interpreter, other IntegerValue) IntegerValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewBitwiseOrBigIntMemoryUsage(vBig, oBig))
	return NewUnmeteredIntValueFromBigInt(
		new(big.Int).Or(vBig, oBig),
	)
}

func (v IntBigValue) BitwiseXor(interpreter *Interpreter, other IntegerValue) IntegerValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewBitwiseXorBigIntMemoryUsage(vBig, oBig))
	return NewUnmeteredIntValueFromBigInt(
		new(big.Int).Xor(vBig, oBig),
	)
}

func (v IntBigValue) BitwiseAnd(interpreter *Interpreter, other IntegerValue) IntegerValue {
	vBig := v._big
	var oBig *big.Int

	switch other := other.(type) {
	case IntBigValue:
		oBig = other._big

	case IntSmallValue:
		// TODO: optimize: avoid allocation
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		oBig = new(big.Int).SetInt64(other._small)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	common.UseMemory(interpreter, common.NewBitwiseAndBigIntMemoryUsage(vBig, oBig))
	return NewUnmeteredIntValueFromBigInt(
		new(big.Int).And(vBig, oBig),
	)
}

func (v IntBigValue) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue) IntegerValue {
	vBig := v._big

	switch other := other.(type) {
	case IntBigValue:
		oBig := other._big
		if oBig.Sign() < 0 {
			panic(UnderflowError{})
		}

		if !oBig.IsUint64() {
			panic(OverflowError{})
		}

		common.UseMemory(interpreter, common.NewBitwiseLeftShiftBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Lsh(vBig, uint(oBig.Uint64())),
		)

	case IntSmallValue:
		oSmall := other._small
		if oSmall < 0 {
			panic(UnderflowError{})
		}

		// TODO: common.UseMemory(interpreter, common.NewBitwiseLeftShiftBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Lsh(vBig, uint(oSmall)),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntBigValue) BitwiseRightShift(interpreter *Interpreter, other IntegerValue) IntegerValue {
	vBig := v._big

	switch other := other.(type) {
	case IntBigValue:
		oBig := other._big
		if oBig.Sign() < 0 {
			panic(UnderflowError{})
		}

		if !oBig.IsUint64() {
			panic(OverflowError{})
		}

		common.UseMemory(interpreter, common.NewBitwiseLeftShiftBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Rsh(vBig, uint(oBig.Uint64())),
		)

	case IntSmallValue:
		oSmall := other._small
		if oSmall < 0 {
			panic(UnderflowError{})
		}

		// TODO: common.UseMemory(interpreter, common.NewBitwiseRightShiftBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Rsh(vBig, uint(oSmall)),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntBigValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.IntType)
}

func (IntBigValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (IntBigValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v IntBigValue) ToBigEndianBytes() []byte {
	_big := v._big
	return SignedBigIntToBigEndianBytes(_big)
}

func (v IntBigValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v IntBigValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (IntBigValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (IntBigValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v IntBigValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v IntBigValue) Clone(_ *Interpreter) Value {
	_big := v._big
	return IntBigValue{_big: new(big.Int).Set(_big)}
}

func (IntBigValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v IntBigValue) ByteSize() uint32 {
	_big := v._big
	return cborTagSize + getBigIntCBORSize(_big)
}

func (v IntBigValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (IntBigValue) ChildStorables() []atree.Storable {
	return nil
}
