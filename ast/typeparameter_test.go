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

func TestTypeParameter_Doc(t *testing.T) {

	t.Parallel()

	t.Run("without bound", func(t *testing.T) {
		t.Parallel()

		parameter := &TypeParameter{
			Identifier: Identifier{Identifier: "T"},
		}

		require.Equal(t,
			prettier.Text("T"),
			parameter.Doc(),
		)

	})

	t.Run("with bound", func(t *testing.T) {
		t.Parallel()

		parameter := &TypeParameter{
			Identifier: Identifier{Identifier: "T"},
			TypeBound: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{Identifier: "U"},
				},
			},
		}

		require.Equal(t,
			prettier.Concat{
				prettier.Text("T"),
				prettier.Text(": "),
				prettier.Text("U"),
			},
			parameter.Doc(),
		)
	})
}
