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
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"
)

func TestParameterList_ParametersByIdentifier(t *testing.T) {

	t.Run("result", func(t *testing.T) {
		l := &ParameterList{
			Parameters: []*Parameter{
				{
					Label:      "c",
					Identifier: Identifier{Identifier: "d"},
				},
				{
					Label:      "a",
					Identifier: Identifier{Identifier: "b"},
				},
			},
		}

		require.Equal(t,
			map[string]*Parameter{
				"d": l.Parameters[0],
				"b": l.Parameters[1],
			},
			l.ParametersByIdentifier(),
		)
	})

	t.Run("thread-safety", func(t *testing.T) {

		// Ensure the ParametersByIdentifier index
		// is constructed in a thread-safe way

		l := &ParameterList{
			Parameters: []*Parameter{
				{
					Label:      "c",
					Identifier: Identifier{Identifier: "d"},
				},
				{
					Label:      "a",
					Identifier: Identifier{Identifier: "b"},
				},
			},
		}

		var wg sync.WaitGroup
		const parallelExecutionCount = 10

		for i := 0; i < parallelExecutionCount; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				l.ParametersByIdentifier()
			}()
		}

		wg.Wait()
	})
}

func TestParameterList_Doc(t *testing.T) {

	t.Parallel()

	params := &ParameterList{
		Parameters: []*Parameter{
			{
				Identifier: Identifier{Identifier: "e"},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{Identifier: "E"},
					},
				},
			},
			{
				Label:      "c",
				Identifier: Identifier{Identifier: "d"},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{Identifier: "D"},
					},
				},
			},
			{
				Label:      "a",
				Identifier: Identifier{Identifier: "b"},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{Identifier: "B"},
					},
				},
			},
		},
	}

	require.Equal(t,
		prettier.Group{
			Doc: prettier.Concat{
				prettier.Text("("),
				prettier.Indent{
					Doc: prettier.Concat{
						prettier.SoftLine{},
						prettier.Concat{
							prettier.Concat{
								prettier.Text("e"),
								prettier.Text(": "),
								prettier.Text("E"),
							},
							prettier.Concat{
								prettier.Text(","),
								prettier.Line{},
							},
							prettier.Concat{
								prettier.Text("c"),
								prettier.Text(" "),
								prettier.Text("d"),
								prettier.Text(": "),
								prettier.Text("D"),
							},
							prettier.Concat{
								prettier.Text(","),
								prettier.Line{},
							},
							prettier.Concat{
								prettier.Text("a"),
								prettier.Text(" "),
								prettier.Text("b"),
								prettier.Text(": "),
								prettier.Text("B"),
							},
						},
					},
				},
				prettier.SoftLine{},
				prettier.Text(")"),
			},
		},
		params.Doc(),
	)
}

func TestParameterList_String(t *testing.T) {

	t.Parallel()

	params := &ParameterList{
		Parameters: []*Parameter{
			{
				Identifier: Identifier{Identifier: "e"},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{Identifier: "E"},
					},
				},
			},
			{
				Label:      "c",
				Identifier: Identifier{Identifier: "d"},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{Identifier: "D"},
					},
				},
			},
			{
				Label:      "a",
				Identifier: Identifier{Identifier: "b"},
				TypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{Identifier: "B"},
					},
				},
			},
		},
	}

	require.Equal(t,
		"(e: E, c d: D, a b: B)",
		params.String(),
	)
}
