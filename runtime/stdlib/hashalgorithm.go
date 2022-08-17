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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var hashAlgorithmTypeID = sema.HashAlgorithmType.ID()
var hashAlgorithmStaticType interpreter.StaticType = interpreter.CompositeStaticType{
	QualifiedIdentifier: sema.HashAlgorithmType.Identifier,
	TypeID:              hashAlgorithmTypeID,
}

func NewHashAlgorithmCase(rawValue interpreter.UInt8Value) interpreter.MemberAccessibleValue {

	value := interpreter.NewSimpleCompositeValue(
		nil,
		sema.HashAlgorithmType.ID(),
		hashAlgorithmStaticType,
		[]string{sema.EnumRawValueFieldName},
		nil,
		nil,
		nil,
		nil,
	)
	value.Fields = map[string]interpreter.Value{
		sema.EnumRawValueFieldName:                    rawValue,
		sema.HashAlgorithmTypeHashFunctionName:        hashAlgorithmHashFunction(value),
		sema.HashAlgorithmTypeHashWithTagFunctionName: hashAlgorithmHashWithTagFunction(value),
	}
	return value
}

func hashAlgorithmHashFunction(hashAlgoValue interpreter.MemberAccessibleValue) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			dataValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter

			getLocationRange := invocation.GetLocationRange

			return inter.HashHandler(
				inter,
				getLocationRange,
				dataValue,
				nil,
				hashAlgoValue,
			)
		},
		sema.HashAlgorithmTypeHashFunctionType,
	)
}

func hashAlgorithmHashWithTagFunction(hashAlgoValue interpreter.MemberAccessibleValue) *interpreter.HostFunctionValue {
	return interpreter.NewUnmeteredHostFunctionValue(
		func(invocation interpreter.Invocation) interpreter.Value {
			dataValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			tagValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter

			getLocationRange := invocation.GetLocationRange

			return inter.HashHandler(
				inter,
				getLocationRange,
				dataValue,
				tagValue,
				hashAlgoValue,
			)
		},
		sema.HashAlgorithmTypeHashWithTagFunctionType,
	)
}

var hashAlgorithmConstructorValue, HashAlgorithmCaseValues = cryptoAlgorithmEnumValueAndCaseValues(
	sema.HashAlgorithmType,
	sema.HashAlgorithms,
	NewHashAlgorithmCase,
)

var hashAlgorithmConstructor = StandardLibraryValue{
	Name: sema.HashAlgorithmTypeName,
	Type: cryptoAlgorithmEnumConstructorType(
		sema.HashAlgorithmType,
		sema.HashAlgorithms,
	),
	ValueFactory: func(_ *interpreter.Interpreter) interpreter.Value {
		return hashAlgorithmConstructorValue
	},
	Kind: common.DeclarationKindEnum,
}
