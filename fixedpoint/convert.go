/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"math/big"
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

	if negative {
		integer.Neg(integer)
		fractional.Neg(fractional)
	}

	return integer.Add(integer, fractional)
}
