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

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		require.Equal(
			t,
			"-170141183460469.231731687303715884105728",
			Fix128(fixedpoint.Fix128TypeMin),
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		require.Equal(
			t,
			"170141183460469.231731687303715884105727",
			Fix128(fixedpoint.Fix128TypeMax),
		)
	})
}

func TestUFix128(t *testing.T) {

	t.Parallel()

	t.Run("min", func(t *testing.T) {
		t.Parallel()

		require.Equal(
			t,
			"0.000000000000000000000000",
			UFix128(fixedpoint.UFix128TypeMin),
		)
	})

	t.Run("max", func(t *testing.T) {
		t.Parallel()

		require.Equal(
			t,
			"340282366920938.463463374607431768211455",
			UFix128(fixedpoint.UFix128TypeMax),
		)
	})
}
