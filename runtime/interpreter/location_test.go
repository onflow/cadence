/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package interpreter

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/ast"
)

func TestLocationRange_HasPosition(t *testing.T) {

	t.Parallel()

	t.Run("nil", func(t *testing.T) {

		locationRange := LocationRange{}

		assert.Equal(t, ast.EmptyPosition, locationRange.StartPosition())
		assert.Equal(t, ast.EmptyPosition, locationRange.EndPosition(nil))
	})

	t.Run("non-nil", func(t *testing.T) {

		startPos := ast.Position{Offset: 1, Line: 2, Column: 3}
		endPos := ast.Position{Offset: 4, Line: 5, Column: 6}

		locationRange := LocationRange{
			HasPosition: ast.Range{
				StartPos: startPos,
				EndPos:   endPos,
			},
		}

		assert.Equal(t, startPos, locationRange.StartPosition())
		assert.Equal(t, endPos, locationRange.EndPosition(nil))

	})
}
