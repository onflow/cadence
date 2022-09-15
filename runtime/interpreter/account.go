/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

// AuthAccount

var authAccountTypeID = sema.AuthAccountType.ID()
var authAccountStaticType StaticType = PrimitiveStaticTypeAuthAccount // unmetered
var authAccountFieldNames = []string{
	sema.AuthAccountAddressField,
	sema.AuthAccountContractsField,
	sema.AuthAccountKeysField,
}

// NewAuthAccountValue constructs an auth account value.
func NewAuthAccountValue(
	gauge common.MemoryGauge,
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func(interpreter *Interpreter) UInt64Value,
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
			gauge,
			address,
			sema.CapabilityPathType,
			sema.AuthAccountTypeGetCapabilityFunctionType,
		),
	}

	var contracts Value
	var keys Value

	computeField := func(name string, inter *Interpreter, getLocationRange func() LocationRange) Value {
		switch name {
		case sema.AuthAccountContractsField:
			if contracts == nil {
				contracts = contractsConstructor()
			}
			return contracts
		case sema.AuthAccountKeysField:
			if keys == nil {
				keys = keysConstructor()
			}
			return keys
		case sema.AuthAccountPublicPathsField:
			return inter.publicAccountPaths(address, getLocationRange)
		case sema.AuthAccountPrivatePathsField:
			return inter.privateAccountPaths(address, getLocationRange)
		case sema.AuthAccountStoragePathsField:
			return inter.storageAccountPaths(address, getLocationRange)
		case sema.AuthAccountForEachPublicField:
			return inter.newStorageIterationFunction(address, common.PathDomainPublic, sema.PublicPathType)
		case sema.AuthAccountForEachPrivateField:
			return inter.newStorageIterationFunction(address, common.PathDomainPrivate, sema.PrivatePathType)
		case sema.AuthAccountForEachStoredField:
			return inter.newStorageIterationFunction(address, common.PathDomainStorage, sema.StoragePathType)
		case sema.AuthAccountBalanceField:
			return accountBalanceGet()
		case sema.AuthAccountAvailableBalanceField:
			return accountAvailableBalanceGet()
		case sema.AuthAccountStorageUsedField:
			return storageUsedGet(inter)
		case sema.AuthAccountStorageCapacityField:
			return storageCapacityGet(inter)
		case sema.AuthAccountTypeField:
			return inter.authAccountTypeFunction(address)
		case sema.AuthAccountLoadField:
			return inter.authAccountLoadFunction(address)
		case sema.AuthAccountCopyField:
			return inter.authAccountCopyFunction(address)
		case sema.AuthAccountSaveField:
			return inter.authAccountSaveFunction(address)
		case sema.AuthAccountBorrowField:
			return inter.authAccountBorrowFunction(address)
		case sema.AuthAccountLinkField:
			return inter.authAccountLinkFunction(address)
		case sema.AuthAccountUnlinkField:
			return inter.authAccountUnlinkFunction(address)
		case sema.AuthAccountGetLinkTargetField:
			return inter.accountGetLinkTargetFunction(address)
		}

		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountValueStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("AuthAccount(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		authAccountTypeID,
		authAccountStaticType,
		authAccountFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}

// PublicAccount

var publicAccountTypeID = sema.PublicAccountType.ID()
var publicAccountStaticType StaticType = PrimitiveStaticTypePublicAccount // unmetered
var publicAccountFieldNames = []string{
	sema.PublicAccountAddressField,
	sema.PublicAccountContractsField,
	sema.PublicAccountKeysField,
}

// NewPublicAccountValue constructs a public account value.
func NewPublicAccountValue(
	gauge common.MemoryGauge,
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func(interpreter *Interpreter) UInt64Value,
	keysConstructor func() Value,
	contractsConstructor func() Value,
	inboxConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.PublicAccountAddressField: address,
		sema.PublicAccountGetCapabilityField: accountGetCapabilityFunction(
			gauge,
			address,
			sema.PublicPathType,
			sema.PublicAccountTypeGetCapabilityFunctionType,
		),
	}

	var keys Value
	var contracts Value
	var inbox Value

	computeField := func(name string, inter *Interpreter, getLocationRange func() LocationRange) Value {
		switch name {
		case sema.PublicAccountKeysField:
			if keys == nil {
				keys = keysConstructor()
			}
			return keys
		case sema.PublicAccountContractsField:
			if contracts == nil {
				contracts = contractsConstructor()
			}
			return contracts
		case sema.PublicAccountInboxField:
			if inbox == nil {
				inbox = inboxConstructor()
			}
			return inbox
		case sema.PublicAccountPathsField:
			return inter.publicAccountPaths(address, getLocationRange)
		case sema.PublicAccountForEachPublicField:
			return inter.newStorageIterationFunction(address, common.PathDomainPublic, sema.PublicPathType)
		case sema.PublicAccountBalanceField:
			return accountBalanceGet()
		case sema.PublicAccountAvailableBalanceField:
			return accountAvailableBalanceGet()
		case sema.PublicAccountStorageUsedField:
			return storageUsedGet(inter)
		case sema.PublicAccountStorageCapacityField:
			return storageCapacityGet(inter)
		case sema.PublicAccountGetTargetLinkField:
			return inter.accountGetLinkTargetFunction(address)
		}

		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.PublicAccountValueStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("PublicAccount(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		publicAccountTypeID,
		publicAccountStaticType,
		publicAccountFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}
