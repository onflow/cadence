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
)

func TestMembers_MarshalJSON(t *testing.T) {

	t.Parallel()

	members := NewUnmeteredMembers([]Declaration{})

	actual, err := json.Marshal(members)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Declarations": []
        }
        `,
		string(actual),
	)
}

func TestMembers_Doc(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		members := NewUnmeteredMembers([]Declaration{})

		require.Equal(t,
			prettier.Text("{}"),
			members.Doc(),
		)
	})

	t.Run("with members", func(t *testing.T) {
		t.Parallel()

		members := NewUnmeteredMembers([]Declaration{
			&VariableDeclaration{
				Access: AccessNotSpecified,
				Identifier: Identifier{
					Identifier: "x",
				},
				Transfer: &Transfer{
					Operation: TransferOperationCopy,
				},
				Value: &BoolExpression{
					Value: true,
				},
			},
			&VariableDeclaration{
				Access: AccessNotSpecified,
				Identifier: Identifier{
					Identifier: "y",
				},
				Transfer: &Transfer{
					Operation: TransferOperationCopy,
				},
				Value: &BoolExpression{
					Value: false,
				},
			},
		})

		require.Equal(
			t,
			prettier.Concat{
				prettier.Text("{"),
				prettier.Indent{
					Doc: prettier.Concat{
						prettier.Concat{
							prettier.HardLine{},
							prettier.Group{
								Doc: prettier.Concat{
									prettier.Text("var"),
									prettier.Text(" "),
									prettier.Group{
										Doc: prettier.Concat{
											prettier.Group{
												Doc: prettier.Concat{
													prettier.Text("x"),
												},
											},
											prettier.Text(" "),
											prettier.Text("="),
											prettier.Group{
												Doc: prettier.Indent{
													Doc: prettier.Concat{
														prettier.Line{},
														prettier.Text("true"),
													},
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
							prettier.Group{
								Doc: prettier.Concat{
									prettier.Text("var"),
									prettier.Text(" "),
									prettier.Group{
										Doc: prettier.Concat{
											prettier.Group{
												Doc: prettier.Concat{
													prettier.Text("y"),
												},
											},
											prettier.Text(" "),
											prettier.Text("="),
											prettier.Group{
												Doc: prettier.Indent{
													Doc: prettier.Concat{
														prettier.Line{},
														prettier.Text("false"),
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
				prettier.HardLine{},
				prettier.Text("}"),
			},
			members.Doc(),
		)
	})

}

func TestMembers_String(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		members := NewUnmeteredMembers([]Declaration{})

		require.Equal(t,
			prettier.Text("{}"),
			members.Doc(),
		)
	})

	t.Run("with members", func(t *testing.T) {
		t.Parallel()

		members := NewUnmeteredMembers([]Declaration{
			&VariableDeclaration{
				Access: AccessNotSpecified,
				Identifier: Identifier{
					Identifier: "x",
				},
				Transfer: &Transfer{
					Operation: TransferOperationCopy,
				},
				Value: &BoolExpression{
					Value: true,
				},
			},
			&VariableDeclaration{
				Access: AccessNotSpecified,
				Identifier: Identifier{
					Identifier: "y",
				},
				Transfer: &Transfer{
					Operation: TransferOperationCopy,
				},
				Value: &BoolExpression{
					Value: false,
				},
			},
		})

		require.Equal(
			t,
			`{
    var x = true
    
    var y = false
}`,
			members.String(),
		)
	})

}
