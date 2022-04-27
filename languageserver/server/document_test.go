/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDocument_Offset(t *testing.T) {

	doc := Document{Text: "abcd\nefghijk\nlmno\npqr"}

	assert.Equal(t, 1, doc.Offset(1, 1))
	assert.Equal(t, 7, doc.Offset(2, 2))
	assert.Equal(t, 19, doc.Offset(4, 1))
}

func TestDocument_HasAnyPrecedingStringsAtPosition(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		doc := Document{Text: "  pub \t  \n  f"}

		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"pub"}, 2, 1))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"pub"}, 2, 2))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"pub"}, 2, 3))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"access(self)", "pub"}, 2, 2))
		assert.True(t, doc.HasAnyPrecedingStringsAtPosition([]string{"access(self)", "pub"}, 1, 6))
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		doc := Document{Text: "  pub \t  \n  f"}

		assert.False(t, doc.HasAnyPrecedingStringsAtPosition([]string{"access(self)"}, 2, 2))
	})
}
