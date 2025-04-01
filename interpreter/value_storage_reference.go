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
	BorrowedType         sema.Type
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
	borrowedType sema.Type,
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
	borrowedType sema.Type,
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

func (v *StorageReferenceValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitStorageReferenceValue(interpreter, v)
}

func (*StorageReferenceValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (*StorageReferenceValue) String() string {
	return format.StorageReference
}

func (v *StorageReferenceValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *StorageReferenceValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(context, common.StorageReferenceValueStringMemoryUsage)
	return v.String()
}

func (v *StorageReferenceValue) StaticType(context ValueStaticTypeContext) StaticType {
	referencedValue, err := v.dereference(context, EmptyLocationRange)
	if err != nil {
		panic(err)
	}

	self := *referencedValue

	return NewReferenceStaticType(
		context,
		v.Authorization,
		self.StaticType(context),
	)
}

func (v *StorageReferenceValue) GetAuthorization() Authorization {
	return v.Authorization
}

func (*StorageReferenceValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (v *StorageReferenceValue) dereference(context ValueStaticTypeContext, locationRange LocationRange) (*Value, error) {
	address := v.TargetStorageAddress
	domain := v.TargetPath.Domain.StorageDomain()
	identifier := v.TargetPath.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	referenced := context.ReadStored(address, domain, storageMapKey)
	if referenced == nil {
		return nil, nil
	}

	if reference, isReference := referenced.(ReferenceValue); isReference {
		panic(NestedReferenceError{
			Value:         reference,
			LocationRange: locationRange,
		})
	}

	if v.BorrowedType != nil {
		staticType := referenced.StaticType(context)

		if !IsSubTypeOfSemaType(context, staticType, v.BorrowedType) {
			semaType := MustConvertStaticToSemaType(staticType, context)

			return nil, ForceCastTypeMismatchError{
				ExpectedType:  v.BorrowedType,
				ActualType:    semaType,
				LocationRange: locationRange,
			}
		}
	}

	return &referenced, nil
}

func (v *StorageReferenceValue) ReferencedValue(
	context ValueStaticTypeContext,
	locationRange LocationRange,
	errorOnFailedDereference bool,
) *Value {
	referencedValue, err := v.dereference(context, locationRange)
	if err == nil {
		return referencedValue
	}
	if forceCastErr, ok := err.(ForceCastTypeMismatchError); ok {
		if errorOnFailedDereference {
			// relay the type mismatch error with a dereference error context
			panic(DereferenceError{
				ExpectedType:  forceCastErr.ExpectedType,
				ActualType:    forceCastErr.ActualType,
				LocationRange: locationRange,
			})
		}
		return nil
	}
	panic(err)
}

func (v *StorageReferenceValue) mustReferencedValue(
	context ValueStaticTypeContext,
	locationRange LocationRange,
) Value {
	referencedValue := v.ReferencedValue(context, locationRange, true)
	if referencedValue == nil {
		panic(DereferenceError{
			Cause:         "no value is stored at this path",
			LocationRange: locationRange,
		})
	}

	return *referencedValue
}

func (v *StorageReferenceValue) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	referencedValue := v.mustReferencedValue(context, locationRange)

	member := getMember(
		context,
		referencedValue,
		locationRange,
		name,
	)

	// If the member is a function, it is always a bound-function.
	// By default, bound functions create and hold an ephemeral reference (`SelfReference`).
	// For storage references, replace this default one with the actual storage reference.
	// It is not possible (or a lot of work), to create the bound function with the storage reference
	// when it was created originally, because `getMember(referencedValue, ...)` doesn't know
	// whether the member was accessed directly, or via a reference.
	if boundFunction, isBoundFunction := member.(BoundFunctionValue); isBoundFunction {
		boundFunction.SelfReference = v
		return boundFunction
	}

	return member
}

func (v *StorageReferenceValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

	return self.(MemberAccessibleValue).RemoveMember(interpreter, locationRange, name)
}

func (v *StorageReferenceValue) SetMember(context MemberAccessibleContext, locationRange LocationRange, name string, value Value) bool {
	self := v.mustReferencedValue(context, locationRange)

	return setMember(
		context,
		self,
		locationRange,
		name,
		value,
	)
}

func (v *StorageReferenceValue) GetKey(context ValueComparisonContext, locationRange LocationRange, key Value) Value {
	self := v.mustReferencedValue(context, locationRange)

	return self.(ValueIndexableValue).
		GetKey(context, locationRange, key)
}

func (v *StorageReferenceValue) SetKey(context ContainerMutationContext, locationRange LocationRange, key Value, value Value) {
	self := v.mustReferencedValue(context, locationRange)

	self.(ValueIndexableValue).
		SetKey(context, locationRange, key, value)
}

func (v *StorageReferenceValue) InsertKey(context ContainerMutationContext, locationRange LocationRange, key Value, value Value) {
	self := v.mustReferencedValue(context, locationRange)

	self.(ValueIndexableValue).
		InsertKey(context, locationRange, key, value)
}

func (v *StorageReferenceValue) RemoveKey(context ContainerMutationContext, locationRange LocationRange, key Value) Value {
	self := v.mustReferencedValue(context, locationRange)

	return self.(ValueIndexableValue).
		RemoveKey(context, locationRange, key)
}

func (v *StorageReferenceValue) GetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

	if selfComposite, isComposite := self.(*CompositeValue); isComposite {
		return selfComposite.getTypeKey(
			interpreter,
			locationRange,
			key,
			MustConvertStaticAuthorizationToSemaAccess(interpreter, v.Authorization),
		)
	}

	return self.(TypeIndexableValue).
		GetTypeKey(interpreter, locationRange, key)
}

