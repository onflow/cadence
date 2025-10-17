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
	"encoding/binary"
	"math/big"

	fix "github.com/onflow/fixed-point"
)

func ConvertToFixedPointBigInt(
	negative bool,
	unsignedInteger *big.Int,
	fractional *big.Int,
	scale uint,
	targetScale uint,
) *big.Int {
	ten := big.NewInt(10)

	// integer = unsignedInteger * 10 ^ targetScale

	bigTargetScale := new(big.Int).SetUint64(uint64(targetScale))

	integer := new(big.Int).Mul(
		unsignedInteger,
		new(big.Int).Exp(ten, bigTargetScale, nil),
	)

	// fractional = fractional * 10 ^ (targetScale - scale)

	if scale < targetScale {
		scaleDiff := new(big.Int).SetUint64(uint64(targetScale - scale))
		fractional = new(big.Int).Mul(
			fractional,
			new(big.Int).Exp(ten, scaleDiff, nil),
		)
	} else if scale > targetScale {
		scaleDiff := new(big.Int).SetUint64(uint64(scale - targetScale))
		fractional = new(big.Int).Div(fractional,
			new(big.Int).Exp(ten, scaleDiff, nil),
		)
	}

	// value = integer + fractional

	result := integer.Add(integer, fractional)

	if negative {
		result.Neg(result)
	}

	return result
}

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

func Fix128FromIntAndScale(integer, scale int64) fix.Fix128 {
	bigInt := big.NewInt(integer)
	bigInt = new(big.Int).Mul(
		bigInt,
		// To remove the fractional, multiply it by the given scale.
		new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(scale),
			nil,
		),
	)

	return Fix128FromBigInt(bigInt)
}

func Fix128ToBigInt(fix128 fix.Fix128) *big.Int {
	high := new(big.Int).SetUint64(uint64(fix128.Hi))
	low := new(big.Int).SetUint64(uint64(fix128.Lo))

	// v = (high << 64) + low
	result := high.Mul(high, twoPow64)
	result = result.Add(result, low)

	// If sign bit (bit 127) is set, it's a negative number in two's complement.
	// Subtract 2^128 to get negative value.
	if fix128.Hi&(1<<63) != 0 {
		result = result.Sub(result, twoPow128)
	}

	return result
}

func Fix128ToBigEndianBytes(fix128 fix.Fix128) []byte {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b[:8], uint64(fix128.Hi))
	binary.BigEndian.PutUint64(b[8:], uint64(fix128.Lo))
	return b
}

func UFix128FromBigInt(value *big.Int) fix.UFix128 {
	v := new(big.Int).Set(value)

	// Use v.Uint64() to get the low 64 bits
	low := v.Uint64()

	// Shift right to get the high 64 bits
	high := new(big.Int).Rsh(v, 64).Uint64()

	return fix.NewUFix128(high, low)
}

func UFix128ToBigInt(value fix.UFix128) *big.Int {
	high := new(big.Int).SetUint64(uint64(value.Hi))
	low := new(big.Int).SetUint64(uint64(value.Lo))

	result := high.Lsh(high, 64)
	result = result.Add(result, low)

	return result
}

func UFix128ToBigEndianBytes(fix128 fix.UFix128) []byte {
	return Fix128ToBigEndianBytes(fix.Fix128(fix128))
}

func Fix64FromBigInt(value *big.Int) fix.Fix64 {
	return fix.Fix64(value.Int64())
}

func Fix64ToBigInt(value fix.Fix64) *big.Int {
	return big.NewInt(int64(value))
}

func UFix64FromBigInt(value *big.Int) fix.UFix64 {
	return fix.UFix64(value.Uint64())
}

func UFix64ToBigInt(value fix.UFix64) *big.Int {
	return new(big.Int).SetUint64(uint64(value))
}
