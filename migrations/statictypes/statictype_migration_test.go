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

	"github.com/onflow/atree"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestStaticTypeMigration(t *testing.T) {
	t.Parallel()

	migrate := func(
		t *testing.T,
		staticTypeMigration *StaticTypeMigration,
		value interpreter.Value,
		atreeValueValidationEnabled bool,
	) interpreter.Value {

		// Store values

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(ledger, nil)

		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage:                       storage,
				AtreeValueValidationEnabled:   atreeValueValidationEnabled,
				AtreeStorageValidationEnabled: true,
			},
		)
		require.NoError(t, err)

		storageMapKey := interpreter.StringStorageMapKey("test_type_value")
		storageDomain := common.PathDomainStorage.Identifier()

		inter.WriteStored(
			testAddress,
			storageDomain,
			storageMapKey,
			value,
		)

		err = storage.Commit(inter, true)
		require.NoError(t, err)

		// Migrate

		migration := migrations.NewStorageMigration(inter, storage)

		reporter := newTestReporter()

		migration.Migrate(
			&migrations.AddressSliceIterator{
				Addresses: []common.Address{
					testAddress,
				},
			},
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

		storageMap := storage.GetStorageMap(
			testAddress,
			storageDomain,
			false,
		)
		require.NotNil(t, storageMap)
		require.Equal(t, uint64(1), storageMap.Count())

		result := storageMap.ReadValue(nil, storageMapKey)
		require.NotNil(t, value)

		return result
	}

	t.Run("TypeValue with nil type", func(t *testing.T) {
		t.Parallel()

		staticTypeMigration := NewStaticTypeMigration()

		actual := migrate(t,
			staticTypeMigration,
			interpreter.NewTypeValue(nil, nil),
			// NOTE: atree value validation is disabled,
			// because the type value has a nil type (which indicates an invalid or unknown type),
			// and invalid unknown types are always unequal
			false,
		)
		assert.Equal(t,
			interpreter.NewTypeValue(nil, nil),
			actual,
		)
	})

	t.Run("PathCapabilityValue with nil borrow type", func(t *testing.T) {
		t.Parallel()

		staticTypeMigration := NewStaticTypeMigration()

		path := interpreter.NewUnmeteredPathValue(
			common.PathDomainStorage,
			"test",
		)

		actual := migrate(t,
			staticTypeMigration,
			&interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: nil,
				Path:       path,
				Address:    interpreter.AddressValue(testAddress),
			},
			true,
		)
		assert.Equal(t,
			&interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: nil,
				Path:       path,
				Address:    interpreter.AddressValue(testAddress),
			},
			actual,
		)
	})

	t.Run("T{I,...} -> T, for T != AnyStruct/AnyResource", func(t *testing.T) {
		t.Parallel()

		t.Run("T{I} -> T", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						Types: []*interpreter.InterfaceStaticType{
							{
								Location:            nil,
								QualifiedIdentifier: "I",
								TypeID:              "I",
							},
						},
						LegacyType: &interpreter.CompositeStaticType{
							Location:            nil,
							QualifiedIdentifier: "T",
							TypeID:              "T",
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.CompositeStaticType{
						Location:            nil,
						QualifiedIdentifier: "T",
						TypeID:              "T",
					},
				),
				actual,
			)
		})

		t.Run("&T{I} -> &T{I}", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types: []*interpreter.InterfaceStaticType{
								{
									Location:            nil,
									QualifiedIdentifier: "I",
									TypeID:              "I",
								},
							},
							LegacyType: &interpreter.CompositeStaticType{
								Location:            nil,
								QualifiedIdentifier: "T",
								TypeID:              "T",
							},
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types: []*interpreter.InterfaceStaticType{
								{
									Location:            nil,
									QualifiedIdentifier: "I",
									TypeID:              "I",
								},
							},
							LegacyType: &interpreter.CompositeStaticType{
								Location:            nil,
								QualifiedIdentifier: "T",
								TypeID:              "T",
							},
						},
					},
				),
				actual,
			)
		})
	})

	t.Run("T{I,...} -> {I,...}, for T == AnyStruct/AnyResource", func(t *testing.T) {
		t.Parallel()

		t.Run("AnyStruct{I} -> {I}", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						Types: []*interpreter.InterfaceStaticType{
							{
								Location:            nil,
								QualifiedIdentifier: "I",
								TypeID:              "I",
							},
						},
						LegacyType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						Types: []*interpreter.InterfaceStaticType{
							{
								Location:            nil,
								QualifiedIdentifier: "I",
								TypeID:              "I",
							},
						},
					},
				),
				actual,
			)
		})

		t.Run("&AnyStruct{I} -> &AnyStruct{I}", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types: []*interpreter.InterfaceStaticType{
								{
									Location:            nil,
									QualifiedIdentifier: "I",
									TypeID:              "I",
								},
							},
							LegacyType: interpreter.PrimitiveStaticTypeAnyStruct,
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types: []*interpreter.InterfaceStaticType{
								{
									Location:            nil,
									QualifiedIdentifier: "I",
									TypeID:              "I",
								},
							},
							LegacyType: interpreter.PrimitiveStaticTypeAnyStruct,
						},
					},
				),
				actual,
			)
		})

		t.Run("AnyResource{I} -> {I}", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						Types: []*interpreter.InterfaceStaticType{
							{
								Location:            nil,
								QualifiedIdentifier: "I",
								TypeID:              "I",
							},
						},
						LegacyType: interpreter.PrimitiveStaticTypeAnyResource,
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						Types: []*interpreter.InterfaceStaticType{
							{
								Location:            nil,
								QualifiedIdentifier: "I",
								TypeID:              "I",
							},
						},
					},
				),
				actual,
			)
		})

		t.Run("&AnyResource{I} -> &AnyResource{I}", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types: []*interpreter.InterfaceStaticType{
								{
									Location:            nil,
									QualifiedIdentifier: "I",
									TypeID:              "I",
								},
							},
							LegacyType: interpreter.PrimitiveStaticTypeAnyResource,
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types: []*interpreter.InterfaceStaticType{
								{
									Location:            nil,
									QualifiedIdentifier: "I",
									TypeID:              "I",
								},
							},
							LegacyType: interpreter.PrimitiveStaticTypeAnyResource,
						},
					},
				),
				actual,
			)
		})
	})

	t.Run("T{} -> T, for any T", func(t *testing.T) {
		t.Parallel()

		t.Run("AnyStruct{} -> AnyStruct", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						Types:      []*interpreter.InterfaceStaticType{},
						LegacyType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					interpreter.PrimitiveStaticTypeAnyStruct,
				),
				actual,
			)
		})

		t.Run("&AnyStruct{} -> &AnyStruct{}", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types:      []*interpreter.InterfaceStaticType{},
							LegacyType: interpreter.PrimitiveStaticTypeAnyStruct,
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types:      []*interpreter.InterfaceStaticType{},
							LegacyType: interpreter.PrimitiveStaticTypeAnyStruct,
						},
					},
				),
				actual,
			)
		})

		t.Run("AnyResource{} -> AnyResource", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						Types:      []*interpreter.InterfaceStaticType{},
						LegacyType: interpreter.PrimitiveStaticTypeAnyResource,
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					interpreter.PrimitiveStaticTypeAnyResource,
				),
				actual,
			)
		})

		t.Run("&AnyResource{} -> &AnyResource{}", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types:      []*interpreter.InterfaceStaticType{},
							LegacyType: interpreter.PrimitiveStaticTypeAnyResource,
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							Types:      []*interpreter.InterfaceStaticType{},
							LegacyType: interpreter.PrimitiveStaticTypeAnyResource,
						},
					},
				),
				actual,
			)
		})

		t.Run("T{} -> T", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.IntersectionStaticType{
						LegacyType: &interpreter.CompositeStaticType{
							Location:            nil,
							QualifiedIdentifier: "T",
							TypeID:              "T",
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.CompositeStaticType{
						Location:            nil,
						QualifiedIdentifier: "T",
						TypeID:              "T",
					},
				),
				actual,
			)
		})

		t.Run("&T{} -> &T", func(t *testing.T) {
			t.Parallel()

			staticTypeMigration := NewStaticTypeMigration()

			actual := migrate(t,
				staticTypeMigration,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							LegacyType: &interpreter.CompositeStaticType{
								Location:            nil,
								QualifiedIdentifier: "T",
								TypeID:              "T",
							},
						},
					},
				),
				true,
			)
			assert.Equal(t,
				interpreter.NewUnmeteredTypeValue(
					&interpreter.ReferenceStaticType{
						Authorization: interpreter.Unauthorized{},
						ReferencedType: &interpreter.IntersectionStaticType{
							LegacyType: &interpreter.CompositeStaticType{
								Location:            nil,
								QualifiedIdentifier: "T",
								TypeID:              "T",
							},
						},
					},
				),
				actual,
			)
		})

	})
}

