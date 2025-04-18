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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

// members

func init() {
	accountCapabilitiesTypeName := sema.Account_CapabilitiesType.QualifiedIdentifier()

	// Account.Capabilities.get
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_CapabilitiesTypeGetFunctionName,
			sema.Account_CapabilitiesTypeGetFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				pathValue, ok := args[typeBoundFunctionArgumentOffset].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				borrowType := typeArguments[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, context)

				return stdlib.AccountCapabilitiesGet(
					context,
					context.GetAccountHandler(),
					pathValue,
					semaBorrowType,
					false,
					address,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Capabilities.borrow
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_CapabilitiesTypeBorrowFunctionName,
			sema.Account_CapabilitiesTypeBorrowFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				// Get path argument
				pathValue, ok := args[typeBoundFunctionArgumentOffset].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				borrowType := typeArguments[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, context)

				return stdlib.AccountCapabilitiesGet(
					context,
					context.GetAccountHandler(),
					pathValue,
					semaBorrowType,
					true,
					address,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Capabilities.publish
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_CapabilitiesTypePublishFunctionName,
			sema.Account_CapabilitiesTypePublishFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				// Get capability argument
				capabilityValue, ok := arguments[0].(interpreter.CapabilityValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get path argument
				pathValue, ok := arguments[1].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return stdlib.AccountCapabilitiesPublish(
					context,
					context,
					capabilityValue,
					pathValue,
					accountAddress,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Capabilities.unpublish
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_CapabilitiesTypeUnpublishFunctionName,
			sema.Account_CapabilitiesTypeUnpublishFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex])

				// Get path argument
				pathValue, ok := args[typeBoundFunctionArgumentOffset].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return stdlib.AccountCapabilitiesUnpublish(
					context,
					context,
					pathValue,
					accountAddress,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Capabilities.exist
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_CapabilitiesTypeExistsFunctionName,
			sema.Account_CapabilitiesTypeExistsFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				pathValue, ok := args[typeBoundFunctionArgumentOffset].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return stdlib.AccountCapabilitiesExists(
					context,
					pathValue,
					accountAddress.ToAddress(),
				)
			},
		),
	)
}
