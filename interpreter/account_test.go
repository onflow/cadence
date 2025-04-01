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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
)

type storageKey struct {
	address common.Address
	domain  string
	key     atree.Value
}

func testAccount(
	t *testing.T,
	address interpreter.AddressValue,
	auth bool,
	handler stdlib.AccountHandler,
	code string,
	checkerConfig sema.Config,
) (
	*interpreter.Interpreter,
	func() map[storageKey]interpreter.Value,
) {
	return testAccountWithErrorHandler(
		t,
		address,
		auth,
		handler,
		code,
		checkerConfig,
		nil,
	)
}

type testAccountHandler struct {
	accountIDs                 map[common.Address]uint64
	generateAccountID          func(address common.Address) (uint64, error)
	getAccountBalance          func(address common.Address) (uint64, error)
	getAccountAvailableBalance func(address common.Address) (uint64, error)
	commitStorageTemporarily   func(context interpreter.ValueTransferContext) error
	getStorageUsed             func(address common.Address) (uint64, error)
	getStorageCapacity         func(address common.Address) (uint64, error)
	validatePublicKey          func(key *stdlib.PublicKey) error
	verifySignature            func(
		signature []byte,
		tag string,
		signedData []byte,
		publicKey []byte,
		signatureAlgorithm sema.SignatureAlgorithm,
		hashAlgorithm sema.HashAlgorithm,
	) (
		bool,
		error,
	)
	blsVerifyPOP     func(publicKey *stdlib.PublicKey, signature []byte) (bool, error)
	hash             func(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error)
	getAccountKey    func(address common.Address, index uint32) (*stdlib.AccountKey, error)
	accountKeysCount func(address common.Address) (uint32, error)
	emitEvent        func(
		inter *interpreter.Interpreter,
		locationRange interpreter.LocationRange,
		eventType *sema.CompositeType,
		values []interpreter.Value,
	)
	addAccountKey func(
		address common.Address,
		key *stdlib.PublicKey,
		algo sema.HashAlgorithm,
		weight int,
	) (
		*stdlib.AccountKey,
		error,
	)
	revokeAccountKey       func(address common.Address, index uint32) (*stdlib.AccountKey, error)
	getAccountContractCode func(location common.AddressLocation) ([]byte, error)
	parseAndCheckProgram   func(
		code []byte,
		location common.Location,
		getAndSetProgram bool,
	) (
		*interpreter.Program,
		error,
	)
	updateAccountContractCode func(location common.AddressLocation, code []byte) error
	recordContractUpdate      func(location common.AddressLocation, value *interpreter.CompositeValue)
	contractUpdateRecorded    func(location common.AddressLocation) bool
	interpretContract         func(
		location common.AddressLocation,
		program *interpreter.Program,
		name string,
		invocation stdlib.DeployedContractConstructorInvocation,
	) (
		*interpreter.CompositeValue,
		error,
	)
	temporarilyRecordCode     func(location common.AddressLocation, code []byte)
	removeAccountContractCode func(location common.AddressLocation) error
	recordContractRemoval     func(location common.AddressLocation)
	getAccountContractNames   func(address common.Address) ([]string, error)
}

var _ stdlib.AccountHandler = &testAccountHandler{}

func (t *testAccountHandler) GenerateAccountID(address common.Address) (uint64, error) {
	if t.generateAccountID == nil {
		if t.accountIDs == nil {
			t.accountIDs = map[common.Address]uint64{}
		}
		t.accountIDs[address]++
		return t.accountIDs[address], nil
	}
	return t.generateAccountID(address)
}

func (t *testAccountHandler) GetAccountBalance(address common.Address) (uint64, error) {
	if t.getAccountBalance == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountBalance"))
	}
	return t.getAccountBalance(address)
}

func (t *testAccountHandler) GetAccountAvailableBalance(address common.Address) (uint64, error) {
	if t.getAccountAvailableBalance == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountAvailableBalance"))
	}
	return t.getAccountAvailableBalance(address)
}

func (t *testAccountHandler) CommitStorageTemporarily(context interpreter.ValueTransferContext) error {
	if t.commitStorageTemporarily == nil {
		panic(errors.NewUnexpectedError("unexpected call to CommitStorageTemporarily"))
	}
	return t.commitStorageTemporarily(context)
}

