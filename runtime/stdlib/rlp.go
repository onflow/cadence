/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/rlp"
)

// This file defines functions built in to the Flow runtime.

const rlpDecodeStringFunctionDocString = `
 accepts an RLP encoded byte array and decodes it into an string.
 `

var rlpDecodeStringFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Identifier: "input",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.ByteArrayType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.ByteArrayType,
	),
}

const rlpDecodeListFunctionDocString = `
 accepts an RLP encoded byte array and decodes it into an array of encoded elements.
 `

var rlpDecodeListFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Identifier: "input",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.ByteArrayType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		&sema.VariableSizedType{
			Type: sema.ByteArrayType,
		},
	),
}

var RLPDecodeStringFunction = NewStandardLibraryFunction(
	"RLPDecodeString",
	rlpDecodeStringFunctionType,
	rlpDecodeStringFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		input := invocation.Arguments[0].(*interpreter.ArrayValue)

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(err)
		}
		output, err := rlp.DecodeString(convertedInput, 0)
		if err != nil {
			panic(err)
		}
		return interpreter.ByteSliceToByteArrayValue(invocation.Interpreter, output)
	},
)

var RLPDecodeListFunction = NewStandardLibraryFunction(
	"RLPDecodeList",
	rlpDecodeListFunctionType,
	rlpDecodeListFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		input := invocation.Arguments[0].(*interpreter.ArrayValue)

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(err)
		}

		output, err := rlp.DecodeList(convertedInput, 0)
		if err != nil {
			panic(err)
		}

		values := make([]interpreter.Value, len(output))
		for i, b := range output {
			values[i] = interpreter.ByteSliceToByteArrayValue(invocation.Interpreter, b)
		}

		return interpreter.NewArrayValue(
			invocation.Interpreter,
			interpreter.VariableSizedStaticType{
				Type: interpreter.ByteArrayStaticType,
			},
			common.Address{},
			values...,
		)
	},
)
