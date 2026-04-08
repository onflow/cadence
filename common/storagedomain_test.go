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

package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStorageDomainValues(t *testing.T) {
	t.Parallel()

	// Ensure that the values of the StorageDomain enum are not accidentally changed,
	// e.g. by adding a new value in between or by changing an existing value.

	expectedValues := map[StorageDomain]uint8{
		StorageDomainUnknown:                 0,
		StorageDomainPathStorage:             1,
		StorageDomainPathPrivate:             2,
		StorageDomainPathPublic:              3,
		StorageDomainContract:                4,
		StorageDomainInbox:                   5,
		StorageDomainCapabilityController:    6,
		StorageDomainCapabilityControllerTag: 7,
		StorageDomainPathCapability:          8,
		StorageDomainAccountCapability:       9,
		StorageDomain_Count:                  10,
	}

	// Check all expected values.
	for domain, expectedValue := range expectedValues {
		require.Equal(t, expectedValue, uint8(domain), "value mismatch for %s", domain)
	}

	// Check that no new named values have been added
	// without updating the expected values above.
	// If a placeholder `_` is replaced with a new named value,
	// its String() representation will no longer be a numeric fallback.
	// NOTE: This requires the stringer-generated file to be up to date (CI runs go generate).
	for i := uint8(0); i < uint8(StorageDomain_Count); i++ {

		domain := StorageDomain(i)
		if _, ok := expectedValues[domain]; ok {
			continue
		}

		require.True(t,
			strings.HasPrefix(domain.String(), "StorageDomain("),
			"unexpected named value %s (%d): update expectedValues", domain, i,
		)
	}
}
