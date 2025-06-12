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

	RegisterBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.ToStringFunctionName,
			sema.ToStringFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				address := arguments[receiverIndex].(interpreter.AddressValue)
				return interpreter.AddressValueToStringFunction(
					context,
					address,
					EmptyLocationRange,
				)
			},
		),
	)

	RegisterBuiltinTypeBoundFunction(
		typeName,
		NewNativeFunctionValue(
			sema.AddressTypeToBytesFunctionName,
			sema.AddressTypeToBytesFunctionType,
			func(context *Context, _ []bbq.StaticType, arguments ...Value) Value {
				addressValue := arguments[receiverIndex].(interpreter.AddressValue)
				address := common.Address(addressValue)
				return interpreter.ByteSliceToByteArrayValue(context, address[:])
			},
		),
	)

	RegisterBuiltinTypeBoundFunction(
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

	RegisterBuiltinTypeBoundFunction(
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
