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
	goruntime "runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"

	"github.com/stretchr/testify/require"
)

func TestAccountStorageMapDomainExists(t *testing.T) {
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

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, accountStorageMap)
		require.Equal(t, uint64(0), accountStorageMap.Count())

		for _, domain := range common.AllStorageDomains {
			exist := accountStorageMap.DomainExists(domain)
			require.False(t, exist)
		}

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
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

		existingDomains := []common.StorageDomain{common.PathDomainStorage.StorageDomain()}

		const count = 10
		accountStorageMap, _ := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

		// Check if domain exists
		for _, domain := range common.AllStorageDomains {
			exist := accountStorageMap.DomainExists(domain)
			require.Equal(t, slices.Contains(existingDomains, domain), exist)
		}

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
	})
}

func TestAccountStorageMapGetDomain(t *testing.T) {
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

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, accountStorageMap)
		require.Equal(t, uint64(0), accountStorageMap.Count())

		for _, domain := range common.AllStorageDomains {
			const createIfNotExists = false
			domainStorageMap := accountStorageMap.GetDomain(nil, inter, domain, createIfNotExists)
			require.Nil(t, domainStorageMap)
		}

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
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

		existingDomains := []common.StorageDomain{common.PathDomainStorage.StorageDomain()}

		const count = 10
		accountStorageMap, accountValues := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

		for _, domain := range common.AllStorageDomains {
			const createIfNotExists = false
			domainStorageMap := accountStorageMap.GetDomain(nil, inter, domain, createIfNotExists)
			require.Equal(t, slices.Contains(existingDomains, domain), domainStorageMap != nil)

			if domainStorageMap != nil {
				checkDomainStorageMapData(t, inter, domainStorageMap, accountValues[domain])
			}
		}

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
	})
}

func TestAccountStorageMapCreateDomain(t *testing.T) {
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

		accountValues := make(accountStorageMapValues)

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, accountStorageMap)
		require.Equal(t, uint64(0), accountStorageMap.Count())

		accountStorageMapRootSlabID := accountStorageMap.SlabID()

		for _, domain := range common.AllStorageDomains {
			const createIfNotExists = true
			domainStorageMap := accountStorageMap.GetDomain(nil, inter, domain, createIfNotExists)
			require.NotNil(t, domainStorageMap)
			require.Equal(t, uint64(0), domainStorageMap.Count())

			accountValues[domain] = make(domainStorageMapValues)
		}

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMapRootSlabID})

		err := storage.PersistentSlabStorage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		checkAccountStorageMapDataWithRawData(t, ledger.StoredValues, ledger.StorageIndices, accountStorageMapRootSlabID, accountValues)
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

		existingDomains := []common.StorageDomain{common.PathDomainStorage.StorageDomain()}

		const count = 10
		accountStorageMap, accountValues := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

		accountStorageMapRootSlabID := accountStorageMap.SlabID()

		for _, domain := range common.AllStorageDomains {
			const createIfNotExists = true
			domainStorageMap := accountStorageMap.GetDomain(nil, inter, domain, createIfNotExists)
			require.NotNil(t, domainStorageMap)
			require.Equal(t, uint64(len(accountValues[domain])), domainStorageMap.Count())

			if !slices.Contains(existingDomains, domain) {
				accountValues[domain] = make(domainStorageMapValues)
			}
		}

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMapRootSlabID})

		err := storage.PersistentSlabStorage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		checkAccountStorageMapDataWithRawData(t, ledger.StoredValues, ledger.StorageIndices, accountStorageMapRootSlabID, accountValues)
	})
}

