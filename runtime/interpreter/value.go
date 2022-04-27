/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/onflow/atree"
	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/norm"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

type TypeConformanceResults map[typeConformanceResultEntry]bool

type typeConformanceResultEntry struct {
	EphemeralReferenceValue       *EphemeralReferenceValue
	EphemeralReferenceDynamicType EphemeralReferenceDynamicType
}

// SeenReferences is a set of seen references.
//
// NOTE: Do not generalize to map[interpreter.Value],
// as not all values are Go hashable, i.e. this might lead to run-time panics
//
type SeenReferences map[*EphemeralReferenceValue]struct{}

// NonStorable represents a value that cannot be stored
//
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
	// but 1 results in less number of slabs which is ideal for non-storable values.
	return 1
}

func (s NonStorable) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return s.Value, nil
}

func (NonStorable) ChildStorables() []atree.Storable {
	return nil
}

// Value

type Value interface {
	atree.Value
	// Stringer provides `func String() string`
	// NOTE: important, error messages rely on values to implement String
	fmt.Stringer
	IsValue()
	Accept(interpreter *Interpreter, visitor Visitor)
	Walk(walkChild func(Value))
	DynamicType(interpreter *Interpreter, seenReferences SeenReferences) DynamicType
	StaticType() StaticType
	ConformsToDynamicType(
		interpreter *Interpreter,
		getLocationRange func() LocationRange,
		dynamicType DynamicType,
		results TypeConformanceResults,
	) bool
	RecursiveString(seenReferences SeenReferences) string
	IsResourceKinded(interpreter *Interpreter) bool
	NeedsStoreTo(address atree.Address) bool
	Transfer(
		interpreter *Interpreter,
		getLocationRange func() LocationRange,
		address atree.Address,
		remove bool,
		storable atree.Storable,
	) Value
	DeepRemove(interpreter *Interpreter)
}

// ValueIndexableValue

type ValueIndexableValue interface {
	Value
	GetKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value) Value
	SetKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value, value Value)
	RemoveKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value) Value
	InsertKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value, value Value)
}

// MemberAccessibleValue

type MemberAccessibleValue interface {
	Value
	GetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value
	RemoveMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value
	SetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string, value Value)
}

// EquatableValue

type EquatableValue interface {
	Value
	// Equal returns true if the given value is equal to this value.
	// If no location range is available, pass e.g. ReturnEmptyLocationRange
	Equal(interpreter *Interpreter, getLocationRange func() LocationRange, other Value) bool
}

func newValueComparator(interpreter *Interpreter, getLocationRange func() LocationRange) atree.ValueComparator {
	return func(storage atree.SlabStorage, atreeValue atree.Value, otherStorable atree.Storable) (bool, error) {
		value := MustConvertStoredValue(atreeValue)
		otherValue := StoredValue(otherStorable, storage)
		return value.(EquatableValue).Equal(interpreter, getLocationRange, otherValue), nil
	}
}

// ResourceKindedValue

type ResourceKindedValue interface {
	Value
	Destroy(interpreter *Interpreter, getLocationRange func() LocationRange)
	IsDestroyed() bool
}

func maybeDestroy(interpreter *Interpreter, getLocationRange func() LocationRange, value Value) {
	resourceKindedValue, ok := value.(ResourceKindedValue)
	if !ok {
		return
	}

	resourceKindedValue.Destroy(interpreter, getLocationRange)
}

// ReferenceTrackedResourceKindedValue is a resource-kinded value
// that must be tracked when a reference of it is taken.
//
type ReferenceTrackedResourceKindedValue interface {
	ResourceKindedValue
	IsReferenceTrackedResourceKindedValue()
	StorageID() atree.StorageID
}

// TypeValue

type TypeValue struct {
	Type StaticType
}

var _ Value = TypeValue{}
var _ atree.Storable = TypeValue{}
var _ EquatableValue = TypeValue{}
var _ MemberAccessibleValue = TypeValue{}

func (TypeValue) IsValue() {}

func (v TypeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitTypeValue(interpreter, v)
}

func (TypeValue) Walk(_ func(Value)) {
	// NO-OP
}

var metaTypeDynamicType DynamicType = MetaTypeDynamicType{}

func (TypeValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return metaTypeDynamicType
}

func (TypeValue) StaticType() StaticType {
	return PrimitiveStaticTypeMetaType
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

func (v TypeValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
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

func (v TypeValue) GetMember(interpreter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "identifier":
		var typeID string
		staticType := v.Type
		if staticType != nil {
			typeID = string(interpreter.ConvertStaticToSemaType(staticType).ID())
		}
		return NewStringValue(typeID)
	case "isSubtype":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				staticType := v.Type
				otherStaticType := invocation.Arguments[0].(TypeValue).Type

				// if either type is unknown, the subtype relation is false, as it doesn't make sense to even ask this question
				if staticType == nil || otherStaticType == nil {
					return BoolValue(false)
				}

				inter := invocation.Interpreter

				result := sema.IsSubType(
					inter.ConvertStaticToSemaType(staticType),
					inter.ConvertStaticToSemaType(otherStaticType),
				)
				return BoolValue(result)
			},
			sema.MetaTypeIsSubtypeFunctionType,
		)
	}

	return nil
}

func (TypeValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Types have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (TypeValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Types have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v TypeValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	_, ok := dynamicType.(MetaTypeDynamicType)
	return ok
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

// VoidValue

type VoidValue struct{}

var _ Value = VoidValue{}
var _ atree.Storable = VoidValue{}
var _ EquatableValue = VoidValue{}

func (VoidValue) IsValue() {}

func (v VoidValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitVoidValue(interpreter, v)
}

func (VoidValue) Walk(_ func(Value)) {
	// NO-OP
}

var voidDynamicType DynamicType = VoidDynamicType{}

func (VoidValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return voidDynamicType
}

func (VoidValue) StaticType() StaticType {
	return PrimitiveStaticTypeVoid
}

func (VoidValue) String() string {
	return format.Void
}

func (v VoidValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v VoidValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	_, ok := dynamicType.(VoidDynamicType)
	return ok
}

func (v VoidValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (VoidValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v VoidValue) ByteSize() uint32 {
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

func (BoolValue) IsValue() {}

func (v BoolValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitBoolValue(interpreter, v)
}

func (BoolValue) Walk(_ func(Value)) {
	// NO-OP
}

var boolDynamicType DynamicType = BoolDynamicType{}

func (BoolValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return boolDynamicType
}

func (BoolValue) StaticType() StaticType {
	return PrimitiveStaticTypeBool
}

func (v BoolValue) Negate() BoolValue {
	return !v
}

func (v BoolValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherBool, ok := other.(BoolValue)
	if !ok {
		return false
	}
	return bool(v) == bool(otherBool)
}

func (v BoolValue) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
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
func (v BoolValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	_, ok := dynamicType.(BoolDynamicType)
	return ok
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

// StringValue

type StringValue struct {
	Str string
	// length is the cached length of the string, based on grapheme clusters.
	// a negative value indicates the length has not been initialized, see Length()
	length int
	// graphemes is a grapheme cluster segmentation iterator,
	// which is initialized lazily and reused/reset in functions
	// that are based on grapheme clusters
	graphemes *uniseg.Graphemes
}

func NewStringValue(str string) *StringValue {
	return &StringValue{
		Str: str,
		// a negative value indicates the length has not been initialized, see Length()
		length: -1,
	}
}

var _ Value = &StringValue{}
var _ atree.Storable = &StringValue{}
var _ EquatableValue = &StringValue{}
var _ HashableValue = &StringValue{}
var _ ValueIndexableValue = &StringValue{}
var _ MemberAccessibleValue = &StringValue{}

func (v *StringValue) prepareGraphemes() {
	if v.graphemes == nil {
		v.graphemes = uniseg.NewGraphemes(v.Str)
	} else {
		v.graphemes.Reset()
	}
}

func (*StringValue) IsValue() {}

func (v *StringValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitStringValue(interpreter, v)
}

func (*StringValue) Walk(_ func(Value)) {
	// NO-OP
}

var stringDynamicType DynamicType = StringDynamicType{}

func (*StringValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return stringDynamicType
}

func (*StringValue) StaticType() StaticType {
	return PrimitiveStaticTypeString
}

func (v *StringValue) String() string {
	return format.String(v.Str)
}

func (v *StringValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *StringValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherString, ok := other.(*StringValue)
	if !ok {
		return false
	}
	return v.NormalForm() == otherString.NormalForm()
}

func (v *StringValue) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	return append(
		[]byte{byte(HashInputTypeString)},
		v.Str...,
	)
}

func (v *StringValue) NormalForm() string {
	return norm.NFC.String(v.Str)
}

func (v *StringValue) Concat(other *StringValue) Value {
	var sb strings.Builder

	sb.WriteString(v.Str)
	sb.WriteString(other.Str)

	return NewStringValue(sb.String())
}

func (v *StringValue) Slice(from IntValue, to IntValue, getLocationRange func() LocationRange) Value {
	fromIndex := from.ToInt()
	v.checkBoundsInclusiveLength(fromIndex, getLocationRange)

	toIndex := to.ToInt()
	v.checkBoundsInclusiveLength(toIndex, getLocationRange)

	if fromIndex == toIndex {
		return NewStringValue("")
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

	return NewStringValue(v.Str[start:end])
}

func (v *StringValue) checkBounds(index int, getLocationRange func() LocationRange) {
	length := v.Length()

	if index < 0 || index >= length {
		panic(StringIndexOutOfBoundsError{
			Index:         index,
			Length:        length,
			LocationRange: getLocationRange(),
		})
	}
}

func (v *StringValue) checkBoundsInclusiveLength(index int, getLocationRange func() LocationRange) {
	length := v.Length()

	if index < 0 || index > length {
		panic(StringIndexOutOfBoundsError{
			Index:         index,
			Length:        length,
			LocationRange: getLocationRange(),
		})
	}
}

func (v *StringValue) GetKey(_ *Interpreter, getLocationRange func() LocationRange, key Value) Value {
	index := key.(NumberValue).ToInt()
	v.checkBounds(index, getLocationRange)

	v.prepareGraphemes()

	for j := 0; j <= index; j++ {
		v.graphemes.Next()
	}

	char := v.graphemes.Str()

	return NewStringValue(char)
}

func (*StringValue) SetKey(_ *Interpreter, _ func() LocationRange, _ Value, _ Value) {
	panic(errors.NewUnreachableError())
}

func (*StringValue) InsertKey(_ *Interpreter, _ func() LocationRange, _ Value, _ Value) {
	panic(errors.NewUnreachableError())
}

func (*StringValue) RemoveKey(_ *Interpreter, _ func() LocationRange, _ Value) Value {
	panic(errors.NewUnreachableError())
}

func (v *StringValue) GetMember(interpreter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "length":
		length := v.Length()
		return NewIntValueFromInt64(int64(length))

	case "utf8":
		return ByteSliceToByteArrayValue(interpreter, []byte(v.Str))

	case "concat":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				otherArray := invocation.Arguments[0].(*StringValue)
				return v.Concat(otherArray)
			},
			sema.StringTypeConcatFunctionType,
		)

	case "slice":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				from := invocation.Arguments[0].(IntValue)
				to := invocation.Arguments[1].(IntValue)
				return v.Slice(from, to, invocation.GetLocationRange)
			},
			sema.StringTypeSliceFunctionType,
		)

	case "decodeHex":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.DecodeHex(invocation.Interpreter)
			},
			sema.StringTypeDecodeHexFunctionType,
		)

	case "toLower":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.ToLower()
			},
			sema.StringTypeToLowerFunctionType,
		)
	}

	return nil
}

func (*StringValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Strings have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*StringValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Strings have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

// Length returns the number of characters (grapheme clusters)
//
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

func (v *StringValue) ToLower() *StringValue {
	return NewStringValue(strings.ToLower(v.Str))
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
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

var ByteArrayStaticType = ConvertSemaArrayTypeToStaticArrayType(sema.ByteArrayType)

// DecodeHex hex-decodes this string and returns an array of UInt8 values
//
func (v *StringValue) DecodeHex(interpreter *Interpreter) *ArrayValue {
	bs, err := hex.DecodeString(v.Str)
	if err != nil {
		panic(err)
	}

	i := 0

	return NewArrayValueWithIterator(
		interpreter,
		ByteArrayStaticType,
		common.Address{},
		func() Value {
			if i >= len(bs) {
				return nil
			}

			value := UInt8Value(bs[i])

			i++

			return value
		},
	)
}

func (*StringValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	_, ok := dynamicType.(StringDynamicType)
	return ok
}

// ArrayValue

type ArrayValue struct {
	Type             ArrayStaticType
	semaType         sema.ArrayType
	array            *atree.Array
	isDestroyed      bool
	isResourceKinded *bool
}

func NewArrayValue(
	interpreter *Interpreter,
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
		func() Value {
			if index >= count {
				return nil
			}

			value := values[index]

			index++

			value = value.Transfer(
				interpreter,
				// TODO: provide proper location range
				ReturnEmptyLocationRange,
				atree.Address(address),
				true,
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
	values func() Value,
) *ArrayValue {

	interpreter.ReportComputation(common.ComputationKindCreateArrayValue, 1)

	var v *ArrayValue

	if interpreter.tracingEnabled {
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

	array, err := atree.NewArrayFromBatchData(
		interpreter.Storage,
		atree.Address(address),
		arrayType,
		func() (atree.Value, error) {
			return values(), nil
		},
	)
	if err != nil {
		panic(ExternalError{err})
	}

	v = &ArrayValue{
		Type:  arrayType,
		array: array,
	}

	return v
}

var _ Value = &ArrayValue{}
var _ atree.Value = &ArrayValue{}
var _ EquatableValue = &ArrayValue{}
var _ ValueIndexableValue = &ArrayValue{}
var _ MemberAccessibleValue = &ArrayValue{}
var _ ReferenceTrackedResourceKindedValue = &ArrayValue{}

func (*ArrayValue) IsValue() {}

func (v *ArrayValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitArrayValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(func(element Value) {
		element.Accept(interpreter, visitor)
	})
}

func (v *ArrayValue) Iterate(f func(element Value) (resume bool)) {
	err := v.array.Iterate(func(element atree.Value) (resume bool, err error) {
		// atree.Array iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value

		resume = f(MustConvertStoredValue(element))

		return resume, nil
	})
	if err != nil {
		panic(ExternalError{err})
	}
}

func (v *ArrayValue) Walk(walkChild func(Value)) {
	v.Iterate(func(element Value) (resume bool) {
		walkChild(element)
		return true
	})
}

func (v *ArrayValue) DynamicType(interpreter *Interpreter, seenReferences SeenReferences) DynamicType {
	elementTypes := make([]DynamicType, v.Count())

	i := 0

	v.Walk(func(element Value) {
		elementTypes[i] = element.DynamicType(interpreter, seenReferences)
		i++
	})

	return &ArrayDynamicType{
		ElementTypes: elementTypes,
		StaticType:   v.Type,
	}
}

func (v *ArrayValue) StaticType() StaticType {
	return v.Type
}

func (v *ArrayValue) Destroy(interpreter *Interpreter, getLocationRange func() LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyArrayValue, 1)

	if interpreter.tracingEnabled {
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

	v.Walk(func(element Value) {
		maybeDestroy(interpreter, getLocationRange, element)
	})

	v.isDestroyed = true
}

func (v *ArrayValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *ArrayValue) Concat(interpreter *Interpreter, getLocationRange func() LocationRange, other *ArrayValue) Value {

	first := true

	firstIterator, err := v.array.Iterator()
	if err != nil {
		panic(ExternalError{err})
	}

	secondIterator, err := other.array.Iterator()
	if err != nil {
		panic(ExternalError{err})
	}

	elementType := v.Type.ElementType()

	return NewArrayValueWithIterator(
		interpreter,
		v.Type,
		common.Address{},
		func() Value {

			var value Value

			if first {
				atreeValue, err := firstIterator.Next()
				if err != nil {
					panic(ExternalError{err})
				}

				if atreeValue == nil {
					first = false
				} else {
					value = MustConvertStoredValue(atreeValue)
				}
			}

			if !first {
				atreeValue, err := secondIterator.Next()
				if err != nil {
					panic(ExternalError{err})
				}

				if atreeValue != nil {
					value = MustConvertStoredValue(atreeValue)

					interpreter.checkContainerMutation(elementType, value, getLocationRange)
				}
			}

			if value == nil {
				return nil
			}

			return value.Transfer(
				interpreter,
				getLocationRange,
				atree.Address{},
				false,
				nil,
			)
		},
	)
}

func (v *ArrayValue) GetKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value) Value {
	index := key.(NumberValue).ToInt()
	return v.Get(interpreter, getLocationRange, index)
}

func (v *ArrayValue) handleIndexOutOfBoundsError(err error, index int, getLocationRange func() LocationRange) {
	if _, ok := err.(*atree.IndexOutOfBoundsError); ok {
		panic(ArrayIndexOutOfBoundsError{
			Index:         index,
			Size:          v.Count(),
			LocationRange: getLocationRange(),
		})
	}
}

func (v *ArrayValue) Get(interpreter *Interpreter, getLocationRange func() LocationRange, index int) Value {
	storable, err := v.array.Get(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, getLocationRange)

		panic(ExternalError{err})
	}

	return StoredValue(storable, interpreter.Storage)
}

func (v *ArrayValue) SetKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value, value Value) {
	index := key.(NumberValue).ToInt()
	v.Set(interpreter, getLocationRange, index, value)
}

func (v *ArrayValue) Set(interpreter *Interpreter, getLocationRange func() LocationRange, index int, element Value) {

	interpreter.checkContainerMutation(v.Type.ElementType(), element, getLocationRange)

	element = element.Transfer(
		interpreter,
		getLocationRange,
		v.array.Address(),
		true,
		nil,
	)

	existingStorable, err := v.array.Set(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, getLocationRange)

		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.array)

	existingValue := StoredValue(existingStorable, interpreter.Storage)

	existingValue.DeepRemove(interpreter)

	interpreter.RemoveReferencedSlab(existingStorable)
}

func (v *ArrayValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *ArrayValue) RecursiveString(seenReferences SeenReferences) string {
	values := make([]string, v.Count())

	i := 0
	v.Walk(func(value Value) {
		values[i] = value.RecursiveString(seenReferences)
		i++
	})

	return format.Array(values)
}

func (v *ArrayValue) Append(interpreter *Interpreter, getLocationRange func() LocationRange, element Value) {

	interpreter.checkContainerMutation(v.Type.ElementType(), element, getLocationRange)

	element = element.Transfer(
		interpreter,
		getLocationRange,
		v.array.Address(),
		true,
		nil,
	)

	err := v.array.Append(element)
	if err != nil {
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) AppendAll(interpreter *Interpreter, getLocationRange func() LocationRange, other *ArrayValue) {
	other.Walk(func(value Value) {
		v.Append(interpreter, getLocationRange, value)
	})
}

func (v *ArrayValue) InsertKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value, value Value) {
	index := key.(NumberValue).ToInt()
	v.Insert(interpreter, getLocationRange, index, value)
}

func (v *ArrayValue) Insert(interpreter *Interpreter, getLocationRange func() LocationRange, index int, element Value) {

	interpreter.checkContainerMutation(v.Type.ElementType(), element, getLocationRange)

	element = element.Transfer(
		interpreter,
		getLocationRange,
		v.array.Address(),
		true,
		nil,
	)

	err := v.array.Insert(uint64(index), element)
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, getLocationRange)

		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) RemoveKey(interpreter *Interpreter, getLocationRange func() LocationRange, key Value) Value {
	index := key.(NumberValue).ToInt()
	return v.Remove(interpreter, getLocationRange, index)
}

