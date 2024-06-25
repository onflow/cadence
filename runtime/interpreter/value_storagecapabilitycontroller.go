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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

type CapabilityControllerValue interface {
	Value
	isCapabilityControllerValue()
	CapabilityControllerBorrowType() *ReferenceStaticType
	ReferenceValue(
		interpreter *Interpreter,
		capabilityAddress common.Address,
		resultBorrowType *sema.ReferenceType,
		locationRange LocationRange,
	) ReferenceValue
	ControllerCapabilityID() UInt64Value
}

// StorageCapabilityControllerValue

type StorageCapabilityControllerValue struct {
	BorrowType   *ReferenceStaticType
	CapabilityID UInt64Value
	TargetPath   PathValue

	// deleted indicates if the controller got deleted. Not stored
	deleted bool

	// Lazily initialized function values.
	// Host functions based on injected functions (see below).
	deleteFunction   FunctionValue
	targetFunction   FunctionValue
	retargetFunction FunctionValue
	setTagFunction   FunctionValue

	// Injected functions.
	// Tags are not stored directly inside the controller
	// to avoid unnecessary storage reads
	// when the controller is loaded for borrowing/checking
	GetCapability func(inter *Interpreter) *IDCapabilityValue
	GetTag        func(inter *Interpreter) *StringValue
	SetTag        func(inter *Interpreter, tag *StringValue)
	Delete        func(inter *Interpreter, locationRange LocationRange)
	SetTarget     func(inter *Interpreter, locationRange LocationRange, target PathValue)
}

func NewUnmeteredStorageCapabilityControllerValue(
	borrowType *ReferenceStaticType,
	capabilityID UInt64Value,
	targetPath PathValue,
) *StorageCapabilityControllerValue {
	return &StorageCapabilityControllerValue{
		BorrowType:   borrowType,
		TargetPath:   targetPath,
		CapabilityID: capabilityID,
	}
}

func NewStorageCapabilityControllerValue(
	memoryGauge common.MemoryGauge,
	borrowType *ReferenceStaticType,
	capabilityID UInt64Value,
	targetPath PathValue,
) *StorageCapabilityControllerValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.StorageCapabilityControllerValueMemoryUsage)
	return NewUnmeteredStorageCapabilityControllerValue(
		borrowType,
		capabilityID,
		targetPath,
	)
}

var _ Value = &StorageCapabilityControllerValue{}
var _ atree.Value = &StorageCapabilityControllerValue{}
var _ EquatableValue = &StorageCapabilityControllerValue{}
var _ CapabilityControllerValue = &StorageCapabilityControllerValue{}
var _ MemberAccessibleValue = &StorageCapabilityControllerValue{}

func (*StorageCapabilityControllerValue) isValue() {}

func (*StorageCapabilityControllerValue) isCapabilityControllerValue() {}

func (v *StorageCapabilityControllerValue) CapabilityControllerBorrowType() *ReferenceStaticType {
	return v.BorrowType
}

func (v *StorageCapabilityControllerValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitStorageCapabilityControllerValue(interpreter, v)
}

func (v *StorageCapabilityControllerValue) Walk(_ *Interpreter, walkChild func(Value), _ LocationRange) {
	walkChild(v.TargetPath)
	walkChild(v.CapabilityID)
}

func (v *StorageCapabilityControllerValue) StaticType(_ *Interpreter) StaticType {
	return PrimitiveStaticTypeStorageCapabilityController
}

