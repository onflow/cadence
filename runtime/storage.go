/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"bytes"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

const StorageDomainContract = "contract"

type StorageKey struct {
	Address common.Address
	Key     string
}

func (k StorageKey) IsLess(o StorageKey) bool {
	switch bytes.Compare(k.Address[:], o.Address[:]) {
	case -1:
		return true
	case 0:
		return strings.Compare(k.Key, o.Key) < 0
	case 1:
		return false
	default:
		panic(errors.NewUnreachableError())
	}
}

type Storage struct {
	*atree.PersistentSlabStorage
	writes          map[StorageKey][]byte
	readCache       map[StorageKey][]byte
	storageMaps     map[interpreter.StorageMapKey]*interpreter.StorageMap
	contractUpdates map[StorageKey]atree.Storable
	Ledger          atree.Ledger
}

var _ atree.SlabStorage = &Storage{}
var _ interpreter.Storage = &Storage{}

func NewStorage(ledger atree.Ledger) *Storage {
	ledgerStorage := atree.NewLedgerBaseStorage(ledger)
	persistentSlabStorage := atree.NewPersistentSlabStorage(
		ledgerStorage,
		interpreter.CBOREncMode,
		interpreter.CBORDecMode,
		interpreter.DecodeStorable,
		interpreter.DecodeTypeInfo,
	)
	return &Storage{
		Ledger:                ledger,
		PersistentSlabStorage: persistentSlabStorage,
		writes:                map[StorageKey][]byte{},
		readCache:             map[StorageKey][]byte{},
		storageMaps:           map[interpreter.StorageMapKey]*interpreter.StorageMap{},
		contractUpdates:       map[StorageKey]atree.Storable{},
	}
}

func (s *Storage) read(address common.Address, domain string) []byte {

	storageKey := StorageKey{
		Address: address,
		Key:     domain,
	}

	// Check locally

	localData, ok := s.writes[storageKey]
	if !ok {
		// Fall back to read cache
		localData, ok = s.readCache[storageKey]
	}
	if ok {
		return localData
	}

	// Load data through the runtime interface

	var storedData []byte
	var err error
	wrapPanic(func() {
		storedData, err = s.Ledger.GetValue(storageKey.Address[:], []byte(storageKey.Key))
	})
	if err != nil {
		panic(err)
	}

	// No data, keep fact in cache

	if len(storedData) == 0 {
		s.readCache[storageKey] = nil
		return nil
	}

	// Existing data, decode and keep in cache

	s.readCache[storageKey] = storedData

	return storedData
}

const storageIndexLength = 8

func (s *Storage) GetStorageMap(address common.Address, domain string) (storageMap *interpreter.StorageMap) {
	key := interpreter.StorageMapKey{
		Address: address,
		Domain:  domain,
	}

	storageMap = s.storageMaps[key]
	if storageMap == nil {

		data := s.read(address, domain)

		// Load existing storage or create and store new one

		atreeAddress := atree.Address(address)

		if len(data) > 0 {
			storageMap = s.loadExistingStorageMap(atreeAddress, domain, data)
		} else {
			storageMap = s.storeNewStorageMap(atreeAddress, domain)
		}

		s.storageMaps[key] = storageMap
	}

	return storageMap
}

func (s *Storage) loadExistingStorageMap(address atree.Address, domain string, data []byte) *interpreter.StorageMap {
	if len(data) != storageIndexLength {
		// TODO: add dedicated error type?
		panic(fmt.Errorf(
			"invalid storage index for storage map with domain '%s': expected length %d, got %d",
			domain, storageIndexLength, len(data),
		))
	}

	var storageIndex atree.StorageIndex
	copy(storageIndex[:], data)

	storageID := atree.StorageID{
		Address: address,
		Index:   storageIndex,
	}

	return interpreter.NewStorageMapWithRootID(s, storageID)
}

func (s *Storage) storeNewStorageMap(address atree.Address, domain string) *interpreter.StorageMap {
	storageMap := interpreter.NewStorageMap(s, address)

	storageIndex := storageMap.StorageID().Index

	storageKey := StorageKey{
		Address: common.Address(address),
		Key:     domain,
	}

	s.writes[storageKey] = storageIndex[:]

	return storageMap
}

func (s *Storage) recordContractUpdate(
	inter *interpreter.Interpreter,
	address common.Address,
	name string,
	contract interpreter.Value,
) {
	key := StorageKey{
		Address: address,
		Key:     name,
	}

	// Remove existing, if any

	existingStorable, ok := s.contractUpdates[key]
	if ok {
		interpreter.StoredValue(existingStorable, s).
			DeepRemove(inter)
		inter.RemoveReferencedSlab(existingStorable)
	}

	if contract == nil {
		// NOTE: do NOT delete the map entry,
		// otherwise the write is lost
		s.contractUpdates[key] = nil
	} else {
		storable, err := contract.Storable(
			s,
			atree.Address(address),
			// NOTE: we already allocate a register for the account storage value,
			// so we might as well store all data of the value in it, if possible,
			// e.g. for a large immutable value.
			//
			// Using a smaller number would only result in an additional register
			// (account storage register would have storage ID storable,
			// and extra slab / register would contain the actual data of the value).
			math.MaxUint64,
		)
		if err != nil {
			panic(err)
		}

		s.contractUpdates[key] = storable
	}
}

type ContractUpdate struct {
	Key      StorageKey
	Storable atree.Storable
}

func SortContractUpdates(updates []ContractUpdate) {
	sort.Slice(updates, func(i, j int) bool {
		a := updates[i].Key
		b := updates[j].Key
		return a.IsLess(b)
	})
}

func (s *Storage) commitContractUpdates(inter *interpreter.Interpreter) {

	var contractUpdates []ContractUpdate

	// NOTE: ranging over maps is safe (deterministic),
	// if it is side effect free and the keys are sorted afterwards

	// Collect contract updates (delayed and not observable during execution)

	for key, storable := range s.contractUpdates { //nolint:maprangecheck
		contractUpdates = append(
			contractUpdates,
			ContractUpdate{
				Key:      key,
				Storable: storable,
			},
		)
	}

	// Sort the contract updates by key in lexicographic order

	SortContractUpdates(contractUpdates)

	// Perform contract updates in order

	for _, contractUpdate := range contractUpdates {

		key := contractUpdate.Key

		storageMap := s.GetStorageMap(key.Address, StorageDomainContract)

		var value interpreter.OptionalValue

		if contractUpdate.Storable == nil {
			value = interpreter.NilValue{}
		} else {
			contractValue := interpreter.StoredValue(contractUpdate.Storable, s)
			value = interpreter.NewSomeValueNonCopying(contractValue)
		}

		storageMap.WriteValue(inter, key.Key, value)
	}
}

type write struct {
	storageKey StorageKey
	data       []byte
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

	// NOTE: ranging over maps is safe (deterministic),
	// if it is side effect free and the keys are sorted afterwards

	for storageKey, data := range s.writes { //nolint:maprangecheck
		writes = append(
			writes,
			write{
				storageKey: storageKey,
				data:       data,
			},
		)
	}

	// Sort the writes by storage key in lexicographic order

	sortWrites(writes)

	// Write account storage entries in order

	for _, write := range writes {

		var err error
		wrapPanic(func() {
			err = s.Ledger.SetValue(
				write.storageKey.Address[:],
				[]byte(write.storageKey.Key),
				write.data,
			)
		})
		if err != nil {
			return err
		}
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
