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
	"github.com/fxamacker/atree"
	"github.com/onflow/cadence/runtime/common"
)

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

	value, err := storable.StoredValue(i.BasicSlabStorage)
	if err != nil {
		panic(ExternalError{err})
	}

	return NewSomeValueOwningNonCopying(value.(Value))
}

func (i InMemoryStorage) Write(_ *Interpreter, address common.Address, key string, value OptionalValue) {
	storageKey := InMemoryStorageKey{
		Address: address,
		Key:     key,
	}

	switch value := value.(type) {
	case *SomeValue:

		i.Data[storageKey] = value.Value.(atree.Value).Storable(i)

	case NilValue:
		delete(i.Data, storageKey)
	}
}

var _ Storage = InMemoryStorage{}

func NewInMemoryStorage() InMemoryStorage {
	slabStorage := atree.NewBasicSlabStorage(CBOREncMode)
	slabStorage.DecodeStorable = DecodeStorableV6

	return InMemoryStorage{
		BasicSlabStorage: slabStorage,
		Data:             make(map[InMemoryStorageKey]atree.Storable),
	}
}

func storableSize(storable atree.Storable, slabStorage atree.SlabStorage) uint32 {
	encode, err := atree.Encode(storable, slabStorage)
	if err != nil {
		panic(err)
	}
	// TODO: check!
	return uint32(len(encode))
}
