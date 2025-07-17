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
	"bytes"
	"cmp"
	"io"
	"math"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/onflow/atree"

	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/values"

	"github.com/onflow/cadence/common"
)

func StoredValue(gauge common.MemoryGauge, storable atree.Storable, storage atree.SlabStorage) Value {
	storedValue, err := storable.StoredValue(storage)
	if err != nil {
		panic(err)
	}

	return MustConvertStoredValue(gauge, storedValue)
}

func MustConvertStoredValue(gauge common.MemoryGauge, value atree.Value) Value {
	converted, err := ConvertStoredValue(gauge, value)
	if err != nil {
		panic(err)
	}
	return converted
}

func MustConvertUnmeteredStoredValue(value atree.Value) Value {
	converted, err := ConvertStoredValue(nil, value)
	if err != nil {
		panic(err)
	}
	return converted
}

func ConvertStoredValue(gauge common.MemoryGauge, value atree.Value) (Value, error) {
	switch value := value.(type) {
	case *atree.Array:
		staticType, ok := value.Type().(ArrayStaticType)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		return newArrayValueFromAtreeArray(
			gauge,
			staticType,
			ArrayElementSize(staticType),
			value,
		), nil

	case *atree.OrderedMap:
		typeInfo := value.Type()
		switch staticType := typeInfo.(type) {
		case *DictionaryStaticType:
			return newDictionaryValueFromAtreeMap(
				gauge,
				staticType,
				DictionaryElementSize(staticType),
				value,
			), nil

		case CompositeTypeInfo:
			return NewCompositeValueFromAtreeMap(
				gauge,
				staticType,
				value,
			), nil

		default:
			return nil, errors.NewUnexpectedError("invalid ordered map type info: %T", staticType)
		}

	case values.BoolValue:
		return BoolValue(value), nil

	case values.IntValue:
		return IntValue{IntValue: value}, nil

	case values.UFix64Value:
		return UFix64Value{UFix64Value: value}, nil

	case Value:
		return value, nil

	default:
		return nil, errors.NewUnexpectedError("cannot convert stored value: %T", value)
	}
}

type StorageDomainKey struct {
	Domain  common.StorageDomain
	Address common.Address
}

func (k StorageDomainKey) Compare(o StorageDomainKey) int {
	switch bytes.Compare(k.Address[:], o.Address[:]) {
	case -1:
		return -1
	case 0:
		return cmp.Compare(k.Domain, o.Domain)
	case 1:
		return 1
	default:
		panic(errors.NewUnreachableError())
	}
}

func NewStorageDomainKey(
	memoryGauge common.MemoryGauge,
	address common.Address,
	domain common.StorageDomain,
) StorageDomainKey {
	common.UseMemory(memoryGauge, common.StorageKeyMemoryUsage)
	return StorageDomainKey{
		Address: address,
		Domain:  domain,
	}
}

type StorageKey struct {
	Key     string
	Address common.Address
}

func NewStorageKey(memoryGauge common.MemoryGauge, address common.Address, key string) StorageKey {
	common.UseMemory(memoryGauge, common.StorageKeyMemoryUsage)
	return StorageKey{
		Address: address,
		Key:     key,
	}
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
type InMemoryStorage struct {
	*atree.BasicSlabStorage
	DomainStorageMaps map[StorageDomainKey]*DomainStorageMap
	memoryGauge       common.MemoryGauge
}

var _ Storage = InMemoryStorage{}

func NewInMemoryStorage(memoryGauge common.MemoryGauge) InMemoryStorage {
	decodeStorable := func(
		decoder *cbor.StreamDecoder,
		storableSlabStorageID atree.SlabID,
		inlinedExtraData []atree.ExtraData,
	) (atree.Storable, error) {
		return DecodeStorable(decoder, storableSlabStorageID, inlinedExtraData, memoryGauge)
	}

	decodeTypeInfo := func(decoder *cbor.StreamDecoder) (atree.TypeInfo, error) {
		return DecodeTypeInfo(decoder, memoryGauge)
	}

	slabStorage := atree.NewBasicSlabStorage(
		CBOREncMode,
		CBORDecMode,
		decodeStorable,
		decodeTypeInfo,
	)

	return InMemoryStorage{
		BasicSlabStorage:  slabStorage,
		DomainStorageMaps: make(map[StorageDomainKey]*DomainStorageMap),
		memoryGauge:       memoryGauge,
	}
}

func (i InMemoryStorage) GetDomainStorageMap(
	_ StorageMutationTracker,
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) (
	domainStorageMap *DomainStorageMap,
) {
	key := NewStorageDomainKey(i.memoryGauge, address, domain)
	domainStorageMap = i.DomainStorageMaps[key]
	if domainStorageMap == nil && createIfNotExists {
		domainStorageMap = NewDomainStorageMap(i.memoryGauge, i, atree.Address(address))
		i.DomainStorageMaps[key] = domainStorageMap
	}
	return domainStorageMap
}

func (i InMemoryStorage) CheckHealth() error {
	_, err := atree.CheckStorageHealth(i, -1)
	return err
}

// writeCounter is an io.Writer which counts the amount of written data.
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
func mustStorableSize(storable atree.Storable) uint32 {
	size, err := StorableSize(storable)
	if err != nil {
		panic(err)
	}
	return size
}

// StorableSize returns the size of the storable in bytes.
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
		return 0, errors.NewUnexpectedError("storable size is too large: expected max uint32, got %d", size)
	}

	return uint32(size), nil
}
