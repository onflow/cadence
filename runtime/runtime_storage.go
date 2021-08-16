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
	"github.com/fxamacker/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type runtimeStorage struct {
	*atree.PersistentSlabStorage
	runtimeInterface Interface
	// NOTE: temporary, will be refactored to dictionary
	accountValues   map[interpreter.StorageKey]interpreter.Value
	contractUpdates map[interpreter.StorageKey]interpreter.Value
}

var _ atree.SlabStorage = &runtimeStorage{}
var _ interpreter.Storage = &runtimeStorage{}

func newRuntimeStorage(runtimeInterface Interface) *runtimeStorage {
	ledgerStorage := atree.NewLedgerBaseStorage(runtimeInterface)
	persistentSlabStorage := atree.NewPersistentSlabStorage(
		ledgerStorage,
		interpreter.CBOREncMode,
		interpreter.CBORDecMode,
	)
	return &runtimeStorage{
		PersistentSlabStorage: persistentSlabStorage,
		runtimeInterface:      runtimeInterface,
		accountValues:         map[interpreter.StorageKey]interpreter.Value{},
		contractUpdates:       map[interpreter.StorageKey]interpreter.Value{},
	}
}

// ValueExists returns true if a value exists in account storage.
//
func (s *runtimeStorage) ValueExists(
	_ *interpreter.Interpreter,
	address common.Address,
	key string,
) bool {

	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	// Check locally

	if value, ok := s.accountValues[storageKey]; ok {
		return value != nil
	}

	// Ask interface

	var exists bool
	var err error
	wrapPanic(func() {
		exists, err = s.runtimeInterface.ValueExists(address[:], []byte(key))
	})
	if err != nil {
		panic(err)
	}

	if !exists {
		s.accountValues[storageKey] = nil
	}

	return exists
}

// ReadValue returns a value from account storage.
//
func (s *runtimeStorage) ReadValue(
	_ *interpreter.Interpreter,
	address common.Address,
	key string,
) interpreter.OptionalValue {

	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	// Check locally

	if value, ok := s.accountValues[storageKey]; ok {
		if value == nil {
			return interpreter.NilValue{}
		}

		return interpreter.NewSomeValueNonCopying(value)
	}

	// TODO:
	//// Load and deserialize the stored value (if any)
	//// through the runtime interface
	//
	//var storedData []byte
	//var err error
	//wrapPanic(func() {
	//	storedData, err = s.runtimeInterface.GetValue(address[:], []byte(key))
	//})
	//if err != nil {
	//	panic(err)
	//}
	//
	//if len(storedData) == 0 {
	//	s.accountValues[storageKey] = nil
	//	return interpreter.NilValue{}
	//}
	//
	//var storedValue interpreter.Value
	//
	//reportMetric(
	//	func() {
	//		storedValue, err = interpreter.DecodeStorableV6(storedData)
	//	},
	//	s.runtimeInterface,
	//	func(metrics Metrics, duration time.Duration) {
	//		metrics.ValueDecoded(duration)
	//	},
	//)
	//if err != nil {
	//	panic(err)
	//}
	//
	//return interpreter.NewSomeValueNonCopying(storedValue)

	return interpreter.NilValue{}
}

func (s *runtimeStorage) WriteValue(
	_ *interpreter.Interpreter,
	address common.Address,
	key string,
	value interpreter.OptionalValue,
) {
	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	// Only write locally.
	// The value is eventually written back through the runtime interface in `commit`.

	var writtenValue interpreter.Value

	switch typedValue := value.(type) {
	case *interpreter.SomeValue:
		writtenValue = typedValue.Value

	case interpreter.NilValue:
		writtenValue = nil

	default:
		panic(errors.NewUnreachableError())
	}

	s.accountValues[storageKey] = writtenValue
}

func (s *runtimeStorage) recordContractUpdate(
	address common.Address,
	key string,
	contract interpreter.Value,
) {
	storageKey := interpreter.StorageKey{
		Address: address,
		Key:     key,
	}

	s.contractUpdates[storageKey] = contract
}

// commit serializes/saves all values in the cache in storage (through the runtime interface).
//
func (s *runtimeStorage) commit(inter *interpreter.Interpreter) error {

	// TODO:
	//var items []writeItem
	//
	//// First, iterate over the cache
	//// and determine which items have to be written
	//
	//for fullKey, entry := range s.cache { //nolint:maprangecheck
	//
	//	if !entry.MustWrite && entry.Value != nil && !entry.Value.IsModified() {
	//		continue
	//	}
	//
	//	items = append(items, writeItem{
	//		storageKey: fullKey,
	//		value:      entry.Value,
	//	})
	//}
	//
	//for fullKey, value := range s.contractUpdates { //nolint:maprangecheck
	//	items = append(items, writeItem{
	//		storageKey: fullKey,
	//		value:      value,
	//	})
	//}
	//
	//// Order the items by storage key in lexicographic order
	//
	//sort.Slice(items, func(i, j int) bool {
	//	a := items[i].storageKey
	//	b := items[j].storageKey
	//
	//	if bytes.Compare(a.Address[:], b.Address[:]) < 0 {
	//		return true
	//	}
	//
	//	if a.Key < b.Key {
	//		return true
	//	}
	//
	//	return false
	//})
	//
	//// Write cache entries in order
	//
	//// run batch in a for loop, each batch will create a new batch
	//// to be run again, until the batch is empty.
	//batch := items
	//for len(batch) > 0 {
	//
	//	// a batch might contain lots of items, whereas
	//	// a bundle only contains up to ENCODING_NUM_WORKER number of items,
	//	// so that we could ensure the memory usage is O(K), instead of O(N).
	//	// K being ENCODING_NUM_WORKER, N being the size of the batch
	//	var bundleSize int
	//	if len(batch) < ENCODING_NUM_WORKER {
	//		bundleSize = len(batch)
	//	} else {
	//		bundleSize = ENCODING_NUM_WORKER
	//	}
	//	bundle, newBatch := batch[:bundleSize], batch[bundleSize:]
	//
	//	// Ensure all items (values) are storable before encoding them
	//
	//	for _, item := range bundle {
	//		if item.value != nil && !item.value.IsStorable() {
	//			return NonStorableValueWriteError{
	//				Value: item.value,
	//			}
	//		}
	//	}
	//
	//	// parallelize the encoding for items within the same batch
	//	// the encoding has no side effect
	//	encodedResults, err := s.encodeWriteItems(ENCODING_NUM_WORKER, bundle)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	// process encoded items of the batch, play side effect for
	//	// each item
	//	for i, result := range encodedResults {
	//		item := batch[i]
	//		newItems, err := s.processEncodedItem(item, result)
	//		if err != nil {
	//			panic(err)
	//		}
	//		newBatch = append(newBatch, newItems...)
	//	}
	//	batch = newBatch
	//}

	return nil
}

