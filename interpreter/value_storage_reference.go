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
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
)

// StorageReferenceValue
type StorageReferenceValue struct {
	BorrowedType         StaticType
	TargetPath           PathValue
	TargetStorageAddress common.Address
	Authorization        Authorization
}

var _ Value = &StorageReferenceValue{}
var _ EquatableValue = &StorageReferenceValue{}
var _ ValueIndexableValue = &StorageReferenceValue{}
var _ TypeIndexableValue = &StorageReferenceValue{}
var _ MemberAccessibleValue = &StorageReferenceValue{}
var _ AuthorizedValue = &StorageReferenceValue{}
var _ ReferenceValue = &StorageReferenceValue{}
var _ IterableValue = &StorageReferenceValue{}

func NewUnmeteredStorageReferenceValue(
	authorization Authorization,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType StaticType,
) *StorageReferenceValue {
	return &StorageReferenceValue{
		Authorization:        authorization,
		TargetStorageAddress: targetStorageAddress,
		TargetPath:           targetPath,
		BorrowedType:         borrowedType,
	}
}

func NewStorageReferenceValue(
	memoryGauge common.MemoryGauge,
	authorization Authorization,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType StaticType,
) *StorageReferenceValue {
	common.UseMemory(memoryGauge, common.StorageReferenceValueMemoryUsage)
	return NewUnmeteredStorageReferenceValue(
		authorization,
		targetStorageAddress,
		targetPath,
		borrowedType,
	)
}

func (*StorageReferenceValue) IsValue() {}

func (v *StorageReferenceValue) Accept(context ValueVisitContext, visitor Visitor) {
	visitor.VisitStorageReferenceValue(context, v)
}

func (*StorageReferenceValue) Walk(_ ValueWalkContext, _ func(Value)) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (*StorageReferenceValue) String() string {
	return format.StorageReference
}

func (v *StorageReferenceValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *StorageReferenceValue) MeteredString(
	context ValueStringContext,
	_ SeenReferences,
) string {
	common.UseMemory(context, common.StorageReferenceValueStringMemoryUsage)
	return v.String()
}

func (v *StorageReferenceValue) StaticType(context ValueStaticTypeContext) StaticType {
	self := v.mustReferencedValue(context)

	return NewReferenceStaticType(
		context,
		v.Authorization,
		self.StaticType(context),
	)
}

func (v *StorageReferenceValue) GetAuthorization() Authorization {
	return v.Authorization
}

func (*StorageReferenceValue) IsImportable(_ ValueImportableContext) bool {
	return false
}

func (v *StorageReferenceValue) dereference(context ValueStaticTypeContext) (*Value, error) {
	address := v.TargetStorageAddress
	domain := v.TargetPath.Domain.StorageDomain()
	identifier := v.TargetPath.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	referenced := context.ReadStored(address, domain, storageMapKey)
	if referenced == nil {
		return nil, nil
	}

	if reference, isReference := referenced.(ReferenceValue); isReference {
		panic(&NestedReferenceError{
			Value: reference,
		})
	}

	if v.BorrowedType != nil {
		staticType := referenced.StaticType(context)

		if !IsSubType(context, staticType, v.BorrowedType) {
			semaType := context.SemaTypeFromStaticType(staticType)

			return nil, &StoredValueTypeMismatchError{
				ExpectedType: context.SemaTypeFromStaticType(v.BorrowedType),
				ActualType:   semaType,
			}
		}
	}

	return &referenced, nil
}

func (v *StorageReferenceValue) ReferencedValue(context ValueStaticTypeContext, errorOnFailedDereference bool) *Value {
	referencedValue, err := v.dereference(context)
	if err == nil {
		return referencedValue
	}
	if forceCastErr, ok := err.(*StoredValueTypeMismatchError); ok {
		if errorOnFailedDereference {
			// relay the type mismatch error with a dereference error context
			panic(&DereferenceError{
				ExpectedType: forceCastErr.ExpectedType,
				ActualType:   forceCastErr.ActualType,
			})
		}
		return nil
	}
	panic(err)
}

