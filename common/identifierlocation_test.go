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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdentifierLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := IdentifierLocation("test")

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "IdentifierLocation",
            "Identifier": "test"
        }
        `,
		string(actual),
	)
}

func TestIdentifierLocation_TypeID(t *testing.T) {

	t.Parallel()

	location := IdentifierLocation("foo")

	assert.Equal(t,
		TypeID("I.foo.Bar.Baz"),
		location.TypeID(nil, "Bar.Baz"),
	)
}

func TestIdentifierLocation_ID(t *testing.T) {

	t.Parallel()

	location, _, err := decodeIdentifierLocationTypeID(nil, "I.foo.Bar.Baz")
	require.NoError(t, err)

	assert.Equal(t,
		"I.foo",
		location.ID(),
	)
}

func TestDecodeIdentifierLocationTypeID(t *testing.T) {

	t.Parallel()

	t.Run("missing prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeIdentifierLocationTypeID(nil, "")
		require.EqualError(t, err, "invalid identifier location type ID: missing prefix")
	})

	t.Run("missing location", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeIdentifierLocationTypeID(nil, "I")
		require.EqualError(t, err, "invalid identifier location type ID: missing location")
	})

	t.Run("missing qualified identifier part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeIdentifierLocationTypeID(nil, "I.test")
		require.NoError(t, err)

		assert.Equal(t,
			IdentifierLocation("test"),
			location,
		)
		assert.Equal(t, "", qualifiedIdentifier)
	})

	t.Run("empty qualified identifier", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeIdentifierLocationTypeID(nil, "I.test.")
		require.NoError(t, err)

		assert.Equal(t,
			IdentifierLocation("test"),
			location,
		)
		assert.Equal(t, "", qualifiedIdentifier)
	})

	t.Run("invalid prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeIdentifierLocationTypeID(nil, "X.test.T")
		require.EqualError(t, err, "invalid identifier location type ID: invalid prefix: expected \"I\", got \"X\"")
	})

	t.Run("qualified identifier with one part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeIdentifierLocationTypeID(nil, "I.test.T")
		require.NoError(t, err)

		assert.Equal(t,
			IdentifierLocation("test"),
			location,
		)
		assert.Equal(t, "T", qualifiedIdentifier)
	})

	t.Run("qualified identifier with two parts", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeIdentifierLocationTypeID(nil, "I.test.T.U")
		require.NoError(t, err)

		assert.Equal(t,
			IdentifierLocation("test"),
			location,
		)
		assert.Equal(t, "T.U", qualifiedIdentifier)
	})
}
