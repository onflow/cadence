/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

func TestExpressionStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &ExpressionStatement{
		Expression: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ExpressionStatement",
            "Expression": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestExpressionStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &ExpressionStatement{
		Expression: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
	}

	assert.Equal(t,
		prettier.Text("false"),
		stmt.Doc(),
	)
}

func TestReturnStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &ReturnStatement{
		Expression: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ReturnStatement",
            "Expression": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}
func TestReturnStatement_Doc(t *testing.T) {

	t.Parallel()

	t.Run("value", func(t *testing.T) {

		t.Parallel()

		stmt := &ReturnStatement{
			Expression: &BoolExpression{
				Value: false,
			},
		}

		require.Equal(t,
			prettier.Concat{
				prettier.Text("return "),
				prettier.Text("false"),
			},
			stmt.Doc(),
		)
	})

	t.Run("no value", func(t *testing.T) {

		t.Parallel()

		stmt := &ReturnStatement{}

		require.Equal(t,
			prettier.Text("return"),
			stmt.Doc(),
		)
	})
}

func TestBreakStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &BreakStatement{
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "BreakStatement",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestBreakStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &BreakStatement{
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	assert.Equal(t,
		prettier.Text("break"),
		stmt.Doc(),
	)
}

func TestContinueStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &ContinueStatement{
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ContinueStatement",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestContinueStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &ContinueStatement{
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	assert.Equal(t,
		prettier.Text("continue"),
		stmt.Doc(),
	)
}

func TestIfStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &IfStatement{
		Test: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Then: &Block{
			Statements: []Statement{},
			Range: Range{
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
				EndPos:   Position{Offset: 10, Line: 11, Column: 12},
			},
		},
		Else: &Block{
			Statements: []Statement{},
			Range: Range{
				StartPos: Position{Offset: 13, Line: 14, Column: 15},
				EndPos:   Position{Offset: 16, Line: 17, Column: 18},
			},
		},
		StartPos: Position{Offset: 19, Line: 20, Column: 21},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "IfStatement",
            "Test": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "Then": {
                "Type": "Block",
                "Statements": [],
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
            },
            "Else": {
                "Type": "Block",
                "Statements": [],
                "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
                "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
            },
            "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
            "EndPos":   {"Offset": 16, "Line": 17, "Column": 18}
        }
        `,
		string(actual),
	)
}

func TestIfStatement_Doc(t *testing.T) {

	t.Parallel()

	t.Run("empty if-else", func(t *testing.T) {

		t.Parallel()

		stmt := &IfStatement{
			Test: &BoolExpression{
				Value: false,
				Range: Range{
					StartPos: Position{Offset: 1, Line: 2, Column: 3},
					EndPos:   Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			Then: &Block{
				Statements: []Statement{},
				Range: Range{
					StartPos: Position{Offset: 7, Line: 8, Column: 9},
					EndPos:   Position{Offset: 10, Line: 11, Column: 12},
				},
			},
			Else: &Block{
				Statements: []Statement{},
				Range: Range{
					StartPos: Position{Offset: 13, Line: 14, Column: 15},
					EndPos:   Position{Offset: 16, Line: 17, Column: 18},
				},
			},
			StartPos: Position{Offset: 19, Line: 20, Column: 21},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("if "),
					prettier.Text("false"),
					prettier.Text(" "),
					prettier.Text("{}"),
					prettier.Text(" else "),
					prettier.Group{
						Doc: prettier.Text("{}"),
					},
				},
			},
			stmt.Doc(),
		)
	})

	t.Run("if-else if", func(t *testing.T) {

		t.Parallel()

		stmt := &IfStatement{
			Test: &BoolExpression{
				Value: false,
				Range: Range{
					StartPos: Position{Offset: 1, Line: 2, Column: 3},
					EndPos:   Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			Then: &Block{
				Statements: []Statement{},
				Range: Range{
					StartPos: Position{Offset: 7, Line: 8, Column: 9},
					EndPos:   Position{Offset: 10, Line: 11, Column: 12},
				},
			},
			Else: &Block{
				Statements: []Statement{
					&IfStatement{
						Test: &BoolExpression{
							Value: true,
							Range: Range{
								StartPos: Position{Offset: 22, Line: 23, Column: 24},
								EndPos:   Position{Offset: 25, Line: 26, Column: 27},
							},
						},
						Then: &Block{
							Statements: []Statement{},
							Range: Range{
								StartPos: Position{Offset: 28, Line: 29, Column: 30},
								EndPos:   Position{Offset: 31, Line: 32, Column: 33},
							},
						},
					},
				},
				Range: Range{
					StartPos: Position{Offset: 13, Line: 14, Column: 15},
					EndPos:   Position{Offset: 16, Line: 17, Column: 18},
				},
			},
			StartPos: Position{Offset: 19, Line: 20, Column: 21},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("if "),
					prettier.Text("false"),
					prettier.Text(" "),
					prettier.Text("{}"),
					prettier.Text(" else "),
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("if "),
								prettier.Text("true"),
								prettier.Text(" "),
								prettier.Text("{}"),
							},
						},
					},
				},
			},
			stmt.Doc(),
		)
	})
}

func TestWhileStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &WhileStatement{
		Test: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Block: &Block{
			Statements: []Statement{},
			Range: Range{
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
				EndPos:   Position{Offset: 10, Line: 11, Column: 12},
			},
		},
		StartPos: Position{Offset: 13, Line: 14, Column: 15},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "WhileStatement",
            "Test": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "Block": {
                "Type": "Block",
                "Statements": [],
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
            },
            "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
            "EndPos":   {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestForStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &ForStatement{
		Identifier: Identifier{
			Identifier: "foobar",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		Value: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 4, Line: 5, Column: 6},
				EndPos:   Position{Offset: 7, Line: 8, Column: 9},
			},
		},
		Block: &Block{
			Statements: []Statement{},
			Range: Range{
				StartPos: Position{Offset: 10, Line: 11, Column: 12},
				EndPos:   Position{Offset: 13, Line: 14, Column: 15},
			},
		},
		StartPos: Position{Offset: 16, Line: 17, Column: 18},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ForStatement",
            "Identifier": {
                "Identifier": "foobar",
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
			"Index": null,
            "Value": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                "EndPos": {"Offset": 7, "Line": 8, "Column": 9}
            },
            "Block": {
                "Type": "Block",
                "Statements": [],
                "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
            },
            "StartPos": {"Offset": 16, "Line": 17, "Column": 18},
            "EndPos":  {"Offset": 13, "Line": 14, "Column": 15}
        }
        `,
		string(actual),
	)
}

func TestAssignmentStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &AssignmentStatement{
		Target: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Transfer: &Transfer{
			Operation: TransferOperationCopy,
			Pos:       Position{Offset: 4, Line: 5, Column: 6},
		},
		Value: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
				EndPos:   Position{Offset: 10, Line: 11, Column: 12},
			},
		},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "AssignmentStatement",
            "Target": {
                "Type": "IdentifierExpression",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "Transfer": {
                "Type": "Transfer",
                "Operation": "TransferOperationCopy",
                "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "Value": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos":  {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestSwapStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &SwapStatement{
		Left: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Right: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 4, Line: 5, Column: 6},
				EndPos:   Position{Offset: 7, Line: 8, Column: 9},
			},
		},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "SwapStatement",
            "Left": {
                "Type": "IdentifierExpression",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "Right": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                "EndPos": {"Offset": 7, "Line": 8, "Column": 9}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos":   {"Offset": 7, "Line": 8, "Column": 9}
        }
        `,
		string(actual),
	)
}

func TestEmitStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &EmitStatement{
		InvocationExpression: &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foobar",
					Pos:        Position{Offset: 1, Line: 2, Column: 3},
				},
			},
			TypeArguments: []*TypeAnnotation{
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AB",
							Pos:        Position{Offset: 4, Line: 5, Column: 6},
						},
					},
					StartPos: Position{Offset: 7, Line: 8, Column: 9},
				},
			},
			Arguments: []*Argument{
				{
					Label:         "ok",
					LabelStartPos: &Position{Offset: 10, Line: 11, Column: 12},
					LabelEndPos:   &Position{Offset: 13, Line: 14, Column: 15},
					Expression: &BoolExpression{
						Value: false,
						Range: Range{
							StartPos: Position{Offset: 16, Line: 17, Column: 18},
							EndPos:   Position{Offset: 19, Line: 20, Column: 21},
						},
					},
					TrailingSeparatorPos: Position{Offset: 28, Line: 29, Column: 30},
				},
			},
			ArgumentsStartPos: Position{Offset: 31, Line: 32, Column: 33},
			EndPos:            Position{Offset: 22, Line: 23, Column: 24},
		},
		StartPos: Position{Offset: 25, Line: 26, Column: 27},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "EmitStatement",
            "InvocationExpression": {
                "Type": "InvocationExpression",
                "InvokedExpression": {
                   "Type": "IdentifierExpression",
                   "Identifier": {
                       "Identifier": "foobar",
                       "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                       "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                   },
                   "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                   "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "TypeArguments": [
                    {
                       "IsResource": true,
                       "AnnotatedType": {
                           "Type": "NominalType",
                           "Identifier": {
                               "Identifier": "AB",
                               "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                               "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                           },
                           "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                           "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                       },
                       "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                       "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                    }
                ],
                "Arguments": [
                    {
                        "Label": "ok",
                        "LabelStartPos": {"Offset": 10, "Line": 11, "Column": 12},
                        "LabelEndPos": {"Offset": 13, "Line": 14, "Column": 15},
                        "Expression": {
                            "Type": "BoolExpression",
                            "Value": false,
                            "StartPos": {"Offset": 16, "Line": 17, "Column": 18},
                            "EndPos": {"Offset": 19, "Line": 20, "Column": 21}
                        },
                        "TrailingSeparatorPos": {"Offset": 28, "Line": 29, "Column": 30},
                        "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                        "EndPos": {"Offset": 19, "Line": 20, "Column": 21}
                    }
                ],
                "ArgumentsStartPos": {"Offset": 31, "Line": 32, "Column": 33},
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 22, "Line": 23, "Column": 24}
            },
            "StartPos": {"Offset": 25, "Line": 26, "Column": 27},
            "EndPos": {"Offset": 22, "Line": 23, "Column": 24}
        }
        `,
		string(actual),
	)
}
