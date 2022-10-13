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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

type IntSmallValue struct {
	// _small is the value when it is representable as an int32 (!).
	// This ensures that arithmetic operations do not overflow.
	_small int64
}

var _ Value = IntSmallValue{}
var _ atree.Storable = IntSmallValue{}
var _ NumberValue = IntSmallValue{}
var _ IntegerValue = IntSmallValue{}
var _ EquatableValue = IntSmallValue{}
var _ HashableValue = IntSmallValue{}
var _ MemberAccessibleValue = IntSmallValue{}

func (IntSmallValue) IsValue() {}

func (v IntSmallValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitIntSmallValue(interpreter, v)
}

func (IntSmallValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (IntSmallValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt)
}

func (IntSmallValue) IsImportable(_ *Interpreter) bool {
	return true
}

func (v IntSmallValue) ToInt() int {
	return int(v._small)
}

func (v IntSmallValue) ByteLength() int {
	return int64Size
}

func (v IntSmallValue) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, int64BigIntMemoryUsage)
	return new(big.Int).SetInt64(v._small)
}

func (v IntSmallValue) String() string {
	return format.Int(v._small)
}

func (v IntSmallValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v IntSmallValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v IntSmallValue) Negate(interpreter *Interpreter) NumberValue {
	return NewIntValueFromInt64(interpreter, -v._small)
}

