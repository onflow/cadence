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

package fixedpoint

import (
	"errors"
	"math/big"
	"strings"
)

func ParseFix64(s string) (*big.Int, error) {
	negative, unsignedInteger, fractional, parsedScale, err := parseFixedPoint(s)
	if err != nil {
		return nil, err
	}

	return NewFix64(negative, unsignedInteger, fractional, parsedScale)
}

func NewFix64(
	negative bool,
	unsignedInteger *big.Int,
	fractional *big.Int,
	parsedScale uint,
) (
	*big.Int,
	error,
) {
	return checkAndConvertFixedPoint(
		negative,
		unsignedInteger,
		fractional,
		parsedScale,
		Fix64Scale,
		Fix64TypeMinIntBig, Fix64TypeMinFractionalBig,
		Fix64TypeMaxIntBig, Fix64TypeMaxFractionalBig,
	)
}

func ParseUFix64(s string) (*big.Int, error) {
	negative, unsignedInteger, fractional, parsedScale, err := parseFixedPoint(s)
	if err != nil {
		return nil, err
	}

	if negative {
		return nil, errors.New("invalid negative integer part")
	}

	return NewUFix64(unsignedInteger, fractional, parsedScale)
}

func NewUFix64(
	unsignedInteger *big.Int,
	fractional *big.Int,
	parsedScale uint,
) (
	*big.Int,
	error,
) {
	return checkAndConvertFixedPoint(
		false,
		unsignedInteger,
		fractional,
		parsedScale,
		Fix64Scale,
		UFix64TypeMinIntBig, UFix64TypeMinFractionalBig,
		UFix64TypeMaxIntBig, UFix64TypeMaxFractionalBig,
	)
}

func parseFixedPoint(v string) (
	negative bool,
	unsignedInteger,
	fractional *big.Int,
	scale uint,
	err error,
) {
	// must contain single radix point
	parts := strings.Split(v, ".")
	if len(parts) != 2 {
		err = errors.New("missing decimal point")
		return
	}

	integerStr := parts[0]
	fractionalStr := parts[1]

	scale = uint(len(fractionalStr))

	negative = len(v) > 0 && v[0] == '-'

	integer, ok := new(big.Int).SetString(integerStr, 10)
	if !ok {
		err = errors.New("invalid integer part")
		return
	}

	if len(fractionalStr) > 0 {
		switch fractionalStr[0] {
		case '+', '-':
			err = errors.New("invalid sign in fractional part")
			return
		}
	}

	fractional, ok = new(big.Int).SetString(fractionalStr, 10)
	if !ok {
		err = errors.New("invalid fractional part")
		return
	}

	if integer.Sign() < 0 {
		unsignedInteger = integer.Neg(integer)
	} else {
		unsignedInteger = integer
	}

	return
}

func checkAndConvertFixedPoint(
	negative bool,
	unsignedInteger,
	fractional *big.Int,
	parsedScale uint,
	targetScale uint,
	minInteger, minFractional,
	maxInteger, maxFractional *big.Int,
) (
	*big.Int,
	error,
) {
	if parsedScale > targetScale {
		return nil, errors.New("invalid scale")
	}

	inRange := CheckRange(
		negative,
		unsignedInteger,
		fractional,
		minInteger, minFractional,
		maxInteger, maxFractional,
	)

	if !inRange {
		return nil, errors.New("out of range")
	}

	return ConvertToFixedPointBigInt(
		negative,
		unsignedInteger,
		fractional,
		parsedScale,
		targetScale,
	), nil
}
