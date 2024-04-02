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

	"github.com/onflow/atree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

const fooBarQualifiedIdentifier = "Foo.Bar"
const fooBazQualifiedIdentifier = "Foo.Baz"

var fooAddressLocation = common.NewAddressLocation(nil, testAddress, "Foo")

func newIntersectionStaticTypeWithOneInterface() *interpreter.IntersectionStaticType {
	return interpreter.NewIntersectionStaticType(
		nil,
		[]*interpreter.InterfaceStaticType{
			interpreter.NewInterfaceStaticType(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBarQualifiedIdentifier,
				),
			),
		},
	)
}

func newIntersectionStaticTypeWithTwoInterfaces() *interpreter.IntersectionStaticType {
	return interpreter.NewIntersectionStaticType(
		nil,
		[]*interpreter.InterfaceStaticType{
			interpreter.NewInterfaceStaticType(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBarQualifiedIdentifier,
				),
			),
			interpreter.NewInterfaceStaticType(
				nil,
				fooAddressLocation,
				fooBazQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBazQualifiedIdentifier,
				),
			),
		},
	)
}

func newIntersectionStaticTypeWithTwoInterfacesReversed() *interpreter.IntersectionStaticType {
	return interpreter.NewIntersectionStaticType(
		nil,
		[]*interpreter.InterfaceStaticType{
			interpreter.NewInterfaceStaticType(
				nil,
				fooAddressLocation,
				fooBazQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBazQualifiedIdentifier,
				),
			),
			interpreter.NewInterfaceStaticType(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBarQualifiedIdentifier,
				),
			),
		},
	)
}

