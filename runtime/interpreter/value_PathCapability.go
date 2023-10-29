package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// PathCapabilityValue

type PathCapabilityValue struct {
	BorrowType StaticType
	Path       PathValue
	Address    AddressValue
}

func NewUnmeteredPathCapabilityValue(
	address AddressValue,
	path PathValue,
	borrowType StaticType,
) *PathCapabilityValue {
	return &PathCapabilityValue{
		Address:    address,
		Path:       path,
		BorrowType: borrowType,
	}
}

func NewPathCapabilityValue(
	memoryGauge common.MemoryGauge,
	address AddressValue,
	path PathValue,
	borrowType StaticType,
) *PathCapabilityValue {
	// Constant because its constituents are already metered.
	common.UseMemory(memoryGauge, common.PathCapabilityValueMemoryUsage)
	return NewUnmeteredPathCapabilityValue(address, path, borrowType)
}

var _ Value = &PathCapabilityValue{}
var _ atree.Storable = &PathCapabilityValue{}
var _ EquatableValue = &PathCapabilityValue{}
var _ CapabilityValue = &PathCapabilityValue{}
var _ MemberAccessibleValue = &PathCapabilityValue{}

func (*PathCapabilityValue) isValue() {}

func (*PathCapabilityValue) isCapabilityValue() {}

func (v *PathCapabilityValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitPathCapabilityValue(interpreter, v)
}

func (v *PathCapabilityValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.Address)
	walkChild(v.Path)
}

func (v *PathCapabilityValue) StaticType(inter *Interpreter) StaticType {
	return NewCapabilityStaticType(
		inter,
		v.BorrowType,
	)
}

func (v *PathCapabilityValue) IsImportable(_ *Interpreter) bool {
	return v.Path.Domain == common.PathDomainPublic
}

func (v *PathCapabilityValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *PathCapabilityValue) RecursiveString(seenReferences SeenReferences) string {
	var borrowType string
	if v.BorrowType != nil {
		borrowType = v.BorrowType.String()
	}
	return format.PathCapability(
		borrowType,
		v.Address.RecursiveString(seenReferences),
		v.Path.RecursiveString(seenReferences),
	)
}

func (v *PathCapabilityValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	common.UseMemory(memoryGauge, common.PathCapabilityValueStringMemoryUsage)

	var borrowType string
	if v.BorrowType != nil {
		borrowType = v.BorrowType.MeteredString(memoryGauge)
	}

	return format.PathCapability(
		borrowType,
		v.Address.MeteredString(memoryGauge, seenReferences),
		v.Path.MeteredString(memoryGauge, seenReferences),
	)
}

func (v *PathCapabilityValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.CapabilityTypeBorrowFunctionName:
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			// this function will panic already if this conversion fails
			borrowType, _ = interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return interpreter.pathCapabilityBorrowFunction(v.Address, v.Path, borrowType)

	case sema.CapabilityTypeCheckFunctionName:
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			// this function will panic already if this conversion fails
			borrowType, _ = interpreter.MustConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return interpreter.pathCapabilityCheckFunction(v.Address, v.Path, borrowType)

	case sema.CapabilityTypeAddressFieldName:
		return v.Address

	case sema.CapabilityTypeIDFieldName:
		return UInt64Value(0)
	}

	return nil
}

func (*PathCapabilityValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Capabilities have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*PathCapabilityValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Capabilities have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *PathCapabilityValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *PathCapabilityValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherCapability, ok := other.(*PathCapabilityValue)
	if !ok {
		return false
	}

	// BorrowType is optional

	if v.BorrowType == nil {
		if otherCapability.BorrowType != nil {
			return false
		}
	} else if !v.BorrowType.Equal(otherCapability.BorrowType) {
		return false
	}

	return otherCapability.Address.Equal(interpreter, locationRange, v.Address) &&
		otherCapability.Path.Equal(interpreter, locationRange, v.Path)
}

func (*PathCapabilityValue) IsStorable() bool {
	return true
}

func (v *PathCapabilityValue) Storable(
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

func (*PathCapabilityValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*PathCapabilityValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *PathCapabilityValue) Transfer(
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

func (v *PathCapabilityValue) Clone(interpreter *Interpreter) Value {
	return NewUnmeteredPathCapabilityValue(
		v.Address.Clone(interpreter).(AddressValue),
		v.Path.Clone(interpreter).(PathValue),
		v.BorrowType,
	)
}

func (v *PathCapabilityValue) DeepRemove(interpreter *Interpreter) {
	v.Address.DeepRemove(interpreter)
	v.Path.DeepRemove(interpreter)
}

func (v *PathCapabilityValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *PathCapabilityValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *PathCapabilityValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.Address,
		v.Path,
	}
}

// PathLinkValue

type PathLinkValue struct {
	Type       StaticType
	TargetPath PathValue
}

func NewUnmeteredPathLinkValue(targetPath PathValue, staticType StaticType) PathLinkValue {
	return PathLinkValue{
		TargetPath: targetPath,
		Type:       staticType,
	}
}

func NewPathLinkValue(memoryGauge common.MemoryGauge, targetPath PathValue, staticType StaticType) PathLinkValue {
	// The only variable is TargetPath, which is already metered as a PathValue.
	common.UseMemory(memoryGauge, common.PathLinkValueMemoryUsage)
	return NewUnmeteredPathLinkValue(targetPath, staticType)
}

var EmptyPathLinkValue = PathLinkValue{}

var _ Value = PathLinkValue{}
var _ atree.Value = PathLinkValue{}
var _ EquatableValue = PathLinkValue{}
var _ LinkValue = PathLinkValue{}

func (PathLinkValue) isValue() {}

func (PathLinkValue) isLinkValue() {}

func (v PathLinkValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitPathLinkValue(interpreter, v)
}

func (v PathLinkValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.TargetPath)
}

func (v PathLinkValue) StaticType(interpreter *Interpreter) StaticType {
	// When iterating over public/private paths,
	// the values at these paths are PathLinkValues,
	// placed there by the `link` function.
	//
	// These are loaded as links, however,
	// for the purposes of checking their type,
	// we treat them as capabilities
	return NewCapabilityStaticType(interpreter, v.Type)
}

func (PathLinkValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v PathLinkValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v PathLinkValue) RecursiveString(seenReferences SeenReferences) string {
	return format.PathLink(
		v.Type.String(),
		v.TargetPath.RecursiveString(seenReferences),
	)
}

func (v PathLinkValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	common.UseMemory(memoryGauge, common.PathLinkValueStringMemoryUsage)

	return format.PathLink(
		v.Type.MeteredString(memoryGauge),
		v.TargetPath.MeteredString(memoryGauge, seenReferences),
	)
}

func (v PathLinkValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v PathLinkValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherLink, ok := other.(PathLinkValue)
	if !ok {
		return false
	}

	return otherLink.TargetPath.Equal(interpreter, locationRange, v.TargetPath) &&
		otherLink.Type.Equal(v.Type)
}

func (PathLinkValue) IsStorable() bool {
	return true
}

func (v PathLinkValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (PathLinkValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (PathLinkValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v PathLinkValue) Transfer(
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

func (v PathLinkValue) Clone(interpreter *Interpreter) Value {
	return PathLinkValue{
		TargetPath: v.TargetPath.Clone(interpreter).(PathValue),
		Type:       v.Type,
	}
}

func (PathLinkValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v PathLinkValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v PathLinkValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v PathLinkValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.TargetPath,
	}
}
