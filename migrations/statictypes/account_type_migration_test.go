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
	"fmt"
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

var _ migrations.Reporter = &testReporter{}

type testReporter struct {
	migrated map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}]struct{}
	errors []error
}

func newTestReporter() *testReporter {
	return &testReporter{
		migrated: map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}]struct{}{},
	}
}

func (t *testReporter) Migrated(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	_ string,
) {
	key := struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}{
		StorageKey:    storageKey,
		StorageMapKey: storageMapKey,
	}
	t.migrated[key] = struct{}{}
}

func (t *testReporter) Error(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	_ string,
	err error,
) {
	t.errors = append(t.errors, err)
}

func TestAccountTypeInTypeValueMigration(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}
	pathDomain := common.PathDomainPublic

	const publicAccountType = interpreter.PrimitiveStaticTypePublicAccount //nolint:staticcheck
	const authAccountType = interpreter.PrimitiveStaticTypeAuthAccount     //nolint:staticcheck
	const stringType = interpreter.PrimitiveStaticTypeString

	const fooBarQualifiedIdentifier = "Foo.Bar"
	fooAddressLocation := common.NewAddressLocation(nil, account, "Foo")

	type testCase struct {
		storedType   interpreter.StaticType
		expectedType interpreter.StaticType
	}

	testCases := map[string]testCase{
		"public_account": {
			storedType:   publicAccountType,
			expectedType: unauthorizedAccountReferenceType,
		},
		"auth_account": {
			storedType:   authAccountType,
			expectedType: authAccountReferenceType,
		},
		"auth_account_capabilities": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountCapabilities, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_Capabilities,
		},
		"public_account_capabilities": {
			storedType:   interpreter.PrimitiveStaticTypePublicAccountCapabilities, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_Capabilities,
		},
		"auth_account_account_capabilities": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_AccountCapabilities,
		},
		"auth_account_storage_capabilities": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_StorageCapabilities,
		},
		"auth_account_contracts": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountContracts, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_Contracts,
		},
		"public_account_contracts": {
			storedType:   interpreter.PrimitiveStaticTypePublicAccountContracts, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_Contracts,
		},
		"auth_account_keys": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountKeys, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_Keys,
		},
		"public_account_keys": {
			storedType:   interpreter.PrimitiveStaticTypePublicAccountKeys, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_Keys,
		},
		"auth_account_inbox": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountInbox, //nolint:staticcheck
			expectedType: interpreter.PrimitiveStaticTypeAccount_Inbox,
		},
		"account_key": {
			storedType:   interpreter.PrimitiveStaticTypeAccountKey, //nolint:staticcheck
			expectedType: interpreter.AccountKeyStaticType,
		},
		"optional_account": {
			storedType:   interpreter.NewOptionalStaticType(nil, publicAccountType),
			expectedType: interpreter.NewOptionalStaticType(nil, unauthorizedAccountReferenceType),
		},
		"optional_string": {
			storedType:   interpreter.NewOptionalStaticType(nil, stringType),
			expectedType: nil,
		},
		"constant_sized_account_array": {
			storedType:   interpreter.NewConstantSizedStaticType(nil, publicAccountType, 3),
			expectedType: interpreter.NewConstantSizedStaticType(nil, unauthorizedAccountReferenceType, 3),
		},
		"constant_sized_string_array": {
			storedType:   interpreter.NewConstantSizedStaticType(nil, stringType, 3),
			expectedType: nil,
		},
		"variable_sized_account_array": {
			storedType:   interpreter.NewVariableSizedStaticType(nil, authAccountType),
			expectedType: interpreter.NewVariableSizedStaticType(nil, authAccountReferenceType),
		},
		"variable_sized_string_array": {
			storedType:   interpreter.NewVariableSizedStaticType(nil, stringType),
			expectedType: nil,
		},
		"dictionary_with_account_type_value": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				authAccountType,
			),
			expectedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				authAccountReferenceType,
			),
		},
		"dictionary_with_account_type_key": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				authAccountType,
				stringType,
			),
			expectedType: interpreter.NewDictionaryStaticType(
				nil,
				authAccountReferenceType,
				stringType,
			),
		},
		"dictionary_with_account_type_key_and_value": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				authAccountType,
				authAccountType,
			),
			expectedType: interpreter.NewDictionaryStaticType(
				nil,
				authAccountReferenceType,
				authAccountReferenceType,
			),
		},
		"string_dictionary": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				stringType,
			),
			expectedType: nil,
		},
		"capability": {
			storedType: interpreter.NewCapabilityStaticType(
				nil,
				publicAccountType,
			),
			expectedType: interpreter.NewCapabilityStaticType(
				nil,
				unauthorizedAccountReferenceType,
			),
		},
		"string_capability": {
			storedType: interpreter.NewCapabilityStaticType(
				nil,
				stringType,
			),
			expectedType: nil,
		},
		"intersection": {
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
		"intersection_with_legacy_type": {
			storedType: &interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{},
				LegacyType: interpreter.NewCompositeStaticType(
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
			expectedType: interpreter.NewCompositeStaticType(
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
		"public_account_reference": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				publicAccountType,
			),
			expectedType: unauthorizedAccountReferenceType,
		},
		"public_account_auth_reference": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				publicAccountType,
			),
			expectedType: unauthorizedAccountReferenceType,
		},
		"auth_account_reference": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				authAccountType,
			),
			expectedType: authAccountReferenceType,
		},
		"auth_account_auth_reference": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				authAccountType,
			),
			expectedType: authAccountReferenceType,
		},
		"string_reference": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				stringType,
			),
		},
		"account_array_reference": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				interpreter.NewVariableSizedStaticType(nil, authAccountType),
			),
			expectedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				interpreter.NewVariableSizedStaticType(nil, authAccountReferenceType),
			),
		},
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
		"composite": {
			storedType: interpreter.NewCompositeStaticType(
				nil,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBarQualifiedIdentifier,
				),
			),
			expectedType: nil,
		},

		// reference to optionals
		"reference_to_optional": {
			storedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				interpreter.NewOptionalStaticType(
					nil,
					interpreter.PrimitiveStaticTypeAccountKey, //nolint:staticcheck
				),
			),
			expectedType: interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				interpreter.NewOptionalStaticType(
					nil,
					interpreter.AccountKeyStaticType,
				),
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
				account,
				pathDomain,
				name,
				testCase.storedType,
			)

			err = storage.Commit(inter, true)
			require.NoError(t, err)

			// Migrate

			migration := migrations.NewStorageMigration(inter, storage)

			reporter := newTestReporter()

			migration.Migrate(
				&migrations.AddressSliceIterator{
					Addresses: []common.Address{
						account,
					},
				},
				migration.NewValueMigrationsPathMigrator(
					reporter,
					NewStaticTypeMigration(),
				),
			)

			err = migration.Commit()
			require.NoError(t, err)

			err = storage.CheckHealth()
			require.NoError(t, err)

			require.Empty(t, reporter.errors)

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
								Address: account,
								Key:     pathDomain.Identifier(),
							},
							StorageMapKey: storageMapKey,
						}: {},
					},
					reporter.migrated)
			}

			// Assert the migrated values.

			storageMap := storage.GetStorageMap(account, pathDomain.Identifier(), false)
			require.NotNil(t, storageMap)
			require.Equal(t, storageMap.Count(), uint64(1))

			value := storageMap.ReadValue(nil, storageMapKey)

			var expectedValue interpreter.Value
			if testCase.expectedType != nil {
				expectedValue = interpreter.NewTypeValue(nil, testCase.expectedType)

				// `IntersectionType.LegacyType` is not considered in the `IntersectionType.Equal` method.
				// Therefore, check for the legacy type equality manually.
				typeValue := value.(interpreter.TypeValue)
				if actualIntersectionType, ok := typeValue.Type.(*interpreter.IntersectionStaticType); ok {
					expectedIntersectionType := testCase.expectedType.(*interpreter.IntersectionStaticType)

					if actualIntersectionType.LegacyType != nil {
						assert.True(t,
							actualIntersectionType.LegacyType.
								Equal(expectedIntersectionType.LegacyType),
						)
					} else if expectedIntersectionType.LegacyType != nil {
						assert.True(t,
							expectedIntersectionType.LegacyType.
								Equal(actualIntersectionType.LegacyType),
						)
					} else {
						assert.Equal(t,
							expectedIntersectionType.LegacyType,
							actualIntersectionType.LegacyType,
						)
					}
				}
			} else {
				expectedValue = interpreter.NewTypeValue(nil, testCase.storedType)
			}

			utils.AssertValuesEqual(t, inter, expectedValue, value)
		})
	}

	for name, testCase := range testCases {
		test(name, testCase)
	}
}