func TestAccountStorageMapSetAndUpdateDomain(t *testing.T) {
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

		accountValues := make(accountStorageMapValues)

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, accountStorageMap)
		require.Equal(t, uint64(0), accountStorageMap.Count())

		accountStorageMapRootSlabID := accountStorageMap.SlabID()

		const count = 10
		for _, domain := range common.AllStorageDomains {

			domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
			domainValues := writeRandomValuesToDomainStorageMap(inter, domainStorageMap, count, random)

			existed := accountStorageMap.WriteDomain(inter, domain, domainStorageMap)
			require.False(t, existed)

			accountValues[domain] = domainValues
		}

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMapRootSlabID})

		err := storage.PersistentSlabStorage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		checkAccountStorageMapDataWithRawData(t, ledger.StoredValues, ledger.StorageIndices, accountStorageMapRootSlabID, accountValues)
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

		existingDomains := []common.StorageDomain{common.PathDomainStorage.StorageDomain()}

		const count = 10
		accountStorageMap, accountValues := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

		accountStorageMapRootSlabID := accountStorageMap.SlabID()

		for _, domain := range common.AllStorageDomains {

			domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))
			domainValues := writeRandomValuesToDomainStorageMap(inter, domainStorageMap, count, random)

			existed := accountStorageMap.WriteDomain(inter, domain, domainStorageMap)
			require.Equal(t, slices.Contains(existingDomains, domain), existed)

			accountValues[domain] = domainValues
		}

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMapRootSlabID})

		err := storage.PersistentSlabStorage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		checkAccountStorageMapDataWithRawData(t, ledger.StoredValues, ledger.StorageIndices, accountStorageMapRootSlabID, accountValues)
	})
}

func TestAccountStorageMapRemoveDomain(t *testing.T) {
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

		accountValues := make(accountStorageMapValues)

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, accountStorageMap)
		require.Equal(t, uint64(0), accountStorageMap.Count())

		accountStorageMapRootSlabID := accountStorageMap.SlabID()

		for _, domain := range common.AllStorageDomains {
			existed := accountStorageMap.WriteDomain(inter, domain, nil)
			require.False(t, existed)
		}

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMapRootSlabID})

		err := storage.PersistentSlabStorage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		checkAccountStorageMapDataWithRawData(t, ledger.StoredValues, ledger.StorageIndices, accountStorageMapRootSlabID, accountValues)
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

		existingDomains := []common.StorageDomain{common.PathDomainStorage.StorageDomain()}

		const count = 10
		accountStorageMap, accountValues := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

		accountStorageMapRootSlabID := accountStorageMap.SlabID()

		for _, domain := range common.AllStorageDomains {

			existed := accountStorageMap.WriteDomain(inter, domain, nil)
			require.Equal(t, slices.Contains(existingDomains, domain), existed)

			delete(accountValues, domain)
		}

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMapRootSlabID})

		err := storage.PersistentSlabStorage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		checkAccountStorageMapDataWithRawData(t, ledger.StoredValues, ledger.StorageIndices, accountStorageMapRootSlabID, accountValues)
	})
}

func TestAccountStorageMapIterator(t *testing.T) {
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

		accountValues := make(accountStorageMapValues)

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, accountStorageMap)
		require.Equal(t, uint64(0), accountStorageMap.Count())

		iterator := accountStorageMap.Iterator()

		// Test calling Next() twice on empty account storage map.
		for range 2 {
			domain, domainStorageMap := iterator.Next()
			require.Empty(t, domain)
			require.Nil(t, domainStorageMap)
		}

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
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

		existingDomains := []common.StorageDomain{
			common.PathDomainStorage.StorageDomain(),
			common.PathDomainPublic.StorageDomain(),
		}

		const count = 10
		accountStorageMap, accountValues := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

		iterator := accountStorageMap.Iterator()

		domainCount := 0
		for {
			domain, domainStorageMap := iterator.Next()
			if domain == common.StorageDomainUnknown {
				break
			}

			domainCount++

			require.True(t, slices.Contains(existingDomains, domain))
			require.NotNil(t, domainStorageMap)

			checkDomainStorageMapData(t, inter, domainStorageMap, accountValues[domain])
		}

		// Test calling Next() after iterator reaches the end.
		domain, domainStorageMap := iterator.Next()
		require.Equal(t, common.StorageDomainUnknown, domain)
		require.Nil(t, domainStorageMap)

		require.Equal(t, len(existingDomains), domainCount)

		checkAccountStorageMapData(t, inter, accountStorageMap, accountValues)

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
	})
}

