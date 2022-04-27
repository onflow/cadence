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
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/errors"

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
	converted, err := ConvertStoredValue(value)
	if err != nil {
		panic(err)
	}
	return converted
}

func ConvertStoredValue(value atree.Value) (Value, error) {
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

func DecodeTypeInfo(dec *cbor.StreamDecoder) (atree.TypeInfo, error) {
	ty, err := dec.NextType()
	if err != nil {
		return nil, err
	}

	switch ty {
	case cbor.TagType:

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
			return nil, fmt.Errorf("invalid type info CBOR tag: %d", tag)
		}

	case cbor.NilType:
		err = dec.DecodeNil()
		if err != nil {
			return nil, err
		}
		return emptyTypeInfo, nil

	default:
		return nil, fmt.Errorf("invalid type info CBOR type: %d", ty)
	}
}

type StorageKey struct {
	Address common.Address
	Key     string
}

func (k StorageKey) IsLess(o StorageKey) bool {
	switch bytes.Compare(k.Address[:], o.Address[:]) {
	case -1:
		return true
	case 0:
		return strings.Compare(k.Key, o.Key) < 0
	case 1:
		return false
	default:
		panic(errors.NewUnreachableError())
	}
}

// InMemoryStorage
//
type InMemoryStorage struct {
	*atree.BasicSlabStorage
	StorageMaps map[StorageKey]*StorageMap
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
		StorageMaps:      make(map[StorageKey]*StorageMap),
	}
}

func (i InMemoryStorage) GetStorageMap(
	address common.Address,
	domain string,
	createIfNotExists bool,
) (
	storageMap *StorageMap,
) {
	key := StorageKey{address, domain}
	storageMap = i.StorageMaps[key]
	if storageMap == nil && createIfNotExists {
		storageMap = NewStorageMap(i, atree.Address(address))
		i.StorageMaps[key] = storageMap
	}
	return storageMap
}

func (i InMemoryStorage) CheckHealth() error {
	_, err := atree.CheckStorageHealth(i, -1)
	return err
}

// writeCounter is an io.Writer which counts the amount of written data.
//
type writeCounter struct {
	length uint64
}

var _ io.Writer = &writeCounter{}

func (w *writeCounter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.length += uint64(n)
	return n, nil
}

// mustStorableSize returns the result of StorableSize, and panics if it fails.
//
func mustStorableSize(storable atree.Storable) uint32 {
	size, err := StorableSize(storable)
	if err != nil {
		panic(err)
	}
	return size
}

// StorableSize returns the size of the storable in bytes.
//
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

// maybeLargeImmutableStorable either returns the given immutable atree.Storable
// if it can be stored inline inside its parent container,
// or else stores it in a separate slab and returns an atree.StorageIDStorable.
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
