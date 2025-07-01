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

	accountStorageTypeName := commons.TypeQualifier(sema.Account_StorageType)

	// Account.Storage.save
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeSaveFunctionName,
			sema.Account_StorageTypeSaveFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageSave(
					context,
					arguments,
					address,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.borrow
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeBorrowFunctionName,
			sema.Account_StorageTypeBorrowFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				borrowType := typeArguments[0]
				semaBorrowType := context.SemaTypeFromStaticType(borrowType)

				return interpreter.AccountStorageBorrow(
					context,
					arguments,
					semaBorrowType,
					address.ToAddress(),
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.forEachPublic
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeForEachPublicFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageIterate(
					context,
					arguments,
					address.ToAddress(),
					common.PathDomainPublic,
					sema.PublicPathType,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.forEachStored
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeForEachStoredFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageIterate(
					context,
					arguments,
					address.ToAddress(),
					common.PathDomainStorage,
					sema.StoragePathType,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.type
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeTypeFunctionName,
			sema.Account_StorageTypeTypeFunctionType,
			func(context *Context, _ []bbq.StaticType, args ...Value) Value {

				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageType(
					context,
					arguments,
					address.ToAddress(),
				)
			},
		),
	)

	// Account.Storage.load
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeLoadFunctionName,
			sema.Account_StorageTypeLoadFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				borrowType := typeArguments[0]
				semaBorrowType := context.SemaTypeFromStaticType(borrowType)

				return interpreter.AccountStorageRead(
					context,
					arguments,
					semaBorrowType,
					address.ToAddress(),
					true,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.copy
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeCopyFunctionName,
			sema.Account_StorageTypeCopyFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				borrowType := typeArguments[0]
				semaBorrowType := context.SemaTypeFromStaticType(borrowType)

				return interpreter.AccountStorageRead(
					context,
					arguments,
					semaBorrowType,
					address,
					false,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.check
	RegisterBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeCheckFunctionName,
			sema.Account_StorageTypeCheckFunctionType,
			func(context *Context, typeArguments []bbq.StaticType, args ...Value) Value {
				address := GetAccountTypePrivateAddressValue(args[ReceiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[TypeBoundFunctionArgumentOffset:]

				borrowType := typeArguments[0]
				semaBorrowType := context.SemaTypeFromStaticType(borrowType)

				return interpreter.AccountStorageCheck(
					context,
					address,
					arguments,
					semaBorrowType,
				)
			},
		),
	)
}
