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

package migrations

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
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

var _ Reporter = &testReporter{}

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

// testStringMigration

type testStringMigration struct{}

var _ ValueMigration = testStringMigration{}

func (testStringMigration) Name() string {
	return "testStringMigration"
}

func (testStringMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	if value, ok := value.(*interpreter.StringValue); ok {
		return interpreter.NewUnmeteredStringValue(fmt.Sprintf("updated_%s", value.Str)), nil
	}

	return nil, nil
}

func (testStringMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testStringMigration) Domains() map[string]struct{} {
	return nil
}

// testInt8Migration

type testInt8Migration struct {
	mustError bool
}

var _ ValueMigration = testInt8Migration{}

func (testInt8Migration) Name() string {
	return "testInt8Migration"
}

func (m testInt8Migration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	int8Value, ok := value.(interpreter.Int8Value)
	if !ok {
		return nil, nil
	}

	if m.mustError {
		return nil, errors.New("error occurred while migrating int8")
	}

	return interpreter.NewUnmeteredInt8Value(int8(int8Value) + 10), nil
}

func (testInt8Migration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testInt8Migration) Domains() map[string]struct{} {
	return nil
}

// testCapMigration

type testCapMigration struct{}

var _ ValueMigration = testCapMigration{}

func (testCapMigration) Name() string {
	return "testCapMigration"
}

func (testCapMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	if value, ok := value.(*interpreter.IDCapabilityValue); ok {
		return interpreter.NewCapabilityValue(
			nil,
			value.ID+10,
			value.Address(),
			value.BorrowType,
		), nil
	}

	return nil, nil
}

func (testCapMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testCapMigration) Domains() map[string]struct{} {
	return nil
}

// testCapConMigration

type testCapConMigration struct{}

var _ ValueMigration = testCapConMigration{}

func (testCapConMigration) Name() string {
	return "testCapConMigration"
}

func (testCapConMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	switch value := value.(type) {
	case *interpreter.StorageCapabilityControllerValue:
		return interpreter.NewStorageCapabilityControllerValue(
			nil,
			value.BorrowType,
			value.CapabilityID+10,
			value.TargetPath,
		), nil

	case *interpreter.AccountCapabilityControllerValue:
		return interpreter.NewAccountCapabilityControllerValue(
			nil,
			value.BorrowType,
			value.CapabilityID+10,
		), nil

	case interpreter.UInt64Value:
		return value + 10, nil

	case interpreter.NilValue:
		return value, nil
	}

	return nil, nil
}

func (testCapConMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testCapConMigration) Domains() map[string]struct{} {
	return nil
}

func TestMultipleMigrations(t *testing.T) {
	t.Parallel()

	account := common.Address{0x42}

	type testCase struct {
		name          string
		migration     string
		storedValue   interpreter.Value
		expectedValue interpreter.Value
		key           string
	}

	ledger := NewTestLedger(nil, nil)
	storage := runtime.NewStorage(ledger, nil)
	locationRange := interpreter.EmptyLocationRange

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

	testCases := []testCase{
		{
			name:          "string_value",
			key:           common.PathDomainStorage.Identifier(),
			migration:     "testStringMigration",
			storedValue:   interpreter.NewUnmeteredStringValue("hello"),
			expectedValue: interpreter.NewUnmeteredStringValue("updated_hello"),
		},
		{
			name:          "int8_value",
			key:           common.PathDomainStorage.Identifier(),
			migration:     "testInt8Migration",
			storedValue:   interpreter.NewUnmeteredInt8Value(5),
			expectedValue: interpreter.NewUnmeteredInt8Value(15),
		},
		{
			name:          "int16_value",
			key:           common.PathDomainStorage.Identifier(),
			migration:     "", // should not be migrated
			storedValue:   interpreter.NewUnmeteredInt16Value(5),
			expectedValue: interpreter.NewUnmeteredInt16Value(5),
		},
		{
			name:      "storage_cap_value",
			key:       common.PathDomainStorage.Identifier(),
			migration: "testCapMigration",
			storedValue: interpreter.NewCapabilityValue(
				nil,
				5,
				interpreter.AddressValue(common.Address{0x1}),
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					interpreter.PrimitiveStaticTypeString,
				),
			),
			expectedValue: interpreter.NewCapabilityValue(
				nil,
				15,
				interpreter.AddressValue(common.Address{0x1}),
				interpreter.NewReferenceStaticType(
					nil,
					interpreter.UnauthorizedAccess,
					interpreter.PrimitiveStaticTypeString,
				),
			),
		},
		{
			name:      "inbox_cap_value",
			key:       stdlib.InboxStorageDomain,
			migration: "testCapMigration",
			storedValue: interpreter.NewPublishedValue(
				nil,
				interpreter.AddressValue(common.Address{0x2}),
				interpreter.NewCapabilityValue(
					nil,
					5,
					interpreter.AddressValue(common.Address{0x1}),
					interpreter.NewReferenceStaticType(
						nil,
						interpreter.UnauthorizedAccess,
						interpreter.PrimitiveStaticTypeString,
					),
				),
			),
			expectedValue: interpreter.NewPublishedValue(
				nil,
				interpreter.AddressValue(common.Address{0x2}),
				interpreter.NewCapabilityValue(
					nil,
					15,
					interpreter.AddressValue(common.Address{0x1}),
					interpreter.NewReferenceStaticType(
						nil,
						interpreter.UnauthorizedAccess,
						interpreter.PrimitiveStaticTypeString,
					),
				),
			),
		},
	}

	variableSizedAnyStructStaticType :=
		interpreter.NewVariableSizedStaticType(nil, interpreter.PrimitiveStaticTypeAnyStruct)

	dictionaryAnyStructStaticType :=
		interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeAnyStruct,
			interpreter.PrimitiveStaticTypeAnyStruct,
		)

	for _, test := range testCases {

		if test.key != common.PathDomainStorage.Identifier() {
			continue
		}

		testCases = append(testCases, testCase{
			name:      "array_" + test.name,
			key:       test.key,
			migration: test.migration,
			storedValue: interpreter.NewArrayValue(
				inter,
				emptyLocationRange,
				variableSizedAnyStructStaticType,
				common.ZeroAddress,
				test.storedValue,
			),
			expectedValue: interpreter.NewArrayValue(
				inter,
				emptyLocationRange,
				variableSizedAnyStructStaticType,
				common.ZeroAddress,
				test.expectedValue,
			),
		})

		if _, ok := test.storedValue.(interpreter.HashableValue); ok {

			testCases = append(testCases, testCase{
				name:      "dict_key_" + test.name,
				key:       test.key,
				migration: test.migration,
				storedValue: interpreter.NewDictionaryValue(
					inter,
					emptyLocationRange,
					dictionaryAnyStructStaticType,
					test.storedValue,
					interpreter.TrueValue,
				),

				expectedValue: interpreter.NewDictionaryValue(
					inter,
					emptyLocationRange,
					dictionaryAnyStructStaticType,
					test.expectedValue,
					interpreter.TrueValue,
				),
			})
		}

		testCases = append(testCases, testCase{
			name:      "dict_value_" + test.name,
			key:       test.key,
			migration: test.migration,
			storedValue: interpreter.NewDictionaryValue(
				inter,
				emptyLocationRange,
				dictionaryAnyStructStaticType,
				interpreter.TrueValue,
				test.storedValue,
			),
			expectedValue: interpreter.NewDictionaryValue(
				inter,
				emptyLocationRange,
				dictionaryAnyStructStaticType,
				interpreter.TrueValue,
				test.expectedValue,
			),
		})

		testCases = append(testCases, testCase{
			name:          "some_" + test.name,
			key:           test.key,
			migration:     test.migration,
			storedValue:   interpreter.NewSomeValueNonCopying(nil, test.storedValue),
			expectedValue: interpreter.NewSomeValueNonCopying(nil, test.expectedValue),
		})

		if _, ok := test.storedValue.(*interpreter.IDCapabilityValue); ok {

			testCases = append(testCases, testCase{
				name:      "published_" + test.name,
				key:       test.key,
				migration: test.migration,
				storedValue: interpreter.NewPublishedValue(
					nil,
					interpreter.AddressValue(common.ZeroAddress),
					test.storedValue.(*interpreter.IDCapabilityValue),
				),
				expectedValue: interpreter.NewPublishedValue(
					nil,
					interpreter.AddressValue(common.ZeroAddress),
					test.expectedValue.(*interpreter.IDCapabilityValue),
				),
			})
		}

		testCases = append(testCases, testCase{
			name:      "struct_" + test.name,
			key:       test.key,
			migration: test.migration,
			storedValue: interpreter.NewCompositeValue(
				inter,
				emptyLocationRange,
				utils.TestLocation,
				"S",
				common.CompositeKindStructure,
				[]interpreter.CompositeField{
					{
						Name:  "test",
						Value: test.storedValue,
					},
				},
				common.ZeroAddress,
			),
			expectedValue: interpreter.NewCompositeValue(
				inter,
				emptyLocationRange,
				utils.TestLocation,
				"S",
				common.CompositeKindStructure,
				[]interpreter.CompositeField{
					{
						Name:  "test",
						Value: test.expectedValue,
					},
				},
				common.ZeroAddress,
			),
		})
	}

	// Store values

	for _, testCase := range testCases {
		transferredValue := testCase.storedValue.Transfer(
			inter,
			locationRange,
			atree.Address(account),
			false,
			nil,
			nil,
			true, // storedValue is standalone
		)

		inter.WriteStored(
			account,
			testCase.key,
			interpreter.StringStorageMapKey(testCase.name),
			transferredValue,
		)
	}

	err = storage.Commit(inter, true)
	require.NoError(t, err)

	// Migrate

	reporter := newTestReporter()

	migration, err := NewStorageMigration(
		inter,
		storage,
		"test",
		account,
	)
	require.NoError(t, err)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testStringMigration{},
			testInt8Migration{},
			testCapMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	for _, testCase := range testCases {

		t.Run(testCase.name, func(t *testing.T) {

			storageMap := storage.GetStorageMap(account, testCase.key, false)
			require.NotNil(t, storageMap)

			readValue := storageMap.ReadValue(nil, interpreter.StringStorageMapKey(testCase.name))

			utils.AssertValuesEqual(t,
				inter,
				testCase.expectedValue,
				readValue,
			)
		})
	}

	// Check the reporter
	expectedMigrations := map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}][]string{}

	for _, testCase := range testCases {

		if testCase.migration == "" {
			continue
		}

		key := struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}{
			StorageKey: interpreter.StorageKey{
				Address: account,
				Key:     testCase.key,
			},
			StorageMapKey: interpreter.StringStorageMapKey(testCase.name),
		}

		expectedMigrations[key] = append(
			expectedMigrations[key],
			testCase.migration,
		)
	}

	require.Equal(t,
		expectedMigrations,
		reporter.migrated,
	)
}

