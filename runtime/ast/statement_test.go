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
		// language=json
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
		},
	}

	assert.Equal(t,
		prettier.Text("false"),
		stmt.Doc(),
	)
}

func TestExpressionStatement_String(t *testing.T) {

	t.Parallel()

	stmt := &ExpressionStatement{
		Expression: &BoolExpression{
			Value: false,
		},
	}

	assert.Equal(t,
		"false",
		stmt.String(),
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
		// language=json
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

func TestReturnStatement_String(t *testing.T) {

	t.Parallel()

	t.Run("value", func(t *testing.T) {

		t.Parallel()

		stmt := &ReturnStatement{
			Expression: &BoolExpression{
				Value: false,
			},
		}

		require.Equal(t,
			"return false",
			stmt.String(),
		)
	})

	t.Run("no value", func(t *testing.T) {

		t.Parallel()

		stmt := &ReturnStatement{}

		require.Equal(t,
			"return",
			stmt.String(),
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
		// language=json
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

	assert.Equal(t,
		prettier.Text("break"),
		(&BreakStatement{}).Doc(),
	)
}

func TestBreakStatement_String(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"break",
		(&BreakStatement{}).String(),
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
		// language=json
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

	assert.Equal(t,
		prettier.Text("continue"),
		(&ContinueStatement{}).Doc(),
	)
}

func TestContinueStatement_String(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"continue",
		(&ContinueStatement{}).String(),
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
		// language=json
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
			},
			Then: &Block{
				Statements: []Statement{},
			},
			Else: &Block{
				Statements: []Statement{},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("if "),
					prettier.Text("false"),
					prettier.Text(" "),
					prettier.Text("{}"),
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
			},
			Then: &Block{
				Statements: []Statement{},
			},
			Else: &Block{
				Statements: []Statement{
					&IfStatement{
						Test: &BoolExpression{
							Value: true,
						},
						Then: &Block{
							Statements: []Statement{},
						},
					},
				},
			},
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

func TestIfStatement_String(t *testing.T) {

	t.Parallel()

	t.Run("empty if-else", func(t *testing.T) {

		t.Parallel()

		stmt := &IfStatement{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &Block{
				Statements: []Statement{},
			},
			Else: &Block{
				Statements: []Statement{},
			},
		}

		assert.Equal(t,
			"if false {}",
			stmt.String(),
		)
	})

	t.Run("if-else if", func(t *testing.T) {

		t.Parallel()

		stmt := &IfStatement{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &Block{
				Statements: []Statement{},
			},
			Else: &Block{
				Statements: []Statement{
					&IfStatement{
						Test: &BoolExpression{
							Value: true,
						},
						Then: &Block{
							Statements: []Statement{},
						},
					},
				},
			},
		}

		assert.Equal(t,
			"if false {} else if true {}",
			stmt.String(),
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
		// language=json
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

func TestWhileStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &WhileStatement{
		Test: &BoolExpression{
			Value: false,
		},
		Block: &Block{
			Statements: []Statement{},
		},
	}

	assert.Equal(t,
		prettier.Group{
			Doc: prettier.Concat{
				prettier.Text("while "),
				prettier.Text("false"),
				prettier.Text(" "),
				prettier.Text("{}"),
			},
		},
		stmt.Doc(),
	)
}

func TestWhileStatement_String(t *testing.T) {

	t.Parallel()

	stmt := &WhileStatement{
		Test: &BoolExpression{
			Value: false,
		},
		Block: &Block{
			Statements: []Statement{},
		},
	}

	assert.Equal(t,
		"while false {}",
		stmt.String(),
	)
}

func TestForStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	t.Run("without index", func(t *testing.T) {

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
			// language=json
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
	})

	t.Run("with index", func(t *testing.T) {

		t.Parallel()

		stmt := &ForStatement{
			Index: &Identifier{
				Identifier: "i",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 4, Line: 5, Column: 6},
			},
			Value: &BoolExpression{
				Value: false,
				Range: Range{
					StartPos: Position{Offset: 7, Line: 8, Column: 9},
					EndPos:   Position{Offset: 10, Line: 11, Column: 12},
				},
			},
			Block: &Block{
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
			// language=json
			`
            {
                "Type": "ForStatement",
                "Index": {
                    "Identifier": "i",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 1, "Line": 2, "Column": 3}
                },
		    	"Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 9, "Line": 5, "Column": 11}
                },
                "Value": {
                    "Type": "BoolExpression",
                    "Value": false,
                    "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                    "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
                },
                "Block": {
                    "Type": "Block",
                    "Statements": [],
                    "StartPos":{"Offset": 13, "Line": 14, "Column": 15},
                    "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
                },
                "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
                "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
            }
            `,
			string(actual),
		)
	})

}

func TestForStatement_Doc(t *testing.T) {

	t.Parallel()

	t.Run("without index", func(t *testing.T) {

		t.Parallel()

		stmt := &ForStatement{
			Identifier: Identifier{
				Identifier: "foobar",
			},
			Value: &BoolExpression{
				Value: false,
			},
			Block: &Block{
				Statements: []Statement{},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("for "),
					prettier.Text("foobar"),
					prettier.Text(" in "),
					prettier.Text("false"),
					prettier.Text(" "),
					prettier.Text("{}"),
				},
			},
			stmt.Doc(),
		)
	})

	t.Run("with index", func(t *testing.T) {

		t.Parallel()

		stmt := &ForStatement{
			Index: &Identifier{
				Identifier: "i",
			},
			Identifier: Identifier{
				Identifier: "foobar",
			},
			Value: &BoolExpression{
				Value: false,
			},
			Block: &Block{
				Statements: []Statement{},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("for "),
					prettier.Text("i"),
					prettier.Text(", "),
					prettier.Text("foobar"),
					prettier.Text(" in "),
					prettier.Text("false"),
					prettier.Text(" "),
					prettier.Text("{}"),
				},
			},
			stmt.Doc(),
		)
	})
}

func TestForStatement_String(t *testing.T) {

	t.Parallel()

	t.Run("without index", func(t *testing.T) {

		t.Parallel()

		stmt := &ForStatement{
			Identifier: Identifier{
				Identifier: "foobar",
			},
			Value: &BoolExpression{
				Value: false,
			},
			Block: &Block{
				Statements: []Statement{},
			},
		}

		assert.Equal(t,
			"for foobar in false {}",
			stmt.String(),
		)
	})

	t.Run("with index", func(t *testing.T) {

		t.Parallel()

		stmt := &ForStatement{
			Index: &Identifier{
				Identifier: "i",
			},
			Identifier: Identifier{
				Identifier: "foobar",
			},
			Value: &BoolExpression{
				Value: false,
			},
			Block: &Block{
				Statements: []Statement{},
			},
		}

		assert.Equal(t,
			"for i, foobar in false {}",
			stmt.String(),
		)
	})
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
		// language=json
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

func TestAssignmentStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &AssignmentStatement{
		Target: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
			},
		},
		Transfer: &Transfer{
			Operation: TransferOperationCopy,
		},
		Value: &BoolExpression{
			Value: false,
		},
	}

	require.Equal(t,
		prettier.Group{
			Doc: prettier.Concat{
				prettier.Text("foobar"),
				prettier.Text(" "),
				prettier.Text("="),
				prettier.Text(" "),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Text("false"),
					},
				},
			},
		},
		stmt.Doc(),
	)
}

func TestAssignmentStatement_String(t *testing.T) {

	t.Parallel()

	stmt := &AssignmentStatement{
		Target: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
			},
		},
		Transfer: &Transfer{
			Operation: TransferOperationCopy,
		},
		Value: &BoolExpression{
			Value: false,
		},
	}

	require.Equal(t,
		"foobar = false",
		stmt.String(),
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
		// language=json
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

func TestSwapStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &SwapStatement{
		Left: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
			},
		},
		Right: &BoolExpression{
			Value: false,
		},
	}

	assert.Equal(t,
		prettier.Group{
			Doc: prettier.Concat{
				prettier.Text("foobar"),
				swapStatementSpaceSymbolSpaceDoc,
				prettier.Text("false"),
			},
		},
		stmt.Doc(),
	)
}

