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

package runtime_utils

import (
	"testing"

	"github.com/onflow/atree"
	"github.com/stretchr/testify/require"
)

func CheckAtreeStorageHealth(tb testing.TB, storage atree.SlabStorage, expectedRootSlabIDs []atree.SlabID) {
	rootSlabIDs, err := atree.CheckStorageHealth(storage, -1)
	require.NoError(tb, err)

	nonTempRootSlabIDs := make([]atree.SlabID, 0, len(rootSlabIDs))

	for rootSlabID := range rootSlabIDs {
		if rootSlabID.HasTempAddress() {
			continue
		}
		nonTempRootSlabIDs = append(nonTempRootSlabIDs, rootSlabID)
	}

	require.ElementsMatch(tb, nonTempRootSlabIDs, expectedRootSlabIDs)
}
