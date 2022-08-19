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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

const publicKeyConstructorFunctionDocString = `
Constructs a new public key
`

var publicKeyConstructorFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Identifier:     sema.PublicKeyPublicKeyField,
			TypeAnnotation: sema.NewTypeAnnotation(sema.ByteArrayType),
		},
		{
			Identifier:     sema.PublicKeySignAlgoField,
			TypeAnnotation: sema.NewTypeAnnotation(sema.SignatureAlgorithmType),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.PublicKeyType),
}

var PublicKeyConstructor = NewStandardLibraryFunction(
	sema.PublicKeyTypeName,
	publicKeyConstructorFunctionType,
	publicKeyConstructorFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		publicKey, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		signAlgo, ok := invocation.Arguments[1].(*interpreter.SimpleCompositeValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		inter := invocation.Interpreter

		return interpreter.NewPublicKeyValue(
			inter,
			invocation.GetLocationRange,
			publicKey,
			signAlgo,
			inter.PublicKeyValidationHandler,
		)
	},
)
