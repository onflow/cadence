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
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeSaveFunctionName,
			sema.Account_StorageTypeSaveFunctionType,
			interpreter.UnifiedAccountStorageSaveFunction(nil),
		),
	)

	// Account.Storage.borrow
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeBorrowFunctionName,
			sema.Account_StorageTypeBorrowFunctionType,
			interpreter.UnifiedAccountStorageBorrowFunction(nil),
		),
	)

	// Account.Storage.forEachPublic
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeForEachPublicFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			interpreter.UnifiedAccountStorageIterateFunction(nil, common.PathDomainPublic, sema.PublicPathType),
		),
	)

	// Account.Storage.forEachStored
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeForEachStoredFunctionName,
			sema.Account_StorageTypeForEachPublicFunctionType,
			interpreter.UnifiedAccountStorageIterateFunction(nil, common.PathDomainStorage, sema.StoragePathType),
		),
	)

	// Account.Storage.type
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeTypeFunctionName,
			sema.Account_StorageTypeTypeFunctionType,
			interpreter.UnifiedAccountStorageTypeFunction(nil),
		),
	)

	// Account.Storage.load
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeLoadFunctionName,
			sema.Account_StorageTypeLoadFunctionType,
			interpreter.UnifiedAccountStorageReadFunction(nil, true),
		),
	)

	// Account.Storage.copy
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeCopyFunctionName,
			sema.Account_StorageTypeCopyFunctionType,
			interpreter.UnifiedAccountStorageReadFunction(nil, false),
		),
	)

	// Account.Storage.check
	registerBuiltinTypeBoundFunction(
		accountStorageTypeName,
		NewUnifiedNativeFunctionValue(
			sema.Account_StorageTypeCheckFunctionName,
			sema.Account_StorageTypeCheckFunctionType,
			interpreter.UnifiedAccountStorageCheckFunction(nil),
		),
	)
}
