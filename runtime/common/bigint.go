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
	"math/big"
)

var bigIntSize = BigIntByteLength(new(big.Int))

var bigIntMemoryUsage = NewBigIntMemoryUsage(bigIntSize)

func NewBigInt(gauge MemoryGauge) *big.Int {
	UseMemory(gauge, bigIntMemoryUsage)
	return new(big.Int)
}

func NewBigIntFromAbsoluteValue(gauge MemoryGauge, value *big.Int) *big.Int {
	UseMemory(
		gauge,
		NewBigIntMemoryUsage(
			BigIntByteLength(value),
		),
	)

	return new(big.Int).Abs(value)
}

const minBigIntLength = 4 * BigIntWordSize

// OverEstimateBigIntFromString is an approximate inverse of `interpreter.OverEstimateBigIntStringLength`.
// Returns the estimated size in bytes.
func OverEstimateBigIntFromString(s string) int {
	l := len(s)

	// Use 6804/2028 to over estimate log_2(10)
	bitLen := (l*6804)>>11 + 1

	// Calculate amount of bytes.
	// First convert to word, and then find the size,
	// to be consistent with `common.BigIntByteLength`
	wordLen := (bitLen + 63) / 64
	result := wordLen * BigIntWordSize
	if result < minBigIntLength {
		return minBigIntLength
	}
	return result
}
