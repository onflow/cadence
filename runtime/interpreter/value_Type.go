package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// TypeValue

type TypeValue struct {
	// Optional. nil represents "unknown"/"invalid" type
	Type StaticType
}

var EmptyTypeValue = TypeValue{}

var _ Value = TypeValue{}
var _ atree.Storable = TypeValue{}
var _ EquatableValue = TypeValue{}
var _ MemberAccessibleValue = TypeValue{}

func NewUnmeteredTypeValue(t StaticType) TypeValue {
	return TypeValue{Type: t}
}

func NewTypeValue(
	memoryGauge common.MemoryGauge,
	staticType StaticType,
) TypeValue {
	common.UseMemory(memoryGauge, common.TypeValueMemoryUsage)
	return NewUnmeteredTypeValue(staticType)
}

func (TypeValue) isValue() {}

func (v TypeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitTypeValue(interpreter, v)
}

func (TypeValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (TypeValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeMetaType)
}

func (TypeValue) IsImportable(_ *Interpreter) bool {
	return sema.MetaType.Importable
}

func (v TypeValue) String() string {
	var typeString string
	staticType := v.Type
	if staticType != nil {
		typeString = staticType.String()
	}

	return format.TypeValue(typeString)
}

func (v TypeValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v TypeValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(memoryGauge, common.TypeValueStringMemoryUsage)

	var typeString string
	if v.Type != nil {
		typeString = v.Type.MeteredString(memoryGauge)
	}

	return format.TypeValue(typeString)
}

func (v TypeValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherTypeValue, ok := other.(TypeValue)
	if !ok {
		return false
	}

	// Unknown types are never equal to another type

	staticType := v.Type
	otherStaticType := otherTypeValue.Type

	if staticType == nil || otherStaticType == nil {
		return false
	}

	return staticType.Equal(otherStaticType)
}

func (v TypeValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.MetaTypeIdentifierFieldName:
		var typeID string
		staticType := v.Type
		if staticType != nil {
			typeID = string(staticType.ID())
		}
		memoryUsage := common.NewStringMemoryUsage(len(typeID))
		return NewStringValue(interpreter, memoryUsage, func() string {
			return typeID
		})

	case sema.MetaTypeIsSubtypeFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.MetaTypeIsSubtypeFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				staticType := v.Type
				otherTypeValue, ok := invocation.Arguments[0].(TypeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				otherStaticType := otherTypeValue.Type

				// if either type is unknown, the subtype relation is false, as it doesn't make sense to even ask this question
				if staticType == nil || otherStaticType == nil {
					return FalseValue
				}

				result := sema.IsSubType(
					interpreter.MustConvertStaticToSemaType(staticType),
					interpreter.MustConvertStaticToSemaType(otherStaticType),
				)
				return AsBoolValue(result)
			},
		)
	}

	return nil
}

func (TypeValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Types have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (TypeValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Types have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v TypeValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v TypeValue) Storable(
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

func (TypeValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (TypeValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v TypeValue) Transfer(
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

func (v TypeValue) Clone(_ *Interpreter) Value {
	return v
}

func (TypeValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v TypeValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v TypeValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (TypeValue) ChildStorables() []atree.Storable {
	return nil
}

// HashInput returns a byte slice containing:
// - HashInputTypeType (1 byte)
// - type id (n bytes)
func (v TypeValue) HashInput(interpreter *Interpreter, _ LocationRange, scratch []byte) []byte {
	typeID := v.Type.ID()

	length := 1 + len(typeID)
	var buf []byte
	if length <= len(scratch) {
		buf = scratch[:length]
	} else {
		buf = make([]byte, length)
	}

	buf[0] = byte(HashInputTypeType)
	copy(buf[1:], typeID)
	return buf
}
