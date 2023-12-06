/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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
	"encoding/binary"
	"encoding/hex"
	goerrors "errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/onflow/atree"
	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/norm"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

type Unsigned interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

type TypeConformanceResults map[typeConformanceResultEntry]bool

type typeConformanceResultEntry struct {
	EphemeralReferenceValue *EphemeralReferenceValue
	EphemeralReferenceType  StaticType
}

// SeenReferences is a set of seen references.
//
// NOTE: Do not generalize to map[interpreter.Value],
// as not all values are Go hashable, i.e. this might lead to run-time panics
type SeenReferences map[*EphemeralReferenceValue]struct{}

// NonStorable represents a value that cannot be stored
type NonStorable struct {
	Value Value
}

var _ atree.Storable = NonStorable{}

func (s NonStorable) Encode(_ *atree.Encoder) error {
	//nolint:gosimple
	return NonStorableValueError{
		Value: s.Value,
	}
}

func (s NonStorable) ByteSize() uint32 {
	// Return 1 so that atree split and merge operations don't have to handle special cases.
	// Any value larger than 0 and smaller than half of the max slab size works,
	// but 1 results in fewer number of slabs which is ideal for non-storable values.
	return 1
}

func (s NonStorable) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return s.Value, nil
}

func (NonStorable) ChildStorables() []atree.Storable {
	return nil
}

// Value is the Cadence value hierarchy which is heavily tied to the interpreter and persistent storage,
// and has lots of implementation details.
//
// We do not want to expose those details to users
// (for example, Cadence is used as a library in flow-go (FVM), in the Flow Go SDK, etc.),
// because we want to be able to change the API and implementation details;
// nor do we want to require users to Cadence (the library) to write lots of low-level/boilerplate code (e.g. setting up storage).
//
// To accomplish this, cadence.Value is the "user-facing" hierarchy that is easy to work with:
// simple Go types that can be used without an interpreter or storage.
//
// cadence.Value can be converted to an interpreter.Value by "importing" it with importValue,
// and interpreter.Value can be "exported" to a cadence.Value with ExportValue.
type Value interface {
	atree.Value
	// Stringer provides `func String() string`
	// NOTE: important, error messages rely on values to implement String
	fmt.Stringer
	isValue()
	Accept(interpreter *Interpreter, visitor Visitor)
	Walk(interpreter *Interpreter, walkChild func(Value))
	StaticType(interpreter *Interpreter) StaticType
	// ConformsToStaticType returns true if the value (i.e. its dynamic type)
	// conforms to its own static type.
	// Non-container values trivially always conform to their own static type.
	// Container values conform to their own static type,
	// and this function recursively checks conformance for nested values.
	// If the container contains static type information about nested values,
	// e.g. the element type of an array, it also ensures the nested values'
	// static types are subtypes.
	ConformsToStaticType(
		interpreter *Interpreter,
		locationRange LocationRange,
		results TypeConformanceResults,
	) bool
	RecursiveString(seenReferences SeenReferences) string
	MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string
	IsResourceKinded(interpreter *Interpreter) bool
	NeedsStoreTo(address atree.Address) bool
	Transfer(
		interpreter *Interpreter,
		locationRange LocationRange,
		address atree.Address,
		remove bool,
		storable atree.Storable,
		preventTransfer map[atree.StorageID]struct{},
	) Value
	DeepRemove(interpreter *Interpreter)
	// Clone returns a new value that is equal to this value.
	// NOTE: not used by interpreter, but used externally (e.g. state migration)
	// NOTE: memory metering is unnecessary for Clone methods
	Clone(interpreter *Interpreter) Value
	IsImportable(interpreter *Interpreter) bool
}

// ValueIndexableValue

type ValueIndexableValue interface {
	Value
	GetKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value
	SetKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value)
	RemoveKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value
	InsertKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value)
}

type TypeIndexableValue interface {
	Value
	GetTypeKey(interpreter *Interpreter, locationRange LocationRange, ty sema.Type) Value
	SetTypeKey(interpreter *Interpreter, locationRange LocationRange, ty sema.Type, value Value)
	RemoveTypeKey(interpreter *Interpreter, locationRange LocationRange, ty sema.Type) Value
}

// MemberAccessibleValue

type MemberAccessibleValue interface {
	Value
	GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value
	RemoveMember(interpreter *Interpreter, locationRange LocationRange, name string) Value
	// returns whether a value previously existed with this name
	SetMember(interpreter *Interpreter, locationRange LocationRange, name string, value Value) bool
}

// EquatableValue

type EquatableValue interface {
	Value
	// Equal returns true if the given value is equal to this value.
	// If no location range is available, pass e.g. EmptyLocationRange
	Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool
}

func newValueComparator(interpreter *Interpreter, locationRange LocationRange) atree.ValueComparator {
	return func(storage atree.SlabStorage, atreeValue atree.Value, otherStorable atree.Storable) (bool, error) {
		value := MustConvertStoredValue(interpreter, atreeValue)
		otherValue := StoredValue(interpreter, otherStorable, storage)
		return value.(EquatableValue).Equal(interpreter, locationRange, otherValue), nil
	}
}

// ComparableValue
type ComparableValue interface {
	EquatableValue
	Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
	LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
	Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
	GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue
}

// ResourceKindedValue

type ResourceKindedValue interface {
	Value
	Destroy(interpreter *Interpreter, locationRange LocationRange)
	IsDestroyed() bool
}

func maybeDestroy(interpreter *Interpreter, locationRange LocationRange, value Value) {
	resourceKindedValue, ok := value.(ResourceKindedValue)
	if !ok {
		return
	}

	resourceKindedValue.Destroy(interpreter, locationRange)
}

// ReferenceTrackedResourceKindedValue is a resource-kinded value
// that must be tracked when a reference of it is taken.
type ReferenceTrackedResourceKindedValue interface {
	ResourceKindedValue
	IsReferenceTrackedResourceKindedValue()
	StorageID() atree.StorageID
	IsStaleResource(*Interpreter) bool
}

// ContractValue is the value of a contract.
// Under normal circumstances, a contract value is always a CompositeValue.
// However, in the test framework, an imported contract is constructed via a constructor function.
// Hence, during tests, the value is a HostFunctionValue.
type ContractValue interface {
	Value
	SetNestedVariables(variables map[string]*Variable)
}

// IterableValue is a value which can be iterated over, e.g. with a for-loop
type IterableValue interface {
	Value
	Iterator(interpreter *Interpreter) ValueIterator
	ForEach(
		interpreter *Interpreter,
		elementType sema.Type,
		function func(value Value) (resume bool),
		locationRange LocationRange,
	)
}

// ValueIterator is an iterator which returns values.
// When Next returns nil, it signals the end of the iterator.
type ValueIterator interface {
	Next(interpreter *Interpreter) Value
}

func safeAdd(a, b int, locationRange LocationRange) int {
	// INT32-C
	if (b > 0) && (a > (goMaxInt - b)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (b < 0) && (a < (goMinInt - b)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	return a + b
}

func safeMul(a, b int, locationRange LocationRange) int {
	// INT32-C
	if a > 0 {
		if b > 0 {
			// positive * positive = positive. overflow?
			if a > (goMaxInt / b) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if b < (goMinInt / a) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if b > 0 {
			// negative * positive = negative. underflow?
			if a < (goMinInt / b) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (a != 0) && (b < (goMaxInt / a)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}
	return a * b
}

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

// VoidValue

type VoidValue struct{}

var Void Value = VoidValue{}
var VoidStorable atree.Storable = VoidValue{}

var _ Value = VoidValue{}
var _ atree.Storable = VoidValue{}
var _ EquatableValue = VoidValue{}

func (VoidValue) isValue() {}

func (v VoidValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitVoidValue(interpreter, v)
}

func (VoidValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (VoidValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeVoid)
}

func (VoidValue) IsImportable(_ *Interpreter) bool {
	return sema.VoidType.Importable
}

func (VoidValue) String() string {
	return format.Void
}

func (v VoidValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v VoidValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(memoryGauge, common.VoidStringMemoryUsage)
	return v.String()
}

func (v VoidValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v VoidValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	_, ok := other.(VoidValue)
	return ok
}

func (v VoidValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (VoidValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (VoidValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v VoidValue) Transfer(
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

func (v VoidValue) Clone(_ *Interpreter) Value {
	return v
}

func (VoidValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (VoidValue) ByteSize() uint32 {
	return uint32(len(cborVoidValue))
}

func (v VoidValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (VoidValue) ChildStorables() []atree.Storable {
	return nil
}

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

// CharacterValue

// CharacterValue represents a Cadence character, which is a Unicode extended grapheme cluster.
// Hence, use a Go string to be able to hold multiple Unicode code points (Go runes).
// It should consist of exactly one grapheme cluster
type CharacterValue string

func NewUnmeteredCharacterValue(r string) CharacterValue {
	return CharacterValue(norm.NFC.String(r))
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

func (v CharacterValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		return false
	}
	return v == otherChar
}

func (v CharacterValue) Less(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v < otherChar
}

func (v CharacterValue) LessEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v <= otherChar
}

func (v CharacterValue) Greater(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v > otherChar
}

func (v CharacterValue) GreaterEqual(_ *Interpreter, other ComparableValue, _ LocationRange) BoolValue {
	otherChar, ok := other.(CharacterValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return v >= otherChar
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

// StringValue

type StringValue struct {
	// graphemes is a grapheme cluster segmentation iterator,
	// which is initialized lazily and reused/reset in functions
	// that are based on grapheme clusters
	graphemes *uniseg.Graphemes
	Str       string
	// length is the cached length of the string, based on grapheme clusters.
	// a negative value indicates the length has not been initialized, see Length()
	length int
}

func NewUnmeteredStringValue(str string) *StringValue {
	return &StringValue{
		Str: norm.NFC.String(str),
		// a negative value indicates the length has not been initialized, see Length()
		length: -1,
	}
}

func NewStringValue(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	stringConstructor func() string,
) *StringValue {
	common.UseMemory(memoryGauge, memoryUsage)
	str := stringConstructor()
	// NewUnmeteredStringValue normalizes (= allocates)
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(len(str)))
	return NewUnmeteredStringValue(str)
}

var _ Value = &StringValue{}
var _ atree.Storable = &StringValue{}
var _ EquatableValue = &StringValue{}
var _ ComparableValue = &StringValue{}
var _ HashableValue = &StringValue{}
var _ ValueIndexableValue = &StringValue{}
var _ MemberAccessibleValue = &StringValue{}
var _ IterableValue = &StringValue{}

var VarSizedArrayOfStringType = NewVariableSizedStaticType(nil, PrimitiveStaticTypeString)

func (v *StringValue) prepareGraphemes() {
	if v.graphemes == nil {
		v.graphemes = uniseg.NewGraphemes(v.Str)
	} else {
		v.graphemes.Reset()
	}
}

func (*StringValue) isValue() {}

func (v *StringValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitStringValue(interpreter, v)
}

func (*StringValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (*StringValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeString)
}

func (*StringValue) IsImportable(_ *Interpreter) bool {
	return sema.StringType.Importable
}

func (v *StringValue) String() string {
	return format.String(v.Str)
}

func (v *StringValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *StringValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	l := format.FormattedStringLength(v.Str)
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(l))
	return v.String()
}

func (v *StringValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherString, ok := other.(*StringValue)
	if !ok {
		return false
	}
	return v.Str == otherString.Str
}

func (v *StringValue) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v.Str < otherString.Str)
}

func (v *StringValue) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v.Str <= otherString.Str)
}

func (v *StringValue) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v.Str > otherString.Str)
}

func (v *StringValue) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v.Str >= otherString.Str)
}

// HashInput returns a byte slice containing:
// - HashInputTypeString (1 byte)
// - string value (n bytes)
func (v *StringValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	length := 1 + len(v.Str)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeString)
	copy(buffer[1:], v.Str)
	return buffer
}

func (v *StringValue) Concat(interpreter *Interpreter, other *StringValue, locationRange LocationRange) Value {

	firstLength := len(v.Str)
	secondLength := len(other.Str)

	newLength := safeAdd(firstLength, secondLength, locationRange)

	memoryUsage := common.NewStringMemoryUsage(newLength)

	return NewStringValue(
		interpreter,
		memoryUsage,
		func() string {
			var sb strings.Builder

			sb.WriteString(v.Str)
			sb.WriteString(other.Str)

			return sb.String()
		},
	)
}

var EmptyString = NewUnmeteredStringValue("")

func (v *StringValue) Slice(from IntValue, to IntValue, locationRange LocationRange) Value {
	fromIndex := from.ToInt(locationRange)

	toIndex := to.ToInt(locationRange)

	length := v.Length()

	if fromIndex < 0 || fromIndex > length || toIndex < 0 || toIndex > length {
		panic(StringSliceIndicesError{
			FromIndex:     fromIndex,
			UpToIndex:     toIndex,
			Length:        length,
			LocationRange: locationRange,
		})
	}

	if fromIndex > toIndex {
		panic(InvalidSliceIndexError{
			FromIndex:     fromIndex,
			UpToIndex:     toIndex,
			LocationRange: locationRange,
		})
	}

	if fromIndex == toIndex {
		return EmptyString
	}

	v.prepareGraphemes()

	j := 0

	for ; j <= fromIndex; j++ {
		v.graphemes.Next()
	}
	start, _ := v.graphemes.Positions()

	for ; j < toIndex; j++ {
		v.graphemes.Next()
	}
	_, end := v.graphemes.Positions()

	// NOTE: string slicing in Go does not copy,
	// see https://stackoverflow.com/questions/52395730/does-slice-of-string-perform-copy-of-underlying-data
	return NewUnmeteredStringValue(v.Str[start:end])
}

func (v *StringValue) checkBounds(index int, locationRange LocationRange) {
	length := v.Length()

	if index < 0 || index >= length {
		panic(StringIndexOutOfBoundsError{
			Index:         index,
			Length:        length,
			LocationRange: locationRange,
		})
	}
}

func (v *StringValue) GetKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value {
	index := key.(NumberValue).ToInt(locationRange)
	v.checkBounds(index, locationRange)

	v.prepareGraphemes()

	for j := 0; j <= index; j++ {
		v.graphemes.Next()
	}

	char := v.graphemes.Str()
	return NewCharacterValue(
		interpreter,
		common.NewCharacterMemoryUsage(len(char)),
		func() string {
			return char
		},
	)
}

func (*StringValue) SetKey(_ *Interpreter, _ LocationRange, _ Value, _ Value) {
	panic(errors.NewUnreachableError())
}

func (*StringValue) InsertKey(_ *Interpreter, _ LocationRange, _ Value, _ Value) {
	panic(errors.NewUnreachableError())
}

func (*StringValue) RemoveKey(_ *Interpreter, _ LocationRange, _ Value) Value {
	panic(errors.NewUnreachableError())
}

func (v *StringValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	switch name {
	case sema.StringTypeLengthFieldName:
		length := v.Length()
		return NewIntValueFromInt64(interpreter, int64(length))

	case sema.StringTypeUtf8FieldName:
		return ByteSliceToByteArrayValue(interpreter, []byte(v.Str))

	case sema.StringTypeConcatFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.StringTypeConcatFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter
				otherArray, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.Concat(interpreter, otherArray, locationRange)
			},
		)

	case sema.StringTypeSliceFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.StringTypeSliceFunctionType,
			func(invocation Invocation) Value {
				from, ok := invocation.Arguments[0].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				to, ok := invocation.Arguments[1].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Slice(from, to, invocation.LocationRange)
			},
		)

	case sema.StringTypeDecodeHexFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.StringTypeDecodeHexFunctionType,
			func(invocation Invocation) Value {
				return v.DecodeHex(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case sema.StringTypeToLowerFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.StringTypeToLowerFunctionType,
			func(invocation Invocation) Value {
				return v.ToLower(invocation.Interpreter)
			},
		)

	case sema.StringTypeSplitFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.StringTypeSplitFunctionType,
			func(invocation Invocation) Value {
				separator, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Split(invocation.Interpreter, invocation.LocationRange, separator.Str)
			},
		)

	case sema.StringTypeReplaceAllFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.StringTypeReplaceAllFunctionType,
			func(invocation Invocation) Value {
				of, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				with, ok := invocation.Arguments[1].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.ReplaceAll(invocation.Interpreter, invocation.LocationRange, of.Str, with.Str)
			},
		)
	}

	return nil
}

