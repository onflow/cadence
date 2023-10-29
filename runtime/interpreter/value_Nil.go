package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// NilValue

type NilValue struct{}

var Nil Value = NilValue{}
var NilOptionalValue OptionalValue = NilValue{}
var NilStorable atree.Storable = NilValue{}

var _ Value = NilValue{}
var _ atree.Storable = NilValue{}
var _ EquatableValue = NilValue{}
var _ MemberAccessibleValue = NilValue{}
var _ OptionalValue = NilValue{}

func (NilValue) isValue() {}

func (v NilValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitNilValue(interpreter, v)
}

func (NilValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (NilValue) StaticType(interpreter *Interpreter) StaticType {
	return NewOptionalStaticType(
		interpreter,
		NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeNever),
	)
}

func (NilValue) IsImportable(_ *Interpreter) bool {
	return true
}

func (NilValue) isOptionalValue() {}

func (NilValue) forEach(_ func(Value)) {}

func (v NilValue) fmap(_ *Interpreter, _ func(Value) Value) OptionalValue {
	return v
}

func (NilValue) IsDestroyed() bool {
	return false
}

func (v NilValue) Destroy(_ *Interpreter, _ LocationRange) {
	// NO-OP
}

func (NilValue) String() string {
	return format.Nil
}

func (v NilValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v NilValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(memoryGauge, common.NilValueStringMemoryUsage)
	return v.String()
}

// nilValueMapFunction is created only once per interpreter.
// Hence, no need to meter, as it's a constant.
var nilValueMapFunction = NewUnmeteredHostFunctionValue(
	&sema.FunctionType{
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.NeverType,
		),
	},
	func(invocation Invocation) Value {
		return Nil
	},
)

func (v NilValue) GetMember(_ *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.OptionalTypeMapFunctionName:
		return nilValueMapFunction
	}

	return nil
}

func (NilValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Nil has no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (NilValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Nil has no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v NilValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v NilValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	_, ok := other.(NilValue)
	return ok
}

func (NilValue) IsStorable() bool {
	return true
}

func (v NilValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (NilValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (NilValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v NilValue) Transfer(
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

func (v NilValue) Clone(_ *Interpreter) Value {
	return v
}

func (NilValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v NilValue) ByteSize() uint32 {
	return 1
}

func (v NilValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (NilValue) ChildStorables() []atree.Storable {
	return nil
}
