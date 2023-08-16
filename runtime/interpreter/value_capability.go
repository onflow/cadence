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

// CapabilityValue

type CapabilityValue struct {
	BorrowType StaticType
	Address    AddressValue
	ID         UInt64Value
}

func NewUnmeteredCapabilityValue(
	id UInt64Value,
	address AddressValue,
	borrowType StaticType,
) *CapabilityValue {
	return &CapabilityValue{
		ID:         id,
		Address:    address,
		BorrowType: borrowType,
	}
}

func NewCapabilityValue(
	memoryGauge common.MemoryGauge,
	id UInt64Value,
	address AddressValue,
	borrowType StaticType,
) *CapabilityValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.CapabilityValueMemoryUsage)
	return NewUnmeteredCapabilityValue(id, address, borrowType)
}

var _ Value = &CapabilityValue{}
var _ atree.Storable = &CapabilityValue{}
var _ EquatableValue = &CapabilityValue{}
var _ MemberAccessibleValue = &CapabilityValue{}

func (*CapabilityValue) isValue() {}

func (v *CapabilityValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitCapabilityValue(interpreter, v)
}

func (v *CapabilityValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.ID)
	walkChild(v.Address)
}

func (v *CapabilityValue) StaticType(inter *Interpreter) StaticType {
	return NewCapabilityStaticType(
		inter,
		v.BorrowType,
	)
}

func (v *CapabilityValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v *CapabilityValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *CapabilityValue) RecursiveString(seenReferences SeenReferences) string {
	return format.Capability(
		v.BorrowType.String(),
		v.Address.RecursiveString(seenReferences),
		v.ID.RecursiveString(seenReferences),
	)
}

func (v *CapabilityValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	common.UseMemory(memoryGauge, common.CapabilityValueStringMemoryUsage)

	return format.Capability(
		v.BorrowType.MeteredString(memoryGauge),
		v.Address.MeteredString(memoryGauge, seenReferences),
		v.ID.MeteredString(memoryGauge, seenReferences),
	)
}

func (v *CapabilityValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.CapabilityTypeBorrowFunctionName:
		// this function will panic already if this conversion fails
		borrowType, _ := interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		return interpreter.capabilityBorrowFunction(v.Address, v.ID, borrowType)

	case sema.CapabilityTypeCheckFunctionName:
		// this function will panic already if this conversion fails
		borrowType, _ := interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		return interpreter.capabilityCheckFunction(v.Address, v.ID, borrowType)

	case sema.CapabilityTypeAddressFieldName:
		return v.Address

	case sema.CapabilityTypeIDFieldName:
		return v.ID
	}

	return nil
}

func (*CapabilityValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Capabilities have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*CapabilityValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Capabilities have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *CapabilityValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *CapabilityValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherCapability, ok := other.(*CapabilityValue)
	if !ok {
		return false
	}

	return otherCapability.ID == v.ID &&
		otherCapability.Address.Equal(interpreter, locationRange, v.Address) &&
		otherCapability.BorrowType.Equal(v.BorrowType)
}

func (*CapabilityValue) IsStorable() bool {
	return true
}

func (v *CapabilityValue) Storable(
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

func (*CapabilityValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*CapabilityValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *CapabilityValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		v.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *CapabilityValue) Clone(interpreter *Interpreter) Value {
	return NewUnmeteredCapabilityValue(
		v.ID,
		v.Address.Clone(interpreter).(AddressValue),
		v.BorrowType,
	)
}

func (v *CapabilityValue) DeepRemove(interpreter *Interpreter) {
	v.Address.DeepRemove(interpreter)
}

func (v *CapabilityValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *CapabilityValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *CapabilityValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.Address,
	}
}
