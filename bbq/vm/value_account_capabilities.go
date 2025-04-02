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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

func init() {
	accountCapabilitiesTypeName := sema.Account_CapabilitiesType.QualifiedIdentifier()

	// Account.Capabilities.get
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		sema.Account_CapabilitiesTypeGetFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_CapabilitiesTypeGetFunctionType.Parameters),
			Function: func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				address := getAddressMetaInfoFromValue(args[0])

				// Path argument
				path, ok := args[1].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				//pathStaticType := path.StaticType(config.MemoryGauge)
				//
				//if !IsSubType(pathStaticType, pathType) {
				//	panic(fmt.Errorf("type mismatch"))
				//}

				// NOTE: the type parameter is optional, for backwards compatibility

				var borrowType *interpreter.ReferenceStaticType
				if len(typeArguments) > 0 {
					ty := typeArguments[0]
					// we handle the nil case for this below
					borrowType, _ = ty.(*interpreter.ReferenceStaticType)
				}

				return getCapability(
					config,
					address,
					path,
					borrowType,
					false,
				)
			},
		})

	// Account.Capabilities.publish
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		sema.Account_CapabilitiesTypePublishFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_CapabilitiesTypePublishFunctionType.Parameters),
			Function: func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				accountAddress := getAddressMetaInfoFromValue(args[0])

				// Get capability argument

				var capabilityValue *interpreter.IDCapabilityValue
				switch firstValue := args[1].(type) {
				case *interpreter.IDCapabilityValue:
					capabilityValue = firstValue
				default:
					panic(errors.NewUnreachableError())
				}

				capabilityAddressValue := common.Address(capabilityValue.Address())
				if capabilityAddressValue != accountAddress {
					panic(interpreter.CapabilityAddressPublishingError{
						CapabilityAddress: interpreter.AddressValue(capabilityAddressValue),
						AccountAddress:    interpreter.AddressValue(accountAddress),
					})
				}

				// Get path argument

				path, ok := args[2].(interpreter.PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !ok || path.Domain != common.PathDomainPublic {
					panic(errors.NewUnreachableError())
				}

				domain := path.Domain.Identifier()
				storageDomain, _ := common.StorageDomainFromIdentifier(domain)

				identifier := path.Identifier

				// Prevent an overwrite

				storageMapKey := interpreter.StringStorageMapKey(identifier)
				if interpreter.StoredValueExists(
					config,
					accountAddress,
					storageDomain,
					storageMapKey,
				) {
					panic(interpreter.OverwriteError{
						Address: interpreter.AddressValue(accountAddress),
						Path:    path,
					})
				}

				capabilityValue, ok = capabilityValue.Transfer(
					config,
					EmptyLocationRange,
					atree.Address(accountAddress),
					true,
					nil,
					nil,
					true,
				).(*interpreter.IDCapabilityValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Write new value

				config.WriteStored(
					accountAddress,
					storageDomain,
					storageMapKey,
					capabilityValue,
				)

				return interpreter.Void
			},
		})
}
