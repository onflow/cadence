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

func cryptoAlgorithmEnumLookupType[T sema.CryptoAlgorithm](
	enumType *sema.CompositeType,
	enumCases []T,
) *sema.FunctionType {

	functionType := sema.EnumLookupFunctionType(enumType)

	for _, algo := range enumCases {
		name := algo.Name()
		functionType.Members.Set(
			name,
			sema.NewUnmeteredPublicConstantFieldMember(
				enumType,
				name,
				enumType,
				algo.DocString(),
			),
		)
	}

	return functionType
}

type enumCaseConstructor func(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue

func cryptoAlgorithmEnumValueAndCaseValues[T sema.CryptoAlgorithm](
	functionType *sema.FunctionType,
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

	value = interpreter.EnumLookupFunction(
		nil,
		functionType,
		caseValues,
		constructorNestedVariables,
	)

	return
}
