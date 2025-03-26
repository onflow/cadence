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

// Members

func init() {
	typeName := interpreter.PrimitiveStaticTypeCapability.String()

	// Capability.borrow
	RegisterTypeBoundFunction(
		typeName,
		sema.CapabilityTypeBorrowFunctionName,
		NativeFunctionValue{
			ParameterCount: 0,
			Function: func(config *Config, typeArguments []bbq.StaticType, args ...Value) Value {
				capabilityValue := getReceiver[*interpreter.IDCapabilityValue](config, args[0])
				capabilityID := capabilityValue.ID

				if capabilityID == interpreter.InvalidCapabilityID {
					return interpreter.Nil
				}

				capabilityBorrowType := capabilityValue.BorrowType.(*interpreter.ReferenceStaticType)

				var wantedBorrowType *interpreter.ReferenceStaticType
				if len(typeArguments) > 0 {
					wantedBorrowType = typeArguments[0].(*interpreter.ReferenceStaticType)
				}

				address := capabilityValue.Address()

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
					return interpreter.Nil
				}

				return interpreter.NewSomeValueNonCopying(config.MemoryGauge, referenceValue)
			},
		})
}

func GetCheckedCapabilityControllerReference(
	context interpreter.CapConReferenceValueContext,
	capabilityAddressValue interpreter.AddressValue,
	capabilityIDValue interpreter.UInt64Value,
	wantedBorrowType *interpreter.ReferenceStaticType,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) interpreter.ReferenceValue {
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

	semaBorrowType := interpreter.MustConvertStaticToSemaType(resultBorrowType, context)
	referenceType := semaBorrowType.(*sema.ReferenceType)

	return controller.ReferenceValue(
		context,
		capabilityAddress,
		referenceType,
		EmptyLocationRange,
	)
}

func getCheckedCapabilityController(
	context interpreter.ValueStaticTypeContext,
	capabilityAddressValue interpreter.AddressValue,
	capabilityIDValue interpreter.UInt64Value,
	wantedBorrowType *interpreter.ReferenceStaticType,
	capabilityBorrowType *interpreter.ReferenceStaticType,
) (
	interpreter.CapabilityControllerValue,
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
	capabilityID := uint64(capabilityIDValue.ToInt(EmptyLocationRange))

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
) interpreter.CapabilityControllerValue {

	storageMapKey := interpreter.Uint64StorageMapKey(capabilityID)

	referenced := storageContext.ReadStored(
		address,
		common.StorageDomainCapabilityController,
		storageMapKey,
	)

	controller, ok := referenced.(interpreter.CapabilityControllerValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return controller
}

func canBorrow(
	context interpreter.ValueStaticTypeContext,
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
