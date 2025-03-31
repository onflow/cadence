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
		sema.Account_StorageTypeSaveFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeSaveFunctionType.Parameters),
			Function: func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAddressMetaInfoFromValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageSave(
					config,
					arguments,
					address,
					EmptyLocationRange,
				)
			},
		},
	)

	// Account.Storage.borrow
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeBorrowFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeBorrowFunctionType.Parameters),
			Function: func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAddressMetaInfoFromValue(args[receiverIndex])

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
		},
	)

	// Account.Storage.forEachPublic
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeForEachPublicFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeForEachPublicFunctionType.Parameters),
			Function: func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAddressMetaInfoFromValue(args[receiverIndex])

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
		},
	)

	// Account.Storage.forEachStored
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeForEachPublicFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeForEachPublicFunctionType.Parameters),
			Function: func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAddressMetaInfoFromValue(args[receiverIndex])

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
		},
	)

	// Account.Storage.type
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeTypeFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeTypeFunctionType.Parameters),
			Function: func(config *Config, typeArs []bbq.StaticType, args ...Value) Value {

				address := getAddressMetaInfoFromValue(args[receiverIndex])

				// arg[0] is the receiver. Actual arguments starts from 1.
				arguments := args[typeBoundFunctionArgumentOffset:]

				return interpreter.AccountStorageType(
					config,
					arguments,
					address.ToAddress(),
				)
			},
		},
	)

	// Account.Storage.load
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeLoadFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeLoadFunctionType.Parameters),
			Function: func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAddressMetaInfoFromValue(args[receiverIndex])

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
		},
	)

	// Account.Storage.copy
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeCopyFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeCopyFunctionType.Parameters),
			Function: func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAddressMetaInfoFromValue(args[receiverIndex]).ToAddress()

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
		},
	)

	// Account.Storage.check
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeCheckFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeCheckFunctionType.Parameters),
			Function: func(config *Config, typeArgs []bbq.StaticType, args ...Value) Value {
				address := getAddressMetaInfoFromValue(args[receiverIndex]).ToAddress()

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
		},
	)
}
