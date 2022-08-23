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
const errorTypeName = "Error"
const matcherTypeName = "Matcher"

const succeededCaseName = "succeeded"
const failedCaseName = "failed"

const transactionCodeFieldName = "code"
const transactionAuthorizerFieldName = "authorizers"
const transactionSignersFieldName = "signers"
const transactionArgsFieldName = "arguments"

const accountAddressFieldName = "address"

const matcherTestFunctionName = "test"

const addressesFieldName = "addresses"

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
		&sema.Config{
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
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
	testFramework interpreter.TestFramework,
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
	compositeValue.Functions[testNewEmulatorBlockchainFunctionName] = testNewEmulatorBlockchainFunction(testFramework)
	compositeValue.Functions[testReadFileFunctionName] = testReadFileFunction(testFramework)

	// Inject natively implemented matchers
	compositeValue.Functions[newMatcherFunctionName] = newMatcherFunction
	compositeValue.Functions[equalMatcherFunctionName] = equalMatcherFunction

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

func typeNotFoundError(parentType, nestedType string) error {
	return errors.NewUnexpectedError("cannot find type '%s.%s'", parentType, nestedType)
}

func memberNotFoundError(parentType, member string) error {
	return errors.NewUnexpectedError("cannot find member '%s.%s'", parentType, member)
}

var blockchainBackendInterfaceType = func() *sema.InterfaceType {
	typ, ok := testContractType.NestedTypes.Get(blockchainBackendTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, blockchainBackendTypeName))
	}

	interfaceType, ok := typ.(*sema.InterfaceType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected interface",
			blockchainBackendTypeName,
		))
	}

	return interfaceType
}()

var matcherType = func() *sema.CompositeType {
	typ, ok := testContractType.NestedTypes.Get(matcherTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, matcherTypeName))
	}

	compositeType, ok := typ.(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected struct type",
			matcherTypeName,
		))
	}

	return compositeType
}()

var matcherTestFunctionType = compositeFunctionType(matcherType, matcherTestFunctionName)

func compositeFunctionType(parent *sema.CompositeType, funcName string) *sema.FunctionType {
	testFunc, ok := parent.Members.Get(funcName)
	if !ok {
		panic(memberNotFoundError(parent.Identifier, funcName))
	}

	return getFunctionTypeFromMember(testFunc, funcName)
}

func interfaceFunctionType(parent *sema.InterfaceType, funcName string) *sema.FunctionType {
	testFunc, ok := parent.Members.Get(funcName)
	if !ok {
		panic(memberNotFoundError(parent.Identifier, funcName))
	}

	return getFunctionTypeFromMember(testFunc, funcName)
}

func getFunctionTypeFromMember(funcMember *sema.Member, funcName string) *sema.FunctionType {
	functionType, ok := funcMember.TypeAnnotation.Type.(*sema.FunctionType)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected function type",
			funcName,
		))
	}

	return functionType
}

func init() {

	// Enrich 'Test' contract with natively implemented functions

	// Test.assert()
	testContractType.Members.Set(
		testAssertFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testAssertFunctionName,
			testAssertFunctionType,
			testAssertFunctionDocString,
		),
	)

	// Test.expect()
	testContractType.Members.Set(
		testExpectFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testExpectFunctionName,
			testExpectFunctionType,
			testExpectFunctionDocString,
		),
	)

	// Test.newEmulatorBlockchain()
	testContractType.Members.Set(
		testNewEmulatorBlockchainFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testNewEmulatorBlockchainFunctionName,
			testNewEmulatorBlockchainFunctionType,
			testNewEmulatorBlockchainFunctionDocString,
		),
	)

	// Test.newMatcher()
	testContractType.Members.Set(
		newMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			newMatcherFunctionName,
			newMatcherFunctionType,
			newMatcherFunctionDocString,
		),
	)

	// Matcher functions
	testContractType.Members.Set(
		equalMatcherFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			equalMatcherFunctionName,
			equalMatcherFunctionType,
			equalMatcherFunctionDocString,
		),
	)

	// Test.readFile()
	testContractType.Members.Set(
		testReadFileFunctionName,
		sema.NewUnmeteredPublicFunctionMember(
			testContractType,
			testReadFileFunctionName,
			testReadFileFunctionType,
			testReadFileFunctionDocString,
		),
	)

	// Enrich 'Test' contract elaboration with natively implemented composite types.
	// e.g: 'EmulatorBackend' type.
	TestContractChecker.Elaboration.CompositeTypes[EmulatorBackendType.ID()] = EmulatorBackendType
}

