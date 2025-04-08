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
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// VoidValue

type VoidValue struct{}

var Void Value = VoidValue{}
var VoidStorable atree.Storable = VoidValue{}

var _ Value = VoidValue{}
var _ atree.Storable = VoidValue{}
var _ EquatableValue = VoidValue{}

func (VoidValue) IsValue() {}

func (v VoidValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitVoidValue(interpreter, v)
}

func (VoidValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (VoidValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeVoid)
}

func (VoidValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return sema.VoidType.Importable
}

func (VoidValue) String() string {
	return format.Void
}

func (v VoidValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v VoidValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(context, common.VoidStringMemoryUsage)
	return v.String()
}

func (v VoidValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v VoidValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	_, ok := other.(VoidValue)
	return ok
}

func (v VoidValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (VoidValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (VoidValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v VoidValue) Transfer(
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

func (v VoidValue) Clone(_ ValueCloneContext) Value {
	return v
}

func (VoidValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (VoidValue) ByteSize() uint32 {
	return uint32(len(cborVoidValue))
}

func (v VoidValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (VoidValue) ChildStorables() []atree.Storable {
	return nil
}
