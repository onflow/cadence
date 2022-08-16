/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package stdlib

import (
	"fmt"
	"strconv"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/contracts"
)

// This is the Cadence standard library for writing tests.
// It provides the Cadence constructs (structs, functions, etc.) that are needed to
// write tests in Cadence.

const testContractTypeName = "Test"
const blockchainTypeName = "Blockchain"
const blockchainBackendTypeName = "BlockchainBackend"
const scriptResultTypeName = "ScriptResult"
const transactionResultTypeName = "TransactionResult"
const resultStatusTypeName = "ResultStatus"
const accountTypeName = "Account"

const succeededCaseName = "succeeded"
const failedCaseName = "failed"

const transactionCodeFieldName = "code"
const transactionAuthorizerFieldName = "authorizers"
const transactionSignersFieldName = "signers"
const transactionArgsFieldName = "arguments"

const accountAddressFieldName = "address"

var TestContractLocation = common.IdentifierLocation(testContractTypeName)

var TestContractChecker = func() *sema.Checker {

	program, err := parser.ParseProgram(contracts.TestContract, nil)
	if err != nil {
		panic(err)
	}

	var checker *sema.Checker
	checker, err = sema.NewChecker(
		program,
		TestContractLocation,
		nil,
		false,
		sema.WithPredeclaredValues(BuiltinFunctions.ToSemaValueDeclarations()),
		sema.WithPredeclaredTypes(BuiltinTypes.ToTypeDeclarations()),
	)
	if err != nil {
		panic(err)
	}

	err = checker.Check()
	if err != nil {
		panic(err)
	}

	return checker
}()

func NewTestContract(
	inter *interpreter.Interpreter,
	constructor interpreter.FunctionValue,
	invocationRange ast.Range,
) (
	*interpreter.CompositeValue,
	error,
) {
	value, err := inter.InvokeFunctionValue(
		constructor,
		[]interpreter.Value{},
		testContractInitializerTypes,
		testContractInitializerTypes,
		invocationRange,
	)
	if err != nil {
		return nil, err
	}

	compositeValue := value.(*interpreter.CompositeValue)

	// Inject natively implemented function values
	compositeValue.Functions[testAssertFunctionName] = testAssertFunction
	compositeValue.Functions[testNewEmulatorBlockchainFunctionName] = testNewEmulatorBlockchainFunction

	return compositeValue, nil
}

var testContractType = func() *sema.CompositeType {
	variable, ok := TestContractChecker.Elaboration.GlobalTypes.Get(testContractTypeName)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return variable.Type.(*sema.CompositeType)
}()

var testContractInitializerTypes = func() (result []sema.Type) {
	result = make([]sema.Type, len(testContractType.ConstructorParameters))
	for i, parameter := range testContractType.ConstructorParameters {
		result[i] = parameter.TypeAnnotation.Type
	}
	return result
}()

var blockchainBackendInterfaceType = func() *sema.InterfaceType {
	typ, ok := testContractType.NestedTypes.Get(blockchainBackendTypeName)
	if !ok {
		panic(errors.NewUnexpectedError("cannot find type %s.%s", testContractTypeName, blockchainBackendTypeName))
	}

	interfaceType, ok := typ.(*sema.InterfaceType)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for %s. expected interface", blockchainBackendTypeName))
	}

	return interfaceType
}()

func init() {

	// Enrich 'Test' contract with natively implemented functions
	testContractType.Members.Set(
		testAssertFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testAssertFunctionName,
			testAssertFunctionType,
			testAssertFunctionDocString,
		),
	)
	testContractType.Members.Set(
		testNewEmulatorBlockchainFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testNewEmulatorBlockchainFunctionName,
			testNewEmulatorBlockchainFunctionType,
			testNewEmulatorBlockchainFunctionDocString,
		),
	)

	// Enrich 'Test' contract elaboration with natively implemented composite types.
	// e.g: 'EmulatorBackend' type.
	TestContractChecker.Elaboration.CompositeTypes[EmulatorBackendType.ID()] = EmulatorBackendType
}

var blockchainType = func() sema.Type {
	typ, ok := testContractType.NestedTypes.Get(blockchainTypeName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			testContractTypeName,
			blockchainTypeName,
		))
	}

	return typ
}()