var blockchainType = func() sema.Type {
	typ, ok := testContractType.NestedTypes.Get(blockchainTypeName)
	if !ok {
		panic(typeNotFoundError(testContractTypeName, blockchainTypeName))
	}

	return typ
}()

// Functions belong to the 'Test' contract

// 'Test.assert' function

const testAssertFunctionDocString = `
Fails the test-case if the given condition is false, and reports a message which explains how the condition is false.
`

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
	RequiredArgumentCount: sema.RequiredArgumentCount(1),
}

var testAssertFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		condition, ok := invocation.Arguments[0].(interpreter.BoolValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		var message string
		if len(invocation.Arguments) > 1 {
			messageValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			message = messageValue.Str
		}

		if !condition {
			panic(AssertionError{
				Message:       message,
				LocationRange: invocation.GetLocationRange(),
			})
		}

		return interpreter.VoidValue{}
	},
	testAssertFunctionType,
)

// 'Test.expect' function

const testExpectFunctionDocString = `
Expect function tests a value against a matcher, and fails the test if it's not a match.
`

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
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "matcher",
			TypeAnnotation: sema.NewTypeAnnotation(matcherType),
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

		result := invokeMatcherTest(
			invocation.Interpreter,
			matcher,
			value,
			invocation.GetLocationRange,
		)

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
	locationRangeGetter func() interpreter.LocationRange,
) bool {
	testFunc := matcher.GetMember(
		inter,
		locationRangeGetter,
		matcherTestFunctionName,
	)

	funcValue, ok := testFunc.(interpreter.FunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError(
			"invalid type for '%s'. expected function",
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

// 'Test.readFile' function

const testReadFileFunctionDocString = `
Read a local file, and return the content as a string.
`

const testReadFileFunctionName = "readFile"

var testReadFileFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "path",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.StringType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.StringType,
	),
}

func testReadFileFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			pathString, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			content, err := testFramework.ReadFile(pathString.Str)
			if err != nil {
				panic(err)
			}

			return interpreter.NewUnmeteredStringValue(content)
		},
		testReadFileFunctionType,
	)
}

// 'Test.newEmulatorBlockchain' function

const testNewEmulatorBlockchainFunctionDocString = `
Creates a blockchain which is backed by a new emulator instance.
`

const testNewEmulatorBlockchainFunctionName = "newEmulatorBlockchain"

var testNewEmulatorBlockchainFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		blockchainType,
	),
}

func testNewEmulatorBlockchainFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			// Create an `EmulatorBackend`
			emulatorBackend := newEmulatorBackend(
				inter,
				testFramework,
				invocation.GetLocationRange,
			)

			// Create a 'Blockchain' struct value, that wraps the emulator backend,
			// by calling the constructor of 'Blockchain'.

			blockchainConstructor := getNestedTypeConstructorValue(invocation.Self, blockchainTypeName)

			blockchain, err := inter.InvokeExternally(
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
}

func getNestedTypeConstructorValue(parent interpreter.Value, typeName string) *interpreter.HostFunctionValue {
	compositeValue, ok := parent.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	constructorVar := compositeValue.NestedVariables[typeName]
	constructor, ok := constructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}
	return constructor
}

// 'Test.NewMatcher' function.
// Constructs a matcher that test only 'AnyStruct'.
// Accepts test function that accepts subtype of 'AnyStruct'.
//
// Signature:
//    fun newMatcher<T: AnyStruct>(test: ((T): Bool)): Test.Matcher
//
// where `T` is optional, and bound to `AnyStruct`.
//
// Sample usage: `Test.newMatcher(fun (_ value: Int: Bool) { return true })`

const newMatcherFunctionDocString = `
Creates a matcher with a test function.
The test function is of type '((T): Bool)', where 'T' is bound to 'AnyStruct'.
`

const newMatcherFunctionName = "newMatcher"