func (v *StorageReferenceValue) mustReferencedValue(
	context ValueStaticTypeContext,
) Value {
	referencedValue := v.ReferencedValue(context, true)
	if referencedValue == nil {
		panic(&DereferenceError{
			Cause: "no value is stored at this path",
		})
	}

	return *referencedValue
}

func (v *StorageReferenceValue) GetMember(context MemberAccessibleContext, name string) Value {
	referencedValue := v.mustReferencedValue(context)

	var member Value

	if memberAccessibleValue, ok := referencedValue.(MemberAccessibleValue); ok {
		member = memberAccessibleValue.GetMember(context, name)
	}

	if member == nil {
		// NOTE: Must call the `GetMethod` of the `StorageReferenceValue`, not of the referenced-value.
		member = context.GetMethod(v, name)
	}

	// If the member is a function, it is always a bound-function.
	// By default, bound functions create and hold an ephemeral reference
	// (in `BoundFunctionValue.SelfReference`).
	// For storage references, replace this default one with a storage reference.
	//
	// However, we cannot use the storage reference as-is:
	// Because we look up the member on the referenced value,
	// we also must use its type as the borrowed type for the `SelfReference` type,
	// because during invocation the bound function can only be invoked
	// if the type of the dereferenced value at that time still matches
	// the type of the dereferenced value at the time of binding (here).
	//
	// For example, imagine storing a value of type T (e.g. `String`),
	// creating a reference with a supertype (e.g. `AnyStruct`),
	// and then creating a bound function on it.
	// Then, if we change the storage location to store a value of unrelated type U instead (e.g. `Int`),
	// and invoke the bound function, the bound function is potentially invalid.
	//
	// It is not possible (or a lot of work), to create the bound function with the storage reference
	// when it was created originally, because `getMember(referencedValue, ...)` doesn't know
	// whether the member was accessed directly, or via a reference.
	return context.MaybeUpdateStorageReferenceMemberReceiver(
		v,
		referencedValue,
		member,
	)
}

func (v *StorageReferenceValue) GetMethod(context MemberAccessibleContext, name string) FunctionValue {
	referencedValue := v.mustReferencedValue(context)
	return getBuiltinFunctionMember(context, referencedValue, name)
}

func (v *StorageReferenceValue) RemoveMember(context ValueTransferContext, name string) Value {
	self := v.mustReferencedValue(context)

	return self.(MemberAccessibleValue).RemoveMember(context, name)
}

func (v *StorageReferenceValue) SetMember(context ValueTransferContext, name string, value Value) bool {
	self := v.mustReferencedValue(context)

	return setMember(
		context,
		self,
		name,
		value,
	)
}

func (v *StorageReferenceValue) GetKey(context ContainerReadContext, key Value) Value {
	self := v.mustReferencedValue(context)

	return self.(ValueIndexableValue).
		GetKey(context, key)
}

func (v *StorageReferenceValue) SetKey(context ContainerMutationContext, key Value, value Value) {
	self := v.mustReferencedValue(context)

	self.(ValueIndexableValue).
		SetKey(context, key, value)
}

func (v *StorageReferenceValue) InsertKey(context ContainerMutationContext, key Value, value Value) {
	self := v.mustReferencedValue(context)

	self.(ValueIndexableValue).
		InsertKey(context, key, value)
}

func (v *StorageReferenceValue) RemoveKey(context ContainerMutationContext, key Value) Value {
	self := v.mustReferencedValue(context)

	return self.(ValueIndexableValue).
		RemoveKey(context, key)
}

func (v *StorageReferenceValue) GetTypeKey(
	context MemberAccessibleContext,
	key sema.Type,
) Value {
	self := v.mustReferencedValue(context)

	if selfComposite, isComposite := self.(*CompositeValue); isComposite {
		access, err := context.SemaAccessFromStaticAuthorization(v.Authorization)
		if err != nil {
			panic(err)
		}
		return selfComposite.getTypeKey(
			context,
			key,
			access,
		)
	}

	return self.(TypeIndexableValue).
		GetTypeKey(context, key)
}