func (v *ArrayValue) Remove(interpreter *Interpreter, getLocationRange func() LocationRange, index int) Value {
	storable, err := v.array.Remove(uint64(index))
	if err != nil {
		v.handleIndexOutOfBoundsError(err, index, getLocationRange)

		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.array)

	value := StoredValue(storable, interpreter.Storage)

	return value.Transfer(
		interpreter,
		getLocationRange,
		atree.Address{},
		true,
		storable,
	)
}

func (v *ArrayValue) RemoveFirst(interpreter *Interpreter, getLocationRange func() LocationRange) Value {
	return v.Remove(interpreter, getLocationRange, 0)
}

func (v *ArrayValue) RemoveLast(interpreter *Interpreter, getLocationRange func() LocationRange) Value {
	return v.Remove(interpreter, getLocationRange, v.Count()-1)
}

func (v *ArrayValue) Contains(interpreter *Interpreter, getLocationRange func() LocationRange, needleValue Value) BoolValue {

	needleEquatable := needleValue.(EquatableValue)

	var result bool
	v.Iterate(func(element Value) (resume bool) {
		if needleEquatable.Equal(interpreter, getLocationRange, element) {
			result = true
			// stop iteration
			return false
		}
		// continue iteration
		return true
	})

	return BoolValue(result)
}

func (v *ArrayValue) GetMember(inter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "length":
		return NewIntValueFromInt64(int64(v.Count()))

	case "append":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				v.Append(
					invocation.Interpreter,
					invocation.GetLocationRange,
					invocation.Arguments[0],
				)
				return VoidValue{}
			},
			sema.ArrayAppendFunctionType(
				v.SemaType(inter).ElementType(false),
			),
		)

	case "appendAll":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				otherArray := invocation.Arguments[0].(*ArrayValue)
				v.AppendAll(
					invocation.Interpreter,
					invocation.GetLocationRange,
					otherArray,
				)
				return VoidValue{}
			},
			sema.ArrayAppendAllFunctionType(
				v.SemaType(inter),
			),
		)

	case "concat":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				otherArray := invocation.Arguments[0].(*ArrayValue)
				return v.Concat(
					invocation.Interpreter,
					invocation.GetLocationRange,
					otherArray,
				)
			},
			sema.ArrayConcatFunctionType(
				v.SemaType(inter),
			),
		)

	case "insert":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				index := invocation.Arguments[0].(NumberValue).ToInt()
				element := invocation.Arguments[1]
				v.Insert(
					invocation.Interpreter,
					invocation.GetLocationRange,
					index,
					element,
				)
				return VoidValue{}
			},
			sema.ArrayInsertFunctionType(
				v.SemaType(inter).ElementType(false),
			),
		)

	case "remove":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				index := invocation.Arguments[0].(NumberValue).ToInt()
				return v.Remove(
					invocation.Interpreter,
					invocation.GetLocationRange,
					index,
				)
			},
			sema.ArrayRemoveFunctionType(
				v.SemaType(inter).ElementType(false),
			),
		)

	case "removeFirst":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.RemoveFirst(
					invocation.Interpreter,
					invocation.GetLocationRange,
				)
			},
			sema.ArrayRemoveFirstFunctionType(
				v.SemaType(inter).ElementType(false),
			),
		)

	case "removeLast":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.RemoveLast(
					invocation.Interpreter,
					invocation.GetLocationRange,
				)
			},
			sema.ArrayRemoveLastFunctionType(
				v.SemaType(inter).ElementType(false),
			),
		)

	case "contains":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.Contains(
					invocation.Interpreter,
					invocation.GetLocationRange,
					invocation.Arguments[0],
				)
			},
			sema.ArrayContainsFunctionType(
				v.SemaType(inter).ElementType(false),
			),
		)
	}

	return nil
}

func (*ArrayValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Arrays have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*ArrayValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Arrays have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) Count() int {
	return int(v.array.Count())
}

func (v *ArrayValue) ConformsToDynamicType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	count := v.Count()

	if interpreter.tracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			interpreter.reportArrayValueConformsToDynamicTypeTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	arrayType, ok := dynamicType.(*ArrayDynamicType)

	if !ok || count != len(arrayType.ElementTypes) {
		return false
	}

	result := true
	index := 0

	v.Iterate(func(element Value) (resume bool) {
		if !element.ConformsToDynamicType(
			interpreter,
			getLocationRange,
			arrayType.ElementTypes[index],
			results,
		) {
			result = false
			return false
		}

		index++

		return true
	})

	return result
}

func (v *ArrayValue) Equal(interpreter *Interpreter, getLocationRange func() LocationRange, other Value) bool {
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
		value := v.Get(interpreter, getLocationRange, i)
		otherValue := otherArray.Get(interpreter, getLocationRange, i)

		equatableValue, ok := value.(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, getLocationRange, otherValue) {
			return false
		}
	}

	return true
}

func (v *ArrayValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return atree.StorageIDStorable(v.StorageID()), nil
}

func (v *ArrayValue) IsReferenceTrackedResourceKindedValue() {}

func (v *ArrayValue) Transfer(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	interpreter.ReportComputation(common.ComputationKindTransferArrayValue, uint(v.Count()))

	if interpreter.tracingEnabled {
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
	currentAddress := currentStorageID.Address

	array := v.array

	needsStoreTo := address != currentAddress
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		iterator, err := v.array.Iterator()
		if err != nil {
			panic(ExternalError{err})
		}

		array, err = atree.NewArrayFromBatchData(
			interpreter.Storage,
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

				element := MustConvertStoredValue(value).
					Transfer(interpreter, getLocationRange, address, remove, nil)

				return element, nil
			},
		)
		if err != nil {
			panic(ExternalError{err})
		}

		if remove {
			err = v.array.PopIterate(func(storable atree.Storable) {
				interpreter.RemoveReferencedSlab(storable)
			})
			if err != nil {
				panic(ExternalError{err})
			}
			interpreter.maybeValidateAtreeValue(v.array)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		v.array = array

		newStorageID := array.StorageID()
		interpreter.updateReferencedResource(
			currentStorageID,
			newStorageID,
			func(value ReferenceTrackedResourceKindedValue) {
				arrayValue, ok := value.(*ArrayValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				arrayValue.array = array
			},
		)

		return v
	} else {
		return &ArrayValue{
			Type:             v.Type,
			semaType:         v.semaType,
			isResourceKinded: v.isResourceKinded,
			array:            array,
			isDestroyed:      v.isDestroyed,
		}
	}
}

func (v *ArrayValue) DeepRemove(interpreter *Interpreter) {

	if interpreter.tracingEnabled {
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
		value := StoredValue(storable, storage)
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(storable)
	})
	if err != nil {
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.array)
}

func (v *ArrayValue) StorageID() atree.StorageID {
	return v.array.StorageID()
}

func (v *ArrayValue) GetOwner() common.Address {
	return common.Address(v.StorageID().Address)
}

func (v *ArrayValue) SemaType(interpreter *Interpreter) sema.ArrayType {
	if v.semaType == nil {
		v.semaType = interpreter.ConvertStaticToSemaType(v.Type).(sema.ArrayType)
	}
	return v.semaType
}

func (v *ArrayValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageID().Address
}

func (v *ArrayValue) IsResourceKinded(interpreter *Interpreter) bool {
	if v.isResourceKinded == nil {
		isResourceKinded := v.SemaType(interpreter).IsResourceType()
		v.isResourceKinded = &isResourceKinded
	}
	return *v.isResourceKinded
}

// NumberValue
//
type NumberValue interface {
	EquatableValue
	ToInt() int
	Negate() NumberValue
	Plus(other NumberValue) NumberValue
	SaturatingPlus(other NumberValue) NumberValue
	Minus(other NumberValue) NumberValue
	SaturatingMinus(other NumberValue) NumberValue
	Mod(other NumberValue) NumberValue
	Mul(other NumberValue) NumberValue
	SaturatingMul(other NumberValue) NumberValue
	Div(other NumberValue) NumberValue
	SaturatingDiv(other NumberValue) NumberValue
	Less(other NumberValue) BoolValue
	LessEqual(other NumberValue) BoolValue
	Greater(other NumberValue) BoolValue
	GreaterEqual(other NumberValue) BoolValue
	ToBigEndianBytes() []byte
}

func getNumberValueMember(v NumberValue, name string, typ sema.Type) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return NewStringValue(v.String())
			},
			sema.ToStringFunctionType,
		)

	case sema.ToBigEndianBytesFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return ByteSliceToByteArrayValue(
					invocation.Interpreter,
					v.ToBigEndianBytes(),
				)
			},
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					sema.ByteArrayType,
				),
			},
		)

	case sema.NumericTypeSaturatingAddFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingPlus(other)
			},
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					typ,
				),
			},
		)

	case sema.NumericTypeSaturatingSubtractFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingMinus(other)
			},
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					typ,
				),
			},
		)

	case sema.NumericTypeSaturatingMultiplyFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingMul(other)
			},
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					typ,
				),
			},
		)

	case sema.NumericTypeSaturatingDivideFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingDiv(other)
			},
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					typ,
				),
			},
		)
	}

	return nil
}

type IntegerValue interface {
	NumberValue
	BitwiseOr(other IntegerValue) IntegerValue
	BitwiseXor(other IntegerValue) IntegerValue
	BitwiseAnd(other IntegerValue) IntegerValue
	BitwiseLeftShift(other IntegerValue) IntegerValue
	BitwiseRightShift(other IntegerValue) IntegerValue
}

// BigNumberValue.
// Implemented by values with an integer value outside the range of int64
//
type BigNumberValue interface {
	NumberValue
	ToBigInt() *big.Int
}

// Int

type IntValue struct {
	BigInt *big.Int
}

func NewIntValueFromInt64(value int64) IntValue {
	return NewIntValueFromBigInt(big.NewInt(value))
}

func NewIntValueFromBigInt(value *big.Int) IntValue {
	return IntValue{BigInt: value}
}

func ConvertInt(value Value) IntValue {
	switch value := value.(type) {
	case BigNumberValue:
		return NewIntValueFromBigInt(value.ToBigInt())

	case NumberValue:
		return NewIntValueFromInt64(int64(value.ToInt()))

	default:
		panic(errors.NewUnreachableError())
	}
}

var _ Value = IntValue{}
var _ atree.Storable = IntValue{}
var _ NumberValue = IntValue{}
var _ IntegerValue = IntValue{}
var _ EquatableValue = IntValue{}
var _ HashableValue = IntValue{}
var _ MemberAccessibleValue = IntValue{}

func (IntValue) IsValue() {}

func (v IntValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitIntValue(interpreter, v)
}

func (IntValue) Walk(_ func(Value)) {
	// NO-OP
}

var intDynamicType DynamicType = NumberDynamicType{sema.IntType}

func (IntValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return intDynamicType
}

func (IntValue) StaticType() StaticType {
	return PrimitiveStaticTypeInt
}

func (v IntValue) ToInt() int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{})
	}
	return int(v.BigInt.Int64())
}

func (v IntValue) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v IntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v IntValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v IntValue) Negate() NumberValue {
	return NewIntValueFromBigInt(new(big.Int).Neg(v.BigInt))
}

func (v IntValue) Plus(other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Add(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingPlus(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingAddFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Plus(other)
}

func (v IntValue) Minus(other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Sub(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingMinus(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Minus(other)
}

func (v IntValue) Mod(other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) Mul(other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingMul(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Mul(other)
}

func (v IntValue) Div(other NumberValue) NumberValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v IntValue) Less(other NumberValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v IntValue) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v IntValue) Greater(other NumberValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v IntValue) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v IntValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt, ok := other.(IntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

func (v IntValue) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	// TODO: optimize?
	return append(
		[]byte{byte(HashInputTypeInt)},
		SignedBigIntToBigEndianBytes(v.BigInt)...,
	)
}

func (v IntValue) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
	return IntValue{res}
}

func (v IntValue) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(IntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
	return IntValue{res}
}

func (v IntValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.IntType)
}

func (IntValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (IntValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v IntValue) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v IntValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.IntType.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (IntValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v IntValue) ByteSize() uint32 {
	// TODO: optimize
	return mustStorableSize(v)
}

func (v IntValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (IntValue) ChildStorables() []atree.Storable {
	return nil
}

// Int8Value

type Int8Value int8

var _ Value = Int8Value(0)
var _ atree.Storable = Int8Value(0)
var _ NumberValue = Int8Value(0)
var _ IntegerValue = Int8Value(0)
var _ EquatableValue = Int8Value(0)
var _ HashableValue = Int8Value(0)

func (Int8Value) IsValue() {}

func (v Int8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt8Value(interpreter, v)
}

func (Int8Value) Walk(_ func(Value)) {
	// NO-OP
}

var int8DynamicType DynamicType = NumberDynamicType{sema.Int8Type}

func (Int8Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return int8DynamicType
}

func (Int8Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt8
}

func (v Int8Value) String() string {
	return format.Int(int64(v))
}

func (v Int8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int8Value) ToInt() int {
	return int(v)
}

func (v Int8Value) Negate() NumberValue {
	// INT32-C
	if v == math.MinInt8 {
		panic(OverflowError{})
	}
	return -v
}

func (v Int8Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt8 - o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v < (math.MinInt8 - o)) {
		panic(UnderflowError{})
	}
	return v + o
}

func (v Int8Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt8 - o)) {
		return Int8Value(math.MaxInt8)
	} else if (o < 0) && (v < (math.MinInt8 - o)) {
		return Int8Value(math.MinInt8)
	}
	return v + o
}

func (v Int8Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt8 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt8 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int8Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt8 + o)) {
		return Int8Value(math.MinInt8)
	} else if (o < 0) && (v > (math.MaxInt8 + o)) {
		return Int8Value(math.MaxInt8)
	}
	return v - o
}

func (v Int8Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int8Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt8 / o) {
				panic(OverflowError{})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt8 / v) {
				panic(UnderflowError{})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt8 / o) {
				panic(UnderflowError{})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt8 / v)) {
				panic(OverflowError{})
			}
		}
	}
	return v * o
}

func (v Int8Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt8 / o) {
				return Int8Value(math.MaxInt8)
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt8 / v) {
				return Int8Value(math.MinInt8)
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt8 / o) {
				return Int8Value(math.MinInt8)
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt8 / v)) {
				return Int8Value(math.MaxInt8)
			}
		}
	}
	return v * o
}

func (v Int8Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt8) && (o == -1) {
		panic(OverflowError{})
	}
	return v / o
}

func (v Int8Value) SaturatingDiv(other NumberValue) NumberValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt8) && (o == -1) {
		return Int8Value(math.MaxInt8)
	}
	return v / o
}

func (v Int8Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Int8Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Int8Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Int8Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Int8Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt8, ok := other.(Int8Value)
	if !ok {
		return false
	}
	return v == otherInt8
}

func (v Int8Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertInt8(value Value) Int8Value {
	var res int8

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.Int8TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Cmp(sema.Int8TypeMinInt) < 0 {
			panic(UnderflowError{})
		}
		res = int8(v.Int64())

	case NumberValue:
		v := value.ToInt()
		if v > math.MaxInt8 {
			panic(OverflowError{})
		} else if v < math.MinInt8 {
			panic(UnderflowError{})
		}
		res = int8(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return Int8Value(res)
}

func (v Int8Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Int8Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v ^ o
}

func (v Int8Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Int8Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Int8Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Int8Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Int8Type)
}

func (Int8Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int8Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Int8Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int8Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

var _ Value = Int16Value(0)
var _ atree.Storable = Int16Value(0)
var _ NumberValue = Int16Value(0)
var _ IntegerValue = Int16Value(0)
var _ EquatableValue = Int16Value(0)
var _ HashableValue = Int16Value(0)
var _ MemberAccessibleValue = Int16Value(0)

func (Int16Value) IsValue() {}

func (v Int16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt16Value(interpreter, v)
}

func (Int16Value) Walk(_ func(Value)) {
	// NO-OP
}

var int16DynamicType DynamicType = NumberDynamicType{sema.Int16Type}

func (Int16Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return int16DynamicType
}

func (Int16Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt16
}

func (v Int16Value) String() string {
	return format.Int(int64(v))
}

func (v Int16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int16Value) ToInt() int {
	return int(v)
}

func (v Int16Value) Negate() NumberValue {
	// INT32-C
	if v == math.MinInt16 {
		panic(OverflowError{})
	}
	return -v
}

func (v Int16Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt16 - o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v < (math.MinInt16 - o)) {
		panic(UnderflowError{})
	}
	return v + o
}

func (v Int16Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt16 - o)) {
		return Int16Value(math.MaxInt16)
	} else if (o < 0) && (v < (math.MinInt16 - o)) {
		return Int16Value(math.MinInt16)
	}
	return v + o
}

func (v Int16Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt16 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt16 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int16Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt16 + o)) {
		return Int16Value(math.MinInt16)
	} else if (o < 0) && (v > (math.MaxInt16 + o)) {
		return Int16Value(math.MaxInt16)
	}
	return v - o
}

func (v Int16Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int16Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt16 / o) {
				panic(OverflowError{})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt16 / v) {
				panic(UnderflowError{})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt16 / o) {
				panic(UnderflowError{})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt16 / v)) {
				panic(OverflowError{})
			}
		}
	}
	return v * o
}

func (v Int16Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt16 / o) {
				return Int16Value(math.MaxInt16)
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt16 / v) {
				return Int16Value(math.MinInt16)
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt16 / o) {
				return Int16Value(math.MinInt16)
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt16 / v)) {
				return Int16Value(math.MaxInt16)
			}
		}
	}
	return v * o
}

func (v Int16Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt16) && (o == -1) {
		panic(OverflowError{})
	}
	return v / o
}

func (v Int16Value) SaturatingDiv(other NumberValue) NumberValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt16) && (o == -1) {
		return Int16Value(math.MaxInt16)
	}
	return v / o
}

func (v Int16Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Int16Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Int16Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Int16Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Int16Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt16, ok := other.(Int16Value)
	if !ok {
		return false
	}
	return v == otherInt16
}

