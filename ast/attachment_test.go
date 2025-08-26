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
)

func TestAttachmentDeclaration_MarshallJSON(t *testing.T) {

	t.Parallel()

	decl := &AttachmentDeclaration{
		Access: AccessAll,
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
            "Access": "AccessAll",
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

	t.Run("with elements", func(t *testing.T) {

		t.Parallel()

		decl := &AttachmentDeclaration{
			Access: AccessAll,
			Identifier: Identifier{
				Identifier: "Foo",
			},
			BaseType: &NominalType{
				Identifier: Identifier{
					Identifier: "Bar",
				},
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "Baz",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{}),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("access(all)"),
				prettier.HardLine{},
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
	})

	t.Run("without elements", func(t *testing.T) {

		t.Parallel()

		decl := &AttachmentDeclaration{}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text(""),
				prettier.HardLine{},
				prettier.Text("attachment"),
				prettier.Text(" "),
				prettier.Text(""),
				prettier.Text(" "),
				prettier.Text("for"),
				prettier.Text(" "),
				prettier.Text(""),
				prettier.Text(" "),
				prettier.Text("{}"),
			},
			decl.Doc(),
		)
	})
}

func TestAttachmentDeclaration_String(t *testing.T) {

	t.Parallel()

	t.Run("with elements", func(t *testing.T) {

		t.Parallel()

		decl := &AttachmentDeclaration{
			Access: AccessAll,
			Identifier: Identifier{
				Identifier: "Foo",
			},
			BaseType: &NominalType{
				Identifier: Identifier{
					Identifier: "Bar",
				},
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "Baz",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{}),
		}

		require.Equal(t,
			`access(all)
attachment Foo for Bar: Baz {}`,
			decl.String(),
		)
	})

	t.Run("without elements", func(t *testing.T) {

		t.Parallel()

		decl := &AttachmentDeclaration{}

		require.Equal(
			t,
			`
attachment  for  {}`,
			decl.String(),
		)
	})
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
		Attachment: NewInvocationExpression(
			nil,
			NewIdentifierExpression(
				nil,
				NewIdentifier(
					nil,
					"bar",
					Position{Offset: 1, Line: 2, Column: 3},
				),
			),
			[]*TypeAnnotation{},
			Arguments{},
			Position{Offset: 1, Line: 2, Column: 3},
			Position{Offset: 1, Line: 2, Column: 3},
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
            "Attachment": {
                "Type": "InvocationExpression",
                "InvokedExpression": {
                    "Type": "IdentifierExpression",
                    "Identifier": { 
                        "Identifier": "bar",
                        "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                        "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                    },
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                },
                "Arguments":[],
                "TypeArguments":[],
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "ArgumentsStartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
            }
        }
        `,
		string(actual),
	)
}

func TestAttachExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		decl := &AttachExpression{
			Base: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			Attachment: &InvocationExpression{
				InvokedExpression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("attach"),
				prettier.Text(" "),
				prettier.Concat{
					prettier.Text("bar"),
					prettier.Text("()"),
				},
				prettier.Text(" "),
				prettier.Text("to"),
				prettier.Text(" "),
				prettier.Text("foo"),
			},
			decl.Doc(),
		)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		decl := &AttachExpression{}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("attach"),
				prettier.Text(" "),
				prettier.Text(""),
				prettier.Text(" "),
				prettier.Text("to"),
				prettier.Text(" "),
				prettier.Text(""),
			},
			decl.Doc(),
		)
	})
}

func TestAttachExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		decl := &AttachExpression{
			Base: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			Attachment: &InvocationExpression{
				InvokedExpression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
		}

		require.Equal(t,
			"attach bar() to foo",
			decl.String(),
		)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		decl := &AttachExpression{}

		require.Equal(t,
			"attach  to ",
			decl.String(),
		)
	})
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

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		decl := &RemoveStatement{
			Attachment: &NominalType{
				Identifier: Identifier{
					Identifier: "E",
				},
			},
			Value: NewIdentifierExpression(
				nil,
				Identifier{
					Identifier: "baz",
				},
			),
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
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		decl := &RemoveStatement{
			Attachment: nil,
			Value:      nil,
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("remove"),
				prettier.Text(" "),
				prettier.Text(""),
				prettier.Text(" "),
				prettier.Text("from"),
				prettier.Text(" "),
				prettier.Text(""),
			},
			decl.Doc(),
		)
	})
}

func TestRemoveStatement_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		t.Parallel()

		decl := &RemoveStatement{
			Attachment: &NominalType{
				Identifier: Identifier{
					Identifier: "E",
				},
			},
			Value: NewIdentifierExpression(
				nil,
				Identifier{
					Identifier: "baz",
				},
			),
		}

		require.Equal(t,
			"remove E from baz",
			decl.String(),
		)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		decl := &RemoveStatement{
			Attachment: nil,
			Value:      nil,
		}

		require.Equal(
			t,
			"remove  from ",
			decl.String(),
		)
	})
}
