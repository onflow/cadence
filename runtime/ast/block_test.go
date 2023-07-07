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
		// language=json
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

func TestBlock_Doc(t *testing.T) {

	t.Parallel()

	block := &Block{
		Statements: []Statement{
			&ExpressionStatement{
				Expression: &BoolExpression{
					Value: false,
				},
			},
			&ExpressionStatement{
				Expression: &StringExpression{
					Value: "test",
				},
			},
		},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("{"),
			prettier.Indent{
				Doc: prettier.Concat{
					prettier.HardLine{},
					prettier.Text("false"),
					prettier.HardLine{},
					prettier.Text("\"test\""),
				},
			},
			prettier.HardLine{},
			prettier.Text("}"),
		},
		block.Doc(),
	)
}

func TestBlock_String(t *testing.T) {

	t.Parallel()

	block := &Block{
		Statements: []Statement{
			&ExpressionStatement{
				Expression: &BoolExpression{
					Value: false,
				},
			},
			&ExpressionStatement{
				Expression: &StringExpression{
					Value: "test",
				},
			},
		},
	}

	require.Equal(
		t,
		`{
    false
    "test"
}`,
		block.String(),
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
			// language=json
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
				&TestCondition{
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
				&EmitCondition{
					InvocationExpression: &InvocationExpression{
						InvokedExpression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "foobar",
								Pos:        Position{Offset: 31, Line: 32, Column: 33},
							},
						},
						TypeArguments:     []*TypeAnnotation{},
						Arguments:         []*Argument{},
						ArgumentsStartPos: Position{Offset: 34, Line: 35, Column: 36},
						EndPos:            Position{Offset: 37, Line: 38, Column: 39},
					},
					StartPos: Position{Offset: 40, Line: 41, Column: 42},
				},
			},
			PostConditions: &Conditions{
				&TestCondition{
					Test: &BoolExpression{
						Value: true,
						Range: Range{
							StartPos: Position{Offset: 19, Line: 20, Column: 21},
							EndPos:   Position{Offset: 22, Line: 23, Column: 24},
						},
					},
				},
			},
		}

		actual, err := json.Marshal(block)
		require.NoError(t, err)

		assert.JSONEq(t,
			// language=json
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
									"Type": "TestCondition",
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
									},
									"StartPos": {"Offset": 7, "Line": 8, "Column": 9},
									"EndPos": {"Offset": 16, "Line": 17, "Column": 18}
								},
								{
									"Type": "EmitCondition",
									"InvocationExpression": {
										"Type": "InvocationExpression",
										"InvokedExpression": {
										   "Type": "IdentifierExpression",
										   "Identifier": {
											   "Identifier": "foobar",
											   "StartPos": {"Offset": 31, "Line": 32, "Column": 33},
											   "EndPos": {"Offset": 36, "Line": 32, "Column": 38}
										   },
										   "StartPos": {"Offset": 31, "Line": 32, "Column": 33},
										   "EndPos": {"Offset": 36, "Line": 32, "Column": 38}
										},
										"TypeArguments": [],
										"Arguments": [],
										"ArgumentsStartPos": {"Offset": 34, "Line": 35, "Column": 36},
										"StartPos": {"Offset": 31, "Line": 32, "Column": 33},
										"EndPos": {"Offset": 37, "Line": 38, "Column": 39}
									},
									"StartPos": {"Offset": 40, "Line": 41, "Column": 42},
									"EndPos": {"Offset": 37, "Line": 38, "Column": 39}
								}
							],
							"PostConditions": [
								{
									"Type": "TestCondition",
									"Test": {
										"Type": "BoolExpression",
										"Value": true,
										"StartPos": {"Offset": 19, "Line": 20, "Column": 21},
										"EndPos": {"Offset": 22, "Line": 23, "Column": 24}
									},
									"Message": null,
									"StartPos": {"Offset": 19, "Line": 20, "Column": 21},
									"EndPos": {"Offset": 22, "Line": 23, "Column": 24}
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

func TestFunctionBlock_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with statements", func(t *testing.T) {

		t.Parallel()

		block := &FunctionBlock{
			Block: &Block{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &BoolExpression{
							Value: false,
						},
					},
					&ExpressionStatement{
						Expression: &StringExpression{
							Value: "test",
						},
					},
				},
			},
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("{"),
				prettier.Indent{
					Doc: prettier.Concat{
						prettier.HardLine{},
						prettier.Text("false"),
						prettier.HardLine{},
						prettier.Text("\"test\""),
					}},
				prettier.HardLine{},
				prettier.Text("}"),
			},
			block.Doc(),
		)
	})

	t.Run("with preconditions and postconditions", func(t *testing.T) {

		t.Parallel()

		block := &FunctionBlock{
			Block: &Block{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &BoolExpression{
							Value: false,
						},
					},
					&ExpressionStatement{
						Expression: &StringExpression{
							Value: "test",
						},
					},
				},
			},
			PreConditions: &Conditions{
				&TestCondition{
					Test: &BoolExpression{
						Value: false,
					},
					Message: &StringExpression{
						Value: "Pre failed",
					},
				},
				&EmitCondition{
					InvocationExpression: &InvocationExpression{
						InvokedExpression: &IdentifierExpression{
							Identifier: Identifier{
								Identifier: "Foo",
							},
						},
					},
				},
			},
			PostConditions: &Conditions{
				&TestCondition{
					Test: &BoolExpression{
						Value: true,
					},
				},
			},
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("{"),
				prettier.Indent{
					Doc: prettier.Concat{
						prettier.HardLine{},
						prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("pre"),
								prettier.Text(" "),
								prettier.Text("{"),
								prettier.Indent{
									Doc: prettier.Concat{
										prettier.HardLine{},
										prettier.Group{
											Doc: prettier.Concat{
												prettier.Text("false"),
												prettier.Text(":"),
												prettier.Indent{
													Doc: prettier.Concat{
														prettier.HardLine{},
														prettier.Text("\"Pre failed\""),
													},
												},
											},
										},
										prettier.HardLine{},
										prettier.Concat{
											prettier.Text("emit "),
											prettier.Concat{
												prettier.Text("Foo"),
												prettier.Text("()"),
											},
										},
									},
								},
								prettier.HardLine{},
								prettier.Text("}"),
							}},
						prettier.HardLine{},
						prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("post"),
								prettier.Text(" "),
								prettier.Text("{"),
								prettier.Indent{
									Doc: prettier.Concat{
										prettier.HardLine{},
										prettier.Group{
											Doc: prettier.Text("true"),
										},
									},
								},
								prettier.HardLine{},
								prettier.Text("}"),
							}},
						prettier.Concat{
							prettier.HardLine{},
							prettier.Text("false"),
							prettier.HardLine{},
							prettier.Text("\"test\""),
						},
					}},
				prettier.HardLine{},
				prettier.Text("}"),
			},
			block.Doc(),
		)
	})
}