func (v Int16Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertInt16(value Value) Int16Value {
	var res int16

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.Int16TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Cmp(sema.Int16TypeMinInt) < 0 {
			panic(UnderflowError{})
		}
		res = int16(v.Int64())

	case NumberValue:
		v := value.ToInt()
		if v > math.MaxInt16 {
			panic(OverflowError{})
		} else if v < math.MinInt16 {
			panic(UnderflowError{})
		}
		res = int16(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return Int16Value(res)
}

func (v Int16Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Int16Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v ^ o
}

func (v Int16Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Int16Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Int16Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Int16Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Int16Type)
}

func (Int16Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int16Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Int16Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int16Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

var _ Value = Int32Value(0)
var _ atree.Storable = Int32Value(0)
var _ NumberValue = Int32Value(0)
var _ IntegerValue = Int32Value(0)
var _ EquatableValue = Int32Value(0)
var _ HashableValue = Int32Value(0)
var _ MemberAccessibleValue = Int32Value(0)

func (Int32Value) IsValue() {}

func (v Int32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt32Value(interpreter, v)
}

func (Int32Value) Walk(_ func(Value)) {
	// NO-OP
}

var int32DynamicType DynamicType = NumberDynamicType{sema.Int32Type}

func (Int32Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return int32DynamicType
}

func (Int32Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt32
}

func (v Int32Value) String() string {
	return format.Int(int64(v))
}

func (v Int32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int32Value) ToInt() int {
	return int(v)
}

func (v Int32Value) Negate() NumberValue {
	// INT32-C
	if v == math.MinInt32 {
		panic(OverflowError{})
	}
	return -v
}

func (v Int32Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt32 - o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v < (math.MinInt32 - o)) {
		panic(UnderflowError{})
	}
	return v + o
}

func (v Int32Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt32 - o)) {
		return Int32Value(math.MaxInt32)
	} else if (o < 0) && (v < (math.MinInt32 - o)) {
		return Int32Value(math.MinInt32)
	}
	return v + o
}

func (v Int32Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt32 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt32 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int32Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt32 + o)) {
		return Int32Value(math.MinInt32)
	} else if (o < 0) && (v > (math.MaxInt32 + o)) {
		return Int32Value(math.MaxInt32)
	}
	return v - o
}

func (v Int32Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int32Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt32 / o) {
				panic(OverflowError{})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt32 / v) {
				panic(UnderflowError{})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt32 / o) {
				panic(UnderflowError{})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt32 / v)) {
				panic(OverflowError{})
			}
		}
	}
	return v * o
}

func (v Int32Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt32 / o) {
				return Int32Value(math.MaxInt32)
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt32 / v) {
				return Int32Value(math.MinInt32)
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt32 / o) {
				return Int32Value(math.MinInt32)
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt32 / v)) {
				return Int32Value(math.MaxInt32)
			}
		}
	}
	return v * o
}

func (v Int32Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt32) && (o == -1) {
		panic(OverflowError{})
	}
	return v / o
}

func (v Int32Value) SaturatingDiv(other NumberValue) NumberValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt32) && (o == -1) {
		return Int32Value(math.MaxInt32)
	}
	return v / o
}

func (v Int32Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Int32Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Int32Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Int32Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Int32Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt32, ok := other.(Int32Value)
	if !ok {
		return false
	}
	return v == otherInt32
}

func (v Int32Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertInt32(value Value) Int32Value {
	var res int32

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.Int32TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Cmp(sema.Int32TypeMinInt) < 0 {
			panic(UnderflowError{})
		}
		res = int32(v.Int64())

	case NumberValue:
		v := value.ToInt()
		if v > math.MaxInt32 {
			panic(OverflowError{})
		} else if v < math.MinInt32 {
			panic(UnderflowError{})
		}
		res = int32(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return Int32Value(res)
}

func (v Int32Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Int32Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v ^ o
}

func (v Int32Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Int32Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Int32Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Int32Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Int32Type)
}

func (Int32Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int32Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Int32Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int32Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

var _ Value = Int64Value(0)
var _ atree.Storable = Int64Value(0)
var _ NumberValue = Int64Value(0)
var _ IntegerValue = Int64Value(0)
var _ EquatableValue = Int64Value(0)
var _ HashableValue = Int64Value(0)
var _ MemberAccessibleValue = Int64Value(0)

func (Int64Value) IsValue() {}

func (v Int64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt64Value(interpreter, v)
}

func (Int64Value) Walk(_ func(Value)) {
	// NO-OP
}

var int64DynamicType DynamicType = NumberDynamicType{sema.Int64Type}

func (Int64Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return int64DynamicType
}

func (Int64Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt64
}

func (v Int64Value) String() string {
	return format.Int(int64(v))
}

func (v Int64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int64Value) ToInt() int {
	return int(v)
}

func (v Int64Value) Negate() NumberValue {
	// INT32-C
	if v == math.MinInt64 {
		panic(OverflowError{})
	}
	return -v
}

func safeAddInt64(a, b int64) int64 {
	// INT32-C
	if (b > 0) && (a > (math.MaxInt64 - b)) {
		panic(OverflowError{})
	} else if (b < 0) && (a < (math.MinInt64 - b)) {
		panic(UnderflowError{})
	}
	return a + b
}

func (v Int64Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return Int64Value(safeAddInt64(int64(v), int64(o)))
}

func (v Int64Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt64 - o)) {
		return Int64Value(math.MaxInt64)
	} else if (o < 0) && (v < (math.MinInt64 - o)) {
		return Int64Value(math.MinInt64)
	}
	return v + o
}

func (v Int64Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int64Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		return Int64Value(math.MinInt64)
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		return Int64Value(math.MaxInt64)
	}
	return v - o
}

func (v Int64Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int64Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt64 / o) {
				panic(OverflowError{})
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt64 / v) {
				panic(UnderflowError{})
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt64 / o) {
				panic(UnderflowError{})
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt64 / v)) {
				panic(OverflowError{})
			}
		}
	}
	return v * o
}

func (v Int64Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if v > 0 {
		if o > 0 {
			// positive * positive = positive. overflow?
			if v > (math.MaxInt64 / o) {
				return Int64Value(math.MaxInt64)
			}
		} else {
			// positive * negative = negative. underflow?
			if o < (math.MinInt64 / v) {
				return Int64Value(math.MinInt64)
			}
		}
	} else {
		if o > 0 {
			// negative * positive = negative. underflow?
			if v < (math.MinInt64 / o) {
				return Int64Value(math.MinInt64)
			}
		} else {
			// negative * negative = positive. overflow?
			if (v != 0) && (o < (math.MaxInt64 / v)) {
				return Int64Value(math.MaxInt64)
			}
		}
	}
	return v * o
}

func (v Int64Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt64) && (o == -1) {
		panic(OverflowError{})
	}
	return v / o
}

func (v Int64Value) SaturatingDiv(other NumberValue) NumberValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT33-C
	// https://golang.org/ref/spec#Integer_operators
	if o == 0 {
		panic(DivisionByZeroError{})
	} else if (v == math.MinInt64) && (o == -1) {
		return Int64Value(math.MaxInt64)
	}
	return v / o
}

func (v Int64Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Int64Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Int64Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Int64Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Int64Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt64, ok := other.(Int64Value)
	if !ok {
		return false
	}
	return v == otherInt64
}

func (v Int64Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeInt64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertInt64(value Value) Int64Value {
	var res int64

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.Int64TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Cmp(sema.Int64TypeMinInt) < 0 {
			panic(UnderflowError{})
		}
		res = v.Int64()

	case NumberValue:
		v := value.ToInt()
		res = int64(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return Int64Value(res)
}

func (v Int64Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Int64Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v ^ o
}

func (v Int64Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Int64Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Int64Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Int64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Int64Type)
}

func (Int64Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Int64Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int64Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

func NewInt128ValueFromInt64(value int64) Int128Value {
	return NewInt128ValueFromBigInt(big.NewInt(value))
}

func NewInt128ValueFromBigInt(value *big.Int) Int128Value {
	return Int128Value{BigInt: value}
}

var _ Value = Int128Value{}
var _ atree.Storable = Int128Value{}
var _ NumberValue = Int128Value{}
var _ IntegerValue = Int128Value{}
var _ EquatableValue = Int128Value{}
var _ HashableValue = Int128Value{}
var _ MemberAccessibleValue = Int128Value{}

func (Int128Value) IsValue() {}

func (v Int128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt128Value(interpreter, v)
}

func (Int128Value) Walk(_ func(Value)) {
	// NO-OP
}

var int128DynamicType DynamicType = NumberDynamicType{sema.Int128Type}

func (Int128Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return int128DynamicType
}

func (Int128Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt128
}

func (v Int128Value) ToInt() int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{})
	}
	return int(v.BigInt.Int64())
}

func (v Int128Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v Int128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int128Value) Negate() NumberValue {
	// INT32-C
	//   if v == Int128TypeMinIntBig {
	//       ...
	//   }
	if v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0 {
		panic(OverflowError{})
	}
	return Int128Value{new(big.Int).Neg(v.BigInt)}
}

func (v Int128Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(UnderflowError{})
	} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return Int128Value{res}
}

func (v Int128Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return Int128Value{sema.Int128TypeMinIntBig}
	} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		return Int128Value{sema.Int128TypeMaxIntBig}
	}
	return Int128Value{res}
}

func (v Int128Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(UnderflowError{})
	} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return Int128Value{res}
}

func (v Int128Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return Int128Value{sema.Int128TypeMinIntBig}
	} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		return Int128Value{sema.Int128TypeMaxIntBig}
	}
	return Int128Value{res}
}

func (v Int128Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
		panic(UnderflowError{})
	} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return Int128Value{res}
}

func (v Int128Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.Int128TypeMinIntBig) < 0 {
		return Int128Value{sema.Int128TypeMinIntBig}
	} else if res.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		return Int128Value{sema.Int128TypeMaxIntBig}
	}
	return Int128Value{res}
}

func (v Int128Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C:
	//   if o == 0 {
	//       ...
	//   } else if (v == Int128TypeMinIntBig) && (o == -1) {
	//       ...
	//   }
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.SetInt64(-1)
	if (v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
		panic(OverflowError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) SaturatingDiv(other NumberValue) NumberValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C:
	//   if o == 0 {
	//       ...
	//   } else if (v == Int128TypeMinIntBig) && (o == -1) {
	//       ...
	//   }
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.SetInt64(-1)
	if (v.BigInt.Cmp(sema.Int128TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
		return Int128Value{sema.Int128TypeMaxIntBig}
	}
	res.Div(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v Int128Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v Int128Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v Int128Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v Int128Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt, ok := other.(Int128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

func (v Int128Value) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	// TODO: optimize?
	return append(
		[]byte{byte(HashInputTypeInt128)},
		SignedBigIntToBigEndianBytes(v.BigInt)...,
	)
}

func ConvertInt128(value Value) Int128Value {
	var v *big.Int

	switch value := value.(type) {
	case BigNumberValue:
		v = value.ToBigInt()

	case NumberValue:
		v = big.NewInt(int64(value.ToInt()))

	default:
		panic(errors.NewUnreachableError())
	}

	if v.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	} else if v.Cmp(sema.Int128TypeMinIntBig) < 0 {
		panic(UnderflowError{})
	}

	return NewInt128ValueFromBigInt(v)
}

func (v Int128Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
	return Int128Value{res}
}

func (v Int128Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
	return Int128Value{res}
}

func (v Int128Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Int128Type)
}

func (Int128Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int128Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int128Value) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Int128Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int128Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (Int128Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int128Value) ByteSize() uint32 {
	// TODO: optimize
	return mustStorableSize(v)
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

func NewInt256ValueFromInt64(value int64) Int256Value {
	return NewInt256ValueFromBigInt(big.NewInt(value))
}

func NewInt256ValueFromBigInt(value *big.Int) Int256Value {
	return Int256Value{BigInt: value}
}

var _ Value = Int256Value{}
var _ atree.Storable = Int256Value{}
var _ NumberValue = Int256Value{}
var _ IntegerValue = Int256Value{}
var _ EquatableValue = Int256Value{}
var _ HashableValue = Int256Value{}
var _ MemberAccessibleValue = Int256Value{}

func (Int256Value) IsValue() {}

func (v Int256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt256Value(interpreter, v)
}

func (Int256Value) Walk(_ func(Value)) {
	// NO-OP
}

var int256DynamicType DynamicType = NumberDynamicType{sema.Int256Type}

func (Int256Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return int256DynamicType
}

func (Int256Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt256
}

func (v Int256Value) ToInt() int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{})
	}
	return int(v.BigInt.Int64())
}

func (v Int256Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v Int256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Int256Value) Negate() NumberValue {
	// INT32-C
	//   if v == Int256TypeMinIntBig {
	//       ...
	//   }
	if v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0 {
		panic(OverflowError{})
	}
	return Int256Value{BigInt: new(big.Int).Neg(v.BigInt)}
}

func (v Int256Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(UnderflowError{})
	} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return Int256Value{res}
}

func (v Int256Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return Int256Value{sema.Int256TypeMinIntBig}
	} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		return Int256Value{sema.Int256TypeMaxIntBig}
	}
	return Int256Value{res}
}

func (v Int256Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(UnderflowError{})
	} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return Int256Value{res}
}

func (v Int256Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return Int256Value{sema.Int256TypeMinIntBig}
	} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		return Int256Value{sema.Int256TypeMaxIntBig}
	}
	return Int256Value{res}
}

func (v Int256Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
		panic(UnderflowError{})
	} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return Int256Value{res}
}

func (v Int256Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.Int256TypeMinIntBig) < 0 {
		return Int256Value{sema.Int256TypeMinIntBig}
	} else if res.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		return Int256Value{sema.Int256TypeMaxIntBig}
	}
	return Int256Value{res}
}

func (v Int256Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C:
	//   if o == 0 {
	//       ...
	//   } else if (v == Int256TypeMinIntBig) && (o == -1) {
	//       ...
	//   }
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.SetInt64(-1)
	if (v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
		panic(OverflowError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) SaturatingDiv(other NumberValue) NumberValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C:
	//   if o == 0 {
	//       ...
	//   } else if (v == Int256TypeMinIntBig) && (o == -1) {
	//       ...
	//   }
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.SetInt64(-1)
	if (v.BigInt.Cmp(sema.Int256TypeMinIntBig) == 0) && (o.BigInt.Cmp(res) == 0) {
		return Int256Value{sema.Int256TypeMaxIntBig}
	}
	res.Div(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v Int256Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v Int256Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v Int256Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v Int256Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt, ok := other.(Int256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

func (v Int256Value) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	// TODO: optimize?
	return append(
		[]byte{byte(HashInputTypeInt256)},
		SignedBigIntToBigEndianBytes(v.BigInt)...,
	)
}

func ConvertInt256(value Value) Int256Value {
	var v *big.Int

	switch value := value.(type) {
	case BigNumberValue:
		v = value.ToBigInt()

	case NumberValue:
		v = big.NewInt(int64(value.ToInt()))

	default:
		panic(errors.NewUnreachableError())
	}

	if v.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	} else if v.Cmp(sema.Int256TypeMinIntBig) < 0 {
		panic(UnderflowError{})
	}

	return NewInt256ValueFromBigInt(v)
}

func (v Int256Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
	return Int256Value{res}
}

func (v Int256Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Int256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
	return Int256Value{res}
}

func (v Int256Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Int256Type)
}

func (Int256Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Int256Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Int256Value) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Int256Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int256Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (Int256Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v Int256Value) ByteSize() uint32 {
	// TODO: optimize
	return mustStorableSize(v)
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

func NewUIntValueFromUint64(value uint64) UIntValue {
	return NewUIntValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUIntValueFromBigInt(value *big.Int) UIntValue {
	return UIntValue{BigInt: value}
}

func ConvertUInt(value Value) UIntValue {
	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Sign() < 0 {
			panic(UnderflowError{})
		}
		return NewUIntValueFromBigInt(value.ToBigInt())

	case NumberValue:
		v := value.ToInt()
		if v < 0 {
			panic(UnderflowError{})
		}
		return NewUIntValueFromUint64(uint64(v))

	default:
		panic(errors.NewUnreachableError())
	}
}

var _ Value = UIntValue{}
var _ atree.Storable = UIntValue{}
var _ NumberValue = UIntValue{}
var _ IntegerValue = UIntValue{}
var _ EquatableValue = UIntValue{}
var _ HashableValue = UIntValue{}
var _ MemberAccessibleValue = UIntValue{}

func (UIntValue) IsValue() {}

func (v UIntValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUIntValue(interpreter, v)
}

func (UIntValue) Walk(_ func(Value)) {
	// NO-OP
}

var uintDynamicType DynamicType = NumberDynamicType{sema.UIntType}

func (UIntValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return uintDynamicType
}

func (UIntValue) StaticType() StaticType {
	return PrimitiveStaticTypeUInt
}

func (v UIntValue) ToInt() int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{})
	}
	return int(v.BigInt.Int64())
}

func (v UIntValue) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v UIntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v UIntValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UIntValue) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UIntValue) Plus(other NumberValue) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Add(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) SaturatingPlus(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingAddFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Plus(other)
}

func (v UIntValue) Minus(other NumberValue) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Sub(v.BigInt, o.BigInt)
	// INT30-C
	if res.Sign() < 0 {
		panic(UnderflowError{})
	}
	return UIntValue{res}
}

func (v UIntValue) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Sub(v.BigInt, o.BigInt)
	// INT30-C
	if res.Sign() < 0 {
		return UIntValue{sema.UIntTypeMin}
	}
	return UIntValue{res}
}

func (v UIntValue) Mod(other NumberValue) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) Mul(other NumberValue) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) SaturatingMul(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Mul(other)
}

func (v UIntValue) Div(other NumberValue) NumberValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UIntValue) Less(other NumberValue) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v UIntValue) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v UIntValue) Greater(other NumberValue) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v UIntValue) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v UIntValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherUInt, ok := other.(UIntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherUInt.BigInt)
	return cmp == 0
}

func (v UIntValue) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	// TODO: optimize?
	return append(
		[]byte{byte(HashInputTypeUInt)},
		UnsignedBigIntToBigEndianBytes(v.BigInt)...,
	)
}

func (v UIntValue) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
	return UIntValue{res}
}

func (v UIntValue) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(UIntValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
	return UIntValue{res}
}

func (v UIntValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UIntType)
}

func (UIntValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UIntValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UIntValue) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UIntValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UIntType.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (UIntValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UIntValue) ByteSize() uint32 {
	// TODO: optimize
	return mustStorableSize(v)
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
var _ HashableValue = UInt8Value(0)
var _ MemberAccessibleValue = UInt8Value(0)

func (UInt8Value) IsValue() {}

func (v UInt8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt8Value(interpreter, v)
}

func (UInt8Value) Walk(_ func(Value)) {
	// NO-OP
}

var uint8DynamicType DynamicType = NumberDynamicType{sema.UInt8Type}

func (UInt8Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return uint8DynamicType
}

func (UInt8Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt8
}

func (v UInt8Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt8Value) ToInt() int {
	return int(v)
}

func (v UInt8Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt8Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	sum := v + o
	// INT30-C
	if sum < v {
		panic(OverflowError{})
	}
	return sum
}

func (v UInt8Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	sum := v + o
	// INT30-C
	if sum < v {
		return UInt8Value(math.MaxUint8)
	}
	return sum
}

func (v UInt8Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt8Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		return UInt8Value(0)
	}
	return diff
}

