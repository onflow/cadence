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
	"github.com/onflow/cadence/values"
)

// Word128Value

type Word128Value struct {
	BigInt *big.Int
}

func NewWord128ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Word128Value {
	return NewWord128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Word128MemoryUsage = common.NewBigIntMemoryUsage(16)

func NewWord128ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Word128Value {
	common.UseMemory(memoryGauge, Word128MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredWord128ValueFromBigInt(value)
}

func NewUnmeteredWord128ValueFromUint64(value uint64) Word128Value {
	return NewUnmeteredWord128ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUnmeteredWord128ValueFromBigInt(value *big.Int) Word128Value {
	return Word128Value{
		BigInt: value,
	}
}

var _ Value = Word128Value{}
var _ atree.Storable = Word128Value{}
var _ NumberValue = Word128Value{}
var _ IntegerValue = Word128Value{}
var _ EquatableValue = Word128Value{}
var _ ComparableValue = Word128Value{}
var _ HashableValue = Word128Value{}
var _ MemberAccessibleValue = Word128Value{}

func (Word128Value) IsValue() {}

func (v Word128Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitWord128Value(interpreter, v)
}

func (Word128Value) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Word128Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeWord128)
}

func (Word128Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v Word128Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Word128Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Word128Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Word128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Word128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word128Value) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		context,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(context, v),
		),
	)
	return v.String()
}

func (v Word128Value) Negate(NumberValueArithmeticContext, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and wrap around in case of overflow.
			//
			// Note that since v and o are both in the range [0, 2**128 - 1),
			// their sum will be in range [0, 2*(2**128 - 1)).
			// Hence it is sufficient to subtract 2**128 to wrap around.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.Word128TypeMaxIntBig) > 0 {
				sum.Sub(sum, sema.Word128TypeMaxIntPlusOneBig)
			}
			return sum
		},
	)
}

func (v Word128Value) SaturatingPlus(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and wrap around in case of underflow.
			//
			// Note that since v and o are both in the range [0, 2**128 - 1),
			// their difference will be in range [-(2**128 - 1), 2**128 - 1).
			// Hence it is sufficient to add 2**128 to wrap around.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Sign() < 0 {
				diff.Add(diff, sema.Word128TypeMaxIntPlusOneBig)
			}
			return diff
		},
	)
}

func (v Word128Value) SaturatingMinus(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.Word128TypeMaxIntBig) > 0 {
				res.Mod(res, sema.Word128TypeMaxIntPlusOneBig)
			}
			return res
		},
	)
}

func (v Word128Value) SaturatingMul(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)

}

func (v Word128Value) SaturatingDiv(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v Word128Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v Word128Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v Word128Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v Word128Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Word128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord128 (1 byte)
// - big int encoded in big endian (n bytes)
func (v Word128Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	b := values.UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeWord128)
	copy(buffer[1:], b)
	return buffer
}

func ConvertWord128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
	return NewWord128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {

			var v *big.Int

			switch value := value.(type) {
			case BigNumberValue:
				v = value.ToBigInt(memoryGauge)

			case NumberValue:
				v = big.NewInt(int64(value.ToInt(locationRange)))

			default:
				panic(errors.NewUnreachableError())
			}

			if v.Cmp(sema.Word128TypeMaxIntBig) > 0 || v.Sign() < 0 {
				// When Sign() < 0, Mod will add sema.Word128TypeMaxIntPlusOneBig
				// to ensure the range is [0, sema.Word128TypeMaxIntPlusOneBig)
				v.Mod(v, sema.Word128TypeMaxIntPlusOneBig)
			}

			return v
		},
	)
}

func (v Word128Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)

}

func (v Word128Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
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
	if !o.BigInt.IsUint64() || o.BigInt.Uint64() >= 128 {
		return NewWord128ValueFromUint64(context, 0)
	}

	// The maximum shift value at this point is 127, which may lead to an
	// additional allocation of up to 128 bits. Add usage for possible
	// intermediate value.
	common.UseMemory(context, Uint128MemoryUsage)

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			res = res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
			return truncate(res, 128/bits.UintSize)
		},
	)
}

func (v Word128Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(context),
			RightType: other.StaticType(context),
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(NegativeShiftError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		return NewWord128ValueFromUint64(context, 0)
	}

	return NewWord128ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word128Value) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return getNumberValueMember(context, v, name, sema.Word128Type, locationRange)
}

func (Word128Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word128Value) SetMember(_ MemberAccessibleContext, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word128Value) ToBigEndianBytes() []byte {
	return values.UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Word128Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word128Value) IsStorable() bool {
	return true
}

func (v Word128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word128Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v Word128Value) Transfer(
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

func (v Word128Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredWord128ValueFromBigInt(v.BigInt)
}

func (Word128Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v Word128Value) ByteSize() uint32 {
	return values.CBORTagSize + values.GetBigIntCBORSize(v.BigInt)
}

func (v Word128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word128Value) ChildStorables() []atree.Storable {
	return nil
}
