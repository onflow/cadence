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
	GetCapability func(inter *Interpreter) *IDCapabilityValue
	GetTag        func(inter *Interpreter) *StringValue
	SetTag        func(inter *Interpreter, tag *StringValue)
	Delete        func(inter *Interpreter, locationRange LocationRange)
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

func (*AccountCapabilityControllerValue) isValue() {}

func (*AccountCapabilityControllerValue) isCapabilityControllerValue() {}

func (v *AccountCapabilityControllerValue) CapabilityControllerBorrowType() *ReferenceStaticType {
	return v.BorrowType
}

func (v *AccountCapabilityControllerValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitAccountCapabilityControllerValue(interpreter, v)
}

func (v *AccountCapabilityControllerValue) Walk(_ *Interpreter, walkChild func(Value), _ LocationRange) {
	walkChild(v.CapabilityID)
}

func (v *AccountCapabilityControllerValue) StaticType(_ ValueStaticTypeContext) StaticType {
	return PrimitiveStaticTypeAccountCapabilityController
}

func (*AccountCapabilityControllerValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
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

func (v *AccountCapabilityControllerValue) MeteredString(
	interpreter *Interpreter,
	seenReferences SeenReferences,
	locationRange LocationRange,
) string {
	common.UseMemory(interpreter, common.AccountCapabilityControllerValueStringMemoryUsage)

	return format.AccountCapabilityController(
		v.BorrowType.MeteredString(interpreter),
		v.CapabilityID.MeteredString(interpreter, seenReferences, locationRange),
	)
}

func (v *AccountCapabilityControllerValue) ConformsToStaticType(
	_ *Interpreter,
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

func (*AccountCapabilityControllerValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v *AccountCapabilityControllerValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *AccountCapabilityControllerValue) Clone(_ *Interpreter) Value {
	return &AccountCapabilityControllerValue{
		BorrowType:   v.BorrowType,
		CapabilityID: v.CapabilityID,
	}
}

func (v *AccountCapabilityControllerValue) DeepRemove(_ *Interpreter, _ bool) {
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

func (v *AccountCapabilityControllerValue) GetMember(inter *Interpreter, _ LocationRange, name string) (result Value) {
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
		return v.GetTag(inter)

	case sema.AccountCapabilityControllerTypeSetTagFunctionName:
		if v.setTagFunction == nil {
			v.setTagFunction = v.newSetTagFunction(inter)
		}
		return v.setTagFunction

	case sema.AccountCapabilityControllerTypeCapabilityIDFieldName:
		return v.CapabilityID

	case sema.AccountCapabilityControllerTypeBorrowTypeFieldName:
		return NewTypeValue(inter, v.BorrowType)

	case sema.AccountCapabilityControllerTypeCapabilityFieldName:
		return v.GetCapability(inter)

	case sema.AccountCapabilityControllerTypeDeleteFunctionName:
		if v.deleteFunction == nil {
			v.deleteFunction = v.newDeleteFunction(inter)
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
	inter *Interpreter,
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
		v.SetTag(inter, stringValue)
		return true
	}

	panic(errors.NewUnreachableError())
}

func (v *AccountCapabilityControllerValue) ControllerCapabilityID() UInt64Value {
	return v.CapabilityID
}

func (v *AccountCapabilityControllerValue) ReferenceValue(
	interpreter *Interpreter,
	capabilityAddress common.Address,
	resultBorrowType *sema.ReferenceType,
	locationRange LocationRange,
) ReferenceValue {
	config := interpreter.SharedState.Config

	account := config.AccountHandler(interpreter, AddressValue(capabilityAddress))

	// Account must be of `Account` type.
	interpreter.ExpectType(
		account,
		sema.AccountType,
		EmptyLocationRange,
	)

	authorization := ConvertSemaAccessToStaticAuthorization(
		interpreter,
		resultBorrowType.Authorization,
	)
	return NewEphemeralReferenceValue(
		interpreter,
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
	inter *Interpreter,
	funcType *sema.FunctionType,
	f func(invocation Invocation) Value,
) FunctionValue {
	return deletionCheckedFunctionValue{
		FunctionValue: NewBoundHostFunctionValue(
			inter,
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
	inter *Interpreter,
) FunctionValue {
	return v.newHostFunctionValue(
		inter,
		sema.AccountCapabilityControllerTypeDeleteFunctionType,
		func(invocation Invocation) Value {
			inter := invocation.Interpreter
			locationRange := invocation.LocationRange

			v.Delete(inter, locationRange)

			v.deleted = true

			return Void
		},
	)
}

func (v *AccountCapabilityControllerValue) newSetTagFunction(
	inter *Interpreter,
) FunctionValue {
	return v.newHostFunctionValue(
		inter,
		sema.AccountCapabilityControllerTypeSetTagFunctionType,
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