func (*StringValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Strings have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*StringValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Strings have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

// Length returns the number of characters (grapheme clusters)
func (v *StringValue) Length() int {
	if v.length < 0 {
		var length int
		v.prepareGraphemes()
		for v.graphemes.Next() {
			length++
		}
		v.length = length
	}
	return v.length
}

func (v *StringValue) ToLower(interpreter *Interpreter) *StringValue {

	// Over-estimate resulting string length,
	// as an uppercase character may be converted to several lower-case characters, e.g İ => [i, ̇]
	// see https://stackoverflow.com/questions/28683805/is-there-a-unicode-string-which-gets-longer-when-converted-to-lowercase

	var lengthEstimate int
	for _, r := range v.Str {
		if r < unicode.MaxASCII {
			lengthEstimate += 1
		} else {
			lengthEstimate += utf8.UTFMax
		}
	}

	memoryUsage := common.NewStringMemoryUsage(lengthEstimate)

	return NewStringValue(
		interpreter,
		memoryUsage,
		func() string {
			return strings.ToLower(v.Str)
		},
	)
}

func (v *StringValue) Split(inter *Interpreter, locationRange LocationRange, separator string) Value {
	split := strings.Split(v.Str, separator)

	var index int
	count := len(split)

	return NewArrayValueWithIterator(
		inter,
		VarSizedArrayOfStringType,
		common.ZeroAddress,
		uint64(count),
		func() Value {
			if index >= count {
				return nil
			}

			str := split[index]
			index++
			return NewStringValue(
				inter,
				common.NewStringMemoryUsage(len(str)),
				func() string {
					return str
				},
			)
		},
	)
}

func (v *StringValue) ReplaceAll(inter *Interpreter, locationRange LocationRange, of string, with string) *StringValue {
	// Over-estimate the resulting string length.
	// In the worst case, `of` can be empty in which case, `with` will be added at every index.
	// e.g. `of` = "", `v` = "ABC", `with` = "1": result = "1A1B1C1".
	lengthOverEstimate := (2*len(v.Str) + 1) * len(with)

	memoryUsage := common.NewStringMemoryUsage(lengthOverEstimate)

	return NewStringValue(
		inter,
		memoryUsage,
		func() string {
			return strings.ReplaceAll(v.Str, of, with)
		},
	)
}

func (v *StringValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (*StringValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*StringValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *StringValue) Transfer(
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

func (v *StringValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredStringValue(v.Str)
}

func (*StringValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v *StringValue) ByteSize() uint32 {
	return cborTagSize + getBytesCBORSize([]byte(v.Str))
}

func (v *StringValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (*StringValue) ChildStorables() []atree.Storable {
	return nil
}

// Memory is NOT metered for this value
var ByteArrayStaticType = ConvertSemaArrayTypeToStaticArrayType(nil, sema.ByteArrayType)

// DecodeHex hex-decodes this string and returns an array of UInt8 values
func (v *StringValue) DecodeHex(interpreter *Interpreter, locationRange LocationRange) *ArrayValue {
	bs, err := hex.DecodeString(v.Str)
	if err != nil {
		if err, ok := err.(hex.InvalidByteError); ok {
			panic(InvalidHexByteError{
				LocationRange: locationRange,
				Byte:          byte(err),
			})
		}

		if err == hex.ErrLength {
			panic(InvalidHexLengthError{
				LocationRange: locationRange,
			})
		}

		panic(err)
	}

	i := 0

	return NewArrayValueWithIterator(
		interpreter,
		ByteArrayStaticType,
		common.ZeroAddress,
		uint64(len(bs)),
		func() Value {
			if i >= len(bs) {
				return nil
			}

			value := NewUInt8Value(
				interpreter,
				func() uint8 {
					return bs[i]
				},
			)

			i++

			return value
		},
	)
}

func (v *StringValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v *StringValue) Iterator(_ *Interpreter) ValueIterator {
	return StringValueIterator{
		graphemes: uniseg.NewGraphemes(v.Str),
	}
}

func (v *StringValue) ForEach(
	interpreter *Interpreter,
	_ sema.Type,
	function func(value Value) (resume bool),
	_ LocationRange,
) {
	iterator := v.Iterator(interpreter)
	for {
		value := iterator.Next(interpreter)
		if value == nil {
			return
		}

		if !function(value) {
			return
		}
	}
}

type StringValueIterator struct {
	graphemes *uniseg.Graphemes
}

var _ ValueIterator = StringValueIterator{}

func (i StringValueIterator) Next(_ *Interpreter) Value {
	if !i.graphemes.Next() {
		return nil
	}
	return NewUnmeteredCharacterValue(i.graphemes.Str())
}

// ArrayValue

type ArrayValue struct {
	Type             ArrayStaticType
	semaType         sema.ArrayType
	array            *atree.Array
	isResourceKinded *bool
	elementSize      uint
	isDestroyed      bool
}

type ArrayValueIterator struct {
	atreeIterator *atree.ArrayIterator
}

func (v *ArrayValue) Iterator(_ *Interpreter) ValueIterator {
	arrayIterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return ArrayValueIterator{
		atreeIterator: arrayIterator,
	}
}

var _ ValueIterator = ArrayValueIterator{}

func (i ArrayValueIterator) Next(interpreter *Interpreter) Value {
	atreeValue, err := i.atreeIterator.Next()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if atreeValue == nil {
		return nil
	}

	// atree.Array iterator returns low-level atree.Value,
	// convert to high-level interpreter.Value
	return MustConvertStoredValue(interpreter, atreeValue)
}

func NewArrayValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	arrayType ArrayStaticType,
	address common.Address,
	values ...Value,
) *ArrayValue {

	var index int
	count := len(values)

	return NewArrayValueWithIterator(
		interpreter,
		arrayType,
		address,
		uint64(count),
		func() Value {
			if index >= count {
				return nil
			}

			value := values[index]

			index++

			value = value.Transfer(
				interpreter,
				locationRange,
				atree.Address(address),
				true,
				nil,
				nil,
			)

			return value
		},
	)
}

func NewArrayValueWithIterator(
	interpreter *Interpreter,
	arrayType ArrayStaticType,
	address common.Address,
	countOverestimate uint64,
	values func() Value,
) *ArrayValue {
	interpreter.ReportComputation(common.ComputationKindCreateArrayValue, 1)

	config := interpreter.SharedState.Config

	var v *ArrayValue

	if config.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function,
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			interpreter.reportArrayValueConstructTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() *atree.Array {
		array, err := atree.NewArrayFromBatchData(
			config.Storage,
			atree.Address(address),
			arrayType,
			func() (atree.Value, error) {
				return values(), nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return array
	}
	// must assign to v here for tracing to work properly
	v = newArrayValueFromConstructor(interpreter, arrayType, countOverestimate, constructor)
	return v
}

func ArrayElementSize(staticType ArrayStaticType) uint {
	if staticType == nil {
		return 0
	}
	return staticType.ElementType().elementSize()
}

func newArrayValueFromConstructor(
	gauge common.MemoryGauge,
	staticType ArrayStaticType,
	countOverestimate uint64,
	constructor func() *atree.Array,
) *ArrayValue {

	elementSize := ArrayElementSize(staticType)

	elementUsage, dataSlabs, metaDataSlabs :=
		common.NewAtreeArrayMemoryUsages(countOverestimate, elementSize)
	common.UseMemory(gauge, elementUsage)
	common.UseMemory(gauge, dataSlabs)
	common.UseMemory(gauge, metaDataSlabs)

	return newArrayValueFromAtreeArray(
		gauge,
		staticType,
		elementSize,
		constructor(),
	)
}

func newArrayValueFromAtreeArray(
	gauge common.MemoryGauge,
	staticType ArrayStaticType,
	elementSize uint,
	atreeArray *atree.Array,
) *ArrayValue {

	common.UseMemory(gauge, common.ArrayValueBaseMemoryUsage)

	return &ArrayValue{
		Type:        staticType,
		array:       atreeArray,
		elementSize: elementSize,
	}
}

var _ Value = &ArrayValue{}
var _ atree.Value = &ArrayValue{}
var _ EquatableValue = &ArrayValue{}
var _ ValueIndexableValue = &ArrayValue{}
var _ MemberAccessibleValue = &ArrayValue{}
var _ ReferenceTrackedResourceKindedValue = &ArrayValue{}
var _ IterableValue = &ArrayValue{}

func (*ArrayValue) isValue() {}

func (v *ArrayValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitArrayValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(interpreter, func(element Value) {
		element.Accept(interpreter, visitor)
	})
}

func (v *ArrayValue) Iterate(interpreter *Interpreter, f func(element Value) (resume bool)) {
	v.iterate(interpreter, v.array.Iterate, f)
}

func (v *ArrayValue) IterateLoaded(interpreter *Interpreter, f func(element Value) (resume bool)) {
	v.iterate(interpreter, v.array.IterateLoadedValues, f)
}

func (v *ArrayValue) iterate(
	interpreter *Interpreter,
	atreeIterate func(fn atree.ArrayIterationFunc) error,
	f func(element Value) (resume bool),
) {
	iterate := func() {
		err := atreeIterate(func(element atree.Value) (resume bool, err error) {
			// atree.Array iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			resume = f(MustConvertStoredValue(interpreter, element))

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	if v.IsResourceKinded(interpreter) {
		interpreter.withMutationPrevention(v.StorageID(), iterate)
	} else {
		iterate()
	}
}

func (v *ArrayValue) Walk(interpreter *Interpreter, walkChild func(Value)) {
	v.Iterate(interpreter, func(element Value) (resume bool) {
		walkChild(element)
		return true
	})
}

func (v *ArrayValue) StaticType(_ *Interpreter) StaticType {
	// TODO meter
	return v.Type
}

func (v *ArrayValue) IsImportable(inter *Interpreter) bool {
	importable := true
	v.Iterate(inter, func(element Value) (resume bool) {
		if !element.IsImportable(inter) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return importable
}

func (v *ArrayValue) checkInvalidatedResourceUse(interpreter *Interpreter, locationRange LocationRange) {
	if v.isDestroyed || v.IsStaleResource(interpreter) {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (v *ArrayValue) IsStaleResource(interpreter *Interpreter) bool {
	return v.array == nil && v.IsResourceKinded(interpreter)
}

func (v *ArrayValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyArrayValue, 1)

	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportArrayValueDestroyTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	storageID := v.StorageID()

	interpreter.withResourceDestruction(
		storageID,
		locationRange,
		func() {
			v.Walk(interpreter, func(element Value) {
				maybeDestroy(interpreter, locationRange, element)
			})
		},
	)

	v.isDestroyed = true

	interpreter.invalidateReferencedResources(v, true)

	v.array = nil
}

func (v *ArrayValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *ArrayValue) Concat(interpreter *Interpreter, locationRange LocationRange, other *ArrayValue) Value {

	first := true

	firstIterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	secondIterator, err := other.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	elementType := v.Type.ElementType()

	return NewArrayValueWithIterator(
		interpreter,
		v.Type,
		common.ZeroAddress,
		v.array.Count()+other.array.Count(),
		func() Value {

			var value Value

			if first {
				atreeValue, err := firstIterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue == nil {
					first = false
				} else {
					value = MustConvertStoredValue(interpreter, atreeValue)
				}
			}

			if !first {
				atreeValue, err := secondIterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				if atreeValue != nil {
					value = MustConvertStoredValue(interpreter, atreeValue)

					interpreter.checkContainerMutation(elementType, value, locationRange)
				}
			}

			if value == nil {
				return nil
			}

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
			)
		},
	)
}

func (v *ArrayValue) GetKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	index := key.(NumberValue).ToInt(locationRange)
	return v.Get(interpreter, locationRange, index)
}

func (v *ArrayValue) handleIndexOutOfBoundsError(err error, index int, locationRange LocationRange) {
	var indexOutOfBoundsError *atree.IndexOutOfBoundsError
	if goerrors.As(err, &indexOutOfBoundsError) {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}
}

func (v *ArrayValue) Get(interpreter *Interpreter, locationRange LocationRange, index int) Value {

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Get function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	storable, err := v.array.Get(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}

	return StoredValue(interpreter, storable, interpreter.Storage())
}

func (v *ArrayValue) SetKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value) {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	index := key.(NumberValue).ToInt(locationRange)
	v.Set(interpreter, locationRange, index, value)
}

func (v *ArrayValue) Set(interpreter *Interpreter, locationRange LocationRange, index int, element Value) {

	interpreter.validateMutation(v.StorageID(), locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Set function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	common.UseMemory(interpreter, common.AtreeArrayElementOverhead)

	element = element.Transfer(
		interpreter,
		locationRange,
		v.array.Address(),
		true,
		nil,
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

	existingStorable, err := v.array.Set(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)

	existingValue := StoredValue(interpreter, existingStorable, interpreter.Storage())

	existingValue.DeepRemove(interpreter)

	interpreter.RemoveReferencedSlab(existingStorable)
}

func (v *ArrayValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *ArrayValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences)
}

func (v *ArrayValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	// if n > 0:
	// len = open-bracket + close-bracket + ((n-1) comma+space)
	//     = 2 + 2n - 2
	//     = 2n
	// Always +2 to include empty array case (over estimate).
	// Each elements' string value is metered individually.
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(v.Count()*2+2))

	values := make([]string, v.Count())

	i := 0

	_ = v.array.Iterate(func(element atree.Value) (resume bool, err error) {
		// ok to not meter anything created as part of this iteration, since we will discard the result
		// upon creating the string
		values[i] = MustConvertUnmeteredStoredValue(element).MeteredString(memoryGauge, seenReferences)
		i++
		return true, nil
	})

	return format.Array(values)
}

func (v *ArrayValue) Append(interpreter *Interpreter, locationRange LocationRange, element Value) {

	interpreter.validateMutation(v.StorageID(), locationRange)

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(
		v.array.Count(),
		v.elementSize,
		true,
	)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)
	common.UseMemory(interpreter, common.AtreeArrayElementOverhead)

	interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	element = element.Transfer(
		interpreter,
		locationRange,
		v.array.Address(),
		true,
		nil,
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

	err := v.array.Append(element)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) AppendAll(interpreter *Interpreter, locationRange LocationRange, other *ArrayValue) {
	other.Walk(interpreter, func(value Value) {
		v.Append(interpreter, locationRange, value)
	})
}

func (v *ArrayValue) InsertKey(interpreter *Interpreter, locationRange LocationRange, key Value, value Value) {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	index := key.(NumberValue).ToInt(locationRange)
	v.Insert(interpreter, locationRange, index, value)
}

func (v *ArrayValue) Insert(interpreter *Interpreter, locationRange LocationRange, index int, element Value) {

	interpreter.validateMutation(v.StorageID(), locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Insert function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(
		v.array.Count(),
		v.elementSize,
		true,
	)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)
	common.UseMemory(interpreter, common.AtreeArrayElementOverhead)

	interpreter.checkContainerMutation(v.Type.ElementType(), element, locationRange)

	element = element.Transfer(
		interpreter,
		locationRange,
		v.array.Address(),
		true,
		nil,
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

	err := v.array.Insert(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) RemoveKey(interpreter *Interpreter, locationRange LocationRange, key Value) Value {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	index := key.(NumberValue).ToInt(locationRange)
	return v.Remove(interpreter, locationRange, index)
}

func (v *ArrayValue) Remove(interpreter *Interpreter, locationRange LocationRange, index int) Value {

	interpreter.validateMutation(v.StorageID(), locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.Remove function will check the upper bound and report an atree.IndexOutOfBoundsError

	if index < 0 {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	storable, err := v.array.Remove(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, locationRange)

		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)

	value := StoredValue(interpreter, storable, interpreter.Storage())

	return value.Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		true,
		storable,
		nil,
	)
}

func (v *ArrayValue) RemoveFirst(interpreter *Interpreter, locationRange LocationRange) Value {
	return v.Remove(interpreter, locationRange, 0)
}

func (v *ArrayValue) RemoveLast(interpreter *Interpreter, locationRange LocationRange) Value {
	return v.Remove(interpreter, locationRange, v.Count()-1)
}

func (v *ArrayValue) FirstIndex(interpreter *Interpreter, locationRange LocationRange, needleValue Value) OptionalValue {

	needleEquatable, ok := needleValue.(EquatableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var counter int64
	var result bool
	v.Iterate(interpreter, func(element Value) (resume bool) {
		if needleEquatable.Equal(interpreter, locationRange, element) {
			result = true
			// stop iteration
			return false
		}
		counter++
		// continue iteration
		return true
	})

	if result {
		value := NewIntValueFromInt64(interpreter, counter)
		return NewSomeValueNonCopying(interpreter, value)
	}
	return NilOptionalValue
}

func (v *ArrayValue) Contains(
	interpreter *Interpreter,
	locationRange LocationRange,
	needleValue Value,
) BoolValue {

	needleEquatable, ok := needleValue.(EquatableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var result bool
	v.Iterate(interpreter, func(element Value) (resume bool) {
		if needleEquatable.Equal(interpreter, locationRange, element) {
			result = true
			// stop iteration
			return false
		}
		// continue iteration
		return true
	})

	return AsBoolValue(result)
}

func (v *ArrayValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	switch name {
	case "length":
		return NewIntValueFromInt64(interpreter, int64(v.Count()))

	case "append":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayAppendFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				v.Append(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
				return Void
			},
		)

	case "appendAll":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayAppendAllFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				otherArray, ok := invocation.Arguments[0].(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				v.AppendAll(
					invocation.Interpreter,
					invocation.LocationRange,
					otherArray,
				)
				return Void
			},
		)

	case "concat":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayConcatFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				otherArray, ok := invocation.Arguments[0].(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.Concat(
					invocation.Interpreter,
					invocation.LocationRange,
					otherArray,
				)
			},
		)

	case "insert":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayInsertFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				indexValue, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				index := indexValue.ToInt(locationRange)

				element := invocation.Arguments[1]

				v.Insert(
					invocation.Interpreter,
					invocation.LocationRange,
					index,
					element,
				)
				return Void
			},
		)

	case "remove":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayRemoveFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				indexValue, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				index := indexValue.ToInt(locationRange)

				return v.Remove(
					invocation.Interpreter,
					invocation.LocationRange,
					index,
				)
			},
		)

	case "removeFirst":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayRemoveFirstFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.RemoveFirst(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case "removeLast":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayRemoveLastFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.RemoveLast(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case "firstIndex":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayFirstIndexFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.FirstIndex(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)

	case "contains":
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayContainsFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				return v.Contains(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)

	case "slice":
		return NewHostFunctionValue(
			interpreter,
			sema.ArraySliceFunctionType(
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				from, ok := invocation.Arguments[0].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				to, ok := invocation.Arguments[1].(IntValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Slice(
					invocation.Interpreter,
					from,
					to,
					invocation.LocationRange,
				)
			},
		)

	case sema.ArrayTypeReverseFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayReverseFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				return v.Reverse(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case sema.ArrayTypeFilterFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayFilterFunctionType(
				interpreter,
				v.SemaType(interpreter).ElementType(false),
			),
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Filter(
					interpreter,
					invocation.LocationRange,
					funcArgument,
				)
			},
		)

	case sema.ArrayTypeMapFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ArrayMapFunctionType(
				interpreter,
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				transformFunctionType, ok := invocation.ArgumentTypes[0].(*sema.FunctionType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Map(
					interpreter,
					invocation.LocationRange,
					funcArgument,
					transformFunctionType,
				)
			},
		)
	}

	return nil
}

func (v *ArrayValue) RemoveMember(interpreter *Interpreter, locationRange LocationRange, _ string) Value {
	// Arrays have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) SetMember(interpreter *Interpreter, locationRange LocationRange, _ string, _ Value) bool {
	// Arrays have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) Count() int {
	return int(v.array.Count())
}

func (v *ArrayValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	config := interpreter.SharedState.Config

	count := v.Count()

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			interpreter.reportArrayValueConformsToStaticTypeTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	var elementType StaticType
	switch staticType := v.StaticType(interpreter).(type) {
	case *ConstantSizedStaticType:
		elementType = staticType.ElementType()
		if v.Count() != int(staticType.Size) {
			return false
		}
	case *VariableSizedStaticType:
		elementType = staticType.ElementType()
	default:
		return false
	}

	var elementMismatch bool

	v.Iterate(interpreter, func(element Value) (resume bool) {

		if !interpreter.IsSubType(element.StaticType(interpreter), elementType) {
			elementMismatch = true
			// stop iteration
			return false
		}

		if !element.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			elementMismatch = true
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return !elementMismatch
}

func (v *ArrayValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherArray, ok := other.(*ArrayValue)
	if !ok {
		return false
	}

	count := v.Count()

	if count != otherArray.Count() {
		return false
	}

	if v.Type == nil {
		if otherArray.Type != nil {
			return false
		}
	} else if otherArray.Type == nil ||
		!v.Type.Equal(otherArray.Type) {

		return false
	}

	for i := 0; i < count; i++ {
		value := v.Get(interpreter, locationRange, i)
		otherValue := otherArray.Get(interpreter, locationRange, i)

		equatableValue, ok := value.(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, locationRange, otherValue) {
			return false
		}
	}

	return true
}

func (v *ArrayValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return v.array.Storable(storage, address, maxInlineSize)
}

func (v *ArrayValue) IsReferenceTrackedResourceKindedValue() {}

func (v *ArrayValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.StorageID]struct{},
) Value {

	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	interpreter.ReportComputation(
		common.ComputationKindTransferArrayValue,
		uint(v.Count()),
	)

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportArrayValueTransferTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	currentStorageID := v.StorageID()

	if preventTransfer == nil {
		preventTransfer = map[atree.StorageID]struct{}{}
	} else if _, ok := preventTransfer[currentStorageID]; ok {
		panic(RecursiveTransferError{
			LocationRange: locationRange,
		})
	}
	preventTransfer[currentStorageID] = struct{}{}
	defer delete(preventTransfer, currentStorageID)

	array := v.array

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		iterator, err := v.array.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementUsage, dataSlabs, metaDataSlabs := common.NewAtreeArrayMemoryUsages(
			v.array.Count(),
			v.elementSize,
		)
		common.UseMemory(interpreter, elementUsage)
		common.UseMemory(interpreter, dataSlabs)
		common.UseMemory(interpreter, metaDataSlabs)

		array, err = atree.NewArrayFromBatchData(
			config.Storage,
			address,
			v.array.Type(),
			func() (atree.Value, error) {
				value, err := iterator.Next()
				if err != nil {
					return nil, err
				}
				if value == nil {
					return nil, nil
				}

				element := MustConvertStoredValue(interpreter, value).
					Transfer(interpreter, locationRange, address, remove, nil, preventTransfer)

				return element, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.array.PopIterate(interpreter.RemoveReferencedSlab)
			if err != nil {
				panic(errors.NewExternalError(err))
			}
			interpreter.maybeValidateAtreeValue(v.array)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
		// to be transferred/moved again (see beginning of this function)

		interpreter.invalidateReferencedResources(v, false)

		v.array = nil
	}

	res := newArrayValueFromAtreeArray(
		interpreter,
		v.Type,
		v.elementSize,
		array,
	)

	res.semaType = v.semaType
	res.isResourceKinded = v.isResourceKinded
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *ArrayValue) Clone(interpreter *Interpreter) Value {
	config := interpreter.SharedState.Config

	array := newArrayValueFromConstructor(
		interpreter,
		v.Type,
		v.array.Count(),
		func() *atree.Array {
			iterator, err := v.array.Iterator()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			array, err := atree.NewArrayFromBatchData(
				config.Storage,
				v.StorageAddress(),
				v.array.Type(),
				func() (atree.Value, error) {
					value, err := iterator.Next()
					if err != nil {
						return nil, err
					}
					if value == nil {
						return nil, nil
					}

					element := MustConvertStoredValue(interpreter, value).
						Clone(interpreter)

					return element, nil
				},
			)
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			return array
		},
	)

	array.semaType = v.semaType
	array.isResourceKinded = v.isResourceKinded
	array.isDestroyed = v.isDestroyed

	return array
}

func (v *ArrayValue) DeepRemove(interpreter *Interpreter) {
	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportArrayValueDeepRemoveTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.array.Storage

	err := v.array.PopIterate(func(storable atree.Storable) {
		value := StoredValue(interpreter, storable, storage)
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(storable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) StorageID() atree.StorageID {
	return v.array.StorageID()
}

func (v *ArrayValue) StorageAddress() atree.Address {
	return v.array.Address()
}

func (v *ArrayValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

func (v *ArrayValue) SemaType(interpreter *Interpreter) sema.ArrayType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = interpreter.MustConvertStaticToSemaType(v.Type).(sema.ArrayType)
	}
	return v.semaType
}

func (v *ArrayValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *ArrayValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(interpreter).IsResourceType()
		v.isResourceKinded = &isResourceKinded
	}
	return *v.isResourceKinded
}

func (v *ArrayValue) Slice(
	interpreter *Interpreter,
	from IntValue,
	to IntValue,
	locationRange LocationRange,
) Value {
	fromIndex := from.ToInt(locationRange)
	toIndex := to.ToInt(locationRange)

	// We only need to check the lower bound before converting from `int` (signed) to `uint64` (unsigned).
	// atree's Array.RangeIterator function will check the upper bound and report an atree.SliceOutOfBoundsError

	if fromIndex < 0 || toIndex < 0 {
		panic(ArraySliceIndicesError{
			FromIndex:     fromIndex,
			UpToIndex:     toIndex,
			Size:          v.Count(),
			LocationRange: locationRange,
		})
	}

	iterator, err := v.array.RangeIterator(uint64(fromIndex), uint64(toIndex))
	if err != nil {

		var sliceOutOfBoundsError *atree.SliceOutOfBoundsError
		if goerrors.As(err, &sliceOutOfBoundsError) {
			panic(ArraySliceIndicesError{
				FromIndex:     fromIndex,
				UpToIndex:     toIndex,
				Size:          v.Count(),
				LocationRange: locationRange,
			})
		}

		var invalidSliceIndexError *atree.InvalidSliceIndexError
		if goerrors.As(err, &invalidSliceIndexError) {
			panic(InvalidSliceIndexError{
				FromIndex:     fromIndex,
				UpToIndex:     toIndex,
				LocationRange: locationRange,
			})
		}

		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		interpreter,
		NewVariableSizedStaticType(interpreter, v.Type.ElementType()),
		common.ZeroAddress,
		uint64(toIndex-fromIndex),
		func() Value {

			var value Value

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue != nil {
				value = MustConvertStoredValue(interpreter, atreeValue)
			}

			if value == nil {
				return nil
			}

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
			)
		},
	)
}

func (v *ArrayValue) Reverse(
	interpreter *Interpreter,
	locationRange LocationRange,
) Value {
	count := v.Count()
	index := count - 1

	return NewArrayValueWithIterator(
		interpreter,
		v.Type,
		common.ZeroAddress,
		uint64(count),
		func() Value {
			if index < 0 {
				return nil
			}

			// Meter computation for iterating the array.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			value := v.Get(interpreter, locationRange, index)
			index--

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
			)
		},
	)
}

func (v *ArrayValue) Filter(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
) Value {

	elementTypeSlice := []sema.Type{v.semaType.ElementType(false)}
	iterationInvocation := func(arrayElement Value) Invocation {
		invocation := NewInvocation(
			interpreter,
			nil,
			nil,
			nil,
			[]Value{arrayElement},
			elementTypeSlice,
			nil,
			locationRange,
		)
		return invocation
	}

	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		interpreter,
		NewVariableSizedStaticType(interpreter, v.Type.ElementType()),
		common.ZeroAddress,
		uint64(v.Count()), // worst case estimation.
		func() Value {

			var value Value

			for {
				// Meter computation for iterating the array.
				interpreter.ReportComputation(common.ComputationKindLoop, 1)

				atreeValue, err := iterator.Next()
				if err != nil {
					panic(errors.NewExternalError(err))
				}

				// Also handles the end of array case since iterator.Next() returns nil for that.
				if atreeValue == nil {
					return nil
				}

				value = MustConvertStoredValue(interpreter, atreeValue)
				if value == nil {
					return nil
				}

				shouldInclude, ok := procedure.invoke(iterationInvocation(value)).(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				// We found the next entry of the filtered array.
				if shouldInclude {
					break
				}
			}

			return value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
			)
		},
	)
}

func (v *ArrayValue) Map(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
	transformFunctionType *sema.FunctionType,
) Value {

	elementTypeSlice := []sema.Type{v.semaType.ElementType(false)}
	iterationInvocation := func(arrayElement Value) Invocation {
		return NewInvocation(
			interpreter,
			nil,
			nil,
			nil,
			[]Value{arrayElement},
			elementTypeSlice,
			nil,
			locationRange,
		)
	}

	procedureStaticType, ok := ConvertSemaToStaticType(interpreter, transformFunctionType).(FunctionStaticType)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	returnType := procedureStaticType.ReturnType(interpreter)

	var returnArrayStaticType ArrayStaticType
	switch v.Type.(type) {
	case *VariableSizedStaticType:
		returnArrayStaticType = NewVariableSizedStaticType(
			interpreter,
			returnType,
		)
	case *ConstantSizedStaticType:
		returnArrayStaticType = NewConstantSizedStaticType(
			interpreter,
			returnType,
			int64(v.Count()),
		)
	default:
		panic(errors.NewUnreachableError())
	}

	iterator, err := v.array.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return NewArrayValueWithIterator(
		interpreter,
		returnArrayStaticType,
		common.ZeroAddress,
		uint64(v.Count()),
		func() Value {

			// Meter computation for iterating the array.
			interpreter.ReportComputation(common.ComputationKindLoop, 1)

			atreeValue, err := iterator.Next()
			if err != nil {
				panic(errors.NewExternalError(err))
			}

			if atreeValue == nil {
				return nil
			}

			value := MustConvertStoredValue(interpreter, atreeValue)

			mappedValue := procedure.invoke(iterationInvocation(value))
			return mappedValue.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
			)
		},
	)
}

func (v *ArrayValue) ForEach(
	interpreter *Interpreter,
	_ sema.Type,
	function func(value Value) (resume bool),
	_ LocationRange,
) {
	v.Iterate(interpreter, function)
}

// NumberValue
type NumberValue interface {
	ComparableValue
	ToInt(locationRange LocationRange) int
	Negate(*Interpreter, LocationRange) NumberValue
	Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue
	ToBigEndianBytes() []byte
}

func getNumberValueMember(interpreter *Interpreter, v NumberValue, name string, typ sema.Type, locationRange LocationRange) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ToStringFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter
				memoryUsage := common.NewStringMemoryUsage(
					OverEstimateNumberStringLength(interpreter, v),
				)
				return NewStringValue(
					interpreter,
					memoryUsage,
					func() string {
						return v.String()
					},
				)
			},
		)

	case sema.ToBigEndianBytesFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ToBigEndianBytesFunctionType,
			func(invocation Invocation) Value {
				return ByteSliceToByteArrayValue(
					invocation.Interpreter,
					v.ToBigEndianBytes(),
				)
			},
		)

	case sema.NumericTypeSaturatingAddFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingPlus(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingSubtractFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingMinus(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingMultiplyFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingMul(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)

	case sema.NumericTypeSaturatingDivideFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.SaturatingArithmeticTypeFunctionTypes[typ],
			func(invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(NumberValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.SaturatingDiv(
					invocation.Interpreter,
					other,
					locationRange,
				)
			},
		)
	}

	return nil
}

type IntegerValue interface {
	NumberValue
	BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
	BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue
}

// BigNumberValue is a number value with an integer value outside the range of int64
type BigNumberValue interface {
	NumberValue
	ByteLength() int
	ToBigInt(memoryGauge common.MemoryGauge) *big.Int
}

// Int

type IntValue struct {
	BigInt *big.Int
}

const int64Size = int(unsafe.Sizeof(int64(0)))

var int64BigIntMemoryUsage = common.NewBigIntMemoryUsage(int64Size)

func NewIntValueFromInt64(memoryGauge common.MemoryGauge, value int64) IntValue {
	return NewIntValueFromBigInt(
		memoryGauge,
		int64BigIntMemoryUsage,
		func() *big.Int {
			return big.NewInt(value)
		},
	)
}

func NewUnmeteredIntValueFromInt64(value int64) IntValue {
	return NewUnmeteredIntValueFromBigInt(big.NewInt(value))
}

func NewIntValueFromBigInt(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) IntValue {
	common.UseMemory(memoryGauge, memoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredIntValueFromBigInt(value)
}

func NewUnmeteredIntValueFromBigInt(value *big.Int) IntValue {
	return IntValue{
		BigInt: value,
	}
}

func ConvertInt(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) IntValue {
	switch value := value.(type) {
	case BigNumberValue:
		return NewUnmeteredIntValueFromBigInt(
			value.ToBigInt(memoryGauge),
		)

	case NumberValue:
		return NewIntValueFromInt64(
			memoryGauge,
			int64(value.ToInt(locationRange)),
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

var _ Value = IntValue{}
var _ atree.Storable = IntValue{}
var _ NumberValue = IntValue{}
var _ IntegerValue = IntValue{}
var _ EquatableValue = IntValue{}
var _ ComparableValue = IntValue{}
var _ HashableValue = IntValue{}
var _ MemberAccessibleValue = IntValue{}

func (IntValue) isValue() {}

func (v IntValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitIntValue(interpreter, v)
}

func (IntValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (IntValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt)
}

func (IntValue) IsImportable(_ *Interpreter) bool {
	return true
}

func (v IntValue) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v IntValue) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v IntValue) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v IntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v IntValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v IntValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v IntValue) Negate(interpreter *Interpreter, _ LocationRange) NumberValue {
	return NewIntValueFromBigInt(
		interpreter,
		common.NewNegateBigIntMemoryUsage(v.BigInt),
		func() *big.Int {
			return new(big.Int).Neg(v.BigInt)
		},
	)
}

func (v IntValue) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewPlusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Add(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Plus(interpreter, other, locationRange)
}

func (v IntValue) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewMinusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Sub(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Minus(interpreter, other, locationRange)
}

func (v IntValue) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewModBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			// INT33-C
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewMulBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Mul(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Mul(interpreter, other, locationRange)
}

func (v IntValue) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewDivBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			// INT33-C
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v IntValue) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v IntValue) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v IntValue) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)

}

func (v IntValue) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v IntValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(IntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt (1 byte)
// - big int encoded in big-endian (n bytes)
func (v IntValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := SignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeInt)
	copy(buffer[1:], b)
	return buffer
}

func (v IntValue) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewBitwiseOrBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewBitwiseXorBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewBitwiseAndBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)
}

func (v IntValue) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewBitwiseLeftShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v IntValue) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewIntValueFromBigInt(
		interpreter,
		common.NewBitwiseRightShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v IntValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.IntType, locationRange)
}

