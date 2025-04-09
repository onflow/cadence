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

// AccountCapabilityControllerValue

type AccountCapabilityControllerValue struct {
	BorrowType   *ReferenceStaticType
	CapabilityID UInt64Value

	// deleted indicates if the controller got deleted. Not stored
	deleted bool

	// Lazily initialized function values.
	// Host functions based on injected functions (see below).
	deleteFunction FunctionValue
	setTagFunction FunctionValue

	// Injected functions.
	// Tags are not stored directly inside the controller
	// to avoid unnecessary storage reads
	// when the controller is loaded for borrowing/checking
	GetCapability func(common.MemoryGauge) *IDCapabilityValue
	GetTag        func(StorageReader) *StringValue
	SetTag        func(storageWriter StorageWriter, tag *StringValue)
	Delete        func(context CapabilityControllerContext, locationRange LocationRange)
}

func NewUnmeteredAccountCapabilityControllerValue(
	borrowType *ReferenceStaticType,
	capabilityID UInt64Value,
) *AccountCapabilityControllerValue {
	return &AccountCapabilityControllerValue{
		BorrowType:   borrowType,
		CapabilityID: capabilityID,
	}
}

func NewAccountCapabilityControllerValue(
	memoryGauge common.MemoryGauge,
	borrowType *ReferenceStaticType,
	capabilityID UInt64Value,
) *AccountCapabilityControllerValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.AccountCapabilityControllerValueMemoryUsage)
	return NewUnmeteredAccountCapabilityControllerValue(
		borrowType,
		capabilityID,
	)
}

var _ Value = &AccountCapabilityControllerValue{}
var _ atree.Value = &AccountCapabilityControllerValue{}
var _ EquatableValue = &AccountCapabilityControllerValue{}
var _ CapabilityControllerValue = &AccountCapabilityControllerValue{}
var _ MemberAccessibleValue = &AccountCapabilityControllerValue{}

func (*AccountCapabilityControllerValue) IsValue() {}

func (*AccountCapabilityControllerValue) isCapabilityControllerValue() {}

func (v *AccountCapabilityControllerValue) CapabilityControllerBorrowType() *ReferenceStaticType {
	return v.BorrowType
}

func (v *AccountCapabilityControllerValue) Accept(context ValueVisitContext, visitor Visitor, _ LocationRange) {
	visitor.VisitAccountCapabilityControllerValue(context, v)
}

func (v *AccountCapabilityControllerValue) Walk(_ ValueWalkContext, walkChild func(Value), _ LocationRange) {
	walkChild(v.CapabilityID)
}

func (v *AccountCapabilityControllerValue) StaticType(_ ValueStaticTypeContext) StaticType {
	return PrimitiveStaticTypeAccountCapabilityController
}

func (*AccountCapabilityControllerValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return false
}

func (v *AccountCapabilityControllerValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *AccountCapabilityControllerValue) RecursiveString(seenReferences SeenReferences) string {
	return format.AccountCapabilityController(
		v.BorrowType.String(),
		v.CapabilityID.RecursiveString(seenReferences),
	)
}

func (v *AccountCapabilityControllerValue) MeteredString(context ValueStringContext, seenReferences SeenReferences, locationRange LocationRange) string {
	common.UseMemory(context, common.AccountCapabilityControllerValueStringMemoryUsage)

	return format.AccountCapabilityController(
		v.BorrowType.MeteredString(context),
		v.CapabilityID.MeteredString(context, seenReferences, locationRange),
	)
}

func (v *AccountCapabilityControllerValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *AccountCapabilityControllerValue) Equal(context ValueComparisonContext, locationRange LocationRange, other Value) bool {
	otherController, ok := other.(*AccountCapabilityControllerValue)
	if !ok {
		return false
	}

	return otherController.BorrowType.Equal(v.BorrowType) &&
		otherController.CapabilityID.Equal(context, locationRange, v.CapabilityID)
}

func (*AccountCapabilityControllerValue) IsStorable() bool {
	return true
}

func (v *AccountCapabilityControllerValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (
	atree.Storable,
	error,
) {
	return values.MaybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (*AccountCapabilityControllerValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*AccountCapabilityControllerValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v *AccountCapabilityControllerValue) Transfer(
	transferContext ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		RemoveReferencedSlab(transferContext, storable)
	}
	return v
}

func (v *AccountCapabilityControllerValue) Clone(_ ValueCloneContext) Value {
	return &AccountCapabilityControllerValue{
		BorrowType:   v.BorrowType,
		CapabilityID: v.CapabilityID,
	}
}

func (v *AccountCapabilityControllerValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v *AccountCapabilityControllerValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *AccountCapabilityControllerValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *AccountCapabilityControllerValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.CapabilityID,
	}
}

