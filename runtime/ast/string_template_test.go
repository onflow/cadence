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

package ast_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/stretchr/testify/assert"
	"github.com/turbolent/prettier"
)

func TestStringTemplate_Doc(t *testing.T) {

	t.Parallel()

	stmt := &ast.StringTemplateExpression{
		Values: []string{
			"abc",
		},
		Expressions: []ast.Expression{},
		Range: ast.Range{
			StartPos: ast.Position{Offset: 4, Line: 2, Column: 3},
			EndPos:   ast.Position{Offset: 11, Line: 2, Column: 10},
		},
	}

	assert.Equal(t,
		prettier.Text("abc"),
		stmt.Doc(),
	)
}