func (v *StorageReferenceValue) SetTypeKey(
	context ValueTransferContext,
	key sema.Type,
	value Value,
) {
	self := v.mustReferencedValue(context)

	self.(TypeIndexableValue).
		SetTypeKey(context, key, value)
}

func (v *StorageReferenceValue) RemoveTypeKey(context ValueTransferContext, key sema.Type) Value {
	self := v.mustReferencedValue(context)

	return self.(TypeIndexableValue).
		RemoveTypeKey(context, key)
}

func (v *StorageReferenceValue) Equal(_ ValueComparisonContext, other Value) bool {
	otherReference, ok := other.(*StorageReferenceValue)
	if !ok ||
		v.TargetStorageAddress != otherReference.TargetStorageAddress ||
		v.TargetPath != otherReference.TargetPath ||
		!v.Authorization.Equal(otherReference.Authorization) {

		return false
	}

	if v.BorrowedType == nil {
		return otherReference.BorrowedType == nil
	} else {
		return v.BorrowedType.Equal(otherReference.BorrowedType)
	}
}

func (v *StorageReferenceValue) ConformsToStaticType(
	context ValueStaticTypeConformanceContext,
	results TypeConformanceResults,
) bool {
	referencedValue, err := v.dereference(context)
	if referencedValue == nil || err != nil {
		return false
	}

	self := *referencedValue

	staticType := self.StaticType(context)

	if !IsSubType(context, staticType, v.BorrowedType) {
		return false
	}

	return self.ConformsToStaticType(context, results)
}

func (*StorageReferenceValue) IsStorable() bool {
	return false
}

func (v *StorageReferenceValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint32) (atree.Storable, error) {
	return NonStorable{Value: v}, nil
}

func (*StorageReferenceValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*StorageReferenceValue) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v *StorageReferenceValue) Transfer(
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

func (v *StorageReferenceValue) Clone(_ ValueCloneContext) Value {
	return NewUnmeteredStorageReferenceValue(
		v.Authorization,
		v.TargetStorageAddress,
		v.TargetPath,
		v.BorrowedType,
	)
}

func (*StorageReferenceValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (*StorageReferenceValue) isReference() {}

func (v *StorageReferenceValue) ForEach(
	context IterableValueForeachContext,
	elementType sema.Type,
	function func(value Value) (resume bool),
	_ bool,
) {
	referencedValue := v.mustReferencedValue(context)
	forEachReference(
		context,
		v,
		referencedValue,
		elementType,
		function,
	)
}

func forEachReference(
	context IterableValueForeachContext,
	reference ReferenceValue,
	referencedValue Value,
	elementType sema.Type,
	function func(value Value) (resume bool),
) {
	referencedIterable, ok := referencedValue.(IterableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	referenceType, isResultReference := sema.MaybeReferenceType(elementType)

	updatedFunction := func(value Value) (resume bool) {
		// The loop dereference the reference once, and hold onto that referenced-value.
		// But the reference could get invalidated during the iteration, making that referenced-value invalid.
		// So check the validity of the reference, before each iteration.
		CheckInvalidatedResourceOrResourceReference(reference, context)

		if isResultReference {
			value = getReferenceValue(
				context,
				value,
				elementType,
			)
		}

		return function(value)
	}

	referencedElementType := elementType
	if isResultReference {
		referencedElementType = referenceType.Type
	}

	// Do not transfer the inner referenced elements.
	// We only take a references to them, but never move them out.
	const transferElements = false

	referencedIterable.ForEach(
		context,
		referencedElementType,
		updatedFunction,
		transferElements,
	)
}

func (v *StorageReferenceValue) BorrowType() StaticType {
	return v.BorrowedType
}

func (v *StorageReferenceValue) Iterator(context ValueStaticTypeContext) ValueIterator {
	referencedValue := v.mustReferencedValue(context)
	referencedIterable, ok := referencedValue.(IterableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return &ReferenceValueIterator{
		iterator: referencedIterable.Iterator(context),
	}
}
