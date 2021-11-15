/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2021 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/sema"
)

// AuthAccount

var authAccountTypeID = sema.AuthAccountType.ID()
var authAccountStaticType StaticType = PrimitiveStaticTypeAuthAccount
var authAccountDynamicType DynamicType = CompositeDynamicType{
	StaticType: sema.AuthAccountType,
}
var authAccountFieldNames = []string{
	sema.AuthAccountAddressField,
	sema.AuthAccountContractsField,
	sema.AuthAccountKeysField,
}

// NewAuthAccountValue constructs an auth account value.
func NewAuthAccountValue(
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func() UInt64Value,
	addPublicKeyFunction FunctionValue,
	removePublicKeyFunction FunctionValue,
	contractsConstructor func() Value,
	keysConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.AuthAccountAddressField:         address,
		sema.AuthAccountAddPublicKeyField:    addPublicKeyFunction,
		sema.AuthAccountRemovePublicKeyField: removePublicKeyFunction,
		sema.AuthAccountGetCapabilityField: accountGetCapabilityFunction(
			address,
			sema.CapabilityPathType,
			sema.AuthAccountTypeGetCapabilityFunctionType,
		),
	}

	var contracts Value
	var keys Value

	computedFields := map[string]ComputedField{
		sema.AuthAccountContractsField: func(_ *Interpreter, _ func() LocationRange) Value {
			if contracts == nil {
				contracts = contractsConstructor()
			}
			return contracts
		},
		sema.AuthAccountKeysField: func(_ *Interpreter, _ func() LocationRange) Value {
			if keys == nil {
				keys = keysConstructor()
			}
			return keys
		},
		sema.AuthAccountBalanceField: func(_ *Interpreter, _ func() LocationRange) Value {
			return accountBalanceGet()
		},
		sema.AuthAccountAvailableBalanceField: func(_ *Interpreter, _ func() LocationRange) Value {
			return accountAvailableBalanceGet()
		},
		sema.AuthAccountStorageUsedField: func(inter *Interpreter, _ func() LocationRange) Value {
			return storageUsedGet(inter)
		},
		sema.AuthAccountStorageCapacityField: func(_ *Interpreter, _ func() LocationRange) Value {
			return storageCapacityGet()
		},
		sema.AuthAccountLoadField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.authAccountLoadFunction(address)
		},
		sema.AuthAccountCopyField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.authAccountCopyFunction(address)
		},
		sema.AuthAccountSaveField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.authAccountSaveFunction(address)
		},
		sema.AuthAccountBorrowField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.authAccountBorrowFunction(address)
		},
		sema.AuthAccountLinkField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.authAccountLinkFunction(address)
		},
		sema.AuthAccountUnlinkField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.authAccountUnlinkFunction(address)
		},
		sema.AuthAccountGetLinkTargetField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.accountGetLinkTargetFunction(address)
		},
	}

	var str string
	stringer := func(_ SeenReferences) string {
		if str == "" {
			str = fmt.Sprintf("AuthAccount(%s)", address)
		}
		return str
	}

	return NewSimpleCompositeValue(
		authAccountTypeID,
		authAccountStaticType,
		authAccountDynamicType,
		authAccountFieldNames,
		fields,
		computedFields,
		nil,
		stringer,
	)
}

// PublicAccount

var publicAccountTypeID = sema.PublicAccountType.ID()
var publicAccountStaticType StaticType = PrimitiveStaticTypePublicAccount
var publicAccountDynamicType DynamicType = CompositeDynamicType{
	StaticType: sema.PublicAccountType,
}
var publicAccountFieldNames = []string{
	sema.PublicAccountAddressField,
	sema.PublicAccountContractsField,
	sema.PublicAccountKeysField,
}

// NewPublicAccountValue constructs a public account value.
func NewPublicAccountValue(
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func() UInt64Value,
	keysConstructor func() Value,
	contractsConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.PublicAccountAddressField: address,
		sema.PublicAccountGetCapabilityField: accountGetCapabilityFunction(
			address,
			sema.PublicPathType,
			sema.PublicAccountTypeGetCapabilityFunctionType,
		),
	}

	var keys Value
	var contracts Value

	computedFields := map[string]ComputedField{
		sema.PublicAccountKeysField: func(_ *Interpreter, _ func() LocationRange) Value {
			if keys == nil {
				keys = keysConstructor()
			}
			return keys
		},
		sema.PublicAccountContractsField: func(_ *Interpreter, _ func() LocationRange) Value {
			if contracts == nil {
				contracts = contractsConstructor()
			}
			return contracts
		},
		sema.PublicAccountBalanceField: func(_ *Interpreter, _ func() LocationRange) Value {
			return accountBalanceGet()
		},
		sema.PublicAccountAvailableBalanceField: func(_ *Interpreter, _ func() LocationRange) Value {
			return accountAvailableBalanceGet()
		},
		sema.PublicAccountStorageUsedField: func(inter *Interpreter, _ func() LocationRange) Value {
			return storageUsedGet(inter)
		},
		sema.PublicAccountStorageCapacityField: func(_ *Interpreter, _ func() LocationRange) Value {
			return storageCapacityGet()
		},
		sema.PublicAccountGetTargetLinkField: func(inter *Interpreter, _ func() LocationRange) Value {
			return inter.accountGetLinkTargetFunction(address)
		},
	}

	var str string
	stringer := func(_ SeenReferences) string {
		if str == "" {
			str = fmt.Sprintf("PublicAccount(%s)", address)
		}
		return str
	}

	return NewSimpleCompositeValue(
		publicAccountTypeID,
		publicAccountStaticType,
		publicAccountDynamicType,
		publicAccountFieldNames,
		fields,
		computedFields,
		nil,
		stringer,
	)
}
