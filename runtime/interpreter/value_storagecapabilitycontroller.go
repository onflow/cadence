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
	"github.com/onflow/cadence/runtime/format"
)

// StorageCapabilityControllerValue

type StorageCapabilityControllerValue struct {
	BorrowType   StaticType
	TargetPath   PathValue
	CapabilityID UInt64Value
}

func NewUnmeteredStorageCapabilityControllerValue(
	staticType StaticType,
	targetPath PathValue,
	capabilityID UInt64Value,
) *StorageCapabilityControllerValue {
	return &StorageCapabilityControllerValue{
		BorrowType:   staticType,
		TargetPath:   targetPath,
		CapabilityID: capabilityID,
	}
}

func NewStorageCapabilityControllerValue(
	memoryGauge common.MemoryGauge,
	staticType StaticType,
	targetPath PathValue,
	capabilityID UInt64Value,
) *StorageCapabilityControllerValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.StorageCapabilityControllerValueMemoryUsage)
	return NewUnmeteredStorageCapabilityControllerValue(
		staticType,
		targetPath,
		capabilityID,
	)
}

var _ Value = &StorageCapabilityControllerValue{}
var _ atree.Value = &StorageCapabilityControllerValue{}
var _ EquatableValue = &StorageCapabilityControllerValue{}

func (*StorageCapabilityControllerValue) IsValue() {}

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
