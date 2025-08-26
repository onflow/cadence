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

	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"
)

func TestParameter_Doc(t *testing.T) {

	t.Parallel()

	t.Run("without label", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Identifier: Identifier{Identifier: "e"},
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{Identifier: "E"},
				},
			},
		}

		require.Equal(t,
			prettier.Concat{
				prettier.Text("e"),
				prettier.Text(": "),
				prettier.Text("E"),
			},
			parameter.Doc(),
		)

	})

	t.Run("with label", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Label:      "c",
			Identifier: Identifier{Identifier: "d"},
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{Identifier: "D"},
				},
			},
		}

		require.Equal(t,
			prettier.Concat{
				prettier.Text("c"),
				prettier.Text(" "),
				prettier.Text("d"),
				prettier.Text(": "),
				prettier.Text("D"),
			},
			parameter.Doc(),
		)
	})

	t.Run("with label, without type annotation", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Label:      "a",
			Identifier: Identifier{Identifier: "b"},
		}

		require.Equal(t,
			prettier.Concat{
				prettier.Text("a"),
				prettier.Text(" "),
				prettier.Text("b"),
				prettier.Text(": "),
				prettier.Text(""),
			},
			parameter.Doc(),
		)
	})

	t.Run("without label, without type annotation", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Identifier: Identifier{Identifier: "b"},
		}

		require.Equal(t,
			prettier.Concat{
				prettier.Text("b"),
				prettier.Text(": "),
				prettier.Text(""),
			},
			parameter.Doc(),
		)
	})

	t.Run("with default argument", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Identifier: Identifier{Identifier: "bool"},
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{Identifier: "Bool"},
				},
			},
			DefaultArgument: &BoolExpression{Value: true},
		}

		require.Equal(t,
			prettier.Concat{
				prettier.Text("bool"),
				prettier.Text(": "),
				prettier.Text("Bool"),
				prettier.Text(" "),
				prettier.Text("="),
				prettier.Text(" "),
				prettier.Text("true"),
			},
			parameter.Doc(),
		)
	})
}

func TestParameter_String(t *testing.T) {

	t.Parallel()

	t.Run("without label", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Identifier: Identifier{Identifier: "e"},
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{Identifier: "E"},
				},
			},
		}

		require.Equal(t,
			"e: E",
			parameter.String(),
		)

	})

	t.Run("with label", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Label:      "c",
			Identifier: Identifier{Identifier: "d"},
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{Identifier: "D"},
				},
			},
		}

		require.Equal(t,
			"c d: D",
			parameter.String(),
		)
	})

	t.Run("with label, without type annotation", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Label:      "a",
			Identifier: Identifier{Identifier: "b"},
		}

		require.Equal(t,
			"a b: ",
			parameter.String(),
		)
	})

	t.Run("without label, without type annotation", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Identifier: Identifier{Identifier: "b"},
		}

		require.Equal(t,
			"b: ",
			parameter.String(),
		)
	})

	t.Run("with default argument", func(t *testing.T) {
		t.Parallel()

		parameter := &Parameter{
			Identifier: Identifier{Identifier: "bool"},
			TypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{Identifier: "Bool"},
				},
			},
			DefaultArgument: &BoolExpression{Value: true},
		}

		require.Equal(t,
			"bool: Bool = true",
			parameter.String(),
		)
	})
}
