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

package interpreter

import (
	goerrors "errors"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

// StorageMap is an ordered map which stores values in an account.
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
		panic(errors.NewExternalError(err))
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
		panic(errors.NewExternalError(err))
	}

	return &StorageMap{
		orderedMap: orderedMap,
	}
}

// ValueExists returns true if the given key exists in the storage map.
func (s StorageMap) ValueExists(key StorageMapKey) bool {
	_, err := s.orderedMap.Get(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return false
		}
		panic(errors.NewExternalError(err))
	}

	return true
}

// ReadValue returns the value for the given key.
// Returns nil if the key does not exist.
func (s StorageMap) ReadValue(gauge common.MemoryGauge, key StorageMapKey) Value {
	storable, err := s.orderedMap.Get(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil
		}
		panic(errors.NewExternalError(err))
	}

	return StoredValue(gauge, storable, s.orderedMap.Storage)
}

// WriteValue sets or removes a value in the storage map.
// If the given value is nil, the key is removed.
// If the given value is non-nil, the key is added/updated.
// Returns true if a value previously existed at the given key.
func (s StorageMap) WriteValue(interpreter *Interpreter, key StorageMapKey, value atree.Value) (existed bool) {
	if value == nil {
		return s.RemoveValue(interpreter, key)
	} else {
		return s.SetValue(interpreter, key, value)
	}
}

// SetValue sets a value in the storage map.
// If the given key already stores a value, it is overwritten.
// Returns true if
func (s StorageMap) SetValue(interpreter *Interpreter, key StorageMapKey, value atree.Value) (existed bool) {
	interpreter.recordStorageMutation()

	existingStorable, err := s.orderedMap.Set(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(s.orderedMap)

	existed = existingStorable != nil
	if existed {
		existingValue := StoredValue(interpreter, existingStorable, interpreter.Storage())
		existingValue.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(existingStorable)
	}
	return
}

// RemoveValue removes a value in the storage map, if it exists.
func (s StorageMap) RemoveValue(interpreter *Interpreter, key StorageMapKey) (existed bool) {
	interpreter.recordStorageMutation()

	existingKeyStorable, existingValueStorable, err := s.orderedMap.Remove(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return
		}
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(s.orderedMap)

	// Key

	// NOTE: Key is just an atree.Value, not an interpreter.Value,
	// so do not need (can) convert and not need to deep remove
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	existed = existingValueStorable != nil
	if existed {
		existingValue := StoredValue(interpreter, existingValueStorable, interpreter.Storage())
		existingValue.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(existingValueStorable)
	}
	return
}

// Iterator returns an iterator (StorageMapIterator),
// which allows iterating over the keys and values of the storage map
func (s StorageMap) Iterator(gauge common.MemoryGauge) StorageMapIterator {
	mapIterator, err := s.orderedMap.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
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

func (s StorageMap) Count() uint64 {
	return s.orderedMap.Count()
}

// StorageMapIterator is an iterator over StorageMap
type StorageMapIterator struct {
	gauge       common.MemoryGauge
	mapIterator *atree.MapIterator
	storage     atree.SlabStorage
}

// Next returns the next key and value of the storage map iterator.
// If there is no further key-value pair, ("", nil) is returned.
func (i StorageMapIterator) Next() (atree.Value, Value) {
	k, v, err := i.mapIterator.Next()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if k == nil || v == nil {
		return nil, nil
	}

	// NOTE: Key is just an atree.Value, not an interpreter.Value,
	// so do not need (can) convert

	value := MustConvertStoredValue(i.gauge, v)

	return k, value
}

// NextKey returns the next key of the storage map iterator.
// If there is no further key, "" is returned.
func (i StorageMapIterator) NextKey() atree.Value {
	k, err := i.mapIterator.NextKey()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return k
}

// NextValue returns the next value in the storage map iterator.
// If there is nop further value, nil is returned.
func (i StorageMapIterator) NextValue() Value {
	v, err := i.mapIterator.NextValue()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if v == nil {
		return nil
	}

	return MustConvertStoredValue(i.gauge, v)
}
