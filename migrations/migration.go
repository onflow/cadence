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
	"fmt"
	"runtime/debug"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/stdlib"
)

type ValueMigration interface {
	Name() string
	Migrate(
		storageKey interpreter.StorageKey,
		storageMapKey interpreter.StorageMapKey,
		value interpreter.Value,
		interpreter *interpreter.Interpreter,
	) (newValue interpreter.Value, err error)
	CanSkip(valueType interpreter.StaticType) bool
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
	migrate StorageMapKeyMigrator,
) {
	for {
		address := addressIterator.NextAddress()
		if address == common.ZeroAddress {
			break
		}

		m.MigrateAccount(address, migrate)
	}
}

func (m *StorageMigration) Commit() error {
	return m.storage.Commit(m.interpreter, false)
}

func (m *StorageMigration) MigrateAccount(
	address common.Address,
	migrate StorageMapKeyMigrator,
) {
	accountStorage := NewAccountStorage(m.storage, address)

	for _, domain := range common.AllPathDomains {
		accountStorage.MigrateStringKeys(
			m.interpreter,
			domain.Identifier(),
			migrate,
		)
	}

	accountStorage.MigrateStringKeys(
		m.interpreter,
		stdlib.InboxStorageDomain,
		migrate,
	)

	accountStorage.MigrateStringKeys(
		m.interpreter,
		runtime.StorageDomainContract,
		migrate,
	)

	accountStorage.MigrateUint64Keys(
		m.interpreter,
		stdlib.CapabilityControllerStorageDomain,
		migrate,
	)
}

func (m *StorageMigration) NewValueMigrationsPathMigrator(
	reporter Reporter,
	valueMigrations ...ValueMigration,
) StorageMapKeyMigrator {
	return NewValueConverterPathMigrator(
		func(
			storageKey interpreter.StorageKey,
			storageMapKey interpreter.StorageMapKey,
			value interpreter.Value,
		) interpreter.Value {
			return m.MigrateNestedValue(
				storageKey,
				storageMapKey,
				value,
				valueMigrations,
				reporter,
			)
		},
	)
}

var emptyLocationRange = interpreter.EmptyLocationRange