func (v IntSmallValue) Plus(interpreter *Interpreter, other NumberValue) NumberValue {

	switch o := other.(type) {
	case IntSmallValue:
		return NewIntValueFromInt64(interpreter, v._small+o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewPlusBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Add(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) SaturatingPlus(interpreter *Interpreter, other NumberValue) NumberValue {
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

func (v IntSmallValue) Minus(interpreter *Interpreter, other NumberValue) NumberValue {

	switch o := other.(type) {
	case IntSmallValue:
		return NewIntValueFromInt64(interpreter, v._small-o._small)
	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewMinusBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Sub(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) SaturatingMinus(interpreter *Interpreter, other NumberValue) NumberValue {
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

func (v IntSmallValue) Mod(interpreter *Interpreter, other NumberValue) NumberValue {

	switch o := other.(type) {
	case IntSmallValue:
		oSmall := o._small
		// INT33-C
		if oSmall == 0 {
			panic(DivisionByZeroError{})
		}
		return NewIntValueFromInt64(interpreter, v._small%oSmall)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewModBigIntMemoryUsage(vBig, oBig))
		res := new(big.Int)
		// INT33-C
		if oBig.Cmp(res) == 0 {
			panic(DivisionByZeroError{})
		}
		return NewUnmeteredIntValueFromBigInt(
			res.Rem(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) Mul(interpreter *Interpreter, other NumberValue) NumberValue {

	switch o := other.(type) {
	case IntSmallValue:
		return NewIntValueFromInt64(interpreter, v._small*o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewMulBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Mul(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) SaturatingMul(interpreter *Interpreter, other NumberValue) NumberValue {
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

func (v IntSmallValue) Div(interpreter *Interpreter, other NumberValue) NumberValue {
	switch o := other.(type) {
	case IntSmallValue:
		oSmall := o._small
		// INT33-C
		if oSmall == 0 {
			panic(DivisionByZeroError{})
		}
		return NewIntValueFromInt64(interpreter, v._small/oSmall)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewDivBigIntMemoryUsage(vBig, oBig))
		res := new(big.Int)
		// INT33-C
		if oBig.Cmp(res) == 0 {
			panic(DivisionByZeroError{})
		}
		return NewUnmeteredIntValueFromBigInt(
			res.Div(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) SaturatingDiv(interpreter *Interpreter, other NumberValue) NumberValue {
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

func (v IntSmallValue) Less(interpreter *Interpreter, other NumberValue) BoolValue {

	switch o := other.(type) {
	case IntSmallValue:
		return AsBoolValue(v._small < o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		cmp := vBig.Cmp(oBig)
		return AsBoolValue(cmp == -1)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) LessEqual(interpreter *Interpreter, other NumberValue) BoolValue {

	switch o := other.(type) {
	case IntSmallValue:
		return AsBoolValue(v._small <= o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		cmp := vBig.Cmp(oBig)
		return AsBoolValue(cmp <= 0)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) Greater(interpreter *Interpreter, other NumberValue) BoolValue {

	switch o := other.(type) {
	case IntSmallValue:
		return AsBoolValue(v._small > o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		cmp := vBig.Cmp(oBig)
		return AsBoolValue(cmp == 1)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) GreaterEqual(interpreter *Interpreter, other NumberValue) BoolValue {

	switch o := other.(type) {
	case IntSmallValue:
		return AsBoolValue(v._small >= o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int for smaller other side
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		cmp := vBig.Cmp(oBig)
		return AsBoolValue(cmp >= 0)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) Equal(interpreter *Interpreter, _ LocationRange, other Value) bool {

	switch o := other.(type) {
	case IntSmallValue:
		return v._small == o._small

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		cmp := vBig.Cmp(oBig)
		return cmp == 0

	default:
		return false
	}
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt (1 byte)
// - big int encoded in big-endian (n bytes)
func (v IntSmallValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
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

func (v IntSmallValue) BitwiseOr(interpreter *Interpreter, other IntegerValue) IntegerValue {

	switch o := other.(type) {
	case IntSmallValue:
		return NewIntValueFromInt64(interpreter, v._small|o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewBitwiseOrBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Or(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) BitwiseXor(interpreter *Interpreter, other IntegerValue) IntegerValue {

	switch o := other.(type) {
	case IntSmallValue:
		return NewIntValueFromInt64(interpreter, v._small^o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewBitwiseXorBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Xor(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) BitwiseAnd(interpreter *Interpreter, other IntegerValue) IntegerValue {

	switch o := other.(type) {
	case IntSmallValue:
		return NewIntValueFromInt64(interpreter, v._small&o._small)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(v._small)

		common.UseMemory(interpreter, common.NewBitwiseAndBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).And(vBig, oBig),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue) IntegerValue {
	vSmall := v._small

	switch o := other.(type) {
	case IntSmallValue:
		oSmall := o._small

		if oSmall < 0 {
			panic(UnderflowError{})
		}

		return NewIntValueFromInt64(interpreter, vSmall<<oSmall)

	case IntBigValue:
		oBig := o._big

		if oBig.Sign() < 0 {
			panic(UnderflowError{})
		}

		if !oBig.IsUint64() {
			panic(OverflowError{})
		}
		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(vSmall)

		common.UseMemory(interpreter, common.NewBitwiseLeftShiftBigIntMemoryUsage(vBig, oBig))
		return NewUnmeteredIntValueFromBigInt(
			new(big.Int).Lsh(vBig, uint(oBig.Uint64())),
		)

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) BitwiseRightShift(interpreter *Interpreter, other IntegerValue) IntegerValue {
	vSmall := v._small

	switch o := other.(type) {
	case IntSmallValue:
		oSmall := o._small

		if oSmall < 0 {
			panic(UnderflowError{})
		}

		return NewIntValueFromInt64(interpreter, vSmall>>oSmall)

	case IntBigValue:
		oBig := o._big

		// TODO: optimize: avoid allocation of big.Int
		common.UseMemory(interpreter, int64BigIntMemoryUsage)
		vBig := new(big.Int).SetInt64(vSmall)

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

	default:
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}
}

func (v IntSmallValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.IntType)
}

func (IntSmallValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (IntSmallValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v IntSmallValue) ToBigEndianBytes() []byte {
	_small := v._small

	switch {
	case math.MinInt8 <= _small && _small <= math.MaxInt8:
		return Int8Value(_small).ToBigEndianBytes()
	case math.MinInt16 <= _small && _small <= math.MaxInt16:
		return Int16Value(_small).ToBigEndianBytes()
	// should always be the case, but perform sanity check
	case math.MinInt32 <= _small && _small <= math.MaxInt32:
		return Int32Value(_small).ToBigEndianBytes()
	default:
		panic(errors.NewUnexpectedError("Int._small outside of int32 range"))
	}
}

func (v IntSmallValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v IntSmallValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (IntSmallValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (IntSmallValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v IntSmallValue) Transfer(
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

func (v IntSmallValue) Clone(_ *Interpreter) Value {
	return IntSmallValue{_small: v._small}
}

func (IntSmallValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v IntSmallValue) ByteSize() uint32 {
	// TODO: optimize: avoid allocation of big.Int
	// TODO: meter, but no gauge available
	_big := new(big.Int).SetInt64(v._small)
	return cborTagSize + getBigIntCBORSize(_big)
}

func (v IntSmallValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (IntSmallValue) ChildStorables() []atree.Storable {
	return nil
}
