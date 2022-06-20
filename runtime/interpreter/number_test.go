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
