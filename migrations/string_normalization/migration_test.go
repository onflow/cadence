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

package string_normalization

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestStringNormalizingMigration(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}
	pathDomain := common.PathDomainPublic

	type testCase struct {
		storedValue   interpreter.Value
		expectedValue interpreter.Value
	}

	ledger := NewTestLedger(nil, nil)
	storage := runtime.NewStorage(ledger, nil)
	locationRange := interpreter.EmptyLocationRange

	inter, err := interpreter.NewInterpreter(
		nil,
		utils.TestLocation,
		&interpreter.Config{
			Storage:                       storage,
			AtreeValueValidationEnabled:   false,
			AtreeStorageValidationEnabled: false,
		},
	)
	require.NoError(t, err)

	newLegacyStringValue := func(s string) *interpreter.StringValue {
		return &interpreter.StringValue{
			Str:             s,
			UnnormalizedStr: s,
		}
	}

	newLegacyCharacterValue := func(s string) interpreter.CharacterValue {
		return interpreter.CharacterValue{
			Str:             s,
			UnnormalizedStr: s,
		}
	}

	testCases := map[string]testCase{
		"normalized_string": {
			storedValue:   newLegacyStringValue("Caf\u00E9"),
			expectedValue: interpreter.NewUnmeteredStringValue("Caf\u00E9"),
		},
		"un-normalized_string": {
			storedValue:   newLegacyStringValue("Cafe\u0301"),
			expectedValue: interpreter.NewUnmeteredStringValue("Caf\u00E9"),
		},
		"normalized_character": {
			storedValue:   newLegacyCharacterValue("\u03A9"),
			expectedValue: interpreter.NewUnmeteredCharacterValue("\u03A9"),
		},
		"un-normalized_character": {
			storedValue:   newLegacyCharacterValue("\u2126"),
			expectedValue: interpreter.NewUnmeteredCharacterValue("\u03A9"),
		},
		"string_array": {
			storedValue: interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct),
				common.ZeroAddress,
				newLegacyStringValue("Cafe\u0301"),
			),
			expectedValue: interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct),
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("Caf\u00E9"),
			),
		},
		"dictionary_with_un-normalized_string": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt8,
					interpreter.PrimitiveStaticTypeString,
				),
				interpreter.NewUnmeteredInt8Value(4),
				newLegacyStringValue("Cafe\u0301"),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt8,
					interpreter.PrimitiveStaticTypeString,
				),
				interpreter.NewUnmeteredInt8Value(4),
				interpreter.NewUnmeteredStringValue("Caf\u00E9"),
			),
		},
		"dictionary_with_un-normalized_string_key": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt8,
				),
				newLegacyStringValue("Cafe\u0301"),
				interpreter.NewUnmeteredInt8Value(4),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt8,
				),
				interpreter.NewUnmeteredStringValue("Caf\u00E9"),
				interpreter.NewUnmeteredInt8Value(4),
			),
		},
		"dictionary_with_un-normalized_string_key_and_value": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeString,
				),
				newLegacyStringValue("Cafe\u0301"),
				newLegacyStringValue("Cafe\u0301"),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeString,
				),
				interpreter.NewUnmeteredStringValue("Caf\u00E9"),
				interpreter.NewUnmeteredStringValue("Caf\u00E9"),
			),
		},
		"composite_with_un-normalized_string": {
			storedValue: interpreter.NewCompositeValue(
				inter,
				interpreter.EmptyLocationRange,
				common.NewAddressLocation(nil, common.Address{0x42}, "Foo"),
				"Bar",
				common.CompositeKindResource,
				[]interpreter.CompositeField{
					interpreter.NewUnmeteredCompositeField(
						"field",
						newLegacyStringValue("Cafe\u0301"),
					),
				},
				common.Address{},
			),
			expectedValue: interpreter.NewCompositeValue(
				inter,
				interpreter.EmptyLocationRange,
				common.NewAddressLocation(nil, common.Address{0x42}, "Foo"),
				"Bar",
				common.CompositeKindResource,
				[]interpreter.CompositeField{
					interpreter.NewUnmeteredCompositeField(
						"field",
						interpreter.NewUnmeteredStringValue("Caf\u00E9"),
					),
				},
				common.Address{},
			),
		},
		"dictionary_with_un-normalized_character_key": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeCharacter,
					interpreter.PrimitiveStaticTypeInt8,
				),
				newLegacyCharacterValue("\u2126"),
				interpreter.NewUnmeteredInt8Value(4),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeCharacter,
					interpreter.PrimitiveStaticTypeInt8,
				),
				interpreter.NewUnmeteredCharacterValue("\u03A9"),
				interpreter.NewUnmeteredInt8Value(4),
			),
		},
	}

	// Store values

	for name, testCase := range testCases {
		transferredValue := testCase.storedValue.Transfer(
			inter,
			locationRange,
			atree.Address(account),
			false,
			nil,
			nil,
		)

		inter.WriteStored(
			account,
			pathDomain.Identifier(),
			interpreter.StringStorageMapKey(name),
			transferredValue,
		)
	}

	err = storage.Commit(inter, true)
	require.NoError(t, err)

	// Migrate

	migration := migrations.NewStorageMigration(inter, storage)

	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			nil,
			NewStringNormalizingMigration(),
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert: Traverse through the storage and see if the values are updated now.

	storageMap := storage.GetStorageMap(account, pathDomain.Identifier(), false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	iterator := storageMap.Iterator(inter)

	for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {
		identifier := string(key.(interpreter.StringAtreeValue))

		t.Run(identifier, func(t *testing.T) {
			testCase, ok := testCases[identifier]
			require.True(t, ok)

			expectedStoredValue := testCase.expectedValue
			if expectedStoredValue == nil {
				expectedStoredValue = testCase.storedValue
			}

			utils.AssertValuesEqual(t, inter, expectedStoredValue, value)
		})
	}
}

