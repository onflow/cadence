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
)

func TestBlock_MarshalJSON(t *testing.T) {

	t.Parallel()

	block := &Block{
		Statements: []Statement{
			&ExpressionStatement{
				Expression: &BoolExpression{
					Value: false,
					Range: Range{
						StartPos: Position{Offset: 1, Line: 2, Column: 3},
						EndPos:   Position{Offset: 4, Line: 5, Column: 6},
					},
				},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
	}

	actual, err := json.Marshal(block)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "Block",
            "Statements": [
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
            ],
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestFunctionBlock_MarshalJSON(t *testing.T) {

	t.Parallel()

	t.Run("with statements", func(t *testing.T) {

		t.Parallel()

		block := &FunctionBlock{
			Block: &Block{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &BoolExpression{
							Value: false,
							Range: Range{
								StartPos: Position{Offset: 1, Line: 2, Column: 3},
								EndPos:   Position{Offset: 4, Line: 5, Column: 6},
							},
						},
					},
				},
				Range: Range{
					StartPos: Position{Offset: 7, Line: 8, Column: 9},
					EndPos:   Position{Offset: 10, Line: 11, Column: 12},
				},
			},
		}

		actual, err := json.Marshal(block)
		require.NoError(t, err)

		assert.JSONEq(t,
			`
            {
                "Type": "FunctionBlock",
                "Block": {
                    "Type": "Block",
                    "Statements": [
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
                    ],
                    "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                    "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
                },
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
            }
            `,
			string(actual),
		)
	})

	t.Run("with preconditions and postconditions", func(t *testing.T) {

		t.Parallel()

		block := &FunctionBlock{
			Block: &Block{
				Statements: []Statement{},
				Range: Range{
					StartPos: Position{Offset: 1, Line: 2, Column: 3},
					EndPos:   Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			PreConditions: &Conditions{
				{
					Kind: ConditionKindPre,
					Test: &BoolExpression{
						Value: false,
						Range: Range{
							StartPos: Position{Offset: 7, Line: 8, Column: 9},
							EndPos:   Position{Offset: 10, Line: 11, Column: 12},
						},
					},
					Message: &StringExpression{
						Value: "Pre failed",
						Range: Range{
							StartPos: Position{Offset: 13, Line: 14, Column: 15},
							EndPos:   Position{Offset: 16, Line: 17, Column: 18},
						},
					},
				},
			},
			PostConditions: &Conditions{
				{
					Kind: ConditionKindPost,
					Test: &BoolExpression{
						Value: true,
						Range: Range{
							StartPos: Position{Offset: 19, Line: 20, Column: 21},
							EndPos:   Position{Offset: 22, Line: 23, Column: 24},
						},
					},
					Message: &StringExpression{
						Value: "Post failed",
						Range: Range{
							StartPos: Position{Offset: 25, Line: 26, Column: 27},
							EndPos:   Position{Offset: 28, Line: 29, Column: 30},
						},
					},
				},
			},
		}

		actual, err := json.Marshal(block)
		require.NoError(t, err)

		assert.JSONEq(t,
			`
            {
                "Type": "FunctionBlock",
                "Block": {
                    "Type": "Block",
                    "Statements": [],
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
                },
                "PreConditions": [
                    {
                        "Kind": "ConditionKindPre",
                        "Test": {
                            "Type": "BoolExpression",
                            "Value": false,
                            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
                        },
                        "Message": {
                            "Type": "StringExpression",
                            "Value": "Pre failed",
                            "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
                            "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
                        }
                    }
                ],
                "PostConditions": [
                    {
                        "Kind": "ConditionKindPost",
                        "Test": {
                            "Type": "BoolExpression",
                            "Value": true,
                            "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
                            "EndPos": {"Offset": 22, "Line": 23, "Column": 24}
                        },
                        "Message": {
                            "Type": "StringExpression",
                            "Value": "Post failed",
                            "StartPos": {"Offset": 25, "Line": 26, "Column": 27},
                            "EndPos": {"Offset": 28, "Line": 29, "Column": 30}
                        }
                    }
                ],
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            }
            `,
			string(actual),
		)
	})
}
