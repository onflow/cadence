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
const matcherTypeName = "Matcher"
const structMatcherTypeName = "StructMatcher"

const succeededCaseName = "succeeded"
const failedCaseName = "failed"

const transactionCodeFieldName = "code"
const transactionAuthorizerFieldName = "authorizer"
const transactionSignersFieldName = "signers"

const accountAddressFieldName = "address"
const accountKeyFieldName = "accountKey"
const accountPrivateKeyFieldName = "privateKey"

const matcherTestFunctionName = "test"

var TestContractLocation = common.IdentifierLocation(testContractTypeName)

var TestContractChecker = func() *sema.Checker {

	program, err := parser.ParseProgram(contracts.TestContract, nil)
	if err != nil {
		panic(err)
	}

	builtinTypes := BuiltinTypes
	builtinTypes = append(builtinTypes, StandardLibraryType{
		Name: sema.AnyType.Name,
		Type: sema.AnyType,
		Kind: common.DeclarationKindType,
	})

	var checker *sema.Checker
	checker, err = sema.NewChecker(
		program,
		TestContractLocation,
		nil,
		false,
		sema.WithPredeclaredValues(BuiltinFunctions.ToSemaValueDeclarations()),
		sema.WithPredeclaredTypes(builtinTypes.ToTypeDeclarations()),
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
	compositeValue.Functions[testExpectFunctionName] = testExpectFunction
	compositeValue.Functions[testNewEmulatorBlockchainFunctionName] = testNewEmulatorBlockchainFunction

	// Inject natively implemented matchers
	compositeValue.Functions[testEqualMatcherFunctionName] = testEqualMatcherFunction

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
		testExpectFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testExpectFunctionName,
			testExpectFunctionType,
			testExpectFunctionDocString,
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

	// Matcher functions
	testContractType.Members.Set(
		testEqualMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testEqualMatcherFunctionName,
			testEqualMatcherFunctionType,
			testEqualMatcherFunctionDocString,
		),
	)

	// Enrich 'Test' contract elaboration with natively implemented composite types.
	// e.g: 'EmulatorBackend' type.
	TestContractChecker.Elaboration.CompositeTypes[EmulatorBackendType.ID()] = EmulatorBackendType
	TestContractChecker.Elaboration.CompositeTypes[resourceMatcherType.ID()] = resourceMatcherType
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

// 'Test.expect' function

const testExpectFunctionDocString = `expect function of Test contract`

const testExpectFunctionName = "expect"

var testExpectFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "value",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.GenericType{
					TypeParameter: typeParameter,
				},
			),
		},
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "matcher",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.RestrictedType{
					Type: sema.AnyStructType,
					Restrictions: []*sema.InterfaceType{
						matcherType,
					},
				},
			),
		},
	},
	TypeParameters: []*sema.TypeParameter{
		typeParameter,
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

var testExpectFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		value := invocation.Arguments[0]

		matcher, ok := invocation.Arguments[1].(*interpreter.CompositeValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		result := invokeMatcherTest(invocation.Interpreter, matcher, value)

		if !result {
			panic(AssertionError{})
		}

		return interpreter.VoidValue{}
	},
	testExpectFunctionType,
)

func invokeMatcherTest(
	inter *interpreter.Interpreter,
	matcher interpreter.MemberAccessibleValue,
	value interpreter.Value,
) bool {
	testFunc := matcher.GetMember(
		inter,
		interpreter.ReturnEmptyLocationRange,
		matcherTestFunctionName,
	)

	funcValue, ok := testFunc.(interpreter.FunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			matcherTestFunctionName,
		))
	}

	functionType := getFunctionType(funcValue)

	testResult, err := inter.InvokeExternally(
		funcValue,
		functionType,
		[]interpreter.Value{
			value,
		},
	)

	if err != nil {
		panic(err)
	}

	result, ok := testResult.(interpreter.BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return bool(result)
}

