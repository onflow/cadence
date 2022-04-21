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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

const goIntSize = 32 << (^uint(0) >> 63) // 32 or 64
const goMaxInt = 1<<(goIntSize-1) - 1
const goMinInt = -1 << (goIntSize - 1)

func OverEstimateNumberStringLength(memoryGauge common.MemoryGauge, value NumberValue) int {
	switch value := value.(type) {
	case BigNumberValue:
		return OverEstimateBigIntStringLength(value.ToBigInt(memoryGauge))

	case FixedPointValue:
		return OverEstimateFixedPointStringLength(
			memoryGauge,
			value.IntegerPart(),
			value.Scale(),
		)

	case NumberValue:
		return OverEstimateIntStringLength(value.ToInt())

	default:
		panic(errors.NewUnreachableError())
	}
}

func OverEstimateFixedPointStringLength(
	memoryGauge common.MemoryGauge,
	integerPart NumberValue,
	scale int,
) int {
	integerPartLength := OverEstimateNumberStringLength(memoryGauge, integerPart)
	return integerPartLength + 1 + scale
}

func OverEstimateIntStringLength(n int) int {
	switch {
	case n < 0:
		// Handle math.MinInt
		return 1 + OverEstimateUintStringLength(uint(-(n+1))+1)
	case n > 0:
		return OverEstimateUintStringLength(uint(n))
	default:
		return 1
	}
}

func OverEstimateUintStringLength(n uint) int {
	return int(math.Floor(math.Log10(float64(n))) + 1)
}

func OverEstimateBigIntStringLength(n *big.Int) int {
	// From https://graphics.stanford.edu/~seander/bithacks.html#IntegerLog10:
	//   By the relationship log10(v) = log2(v) / log2(10), we need to multiply it by 1/log2(10),
	//   which is approximately 1233/4096, or 1233 followed by a right shift of 12.
	//
	// From Tarak:
	//   Looking for the ceiling of the log 10 (the number of digits in base 10),
	//   `(n.BitLen()*1233)>>12 + 1` indeed gives an approximation of that ceiling,
	//   though it won't be an upper-bound for very very big integers.
	//
	//   To be sure it's always an upper bound (over-estimation), just use *1234,
	//   since 1233/4096 is just smaller than 1/log2(10), but 1234/4096 becomes bigger.
	//
	l := n.BitLen()*1234>>12 + 1
	if n.Sign() < 0 {
		return l + 1
	} else {
		return l
	}
}
