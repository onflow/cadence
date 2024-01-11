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
			storedValue:   newLegacyStringValue("Caf\u00E9"),
			expectedValue: interpreter.NewUnmeteredStringValue("Caf\u00E9"),
		},
		"un-normalized_character": {
			storedValue:   newLegacyStringValue("Cafe\u0301"),
			expectedValue: interpreter.NewUnmeteredStringValue("Caf\u00E9"),
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