func (t *testAccountHandler) GetStorageUsed(address common.Address) (uint64, error) {
	if t.getStorageUsed == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetStorageUsed"))
	}
	return t.getStorageUsed(address)
}

func (t *testAccountHandler) GetStorageCapacity(address common.Address) (uint64, error) {
	if t.getStorageCapacity == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetStorageCapacity"))
	}
	return t.getStorageCapacity(address)
}

func (t *testAccountHandler) ValidatePublicKey(key *stdlib.PublicKey) error {
	if t.validatePublicKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to ValidatePublicKey"))
	}
	return t.validatePublicKey(key)
}

func (t *testAccountHandler) VerifySignature(
	signature []byte,
	tag string,
	signedData []byte,
	publicKey []byte,
	signatureAlgorithm sema.SignatureAlgorithm,
	hashAlgorithm sema.HashAlgorithm,
) (
	bool,
	error,
) {
	if t.verifySignature == nil {
		panic(errors.NewUnexpectedError("unexpected call to VerifySignature"))
	}
	return t.verifySignature(
		signature,
		tag,
		signedData,
		publicKey,
		signatureAlgorithm,
		hashAlgorithm,
	)
}

func (t *testAccountHandler) BLSVerifyPOP(publicKey *stdlib.PublicKey, signature []byte) (bool, error) {
	if t.blsVerifyPOP == nil {
		panic(errors.NewUnexpectedError("unexpected call to BLSVerifyPOP"))
	}
	return t.blsVerifyPOP(publicKey, signature)
}

func (t *testAccountHandler) Hash(data []byte, tag string, algorithm sema.HashAlgorithm) ([]byte, error) {
	if t.hash == nil {
		panic(errors.NewUnexpectedError("unexpected call to Hash"))
	}
	return t.hash(data, tag, algorithm)
}

func (t *testAccountHandler) GetAccountKey(address common.Address, index uint32) (*stdlib.AccountKey, error) {
	if t.getAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountKey"))
	}
	return t.getAccountKey(address, index)
}

func (t *testAccountHandler) AccountKeysCount(address common.Address) (uint32, error) {
	if t.accountKeysCount == nil {
		panic(errors.NewUnexpectedError("unexpected call to AccountKeysCount"))
	}
	return t.accountKeysCount(address)
}

func (t *testAccountHandler) EmitEvent(
	inter *interpreter.Interpreter,
	locationRange interpreter.LocationRange,
	eventType *sema.CompositeType,
	values []interpreter.Value,
) {
	if t.emitEvent == nil {
		panic(errors.NewUnexpectedError("unexpected call to EmitEvent"))
	}
	t.emitEvent(
		inter,
		locationRange,
		eventType,
		values,
	)
}

func (t *testAccountHandler) AddAccountKey(
	address common.Address,
	key *stdlib.PublicKey,
	algo sema.HashAlgorithm,
	weight int,
) (
	*stdlib.AccountKey,
	error,
) {
	if t.addAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to AddAccountKey"))
	}
	return t.addAccountKey(
		address,
		key,
		algo,
		weight,
	)
}

func (t *testAccountHandler) RevokeAccountKey(address common.Address, index uint32) (*stdlib.AccountKey, error) {
	if t.revokeAccountKey == nil {
		panic(errors.NewUnexpectedError("unexpected call to RevokeAccountKey"))
	}
	return t.revokeAccountKey(address, index)
}

func (t *testAccountHandler) GetAccountContractCode(location common.AddressLocation) ([]byte, error) {
	if t.getAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountContractCode"))
	}
	return t.getAccountContractCode(location)
}

func (t *testAccountHandler) ParseAndCheckProgram(
	code []byte,
	location common.Location,
	getAndSetProgram bool,
) (
	*interpreter.Program,
	error,
) {
	if t.parseAndCheckProgram == nil {
		panic(errors.NewUnexpectedError("unexpected call to ParseAndCheckProgram"))
	}
	return t.parseAndCheckProgram(code, location, getAndSetProgram)
}

func (t *testAccountHandler) UpdateAccountContractCode(location common.AddressLocation, code []byte) error {
	if t.updateAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to UpdateAccountContractCode"))
	}
	return t.updateAccountContractCode(location, code)
}

