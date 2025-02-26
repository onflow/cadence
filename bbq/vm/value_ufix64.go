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

package vm

import (
	"github.com/onflow/cadence/bbq"
	"math/big"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type UFix64Value uint64

func NewUFix64Value(value uint64) UFix64Value {
	return UFix64Value(value)
}

func (v UFix64Value) String() string {
	return format.UFix64(uint64(v))
}

var _ Value = UFix64Value(0)
var _ EquatableValue = UFix64Value(0)
var _ ComparableValue = UFix64Value(0)
var _ NumberValue = UFix64Value(0)

func (UFix64Value) isValue() {}

func (UFix64Value) StaticType(*Config) bbq.StaticType {
	return interpreter.PrimitiveStaticTypeUFix64
}

func (v UFix64Value) Transfer(*Config, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v UFix64Value) Add(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}
	sum := safeAddUint64(uint64(v), uint64(o))
	return NewUFix64Value(sum)
}

func (v UFix64Value) Subtract(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}

	diff := v - o

	// INT30-C
	if diff > v {
		panic(interpreter.UnderflowError{})
	}

	return diff
}

func (v UFix64Value) Multiply(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	result := new(big.Int).Mul(a, b)
	result.Div(result, sema.Fix64FactorBig)

	if !result.IsUint64() {
		panic(interpreter.OverflowError{})
	}

	return NewUFix64Value(result.Uint64())
}

func (v UFix64Value) Divide(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	result := new(big.Int).Mul(a, sema.Fix64FactorBig)
	result.Div(result, b)

	return NewUFix64Value(result.Uint64())
}

func (v UFix64Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}

	// v - int(v/o) * o
	quotient, ok := v.Divide(o).(UFix64Value)
	if !ok {
		panic("invalid operand")
	}

	truncatedQuotient := NewUFix64Value(
		(uint64(quotient) / sema.Fix64Factor) * sema.Fix64Factor,
	)

	return v.Subtract(
		truncatedQuotient.Multiply(o),
	)
}

func (v UFix64Value) Equal(other Value) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		return false
	}
	return v == o
}

func (v UFix64Value) Less(other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}
	return v < o
}

func (v UFix64Value) LessEqual(other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}
	return v <= o
}

func (v UFix64Value) Greater(other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}
	return v > o
}

func (v UFix64Value) GreaterEqual(other ComparableValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic("invalid operand")
	}
	return v >= o
}
