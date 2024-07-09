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

package vm

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

func NewAuthAccountReferenceValue(
	address common.Address,
) *EphemeralReferenceValue {
	return newAccountReferenceValue(
		address,
		interpreter.FullyEntitledAccountAccess,
	)
}

func NewAccountReferenceValue(
	address common.Address,
) *EphemeralReferenceValue {
	return newAccountReferenceValue(
		address,
		interpreter.UnauthorizedAccess,
	)
}

func newAccountReferenceValue(
	address common.Address,
	authorization interpreter.Authorization,
) *EphemeralReferenceValue {
	return NewEphemeralReferenceValue(
		newAccountValue(address),
		authorization,
		interpreter.PrimitiveStaticTypeAccount,
	)
}

func newAccountValue(
	address common.Address,
) *SimpleCompositeValue {
	return &SimpleCompositeValue{
		typeID:     sema.AccountType.ID(),
		staticType: interpreter.PrimitiveStaticTypeAccount,
		Kind:       common.CompositeKindStructure,
		fields: map[string]Value{
			sema.AccountTypeAddressFieldName: AddressValue(address),
			// TODO: add the remaining fields
		},
	}
}

// members

func init() {
	// AuthAccount.link
	//RegisterTypeBoundFunction(typeName, sema.AccountLinkField, NativeFunctionValue{
	//	ParameterCount: len(sema.AuthAccountTypeLinkFunctionType.Parameters),
	//	Function: func(config *Config, typeArgs []StaticType, args ...Value) Value {
	//		borrowType, ok := typeArgs[0].(interpreter.ReferenceStaticType)
	//		if !ok {
	//			panic(errors.NewUnreachableError())
	//		}
	//
	//		authAccount, ok := args[0].(*SimpleCompositeValue)
	//		if !ok {
	//			panic(errors.NewUnreachableError())
	//		}
	//		address := authAccount.GetMember(config, sema.AuthAccountAddressField)
	//		addressValue, ok := address.(AddressValue)
	//		if !ok {
	//			panic(errors.NewUnreachableError())
	//		}
	//
	//		newCapabilityPath, ok := args[1].(PathValue)
	//		if !ok {
	//			panic(errors.NewUnreachableError())
	//		}
	//
	//		targetPath, ok := args[2].(PathValue)
	//		if !ok {
	//			panic(errors.NewUnreachableError())
	//		}
	//
	//		newCapabilityDomain := newCapabilityPath.Domain.Identifier()
	//		newCapabilityIdentifier := newCapabilityPath.Identifier
	//
	//		//if interpreter.storedValueExists(
	//		//	address,
	//		//	newCapabilityDomain,
	//		//	newCapabilityIdentifier,
	//		//) {
	//		//	return Nil
	//		//}
	//
	//		// Write new value
	//
	//		// Note that this will be metered twice if Atree validation is enabled.
	//		linkValue := NewLinkValue(targetPath, borrowType)
	//
	//		WriteStored(
	//			config,
	//			common.Address(addressValue),
	//			newCapabilityDomain,
	//			newCapabilityIdentifier,
	//			linkValue,
	//		)
	//
	//		return NewSomeValueNonCopying(
	//			NewCapabilityValue(
	//				addressValue,
	//				newCapabilityPath,
	//				borrowType,
	//			),
	//		)
	//	},
	//})

	accountStorageTypeName := interpreter.PrimitiveStaticTypeAccount_Storage.String()
	accountCapabilitiesTypeName := interpreter.PrimitiveStaticTypeAccount_Capabilities.String()

	// Account.Storage.save
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeSaveFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeSaveFunctionType.Parameters),
			Function: func(config *Config, typeArs []StaticType, args ...Value) Value {
				authAccount, ok := args[0].(*SimpleCompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				address := authAccount.GetMember(config, sema.AccountTypeAddressFieldName)
				addressValue, ok := address.(AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				value := args[1]

				path, ok := args[2].(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				domain := path.Domain.Identifier()
				identifier := path.Identifier

				// Prevent an overwrite

				//if interpreter.storedValueExists(
				//	address,
				//	domain,
				//	identifier,
				//) {
				//	panic("overwrite error")
				//}

				value = value.Transfer(
					config,
					atree.Address(addressValue),
					true,
					nil,
				)

				// Write new value

				WriteStored(
					config,
					common.Address(addressValue),
					domain,
					identifier,
					value,
				)

				return VoidValue{}
			},
		})

	// Account.Storage.borrow
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeBorrowFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeBorrowFunctionType.Parameters),
			Function: func(config *Config, typeArgs []StaticType, args ...Value) Value {
				authAccount, ok := args[0].(*SimpleCompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				path, ok := args[1].(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				referenceType, ok := typeArgs[0].(*interpreter.ReferenceStaticType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				address := authAccount.GetMember(config, sema.AccountTypeAddressFieldName)
				addressValue, ok := address.(AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				reference := NewStorageReferenceValue(
					config.Storage,
					referenceType.Authorization,
					common.Address(addressValue),
					path,
					referenceType,
				)

				// Attempt to dereference,
				// which reads the stored value
				// and performs a dynamic type check

				referenced, err := reference.dereference(config.MemoryGauge)
				if err != nil {
					panic(err)
				}
				if referenced == nil {
					return NilValue{}
				}

				return NewSomeValueNonCopying(reference)
			},
		})

	// Account.Capabilities.get
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		sema.Account_CapabilitiesTypeGetFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_CapabilitiesTypeGetFunctionType.Parameters),
			Function: func(config *Config, typeArguments []StaticType, args ...Value) Value {
				// Get address field from the receiver (PublicAccount)
				authAccount, ok := args[0].(*SimpleCompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				address := authAccount.GetMember(config, sema.AccountTypeAddressFieldName)
				addressValue, ok := address.(AddressValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Path argument
				path, ok := args[1].(PathValue)
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
					ty := typeArguments[1]
					// we handle the nil case for this below
					borrowType, _ = ty.(*interpreter.ReferenceStaticType)
				}

				return getCapability(
					config,
					addressValue,
					path,
					borrowType,
					true,
				)
			},
		})
}

