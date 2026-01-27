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

var _ Value = &PlaceholderValue{}

func (*PlaceholderValue) IsValue() {}

func (v *PlaceholderValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (*PlaceholderValue) RecursiveString(_ SeenReferences) string {
	return ""
}

func (*PlaceholderValue) MeteredString(
	_ ValueStringContext,
	_ SeenReferences,
) string {
	return ""
}

func (*PlaceholderValue) Accept(_ ValueVisitContext, _ Visitor) {
	// NO-OP
}

func (*PlaceholderValue) Walk(_ ValueWalkContext, _ func(Value)) {
	// NO-OP
}

func (*PlaceholderValue) StaticType(_ ValueStaticTypeContext) StaticType {
	return PrimitiveStaticTypeInvalid
}

func (*PlaceholderValue) IsImportable(_ ValueImportableContext) bool {
	return false
}

func (*PlaceholderValue) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *PlaceholderValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint32) (atree.Storable, error) {
	return NonStorable{Value: v}, nil
}

func (*PlaceholderValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*PlaceholderValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v *PlaceholderValue) Transfer(
	context ValueTransferContext,
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
	return v
}

func (v *PlaceholderValue) Clone(_ ValueCloneContext) Value {
	return v
}

func (*PlaceholderValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}
