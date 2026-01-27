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
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {

	accountStorageTypeName := commons.TypeQualifier(sema.Account_StorageType)

	// Account.Storage.save
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeSaveFunctionName,
			sema.Account_StorageTypeSaveFunctionType,
			interpreter.NativeAccountStorageSaveFunction(nil),
		),
	)

	// Account.Storage.borrow
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeBorrowFunctionName,
			sema.Account_StorageTypeBorrowFunctionType,
			interpreter.NativeAccountStorageBorrowFunction(nil),
		),
	)

	// Account.Storage.forEachPublic
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeForEachPublicFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			interpreter.NativeAccountStorageIterateFunction(nil, common.PathDomainPublic, sema.PublicPathType),
		),
	)

	// Account.Storage.forEachStored
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeForEachStoredFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			interpreter.NativeAccountStorageIterateFunction(nil, common.PathDomainStorage, sema.StoragePathType),
		),
	)

	// Account.Storage.type
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeTypeFunctionName,
			sema.Account_StorageTypeTypeFunctionType,
			interpreter.NativeAccountStorageTypeFunction(nil),
		),
	)

	// Account.Storage.load
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeLoadFunctionName,
			sema.Account_StorageTypeLoadFunctionType,
			interpreter.NativeAccountStorageLoadFunction(nil),
		),
	)

	// Account.Storage.copy
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeCopyFunctionName,
			sema.Account_StorageTypeCopyFunctionType,
			interpreter.NativeAccountStorageCopyFunction(nil),
		),
	)

	// Account.Storage.check
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewNativeFunctionValue(
			sema.Account_StorageTypeCheckFunctionName,
			sema.Account_StorageTypeCheckFunctionType,
			interpreter.NativeAccountStorageCheckFunction(nil),
		),
	)
}
