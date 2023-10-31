package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
)

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
