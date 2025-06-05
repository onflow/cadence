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
)

// PlaceholderValue
type PlaceholderValue struct{}

var placeholder Value = PlaceholderValue{}

var _ Value = PlaceholderValue{}

func (PlaceholderValue) IsValue() {}

func (f PlaceholderValue) String() string {
	return f.RecursiveString(SeenReferences{})
}

func (f PlaceholderValue) RecursiveString(_ SeenReferences) string {
	return ""
}

func (f PlaceholderValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	return ""
}

func (f PlaceholderValue) Accept(context ValueVisitContext, visitor Visitor, locationRange LocationRange) {
	// NO-OP
}

func (f PlaceholderValue) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f PlaceholderValue) StaticType(_ ValueStaticTypeContext) StaticType {
	return PrimitiveStaticTypeNever
}

func (PlaceholderValue) IsImportable(_ ValueImportableContext, _ LocationRange) bool {
	return false
}

func (f PlaceholderValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (f PlaceholderValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (PlaceholderValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (PlaceholderValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (f PlaceholderValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	// TODO: actually not needed, value is not storable
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return f
}

func (f PlaceholderValue) Clone(_ ValueCloneContext) Value {
	return f
}

func (PlaceholderValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}
