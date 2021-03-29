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
	"sort"
	"time"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

const ENCODING_NUM_WORKER = 10

type StorageKey struct {
	Address common.Address
	Key     string
}

type Cache map[StorageKey]CacheEntry

type CacheEntry struct {
	// true indicates that the value definitely must be written, independent of the value.
	// false indicates that the value may has to be written if the value is modified.
	MustWrite bool
	Value     interpreter.Value
}

type ContractUpdates map[StorageKey]ContractUpdate

type ContractUpdate struct {
	Contract         interpreter.Value
	ExportedContract cadence.Value
}

type runtimeStorage struct {
	runtimeInterface        Interface
	highLevelStorageEnabled bool
	highLevelStorage        HighLevelStorage
	cache                   Cache
	contractUpdates         ContractUpdates
}

func newRuntimeStorage(runtimeInterface Interface) *runtimeStorage {
	highLevelStorageEnabled := false
	highLevelStorage, ok := runtimeInterface.(HighLevelStorage)
	if ok {
		highLevelStorageEnabled = highLevelStorage.HighLevelStorageEnabled()
	}

	return &runtimeStorage{
		runtimeInterface:        runtimeInterface,
		cache:                   Cache{},
		contractUpdates:         ContractUpdates{},
		highLevelStorage:        highLevelStorage,
		highLevelStorageEnabled: highLevelStorageEnabled,
	}
}

// valueExists is the StorageExistenceHandlerFunc for the interpreter.
//
// It checks the cache for values which were already previously loaded/deserialized
// from storage (through the runtime interface) and returns true if the cached value exists.
//
// If there is a cache miss, the key is read from storage through the runtime interface,
// places in the cache, and returned.
//
func (s *runtimeStorage) valueExists(
	address common.Address,
	key string,
) bool {

	fullKey := StorageKey{
		Address: address,
		Key:     key,
	}

	// Check cache

	if entry, ok := s.cache[fullKey]; ok {
		return entry.Value != nil
	}

	// Cache miss: Ask interface

	var exists bool
	var err error
	wrapPanic(func() {
		exists, err = s.runtimeInterface.ValueExists(address[:], []byte(key))
	})
	if err != nil {
		panic(err)
	}

	if !exists {
		s.cache[fullKey] = CacheEntry{
			MustWrite: false,
			Value:     nil,
		}
	}

	return exists
}

// readValue is the StorageReadHandlerFunc for the interpreter.
//
// It checks the cache for values which were already previously loaded/deserialized
// from storage (through the runtime interface) and returns the cached value if it exists.
//
// If there is a cache miss, the key is read from storage through the runtime interface,
// places in the cache, and returned.
//
func (s *runtimeStorage) readValue(
	address common.Address,
	key string,
	deferred bool,
) interpreter.OptionalValue {

	fullKey := StorageKey{
		Address: address,
		Key:     key,
	}

	// Check cache. Return cached value, if any

	if entry, ok := s.cache[fullKey]; ok {
		if entry.Value == nil {
			return interpreter.NilValue{}
		}

		return interpreter.NewSomeValueOwningNonCopying(entry.Value)
	}

	// Cache miss: Load and deserialize the stored value (if any)
	// through the runtime interface

	var storedData []byte
	var err error
	wrapPanic(func() {
		storedData, err = s.runtimeInterface.GetValue(address[:], []byte(key))
	})
	if err != nil {
		panic(err)
	}

	var version uint16
	storedData, version = interpreter.StripMagic(storedData)

	if len(storedData) == 0 {
		s.cache[fullKey] = CacheEntry{
			MustWrite: false,
			Value:     nil,
		}
		return interpreter.NilValue{}
	}

	var storedValue interpreter.Value

	reportMetric(
		func() {
			storedValue, err = interpreter.DecodeValue(
				storedData,
				&address,
				[]string{key},
				version,
				nil,
			)
		},
		s.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ValueDecoded(duration)
		},
	)
	if err != nil {
		panic(err)
	}

	if !deferred {
		s.cache[fullKey] = CacheEntry{
			MustWrite: false,
			Value:     storedValue,
		}
	}

	return interpreter.NewSomeValueOwningNonCopying(storedValue)
}

