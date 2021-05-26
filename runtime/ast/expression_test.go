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
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIntegerExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &IntegerExpression{
		Value: big.NewInt(42),
		Base:  10,
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
            "Value": "42",
            "Base": 10,
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestFixedPointExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &FixedPointExpression{
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

func TestUnaryExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &UnaryExpression{
		Operation: OperationNegate,
		Expression: &IntegerExpression{
			Value: big.NewInt(42),
			Base:  10,
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

func TestBinaryExpression_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &BinaryExpression{
		Operation: OperationPlus,
		Left: &IntegerExpression{
			Value: big.NewInt(42),
			Base:  10,
			Range: Range{
				StartPos: Position{Offset: 1, Line: 2, Column: 3},
				EndPos:   Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Right: &IntegerExpression{
			Value: big.NewInt(99),
			Base:  10,
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
                "Value": "42",
                "Base": 10,
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            },
            "Right": {
                "Type": "IntegerExpression",
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
			Value: big.NewInt(42),
			Base:  10,
			Range: Range{
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
				EndPos:   Position{Offset: 10, Line: 11, Column: 12},
			},
		},
		Else: &IntegerExpression{
			Value: big.NewInt(99),
			Base:  10,
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
                "Value": "42",
                "Base": 10,
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
            },
            "Else": {
                "Type": "IntegerExpression",
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