func (IntValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (IntValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v IntValue) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v IntValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v IntValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (IntValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (IntValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v IntValue) Transfer(
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

func (v IntValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredIntValueFromBigInt(v.BigInt)
}

func (IntValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v IntValue) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v IntValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (IntValue) ChildStorables() []atree.Storable {
	return nil
}

// Int8Value

type Int8Value int8

const int8Size = int(unsafe.Sizeof(Int8Value(0)))

var Int8MemoryUsage = common.NewNumberMemoryUsage(int8Size)

func NewInt8Value(gauge common.MemoryGauge, valueGetter func() int8) Int8Value {
	common.UseMemory(gauge, Int8MemoryUsage)

	return NewUnmeteredInt8Value(valueGetter())
}

func NewUnmeteredInt8Value(value int8) Int8Value {
	return Int8Value(value)
}

var _ Value = Int8Value(0)
var _ atree.Storable = Int8Value(0)
var _ NumberValue = Int8Value(0)
var _ IntegerValue = Int8Value(0)
var _ EquatableValue = Int8Value(0)
var _ ComparableValue = Int8Value(0)
var _ HashableValue = Int8Value(0)

func (Int8Value) isValue() {}

func (v Int8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt8Value(interpreter, v)
}

func (Int8Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int8Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt8)
}

func (Int8Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int8Value) String() string {
	return format.Int(int64(v))
}

func (v Int8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int8Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int8Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Int8Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt8 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(-v)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt8 - o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v < (math.MinInt8 - o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v + o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt8 - o)) {
			return math.MaxInt8
		} else if (o < 0) && (v < (math.MinInt8 - o)) {
			return math.MinInt8
		}
		return int8(v + o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt8 + o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v > (math.MaxInt8 + o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v - o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt8 + o)) {
			return math.MinInt8
		} else if (o < 0) && (v > (math.MaxInt8 + o)) {
			return math.MaxInt8
		}
		return int8(v - o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v % o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt8 / o) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt8 / v) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt8 / o) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt8 / v)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}

	valueGetter := func() int8 {
		return int8(v * o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		// INT32-C
		if v > 0 {
			if o > 0 {
				// positive * positive = positive. overflow?
				if v > (math.MaxInt8 / o) {
					return math.MaxInt8
				}
			} else {
				// positive * negative = negative. underflow?
				if o < (math.MinInt8 / v) {
					return math.MinInt8
				}
			}
		} else {
			if o > 0 {
				// negative * positive = negative. underflow?
				if v < (math.MinInt8 / o) {
					return math.MinInt8
				}
			} else {
				// negative * negative = positive. overflow?
				if (v != 0) && (o < (math.MaxInt8 / v)) {
					return math.MaxInt8
				}
			}
		}

		return int8(v * o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	} else if (v == math.MinInt8) && (o == -1) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v / o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		// INT33-C
		// https://golang.org/ref/spec#Integer_operators
		if o == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		} else if (v == math.MinInt8) && (o == -1) {
			return math.MaxInt8
		}
		return int8(v / o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Int8Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Int8Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Int8Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Int8Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt8, ok := other.(Int8Value)
	if !ok {
		return false
	}
	return v == otherInt8
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt8 (1 byte)
// - int8 value (1 byte)
func (v Int8Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertInt8(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int8Value {
	converter := func() int8 {

		switch value := value.(type) {
		case BigNumberValue:
			v := value.ToBigInt(memoryGauge)
			if v.Cmp(sema.Int8TypeMaxInt) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Cmp(sema.Int8TypeMinInt) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int8(v.Int64())

		case NumberValue:
			v := value.ToInt(locationRange)
			if v > math.MaxInt8 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v < math.MinInt8 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int8(v)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return NewInt8Value(memoryGauge, converter)
}

func (v Int8Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v | o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v ^ o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v & o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v << o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int8 {
		return int8(v >> o)
	}

	return NewInt8Value(interpreter, valueGetter)
}

func (v Int8Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int8Type, locationRange)
}

func (Int8Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int8Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Int8Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int8Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int8Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int8Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int8Value) Transfer(
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

func (v Int8Value) Clone(_ *Interpreter) Value {
	return v
}

func (Int8Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int8Value) ByteSize() uint32 {
	return cborTagSize + getIntCBORSize(int64(v))
}

func (v Int8Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int8Value) ChildStorables() []atree.Storable {
	return nil
}

// Int16Value

type Int16Value int16

const int16Size = int(unsafe.Sizeof(Int16Value(0)))

var Int16MemoryUsage = common.NewNumberMemoryUsage(int16Size)

func NewInt16Value(gauge common.MemoryGauge, valueGetter func() int16) Int16Value {
	common.UseMemory(gauge, Int16MemoryUsage)

	return NewUnmeteredInt16Value(valueGetter())
}

func NewUnmeteredInt16Value(value int16) Int16Value {
	return Int16Value(value)
}

var _ Value = Int16Value(0)
var _ atree.Storable = Int16Value(0)
var _ NumberValue = Int16Value(0)
var _ IntegerValue = Int16Value(0)
var _ EquatableValue = Int16Value(0)
var _ ComparableValue = Int16Value(0)
var _ HashableValue = Int16Value(0)
var _ MemberAccessibleValue = Int16Value(0)

func (Int16Value) isValue() {}

func (v Int16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt16Value(interpreter, v)
}

func (Int16Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int16Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt16)
}

func (Int16Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int16Value) String() string {
	return format.Int(int64(v))
}

func (v Int16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int16Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int16Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Int16Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt16 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(-v)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt16 - o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v < (math.MinInt16 - o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v + o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt16 - o)) {
			return math.MaxInt16
		} else if (o < 0) && (v < (math.MinInt16 - o)) {
			return math.MinInt16
		}
		return int16(v + o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt16 + o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v > (math.MaxInt16 + o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v - o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt16 + o)) {
			return math.MinInt16
		} else if (o < 0) && (v > (math.MaxInt16 + o)) {
			return math.MaxInt16
		}
		return int16(v - o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v % o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt16 / o) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt16 / v) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt16 / o) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt16 / v)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}

	valueGetter := func() int16 {
		return int16(v * o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT32-C
		if v > 0 {
			if o > 0 {
				// positive * positive = positive. overflow?
				if v > (math.MaxInt16 / o) {
					return math.MaxInt16
				}
			} else {
				// positive * negative = negative. underflow?
				if o < (math.MinInt16 / v) {
					return math.MinInt16
				}
			}
		} else {
			if o > 0 {
				// negative * positive = negative. underflow?
				if v < (math.MinInt16 / o) {
					return math.MinInt16
				}
			} else {
				// negative * negative = positive. overflow?
				if (v != 0) && (o < (math.MaxInt16 / v)) {
					return math.MaxInt16
				}
			}
		}
		return int16(v * o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	} else if (v == math.MinInt16) && (o == -1) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v / o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		// INT33-C
		// https://golang.org/ref/spec#Integer_operators
		if o == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		} else if (v == math.MinInt16) && (o == -1) {
			return math.MaxInt16
		}
		return int16(v / o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Int16Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Int16Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Int16Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Int16Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt16, ok := other.(Int16Value)
	if !ok {
		return false
	}
	return v == otherInt16
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt16 (1 byte)
// - int16 value encoded in big-endian (2 bytes)
func (v Int16Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertInt16(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int16Value {
	converter := func() int16 {

		switch value := value.(type) {
		case BigNumberValue:
			v := value.ToBigInt(memoryGauge)
			if v.Cmp(sema.Int16TypeMaxInt) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Cmp(sema.Int16TypeMinInt) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int16(v.Int64())

		case NumberValue:
			v := value.ToInt(locationRange)
			if v > math.MaxInt16 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v < math.MinInt16 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int16(v)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return NewInt16Value(memoryGauge, converter)
}

func (v Int16Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v | o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v ^ o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v & o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v << o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int16 {
		return int16(v >> o)
	}

	return NewInt16Value(interpreter, valueGetter)
}

func (v Int16Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int16Type, locationRange)
}

func (Int16Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int16Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Int16Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int16Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int16Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int16Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int16Value) Transfer(
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

func (v Int16Value) Clone(_ *Interpreter) Value {
	return v
}

func (Int16Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int16Value) ByteSize() uint32 {
	return cborTagSize + getIntCBORSize(int64(v))
}

func (v Int16Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int16Value) ChildStorables() []atree.Storable {
	return nil
}

// Int32Value

type Int32Value int32

const int32Size = int(unsafe.Sizeof(Int32Value(0)))

var Int32MemoryUsage = common.NewNumberMemoryUsage(int32Size)

func NewInt32Value(gauge common.MemoryGauge, valueGetter func() int32) Int32Value {
	common.UseMemory(gauge, Int32MemoryUsage)

	return NewUnmeteredInt32Value(valueGetter())
}

func NewUnmeteredInt32Value(value int32) Int32Value {
	return Int32Value(value)
}

var _ Value = Int32Value(0)
var _ atree.Storable = Int32Value(0)
var _ NumberValue = Int32Value(0)
var _ IntegerValue = Int32Value(0)
var _ EquatableValue = Int32Value(0)
var _ ComparableValue = Int32Value(0)
var _ HashableValue = Int32Value(0)
var _ MemberAccessibleValue = Int32Value(0)

func (Int32Value) isValue() {}

func (v Int32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt32Value(interpreter, v)
}

func (Int32Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int32Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt32)
}

func (Int32Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int32Value) String() string {
	return format.Int(int64(v))
}

func (v Int32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int32Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int32Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Int32Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt32 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(-v)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt32 - o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v < (math.MinInt32 - o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v + o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt32 - o)) {
			return math.MaxInt32
		} else if (o < 0) && (v < (math.MinInt32 - o)) {
			return math.MinInt32
		}
		return int32(v + o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt32 + o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v > (math.MaxInt32 + o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v - o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt32 + o)) {
			return math.MinInt32
		} else if (o < 0) && (v > (math.MaxInt32 + o)) {
			return math.MaxInt32
		}
		return int32(v - o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v % o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt32 / o) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt32 / v) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt32 / o) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt32 / v)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}

	valueGetter := func() int32 {
		return int32(v * o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		// INT32-C
		if v > 0 {
			if o > 0 {
				// positive * positive = positive. overflow?
				if v > (math.MaxInt32 / o) {
					return math.MaxInt32
				}
			} else {
				// positive * negative = negative. underflow?
				if o < (math.MinInt32 / v) {
					return math.MinInt32
				}
			}
		} else {
			if o > 0 {
				// negative * positive = negative. underflow?
				if v < (math.MinInt32 / o) {
					return math.MinInt32
				}
			} else {
				// negative * negative = positive. overflow?
				if (v != 0) && (o < (math.MaxInt32 / v)) {
					return math.MaxInt32
				}
			}
		}
		return int32(v * o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	} else if (v == math.MinInt32) && (o == -1) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v / o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		// INT33-C
		// https://golang.org/ref/spec#Integer_operators
		if o == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		} else if (v == math.MinInt32) && (o == -1) {
			return math.MaxInt32
		}

		return int32(v / o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Int32Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Int32Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Int32Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Int32Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt32, ok := other.(Int32Value)
	if !ok {
		return false
	}
	return v == otherInt32
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt32 (1 byte)
// - int32 value encoded in big-endian (4 bytes)
func (v Int32Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertInt32(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int32Value {
	converter := func() int32 {
		switch value := value.(type) {
		case BigNumberValue:
			v := value.ToBigInt(memoryGauge)
			if v.Cmp(sema.Int32TypeMaxInt) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Cmp(sema.Int32TypeMinInt) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int32(v.Int64())

		case NumberValue:
			v := value.ToInt(locationRange)
			if v > math.MaxInt32 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v < math.MinInt32 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return int32(v)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return NewInt32Value(memoryGauge, converter)
}

func (v Int32Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v | o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v ^ o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v & o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v << o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int32 {
		return int32(v >> o)
	}

	return NewInt32Value(interpreter, valueGetter)
}

func (v Int32Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int32Type, locationRange)
}

func (Int32Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int32Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Int32Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int32Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int32Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int32Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int32Value) Transfer(
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

func (v Int32Value) Clone(_ *Interpreter) Value {
	return v
}

func (Int32Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int32Value) ByteSize() uint32 {
	return cborTagSize + getIntCBORSize(int64(v))
}

func (v Int32Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int32Value) ChildStorables() []atree.Storable {
	return nil
}

// Int64Value

type Int64Value int64

var Int64MemoryUsage = common.NewNumberMemoryUsage(int64Size)

func NewInt64Value(gauge common.MemoryGauge, valueGetter func() int64) Int64Value {
	common.UseMemory(gauge, Int64MemoryUsage)

	return NewUnmeteredInt64Value(valueGetter())
}

func NewUnmeteredInt64Value(value int64) Int64Value {
	return Int64Value(value)
}

var _ Value = Int64Value(0)
var _ atree.Storable = Int64Value(0)
var _ NumberValue = Int64Value(0)
var _ IntegerValue = Int64Value(0)
var _ EquatableValue = Int64Value(0)
var _ ComparableValue = Int64Value(0)
var _ HashableValue = Int64Value(0)
var _ MemberAccessibleValue = Int64Value(0)

func (Int64Value) isValue() {}

func (v Int64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt64Value(interpreter, v)
}

func (Int64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt64)
}

func (Int64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int64Value) String() string {
	return format.Int(int64(v))
}

func (v Int64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int64Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Int64Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(-v)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func safeAddInt64(a, b int64, locationRange LocationRange) int64 {
	// INT32-C
	if (b > 0) && (a > (math.MaxInt64 - b)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (b < 0) && (a < (math.MinInt64 - b)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	return a + b
}

func (v Int64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return safeAddInt64(int64(v), int64(o), locationRange)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt64 - o)) {
			return math.MaxInt64
		} else if (o < 0) && (v < (math.MinInt64 - o)) {
			return math.MinInt64
		}
		return int64(v + o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v - o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt64 + o)) {
			return math.MinInt64
		} else if (o < 0) && (v > (math.MaxInt64 + o)) {
			return math.MaxInt64
		}
		return int64(v - o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v % o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt64 / o) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt64 / v) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt64 / o) {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt64 / v)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
		}
	}

	valueGetter := func() int64 {
		return int64(v * o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if v > 0 {
			if o > 0 {
				// positive * positive = positive. overflow?
				if v > (math.MaxInt64 / o) {
					return math.MaxInt64
				}
			} else {
				// positive * negative = negative. underflow?
				if o < (math.MinInt64 / v) {
					return math.MinInt64
				}
			}
		} else {
			if o > 0 {
				// negative * positive = negative. underflow?
				if v < (math.MinInt64 / o) {
					return math.MinInt64
				}
			} else {
				// negative * negative = positive. overflow?
				if (v != 0) && (o < (math.MaxInt64 / v)) {
					return math.MaxInt64
				}
			}
		}
		return int64(v * o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	} else if (v == math.MinInt64) && (o == -1) {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v / o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT33-C
		// https://golang.org/ref/spec#Integer_operators
		if o == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		} else if (v == math.MinInt64) && (o == -1) {
			return math.MaxInt64
		}
		return int64(v / o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Int64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Int64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)

}

func (v Int64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Int64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt64, ok := other.(Int64Value)
	if !ok {
		return false
	}
	return v == otherInt64
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt64 (1 byte)
// - int64 value encoded in big-endian (8 bytes)
func (v Int64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertInt64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int64Value {
	converter := func() int64 {
		switch value := value.(type) {
		case BigNumberValue:
			v := value.ToBigInt(memoryGauge)
			if v.Cmp(sema.Int64TypeMaxInt) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Cmp(sema.Int64TypeMinInt) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return v.Int64()

		case NumberValue:
			v := value.ToInt(locationRange)
			return int64(v)

		default:
			panic(errors.NewUnreachableError())
		}
	}

	return NewInt64Value(memoryGauge, converter)
}

func (v Int64Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v | o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v ^ o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v & o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v << o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(v >> o)
	}

	return NewInt64Value(interpreter, valueGetter)
}

func (v Int64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int64Type, locationRange)
}

func (Int64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Int64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int64Value) Transfer(
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

func (v Int64Value) Clone(_ *Interpreter) Value {
	return v
}

func (Int64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int64Value) ByteSize() uint32 {
	return cborTagSize + getIntCBORSize(int64(v))
}

func (v Int64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int64Value) ChildStorables() []atree.Storable {
	return nil
}

// Int128Value

type Int128Value struct {
	BigInt *big.Int
}

func NewInt128ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Int128Value {
	return NewInt128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Int128MemoryUsage = common.NewBigIntMemoryUsage(16)

func NewInt128ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Int128Value {
	common.UseMemory(memoryGauge, Int128MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredInt128ValueFromBigInt(value)
}

func NewUnmeteredInt128ValueFromInt64(value int64) Int128Value {
	return NewUnmeteredInt128ValueFromBigInt(big.NewInt(value))
}

func NewUnmeteredInt128ValueFromBigInt(value *big.Int) Int128Value {
	return Int128Value{
		BigInt: value,
	}
}

var _ Value = Int128Value{}
var _ atree.Storable = Int128Value{}
var _ NumberValue = Int128Value{}
var _ IntegerValue = Int128Value{}
var _ EquatableValue = Int128Value{}
var _ ComparableValue = Int128Value{}
var _ HashableValue = Int128Value{}
var _ MemberAccessibleValue = Int128Value{}

func (Int128Value) isValue() {}

func (v Int128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt128Value(interpreter, v)
}

func (Int128Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int128Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt128)
}

func (Int128Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int128Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Int128Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Int128Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Int128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int128Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int128Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	//   if v == Int128TypeMinIntBig {
	//       ...
	//   }
	if v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		return new(big.Int).Neg(v.BigInt)
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int128TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int128TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int128TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int128TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			return sema.Int128TypeMinIntBig
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			return sema.Int128TypeMaxIntBig
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int128TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int128TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int128 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int128TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int128TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			return sema.Int128TypeMinIntBig
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			return sema.Int128TypeMaxIntBig
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.Rem(v.BigInt, o.BigInt)

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
			return sema.Int128TypeMinIntBig
		} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			return sema.Int128TypeMaxIntBig
		}

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int128TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		res.Div(v.BigInt, o.BigInt)

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int128TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			return sema.Int128TypeMaxIntBig
		}
		res.Div(v.BigInt, o.BigInt)

		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Int128Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Int128Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Int128Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Int128Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Int128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt128 (1 byte)
// - big int value encoded in big-endian (n bytes)
func (v Int128Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := SignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeInt128)
	copy(buffer[1:], b)
	return buffer
}

func ConvertInt128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int128Value {
	converter := func() *big.Int {
		var v *big.Int

		switch value := value.(type) {
		case BigNumberValue:
			v = value.ToBigInt(memoryGauge)

		case NumberValue:
			v = big.NewInt(int64(value.ToInt(locationRange)))

		default:
			panic(errors.NewUnreachableError())
		}

		if v.Cmp(sema.Int128TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		} else if v.Cmp(sema.Int128TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}

		return v
	}

	return NewInt128ValueFromBigInt(memoryGauge, converter)
}

func (v Int128Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Or(v.BigInt, o.BigInt)
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Xor(v.BigInt, o.BigInt)
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.And(v.BigInt, o.BigInt)
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		return res
	}

	return NewInt128ValueFromBigInt(interpreter, valueGetter)
}

func (v Int128Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int128Type, locationRange)
}

func (Int128Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int128Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int128Value) ToBigEndianBytes() []byte {
	return SignedBigIntToSizedBigEndianBytes(v.BigInt, sema.Int128TypeSize)
}

func (v Int128Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int128Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int128Value) Transfer(
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

func (v Int128Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredInt128ValueFromBigInt(v.BigInt)
}

func (Int128Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int128Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Int128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int128Value) ChildStorables() []atree.Storable {
	return nil
}

// Int256Value

type Int256Value struct {
	BigInt *big.Int
}

func NewInt256ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Int256Value {
	return NewInt256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Int256MemoryUsage = common.NewBigIntMemoryUsage(32)

func NewInt256ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Int256Value {
	common.UseMemory(memoryGauge, Int256MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredInt256ValueFromBigInt(value)
}

func NewUnmeteredInt256ValueFromInt64(value int64) Int256Value {
	return NewUnmeteredInt256ValueFromBigInt(big.NewInt(value))
}

func NewUnmeteredInt256ValueFromBigInt(value *big.Int) Int256Value {
	return Int256Value{
		BigInt: value,
	}
}

var _ Value = Int256Value{}
var _ atree.Storable = Int256Value{}
var _ NumberValue = Int256Value{}
var _ IntegerValue = Int256Value{}
var _ EquatableValue = Int256Value{}
var _ ComparableValue = Int256Value{}
var _ HashableValue = Int256Value{}
var _ MemberAccessibleValue = Int256Value{}

func (Int256Value) isValue() {}

func (v Int256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt256Value(interpreter, v)
}

func (Int256Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Int256Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeInt256)
}

func (Int256Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Int256Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Int256Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Int256Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Int256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int256Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Int256Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	//   if v == Int256TypeMinIntBig {
	//       ...
	//   }
	if v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		return new(big.Int).Neg(v.BigInt)
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int256TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int256TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just add and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v > (Int256TypeMaxIntBig - o)) {
		//       ...
		//   } else if (o < 0) && (v < (Int256TypeMinIntBig - o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Add(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			return sema.Int256TypeMinIntBig
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			return sema.Int256TypeMaxIntBig
		}

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int256TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int256TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		// Given that this value is backed by an arbitrary size integer,
		// we can just subtract and check the range of the result.
		//
		// If Go gains a native int256 type and we switch this value
		// to be based on it, then we need to follow INT32-C:
		//
		//   if (o > 0) && (v < (Int256TypeMinIntBig + o)) {
		// 	     ...
		//   } else if (o < 0) && (v > (Int256TypeMaxIntBig + o)) {
		//       ...
		//   }
		//
		res := new(big.Int)
		res.Sub(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			return sema.Int256TypeMinIntBig
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			return sema.Int256TypeMaxIntBig
		}

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.Rem(v.BigInt, o.BigInt)

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Mul(v.BigInt, o.BigInt)
		if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
			return sema.Int256TypeMinIntBig
		} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			return sema.Int256TypeMaxIntBig
		}

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int256TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		res.Div(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		// INT33-C:
		//   if o == 0 {
		//       ...
		//   } else if (v == Int256TypeMinIntBig) && (o == -1) {
		//       ...
		//   }
		if o.BigInt.Cmp(res) == 0 {
			panic(DivisionByZeroError{
				LocationRange: locationRange,
			})
		}
		res.SetInt64(-1)
		if (v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
			return sema.Int256TypeMaxIntBig
		}
		res.Div(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Int256Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Int256Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Int256Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Int256Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Int256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeInt256 (1 byte)
// - big int value encoded in big-endian (n bytes)
func (v Int256Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := SignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeInt256)
	copy(buffer[1:], b)
	return buffer
}

func ConvertInt256(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Int256Value {
	converter := func() *big.Int {
		var v *big.Int

		switch value := value.(type) {
		case BigNumberValue:
			v = value.ToBigInt(memoryGauge)

		case NumberValue:
			v = big.NewInt(int64(value.ToInt(locationRange)))

		default:
			panic(errors.NewUnreachableError())
		}

		if v.Cmp(sema.Int256TypeMaxIntBig) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		} else if v.Cmp(sema.Int256TypeMinIntBig) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}

		return v
	}

	return NewInt256ValueFromBigInt(memoryGauge, converter)
}

func (v Int256Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Or(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.Xor(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		res.And(v.BigInt, o.BigInt)
		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		if o.BigInt.Sign() < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		if !o.BigInt.IsUint64() {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))

		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() *big.Int {
		res := new(big.Int)
		if o.BigInt.Sign() < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		if !o.BigInt.IsUint64() {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		return res
	}

	return NewInt256ValueFromBigInt(interpreter, valueGetter)
}

func (v Int256Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Int256Type, locationRange)
}

func (Int256Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int256Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int256Value) ToBigEndianBytes() []byte {
	return SignedBigIntToSizedBigEndianBytes(v.BigInt, sema.Int256TypeSize)
}

func (v Int256Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v Int256Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Int256Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Int256Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Int256Value) Transfer(
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

func (v Int256Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredInt256ValueFromBigInt(v.BigInt)
}

func (Int256Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int256Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Int256Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Int256Value) ChildStorables() []atree.Storable {
	return nil
}

// UIntValue

type UIntValue struct {
	BigInt *big.Int
}

const uint64Size = int(unsafe.Sizeof(uint64(0)))

var uint64BigIntMemoryUsage = common.NewBigIntMemoryUsage(uint64Size)

func NewUIntValueFromUint64(memoryGauge common.MemoryGauge, value uint64) UIntValue {
	return NewUIntValueFromBigInt(
		memoryGauge,
		uint64BigIntMemoryUsage,
		func() *big.Int {
			return new(big.Int).SetUint64(value)
		},
	)
}

func NewUnmeteredUIntValueFromUint64(value uint64) UIntValue {
	return NewUnmeteredUIntValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUIntValueFromBigInt(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) UIntValue {
	common.UseMemory(memoryGauge, memoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredUIntValueFromBigInt(value)
}

func NewUnmeteredUIntValueFromBigInt(value *big.Int) UIntValue {
	return UIntValue{
		BigInt: value,
	}
}

func ConvertUInt(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UIntValue {
	switch value := value.(type) {
	case BigNumberValue:
		return NewUIntValueFromBigInt(
			memoryGauge,
			common.NewBigIntMemoryUsage(value.ByteLength()),
			func() *big.Int {
				v := value.ToBigInt(memoryGauge)
				if v.Sign() < 0 {
					panic(UnderflowError{
						LocationRange: locationRange,
					})
				}
				return v
			},
		)

	case NumberValue:
		v := value.ToInt(locationRange)
		if v < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return NewUIntValueFromUint64(
			memoryGauge,
			uint64(v),
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

var _ Value = UIntValue{}
var _ atree.Storable = UIntValue{}
var _ NumberValue = UIntValue{}
var _ IntegerValue = UIntValue{}
var _ EquatableValue = UIntValue{}
var _ ComparableValue = UIntValue{}
var _ HashableValue = UIntValue{}
var _ MemberAccessibleValue = UIntValue{}

func (UIntValue) isValue() {}

func (v UIntValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUIntValue(interpreter, v)
}

func (UIntValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UIntValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt)
}

func (v UIntValue) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UIntValue) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v UIntValue) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v UIntValue) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v UIntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v UIntValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UIntValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UIntValue) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UIntValue) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewPlusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Add(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Plus(interpreter, other, locationRange)
}

func (v UIntValue) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewMinusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			res.Sub(v.BigInt, o.BigInt)
			// INT30-C
			if res.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return res
		},
	)
}

func (v UIntValue) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewMinusBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			res.Sub(v.BigInt, o.BigInt)
			// INT30-C
			if res.Sign() < 0 {
				return sema.UIntTypeMin
			}
			return res
		},
	)
}

func (v UIntValue) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewModBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			// INT33-C
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			res.Rem(v.BigInt, o.BigInt)
			return res
		},
	)
}

func (v UIntValue) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewMulBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Mul(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Mul(interpreter, other, locationRange)
}

func (v UIntValue) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewDivBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			// INT33-C
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UIntValue) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v UIntValue) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v UIntValue) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v UIntValue) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v UIntValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt, ok := other.(UIntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherUInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt (1 byte)
// - big int value encoded in big-endian (n bytes)
func (v UIntValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeUInt)
	copy(buffer[1:], b)
	return buffer
}

func (v UIntValue) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseOrBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseXorBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseAndBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)
}

func (v UIntValue) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseLeftShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))

		},
	)
}

func (v UIntValue) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUIntValueFromBigInt(
		interpreter,
		common.NewBitwiseRightShiftBigIntMemoryUsage(v.BigInt, o.BigInt),
		func() *big.Int {
			res := new(big.Int)
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v UIntValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UIntType, locationRange)
}

func (UIntValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UIntValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UIntValue) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UIntValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v UIntValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (UIntValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UIntValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UIntValue) Transfer(
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

func (v UIntValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredUIntValueFromBigInt(v.BigInt)
}

func (UIntValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UIntValue) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v UIntValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UIntValue) ChildStorables() []atree.Storable {
	return nil
}

// UInt8Value

type UInt8Value uint8

var _ Value = UInt8Value(0)
var _ atree.Storable = UInt8Value(0)
var _ NumberValue = UInt8Value(0)
var _ IntegerValue = UInt8Value(0)
var _ EquatableValue = UInt8Value(0)
var _ ComparableValue = UInt8Value(0)
var _ HashableValue = UInt8Value(0)
var _ MemberAccessibleValue = UInt8Value(0)

var UInt8MemoryUsage = common.NewNumberMemoryUsage(int(unsafe.Sizeof(UInt8Value(0))))

func NewUInt8Value(gauge common.MemoryGauge, uint8Constructor func() uint8) UInt8Value {
	common.UseMemory(gauge, UInt8MemoryUsage)

	return NewUnmeteredUInt8Value(uint8Constructor())
}

func NewUnmeteredUInt8Value(value uint8) UInt8Value {
	return UInt8Value(value)
}

func (UInt8Value) isValue() {}

func (v UInt8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt8Value(interpreter, v)
}

func (UInt8Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UInt8Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt8)
}

func (UInt8Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UInt8Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt8Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UInt8Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v UInt8Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt8Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(interpreter, func() uint8 {
		sum := v + o
		// INT30-C
		if sum < v {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		return uint8(sum)
	})
}

func (v UInt8Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(interpreter, func() uint8 {
		sum := v + o
		// INT30-C
		if sum < v {
			return math.MaxUint8
		}
		return uint8(sum)
	})
}

func (v UInt8Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			diff := v - o
			// INT30-C
			if diff > v {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return uint8(diff)
		},
	)
}

func (v UInt8Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			diff := v - o
			// INT30-C
			if diff > v {
				return 0
			}
			return uint8(diff)
		},
	)
}

func (v UInt8Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint8(v % o)
		},
	)
}

func (v UInt8Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint8 / o)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint8(v * o)
		},
	)
}

func (v UInt8Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint8 / o)) {
				return math.MaxUint8
			}
			return uint8(v * o)
		},
	)
}

func (v UInt8Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint8(v / o)
		},
	)
}

func (v UInt8Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UInt8Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v UInt8Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v UInt8Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v UInt8Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v UInt8Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt8, ok := other.(UInt8Value)
	if !ok {
		return false
	}
	return v == otherUInt8
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt8 (1 byte)
// - uint8 value (1 byte)
func (v UInt8Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertUnsigned[T Unsigned](
	memoryGauge common.MemoryGauge,
	value Value,
	maxBigNumber *big.Int,
	maxNumber int,
	locationRange LocationRange,
) T {
	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt(memoryGauge)
		if v.Cmp(maxBigNumber) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		} else if v.Sign() < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return T(v.Int64())

	case NumberValue:
		v := value.ToInt(locationRange)
		if maxNumber > 0 && v > maxNumber {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		} else if v < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return T(v)

	default:
		panic(errors.NewUnreachableError())
	}
}

func ConvertWord[T Unsigned](
	memoryGauge common.MemoryGauge,
	value Value,
	locationRange LocationRange,
) T {
	switch value := value.(type) {
	case BigNumberValue:
		return T(value.ToBigInt(memoryGauge).Int64())

	case NumberValue:
		return T(value.ToInt(locationRange))

	default:
		panic(errors.NewUnreachableError())
	}
}

func ConvertUInt8(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt8Value {
	return NewUInt8Value(
		memoryGauge,
		func() uint8 {
			return ConvertUnsigned[uint8](
				memoryGauge,
				value,
				sema.UInt8TypeMaxInt,
				math.MaxUint8,
				locationRange,
			)
		},
	)
}

func (v UInt8Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			return uint8(v | o)
		},
	)
}

