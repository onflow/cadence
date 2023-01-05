/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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
	"strconv"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverEstimateIntStringLength(t *testing.T) {

	properties := gopter.NewProperties(nil)

	properties.Property("estimate is greater or equal to actual length", prop.ForAll(
		func(v int) bool {
			return OverEstimateIntStringLength(v) >= len(strconv.Itoa(v))
		},
		gen.Int(),
	))

	properties.TestingRun(t)

	for _, v := range []int{
		goMinInt,
		-1000,
		-999,
		-100,
		-99,
		-10,
		-9,
		-1,
		0,
		1,
		9,
		10,
		99,
		100,
		999,
		1000,
		goMaxInt,
	} {
		assert.LessOrEqual(t,
			len(strconv.Itoa(v)),
			OverEstimateIntStringLength(v),
		)
	}
}

func TestOverEstimateBigIntStringLength(t *testing.T) {

	properties := gopter.NewProperties(nil)

	properties.Property("estimate is greater or equal to actual length", prop.ForAll(
		func(v *big.Int) bool {
			return OverEstimateBigIntStringLength(v) >= len(v.String())
		},
		gen.Int64().Map(func(v int64) *big.Int {
			b := big.NewInt(v)
			return b.Exp(b, big.NewInt(42), nil)
		}),
	))

	properties.TestingRun(t)

	for _, v := range []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(9),
		big.NewInt(10),
		big.NewInt(99),
		big.NewInt(100),
		big.NewInt(999),
		big.NewInt(1000),
		big.NewInt(math.MaxUint16),
		big.NewInt(math.MaxUint32),
		new(big.Int).SetUint64(math.MaxUint64),
		func() *big.Int {
			v := new(big.Int).SetUint64(math.MaxUint64)
			return v.Mul(v, big.NewInt(2))
		}(),
		func() *big.Int {
			v := new(big.Int).SetUint64(math.MaxUint64)
			return v.Mul(v, new(big.Int).SetUint64(math.MaxUint64))
		}(),
	} {
		assert.LessOrEqual(t,
			len(v.String()),
			OverEstimateBigIntStringLength(v),
		)

		neg := new(big.Int).Neg(v)
		assert.LessOrEqual(t,
			len(neg.String()),
			OverEstimateBigIntStringLength(neg),
		)
	}
}

func TestOverEstimateFixedPointStringLength(t *testing.T) {

	properties := gopter.NewProperties(nil)

	properties.Property("estimate is greater or equal to actual length", prop.ForAll(
		func(v NumberValue, scale int) bool {
			return OverEstimateFixedPointStringLength(nil, v, scale) >= len(v.String())+1+scale
		},
		gen.Int64().Map(func(v int64) NumberValue {
			return NewUnmeteredIntValueFromInt64(v)
		}),
		gen.Int(),
	))

	properties.TestingRun(t)

	const testScale = 10

	for _, v := range []int{
		goMinInt,
		-1000,
		-999,
		-100,
		-99,
		-10,
		-9,
		-1,
		0,
		1,
		9,
		10,
		99,
		100,
		999,
		1000,
		goMaxInt,
	} {
		assert.LessOrEqual(t,
			len(strconv.Itoa(v))+1+testScale,
			OverEstimateFixedPointStringLength(
				nil,
				NewUnmeteredIntValueFromInt64(int64(v)),
				testScale,
			),
		)
	}
}