func (t *testAccountHandler) RecordContractUpdate(location common.AddressLocation, value *interpreter.CompositeValue) {
	if t.recordContractUpdate == nil {
		panic(errors.NewUnexpectedError("unexpected call to RecordContractUpdate"))
	}
	t.recordContractUpdate(location, value)
}

func (t *testAccountHandler) ContractUpdateRecorded(location common.AddressLocation) bool {
	if t.contractUpdateRecorded == nil {
		panic(errors.NewUnexpectedError("unexpected call to ContractUpdateRecorded"))
	}
	return t.contractUpdateRecorded(location)
}

func (t *testAccountHandler) InterpretContract(
	location common.AddressLocation,
	program *interpreter.Program,
	name string,
	invocation stdlib.DeployedContractConstructorInvocation,
) (
	*interpreter.CompositeValue,
	error,
) {
	if t.interpretContract == nil {
		panic(errors.NewUnexpectedError("unexpected call to InterpretContract"))
	}
	return t.interpretContract(
		location,
		program,
		name,
		invocation,
	)
}

func (t *testAccountHandler) TemporarilyRecordCode(location common.AddressLocation, code []byte) {
	if t.temporarilyRecordCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to TemporarilyRecordCode"))
	}
	t.temporarilyRecordCode(location, code)
}

func (t *testAccountHandler) RemoveAccountContractCode(location common.AddressLocation) error {
	if t.removeAccountContractCode == nil {
		panic(errors.NewUnexpectedError("unexpected call to RemoveAccountContractCode"))
	}
	return t.removeAccountContractCode(location)
}

func (t *testAccountHandler) RecordContractRemoval(location common.AddressLocation) {
	if t.recordContractRemoval == nil {
		panic(errors.NewUnexpectedError("unexpected call to RecordContractRemoval"))
	}
	t.recordContractRemoval(location)
}

func (t *testAccountHandler) GetAccountContractNames(address common.Address) ([]string, error) {
	if t.getAccountContractNames == nil {
		panic(errors.NewUnexpectedError("unexpected call to GetAccountContractNames"))
	}
	return t.getAccountContractNames(address)
}

func (t *testAccountHandler) StartContractAddition(common.AddressLocation) {
	// NO-OP
}

func (t *testAccountHandler) EndContractAddition(common.AddressLocation) {
	// NO-OP
}

func (t *testAccountHandler) IsContractBeingAdded(common.AddressLocation) bool {
	// NO-OP
	return false
}

type NoOpReferenceCreationContext struct{}

var _ interpreter.ReferenceCreationContext = NoOpReferenceCreationContext{}

func (n NoOpReferenceCreationContext) InvalidateReferencedResources(v interpreter.Value, locationRange interpreter.LocationRange) {
	// NO-OP
}

func (n NoOpReferenceCreationContext) CheckInvalidatedResourceOrResourceReference(value interpreter.Value, locationRange interpreter.LocationRange) {
	// NO-OP
}

func (n NoOpReferenceCreationContext) MaybeTrackReferencedResourceKindedValue(ref *interpreter.EphemeralReferenceValue) {
	// NO-OP
}

func (n NoOpReferenceCreationContext) MeterMemory(usage common.MemoryUsage) error {
	// NO-OP
	return nil
}

type NoOpFunctionCreationContext struct {
	NoOpReferenceCreationContext
}

func (n NoOpFunctionCreationContext) ReadStored(storageAddress common.Address, domain common.StorageDomain, identifier interpreter.StorageMapKey) interpreter.Value {
	// NO-OP
	return nil
}

func (n NoOpFunctionCreationContext) GetEntitlementType(typeID interpreter.TypeID) (*sema.EntitlementType, error) {
	// NO-OP
	return nil, nil
}

func (n NoOpFunctionCreationContext) GetEntitlementMapType(typeID interpreter.TypeID) (*sema.EntitlementMapType, error) {
	// NO-OP
	return nil, nil
}

func (n NoOpFunctionCreationContext) GetInterfaceType(location common.Location, qualifiedIdentifier string, typeID interpreter.TypeID) (*sema.InterfaceType, error) {
	// NO-OP
	return nil, nil
}

