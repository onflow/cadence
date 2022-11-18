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

package ast

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"
)

func TestAttachmentDeclaration_MarshallJSON(t *testing.T) {

	t.Parallel()

	decl := &AttachmentDeclaration{
		Access: AccessPublic,
		Identifier: NewIdentifier(
			nil,
			"Foo",
			Position{Offset: 1, Line: 2, Column: 3},
		),
		BaseType: NewNominalType(
			nil,
			NewIdentifier(
				nil,
				"Bar",
				Position{Offset: 1, Line: 2, Column: 3},
			),
			[]Identifier{},
		),
		Conformances: []*NominalType{
			{
				Identifier: NewIdentifier(
					nil,
					"Baz",
					Position{Offset: 1, Line: 2, Column: 3},
				),
			},
		},
		Members:   NewMembers(nil, []Declaration{}),
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "AttachmentDeclaration",
            "Access": "AccessPublic",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6},
            "Identifier": {
                "Identifier": "Foo",
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
            "BaseType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "Bar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
            "DocString": "test",
            "Conformances": [
                {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "Baz",
                        "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                        "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                    },
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                }
            ], 
            "Members": {
                "Declarations": []
            }
        }
        `,
		string(actual),
	)
}

func TestAttachmentDeclaration_Doc(t *testing.T) {

	t.Parallel()

	decl := &AttachmentDeclaration{
		Access: AccessPublic,
		Identifier: NewIdentifier(
			nil,
			"Foo",
			Position{Offset: 1, Line: 2, Column: 3},
		),
		BaseType: NewNominalType(
			nil,
			NewIdentifier(
				nil,
				"Bar",
				Position{Offset: 1, Line: 2, Column: 3},
			),
			[]Identifier{},
		),
		Conformances: []*NominalType{
			{
				Identifier: NewIdentifier(
					nil,
					"Baz",
					Position{Offset: 1, Line: 2, Column: 3},
				),
			},
		},
		Members:   NewMembers(nil, []Declaration{}),
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("pub"),
			prettier.Text(" "),
			prettier.Text("attachment"),
			prettier.Text(" "),
			prettier.Text("Foo"),
			prettier.Text(" "),
			prettier.Text("for"),
			prettier.Text(" "),
			prettier.Text("Bar"),
			prettier.Text(":"),
			prettier.Group{
				Doc: prettier.Indent{
					Doc: prettier.Concat{
						prettier.Line{},
						prettier.Text("Baz"),
						prettier.Dedent{
							Doc: prettier.Concat{
								prettier.Line{},
								prettier.Text("{}"),
							},
						},
					},
				},
			},
		},
		decl.Doc(),
	)

	require.Equal(t, "pub attachment Foo for Bar: Baz {}", decl.String())
}

func TestAttachExpressionMarshallJSON(t *testing.T) {

	t.Parallel()

	decl := &AttachExpression{
		Base: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"foo",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		Attachment: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"bar",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		StartPos: Position{Offset: 1, Line: 2, Column: 3},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "AttachExpression",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 3, "Line": 2, "Column": 5},
            "Base":  {
                "Type": "IdentifierExpression",
                "Identifier": { 
                    "Identifier": "foo",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
            "Attachment":  {
                "Type": "IdentifierExpression",
                "Identifier": { 
                    "Identifier": "bar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            }
        }
        `,
		string(actual),
	)
}

func TestAttachExpression_Doc(t *testing.T) {

	t.Parallel()

	decl := &AttachExpression{
		Base: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"foo",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		Attachment: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"bar",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		StartPos: Position{Offset: 1, Line: 2, Column: 3},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("attach"),
			prettier.Text(" "),
			prettier.Text("bar"),
			prettier.Text(" "),
			prettier.Text("to"),
			prettier.Text(" "),
			prettier.Text("foo"),
		},
		decl.Doc(),
	)

	require.Equal(t, "attach bar to foo", decl.String())
}

func TestRemoveStatement_MarshallJSON(t *testing.T) {

	t.Parallel()

	decl := &RemoveStatement{
		Attachment: NewNominalType(
			nil,
			NewIdentifier(
				nil,
				"E",
				Position{Offset: 1, Line: 2, Column: 3},
			),
			[]Identifier{},
		),
		Value: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"baz",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		StartPos: Position{Offset: 1, Line: 2, Column: 3},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "RemoveStatement",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 3, "Line": 2, "Column": 5},
            "Value":  {
                "Type": "IdentifierExpression",
                "Identifier": { 
                    "Identifier": "baz",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
            "Attachment": {
                "Type": "NominalType",
                "Identifier": { 
                    "Identifier": "E",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
            }
        }
        `,
		string(actual),
	)
}

func TestRemoveStatement_Doc(t *testing.T) {

	t.Parallel()

	decl := &RemoveStatement{
		Attachment: NewNominalType(
			nil,
			NewIdentifier(
				nil,
				"E",
				Position{Offset: 1, Line: 2, Column: 3},
			),
			[]Identifier{},
		),
		Value: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"baz",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		StartPos: Position{Offset: 1, Line: 2, Column: 3},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("remove"),
			prettier.Text(" "),
			prettier.Text("E"),
			prettier.Text(" "),
			prettier.Text("from"),
			prettier.Text(" "),
			prettier.Text("baz"),
		},
		decl.Doc(),
	)

	require.Equal(t, "remove E from baz", decl.String())
}
