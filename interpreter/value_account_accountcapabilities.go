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

	var accountCapabilities *SimpleCompositeValue

	fields := map[string]Value{}

	computeLazyStoredField := func(name string) Value {
		switch name {
		case sema.Account_AccountCapabilitiesTypeGetControllerFunctionName:
			return getControllerFunction(accountCapabilities)
		case sema.Account_AccountCapabilitiesTypeGetControllersFunctionName:
			return getControllersFunction(accountCapabilities)
		case sema.Account_AccountCapabilitiesTypeForEachControllerFunctionName:
			return forEachControllerFunction(accountCapabilities)
		case sema.Account_AccountCapabilitiesTypeIssueFunctionName:
			return issueFunction(accountCapabilities)
		case sema.Account_AccountCapabilitiesTypeIssueWithTypeFunctionName:
			return issueWithTypeFunction(accountCapabilities)
		}

		return nil
	}

	computeField := func(name string, _ MemberAccessibleContext, _ LocationRange) Value {
		field := computeLazyStoredField(name)
		if field != nil {
			fields[name] = field
		}
		return field
	}

	var str string
	stringer := func(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(context, common.AccountAccountCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(context, seenReferences, locationRange)
			str = fmt.Sprintf("Account.AccountCapabilities(%s)", addressStr)
		}
		return str
	}

	accountCapabilities = NewSimpleCompositeValue(
		gauge,
		account_AccountCapabilitiesTypeID,
		account_AccountCapabilitiesStaticType,
		account_AccountCapabilitiesFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	).WithPrivateField(AccountTypePrivateAddressFieldName, address)

	return accountCapabilities
}
