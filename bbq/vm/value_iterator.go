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

func (v IteratorWrapperValue) Storable(storage atree.SlabStorage, address atree.Address, u uint64) (atree.Storable, error) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) String() string {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Accept(interpreter *interpreter.Interpreter, visitor interpreter.Visitor, locationRange interpreter.LocationRange) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Walk(interpreter interpreter.ValueWalkContext, walkChild func(interpreter.Value), locationRange interpreter.LocationRange) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) StaticType(context interpreter.ValueStaticTypeContext) interpreter.StaticType {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) ConformsToStaticType(interpreter *interpreter.Interpreter, locationRange interpreter.LocationRange, results interpreter.TypeConformanceResults) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) RecursiveString(seenReferences interpreter.SeenReferences) string {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) MeteredString(context interpreter.ValueStringContext, seenReferences interpreter.SeenReferences, locationRange interpreter.LocationRange) string {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) IsResourceKinded(context interpreter.ValueStaticTypeContext) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) NeedsStoreTo(address atree.Address) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Transfer(
	_ interpreter.ValueTransferContext,
	_ interpreter.LocationRange,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v IteratorWrapperValue) DeepRemove(removeContext interpreter.ValueRemoveContext, hasNoParentContainer bool) {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) Clone(interpreter *interpreter.Interpreter) interpreter.Value {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v IteratorWrapperValue) IsImportable(interpreter *interpreter.Interpreter, locationRange interpreter.LocationRange) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}
