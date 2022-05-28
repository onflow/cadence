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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"
)

func TestBoolExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &BoolExpression{
		Value: false,
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "BoolExpression",
            "Value": false,
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestBoolExpression_Doc(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		prettier.Text("true"),
		(&BoolExpression{Value: true}).Doc(),
	)

	assert.Equal(t,
		prettier.Text("false"),
		(&BoolExpression{Value: false}).Doc(),
	)
}

func TestBoolExpression_String(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"true",
		(&BoolExpression{Value: true}).String(),
	)

	assert.Equal(t,
		"false",
		(&BoolExpression{Value: false}).String(),
	)
}

func TestNilExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &NilExpression{
		Pos: Position{Offset: 1, Line: 2, Column: 3},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "NilExpression",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
        }
        `,
		string(actual),
	)
}

func TestNilExpression_Doc(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		prettier.Text("nil"),
		(&NilExpression{}).Doc(),
	)
}

func TestNilExpression_String(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"nil",
		(&NilExpression{}).String(),
	)
}

func TestStringExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &StringExpression{
		Value: "Hello, World!",
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "StringExpression",
            "Value": "Hello, World!",
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestStringExpression_Doc(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		prettier.Text(`"test"`),
		(&StringExpression{Value: "test"}).Doc(),
	)
}

func TestStringExpression_String(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		`"test"`,
		(&StringExpression{Value: "test"}).String(),
	)
}

func TestIntegerExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &IntegerExpression{
		PositiveLiteral: "4_2",
		Value:           big.NewInt(42),
		Base:            10,
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "IntegerExpression",
            "PositiveLiteral": "4_2",
            "Value": "42",
            "Base": 10,
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestIntegerExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("decimal", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "4_2",
			Value:           big.NewInt(42),
			Base:            10,
		}

		assert.Equal(t,
			prettier.Text(`4_2`),
			expr.Doc(),
		)
	})

	t.Run("negative", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "4_2",
			Value:           big.NewInt(-42),
			Base:            10,
		}

		assert.Equal(t,
			prettier.Text(`-4_2`),
			expr.Doc(),
		)
	})

	t.Run("binary", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "0b10_10_10",
			Value:           big.NewInt(42),
			Base:            2,
		}

		assert.Equal(t,
			prettier.Text(`0b10_10_10`),
			expr.Doc(),
		)
	})

	t.Run("octal", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "0o5_2",
			Value:           big.NewInt(42),
			Base:            8,
		}

		assert.Equal(t,
			prettier.Text(`0o5_2`),
			expr.Doc(),
		)
	})

	t.Run("hex", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "0x2_A",
			Value:           big.NewInt(42),
			Base:            16,
		}

		assert.Equal(t,
			prettier.Text(`0x2_A`),
			expr.Doc(),
		)
	})
}

func TestIntegerExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("decimal", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "4_2",
			Value:           big.NewInt(42),
			Base:            10,
		}

		assert.Equal(t,
			"4_2",
			expr.String(),
		)
	})

	t.Run("negative", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "4_2",
			Value:           big.NewInt(-42),
			Base:            10,
		}

		assert.Equal(t,
			"-4_2",
			expr.String(),
		)
	})

	t.Run("binary", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "0b10_10_10",
			Value:           big.NewInt(42),
			Base:            2,
		}

		assert.Equal(t,
			"0b10_10_10",
			expr.String(),
		)
	})

	t.Run("octal", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "0o5_2",
			Value:           big.NewInt(42),
			Base:            8,
		}

		assert.Equal(t,
			"0o5_2",
			expr.String(),
		)
	})

	t.Run("hex", func(t *testing.T) {

		t.Parallel()

		expr := &IntegerExpression{
			PositiveLiteral: "0x2_A",
			Value:           big.NewInt(42),
			Base:            16,
		}

		assert.Equal(t,
			"0x2_A",
			expr.String(),
		)
	})
}

func TestFixedPointExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &FixedPointExpression{
		PositiveLiteral: "42.2400000000",
		Negative:        true,
		UnsignedInteger: big.NewInt(42),
		Fractional:      big.NewInt(24),
		Scale:           10,
		Range: Range{
			StartPos: Position{Offset: 1, Line: 2, Column: 3},
			EndPos:   Position{Offset: 4, Line: 5, Column: 6},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "FixedPointExpression",
            "PositiveLiteral": "42.2400000000",
            "Negative": true,
            "UnsignedInteger": "42",
            "Fractional": "24",
            "Scale": 10,
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestFixedPointExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("positive", func(t *testing.T) {

		t.Parallel()

		expr := &FixedPointExpression{
			PositiveLiteral: "1_2.3_4",
			UnsignedInteger: big.NewInt(42),
			Scale:           2,
		}

		assert.Equal(t,
			prettier.Text(`1_2.3_4`),
			expr.Doc(),
		)
	})

	t.Run("negative", func(t *testing.T) {

		t.Parallel()

		expr := &FixedPointExpression{
			PositiveLiteral: "1_2.3_4",
			Negative:        true,
			UnsignedInteger: big.NewInt(42),
			Scale:           2,
		}

		assert.Equal(t,
			prettier.Text(`-1_2.3_4`),
			expr.Doc(),
		)
	})
}

func TestFixedPointExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("positive", func(t *testing.T) {

		t.Parallel()

		expr := &FixedPointExpression{
			PositiveLiteral: "1_2.3_4",
			UnsignedInteger: big.NewInt(42),
			Scale:           2,
		}

		assert.Equal(t,
			"1_2.3_4",
			expr.String(),
		)
	})

	t.Run("negative", func(t *testing.T) {

		t.Parallel()

		expr := &FixedPointExpression{
			PositiveLiteral: "1_2.3_4",
			Negative:        true,
			UnsignedInteger: big.NewInt(42),
			Scale:           2,
		}

		assert.Equal(t,
			"-1_2.3_4",
			expr.String(),
		)
	})
}

func TestArrayExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &ArrayExpression{
		Values: []Expression{
			&BoolExpression{
				Value: true,
				Range: Range{
					StartPos: Position{Offset: 1, Line: 2, Column: 3},
					EndPos:   Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			&NilExpression{
				Pos: Position{Offset: 7, Line: 8, Column: 9},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 10, Line: 11, Column: 12},
			EndPos:   Position{Offset: 13, Line: 14, Column: 15},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ArrayExpression",
            "Values": [
                {
                    "Type": "BoolExpression",
                    "Value": true,
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
                },
                {
                    "Type": "NilExpression",
                    "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                    "EndPos": {"Offset": 9, "Line": 8, "Column": 11}
                }
            ],
            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
            "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
        }
        `,
		string(actual),
	)
}

func TestArrayExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		assert.Equal(t,
			prettier.Text("[]"),
			(&ArrayExpression{}).Doc(),
		)
	})

	t.Run("non-empty", func(t *testing.T) {

		t.Parallel()

		expr := &ArrayExpression{
			Values: []Expression{
				&NilExpression{},
				&BoolExpression{Value: true},
				&StringExpression{Value: "test"},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("["),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Concat{
								prettier.Text("nil"),
								prettier.Concat{
									prettier.Text(","),
									prettier.Line{},
								},
								prettier.Text("true"),
								prettier.Concat{
									prettier.Text(","),
									prettier.Line{},
								},
								prettier.Text(`"test"`),
							},
						},
					},
					prettier.SoftLine{},
					prettier.Text("]"),
				},
			},
			expr.Doc(),
		)
	})
}

func TestArrayExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		assert.Equal(t,
			"[]",
			(&ArrayExpression{}).String(),
		)
	})

	t.Run("non-empty", func(t *testing.T) {

		t.Parallel()

		expr := &ArrayExpression{
			Values: []Expression{
				&NilExpression{},
				&BoolExpression{Value: true},
				&StringExpression{Value: "test"},
			},
		}

		assert.Equal(t,
			`[nil, true, "test"]`,
			expr.String(),
		)
	})
}

func TestDictionaryExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &DictionaryExpression{
		Entries: []DictionaryEntry{
			{
				Key: &BoolExpression{
					Value: true,
					Range: Range{
						StartPos: Position{Offset: 1, Line: 2, Column: 3},
						EndPos:   Position{Offset: 4, Line: 5, Column: 6},
					},
				},
				Value: &NilExpression{
					Pos: Position{Offset: 7, Line: 8, Column: 9},
				},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 10, Line: 11, Column: 12},
			EndPos:   Position{Offset: 13, Line: 14, Column: 15},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "DictionaryExpression",
            "Entries": [
                {
                    "Type": "DictionaryEntry",
                    "Key": {
                        "Type": "BoolExpression",
                        "Value": true,
                        "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                        "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
                    },
                    "Value": {
                        "Type": "NilExpression",
                        "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                        "EndPos": {"Offset": 9, "Line": 8, "Column": 11}
                    }
                }
            ],
            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
            "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
        }
        `,
		string(actual),
	)
}

func TestDictionaryExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		assert.Equal(t,
			prettier.Text("{}"),
			(&DictionaryExpression{}).Doc(),
		)
	})

	t.Run("non-empty", func(t *testing.T) {

		t.Parallel()

		expr := &DictionaryExpression{
			Entries: []DictionaryEntry{
				{
					Key:   &StringExpression{Value: "foo"},
					Value: &NilExpression{},
				},
				{
					Key:   &StringExpression{Value: "bar"},
					Value: &BoolExpression{Value: true},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Concat{
								prettier.Group{
									Doc: prettier.Concat{
										prettier.Text(`"foo"`),
										prettier.Concat{
											prettier.Text(":"),
											prettier.Line{},
										},
										prettier.Text("nil"),
									},
								},
								prettier.Concat{
									prettier.Text(","),
									prettier.Line{},
								},
								prettier.Group{
									Doc: prettier.Concat{
										prettier.Text(`"bar"`),
										prettier.Concat{
											prettier.Text(":"),
											prettier.Line{},
										},
										prettier.Text("true"),
									},
								},
							},
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			expr.Doc(),
		)
	})

}

func TestDictionaryExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()

		assert.Equal(t,
			"{}",
			(&DictionaryExpression{}).String(),
		)
	})

	t.Run("non-empty", func(t *testing.T) {

		t.Parallel()

		expr := &DictionaryExpression{
			Entries: []DictionaryEntry{
				{
					Key:   &StringExpression{Value: "foo"},
					Value: &NilExpression{},
				},
				{
					Key:   &StringExpression{Value: "bar"},
					Value: &BoolExpression{Value: true},
				},
			},
		}

		assert.Equal(t,
			`{"foo": nil, "bar": true}`,
			expr.String(),
		)
	})

}

func TestIdentifierExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &IdentifierExpression{
		Identifier: Identifier{
			Identifier: "foobar",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "IdentifierExpression",
            "Identifier": {
                "Identifier": "foobar",
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
        }
        `,
		string(actual),
	)
}

func TestIdentifierExpression_Doc(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		prettier.Text("test"),
		(&IdentifierExpression{
			Identifier: Identifier{
				Identifier: "test",
			},
		}).Doc(),
	)
}

func TestIdentifierExpression_String(t *testing.T) {

	t.Parallel()

	assert.Equal(t,
		"test",
		(&IdentifierExpression{
			Identifier: Identifier{
				Identifier: "test",
			},
		}).String(),
	)
}

func TestPathExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &PathExpression{
		StartPos: Position{Offset: 1, Line: 2, Column: 3},
		Domain: Identifier{
			Identifier: "storage",
			Pos:        Position{Offset: 4, Line: 5, Column: 6},
		},
		Identifier: Identifier{
			Identifier: "foobar",
			Pos:        Position{Offset: 7, Line: 8, Column: 9},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "PathExpression",
            "Domain": {
                "Identifier": "storage",
                "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                "EndPos": {"Offset": 10, "Line": 5, "Column": 12}
            },
            "Identifier": {
                "Identifier": "foobar",
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 12, "Line": 8, "Column": 14}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 12, "Line": 8, "Column": 14}
        }
        `,
		string(actual),
	)
}

func TestPathExpression_Doc(t *testing.T) {

	t.Parallel()

	expr := &PathExpression{
		Domain: Identifier{
			Identifier: "storage",
		},
		Identifier: Identifier{
			Identifier: "test",
		},
	}

	assert.Equal(t,
		prettier.Concat{
			prettier.Text("/"),
			prettier.Text("storage"),
			prettier.Text("/"),
			prettier.Text("test"),
		},
		expr.Doc(),
	)
}

func TestPathExpression_String(t *testing.T) {

	t.Parallel()

	expr := &PathExpression{
		Domain: Identifier{
			Identifier: "storage",
		},
		Identifier: Identifier{
			Identifier: "test",
		},
	}

	assert.Equal(t,
		"/storage/test",
		expr.String(),
	)
}

func TestMemberExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &MemberExpression{
		Expression: &BoolExpression{
			Value: true,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Optional:  true,
		AccessPos: Position{Offset: 7, Line: 8, Column: 9},
		Identifier: Identifier{
			Identifier: "foobar",
			Pos:        Position{Offset: 10, Line: 11, Column: 12},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "MemberExpression",
            "Expression": {
                "Type": "BoolExpression",
                "Value": true,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "Optional": true,
            "AccessPos": {"Offset": 7, "Line": 8, "Column": 9},
            "Identifier": {
                "Identifier": "foobar",
                "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                "EndPos": {"Offset": 15, "Line": 11, "Column": 17}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 15, "Line": 11, "Column": 17}
        }
        `,
		string(actual),
	)
}

func TestMemberExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("non-optional", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			Identifier: Identifier{
				Identifier: "bar",
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("foo"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("."),
							prettier.Text("bar"),
						},
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("optional", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			Optional: true,
			Identifier: Identifier{
				Identifier: "bar",
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("foo"),
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("?."),
							prettier.Text("bar"),
						},
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &MemberExpression{
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Identifier: Identifier{
					Identifier: "bar",
				},
			},
			Identifier: Identifier{
				Identifier: "baz",
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Concat{
					prettier.Text("foo"),
					prettier.Group{
						Doc: prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Text("."),
								prettier.Text("bar"),
							},
						},
					},
				},
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("."),
							prettier.Text("baz"),
						},
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Identifier: Identifier{
				Identifier: "baz",
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Group{
									Doc: prettier.Concat{
										prettier.Group{
											Doc: prettier.Text("foo"),
										},
										prettier.Line{},
										prettier.Text("-"),
										prettier.Text(" "),
										prettier.Group{
											Doc: prettier.Text("bar"),
										},
									},
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
				prettier.Group{
					Doc: prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("."),
							prettier.Text("baz"),
						},
					},
				},
			},
			expr.Doc(),
		)
	})
}

func TestMemberExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("non-optional", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			Identifier: Identifier{
				Identifier: "bar",
			},
		}

		assert.Equal(t,
			"foo.bar",
			expr.String(),
		)
	})

	t.Run("optional", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			Optional: true,
			Identifier: Identifier{
				Identifier: "bar",
			},
		}

		assert.Equal(t,
			"foo?.bar",
			expr.String(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &MemberExpression{
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Identifier: Identifier{
					Identifier: "bar",
				},
			},
			Identifier: Identifier{
				Identifier: "baz",
			},
		}

		assert.Equal(t,
			"foo.bar.baz",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &MemberExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Identifier: Identifier{
				Identifier: "baz",
			},
		}

		assert.Equal(t,
			"(foo - bar).baz",
			expr.String(),
		)
	})
}

func TestIndexExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &IndexExpression{
		TargetExpression: &BoolExpression{
			Value: true,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		IndexingExpression: &NilExpression{
			Pos: Position{Offset: 7, Line: 8, Column: 9},
		},
		Range: Range{
			StartPos: Position{Offset: 10, Line: 11, Column: 12},
			EndPos:   Position{Offset: 13, Line: 14, Column: 15},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "IndexExpression",
            "TargetExpression": {
                "Type": "BoolExpression",
                "Value": true,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "IndexingExpression": {
                "Type": "NilExpression",
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 9, "Line": 8, "Column": 11}
            },
            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
            "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
        }
        `,
		string(actual),
	)
}

func TestIndexExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &IndexExpression{
			TargetExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			IndexingExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "bar",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("foo"),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("["),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Text("bar"),
							},
						},
						prettier.SoftLine{},
						prettier.Text("]"),
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &IndexExpression{
			TargetExpression: &IndexExpression{
				TargetExpression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				IndexingExpression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			IndexingExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "baz",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Concat{
					prettier.Text("foo"),
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("["),
							prettier.Indent{
								Doc: prettier.Concat{
									prettier.SoftLine{},
									prettier.Text("bar"),
								},
							},
							prettier.SoftLine{},
							prettier.Text("]"),
						},
					},
				},
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("["),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Text("baz"),
							},
						},
						prettier.SoftLine{},
						prettier.Text("]"),
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &IndexExpression{
			TargetExpression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			IndexingExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "baz",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Group{
									Doc: prettier.Concat{
										prettier.Group{
											Doc: prettier.Text("foo"),
										},
										prettier.Line{},
										prettier.Text("-"),
										prettier.Text(" "),
										prettier.Group{
											Doc: prettier.Text("bar"),
										},
									},
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("["),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Text("baz"),
							},
						},
						prettier.SoftLine{},
						prettier.Text("]"),
					},
				},
			},
			expr.Doc(),
		)
	})

}

func TestIndexExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &IndexExpression{
			TargetExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
			IndexingExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "bar",
				},
			},
		}

		assert.Equal(t,
			"foo[bar]",
			expr.String(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &IndexExpression{
			TargetExpression: &IndexExpression{
				TargetExpression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				IndexingExpression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			IndexingExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "baz",
				},
			},
		}

		assert.Equal(t,
			"foo[bar][baz]",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &IndexExpression{
			TargetExpression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			IndexingExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "baz",
				},
			},
		}

		assert.Equal(t,
			"(foo - bar)[baz]",
			expr.String(),
		)
	})
}

func TestUnaryExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &UnaryExpression{
		Operation: OperationNegate,
		Expression: &IntegerExpression{
			PositiveLiteral: "42",
			Value:           big.NewInt(42),
			Base:            10,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		StartPos: Position{Offset: 7, Line: 8, Column: 9},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "UnaryExpression",
            "Operation": "OperationNegate",
            "Expression": {
                "Type": "IntegerExpression",
                "PositiveLiteral": "42",
                "Value": "42",
                "Base": 10,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestUnaryExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &UnaryExpression{
			Operation: OperationMinus,
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("-"),
				prettier.Text("foo"),
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &UnaryExpression{
			Operation: OperationMinus,
			Expression: &UnaryExpression{
				Operation: OperationMinus,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("-"),
				prettier.Concat{
					prettier.Text("-"),
					prettier.Text("foo"),
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &UnaryExpression{
			Operation: OperationMinus,
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("-"),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Group{
									Doc: prettier.Concat{
										prettier.Group{
											Doc: prettier.Text("foo"),
										},
										prettier.Line{},
										prettier.Text("-"),
										prettier.Text(" "),
										prettier.Group{
											Doc: prettier.Text("bar"),
										},
									},
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
			},
			expr.Doc(),
		)
	})
}

func TestUnaryExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &UnaryExpression{
			Operation: OperationMinus,
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		}

		assert.Equal(t,
			"-foo",
			expr.String(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &UnaryExpression{
			Operation: OperationMinus,
			Expression: &UnaryExpression{
				Operation: OperationMinus,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			"--foo",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &UnaryExpression{
			Operation: OperationMinus,
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
		}

		assert.Equal(t,
			"-(foo - bar)",
			expr.String(),
		)
	})
}

func TestBinaryExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &BinaryExpression{
		Operation: OperationPlus,
		Left: &IntegerExpression{
			PositiveLiteral: "42",
			Value:           big.NewInt(42),
			Base:            10,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Right: &IntegerExpression{
			PositiveLiteral: "99",
			Value:           big.NewInt(99),
			Base:            10,
			Range: Range{
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
				EndPos:   Position{Offset: 10, Line: 11, Column: 12},
			},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "BinaryExpression",
            "Operation": "OperationPlus",
            "Left": {
                "Type": "IntegerExpression",
                "PositiveLiteral": "42",
                "Value": "42",
                "Base": 10,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "Right": {
                "Type": "IntegerExpression",
                "PositiveLiteral": "99",
                "Value": "99",
                "Base": 10,
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestBinaryExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationPlus,
			Left: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Text("42"),
					},
					prettier.Line{},
					prettier.Text("+"),
					prettier.Space,
					prettier.Group{
						Doc: prettier.Text("99"),
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence, left associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationPlus,
			Left: &BinaryExpression{
				Operation: OperationPlus,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Group{
									Doc: prettier.Text("42"),
								},
								prettier.Line{},
								prettier.Text("+"),
								prettier.Text(" "),
								prettier.Group{
									Doc: prettier.Text("1"),
								},
							},
						},
					},
					prettier.Line{},
					prettier.Text("+"),
					prettier.Text(" "),
					prettier.Group{
						Doc: prettier.Text("99"),
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence, left associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationPlus,
			Left: &BinaryExpression{
				Operation: OperationBitwiseOr,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("("),
								prettier.Indent{
									Doc: prettier.Concat{
										prettier.SoftLine{},
										prettier.Group{
											Doc: prettier.Concat{
												prettier.Group{
													Doc: prettier.Text("42"),
												},
												prettier.Line{},
												prettier.Text("|"),
												prettier.Text(" "),
												prettier.Group{
													Doc: prettier.Text("1"),
												},
											},
										},
									},
								},
								prettier.SoftLine{},
								prettier.Text(")"),
							},
						},
					},
					prettier.Line{},
					prettier.Text("+"),
					prettier.Text(" "),
					prettier.Group{
						Doc: prettier.Text("99"),
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence, right associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationNilCoalesce,
			Left: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
			Right: &BinaryExpression{
				Operation: OperationNilCoalesce,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Text("99"),
					},
					prettier.Line{},
					prettier.Text("??"),
					prettier.Text(" "),
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Group{
									Doc: prettier.Text("42"),
								},
								prettier.Line{},
								prettier.Text("??"),
								prettier.Text(" "),
								prettier.Group{
									Doc: prettier.Text("1"),
								},
							},
						},
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence, right associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationNilCoalesce,
			Left: &BinaryExpression{
				Operation: OperationOr,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("("),
								prettier.Indent{
									Doc: prettier.Concat{
										prettier.SoftLine{},
										prettier.Group{
											Doc: prettier.Concat{
												prettier.Group{
													Doc: prettier.Text("42"),
												},
												prettier.Line{},
												prettier.Text("||"),
												prettier.Text(" "),
												prettier.Group{
													Doc: prettier.Text("1"),
												},
											},
										},
									},
								},
								prettier.SoftLine{},
								prettier.Text(")"),
							},
						},
					},
					prettier.Line{},
					prettier.Text("??"),
					prettier.Text(" "),
					prettier.Group{
						Doc: prettier.Text("99"),
					},
				},
			},
			expr.Doc(),
		)
	})

}

func TestBinaryExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationPlus,
			Left: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			"42 + 99",
			expr.String(),
		)
	})

	t.Run("nested, same precedence, left associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationPlus,
			Left: &BinaryExpression{
				Operation: OperationPlus,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			"42 + 1 + 99",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence, left associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationPlus,
			Left: &BinaryExpression{
				Operation: OperationBitwiseOr,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			"(42 | 1) + 99",
			expr.String(),
		)
	})

	t.Run("nested, same precedence, right associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationNilCoalesce,
			Left: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
			Right: &BinaryExpression{
				Operation: OperationNilCoalesce,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
		}

		assert.Equal(t,
			"99 ?? 42 ?? 1",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence, right associative", func(t *testing.T) {

		t.Parallel()

		expr := &BinaryExpression{
			Operation: OperationNilCoalesce,
			Left: &BinaryExpression{
				Operation: OperationOr,
				Left: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Right: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(42),
					Base:            10,
				},
			},
			Right: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			"(42 || 1) ?? 99",
			expr.String(),
		)
	})

}

func TestDestroyExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &DestroyExpression{
		Expression: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		StartPos: Position{Offset: 4, Line: 5, Column: 6},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "DestroyExpression",
            "Expression": {
                "Type": "IdentifierExpression",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
            "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
        }
        `,
		string(actual),
	)
}

func TestDestroyExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &DestroyExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("destroy "),
				prettier.Text("foo"),
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &DestroyExpression{
			Expression: &UnaryExpression{
				Operation: OperationMinus,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("destroy "),
				prettier.Concat{
					prettier.Text("-"),
					prettier.Text("foo"),
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &DestroyExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("destroy "),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Group{
									Doc: prettier.Concat{
										prettier.Group{
											Doc: prettier.Text("foo"),
										},
										prettier.Line{},
										prettier.Text("-"),
										prettier.Text(" "),
										prettier.Group{
											Doc: prettier.Text("bar"),
										},
									},
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
			},
			expr.Doc(),
		)
	})
}

func TestDestroyExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &DestroyExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		}

		assert.Equal(t,
			"destroy foo",
			expr.String(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &DestroyExpression{
			Expression: &UnaryExpression{
				Operation: OperationMinus,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			"destroy -foo",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &DestroyExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
		}

		assert.Equal(t,
			"destroy (foo - bar)",
			expr.String(),
		)
	})
}

func TestForceExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &ForceExpression{
		Expression: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		EndPos: Position{Offset: 4, Line: 5, Column: 6},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ForceExpression",
            "Expression": {
                "Type": "IdentifierExpression",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestForceExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &ForceExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("foo"),
				prettier.Text("!"),
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ForceExpression{
			Expression: &ForceExpression{
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Concat{
					prettier.Text("foo"),
					prettier.Text("!"),
				},
				prettier.Text("!"),
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ForceExpression{
			Expression: &UnaryExpression{
				Operation: OperationMinus,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Concat{
									prettier.Text("-"),
									prettier.Text("foo"),
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
				prettier.Text("!"),
			},
			expr.Doc(),
		)
	})
}

func TestForceExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &ForceExpression{
			Expression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		}

		assert.Equal(t,
			"foo!",
			expr.String(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ForceExpression{
			Expression: &ForceExpression{
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			"foo!!",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ForceExpression{
			Expression: &UnaryExpression{
				Operation: OperationMinus,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
			},
		}

		assert.Equal(t,
			"(-foo)!",
			expr.String(),
		)
	})
}

func TestConditionalExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &ConditionalExpression{
		Test: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Then: &IntegerExpression{
			PositiveLiteral: "42",
			Value:           big.NewInt(42),
			Base:            10,
			Range: Range{
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
				EndPos:   Position{Offset: 10, Line: 11, Column: 12},
			},
		},
		Else: &IntegerExpression{
			PositiveLiteral: "99",
			Value:           big.NewInt(99),
			Base:            10,
			Range: Range{
				StartPos: Position{Offset: 13, Line: 14, Column: 15},
				EndPos:   Position{Offset: 16, Line: 17, Column: 18},
			},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ConditionalExpression",
            "Test": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "Then": {
                "Type": "IntegerExpression",
                "PositiveLiteral": "42",
                "Value": "42",
                "Base": 10,
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
            },
            "Else": {
                "Type": "IntegerExpression",
                "PositiveLiteral": "99",
                "Value": "99",
                "Base": 10,
                "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
                "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
        }
        `,
		string(actual),
	)
}

func TestConditionalExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Else: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text(`false`),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.Concat{
								prettier.Line{},
								prettier.Text("? "),
							},
							prettier.Indent{
								Doc: prettier.Text(`42`),
							},
							prettier.Concat{
								prettier.Line{},
								prettier.Text(": "),
							},
							prettier.Indent{
								Doc: prettier.Text(`99`),
							},
						},
					},
				},
			},
			expr.Doc(),
		)

	})

	t.Run("nested test, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &ConditionalExpression{
				Test: &BoolExpression{
					Value: false,
				},
				Then: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(1),
					Base:            10,
				},
				Else: &IntegerExpression{
					PositiveLiteral: "2",
					Value:           big.NewInt(2),
					Base:            10,
				},
			},
			Then: &IntegerExpression{
				PositiveLiteral: "3",
				Value:           big.NewInt(3),
				Base:            10,
			},
			Else: &IntegerExpression{
				PositiveLiteral: "4",
				Value:           big.NewInt(4),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("("),
							prettier.Indent{
								Doc: prettier.Concat{
									prettier.SoftLine{},
									prettier.Group{
										Doc: prettier.Concat{
											prettier.Text("false"),
											prettier.Indent{
												Doc: prettier.Concat{
													prettier.Concat{
														prettier.Line{},
														prettier.Text("? "),
													},
													prettier.Indent{
														Doc: prettier.Text("1"),
													},
													prettier.Concat{
														prettier.Line{},
														prettier.Text(": "),
													},
													prettier.Indent{
														Doc: prettier.Text("2"),
													},
												},
											},
										},
									},
								},
							},
							prettier.SoftLine{},
							prettier.Text(")"),
						},
					},
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.Concat{
								prettier.Line{},
								prettier.Text("? "),
							},
							prettier.Indent{
								Doc: prettier.Text("3"),
							},
							prettier.Concat{
								prettier.Line{},
								prettier.Text(": "),
							},
							prettier.Indent{
								Doc: prettier.Text("4"),
							},
						},
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested then, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &ConditionalExpression{
				Test: &BoolExpression{
					Value: false,
				},
				Then: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(1),
					Base:            10,
				},
				Else: &IntegerExpression{
					PositiveLiteral: "2",
					Value:           big.NewInt(2),
					Base:            10,
				},
			},
			Else: &IntegerExpression{
				PositiveLiteral: "3",
				Value:           big.NewInt(3),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("false"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.Concat{
								prettier.Line{},
								prettier.Text("? "),
							},
							prettier.Indent{
								Doc: prettier.Group{
									Doc: prettier.Concat{
										prettier.Text("("),
										prettier.Indent{
											Doc: prettier.Concat{
												prettier.SoftLine{},
												prettier.Group{
													Doc: prettier.Concat{
														prettier.Text("false"),
														prettier.Indent{
															Doc: prettier.Concat{
																prettier.Concat{
																	prettier.Line{},
																	prettier.Text("? "),
																},
																prettier.Indent{
																	Doc: prettier.Text("1"),
																},
																prettier.Concat{
																	prettier.Line{},
																	prettier.Text(": "),
																},
																prettier.Indent{
																	Doc: prettier.Text("2"),
																},
															},
														},
													},
												},
											},
										},
										prettier.SoftLine{},
										prettier.Text(")"),
									},
								},
							},
							prettier.Concat{
								prettier.Line{},
								prettier.Text(": "),
							},
							prettier.Indent{
								Doc: prettier.Text("3"),
							},
						},
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested else, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &IntegerExpression{
				PositiveLiteral: "1",
				Value:           big.NewInt(1),
				Base:            10,
			},
			Else: &ConditionalExpression{
				Test: &BoolExpression{
					Value: false,
				},
				Then: &IntegerExpression{
					PositiveLiteral: "2",
					Value:           big.NewInt(2),
					Base:            10,
				},
				Else: &IntegerExpression{
					PositiveLiteral: "3",
					Value:           big.NewInt(3),
					Base:            10,
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("false"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.Concat{
								prettier.Line{},
								prettier.Text("? "),
							},
							prettier.Indent{
								Doc: prettier.Text("1"),
							},
							prettier.Concat{
								prettier.Line{},
								prettier.Text(": "),
							},
							prettier.Indent{
								Doc: prettier.Group{
									Doc: prettier.Concat{
										prettier.Text("false"),
										prettier.Indent{
											Doc: prettier.Concat{
												prettier.Concat{
													prettier.Line{},
													prettier.Text("? "),
												},
												prettier.Indent{
													Doc: prettier.Text("2"),
												},
												prettier.Concat{
													prettier.Line{},
													prettier.Text(": "),
												},
												prettier.Indent{
													Doc: prettier.Text("3"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expr.Doc(),
		)
	})

}

func TestConditionalExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Else: &IntegerExpression{
				PositiveLiteral: "99",
				Value:           big.NewInt(99),
				Base:            10,
			},
		}

		assert.Equal(t,
			"false ? 42 : 99",
			expr.String(),
		)
	})

	t.Run("nested test, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &ConditionalExpression{
				Test: &BoolExpression{
					Value: false,
				},
				Then: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(1),
					Base:            10,
				},
				Else: &IntegerExpression{
					PositiveLiteral: "2",
					Value:           big.NewInt(2),
					Base:            10,
				},
			},
			Then: &IntegerExpression{
				PositiveLiteral: "3",
				Value:           big.NewInt(3),
				Base:            10,
			},
			Else: &IntegerExpression{
				PositiveLiteral: "4",
				Value:           big.NewInt(4),
				Base:            10,
			},
		}

		assert.Equal(t,
			"(false ? 1 : 2) ? 3 : 4",
			expr.String(),
		)
	})

	t.Run("nested then, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &ConditionalExpression{
				Test: &BoolExpression{
					Value: false,
				},
				Then: &IntegerExpression{
					PositiveLiteral: "1",
					Value:           big.NewInt(1),
					Base:            10,
				},
				Else: &IntegerExpression{
					PositiveLiteral: "2",
					Value:           big.NewInt(2),
					Base:            10,
				},
			},
			Else: &IntegerExpression{
				PositiveLiteral: "3",
				Value:           big.NewInt(3),
				Base:            10,
			},
		}

		assert.Equal(t,
			"false ? (false ? 1 : 2) : 3",
			expr.String(),
		)
	})

	t.Run("nested else, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ConditionalExpression{
			Test: &BoolExpression{
				Value: false,
			},
			Then: &IntegerExpression{
				PositiveLiteral: "1",
				Value:           big.NewInt(1),
				Base:            10,
			},
			Else: &ConditionalExpression{
				Test: &BoolExpression{
					Value: false,
				},
				Then: &IntegerExpression{
					PositiveLiteral: "2",
					Value:           big.NewInt(2),
					Base:            10,
				},
				Else: &IntegerExpression{
					PositiveLiteral: "3",
					Value:           big.NewInt(3),
					Base:            10,
				},
			},
		}

		assert.Equal(t,
			"false ? 1 : false ? 2 : 3",
			expr.String(),
		)
	})

}

func TestInvocationExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &InvocationExpression{
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
				TrailingSeparatorPos: Position{Offset: 25, Line: 26, Column: 27},
			},
		},
		ArgumentsStartPos: Position{Offset: 28, Line: 29, Column: 30},
		EndPos:            Position{Offset: 22, Line: 23, Column: 24},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
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
                    "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                    "EndPos": {"Offset": 19, "Line": 20, "Column": 21},
                    "TrailingSeparatorPos": {"Offset": 25, "Line": 26, "Column": 27}
                }
            ],
            "ArgumentsStartPos": {"Offset": 28, "Line": 29, "Column": 30},
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 22, "Line": 23, "Column": 24}
        }
        `,
		string(actual),
	)
}

func TestInvocationExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("without type arguments and arguments", func(t *testing.T) {

		t.Parallel()

		expr := &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foobar",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("foobar"),
				prettier.Text("()"),
			},
			expr.Doc(),
		)
	})

	t.Run("target expression with lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &InvocationExpression{
			InvokedExpression: &CastingExpression{
				Operation: OperationCast,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "T",
						},
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Group{
									Doc: prettier.Concat{
										prettier.Group{
											Doc: prettier.Text("foo"),
										},
										prettier.Line{},
										prettier.Text("as"),
										prettier.Line{},
										prettier.Text("T"),
									},
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
				prettier.Text("()"),
			},
			expr.Doc(),
		)
	})

	t.Run("with type argument and argument", func(t *testing.T) {

		t.Parallel()

		expr := &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foobar",
				},
			},
			TypeArguments: []*TypeAnnotation{
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AB",
						},
					},
				},
			},
			Arguments: []*Argument{
				{
					Label: "ok",
					Expression: &BoolExpression{
						Value: false,
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("foobar"),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("<"),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Concat{
									prettier.Text("@"),
									prettier.Text("AB"),
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(">"),
					},
				},
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Concat{
									prettier.Text("ok: "),
									prettier.Text("false"),
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
			},
			expr.Doc(),
		)
	})
}

func TestInvocationExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("without type arguments and arguments", func(t *testing.T) {

		t.Parallel()

		expr := &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foobar",
				},
			},
		}

		assert.Equal(t,
			"foobar()",
			expr.String(),
		)
	})

	t.Run("target expression with lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &InvocationExpression{
			InvokedExpression: &CastingExpression{
				Operation: OperationCast,
				Expression: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "T",
						},
					},
				},
			},
		}

		assert.Equal(t,
			"(foo as T)()",
			expr.String(),
		)
	})

	t.Run("with type argument and argument", func(t *testing.T) {

		t.Parallel()

		expr := &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foobar",
				},
			},
			TypeArguments: []*TypeAnnotation{
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AB",
						},
					},
				},
			},
			Arguments: []*Argument{
				{
					Label: "ok",
					Expression: &BoolExpression{
						Value: false,
					},
				},
			},
		}

		assert.Equal(t,
			"foobar<@AB>(ok: false)",
			expr.String(),
		)
	})
}

func TestCastingExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &CastingExpression{
		Expression: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Operation: OperationForceCast,
		TypeAnnotation: &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
					Pos:        Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "CastingExpression",
            "Expression": {
               "Type": "IdentifierExpression",
               "Identifier": {
                   "Identifier": "foobar",
                   "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                   "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
               },
               "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
               "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "Operation": "OperationForceCast",
            "TypeAnnotation": {
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
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
        }
        `,
		string(actual),
	)
}

func TestCastingExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &CastingExpression{
			Expression: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Operation: OperationFailableCast,
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Text("42"),
					},
					prettier.Line{},
					prettier.Text("as?"),
					prettier.Line{},
					prettier.Text("Int"),
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &CastingExpression{
			Expression: &CastingExpression{
				Expression: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Operation: OperationFailableCast,
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AnyStruct",
						},
					},
				},
			},
			Operation: OperationFailableCast,
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Group{
									Doc: prettier.Text("42"),
								},
								prettier.Line{},
								prettier.Text("as?"),
								prettier.Line{},
								prettier.Text("AnyStruct"),
							},
						},
					},
					prettier.Line{},
					prettier.Text("as?"),
					prettier.Line{},
					prettier.Text("Int"),
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &CastingExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Operation: OperationFailableCast,
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("("),
								prettier.Indent{
									Doc: prettier.Concat{
										prettier.SoftLine{},
										prettier.Group{
											Doc: prettier.Concat{
												prettier.Group{
													Doc: prettier.Text("foo"),
												},
												prettier.Line{},
												prettier.Text("-"),
												prettier.Text(" "),
												prettier.Group{
													Doc: prettier.Text("bar"),
												},
											},
										},
									}},
								prettier.SoftLine{},
								prettier.Text(")"),
							},
						},
					},
					prettier.Line{},
					prettier.Text("as?"),
					prettier.Line{},
					prettier.Text("Int"),
				},
			},
			expr.Doc(),
		)
	})
}

func TestCastingExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &CastingExpression{
			Expression: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Operation: OperationFailableCast,
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			"42 as? Int",
			expr.String(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &CastingExpression{
			Expression: &CastingExpression{
				Expression: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Operation: OperationFailableCast,
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AnyStruct",
						},
					},
				},
			},
			Operation: OperationFailableCast,
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			"42 as? AnyStruct as? Int",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &CastingExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Operation: OperationFailableCast,
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			"(foo - bar) as? Int",
			expr.String(),
		)
	})
}

func TestCreateExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &CreateExpression{
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

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "CreateExpression",
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
                        "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                        "EndPos": {"Offset": 19, "Line": 20, "Column": 21},
                        "TrailingSeparatorPos": {"Offset": 28, "Line": 29, "Column": 30}
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

func TestCreateExpression_Doc(t *testing.T) {

	t.Parallel()

	expr := &CreateExpression{
		InvocationExpression: &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		},
	}

	assert.Equal(t,
		prettier.Concat{
			prettier.Text("create "),
			prettier.Concat{
				prettier.Text("foo"),
				prettier.Text("()"),
			},
		},
		expr.Doc(),
	)
}

func TestCreateExpression_String(t *testing.T) {

	t.Parallel()

	expr := &CreateExpression{
		InvocationExpression: &InvocationExpression{
			InvokedExpression: &IdentifierExpression{
				Identifier: Identifier{
					Identifier: "foo",
				},
			},
		},
	}

	assert.Equal(t,
		"create foo()",
		expr.String(),
	)
}

func TestReferenceExpression_MarshalJSON(t *testing.T) {

	expr := &ReferenceExpression{
		Expression: &IdentifierExpression{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Type: &NominalType{
			Identifier: Identifier{
				Identifier: "AB",
				Pos:        Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		StartPos: Position{Offset: 7, Line: 8, Column: 9},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "ReferenceExpression",
            "Expression": {
               "Type": "IdentifierExpression",
               "Identifier": {
                   "Identifier": "foobar",
                   "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                   "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
               },
               "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
               "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "TargetType": {
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
        `,
		string(actual),
	)
}

func TestReferenceExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &ReferenceExpression{
			Expression: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Type: &ReferenceType{
				Authorized: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("&"),
					prettier.Group{
						Doc: prettier.Text("42"),
					},
					prettier.Line{},
					prettier.Text("as"),
					prettier.Line{},
					prettier.Concat{
						prettier.Text("auth "),
						prettier.Text("&"),
						prettier.Text("Int"),
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ReferenceExpression{
			Expression: &ReferenceExpression{
				Expression: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Type: &ReferenceType{
					Authorized: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AnyStruct",
						},
					},
				},
			},
			Type: &ReferenceType{
				Authorized: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "XYZ",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("&"),
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("&"),
								prettier.Group{
									Doc: prettier.Text("42"),
								},
								prettier.Line{},
								prettier.Text("as"),
								prettier.Line{},
								prettier.Concat{
									prettier.Text("auth "),
									prettier.Text("&"),
									prettier.Text("AnyStruct"),
								},
							},
						},
					},
					prettier.Line{},
					prettier.Text("as"),
					prettier.Line{},
					prettier.Concat{
						prettier.Text("auth "),
						prettier.Text("&"),
						prettier.Text("XYZ"),
					},
				},
			},
			expr.Doc(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ReferenceExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Type: &ReferenceType{
				Authorized: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("&"),
					prettier.Group{
						Doc: prettier.Group{
							Doc: prettier.Concat{
								prettier.Text("("),
								prettier.Indent{
									Doc: prettier.Concat{
										prettier.SoftLine{},
										prettier.Group{
											Doc: prettier.Concat{
												prettier.Group{
													Doc: prettier.Text("foo"),
												},
												prettier.Line{},
												prettier.Text("-"),
												prettier.Text(" "),
												prettier.Group{
													Doc: prettier.Text("bar"),
												},
											},
										},
									}},
								prettier.SoftLine{},
								prettier.Text(")"),
							},
						},
					},
					prettier.Line{},
					prettier.Text("as"),
					prettier.Line{},
					prettier.Concat{
						prettier.Text("auth "),
						prettier.Text("&"),
						prettier.Text("Int"),
					},
				},
			},
			expr.Doc(),
		)
	})
}

func TestReferenceExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		expr := &ReferenceExpression{
			Expression: &IntegerExpression{
				PositiveLiteral: "42",
				Value:           big.NewInt(42),
				Base:            10,
			},
			Type: &ReferenceType{
				Authorized: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			"&42 as auth &Int",
			expr.String(),
		)
	})

	t.Run("nested, same precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ReferenceExpression{
			Expression: &ReferenceExpression{
				Expression: &IntegerExpression{
					PositiveLiteral: "42",
					Value:           big.NewInt(42),
					Base:            10,
				},
				Type: &ReferenceType{
					Authorized: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AnyStruct",
						},
					},
				},
			},
			Type: &ReferenceType{
				Authorized: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "XYZ",
					},
				},
			},
		}

		assert.Equal(t,
			"&&42 as auth &AnyStruct as auth &XYZ",
			expr.String(),
		)
	})

	t.Run("nested, lower precedence", func(t *testing.T) {

		t.Parallel()

		expr := &ReferenceExpression{
			Expression: &BinaryExpression{
				Operation: OperationMinus,
				Left: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "foo",
					},
				},
				Right: &IdentifierExpression{
					Identifier: Identifier{
						Identifier: "bar",
					},
				},
			},
			Type: &ReferenceType{
				Authorized: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Int",
					},
				},
			},
		}

		assert.Equal(t,
			"&(foo - bar) as auth &Int",
			expr.String(),
		)
	})
}

func TestFunctionExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &FunctionExpression{
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
		StartPos: Position{Offset: 34, Line: 35, Column: 36},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "FunctionExpression",
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
            "StartPos": {"Offset": 34, "Line": 35, "Column": 36},
            "EndPos": {"Offset": 31, "Line": 32, "Column": 33}
        }
        `,
		string(actual),
	)
}

func TestFunctionExpression_Doc(t *testing.T) {

	t.Parallel()

	t.Run("no parameters, no return type, no statements", func(t *testing.T) {

		t.Parallel()

		expr := &FunctionExpression{
			ParameterList: &ParameterList{},
			FunctionBlock: &FunctionBlock{
				Block: &Block{
					Statements: []Statement{},
				},
			},
		}

		expected := prettier.Concat{
			prettier.Text("fun "),
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("()"),
				},
			},
			prettier.Text(" {}"),
		}

		assert.Equal(t, expected, expr.Doc())
	})

	t.Run("multiple parameters, return type, statements", func(t *testing.T) {

		t.Parallel()

		// TODO: pre-conditions and post-conditions

		expr := &FunctionExpression{
			ParameterList: &ParameterList{
				Parameters: []*Parameter{
					{
						Label: "a",
						Identifier: Identifier{
							Identifier: "b",
						},
						TypeAnnotation: &TypeAnnotation{
							Type: &NominalType{
								Identifier: Identifier{
									Identifier: "C",
								},
							},
						},
					},
					{
						Identifier: Identifier{
							Identifier: "d",
						},
						TypeAnnotation: &TypeAnnotation{
							Type: &NominalType{
								Identifier: Identifier{
									Identifier: "E",
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
						Identifier: "R",
					},
				},
			},
			FunctionBlock: &FunctionBlock{
				Block: &Block{
					Statements: []Statement{
						&ReturnStatement{
							Expression: &IntegerExpression{
								PositiveLiteral: "1",
								Value:           big.NewInt(1),
								Base:            10,
							},
						},
					},
				},
			},
		}

		expected := prettier.Concat{
			prettier.Text("fun "),
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Group{
						Doc: prettier.Concat{
							prettier.Text("("),
							prettier.Indent{
								Doc: prettier.Concat{
									prettier.SoftLine{},
									prettier.Concat{
										prettier.Concat{
											prettier.Text("a"),
											prettier.Space,
											prettier.Text("b"),
											prettier.Text(": "),
											prettier.Text("C"),
										},
										prettier.Concat{
											prettier.Text(","),
											prettier.Line{},
										},
										prettier.Concat{
											prettier.Text("d"),
											prettier.Text(": "),
											prettier.Text("E"),
										},
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
						prettier.Text("R"),
					},
				},
			},
			prettier.Text(" "),
			prettier.Concat{
				prettier.Text("{"),
				prettier.Indent{
					Doc: prettier.Concat{
						prettier.HardLine{},
						prettier.Concat{
							prettier.Text("return "),
							prettier.Text("1"),
						},
					},
				},
				prettier.HardLine{},
				prettier.Text("}"),
			},
		}

		assert.Equal(t, expected, expr.Doc())
	})

	t.Run("pre-conditions and post-conditions", func(t *testing.T) {

		t.Parallel()

		expr := &FunctionExpression{
			ParameterList: &ParameterList{},
			ReturnTypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Void",
					},
				},
			},
			FunctionBlock: &FunctionBlock{
				PreConditions: &Conditions{
					{
						Kind: ConditionKindPre,
						Test: &BoolExpression{
							Value: true,
						},
						Message: &StringExpression{
							Value: "pre",
						},
					},
				},
				PostConditions: &Conditions{
					{
						Kind: ConditionKindPre,
						Test: &BoolExpression{
							Value: false,
						},
						Message: &StringExpression{
							Value: "post",
						},
					},
				},
				Block: &Block{
					Statements: []Statement{
						&ReturnStatement{
							Expression: &IntegerExpression{
								PositiveLiteral: "1",
								Value:           big.NewInt(1),
								Base:            10,
							},
						},
					},
				},
			},
		}

		expected := prettier.Concat{
			prettier.Text("fun "),
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("()"),
					prettier.Text(": "),
					prettier.Text("Void"),
				},
			},
			prettier.Text(" "),
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
						prettier.Concat{
							prettier.HardLine{},
							prettier.Concat{
								prettier.Text("return "),
								prettier.Text("1"),
							},
						},
					},
				},
				prettier.HardLine{},
				prettier.Text("}"),
			},
		}

		assert.Equal(t, expected, expr.Doc())
	})
}

func TestFunctionExpression_String(t *testing.T) {

	t.Parallel()

	t.Run("no parameters, no return type, no statements", func(t *testing.T) {

		t.Parallel()

		expr := &FunctionExpression{
			ParameterList: &ParameterList{},
			FunctionBlock: &FunctionBlock{
				Block: &Block{
					Statements: []Statement{},
				},
			},
		}

		assert.Equal(t,
			"fun () {}",
			expr.String(),
		)
	})

	t.Run("multiple parameters, return type, statements", func(t *testing.T) {

		t.Parallel()

		// TODO: pre-conditions and post-conditions

		expr := &FunctionExpression{
			ParameterList: &ParameterList{
				Parameters: []*Parameter{
					{
						Label: "a",
						Identifier: Identifier{
							Identifier: "b",
						},
						TypeAnnotation: &TypeAnnotation{
							Type: &NominalType{
								Identifier: Identifier{
									Identifier: "C",
								},
							},
						},
					},
					{
						Identifier: Identifier{
							Identifier: "d",
						},
						TypeAnnotation: &TypeAnnotation{
							Type: &NominalType{
								Identifier: Identifier{
									Identifier: "E",
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
						Identifier: "R",
					},
				},
			},
			FunctionBlock: &FunctionBlock{
				Block: &Block{
					Statements: []Statement{
						&ReturnStatement{
							Expression: &IntegerExpression{
								PositiveLiteral: "1",
								Value:           big.NewInt(1),
								Base:            10,
							},
						},
					},
				},
			},
		}

		assert.Equal(t,
			"fun (a b: C, d: E): @R {\n"+
				"    return 1\n"+
				"}",
			expr.String(),
		)
	})

	t.Run("pre-conditions and post-conditions", func(t *testing.T) {

		t.Parallel()

		expr := &FunctionExpression{
			ParameterList: &ParameterList{},
			ReturnTypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Void",
					},
				},
			},
			FunctionBlock: &FunctionBlock{
				PreConditions: &Conditions{
					{
						Kind: ConditionKindPre,
						Test: &BoolExpression{
							Value: true,
						},
						Message: &StringExpression{
							Value: "pre",
						},
					},
				},
				PostConditions: &Conditions{
					{
						Kind: ConditionKindPre,
						Test: &BoolExpression{
							Value: false,
						},
						Message: &StringExpression{
							Value: "post",
						},
					},
				},
				Block: &Block{
					Statements: []Statement{
						&ReturnStatement{
							Expression: &IntegerExpression{
								PositiveLiteral: "1",
								Value:           big.NewInt(1),
								Base:            10,
							},
						},
					},
				},
			},
		}

		assert.Equal(t,
			"fun (): Void {\n"+
				"    pre {\n"+
				"        true:\n"+
				"            \"pre\"\n"+
				"    }\n"+
				"    post {\n"+
				"        false:\n"+
				"            \"post\"\n"+
				"    }\n"+
				"    return 1\n"+
				"}",
			expr.String(),
		)
	})
}