func (v UInt8Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			return uint8(v ^ o)
		},
	)
}

func (v UInt8Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			return uint8(v & o)
		},
	)
}

func (v UInt8Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			return uint8(v << o)
		},
	)
}

func (v UInt8Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt8Value(
		interpreter,
		func() uint8 {
			return uint8(v >> o)
		},
	)
}

func (v UInt8Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt8Type, locationRange)
}

func (UInt8Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt8Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v UInt8Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v UInt8Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt8Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt8Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt8Value) Transfer(
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

func (v UInt8Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt8Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt8Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UInt8Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt8Value) ChildStorables() []atree.Storable {
	return nil
}

// UInt16Value

type UInt16Value uint16

var _ Value = UInt16Value(0)
var _ atree.Storable = UInt16Value(0)
var _ NumberValue = UInt16Value(0)
var _ IntegerValue = UInt16Value(0)
var _ EquatableValue = UInt16Value(0)
var _ ComparableValue = UInt16Value(0)
var _ HashableValue = UInt16Value(0)
var _ MemberAccessibleValue = UInt16Value(0)

var UInt16MemoryUsage = common.NewNumberMemoryUsage(int(unsafe.Sizeof(UInt16Value(0))))

func NewUInt16Value(gauge common.MemoryGauge, uint16Constructor func() uint16) UInt16Value {
	common.UseMemory(gauge, UInt16MemoryUsage)

	return NewUnmeteredUInt16Value(uint16Constructor())
}

func NewUnmeteredUInt16Value(value uint16) UInt16Value {
	return UInt16Value(value)
}

func (UInt16Value) isValue() {}

func (v UInt16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt16Value(interpreter, v)
}

func (UInt16Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UInt16Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt16)
}

func (UInt16Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UInt16Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt16Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UInt16Value) ToInt(_ LocationRange) int {
	return int(v)
}
func (v UInt16Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			sum := v + o
			// INT30-C
			if sum < v {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint16(sum)
		},
	)
}

func (v UInt16Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			sum := v + o
			// INT30-C
			if sum < v {
				return math.MaxUint16
			}
			return uint16(sum)
		},
	)
}

func (v UInt16Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			diff := v - o
			// INT30-C
			if diff > v {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return uint16(diff)
		},
	)
}

func (v UInt16Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			diff := v - o
			// INT30-C
			if diff > v {
				return 0
			}
			return uint16(diff)
		},
	)
}

func (v UInt16Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint16(v % o)
		},
	)
}

func (v UInt16Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint16(v * o)
		},
	)
}

func (v UInt16Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
				return math.MaxUint16
			}
			return uint16(v * o)
		},
	)
}

func (v UInt16Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint16(v / o)
		},
	)
}

func (v UInt16Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UInt16Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v UInt16Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v UInt16Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v UInt16Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v UInt16Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt16, ok := other.(UInt16Value)
	if !ok {
		return false
	}
	return v == otherUInt16
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt16 (1 byte)
// - uint16 value encoded in big-endian (2 bytes)
func (v UInt16Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertUInt16(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt16Value {
	return NewUInt16Value(
		memoryGauge,
		func() uint16 {
			return ConvertUnsigned[uint16](
				memoryGauge,
				value,
				sema.UInt16TypeMaxInt,
				math.MaxUint16,
				locationRange,
			)
		},
	)
}

func (v UInt16Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v | o)
		},
	)
}

func (v UInt16Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v ^ o)
		},
	)
}

func (v UInt16Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v & o)
		},
	)
}

func (v UInt16Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v << o)
		},
	)
}

func (v UInt16Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt16Value(
		interpreter,
		func() uint16 {
			return uint16(v >> o)
		},
	)
}

func (v UInt16Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt16Type, locationRange)
}

func (UInt16Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt16Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v UInt16Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt16Value) IsStorable() bool {
	return true
}

func (v UInt16Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt16Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt16Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt16Value) Transfer(
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

func (v UInt16Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt16Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt16Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UInt16Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt16Value) ChildStorables() []atree.Storable {
	return nil
}

// UInt32Value

type UInt32Value uint32

var UInt32MemoryUsage = common.NewNumberMemoryUsage(int(unsafe.Sizeof(UInt32Value(0))))

func NewUInt32Value(gauge common.MemoryGauge, uint32Constructor func() uint32) UInt32Value {
	common.UseMemory(gauge, UInt32MemoryUsage)

	return NewUnmeteredUInt32Value(uint32Constructor())
}

func NewUnmeteredUInt32Value(value uint32) UInt32Value {
	return UInt32Value(value)
}

var _ Value = UInt32Value(0)
var _ atree.Storable = UInt32Value(0)
var _ NumberValue = UInt32Value(0)
var _ IntegerValue = UInt32Value(0)
var _ EquatableValue = UInt32Value(0)
var _ ComparableValue = UInt32Value(0)
var _ HashableValue = UInt32Value(0)
var _ MemberAccessibleValue = UInt32Value(0)

func (UInt32Value) isValue() {}

func (v UInt32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt32Value(interpreter, v)
}

func (UInt32Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UInt32Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt32)
}

func (UInt32Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UInt32Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt32Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UInt32Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v UInt32Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			sum := v + o
			// INT30-C
			if sum < v {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint32(sum)
		},
	)
}

func (v UInt32Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			sum := v + o
			// INT30-C
			if sum < v {
				return math.MaxUint32
			}
			return uint32(sum)
		},
	)
}

func (v UInt32Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			diff := v - o
			// INT30-C
			if diff > v {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return uint32(diff)
		},
	)
}

func (v UInt32Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			diff := v - o
			// INT30-C
			if diff > v {
				return 0
			}
			return uint32(diff)
		},
	)
}

func (v UInt32Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint32(v % o)
		},
	)
}

func (v UInt32Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint32(v * o)
		},
	)
}

func (v UInt32Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {

			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
				return math.MaxUint32
			}
			return uint32(v * o)
		},
	)
}

func (v UInt32Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint32(v / o)
		},
	)
}

func (v UInt32Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UInt32Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v UInt32Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v UInt32Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v UInt32Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v UInt32Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt32, ok := other.(UInt32Value)
	if !ok {
		return false
	}
	return v == otherUInt32
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt32 (1 byte)
// - uint32 value encoded in big-endian (4 bytes)
func (v UInt32Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertUInt32(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt32Value {
	return NewUInt32Value(
		memoryGauge,
		func() uint32 {
			return ConvertUnsigned[uint32](
				memoryGauge,
				value,
				sema.UInt32TypeMaxInt,
				math.MaxUint32,
				locationRange,
			)
		},
	)
}

func (v UInt32Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v | o)
		},
	)
}

func (v UInt32Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v ^ o)
		},
	)
}

func (v UInt32Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v & o)
		},
	)
}

func (v UInt32Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v << o)
		},
	)
}

func (v UInt32Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt32Value(
		interpreter,
		func() uint32 {
			return uint32(v >> o)
		},
	)
}

func (v UInt32Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt32Type, locationRange)
}

func (UInt32Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt32Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v UInt32Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt32Value) IsStorable() bool {
	return true
}

func (v UInt32Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt32Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt32Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt32Value) Transfer(
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

func (v UInt32Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt32Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt32Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UInt32Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt32Value) ChildStorables() []atree.Storable {
	return nil
}

// UInt64Value

type UInt64Value uint64

var _ Value = UInt64Value(0)
var _ atree.Storable = UInt64Value(0)
var _ NumberValue = UInt64Value(0)
var _ IntegerValue = UInt64Value(0)
var _ EquatableValue = UInt64Value(0)
var _ ComparableValue = UInt64Value(0)
var _ HashableValue = UInt64Value(0)
var _ MemberAccessibleValue = UInt64Value(0)

// NOTE: important, do *NOT* remove:
// UInt64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
var _ BigNumberValue = UInt64Value(0)

var UInt64MemoryUsage = common.NewNumberMemoryUsage(int(unsafe.Sizeof(UInt64Value(0))))

func NewUInt64Value(gauge common.MemoryGauge, uint64Constructor func() uint64) UInt64Value {
	common.UseMemory(gauge, UInt64MemoryUsage)

	return NewUnmeteredUInt64Value(uint64Constructor())
}

func NewUnmeteredUInt64Value(value uint64) UInt64Value {
	return UInt64Value(value)
}

func (UInt64Value) isValue() {}

func (v UInt64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt64Value(interpreter, v)
}

func (UInt64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UInt64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt64)
}

func (UInt64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UInt64Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UInt64Value) ToInt(locationRange LocationRange) int {
	if v > math.MaxInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v)
}

func (v UInt64Value) ByteLength() int {
	return 8
}

// ToBigInt
//
// NOTE: important, do *NOT* remove:
// UInt64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
func (v UInt64Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).SetUint64(uint64(v))
}

func (v UInt64Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func safeAddUint64(a, b uint64, locationRange LocationRange) uint64 {
	sum := a + b
	// INT30-C
	if sum < a {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return sum
}

func (v UInt64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return safeAddUint64(uint64(v), uint64(o), locationRange)
		},
	)
}

func (v UInt64Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			sum := v + o
			// INT30-C
			if sum < v {
				return math.MaxUint64
			}
			return uint64(sum)
		},
	)
}

func (v UInt64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			diff := v - o
			// INT30-C
			if diff > v {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return uint64(diff)
		},
	)
}

func (v UInt64Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			diff := v - o
			// INT30-C
			if diff > v {
				return 0
			}
			return uint64(diff)
		},
	)
}

func (v UInt64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint64(v % o)
		},
	)
}

func (v UInt64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return uint64(v * o)
		},
	)
}

func (v UInt64Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			// INT30-C
			if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
				return math.MaxUint64
			}
			return uint64(v * o)
		},
	)
}

func (v UInt64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			if o == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return uint64(v / o)
		},
	)
}

func (v UInt64Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UInt64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v UInt64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v UInt64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v UInt64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v UInt64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUInt64, ok := other.(UInt64Value)
	if !ok {
		return false
	}
	return v == otherUInt64
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v UInt64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertUInt64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt64Value {
	return NewUInt64Value(
		memoryGauge,
		func() uint64 {
			return ConvertUnsigned[uint64](
				memoryGauge,
				value,
				sema.UInt64TypeMaxInt,
				-1,
				locationRange,
			)
		},
	)
}

func (v UInt64Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v | o)
		},
	)
}

func (v UInt64Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v ^ o)
		},
	)
}

func (v UInt64Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v & o)
		},
	)
}

func (v UInt64Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v << o)
		},
	)
}

func (v UInt64Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt64Value(
		interpreter,
		func() uint64 {
			return uint64(v >> o)
		},
	)
}

func (v UInt64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt64Type, locationRange)
}

func (UInt64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UInt64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt64Value) IsStorable() bool {
	return true
}

func (v UInt64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt64Value) Transfer(
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

func (v UInt64Value) Clone(_ *Interpreter) Value {
	return v
}

func (UInt64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt64Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UInt64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt64Value) ChildStorables() []atree.Storable {
	return nil
}

// UInt128Value

type UInt128Value struct {
	BigInt *big.Int
}

func NewUInt128ValueFromUint64(interpreter *Interpreter, value uint64) UInt128Value {
	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			return new(big.Int).SetUint64(value)
		},
	)
}

func NewUnmeteredUInt128ValueFromUint64(value uint64) UInt128Value {
	return NewUnmeteredUInt128ValueFromBigInt(new(big.Int).SetUint64(value))
}

var Uint128MemoryUsage = common.NewBigIntMemoryUsage(16)

func NewUInt128ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) UInt128Value {
	common.UseMemory(memoryGauge, Uint128MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredUInt128ValueFromBigInt(value)
}

func NewUnmeteredUInt128ValueFromBigInt(value *big.Int) UInt128Value {
	return UInt128Value{
		BigInt: value,
	}
}

var _ Value = UInt128Value{}
var _ atree.Storable = UInt128Value{}
var _ NumberValue = UInt128Value{}
var _ IntegerValue = UInt128Value{}
var _ EquatableValue = UInt128Value{}
var _ ComparableValue = UInt128Value{}
var _ HashableValue = UInt128Value{}
var _ MemberAccessibleValue = UInt128Value{}

func (UInt128Value) isValue() {}

func (v UInt128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt128Value(interpreter, v)
}

func (UInt128Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UInt128Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt128)
}

func (UInt128Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UInt128Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v UInt128Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v UInt128Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v UInt128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v UInt128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt128Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UInt128Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt128Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and check the range of the result.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return sum
		},
	)
}

func (v UInt128Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and check the range of the result.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
				return sema.UInt128TypeMaxIntBig
			}
			return sum
		},
	)
}

func (v UInt128Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and check the range of the result.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Cmp(sema.UInt128TypeMinIntBig) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return diff
		},
	)
}

func (v UInt128Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and check the range of the result.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Cmp(sema.UInt128TypeMinIntBig) < 0 {
				return sema.UInt128TypeMinIntBig
			}
			return diff
		},
	)
}

func (v UInt128Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt128Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res
		},
	)
}

func (v UInt128Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
				return sema.UInt128TypeMaxIntBig
			}
			return res
		},
	)
}

func (v UInt128Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)

}

func (v UInt128Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UInt128Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v UInt128Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v UInt128Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v UInt128Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v UInt128Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(UInt128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt128 (1 byte)
// - big int encoded in big endian (n bytes)
func (v UInt128Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeUInt128)
	copy(buffer[1:], b)
	return buffer
}

func ConvertUInt128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
	return NewUInt128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {

			var v *big.Int

			switch value := value.(type) {
			case BigNumberValue:
				v = value.ToBigInt(memoryGauge)

			case NumberValue:
				v = big.NewInt(int64(value.ToInt(locationRange)))

			default:
				panic(errors.NewUnreachableError())
			}

			if v.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}

			return v
		},
	)
}

func (v UInt128Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt128Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt128Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)

}

func (v UInt128Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v UInt128Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v UInt128Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt128Type, locationRange)
}

func (UInt128Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt128Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt128Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToSizedBigEndianBytes(v.BigInt, sema.UInt128TypeSize)
}

func (v UInt128Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt128Value) IsStorable() bool {
	return true
}

func (v UInt128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt128Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UInt128Value) Transfer(
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

func (v UInt128Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredUInt128ValueFromBigInt(v.BigInt)
}

func (UInt128Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt128Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v UInt128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt128Value) ChildStorables() []atree.Storable {
	return nil
}

// UInt256Value

type UInt256Value struct {
	BigInt *big.Int
}

func NewUInt256ValueFromUint64(interpreter *Interpreter, value uint64) UInt256Value {
	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			return new(big.Int).SetUint64(value)
		},
	)
}

func NewUnmeteredUInt256ValueFromUint64(value uint64) UInt256Value {
	return NewUnmeteredUInt256ValueFromBigInt(new(big.Int).SetUint64(value))
}

var Uint256MemoryUsage = common.NewBigIntMemoryUsage(32)

func NewUInt256ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) UInt256Value {
	common.UseMemory(memoryGauge, Uint256MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredUInt256ValueFromBigInt(value)
}

func NewUnmeteredUInt256ValueFromBigInt(value *big.Int) UInt256Value {
	return UInt256Value{
		BigInt: value,
	}
}

var _ Value = UInt256Value{}
var _ atree.Storable = UInt256Value{}
var _ NumberValue = UInt256Value{}
var _ IntegerValue = UInt256Value{}
var _ EquatableValue = UInt256Value{}
var _ ComparableValue = UInt256Value{}
var _ HashableValue = UInt256Value{}
var _ MemberAccessibleValue = UInt256Value{}

func (UInt256Value) isValue() {}

func (v UInt256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt256Value(interpreter, v)
}

func (UInt256Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UInt256Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUInt256)
}

func (UInt256Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UInt256Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return int(v.BigInt.Int64())
}

func (v UInt256Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v UInt256Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v UInt256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v UInt256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt256Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UInt256Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return sum
		},
	)

}

func (v UInt256Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				return sema.UInt256TypeMaxIntBig
			}
			return sum
		},
	)
}

func (v UInt256Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Cmp(sema.UInt256TypeMinIntBig) < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			return diff
		},
	)
}

func (v UInt256Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and check the range of the result.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Cmp(sema.UInt256TypeMinIntBig) < 0 {
				return sema.UInt256TypeMinIntBig
			}
			return diff
		},
	)

}

func (v UInt256Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res
		},
	)
}

func (v UInt256Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				return sema.UInt256TypeMaxIntBig
			}
			return res
		},
	)
}

func (v UInt256Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UInt256Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v UInt256Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v UInt256Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v UInt256Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v UInt256Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(UInt256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeUInt256 (1 byte)
// - big int encoded in big endian (n bytes)
func (v UInt256Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeUInt256)
	copy(buffer[1:], b)
	return buffer
}

func ConvertUInt256(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UInt256Value {
	return NewUInt256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			var v *big.Int

			switch value := value.(type) {
			case BigNumberValue:
				v = value.ToBigInt(memoryGauge)

			case NumberValue:
				v = big.NewInt(int64(value.ToInt(locationRange)))

			default:
				panic(errors.NewUnreachableError())
			}

			if v.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			} else if v.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}

			return v
		},
	)
}

func (v UInt256Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)
}

func (v UInt256Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v UInt256Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewUInt256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v UInt256Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UInt256Type, locationRange)
}

func (UInt256Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt256Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToSizedBigEndianBytes(v.BigInt, sema.UInt256TypeSize)
}

func (v UInt256Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UInt256Value) IsStorable() bool {
	return true
}

func (v UInt256Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UInt256Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UInt256Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}
func (v UInt256Value) Transfer(
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

func (v UInt256Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredUInt256ValueFromBigInt(v.BigInt)
}

func (UInt256Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt256Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v UInt256Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UInt256Value) ChildStorables() []atree.Storable {
	return nil
}

// Word8Value

type Word8Value uint8

var _ Value = Word8Value(0)
var _ atree.Storable = Word8Value(0)
var _ NumberValue = Word8Value(0)
var _ IntegerValue = Word8Value(0)
var _ EquatableValue = Word8Value(0)
var _ ComparableValue = Word8Value(0)
var _ HashableValue = Word8Value(0)
var _ MemberAccessibleValue = Word8Value(0)

const word8Size = int(unsafe.Sizeof(Word8Value(0)))

var word8MemoryUsage = common.NewNumberMemoryUsage(word8Size)

func NewWord8Value(gauge common.MemoryGauge, valueGetter func() uint8) Word8Value {
	common.UseMemory(gauge, word8MemoryUsage)

	return NewUnmeteredWord8Value(valueGetter())
}

func NewUnmeteredWord8Value(value uint8) Word8Value {
	return Word8Value(value)
}

func (Word8Value) isValue() {}

func (v Word8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord8Value(interpreter, v)
}

func (Word8Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word8Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord8)
}

func (Word8Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word8Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word8Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word8Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Word8Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v + o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v - o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v % o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v * o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v / o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Word8Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Word8Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Word8Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Word8Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord8, ok := other.(Word8Value)
	if !ok {
		return false
	}
	return v == otherWord8
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord8 (1 byte)
// - uint8 value (1 byte)
func (v Word8Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertWord8(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word8Value {
	return NewWord8Value(
		memoryGauge,
		func() uint8 {
			return ConvertWord[uint8](memoryGauge, value, locationRange)
		},
	)
}

func (v Word8Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v | o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v ^ o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v & o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v << o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint8 {
		return uint8(v >> o)
	}

	return NewWord8Value(interpreter, valueGetter)
}

func (v Word8Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word8Type, locationRange)
}

func (Word8Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word8Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Word8Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word8Value) IsStorable() bool {
	return true
}

func (v Word8Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word8Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word8Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word8Value) Transfer(
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

func (v Word8Value) Clone(_ *Interpreter) Value {
	return v
}

func (Word8Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word8Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v Word8Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word8Value) ChildStorables() []atree.Storable {
	return nil
}

// Word16Value

type Word16Value uint16

var _ Value = Word16Value(0)
var _ atree.Storable = Word16Value(0)
var _ NumberValue = Word16Value(0)
var _ IntegerValue = Word16Value(0)
var _ EquatableValue = Word16Value(0)
var _ ComparableValue = Word16Value(0)
var _ HashableValue = Word16Value(0)
var _ MemberAccessibleValue = Word16Value(0)

const word16Size = int(unsafe.Sizeof(Word16Value(0)))

var word16MemoryUsage = common.NewNumberMemoryUsage(word16Size)

func NewWord16Value(gauge common.MemoryGauge, valueGetter func() uint16) Word16Value {
	common.UseMemory(gauge, word16MemoryUsage)

	return NewUnmeteredWord16Value(valueGetter())
}

func NewUnmeteredWord16Value(value uint16) Word16Value {
	return Word16Value(value)
}

func (Word16Value) isValue() {}

func (v Word16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord16Value(interpreter, v)
}

func (Word16Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word16Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord16)
}

func (Word16Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word16Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word16Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word16Value) ToInt(_ LocationRange) int {
	return int(v)
}
func (v Word16Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v + o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v - o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v % o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v * o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v / o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Word16Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Word16Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Word16Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Word16Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord16, ok := other.(Word16Value)
	if !ok {
		return false
	}
	return v == otherWord16
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord16 (1 byte)
// - uint16 value encoded in big-endian (2 bytes)
func (v Word16Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertWord16(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word16Value {
	return NewWord16Value(
		memoryGauge,
		func() uint16 {
			return ConvertWord[uint16](memoryGauge, value, locationRange)
		},
	)
}

func (v Word16Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v | o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v ^ o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v & o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v << o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint16 {
		return uint16(v >> o)
	}

	return NewWord16Value(interpreter, valueGetter)
}

func (v Word16Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word16Type, locationRange)
}

func (Word16Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word16Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Word16Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word16Value) IsStorable() bool {
	return true
}

func (v Word16Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word16Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word16Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word16Value) Transfer(
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

func (v Word16Value) Clone(_ *Interpreter) Value {
	return v
}

func (Word16Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word16Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v Word16Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word16Value) ChildStorables() []atree.Storable {
	return nil
}

// Word32Value

type Word32Value uint32

var _ Value = Word32Value(0)
var _ atree.Storable = Word32Value(0)
var _ NumberValue = Word32Value(0)
var _ IntegerValue = Word32Value(0)
var _ EquatableValue = Word32Value(0)
var _ ComparableValue = Word32Value(0)
var _ HashableValue = Word32Value(0)
var _ MemberAccessibleValue = Word32Value(0)

const word32Size = int(unsafe.Sizeof(Word32Value(0)))

var word32MemoryUsage = common.NewNumberMemoryUsage(word32Size)

func NewWord32Value(gauge common.MemoryGauge, valueGetter func() uint32) Word32Value {
	common.UseMemory(gauge, word32MemoryUsage)

	return NewUnmeteredWord32Value(valueGetter())
}

func NewUnmeteredWord32Value(value uint32) Word32Value {
	return Word32Value(value)
}

func (Word32Value) isValue() {}

func (v Word32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord32Value(interpreter, v)
}

func (Word32Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word32Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord32)
}

func (Word32Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word32Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word32Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word32Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Word32Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v + o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v - o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v % o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v * o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v / o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Word32Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Word32Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Word32Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Word32Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord32, ok := other.(Word32Value)
	if !ok {
		return false
	}
	return v == otherWord32
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord32 (1 byte)
// - uint32 value encoded in big-endian (4 bytes)
func (v Word32Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertWord32(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word32Value {
	return NewWord32Value(
		memoryGauge,
		func() uint32 {
			return ConvertWord[uint32](memoryGauge, value, locationRange)
		},
	)
}

func (v Word32Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v | o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v ^ o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v & o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v << o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v >> o)
	}

	return NewWord32Value(interpreter, valueGetter)
}

func (v Word32Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word32Type, locationRange)
}

func (Word32Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word32Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Word32Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word32Value) IsStorable() bool {
	return true
}

func (v Word32Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word32Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word32Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word32Value) Transfer(
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

func (v Word32Value) Clone(_ *Interpreter) Value {
	return v
}

func (Word32Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word32Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v Word32Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word32Value) ChildStorables() []atree.Storable {
	return nil
}

// Word64Value

type Word64Value uint64

var _ Value = Word64Value(0)
var _ atree.Storable = Word64Value(0)
var _ NumberValue = Word64Value(0)
var _ IntegerValue = Word64Value(0)
var _ EquatableValue = Word64Value(0)
var _ ComparableValue = Word64Value(0)
var _ HashableValue = Word64Value(0)
var _ MemberAccessibleValue = Word64Value(0)

const word64Size = int(unsafe.Sizeof(Word64Value(0)))

var word64MemoryUsage = common.NewNumberMemoryUsage(word64Size)

func NewWord64Value(gauge common.MemoryGauge, valueGetter func() uint64) Word64Value {
	common.UseMemory(gauge, word64MemoryUsage)

	return NewUnmeteredWord64Value(valueGetter())
}

func NewUnmeteredWord64Value(value uint64) Word64Value {
	return Word64Value(value)
}

// NOTE: important, do *NOT* remove:
// Word64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
var _ BigNumberValue = Word64Value(0)

func (Word64Value) isValue() {}

func (v Word64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord64Value(interpreter, v)
}

func (Word64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord64)
}

func (Word64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word64Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word64Value) ToInt(locationRange LocationRange) int {
	if v > math.MaxInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v)
}

func (v Word64Value) ByteLength() int {
	return 8
}

// ToBigInt
//
// NOTE: important, do *NOT* remove:
// Word64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
func (v Word64Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).SetUint64(uint64(v))
}

func (v Word64Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v + o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingPlus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v - o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingMinus(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v % o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v * o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingMul(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v / o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) SaturatingDiv(*Interpreter, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Word64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Word64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Word64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Word64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherWord64, ok := other.(Word64Value)
	if !ok {
		return false
	}
	return v == otherWord64
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v Word64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertWord64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Word64Value {
	return NewWord64Value(
		memoryGauge,
		func() uint64 {
			return ConvertWord[uint64](memoryGauge, value, locationRange)
		},
	)
}

func (v Word64Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v | o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v ^ o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v & o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v << o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v >> o)
	}

	return NewWord64Value(interpreter, valueGetter)
}

func (v Word64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word64Type, locationRange)
}

func (Word64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Word64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word64Value) IsStorable() bool {
	return true
}

func (v Word64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word64Value) Transfer(
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

func (v Word64Value) Clone(_ *Interpreter) Value {
	return v
}

func (v Word64Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (Word64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word64Value) ChildStorables() []atree.Storable {
	return nil
}

// Word128Value

type Word128Value struct {
	BigInt *big.Int
}

func NewWord128ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Word128Value {
	return NewWord128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Word128MemoryUsage = common.NewBigIntMemoryUsage(16)

func NewWord128ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Word128Value {
	common.UseMemory(memoryGauge, Word128MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredWord128ValueFromBigInt(value)
}

func NewUnmeteredWord128ValueFromUint64(value uint64) Word128Value {
	return NewUnmeteredWord128ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUnmeteredWord128ValueFromBigInt(value *big.Int) Word128Value {
	return Word128Value{
		BigInt: value,
	}
}

var _ Value = Word128Value{}
var _ atree.Storable = Word128Value{}
var _ NumberValue = Word128Value{}
var _ IntegerValue = Word128Value{}
var _ EquatableValue = Word128Value{}
var _ ComparableValue = Word128Value{}
var _ HashableValue = Word128Value{}
var _ MemberAccessibleValue = Word128Value{}

func (Word128Value) isValue() {}

func (v Word128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord128Value(interpreter, v)
}

func (Word128Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word128Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord128)
}

func (Word128Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word128Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Word128Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Word128Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Word128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Word128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word128Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word128Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and wrap around in case of overflow.
			//
			// Note that since v and o are both in the range [0, 2**128 - 1),
			// their sum will be in range [0, 2*(2**128 - 1)).
			// Hence it is sufficient to subtract 2**128 to wrap around.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.Word128TypeMaxIntBig) > 0 {
				sum.Sub(sum, sema.Word128TypeMaxIntPlusOneBig)
			}
			return sum
		},
	)
}

func (v Word128Value) SaturatingPlus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and wrap around in case of underflow.
			//
			// Note that since v and o are both in the range [0, 2**128 - 1),
			// their difference will be in range [-(2**128 - 1), 2**128 - 1).
			// Hence it is sufficient to add 2**128 to wrap around.
			//
			// If Go gains a native uint128 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Sign() < 0 {
				diff.Add(diff, sema.Word128TypeMaxIntPlusOneBig)
			}
			return diff
		},
	)
}

