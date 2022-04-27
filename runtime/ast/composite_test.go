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

	"github.com/onflow/cadence/runtime/common"
)

func TestFieldDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &FieldDeclaration{
		Access:       AccessPublic,
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

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "FieldDeclaration",
            "Access": "AccessPublic",
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

func TestCompositeDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	expr := &CompositeDeclaration{
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
		Members:   NewMembers([]Declaration{}),
		DocString: "test",
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
	}

	actual, err := json.Marshal(expr)
	require.NoError(t, err)

	assert.JSONEq(t,
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
