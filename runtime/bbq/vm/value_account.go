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
	"github.com/onflow/cadence/runtime/stdlib"
)

type AccountIDGenerator interface {
	// GenerateAccountID generates a new, *non-zero*, unique ID for the given account.
	GenerateAccountID(address common.Address) (uint64, error)
}

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
			sema.AccountTypeAddressFieldName:      AddressValue(address),
			sema.AccountTypeStorageFieldName:      NewAccountStorageValue(address),
			sema.AccountTypeCapabilitiesFieldName: NewAccountCapabilitiesValue(address),
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

	accountStorageTypeName := sema.Account_StorageType.QualifiedIdentifier()
	accountCapabilitiesTypeName := sema.Account_CapabilitiesType.QualifiedIdentifier()
	accountStorageCapabilitiesTypeName := sema.Account_StorageCapabilitiesType.QualifiedIdentifier()

	// Account.Storage.save
	RegisterTypeBoundFunction(
		accountStorageTypeName,
		sema.Account_StorageTypeSaveFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_StorageTypeSaveFunctionType.Parameters),
			Function: func(config *Config, typeArs []StaticType, args ...Value) Value {
				address := getAddressMetaInfoFromValue(args[0])

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
					atree.Address(address),
					true,
					nil,
				)

				// Write new value

				WriteStored(
					config,
					address,
					domain,
					interpreter.StringStorageMapKey(identifier),
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
				address := getAddressMetaInfoFromValue(args[0])

				path, ok := args[1].(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				referenceType, ok := typeArgs[0].(*interpreter.ReferenceStaticType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				reference := NewStorageReferenceValue(
					config.Storage,
					referenceType.Authorization,
					address,
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
				// Get address field from the receiver (Account.Capabilities)
				address := getAddressMetaInfoFromValue(args[0])

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
					ty := typeArguments[0]
					// we handle the nil case for this below
					borrowType, _ = ty.(*interpreter.ReferenceStaticType)
				}

				return getCapability(
					config,
					address,
					path,
					borrowType,
					true,
				)
			},
		})

	// Account.Capabilities.publish
	RegisterTypeBoundFunction(
		accountCapabilitiesTypeName,
		sema.Account_CapabilitiesTypePublishFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_CapabilitiesTypeGetFunctionType.Parameters),
			Function: func(config *Config, typeArguments []StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.Capabilities)
				accountAddress := getAddressMetaInfoFromValue(args[0])

				// Get capability argument

				var capabilityValue *CapabilityValue
				switch firstValue := args[1].(type) {
				case *CapabilityValue:
					capabilityValue = firstValue
				default:
					panic(errors.NewUnreachableError())
				}

				capabilityAddressValue := common.Address(capabilityValue.Address)
				if capabilityAddressValue != accountAddress {
					panic(interpreter.CapabilityAddressPublishingError{
						CapabilityAddress: interpreter.AddressValue(capabilityAddressValue),
						AccountAddress:    interpreter.AddressValue(accountAddress),
					})
				}

				// Get path argument

				path, ok := args[2].(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !ok || path.Domain != common.PathDomainPublic {
					panic(errors.NewUnreachableError())
				}

				domain := path.Domain.Identifier()
				identifier := path.Identifier

				// Prevent an overwrite

				storageMapKey := interpreter.StringStorageMapKey(identifier)
				if StoredValueExists(
					config.Storage,
					accountAddress,
					domain,
					storageMapKey,
				) {
					panic(interpreter.OverwriteError{
						Address: interpreter.AddressValue(accountAddress),
						Path:    VMValueToInterpreterValue(config.Storage, path).(interpreter.PathValue),
					})
				}

				capabilityValue, ok = capabilityValue.Transfer(
					config,
					atree.Address(accountAddress),
					true,
					nil,
				).(*CapabilityValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// Write new value

				WriteStored(
					config,
					accountAddress,
					domain,
					storageMapKey,
					capabilityValue,
				)

				return Void
			},
		})

	// Account.StorageCapabilities.issue
	RegisterTypeBoundFunction(
		accountStorageCapabilitiesTypeName,
		sema.Account_StorageCapabilitiesTypeIssueFunctionName,
		NativeFunctionValue{
			ParameterCount: len(sema.Account_CapabilitiesTypeGetFunctionType.Parameters),
			Function: func(config *Config, typeArguments []StaticType, args ...Value) Value {
				// Get address field from the receiver (Account.StorageCapabilities)
				accountAddress := getAddressMetaInfoFromValue(args[0])

				// Path argument
				targetPathValue, ok := args[1].(PathValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				if !ok || targetPathValue.Domain != common.PathDomainStorage {
					panic(errors.NewUnreachableError())
				}

				// Get borrow type type-argument
				ty := typeArguments[0]

				// Issue capability controller and return capability

				return checkAndIssueStorageCapabilityControllerWithType(
					config,
					config.AccountHandler,
					accountAddress,
					targetPathValue,
					ty,
				)
			},
		})
}

