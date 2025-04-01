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
	"encoding/binary"
	"math"
	"math/big"
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

type UFix64Value uint64

const ufix64Size = int(unsafe.Sizeof(UFix64Value(0)))

var ufix64MemoryUsage = common.NewNumberMemoryUsage(ufix64Size)

func NewUFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() (uint64, error)) (UFix64Value, error) {
	common.UseMemory(gauge, ufix64MemoryUsage)
	v, err := constructor()
	if err != nil {
		return 0, err
	}
	return NewUnmeteredUFix64ValueWithInteger(v)
}

func NewUnmeteredUFix64ValueWithInteger(integer uint64) (UFix64Value, error) {
	if integer > sema.UFix64TypeMaxInt {
		return 0, OverflowError{}
	}

	return NewUnmeteredUFix64Value(integer * sema.Fix64Factor), nil
}

func NewUFix64Value(gauge common.MemoryGauge, constructor func() (uint64, error)) (UFix64Value, error) {
	common.UseMemory(gauge, ufix64MemoryUsage)
	v, err := constructor()
	if err != nil {
		return 0, err
	}
	return NewUnmeteredUFix64Value(v), nil
}

func NewUnmeteredUFix64Value(integer uint64) UFix64Value {
	return UFix64Value(integer)
}

var _ Value = UFix64Value(0)
var _ EquatableValue = UFix64Value(0)
var _ ComparableValue[UFix64Value] = UFix64Value(0)
var _ NumberValue[UFix64Value] = UFix64Value(0)
var _ FixedPointValue[UFix64Value, uint64] = UFix64Value(0)
var _ atree.Storable = UFix64Value(0)

func (UFix64Value) isValue() {}

func (v UFix64Value) String() string {
	return format.UFix64(uint64(v))
}

func (v UFix64Value) Negate(_ common.MemoryGauge) UFix64Value {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Plus(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {

	valueGetter := func() (uint64, error) {
		return SafeAddUint64(uint64(v), uint64(other))
	}

	return NewUFix64Value(gauge, valueGetter)
}

func (v UFix64Value) SaturatingPlus(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {
	valueGetter := func() (uint64, error) {
		sum := v + other
		// INT30-C
		if sum < v {
			return math.MaxUint64, nil
		}
		return uint64(sum), nil
	}

	return NewUFix64Value(gauge, valueGetter)
}

func (v UFix64Value) Minus(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {
	valueGetter := func() (uint64, error) {
		diff := v - other

		// INT30-C
		if diff > v {
			return 0, UnderflowError{}
		}
		return uint64(diff), nil
	}

	return NewUFix64Value(gauge, valueGetter)
}

func (v UFix64Value) SaturatingMinus(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {
	valueGetter := func() (uint64, error) {
		diff := v - other

		// INT30-C
		if diff > v {
			return 0, nil
		}
		return uint64(diff), nil
	}

	return NewUFix64Value(gauge, valueGetter)
}

func (v UFix64Value) Mul(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(other))

	valueGetter := func() (uint64, error) {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if !result.IsUint64() {
			return 0, OverflowError{}
		}

		return result.Uint64(), nil
	}

	return NewUFix64Value(gauge, valueGetter)
}

func (v UFix64Value) SaturatingMul(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(other))

	valueGetter := func() (uint64, error) {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if !result.IsUint64() {
			return math.MaxUint64, nil
		}

		return result.Uint64(), nil
	}

	return NewUFix64Value(gauge, valueGetter)
}

func (v UFix64Value) Div(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(other))

	valueGetter := func() (uint64, error) {
		result := new(big.Int).Mul(a, sema.Fix64FactorBig)
		result.Div(result, b)

		return result.Uint64(), nil
	}

	return NewUFix64Value(gauge, valueGetter)
}

func (v UFix64Value) SaturatingDiv(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {
	return v.Div(gauge, other)
}

func (v UFix64Value) Mod(gauge common.MemoryGauge, other UFix64Value) (UFix64Value, error) {
	// v - int(v/o) * o
	quotient, err := v.Div(gauge, other)
	if err != nil {
		return 0, err
	}

	truncatedQuotient, err := NewUFix64Value(
		gauge,
		func() (uint64, error) {
			return (uint64(quotient) / sema.Fix64Factor) * sema.Fix64Factor, nil
		},
	)
	if err != nil {
		return 0, err
	}

	subtrahend, err := truncatedQuotient.Mul(gauge, other)
	if err != nil {
		return 0, err
	}

	return v.Minus(gauge, subtrahend)
}

func (v UFix64Value) Less(other UFix64Value) bool {
	return v < other
}

func (v UFix64Value) LessEqual(other UFix64Value) bool {
	return v <= other
}

func (v UFix64Value) Greater(other UFix64Value) bool {
	return v > other
}

func (v UFix64Value) GreaterEqual(other UFix64Value) bool {
	return v >= other
}

func (v UFix64Value) Equal(other Value) bool {
	otherUFix64, ok := other.(UFix64Value)
	if !ok {
		return false
	}
	return v == otherUFix64
}

func (v UFix64Value) IntegerPart() uint64 {
	return uint64(v / sema.Fix64Factor)
}

func (UFix64Value) Scale() int {
	return sema.Fix64Scale
}

func (v UFix64Value) ToInt() (int, error) {
	return int(v / sema.Fix64Factor), nil
}

func (v UFix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UFix64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (v UFix64Value) ByteSize() uint32 {
	return CBORTagSize + GetUintCBORSize(uint64(v))
}

func (v UFix64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UFix64Value) ChildStorables() []atree.Storable {
	return nil
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
		0xd8, CBORTagUFix64Value,
	})
	if err != nil {
		return err
	}
	return e.CBOR.EncodeUint64(uint64(v))
}
