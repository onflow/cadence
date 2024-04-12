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

		migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
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
			interpreter.NewUnmeteredTypeValue(nil),
			// NOTE: atree value validation is disabled,
			// because the type value has a nil type (which indicates an invalid or unknown type),
			// and invalid unknown types are always unequal
			false,
		)
		assert.Equal(t,
			interpreter.NewUnmeteredTypeValue(nil),
			actual,
		)
	})

	t.Run("TypeValue with unparameterized Capability type", func(t *testing.T) {
		t.Parallel()

		staticTypeMigration := NewStaticTypeMigration()

		actual := migrate(t,
			staticTypeMigration,
			interpreter.NewUnmeteredTypeValue(
				interpreter.NewCapabilityStaticType(nil, nil),
			),
			true,
		)
		assert.Equal(t,
			interpreter.NewUnmeteredTypeValue(
				interpreter.NewCapabilityStaticType(nil, nil),
			),
			actual,
		)
	})

	t.Run("TypeValue with reference to AuthAccount (as primitive)", func(t *testing.T) {
		t.Parallel()

		staticTypeMigration := NewStaticTypeMigration()

		actual := migrate(t,
			staticTypeMigration,
			interpreter.NewUnmeteredTypeValue(
				interpreter.NewDictionaryStaticType(nil,
					interpreter.PrimitiveStaticTypeAddress,
					interpreter.NewCapabilityStaticType(nil,
						interpreter.NewReferenceStaticType(
							nil,
							interpreter.UnauthorizedAccess,
							interpreter.PrimitiveStaticTypeAuthAccount, //nolint:staticcheck
						),
					),
				),
			),
			true,
		)
		assert.Equal(t,
			interpreter.NewUnmeteredTypeValue(
				interpreter.NewDictionaryStaticType(nil,
					interpreter.PrimitiveStaticTypeAddress,
					interpreter.NewCapabilityStaticType(nil,
						// NOTE: NOT reference to reference type
						authAccountReferenceType,
					),
				),
			),
			actual,
		)
	})

	t.Run("TypeValue with reference to AuthAccount (as composite)", func(t *testing.T) {
		t.Parallel()

		staticTypeMigration := NewStaticTypeMigration()

		authAccountTypeID := interpreter.PrimitiveStaticTypeAuthAccount.ID() //nolint:staticcheck

		actual := migrate(t,
			staticTypeMigration,
			interpreter.NewUnmeteredTypeValue(
				interpreter.NewDictionaryStaticType(nil,
					interpreter.PrimitiveStaticTypeAddress,
					interpreter.NewCapabilityStaticType(nil,
						interpreter.NewReferenceStaticType(
							nil,
							interpreter.UnauthorizedAccess,
							// NOTE: AuthAccount as composite type
							interpreter.NewCompositeStaticType(
								nil,
								nil,
								string(authAccountTypeID),
								authAccountTypeID,
							),
						),
					),
				),
			),
			true,
		)
		assert.Equal(t,
			interpreter.NewUnmeteredTypeValue(
				interpreter.NewDictionaryStaticType(nil,
					interpreter.PrimitiveStaticTypeAddress,
					interpreter.NewCapabilityStaticType(nil,
						// NOTE: NOT reference to reference type
						authAccountReferenceType,
					),
				),
			),
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

	t.Run("legacy type gets converted intersection", func(t *testing.T) {

		t.Parallel()

		const compositeQualifiedIdentifier = "S"
		compositeType := interpreter.NewCompositeStaticType(
			nil,
			utils.TestLocation,
			compositeQualifiedIdentifier,
			utils.TestLocation.TypeID(nil, compositeQualifiedIdentifier),
		)

		const interface1QualifiedIdentifier = "SI1"
		interfaceType1 := interpreter.NewInterfaceStaticType(
			nil,
			utils.TestLocation,
			interface1QualifiedIdentifier,
			utils.TestLocation.TypeID(nil, interface1QualifiedIdentifier),
		)

		const interface2QualifiedIdentifier = "SI2"
		interfaceType2 := interpreter.NewInterfaceStaticType(
			nil,
			utils.TestLocation,
			interface2QualifiedIdentifier,
			utils.TestLocation.TypeID(nil, interface2QualifiedIdentifier),
		)

		intersectionType := interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				interfaceType1,
			},
		)
		// NOTE: the legacy type is a composite type,
		// but it will get rewritten to an intersection type

		intersectionType.LegacyType = compositeType

		staticTypeMigration := NewStaticTypeMigration().WithCompositeTypeConverter(
			func(staticType *interpreter.CompositeStaticType) interpreter.StaticType {
				if staticType.TypeID != compositeType.TypeID {
					return nil
				}

				return interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						interfaceType2,
					},
				)
			},
		)

		storedValue := interpreter.NewTypeValue(
			nil,
			interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				intersectionType,
			),
		)

		actual := migrate(t,
			staticTypeMigration,
			storedValue,
			true,
		)

		// NOTE: the expected type {S2}{S1} is expected to be ("temporarily") invalid.
		// The entitlements migrations will handle such cases, i.e. rewrite the type to a valid type ({S2}).
		// This is important to ensure that the entitlement migration does not infer entitlements for {S1, S2}.

		expectedIntersection := interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				interfaceType1,
			},
		)
		expectedIntersection.LegacyType = interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				interfaceType2,
			},
		)

		expected := interpreter.NewTypeValue(
			nil,
			interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				expectedIntersection,
			),
		)

		assert.Equal(t, expected, actual)
	})

	t.Run(
		"composite types of (non-deprecated) built-in types are converted to primitive static types",
		func(t *testing.T) {
			t.Parallel()

			test := func(t *testing.T, ty interpreter.PrimitiveStaticType) {

				typeID := ty.ID()

				t.Run(string(typeID), func(t *testing.T) {
					t.Parallel()

					staticTypeMigration := NewStaticTypeMigration()

					actual := migrate(t,
						staticTypeMigration,
						interpreter.NewUnmeteredTypeValue(
							// NOTE: AuthAccount as composite type
							interpreter.NewCompositeStaticType(
								nil,
								nil,
								string(typeID),
								typeID,
							),
						),
						true,
					)
					assert.Equal(t,
						interpreter.NewUnmeteredTypeValue(ty),
						actual,
					)
				})
			}

			for ty := interpreter.PrimitiveStaticTypeUnknown + 1; ty < interpreter.PrimitiveStaticType_Count; ty++ {
				if !ty.IsDefined() || ty.IsDeprecated() { //nolint:staticcheck
					continue
				}

				test(t, ty)
			}
		},
	)

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

		migration, err := migrations.NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
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
					interpreter.NewAddressValue(nil, common.ZeroAddress),
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

