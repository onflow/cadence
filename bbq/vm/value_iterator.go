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

package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type IteratorWrapperValue struct {
	interpreter.ValueIterator
}

var _ Value = &IteratorWrapperValue{}

func NewIteratorWrapperValue(iterator interpreter.ValueIterator) *IteratorWrapperValue {
	return &IteratorWrapperValue{
		ValueIterator: iterator,
	}
}

func (v IteratorWrapperValue) IsValue() {}

func (v IteratorWrapperValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint32) (atree.Storable, error) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) String() string {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Accept(_ interpreter.ValueVisitContext, _ interpreter.Visitor) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Walk(_ interpreter.ValueWalkContext, _ func(interpreter.Value)) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) StaticType(_ interpreter.ValueStaticTypeContext) interpreter.StaticType {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.TypeConformanceResults,
) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) RecursiveString(_ interpreter.SeenReferences) string {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) MeteredString(
	_ interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
) string {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v IteratorWrapperValue) NeedsStoreTo(_ atree.Address) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Transfer(
	_ interpreter.ValueTransferContext,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v IteratorWrapperValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) IsImportable(_ interpreter.ValueImportableContext) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}