func BenchmarkValueArithmetic(b *testing.B) {

	b.ReportAllocs()

	isNegatable := func(v NumberValue) bool {
		switch v.(type) {
		case IntValue, Int8Value, Int16Value,
			Int32Value, Int64Value, Int128Value,
			Int256Value, Fix64Value:
			return true
		}
		return false
	}

	isSaturating := func(v NumberValue) bool {
		switch v.(type) {
		case Word8Value, Word16Value, Word32Value, Word64Value:
			return false
		}
		return true
	}

	testArithmeticOps := func(name string, v2 NumberValue, v1 NumberValue) {
		b.Run(name, func(b *testing.B) {
			storage := NewInMemoryStorage(nil)

			inter, err := NewInterpreter(
				nil,
				EmptyLocationRange.Location,
				&Config{
					Storage:                       storage,
					AtreeValueValidationEnabled:   true,
					AtreeStorageValidationEnabled: true,
				},
			)

			require.NoError(b, err)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				v1.Plus(inter, v2, EmptyLocationRange)
				v1.Minus(inter, v2, EmptyLocationRange)
				v1.Mod(inter, v2, EmptyLocationRange)
				v1.Mul(inter, v2, EmptyLocationRange)
				v1.Div(inter, v2, EmptyLocationRange)
				v1.Less(inter, v2, EmptyLocationRange)
				v1.LessEqual(inter, v2, EmptyLocationRange)
				v1.Greater(inter, v2, EmptyLocationRange)
				v1.GreaterEqual(inter, v2, EmptyLocationRange)

				if isNegatable(v1) {
					v1.Negate(inter, EmptyLocationRange)
				}
				if isSaturating(v1) {
					v1.SaturatingPlus(inter, v2, EmptyLocationRange)
					v1.SaturatingMinus(inter, v2, EmptyLocationRange)
					v1.SaturatingMul(inter, v2, EmptyLocationRange)
					v1.SaturatingDiv(inter, v2, EmptyLocationRange)
				}
			}
		})
	}

	testArithmeticOps("Int8Value", NewUnmeteredInt8Value(4), NewUnmeteredInt8Value(5))
	testArithmeticOps("Int16Value", NewUnmeteredInt16Value(40), NewUnmeteredInt16Value(50))
	testArithmeticOps("Int32Value", NewUnmeteredInt32Value(400), NewUnmeteredInt32Value(500))
	testArithmeticOps("Int64Value", NewUnmeteredInt64Value(4000), NewUnmeteredInt64Value(5000))
	testArithmeticOps("Fix64Value", NewUnmeteredFix64Value(4000), NewUnmeteredFix64Value(5000))
	testArithmeticOps("Int128Value", NewUnmeteredInt128ValueFromInt64(400000), NewUnmeteredInt128ValueFromInt64(500000))
	testArithmeticOps("Int256Value", NewUnmeteredInt256ValueFromInt64(400000), NewUnmeteredInt256ValueFromInt64(500000))

	testArithmeticOps("UInt8Value", NewUnmeteredUInt8Value(4), NewUnmeteredUInt8Value(5))
	testArithmeticOps("UInt16Value", NewUnmeteredUInt16Value(40), NewUnmeteredUInt16Value(50))
	testArithmeticOps("UInt32Value", NewUnmeteredUInt32Value(400), NewUnmeteredUInt32Value(500))
	testArithmeticOps("UInt64Value", NewUnmeteredUInt64Value(4000), NewUnmeteredUInt64Value(5000))
	testArithmeticOps("UFix64Value", NewUnmeteredUFix64Value(4000), NewUnmeteredUFix64Value(5000))
	testArithmeticOps("UInt128Value", NewUnmeteredUInt128ValueFromUint64(400000), NewUnmeteredUInt128ValueFromUint64(500000))
	testArithmeticOps("UInt256Value", NewUnmeteredUInt256ValueFromUint64(400000), NewUnmeteredUInt256ValueFromUint64(500000))

	testArithmeticOps("Word8Value", NewUnmeteredWord8Value(4), NewUnmeteredWord8Value(5))
	testArithmeticOps("Word16Value", NewUnmeteredWord16Value(40), NewUnmeteredWord16Value(50))
	testArithmeticOps("Word32Value", NewUnmeteredWord32Value(400), NewUnmeteredWord32Value(500))
	testArithmeticOps("Word64Value", NewUnmeteredWord64Value(4000), NewUnmeteredWord64Value(5000))
}
