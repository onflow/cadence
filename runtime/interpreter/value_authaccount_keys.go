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

// Account.Keys

var account_KeysTypeID = sema.Account_KeysType.ID()
var account_KeysStaticType StaticType = PrimitiveStaticTypeAccountKeys

// NewAccountKeysValue constructs an Account.Keys value.
func NewAccountKeysValue(
	gauge common.MemoryGauge,
	address AddressValue,
	addFunction FunctionValue,
	getFunction FunctionValue,
	revokeFunction FunctionValue,
	forEachFunction FunctionValue,
	getKeysCount AccountKeysCountGetter,
) Value {

	fields := map[string]Value{
		sema.Account_KeysTypeAddFunctionName:     addFunction,
		sema.Account_KeysTypeGetFunctionName:     getFunction,
		sema.Account_KeysTypeRevokeFunctionName:  revokeFunction,
		sema.Account_KeysTypeForEachFunctionName: forEachFunction,
	}

	computeField := func(name string, _ *Interpreter, _ LocationRange) Value {
		switch name {
		case sema.Account_KeysTypeCountFieldName:
			return getKeysCount()
		}
		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AccountKeysStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			str = fmt.Sprintf("Account.Keys(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		account_KeysTypeID,
		account_KeysStaticType,
		nil,
		fields,
		computeField,
		nil,
		stringer,
	)
}

type AccountKeysCountGetter func() UInt64Value