func TestMigratingNestedContainers(t *testing.T) {
	t.Parallel()

	migrate := func(
		t *testing.T,
		staticTypeMigration *StaticTypeMigration,
		storage *runtime.Storage,
		inter *interpreter.Interpreter,
		value interpreter.Value,
	) interpreter.Value {

		// Store values

		storageMapKey := interpreter.StringStorageMapKey("test_value")
		storageDomain := common.PathDomainStorage.Identifier()

		value = value.Transfer(
			inter,
			interpreter.EmptyLocationRange,
			atree.Address(testAddress),
			false,
			nil,
			nil,
		)

		inter.WriteStored(
			testAddress,
			storageDomain,
			storageMapKey,
			value,
		)

		err := storage.Commit(inter, true)
		require.NoError(t, err)

		// Migrate

		migration := migrations.NewStorageMigration(inter, storage)

		reporter := newTestReporter()

		migration.Migrate(
			&migrations.AddressSliceIterator{
				Addresses: []common.Address{
					testAddress,
				},
			},
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

		storageMap := storage.GetStorageMap(
			testAddress,
			storageDomain,
			false,
		)
		require.NotNil(t, storageMap)
		require.Equal(t, uint64(1), storageMap.Count())

		result := storageMap.ReadValue(nil, storageMapKey)
		require.NotNil(t, value)

		return result
	}

	t.Run("nested dictionary", func(t *testing.T) {
		t.Parallel()

		staticTypeMigration := NewStaticTypeMigration()

		locationRange := interpreter.EmptyLocationRange

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(ledger, nil)

		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage:                     storage,
				AtreeValueValidationEnabled: true,
				// NOTE: disabled, as storage is not expected to be always valid _during_ migration
				AtreeStorageValidationEnabled: false,
			},
		)
		require.NoError(t, err)

		storedValue := interpreter.NewDictionaryValue(
			inter,
			locationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeString,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.NewCapabilityStaticType(
						nil,
						interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
					),
				),
			),
			interpreter.NewUnmeteredStringValue("key"),
			interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.NewCapabilityStaticType(
						nil,
						interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
					),
				),
				interpreter.NewUnmeteredStringValue("key"),
				interpreter.NewCapabilityValue(
					nil,
					interpreter.NewUnmeteredUInt64Value(1234),
					interpreter.NewAddressValue(nil, common.ZeroAddress),
					interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
				),
			),
		)

		actual := migrate(t,
			staticTypeMigration,
			storage,
			inter,
			storedValue,
		)

		expected := interpreter.NewDictionaryValue(
			inter,
			locationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeString,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.NewCapabilityStaticType(
						nil,
						unauthorizedAccountReferenceType,
					),
				),
			),
			interpreter.NewUnmeteredStringValue("key"),
			interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.NewCapabilityStaticType(
						nil,
						unauthorizedAccountReferenceType,
					),
				),
				interpreter.NewUnmeteredStringValue("key"),
				interpreter.NewCapabilityValue(
					nil,
					interpreter.NewUnmeteredUInt64Value(1234),
					interpreter.NewAddressValue(nil, common.Address{}),
					unauthorizedAccountReferenceType,
				),
			),
		)

		utils.AssertValuesEqual(t, inter, expected, actual)
	})

	t.Run("nested arrays", func(t *testing.T) {
		t.Parallel()

		staticTypeMigration := NewStaticTypeMigration()

		locationRange := interpreter.EmptyLocationRange

		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(ledger, nil)

		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage:                     storage,
				AtreeValueValidationEnabled: true,
				// NOTE: disabled, as storage is not expected to be always valid _during_ migration
				AtreeStorageValidationEnabled: false,
			},
		)
		require.NoError(t, err)

		storedValue := interpreter.NewArrayValue(
			inter,
			locationRange,
			interpreter.NewVariableSizedStaticType(
				nil,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.NewCapabilityStaticType(
						nil,
						interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
					),
				),
			),
			common.ZeroAddress,
			interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.NewCapabilityStaticType(
						nil,
						interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
					),
				),
				common.ZeroAddress,
				interpreter.NewCapabilityValue(
					nil,
					interpreter.NewUnmeteredUInt64Value(1234),
					interpreter.NewAddressValue(nil, common.ZeroAddress),
					interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
				),
			),
		)

		actual := migrate(t,
			staticTypeMigration,
			storage,
			inter,
			storedValue,
		)

		expected := interpreter.NewArrayValue(
			inter,
			locationRange,
			interpreter.NewVariableSizedStaticType(
				nil,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.NewCapabilityStaticType(
						nil,
						unauthorizedAccountReferenceType,
					),
				),
			),
			common.ZeroAddress,
			interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.NewCapabilityStaticType(
						nil,
						unauthorizedAccountReferenceType,
					),
				),
				common.ZeroAddress,
				interpreter.NewCapabilityValue(
					nil,
					interpreter.NewUnmeteredUInt64Value(1234),
					interpreter.NewAddressValue(nil, common.Address{}),
					unauthorizedAccountReferenceType,
				),
			),
		)

		utils.AssertValuesEqual(t, inter, expected, actual)
	})

}
