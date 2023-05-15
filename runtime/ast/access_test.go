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

package ast

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrimitiveAccess_MarshalJSON(t *testing.T) {

	t.Parallel()

	for access := PrimitiveAccess(0); access < PrimitiveAccess(PrimitiveAccessCount()); access++ {
		actual, err := json.Marshal(access)
		require.NoError(t, err)

		assert.JSONEq(t, fmt.Sprintf(`"%s"`, access), string(actual))
	}
}

func TestEntitlementAccess_MarshalJSON(t *testing.T) {

	t.Parallel()

	e := NewNominalType(nil, NewIdentifier(nil, "E", Position{Offset: 0, Line: 0, Column: 0}), []Identifier{})
	f := NewNominalType(nil, NewIdentifier(nil, "F", Position{Offset: 1, Line: 2, Column: 3}), []Identifier{})

	t.Run("conjunction", func(t *testing.T) {
		t.Parallel()

		access := NewConjunctiveEntitlementSet([]*NominalType{e, f})
		actual, err := json.Marshal(access)
		require.NoError(t, err)

		assert.JSONEq(t, `{
			"ConjunctiveElements": [
				{
					"Type": "NominalType",
					"Identifier": {
						"Identifier": "E",
						"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
						"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
					},
					"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
					"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
				},
				{
					"Type": "NominalType",
					"Identifier": {
						"Identifier": "F",
						"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
						"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
					},
					"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
					"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
				}
			]
		}`, string(actual))
	})

	t.Run("disjunction", func(t *testing.T) {
		t.Parallel()

		access := NewDisjunctiveEntitlementSet([]*NominalType{e, f})
		actual, err := json.Marshal(access)
		require.NoError(t, err)

		assert.JSONEq(t, `{
			"DisjunctiveElements": [
				{
					"Type": "NominalType",
					"Identifier": {
						"Identifier": "E",
						"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
						"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
					},
					"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
					"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
				},
				{
					"Type": "NominalType",
					"Identifier": {
						"Identifier": "F",
						"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
						"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
					},
					"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
					"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
				}
			]
		}`, string(actual))
	})
}
