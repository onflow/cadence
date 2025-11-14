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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// EphemeralReferenceValue

type EphemeralReferenceValue struct {
	Value Value
	// BorrowedType is the T in &T
	BorrowedType  StaticType
	Authorization Authorization
}

var _ Value = &EphemeralReferenceValue{}
var _ EquatableValue = &EphemeralReferenceValue{}
var _ ValueIndexableValue = &EphemeralReferenceValue{}
var _ TypeIndexableValue = &EphemeralReferenceValue{}
var _ MemberAccessibleValue = &EphemeralReferenceValue{}
var _ AuthorizedValue = &EphemeralReferenceValue{}
var _ ReferenceValue = &EphemeralReferenceValue{}
var _ IterableValue = &EphemeralReferenceValue{}

func NewUnmeteredEphemeralReferenceValue(
	referenceTracker ReferenceTracker,
	authorization Authorization,
	value Value,
	borrowedType StaticType,
) *EphemeralReferenceValue {
	if reference, isReference := value.(ReferenceValue); isReference {
		panic(&NestedReferenceError{
			Value: reference,
		})
	}

	ref := &EphemeralReferenceValue{
		Authorization: authorization,
		Value:         value,
		BorrowedType:  borrowedType,
	}

	referenceTracker.MaybeTrackReferencedResourceKindedValue(ref)

	return ref
}

func NewEphemeralReferenceValue(
	context ReferenceCreationContext,
	authorization Authorization,
	value Value,
	borrowedType StaticType,
) *EphemeralReferenceValue {
	common.UseMemory(context, common.EphemeralReferenceValueMemoryUsage)
	return NewUnmeteredEphemeralReferenceValue(context, authorization, value, borrowedType)
}

func (*EphemeralReferenceValue) IsValue() {}

func (v *EphemeralReferenceValue) Accept(context ValueVisitContext, visitor Visitor) {
	visitor.VisitEphemeralReferenceValue(context, v)
}

func (*EphemeralReferenceValue) Walk(_ ValueWalkContext, _ func(Value)) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (v *EphemeralReferenceValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *EphemeralReferenceValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(NoOpStringContext{}, seenReferences)
}

func (v *EphemeralReferenceValue) MeteredString(
	context ValueStringContext,
	seenReferences SeenReferences,
) string {
	if _, ok := seenReferences[v]; ok {
		common.UseMemory(context, common.SeenReferenceStringMemoryUsage)
		return "..."
	}

	seenReferences[v] = struct{}{}
	defer delete(seenReferences, v)

	return v.Value.MeteredString(context, seenReferences)
}

func (v *EphemeralReferenceValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewReferenceStaticType(
		context,
		v.Authorization,
		v.Value.StaticType(context),
	)
}

func (v *EphemeralReferenceValue) GetAuthorization() Authorization {
	return v.Authorization
}

func (*EphemeralReferenceValue) IsImportable(_ ValueImportableContext) bool {
	return false
}

func (v *EphemeralReferenceValue) ReferencedValue(_ ValueStaticTypeContext, _ bool) *Value {
	return &v.Value
}

func (v *EphemeralReferenceValue) GetMember(context MemberAccessibleContext, name string) Value {
	var result Value

	if memberAccessibleValue, ok := v.Value.(MemberAccessibleValue); ok {
		result = memberAccessibleValue.GetMember(context, name)
	}

	if result == nil {
		// NOTE: Must call the `GetMethod` of the `EphemeralReferenceValue`, not of the referenced-value.
		result = context.GetMethod(v, name)
	}

	return result
}

func (v *EphemeralReferenceValue) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	return getBuiltinFunctionMember(context, v.Value, name)
}

func (v *EphemeralReferenceValue) RemoveMember(context ValueTransferContext, name string) Value {
	if memberAccessibleValue, ok := v.Value.(MemberAccessibleValue); ok {
		return memberAccessibleValue.RemoveMember(context, name)
	}

	return nil
}

func (v *EphemeralReferenceValue) SetMember(context ValueTransferContext, name string, value Value) bool {
	return setMember(context, v.Value, name, value)
}

func (v *EphemeralReferenceValue) GetKey(context ContainerReadContext, key Value) Value {
	return v.Value.(ValueIndexableValue).
		GetKey(context, key)
}

func (v *EphemeralReferenceValue) SetKey(context ContainerMutationContext, key Value, value Value) {
	v.Value.(ValueIndexableValue).
		SetKey(context, key, value)
}

