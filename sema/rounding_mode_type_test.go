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

package sema

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundingModeValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the RoundingMode enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[RoundingMode]uint8{
		RoundingModeTowardZero:      0,
		RoundingModeAwayFromZero:    1,
		RoundingModeNearestHalfAway: 2,
		RoundingModeNearestHalfEven: 3,
		RoundingMode_Count:          4,
	}

	// Check all expected values.
	for mode, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, uint8(mode), "value mismatch for %d", mode)
	}

	// Check that no new values have been added
	// without updating the expected values above.
	for i := uint8(0); i < uint8(RoundingMode_Count); i++ {
		mode := RoundingMode(i)
		_, ok := expectedValues[mode]
		require.True(t, ok,
			fmt.Sprintf("unexpected RoundingMode value %d: update expectedValues", i),
		)
	}
}