func getCapability(
	config *Config,
	address AddressValue,
	path PathValue,
	wantedBorrowType *interpreter.ReferenceStaticType,
	borrow bool,
) Value {
	var failValue Value
	if borrow {
		failValue = Nil
	} else {
		failValue =
			NewInvalidCapabilityValue(
				config.MemoryGauge,
				address,
				wantedBorrowType,
			)
	}

	domain := path.Domain.Identifier()
	identifier := path.Identifier

	// Read stored capability, if any

	readValue := ReadStored(
		config.MemoryGauge,
		config.Storage,
		common.Address(address),
		domain,
		identifier,
	)
	if readValue == nil {
		return failValue
	}

	var readCapabilityValue *CapabilityValue

	switch readValue := readValue.(type) {
	case *CapabilityValue:
		readCapabilityValue = readValue

	default:
		panic(errors.NewUnreachableError())
	}

	capabilityBorrowType, ok := readCapabilityValue.BorrowType.(*interpreter.ReferenceStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	capabilityID := readCapabilityValue.ID
	capabilityAddress := readCapabilityValue.Address

	var resultValue Value
	if borrow {
		// When borrowing,
		// check the controller and types,
		// and return a checked reference

		resultValue = BorrowCapabilityController(
			config,
			capabilityAddress,
			capabilityID,
			wantedBorrowType,
			capabilityBorrowType,
		)
	} else {
		// When not borrowing,
		// check the controller and types,
		// and return a capability

		controller, resultBorrowType := getCheckedCapabilityController(
			config,
			capabilityAddress,
			capabilityID,
			wantedBorrowType,
			capabilityBorrowType,
		)
		if controller != nil {
			resultValue = NewCapabilityValue(
				capabilityAddress,
				capabilityID,
				resultBorrowType,
			)
		}
	}

	if resultValue == nil {
		return failValue
	}

	if borrow {
		resultValue = NewSomeValueNonCopying(
			resultValue,
		)
	}

	return resultValue
}

func BorrowCapabilityController(
	config *Config,
	capabilityAddress AddressValue,
	capabilityID IntValue,
	wantedBorrowType *interpreter.ReferenceStaticType,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) ReferenceValue {
	referenceValue := GetCheckedCapabilityControllerReference(
		config,
		capabilityAddress,
		capabilityID,
		wantedBorrowType,
		capabilityBorrowType,
	)
	if referenceValue == nil {
		return nil
	}

	// Attempt to dereference,
	// which reads the stored value
	// and performs a dynamic type check

	referencedValue := referenceValue.ReferencedValue(
		config.MemoryGauge,
		false,
	)
	if referencedValue == nil {
		return nil
	}

	return referenceValue
}
