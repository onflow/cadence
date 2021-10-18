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
	AccountValues map[StorageKey]Value
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
		AccountValues:    make(map[StorageKey]Value),
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

type compositeTypeInfo struct {
	location            common.Location
	qualifiedIdentifier string
	kind                common.CompositeKind
}

var _ atree.TypeInfo = compositeTypeInfo{}

const encodedCompositeTypeInfoLength = 3

func (c compositeTypeInfo) Encode(e *cbor.StreamEncoder) error {
	err := e.EncodeRawBytes([]byte{
		// tag number
		0xd8, CBORTagCompositeValue,
		// array, 3 items follow
		0x83,
	})
	if err != nil {
		return err
	}

	err = encodeLocation(e, c.location)
	if err != nil {
		panic(err)
	}

	err = e.EncodeString(c.qualifiedIdentifier)
	if err != nil {
		return err
	}

	err = e.EncodeUint64(uint64(c.kind))
	if err != nil {
		return err
	}

	return nil
}

func (c compositeTypeInfo) Equal(o atree.TypeInfo) bool {
	other, ok := o.(compositeTypeInfo)
	return ok &&
		common.LocationsMatch(c.location, other.location) &&
		c.qualifiedIdentifier == other.qualifiedIdentifier &&
		c.kind == other.kind
}

func decodeCompositeTypeInfo(dec *cbor.StreamDecoder) (atree.TypeInfo, error) {

	length, err := dec.DecodeArrayHead()
	if err != nil {
		return nil, err
	}

	if length != encodedCompositeTypeInfoLength {
		return nil, fmt.Errorf(
			"invalid composite type info: expected %d elements, got %d",
			encodedCompositeTypeInfoLength, length,
		)
	}

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

	return compositeTypeInfo{
		location:            location,
		qualifiedIdentifier: qualifiedIdentifier,
		kind:                common.CompositeKind(kind),
	}, nil
}

type stringAtreeValue string

var _ atree.Value = stringAtreeValue("")
var _ atree.Storable = stringAtreeValue("")

func (v stringAtreeValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (v stringAtreeValue) ByteSize() uint32 {
	return getBytesCBORSize([]byte(v))
}

func (v stringAtreeValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func stringAtreeHashInput(v atree.Value, _ []byte) ([]byte, error) {
	return []byte(v.(stringAtreeValue)), nil
}

func stringAtreeComparator(_ atree.SlabStorage, v atree.Value, o atree.Storable) (bool, error) {
	return v.(stringAtreeValue) == o.(stringAtreeValue), nil
}
