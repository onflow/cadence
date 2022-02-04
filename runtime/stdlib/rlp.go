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
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/rlp"
)

const rlpDecodeStringFunctionDocString = `
 Accepts an RLP encoded byte array and decodes it into an string. 
 Input should only contain a single encoded value for an string;
 if the encoded value type doesn't match or it has trailing unnecessary bytes it would error out.
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

type RLPDecodeStringError struct {
	Msg string
}

func (e RLPDecodeStringError) Error() string {
	return fmt.Sprintf("rlpDecodeString has Failed: %s", e.Msg)
}

var RLPDecodeStringFunction = NewStandardLibraryFunction(
	"rlpDecodeString",
	rlpDecodeStringFunctionType,
	rlpDecodeStringFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		input := invocation.Arguments[0].(*interpreter.ArrayValue)

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(&RLPDecodeStringError{err.Error()})
		}
		output, err := rlp.DecodeString(convertedInput, 0)
		if err != nil {
			panic(&RLPDecodeStringError{err.Error()})
		}
		return interpreter.ByteSliceToByteArrayValue(invocation.Interpreter, output)
	},
)

const rlpDecodeListFunctionDocString = `
 Accepts an RLP encoded byte array and decodes it into an array of encoded elements, 
 note that this method does not do the recursive decoding so each array element would be an RLP encoded byte array, 
 which again can be decoded by calling 'RLPDecodeString' or 'RLPDecodeList'.
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

type RLPDecodeListError struct {
	Msg string
}

func (e RLPDecodeListError) Error() string {
	return fmt.Sprintf("rlpDecodeList has Failed: %s", e.Msg)
}

var RLPDecodeListFunction = NewStandardLibraryFunction(
	"rlpDecodeList",
	rlpDecodeListFunctionType,
	rlpDecodeListFunctionDocString,
	func(invocation interpreter.Invocation) interpreter.Value {
		input := invocation.Arguments[0].(*interpreter.ArrayValue)

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(&RLPDecodeListError{err.Error()})
		}

		output, err := rlp.DecodeList(convertedInput, 0)
		if err != nil {
			panic(&RLPDecodeListError{err.Error()})
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
