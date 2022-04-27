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

func TestVariableDeclaration_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &VariableDeclaration{
		Access:     AccessPublic,
		IsConstant: true,
		Identifier: Identifier{
			Identifier: "foo",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
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
		Value: &BoolExpression{
			Value: true,
			Range: Range{
				StartPos: Position{Offset: 10, Line: 11, Column: 12},
				EndPos:   Position{Offset: 13, Line: 14, Column: 15},
			},
		},
		Transfer: &Transfer{
			Operation: TransferOperationMove,
			Pos:       Position{Offset: 16, Line: 17, Column: 18},
		},
		StartPos: Position{Offset: 19, Line: 20, Column: 21},
		SecondTransfer: &Transfer{
			Operation: TransferOperationMove,
			Pos:       Position{Offset: 22, Line: 23, Column: 24},
		},
		SecondValue: &BoolExpression{
			Value: false,
			Range: Range{
				StartPos: Position{Offset: 25, Line: 26, Column: 27},
				EndPos:   Position{Offset: 28, Line: 29, Column: 30},
			},
		},
		DocString: "test",
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		`
        {
            "Type": "VariableDeclaration",
            "Access": "AccessPublic",
            "IsConstant": true,
            "Identifier": {
                "Identifier": "foo",
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 3, "Line": 2, "Column": 5}
            },
            "TypeAnnotation": {
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 5, "Line": 5, "Column": 7},
                "IsResource": true,
                "AnnotatedType": {
                    "Type": "NominalType",
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7},
                    "Identifier": {
                        "Identifier": "AB",
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                    }
                }
            }, 
            "Transfer": {
                "Type": "Transfer",
                "Operation": "TransferOperationMove",
                "StartPos": {"Offset": 16, "Line": 17, "Column": 18},
                "EndPos": {"Offset": 17, "Line": 17, "Column": 19}
            },
            "Value": {
                "Type": "BoolExpression",
                "Value": true,
                "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
            },
            "SecondTransfer": {
                "Type": "Transfer", 
                "Operation": "TransferOperationMove",
                "StartPos": {"Offset": 22, "Line": 23, "Column": 24},
                "EndPos": {"Offset": 23, "Line": 23, "Column": 25}
            },
            "SecondValue": {
                "Type": "BoolExpression",
                "Value": false,
                "StartPos": {"Offset": 25, "Line": 26, "Column": 27},
                "EndPos": {"Offset": 28, "Line": 29, "Column": 30}
            },
            "DocString": "test",
            "StartPos": {"Offset": 19, "Line": 20, "Column": 21},
            "EndPos": {"Offset": 28, "Line": 29, "Column": 30}
        }
        `,
		string(actual),
	)
}