func (v UInt8Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt8Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint8 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt8Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint8 / o)) {
		return UInt8Value(math.MaxUint8)
	}
	return v * o
}

func (v UInt8Value) Div(other NumberValue) NumberValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt8Value) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UInt8Value) Less(other NumberValue) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v UInt8Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v UInt8Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v UInt8Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v UInt8Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherUInt8, ok := other.(UInt8Value)
	if !ok {
		return false
	}
	return v == otherUInt8
}

func (v UInt8Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertUInt8(value Value) UInt8Value {
	var res uint8

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.UInt8TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Sign() < 0 {
			panic(UnderflowError{})
		}
		res = uint8(v.Int64())

	case NumberValue:
		v := value.ToInt()
		if v > math.MaxUint8 {
			panic(OverflowError{})
		} else if v < 0 {
			panic(UnderflowError{})
		}
		res = uint8(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return UInt8Value(res)
}

func (v UInt8Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v UInt8Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v ^ o
}

func (v UInt8Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v UInt8Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v UInt8Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v UInt8Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UInt8Type)
}

func (UInt8Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt8Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v UInt8Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt8Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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
var _ HashableValue = UInt16Value(0)
var _ MemberAccessibleValue = UInt16Value(0)

func (UInt16Value) IsValue() {}

func (v UInt16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt16Value(interpreter, v)
}

func (UInt16Value) Walk(_ func(Value)) {
	// NO-OP
}

var uint16DynamicType DynamicType = NumberDynamicType{sema.UInt16Type}

func (UInt16Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return uint16DynamicType
}

func (UInt16Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt16
}

func (v UInt16Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt16Value) ToInt() int {
	return int(v)
}
func (v UInt16Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	sum := v + o
	// INT30-C
	if sum < v {
		panic(OverflowError{})
	}
	return sum
}

func (v UInt16Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	sum := v + o
	// INT30-C
	if sum < v {
		return UInt16Value(math.MaxUint16)
	}
	return sum
}

func (v UInt16Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt16Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		return UInt16Value(0)
	}
	return diff
}

func (v UInt16Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt16Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt16Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
		return UInt16Value(math.MaxUint16)
	}
	return v * o
}

func (v UInt16Value) Div(other NumberValue) NumberValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt16Value) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UInt16Value) Less(other NumberValue) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v UInt16Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v UInt16Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v UInt16Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v UInt16Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherUInt16, ok := other.(UInt16Value)
	if !ok {
		return false
	}
	return v == otherUInt16
}

func (v UInt16Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertUInt16(value Value) UInt16Value {
	var res uint16

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.UInt16TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Sign() < 0 {
			panic(UnderflowError{})
		}
		res = uint16(v.Int64())

	case NumberValue:
		v := value.ToInt()
		if v > math.MaxUint16 {
			panic(OverflowError{})
		} else if v < 0 {
			panic(UnderflowError{})
		}
		res = uint16(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return UInt16Value(res)
}

func (v UInt16Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v UInt16Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}
	return v ^ o
}

func (v UInt16Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v UInt16Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v UInt16Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v UInt16Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UInt16Type)
}

func (UInt16Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt16Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v UInt16Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt16Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

var _ Value = UInt32Value(0)
var _ atree.Storable = UInt32Value(0)
var _ NumberValue = UInt32Value(0)
var _ IntegerValue = UInt32Value(0)
var _ EquatableValue = UInt32Value(0)
var _ HashableValue = UInt32Value(0)
var _ MemberAccessibleValue = UInt32Value(0)

func (UInt32Value) IsValue() {}

func (v UInt32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt32Value(interpreter, v)
}

func (UInt32Value) Walk(_ func(Value)) {
	// NO-OP
}

var uint32DynamicType DynamicType = NumberDynamicType{sema.UInt32Type}

func (UInt32Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return uint32DynamicType
}

func (UInt32Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt32
}

func (v UInt32Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt32Value) ToInt() int {
	return int(v)
}

func (v UInt32Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	sum := v + o

	// INT30-C
	if sum < v {
		panic(OverflowError{})
	}
	return sum
}

func (v UInt32Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	sum := v + o
	// INT30-C
	if sum < v {
		return UInt32Value(math.MaxUint32)
	}
	return sum
}

func (v UInt32Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt32Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		return UInt32Value(0)
	}
	return diff
}

func (v UInt32Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt32Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt32Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
		return UInt32Value(math.MaxUint32)
	}
	return v * o
}

func (v UInt32Value) Div(other NumberValue) NumberValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt32Value) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UInt32Value) Less(other NumberValue) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v UInt32Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v UInt32Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v UInt32Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v UInt32Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherUInt32, ok := other.(UInt32Value)
	if !ok {
		return false
	}
	return v == otherUInt32
}

func (v UInt32Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertUInt32(value Value) UInt32Value {
	var res uint32

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.UInt32TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Sign() < 0 {
			panic(UnderflowError{})
		}
		res = uint32(v.Int64())

	case NumberValue:
		v := value.ToInt()
		if v > math.MaxUint32 {
			panic(OverflowError{})
		} else if v < 0 {
			panic(UnderflowError{})
		}
		res = uint32(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return UInt32Value(res)
}

func (v UInt32Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v UInt32Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}
	return v ^ o
}

func (v UInt32Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v UInt32Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v UInt32Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v UInt32Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UInt32Type)
}

func (UInt32Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt32Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v UInt32Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt32Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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
var _ HashableValue = UInt64Value(0)
var _ MemberAccessibleValue = UInt64Value(0)

// NOTE: important, do *NOT* remove:
// UInt64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
//
var _ BigNumberValue = UInt64Value(0)

func (UInt64Value) IsValue() {}

func (v UInt64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt64Value(interpreter, v)
}

func (UInt64Value) Walk(_ func(Value)) {
	// NO-OP
}

var uint64DynamicType DynamicType = NumberDynamicType{sema.UInt64Type}

func (UInt64Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return uint64DynamicType
}

func (UInt64Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt64
}

func (v UInt64Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt64Value) ToInt() int {
	if v > math.MaxInt64 {
		panic(OverflowError{})
	}
	return int(v)
}

// ToBigInt
//
// NOTE: important, do *NOT* remove:
// UInt64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
//
func (v UInt64Value) ToBigInt() *big.Int {
	return new(big.Int).SetUint64(uint64(v))
}

func (v UInt64Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func safeAddUint64(a, b uint64) uint64 {
	sum := a + b
	// INT30-C
	if sum < a {
		panic(OverflowError{})
	}
	return sum
}

func (v UInt64Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return UInt64Value(safeAddUint64(uint64(v), uint64(o)))
}

func (v UInt64Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	sum := v + o

	// INT30-C
	if sum < v {
		return UInt64Value(math.MaxUint64)
	}
	return sum
}

func (v UInt64Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt64Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		return UInt64Value(0)
	}
	return diff
}

func (v UInt64Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt64Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt64Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
		return UInt64Value(math.MaxUint64)
	}
	return v * o
}

func (v UInt64Value) Div(other NumberValue) NumberValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt64Value) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UInt64Value) Less(other NumberValue) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v UInt64Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v UInt64Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v UInt64Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v UInt64Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherUInt64, ok := other.(UInt64Value)
	if !ok {
		return false
	}
	return v == otherUInt64
}

func (v UInt64Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUInt64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertUInt64(value Value) UInt64Value {
	var res uint64

	switch value := value.(type) {
	case BigNumberValue:
		v := value.ToBigInt()
		if v.Cmp(sema.UInt64TypeMaxInt) > 0 {
			panic(OverflowError{})
		} else if v.Sign() < 0 {
			panic(UnderflowError{})
		}
		res = uint64(v.Int64())

	case NumberValue:
		v := value.ToInt()
		if v < 0 {
			panic(UnderflowError{})
		}
		res = uint64(v)

	default:
		panic(errors.NewUnreachableError())
	}

	return UInt64Value(res)
}

func (v UInt64Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v UInt64Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}
	return v ^ o
}

func (v UInt64Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v UInt64Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v UInt64Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v UInt64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UInt64Type)
}

func (UInt64Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UInt64Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt64Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

func NewUInt128ValueFromUint64(value uint64) UInt128Value {
	return NewUInt128ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUInt128ValueFromBigInt(value *big.Int) UInt128Value {
	return UInt128Value{BigInt: value}
}

var _ Value = UInt128Value{}
var _ atree.Storable = UInt128Value{}
var _ NumberValue = UInt128Value{}
var _ IntegerValue = UInt128Value{}
var _ EquatableValue = UInt128Value{}
var _ HashableValue = UInt128Value{}
var _ MemberAccessibleValue = UInt128Value{}

func (UInt128Value) IsValue() {}

func (v UInt128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt128Value(interpreter, v)
}

func (UInt128Value) Walk(_ func(Value)) {
	// NO-OP
}

var uint128DynamicType DynamicType = NumberDynamicType{sema.UInt128Type}

func (UInt128Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return uint128DynamicType
}

func (UInt128Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt128
}

func (v UInt128Value) ToInt() int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{})
	}
	return int(v.BigInt.Int64())
}

func (v UInt128Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v UInt128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v UInt128Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt128Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt128Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(OverflowError{})
	}
	return UInt128Value{sum}
}

func (v UInt128Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return UInt128Value{sema.UInt128TypeMaxIntBig}
	}
	return UInt128Value{sum}
}

func (v UInt128Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(UnderflowError{})
	}
	return UInt128Value{diff}
}

func (v UInt128Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return UInt128Value{sema.UInt128TypeMinIntBig}
	}
	return UInt128Value{diff}
}

func (v UInt128Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return UInt128Value{res}
}

func (v UInt128Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
		return UInt128Value{sema.UInt128TypeMaxIntBig}
	}
	return UInt128Value{res}
}

func (v UInt128Value) Div(other NumberValue) NumberValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UInt128Value) Less(other NumberValue) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v UInt128Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v UInt128Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v UInt128Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v UInt128Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt, ok := other.(UInt128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

func (v UInt128Value) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	// TODO: optimize?
	return append(
		[]byte{byte(HashInputTypeUInt128)},
		UnsignedBigIntToBigEndianBytes(v.BigInt)...,
	)
}

func ConvertUInt128(value Value) UInt128Value {
	var v *big.Int

	switch value := value.(type) {
	case BigNumberValue:
		v = value.ToBigInt()

	case NumberValue:
		v = big.NewInt(int64(value.ToInt()))

	default:
		panic(errors.NewUnreachableError())
	}

	if v.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	} else if v.Sign() < 0 {
		panic(UnderflowError{})
	}

	return NewUInt128ValueFromBigInt(v)
}

func (v UInt128Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
	return UInt128Value{res}
}

func (v UInt128Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt128Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
	return UInt128Value{res}
}

func (v UInt128Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UInt128Type)
}

func (UInt128Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt128Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt128Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UInt128Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt128Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (UInt128Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt128Value) ByteSize() uint32 {
	// TODO: optimize
	return mustStorableSize(v)
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

func NewUInt256ValueFromUint64(value uint64) UInt256Value {
	return NewUInt256ValueFromBigInt(new(big.Int).SetUint64(value))
}

func NewUInt256ValueFromBigInt(value *big.Int) UInt256Value {
	return UInt256Value{BigInt: value}
}

var _ Value = UInt256Value{}
var _ atree.Storable = UInt256Value{}
var _ NumberValue = UInt256Value{}
var _ IntegerValue = UInt256Value{}
var _ EquatableValue = UInt256Value{}
var _ HashableValue = UInt256Value{}
var _ MemberAccessibleValue = UInt256Value{}

func (UInt256Value) IsValue() {}

func (v UInt256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt256Value(interpreter, v)
}

func (UInt256Value) Walk(_ func(Value)) {
	// NO-OP
}

var uint256DynamicType DynamicType = NumberDynamicType{sema.UInt256Type}

func (UInt256Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return uint256DynamicType
}

func (UInt256Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt256
}

func (v UInt256Value) ToInt() int {
	if !v.BigInt.IsInt64() {
		panic(OverflowError{})
	}

	return int(v.BigInt.Int64())
}

func (v UInt256Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v UInt256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v UInt256Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UInt256Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(OverflowError{})
	}
	return UInt256Value{sum}
}

func (v UInt256Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return UInt256Value{sema.UInt256TypeMaxIntBig}
	}
	return UInt256Value{sum}
}

func (v UInt256Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

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
		panic(UnderflowError{})
	}
	return UInt256Value{diff}
}

func (v UInt256Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

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
		return UInt256Value{sema.UInt256TypeMinIntBig}
	}
	return UInt256Value{diff}
}

func (v UInt256Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return UInt256Value{res}
}

func (v UInt256Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
		return UInt256Value{sema.UInt256TypeMaxIntBig}
	}
	return UInt256Value{res}
}

func (v UInt256Value) Div(other NumberValue) NumberValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UInt256Value) Less(other NumberValue) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == -1
}

func (v UInt256Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp <= 0
}

func (v UInt256Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp == 1
}

func (v UInt256Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	cmp := v.BigInt.Cmp(o.BigInt)
	return cmp >= 0
}

func (v UInt256Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherInt, ok := other.(UInt256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

func (v UInt256Value) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	// TODO: optimize?
	return append(
		[]byte{byte(HashInputTypeUInt256)},
		UnsignedBigIntToBigEndianBytes(v.BigInt)...,
	)
}

func ConvertUInt256(value Value) UInt256Value {
	var v *big.Int

	switch value := value.(type) {
	case BigNumberValue:
		v = value.ToBigInt()

	case NumberValue:
		v = big.NewInt(int64(value.ToInt()))

	default:
		panic(errors.NewUnreachableError())
	}

	if v.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	} else if v.Sign() < 0 {
		panic(UnderflowError{})
	}

	return NewUInt256ValueFromBigInt(v)
}

func (v UInt256Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Lsh(v.BigInt, uint(o.BigInt.Uint64()))
	return UInt256Value{res}
}

func (v UInt256Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(UInt256Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	res := new(big.Int)
	if o.BigInt.Sign() < 0 {
		panic(UnderflowError{})
	}
	if !o.BigInt.IsUint64() {
		panic(OverflowError{})
	}
	res.Rsh(v.BigInt, uint(o.BigInt.Uint64()))
	return UInt256Value{res}
}

func (v UInt256Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UInt256Type)
}

func (UInt256Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UInt256Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UInt256Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt256Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (UInt256Value) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v UInt256Value) ByteSize() uint32 {
	// TODO: optimize
	return mustStorableSize(v)
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
var _ HashableValue = Word8Value(0)
var _ MemberAccessibleValue = Word8Value(0)

func (Word8Value) IsValue() {}

func (v Word8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord8Value(interpreter, v)
}

func (Word8Value) Walk(_ func(Value)) {
	// NO-OP
}

var word8DynamicType DynamicType = NumberDynamicType{sema.Word8Type}

func (Word8Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return word8DynamicType
}

func (Word8Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord8
}

func (v Word8Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word8Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word8Value) ToInt() int {
	return int(v)
}

func (v Word8Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v + o
}

func (v Word8Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v - o
}

func (v Word8Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word8Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v * o
}

func (v Word8Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word8Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Word8Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Word8Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Word8Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Word8Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherWord8, ok := other.(Word8Value)
	if !ok {
		return false
	}
	return v == otherWord8
}

func (v Word8Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord8)
	scratch[1] = byte(v)
	return scratch[:2]
}

func ConvertWord8(value Value) Word8Value {
	return Word8Value(ConvertUInt8(value))
}

func (v Word8Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Word8Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}
	return v ^ o
}

func (v Word8Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Word8Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Word8Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word8Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Word8Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Word8Type)
}

func (Word8Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word8Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Word8Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word8Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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
var _ HashableValue = Word16Value(0)
var _ MemberAccessibleValue = Word16Value(0)

func (Word16Value) IsValue() {}

func (v Word16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord16Value(interpreter, v)
}

func (Word16Value) Walk(_ func(Value)) {
	// NO-OP
}

var word16DynamicType DynamicType = NumberDynamicType{sema.Word16Type}

func (Word16Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return word16DynamicType
}

func (Word16Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord16
}

func (v Word16Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word16Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word16Value) ToInt() int {
	return int(v)
}
func (v Word16Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v + o
}

func (v Word16Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v - o
}

func (v Word16Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word16Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v * o
}

func (v Word16Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word16Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Word16Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Word16Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Word16Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Word16Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherWord16, ok := other.(Word16Value)
	if !ok {
		return false
	}
	return v == otherWord16
}

func (v Word16Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord16)
	binary.BigEndian.PutUint16(scratch[1:], uint16(v))
	return scratch[:3]
}

func ConvertWord16(value Value) Word16Value {
	return Word16Value(ConvertUInt16(value))
}

func (v Word16Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Word16Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}
	return v ^ o
}

func (v Word16Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Word16Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Word16Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word16Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Word16Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Word16Type)
}

func (Word16Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word16Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Word16Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word16Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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
var _ HashableValue = Word32Value(0)
var _ MemberAccessibleValue = Word32Value(0)

func (Word32Value) IsValue() {}

func (v Word32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord32Value(interpreter, v)
}

func (Word32Value) Walk(_ func(Value)) {
	// NO-OP
}

var word32DynamicType DynamicType = NumberDynamicType{sema.Word32Type}

func (Word32Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return word32DynamicType
}

func (Word32Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord32
}

func (v Word32Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word32Value) ToInt() int {
	return int(v)
}

func (v Word32Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v + o
}

func (v Word32Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v - o
}

func (v Word32Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word32Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v * o
}

func (v Word32Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word32Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Word32Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Word32Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Word32Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Word32Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherWord32, ok := other.(Word32Value)
	if !ok {
		return false
	}
	return v == otherWord32
}

func (v Word32Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord32)
	binary.BigEndian.PutUint32(scratch[1:], uint32(v))
	return scratch[:5]
}

func ConvertWord32(value Value) Word32Value {
	return Word32Value(ConvertUInt32(value))
}

func (v Word32Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Word32Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}
	return v ^ o
}

func (v Word32Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Word32Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Word32Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Word32Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Word32Type)
}

func (Word32Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word32Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Word32Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word32Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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
var _ HashableValue = Word64Value(0)
var _ MemberAccessibleValue = Word64Value(0)

// NOTE: important, do *NOT* remove:
// Word64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
//
var _ BigNumberValue = Word64Value(0)

func (Word64Value) IsValue() {}

func (v Word64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord64Value(interpreter, v)
}

func (Word64Value) Walk(_ func(Value)) {
	// NO-OP
}

var word64DynamicType DynamicType = NumberDynamicType{sema.Word64Type}

func (Word64Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return word64DynamicType
}

func (Word64Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord64
}

func (v Word64Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word64Value) ToInt() int {
	if v > math.MaxInt64 {
		panic(OverflowError{})
	}
	return int(v)
}

// ToBigInt
//
// NOTE: important, do *NOT* remove:
// Word64 values > math.MaxInt64 overflow int.
// Implementing BigNumberValue ensures conversion functions
// call ToBigInt instead of ToInt.
//
func (v Word64Value) ToBigInt() *big.Int {
	return new(big.Int).SetUint64(uint64(v))
}

func (v Word64Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v + o
}

func (v Word64Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v - o
}

func (v Word64Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word64Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v * o
}

func (v Word64Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word64Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Word64Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Word64Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Word64Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Word64Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherWord64, ok := other.(Word64Value)
	if !ok {
		return false
	}
	return v == otherWord64
}

func (v Word64Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeWord64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertWord64(value Value) Word64Value {
	return Word64Value(ConvertUInt64(value))
}

func (v Word64Value) BitwiseOr(other IntegerValue) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseOr,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v | o
}

func (v Word64Value) BitwiseXor(other IntegerValue) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseXor,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}
	return v ^ o
}

