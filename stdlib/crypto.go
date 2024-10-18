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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

const CryptoContractLocation = common.IdentifierLocation("Crypto")

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

	return &sema.FunctionType{
		Purity:        sema.FunctionPurityView,
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
	constructorNestedVariables := make(map[string]interpreter.Variable, caseCount)
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
