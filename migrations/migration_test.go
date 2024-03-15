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

package migrations

import (
	"errors"
	"fmt"
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
) (interpreter.Value, error) {
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
) (interpreter.Value, error) {
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
) (interpreter.Value, error) {
	if value, ok := value.(*interpreter.IDCapabilityValue); ok {
		return interpreter.NewCapabilityValue(
			nil,
			value.ID+10,
			value.Address,
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
) (interpreter.Value, error) {

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

	migration := NewStorageMigration(inter, storage)

	reporter := newTestReporter()

	migration.MigrateAccount(
		account,
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

	migration := NewStorageMigration(inter, storage)

	reporter := newTestReporter()

	migration.MigrateAccount(
		account,
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

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnGetSigningAccounts: func() ([]runtime.Address, error) {
			return []runtime.Address{testAddress}, nil
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

	storageMap := storage.GetStorageMap(
		testAddress,
		stdlib.CapabilityControllerStorageDomain,
		false,
	)

	assert.Equal(t, uint64(2), storageMap.Count())

	controller1 := storageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(1))
	require.IsType(t, &interpreter.StorageCapabilityControllerValue{}, controller1)
	assert.Equal(t,
		interpreter.UInt64Value(1),
		controller1.(*interpreter.StorageCapabilityControllerValue).CapabilityID,
	)

	controller2 := storageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(2))
	require.IsType(t, &interpreter.AccountCapabilityControllerValue{}, controller2)
	assert.Equal(t,
		interpreter.UInt64Value(2),
		controller2.(*interpreter.AccountCapabilityControllerValue).CapabilityID,
	)

	// Migrate

	reporter := newTestReporter()

	migration := NewStorageMigration(inter, storage)

	migration.MigrateAccount(
		testAddress,
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testCapConMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Len(t, reporter.migrated, 2)

	require.Empty(t, reporter.errors)

	err = storage.CheckHealth()
	require.NoError(t, err)

	storageMap = storage.GetStorageMap(
		testAddress,
		stdlib.CapabilityControllerStorageDomain,
		false,
	)

	assert.Equal(t, uint64(2), storageMap.Count())

	controller1 = storageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(1))
	require.IsType(t, &interpreter.StorageCapabilityControllerValue{}, controller1)
	assert.Equal(t,
		interpreter.UInt64Value(11),
		controller1.(*interpreter.StorageCapabilityControllerValue).CapabilityID,
	)

	controller2 = storageMap.ReadValue(nil, interpreter.Uint64StorageMapKey(2))
	require.IsType(t, &interpreter.AccountCapabilityControllerValue{}, controller2)
	assert.Equal(t,
		interpreter.UInt64Value(12),
		controller2.(*interpreter.AccountCapabilityControllerValue).CapabilityID,
	)
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

	reporter := newTestReporter()

	migration := NewStorageMigration(inter, storage)

	migration.MigrateAccount(
		testAddress,
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

	dictionaryValue.Insert(
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

	reporter := newTestReporter()

	migration := NewStorageMigration(inter, storage)

	migration.MigrateAccount(
		testAddress,
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
) (interpreter.Value, error) {

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

		migration := NewStorageMigration(inter, storage)

		reporter := newTestReporter()

		migration.MigrateAccount(
			testAddress,
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
) (interpreter.Value, error) {

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

	migration := NewStorageMigration(inter, storage)

	reporter := newTestReporter()

	migration.MigrateAccount(
		testAddress,
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testPanicMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Len(t, reporter.errors, 1)

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
) (interpreter.Value, error) {

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

		migration := NewStorageMigration(inter, storage)

		reporter := newTestReporter()

		valueMigration := &testSkipMigration{
			canSkip: canSkip,
		}

		migration.MigrateAccount(
			testAddress,
			migration.NewValueMigrationsPathMigrator(
				reporter,
				valueMigration,
			),
		)

		err = migration.Commit()
		require.NoError(t, err)

		// Assert

		require.Empty(t, reporter.errors)

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

					return interpreter.NewArrayValue(
						inter,
						interpreter.EmptyLocationRange,
						interpreter.NewVariableSizedStaticType(
							nil,
							dictionaryStaticType,
						),
						testAddress,
						interpreter.NewDictionaryValueWithAddress(
							inter,
							interpreter.EmptyLocationRange,
							dictionaryStaticType,
							testAddress,
							interpreter.NewUnmeteredIntValueFromInt64(42),
							interpreter.BoolValue(true),
						),
					)
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
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.NewVariableSizedStaticType(
						nil,
						dictionaryStaticType,
					),
					testAddress,
					newDictionaryValue(inter),
				)
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
) (interpreter.Value, error) {

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
			&interpreter.PathCapabilityValue{ //nolint:staticcheck
				BorrowType: nil,
				Path: interpreter.PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
				Address: interpreter.AddressValue{0x2},
			},
		),
	)

	// Migrate

	reporter := newTestReporter()

	migration := NewStorageMigration(inter, storage)

	migration.MigrateAccount(
		testAddress,
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
) (interpreter.Value, error) {

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

		reporter := newTestReporter()

		migration := NewStorageMigration(inter, storage)

		migration.MigrateAccount(
			testAddress,
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
