/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

func TestREPLLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := REPLLocation{}

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "REPLLocation"
        }
        `,
		string(actual),
	)
}

func TestREPLLocation_TypeID(t *testing.T) {

	t.Parallel()

	location := REPLLocation{}

	assert.Equal(t,
		TypeID("REPL.Bar.Baz"),
		location.TypeID(nil, "Bar.Baz"),
	)
}

func TestDecodeREPLLocationTypeID(t *testing.T) {

	t.Parallel()

	t.Run("missing prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeREPLLocationTypeID("")
		require.EqualError(t, err, "invalid REPL location type ID: missing prefix")
	})

	t.Run("missing qualified identifier", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeREPLLocationTypeID("REPL")
		require.EqualError(t, err, "invalid REPL location type ID: missing qualified identifier")
	})

	t.Run("missing qualified identifier", func(t *testing.T) {

		t.Parallel()

		_, _, err := decodeREPLLocationTypeID("X.T")
		require.EqualError(t, err, "invalid REPL location type ID: invalid prefix: expected \"REPL\", got \"X\"")
	})

	t.Run("qualified identifier with one part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeREPLLocationTypeID("REPL.T")
		require.NoError(t, err)

		assert.Equal(t,
			REPLLocation{},
			location,
		)
		assert.Equal(t, "T", qualifiedIdentifier)
	})

	t.Run("qualified identifier with two parts", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := decodeREPLLocationTypeID("REPL.T.U")
		require.NoError(t, err)

		assert.Equal(t,
			REPLLocation{},
			location,
		)
		assert.Equal(t, "T.U", qualifiedIdentifier)
	})
}
