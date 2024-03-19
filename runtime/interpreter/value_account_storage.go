/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// Account.Storage

var account_StorageTypeID = sema.Account_StorageType.ID()
var account_StorageStaticType StaticType = PrimitiveStaticTypeAccount_Storage

// NewAccountStorageValue constructs an Account.Storage value.
func NewAccountStorageValue(
	gauge common.MemoryGauge,
	address AddressValue,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func(interpreter *Interpreter) UInt64Value,
) Value {

	var forEachStoredFunction *HostFunctionValue
	var forEachPublicFunction *HostFunctionValue
	var typeFunction *HostFunctionValue
	var loadFunction *HostFunctionValue
	var copyFunction *HostFunctionValue
	var saveFunction *HostFunctionValue
	var borrowFunction *HostFunctionValue
	var checkFunction *HostFunctionValue

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.Account_StorageTypePublicPathsFieldName:
			return inter.publicAccountPaths(address, locationRange)

		case sema.Account_StorageTypeStoragePathsFieldName:
			return inter.storageAccountPaths(address, locationRange)

		case sema.Account_StorageTypeForEachPublicFunctionName:
			if forEachPublicFunction == nil {
				forEachPublicFunction = inter.newStorageIterationFunction(
					sema.Account_StorageTypeForEachPublicFunctionType,
					address,
					common.PathDomainPublic,
					sema.PublicPathType,
				)
			}
			return forEachPublicFunction

		case sema.Account_StorageTypeForEachStoredFunctionName:
			if forEachStoredFunction == nil {
				forEachStoredFunction = inter.newStorageIterationFunction(
					sema.Account_StorageTypeForEachStoredFunctionType,
					address,
					common.PathDomainStorage,
					sema.StoragePathType,
				)
			}
			return forEachStoredFunction

		case sema.Account_StorageTypeUsedFieldName:
			return storageUsedGet(inter)

		case sema.Account_StorageTypeCapacityFieldName:
			return storageCapacityGet(inter)

		case sema.Account_StorageTypeTypeFunctionName:
			if typeFunction == nil {
				typeFunction = inter.authAccountTypeFunction(address)
			}
			return typeFunction

		case sema.Account_StorageTypeLoadFunctionName:
			if loadFunction == nil {
				loadFunction = inter.authAccountLoadFunction(address)
			}
			return loadFunction

		case sema.Account_StorageTypeCopyFunctionName:
			if copyFunction == nil {
				copyFunction = inter.authAccountCopyFunction(address)
			}
			return copyFunction

		case sema.Account_StorageTypeSaveFunctionName:
			if saveFunction == nil {
				saveFunction = inter.authAccountSaveFunction(address)
			}
			return saveFunction

		case sema.Account_StorageTypeBorrowFunctionName:
			if borrowFunction == nil {
				borrowFunction = inter.authAccountBorrowFunction(address)
			}
			return borrowFunction

		case sema.Account_StorageTypeCheckFunctionName:
			if checkFunction == nil {
				checkFunction = inter.authAccountCheckFunction(address)
			}
			return checkFunction
		}

		return nil
	}

	var str string
	stringer := func(interpreter *Interpreter, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(interpreter, common.AccountStorageStringMemoryUsage)
			addressStr := address.MeteredString(interpreter, seenReferences)
			str = fmt.Sprintf("Account.Storage(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		account_StorageTypeID,
		account_StorageStaticType,
		nil,
		nil,
		computeField,
		nil,
		stringer,
	)
}
