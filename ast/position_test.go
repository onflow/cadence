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
)

func TestPosition_AttachLeft(t *testing.T) {

	t.Parallel()

	t.Run("nothing", func(t *testing.T) {
		t.Parallel()

		const code = "bar"

		assert.Equal(
			t,
			Position{Offset: 0, Line: 1, Column: 0},
			Position{Offset: 0, Line: 1, Column: 0}.AttachLeft(code),
		)
	})

	t.Run("only whitespace", func(t *testing.T) {
		t.Parallel()

		const code = "  bar"

		assert.Equal(
			t,
			Position{Offset: 0, Line: 1, Column: 0},
			Position{Offset: 2, Line: 1, Column: 2}.AttachLeft(code),
		)
	})

	t.Run("non-whitespace", func(t *testing.T) {
		t.Parallel()

		const code = "foo  bar"

		assert.Equal(
			t,
			Position{Offset: 3, Line: 1, Column: 3},
			Position{Offset: 5, Line: 1, Column: 5}.AttachLeft(code),
		)
	})

	t.Run("whitespace, across newline (\\n)", func(t *testing.T) {
		t.Parallel()

		const code = "foo\n  bar"

		assert.Equal(
			t,
			Position{Offset: 3, Line: 1, Column: 3},
			Position{Offset: 6, Line: 2, Column: 2}.AttachLeft(code),
		)
	})

	t.Run("whitespace, across newline (\\r\\n)", func(t *testing.T) {
		t.Parallel()

		const code = "foo\r\n  bar"

		assert.Equal(
			t,
			Position{Offset: 3, Line: 1, Column: 3},
			Position{Offset: 7, Line: 2, Column: 2}.AttachLeft(code),
		)
	})
}

func TestPosition_AttachRight(t *testing.T) {

	t.Parallel()

	t.Run("nothing", func(t *testing.T) {
		t.Parallel()

		const code = "bar"

		assert.Equal(
			t,
			Position{Offset: 0, Line: 1, Column: 0},
			Position{Offset: 0, Line: 1, Column: 0}.AttachRight(code),
		)
	})

	t.Run("only whitespace", func(t *testing.T) {
		t.Parallel()

		const code = "bar  "

		assert.Equal(
			t,
			Position{Offset: 4, Line: 1, Column: 4},
			Position{Offset: 2, Line: 1, Column: 2}.AttachRight(code),
		)
	})

	t.Run("non-whitespace", func(t *testing.T) {
		t.Parallel()

		const code = "foo  bar"

		assert.Equal(
			t,
			Position{Offset: 4, Line: 1, Column: 4},
			Position{Offset: 2, Line: 1, Column: 2}.AttachRight(code),
		)
	})

	t.Run("whitespace, across newline (\\n)", func(t *testing.T) {
		t.Parallel()

		const code = "foo\n  bar"

		assert.Equal(
			t,
			Position{Offset: 5, Line: 2, Column: 1},
			Position{Offset: 2, Line: 1, Column: 2}.AttachRight(code),
		)
	})

	t.Run("whitespace, across newline (\\r\\n)", func(t *testing.T) {
		t.Parallel()

		const code = "foo\r\n  bar"

		assert.Equal(
			t,
			Position{Offset: 6, Line: 2, Column: 1},
			Position{Offset: 2, Line: 1, Column: 2}.AttachRight(code),
		)
	})
}
