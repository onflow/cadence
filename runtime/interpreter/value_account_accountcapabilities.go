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

// Account.AccountCapabilities

var Account_AccountCapabilitiesTypeID = sema.Account_AccountCapabilitiesType.ID()
var Account_AccountCapabilitiesStaticType StaticType = PrimitiveStaticTypeAccount_AccountCapabilities // unmetered
var Account_AccountCapabilitiesFieldNames []string = nil

func NewAccountAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getControllerFunction BoundFunctionGenerator,
	getControllersFunction BoundFunctionGenerator,
	forEachControllerFunction BoundFunctionGenerator,
	issueFunction BoundFunctionGenerator,
	issueWithTypeFunction BoundFunctionGenerator,
) *SimpleCompositeValue {

	var str string
	stringer := func(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(interpreter, common.AccountAccountCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(interpreter, seenReferences, locationRange)
			str = fmt.Sprintf("Account.AccountCapabilities(%s)", addressStr)
		}
		return str
	}

	accountCapabilities := NewSimpleCompositeValue(
		gauge,
		Account_AccountCapabilitiesTypeID,
		Account_AccountCapabilitiesStaticType,
		Account_AccountCapabilitiesFieldNames,
		nil,
		nil,
		nil,
		stringer,
	)

	accountCapabilities.Fields = map[string]Value{
		sema.Account_AccountCapabilitiesTypeGetControllerFunctionName:     getControllerFunction(accountCapabilities),
		sema.Account_AccountCapabilitiesTypeGetControllersFunctionName:    getControllersFunction(accountCapabilities),
		sema.Account_AccountCapabilitiesTypeForEachControllerFunctionName: forEachControllerFunction(accountCapabilities),
		sema.Account_AccountCapabilitiesTypeIssueFunctionName:             issueFunction(accountCapabilities),
		sema.Account_AccountCapabilitiesTypeIssueWithTypeFunctionName:     issueWithTypeFunction(accountCapabilities),
	}

	return accountCapabilities
}
