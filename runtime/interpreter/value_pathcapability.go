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
)

// Deprecated: PathCapabilityValue
type PathCapabilityValue struct {
	BorrowType StaticType
	Path       PathValue
	Address    AddressValue
}

var _ Value = &PathCapabilityValue{}
var _ atree.Storable = &PathCapabilityValue{}
var _ EquatableValue = &PathCapabilityValue{}
var _ MemberAccessibleValue = &PathCapabilityValue{}

func (*PathCapabilityValue) isValue() {}

func (*PathCapabilityValue) isCapabilityValue() {}

func (v *PathCapabilityValue) Accept(_ *Interpreter, _ Visitor) {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) Walk(_ *Interpreter, _ func(Value)) {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) StaticType(_ *Interpreter) StaticType {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) IsImportable(_ *Interpreter) bool {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) String() string {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) RecursiveString(_ SeenReferences) string {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) MeteredString(_ common.MemoryGauge, _ SeenReferences) string {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) GetMember(_ *Interpreter, _ LocationRange, _ string) Value {
	panic(errors.NewUnreachableError())
}

func (*PathCapabilityValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	panic(errors.NewUnreachableError())
}

func (*PathCapabilityValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	panic(errors.NewUnreachableError())
}

func (*PathCapabilityValue) IsStorable() bool {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) Storable(
	_ atree.SlabStorage,
	_ atree.Address,
	_ uint64,
) (atree.Storable, error) {
	panic(errors.NewUnreachableError())
}

func (*PathCapabilityValue) NeedsStoreTo(_ atree.Address) bool {
	panic(errors.NewUnreachableError())
}

func (*PathCapabilityValue) IsResourceKinded(_ *Interpreter) bool {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) Transfer(
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

func (v *PathCapabilityValue) Clone(_ *Interpreter) Value {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) DeepRemove(interpreter *Interpreter) {
	v.Address.DeepRemove(interpreter)
	v.Path.DeepRemove(interpreter)
}

func (v *PathCapabilityValue) ByteSize() uint32 {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) ChildStorables() []atree.Storable {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) Encode(_ *atree.Encoder) error {
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) AddressPath() AddressPath {
	return AddressPath{
		Address: common.Address(v.Address),
		Path:    v.Path,
	}
}