func getAddressMetaInfoFromValue(value Value) common.Address {
	simpleCompositeValue, ok := value.(*SimpleCompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	addressMetaInfo := simpleCompositeValue.metaInfo[sema.AccountTypeAddressFieldName]
	address, ok := addressMetaInfo.(common.Address)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return address
}

func getCapability(
	config *Config,
	address common.Address,
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
		address,
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

func checkAndIssueStorageCapabilityControllerWithType(
	config *Config,
	idGenerator AccountIDGenerator,
	address common.Address,
	targetPathValue PathValue,
	ty StaticType,
) CapabilityValue {

	borrowType, ok := ty.(*interpreter.ReferenceStaticType)
	if !ok {
		// TODO: remove conversion. se static type in error
		semaType, err := config.interpreter().ConvertStaticToSemaType(ty)
		if err != nil {
			panic(err)
		}
		panic(interpreter.InvalidCapabilityIssueTypeError{
			ExpectedTypeDescription: "reference type",
			ActualType:              semaType,
		})
	}

	// Issue capability controller

	capabilityIDValue := IssueStorageCapabilityController(
		config,
		idGenerator,
		address,
		borrowType,
		targetPathValue,
	)

	if capabilityIDValue == InvalidCapabilityID {
		panic(interpreter.InvalidCapabilityIDError{})
	}

	// Return controller's capability

	return NewCapabilityValue(
		AddressValue(address),
		capabilityIDValue,
		borrowType,
	)
}

func IssueStorageCapabilityController(
	config *Config,
	idGenerator AccountIDGenerator,
	address common.Address,
	borrowType *interpreter.ReferenceStaticType,
	targetPathValue PathValue,
) IntValue {
	// Create and write StorageCapabilityController

	var capabilityID uint64
	var err error
	errors.WrapPanic(func() {
		capabilityID, err = idGenerator.GenerateAccountID(address)
	})
	if err != nil {
		panic(interpreter.WrappedExternalError(err))
	}
	if capabilityID == 0 {
		panic(errors.NewUnexpectedError("invalid zero account ID"))
	}

	capabilityIDValue := IntValue{
		SmallInt: int64(capabilityID),
	}

	controller := NewStorageCapabilityControllerValue(
		borrowType,
		capabilityIDValue,
		targetPathValue,
	)

	storeCapabilityController(config, address, capabilityIDValue, controller)
	recordStorageCapabilityController(config, address, targetPathValue, capabilityIDValue)

	return capabilityIDValue
}

func storeCapabilityController(
	config *Config,
	address common.Address,
	capabilityIDValue IntValue,
	controller CapabilityControllerValue,
) {
	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue.SmallInt)

	existed := WriteStored(
		config,
		address,
		stdlib.CapabilityControllerStorageDomain,
		storageMapKey,
		controller,
	)

	if existed {
		panic(errors.NewUnreachableError())
	}
}

var capabilityIDSetStaticType = &interpreter.DictionaryStaticType{
	KeyType:   interpreter.PrimitiveStaticTypeUInt64,
	ValueType: interpreter.NilStaticType,
}

func recordStorageCapabilityController(
	config *Config,
	address common.Address,
	targetPathValue PathValue,
	capabilityIDValue IntValue,
) {
	if targetPathValue.Domain != common.PathDomainStorage {
		panic(errors.NewUnreachableError())
	}

	addressPath := AddressPath{
		Address: address,
		Path:    targetPathValue,
	}
	if config.CapabilityControllerIterations[addressPath] > 0 {
		config.MutationDuringCapabilityControllerIteration = true
	}

	storage := config.Storage
	identifier := targetPathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	accountStorage := storage.GetStorageMap(address, stdlib.PathCapabilityStorageDomain, false)

	referenced := accountStorage.ReadValue(config.MemoryGauge, interpreter.StringStorageMapKey(identifier))
	readValue := InterpreterValueToVMValue(referenced)

	setKey := capabilityIDValue
	setValue := Nil

	if readValue == nil {
		capabilityIDSet := NewDictionaryValue(
			config,
			capabilityIDSetStaticType,
			setKey,
			setValue,
		)
		capabilityIDSetInterValue := VMValueToInterpreterValue(storage, capabilityIDSet)
		accountStorage.SetValue(config.interpreter(), storageMapKey, capabilityIDSetInterValue)
	} else {
		capabilityIDSet := readValue.(*DictionaryValue)
		existing := capabilityIDSet.Insert(config, setKey, setValue)
		if existing != Nil {
			panic(errors.NewUnreachableError())
		}
	}
}
