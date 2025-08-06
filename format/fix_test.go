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

package format

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/fixedpoint"
)

func TestUFix64(t *testing.T) {

	t.Parallel()

	require.Equal(t, "99999999999.70000000", UFix64(9999999999970000000))
}

func TestFix128(t *testing.T) {

	t.Parallel()

	t.Run("big.Int to fix128 roundtrip", func(t *testing.T) {
		t.Parallel()

		// -12.34
		originalBigInt := new(big.Int).Mul(
			big.NewInt(-1234),
			new(big.Int).Exp(
				big.NewInt(10),
				big.NewInt(22),
				nil,
			),
		)

		fix128 := fixedpoint.Fix128FromBigInt(originalBigInt)

		convertedBigInt := fixedpoint.Fix128ToBigInt(fix128)

		require.Equal(t, originalBigInt, convertedBigInt)
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
}