var newMatcherFunctionType = &sema.FunctionType{
	IsConstructor: true,
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "test",
			TypeAnnotation: sema.NewTypeAnnotation(
				// Type of the 'test' function: ((T): Bool)
				&sema.FunctionType{
					Parameters: []*sema.Parameter{
						{
							Label:      sema.ArgumentLabelNotRequired,
							Identifier: "value",
							TypeAnnotation: sema.NewTypeAnnotation(
								&sema.GenericType{
									TypeParameter: newMatcherFunctionTypeParameter,
								},
							),
						},
					},
					ReturnTypeAnnotation: sema.NewTypeAnnotation(
						sema.BoolType,
					),
				},
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
	TypeParameters: []*sema.TypeParameter{
		newMatcherFunctionTypeParameter,
	},
}

var newMatcherFunctionTypeParameter = &sema.TypeParameter{
	TypeBound: sema.AnyStructType,
	Name:      "T",
	Optional:  true,
}

var newMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		test, ok := invocation.Arguments[0].(interpreter.FunctionValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return newMatcherWithGenericTestFunction(invocation, test)
	},
	equalMatcherFunctionType,
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
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			emulatorBackendDeployContractFunctionName,
			emulatorBackendDeployContractFunctionType,
			emulatorBackendDeployContractFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			emulatorBackendUseConfigFunctionName,
			emulatorBackendUseConfigFunctionType,
			emulatorBackendUseConfigFunctionDocString,
		),
	}

	ty.Members = sema.GetMembersAsMap(members)
	ty.Fields = sema.GetFieldNames(members)

	return ty
}()

func newEmulatorBackend(
	inter *interpreter.Interpreter,
	testFramework interpreter.TestFramework,
	locationRangeGetter func() interpreter.LocationRange,
) *interpreter.CompositeValue {
	var fields = []interpreter.CompositeField{
		{
			Name:  emulatorBackendExecuteScriptFunctionName,
			Value: emulatorBackendExecuteScriptFunction(testFramework),
		},
		{
			Name:  emulatorBackendCreateAccountFunctionName,
			Value: emulatorBackendCreateAccountFunction(testFramework),
		}, {
			Name:  emulatorBackendAddTransactionFunctionName,
			Value: emulatorBackendAddTransactionFunction(testFramework),
		},
		{
			Name:  emulatorBackendExecuteNextTransactionFunctionName,
			Value: emulatorBackendExecuteNextTransactionFunction(testFramework),
		},
		{
			Name:  emulatorBackendCommitBlockFunctionName,
			Value: emulatorBackendCommitBlockFunction(testFramework),
		},
		{
			Name:  emulatorBackendDeployContractFunctionName,
			Value: emulatorBackendDeployContractFunction(testFramework),
		},
		{
			Name:  emulatorBackendUseConfigFunctionName,
			Value: emulatorBackendUseConfigFunction(testFramework),
		},
	}

	return interpreter.NewCompositeValue(
		inter,
		locationRangeGetter,
		EmulatorBackendType.Location,
		emulatorBackendTypeName,
		common.CompositeKindStructure,
		fields,
		common.Address{},
	)
}

// 'EmulatorBackend.executeScript' function

const emulatorBackendExecuteScriptFunctionName = "executeScript"

const emulatorBackendExecuteScriptFunctionDocString = `
Executes a script and returns the script return value and the status.
The 'returnValue' field of the result will be nil if the script failed.
`

var emulatorBackendExecuteScriptFunctionType = interfaceFunctionType(
	blockchainBackendInterfaceType,
	emulatorBackendExecuteScriptFunctionName,
)

func emulatorBackendExecuteScriptFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			script, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			args, err := arrayValueToSlice(invocation.Arguments[1])
			if err != nil {
				panic(errors.NewUnexpectedErrorFromCause(err))
			}

			result := testFramework.RunScript(script.Str, args)

			return newScriptResult(invocation.Interpreter, result.Value, result)
		},
		emulatorBackendExecuteScriptFunctionType,
	)
}

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

// newScriptResult Creates a "ScriptResult" using the return value of the executed script.
//
func newScriptResult(
	inter *interpreter.Interpreter,
	returnValue interpreter.Value,
	result *interpreter.ScriptResult,
) interpreter.Value {

	if returnValue == nil {
		returnValue = interpreter.NilValue{}
	}

	// Lookup and get 'ResultStatus' enum value.
	resultStatusConstructor := getConstructor(inter, resultStatusTypeName)
	var status interpreter.Value
	if result.Error == nil {
		succeededVar := resultStatusConstructor.NestedVariables[succeededCaseName]
		status = succeededVar.GetValue()
	} else {
		failedVar := resultStatusConstructor.NestedVariables[failedCaseName]
		status = failedVar.GetValue()
	}

	errValue := newErrorValue(inter, result.Error)

	// Create a 'ScriptResult' by calling its constructor.
	scriptResultConstructor := getConstructor(inter, scriptResultTypeName)
	scriptResult, err := inter.InvokeExternally(
		scriptResultConstructor,
		scriptResultConstructor.Type,
		[]interpreter.Value{
			status,
			returnValue,
			errValue,
		},
	)

	if err != nil {
		panic(err)
	}

	return scriptResult
}