func TestMigrationError(t *testing.T) {
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
			AtreeValueValidationEnabled:   true,
			AtreeStorageValidationEnabled: true,
		},
	)
	require.NoError(t, err)

	testCases := map[string]testCase{
		"string_value": {
			storedValue:   interpreter.NewUnmeteredStringValue("hello"),
			expectedValue: interpreter.NewUnmeteredStringValue("updated_hello"),
		},
		// Since Int8 migration expected to produce error,
		// int8 value should not have been migrated.
		"int8_value": {
			storedValue:   interpreter.NewUnmeteredInt8Value(5),
			expectedValue: interpreter.NewUnmeteredInt8Value(5),
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
			true, // storedValue is standalone
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

	migration, err := NewStorageMigration(inter, storage, "test", account)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testStringMigration{},

			// Int8 migration should produce errors
			testInt8Migration{
				mustError: true,
			},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	require.Equal(t,
		[]error{
			StorageMigrationError{
				StorageKey: interpreter.StorageKey{
					Address: account,
					Key:     pathDomain.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("int8_value"),
				Migration:     "testInt8Migration",
				Err:           errors.New("error occurred while migrating int8"),
			},
		},
		reporter.errors,
	)

	err = storage.CheckHealth()
	require.NoError(t, err)

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
			utils.AssertValuesEqual(t, inter, testCase.expectedValue, value)
		})
	}

	// Check the reporter.
	// Since Int8 migration produces an error, only the string value must have been migrated.
	require.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}][]string{
			{
				StorageKey: interpreter.StorageKey{
					Address: account,
					Key:     pathDomain.Identifier(),
				},
				StorageMapKey: interpreter.StringStorageMapKey("string_value"),
			}: {
				"testStringMigration",
			},
		},
		reporter.migrated,
	)
}

func TestCapConMigration(t *testing.T) {

	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	rt := NewTestInterpreterRuntime()

	var events []cadence.Event

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	// Prepare

	setupTx := `
	  transaction {
          prepare(signer: auth(Capabilities) &Account) {
              signer.capabilities.storage.issue<&AnyStruct>(/storage/foo)
              signer.capabilities.account.issue<&Account>()
          }
      }
    `

	nextTransactionLocation := NewTransactionLocationGenerator()

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: []byte(setupTx),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	capConStorageMap := storage.GetStorageMap(
		testAddress,
		stdlib.CapabilityControllerStorageDomain,
		false,
	)

	assert.Equal(t, uint64(2), capConStorageMap.Count())

	controller1 := capConStorageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(1))
	require.IsType(t, &interpreter.StorageCapabilityControllerValue{}, controller1)
	assert.Equal(t,
		interpreter.UInt64Value(1),
		controller1.(*interpreter.StorageCapabilityControllerValue).CapabilityID,
	)

	controller2 := capConStorageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(2))
	require.IsType(t, &interpreter.AccountCapabilityControllerValue{}, controller2)
	assert.Equal(t,
		interpreter.UInt64Value(2),
		controller2.(*interpreter.AccountCapabilityControllerValue).CapabilityID,
	)

	pathCapStorageMap := storage.GetStorageMap(
		testAddress,
		stdlib.PathCapabilityStorageDomain,
		false,
	)

	pathCapSet := pathCapStorageMap.ReadValue(nil, interpreter.StringStorageMapKey("foo"))
	require.IsType(t, &interpreter.DictionaryValue{}, pathCapSet)
	capSetDict := pathCapSet.(*interpreter.DictionaryValue)
	assert.Equal(t, 1, capSetDict.Count())
	assert.True(t, bool(capSetDict.ContainsKey(inter, emptyLocationRange, interpreter.UInt64Value(1))))

	accountCapStorageMap := storage.GetStorageMap(
		testAddress,
		stdlib.AccountCapabilityStorageDomain,
		false,
	)

	accountCapEntry := accountCapStorageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(2))
	assert.NotNil(t, accountCapEntry)

	// Migrate

	migration, err := NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testCapConMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Empty(t, reporter.errors)
	assert.Equal(t,
		map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}][]string{
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(1),
			}: {"testCapConMigration"},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     stdlib.CapabilityControllerStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(2),
			}: {"testCapConMigration"},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     stdlib.PathCapabilityStorageDomain,
				},
				StorageMapKey: interpreter.StringStorageMapKey("foo"),
			}: {"testCapConMigration", "testCapConMigration"},
			{
				StorageKey: interpreter.StorageKey{
					Address: testAddress,
					Key:     stdlib.AccountCapabilityStorageDomain,
				},
				StorageMapKey: interpreter.Uint64StorageMapKey(2),
			}: {"testCapConMigration"},
		},
		reporter.migrated,
	)

	err = storage.CheckHealth()
	require.NoError(t, err)

	capConStorageMap = storage.GetStorageMap(
		testAddress,
		stdlib.CapabilityControllerStorageDomain,
		false,
	)

	assert.Equal(t, uint64(2), capConStorageMap.Count())

	controller1 = capConStorageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(1))
	require.IsType(t, &interpreter.StorageCapabilityControllerValue{}, controller1)
	assert.Equal(t,
		interpreter.UInt64Value(11),
		controller1.(*interpreter.StorageCapabilityControllerValue).CapabilityID,
	)

	controller2 = capConStorageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(2))
	require.IsType(t, &interpreter.AccountCapabilityControllerValue{}, controller2)
	assert.Equal(t,
		interpreter.UInt64Value(12),
		controller2.(*interpreter.AccountCapabilityControllerValue).CapabilityID,
	)

	pathCapStorageMap = storage.GetStorageMap(
		testAddress,
		stdlib.PathCapabilityStorageDomain,
		false,
	)

	pathCapSet = pathCapStorageMap.ReadValue(nil, interpreter.StringStorageMapKey("foo"))
	require.IsType(t, &interpreter.DictionaryValue{}, pathCapSet)
	capSetDict = pathCapSet.(*interpreter.DictionaryValue)
	assert.Equal(t, 1, capSetDict.Count())
	assert.True(t, bool(capSetDict.ContainsKey(inter, emptyLocationRange, interpreter.UInt64Value(11))))

	accountCapStorageMap = storage.GetStorageMap(
		testAddress,
		stdlib.AccountCapabilityStorageDomain,
		false,
	)

	accountCapEntry = accountCapStorageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(2))
	assert.NotNil(t, accountCapEntry)

	require.Equal(t,
		[]string{
			`flow.StorageCapabilityControllerIssued(id: 1, address: 0x0000000000000001, type: Type<&AnyStruct>(), path: /storage/foo)`,
			`flow.AccountCapabilityControllerIssued(id: 2, address: 0x0000000000000001, type: Type<&Account>())`,
		},
		eventStrings(events),
	)
}

func eventStrings(events []cadence.Event) []string {
	strings := make([]string, 0, len(events))
	for _, event := range events {
		strings = append(strings, event.String())
	}
	return strings
}

