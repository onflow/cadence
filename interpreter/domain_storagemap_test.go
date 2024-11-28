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

package interpreter_test

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"

	"github.com/stretchr/testify/require"
)

func TestDomainStorageMapValueExists(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, domainStorageMap)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		key := interpreter.StringAtreeValue("key")
		exist := domainStorageMap.ValueExists(interpreter.StringStorageMapKey(key))
		require.False(t, exist)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because DomainStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match DomainStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		const count = 10
		domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

		// Check if value exists
		for key := range domainValues {
			exist := domainStorageMap.ValueExists(key)
			require.True(t, exist)
		}

		// Check if random value exists
		for range 10 {
			n := random.Int()
			key := interpreter.StringStorageMapKey(strconv.Itoa(n))
			_, keyExist := domainValues[key]

			exist := domainStorageMap.ValueExists(key)
			require.Equal(t, keyExist, exist)
		}

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})
}

func TestDomainStorageMapReadValue(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, domainStorageMap)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		key := interpreter.StringAtreeValue("key")
		v := domainStorageMap.ReadValue(nil, interpreter.StringStorageMapKey(key))
		require.Nil(t, v)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because DomainStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match DomainStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		const count = 10
		domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

		for key, expectedValue := range domainValues {
			value := domainStorageMap.ReadValue(nil, key)
			require.NotNil(t, value)

			checkCadenceValue(t, inter, value, expectedValue)
		}

		// Get non-existent value
		for range 10 {
			n := random.Int()
			key := interpreter.StringStorageMapKey(strconv.Itoa(n))
			if _, keyExist := domainValues[key]; keyExist {
				continue
			}

			value := domainStorageMap.ReadValue(nil, key)
			require.Nil(t, value)
		}

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})
}

func TestDomainStorageMapSetAndUpdateValue(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, domainStorageMap)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		const count = 10
		domainValues := writeRandomValuesToDomainStorageMap(inter, domainStorageMap, count, random)

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		const count = 10
		domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

		for key := range domainValues {
			// Overwrite existing values
			n := random.Int()

			value := interpreter.NewUnmeteredIntValueFromInt64(int64(n))

			domainStorageMap.WriteValue(inter, key, value)

			domainValues[key] = value
		}
		require.Equal(t, uint64(count), domainStorageMap.Count())

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})
}

func TestDomainStorageMapRemoveValue(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, domainStorageMap)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		key := interpreter.StringAtreeValue("key")
		existed := domainStorageMap.WriteValue(inter, interpreter.StringStorageMapKey(key), nil)
		require.False(t, existed)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		const count = 10
		domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

		for key := range domainValues {
			existed := domainStorageMap.WriteValue(inter, key, nil)
			require.True(t, existed)
		}

		// Remove non-existent value
		for range 10 {
			n := random.Int()
			key := interpreter.StringStorageMapKey(strconv.Itoa(n))
			if _, keyExist := domainValues[key]; keyExist {
				continue
			}

			existed := domainStorageMap.WriteValue(inter, key, nil)
			require.False(t, existed)
		}

		clear(domainValues)

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})
}

func TestDomainStorageMapIteratorNext(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		domainValues := make(domainStorageMapValues)

		domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, domainStorageMap)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		iterator := domainStorageMap.Iterator(nil)

		// Test calling Next() twice on empty account storage map.
		for range 2 {
			k, v := iterator.Next()
			require.Nil(t, k)
			require.Nil(t, v)
		}

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		const count = 10
		domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

		iterator := domainStorageMap.Iterator(nil)

		elementCount := 0
		for {
			k, v := iterator.Next()
			if k == nil {
				break
			}

			elementCount++

			kv := k.(interpreter.StringAtreeValue)

			expectedValue, expectedValueExist := domainValues[interpreter.StringStorageMapKey(kv)]
			require.True(t, expectedValueExist)

			checkCadenceValue(t, inter, v, expectedValue)
		}
		require.Equal(t, uint64(elementCount), domainStorageMap.Count())

		// Test calling Next() after iterator reaches the end.
		for range 2 {
			k, v := iterator.Next()
			require.Nil(t, k)
			require.Nil(t, v)
		}

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})
}

func TestDomainStorageMapIteratorNextKey(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		domainValues := make(domainStorageMapValues)

		domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, domainStorageMap)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		iterator := domainStorageMap.Iterator(nil)

		// Test calling NextKey() twice on empty account storage map.
		for range 2 {
			k := iterator.NextKey()
			require.Nil(t, k)
		}

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		const count = 10
		domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

		iterator := domainStorageMap.Iterator(nil)

		elementCount := 0
		for {
			k := iterator.NextKey()
			if k == nil {
				break
			}

			elementCount++

			kv := k.(interpreter.StringAtreeValue)

			_, expectedValueExist := domainValues[interpreter.StringStorageMapKey(kv)]
			require.True(t, expectedValueExist)
		}
		require.Equal(t, uint64(elementCount), domainStorageMap.Count())

		// Test calling Next() after iterator reaches the end.
		for range 2 {
			k := iterator.NextKey()
			require.Nil(t, k)
		}

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})
}

