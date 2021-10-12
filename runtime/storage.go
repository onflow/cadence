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
	"math"
	"runtime"
	"sort"
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type Storage struct {
	*atree.PersistentSlabStorage
	// NOTE: temporary, will be refactored to dictionary
	writes          map[interpreter.StorageKey]atree.Storable
	readCache       map[interpreter.StorageKey]atree.Storable
	contractUpdates map[interpreter.StorageKey]atree.Storable
	Ledger          atree.Ledger
	reportMetric    func(f func(), report func(metrics Metrics, duration time.Duration))
}

var _ atree.SlabStorage = &Storage{}
var _ interpreter.Storage = &Storage{}

func NewStorage(
	ledger atree.Ledger,
	reportMetric func(f func(), report func(metrics Metrics, duration time.Duration)),
) *Storage {
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
		writes:                map[interpreter.StorageKey]atree.Storable{},
		readCache:             map[interpreter.StorageKey]atree.Storable{},
		contractUpdates:       map[interpreter.StorageKey]atree.Storable{},
		reportMetric:          reportMetric,
	}
}

// ValueExists returns true if a value exists in account storage.
//
func (s *Storage) ValueExists(
	_ *interpreter.Interpreter,
	address common.Address,
	key string,
) bool {

	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	// Check locally

	storable, ok := s.writes[storageKey]
	if !ok {
		// Fall back to read cache
		storable, ok = s.readCache[storageKey]
	}
	if ok {
		return storable != nil
	}

	// Ask interface

	var exists bool
	var err error
	wrapPanic(func() {
		exists, err = s.Ledger.ValueExists(address[:], []byte(key))
	})
	if err != nil {
		panic(err)
	}

	if !exists {
		s.readCache[storageKey] = nil
	}

	return exists
}

// ReadValue returns a value from account storage.
//
func (s *Storage) ReadValue(
	_ *interpreter.Interpreter,
	address common.Address,
	key string,
) interpreter.OptionalValue {

	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	storable := s.readStorable(storageKey)
	if storable == nil {
		return interpreter.NilValue{}
	} else {
		storedValue := interpreter.StoredValue(storable, s)
		return interpreter.NewSomeValueNonCopying(storedValue)
	}
}

func (s *Storage) readStorable(storageKey interpreter.StorageKey) atree.Storable {

	// Check locally

	localStorable, ok := s.writes[storageKey]
	if !ok {
		// Fall back to read cache
		localStorable, ok = s.readCache[storageKey]
	}
	if ok {
		return localStorable
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

	var readStorable atree.Storable

	decoder := interpreter.CBORDecMode.NewByteStreamDecoder(storedData)

	s.reportMetric(
		func() {
			readStorable, err = interpreter.DecodeStorable(decoder, atree.StorageIDUndefined)
		},
		func(metrics Metrics, duration time.Duration) {
			metrics.ValueDecoded(duration)
		},
	)
	if err != nil {
		panic(err)
	}

	s.readCache[storageKey] = readStorable

	return readStorable
}

func (s *Storage) WriteValue(
	inter *interpreter.Interpreter,
	address common.Address,
	key string,
	value interpreter.OptionalValue,
) {
	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	// Remove existing value, if any

	existingStorable := s.readStorable(storageKey)
	if existingStorable != nil {
		interpreter.StoredValue(existingStorable, s).
			DeepRemove(inter)
		inter.RemoveReferencedSlab(existingStorable)
	}

	// Get storable representation for new value

	var newStorable atree.Storable

	switch typedValue := value.(type) {
	case *interpreter.SomeValue:
		var err error
		newStorable, err = typedValue.Value.Storable(
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

	case interpreter.NilValue:
		break

	default:
		panic(errors.NewUnreachableError())
	}

	// Only write locally.
	// The value is eventually written back through the runtime interface in `Commit`.

	s.writes[storageKey] = newStorable
}

func (s *Storage) recordContractUpdate(
	inter *interpreter.Interpreter,
	address common.Address,
	key string,
	contract interpreter.Value,
) {
	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	// Remove existing, if any

	existingStorable, ok := s.contractUpdates[storageKey]
	if ok {
		interpreter.StoredValue(existingStorable, s).
			DeepRemove(inter)
		inter.RemoveReferencedSlab(existingStorable)
	}

	if contract == nil {
		// NOTE: do NOT delete the map entry,
		// otherwise the write is lost
		s.contractUpdates[storageKey] = nil
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

		s.contractUpdates[storageKey] = storable
	}
}

type AccountStorageEntry struct {
	StorageKey       interpreter.StorageKey
	Storable         atree.Storable
	IsContractUpdate bool
}

// TODO: bring back concurrent encoding
// Commit serializes/saves all values in the readCache in storage (through the runtime interface).
//
func (s *Storage) Commit(inter *interpreter.Interpreter, commitContractUpdates bool) error {

	var accountStorageEntries []AccountStorageEntry

	// NOTE: ranging over maps is safe (deterministic),
	// if it is side-effect free and the keys are sorted afterwards

	// First, write all values in the account storage

	for storageKey, storable := range s.writes { //nolint:maprangecheck
		accountStorageEntries = append(
			accountStorageEntries,
			AccountStorageEntry{
				StorageKey: storageKey,
				Storable:   storable,
			},
		)
	}

	// Second, if enabled,
	// write all contract updates (which were delayed and not observable)

	if commitContractUpdates {
		for storageKey, storable := range s.contractUpdates { //nolint:maprangecheck
			accountStorageEntries = append(
				accountStorageEntries,
				AccountStorageEntry{
					StorageKey:       storageKey,
					Storable:         storable,
					IsContractUpdate: true,
				},
			)
		}
	}

	// Sort the account storage entries by storage key in lexicographic order

	SortAccountStorageEntries(accountStorageEntries)

	// Write account storage entries in order

	// TODO: bring back concurrent encoding
	for _, entry := range accountStorageEntries {

		storageKey := entry.StorageKey
		storable := entry.Storable

		address := storageKey.Address

		// If the account storage change is a contract update,
		// and it is overwriting an existing contract value,
		// it must be properly removed first:
		// The removal did not occur during execution,
		// because contract updates are deferred to the commit

		if entry.IsContractUpdate {
			existingStorable := s.readStorable(storageKey)
			if existingStorable != nil {
				interpreter.StoredValue(existingStorable, s).
					DeepRemove(inter)
				inter.RemoveReferencedSlab(existingStorable)
			}
		}

		var encoded []byte

		if storable != nil {
			var err error

			var buf bytes.Buffer
			encoder := atree.NewEncoder(&buf, interpreter.CBOREncMode)

			s.reportMetric(
				func() {
					err = storable.Encode(encoder)
				},
				func(metrics Metrics, duration time.Duration) {
					metrics.ValueEncoded(duration)
				},
			)
			if err != nil {
				return err
			}

			err = encoder.CBOR.Flush()
			if err != nil {
				return err
			}

			encoded = buf.Bytes()
		}

		var err error
		wrapPanic(func() {
			err = s.Ledger.SetValue(
				address[:],
				[]byte(storageKey.Key),
				encoded,
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

func SortAccountStorageEntries(entries []AccountStorageEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a := entries[i].StorageKey
		b := entries[j].StorageKey

		switch bytes.Compare(a.Address[:], b.Address[:]) {
		case -1:
			return true
		case 0:
			return a.Key < b.Key
		case 1:
			return false
		default:
			panic(errors.NewUnreachableError())
		}
	})
}
