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

// AuthAccount.AccountCapabilities

var authAccountAccountCapabilitiesTypeID = sema.AuthAccountAccountCapabilitiesType.ID()
var authAccountAccountCapabilitiesStaticType StaticType = PrimitiveStaticTypeAuthAccountAccountCapabilities // unmetered
var authAccountAccountCapabilitiesFieldNames []string = nil

func NewAuthAccountAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getControllerFunction FunctionValue,
	getControllersFunction FunctionValue,
	forEachControllerFunction FunctionValue,
	issueFunction FunctionValue,
) Value {

	fields := map[string]Value{
		sema.AuthAccountAccountCapabilitiesTypeGetControllerFunctionName:     getControllerFunction,
		sema.AuthAccountAccountCapabilitiesTypeGetControllersFunctionName:    getControllersFunction,
		sema.AuthAccountAccountCapabilitiesTypeForEachControllerFunctionName: forEachControllerFunction,
		sema.AuthAccountAccountCapabilitiesTypeIssueFunctionName:             issueFunction,
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountAccountCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("AuthAccount.AccountCapabilities(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		authAccountAccountCapabilitiesTypeID,
		authAccountAccountCapabilitiesStaticType,
		authAccountAccountCapabilitiesFieldNames,
		fields,
		nil,
		nil,
		stringer,
	)
}
