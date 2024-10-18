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

// NewAccountStorageValue constructs an Account.Storage value.
func NewAccountStorageValue(
	gauge common.MemoryGauge,
	address AddressValue,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func(interpreter *Interpreter) UInt64Value,
) Value {

	var str string
	stringer := func(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(interpreter, common.AccountStorageStringMemoryUsage)
			addressStr := address.MeteredString(interpreter, seenReferences, locationRange)
			str = fmt.Sprintf("Account.Storage(%s)", addressStr)
		}
		return str
	}

	storageValue := NewSimpleCompositeValue(
		gauge,
		account_StorageTypeID,
		account_StorageStaticType,
		nil,
		nil,
		nil,
		nil,
		stringer,
	)

	var forEachStoredFunction FunctionValue
	var forEachPublicFunction FunctionValue
	var typeFunction FunctionValue
	var loadFunction FunctionValue
	var copyFunction FunctionValue
	var saveFunction FunctionValue
	var borrowFunction FunctionValue
	var checkFunction FunctionValue

	storageValue.ComputeField = func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.Account_StorageTypePublicPathsFieldName:
			return inter.publicAccountPaths(address, locationRange)

		case sema.Account_StorageTypeStoragePathsFieldName:
			return inter.storageAccountPaths(address, locationRange)

		case sema.Account_StorageTypeForEachPublicFunctionName:
			if forEachPublicFunction == nil {
				forEachPublicFunction = inter.newStorageIterationFunction(
					storageValue,
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
					storageValue,
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
				typeFunction = inter.authAccountTypeFunction(storageValue, address)
			}
			return typeFunction

		case sema.Account_StorageTypeLoadFunctionName:
			if loadFunction == nil {
				loadFunction = inter.authAccountLoadFunction(storageValue, address)
			}
			return loadFunction

		case sema.Account_StorageTypeCopyFunctionName:
			if copyFunction == nil {
				copyFunction = inter.authAccountCopyFunction(storageValue, address)
			}
			return copyFunction

		case sema.Account_StorageTypeSaveFunctionName:
			if saveFunction == nil {
				saveFunction = inter.authAccountSaveFunction(storageValue, address)
			}
			return saveFunction

		case sema.Account_StorageTypeBorrowFunctionName:
			if borrowFunction == nil {
				borrowFunction = inter.authAccountBorrowFunction(storageValue, address)
			}
			return borrowFunction

		case sema.Account_StorageTypeCheckFunctionName:
			if checkFunction == nil {
				checkFunction = inter.authAccountCheckFunction(storageValue, address)
			}
			return checkFunction
		}

		return nil
	}

	return storageValue
}
