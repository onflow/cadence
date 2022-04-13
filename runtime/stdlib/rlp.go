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
	"fmt"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib/rlp"
)

var rlpContractType = func() *sema.CompositeType {
	ty := &sema.CompositeType{
		Identifier: "RLP",
		Kind:       common.CompositeKindContract,
	}

	ty.Members = sema.GetMembersAsMap([]*sema.Member{
		sema.NewPublicFunctionMember(
			ty,
			rlpDecodeListFunctionName,
			rlpDecodeListFunctionType,
			rlpDecodeListFunctionDocString,
		),
		sema.NewPublicFunctionMember(
			ty,
			rlpDecodeStringFunctionName,
			rlpDecodeStringFunctionType,
			rlpDecodeStringFunctionDocString,
		),
	})
	return ty
}()

var rlpContractTypeID = rlpContractType.ID()
var rlpContractStaticType interpreter.StaticType = interpreter.CompositeStaticType{
	QualifiedIdentifier: rlpContractType.Identifier,
	TypeID:              rlpContractTypeID,
}

const rlpErrMsgInputContainsExtraBytes = "input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes."

const rlpDecodeStringFunctionDocString = `
Decodes an RLP-encoded byte array (called string in the context of RLP). 
The byte array should only contain of a single encoded value for a string;
if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
If any error is encountered while decoding, the program aborts.
`

const rlpDecodeStringFunctionName = "decodeString"

var rlpDecodeStringFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
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
	interpreter.LocationRange
}

func (e RLPDecodeStringError) Error() string {
	return fmt.Sprintf("failed to RLP-decode string: %s", e.Msg)
}

var rlpDecodeStringFunction = interpreter.NewHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		invocation.Interpreter.ReportComputation(common.ComputationKindSTDLIBRLPDecodeString, uint(input.Count()))

		getLocationRange := invocation.GetLocationRange

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(RLPDecodeStringError{
				Msg:           err.Error(),
				LocationRange: getLocationRange(),
			})
		}
		output, bytesRead, err := rlp.DecodeString(convertedInput, 0)
		if err != nil {
			panic(RLPDecodeStringError{
				Msg:           err.Error(),
				LocationRange: getLocationRange(),
			})
		}
		if bytesRead != len(convertedInput) {
			panic(RLPDecodeStringError{
				Msg:           rlpErrMsgInputContainsExtraBytes,
				LocationRange: getLocationRange(),
			})
		}
		return interpreter.ByteSliceToByteArrayValue(invocation.Interpreter, output)
	},
	rlpDecodeStringFunctionType,
)

const rlpDecodeListFunctionDocString = `
Decodes an RLP-encoded list into an array of RLP-encoded items.
Note that this function does not recursively decode, so each element of the resulting array is RLP-encoded data. 
The byte array should only contain of a single encoded value for a list;
if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
If any error is encountered while decoding, the program aborts.
`

const rlpDecodeListFunctionName = "decodeList"

var rlpDecodeListFunctionType = &sema.FunctionType{
	Parameters: []*sema.Parameter{
		{
			Label:      sema.ArgumentLabelNotRequired,
			Identifier: "input",
			TypeAnnotation: sema.NewTypeAnnotation(
				sema.ByteArrayType,
			),
		},
	},
	ReturnTypeAnnotation: sema.NewTypeAnnotation(
		sema.ByteArrayArrayType,
	),
}

type RLPDecodeListError struct {
	Msg string
	interpreter.LocationRange
}

func (e RLPDecodeListError) Error() string {
	return fmt.Sprintf("failed to RLP-decode list: %s", e.Msg)
}

var rlpDecodeListFunction = interpreter.NewHostFunctionValue(
	func(invocation interpreter.Invocation) interpreter.Value {
		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		invocation.Interpreter.ReportComputation(common.ComputationKindSTDLIBRLPDecodeList, uint(input.Count()))

		getLocationRange := invocation.GetLocationRange

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(input)
		if err != nil {
			panic(RLPDecodeListError{
				Msg:           err.Error(),
				LocationRange: getLocationRange(),
			})
		}

		output, bytesRead, err := rlp.DecodeList(convertedInput, 0)

		if err != nil {
			panic(RLPDecodeListError{
				Msg:           err.Error(),
				LocationRange: getLocationRange(),
			})
		}

		if bytesRead != len(convertedInput) {
			panic(RLPDecodeListError{
				Msg:           rlpErrMsgInputContainsExtraBytes,
				LocationRange: getLocationRange(),
			})
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
	rlpDecodeListFunctionType,
)

var rlpContractFields = map[string]interpreter.Value{
	rlpDecodeListFunctionName:   rlpDecodeListFunction,
	rlpDecodeStringFunctionName: rlpDecodeStringFunction,
}

var rlpContract = StandardLibraryValue{
	Name: "RLP",
	Type: rlpContractType,
	ValueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
		return interpreter.NewSimpleCompositeValue(
			rlpContractType.ID(),
			rlpContractStaticType,
			nil,
			rlpContractFields,
			nil,
			nil,
			nil,
		)
	},
	Kind: common.DeclarationKindContract,
}