func (m *StorageMigration) MigrateNestedValue(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	value interpreter.Value,
	valueMigrations []ValueMigration,
	reporter Reporter,
) (migratedValue interpreter.Value) {

	defer func() {
		// Here it catches the panics that may occur at the framework level,
		// even before going to each individual migration. e.g: iterating the dictionary for elements.
		//
		// There is a similar recovery at the `StorageMigration.migrate()` method,
		// which handles panics from each individual migrations (e.g: capcon migration, static type migration, etc.).

		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}

			err = StorageMigrationError{
				StorageKey:    storageKey,
				StorageMapKey: storageMapKey,
				Migration:     "StorageMigration",
				Err:           err,
				Stack:         debug.Stack(),
			}

			if reporter != nil {
				reporter.Error(err)
			}
		}
	}()

	inter := m.interpreter

	// skip the migration of the value,
	// if all value migrations agree

	canSkip := true
	staticType := value.StaticType(inter)
	for _, migration := range valueMigrations {
		if !migration.CanSkip(staticType) {
			canSkip = false
			break
		}
	}

	if canSkip {
		return
	}

	// Visit the children first, and migrate them.
	// i.e: depth-first traversal
	switch typedValue := value.(type) {
	case *interpreter.SomeValue:
		innerValue := typedValue.InnerValue(inter, emptyLocationRange)
		newInnerValue := m.MigrateNestedValue(
			storageKey,
			storageMapKey,
			innerValue,
			valueMigrations,
			reporter,
		)
		if newInnerValue != nil {
			migratedValue = interpreter.NewSomeValueNonCopying(inter, newInnerValue)

			// chain the migrations
			value = migratedValue
		}

	case *interpreter.ArrayValue:
		array := typedValue

		// Migrate array elements
		count := array.Count()
		for index := 0; index < count; index++ {

			element := array.Get(inter, emptyLocationRange, index)

			newElement := m.MigrateNestedValue(
				storageKey,
				storageMapKey,
				element,
				valueMigrations,
				reporter,
			)

			if newElement == nil {
				continue
			}

			existingStorable := array.RemoveWithoutTransfer(
				inter,
				emptyLocationRange,
				index,
			)

			interpreter.StoredValue(inter, existingStorable, m.storage).
				DeepRemove(inter)
			inter.RemoveReferencedSlab(existingStorable)

			array.InsertWithoutTransfer(
				inter,
				emptyLocationRange,
				index,
				newElement,
			)
		}

	case *interpreter.CompositeValue:
		composite := typedValue

		// Read the field names first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var fieldNames []string
		composite.ForEachFieldName(func(fieldName string) (resume bool) {
			fieldNames = append(fieldNames, fieldName)
			return true
		})

		for _, fieldName := range fieldNames {
			existingValue := composite.GetField(
				inter,
				emptyLocationRange,
				fieldName,
			)

			migratedValue := m.MigrateNestedValue(
				storageKey,
				storageMapKey,
				existingValue,
				valueMigrations,
				reporter,
			)

			if migratedValue == nil {
				continue
			}

			composite.SetMemberWithoutTransfer(
				inter,
				emptyLocationRange,
				fieldName,
				migratedValue,
			)
		}

	case *interpreter.DictionaryValue:
		dictionary := typedValue

		type keyValuePair struct {
			key, value interpreter.Value
		}

		// Read the keys first, so the iteration wouldn't be affected
		// by the modification of the nested values.
		var existingKeysAndValues []keyValuePair

		iterator := dictionary.Iterator()

		for {
			key, value := iterator.Next(nil)
			if key == nil {
				break
			}

			existingKeysAndValues = append(
				existingKeysAndValues,
				keyValuePair{
					key:   key,
					value: value,
				},
			)
		}

		for _, existingKeyAndValue := range existingKeysAndValues {
			existingKey := existingKeyAndValue.key
			existingValue := existingKeyAndValue.value

			newKey := m.MigrateNestedValue(
				storageKey,
				storageMapKey,
				existingKey,
				valueMigrations,
				reporter,
			)

			newValue := m.MigrateNestedValue(
				storageKey,
				storageMapKey,
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
				keyToSet = newKey
			}

			existingKey = legacyKey(existingKey)
			existingKeyStorable, existingValueStorable := dictionary.RemoveWithoutTransfer(
				inter,
				emptyLocationRange,
				existingKey,
			)

			if existingKeyStorable == nil {
				panic(errors.NewUnexpectedError(
					"failed to remove old value for migrated key: %s",
					existingKey,
				))
			}

			if newValue == nil {
				valueToSet = existingValue
			} else {
				// Value was migrated
				valueToSet = newValue

				interpreter.StoredValue(inter, existingValueStorable, m.storage).
					DeepRemove(inter)
				inter.RemoveReferencedSlab(existingValueStorable)
			}

			dictionary.InsertWithoutTransfer(
				inter,
				emptyLocationRange,
				keyToSet,
				valueToSet,
			)
		}

	case *interpreter.PublishedValue:
		publishedValue := typedValue
		newInnerValue := m.MigrateNestedValue(
			storageKey,
			storageMapKey,
			publishedValue.Value,
			valueMigrations,
			reporter,
		)
		if newInnerValue != nil {
			newInnerCapability := newInnerValue.(interpreter.CapabilityValue)
			migratedValue = interpreter.NewPublishedValue(
				inter,
				publishedValue.Recipient,
				newInnerCapability,
			)

			// chain the migrations
			value = migratedValue
		}
	}

	// Once the children are migrated, then migrate the current/wrapper value.
	// Result of each migration is passed as the input to the next migration.
	// i.e: A single value is migrated by all the migrations, before moving onto the next value.

	for _, migration := range valueMigrations {
		convertedValue, err := m.migrate(
			migration,
			storageKey,
			storageMapKey,
			value,
		)

		if err != nil {
			if reporter != nil {
				if _, ok := err.(StorageMigrationError); !ok {
					err = StorageMigrationError{
						StorageKey:    storageKey,
						StorageMapKey: storageMapKey,
						Migration:     migration.Name(),
						Err:           err,
					}
				}

				reporter.Error(err)
			}
			continue
		}

		if convertedValue != nil {

			// Sanity check: ensure that the owner of the new value
			// is the same as the owner of the old value
			if ownedValue, ok := value.(interpreter.OwnedValue); ok {
				if ownedConvertedValue, ok := convertedValue.(interpreter.OwnedValue); ok {
					convertedOwner := ownedConvertedValue.GetOwner()
					originalOwner := ownedValue.GetOwner()
					if convertedOwner != originalOwner {
						panic(errors.NewUnexpectedError(
							"migrated value has different owner: expected %s, got %s",
							originalOwner,
							convertedOwner,
						))
					}
				}
			}

			// Chain the migrations.
			value = convertedValue

			migratedValue = convertedValue

			if reporter != nil {
				reporter.Migrated(
					storageKey,
					storageMapKey,
					migration.Name(),
				)
			}
		}
	}
	return

}

type StorageMigrationError struct {
	StorageKey    interpreter.StorageKey
	StorageMapKey interpreter.StorageMapKey
	Migration     string
	Err           error
	Stack         []byte
}

func (e StorageMigrationError) Error() string {
	return fmt.Sprintf(
		"failed to perform migration %s for %s, %s: %s\n%s",
		e.Migration,
		e.StorageKey,
		e.StorageMapKey,
		e.Err.Error(),
		e.Stack,
	)
}

func (m *StorageMigration) migrate(
	migration ValueMigration,
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	value interpreter.Value,
) (
	converted interpreter.Value,
	err error,
) {

	// Handles panics from each individual migrations (e.g: capcon migration, static type migration, etc.).
	// So even if one migration panics, others could still run (i.e: panics are caught inside the loop).
	// Removing that would cause all migrations to stop for a particular value, if one of them panics.
	// NOTE: this won't catch panics occur at the migration framework level.
	// They are caught at `StorageMigration.MigrateNestedValue()`.
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}

			err = StorageMigrationError{
				StorageKey:    storageKey,
				StorageMapKey: storageMapKey,
				Migration:     migration.Name(),
				Err:           err,
				Stack:         debug.Stack(),
			}
		}
	}()

	return migration.Migrate(
		storageKey,
		storageMapKey,
		value,
		m.interpreter,
	)
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

	case interpreter.CharacterValue:
		return &LegacyCharacterValue{
			CharacterValue: key,
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

	case interpreter.PrimitiveStaticType:
		switch typ {
		case interpreter.PrimitiveStaticTypeAuthAccount, //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccount,                  //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountCapabilities,        //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountCapabilities,      //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities, //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities, //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountContracts,           //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountContracts,         //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountKeys,                //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountKeys,              //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountInbox,               //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAccountKey:                     //nolint:staticcheck
			return LegacyPrimitiveStaticType{
				PrimitiveStaticType: typ,
			}
		}
	}

	return nil
}
