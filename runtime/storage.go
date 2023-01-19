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

package runtime

import (
	"runtime"
	"sort"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

const StorageDomainContract = "contract"

type Storage struct {
	*atree.PersistentSlabStorage
	newStorageMaps  *orderedmap.OrderedMap[interpreter.StorageKey, atree.StorageIndex]
	storageMaps     map[interpreter.StorageKey]*interpreter.StorageMap
	contractUpdates *orderedmap.OrderedMap[interpreter.StorageKey, *interpreter.CompositeValue]
	Ledger          atree.Ledger
	memoryGauge     common.MemoryGauge
}

var _ atree.SlabStorage = &Storage{}
var _ interpreter.Storage = &Storage{}

func NewStorage(ledger atree.Ledger, memoryGauge common.MemoryGauge) *Storage {
	decodeStorable := func(
		decoder *cbor.StreamDecoder,
		slabStorageID atree.StorageID,
	) (
		atree.Storable,
		error,
	) {
		return interpreter.DecodeStorable(
			decoder,
			slabStorageID,
			memoryGauge,
		)
	}

	decodeTypeInfo := func(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
		return interpreter.DecodeTypeInfo(decoder, memoryGauge)
	}

	ledgerStorage := atree.NewLedgerBaseStorage(ledger)
	persistentSlabStorage := atree.NewPersistentSlabStorage(
		ledgerStorage,
		interpreter.CBOREncMode,
		interpreter.CBORDecMode,
		decodeStorable,
		decodeTypeInfo,
	)
	return &Storage{
		Ledger:                ledger,
		PersistentSlabStorage: persistentSlabStorage,
		storageMaps:           map[interpreter.StorageKey]*interpreter.StorageMap{},
		memoryGauge:           memoryGauge,
	}
}

const storageIndexLength = 8

func (s *Storage) GetStorageMap(
	address common.Address,
	domain string,
	createIfNotExists bool,
) (
	storageMap *interpreter.StorageMap,
) {
	key := interpreter.NewStorageKey(s.memoryGauge, address, domain)

	storageMap = s.storageMaps[key]
	if storageMap == nil {

		// Load data through the runtime interface

		var data []byte
		var err error
		wrapPanic(func() {
			data, err = s.Ledger.GetValue(key.Address[:], []byte(key.Key))
		})
		if err != nil {
			panic(err)
		}

		dataLength := len(data)
		isStorageIndex := dataLength == storageIndexLength
		if dataLength > 0 && !isStorageIndex {
			// TODO: add dedicated error type?
			panic(errors.NewUnexpectedError(
				"invalid storage index for storage map with domain '%s': expected length %d, got %d",
				domain, storageIndexLength, dataLength,
			))
		}

		// Load existing storage or create and store new one

		atreeAddress := atree.Address(address)

		if isStorageIndex {
			var storageIndex atree.StorageIndex
			copy(storageIndex[:], data[:])
			storageMap = s.loadExistingStorageMap(atreeAddress, storageIndex)
		} else if createIfNotExists {
			storageMap = s.storeNewStorageMap(atreeAddress, domain)
		}

		if storageMap != nil {
			s.storageMaps[key] = storageMap
		}
	}

	return storageMap
}

func (s *Storage) loadExistingStorageMap(address atree.Address, storageIndex atree.StorageIndex) *interpreter.StorageMap {

	storageID := atree.StorageID{
		Address: address,
		Index:   storageIndex,
	}

	return interpreter.NewStorageMapWithRootID(s, storageID)
}

func (s *Storage) storeNewStorageMap(address atree.Address, domain string) *interpreter.StorageMap {
	storageMap := interpreter.NewStorageMap(s.memoryGauge, s, address)

	storageIndex := storageMap.StorageID().Index

	storageKey := interpreter.NewStorageKey(s.memoryGauge, common.Address(address), domain)

	if s.newStorageMaps == nil {
		s.newStorageMaps = &orderedmap.OrderedMap[interpreter.StorageKey, atree.StorageIndex]{}
	}
	s.newStorageMaps.Set(storageKey, storageIndex)

	return storageMap
}

func (s *Storage) recordContractUpdate(
	address common.Address,
	name string,
	contractValue *interpreter.CompositeValue,
) {
	key := interpreter.NewStorageKey(s.memoryGauge, address, name)

	// NOTE: do NOT delete the map entry,
	// otherwise the removal write is lost

	if s.contractUpdates == nil {
		s.contractUpdates = &orderedmap.OrderedMap[interpreter.StorageKey, *interpreter.CompositeValue]{}
	}
	s.contractUpdates.Set(key, contractValue)
}

type ContractUpdate struct {
	ContractValue *interpreter.CompositeValue
	Key           interpreter.StorageKey
}

func SortContractUpdates(updates []ContractUpdate) {
	sort.Slice(updates, func(i, j int) bool {
		a := updates[i].Key
		b := updates[j].Key
		return a.IsLess(b)
	})
}

// commitContractUpdates writes the contract updates to storage.
// The contract updates were delayed so they are not observable during execution.
func (s *Storage) commitContractUpdates(inter *interpreter.Interpreter) {
	if s.contractUpdates == nil {
		return
	}

	for pair := s.contractUpdates.Oldest(); pair != nil; pair = pair.Next() {
		s.writeContractUpdate(inter, pair.Key, pair.Value)
	}
}

func (s *Storage) writeContractUpdate(
	inter *interpreter.Interpreter,
	key interpreter.StorageKey,
	contractValue *interpreter.CompositeValue,
) {
	storageMap := s.GetStorageMap(key.Address, StorageDomainContract, true)
	// NOTE: pass nil instead of allocating a Value-typed  interface that points to nil
	if contractValue == nil {
		storageMap.WriteValue(inter, key.Key, nil)
	} else {
		storageMap.WriteValue(inter, key.Key, contractValue)
	}
}

// Commit serializes/saves all values in the readCache in storage (through the runtime interface).
func (s *Storage) Commit(inter *interpreter.Interpreter, commitContractUpdates bool) error {

	if commitContractUpdates {
		s.commitContractUpdates(inter)
	}

	err := s.commitNewStorageMaps()
	if err != nil {
		return err
	}

	// Commit the underlying slab storage's writes

	deltas := s.PersistentSlabStorage.DeltasWithoutTempAddresses()
	common.UseMemory(s.memoryGauge, common.NewAtreeEncodedSlabMemoryUsage(deltas))

	// TODO: report encoding metric for all encoded slabs
	return s.PersistentSlabStorage.FastCommit(runtime.NumCPU())
}

func (s *Storage) commitNewStorageMaps() error {
	if s.newStorageMaps == nil {
		return nil
	}

	for pair := s.newStorageMaps.Oldest(); pair != nil; pair = pair.Next() {
		var err error
		wrapPanic(func() {
			err = s.Ledger.SetValue(
				pair.Key.Address[:],
				[]byte(pair.Key.Key),
				pair.Value[:],
			)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) CheckHealth() error {
	// Check slab storage health
	rootSlabIDs, err := atree.CheckStorageHealth(s, -1)
	if err != nil {
		return err
	}

	// Find account / non-temporary root slab IDs

	accountRootSlabIDs := make(map[atree.StorageID]struct{}, len(rootSlabIDs))

	// NOTE: map range is safe, as it creates a subset
	for rootSlabID := range rootSlabIDs { //nolint:maprange
		if rootSlabID.Address == (atree.Address{}) {
			continue
		}

		accountRootSlabIDs[rootSlabID] = struct{}{}
	}

	// Check that each storage map refers to an existing slab.

	found := map[atree.StorageID]struct{}{}

	var storageMapStorageIDs []atree.StorageID

	for _, storageMap := range s.storageMaps { //nolint:maprange
		storageMapStorageIDs = append(
			storageMapStorageIDs,
			storageMap.StorageID(),
		)
	}

	sort.Slice(storageMapStorageIDs, func(i, j int) bool {
		a := storageMapStorageIDs[i]
		b := storageMapStorageIDs[j]
		return a.Compare(b) < 0
	})

	for _, storageMapStorageID := range storageMapStorageIDs {
		if _, ok := accountRootSlabIDs[storageMapStorageID]; !ok {
			return errors.NewUnexpectedError("account storage map points to non-existing slab %s", storageMapStorageID)
		}

		found[storageMapStorageID] = struct{}{}
	}

	// Check that all slabs in slab storage
	// are referenced by storables in account storage.
	// If a slab is not referenced, it is garbage.

	if len(accountRootSlabIDs) > len(found) {
		var unreferencedRootSlabIDs []atree.StorageID

		for accountRootSlabID := range accountRootSlabIDs { //nolint:maprange
			if _, ok := found[accountRootSlabID]; ok {
				continue
			}

			unreferencedRootSlabIDs = append(
				unreferencedRootSlabIDs,
				accountRootSlabID,
			)
		}

		sort.Slice(unreferencedRootSlabIDs, func(i, j int) bool {
			a := unreferencedRootSlabIDs[i]
			b := unreferencedRootSlabIDs[j]
			return a.Compare(b) < 0
		})

		return errors.NewUnexpectedError("slabs not referenced from account storage: %s", unreferencedRootSlabIDs)
	}

	return nil
}
