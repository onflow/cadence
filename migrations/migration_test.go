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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testReporter struct {
	migrated map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}][]string
	errored map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}][]string
}

var _ Reporter = &testReporter{}

func newTestReporter() *testReporter {
	return &testReporter{
		migrated: map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}][]string{},
		errored: map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}][]string{},
	}
}

func (t *testReporter) Migrated(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	migration string,
) {
	key := struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}{
		StorageKey:    storageKey,
		StorageMapKey: storageMapKey,
	}

	t.migrated[key] = append(
		t.migrated[key],
		migration,
	)
}

func (t *testReporter) Error(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	migration string,
	_ error,
) {
	key := struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}{
		StorageKey:    storageKey,
		StorageMapKey: storageMapKey,
	}

	t.errored[key] = append(
		t.errored[key],
		migration,
	)
}

// testStringMigration

type testStringMigration struct{}

var _ ValueMigration = testStringMigration{}

func (testStringMigration) Name() string {
	return "testStringMigration"
}

func (testStringMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (interpreter.Value, error) {
	if value, ok := value.(*interpreter.StringValue); ok {
		return interpreter.NewUnmeteredStringValue(fmt.Sprintf("updated_%s", value.Str)), nil
	}

	return nil, nil
}

// testInt8Migration

type testInt8Migration struct {
	mustError bool
}

var _ ValueMigration = testInt8Migration{}

func (testInt8Migration) Name() string {
	return "testInt8Migration"
}

func (m testInt8Migration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (interpreter.Value, error) {
	int8Value, ok := value.(interpreter.Int8Value)
	if !ok {
		return nil, nil
	}

	if m.mustError {
		return nil, errors.New("error occurred while migrating int8")
	}

	return interpreter.NewUnmeteredInt8Value(int8(int8Value) + 10), nil
}

// testCapMigration

type testCapMigration struct{}

var _ ValueMigration = testCapMigration{}

func (testCapMigration) Name() string {
	return "testCapMigration"
}

func (testCapMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) (interpreter.Value, error) {
	if value, ok := value.(*interpreter.CapabilityValue); ok {
		return interpreter.NewCapabilityValue(
			nil,
			value.ID+10,
			value.Address,
			value.BorrowType,
		), nil
	}

	return nil, nil
}

func TestMultipleMigrations(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}
	pathDomain := common.PathDomainPublic

	type testCase struct {
		name          string
		migration     string
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

	testCases := []testCase{
		{
			name:          "string_value",
			migration:     "testStringMigration",
			storedValue:   interpreter.NewUnmeteredStringValue("hello"),
			expectedValue: interpreter.NewUnmeteredStringValue("updated_hello"),
		},
		{
			name:          "int8_value",
			migration:     "testInt8Migration",
			storedValue:   interpreter.NewUnmeteredInt8Value(5),
			expectedValue: interpreter.NewUnmeteredInt8Value(15),
		},
		{
			name:          "int16_value",
			migration:     "", // should not be migrated
			storedValue:   interpreter.NewUnmeteredInt16Value(5),
			expectedValue: interpreter.NewUnmeteredInt16Value(5),
		},
		{
			name:      "cap_value",
			migration: "testCapMigration",
			storedValue: interpreter.NewCapabilityValue(
				nil,
				5,
				interpreter.AddressValue(common.Address{0x1}),
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					interpreter.PrimitiveStaticTypeString,
				),
			),
			expectedValue: interpreter.NewCapabilityValue(
				nil,
				15,
				interpreter.AddressValue(common.Address{0x1}),
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					interpreter.PrimitiveStaticTypeString,
				),
			),
		},
	}

	variableSizedAnyStructStaticType :=
		interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct)

	dictionaryAnyStructStaticType :=
		interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeAnyStruct,
			interpreter.PrimitiveStaticTypeAnyStruct,
		)

	for _, test := range testCases {
		testCases = append(testCases, testCase{
			name:      "array_" + test.name,
			migration: test.migration,
			storedValue: interpreter.NewArrayValue(
				inter,
				emptyLocationRange,
				variableSizedAnyStructStaticType,
				common.ZeroAddress,
				test.storedValue,
			),
			expectedValue: interpreter.NewArrayValue(
				inter,
				emptyLocationRange,
				variableSizedAnyStructStaticType,
				common.ZeroAddress,
				test.expectedValue,
			),
		})

		if _, ok := test.storedValue.(interpreter.HashableValue); ok {

			testCases = append(testCases, testCase{
				name:      "dict_key_" + test.name,
				migration: test.migration,
				storedValue: interpreter.NewDictionaryValue(
					inter,
					emptyLocationRange,
					dictionaryAnyStructStaticType,
					test.storedValue,
					interpreter.TrueValue,
				),

				expectedValue: interpreter.NewDictionaryValue(
					inter,
					emptyLocationRange,
					dictionaryAnyStructStaticType,
					test.expectedValue,
					interpreter.TrueValue,
				),
			})
		}

		testCases = append(testCases, testCase{
			name:      "dict_value_" + test.name,
			migration: test.migration,
			storedValue: interpreter.NewDictionaryValue(
				inter,
				emptyLocationRange,
				dictionaryAnyStructStaticType,
				interpreter.TrueValue,
				test.storedValue,
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				emptyLocationRange,
				dictionaryAnyStructStaticType,
				interpreter.TrueValue,
				test.expectedValue,
			),
		})

		testCases = append(testCases, testCase{
			name:          "some_" + test.name,
			migration:     test.migration,
			storedValue:   interpreter.NewSomeValueNonCopying(nil, test.storedValue),
			expectedValue: interpreter.NewSomeValueNonCopying(nil, test.expectedValue),
		})

		if _, ok := test.storedValue.(*interpreter.CapabilityValue); ok {

			testCases = append(testCases, testCase{
				name:      "published_" + test.name,
				migration: test.migration,
				storedValue: interpreter.NewPublishedValue(
					nil,
					interpreter.AddressValue(common.ZeroAddress),
					test.storedValue.(*interpreter.CapabilityValue),
				),
				expectedValue: interpreter.NewPublishedValue(
					nil,
					interpreter.AddressValue(common.ZeroAddress),
					test.expectedValue.(*interpreter.CapabilityValue),
				),
			})
		}

		testCases = append(testCases, testCase{
			name:      "struct_" + test.name,
			migration: test.migration,
			storedValue: interpreter.NewCompositeValue(
				inter,
				emptyLocationRange,
				utils.TestLocation,
				"S",
				common.CompositeKindStructure,
				[]interpreter.CompositeField{
					{
						Name:  "test",
						Value: test.storedValue,
					},
				},
				common.ZeroAddress,
			),
			expectedValue: interpreter.NewCompositeValue(
				inter,
				emptyLocationRange,
				utils.TestLocation,
				"S",
				common.CompositeKindStructure,
				[]interpreter.CompositeField{
					{
						Name:  "test",
						Value: test.expectedValue,
					},
				},
				common.ZeroAddress,
			),
		})
	}

	// Store values

	for _, testCase := range testCases {
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
			interpreter.StringStorageMapKey(testCase.name),
			transferredValue,
		)
	}

	err = storage.Commit(inter, true)
	require.NoError(t, err)

	// Migrate

	migration := NewStorageMigration(inter, storage)

	reporter := newTestReporter()

	migration.Migrate(
		&AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testStringMigration{},
			testInt8Migration{},
			testCapMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert: Traverse through the storage and see if the values are updated now.

	storageMap := storage.GetStorageMap(account, pathDomain.Identifier(), false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	iterator := storageMap.Iterator(inter)

	testCasesByName := map[string]testCase{}

	for _, testCase := range testCases {
		testCasesByName[testCase.name] = testCase
	}

	for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {
		identifier := string(key.(interpreter.StringAtreeValue))

		t.Run(identifier, func(t *testing.T) {
			testCase, ok := testCasesByName[identifier]
			require.True(t, ok)
			utils.AssertValuesEqual(t, inter, testCase.expectedValue, value)
		})
	}

	// Check the reporter
	expectedMigrations := map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}][]string{}

	for _, testCase := range testCases {
		if testCase.migration == "" {
			continue
		}

		key := struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}{
			StorageKey: interpreter.StorageKey{
				Address: account,
				Key:     pathDomain.Identifier(),
			},
			StorageMapKey: interpreter.StringStorageMapKey(testCase.name),
		}

		expectedMigrations[key] = append(
			expectedMigrations[key],
			testCase.migration,
		)
	}

	require.Equal(t,
		expectedMigrations,
		reporter.migrated,
	)
}