func TestFunctionBlock_String(t *testing.T) {

	t.Parallel()

	t.Run("with statements", func(t *testing.T) {

		t.Parallel()

		block := &FunctionBlock{
			Block: &Block{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &BoolExpression{
							Value: false,
						},
					},
					&ExpressionStatement{
						Expression: &StringExpression{
							Value: "test",
						},
					},
				},
			},
		}

		require.Equal(
			t,
			"{\n"+
				"    false\n"+
				"    \"test\"\n"+
				"}",
			block.String(),
		)
	})

	t.Run("with preconditions and postconditions", func(t *testing.T) {

		t.Parallel()

		block := &FunctionBlock{
			Block: &Block{
				Statements: []Statement{
					&ExpressionStatement{
						Expression: &BoolExpression{
							Value: false,
						},
					},
					&ExpressionStatement{
						Expression: &StringExpression{
							Value: "test",
						},
					},
				},
			},
			PreConditions: &Conditions{
				&TestCondition{
					Test: &BoolExpression{
						Value: false,
					},
					Message: &StringExpression{
						Value: "Pre failed",
					},
				},
			},
			PostConditions: &Conditions{
				&TestCondition{
					Test: &BoolExpression{
						Value: true,
					},
					Message: &StringExpression{
						Value: "Post failed",
					},
				},
			},
		}

		require.Equal(
			t,
			"{\n"+
				"    pre {\n"+
				"        false:\n"+
				"            \"Pre failed\"\n"+
				"    }\n"+
				"    post {\n"+
				"        true:\n"+
				"            \"Post failed\"\n"+
				"    }\n"+
				"    false\n"+
				"    \"test\"\n"+
				"}",
			block.String(),
		)
	})
}