func TestIntersectionTypeMigration(t *testing.T) {
	t.Parallel()

	pathDomain := common.PathDomainPublic

	const stringType = interpreter.PrimitiveStaticTypeString

	type testCase struct {
		storedType   interpreter.StaticType
		expectedType interpreter.StaticType
	}

	testCases := map[string]testCase{
		// base cases
		"primitive": {
			storedType:   stringType,
			expectedType: nil,
		},
		"intersection_with_one_interface": {
			storedType:   newIntersectionStaticTypeWithOneInterface(),
			expectedType: newIntersectionStaticTypeWithOneInterface(),
		},
		// optional
		"optional_primitive": {
			storedType:   interpreter.NewOptionalStaticType(nil, stringType),
			expectedType: nil,
		},
		"optional_intersection_with_one_interface": {
			storedType: interpreter.NewOptionalStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
			),
			expectedType: interpreter.NewOptionalStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
			),
		},
		"optional_intersection_with_two_interfaces": {
			storedType: interpreter.NewOptionalStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
			expectedType: interpreter.NewOptionalStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
		},
		// constant-sized array
		"constant_sized_array_of_primitive": {
			storedType:   interpreter.NewConstantSizedStaticType(nil, stringType, 3),
			expectedType: nil,
		},
		"constant_sized_array_of_intersection_with_one_interface": {
			storedType: interpreter.NewConstantSizedStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
				3,
			),
			expectedType: interpreter.NewConstantSizedStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
				3,
			),
		},
		"constant_sized_array_of_intersection_with_two_interfaces": {
			storedType: interpreter.NewConstantSizedStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
				3,
			),
			expectedType: interpreter.NewConstantSizedStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
				3,
			),
		},
		// variable-sized array
		"variable_sized_array_of_primitive": {
			storedType:   interpreter.NewVariableSizedStaticType(nil, stringType),
			expectedType: nil,
		},
		"variable_sized_array_of_intersection_with_one_interface": {
			storedType: interpreter.NewVariableSizedStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
			),
			expectedType: interpreter.NewVariableSizedStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
			),
		},
		"variable_sized_array_of_intersection_with_two_interfaces": {
			storedType: interpreter.NewVariableSizedStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
			expectedType: interpreter.NewVariableSizedStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
		},
		// dictionary
		"dictionary_of_primitive_key_and_value": {
			storedType:   interpreter.NewDictionaryStaticType(nil, stringType, stringType),
			expectedType: nil,
		},
		"dictionary_of_intersection_with_one_interface_key": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
				stringType,
			),
			expectedType: interpreter.NewDictionaryStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
				stringType,
			),
		},
		"dictionary_of_intersection_with_one_interface_value": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				newIntersectionStaticTypeWithOneInterface(),
			),
			expectedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				newIntersectionStaticTypeWithOneInterface(),
			),
		},
		"dictionary_of_intersection_with_two_interfaces_key": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
				stringType,
			),
			expectedType: interpreter.NewDictionaryStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
				stringType,
			),
		},
		"dictionary_of_intersection_with_two_interfaces_value": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
			expectedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
		},
		// capability
		"capability_primitive": {
			storedType:   interpreter.NewCapabilityStaticType(nil, stringType),
			expectedType: nil,
		},
		"capability_intersection_with_one_interface": {
			storedType: interpreter.NewCapabilityStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
			),
			expectedType: interpreter.NewCapabilityStaticType(
				nil,
				newIntersectionStaticTypeWithOneInterface(),
			),
		},
		"capability_intersection_with_two_interfaces": {
			storedType: interpreter.NewCapabilityStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
			expectedType: interpreter.NewCapabilityStaticType(
				nil,
				newIntersectionStaticTypeWithTwoInterfaces(),
			),
		},
		// interface
		"non_intersection_interface": {
			storedType: interpreter.NewInterfaceStaticType(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBarQualifiedIdentifier,
				),
			),
			expectedType: interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticType(
						nil,
						fooAddressLocation,
						fooBarQualifiedIdentifier,
						common.NewTypeIDFromQualifiedName(
							nil,
							fooAddressLocation,
							fooBarQualifiedIdentifier,
						),
					),
				},
			),
		},
		"intersection_interface": {
			storedType: interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticType(
						nil,
						fooAddressLocation,
						fooBarQualifiedIdentifier,
						common.NewTypeIDFromQualifiedName(
							nil,
							fooAddressLocation,
							fooBarQualifiedIdentifier,
						),
					),
				},
			),
			expectedType: interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticType(
						nil,
						fooAddressLocation,
						fooBarQualifiedIdentifier,
						common.NewTypeIDFromQualifiedName(
							nil,
							fooAddressLocation,
							fooBarQualifiedIdentifier,
						),
					),
				},
			),
		},
		// composite
		"composite": {
			storedType: interpreter.NewCompositeStaticType(
				nil,
				fooAddressLocation,
				"Foo.Bar",
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBarQualifiedIdentifier,
				),
			),
			expectedType: nil,
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
			AtreeValueValidationEnabled:   true,
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

	migration := migrations.NewStorageMigration(inter, storage, "test")

	reporter := newTestReporter()

	migration.MigrateAccount(
		testAddress,
		migration.NewValueMigrationsPathMigrator(
			reporter,
			NewStaticTypeMigration(),
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

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

			key := struct {
				interpreter.StorageKey
				interpreter.StorageMapKey
			}{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     pathDomain.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey(identifier),
			}

			if testCase.expectedType == nil {
				assert.NotContains(t, reporter.migrated, key)
			} else {
				assert.Contains(t, reporter.migrated, key)
			}

			var expectedValue interpreter.Value
			if testCase.expectedType != nil {
				expectedValue = interpreter.NewTypeValue(nil, testCase.expectedType)

				// `IntersectionType.LegacyType` is not considered in the `IntersectionType.Equal` method.
				// Therefore, check for the legacy type equality manually.
				typeValue := value.(interpreter.TypeValue)
				if actualIntersectionType, ok := typeValue.Type.(*interpreter.IntersectionStaticType); ok {
					expectedIntersectionType := testCase.expectedType.(*interpreter.IntersectionStaticType)

					if actualIntersectionType.LegacyType == nil {
						assert.Nil(t, expectedIntersectionType.LegacyType)
					} else {
						assert.True(t, actualIntersectionType.LegacyType.Equal(expectedIntersectionType.LegacyType))
					}
				}
			} else {
				expectedValue = interpreter.NewTypeValue(nil, testCase.storedType)
			}

			utils.AssertValuesEqual(t, inter, expectedValue, value)
		})
	}
}

