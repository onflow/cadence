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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {

	accountStorageTypeName := sema.Account_StorageType.QualifiedIdentifier()

	// Account.Storage.save
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeSaveFunctionName,
			sema.Account_StorageTypeSaveFunctionType,
			func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageSave(
					config,
					arguments,
					address,
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.borrow
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeBorrowFunctionName,
			sema.Account_StorageTypeBorrowFunctionType,
			func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				borrowType := typeArgs[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, config)

				return interpreter.AccountStorageBorrow(
					config,
					arguments,
					semaBorrowType,
					address.ToAddress(),
					EmptyLocationRange,
				)
			},
		),
	)

	// Account.Storage.forEachPublic
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeForEachPublicFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageIterate(
					config,
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
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeForEachStoredFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageIterate(
					config,
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
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeTypeFunctionName,
			sema.Account_StorageTypeTypeFunctionType,
			func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageType(
					config,
					arguments,
					address.ToAddress(),
				)
			},
		),
	)

	// Account.Storage.load
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeLoadFunctionName,
			sema.Account_StorageTypeLoadFunctionType,
			func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAccountTypePrivateAddressValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				borrowType := typeArgs[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, config)

				return interpreter.AccountStorageRead(
					config,
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
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeCopyFunctionName,
			sema.Account_StorageTypeCopyFunctionType,
			func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAccountTypePrivateAddressValue(args[receiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				borrowType := typeArgs[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, config)

				return interpreter.AccountStorageRead(
					config,
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
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		NewBoundNativeFunctionValue(
			sema.Account_StorageTypeCheckFunctionName,
			sema.Account_StorageTypeCheckFunctionType,
			func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAccountTypePrivateAddressValue(args[receiverIndex]).ToAddress()

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				borrowType := typeArgs[0]
				semaBorrowType := interpreter.MustConvertStaticToSemaType(borrowType, config)

				return interpreter.AccountStorageCheck(
					config,
					address,
					arguments,
					semaBorrowType,
				)
			},
		),
	)
}
