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

package interpreter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
)

func TestLargeStringAtreeValueInSeparateSlab(t *testing.T) {

	t.Parallel()

	// Ensure that StringAtreeValue handles the case where it is stored in a separate slab,
	// when the string is very large

	storage := NewInMemoryStorage(nil)

	inter, err := NewInterpreter(
		nil,
		common.StringLocation("test"),
		&Config{
			Storage: storage,
		},
	)
	require.NoError(t, err)

	storageMap := storage.GetDomainStorageMap(
		inter,
		common.MustBytesToAddress([]byte{0x1}),
		common.PathDomainStorage.Identifier(),
		true,
	)

	// Generate a large key to force the string to get stored in a separate slab
	keyValue := NewStringAtreeValue(nil, strings.Repeat("x", 10_000))

	key := StringStorageMapKey(keyValue)

	expected := NewUnmeteredUInt8Value(42)
	storageMap.SetValue(inter, key, expected)

	actual := storageMap.ReadValue(nil, key)

	require.Equal(t, expected, actual)
}