// TestIntersectionTypeRehash stores a dictionary in storage,
// which has a key that is a type value with a restricted type that has two interface types,
// runs the migration, and ensures the dictionary is still usable
func TestIntersectionTypeRehash(t *testing.T) {

	t.Parallel()

	locationRange := interpreter.EmptyLocationRange

	ledger := NewTestLedger(nil, nil)

	storageMapKey := interpreter.StringStorageMapKey("dict")
	newTestValue := func() interpreter.Value {
		return interpreter.NewUnmeteredStringValue("test")
	}

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
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

		return storage, inter
	}

	t.Run("prepare", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		dictionaryStaticType := interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeMetaType,
			interpreter.PrimitiveStaticTypeString,
		)
		dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

		intersectionType := &migrations.LegacyIntersectionType{
			IntersectionStaticType: newIntersectionStaticTypeWithTwoInterfacesReversed(),
		}

		typeValue := interpreter.NewUnmeteredTypeValue(intersectionType)

		dictValue.Insert(
			inter,
			locationRange,
			typeValue,
			newTestValue(),
		)

		// NOTE: intentionally in reverse order
		assert.Equal(t,
			common.TypeID("{A.4200000000000000.Foo.Baz,A.4200000000000000.Foo.Bar}"),
			intersectionType.ID(),
		)

		storageMap := storage.GetStorageMap(
			testAddress,
			common.PathDomainStorage.Identifier(),
			true,
		)

		storageMap.SetValue(inter,
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

		migration := migrations.NewStorageMigration(inter, storage, "test")

		reporter := newTestReporter()

		migration.MigrateAccount(
			testAddress,
			migration.NewValueMigrationsPathMigrator(
				reporter,
				NewStaticTypeMigration(),
			),
		)

		err := migration.Commit()
		require.NoError(t, err)

		// Assert

		require.Empty(t, reporter.errors)

		err = storage.CheckHealth()
		require.NoError(t, err)

		require.Equal(t,
			map[struct {
				interpreter.StorageKey
				interpreter.StorageMapKey
			}]struct{}{
				{
					StorageKey: interpreter.StorageKey{
						Address: testAddress,
						Key:     common.PathDomainStorage.Identifier(),
					},
					StorageMapKey: storageMapKey,
				}: {},
			},
			reporter.migrated,
		)
	})

	t.Run("load", func(t *testing.T) {

		storage, inter := newStorageAndInterpreter(t)

		storageMap := storage.GetStorageMap(testAddress, common.PathDomainStorage.Identifier(), false)
		storedValue := storageMap.ReadValue(inter, storageMapKey)

		require.IsType(t, &interpreter.DictionaryValue{}, storedValue)

		dictValue := storedValue.(*interpreter.DictionaryValue)

		intersectionType := newIntersectionStaticTypeWithTwoInterfaces()
		typeValue := interpreter.NewUnmeteredTypeValue(intersectionType)

		// NOTE: in *sorted* order
		assert.Equal(t,
			common.TypeID("{A.4200000000000000.Foo.Bar,A.4200000000000000.Foo.Baz}"),
			intersectionType.ID(),
		)

		value, ok := dictValue.Get(inter, locationRange, typeValue)
		require.True(t, ok)

		require.IsType(t, &interpreter.StringValue{}, value)
		require.Equal(t,
			newTestValue(),
			value.(*interpreter.StringValue),
		)
	})
}

