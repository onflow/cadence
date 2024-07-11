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
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type CapabilityControllerValue interface {
	Value
	isCapabilityControllerValue()
	ReferenceValue(
		conf *Config,
		capabilityAddress common.Address,
		resultBorrowType *interpreter.ReferenceStaticType,
	) ReferenceValue
	ControllerCapabilityID() IntValue // TODO: UInt64Value
	CapabilityControllerBorrowType() *interpreter.ReferenceStaticType
}

// StorageCapabilityControllerValue

type StorageCapabilityControllerValue struct {
	BorrowType   *interpreter.ReferenceStaticType
	CapabilityID IntValue
	TargetPath   PathValue

	// deleted indicates if the controller got deleted. Not stored
	deleted bool

	// Injected functions.
	// Tags are not stored directly inside the controller
	// to avoid unnecessary storage reads
	// when the controller is loaded for borrowing/checking
	GetCapability func(config *Config) *CapabilityValue
	GetTag        func(config *Config) *StringValue
	SetTag        func(config *Config, tag *StringValue)
	Delete        func(config *Config, locationRange interpreter.LocationRange)
	SetTarget     func(config *Config, locationRange interpreter.LocationRange, target PathValue)
}

func NewUnmeteredStorageCapabilityControllerValue(
	borrowType *interpreter.ReferenceStaticType,
	capabilityID IntValue,
	targetPath PathValue,
) *StorageCapabilityControllerValue {
	return &StorageCapabilityControllerValue{
		BorrowType:   borrowType,
		TargetPath:   targetPath,
		CapabilityID: capabilityID,
	}
}

func NewStorageCapabilityControllerValue(
	borrowType *interpreter.ReferenceStaticType,
	capabilityID IntValue,
	targetPath PathValue,
) *StorageCapabilityControllerValue {
	return NewUnmeteredStorageCapabilityControllerValue(
		borrowType,
		capabilityID,
		targetPath,
	)
}

var _ Value = &StorageCapabilityControllerValue{}
var _ CapabilityControllerValue = &StorageCapabilityControllerValue{}
var _ MemberAccessibleValue = &StorageCapabilityControllerValue{}

func (*StorageCapabilityControllerValue) isValue() {}

func (*StorageCapabilityControllerValue) isCapabilityControllerValue() {}

func (v *StorageCapabilityControllerValue) CapabilityControllerBorrowType() *interpreter.ReferenceStaticType {
	return v.BorrowType
}

func (v *StorageCapabilityControllerValue) StaticType(_ common.MemoryGauge) StaticType {
	return interpreter.PrimitiveStaticTypeStorageCapabilityController
}

func (v *StorageCapabilityControllerValue) String() string {
	// TODO: call recursive string
	return format.StorageCapabilityController(
		v.BorrowType.String(),
		v.CapabilityID.String(),
		v.TargetPath.String(),
	)
}

func (v *StorageCapabilityControllerValue) Transfer(
	config *Config,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	//if remove {
	//	interpreter.RemoveReferencedSlab(storable)
	//}
	return v
}

func (v *StorageCapabilityControllerValue) GetMember(config *Config, name string) (result Value) {
	//defer func() {
	//	switch typedResult := result.(type) {
	//	case deletionCheckedFunctionValue:
	//		result = typedResult.FunctionValue
	//	case FunctionValue:
	//		panic(errors.NewUnexpectedError("functions need to check deletion. Use newHostFunctionValue"))
	//	}
	//}()

	// NOTE: check if controller is already deleted
	v.checkDeleted()

	switch name {
	case sema.StorageCapabilityControllerTypeTagFieldName:
		return v.GetTag(config)

	case sema.StorageCapabilityControllerTypeCapabilityIDFieldName:
		return v.CapabilityID

	case sema.StorageCapabilityControllerTypeBorrowTypeFieldName:
		panic(errors.NewUnreachableError())
		// TODO:
		//return NewTypeValue(inter, v.BorrowType)

	case sema.StorageCapabilityControllerTypeCapabilityFieldName:
		return v.GetCapability(config)

	}

	return nil
}

func init() {
	typeName := sema.StorageCapabilityControllerType.QualifiedName

	// Capability.borrow
	RegisterTypeBoundFunction(
		typeName,
		sema.StorageCapabilityControllerTypeSetTagFunctionName,
		NativeFunctionValue{
			ParameterCount: 0,
			Function: func(config *Config, typeArguments []StaticType, args ...Value) Value {
				capabilityValue := args[0].(*StorageCapabilityControllerValue)

				//stdlib.SetCapabilityControllerTag(config.interpreter())

				capabilityValue.checkDeleted()

				return Nil
			},
		})
}

func (v *StorageCapabilityControllerValue) SetMember(
	conf *Config,
	name string,
	value Value,
) {
	// NOTE: check if controller is already deleted
	v.checkDeleted()

	switch name {
	case sema.StorageCapabilityControllerTypeTagFieldName:
		stringValue, ok := value.(*StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		v.SetTag(conf, stringValue)
	}

	panic(errors.NewUnreachableError())
}

func (v *StorageCapabilityControllerValue) ControllerCapabilityID() IntValue {
	return v.CapabilityID
}

func (v *StorageCapabilityControllerValue) ReferenceValue(
	conf *Config,
	capabilityAddress common.Address,
	resultBorrowType *interpreter.ReferenceStaticType,
) ReferenceValue {
	return NewStorageReferenceValue(
		conf.Storage,
		resultBorrowType.Authorization,
		capabilityAddress,
		v.TargetPath,
		resultBorrowType.ReferencedType,
	)
}

// checkDeleted checks if the controller is deleted,
// and panics if it is.
func (v *StorageCapabilityControllerValue) checkDeleted() {
	if v.deleted {
		panic(errors.NewDefaultUserError("controller is deleted"))
	}
}