// TestStringValueRehash stores a dictionary in storage,
// which has a key that is a string value with a non-normalized representation,
// runs the migration, and ensures the dictionary is still usable
func TestStringValueRehash(t *testing.T) {

	t.Parallel()

	var testAddress = common.MustBytesToAddress([]byte{0x1})

	locationRange := interpreter.EmptyLocationRange

	ledger := NewTestLedger(nil, nil)

	storageMapKey := interpreter.StringStorageMapKey("dict")
	newTestValue := func() interpreter.Value {
		return interpreter.NewUnmeteredIntValueFromInt64(42)
	}

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
		storage := runtime.NewStorage(ledger, nil)
		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage:                       storage,
				AtreeValueValidationEnabled:   false,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		return storage, inter
	}

	t.Run("prepare", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		dictionaryStaticType := interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeString,
			interpreter.PrimitiveStaticTypeInt,
		)
		dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

		// NOTE: un-normalized
		unnormalizedString := "Cafe\u0301"
		stringValue := &interpreter.StringValue{
			Str:             unnormalizedString,
			UnnormalizedStr: unnormalizedString,
		}

		dictValue.Insert(
			inter,
			locationRange,
			stringValue,
			newTestValue(),
		)

		assert.Equal(t,
			[]byte("\x01Cafe\xCC\x81"),
			stringValue.HashInput(inter, locationRange, nil),
		)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			true,
		)

		storageMap.SetValue(
			inter,
			storageMapKey,
			dictValue.Transfer(
				inter,
				locationRange,
				atree.Address(testAddress),
				false,
				nil,
				nil,
			),
		)

		err := storage.Commit(inter, false)
		require.NoError(t, err)
	})

	t.Run("migrate", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		migration := migrations.NewStorageMigration(inter, storage)

		migration.Migrate(
			&migrations.AddressSliceIterator{
				Addresses: []common.Address{
					testAddress,
				},
			},
			migration.NewValueMigrationsPathMigrator(
				nil,
				NewStringNormalizingMigration(),
			),
		)

		err := migration.Commit()
		require.NoError(t, err)
	})

	t.Run("load", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		storageMap := storage.GetStorageMap(testAddress, common.PathDomainStorage.Identifier(), false)
		storedValue := storageMap.ReadValue(inter, storageMapKey)

		require.IsType(t, &interpreter.DictionaryValue{}, storedValue)

		dictValue := storedValue.(*interpreter.DictionaryValue)

		stringValue := interpreter.NewUnmeteredStringValue("Caf\u00E9")

		assert.Equal(t,
			[]byte("\x01Caf\xC3\xA9"),
			stringValue.HashInput(inter, locationRange, nil),
		)

		value, ok := dictValue.Get(inter, locationRange, stringValue)
		require.True(t, ok)

		require.IsType(t, interpreter.IntValue{}, value)
		require.Equal(t,
			newTestValue(),
			value.(interpreter.IntValue),
		)
	})
}