func TestAccountStorageMapDomains(t *testing.T) {
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

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
		require.NotNil(t, accountStorageMap)
		require.Equal(t, uint64(0), accountStorageMap.Count())

		domains := accountStorageMap.Domains()
		require.Equal(t, 0, len(domains))

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
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

		// Turn off automatic AtreeStorageValidationEnabled and explicitly check atree storage health directly.
		// This is because AccountStorageMap isn't created through storage, so there isn't any account register to match AccountStorageMap root slab.
		const atreeValueValidationEnabled = true
		const atreeStorageValidationEnabled = false
		inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(t, storage, atreeValueValidationEnabled, atreeStorageValidationEnabled)

		existingDomains := []common.StorageDomain{
			common.PathDomainStorage.StorageDomain(),
			common.PathDomainPublic.StorageDomain(),
			common.PathDomainPrivate.StorageDomain(),
		}

		const count = 10
		accountStorageMap, _ := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

		domains := accountStorageMap.Domains()
		require.Equal(t, len(existingDomains), len(domains))

		for _, domain := range existingDomains {
			_, exist := domains[domain]
			require.True(t, exist)
		}

		CheckAtreeStorageHealth(t, storage, []atree.SlabID{accountStorageMap.SlabID()})
	})
}

func TestAccountStorageMapLoadFromRootSlabID(t *testing.T) {
	t.Parallel()

	address := common.MustBytesToAddress([]byte{0x1})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		init := func() (atree.SlabID, accountStorageMapValues, map[string][]byte, map[string]uint64) {
			ledger := NewTestLedger(nil, nil)
			storage := runtime.NewStorage(
				ledger,
				nil,
				runtime.StorageConfig{},
			)

			inter := NewTestInterpreterWithStorage(t, storage)

			accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))
			require.NotNil(t, accountStorageMap)
			require.Equal(t, uint64(0), accountStorageMap.Count())

			err := storage.Commit(inter, false)
			require.NoError(t, err)

			return accountStorageMap.SlabID(), make(accountStorageMapValues), ledger.StoredValues, ledger.StorageIndices
		}

		accountStorageMapRootSlabID, accountValues, storedValues, storageIndices := init()

		checkAccountStorageMapDataWithRawData(t, storedValues, storageIndices, accountStorageMapRootSlabID, accountValues)
	})

	t.Run("non-empty", func(t *testing.T) {
		existingDomains := []common.StorageDomain{
			common.PathDomainStorage.StorageDomain(),
			common.PathDomainPublic.StorageDomain(),
			common.PathDomainPrivate.StorageDomain(),
		}

		init := func() (atree.SlabID, accountStorageMapValues, map[string][]byte, map[string]uint64) {
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
			accountStorageMap, accountValues := createAccountStorageMap(storage, inter, address, existingDomains, count, random)

			err := storage.Commit(inter, false)
			require.NoError(t, err)

			return accountStorageMap.SlabID(), accountValues, ledger.StoredValues, ledger.StorageIndices
		}

		accountStorageMapRootSlabID, accountValues, storedValues, storageIndices := init()

		checkAccountStorageMapDataWithRawData(t, storedValues, storageIndices, accountStorageMapRootSlabID, accountValues)
	})
}

type (
	domainStorageMapValues  map[interpreter.StorageMapKey]interpreter.Value
	accountStorageMapValues map[common.StorageDomain]domainStorageMapValues
)

func createAccountStorageMap(
	storage atree.SlabStorage,
	inter *interpreter.Interpreter,
	address common.Address,
	domains []common.StorageDomain,
	count int,
	random *rand.Rand,
) (*interpreter.AccountStorageMap, accountStorageMapValues) {

	// Create account storage map
	accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))

	accountValues := make(accountStorageMapValues)

	for _, domain := range domains {
		// Create domain storage map
		domainStorageMap := accountStorageMap.NewDomain(nil, inter, domain)

		// Write to new domain storage map
		domainValues := writeRandomValuesToDomainStorageMap(inter, domainStorageMap, count, random)

		accountValues[domain] = domainValues
	}

	return accountStorageMap, accountValues
}