// TestRehashNestedIntersectionType stores a dictionary in storage,
// which has a key that is a type value with a restricted type that has two interface types,
// runs the migration, and ensures the dictionary is still usable
func TestRehashNestedIntersectionType(t *testing.T) {

	locationRange := interpreter.EmptyLocationRange

	storageMapKey := interpreter.StringStorageMapKey("dict")
	newTestValue := func() interpreter.Value {
		return interpreter.NewUnmeteredStringValue("test")
	}

	newStorageAndInterpreter := func(t *testing.T, ledger atree.Ledger) (*runtime.Storage, *interpreter.Interpreter) {
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

		return storage, inter
	}

	t.Run("array type", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)

		t.Run("prepare", func(t *testing.T) {

			storage, inter := newStorageAndInterpreter(t, ledger)

			dictionaryStaticType := interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeMetaType,
				interpreter.PrimitiveStaticTypeString,
			)
			dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

			intersectionStaticType := newIntersectionStaticTypeWithTwoInterfacesReversed()
			intersectionStaticType.LegacyType = interpreter.PrimitiveStaticTypeAnyStruct

			intersectionType := &migrations.LegacyIntersectionType{
				IntersectionStaticType: intersectionStaticType,
			}

			typeValue := interpreter.NewUnmeteredTypeValue(
				interpreter.NewVariableSizedStaticType(
					nil,
					intersectionType,
				),
			)

			dictValue.Insert(
				inter,
				locationRange,
				typeValue,
				newTestValue(),
			)

			// NOTE: intentionally in reverse order
			assert.Equal(t,
				common.TypeID("AnyStruct{A.4200000000000000.Foo.Baz,A.4200000000000000.Foo.Bar}"),
				intersectionType.ID(),
			)

			storageMap := storage.GetStorageMap(
				testAddress,
				common.PathDomainStorage.Identifier(),
				true,
			)

			storageMap.SetValue(inter,
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

			storage, inter := newStorageAndInterpreter(t, ledger)

			migration := migrations.NewStorageMigration(inter, storage, "test")

			reporter := newTestReporter()

			migration.MigrateAccount(
				testAddress,
				migration.NewValueMigrationsPathMigrator(
					reporter,
					NewStaticTypeMigration(),
				),
			)

			err := migration.Commit()
			require.NoError(t, err)

			// Assert

			require.Empty(t, reporter.errors)

			err = storage.CheckHealth()
			require.NoError(t, err)

			require.Equal(t,
				map[struct {
					interpreter.StorageKey
					interpreter.StorageMapKey
				}]struct{}{
					{
						StorageKey: interpreter.StorageKey{
							Address: testAddress,
							Key:     common.PathDomainStorage.Identifier(),
						},
						StorageMapKey: storageMapKey,
					}: {},
				},
				reporter.migrated,
			)
		})

		t.Run("load", func(t *testing.T) {

			storage, inter := newStorageAndInterpreter(t, ledger)

			storageMap := storage.GetStorageMap(testAddress, common.PathDomainStorage.Identifier(), false)
			storedValue := storageMap.ReadValue(inter, storageMapKey)

			require.IsType(t, &interpreter.DictionaryValue{}, storedValue)

			dictValue := storedValue.(*interpreter.DictionaryValue)

			intersectionType := newIntersectionStaticTypeWithTwoInterfaces()
			typeValue := interpreter.NewUnmeteredTypeValue(
				interpreter.NewVariableSizedStaticType(nil, intersectionType),
			)

			// NOTE: in *sorted* order
			assert.Equal(t,
				common.TypeID("{A.4200000000000000.Foo.Bar,A.4200000000000000.Foo.Baz}"),
				intersectionType.ID(),
			)

			value, ok := dictValue.Get(inter, locationRange, typeValue)
			require.True(t, ok)

			require.IsType(t, &interpreter.StringValue{}, value)
			require.Equal(t,
				newTestValue(),
				value.(*interpreter.StringValue),
			)
		})
	})

	t.Run("dictionary type", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)

		t.Run("prepare", func(t *testing.T) {

			storage, inter := newStorageAndInterpreter(t, ledger)

			dictionaryStaticType := interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeMetaType,
				interpreter.PrimitiveStaticTypeString,
			)
			dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

			intersectionStaticType := newIntersectionStaticTypeWithTwoInterfacesReversed()
			intersectionStaticType.LegacyType = interpreter.PrimitiveStaticTypeAnyStruct

			intersectionType := &migrations.LegacyIntersectionType{
				IntersectionStaticType: intersectionStaticType,
			}

			typeValue := interpreter.NewUnmeteredTypeValue(
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					intersectionType,
				),
			)

			dictValue.Insert(
				inter,
				locationRange,
				typeValue,
				newTestValue(),
			)

			// NOTE: intentionally in reverse order
			assert.Equal(t,
				common.TypeID("AnyStruct{A.4200000000000000.Foo.Baz,A.4200000000000000.Foo.Bar}"),
				intersectionType.ID(),
			)

			storageMap := storage.GetStorageMap(
				testAddress,
				common.PathDomainStorage.Identifier(),
				true,
			)

			storageMap.SetValue(inter,
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

			storage, inter := newStorageAndInterpreter(t, ledger)

			migration := migrations.NewStorageMigration(inter, storage, "test")

			reporter := newTestReporter()

			migration.MigrateAccount(
				testAddress,
				migration.NewValueMigrationsPathMigrator(
					reporter,
					NewStaticTypeMigration(),
				),
			)

			err := migration.Commit()
			require.NoError(t, err)

			// Assert

			require.Empty(t, reporter.errors)

			err = storage.CheckHealth()
			require.NoError(t, err)

			require.Equal(t,
				map[struct {
					interpreter.StorageKey
					interpreter.StorageMapKey
				}]struct{}{
					{
						StorageKey: interpreter.StorageKey{
							Address: testAddress,
							Key:     common.PathDomainStorage.Identifier(),
						},
						StorageMapKey: storageMapKey,
					}: {},
				},
				reporter.migrated,
			)
		})

		t.Run("load", func(t *testing.T) {

			storage, inter := newStorageAndInterpreter(t, ledger)

			storageMap := storage.GetStorageMap(testAddress, common.PathDomainStorage.Identifier(), false)
			storedValue := storageMap.ReadValue(inter, storageMapKey)

			require.IsType(t, &interpreter.DictionaryValue{}, storedValue)

			dictValue := storedValue.(*interpreter.DictionaryValue)

			intersectionType := newIntersectionStaticTypeWithTwoInterfaces()
			typeValue := interpreter.NewUnmeteredTypeValue(
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					intersectionType,
				),
			)

			// NOTE: in *sorted* order
			assert.Equal(t,
				common.TypeID("{A.4200000000000000.Foo.Bar,A.4200000000000000.Foo.Baz}"),
				intersectionType.ID(),
			)

			value, ok := dictValue.Get(inter, locationRange, typeValue)
			require.True(t, ok)

			require.IsType(t, &interpreter.StringValue{}, value)
			require.Equal(t,
				newTestValue(),
				value.(*interpreter.StringValue),
			)
		})
	})
}