func TestContractMigration(t *testing.T) {

	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	rt := NewTestInterpreterRuntime()

	accountCodes := map[common.Location][]byte{}

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
		},
		OnEmitEvent: func(event cadence.Event) error {
			return nil
		},
		OnGetCode: func(location common.Location) (bytes []byte, err error) {
			return accountCodes[location], nil
		},
		OnResolveLocation: NewSingleIdentifierLocationResolver(t),
		OnGetAccountContractCode: func(location common.AddressLocation) (code []byte, err error) {
			code = accountCodes[location]
			return code, nil
		},
		OnUpdateAccountContractCode: func(location common.AddressLocation, code []byte) error {
			accountCodes[location] = code
			return nil
		},
	}

	const testContract = `
      access(all)
      contract Test {

          access(all)
          let foo: String

          init() {
              self.foo = "bar"
          }
      }
    `

	// Prepare

	nextTransactionLocation := NewTransactionLocationGenerator()
	nextScriptLocation := NewScriptLocationGenerator()

	err := rt.ExecuteTransaction(
		runtime.Script{
			Source: utils.DeploymentTransaction("Test", []byte(testContract)),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextTransactionLocation(),
		},
	)
	require.NoError(t, err)

	storage, inter, err := rt.Storage(runtime.Context{
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	// Migrate

	migration, err := NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testStringMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert
	assert.Len(t, reporter.migrated, 1)

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	value, err := rt.ExecuteScript(
		runtime.Script{
			Source: []byte(`
              import Test from 0x1

              access(all)
              fun main(): String {
                  return Test.foo
              }
            `),
		},
		runtime.Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.NoError(t, err)

	require.Equal(t,
		cadence.String("updated_bar"),
		value,
	)
}

// testCompositeValueMigration

type testCompositeValueMigration struct {
}

var _ ValueMigration = testCompositeValueMigration{}

func (testCompositeValueMigration) Name() string {
	return "testCompositeValueMigration"
}

func (m testCompositeValueMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	inter *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	compositeValue, ok := value.(*interpreter.CompositeValue)
	if !ok {
		return nil, nil
	}

	return interpreter.NewCompositeValue(
		inter,
		emptyLocationRange,
		utils.TestLocation,
		"S2",
		common.CompositeKindStructure,
		nil,
		common.Address(compositeValue.StorageAddress()),
	), nil
}

func (testCompositeValueMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testCompositeValueMigration) Domains() map[string]struct{} {
	return nil
}

func TestEmptyIntersectionTypeMigration(t *testing.T) {

	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	rt := NewTestInterpreterRuntime()

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
	}

	// Prepare

	storage, inter, err := rt.Storage(runtime.Context{
		Location:  utils.TestLocation,
		Interface: runtimeInterface,
	})
	require.NoError(t, err)

	storageMap := storage.GetStorageMap(
		testAddress,
		common.PathDomainStorage.Identifier(),
		true,
	)

	elaboration := sema.NewElaboration(nil)

	const s1QualifiedIdentifier = "S1"
	const s2QualifiedIdentifier = "S2"

	elaboration.SetCompositeType(
		utils.TestLocation.TypeID(nil, s1QualifiedIdentifier),
		&sema.CompositeType{
			Location:   utils.TestLocation,
			Members:    &sema.StringMemberOrderedMap{},
			Identifier: s1QualifiedIdentifier,
			Kind:       common.CompositeKindStructure,
		},
	)

	elaboration.SetCompositeType(
		utils.TestLocation.TypeID(nil, s2QualifiedIdentifier),
		&sema.CompositeType{
			Location:   utils.TestLocation,
			Members:    &sema.StringMemberOrderedMap{},
			Identifier: s2QualifiedIdentifier,
			Kind:       common.CompositeKindStructure,
		},
	)

	compositeValue := interpreter.NewCompositeValue(
		inter,
		emptyLocationRange,
		utils.TestLocation,
		s1QualifiedIdentifier,
		common.CompositeKindStructure,
		nil,
		testAddress,
	)

	inter.Program = &interpreter.Program{
		Elaboration: elaboration,
	}

	// NOTE: create an empty intersection type with a legacy type: AnyStruct{}
	emptyIntersectionType := interpreter.NewIntersectionStaticType(
		nil,
		nil,
	)
	emptyIntersectionType.LegacyType = interpreter.PrimitiveStaticTypeAnyStruct

	storageMapKey := interpreter.StringStorageMapKey("test")

	dictionaryKey := interpreter.NewUnmeteredStringValue("foo")

	dictionaryValue := interpreter.NewDictionaryValueWithAddress(
		inter,
		emptyLocationRange,
		interpreter.NewDictionaryStaticType(
			nil,
			interpreter.PrimitiveStaticTypeString,
			emptyIntersectionType,
		),
		testAddress,
	)

	dictionaryValue.InsertWithoutTransfer(
		inter,
		emptyLocationRange,
		dictionaryKey,
		compositeValue,
	)

	storageMap.WriteValue(
		inter,
		storageMapKey,
		dictionaryValue,
	)

	// Migrate

	migration, err := NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testCompositeValueMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Len(t, reporter.errors, 0)
	assert.Len(t, reporter.migrated, 1)

	err = storage.CheckHealth()
	require.NoError(t, err)

	storageMap = storage.GetStorageMap(
		testAddress,
		common.PathDomainStorage.Identifier(),
		false,
	)

	assert.Equal(t, uint64(1), storageMap.Count())

	migratedValue := storageMap.ReadValue(nil, storageMapKey)

	require.IsType(t, &interpreter.DictionaryValue{}, migratedValue)
	migratedDictionaryValue := migratedValue.(*interpreter.DictionaryValue)

	migratedChildValue, ok := migratedDictionaryValue.Get(inter, emptyLocationRange, dictionaryKey)
	require.True(t, ok)

	require.IsType(t, &interpreter.CompositeValue{}, migratedChildValue)
	migratedCompositeValue := migratedChildValue.(*interpreter.CompositeValue)

	require.Equal(
		t,
		s2QualifiedIdentifier,
		migratedCompositeValue.QualifiedIdentifier,
	)
}

// testContainerMigration

type testContainerMigration struct{}

var _ ValueMigration = testContainerMigration{}

func (testContainerMigration) Name() string {
	return "testContainerMigration"
}

func (testContainerMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	inter *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	switch value := value.(type) {
	case *interpreter.DictionaryValue:

		newType := interpreter.NewDictionaryStaticType(nil,
			interpreter.PrimitiveStaticTypeAnyStruct,
			interpreter.PrimitiveStaticTypeAnyStruct,
		)

		value.SetType(newType)

	case *interpreter.ArrayValue:

		newType := interpreter.NewVariableSizedStaticType(nil,
			interpreter.PrimitiveStaticTypeAnyStruct,
		)

		value.SetType(newType)

	case *interpreter.CompositeValue:
		if value.QualifiedIdentifier == "Inner" {
			return interpreter.NewCompositeValue(
				inter,
				emptyLocationRange,
				utils.TestLocation,
				"Inner2",
				common.CompositeKindStructure,
				nil,
				value.GetOwner(),
			), nil
		}
	}

	return nil, nil
}

func (testContainerMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testContainerMigration) Domains() map[string]struct{} {
	return nil
}

func TestMigratingNestedContainers(t *testing.T) {

	t.Parallel()

	var testAddress = common.Address{0x42}

	migrate := func(
		t *testing.T,
		valueMigration ValueMigration,
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
			true, // standalone values doesn't have a parent container.
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

		migration, err := NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				valueMigration,
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

	t.Run("nested dictionary, value migrated", func(t *testing.T) {
		t.Parallel()

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

		// {"key1": {"key2": 1234}}: {String: {String: Int}}

		storedValue := interpreter.NewDictionaryValue(
			inter,
			locationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeString,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt,
				),
			),
			interpreter.NewUnmeteredStringValue("key1"),
			interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt,
				),
				interpreter.NewUnmeteredStringValue("key2"),
				interpreter.NewUnmeteredIntValueFromInt64(1234),
			),
		)

		actual := migrate(t,
			testContainerMigration{},
			storage,
			inter,
			storedValue,
		)

		// {AnyStruct: AnyStruct} with {AnyStruct: AnyStruct}

		expected := interpreter.NewDictionaryValue(
			inter,
			locationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeAnyStruct,
				interpreter.PrimitiveStaticTypeAnyStruct,
			),
			interpreter.NewUnmeteredStringValue("key1"),
			interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeAnyStruct,
					interpreter.PrimitiveStaticTypeAnyStruct,
				),
				interpreter.NewUnmeteredStringValue("key2"),
				interpreter.NewUnmeteredIntValueFromInt64(1234),
			),
		)

		utils.AssertValuesEqual(t, inter, expected, actual)
	})

	t.Run("nested dictionary, key migrated", func(t *testing.T) {
		t.Parallel()

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

		// {"key1": {"key2": 1234}}: {String: {String: Int}}

		storedValue := interpreter.NewDictionaryValue(
			inter,
			locationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeString,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt,
				),
			),
			interpreter.NewUnmeteredStringValue("key1"),
			interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt,
				),
				interpreter.NewUnmeteredStringValue("key2"),
				interpreter.NewUnmeteredIntValueFromInt64(1234),
			),
		)

		actual := migrate(t,
			testStringMigration{},
			storage,
			inter,
			storedValue,
		)

		// {"updated_key1": {"updated_key2": 1234}}: {String: {String: Int}}

		expected := interpreter.NewDictionaryValue(
			inter,
			locationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeString,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt,
				),
			),
			interpreter.NewUnmeteredStringValue("updated_key1"),
			interpreter.NewDictionaryValue(
				inter,
				locationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.PrimitiveStaticTypeInt,
				),
				interpreter.NewUnmeteredStringValue("updated_key2"),
				interpreter.NewUnmeteredIntValueFromInt64(1234),
			),
		)

		utils.AssertValuesEqual(t, inter, expected, actual)
	})

	t.Run("nested arrays", func(t *testing.T) {
		t.Parallel()

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

		// [["abc"]]: [[String]]

		storedValue := interpreter.NewArrayValue(
			inter,
			locationRange,
			interpreter.NewVariableSizedStaticType(
				nil,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
				),
			),
			common.ZeroAddress,
			interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
				),
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("abc"),
			),
		)

		actual := migrate(t,
			testContainerMigration{},
			storage,
			inter,
			storedValue,
		)

		// [AnyStruct] with [AnyStruct]

		expected := interpreter.NewArrayValue(
			inter,
			locationRange,
			interpreter.NewVariableSizedStaticType(
				nil,
				interpreter.PrimitiveStaticTypeAnyStruct,
			),
			common.ZeroAddress,
			interpreter.NewArrayValue(
				inter,
				locationRange,
				interpreter.NewVariableSizedStaticType(
					nil,
					interpreter.PrimitiveStaticTypeAnyStruct,
				),
				common.ZeroAddress,
				interpreter.NewUnmeteredStringValue("abc"),
			),
		)

		utils.AssertValuesEqual(t, inter, expected, actual)
	})

	t.Run("nested composite", func(t *testing.T) {
		t.Parallel()

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

		// Outer(Inner())

		storedValue := interpreter.NewCompositeValue(
			inter,
			locationRange,
			utils.TestLocation,
			"Outer",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name: "inner",
					Value: interpreter.NewCompositeValue(
						inter,
						locationRange,
						utils.TestLocation,
						"Inner",
						common.CompositeKindStructure,
						nil,
						common.ZeroAddress,
					),
				},
			},
			common.ZeroAddress,
		)

		actual := migrate(t,
			testContainerMigration{},
			storage,
			inter,
			storedValue,
		)

		// Outer(Inner2())

		expected := interpreter.NewCompositeValue(
			inter,
			locationRange,
			utils.TestLocation,
			"Outer",
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name: "inner",
					Value: interpreter.NewCompositeValue(
						inter,
						locationRange,
						utils.TestLocation,
						"Inner2",
						common.CompositeKindStructure,
						nil,
						common.ZeroAddress,
					),
				},
			},
			common.ZeroAddress,
		)

		utils.AssertValuesEqual(t, inter, expected, actual)
	})
}

// testPanicMigration

type testPanicMigration struct{}