func storeTypeValue(
	inter *interpreter.Interpreter,
	address common.Address,
	domain common.PathDomain,
	pathIdentifier string,
	staticType interpreter.StaticType,
) {
	inter.WriteStored(
		address,
		domain.Identifier(),
		interpreter.StringStorageMapKey(pathIdentifier),
		interpreter.NewTypeValue(inter, staticType),
	)
}

func TestAccountTypeInNestedTypeValueMigration(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}
	pathDomain := common.PathDomainPublic

	type testCase struct {
		storedValue   interpreter.Value
		expectedValue interpreter.Value
	}

	storedAccountTypeValue := interpreter.NewTypeValue(nil, interpreter.PrimitiveStaticTypePublicAccount) //nolint:staticcheck
	expectedAccountTypeValue := interpreter.NewTypeValue(nil, unauthorizedAccountReferenceType)
	stringTypeValue := interpreter.NewTypeValue(nil, interpreter.PrimitiveStaticTypeString)

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

	fooAddressLocation := common.NewAddressLocation(nil, account, "Foo")
	const fooBarQualifiedIdentifier = "Foo.Bar"

	testCases := map[string]testCase{
		"account_some_value": {
			storedValue:   interpreter.NewUnmeteredSomeValueNonCopying(storedAccountTypeValue),
			expectedValue: interpreter.NewUnmeteredSomeValueNonCopying(expectedAccountTypeValue),
		},
		"int8_some_value": {
			storedValue: interpreter.NewUnmeteredSomeValueNonCopying(stringTypeValue),
		},
		"account_array": {
			storedValue: interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct),
				common.ZeroAddress,
				stringTypeValue,
				storedAccountTypeValue,
				stringTypeValue,
				stringTypeValue,
				storedAccountTypeValue,
			),
			expectedValue: interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct),
				common.ZeroAddress,
				stringTypeValue,
				expectedAccountTypeValue,
				stringTypeValue,
				stringTypeValue,
				expectedAccountTypeValue,
			),
		},
		"non_account_array": {
			storedValue: interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct),
				common.ZeroAddress,
				stringTypeValue,
				stringTypeValue,
				stringTypeValue,
			),
		},
		"dictionary_with_account_type_value": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt8,
					interpreter.PrimitiveStaticTypeAnyStruct,
				),
				interpreter.NewUnmeteredInt8Value(4),
				storedAccountTypeValue,
				interpreter.NewUnmeteredInt8Value(5),
				interpreter.NewUnmeteredStringValue("hello"),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt8,
					interpreter.PrimitiveStaticTypeAnyStruct,
				),
				interpreter.NewUnmeteredInt8Value(4),
				expectedAccountTypeValue,
				interpreter.NewUnmeteredInt8Value(5),
				interpreter.NewUnmeteredStringValue("hello"),
			),
		},
		"dictionary_with_optional_account_type_value": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt8,
					interpreter.NewOptionalStaticType(nil, interpreter.PrimitiveStaticTypeMetaType),
				),
				interpreter.NewUnmeteredInt8Value(4),
				interpreter.NewUnmeteredSomeValueNonCopying(storedAccountTypeValue),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeInt8,
					interpreter.NewOptionalStaticType(nil, interpreter.PrimitiveStaticTypeMetaType),
				),
				interpreter.NewUnmeteredInt8Value(4),
				interpreter.NewUnmeteredSomeValueNonCopying(expectedAccountTypeValue),
			),
		},
		"dictionary_with_account_type_key": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeMetaType,
					interpreter.PrimitiveStaticTypeInt8,
				),
				interpreter.NewTypeValue(
					nil,
					dummyStaticType{
						PrimitiveStaticType: interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
					},
				),
				interpreter.NewUnmeteredInt8Value(4),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeMetaType,
					interpreter.PrimitiveStaticTypeInt8,
				),
				expectedAccountTypeValue,
				interpreter.NewUnmeteredInt8Value(4),
			),
		},
		"dictionary_with_account_type_key_and_value": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeMetaType,
					interpreter.PrimitiveStaticTypeMetaType,
				),
				interpreter.NewTypeValue(
					nil,
					dummyStaticType{
						PrimitiveStaticType: interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
					},
				),
				storedAccountTypeValue,
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeMetaType,
					interpreter.PrimitiveStaticTypeMetaType,
				),
				expectedAccountTypeValue,
				expectedAccountTypeValue,
			),
		},
		"composite_with_account_type": {
			storedValue: interpreter.NewCompositeValue(
				inter,
				interpreter.EmptyLocationRange,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
				common.CompositeKindResource,
				[]interpreter.CompositeField{
					interpreter.NewUnmeteredCompositeField("field1", storedAccountTypeValue),
					interpreter.NewUnmeteredCompositeField("field2", interpreter.NewUnmeteredStringValue("hello")),
				},
				common.Address{},
			),
			expectedValue: interpreter.NewCompositeValue(
				inter,
				interpreter.EmptyLocationRange,
				fooAddressLocation,
				fooBarQualifiedIdentifier,
				common.CompositeKindResource,
				[]interpreter.CompositeField{
					interpreter.NewUnmeteredCompositeField("field1", expectedAccountTypeValue),
					interpreter.NewUnmeteredCompositeField("field2", interpreter.NewUnmeteredStringValue("hello")),
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

	reporter := newTestReporter()

	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			reporter,
			NewStaticTypeMigration(),
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	err = storage.CheckHealth()
	require.NoError(t, err)

	require.Empty(t, reporter.errors)

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

func TestMigratingValuesWithAccountStaticType(t *testing.T) {

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
		"dictionary_value": {
			storedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
				),
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					unauthorizedAccountReferenceType,
				),
			),
		},
		"array_value": {
			storedValue: interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
				),
				common.Address{},
			),
			expectedValue: interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(
					nil,
					unauthorizedAccountReferenceType,
				),
				common.Address{},
			),
		},
		"account_capability_value": {
			storedValue: interpreter.NewUnmeteredCapabilityValue(
				123,
				interpreter.NewAddressValue(nil, common.Address{0x42}),
				interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
			),
			expectedValue: interpreter.NewUnmeteredCapabilityValue(
				123,
				interpreter.NewAddressValue(nil, common.Address{0x42}),
				unauthorizedAccountReferenceType,
			),
		},
		"string_capability_value": {
			storedValue: interpreter.NewUnmeteredCapabilityValue(
				123,
				interpreter.NewAddressValue(nil, common.Address{0x42}),
				interpreter.PrimitiveStaticTypeString,
			),
		},
		"account_capability_controller": {
			storedValue: interpreter.NewUnmeteredAccountCapabilityControllerValue(
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					interpreter.PrimitiveStaticTypeAuthAccount, //nolint:staticcheck,
				),
				1234,
			),
			expectedValue: interpreter.NewUnmeteredAccountCapabilityControllerValue(
				authAccountReferenceType,
				1234,
			),
		},
		"storage_capability_controller": {
			storedValue: interpreter.NewUnmeteredStorageCapabilityControllerValue(
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck,
				),
				1234,
				interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "v1"),
			),
			expectedValue: interpreter.NewUnmeteredStorageCapabilityControllerValue(
				unauthorizedAccountReferenceType,
				1234,
				interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "v1"),
			),
		},
		"path_link_value": {
			storedValue: interpreter.PathLinkValue{ //nolint:staticcheck
				TargetPath: interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "v1"),
				Type:       interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
			},
			expectedValue: interpreter.PathLinkValue{ //nolint:staticcheck
				TargetPath: interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "v1"),
				Type:       unauthorizedAccountReferenceType,
			},
		},
		"account_link_value": {
			storedValue:   interpreter.AccountLinkValue{}, //nolint:staticcheck
			expectedValue: interpreter.AccountLinkValue{}, //nolint:staticcheck
		},
		"path_capability_value": {
			storedValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				Address:    interpreter.NewAddressValue(nil, common.Address{0x42}),
				Path:       interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "v1"),
				BorrowType: interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
			},
			expectedValue: &interpreter.PathCapabilityValue{ //nolint:staticcheck
				Address:    interpreter.NewAddressValue(nil, common.Address{0x42}),
				Path:       interpreter.NewUnmeteredPathValue(common.PathDomainStorage, "v1"),
				BorrowType: unauthorizedAccountReferenceType,
			},
		},
		"capability_dictionary": {
			storedValue: interpreter.NewDictionaryValue(
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
					interpreter.NewAddressValue(nil, common.Address{}),
					interpreter.PrimitiveStaticTypePublicAccount, //nolint:staticcheck
				),
			),
			expectedValue: interpreter.NewDictionaryValue(
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

	reporter := newTestReporter()

	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			reporter,
			NewStaticTypeMigration(),
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	err = storage.CheckHealth()
	require.NoError(t, err)

	require.Empty(t, reporter.errors)

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

var testAddress = common.Address{0x42}

func TestAccountTypeRehash(t *testing.T) {

	t.Parallel()

	locationRange := interpreter.EmptyLocationRange

	ledger := NewTestLedger(nil, nil)

	storageMapKey := interpreter.StringStorageMapKey("dict")
	newStringValue := func(s string) interpreter.Value {
		return interpreter.NewUnmeteredStringValue(s)
	}

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
		storage := runtime.NewStorage(ledger, nil)
		inter, err := interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage: storage,
				// NOTE: atree value validation is disabled
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
			interpreter.PrimitiveStaticTypeMetaType,
			interpreter.PrimitiveStaticTypeString,
		)
		dictValue := interpreter.NewDictionaryValue(inter, locationRange, dictionaryStaticType)

		accountTypes := []interpreter.PrimitiveStaticType{
			interpreter.PrimitiveStaticTypePublicAccount,                  //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccount,                    //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountCapabilities,        //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountCapabilities,      //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities, //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities, //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountContracts,           //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountContracts,         //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountKeys,                //nolint:staticcheck
			interpreter.PrimitiveStaticTypePublicAccountKeys,              //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAuthAccountInbox,               //nolint:staticcheck
			interpreter.PrimitiveStaticTypeAccountKey,                     //nolint:staticcheck
		}

		for _, typ := range accountTypes {
			typeValue := interpreter.NewUnmeteredTypeValue(
				migrations.LegacyPrimitiveStaticType{
					PrimitiveStaticType: typ,
				},
			)
			dictValue.Insert(
				inter,
				locationRange,
				typeValue,
				newStringValue(typ.String()),
			)
		}

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
				NewStaticTypeMigration(),
			),
		)

		err := migration.Commit()
		require.NoError(t, err)

		err = storage.CheckHealth()
		require.NoError(t, err)

		require.Empty(t, reporter.errors)

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

		var existingKeys []interpreter.Value
		dictValue.Iterate(inter, func(key, value interpreter.Value) (resume bool) {
			existingKeys = append(existingKeys, key)
			// continue iteration
			return true
		})

		for _, key := range existingKeys {
			actual := dictValue.Remove(
				inter,
				interpreter.EmptyLocationRange,
				key,
			)

			assert.NotNil(t, actual)

			staticType := key.(interpreter.TypeValue).Type

			var possibleExpectedValues []interpreter.Value
			var str string

			switch {
			case staticType.Equal(unauthorizedAccountReferenceType):
				str = "PublicAccount"
			case staticType.Equal(authAccountReferenceType):
				str = "AuthAccount"
			case staticType.Equal(interpreter.PrimitiveStaticTypeAccount_Capabilities):
				// For both `AuthAccount.Capabilities` and `PublicAccount.Capabilities`,
				// the migrated key is the same (`Account_Capabilities`).
				// So the value at the key could be any of the two original values,
				// depending on the order of migration.
				possibleExpectedValues = []interpreter.Value{
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredStringValue("AuthAccountCapabilities"),
					),
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredStringValue("PublicAccountCapabilities"),
					),
				}
			case staticType.Equal(interpreter.PrimitiveStaticTypeAccount_AccountCapabilities):
				str = "AuthAccountAccountCapabilities"
			case staticType.Equal(interpreter.PrimitiveStaticTypeAccount_StorageCapabilities):
				str = "AuthAccountStorageCapabilities"
			case staticType.Equal(interpreter.PrimitiveStaticTypeAccount_Contracts):
				// For both `AuthAccount.Contracts` and `PublicAccount.Contracts`,
				// the migrated key is the same (Account_Contracts).
				// So the value at the key could be any of the two original values,
				// depending on the order of migration.
				possibleExpectedValues = []interpreter.Value{
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredStringValue("AuthAccountContracts"),
					),
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredStringValue("PublicAccountContracts"),
					),
				}
			case staticType.Equal(interpreter.PrimitiveStaticTypeAccount_Keys):
				// For both `AuthAccount.Keys` and `PublicAccount.Keys`,
				// the migrated key is the same (Account_Keys).
				// So the value at the key could be any of the two original values,
				// depending on the order of migration.
				possibleExpectedValues = []interpreter.Value{
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredStringValue("AuthAccountKeys"),
					),
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredStringValue("PublicAccountKeys"),
					),
				}
			case staticType.Equal(interpreter.PrimitiveStaticTypeAccount_Inbox):
				str = "AuthAccountInbox"
			case staticType.Equal(interpreter.AccountKeyStaticType):
				str = "AccountKey"
			default:
				require.Fail(t, fmt.Sprintf("Unexpected type `%s` in dictionary key", staticType.ID()))
			}

			if possibleExpectedValues != nil {
				assert.Contains(t, possibleExpectedValues, actual)
			} else {
				expected := interpreter.NewUnmeteredSomeValueNonCopying(
					interpreter.NewUnmeteredStringValue(str),
				)
				assert.Equal(t, expected, actual)
			}
		}
	})
}
