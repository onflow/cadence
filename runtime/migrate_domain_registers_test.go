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

package runtime_test

import (
	"math"
	"math/rand"
	goruntime "runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/runtime"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestMigrateDomainRegisters(t *testing.T) {
	t.Parallel()

	isAtreeRegister := func(key string) bool {
		return key[0] == '$' && len(key) == 9
	}

	getNonAtreeRegisters := func(values map[string][]byte) map[string][]byte {
		nonAtreeRegisters := make(map[string][]byte)
		for k, v := range values {
			ks := strings.Split(k, "|")
			if !isAtreeRegister(ks[1]) && len(v) > 0 {
				nonAtreeRegisters[k] = v
			}
		}
		return nonAtreeRegisters
	}

	address1 := common.MustBytesToAddress([]byte{0x1})
	address2 := common.MustBytesToAddress([]byte{0x2})

	addresses := []common.Address{address2, address1}

	t.Run("accounts without domain registers", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{
				StorageFormatV2Enabled: true,
			},
		)

		inter := NewTestInterpreterWithStorage(t, storage)

		migrator := runtime.NewDomainRegisterMigration(
			ledger,
			storage,
			inter,
			nil,
			nil,
		)

		for _, address := range addresses {
			accountStorageMap, err := migrator.MigrateAccount(address)
			require.Nil(t, accountStorageMap)
			require.NoError(t, err)
		}

		err := storage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		require.Equal(t, 0, len(ledger.StoredValues))
	})

	t.Run("accounts with domain registers", func(t *testing.T) {
		t.Parallel()

		accountsInfo := []accountInfo{
			{
				address: address1,
				domains: []domainInfo{
					{domain: common.PathDomainStorage.StorageDomain(), domainStorageMapCount: 10, maxDepth: 3},
					{domain: common.PathDomainPrivate.StorageDomain(), domainStorageMapCount: 10, maxDepth: 3},
				},
			},
			{
				address: address2,
				domains: []domainInfo{
					{domain: common.PathDomainPublic.StorageDomain(), domainStorageMapCount: 10, maxDepth: 3},
				},
			},
		}

		ledger, accountsValues := newTestLedgerWithUnmigratedAccounts(t, nil, nil, accountsInfo)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{
				StorageFormatV2Enabled: true,
			},
		)

		inter := NewTestInterpreterWithStorage(t, storage)

		migrator := runtime.NewDomainRegisterMigration(
			ledger,
			storage,
			inter,
			nil,
			nil,
		)

		var accountStorageMaps []*interpreter.AccountStorageMap
		for _, address := range addresses {
			accountStorageMap, err := migrator.MigrateAccount(address)
			require.NotNil(t, accountStorageMap)
			require.NoError(t, err)
			accountStorageMaps = append(accountStorageMaps, accountStorageMap)
		}

		err := storage.FastCommit(goruntime.NumCPU())
		require.NoError(t, err)

		// Check non-atree registers
		nonAtreeRegisters := getNonAtreeRegisters(ledger.StoredValues)
		require.Equal(t, len(addresses), len(nonAtreeRegisters))
		require.Contains(t, nonAtreeRegisters, string(address1[:])+"|"+runtime.AccountStorageKey)
		require.Contains(t, nonAtreeRegisters, string(address2[:])+"|"+runtime.AccountStorageKey)

		// Check atree storage
		expectedRootSlabIDs := make([]atree.SlabID, 0, len(accountStorageMaps))
		for _, accountStorageMap := range accountStorageMaps {
			expectedRootSlabIDs = append(expectedRootSlabIDs, accountStorageMap.SlabID())
		}

		CheckAtreeStorageHealth(t, storage, expectedRootSlabIDs)

		// Check account storage map data
		for address, accountValues := range accountsValues {
			checkAccountStorageMapData(t, ledger.StoredValues, ledger.StorageIndices, address, accountValues)
		}
	})

	t.Run("migrated accounts", func(t *testing.T) {
		t.Parallel()

		accountsInfo := []accountInfo{
			{
				address: address1,
				domains: []domainInfo{
					{domain: common.PathDomainStorage.StorageDomain(), domainStorageMapCount: 10, maxDepth: 3},
					{domain: common.PathDomainPrivate.StorageDomain(), domainStorageMapCount: 10, maxDepth: 3},
				},
			},
			{
				address: address2,
				domains: []domainInfo{
					{domain: common.PathDomainPublic.StorageDomain(), domainStorageMapCount: 10, maxDepth: 3},
				},
			},
		}

		ledger, accountsValues := newTestLedgerWithMigratedAccounts(t, nil, nil, accountsInfo)
		storage := runtime.NewStorage(
			ledger,
			nil,
			runtime.StorageConfig{
				StorageFormatV2Enabled: true,
			},
		)

		inter := NewTestInterpreterWithStorage(t, storage)

		migrator := runtime.NewDomainRegisterMigration(
			ledger,
			storage,
			inter,
			nil,
			nil,
		)

		for _, address := range addresses {
			accountStorageMap, err := migrator.MigrateAccount(address)
			require.Nil(t, accountStorageMap)
			require.NoError(t, err)
		}

		// Check account storage map data
		for address, accountValues := range accountsValues {
			checkAccountStorageMapData(
				t,
				ledger.StoredValues,
				ledger.StorageIndices,
				address,
				accountValues,
			)
		}
	})
}

