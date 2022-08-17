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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	errors2 "github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/contracts"
)

var CryptoChecker = func() *sema.Checker {

	program, err := parser.ParseProgram(contracts.Crypto, nil)
	if err != nil {
		panic(err)
	}

	location := common.IdentifierLocation("Crypto")

	var checker *sema.Checker
	checker, err = sema.NewChecker(
		program,
		location,
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

var cryptoContractType = func() *sema.CompositeType {
	variable, ok := CryptoChecker.Elaboration.GlobalTypes.Get("Crypto")
	if !ok {
		panic(errors2.NewUnreachableError())
	}
	return variable.Type.(*sema.CompositeType)
}()

var cryptoContractInitializerTypes = func() (result []sema.Type) {
	result = make([]sema.Type, len(cryptoContractType.ConstructorParameters))
	for i, parameter := range cryptoContractType.ConstructorParameters {
		result[i] = parameter.TypeAnnotation.Type
	}
	return result
}()

func NewCryptoContract(
	inter *interpreter.Interpreter,
	constructor interpreter.FunctionValue,
	invocationRange ast.Range,
) (
	*interpreter.CompositeValue,
	error,
) {
	value, err := inter.InvokeFunctionValue(
		constructor,
		nil,
		cryptoContractInitializerTypes,
		cryptoContractInitializerTypes,
		invocationRange,
	)
	if err != nil {
		return nil, err
	}

	compositeValue := value.(*interpreter.CompositeValue)

	return compositeValue, nil
}

func cryptoAlgorithmEnumConstructorType(
	enumType *sema.CompositeType,
	enumCases []sema.CryptoAlgorithm,
) *sema.FunctionType {

	members := make([]*sema.Member, len(enumCases))
	for i, algo := range enumCases {
		members[i] = sema.NewUnmeteredPublicConstantFieldMember(
			enumType,
			algo.Name(),
			enumType,
			algo.DocString(),
		)
	}

	constructorType := &sema.FunctionType{
		IsConstructor: true,
		Parameters: []*sema.Parameter{
			{
				Identifier:     sema.EnumRawValueFieldName,
				TypeAnnotation: sema.NewTypeAnnotation(enumType.EnumRawType),
			},
		},
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			&sema.OptionalType{
				Type: enumType,
			},
		),
		Members: sema.GetMembersAsMap(members),
	}

	return constructorType
}

type enumCaseConstructor func(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue

func cryptoAlgorithmEnumValueAndCaseValues(
	enumType *sema.CompositeType,
	enumCases []sema.CryptoAlgorithm,
	caseConstructor enumCaseConstructor,
) (
	value interpreter.Value,
	cases map[interpreter.UInt8Value]interpreter.MemberAccessibleValue,
) {

	caseCount := len(enumCases)
	caseValues := make([]struct {
		Value    interpreter.MemberAccessibleValue
		RawValue interpreter.IntegerValue
	}, caseCount)
	constructorNestedVariables := make(map[string]*interpreter.Variable, caseCount)
	cases = make(map[interpreter.UInt8Value]interpreter.MemberAccessibleValue, caseCount)

	for i, enumCase := range enumCases {
		rawValue := interpreter.UInt8Value(enumCase.RawValue())
		caseValue := caseConstructor(rawValue)
		cases[rawValue] = caseValue
		caseValues[i] = struct {
			Value    interpreter.MemberAccessibleValue
			RawValue interpreter.IntegerValue
		}{
			Value:    caseValue,
			RawValue: rawValue,
		}
		constructorNestedVariables[enumCase.Name()] =
			interpreter.NewVariableWithValue(nil, caseValue)
	}

	value = interpreter.EnumConstructorFunction(
		nil,
		interpreter.ReturnEmptyLocationRange,
		enumType,
		caseValues,
		constructorNestedVariables,
	)

	return
}
