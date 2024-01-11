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

type ValueMigration interface {
	Name() string
	Migrate(
		addressPath interpreter.AddressPath,
		value interpreter.Value,
		interpreter *interpreter.Interpreter,
	) (newValue interpreter.Value, err error)
}

type DomainMigration interface {
	Name() string
	Migrate(
		addressPath interpreter.AddressPath,
	)
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
	migratePath PathMigrator,
) {
	for {
		address := addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.MigrateAccount(address, migratePath)
	}
}

func (m *StorageMigration) Commit() error {
	return m.storage.Commit(m.interpreter, false)
}

func (m *StorageMigration) MigrateAccount(
	address common.Address,
	migratePath PathMigrator,
) {
	accountStorage := NewAccountStorage(m.storage, address)

	accountStorage.MigratePathsInDomains(
		m.interpreter,
		common.AllPathDomains,
		migratePath,
	)
}

func (m *StorageMigration) NewValueMigrationsPathMigrator(
	reporter Reporter,
	valueMigrations ...ValueMigration,
) PathMigrator {
	return NewValueConverterPathMigrator(
		func(addressPath interpreter.AddressPath, value interpreter.Value) interpreter.Value {
			return m.migrateNestedValue(
				addressPath,
				value,
				valueMigrations,
				reporter,
			)
		},
	)
}

var emptyLocationRange = interpreter.EmptyLocationRange

func (m *StorageMigration) migrateNestedValue(
	addressPath interpreter.AddressPath,
	value interpreter.Value,
	valueMigrations []ValueMigration,
	reporter Reporter,
) (newValue interpreter.Value) {
	switch value := value.(type) {
	case *interpreter.SomeValue:
		innerValue := value.InnerValue(m.interpreter, emptyLocationRange)
		newInnerValue := m.migrateNestedValue(
			addressPath,
			innerValue,
			valueMigrations,
			reporter,
		)
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
			newElement := m.migrateNestedValue(
				addressPath,
				element,
				valueMigrations,
				reporter,
			)
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
		composite.ForEachFieldName(func(fieldName string) (resume bool) {
			fieldNames = append(fieldNames, fieldName)
			return true
		})

		for _, fieldName := range fieldNames {
			existingValue := composite.GetField(
				m.interpreter,
				emptyLocationRange,
				fieldName,
			)

			migratedValue := m.migrateNestedValue(
				addressPath,
				existingValue,
				valueMigrations,
				reporter,
			)
			if migratedValue == nil {
				continue
			}

			composite.SetMember(
				m.interpreter,
				emptyLocationRange,
				fieldName,
				migratedValue,
			)
		}

		// The composite itself does not have to be replaced
		return

	case *interpreter.DictionaryValue:
		dictionary := value

		type keyValuePair struct {
			key, value interpreter.Value
		}

		// Read the keys first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var existingKeysAndValues []keyValuePair
		dictionary.Iterate(m.interpreter, func(key, value interpreter.Value) (resume bool) {
			existingKeysAndValues = append(
				existingKeysAndValues,
				keyValuePair{
					key:   key,
					value: value,
				},
			)

			// continue iteration
			return true
		})

		for _, existingKeyAndValue := range existingKeysAndValues {
			existingKey := existingKeyAndValue.key
			existingValue := existingKeyAndValue.value

			newKey := m.migrateNestedValue(
				addressPath,
				existingKey,
				valueMigrations,
				reporter,
			)

			newValue := m.migrateNestedValue(
				addressPath,
				existingValue,
				valueMigrations,
				reporter,
			)

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
				existingKey = legacyKey(existingKey)
				oldValue := dictionary.Remove(
					m.interpreter,
					emptyLocationRange,
					existingKey,
				)

				if _, ok := oldValue.(*interpreter.SomeValue); !ok {
					panic(errors.NewUnreachableError())
				}

				keyToSet = newKey
			}

			if newValue == nil {
				valueToSet = existingValue
			} else {
				// Value was migrated
				valueToSet = newValue
			}

			dictionary.Insert(
				m.interpreter,
				emptyLocationRange,
				keyToSet,
				valueToSet,
			)
		}

		// The dictionary itself does not have to be replaced
		return
	default:
		// Assumption: all migrations only migrate non-container typed values.
		for _, migration := range valueMigrations {
			converted, err := m.migrate(migration, addressPath, value)

			if err != nil {
				if reporter != nil {
					reporter.Error(addressPath, migration.Name(), err)
				}
				continue
			}

			if converted != nil {
				// Chain the migrations.
				// Probably not needed, because of the assumption above.
				// i.e: A single non-container value may not get converted from two migrations.
				// But have it here to be safe.
				value = converted

				newValue = converted

				if reporter != nil {
					reporter.Migrated(addressPath, migration.Name())
				}
			}
		}

		return
	}
}

