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
	"github.com/onflow/cadence/stdlib"
)

type AccountIDGenerator interface {
	// GenerateAccountID generates a new, *non-zero*, unique ID for the given account.
	GenerateAccountID(address common.Address) (uint64, error)
}

func NewAuthAccountReferenceValue(
	conf *Config,
	handler stdlib.AccountHandler,
	address common.Address,
) interpreter.Value {
	return stdlib.NewAccountReferenceValue(
		conf,
		handler,
		interpreter.AddressValue(address),
		interpreter.FullyEntitledAccountAccess,
		EmptyLocationRange,
	)
}

func NewAccountReferenceValue(
	conf *Config,
	handler stdlib.AccountHandler,
	address common.Address,
) interpreter.Value {
	return stdlib.NewAccountReferenceValue(
		conf,
		handler,
		interpreter.AddressValue(address),
		interpreter.UnauthorizedAccess,
		EmptyLocationRange,
	)
}

// members

func init() {
	// Any member methods goes here
}

func getAddressMetaInfoFromValue(value Value) common.Address {
	// TODO: How to get the address?

	//simpleCompositeValue, ok := value.(*interpreter.SimpleCompositeValue)
	//if !ok {
	//	panic(errors.NewUnreachableError())
	//}

	//addressMetaInfo := simpleCompositeValue.metadata[sema.AccountTypeAddressFieldName]
	//address, ok := addressMetaInfo.(common.Address)
	//if !ok {
	//	panic(errors.NewUnreachableError())
	//}
	//
	//return address

	return common.Address{42}
}

func getCapability(
	config *Config,
	address common.Address,
	path interpreter.PathValue,
	wantedBorrowType *interpreter.ReferenceStaticType,
	borrow bool,
) Value {
	var failValue Value
	if borrow {
		failValue = interpreter.Nil
	} else {
		failValue =
			interpreter.NewInvalidCapabilityValue(
				config.MemoryGauge,
				interpreter.AddressValue(address),
				wantedBorrowType,
			)
	}

	domain := path.Domain.StorageDomain()
	identifier := path.Identifier

	// Read stored capability, if any

	readValue := config.ReadStored(
		address,
		domain,
		interpreter.StringStorageMapKey(identifier),
	)
	if readValue == nil {
		return failValue
	}

	var readCapabilityValue *interpreter.IDCapabilityValue
	switch readValue := readValue.(type) {
	case *interpreter.IDCapabilityValue:
		readCapabilityValue = readValue
	default:
		panic(errors.NewUnreachableError())
	}

	capabilityBorrowType, ok := readCapabilityValue.BorrowType.(*interpreter.ReferenceStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	capabilityID := readCapabilityValue.ID
	capabilityAddress := readCapabilityValue.Address()

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
			resultValue = interpreter.NewCapabilityValue(
				config.MemoryGauge,
				capabilityID,
				capabilityAddress,
				resultBorrowType,
			)
		}
	}

	if resultValue == nil {
		return failValue
	}

	if borrow {
		resultValue = interpreter.NewSomeValueNonCopying(
			config.MemoryGauge,
			resultValue,
		)
	}

	return resultValue
}

func BorrowCapabilityController(
	config *Config,
	capabilityAddress interpreter.AddressValue,
	capabilityID interpreter.UInt64Value,
	wantedBorrowType *interpreter.ReferenceStaticType,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) interpreter.ReferenceValue {
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
		EmptyLocationRange,
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
	targetPathValue interpreter.PathValue,
	ty bbq.StaticType,
) interpreter.CapabilityValue {

	borrowType, ok := ty.(*interpreter.ReferenceStaticType)
	if !ok {
		// TODO: remove conversion. se static type in error
		semaType, err := interpreter.ConvertStaticToSemaType(config.Interpreter(), ty)
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

	if capabilityIDValue == interpreter.InvalidCapabilityID {
		panic(interpreter.InvalidCapabilityIDError{})
	}

	// Return controller's capability

	return interpreter.NewCapabilityValue(
		config.MemoryGauge,
		capabilityIDValue,
		interpreter.AddressValue(address),
		borrowType,
	)
}

func IssueStorageCapabilityController(
	config *Config,
	idGenerator AccountIDGenerator,
	address common.Address,
	borrowType *interpreter.ReferenceStaticType,
	targetPathValue interpreter.PathValue,
) interpreter.UInt64Value {
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

	capabilityIDValue := interpreter.NewUnmeteredUInt64Value(
		capabilityID,
	)

	controller := interpreter.NewStorageCapabilityControllerValue(
		config.MemoryGauge,
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
	capabilityIDValue interpreter.UInt64Value,
	controller interpreter.CapabilityControllerValue,
) {
	storageMapKey := interpreter.Uint64StorageMapKey(capabilityIDValue.ToInt(EmptyLocationRange))

	existed := config.WriteStored(
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
	targetPathValue interpreter.PathValue,
	capabilityIDValue interpreter.UInt64Value,
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

	readValue := accountStorage.ReadValue(config.MemoryGauge, interpreter.StringStorageMapKey(identifier))

	setKey := capabilityIDValue
	setValue := interpreter.Nil

	if readValue == nil {
		capabilityIDSet := interpreter.NewDictionaryValue(
			config,
			EmptyLocationRange,
			capabilityIDSetStaticType,
			setKey,
			setValue,
		)
		accountStorage.SetValue(config.Interpreter(), storageMapKey, capabilityIDSet)
	} else {
		capabilityIDSet := readValue.(*interpreter.DictionaryValue)
		existing := capabilityIDSet.Insert(config, EmptyLocationRange, setKey, setValue)
		if existing != interpreter.Nil {
			panic(errors.NewUnreachableError())
		}
	}
}