var _ ValueMigration = testInt8Migration{}

func (testPanicMigration) Name() string {
	return "testPanicMigration"
}

func (m testPanicMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	_ interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	// NOTE: out-of-bounds access, panic
	_ = []int{}[0]

	return nil, nil
}

func (testPanicMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testPanicMigration) Domains() map[string]struct{} {
	return nil
}

func TestMigrationPanic(t *testing.T) {
	t.Parallel()

	testAddress := common.Address{0x42}

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

	// Store value

	storagePathDomain := common.PathDomainStorage.Identifier()

	storageMapKey := interpreter.StringStorageMapKey("test_value")

	inter.WriteStored(
		testAddress,
		storagePathDomain,
		storageMapKey,
		interpreter.NewUnmeteredUInt8Value(42),
	)

	err = storage.Commit(inter, true)
	require.NoError(t, err)

	// Migrate

	migration, err := NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration = migration.WithErrorStacktrace(true)

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testPanicMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	require.Len(t, reporter.errors, 1)

	var migrationError StorageMigrationError
	require.ErrorAs(t, reporter.errors[0], &migrationError)

	assert.Equal(
		t,
		interpreter.StorageKey{
			Address: testAddress,
			Key:     storagePathDomain,
		},
		migrationError.StorageKey,
	)
	assert.Equal(
		t,
		storageMapKey,
		migrationError.StorageMapKey,
	)
	assert.Equal(
		t,
		"testPanicMigration",
		migrationError.Migration,
	)
	assert.ErrorContains(
		t,
		migrationError,
		"index out of range",
	)
	assert.NotEmpty(t, migrationError.Stack)
}

type testSkipMigration struct {
	migrationCalls []interpreter.Value
	canSkip        func(valueType interpreter.StaticType) bool
}

var _ ValueMigration = &testSkipMigration{}

func (*testSkipMigration) Name() string {
	return "testSkipMigration"
}

func (m *testSkipMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	m.migrationCalls = append(m.migrationCalls, value)

	// Do not actually migrate anything

	return nil, nil
}

func (m *testSkipMigration) CanSkip(valueType interpreter.StaticType) bool {
	return m.canSkip(valueType)
}

func (*testSkipMigration) Domains() map[string]struct{} {
	return nil
}

func TestSkip(t *testing.T) {
	t.Parallel()

	testAddress := common.Address{0x42}

	migrate := func(
		t *testing.T,
		valueFactory func(interpreter *interpreter.Interpreter) interpreter.Value,
		canSkip func(valueType interpreter.StaticType) bool,
	) (
		migrationCalls []interpreter.Value,
		inter *interpreter.Interpreter,
	) {

		ledger := NewTestLedger(nil, nil)

		storage := runtime.NewStorage(ledger, nil)

		var err error
		inter, err = interpreter.NewInterpreter(
			nil,
			utils.TestLocation,
			&interpreter.Config{
				Storage:                       storage,
				AtreeValueValidationEnabled:   true,
				AtreeStorageValidationEnabled: false,
			},
		)
		require.NoError(t, err)

		// Store value

		storagePathDomain := common.PathDomainStorage.Identifier()
		storageMapKey := interpreter.StringStorageMapKey("test_value")

		value := valueFactory(inter)

		inter.WriteStored(
			testAddress,
			storagePathDomain,
			storageMapKey,
			value,
		)

		// Migrate

		migration, err := NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		valueMigration := &testSkipMigration{
			canSkip: canSkip,
		}

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				valueMigration,
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		require.Empty(t, reporter.errors)

		err = storage.CheckHealth()
		require.NoError(t, err)

		return valueMigration.migrationCalls, inter
	}

	t.Run("skip non-string values", func(t *testing.T) {
		t.Parallel()

		var canSkip func(valueType interpreter.StaticType) bool
		canSkip = func(valueType interpreter.StaticType) bool {
			switch ty := valueType.(type) {
			case *interpreter.DictionaryStaticType:
				return canSkip(ty.KeyType) &&
					canSkip(ty.ValueType)

			case interpreter.ArrayStaticType:
				return canSkip(ty.ElementType())

			case *interpreter.OptionalStaticType:
				return canSkip(ty.Type)

			case *interpreter.CapabilityStaticType:
				return true

			case interpreter.PrimitiveStaticType:

				switch ty {
				case interpreter.PrimitiveStaticTypeBool,
					interpreter.PrimitiveStaticTypeVoid,
					interpreter.PrimitiveStaticTypeAddress,
					interpreter.PrimitiveStaticTypeMetaType,
					interpreter.PrimitiveStaticTypeBlock,
					interpreter.PrimitiveStaticTypeCharacter,
					interpreter.PrimitiveStaticTypeCapability:

					return true
				}

				if !ty.IsDeprecated() { //nolint:staticcheck
					semaType := ty.SemaType()

					if sema.IsSubType(semaType, sema.NumberType) ||
						sema.IsSubType(semaType, sema.PathType) {

						return true
					}
				}
			}

			return false
		}

		t.Run("[{Int: Bool}]", func(t *testing.T) {

			t.Parallel()

			migrationCalls, _ := migrate(
				t,
				func(inter *interpreter.Interpreter) interpreter.Value {

					dictionaryStaticType := interpreter.NewDictionaryStaticType(
						nil,
						interpreter.PrimitiveStaticTypeInt,
						interpreter.PrimitiveStaticTypeBool,
					)

					array := interpreter.NewArrayValue(
						inter,
						interpreter.EmptyLocationRange,
						interpreter.NewVariableSizedStaticType(
							nil,
							dictionaryStaticType,
						),
						testAddress,
					)

					array.InsertWithoutTransfer(
						inter,
						interpreter.EmptyLocationRange,
						0,
						interpreter.NewDictionaryValueWithAddress(
							inter,
							interpreter.EmptyLocationRange,
							dictionaryStaticType,
							testAddress,
							interpreter.NewUnmeteredIntValueFromInt64(42),
							interpreter.BoolValue(true),
						),
					)

					return array
				},
				canSkip,
			)

			require.Empty(t, migrationCalls)
		})

		t.Run("[{Int: AnyStruct}]", func(t *testing.T) {

			t.Parallel()

			dictionaryStaticType := interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeInt,
				interpreter.PrimitiveStaticTypeAnyStruct,
			)

			newStringValue := func() *interpreter.StringValue {
				return interpreter.NewUnmeteredStringValue("abc")
			}

			newDictionaryValue := func(inter *interpreter.Interpreter) *interpreter.DictionaryValue {
				return interpreter.NewDictionaryValueWithAddress(
					inter,
					interpreter.EmptyLocationRange,
					dictionaryStaticType,
					testAddress,
					interpreter.NewUnmeteredIntValueFromInt64(42),
					newStringValue(),
				)
			}

			newArrayValue := func(inter *interpreter.Interpreter) *interpreter.ArrayValue {
				array := interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(
						nil,
						dictionaryStaticType,
					),
					testAddress,
				)

				array.InsertWithoutTransfer(
					inter,
					interpreter.EmptyLocationRange,
					0,
					newDictionaryValue(inter),
				)

				return array
			}

			migrationCalls, inter := migrate(
				t,
				func(inter *interpreter.Interpreter) interpreter.Value {
					return newArrayValue(inter)
				},
				canSkip,
			)

			// NOTE: the integer value, the key of the dictionary, is skipped!
			require.Len(t, migrationCalls, 3)

			// first

			first := migrationCalls[0]
			require.IsType(t, &interpreter.StringValue{}, first)

			assert.True(t,
				first.(*interpreter.StringValue).
					Equal(inter, emptyLocationRange, newStringValue()),
			)

			// second

			second := migrationCalls[1]
			require.IsType(t, &interpreter.DictionaryValue{}, second)

			assert.True(t,
				second.(*interpreter.DictionaryValue).
					Equal(inter, emptyLocationRange, newDictionaryValue(inter)),
			)

			// third

			third := migrationCalls[2]
			require.IsType(t, &interpreter.ArrayValue{}, third)

			assert.True(t,
				third.(*interpreter.ArrayValue).
					Equal(inter, emptyLocationRange, newArrayValue(inter)),
			)
		})

		t.Run("S(foo: {Int: Bool})", func(t *testing.T) {

			t.Parallel()

			dictionaryStaticType := interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeInt,
				interpreter.PrimitiveStaticTypeBool,
			)

			newDictionaryValue := func(inter *interpreter.Interpreter) *interpreter.DictionaryValue {
				return interpreter.NewDictionaryValueWithAddress(
					inter,
					interpreter.EmptyLocationRange,
					dictionaryStaticType,
					testAddress,
					interpreter.NewUnmeteredIntValueFromInt64(42),
					interpreter.BoolValue(true),
				)
			}

			newCompositeValue := func(inter *interpreter.Interpreter) *interpreter.CompositeValue {
				compositeValue := interpreter.NewCompositeValue(
					inter,
					interpreter.EmptyLocationRange,
					utils.TestLocation,
					"S",
					common.CompositeKindStructure,
					nil,
					testAddress,
				)

				compositeValue.SetMemberWithoutTransfer(
					inter,
					emptyLocationRange,
					"foo",
					newDictionaryValue(inter),
				)

				return compositeValue
			}

			migrationCalls, inter := migrate(
				t,
				func(inter *interpreter.Interpreter) interpreter.Value {
					return newCompositeValue(inter)
				},
				canSkip,
			)

			// NOTE: the dictionary value and its children are skipped!
			require.Len(t, migrationCalls, 1)

			// first

			first := migrationCalls[0]
			require.IsType(t, &interpreter.CompositeValue{}, first)

			assert.True(t,
				first.(*interpreter.CompositeValue).
					Equal(inter, emptyLocationRange, newCompositeValue(inter)),
			)
		})

	})
}

// testPublishedValueMigration

type testPublishedValueMigration struct{}

var _ ValueMigration = testPublishedValueMigration{}

func (testPublishedValueMigration) Name() string {
	return "testPublishedValueMigration"
}