func (v *StorageReferenceValue) SetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
	value Value,
) {
	self := v.mustReferencedValue(interpreter, locationRange)

	self.(TypeIndexableValue).
		SetTypeKey(interpreter, locationRange, key, value)
}

func (v *StorageReferenceValue) RemoveTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

	return self.(TypeIndexableValue).
		RemoveTypeKey(interpreter, locationRange, key)
}

func (v *StorageReferenceValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
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
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	referencedValue, err := v.dereference(interpreter, locationRange)
	if referencedValue == nil || err != nil {
		return false
	}

	self := *referencedValue

	staticType := self.StaticType(interpreter)

	if !IsSubTypeOfSemaType(interpreter, staticType, v.BorrowedType) {
		return false
	}

	return self.ConformsToStaticType(
		interpreter,
		locationRange,
		results,
	)
}

func (*StorageReferenceValue) IsStorable() bool {
	return false
}

func (v *StorageReferenceValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
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

func (v *StorageReferenceValue) Clone(_ *Interpreter) Value {
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
	interpreter *Interpreter,
	elementType sema.Type,
	function func(value Value) (resume bool),
	_ bool,
	locationRange LocationRange,
) {
	referencedValue := v.mustReferencedValue(interpreter, locationRange)
	forEachReference(
		interpreter,
		v,
		referencedValue,
		elementType,
		function,
		locationRange,
	)
}

func forEachReference(
	interpreter *Interpreter,
	reference ReferenceValue,
	referencedValue Value,
	elementType sema.Type,
	function func(value Value) (resume bool),
	locationRange LocationRange,
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
		checkInvalidatedResourceOrResourceReference(reference, locationRange, interpreter)

		if isResultReference {
			value = interpreter.getReferenceValue(value, elementType, locationRange)
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
		interpreter,
		referencedElementType,
		updatedFunction,
		transferElements,
		locationRange,
	)
}

func (v *StorageReferenceValue) BorrowType() sema.Type {
	return v.BorrowedType
}

func (v *StorageReferenceValue) Iterator(context ValueStaticTypeContext, locationRange LocationRange) ValueIterator {
	//TODO implement me
	panic("implement me")
}