func (v Word128Value) SaturatingMinus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.Word128TypeMaxIntBig) > 0 {
				res.Mod(res, sema.Word128TypeMaxIntPlusOneBig)
			}
			return res
		},
	)
}

func (v Word128Value) SaturatingMul(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)

}

func (v Word128Value) SaturatingDiv(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word128Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Word128Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Word128Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Word128Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Word128Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Word128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord128 (1 byte)
// - big int encoded in big endian (n bytes)
func (v Word128Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeWord128)
	copy(buffer[1:], b)
	return buffer
}

func ConvertWord128(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
	return NewWord128ValueFromBigInt(
		memoryGauge,
		func() *big.Int {

			var v *big.Int

			switch value := value.(type) {
			case BigNumberValue:
				v = value.ToBigInt(memoryGauge)

			case NumberValue:
				v = big.NewInt(int64(value.ToInt(locationRange)))

			default:
				panic(errors.NewUnreachableError())
			}

			if v.Cmp(sema.Word128TypeMaxIntBig) > 0 || v.Sign() < 0 {
				// When Sign() < 0, Mod will add sema.Word128TypeMaxIntPlusOneBig
				// to ensure the range is [0, sema.Word128TypeMaxIntPlusOneBig)
				v.Mod(v, sema.Word128TypeMaxIntPlusOneBig)
			}

			return v
		},
	)
}

func (v Word128Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v Word128Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)

}

func (v Word128Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word128Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(interpreter),
			RightType: other.StaticType(interpreter),
		})
	}

	return NewWord128ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word128Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word128Type, locationRange)
}

func (Word128Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word128Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word128Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Word128Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word128Value) IsStorable() bool {
	return true
}

func (v Word128Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word128Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word128Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word128Value) Transfer(
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

func (v Word128Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredWord128ValueFromBigInt(v.BigInt)
}

func (Word128Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word128Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Word128Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word128Value) ChildStorables() []atree.Storable {
	return nil
}

// Word256Value

type Word256Value struct {
	BigInt *big.Int
}

func NewWord256ValueFromUint64(memoryGauge common.MemoryGauge, value int64) Word256Value {
	return NewWord256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {
			return new(big.Int).SetInt64(value)
		},
	)
}

var Word256MemoryUsage = common.NewBigIntMemoryUsage(32)

func NewWord256ValueFromBigInt(memoryGauge common.MemoryGauge, bigIntConstructor func() *big.Int) Word256Value {
	common.UseMemory(memoryGauge, Word256MemoryUsage)
	value := bigIntConstructor()
	return NewUnmeteredWord256ValueFromBigInt(value)
}

func NewUnmeteredWord256ValueFromUint64(value uint64) Word256Value {
	return NewUnmeteredWord256ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUnmeteredWord256ValueFromBigInt(value *big.Int) Word256Value {
	return Word256Value{
		BigInt: value,
	}
}

var _ Value = Word256Value{}
var _ atree.Storable = Word256Value{}
var _ NumberValue = Word256Value{}
var _ IntegerValue = Word256Value{}
var _ EquatableValue = Word256Value{}
var _ ComparableValue = Word256Value{}
var _ HashableValue = Word256Value{}
var _ MemberAccessibleValue = Word256Value{}

func (Word256Value) isValue() {}

func (v Word256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord256Value(interpreter, v)
}

func (Word256Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Word256Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeWord256)
}

func (Word256Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Word256Value) ToInt(locationRange LocationRange) int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}
	return int(v.BigInt.Int64())
}

func (v Word256Value) ByteLength() int {
	return common.BigIntByteLength(v.BigInt)
}

func (v Word256Value) ToBigInt(memoryGauge common.MemoryGauge) *big.Int {
	common.UseMemory(memoryGauge, common.NewBigIntMemoryUsage(v.ByteLength()))
	return new(big.Int).Set(v.BigInt)
}

func (v Word256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Word256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word256Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Word256Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			sum := new(big.Int)
			sum.Add(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just add and wrap around in case of overflow.
			//
			// Note that since v and o are both in the range [0, 2**256 - 1),
			// their sum will be in range [0, 2*(2**256 - 1)).
			// Hence it is sufficient to subtract 2**256 to wrap around.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//  if sum < v {
			//      ...
			//  }
			//
			if sum.Cmp(sema.Word256TypeMaxIntBig) > 0 {
				sum.Sub(sum, sema.Word256TypeMaxIntPlusOneBig)
			}
			return sum
		},
	)
}

func (v Word256Value) SaturatingPlus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			diff := new(big.Int)
			diff.Sub(v.BigInt, o.BigInt)
			// Given that this value is backed by an arbitrary size integer,
			// we can just subtract and wrap around in case of underflow.
			//
			// Note that since v and o are both in the range [0, 2**256 - 1),
			// their difference will be in range [-(2**256 - 1), 2**256 - 1).
			// Hence it is sufficient to add 2**256 to wrap around.
			//
			// If Go gains a native uint256 type and we switch this value
			// to be based on it, then we need to follow INT30-C:
			//
			//   if diff > v {
			// 	     ...
			//   }
			//
			if diff.Sign() < 0 {
				diff.Add(diff, sema.Word256TypeMaxIntPlusOneBig)
			}
			return diff
		},
	)
}

func (v Word256Value) SaturatingMinus(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Rem(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			res.Mul(v.BigInt, o.BigInt)
			if res.Cmp(sema.Word256TypeMaxIntBig) > 0 {
				res.Mod(res, sema.Word256TypeMaxIntPlusOneBig)
			}
			return res
		},
	)
}

func (v Word256Value) SaturatingMul(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Cmp(res) == 0 {
				panic(DivisionByZeroError{
					LocationRange: locationRange,
				})
			}
			return res.Div(v.BigInt, o.BigInt)
		},
	)

}

func (v Word256Value) SaturatingDiv(_ *Interpreter, _ NumberValue, _ LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word256Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == -1)
}

func (v Word256Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp <= 0)
}

func (v Word256Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp == 1)
}

func (v Word256Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return AsBoolValue(cmp >= 0)
}

func (v Word256Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherInt, ok := other.(Word256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord256 (1 byte)
// - big int encoded in big endian (n bytes)
func (v Word256Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	b := UnsignedBigIntToBigEndianBytes(v.BigInt)

	length := 1 + len(b)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeWord256)
	copy(buffer[1:], b)
	return buffer
}

func ConvertWord256(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Value {
	return NewWord256ValueFromBigInt(
		memoryGauge,
		func() *big.Int {

			var v *big.Int

			switch value := value.(type) {
			case BigNumberValue:
				v = value.ToBigInt(memoryGauge)

			case NumberValue:
				v = big.NewInt(int64(value.ToInt(locationRange)))

			default:
				panic(errors.NewUnreachableError())
			}

			if v.Cmp(sema.Word256TypeMaxIntBig) > 0 || v.Sign() < 0 {
				// When Sign() < 0, Mod will add sema.Word256TypeMaxIntPlusOneBig
				// to ensure the range is [0, sema.Word256TypeMaxIntPlusOneBig)
				v.Mod(v, sema.Word256TypeMaxIntPlusOneBig)
			}

			return v
		},
	)
}

func (v Word256Value) BitwiseOr(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Or(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) BitwiseXor(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.Xor(v.BigInt, o.BigInt)
		},
	)
}

func (v Word256Value) BitwiseAnd(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			return res.And(v.BigInt, o.BigInt)
		},
	)

}

func (v Word256Value) BitwiseLeftShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word256Value) BitwiseRightShift(interpreter *Interpreter, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return NewWord256ValueFromBigInt(
		interpreter,
		func() *big.Int {
			res := new(big.Int)
			if o.BigInt.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}
			if !o.BigInt.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}
			return res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
		},
	)
}

func (v Word256Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Word256Type, locationRange)
}

func (Word256Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word256Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word256Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Word256Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Word256Value) IsStorable() bool {
	return true
}

func (v Word256Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Word256Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Word256Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Word256Value) Transfer(
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

func (v Word256Value) Clone(_ *Interpreter) Value {
	return NewUnmeteredWord256ValueFromBigInt(v.BigInt)
}

func (Word256Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Word256Value) ByteSize() uint32 {
	return cborTagSize + getBigIntCBORSize(v.BigInt)
}

func (v Word256Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word256Value) ChildStorables() []atree.Storable {
	return nil
}

// FixedPointValue is a fixed-point number value
type FixedPointValue interface {
	NumberValue
	IntegerPart() NumberValue
	Scale() int
}

// Fix64Value
type Fix64Value int64

const Fix64MaxValue = math.MaxInt64

const fix64Size = int(unsafe.Sizeof(Fix64Value(0)))

var fix64MemoryUsage = common.NewNumberMemoryUsage(fix64Size)

func NewFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() int64, locationRange LocationRange) Fix64Value {
	common.UseMemory(gauge, fix64MemoryUsage)
	return NewUnmeteredFix64ValueWithInteger(constructor(), locationRange)
}

func NewUnmeteredFix64ValueWithInteger(integer int64, locationRange LocationRange) Fix64Value {

	if integer < sema.Fix64TypeMinInt {
		panic(UnderflowError{
			LocationRange: locationRange,
		})
	}

	if integer > sema.Fix64TypeMaxInt {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUnmeteredFix64Value(integer * sema.Fix64Factor)
}

func NewFix64Value(gauge common.MemoryGauge, valueGetter func() int64) Fix64Value {
	common.UseMemory(gauge, fix64MemoryUsage)
	return NewUnmeteredFix64Value(valueGetter())
}

func NewUnmeteredFix64Value(integer int64) Fix64Value {
	return Fix64Value(integer)
}

var _ Value = Fix64Value(0)
var _ atree.Storable = Fix64Value(0)
var _ NumberValue = Fix64Value(0)
var _ FixedPointValue = Fix64Value(0)
var _ EquatableValue = Fix64Value(0)
var _ ComparableValue = Fix64Value(0)
var _ HashableValue = Fix64Value(0)
var _ MemberAccessibleValue = Fix64Value(0)

func (Fix64Value) isValue() {}

func (v Fix64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitFix64Value(interpreter, v)
}

func (Fix64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (Fix64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeFix64)
}

func (Fix64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v Fix64Value) String() string {
	return format.Fix64(int64(v))
}

func (v Fix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Fix64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v Fix64Value) ToInt(_ LocationRange) int {
	return int(v / sema.Fix64Factor)
}

func (v Fix64Value) Negate(interpreter *Interpreter, locationRange LocationRange) NumberValue {
	// INT32-C
	if v == math.MinInt64 {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return int64(-v)
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		return safeAddInt64(int64(v), int64(o), locationRange)
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if (o > 0) && (v > (math.MaxInt64 - o)) {
			return math.MaxInt64
		} else if (o < 0) && (v < (math.MinInt64 - o)) {
			return math.MinInt64
		}
		return int64(v + o)
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt64 + o)) {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		} else if (o < 0) && (v > (math.MaxInt64 + o)) {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}

		return int64(v - o)
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() int64 {
		// INT32-C
		if (o > 0) && (v < (math.MinInt64 + o)) {
			return math.MinInt64
		} else if (o < 0) && (v > (math.MaxInt64 + o)) {
			return math.MaxInt64
		}
		return int64(v - o)
	}

	return NewFix64Value(interpreter, valueGetter)
}

var minInt64Big = big.NewInt(math.MinInt64)
var maxInt64Big = big.NewInt(math.MaxInt64)

func (v Fix64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if result.Cmp(minInt64Big) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if result.Cmp(maxInt64Big) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return result.Int64()
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if result.Cmp(minInt64Big) < 0 {
			return math.MinInt64
		} else if result.Cmp(maxInt64Big) > 0 {
			return math.MaxInt64
		}

		return result.Int64()
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, sema.Fix64FactorBig)
		result.Div(result, b)

		if result.Cmp(minInt64Big) < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		} else if result.Cmp(maxInt64Big) > 0 {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return result.Int64()
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	valueGetter := func() int64 {
		result := new(big.Int).Mul(a, sema.Fix64FactorBig)
		result.Div(result, b)

		if result.Cmp(minInt64Big) < 0 {
			return math.MinInt64
		} else if result.Cmp(maxInt64Big) > 0 {
			return math.MaxInt64
		}

		return result.Int64()
	}

	return NewFix64Value(interpreter, valueGetter)
}

func (v Fix64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// v - int(v/o) * o
	quotient, ok := v.Div(interpreter, o, locationRange).(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	truncatedQuotient := NewFix64Value(
		interpreter,
		func() int64 {
			return (int64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
		},
	)

	return v.Minus(
		interpreter,
		truncatedQuotient.Mul(interpreter, o, locationRange),
		locationRange,
	)
}

func (v Fix64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v Fix64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v Fix64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v Fix64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v Fix64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherFix64, ok := other.(Fix64Value)
	if !ok {
		return false
	}
	return v == otherFix64
}

// HashInput returns a byte slice containing:
// - HashInputTypeFix64 (1 byte)
// - int64 value encoded in big-endian (8 bytes)
func (v Fix64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertFix64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) Fix64Value {
	switch value := value.(type) {
	case Fix64Value:
		return value

	case UFix64Value:
		if value > Fix64MaxValue {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}
		return NewFix64Value(
			memoryGauge,
			func() int64 {
				return int64(value)
			},
		)

	case BigNumberValue:
		converter := func() int64 {
			v := value.ToBigInt(memoryGauge)

			// First, check if the value is at least in the int64 range.
			// The integer range for Fix64 is smaller, but this test at least
			// allows us to call `v.Int64()` safely.

			if !v.IsInt64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}

			return v.Int64()
		}

		// Now check that the integer value fits the range of Fix64
		return NewFix64ValueWithInteger(memoryGauge, converter, locationRange)

	case NumberValue:
		// Check that the integer value fits the range of Fix64
		return NewFix64ValueWithInteger(
			memoryGauge,
			func() int64 {
				return int64(value.ToInt(locationRange))
			},
			locationRange,
		)

	default:
		panic(fmt.Sprintf("can't convert Fix64: %s", value))
	}
}

func (v Fix64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.Fix64Type, locationRange)
}

func (Fix64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Fix64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Fix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Fix64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (Fix64Value) IsStorable() bool {
	return true
}

func (v Fix64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (Fix64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (Fix64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v Fix64Value) Transfer(
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

func (v Fix64Value) Clone(_ *Interpreter) Value {
	return v
}

func (Fix64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Fix64Value) ByteSize() uint32 {
	return cborTagSize + getIntCBORSize(int64(v))
}

func (v Fix64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Fix64Value) ChildStorables() []atree.Storable {
	return nil
}

func (v Fix64Value) IntegerPart() NumberValue {
	return UInt64Value(v / sema.Fix64Factor)
}

func (Fix64Value) Scale() int {
	return sema.Fix64Scale
}

// UFix64Value
type UFix64Value uint64

const UFix64MaxValue = math.MaxUint64

const ufix64Size = int(unsafe.Sizeof(UFix64Value(0)))

var ufix64MemoryUsage = common.NewNumberMemoryUsage(ufix64Size)

func NewUFix64ValueWithInteger(gauge common.MemoryGauge, constructor func() uint64, locationRange LocationRange) UFix64Value {
	common.UseMemory(gauge, ufix64MemoryUsage)
	return NewUnmeteredUFix64ValueWithInteger(constructor(), locationRange)
}

func NewUnmeteredUFix64ValueWithInteger(integer uint64, locationRange LocationRange) UFix64Value {
	if integer > sema.UFix64TypeMaxInt {
		panic(OverflowError{
			LocationRange: locationRange,
		})
	}

	return NewUnmeteredUFix64Value(integer * sema.Fix64Factor)
}

func NewUFix64Value(gauge common.MemoryGauge, constructor func() uint64) UFix64Value {
	common.UseMemory(gauge, ufix64MemoryUsage)
	return NewUnmeteredUFix64Value(constructor())
}

func NewUnmeteredUFix64Value(integer uint64) UFix64Value {
	return UFix64Value(integer)
}

var _ Value = UFix64Value(0)
var _ atree.Storable = UFix64Value(0)
var _ NumberValue = UFix64Value(0)
var _ FixedPointValue = Fix64Value(0)
var _ EquatableValue = UFix64Value(0)
var _ ComparableValue = UFix64Value(0)
var _ HashableValue = UFix64Value(0)
var _ MemberAccessibleValue = UFix64Value(0)

func (UFix64Value) isValue() {}

func (v UFix64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUFix64Value(interpreter, v)
}

func (UFix64Value) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (UFix64Value) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeUFix64)
}

func (UFix64Value) IsImportable(_ *Interpreter) bool {
	return true
}

func (v UFix64Value) String() string {
	return format.UFix64(uint64(v))
}

func (v UFix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UFix64Value) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(
		memoryGauge,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(memoryGauge, v),
		),
	)
	return v.String()
}

func (v UFix64Value) ToInt(_ LocationRange) int {
	return int(v / sema.Fix64Factor)
}

func (v UFix64Value) Negate(*Interpreter, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Plus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return safeAddUint64(uint64(v), uint64(o), locationRange)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingPlus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingAddFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		sum := v + o
		// INT30-C
		if sum < v {
			return math.MaxUint64
		}
		return uint64(sum)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) Minus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		diff := v - o

		// INT30-C
		if diff > v {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return uint64(diff)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingMinus(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		diff := v - o

		// INT30-C
		if diff > v {
			return 0
		}
		return uint64(diff)
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) Mul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	valueGetter := func() uint64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if !result.IsUint64() {
			panic(OverflowError{
				LocationRange: locationRange,
			})
		}

		return result.Uint64()
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingMul(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName:  sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	valueGetter := func() uint64 {
		result := new(big.Int).Mul(a, b)
		result.Div(result, sema.Fix64FactorBig)

		if !result.IsUint64() {
			return math.MaxUint64
		}

		return result.Uint64()
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) Div(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	valueGetter := func() uint64 {
		result := new(big.Int).Mul(a, sema.Fix64FactorBig)
		result.Div(result, b)

		return result.Uint64()
	}

	return NewUFix64Value(interpreter, valueGetter)
}

func (v UFix64Value) SaturatingDiv(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName:  sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:      v.StaticType(interpreter),
				RightType:     other.StaticType(interpreter),
				LocationRange: locationRange,
			})
		}
	}()

	return v.Div(interpreter, other, locationRange)
}

func (v UFix64Value) Mod(interpreter *Interpreter, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	// v - int(v/o) * o
	quotient, ok := v.Div(interpreter, o, locationRange).(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	truncatedQuotient := NewUFix64Value(
		interpreter,
		func() uint64 {
			return (uint64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
		},
	)

	return v.Minus(
		interpreter,
		truncatedQuotient.Mul(interpreter, o, locationRange),
		locationRange,
	)
}

func (v UFix64Value) Less(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v < o)
}

func (v UFix64Value) LessEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v <= o)
}

func (v UFix64Value) Greater(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v > o)
}

func (v UFix64Value) GreaterEqual(interpreter *Interpreter, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(interpreter),
			RightType:     other.StaticType(interpreter),
			LocationRange: locationRange,
		})
	}

	return AsBoolValue(v >= o)
}

func (v UFix64Value) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherUFix64, ok := other.(UFix64Value)
	if !ok {
		return false
	}
	return v == otherUFix64
}

// HashInput returns a byte slice containing:
// - HashInputTypeUFix64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v UFix64Value) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertUFix64(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) UFix64Value {
	switch value := value.(type) {
	case UFix64Value:
		return value

	case Fix64Value:
		if value < 0 {
			panic(UnderflowError{
				LocationRange: locationRange,
			})
		}
		return NewUFix64Value(
			memoryGauge,
			func() uint64 {
				return uint64(value)
			},
		)

	case BigNumberValue:
		converter := func() uint64 {
			v := value.ToBigInt(memoryGauge)

			if v.Sign() < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}

			// First, check if the value is at least in the uint64 range.
			// The integer range for UFix64 is smaller, but this test at least
			// allows us to call `v.UInt64()` safely.

			if !v.IsUint64() {
				panic(OverflowError{
					LocationRange: locationRange,
				})
			}

			return v.Uint64()
		}

		// Now check that the integer value fits the range of UFix64
		return NewUFix64ValueWithInteger(memoryGauge, converter, locationRange)

	case NumberValue:
		converter := func() uint64 {
			v := value.ToInt(locationRange)
			if v < 0 {
				panic(UnderflowError{
					LocationRange: locationRange,
				})
			}

			return uint64(v)
		}

		// Check that the integer value fits the range of UFix64
		return NewUFix64ValueWithInteger(memoryGauge, converter, locationRange)

	default:
		panic(fmt.Sprintf("can't convert to UFix64: %s", value))
	}
}

func (v UFix64Value) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	return getNumberValueMember(interpreter, v, name, sema.UFix64Type, locationRange)
}

func (UFix64Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UFix64Value) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UFix64Value) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (UFix64Value) IsStorable() bool {
	return true
}