func (testPublishedValueMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	if pathCap, ok := value.(*interpreter.PathCapabilityValue); ok { //nolint:staticcheck
		return pathCap, nil
	}

	return nil, nil
}

func (testPublishedValueMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testPublishedValueMigration) Domains() map[string]struct{} {
	return nil
}

func TestPublishedValueMigration(t *testing.T) {

	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})

	ledger := NewTestLedger(nil, nil)
	storage := runtime.NewStorage(ledger, nil)

	storageMap := storage.GetStorageMap(
		testAddress,
		stdlib.InboxStorageDomain,
		true,
	)
	require.NotNil(t, storageMap)

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

	storageMapKey := interpreter.StringStorageMapKey("test")

	storageMap.WriteValue(
		inter,
		storageMapKey,
		interpreter.NewPublishedValue(
			nil,
			interpreter.AddressValue(testAddress),
			interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				nil,
				interpreter.AddressValue{0x2},
				interpreter.PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
			),
		),
	)

	// Migrate
	migration, err := NewStorageMigration(inter, storage, "test", testAddress)
	require.NoError(t, err)

	reporter := newTestReporter()

	migration.Migrate(
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testPublishedValueMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Len(t, reporter.errors, 0)
	assert.Len(t, reporter.migrated, 1)

	err = storage.CheckHealth()
	require.NoError(t, err)
}

// testDomainsMigration

type testDomainsMigration struct {
	domains map[string]struct{}
}

var _ ValueMigration = testDomainsMigration{}

func (testDomainsMigration) Name() string {
	return "testDomainsMigration"
}

func (m testDomainsMigration) Migrate(
	storageKey interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	_ interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {

	if m.domains != nil {
		_, ok := m.domains[storageKey.Key]
		if !ok {
			panic("invalid domain")
		}
	}

	return interpreter.NewUnmeteredStringValue("42"), nil
}

func (testDomainsMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (m testDomainsMigration) Domains() map[string]struct{} {
	return m.domains
}

func TestDomainsMigration(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, migratorDomains map[string]struct{}) {

		testAddress := common.MustBytesToAddress([]byte{0x1})

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

		storageMapKey := interpreter.StringStorageMapKey("test")

		storedDomains := []string{
			common.PathDomainStorage.Identifier(),
			common.PathDomainPublic.Identifier(),
			common.PathDomainPrivate.Identifier(),
			stdlib.InboxStorageDomain,
		}

		for _, domain := range storedDomains {

			storageMap := storage.GetStorageMap(
				testAddress,
				domain,
				true,
			)
			require.NotNil(t, storageMap)

			storageMap.WriteValue(
				inter,
				storageMapKey,
				interpreter.NewUnmeteredInt8Value(42),
			)
		}

		// Migrate

		migration, err := NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				testDomainsMigration{
					domains: migratorDomains,
				},
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		assert.Len(t, reporter.errors, 0)

		expectedMigrated := len(migratorDomains)
		if migratorDomains == nil {
			expectedMigrated = len(storedDomains)
		}
		assert.Len(t, reporter.migrated, expectedMigrated)
	}

	t.Run("no domains", func(t *testing.T) {
		test(t, nil)
	})

	t.Run("only storage", func(t *testing.T) {
		t.Parallel()

		test(t, map[string]struct{}{
			common.PathDomainStorage.Identifier(): {},
		})
	})

	t.Run("only storage and inbox", func(t *testing.T) {
		t.Parallel()

		test(t, map[string]struct{}{
			common.PathDomainStorage.Identifier(): {},
			stdlib.InboxStorageDomain:             {},
		})
	})
}

// testDictionaryKeyConflictMigration

type testDictionaryKeyConflictMigration struct {
	migrateValue bool
}

var _ ValueMigration = testDictionaryKeyConflictMigration{}

func (testDictionaryKeyConflictMigration) Name() string {
	return "testDictionaryKeyConflictMigration"
}

func (m testDictionaryKeyConflictMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	_ *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	typeValue, ok := value.(interpreter.TypeValue)
	if ok {
		return typeValue, nil
	}

	if m.migrateValue {
		intValue, ok := value.(interpreter.IntValue)
		if ok {
			return interpreter.NewUnmeteredIntValueFromInt64(int64(intValue.ToInt(emptyLocationRange)) + 10), nil
		}
	}

	return nil, nil
}

func (testDictionaryKeyConflictMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testDictionaryKeyConflictMigration) Domains() map[string]struct{} {
	return nil
}

func TestDictionaryKeyConflict(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, migrateValue bool) {

		testAddress := common.MustBytesToAddress([]byte{0x1})
		storagePathDomain := common.PathDomainStorage.Identifier()
		storageMapKey := interpreter.StringStorageMapKey("test")

		ledger := NewTestLedger(nil, nil)

		newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
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

			return storage, inter
		}

		fooQualifiedIdentifier := "Test.Foo"
		fooType := &interpreter.InterfaceStaticType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: fooQualifiedIdentifier,
			TypeID:              utils.TestLocation.TypeID(nil, fooQualifiedIdentifier),
		}

		barQualifiedIdentifier := "Test.Bar"
		barType := &interpreter.InterfaceStaticType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: barQualifiedIdentifier,
			TypeID:              utils.TestLocation.TypeID(nil, barQualifiedIdentifier),
		}

		// Intersection types only differ in order of interfaces

		intersectionType1 := interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				fooType,
				barType,
			},
		)

		intersectionType2 := interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				barType,
				fooType,
			},
		)

		dictionaryKey1 := interpreter.NewTypeValue(nil, intersectionType1)
		dictionaryKey2 := interpreter.NewTypeValue(nil, intersectionType2)

		// {Type: [Int]}
		// Value is an array to ensure slabs are created
		arrayType := interpreter.NewVariableSizedStaticType(nil,
			interpreter.PrimitiveStaticTypeInt,
		)

		// Prepare
		(func() {
			storage, inter := newStorageAndInterpreter(t)

			storageMap := storage.GetStorageMap(
				testAddress,
				storagePathDomain,
				true,
			)

			dictionaryValue := interpreter.NewDictionaryValueWithAddress(
				inter,
				emptyLocationRange,
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeMetaType,
					arrayType,
				),
				testAddress,
			)

			// Write the dictionary value to storage before inserting values into dictionary,
			// as the insertion of values into the dictionary triggers a storage health check,
			// which fails if the dictionary value is not yet stored (unreferenced slabs)

			storageMap.WriteValue(
				inter,
				storageMapKey,
				dictionaryValue,
			)

			// NOTE: use LegacyKey to ensure the key is encoded in old format

			dictionaryValue.InsertWithoutTransfer(
				inter,
				emptyLocationRange,
				LegacyKey(dictionaryKey1),
				interpreter.NewArrayValue(
					inter,
					emptyLocationRange,
					arrayType,
					testAddress,
					interpreter.NewUnmeteredIntValueFromInt64(1),
				),
			)

			dictionaryValue.InsertWithoutTransfer(
				inter,
				emptyLocationRange,
				LegacyKey(dictionaryKey2),
				interpreter.NewArrayValue(
					inter,
					emptyLocationRange,
					arrayType,
					testAddress,
					interpreter.NewUnmeteredIntValueFromInt64(2),
				),
			)

			oldValue1, ok := dictionaryValue.Get(
				inter,
				emptyLocationRange,
				LegacyKey(dictionaryKey1),
			)
			require.True(t, ok)

			utils.AssertValuesEqual(t,
				inter,
				oldValue1,
				interpreter.NewArrayValue(
					inter,
					emptyLocationRange,
					arrayType,
					common.ZeroAddress,
					interpreter.NewUnmeteredIntValueFromInt64(1),
				),
			)

			oldValue2, ok := dictionaryValue.Get(
				inter,
				emptyLocationRange,
				LegacyKey(dictionaryKey2),
			)
			require.True(t, ok)

			utils.AssertValuesEqual(t,
				inter,
				oldValue2,
				interpreter.NewArrayValue(
					inter,
					emptyLocationRange,
					arrayType,
					common.ZeroAddress,
					interpreter.NewUnmeteredIntValueFromInt64(2),
				),
			)

			err := storage.Commit(inter, false)
			require.NoError(t, err)

			err = storage.CheckHealth()
			require.NoError(t, err)
		})()

		// Migrate
		(func() {

			storage, inter := newStorageAndInterpreter(t)

			migration, err := NewStorageMigration(inter, storage, "test", testAddress)
			require.NoError(t, err)

			reporter := newTestReporter()

			migration.Migrate(
				migration.NewValueMigrationsPathMigrator(
					reporter,
					testDictionaryKeyConflictMigration{
						migrateValue: migrateValue,
					},
				),
			)

			err = migration.Commit()
			require.NoError(t, err)

			// Assert

			require.Len(t, reporter.errors, 1)
			assert.ErrorContains(
				t,
				reporter.errors[0],
				"dictionary key conflict",
			)

			assert.Len(t, reporter.migrated, 1)

			err = storage.CheckHealth()
			require.NoError(t, err)

			// Check storage map

			storageMap := storage.GetStorageMap(testAddress, storagePathDomain, false)
			require.NotNil(t, storageMap)
			require.Equal(t, uint64(2), storageMap.Count())

			// Check existing migrated dictionary

			migratedValue := storageMap.ReadValue(nil, storageMapKey)
			require.NotNil(t, migratedValue)

			require.IsType(t, &interpreter.DictionaryValue{}, migratedValue)
			migratedDict := migratedValue.(*interpreter.DictionaryValue)

			value, _ := migratedDict.Get(inter, emptyLocationRange, dictionaryKey2)
			require.NotNil(t, value)

			expectedInt2 := interpreter.NewUnmeteredIntValueFromInt64(2)
			if migrateValue {
				expectedInt2 = interpreter.NewUnmeteredIntValueFromInt64(12)
			}

			utils.RequireValuesEqual(t,
				inter,
				interpreter.NewArrayValue(
					inter,
					emptyLocationRange,
					arrayType,
					common.ZeroAddress,
					expectedInt2,
				),
				value,
			)

			// Check newly created conflict dictionary

			conflictValue := storageMap.ReadValue(nil, migration.DictionaryKeyConflictStorageMapKey(1))
			require.NotNil(t, conflictValue)

			require.IsType(t, &interpreter.DictionaryValue{}, conflictValue)
			conflictDict := conflictValue.(*interpreter.DictionaryValue)

			value, _ = conflictDict.Get(inter, emptyLocationRange, dictionaryKey1)
			require.NotNil(t, value)

			expectedInt1 := interpreter.NewUnmeteredIntValueFromInt64(1)
			if migrateValue {
				expectedInt1 = interpreter.NewUnmeteredIntValueFromInt64(11)
			}

			utils.RequireValuesEqual(t,
				inter,
				interpreter.NewArrayValue(
					inter,
					emptyLocationRange,
					arrayType,
					common.ZeroAddress,
					expectedInt1,
				),
				value,
			)
		})()
	}

	t.Run("value migrated", func(t *testing.T) {
		t.Parallel()

		test(t, true)
	})

	t.Run("value not migrated", func(t *testing.T) {
		t.Parallel()

		test(t, false)
	})
}

