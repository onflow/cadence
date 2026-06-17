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

package interpreter

import (
	"fmt"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
)

// Account.Storage

var account_StorageTypeID = sema.Account_StorageType.ID()
var account_StorageStaticType StaticType = PrimitiveStaticTypeAccount_Storage
var account_StorageFieldNames []string = nil

// NewAccountStorageValue constructs an Account.Storage value.
func NewAccountStorageValue(
	gauge common.MemoryGauge,
	address AddressValue,
	storageUsedGet func(context MemberAccessibleContext) UInt64Value,
	storageCapacityGet func(context MemberAccessibleContext) UInt64Value,
) Value {

	var storageValue *SimpleCompositeValue

	computeMethod := func(name string, context MemberAccessibleContext, accessedReference ReferenceValue) FunctionValue {
		switch name {
		case sema.Account_StorageTypeForEachPublicFunctionName:
			return newStorageIterationFunction(
				context,
				storageValue,
				accessedReference,
				sema.Account_StorageTypeForEachPublicFunctionType,
				address,
				common.PathDomainPublic,
				sema.PublicPathType,
			)

		case sema.Account_StorageTypeForEachStoredFunctionName:
			return newStorageIterationFunction(
				context,
				storageValue,
				accessedReference,
				sema.Account_StorageTypeForEachStoredFunctionType,
				address,
				common.PathDomainStorage,
				sema.StoragePathType,
			)

		case sema.Account_StorageTypeTypeFunctionName:
			return authAccountStorageTypeFunction(context, storageValue, accessedReference, address)

		case sema.Account_StorageTypeLoadFunctionName:
			return authAccountStorageLoadFunction(context, storageValue, accessedReference, address)

		case sema.Account_StorageTypeCopyFunctionName:
			return authAccountStorageCopyFunction(context, storageValue, accessedReference, address)

		case sema.Account_StorageTypeSaveFunctionName:
			return authAccountStorageSaveFunction(context, storageValue, accessedReference, address)

		case sema.Account_StorageTypeBorrowFunctionName:
			return authAccountStorageBorrowFunction(context, storageValue, accessedReference, address)

		case sema.Account_StorageTypeCheckFunctionName:
			return authAccountStorageCheckFunction(context, storageValue, accessedReference, address)
		}

		return nil
	}

	computeField := func(name string, context MemberAccessibleContext) Value {
		switch name {
		case sema.Account_StorageTypePublicPathsFieldName:
			return publicAccountPaths(context, address)

		case sema.Account_StorageTypeStoragePathsFieldName:
			return storageAccountPaths(context, address)

		case sema.Account_StorageTypeUsedFieldName:
			return storageUsedGet(context)

		case sema.Account_StorageTypeCapacityFieldName:
			return storageCapacityGet(context)
		}

		return nil
	}

	var str string
	stringer := func(context ValueStringContext, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(context, common.AccountStorageStringMemoryUsage)
			addressStr := address.MeteredString(context, seenReferences)
			str = fmt.Sprintf("Account.Storage(%s)", addressStr)
		}
		return str
	}

	storageValue = NewSimpleCompositeValue(
		gauge,
		account_StorageTypeID,
		account_StorageStaticType,
		account_StorageFieldNames,
		// No fields, only computed fields, and methods.
		nil,
		computeField,
		computeMethod,
		nil,
		stringer,
	).WithPrivateField(AccountTypePrivateAddressFieldName, address)

	return storageValue
}
