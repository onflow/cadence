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

// Account.Contracts

var account_ContractsTypeID = sema.Account_ContractsType.ID()
var account_ContractsStaticType StaticType = PrimitiveStaticTypeAccount_Contracts // unmetered
var account_ContractsFieldNames []string = nil

type ContractNamesGetter func(context MemberAccessibleContext, locationRange LocationRange) *ArrayValue

func NewAccountContractsValue(
	gauge common.MemoryGauge,
	address AddressValue,
	addFunction BoundFunctionGenerator,
	updateFunction BoundFunctionGenerator,
	tryUpdateFunction BoundFunctionGenerator,
	getFunction BoundFunctionGenerator,
	borrowFunction BoundFunctionGenerator,
	removeFunction BoundFunctionGenerator,
	namesGetter ContractNamesGetter,
) Value {

	var accountContracts *SimpleCompositeValue

	methods := map[string]FunctionValue{}

	computeLazyStoredMethod := func(name string) FunctionValue {
		switch name {
		case sema.Account_ContractsTypeAddFunctionName:
			return addFunction(accountContracts)
		case sema.Account_ContractsTypeGetFunctionName:
			return getFunction(accountContracts)
		case sema.Account_ContractsTypeBorrowFunctionName:
			return borrowFunction(accountContracts)
		case sema.Account_ContractsTypeRemoveFunctionName:
			return removeFunction(accountContracts)
		case sema.Account_ContractsTypeUpdateFunctionName:
			return updateFunction(accountContracts)
		case sema.Account_ContractsTypeTryUpdateFunctionName:
			return tryUpdateFunction(accountContracts)
		}

		return nil
	}

	computeField := func(name string, context MemberAccessibleContext, locationRange LocationRange) Value {
		switch name {
		case sema.Account_ContractsTypeNamesFieldName:
			return namesGetter(context, locationRange)
		}

		return nil
	}

	methodGetter := func(name string, _ MemberAccessibleContext) FunctionValue {
		method, ok := methods[name]
		if !ok {
			method = computeLazyStoredMethod(name)
			if method != nil {
				methods[name] = method
			}
		}

		return method
	}

	var str string
	stringer := func(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(context, common.AccountContractsStringMemoryUsage)
			addressStr := address.MeteredString(context, seenReferences, locationRange)
			str = fmt.Sprintf("Account.Contracts(%s)", addressStr)
		}
		return str
	}

	accountContracts = NewSimpleCompositeValue(
		gauge,
		account_ContractsTypeID,
		account_ContractsStaticType,
		account_ContractsFieldNames,
		// No fields, only computed fields, and methods.
		nil,
		computeField,
		methodGetter,
		nil,
		stringer,
	).WithPrivateField(AccountTypePrivateAddressFieldName, address)

	return accountContracts
}