//go:embed testdata/missing-slabs-payloads.csv
var missingSlabsPayloadsData []byte

// '$' + 8 byte index
const slabKeyLength = 9

func isSlabStorageKey(key []byte) bool {
	return len(key) == slabKeyLength && key[0] == '$'
}

func TestFixLoadedBrokenReferences(t *testing.T) {

	t.Parallel()

	// Read CSV file with test data

	reader := csv.NewReader(bytes.NewReader(missingSlabsPayloadsData))

	// account, key, value
	reader.FieldsPerRecord = 3

	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Load data into ledger. Skip header

	ledger := NewTestLedger(nil, nil)

	for _, record := range records[1:] {
		account, err := hex.DecodeString(record[0])
		require.NoError(t, err)

		key, err := hex.DecodeString(record[1])
		require.NoError(t, err)

		value, err := hex.DecodeString(record[2])
		require.NoError(t, err)

		err = ledger.SetValue(account, key, value)
		require.NoError(t, err)
	}

	storage := runtime.NewStorage(ledger, nil)

	// Check health.
	// Retrieve all slabs before migration

	err = ledger.ForEach(func(owner, key, value []byte) error {

		if !isSlabStorageKey(key) {
			return nil
		}

		// Convert the owner/key to a storage ID.

		var slabIndex atree.SlabIndex
		copy(slabIndex[:], key[1:])

		storageID := atree.NewSlabID(atree.Address(owner), slabIndex)

		// Retrieve the slab.
		_, _, err = storage.Retrieve(storageID)
		require.NoError(t, err)

		return nil
	})
	require.NoError(t, err)

	address, err := common.HexToAddress("0x5d63c34d7f05e5a4")
	require.NoError(t, err)

	for _, domain := range common.AllPathDomains {
		_ = storage.GetStorageMap(address, domain.Identifier(), false)
	}

	err = storage.CheckHealth()
	require.Error(t, err)

	require.ErrorContains(t, err, "slab (0x0.49) not found: slab not found during slab iteration")

	// Fix the broken slab references

	fixedSlabs, skippedSlabIDs, err := storage.PersistentSlabStorage.
		FixLoadedBrokenReferences(ShouldFixBrokenCompositeKeyedDictionary)
	require.NoError(t, err)

	require.NotEmpty(t, fixedSlabs)
	require.Empty(t, skippedSlabIDs)

	// Re-run health check. This time it should pass.

	err = storage.CheckHealth()
	require.NoError(t, err)
}

// TestMigrateNestedValue is a reproducer for issue #3288.
// https://github.com/onflow/cadence/issues/3288
// The reproducer uses a simplified data structure:
// dict (not inlined) -> composite (inlined) -> dict (not inlined)
// After migration, data structure is changed to:
// dict (not inlined) -> composite (inlined) -> dict (inlined)
func TestMigrateNestedValue(t *testing.T) {

	account := common.Address{0x42}

	elaboration := sema.NewElaboration(nil)

	const s1QualifiedIdentifier = "S1"

	elaboration.SetCompositeType(
		utils.TestLocation.TypeID(nil, s1QualifiedIdentifier),
		&sema.CompositeType{
			Location:   utils.TestLocation,
			Members:    &sema.StringMemberOrderedMap{},
			Identifier: s1QualifiedIdentifier,
			Kind:       common.CompositeKindStructure,
		},
	)

	storageDomain := "storage"
	storageMapKey := interpreter.StringStorageMapKey("foo")

	createData := func(storageDomain string, storageMapKey interpreter.StorageMapKey) map[string][]byte {
		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(ledger, nil)

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Elaboration: elaboration,
			},
			utils.TestLocation,
			&interpreter.Config{
				Storage:                     storage,
				AtreeValueValidationEnabled: true,
				// NOTE: disabled, as storage is not expected to be always valid _during_ migration
				AtreeStorageValidationEnabled: false,
			},
		)
		require.NoError(t, err)

		dictionaryAnyStructStaticType :=
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeAnyStruct,
				interpreter.PrimitiveStaticTypeAnyStruct,
			)

		// Nested data structure in testnet account 0xa47a2d3a3b7e9133:
		// dictionary (not inlined) ->
		// 	  composite (inlined) ->
		//    	  dictionary (inlined) ->
		//    		  composite (inlined) ->
		//    			  dictionary (not inlined)

		// Nested data structure used to reproduce issue #3288:
		// "parentDict" (not inlined) ->
		//     "childComposite" (inlined) ->
		//         "gchildDict" (not inlined)

		// Create a dictionary value with 8 elements:
		// {
		//	    "grand_child_dict_key_0":"grand_child_dict_value_0",
		//      ...,
		//      "grand_child_dict_key_7":"grand_child_dict_value_7"
		// }
		const gchildDictCount = 8
		gchildDictElements := make([]interpreter.Value, 0, 2*gchildDictCount)
		for i := 0; i < gchildDictCount; i++ {
			k := interpreter.NewUnmeteredStringValue("grand_child_dict_key_" + strconv.Itoa(i))
			v := interpreter.NewUnmeteredStringValue("grand_child_dict_value_" + strconv.Itoa(i))
			gchildDictElements = append(gchildDictElements, k, v)
		}

		gchildDict := interpreter.NewDictionaryValue(
			inter,
			emptyLocationRange,
			dictionaryAnyStructStaticType,
			gchildDictElements...,
		)

		// Create a composite value with 1 field "bar":
		// {
		//     bar:{
		//	        "grand_child_dict_key_0":"grand_child_dict_value_0",
		//          ...,
		//          "grand_child_dict_key_9":"grand_child_dict_value_9"
		//     }
		// }
		// Under the hood, nested dictionary is referenced by atree SlabID (not inlined).
		childComposite := interpreter.NewCompositeValue(
			inter,
			emptyLocationRange,
			utils.TestLocation,
			s1QualifiedIdentifier,
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "bar",
					Value: gchildDict,
				},
			},
			common.ZeroAddress,
		)

		// Create a dictionary value with 2 elements:
		// {
		//	    "parent_dict_key_0": {bar:{"grand_child_dict_key_0":"grand_child_dict_value_0", ...}},
		//      ...,
		//      "parent_dict_key_1":"parent_dict_value_1"
		// }
		// Under the hood, nested composite (childComposite) is inlined, while gchildDict remains to be not inlined.
		const parentDictCount = 2
		parentDictElements := make([]interpreter.Value, 0, 2*parentDictCount)
		for i := 0; i < parentDictCount; i++ {
			var k, v interpreter.Value

			k = interpreter.NewUnmeteredStringValue("parent_dict_key_" + strconv.Itoa(i))

			if i == 0 {
				v = childComposite
			} else {
				v = interpreter.NewUnmeteredStringValue("parent_dict_value_" + strconv.Itoa(i))
			}

			parentDictElements = append(parentDictElements, k, v)
		}

		parentDict := interpreter.NewDictionaryValueWithAddress(
			inter,
			emptyLocationRange,
			dictionaryAnyStructStaticType,
			account,
			parentDictElements...,
		)

		// Create storage map under "storage" domain.
		storageMap := storage.GetStorageMap(account, storageDomain, true)

		// Add parentDict (not inlined) to storage map.
		exist := storageMap.WriteValue(inter, storageMapKey, parentDict)
		require.False(t, exist)

		err = storage.Commit(inter, true)
		require.NoError(t, err)

		// Expect 3 registers:
		// - register contains slab index for storage map of "storage" domain
		// - register for storage map of "storage" domain with parentDict inlined
		// - register for gchildDict
		const expectedNonEmptyRegisterCount = 3

		// Verify that not empty registers
		storedValues := ledger.StoredValues
		nonEmptyRegisterCount := 0
		for _, v := range storedValues {
			if len(v) > 0 {
				nonEmptyRegisterCount++
			}
		}
		require.Equal(t, expectedNonEmptyRegisterCount, nonEmptyRegisterCount)

		return storedValues
	}

	ledgerData := createData(storageDomain, storageMapKey)

	// Check health of ledger data before migration.
	checkHealth(t, account, ledgerData)

	ledger := NewTestLedgerWithData(
		nil,
		nil,
		ledgerData,
		map[string]uint64{string(account[:]): uint64(len(ledgerData))},
	)

	storage := runtime.NewStorage(ledger, nil)

	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		&interpreter.Config{
			Storage:                     storage,
			AtreeValueValidationEnabled: true,
			// NOTE: disabled, as storage is not expected to be always valid _during_ migration
			AtreeStorageValidationEnabled: false,
		},
	)
	require.NoError(t, err)

	storageMap := storage.GetStorageMap(account, storageDomain, false)
	require.NotNil(t, storageMap)

	value := storageMap.ReadValue(inter, storageMapKey)
	require.NotNil(t, value)

	migration, err := NewStorageMigration(
		inter,
		storage,
		"test",
		account,
	)
	require.NoError(t, err)

	reporter := newTestReporter()

	// Migration migrates all gchildDict element values from "grand_child_dict_value_x" to 0.
	// This causes gchildDict (was not inlined) to be inlined in its parent childComposite.
	// So after migration, number of registers should be decreased by 1 (from not inlined to inlined).
	migration.MigrateNestedValue(
		interpreter.StorageKey{
			Key:     storageDomain,
			Address: account,
		},
		storageMapKey,
		value,
		[]ValueMigration{
			newTestMigration(inter, func(
				_ interpreter.StorageKey,
				_ interpreter.StorageMapKey,
				value interpreter.Value,
				_ *interpreter.Interpreter,
				_ ValueMigrationPosition,
			) (
				interpreter.Value,
				error,
			) {
				switch value := value.(type) {
				case *interpreter.StringValue:
					if strings.HasPrefix(value.Str, "grand_child_dict_value_") {
						return interpreter.Int64Value(0), nil
					}
				}

				return nil, nil
			}),
		},
		reporter,
		true,
		ValueMigrationPositionOther,
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Check health of ledger data after migration.
	checkHealth(t, account, ledger.StoredValues)

	// Expect 2 registers:
	// - register contains slab index for storage map of "storage" domain
	// - register for storage map of "storage" domain with parentDict, childComposite and gchildDict inlined
	const expectedNonEmptyRegisterCount = 2

	// Verify that not empty registers
	storedValues := ledger.StoredValues
	nonEmptyRegisterCount := 0
	for _, v := range storedValues {
		if len(v) > 0 {
			nonEmptyRegisterCount++
		}
	}
	require.Equal(t, expectedNonEmptyRegisterCount, nonEmptyRegisterCount)
}