// Functions belong to the 'Test' contract

// 'Test.assert' function

const testAssertFunctionDocString = `assert function of Test contract`

const testAssertFunctionName = "assert"

var testAssertFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "condition",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.BoolType,
			),
		},
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "message",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.StringType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

var testAssertFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		condition, ok := invocation.Arguments[0].(interpreter.BoolValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		message, ok := invocation.Arguments[1].(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		if !condition {
			panic(AssertionError{
				Message: message.String(),
			})
		}

		return interpreter.VoidValue{}
	},
	testAssertFunctionType,
)

// 'Test.newEmulatorBlockchain' function

const testNewEmulatorBlockchainFunctionDocString = `newEmulatorBlockchain function of Test contract`

const testNewEmulatorBlockchainFunctionName = "newEmulatorBlockchain"

var testNewEmulatorBlockchainFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		blockchainType,
	),
}

var testNewEmulatorBlockchainFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {

		// Create an `EmulatorBackend`
		emulatorBackend := newEmulatorBackend(invocation.Interpreter)

		// Create a 'Blockchain' struct value, that wraps the emulator backend,
		// by calling the constructor of 'Blockchain'.

		testContract := invocation.Self.(*interpreter.CompositeValue)
		blockchainConstructorVar := testContract.NestedVariables[blockchainTypeName]
		blockchainConstructor, ok := blockchainConstructorVar.GetValue().(*interpreter.HostFunctionValue)
		if !ok {
			panic(errors.NewUnexpectedError("invalid type for constructor"))
		}

		blockchain, err := invocation.Interpreter.InvokeExternally(
			blockchainConstructor,
			blockchainConstructor.Type,
			[]interpreter.Value{
				emulatorBackend,
			},
		)

		if err != nil {
			panic(err)
		}

		return blockchain
	},
	testNewEmulatorBlockchainFunctionType,
)

// 'EmulatorBackend' struct.
//
// 'EmulatorBackend' is the native implementation of the 'Test.BlockchainBackend' interface.
// It provides a blockchain backed by the emulator.

const emulatorBackendTypeName = "EmulatorBackend"

var EmulatorBackendType = func() *sema.CompositeType {

	ty := &sema.CompositeType{
		Identifier: emulatorBackendTypeName,
		Kind:       common.CompositeKindStructure,
		Location:   TestContractLocation,
		ExplicitInterfaceConformances: []*sema.InterfaceType{
			blockchainBackendInterfaceType,
		},
	}

	var members = []*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			emulatorBackendExecuteScriptFunctionName,
			emulatorBackendExecuteScriptFunctionType,
			emulatorBackendExecuteScriptFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			emulatorBackendCreateAccountFunctionName,
			emulatorBackendCreateAccountFunctionType,
			emulatorBackendCreateAccountFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			emulatorBackendAddTransactionFunctionName,
			emulatorBackendAddTransactionFunctionType,
			emulatorBackendAddTransactionFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			emulatorBackendExecuteNextTransactionFunctionName,
			emulatorBackendExecuteNextTransactionFunctionType,
			emulatorBackendExecuteNextTransactionFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			emulatorBackendCommitBlockFunctionName,
			emulatorBackendCommitBlockFunctionType,
			emulatorBackendCommitBlockFunctionDocString,
		),
	}

	ty.Members = sema.GetMembersAsMap(members)
	ty.Fields = sema.GetFieldNames(members)

	return ty
}()

func newEmulatorBackend(inter *interpreter.Interpreter) *interpreter.CompositeValue {
	var fields = []interpreter.CompositeField{
		{
			Name:  emulatorBackendExecuteScriptFunctionName,
			Value: emulatorBackendExecuteScriptFunction,
		},
		{
			Name:  emulatorBackendCreateAccountFunctionName,
			Value: emulatorBackendCreateAccountFunction,
		}, {
			Name:  emulatorBackendAddTransactionFunctionName,
			Value: emulatorBackendAddTransactionFunction,
		},
		{
			Name:  emulatorBackendExecuteNextTransactionFunctionName,
			Value: emulatorBackendExecuteNextTransactionFunction,
		},
		{
			Name:  emulatorBackendCommitBlockFunctionName,
			Value: emulatorBackendCommitBlockFunction,
		},
	}

	return interpreter.NewCompositeValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		EmulatorBackendType.Location,
		emulatorBackendTypeName,
		common.CompositeKindStructure,
		fields,
		common.Address{},
	)
}

