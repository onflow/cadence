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

func TestExtendDeclaration_MarshallJSON(t *testing.T) {

	t.Parallel()

	decl := &ExtensionDeclaration{
		Access: AccessPublic,
		Identifier: NewIdentifier(
			nil,
			"Foo",
			Position{Offset: 1, Line: 2, Column: 3},
		),
		BaseType: NewIdentifier(
			nil,
			"Bar",
			Position{Offset: 1, Line: 2, Column: 3},
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
		`
        {
            "Type": "ExtensionDeclaration",
			"Access": "AccessPublic",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6},
			"Identifier": {
                "Identifier": "Foo",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
			"BaseType": {
                "Identifier": "Bar",
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

func TestExtensionDeclaration_Doc(t *testing.T) {

	t.Parallel()

	decl := &ExtensionDeclaration{
		Access: AccessPublic,
		Identifier: NewIdentifier(
			nil,
			"Foo",
			Position{Offset: 1, Line: 2, Column: 3},
		),
		BaseType: NewIdentifier(
			nil,
			"Bar",
			Position{Offset: 1, Line: 2, Column: 3},
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
			prettier.Text("extension"),
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

	require.Equal(t, "pub extension Foo for Bar: Baz {}", decl.String())
}

func TestExtendExpressionMarshallJSON(t *testing.T) {

	t.Parallel()

	decl := &ExtendExpression{
		Base: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"foo",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		Extensions: []Expression{
			NewIdentifierExpression(
				nil,
				NewIdentifier(
					nil,
					"bar",
					Position{Offset: 1, Line: 2, Column: 3},
				),
			),
			NewIdentifierExpression(
				nil,
				NewIdentifier(
					nil,
					"baz",
					Position{Offset: 1, Line: 2, Column: 3},
				),
			),
		},
		StartPos: Position{Offset: 1, Line: 2, Column: 3},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ExtendExpression",
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
			"Extensions": [
                {
                    "Type": "IdentifierExpression",
					"Identifier": { 
						"Identifier": "bar",
						"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
						"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
					},
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                },
				{
                    "Type": "IdentifierExpression",
					"Identifier": { 
						"Identifier": "baz",
						"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
						"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
					},
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                }
            ]
        }
        `,
		string(actual),
	)
}

func TestExtendExpression_Doc(t *testing.T) {

	t.Parallel()

	decl := &ExtendExpression{
		Base: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"foo",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		Extensions: []Expression{
			NewIdentifierExpression(
				nil,
				NewIdentifier(
					nil,
					"bar",
					Position{Offset: 1, Line: 2, Column: 3},
				),
			),
			NewIdentifierExpression(
				nil,
				NewIdentifier(
					nil,
					"baz",
					Position{Offset: 1, Line: 2, Column: 3},
				),
			),
		},
		StartPos: Position{Offset: 1, Line: 2, Column: 3},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("extend"),
			prettier.Text(" "),
			prettier.Text("foo"),
			prettier.Text(" "),
			prettier.Text("with"),
			prettier.Text(" "),
			prettier.Text("bar"),
			prettier.Text(" "),
			prettier.Text("and"),
			prettier.Text(" "),
			prettier.Text("baz"),
		},
		decl.Doc(),
	)

	require.Equal(t, "extend foo with bar and baz", decl.String())
}

func TestRemoveStatement_MarshallJSON(t *testing.T) {

	t.Parallel()

	decl := &RemoveStatement{
		ValueTarget: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"foo",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		ExtensionTarget: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"bar",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		Transfer: NewTransfer(
			nil,
			TransferOperation(TransferOperationCopy),
			Position{Offset: 1, Line: 2, Column: 3},
		),
		Extension: NewIdentifier(
			nil,
			"E",
			Position{Offset: 1, Line: 2, Column: 3},
		),
		Value: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"baz",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		IsDeclaration: true,
		IsConstant:    false,
		StartPos:      Position{Offset: 1, Line: 2, Column: 3},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "RemoveStatement",
			"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
			"EndPos": {"Offset": 3, "Line": 2, "Column": 5},
			"ValueTarget":  {
				"Type": "IdentifierExpression",
				"Identifier": { 
					"Identifier": "foo",
					"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
					"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
				},
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
			},
			"ExtensionTarget":  {
				"Type": "IdentifierExpression",
				"Identifier": { 
					"Identifier": "bar",
					"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
					"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
				},
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
			},
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
			"Extension":  {
				"Identifier": "E",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 1, "Line": 2, "Column": 3}
			},
			"Transfer": {
                "Type": "Transfer",
                "Operation": "TransferOperationCopy",
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
            },
			"IsDeclaration": true,
			"IsConstant": false
        }
        `,
		string(actual),
	)
}

func TestRemoveStatement_Doc(t *testing.T) {

	t.Parallel()

	decl := &RemoveStatement{
		ValueTarget: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"foo",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		ExtensionTarget: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"bar",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		Transfer: NewTransfer(
			nil,
			TransferOperation(TransferOperationCopy),
			Position{Offset: 1, Line: 2, Column: 3},
		),
		Extension: NewIdentifier(
			nil,
			"E",
			Position{Offset: 1, Line: 2, Column: 3},
		),
		Value: NewIdentifierExpression(
			nil,
			NewIdentifier(
				nil,
				"baz",
				Position{Offset: 1, Line: 2, Column: 3},
			),
		),
		IsDeclaration: true,
		IsConstant:    false,
		StartPos:      Position{Offset: 1, Line: 2, Column: 3},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("var"),
			prettier.Text(" "),
			prettier.Text("foo"),
			prettier.Text(","),
			prettier.Text(" "),
			prettier.Text("bar"),
			prettier.Text(" "),
			prettier.Text("="),
			prettier.Text(" "),
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

	require.Equal(t, "var foo, bar = remove E from baz", decl.String())
}