// TestMigrateNestedComposite is counterpart to TestMigrateNestedValue by
// using a slightly different data structure:
// composite (not inlined) -> composite (inlined) -> dict (not inlined)
// After migration, data structure is changed to:
// composite (not inlined) -> composite (inlined) -> dict (inlined)
func TestMigrateNestedComposite(t *testing.T) {

	account := common.Address{0x42}

	elaboration := sema.NewElaboration(nil)

	const s1QualifiedIdentifier = "S1"

	elaboration.SetCompositeType(
		utils.TestLocation.TypeID(nil, s1QualifiedIdentifier),
		&sema.CompositeType{
			Location:   utils.TestLocation,
			Members:    &sema.StringMemberOrderedMap{},
			Identifier: s1QualifiedIdentifier,
			Kind:       common.CompositeKindStructure,
		},
	)

	storageDomain := "storage"
	storageMapKey := interpreter.StringStorageMapKey("foo")

	createData := func(storageDomain string, storageMapKey interpreter.StorageMapKey) map[string][]byte {
		ledger := NewTestLedger(nil, nil)
		storage := runtime.NewStorage(ledger, nil)

		inter, err := interpreter.NewInterpreter(
			&interpreter.Program{
				Elaboration: elaboration,
			},
			utils.TestLocation,
			&interpreter.Config{
				Storage:                     storage,
				AtreeValueValidationEnabled: true,
				// NOTE: disabled, as storage is not expected to be always valid _during_ migration
				AtreeStorageValidationEnabled: false,
			},
		)
		require.NoError(t, err)

		dictionaryAnyStructStaticType :=
			interpreter.NewDictionaryStaticType(
				nil,
				interpreter.PrimitiveStaticTypeAnyStruct,
				interpreter.PrimitiveStaticTypeAnyStruct,
			)

		// Nested data structure used in this test (no dictionary):
		// "parentComposite" (not inlined) ->
		//     "childComposite" (inlined) ->
		//         "gchildDict" (not inlined)

		// Create a dictionary value with 10 elements:
		// {
		//	    "grand_child_dict_key_0":"grand_child_dict_value_0",
		//      ...,
		//      "grand_child_dict_key_9":"grand_child_dict_value_9"
		// }
		const gchildDictCount = 10
		gchildDictElements := make([]interpreter.Value, 0, 2*gchildDictCount)
		for i := 0; i < gchildDictCount; i++ {
			k := interpreter.NewUnmeteredStringValue("grand_child_dict_key_" + strconv.Itoa(i))
			v := interpreter.NewUnmeteredStringValue("grand_child_dict_value_" + strconv.Itoa(i))
			gchildDictElements = append(gchildDictElements, k, v)
		}

		gchildDict := interpreter.NewDictionaryValue(
			inter,
			emptyLocationRange,
			dictionaryAnyStructStaticType,
			gchildDictElements...,
		)

		// Create a composite value with 1 field "bar":
		// {
		//     bar:{
		//	        "grand_child_dict_key_0":"grand_child_dict_value_0",
		//          ...,
		//          "grand_child_dict_key_9":"grand_child_dict_value_9"
		//     }
		// }
		// Under the hood, nested composite is referenced by atree SlabID (not inlined).
		childComposite := interpreter.NewCompositeValue(
			inter,
			emptyLocationRange,
			utils.TestLocation,
			s1QualifiedIdentifier,
			common.CompositeKindStructure,
			[]interpreter.CompositeField{
				{
					Name:  "bar",
					Value: gchildDict,
				},
			},
			common.ZeroAddress,
		)

		// Create a composite value with 20 fields:
		// {
		//	    parent_field_0: {bar:{"grand_child_dict_key_0":"grand_child_dict_value_0", ...}},
		//      ...,
		//      parent_field_19:"parent_field_value_19"
		// }
		// Under the hood, nested composite (childComposite) is inlined, while gchildDict remains to be not inlined.
		const parentCompositeFieldCount = 20
		parentCompositeFields := make([]interpreter.CompositeField, 0, parentCompositeFieldCount)
		for i := 0; i < parentCompositeFieldCount; i++ {
			name := fmt.Sprintf("parent_field_%d", i)

			var value interpreter.Value
			if i == 0 {
				value = childComposite
			} else {
				value = interpreter.NewUnmeteredStringValue("parent_field_value_" + strconv.Itoa(i))
			}

			parentCompositeFields = append(
				parentCompositeFields,
				interpreter.CompositeField{
					Name:  name,
					Value: value,
				})
		}

		parentComposite := interpreter.NewCompositeValue(
			inter,
			emptyLocationRange,
			utils.TestLocation,
			s1QualifiedIdentifier,
			common.CompositeKindStructure,
			parentCompositeFields,
			account,
		)

		// Create storage map under "storage" domain.
		storageMap := storage.GetStorageMap(account, storageDomain, true)

		// Add parentComposite (not inlined) to storage map.
		exist := storageMap.WriteValue(inter, storageMapKey, parentComposite)
		require.False(t, exist)

		err = storage.Commit(inter, true)
		require.NoError(t, err)

		// Expect 4 registers:
		// - register contains slab index for storage map of "storage" domain
		// - register for storage map of "storage" domain
		// - register for parentComposite
		// - register for gchildDict
		const expectedNonEmptyRegisterCount = 4

		// Verify that not empty registers
		storedValues := ledger.StoredValues
		nonEmptyRegisterCount := 0
		for _, v := range storedValues {
			if len(v) > 0 {
				nonEmptyRegisterCount++
			}
		}
		require.Equal(t, expectedNonEmptyRegisterCount, nonEmptyRegisterCount)

		return storedValues
	}

	ledgerData := createData(storageDomain, storageMapKey)

	// Check health of ledger data before migration.
	checkHealth(t, account, ledgerData)

	ledger := NewTestLedgerWithData(
		nil,
		nil,
		ledgerData,
		map[string]uint64{string(account[:]): uint64(len(ledgerData))},
	)

	storage := runtime.NewStorage(ledger, nil)

	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{
			Elaboration: elaboration,
		},
		utils.TestLocation,
		&interpreter.Config{
			Storage:                     storage,
			AtreeValueValidationEnabled: true,
			// NOTE: disabled, as storage is not expected to be always valid _during_ migration
			AtreeStorageValidationEnabled: false,
		},
	)
	require.NoError(t, err)

	storageMap := storage.GetStorageMap(account, storageDomain, false)
	require.NotNil(t, storageMap)

	value := storageMap.ReadValue(inter, storageMapKey)
	require.NotNil(t, value)

	migration, err := NewStorageMigration(
		inter,
		storage,
		"test",
		account,
	)
	require.NoError(t, err)

	reporter := newTestReporter()

	// Migration migrates all gchildComposite element values from "grand_child_dict_value_x" to 0.
	// This causes gchildDict (was not inlined) to be inlined in its parent childComposite.
	// So after migration, number of registers should be decreased by 1 (from not inlined to inlined).
	migration.MigrateNestedValue(
		interpreter.StorageKey{
			Key:     storageDomain,
			Address: account,
		},
		storageMapKey,
		value,
		[]ValueMigration{
			newTestMigration(inter, func(
				_ interpreter.StorageKey,
				_ interpreter.StorageMapKey,
				value interpreter.Value,
				_ *interpreter.Interpreter,
				_ ValueMigrationPosition,
			) (
				interpreter.Value,
				error,
			) {
				switch value := value.(type) {
				case *interpreter.StringValue:
					if strings.HasPrefix(value.Str, "grand_child_dict_value_") {
						return interpreter.Int64Value(0), nil
					}
				}

				return nil, nil
			}),
		},
		reporter,
		true,
		ValueMigrationPositionOther,
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Check health of ledger data after migration.
	checkHealth(t, account, ledger.StoredValues)

	// Expect 3 registers:
	// - register contains slab index for storage map of "storage" domain
	// - register for storage map of "storage" domain
	// - register for parentComposite (childComposite and gchildDict are inlined)
	const expectedNonEmptyRegisterCount = 3

	// Verify that not empty registers
	storedValues := ledger.StoredValues
	nonEmptyRegisterCount := 0
	for _, v := range storedValues {
		if len(v) > 0 {
			nonEmptyRegisterCount++
		}
	}
	require.Equal(t, expectedNonEmptyRegisterCount, nonEmptyRegisterCount)
}