func (v Word64Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseAnd,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v & o
}

func (v Word64Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseLeftShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v << o
}

func (v Word64Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationBitwiseRightShift,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >> o
}

func (v Word64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Word64Type)
}

func (Word64Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Word64Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word64Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

// Fix64Value
//
type Fix64Value int64

const Fix64MaxValue = math.MaxInt64

func NewFix64ValueWithInteger(integer int64) Fix64Value {

	if integer < sema.Fix64TypeMinInt {
		panic(UnderflowError{})
	}

	if integer > sema.Fix64TypeMaxInt {
		panic(OverflowError{})
	}

	return Fix64Value(integer * sema.Fix64Factor)
}

var _ Value = Fix64Value(0)
var _ atree.Storable = Fix64Value(0)
var _ NumberValue = Fix64Value(0)
var _ EquatableValue = Fix64Value(0)
var _ HashableValue = Fix64Value(0)
var _ MemberAccessibleValue = Fix64Value(0)

func (Fix64Value) IsValue() {}

func (v Fix64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitFix64Value(interpreter, v)
}

func (Fix64Value) Walk(_ func(Value)) {
	// NO-OP
}

var fix64DynamicType DynamicType = NumberDynamicType{sema.Fix64Type}

func (Fix64Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return fix64DynamicType
}

func (Fix64Value) StaticType() StaticType {
	return PrimitiveStaticTypeFix64
}

func (v Fix64Value) String() string {
	return format.Fix64(int64(v))
}

func (v Fix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Fix64Value) ToInt() int {
	return int(v / sema.Fix64Factor)
}

func (v Fix64Value) Negate() NumberValue {
	// INT32-C
	if v == math.MinInt64 {
		panic(OverflowError{})
	}
	return -v
}

func (v Fix64Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return Fix64Value(safeAddInt64(int64(v), int64(o)))
}

func (v Fix64Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v > (math.MaxInt64 - o)) {
		return Fix64Value(math.MaxInt64)
	} else if (o < 0) && (v < (math.MinInt64 - o)) {
		return Fix64Value(math.MinInt64)
	}
	return v + o
}

func (v Fix64Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Fix64Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		return Fix64Value(math.MinInt64)
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		return Fix64Value(math.MaxInt64)
	}
	return v - o
}

var minInt64Big = big.NewInt(math.MinInt64)
var maxInt64Big = big.NewInt(math.MaxInt64)

func (v Fix64Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	result := new(big.Int).Mul(a, b)
	result.Div(result, sema.Fix64FactorBig)

	if result.Cmp(minInt64Big) < 0 {
		panic(UnderflowError{})
	} else if result.Cmp(maxInt64Big) > 0 {
		panic(OverflowError{})
	}

	return Fix64Value(result.Int64())
}

func (v Fix64Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	result := new(big.Int).Mul(a, b)
	result.Div(result, sema.Fix64FactorBig)

	if result.Cmp(minInt64Big) < 0 {
		return Fix64Value(math.MinInt64)
	} else if result.Cmp(maxInt64Big) > 0 {
		return Fix64Value(math.MaxInt64)
	}

	return Fix64Value(result.Int64())
}

func (v Fix64Value) Div(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	result := new(big.Int).Mul(a, sema.Fix64FactorBig)
	result.Div(result, b)

	if result.Cmp(minInt64Big) < 0 {
		panic(UnderflowError{})
	} else if result.Cmp(maxInt64Big) > 0 {
		panic(OverflowError{})
	}

	return Fix64Value(result.Int64())
}

func (v Fix64Value) SaturatingDiv(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	a := new(big.Int).SetInt64(int64(v))
	b := new(big.Int).SetInt64(int64(o))

	result := new(big.Int).Mul(a, sema.Fix64FactorBig)
	result.Div(result, b)

	if result.Cmp(minInt64Big) < 0 {
		return Fix64Value(math.MinInt64)
	} else if result.Cmp(maxInt64Big) > 0 {
		return Fix64Value(math.MaxInt64)
	}

	return Fix64Value(result.Int64())
}

func (v Fix64Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// v - int(v/o) * o
	quotient := v.Div(o).(Fix64Value)
	truncatedQuotient := (int64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
	return v.Minus(Fix64Value(truncatedQuotient).Mul(o))
}

func (v Fix64Value) Less(other NumberValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v Fix64Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v Fix64Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v Fix64Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(Fix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v Fix64Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherFix64, ok := other.(Fix64Value)
	if !ok {
		return false
	}
	return v == otherFix64
}

func (v Fix64Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertFix64(value Value) Fix64Value {
	switch value := value.(type) {
	case Fix64Value:
		return value

	case UFix64Value:
		if value > Fix64MaxValue {
			panic(OverflowError{})
		}
		return Fix64Value(value)

	case BigNumberValue:
		v := value.ToBigInt()

		// First, check if the value is at least in the int64 range.
		// The integer range for Fix64 is smaller, but this test at least
		// allows us to call `v.Int64()` safely.

		if !v.IsInt64() {
			panic(OverflowError{})
		}

		// Now check that the integer value fits the range of Fix64
		return NewFix64ValueWithInteger(v.Int64())

	case NumberValue:
		v := value.ToInt()
		// Check that the integer value fits the range of Fix64
		return NewFix64ValueWithInteger(int64(v))

	default:
		panic(fmt.Sprintf("can't convert Fix64: %s", value))
	}
}

func (v Fix64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.Fix64Type)
}

func (Fix64Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Fix64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Fix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Fix64Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Fix64Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

// UFix64Value
//
type UFix64Value uint64

const UFix64MaxValue = math.MaxUint64

func NewUFix64ValueWithInteger(integer uint64) UFix64Value {
	if integer > sema.UFix64TypeMaxInt {
		panic(OverflowError{})
	}

	return UFix64Value(integer * sema.Fix64Factor)
}

var _ Value = UFix64Value(0)
var _ atree.Storable = UFix64Value(0)
var _ NumberValue = UFix64Value(0)
var _ EquatableValue = UFix64Value(0)
var _ HashableValue = UFix64Value(0)
var _ MemberAccessibleValue = UFix64Value(0)

func (UFix64Value) IsValue() {}

func (v UFix64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUFix64Value(interpreter, v)
}

func (UFix64Value) Walk(_ func(Value)) {
	// NO-OP
}

var ufix64DynamicType DynamicType = NumberDynamicType{sema.UFix64Type}

func (UFix64Value) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return ufix64DynamicType
}

func (UFix64Value) StaticType() StaticType {
	return PrimitiveStaticTypeUFix64
}

func (v UFix64Value) String() string {
	return format.UFix64(uint64(v))
}

func (v UFix64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v UFix64Value) ToInt() int {
	return int(v / sema.Fix64Factor)
}

func (v UFix64Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Plus(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationPlus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return UFix64Value(safeAddUint64(uint64(v), uint64(o)))
}

func (v UFix64Value) SaturatingPlus(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingAddFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	sum := v + o
	// INT30-C
	if sum < v {
		return UFix64Value(math.MaxUint64)
	}
	return sum
}

func (v UFix64Value) Minus(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMinus,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UFix64Value) SaturatingMinus(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingSubtractFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	diff := v - o

	// INT30-C
	if diff > v {
		return UFix64Value(0)
	}
	return diff
}

func (v UFix64Value) Mul(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMul,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	result := new(big.Int).Mul(a, b)
	result.Div(result, sema.Fix64FactorBig)

	if !result.IsUint64() {
		panic(OverflowError{})
	}

	return UFix64Value(result.Uint64())
}

func (v UFix64Value) SaturatingMul(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			FunctionName: sema.NumericTypeSaturatingMultiplyFunctionName,
			LeftType:     v.StaticType(),
			RightType:    other.StaticType(),
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	result := new(big.Int).Mul(a, b)
	result.Div(result, sema.Fix64FactorBig)

	if !result.IsUint64() {
		return UFix64Value(math.MaxUint64)
	}

	return UFix64Value(result.Uint64())
}

func (v UFix64Value) Div(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationDiv,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	result := new(big.Int).Mul(a, sema.Fix64FactorBig)
	result.Div(result, b)

	return UFix64Value(result.Uint64())
}

func (v UFix64Value) SaturatingDiv(other NumberValue) NumberValue {
	defer func() {
		r := recover()
		if _, ok := r.(InvalidOperandsError); ok {
			panic(InvalidOperandsError{
				FunctionName: sema.NumericTypeSaturatingDivideFunctionName,
				LeftType:     v.StaticType(),
				RightType:    other.StaticType(),
			})
		}
	}()

	return v.Div(other)
}

func (v UFix64Value) Mod(other NumberValue) NumberValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationMod,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	// v - int(v/o) * o
	quotient := v.Div(o).(UFix64Value)
	truncatedQuotient := (uint64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
	return v.Minus(UFix64Value(truncatedQuotient).Mul(o))
}

func (v UFix64Value) Less(other NumberValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLess,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v < o
}

func (v UFix64Value) LessEqual(other NumberValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationLessEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v <= o
}

func (v UFix64Value) Greater(other NumberValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreater,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v > o
}

func (v UFix64Value) GreaterEqual(other NumberValue) BoolValue {
	o, ok := other.(UFix64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation: ast.OperationGreaterEqual,
			LeftType:  v.StaticType(),
			RightType: other.StaticType(),
		})
	}

	return v >= o
}

func (v UFix64Value) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherUFix64, ok := other.(UFix64Value)
	if !ok {
		return false
	}
	return v == otherUFix64
}

func (v UFix64Value) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypeUFix64)
	binary.BigEndian.PutUint64(scratch[1:], uint64(v))
	return scratch[:9]
}

func ConvertUFix64(value Value) UFix64Value {
	switch value := value.(type) {
	case UFix64Value:
		return value

	case Fix64Value:
		if value < 0 {
			panic(UnderflowError{})
		}
		return UFix64Value(value)

	case BigNumberValue:
		v := value.ToBigInt()

		if v.Sign() < 0 {
			panic(UnderflowError{})
		}

		// First, check if the value is at least in the uint64 range.
		// The integer range for UFix64 is smaller, but this test at least
		// allows us to call `v.UInt64()` safely.

		if !v.IsUint64() {
			panic(OverflowError{})
		}

		// Now check that the integer value fits the range of UFix64
		return NewUFix64ValueWithInteger(v.Uint64())

	case NumberValue:
		v := value.ToInt()
		if v < 0 {
			panic(UnderflowError{})
		}
		// Check that the integer value fits the range of UFix64
		return NewUFix64ValueWithInteger(uint64(v))

	default:
		panic(fmt.Sprintf("can't convert to UFix64: %s", value))
	}
}

func (v UFix64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name, sema.UFix64Type)
}

func (UFix64Value) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (UFix64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UFix64Value) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UFix64Type.Equal(numberType.StaticType)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

// CompositeValue

type CompositeValue struct {
	dictionary          *atree.OrderedMap
	Location            common.Location
	QualifiedIdentifier string
	Kind                common.CompositeKind
	InjectedFields      map[string]Value
	ComputedFields      map[string]ComputedField
	NestedVariables     map[string]*Variable
	Functions           map[string]FunctionValue
	Destructor          FunctionValue
	Stringer            func(value *CompositeValue, seenReferences SeenReferences) string
	isDestroyed         bool
	typeID              common.TypeID
	staticType          StaticType
	dynamicType         DynamicType
}

type ComputedField func(*Interpreter, func() LocationRange) Value

type CompositeField struct {
	Name  string
	Value Value
}

func NewCompositeValue(
	interpreter *Interpreter,
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	fields []CompositeField,
	address common.Address,
) *CompositeValue {

	interpreter.ReportComputation(common.ComputationKindCreateCompositeValue, 1)

	var v *CompositeValue
	if interpreter.tracingEnabled {
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

	dictionary, err := atree.NewMap(
		interpreter.Storage,
		atree.Address(address),
		atree.NewDefaultDigesterBuilder(),
		compositeTypeInfo{
			location:            location,
			qualifiedIdentifier: qualifiedIdentifier,
			kind:                kind,
		},
	)
	if err != nil {
		panic(ExternalError{err})
	}

	v = &CompositeValue{
		dictionary:          dictionary,
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Kind:                kind,
	}

	for _, field := range fields {
		v.SetMember(
			interpreter,
			// TODO: provide proper location range
			ReturnEmptyLocationRange,
			field.Name,
			field.Value,
		)
	}

	return v
}

var _ Value = &CompositeValue{}
var _ EquatableValue = &CompositeValue{}
var _ HashableValue = &CompositeValue{}
var _ MemberAccessibleValue = &CompositeValue{}
var _ ReferenceTrackedResourceKindedValue = &CompositeValue{}

func (*CompositeValue) IsValue() {}

func (v *CompositeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitCompositeValue(interpreter, v)
	if !descend {
		return
	}

	v.ForEachField(func(_ string, value Value) {
		value.Accept(interpreter, visitor)
	})
}

// Walk iterates over all field values of the composite value.
// It does NOT walk the computed fields and functions!
//
func (v *CompositeValue) Walk(walkChild func(Value)) {
	v.ForEachField(func(_ string, value Value) {
		walkChild(value)
	})
}

func (v *CompositeValue) DynamicType(interpreter *Interpreter, _ SeenReferences) DynamicType {
	if v.dynamicType == nil {
		var staticType sema.Type
		if v.Location == nil {
			staticType = interpreter.getNativeCompositeType(v.QualifiedIdentifier)
		} else {
			staticType = interpreter.getUserCompositeType(v.Location, v.TypeID())
		}
		v.dynamicType = CompositeDynamicType{
			StaticType: staticType,
		}
	}
	return v.dynamicType
}

func (v *CompositeValue) StaticType() StaticType {
	if v.staticType == nil {
		// NOTE: Instead of using NewCompositeStaticType, which always generates the type ID,
		// use the TypeID accessor, which may return an already computed type ID
		v.staticType = CompositeStaticType{
			Location:            v.Location,
			QualifiedIdentifier: v.QualifiedIdentifier,
			TypeID:              v.TypeID(),
		}
	}
	return v.staticType
}

func (v *CompositeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *CompositeValue) Destroy(interpreter *Interpreter, getLocationRange func() LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyCompositeValue, 1)

	if interpreter.tracingEnabled {
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

	interpreter = v.getInterpreter(interpreter)

	// if composite was deserialized, dynamically link in the destructor
	if v.Destructor == nil {
		v.Destructor = interpreter.typeCodes.CompositeCodes[v.TypeID()].DestructorFunction
	}

	destructor := v.Destructor

	if destructor != nil {
		invocation := Invocation{
			Self:             v,
			Arguments:        nil,
			ArgumentTypes:    nil,
			GetLocationRange: getLocationRange,
			Interpreter:      interpreter,
		}

		destructor.invoke(invocation)
	}

	v.isDestroyed = true
}

func (v *CompositeValue) GetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value {

	if interpreter.tracingEnabled {
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

	if v.Kind == common.CompositeKindResource &&
		name == sema.ResourceOwnerFieldName {

		return v.OwnerValue(interpreter, getLocationRange)
	}

	storable, err := v.dictionary.Get(
		stringAtreeComparator,
		stringAtreeHashInput,
		stringAtreeValue(name),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); !ok {
			panic(ExternalError{err})
		}
	}
	if storable != nil {
		return StoredValue(storable, interpreter.Storage)
	}

	if v.NestedVariables != nil {
		variable, ok := v.NestedVariables[name]
		if ok {
			return variable.GetValue()
		}
	}

	interpreter = v.getInterpreter(interpreter)

	if v.ComputedFields != nil {
		if computedField, ok := v.ComputedFields[name]; ok {
			return computedField(interpreter, getLocationRange)
		}
	}

	// If the composite value was deserialized, dynamically link in the functions
	// and get injected fields

	v.InitializeFunctions(interpreter)

	if v.InjectedFields == nil && interpreter.injectedCompositeFieldsHandler != nil {
		v.InjectedFields = interpreter.injectedCompositeFieldsHandler(
			interpreter,
			v.Location,
			v.QualifiedIdentifier,
			v.Kind,
		)
	}

	if v.InjectedFields != nil {
		value, ok := v.InjectedFields[name]
		if ok {
			return value
		}
	}

	function, ok := v.Functions[name]
	if ok {
		return BoundFunctionValue{
			Self:     v,
			Function: function,
		}
	}

	return nil
}

func (v *CompositeValue) getInterpreter(interpreter *Interpreter) *Interpreter {

	// Get the correct interpreter. The program code might need to be loaded.
	// NOTE: standard library values have no location

	location := v.Location

	if location == nil || common.LocationsMatch(interpreter.Location, location) {
		return interpreter
	}

	return interpreter.EnsureLoaded(v.Location)
}

func (v *CompositeValue) InitializeFunctions(interpreter *Interpreter) {
	if v.Functions != nil {
		return
	}

	v.Functions = interpreter.typeCodes.CompositeCodes[v.TypeID()].CompositeFunctions
}

func (v *CompositeValue) OwnerValue(interpreter *Interpreter, getLocationRange func() LocationRange) OptionalValue {
	address := v.StorageID().Address

	if address == (atree.Address{}) {
		return NilValue{}
	}

	ownerAccount := interpreter.publicAccountHandler(interpreter, AddressValue(address))

	// Owner must be of `PublicAccount` type.
	interpreter.ExpectType(ownerAccount, sema.PublicAccountType, getLocationRange)

	return NewSomeValueNonCopying(ownerAccount)
}

func (v *CompositeValue) RemoveMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
) Value {

	if interpreter.tracingEnabled {
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
		stringAtreeComparator,
		stringAtreeHashInput,
		stringAtreeValue(name),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return nil
		}
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	storage := interpreter.Storage

	// Key
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	storedValue := StoredValue(existingValueStorable, storage)
	return storedValue.
		Transfer(
			interpreter,
			getLocationRange,
			atree.Address{},
			true,
			existingValueStorable,
		)
}

