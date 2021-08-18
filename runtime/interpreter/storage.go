/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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
	"fmt"
	"math"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
)

func StoredValue(storable atree.Storable, storage atree.SlabStorage) (Value, error) {
	storedValue, err := storable.StoredValue(storage)
	if err != nil {
		return nil, err
	}

	return convertStoredValue(storedValue)
}

func MustConvertStoredValue(value atree.Value) Value {
	converted, err := convertStoredValue(value)
	if err != nil {
		panic(ExternalError{err})
	}
	return converted
}

func convertStoredValue(value atree.Value) (Value, error) {
	switch value := value.(type) {
	case *atree.Array:
		// TODO: optimize
		staticType, err := StaticTypeFromBytes([]byte(value.Type()))
		if err != nil {
			return nil, err
		}

		arrayType, ok := staticType.(ArrayStaticType)
		if !ok {
			return nil, fmt.Errorf(
				"invalid array static type: %v",
				staticType,
			)
		}

		return &ArrayValue{
			array: value,
			Type:  arrayType,
		}, nil

	case Value:
		return value, nil

	default:
		return nil, fmt.Errorf("cannot convert stored value: %T", value)
	}
}

type StorageKey struct {
	Address common.Address
	Key     string
}

type InMemoryStorage struct {
	*atree.BasicSlabStorage
	AccountValues map[StorageKey]Value
}

var _ Storage = InMemoryStorage{}

func NewInMemoryStorage() InMemoryStorage {
	slabStorage := atree.NewBasicSlabStorage(CBOREncMode, CBORDecMode)
	slabStorage.DecodeStorable = DecodeStorable

	return InMemoryStorage{
		BasicSlabStorage: slabStorage,
		AccountValues:    make(map[StorageKey]Value),
	}
}

func (i InMemoryStorage) ValueExists(_ *Interpreter, address common.Address, key string) bool {
	storageKey := StorageKey{
		Address: address,
		Key:     key,
	}
	_, ok := i.AccountValues[storageKey]
	return ok
}

func (i InMemoryStorage) ReadValue(_ *Interpreter, address common.Address, key string) OptionalValue {
	storageKey := StorageKey{
		Address: address,
		Key:     key,
	}

	value, ok := i.AccountValues[storageKey]
	if !ok {
		return NilValue{}
	}

	return NewSomeValueNonCopying(MustConvertStoredValue(value))
}

func (i InMemoryStorage) WriteValue(_ *Interpreter, address common.Address, key string, value OptionalValue) {
	storageKey := StorageKey{
		Address: address,
		Key:     key,
	}

	switch value := value.(type) {
	case *SomeValue:
		i.AccountValues[storageKey] = value.Value

	case NilValue:
		delete(i.AccountValues, storageKey)
	}
}

type writeCounter struct {
	length uint64
}

func (w *writeCounter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.length += uint64(n)
	return n, nil
}

func mustStorableSize(storable atree.Storable) uint32 {
	size, err := StorableSize(storable)
	if err != nil {
		panic(err)
	}
	return size
}

func StorableSize(storable atree.Storable) (uint32, error) {
	var writer writeCounter
	enc := atree.NewEncoder(&writer, CBOREncMode)

	err := storable.Encode(enc)
	if err != nil {
		return 0, err
	}

	err = enc.CBOR.Flush()
	if err != nil {
		return 0, err
	}

	size := writer.length
	if size > math.MaxUint32 {
		return 0, fmt.Errorf("storable size is too large: expected max uint32, got %d", size)
	}

	return uint32(size), nil
}

// maybeStoreExternally either returns the given immutable storable
// if it it can be inlined, or else stores it in a separate slab
// and returns a StorageIDStorable.
//
func maybeLargeImmutableStorable(
	storable atree.Storable,
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (
	atree.Storable,
	error,
) {

	if uint64(storable.ByteSize()) < maxInlineSize {
		return storable, nil
	}

	storageID, err := storage.GenerateStorageID(address)
	if err != nil {
		return nil, err
	}

	slab := &atree.StorableSlab{
		StorageID: storageID,
		Storable:  storable,
	}

	err = storage.Store(storageID, slab)
	if err != nil {
		return nil, err
	}

	return atree.StorageIDStorable(storageID), nil
}
