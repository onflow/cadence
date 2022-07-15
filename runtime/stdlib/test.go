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
	"github.com/onflow/cadence/runtime/sema"
)

var TestContractLocation = common.IdentifierLocation("Test")

var TestContractChecker = func() *sema.Checker {
	checker, err := sema.NewChecker(
		ast.NewProgram(nil, nil),
		TestContractLocation,
		nil,
		false,
		sema.WithPredeclaredValues(BuiltinFunctions.ToSemaValueDeclarations()),
		sema.WithPredeclaredTypes(BuiltinTypes.ToTypeDeclarations()),
	)

	if err != nil {
		panic(err)
	}

	checker.Elaboration.CompositeTypes[testBlockchainType.ID()] = testBlockchainType

	return checker
}()

const testContractTypeName = "Test"

var testContractType = func() *sema.CompositeType {
	ty := &sema.CompositeType{
		Identifier: testContractTypeName,
		Kind:       common.CompositeKindContract,
	}

	ty.Members = sema.GetMembersAsMap([]*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			testAssertFunctionName,
			testAssertFunctionType,
			testAssertFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			blockchainTypeName,
			blockchainConstructorType,
			blockchainConstructorDocString,
		),
	})

	nestedTypes := &sema.StringTypeOrderedMap{}
	nestedTypes.Set(blockchainTypeName, testBlockchainType)
	ty.NestedTypes = nestedTypes

	return ty
}()

var testContract = StandardLibraryValue{
	Name: testContractTypeName,
	Type: testContractType,
	ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
		return interpreter.NewSimpleCompositeValue(
			inter,
			testContractType.ID(),
			testContractStaticType,
			nil,
			testContractFields,
			nil,
			nil,
			nil,
		)
	},
	Kind: common.DeclarationKindContract,
}

var testContractFields = map[string]interpreter.Value{
	testAssertFunctionName: testAssertFunction,
	blockchainTypeName:     blockchainConstructor,
}

var testContractTypeID = testContractType.ID()
var testContractStaticType interpreter.StaticType = interpreter.CompositeStaticType{
	QualifiedIdentifier: testContractType.Identifier,
	TypeID:              testContractTypeID,
}

// Functions belong to the 'Test' contract

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

// 'Blockchain' struct

const blockchainTypeName = "Blockchain"

var testBlockchainType = func() *sema.CompositeType {

	ty := &sema.CompositeType{
		Identifier: blockchainTypeName,
		Kind:       common.CompositeKindStructure,
		Location:   TestContractLocation,
	}

	var members = []*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			blockchainExecuteScriptFunctionName,
			blockchainExecuteScriptFunctionType,
			blockchainExecuteScriptFunctionDocString,
		),
	}

	ty.Members = sema.GetMembersAsMap(members)
	ty.Fields = sema.GetFieldNames(members)
	return ty
}()

// Functions belong to the 'Blockchain' struct

// Blockchain constructor

const blockchainConstructorDocString = `This is the Blockchain constructor`

var blockchainConstructorType = &sema.FunctionType{
	Parameters: []*sema.Parameter{},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		testBlockchainType,
	),
}

var blockchainConstructor = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		var fields = []interpreter.CompositeField{
			{
				Name:  blockchainExecuteScriptFunctionName,
				Value: blockchainExecuteScriptFunction,
			},
		}

		blockchain := interpreter.NewCompositeValue(
			invocation.Interpreter,
			interpreter.ReturnEmptyLocationRange,
			common.IdentifierLocation(testContractTypeID),
			blockchainTypeName,
			common.CompositeKindStructure,
			fields,
			common.Address{},
		)

		return blockchain
	},
	testAssertFunctionType,
)

// Execute script function

const blockchainExecuteScriptFunctionName = "executeScript"
const blockchainExecuteScriptFunctionDocString = `execute script function`

var blockchainExecuteScriptFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "script",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.StringType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.VoidType,
	),
}

var blockchainExecuteScriptFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		scriptString, ok := invocation.Arguments[0].(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Strip off the starting and ending double quotes
		script := scriptString.String()
		script = script[1 : len(script)-1]

		var result interpreter.ScriptResult

		testFramework := invocation.Interpreter.TestFramework
		if testFramework != nil {
			result = testFramework.RunScript(script)
		} else {
			panic(interpreter.TestFrameworkNotProvidedError{})
		}

		err := result.Error
		if err != nil {
			// TODO: Revisit this logic
			if errors.IsUserError(err) {
				panic(TestFailedError{
					Err: err,
				})
			} else {
				panic(err)
			}
		}

		return result.Value
	},
	blockchainExecuteScriptFunctionType,
)

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
