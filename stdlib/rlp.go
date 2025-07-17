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

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/vm"
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

// interpreterRLPDecodeStringFunction is a static function
var interpreterRLPDecodeStringFunction = interpreter.NewUnmeteredStaticHostFunctionValue(
	RLPTypeDecodeStringFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		context := invocation.InvocationContext
		locationRange := invocation.LocationRange

		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return RLPDecodeString(
			input,
			context,
			locationRange,
		)
	},
)

var VMRLPDecodeStringFunction = VMFunction{
	BaseType: RLPType,
	FunctionValue: vm.NewNativeFunctionValue(
		RLPTypeDecodeStringFunctionName,
		RLPTypeDecodeStringFunctionType,
		func(context *vm.Context, _ []bbq.StaticType, _ vm.Value, arguments ...vm.Value) vm.Value {

			input, ok := arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return RLPDecodeString(
				input,
				context,
				interpreter.EmptyLocationRange,
			)
		},
	),
}

func RLPDecodeString(
	input *interpreter.ArrayValue,
	context interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	convertedInput, err := interpreter.ByteArrayValueToByteSlice(context, input, locationRange)
	if err != nil {
		panic(RLPDecodeStringError{
			Msg:           err.Error(),
			LocationRange: locationRange,
		})
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindSTDLIBRLPDecodeString,
			Intensity: uint64(input.Count()),
		},
	)

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

	return interpreter.ByteSliceToByteArrayValue(context, output)
}

type RLPDecodeListError struct {
	interpreter.LocationRange
	Msg string
}

var _ errors.UserError = RLPDecodeListError{}

func (RLPDecodeListError) IsUserError() {}

func (e RLPDecodeListError) Error() string {
	return fmt.Sprintf("failed to RLP-decode list: %s", e.Msg)
}

// interpreterRLPDecodeListFunction is a static function
var interpreterRLPDecodeListFunction = interpreter.NewUnmeteredStaticHostFunctionValue(
	RLPTypeDecodeListFunctionType,
	func(invocation interpreter.Invocation) interpreter.Value {
		context := invocation.InvocationContext
		locationRange := invocation.LocationRange

		input, ok := invocation.Arguments[0].(*interpreter.ArrayValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		return RLPDecodeList(
			input,
			context,
			locationRange,
		)
	},
)

var VMRLPDecodeListFunction = VMFunction{
	BaseType: RLPType,
	FunctionValue: vm.NewNativeFunctionValue(
		RLPTypeDecodeListFunctionName,
		RLPTypeDecodeListFunctionType,
		func(context *vm.Context, _ []bbq.StaticType, _ vm.Value, arguments ...vm.Value) vm.Value {

			input, ok := arguments[0].(*interpreter.ArrayValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			return RLPDecodeList(
				input,
				context,
				interpreter.EmptyLocationRange,
			)
		},
	),
}

func RLPDecodeList(
	input *interpreter.ArrayValue,
	context interpreter.InvocationContext,
	locationRange interpreter.LocationRange,
) interpreter.Value {
	convertedInput, err := interpreter.ByteArrayValueToByteSlice(context, input, locationRange)
	if err != nil {
		panic(RLPDecodeListError{
			Msg:           err.Error(),
			LocationRange: locationRange,
		})
	}

	common.UseComputation(
		context,
		common.ComputationUsage{
			Kind:      common.ComputationKindSTDLIBRLPDecodeList,
			Intensity: uint64(input.Count()),
		},
	)

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

	outputLength := len(output)

	var index int
	return interpreter.NewArrayValueWithIterator(
		context,
		interpreter.NewVariableSizedStaticType(
			context,
			interpreter.ByteArrayStaticType,
		),
		common.ZeroAddress,
		uint64(outputLength),
		func() interpreter.Value {
			if index >= outputLength {
				return nil
			}
			result := interpreter.ByteSliceToByteArrayValue(context, output[index])
			index++
			return result
		},
	)
}

var rlpContractFields = map[string]interpreter.Value{
	RLPTypeDecodeListFunctionName:   interpreterRLPDecodeListFunction,
	RLPTypeDecodeStringFunctionName: interpreterRLPDecodeStringFunction,
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
	nil,
)

var RLPContract = StandardLibraryValue{
	Name:  RLPTypeName,
	Type:  RLPType,
	Value: rlpContractValue,
	Kind:  common.DeclarationKindContract,
}