type domainInfo struct {
	domain                common.StorageDomain
	domainStorageMapCount int
	maxDepth              int
}

type accountInfo struct {
	address common.Address
	domains []domainInfo
}

func newTestLedgerWithUnmigratedAccounts(
	tb testing.TB,
	onRead LedgerOnRead,
	onWrite LedgerOnWrite,
	accounts []accountInfo,
) (TestLedger, map[common.Address]accountStorageMapValues) {
	ledger := NewTestLedger(nil, nil)
	storage := runtime.NewStorage(
		ledger,
		nil,
		runtime.StorageConfig{
			StorageFormatV2Enabled: true,
		},
	)

	// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
	// This is because DomainStorageMap isn't created through runtime.Storage, so there isn't any
	// domain register to match DomainStorageMap root slab.
	const atreeValueValidationEnabled = true
	const atreeStorageValidationEnabled = false
	inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
		tb,
		storage,
		atreeValueValidationEnabled,
		atreeStorageValidationEnabled,
	)

	random := rand.New(rand.NewSource(42))

	accountsValues := make(map[common.Address]accountStorageMapValues)

	var expectedDomainRootSlabIDs []atree.SlabID

	for _, account := range accounts {

		address := account.address

		accountValues := make(accountStorageMapValues)

		accountsValues[address] = accountValues

		for _, domainInfo := range account.domains {

			domain := domainInfo.domain
			domainStorageMapCount := domainInfo.domainStorageMapCount
			maxDepth := domainInfo.maxDepth

			accountValues[domain] = make(domainStorageMapValues)

			// Create domain storage map
			domainStorageMap := interpreter.NewDomainStorageMap(nil, storage, atree.Address(address))

			// Write domain register
			domainStorageMapValueID := domainStorageMap.ValueID()
			err := ledger.SetValue(address[:], []byte(domain.Identifier()), domainStorageMapValueID[8:])
			require.NoError(tb, err)

			vid := domainStorageMap.ValueID()
			expectedDomainRootSlabIDs = append(
				expectedDomainRootSlabIDs,
				atree.NewSlabID(atree.Address(address), atree.SlabIndex(vid[8:])))

			// Write elements to to domain storage map
			for len(accountValues[domain]) < domainStorageMapCount {

				key := interpreter.StringStorageMapKey(strconv.Itoa(random.Int()))

				depth := random.Intn(maxDepth + 1)
				value := randomCadenceValues(inter, address, depth, random)

				_ = domainStorageMap.WriteValue(inter, key, value)

				accountValues[domain][key] = value
			}
		}
	}

	// Commit changes
	const commitContractUpdates = false
	err := storage.Commit(inter, commitContractUpdates)
	require.NoError(tb, err)

	CheckAtreeStorageHealth(tb, storage, expectedDomainRootSlabIDs)

	// Create a new storage
	newLedger := NewTestLedgerWithData(onRead, onWrite, ledger.StoredValues, ledger.StorageIndices)

	return newLedger, accountsValues
}

