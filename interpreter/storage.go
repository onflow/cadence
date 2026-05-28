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

// AtreeContainerCache deduplicates Cadence-level wrappers (ArrayValue,
// DictionaryValue, CompositeValue) created for atree containers, keyed by
// their atree value ID. See SharedState.canonicalAtreeContainers for the
// rationale.
type AtreeContainerCache interface {
	CanonicalAtreeContainer(valueID atree.ValueID) Value
	SetCanonicalAtreeContainer(valueID atree.ValueID, v Value)
	ClearCanonicalAtreeContainer(valueID atree.ValueID)
}

// canonicalizeContainerElement returns the canonical cached wrapper for a
// container element, populating the cache on first sight or adopting the
// freshly-loaded `*atree.Array`/`*atree.OrderedMap` into the existing
// cached wrapper. atree's `Array.Get`/`OrderedMap.Get` sets up the
// parent updater (via setCallbackWithChild) on the
// `*atree.Array`/`*atree.OrderedMap` instance it just returned, not on
// the one we may have previously cached, so the freshly-returned
// instance is the one that will correctly notify the parent on
// mutation.
func canonicalizeContainerElement(cache AtreeContainerCache, fresh Value) Value {
	switch v := fresh.(type) {
	case *ArrayValue:
		if v.array == nil || v.isDestroyed {
			return fresh
		}
		if existing, ok := cache.CanonicalAtreeContainer(v.valueID).(*ArrayValue); ok {
			if existing.array != nil && !existing.isDestroyed {
				existing.array = v.array
				return existing
			}
			cache.ClearCanonicalAtreeContainer(v.valueID)
		}
		cache.SetCanonicalAtreeContainer(v.valueID, v)
		return v
	case *DictionaryValue:
		if v.dictionary == nil || v.isDestroyed {
			return fresh
		}
		if existing, ok := cache.CanonicalAtreeContainer(v.valueID).(*DictionaryValue); ok {
			if existing.dictionary != nil && !existing.isDestroyed {
				existing.dictionary = v.dictionary
				return existing
			}
			cache.ClearCanonicalAtreeContainer(v.valueID)
		}
		cache.SetCanonicalAtreeContainer(v.valueID, v)
		return v
	case *CompositeValue:
		if v.dictionary == nil || v.isDestroyed {
			return fresh
		}
		if existing, ok := cache.CanonicalAtreeContainer(v.valueID).(*CompositeValue); ok {
			if existing.dictionary != nil && !existing.isDestroyed {
				existing.dictionary = v.dictionary
				return existing
			}
			cache.ClearCanonicalAtreeContainer(v.valueID)
		}
		cache.SetCanonicalAtreeContainer(v.valueID, v)
		return v
	case *SomeValue:
		// An optional wrapping a container must canonicalize its inner so
		// that aliased references see a shared wrapper. SomeStorable.
		// StoredValue produces a fresh SomeValue per load whose inner is
		// built via the non-canonicalizing StoredValue path (it has no
		// access to the current context's cache); re-canonicalize the
		// inner here so the SomeValue we return contains the canonical
		// wrapper. The recursion also handles nested optionals (T??, ...).
		if v.value == nil {
			return fresh
		}
		canonicalized := canonicalizeContainerElement(cache, v.value)
		if canonicalized != v.value {
			v.value = canonicalized
		}
		return v
	}
	return fresh
}

// MustConvertStoredContainerElement wraps an atree value retrieved as a
// container element (e.g. via `*atree.Array.Get` or
// `*atree.OrderedMap.Get`) as a Cadence-level `Value`, deduplicating the
// resulting wrapper via the canonical wrapper cache when supported. Use
// this instead of `MustConvertStoredValue` whenever a container's
// element is being returned to user code (e.g. for `&outer[0]`), so that
// aliased references share state. Internal callers that immediately
// Transfer (and thereby invalidate) the wrapper must continue to use
// `MustConvertStoredValue` so their transient wrapper does not poison
// the cache.
func MustConvertStoredContainerElement(gauge common.MemoryGauge, value atree.Value) Value {
	result := MustConvertStoredValue(gauge, value)
	if cache, ok := gauge.(AtreeContainerCache); ok {
		return canonicalizeContainerElement(cache, result)
	}
	return result
}

// ConvertStoredValue wraps the given atree value as a Cadence-level
// `Value` without canonicalization. Callers that return the wrapper to
// user code as a container element (e.g. `&outer[0]`) should use
// `MustConvertStoredContainerElement` instead, so aliased references
// see a shared wrapper.
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
	computationGauge  common.ComputationGauge
}

var _ Storage = InMemoryStorage{}

func NewInMemoryStorage(
	memoryGauge common.MemoryGauge,
	computationGauge common.ComputationGauge,
) InMemoryStorage {

	decodeStorable := func(
		decoder *cbor.StreamDecoder,
		storableSlabStorageID atree.SlabID,
		inlinedExtraData []atree.ExtraData,
	) (atree.Storable, error) {
		return DecodeStorable(
			decoder,
			storableSlabStorageID,
			inlinedExtraData,
			memoryGauge,
		)
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
		computationGauge:  computationGauge,
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
		domainStorageMap = NewDomainStorageMap(
			i.memoryGauge,
			i.computationGauge,
			i,
			atree.Address(address),
		)
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
