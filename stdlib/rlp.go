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

var _ errors.UserError = &RLPDecodeStringError{}
var _ interpreter.HasLocationRange = &RLPDecodeStringError{}

func (*RLPDecodeStringError) IsUserError() {}

func (e *RLPDecodeStringError) Error() string {
	return fmt.Sprintf("failed to RLP-decode string: %s", e.Msg)
}

func (e *RLPDecodeStringError) SetLocationRange(locationRange interpreter.LocationRange) {
	e.LocationRange = locationRange
}

const rlpErrMsgInputContainsExtraBytes = "input data is expected to be RLP-encoded of a single string or a single list but it seems it contains extra trailing bytes."

// Native RLP functions
var NativeRLPDecodeStringFunction = interpreter.NativeFunction(
	func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		args []interpreter.Value,
	) interpreter.Value {
		input := interpreter.AssertValueOfType[*interpreter.ArrayValue](args[0])
		return RLPDecodeString(input, context)
	},
)

var NativeRLPDecodeListFunction = interpreter.NativeFunction(
	func(
		context interpreter.NativeFunctionContext,
		_ interpreter.TypeArgumentsIterator,
		_ interpreter.ArgumentTypesIterator,
		_ interpreter.Value,
		args []interpreter.Value,
	) interpreter.Value {
		input := interpreter.AssertValueOfType[*interpreter.ArrayValue](args[0])
		return RLPDecodeList(input, context)
	},
)

// interpreterRLPDecodeStringFunction is a static function
var interpreterRLPDecodeStringFunction = interpreter.NewUnmeteredStaticHostFunctionValueFromNativeFunction(
	RLPTypeDecodeStringFunctionType,
	NativeRLPDecodeStringFunction,
)

var VMRLPDecodeStringFunction = VMFunction{
	BaseType: RLPType,
	FunctionValue: vm.NewNativeFunctionValue(
		RLPTypeDecodeStringFunctionName,
		RLPTypeDecodeStringFunctionType,
		NativeRLPDecodeStringFunction,
	),
}

func RLPDecodeString(
	input *interpreter.ArrayValue,
	context interpreter.InvocationContext,
) interpreter.Value {
	convertedInput, err := interpreter.ByteArrayValueToByteSlice(context, input)
	if err != nil {
		panic(&RLPDecodeStringError{
			Msg: err.Error(),
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
		panic(&RLPDecodeStringError{
			Msg: err.Error(),
		})
	}

	if bytesRead != len(convertedInput) {
		panic(&RLPDecodeStringError{
			Msg: rlpErrMsgInputContainsExtraBytes,
		})
	}

	return interpreter.ByteSliceToByteArrayValue(context, output)
}

type RLPDecodeListError struct {
	interpreter.LocationRange
	Msg string
}

var _ errors.UserError = &RLPDecodeListError{}
var _ interpreter.HasLocationRange = &RLPDecodeListError{}

func (*RLPDecodeListError) IsUserError() {}

func (e *RLPDecodeListError) Error() string {
	return fmt.Sprintf("failed to RLP-decode list: %s", e.Msg)
}

func (e *RLPDecodeListError) SetLocationRange(locationRange interpreter.LocationRange) {
	e.LocationRange = locationRange
}

// interpreterRLPDecodeListFunction is a static function
var interpreterRLPDecodeListFunction = interpreter.NewUnmeteredStaticHostFunctionValueFromNativeFunction(
	RLPTypeDecodeListFunctionType,
	NativeRLPDecodeListFunction,
)

var VMRLPDecodeListFunction = VMFunction{
	BaseType: RLPType,
	FunctionValue: vm.NewNativeFunctionValue(
		RLPTypeDecodeListFunctionName,
		RLPTypeDecodeListFunctionType,
		NativeRLPDecodeListFunction,
	),
}

func RLPDecodeList(
	input *interpreter.ArrayValue,
	context interpreter.InvocationContext,
) interpreter.Value {
	convertedInput, err := interpreter.ByteArrayValueToByteSlice(context, input)
	if err != nil {
		panic(&RLPDecodeListError{
			Msg: err.Error(),
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
		panic(&RLPDecodeListError{
			Msg: err.Error(),
		})
	}

	if bytesRead != len(convertedInput) {
		panic(&RLPDecodeListError{
			Msg: rlpErrMsgInputContainsExtraBytes,
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

var rlpContractMethods = map[string]interpreter.FunctionValue{
	RLPTypeDecodeListFunctionName:   interpreterRLPDecodeListFunction,
	RLPTypeDecodeStringFunctionName: interpreterRLPDecodeStringFunction,
}

var RLPTypeStaticType = interpreter.ConvertSemaToStaticType(nil, RLPType)

var rlpContractValue = interpreter.NewSimpleCompositeValue(
	nil,
	RLPType.ID(),
	RLPTypeStaticType,
	nil,
	nil,
	nil,
	func(name string, context interpreter.MemberAccessibleContext) interpreter.FunctionValue {
		return rlpContractMethods[name]
	},
	nil,
	nil,
)

var RLPContract = StandardLibraryValue{
	Name:  RLPTypeName,
	Type:  RLPType,
	Value: rlpContractValue,
	Kind:  common.DeclarationKindContract,
}
