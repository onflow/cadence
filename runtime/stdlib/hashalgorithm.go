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

var hashAlgorithmFunctions = map[string]interpreter.FunctionValue{
	sema.HashAlgorithmTypeHashFunctionName:        hashAlgorithmHashFunction,
	sema.HashAlgorithmTypeHashWithTagFunctionName: hashAlgorithmHashWithTagFunction,
}

func NewHashAlgorithmCase(inter *interpreter.Interpreter, rawValue uint8) *interpreter.CompositeValue {
	return interpreter.NewEnumCaseValue(
		inter,
		sema.HashAlgorithmType,
		interpreter.NewUInt8Value(inter, func() uint8 {
			return rawValue
		}),
		hashAlgorithmFunctions,
	)
}

var hashAlgorithmHashFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		dataValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		hashAlgoValue := invocation.Self

		inter := invocation.Interpreter

		getLocationRange := invocation.GetLocationRange

		inter.ExpectType(
			hashAlgoValue,
			sema.HashAlgorithmType,
			getLocationRange,
		)

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

var hashAlgorithmHashWithTagFunction = interpreter.NewUnmeteredHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		dataValue, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		tagValue, ok := invocation.Arguments[1].(*interpreter.StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		hashAlgoValue := invocation.Self

		inter := invocation.Interpreter

		getLocationRange := invocation.GetLocationRange

		inter.ExpectType(
			hashAlgoValue,
			sema.HashAlgorithmType,
			getLocationRange,
		)

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

var hashAlgorithmConstructor = StandardLibraryValue{
	Name: sema.HashAlgorithmTypeName,
	Type: cryptoAlgorithmEnumConstructorType(
		sema.HashAlgorithmType,
		sema.HashAlgorithms,
	),
	ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
		return cryptoAlgorithmEnumValue(
			inter,
			interpreter.ReturnEmptyLocationRange,
			sema.HashAlgorithmType,
			sema.HashAlgorithms,
			NewHashAlgorithmCase,
		)
	},
	Kind: common.DeclarationKindEnum,
}
