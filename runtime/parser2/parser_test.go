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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
)

func TestParseExpression(t *testing.T) {

	expectedSimpleExpression := &ast.BinaryExpression{
		Operation: ast.OperationPlus,
		Left: &ast.IntegerExpression{
			Value: big.NewInt(1),
			Base:  10,
		},
		Right: &ast.BinaryExpression{
			Operation: ast.OperationMul,
			Left: &ast.IntegerExpression{
				Value: big.NewInt(2),
				Base:  10,
			},
			Right: &ast.IntegerExpression{
				Value: big.NewInt(3),
				Base:  10,
			},
		},
	}

	t.Run("simple, no spaces", func(t *testing.T) {
		result, errors := Parse("1+2*3")
		require.Empty(t, errors)

		assert.Equal(t,
			expectedSimpleExpression,
			result,
		)
	})

	t.Run("simple, spaces", func(t *testing.T) {
		result, errors := Parse("  1   +   2  *   3 ")
		require.Empty(t, errors)

		assert.Equal(t,
			expectedSimpleExpression,
			result,
		)
	})
}
