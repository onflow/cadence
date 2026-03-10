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

func TestMappedAccess_MarshalJSON(t *testing.T) {

	t.Parallel()

	e := NewNominalType(nil, NewIdentifier(nil, "E", Position{Offset: 1, Line: 2, Column: 3}), []Identifier{})

	access := NewMappedAccess(e, Position{Offset: 0, Line: 0, Column: 0})
	actual, err := json.Marshal(access)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
            {
                "EntitlementMap": {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "E",
                        "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                        "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
                    },
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
                },
                "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
            }
        `,
		string(actual),
	)
}

func TestMappedAccess_Walk(t *testing.T) {

	t.Parallel()

	mapType := &NominalType{
		Identifier: Identifier{Identifier: "M"},
	}

	access := NewMappedAccess(mapType, Position{})

	var visited []Element
	access.Walk(func(element Element) {
		visited = append(visited, element)
	})

	assert.Equal(t, []Element{mapType}, visited)
}

func TestConjunctiveEntitlementSet_MarshalJSON(t *testing.T) {

	t.Parallel()

	e := NewNominalType(nil, NewIdentifier(nil, "E", Position{Offset: 0, Line: 0, Column: 0}), []Identifier{})
	f := NewNominalType(nil, NewIdentifier(nil, "F", Position{Offset: 1, Line: 2, Column: 3}), []Identifier{})

	access := NewConjunctiveEntitlementSet([]*NominalType{e, f})
	actual, err := json.Marshal(access)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
            {
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
            }
        `,
		string(actual),
	)
}

func TestConjunctiveEntitlementSet_Walk(t *testing.T) {

	t.Parallel()

	e := &NominalType{
		Identifier: Identifier{Identifier: "E"},
	}
	f := &NominalType{
		Identifier: Identifier{Identifier: "F"},
	}

	set := NewConjunctiveEntitlementSet([]*NominalType{e, f})

	var visited []Element
	set.Walk(func(element Element) {
		visited = append(visited, element)
	})

	assert.Equal(t,
		[]Element{
			e,
			f,
		},
		visited,
	)
}

func TestDisjunctiveEntitlementSet_MarshalJSON(t *testing.T) {

	t.Parallel()

	e := NewNominalType(nil, NewIdentifier(nil, "E", Position{Offset: 0, Line: 0, Column: 0}), []Identifier{})
	f := NewNominalType(nil, NewIdentifier(nil, "F", Position{Offset: 1, Line: 2, Column: 3}), []Identifier{})

	access := NewDisjunctiveEntitlementSet([]*NominalType{e, f})
	actual, err := json.Marshal(access)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
            {
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
            }
        `,
		string(actual),
	)
}

func TestDisjunctiveEntitlementSet_Walk(t *testing.T) {

	t.Parallel()

	e := &NominalType{
		Identifier: Identifier{Identifier: "E"},
	}
	f := &NominalType{
		Identifier: Identifier{Identifier: "F"},
	}

	set := NewDisjunctiveEntitlementSet([]*NominalType{e, f})

	var visited []Element
	set.Walk(func(element Element) {
		visited = append(visited, element)
	})

	assert.Equal(t,
		[]Element{
			e,
			f,
		},
		visited,
	)
}
