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

// AccountCapabilityControllerValue

type AccountCapabilityControllerValue struct {
	BorrowType   ReferenceStaticType
	CapabilityID UInt64Value

	// tag is locally cached result of GetTag, and not stored.
	// It is populated when the field `tag` is read.
	tag *StringValue

	// Injected functions
	// Tags are not stored directly inside the controller
	// to avoid unnecessary storage reads
	// when the controller is loaded for borrowing/checking
	GetTag         func() *StringValue
	SetTag         func(*StringValue)
	DeleteFunction FunctionValue
}

func NewUnmeteredAccountCapabilityControllerValue(
	borrowType ReferenceStaticType,
	capabilityID UInt64Value,
) *AccountCapabilityControllerValue {
	return &AccountCapabilityControllerValue{
		BorrowType:   borrowType,
		CapabilityID: capabilityID,
	}
}

func NewAccountCapabilityControllerValue(
	memoryGauge common.MemoryGauge,
	borrowType ReferenceStaticType,
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

func (v *AccountCapabilityControllerValue) CapabilityControllerBorrowType() ReferenceStaticType {
	return v.BorrowType
}

func (v *AccountCapabilityControllerValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitAccountCapabilityControllerValue(interpreter, v)
}

func (v *AccountCapabilityControllerValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.CapabilityID)
}

func (v *AccountCapabilityControllerValue) StaticType(_ *Interpreter) StaticType {
	return PrimitiveStaticTypeAccountCapabilityController
}

func (*AccountCapabilityControllerValue) IsImportable(_ *Interpreter) bool {
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

func (v *AccountCapabilityControllerValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	common.UseMemory(memoryGauge, common.AccountCapabilityControllerValueStringMemoryUsage)

	return format.AccountCapabilityController(
		v.BorrowType.MeteredString(memoryGauge),
		v.CapabilityID.MeteredString(memoryGauge, seenReferences),
	)
}

func (v *AccountCapabilityControllerValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *AccountCapabilityControllerValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherController, ok := other.(*AccountCapabilityControllerValue)
	if !ok {
		return false
	}

	return otherController.BorrowType.Equal(v.BorrowType) &&
		otherController.CapabilityID.Equal(interpreter, locationRange, v.CapabilityID)
}

func (*AccountCapabilityControllerValue) IsStorable() bool {
	return true
}

func (v *AccountCapabilityControllerValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (*AccountCapabilityControllerValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*AccountCapabilityControllerValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *AccountCapabilityControllerValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
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

func (v *AccountCapabilityControllerValue) DeepRemove(_ *Interpreter) {
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

func (v *AccountCapabilityControllerValue) GetMember(inter *Interpreter, _ LocationRange, name string) Value {

	switch name {
	case sema.AccountCapabilityControllerTypeTagFieldName:
		if v.tag == nil {
			v.tag = v.GetTag()
			if v.tag == nil {
				v.tag = EmptyString
			}
		}
		return v.tag

	case sema.AccountCapabilityControllerTypeCapabilityIDFieldName:
		return v.CapabilityID

	case sema.AccountCapabilityControllerTypeBorrowTypeFieldName:
		return NewTypeValue(inter, v.BorrowType)

	case sema.AccountCapabilityControllerTypeDeleteFunctionName:
		return v.DeleteFunction
	}

	return nil
}

func (*AccountCapabilityControllerValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Storage capability controllers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *AccountCapabilityControllerValue) SetMember(_ *Interpreter, _ LocationRange, identifier string, value Value) bool {
	switch identifier {
	case sema.AccountCapabilityControllerTypeTagFieldName:
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

func (v *AccountCapabilityControllerValue) ReferenceValue(
	interpreter *Interpreter,
	capabilityAddress common.Address,
	resultBorrowType *sema.ReferenceType,
) ReferenceValue {
	return NewAccountReferenceValue(
		interpreter,
		capabilityAddress,
		resultBorrowType.Type,
	)
}

// SetDeleted sets the controller as deleted, i.e. functions panic from now on
func (v *AccountCapabilityControllerValue) SetDeleted(gauge common.MemoryGauge) {

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

	v.DeleteFunction = NewHostFunctionValue(
		gauge,
		sema.AccountCapabilityControllerTypeDeleteFunctionType,
		panicHostFunction,
	)
}
