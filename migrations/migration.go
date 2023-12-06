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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
)

type Migration interface {
	Name() string
	Migrate(
		addressPath interpreter.AddressPath,
		value interpreter.Value,
		interpreter *interpreter.Interpreter,
	) (newValue interpreter.Value)
}

type StorageMigration struct {
	storage     *runtime.Storage
	interpreter *interpreter.Interpreter
}

func NewStorageMigration(
	interpreter *interpreter.Interpreter,
	storage *runtime.Storage,
) *StorageMigration {
	return &StorageMigration{
		storage:     storage,
		interpreter: interpreter,
	}
}

func (m *StorageMigration) Migrate(
	addressIterator AddressIterator,
	reporter Reporter,
	migrations ...Migration,
) {
	for {
		address := addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.migrateValuesInAccount(
			address,
			reporter,
			migrations,
		)
	}

	err := m.storage.Commit(m.interpreter, false)
	if err != nil {
		panic(err)
	}
}

func (m *StorageMigration) migrateValuesInAccount(
	address common.Address,
	reporter Reporter,
	migrations []Migration,
) {

	accountStorage := NewAccountStorage(m.storage, address)

	migrateValue := func(
		addressPath interpreter.AddressPath,
		value interpreter.Value,
	) interpreter.Value {
		return m.migrateNestedValue(addressPath, value, migrations, reporter)
	}

	accountStorage.ForEachValue(
		m.interpreter,
		common.AllPathDomains,
		migrateValue,
	)
}

var emptyLocationRange = interpreter.EmptyLocationRange

func (m *StorageMigration) migrateNestedValue(
	addressPath interpreter.AddressPath,
	value interpreter.Value,
	migrations []Migration,
	reporter Reporter,
) (newValue interpreter.Value) {
	switch value := value.(type) {
	case *interpreter.SomeValue:
		innerValue := value.InnerValue(m.interpreter, emptyLocationRange)
		newInnerValue := m.migrateNestedValue(addressPath, innerValue, migrations, reporter)
		if newInnerValue != nil {
			return interpreter.NewSomeValueNonCopying(m.interpreter, newInnerValue)
		}

		return

	case *interpreter.ArrayValue:
		array := value

		// Migrate array elements
		count := array.Count()
		for index := 0; index < count; index++ {
			element := array.Get(m.interpreter, emptyLocationRange, index)
			newElement := m.migrateNestedValue(addressPath, element, migrations, reporter)
			if newElement != nil {
				array.Set(
					m.interpreter,
					emptyLocationRange,
					index,
					newElement,
				)
			}
		}

		// The array itself doesn't need to be replaced.
		return

	case *interpreter.CompositeValue:
		composite := value

		// Read the field names first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var fieldNames []string
		composite.ForEachField(nil, func(fieldName string, fieldValue interpreter.Value) (resume bool) {
			fieldNames = append(fieldNames, fieldName)
			return true
		})

		for _, fieldName := range fieldNames {
			existingValue := composite.GetField(m.interpreter, interpreter.EmptyLocationRange, fieldName)

			migratedValue := m.migrateNestedValue(addressPath, existingValue, migrations, reporter)
			if migratedValue == nil {
				continue
			}

			composite.SetMember(m.interpreter, emptyLocationRange, fieldName, migratedValue)
		}

		// The composite itself does not have to be replaced
		return

	case *interpreter.DictionaryValue:
		dictionary := value

		// Read the keys first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var existingKeys []interpreter.Value
		dictionary.Iterate(m.interpreter, func(key, _ interpreter.Value) (resume bool) {
			existingKeys = append(existingKeys, key)
			return true
		})

		for _, existingKey := range existingKeys {
			existingValue, exist := dictionary.Get(nil, interpreter.EmptyLocationRange, existingKey)
			if !exist {
				panic(errors.NewUnreachableError())
			}

			newKey := m.migrateNestedValue(addressPath, existingKey, migrations, reporter)
			newValue := m.migrateNestedValue(addressPath, existingValue, migrations, reporter)
			if newKey == nil && newValue == nil {
				continue
			}

			// We only reach here at least one of key or value has been migrated.
			var keyToSet, valueToSet interpreter.Value

			if newKey == nil {
				keyToSet = existingKey
			} else {
				// Key was migrated.
				// Remove the old value at the old key.
				// This old value will be inserted again with the new key, unless the value is also migrated.
				_ = dictionary.RemoveKey(m.interpreter, emptyLocationRange, existingKey)
				keyToSet = newKey
			}

			if newValue == nil {
				valueToSet = existingValue
			} else {
				// Value was migrated
				valueToSet = newValue
			}

			// Always wrap with an optional, when inserting to the dictionary.
			valueToSet = interpreter.NewUnmeteredSomeValueNonCopying(valueToSet)

			dictionary.SetKey(m.interpreter, emptyLocationRange, keyToSet, valueToSet)
		}

		// The dictionary itself does not have to be replaced
		return
	default:
		// Assumption: all migrations only migrate non-container typed values.
		for _, migration := range migrations {
			converted := migration.Migrate(addressPath, value, m.interpreter)

			if converted != nil {
				// Chain the migrations.
				// Probably not needed, because of the assumption above.
				// i.e: A single non-container value may not get converted from two migrations.
				// But have it here to be safe.
				value = converted

				newValue = converted

				if reporter != nil {
					reporter.Report(addressPath, migration.Name())
				}
			}
		}

		return
	}
}
