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

package format

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBytes(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		require.Equal(t, "[]", Bytes([]byte{}))
	})

	t.Run("one", func(t *testing.T) {
		require.Equal(t, "[0x1]", Bytes([]byte{0x1}))
	})

	t.Run("two", func(t *testing.T) {
		require.Equal(t, "[0x1, 0x2]", Bytes([]byte{0x1, 0x2}))
	})
}
