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

	newCompositeType := func() interpreter.StaticType {
		return interpreter.NewCompositeStaticType(
			nil,
			nil,
			"Bar",
			common.NewTypeIDFromQualifiedName(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
			),
		)
	}

	newInterfaceType := func() interpreter.StaticType {
		return interpreter.NewInterfaceStaticType(
			nil,
			nil,
			"Baz",
			common.NewTypeIDFromQualifiedName(
				nil,
				fooAddressLocation,
				fooBazQualifiedIdentifier,
			),
		)
	}

	testCases := map[string]testCase{
		// base cases
		"compositeToInterface": {
			storedType:   newCompositeType(),
			expectedType: newInterfaceType(),
		},
		"interfaceToComposite": {
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
	}

	// Store values

	ledger := NewTestLedger(nil, nil)
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

	for name, testCase := range testCases {
		storeTypeValue(
			inter,
			testAddress,
			pathDomain,
			name,
			testCase.storedType,
		)
	}

	err = storage.Commit(inter, true)
	require.NoError(t, err)

	// Migrate

	migration := migrations.NewStorageMigration(inter, storage)

	reporter := newTestReporter()

	barStaticType := newCompositeType()
	bazStaticType := newInterfaceType()

	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				testAddress,
			},
		},
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

	// Check reported migrated paths
	for identifier, test := range testCases {
		addressPath := interpreter.AddressPath{
			Address: testAddress,
			Path: interpreter.PathValue{
				Domain:     pathDomain,
				Identifier: identifier,
			},
		}

		if test.expectedType == nil {
			assert.NotContains(t, reporter.migratedPaths, addressPath)
		} else {
			assert.Contains(t, reporter.migratedPaths, addressPath)
		}
	}

	// Assert the migrated values.
	// Traverse through the storage and see if the values are updated now.

	storageMap := storage.GetStorageMap(testAddress, pathDomain.Identifier(), false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	iterator := storageMap.Iterator(inter)

	for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {
		identifier := string(key.(interpreter.StringAtreeValue))

		t.Run(identifier, func(t *testing.T) {
			testCase, ok := testCases[identifier]
			require.True(t, ok)

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
}