func TestIntersectionTypeMigrationWithInterfaceTypeConverter(t *testing.T) {
	t.Parallel()

	const fooCompositeQualifiedIdentifierA = "Foo.A"
	const fooCompositeQualifiedIdentifierB = "Foo.B"

	fooACompositeType := interpreter.NewCompositeStaticType(
		nil,
		fooAddressLocation,
		fooCompositeQualifiedIdentifierA,
		fooAddressLocation.TypeID(nil, fooCompositeQualifiedIdentifierA),
	)

	fooBCompositeType := interpreter.NewCompositeStaticType(
		nil,
		fooAddressLocation,
		fooCompositeQualifiedIdentifierB,
		fooAddressLocation.TypeID(nil, fooCompositeQualifiedIdentifierB),
	)

	const fooBarQualifiedIdentifier = "Foo.Bar"
	const fooBazQualifiedIdentifier = "Foo.Baz"

	fooBarInterfaceType := interpreter.NewInterfaceStaticType(
		nil,
		fooAddressLocation,
		fooBarQualifiedIdentifier,
		fooAddressLocation.TypeID(nil, fooBarQualifiedIdentifier),
	)
	fooBazInterfaceType := interpreter.NewInterfaceStaticType(
		nil,
		fooAddressLocation,
		fooBazQualifiedIdentifier,
		fooAddressLocation.TypeID(nil, fooBazQualifiedIdentifier),
	)

	test := func(
		t *testing.T,
		inputType interpreter.StaticType,
		convertCompositeType bool,
		convertInterfaceType bool,
		expectedType interpreter.StaticType,
	) {

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

		const testPathDomain = common.PathDomainStorage
		const testPathIdentifier = "test_type_value"

		storeTypeValue(
			inter,
			testAddress,
			testPathDomain,
			testPathIdentifier,
			inputType,
		)

		err = storage.Commit(inter, true)
		require.NoError(t, err)

		// Migrate

		migration := migrations.NewStorageMigration(inter, storage, "test")

		reporter := newTestReporter()

		staticTypeMigration := NewStaticTypeMigration()
		if convertCompositeType {
			staticTypeMigration.WithCompositeTypeConverter(
				func(staticType *interpreter.CompositeStaticType) interpreter.StaticType {
					if staticType == fooACompositeType {
						return fooBCompositeType
					}
					return nil
				},
			)
		}
		if convertInterfaceType {
			staticTypeMigration.WithInterfaceTypeConverter(
				func(staticType *interpreter.InterfaceStaticType) interpreter.StaticType {
					if staticType.ID() == fooBarInterfaceType.ID() {
						return fooBazInterfaceType
					}
					return nil
				},
			)
		}

		migration.MigrateAccount(
			testAddress,
			migration.NewValueMigrationsPathMigrator(
				reporter,
				staticTypeMigration,
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		require.Empty(t, reporter.errors)

		err = storage.CheckHealth()
		require.NoError(t, err)

		// Assert the migrated value.

		storageMap := storage.GetStorageMap(testAddress, testPathDomain.Identifier(), false)
		require.NotNil(t, storageMap)
		require.Equal(t, uint64(1), storageMap.Count())

		value := storageMap.ReadValue(nil, interpreter.StringStorageMapKey(testPathIdentifier))
		assert.NotNil(t, value)

		expectedValue := interpreter.NewTypeValue(nil, expectedType)

		assert.Equal(t, value, expectedValue)
	}

	t.Run("A{}", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.IntersectionStaticType{
				LegacyType: fooACompositeType,
			},
			false,
			false,
			fooACompositeType,
		)
	})

	t.Run("A{}, convert composite", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.IntersectionStaticType{
				LegacyType: fooACompositeType,
			},
			true,
			false,
			fooBCompositeType,
		)
	})

	t.Run("A{Bar}", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.IntersectionStaticType{
				LegacyType: fooACompositeType,
				Types: []*interpreter.InterfaceStaticType{
					fooBarInterfaceType,
				},
			},
			false,
			false,
			fooACompositeType,
		)
	})

	t.Run("A{Bar}, convert interface", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.IntersectionStaticType{
				LegacyType: fooACompositeType,
				Types: []*interpreter.InterfaceStaticType{
					fooBarInterfaceType,
				},
			},
			false,
			true,
			fooACompositeType,
		)
	})

	t.Run("A{Bar}, convert composite, convert interface", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.IntersectionStaticType{
				LegacyType: fooACompositeType,
				Types: []*interpreter.InterfaceStaticType{
					fooBarInterfaceType,
				},
			},
			true,
			true,
			fooBCompositeType,
		)

	})

	t.Run("{Bar}", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					fooBarInterfaceType,
				},
			},
			false,
			false,
			&interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					fooBarInterfaceType,
				},
			},
		)

	})

	t.Run("{Bar}, convert interface", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					fooBarInterfaceType,
				},
			},
			false,
			true,
			&interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					fooBazInterfaceType,
				},
			},
		)
	})

	t.Run("&A{}", func(t *testing.T) {
		t.Parallel()

		test(t,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
				},
			},
			false,
			false,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
				},
			},
		)
	})

	t.Run("&A{}, convert composite", func(t *testing.T) {
		t.Parallel()

		test(
			t,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
				},
			},
			true,
			false,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooBCompositeType,
				},
			},
		)
	})

	t.Run("&A{Bar}", func(t *testing.T) {
		t.Parallel()

		test(
			t,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
					Types: []*interpreter.InterfaceStaticType{
						fooBarInterfaceType,
					},
				},
			},
			false,
			false,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
					Types: []*interpreter.InterfaceStaticType{
						fooBarInterfaceType,
					},
				},
			},
		)
	})

	t.Run("&A{Bar}, convert interface", func(t *testing.T) {
		t.Parallel()

		test(
			t,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
					Types: []*interpreter.InterfaceStaticType{
						fooBarInterfaceType,
					},
				},
			},
			false,
			true,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
					Types: []*interpreter.InterfaceStaticType{
						fooBazInterfaceType,
					},
				},
			},
		)
	})

	t.Run("&A{Bar}, convert composite, convert interface", func(t *testing.T) {
		t.Parallel()

		test(
			t,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooACompositeType,
					Types: []*interpreter.InterfaceStaticType{
						fooBarInterfaceType,
					},
				},
			},
			true,
			true,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					LegacyType: fooBCompositeType,
					Types: []*interpreter.InterfaceStaticType{
						fooBazInterfaceType,
					},
				},
			},
		)

	})

	t.Run("&{Bar}", func(t *testing.T) {
		t.Parallel()

		test(
			t,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					Types: []*interpreter.InterfaceStaticType{
						fooBarInterfaceType,
					},
				},
			},
			false,
			false,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					Types: []*interpreter.InterfaceStaticType{
						fooBarInterfaceType,
					},
				},
			},
		)

	})

	t.Run("&{Bar}, convert interface", func(t *testing.T) {
		t.Parallel()

		test(
			t,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					Types: []*interpreter.InterfaceStaticType{
						fooBarInterfaceType,
					},
				},
			},
			false,
			true,
			&interpreter.ReferenceStaticType{
				Authorization: interpreter.Unauthorized{},
				ReferencedType: &interpreter.IntersectionStaticType{
					Types: []*interpreter.InterfaceStaticType{
						fooBazInterfaceType,
					},
				},
			},
		)
	})
}

