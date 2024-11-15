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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type AccountStorageV2 struct {
	ledger      atree.Ledger
	slabStorage atree.SlabStorage
	memoryGauge common.MemoryGauge

	// cachedAccountStorageMaps is a cache of account storage maps.
	cachedAccountStorageMaps map[common.Address]*interpreter.AccountStorageMap

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

func (s *AccountStorageV2) cacheAccountStorageMap(
	address common.Address,
	accountStorageMap *interpreter.AccountStorageMap,
) {
	if s.cachedAccountStorageMaps == nil {
		s.cachedAccountStorageMaps = map[common.Address]*interpreter.AccountStorageMap{}
	}
	s.cachedAccountStorageMaps[address] = accountStorageMap
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
		address,
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

	for pair := s.newAccountStorageMapSlabIndices.Oldest(); pair != nil; pair = pair.Next() {
		var err error
		errors.WrapPanic(func() {
			err = s.ledger.SetValue(
				pair.Key.Address[:],
				[]byte(pair.Key.Key),
				pair.Value[:],
			)
		})
		if err != nil {
			return interpreter.WrappedExternalError(err)
		}
	}

	return nil
}

func getAccountStorageSlabIndex(
	ledger atree.Ledger,
	address common.Address,
) (
	atree.SlabIndex,
	bool,
	error,
) {
	return getSlabIndexFromRegisterValue(
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
	slabIndex, registerExists, err := getAccountStorageSlabIndex(
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

func hasAccountStorageMap(
	ledger atree.Ledger,
	address common.Address,
) (bool, error) {

	_, registerExists, err := getAccountStorageSlabIndex(
		ledger,
		address,
	)
	if err != nil {
		return false, err
	}
	return registerExists, nil
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