func TestCanSkipStaticTypeMigration(t *testing.T) {

	t.Parallel()

	testCases := map[interpreter.StaticType]bool{

		// Primitive types, like Bool and Address

		interpreter.PrimitiveStaticTypeBool:    true,
		interpreter.PrimitiveStaticTypeAddress: true,

		// Number and Path types, like UInt8 and StoragePath

		interpreter.PrimitiveStaticTypeUInt8:       true,
		interpreter.PrimitiveStaticTypeStoragePath: true,

		// Capability types

		// Untyped capability, can skip
		interpreter.PrimitiveStaticTypeCapability: true,
		// Typed capabilities, cannot skip
		&interpreter.CapabilityStaticType{
			BorrowType: interpreter.PrimitiveStaticTypeString,
		}: false,
		&interpreter.CapabilityStaticType{
			BorrowType: interpreter.PrimitiveStaticTypeCharacter,
		}: false,

		// Existential types, like AnyStruct and AnyResource

		interpreter.PrimitiveStaticTypeAnyStruct:   false,
		interpreter.PrimitiveStaticTypeAnyResource: false,
	}

	test := func(ty interpreter.StaticType, expected bool) {

		t.Run(ty.String(), func(t *testing.T) {

			t.Parallel()

			t.Run("base", func(t *testing.T) {

				t.Parallel()

				actual := CanSkipStaticTypeMigration(ty)
				assert.Equal(t, expected, actual)

			})

			t.Run("optional", func(t *testing.T) {

				t.Parallel()

				optionalType := interpreter.NewOptionalStaticType(nil, ty)

				actual := CanSkipStaticTypeMigration(optionalType)
				assert.Equal(t, expected, actual)
			})

			t.Run("variable-sized", func(t *testing.T) {

				t.Parallel()

				arrayType := interpreter.NewVariableSizedStaticType(nil, ty)

				actual := CanSkipStaticTypeMigration(arrayType)
				assert.Equal(t, expected, actual)
			})

			t.Run("constant-sized", func(t *testing.T) {

				t.Parallel()

				arrayType := interpreter.NewConstantSizedStaticType(nil, ty, 2)

				actual := CanSkipStaticTypeMigration(arrayType)
				assert.Equal(t, expected, actual)
			})

			t.Run("dictionary key", func(t *testing.T) {

				t.Parallel()

				dictionaryType := interpreter.NewDictionaryStaticType(
					nil,
					ty,
					interpreter.PrimitiveStaticTypeInt,
				)

				actual := CanSkipStaticTypeMigration(dictionaryType)
				assert.Equal(t, expected, actual)

			})

			t.Run("dictionary value", func(t *testing.T) {

				t.Parallel()

				dictionaryType := interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt,
					ty,
				)

				actual := CanSkipStaticTypeMigration(dictionaryType)
				assert.Equal(t, expected, actual)
			})
		})
	}

	for ty, expected := range testCases {
		test(ty, expected)
	}
}