func (v *CompositeValue) SetMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
	value Value,
) {
	if interpreter.tracingEnabled {
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

	address := v.StorageID().Address

	value = value.Transfer(
		interpreter,
		getLocationRange,
		address,
		true,
		nil,
	)

	existingStorable, err := v.dictionary.Set(
		stringAtreeComparator,
		stringAtreeHashInput,
		stringAtreeValue(name),
		value,
	)
	if err != nil {
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	if existingStorable != nil {
		existingValue := StoredValue(existingStorable, interpreter.Storage)

		existingValue.DeepRemove(interpreter)

		interpreter.RemoveReferencedSlab(existingStorable)
	}
}

func (v *CompositeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *CompositeValue) RecursiveString(seenReferences SeenReferences) string {
	if v.Stringer != nil {
		return v.Stringer(v, seenReferences)
	}

	var fields []CompositeField

	v.ForEachField(func(name string, value Value) {
		fields = append(
			fields,
			CompositeField{
				Name:  name,
				Value: value,
			},
		)
	})

	return formatComposite(string(v.TypeID()), fields, seenReferences)
}

func formatComposite(typeId string, fields []CompositeField, seenReferences SeenReferences) string {
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
				Value: field.Value.RecursiveString(seenReferences),
			},
		)
	}

	return format.Composite(typeId, preparedFields)
}

func (v *CompositeValue) GetField(_ *Interpreter, _ func() LocationRange, name string) Value {

	storable, err := v.dictionary.Get(
		stringAtreeComparator,
		stringAtreeHashInput,
		stringAtreeValue(name),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return nil
		}
		panic(ExternalError{err})
	}

	return StoredValue(storable, v.dictionary.Storage)
}

func (v *CompositeValue) Equal(interpreter *Interpreter, getLocationRange func() LocationRange, other Value) bool {
	otherComposite, ok := other.(*CompositeValue)
	if !ok {
		return false
	}

	if !v.StaticType().Equal(otherComposite.StaticType()) ||
		v.Kind != otherComposite.Kind ||
		v.dictionary.Count() != otherComposite.dictionary.Count() {

		return false
	}

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(ExternalError{err})
	}

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(ExternalError{err})
		}
		if key == nil {
			return true
		}

		fieldName := string(key.(stringAtreeValue))

		// NOTE: Do NOT use an iterator, iteration order of fields may be different
		// (if stored in different account, as storage ID is used as hash seed)
		otherValue := otherComposite.GetField(interpreter, getLocationRange, fieldName)

		equatableValue, ok := MustConvertStoredValue(value).(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, getLocationRange, otherValue) {
			return false
		}
	}
}

func (v *CompositeValue) HashInput(interpreter *Interpreter, getLocationRange func() LocationRange, scratch []byte) []byte {
	if v.Kind == common.CompositeKindEnum {
		enumHashInput := append(
			[]byte{byte(HashInputTypeEnum)},
			v.TypeID()...,
		)

		rawValue := v.GetField(interpreter, getLocationRange, sema.EnumRawValueFieldName)
		rawValueHashInput := rawValue.(HashableValue).
			HashInput(interpreter, getLocationRange, scratch)

		return append(
			enumHashInput,
			rawValueHashInput...,
		)
	}

	panic(errors.NewUnreachableError())
}

func (v *CompositeValue) TypeID() common.TypeID {
	if v.typeID == "" {
		location := v.Location
		qualifiedIdentifier := v.QualifiedIdentifier
		if location == nil {
			return common.TypeID(qualifiedIdentifier)
		}
		v.typeID = location.TypeID(qualifiedIdentifier)
	}
	return v.typeID
}

func (v *CompositeValue) ConformsToDynamicType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	if interpreter.tracingEnabled {
		startTime := time.Now()

		owner := v.GetOwner().String()
		typeID := string(v.TypeID())
		kind := v.Kind.String()

		defer func() {
			interpreter.reportCompositeValueConformsToDynamicTypeTrace(
				owner,
				typeID,
				kind,
				time.Since(startTime),
			)
		}()
	}

	compositeDynamicType, ok := dynamicType.(CompositeDynamicType)
	if !ok {
		return false
	}

	compositeType, ok := compositeDynamicType.StaticType.(*sema.CompositeType)
	if !ok ||
		v.Kind != compositeType.Kind ||
		v.TypeID() != compositeType.ID() {

		return false
	}

	fieldsLen := int(v.dictionary.Count())
	if v.ComputedFields != nil {
		fieldsLen += len(v.ComputedFields)
	}

	if fieldsLen != len(compositeType.Fields) {
		return false
	}

	for _, fieldName := range compositeType.Fields {
		value := v.GetField(interpreter, getLocationRange, fieldName)
		if value == nil {
			if v.ComputedFields == nil {
				return false
			}

			fieldGetter, ok := v.ComputedFields[fieldName]
			if !ok {
				return false
			}

			value = fieldGetter(interpreter, getLocationRange)
		}

		member, ok := compositeType.Members.Get(fieldName)
		if !ok {
			return false
		}

		fieldDynamicType := value.DynamicType(interpreter, SeenReferences{})

		if !interpreter.IsSubType(fieldDynamicType, member.TypeAnnotation.Type) {
			return false
		}

		if !value.ConformsToDynamicType(
			interpreter,
			getLocationRange,
			fieldDynamicType,
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
		common.CompositeKindContract:
		break
	default:
		return false
	}

	// Composite value's of native/built-in types are not storable for now
	return v.Location != nil
}

func (v *CompositeValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	if !v.IsStorable() {
		return NonStorable{Value: v}, nil
	}

	return atree.StorageIDStorable(v.StorageID()), nil
}

func (v *CompositeValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageID().Address
}

func (v *CompositeValue) IsResourceKinded(_ *Interpreter) bool {
	return v.Kind == common.CompositeKindResource
}

func (v *CompositeValue) IsReferenceTrackedResourceKindedValue() {}

func (v *CompositeValue) Transfer(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {

	interpreter.ReportComputation(common.ComputationKindTransferCompositeValue, 1)

	if interpreter.tracingEnabled {
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
	currentAddress := currentStorageID.Address

	dictionary := v.dictionary

	needsStoreTo := address != currentAddress
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {
		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(ExternalError{err})
		}

		dictionary, err = atree.NewMapFromBatchData(
			interpreter.Storage,
			address,
			atree.NewDefaultDigesterBuilder(),
			v.dictionary.Type(),
			stringAtreeComparator,
			stringAtreeHashInput,
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

				value := MustConvertStoredValue(atreeValue).
					Transfer(interpreter, getLocationRange, address, remove, nil)

				return atreeKey, value, nil
			},
		)
		if err != nil {
			panic(ExternalError{err})
		}

		if remove {
			err = v.dictionary.PopIterate(func(nameStorable atree.Storable, valueStorable atree.Storable) {
				interpreter.RemoveReferencedSlab(nameStorable)
				interpreter.RemoveReferencedSlab(valueStorable)
			})
			if err != nil {
				panic(ExternalError{err})
			}
			interpreter.maybeValidateAtreeValue(v.dictionary)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		v.dictionary = dictionary

		newStorageID := dictionary.StorageID()
		interpreter.updateReferencedResource(
			currentStorageID,
			newStorageID,
			func(value ReferenceTrackedResourceKindedValue) {
				compositeValue, ok := value.(*CompositeValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				compositeValue.dictionary = dictionary
			},
		)

		return v
	} else {
		return &CompositeValue{
			dictionary:          dictionary,
			Location:            v.Location,
			QualifiedIdentifier: v.QualifiedIdentifier,
			Kind:                v.Kind,
			InjectedFields:      v.InjectedFields,
			ComputedFields:      v.ComputedFields,
			NestedVariables:     v.NestedVariables,
			Functions:           v.Functions,
			Destructor:          v.Destructor,
			Stringer:            v.Stringer,
			isDestroyed:         v.isDestroyed,
			typeID:              v.typeID,
			staticType:          v.staticType,
			dynamicType:         v.dynamicType,
		}
	}
}

func (v *CompositeValue) DeepRemove(interpreter *Interpreter) {

	if interpreter.tracingEnabled {
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

		value := StoredValue(valueStorable, storage)
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)
}

func (v *CompositeValue) GetOwner() common.Address {
	return common.Address(v.StorageID().Address)
}

// ForEachField iterates over all field-name field-value pairs of the composite value.
// It does NOT iterate over computed fields and functions!
//
func (v *CompositeValue) ForEachField(f func(fieldName string, fieldValue Value)) {
	err := v.dictionary.Iterate(func(key atree.Value, value atree.Value) (resume bool, err error) {
		f(
			string(key.(stringAtreeValue)),
			MustConvertStoredValue(value),
		)
		return true, nil
	})
	if err != nil {
		panic(ExternalError{err})
	}
}

func (v *CompositeValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *CompositeValue) RemoveField(
	interpreter *Interpreter,
	_ func() LocationRange,
	name string,
) {

	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		stringAtreeComparator,
		stringAtreeHashInput,
		stringAtreeValue(name),
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return
		}
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	storage := interpreter.Storage

	// Key

	// NOTE: key / field name is stringAtreeValue,
	// and not a Value, so no need to deep remove
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	existingValue := StoredValue(existingValueStorable, storage)
	existingValue.DeepRemove(interpreter)
	interpreter.RemoveReferencedSlab(existingValueStorable)
}

func NewEnumCaseValue(
	interpreter *Interpreter,
	enumType *sema.CompositeType,
	rawValue NumberValue,
	functions map[string]FunctionValue,
) *CompositeValue {

	fields := []CompositeField{
		{
			Name:  sema.EnumRawValueFieldName,
			Value: rawValue,
		},
	}

	v := NewCompositeValue(
		interpreter,
		enumType.Location,
		enumType.QualifiedIdentifier(),
		enumType.Kind,
		fields,
		common.Address{},
	)

	v.Functions = functions

	return v
}

// DictionaryValue

type DictionaryValue struct {
	Type             DictionaryStaticType
	semaType         *sema.DictionaryType
	isResourceKinded *bool
	dictionary       *atree.OrderedMap
	isDestroyed      bool
}

func NewDictionaryValue(
	interpreter *Interpreter,
	dictionaryType DictionaryStaticType,
	keysAndValues ...Value,
) *DictionaryValue {
	return NewDictionaryValueWithAddress(
		interpreter,
		dictionaryType,
		common.Address{},
		keysAndValues...,
	)
}

func NewDictionaryValueWithAddress(
	interpreter *Interpreter,
	dictionaryType DictionaryStaticType,
	address common.Address,
	keysAndValues ...Value,
) *DictionaryValue {

	interpreter.ReportComputation(common.ComputationKindCreateDictionaryValue, 1)

	var v *DictionaryValue

	if interpreter.tracingEnabled {
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

	dictionary, err := atree.NewMap(
		interpreter.Storage,
		atree.Address(address),
		atree.NewDefaultDigesterBuilder(),
		dictionaryType,
	)
	if err != nil {
		panic(ExternalError{err})
	}

	v = &DictionaryValue{
		Type:       dictionaryType,
		dictionary: dictionary,
	}

	for i := 0; i < keysAndValuesCount; i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		// TODO: handle existing value
		// TODO: provide proper location range
		_ = v.Insert(interpreter, ReturnEmptyLocationRange, key, value)
	}

	return v
}

var _ Value = &DictionaryValue{}
var _ atree.Value = &DictionaryValue{}
var _ EquatableValue = &DictionaryValue{}
var _ ValueIndexableValue = &DictionaryValue{}
var _ MemberAccessibleValue = &DictionaryValue{}
var _ ReferenceTrackedResourceKindedValue = &DictionaryValue{}

func (*DictionaryValue) IsValue() {}

func (v *DictionaryValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitDictionaryValue(interpreter, v)
	if !descend {
		return
	}

	v.Walk(func(value Value) {
		value.Accept(interpreter, visitor)
	})
}

func (v *DictionaryValue) Iterate(f func(key, value Value) (resume bool)) {
	err := v.dictionary.Iterate(func(key, value atree.Value) (resume bool, err error) {
		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value

		resume = f(
			MustConvertStoredValue(key),
			MustConvertStoredValue(value),
		)

		return resume, nil
	})
	if err != nil {
		panic(ExternalError{err})
	}
}

func (v *DictionaryValue) Walk(walkChild func(Value)) {
	v.Iterate(func(key, value Value) (resume bool) {
		walkChild(key)
		walkChild(value)
		return true
	})
}

func (v *DictionaryValue) DynamicType(interpreter *Interpreter, seenReferences SeenReferences) DynamicType {
	entryTypes := make([]DictionaryStaticTypeEntry, v.Count())

	index := 0
	v.Iterate(func(key, value Value) (resume bool) {
		entryTypes[index] =
			DictionaryStaticTypeEntry{
				KeyType:   key.DynamicType(interpreter, seenReferences),
				ValueType: value.DynamicType(interpreter, seenReferences),
			}
		index++
		return true
	})

	return &DictionaryDynamicType{
		EntryTypes: entryTypes,
		StaticType: v.Type,
	}
}

func (v *DictionaryValue) StaticType() StaticType {
	return v.Type
}

func (v *DictionaryValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *DictionaryValue) Destroy(interpreter *Interpreter, getLocationRange func() LocationRange) {

	interpreter.ReportComputation(common.ComputationKindDestroyDictionaryValue, 1)

	if interpreter.tracingEnabled {
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

	v.Iterate(func(key, value Value) (resume bool) {
		// Resources cannot be keys at the moment, so should theoretically not be needed
		maybeDestroy(interpreter, getLocationRange, key)
		maybeDestroy(interpreter, getLocationRange, value)
		return true
	})

	v.isDestroyed = true
}

func (v *DictionaryValue) ContainsKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	keyValue Value,
) BoolValue {

	valueComparator := newValueComparator(interpreter, getLocationRange)
	hashInputProvider := newHashInputProvider(interpreter, getLocationRange)

	_, err := v.dictionary.Get(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return false
		}
		panic(ExternalError{err})
	}
	return true
}

func (v *DictionaryValue) Get(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	keyValue Value,
) (Value, bool) {

	valueComparator := newValueComparator(interpreter, getLocationRange)
	hashInputProvider := newHashInputProvider(interpreter, getLocationRange)

	storable, err := v.dictionary.Get(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return nil, false
		}
		panic(ExternalError{err})
	}

	storage := v.dictionary.Storage
	value := StoredValue(storable, storage)
	return value, true
}

func (v *DictionaryValue) GetKey(interpreter *Interpreter, getLocationRange func() LocationRange, keyValue Value) Value {
	value, ok := v.Get(interpreter, getLocationRange, keyValue)
	if ok {
		return NewSomeValueNonCopying(value)
	}

	return NilValue{}
}

func (v *DictionaryValue) SetKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	keyValue Value,
	value Value,
) {
	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, getLocationRange)
	interpreter.checkContainerMutation(
		OptionalStaticType{
			Type: v.Type.ValueType,
		},
		value,
		getLocationRange,
	)

	switch value := value.(type) {
	case *SomeValue:
		_ = v.Insert(interpreter, getLocationRange, keyValue, value.Value)

	case NilValue:
		_ = v.Remove(interpreter, getLocationRange, keyValue)

	default:
		panic(errors.NewUnreachableError())
	}
}

func (v *DictionaryValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *DictionaryValue) RecursiveString(seenReferences SeenReferences) string {

	pairs := make([]struct {
		Key   string
		Value string
	}, v.Count())

	index := 0
	v.Iterate(func(key, value Value) (resume bool) {
		pairs[index] = struct {
			Key   string
			Value string
		}{
			Key:   key.RecursiveString(seenReferences),
			Value: value.RecursiveString(seenReferences),
		}

		index++

		return true
	})

	return format.Dictionary(pairs)
}

func (v *DictionaryValue) GetMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
) Value {

	if interpreter.tracingEnabled {
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
		return NewIntValueFromInt64(int64(v.Count()))

	case "keys":

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(ExternalError{err})
		}

		return NewArrayValueWithIterator(
			interpreter,
			VariableSizedStaticType{
				Type: v.Type.KeyType,
			},
			common.Address{},
			func() Value {

				key, err := iterator.NextKey()
				if err != nil {
					panic(ExternalError{err})
				}
				if key == nil {
					return nil
				}

				return MustConvertStoredValue(key).
					Transfer(interpreter, getLocationRange, atree.Address{}, false, nil)
			},
		)

	case "values":

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(ExternalError{err})
		}

		return NewArrayValueWithIterator(
			interpreter,
			VariableSizedStaticType{
				Type: v.Type.ValueType,
			},
			common.Address{},
			func() Value {

				value, err := iterator.NextValue()
				if err != nil {
					panic(ExternalError{err})
				}
				if value == nil {
					return nil
				}

				return MustConvertStoredValue(value).
					Transfer(interpreter, getLocationRange, atree.Address{}, false, nil)
			})

	case "remove":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				keyValue := invocation.Arguments[0]

				return v.Remove(
					invocation.Interpreter,
					invocation.GetLocationRange,
					keyValue,
				)
			},
			sema.DictionaryRemoveFunctionType(
				v.SemaType(interpreter),
			),
		)

	case "insert":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				keyValue := invocation.Arguments[0]
				newValue := invocation.Arguments[1]

				return v.Insert(
					invocation.Interpreter,
					invocation.GetLocationRange,
					keyValue,
					newValue,
				)
			},
			sema.DictionaryInsertFunctionType(
				v.SemaType(interpreter),
			),
		)

	case "containsKey":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.ContainsKey(
					invocation.Interpreter,
					invocation.GetLocationRange,
					invocation.Arguments[0],
				)
			},
			sema.DictionaryContainsKeyFunctionType(
				v.SemaType(interpreter),
			),
		)

	}

	return nil
}

func (*DictionaryValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Dictionaries have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*DictionaryValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Dictionaries have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) Count() int {
	return int(v.dictionary.Count())
}

func (v *DictionaryValue) RemoveKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
) Value {
	return v.Remove(interpreter, getLocationRange, key)
}

func (v *DictionaryValue) Remove(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	keyValue Value,
) OptionalValue {

	valueComparator := newValueComparator(interpreter, getLocationRange)
	hashInputProvider := newHashInputProvider(interpreter, getLocationRange)

	// No need to clean up storable for passed-in key value,
	// as atree never calls Storable()
	existingKeyStorable, existingValueStorable, err := v.dictionary.Remove(
		valueComparator,
		hashInputProvider,
		keyValue,
	)
	if err != nil {
		if _, ok := err.(*atree.KeyNotFoundError); ok {
			return NilValue{}
		}
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	storage := interpreter.Storage

	// Key

	existingKeyValue := StoredValue(existingKeyStorable, storage)
	existingKeyValue.DeepRemove(interpreter)
	interpreter.RemoveReferencedSlab(existingKeyStorable)

	// Value

	existingValue := StoredValue(existingValueStorable, storage).
		Transfer(
			interpreter,
			getLocationRange,
			atree.Address{},
			true,
			existingValueStorable,
		)

	return NewSomeValueNonCopying(existingValue)
}

func (v *DictionaryValue) InsertKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key, value Value,
) {
	v.SetKey(interpreter, getLocationRange, key, value)
}

