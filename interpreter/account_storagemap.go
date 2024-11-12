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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
)

// AccountStorageMap stores domain storage maps in an account.
type AccountStorageMap struct {
	orderedMap *atree.OrderedMap
}

// NewAccountStorageMap creates account storage map.
func NewAccountStorageMap(
	memoryGauge common.MemoryGauge,
	storage atree.SlabStorage,
	address atree.Address,
) *AccountStorageMap {
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

	return &AccountStorageMap{
		orderedMap: orderedMap,
	}
}

// NewAccountStorageMapWithRootID loads existing account storage map with given atree SlabID.
func NewAccountStorageMapWithRootID(
	storage atree.SlabStorage,
	slabID atree.SlabID,
) *AccountStorageMap {
	orderedMap, err := atree.NewMapWithRootID(
		storage,
		slabID,
		atree.NewDefaultDigesterBuilder(),
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &AccountStorageMap{
		orderedMap: orderedMap,
	}
}

// DomainExists returns true if the given domain exists in the account storage map.
func (s *AccountStorageMap) DomainExists(domain common.StorageDomain) bool {
	key := Uint64StorageMapKey(domain)

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

// GetDomain returns domain storage map for the given domain.
// If createIfNotExists is true and domain doesn't exist, new domain storage map
// is created and inserted into account storage map with given domain as key.
func (s *AccountStorageMap) GetDomain(
	gauge common.MemoryGauge,
	interpreter *Interpreter,
	domain common.StorageDomain,
	createIfNotExists bool,
) *DomainStorageMap {
	key := Uint64StorageMapKey(domain)

	storedValue, err := s.orderedMap.Get(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			// Create domain storage map if needed.

			if createIfNotExists {
				return s.NewDomain(gauge, interpreter, domain)
			}

			return nil
		}

		panic(errors.NewExternalError(err))
	}

	// Create domain storage map from raw atree value.
	return NewDomainStorageMapWithAtreeValue(storedValue)
}

// NewDomain creates new domain storage map and inserts it to AccountStorageMap with given domain as key.
func (s *AccountStorageMap) NewDomain(
	gauge common.MemoryGauge,
	interpreter *Interpreter,
	domain common.StorageDomain,
) *DomainStorageMap {
	interpreter.recordStorageMutation()

	domainStorageMap := NewDomainStorageMap(gauge, s.orderedMap.Storage, s.orderedMap.Address())

	key := Uint64StorageMapKey(domain)

	existingStorable, err := s.orderedMap.Set(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
		domainStorageMap.orderedMap,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	if existingStorable != nil {
		panic(errors.NewUnexpectedError(
			"account %x domain %s should not exist",
			s.orderedMap.Address(),
			domain.Identifier(),
		))
	}

	return domainStorageMap
}

// WriteDomain sets or removes domain storage map in account storage map.
// If the given storage map is nil, domain is removed.
// If the given storage map is non-nil, domain is added/updated.
// Returns true if domain storage map previously existed at the given domain.
func (s *AccountStorageMap) WriteDomain(
	interpreter *Interpreter,
	domain common.StorageDomain,
	storageMap *DomainStorageMap,
) (existed bool) {
	if storageMap == nil {
		return s.removeDomain(interpreter, domain)
	}
	return s.setDomain(interpreter, domain, storageMap)
}

// setDomain sets domain storage map in the account storage map and returns true if domain previously existed.
// If the given domain already stores a domain storage map, it is overwritten.
func (s *AccountStorageMap) setDomain(
	interpreter *Interpreter,
	domain common.StorageDomain,
	storageMap *DomainStorageMap,
) (existed bool) {
	interpreter.recordStorageMutation()

	key := Uint64StorageMapKey(domain)

	existingValueStorable, err := s.orderedMap.Set(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
		storageMap.orderedMap,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	existed = existingValueStorable != nil
	if existed {
		// Create domain storage map from overwritten storable
		domainStorageMap := newDomainStorageMapWithAtreeStorable(s.orderedMap.Storage, existingValueStorable)

		// Deep remove elements in domain storage map
		domainStorageMap.DeepRemove(interpreter, true)

		// Remove domain storage map slab
		interpreter.RemoveReferencedSlab(existingValueStorable)
	}

	interpreter.maybeValidateAtreeValue(s.orderedMap)

	// NOTE: Don't call maybeValidateAtreeStorage() here because it is possible
	// that domain storage map is in the process of being migrated to account
	// storage map and state isn't consistent during migration.

	return
}

// removeDomain removes domain storage map with given domain in account storage map, if it exists.
func (s *AccountStorageMap) removeDomain(interpreter *Interpreter, domain common.StorageDomain) (existed bool) {
	interpreter.recordStorageMutation()

	key := Uint64StorageMapKey(domain)

	existingKeyStorable, existingValueStorable, err := s.orderedMap.Remove(
		key.AtreeValueCompare,
		key.AtreeValueHashInput,
		key.AtreeValue(),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			// No-op to remove non-existent domain.
			return
		}
		panic(errors.NewExternalError(err))
	}

	// Key

	// NOTE: Key is just an atree.Value (Uint64AtreeValue), not an interpreter.Value,
	// so do not need (can) convert and not need to deep remove
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	existed = existingValueStorable != nil
	if existed {
		// Create domain storage map from removed storable
		domainStorageMap := newDomainStorageMapWithAtreeStorable(s.orderedMap.Storage, existingValueStorable)

		// Deep remove elements in domain storage map
		domainStorageMap.DeepRemove(interpreter, true)

		// Remove domain storage map slab
		interpreter.RemoveReferencedSlab(existingValueStorable)
	}

	interpreter.maybeValidateAtreeValue(s.orderedMap)
	interpreter.maybeValidateAtreeStorage()

	return
}

func (s *AccountStorageMap) SlabID() atree.SlabID {
	return s.orderedMap.SlabID()
}

func (s *AccountStorageMap) Count() uint64 {
	return s.orderedMap.Count()
}

// Domains returns a set of domains in account storage map
func (s *AccountStorageMap) Domains() map[common.StorageDomain]struct{} {
	domains := make(map[common.StorageDomain]struct{})

	iterator := s.Iterator()

	for {
		k, err := iterator.mapIterator.NextKey()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if k == nil {
			break
		}

		domain := convertKeyToDomain(k)
		domains[domain] = struct{}{}
	}

	return domains
}

// Iterator returns a mutable iterator (AccountStorageMapIterator),
// which allows iterating over the domain and domain storage map.
func (s *AccountStorageMap) Iterator() *AccountStorageMapIterator {
	mapIterator, err := s.orderedMap.Iterator(
		StorageMapKeyAtreeValueComparator,
		StorageMapKeyAtreeValueHashInput,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &AccountStorageMapIterator{
		mapIterator: mapIterator,
		storage:     s.orderedMap.Storage,
	}
}

// AccountStorageMapIterator is an iterator over AccountStorageMap.
type AccountStorageMapIterator struct {
	mapIterator atree.MapIterator
	storage     atree.SlabStorage
}

// Next returns the next domain and domain storage map.
// If there is no more domain, (common.StorageDomainUnknown, nil) is returned.
func (i *AccountStorageMapIterator) Next() (common.StorageDomain, *DomainStorageMap) {
	k, v, err := i.mapIterator.Next()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if k == nil || v == nil {
		return common.StorageDomainUnknown, nil
	}

	key := convertKeyToDomain(k)

	value := NewDomainStorageMapWithAtreeValue(v)

	return key, value
}

func convertKeyToDomain(v atree.Value) common.StorageDomain {
	key, ok := v.(Uint64AtreeValue)
	if !ok {
		panic(errors.NewUnexpectedError("domain key type %T isn't expected", key))
	}
	domain, err := common.StorageDomainFromUint64(uint64(key))
	if err != nil {
		panic(errors.NewUnexpectedError("domain key %d isn't expected: %w", key, err))
	}
	return domain
}
