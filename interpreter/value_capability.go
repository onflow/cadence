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
)

const InvalidCapabilityID UInt64Value = 0

// CapabilityValue

// TODO: remove once migration to Cadence 1.0 / ID capabilities is complete
type CapabilityValue interface {
	EquatableValue
	MemberAccessibleValue
	atree.Storable
	isCapabilityValue()
	Address() AddressValue
}

// IDCapabilityValue

type IDCapabilityValue struct {
	BorrowType StaticType
	address    AddressValue
	ID         UInt64Value
}

func NewUnmeteredCapabilityValue(
	id UInt64Value,
	address AddressValue,
	borrowType StaticType,
) *IDCapabilityValue {
	return &IDCapabilityValue{
		ID:         id,
		address:    address,
		BorrowType: borrowType,
	}
}

func NewCapabilityValue(
	memoryGauge common.MemoryGauge,
	id UInt64Value,
	address AddressValue,
	borrowType StaticType,
) *IDCapabilityValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.CapabilityValueMemoryUsage)
	return NewUnmeteredCapabilityValue(id, address, borrowType)
}

func NewInvalidCapabilityValue(
	memoryGauge common.MemoryGauge,
	address AddressValue,
	borrowType StaticType,
) *IDCapabilityValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.CapabilityValueMemoryUsage)
	return &IDCapabilityValue{
		ID:         InvalidCapabilityID,
		address:    address,
		BorrowType: borrowType,
	}
}

var _ Value = &IDCapabilityValue{}
var _ atree.Storable = &IDCapabilityValue{}
var _ EquatableValue = &IDCapabilityValue{}
var _ MemberAccessibleValue = &IDCapabilityValue{}
var _ CapabilityValue = &IDCapabilityValue{}

func (*IDCapabilityValue) isValue() {}

func (*IDCapabilityValue) isCapabilityValue() {}

func (v *IDCapabilityValue) isInvalid() bool {
	return v.ID == InvalidCapabilityID
}

func (v *IDCapabilityValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitCapabilityValue(interpreter, v)
}

func (v *IDCapabilityValue) Walk(_ *Interpreter, walkChild func(Value), _ LocationRange) {
	walkChild(v.ID)
	walkChild(v.address)
}

func (v *IDCapabilityValue) StaticType(staticTypeGetter StaticTypeGetter) StaticType {
	return NewCapabilityStaticType(
		staticTypeGetter,
		v.BorrowType,
	)
}

func (v *IDCapabilityValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (v *IDCapabilityValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *IDCapabilityValue) RecursiveString(seenReferences SeenReferences) string {
	return format.Capability(
		v.BorrowType.String(),
		v.address.RecursiveString(seenReferences),
		v.ID.RecursiveString(seenReferences),
	)
}

func (v *IDCapabilityValue) MeteredString(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
	common.UseMemory(interpreter, common.IDCapabilityValueStringMemoryUsage)

	return format.Capability(
		v.BorrowType.MeteredString(interpreter),
		v.address.MeteredString(interpreter, seenReferences, locationRange),
		v.ID.MeteredString(interpreter, seenReferences, locationRange),
	)
}

func (v *IDCapabilityValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.CapabilityTypeBorrowFunctionName:
		// this function will panic already if this conversion fails
		borrowType, _ := interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		return interpreter.capabilityBorrowFunction(v, v.address, v.ID, borrowType)

	case sema.CapabilityTypeCheckFunctionName:
		// this function will panic already if this conversion fails
		borrowType, _ := interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		return interpreter.capabilityCheckFunction(v, v.address, v.ID, borrowType)

	case sema.CapabilityTypeAddressFieldName:
		return v.address

	case sema.CapabilityTypeIDFieldName:
		return v.ID
	}

	return nil
}

func (*IDCapabilityValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Capabilities have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*IDCapabilityValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Capabilities have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *IDCapabilityValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *IDCapabilityValue) Equal(context ComparisonContext, locationRange LocationRange, other Value) bool {
	otherCapability, ok := other.(*IDCapabilityValue)
	if !ok {
		return false
	}

	return otherCapability.ID == v.ID &&
		otherCapability.address.Equal(context, locationRange, v.address) &&
		otherCapability.BorrowType.Equal(v.BorrowType)
}

func (*IDCapabilityValue) IsStorable() bool {
	return true
}

func (v *IDCapabilityValue) Address() AddressValue {
	return v.address
}

func (v *IDCapabilityValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return maybeLargeImmutableStorable(
		v,
		storage,
		address,
		maxInlineSize,
	)
}

func (*IDCapabilityValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*IDCapabilityValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *IDCapabilityValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		v.DeepRemove(interpreter, true)
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *IDCapabilityValue) Clone(interpreter *Interpreter) Value {
	return NewUnmeteredCapabilityValue(
		v.ID,
		v.address.Clone(interpreter).(AddressValue),
		v.BorrowType,
	)
}

func (v *IDCapabilityValue) DeepRemove(interpreter *Interpreter, _ bool) {
	v.address.DeepRemove(interpreter, false)
}

func (v *IDCapabilityValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *IDCapabilityValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *IDCapabilityValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.address,
	}
}
