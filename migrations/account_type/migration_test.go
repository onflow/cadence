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

package account_type

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

var _ migrations.Reporter = &testReporter{}

type testReporter struct {
	migratedPaths map[interpreter.AddressPath]struct{}
}

func newTestReporter() *testReporter {
	return &testReporter{
		migratedPaths: map[interpreter.AddressPath]struct{}{},
	}
}

func (t *testReporter) Report(
	addressPath interpreter.AddressPath,
	_ string,
) {
	t.migratedPaths[addressPath] = struct{}{}
}

func TestTypeValueMigration(t *testing.T) {
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
						nil,
						fooBarQualifiedIdentifier,
						common.NewTypeIDFromQualifiedName(
							nil,
							fooAddressLocation,
							fooBarQualifiedIdentifier,
						),
					),
				},
			),
			expectedType: nil,
		},
		"empty intersection": {
			storedType: interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{},
			),
			expectedType: nil,
		},
		"intersection_with_legacy_type": {
			storedType: &interpreter.IntersectionStaticType{
				Types:      []*interpreter.InterfaceStaticType{},
				LegacyType: publicAccountType,
			},
			expectedType: &interpreter.IntersectionStaticType{
				Types:      []*interpreter.InterfaceStaticType{},
				LegacyType: unauthorizedAccountReferenceType,
			},
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
		"interface": {
			storedType: interpreter.NewInterfaceStaticType(
				nil,
				nil,
				fooBarQualifiedIdentifier,
				common.NewTypeIDFromQualifiedName(
					nil,
					fooAddressLocation,
					fooBarQualifiedIdentifier,
				),
			),
			expectedType: nil,
		},
		"composite": {
			storedType: interpreter.NewCompositeStaticType(
				nil,
				nil,
				fooBarQualifiedIdentifier,
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

	ledger := runtime_utils.NewTestLedger(nil, nil)
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
			account,
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

	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		reporter,
		NewAccountTypeMigration(),
	)

	migration.Commit()

	// Check reported migrated paths
	for identifier, test := range testCases {
		addressPath := interpreter.AddressPath{
			Address: account,
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

	storageMap := storage.GetStorageMap(account, pathDomain.Identifier(), false)
	require.NotNil(t, storageMap)
	require.Greater(t, storageMap.Count(), uint64(0))

	iterator := storageMap.Iterator(inter)

	for key, value := iterator.Next(); key != nil; key, value = iterator.Next() {
		identifier := string(key.(interpreter.StringAtreeValue))

		t.Run(identifier, func(t *testing.T) {
			testCase, ok := testCases[identifier]
			require.True(t, ok)

			var expectedValue interpreter.Value
			if testCase.expectedType != nil {
				expectedValue = interpreter.NewTypeValue(nil, testCase.expectedType)

				// `IntersectionType.LegacyType` is not considered in the `IntersectionType.Equal` method.
				// Therefore, check for the legacy type equality manually.
				typeValue := value.(interpreter.TypeValue)
				if actualIntersectionType, ok := typeValue.Type.(*interpreter.IntersectionStaticType); ok {
					expectedIntersectionType := testCase.expectedType.(*interpreter.IntersectionStaticType)
					assert.True(t, actualIntersectionType.LegacyType.Equal(expectedIntersectionType.LegacyType))
				}
			} else {
				expectedValue = interpreter.NewTypeValue(nil, testCase.storedType)
			}

			utils.AssertValuesEqual(t, inter, expectedValue, value)
		})
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

func TestNestedTypeValueMigration(t *testing.T) {
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

	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		nil,
		NewAccountTypeMigration(),
	)

	migration.Commit()

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

func TestValuesWithStaticTypeMigration(t *testing.T) {

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
		nil,
		NewAccountTypeMigration(),
	)

	migration.Commit()

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
