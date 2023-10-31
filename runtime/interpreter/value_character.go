package interpreter

import (
	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
	"golang.org/x/text/unicode/norm"
)

// CharacterValue

// CharacterValue represents a Cadence character, which is a Unicode extended grapheme cluster.
// Hence, use a Go string to be able to hold multiple Unicode code points (Go runes).
// It should consist of exactly one grapheme cluster
type CharacterValue string

func NewUnmeteredCharacterValue(r string) CharacterValue {
	return CharacterValue(r)
}

func NewCharacterValue(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	characterConstructor func() string,
) CharacterValue {
	common.UseMemory(memoryGauge, memoryUsage)

	character := characterConstructor()
	return NewUnmeteredCharacterValue(character)
}

var _ Value = CharacterValue("a")
var _ atree.Storable = CharacterValue("a")
var _ EquatableValue = CharacterValue("a")
var _ ComparableValue = CharacterValue("a")
var _ HashableValue = CharacterValue("a")
var _ MemberAccessibleValue = CharacterValue("a")

func (CharacterValue) isValue() {}

func (v CharacterValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitCharacterValue(interpreter, v)
}

func (CharacterValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (CharacterValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeCharacter)
}

func (CharacterValue) IsImportable(_ *Interpreter) bool {
	return sema.CharacterType.Importable
}

func (v CharacterValue) String() string {
	return format.String(string(v))
}

func (v CharacterValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v CharacterValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	l := format.FormattedStringLength(string(v))
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(l))
	return v.String()
}

func (v CharacterValue) NormalForm() string {
	return norm.NFC.String(string(v))
}

func (v CharacterValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		return false
	}
	return v.NormalForm() == otherChar.NormalForm()
}

func (v CharacterValue) Less(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.NormalForm() < otherChar.NormalForm()
}

func (v CharacterValue) LessEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.NormalForm() <= otherChar.NormalForm()
}

func (v CharacterValue) Greater(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.NormalForm() > otherChar.NormalForm()
}

func (v CharacterValue) GreaterEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.NormalForm() >= otherChar.NormalForm()
}

func (v CharacterValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	s := []byte(string(v))
	length := 1 + len(s)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeCharacter)
	copy(buffer[1:], s)
	return buffer
}

func (v CharacterValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v CharacterValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (CharacterValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (CharacterValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v CharacterValue) Transfer(
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

func (v CharacterValue) Clone(_ *Interpreter) Value {
	return v
}

func (CharacterValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v CharacterValue) ByteSize() uint32 {
	return cborTagSize + getBytesCBORSize([]byte(v))
}

func (v CharacterValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (CharacterValue) ChildStorables() []atree.Storable {
	return nil
}

func (v CharacterValue) GetMember(interpreter *Interpreter, _ LocationRange, name string) Value {
	switch name {
	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ToStringFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				memoryUsage := common.NewStringMemoryUsage(len(v))

				return NewStringValue(
					interpreter,
					memoryUsage,
					func() string {
						return string(v)
					},
				)
			},
		)

	case sema.CharacterTypeUtf8FieldName:
		common.UseMemory(interpreter, common.NewBytesMemoryUsage(len(v)))
		return ByteSliceToByteArrayValue(interpreter, []byte(v))
	}
	return nil
}

func (CharacterValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Characters have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (CharacterValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Characters have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}
