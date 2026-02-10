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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"
)

func TestTypeParameterList_Doc(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		params := &TypeParameterList{}
		require.Equal(t,
			prettier.Text(""),
			params.Doc(),
		)
	})

	t.Run("with nil type parameter", func(t *testing.T) {
		t.Parallel()

		params := &TypeParameterList{
			TypeParameters: []*TypeParameter{nil},
		}
		require.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("<"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text(""),
						},
					},
					prettier.SoftLine{},
					prettier.Text(">"),
				},
			},
			params.Doc(),
		)
	})

	t.Run("with type parameters", func(t *testing.T) {
		t.Parallel()

		params := &TypeParameterList{
			TypeParameters: []*TypeParameter{
				{
					Identifier: Identifier{Identifier: "T"},
					TypeBound: &TypeAnnotation{
						Type: &NominalType{
							Identifier: Identifier{Identifier: "U"},
						},
					},
				},
				{
					Identifier: Identifier{Identifier: "V"},
				},
			},
		}

		require.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("<"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Concat{
								prettier.Concat{
									prettier.Text("T"),
									prettier.Text(": "),
									prettier.Text("U"),
								},
								prettier.Concat{
									prettier.Text(","),
									prettier.Line{},
								},
								prettier.Text("V"),
							},
						},
					},
					prettier.SoftLine{},
					prettier.Text(">"),
				},
			},
			params.Doc(),
		)
	})
}

func TestTypeParameterList_String(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		params := &TypeParameterList{}
		require.Equal(t,
			"",
			params.String(),
		)
	})

	t.Run("with nil type parameter", func(t *testing.T) {
		t.Parallel()

		params := &TypeParameterList{
			TypeParameters: []*TypeParameter{nil},
		}
		require.Equal(t,
			"<>",
			params.String(),
		)
	})

	t.Run("with type parameters", func(t *testing.T) {
		t.Parallel()

		params := &TypeParameterList{
			TypeParameters: []*TypeParameter{
				{
					Identifier: Identifier{Identifier: "T"},
					TypeBound: &TypeAnnotation{
						Type: &NominalType{
							Identifier: Identifier{Identifier: "U"},
						},
					},
				},
				{
					Identifier: Identifier{Identifier: "V"},
				},
			},
		}

		require.Equal(t,
			"<T: U, V>",
			params.String(),
		)
	})
}

func TestTypeParameterList_Walk(t *testing.T) {

	t.Parallel()

	typeBound1 := &TypeAnnotation{
		Type: &NominalType{
			Identifier: Identifier{Identifier: "Bound1"},
		},
	}

	typeBound2 := &TypeAnnotation{
		Type: &NominalType{
			Identifier: Identifier{Identifier: "Bound2"},
		},
	}

	params := &TypeParameterList{
		TypeParameters: []*TypeParameter{
			{
				Identifier: Identifier{Identifier: "T"},
				TypeBound:  typeBound1,
			},
			{
				Identifier: Identifier{Identifier: "U"},
				TypeBound:  typeBound2,
			},
			{
				Identifier: Identifier{Identifier: "V"},
				// No type bound
			},
		},
	}

	var visited []Element
	params.Walk(func(element Element) {
		visited = append(visited, element)
	})

	assert.Equal(t,
		[]Element{
			typeBound1,
			typeBound2,
		},
		visited,
	)
}