func (n NoOpFunctionCreationContext) GetCompositeType(location common.Location, qualifiedIdentifier string, typeID interpreter.TypeID) (*sema.CompositeType, error) {
	// NO-OP
	return nil, nil
}

func (n NoOpFunctionCreationContext) IsTypeInfoRecovered(location common.Location) bool {
	// NO-OP
	return false
}

func (n NoOpFunctionCreationContext) GetCompositeValueFunctions(v *interpreter.CompositeValue, locationRange interpreter.LocationRange) *interpreter.FunctionOrderedMap {
	// NO-OP
	return nil
}

var _ interpreter.FunctionCreationContext = NoOpFunctionCreationContext{}

func testAccountWithErrorHandler(
	t *testing.T,
	address interpreter.AddressValue,
	auth bool,
	handler stdlib.AccountHandler,
	code string,
	checkerConfig sema.Config,
	checkerErrorHandler func(error),
) (*interpreter.Interpreter, func() map[storageKey]interpreter.Value) {

	account := stdlib.NewAccountValue(nil, handler, address)

	var valueDeclarations []stdlib.StandardLibraryValue

	// `authAccount`

	authAccountValueDeclaration := stdlib.StandardLibraryValue{
		Name: "authAccount",
		Type: sema.FullyEntitledAccountReferenceType,
		Value: interpreter.NewEphemeralReferenceValue(
			NoOpReferenceCreationContext{},
			interpreter.FullyEntitledAccountAccess,
			account,
			sema.AccountType,
			interpreter.EmptyLocationRange,
		),
		Kind: common.DeclarationKindConstant,
	}
	valueDeclarations = append(valueDeclarations, authAccountValueDeclaration)

	// `pubAccount`

	pubAccountValueDeclaration := stdlib.StandardLibraryValue{
		Name: "pubAccount",
		Type: sema.AccountReferenceType,
		Value: interpreter.NewEphemeralReferenceValue(
			NoOpReferenceCreationContext{},
			interpreter.UnauthorizedAccess,
			account,
			sema.AccountType,
			interpreter.EmptyLocationRange,
		),
		Kind: common.DeclarationKindConstant,
	}
	valueDeclarations = append(valueDeclarations, pubAccountValueDeclaration)

	// `account`

	var accountValueDeclaration stdlib.StandardLibraryValue

	if auth {
		accountValueDeclaration = authAccountValueDeclaration
	} else {
		accountValueDeclaration = pubAccountValueDeclaration
	}
	accountValueDeclaration.Name = "account"
	valueDeclarations = append(valueDeclarations, accountValueDeclaration)

	valueDeclarations = append(valueDeclarations, stdlib.InclusiveRangeConstructorFunction)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range valueDeclarations {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	require.Nil(t, checkerConfig.BaseValueActivationHandler)
	checkerConfig.BaseValueActivationHandler = func(_ common.Location) *sema.VariableActivation {
		return baseValueActivation
	}

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	for _, valueDeclaration := range valueDeclarations {
		interpreter.Declare(baseActivation, valueDeclaration)
	}

	inter, err := parseCheckAndInterpretWithOptions(t,
		code,
		ParseCheckAndInterpretOptions{
			CheckerConfig: &checkerConfig,
			Config: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				AccountHandler: func(context interpreter.FunctionCreationContext, address interpreter.AddressValue) interpreter.Value {
					return stdlib.NewAccountValue(context, nil, address)
				},
			},
			HandleCheckerError: checkerErrorHandler,
		},
	)
	require.NoError(t, err)

	getAccountValues := func() map[storageKey]interpreter.Value {
		accountValues := make(map[storageKey]interpreter.Value)

		for storageMapKey, accountStorage := range inter.Storage().(interpreter.InMemoryStorage).DomainStorageMaps {
			iterator := accountStorage.Iterator(inter)
			for {
				key, value := iterator.Next()
				if key == nil {
					break
				}
				storageKey := storageKey{
					address: storageMapKey.Address,
					domain:  storageMapKey.Domain.Identifier(),
					key:     key,
				}
				accountValues[storageKey] = value
			}
		}

		return accountValues
	}
	return inter, getAccountValues
}

