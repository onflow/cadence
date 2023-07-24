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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"
)

func TestEntitlementDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &EntitlementDeclaration{
		Access: AccessAll,
		Identifier: Identifier{
			Identifier: "AB",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "EntitlementDeclaration",
            "Access": "AccessAll", 
            "Identifier": {
                "Identifier": "AB",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
            "DocString": "test",
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestEntitlementDeclaration_Doc(t *testing.T) {

	t.Parallel()

	t.Run("no members", func(t *testing.T) {

		t.Parallel()

		decl := &EntitlementDeclaration{
			Access: AccessAll,
			Identifier: Identifier{
				Identifier: "AB",
			},
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("access(all)"),
				prettier.HardLine{},
				prettier.Text("entitlement "),
				prettier.Text("AB"),
			},
			decl.Doc(),
		)

	})
}

func TestEntitlementDeclaration_String(t *testing.T) {

	t.Parallel()

	t.Run("no members", func(t *testing.T) {

		t.Parallel()

		decl := &EntitlementDeclaration{
			Access: AccessAll,
			Identifier: Identifier{
				Identifier: "AB",
			},
		}

		require.Equal(
			t,
			`access(all)
entitlement AB`,
			decl.String(),
		)

	})
}

func TestEntitlementMappingDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &EntitlementMappingDeclaration{
		Access: AccessAll,
		Identifier: Identifier{
			Identifier: "AB",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
		Associations: []*EntitlementMapElement{
			{
				Input: &NominalType{
					Identifier: Identifier{
						Identifier: "X",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
				},
				Output: &NominalType{
					Identifier: Identifier{
						Identifier: "Y",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
				},
			},
		},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "EntitlementMappingDeclaration",
            "Access": "AccessAll", 
            "Identifier": {
                "Identifier": "AB",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
			"Associations": [
				{
					"Input": {
						"Type": "NominalType",
						"Identifier": {
							"Identifier": "X",
							"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
							"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
						},
						"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
						"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
					},
					"Output":  {
						"Type": "NominalType",
						"Identifier": {
							"Identifier": "Y",
							"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
							"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
						},
						"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
						"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
					}
				}
			],
            "DocString": "test",
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestEntitlementMappingDeclaration_Doc(t *testing.T) {

	t.Parallel()

	decl := &EntitlementMappingDeclaration{
		Access: AccessAll,
		Identifier: Identifier{
			Identifier: "AB",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
		Associations: []*EntitlementMapElement{
			{
				Input: &NominalType{
					Identifier: Identifier{
						Identifier: "X",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
				},
				Output: &NominalType{
					Identifier: Identifier{
						Identifier: "Y",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
				},
			},
		},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("access(all)"),
			prettier.HardLine{},
			prettier.Text("entitlement "),
			prettier.Text("mapping "),
			prettier.Text("AB"),
			prettier.Space,
			prettier.Text("{"),
			prettier.Indent{
				Doc: prettier.Concat{
					prettier.HardLine{},
					prettier.Concat{
						prettier.Text("X"),
						prettier.Text(" -> "),
						prettier.Text("Y"),
					},
				},
			},
			prettier.HardLine{},
			prettier.Text("}"),
		},
		decl.Doc(),
	)

}

func TestEntitlementMappingDeclaration_String(t *testing.T) {

	t.Parallel()

	decl := &EntitlementMappingDeclaration{
		Access: AccessAll,
		Identifier: Identifier{
			Identifier: "AB",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
		Associations: []*EntitlementMapElement{
			{
				Input: &NominalType{
					Identifier: Identifier{
						Identifier: "X",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
				},
				Output: &NominalType{
					Identifier: Identifier{
						Identifier: "Y",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
				},
			},
		},
	}

	require.Equal(
		t,
		`access(all)
entitlement mapping AB {
    X -> Y
}`,
		decl.String(),
	)

}
