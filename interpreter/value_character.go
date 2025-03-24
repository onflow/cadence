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
	"golang.org/x/text/unicode/norm"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// CharacterValue

// CharacterValue represents a Cadence character, which is a Unicode extended grapheme cluster.
// Hence, use a Go string to be able to hold multiple Unicode code points (Go runes).
// It should consist of exactly one grapheme cluster
type CharacterValue struct {
	Str             string
	UnnormalizedStr string
}

func NewUnmeteredCharacterValue(str string) CharacterValue {
	return CharacterValue{
		Str:             norm.NFC.String(str),
		UnnormalizedStr: str,
	}
}

// Deprecated: NewStringValue_UnsafeNewCharacterValue_Unsafe creates a new character value
// from the given normalized and unnormalized string.
// NOTE: this function is unsafe, as it does not normalize the string.
// It should only be used for e.g. migration purposes.
func NewCharacterValue_Unsafe(normalizedStr, unnormalizedStr string) CharacterValue {
	return CharacterValue{
		Str:             normalizedStr,
		UnnormalizedStr: unnormalizedStr,
	}
}

func NewCharacterValue(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	characterConstructor func() string,
) CharacterValue {
	common.UseMemory(memoryGauge, memoryUsage)
	character := characterConstructor()
	// NewUnmeteredCharacterValue normalizes (= allocates)
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(character)))
	return NewUnmeteredCharacterValue(character)
}

var _ Value = CharacterValue{}
var _ atree.Storable = CharacterValue{}
var _ EquatableValue = CharacterValue{}
var _ ComparableValue = CharacterValue{}
var _ HashableValue = CharacterValue{}
var _ MemberAccessibleValue = CharacterValue{}

func (CharacterValue) isValue() {}

func (v CharacterValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitCharacterValue(interpreter, v)
}

func (CharacterValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (CharacterValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeCharacter)
}

func (CharacterValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return sema.CharacterType.Importable
}

func (v CharacterValue) String() string {
	return format.String(v.Str)
}

func (v CharacterValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v CharacterValue) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	l := format.FormattedStringLength(v.Str)
	common.UseMemory(context, common.NewRawStringMemoryUsage(l))
	return v.String()
}

func (v CharacterValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		return false
	}
	return v.Str == otherChar.Str
}

func (v CharacterValue) Less(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.Str < otherChar.Str
}

func (v CharacterValue) LessEqual(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.Str <= otherChar.Str
}

func (v CharacterValue) Greater(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.Str > otherChar.Str
}

func (v CharacterValue) GreaterEqual(_ ValueComparisonContext, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v.Str >= otherChar.Str
}

func (v CharacterValue) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
	s := []byte(v.Str)
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

func (CharacterValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v CharacterValue) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		context.RemoveReferencedSlab(storable)
	}
	return v
}

func (v CharacterValue) Clone(_ *Interpreter) Value {
	return v
}

func (CharacterValue) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v CharacterValue) ByteSize() uint32 {
	return values.CBORTagSize + values.GetBytesCBORSize([]byte(v.Str))
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
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.ToStringFunctionType,
			func(v CharacterValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter

				memoryUsage := common.NewStringMemoryUsage(len(v.Str))

				return NewStringValue(
					interpreter,
					memoryUsage,
					func() string {
						return v.Str
					},
				)
			},
		)

	case sema.CharacterTypeUtf8FieldName:
		common.UseMemory(interpreter, common.NewBytesMemoryUsage(len(v.Str)))
		return ByteSliceToByteArrayValue(interpreter, []byte(v.Str))
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
