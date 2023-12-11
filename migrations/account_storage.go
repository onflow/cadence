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

// ForEachValue iterates over the values in the account.
// The `valueConverter takes a function to be applied to each value.
// It returns the converted, if a new value was created during conversion.
func (i *AccountStorage) ForEachValue(
	inter *interpreter.Interpreter,
	domains []common.PathDomain,
	valueConverter func(
		value interpreter.Value,
		address common.Address,
		domain common.PathDomain,
		key string,
	) interpreter.Value,
) {
	for _, domain := range domains {
		storageMap := i.storage.GetStorageMap(i.address, domain.Identifier(), false)
		if storageMap == nil || storageMap.Count() == 0 {
			continue
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

			value := storageMap.ReadValue(nil, storageKey)

			newValue := valueConverter(value, i.address, domain, key)
			if newValue == nil {
				continue
			}

			// If the converter returns a new value, then replace the existing value with the new one.
			storageMap.SetValue(
				inter,
				storageKey,
				newValue,
			)
		}
	}
}