// writeValue is the StorageWriteHandlerFunc for the interpreter.
//
// It only places the written value in the cache.
//
// It does *not* serialize/save the value in  storage (through the runtime interface).
// (The Cache is finally written back through the runtime interface in `writeCached`.)
//
func (s *runtimeStorage) writeValue(
	address common.Address,
	key string,
	value interpreter.OptionalValue,
) {
	fullKey := StorageKey{
		Address: address,
		Key:     key,
	}

	// Only write the value to the cache.
	// The Cache is finally written back through the runtime interface in `writeCached`

	entry := s.cache[fullKey]
	entry.MustWrite = true

	switch typedValue := value.(type) {
	case *interpreter.SomeValue:
		entry.Value = typedValue.Value

	case interpreter.NilValue:
		entry.Value = nil

	default:
		panic(errors.NewUnreachableError())
	}

	s.cache[fullKey] = entry
}

func (s *runtimeStorage) recordContractUpdate(
	address common.Address,
	key string,
	contract interpreter.Value,
	exportedContract cadence.Value,
) {
	fullKey := StorageKey{
		Address: address,
		Key:     key,
	}

	s.contractUpdates[fullKey] = ContractUpdate{
		Contract:         contract,
		ExportedContract: exportedContract,
	}
}

type writeItem struct {
	storageKey    StorageKey
	value         interpreter.Value
	exportedValue cadence.Value
}

// writeCached serializes/saves all values in the cache in storage (through the runtime interface).
//
func (s *runtimeStorage) writeCached(inter *interpreter.Interpreter) {

	var items []writeItem

	// First, iterate over the cache
	// and determine which items have to be written

	for fullKey, entry := range s.cache { //nolint:maprangecheck

		if !entry.MustWrite && entry.Value != nil && !entry.Value.IsModified() {
			continue
		}

		items = append(items, writeItem{
			storageKey: fullKey,
			value:      entry.Value,
		})
	}

	for fullKey, contractUpdate := range s.contractUpdates { //nolint:maprangecheck
		items = append(items, writeItem{
			storageKey:    fullKey,
			value:         contractUpdate.Contract,
			exportedValue: contractUpdate.ExportedContract,
		})
	}

	// Order the items by storage key in lexicographic order

	sort.Slice(items, func(i, j int) bool {
		a := items[i].storageKey
		b := items[j].storageKey

		if bytes.Compare(a.Address[:], b.Address[:]) < 0 {
			return true
		}

		if a.Key < b.Key {
			return true
		}

		return false
	})

	// Write cache entries in order

	if s.highLevelStorageEnabled {
		for _, item := range items {

			var value cadence.Value
			switch {
			case item.exportedValue != nil:
				value = item.exportedValue
			case item.value != nil:
				value = exportValueWithInterpreter(item.value, inter, exportResults{})
			}

			var err error
			wrapPanic(func() {
				err = s.highLevelStorage.SetCadenceValue(
					item.storageKey.Address,
					item.storageKey.Key,
					value,
				)
			})
			if err != nil {
				panic(err)
			}
		}
	}

	// run batch in a for loop, each batch will create a new batch
	// to be run again, until the batch is empty.
	batch := items
	for len(batch) > 0 {
		// parallelize the encoding for items within the same batch
		// the encoding has no side effect
		encodedResults, err := s.encodeWriteItems(ENCODING_NUM_WORKER, batch)
		if err != nil {
			panic(err)
		}
		var newBatch []writeItem
		// process encoded items of the batch, play side effect for
		// each item
		for i, result := range encodedResults {
			item := batch[i]
			newItems, err := s.processEncodedItem(item, result)
			if err != nil {
				panic(err)
			}
			newBatch = append(newBatch, newItems...)
		}
		batch = newBatch
	}
}

type encodedResult struct {
	newData   []byte
	deferrals *interpreter.EncodingDeferrals
	index     int // the index of the encoded item in the batch
	err       error
}