func getConstructor(inter *interpreter.Interpreter, typeName string) *interpreter.HostFunctionValue {
	resultStatusConstructorVar := inter.FindVariable(typeName)
	resultStatusConstructor, ok := resultStatusConstructorVar.GetValue().(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor of '%s'", typeName))
	}

	return resultStatusConstructor
}

// 'EmulatorBackend.createAccount' function

const emulatorBackendCreateAccountFunctionName = "createAccount"

const emulatorBackendCreateAccountFunctionDocString = `
Creates an account by submitting an account creation transaction.
The transaction is paid by the service account.
The returned account can be used to sign and authorize transactions.
`

var emulatorBackendCreateAccountFunctionType = interfaceFunctionType(
	blockchainBackendInterfaceType,
	emulatorBackendCreateAccountFunctionName,
)

func emulatorBackendCreateAccountFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			account, err := testFramework.CreateAccount()
			if err != nil {
				panic(err)
			}

			return newAccountValue(invocation.Interpreter, account, invocation.GetLocationRange)
		},
		emulatorBackendCreateAccountFunctionType,
	)
}

func newAccountValue(
	inter *interpreter.Interpreter,
	account *interpreter.Account,
	locationRangeGetter func() interpreter.LocationRange,
) interpreter.Value {

	// Create address value
	address := interpreter.NewAddressValue(nil, account.Address)

	// Create public key
	publicKey := interpreter.NewPublicKeyValue(
		inter,
		locationRangeGetter,
		interpreter.ByteSliceToByteArrayValue(
			inter,
			account.PublicKey.PublicKey,
		),
		NewSignatureAlgorithmCase(
			interpreter.UInt8Value(account.PublicKey.SignAlgo.RawValue()),
		),
		inter.Config.PublicKeyValidationHandler,
	)

	// Create an 'Account' by calling its constructor.
	accountConstructor := getConstructor(inter, accountTypeName)
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

const emulatorBackendAddTransactionFunctionDocString = `
Add a transaction to the current block.
`

var emulatorBackendAddTransactionFunctionType = interfaceFunctionType(
	blockchainBackendInterfaceType,
	emulatorBackendAddTransactionFunctionName,
)

func emulatorBackendAddTransactionFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter
			locationRangeGetter := invocation.GetLocationRange

			transactionValue, ok := invocation.Arguments[0].(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Get transaction code
			codeValue := transactionValue.GetMember(
				inter,
				locationRangeGetter,
				transactionCodeFieldName,
			)
			code, ok := codeValue.(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Get authorizers
			authorizerValue := transactionValue.GetMember(
				inter,
				locationRangeGetter,
				transactionAuthorizerFieldName,
			)

			authorizers := addressesFromValue(authorizerValue)

			// Get signers
			signersValue := transactionValue.GetMember(
				inter,
				locationRangeGetter,
				transactionSignersFieldName,
			)

			signerAccounts := accountsFromValue(inter, signersValue, invocation.GetLocationRange)

			// Get arguments
			argsValue := transactionValue.GetMember(
				inter,
				locationRangeGetter,
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
}

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

func accountsFromValue(
	inter *interpreter.Interpreter,
	accountsValue interpreter.Value,
	locationRangeGetter func() interpreter.LocationRange,
) []*interpreter.Account {

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

		account := accountFromValue(inter, accountValue, locationRangeGetter)

		accounts = append(accounts, account)

		return true
	})

	return accounts
}

func accountFromValue(
	inter *interpreter.Interpreter,
	accountValue interpreter.MemberAccessibleValue,
	locationRangeGetter func() interpreter.LocationRange,
) *interpreter.Account {

	// Get address
	addressValue := accountValue.GetMember(
		inter,
		locationRangeGetter,
		accountAddressFieldName,
	)
	address, ok := addressValue.(interpreter.AddressValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Get public key
	publicKeyVal, ok := accountValue.GetMember(
		inter,
		locationRangeGetter,
		sema.AccountKeyPublicKeyField,
	).(interpreter.MemberAccessibleValue)

	if !ok {
		panic(errors.NewUnreachableError())
	}

	publicKey, err := NewPublicKeyFromValue(inter, locationRangeGetter, publicKeyVal)
	if err != nil {
		panic(err)
	}

	return &interpreter.Account{
		Address:   common.Address(address),
		PublicKey: publicKey,
	}
}

// 'EmulatorBackend.executeNextTransaction' function

const emulatorBackendExecuteNextTransactionFunctionName = "executeNextTransaction"

const emulatorBackendExecuteNextTransactionFunctionDocString = `
Executes the next transaction in the block, if any.
Returns the result of the transaction, or nil if no transaction was scheduled.
`

var emulatorBackendExecuteNextTransactionFunctionType = interfaceFunctionType(
	blockchainBackendInterfaceType,
	emulatorBackendExecuteNextTransactionFunctionName,
)

func emulatorBackendExecuteNextTransactionFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			result := testFramework.ExecuteNextTransaction()

			// If there are no transactions to run, then return `nil`.
			if result == nil {
				return interpreter.NilValue{}
			}

			return newTransactionResult(invocation.Interpreter, result)
		},
		emulatorBackendExecuteNextTransactionFunctionType,
	)
}

