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

// Account.Capabilities

var account_CapabilitiesTypeID = sema.AccountCapabilitiesType.ID()
var account_CapabilitiesStaticType StaticType = PrimitiveStaticTypeAccountCapabilities

func NewAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getFunction FunctionValue,
	borrowFunction FunctionValue,
	publishFunction FunctionValue,
	unpublishFunction FunctionValue,
	storageCapabilitiesConstructor func() Value,
	accountCapabilitiesConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.Account_CapabilitiesTypeGetFunctionName:       getFunction,
		sema.Account_CapabilitiesTypeBorrowFunctionName:    borrowFunction,
		sema.Account_CapabilitiesTypePublishFunctionName:   publishFunction,
		sema.Account_CapabilitiesTypeUnpublishFunctionName: unpublishFunction,
	}

	var storageCapabilities Value
	var accountCapabilities Value

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.Account_CapabilitiesTypeStorageFieldName:
			if storageCapabilities == nil {
				storageCapabilities = storageCapabilitiesConstructor()
			}
			return storageCapabilities

		case sema.Account_CapabilitiesTypeAccountFieldName:
			if accountCapabilities == nil {
				accountCapabilities = accountCapabilitiesConstructor()
			}
			return accountCapabilities
		}

		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AccountCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("Account.Capabilities(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		account_CapabilitiesTypeID,
		account_CapabilitiesStaticType,
		nil,
		fields,
		computeField,
		nil,
		stringer,
	)
}
