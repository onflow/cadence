package fixedpoint_test

import (
	"math/big"
	"testing"

	fix "github.com/onflow/fixed-point"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/fixedpoint"
)

func TestFix128(t *testing.T) {

	t.Parallel()

	t.Run("big.Int to fix128 roundtrip", func(t *testing.T) {
		t.Parallel()

		for _, bigInt := range []*big.Int{
			// 1
			new(big.Int).Mul(
				big.NewInt(1),
				fixedpoint.Fix128FactorAsBigInt,
			),

			// -1
			new(big.Int).Mul(
				big.NewInt(-1),
				fixedpoint.Fix128FactorAsBigInt,
			),

			// -12.34
			new(big.Int).Mul(
				big.NewInt(-1234),
				new(big.Int).Exp(
					big.NewInt(10),
					big.NewInt(22),
					nil,
				),
			),

			// Max fix128
			func() *big.Int {
				b, _ := new(big.Int).SetString("170141183460469231731687303715884105727", 10)
				require.NotNil(t, b)
				return b
			}(),

			// Min fix128
			func() *big.Int {
				b, _ := new(big.Int).SetString("-170141183460469231731687303715884105728", 10)
				require.NotNil(t, b)
				return b
			}(),
		} {
			originalBigInt := bigInt

			t.Run(bigInt.String(), func(t *testing.T) {
				t.Parallel()

				fix128 := fixedpoint.Fix128FromBigInt(originalBigInt)

				convertedBigInt := fixedpoint.Fix128ToBigInt(fix128)

				require.Equal(t, originalBigInt, convertedBigInt)
			})
		}
	})

	t.Run("fix128 as bigInt from parts", func(t *testing.T) {
		t.Parallel()

		// -12.34

		expected := new(big.Int).Mul(
			big.NewInt(-1234),
			new(big.Int).Exp(
				big.NewInt(10),
				big.NewInt(22),
				nil,
			),
		)

		convertedBigInt := fixedpoint.ConvertToFixedPointBigInt(
			true,
			big.NewInt(12),
			big.NewInt(34),
			2,
			fixedpoint.Fix128Scale,
		)

		require.Equal(t, expected, convertedBigInt)
	})

	t.Run("fix128 to bigInt roundtrip", func(t *testing.T) {
		t.Parallel()

		type testCase struct {
			value fix.Fix128
			str   string
		}

		for _, fix128 := range []testCase{
			{
				value: fix.NewFix128(0xFFFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF),
				str:   "-1",
			},
			{
				value: fix.NewFix128(0x0000000000000000, 0x0000000000000001),
				str:   "1",
			},
			{
				value: fix.NewFix128(0xFFFFFFFFFFFFFFFF, 0x0000000000000000),
				str:   "-18446744073709551616",
			},
			{
				value: fix.NewFix128(0x0000000000000000, 0xFFFFFFFFFFFFFFFF),
				str:   "18446744073709551615",
			},
			// Max 128
			{
				value: fix.NewFix128(0x7FFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF),
				str:   "170141183460469231731687303715884105727",
			},
			// Min 128
			{
				value: fix.NewFix128(0x8000000000000000, 0x0000000000000000),
				str:   "-170141183460469231731687303715884105728",
			},
		} {

			fix128 := fix128

			t.Run(fix128.str, func(t *testing.T) {
				t.Parallel()

				originalFix128 := fix128.value

				convertedBigInt := fixedpoint.Fix128ToBigInt(originalFix128)

				convertedFix128 := fixedpoint.Fix128FromBigInt(convertedBigInt)

				require.Equal(t, originalFix128, convertedFix128)
				require.Equal(t, fix128.str, convertedBigInt.String())
			})
		}
	})
}
