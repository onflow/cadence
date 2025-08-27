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

func TestProgram_MarshalJSON(t *testing.T) {

	t.Parallel()

	program := NewProgram(nil, []Declaration{})

	actual, err := json.Marshal(program)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "Program",
            "Declarations": []
        }
        `,
		string(actual),
	)
}

func TestProgram_Doc(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		program := NewProgram(nil, []Declaration{})

		assert.Equal(t,
			prettier.Text(""),
			program.Doc(),
		)
	})

	t.Run("with nil declaration", func(t *testing.T) {
		t.Parallel()

		program := NewProgram(nil, []Declaration{
			nil,
		})

		assert.Equal(t,
			prettier.Text(""),
			program.Doc(),
		)
	})

	t.Run("with declarations", func(t *testing.T) {
		t.Parallel()

		program := NewProgram(nil, []Declaration{
			&PragmaDeclaration{
				Expression: &BoolExpression{Value: true},
			},
			&PragmaDeclaration{
				Expression: &BoolExpression{Value: false},
			},
		})

		assert.Equal(t,
			prettier.Concat{
				prettier.Concat{
					prettier.Text("#"),
					prettier.Text("true"),
				},
				prettier.Concat{
					prettier.HardLine{},
					prettier.HardLine{},
				},
				prettier.Concat{
					prettier.Text("#"),
					prettier.Text("false"),
				},
			},
			program.Doc(),
		)
	})
}

func TestProgram_String(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		program := NewProgram(nil, []Declaration{})

		assert.Equal(t,
			"",
			program.String(),
		)
	})

	t.Run("with nil declaration", func(t *testing.T) {
		t.Parallel()

		program := NewProgram(nil, []Declaration{
			nil,
		})

		assert.Equal(t,
			"",
			program.String(),
		)
	})

	t.Run("with declarations", func(t *testing.T) {
		t.Parallel()

		program := NewProgram(nil, []Declaration{
			&PragmaDeclaration{
				Expression: &BoolExpression{Value: true},
			},
			&PragmaDeclaration{
				Expression: &BoolExpression{Value: false},
			},
		})

		assert.Equal(t,
			"#true\n"+
				"\n"+
				"#false",
			program.String(),
		)
	})

}