// newTransactionResult Creates a "TransactionResult" indicating the status of the transaction execution.
//
func newTransactionResult(inter *interpreter.Interpreter, result *interpreter.TransactionResult) interpreter.Value {
	// Lookup and get 'ResultStatus' enum value.
	resultStatusConstructor := getConstructor(inter, resultStatusTypeName)
	var status interpreter.Value
	if result.Error == nil {
		succeededVar := resultStatusConstructor.NestedVariables[succeededCaseName]
		status = succeededVar.GetValue()
	} else {
		failedVar := resultStatusConstructor.NestedVariables[failedCaseName]
		status = failedVar.GetValue()
	}

	// Create a 'TransactionResult' by calling its constructor.
	transactionResultConstructor := getConstructor(inter, transactionResultTypeName)

	errValue := newErrorValue(inter, result.Error)

	transactionResult, err := inter.InvokeExternally(
		transactionResultConstructor,
		transactionResultConstructor.Type,
		[]interpreter.Value{
			status,
			errValue,
		},
	)

	if err != nil {
		panic(err)
	}

	return transactionResult
}

func newErrorValue(inter *interpreter.Interpreter, err error) interpreter.Value {
	if err == nil {
		return interpreter.NilValue{}
	}

	// Create a 'Error' by calling its constructor.
	errorConstructor := getConstructor(inter, errorTypeName)

	errorValue, invocationErr := inter.InvokeExternally(
		errorConstructor,
		errorConstructor.Type,
		[]interpreter.Value{
			interpreter.NewUnmeteredStringValue(err.Error()),
		},
	)

	if invocationErr != nil {
		panic(invocationErr)
	}

	return errorValue
}

// 'EmulatorBackend.commitBlock' function

const emulatorBackendCommitBlockFunctionName = "commitBlock"

const emulatorBackendCommitBlockFunctionDocString = `
Commit the current block. Committing will fail if there are un-executed transactions in the block.
`

var emulatorBackendCommitBlockFunctionType = interfaceFunctionType(
	blockchainBackendInterfaceType,
	emulatorBackendCommitBlockFunctionName,
)

func emulatorBackendCommitBlockFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			err := testFramework.CommitBlock()
			if err != nil {
				panic(err)
			}

			return interpreter.VoidValue{}
		},
		emulatorBackendCommitBlockFunctionType,
	)
}

// Built-in matchers

const equalMatcherFunctionName = "equal"

const equalMatcherFunctionDocString = `
Returns a matcher that succeeds if the tested value is equal to the given value.
`

var typeParameter = &sema.TypeParameter{
	TypeBound: sema.AnyStructType,
	Name:      "T",
	Optional:  true,
}

var equalMatcherFunctionType = &sema.FunctionType{
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
	ReturnTypeAnnotation: sema.NewTypeAnnotation(matcherType),
}

