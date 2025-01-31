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
	"github.com/onflow/cadence/sema"
)

// EphemeralReferenceValue

type EphemeralReferenceValue struct {
	Value Value
	// BorrowedType is the T in &T
	BorrowedType  sema.Type
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
	interpreter *Interpreter,
	authorization Authorization,
	value Value,
	borrowedType sema.Type,
	locationRange LocationRange,
) *EphemeralReferenceValue {
	if reference, isReference := value.(ReferenceValue); isReference {
		panic(NestedReferenceError{
			Value:         reference,
			LocationRange: locationRange,
		})
	}

	ref := &EphemeralReferenceValue{
		Authorization: authorization,
		Value:         value,
		BorrowedType:  borrowedType,
	}

	interpreter.maybeTrackReferencedResourceKindedValue(ref)

	return ref
}

func NewEphemeralReferenceValue(
	interpreter *Interpreter,
	authorization Authorization,
	value Value,
	borrowedType sema.Type,
	locationRange LocationRange,
) *EphemeralReferenceValue {
	common.UseMemory(interpreter, common.EphemeralReferenceValueMemoryUsage)
	return NewUnmeteredEphemeralReferenceValue(interpreter, authorization, value, borrowedType, locationRange)
}

func (*EphemeralReferenceValue) isValue() {}

func (v *EphemeralReferenceValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitEphemeralReferenceValue(interpreter, v)
}

func (*EphemeralReferenceValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (v *EphemeralReferenceValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *EphemeralReferenceValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences, EmptyLocationRange)
}

func (v *EphemeralReferenceValue) MeteredString(interpreter *Interpreter, seenReferences SeenReferences, locationRange LocationRange) string {
	if _, ok := seenReferences[v]; ok {
		common.UseMemory(interpreter, common.SeenReferenceStringMemoryUsage)
		return "..."
	}

	seenReferences[v] = struct{}{}
	defer delete(seenReferences, v)

	return v.Value.MeteredString(interpreter, seenReferences, locationRange)
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

func (*EphemeralReferenceValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return false
}

func (v *EphemeralReferenceValue) ReferencedValue(
	_ *Interpreter,
	_ LocationRange,
	_ bool,
) *Value {
	return &v.Value
}

func (v *EphemeralReferenceValue) GetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	return interpreter.getMember(v.Value, locationRange, name)
}

func (v *EphemeralReferenceValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	identifier string,
) Value {
	if memberAccessibleValue, ok := v.Value.(MemberAccessibleValue); ok {
		return memberAccessibleValue.RemoveMember(interpreter, locationRange, identifier)
	}

	return nil
}

func (v *EphemeralReferenceValue) SetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
	value Value,
) bool {
	return interpreter.setMember(v.Value, locationRange, name, value)
}

func (v *EphemeralReferenceValue) GetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	return v.Value.(ValueIndexableValue).
		GetKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) SetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	v.Value.(ValueIndexableValue).
		SetKey(interpreter, locationRange, key, value)
}

func (v *EphemeralReferenceValue) InsertKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	v.Value.(ValueIndexableValue).
		InsertKey(interpreter, locationRange, key, value)
}

func (v *EphemeralReferenceValue) RemoveKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	return v.Value.(ValueIndexableValue).
		RemoveKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) GetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
) Value {
	self := v.Value

	if selfComposite, isComposite := self.(*CompositeValue); isComposite {
		return selfComposite.getTypeKey(
			interpreter,
			locationRange,
			key,
			interpreter.MustConvertStaticAuthorizationToSemaAccess(v.Authorization),
		)
	}

	return self.(TypeIndexableValue).
		GetTypeKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) SetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
	value Value,
) {
	v.Value.(TypeIndexableValue).
		SetTypeKey(interpreter, locationRange, key, value)
}

func (v *EphemeralReferenceValue) RemoveTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
) Value {
	return v.Value.(TypeIndexableValue).
		RemoveTypeKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
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
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	self := v.Value

	staticType := v.Value.StaticType(interpreter)

	if !interpreter.IsSubTypeOfSemaType(staticType, v.BorrowedType) {
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

	result := self.ConformsToStaticType(
		interpreter,
		locationRange,
		results,
	)

	results[entry] = result

	return result
}

func (*EphemeralReferenceValue) IsStorable() bool {
	return false
}

func (v *EphemeralReferenceValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return NonStorable{Value: v}, nil
}

func (*EphemeralReferenceValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*EphemeralReferenceValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v *EphemeralReferenceValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *EphemeralReferenceValue) Clone(inter *Interpreter) Value {
	return NewUnmeteredEphemeralReferenceValue(inter, v.Authorization, v.Value, v.BorrowedType, EmptyLocationRange)
}

func (*EphemeralReferenceValue) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (*EphemeralReferenceValue) isReference() {}

func (v *EphemeralReferenceValue) ForEach(
	interpreter *Interpreter,
	elementType sema.Type,
	function func(value Value) (resume bool),
	_ bool,
	locationRange LocationRange,
) {
	forEachReference(
		interpreter,
		v,
		v.Value,
		elementType,
		function,
		locationRange,
	)
}

func (v *EphemeralReferenceValue) BorrowType() sema.Type {
	return v.BorrowedType
}
