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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// Account

var accountTypeID = sema.AccountType.ID()
var accountStaticType StaticType = PrimitiveStaticTypeAccount // unmetered
var accountFieldNames = []string{
	sema.AccountTypeAddressFieldName,
	sema.AccountTypeStorageFieldName,
	sema.AccountTypeContractsFieldName,
	sema.AccountTypeKeysFieldName,
	sema.AccountTypeInboxFieldName,
	sema.AccountTypeCapabilitiesFieldName,
}

// NewAccountValue constructs an account value.
func NewAccountValue(
	gauge common.MemoryGauge,
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageConstructor func() Value,
	contractsConstructor func() Value,
	keysConstructor func() Value,
	inboxConstructor func() Value,
	capabilitiesConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.AccountTypeAddressFieldName: address,
	}

	var storage Value
	var contracts Value
	var keys Value
	var inbox Value
	var capabilities Value

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.AccountTypeStorageFieldName:
			if storage == nil {
				storage = storageConstructor()
			}
			return storage

		case sema.AccountTypeContractsFieldName:
			if contracts == nil {
				contracts = contractsConstructor()
			}
			return contracts

		case sema.AccountTypeKeysFieldName:
			if keys == nil {
				keys = keysConstructor()
			}
			return keys

		case sema.AccountTypeInboxFieldName:
			if inbox == nil {
				inbox = inboxConstructor()
			}
			return inbox

		case sema.AccountTypeCapabilitiesFieldName:
			if capabilities == nil {
				capabilities = capabilitiesConstructor()
			}
			return capabilities

		case sema.AccountTypeBalanceFieldName:
			return accountBalanceGet()

		case sema.AccountTypeAvailableBalanceFieldName:
			return accountAvailableBalanceGet()
		}

		return nil
	}

	var str string
	stringer := func(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
		if str == "" {
			common.UseMemory(interpreter, common.AccountValueStringMemoryUsage)
			addressStr := address.MeteredString(interpreter, seenReferences, locationRange)
			str = fmt.Sprintf("Account(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		accountTypeID,
		accountStaticType,
		accountFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}
