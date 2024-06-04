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

package type_keys

import (
	"fmt"
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

type testReporter struct {
	migrated map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}][]string
	errors []error
}

var _ migrations.Reporter = &testReporter{}

func newTestReporter() *testReporter {
	return &testReporter{
		migrated: map[struct {
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

func (t *testReporter) Error(err error) {
	t.errors = append(t.errors, err)
}

func (t *testReporter) DictionaryKeyConflict(addressPath interpreter.AddressPath) {
	// For testing purposes, record the conflict as an error
	t.errors = append(t.errors, fmt.Errorf("dictionary key conflict: %s", addressPath))
}

func TestTypeKeyMigration(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}
	pathDomain := common.PathDomainPublic
	locationRange := interpreter.EmptyLocationRange

	type testCase struct {
		name          string
		storedValue   func(inter *interpreter.Interpreter) interpreter.Value
		expectedValue func(inter *interpreter.Interpreter) interpreter.Value
	}

	test := func(t *testing.T, testCase testCase) {

		t.Run(testCase.name, func(t *testing.T) {

			t.Parallel()

			ledger := NewTestLedger(nil, nil)

			storageMapKey := interpreter.StringStorageMapKey("test")

			newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
				storage := runtime.NewStorage(ledger, nil)
				inter, err := interpreter.NewInterpreter(
					nil,
					utils.TestLocation,
					&interpreter.Config{
						Storage: storage,
						// NOTE: disabled, because encoded and decoded values are expected to not match
						AtreeValueValidationEnabled:   false,
						AtreeStorageValidationEnabled: true,
					},
				)
				require.NoError(t, err)

				return storage, inter
			}

			// Store value
			(func() {

				storage, inter := newStorageAndInterpreter(t)

				transferredValue := testCase.storedValue(inter).Transfer(
					inter,
					locationRange,
					atree.Address(account),
					false,
					nil,
					nil,
					true, // value is standalone
				)

				inter.WriteStored(
					account,
					pathDomain.Identifier(),
					storageMapKey,
					transferredValue,
				)

				err := storage.Commit(inter, true)
				require.NoError(t, err)
			})()

			// Migrate
			(func() {

				storage, inter := newStorageAndInterpreter(t)

				migration, err := migrations.NewStorageMigration(inter, storage, "test", account)
				require.NoError(t, err)

				reporter := newTestReporter()

				migration.Migrate(
					migration.NewValueMigrationsPathMigrator(
						reporter,
						NewTypeKeyMigration(),
					),
				)

				err = migration.Commit()
				require.NoError(t, err)

				require.Empty(t, reporter.errors)

			})()

			// Load
			(func() {

				storage, inter := newStorageAndInterpreter(t)

				err := storage.CheckHealth()
				require.NoError(t, err)

				storageMap := storage.GetStorageMap(account, pathDomain.Identifier(), false)
				require.NotNil(t, storageMap)
				require.Equal(t, uint64(1), storageMap.Count())

				actualValue := storageMap.ReadValue(nil, storageMapKey)

				expectedValue := testCase.expectedValue(inter)

				utils.AssertValuesEqual(t, inter, expectedValue, actualValue)
			})()
		})
	}

	testCases := []testCase{
		{
			name: "optional reference",
			storedValue: func(inter *interpreter.Interpreter) interpreter.Value {

				dictValue := interpreter.NewDictionaryValue(
					inter,
					locationRange,
					interpreter.NewDictionaryStaticType(
						nil,
						interpreter.PrimitiveStaticTypeMetaType,
						interpreter.PrimitiveStaticTypeInt,
					),
				)

				dictValue.Insert(
					inter,
					locationRange,
					// NOTE: storing with legacy key
					migrations.LegacyKey(
						interpreter.NewTypeValue(
							nil,
							interpreter.NewOptionalStaticType(
								nil,
								interpreter.NewReferenceStaticType(
									nil,
									interpreter.UnauthorizedAccess,
									interpreter.PrimitiveStaticTypeInt,
								),
							),
						),
					),
					interpreter.NewUnmeteredIntValueFromInt64(42),
				)

				return dictValue
			},
			expectedValue: func(inter *interpreter.Interpreter) interpreter.Value {
				dictValue := interpreter.NewDictionaryValue(
					inter,
					locationRange,
					interpreter.NewDictionaryStaticType(
						nil,
						interpreter.PrimitiveStaticTypeMetaType,
						interpreter.PrimitiveStaticTypeInt,
					),
				)

				dictValue.Insert(
					inter,
					locationRange,
					// NOTE: expecting to load with new key
					interpreter.NewTypeValue(
						nil,
						interpreter.NewOptionalStaticType(
							nil,
							interpreter.NewReferenceStaticType(
								nil,
								interpreter.UnauthorizedAccess,
								interpreter.PrimitiveStaticTypeInt,
							),
						),
					),
					interpreter.NewUnmeteredIntValueFromInt64(42),
				)

				return dictValue
			},
		},
	}

	for _, testCase := range testCases {
		test(t, testCase)
	}

}
