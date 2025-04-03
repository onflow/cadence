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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type GetDomainStorageMapFunc func(
	ledger atree.Ledger,
	storage atree.SlabStorage,
	address common.Address,
	domain common.StorageDomain,
) (
	*interpreter.DomainStorageMap,
	error,
)

// DomainRegisterMigration migrates domain registers to account storage maps.
type DomainRegisterMigration struct {
	ledger              atree.Ledger
	storage             atree.SlabStorage
	context             interpreter.ValueTransferContext
	memoryGauge         common.MemoryGauge
	getDomainStorageMap GetDomainStorageMapFunc
}

func NewDomainRegisterMigration(
	ledger atree.Ledger,
	storage atree.SlabStorage,
	context interpreter.ValueTransferContext,
	memoryGauge common.MemoryGauge,
	getDomainStorageMap GetDomainStorageMapFunc,
) *DomainRegisterMigration {
	if getDomainStorageMap == nil {
		getDomainStorageMap = getDomainStorageMapFromV1DomainRegister
	}
	return &DomainRegisterMigration{
		ledger:              ledger,
		storage:             storage,
		context:             context,
		memoryGauge:         memoryGauge,
		getDomainStorageMap: getDomainStorageMap,
	}
}

func (m *DomainRegisterMigration) MigrateAccount(
	address common.Address,
) (
	*interpreter.AccountStorageMap,
	error,
) {
	exists, err := hasAccountStorageMap(m.ledger, address)
	if err != nil {
		return nil, err
	}
	if exists {
		// Account storage map already exists
		return nil, nil
	}

	// Migrate existing domains
	accountStorageMap, err := m.migrateDomainRegisters(address)
	if err != nil {
		return nil, err
	}

	if accountStorageMap == nil {
		// Nothing migrated
		return nil, nil
	}

	slabIndex := accountStorageMap.SlabID().Index()

	// Write account register
	errors.WrapPanic(func() {
		err = m.ledger.SetValue(
			address[:],
			[]byte(AccountStorageKey),
			slabIndex[:],
		)
	})
	if err != nil {
		return nil, interpreter.WrappedExternalError(err)
	}

	return accountStorageMap, nil
}

// migrateDomainRegisters migrates all existing domain storage maps to a new account storage map,
// and removes the domain registers.
func (m *DomainRegisterMigration) migrateDomainRegisters(
	address common.Address,
) (
	*interpreter.AccountStorageMap,
	error,
) {

	var accountStorageMap *interpreter.AccountStorageMap

	for _, domain := range common.AllStorageDomains {

		domainStorageMap, err := m.getDomainStorageMap(
			m.ledger,
			m.storage,
			address,
			domain,
		)
		if err != nil {
			return nil, err
		}

		if domainStorageMap == nil {
			// Skip non-existent domain
			continue
		}

		if accountStorageMap == nil {
			accountStorageMap = interpreter.NewAccountStorageMap(
				m.memoryGauge,
				m.storage,
				atree.Address(address),
			)
		}

		// Migrate (insert) existing domain storage map to account storage map
		existed := accountStorageMap.WriteDomain(m.context, domain, domainStorageMap)
		if existed {
			// This shouldn't happen because we are inserting domain storage map into empty account storage map.
			return nil, errors.NewUnexpectedError(
				"failed to migrate domain %s for account %x: domain already exists in account storage map",
				domain.Identifier(),
				address,
			)
		}

		// Remove migrated domain registers
		errors.WrapPanic(func() {
			// NOTE: removing non-existent domain registers is no-op.
			err = m.ledger.SetValue(
				address[:],
				[]byte(domain.Identifier()),
				nil)
		})
		if err != nil {
			return nil, interpreter.WrappedExternalError(err)
		}
	}

	return accountStorageMap, nil
}
