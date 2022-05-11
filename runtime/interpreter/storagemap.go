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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
)

// StorageMap is an ordered map which stores values in an account.
//
type StorageMap struct {
	orderedMap *atree.OrderedMap
}

func NewStorageMap(memoryGauge common.MemoryGauge, storage atree.SlabStorage, address atree.Address) *StorageMap {
	common.UseMemory(memoryGauge, common.StorageMapMemoryUsage)

	orderedMap, err := atree.NewMap(
		storage,
		address,
		atree.NewDefaultDigesterBuilder(),
		emptyTypeInfo,
	)
	if err != nil {
		panic(ExternalError{err})
	}

	return &StorageMap{
		orderedMap: orderedMap,
	}
}

func NewStorageMapWithRootID(storage atree.SlabStorage, storageID atree.StorageID) *StorageMap {
	orderedMap, err := atree.NewMapWithRootID(
		storage,
		storageID,
		atree.NewDefaultDigesterBuilder(),
	)
	if err != nil {
		panic(ExternalError{err})
	}

	return &StorageMap{
		orderedMap: orderedMap,
	}
}

// ValueExists returns true if the given key exists in the storage map.
//
func (s StorageMap) ValueExists(key string) bool {
	_, err := s.orderedMap.Get(
		StringAtreeComparator,
		StringAtreeHashInput,
		StringAtreeValue(key),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return false
		}
		panic(ExternalError{err})
	}

	return true
}

// ReadValue returns the value for the given key.
// Returns nil if the key does not exist.
//
func (s StorageMap) ReadValue(interpreter *Interpreter, key string) Value {
	storable, err := s.orderedMap.Get(
		StringAtreeComparator,
		StringAtreeHashInput,
		StringAtreeValue(key),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return nil
		}
		panic(ExternalError{err})
	}

	return StoredValue(interpreter, storable, s.orderedMap.Storage)
}

// WriteValue sets or removes a value in the storage map.
// If the given value is nil, the key is removed.
// If the given value is non-nil, the key is added/updated.
//
func (s StorageMap) WriteValue(interpreter *Interpreter, key string, value atree.Value) {
	if value == nil {
		s.removeValue(interpreter, key)
	} else {
		s.SetValue(interpreter, key, value)
	}
}

// SetValue sets a value in the storage map.
// If the given key already stores a value, it is overwritten.
//
func (s StorageMap) SetValue(interpreter *Interpreter, key string, value atree.Value) {
	existingStorable, err := s.orderedMap.Set(
		StringAtreeComparator,
		StringAtreeHashInput,
		StringAtreeValue(key),
		value,
	)
	if err != nil {
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(s.orderedMap)

	if existingStorable != nil {
		existingValue := StoredValue(interpreter, existingStorable, interpreter.Storage)
		existingValue.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(existingStorable)
	}
}

// removeValue removes a value in the storage map, if it exists.
//
func (s StorageMap) removeValue(interpreter *Interpreter, key string) {
	existingKeyStorable, existingValueStorable, err := s.orderedMap.Remove(
		StringAtreeComparator,
		StringAtreeHashInput,
		StringAtreeValue(key),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return
		}
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(s.orderedMap)

	// Key

	// NOTE: key / field name is stringAtreeValue,
	// and not a Value, so no need to deep remove
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	if existingValueStorable != nil {
		existingValue := StoredValue(interpreter, existingValueStorable, interpreter.Storage)
		existingValue.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(existingValueStorable)
	}
}

// Iterator returns an iterator (StorageMapIterator),
// which allows iterating over the keys and values of the storage map
//
func (s StorageMap) Iterator(gauge common.MemoryGauge) StorageMapIterator {
	mapIterator, err := s.orderedMap.Iterator()
	if err != nil {
		panic(ExternalError{err})
	}

	return StorageMapIterator{
		gauge:       gauge,
		mapIterator: mapIterator,
		storage:     s.orderedMap.Storage,
	}
}

func (s StorageMap) StorageID() atree.StorageID {
	return s.orderedMap.StorageID()
}

// StorageMapIterator is an iterator over StorageMap
//
type StorageMapIterator struct {
	gauge       common.MemoryGauge
	mapIterator *atree.MapIterator
	storage     atree.SlabStorage
}

// Next returns the next key and value of the storage map iterator.
// If there is no further key-value pair, ("", nil) is returned.
//
func (i StorageMapIterator) Next() (string, Value) {
	k, v, err := i.mapIterator.Next()
	if err != nil {
		panic(ExternalError{err})
	}

	if k == nil || v == nil {
		return "", nil
	}

	key := string(k.(StringAtreeValue))
	value := MustConvertStoredValue(i.gauge, v)

	return key, value
}

// NextKey returns the next key of the storage map iterator.
// If there is no further key, "" is returned.
//
func (i StorageMapIterator) NextKey() string {
	k, err := i.mapIterator.NextKey()
	if err != nil {
		panic(ExternalError{err})
	}

	if k == nil {
		return ""
	}

	return string(k.(StringAtreeValue))
}

// NextValue returns the next value in the storage map iterator.
// If there is nop further value, nil is returned.
//
func (i StorageMapIterator) NextValue() Value {
	v, err := i.mapIterator.NextValue()
	if err != nil {
		panic(ExternalError{err})
	}

	if v == nil {
		return nil
	}

	return MustConvertStoredValue(i.gauge, v)
}
