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

package runtime

import (
	"fmt"
	"runtime"
	"sort"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

const StorageDomainContract = "contract"

type Storage struct {
	*atree.PersistentSlabStorage
	writes          map[interpreter.StorageKey]atree.StorageIndex
	storageMaps     map[interpreter.StorageKey]*interpreter.StorageMap
	contractUpdates map[interpreter.StorageKey]*interpreter.CompositeValue
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
		writes:                map[interpreter.StorageKey]atree.StorageIndex{},
		storageMaps:           map[interpreter.StorageKey]*interpreter.StorageMap{},
		contractUpdates:       map[interpreter.StorageKey]*interpreter.CompositeValue{},
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
			panic(fmt.Errorf(
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

	s.writes[storageKey] = storageIndex

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

	s.contractUpdates[key] = contractValue
}

type ContractUpdate struct {
	Key           interpreter.StorageKey
	ContractValue *interpreter.CompositeValue
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
//
func (s *Storage) commitContractUpdates(inter *interpreter.Interpreter) {

	contractUpdateCount := len(s.contractUpdates)

	if contractUpdateCount <= 1 {
		// NOTE: ranging over maps is safe (deterministic),
		// if the loop breaks after the first element (if any)

		for key, contractValue := range s.contractUpdates { //nolint:maprangecheck
			s.writeContractUpdate(inter, key, contractValue)
			break
		}
	} else {

		contractUpdates := make([]ContractUpdate, 0, contractUpdateCount)

		// NOTE: ranging over maps is safe (deterministic),
		// if it is side effect free and the keys are sorted afterwards

		for key, contractValue := range s.contractUpdates { //nolint:maprangecheck
			contractUpdates = append(
				contractUpdates,
				ContractUpdate{
					Key:           key,
					ContractValue: contractValue,
				},
			)
		}

		// Sort the contract updates by key in lexicographic order

		SortContractUpdates(contractUpdates)

		// Perform contract updates in order

		for _, contractUpdate := range contractUpdates {
			s.writeContractUpdate(inter, contractUpdate.Key, contractUpdate.ContractValue)
		}
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

type write struct {
	storageKey   interpreter.StorageKey
	storageIndex atree.StorageIndex
}

func sortWrites(writes []write) {
	sort.Slice(writes, func(i, j int) bool {
		a := writes[i].storageKey
		b := writes[j].storageKey
		return a.IsLess(b)
	})
}

// Commit serializes/saves all values in the readCache in storage (through the runtime interface).
//
func (s *Storage) Commit(inter *interpreter.Interpreter, commitContractUpdates bool) error {

	if commitContractUpdates {
		s.commitContractUpdates(inter)
	}

	var writes []write

	writeCount := len(s.writes)
	if writeCount > 0 {
		writes = make([]write, 0, writeCount)
	}

	// NOTE: ranging over maps is safe (deterministic),
	// if it is side effect free and the keys are sorted afterwards

	for storageKey, storageIndex := range s.writes { //nolint:maprangecheck
		writes = append(
			writes,
			write{
				storageKey:   storageKey,
				storageIndex: storageIndex,
			},
		)
	}

	// Sort the writes by storage key in lexicographic order
	if writeCount > 1 {
		sortWrites(writes)
	}

	// Write account storage entries in order

	// NOTE: Important: do not use a for-range loop,
	// as the introduced variable will be overridden on each loop iteration,
	// leading to the slices created in the loop body being backed by the same data
	for i := 0; i < len(writes); i++ {
		write := writes[i]

		var err error
		wrapPanic(func() {
			err = s.Ledger.SetValue(
				write.storageKey.Address[:],
				[]byte(write.storageKey.Key),
				write.storageIndex[:],
			)
		})
		if err != nil {
			return err
		}

		delete(s.writes, write.storageKey)
	}

	// Commit the underlying slab storage's writes

	// TODO: report encoding metric for all encoded slabs
	return s.PersistentSlabStorage.FastCommit(runtime.NumCPU())
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
	for rootSlabID := range rootSlabIDs { //nolint:maprangecheck
		if rootSlabID.Address == (atree.Address{}) {
			continue
		}

		accountRootSlabIDs[rootSlabID] = struct{}{}
	}

	// Check that each storage map refers to an existing slab.

	found := map[atree.StorageID]struct{}{}

	var storageMapStorageIDs []atree.StorageID

	for _, storageMap := range s.storageMaps { //nolint:maprangecheck
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
			return fmt.Errorf("account storage map points to non-existing slab %s", storageMapStorageID)
		}

		found[storageMapStorageID] = struct{}{}
	}

	// Check that all slabs in slab storage
	// are referenced by storables in account storage.
	// If a slab is not referenced, it is garbage.

	if len(accountRootSlabIDs) > len(found) {
		var unreferencedRootSlabIDs []atree.StorageID

		for accountRootSlabID := range accountRootSlabIDs { //nolint:maprangecheck
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

		return fmt.Errorf("slabs not referenced from account storage: %s", unreferencedRootSlabIDs)
	}

	return nil
}