func TestDomainStorageMapIteratorNextValue(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		domainValues := make(domainStorageMapValues)

		domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, domainStorageMap)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		iterator := domainStorageMap.Iterator(nil)

		// Test calling NextKey() twice on empty account storage map.
		for range 2 {
			v := iterator.NextValue()
			require.Nil(t, v)
		}

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		random := rand.New(rand.NewSource(42))

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
		// This is because AccountStorageMap isn't created through runtime.Storage, so there isn't any
		// account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
			t,
			storage,
			atreeValueValidationEnabled,
			atreeStorageValidationEnabled,
		)

		const count = 10
		domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

		iterator := domainStorageMap.Iterator(nil)

		elementCount := 0
		for {
			v := iterator.NextValue()
			if v == nil {
				break
			}

			elementCount++

			ev, ok := v.(interpreter.EquatableValue)
			require.True(t, ok)

			match := false
			for _, expectedValue := range domainValues {
				if ev.Equal(inter, interpreter.EmptyLocationRange, expectedValue) {
					match = true
					break
				}
			}
			require.True(t, match)
		}
		require.Equal(t, uint64(elementCount), domainStorageMap.Count())

		// Test calling NextValue() after iterator reaches the end.
		for range 2 {
			v := iterator.NextValue()
			require.Nil(t, v)
		}

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		valueID := domainStorageMap.ValueID()
		CheckAtreeStorageHealth(t, storage, []atree.SlabID{atreeValueIDToSlabID(valueID)})
	})
}

func TestDomainStorageMapLoadFromRootSlabID(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		init := func() (atree.SlabID, domainStorageMapValues, map[string][]byte, map[string]uint64) {
			ledger := NewTestLedger(nil, nil)
			storage := runtime.NewStorage(
				ledger,
				nil,
				runtime.StorageConfig{},
			)

			inter := NewTestInterpreterWithStorage(t, storage)

			domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
			require.NotNil(t, domainStorageMap)
			require.Equal(t, uint64(0), domainStorageMap.Count())

			err := storage.Commit(inter, false)
			require.NoError(t, err)

			valueID := domainStorageMap.ValueID()
			return atreeValueIDToSlabID(valueID), make(domainStorageMapValues), ledger.StoredValues, ledger.StorageIndices
		}

		domainStorageMapRootSlabID, domainValues, storedValues, storageIndices := init()

		ledger := NewTestLedgerWithData(nil, nil, storedValues, storageIndices)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		domainStorageMap := interpreter.NewDomainStorageMapWithRootID(storage, domainStorageMapRootSlabID)
		require.Equal(t, uint64(0), domainStorageMap.Count())

		inter := NewTestInterpreterWithStorage(t, storage)

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{domainStorageMapRootSlabID})
	})

	t.Run("non-empty", func(t *testing.T) {
		t.Parallel()

		init := func() (atree.SlabID, domainStorageMapValues, map[string][]byte, map[string]uint64) {
			random := rand.New(rand.NewSource(42))

			ledger := NewTestLedger(nil, nil)
			storage := runtime.NewStorage(
				ledger,
				nil,
				runtime.StorageConfig{},
			)

			// Turn off automatic AtreeStorageValidationEnabled and explicitly check atree storage health directly.
			// This is because AccountStorageMap isn't created through storage, so there isn't any account register to match AccountStorageMap root slab.
			const atreeValueValidationEnabled = true
			const atreeStorageValidationEnabled = false
			inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(t, storage, atreeValueValidationEnabled, atreeStorageValidationEnabled)

			const count = 10
			domainStorageMap, domainValues := createDomainStorageMap(storage, inter, address, count, random)

			err := storage.Commit(inter, false)
			require.NoError(t, err)

			valueID := domainStorageMap.ValueID()
			return atreeValueIDToSlabID(valueID), domainValues, ledger.StoredValues, ledger.StorageIndices
		}

		domainStorageMapRootSlabID, domainValues, storedValues, storageIndices := init()

		ledger := NewTestLedgerWithData(nil, nil, storedValues, storageIndices)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{},
		)

		domainStorageMap := interpreter.NewDomainStorageMapWithRootID(storage, domainStorageMapRootSlabID)

		inter := NewTestInterpreterWithStorage(t, storage)

		checkDomainStorageMapData(t, inter, domainStorageMap, domainValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{domainStorageMapRootSlabID})
	})
}

func createDomainStorageMap(
	storage atree.SlabStorage,
	inter *interpreter.Interpreter,
	address common.Address,
	count int,
	random *rand.Rand,
) (*interpreter.DomainStorageMap, domainStorageMapValues) {

	// Create domain storage map
	domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))

	// Write to new domain storage map
	domainValues := writeRandomValuesToDomainStorageMap(inter, domainStorageMap, count, random)

	return domainStorageMap, domainValues
}

func atreeValueIDToSlabID(vid atree.ValueID) atree.SlabID {
	return atree.NewSlabID(
		atree.Address(vid[:8]),
		atree.SlabIndex(vid[8:]),
	)
}
