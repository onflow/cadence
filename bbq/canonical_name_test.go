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

package bbq_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
)

func TestCanonicalNameMapIdentityIncludesCompleteLocation(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{1})
	first := bbq.NewCanonicalName(
		common.NewAddressLocation(nil, address, "First"),
		"value",
	)
	second := bbq.NewCanonicalName(
		common.NewAddressLocation(nil, address, "Second"),
		"value",
	)

	// Address-location TypeIDs omit the program name.
	require.Equal(t, first.String(), second.String())
	require.NotEqual(t, first, second)

	values := map[bbq.CanonicalName]int{
		first:  1,
		second: 2,
	}
	require.Len(t, values, 2)
}
