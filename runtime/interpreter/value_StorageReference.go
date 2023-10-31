package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// StorageReferenceValue

type StorageReferenceValue struct {
	BorrowedType         sema.Type
	TargetPath           PathValue
	TargetStorageAddress common.Address
	Authorized           bool
}

var _ Value = &StorageReferenceValue{}
var _ EquatableValue = &StorageReferenceValue{}
var _ ValueIndexableValue = &StorageReferenceValue{}
var _ TypeIndexableValue = &StorageReferenceValue{}
var _ MemberAccessibleValue = &StorageReferenceValue{}
var _ ReferenceValue = &StorageReferenceValue{}

func NewUnmeteredStorageReferenceValue(
	authorized bool,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType sema.Type,
) *StorageReferenceValue {
	return &StorageReferenceValue{
		Authorized:           authorized,
		TargetStorageAddress: targetStorageAddress,
		TargetPath:           targetPath,
		BorrowedType:         borrowedType,
	}
}

func NewStorageReferenceValue(
	memoryGauge common.MemoryGauge,
	authorized bool,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType sema.Type,
) *StorageReferenceValue {
	common.UseMemory(memoryGauge, common.StorageReferenceValueMemoryUsage)
	return NewUnmeteredStorageReferenceValue(
		authorized,
		targetStorageAddress,
		targetPath,
		borrowedType,
	)
}

func (*StorageReferenceValue) isValue() {}

func (v *StorageReferenceValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitStorageReferenceValue(interpreter, v)
}

func (*StorageReferenceValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (*StorageReferenceValue) String() string {
	return format.StorageReference
}

func (v *StorageReferenceValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *StorageReferenceValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(memoryGauge, common.StorageReferenceValueStringMemoryUsage)
	return v.String()
}

func (v *StorageReferenceValue) StaticType(inter *Interpreter) StaticType {
	referencedValue, err := v.dereference(inter, EmptyLocationRange)
	if err != nil {
		panic(err)
	}

	self := *referencedValue

	return NewReferenceStaticType(
		inter,
		v.Authorized,
		ConvertSemaToStaticType(inter, v.BorrowedType),
		self.StaticType(inter),
	)
}

func (*StorageReferenceValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v *StorageReferenceValue) dereference(interpreter *Interpreter, locationRange LocationRange) (*Value, error) {
	address := v.TargetStorageAddress
	domain := v.TargetPath.Domain.Identifier()
	identifier := v.TargetPath.Identifier

	storageMapKey := StringStorageMapKey(identifier)

	referenced := interpreter.ReadStored(address, domain, storageMapKey)
	if referenced == nil {
		return nil, nil
	}

	if v.BorrowedType != nil {
		staticType := referenced.StaticType(interpreter)

		if !interpreter.IsSubTypeOfSemaType(staticType, v.BorrowedType) {
			semaType := interpreter.MustConvertStaticToSemaType(staticType)

			return nil, ForceCastTypeMismatchError{
				ExpectedType:  v.BorrowedType,
				ActualType:    semaType,
				LocationRange: locationRange,
			}
		}
	}

	return &referenced, nil
}

func (v *StorageReferenceValue) ReferencedValue(interpreter *Interpreter, locationRange LocationRange, errorOnFailedDereference bool) *Value {
	referencedValue, err := v.dereference(interpreter, locationRange)
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
	interpreter *Interpreter,
	locationRange LocationRange,
) Value {
	referencedValue := v.ReferencedValue(interpreter, locationRange, true)
	if referencedValue == nil {
		panic(DereferenceError{
			Cause:         "no value is stored at this path",
			LocationRange: locationRange,
		})
	}

	self := *referencedValue

	interpreter.checkReferencedResourceNotDestroyed(self, locationRange)

	return self
}

func (v *StorageReferenceValue) GetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

	return interpreter.getMember(self, locationRange, name)
}

func (v *StorageReferenceValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

	return self.(MemberAccessibleValue).RemoveMember(interpreter, locationRange, name)
}

func (v *StorageReferenceValue) SetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
	value Value,
) bool {
	self := v.mustReferencedValue(interpreter, locationRange)

	return interpreter.setMember(self, locationRange, name, value)
}

func (v *StorageReferenceValue) GetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

	return self.(ValueIndexableValue).
		GetKey(interpreter, locationRange, key)
}

func (v *StorageReferenceValue) SetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	self := v.mustReferencedValue(interpreter, locationRange)

	self.(ValueIndexableValue).
		SetKey(interpreter, locationRange, key, value)
}

func (v *StorageReferenceValue) InsertKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
	value Value,
) {
	self := v.mustReferencedValue(interpreter, locationRange)

	self.(ValueIndexableValue).
		InsertKey(interpreter, locationRange, key, value)
}

func (v *StorageReferenceValue) RemoveKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

	return self.(ValueIndexableValue).
		RemoveKey(interpreter, locationRange, key)
}

func (v *StorageReferenceValue) GetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key sema.Type,
) Value {
	self := v.mustReferencedValue(interpreter, locationRange)

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

func (v *StorageReferenceValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherReference, ok := other.(*StorageReferenceValue)
	if !ok ||
		v.TargetStorageAddress != otherReference.TargetStorageAddress ||
		v.TargetPath != otherReference.TargetPath ||
		v.Authorized != otherReference.Authorized {

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

	if !interpreter.IsSubTypeOfSemaType(staticType, v.BorrowedType) {
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

func (*StorageReferenceValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *StorageReferenceValue) Transfer(
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

func (v *StorageReferenceValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredStorageReferenceValue(
		v.Authorized,
		v.TargetStorageAddress,
		v.TargetPath,
		v.BorrowedType,
	)
}

func (*StorageReferenceValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (*StorageReferenceValue) isReference() {}