func TestMigrationError(t *testing.T) {
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

	testCases := map[string]testCase{
		"string_value": {
			storedValue:   interpreter.NewUnmeteredStringValue("hello"),
			expectedValue: interpreter.NewUnmeteredStringValue("updated_hello"),
		},
		// Since Int8 migration expected to produce error,
		// int8 value should not have been migrated.
		"int8_value": {
			storedValue:   interpreter.NewUnmeteredInt8Value(5),
			expectedValue: interpreter.NewUnmeteredInt8Value(5),
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

	migration := NewStorageMigration(inter, storage)

	reporter := newTestReporter()

	migration.Migrate(
		&AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testStringMigration{},

			// Int8 migration should produce errors
			testInt8Migration{
				mustError: true,
			},
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
			utils.AssertValuesEqual(t, inter, testCase.expectedValue, value)
		})
	}

	// Check the reporter.
	// Since Int8 migration produces an error, only the string value must have been migrated.
	require.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}][]string{
			{
				StorageKey: interpreter.StorageKey{
					Address: account,
					Key:     pathDomain.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("string_value"),
			}: {
				"testStringMigration",
			},
		},
		reporter.migrated,
	)

	require.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}][]string{
			{
				StorageKey: interpreter.StorageKey{
					Address: account,
					Key:     pathDomain.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("int8_value"),
			}: {
				"testInt8Migration",
			},
		},
		reporter.errored,
	)
}