func TestIntersectionTypeMigrationWithTypeConverters(t *testing.T) {
	t.Parallel()

	migrate := func(
		t *testing.T,
		staticTypeMigration *StaticTypeMigration,
		input interpreter.StaticType,
	) interpreter.StaticType {

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

		const testPathDomain = common.PathDomainStorage
		const testPathIdentifier = "test_type_value"

		storeTypeValue(
			inter,
			testAddress,
			testPathDomain,
			testPathIdentifier,
			input,
		)

		err = storage.Commit(inter, true)
		require.NoError(t, err)

		// Migrate

		migration := migrations.NewStorageMigration(inter, storage, "test")

		reporter := newTestReporter()

		migration.MigrateAccount(
			testAddress,
			migration.NewValueMigrationsPathMigrator(
				reporter,
				staticTypeMigration,
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		require.Empty(t, reporter.errors)

		err = storage.CheckHealth()
		require.NoError(t, err)

		key := struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}{
			StorageKey: interpreter.StorageKey{
				Address: testAddress,
				Key:     testPathDomain.Identifier(),
			},
			StorageMapKey: interpreter.StringStorageMapKey(testPathIdentifier),
		}

		assert.Contains(t, reporter.migrated, key)

		storageMap := storage.GetStorageMap(testAddress, testPathDomain.Identifier(), false)
		require.NotNil(t, storageMap)
		require.Equal(t, uint64(1), storageMap.Count())

		value := storageMap.ReadValue(nil, interpreter.StringStorageMapKey(testPathIdentifier))
		require.NotNil(t, value)

		require.IsType(t, interpreter.TypeValue{}, value)

		return value.(interpreter.TypeValue).Type
	}

	const fooCompositeQualifiedIdentifierA = "Foo.A"
	const fooCompositeQualifiedIdentifierB = "Foo.B"

	fooACompositeType := interpreter.NewCompositeStaticType(
		nil,
		fooAddressLocation,
		fooCompositeQualifiedIdentifierA,
		fooAddressLocation.TypeID(nil, fooCompositeQualifiedIdentifierA),
	)

	fooBCompositeType := interpreter.NewCompositeStaticType(
		nil,
		fooAddressLocation,
		fooCompositeQualifiedIdentifierB,
		fooAddressLocation.TypeID(nil, fooCompositeQualifiedIdentifierB),
	)

	const fooInterfaceQualifiedIdentifierC = "Foo.C"
	const fooInterfaceQualifiedIdentifierD = "Foo.D"

	fooCInterfaceType := interpreter.NewInterfaceStaticType(
		nil,
		fooAddressLocation,
		fooInterfaceQualifiedIdentifierC,
		fooAddressLocation.TypeID(nil, fooInterfaceQualifiedIdentifierC),
	)
	fooDInterfaceType := interpreter.NewInterfaceStaticType(
		nil,
		fooAddressLocation,
		fooInterfaceQualifiedIdentifierD,
		fooAddressLocation.TypeID(nil, fooInterfaceQualifiedIdentifierD),
	)

	t.Run("composite type converter", func(t *testing.T) {
		t.Parallel()

		t.Run("return non-interface", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration().
				WithCompositeTypeConverter(func(staticType *interpreter.CompositeStaticType) interpreter.StaticType {
					if staticType == fooACompositeType {
						return fooBCompositeType
					}
					return nil
				})

			actual := migrate(t, staticTypeMigration, fooACompositeType)
			assert.Equal(t, fooBCompositeType, actual)
		})

		t.Run("return interface", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration().
				WithCompositeTypeConverter(func(staticType *interpreter.CompositeStaticType) interpreter.StaticType {
					if staticType == fooACompositeType {
						// NOTE: return interface type as-is, not wrapped in intersection type,
						// to test if it gets wrapped properly into an intersection type
						return fooCInterfaceType
					}
					return nil
				})

			actual := migrate(t, staticTypeMigration, fooACompositeType)
			assert.Equal(t,
				interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooCInterfaceType,
					},
				),
				actual,
			)
		})

		t.Run("return intersection", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration().
				WithCompositeTypeConverter(func(staticType *interpreter.CompositeStaticType) interpreter.StaticType {
					if staticType == fooACompositeType {
						// NOTE: return interface type wrapped in intersection type,
						// to test if it does not get re-wrapped into an intersection type
						return interpreter.NewIntersectionStaticType(
							nil,
							[]*interpreter.InterfaceStaticType{
								fooCInterfaceType,
							},
						)
					}
					return nil
				})

			actual := migrate(t, staticTypeMigration, fooACompositeType)
			assert.Equal(t,
				interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooCInterfaceType,
					},
				),
				actual,
			)

		})
	})

	t.Run("interface type converter", func(t *testing.T) {
		t.Parallel()

		t.Run("return non-interface", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration().
				WithInterfaceTypeConverter(func(staticType *interpreter.InterfaceStaticType) interpreter.StaticType {
					if staticType == fooCInterfaceType {
						return fooBCompositeType
					}
					return nil
				})

			actual := migrate(t, staticTypeMigration, fooCInterfaceType)
			assert.Equal(t, fooBCompositeType, actual)
		})

		t.Run("return interface", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration().
				WithInterfaceTypeConverter(func(staticType *interpreter.InterfaceStaticType) interpreter.StaticType {
					if staticType == fooCInterfaceType {
						// NOTE: return interface type as-is, not wrapped in intersection type,
						// to test if it gets wrapped properly into an intersection type
						return fooDInterfaceType
					}
					return nil
				})

			actual := migrate(t, staticTypeMigration, fooCInterfaceType)
			assert.Equal(t,
				interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooDInterfaceType,
					},
				),
				actual,
			)

		})

		t.Run("return intersection", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration().
				WithInterfaceTypeConverter(func(staticType *interpreter.InterfaceStaticType) interpreter.StaticType {
					if staticType == fooCInterfaceType {
						// NOTE: return interface type wrapped in intersection type,
						// to test if it does not get re-wrapped into an intersection type
						return interpreter.NewIntersectionStaticType(
							nil,
							[]*interpreter.InterfaceStaticType{
								fooDInterfaceType,
							},
						)
					}
					return nil
				})

			actual := migrate(t, staticTypeMigration, fooCInterfaceType)
			assert.Equal(t,
				interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooDInterfaceType,
					},
				),
				actual,
			)
		})
	})
}