func checkHealth(t *testing.T, account common.Address, storedValues map[string][]byte) {
	ledger := NewTestLedgerWithData(nil, nil, storedValues, nil)

	storage := runtime.NewStorage(ledger, nil)

	// Load storage maps
	for _, domain := range common.AllPathDomains {
		_ = storage.GetStorageMap(account, domain.Identifier(), false)
	}

	// Load atree slabs
	err := loadAtreeSlabsInStorage(storage, storedValues)
	require.NoError(t, err)

	err = storage.CheckHealth()
	require.NoError(t, err)
}

type migrateFunc func(
	interpreter.StorageKey,
	interpreter.StorageMapKey,
	interpreter.Value,
	*interpreter.Interpreter,
	ValueMigrationPosition,
) (interpreter.Value, error)

type testMigration struct {
	inter   *interpreter.Interpreter
	migrate migrateFunc
}

var _ ValueMigration = testMigration{}

func newTestMigration(inter *interpreter.Interpreter, migrate migrateFunc) testMigration {
	return testMigration{
		inter:   inter,
		migrate: migrate,
	}
}

func (testMigration) Name() string {
	return "Test Migration"
}

func (m testMigration) Migrate(
	key interpreter.StorageKey,
	mapKey interpreter.StorageMapKey,
	value interpreter.Value,
	inter *interpreter.Interpreter,
	position ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	if m.migrate != nil {
		return m.migrate(key, mapKey, value, inter, position)
	}
	return nil, nil
}

func (m testMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testMigration) Domains() map[string]struct{} {
	return nil
}

func loadAtreeSlabsInStorage(storage *runtime.Storage, storedValues map[string][]byte) error {
	splitKey := func(s string) (owner string, key string, err error) {
		results := strings.Split(s, "|")
		if len(results) != 2 {
			return "", "", fmt.Errorf("failed to split key into owner and key: expected 2 elements, got %d elements", len(results))
		}
		return results[0], results[1], nil
	}

	for k := range storedValues {
		owner, key, err := splitKey(k)
		if err != nil {
			return err
		}

		if key[0] != '$' {
			continue
		}

		slabID := atree.NewSlabID(
			atree.Address([]byte(owner[:])),
			atree.SlabIndex([]byte(key[1:])))

		// Retrieve the slab.
		_, _, err = storage.Retrieve(slabID)
		if err != nil {
			return fmt.Errorf("failed to retrieve slab %s: %w", slabID, err)
		}
	}

	return nil
}

// testEnumMigration

type testEnumMigration struct{}

var _ ValueMigration = testEnumMigration{}

func (testEnumMigration) Name() string {
	return "testEnumMigration"
}

func (testEnumMigration) Migrate(
	_ interpreter.StorageKey,
	_ interpreter.StorageMapKey,
	value interpreter.Value,
	inter *interpreter.Interpreter,
	_ ValueMigrationPosition,
) (
	interpreter.Value,
	error,
) {
	if composite, ok := value.(*interpreter.CompositeValue); ok && composite.Kind == common.CompositeKindEnum {
		rawValue := composite.GetField(inter, emptyLocationRange, sema.EnumRawValueFieldName)
		raw := rawValue.(interpreter.UInt8Value)
		return interpreter.NewCompositeValue(
			inter,
			emptyLocationRange,
			composite.Location,
			composite.QualifiedIdentifier,
			common.CompositeKindEnum,
			[]interpreter.CompositeField{
				{
					Name:  sema.EnumRawValueFieldName,
					Value: raw + 1,
				},
			},
			composite.GetOwner(),
		), nil
	}

	return nil, nil
}

func (testEnumMigration) CanSkip(_ interpreter.StaticType) bool {
	return false
}

func (testEnumMigration) Domains() map[string]struct{} {
	return nil
}

func TestDictionaryWithEnumKey(t *testing.T) {

	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})
	storagePathDomain := common.PathDomainStorage.Identifier()
	storageMapKey := interpreter.StringStorageMapKey("test")

	ledger := NewTestLedger(nil, nil)

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
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

		return storage, inter
	}

	fooQualifiedIdentifier := "Test.Foo"
	fooType := &interpreter.CompositeStaticType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: fooQualifiedIdentifier,
		TypeID:              utils.TestLocation.TypeID(nil, fooQualifiedIdentifier),
	}

	// Prepare
	(func() {
		storage, inter := newStorageAndInterpreter(t)

		storageMap := storage.GetStorageMap(
			testAddress,
			storagePathDomain,
			true,
		)

		// {Test.Foo: String}
		dictionaryValue := interpreter.NewDictionaryValueWithAddress(
			inter,
			emptyLocationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				fooType,
				interpreter.PrimitiveStaticTypeAnyStruct,
			),
			testAddress,
		)

		// Write the dictionary value to storage before inserting values into dictionary,
		// as the insertion of values into the dictionary triggers a storage health check,
		// which fails if the dictionary value is not yet stored (unreferenced slabs)

		storageMap.WriteValue(
			inter,
			storageMapKey,
			dictionaryValue,
		)

		dictionaryKey := interpreter.NewCompositeValue(
			inter,
			emptyLocationRange,
			utils.TestLocation,
			"Foo",
			common.CompositeKindEnum,
			[]interpreter.CompositeField{
				{
					Name:  sema.EnumRawValueFieldName,
					Value: interpreter.UInt8Value(42),
				},
			},
			testAddress,
		)

		dictionaryValue.InsertWithoutTransfer(
			inter,
			emptyLocationRange,
			dictionaryKey,
			interpreter.NewUnmeteredStringValue("test"),
		)

		err := storage.Commit(inter, false)
		require.NoError(t, err)

		err = storage.CheckHealth()
		require.NoError(t, err)
	})()

	// Migrate
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		migration, err := NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				testStringMigration{},
				testEnumMigration{},
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		require.Len(t, reporter.errors, 0)

		assert.Len(t, reporter.migrated, 1)

		err = storage.CheckHealth()
		require.NoError(t, err)

		// Check storage map

		storageMap := storage.GetStorageMap(testAddress, storagePathDomain, false)
		require.NotNil(t, storageMap)
		require.Equal(t, uint64(1), storageMap.Count())

		// Check existing migrated dictionary

		migratedValue := storageMap.ReadValue(nil, storageMapKey)
		require.NotNil(t, migratedValue)

		require.IsType(t, &interpreter.DictionaryValue{}, migratedValue)
		migratedDict := migratedValue.(*interpreter.DictionaryValue)

		dictionaryKey2 := interpreter.NewCompositeValue(
			inter,
			emptyLocationRange,
			utils.TestLocation,
			"Foo",
			common.CompositeKindEnum,
			[]interpreter.CompositeField{
				{
					Name: sema.EnumRawValueFieldName,
					// NOTE: updated raw value
					Value: interpreter.UInt8Value(43),
				},
			},
			common.ZeroAddress,
		)

		value, _ := migratedDict.Get(inter, emptyLocationRange, dictionaryKey2)
		require.NotNil(t, value)

		utils.RequireValuesEqual(t,
			inter,
			interpreter.NewUnmeteredStringValue("updated_test"),
			value,
		)
	})()

}

func TestDictionaryKeyMutationMigration(t *testing.T) {

	t.Parallel()

	testAddress := common.MustBytesToAddress([]byte{0x1})
	storagePathDomain := common.PathDomainStorage.Identifier()
	storageMapKey := interpreter.StringStorageMapKey("test")

	ledger := NewTestLedger(nil, nil)

	newStorageAndInterpreter := func(t *testing.T) (*runtime.Storage, *interpreter.Interpreter) {
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

		return storage, inter
	}

	fooQualifiedIdentifier := "Test.Foo"
	fooType := &interpreter.CompositeStaticType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: fooQualifiedIdentifier,
		TypeID:              utils.TestLocation.TypeID(nil, fooQualifiedIdentifier),
	}

	// Prepare
	(func() {
		storage, inter := newStorageAndInterpreter(t)

		storageMap := storage.GetStorageMap(
			testAddress,
			storagePathDomain,
			true,
		)

		// {Test.Foo: Int8}
		dictionaryValue := interpreter.NewDictionaryValueWithAddress(
			inter,
			emptyLocationRange,
			interpreter.NewDictionaryStaticType(
				nil,
				fooType,
				interpreter.PrimitiveStaticTypeInt8,
			),
			testAddress,
		)

		// Write the dictionary value to storage before inserting values into dictionary,
		// as the insertion of values into the dictionary triggers a storage health check,
		// which fails if the dictionary value is not yet stored (unreferenced slabs)

		storageMap.WriteValue(
			inter,
			storageMapKey,
			dictionaryValue,
		)

		dictionaryKey := interpreter.NewCompositeValue(
			inter,
			emptyLocationRange,
			utils.TestLocation,
			"Foo",
			common.CompositeKindEnum,
			[]interpreter.CompositeField{
				{
					Name:  sema.EnumRawValueFieldName,
					Value: interpreter.Int8Value(10),
				},
			},
			testAddress,
		)

		dictionaryValue.InsertWithoutTransfer(
			inter,
			emptyLocationRange,
			dictionaryKey,
			interpreter.Int8Value(100),
		)

		err := storage.Commit(inter, false)
		require.NoError(t, err)

		err = storage.CheckHealth()
		require.NoError(t, err)
	})()

	// Migrate
	(func() {

		storage, inter := newStorageAndInterpreter(t)

		migration, err := NewStorageMigration(inter, storage, "test", testAddress)
		require.NoError(t, err)

		reporter := newTestReporter()

		migration.Migrate(
			migration.NewValueMigrationsPathMigrator(
				reporter,
				testInt8Migration{},
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		require.Len(t, reporter.errors, 1)
		assert.ErrorContains(
			t,
			reporter.errors[0],
			"mutation not allowed: attempting to migrate composite value field rawValue",
		)

		assert.Len(t, reporter.migrated, 1)

		err = storage.CheckHealth()
		require.NoError(t, err)
	})()
}