func (s *runtimeStorage) encodeWriteItems(nWorker int, batch []writeItem) ([]*encodedResult, error) {
	// cache all the encoded results, including errors
	results := make(chan *encodedResult, len(batch))
	defer close(results)

	// if the number of items in a batch is less than the default number of workers, we will only create the
	// same amount of workers as the items
	if len(batch) < nWorker {
		nWorker = len(batch)
	}

	// jobs buffers at most nWorker number of jobs for workers to work on in parallel
	// each job contains the index of the item in the batch
	jobs := make(chan int, nWorker)
	defer close(jobs)

	worker := func(jobs <-chan int, results chan<- *encodedResult) {
		for i := range jobs {
			item := batch[i]
			if item.value == nil {
				results <- nil
			} else {
				// TODO: encode value with a context, so that if there 10 jobs to work on,
				// and one job failed, we could cancel the context and skip the rest of
				// unprocssed jobs without processing them.
				newData, deferrals, err := s.encodeValue(item.value, item.storageKey.Key)
				results <- &encodedResult{
					newData:   newData,
					deferrals: deferrals,
					index:     i,
					err:       err,
				}
			}
		}
	}

	for i := 0; i < nWorker; i++ {
		go worker(jobs, results)
	}

	// push jobs to the workers.
	// block if no more worker is available
	for i := range batch {
		jobs <- i
	}

	// initialize the array, so that we can insert the encoded result at
	// the right position.
	encodedResults := make([]*encodedResult, len(batch))

	for i := 0; i < len(batch); i++ {
		result := <-results
		if result == nil {
			continue
		}

		if result.err != nil {
			return nil, fmt.Errorf("could not encode value: %v, %w", batch[result.index], result.err)
		}

		// since worker works on different job concurrently
		// the results might arrive in a different order than the original jobs.
		// use the index to insert the encoded result at the original position
		encodedResults[result.index] = result
	}

	// the returned encodedResults has no error, because all errors have been handled already
	return encodedResults, nil
}

// encoded could be nil if the given item doesn't have value
func (s *runtimeStorage) processEncodedItem(item writeItem, encoded *encodedResult) ([]writeItem, error) {
	var newItems []writeItem

	if item.value != nil {
		for _, deferredValue := range encoded.deferrals.Values {

			deferredStorageKey := StorageKey{
				Address: item.storageKey.Address,
				Key:     deferredValue.Key,
			}

			if !deferredValue.Value.IsModified() {
				continue
			}

			newItems = append(newItems, writeItem{
				storageKey: deferredStorageKey,
				value:      deferredValue.Value,
			})
		}

		for _, deferralMove := range encoded.deferrals.Moves {
			s.move(
				deferralMove.DeferredOwner,
				deferralMove.DeferredStorageKey,
				deferralMove.NewOwner,
				deferralMove.NewStorageKey,
			)
		}
	}

	var newData []byte
	if encoded != nil && len(encoded.newData) > 0 {
		newData = interpreter.PrependMagic(encoded.newData, interpreter.CurrentEncodingVersion)
	}

	var err error
	wrapPanic(func() {
		err = s.runtimeInterface.SetValue(
			item.storageKey.Address[:],
			[]byte(item.storageKey.Key),
			newData,
		)
	})
	if err != nil {
		return nil, err
	}
	return newItems, nil
}

func (s *runtimeStorage) encodeValue(
	value interpreter.Value,
	path string,
) (
	data []byte,
	deferrals *interpreter.EncodingDeferrals,
	err error,
) {
	reportMetric(
		func() {
			data, deferrals, err = interpreter.EncodeValue(
				value,
				[]string{path},
				true,
				nil,
			)
		},
		s.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ValueEncoded(duration)
		},
	)
	return
}

func (s *runtimeStorage) move(
	oldOwner common.Address, oldKey string,
	newOwner common.Address, newKey string,
) {
	data, err := s.runtimeInterface.GetValue(oldOwner[:], []byte(oldKey))
	if err != nil {
		panic(err)
	}

	err = s.runtimeInterface.SetValue(oldOwner[:], []byte(oldKey), nil)
	if err != nil {
		panic(err)
	}

	// NOTE: not prefix with magic, as data is moved, so might already have it
	err = s.runtimeInterface.SetValue(newOwner[:], []byte(newKey), data)
	if err != nil {
		panic(err)
	}
}
