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

package values

import (
	"math/big"
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/format"
)

type IntValue struct {
	BigInt *big.Int
}

const int64Size = int(unsafe.Sizeof(int64(0)))

var int64BigIntMemoryUsage = common.NewBigIntMemoryUsage(int64Size)

func NewIntValueFromInt64(memoryGauge common.MemoryGauge, value int64) IntValue {
	return NewIntValueFromBigInt(
		memoryGauge,
		int64BigIntMemoryUsage,
		func() *big.Int {
			return big.NewInt(value)
		},
	)
}

func NewUnmeteredIntValueFromInt64(value int64) IntValue {
	return NewUnmeteredIntValueFromBigInt(big.NewInt(value))
}

func NewIntValueFromBigInt(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) IntValue {
	common.UseMemory(memoryGauge, memoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredIntValueFromBigInt(value)
}

func NewUnmeteredIntValueFromBigInt(value *big.Int) IntValue {
	return IntValue{
		BigInt: value,
	}
}

var _ Value = IntValue{}
var _ EquatableValue = IntValue{}
var _ ComparableValue = IntValue{}
var _ NumberValue = IntValue{}
var _ IntegerValue = IntValue{}
var _ atree.Storable = IntValue{}
var _ atree.Value = IntValue{}

func (IntValue) isValue() {}

func (v IntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v IntValue) Negate(gauge common.MemoryGauge) NumberValue {
	return NewIntValueFromBigInt(
		gauge,
		common.NewNegateBigIntMemoryUsage(v.BigInt),
		func() *big.Int {
			return new(big.Int).Neg(v.BigInt)
		},
	)
}

func (v IntValue) Plus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewPlusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Add(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) SaturatingPlus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	return v.Plus(gauge, other)
}

func (v IntValue) Minus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewMinusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Sub(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) SaturatingMinus(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	return v.Minus(gauge, other)
}

func (v IntValue) Mod(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	// INT33-C
	if o.BigInt.Sign() == 0 {
		return nil, DivisionByZeroError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewModBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Rem(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) Mul(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewMulBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Mul(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) SaturatingMul(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	return v.Mul(gauge, other)
}

func (v IntValue) Div(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	// INT33-C
	if o.BigInt.Sign() == 0 {
		return nil, DivisionByZeroError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewDivBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Div(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) SaturatingDiv(gauge common.MemoryGauge, other NumberValue) (NumberValue, error) {
	return v.Div(gauge, other)
}

func (v IntValue) Less(other ComparableValue) (BoolValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return false, InvalidOperandsError{}
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1, nil
}

func (v IntValue) LessEqual(other ComparableValue) (BoolValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return false, InvalidOperandsError{}
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0, nil
}

func (v IntValue) Greater(other ComparableValue) (BoolValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return false, InvalidOperandsError{}
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1, nil

}

func (v IntValue) GreaterEqual(other ComparableValue) (BoolValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return false, InvalidOperandsError{}
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0, nil
}

func (v IntValue) Equal(other Value) BoolValue {
	otherInt, ok := other.(IntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

func (v IntValue) BitwiseOr(gauge common.MemoryGauge, other IntegerValue) (IntegerValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewBitwiseOrBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) BitwiseXor(gauge common.MemoryGauge, other IntegerValue) (IntegerValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewBitwiseXorBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) BitwiseAnd(gauge common.MemoryGauge, other IntegerValue) (IntegerValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewBitwiseAndBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	), nil
}

func (v IntValue) BitwiseLeftShift(gauge common.MemoryGauge, other IntegerValue) (IntegerValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	if o.BigInt.Sign() < 0 {
		return nil, NegativeShiftError{}
	}

	if !o.BigInt.IsUint64() {
		return nil, OverflowError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewBitwiseLeftShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	), nil
}

func (v IntValue) BitwiseRightShift(gauge common.MemoryGauge, other IntegerValue) (IntegerValue, error) {
	o, ok := other.(IntValue)
	if !ok {
		return nil, InvalidOperandsError{}
	}

	if o.BigInt.Sign() < 0 {
		return nil, NegativeShiftError{}
	}

	if !o.BigInt.IsUint64() {
		return nil, OverflowError{}
	}

	return NewIntValueFromBigInt(
		gauge,
		common.NewBitwiseRightShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	), nil
}

func (v IntValue) ToInt() (int, error) {
	if !v.BigInt.IsInt64() {
		return 0, OverflowError{}
	}
	return int(v.BigInt.Int64()), nil
}

func (v IntValue) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v IntValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return MaybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (v IntValue) ByteSize() uint32 {
	return CBORTagSize + GetBigIntCBORSize(v.BigInt)
}

func (v IntValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (IntValue) ChildStorables() []atree.Storable {
	return nil
}

// Encode encodes the value as
//
//	cbor.Tag{
//			Number:  CBORTagIntValue,
//			Content: *big.Int(v.BigInt),
//	}
func (v IntValue) Encode(e *atree.Encoder) error {
	err := e.CBOR.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagIntValue,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeBigInt(v.BigInt)
}
