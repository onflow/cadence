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
	"github.com/onflow/cadence/bbq"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type AccountIDGenerator interface {
	// GenerateAccountID generates a new, *non-zero*, unique ID for the given account.
	GenerateAccountID(address common.Address) (uint64, error)
}

func NewAuthAccountReferenceValue(
	conf *Config,
	address common.Address,
) *EphemeralReferenceValue {
	return newAccountReferenceValue(
		conf,
		address,
		interpreter.FullyEntitledAccountAccess,
	)
}

func NewAccountReferenceValue(
	conf *Config,
	address common.Address,
) *EphemeralReferenceValue {
	return newAccountReferenceValue(
		conf,
		address,
		interpreter.UnauthorizedAccess,
	)
}

func newAccountReferenceValue(
	conf *Config,
	address common.Address,
	authorization interpreter.Authorization,
) *EphemeralReferenceValue {
	return NewEphemeralReferenceValue(
		conf,
		newAccountValue(address),
		authorization,
		interpreter.PrimitiveStaticTypeAccount,
	)
}

func newAccountValue(
	address common.Address,
) *SimpleCompositeValue {
	value := &SimpleCompositeValue{
		typeID:     sema.AccountType.ID(),
		staticType: interpreter.PrimitiveStaticTypeAccount,
		Kind:       common.CompositeKindStructure,
		fields: map[string]Value{
			sema.AccountTypeAddressFieldName: AddressValue(address),
		},
	}

	value.computeField = func(name string) Value {
		var field Value
		switch name {
		case sema.AccountTypeStorageFieldName:
			field = NewAccountStorageValue(address)
		case sema.AccountTypeCapabilitiesFieldName:
			field = NewAccountCapabilitiesValue(address)
		default:
			return nil
		}

		value.fields[name] = field
		return field
	}

	// TODO: add the remaining fields

	return value
}

// members

func init() {
	// Any member methods goes here
}

func getAddressMetaInfoFromValue(value Value) common.Address {
	simpleCompositeValue, ok := value.(*SimpleCompositeValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	addressMetaInfo := simpleCompositeValue.metadata[sema.AccountTypeAddressFieldName]
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
		config,
		address,
		domain,
		identifier,
	)
	if readValue == nil {
		return failValue
	}

	var readCapabilityValue CapabilityValue
	switch readValue := readValue.(type) {
	case CapabilityValue:
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
		config,
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
	ty bbq.StaticType,
) CapabilityValue {

	borrowType, ok := ty.(*interpreter.ReferenceStaticType)
	if !ok {
		// TODO: remove conversion. se static type in error
		semaType, err := config.Interpreter().ConvertStaticToSemaType(ty)
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

	capabilityIDValue := NewIntValue(int64(capabilityID))

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
		common.StorageDomainCapabilityController,
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

	identifier := targetPathValue.Identifier

	storageMapKey := interpreter.StringStorageMapKey(identifier)

	accountStorage := config.storage.GetDomainStorageMap(
		config.Interpreter(),
		address,
		common.StorageDomainPathCapability,
		true,
	)

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
		capabilityIDSetInterValue := VMValueToInterpreterValue(config, capabilityIDSet)
		accountStorage.SetValue(config.Interpreter(), storageMapKey, capabilityIDSetInterValue)
	} else {
		capabilityIDSet := readValue.(*DictionaryValue)
		existing := capabilityIDSet.Insert(config, setKey, setValue)
		if existing != Nil {
			panic(errors.NewUnreachableError())
		}
	}
}