var equalMatcherFunction = interpreter.NewUnmeteredHostFunctionValue(
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
					invocation.GetLocationRange,
					otherValue,
				)

				return interpreter.BoolValue(equal)
			},
			matcherTestFunctionType,
		)

		return newMatcherWithGenericTestFunction(invocation, equalTestFunc)
	},
	equalMatcherFunctionType,
)

// 'EmulatorBackend.deployContract' function

const emulatorBackendDeployContractFunctionName = "deployContract"

const emulatorBackendDeployContractFunctionDocString = `
Deploys a given contract, and initializes it with the provided arguments.
`

var emulatorBackendDeployContractFunctionType = interfaceFunctionType(
	blockchainBackendInterfaceType,
	emulatorBackendDeployContractFunctionName,
)

func emulatorBackendDeployContractFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			// Contract name
			name, ok := invocation.Arguments[0].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// Contract code
			code, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			// authorizer
			accountValue, ok := invocation.Arguments[2].(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			account := accountFromValue(inter, accountValue, invocation.GetLocationRange)

			// Contract init arguments
			args, err := arrayValueToSlice(invocation.Arguments[3])
			if err != nil {
				panic(err)
			}

			err = testFramework.DeployContract(
				name.Str,
				code.Str,
				account,
				args,
			)

			return newErrorValue(inter, err)
		},
		emulatorBackendDeployContractFunctionType,
	)
}

// 'EmulatorBackend.useConfiguration' function

const emulatorBackendUseConfigFunctionName = "useConfiguration"

const emulatorBackendUseConfigFunctionDocString = `Use configurations function`

var emulatorBackendUseConfigFunctionType = interfaceFunctionType(
	blockchainBackendInterfaceType,
	emulatorBackendUseConfigFunctionName,
)

func emulatorBackendUseConfigFunction(testFramework interpreter.TestFramework) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			// configurations
			configsValue, ok := invocation.Arguments[0].(*interpreter.CompositeValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			addresses, ok := configsValue.GetMember(
				inter,
				invocation.GetLocationRange,
				addressesFieldName,
			).(*interpreter.DictionaryValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			mapping := make(map[string]common.Address, addresses.Count())

			addresses.Iterate(nil, func(locationValue, addressValue interpreter.Value) bool {
				location, ok := locationValue.(*interpreter.StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				address, ok := addressValue.(interpreter.AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				mapping[location.Str] = common.Address(address)

				return true
			})

			testFramework.UseConfiguration(&interpreter.Configuration{
				Addresses: mapping,
			})

			return interpreter.VoidValue{}
		},
		emulatorBackendUseConfigFunctionType,
	)
}

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

func newMatcherWithGenericTestFunction(
	invocation interpreter.Invocation,
	testFunc interpreter.FunctionValue,
) interpreter.Value {

	inter := invocation.Interpreter

	staticType, ok := testFunc.StaticType(inter).(interpreter.FunctionStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	parameters := staticType.Type.Parameters

	// Wrap the user provided test function with a function that validates the argument types.
	// i.e: create a closure that cast the arguments.
	//
	// e.g: convert `newMatcher(test: ((Int): Bool))` to:
	//
	//  newMatcher(fun (b: AnyStruct): Bool {
	//      return test(b as! Int)
	//  })
	//
	// Note: This argument validation is only needed if the matcher was created with a user-provided function.
	// No need to validate if the matcher is created as a matcher combinator.
	//
	matcherTestFunction := interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			for i, argument := range invocation.Arguments {
				paramType := parameters[i].TypeAnnotation.Type
				argumentType := argument.StaticType(inter)
				argTypeMatch := inter.IsSubTypeOfSemaType(argumentType, paramType)

				if !argTypeMatch {
					panic(interpreter.TypeMismatchError{
						ExpectedType:  paramType,
						LocationRange: invocation.GetLocationRange(),
					})
				}
			}

			value, err := inter.InvokeFunction(testFunc, invocation)
			if err != nil {
				panic(err)
			}

			return value
		},
		matcherTestFunctionType,
	)

	matcherConstructor := getNestedTypeConstructorValue(invocation.Self, matcherTypeName)
	matcher, err := inter.InvokeExternally(
		matcherConstructor,
		matcherConstructor.Type,
		[]interpreter.Value{
			matcherTestFunction,
		},
	)

	if err != nil {
		panic(err)
	}

	return matcher
}