func writeRandomValuesToDomainStorageMap(
	inter *interpreter.Interpreter,
	domainStorageMap *interpreter.DomainStorageMap,
	count int,
	random *rand.Rand,
) domainStorageMapValues {

	domainValues := make(domainStorageMapValues)

	for len(domainValues) < count {
		n := random.Int()

		key := interpreter.StringStorageMapKey(strconv.Itoa(n))

		var value interpreter.Value

		if len(domainValues) == 0 {
			// First element is a large value that is stored in its own slabs.
			value = interpreter.NewUnmeteredStringValue(strings.Repeat("a", 1_000))
		} else {
			value = interpreter.NewUnmeteredIntValueFromInt64(int64(n))
		}

		domainStorageMap.WriteValue(inter, key, value)

		domainValues[key] = value
	}

	return domainValues
}

// checkAccountStorageMapDataWithRawData checks loaded account storage map against expected account values.
func checkAccountStorageMapDataWithRawData(
	tb testing.TB,
	storedValues map[string][]byte,
	storageIndices map[string]uint64,
	rootSlabID atree.SlabID,
	expectedAccountValues accountStorageMapValues,
) {
	// Create new storage from raw data
	ledger := NewTestLedgerWithData(nil, nil, storedValues, storageIndices)
	storage := runtime.NewStorage(
		ledger,
		nil,
		runtime.StorageConfig{},
	)

	inter := NewTestInterpreterWithStorage(tb, storage)

	loadedAccountStorageMap := interpreter.NewAccountStorageMapWithRootID(storage, rootSlabID)
	require.Equal(tb, uint64(len(expectedAccountValues)), loadedAccountStorageMap.Count())
	require.Equal(tb, rootSlabID, loadedAccountStorageMap.SlabID())

	checkAccountStorageMapData(tb, inter, loadedAccountStorageMap, expectedAccountValues)

	CheckAtreeStorageHealth(tb, storage, []atree.SlabID{rootSlabID})
}

// checkAccountStorageMapData iterates account storage map and compares values with given expectedAccountValues.
func checkAccountStorageMapData(
	tb testing.TB,
	inter *interpreter.Interpreter,
	accountStorageMap *interpreter.AccountStorageMap,
	expectedAccountValues accountStorageMapValues,
) {
	require.Equal(tb, uint64(len(expectedAccountValues)), accountStorageMap.Count())

	domainCount := 0
	iter := accountStorageMap.Iterator()
	for {
		domain, domainStorageMap := iter.Next()
		if domain == common.StorageDomainUnknown {
			break
		}

		domainCount++

		expectedDomainValues, exist := expectedAccountValues[domain]
		require.True(tb, exist)

		checkDomainStorageMapData(tb, inter, domainStorageMap, expectedDomainValues)
	}

	require.Equal(tb, len(expectedAccountValues), domainCount)
}

// checkDomainStorageMapData iterates domain storage map and compares values with given expectedDomainValues.
func checkDomainStorageMapData(
	tb testing.TB,
	inter *interpreter.Interpreter,
	domainStorageMap *interpreter.DomainStorageMap,
	expectedDomainValues domainStorageMapValues,
) {
	require.Equal(tb, uint64(len(expectedDomainValues)), domainStorageMap.Count())

	count := 0
	iter := domainStorageMap.Iterator(nil)
	for {
		k, v := iter.Next()
		if k == nil {
			break
		}

		count++

		kv := k.(interpreter.StringAtreeValue)

		expectedValue := expectedDomainValues[interpreter.StringStorageMapKey(kv)]

		checkCadenceValue(tb, inter, v, expectedValue)
	}

	require.Equal(tb, len(expectedDomainValues), count)
}

func checkCadenceValue(
	tb testing.TB,
	inter *interpreter.Interpreter,
	value,
	expectedValue interpreter.Value,
) {
	ev, ok := value.(interpreter.EquatableValue)
	require.True(tb, ok)
	require.True(tb, ev.Equal(inter, interpreter.EmptyLocationRange, expectedValue))
}
