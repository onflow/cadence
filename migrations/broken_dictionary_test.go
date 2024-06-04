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

package migrations

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
)

//go:embed testdata/missing-slabs-payloads.csv
var missingSlabsPayloadsData []byte

// '$' + 8 byte index
const slabKeyLength = 9

func isSlabStorageKey(key []byte) bool {
	return len(key) == slabKeyLength && key[0] == '$'
}

func TestFixLoadedBrokenReferences(t *testing.T) {

	t.Parallel()

	// Read CSV file with test data

	reader := csv.NewReader(bytes.NewReader(missingSlabsPayloadsData))

	// account, key, value
	reader.FieldsPerRecord = 3

	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Load data into ledger. Skip header

	ledger := NewTestLedger(nil, nil)

	for _, record := range records[1:] {
		account, err := hex.DecodeString(record[0])
		require.NoError(t, err)

		key, err := hex.DecodeString(record[1])
		require.NoError(t, err)

		value, err := hex.DecodeString(record[2])
		require.NoError(t, err)

		err = ledger.SetValue(account, key, value)
		require.NoError(t, err)
	}

	storage := runtime.NewStorage(ledger, nil)

	// Check health.
	// Retrieve all slabs before migration

	err = ledger.ForEach(func(owner, key, value []byte) error {

		if !isSlabStorageKey(key) {
			return nil
		}

		// Convert the owner/key to a storage ID.

		var storageIndex atree.SlabIndex
		copy(storageIndex[:], key[1:])

		storageID := atree.NewSlabID(atree.Address(owner), storageIndex)

		// Retrieve the slab.
		_, found, err := storage.Retrieve(storageID)
		require.NoError(t, err)
		require.True(t, found)

		return nil
	})
	require.NoError(t, err)

	address, err := common.HexToAddress("0x5d63c34d7f05e5a4")
	require.NoError(t, err)

	for _, domain := range common.AllPathDomains {
		_ = storage.GetStorageMap(address, domain.Identifier(), false)
	}

	err = storage.CheckHealth()
	require.Error(t, err)

	require.ErrorContains(t, err, "slab (0x0.49) not found: slab not found during slab iteration")

	// Fix the broken slab references

	fixedSlabs, skippedSlabIDs, err := storage.PersistentSlabStorage.
		FixLoadedBrokenReferences(ShouldFixBrokenCompositeKeyedDictionary)
	require.NoError(t, err)

	require.NotEmpty(t, fixedSlabs)
	require.Empty(t, skippedSlabIDs)

	// Re-run health check. This time it should pass.

	err = storage.CheckHealth()
	require.NoError(t, err)
}
