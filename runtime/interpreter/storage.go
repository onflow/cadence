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

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
)

func StoredValue(storable atree.Storable, storage atree.SlabStorage) Value {
	storedValue, err := storable.StoredValue(storage)
	if err != nil {
		panic(err)
	}

	return MustConvertStoredValue(storedValue)
}

func MustConvertStoredValue(value atree.Value) Value {
	converted, err := convertStoredValue(value)
	if err != nil {
		panic(err)
	}
	return converted
}

func convertStoredValue(value atree.Value) (Value, error) {
	switch value := value.(type) {
	case *atree.Array:
		return &ArrayValue{
			Type:  value.Type().(ArrayStaticType),
			array: value,
		}, nil

	case *atree.OrderedMap:
		typeInfo := value.Type()
		switch typeInfo := typeInfo.(type) {
		case DictionaryStaticType:
			return &DictionaryValue{
				Type:       typeInfo,
				dictionary: value,
			}, nil

		case compositeTypeInfo:
			return &CompositeValue{
				dictionary:          value,
				Location:            typeInfo.location,
				QualifiedIdentifier: typeInfo.qualifiedIdentifier,
				Kind:                typeInfo.kind,
			}, nil

		default:
			return nil, fmt.Errorf("invalid ordered map type info: %T", typeInfo)
		}

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
	AccountStorage map[StorageKey]atree.Storable
}

var _ Storage = InMemoryStorage{}

func NewInMemoryStorage() InMemoryStorage {
	slabStorage := atree.NewBasicSlabStorage(
		CBOREncMode,
		CBORDecMode,
		DecodeStorable,
		DecodeTypeInfo,
	)

	return InMemoryStorage{
		BasicSlabStorage: slabStorage,
		AccountStorage:   make(map[StorageKey]atree.Storable),
	}
}

func DecodeTypeInfo(dec *cbor.StreamDecoder) (atree.TypeInfo, error) {
	tag, err := dec.DecodeTagNumber()
	if err != nil {
		return nil, err
	}

	switch tag {
	case CBORTagConstantSizedStaticType:
		return decodeConstantSizedStaticType(dec)
	case CBORTagVariableSizedStaticType:
		return decodeVariableSizedStaticType(dec)
	case CBORTagDictionaryStaticType:
		return decodeDictionaryStaticType(dec)
	case CBORTagCompositeValue:
		return decodeCompositeTypeInfo(dec)
	default:
		return nil, fmt.Errorf("invalid type info tag: %d", tag)
	}
}

func (i InMemoryStorage) ValueExists(_ *Interpreter, address common.Address, key string) bool {
	storageKey := StorageKey{
		Address: address,
		Key:     key,
	}
	_, ok := i.AccountStorage[storageKey]
	return ok
}

func (i InMemoryStorage) ReadValue(_ *Interpreter, address common.Address, key string) OptionalValue {
	storageKey := StorageKey{
		Address: address,
		Key:     key,
	}

	storable, ok := i.AccountStorage[storageKey]
	if !ok {
		return NilValue{}
	}

	storedValue := StoredValue(storable, i)
	return NewSomeValueNonCopying(storedValue)
}

func (i InMemoryStorage) WriteValue(
	interpreter *Interpreter,
	address common.Address,
	key string,
	value OptionalValue,
) {
	storageKey := StorageKey{
		Address: address,
		Key:     key,
	}

	// Remove existing, if any

	if existingStorable, ok := i.AccountStorage[storageKey]; ok {
		StoredValue(existingStorable, i).DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(existingStorable)
	}

	switch value := value.(type) {
	case *SomeValue:
		// Store new value (as storable)
		storable, err := value.Value.Storable(
			i,
			atree.Address(address),
			// NOTE: we already allocate a register for the account storage value,
			// so we might as well store all data of the value in it, if possible,
			// e.g. for a large immutable value.
			//
			// Using a smaller number would only result in an additional register
			// (account storage register would have storage ID storable,
			// and extra slab / register would contain the actual data of the value).
			math.MaxUint64,
		)
		if err != nil {
			panic(err)
		}
		i.AccountStorage[storageKey] = storable

	case NilValue:
		// Remove entry
		delete(i.AccountStorage, storageKey)
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
