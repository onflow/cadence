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
	"github.com/onflow/cadence/interpreter"
)

type AccountStorageV1 struct {
	ledger      atree.Ledger
	slabStorage atree.SlabStorage
	memoryGauge common.MemoryGauge

	// newDomainStorageMapSlabIndices contains root slab index of new domain storage maps.
	// The indices are saved using Ledger.SetValue() during Commit().
	// Key is StorageKey{address, accountStorageKey} and value is 8-byte slab index.
	newDomainStorageMapSlabIndices *orderedmap.OrderedMap[interpreter.StorageKey, atree.SlabIndex]
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
	domain string,
	createIfNotExists bool,
) (
	domainStorageMap *interpreter.DomainStorageMap,
) {
	var err error
	domainStorageMap, err = getDomainStorageMapFromLegacyDomainRegister(
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
	domain string,
) *interpreter.DomainStorageMap {

	domainStorageMap := interpreter.NewDomainStorageMap(
		s.memoryGauge,
		s.slabStorage,
		atree.Address(address),
	)

	slabIndex := domainStorageMap.SlabID().Index()

	storageKey := interpreter.NewStorageKey(s.memoryGauge, address, domain)

	if s.newDomainStorageMapSlabIndices == nil {
		s.newDomainStorageMapSlabIndices = &orderedmap.OrderedMap[interpreter.StorageKey, atree.SlabIndex]{}
	}
	s.newDomainStorageMapSlabIndices.Set(storageKey, slabIndex)

	return domainStorageMap
}

func (s *AccountStorageV1) commit() error {
	if s.newDomainStorageMapSlabIndices == nil {
		return nil
	}

	return commitSlabIndices(
		s.newDomainStorageMapSlabIndices,
		s.ledger,
	)
}

// getDomainStorageMapFromLegacyDomainRegister returns domain storage map from legacy domain register.
func getDomainStorageMapFromLegacyDomainRegister(
	ledger atree.Ledger,
	storage atree.SlabStorage,
	address common.Address,
	domain string,
) (*interpreter.DomainStorageMap, error) {

	domainStorageSlabIndex, domainRegisterExists, err := getSlabIndexFromRegisterValue(
		ledger,
		address,
		[]byte(domain),
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
