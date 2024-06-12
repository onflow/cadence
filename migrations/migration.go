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
	"fmt"
	"runtime/debug"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser/lexer"
	"github.com/onflow/cadence/runtime/stdlib"
)

type ValueMigration interface {
	Name() string
	Migrate(
		storageKey interpreter.StorageKey,
		storageMapKey interpreter.StorageMapKey,
		value interpreter.Value,
		interpreter *interpreter.Interpreter,
		position ValueMigrationPosition,
	) (newValue interpreter.Value, err error)
	CanSkip(valueType interpreter.StaticType) bool
	Domains() map[string]struct{}
}

type ValueMigrationPosition uint8

const (
	ValueMigrationPositionOther ValueMigrationPosition = iota
	ValueMigrationPositionDictionaryKey
)

type DomainMigration interface {
	Name() string
	Migrate(
		addressPath interpreter.AddressPath,
	)
}

type StorageMigration struct {
	storage                *runtime.Storage
	interpreter            *interpreter.Interpreter
	name                   string
	address                common.Address
	dictionaryKeyConflicts int
	stacktraceEnabled      bool
}

func NewStorageMigration(
	interpreter *interpreter.Interpreter,
	storage *runtime.Storage,
	name string,
	address common.Address,
) (
	*StorageMigration,
	error,
) {
	if !lexer.IsValidIdentifier(name) {
		return nil, fmt.Errorf("invalid migration name: %s", name)
	}

	return &StorageMigration{
		storage:                storage,
		interpreter:            interpreter,
		name:                   name,
		address:                address,
		dictionaryKeyConflicts: 0,
	}, nil
}

func (m *StorageMigration) WithErrorStacktrace(stacktraceEnabled bool) *StorageMigration {
	m.stacktraceEnabled = stacktraceEnabled
	return m
}

func (m *StorageMigration) Commit() error {
	return m.storage.NondeterministicCommit(m.interpreter, false)
}

func (m *StorageMigration) Migrate(migrator StorageMapKeyMigrator) {
	accountStorage := NewAccountStorage(m.storage, m.address)

	for _, domain := range common.AllPathDomains {
		accountStorage.MigrateStringKeys(
			m.interpreter,
			domain.Identifier(),
			migrator,
		)
	}

	accountStorage.MigrateStringKeys(
		m.interpreter,
		stdlib.InboxStorageDomain,
		migrator,
	)

	accountStorage.MigrateStringKeys(
		m.interpreter,
		runtime.StorageDomainContract,
		migrator,
	)

	accountStorage.MigrateUint64Keys(
		m.interpreter,
		stdlib.CapabilityControllerStorageDomain,
		migrator,
	)

	accountStorage.MigrateStringKeys(
		m.interpreter,
		stdlib.PathCapabilityStorageDomain,
		migrator,
	)

	accountStorage.MigrateUint64Keys(
		m.interpreter,
		stdlib.AccountCapabilityStorageDomain,
		migrator,
	)
}

