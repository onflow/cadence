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
	"fmt"
	"math"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverEstimateBigIntFromString_LowerBound(t *testing.T) {

	t.Parallel()

	require.Equal(t, 32, OverEstimateBigIntFromString("1", IntegerLiteralKindDecimal))
}

func TestOverEstimateBigIntFromString_OverEstimation(t *testing.T) {

	t.Parallel()

	t.Run("base_10", func(t *testing.T) {

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
				OverEstimateBigIntFromString(v.String(), IntegerLiteralKindDecimal),
			)

			neg := new(big.Int).Neg(v)
			assert.LessOrEqual(t,
				BigIntByteLength(neg),
				OverEstimateBigIntFromString(neg.String(), IntegerLiteralKindDecimal),
			)
		}
	})

	t.Run("base_2", func(t *testing.T) {

		t.Parallel()

		const base = 2

		for _, s := range []string{
			strconv.FormatInt(0, base),
			strconv.FormatInt(1, base),
			strconv.FormatInt(7, base),
			strconv.FormatInt(8, base),
			strconv.FormatInt(15, base),
			strconv.FormatInt(16, base),
			strconv.FormatInt(1023, base),
			strconv.FormatInt(1024, base),
			strconv.FormatUint(math.MaxUint16, base),
			strconv.FormatUint(math.MaxUint32, base),
			strconv.FormatUint(math.MaxUint64, base),
		} {

			v, ok := new(big.Int).SetString(s, base)
			assert.True(t, ok)

			// Always should be equal or overestimate
			assert.LessOrEqual(t,
				BigIntByteLength(v),
				OverEstimateBigIntFromString(s, IntegerLiteralKindBinary),
			)

			neg := new(big.Int).Neg(v)
			assert.LessOrEqual(t,
				BigIntByteLength(neg),
				OverEstimateBigIntFromString(
					fmt.Sprintf("-%s", s),
					IntegerLiteralKindBinary,
				),
			)
		}
	})

	t.Run("base_8", func(t *testing.T) {

		t.Parallel()

		const base = 8

		for _, s := range []string{
			strconv.FormatInt(0, base),
			strconv.FormatInt(1, base),
			strconv.FormatInt(7, base),
			strconv.FormatInt(8, base),
			strconv.FormatInt(63, base),
			strconv.FormatInt(64, base),
			strconv.FormatInt(1023, base),
			strconv.FormatInt(1024, base),
			strconv.FormatUint(math.MaxUint16, base),
			strconv.FormatUint(math.MaxUint32, base),
			strconv.FormatUint(math.MaxUint64, base),
		} {

			v, ok := new(big.Int).SetString(s, base)
			assert.True(t, ok)

			// Always should be equal or overestimate
			assert.LessOrEqual(t,
				BigIntByteLength(v),
				OverEstimateBigIntFromString(s, IntegerLiteralKindOctal),
			)

			neg := new(big.Int).Neg(v)
			assert.LessOrEqual(t,
				BigIntByteLength(neg),
				OverEstimateBigIntFromString(
					fmt.Sprintf("-%s", s),
					IntegerLiteralKindOctal,
				),
			)
		}
	})

	t.Run("base_16", func(t *testing.T) {

		t.Parallel()

		const base = 16

		for _, s := range []string{
			strconv.FormatInt(0, base),
			strconv.FormatInt(1, base),
			strconv.FormatInt(15, base),
			strconv.FormatInt(16, base),
			strconv.FormatInt(127, base),
			strconv.FormatInt(128, base),
			strconv.FormatInt(1023, base),
			strconv.FormatInt(1024, base),
			strconv.FormatUint(math.MaxUint16, base),
			strconv.FormatUint(math.MaxUint32, base),
			strconv.FormatUint(math.MaxUint64, base),
		} {

			v, ok := new(big.Int).SetString(s, base)
			assert.True(t, ok)

			// Always should be equal or overestimate
			assert.LessOrEqual(t,
				BigIntByteLength(v),
				OverEstimateBigIntFromString(s, IntegerLiteralKindHexadecimal),
			)

			neg := new(big.Int).Neg(v)
			assert.LessOrEqual(t,
				BigIntByteLength(neg),
				OverEstimateBigIntFromString(
					fmt.Sprintf("-%s", s),
					IntegerLiteralKindHexadecimal,
				),
			)
		}
	})
}
