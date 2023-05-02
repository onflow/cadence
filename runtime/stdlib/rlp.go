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

	ty.Members = sema.MembersAsMap([]*sema.Member{
		sema.NewUnmeteredPublicFunctionMember(
			ty,
			rlpDecodeListFunctionName,
			rlpDecodeListFunctionType,
			rlpDecodeListFunctionDocString,
		),
		sema.NewUnmeteredPublicFunctionMember(
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

var rlpDecodeStringFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "input",
			TypeAnnotation: sema.ByteArrayTypeAnnotation,
		},
	},
	sema.ByteArrayTypeAnnotation,
)

type RLPDecodeStringError struct {
	interpreter.LocationRange
	Msg string
}

var _ errors.UserError = RLPDecodeStringError{}

func (RLPDecodeStringError) IsUserError() {}

func (e RLPDecodeStringError) Error() string {
	return fmt.Sprintf("failed to RLP-decode string: %s", e.Msg)
}

var rlpDecodeStringFunction = interpreter.NewUnmeteredHostFunctionValue(
	rlpDecodeStringFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		invocation.Interpreter.ReportComputation(common.ComputationKindSTDLIBRLPDecodeString, uint(input.Count()))

		locationRange := invocation.LocationRange

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(invocation.Interpreter, input, locationRange)
		if err != nil {
			panic(RLPDecodeStringError{
				Msg:           err.Error(),
				LocationRange: locationRange,
			})
		}
		output, bytesRead, err := rlp.DecodeString(convertedInput, 0)
		if err != nil {
			panic(RLPDecodeStringError{
				Msg:           err.Error(),
				LocationRange: locationRange,
			})
		}
		if bytesRead != len(convertedInput) {
			panic(RLPDecodeStringError{
				Msg:           rlpErrMsgInputContainsExtraBytes,
				LocationRange: locationRange,
			})
		}
		return interpreter.ByteSliceToByteArrayValue(invocation.Interpreter, output)
	},
)

const rlpDecodeListFunctionDocString = `
Decodes an RLP-encoded list into an array of RLP-encoded items.
Note that this function does not recursively decode, so each element of the resulting array is RLP-encoded data. 
The byte array should only contain of a single encoded value for a list;
if the encoded value type does not match, or it has trailing unnecessary bytes, the program aborts.
If any error is encountered while decoding, the program aborts.
`

const rlpDecodeListFunctionName = "decodeList"

var rlpDecodeListFunctionType = sema.NewSimpleFunctionType(
	sema.FunctionPurityView,
	[]sema.Parameter{
		{
			Label:          sema.ArgumentLabelNotRequired,
			Identifier:     "input",
			TypeAnnotation: sema.ByteArrayTypeAnnotation,
		},
	},
	sema.ByteArrayArrayTypeAnnotation,
)

type RLPDecodeListError struct {
	interpreter.LocationRange
	Msg string
}

var _ errors.UserError = RLPDecodeListError{}

func (RLPDecodeListError) IsUserError() {}

func (e RLPDecodeListError) Error() string {
	return fmt.Sprintf("failed to RLP-decode list: %s", e.Msg)
}

var rlpDecodeListFunction = interpreter.NewUnmeteredHostFunctionValue(
	rlpDecodeListFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		invocation.Interpreter.ReportComputation(common.ComputationKindSTDLIBRLPDecodeList, uint(input.Count()))

		locationRange := invocation.LocationRange

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(invocation.Interpreter, input, locationRange)
		if err != nil {
			panic(RLPDecodeListError{
				Msg:           err.Error(),
				LocationRange: locationRange,
			})
		}

		output, bytesRead, err := rlp.DecodeList(convertedInput, 0)

		if err != nil {
			panic(RLPDecodeListError{
				Msg:           err.Error(),
				LocationRange: locationRange,
			})
		}

		if bytesRead != len(convertedInput) {
			panic(RLPDecodeListError{
				Msg:           rlpErrMsgInputContainsExtraBytes,
				LocationRange: locationRange,
			})
		}

		values := make([]interpreter.Value, len(output))
		for i, b := range output {
			values[i] = interpreter.ByteSliceToByteArrayValue(invocation.Interpreter, b)
		}

		return interpreter.NewArrayValue(
			invocation.Interpreter,
			locationRange,
			interpreter.NewVariableSizedStaticType(
				invocation.Interpreter,
				interpreter.ByteArrayStaticType,
			),
			common.ZeroAddress,
			values...,
		)
	},
)

var rlpContractFields = map[string]interpreter.Value{
	rlpDecodeListFunctionName:   rlpDecodeListFunction,
	rlpDecodeStringFunctionName: rlpDecodeStringFunction,
}

var rlpContractValue = interpreter.NewSimpleCompositeValue(
	nil,
	rlpContractType.ID(),
	rlpContractStaticType,
	nil,
	rlpContractFields,
	nil,
	nil,
	nil,
)

var RLPContract = StandardLibraryValue{
	Name:  "RLP",
	Type:  rlpContractType,
	Value: rlpContractValue,
	Kind:  common.DeclarationKindContract,
}
