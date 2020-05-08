/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package parser2

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseReturnStatement(t *testing.T) {

	t.Run("no expression", func(t *testing.T) {
		result, errs := ParseStatements("return")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
			},
			result,
		)
	})

	t.Run("expression on same line", func(t *testing.T) {
		result, errs := ParseStatements("return 1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Expression: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 1, Column: 7, Offset: 7},
							EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
						},
					},
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 7, Offset: 7},
					},
				},
			},
			result,
		)
	})

	t.Run("expression on next line, no semicolon", func(t *testing.T) {
		result, errs := ParseStatements("return \n1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
				&ast.ExpressionStatement{
					Expression: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 8},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 8},
						},
					},
				},
			},
			result,
		)
	})

	t.Run("expression on next line, semicolon", func(t *testing.T) {
		result, errs := ParseStatements("return ;\n1")
		require.Empty(t, errs)

		utils.AssertEqualWithDiff(t,
			[]ast.Statement{
				&ast.ReturnStatement{
					Range: ast.Range{
						StartPos: ast.Position{Line: 1, Column: 0, Offset: 0},
						EndPos:   ast.Position{Line: 1, Column: 5, Offset: 5},
					},
				},
				&ast.ExpressionStatement{
					Expression: &ast.IntegerExpression{
						Value: big.NewInt(1),
						Base:  10,
						Range: ast.Range{
							StartPos: ast.Position{Line: 2, Column: 0, Offset: 9},
							EndPos:   ast.Position{Line: 2, Column: 0, Offset: 9},
						},
					},
				},
			},
			result,
		)
	})
}
