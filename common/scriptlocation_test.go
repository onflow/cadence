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

func TestScriptLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := ScriptLocation{0x1, 0x2}

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ScriptLocation",
            "Script": "0102000000000000000000000000000000000000000000000000000000000000"
        }
        `,
		string(actual),
	)
}

func TestScriptLocationTypeID(t *testing.T) {

	t.Parallel()

	location := ScriptLocation{0x1, 0x2}

	assert.Equal(t,
		TypeID("s.0102000000000000000000000000000000000000000000000000000000000000.Bar.Baz"),
		location.TypeID(nil, "Bar.Baz"),
	)
}

func TestScriptLocationID(t *testing.T) {

	t.Parallel()

	location, _, err := decodeScriptLocationTypeID(
		nil,
		"s.0102000000000000000000000000000000000000000000000000000000000000.Bar.Baz",
	)
	require.NoError(t, err)

	assert.Equal(t,
		"s.0102000000000000000000000000000000000000000000000000000000000000",
		location.ID(),
	)
}

func TestDecodeScriptLocationTypeID(t *testing.T) {

	t.Parallel()

	t.Run("missing prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeScriptLocationTypeID(nil, "")
		require.EqualError(t, err, "invalid script location type ID: missing prefix")
	})

	t.Run("missing location", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeScriptLocationTypeID(nil, "s")
		require.EqualError(t, err, "invalid script location type ID: missing location")
	})

	t.Run("missing qualified identifier", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeScriptLocationTypeID(nil, "s.0102")
		require.NoError(t, err)

		assert.Equal(t,
			ScriptLocation{0x1, 0x2},
			location,
		)
		assert.Equal(t, "", qualifiedIdentifier)
	})

	t.Run("invalid prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeScriptLocationTypeID(nil, "X.0102.T")
		require.EqualError(t, err, "invalid script location type ID: invalid prefix: expected \"s\", got \"X\"")
	})

	t.Run("qualified identifier with one part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeScriptLocationTypeID(nil, "s.0102.T")
		require.NoError(t, err)

		assert.Equal(t,
			ScriptLocation{0x1, 0x2},
			location,
		)
		assert.Equal(t, "T", qualifiedIdentifier)
	})

	t.Run("qualified identifier with two parts", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeScriptLocationTypeID(nil, "s.0102.T.U")
		require.NoError(t, err)

		assert.Equal(t,
			ScriptLocation{0x1, 0x2},
			location,
		)
		assert.Equal(t, "T.U", qualifiedIdentifier)
	})
}
