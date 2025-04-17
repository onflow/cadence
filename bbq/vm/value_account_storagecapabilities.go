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
	accountStorageCapabilitiesTypeName := sema.Account_StorageCapabilitiesType.QualifiedIdentifier()

	// Account.StorageCapabilities.issue
	RegisterTypeBoundFunction(
		accountStorageCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeIssueFunctionName,
			sema.Account_StorageCapabilitiesTypeIssueFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.StorageCapabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				// Get borrow type type-argument
				typeParameter := typeArguments[0]
				semaType := interpreter.MustConvertStaticToSemaType(typeParameter, context)

				return stdlib.AccountStorageCapabilitiesIssue(
					arguments,
					context,
					EmptyLocationRange,
					context.GetAccountHandler(),
					accountAddress,
					semaType,
				)
			},
		),
	)

	// Account.StorageCapabilities.issueWithType
	RegisterTypeBoundFunction(
		accountStorageCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeIssueWithTypeFunctionName,
			sema.Account_StorageCapabilitiesTypeIssueWithTypeFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.StorageCapabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				// Get path argument
				targetPathValue, ok := arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get type argument
				typeValue, ok := arguments[1].(interpreter.TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return stdlib.AccountStorageCapabilitiesIssueWithType(
					context,
					context.GetAccountHandler(),
					typeValue,
					accountAddress,
					targetPathValue,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.StorageCapabilities.getController
	RegisterTypeBoundFunction(
		accountStorageCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeGetControllerFunctionName,
			sema.Account_StorageCapabilitiesTypeGetControllerFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.StorageCapabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex]).ToAddress()

				// Get capability ID argument
				capabilityIDValue, ok := args[typeBoundFunctionArgumentOffset].(interpreter.UInt64Value)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return stdlib.AccountStorageCapabilitiesGetController(
					context,
					context.GetAccountHandler(),
					capabilityIDValue,
					accountAddress,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.StorageCapabilities.getControllers
	RegisterTypeBoundFunction(
		accountStorageCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeGetControllersFunctionName,
			sema.Account_StorageCapabilitiesTypeGetControllersFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.StorageCapabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex]).ToAddress()

				// Get path argument
				targetPathValue, ok := args[typeBoundFunctionArgumentOffset].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return stdlib.AccountStorageCapabilitiesGetControllers(
					context,
					context.GetAccountHandler(),
					targetPathValue,
					accountAddress,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.StorageCapabilities.forEachController
	RegisterTypeBoundFunction(
		accountStorageCapabilitiesTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageCapabilitiesTypeForEachControllerFunctionName,
			sema.Account_StorageCapabilitiesTypeForEachControllerFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.StorageCapabilities)
				accountAddress := getAccountTypePrivateAddressValue(args[receiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				// Get path argument
				targetPathValue, ok := arguments[0].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Get function argument
				functionValue, ok := arguments[1].(interpreter.FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return stdlib.AccountStorageCapabilitiesForeachController(
					context,
					context.GetAccountHandler(),
					functionValue,
					accountAddress,
					targetPathValue,
					EmptyLocationRange,
				)
			},
		),
	)
}
