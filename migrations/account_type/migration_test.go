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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/migrations"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testCase struct {
	storedType   interpreter.StaticType
	expectedType interpreter.StaticType
}

func TestMigration(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}
	pathDomain := common.PathDomainPublic

	const publicAccountType = interpreter.PrimitiveStaticTypePublicAccount
	const authAccountType = interpreter.PrimitiveStaticTypeAuthAccount
	const stringType = interpreter.PrimitiveStaticTypeString

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
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountCapabilities,
			expectedType: interpreter.PrimitiveStaticTypeAccount_Capabilities,
		},
		"public_account_capabilities": {
			storedType:   interpreter.PrimitiveStaticTypePublicAccountCapabilities,
			expectedType: interpreter.PrimitiveStaticTypeAccount_Capabilities,
		},
		"auth_account_account_capabilities": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountAccountCapabilities,
			expectedType: interpreter.PrimitiveStaticTypeAccount_AccountCapabilities,
		},
		"auth_account_storage_capabilities": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountStorageCapabilities,
			expectedType: interpreter.PrimitiveStaticTypeAccount_StorageCapabilities,
		},
		"auth_account_contracts": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountContracts,
			expectedType: interpreter.PrimitiveStaticTypeAccount_Contracts,
		},
		"public_account_contracts": {
			storedType:   interpreter.PrimitiveStaticTypePublicAccountContracts,
			expectedType: interpreter.PrimitiveStaticTypeAccount_Contracts,
		},
		"auth_account_keys": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountKeys,
			expectedType: interpreter.PrimitiveStaticTypeAccount_Keys,
		},
		"public_account_keys": {
			storedType:   interpreter.PrimitiveStaticTypePublicAccountKeys,
			expectedType: interpreter.PrimitiveStaticTypeAccount_Keys,
		},
		"auth_account_inbox": {
			storedType:   interpreter.PrimitiveStaticTypeAuthAccountInbox,
			expectedType: interpreter.PrimitiveStaticTypeAccount_Inbox,
		},
		"account_key": {
			storedType:   interpreter.PrimitiveStaticTypeAccountKey,
			expectedType: interpreter.AccountKeyStaticType,
		},
		"optional_account": {
			storedType:   interpreter.NewOptionalStaticType(nil, publicAccountType),
			expectedType: interpreter.NewOptionalStaticType(nil, unauthorizedAccountReferenceType),
		},
		"optional_string": {
			storedType: interpreter.NewOptionalStaticType(nil, stringType),
		},
		"constant_sized_account_array": {
			storedType:   interpreter.NewConstantSizedStaticType(nil, publicAccountType, 3),
			expectedType: interpreter.NewConstantSizedStaticType(nil, unauthorizedAccountReferenceType, 3),
		},
		"constant_sized_string_array": {
			storedType: interpreter.NewConstantSizedStaticType(nil, stringType, 3),
		},
		"variable_sized_account_array": {
			storedType:   interpreter.NewVariableSizedStaticType(nil, authAccountType),
			expectedType: interpreter.NewVariableSizedStaticType(nil, authAccountReferenceType),
		},
		"variable_sized_string_array": {
			storedType: interpreter.NewVariableSizedStaticType(nil, stringType),
		},
		"dictionary": {
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
		"string_dictionary": {
			storedType: interpreter.NewDictionaryStaticType(
				nil,
				stringType,
				stringType,
			),
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
		},
		"intersection": {
			storedType: interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticType(
						nil,
						nil,
						"Bar",
						common.NewTypeIDFromQualifiedName(
							nil,
							common.NewAddressLocation(nil, account, "Foo"),
							"Bar",
						),
					),
				},
			),
		},
		"empty intersection": {
			storedType: interpreter.NewIntersectionStaticType(
				nil,
				[]*interpreter.InterfaceStaticType{},
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
		"interface": {
			storedType: interpreter.NewInterfaceStaticType(
				nil,
				nil,
				"Bar",
				common.NewTypeIDFromQualifiedName(
					nil,
					common.NewAddressLocation(nil, account, "Foo"),
					"Bar",
				),
			),
		},
		"composite": {
			storedType: interpreter.NewCompositeStaticType(
				nil,
				nil,
				"Bar",
				common.NewTypeIDFromQualifiedName(
					nil,
					common.NewAddressLocation(nil, account, "Foo"),
					"Bar",
				),
			),
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

	rt := runtime_utils.NewTestInterpreterRuntime()

	runtimeInterface := &runtime_utils.TestRuntimeInterface{
		Storage: ledger,
	}

	migration, err := NewAccountTypeMigration(
		rt,
		runtime.Context{
			Interface: runtimeInterface,
		},
	)

	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		&migrations.AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		reporter,
	)

	migratedPathsInDomain := reporter.migratedPaths[account][pathDomain]

	for path, _ := range migratedPathsInDomain {
		require.Contains(t, testCases, path)
	}

	for path, test := range testCases {
		t.Run(path, func(t *testing.T) {

			test := test
			path := path

			t.Parallel()

			if test.expectedType == nil {
				require.NotContains(t, migratedPathsInDomain, path)
			} else {
				require.Contains(t, migratedPathsInDomain, path)

				actualValue := migratedPathsInDomain[path]
				actualTypeValue := actualValue.(interpreter.TypeValue)

				assert.True(
					t,
					test.expectedType.Equal(actualTypeValue.Type),
					fmt.Sprintf("expected `%s`, found `%s`", test.expectedType, actualTypeValue.Type),
				)
			}
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

var _ migrations.Reporter = &testReporter{}

type testReporter struct {
	migratedPaths map[common.Address]map[common.PathDomain]map[string]interpreter.Value
}

func newTestReporter() *testReporter {
	return &testReporter{
		migratedPaths: map[common.Address]map[common.PathDomain]map[string]interpreter.Value{},
	}
}

func (t *testReporter) Report(
	address common.Address,
	domain common.PathDomain,
	identifier string,
	value interpreter.Value,
) {
	migratedPathsInAddress, ok := t.migratedPaths[address]
	if !ok {
		migratedPathsInAddress = make(map[common.PathDomain]map[string]interpreter.Value)
		t.migratedPaths[address] = migratedPathsInAddress
	}

	migratedPathsInDomain, ok := migratedPathsInAddress[domain]
	if !ok {
		migratedPathsInDomain = make(map[string]interpreter.Value)
		migratedPathsInAddress[domain] = migratedPathsInDomain
	}

	migratedPathsInDomain[identifier] = value
}

func (t *testReporter) ReportError(err error) {
	panic("implement me")
}