func (v *DictionaryValue) Insert(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	keyValue, value Value,
) OptionalValue {

	interpreter.checkContainerMutation(v.Type.KeyType, keyValue, getLocationRange)
	interpreter.checkContainerMutation(v.Type.ValueType, value, getLocationRange)

	address := v.dictionary.Address()

	keyValue = keyValue.Transfer(
		interpreter,
		getLocationRange,
		address,
		true,
		nil,
	)

	value = value.Transfer(
		interpreter,
		getLocationRange,
		address,
		true,
		nil,
	)

	valueComparator := newValueComparator(interpreter, getLocationRange)
	hashInputProvider := newHashInputProvider(interpreter, getLocationRange)

	// atree only calls Storable() on keyValue if needed,
	// i.e., if the key is a new key
	existingValueStorable, err := v.dictionary.Set(
		valueComparator,
		hashInputProvider,
		keyValue,
		value,
	)
	if err != nil {
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)

	if existingValueStorable == nil {
		return NilValue{}
	}

	existingValue := StoredValue(existingValueStorable, interpreter.Storage).
		Transfer(
			interpreter,
			getLocationRange,
			atree.Address{},
			true,
			existingValueStorable,
		)

	return NewSomeValueNonCopying(existingValue)
}

type DictionaryEntryValues struct {
	Key   Value
	Value Value
}

func (v *DictionaryValue) ConformsToDynamicType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	count := v.Count()

	if interpreter.tracingEnabled {
		startTime := time.Now()

		typeInfo := v.Type.String()

		defer func() {
			interpreter.reportDictionaryValueConformsToDynamicTypeTrace(
				typeInfo,
				count,
				time.Since(startTime),
			)
		}()
	}

	dictionaryType, ok := dynamicType.(*DictionaryDynamicType)
	if !ok || count != len(dictionaryType.EntryTypes) {
		return false
	}

	iterator, err := v.dictionary.Iterator()
	if err != nil {
		panic(ExternalError{err})
	}

	index := 0
	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(ExternalError{err})
		}
		if key == nil {
			return true
		}

		entryType := dictionaryType.EntryTypes[index]

		// Check the key

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryKey := MustConvertStoredValue(key)
		if !entryKey.ConformsToDynamicType(
			interpreter,
			getLocationRange,
			entryType.KeyType,
			results,
		) {
			return false
		}

		// Check the value

		// atree.OrderedMap iteration provides low-level atree.Value,
		// convert to high-level interpreter.Value
		entryValue := MustConvertStoredValue(value)
		if !entryValue.ConformsToDynamicType(
			interpreter,
			getLocationRange,
			entryType.ValueType,
			results,
		) {
			return false
		}

		index++
	}
}

func (v *DictionaryValue) Equal(interpreter *Interpreter, getLocationRange func() LocationRange, other Value) bool {

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
		panic(ExternalError{err})
	}

	for {
		key, value, err := iterator.Next()
		if err != nil {
			panic(ExternalError{err})
		}
		if key == nil {
			return true
		}

		// Do NOT use an iterator, as other value may be stored in another account,
		// leading to a different iteration order, as the storage ID is used in the seed
		otherValue, otherValueExists :=
			otherDictionary.Get(
				interpreter,
				getLocationRange,
				MustConvertStoredValue(key),
			)

		if !otherValueExists {
			return false
		}

		equatableValue, ok := MustConvertStoredValue(value).(EquatableValue)
		if !ok || !equatableValue.Equal(interpreter, getLocationRange, otherValue) {
			return false
		}
	}
}

func (v *DictionaryValue) Storable(_ atree.SlabStorage, _ atree.Address, _ uint64) (atree.Storable, error) {
	return atree.StorageIDStorable(v.StorageID()), nil
}

func (v *DictionaryValue) IsReferenceTrackedResourceKindedValue() {}

func (v *DictionaryValue) Transfer(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {

	interpreter.ReportComputation(common.ComputationKindTransferDictionaryValue, uint(v.Count()))

	if interpreter.tracingEnabled {
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
	currentAddress := currentStorageID.Address

	dictionary := v.dictionary

	needsStoreTo := address != currentAddress
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		valueComparator := newValueComparator(interpreter, getLocationRange)
		hashInputProvider := newHashInputProvider(interpreter, getLocationRange)

		iterator, err := v.dictionary.Iterator()
		if err != nil {
			panic(ExternalError{err})
		}

		dictionary, err = atree.NewMapFromBatchData(
			interpreter.Storage,
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

				key := MustConvertStoredValue(atreeKey).
					Transfer(interpreter, getLocationRange, address, remove, nil)

				value := MustConvertStoredValue(atreeValue).
					Transfer(interpreter, getLocationRange, address, remove, nil)

				return key, value, nil
			},
		)
		if err != nil {
			panic(ExternalError{err})
		}

		if remove {
			err = v.dictionary.PopIterate(func(keyStorable atree.Storable, valueStorable atree.Storable) {
				interpreter.RemoveReferencedSlab(keyStorable)
				interpreter.RemoveReferencedSlab(valueStorable)
			})
			if err != nil {
				panic(ExternalError{err})
			}
			interpreter.maybeValidateAtreeValue(v.dictionary)

			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		// Update the resource in-place,
		// and also update all values that are referencing the same value
		// (but currently point to an outdated Go instance of the value)

		v.dictionary = dictionary

		newStorageID := dictionary.StorageID()

		interpreter.updateReferencedResource(
			currentStorageID,
			newStorageID,
			func(value ReferenceTrackedResourceKindedValue) {
				dictionaryValue, ok := value.(*DictionaryValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				dictionaryValue.dictionary = dictionary
			},
		)

		return v
	} else {
		return &DictionaryValue{
			Type:             v.Type,
			semaType:         v.semaType,
			isResourceKinded: v.isResourceKinded,
			dictionary:       dictionary,
			isDestroyed:      v.isDestroyed,
		}
	}
}

func (v *DictionaryValue) DeepRemove(interpreter *Interpreter) {

	if interpreter.tracingEnabled {
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

		key := StoredValue(keyStorable, storage)
		key.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(keyStorable)

		value := StoredValue(valueStorable, storage)
		value.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(valueStorable)
	})
	if err != nil {
		panic(ExternalError{err})
	}
	interpreter.maybeValidateAtreeValue(v.dictionary)
}

func (v *DictionaryValue) GetOwner() common.Address {
	return common.Address(v.StorageID().Address)
}

func (v *DictionaryValue) StorageID() atree.StorageID {
	return v.dictionary.StorageID()
}

func (v *DictionaryValue) SemaType(interpreter *Interpreter) *sema.DictionaryType {
	if v.semaType == nil {
		v.semaType = interpreter.ConvertStaticToSemaType(v.Type).(*sema.DictionaryType)
	}
	return v.semaType
}

func (v *DictionaryValue) NeedsStoreTo(address atree.Address) bool {
	return address != v.StorageID().Address
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
}

// NilValue

type NilValue struct{}

var _ Value = NilValue{}
var _ atree.Storable = NilValue{}
var _ EquatableValue = NilValue{}
var _ MemberAccessibleValue = NilValue{}

func (NilValue) IsValue() {}

func (v NilValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitNilValue(interpreter, v)
}

func (NilValue) Walk(_ func(Value)) {
	// NO-OP
}

var nilDynamicType DynamicType = NilDynamicType{}

func (NilValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return nilDynamicType
}

func (NilValue) StaticType() StaticType {
	return OptionalStaticType{
		Type: PrimitiveStaticTypeNever,
	}
}

func (NilValue) isOptionalValue() {}

func (NilValue) IsDestroyed() bool {
	return false
}

func (v NilValue) Destroy(_ *Interpreter, _ func() LocationRange) {
	// NO-OP
}

func (NilValue) String() string {
	return format.Nil
}

func (v NilValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

var nilValueMapFunction = NewHostFunctionValue(
	func(invocation Invocation) Value {
		return NilValue{}
	},
	&sema.FunctionType{
		ReturnTypeAnnotation: sema.NewTypeAnnotation(
			sema.NeverType,
		),
	},
)

func (v NilValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "map":
		return nilValueMapFunction
	}

	return nil
}

func (NilValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Nil has no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (NilValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Nil has no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v NilValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	_, ok := dynamicType.(NilDynamicType)
	return ok
}

func (v NilValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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
	Value         Value
	valueStorable atree.Storable
	// TODO: Store isDestroyed in SomeStorable?
	isDestroyed bool
}

func NewSomeValueNonCopying(value Value) *SomeValue {
	return &SomeValue{
		Value: value,
	}
}

var _ Value = &SomeValue{}
var _ EquatableValue = &SomeValue{}
var _ MemberAccessibleValue = &SomeValue{}

func (*SomeValue) IsValue() {}

func (v *SomeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitSomeValue(interpreter, v)
	if !descend {
		return
	}
	v.Value.Accept(interpreter, visitor)
}

func (v *SomeValue) Walk(walkChild func(Value)) {
	walkChild(v.Value)
}

func (v *SomeValue) DynamicType(interpreter *Interpreter, seenReferences SeenReferences) DynamicType {
	innerType := v.Value.DynamicType(interpreter, seenReferences)
	return SomeDynamicType{InnerType: innerType}
}

func (v *SomeValue) StaticType() StaticType {
	innerType := v.Value.StaticType()
	if innerType == nil {
		return nil
	}
	return OptionalStaticType{
		Type: innerType,
	}
}

func (*SomeValue) isOptionalValue() {}

func (v *SomeValue) IsDestroyed() bool {
	return v.isDestroyed
}

func (v *SomeValue) Destroy(interpreter *Interpreter, getLocationRange func() LocationRange) {
	maybeDestroy(interpreter, getLocationRange, v.Value)
	v.isDestroyed = true
}

func (v *SomeValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *SomeValue) RecursiveString(seenReferences SeenReferences) string {
	return v.Value.RecursiveString(seenReferences)
}

func (v *SomeValue) GetMember(inter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "map":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {

				transformFunction := invocation.Arguments[0].(FunctionValue)
				transformFunctionType := invocation.ArgumentTypes[0].(*sema.FunctionType)
				valueType := transformFunctionType.Parameters[0].TypeAnnotation.Type

				transformInvocation := Invocation{
					Arguments:        []Value{v.Value},
					ArgumentTypes:    []sema.Type{valueType},
					GetLocationRange: invocation.GetLocationRange,
					Interpreter:      invocation.Interpreter,
				}

				newValue := transformFunction.invoke(transformInvocation)

				return NewSomeValueNonCopying(newValue)
			},
			sema.OptionalTypeMapFunctionType(inter.ConvertStaticToSemaType(v.Value.StaticType())),
		)
	}

	return nil
}

func (*SomeValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	panic(errors.NewUnreachableError())
}

func (*SomeValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v SomeValue) ConformsToDynamicType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {
	someType, ok := dynamicType.(SomeDynamicType)
	return ok && v.Value.ConformsToDynamicType(
		interpreter,
		getLocationRange,
		someType.InnerType,
		results,
	)
}

func (v *SomeValue) Equal(interpreter *Interpreter, getLocationRange func() LocationRange, other Value) bool {
	otherSome, ok := other.(*SomeValue)
	if !ok {
		return false
	}

	equatableValue, ok := v.Value.(EquatableValue)
	if !ok {
		return false
	}

	return equatableValue.Equal(interpreter, getLocationRange, otherSome.Value)
}

func (v *SomeValue) Storable(
	storage atree.SlabStorage,
	address atree.Address,
	maxInlineSize uint64,
) (atree.Storable, error) {

	if v.valueStorable == nil {
		var err error
		v.valueStorable, err = v.Value.Storable(
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
	return v.Value.NeedsStoreTo(address)
}

func (v *SomeValue) IsResourceKinded(interpreter *Interpreter) bool {
	return v.Value.IsResourceKinded(interpreter)
}

func (v *SomeValue) Transfer(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	address atree.Address,
	remove bool,
	storable atree.Storable,
) Value {

	innerValue := v.Value

	needsStoreTo := v.NeedsStoreTo(address)
	isResourceKinded := v.IsResourceKinded(interpreter)

	if needsStoreTo || !isResourceKinded {

		innerValue = v.Value.Transfer(interpreter, getLocationRange, address, remove, nil)

		if remove {
			interpreter.RemoveReferencedSlab(v.valueStorable)
			interpreter.RemoveReferencedSlab(storable)
		}
	}

	if isResourceKinded {
		v.Value = innerValue
		v.valueStorable = nil
		return v
	} else {
		result := NewSomeValueNonCopying(innerValue)
		result.valueStorable = nil
		result.isDestroyed = v.isDestroyed
		return result
	}
}

func (v *SomeValue) DeepRemove(interpreter *Interpreter) {
	v.Value.DeepRemove(interpreter)
	if v.valueStorable != nil {
		interpreter.RemoveReferencedSlab(v.valueStorable)
	}
}

type SomeStorable struct {
	Storable atree.Storable
}

var _ atree.Storable = SomeStorable{}

func (s SomeStorable) ByteSize() uint32 {
	return cborTagSize + s.Storable.ByteSize()
}

func (s SomeStorable) StoredValue(storage atree.SlabStorage) (atree.Value, error) {
	value := StoredValue(s.Storable, storage)

	return &SomeValue{
		Value:         value,
		valueStorable: s.Storable,
	}, nil
}

func (s SomeStorable) ChildStorables() []atree.Storable {
	return []atree.Storable{
		s.Storable,
	}
}

// StorageReferenceValue

type StorageReferenceValue struct {
	Authorized           bool
	TargetStorageAddress common.Address
	TargetPath           PathValue
	BorrowedType         sema.Type
}

var _ Value = &StorageReferenceValue{}
var _ EquatableValue = &StorageReferenceValue{}
var _ ValueIndexableValue = &StorageReferenceValue{}
var _ MemberAccessibleValue = &StorageReferenceValue{}

func (*StorageReferenceValue) IsValue() {}

func (v *StorageReferenceValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitStorageReferenceValue(interpreter, v)
}

func (*StorageReferenceValue) Walk(_ func(Value)) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (*StorageReferenceValue) String() string {
	return "StorageReference()"
}

func (v *StorageReferenceValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *StorageReferenceValue) DynamicType(interpreter *Interpreter, seenReferences SeenReferences) DynamicType {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{})
	}

	innerType := (*referencedValue).DynamicType(interpreter, seenReferences)

	return StorageReferenceDynamicType{
		authorized:   v.Authorized,
		innerType:    innerType,
		borrowedType: v.BorrowedType,
	}
}

func (v *StorageReferenceValue) StaticType() StaticType {
	var borrowedType StaticType
	if v.BorrowedType != nil {
		borrowedType = ConvertSemaToStaticType(v.BorrowedType)
	}
	return ReferenceStaticType{
		Authorized: v.Authorized,
		Type:       borrowedType,
	}
}

func (v *StorageReferenceValue) ReferencedValue(interpreter *Interpreter) *Value {

	address := v.TargetStorageAddress
	domain := v.TargetPath.Domain.Identifier()
	identifier := v.TargetPath.Identifier

	referenced := interpreter.ReadStored(address, domain, identifier)
	if referenced == nil {
		return nil
	}

	if v.BorrowedType != nil {
		dynamicType := referenced.DynamicType(interpreter, SeenReferences{})
		if !interpreter.IsSubType(dynamicType, v.BorrowedType) {
			interpreter.IsSubType(dynamicType, v.BorrowedType)
			return nil
		}
	}

	return &referenced
}

func (v *StorageReferenceValue) GetMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
) Value {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	return interpreter.getMember(self, getLocationRange, name)
}

func (v *StorageReferenceValue) RemoveMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
) Value {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	return self.(MemberAccessibleValue).RemoveMember(interpreter, getLocationRange, name)
}

func (v *StorageReferenceValue) SetMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
	value Value,
) {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	interpreter.setMember(self, getLocationRange, name, value)
}

func (v *StorageReferenceValue) GetKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
) Value {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	return self.(ValueIndexableValue).
		GetKey(interpreter, getLocationRange, key)
}

func (v *StorageReferenceValue) SetKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
	value Value,
) {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	self.(ValueIndexableValue).
		SetKey(interpreter, getLocationRange, key, value)
}

func (v *StorageReferenceValue) InsertKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
	value Value,
) {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	self.(ValueIndexableValue).
		InsertKey(interpreter, getLocationRange, key, value)
}

func (v *StorageReferenceValue) RemoveKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
) Value {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	return self.(ValueIndexableValue).
		RemoveKey(interpreter, getLocationRange, key)
}

func (v *StorageReferenceValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
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

func (v *StorageReferenceValue) ConformsToDynamicType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	refType, ok := dynamicType.(StorageReferenceDynamicType)
	if !ok ||
		refType.authorized != v.Authorized {

		return false
	}

	if refType.borrowedType == nil {
		if v.BorrowedType != nil {
			return false
		}
	} else if !refType.borrowedType.Equal(v.BorrowedType) {
		return false
	}

	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		return false
	}

	return (*referencedValue).ConformsToDynamicType(
		interpreter,
		getLocationRange,
		refType.InnerType(),
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *StorageReferenceValue) Clone(_ *Interpreter) Value {
	return &StorageReferenceValue{
		Authorized:           v.Authorized,
		TargetStorageAddress: v.TargetStorageAddress,
		TargetPath:           v.TargetPath,
		BorrowedType:         v.BorrowedType,
	}
}

func (*StorageReferenceValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

// EphemeralReferenceValue

type EphemeralReferenceValue struct {
	Authorized   bool
	Value        Value
	BorrowedType sema.Type
}

var _ Value = &EphemeralReferenceValue{}
var _ EquatableValue = &EphemeralReferenceValue{}
var _ ValueIndexableValue = &EphemeralReferenceValue{}
var _ MemberAccessibleValue = &EphemeralReferenceValue{}

func (*EphemeralReferenceValue) IsValue() {}

func (v *EphemeralReferenceValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitEphemeralReferenceValue(interpreter, v)
}

func (*EphemeralReferenceValue) Walk(_ func(Value)) {
	// NO-OP
	// NOTE: *not* walking referenced value!
}

func (v *EphemeralReferenceValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *EphemeralReferenceValue) RecursiveString(seenReferences SeenReferences) string {
	if _, ok := seenReferences[v]; ok {
		return "..."
	}
	seenReferences[v] = struct{}{}
	defer delete(seenReferences, v)

	return v.Value.RecursiveString(seenReferences)
}

func (v *EphemeralReferenceValue) DynamicType(interpreter *Interpreter, seenReferences SeenReferences) DynamicType {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{})
	}

	if _, ok := seenReferences[v]; ok {
		return nil
	}

	seenReferences[v] = struct{}{}
	defer delete(seenReferences, v)

	innerType := (*referencedValue).DynamicType(interpreter, seenReferences)

	return EphemeralReferenceDynamicType{
		authorized:   v.Authorized,
		innerType:    innerType,
		borrowedType: v.BorrowedType,
	}
}

func (v *EphemeralReferenceValue) StaticType() StaticType {
	var borrowedType StaticType
	if v.BorrowedType != nil {
		borrowedType = ConvertSemaToStaticType(v.BorrowedType)
	}
	return ReferenceStaticType{
		Authorized: v.Authorized,
		Type:       borrowedType,
	}
}

