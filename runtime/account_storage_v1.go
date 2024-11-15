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

package runtime

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/common/orderedmap"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type AccountStorageV1 struct {
	ledger      atree.Ledger
	slabStorage atree.SlabStorage
	memoryGauge common.MemoryGauge

	// newDomainStorageMapSlabIndices contains root slab indices of new domain storage maps.
	// The indices are saved using Ledger.SetValue() during commit().
	// Key is StorageDomainKey{common.StorageDomain, Address} and value is 8-byte slab index.
	newDomainStorageMapSlabIndices *orderedmap.OrderedMap[interpreter.StorageDomainKey, atree.SlabIndex]
}

func NewAccountStorageV1(
	ledger atree.Ledger,
	slabStorage atree.SlabStorage,
	memoryGauge common.MemoryGauge,
) *AccountStorageV1 {
	return &AccountStorageV1{
		ledger:      ledger,
		slabStorage: slabStorage,
		memoryGauge: memoryGauge,
	}
}

func (s *AccountStorageV1) GetDomainStorageMap(
	address common.Address,
	domain common.StorageDomain,
	createIfNotExists bool,
) (
	domainStorageMap *interpreter.DomainStorageMap,
) {
	var err error
	domainStorageMap, err = getDomainStorageMapFromV1DomainRegister(
		s.ledger,
		s.slabStorage,
		address,
		domain,
	)
	if err != nil {
		panic(err)
	}

	if domainStorageMap == nil && createIfNotExists {
		domainStorageMap = s.storeNewDomainStorageMap(address, domain)
	}

	return domainStorageMap
}

func (s *AccountStorageV1) storeNewDomainStorageMap(
	address common.Address,
	domain common.StorageDomain,
) *interpreter.DomainStorageMap {

	domainStorageMap := interpreter.NewDomainStorageMap(
		s.memoryGauge,
		s.slabStorage,
		atree.Address(address),
	)

	slabIndex := domainStorageMap.SlabID().Index()

	storageKey := interpreter.NewStorageDomainKey(s.memoryGauge, address, domain)

	if s.newDomainStorageMapSlabIndices == nil {
		s.newDomainStorageMapSlabIndices = &orderedmap.OrderedMap[interpreter.StorageDomainKey, atree.SlabIndex]{}
	}
	s.newDomainStorageMapSlabIndices.Set(storageKey, slabIndex)

	return domainStorageMap
}

func (s *AccountStorageV1) commit() error {
	if s.newDomainStorageMapSlabIndices == nil {
		return nil
	}

	for pair := s.newDomainStorageMapSlabIndices.Oldest(); pair != nil; pair = pair.Next() {
		var err error
		errors.WrapPanic(func() {
			err = s.ledger.SetValue(
				pair.Key.Address[:],
				[]byte(pair.Key.Domain.Identifier()),
				pair.Value[:],
			)
		})
		if err != nil {
			return interpreter.WrappedExternalError(err)
		}
	}

	return nil
}

// getDomainStorageMapFromV1DomainRegister returns domain storage map from legacy domain register.
func getDomainStorageMapFromV1DomainRegister(
	ledger atree.Ledger,
	storage atree.SlabStorage,
	address common.Address,
	domain common.StorageDomain,
) (*interpreter.DomainStorageMap, error) {

	domainStorageSlabIndex, domainRegisterExists, err := getSlabIndexFromRegisterValue(
		ledger,
		address,
		[]byte(domain.Identifier()),
	)
	if err != nil {
		return nil, err
	}
	if !domainRegisterExists {
		return nil, nil
	}

	slabID := atree.NewSlabID(
		atree.Address(address),
		domainStorageSlabIndex,
	)

	return interpreter.NewDomainStorageMapWithRootID(storage, slabID), nil
}
