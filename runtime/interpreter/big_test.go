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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverEstimateIntStringLength(t *testing.T) {
	assert.LessOrEqual(t, 5, OverEstimateIntStringLength(-1000))
	assert.LessOrEqual(t, 4, OverEstimateIntStringLength(-999))
	assert.LessOrEqual(t, 4, OverEstimateIntStringLength(-100))
	assert.LessOrEqual(t, 3, OverEstimateIntStringLength(-99))
	assert.LessOrEqual(t, 3, OverEstimateIntStringLength(-10))
	assert.LessOrEqual(t, 2, OverEstimateIntStringLength(-9))
	assert.LessOrEqual(t, 2, OverEstimateIntStringLength(-1))
	assert.LessOrEqual(t, 1, OverEstimateIntStringLength(0))
	assert.LessOrEqual(t, 1, OverEstimateIntStringLength(1))
	assert.LessOrEqual(t, 1, OverEstimateIntStringLength(9))
	assert.LessOrEqual(t, 2, OverEstimateIntStringLength(10))
	assert.LessOrEqual(t, 2, OverEstimateIntStringLength(99))
	assert.LessOrEqual(t, 3, OverEstimateIntStringLength(100))
	assert.LessOrEqual(t, 3, OverEstimateIntStringLength(999))
	assert.LessOrEqual(t, 4, OverEstimateIntStringLength(1000))
}

func TestOverEstimateBigIntStringLength(t *testing.T) {
	assert.LessOrEqual(t, 5, OverEstimateBigIntStringLength(big.NewInt(-1000)))
	assert.LessOrEqual(t, 4, OverEstimateBigIntStringLength(big.NewInt(-999)))
	assert.LessOrEqual(t, 4, OverEstimateBigIntStringLength(big.NewInt(-100)))
	assert.LessOrEqual(t, 3, OverEstimateBigIntStringLength(big.NewInt(-99)))
	assert.LessOrEqual(t, 3, OverEstimateBigIntStringLength(big.NewInt(-10)))
	assert.LessOrEqual(t, 2, OverEstimateBigIntStringLength(big.NewInt(-9)))
	assert.LessOrEqual(t, 2, OverEstimateBigIntStringLength(big.NewInt(-1)))
	assert.LessOrEqual(t, 1, OverEstimateBigIntStringLength(big.NewInt(0)))
	assert.LessOrEqual(t, 1, OverEstimateBigIntStringLength(big.NewInt(1)))
	assert.LessOrEqual(t, 1, OverEstimateBigIntStringLength(big.NewInt(9)))
	assert.LessOrEqual(t, 2, OverEstimateBigIntStringLength(big.NewInt(10)))
	assert.LessOrEqual(t, 2, OverEstimateBigIntStringLength(big.NewInt(99)))
	assert.LessOrEqual(t, 3, OverEstimateBigIntStringLength(big.NewInt(100)))
	assert.LessOrEqual(t, 3, OverEstimateBigIntStringLength(big.NewInt(999)))
	assert.LessOrEqual(t, 4, OverEstimateBigIntStringLength(big.NewInt(1000)))
}

func TestOverEstimateFixedPointStringLength(t *testing.T) {
	const testScale = 10
	assert.LessOrEqual(t, 5+testScale, OverEstimateFixedPointStringLength(Int64Value(-1000), testScale))
	assert.LessOrEqual(t, 4+testScale, OverEstimateFixedPointStringLength(Int64Value(-999), testScale))
	assert.LessOrEqual(t, 4+testScale, OverEstimateFixedPointStringLength(Int64Value(-100), testScale))
	assert.LessOrEqual(t, 3+testScale, OverEstimateFixedPointStringLength(Int64Value(-99), testScale))
	assert.LessOrEqual(t, 3+testScale, OverEstimateFixedPointStringLength(Int64Value(-10), testScale))
	assert.LessOrEqual(t, 2+testScale, OverEstimateFixedPointStringLength(Int64Value(-9), testScale))
	assert.LessOrEqual(t, 2+testScale, OverEstimateFixedPointStringLength(Int64Value(-1), testScale))
	assert.LessOrEqual(t, 1+testScale, OverEstimateFixedPointStringLength(Int64Value(0), testScale))
	assert.LessOrEqual(t, 1+testScale, OverEstimateFixedPointStringLength(Int64Value(1), testScale))
	assert.LessOrEqual(t, 1+testScale, OverEstimateFixedPointStringLength(Int64Value(9), testScale))
	assert.LessOrEqual(t, 2+testScale, OverEstimateFixedPointStringLength(Int64Value(10), testScale))
	assert.LessOrEqual(t, 2+testScale, OverEstimateFixedPointStringLength(Int64Value(99), testScale))
	assert.LessOrEqual(t, 3+testScale, OverEstimateFixedPointStringLength(Int64Value(100), testScale))
	assert.LessOrEqual(t, 3+testScale, OverEstimateFixedPointStringLength(Int64Value(999), testScale))
	assert.LessOrEqual(t, 4+testScale, OverEstimateFixedPointStringLength(Int64Value(1000), testScale))
}
