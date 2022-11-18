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
	sema.AuthAccountInboxField,
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
	inboxConstructor func() Value,
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
	var inbox Value
	var forEachStoredFunction *HostFunctionValue
	var forEachPublicFunction *HostFunctionValue
	var forEachPrivateFunction *HostFunctionValue
	var typeFunction *HostFunctionValue
	var loadFunction *HostFunctionValue
	var copyFunction *HostFunctionValue
	var saveFunction *HostFunctionValue
	var borrowFunction *HostFunctionValue
	var linkFunction *HostFunctionValue
	var unlinkFunction *HostFunctionValue
	var getLinkTargetFunction *HostFunctionValue

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
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
		case sema.AuthAccountInboxField:
			if inbox == nil {
				inbox = inboxConstructor()
			}
			return inbox

		case sema.AuthAccountPublicPathsField:
			return inter.publicAccountPaths(address, locationRange)

		case sema.AuthAccountPrivatePathsField:
			return inter.privateAccountPaths(address, locationRange)

		case sema.AuthAccountStoragePathsField:
			return inter.storageAccountPaths(address, locationRange)

		case sema.AuthAccountForEachPublicField:
			if forEachPublicFunction == nil {
				forEachPublicFunction = inter.newStorageIterationFunction(
					address,
					common.PathDomainPublic,
					sema.PublicPathType,
				)
			}
			return forEachPublicFunction

		case sema.AuthAccountForEachPrivateField:
			if forEachPrivateFunction == nil {
				forEachPrivateFunction = inter.newStorageIterationFunction(
					address,
					common.PathDomainPrivate,
					sema.PrivatePathType,
				)
			}
			return forEachPrivateFunction

		case sema.AuthAccountForEachStoredField:
			if forEachStoredFunction == nil {
				forEachStoredFunction = inter.newStorageIterationFunction(
					address,
					common.PathDomainStorage,
					sema.StoragePathType,
				)
			}
			return forEachStoredFunction

		case sema.AuthAccountBalanceField:
			return accountBalanceGet()

		case sema.AuthAccountAvailableBalanceField:
			return accountAvailableBalanceGet()

		case sema.AuthAccountStorageUsedField:
			return storageUsedGet(inter)

		case sema.AuthAccountStorageCapacityField:
			return storageCapacityGet(inter)

		case sema.AuthAccountTypeField:
			if typeFunction == nil {
				typeFunction = inter.authAccountTypeFunction(address)
			}
			return typeFunction

		case sema.AuthAccountLoadField:
			if loadFunction == nil {
				loadFunction = inter.authAccountLoadFunction(address)
			}
			return loadFunction

		case sema.AuthAccountCopyField:
			if copyFunction == nil {
				copyFunction = inter.authAccountCopyFunction(address)
			}
			return copyFunction

		case sema.AuthAccountSaveField:
			if saveFunction == nil {
				saveFunction = inter.authAccountSaveFunction(address)
			}
			return saveFunction

		case sema.AuthAccountBorrowField:
			if borrowFunction == nil {
				borrowFunction = inter.authAccountBorrowFunction(address)
			}
			return borrowFunction

		case sema.AuthAccountLinkField:
			if linkFunction == nil {
				linkFunction = inter.authAccountLinkFunction(address)
			}
			return linkFunction

		case sema.AuthAccountUnlinkField:
			if unlinkFunction == nil {
				unlinkFunction = inter.authAccountUnlinkFunction(address)
			}
			return unlinkFunction

		case sema.AuthAccountGetLinkTargetField:
			if getLinkTargetFunction == nil {
				getLinkTargetFunction = inter.accountGetLinkTargetFunction(address)
			}
			return getLinkTargetFunction

		}

		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountValueStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
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
	var forEachPublicFunction *HostFunctionValue

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
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

		case sema.PublicAccountPathsField:
			return inter.publicAccountPaths(address, locationRange)

		case sema.PublicAccountForEachPublicField:
			if forEachPublicFunction == nil {
				forEachPublicFunction = inter.newStorageIterationFunction(
					address,
					common.PathDomainPublic,
					sema.PublicPathType,
				)
			}
			return forEachPublicFunction

		case sema.PublicAccountBalanceField:
			return accountBalanceGet()

		case sema.PublicAccountAvailableBalanceField:
			return accountAvailableBalanceGet()

		case sema.PublicAccountStorageUsedField:
			return storageUsedGet(inter)

		case sema.PublicAccountStorageCapacityField:
			return storageCapacityGet(inter)

		case sema.PublicAccountGetTargetLinkField:
			var getLinkTargetFunction *HostFunctionValue
			if getLinkTargetFunction == nil {
				getLinkTargetFunction = inter.accountGetLinkTargetFunction(address)
			}
			return getLinkTargetFunction
		}

		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.PublicAccountValueStringMemoryUsage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
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