func (*StorageCapabilityControllerValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (v *StorageCapabilityControllerValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *StorageCapabilityControllerValue) RecursiveString(seenReferences SeenReferences) string {
	return format.StorageCapabilityController(
		v.BorrowType.String(),
		v.CapabilityID.RecursiveString(seenReferences),
		v.TargetPath.RecursiveString(seenReferences),
	)
}

func (v *StorageCapabilityControllerValue) MeteredString(
	interpreter *Interpreter,
	seenReferences SeenReferences,
	locationRange LocationRange,
) string {
	common.UseMemory(interpreter, common.StorageCapabilityControllerValueStringMemoryUsage)

	return format.StorageCapabilityController(
		v.BorrowType.MeteredString(interpreter),
		v.CapabilityID.MeteredString(interpreter, seenReferences, locationRange),
		v.TargetPath.MeteredString(interpreter, seenReferences, locationRange),
	)
}

func (v *StorageCapabilityControllerValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *StorageCapabilityControllerValue) Equal(
	interpreter *Interpreter,
	locationRange LocationRange,
	other Value,
) bool {
	otherController, ok := other.(*StorageCapabilityControllerValue)
	if !ok {
		return false
	}

	return otherController.TargetPath.Equal(interpreter, locationRange, v.TargetPath) &&
		otherController.BorrowType.Equal(v.BorrowType) &&
		otherController.CapabilityID.Equal(interpreter, locationRange, v.CapabilityID)
}

func (*StorageCapabilityControllerValue) IsStorable() bool {
	return true
}

func (v *StorageCapabilityControllerValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (
	atree.Storable,
	error,
) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (*StorageCapabilityControllerValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*StorageCapabilityControllerValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *StorageCapabilityControllerValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *StorageCapabilityControllerValue) Clone(interpreter *Interpreter) Value {
	return &StorageCapabilityControllerValue{
		TargetPath:   v.TargetPath.Clone(interpreter).(PathValue),
		BorrowType:   v.BorrowType,
		CapabilityID: v.CapabilityID,
	}
}

func (v *StorageCapabilityControllerValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v *StorageCapabilityControllerValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *StorageCapabilityControllerValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *StorageCapabilityControllerValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.TargetPath,
		v.CapabilityID,
	}
}

func (v *StorageCapabilityControllerValue) GetMember(inter *Interpreter, _ LocationRange, name string) (result Value) {
	defer func() {
		switch typedResult := result.(type) {
		case deletionCheckedFunctionValue:
			result = typedResult.FunctionValue
		case FunctionValue:
			panic(errors.NewUnexpectedError("functions need to check deletion. Use newHostFunctionValue"))
		}
	}()

	// NOTE: check if controller is already deleted
	v.checkDeleted()

	switch name {
	case sema.StorageCapabilityControllerTypeTagFieldName:
		return v.GetTag(inter)

	case sema.StorageCapabilityControllerTypeSetTagFunctionName:
		if v.setTagFunction == nil {
			v.setTagFunction = v.newSetTagFunction(inter)
		}
		return v.setTagFunction

	case sema.StorageCapabilityControllerTypeCapabilityIDFieldName:
		return v.CapabilityID

	case sema.StorageCapabilityControllerTypeBorrowTypeFieldName:
		return NewTypeValue(inter, v.BorrowType)

	case sema.StorageCapabilityControllerTypeCapabilityFieldName:
		return v.GetCapability(inter)

	case sema.StorageCapabilityControllerTypeDeleteFunctionName:
		if v.deleteFunction == nil {
			v.deleteFunction = v.newDeleteFunction(inter)
		}
		return v.deleteFunction

	case sema.StorageCapabilityControllerTypeTargetFunctionName:
		if v.targetFunction == nil {
			v.targetFunction = v.newTargetFunction(inter)
		}
		return v.targetFunction

	case sema.StorageCapabilityControllerTypeRetargetFunctionName:
		if v.retargetFunction == nil {
			v.retargetFunction = v.newRetargetFunction(inter)
		}
		return v.retargetFunction

		// NOTE: when adding new functions, ensure checkDeleted is called,
		// by e.g. using StorageCapabilityControllerValue.newHostFunction
	}

	return nil
}

func (*StorageCapabilityControllerValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Storage capability controllers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *StorageCapabilityControllerValue) SetMember(
	inter *Interpreter,
	_ LocationRange,
	identifier string,
	value Value,
) bool {
	// NOTE: check if controller is already deleted
	v.checkDeleted()

	switch identifier {
	case sema.StorageCapabilityControllerTypeTagFieldName:
		stringValue, ok := value.(*StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		v.SetTag(inter, stringValue)
		return true
	}

	panic(errors.NewUnreachableError())
}

func (v *StorageCapabilityControllerValue) ControllerCapabilityID() UInt64Value {
	return v.CapabilityID
}

func (v *StorageCapabilityControllerValue) ReferenceValue(
	interpreter *Interpreter,
	capabilityAddress common.Address,
	resultBorrowType *sema.ReferenceType,
	_ LocationRange,
) ReferenceValue {
	authorization := ConvertSemaAccessToStaticAuthorization(
		interpreter,
		resultBorrowType.Authorization,
	)
	return NewStorageReferenceValue(
		interpreter,
		authorization,
		capabilityAddress,
		v.TargetPath,
		resultBorrowType.Type,
	)
}

// checkDeleted checks if the controller is deleted,
// and panics if it is.
func (v *StorageCapabilityControllerValue) checkDeleted() {
	if v.deleted {
		panic(errors.NewDefaultUserError("controller is deleted"))
	}
}

func (v *StorageCapabilityControllerValue) newHostFunctionValue(
	inter *Interpreter,
	funcType *sema.FunctionType,
	f func(invocation Invocation) Value,
) FunctionValue {
	return deletionCheckedFunctionValue{
		FunctionValue: NewBoundHostFunctionValue(
			inter,
			v,
			funcType,
			func(invocation Invocation) Value {
				// NOTE: check if controller is already deleted
				v.checkDeleted()

				return f(invocation)
			},
		),
	}
}

func (v *StorageCapabilityControllerValue) newDeleteFunction(
	inter *Interpreter,
) FunctionValue {
	return v.newHostFunctionValue(
		inter,
		sema.StorageCapabilityControllerTypeDeleteFunctionType,
		func(invocation Invocation) Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			v.Delete(inter, locationRange)

			v.deleted = true

			return Void
		},
	)
}

func (v *StorageCapabilityControllerValue) newTargetFunction(
	inter *Interpreter,
) FunctionValue {
	return v.newHostFunctionValue(
		inter,
		sema.StorageCapabilityControllerTypeTargetFunctionType,
		func(invocation Invocation) Value {
			return v.TargetPath
		},
	)
}

func (v *StorageCapabilityControllerValue) newRetargetFunction(
	inter *Interpreter,
) FunctionValue {
	return v.newHostFunctionValue(
		inter,
		sema.StorageCapabilityControllerTypeRetargetFunctionType,
		func(invocation Invocation) Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			// Get path argument

			newTargetPathValue, ok := invocation.Arguments[0].(PathValue)
			if !ok || newTargetPathValue.Domain != common.PathDomainStorage {
				panic(errors.NewUnreachableError())
			}

			v.SetTarget(inter, locationRange, newTargetPathValue)
			v.TargetPath = newTargetPathValue

			return Void
		},
	)
}

func (v *StorageCapabilityControllerValue) newSetTagFunction(
	inter *Interpreter,
) FunctionValue {
	return v.newHostFunctionValue(
		inter,
		sema.StorageCapabilityControllerTypeSetTagFunctionType,
		func(invocation Invocation) Value {
			inter := invocation.Interpreter

			newTagValue, ok := invocation.Arguments[0].(*StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			v.SetTag(inter, newTagValue)

			return Void
		},
	)
}
