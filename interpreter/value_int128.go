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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// toTwosComplement sets `res` to the two's complement representation of a big.Int `x` in the given target bit size.
// `res` is returned and is awlways a positive big.Int.
func toTwosComplement(res, x *big.Int, targetBitSize uint) *big.Int {
	bytes := SignedBigIntToSizedBigEndianBytes(x, targetBitSize/8)
	return res.SetBytes(bytes)
}

// toTwosComplement converts `res` to the big.Int representation from the two's complement format of a
// signed integer.
// `res` is returned and can be positive or negative.
func fromTwosComplement(res *big.Int) *big.Int {
	bytes := res.Bytes()
	return BigEndianBytesToSignedBigInt(bytes)
}

// truncate trims a big.Int to maxWords by directly modifying its underlying representation.
func truncate(x *big.Int, maxWords int) *big.Int {
	// Get the absolute value of x as a nat slice.
	abs := x.Bits()

	// Limit the nat slice to maxWords.
	if len(abs) > maxWords {
		abs = abs[:maxWords]
	}

	// Update the big.Int's internal representation.
	x.SetBits(abs)

	return x
}

// Int128Value

type Int128Value struct {
	BigInt *big.Int
}

func NewInt128ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Int128Value {
	return NewInt128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Int128MemoryUsage = common.NewBigIntMemoryUsage(16)

func NewInt128ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Int128Value {
	common.UseMemory(memoryGauge, Int128MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredInt128ValueFromBigInt(value)
}

func NewUnmeteredInt128ValueFromInt64(value int64) Int128Value {
	return NewUnmeteredInt128ValueFromBigInt(big.NewInt(value))
}

func NewUnmeteredInt128ValueFromBigInt(value *big.Int) Int128Value {
	return Int128Value{
		BigInt: value,
	}
}

var _ Value = Int128Value{}
var _ atree.Storable = Int128Value{}
var _ NumberValue = Int128Value{}
var _ IntegerValue = Int128Value{}
var _ EquatableValue = Int128Value{}
var _ ComparableValue = Int128Value{}
var _ HashableValue = Int128Value{}
var _ MemberAccessibleValue = Int128Value{}

func (Int128Value) isValue() {}

func (v Int128Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitInt128Value(interpreter, v)
}

func (Int128Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Int128Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt128)
}

func (Int128Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v Int128Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Int128Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Int128Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Int128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int128Value) MeteredString(interpreter *Interpreter, _ SeenReferences, locationRange LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
		),
	)
	return v.String()
}

func (v Int128Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	//   if v == Int128TypeMinIntBig {
	//       ...
	//   }
	if v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		return new(big.Int).Neg(v.BigInt)
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int128TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int128TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int128TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int128TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			return sema.Int128TypeMinIntBig
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			return sema.Int128TypeMaxIntBig
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int128TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int128TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int128TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int128TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			return sema.Int128TypeMinIntBig
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			return sema.Int128TypeMaxIntBig
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
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

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			return sema.Int128TypeMinIntBig
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			return sema.Int128TypeMaxIntBig
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int128TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		res.Div(v.BigInt, o.BigInt)

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int128TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			return sema.Int128TypeMaxIntBig
		}
		res.Div(v.BigInt, o.BigInt)

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Int128Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Int128Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Int128Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Int128Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Int128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt128 (1 byte)
// - big int value encoded in big-endian (n bytes)
func (v Int128Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := SignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeInt128)
	copy(buffer[1:], b)
	return buffer
}

func ConvertInt128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int128Value {
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

		if v.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		} else if v.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}

		return v
	}

	return NewInt128ValueFromBigInt(memoryGauge, converter)
}

func (v Int128Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Or(v.BigInt, o.BigInt)
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Xor(v.BigInt, o.BigInt)
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.And(v.BigInt, o.BigInt)
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(NegativeShiftError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() || o.BigInt.Uint64() >= 128 {
		return NewInt128ValueFromUint64(interpreter, 0)
	}

	// The maximum shift value at this point is 127, which may lead to an
	// additional allocation of up to 128 bits. Add usage for possible
	// intermediate value.
	common.UseMemory(interpreter, Int128MemoryUsage)

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res = toTwosComplement(res, v.BigInt, 128)
		res = res.Lsh(res, uint(o.BigInt.Uint64()))
		res = truncate(res, 128/bits.UintSize)
		return fromTwosComplement(res)
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(NegativeShiftError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		return NewInt128ValueFromUint64(interpreter, 0)
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int128Type, locationRange)
}

func (Int128Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int128Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int128Value) ToBigEndianBytes() []byte {
	return SignedBigIntToSizedBigEndianBytes(v.BigInt, sema.Int128TypeSize)
}

func (v Int128Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int128Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int128Value) Transfer(
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

func (v Int128Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredInt128ValueFromBigInt(v.BigInt)
}

func (Int128Value) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v Int128Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Int128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int128Value) ChildStorables() []atree.Storable {
	return nil
}
