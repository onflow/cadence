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

// Account.StorageCapabilities

var account_StorageCapabilitiesTypeID = sema.Account_StorageCapabilitiesType.ID()
var account_StorageCapabilitiesStaticType StaticType = PrimitiveStaticTypeAccount_StorageCapabilities // unmetered
var account_StorageCapabilitiesFieldNames []string = nil

func NewAccountStorageCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getControllerFunction BoundFunctionGenerator,
	getControllersFunction BoundFunctionGenerator,
	forEachControllerFunction BoundFunctionGenerator,
	issueFunction BoundFunctionGenerator,
	issueWithTypeFunction BoundFunctionGenerator,
) Value {

	var storageCapabilities *SimpleCompositeValue

	fields := map[string]Value{}

	computeLazyStoredField := func(name string) Value {
		switch name {
		case sema.Account_StorageCapabilitiesTypeGetControllerFunctionName:
			return getControllerFunction(storageCapabilities)
		case sema.Account_StorageCapabilitiesTypeGetControllersFunctionName:
			return getControllersFunction(storageCapabilities)
		case sema.Account_StorageCapabilitiesTypeForEachControllerFunctionName:
			return forEachControllerFunction(storageCapabilities)
		case sema.Account_StorageCapabilitiesTypeIssueFunctionName:
			return issueFunction(storageCapabilities)
		case sema.Account_StorageCapabilitiesTypeIssueWithTypeFunctionName:
			return issueWithTypeFunction(storageCapabilities)
		}

		return nil
	}

	computeField := func(name string, _ *Interpreter, _ LocationRange) Value {
		field := computeLazyStoredField(name)
		if field != nil {
			fields[name] = field
		}
		return field
	}

	var str string
	stringer := func(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(context, common.AccountStorageCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(context, seenReferences, locationRange)
			str = fmt.Sprintf("Account.StorageCapabilities(%s)", addressStr)
		}
		return str
	}

	storageCapabilities = NewSimpleCompositeValue(
		gauge,
		account_StorageCapabilitiesTypeID,
		account_StorageCapabilitiesStaticType,
		account_StorageCapabilitiesFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)

	return storageCapabilities
}
