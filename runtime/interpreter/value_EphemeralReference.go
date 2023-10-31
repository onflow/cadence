package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// EphemeralReferenceValue

type EphemeralReferenceValue struct {
	Value Value
	// BorrowedType is the T in &T
	BorrowedType sema.Type
	Authorized   bool
}

var _ Value = &EphemeralReferenceValue{}
var _ EquatableValue = &EphemeralReferenceValue{}
var _ ValueIndexableValue = &EphemeralReferenceValue{}
var _ TypeIndexableValue = &EphemeralReferenceValue{}
var _ MemberAccessibleValue = &EphemeralReferenceValue{}
var _ ReferenceValue = &EphemeralReferenceValue{}

func NewUnmeteredEphemeralReferenceValue(
	authorized bool,
	value Value,
	borrowedType sema.Type,
) *EphemeralReferenceValue {
	return &EphemeralReferenceValue{
		Authorized:   authorized,
		Value:        value,
		BorrowedType: borrowedType,
	}
}

func NewEphemeralReferenceValue(
	gauge common.MemoryGauge,
	authorized bool,
	value Value,
	borrowedType sema.Type,
) *EphemeralReferenceValue {
	common.UseMemory(gauge, common.EphemeralReferenceValueMemoryUsage)
	return NewUnmeteredEphemeralReferenceValue(authorized, value, borrowedType)
}

func (*EphemeralReferenceValue) isValue() {}

func (v *EphemeralReferenceValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitEphemeralReferenceValue(interpreter, v)
}

func (*EphemeralReferenceValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (v *EphemeralReferenceValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *EphemeralReferenceValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences)
}

func (v *EphemeralReferenceValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	if _, ok := seenReferences[v]; ok {
		common.UseMemory(memoryGauge, common.SeenReferenceStringMemoryUsage)
		return "..."
	}

	seenReferences[v] = struct{}{}
	defer delete(seenReferences, v)

	return v.Value.MeteredString(memoryGauge, seenReferences)
}

func (v *EphemeralReferenceValue) StaticType(inter *Interpreter) StaticType {
	referencedValue := v.ReferencedValue(inter, EmptyLocationRange, true)
	if referencedValue == nil {
		panic(DereferenceError{
			Cause: "the value being referenced has been destroyed or moved",
		})
	}

	self := *referencedValue

	return NewReferenceStaticType(
		inter,
		v.Authorized,
		ConvertSemaToStaticType(inter, v.BorrowedType),
		self.StaticType(inter),
	)
}

func (*EphemeralReferenceValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v *EphemeralReferenceValue) ReferencedValue(
	_ *Interpreter,
	_ LocationRange,
	_ bool,
) *Value {
	return &v.Value
}

func (v *EphemeralReferenceValue) MustReferencedValue(
	interpreter *Interpreter,
	locationRange LocationRange,
) Value {
	referencedValue := v.ReferencedValue(interpreter, locationRange, true)
	if referencedValue == nil {
		panic(DereferenceError{
			Cause:         "the value being referenced has been destroyed or moved",
			LocationRange: locationRange,
		})
	}

	self := *referencedValue

	interpreter.checkReferencedResourceNotDestroyed(self, locationRange)
	return self
}

func (v *EphemeralReferenceValue) GetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	self := v.MustReferencedValue(interpreter, locationRange)

	return interpreter.getMember(self, locationRange, name)
}

func (v *EphemeralReferenceValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	identifier string,
) Value {
	self := v.MustReferencedValue(interpreter, locationRange)

	if memberAccessibleValue, ok := self.(MemberAccessibleValue); ok {
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
	self := v.MustReferencedValue(interpreter, locationRange)

	return interpreter.setMember(self, locationRange, name, value)
}

func (v *EphemeralReferenceValue) GetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	self := v.MustReferencedValue(interpreter, locationRange)

	return self.(ValueIndexableValue).
		GetKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) SetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	self := v.MustReferencedValue(interpreter, locationRange)

	self.(ValueIndexableValue).
		SetKey(interpreter, locationRange, key, value)
}

func (v *EphemeralReferenceValue) InsertKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	self := v.MustReferencedValue(interpreter, locationRange)

	self.(ValueIndexableValue).
		InsertKey(interpreter, locationRange, key, value)
}

func (v *EphemeralReferenceValue) RemoveKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	self := v.MustReferencedValue(interpreter, locationRange)

	return self.(ValueIndexableValue).
		RemoveKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) GetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
) Value {
	self := v.MustReferencedValue(interpreter, locationRange)

	return self.(TypeIndexableValue).
		GetTypeKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) SetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
	value Value,
) {
	self := v.MustReferencedValue(interpreter, locationRange)

	self.(TypeIndexableValue).
		SetTypeKey(interpreter, locationRange, key, value)
}

func (v *EphemeralReferenceValue) RemoveTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
) Value {
	self := v.MustReferencedValue(interpreter, locationRange)

	return self.(TypeIndexableValue).
		RemoveTypeKey(interpreter, locationRange, key)
}

func (v *EphemeralReferenceValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherReference, ok := other.(*EphemeralReferenceValue)
	if !ok ||
		v.Value != otherReference.Value ||
		v.Authorized != otherReference.Authorized {

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
	referencedValue := v.ReferencedValue(interpreter, locationRange, true)
	if referencedValue == nil {
		return false
	}

	self := *referencedValue

	staticType := self.StaticType(interpreter)

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

func (*EphemeralReferenceValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *EphemeralReferenceValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *EphemeralReferenceValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredEphemeralReferenceValue(v.Authorized, v.Value, v.BorrowedType)
}

func (*EphemeralReferenceValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (*EphemeralReferenceValue) isReference() {}