//
//type encodedResult struct {
//	newData []byte
//	index   int // the index of the encoded item in the batch
//	err     error
//}
//
//func (s *runtimeStorage) encodeWriteItems(nWorker int, batch []writeItem) ([]*encodedResult, error) {
//	// cache all the encoded results, including errors
//	results := make(chan *encodedResult, len(batch))
//	defer close(results)
//
//	// if the number of items in a batch is less than the default number of workers, we will only create the
//	// same amount of workers as the items
//	if len(batch) < nWorker {
//		nWorker = len(batch)
//	}
//
//	// jobs buffers at most nWorker number of jobs for workers to work on in parallel
//	// each job contains the index of the item in the batch
//	jobs := make(chan int, nWorker)
//	defer close(jobs)
//
//	worker := func(jobs <-chan int, results chan<- *encodedResult) {
//		for i := range jobs {
//			item := batch[i]
//			if item.value == nil {
//				results <- nil
//			} else {
//				// TODO: encode value with a context, so that if there 10 jobs to work on,
//				// and one job failed, we could cancel the context and skip the rest of
//				// unprocssed jobs without processing them.
//				newData, deferrals, err := s.encodeValue(item.value, item.storageKey.Key)
//				results <- &encodedResult{
//					newData:   newData,
//					deferrals: deferrals,
//					index:     i,
//					err:       err,
//				}
//			}
//		}
//	}
//
//	for i := 0; i < nWorker; i++ {
//		go worker(jobs, results)
//	}
//
//	// push jobs to the workers.
//	// block if no more worker is available
//	for i := range batch {
//		jobs <- i
//	}
//
//	// initialize the array, so that we can insert the encoded result at
//	// the right position.
//	encodedResults := make([]*encodedResult, len(batch))
//
//	for i := 0; i < len(batch); i++ {
//		result := <-results
//		if result == nil {
//			continue
//		}
//
//		if result.err != nil {
//			return nil, fmt.Errorf("could not encode value: %v, %w", batch[result.index], result.err)
//		}
//
//		// since worker works on different job concurrently
//		// the results might arrive in a different order than the original jobs.
//		// use the index to insert the encoded result at the original position
//		encodedResults[result.index] = result
//	}
//
//	// the returned encodedResults has no error, because all errors have been handled already
//	return encodedResults, nil
//}
//
//// encoded could be nil if the given item doesn't have value
//func (s *runtimeStorage) processEncodedItem(item writeItem, encoded *encodedResult) ([]writeItem, error) {
//	var newItems []writeItem
//
//	if item.value != nil {
//		for _, deferredValue := range encoded.deferrals.Values {
//
//			deferredStorageKey := StorageKey{
//				Address: item.storageKey.Address,
//				Key:     deferredValue.Key,
//			}
//
//			if !deferredValue.Value.IsModified() {
//				continue
//			}
//
//			newItems = append(newItems, writeItem{
//				storageKey: deferredStorageKey,
//				value:      deferredValue.Value,
//			})
//		}
//
//		for _, deferralMove := range encoded.deferrals.Moves {
//			s.move(
//				deferralMove.DeferredOwner,
//				deferralMove.DeferredStorageKey,
//				deferralMove.NewOwner,
//				deferralMove.NewStorageKey,
//			)
//		}
//	}
//
//	var newData []byte
//	if encoded != nil && len(encoded.newData) > 0 {
//		newData = interpreter.PrependMagic(encoded.newData, interpreter.CurrentEncodingVersion)
//	}
//
//	var err error
//	wrapPanic(func() {
//		err = s.runtimeInterface.SetValue(
//			item.storageKey.Address[:],
//			[]byte(item.storageKey.Key),
//			newData,
//		)
//	})
//	if err != nil {
//		return nil, err
//	}
//	return newItems, nil
//}
//
//func (s *runtimeStorage) encodeValue(
//	value interpreter.Value,
//	path string,
//) (
//	data []byte,
//	deferrals *interpreter.EncodingDeferrals,
//	err error,
//) {
//	reportMetric(
//		func() {
//			data, deferrals, err = interpreter.EncodeValue(
//				value,
//			)
//		},
//		s.runtimeInterface,
//		func(metrics Metrics, duration time.Duration) {
//			metrics.ValueEncoded(duration)
//		},
//	)
//	return
//}
