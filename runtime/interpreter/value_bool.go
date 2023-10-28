package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

// BoolValue

type BoolValue bool

var _ Value = BoolValue(false)
var _ atree.Storable = BoolValue(false)
var _ EquatableValue = BoolValue(false)
var _ HashableValue = BoolValue(false)

const TrueValue = BoolValue(true)
const FalseValue = BoolValue(false)

func AsBoolValue(v bool) BoolValue {
	if v {
		return TrueValue
	}
	return FalseValue
}

func (BoolValue) isValue() {}

func (v BoolValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitBoolValue(interpreter, v)
}

func (BoolValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (BoolValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeBool)
}

func (BoolValue) IsImportable(_ *Interpreter) bool {
	return sema.BoolType.Importable
}

func (v BoolValue) Negate(_ *Interpreter) BoolValue {
	if v == TrueValue {
		return FalseValue
	}
	return TrueValue
}

func (v BoolValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherBool, ok := other.(BoolValue)
	if !ok {
		return false
	}
	return bool(v) == bool(otherBool)
}

func (v BoolValue) Less(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return !v && o
}

func (v BoolValue) LessEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return !v || o
}

func (v BoolValue) Greater(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return v && !o
}

func (v BoolValue) GreaterEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	o, ok := other.(BoolValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	return v || !o
}

// HashInput returns a byte slice containing:
// - HashInputTypeBool (1 byte)
// - 1/0 (1 byte)
func (v BoolValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeBool)
	if v {
		scratch[1] = 1
	} else {
		scratch[1] = 0
	}
	return scratch[:2]
}

func (v BoolValue) String() string {
	return format.Bool(bool(v))
}

func (v BoolValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v BoolValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	if v {
		common.UseMemory(memoryGauge, common.TrueStringMemoryUsage)
	} else {
		common.UseMemory(memoryGauge, common.FalseStringMemoryUsage)
	}

	return v.String()
}

func (v BoolValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v BoolValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (BoolValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (BoolValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v BoolValue) Transfer(
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

func (v BoolValue) Clone(_ *Interpreter) Value {
	return v
}

func (BoolValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v BoolValue) ByteSize() uint32 {
	return 1
}

func (v BoolValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (BoolValue) ChildStorables() []atree.Storable {
	return nil
}
