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

	"github.com/onflow/cadence/runtime/common"
)

func TestFunctionDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &FunctionDeclaration{
		Access: AccessPublic,
		Identifier: Identifier{
			Identifier: "xyz",
			Pos:        Position{Offset: 37, Line: 38, Column: 39},
		},
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
					Label: "ok",
					Identifier: Identifier{
						Identifier: "foobar",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
					TypeAnnotation: &TypeAnnotation{
						Type: &NominalType{
							Identifier: Identifier{
								Identifier: "AB",
								Pos:        Position{Offset: 4, Line: 5, Column: 6},
							},
						},
						StartPos: Position{Offset: 7, Line: 8, Column: 9},
					},
					Range: Range{
						StartPos: Position{Offset: 10, Line: 11, Column: 12},
						EndPos:   Position{Offset: 13, Line: 14, Column: 15},
					},
				},
			},
			Range: Range{
				StartPos: Position{Offset: 16, Line: 17, Column: 18},
				EndPos:   Position{Offset: 19, Line: 20, Column: 21},
			},
		},
		ReturnTypeAnnotation: &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
					Pos:        Position{Offset: 22, Line: 23, Column: 24},
				},
			},
			StartPos: Position{Offset: 25, Line: 26, Column: 27},
		},
		FunctionBlock: &FunctionBlock{
			Block: &Block{
				Statements: []Statement{},
				Range: Range{
					StartPos: Position{Offset: 28, Line: 29, Column: 30},
					EndPos:   Position{Offset: 31, Line: 32, Column: 33},
				},
			},
		},
		DocString: "test",
		StartPos:  Position{Offset: 34, Line: 35, Column: 36},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "FunctionDeclaration",
            "Access": "AccessPublic",
            "Identifier": {
                "Identifier": "xyz",
				"StartPos": {"Offset": 37, "Line": 38, "Column": 39},
				"EndPos": {"Offset": 39, "Line": 38, "Column": 41}
            },
            "ParameterList": {
                "Parameters": [
                    {
                        "Label": "ok",
                        "Identifier": {
                            "Identifier": "foobar",
                            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                            "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                        },
                        "TypeAnnotation": {
                            "IsResource": false,
                            "AnnotatedType": {
                                "Type": "NominalType",
                                "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                                "EndPos": {"Offset": 5, "Line": 5, "Column": 7},
                                "Identifier": {
                                    "Identifier": "AB",
                                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                                }
                            },
                            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                            "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                        },
                        "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                        "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
                    }
                ],
                "StartPos": {"Offset": 16, "Line": 17, "Column": 18},
                "EndPos": {"Offset": 19, "Line": 20, "Column": 21}
            },
			"Purity": 0,
            "ReturnTypeAnnotation": {
                "IsResource": true,
                "AnnotatedType": {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "CD",
                        "StartPos": {"Offset": 22, "Line": 23, "Column": 24},
                        "EndPos": {"Offset": 23, "Line": 23, "Column": 25}
                    },
                    "StartPos": {"Offset": 22, "Line": 23, "Column": 24},
                    "EndPos": {"Offset": 23, "Line": 23, "Column": 25}
                },
                "StartPos": {"Offset": 25, "Line": 26, "Column": 27},
                "EndPos": {"Offset": 23, "Line": 23, "Column": 25}
            },
            "FunctionBlock": {
                "Type": "FunctionBlock",
                "Block": {
                    "Type": "Block",
                    "Statements": [],
                    "StartPos": {"Offset": 28, "Line": 29, "Column": 30},
                    "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
                },
                "StartPos": {"Offset": 28, "Line": 29, "Column": 30},
                "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
            },
            "DocString": "test",
            "StartPos": {"Offset": 34, "Line": 35, "Column": 36},
            "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
        }
        `,
		string(actual),
	)
}

func TestFunctionDeclaration_Doc(t *testing.T) {

	t.Parallel()

	decl := &FunctionDeclaration{
		Access: AccessPublic,
		Identifier: Identifier{
			Identifier: "xyz",
		},
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
					Label: "ok",
					Identifier: Identifier{
						Identifier: "foobar",
					},
					TypeAnnotation: &TypeAnnotation{
						Type: &NominalType{
							Identifier: Identifier{
								Identifier: "AB",
							},
						},
					},
				},
			},
		},
		ReturnTypeAnnotation: &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
				},
			},
		},
		FunctionBlock: &FunctionBlock{
			Block: &Block{
				Statements: []Statement{},
			},
		},
	}

	require.Equal(t,
		prettier.Concat{
			prettier.Text("pub"),
			prettier.Space,
			prettier.Text("fun "),
			prettier.Text("xyz"),
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("("),
							prettier.Indent{
								Doc: prettier.Concat{
									prettier.SoftLine{},
									prettier.Concat{
										prettier.Text("ok"),
										prettier.Text(" "),
										prettier.Text("foobar"),
										prettier.Text(": "),
										prettier.Text("AB"),
									},
								},
							},
							prettier.SoftLine{},
							prettier.Text(")"),
						},
					},
					prettier.Text(": "),
					prettier.Concat{
						prettier.Text("@"),
						prettier.Text("CD"),
					},
				},
			},
			prettier.Text(" {}"),
		},
		decl.Doc(),
	)
}

func TestFunctionDeclaration_String(t *testing.T) {

	t.Parallel()

	decl := &FunctionDeclaration{
		Access: AccessPublic,
		Identifier: Identifier{
			Identifier: "xyz",
		},
		ParameterList: &ParameterList{
			Parameters: []*Parameter{
				{
					Label: "ok",
					Identifier: Identifier{
						Identifier: "foobar",
					},
					TypeAnnotation: &TypeAnnotation{
						Type: &NominalType{
							Identifier: Identifier{
								Identifier: "AB",
							},
						},
					},
				},
			},
		},
		ReturnTypeAnnotation: &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
				},
			},
		},
		FunctionBlock: &FunctionBlock{
			Block: &Block{
				Statements: []Statement{},
			},
		},
	}

	require.Equal(t,
		"pub fun xyz(ok foobar: AB): @CD {}",
		decl.String(),
	)
}

func TestSpecialFunctionDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &SpecialFunctionDeclaration{
		Kind: common.DeclarationKindInitializer,
		FunctionDeclaration: &FunctionDeclaration{
			Access: AccessNotSpecified,
			Identifier: Identifier{
				Identifier: "xyz",
				Pos:        Position{Offset: 37, Line: 38, Column: 39},
			},
			ParameterList: &ParameterList{
				Parameters: []*Parameter{
					{
						Label: "ok",
						Identifier: Identifier{
							Identifier: "foobar",
							Pos:        Position{Offset: 1, Line: 2, Column: 3},
						},
						TypeAnnotation: &TypeAnnotation{
							Type: &NominalType{
								Identifier: Identifier{
									Identifier: "AB",
									Pos:        Position{Offset: 4, Line: 5, Column: 6},
								},
							},
							StartPos: Position{Offset: 7, Line: 8, Column: 9},
						},
						Range: Range{
							StartPos: Position{Offset: 10, Line: 11, Column: 12},
							EndPos:   Position{Offset: 13, Line: 14, Column: 15},
						},
					},
				},
				Range: Range{
					StartPos: Position{Offset: 16, Line: 17, Column: 18},
					EndPos:   Position{Offset: 19, Line: 20, Column: 21},
				},
			},
			ReturnTypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
						Pos:        Position{Offset: 22, Line: 23, Column: 24},
					},
				},
				StartPos: Position{Offset: 25, Line: 26, Column: 27},
			},
			FunctionBlock: &FunctionBlock{
				Block: &Block{
					Statements: []Statement{},
					Range: Range{
						StartPos: Position{Offset: 28, Line: 29, Column: 30},
						EndPos:   Position{Offset: 31, Line: 32, Column: 33},
					},
				},
			},
			DocString: "test",
			StartPos:  Position{Offset: 34, Line: 35, Column: 36},
		},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "SpecialFunctionDeclaration",
            "Kind": "DeclarationKindInitializer",
            "FunctionDeclaration": {
                "Type": "FunctionDeclaration",
                "Access": "AccessNotSpecified",
                "Identifier": {
                    "Identifier": "xyz",
		    		"StartPos": {"Offset": 37, "Line": 38, "Column": 39},
		    		"EndPos": {"Offset": 39, "Line": 38, "Column": 41}
                },
                "ParameterList": {
                    "Parameters": [
                        {
                            "Label": "ok",
                            "Identifier": {
                                "Identifier": "foobar",
                                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                            },
                            "TypeAnnotation": {
                                "IsResource": false,
                                "AnnotatedType": {
                                    "Type": "NominalType",
                                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7},
                                    "Identifier": {
                                        "Identifier": "AB",
                                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                                        "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                                    }
                                },
                                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                                "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                            },
                            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                            "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
                        }
                    ],
                    "StartPos": {"Offset": 16, "Line": 17, "Column": 18},
                    "EndPos": {"Offset": 19, "Line": 20, "Column": 21}
                },
				"Purity": 0,
                "ReturnTypeAnnotation": {
                    "IsResource": true,
                    "AnnotatedType": {
                        "Type": "NominalType",
                        "Identifier": {
                            "Identifier": "CD",
                            "StartPos": {"Offset": 22, "Line": 23, "Column": 24},
                            "EndPos": {"Offset": 23, "Line": 23, "Column": 25}
                        },
                        "StartPos": {"Offset": 22, "Line": 23, "Column": 24},
                        "EndPos": {"Offset": 23, "Line": 23, "Column": 25}
                    },
                    "StartPos": {"Offset": 25, "Line": 26, "Column": 27},
                    "EndPos": {"Offset": 23, "Line": 23, "Column": 25}
                },
                "FunctionBlock": {
                    "Type": "FunctionBlock",
                    "Block": {
                        "Type": "Block",
                        "Statements": [],
                        "StartPos": {"Offset": 28, "Line": 29, "Column": 30},
                        "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
                    },
                    "StartPos": {"Offset": 28, "Line": 29, "Column": 30},
                    "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
                },
                "DocString": "test",
                "StartPos": {"Offset": 34, "Line": 35, "Column": 36},
                "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
            },
            "StartPos": {"Offset": 34, "Line": 35, "Column": 36},
            "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
        }
        `,
		string(actual),
	)
}

