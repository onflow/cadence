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
var account_KeysStaticType StaticType = PrimitiveStaticTypeAccount_Keys

// NewAccountKeysValue constructs an Account.Keys value.
func NewAccountKeysValue(
	gauge common.MemoryGauge,
	address AddressValue,
	addFunction BoundFunctionGenerator,
	getFunction BoundFunctionGenerator,
	revokeFunction BoundFunctionGenerator,
	forEachFunction BoundFunctionGenerator,
	getKeysCount AccountKeysCountGetter,
) Value {

	computeField := func(name string, _ *Interpreter, _ LocationRange) Value {
		switch name {
		case sema.Account_KeysTypeCountFieldName:
			return getKeysCount()
		}
		return nil
	}

	var str string
	stringer := func(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(interpreter, common.AccountKeysStringMemoryUsage)
			addressStr := address.MeteredString(interpreter, seenReferences, locationRange)
			str = fmt.Sprintf("Account.Keys(%s)", addressStr)
		}
		return str
	}

	accountKeys := NewSimpleCompositeValue(
		gauge,
		account_KeysTypeID,
		account_KeysStaticType,
		nil,
		nil,
		computeField,
		nil,
		stringer,
	)

	accountKeys.Fields = map[string]Value{
		sema.Account_KeysTypeAddFunctionName:     addFunction(accountKeys),
		sema.Account_KeysTypeGetFunctionName:     getFunction(accountKeys),
		sema.Account_KeysTypeRevokeFunctionName:  revokeFunction(accountKeys),
		sema.Account_KeysTypeForEachFunctionName: forEachFunction(accountKeys),
	}

	return accountKeys
}

type AccountKeysCountGetter func() UInt64Value
