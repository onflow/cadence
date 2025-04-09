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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

type CapabilityControllerValue interface {
	Value
	isCapabilityControllerValue()
	CapabilityControllerBorrowType() *ReferenceStaticType
	ReferenceValue(
		context ValueCapabilityControllerReferenceValueContext,
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
	GetCapability func(common.MemoryGauge) *IDCapabilityValue
	GetTag        func(storageReader StorageReader) *StringValue
	SetTag        func(storageWriter StorageWriter, tag *StringValue)
	Delete        func(context CapabilityControllerContext, locationRange LocationRange)
	SetTarget     func(context CapabilityControllerContext, locationRange LocationRange, target PathValue)
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

func (*StorageCapabilityControllerValue) IsValue() {}

func (*StorageCapabilityControllerValue) isCapabilityControllerValue() {}

func (v *StorageCapabilityControllerValue) CapabilityControllerBorrowType() *ReferenceStaticType {
	return v.BorrowType
}

func (v *StorageCapabilityControllerValue) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitStorageCapabilityControllerValue(context, v)
}

func (v *StorageCapabilityControllerValue) Walk(_ ValueWalkContext, walkChild func(Value), _ LocationRange) {
	walkChild(v.TargetPath)
	walkChild(v.CapabilityID)
}

func (v *StorageCapabilityControllerValue) StaticType(_ ValueStaticTypeContext) StaticType {
	return PrimitiveStaticTypeStorageCapabilityController
}

func (*StorageCapabilityControllerValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
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

func (v *StorageCapabilityControllerValue) MeteredString(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
	common.UseMemory(context, common.StorageCapabilityControllerValueStringMemoryUsage)

	return format.StorageCapabilityController(
		v.BorrowType.MeteredString(context),
		v.CapabilityID.MeteredString(context, seenReferences, locationRange),
		v.TargetPath.MeteredString(context, seenReferences, locationRange),
	)
}

func (v *StorageCapabilityControllerValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *StorageCapabilityControllerValue) Equal(context ValueComparisonContext, locationRange LocationRange, other Value) bool {
	otherController, ok := other.(*StorageCapabilityControllerValue)
	if !ok {
		return false
	}

	return otherController.TargetPath.Equal(context, locationRange, v.TargetPath) &&
		otherController.BorrowType.Equal(v.BorrowType) &&
		otherController.CapabilityID.Equal(context, locationRange, v.CapabilityID)
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
	return values.MaybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (*StorageCapabilityControllerValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*StorageCapabilityControllerValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v *StorageCapabilityControllerValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return v
}

func (v *StorageCapabilityControllerValue) Clone(context ValueCloneContext) Value {
	return &StorageCapabilityControllerValue{
		TargetPath:   v.TargetPath.Clone(context).(PathValue),
		BorrowType:   v.BorrowType,
		CapabilityID: v.CapabilityID,
	}
}

func (v *StorageCapabilityControllerValue) DeepRemove(_ ValueRemoveContext, _ bool) {
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

func (v *StorageCapabilityControllerValue) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) (result Value) {
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
		return v.GetTag(context)

	case sema.StorageCapabilityControllerTypeSetTagFunctionName:
		if v.setTagFunction == nil {
			v.setTagFunction = v.newSetTagFunction(context)
		}
		return v.setTagFunction

	case sema.StorageCapabilityControllerTypeCapabilityIDFieldName:
		return v.CapabilityID

	case sema.StorageCapabilityControllerTypeBorrowTypeFieldName:
		return NewTypeValue(context, v.BorrowType)

	case sema.StorageCapabilityControllerTypeCapabilityFieldName:
		return v.GetCapability(context)

	case sema.StorageCapabilityControllerTypeDeleteFunctionName:
		if v.deleteFunction == nil {
			v.deleteFunction = v.newDeleteFunction(context)
		}
		return v.deleteFunction

	case sema.StorageCapabilityControllerTypeTargetFunctionName:
		if v.targetFunction == nil {
			v.targetFunction = v.newTargetFunction(context)
		}
		return v.targetFunction

	case sema.StorageCapabilityControllerTypeRetargetFunctionName:
		if v.retargetFunction == nil {
			v.retargetFunction = v.newRetargetFunction(context)
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
	context MemberAccessibleContext,
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
		v.SetTag(context, stringValue)
		return true
	}

	panic(errors.NewUnreachableError())
}

func (v *StorageCapabilityControllerValue) ControllerCapabilityID() UInt64Value {
	return v.CapabilityID
}

func (v *StorageCapabilityControllerValue) ReferenceValue(
	context ValueCapabilityControllerReferenceValueContext,
	capabilityAddress common.Address,
	resultBorrowType *sema.ReferenceType,
	_ LocationRange,
) ReferenceValue {
	authorization := ConvertSemaAccessToStaticAuthorization(
		context,
		resultBorrowType.Authorization,
	)
	return NewStorageReferenceValue(
		context,
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
	context FunctionCreationContext,
	funcType *sema.FunctionType,
	f func(invocation Invocation) Value,
) FunctionValue {
	return deletionCheckedFunctionValue{
		FunctionValue: NewBoundHostFunctionValue(
			context,
			v,
			funcType,
			func(v *StorageCapabilityControllerValue, invocation Invocation) Value {
				// NOTE: check if controller is already deleted
				v.checkDeleted()

				return f(invocation)
			},
		),
	}
}

func (v *StorageCapabilityControllerValue) newDeleteFunction(
	context FunctionCreationContext,
) FunctionValue {
	return v.newHostFunctionValue(
		context,
		sema.StorageCapabilityControllerTypeDeleteFunctionType,
		func(invocation Invocation) Value {
			inter := invocation.InvocationContext
			locationRange := invocation.LocationRange

			v.Delete(inter, locationRange)

			v.deleted = true

			return Void
		},
	)
}

func (v *StorageCapabilityControllerValue) newTargetFunction(
	context FunctionCreationContext,
) FunctionValue {
	return v.newHostFunctionValue(
		context,
		sema.StorageCapabilityControllerTypeTargetFunctionType,
		func(invocation Invocation) Value {
			return v.TargetPath
		},
	)
}

func (v *StorageCapabilityControllerValue) newRetargetFunction(
	context FunctionCreationContext,
) FunctionValue {
	return v.newHostFunctionValue(
		context,
		sema.StorageCapabilityControllerTypeRetargetFunctionType,
		func(invocation Invocation) Value {
			inter := invocation.InvocationContext
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
	context FunctionCreationContext,
) FunctionValue {
	return v.newHostFunctionValue(
		context,
		sema.StorageCapabilityControllerTypeSetTagFunctionType,
		func(invocation Invocation) Value {
			inter := invocation.InvocationContext

			newTagValue, ok := invocation.Arguments[0].(*StringValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			v.SetTag(inter, newTagValue)

			return Void
		},
	)
}
