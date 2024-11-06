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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/interpreter"
)

type AccountStorageV2 struct {
	ledger      atree.Ledger
	slabStorage atree.SlabStorage
	memoryGauge common.MemoryGauge

	// cachedAccountStorageMaps is a cache of account storage maps.
	// Key is StorageKey{address, accountStorageKey} and value is account storage map.
	cachedAccountStorageMaps map[interpreter.StorageKey]*interpreter.AccountStorageMap

	// newAccountStorageMapSlabIndices contains root slab index of new account storage maps.
	// The indices are saved using Ledger.SetValue() during Commit().
	// Key is StorageKey{address, accountStorageKey} and value is 8-byte slab index.
	newAccountStorageMapSlabIndices *orderedmap.OrderedMap[interpreter.StorageKey, atree.SlabIndex]
}

func NewAccountStorageV2(
	ledger atree.Ledger,
	slabStorage atree.SlabStorage,
	memoryGauge common.MemoryGauge,
) *AccountStorageV2 {
	return &AccountStorageV2{
		ledger:      ledger,
		slabStorage: slabStorage,
		memoryGauge: memoryGauge,
	}
}

func (s *AccountStorageV2) accountStorageKey(address common.Address) interpreter.StorageKey {
	return interpreter.NewStorageKey(
		s.memoryGauge,
		address,
		AccountStorageKey,
	)
}

func (s *AccountStorageV2) GetDomainStorageMap(
	inter *interpreter.Interpreter,
	address common.Address,
	domain string,
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
			inter,
			domain,
			createIfNotExists,
		)
	}

	return
}

// getAccountStorageMap returns AccountStorageMap if exists, or nil otherwise.
func (s *AccountStorageV2) getAccountStorageMap(
	address common.Address,
) (
	accountStorageMap *interpreter.AccountStorageMap,
) {
	accountStorageKey := s.accountStorageKey(address)

	// Return cached account storage map if it exists.

	if s.cachedAccountStorageMaps != nil {
		accountStorageMap = s.cachedAccountStorageMaps[accountStorageKey]
		if accountStorageMap != nil {
			return accountStorageMap
		}
	}

	defer func() {
		if accountStorageMap != nil {
			s.cacheAccountStorageMap(
				accountStorageKey,
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

func (s *AccountStorageV2) cacheAccountStorageMap(
	accountStorageKey interpreter.StorageKey,
	accountStorageMap *interpreter.AccountStorageMap,
) {
	if s.cachedAccountStorageMaps == nil {
		s.cachedAccountStorageMaps = map[interpreter.StorageKey]*interpreter.AccountStorageMap{}
	}
	s.cachedAccountStorageMaps[accountStorageKey] = accountStorageMap
}

func (s *AccountStorageV2) storeNewAccountStorageMap(
	address common.Address,
) *interpreter.AccountStorageMap {

	accountStorageMap := interpreter.NewAccountStorageMap(
		s.memoryGauge,
		s.slabStorage,
		atree.Address(address),
	)

	slabIndex := accountStorageMap.SlabID().Index()

	accountStorageKey := s.accountStorageKey(address)

	s.SetNewAccountStorageMapSlabIndex(accountStorageKey, slabIndex)

	s.cacheAccountStorageMap(
		accountStorageKey,
		accountStorageMap,
	)

	return accountStorageMap
}

func (s *AccountStorageV2) SetNewAccountStorageMapSlabIndex(
	accountStorageKey interpreter.StorageKey,
	slabIndex atree.SlabIndex,
) {
	if s.newAccountStorageMapSlabIndices == nil {
		s.newAccountStorageMapSlabIndices = &orderedmap.OrderedMap[interpreter.StorageKey, atree.SlabIndex]{}
	}
	s.newAccountStorageMapSlabIndices.Set(accountStorageKey, slabIndex)
}

func (s *AccountStorageV2) commit() error {
	if s.newAccountStorageMapSlabIndices == nil {
		return nil
	}

	return commitSlabIndices(
		s.newAccountStorageMapSlabIndices,
		s.ledger,
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
	accountStorageSlabIndex, accountStorageRegisterExists, err := getSlabIndexFromRegisterValue(
		ledger,
		address,
		[]byte(AccountStorageKey),
	)
	if err != nil {
		return nil, err
	}
	if !accountStorageRegisterExists {
		return nil, nil
	}

	slabID := atree.NewSlabID(
		atree.Address(address),
		accountStorageSlabIndex,
	)

	return interpreter.NewAccountStorageMapWithRootID(slabStorage, slabID), nil
}

func (s *AccountStorageV2) cachedRootSlabIDs() []atree.SlabID {

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