func (v *EphemeralReferenceValue) ReferencedValue() *Value {
	// Just like for storage references, references to optionals are unwrapped,
	// i.e. a reference to `nil` aborts when dereferenced.

	switch referenced := v.Value.(type) {
	case *SomeValue:
		return &referenced.Value
	case NilValue:
		return nil
	default:
		return &v.Value
	}
}

func (v *EphemeralReferenceValue) GetMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
) Value {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	return interpreter.getMember(self, getLocationRange, name)
}

func (v *EphemeralReferenceValue) RemoveMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	identifier string,
) Value {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	if memberAccessibleValue, ok := self.(MemberAccessibleValue); ok {
		return memberAccessibleValue.RemoveMember(interpreter, getLocationRange, identifier)
	}

	return nil
}

func (v *EphemeralReferenceValue) SetMember(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	name string,
	value Value,
) {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	interpreter.setMember(self, getLocationRange, name, value)
}

func (v *EphemeralReferenceValue) GetKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
) Value {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	return self.(ValueIndexableValue).
		GetKey(interpreter, getLocationRange, key)
}

func (v *EphemeralReferenceValue) SetKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
	value Value,
) {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	self.(ValueIndexableValue).
		SetKey(interpreter, getLocationRange, key, value)
}

func (v *EphemeralReferenceValue) InsertKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
	value Value,
) {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	self.(ValueIndexableValue).
		InsertKey(interpreter, getLocationRange, key, value)
}

func (v *EphemeralReferenceValue) RemoveKey(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	key Value,
) Value {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	self := *referencedValue

	interpreter.checkResourceNotDestroyed(self, getLocationRange)

	return self.(ValueIndexableValue).
		RemoveKey(interpreter, getLocationRange, key)
}

func (v *EphemeralReferenceValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherReference, ok := other.(*EphemeralReferenceValue)
	if !ok ||
		v.Value != otherReference.Value ||
		v.Authorized != otherReference.Authorized {

		return false
	}

	if v.BorrowedType == nil {
		return otherReference.BorrowedType == nil
	} else {
		return v.BorrowedType.Equal(otherReference.BorrowedType)
	}
}

func (v *EphemeralReferenceValue) ConformsToDynamicType(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	refType, ok := dynamicType.(EphemeralReferenceDynamicType)
	if !ok ||
		refType.authorized != v.Authorized {

		return false
	}

	if refType.borrowedType == nil {
		if v.BorrowedType != nil {
			return false
		}
	} else if !refType.borrowedType.Equal(v.BorrowedType) {
		return false
	}

	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		return false
	}

	entry := typeConformanceResultEntry{
		EphemeralReferenceValue:       v,
		EphemeralReferenceDynamicType: refType,
	}

	if result, contains := results[entry]; contains {
		return result
	}

	// It is safe to set 'true' here even this is not checked yet, because the final result
	// doesn't depend on this. It depends on the rest of values of the object tree.
	results[entry] = true

	result := (*referencedValue).ConformsToDynamicType(
		interpreter,
		getLocationRange,
		refType.InnerType(),
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (*EphemeralReferenceValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

// AddressValue
//
type AddressValue common.Address

func NewAddressValue(a common.Address) AddressValue {
	return NewAddressValueFromBytes(a[:])
}

func NewAddressValueFromBytes(b []byte) AddressValue {
	result := AddressValue{}
	copy(result[common.AddressLength-len(b):], b)
	return result
}

func ConvertAddress(value Value) AddressValue {
	var result AddressValue

	uint64Value := ConvertUInt64(value)

	binary.BigEndian.PutUint64(
		result[:common.AddressLength],
		uint64(uint64Value),
	)

	return result
}

var _ Value = AddressValue{}
var _ atree.Storable = AddressValue{}
var _ EquatableValue = AddressValue{}
var _ HashableValue = AddressValue{}
var _ MemberAccessibleValue = AddressValue{}

func (AddressValue) IsValue() {}

func (v AddressValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitAddressValue(interpreter, v)
}

func (AddressValue) Walk(_ func(Value)) {
	// NO-OP
}

var addressDynamicType DynamicType = AddressDynamicType{}

func (AddressValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return addressDynamicType
}

func (AddressValue) StaticType() StaticType {
	return PrimitiveStaticTypeAddress
}

func (v AddressValue) String() string {
	return format.Address(common.Address(v))
}

func (v AddressValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v AddressValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherAddress, ok := other.(AddressValue)
	if !ok {
		return false
	}
	return v == otherAddress
}

func (v AddressValue) HashInput(_ *Interpreter, _ func() LocationRange, _ []byte) []byte {
	return append([]byte{byte(HashInputTypeAddress)}, v[:]...)
}

func (v AddressValue) Hex() string {
	return v.ToAddress().Hex()
}

func (v AddressValue) ToAddress() common.Address {
	return common.Address(v)
}

func (v AddressValue) GetMember(interpreter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return NewStringValue(v.String())
			},
			sema.ToStringFunctionType,
		)

	case sema.AddressTypeToBytesFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				address := common.Address(v)
				return ByteSliceToByteArrayValue(interpreter, address[:])
			},
			sema.AddressTypeToBytesFunctionType,
		)
	}

	return nil
}

func (AddressValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Addresses have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (AddressValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Addresses have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v AddressValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	_, ok := dynamicType.(AddressDynamicType)
	return ok
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
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

func accountGetCapabilityFunction(
	addressValue AddressValue,
	pathType sema.Type,
	funcType *sema.FunctionType,
) *HostFunctionValue {

	return NewHostFunctionValue(
		func(invocation Invocation) Value {

			path := invocation.Arguments[0].(PathValue)
			pathDynamicType := path.DynamicType(invocation.Interpreter, SeenReferences{})
			if !invocation.Interpreter.IsSubType(pathDynamicType, pathType) {
				panic(TypeMismatchError{
					ExpectedType:  pathType,
					LocationRange: invocation.GetLocationRange(),
				})
			}

			// NOTE: the type parameter is optional, for backwards compatibility

			var borrowType *sema.ReferenceType
			typeParameterPair := invocation.TypeParameterTypes.Oldest()
			if typeParameterPair != nil {
				ty := typeParameterPair.Value
				borrowType = ty.(*sema.ReferenceType)
			}

			var borrowStaticType StaticType
			if borrowType != nil {
				borrowStaticType = ConvertSemaToStaticType(borrowType)
			}

			return &CapabilityValue{
				Address:    addressValue,
				Path:       path,
				BorrowType: borrowStaticType,
			}
		},
		funcType,
	)
}

// PathValue

type PathValue struct {
	Domain     common.PathDomain
	Identifier string
}

var EmptyPathValue = PathValue{}

var _ Value = PathValue{}
var _ atree.Storable = PathValue{}
var _ EquatableValue = PathValue{}
var _ HashableValue = PathValue{}
var _ MemberAccessibleValue = PathValue{}

func (PathValue) IsValue() {}

func (v PathValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitPathValue(interpreter, v)
}

func (PathValue) Walk(_ func(Value)) {
	// NO-OP
}

var storagePathDynamicType DynamicType = StoragePathDynamicType{}
var publicPathDynamicType DynamicType = PublicPathDynamicType{}
var privatePathDynamicType DynamicType = PrivatePathDynamicType{}

func (v PathValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	switch v.Domain {
	case common.PathDomainStorage:
		return storagePathDynamicType
	case common.PathDomainPublic:
		return publicPathDynamicType
	case common.PathDomainPrivate:
		return privatePathDynamicType
	default:
		panic(errors.NewUnreachableError())
	}
}

func (v PathValue) StaticType() StaticType {
	switch v.Domain {
	case common.PathDomainStorage:
		return PrimitiveStaticTypeStoragePath
	case common.PathDomainPublic:
		return PrimitiveStaticTypePublicPath
	case common.PathDomainPrivate:
		return PrimitiveStaticTypePrivatePath
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

func (v PathValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return NewStringValue(v.String())
			},
			sema.ToStringFunctionType,
		)
	}

	return nil
}

func (PathValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Paths have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (PathValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Paths have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v PathValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	switch dynamicType.(type) {
	case PublicPathDynamicType:
		return v.Domain == common.PathDomainPublic
	case PrivatePathDynamicType:
		return v.Domain == common.PathDomainPrivate
	case StoragePathDynamicType:
		return v.Domain == common.PathDomainStorage
	default:
		return false
	}
}

func (v PathValue) Equal(_ *Interpreter, _ func() LocationRange, other Value) bool {
	otherPath, ok := other.(PathValue)
	if !ok {
		return false
	}

	return otherPath.Identifier == v.Identifier &&
		otherPath.Domain == v.Domain
}

func (v PathValue) HashInput(_ *Interpreter, _ func() LocationRange, scratch []byte) []byte {
	scratch[0] = byte(HashInputTypePath)
	scratch[1] = byte(v.Domain)
	return append(scratch[:2], []byte(v.Identifier)...)
}

func (PathValue) IsStorable() bool {
	return true
}

func convertPath(domain common.PathDomain, value Value) Value {
	stringValue, ok := value.(*StringValue)
	if !ok {
		return NilValue{}
	}

	_, err := sema.CheckPathLiteral(
		domain.Identifier(),
		stringValue.Str,
		ReturnEmptyRange,
		ReturnEmptyRange,
	)
	if err != nil {
		return NilValue{}
	}

	return NewSomeValueNonCopying(PathValue{
		Domain:     domain,
		Identifier: stringValue.Str,
	})
}

func ConvertPublicPath(value Value) Value {
	return convertPath(common.PathDomainPublic, value)
}

func ConvertPrivatePath(value Value) Value {
	return convertPath(common.PathDomainPrivate, value)
}

func ConvertStoragePath(value Value) Value {
	return convertPath(common.PathDomainStorage, value)
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
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (PathValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v PathValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v PathValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (PathValue) ChildStorables() []atree.Storable {
	return nil
}

// CapabilityValue

type CapabilityValue struct {
	Address    AddressValue
	Path       PathValue
	BorrowType StaticType
}

var _ Value = &CapabilityValue{}
var _ atree.Storable = &CapabilityValue{}
var _ EquatableValue = &CapabilityValue{}
var _ MemberAccessibleValue = &CapabilityValue{}

func (*CapabilityValue) IsValue() {}

func (v *CapabilityValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitCapabilityValue(interpreter, v)
}

func (v *CapabilityValue) Walk(walkChild func(Value)) {
	walkChild(v.Address)
	walkChild(v.Path)
}

func (v *CapabilityValue) DynamicType(interpreter *Interpreter, _ SeenReferences) DynamicType {
	var borrowType *sema.ReferenceType
	if v.BorrowType != nil {
		borrowType = interpreter.ConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
	}

	return CapabilityDynamicType{
		BorrowType: borrowType,
	}
}

func (v *CapabilityValue) StaticType() StaticType {
	return CapabilityStaticType{
		BorrowType: v.BorrowType,
	}
}

func (v *CapabilityValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v *CapabilityValue) RecursiveString(seenReferences SeenReferences) string {
	var borrowType string
	if v.BorrowType != nil {
		borrowType = v.BorrowType.String()
	}
	return format.Capability(
		borrowType,
		v.Address.RecursiveString(seenReferences),
		v.Path.RecursiveString(seenReferences),
	)
}

func (v *CapabilityValue) GetMember(interpreter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case sema.CapabilityTypeBorrowField:
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			borrowType = interpreter.ConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return interpreter.capabilityBorrowFunction(v.Address, v.Path, borrowType)

	case sema.CapabilityTypeCheckField:
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			borrowType = interpreter.ConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return interpreter.capabilityCheckFunction(v.Address, v.Path, borrowType)

	case sema.CapabilityTypeAddressField:
		return v.Address
	}

	return nil
}

func (*CapabilityValue) RemoveMember(_ *Interpreter, _ func() LocationRange, _ string) Value {
	// Capabilities have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (*CapabilityValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Capabilities have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *CapabilityValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	dynamicType DynamicType,
	_ TypeConformanceResults,
) bool {
	_, ok := dynamicType.(CapabilityDynamicType)
	return ok
}

func (v *CapabilityValue) Equal(interpreter *Interpreter, getLocationRange func() LocationRange, other Value) bool {
	otherCapability, ok := other.(*CapabilityValue)
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

	return otherCapability.Address.Equal(interpreter, getLocationRange, v.Address) &&
		otherCapability.Path.Equal(interpreter, getLocationRange, v.Path)
}

func (*CapabilityValue) IsStorable() bool {
	return true
}

func (v *CapabilityValue) Storable(
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

func (*CapabilityValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*CapabilityValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v *CapabilityValue) Transfer(
	interpreter *Interpreter,
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		v.DeepRemove(interpreter)
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (v *CapabilityValue) DeepRemove(interpreter *Interpreter) {
	v.Address.DeepRemove(interpreter)
	v.Path.DeepRemove(interpreter)
}

func (v *CapabilityValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v *CapabilityValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v *CapabilityValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.Address,
		v.Path,
	}
}

// LinkValue

type LinkValue struct {
	TargetPath PathValue
	Type       StaticType
}

var _ Value = LinkValue{}
var _ atree.Value = LinkValue{}
var _ EquatableValue = LinkValue{}

func (LinkValue) IsValue() {}

func (v LinkValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitLinkValue(interpreter, v)
}

func (v LinkValue) Walk(walkChild func(Value)) {
	walkChild(v.TargetPath)
}

func (LinkValue) DynamicType(_ *Interpreter, _ SeenReferences) DynamicType {
	return nil
}

func (LinkValue) StaticType() StaticType {
	return nil
}

func (v LinkValue) String() string {
	return v.RecursiveString(SeenReferences{})
}

func (v LinkValue) RecursiveString(seenReferences SeenReferences) string {
	return format.Link(
		v.Type.String(),
		v.TargetPath.RecursiveString(seenReferences),
	)
}

func (v LinkValue) ConformsToDynamicType(
	_ *Interpreter,
	_ func() LocationRange,
	_ DynamicType,
	_ TypeConformanceResults,
) bool {
	// There is no dynamic type for links,
	// as they are not first-class values in programs,
	// but only stored
	return false
}

func (v LinkValue) Equal(interpreter *Interpreter, getLocationRange func() LocationRange, other Value) bool {
	otherLink, ok := other.(LinkValue)
	if !ok {
		return false
	}

	return otherLink.TargetPath.Equal(interpreter, getLocationRange, v.TargetPath) &&
		otherLink.Type.Equal(v.Type)
}

func (LinkValue) IsStorable() bool {
	return true
}

func (v LinkValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return maybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (LinkValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (LinkValue) IsResourceKinded(_ *Interpreter) bool {
	return false
}

func (v LinkValue) Transfer(
	interpreter *Interpreter,
	_ func() LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
) Value {
	if remove {
		interpreter.RemoveReferencedSlab(storable)
	}
	return v
}

func (LinkValue) DeepRemove(_ *Interpreter) {
	// NO-OP
}

func (v LinkValue) ByteSize() uint32 {
	return mustStorableSize(v)
}

func (v LinkValue) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (v LinkValue) ChildStorables() []atree.Storable {
	return []atree.Storable{
		v.TargetPath,
	}
}

// NewPublicKeyValue constructs a PublicKey value.
func NewPublicKeyValue(
	interpreter *Interpreter,
	getLocationRange func() LocationRange,
	publicKey *ArrayValue,
	signAlgo *CompositeValue,
	validatePublicKey PublicKeyValidationHandlerFunc,
) *CompositeValue {

	fields := []CompositeField{
		{
			Name:  sema.PublicKeySignAlgoField,
			Value: signAlgo,
		},
	}

	publicKeyValue := NewCompositeValue(
		interpreter,
		sema.PublicKeyType.Location,
		sema.PublicKeyType.QualifiedIdentifier(),
		sema.PublicKeyType.Kind,
		fields,
		common.Address{},
	)

	publicKeyValue.ComputedFields = map[string]ComputedField{
		sema.PublicKeyPublicKeyField: func(interpreter *Interpreter, getLocationRange func() LocationRange) Value {
			return publicKey.Transfer(interpreter, getLocationRange, atree.Address{}, false, nil)
		},
	}
	publicKeyValue.Functions = map[string]FunctionValue{
		sema.PublicKeyVerifyFunction:    publicKeyVerifyFunction,
		sema.PublicKeyVerifyPoPFunction: publicKeyVerifyPoPFunction,
	}

	err := validatePublicKey(interpreter, getLocationRange, publicKeyValue)
	if err != nil {
		panic(InvalidPublicKeyError{
			PublicKey:     publicKey,
			Err:           err,
			LocationRange: getLocationRange(),
		})
	}

	// Public key value to string should include the key even though it is a computed field
	publicKeyValue.Stringer = func(publicKeyValue *CompositeValue, seenReferences SeenReferences) string {
		stringerFields := []CompositeField{
			{
				Name:  sema.PublicKeyPublicKeyField,
				Value: publicKey,
			},
			{
				Name: sema.PublicKeySignAlgoField,
				// TODO: provide proper location range
				Value: publicKeyValue.GetField(interpreter, ReturnEmptyLocationRange, sema.PublicKeySignAlgoField),
			},
		}

		return formatComposite(
			string(publicKeyValue.TypeID()),
			stringerFields,
			seenReferences,
		)
	}

	return publicKeyValue
}

var publicKeyVerifyFunction = NewHostFunctionValue(
	func(invocation Invocation) Value {
		signatureValue := invocation.Arguments[0].(*ArrayValue)
		signedDataValue := invocation.Arguments[1].(*ArrayValue)
		domainSeparationTag := invocation.Arguments[2].(*StringValue)
		hashAlgo := invocation.Arguments[3].(*CompositeValue)
		publicKey := invocation.Self

		interpreter := invocation.Interpreter

		getLocationRange := invocation.GetLocationRange

		interpreter.ExpectType(
			publicKey,
			sema.PublicKeyType,
			getLocationRange,
		)

		return interpreter.SignatureVerificationHandler(
			interpreter,
			getLocationRange,
			signatureValue,
			signedDataValue,
			domainSeparationTag,
			hashAlgo,
			publicKey,
		)
	},
	sema.PublicKeyVerifyFunctionType,
)

var publicKeyVerifyPoPFunction = NewHostFunctionValue(
	func(invocation Invocation) (v Value) {
		signatureValue := invocation.Arguments[0].(*ArrayValue)
		publicKey := invocation.Self

		interpreter := invocation.Interpreter

		getLocationRange := invocation.GetLocationRange

		interpreter.ExpectType(
			publicKey,
			sema.PublicKeyType,
			getLocationRange,
		)

		return interpreter.BLSVerifyPoPHandler(
			interpreter,
			getLocationRange,
			publicKey,
			signatureValue,
		)
	},
	sema.PublicKeyVerifyPoPFunctionType,
)
