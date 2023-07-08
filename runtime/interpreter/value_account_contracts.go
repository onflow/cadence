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

// Account.Contracts

var account_ContractsTypeID = sema.Account_ContractsType.ID()
var account_ContractsStaticType StaticType = PrimitiveStaticTypeAccountContracts // unmetered
var account_ContractsFieldNames []string = nil

type ContractNamesGetter func(interpreter *Interpreter, locationRange LocationRange) *ArrayValue

func NewAccountContractsValue(
	gauge common.MemoryGauge,
	address AddressValue,
	addFunction FunctionValue,
	updateFunction FunctionValue,
	getFunction FunctionValue,
	borrowFunction FunctionValue,
	removeFunction FunctionValue,
	namesGetter ContractNamesGetter,
) Value {

	fields := map[string]Value{
		sema.Account_ContractsTypeAddFunctionName:    addFunction,
		sema.Account_ContractsTypeGetFunctionName:    getFunction,
		sema.Account_ContractsTypeBorrowFunctionName: borrowFunction,
		sema.Account_ContractsTypeRemoveFunctionName: removeFunction,
		sema.Account_ContractsTypeUpdateFunctionName: updateFunction,
	}

	computeField := func(
		name string,
		interpreter *Interpreter,
		locationRange LocationRange,
	) Value {
		switch name {
		case sema.Account_ContractsTypeNamesFieldName:
			return namesGetter(interpreter, locationRange)
		}
		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AccountContractsStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("Account.Contracts(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		account_ContractsTypeID,
		account_ContractsStaticType,
		account_ContractsFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}