// TestCharacterValueRehash stores a dictionary in storage,
// which has a key that is a character value with a non-normalized representation,
// runs the migration, and ensures the dictionary is still usable
func TestCharacterValueRehash(t *testing.T) {

	t.Parallel()

	var testAddress = common.MustBytesToAddress([]byte{0x1})

	locationRange := interpreter.EmptyLocationRange

	ledger := NewTestLedger(nil, nil)

	storageMapKey := interpreter.StringStorageMapKey("dict")
	newTestValue := func() interpreter.Value {
		return interpreter.NewUnmeteredIntValueFromInt64(42)
	}

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
		storage := runtime.NewStorage(ledger, nil)
		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage:                       storage,
				AtreeValueValidationEnabled:   false,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		return storage, inter
	}

	t.Run("prepare", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		dictionaryStaticType := interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeCharacter,
			interpreter.PrimitiveStaticTypeInt,
		)
		dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

		// NOTE: un-normalized 'Ω'.
		unnormalizedString := "\u2126"

		characterValue := &interpreter.CharacterValue{
			Str:             unnormalizedString,
			UnnormalizedStr: unnormalizedString,
		}

		dictValue.Insert(
			inter,
			locationRange,
			characterValue,
			newTestValue(),
		)

		assert.Equal(t,
			[]byte("\x06\xe2\x84\xa6"),
			characterValue.HashInput(inter, locationRange, nil),
		)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			true,
		)

		storageMap.SetValue(
			inter,
			storageMapKey,
			dictValue.Transfer(
				inter,
				locationRange,
				atree.Address(testAddress),
				false,
				nil,
				nil,
			),
		)

		err := storage.Commit(inter, false)
		require.NoError(t, err)
	})

	t.Run("migrate", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		migration := migrations.NewStorageMigration(inter, storage)

		migration.Migrate(
			&migrations.AddressSliceIterator{
				Addresses: []common.Address{
					testAddress,
				},
			},
			migration.NewValueMigrationsPathMigrator(
				nil,
				NewStringNormalizingMigration(),
			),
		)

		err := migration.Commit()
		require.NoError(t, err)
	})

	t.Run("load", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		storageMap := storage.GetStorageMap(testAddress, common.PathDomainStorage.Identifier(), false)
		storedValue := storageMap.ReadValue(inter, storageMapKey)

		require.IsType(t, &interpreter.DictionaryValue{}, storedValue)

		dictValue := storedValue.(*interpreter.DictionaryValue)

		characterValue := interpreter.NewUnmeteredCharacterValue("\u03A9")

		assert.Equal(t,
			[]byte("\x06\xCe\xA9"),
			characterValue.HashInput(inter, locationRange, nil),
		)

		value, ok := dictValue.Get(inter, locationRange, characterValue)
		require.True(t, ok)

		require.IsType(t, interpreter.IntValue{}, value)
		require.Equal(t,
			newTestValue(),
			value.(interpreter.IntValue),
		)
	})
}
