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
	"bytes"
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
		// TODO: optimize
		arrayType, err := decodeArrayTypeInfo(value.Type())
		if err != nil {
			return nil, err
		}

		return &ArrayValue{
			array: value,
			Type:  arrayType,
		}, nil

	case *atree.OrderedMap:
		// TODO: optimize
		info, err := decodeOrderedMapTypeInfo(value.Type())
		if err != nil {
			return nil, err
		}

		switch info := info.(type) {
		case dictionaryOrderedMapTypeInfo:
			return &DictionaryValue{
				dictionary: value,
				Type:       DictionaryStaticType(info),
			}, nil

		case compositeOrderedMapTypeInfo:
			return &CompositeValue{
				dictionary:          value,
				Location:            info.location,
				QualifiedIdentifier: info.qualifiedIdentifier,
				Kind:                info.kind,
			}, nil

		default:
			return nil, fmt.Errorf(
				"invalid ordered map info: %T",
				info,
			)
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

func encodeArrayTypeInfo(arrayType ArrayStaticType) cbor.RawMessage {
	var buf bytes.Buffer
	enc := CBOREncMode.NewStreamEncoder(&buf)

	err := enc.EncodeTagHead(CBORTagArrayValue)
	if err != nil {
		panic(ExternalError{err})
	}

	err = EncodeStaticType(enc, arrayType)
	if err != nil {
		panic(ExternalError{err})
	}

	err = enc.Flush()
	if err != nil {
		panic(ExternalError{err})
	}

	return buf.Bytes()
}

func decodeArrayTypeInfo(typeInfo cbor.RawMessage) (ArrayStaticType, error) {
	dec := CBORDecMode.NewByteStreamDecoder(typeInfo)

	tagNumber, err := dec.DecodeTagNumber()
	if err != nil {
		panic(ExternalError{err})
	}

	if tagNumber != CBORTagArrayValue {
		return nil, fmt.Errorf(
			"invalid array type info: expected tag %d, got %d",
			CBORTagArrayValue, tagNumber,
		)
	}

	staticType, err := decodeStaticType(dec)
	if err != nil {
		return nil, err
	}

	arrayType, ok := staticType.(ArrayStaticType)
	if !ok {
		return nil, fmt.Errorf(
			"invalid array static type in array type info: %T",
			staticType,
		)
	}

	return arrayType, nil
}

type orderedMapTypeInfo interface {
	isOrderedMapTypeInfo()
}

type dictionaryOrderedMapTypeInfo DictionaryStaticType

func (dictionaryOrderedMapTypeInfo) isOrderedMapTypeInfo() {}

type compositeOrderedMapTypeInfo struct {
	location            common.Location
	qualifiedIdentifier string
	kind                common.CompositeKind
}

func (compositeOrderedMapTypeInfo) isOrderedMapTypeInfo() {}

func encodeDictionaryOrderedMapTypeInfo(dictionaryType DictionaryStaticType) cbor.RawMessage {
	var buf bytes.Buffer
	enc := CBOREncMode.NewStreamEncoder(&buf)

	err := enc.EncodeTagHead(CBORTagDictionaryValue)
	if err != nil {
		panic(ExternalError{err})
	}

	err = EncodeStaticType(enc, dictionaryType)
	if err != nil {
		panic(ExternalError{err})
	}

	err = enc.Flush()
	if err != nil {
		panic(ExternalError{err})
	}

	return buf.Bytes()
}

func encodeCompositeOrderedMapTypeInfo(
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
) cbor.RawMessage {
	var buf bytes.Buffer
	enc := CBOREncMode.NewStreamEncoder(&buf)

	err := enc.EncodeTagHead(CBORTagCompositeValue)
	if err != nil {
		panic(ExternalError{err})
	}

	err = encodeLocation(enc, location)
	if err != nil {
		panic(err)
	}

	err = enc.EncodeString(qualifiedIdentifier)
	if err != nil {
		panic(ExternalError{err})
	}

	err = enc.EncodeUint64(uint64(kind))
	if err != nil {
		panic(ExternalError{err})
	}

	err = enc.Flush()
	if err != nil {
		panic(ExternalError{err})
	}

	return buf.Bytes()
}

func decodeOrderedMapTypeInfo(typeInfo cbor.RawMessage) (orderedMapTypeInfo, error) {
	dec := CBORDecMode.NewByteStreamDecoder(typeInfo)

	tagNumber, err := dec.DecodeTagNumber()
	if err != nil {
		panic(ExternalError{err})
	}

	switch tagNumber {
	case CBORTagDictionaryValue:
		return decodeDictionaryOrderedMapTypeInfo(dec)

	case CBORTagCompositeValue:
		return decodeCompositeOrderedMapTypeInfo(dec)

	default:
		return nil, fmt.Errorf(
			"invalid array type info: expected tag %d, got %d",
			CBORTagArrayValue, tagNumber,
		)
	}
}

func decodeDictionaryOrderedMapTypeInfo(dec *cbor.StreamDecoder) (orderedMapTypeInfo, error) {
	staticType, err := decodeStaticType(dec)
	if err != nil {
		return nil, err
	}

	dictionaryType, ok := staticType.(DictionaryStaticType)
	if !ok {
		return nil, fmt.Errorf(
			"invalid array static type in array type info: %T",
			staticType,
		)
	}

	return dictionaryOrderedMapTypeInfo(dictionaryType), nil
}

func decodeCompositeOrderedMapTypeInfo(dec *cbor.StreamDecoder) (orderedMapTypeInfo, error) {
	location, err := decodeLocation(dec)
	if err != nil {
		panic(err)
	}

	qualifiedIdentifier, err := dec.DecodeString()
	if err != nil {
		return nil, err
	}

	kind, err := dec.DecodeUint64()
	if err != nil {
		return nil, err
	}

	if kind >= uint64(common.CompositeKindCount()) {
		return nil, fmt.Errorf(
			"invalid composite ordered map type info: invalid kind %d",
			kind,
		)
	}

	return compositeOrderedMapTypeInfo{
		location:            location,
		qualifiedIdentifier: qualifiedIdentifier,
		kind:                common.CompositeKind(kind),
	}, nil
}
