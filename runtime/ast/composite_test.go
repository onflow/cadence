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

func TestFieldDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &FieldDeclaration{
		Access:       AccessPublic,
		Flags:        FieldDeclarationFlagsIsStatic | FieldDeclarationFlagsIsNative,
		VariableKind: VariableKindConstant,
		Identifier: Identifier{
			Identifier: "xyz",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		TypeAnnotation: &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
					Pos:        Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
		},
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 10, Line: 11, Column: 12},
			EndPos:   Position{Offset: 13, Line: 14, Column: 15},
		},
	}

	actual, err := json.Marshal(decl)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "FieldDeclaration",
            "Access": "AccessPublic",
            "IsStatic": true,
            "IsNative": true,
            "VariableKind": "VariableKindConstant",
            "Identifier": {
                "Identifier": "xyz",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
            "TypeAnnotation": {
                "IsResource": true,
                "AnnotatedType": {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "CD",
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                    },
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                },
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
            }, 
            "DocString": "test",
            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
            "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
        }
        `,
		string(actual),
	)
}

func TestFieldDeclaration_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with access, with kind, with static, with native", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			Access:       AccessPublic,
			VariableKind: VariableKindConstant,
			Flags:        FieldDeclarationFlagsIsNative | FieldDeclarationFlagsIsStatic,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("pub"),
					prettier.Text(" "),
					prettier.Text("static"),
					prettier.Text(" "),
					prettier.Text("native"),
					prettier.Text(" "),
					prettier.Text("let"),
					prettier.Text(" "),
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("xyz"),
							prettier.Text(": "),
							prettier.Concat{
								prettier.Text("@"),
								prettier.Text("CD"),
							},
						},
					},
				},
			},
			decl.Doc(),
		)
	})

	t.Run("without access, with kind", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			VariableKind: VariableKindConstant,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("let"),
					prettier.Text(" "),
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("xyz"),
							prettier.Text(": "),
							prettier.Concat{
								prettier.Text("@"),
								prettier.Text("CD"),
							},
						},
					},
				},
			},
			decl.Doc(),
		)
	})

	t.Run("with access, without kind", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			Access: AccessPublic,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("pub"),
					prettier.Text(" "),
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("xyz"),
							prettier.Text(": "),
							prettier.Concat{
								prettier.Text("@"),
								prettier.Text("CD"),
							},
						},
					},
				},
			},
			decl.Doc(),
		)
	})

	t.Run("without access, without kind", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("xyz"),
					prettier.Text(": "),
					prettier.Concat{
						prettier.Text("@"),
						prettier.Text("CD"),
					},
				},
			},
			decl.Doc(),
		)
	})

}

func TestFieldDeclaration_String(t *testing.T) {

	t.Parallel()

	t.Run("with access, with kind", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			Access:       AccessPublic,
			VariableKind: VariableKindConstant,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			"pub let xyz: @CD",
			decl.String(),
		)
	})

	t.Run("without access, with kind", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			VariableKind: VariableKindConstant,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			"let xyz: @CD",
			decl.String(),
		)
	})

	t.Run("with access, without kind", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			Access: AccessPublic,
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			"pub xyz: @CD",
			decl.String(),
		)

	})

	t.Run("without access, without kind", func(t *testing.T) {

		t.Parallel()

		decl := &FieldDeclaration{
			Identifier: Identifier{
				Identifier: "xyz",
			},
			TypeAnnotation: &TypeAnnotation{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
		}

		require.Equal(
			t,
			"xyz: @CD",
			decl.String(),
		)
	})

}

func TestCompositeDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	decl := &CompositeDeclaration{
		Access:        AccessPublic,
		CompositeKind: common.CompositeKindResource,
		Identifier: Identifier{
			Identifier: "AB",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		Conformances: []*NominalType{
			{
				Identifier: Identifier{
					Identifier: "CD",
					Pos:        Position{Offset: 4, Line: 5, Column: 6},
				},
			},
		},
		Members:   NewUnmeteredMembers([]Declaration{}),
		DocString: "test",
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
            "Type": "CompositeDeclaration",
            "Access": "AccessPublic", 
            "CompositeKind": "CompositeKindResource",
            "Identifier": {
                "Identifier": "AB",
				"StartPos": {"Offset": 1, "Line": 2, "Column": 3},
				"EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
            "Conformances": [
                {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "CD",
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                    },
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                }
            ], 
            "Members": {
                "Declarations": []
            },
            "DocString": "test",
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestCompositeDeclaration_Doc(t *testing.T) {

	t.Parallel()

	t.Run("no members, conformances", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{}),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("pub"),
				prettier.Text(" "),
				prettier.Text("resource"),
				prettier.Text(" "),
				prettier.Text("AB"),
				prettier.Text(":"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.Line{},
							prettier.Text("CD"),
							prettier.Concat{
								prettier.Text(","),
								prettier.Line{},
							},
							prettier.Text("EF"),
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

	t.Run("members, conformances", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{
				&FieldDeclaration{
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
			}),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("pub"),
				prettier.Text(" "),
				prettier.Text("resource"),
				prettier.Text(" "),
				prettier.Text("AB"),
				prettier.Text(":"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.Line{},
							prettier.Text("CD"),
							prettier.Concat{
								prettier.Text(","),
								prettier.Line{},
							},
							prettier.Text("EF"),
							prettier.Dedent{
								Doc: prettier.Concat{
									prettier.Line{},
									prettier.Concat{
										prettier.Text("{"),
										prettier.Indent{
											Doc: prettier.Concat{
												prettier.HardLine{},
												prettier.Group{
													Doc: prettier.Concat{
														prettier.Text("x"),
														prettier.Text(": "),
														prettier.Text("X"),
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
				},
			},
			decl.Doc(),
		)
	})

	t.Run("event", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindEvent,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Members: NewMembers(nil, []Declaration{
				&SpecialFunctionDeclaration{
					Kind: common.DeclarationKindInitializer,
					FunctionDeclaration: &FunctionDeclaration{
						ParameterList: &ParameterList{
							Parameters: []*Parameter{
								{
									Identifier: Identifier{Identifier: "e"},
									TypeAnnotation: &TypeAnnotation{
										Type: &NominalType{
											Identifier: Identifier{Identifier: "E"},
										},
									},
								},
							},
						},
					},
				},
			}),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("pub"),
				prettier.Text(" "),
				prettier.Text("event"),
				prettier.Text(" "),
				prettier.Text("AB"),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Concat{
									prettier.Text("e"),
									prettier.Text(": "),
									prettier.Text("E"),
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
			},
			decl.Doc(),
		)
	})

	t.Run("enum", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindEnum,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{
				&EnumCaseDeclaration{
					Identifier: Identifier{
						Identifier: "x",
					},
				},
			}),
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("pub"),
				prettier.Text(" "),
				prettier.Text("enum"),
				prettier.Text(" "),
				prettier.Text("AB"),
				prettier.Text(":"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.Line{},
							prettier.Text("CD"),
							prettier.Dedent{
								Doc: prettier.Concat{
									prettier.Line{},
									prettier.Concat{
										prettier.Text("{"),
										prettier.Indent{
											Doc: prettier.Concat{
												prettier.HardLine{},
												prettier.Concat{
													prettier.Text("case "),
													prettier.Text("x"),
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
				},
			},
			decl.Doc(),
		)
	})
}

func TestCompositeDeclaration_String(t *testing.T) {

	t.Parallel()

	t.Run("no members, conformances", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{}),
		}

		require.Equal(
			t,
			"pub resource AB: CD, EF {}",
			decl.String(),
		)
	})

	t.Run("members, conformances", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindResource,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{
				&FieldDeclaration{
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
			}),
		}

		require.Equal(
			t,
			"pub resource AB: CD, EF {\n"+
				"    x: X\n"+
				"}",
			decl.String(),
		)
	})

	t.Run("event", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindEvent,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Members: NewMembers(nil, []Declaration{
				&SpecialFunctionDeclaration{
					Kind: common.DeclarationKindInitializer,
					FunctionDeclaration: &FunctionDeclaration{
						ParameterList: &ParameterList{
							Parameters: []*Parameter{
								{
									Identifier: Identifier{Identifier: "e"},
									TypeAnnotation: &TypeAnnotation{
										Type: &NominalType{
											Identifier: Identifier{Identifier: "E"},
										},
									},
								},
							},
						},
					},
				},
			}),
		}

		require.Equal(
			t,
			"pub event AB(e: E)",
			decl.String(),
		)
	})

	t.Run("enum", func(t *testing.T) {

		t.Parallel()

		decl := &CompositeDeclaration{
			Access:        AccessPublic,
			CompositeKind: common.CompositeKindEnum,
			Identifier: Identifier{
				Identifier: "AB",
			},
			Conformances: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
			},
			Members: NewMembers(nil, []Declaration{
				&EnumCaseDeclaration{
					Identifier: Identifier{
						Identifier: "x",
					},
				},
			}),
		}

		require.Equal(
			t,
			"pub enum AB: CD {\n"+
				"    case x\n"+
				"}",
			decl.String(),
		)
	})
}

func TestEnumCaseDeclaration_Doc(t *testing.T) {

	t.Parallel()

	decl := &EnumCaseDeclaration{
		Identifier: Identifier{
			Identifier: "x",
		},
	}

	require.Equal(t,
		prettier.Concat{
			prettier.Text("case "),
			prettier.Text("x"),
		},
		decl.Doc(),
	)
}

func TestEnumCaseDeclaration_String(t *testing.T) {

	t.Parallel()

	decl := &EnumCaseDeclaration{
		Identifier: Identifier{
			Identifier: "x",
		},
	}

	require.Equal(t,
		"case x",
		decl.String(),
	)
}
