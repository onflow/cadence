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

var Fix64TypeMin = new(big.Int).SetInt64(math.MinInt64)
var Fix64TypeMax = new(big.Int).SetInt64(math.MaxInt64)

var Fix64TypeMinScaledTo128 = new(big.Int).Mul(
	Fix64TypeMin,
	Fix64ToFix128FactorAsBigInt,
)

var Fix64TypeMaxScaledTo128 = new(big.Int).Mul(
	Fix64TypeMax,
	Fix64ToFix128FactorAsBigInt,
)

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

	Fix64ToFix128FactorAsBigInt = new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(Fix128Scale-Fix64Scale),
		nil,
	)

	Fix128TypeMin = fix.Fix128Min
	Fix128TypeMax = fix.Fix128Max

	Fix128TypeMinBig = Fix128ToBigInt(Fix128TypeMin)
	Fix128TypeMaxBig = Fix128ToBigInt(Fix128TypeMax)

	Fix128TypeMinIntBig, Fix128TypeMinFractionalBig = func() (*big.Int, *big.Int) {
		quotient := new(big.Int)
		remainder := new(big.Int)

		// Use `QuoRem` (truncated division and modulus), instead of `DivMod`,
		// to be consistent with to Go's division used in Fix64.
		quotient, remainder = quotient.QuoRem(Fix128TypeMinBig, Fix128FactorAsBigInt, remainder)
		remainder = remainder.Abs(remainder)

		return quotient, remainder
	}()

	Fix128TypeMaxIntBig, Fix128TypeMaxFractionalBig = func() (*big.Int, *big.Int) {
		quotient := new(big.Int)
		remainder := new(big.Int)

		// Use `QuoRem` (truncated division and modulus), instead of `DivMod`,
		// to be consistent with to Go's division used in Fix64.
		quotient, remainder = quotient.QuoRem(Fix128TypeMaxBig, Fix128FactorAsBigInt, remainder)
		remainder = remainder.Abs(remainder)

		return quotient, remainder
	}()
)

// UFix64

const UFix64TypeMinInt = 0
const UFix64TypeMaxInt = math.MaxUint64 / uint64(Fix64Factor)

var UFix64TypeMinIntBig = new(big.Int).SetUint64(UFix64TypeMinInt)
var UFix64TypeMaxIntBig = new(big.Int).SetUint64(UFix64TypeMaxInt)

const UFix64TypeMinFractional = 0
const UFix64TypeMaxFractional = math.MaxUint64 % uint64(Fix64Factor)

var UFix64TypeMinFractionalBig = new(big.Int).SetUint64(UFix64TypeMinFractional)
var UFix64TypeMaxFractionalBig = new(big.Int).SetUint64(UFix64TypeMaxFractional)

var UFix64TypeMinScaledTo128 = new(big.Int).Mul(
	new(big.Int).SetUint64(0),
	Fix64ToFix128FactorAsBigInt,
)

var UFix64TypeMaxScaledTo128 = new(big.Int).Mul(
	new(big.Int).SetUint64(math.MaxUint64),
	Fix64ToFix128FactorAsBigInt,
)

// UFix128

const (
	UFix128Scale = Fix128Scale
)

var (
	UFix128FactorAsBigInt = Fix128FactorAsBigInt

	UFix128TypeMin = fix.UFix128Zero
	UFix128TypeMax = fix.UFix128Max

	UFix128TypeMinBig = UFix128ToBigInt(UFix128TypeMin)
	UFix128TypeMaxBig = UFix128ToBigInt(UFix128TypeMax)

	UFix128TypeMinIntBig        = big.NewInt(0)
	UFix128TypeMinFractionalBig = big.NewInt(0)

	UFix128TypeMaxIntBig, UFix128TypeMaxFractionalBig = func() (*big.Int, *big.Int) {
		quotient := new(big.Int)
		remainder := new(big.Int)

		// Use `QuoRem` (truncated division and modulus), instead of `DivMod`,
		// to be consistent with to Go's division used in Fix64.
		quotient, remainder = quotient.QuoRem(UFix128TypeMaxBig, UFix128FactorAsBigInt, remainder)
		remainder = remainder.Abs(remainder)

		return quotient, remainder
	}()
)

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