func getFunctionType(value interpreter.FunctionValue) *sema.FunctionType {
	switch funcValue := value.(type) {
	case *interpreter.InterpretedFunctionValue:
		return funcValue.Type
	case *interpreter.HostFunctionValue:
		return funcValue.Type
	case interpreter.BoundFunctionValue:
		return getFunctionType(funcValue.Function)
	default:
		panic(errors.NewUnreachableError())
	}
}

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

		testContract, ok := invocation.Self.(*interpreter.CompositeValue)
		if !ok {
			panic(errors.NewUnexpectedError("invalid type for %s contract", testContractTypeName))
		}

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

		result := testFramework.RunScript(script)

		succeeded := result.Error == nil

		return createScriptResult(invocation.Interpreter, result.Value, succeeded)
	},
	emulatorBackendExecuteScriptFunctionType,
)

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

	// Create account key
	accountKey := newAccountKeyValue(inter, account.AccountKey)

	// Create private key
	privateKey := interpreter.ByteSliceToByteArrayValue(inter, account.PrivateKey)

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
			accountKey,
			privateKey,
		},
	)

	if err != nil {
		panic(err)
	}

	return accountValue
}

func newAccountKeyValue(inter *interpreter.Interpreter, accountKey *interpreter.AccountKey) interpreter.Value {
	index := interpreter.NewIntValueFromInt64(nil, int64(accountKey.KeyIndex))

	publicKey := interpreter.NewPublicKeyValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		interpreter.ByteSliceToByteArrayValue(
			inter,
			accountKey.PublicKey.PublicKey,
		),
		NewSignatureAlgorithmCase(
			inter,
			accountKey.PublicKey.SignAlgo.RawValue(),
		),
		inter.PublicKeyValidationHandler,
	)

	hashAlgorithm := NewHashAlgorithmCase(
		inter,
		accountKey.HashAlgo.RawValue(),
	)

	weight := interpreter.NewUnmeteredUFix64ValueWithInteger(uint64(accountKey.Weight))

	revoked := interpreter.BoolValue(accountKey.IsRevoked)

	return interpreter.NewAccountKeyValue(
		inter,
		index,
		publicKey,
		hashAlgorithm,
		weight,
		revoked,
	)
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
		codeString, ok := codeValue.(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		code, err := strconv.Unquote(codeString.String())
		if err != nil {
			panic(errors.NewUnexpectedErrorFromCause(err))
		}

		// Get authorizer
		authorizerValue := transactionValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			transactionAuthorizerFieldName,
		)

		var authorizer *common.Address
		switch authorizerValue := authorizerValue.(type) {
		case interpreter.NilValue:
			authorizer = nil
		case *interpreter.SomeValue:
			authorizerAddress, ok := authorizerValue.InnerValue(inter,
				interpreter.ReturnEmptyLocationRange).(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			authorizer = (*common.Address)(&authorizerAddress)
		}

		// Get signers
		signersValue := transactionValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			transactionSignersFieldName,
		)

		signerAccounts := accountsFromValue(inter, signersValue)

		err = testFramework.AddTransaction(code, authorizer, signerAccounts)
		if err != nil {
			panic(err)
		}

		return interpreter.VoidValue{}
	},
	emulatorBackendAddTransactionFunctionType,
)

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

		// Get account key
		accountKeyValue := accountValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			accountKeyFieldName,
		)
		accountKey := accountKeyFromValue(inter, accountKeyValue)

		// Get private key
		privateKeyValue := accountValue.GetMember(
			inter,
			interpreter.ReturnEmptyLocationRange,
			accountPrivateKeyFieldName,
		)

		privateKey, err := interpreter.ByteArrayValueToByteSlice(nil, privateKeyValue)
		if err != nil {
			panic(errors.NewUnreachableError())
		}

		accounts = append(accounts, &interpreter.Account{
			Address:    common.Address(address),
			AccountKey: accountKey,
			PrivateKey: privateKey,
		})

		return true
	})

	return accounts
}