func (m *StorageMigration) migrate(
	migration ValueMigration,
	addressPath interpreter.AddressPath,
	value interpreter.Value,
) (converted interpreter.Value, err error) {

	defer func() {
		if r := recover(); r != nil {
			switch r := r.(type) {
			case error:
				err = r
			default:
				panic(r)
			}
		}
	}()

	return migration.Migrate(addressPath, value, m.interpreter)
}

// legacyKey return the same type with the "old" hash/ID generation function.
func legacyKey(key interpreter.Value) interpreter.Value {
	switch key := key.(type) {
	case interpreter.TypeValue:
		legacyType := legacyType(key.Type)
		if legacyType != nil {
			return interpreter.NewUnmeteredTypeValue(legacyType)
		}

	case *interpreter.StringValue:
		return &LegacyStringValue{
			StringValue: key,
		}
	}

	return key
}

func legacyType(staticType interpreter.StaticType) interpreter.StaticType {
	switch typ := staticType.(type) {
	case *interpreter.IntersectionStaticType:
		return &LegacyIntersectionType{
			IntersectionStaticType: typ,
		}

	case *interpreter.ConstantSizedStaticType:
		legacyType := legacyType(typ.Type)
		if legacyType != nil {
			return interpreter.NewConstantSizedStaticType(nil, legacyType, typ.Size)
		}

	case *interpreter.VariableSizedStaticType:
		legacyType := legacyType(typ.Type)
		if legacyType != nil {
			return interpreter.NewVariableSizedStaticType(nil, legacyType)
		}

	case *interpreter.DictionaryStaticType:
		legacyKeyType := legacyType(typ.KeyType)
		legacyValueType := legacyType(typ.ValueType)
		if legacyKeyType != nil && legacyValueType != nil {
			return interpreter.NewDictionaryStaticType(nil, legacyKeyType, legacyValueType)
		}
		if legacyKeyType != nil {
			return interpreter.NewDictionaryStaticType(nil, legacyKeyType, typ.ValueType)
		}
		if legacyValueType != nil {
			return interpreter.NewDictionaryStaticType(nil, typ.KeyType, legacyValueType)
		}

	case *interpreter.OptionalStaticType:
		legacyInnerType := legacyType(typ.Type)
		if legacyInnerType != nil {
			return interpreter.NewOptionalStaticType(nil, legacyInnerType)
		}

	case *interpreter.CapabilityStaticType:
		legacyBorrowType := legacyType(typ.BorrowType)
		if legacyBorrowType != nil {
			return interpreter.NewCapabilityStaticType(nil, legacyBorrowType)
		}

	case *interpreter.ReferenceStaticType:
		referenceType := typ

		legacyReferencedType := legacyType(typ.ReferencedType)
		if legacyReferencedType != nil {
			referenceType = interpreter.NewReferenceStaticType(nil, typ.Authorization, legacyReferencedType)
		}

		return &LegacyReferenceType{
			ReferenceStaticType: referenceType,
		}
	}

	return nil
}
