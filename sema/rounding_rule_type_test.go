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

func TestRoundingRuleValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the RoundingRule enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[RoundingRule]uint8{
		RoundingRuleTowardZero:      0,
		RoundingRuleAwayFromZero:    1,
		RoundingRuleNearestHalfAway: 2,
		RoundingRuleNearestHalfEven: 3,
		RoundingRule_Count:          4,
	}

	// Check all expected values.
	for rule, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, uint8(rule), "value mismatch for %d", rule)
	}

	// Check that no new values have been added
	// without updating the expected values above.
	for i := uint8(0); i < uint8(RoundingRule_Count); i++ {
		rule := RoundingRule(i)
		_, ok := expectedValues[rule]
		require.True(t, ok,
			fmt.Sprintf("unexpected RoundingRule value %d: update expectedValues", i),
		)
	}
}
