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

package vm

import (
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {

	typeName := commons.TypeQualifier(sema.TheAddressType)

	// Methods on `Address` value.
	// Receiver is at 0-th index. Arguments starts from 1.

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.ToStringFunctionName,
			sema.ToStringFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				addressValue, arguments := SplitTypedReceiverAndArgs[interpreter.AddressValue](context, arguments) // nolint:ineffassign
				return interpreter.AddressValueToStringFunction(
					context,
					addressValue,
					EmptyLocationRange,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.AddressTypeToBytesFunctionName,
			sema.AddressTypeToBytesFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				addressValue, arguments := SplitTypedReceiverAndArgs[interpreter.AddressValue](context, arguments) // nolint:ineffassign
				address := common.Address(addressValue)
				return interpreter.ByteSliceToByteArrayValue(context, address[:])
			},
		),
	)

	// Methods on `Address` type.
	// Arguments starts from 0-th index.

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.AddressTypeFromBytesFunctionName,
			sema.AddressTypeFromBytesFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				byteArrayValue := arguments[0].(*interpreter.ArrayValue)
				return interpreter.AddressValueFromByteArray(
					context,
					byteArrayValue,
					EmptyLocationRange,
				)
			},
		),
	)

	registerBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.AddressTypeFromStringFunctionName,
			sema.AddressTypeFromStringFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				stringValue := arguments[0].(*interpreter.StringValue)
				return interpreter.AddressValueFromString(context, stringValue)
			},
		),
	)

}