// 'EmulatorBackend.executeScript' function

const emulatorBackendExecuteScriptFunctionName = "executeScript"

const emulatorBackendExecuteScriptFunctionDocString = `execute script function`

var emulatorBackendExecuteScriptFunctionType = func() *sema.FunctionType {
	// The type of the 'executeScript' function of 'EmulatorBackend' (interface-implementation)
	// is same as that of 'BlockchainBackend' interface.
	typ, ok := blockchainBackendInterfaceType.Members.Get(emulatorBackendExecuteScriptFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			blockchainBackendTypeName,
			emulatorBackendExecuteScriptFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			emulatorBackendExecuteScriptFunctionName,
		))
	}

	return functionType
}()

var emulatorBackendExecuteScriptFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		testFramework := invocation.Interpreter.TestFramework
		if testFramework == nil {
			panic(interpreter.TestFrameworkNotProvidedError{})
		}

		scriptString, ok := invocation.Arguments[0].(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// String conversion of the value gives the quoted string.
		// Unquote the script-string to remove starting/ending quotes
		// and to unescape the string literals in the code.
		//
		// TODO: Is the reverse conversion loss-less?

		script, err := strconv.Unquote(scriptString.String())
		if err != nil {
			panic(errors.NewUnexpectedErrorFromCause(err))
		}

		args, err := arrayValueToSlice(invocation.Arguments[1])
		if err != nil {
			panic(errors.NewUnexpectedErrorFromCause(err))
		}

		result := testFramework.RunScript(script, args)

		succeeded := result.Error == nil

		return createScriptResult(invocation.Interpreter, result.Value, succeeded)
	},
	emulatorBackendExecuteScriptFunctionType,
)

func arrayValueToSlice(value interpreter.Value) ([]interpreter.Value, error) {
	array, ok := value.(*interpreter.ArrayValue)
	if !ok {
		return nil, errors.NewDefaultUserError("value is not an array")
	}

	result := make([]interpreter.Value, 0, array.Count())

	array.Iterate(nil, func(element interpreter.Value) (resume bool) {
		result = append(result, element)
		return true
	})

	return result, nil
}

// createScriptResult Creates a "ScriptResult" using the return value of the executed script.
//
func createScriptResult(inter *interpreter.Interpreter, returnValue interpreter.Value, succeeded bool) interpreter.Value {
	// Lookup and get 'ResultStatus' enum value.

	resultStatusConstructorVar := inter.Activations.Find(resultStatusTypeName)
	resultStatusConstructor, ok := resultStatusConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}

	var status interpreter.Value
	if succeeded {
		succeededVar := resultStatusConstructor.NestedVariables[succeededCaseName]
		status = succeededVar.GetValue()
	} else {
		succeededVar := resultStatusConstructor.NestedVariables[failedCaseName]
		status = succeededVar.GetValue()
	}

	// Create a 'ScriptResult' by calling its constructor.

	scriptResultConstructorVar := inter.Activations.Find(scriptResultTypeName)
	scriptResultConstructor, ok := scriptResultConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}

	scriptResult, err := inter.InvokeExternally(
		scriptResultConstructor,
		scriptResultConstructor.Type,
		[]interpreter.Value{
			status,
			returnValue,
		},
	)

	if err != nil {
		panic(err)
	}

	return scriptResult
}

// 'EmulatorBackend.createAccount' function

const emulatorBackendCreateAccountFunctionName = "createAccount"

const emulatorBackendCreateAccountFunctionDocString = `create account function`

var emulatorBackendCreateAccountFunctionType = func() *sema.FunctionType {
	// The type of the 'createAccount' function of 'EmulatorBackend' (interface-implementation)
	// is same as that of 'BlockchainBackend' interface.
	typ, ok := blockchainBackendInterfaceType.Members.Get(emulatorBackendCreateAccountFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			blockchainBackendTypeName,
			emulatorBackendCreateAccountFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			emulatorBackendCreateAccountFunctionName,
		))
	}

	return functionType
}()

var emulatorBackendCreateAccountFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		testFramework := invocation.Interpreter.TestFramework
		if testFramework == nil {
			panic(interpreter.TestFrameworkNotProvidedError{})
		}

		account, err := testFramework.CreateAccount()
		if err != nil {
			panic(err)
		}

		return newAccountValue(invocation.Interpreter, account)
	},
	emulatorBackendCreateAccountFunctionType,
)

func newAccountValue(inter *interpreter.Interpreter, account *interpreter.Account) interpreter.Value {

	// Create address value
	address := interpreter.NewAddressValue(nil, account.Address)

	// Create public key
	publicKey := interpreter.NewPublicKeyValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.ByteSliceToByteArrayValue(
			inter,
			account.PublicKey.PublicKey,
		),
		NewSignatureAlgorithmCase(
			inter,
			account.PublicKey.SignAlgo.RawValue(),
		),
		inter.PublicKeyValidationHandler,
	)

	// Create an 'Account' by calling its constructor.
	accountConstructorVar := inter.Activations.Find(accountTypeName)
	accountConstructor, ok := accountConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}

	accountValue, err := inter.InvokeExternally(
		accountConstructor,
		accountConstructor.Type,
		[]interpreter.Value{
			address,
			publicKey,
		},
	)

	if err != nil {
		panic(err)
	}

	return accountValue
}

// 'EmulatorBackend.addTransaction' function

const emulatorBackendAddTransactionFunctionName = "addTransaction"

const emulatorBackendAddTransactionFunctionDocString = `add transaction function`

var emulatorBackendAddTransactionFunctionType = func() *sema.FunctionType {
	// The type of the 'addTransaction' function of 'EmulatorBackend' (interface-implementation)
	// is same as that of 'BlockchainBackend' interface.
	typ, ok := blockchainBackendInterfaceType.Members.Get(emulatorBackendAddTransactionFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			blockchainBackendTypeName,
			emulatorBackendAddTransactionFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			emulatorBackendAddTransactionFunctionName,
		))
	}

	return functionType
}()

var emulatorBackendAddTransactionFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		testFramework := invocation.Interpreter.TestFramework
		if testFramework == nil {
			panic(interpreter.TestFrameworkNotProvidedError{})
		}

		inter := invocation.Interpreter

		transactionValue, ok := invocation.Arguments[0].(interpreter.MemberAccessibleValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Get transaction code
		codeValue := transactionValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			transactionCodeFieldName,
		)
		code, ok := codeValue.(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Get authorizers
		authorizerValue := transactionValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			transactionAuthorizerFieldName,
		)

		authorizers := addressesFromValue(authorizerValue)

		// Get signers
		signersValue := transactionValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			transactionSignersFieldName,
		)

		signerAccounts := accountsFromValue(inter, signersValue)

		// Get arguments
		argsValue := transactionValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			transactionArgsFieldName,
		)
		args, err := arrayValueToSlice(argsValue)
		if err != nil {
			panic(errors.NewUnexpectedErrorFromCause(err))
		}

		err = testFramework.AddTransaction(code.Str, authorizers, signerAccounts, args)
		if err != nil {
			panic(err)
		}

		return interpreter.VoidValue{}
	},
	emulatorBackendAddTransactionFunctionType,
)