func TestSwapStatement_String(t *testing.T) {

	t.Parallel()

	stmt := &SwapStatement{
		Left: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
			},
		},
		Right: &BoolExpression{
			Value: false,
		},
	}

	assert.Equal(t,
		"foobar <-> false",
		stmt.String(),
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
		// language=json
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

func TestEmitStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &EmitStatement{
		InvocationExpression: &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foobar",
				},
			},
		},
	}

	require.Equal(t,
		prettier.Concat{
			prettier.Text("emit "),
			prettier.Concat{
				prettier.Text("foobar"),
				prettier.Text("()"),
			},
		},
		stmt.Doc(),
	)
}

func TestEmitStatement_String(t *testing.T) {

	t.Parallel()

	stmt := &EmitStatement{
		InvocationExpression: &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foobar",
				},
			},
		},
	}

	require.Equal(t,
		"emit foobar()",
		stmt.String(),
	)
}

func TestSwitchStatement_MarshalJSON(t *testing.T) {

	t.Parallel()

	stmt := &SwitchStatement{
		Expression: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foo",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Cases: []*SwitchCase{
			{
				Expression: &BoolExpression{
					Value: false,
					Range: Range{
						StartPos: Position{Offset: 4, Line: 5, Column: 6},
						EndPos:   Position{Offset: 7, Line: 8, Column: 9},
					},
				},
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "bar",
								Pos:        Position{Offset: 10, Line: 11, Column: 12},
							},
						},
					},
				},
				Range: Range{
					StartPos: Position{Offset: 13, Line: 14, Column: 15},
					EndPos:   Position{Offset: 16, Line: 17, Column: 18},
				},
			},
			{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "baz",
								Pos:        Position{Offset: 19, Line: 20, Column: 21},
							},
						},
					},
				},
				Range: Range{
					StartPos: Position{Offset: 22, Line: 23, Column: 24},
					EndPos:   Position{Offset: 25, Line: 26, Column: 27},
				},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 28, Line: 29, Column: 30},
			EndPos:   Position{Offset: 31, Line: 32, Column: 33},
		},
	}

	actual, err := json.Marshal(stmt)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "SwitchStatement",
            "Expression": {
                "Type": "IdentifierExpression",
                "Identifier": {
                    "Identifier": "foo",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
            "Cases": [
                {
                    "Type": "SwitchCase",
                    "Expression": {
                        "Type": "BoolExpression",
                        "Value": false,
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 7, "Line": 8, "Column": 9}
                    },
                    "Statements": [
                        {
                            "Type": "ExpressionStatement",
                            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                            "EndPos": {"Offset": 12, "Line": 11, "Column": 14},
                            "Expression": {
                                "Type": "IdentifierExpression",
                                "Identifier": {
                                    "Identifier": "bar",
                                    "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                                    "EndPos": {"Offset": 12, "Line": 11, "Column": 14}
                                },
                                "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                                "EndPos": {"Offset": 12, "Line": 11, "Column": 14}
                            }
                        }
                    ],
                    "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
                    "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
                },
                {
                    "Type": "SwitchCase",
                    "Expression": null,
                    "Statements": [
                        {
                            "Type": "ExpressionStatement",
                            "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
                            "EndPos": {"Offset": 21, "Line": 20, "Column": 23},
                            "Expression": {
                                "Type": "IdentifierExpression",
                                "Identifier": {
                                    "Identifier": "baz",
                                    "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
                                    "EndPos": {"Offset": 21, "Line": 20, "Column": 23}
                                },
                                "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
                                "EndPos": {"Offset": 21, "Line": 20, "Column": 23}
                            }
                        }
                    ],
                    "StartPos": {"Offset": 22, "Line": 23, "Column": 24},
                    "EndPos": {"Offset": 25, "Line": 26, "Column": 27}
                }
            ],
            "StartPos": {"Offset": 28, "Line": 29, "Column": 30},
            "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
        }
        `,
		string(actual),
	)
}

func TestSwitchStatement_String(t *testing.T) {

	t.Parallel()

	stmt := &SwitchStatement{
		Expression: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foo",
			},
		},
		Cases: []*SwitchCase{
			{
				Expression: &BoolExpression{
					Value: false,
				},
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "bar",
							},
						},
					},
				},
			},
			{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "baz",
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t,
		prettier.Concat{
			prettier.Group{
				Doc: prettier.Concat{
					switchStatementKeywordSpaceDoc,
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("foo"),
						},
					},
					prettier.Line{},
				},
			},
			prettier.Text("{"),
			prettier.Indent{
				Doc: prettier.Concat{
					prettier.HardLine{},
					prettier.Concat{
						switchCaseKeywordSpaceDoc,
						prettier.Text("false"),
						switchCaseColonSymbolDoc,
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.HardLine{},
								prettier.Text("bar"),
							},
						},
					},
					prettier.HardLine{},
					prettier.Concat{
						switchCaseDefaultKeywordSpaceDoc,
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.HardLine{},
								prettier.Text("baz"),
							},
						},
					},
				},
			},
			prettier.HardLine{},
			prettier.Text("}"),
		},
		stmt.Doc(),
	)
}

func TestSwitchStatement_Doc(t *testing.T) {

	t.Parallel()

	stmt := &SwitchStatement{
		Expression: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foo",
			},
		},
		Cases: []*SwitchCase{
			{
				Expression: &BoolExpression{
					Value: false,
				},
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "bar",
							},
						},
					},
				},
			},
			{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "baz",
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t,
		"switch foo {\n"+
			"    case false:\n"+
			"        bar\n"+
			"    default:\n"+
			"        baz\n"+
			"}",
		stmt.String(),
	)
}
