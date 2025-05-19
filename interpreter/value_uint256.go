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

// UInt256Value

type UInt256Value struct {
	BigInt *big.Int
}

func NewUInt256ValueFromUint64(memoryGauge common.MemoryGauge, value uint64) UInt256Value {
	return NewUInt256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetUint64(value)
		},
	)
}

func NewUnmeteredUInt256ValueFromUint64(value uint64) UInt256Value {
	return NewUnmeteredUInt256ValueFromBigInt(new(big.Int).SetUint64(value))
}

var Uint256MemoryUsage = common.NewBigIntMemoryUsage(32)

func NewUInt256ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) UInt256Value {
	common.UseMemory(memoryGauge, Uint256MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredUInt256ValueFromBigInt(value)
}

func NewUnmeteredUInt256ValueFromBigInt(value *big.Int) UInt256Value {
	return UInt256Value{
		BigInt: value,
	}
}

var _ Value = UInt256Value{}
var _ atree.Storable = UInt256Value{}
var _ NumberValue = UInt256Value{}
var _ IntegerValue = UInt256Value{}
var _ EquatableValue = UInt256Value{}
var _ ComparableValue = UInt256Value{}
var _ HashableValue = UInt256Value{}
var _ MemberAccessibleValue = UInt256Value{}

func (UInt256Value) IsValue() {}

func (v UInt256Value) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitUInt256Value(context, v)
}

func (UInt256Value) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (UInt256Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeUInt256)
}

func (UInt256Value) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return true
}

func (v UInt256Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(&OverflowError{
			LocationRange: locationRange,
		})
	}

	return int(v.BigInt.Int64())
}

func (v UInt256Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v UInt256Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v UInt256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v UInt256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt256Value) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		context,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(context, v),
		),
	)
	return v.String()
}

func (v UInt256Value) Negate(NumberValueArithmeticContext, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				panic(&OverflowError{
					LocationRange: locationRange,
				})
			}
			return sum
		},
	)

}

func (v UInt256Value) SaturatingPlus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				return sema.UInt256TypeMaxIntBig
			}
			return sum
		},
	)
}

func (v UInt256Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Cmp(sema.UInt256TypeMinIntBig) < 0 {
				panic(&UnderflowError{
					LocationRange: locationRange,
				})
			}
			return diff
		},
	)
}

func (v UInt256Value) SaturatingMinus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Cmp(sema.UInt256TypeMinIntBig) < 0 {
				return sema.UInt256TypeMinIntBig
			}
			return diff
		},
	)

}

func (v UInt256Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(&DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				panic(&OverflowError{
					LocationRange: locationRange,
				})
			}
			return res
		},
	)
}

func (v UInt256Value) SaturatingMul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				return sema.UInt256TypeMaxIntBig
			}
			return res
		},
	)
}

func (v UInt256Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(&DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) SaturatingDiv(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(*InvalidOperandsError); ok {
			panic(&InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(context),
				RightType:     other.StaticType(context),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(context, other, locationRange)
}

func (v UInt256Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v UInt256Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v UInt256Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v UInt256Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v UInt256Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherInt, ok := other.(UInt256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt256 (1 byte)
// - big int encoded in big endian (n bytes)
func (v UInt256Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	b := values.UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeUInt256)
	copy(buffer[1:], b)
	return buffer
}

func ConvertUInt256(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt256Value {
	return NewUInt256ValueFromBigInt(
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

			if v.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				panic(&OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Sign() < 0 {
				panic(&UnderflowError{
					LocationRange: locationRange,
				})
			}

			return v
		},
	)
}

func (v UInt256Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(&NegativeShiftError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() || o.BigInt.Uint64() >= 256 {
		return NewUInt256ValueFromUint64(context, 0)
	}

	// The maximum shift value at this point is 255, which may lead to an
	// additional allocation of up to 256 bits. Add usage for possible
	// intermediate value.
	common.UseMemory(context, Uint256MemoryUsage)

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			res = res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
			return truncate(res, 256/bits.UintSize)
		},
	)
}

func (v UInt256Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(&InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(&NegativeShiftError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		return NewUInt256ValueFromUint64(context, 0)
	}

	return NewUInt256ValueFromBigInt(
		context,
		func() *big.Int {
			res := new(big.Int)
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v UInt256Value) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return context.GetMethod(v, name, locationRange)
}

func (v UInt256Value) GetMethod(
	context MemberAccessibleContext,
	locationRange LocationRange,
	name string,
) FunctionValue {
	return getNumberValueFunctionMember(context, v, name, sema.UInt256Type, locationRange)
}

func (UInt256Value) RemoveMember(_ ValueTransferContext, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt256Value) SetMember(_ ValueTransferContext, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) ToBigEndianBytes() []byte {
	return values.UnsignedBigIntToSizedBigEndianBytes(v.BigInt, sema.UInt256TypeSize)
}

func (v UInt256Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt256Value) IsStorable() bool {
	return true
}

func (v UInt256Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt256Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt256Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}
func (v UInt256Value) Transfer(
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

func (v UInt256Value) Clone(_ ValueCloneContext) Value {
	return NewUnmeteredUInt256ValueFromBigInt(v.BigInt)
}

func (UInt256Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v UInt256Value) ByteSize() uint32 {
	return values.CBORTagSize + values.GetBigIntCBORSize(v.BigInt)
}

func (v UInt256Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt256Value) ChildStorables() []atree.Storable {
	return nil
}