func (v UFix64Value) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (UFix64Value) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (UFix64Value) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v UFix64Value) Transfer(
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

func (v UFix64Value) Clone(_ *Interpreter) Value {
	return v
}

func (UFix64Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UFix64Value) ByteSize() uint32 {
	return cborTagSize + getUintCBORSize(uint64(v))
}

func (v UFix64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (UFix64Value) ChildStorables() []atree.Storable {
	return nil
}

func (v UFix64Value) IntegerPart() NumberValue {
	return UInt64Value(v / sema.Fix64Factor)
}

func (UFix64Value) Scale() int {
	return sema.Fix64Scale
}

// CompositeValue

type FunctionOrderedMap = orderedmap.OrderedMap[string, FunctionValue]

type CompositeValue struct {
	Location        common.Location
	staticType      StaticType
	Stringer        func(gauge common.MemoryGauge, value *CompositeValue, seenReferences SeenReferences) string
	injectedFields  map[string]Value
	computedFields  map[string]ComputedField
	NestedVariables map[string]*Variable
	Functions       *FunctionOrderedMap
	dictionary      *atree.OrderedMap
	typeID          TypeID

	// attachments also have a reference to their base value. This field is set in three cases:
	// 1) when an attachment `A` is accessed off `v` using `v[A]`, this is set to `&v`
	// 2) When a resource `r`'s destructor is invoked, all of `r`'s attachments' destructors will also run, and
	//    have their `base` fields set to `&r`
	// 3) When a value is transferred, this field is copied between its attachments
	base                *CompositeValue
	QualifiedIdentifier string
	Kind                common.CompositeKind
	isDestroyed         bool
}

type ComputedField func(*Interpreter, LocationRange) Value

type CompositeField struct {
	Value Value
	Name  string
}

const unrepresentableNamePrefix = "$"
const resourceDefaultDestroyEventPrefix = ast.ResourceDestructionDefaultEventName + unrepresentableNamePrefix

var _ TypeIndexableValue = &CompositeValue{}

func NewCompositeField(memoryGauge common.MemoryGauge, name string, value Value) CompositeField {
	common.UseMemory(memoryGauge, common.CompositeFieldMemoryUsage)
	return NewUnmeteredCompositeField(name, value)
}

func NewUnmeteredCompositeField(name string, value Value) CompositeField {
	return CompositeField{
		Name:  name,
		Value: value,
	}
}

func NewCompositeValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	fields []CompositeField,
	address common.Address,
) *CompositeValue {

	interpreter.ReportComputation(common.ComputationKindCreateCompositeValue, 1)

	config := interpreter.SharedState.Config

	var v *CompositeValue

	if config.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			owner := v.GetOwner().String()
			typeID := string(v.TypeID())
			kind := v.Kind.String()

			interpreter.reportCompositeValueConstructTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() *atree.OrderedMap {
		dictionary, err := atree.NewMap(
			config.Storage,
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			NewCompositeTypeInfo(
				interpreter,
				location,
				qualifiedIdentifier,
				kind,
			),
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return dictionary
	}

	typeInfo := NewCompositeTypeInfo(
		interpreter,
		location,
		qualifiedIdentifier,
		kind,
	)

	v = newCompositeValueFromConstructor(interpreter, uint64(len(fields)), typeInfo, constructor)

	for _, field := range fields {
		v.SetMember(
			interpreter,
			locationRange,
			field.Name,
			field.Value,
		)
	}

	return v
}

func newCompositeValueFromConstructor(
	gauge common.MemoryGauge,
	count uint64,
	typeInfo compositeTypeInfo,
	constructor func() *atree.OrderedMap,
) *CompositeValue {

	elementOverhead, dataUse, metaDataUse :=
		common.NewAtreeMapMemoryUsages(count, 0)
	common.UseMemory(gauge, elementOverhead)
	common.UseMemory(gauge, dataUse)
	common.UseMemory(gauge, metaDataUse)

	return newCompositeValueFromAtreeMap(
		gauge,
		typeInfo,
		constructor(),
	)
}

func newCompositeValueFromAtreeMap(
	gauge common.MemoryGauge,
	typeInfo compositeTypeInfo,
	atreeOrderedMap *atree.OrderedMap,
) *CompositeValue {

	common.UseMemory(gauge, common.CompositeValueBaseMemoryUsage)

	return &CompositeValue{
		dictionary:          atreeOrderedMap,
		Location:            typeInfo.location,
		QualifiedIdentifier: typeInfo.qualifiedIdentifier,
		Kind:                typeInfo.kind,
	}
}

var _ Value = &CompositeValue{}
var _ EquatableValue = &CompositeValue{}
var _ HashableValue = &CompositeValue{}
var _ MemberAccessibleValue = &CompositeValue{}
var _ ReferenceTrackedResourceKindedValue = &CompositeValue{}
var _ ContractValue = &CompositeValue{}

func (*CompositeValue) isValue() {}

func (v *CompositeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitCompositeValue(interpreter, v)
	if !descend {
		return
	}

	v.ForEachField(interpreter, func(_ string, value Value) (resume bool) {
		value.Accept(interpreter, visitor)

		// continue iteration
		return true
	})
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed field or functions!
func (v *CompositeValue) Walk(interpreter *Interpreter, walkChild func(Value)) {
	v.ForEachField(interpreter, func(_ string, value Value) (resume bool) {
		walkChild(value)

		// continue iteration
		return true
	})
}

func (v *CompositeValue) StaticType(interpreter *Interpreter) StaticType {
	if v.staticType == nil {
		// NOTE: Instead of using NewCompositeStaticType, which always generates the type ID,
		// use the TypeID accessor, which may return an already computed type ID
		v.staticType = NewCompositeStaticType(
			interpreter,
			v.Location,
			v.QualifiedIdentifier,
			v.TypeID(),
		)
	}
	return v.staticType
}

func (v *CompositeValue) IsImportable(inter *Interpreter) bool {
	// Check type is importable
	staticType := v.StaticType(inter)
	semaType := inter.MustConvertStaticToSemaType(staticType)
	if !semaType.IsImportable(map[*sema.Member]bool{}) {
		return false
	}

	// Check all field values are importable
	importable := true
	v.ForEachField(inter, func(_ string, value Value) (resume bool) {
		if !value.IsImportable(inter) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return importable
}

func (v *CompositeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func resourceDefaultDestroyEventName(t sema.ContainerType) string {
	return resourceDefaultDestroyEventPrefix + string(t.ID())
}

// get all the default destroy event constructors associated with this composite value.
// note that there can be more than one in the case where a resource inherits from an interface
// that also defines a default destroy event. When that composite is destroyed, all of these
// events will need to be emitted.
func (v *CompositeValue) defaultDestroyEventConstructors() (constructors []FunctionValue) {
	if v.Functions == nil {
		return
	}
	v.Functions.Foreach(func(name string, f FunctionValue) {
		if strings.HasPrefix(name, resourceDefaultDestroyEventPrefix) {
			constructors = append(constructors, f)
		}
	})
	return
}

func (v *CompositeValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyCompositeValue, 1)

	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {

			interpreter.reportCompositeValueDestroyTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	// before actually performing the destruction (i.e. so that any fields are still available),
	// compute the default arguments of the default destruction events (if any exist). However,
	// wait until after the destruction completes to actually emit the events, so that the correct order
	// is preserved and nested resource destroy events happen first

	// default destroy event constructors are encoded as functions on the resource (with an unrepresentable name)
	// so that we can leverage existing atree encoding and decoding. However, we need to make sure functions are initialized
	// if the composite was recently loaded from storage
	if v.Functions == nil {
		v.Functions = interpreter.SharedState.typeCodes.CompositeCodes[v.TypeID()].CompositeFunctions
	}
	for _, constructor := range v.defaultDestroyEventConstructors() {

		// pass the container value to the creation of the default event as an implicit argument, so that
		// its fields are accessible in the body of the event constructor
		eventConstructorInvocation := NewInvocation(
			interpreter,
			nil,
			nil,
			nil,
			[]Value{v},
			[]sema.Type{},
			nil,
			locationRange,
		)

		event := constructor.invoke(eventConstructorInvocation).(*CompositeValue)
		eventType := interpreter.MustSemaTypeOfValue(event).(*sema.CompositeType)

		// emit the event once destruction is complete
		defer interpreter.emitEvent(event, eventType, locationRange)
	}

	storageID := v.StorageID()

	interpreter.withResourceDestruction(
		storageID,
		locationRange,
		func() {
			interpreter = v.getInterpreter(interpreter)

			// destroy every nested resource in this composite; note that this iteration includes attachments
			v.ForEachField(interpreter, func(_ string, fieldValue Value) bool {
				if compositeFieldValue, ok := fieldValue.(*CompositeValue); ok && compositeFieldValue.Kind == common.CompositeKindAttachment {
					compositeFieldValue.setBaseValue(interpreter, v)
				}
				maybeDestroy(interpreter, locationRange, fieldValue)
				return true
			})
		},
	)

	v.isDestroyed = true

	interpreter.invalidateReferencedResources(v, true)

	v.dictionary = nil
}

func (v *CompositeValue) getBuiltinMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {

	switch name {
	case sema.ResourceOwnerFieldName:
		if v.Kind == common.CompositeKindResource {
			return v.OwnerValue(interpreter, locationRange)
		}
	case sema.CompositeForEachAttachmentFunctionName:
		if v.Kind.SupportsAttachments() {
			return v.forEachAttachmentFunction(interpreter, locationRange)
		}
	}

	return nil
}

func (v *CompositeValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueGetMemberTrace(
				owner,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	if builtin := v.getBuiltinMember(interpreter, locationRange, name); builtin != nil {
		return builtin
	}

	// Give computed fields precedence over stored fields for built-in types
	if v.Location == nil {
		if computedField := v.GetComputedField(interpreter, locationRange, name); computedField != nil {
			return computedField
		}
	}

	if field := v.GetField(interpreter, locationRange, name); field != nil {
		return field
	}

	if v.NestedVariables != nil {
		variable, ok := v.NestedVariables[name]
		if ok {
			return variable.GetValue()
		}
	}

	interpreter = v.getInterpreter(interpreter)

	// Dynamically link in the computed fields, injected fields, and functions

	if computedField := v.GetComputedField(interpreter, locationRange, name); computedField != nil {
		return computedField
	}

	if injectedField := v.GetInjectedField(interpreter, name); injectedField != nil {
		return injectedField
	}

	if function := v.GetFunction(interpreter, locationRange, name); function != nil {
		return function
	}

	return nil
}

func (v *CompositeValue) checkInvalidatedResourceUse(interpreter *Interpreter, locationRange LocationRange) {
	if v.isDestroyed || v.IsStaleResource(interpreter) {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (v *CompositeValue) IsStaleResource(inter *Interpreter) bool {
	return v.dictionary == nil && v.IsResourceKinded(inter)
}

func (v *CompositeValue) getInterpreter(interpreter *Interpreter) *Interpreter {

	// Get the correct interpreter. The program code might need to be loaded.
	// NOTE: standard library values have no location

	location := v.Location

	if location == nil || interpreter.Location == location {
		return interpreter
	}

	return interpreter.EnsureLoaded(v.Location)
}

func (v *CompositeValue) GetComputedFields(interpreter *Interpreter) map[string]ComputedField {
	if v.computedFields == nil {
		v.computedFields = interpreter.GetCompositeValueComputedFields(v)
	}
	return v.computedFields
}

func (v *CompositeValue) GetComputedField(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	computedFields := v.GetComputedFields(interpreter)

	computedField, ok := computedFields[name]
	if !ok {
		return nil
	}

	return computedField(interpreter, locationRange)
}

func (v *CompositeValue) GetInjectedField(interpreter *Interpreter, name string) Value {
	if v.injectedFields == nil {
		v.injectedFields = interpreter.GetCompositeValueInjectedFields(v)
	}

	value, ok := v.injectedFields[name]
	if !ok {
		return nil
	}

	return value
}

func (v *CompositeValue) GetFunction(interpreter *Interpreter, locationRange LocationRange, name string) FunctionValue {
	if v.Functions == nil {
		v.Functions = interpreter.GetCompositeValueFunctions(v, locationRange)
	}
	// if no functions were produced, the `Get` below will be nil
	if v.Functions == nil {
		return nil
	}

	function, present := v.Functions.Get(name)
	if !present {
		return nil
	}

	var base *EphemeralReferenceValue
	var self MemberAccessibleValue = v
	if v.Kind == common.CompositeKindAttachment {
		functionAccess := interpreter.getAccessOfMember(v, name)

		// with respect to entitlements, any access inside an attachment that is not an entitlement access
		// does not provide any entitlements to base and self
		// E.g. consider:
		//
		//    access(E) fun foo() {}
		//    access(self) fun bar() {
		//        self.foo()
		//    }
		//    access(all) fun baz() {
		//        self.bar()
		//    }
		//
		// clearly `bar` should be callable within `baz`, but we cannot allow `foo`
		// to be callable within `bar`, or it will be possible to access `E` entitled
		// methods on `base`
		if functionAccess.IsPrimitiveAccess() {
			functionAccess = sema.UnauthorizedAccess
		}
		base, self = attachmentBaseAndSelfValues(interpreter, functionAccess, v, locationRange)
	}
	return NewBoundFunctionValue(interpreter, function, &self, base, nil)
}

func (v *CompositeValue) OwnerValue(interpreter *Interpreter, locationRange LocationRange) OptionalValue {
	address := v.StorageAddress()

	if address == (atree.Address{}) {
		return NilOptionalValue
	}

	config := interpreter.SharedState.Config

	ownerAccount := config.AccountHandler(AddressValue(address))

	// Owner must be of `Account` type.
	interpreter.ExpectType(
		ownerAccount,
		sema.AccountType,
		locationRange,
	)

	reference := NewEphemeralReferenceValue(
		interpreter,
		UnauthorizedAccess,
		ownerAccount,
		sema.AccountType,
		locationRange,
	)

	return NewSomeValueNonCopying(interpreter, reference)
}

func (v *CompositeValue) RemoveMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {

	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueRemoveMemberTrace(
				owner,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	// No need to clean up storable for passed-in key value,
	// as atree never calls Storable()
	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		StringAtreeValue(name),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil
		}
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	// Key
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	storedValue := StoredValue(
		interpreter,
		existingValueStorable,
		config.Storage,
	)
	return storedValue.
		Transfer(
			interpreter,
			locationRange,
			atree.Address{},
			true,
			existingValueStorable,
			nil,
		)
}

func (v *CompositeValue) SetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
	value Value,
) bool {
	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	interpreter.enforceNotResourceDestruction(v.StorageID(), locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueSetMemberTrace(
				owner,
				typeID,
				kind,
				name,
				time.Since(startTime),
			)
		}()
	}

	address := v.StorageAddress()

	value = value.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		map[atree.StorageID]struct{}{
			v.StorageID(): {},
		},
	)

	existingStorable, err := v.dictionary.Set(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		NewStringAtreeValue(interpreter, name),
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	if existingStorable != nil {
		existingValue := StoredValue(interpreter, existingStorable, config.Storage)

		existingValue.DeepRemove(interpreter)

		interpreter.RemoveReferencedSlab(existingStorable)
		return true
	}

	return false
}

func (v *CompositeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *CompositeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences)
}

var emptyCompositeStringLen = len(format.Composite("", nil))

func (v *CompositeValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {

	if v.Stringer != nil {
		return v.Stringer(memoryGauge, v, seenReferences)
	}

	strLen := emptyCompositeStringLen

	var fields []CompositeField
	_ = v.dictionary.Iterate(func(key atree.Value, value atree.Value) (resume bool, err error) {
		field := NewCompositeField(
			memoryGauge,
			string(key.(StringAtreeValue)),
			MustConvertStoredValue(memoryGauge, value),
		)

		fields = append(fields, field)

		strLen += len(field.Name)

		return true, nil
	})

	typeId := string(v.TypeID())

	// bodyLen = len(fieldNames) + len(typeId) + (n times colon+space) + ((n-1) times comma+space)
	//         = len(fieldNames) + len(typeId) + 2n + 2n - 2
	//         = len(fieldNames) + len(typeId) + 4n - 2
	//
	// Since (-2) only occurs if its non-empty, ignore the (-2). i.e: overestimate
	// 		bodyLen = len(fieldNames) + len(typeId) + 4n
	//
	strLen = strLen + len(typeId) + len(fields)*4

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))

	return formatComposite(memoryGauge, typeId, fields, seenReferences)
}

func formatComposite(memoryGauge common.MemoryGauge, typeId string, fields []CompositeField, seenReferences SeenReferences) string {
	preparedFields := make(
		[]struct {
			Name  string
			Value string
		},
		0,
		len(fields),
	)

	for _, field := range fields {
		preparedFields = append(
			preparedFields,
			struct {
				Name  string
				Value string
			}{
				Name:  field.Name,
				Value: field.Value.MeteredString(memoryGauge, seenReferences),
			},
		)
	}

	return format.Composite(typeId, preparedFields)
}

func (v *CompositeValue) GetField(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	storable, err := v.dictionary.Get(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		StringAtreeValue(name),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil
		}
		panic(errors.NewExternalError(err))
	}

	return StoredValue(interpreter, storable, v.dictionary.Storage)
}

func (v *CompositeValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherComposite, ok := other.(*CompositeValue)
	if !ok {
		return false
	}

	if !v.StaticType(interpreter).Equal(otherComposite.StaticType(interpreter)) ||
		v.Kind != otherComposite.Kind ||
		v.dictionary.Count() != otherComposite.dictionary.Count() {

		return false
	}

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			return true
		}

		fieldName := string(key.(StringAtreeValue))

		// NOTE: Do NOT use an iterator, iteration order of fields may be different
		// (if stored in different account, as storage ID is used as hash seed)
		otherValue := otherComposite.GetField(interpreter, locationRange, fieldName)

		equatableValue, ok := MustConvertStoredValue(interpreter, value).(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, locationRange, otherValue) {
			return false
		}
	}
}

// HashInput returns a byte slice containing:
// - HashInputTypeEnum (1 byte)
// - type id (n bytes)
// - hash input of raw value field name (n bytes)
func (v *CompositeValue) HashInput(interpreter *Interpreter, locationRange LocationRange, scratch []byte) []byte {
	if v.Kind == common.CompositeKindEnum {
		typeID := v.TypeID()

		rawValue := v.GetField(interpreter, locationRange, sema.EnumRawValueFieldName)
		rawValueHashInput := rawValue.(HashableValue).
			HashInput(interpreter, locationRange, scratch)

		length := 1 + len(typeID) + len(rawValueHashInput)
		if length <= len(scratch) {
			// Copy rawValueHashInput first because
			// rawValueHashInput and scratch can point to the same underlying scratch buffer
			copy(scratch[1+len(typeID):], rawValueHashInput)

			scratch[0] = byte(HashInputTypeEnum)
			copy(scratch[1:], typeID)
			return scratch[:length]
		}

		buffer := make([]byte, length)
		buffer[0] = byte(HashInputTypeEnum)
		copy(buffer[1:], typeID)
		copy(buffer[1+len(typeID):], rawValueHashInput)
		return buffer
	}

	panic(errors.NewUnreachableError())
}

func (v *CompositeValue) TypeID() TypeID {
	if v.typeID == "" {
		v.typeID = common.NewTypeIDFromQualifiedName(nil, v.Location, v.QualifiedIdentifier)
	}
	return v.typeID
}

func (v *CompositeValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {
	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueConformsToStaticTypeTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	staticType := v.StaticType(interpreter).(*CompositeStaticType)

	semaType := interpreter.MustConvertStaticToSemaType(staticType)

	compositeType, ok := semaType.(*sema.CompositeType)
	if !ok ||
		v.Kind != compositeType.Kind ||
		v.TypeID() != compositeType.ID() {

		return false
	}

	if compositeType.Kind == common.CompositeKindAttachment {
		base := v.getBaseValue(interpreter, UnauthorizedAccess, locationRange).Value
		if base == nil || !base.ConformsToStaticType(interpreter, locationRange, results) {
			return false
		}
	}

	fieldsLen := int(v.dictionary.Count())

	computedFields := v.GetComputedFields(interpreter)
	if computedFields != nil {
		fieldsLen += len(computedFields)
	}

	// The composite might store additional fields
	// which are not statically declared in the composite type.
	if fieldsLen < len(compositeType.Fields) {
		return false
	}

	for _, fieldName := range compositeType.Fields {
		value := v.GetField(interpreter, locationRange, fieldName)
		if value == nil {
			if computedFields == nil {
				return false
			}

			fieldGetter, ok := computedFields[fieldName]
			if !ok {
				return false
			}

			value = fieldGetter(interpreter, locationRange)
		}

		member, ok := compositeType.Members.Get(fieldName)
		if !ok {
			return false
		}

		fieldStaticType := value.StaticType(interpreter)

		if !interpreter.IsSubTypeOfSemaType(fieldStaticType, member.TypeAnnotation.Type) {
			return false
		}

		if !value.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			return false
		}
	}

	return true
}

func (v *CompositeValue) IsStorable() bool {

	// Only structures, resources, enums, and contracts can be stored.
	// Contracts are not directly storable by programs,
	// but they are still stored in storage by the interpreter

	switch v.Kind {
	case common.CompositeKindStructure,
		common.CompositeKindResource,
		common.CompositeKindEnum,
		common.CompositeKindAttachment,
		common.CompositeKindContract:
		break
	default:
		return false
	}

	// Composite value's of native/built-in types are not storable for now
	return v.Location != nil
}

func (v *CompositeValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	if !v.IsStorable() {
		return NonStorable{Value: v}, nil
	}

	return v.dictionary.Storable(storage, address, maxInlineSize)
}

func (v *CompositeValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *CompositeValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.Kind == common.CompositeKindAttachment {
		return interpreter.MustSemaTypeOfValue(v).IsResourceType()
	}
	return v.Kind == common.CompositeKindResource
}

func (v *CompositeValue) IsReferenceTrackedResourceKindedValue() {}

func (v *CompositeValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.StorageID]struct{},
) Value {

	config := interpreter.SharedState.Config

	// Should be checked before accessing `v.dictionary`.
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	interpreter.ReportComputation(common.ComputationKindTransferCompositeValue, 1)

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueTransferTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	currentStorageID := v.StorageID()
	currentAddress := v.StorageAddress()

	if preventTransfer == nil {
		preventTransfer = map[atree.StorageID]struct{}{}
	} else if _, ok := preventTransfer[currentStorageID]; ok {
		panic(RecursiveTransferError{
			LocationRange: locationRange,
		})
	}
	preventTransfer[currentStorageID] = struct{}{}
	defer delete(preventTransfer, currentStorageID)

	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo && v.Kind == common.CompositeKindContract {
		panic(NonTransferableValueError{
			Value: v,
		})
	}

	if needsStoreTo || !isResourceKinded {
		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementCount := v.dictionary.Count()

		elementOverhead, dataUse, metaDataUse := common.NewAtreeMapMemoryUsages(elementCount, 0)
		common.UseMemory(interpreter, elementOverhead)
		common.UseMemory(interpreter, dataUse)
		common.UseMemory(interpreter, metaDataUse)

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(elementCount, 0)
		common.UseMemory(config.MemoryGauge, elementMemoryUse)

		dictionary, err = atree.NewMapFromBatchData(
			config.Storage,
			address,
			atree.NewDefaultDigesterBuilder(),
			v.dictionary.Type(),
			StringAtreeValueComparator,
			StringAtreeValueHashInput,
			v.dictionary.Seed(),
			func() (atree.Value, atree.Value, error) {

				atreeKey, atreeValue, err := iterator.Next()
				if err != nil {
					return nil, nil, err
				}
				if atreeKey == nil || atreeValue == nil {
					return nil, nil, nil
				}

				// NOTE: key is stringAtreeValue
				// and does not need to be converted or copied

				value := MustConvertStoredValue(interpreter, atreeValue)
				// the base of an attachment is not stored in the atree, so in order to make the
				// transfer happen properly, we set the base value here if this field is an attachment
				if compositeValue, ok := value.(*CompositeValue); ok &&
					compositeValue.Kind == common.CompositeKindAttachment {

					compositeValue.setBaseValue(interpreter, v)
				}

				value = value.Transfer(
					interpreter,
					locationRange,
					address,
					remove,
					nil,
					preventTransfer,
				)

				return atreeKey, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
				interpreter.RemoveReferencedSlab(nameStorable)
				interpreter.RemoveReferencedSlab(valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}
			interpreter.maybeValidateAtreeValue(v.dictionary)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource as invalidated, by unsetting the backing dictionary.
		// This allows raising an error when the resource is attempted
		// to be transferred/moved again (see beginning of this function)

		interpreter.invalidateReferencedResources(v, false)

		v.dictionary = nil
	}

	info := NewCompositeTypeInfo(
		interpreter,
		v.Location,
		v.QualifiedIdentifier,
		v.Kind,
	)

	res := newCompositeValueFromAtreeMap(
		interpreter,
		info,
		dictionary,
	)

	res.injectedFields = v.injectedFields
	res.computedFields = v.computedFields
	res.NestedVariables = v.NestedVariables
	res.Functions = v.Functions
	res.Stringer = v.Stringer
	res.isDestroyed = v.isDestroyed
	res.typeID = v.typeID
	res.staticType = v.staticType
	res.base = v.base

	onResourceOwnerChange := config.OnResourceOwnerChange

	if needsStoreTo &&
		res.Kind == common.CompositeKindResource &&
		onResourceOwnerChange != nil {

		onResourceOwnerChange(
			interpreter,
			res,
			common.Address(currentAddress),
			common.Address(address),
		)
	}

	return res
}

func (v *CompositeValue) ResourceUUID(interpreter *Interpreter, locationRange LocationRange) *UInt64Value {
	fieldValue := v.GetField(interpreter, locationRange, sema.ResourceUUIDFieldName)
	uuid, ok := fieldValue.(UInt64Value)
	if !ok {
		return nil
	}
	return &uuid
}

func (v *CompositeValue) Clone(interpreter *Interpreter) Value {

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	config := interpreter.SharedState.Config

	dictionary, err := atree.NewMapFromBatchData(
		config.Storage,
		v.StorageAddress(),
		atree.NewDefaultDigesterBuilder(),
		v.dictionary.Type(),
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		v.dictionary.Seed(),
		func() (atree.Value, atree.Value, error) {

			atreeKey, atreeValue, err := iterator.Next()
			if err != nil {
				return nil, nil, err
			}
			if atreeKey == nil || atreeValue == nil {
				return nil, nil, nil
			}

			// The key is always interpreter.StringAtreeValue,
			// an "atree-level string", not an interpreter.Value.
			// Thus, we do not, and cannot, convert.
			key := atreeKey
			value := MustConvertStoredValue(interpreter, atreeValue).Clone(interpreter)

			return key, value, nil
		},
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return &CompositeValue{
		dictionary:          dictionary,
		Location:            v.Location,
		QualifiedIdentifier: v.QualifiedIdentifier,
		Kind:                v.Kind,
		injectedFields:      v.injectedFields,
		computedFields:      v.computedFields,
		NestedVariables:     v.NestedVariables,
		Functions:           v.Functions,
		Stringer:            v.Stringer,
		isDestroyed:         v.isDestroyed,
		typeID:              v.typeID,
		staticType:          v.staticType,
		base:                v.base,
	}
}

func (v *CompositeValue) DeepRemove(interpreter *Interpreter) {
	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueDeepRemoveTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.dictionary.Storage

	err := v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
		// NOTE: key / field name is stringAtreeValue,
		// and not a Value, so no need to deep remove
		interpreter.RemoveReferencedSlab(nameStorable)

		value := StoredValue(interpreter, valueStorable, storage)
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)
}

func (v *CompositeValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

// ForEachField iterates over all field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *CompositeValue) ForEachField(
	gauge common.MemoryGauge,
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	v.forEachField(gauge, v.dictionary.Iterate, f)
}

// ForEachLoadedField iterates over all LOADED field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
func (v *CompositeValue) ForEachLoadedField(
	gauge common.MemoryGauge,
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	v.forEachField(gauge, v.dictionary.IterateLoadedValues, f)
}

func (v *CompositeValue) forEachField(
	gauge common.MemoryGauge,
	atreeIterate func(fn atree.MapEntryIterationFunc) error,
	f func(fieldName string, fieldValue Value) (resume bool),
) {
	err := atreeIterate(func(key atree.Value, value atree.Value) (resume bool, err error) {
		resume = f(
			string(key.(StringAtreeValue)),
			MustConvertStoredValue(gauge, value),
		)
		return
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}

func (v *CompositeValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *CompositeValue) StorageAddress() atree.Address {
	return v.dictionary.Address()
}

func (v *CompositeValue) RemoveField(
	interpreter *Interpreter,
	_ LocationRange,
	name string,
) {

	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		StringAtreeValueComparator,
		StringAtreeValueHashInput,
		StringAtreeValue(name),
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return
		}
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	// Key

	// NOTE: key / field name is stringAtreeValue,
	// and not a Value, so no need to deep remove
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value
	existingValue := StoredValue(interpreter, existingValueStorable, interpreter.Storage())
	existingValue.DeepRemove(interpreter)
	interpreter.RemoveReferencedSlab(existingValueStorable)
}

func (v *CompositeValue) SetNestedVariables(variables map[string]*Variable) {
	v.NestedVariables = variables
}

func NewEnumCaseValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	enumType *sema.CompositeType,
	rawValue NumberValue,
	functions *FunctionOrderedMap,
) *CompositeValue {

	fields := []CompositeField{
		{
			Name:  sema.EnumRawValueFieldName,
			Value: rawValue,
		},
	}

	v := NewCompositeValue(
		interpreter,
		locationRange,
		enumType.Location,
		enumType.QualifiedIdentifier(),
		enumType.Kind,
		fields,
		common.ZeroAddress,
	)

	v.Functions = functions

	return v
}

func (v *CompositeValue) getBaseValue(
	interpreter *Interpreter,
	functionAuthorization Authorization,
	locationRange LocationRange,
) *EphemeralReferenceValue {
	attachmentType, ok := interpreter.MustSemaTypeOfValue(v).(*sema.CompositeType)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	var baseType sema.Type
	switch ty := attachmentType.GetBaseType().(type) {
	case *sema.InterfaceType:
		baseType, _ = ty.RewriteWithIntersectionTypes()
	default:
		baseType = ty
	}

	return NewEphemeralReferenceValue(interpreter, functionAuthorization, v.base, baseType, locationRange)
}

func (v *CompositeValue) setBaseValue(interpreter *Interpreter, base *CompositeValue) {
	v.base = base
}

func attachmentMemberName(ty sema.Type) string {
	return unrepresentableNamePrefix + string(ty.ID())
}

func (v *CompositeValue) getAttachmentValue(interpreter *Interpreter, locationRange LocationRange, ty sema.Type) *CompositeValue {
	if attachment := v.GetMember(interpreter, locationRange, attachmentMemberName(ty)); attachment != nil {
		return attachment.(*CompositeValue)
	}
	return nil
}

func (v *CompositeValue) GetAttachments(interpreter *Interpreter, locationRange LocationRange) []*CompositeValue {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	var attachments []*CompositeValue
	v.forEachAttachment(interpreter, locationRange, func(attachment *CompositeValue) {
		attachments = append(attachments, attachment)
	})
	return attachments
}

func (v *CompositeValue) forEachAttachmentFunction(interpreter *Interpreter, locationRange LocationRange) Value {
	return NewHostFunctionValue(
		interpreter,
		sema.CompositeForEachAttachmentFunctionType(interpreter.MustSemaTypeOfValue(v).(*sema.CompositeType).GetCompositeKind()),
		func(invocation Invocation) Value {
			interpreter := invocation.Interpreter

			functionValue, ok := invocation.Arguments[0].(FunctionValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			fn := func(attachment *CompositeValue) {

				attachmentType := interpreter.MustSemaTypeOfValue(attachment).(*sema.CompositeType)

				// attachments are unauthorized during iteration
				attachmentReferenceAuth := UnauthorizedAccess

				attachmentReference := NewEphemeralReferenceValue(
					interpreter,
					attachmentReferenceAuth,
					attachment,
					attachmentType,
					locationRange,
				)

				invocation := NewInvocation(
					interpreter,
					nil,
					nil,
					nil,
					[]Value{attachmentReference},
					[]sema.Type{sema.NewReferenceType(interpreter, sema.UnauthorizedAccess, attachmentType)},
					nil,
					locationRange,
				)
				functionValue.invoke(invocation)
			}

			v.forEachAttachment(interpreter, locationRange, fn)
			return Void
		},
	)
}

func attachmentBaseAndSelfValues(
	interpreter *Interpreter,
	fnAccess sema.Access,
	v *CompositeValue,
	locationRange LocationRange,
) (base *EphemeralReferenceValue, self *EphemeralReferenceValue) {
	attachmentReferenceAuth := ConvertSemaAccessToStaticAuthorization(interpreter, fnAccess)

	base = v.getBaseValue(interpreter, attachmentReferenceAuth, locationRange)
	// in attachment functions, self is a reference value
	self = NewEphemeralReferenceValue(interpreter, attachmentReferenceAuth, v, interpreter.MustSemaTypeOfValue(v), locationRange)

	return
}

func (v *CompositeValue) forEachAttachment(interpreter *Interpreter, locationRange LocationRange, f func(*CompositeValue)) {
	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	oldSharedState := interpreter.SharedState.inAttachmentIteration(v)
	interpreter.SharedState.setAttachmentIteration(v, true)
	defer func() {
		interpreter.SharedState.setAttachmentIteration(v, oldSharedState)
	}()

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			break
		}
		if strings.HasPrefix(string(key.(StringAtreeValue)), unrepresentableNamePrefix) {
			attachment, ok := MustConvertStoredValue(interpreter, value).(*CompositeValue)
			if !ok {
				panic(errors.NewExternalError(err))
			}
			// `f` takes the `attachment` value directly, but if a method to later iterate over
			// attachments is added that takes a `fun (&Attachment): Void` callback, the `f` provided here
			// should convert the provided attachment value into a reference before passing it to the user
			// callback
			attachment.setBaseValue(interpreter, v)
			f(attachment)
		}
	}
}

func (v *CompositeValue) getTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyType sema.Type,
	baseAccess sema.Access,
) Value {
	attachment := v.getAttachmentValue(interpreter, locationRange, keyType)
	if attachment == nil {
		return Nil
	}
	attachmentType := keyType.(*sema.CompositeType)
	// dynamically set the attachment's base to this composite
	attachment.setBaseValue(interpreter, v)

	// The attachment reference has the same entitlements as the base access
	attachmentRef := NewEphemeralReferenceValue(
		interpreter,
		ConvertSemaAccessToStaticAuthorization(interpreter, baseAccess),
		attachment,
		attachmentType,
		locationRange,
	)

	return NewSomeValueNonCopying(interpreter, attachmentRef)
}

func (v *CompositeValue) GetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	ty sema.Type,
) Value {
	var access sema.Access = sema.UnauthorizedAccess
	attachmentTyp, isAttachmentType := ty.(*sema.CompositeType)
	if isAttachmentType {
		access = sema.NewAccessFromEntitlementSet(attachmentTyp.SupportedEntitlements(), sema.Conjunction)
	}
	return v.getTypeKey(interpreter, locationRange, ty, access)
}