func newTestLedgerWithMigratedAccounts(
	tb testing.TB,
	onRead LedgerOnRead,
	onWrite LedgerOnWrite,
	accounts []accountInfo,
) (TestLedger, map[common.Address]accountStorageMapValues) {
	ledger := NewTestLedger(nil, nil)
	storage := runtime.NewStorage(
		ledger,
		nil,
		runtime.StorageConfig{
			StorageFormatV2Enabled: true,
		},
	)

	// Turn off AtreeStorageValidationEnabled and explicitly check atree storage health at the end of test.
	// This is because DomainStorageMap isn't created through runtime.Storage, so there isn't any
	// domain register to match DomainStorageMap root slab.
	const atreeValueValidationEnabled = true
	const atreeStorageValidationEnabled = false
	inter := NewTestInterpreterWithStorageAndAtreeValidationConfig(
		tb,
		storage,
		atreeValueValidationEnabled,
		atreeStorageValidationEnabled,
	)

	random := rand.New(rand.NewSource(42))

	expectedRootSlabIDs := make([]atree.SlabID, 0, len(accounts))

	accountsValues := make(map[common.Address]accountStorageMapValues)

	for _, account := range accounts {

		address := account.address

		accountValues := make(accountStorageMapValues)

		accountsValues[address] = accountValues

		accountStorageMap := interpreter.NewAccountStorageMap(nil, storage, atree.Address(address))

		// Write account register
		accountStorageMapSlabIndex := accountStorageMap.SlabID().Index()
		err := ledger.SetValue(address[:], []byte(runtime.AccountStorageKey), accountStorageMapSlabIndex[:])
		require.NoError(tb, err)

		expectedRootSlabIDs = append(expectedRootSlabIDs, accountStorageMap.SlabID())

		for _, domainInfo := range account.domains {

			domain := domainInfo.domain
			domainStorageMapCount := domainInfo.domainStorageMapCount
			maxDepth := domainInfo.maxDepth

			accountValues[domain] = make(domainStorageMapValues)

			// Create domain storage map
			domainStorageMap := accountStorageMap.NewDomain(nil, inter, domain)

			// Write elements to to domain storage map
			for len(accountValues[domain]) < domainStorageMapCount {

				key := interpreter.StringStorageMapKey(strconv.Itoa(random.Int()))

				depth := random.Intn(maxDepth + 1)
				value := randomCadenceValues(inter, address, depth, random)

				_ = domainStorageMap.WriteValue(inter, key, value)

				accountValues[domain][key] = value
			}
		}
	}

	// Commit changes
	const commitContractUpdates = false
	err := storage.Commit(inter, commitContractUpdates)
	require.NoError(tb, err)

	CheckAtreeStorageHealth(tb, storage, expectedRootSlabIDs)

	newLedger := NewTestLedgerWithData(onRead, onWrite, ledger.StoredValues, ledger.StorageIndices)

	return newLedger, accountsValues
}

func randomCadenceValues(
	inter *interpreter.Interpreter,
	address common.Address,
	depth int,
	random *rand.Rand,
) interpreter.EquatableValue {
	var typeIndex int
	if depth == 0 {
		typeIndex = random.Intn(typeLargeString + 1)
	} else {
		typeIndex = random.Intn(maxType)
	}

	switch typeIndex {
	case typeUint8:
		num := random.Intn(math.MaxUint8 + 1)
		return interpreter.NewUnmeteredUInt8Value(uint8(num))

	case typeUint16:
		num := random.Intn(math.MaxUint16 + 1)
		return interpreter.NewUnmeteredUInt16Value(uint16(num))

	case typeUint32:
		num := random.Uint32()
		return interpreter.NewUnmeteredUInt32Value(num)

	case typeUint64:
		num := random.Uint64()
		return interpreter.NewUnmeteredUInt64Value(num)

	case typeSmallString:
		const maxSmallStringLength = 32

		size := random.Intn(maxSmallStringLength + 1)

		b := make([]byte, size)
		random.Read(b)
		s := strings.ToValidUTF8(string(b), "$")
		return interpreter.NewUnmeteredStringValue(s)

	case typeLargeString:
		const minLargeStringLength = 256
		const maxLargeStringLength = 1024

		size := random.Intn(maxLargeStringLength+1-minLargeStringLength) + minLargeStringLength

		b := make([]byte, size)
		random.Read(b)
		s := strings.ToValidUTF8(string(b), "$")
		return interpreter.NewUnmeteredStringValue(s)

	case typeArray:
		const minArrayLength = 1
		const maxArrayLength = 20

		size := random.Intn(maxArrayLength+1-minArrayLength) + minArrayLength

		arrayType := interpreter.NewVariableSizedStaticType(
			nil,
			interpreter.PrimitiveStaticTypeAny,
		)

		depth--

		values := make([]interpreter.Value, size)
		for i := range size {
			values[i] = randomCadenceValues(inter, common.ZeroAddress, depth, random)
		}

		return interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			arrayType,
			address,
			values...,
		)

	case typeDictionary:
		const minDictLength = 1
		const maxDictLength = 20

		size := random.Intn(maxDictLength+1-minDictLength) + minDictLength

		dictType := interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeAny,
			interpreter.PrimitiveStaticTypeAny,
		)

		depth--

		keyAndValues := make([]interpreter.Value, 0, size*2)
		for i := range size * 2 {
			if i%2 == 0 {
				// Key (0 depth for element)
				keyAndValues = append(keyAndValues, randomCadenceValues(inter, common.ZeroAddress, 0, random))
			} else {
				// Value (decremented depth for element)
				keyAndValues = append(keyAndValues, randomCadenceValues(inter, common.ZeroAddress, depth, random))
			}
		}

		return interpreter.NewDictionaryValueWithAddress(inter, interpreter.EmptyLocationRange, dictType, address, keyAndValues...)

	default:
		panic(errors.NewUnreachableError())
	}
}

const (
	typeUint8 = iota
	typeUint16
	typeUint32
	typeUint64
	typeSmallString
	typeLargeString
	typeArray
	typeDictionary
	maxType
)