func (m *StorageMigration) NewValueMigrationsPathMigrator(
	reporter Reporter,
	valueMigrations ...ValueMigration,
) StorageMapKeyMigrator {

	// Gather all domains that have to be migrated
	// from all value migrations

	var allDomains map[string]struct{}

	if len(valueMigrations) == 1 {
		// Optimization: Avoid allocating a new map
		allDomains = valueMigrations[0].Domains()
	} else {
		for _, valueMigration := range valueMigrations {
			migrationDomains := valueMigration.Domains()
			if migrationDomains == nil {
				continue
			}
			if allDomains == nil {
				allDomains = make(map[string]struct{})
			}
			// Safe to iterate, as the order does not matter
			for domain := range migrationDomains { //nolint:maprange
				allDomains[domain] = struct{}{}
			}
		}
	}

	return NewValueConverterPathMigrator(
		allDomains,
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
				true,
				ValueMigrationPositionOther,
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
	allowMutation bool,
	position ValueMigrationPosition,
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

			var stack []byte
			if m.stacktraceEnabled {
				stack = debug.Stack()
			}

			err = StorageMigrationError{
				StorageKey:    storageKey,
				StorageMapKey: storageMapKey,
				Migration:     m.name,
				Err:           err,
				Stack:         stack,
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
			allowMutation,
			ValueMigrationPositionOther,
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
				allowMutation,
				ValueMigrationPositionOther,
			)

			if newElement == nil {
				continue
			}

			// We should only check if we're allowed to mutate if we actually are going to mutate,
			// i.e. if newValue != nil. It might be the case that none of the values need to be migrated,
			// in which case we should not panic with an error that we're not allowed to mutate

			if !allowMutation {
				panic(errors.NewUnexpectedError(
					"mutation not allowed: attempting to migrate array element at index %d: %s",
					index,
					element,
				))
			}

			existingStorable := array.RemoveWithoutTransfer(
				inter,
				emptyLocationRange,
				index,
			)

			interpreter.StoredValue(inter, existingStorable, m.storage).
				DeepRemove(inter, false)
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

			newValue := m.MigrateNestedValue(
				storageKey,
				storageMapKey,
				existingValue,
				valueMigrations,
				reporter,
				allowMutation,
				ValueMigrationPositionOther,
			)

			if newValue == nil {
				continue
			}

			// We should only check if we're allowed to mutate if we actually are going to mutate,
			// i.e. if newValue != nil. It might be the case that none of the values need to be migrated,
			// in which case we should not panic with an error that we're not allowed to mutate

			if !allowMutation {
				panic(errors.NewUnexpectedError(
					"mutation not allowed: attempting to migrate composite value field %s: %s",
					fieldName,
					existingValue,
				))
			}

			composite.SetMemberWithoutTransfer(
				inter,
				emptyLocationRange,
				fieldName,
				newValue,
			)
		}

	case *interpreter.DictionaryValue:
		dictionary := typedValue

		// Dictionaries are migrated in two passes:
		// First, the keys are migrated, then the values.
		//
		// This is necessary because in the atree register inlining version,
		// only the read-only iterator is able to read old keys,
		// as they potentially have different hash values.
		// The mutating iterator is only able to read new keys,
		// as it recalculates the stored values' hashes.

		m.migrateDictionaryKeys(
			storageKey,
			storageMapKey,
			dictionary,
			valueMigrations,
			reporter,
			allowMutation,
		)

		m.migrateDictionaryValues(
			storageKey,
			storageMapKey,
			dictionary,
			valueMigrations,
			reporter,
			allowMutation,
		)

	case *interpreter.PublishedValue:
		publishedValue := typedValue
		newInnerValue := m.MigrateNestedValue(
			storageKey,
			storageMapKey,
			publishedValue.Value,
			valueMigrations,
			reporter,
			allowMutation,
			ValueMigrationPositionOther,
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
			position,
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

func (m *StorageMigration) migrateDictionaryKeys(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	dictionary *interpreter.DictionaryValue,
	valueMigrations []ValueMigration,
	reporter Reporter,
	allowMutation bool,
) {
	inter := m.interpreter

	var existingKeys []interpreter.Value

	dictionary.IterateReadOnly(
		inter,
		emptyLocationRange,
		func(key, _ interpreter.Value) (resume bool) {

			existingKeys = append(existingKeys, key)

			// Continue iteration
			return true
		},
	)

	for _, existingKey := range existingKeys {

		newKey := m.MigrateNestedValue(
			storageKey,
			storageMapKey,
			existingKey,
			valueMigrations,
			reporter,
			// NOTE: Mutation of keys is not allowed.
			false,
			ValueMigrationPositionDictionaryKey,
		)

		if newKey == nil {
			continue
		}

		// We should only check if we're allowed to mutate if we actually are going to mutate,
		// i.e. if newKey != nil. It might be the case that none of the keys need to be migrated,
		// in which case we should not panic with an error that we're not allowed to mutate

		if !allowMutation {
			panic(errors.NewUnexpectedError(
				"mutation not allowed: attempting to migrate dictionary key: %s",
				existingKey,
			))
		}

		// We only reach here because key needs to be migrated.

		// Remove the old key-value pair.

		var existingKeyStorable, existingValueStorable atree.Storable

		legacyKey := LegacyKey(existingKey)
		if legacyKey != nil {
			existingKeyStorable, existingValueStorable = dictionary.RemoveWithoutTransfer(
				inter,
				emptyLocationRange,
				legacyKey,
			)
		}
		if existingKeyStorable == nil {
			existingKeyStorable, existingValueStorable = dictionary.RemoveWithoutTransfer(
				inter,
				emptyLocationRange,
				existingKey,
			)
		}
		if existingKeyStorable == nil {
			panic(errors.NewUnexpectedError(
				"failed to remove old value for migrated key: %s",
				existingKey,
			))
		}

		// Remove existing key since old key is migrated
		interpreter.StoredValue(inter, existingKeyStorable, m.storage).
			DeepRemove(inter, false)
		inter.RemoveReferencedSlab(existingKeyStorable)

		// Convert removed value storable to Value.
		existingValue := interpreter.StoredValue(inter, existingValueStorable, m.storage)

		// Handle dictionary key conflicts.
		//
		// If the dictionary contains the key/value pairs
		// - key1: value1
		// - key2: value2
		//
		// then key1 is migrated to key1_migrated, and value1 is migrated to value1_migrated.
		//
		// If key1_migrated happens to be equal to key2, then we have a conflict.
		//
		// Check if the key to set already exists.
		//
		// - If it already exists, leave it as is, and store the migrated key-value pair
		//   into a new dictionary under a new unique storage path, and report it.
		//
		//   The new key that already exists, key2, was already or will be migrated,
		//   so we must NOT handle it here (e.g. remove it from the dictionary).
		//
		// - If it does not exist, insert the migrated key-value pair normally.

		// NOTE: Do NOT attempt to change the logic here to instead remove newKey
		// and move it to the new dictionary instead!

		if dictionary.ContainsKey(
			inter,
			emptyLocationRange,
			newKey,
		) {
			newValue := m.MigrateNestedValue(
				storageKey,
				storageMapKey,
				existingValue,
				valueMigrations,
				reporter,
				allowMutation,
				ValueMigrationPositionOther,
			)

			var valueToSet interpreter.Value
			if newValue == nil {
				valueToSet = existingValue
			} else {
				valueToSet = newValue

				// Remove existing value since value is migrated.
				existingValue.DeepRemove(inter, false)
				inter.RemoveReferencedSlab(existingValueStorable)
			}

			owner := dictionary.GetOwner()

			pathDomain := common.PathDomainStorage

			storageMap := m.storage.GetStorageMap(owner, pathDomain.Identifier(), true)
			conflictDictionary := interpreter.NewDictionaryValueWithAddress(
				inter,
				emptyLocationRange,
				dictionary.Type,
				owner,
			)
			conflictDictionary.InsertWithoutTransfer(
				inter,
				emptyLocationRange,
				newKey,
				valueToSet,
			)

			conflictStorageMapKey := m.nextDictionaryKeyConflictStorageMapKey()

			addressPath := interpreter.AddressPath{
				Address: owner,
				Path: interpreter.PathValue{
					Domain:     pathDomain,
					Identifier: string(conflictStorageMapKey),
				},
			}

			if storageMap.ValueExists(conflictStorageMapKey) {
				panic(errors.NewUnexpectedError(
					"conflict storage map key already exists: %s", addressPath,
				))
			}

			storageMap.SetValue(
				inter,
				conflictStorageMapKey,
				conflictDictionary,
			)

			reporter.DictionaryKeyConflict(addressPath)

		} else {

			// No conflict, insert the new key and existing value pair
			// Don't migrate value here because we are going to migrate all values in the dictionary next.

			dictionary.InsertWithoutTransfer(
				inter,
				emptyLocationRange,
				newKey,
				existingValue,
			)
		}
	}
}

func (m *StorageMigration) migrateDictionaryValues(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	dictionary *interpreter.DictionaryValue,
	valueMigrations []ValueMigration,
	reporter Reporter,
	allowMutation bool,
) {

	inter := m.interpreter

	type keyValuePair struct {
		key, value interpreter.Value
	}

	var existingKeysAndValues []keyValuePair

	dictionary.Iterate(
		inter,
		emptyLocationRange,
		func(key, value interpreter.Value) (resume bool) {

			existingKeysAndValues = append(
				existingKeysAndValues,
				keyValuePair{
					key:   key,
					value: value,
				},
			)

			// Continue iteration
			return true
		},
	)

	for _, existingKeyAndValue := range existingKeysAndValues {
		existingKey := existingKeyAndValue.key
		existingValue := existingKeyAndValue.value

		newValue := m.MigrateNestedValue(
			storageKey,
			storageMapKey,
			existingValue,
			valueMigrations,
			reporter,
			allowMutation,
			ValueMigrationPositionOther,
		)

		if newValue == nil {
			continue
		}

		// We should only check if we're allowed to mutate if we actually are going to mutate,
		// i.e. if newValue != nil. It might be the case that none of the values need to be migrated,
		// in which case we should not panic with an error that we're not allowed to mutate

		if !allowMutation {
			panic(errors.NewUnexpectedError(
				"mutation not allowed: attempting to migrate dictionary value: %s",
				existingValue,
			))
		}

		// Set new value with existing key in the dictionary.
		existingValueStorable := dictionary.InsertWithoutTransfer(
			inter,
			emptyLocationRange,
			existingKey,
			newValue,
		)
		if existingValueStorable == nil {
			panic(errors.NewUnexpectedError(
				"failed to set migrated value for key: %s",
				existingKey,
			))
		}

		// Remove existing value since value is migrated
		interpreter.StoredValue(inter, existingValueStorable, m.storage).
			DeepRemove(inter, false)
		inter.RemoveReferencedSlab(existingValueStorable)
	}
}

func (m *StorageMigration) nextDictionaryKeyConflictStorageMapKey() interpreter.StringStorageMapKey {
	m.dictionaryKeyConflicts++
	return m.DictionaryKeyConflictStorageMapKey(m.dictionaryKeyConflicts)
}

func (m *StorageMigration) DictionaryKeyConflictStorageMapKey(index int) interpreter.StringStorageMapKey {
	return interpreter.StringStorageMapKey(fmt.Sprintf(
		"cadence1_%s_dictionaryKeyConflict_%d",
		m.name,
		index,
	))
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
	position ValueMigrationPosition,
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

			var stack []byte
			if m.stacktraceEnabled {
				stack = debug.Stack()
			}

			err = StorageMigrationError{
				StorageKey:    storageKey,
				StorageMapKey: storageMapKey,
				Migration:     migration.Name(),
				Err:           err,
				Stack:         stack,
			}
		}
	}()

	return migration.Migrate(
		storageKey,
		storageMapKey,
		value,
		m.interpreter,
		position,
	)
}

// LegacyKey return the same type with the "old" hash/ID generation function.
func LegacyKey(key interpreter.Value) interpreter.Value {
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

	return nil
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
		optionalType := typ

		legacyInnerType := legacyType(typ.Type)
		if legacyInnerType != nil {
			optionalType = interpreter.NewOptionalStaticType(nil, legacyInnerType)
		}

		return &LegacyOptionalType{
			OptionalStaticType: optionalType,
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
