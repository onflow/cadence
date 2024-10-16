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

	"github.com/onflow/cadence/runtime/common"
)

func TestTransactionDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &TransactionDeclaration{
		ParameterList: &ParameterList{
			Parameters: []*Parameter{},
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Fields:  []*FieldDeclaration{},
		Prepare: nil,
		PreConditions: &Conditions{
			Range: Range{
				StartPos: Position{Offset: 13, Line: 14, Column: 15},
				EndPos:   Position{Offset: 16, Line: 17, Column: 18},
			},
		},
		PostConditions: &Conditions{
			Range: Range{
				StartPos: Position{Offset: 19, Line: 20, Column: 21},
				EndPos:   Position{Offset: 22, Line: 23, Column: 24},
			},
		},
		DocString: "test",
		Execute:   nil,
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
            "Type": "TransactionDeclaration",
            "ParameterList":  {
                "Parameters": [],
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
		    "Fields":         [],
		    "Prepare":        null,
		    "PreConditions":  {
                "Conditions": null,
                "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
                "EndPos":  {"Offset": 16, "Line": 17, "Column": 18}
		    },
		    "PostConditions": {
                "Conditions": null,
                "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
                "EndPos": {"Offset": 22, "Line": 23, "Column": 24}
		    },
		    "Execute":        null,
            "DocString":      "test",
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestTransactionDeclaration_Doc(t *testing.T) {

	t.Parallel()

	decl := &TransactionDeclaration{
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
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
			},
		},
		Fields: []*FieldDeclaration{
			{
				Access:       AccessAll,
				VariableKind: VariableKindConstant,
				Identifier: Identifier{
					Identifier: "f",
				},
				TypeAnnotation: &TypeAnnotation{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "F",
						},
					},
				},
			},
		},
		Prepare: &SpecialFunctionDeclaration{
			Kind: common.DeclarationKindPrepare,
			FunctionDeclaration: &FunctionDeclaration{
				Access: AccessNotSpecified,
				ParameterList: &ParameterList{
					Parameters: []*Parameter{
						{
							Identifier: Identifier{
								Identifier: "signer",
							},
							TypeAnnotation: &TypeAnnotation{
								Type: &NominalType{
									Identifier: Identifier{
										Identifier: "AuthAccount",
									},
								},
							},
						},
					},
				},
				FunctionBlock: &FunctionBlock{
					Block: &Block{
						Statements: []Statement{},
					},
				},
			},
		},
		PreConditions: &Conditions{
			Conditions: []Condition{
				&TestCondition{
					Test: &BoolExpression{
						Value: true,
					},
					Message: &StringExpression{
						Value: "pre",
					},
				},
			},
		},
		Execute: &SpecialFunctionDeclaration{
			Kind: common.DeclarationKindExecute,
			FunctionDeclaration: &FunctionDeclaration{
				Access: AccessNotSpecified,
				FunctionBlock: &FunctionBlock{
					Block: &Block{
						Statements: []Statement{
							&ExpressionStatement{
								Expression: &StringExpression{
									Value: "xyz",
								},
							},
						},
					},
				},
			},
		},
		PostConditions: &Conditions{
			Conditions: []Condition{
				&TestCondition{
					Test: &BoolExpression{
						Value: false,
					},
					Message: &StringExpression{
						Value: "post",
					},
				},
			},
		},
	}

	require.Equal(
		t,
		prettier.Concat{
			prettier.Text("transaction"),
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("("),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Concat{
								prettier.Text("x"),
								prettier.Text(": "),
								prettier.Text("X"),
							},
						},
					},
					prettier.SoftLine{},
					prettier.Text(")"),
				},
			},
			prettier.Text(" "),
			prettier.Text("{"),
			prettier.Indent{
				Doc: prettier.Concat{
					prettier.Concat{
						prettier.HardLine{},
						prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("access(all)"),
								prettier.HardLine{},
								prettier.Concat{
									prettier.Text("let"),
									prettier.Text(" "),
									prettier.Group{
										Doc: prettier.Concat{
											prettier.Text("f"),
											prettier.Text(": "),
											prettier.Concat{
												prettier.Text("@"),
												prettier.Text("F"),
											},
										},
									},
								},
							},
						},
					},
					prettier.HardLine{},
					prettier.Concat{
						prettier.HardLine{},
						prettier.Concat{
							prettier.Text("prepare"),
							prettier.Group{
								Doc: prettier.Concat{
									prettier.Group{
										Doc: prettier.Concat{
											prettier.Text("("),
											prettier.Indent{
												Doc: prettier.Concat{
													prettier.SoftLine{},
													prettier.Concat{
														prettier.Text("signer"),
														prettier.Text(": "),
														prettier.Text("AuthAccount"),
													},
												},
											},
											prettier.SoftLine{},
											prettier.Text(")"),
										},
									},
								},
							},
							prettier.Text(" "),
							prettier.Text("{}"),
						},
					},
					prettier.HardLine{},
					prettier.Concat{
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
												prettier.Text("true"),
												prettier.Text(":"),
												prettier.Indent{
													Doc: prettier.Concat{
														prettier.HardLine{},
														prettier.Text("\"pre\""),
													},
												},
											},
										},
									},
								},
								prettier.HardLine{},
								prettier.Text("}"),
							},
						},
					},
					prettier.HardLine{},
					prettier.Concat{
						prettier.HardLine{},
						prettier.Concat{
							prettier.Text("execute"),
							prettier.Text(" "),
							prettier.Concat{
								prettier.Text("{"),
								prettier.Indent{
									Doc: prettier.Concat{
										prettier.HardLine{},
										prettier.Text("\"xyz\""),
									},
								},
								prettier.HardLine{},
								prettier.Text("}"),
							},
						},
					},
					prettier.HardLine{},
					prettier.Concat{
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
											Doc: prettier.Concat{
												prettier.Text("false"),
												prettier.Text(":"),
												prettier.Indent{
													Doc: prettier.Concat{
														prettier.HardLine{},
														prettier.Text("\"post\""),
													},
												},
											},
										},
									},
								},
								prettier.HardLine{},
								prettier.Text("}"),
							},
						},
					},
				},
			},
			prettier.HardLine{},
			prettier.Text("}"),
		},
		decl.Doc(),
	)
}

