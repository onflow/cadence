/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2022 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/rlp"
)

const DecodeRLPStringFunctionDocString = `
Decodes an RLP-encoded byte array (called string in the context of RLP). 
The byte array should only contain of a single encoded value for a string; if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
If any error is encountered while decoding, the program aborts.
`

var DecodeRLPStringFunctionType = &sema.FunctionType{
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

type DecodeRLPStringError struct {
	Msg string
}

func (e DecodeRLPStringError) Error() string {
	return fmt.Sprintf("failed to RLP-decode string: %s", e.Msg)
}

var DecodeRLPStringFunction = NewStandardLibraryFunction(
	"DecodeRLPString",
	DecodeRLPStringFunctionType,
	DecodeRLPStringFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		input := invocation.Arguments[0].(*interpreter.ArrayValue)

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(DecodeRLPStringError{err.Error()})
		}
		output, err := rlp.DecodeString(convertedInput, 0)
		if err != nil {
			panic(DecodeRLPStringError{err.Error()})
		}
		return interpreter.ByteSliceToByteArrayValue(invocation.Interpreter, output)
	},
)

const DecodeRLPListFunctionDocString = `
Decodes an RLP-encoded list into an array of RLP-encoded items.
Note that this function does not recursively decode, so each element of the resulting array is RLP-encoded data. 
The byte array should only contain of a single encoded value for a list; if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
If any error is encountered while decoding, the program aborts.
`

var DecodeRLPListFunctionType = &sema.FunctionType{
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

type DecodeRLPListError struct {
	Msg string
}

func (e DecodeRLPListError) Error() string {
	return fmt.Sprintf("failed to RLP-decode list: %s", e.Msg)
}

var DecodeRLPListFunction = NewStandardLibraryFunction(
	"DecodeRLPList",
	DecodeRLPListFunctionType,
	DecodeRLPListFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		input := invocation.Arguments[0].(*interpreter.ArrayValue)

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(DecodeRLPListError{err.Error()})
		}

		output, err := rlp.DecodeList(convertedInput, 0)
		if err != nil {
			panic(DecodeRLPListError{err.Error()})
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