func TestSpecialFunctionDeclaration_Doc(t *testing.T) {

	t.Parallel()

	decl := &SpecialFunctionDeclaration{
		Kind: common.DeclarationKindInitializer,
		FunctionDeclaration: &FunctionDeclaration{
			Access: AccessNotSpecified,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			ParameterList: &ParameterList{
				Parameters: []*Parameter{
					{
						Label: "ok",
						Identifier: Identifier{
							Identifier: "foobar",
						},
						TypeAnnotation: &TypeAnnotation{
							Type: &NominalType{
								Identifier: Identifier{
									Identifier: "AB",
								},
							},
						},
					},
				},
			},
			ReturnTypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
			FunctionBlock: &FunctionBlock{
				Block: &Block{
					Statements: []Statement{},
				},
			},
		},
	}

	require.Equal(t,
		prettier.Concat{
			prettier.Text("init"),
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("("),
							prettier.Indent{
								Doc: prettier.Concat{
									prettier.SoftLine{},
									prettier.Concat{
										prettier.Text("ok"),
										prettier.Text(" "),
										prettier.Text("foobar"),
										prettier.Text(": "),
										prettier.Text("AB"),
									},
								},
							},
							prettier.SoftLine{},
							prettier.Text(")"),
						},
					},
					prettier.Text(": "),
					prettier.Concat{
						prettier.Text("@"),
						prettier.Text("CD"),
					},
				},
			},
			prettier.Text(" {}"),
		},
		decl.Doc(),
	)
}

func TestSpecialFunctionDeclaration_String(t *testing.T) {

	t.Parallel()

	decl := &SpecialFunctionDeclaration{
		Kind: common.DeclarationKindInitializer,
		FunctionDeclaration: &FunctionDeclaration{
			Access: AccessNotSpecified,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			ParameterList: &ParameterList{
				Parameters: []*Parameter{
					{
						Label: "ok",
						Identifier: Identifier{
							Identifier: "foobar",
						},
						TypeAnnotation: &TypeAnnotation{
							Type: &NominalType{
								Identifier: Identifier{
									Identifier: "AB",
								},
							},
						},
					},
				},
			},
			ReturnTypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
			FunctionBlock: &FunctionBlock{
				Block: &Block{
					Statements: []Statement{},
				},
			},
		},
	}

	require.Equal(t,
		"init(ok foobar: AB): @CD {}",
		decl.String(),
	)
}
