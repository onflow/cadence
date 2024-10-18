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

// Account.AccountCapabilities

var account_AccountCapabilitiesTypeID = sema.Account_AccountCapabilitiesType.ID()
var account_AccountCapabilitiesStaticType StaticType = PrimitiveStaticTypeAccount_AccountCapabilities // unmetered
var account_AccountCapabilitiesFieldNames []string = nil

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
		account_AccountCapabilitiesTypeID,
		account_AccountCapabilitiesStaticType,
		account_AccountCapabilitiesFieldNames,
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
