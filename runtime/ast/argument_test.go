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
)

func TestArgument_MarshalJSON(t *testing.T) {

	t.Parallel()

	t.Run("without label", func(t *testing.T) {

		t.Parallel()

		argument := &Argument{
			Expression: &BoolExpression{
				Value: false,
				Range: Range{
					StartPos: Position{Offset: 1, Line: 2, Column: 3},
					EndPos:   Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			TrailingSeparatorPos: Position{Offset: 7, Line: 8, Column: 9},
		}

		actual, err := json.Marshal(argument)
		require.NoError(t, err)

		assert.JSONEq(t,
			// language=json
			`
            {
                "Expression": {
                    "Type": "BoolExpression",
                    "Value": false,
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6},
                "TrailingSeparatorPos": {"Offset": 7, "Line": 8, "Column": 9}
            }
            `,
			string(actual),
		)
	})

	t.Run("with label", func(t *testing.T) {

		t.Parallel()

		argument := &Argument{
			Label:         "ok",
			LabelStartPos: &Position{Offset: 7, Line: 8, Column: 9},
			LabelEndPos:   &Position{Offset: 10, Line: 11, Column: 12},
			Expression: &BoolExpression{
				Value: false,
				Range: Range{
					StartPos: Position{Offset: 1, Line: 2, Column: 3},
					EndPos:   Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			TrailingSeparatorPos: Position{Offset: 13, Line: 14, Column: 15},
		}

		actual, err := json.Marshal(argument)
		require.NoError(t, err)

		assert.JSONEq(t,
			// language=json
			`
            {
                "Label": "ok",
                "LabelStartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "LabelEndPos": {"Offset": 10, "Line": 11, "Column": 12},
                "Expression": {
                    "Type": "BoolExpression",
                    "Value": false,
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
                },
			    "TrailingSeparatorPos": {"Offset": 13, "Line": 14, "Column": 15},
                "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
            }
            `,
			string(actual),
		)
	})
}

func TestArgument_Doc(t *testing.T) {

	t.Parallel()

	t.Run("without label", func(t *testing.T) {

		t.Parallel()

		argument := &Argument{
			Expression: &BoolExpression{
				Value: false,
			},
		}

		require.Equal(
			t,
			prettier.Text("false"),
			argument.Doc(),
		)
	})

	t.Run("with label", func(t *testing.T) {

		t.Parallel()

		argument := &Argument{
			Label: "ok",
			Expression: &BoolExpression{
				Value: false,
			},
		}

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("ok: "),
				prettier.Text("false"),
			},
			argument.Doc(),
		)
	})
}

func TestArgument_String(t *testing.T) {

	t.Parallel()

	t.Run("without label", func(t *testing.T) {

		t.Parallel()

		argument := &Argument{
			Expression: &BoolExpression{
				Value: false,
			},
		}

		require.Equal(
			t,
			"false",
			argument.String(),
		)
	})

	t.Run("with label", func(t *testing.T) {

		t.Parallel()

		argument := &Argument{
			Label: "ok",
			Expression: &BoolExpression{
				Value: false,
			},
		}

		require.Equal(
			t,
			"ok: false",
			argument.String(),
		)
	})
}
