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

package fixedpoint

import (
	"math"
	"math/big"

	fix "github.com/onflow/fixed-point"
)

const Fix64Scale = 8
const Fix64Factor = 100_000_000

// Fix64

const Fix64TypeMinInt = math.MinInt64 / Fix64Factor
const Fix64TypeMaxInt = math.MaxInt64 / Fix64Factor

var Fix64TypeMinIntBig = new(big.Int).SetInt64(Fix64TypeMinInt)
var Fix64TypeMaxIntBig = new(big.Int).SetInt64(Fix64TypeMaxInt)

const Fix64TypeMinFractional = math.MinInt64 % Fix64Factor
const Fix64TypeMaxFractional = math.MaxInt64 % Fix64Factor

var Fix64TypeMinFractionalBig = new(big.Int).SetInt64(Fix64TypeMinFractional)
var Fix64TypeMaxFractionalBig = new(big.Int).SetInt64(Fix64TypeMaxFractional)

// Fix128

const (
	Fix128Scale      = 24
	Fix128MaxBits    = 128
	Fix128LowMaxBits = 64
)

var (
	twoPow64  = new(big.Int).Lsh(big.NewInt(1), Fix128LowMaxBits) // 2^64
	twoPow128 = new(big.Int).Lsh(big.NewInt(1), Fix128MaxBits)    // 2^128

	Fix128FactorAsBigInt = new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(Fix128Scale),
		nil,
	)

	Fix128FactorAsFix128 = Fix128FromBigInt(Fix128FactorAsBigInt)
)

func Fix128FromBigInt(value *big.Int) fix.Fix128 {
	v := new(big.Int).Set(value)

	// Handle negative values using two's complement
	if value.Sign() < 0 {
		// Convert to 2's complement: x + 2^128
		v = v.Add(v, twoPow128)
	}

	// Use v.Uint64() if it fits in 64 bits
	low := v.Uint64()

	// Shift right to get the high 64 bits
	high := v.Rsh(v, 64).Uint64()

	return fix.NewFix128(high, low)
}

func Fix128ToBigInt(fix128 fix.Fix128) *big.Int {
	high := new(big.Int).SetUint64(uint64(fix128.Hi))
	low := new(big.Int).SetUint64(uint64(fix128.Lo))

	// v = (high << 64) + low, done in place with minimal temp vars
	result := high.Mul(high, twoPow64)
	result = result.Add(result, low)

	// If sign bit (bit 127) is set, it's a negative number in two's complement.
	// Subtract 2^128 to get negative value.
	if fix128.Hi&(1<<63) != 0 {
		result = result.Sub(result, twoPow128)
	}

	return result
}

// UFix64

const UFix64TypeMinInt = 0
const UFix64TypeMaxInt = math.MaxUint64 / uint64(Fix64Factor)

var UFix64TypeMinIntBig = new(big.Int).SetUint64(UFix64TypeMinInt)
var UFix64TypeMaxIntBig = new(big.Int).SetUint64(UFix64TypeMaxInt)

const UFix64TypeMinFractional = 0
const UFix64TypeMaxFractional = math.MaxUint64 % uint64(Fix64Factor)

var UFix64TypeMinFractionalBig = new(big.Int).SetUint64(UFix64TypeMinFractional)
var UFix64TypeMaxFractionalBig = new(big.Int).SetUint64(UFix64TypeMaxFractional)

func init() {
	Fix64TypeMinFractionalBig.Abs(Fix64TypeMinFractionalBig)
}

func CheckRange(
	negative bool,
	unsignedIntegerValue, fractionalValue,
	minInt, minFractional,
	maxInt, maxFractional *big.Int,
) bool {
	minIntSign := minInt.Sign()

	integerValue := new(big.Int).Set(unsignedIntegerValue)
	if negative {
		if minIntSign == 0 && negative {
			return false
		}

		integerValue.Neg(integerValue)
	}

	switch integerValue.Cmp(minInt) {
	case -1:
		return false
	case 0:
		if minIntSign < 0 {
			if fractionalValue.Cmp(minFractional) > 0 {
				return false
			}
		} else {
			if fractionalValue.Cmp(minFractional) < 0 {
				return false
			}
		}
	case 1:
		break
	}

	switch integerValue.Cmp(maxInt) {
	case -1:
		break
	case 0:
		if maxInt.Sign() >= 0 {
			if fractionalValue.Cmp(maxFractional) > 0 {
				return false
			}
		} else {
			if fractionalValue.Cmp(maxFractional) < 0 {
				return false
			}
		}
	case 1:
		return false
	}

	return true
}
