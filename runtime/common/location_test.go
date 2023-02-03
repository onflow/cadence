/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocationEquality(t *testing.T) {

	t.Parallel()

	t.Run("AddressLocation", func(t *testing.T) {

		require.True(t,
			Location(AddressLocation{
				Address: Address{0x1},
				Name:    "A",
			}) ==
				Location(AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				}),
		)

		require.False(t,
			Location(AddressLocation{
				Address: Address{0x1},
				Name:    "A",
			}) ==
				Location(AddressLocation{
					Address: Address{0x2},
					Name:    "A",
				}),
		)

		require.False(t,
			Location(AddressLocation{
				Address: Address{0x1},
				Name:    "A",
			}) ==
				Location(AddressLocation{
					Address: Address{0x1},
					Name:    "B",
				}),
		)

		require.False(t,
			Location(AddressLocation{
				Address: Address{0x1},
				Name:    "A",
			}) ==
				Location(StringLocation("A.0000000000000001")),
		)

		require.False(t,
			Location(StringLocation("A.0000000000000001")) ==
				Location(AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				}),
		)

		require.True(t,
			Location(TransactionLocation{1}) == //nolint:gocritic
				Location(TransactionLocation{1}),
		)

		require.False(t,
			Location(TransactionLocation{1}) ==
				Location(ScriptLocation{1}),
		)
	})
}

func TestLocationsInSameAccount(t *testing.T) {

	t.Parallel()

	t.Run("AddressLocation", func(t *testing.T) {

		require.True(t,
			LocationsInSameAccount(
				AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				},
				AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				},
			),
		)

		require.False(t,
			LocationsInSameAccount(
				AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				},
				AddressLocation{
					Address: Address{0x2},
					Name:    "A",
				},
			),
		)

		require.True(t,
			LocationsInSameAccount(
				AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				},
				AddressLocation{
					Address: Address{0x1},
					Name:    "B",
				},
			),
		)

		require.False(t,
			LocationsInSameAccount(
				AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				},
				StringLocation("A.0000000000000001"),
			),
		)

		require.False(t,
			LocationsInSameAccount(
				StringLocation("A.0000000000000001"),
				AddressLocation{
					Address: Address{0x1},
					Name:    "A",
				},
			),
		)
	})
}
