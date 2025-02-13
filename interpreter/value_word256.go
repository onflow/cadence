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

// Word256Value

type Word256Value struct {
	BigInt *big.Int
}

func NewWord256ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Word256Value {
	return NewWord256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Word256MemoryUsage = common.NewBigIntMemoryUsage(32)

func NewWord256ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Word256Value {
	common.UseMemory(memoryGauge, Word256MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredWord256ValueFromBigInt(value)
}

func NewUnmeteredWord256ValueFromUint64(value uint64) Word256Value {
	return NewUnmeteredWord256ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUnmeteredWord256ValueFromBigInt(value *big.Int) Word256Value {
	return Word256Value{
		BigInt: value,
	}
}

var _ Value = Word256Value{}
var _ atree.Storable = Word256Value{}
var _ NumberValue = Word256Value{}
var _ IntegerValue = Word256Value{}
var _ EquatableValue = Word256Value{}
var _ ComparableValue = Word256Value{}
var _ HashableValue = Word256Value{}
var _ MemberAccessibleValue = Word256Value{}

func (Word256Value) isValue() {}

func (v Word256Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitWord256Value(interpreter, v)
}

func (Word256Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Word256Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeWord256)
}

func (Word256Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v Word256Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Word256Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Word256Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Word256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Word256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word256Value) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
		),
	)
	return v.String()
}

func (v Word256Value) Negate(NumberValueArithmeticContext, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and wrap around in case of overflow.
			//
			// Note that since v and o are both in the range [0, 2**256 - 1),
			// their sum will be in range [0, 2*(2**256 - 1)).
			// Hence it is sufficient to subtract 2**256 to wrap around.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.Word256TypeMaxIntBig) > 0 {
				sum.Sub(sum, sema.Word256TypeMaxIntPlusOneBig)
			}
			return sum
		},
	)
}

func (v Word256Value) SaturatingPlus(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and wrap around in case of underflow.
			//
			// Note that since v and o are both in the range [0, 2**256 - 1),
			// their difference will be in range [-(2**256 - 1), 2**256 - 1).
			// Hence it is sufficient to add 2**256 to wrap around.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Sign() < 0 {
				diff.Add(diff, sema.Word256TypeMaxIntPlusOneBig)
			}
			return diff
		},
	)
}

func (v Word256Value) SaturatingMinus(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
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

func (v Word256Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.Word256TypeMaxIntBig) > 0 {
				res.Mod(res, sema.Word256TypeMaxIntPlusOneBig)
			}
			return res
		},
	)
}

func (v Word256Value) SaturatingMul(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
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

func (v Word256Value) SaturatingDiv(_ NumberValueArithmeticContext, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
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

func (v Word256Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
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

func (v Word256Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
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

func (v Word256Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
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

func (v Word256Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Word256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord256 (1 byte)
// - big int encoded in big endian (n bytes)
func (v Word256Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeWord256)
	copy(buffer[1:], b)
	return buffer
}

func ConvertWord256(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
	return NewWord256ValueFromBigInt(
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

			if v.Cmp(sema.Word256TypeMaxIntBig) > 0 || v.Sign() < 0 {
				// When Sign() < 0, Mod will add sema.Word256TypeMaxIntPlusOneBig
				// to ensure the range is [0, sema.Word256TypeMaxIntPlusOneBig)
				v.Mod(v, sema.Word256TypeMaxIntPlusOneBig)
			}

			return v
		},
	)
}

func (v Word256Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)

}

func (v Word256Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
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
		return NewWord256ValueFromUint64(context, 0)
	}

	// The maximum shift value at this point is 255, which may lead to an
	// additional allocation of up to 256 bits. Add usage for possible
	// intermediate value.
	common.UseMemory(context, Uint256MemoryUsage)

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			res = res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
			return truncate(res, 256/bits.UintSize)
		},
	)
}

func (v Word256Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
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
		return NewWord256ValueFromUint64(context, 0)
	}

	return NewWord256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word256Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word256Type, locationRange)
}

func (Word256Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word256Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word256Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Word256Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word256Value) IsStorable() bool {
	return true
}

func (v Word256Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word256Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word256Value) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v Word256Value) Transfer(
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

func (v Word256Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredWord256ValueFromBigInt(v.BigInt)
}

func (Word256Value) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v Word256Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Word256Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word256Value) ChildStorables() []atree.Storable {
	return nil
}
