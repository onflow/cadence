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

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

const inboxStorageDomain = "inbox"
const inboxStorageSeparator = "$"
const inboxStorageRecipientTag = "recipient"
const inboxStorageValueTag = "value"

func recipientPath(name string) string {
	return name + inboxStorageSeparator + inboxStorageRecipientTag
}

func valuePath(name string) string {
	return name + inboxStorageSeparator + inboxStorageValueTag
}

// AuthAccountInbox

var authAccountInboxTypeID = sema.AuthAccountInboxType.ID()
var authAccountInboxStaticType StaticType = PrimitiveStaticTypeAuthAccountKeys

// NewAuthAccountInboxValue constructs a AuthAccount.Inbox value.
func NewAuthAccountInboxValue(
	gauge common.MemoryGauge,
	addressValue AddressValue,
) Value {

	address := addressValue.ToAddress()

	fields := map[string]Value{
		sema.AuthAccountInboxPermitField:    accountInboxPermitFunction(gauge, address),
		sema.AuthAccountInboxUnpermitField:  accountInboxUnpermitFunction(gauge, address),
		sema.AuthAccountInboxPublishField:   accountInboxPublishFunction(gauge, address, addressValue),
		sema.AuthAccountInboxUnpublishField: nil,
		sema.AuthAccountInboxClaimField:     nil,
	}

	fieldNames := []string{
		sema.AuthAccountInboxAllowlistField,
		sema.AuthAccountInboxPermitField,
		sema.AuthAccountInboxUnpermitField,
		sema.AuthAccountInboxPublishField,
		sema.AuthAccountInboxUnpublishField,
		sema.AuthAccountInboxClaimField,
	}
	computeField := func(name string, inter *Interpreter, getLocationRange func() LocationRange) Value {
		switch name {
		case sema.AuthAccountInboxAllowlistField:
			return getAccountAllowlist(inter, getLocationRange, address)
		}
		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.AuthAccountInboxStringMemoryUsage)
			addressStr := addressValue.MeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("AuthAccount.Inbox(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		authAccountInboxTypeID,
		authAccountInboxStaticType,
		fieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}

// PublicAccountInbox

var publicAccountInboxTypeID = sema.PublicAccountInboxType.ID()
var publicAccountInboxStaticType StaticType = PrimitiveStaticTypePublicAccountKeys

// NewPublicAccountInboxValue constructs a PublicAccount.Inbox value.
func NewPublicAccountInboxValue(
	gauge common.MemoryGauge,
	addressValue AddressValue,
) Value {

	address := addressValue.ToAddress()

	fields := map[string]Value{}
	fieldNames := []string{
		sema.PublicAccountInboxAllowlistField,
	}
	computeField := func(name string, inter *Interpreter, getLocationRange func() LocationRange) Value {
		switch name {
		case sema.PublicAccountInboxAllowlistField:
			return getAccountAllowlist(inter, getLocationRange, address)
		}
		return nil
	}

	var str string
	stringer := func(memoryGauge common.MemoryGauge, _ SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, common.PublicAccountInboxStringMemoryUsage)
			addressStr := addressValue.MeteredString(memoryGauge, SeenReferences{})
			str = fmt.Sprintf("PublicAccount.Inbox(%s)", addressStr)
		}
		return str
	}

	return NewSimpleCompositeValue(
		gauge,
		publicAccountInboxTypeID,
		publicAccountInboxStaticType,
		fieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}

func getAccountAllowlist(
	inter *Interpreter,
	getLocationRange func() LocationRange,
	address common.Address,
) *ArrayValue {
	allowlist := inter.ReadStored(address, inboxStorageDomain, "allowlist")

	if allowlist == nil {
		allowlist = NewArrayValue(
			inter,
			getLocationRange,
			VariableSizedStaticType{
				Type: PrimitiveStaticTypeAddress,
			},
			address)
	} else {
		allowlist = allowlist.Transfer(
			inter,
			getLocationRange,
			atree.Address(address),
			false,
			nil,
		)
	}

	allowListArray, ok := allowlist.(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return allowListArray
}

func accountInboxPermitFunction(
	gauge common.MemoryGauge,
	address common.Address,
) *HostFunctionValue {
	return NewHostFunctionValue(
		gauge,
		func(invocation Invocation) Value {
			providerValue, ok := invocation.Arguments[0].(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

			allowlist := getAccountAllowlist(inter, getLocationRange, address)

			if allowlist.Contains(inter, getLocationRange, providerValue) {
				return VoidValue{}
			}

			allowlist.Append(inter, getLocationRange, providerValue)
			inter.writeStored(address, inboxStorageDomain, "allowlist", allowlist)

			return VoidValue{}
		},
		sema.AuthAccountInboxPermitFunctionType,
	)
}

func accountInboxUnpermitFunction(
	gauge common.MemoryGauge,
	address common.Address,
) *HostFunctionValue {
	return NewHostFunctionValue(
		gauge,
		func(invocation Invocation) Value {
			providerValue, ok := invocation.Arguments[0].(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

			allowlist := getAccountAllowlist(inter, getLocationRange, address)

			index := allowlist.FirstIndex(inter, getLocationRange, providerValue)

			index.iter(func(index Value) {
				allowlist.Remove(inter, getLocationRange, index.(IntValue).ToInt())
				inter.writeStored(address, inboxStorageDomain, "allowlist", allowlist)
			})

			return VoidValue{}
		},
		sema.AuthAccountInboxUnpermitFunctionType,
	)
}

func accountInboxPublishFunction(
	gauge common.MemoryGauge,
	address common.Address,
	providerValue AddressValue,
) *HostFunctionValue {
	return NewHostFunctionValue(
		gauge,
		func(invocation Invocation) Value {
			publishedValue := invocation.Arguments[0]

			nameValue, ok := invocation.Arguments[1].(*StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			recipientValue := invocation.Arguments[2].(*AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

			recipientAllowlist := getAccountAllowlist(inter, getLocationRange, recipientValue.ToAddress())

			// if the recipient's allowlist does not contain the provider's address, return false
			if !recipientAllowlist.Contains(inter, getLocationRange, providerValue) {
				return BoolValue(false)
			}

			publishedValue = publishedValue.Transfer(
				inter,
				getLocationRange,
				atree.Address(address),
				true,
				nil,
			)

			recipient := recipientValue.Transfer(
				inter,
				getLocationRange,
				atree.Address(address),
				true,
				nil,
			)

			// we need to store both a value and an intended recipient for each name,
			// so we do two writes to represent each published value.
			inter.writeStored(address, inboxStorageDomain, valuePath(nameValue.Str), publishedValue)
			inter.writeStored(address, inboxStorageDomain, recipientPath(nameValue.Str), recipient)

			return BoolValue(true)
		},
		sema.AuthAccountTypeInboxPublishFunctionType,
	)
}