func accountKeyFromValue(inter *interpreter.Interpreter, value interpreter.Value) *interpreter.AccountKey {
	accountKeyValue, ok := value.(interpreter.MemberAccessibleValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Key index field
	keyIndexVal := accountKeyValue.GetMember(
		inter,
		interpreter.ReturnEmptyLocationRange,
		sema.AccountKeyKeyIndexField,
	)
	keyIndex, ok := keyIndexVal.(interpreter.IntValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Public key field
	publicKeyVal := accountKeyValue.GetMember(
		inter,
		interpreter.ReturnEmptyLocationRange,
		sema.AccountKeyPublicKeyField,
	)
	publicKey := publicKeyFromValue(inter, interpreter.ReturnEmptyLocationRange, publicKeyVal)

	// Hash algo field
	hashAlgoField := accountKeyValue.GetMember(inter, interpreter.ReturnEmptyLocationRange, sema.AccountKeyHashAlgoField)
	if hashAlgoField == nil {
		panic(errors.NewUnreachableError())
	}
	hashAlgo := hashAlgoFromValue(inter, hashAlgoField)

	// Weight field
	weightVal := accountKeyValue.GetMember(
		inter,
		interpreter.ReturnEmptyLocationRange,
		sema.AccountKeyWeightField,
	)
	weight, ok := weightVal.(interpreter.UFix64Value)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// isRevoked field
	isRevokedVal := accountKeyValue.GetMember(
		inter,
		interpreter.ReturnEmptyLocationRange,
		sema.AccountKeyIsRevokedField,
	)
	isRevoked, ok := isRevokedVal.(interpreter.BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	accountKey := &interpreter.AccountKey{
		KeyIndex:  keyIndex.ToInt(),
		PublicKey: publicKey,
		HashAlgo:  hashAlgo,
		Weight:    weight.ToInt(),
		IsRevoked: bool(isRevoked),
	}

	return accountKey
}

func hashAlgoFromValue(inter *interpreter.Interpreter, hashAlgoField interpreter.Value) sema.HashAlgorithm {
	hashAlgoValue, ok := hashAlgoField.(interpreter.MemberAccessibleValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	rawValue := hashAlgoValue.GetMember(inter, interpreter.ReturnEmptyLocationRange, sema.EnumRawValueFieldName)
	if rawValue == nil {
		panic(errors.NewUnreachableError())
	}

	hashAlgoRawValue, ok := rawValue.(interpreter.UInt8Value)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return sema.HashAlgorithm(hashAlgoRawValue)
}

func publicKeyFromValue(
	inter *interpreter.Interpreter,
	getLocationRange func() interpreter.LocationRange,
	value interpreter.Value,
) *interpreter.PublicKey {

	publicKey, ok := value.(interpreter.MemberAccessibleValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Public key field
	key := publicKey.GetMember(inter, getLocationRange, sema.PublicKeyPublicKeyField)

	byteArray, err := interpreter.ByteArrayValueToByteSlice(inter, key)
	if err != nil {
		panic(err)
	}

	// sign algo field
	signAlgoField := publicKey.GetMember(inter, getLocationRange, sema.PublicKeySignAlgoField)
	if signAlgoField == nil {
		panic(errors.NewUnexpectedError("sign algorithm is not set"))
	}

	signAlgoValue, ok := signAlgoField.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	rawValue := signAlgoValue.GetField(inter, getLocationRange, sema.EnumRawValueFieldName)
	if rawValue == nil {
		panic(errors.NewUnreachableError())
	}

	signAlgoRawValue, ok := rawValue.(interpreter.UInt8Value)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return &interpreter.PublicKey{
		PublicKey: byteArray,
		SignAlgo:  sema.SignatureAlgorithm(signAlgoRawValue.ToInt()),
	}
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

// Matcher type

var matcherType = func() *sema.InterfaceType {
	typeName := matcherTypeName

	typ, ok := testContractType.NestedTypes.Get(typeName)
	if !ok {
		panic(errors.NewUnexpectedError("cannot find type %s.%s", testContractTypeName, typeName))
	}

	compositeType, ok := typ.(*sema.InterfaceType)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for '%s'. expected struct", typeName))
	}

	return compositeType
}()

var matcherTestType = func() *sema.FunctionType {
	member, ok := matcherType.Members.Get(matcherTestFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError("cannot find field %s in ", matcherTestFunctionName, matcherTypeName))
	}

	testFieldType, ok := member.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for field %s in ", matcherTestFunctionName, matcherTypeName))
	}

	return testFieldType
}()

func newStructMatcher(
	inter *interpreter.Interpreter,
	testContract interpreter.Value,
	matcherTestFunc *interpreter.HostFunctionValue,
) interpreter.Value {

	testContractValue, ok := testContract.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for %s contract", testContractTypeName))
	}

	matcherConstructorVar := testContractValue.NestedVariables[structMatcherTypeName]
	matcherConstructor, ok := matcherConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}

	matcher, err := inter.InvokeExternally(
		matcherConstructor,
		matcherConstructor.Type,
		[]interpreter.Value{
			matcherTestFunc,
		},
	)

	if err != nil {
		panic(err)
	}

	return matcher
}

// Built-in matchers

const testEqualMatcherFunctionName = "equal"

const testEqualMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is equal to the given value
`

var typeParameter = &sema.TypeParameter{
	TypeBound: sema.AnyType,
	Name:      "T",
	Optional:  true,
}

var testEqualMatcherFunctionType = &sema.FunctionType{
	IsConstructor: false,
	TypeParameters: []*sema.TypeParameter{
		typeParameter,
	},
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "value",
			TypeAnnotation: sema.NewTypeAnnotation(
				&sema.GenericType{
					TypeParameter: typeParameter,
				},
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.RestrictedType{
			Type: sema.AnyStructType,
			Restrictions: []*sema.InterfaceType{
				matcherType,
			},
		},
	),
}

var testEqualMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		otherValue, ok := invocation.Arguments[0].(interpreter.EquatableValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		inter := invocation.Interpreter

		equalTestFunc := interpreter.NewHostFunctionValue(
			nil,
			func(invocation interpreter.Invocation) interpreter.Value {

				thisValue, ok := invocation.Arguments[0].(interpreter.EquatableValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				equal := thisValue.Equal(
					inter,
					interpreter.ReturnEmptyLocationRange,
					otherValue,
				)

				return interpreter.BoolValue(equal)
			},
			matcherTestType,
		)

		if otherValue.IsResourceKinded(inter) {
			return newResourceMatcher(inter, equalTestFunc)
		}

		return newStructMatcher(inter, invocation.Self, equalTestFunc)
	},
	testEqualMatcherFunctionType,
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

// 'ResourceMatcher' struct.
//

const resourceMatcherTypeName = "ResourceMatcher"

var resourceMatcherType = func() *sema.CompositeType {

	ty := &sema.CompositeType{
		Identifier: resourceMatcherTypeName,
		Kind:       common.CompositeKindStructure,
		Location:   TestContractLocation,
		ExplicitInterfaceConformances: []*sema.InterfaceType{
			matcherType,
		},
	}

	var members = []*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			resourceMatcherTestFunctionName,
			resourceMatcherTestFieldType(),
			resourceMatcherTestFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			resourceMatcherOrFunctionName,
			resourceMatcherOrFunctionType,
			resourceMatcherOrFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			resourceMatcherAndFunctionName,
			resourceMatcherAndFunctionType,
			resourceMatcherAndFunctionDocString,
		),
	}

	ty.Members = sema.GetMembersAsMap(members)
	ty.Fields = sema.GetFieldNames(members)

	return ty
}()

func newResourceMatcher(
	inter *interpreter.Interpreter,
	testFunc *interpreter.HostFunctionValue,
) *interpreter.CompositeValue {

	matcher := interpreter.NewCompositeValue(
		inter,
		interpreter.ReturnEmptyLocationRange,
		resourceMatcherType.Location,
		resourceMatcherTypeName,
		common.CompositeKindStructure,
		nil,
		common.Address{},
	)

	matcher.Functions = map[string]interpreter.FunctionValue{
		resourceMatcherTestFunctionName: testFunc,
		resourceMatcherOrFunctionName:   resourceMatcherOrFunction,
		resourceMatcherAndFunctionName:  resourceMatcherAndFunction,
	}

	return matcher
}

// 'ResourceMatcher.test' function

const resourceMatcherTestFunctionName = "test"

const resourceMatcherTestFunctionDocString = `test field`

var resourceMatcherTestFieldType = func() *sema.FunctionType {
	// The type of the 'test' field of 'ResourceMatcher' (interface-implementation)
	// is same as that of 'Matcher' interface.
	typ, ok := matcherType.Members.Get(resourceMatcherTestFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			matcherTypeName,
			resourceMatcherTestFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			resourceMatcherTestFunctionName,
		))
	}

	return functionType
}

// 'ResourceMatcher.or' function

const resourceMatcherOrFunctionName = "or"

const resourceMatcherOrFunctionDocString = `or function`

var resourceMatcherOrFunctionType = func() *sema.FunctionType {
	// The type of the 'or' function of 'ResourceMatcher' (interface-implementation)
	// is same as that of 'Matcher' interface.
	typ, ok := matcherType.Members.Get(resourceMatcherOrFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			matcherTypeName,
			resourceMatcherOrFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			resourceMatcherOrFunctionName,
		))
	}

	return functionType
}()

// 'ResourceMatcher.and' function

const resourceMatcherAndFunctionName = "and"

const resourceMatcherAndFunctionDocString = `or function`

var resourceMatcherAndFunctionType = func() *sema.FunctionType {
	// The type of the 'and' function of 'ResourceMatcher' (interface-implementation)
	// is same as that of 'Matcher' interface.
	typ, ok := matcherType.Members.Get(resourceMatcherAndFunctionName)
	if !ok {
		panic(errors.NewUnexpectedError(
			"cannot find type %s.%s",
			matcherTypeName,
			resourceMatcherAndFunctionName,
		))
	}

	functionType, ok := typ.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for %s. expected function",
			resourceMatcherAndFunctionName,
		))
	}

	return functionType
}()

var resourceMatcherOrFunction interpreter.FunctionValue

var resourceMatcherAndFunction interpreter.FunctionValue

func init() {
	// initialize this inside 'init' to break the initialization loop.

	resourceMatcherOrFunction = interpreter.NewUnmeteredHostFunctionValue(
		func(orFuncInvocation interpreter.Invocation) interpreter.Value {
			inter := orFuncInvocation.Interpreter

			thisMatcher := orFuncInvocation.Self

			otherMatcher, ok := orFuncInvocation.Arguments[0].(*interpreter.CompositeValue)
			if !ok {
				panic(errors.NewUnexpectedError("invalid type for matcher"))
			}

			testFunc := interpreter.NewHostFunctionValue(
				nil,
				func(invocation interpreter.Invocation) interpreter.Value {

					value, ok := invocation.Arguments[0].(interpreter.EquatableValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					thisMatcherTestResult := invokeMatcherTest(invocation.Interpreter, thisMatcher, value)
					if thisMatcherTestResult {
						return interpreter.BoolValue(true)
					}

					otherMatcherTestResult := invokeMatcherTest(invocation.Interpreter, otherMatcher, value)
					return interpreter.BoolValue(otherMatcherTestResult)
				},
				matcherTestType,
			)

			return newResourceMatcher(inter, testFunc)
		},
		resourceMatcherOrFunctionType,
	)

	resourceMatcherAndFunction = interpreter.NewUnmeteredHostFunctionValue(
		func(orFuncInvocation interpreter.Invocation) interpreter.Value {
			inter := orFuncInvocation.Interpreter

			thisMatcher := orFuncInvocation.Self

			otherMatcher, ok := orFuncInvocation.Arguments[0].(*interpreter.CompositeValue)
			if !ok {
				panic(errors.NewUnexpectedError("invalid type for matcher"))
			}

			testFunc := interpreter.NewHostFunctionValue(
				nil,
				func(invocation interpreter.Invocation) interpreter.Value {

					value, ok := invocation.Arguments[0].(interpreter.EquatableValue)
					if !ok {
						panic(errors.NewUnreachableError())
					}

					thisMatcherTestResult := invokeMatcherTest(invocation.Interpreter, thisMatcher, value)
					if !thisMatcherTestResult {
						return interpreter.BoolValue(false)
					}

					otherMatcherTestResult := invokeMatcherTest(invocation.Interpreter, otherMatcher, value)
					return interpreter.BoolValue(otherMatcherTestResult)
				},
				matcherTestType,
			)

			return newResourceMatcher(inter, testFunc)
		},
		resourceMatcherOrFunctionType,
	)
}
