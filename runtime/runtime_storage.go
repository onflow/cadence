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

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type storageKey struct {
	address common.Address
	key     string
}

type cacheEntry struct {
	// true indicates that the value definitely must be written, independent of the value.
	// false indicates that the value may has to be written if the value is modified.
	mustWrite bool
	value     interpreter.Value
}

type interpreterRuntimeStorage struct {
	runtimeInterface        Interface
	highLevelStorageEnabled bool
	highLevelStorage        HighLevelStorage
	cache                   map[storageKey]cacheEntry
}

// temporary export the type for usage in ParsingCheckingError
type InterpreterRuntimeStorage = interpreterRuntimeStorage

func newInterpreterRuntimeStorage(runtimeInterface Interface) *interpreterRuntimeStorage {
	highLevelStorageEnabled := false
	highLevelStorage, ok := runtimeInterface.(HighLevelStorage)
	if ok {
		highLevelStorageEnabled = highLevelStorage.HighLevelStorageEnabled()
	}

	return &interpreterRuntimeStorage{
		runtimeInterface:        runtimeInterface,
		cache:                   map[storageKey]cacheEntry{},
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
func (s *interpreterRuntimeStorage) valueExists(
	address common.Address,
	key string,
) bool {

	fullKey := storageKey{
		address: address,
		key:     key,
	}

	// Check cache

	if entry, ok := s.cache[fullKey]; ok {
		return entry.value != nil
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
	address common.Address,
	key string,
	deferred bool,
) interpreter.OptionalValue {

	fullKey := storageKey{
		address: address,
		key:     key,
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
		storedData, err = s.runtimeInterface.GetValue(address[:], []byte(key))
	})
	if err != nil {
		panic(err)
	}

	var version uint16
	storedData, version = interpreter.StripMagic(storedData)

	if len(storedData) == 0 {
		s.cache[fullKey] = cacheEntry{
			mustWrite: false,
			value:     nil,
		}
		return interpreter.NilValue{}
	}

	var storedValue interpreter.Value

	reportMetric(
		func() {
			storedValue, err = interpreter.DecodeValue(storedData, &address, []string{key}, version)
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
	address common.Address,
	key string,
	value interpreter.OptionalValue,
) {
	fullKey := storageKey{
		address: address,
		key:     key,
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
func (s *interpreterRuntimeStorage) writeCached(inter *interpreter.Interpreter) {

	type writeItem struct {
		storageKey storageKey
		value      interpreter.Value
	}

	var items []writeItem

	for fullKey, entry := range s.cache {

		if !entry.mustWrite && entry.value != nil && !entry.value.IsModified() {
			continue
		}

		items = append(items, writeItem{
			storageKey: fullKey,
			value:      entry.value,
		})

		if s.highLevelStorageEnabled {
			var err error

			var value cadence.Value
			if entry.value != nil {
				value = exportValueWithInterpreter(entry.value, inter, exportResults{})
			}

			wrapPanic(func() {
				err = s.highLevelStorage.SetCadenceValue(fullKey.address, fullKey.key, value)
			})
			if err != nil {
				panic(err)
			}
		}
	}

	// Don't use a for-range loop, as keys are added while iterating
	for len(items) > 0 {
		var item writeItem
		item, items = items[0], items[1:]

		var newData []byte
		if item.value != nil {
			var deferrals *interpreter.EncodingDeferrals
			var err error
			newData, deferrals, err = s.encodeValue(item.value, item.storageKey.key)
			if err != nil {
				panic(err)
			}

			for deferredKey, deferredValue := range deferrals.Values {

				deferredStorageKey := storageKey{
					address: item.storageKey.address,
					key:     deferredKey,
				}

				if !deferredValue.IsModified() {
					continue
				}

				items = append(items, writeItem{
					storageKey: deferredStorageKey,
					value:      deferredValue,
				})
			}

			for _, deferralMove := range deferrals.Moves {

				s.move(
					deferralMove.DeferredOwner,
					deferralMove.DeferredStorageKey,
					deferralMove.NewOwner,
					deferralMove.NewStorageKey,
				)
			}
		}

		if len(newData) > 0 {
			newData = interpreter.PrependMagic(newData, interpreter.CurrentEncodingVersion)
		}

		var err error
		wrapPanic(func() {
			err = s.runtimeInterface.SetValue(
				item.storageKey.address[:],
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
	deferrals *interpreter.EncodingDeferrals,
	err error,
) {
	reportMetric(
		func() {
			data, deferrals, err = interpreter.EncodeValue(value, []string{path}, true)
		},
		s.runtimeInterface,
		func(metrics Metrics, duration time.Duration) {
			metrics.ValueEncoded(duration)
		},
	)
	return
}

func (s *interpreterRuntimeStorage) move(
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
