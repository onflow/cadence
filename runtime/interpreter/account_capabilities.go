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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/sema"
)

var authAccountCapabilitiesFieldNames []string = nil
var authAccountCapabilitiesStaticType StaticType = PrimitiveStaticTypeAuthAccountCapabilities
var authAccountCapabilitiesTypeID = sema.AuthAccountCapabilitiesType.ID()

func NewAuthAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
) Value {
	getCapabilityFn := wrapGetCapabilityAsOptional(gauge, address)

	fields := map[string]Value{
		sema.AccountCapabilitiesTypeGetFunctionName:    getCapabilityFn,
		sema.AccountCapabilitiesTypeBorrowFunctionName: newAccountCapabilitiesBorrowFunction(gauge, getCapabilityFn),
	}

	var forEachFunction *HostFunctionValue           // forEach
	var getOneControllerFunction *HostFunctionValue  // getController
	var getAllControllersFunction *HostFunctionValue // getControllers
	var forEachControllerFunction *HostFunctionValue
	var issueFunction *HostFunctionValue

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.AccountCapabilitiesTypeForEachFunctionName:
			if forEachFunction == nil {
				forEachFunction = inter.newStorageIterationFunction(
					address,
					common.PathDomainPublic,
					sema.PublicPathType,
				)
			}
			return forEachFunction

		case sema.AuthAccountCapabilitiesTypeGetControllerFunctionName:
			_ = getOneControllerFunction // TODO
		case sema.AuthAccountCapabilitiesTypeGetControllersFunctionName:
			_ = getAllControllersFunction // TODO
		case sema.AuthAccountCapabilitiesTypeForEachControllerFunctionName:
			_ = forEachControllerFunction // TODO
		case sema.AuthAccountCapabilitiesTypeIssueFunctionName:
			_ = issueFunction // TODO
		}

		return nil
	}

	stringer := capabilitiesStringer(address, common.AuthAccountCapabilitiesStringMemoryUsage, "AuthAccount")

	return NewSimpleCompositeValue(
		gauge,
		authAccountCapabilitiesTypeID,
		authAccountCapabilitiesStaticType,
		authAccountCapabilitiesFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}

var publicAccountCapabilitiesFieldNames []string = nil
var publicAccountCapabilitiesStaticType StaticType = PrimitiveStaticTypeAuthAccountCapabilities
var publicAccountCapabilitiesTypeID = sema.PublicAccountCapabilitiesType.ID()

func NewPublicAccountCapabilitiesValue(
	gauge common.MemoryGauge,
	address AddressValue,
) Value {
	getCapabilityFn := wrapGetCapabilityAsOptional(gauge, address)

	fields := map[string]Value{
		sema.AccountCapabilitiesTypeGetFunctionName:    getCapabilityFn,
		sema.AccountCapabilitiesTypeBorrowFunctionName: newAccountCapabilitiesBorrowFunction(gauge, getCapabilityFn),
	}

	var forEachFunction *HostFunctionValue // forEach

	computeField := func(name string, inter *Interpreter, locationRange LocationRange) Value {
		switch name {
		case sema.AccountCapabilitiesTypeForEachFunctionName:
			if forEachFunction == nil {
				forEachFunction = inter.newStorageIterationFunction(
					address,
					common.PathDomainPublic,
					sema.PublicPathType,
				)
			}

			return forEachFunction
		}

		return nil
	}

	stringer := capabilitiesStringer(address, common.PublicAccountCapabilitiesStringMemoryUsage, "PublicAccount")

	return NewSimpleCompositeValue(
		gauge,
		publicAccountCapabilitiesTypeID,
		publicAccountCapabilitiesStaticType,
		publicAccountCapabilitiesFieldNames,
		fields,
		computeField,
		nil,
		stringer,
	)
}

// lift the nullable result of invoking accountGetCapabilityFunction into an Optional
func wrapGetCapabilityAsOptional(
	gauge common.MemoryGauge,
	address AddressValue,
) *HostFunctionValue {
	getCapabilityFnValue := accountGetCapabilityFunction(
		gauge,
		address,
		sema.PublicPathType, // there's some discussion in FLIP #798 about changing this to a CapabilityPath
		sema.AccountCapabilitiesTypeGetFunctionType,
		true, // check if the capability exists
	)

	innerHostFn := getCapabilityFnValue.Function

	getCapabilityFnValue.Function = func(invocation Invocation) Value {
		maybeStorageCapabilityValue := innerHostFn(invocation)
		if maybeStorageCapabilityValue == nil {
			return Nil
		}

		return NewSomeValueNonCopying(
			invocation.Interpreter,
			maybeStorageCapabilityValue,
		)
	}

	return getCapabilityFnValue
}

func newAccountCapabilitiesBorrowFunction(gauge common.MemoryGauge, getCapabilityFn *HostFunctionValue) *HostFunctionValue {
	return NewHostFunctionValue(gauge, func(invocation Invocation) Value {
		optionalCapabilityValue, ok := getCapabilityFn.invoke(invocation).(OptionalValue)
		assertUnreachable(ok)

		return optionalCapabilityValue.andThen(func(capability Value) OptionalValue {
			storageCapability, ok := capability.(*StorageCapabilityValue)
			assertUnreachable(ok)

			borrowFnValue := invocation.Interpreter.getMember(storageCapability, invocation.LocationRange, sema.CapabilityTypeBorrowFunctionName)
			if borrowFnValue == nil {
				panic(errors.NewUnexpectedError("Could not resolve StorageCapabilityValue.borrow"))
			}

			borrowHostFn, ok := borrowFnValue.(*HostFunctionValue)
			assertUnreachable(ok)

			result, ok := borrowHostFn.invoke(invocation).(OptionalValue)
			assertUnreachable(ok)

			return result
		})
	}, sema.AccountCapabilitiesTypeBorrowFunctionType)
}

func capabilitiesStringer(address AddressValue, usage common.MemoryUsage, prefix string) func(common.MemoryGauge, SeenReferences) string {
	var str string
	return func(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
		if str == "" {
			common.UseMemory(memoryGauge, usage)
			addressStr := address.MeteredString(memoryGauge, seenReferences)
			return fmt.Sprintf("%s.Capabilities(%s)", prefix, addressStr)
		}

		return str
	}
}

func assertUnreachable(condition bool) {
	if !condition {
		panic(errors.NewUnreachableError())
	}
}
