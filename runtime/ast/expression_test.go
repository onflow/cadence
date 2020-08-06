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