func TestTransactionDeclaration_String(t *testing.T) {

	t.Parallel()

	decl := &TransactionDeclaration{
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
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
			},
		},
		Fields: []*FieldDeclaration{
			{
				Access:       AccessAll,
				VariableKind: VariableKindConstant,
				Identifier: Identifier{
					Identifier: "f",
				},
				TypeAnnotation: &TypeAnnotation{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "F",
						},
					},
				},
			},
		},
		Prepare: &SpecialFunctionDeclaration{
			Kind: common.DeclarationKindPrepare,
			FunctionDeclaration: &FunctionDeclaration{
				Access: AccessNotSpecified,
				ParameterList: &ParameterList{
					Parameters: []*Parameter{
						{
							Identifier: Identifier{
								Identifier: "signer",
							},
							TypeAnnotation: &TypeAnnotation{
								Type: &ReferenceType{
									Type: &NominalType{
										Identifier: Identifier{
											Identifier: "Account",
										},
									},
								},
							},
						},
					},
				},
				FunctionBlock: &FunctionBlock{
					Block: &Block{
						Statements: []Statement{},
					},
				},
			},
		},
		PreConditions: &Conditions{
			Conditions: []Condition{
				&TestCondition{
					Test: &BoolExpression{
						Value: true,
					},
					Message: &StringExpression{
						Value: "pre",
					},
				},
			},
		},
		Execute: &SpecialFunctionDeclaration{
			Kind: common.DeclarationKindExecute,
			FunctionDeclaration: &FunctionDeclaration{
				Access: AccessNotSpecified,
				FunctionBlock: &FunctionBlock{
					Block: &Block{
						Statements: []Statement{
							&ExpressionStatement{
								Expression: &StringExpression{
									Value: "xyz",
								},
							},
						},
					},
				},
			},
		},
		PostConditions: &Conditions{
			Conditions: []Condition{
				&TestCondition{
					Test: &BoolExpression{
						Value: false,
					},
					Message: &StringExpression{
						Value: "post",
					},
				},
			},
		},
	}

	require.Equal(
		t,
		`transaction(x: X) {
    access(all)
    let f: @F
    
    prepare(signer: &Account) {}
    
    pre {
        true:
            "pre"
    }
    
    execute {
        "xyz"
    }
    
    post {
        false:
            "post"
    }
}`,
		decl.String(),
	)
}
