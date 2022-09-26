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
		sema.AuthAccountInboxPublishField:   accountInboxPublishFunction(gauge, address, addressValue),
		sema.AuthAccountInboxUnpublishField: accountInboxUnpublishFunction(gauge, address, addressValue),
		sema.AuthAccountInboxClaimField:     accountInboxClaimFunction(gauge, address, addressValue),
	}

	fieldNames := []string{
		sema.AuthAccountInboxPublishField,
		sema.AuthAccountInboxUnpublishField,
		sema.AuthAccountInboxClaimField,
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
		nil,
		nil,
		stringer,
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

			recipientValue := invocation.Arguments[2].(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

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

			return NewVoidValue(gauge)
		},
		sema.AuthAccountTypeInboxPublishFunctionType,
	)
}

func accountInboxUnpublishFunction(
	gauge common.MemoryGauge,
	address common.Address,
	providerValue AddressValue,
) *HostFunctionValue {
	return NewHostFunctionValue(
		gauge,
		func(invocation Invocation) Value {
			nameValue, ok := invocation.Arguments[0].(*StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

			publishedValue := inter.ReadStored(address, inboxStorageDomain, valuePath(nameValue.Str))

			if publishedValue == nil {
				return NewNilValue(gauge)
			}

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := sema.NewCapabilityType(gauge, typeParameterPair.Value)
			publishedType := publishedValue.StaticType(invocation.Interpreter)
			if !inter.IsSubTypeOfSemaType(publishedType, ty) {
				panic(ForceCastTypeMismatchError{
					ExpectedType:  ty,
					ActualType:    inter.MustConvertStaticToSemaType(publishedType),
					LocationRange: getLocationRange(),
				})
			}

			inter.writeStored(address, inboxStorageDomain, valuePath(nameValue.Str), nil)
			inter.writeStored(address, inboxStorageDomain, recipientPath(nameValue.Str), nil)

			publishedValue = publishedValue.Transfer(
				inter,
				getLocationRange,
				atree.Address(address),
				true,
				nil,
			)

			return NewSomeValueNonCopying(inter, publishedValue)
		},
		sema.AuthAccountTypeInboxPublishFunctionType,
	)
}

func accountInboxClaimFunction(
	gauge common.MemoryGauge,
	address common.Address,
	recipientValue AddressValue,
) *HostFunctionValue {
	return NewHostFunctionValue(
		gauge,
		func(invocation Invocation) Value {
			nameValue, ok := invocation.Arguments[0].(*StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			providerValue, ok := invocation.Arguments[1].(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			inter := invocation.Interpreter
			getLocationRange := invocation.GetLocationRange

			providerAddress := providerValue.ToAddress()

			publishedValue := inter.ReadStored(providerAddress, inboxStorageDomain, valuePath(nameValue.Str))

			if publishedValue == nil {
				return NewNilValue(gauge)
			}

			// compare the intended recipient with the caller
			intendedRecipient, ok := inter.ReadStored(providerAddress, inboxStorageDomain, recipientPath(nameValue.Str)).(AddressValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}
			if intendedRecipient.ToAddress() != recipientValue.ToAddress() {
				return NewNilValue(gauge)
			}

			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair == nil {
				panic(errors.NewUnreachableError())
			}

			ty := sema.NewCapabilityType(gauge, typeParameterPair.Value)
			publishedType := publishedValue.StaticType(invocation.Interpreter)
			if !inter.IsSubTypeOfSemaType(publishedType, ty) {
				panic(ForceCastTypeMismatchError{
					ExpectedType:  ty,
					ActualType:    inter.MustConvertStaticToSemaType(publishedType),
					LocationRange: getLocationRange(),
				})
			}

			inter.writeStored(providerAddress, inboxStorageDomain, valuePath(nameValue.Str), nil)
			inter.writeStored(providerAddress, inboxStorageDomain, recipientPath(nameValue.Str), nil)

			publishedValue = publishedValue.Transfer(
				inter,
				getLocationRange,
				atree.Address(address),
				true,
				nil,
			)

			return NewSomeValueNonCopying(inter, publishedValue)
		},
		sema.AuthAccountTypeInboxPublishFunctionType,
	)
}