func addressesFromValue(accountsValue interpreter.Value) []common.Address {
	accountsArray, ok := accountsValue.(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	addresses := make([]common.Address, 0)

	accountsArray.Iterate(nil, func(element interpreter.Value) (resume bool) {
		address, ok := element.(interpreter.AddressValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		addresses = append(addresses, common.Address(address))

		return true
	})

	return addresses
}

func accountsFromValue(inter *interpreter.Interpreter, accountsValue interpreter.Value) []*interpreter.Account {
	accountsArray, ok := accountsValue.(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	accounts := make([]*interpreter.Account, 0)

	accountsArray.Iterate(nil, func(element interpreter.Value) (resume bool) {
		accountValue, ok := element.(interpreter.MemberAccessibleValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Get address
		addressValue := accountValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			accountAddressFieldName,
		)
		address, ok := addressValue.(interpreter.AddressValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Get public key
		publicKeyVal, ok := accountValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			sema.AccountKeyPublicKeyField,
		).(interpreter.MemberAccessibleValue)

		if !ok {
			panic(errors.NewUnreachableError())
		}

		publicKey, err := NewPublicKeyFromValue(inter, interpreter.ReturnEmptyLocationRange, publicKeyVal)
		if err != nil {
			panic(err)
		}

		accounts = append(accounts, &interpreter.Account{
			Address:   common.Address(address),
			PublicKey: publicKey,
		})

		return true
	})

	return accounts
}

// 'EmulatorBackend.executeNextTransaction' function

const emulatorBackendExecuteNextTransactionFunctionName = "executeNextTransaction"

const emulatorBackendExecuteNextTransactionFunctionDocString = `execute next transaction function`

var emulatorBackendExecuteNextTransactionFunctionType = func() *sema.FunctionType {
	// The type of the 'executeNextTransaction' function of 'EmulatorBackend' (interface-implementation)
	// is same as that of 'BlockchainBackend' interface.
	typ, ok := blockchainBackendInterfaceType.Members.Get(emulatorBackendExecuteNextTransactionFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			blockchainBackendTypeName,
			emulatorBackendExecuteNextTransactionFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			emulatorBackendExecuteNextTransactionFunctionName,
		))
	}

	return functionType
}()

var emulatorBackendExecuteNextTransactionFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		testFramework := invocation.Interpreter.TestFramework
		if testFramework == nil {
			panic(interpreter.TestFrameworkNotProvidedError{})
		}

		result := testFramework.ExecuteNextTransaction()

		// If there are no transactions to run, then return `nil`.
		if result == nil {
			return interpreter.NilValue{}
		}

		succeeded := result.Error == nil

		return createTransactionResult(invocation.Interpreter, succeeded)
	},
	emulatorBackendExecuteNextTransactionFunctionType,
)

// createTransactionResult Creates a "TransactionResult" indicating the status of the transaction execution.
//
func createTransactionResult(inter *interpreter.Interpreter, succeeded bool) interpreter.Value {
	// Lookup and get 'ResultStatus' enum value.
	resultStatusConstructorVar := inter.Activations.Find(resultStatusTypeName)
	resultStatusConstructor, ok := resultStatusConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}

	var status interpreter.Value
	if succeeded {
		succeededVar := resultStatusConstructor.NestedVariables[succeededCaseName]
		status = succeededVar.GetValue()
	} else {
		succeededVar := resultStatusConstructor.NestedVariables[failedCaseName]
		status = succeededVar.GetValue()
	}

	// Create a 'TransactionResult' by calling its constructor.
	transactionResultConstructorVar := inter.Activations.Find(transactionResultTypeName)
	transactionResultConstructor, ok := transactionResultConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}

	transactionResult, err := inter.InvokeExternally(
		transactionResultConstructor,
		transactionResultConstructor.Type,
		[]interpreter.Value{
			status,
		},
	)

	if err != nil {
		panic(err)
	}

	return transactionResult
}

// 'EmulatorBackend.commitBlock' function

const emulatorBackendCommitBlockFunctionName = "commitBlock"

const emulatorBackendCommitBlockFunctionDocString = `commit block function`

var emulatorBackendCommitBlockFunctionType = func() *sema.FunctionType {
	// The type of the 'commitBlock' function of 'EmulatorBackend' (interface-implementation)
	// is same as that of 'BlockchainBackend' interface.
	typ, ok := blockchainBackendInterfaceType.Members.Get(emulatorBackendCommitBlockFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			blockchainBackendTypeName,
			emulatorBackendCommitBlockFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			emulatorBackendCommitBlockFunctionName,
		))
	}

	return functionType
}()

var emulatorBackendCommitBlockFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		testFramework := invocation.Interpreter.TestFramework
		if testFramework == nil {
			panic(interpreter.TestFrameworkNotProvidedError{})
		}

		err := testFramework.CommitBlock()
		if err != nil {
			panic(err)
		}

		return interpreter.VoidValue{}
	},
	emulatorBackendCommitBlockFunctionType,
)

// TestFailedError

type TestFailedError struct {
	Err error
}

var _ errors.UserError = TestFailedError{}

func (TestFailedError) IsUserError() {}

func (e TestFailedError) Unwrap() error {
	return e.Err
}

func (e TestFailedError) Error() string {
	return fmt.Sprintf("test failed: %s", e.Err.Error())
}
