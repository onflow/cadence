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

package sema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
)

func TestCheckPathLiteral(t *testing.T) {

	t.Parallel()

	rangeThunk := func() ast.Range {
		return ast.EmptyRange
	}

	t.Run("valid domain (storage), valid identifier", func(t *testing.T) {
		ty, err := CheckPathLiteral("storage", "test", rangeThunk, rangeThunk)
		require.NoError(t, err)
		assert.Equal(t, StoragePathType, ty)
	})

	t.Run("valid domain (private), valid identifier", func(t *testing.T) {
		ty, err := CheckPathLiteral("private", "test", rangeThunk, rangeThunk)
		require.NoError(t, err)
		assert.Equal(t, PrivatePathType, ty)
	})

	t.Run("valid domain (public), valid identifier", func(t *testing.T) {
		ty, err := CheckPathLiteral("public", "test", rangeThunk, rangeThunk)
		require.NoError(t, err)
		assert.Equal(t, PublicPathType, ty)
	})

	t.Run("invalid domain (empty), valid identifier", func(t *testing.T) {
		_, err := CheckPathLiteral("", "test", rangeThunk, rangeThunk)
		var invalidPathDomainError *InvalidPathDomainError
		require.ErrorAs(t, err, &invalidPathDomainError)
	})

	t.Run("invalid domain (foo), valid identifier", func(t *testing.T) {
		_, err := CheckPathLiteral("foo", "test", rangeThunk, rangeThunk)
		var invalidPathDomainError *InvalidPathDomainError
		require.ErrorAs(t, err, &invalidPathDomainError)
	})

	t.Run("valid domain (public), invalid identifier (empty)", func(t *testing.T) {
		_, err := CheckPathLiteral("public", "", rangeThunk, rangeThunk)
		var invalidPathIdentifierError *InvalidPathIdentifierError
		require.ErrorAs(t, err, &invalidPathIdentifierError)
	})

	t.Run("valid domain (public), invalid identifier ($)", func(t *testing.T) {
		_, err := CheckPathLiteral("public", "$", rangeThunk, rangeThunk)
		var invalidPathIdentifierError *InvalidPathIdentifierError
		require.ErrorAs(t, err, &invalidPathIdentifierError)
	})

	t.Run("valid domain (public), invalid identifier (0)", func(t *testing.T) {
		_, err := CheckPathLiteral("public", "0", rangeThunk, rangeThunk)
		var invalidPathIdentifierError *InvalidPathIdentifierError
		require.ErrorAs(t, err, &invalidPathIdentifierError)
	})
}
