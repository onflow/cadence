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

// AuthAccount

var authAccountTypeID = sema.AuthAccountType.ID()
var authAccountStaticType StaticType = PrimitiveStaticTypeAuthAccount // unmetered
var authAccountFieldNames = []string{
	sema.AuthAccountTypeAddressFieldName,
	sema.AuthAccountTypeContractsFieldName,
	sema.AuthAccountTypeKeysFieldName,
	sema.AuthAccountTypeInboxFieldName,
	sema.AuthAccountTypeCapabilitiesFieldName,
}

// NewAuthAccountValue constructs an auth account value.
func NewAuthAccountValue(
	gauge common.MemoryGauge,
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func(interpreter *Interpreter) UInt64Value,
	contractsConstructor func() Value,
	keysConstructor func() Value,
	inboxConstructor func() Value,
	capabilitiesConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.AuthAccountTypeAddressFieldName: address,
	}

	var contracts Value
	var keys Value
	var inbox Value
	var capabilities Value
	var forEachStoredFunction *HostFunctionValue
	var forEachPublicFunction *HostFunctionValue
	var forEachPrivateFunction *HostFunctionValue
	var typeFunction *HostFunctionValue
	var loadFunction *HostFunctionValue
	var copyFunction *HostFunctionValue
	var saveFunction *HostFunctionValue
	var borrowFunction *HostFunctionValue
	var checkFunction *HostFunctionValue

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.AuthAccountTypeContractsFieldName:
			if contracts == nil {
				contracts = contractsConstructor()
			}
			return contracts

		case sema.AuthAccountTypeKeysFieldName:
			if keys == nil {
				keys = keysConstructor()
			}
			return keys

		case sema.AuthAccountTypeInboxFieldName:
			if inbox == nil {
				inbox = inboxConstructor()
			}
			return inbox

		case sema.AuthAccountTypeCapabilitiesFieldName:
			if capabilities == nil {
				capabilities = capabilitiesConstructor()
			}
			return capabilities

		case sema.AuthAccountTypePublicPathsFieldName:
			return inter.publicAccountPaths(address, locationRange)

		case sema.AuthAccountTypePrivatePathsFieldName:
			return inter.privateAccountPaths(address, locationRange)

		case sema.AuthAccountTypeStoragePathsFieldName:
			return inter.storageAccountPaths(address, locationRange)

		case sema.AuthAccountTypeForEachPublicFunctionName:
			if forEachPublicFunction == nil {
				forEachPublicFunction = inter.newStorageIterationFunction(
					sema.AuthAccountTypeForEachPublicFunctionType,
					address,
					common.PathDomainPublic,
					sema.PublicPathType,
				)
			}
			return forEachPublicFunction

		case sema.AuthAccountTypeForEachPrivateFunctionName:
			if forEachPrivateFunction == nil {
				forEachPrivateFunction = inter.newStorageIterationFunction(
					sema.AuthAccountTypeForEachPrivateFunctionType,
					address,
					common.PathDomainPrivate,
					sema.PrivatePathType,
				)
			}
			return forEachPrivateFunction

		case sema.AuthAccountTypeForEachStoredFunctionName:
			if forEachStoredFunction == nil {
				forEachStoredFunction = inter.newStorageIterationFunction(
					sema.AuthAccountTypeForEachStoredFunctionType,
					address,
					common.PathDomainStorage,
					sema.StoragePathType,
				)
			}
			return forEachStoredFunction

		case sema.AuthAccountTypeBalanceFieldName:
			return accountBalanceGet()

		case sema.AuthAccountTypeAvailableBalanceFieldName:
			return accountAvailableBalanceGet()

		case sema.AuthAccountTypeStorageUsedFieldName:
			return storageUsedGet(inter)

		case sema.AuthAccountTypeStorageCapacityFieldName:
			return storageCapacityGet(inter)

		case sema.AuthAccountTypeTypeFunctionName:
			if typeFunction == nil {
				typeFunction = inter.authAccountTypeFunction(address)
			}
			return typeFunction

		case sema.AuthAccountTypeLoadFunctionName:
			if loadFunction == nil {
				loadFunction = inter.authAccountLoadFunction(address)
			}
			return loadFunction

		case sema.AuthAccountTypeCopyFunctionName:
			if copyFunction == nil {
				copyFunction = inter.authAccountCopyFunction(address)
			}
			return copyFunction

		case sema.AuthAccountTypeSaveFunctionName:
			if saveFunction == nil {
				saveFunction = inter.authAccountSaveFunction(address)
			}
			return saveFunction

		case sema.AuthAccountTypeBorrowFunctionName:
			if borrowFunction == nil {
				borrowFunction = inter.authAccountBorrowFunction(address)
			}
			return borrowFunction

		case sema.AuthAccountTypeCheckFunctionName:
			if checkFunction == nil {
				checkFunction = inter.authAccountCheckFunction(address)
			}
			return checkFunction
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
	sema.PublicAccountTypeAddressFieldName,
	sema.PublicAccountTypeContractsFieldName,
	sema.PublicAccountTypeKeysFieldName,
	sema.PublicAccountTypeCapabilitiesFieldName,
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
	capabilitiesConstructor func() Value,
) Value {

	fields := map[string]Value{
		sema.PublicAccountTypeAddressFieldName: address,
	}

	var keys Value
	var contracts Value
	var capabilities Value
	var forEachPublicFunction *HostFunctionValue

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.PublicAccountTypeKeysFieldName:
			if keys == nil {
				keys = keysConstructor()
			}
			return keys

		case sema.PublicAccountTypeContractsFieldName:
			if contracts == nil {
				contracts = contractsConstructor()
			}
			return contracts

		case sema.PublicAccountTypeCapabilitiesFieldName:
			if capabilities == nil {
				capabilities = capabilitiesConstructor()
			}
			return capabilities

		case sema.PublicAccountTypePublicPathsFieldName:
			return inter.publicAccountPaths(address, locationRange)

		case sema.PublicAccountTypeForEachPublicFunctionName:
			if forEachPublicFunction == nil {
				forEachPublicFunction = inter.newStorageIterationFunction(
					sema.PublicAccountTypeForEachPublicFunctionType,
					address,
					common.PathDomainPublic,
					sema.PublicPathType,
				)
			}
			return forEachPublicFunction

		case sema.PublicAccountTypeBalanceFieldName:
			return accountBalanceGet()

		case sema.PublicAccountTypeAvailableBalanceFieldName:
			return accountAvailableBalanceGet()

		case sema.PublicAccountTypeStorageUsedFieldName:
			return storageUsedGet(inter)

		case sema.PublicAccountTypeStorageCapacityFieldName:
			return storageCapacityGet(inter)
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
