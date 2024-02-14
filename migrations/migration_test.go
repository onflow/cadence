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
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type testReporter struct {
	migrated map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}][]string
	errored map[struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}][]string
}

var _ Reporter = &testReporter{}

func newTestReporter() *testReporter {
	return &testReporter{
		migrated: map[struct {
			interpreter.StorageKey
			interpreter.StorageMapKey
		}][]string{},
		errored: map[struct {
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

func (t *testReporter) Error(
	storageKey interpreter.StorageKey,
	storageMapKey interpreter.StorageMapKey,
	migration string,
	_ error,
) {
	key := struct {
		interpreter.StorageKey
		interpreter.StorageMapKey
	}{
		StorageKey:    storageKey,
		StorageMapKey: storageMapKey,
	}

	t.errored[key] = append(
		t.errored[key],
		migration,
	)
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
			AtreeValueValidationEnabled:   false,
			AtreeStorageValidationEnabled: false,
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

	migration.Migrate(
		&AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testStringMigration{},
			testInt8Migration{},
			testCapMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert: Traverse through the storage and see if the values are updated now.

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
			AtreeValueValidationEnabled:   false,
			AtreeStorageValidationEnabled: false,
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

	migration.Migrate(
		&AddressSliceIterator{
			Addresses: []common.Address{
				account,
			},
		},
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
				StorageMapKey: interpreter.StringStorageMapKey("int8_value"),
			}: {
				"testInt8Migration",
			},
		},
		reporter.errored,
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

	migration.Migrate(
		&AddressSliceIterator{
			Addresses: []common.Address{
				testAddress,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testCapConMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Len(t, reporter.errored, 0)
	assert.Len(t, reporter.migrated, 2)

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

	migration.Migrate(
		&AddressSliceIterator{
			Addresses: []common.Address{
				testAddress,
			},
		},
		migration.NewValueMigrationsPathMigrator(
			reporter,
			testStringMigration{},
		),
	)

	err = migration.Commit()
	require.NoError(t, err)

	// Assert

	assert.Len(t, reporter.errored, 0)
	assert.Len(t, reporter.migrated, 1)

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

func TestMigrationPanics(t *testing.T) {

	t.Parallel()

	t.Run("composite dictionary", func(t *testing.T) {
		t.Parallel()

		ledger := NewTestLedger(nil, nil)
		account := common.Address{0x42}

		t.Run("prepare", func(t *testing.T) {

			fooContractLocation := common.NewAddressLocation(nil, account, "Foo")

			storage := runtime.NewStorage(ledger, nil)
			locationRange := interpreter.EmptyLocationRange

			contractChecker, err := checker.ParseAndCheckWithOptions(t, `
                access(all) contract Foo {
                    access(all) struct Bar {}
                }`,
				checker.ParseAndCheckOptions{
					Location: fooContractLocation,
				},
			)
			require.NoError(t, err)

			inter, err := interpreter.NewInterpreter(
				nil,
				utils.TestLocation,
				&interpreter.Config{
					Storage:                       storage,
					AtreeValueValidationEnabled:   false,
					AtreeStorageValidationEnabled: false,
					ImportLocationHandler: func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
						program := interpreter.ProgramFromChecker(contractChecker)
						subInterpreter, err := inter.NewSubInterpreter(program, location)
						if err != nil {
							panic(err)
						}

						return interpreter.InterpreterImport{
							Interpreter: subInterpreter,
						}
					},
				},
			)
			require.NoError(t, err)

			const fooBarQualifiedIdentifier = "Foo.Bar"

			dictionaryAnyStructStaticType :=
				interpreter.NewDictionaryStaticType(
					nil,
					interpreter.PrimitiveStaticTypeString,
					interpreter.NewCompositeStaticTypeComputeTypeID(
						nil,
						fooContractLocation,
						fooBarQualifiedIdentifier,
					),
				)

			storedValue := interpreter.NewDictionaryValue(
				inter,
				emptyLocationRange,
				dictionaryAnyStructStaticType,
			)

			transferredValue := storedValue.Transfer(
				inter,
				locationRange,
				atree.Address(account),
				false,
				nil,
				nil,
			)

			inter.WriteStored(
				account,
				common.PathDomainStorage.Identifier(),
				interpreter.StringStorageMapKey("dictionary_value"),
				transferredValue,
			)

			err = storage.Commit(inter, true)
			require.NoError(t, err)
		})

		t.Run("migrate", func(t *testing.T) {
			storage := runtime.NewStorage(ledger, nil)

			inter, err := interpreter.NewInterpreter(
				nil,
				utils.TestLocation,
				&interpreter.Config{
					Storage:                       storage,
					AtreeValueValidationEnabled:   false,
					AtreeStorageValidationEnabled: false,
					ImportLocationHandler: func(_ *interpreter.Interpreter, _ common.Location) interpreter.Import {
						// During the migration, treat the imported `Foo` contract as un-migrated.
						panic(errors.New("type checking error in Foo"))
					},
				},
			)
			require.NoError(t, err)

			migration := NewStorageMigration(inter, storage)

			reporter := newTestReporter()

			migration.Migrate(
				&AddressSliceIterator{
					Addresses: []common.Address{
						account,
					},
				},
				migration.NewValueMigrationsPathMigrator(
					reporter,
					testStringMigration{},
					testInt8Migration{},
					testCapMigration{},
				),
			)

			err = migration.Commit()
			require.NoError(t, err)

			expectedErrors := map[struct {
				interpreter.StorageKey
				interpreter.StorageMapKey
			}][]string{
				{
					StorageKey: interpreter.StorageKey{
						Key:     common.PathDomainStorage.Identifier(),
						Address: account,
					},
					StorageMapKey: interpreter.StringStorageMapKey("dictionary_value"),
				}: {"StorageMigration"},
			}

			assert.Equal(t, expectedErrors, reporter.errored)
		})
	})
}
