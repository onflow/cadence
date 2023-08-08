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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

type CapabilityControllerValue interface {
	Value
	isCapabilityControllerValue()
	CapabilityControllerBorrowType() ReferenceStaticType
	ReferenceValue(
		interpreter *Interpreter,
		capabilityAddress common.Address,
		resultBorrowType *sema.ReferenceType,
	) ReferenceValue
}

// StorageCapabilityControllerValue

type StorageCapabilityControllerValue struct {
	BorrowType   ReferenceStaticType
	CapabilityID UInt64Value
	TargetPath   PathValue

	// tag is locally cached result of GetTag, and not stored.
	// It is populated when the field `tag` is read.
	tag *StringValue

	// Injected functions.
	// Tags are not stored directly inside the controller
	// to avoid unnecessary storage reads
	// when the controller is loaded for borrowing/checking
	GetTag           func() *StringValue
	SetTag           func(*StringValue)
	TargetFunction   FunctionValue
	RetargetFunction FunctionValue
	DeleteFunction   FunctionValue
}

func NewUnmeteredStorageCapabilityControllerValue(
	borrowType ReferenceStaticType,
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
	borrowType ReferenceStaticType,
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

func (v *StorageCapabilityControllerValue) CapabilityControllerBorrowType() ReferenceStaticType {
	return v.BorrowType
}

func (v *StorageCapabilityControllerValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitStorageCapabilityControllerValue(interpreter, v)
}

func (v *StorageCapabilityControllerValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.TargetPath)
	walkChild(v.CapabilityID)
}

func (v *StorageCapabilityControllerValue) StaticType(_ *Interpreter) StaticType {
	return PrimitiveStaticTypeStorageCapabilityController
}

func (*StorageCapabilityControllerValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v *StorageCapabilityControllerValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *StorageCapabilityControllerValue) RecursiveString(seenReferences SeenReferences) string {
	return format.StorageCapabilityController(
		v.BorrowType.String(),
		v.TargetPath.RecursiveString(seenReferences),
		v.CapabilityID.RecursiveString(seenReferences),
	)
}

func (v *StorageCapabilityControllerValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	common.UseMemory(memoryGauge, common.StorageCapabilityControllerValueStringMemoryUsage)

	return format.StorageCapabilityController(
		v.BorrowType.MeteredString(memoryGauge),
		v.CapabilityID.MeteredString(memoryGauge, seenReferences),
		v.TargetPath.MeteredString(memoryGauge, seenReferences),
	)
}

func (v *StorageCapabilityControllerValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *StorageCapabilityControllerValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
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

func (v *StorageCapabilityControllerValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
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

func (v *StorageCapabilityControllerValue) GetMember(inter *Interpreter, _ LocationRange, name string) Value {

	switch name {
	case sema.StorageCapabilityControllerTypeTagFieldName:
		if v.tag == nil {
			v.tag = v.GetTag()
			if v.tag == nil {
				v.tag = EmptyString
			}
		}
		return v.tag

	case sema.StorageCapabilityControllerTypeCapabilityIDFieldName:
		return v.CapabilityID

	case sema.StorageCapabilityControllerTypeBorrowTypeFieldName:
		return NewTypeValue(inter, v.BorrowType)

	case sema.StorageCapabilityControllerTypeTargetFunctionName:
		return v.TargetFunction

	case sema.StorageCapabilityControllerTypeRetargetFunctionName:
		return v.RetargetFunction

	case sema.StorageCapabilityControllerTypeDeleteFunctionName:
		return v.DeleteFunction
	}

	return nil
}

func (*StorageCapabilityControllerValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Storage capability controllers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *StorageCapabilityControllerValue) SetMember(_ *Interpreter, _ LocationRange, identifier string, value Value) bool {
	switch identifier {
	case sema.StorageCapabilityControllerTypeTagFieldName:
		stringValue, ok := value.(*StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		v.tag = stringValue
		v.SetTag(stringValue)
		return true
	}

	panic(errors.NewUnreachableError())
}

func (v *StorageCapabilityControllerValue) ReferenceValue(
	interpreter *Interpreter,
	capabilityAddress common.Address,
	resultBorrowType *sema.ReferenceType,
) ReferenceValue {
	return NewStorageReferenceValue(
		interpreter,
		resultBorrowType.Authorized,
		capabilityAddress,
		v.TargetPath,
		resultBorrowType.Type,
	)
}

// SetDeleted sets the controller as deleted, i.e. functions panic from now on
func (v *StorageCapabilityControllerValue) SetDeleted(gauge common.MemoryGauge) {

	raiseError := func() {
		panic(errors.NewDefaultUserError("controller is deleted"))
	}

	v.SetTag = func(s *StringValue) {
		raiseError()
	}
	v.GetTag = func() *StringValue {
		raiseError()
		return nil
	}

	panicHostFunction := func(Invocation) Value {
		raiseError()
		return nil
	}

	v.TargetFunction = NewHostFunctionValue(
		gauge,
		sema.StorageCapabilityControllerTypeTargetFunctionType,
		panicHostFunction,
	)
	v.RetargetFunction = NewHostFunctionValue(
		gauge,
		sema.StorageCapabilityControllerTypeRetargetFunctionType,
		panicHostFunction,
	)
	v.DeleteFunction = NewHostFunctionValue(
		gauge,
		sema.StorageCapabilityControllerTypeDeleteFunctionType,
		panicHostFunction,
	)
}
