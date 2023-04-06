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

package stdlib

import (
	"sync"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/stdlib/contracts"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const CryptoCheckerLocation = common.IdentifierLocation("Crypto")

var cryptoOnce sync.Once

// Deprecated: Use CryptoChecker instead
var cryptoChecker *sema.Checker

// Deprecated: Use CryptoContractType instead
var cryptoContractType *sema.CompositeType

// Deprecated: Use CryptoContractInitializerTypes
var cryptoContractInitializerTypes []sema.Type

func CryptoChecker() *sema.Checker {
	cryptoOnce.Do(initCrypto)
	return cryptoChecker
}

func CryptoContractType() *sema.CompositeType {
	cryptoOnce.Do(initCrypto)
	return cryptoContractType
}

func CryptoContractInitializerTypes() []sema.Type {
	cryptoOnce.Do(initCrypto)
	return cryptoContractInitializerTypes
}

func initCrypto() {
	program, err := parser.ParseProgram(
		nil,
		contracts.Crypto,
		parser.Config{},
	)
	if err != nil {
		panic(err)
	}

	cryptoChecker, err = sema.NewChecker(
		program,
		CryptoCheckerLocation,
		nil,
		&sema.Config{
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
	)
	if err != nil {
		panic(err)
	}

	err = cryptoChecker.Check()
	if err != nil {
		panic(err)
	}

	variable, ok := cryptoChecker.Elaboration.GetGlobalType("Crypto")
	if !ok {
		panic(errors.NewUnreachableError())
	}
	cryptoContractType = variable.Type.(*sema.CompositeType)

	cryptoContractInitializerTypes = make([]sema.Type, len(cryptoContractType.ConstructorParameters))
	for i, parameter := range cryptoContractType.ConstructorParameters {
		cryptoContractInitializerTypes[i] = parameter.TypeAnnotation.Type
	}
}

func NewCryptoContract(
	inter *interpreter.Interpreter,
	constructor interpreter.FunctionValue,
	invocationRange ast.Range,
) (
	*interpreter.CompositeValue,
	error,
) {
	initializerTypes := CryptoContractInitializerTypes()
	value, err := inter.InvokeFunctionValue(
		constructor,
		nil,
		initializerTypes,
		initializerTypes,
		invocationRange,
	)
	if err != nil {
		return nil, err
	}

	compositeValue := value.(*interpreter.CompositeValue)

	return compositeValue, nil
}

func cryptoAlgorithmEnumConstructorType[T sema.CryptoAlgorithm](
	enumType *sema.CompositeType,
	enumCases []T,
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
		Parameters: []sema.Parameter{
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
		Members: sema.MembersAsMap(members),
	}

	return constructorType
}

type enumCaseConstructor func(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue

func cryptoAlgorithmEnumValueAndCaseValues[T sema.CryptoAlgorithm](
	enumType *sema.CompositeType,
	enumCases []T,
	caseConstructor enumCaseConstructor,
) (
	value interpreter.Value,
	cases map[interpreter.UInt8Value]interpreter.MemberAccessibleValue,
) {

	caseCount := len(enumCases)
	caseValues := make([]interpreter.EnumCase, caseCount)
	constructorNestedVariables := make(map[string]*interpreter.Variable, caseCount)
	cases = make(map[interpreter.UInt8Value]interpreter.MemberAccessibleValue, caseCount)

	for i, enumCase := range enumCases {
		rawValue := interpreter.UInt8Value(enumCase.RawValue())
		caseValue := caseConstructor(rawValue)
		cases[rawValue] = caseValue
		caseValues[i] = interpreter.EnumCase{
			Value:    caseValue,
			RawValue: rawValue,
		}
		constructorNestedVariables[enumCase.Name()] =
			interpreter.NewVariableWithValue(nil, caseValue)
	}

	value = interpreter.EnumConstructorFunction(
		nil,
		interpreter.EmptyLocationRange,
		enumType,
		caseValues,
		constructorNestedVariables,
	)

	return
}
