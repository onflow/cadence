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
)

func TestExpressionStatement_MarshalJSON(t *testing.T) {

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

func TestReturnStatement_MarshalJSON(t *testing.T) {

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

func TestBreakStatement_MarshalJSON(t *testing.T) {

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

func TestContinueStatement_MarshalJSON(t *testing.T) {

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

func TestIfStatement_MarshalJSON(t *testing.T) {

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

func TestWhileStatement_MarshalJSON(t *testing.T) {

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
