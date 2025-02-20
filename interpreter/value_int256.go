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
	"math/big"
	"math/bits"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// Int256Value

type Int256Value struct {
	BigInt *big.Int
}

func NewInt256ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Int256Value {
	return NewInt256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Int256MemoryUsage = common.NewBigIntMemoryUsage(32)

func NewInt256ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Int256Value {
	common.UseMemory(memoryGauge, Int256MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredInt256ValueFromBigInt(value)
}

func NewUnmeteredInt256ValueFromInt64(value int64) Int256Value {
	return NewUnmeteredInt256ValueFromBigInt(big.NewInt(value))
}

func NewUnmeteredInt256ValueFromBigInt(value *big.Int) Int256Value {
	return Int256Value{
		BigInt: value,
	}
}

var _ Value = Int256Value{}
var _ atree.Storable = Int256Value{}
var _ NumberValue = Int256Value{}
var _ IntegerValue = Int256Value{}
var _ EquatableValue = Int256Value{}
var _ ComparableValue = Int256Value{}
var _ HashableValue = Int256Value{}
var _ MemberAccessibleValue = Int256Value{}

func (Int256Value) isValue() {}

func (v Int256Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitInt256Value(interpreter, v)
}

func (Int256Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Int256Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeInt256)
}

func (Int256Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v Int256Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Int256Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Int256Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Int256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int256Value) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
		),
	)
	return v.String()
}

func (v Int256Value) Negate(context NumberValueArithmeticContext, locationRange LocationRange) NumberValue {
	// INT32-C
	//   if v == Int256TypeMinIntBig {
	//       ...
	//   }
	if v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		return new(big.Int).Neg(v.BigInt)
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int256TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int256TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int256TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int256TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			return sema.Int256TypeMinIntBig
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			return sema.Int256TypeMaxIntBig
		}

		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int256TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int256TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int256TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int256TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			return sema.Int256TypeMinIntBig
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			return sema.Int256TypeMaxIntBig
		}

		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.Rem(v.BigInt, o.BigInt)

		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			return sema.Int256TypeMinIntBig
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			return sema.Int256TypeMaxIntBig
		}

		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int256TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		res.Div(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int256TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			return sema.Int256TypeMaxIntBig
		}
		res.Div(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Int256Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Int256Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Int256Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Int256Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Int256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt256 (1 byte)
// - big int value encoded in big-endian (n bytes)
func (v Int256Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	b := SignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeInt256)
	copy(buffer[1:], b)
	return buffer
}

func ConvertInt256(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int256Value {
	converter := func() *big.Int {
		var v *big.Int

		switch value := value.(type) {
		case BigNumberValue:
			v = value.ToBigInt(memoryGauge)

		case NumberValue:
			v = big.NewInt(int64(value.ToInt(locationRange)))

		default:
			panic(errors.NewUnreachableError())
		}

		if v.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		} else if v.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}

		return v
	}

	return NewInt256ValueFromBigInt(memoryGauge, converter)
}

func (v Int256Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Or(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Xor(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.And(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(NegativeShiftError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() || o.BigInt.Uint64() >= 256 {
		return NewInt256ValueFromUint64(context, 0)
	}

	// The maximum shift value at this point is 255, which may lead to an
	// additional allocation of up to 256 bits. Add usage for possible
	// intermediate value.
	common.UseMemory(context, Int256MemoryUsage)

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res = toTwosComplement(res, v.BigInt, 256)
		res = res.Lsh(res, uint(o.BigInt.Uint64()))
		res = truncate(res, 256/bits.UintSize)
		return fromTwosComplement(res)
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(NegativeShiftError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		return NewInt256ValueFromUint64(context, 0)
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		return res
	}

	return NewInt256ValueFromBigInt(context, valueGetter)
}

func (v Int256Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int256Type, locationRange)
}

func (Int256Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int256Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int256Value) ToBigEndianBytes() []byte {
	return SignedBigIntToSizedBigEndianBytes(v.BigInt, sema.Int256TypeSize)
}

func (v Int256Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int256Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int256Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int256Value) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v Int256Value) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v Int256Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredInt256ValueFromBigInt(v.BigInt)
}

func (Int256Value) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v Int256Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Int256Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int256Value) ChildStorables() []atree.Storable {
	return nil
}
