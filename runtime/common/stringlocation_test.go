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

func TestStringLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := StringLocation("test")

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "StringLocation",
            "String": "test"
        }
        `,
		string(actual),
	)
}

func TestStringLocation_TypeID(t *testing.T) {

	t.Parallel()

	location := StringLocation("foo")

	assert.Equal(t,
		TypeID("S.foo.Bar.Baz"),
		location.TypeID(nil, "Bar.Baz"),
	)
}

func TestStringLocation_ID(t *testing.T) {

	t.Parallel()

	location, _, err := decodeStringLocationTypeID(nil, "S.foo.Bar.Baz")
	require.NoError(t, err)

	assert.Equal(t,
		"S.foo",
		location.ID(),
	)
}

func TestDecodeStringLocationTypeID(t *testing.T) {

	t.Parallel()

	t.Run("missing prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeStringLocationTypeID(nil, "")
		require.EqualError(t, err, "invalid string location type ID: missing prefix")
	})

	t.Run("missing location", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeStringLocationTypeID(nil, "S")
		require.EqualError(t, err, "invalid string location type ID: missing location")
	})

	t.Run("missing qualified identifier part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeStringLocationTypeID(nil, "S.test")
		require.NoError(t, err)

		assert.Equal(t,
			StringLocation("test"),
			location,
		)
		assert.Equal(t, "", qualifiedIdentifier)
	})

	t.Run("empty qualified identifier", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeStringLocationTypeID(nil, "S.test.")
		require.NoError(t, err)

		assert.Equal(t,
			StringLocation("test"),
			location,
		)
		assert.Equal(t, "", qualifiedIdentifier)
	})

	t.Run("invalid prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeStringLocationTypeID(nil, "X.test.T")
		require.EqualError(t, err, "invalid string location type ID: invalid prefix: expected \"S\", got \"X\"")
	})

	t.Run("qualified identifier with one part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeStringLocationTypeID(nil, "S.test.T")
		require.NoError(t, err)

		assert.Equal(t,
			StringLocation("test"),
			location,
		)
		assert.Equal(t, "T", qualifiedIdentifier)
	})

	t.Run("qualified identifier with two parts", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeStringLocationTypeID(nil, "S.test.T.U")
		require.NoError(t, err)

		assert.Equal(t,
			StringLocation("test"),
			location,
		)
		assert.Equal(t, "T.U", qualifiedIdentifier)
	})
}
