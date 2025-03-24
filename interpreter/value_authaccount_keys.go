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

// Account.Keys

var account_KeysTypeID = sema.Account_KeysType.ID()
var account_KeysStaticType StaticType = PrimitiveStaticTypeAccount_Keys
var account_KeysFieldNames []string = nil

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

	var accountKeys *SimpleCompositeValue

	fields := map[string]Value{}

	computeLazyStoredField := func(name string) Value {
		switch name {
		case sema.Account_KeysTypeAddFunctionName:
			return addFunction(accountKeys)
		case sema.Account_KeysTypeGetFunctionName:
			return getFunction(accountKeys)
		case sema.Account_KeysTypeRevokeFunctionName:
			return revokeFunction(accountKeys)
		case sema.Account_KeysTypeForEachFunctionName:
			return forEachFunction(accountKeys)
		}

		return nil
	}

	computeField := func(name string, _ MemberAccessibleContext, _ LocationRange) Value {
		switch name {
		case sema.Account_KeysTypeCountFieldName:
			return getKeysCount()
		}

		field := computeLazyStoredField(name)
		if field != nil {
			fields[name] = field
		}
		return field
	}

	var str string
	stringer := func(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(context, common.AccountKeysStringMemoryUsage)
			addressStr := address.MeteredString(context, seenReferences, locationRange)
			str = fmt.Sprintf("Account.Keys(%s)", addressStr)
		}
		return str
	}

	accountKeys = NewSimpleCompositeValue(
		gauge,
		account_KeysTypeID,
		account_KeysStaticType,
		account_KeysFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)

	return accountKeys
}

type AccountKeysCountGetter func() UInt64Value
