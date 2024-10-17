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

package sema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckPathLiteral(t *testing.T) {

	t.Parallel()

	t.Run("valid domain (storage), valid identifier", func(t *testing.T) {
		t.Parallel()

		ty, err := CheckPathLiteral(nil, "storage", "test", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, StoragePathType, ty)
	})

	t.Run("valid domain (private), valid identifier", func(t *testing.T) {
		t.Parallel()

		ty, err := CheckPathLiteral(nil, "private", "test", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, PrivatePathType, ty)
	})

	t.Run("valid domain (public), valid identifier", func(t *testing.T) {
		t.Parallel()

		ty, err := CheckPathLiteral(nil, "public", "test", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, PublicPathType, ty)
	})

	t.Run("invalid domain (empty), valid identifier", func(t *testing.T) {
		t.Parallel()

		_, err := CheckPathLiteral(nil, "", "test", nil, nil)
		var invalidPathDomainError *InvalidPathDomainError
		require.ErrorAs(t, err, &invalidPathDomainError)
	})

	t.Run("invalid domain (foo), valid identifier", func(t *testing.T) {
		t.Parallel()

		_, err := CheckPathLiteral(nil, "foo", "test", nil, nil)
		var invalidPathDomainError *InvalidPathDomainError
		require.ErrorAs(t, err, &invalidPathDomainError)
	})

	t.Run("valid domain (public), invalid identifier (empty)", func(t *testing.T) {
		t.Parallel()

		_, err := CheckPathLiteral(nil, "public", "", nil, nil)
		var invalidPathIdentifierError *InvalidPathIdentifierError
		require.ErrorAs(t, err, &invalidPathIdentifierError)
	})

	t.Run("valid domain (public), invalid identifier ($)", func(t *testing.T) {
		t.Parallel()

		_, err := CheckPathLiteral(nil, "public", "$", nil, nil)
		var invalidPathIdentifierError *InvalidPathIdentifierError
		require.ErrorAs(t, err, &invalidPathIdentifierError)
	})

	t.Run("valid domain (public), invalid identifier (0)", func(t *testing.T) {
		t.Parallel()

		_, err := CheckPathLiteral(nil, "public", "0", nil, nil)
		var invalidPathIdentifierError *InvalidPathIdentifierError
		require.ErrorAs(t, err, &invalidPathIdentifierError)
	})
}
