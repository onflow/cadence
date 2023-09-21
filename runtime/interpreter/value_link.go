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

// Deprecated: LinkValue
type LinkValue interface {
	Value
	isLinkValue()
}

// Deprecated: PathLinkValue
type PathLinkValue struct {
	Type       StaticType
	TargetPath PathValue
}

var _ Value = PathLinkValue{}
var _ atree.Value = PathLinkValue{}
var _ EquatableValue = PathLinkValue{}
var _ LinkValue = PathLinkValue{}

func (PathLinkValue) isValue() {}

func (PathLinkValue) isLinkValue() {}

func (v PathLinkValue) Accept(_ *Interpreter, _ Visitor) {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) Walk(_ *Interpreter, _ func(Value)) {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) StaticType(_ *Interpreter) StaticType {
	panic(errors.NewUnreachableError())
}

func (PathLinkValue) IsImportable(_ *Interpreter) bool {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) String() string {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) RecursiveString(_ SeenReferences) string {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) MeteredString(_ common.MemoryGauge, _ SeenReferences) string {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) Equal(_ *Interpreter, _ LocationRange, _ Value) bool {
	panic(errors.NewUnreachableError())
}

func (PathLinkValue) IsStorable() bool {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	panic(errors.NewUnreachableError())
}

func (PathLinkValue) NeedsStoreTo(_ atree.Address) bool {
	panic(errors.NewUnreachableError())
}

func (PathLinkValue) IsResourceKinded(_ *Interpreter) bool {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) Transfer(
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

func (v PathLinkValue) Clone(_ *Interpreter) Value {
	panic(errors.NewUnreachableError())
}

func (PathLinkValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v PathLinkValue) ByteSize() uint32 {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	panic(errors.NewUnreachableError())
}

func (v PathLinkValue) ChildStorables() []atree.Storable {
	panic(errors.NewUnreachableError())
}

// Deprecated: AccountLinkValue
type AccountLinkValue struct{}

var _ Value = AccountLinkValue{}
var _ atree.Value = AccountLinkValue{}
var _ EquatableValue = AccountLinkValue{}
var _ LinkValue = AccountLinkValue{}

func (AccountLinkValue) isValue() {}

func (AccountLinkValue) isLinkValue() {}

func (v AccountLinkValue) Accept(_ *Interpreter, _ Visitor) {
	panic(errors.NewUnreachableError())
}

func (AccountLinkValue) Walk(_ *Interpreter, _ func(Value)) {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) StaticType(_ *Interpreter) StaticType {
	panic(errors.NewUnreachableError())
}

func (AccountLinkValue) IsImportable(_ *Interpreter) bool {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) String() string {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) RecursiveString(_ SeenReferences) string {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) MeteredString(_ common.MemoryGauge, _ SeenReferences) string {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) Equal(_ *Interpreter, _ LocationRange, _ Value) bool {
	panic(errors.NewUnreachableError())
}

func (AccountLinkValue) IsStorable() bool {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	panic(errors.NewUnreachableError())
}

func (AccountLinkValue) NeedsStoreTo(_ atree.Address) bool {
	panic(errors.NewUnreachableError())
}

func (AccountLinkValue) IsResourceKinded(_ *Interpreter) bool {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) Transfer(
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

func (AccountLinkValue) Clone(_ *Interpreter) Value {
	return AccountLinkValue{}
}

func (AccountLinkValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v AccountLinkValue) ByteSize() uint32 {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	panic(errors.NewUnreachableError())
}

func (v AccountLinkValue) ChildStorables() []atree.Storable {
	panic(errors.NewUnreachableError())
}
