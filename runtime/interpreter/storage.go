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

	"github.com/fxamacker/atree"
	"github.com/onflow/cadence/runtime/common"
)

func StoredValue(storable atree.Storable, storage atree.SlabStorage) (Value, error) {
	storedValue, err := storable.StoredValue(storage)
	if err != nil {
		return nil, err
	}
	switch storedValue := storedValue.(type) {
	case *atree.Array:
		staticType, err := StaticTypeFromBytes([]byte(storedValue.Type()))
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
			array: storedValue,
			Type:  arrayType,
			// TODO: owner
		}, nil

	case Value:
		return storedValue, nil

	default:
		return nil, fmt.Errorf("invalid stored value: %T", storedValue)
	}
}

type InMemoryStorageKey struct {
	Address common.Address
	Key     string
}

type InMemoryStorage struct {
	*atree.BasicSlabStorage
	Data map[InMemoryStorageKey]atree.Storable
}

func (i InMemoryStorage) Exists(_ *Interpreter, address common.Address, key string) bool {
	_, ok := i.Data[InMemoryStorageKey{Address: address, Key: key}]
	return ok
}

func (i InMemoryStorage) Read(_ *Interpreter, address common.Address, key string) OptionalValue {
	storable, ok := i.Data[InMemoryStorageKey{Address: address, Key: key}]
	if !ok {
		return NilValue{}
	}

	value, err := StoredValue(storable, i.BasicSlabStorage)
	if err != nil {
		panic(ExternalError{err})
	}

	return NewSomeValueNonCopying(value.(Value))
}

func (i InMemoryStorage) Write(_ *Interpreter, address common.Address, key string, value OptionalValue) {
	storageKey := InMemoryStorageKey{
		Address: address,
		Key:     key,
	}

	switch value := value.(type) {
	case *SomeValue:
		storable, err := value.Value.(atree.Value).Storable(i, atree.Address(address))
		if err != nil {
			panic(ExternalError{err})
		}
		i.Data[storageKey] = storable

	case NilValue:
		delete(i.Data, storageKey)
	}
}

var _ Storage = InMemoryStorage{}

func NewInMemoryStorage() InMemoryStorage {
	slabStorage := atree.NewBasicSlabStorage(CBOREncMode, CBORDecMode)
	slabStorage.DecodeStorable = DecodeStorableV6

	return InMemoryStorage{
		BasicSlabStorage: slabStorage,
		Data:             make(map[InMemoryStorageKey]atree.Storable),
	}
}

func storableSize(storable atree.Storable) uint32 {
	// TODO: reduce allocation by encoding to encoder which only increases length counter
	encode, err := atree.Encode(storable, CBOREncMode)
	if err != nil {
		panic(err)
	}
	// TODO: check!
	return uint32(len(encode))
}

// maybeStoreExternally either returns the given immutable storable
// if it it can be inlined, or else stores it in a separate slab
// and returns a StorageIDStorable.
//
func maybeLargeImmutableStorable(
	storable atree.Storable,
	storage atree.SlabStorage,
	address atree.Address,
) (
	atree.Storable,
	error,
) {

	if storable.ByteSize() < uint32(atree.MaxInlineElementSize) {
		return storable, nil
	}

	storageID := storage.GenerateStorageID(address)
	slab := &atree.StorableSlab{
		StorageID: storageID,
		Storable:  storable,
	}

	err := storage.Store(storageID, slab)
	if err != nil {
		return nil, err
	}

	return atree.StorageIDStorable(storageID), nil
}
