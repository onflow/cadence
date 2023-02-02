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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := AddressLocation{
		Address: MustBytesToAddress([]byte{1}),
		Name:    "A",
	}

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "AddressLocation",
            "Address": "0x0000000000000001",
            "Name": "A"
        }
        `,
		string(actual),
	)
}

func TestAddressLocationTypeID(t *testing.T) {

	t.Parallel()

	location := AddressLocation{
		Address: MustBytesToAddress([]byte{1}),
		Name:    "Foo",
	}

	assert.Equal(t,
		TypeID("A.0000000000000001.Bar.Baz"),
		location.TypeID(nil, "Bar.Baz"),
	)
}

func TestDecodeAddressLocationTypeID(t *testing.T) {

	t.Parallel()

	t.Run("missing prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeAddressLocationTypeID(nil, "")
		require.EqualError(t, err, "invalid address location type ID: missing prefix")
	})

	t.Run("missing location", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeAddressLocationTypeID(nil, "A")
		require.EqualError(t, err, "invalid address location type ID: missing location")
	})

	t.Run("missing qualified identifier", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeAddressLocationTypeID(nil, "A.0000000000000001")
		require.EqualError(t, err, "invalid address location type ID: missing qualified identifier")
	})

	t.Run("invalid prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeAddressLocationTypeID(nil, "X.0000000000000001.T")
		require.EqualError(t, err, "invalid address location type ID: invalid prefix: expected \"A\", got \"X\"")
	})

	t.Run("qualified identifier with one part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeAddressLocationTypeID(nil, "A.0000000000000001.T")
		require.NoError(t, err)

		assert.Equal(t,
			AddressLocation{
				Address: MustBytesToAddress([]byte{1}),
				Name:    "T",
			},
			location,
		)
		assert.Equal(t, "T", qualifiedIdentifier)
	})

	t.Run("qualified identifier with two parts", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeAddressLocationTypeID(nil, "A.0000000000000001.T.U")
		require.NoError(t, err)

		assert.Equal(t,
			AddressLocation{
				Address: MustBytesToAddress([]byte{1}),
				Name:    "T",
			},
			location,
		)
		assert.Equal(t, "T.U", qualifiedIdentifier)
	})
}
