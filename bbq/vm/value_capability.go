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
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

// members

type CapabilityValue struct {
	Address    AddressValue
	BorrowType bbq.StaticType
	ID         IntValue // TODO: UInt64Value
}

var _ Value = CapabilityValue{}

func NewCapabilityValue(address AddressValue, id IntValue, borrowType bbq.StaticType) CapabilityValue {
	return CapabilityValue{
		Address:    address,
		BorrowType: borrowType,
		ID:         id,
	}
}

func NewInvalidCapabilityValue(
	address common.Address,
	borrowType bbq.StaticType,
) CapabilityValue {
	return CapabilityValue{
		ID:         InvalidCapabilityID,
		Address:    AddressValue(address),
		BorrowType: borrowType,
	}
}

func (CapabilityValue) isValue() {}

func (v CapabilityValue) StaticType(context StaticTypeContext) bbq.StaticType {
	return interpreter.NewCapabilityStaticType(context, v.BorrowType)
}

func (v CapabilityValue) Transfer(TransferContext, atree.Address, bool, atree.Storable) Value {
	return v
}

func (v CapabilityValue) String() string {
	var borrowType string
	if v.BorrowType != nil {
		borrowType = v.BorrowType.String()
	}
	return format.Capability(
		borrowType,
		v.Address.String(),
		v.ID.String(),
	)
}

var InvalidCapabilityID = IntValue{}

func init() {
	typeName := interpreter.PrimitiveStaticTypeCapability.String()

	// Capability.borrow
	RegisterTypeBoundFunction(
		typeName,
		sema.CapabilityTypeBorrowFunctionName,
		NativeFunctionValue{
			ParameterCount: 0,
			Function: func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				capabilityValue := getReceiver[CapabilityValue](config, args[0])
				capabilityID := capabilityValue.ID

				if capabilityID == InvalidCapabilityID {
					return Nil
				}

				capabilityBorrowType := capabilityValue.BorrowType.(*interpreter.ReferenceStaticType)

				var wantedBorrowType *interpreter.ReferenceStaticType
				if len(typeArguments) > 0 {
					wantedBorrowType = typeArguments[0].(*interpreter.ReferenceStaticType)
				}

				address := capabilityValue.Address

				referenceValue := GetCheckedCapabilityControllerReference(
					config,
					address,
					capabilityID,
					wantedBorrowType,
					capabilityBorrowType,
				)
				if referenceValue == nil {
					return nil
				}

				// TODO: Is this needed?
				// Attempt to dereference,
				// which reads the stored value
				// and performs a dynamic type check

				//value, err := referenceValue.dereference(config.MemoryGauge)
				//if err != nil {
				//	panic(err)
				//}

				if referenceValue == nil {
					return Nil
				}

				return NewSomeValueNonCopying(referenceValue)
			},
		})
}

func GetCheckedCapabilityControllerReference(
	context StaticTypeContext,
	capabilityAddressValue AddressValue,
	capabilityIDValue IntValue,
	wantedBorrowType *interpreter.ReferenceStaticType,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) ReferenceValue {
	controller, resultBorrowType := getCheckedCapabilityController(
		context,
		capabilityAddressValue,
		capabilityIDValue,
		wantedBorrowType,
		capabilityBorrowType,
	)
	if controller == nil {
		return nil
	}

	capabilityAddress := common.Address(capabilityAddressValue)

	return controller.ReferenceValue(
		capabilityAddress,
		resultBorrowType,
	)
}

func getCheckedCapabilityController(
	context StaticTypeContext,
	capabilityAddressValue AddressValue,
	capabilityIDValue IntValue,
	wantedBorrowType *interpreter.ReferenceStaticType,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) (
	CapabilityControllerValue,
	*interpreter.ReferenceStaticType,
) {
	if wantedBorrowType == nil {
		wantedBorrowType = capabilityBorrowType
	} else { //nolint:gocritic
		// TODO:
		//   wantedBorrowType = inter.SubstituteMappedEntitlements(wantedBorrowType).(*sema.ReferenceType)

		if !canBorrow(context, wantedBorrowType, capabilityBorrowType) {
			return nil, nil
		}
	}

	capabilityAddress := common.Address(capabilityAddressValue)
	capabilityID := uint64(capabilityIDValue.SmallInt)

	controller := getCapabilityController(context, capabilityAddress, capabilityID)
	if controller == nil {
		return nil, nil
	}

	controllerBorrowType := controller.CapabilityControllerBorrowType()
	if !canBorrow(context, wantedBorrowType, controllerBorrowType) {
		return nil, nil
	}

	return controller, wantedBorrowType
}

// getCapabilityController gets the capability controller for the given capability ID
func getCapabilityController(
	storageContext interpreter.StorageReader,
	address common.Address,
	capabilityID uint64,
) CapabilityControllerValue {

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityID)

	referenced := storageContext.ReadStored(
		address,
		common.StorageDomainCapabilityController,
		storageMapKey,
	)
	vmReferencedValue := InterpreterValueToVMValue(referenced)

	controller, ok := vmReferencedValue.(CapabilityControllerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return controller
}

func canBorrow(
	context StaticTypeContext,
	wantedBorrowType *interpreter.ReferenceStaticType,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) bool {

	// Ensure the wanted borrow type is not more permissive than the capability borrow type
	// TODO:
	//if !wantedBorrowType.Authorization.
	//	PermitsAccess(capabilityBorrowType.Authorization) {
	//
	//	return false
	//}

	// Ensure the wanted borrow type is a subtype or supertype of the capability borrow type

	return interpreter.IsSubType(
		context,
		wantedBorrowType.ReferencedType,
		capabilityBorrowType.ReferencedType,
	) ||
		interpreter.IsSubType(
			context,
			capabilityBorrowType.ReferencedType,
			wantedBorrowType.ReferencedType,
		)
}