func (v *EphemeralReferenceValue) InsertKey(context ContainerMutationContext, key Value, value Value) {
	v.Value.(ValueIndexableValue).
		InsertKey(context, key, value)
}

func (v *EphemeralReferenceValue) RemoveKey(context ContainerMutationContext, key Value) Value {
	return v.Value.(ValueIndexableValue).
		RemoveKey(context, key)
}

func (v *EphemeralReferenceValue) GetTypeKey(context MemberAccessibleContext, key sema.Type) Value {
	self := v.Value

	if selfComposite, isComposite := self.(*CompositeValue); isComposite {
		semaAccess, err := context.SemaAccessFromStaticAuthorization(v.Authorization)
		if err != nil {
			panic(err)
		}

		return selfComposite.getTypeKey(
			context,
			key,
			semaAccess,
		)
	}

	return self.(TypeIndexableValue).
		GetTypeKey(context, key)
}

func (v *EphemeralReferenceValue) SetTypeKey(context ValueTransferContext, key sema.Type, value Value) {
	v.Value.(TypeIndexableValue).
		SetTypeKey(context, key, value)
}

func (v *EphemeralReferenceValue) RemoveTypeKey(context ValueTransferContext, key sema.Type) Value {
	return v.Value.(TypeIndexableValue).
		RemoveTypeKey(context, key)
}

func (v *EphemeralReferenceValue) Equal(_ ValueComparisonContext, other Value) bool {
	otherReference, ok := other.(*EphemeralReferenceValue)
	if !ok ||
		v.Value != otherReference.Value ||
		!v.Authorization.Equal(otherReference.Authorization) {

		return false
	}

	if v.BorrowedType == nil {
		return otherReference.BorrowedType == nil
	} else {
		return v.BorrowedType.Equal(otherReference.BorrowedType)
	}
}

func (v *EphemeralReferenceValue) ConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	results TypeConformanceResults,
) bool {
	self := v.Value

	staticType := v.Value.StaticType(context)

	if !IsSubType(context, staticType, v.BorrowedType) {
		return false
	}

	entry := typeConformanceResultEntry{
		EphemeralReferenceValue: v,
		EphemeralReferenceType:  staticType,
	}

	if result, contains := results[entry]; contains {
		return result
	}

	// It is safe to set 'true' here even this is not checked yet, because the final result
	// doesn't depend on this. It depends on the rest of values of the object tree.
	results[entry] = true

	result := self.ConformsToStaticType(context, results)

	results[entry] = result

	return result
}

func (*EphemeralReferenceValue) IsStorable() bool {
	return false
}

func (v *EphemeralReferenceValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint32) (atree.Storable, error) {
	return NonStorable{Value: v}, nil
}

func (*EphemeralReferenceValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*EphemeralReferenceValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v *EphemeralReferenceValue) Transfer(
	context ValueTransferContext,
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

func (v *EphemeralReferenceValue) Clone(context ValueCloneContext) Value {
	return NewUnmeteredEphemeralReferenceValue(
		context,
		v.Authorization,
		v.Value,
		v.BorrowedType,
	)
}

func (*EphemeralReferenceValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (*EphemeralReferenceValue) isReference() {}

func (v *EphemeralReferenceValue) ForEach(
	context IterableValueForeachContext,
	elementType sema.Type,
	function func(value Value) (resume bool),
	_ bool,
) {
	forEachReference(
		context,
		v,
		v.Value,
		elementType,
		function,
	)
}

func (v *EphemeralReferenceValue) BorrowType() StaticType {
	return v.BorrowedType
}

func (v *EphemeralReferenceValue) Iterator(context ValueStaticTypeContext) ValueIterator {
	referencedIterable, ok := v.Value.(IterableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return &ReferenceValueIterator{
		reference: v,
		iterator:  referencedIterable.Iterator(context),
	}
}

type ReferenceValueIterator struct {
	reference Value
	iterator  ValueIterator
}

var _ ValueIterator = &ReferenceValueIterator{}

func (i *ReferenceValueIterator) Next(context ValueIteratorContext) Value {
	// Iterator implicitly captures the reference.
	// Therefore, check whether the reference is valid, everytime the iterator is used.
	CheckInvalidatedResourceOrResourceReference(i.reference, context)
	return i.iterator.Next(context)
}

func (i *ReferenceValueIterator) HasNext(context ValueIteratorContext) bool {
	return i.iterator.HasNext(context)
}

func (i *ReferenceValueIterator) ValueID() (atree.ValueID, bool) {
	return i.iterator.ValueID()
}
