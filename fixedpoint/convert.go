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
