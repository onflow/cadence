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

	return checkAndConvertFixedPoint(
		negative,
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

	negative = false

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
		negative = true
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