func (v *CompositeValue) SetTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	attachmentType sema.Type,
	attachment Value,
) {
	if v.SetMember(interpreter, locationRange, attachmentMemberName(attachmentType), attachment) {
		panic(DuplicateAttachmentError{
			AttachmentType: attachmentType,
			Value:          v,
			LocationRange:  locationRange,
		})
	}
}

func (v *CompositeValue) RemoveTypeKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	attachmentType sema.Type,
) Value {
	return v.RemoveMember(interpreter, locationRange, attachmentMemberName(attachmentType))
}

// DictionaryValue

type DictionaryValue struct {
	Type             *DictionaryStaticType
	semaType         *sema.DictionaryType
	isResourceKinded *bool
	dictionary       *atree.OrderedMap
	isDestroyed      bool
	elementSize      uint
}

func NewDictionaryValue(
	interpreter *Interpreter,
	locationRange LocationRange,
	dictionaryType *DictionaryStaticType,
	keysAndValues ...Value,
) *DictionaryValue {
	return NewDictionaryValueWithAddress(
		interpreter,
		locationRange,
		dictionaryType,
		common.ZeroAddress,
		keysAndValues...,
	)
}

func NewDictionaryValueWithAddress(
	interpreter *Interpreter,
	locationRange LocationRange,
	dictionaryType *DictionaryStaticType,
	address common.Address,
	keysAndValues ...Value,
) *DictionaryValue {

	interpreter.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			interpreter.reportDictionaryValueConstructTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	keysAndValuesCount := len(keysAndValues)
	if keysAndValuesCount%2 != 0 {
		panic("uneven number of keys and values")
	}

	constructor := func() *atree.OrderedMap {
		dictionary, err := atree.NewMap(
			config.Storage,
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			dictionaryType,
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return dictionary
	}

	// values are added to the dictionary after creation, not here
	v = newDictionaryValueFromConstructor(interpreter, dictionaryType, 0, constructor)

	// NOTE: lazily initialized when needed for performance reasons
	var lazyIsResourceTyped *bool

	for i := 0; i < keysAndValuesCount; i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		existingValue := v.Insert(interpreter, locationRange, key, value)
		// If the dictionary already contained a value for the key,
		// and the dictionary is resource-typed,
		// then we need to prevent a resource loss
		if _, ok := existingValue.(*SomeValue); ok {
			// Lazily determine if the dictionary is resource-typed, once
			if lazyIsResourceTyped == nil {
				isResourceTyped := v.SemaType(interpreter).IsResourceType()
				lazyIsResourceTyped = &isResourceTyped
			}
			if *lazyIsResourceTyped {
				panic(DuplicateKeyInResourceDictionaryError{
					LocationRange: locationRange,
				})
			}
		}
	}

	return v
}

func DictionaryElementSize(staticType *DictionaryStaticType) uint {
	keySize := staticType.KeyType.elementSize()
	valueSize := staticType.ValueType.elementSize()
	if keySize == 0 || valueSize == 0 {
		return 0
	}
	return keySize + valueSize
}

func newDictionaryValueWithIterator(
	interpreter *Interpreter,
	locationRange LocationRange,
	staticType *DictionaryStaticType,
	count uint64,
	seed uint64,
	address common.Address,
	values func() (Value, Value),
) *DictionaryValue {
	interpreter.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		defer func() {
			// NOTE: in defer, as v is only initialized at the end of the function
			// if there was no error during construction
			if v == nil {
				return
			}

			typeInfo := v.Type.String()
			count := v.Count()

			interpreter.reportDictionaryValueConstructTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	constructor := func() *atree.OrderedMap {
		orderedMap, err := atree.NewMapFromBatchData(
			config.Storage,
			atree.Address(address),
			atree.NewDefaultDigesterBuilder(),
			staticType,
			newValueComparator(interpreter, locationRange),
			newHashInputProvider(interpreter, locationRange),
			seed,
			func() (atree.Value, atree.Value, error) {
				key, value := values()
				return key, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		return orderedMap
	}

	// values are added to the dictionary after creation, not here
	v = newDictionaryValueFromConstructor(interpreter, staticType, count, constructor)

	return v
}

func newDictionaryValueFromConstructor(
	gauge common.MemoryGauge,
	staticType *DictionaryStaticType,
	count uint64,
	constructor func() *atree.OrderedMap,
) *DictionaryValue {

	elementSize := DictionaryElementSize(staticType)

	overheadUsage, dataSlabs, metaDataSlabs :=
		common.NewAtreeMapMemoryUsages(count, elementSize)
	common.UseMemory(gauge, overheadUsage)
	common.UseMemory(gauge, dataSlabs)
	common.UseMemory(gauge, metaDataSlabs)

	return newDictionaryValueFromAtreeMap(
		gauge,
		staticType,
		elementSize,
		constructor(),
	)
}

func newDictionaryValueFromAtreeMap(
	gauge common.MemoryGauge,
	staticType *DictionaryStaticType,
	elementSize uint,
	atreeOrderedMap *atree.OrderedMap,
) *DictionaryValue {

	common.UseMemory(gauge, common.DictionaryValueBaseMemoryUsage)

	return &DictionaryValue{
		Type:        staticType,
		dictionary:  atreeOrderedMap,
		elementSize: elementSize,
	}
}

var _ Value = &DictionaryValue{}
var _ atree.Value = &DictionaryValue{}
var _ EquatableValue = &DictionaryValue{}
var _ ValueIndexableValue = &DictionaryValue{}
var _ MemberAccessibleValue = &DictionaryValue{}
var _ ReferenceTrackedResourceKindedValue = &DictionaryValue{}

func (*DictionaryValue) isValue() {}

func (v *DictionaryValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitDictionaryValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(interpreter, func(value Value) {
		value.Accept(interpreter, visitor)
	})
}

func (v *DictionaryValue) Iterate(interpreter *Interpreter, f func(key, value Value) (resume bool)) {
	v.iterate(interpreter, v.dictionary.Iterate, f)
}

func (v *DictionaryValue) IterateLoaded(interpreter *Interpreter, f func(key, value Value) (resume bool)) {
	v.iterate(interpreter, v.dictionary.IterateLoadedValues, f)
}

func (v *DictionaryValue) iterate(
	interpreter *Interpreter,
	atreeIterate func(fn atree.MapEntryIterationFunc) error,
	f func(key Value, value Value) (resume bool),
) {
	iterate := func() {
		err := atreeIterate(func(key, value atree.Value) (resume bool, err error) {
			// atree.OrderedMap iteration provides low-level atree.Value,
			// convert to high-level interpreter.Value

			resume = f(
				MustConvertStoredValue(interpreter, key),
				MustConvertStoredValue(interpreter, value),
			)

			return resume, nil
		})
		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}
	if v.IsResourceKinded(interpreter) {
		interpreter.withMutationPrevention(v.StorageID(), iterate)
	} else {
		iterate()
	}
}

type DictionaryIterator struct {
	mapIterator *atree.MapIterator
}

func (i DictionaryIterator) NextKey(gauge common.MemoryGauge) Value {
	atreeValue, err := i.mapIterator.NextKey()
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	if atreeValue == nil {
		return nil
	}
	return MustConvertStoredValue(gauge, atreeValue)
}

func (v *DictionaryValue) Iterator() DictionaryIterator {
	mapIterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	return DictionaryIterator{
		mapIterator: mapIterator,
	}
}

func (v *DictionaryValue) Walk(interpreter *Interpreter, walkChild func(Value)) {
	v.Iterate(interpreter, func(key, value Value) (resume bool) {
		walkChild(key)
		walkChild(value)
		return true
	})
}

func (v *DictionaryValue) StaticType(_ *Interpreter) StaticType {
	// TODO meter
	return v.Type
}

func (v *DictionaryValue) IsImportable(inter *Interpreter) bool {
	importable := true
	v.Iterate(inter, func(key, value Value) (resume bool) {
		if !key.IsImportable(inter) || !value.IsImportable(inter) {
			importable = false
			// stop iteration
			return false
		}

		// continue iteration
		return true
	})

	return importable
}

func (v *DictionaryValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *DictionaryValue) checkInvalidatedResourceUse(interpreter *Interpreter, locationRange LocationRange) {
	if v.isDestroyed || v.IsStaleResource(interpreter) {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (v *DictionaryValue) IsStaleResource(interpreter *Interpreter) bool {
	return v.dictionary == nil && v.IsResourceKinded(interpreter)
}

func (v *DictionaryValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyDictionaryValue, 1)

	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueDestroyTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	storageID := v.StorageID()

	interpreter.withResourceDestruction(
		storageID,
		locationRange,
		func() {
			v.Iterate(interpreter, func(key, value Value) (resume bool) {
				// Resources cannot be keys at the moment, so should theoretically not be needed
				maybeDestroy(interpreter, locationRange, key)
				maybeDestroy(interpreter, locationRange, value)

				return true
			})
		},
	)

	v.isDestroyed = true

	interpreter.invalidateReferencedResources(v, true)

	v.dictionary = nil
}

func (v *DictionaryValue) ForEachKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	procedure FunctionValue,
) {
	keyType := v.SemaType(interpreter).KeyType

	iterationInvocation := func(key Value) Invocation {
		return NewInvocation(
			interpreter,
			nil,
			nil,
			nil,
			[]Value{key},
			[]sema.Type{keyType},
			nil,
			locationRange,
		)
	}

	iterate := func() {
		err := v.dictionary.IterateKeys(
			func(item atree.Value) (bool, error) {
				key := MustConvertStoredValue(interpreter, item)

				shouldContinue, ok := procedure.invoke(iterationInvocation(key)).(BoolValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return bool(shouldContinue), nil
			},
		)

		if err != nil {
			panic(errors.NewExternalError(err))
		}
	}

	if v.IsResourceKinded(interpreter) {
		interpreter.withMutationPrevention(v.StorageID(), iterate)
	} else {
		iterate()
	}
}

func (v *DictionaryValue) ContainsKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) BoolValue {

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	exists, err := v.dictionary.Has(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	return AsBoolValue(exists)
}

func (v *DictionaryValue) Get(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) (Value, bool) {

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	storable, err := v.dictionary.Get(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return nil, false
		}
		panic(errors.NewExternalError(err))
	}

	storage := v.dictionary.Storage
	value := StoredValue(interpreter, storable, storage)
	return value, true
}

func (v *DictionaryValue) GetKey(interpreter *Interpreter, locationRange LocationRange, keyValue Value) Value {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	value, ok := v.Get(interpreter, locationRange, keyValue)
	if ok {
		return NewSomeValueNonCopying(interpreter, value)
	}

	return Nil
}

func (v *DictionaryValue) SetKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
	value Value,
) {
	interpreter.validateMutation(v.StorageID(), locationRange)

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, locationRange)
	interpreter.checkContainerMutation(
		&OptionalStaticType{ // intentionally unmetered
			Type: v.Type.ValueType,
		},
		value,
		locationRange,
	)

	switch value := value.(type) {
	case *SomeValue:
		innerValue := value.InnerValue(interpreter, locationRange)
		_ = v.Insert(interpreter, locationRange, keyValue, innerValue)

	case NilValue:
		_ = v.Remove(interpreter, locationRange, keyValue)

	case placeholderValue:
		// NO-OP

	default:
		panic(errors.NewUnreachableError())
	}
}

func (v *DictionaryValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *DictionaryValue) RecursiveString(seenReferences SeenReferences) string {
	return v.MeteredString(nil, seenReferences)
}

func (v *DictionaryValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {

	pairs := make([]struct {
		Key   string
		Value string
	}, v.Count())

	index := 0
	_ = v.dictionary.Iterate(func(key, value atree.Value) (resume bool, err error) {
		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value

		pairs[index] = struct {
			Key   string
			Value string
		}{
			Key:   MustConvertStoredValue(memoryGauge, key).MeteredString(memoryGauge, seenReferences),
			Value: MustConvertStoredValue(memoryGauge, value).MeteredString(memoryGauge, seenReferences),
		}
		index++
		return true, nil
	})

	// len = len(open-brace) + len(close-brace) + (n times colon+space) + ((n-1) times comma+space)
	//     = 2 + 2n + 2n - 2
	//     = 4n + 2 - 2
	//
	// Since (-2) only occurs if its non-empty (i.e: n>0), ignore the (-2). i.e: overestimate
	//    len = 4n + 2
	//
	// String of each key and value are metered separately.
	strLen := len(pairs)*4 + 2

	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))

	return format.Dictionary(pairs)
}

func (v *DictionaryValue) GetMember(
	interpreter *Interpreter,
	locationRange LocationRange,
	name string,
) Value {
	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueGetMemberTrace(
				typeInfo,
				count,
				name,
				time.Since(startTime),
			)
		}()
	}

	switch name {
	case "length":
		return NewIntValueFromInt64(interpreter, int64(v.Count()))

	case "keys":

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		return NewArrayValueWithIterator(
			interpreter,
			NewVariableSizedStaticType(interpreter, v.Type.KeyType),
			common.ZeroAddress,
			v.dictionary.Count(),
			func() Value {

				key, err := iterator.NextKey()
				if err != nil {
					panic(errors.NewExternalError(err))
				}
				if key == nil {
					return nil
				}

				return MustConvertStoredValue(interpreter, key).
					Transfer(
						interpreter,
						locationRange,
						atree.Address{},
						false,
						nil,
						nil,
					)
			},
		)

	case "values":

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		return NewArrayValueWithIterator(
			interpreter,
			NewVariableSizedStaticType(interpreter, v.Type.ValueType),
			common.ZeroAddress,
			v.dictionary.Count(),
			func() Value {

				value, err := iterator.NextValue()
				if err != nil {
					panic(errors.NewExternalError(err))
				}
				if value == nil {
					return nil
				}

				return MustConvertStoredValue(interpreter, value).
					Transfer(
						interpreter,
						locationRange,
						atree.Address{},
						false,
						nil,
						nil,
					)
			})

	case "remove":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryRemoveFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				keyValue := invocation.Arguments[0]

				return v.Remove(
					invocation.Interpreter,
					invocation.LocationRange,
					keyValue,
				)
			},
		)

	case "insert":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryInsertFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				keyValue := invocation.Arguments[0]
				newValue := invocation.Arguments[1]

				return v.Insert(
					invocation.Interpreter,
					invocation.LocationRange,
					keyValue,
					newValue,
				)
			},
		)

	case "containsKey":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryContainsKeyFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				return v.ContainsKey(
					invocation.Interpreter,
					invocation.LocationRange,
					invocation.Arguments[0],
				)
			},
		)
	case "forEachKey":
		return NewHostFunctionValue(
			interpreter,
			sema.DictionaryForEachKeyFunctionType(
				v.SemaType(interpreter),
			),
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				funcArgument, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				v.ForEachKey(
					interpreter,
					invocation.LocationRange,
					funcArgument,
				)

				return Void
			},
		)
	}

	return nil
}

func (v *DictionaryValue) RemoveMember(interpreter *Interpreter, locationRange LocationRange, _ string) Value {
	// Dictionaries have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) SetMember(interpreter *Interpreter, locationRange LocationRange, _ string, _ Value) bool {
	// Dictionaries have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) Count() int {
	return int(v.dictionary.Count())
}

func (v *DictionaryValue) RemoveKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key Value,
) Value {
	v.checkInvalidatedResourceUse(interpreter, locationRange)

	return v.Remove(interpreter, locationRange, key)
}

func (v *DictionaryValue) Remove(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue Value,
) OptionalValue {

	interpreter.validateMutation(v.StorageID(), locationRange)

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	// No need to clean up storable for passed-in key value,
	// as atree never calls Storable()
	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		var keyNotFoundError *atree.KeyNotFoundError
		if goerrors.As(err, &keyNotFoundError) {
			return NilOptionalValue
		}
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	storage := interpreter.Storage()

	// Key

	existingKeyValue := StoredValue(interpreter, existingKeyStorable, storage)
	existingKeyValue.DeepRemove(interpreter)
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	existingValue := StoredValue(interpreter, existingValueStorable, storage).
		Transfer(
			interpreter,
			locationRange,
			atree.Address{},
			true,
			existingValueStorable,
			nil,
		)

	return NewSomeValueNonCopying(interpreter, existingValue)
}

func (v *DictionaryValue) InsertKey(
	interpreter *Interpreter,
	locationRange LocationRange,
	key, value Value,
) {
	v.SetKey(interpreter, locationRange, key, value)
}

func (v *DictionaryValue) Insert(
	interpreter *Interpreter,
	locationRange LocationRange,
	keyValue, value Value,
) OptionalValue {

	interpreter.validateMutation(v.StorageID(), locationRange)

	// length increases by 1
	dataSlabs, metaDataSlabs := common.AdditionalAtreeMemoryUsage(v.dictionary.Count(), v.elementSize, false)
	common.UseMemory(interpreter, common.AtreeMapElementOverhead)
	common.UseMemory(interpreter, dataSlabs)
	common.UseMemory(interpreter, metaDataSlabs)

	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, locationRange)
	interpreter.checkContainerMutation(v.Type.ValueType, value, locationRange)

	address := v.dictionary.Address()

	preventTransfer := map[atree.StorageID]struct{}{
		v.StorageID(): {},
	}

	keyValue = keyValue.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
	)

	value = value.Transfer(
		interpreter,
		locationRange,
		address,
		true,
		nil,
		preventTransfer,
	)

	valueComparator := newValueComparator(interpreter, locationRange)
	hashInputProvider := newHashInputProvider(interpreter, locationRange)

	// atree only calls Storable() on keyValue if needed,
	// i.e., if the key is a new key
	existingValueStorable, err := v.dictionary.Set(
		valueComparator,
		hashInputProvider,
		keyValue,
		value,
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	if existingValueStorable == nil {
		return NilOptionalValue
	}

	storage := interpreter.Storage()

	existingValue := StoredValue(
		interpreter,
		existingValueStorable,
		storage,
	).Transfer(
		interpreter,
		locationRange,
		atree.Address{},
		true,
		existingValueStorable,
		nil,
	)

	return NewSomeValueNonCopying(interpreter, existingValue)
}

type DictionaryEntryValues struct {
	Key   Value
	Value Value
}

func (v *DictionaryValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	count := v.Count()

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			interpreter.reportDictionaryValueConformsToStaticTypeTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	staticType, ok := v.StaticType(interpreter).(*DictionaryStaticType)
	if !ok {
		return false
	}

	keyType := staticType.KeyType
	valueType := staticType.ValueType

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			return true
		}

		// Check the key

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryKey := MustConvertStoredValue(interpreter, key)

		if !interpreter.IsSubType(entryKey.StaticType(interpreter), keyType) {
			return false
		}

		if !entryKey.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			return false
		}

		// Check the value

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryValue := MustConvertStoredValue(interpreter, value)

		if !interpreter.IsSubType(entryValue.StaticType(interpreter), valueType) {
			return false
		}

		if !entryValue.ConformsToStaticType(
			interpreter,
			locationRange,
			results,
		) {
			return false
		}
	}
}

func (v *DictionaryValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {

	otherDictionary, ok := other.(*DictionaryValue)
	if !ok {
		return false
	}

	if v.Count() != otherDictionary.Count() {
		return false
	}

	if !v.Type.Equal(otherDictionary.Type) {
		return false
	}

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(errors.NewExternalError(err))
		}
		if key == nil {
			return true
		}

		// Do NOT use an iterator, as other value may be stored in another account,
		// leading to a different iteration order, as the storage ID is used in the seed
		otherValue, otherValueExists :=
			otherDictionary.Get(
				interpreter,
				locationRange,
				MustConvertStoredValue(interpreter, key),
			)

		if !otherValueExists {
			return false
		}

		equatableValue, ok := MustConvertStoredValue(interpreter, value).(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, locationRange, otherValue) {
			return false
		}
	}
}

func (v *DictionaryValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {
	return v.dictionary.Storable(storage, address, maxInlineSize)
}

func (v *DictionaryValue) IsReferenceTrackedResourceKindedValue() {}

func (v *DictionaryValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.StorageID]struct{},
) Value {

	config := interpreter.SharedState.Config

	v.checkInvalidatedResourceUse(interpreter, locationRange)

	interpreter.ReportComputation(
		common.ComputationKindTransferDictionaryValue,
		uint(v.Count()),
	)

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueTransferTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	currentStorageID := v.StorageID()

	if preventTransfer == nil {
		preventTransfer = map[atree.StorageID]struct{}{}
	} else if _, ok := preventTransfer[currentStorageID]; ok {
		panic(RecursiveTransferError{
			LocationRange: locationRange,
		})
	}
	preventTransfer[currentStorageID] = struct{}{}
	defer delete(preventTransfer, currentStorageID)

	dictionary := v.dictionary

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		valueComparator := newValueComparator(interpreter, locationRange)
		hashInputProvider := newHashInputProvider(interpreter, locationRange)

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		elementCount := v.dictionary.Count()

		elementOverhead, dataUse, metaDataUse := common.NewAtreeMapMemoryUsages(
			elementCount,
			v.elementSize,
		)
		common.UseMemory(interpreter, elementOverhead)
		common.UseMemory(interpreter, dataUse)
		common.UseMemory(interpreter, metaDataUse)

		elementMemoryUse := common.NewAtreeMapPreAllocatedElementsMemoryUsage(
			elementCount,
			v.elementSize,
		)
		common.UseMemory(config.MemoryGauge, elementMemoryUse)

		dictionary, err = atree.NewMapFromBatchData(
			config.Storage,
			address,
			atree.NewDefaultDigesterBuilder(),
			v.dictionary.Type(),
			valueComparator,
			hashInputProvider,
			v.dictionary.Seed(),
			func() (atree.Value, atree.Value, error) {

				atreeKey, atreeValue, err := iterator.Next()
				if err != nil {
					return nil, nil, err
				}
				if atreeKey == nil || atreeValue == nil {
					return nil, nil, nil
				}

				key := MustConvertStoredValue(interpreter, atreeKey).
					Transfer(interpreter, locationRange, address, remove, nil, preventTransfer)

				value := MustConvertStoredValue(interpreter, atreeValue).
					Transfer(interpreter, locationRange, address, remove, nil, preventTransfer)

				return key, value, nil
			},
		)
		if err != nil {
			panic(errors.NewExternalError(err))
		}

		if remove {
			err = v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {
				interpreter.RemoveReferencedSlab(keyStorable)
				interpreter.RemoveReferencedSlab(valueStorable)
			})
			if err != nil {
				panic(errors.NewExternalError(err))
			}
			interpreter.maybeValidateAtreeValue(v.dictionary)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
		// to be transferred/moved again (see beginning of this function)

		interpreter.invalidateReferencedResources(v, false)

		v.dictionary = nil
	}

	res := newDictionaryValueFromAtreeMap(
		interpreter,
		v.Type,
		v.elementSize,
		dictionary,
	)

	res.semaType = v.semaType
	res.isResourceKinded = v.isResourceKinded
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *DictionaryValue) Clone(interpreter *Interpreter) Value {
	config := interpreter.SharedState.Config

	valueComparator := newValueComparator(interpreter, EmptyLocationRange)
	hashInputProvider := newHashInputProvider(interpreter, EmptyLocationRange)

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	orderedMap, err := atree.NewMapFromBatchData(
		config.Storage,
		v.StorageAddress(),
		atree.NewDefaultDigesterBuilder(),
		v.dictionary.Type(),
		valueComparator,
		hashInputProvider,
		v.dictionary.Seed(),
		func() (atree.Value, atree.Value, error) {

			atreeKey, atreeValue, err := iterator.Next()
			if err != nil {
				return nil, nil, err
			}
			if atreeKey == nil || atreeValue == nil {
				return nil, nil, nil
			}

			key := MustConvertStoredValue(interpreter, atreeKey).
				Clone(interpreter)

			value := MustConvertStoredValue(interpreter, atreeValue).
				Clone(interpreter)

			return key, value, nil
		},
	)
	if err != nil {
		panic(errors.NewExternalError(err))
	}

	dictionary := newDictionaryValueFromAtreeMap(
		interpreter,
		v.Type,
		v.elementSize,
		orderedMap,
	)

	dictionary.semaType = v.semaType
	dictionary.isResourceKinded = v.isResourceKinded
	dictionary.isDestroyed = v.isDestroyed

	return dictionary
}

func (v *DictionaryValue) DeepRemove(interpreter *Interpreter) {

	config := interpreter.SharedState.Config

	if config.TracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()
		count := v.Count()

		defer func() {
			interpreter.reportDictionaryValueDeepRemoveTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	// Remove nested values and storables

	storage := v.dictionary.Storage

	err := v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {

		key := StoredValue(interpreter, keyStorable, storage)
		key.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(keyStorable)

		value := StoredValue(interpreter, valueStorable, storage)
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(errors.NewExternalError(err))
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)
}

func (v *DictionaryValue) GetOwner() common.Address {
	return common.Address(v.StorageAddress())
}

func (v *DictionaryValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *DictionaryValue) StorageAddress() atree.Address {
	return v.dictionary.Address()
}

func (v *DictionaryValue) SemaType(interpreter *Interpreter) *sema.DictionaryType {
	if v.semaType == nil {
		// this function will panic already if this conversion fails
		v.semaType, _ = interpreter.MustConvertStaticToSemaType(v.Type).(*sema.DictionaryType)
	}
	return v.semaType
}

func (v *DictionaryValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageAddress()
}

func (v *DictionaryValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(interpreter).IsResourceType()
		v.isResourceKinded = &isResourceKinded
	}
	return *v.isResourceKinded
}

// OptionalValue

type OptionalValue interface {
	Value
	isOptionalValue()
	forEach(f func(Value))
	fmap(inter *Interpreter, f func(Value) Value) OptionalValue
}

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

func (v NilValue) Destroy(_ *Interpreter, _ LocationRange) {}

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
	sema.OptionalTypeMapFunctionType(sema.NeverType),
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

// SomeValue

type SomeValue struct {
	value         Value
	valueStorable atree.Storable
	// TODO: Store isDestroyed in SomeStorable?
	isDestroyed bool
}

