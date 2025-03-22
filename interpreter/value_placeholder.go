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

// placeholderValue
type placeholderValue struct{}

var placeholder Value = placeholderValue{}

var _ Value = placeholderValue{}

func (placeholderValue) isValue() {}

func (f placeholderValue) String() string {
	return f.RecursiveString(SeenReferences{})
}

func (f placeholderValue) RecursiveString(_ SeenReferences) string {
	return ""
}

func (f placeholderValue) MeteredString(_ *Interpreter, _ SeenReferences, _ LocationRange) string {
	return ""
}

func (f placeholderValue) Accept(_ *Interpreter, _ Visitor, _ LocationRange) {
	// NO-OP
}

func (f placeholderValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (f placeholderValue) StaticType(_ ValueStaticTypeContext) StaticType {
	return PrimitiveStaticTypeNever
}

func (placeholderValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (f placeholderValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (f placeholderValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: f}, nil
}

func (placeholderValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (placeholderValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (f placeholderValue) Transfer(
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
		context.RemoveReferencedSlab(storable)
	}
	return f
}

func (f placeholderValue) Clone(_ *Interpreter) Value {
	return f
}

func (placeholderValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}
