package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// IDCapabilityValue

type IDCapabilityValue struct {
	BorrowType StaticType
	Address    AddressValue
	ID         UInt64Value
}

func NewUnmeteredIDCapabilityValue(
	id UInt64Value,
	address AddressValue,
	borrowType StaticType,
) *IDCapabilityValue {
	return &IDCapabilityValue{
		ID:         id,
		Address:    address,
		BorrowType: borrowType,
	}
}

func NewIDCapabilityValue(
	memoryGauge common.MemoryGauge,
	id UInt64Value,
	address AddressValue,
	borrowType StaticType,
) *IDCapabilityValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.IDCapabilityValueMemoryUsage)
	return NewUnmeteredIDCapabilityValue(id, address, borrowType)
}

var _ Value = &IDCapabilityValue{}
var _ atree.Storable = &IDCapabilityValue{}
var _ CapabilityValue = &IDCapabilityValue{}
var _ EquatableValue = &IDCapabilityValue{}
var _ MemberAccessibleValue = &IDCapabilityValue{}

func (*IDCapabilityValue) isValue() {}

func (*IDCapabilityValue) isCapabilityValue() {}

func (v *IDCapabilityValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitIDCapabilityValue(interpreter, v)
}

func (v *IDCapabilityValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.ID)
	walkChild(v.Address)
}

func (v *IDCapabilityValue) StaticType(inter *Interpreter) StaticType {
	return NewCapabilityStaticType(
		inter,
		v.BorrowType,
	)
}

func (v *IDCapabilityValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v *IDCapabilityValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *IDCapabilityValue) RecursiveString(seenReferences SeenReferences) string {
	return format.IDCapability(
		v.BorrowType.String(),
		v.Address.RecursiveString(seenReferences),
		v.ID.RecursiveString(seenReferences),
	)
}

func (v *IDCapabilityValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	common.UseMemory(memoryGauge, common.IDCapabilityValueStringMemoryUsage)

	return format.IDCapability(
		v.BorrowType.MeteredString(memoryGauge),
		v.Address.MeteredString(memoryGauge, seenReferences),
		v.ID.MeteredString(memoryGauge, seenReferences),
	)
}

func (v *IDCapabilityValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.CapabilityTypeBorrowFunctionName:
		// this function will panic already if this conversion fails
		borrowType, _ := interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		return interpreter.idCapabilityBorrowFunction(v.Address, v.ID, borrowType)

	case sema.CapabilityTypeCheckFunctionName:
		// this function will panic already if this conversion fails
		borrowType, _ := interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		return interpreter.idCapabilityCheckFunction(v.Address, v.ID, borrowType)

	case sema.CapabilityTypeAddressFieldName:
		return v.Address

	case sema.CapabilityTypeIDFieldName:
		return v.ID
	}

	return nil
}

func (*IDCapabilityValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Capabilities have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*IDCapabilityValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Capabilities have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *IDCapabilityValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *IDCapabilityValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherCapability, ok := other.(*IDCapabilityValue)
	if !ok {
		return false
	}

	return otherCapability.ID == v.ID &&
		otherCapability.Address.Equal(interpreter, locationRange, v.Address) &&
		otherCapability.BorrowType.Equal(v.BorrowType)
}

func (*IDCapabilityValue) IsStorable() bool {
	return true
}

func (v *IDCapabilityValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return maybeLargeImmutableStorable(
		v,
		storage,
		address,
		maxInlineSize,
	)
}

func (*IDCapabilityValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*IDCapabilityValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *IDCapabilityValue) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.StorageID]struct{},
) Value {
	if remove {
		v.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *IDCapabilityValue) Clone(interpreter *Interpreter) Value {
	return NewUnmeteredIDCapabilityValue(
		v.ID,
		v.Address.Clone(interpreter).(AddressValue),
		v.BorrowType,
	)
}

func (v *IDCapabilityValue) DeepRemove(interpreter *Interpreter) {
	v.Address.DeepRemove(interpreter)
}

func (v *IDCapabilityValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *IDCapabilityValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *IDCapabilityValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.Address,
	}
}
