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

package stdlib_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/onflow/cadence/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

func TestFlowEventTypeIDs(t *testing.T) {

	t.Parallel()

	for _, ty := range []sema.Type{
		stdlib.AccountCreatedEventType,
		stdlib.AccountKeyAddedFromPublicKeyEventType,
		stdlib.AccountKeyRemovedFromPublicKeyIndexEventType,
		stdlib.AccountContractAddedEventType,
		stdlib.AccountContractUpdatedEventType,
		stdlib.AccountContractRemovedEventType,
	} {
		assert.True(t, strings.HasPrefix(string(ty.ID()), "flow"))
	}
}

func TestFlowLocation_MarshalJSON(t *testing.T) {

	t.Parallel()

	loc := stdlib.FlowLocation{}

	actual, err := json.Marshal(loc)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "FlowLocation"
        }
        `,
		string(actual),
	)
}

func TestFlowLocationTypeID(t *testing.T) {

	t.Parallel()

	var location stdlib.FlowLocation

	assert.Equal(t,
		common.TypeID("flow.Bar.Baz"),
		location.TypeID(nil, "Bar.Baz"),
	)
}

func TestFlowLocationID(t *testing.T) {

	t.Parallel()

	location, _, err := stdlib.DecodeFlowLocationTypeID("flow.Bar.Baz")
	require.NoError(t, err)

	assert.Equal(t,
		"flow",
		location.ID(),
	)
}

func TestDecodeFlowLocationTypeID(t *testing.T) {

	t.Parallel()

	t.Run("missing prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := stdlib.DecodeFlowLocationTypeID("")
		require.EqualError(t, err, "invalid Flow location type ID: missing prefix")
	})

	t.Run("missing qualified identifier part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := stdlib.DecodeFlowLocationTypeID("flow")
		require.NoError(t, err)

		assert.Equal(t,
			stdlib.FlowLocation{},
			location,
		)
		assert.Equal(t, "", qualifiedIdentifier)
	})

	t.Run("empty qualified identifier", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := stdlib.DecodeFlowLocationTypeID("flow.")
		require.NoError(t, err)

		assert.Equal(t,
			stdlib.FlowLocation{},
			location,
		)
		assert.Equal(t, "", qualifiedIdentifier)
	})

	t.Run("invalid prefix", func(t *testing.T) {

		t.Parallel()

		_, _, err := stdlib.DecodeFlowLocationTypeID("X.T")
		require.EqualError(t, err, "invalid Flow location type ID: invalid prefix: expected \"flow\", got \"X\"")
	})

	t.Run("qualified identifier with one part", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := stdlib.DecodeFlowLocationTypeID("flow.T")
		require.NoError(t, err)

		assert.Equal(t,
			stdlib.FlowLocation{},
			location,
		)
		assert.Equal(t, "T", qualifiedIdentifier)
	})

	t.Run("qualified identifier with two parts", func(t *testing.T) {

		t.Parallel()

		location, qualifiedIdentifier, err := stdlib.DecodeFlowLocationTypeID("flow.T.U")
		require.NoError(t, err)

		assert.Equal(t,
			stdlib.FlowLocation{},
			location,
		)
		assert.Equal(t, "T.U", qualifiedIdentifier)
	})
}
