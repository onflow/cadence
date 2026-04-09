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

package interpreter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashInputTypeValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the HashInputType enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[HashInputType]byte{
		HashInputTypeBool:      0,
		HashInputTypeString:    1,
		HashInputTypeEnum:      2,
		HashInputTypeAddress:   3,
		HashInputTypePath:      4,
		HashInputTypeType:      5,
		HashInputTypeCharacter: 6,
		HashInputTypeInt:       10,
		HashInputTypeInt8:      11,
		HashInputTypeInt16:     12,
		HashInputTypeInt32:     13,
		HashInputTypeInt64:     14,
		HashInputTypeInt128:    15,
		HashInputTypeInt256:    16,
		HashInputTypeUInt:      18,
		HashInputTypeUInt8:     19,
		HashInputTypeUInt16:    20,
		HashInputTypeUInt32:    21,
		HashInputTypeUInt64:    22,
		HashInputTypeUInt128:   23,
		HashInputTypeUInt256:   24,
		HashInputTypeWord8:     27,
		HashInputTypeWord16:    28,
		HashInputTypeWord32:    29,
		HashInputTypeWord64:    30,
		HashInputTypeWord128:   31,
		HashInputTypeWord256:   32,
		HashInputTypeFix64:     38,
		HashInputTypeFix128:    39,
		HashInputTypeUFix64:    46,
		HashInputTypeUFix128:   47,
		HashInputType_Count:    50,
	}

	// Check all expected values.
	for typ, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, byte(typ), "value mismatch for %s", typ)
	}

	// Check that no new named values have been added
	// without updating the expected values above.
	// If a placeholder `_` is replaced with a new named value,
	// its String() representation will no longer be a numeric fallback.
	// NOTE: This requires the stringer-generated file to be up to date (CI runs go generate).
	for i := byte(0); i < byte(HashInputType_Count); i++ {

		typ := HashInputType(i)
		if _, ok := expectedValues[typ]; ok {
			continue
		}

		require.True(t,
			strings.HasPrefix(typ.String(), "HashInputType("),
			"unexpected named value %s (%d): update expectedValues", typ, i,
		)
	}
}
