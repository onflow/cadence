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

// Account.StorageCapabilities

var account_StorageCapabilitiesTypeID = sema.Account_StorageCapabilitiesType.ID()
var account_StorageCapabilitiesStaticType StaticType = PrimitiveStaticTypeAccountStorageCapabilities // unmetered
var account_StorageCapabilitiesFieldNames []string = nil

func NewAccountStorageCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getControllerFunction FunctionValue,
	getControllersFunction FunctionValue,
	forEachControllerFunction FunctionValue,
	issueFunction FunctionValue,
) Value {

	fields := map[string]Value{
		sema.Account_StorageCapabilitiesTypeGetControllerFunctionName:     getControllerFunction,
		sema.Account_StorageCapabilitiesTypeGetControllersFunctionName:    getControllersFunction,
		sema.Account_StorageCapabilitiesTypeForEachControllerFunctionName: forEachControllerFunction,
		sema.Account_StorageCapabilitiesTypeIssueFunctionName:             issueFunction,
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AccountStorageCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("Account.StorageCapabilities(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		account_StorageCapabilitiesTypeID,
		account_StorageCapabilitiesStaticType,
		account_StorageCapabilitiesFieldNames,
		fields,
		nil,
		nil,
		stringer,
	)
}
