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

package statictypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCompositeAndInterfaceTypeMigration(t *testing.T) {
	t.Parallel()

	pathDomain := common.PathDomainPublic

	type testCase struct {
		storedType   interpreter.StaticType
		expectedType interpreter.StaticType
	}

	newCompositeType := func() *interpreter.CompositeStaticType {
		return interpreter.NewCompositeStaticType(
			nil,
			fooAddressLocation,
			fooBarQualifiedIdentifier,
			common.NewTypeIDFromQualifiedName(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
			),
		)
	}

	newInterfaceType := func() *interpreter.InterfaceStaticType {
		return interpreter.NewInterfaceStaticType(
			nil,
			fooAddressLocation,
			fooBazQualifiedIdentifier,
			common.NewTypeIDFromQualifiedName(
				nil,
				fooAddressLocation,
				fooBazQualifiedIdentifier,
			),
		)
	}

	testCases := map[string]testCase{
		// base cases
		"composite_to_interface": {
			storedType: newCompositeType(),
			expectedType: interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{
					newInterfaceType(),
				},
			),
		},
		"interface_to_composite": {
			storedType:   newInterfaceType(),
			expectedType: newCompositeType(),
		},
		// optional
		"optional": {
			storedType:   interpreter.NewOptionalStaticType(nil, newInterfaceType()),
			expectedType: interpreter.NewOptionalStaticType(nil, newCompositeType()),
		},
		// array
		"array": {
			storedType:   interpreter.NewConstantSizedStaticType(nil, newInterfaceType(), 3),
			expectedType: interpreter.NewConstantSizedStaticType(nil, newCompositeType(), 3),
		},
		// dictionary
		"dictionary": {
			storedType:   interpreter.NewDictionaryStaticType(nil, newInterfaceType(), newInterfaceType()),
			expectedType: interpreter.NewDictionaryStaticType(nil, newCompositeType(), newCompositeType()),
		},
		// reference to optional
		"reference_to_optional": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				interpreter.NewOptionalStaticType(nil, newInterfaceType()),
			),
			expectedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				interpreter.NewOptionalStaticType(nil, newCompositeType()),
			),
		},
	}

	test := func(name string, testCase testCase) {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Store values

			ledger := NewTestLedger(nil, nil)
			storage := runtime.NewStorage(ledger, nil)

			inter, err := interpreter.NewInterpreter(
				nil,
				utils.TestLocation,
				&interpreter.Config{
					Storage:                       storage,
					AtreeValueValidationEnabled:   true,
					AtreeStorageValidationEnabled: true,
				},
			)
			require.NoError(t, err)

			storeTypeValue(
				inter,
				testAddress,
				pathDomain,
				name,
				testCase.storedType,
			)

			err = storage.Commit(inter, true)
			require.NoError(t, err)

			// Migrate

			migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
			require.NoError(t, err)

			reporter := newTestReporter()

			barStaticType := newCompositeType()
			bazStaticType := newInterfaceType()

			migration.Migrate(
				migration.NewValueMigrationsPathMigrator(
					reporter,
					NewStaticTypeMigration().
						WithCompositeTypeConverter(
							func(staticType *interpreter.CompositeStaticType) interpreter.StaticType {
								if staticType.Equal(barStaticType) {
									return bazStaticType
								} else {
									panic(errors.NewUnreachableError())
								}
							},
						).
						WithInterfaceTypeConverter(
							func(staticType *interpreter.InterfaceStaticType) interpreter.StaticType {
								if staticType.Equal(bazStaticType) {
									return barStaticType
								} else {
									panic(errors.NewUnreachableError())
								}
							},
						),
				),
			)

			err = migration.Commit()
			require.NoError(t, err)

			// Assert

			require.Empty(t, reporter.errors)

			err = storage.CheckHealth()
			require.NoError(t, err)

			storageMapKey := interpreter.StringStorageMapKey(name)

			if testCase.expectedType == nil {
				assert.Empty(t, reporter.migrated)
			} else {
				assert.Equal(t,
					map[struct {
						interpreter.StorageKey
						interpreter.StorageMapKey
					}]struct{}{
						{
							StorageKey: interpreter.StorageKey{
								Address: testAddress,
								Key:     pathDomain.Identifier(),
							},
							StorageMapKey: storageMapKey,
						}: {},
					},
					reporter.migrated,
				)
			}

			// Assert the migrated values.

			storageMap := storage.GetStorageMap(testAddress, pathDomain.Identifier(), false)
			require.NotNil(t, storageMap)
			require.Equal(t, uint64(1), storageMap.Count())

			value := storageMap.ReadValue(nil, storageMapKey)

			var expectedType interpreter.StaticType
			if testCase.expectedType != nil {
				expectedType = testCase.expectedType
			} else {
				expectedType = testCase.storedType
			}

			expectedValue := interpreter.NewTypeValue(nil, expectedType)
			utils.AssertValuesEqual(t, inter, expectedValue, value)
		})
	}

	for name, testCase := range testCases {
		test(name, testCase)
	}
}
