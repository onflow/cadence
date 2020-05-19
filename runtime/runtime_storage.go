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
	"time"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type storageKey struct {
	storageIdentifier string
	key               string
}

type cacheEntry struct {
	// true indicates that the value definitely must be written, independent of the value.
	// false indicates that the value may has to be written if the value is modified.
	mustWrite bool
	value     interpreter.Value
}

type interpreterRuntimeStorage struct {
	runtimeInterface Interface
	cache            map[storageKey]cacheEntry
}

func newInterpreterRuntimeStorage(runtimeInterface Interface) *interpreterRuntimeStorage {
	return &interpreterRuntimeStorage{
		runtimeInterface: runtimeInterface,
		cache:            map[storageKey]cacheEntry{},
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
func (s *interpreterRuntimeStorage) valueExists(
	storageIdentifier string,
	key string,
) bool {

	fullKey := storageKey{
		storageIdentifier: storageIdentifier,
		key:               key,
	}

	// Check cache

	if entry, ok := s.cache[fullKey]; ok {
		return entry.value != nil
	}

	// Cache miss: Ask interface

	var exists bool
	var err error
	wrapPanic(func() {
		// TODO: fix controller
		exists, err = s.runtimeInterface.ValueExists([]byte(storageIdentifier), []byte{}, []byte(key))
	})
	if err != nil {
		panic(err)
	}

	if !exists {
		s.cache[fullKey] = cacheEntry{
			mustWrite: false,
			value:     nil,
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
func (s *interpreterRuntimeStorage) readValue(
	storageIdentifier string,
	key string,
	deferred bool,
) interpreter.OptionalValue {

	fullKey := storageKey{
		storageIdentifier: storageIdentifier,
		key:               key,
	}

	// Check cache. Return cached value, if any

	if entry, ok := s.cache[fullKey]; ok {
		if entry.value == nil {
			return interpreter.NilValue{}
		}

		return interpreter.NewSomeValueOwningNonCopying(entry.value)
	}

	// Cache miss: Load and deserialize the stored value (if any)
	// through the runtime interface

	var storedData []byte
	var err error
	wrapPanic(func() {
		// TODO: fix controller
		storedData, err = s.runtimeInterface.GetValue([]byte(storageIdentifier), []byte{}, []byte(key))
	})
	if err != nil {
		panic(err)
	}

	if len(storedData) == 0 {
		s.cache[fullKey] = cacheEntry{
			mustWrite: false,
			value:     nil,
		}
		return interpreter.NilValue{}
	}

	address := common.BytesToAddress([]byte(storageIdentifier))

	var storedValue interpreter.Value

	reportMetric(
		func() {
			storedValue, err = interpreter.DecodeValue(storedData, &address, []string{key})
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
		s.cache[fullKey] = cacheEntry{
			mustWrite: false,
			value:     storedValue,
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
func (s *interpreterRuntimeStorage) writeValue(
	storageIdentifier string,
	key string,
	value interpreter.OptionalValue,
) {
	fullKey := storageKey{
		storageIdentifier: storageIdentifier,
		key:               key,
	}

	// Only write the value to the cache.
	// The Cache is finally written back through the runtime interface in `writeCached`

	entry := s.cache[fullKey]
	entry.mustWrite = true

	switch typedValue := value.(type) {
	case *interpreter.SomeValue:
		entry.value = typedValue.Value

	case interpreter.NilValue:
		entry.value = nil

	default:
		panic(errors.NewUnreachableError())
	}

	s.cache[fullKey] = entry
}

// writeCached serializes/saves all values in the cache in storage (through the runtime interface).
//
func (s *interpreterRuntimeStorage) writeCached() {

	type writeItem struct {
		storageKey storageKey
		value      interpreter.Value
	}

	var items []writeItem

	for fullKey, entry := range s.cache {

		if !entry.mustWrite && entry.value != nil && !entry.value.Modified() {
			continue
		}

		items = append(items, writeItem{
			storageKey: fullKey,
			value:      entry.value,
		})
	}

	// Don't use a for-range loop, as keys are added while iterating
	for len(items) > 0 {
		var item writeItem
		item, items = items[0], items[1:]

		var newData []byte
		if item.value != nil {
			var deferredValues map[string]interpreter.Value
			var err error
			newData, deferredValues, err = s.encodeValue(item.value, item.storageKey.key)
			if err != nil {
				panic(err)
			}

			for deferredKey, deferredValue := range deferredValues {

				deferredStorageKey := storageKey{
					storageIdentifier: item.storageKey.storageIdentifier,
					key:               deferredKey,
				}

				if !deferredValue.Modified() {
					continue
				}

				items = append(items, writeItem{
					storageKey: deferredStorageKey,
					value:      deferredValue,
				})
			}
		}

		var err error
		wrapPanic(func() {
			// TODO: fix controller
			err = s.runtimeInterface.SetValue(
				[]byte(item.storageKey.storageIdentifier),
				[]byte{},
				[]byte(item.storageKey.key),
				newData,
			)
		})
		if err != nil {
			panic(err)
		}
	}
}

func (s *interpreterRuntimeStorage) encodeValue(
	value interpreter.Value,
	path string,
) (
	data []byte,
	deferredValues map[string]interpreter.Value,
	err error,
) {
	reportMetric(
		func() {
			data, deferredValues, err = interpreter.EncodeValue(value, []string{path}, true)
		},
		s.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ValueEncoded(duration)
		},
	)
	return
}
