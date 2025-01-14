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
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func(interpreter *Interpreter) UInt64Value,
) Value {

	var storageValue *SimpleCompositeValue

	fields := map[string]Value{}

	computeLazyStoredField := func(name string, inter *Interpreter) Value {
		switch name {
		case sema.Account_StorageTypeForEachPublicFunctionName:
			return inter.newStorageIterationFunction(
				storageValue,
				sema.Account_StorageTypeForEachPublicFunctionType,
				address,
				common.PathDomainPublic,
				sema.PublicPathType,
			)

		case sema.Account_StorageTypeForEachStoredFunctionName:
			return inter.newStorageIterationFunction(
				storageValue,
				sema.Account_StorageTypeForEachStoredFunctionType,
				address,
				common.PathDomainStorage,
				sema.StoragePathType,
			)

		case sema.Account_StorageTypeTypeFunctionName:
			return inter.authAccountTypeFunction(storageValue, address)

		case sema.Account_StorageTypeLoadFunctionName:
			return inter.authAccountLoadFunction(storageValue, address)

		case sema.Account_StorageTypeCopyFunctionName:
			return inter.authAccountCopyFunction(storageValue, address)

		case sema.Account_StorageTypeSaveFunctionName:
			return inter.authAccountSaveFunction(storageValue, address)

		case sema.Account_StorageTypeBorrowFunctionName:
			return inter.authAccountBorrowFunction(storageValue, address)

		case sema.Account_StorageTypeCheckFunctionName:
			return inter.authAccountCheckFunction(storageValue, address)
		}

		return nil
	}

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.Account_StorageTypePublicPathsFieldName:
			return inter.publicAccountPaths(address, locationRange)

		case sema.Account_StorageTypeStoragePathsFieldName:
			return inter.storageAccountPaths(address, locationRange)

		case sema.Account_StorageTypeUsedFieldName:
			return storageUsedGet(inter)

		case sema.Account_StorageTypeCapacityFieldName:
			return storageCapacityGet(inter)
		}

		field := computeLazyStoredField(name, inter)
		if field != nil {
			fields[name] = field
		}
		return field
	}

	var str string
	stringer := func(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(interpreter, common.AccountStorageStringMemoryUsage)
			addressStr := address.MeteredString(interpreter, seenReferences, locationRange)
			str = fmt.Sprintf("Account.Storage(%s)", addressStr)
		}
		return str
	}

	storageValue = NewSimpleCompositeValue(
		gauge,
		account_StorageTypeID,
		account_StorageStaticType,
		account_StorageFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)

	return storageValue
}
