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

package stdlib

import (
	"fmt"
	"sync"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// This is the Cadence standard library for writing tests.
// It provides the Cadence constructs (structs, functions, etc.) that are needed to
// write tests in Cadence.

const testContractTypeName = "Test"

const testScriptResultTypeName = "ScriptResult"
const testTransactionResultTypeName = "TransactionResult"
const testResultStatusTypeName = "ResultStatus"
const testResultStatusTypeSucceededCaseName = "succeeded"
const testResultStatusTypeFailedCaseName = "failed"
const testAccountTypeName = "TestAccount"
const testErrorTypeName = "Error"
const testMatcherTypeName = "Matcher"

const accountAddressFieldName = "address"

const matcherTestFieldName = "test"

const TestContractLocation = common.IdentifierLocation(testContractTypeName)

var testOnce sync.Once

// Deprecated: Use GetTestContract instead
var testContractType *TestContractType

func GetTestContractType() *TestContractType {
	testOnce.Do(func() {
		testContractType = newTestContractType()
	})
	return testContractType
}

func typeNotFoundError(parentType, nestedType string) error {
	return errors.NewUnexpectedError("cannot find type '%s.%s'", parentType, nestedType)
}

func memberNotFoundError(parentType, member string) error {
	return errors.NewUnexpectedError("cannot find member '%s.%s'", parentType, member)
}

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

func getNestedTypeConstructorValue(
	inter *interpreter.Interpreter,
	parent interpreter.Value,
	typeName string,
) *interpreter.HostFunctionValue {
	compositeValue, ok := parent.(*interpreter.CompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	constructorVar := compositeValue.NestedVariables[typeName]
	constructor, ok := constructorVar.GetValue(inter).(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor"))
	}
	return constructor
}

func arrayValueToSlice(
	inter *interpreter.Interpreter,
	value interpreter.Value,
	locationRange interpreter.LocationRange,
) ([]interpreter.Value, error) {
	array, ok := value.(*interpreter.ArrayValue)
	if !ok {
		return nil, errors.NewDefaultUserError("value is not an array")
	}

	result := make([]interpreter.Value, 0, array.Count())

	array.Iterate(
		inter,
		func(element interpreter.Value) (resume bool) {
			result = append(result, element)
			return true
		},
		false,
		locationRange,
	)

	return result, nil
}

// newScriptResult Creates a "ScriptResult" using the return value of the executed script.
func newScriptResult(
	inter *interpreter.Interpreter,
	returnValue interpreter.Value,
	result *ScriptResult,
) interpreter.Value {

	if returnValue == nil {
		returnValue = interpreter.Nil
	}

	// Lookup and get 'ResultStatus' enum value.
	resultStatusConstructor := getConstructor(inter, testResultStatusTypeName)
	var status interpreter.Value
	if result.Error == nil {
		succeededVar := resultStatusConstructor.NestedVariables[testResultStatusTypeSucceededCaseName]
		status = succeededVar.GetValue(inter)
	} else {
		failedVar := resultStatusConstructor.NestedVariables[testResultStatusTypeFailedCaseName]
		status = failedVar.GetValue(inter)
	}

	errValue := newErrorValue(inter, result.Error)

	// Create a 'ScriptResult' by calling its constructor.
	scriptResultConstructor := getConstructor(inter, testScriptResultTypeName)
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
	resultStatusConstructor, ok := resultStatusConstructorVar.GetValue(inter).(*interpreter.HostFunctionValue)
	if !ok {
		panic(errors.NewUnexpectedError("invalid type for constructor of '%s'", typeName))
	}

	return resultStatusConstructor
}

func addressArrayValueToSlice(
	inter *interpreter.Interpreter,
	accountsValue interpreter.Value,
	locationRange interpreter.LocationRange,
) []common.Address {
	accountsArray, ok := accountsValue.(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	addresses := make([]common.Address, 0)

	accountsArray.Iterate(
		inter,
		func(element interpreter.Value) (resume bool) {
			address, ok := element.(interpreter.AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			addresses = append(addresses, common.Address(address))

			return true
		},
		false,
		locationRange,
	)

	return addresses
}

func accountsArrayValueToSlice(
	inter *interpreter.Interpreter,
	accountsValue interpreter.Value,
	locationRange interpreter.LocationRange,
) []*Account {

	accountsArray, ok := accountsValue.(*interpreter.ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	accounts := make([]*Account, 0)

	accountsArray.Iterate(
		inter,
		func(element interpreter.Value) (resume bool) {
			accountValue, ok := element.(interpreter.MemberAccessibleValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			account := accountFromValue(inter, accountValue, locationRange)

			accounts = append(accounts, account)

			return true
		},
		false,
		locationRange,
	)

	return accounts
}

func accountFromValue(
	inter *interpreter.Interpreter,
	accountValue interpreter.MemberAccessibleValue,
	locationRange interpreter.LocationRange,
) *Account {

	// Get address
	addressValue := accountValue.GetMember(
		inter,
		locationRange,
		accountAddressFieldName,
	)
	address, ok := addressValue.(interpreter.AddressValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// Get public key
	publicKeyVal, ok := accountValue.GetMember(
		inter,
		locationRange,
		sema.AccountKeyPublicKeyFieldName,
	).(interpreter.MemberAccessibleValue)

	if !ok {
		panic(errors.NewUnreachableError())
	}

	publicKey, err := NewPublicKeyFromValue(inter, locationRange, publicKeyVal)
	if err != nil {
		panic(err)
	}

	return &Account{
		Address:   common.Address(address),
		PublicKey: publicKey,
	}
}

// newTransactionResult Creates a "TransactionResult" indicating the status of the transaction execution.
func newTransactionResult(inter *interpreter.Interpreter, result *TransactionResult) interpreter.Value {
	// Lookup and get 'ResultStatus' enum value.
	resultStatusConstructor := getConstructor(inter, testResultStatusTypeName)
	var status interpreter.Value
	if result.Error == nil {
		succeededVar := resultStatusConstructor.NestedVariables[testResultStatusTypeSucceededCaseName]
		status = succeededVar.GetValue(inter)
	} else {
		failedVar := resultStatusConstructor.NestedVariables[testResultStatusTypeFailedCaseName]
		status = failedVar.GetValue(inter)
	}

	// Create a 'TransactionResult' by calling its constructor.
	transactionResultConstructor := getConstructor(inter, testTransactionResultTypeName)

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
		return interpreter.Nil
	}

	// Create a 'Error' by calling its constructor.
	errorConstructor := getConstructor(inter, testErrorTypeName)

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

// Creates a matcher using a function that accepts an `AnyStruct` typed parameter.
// i.e: invokes `newMatcher(fun (value: AnyStruct): Bool)`.
func newMatcherWithAnyStructTestFunction(
	invocation interpreter.Invocation,
	testFunc interpreter.FunctionValue,
) interpreter.Value {

	inter := invocation.Interpreter

	matcherConstructor := getNestedTypeConstructorValue(
		inter,
		*invocation.Self,
		testMatcherTypeName,
	)
	matcher, err := inter.InvokeExternally(
		matcherConstructor,
		matcherConstructor.Type,
		[]interpreter.Value{
			testFunc,
		},
	)

	if err != nil {
		panic(err)
	}

	return matcher
}

// Creates a matcher using a function that accepts a generic `T` typed parameter.
// NOTE: Use this function only if the matcher function has a generic type.
func newMatcherWithGenericTestFunction(
	invocation interpreter.Invocation,
	testFunc interpreter.FunctionValue,
	matcherTestFunctionType *sema.FunctionType,
) interpreter.Value {

	typeParameterPair := invocation.TypeParameterTypes.Oldest()
	if typeParameterPair == nil {
		panic(errors.NewUnreachableError())
	}

	parameterType := typeParameterPair.Value

	// Wrap the user provided test function with a function that validates the argument types.
	// i.e: create a closure that cast the arguments.
	//
	// e.g: convert `newMatcher(test: fun(Int): Bool)` to:
	//
	//  newMatcher(fun (b: AnyStruct): Bool {
	//      return test(b as! Int)
	//  })
	//
	// Note: This argument validation is only needed if the matcher was created with a user-provided function.
	// No need to validate if the matcher is created as a matcher combinator.
	//
	matcherTestFunction := interpreter.NewUnmeteredStaticHostFunctionValue(
		matcherTestFunctionType,
		func(invocation interpreter.Invocation) interpreter.Value {
			inter := invocation.Interpreter

			for _, argument := range invocation.Arguments {
				argumentStaticType := argument.StaticType(inter)

				if !inter.IsSubTypeOfSemaType(argumentStaticType, parameterType) {
					argumentSemaType := interpreter.MustConvertStaticToSemaType(argumentStaticType, inter)

					panic(interpreter.TypeMismatchError{
						ExpectedType:  parameterType,
						ActualType:    argumentSemaType,
						LocationRange: invocation.LocationRange,
					})
				}
			}

			value, err := inter.InvokeFunction(testFunc, invocation)
			if err != nil {
				panic(err)
			}

			return value
		},
	)

	return newMatcherWithAnyStructTestFunction(invocation, matcherTestFunction)
}

func TestCheckerContractValueHandler(
	checker *sema.Checker,
	declaration *ast.CompositeDeclaration,
	compositeType *sema.CompositeType,
) sema.ValueDeclaration {
	constructorType, constructorArgumentLabels := sema.CompositeLikeConstructorType(
		checker.Elaboration,
		declaration,
		compositeType,
	)

	return StandardLibraryValue{
		Name:           declaration.Identifier.Identifier,
		Type:           constructorType,
		DocString:      declaration.DocString,
		Kind:           declaration.DeclarationKind(),
		Position:       &declaration.Identifier.Pos,
		ArgumentLabels: constructorArgumentLabels,
	}
}

func NewTestInterpreterContractValueHandler(
	testFramework TestFramework,
) interpreter.ContractValueHandlerFunc {
	return func(
		inter *interpreter.Interpreter,
		compositeType *sema.CompositeType,
		constructorGenerator func(common.Address) *interpreter.HostFunctionValue,
		invocationRange ast.Range,
	) interpreter.ContractValue {

		switch compositeType.Location {
		case TestContractLocation:
			contract, err := GetTestContractType().NewTestContract(
				inter,
				testFramework,
				constructorGenerator(common.ZeroAddress),
				invocationRange,
			)
			if err != nil {
				panic(err)
			}
			return contract

		default:
			// During tests, imported contracts can be constructed using the constructor,
			// similar to structs. Therefore, generate a constructor function.
			return constructorGenerator(common.ZeroAddress)
		}
	}
}
