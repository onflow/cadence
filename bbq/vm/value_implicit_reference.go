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

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
)

type ImplicitReferenceValue struct {
	value   Value
	selfRef interpreter.ReferenceValue
}

var _ Value = ImplicitReferenceValue{}

var implicitReferenceMemoryUsage = common.NewConstantMemoryUsage(common.MemoryKindImplicitReferenceVMValue)

func NewImplicitReferenceValue(context interpreter.ReferenceCreationContext, value Value) ImplicitReferenceValue {
	common.UseMemory(context, implicitReferenceMemoryUsage)

	semaType := interpreter.MustSemaTypeOfValue(value, context)

	// Create an explicit reference to represent the implicit reference behavior of 'self' value.
	// Authorization doesn't matter, we just need a reference to add to tracking.
	selfRef := interpreter.NewEphemeralReferenceValue(
		context,
		interpreter.UnauthorizedAccess,
		value,
		semaType,
	)

	return ImplicitReferenceValue{
		value:   value,
		selfRef: selfRef,
	}
}

func (v ImplicitReferenceValue) IsValue() {}

func (v ImplicitReferenceValue) ReferencedValue(
	context interpreter.ValueStaticTypeContext,
) interpreter.Value {
	interpreter.CheckInvalidatedResourceOrResourceReference(v.selfRef, context)
	return v.value
}

func (v ImplicitReferenceValue) GetAuthorization() interpreter.Authorization {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) String() string {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) Accept(_ interpreter.ValueVisitContext, _ interpreter.Visitor) {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) Walk(_ interpreter.ValueWalkContext, _ func(interpreter.Value)) {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) StaticType(_ interpreter.ValueStaticTypeContext) interpreter.StaticType {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) ConformsToStaticType(
	_ interpreter.ValueStaticTypeConformanceContext,
	_ interpreter.TypeConformanceResults,
) bool {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) RecursiveString(_ interpreter.SeenReferences) string {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) MeteredString(
	_ interpreter.ValueStringContext,
	_ interpreter.SeenReferences,
) string {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) IsResourceKinded(_ interpreter.ValueStaticTypeContext) bool {
	return false
}

func (v ImplicitReferenceValue) NeedsStoreTo(_ atree.Address) bool {
	// Iterator is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) Transfer(
	_ interpreter.ValueTransferContext,
	_ atree.Address,
	_ bool,
	_ atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) interpreter.Value {
	return v
}

func (v ImplicitReferenceValue) DeepRemove(_ interpreter.ValueRemoveContext, _ bool) {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) Clone(_ interpreter.ValueCloneContext) interpreter.Value {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}

func (v ImplicitReferenceValue) IsImportable(_ interpreter.ValueImportableContext) bool {
	// ImplicitReferenceValue is an internal-only value.
	// Hence, this should never be called.
	panic(errors.NewUnreachableError())
}
