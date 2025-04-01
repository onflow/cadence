/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
	"time"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

// DomainStorageMap is an ordered map which stores values in an account domain.
type DomainStorageMap struct {
	orderedMap *atree.OrderedMap
}

// NewDomainStorageMap creates new domain storage map for given address.
func NewDomainStorageMap(memoryGauge common.MemoryGauge, storage atree.SlabStorage, address atree.Address) *DomainStorageMap {
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

	return &DomainStorageMap{
		orderedMap: orderedMap,
	}
}

// NewDomainStorageMapWithRootID loads domain storage map with given slabID.
// This function is only used with legacy domain registers for unmigrated accounts.
// For migrated accounts, NewDomainStorageMapWithAtreeValue() is used to load
// domain storage map as an element of AccountStorageMap.
func NewDomainStorageMapWithRootID(storage atree.SlabStorage, slabID atree.SlabID) *DomainStorageMap {
	orderedMap, err := atree.NewMapWithRootID(
		storage,
		slabID,
		atree.NewDefaultDigesterBuilder(),
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &DomainStorageMap{
		orderedMap: orderedMap,
	}
}

// newDomainStorageMapWithAtreeStorable loads domain storage map with given atree.Storable.
func newDomainStorageMapWithAtreeStorable(storage atree.SlabStorage, storable atree.Storable) *DomainStorageMap {

	// NOTE: Don't use interpreter.StoredValue() to convert given storable
	// to DomainStorageMap because DomainStorageMap isn't interpreter.Value.

	value, err := storable.StoredValue(storage)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewDomainStorageMapWithAtreeValue(value)
}

// NewDomainStorageMapWithAtreeValue loads domain storage map with given atree.Value.
// This function is used by migrated account to load domain as an element of AccountStorageMap.
func NewDomainStorageMapWithAtreeValue(value atree.Value) *DomainStorageMap {
	// Check if type of given value is *atree.OrderedMap
	dm, isAtreeOrderedMap := value.(*atree.OrderedMap)
	if !isAtreeOrderedMap {
		panic(errors.NewUnexpectedError(
			"domain storage map has unexpected type %T, expect *atree.OrderedMap",
			value,
		))
	}

	// Check if TypeInfo of atree.OrderedMap is EmptyTypeInfo
	dt, isEmptyTypeInfo := dm.Type().(EmptyTypeInfo)
	if !isEmptyTypeInfo {
		panic(errors.NewUnexpectedError(
			"domain storage map has unexpected encoded type %T, expect EmptyTypeInfo",
			dt,
		))
	}

	return &DomainStorageMap{orderedMap: dm}
}

// ValueExists returns true if the given key exists in the storage map.
func (s *DomainStorageMap) ValueExists(key StorageMapKey) bool {
	exists, err := s.orderedMap.Has(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return exists
}

// ReadValue returns the value for the given key.
// Returns nil if the key does not exist.
func (s *DomainStorageMap) ReadValue(gauge common.MemoryGauge, key StorageMapKey) Value {
	storedValue, err := s.orderedMap.Get(
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

	return MustConvertStoredValue(gauge, storedValue)
}

// WriteValue sets or removes a value in the storage map.
// If the given value is nil, the key is removed.
// If the given value is non-nil, the key is added/updated.
// Returns true if a value previously existed at the given key.
func (s *DomainStorageMap) WriteValue(context ValueTransferContext, key StorageMapKey, value atree.Value) (existed bool) {
	if value == nil {
		return s.RemoveValue(context, key)
	} else {
		return s.SetValue(context, key, value)
	}
}

// SetValue sets a value in the storage map.
// If the given key already stores a value, it is overwritten.
// Returns true if given key already exists and existing value is overwritten.
func (s *DomainStorageMap) SetValue(context ValueTransferContext, key StorageMapKey, value atree.Value) (existed bool) {
	context.RecordStorageMutation()

	existingStorable, err := s.orderedMap.Set(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	existed = existingStorable != nil
	if existed {
		existingValue := StoredValue(context, existingStorable, context.Storage())
		existingValue.DeepRemove(context, true) // existingValue is standalone because it was overwritten in parent container.
		RemoveReferencedSlab(context, existingStorable)
	}

	context.MaybeValidateAtreeValue(s.orderedMap)
	context.MaybeValidateAtreeStorage()

	return
}

// RemoveValue removes a value in the storage map, if it exists.
func (s *DomainStorageMap) RemoveValue(context ValueRemoveContext, key StorageMapKey) (existed bool) {
	context.RecordStorageMutation()

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

	// Key

	// NOTE: Key is just an atree.Value, not an interpreter.Value,
	// so do not need (can) convert and not need to deep remove
	RemoveReferencedSlab(context, existingKeyStorable)

	// Value

	existed = existingValueStorable != nil
	if existed {
		existingValue := StoredValue(context, existingValueStorable, context.Storage())
		existingValue.DeepRemove(context, true) // existingValue is standalone because it was removed from parent container.
		RemoveReferencedSlab(context, existingValueStorable)
	}

	context.MaybeValidateAtreeValue(s.orderedMap)
	context.MaybeValidateAtreeStorage()

	return
}

// DeepRemove removes all elements (and their slabs) of domain storage map.
func (s *DomainStorageMap) DeepRemove(context ValueRemoveContext, hasNoParentContainer bool) {

	if context.TracingEnabled() {
		startTime := time.Now()

		typeInfo := "DomainStorageMap"
		count := s.Count()

		defer func() {
			context.reportDomainStorageMapDeepRemoveTrace(
				typeInfo,
				int(count),
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	// Remove keys and values

	storage := s.orderedMap.Storage

	err := s.orderedMap.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {
		// Key

		// NOTE: Key is just an atree.Value, not an interpreter.Value,
		// so do not need (can) convert and not need to deep remove
		RemoveReferencedSlab(context, keyStorable)

		// Value

		value := StoredValue(context, valueStorable, storage)
		value.DeepRemove(context, false) // value is an element of v.dictionary because it is from PopIterate() callback.
		RemoveReferencedSlab(context, valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	context.MaybeValidateAtreeValue(s.orderedMap)
	if hasNoParentContainer {
		context.MaybeValidateAtreeStorage()
	}
}

func (s *DomainStorageMap) SlabID() atree.SlabID {
	return s.orderedMap.SlabID()
}

func (s *DomainStorageMap) ValueID() atree.ValueID {
	return s.orderedMap.ValueID()
}

func (s *DomainStorageMap) Count() uint64 {
	return s.orderedMap.Count()
}

func (s *DomainStorageMap) Inlined() bool {
	// This is only used for testing currently.
	return s.orderedMap.Inlined()
}

// Iterator returns an iterator (StorageMapIterator),
// which allows iterating over the keys and values of the storage map
func (s *DomainStorageMap) Iterator(gauge common.MemoryGauge) DomainStorageMapIterator {
	mapIterator, err := s.orderedMap.Iterator(
		StorageMapKeyAtreeValueComparator,
		StorageMapKeyAtreeValueHashInput,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return DomainStorageMapIterator{
		gauge:       gauge,
		mapIterator: mapIterator,
		storage:     s.orderedMap.Storage,
	}
}

// DomainStorageMapIterator is an iterator over DomainStorageMap
type DomainStorageMapIterator struct {
	gauge       common.MemoryGauge
	mapIterator atree.MapIterator
	storage     atree.SlabStorage
}

// Next returns the next key and value of the storage map iterator.
// If there is no further key-value pair, (nil, nil) is returned.
func (i DomainStorageMapIterator) Next() (atree.Value, Value) {
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
func (i DomainStorageMapIterator) NextKey() atree.Value {
	k, err := i.mapIterator.NextKey()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return k
}

// NextValue returns the next value in the storage map iterator.
// If there is no further value, nil is returned.
func (i DomainStorageMapIterator) NextValue() Value {
	v, err := i.mapIterator.NextValue()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if v == nil {
		return nil
	}

	return MustConvertStoredValue(i.gauge, v)
}
