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

//go:generate go run ../sema/gen -p stdlib rlp.cdc rlp.gen.go

import (
	"fmt"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/stdlib/rlp"
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

const rlpErrMsgInputContainsExtraBytes = "input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes."

// rlpDecodeStringFunction is a static function
var rlpDecodeStringFunction = interpreter.NewUnmeteredStaticHostFunctionValue(
	RLPTypeDecodeStringFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		invocation.InvocationContext.ReportComputation(common.ComputationKindSTDLIBRLPDecodeString, uint(input.Count()))

		locationRange := invocation.LocationRange

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(invocation.InvocationContext, input, locationRange)
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
		return interpreter.ByteSliceToByteArrayValue(invocation.InvocationContext, output)
	},
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

// rlpDecodeListFunction is a static function
var rlpDecodeListFunction = interpreter.NewUnmeteredStaticHostFunctionValue(
	RLPTypeDecodeListFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		invocation.InvocationContext.ReportComputation(common.ComputationKindSTDLIBRLPDecodeList, uint(input.Count()))

		locationRange := invocation.LocationRange

		convertedInput, err := interpreter.ByteArrayValueToByteSlice(invocation.InvocationContext, input, locationRange)
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
			values[i] = interpreter.ByteSliceToByteArrayValue(invocation.InvocationContext, b)
		}

		return interpreter.NewArrayValue(
			invocation.InvocationContext,
			locationRange,
			interpreter.NewVariableSizedStaticType(
				invocation.InvocationContext,
				interpreter.ByteArrayStaticType,
			),
			common.ZeroAddress,
			values...,
		)
	},
)

var rlpContractFields = map[string]interpreter.Value{
	RLPTypeDecodeListFunctionName:   rlpDecodeListFunction,
	RLPTypeDecodeStringFunctionName: rlpDecodeStringFunction,
}

var RLPTypeStaticType = interpreter.ConvertSemaToStaticType(nil, RLPType)

var rlpContractValue = interpreter.NewSimpleCompositeValue(
	nil,
	RLPType.ID(),
	RLPTypeStaticType,
	nil,
	rlpContractFields,
	nil,
	nil,
	nil,
)

var RLPContract = StandardLibraryValue{
	Name:  RLPTypeName,
	Type:  RLPType,
	Value: rlpContractValue,
	Kind:  common.DeclarationKindContract,
}