func NewSomeValueNonCopying(interpreter *Interpreter, value Value) *SomeValue {
	common.UseMemory(interpreter, common.OptionalValueMemoryUsage)

	return NewUnmeteredSomeValueNonCopying(value)
}

func NewUnmeteredSomeValueNonCopying(value Value) *SomeValue {
	return &SomeValue{
		value: value,
	}
}

var _ Value = &SomeValue{}
var _ EquatableValue = &SomeValue{}
var _ MemberAccessibleValue = &SomeValue{}
var _ OptionalValue = &SomeValue{}

func (*SomeValue) isValue() {}

func (v *SomeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitSomeValue(interpreter, v)
	if !descend {
		return
	}
	v.value.Accept(interpreter, visitor)
}

func (v *SomeValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.value)
}

func (v *SomeValue) StaticType(inter *Interpreter) StaticType {
	if v.isDestroyed {
		return nil
	}

	innerType := v.value.StaticType(inter)
	if innerType == nil {
		return nil
	}
	return NewOptionalStaticType(
		inter,
		innerType,
	)
}

func (v *SomeValue) IsImportable(inter *Interpreter) bool {
	return v.value.IsImportable(inter)
}

func (*SomeValue) isOptionalValue() {}

func (v *SomeValue) forEach(f func(Value)) {
	f(v.value)
}

func (v *SomeValue) fmap(inter *Interpreter, f func(Value) Value) OptionalValue {
	newValue := f(v.value)
	return NewSomeValueNonCopying(inter, newValue)
}

func (v *SomeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *SomeValue) Destroy(interpreter *Interpreter, locationRange LocationRange) {
	v.checkInvalidatedResourceUse(locationRange)

	innerValue := v.InnerValue(interpreter, locationRange)
	maybeDestroy(interpreter, locationRange, innerValue)

	v.isDestroyed = true
	v.value = nil
}

func (v *SomeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *SomeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.value.RecursiveString(seenReferences)
}

func (v *SomeValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	return v.value.MeteredString(memoryGauge, seenReferences)
}

func (v *SomeValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	v.checkInvalidatedResourceUse(locationRange)

	switch name {
	case sema.OptionalTypeMapFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.OptionalTypeMapFunctionType(
				interpreter.MustConvertStaticToSemaType(
					v.value.StaticType(interpreter),
				),
			),
			func(invocation Invocation) Value {

				transformFunction, ok := invocation.Arguments[0].(FunctionValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				transformFunctionType, ok := invocation.ArgumentTypes[0].(*sema.FunctionType)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				valueType := transformFunctionType.Parameters[0].TypeAnnotation.Type

				f := func(v Value) Value {
					transformInvocation := NewInvocation(
						invocation.Interpreter,
						nil,
						nil,
						nil,
						[]Value{v},
						[]sema.Type{valueType},
						nil,
						invocation.LocationRange,
					)
					return transformFunction.invoke(transformInvocation)
				}

				return v.fmap(invocation.Interpreter, f)
			},
		)
	}

	return nil
}

func (v *SomeValue) RemoveMember(interpreter *Interpreter, locationRange LocationRange, _ string) Value {
	panic(errors.NewUnreachableError())
}

func (v *SomeValue) SetMember(interpreter *Interpreter, locationRange LocationRange, _ string, _ Value) bool {
	panic(errors.NewUnreachableError())
}

func (v *SomeValue) ConformsToStaticType(
	interpreter *Interpreter,
	locationRange LocationRange,
	results TypeConformanceResults,
) bool {

	// NOTE: value does not have static type information on its own,
	// SomeValue.StaticType builds type from inner value (if available),
	// so no need to check it

	innerValue := v.InnerValue(interpreter, locationRange)

	return innerValue.ConformsToStaticType(
		interpreter,
		locationRange,
		results,
	)
}

func (v *SomeValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherSome, ok := other.(*SomeValue)
	if !ok {
		return false
	}

	innerValue := v.InnerValue(interpreter, locationRange)

	equatableValue, ok := innerValue.(EquatableValue)
	if !ok {
		return false
	}

	return equatableValue.Equal(interpreter, locationRange, otherSome.value)
}

func (v *SomeValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {

	if v.valueStorable == nil {
		var err error
		v.valueStorable, err = v.value.Storable(
			storage,
			address,
			maxInlineSize,
		)
		if err != nil {
			return nil, err
		}
	}

	return maybeLargeImmutableStorable(
		SomeStorable{
			Storable: v.valueStorable,
		},
		storage,
		address,
		maxInlineSize,
	)
}

func (v *SomeValue) NeedsStoreTo(address atree.Address) bool {
	return v.value.NeedsStoreTo(address)
}

func (v *SomeValue) IsResourceKinded(interpreter *Interpreter) bool {
	// If the inner value is `nil`, then this is an invalidated resource.
	if v.value == nil {
		return true
	}

	return v.value.IsResourceKinded(interpreter)
}

func (v *SomeValue) checkInvalidatedResourceUse(locationRange LocationRange) {
	if v.isDestroyed || v.value == nil {
		panic(InvalidatedResourceError{
			LocationRange: locationRange,
		})
	}
}

func (v *SomeValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.StorageID]struct{},
) Value {

	v.checkInvalidatedResourceUse(locationRange)

	innerValue := v.value

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		innerValue = v.value.Transfer(
			interpreter,
			locationRange,
			address,
			remove,
			nil,
			preventTransfer,
		)

		if remove {
			interpreter.RemoveReferencedSlab(v.valueStorable)
			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		// If checking of transfers of invalidated resource is enabled,
		// then mark the resource array as invalidated, by unsetting the backing array.
		// This allows raising an error when the resource array is attempted
		// to be transferred/moved again (see beginning of this function)
		v.value = nil
	}

	res := NewSomeValueNonCopying(interpreter, innerValue)
	res.valueStorable = nil
	res.isDestroyed = v.isDestroyed

	return res
}

func (v *SomeValue) Clone(interpreter *Interpreter) Value {
	innerValue := v.value.Clone(interpreter)
	return NewUnmeteredSomeValueNonCopying(innerValue)
}

func (v *SomeValue) DeepRemove(interpreter *Interpreter) {
	v.value.DeepRemove(interpreter)
	if v.valueStorable != nil {
		interpreter.RemoveReferencedSlab(v.valueStorable)
	}
}

func (v *SomeValue) InnerValue(_ *Interpreter, locationRange LocationRange) Value {
	v.checkInvalidatedResourceUse(locationRange)
	return v.value
}

type SomeStorable struct {
	gauge    common.MemoryGauge
	Storable atree.Storable
}

var _ atree.Storable = SomeStorable{}

func (s SomeStorable) ByteSize() uint32 {
	return cborTagSize + s.Storable.ByteSize()
}

func (s SomeStorable) StoredValue(storage atree.SlabStorage) (atree.Value, error) {
	value := StoredValue(s.gauge, s.Storable, storage)

	return &SomeValue{
		value:         value,
		valueStorable: s.Storable,
	}, nil
}

func (s SomeStorable) ChildStorables() []atree.Storable {
	return []atree.Storable{
		s.Storable,
	}
}

type AuthorizedValue interface {
	GetAuthorization() Authorization
}

type ReferenceValue interface {
	Value
	isReference()
	ReferencedValue(interpreter *Interpreter, locationRange LocationRange, errorOnFailedDereference bool) *Value
}

// StorageReferenceValue
type StorageReferenceValue struct {
	BorrowedType         sema.Type
	TargetPath           PathValue
	TargetStorageAddress common.Address
	Authorization        Authorization
}

var _ Value = &StorageReferenceValue{}
var _ EquatableValue = &StorageReferenceValue{}
var _ ValueIndexableValue = &StorageReferenceValue{}
var _ TypeIndexableValue = &StorageReferenceValue{}
var _ MemberAccessibleValue = &StorageReferenceValue{}
var _ AuthorizedValue = &StorageReferenceValue{}
var _ ReferenceValue = &StorageReferenceValue{}
var _ IterableValue = &StorageReferenceValue{}

func NewUnmeteredStorageReferenceValue(
	authorization Authorization,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType sema.Type,
) *StorageReferenceValue {
	return &StorageReferenceValue{
		Authorization:        authorization,
		TargetStorageAddress: targetStorageAddress,
		TargetPath:           targetPath,
		BorrowedType:         borrowedType,
	}
}

func NewStorageReferenceValue(
	memoryGauge common.MemoryGauge,
	authorization Authorization,
	targetStorageAddress common.Address,
	targetPath PathValue,
	borrowedType sema.Type,
) *StorageReferenceValue {
	common.UseMemory(memoryGauge, common.StorageReferenceValueMemoryUsage)
	return NewUnmeteredStorageReferenceValue(
		authorization,
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
		v.Authorization,
		self.StaticType(inter),
	)
}

func (v *StorageReferenceValue) GetAuthorization() Authorization {
	return v.Authorization
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

	if selfComposite, isComposite := self.(*CompositeValue); isComposite {
		return selfComposite.getTypeKey(
			interpreter,
			locationRange,
			key,
			interpreter.MustConvertStaticAuthorizationToSemaAccess(v.Authorization),
		)
	}

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
		!v.Authorization.Equal(otherReference.Authorization) {

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
		v.Authorization,
		v.TargetStorageAddress,
		v.TargetPath,
		v.BorrowedType,
	)
}

func (*StorageReferenceValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (*StorageReferenceValue) isReference() {}

func (v *StorageReferenceValue) Iterator(_ *Interpreter) ValueIterator {
	// Not used for now
	panic(errors.NewUnreachableError())
}

func (v *StorageReferenceValue) ForEach(
	interpreter *Interpreter,
	elementType sema.Type,
	function func(value Value) (resume bool),
	locationRange LocationRange,
) {
	referencedValue := v.mustReferencedValue(interpreter, locationRange)
	forEachReference(
		interpreter,
		referencedValue,
		elementType,
		function,
		locationRange,
	)
}

func forEachReference(
	interpreter *Interpreter,
	referencedValue Value,
	elementType sema.Type,
	function func(value Value) (resume bool),
	locationRange LocationRange,
) {
	referencedIterable, ok := referencedValue.(IterableValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	referenceType, isResultReference := sema.MaybeReferenceType(elementType)

	updatedFunction := func(value Value) (resume bool) {
		if isResultReference {
			value = interpreter.getReferenceValue(value, elementType, locationRange)
		}

		return function(value)
	}

	referencedElementType := elementType
	if isResultReference {
		referencedElementType = referenceType.Type
	}

	referencedIterable.ForEach(
		interpreter,
		referencedElementType,
		updatedFunction,
		locationRange,
	)
}

// EphemeralReferenceValue

type EphemeralReferenceValue struct {
	Value Value
	// BorrowedType is the T in &T
	BorrowedType  sema.Type
	Authorization Authorization
}

var _ Value = &EphemeralReferenceValue{}
var _ EquatableValue = &EphemeralReferenceValue{}
var _ ValueIndexableValue = &EphemeralReferenceValue{}
var _ TypeIndexableValue = &EphemeralReferenceValue{}
var _ MemberAccessibleValue = &EphemeralReferenceValue{}
var _ AuthorizedValue = &EphemeralReferenceValue{}
var _ ReferenceValue = &EphemeralReferenceValue{}
var _ IterableValue = &EphemeralReferenceValue{}

func NewUnmeteredEphemeralReferenceValue(
	authorization Authorization,
	value Value,
	borrowedType sema.Type,
	locationRange LocationRange,
) *EphemeralReferenceValue {
	if reference, isReference := value.(*EphemeralReferenceValue); isReference {
		panic(NestedReferenceError{
			Value:         reference,
			LocationRange: locationRange,
		})
	}

	return &EphemeralReferenceValue{
		Authorization: authorization,
		Value:         value,
		BorrowedType:  borrowedType,
	}
}

func NewEphemeralReferenceValue(
	interpreter *Interpreter,
	authorization Authorization,
	value Value,
	borrowedType sema.Type,
	locationRange LocationRange,
) *EphemeralReferenceValue {
	common.UseMemory(interpreter, common.EphemeralReferenceValueMemoryUsage)
	interpreter.maybeTrackReferencedResourceKindedValue(value)
	return NewUnmeteredEphemeralReferenceValue(authorization, value, borrowedType, locationRange)
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
		v.Authorization,
		self.StaticType(inter),
	)
}

func (v *EphemeralReferenceValue) GetAuthorization() Authorization {
	return v.Authorization
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

	interpreter.checkReferencedResourceNotMovedOrDestroyed(self, locationRange)
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

	if selfComposite, isComposite := self.(*CompositeValue); isComposite {
		return selfComposite.getTypeKey(
			interpreter,
			locationRange,
			key,
			interpreter.MustConvertStaticAuthorizationToSemaAccess(v.Authorization),
		)
	}

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
		!v.Authorization.Equal(otherReference.Authorization) {

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

	interpreter.checkReferencedResourceNotMovedOrDestroyed(*referencedValue, locationRange)

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

func (v *EphemeralReferenceValue) Clone(*Interpreter) Value {
	return NewUnmeteredEphemeralReferenceValue(v.Authorization, v.Value, v.BorrowedType, EmptyLocationRange)
}

func (*EphemeralReferenceValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (*EphemeralReferenceValue) isReference() {}

func (v *EphemeralReferenceValue) Iterator(_ *Interpreter) ValueIterator {
	// Not used for now
	panic(errors.NewUnreachableError())
}

func (v *EphemeralReferenceValue) ForEach(
	interpreter *Interpreter,
	elementType sema.Type,
	function func(value Value) (resume bool),
	locationRange LocationRange,
) {
	referencedValue := v.MustReferencedValue(interpreter, locationRange)
	forEachReference(
		interpreter,
		referencedValue,
		elementType,
		function,
		locationRange,
	)
}

// AddressValue
type AddressValue common.Address

func NewAddressValueFromBytes(memoryGauge common.MemoryGauge, constructor func() []byte) AddressValue {
	common.UseMemory(memoryGauge, common.AddressValueMemoryUsage)
	return NewUnmeteredAddressValueFromBytes(constructor())
}

func NewUnmeteredAddressValueFromBytes(b []byte) AddressValue {
	result := AddressValue{}
	copy(result[common.AddressLength-len(b):], b)
	return result
}

// NewAddressValue constructs an address-value from a `common.Address`.
//
// NOTE:
// This method must only be used if the `address` value is already constructed,
// and/or already loaded onto memory. This is a convenient method for better performance.
// If the `address` needs to be constructed, the `NewAddressValueFromConstructor` must be used.
func NewAddressValue(
	memoryGauge common.MemoryGauge,
	address common.Address,
) AddressValue {
	common.UseMemory(memoryGauge, common.AddressValueMemoryUsage)
	return NewUnmeteredAddressValueFromBytes(address[:])
}

func NewAddressValueFromConstructor(
	memoryGauge common.MemoryGauge,
	addressConstructor func() common.Address,
) AddressValue {
	common.UseMemory(memoryGauge, common.AddressValueMemoryUsage)
	address := addressConstructor()
	return NewUnmeteredAddressValueFromBytes(address[:])
}

func ConvertAddress(memoryGauge common.MemoryGauge, value Value, locationRange LocationRange) AddressValue {
	converter := func() (result common.Address) {
		uint64Value := ConvertUInt64(memoryGauge, value, locationRange)

		binary.BigEndian.PutUint64(
			result[:common.AddressLength],
			uint64(uint64Value),
		)

		return
	}

	return NewAddressValueFromConstructor(memoryGauge, converter)
}

var _ Value = AddressValue{}
var _ atree.Storable = AddressValue{}
var _ EquatableValue = AddressValue{}
var _ HashableValue = AddressValue{}
var _ MemberAccessibleValue = AddressValue{}

func (AddressValue) isValue() {}

func (v AddressValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitAddressValue(interpreter, v)
}

func (AddressValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (AddressValue) StaticType(interpreter *Interpreter) StaticType {
	return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeAddress)
}

func (AddressValue) IsImportable(_ *Interpreter) bool {
	return true
}

func (v AddressValue) String() string {
	return format.Address(common.Address(v))
}

func (v AddressValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v AddressValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	common.UseMemory(memoryGauge, common.AddressValueStringMemoryUsage)
	return v.String()
}

func (v AddressValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherAddress, ok := other.(AddressValue)
	if !ok {
		return false
	}
	return v == otherAddress
}

// HashInput returns a byte slice containing:
// - HashInputTypeAddress (1 byte)
// - address (8 bytes)
func (v AddressValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	length := 1 + len(v)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypeAddress)
	copy(buffer[1:], v[:])
	return buffer
}

func (v AddressValue) Hex() string {
	return v.ToAddress().Hex()
}

func (v AddressValue) ToAddress() common.Address {
	return common.Address(v)
}

func (v AddressValue) GetMember(interpreter *Interpreter, locationRange LocationRange, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.ToStringFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter
				memoryUsage := common.NewStringMemoryUsage(
					safeMul(common.AddressLength, 2, locationRange),
				)
				return NewStringValue(
					interpreter,
					memoryUsage,
					func() string {
						return v.String()
					},
				)
			},
		)

	case sema.AddressTypeToBytesFunctionName:
		return NewHostFunctionValue(
			interpreter,
			sema.AddressTypeToBytesFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter
				address := common.Address(v)
				return ByteSliceToByteArrayValue(interpreter, address[:])
			},
		)
	}

	return nil
}

func (AddressValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Addresses have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (AddressValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Addresses have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v AddressValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (AddressValue) IsStorable() bool {
	return true
}

func (v AddressValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return v, nil
}

func (AddressValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (AddressValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v AddressValue) Transfer(
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

func (v AddressValue) Clone(_ *Interpreter) Value {
	return v
}

func (AddressValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v AddressValue) ByteSize() uint32 {
	return cborTagSize + getBytesCBORSize(v.ToAddress().Bytes())
}

func (v AddressValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (AddressValue) ChildStorables() []atree.Storable {
	return nil
}

func AddressFromBytes(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter

	bytes, err := ByteArrayValueToByteSlice(inter, argument, invocation.LocationRange)
	if err != nil {
		panic(err)
	}

	return NewAddressValue(invocation.Interpreter, common.MustBytesToAddress(bytes))
}

func AddressFromString(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	addr, err := common.HexToAddressAssertPrefix(argument.Str)
	if err != nil {
		return Nil
	}

	inter := invocation.Interpreter
	return NewSomeValueNonCopying(inter, NewAddressValue(inter, addr))
}

// PathValue

type PathValue struct {
	Identifier string
	Domain     common.PathDomain
}

func NewUnmeteredPathValue(domain common.PathDomain, identifier string) PathValue {
	return PathValue{Domain: domain, Identifier: identifier}
}

func NewPathValue(
	memoryGauge common.MemoryGauge,
	domain common.PathDomain,
	identifier string,
) PathValue {
	common.UseMemory(memoryGauge, common.PathValueMemoryUsage)
	return NewUnmeteredPathValue(domain, identifier)
}

var EmptyPathValue = PathValue{}

var _ Value = PathValue{}
var _ atree.Storable = PathValue{}
var _ EquatableValue = PathValue{}
var _ HashableValue = PathValue{}
var _ MemberAccessibleValue = PathValue{}

func (PathValue) isValue() {}

func (v PathValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitPathValue(interpreter, v)
}

func (PathValue) Walk(_ *Interpreter, _ func(Value)) {
	// NO-OP
}

func (v PathValue) StaticType(interpreter *Interpreter) StaticType {
	switch v.Domain {
	case common.PathDomainStorage:
		return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypeStoragePath)
	case common.PathDomainPublic:
		return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypePublicPath)
	case common.PathDomainPrivate:
		return NewPrimitiveStaticType(interpreter, PrimitiveStaticTypePrivatePath)
	default:
		panic(errors.NewUnreachableError())
	}
}

func (v PathValue) IsImportable(_ *Interpreter) bool {
	switch v.Domain {
	case common.PathDomainStorage:
		return sema.StoragePathType.Importable
	case common.PathDomainPublic:
		return sema.PublicPathType.Importable
	case common.PathDomainPrivate:
		return sema.PrivatePathType.Importable
	default:
		panic(errors.NewUnreachableError())
	}
}

func (v PathValue) String() string {
	return format.Path(
		v.Domain.Identifier(),
		v.Identifier,
	)
}

func (v PathValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v PathValue) MeteredString(memoryGauge common.MemoryGauge, _ SeenReferences) string {
	// len(domain) + len(identifier) + '/' x2
	strLen := len(v.Domain.Identifier()) + len(v.Identifier) + 2
	common.UseMemory(memoryGauge, common.NewRawStringMemoryUsage(strLen))
	return v.String()
}

func (v PathValue) GetMember(inter *Interpreter, locationRange LocationRange, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			inter,
			sema.ToStringFunctionType,
			func(invocation Invocation) Value {
				interpreter := invocation.Interpreter

				domainLength := len(v.Domain.Identifier())
				identifierLength := len(v.Identifier)

				memoryUsage := common.NewStringMemoryUsage(
					safeAdd(domainLength, identifierLength, locationRange),
				)

				return NewStringValue(
					interpreter,
					memoryUsage,
					v.String,
				)
			},
		)
	}

	return nil
}

func (PathValue) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Paths have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (PathValue) SetMember(_ *Interpreter, _ LocationRange, _ string, _ Value) bool {
	// Paths have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v PathValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return true
}

func (v PathValue) Equal(_ *Interpreter, _ LocationRange, other Value) bool {
	otherPath, ok := other.(PathValue)
	if !ok {
		return false
	}

	return otherPath.Identifier == v.Identifier &&
		otherPath.Domain == v.Domain
}

// HashInput returns a byte slice containing:
// - HashInputTypePath (1 byte)
// - domain (1 byte)
// - identifier (n bytes)
func (v PathValue) HashInput(_ *Interpreter, _ LocationRange, scratch []byte) []byte {
	length := 1 + 1 + len(v.Identifier)
	var buffer []byte
	if length <= len(scratch) {
		buffer = scratch[:length]
	} else {
		buffer = make([]byte, length)
	}

	buffer[0] = byte(HashInputTypePath)
	buffer[1] = byte(v.Domain)
	copy(buffer[2:], v.Identifier)
	return buffer
}

func (PathValue) IsStorable() bool {
	return true
}

func convertPath(interpreter *Interpreter, domain common.PathDomain, value Value) Value {
	stringValue, ok := value.(*StringValue)
	if !ok {
		return Nil
	}

	_, err := sema.CheckPathLiteral(
		domain.Identifier(),
		stringValue.Str,
		ReturnEmptyRange,
		ReturnEmptyRange,
	)
	if err != nil {
		return Nil
	}

	return NewSomeValueNonCopying(
		interpreter,
		NewPathValue(
			interpreter,
			domain,
			stringValue.Str,
		),
	)
}

func ConvertPublicPath(interpreter *Interpreter, value Value) Value {
	return convertPath(interpreter, common.PathDomainPublic, value)
}

func ConvertPrivatePath(interpreter *Interpreter, value Value) Value {
	return convertPath(interpreter, common.PathDomainPrivate, value)
}

func ConvertStoragePath(interpreter *Interpreter, value Value) Value {
	return convertPath(interpreter, common.PathDomainStorage, value)
}

func (v PathValue) Storable(
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

func (PathValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (PathValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v PathValue) Transfer(
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

func (v PathValue) Clone(_ *Interpreter) Value {
	return v
}

func (PathValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v PathValue) ByteSize() uint32 {
	// tag number (2 bytes) + array head (1 byte) + domain (CBOR uint) + identifier (CBOR string)
	return cborTagSize + 1 + getUintCBORSize(uint64(v.Domain)) + getBytesCBORSize([]byte(v.Identifier))
}

func (v PathValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (PathValue) ChildStorables() []atree.Storable {
	return nil
}

// PublishedValue

type PublishedValue struct {
	// NB: If `publish` and `claim` are ever extended to support arbitrary values, rather than just capabilities,
	// this will need to be changed to `Value`, and more storage-related operations must be implemented for `PublishedValue`
	Value     *CapabilityValue
	Recipient AddressValue
}

func NewPublishedValue(memoryGauge common.MemoryGauge, recipient AddressValue, value *CapabilityValue) *PublishedValue {
	common.UseMemory(memoryGauge, common.PublishedValueMemoryUsage)
	return &PublishedValue{
		Recipient: recipient,
		Value:     value,
	}
}

var _ Value = &PublishedValue{}
var _ atree.Value = &PublishedValue{}
var _ EquatableValue = &PublishedValue{}

func (*PublishedValue) isValue() {}

func (v *PublishedValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitPublishedValue(interpreter, v)
}

func (v *PublishedValue) StaticType(interpreter *Interpreter) StaticType {
	// checking the static type of a published value should show us the
	// static type of the underlying value
	return v.Value.StaticType(interpreter)
}

func (*PublishedValue) IsImportable(_ *Interpreter) bool {
	return false
}

func (v *PublishedValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *PublishedValue) RecursiveString(seenReferences SeenReferences) string {
	return fmt.Sprintf(
		"PublishedValue<%s>(%s)",
		v.Recipient.RecursiveString(seenReferences),
		v.Value.RecursiveString(seenReferences),
	)
}

func (v *PublishedValue) MeteredString(memoryGauge common.MemoryGauge, seenReferences SeenReferences) string {
	common.UseMemory(memoryGauge, common.PublishedValueStringMemoryUsage)

	return fmt.Sprintf(
		"PublishedValue<%s>(%s)",
		v.Recipient.MeteredString(memoryGauge, seenReferences),
		v.Value.MeteredString(memoryGauge, seenReferences),
	)
}

func (v *PublishedValue) Walk(_ *Interpreter, walkChild func(Value)) {
	walkChild(v.Recipient)
	walkChild(v.Value)
}

func (v *PublishedValue) ConformsToStaticType(
	_ *Interpreter,
	_ LocationRange,
	_ TypeConformanceResults,
) bool {
	return false
}

func (v *PublishedValue) Equal(interpreter *Interpreter, locationRange LocationRange, other Value) bool {
	otherValue, ok := other.(*PublishedValue)
	if !ok {
		return false
	}

	return otherValue.Recipient.Equal(interpreter, locationRange, v.Recipient) &&
		otherValue.Value.Equal(interpreter, locationRange, v.Value)
}

func (*PublishedValue) IsStorable() bool {
	return true
}

func (v *PublishedValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (v *PublishedValue) NeedsStoreTo(address atree.Address) bool {
	return v.Value.NeedsStoreTo(address)
}

func (*PublishedValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *PublishedValue) Transfer(
	interpreter *Interpreter,
	locationRange LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
	preventTransfer map[atree.StorageID]struct{},
) Value {
	// NB: if the inner value of a PublishedValue can be a resource,
	// we must perform resource-related checks here as well

	if v.NeedsStoreTo(address) {

		innerValue := v.Value.Transfer(
			interpreter,
			locationRange,
			address,
			remove,
			nil,
			preventTransfer,
		).(*CapabilityValue)

		addressValue := v.Recipient.Transfer(
			interpreter,
			locationRange,
			address,
			remove,
			nil,
			preventTransfer,
		).(AddressValue)

		if remove {
			interpreter.RemoveReferencedSlab(storable)
		}

		return NewPublishedValue(interpreter, addressValue, innerValue)
	}

	return v

}

func (v *PublishedValue) Clone(interpreter *Interpreter) Value {
	return &PublishedValue{
		Recipient: v.Recipient,
		Value:     v.Value.Clone(interpreter).(*CapabilityValue),
	}
}

func (*PublishedValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v *PublishedValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *PublishedValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *PublishedValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.Recipient,
		v.Value,
	}
}
