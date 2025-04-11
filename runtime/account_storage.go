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

package runtime

import (
	"sort"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type AccountStorage struct {
	ledger      atree.Ledger
	slabStorage atree.SlabStorage
	memoryGauge common.MemoryGauge

	// cachedAccountStorageMaps is a cache of account storage maps.
	cachedAccountStorageMaps map[common.Address]*interpreter.AccountStorageMap

	// newAccountStorageMapSlabIndices contains root slab indices of new account storage maps.
	// The indices are saved using Ledger.SetValue() during commit().
	newAccountStorageMapSlabIndices map[common.Address]atree.SlabIndex
}

func NewAccountStorage(
	ledger atree.Ledger,
	slabStorage atree.SlabStorage,
	memoryGauge common.MemoryGauge,
) *AccountStorage {
	return &AccountStorage{
		ledger:      ledger,
		slabStorage: slabStorage,
		memoryGauge: memoryGauge,
	}
}

func (s *AccountStorage) GetDomainStorageMap(
	storageMutationTracker interpreter.StorageMutationTracker,
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) (
	domainStorageMap *interpreter.DomainStorageMap,
) {
	accountStorageMap := s.getAccountStorageMap(address)

	if accountStorageMap == nil && createIfNotExists {
		accountStorageMap = s.storeNewAccountStorageMap(address)
	}

	if accountStorageMap != nil {
		domainStorageMap = accountStorageMap.GetDomain(
			s.memoryGauge,
			storageMutationTracker,
			domain,
			createIfNotExists,
		)
	}

	return
}

// getAccountStorageMap returns AccountStorageMap if exists, or nil otherwise.
func (s *AccountStorage) getAccountStorageMap(
	address common.Address,
) (
	accountStorageMap *interpreter.AccountStorageMap,
) {
	// Return cached account storage map if it exists.

	if s.cachedAccountStorageMaps != nil {
		accountStorageMap = s.cachedAccountStorageMaps[address]
		if accountStorageMap != nil {
			return accountStorageMap
		}
	}

	defer func() {
		if accountStorageMap != nil {
			s.cacheAccountStorageMap(
				address,
				accountStorageMap,
			)
		}
	}()

	// Load account storage map if account storage register exists.

	var err error
	accountStorageMap, err = getAccountStorageMapFromRegister(
		s.ledger,
		s.slabStorage,
		address,
	)
	if err != nil {
		panic(err)
	}

	return
}

func (s *AccountStorage) cacheAccountStorageMap(
	address common.Address,
	accountStorageMap *interpreter.AccountStorageMap,
) {
	if s.cachedAccountStorageMaps == nil {
		s.cachedAccountStorageMaps = map[common.Address]*interpreter.AccountStorageMap{}
	}
	s.cachedAccountStorageMaps[address] = accountStorageMap
}

func (s *AccountStorage) storeNewAccountStorageMap(
	address common.Address,
) *interpreter.AccountStorageMap {

	accountStorageMap := interpreter.NewAccountStorageMap(
		s.memoryGauge,
		s.slabStorage,
		atree.Address(address),
	)

	slabIndex := accountStorageMap.SlabID().Index()

	s.SetNewAccountStorageMapSlabIndex(
		address,
		slabIndex,
	)

	s.cacheAccountStorageMap(
		address,
		accountStorageMap,
	)

	return accountStorageMap
}

func (s *AccountStorage) SetNewAccountStorageMapSlabIndex(
	address common.Address,
	slabIndex atree.SlabIndex,
) {
	if s.newAccountStorageMapSlabIndices == nil {
		s.newAccountStorageMapSlabIndices = map[common.Address]atree.SlabIndex{}
	}
	s.newAccountStorageMapSlabIndices[address] = slabIndex
}

func (s *AccountStorage) commit() error {
	switch len(s.newAccountStorageMapSlabIndices) {
	case 0:
		// Nothing to commit.
		return nil

	case 1:
		// Optimize for the common case of a single account storage map.

		var updated int
		for address, slabIndex := range s.newAccountStorageMapSlabIndices { //nolint:maprange
			if updated > 0 {
				panic(errors.NewUnreachableError())
			}

			err := s.writeAccountStorageSlabIndex(
				address,
				slabIndex,
			)
			if err != nil {
				return err
			}

			updated++
		}

	default:
		// Sort the indices to ensure deterministic order

		type accountStorageMapSlabIndex struct {
			Address   common.Address
			SlabIndex atree.SlabIndex
		}

		slabIndices := make([]accountStorageMapSlabIndex, 0, len(s.newAccountStorageMapSlabIndices))
		for address, slabIndex := range s.newAccountStorageMapSlabIndices { //nolint:maprange
			slabIndices = append(
				slabIndices,
				accountStorageMapSlabIndex{
					Address:   address,
					SlabIndex: slabIndex,
				},
			)
		}
		sort.Slice(
			slabIndices,
			func(i, j int) bool {
				slabIndex1 := slabIndices[i]
				slabIndex2 := slabIndices[j]
				address1 := slabIndex1.Address
				address2 := slabIndex2.Address
				return address1.Compare(address2) < 0
			},
		)

		for _, slabIndex := range slabIndices {
			err := s.writeAccountStorageSlabIndex(
				slabIndex.Address,
				slabIndex.SlabIndex,
			)
			if err != nil {
				return err
			}
		}
	}

	s.newAccountStorageMapSlabIndices = nil

	return nil
}

func (s *AccountStorage) writeAccountStorageSlabIndex(
	address common.Address,
	slabIndex atree.SlabIndex,
) error {
	return writeSlabIndexToRegister(
		s.ledger,
		address,
		[]byte(AccountStorageKey),
		slabIndex,
	)
}

func readAccountStorageSlabIndexFromRegister(
	ledger atree.Ledger,
	address common.Address,
) (
	atree.SlabIndex,
	bool,
	error,
) {
	return readSlabIndexFromRegister(
		ledger,
		address,
		[]byte(AccountStorageKey),
	)
}

func getAccountStorageMapFromRegister(
	ledger atree.Ledger,
	slabStorage atree.SlabStorage,
	address common.Address,
) (
	*interpreter.AccountStorageMap,
	error,
) {
	slabIndex, registerExists, err := readAccountStorageSlabIndexFromRegister(
		ledger,
		address,
	)
	if err != nil {
		return nil, err
	}
	if !registerExists {
		return nil, nil
	}

	slabID := atree.NewSlabID(
		atree.Address(address),
		slabIndex,
	)

	return interpreter.NewAccountStorageMapWithRootID(slabStorage, slabID), nil
}

func (s *AccountStorage) cachedRootSlabIDs() []atree.SlabID {

	var slabIDs []atree.SlabID

	// Get cached account storage map slab IDs.
	for _, storageMap := range s.cachedAccountStorageMaps { //nolint:maprange
		slabIDs = append(
			slabIDs,
			storageMap.SlabID(),
		)
	}

	return slabIDs
}
