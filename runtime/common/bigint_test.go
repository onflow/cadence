/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package common

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverEstimateBigIntFromString_LowerBound(t *testing.T) {

	t.Parallel()

	require.Equal(t, 32, OverEstimateBigIntFromString("1"))
}

func TestOverEstimateBigIntFromString_OverEstimation(t *testing.T) {

	t.Parallel()

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

		// Always should be equal or overestimate
		assert.LessOrEqual(t,
			BigIntByteLength(v),
			OverEstimateBigIntFromString(v.String()),
		)

		neg := new(big.Int).Neg(v)
		assert.LessOrEqual(t,
			BigIntByteLength(neg),
			OverEstimateBigIntFromString(neg.String()),
		)
	}
}
