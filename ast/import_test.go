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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"

	"github.com/onflow/cadence/common"
)

func TestImportDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &ImportDeclaration{
		Imports: []Import{
			{
				Identifier: Identifier{
					Identifier: "foo",
					Pos:        Position{Offset: 1, Line: 2, Column: 3},
				},
			},
			{
				Identifier: Identifier{
					Identifier: "bar",
					Pos:        Position{Offset: 4, Line: 5, Column: 6},
				},
				Alias: Identifier{
					Identifier: "baz",
					Pos:        Position{Offset: 7, Line: 8, Column: 9},
				},
			},
		},
		Location:    common.StringLocation("test"),
		LocationPos: Position{Offset: 10, Line: 11, Column: 12},
		Range: Range{
			StartPos: Position{Offset: 13, Line: 14, Column: 15},
			EndPos:   Position{Offset: 16, Line: 17, Column: 18},
		},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "ImportDeclaration",
            "Comments": {},
            "Imports": [
                {
                    "Identifier": {
                        "Identifier": "foo",
                        "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                        "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                    }
                },
                {
                    "Identifier": {
                        "Identifier": "bar",
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 6, "Line": 5, "Column": 8}
                    },
                    "Alias": {
                        "Identifier": "baz",
                        "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                        "EndPos": {"Offset": 9, "Line": 8, "Column": 11}
                    }
                }
            ],
            "Location": {
                "Type": "StringLocation",
                "String": "test"
            },
            "LocationPos": {"Offset": 10, "Line": 11, "Column": 12},
            "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
            "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
        }
        `,
		string(actual),
	)
}

func TestImportDeclaration_Doc(t *testing.T) {

	t.Parallel()

	t.Run("no identifiers", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Location: common.StringLocation("test"),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("import"),
				prettier.Space,
				prettier.Text("\"test\""),
			},
			decl.Doc(),
		)
	})

	t.Run("one identifier", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Imports: []Import{
				{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
			Location: common.AddressLocation{
				Address: common.MustBytesToAddress([]byte{0x1}),
			},
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("import"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.Line{},
							prettier.Text("foo"),
							prettier.Line{},
							prettier.Text("from "),
						},
					},
				},
				prettier.Text("0x1"),
			},
			decl.Doc(),
		)
	})

	t.Run("two identifiers", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Imports: []Import{
				{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Location: common.IdentifierLocation("test"),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("import"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.Line{},
							prettier.Text("foo"),
							prettier.Concat{
								prettier.Text(","),
								prettier.Line{},
							},
							prettier.Text("bar"),
							prettier.Line{},
							prettier.Text("from "),
						},
					},
				},
				prettier.Text("test"),
			},
			decl.Doc(),
		)
	})

	t.Run("two imports, one with alias", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Imports: []Import{
				{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "bar",
					},
					Alias: Identifier{
						Identifier: "baz",
					},
				},
			},
			Location: common.IdentifierLocation("test"),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("import"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.Line{},
							prettier.Text("foo"),
							prettier.Concat{
								prettier.Text(","),
								prettier.Line{},
							},
							prettier.Group{
								Doc: prettier.Indent{
									Doc: prettier.Concat{
										prettier.Text("bar"),
										prettier.Line{},
										prettier.Text("as"),
										prettier.Space,
										prettier.Text("baz"),
									},
								},
							},
							prettier.Line{},
							prettier.Text("from "),
						},
					},
				},
				prettier.Text("test"),
			},
			decl.Doc(),
		)
	})

}

func TestImportDeclaration_String(t *testing.T) {

	t.Parallel()

	t.Run("no imports", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Location: common.StringLocation("test"),
		}

		require.Equal(
			t,
			`import "test"`,
			decl.String(),
		)
	})

	t.Run("one import", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Imports: []Import{
				{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
			Location: common.AddressLocation{
				Address: common.MustBytesToAddress([]byte{0x1}),
			},
		}

		require.Equal(
			t,
			`import foo from 0x1`,
			decl.String(),
		)
	})

	t.Run("two imports", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Imports: []Import{
				{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Location: common.IdentifierLocation("test"),
		}

		require.Equal(
			t,
			`import foo, bar from test`,
			decl.String(),
		)
	})

	t.Run("two imports, one with alias", func(t *testing.T) {

		t.Parallel()

		decl := &ImportDeclaration{
			Imports: []Import{
				{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "bar",
					},
					Alias: Identifier{
						Identifier: "baz",
					},
				},
			},
			Location: common.IdentifierLocation("test"),
		}

		require.Equal(
			t,
			`import foo, bar as baz from test`,
			decl.String(),
		)
	})
}
