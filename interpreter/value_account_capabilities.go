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

// Account.Capabilities

var account_CapabilitiesTypeID = sema.AccountCapabilitiesType.ID()
var account_CapabilitiesStaticType StaticType = PrimitiveStaticTypeAccount_Capabilities
var account_CapabilitiesFieldNames = []string{
	sema.Account_CapabilitiesTypeStorageFieldName,
	sema.Account_CapabilitiesTypeAccountFieldName,
}

func NewAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getFunction BoundFunctionGenerator,
	borrowFunction BoundFunctionGenerator,
	existsFunction BoundFunctionGenerator,
	publishFunction BoundFunctionGenerator,
	unpublishFunction BoundFunctionGenerator,
	storageCapabilitiesConstructor func() Value,
	accountCapabilitiesConstructor func() Value,
) Value {

	var capabilities *SimpleCompositeValue

	fields := map[string]Value{}

	computeLazyStoredField := func(name string) Value {
		switch name {
		case sema.Account_CapabilitiesTypeStorageFieldName:
			return storageCapabilitiesConstructor()
		case sema.Account_CapabilitiesTypeAccountFieldName:
			return accountCapabilitiesConstructor()
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

	methods := map[string]FunctionValue{}

	computeLazyStoredMethod := func(name string) FunctionValue {
		switch name {
		case sema.Account_CapabilitiesTypeGetFunctionName:
			return getFunction(capabilities)
		case sema.Account_CapabilitiesTypeBorrowFunctionName:
			return borrowFunction(capabilities)
		case sema.Account_CapabilitiesTypeExistsFunctionName:
			return existsFunction(capabilities)
		case sema.Account_CapabilitiesTypePublishFunctionName:
			return publishFunction(capabilities)
		case sema.Account_CapabilitiesTypeUnpublishFunctionName:
			return unpublishFunction(capabilities)
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
			common.UseMemory(context, common.AccountCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(context, seenReferences, locationRange)
			str = fmt.Sprintf("Account.Capabilities(%s)", addressStr)
		}
		return str
	}

	capabilities = NewSimpleCompositeValue(
		gauge,
		account_CapabilitiesTypeID,
		account_CapabilitiesStaticType,
		account_CapabilitiesFieldNames,
		fields,
		computeField,
		methodGetter,
		nil,
		stringer,
	).WithPrivateField(accountTypePrivateAddressFieldName, address)

	return capabilities
}
