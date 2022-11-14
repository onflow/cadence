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

	"github.com/onflow/cadence/runtime/errors"
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
func OverEstimateBigIntFromString(s string, literalKind IntegerLiteralKind) int {
	l := len(s)

	// By definition: log_b(v) = log_2(v) / log_2(b)
	// i.e: log_2(v) = log_b(v) * log_2(b)

	// Each digit in base 'b' requires 'log_2(b)' number of digits to represent in binary.
	// Therefore, to get the number of digits in base 'b', we could multiply by 'log_2(b)'.

	var bitLen int
	switch literalKind {
	case IntegerLiteralKindBinary:
		// Already in binary, hence 'bitLen' is same as length of the string. i.e: 'l'
		// Also from: bitLen = l * log_2(2)
		bitLen = l
	case IntegerLiteralKindOctal:
		// bitLen = l * log_2(8)
		bitLen = l * 3
	case IntegerLiteralKindDecimal:
		// bitLen = l * log_2(10) + 1
		// Use 6804/2028 to over estimate log_2(10)
		bitLen = (l*6804)>>11 + 1
	case IntegerLiteralKindHexadecimal:
		// bitLen = l * log_2(16)
		bitLen = l * 4
	default:
		panic(errors.NewUnreachableError())
	}

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
