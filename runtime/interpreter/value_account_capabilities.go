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

// AuthAccount.Capabilities

var authAccountCapabilitiesTypeID = sema.AuthAccountCapabilitiesType.ID()
var authAccountCapabilitiesStaticType StaticType = PrimitiveStaticTypeAuthAccountCapabilities

func NewAuthAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getFunction FunctionValue,
	borrowFunction FunctionValue,
	publishFunction FunctionValue,
	unpublishFunction FunctionValue,
	migrateLinkFunction FunctionValue,
	storageCapabilitiesConstructor func() Value,
	accountCapabilitiesConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.AuthAccountCapabilitiesTypeGetFunctionName:         getFunction,
		sema.AuthAccountCapabilitiesTypeBorrowFunctionName:      borrowFunction,
		sema.AuthAccountCapabilitiesTypePublishFunctionName:     publishFunction,
		sema.AuthAccountCapabilitiesTypeUnpublishFunctionName:   unpublishFunction,
		sema.AuthAccountCapabilitiesTypeMigrateLinkFunctionName: migrateLinkFunction,
	}

	var storageCapabilities Value
	var accountCapabilities Value

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.AuthAccountCapabilitiesTypeStorageFieldName:
			if storageCapabilities == nil {
				storageCapabilities = storageCapabilitiesConstructor()
			}
			return storageCapabilities

		case sema.AuthAccountCapabilitiesTypeAccountFieldName:
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
			common.UseMemory(memoryGauge, common.AuthAccountCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("AuthAccount.Capabilities(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		authAccountCapabilitiesTypeID,
		authAccountCapabilitiesStaticType,
		nil,
		fields,
		computeField,
		nil,
		stringer,
	)
}

// PublicAccount.Capabilities

var publicAccountCapabilitiesTypeID = sema.PublicAccountCapabilitiesType.ID()
var publicAccountCapabilitiesStaticType StaticType = PrimitiveStaticTypePublicAccountCapabilities

func NewPublicAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
	getFunction FunctionValue,
	borrowFunction FunctionValue,
) Value {

	fields := map[string]Value{
		sema.PublicAccountCapabilitiesTypeGetFunctionName:    getFunction,
		sema.PublicAccountCapabilitiesTypeBorrowFunctionName: borrowFunction,
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.PublicAccountCapabilitiesStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("PublicAccount.Capabilities(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		publicAccountCapabilitiesTypeID,
		publicAccountCapabilitiesStaticType,
		nil,
		fields,
		nil,
		nil,
		stringer,
	)
}
