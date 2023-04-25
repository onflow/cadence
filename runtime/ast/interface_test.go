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

	"github.com/onflow/cadence/runtime/common"
)

func TestInterfaceDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	t.Run("no conformances", func(t *testing.T) {

		decl := &InterfaceDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
			Members:   NewUnmeteredMembers([]Declaration{}),
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
            "Type": "InterfaceDeclaration",
            "Access": "AccessPublic",
            "CompositeKind": "CompositeKindResource",
            "Identifier": {
                "Identifier": "AB",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
            "Conformances": null,
            "Members": {
                "Declarations": []
            },
            "DocString": "test",
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
			string(actual),
		)
	})

	t.Run("with conformances", func(t *testing.T) {

		decl := &InterfaceDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
						Pos:        Position{Offset: 4, Line: 5, Column: 6},
					},
				},
			},
			Members:   NewUnmeteredMembers([]Declaration{}),
			DocString: "test",
			Range: Range{
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
				EndPos:   Position{Offset: 10, Line: 11, Column: 12},
			},
		}

		actual, err := json.Marshal(decl)
		require.NoError(t, err)

		assert.JSONEq(t,
			`
        {
            "Type": "InterfaceDeclaration",
            "Access": "AccessPublic", 
            "CompositeKind": "CompositeKindResource",
            "Identifier": {
                "Identifier": "AB",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
            "Conformances": [
                {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "CD",
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                    },
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                }
            ],
            "Members": {
                "Declarations": []
            },
            "DocString": "test",
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
			string(actual),
		)
	})
}

func TestInterfaceDeclaration_Doc(t *testing.T) {

	t.Parallel()

	t.Run("no members", func(t *testing.T) {

		t.Parallel()

		decl := &InterfaceDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Members: NewMembers(nil, []Declaration{}),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("pub"),
				prettier.Text(" "),
				prettier.Text("resource"),
				prettier.Text(" "),
				prettier.Text("interface "),
				prettier.Text("AB"),
				prettier.Text(" "),
				prettier.Text("{}"),
			},
			decl.Doc(),
		)

	})

	t.Run("members", func(t *testing.T) {

		t.Parallel()

		decl := &InterfaceDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Members: NewMembers(nil, []Declaration{
				&FieldDeclaration{
					Identifier: Identifier{
						Identifier: "x",
					},
					TypeAnnotation: &TypeAnnotation{
						Type: &NominalType{
							Identifier: Identifier{
								Identifier: "X",
							},
						},
					},
				},
			}),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("pub"),
				prettier.Text(" "),
				prettier.Text("resource"),
				prettier.Text(" "),
				prettier.Text("interface "),
				prettier.Text("AB"),
				prettier.Text(" "),
				prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.HardLine{},
							prettier.Group{
								Doc: prettier.Concat{
									prettier.Text("x"),
									prettier.Text(": "),
									prettier.Text("X"),
								},
							},
						},
					},
					prettier.HardLine{},
					prettier.Text("}"),
				},
			},
			decl.Doc(),
		)

	})
}

func TestInterfaceDeclaration_String(t *testing.T) {

	t.Parallel()

	t.Run("no members", func(t *testing.T) {

		t.Parallel()

		decl := &InterfaceDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Members: NewMembers(nil, []Declaration{}),
		}

		require.Equal(
			t,
			"pub resource interface AB {}",
			decl.String(),
		)

	})

	t.Run("members", func(t *testing.T) {

		t.Parallel()

		decl := &InterfaceDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Members: NewMembers(nil, []Declaration{
				&FieldDeclaration{
					Identifier: Identifier{
						Identifier: "x",
					},
					TypeAnnotation: &TypeAnnotation{
						Type: &NominalType{
							Identifier: Identifier{
								Identifier: "X",
							},
						},
					},
				},
			}),
		}

		require.Equal(
			t,
			"pub resource interface AB {\n"+
				"    x: X\n"+
				"}",
			decl.String(),
		)

	})
}
