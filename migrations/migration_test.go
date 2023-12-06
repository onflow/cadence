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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testReporter struct {
	migratedPaths map[interpreter.AddressPath][]string
}

func newTestReporter() *testReporter {
	return &testReporter{
		migratedPaths: map[interpreter.AddressPath][]string{},
	}
}

func (t *testReporter) Report(
	addressPath interpreter.AddressPath,
	migration string,
) {
	t.migratedPaths[addressPath] = append(
		t.migratedPaths[addressPath],
		migration,
	)
}

// testStringMigration

type testStringMigration struct{}

func (testStringMigration) Name() string {
	return "testStringMigration"
}

func (testStringMigration) Migrate(
	_ interpreter.AddressPath,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) interpreter.Value {
	if value, ok := value.(*interpreter.StringValue); ok {
		return interpreter.NewUnmeteredStringValue(fmt.Sprintf("updated_%s", value.Str))
	}

	return nil
}

// testInt8Migration

type testInt8Migration struct{}

func (testInt8Migration) Name() string {
	return "testInt8Migration"
}

func (testInt8Migration) Migrate(
	_ interpreter.AddressPath,
	value interpreter.Value,
	_ *interpreter.Interpreter,
) interpreter.Value {
	if value, ok := value.(interpreter.Int8Value); ok {
		return interpreter.NewUnmeteredInt8Value(int8(value) + 10)
	}

	return nil
}

var _ Migration = testStringMigration{}

func TestMultipleMigrations(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}
	pathDomain := common.PathDomainPublic

	type testCase struct {
		storedValue   interpreter.Value
		expectedValue interpreter.Value
	}

	ledger := runtime_utils.NewTestLedger(nil, nil)
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
		"int8_value": {
			storedValue:   interpreter.NewUnmeteredInt8Value(5),
			expectedValue: interpreter.NewUnmeteredInt8Value(15),
		},
		"int16_value": {
			storedValue: interpreter.NewUnmeteredInt16Value(5),
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
		reporter,
		testStringMigration{},
		testInt8Migration{},
	)

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

	// Check the reporter
	require.Equal(t,
		map[interpreter.AddressPath][]string{
			{
				Address: account,
				Path: interpreter.PathValue{
					Domain:     pathDomain,
					Identifier: "int8_value",
				},
			}: {
				"testInt8Migration",
			},
			{
				Address: account,
				Path: interpreter.PathValue{
					Domain:     pathDomain,
					Identifier: "string_value",
				},
			}: {
				"testStringMigration",
			},
		},
		reporter.migratedPaths,
	)
}
