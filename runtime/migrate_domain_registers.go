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

type GetDomainStorageMapFunc func(
	ledger atree.Ledger,
	storage atree.SlabStorage,
	address common.Address,
	domain string,
) (*interpreter.DomainStorageMap, error)

type DomainRegisterMigration struct {
	ledger              atree.Ledger
	storage             atree.SlabStorage
	inter               *interpreter.Interpreter
	memoryGauge         common.MemoryGauge
	getDomainStorageMap GetDomainStorageMapFunc
}

func NewDomainRegisterMigration(
	ledger atree.Ledger,
	storage atree.SlabStorage,
	inter *interpreter.Interpreter,
	memoryGauge common.MemoryGauge,
) *DomainRegisterMigration {
	return &DomainRegisterMigration{
		ledger:              ledger,
		storage:             storage,
		inter:               inter,
		memoryGauge:         memoryGauge,
		getDomainStorageMap: getDomainStorageMapFromLegacyDomainRegister,
	}
}

// SetGetDomainStorageMapFunc allows user to provide custom GetDomainStorageMap function.
func (m *DomainRegisterMigration) SetGetDomainStorageMapFunc(
	getDomainStorageMapFunc GetDomainStorageMapFunc,
) {
	m.getDomainStorageMap = getDomainStorageMapFunc
}

// MigrateAccounts migrates given accounts.
func (m *DomainRegisterMigration) MigrateAccounts(
	accounts *orderedmap.OrderedMap[common.Address, struct{}],
	pred func(common.Address) bool,
) (
	*orderedmap.OrderedMap[common.Address, *interpreter.AccountStorageMap],
	error,
) {
	if accounts == nil || accounts.Len() == 0 {
		return nil, nil
	}

	var migratedAccounts *orderedmap.OrderedMap[common.Address, *interpreter.AccountStorageMap]

	for pair := accounts.Oldest(); pair != nil; pair = pair.Next() {
		address := pair.Key

		if !pred(address) {
			continue
		}

		migrated, err := isMigrated(m.ledger, address)
		if err != nil {
			return nil, err
		}
		if migrated {
			continue
		}

		accountStorageMap, err := m.MigrateAccount(address)
		if err != nil {
			return nil, err
		}

		if accountStorageMap == nil {
			continue
		}

		if migratedAccounts == nil {
			migratedAccounts = &orderedmap.OrderedMap[common.Address, *interpreter.AccountStorageMap]{}
		}
		migratedAccounts.Set(address, accountStorageMap)
	}

	return migratedAccounts, nil
}

func (m *DomainRegisterMigration) MigrateAccount(
	address common.Address,
) (*interpreter.AccountStorageMap, error) {

	// Migrate existing domains
	accountStorageMap, err := m.migrateDomains(address)
	if err != nil {
		return nil, err
	}

	if accountStorageMap == nil {
		// Nothing migrated
		return nil, nil
	}

	accountStorageMapSlabIndex := accountStorageMap.SlabID().Index()

	// Write account register
	errors.WrapPanic(func() {
		err = m.ledger.SetValue(
			address[:],
			[]byte(AccountStorageKey),
			accountStorageMapSlabIndex[:],
		)
	})
	if err != nil {
		return nil, interpreter.WrappedExternalError(err)
	}

	return accountStorageMap, nil
}

// migrateDomains migrates existing domain storage maps and removes domain registers.
func (m *DomainRegisterMigration) migrateDomains(
	address common.Address,
) (*interpreter.AccountStorageMap, error) {

	var accountStorageMap *interpreter.AccountStorageMap

	for _, domain := range AccountDomains {

		domainStorageMap, err := m.getDomainStorageMap(m.ledger, m.storage, address, domain)
		if err != nil {
			return nil, err
		}

		if domainStorageMap == nil {
			// Skip non-existent domain
			continue
		}

		if accountStorageMap == nil {
			accountStorageMap = interpreter.NewAccountStorageMap(m.memoryGauge, m.storage, atree.Address(address))
		}

		// Migrate (insert) existing domain storage map to account storage map
		existed := accountStorageMap.WriteDomain(m.inter, domain, domainStorageMap)
		if existed {
			// This shouldn't happen because we are inserting domain storage map into empty account storage map.
			return nil, errors.NewUnexpectedError(
				"failed to migrate domain %s for account %x: domain already exists in account storage map", domain, address,
			)
		}

		// Remove migrated domain registers
		errors.WrapPanic(func() {
			// NOTE: removing non-existent domain registers is no-op.
			err = m.ledger.SetValue(
				address[:],
				[]byte(domain),
				nil)
		})
		if err != nil {
			return nil, interpreter.WrappedExternalError(err)
		}
	}

	return accountStorageMap, nil
}

func isMigrated(ledger atree.Ledger, address common.Address) (bool, error) {
	_, registerExists, err := getSlabIndexFromRegisterValue(ledger, address, []byte(AccountStorageKey))
	if err != nil {
		return false, err
	}
	return registerExists, nil
}