func TestInterpretAccountStorageSave(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, `
              resource R {}

              fun test() {
                  let r <- create R()
                  account.storage.save(<-r, to: /storage/r)
              }
            `, sema.Config{})

		// Save first value

		t.Run("initial save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			accountValues := getAccountValues()
			require.Len(t, accountValues, 1)
			for _, value := range accountValues {
				assert.IsType(t, &interpreter.CompositeValue{}, value)
			}
		})

		// Attempt to save again, overwriting should fail

		t.Run("second save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.OverwriteError{})
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, `
              struct S {}

              fun test() {
                  let s = S()
                  account.storage.save(s, to: /storage/s)
              }
            `, sema.Config{})

		// Save first value

		t.Run("initial save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			require.NoError(t, err)

			accountValues := getAccountValues()
			require.Len(t, accountValues, 1)
			for _, value := range accountValues {
				assert.IsType(t, &interpreter.CompositeValue{}, value)
			}

		})

		// Attempt to save again, overwriting should fail

		t.Run("second save", func(t *testing.T) {

			_, err := inter.Invoke("test")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.OverwriteError{})
		})
	})
}

func TestInterpretAccountStorageType(t *testing.T) {

	t.Parallel()

	t.Run("type", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountStorables := testAccount(t, address, true, nil, `
              struct S {}

              resource R {}

              fun saveR() {
                let r <- create R()
                account.storage.save(<-r, to: /storage/x)
              }

              fun saveS() {
                let s = S()
                destroy account.storage.load<@R>(from: /storage/x)
                account.storage.save(s, to: /storage/x)
              }

              fun typeAt(): AnyStruct {
                return account.storage.type(at: /storage/x)
              }
            `, sema.Config{})

		// type empty path is nil

		value, err := inter.Invoke("typeAt")
		require.NoError(t, err)
		require.Len(t, getAccountStorables(), 0)
		require.Equal(t, interpreter.Nil, value)

		// save R

		_, err = inter.Invoke("saveR")
		require.NoError(t, err)
		require.Len(t, getAccountStorables(), 1)

		// type is now type of R

		value, err = inter.Invoke("typeAt")
		require.NoError(t, err)
		require.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
				},
			),
			value,
		)

		// save S

		_, err = inter.Invoke("saveS")
		require.NoError(t, err)
		require.Len(t, getAccountStorables(), 1)

		// type is now type of S

		value, err = inter.Invoke("typeAt")
		require.NoError(t, err)
		require.Equal(t,
			interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.TypeValue{
					Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
				},
			),
			value,
		)
	})
}

