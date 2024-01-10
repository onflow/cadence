/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package migrations

import (
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
)

type AccountStorage struct {
	storage *runtime.Storage
	address common.Address
}

// NewAccountStorage constructs an `AccountStorage` for a given account.
func NewAccountStorage(storage *runtime.Storage, address common.Address) AccountStorage {
	return AccountStorage{
		storage: storage,
		address: address,
	}
}

type PathMigrator func(
	inter *interpreter.Interpreter,
	storageMap *interpreter.StorageMap,
	storageKey interpreter.StringStorageMapKey,
	addressPath interpreter.AddressPath,
)

type ValueConverter func(
	addressPath interpreter.AddressPath,
	value interpreter.Value,
) interpreter.Value

func NewValueConverterPathMigrator(convertValue ValueConverter) PathMigrator {
	return func(
		inter *interpreter.Interpreter,
		storageMap *interpreter.StorageMap,
		storageKey interpreter.StringStorageMapKey,
		addressPath interpreter.AddressPath,
	) {
		value := storageMap.ReadValue(nil, storageKey)

		newValue := convertValue(addressPath, value)
		if newValue != nil {
			// If the converter returns a new value,
			// then replace the existing value with the new one.
			storageMap.SetValue(
				inter,
				storageKey,
				newValue,
			)
		}
	}
}

func (i *AccountStorage) MigratePathsInDomain(
	inter *interpreter.Interpreter,
	domain common.PathDomain,
	migratePath PathMigrator,
) {
	storageMap := i.storage.GetStorageMap(i.address, domain.Identifier(), false)
	if storageMap == nil || storageMap.Count() == 0 {
		return
	}

	iterator := storageMap.Iterator(inter)

	// Read the keys first, so the iteration wouldn't be affected
	// by the modification of the storage values.
	var keys []string
	for key, _ := iterator.Next(); key != nil; key, _ = iterator.Next() {
		identifier := string(key.(interpreter.StringAtreeValue))
		keys = append(keys, identifier)
	}

	for _, key := range keys {
		storageKey := interpreter.StringStorageMapKey(key)

		path := interpreter.PathValue{
			Identifier: key,
			Domain:     domain,
		}

		addressPath := interpreter.AddressPath{
			Address: i.address,
			Path:    path,
		}

		migratePath(
			inter,
			storageMap,
			storageKey,
			addressPath,
		)
	}
}

func (i *AccountStorage) MigratePathsInDomains(
	inter *interpreter.Interpreter,
	domains []common.PathDomain,
	migratePath PathMigrator,
) {
	for _, domain := range domains {
		i.MigratePathsInDomain(
			inter,
			domain,
			migratePath,
		)
	}
}