type deletionCheckedFunctionValue struct {
	FunctionValue
}

func (v *AccountCapabilityControllerValue) GetMember(context MemberAccessibleContext, _ LocationRange, name string) (result Value) {
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
	case sema.AccountCapabilityControllerTypeTagFieldName:
		return v.GetTag(context)

	case sema.AccountCapabilityControllerTypeSetTagFunctionName:
		if v.setTagFunction == nil {
			v.setTagFunction = v.newSetTagFunction(context)
		}
		return v.setTagFunction

	case sema.AccountCapabilityControllerTypeCapabilityIDFieldName:
		return v.CapabilityID

	case sema.AccountCapabilityControllerTypeBorrowTypeFieldName:
		return NewTypeValue(context, v.BorrowType)

	case sema.AccountCapabilityControllerTypeCapabilityFieldName:
		return v.GetCapability(context)

	case sema.AccountCapabilityControllerTypeDeleteFunctionName:
		if v.deleteFunction == nil {
			v.deleteFunction = v.newDeleteFunction(context)
		}
		return v.deleteFunction

		// NOTE: when adding new functions, ensure checkDeleted is called,
		// by e.g. using AccountCapabilityControllerValue.newHostFunction
	}

	return nil
}

func (*AccountCapabilityControllerValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Account capability controllers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *AccountCapabilityControllerValue) SetMember(
	context MemberAccessibleContext,
	_ LocationRange,
	identifier string,
	value Value,
) bool {
	// NOTE: check if controller is already deleted
	v.checkDeleted()

	switch identifier {
	case sema.AccountCapabilityControllerTypeTagFieldName:
		stringValue, ok := value.(*StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}
		v.SetTag(context, stringValue)
		return true
	}

	panic(errors.NewUnreachableError())
}

func (v *AccountCapabilityControllerValue) ControllerCapabilityID() UInt64Value {
	return v.CapabilityID
}

func (v *AccountCapabilityControllerValue) ReferenceValue(
	context ValueCapabilityControllerReferenceValueContext,
	capabilityAddress common.Address,
	resultBorrowType *sema.ReferenceType,
	locationRange LocationRange,
) ReferenceValue {

	accountHandler := context.AccountHandler()
	account := accountHandler(context, AddressValue(capabilityAddress))

	// Account must be of `Account` type.
	ExpectType(
		context,
		account,
		sema.AccountType,
		EmptyLocationRange,
	)

	authorization := ConvertSemaAccessToStaticAuthorization(
		context,
		resultBorrowType.Authorization,
	)
	return NewEphemeralReferenceValue(
		context,
		authorization,
		account,
		resultBorrowType.Type,
		locationRange,
	)
}

// checkDeleted checks if the controller is deleted,
// and panics if it is.
func (v *AccountCapabilityControllerValue) checkDeleted() {
	if v.deleted {
		panic(errors.NewDefaultUserError("controller is deleted"))
	}
}

func (v *AccountCapabilityControllerValue) newHostFunctionValue(
	context FunctionCreationContext,
	funcType *sema.FunctionType,
	f func(invocation Invocation) Value,
) FunctionValue {
	return deletionCheckedFunctionValue{
		FunctionValue: NewBoundHostFunctionValue(
			context,
			v,
			funcType,
			func(v *AccountCapabilityControllerValue, invocation Invocation) Value {
				// NOTE: check if controller is already deleted
				v.checkDeleted()

				return f(invocation)
			},
		),
	}
}

func (v *AccountCapabilityControllerValue) newDeleteFunction(
	context FunctionCreationContext,
) FunctionValue {
	return v.newHostFunctionValue(
		context,
		sema.AccountCapabilityControllerTypeDeleteFunctionType,
		func(invocation Invocation) Value {
			invocationContext := invocation.InvocationContext
			locationRange := invocation.LocationRange

			v.Delete(invocationContext, locationRange)

			v.deleted = true

			return Void
		},
	)
}

func (v *AccountCapabilityControllerValue) newSetTagFunction(
	context FunctionCreationContext,
) FunctionValue {
	return v.newHostFunctionValue(
		context,
		sema.AccountCapabilityControllerTypeSetTagFunctionType,
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