func TestInterpretAccountStorageLoad(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, `
              resource R {}

              resource R2 {}

              fun save() {
                  let r <- create R()
                  account.storage.save(<-r, to: /storage/r)
              }

              fun loadR(): @R? {
                  return <-account.storage.load<@R>(from: /storage/r)
              }

              fun loadR2(): @R2? {
                  return <-account.storage.load<@R2>(from: /storage/r)
              }
            `, sema.Config{})

		t.Run("save R and load R ", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// first load

			value, err := inter.Invoke("loadR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue()

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was removed from storage
			require.Len(t, getAccountValues(), 0)

			// second load

			value, err = inter.Invoke("loadR")
			require.NoError(t, err)

			require.IsType(t, interpreter.Nil, value)
		})

		t.Run("save R and load R2", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// load

			_, err = inter.Invoke("loadR2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, `
              struct S {}

              struct S2 {}

              fun save() {
                  let s = S()
                  account.storage.save(s, to: /storage/s)
              }

              fun loadS(): S? {
                  return account.storage.load<S>(from: /storage/s)
              }

              fun loadS2(): S2? {
                  return account.storage.load<S2>(from: /storage/s)
              }
            `, sema.Config{})

		t.Run("save S and load S", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// first load

			value, err := inter.Invoke("loadS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue()

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was removed from storage
			require.Len(t, getAccountValues(), 0)

			// second load

			value, err = inter.Invoke("loadS")
			require.NoError(t, err)

			require.IsType(t, interpreter.Nil, value)
		})

		t.Run("save S and load S2", func(t *testing.T) {

			// save

			_, err := inter.Invoke("save")
			require.NoError(t, err)

			require.Len(t, getAccountValues(), 1)

			// load

			_, err = inter.Invoke("loadS2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})
	})
}

func TestInterpretAccountStorageCopy(t *testing.T) {

	t.Parallel()

	const code = `
      struct S {}

      struct S2 {}

      fun save() {
          let s = S()
          account.storage.save(s, to: /storage/s)
      }

      fun copyS(): S? {
          return account.storage.copy<S>(from: /storage/s)
      }

      fun copyS2(): S2? {
          return account.storage.copy<S2>(from: /storage/s)
      }
    `

	t.Run("save S and copy S ", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, code, sema.Config{})

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		testCopyS := func() {

			value, err := inter.Invoke("copyS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue()

			assert.IsType(t, &interpreter.CompositeValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		}

		testCopyS()

		testCopyS()
	})

	t.Run("save S and copy S2", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, code, sema.Config{})

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		// load

		_, err = inter.Invoke("copyS2")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

		// NOTE: check loaded value was *not* removed from storage
		require.Len(t, getAccountValues(), 1)
	})
}

func TestInterpretAccountStorageBorrow(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, `
              resource R {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              resource R2 {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun save() {
                  let r <- create R()
                  account.storage.save(<-r, to: /storage/r)
              }

			  fun checkR(): Bool {
				  return account.storage.check<@R>(from: /storage/r)
			  }

              fun borrowR(): &R? {
                  return account.storage.borrow<&R>(from: /storage/r)
              }

              fun foo(): Int {
                  return account.storage.borrow<&R>(from: /storage/r)!.foo
              }

			  fun checkR2(): Bool {
				  return account.storage.check<@R2>(from: /storage/r)
			  }

              fun borrowR2(): &R2? {
                  return account.storage.borrow<&R2>(from: /storage/r)
              }

			  fun checkR2WithInvalidPath(): Bool {
				  return account.storage.check<@R2>(from: /storage/wrongpath)
			  }

              fun changeAfterBorrow(): Int {
                 let ref = account.storage.borrow<&R>(from: /storage/r)!

                 let r <- account.storage.load<@R>(from: /storage/r)
                 destroy r

                 let r2 <- create R2()
                 account.storage.save(<-r2, to: /storage/r)

                 return ref.foo
              }
            `, sema.Config{})

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		t.Run("borrow R ", func(t *testing.T) {

			// first check & borrow
			checkRes, err := inter.Invoke("checkR")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				checkRes,
			)

			value, err := inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue()

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// foo

			value, err = inter.Invoke("foo")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// TODO: should fail, i.e. return nil

			// second check & borrow
			checkRes, err = inter.Invoke("checkR")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				checkRes,
			)

			value, err = inter.Invoke("borrowR")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue = value.(*interpreter.SomeValue).InnerValue()

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("borrow R2", func(t *testing.T) {
			checkRes, err := inter.Invoke("checkR2")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				checkRes,
			)

			_, err = inter.Invoke("borrowR2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("changeAfterBorrow")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})

		t.Run("check R2 with wrong path", func(t *testing.T) {
			checkRes, err := inter.Invoke("checkR2WithInvalidPath")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				checkRes,
			)
		})
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

		inter, getAccountValues := testAccount(t, address, true, nil, `
              struct S {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              struct S2 {
                  let foo: Int

                  init() {
                      self.foo = 42
                  }
              }

              fun save() {
                  let s = S()
                  account.storage.save(s, to: /storage/s)
              }

			  fun checkS(): Bool {
				  return account.storage.check<S>(from: /storage/s)
			  }

              fun borrowS(): &S? {
                  return account.storage.borrow<&S>(from: /storage/s)
              }

              fun foo(): Int {
                  return account.storage.borrow<&S>(from: /storage/s)!.foo
              }
			 
			  fun checkS2(): Bool {
				  return account.storage.check<S2>(from: /storage/s)
			  }
             
			  fun borrowS2(): &S2? {
                  return account.storage.borrow<&S2>(from: /storage/s)
              }

              fun changeAfterBorrow(): Int {
                 let ref = account.storage.borrow<&S>(from: /storage/s)!

                 // remove stored value
                 account.storage.load<S>(from: /storage/s)

                 let s2 = S2()
                 account.storage.save(s2, to: /storage/s)

                 return ref.foo
              }

              fun invalidBorrowS(): &S2? {
                  let s = S()
                  account.storage.save(s, to: /storage/another_s)
                  let borrowedS = account.storage.borrow<&AnyStruct>(from: /storage/another_s)
                  return borrowedS as! &S2?
              }
            `, sema.Config{})

		// save

		_, err := inter.Invoke("save")
		require.NoError(t, err)

		require.Len(t, getAccountValues(), 1)

		t.Run("borrow S", func(t *testing.T) {

			// first check & borrow
			checkRes, err := inter.Invoke("checkS")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				checkRes,
			)

			value, err := inter.Invoke("borrowS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue := value.(*interpreter.SomeValue).InnerValue()

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// foo

			value, err = inter.Invoke("foo")
			require.NoError(t, err)

			RequireValuesEqual(
				t,
				inter,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				value,
			)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)

			// TODO: should fail, i.e. return nil

			// second check & borrow
			checkRes, err = inter.Invoke("checkS")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				checkRes,
			)

			value, err = inter.Invoke("borrowS")
			require.NoError(t, err)

			require.IsType(t, &interpreter.SomeValue{}, value)

			innerValue = value.(*interpreter.SomeValue).InnerValue()

			assert.IsType(t, &interpreter.StorageReferenceValue{}, innerValue)

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("borrow S2", func(t *testing.T) {
			checkRes, err := inter.Invoke("checkS2")
			require.NoError(t, err)
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				checkRes,
			)

			_, err = inter.Invoke("borrowS2")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

			// NOTE: check loaded value was *not* removed from storage
			require.Len(t, getAccountValues(), 1)
		})

		t.Run("change after borrow", func(t *testing.T) {

			_, err := inter.Invoke("changeAfterBorrow")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.DereferenceError{})
		})

		t.Run("borrow as invalid type", func(t *testing.T) {
			_, err = inter.Invoke("invalidBorrowS")
			RequireError(t, err)

			require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
		})
	})
}

func TestInterpretAccountBalanceFields(t *testing.T) {
	t.Parallel()

	const availableBalance = 42
	const balance = 43

	handler := &testAccountHandler{

		getAccountAvailableBalance: func(_ common.Address) (uint64, error) {
			return availableBalance, nil
		},
		getAccountBalance: func(_ common.Address) (uint64, error) {
			return balance, nil
		},
	}

	for _, auth := range []bool{true, false} {

		for fieldName, expected := range map[string]uint64{
			"balance":          balance,
			"availableBalance": availableBalance,
		} {

			testName := fmt.Sprintf("%s, auth: %v", fieldName, auth)

			t.Run(testName, func(t *testing.T) {

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1})

				code := fmt.Sprintf(
					`
                      fun test(): UFix64 {
                          return account.%s
                      }
                    `,
					fieldName,
				)
				inter, _ := testAccount(
					t,
					address,
					auth,
					handler,
					code,
					sema.Config{},
				)

				value, err := inter.Invoke("test")
				require.NoError(t, err)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredUFix64Value(expected),
					value,
				)
			})
		}
	}
}

func TestInterpretAccountStorageFields(t *testing.T) {
	t.Parallel()

	const storageUsed = 42
	const storageCapacity = 43

	handler := &testAccountHandler{
		commitStorageTemporarily: func(_ interpreter.ValueTransferContext) error {
			return nil
		},
		getStorageUsed: func(_ common.Address) (uint64, error) {
			return storageUsed, nil
		},
		getStorageCapacity: func(address common.Address) (uint64, error) {
			return storageCapacity, nil
		},
	}

	for _, auth := range []bool{true, false} {

		for fieldName, expected := range map[string]uint64{
			"used":     storageUsed,
			"capacity": storageCapacity,
		} {

			testName := fmt.Sprintf("%s, auth: %v", fieldName, auth)

			t.Run(testName, func(t *testing.T) {

				code := fmt.Sprintf(
					`
                      fun test(): UInt64 {
                          return account.storage.%s
                      }
                    `,
					fieldName,
				)

				address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1})

				inter, _ := testAccount(t, address, auth, handler, code, sema.Config{})

				value, err := inter.Invoke("test")
				require.NoError(t, err)

				AssertValuesEqual(
					t,
					inter,
					interpreter.NewUnmeteredUInt64Value(expected),
					value,
				)
			})
		}
	}
}
