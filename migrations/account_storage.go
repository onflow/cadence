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

package migrations

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/runtime"
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

type StorageMapKeyMigrator interface {
	Migrate(
		inter *interpreter.Interpreter,
		storageKey interpreter.StorageKey,
		storageMap *interpreter.StorageMap,
		storageMapKey interpreter.StorageMapKey,
	)
	Domains() map[string]struct{}
}

type ValueConverter func(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	value interpreter.Value,
) interpreter.Value

type ValueConverterPathMigrator struct {
	domains      map[string]struct{}
	ConvertValue ValueConverter
}

var _ StorageMapKeyMigrator = ValueConverterPathMigrator{}

func NewValueConverterPathMigrator(
	domains map[string]struct{},
	convertValue ValueConverter,
) StorageMapKeyMigrator {
	return ValueConverterPathMigrator{
		domains:      domains,
		ConvertValue: convertValue,
	}
}

func (m ValueConverterPathMigrator) Migrate(
	inter *interpreter.Interpreter,
	storageKey interpreter.StorageKey,
	storageMap *interpreter.StorageMap,
	storageMapKey interpreter.StorageMapKey,
) {
	value := storageMap.ReadValue(nil, storageMapKey)

	newValue := m.ConvertValue(storageKey, storageMapKey, value)
	if newValue != nil {
		// If the converter returns a new value,
		// then replace the existing value with the new one.
		storageMap.SetValue(
			inter,
			storageMapKey,
			newValue,
		)
	}
}

func (m ValueConverterPathMigrator) Domains() map[string]struct{} {
	return m.domains
}

func (i *AccountStorage) MigrateStringKeys(
	inter *interpreter.Interpreter,
	key string,
	migrator StorageMapKeyMigrator,
) {
	i.MigrateStorageMap(
		inter,
		key,
		migrator,
		func(key atree.Value) interpreter.StorageMapKey {
			return interpreter.StringStorageMapKey(key.(interpreter.StringAtreeValue))
		},
	)
}

func (i *AccountStorage) MigrateUint64Keys(
	inter *interpreter.Interpreter,
	key string,
	migrator StorageMapKeyMigrator,
) {
	i.MigrateStorageMap(
		inter,
		key,
		migrator,
		func(key atree.Value) interpreter.StorageMapKey {
			return interpreter.Uint64StorageMapKey(key.(interpreter.Uint64AtreeValue))
		},
	)
}

func (i *AccountStorage) MigrateStorageMap(
	inter *interpreter.Interpreter,
	domain string,
	migrator StorageMapKeyMigrator,
	atreeKeyToStorageMapKey func(atree.Value) interpreter.StorageMapKey,
) {
	address := i.address

	if domains := migrator.Domains(); domains != nil {
		if _, ok := domains[domain]; !ok {
			return
		}
	}

	storageMap := i.storage.GetStorageMap(address, domain, false)
	if storageMap == nil || storageMap.Count() == 0 {
		return
	}

	storageKey := interpreter.NewStorageKey(nil, address, domain)

	iterator := storageMap.Iterator(inter)

	// Read the keys first, so the iteration won't be affected
	// by the modification of the storage values.
	var keys []interpreter.StorageMapKey
	for key, _ := iterator.Next(); key != nil; key, _ = iterator.Next() {
		identifier := atreeKeyToStorageMapKey(key)
		keys = append(keys, identifier)
	}

	for _, storageMapKey := range keys {

		migrator.Migrate(
			inter,
			storageKey,
			storageMap,
			storageMapKey,
		)
	}
}
