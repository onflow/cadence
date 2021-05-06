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
	"strconv"
	"strings"

	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/norm"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/common/orderedmap"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
)

type DynamicTypeResults map[Value]DynamicType

type TypeConformanceResults map[valueDynamicTypePair]bool

type valueDynamicTypePair struct {
	value       Value
	dynamicType DynamicType
}

// Value

type Value interface {
	fmt.Stringer
	IsValue()
	Accept(interpreter *Interpreter, visitor Visitor)
	DynamicType(interpreter *Interpreter, results DynamicTypeResults) DynamicType
	Copy() Value
	GetOwner() *common.Address
	SetOwner(address *common.Address)
	IsModified() bool
	SetModified(modified bool)
	StaticType() StaticType
	ConformsToDynamicType(interpreter *Interpreter, dynamicType DynamicType, results TypeConformanceResults) bool
}

// ValueIndexableValue

type ValueIndexableValue interface {
	Get(interpreter *Interpreter, getLocationRange func() LocationRange, key Value) Value
	Set(interpreter *Interpreter, getLocationRange func() LocationRange, key Value, value Value)
}

// MemberAccessibleValue

type MemberAccessibleValue interface {
	GetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value
	SetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string, value Value)
}

// ConcatenatableValue

type ConcatenatableValue interface {
	Concat(other ConcatenatableValue) Value
}

// AllAppendableValue

type AllAppendableValue interface {
	AppendAll(other AllAppendableValue)
}

// EquatableValue

type EquatableValue interface {
	Value
	Equal(other Value, interpreter *Interpreter, loadDeferred bool) bool
}

// DestroyableValue

type DestroyableValue interface {
	Destroy(interpreter *Interpreter, getLocationRange func() LocationRange)
}

// HasKeyString

type HasKeyString interface {
	KeyString() string
}

// TypeValue

type TypeValue struct {
	Type StaticType
}

func (TypeValue) IsValue() {}

func (v TypeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitTypeValue(interpreter, v)
}

func (TypeValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return MetaTypeDynamicType{}
}

func (TypeValue) StaticType() StaticType {
	return PrimitiveStaticTypeMetaType
}

func (v TypeValue) Copy() Value {
	return v
}

func (TypeValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (TypeValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (TypeValue) IsModified() bool {
	return false
}

func (TypeValue) SetModified(_ bool) {
	// NO-OP
}

func (v TypeValue) String() string {
	var typeString string
	staticType := v.Type
	if staticType != nil {
		typeString = staticType.String()
	}

	return format.TypeValue(typeString)
}

func (v TypeValue) Equal(other Value, _ *Interpreter, _ bool) bool {
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

func (v TypeValue) GetMember(inter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "identifier":
		var typeID string
		staticType := v.Type
		if staticType != nil {
			typeID = string(inter.ConvertStaticToSemaType(staticType).ID())
		}
		return NewStringValue(typeID)
	}

	return nil
}

func (v TypeValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v TypeValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(MetaTypeDynamicType)
	return ok
}

// VoidValue

type VoidValue struct{}

func (VoidValue) IsValue() {}

func (v VoidValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitVoidValue(interpreter, v)
}

func (VoidValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return VoidDynamicType{}
}

func (VoidValue) StaticType() StaticType {
	return PrimitiveStaticTypeVoid
}

func (v VoidValue) Copy() Value {
	return v
}

func (VoidValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (VoidValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (VoidValue) IsModified() bool {
	return false
}

func (VoidValue) SetModified(_ bool) {
	// NO-OP
}

func (VoidValue) String() string {
	return format.Void
}

func (v VoidValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(VoidDynamicType)
	return ok
}

// BoolValue

type BoolValue bool

func (BoolValue) IsValue() {}

func (v BoolValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitBoolValue(interpreter, v)
}

func (BoolValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return BoolDynamicType{}
}

func (BoolValue) StaticType() StaticType {
	return PrimitiveStaticTypeBool
}

func (v BoolValue) Copy() Value {
	return v
}

func (BoolValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (BoolValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (BoolValue) IsModified() bool {
	return false
}

func (BoolValue) SetModified(_ bool) {
	// NO-OP
}

func (v BoolValue) Negate() BoolValue {
	return !v
}

func (v BoolValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherBool, ok := other.(BoolValue)
	if !ok {
		return false
	}
	return bool(v) == bool(otherBool)
}

func (v BoolValue) String() string {
	return format.Bool(bool(v))
}

func (v BoolValue) KeyString() string {
	if v {
		return "true"
	}
	return "false"
}

func (v BoolValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(BoolDynamicType)
	return ok
}

// StringValue

type StringValue struct {
	Str      string
	modified bool
}

func NewStringValue(str string) *StringValue {
	return &StringValue{
		Str:      str,
		modified: true,
	}
}

func (*StringValue) IsValue() {}

func (v *StringValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitStringValue(interpreter, v)
}

func (*StringValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return StringDynamicType{}
}

func (StringValue) StaticType() StaticType {
	return PrimitiveStaticTypeString
}

func (v *StringValue) Copy() Value {
	return NewStringValue(v.Str)
}

func (*StringValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (*StringValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (v *StringValue) IsModified() bool {
	return v.modified
}

func (v *StringValue) SetModified(modified bool) {
	v.modified = modified
}

func (v *StringValue) String() string {
	return format.String(v.Str)
}

func (v *StringValue) KeyString() string {
	return v.Str
}

func (v *StringValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherString, ok := other.(*StringValue)
	if !ok {
		return false
	}
	return v.NormalForm() == otherString.NormalForm()
}

func (v *StringValue) NormalForm() string {
	return norm.NFC.String(v.Str)
}

func (v *StringValue) Concat(other ConcatenatableValue) Value {
	otherString := other.(*StringValue)

	var sb strings.Builder

	sb.WriteString(v.Str)
	sb.WriteString(otherString.Str)

	return NewStringValue(sb.String())
}

func (v *StringValue) Slice(from IntValue, to IntValue) Value {
	fromInt := from.ToInt()
	toInt := to.ToInt()
	return NewStringValue(v.Str[fromInt:toInt])
}

func (v *StringValue) Get(_ *Interpreter, _ func() LocationRange, key Value) Value {
	i := key.(NumberValue).ToInt()

	// TODO: optimize grapheme clusters to prevent unnecessary iteration
	graphemes := uniseg.NewGraphemes(v.Str)
	graphemes.Next()

	for j := 0; j < i; j++ {
		graphemes.Next()
	}

	char := graphemes.Str()

	return NewStringValue(char)
}

func (v *StringValue) Set(_ *Interpreter, _ func() LocationRange, key Value, value Value) {
	index := key.(NumberValue).ToInt()
	char := value.(*StringValue)
	v.SetIndex(index, char)
}

func (v *StringValue) SetIndex(index int, char *StringValue) {
	v.modified = true

	str := v.Str

	// TODO: optimize grapheme clusters to prevent unnecessary iteration
	graphemes := uniseg.NewGraphemes(str)
	graphemes.Next()

	for j := 0; j < index; j++ {
		graphemes.Next()
	}

	start, end := graphemes.Positions()

	var sb strings.Builder

	sb.WriteString(str[:start])
	sb.WriteString(char.Str)
	sb.WriteString(str[end:])

	v.Str = sb.String()
}

func (v *StringValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "length":
		count := v.Length()
		return NewIntValueFromInt64(int64(count))

	case "concat":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				otherValue := invocation.Arguments[0].(ConcatenatableValue)
				return v.Concat(otherValue)
			},
		)

	case "slice":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				from := invocation.Arguments[0].(IntValue)
				to := invocation.Arguments[1].(IntValue)
				return v.Slice(from, to)
			},
		)

	case "decodeHex":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.DecodeHex()
			},
		)
	}

	return nil
}

// Length returns the number of characters (grapheme clusters)
//
func (v *StringValue) Length() int {
	return uniseg.GraphemeClusterCount(v.Str)
}

// DecodeHex hex-decodes this string and returns an array of UInt8 values
//
func (v *StringValue) DecodeHex() *ArrayValue {
	str := v.Str

	bs, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}

	values := make([]Value, len(str)/2)
	for i, b := range bs {
		values[i] = UInt8Value(b)
	}

	return NewArrayValueUnownedNonCopying(values...)
}

func (*StringValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v *StringValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(StringDynamicType)
	return ok
}

// ArrayValue

type ArrayValue struct {
	Values   []Value
	Owner    *common.Address
	modified bool
}

func NewArrayValueUnownedNonCopying(values ...Value) *ArrayValue {
	// NOTE: new value has no owner

	for _, value := range values {
		value.SetOwner(nil)
	}

	if values == nil {
		values = make([]Value, 0)
	}

	return &ArrayValue{
		Values:   values,
		modified: true,
		Owner:    nil,
	}
}

func (*ArrayValue) IsValue() {}

func (v *ArrayValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitArrayValue(interpreter, v)
	if !descend {
		return
	}

	for _, value := range v.Values {
		value.Accept(interpreter, visitor)
	}
}

func (v *ArrayValue) DynamicType(interpreter *Interpreter, results DynamicTypeResults) DynamicType {
	elementTypes := make([]DynamicType, len(v.Values))

	for i, value := range v.Values {
		elementTypes[i] = value.DynamicType(interpreter, results)
	}

	return ArrayDynamicType{
		ElementTypes: elementTypes,
	}
}

func (v *ArrayValue) StaticType() StaticType {
	// TODO: store static type in array values
	return nil
}

func (v *ArrayValue) Copy() Value {
	// TODO: optimize, use copy-on-write
	copies := make([]Value, len(v.Values))
	for i, value := range v.Values {
		copies[i] = value.Copy()
	}
	return NewArrayValueUnownedNonCopying(copies...)
}

func (v *ArrayValue) GetOwner() *common.Address {
	return v.Owner
}

func (v *ArrayValue) SetOwner(owner *common.Address) {
	if v.Owner == owner {
		return
	}

	v.Owner = owner

	for _, value := range v.Values {
		value.SetOwner(owner)
	}
}

func (v *ArrayValue) IsModified() bool {
	if v.modified {
		return true
	}

	for _, value := range v.Values {
		if value.IsModified() {
			return true
		}
	}

	return false
}

func (v *ArrayValue) SetModified(modified bool) {
	v.modified = modified
}

func (v *ArrayValue) Destroy(interpreter *Interpreter, getLocationRange func() LocationRange) {
	for _, value := range v.Values {
		maybeDestroy(interpreter, getLocationRange, value)
	}
}

func (v *ArrayValue) Concat(other ConcatenatableValue) Value {
	otherArray := other.(*ArrayValue)
	concatenated := append(v.Copy().(*ArrayValue).Values, otherArray.Values...)
	return NewArrayValueUnownedNonCopying(concatenated...)
}

func (v *ArrayValue) Get(_ *Interpreter, getLocationRange func() LocationRange, key Value) Value {
	integerKey := key.(NumberValue).ToInt()
	count := v.Count()

	// Check bounds
	if integerKey < 0 || integerKey >= count {
		panic(ArrayIndexOutOfBoundsError{
			Index:         integerKey,
			MaxIndex:      count - 1,
			LocationRange: getLocationRange(),
		})
	}

	return v.Values[integerKey]
}

func (v *ArrayValue) Set(_ *Interpreter, _ func() LocationRange, key Value, value Value) {
	index := key.(NumberValue).ToInt()
	v.SetIndex(index, value)
}

func (v *ArrayValue) SetIndex(index int, value Value) {
	v.modified = true
	value.SetOwner(v.Owner)
	v.Values[index] = value
}

func (v *ArrayValue) String() string {
	values := make([]string, len(v.Values))
	for i, value := range v.Values {
		values[i] = value.String()
	}
	return format.Array(values)
}

func (v *ArrayValue) Append(element Value) {
	v.modified = true

	element.SetOwner(v.Owner)
	v.Values = append(v.Values, element)
}

func (v *ArrayValue) AppendAll(other AllAppendableValue) {
	v.modified = true

	otherArray := other.(*ArrayValue)
	for _, element := range otherArray.Values {
		element.SetOwner(v.Owner)
	}
	v.Values = append(v.Values, otherArray.Values...)
}

func (v *ArrayValue) Insert(i int, element Value) {
	v.modified = true

	element.SetOwner(v.Owner)
	v.Values = append(v.Values[:i], append([]Value{element}, v.Values[i:]...)...)
}

// TODO: unset owner?
func (v *ArrayValue) Remove(i int) Value {
	v.modified = true

	result := v.Values[i]

	lastIndex := len(v.Values) - 1
	copy(v.Values[i:], v.Values[i+1:])

	// avoid memory leaks by explicitly setting value to nil
	v.Values[lastIndex] = nil

	v.Values = v.Values[:lastIndex]

	return result
}

// TODO: unset owner?
func (v *ArrayValue) RemoveFirst() Value {
	v.modified = true

	var firstElement Value
	firstElement, v.Values = v.Values[0], v.Values[1:]
	return firstElement
}

// TODO: unset owner?
func (v *ArrayValue) RemoveLast() Value {
	v.modified = true

	var lastElement Value
	lastIndex := len(v.Values) - 1
	lastElement, v.Values = v.Values[lastIndex], v.Values[:lastIndex]
	return lastElement
}

func (v *ArrayValue) Contains(needleValue Value) BoolValue {
	needleEquatable := needleValue.(EquatableValue)

	for _, arrayValue := range v.Values {
		if needleEquatable.Equal(arrayValue, nil, true) {
			return true
		}
	}

	return false
}

func (v *ArrayValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "length":
		return NewIntValueFromInt64(int64(v.Count()))

	case "append":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				v.Append(invocation.Arguments[0])
				return VoidValue{}
			},
		)

	case "appendAll":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				otherArray := invocation.Arguments[0].(AllAppendableValue)
				v.AppendAll(otherArray)
				return VoidValue{}
			},
		)

	case "concat":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				otherArray := invocation.Arguments[0].(ConcatenatableValue)
				return v.Concat(otherArray)
			},
		)

	case "insert":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				i := invocation.Arguments[0].(NumberValue).ToInt()
				element := invocation.Arguments[1]
				v.Insert(i, element)
				return VoidValue{}
			},
		)

	case "remove":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				i := invocation.Arguments[0].(NumberValue).ToInt()
				return v.Remove(i)
			},
		)

	case "removeFirst":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.RemoveFirst()
			},
		)

	case "removeLast":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.RemoveLast()
			},
		)

	case "contains":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.Contains(invocation.Arguments[0])
			},
		)

	}

	return nil
}

func (v *ArrayValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v *ArrayValue) Count() int {
	return len(v.Values)
}

func (v *ArrayValue) ConformsToDynamicType(
	interpreter *Interpreter,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	arrayType, ok := dynamicType.(ArrayDynamicType)
	if !ok || len(v.Values) != len(arrayType.ElementTypes) {
		return false
	}

	for index, element := range v.Values {
		if !element.ConformsToDynamicType(interpreter, arrayType.ElementTypes[index], results) {
			return false
		}
	}

	return true
}

func (v *ArrayValue) Equal(other Value, interpreter *Interpreter, loadDeferred bool) bool {
	otherArray, ok := other.(*ArrayValue)
	if !ok {
		return false
	}

	if len(v.Values) != len(otherArray.Values) {
		return false
	}

	for i, value := range v.Values {
		otherValue := otherArray.Values[i]

		equatableValue, ok := value.(EquatableValue)
		if !ok || !equatableValue.Equal(otherValue, interpreter, loadDeferred) {
			return false
		}
	}

	return true
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

func getNumberValueMember(v NumberValue, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return NewStringValue(v.String())
			},
		)

	case sema.ToBigEndianBytesFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return ByteSliceToByteArrayValue(v.ToBigEndianBytes())
			},
		)

	case sema.NumericTypeSaturatingAddFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingPlus(other)
			},
		)

	case sema.NumericTypeSaturatingSubtractFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingMinus(other)
			},
		)

	case sema.NumericTypeSaturatingMultiplyFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingMul(other)
			},
		)

	case sema.NumericTypeSaturatingDivideFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				other := invocation.Arguments[0].(NumberValue)
				return v.SaturatingDiv(other)
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
		// NOTE: safe, UInt64Value is handled by BigNumberValue above
		return NewIntValueFromInt64(int64(value.ToInt()))

	default:
		panic(errors.NewUnreachableError())
	}
}

func (v IntValue) IsValue() {}

func (v IntValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitIntValue(interpreter, v)
}

func (IntValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.IntType}
}

func (IntValue) StaticType() StaticType {
	return PrimitiveStaticTypeInt
}

func (v IntValue) Copy() Value {
	return IntValue{new(big.Int).Set(v.BigInt)}
}

func (IntValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (IntValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (IntValue) IsModified() bool {
	return false
}

func (IntValue) SetModified(_ bool) {
	// NO-OP
}

func (v IntValue) ToInt() int {
	// TODO: handle overflow
	return int(v.BigInt.Int64())
}

func (v IntValue) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v IntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v IntValue) KeyString() string {
	return v.BigInt.String()
}

func (v IntValue) Negate() NumberValue {
	return NewIntValueFromBigInt(new(big.Int).Neg(v.BigInt))
}

func (v IntValue) Plus(other NumberValue) NumberValue {
	o := other.(IntValue)
	res := new(big.Int)
	res.Add(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingPlus(other NumberValue) NumberValue {
	return v.Plus(other)
}

func (v IntValue) Minus(other NumberValue) NumberValue {
	o := other.(IntValue)
	res := new(big.Int)
	res.Sub(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingMinus(other NumberValue) NumberValue {
	return v.Minus(other)
}

func (v IntValue) Mod(other NumberValue) NumberValue {
	o := other.(IntValue)
	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) Mul(other NumberValue) NumberValue {
	o := other.(IntValue)
	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingMul(other NumberValue) NumberValue {
	return v.Mul(other)
}

func (v IntValue) Div(other NumberValue) NumberValue {
	o := other.(IntValue)
	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v IntValue) Less(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(IntValue).BigInt)
	return cmp == -1
}

func (v IntValue) LessEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(IntValue).BigInt)
	return cmp <= 0
}

func (v IntValue) Greater(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(IntValue).BigInt)
	return cmp == 1
}

func (v IntValue) GreaterEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(IntValue).BigInt)
	return cmp >= 0
}

func (v IntValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt, ok := other.(IntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
}

func (v IntValue) BitwiseOr(other IntegerValue) IntegerValue {
	o := other.(IntValue)
	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(IntValue)
	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(IntValue)
	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return IntValue{res}
}

func (v IntValue) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(IntValue)
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
	o := other.(IntValue)
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
	return getNumberValueMember(v, name)
}

func (IntValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v IntValue) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v IntValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.IntType.Equal(numberType.StaticType)
}

// Int8Value

type Int8Value int8

func (Int8Value) IsValue() {}

func (v Int8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt8Value(interpreter, v)
}

func (Int8Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Int8Type}
}

func (Int8Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt8
}

func (v Int8Value) Copy() Value {
	return v
}

func (Int8Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Int8Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Int8Value) IsModified() bool {
	return false
}

func (Int8Value) SetModified(_ bool) {
	// NO-OP
}

func (v Int8Value) String() string {
	return format.Int(int64(v))
}

func (v Int8Value) KeyString() string {
	return strconv.FormatInt(int64(v), 10)
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
	o := other.(Int8Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt8 - o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v < (math.MinInt8 - o)) {
		panic(UnderflowError{})
	}
	return v + o
}

func (v Int8Value) SaturatingPlus(other NumberValue) NumberValue {
	o := other.(Int8Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt8 - o)) {
		return Int8Value(math.MaxInt8)
	} else if (o < 0) && (v < (math.MinInt8 - o)) {
		return Int8Value(math.MinInt8)
	}
	return v + o
}

func (v Int8Value) Minus(other NumberValue) NumberValue {
	o := other.(Int8Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt8 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt8 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int8Value) SaturatingMinus(other NumberValue) NumberValue {
	o := other.(Int8Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt8 + o)) {
		return Int8Value(math.MinInt8)
	} else if (o < 0) && (v > (math.MaxInt8 + o)) {
		return Int8Value(math.MaxInt8)
	}
	return v - o
}

func (v Int8Value) Mod(other NumberValue) NumberValue {
	o := other.(Int8Value)
	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int8Value) Mul(other NumberValue) NumberValue {
	o := other.(Int8Value)
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
	o := other.(Int8Value)
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
	o := other.(Int8Value)
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
	o := other.(Int8Value)
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
	return v < other.(Int8Value)
}

func (v Int8Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Int8Value)
}

func (v Int8Value) Greater(other NumberValue) BoolValue {
	return v > other.(Int8Value)
}

func (v Int8Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Int8Value)
}

func (v Int8Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt8, ok := other.(Int8Value)
	if !ok {
		return false
	}
	return v == otherInt8
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
	o := other.(Int8Value)
	return v | o
}

func (v Int8Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Int8Value)
	return v ^ o
}

func (v Int8Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Int8Value)
	return v & o
}

func (v Int8Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Int8Value)
	return v << o
}

func (v Int8Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Int8Value)
	return v >> o
}

func (v Int8Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Int8Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Int8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Int8Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int8Type.Equal(numberType.StaticType)
}

// Int16Value

type Int16Value int16

func (Int16Value) IsValue() {}

func (v Int16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt16Value(interpreter, v)
}

func (Int16Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Int16Type}
}

func (Int16Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt16
}

func (v Int16Value) Copy() Value {
	return v
}

func (Int16Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Int16Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Int16Value) IsModified() bool {
	return false
}

func (Int16Value) SetModified(_ bool) {
	// NO-OP
}

func (v Int16Value) String() string {
	return format.Int(int64(v))
}

func (v Int16Value) KeyString() string {
	return strconv.FormatInt(int64(v), 10)
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
	o := other.(Int16Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt16 - o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v < (math.MinInt16 - o)) {
		panic(UnderflowError{})
	}
	return v + o
}

func (v Int16Value) SaturatingPlus(other NumberValue) NumberValue {
	o := other.(Int16Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt16 - o)) {
		return Int16Value(math.MaxInt16)
	} else if (o < 0) && (v < (math.MinInt16 - o)) {
		return Int16Value(math.MinInt16)
	}
	return v + o
}

func (v Int16Value) Minus(other NumberValue) NumberValue {
	o := other.(Int16Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt16 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt16 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int16Value) SaturatingMinus(other NumberValue) NumberValue {
	o := other.(Int16Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt16 + o)) {
		return Int16Value(math.MinInt16)
	} else if (o < 0) && (v > (math.MaxInt16 + o)) {
		return Int16Value(math.MaxInt16)
	}
	return v - o
}

func (v Int16Value) Mod(other NumberValue) NumberValue {
	o := other.(Int16Value)
	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int16Value) Mul(other NumberValue) NumberValue {
	o := other.(Int16Value)
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
	o := other.(Int16Value)
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
	o := other.(Int16Value)
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
	o := other.(Int16Value)
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
	return v < other.(Int16Value)
}

func (v Int16Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Int16Value)
}

func (v Int16Value) Greater(other NumberValue) BoolValue {
	return v > other.(Int16Value)
}

func (v Int16Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Int16Value)
}

func (v Int16Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt16, ok := other.(Int16Value)
	if !ok {
		return false
	}
	return v == otherInt16
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
	o := other.(Int16Value)
	return v | o
}

func (v Int16Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Int16Value)
	return v ^ o
}

func (v Int16Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Int16Value)
	return v & o
}

func (v Int16Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Int16Value)
	return v << o
}

func (v Int16Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Int16Value)
	return v >> o
}

func (v Int16Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Int16Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Int16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Int16Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int16Type.Equal(numberType.StaticType)
}

// Int32Value

type Int32Value int32

func (Int32Value) IsValue() {}

func (v Int32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt32Value(interpreter, v)
}

func (Int32Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Int32Type}
}

func (Int32Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt32
}

func (v Int32Value) Copy() Value {
	return v
}

func (Int32Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Int32Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Int32Value) IsModified() bool {
	return false
}

func (Int32Value) SetModified(_ bool) {
	// NO-OP
}

func (v Int32Value) String() string {
	return format.Int(int64(v))
}

func (v Int32Value) KeyString() string {
	return strconv.FormatInt(int64(v), 10)
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
	o := other.(Int32Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt32 - o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v < (math.MinInt32 - o)) {
		panic(UnderflowError{})
	}
	return v + o
}

func (v Int32Value) SaturatingPlus(other NumberValue) NumberValue {
	o := other.(Int32Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt32 - o)) {
		return Int32Value(math.MaxInt32)
	} else if (o < 0) && (v < (math.MinInt32 - o)) {
		return Int32Value(math.MinInt32)
	}
	return v + o
}

func (v Int32Value) Minus(other NumberValue) NumberValue {
	o := other.(Int32Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt32 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt32 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int32Value) SaturatingMinus(other NumberValue) NumberValue {
	o := other.(Int32Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt32 + o)) {
		return Int32Value(math.MinInt32)
	} else if (o < 0) && (v > (math.MaxInt32 + o)) {
		return Int32Value(math.MaxInt32)
	}
	return v - o
}

func (v Int32Value) Mod(other NumberValue) NumberValue {
	o := other.(Int32Value)
	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int32Value) Mul(other NumberValue) NumberValue {
	o := other.(Int32Value)
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
	o := other.(Int32Value)
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
	o := other.(Int32Value)
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
	o := other.(Int32Value)
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
	return v < other.(Int32Value)
}

func (v Int32Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Int32Value)
}

func (v Int32Value) Greater(other NumberValue) BoolValue {
	return v > other.(Int32Value)
}

func (v Int32Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Int32Value)
}

func (v Int32Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt32, ok := other.(Int32Value)
	if !ok {
		return false
	}
	return v == otherInt32
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
	o := other.(Int32Value)
	return v | o
}

func (v Int32Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Int32Value)
	return v ^ o
}

func (v Int32Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Int32Value)
	return v & o
}

func (v Int32Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Int32Value)
	return v << o
}

func (v Int32Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Int32Value)
	return v >> o
}

func (v Int32Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Int32Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Int32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Int32Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int32Type.Equal(numberType.StaticType)
}

// Int64Value

type Int64Value int64

func (Int64Value) IsValue() {}

func (v Int64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt64Value(interpreter, v)
}

func (Int64Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Int64Type}
}

func (Int64Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt64
}

func (v Int64Value) Copy() Value {
	return v
}

func (Int64Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Int64Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Int64Value) IsModified() bool {
	return false
}

func (Int64Value) SetModified(_ bool) {
	// NO-OP
}

func (v Int64Value) String() string {
	return format.Int(int64(v))
}

func (v Int64Value) KeyString() string {
	return strconv.FormatInt(int64(v), 10)
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
	o := other.(Int64Value)
	return Int64Value(safeAddInt64(int64(v), int64(o)))
}

func (v Int64Value) SaturatingPlus(other NumberValue) NumberValue {
	o := other.(Int64Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt64 - o)) {
		return Int64Value(math.MaxInt64)
	} else if (o < 0) && (v < (math.MinInt64 - o)) {
		return Int64Value(math.MinInt64)
	}
	return v + o
}

func (v Int64Value) Minus(other NumberValue) NumberValue {
	o := other.(Int64Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Int64Value) SaturatingMinus(other NumberValue) NumberValue {
	o := other.(Int64Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		return Int64Value(math.MinInt64)
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		return Int64Value(math.MaxInt64)
	}
	return v - o
}

func (v Int64Value) Mod(other NumberValue) NumberValue {
	o := other.(Int64Value)
	// INT33-C
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Int64Value) Mul(other NumberValue) NumberValue {
	o := other.(Int64Value)
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
	o := other.(Int64Value)
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
	o := other.(Int64Value)
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
	o := other.(Int64Value)
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
	return v < other.(Int64Value)
}

func (v Int64Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Int64Value)
}

func (v Int64Value) Greater(other NumberValue) BoolValue {
	return v > other.(Int64Value)
}

func (v Int64Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Int64Value)
}

func (v Int64Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt64, ok := other.(Int64Value)
	if !ok {
		return false
	}
	return v == otherInt64
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
	o := other.(Int64Value)
	return v | o
}

func (v Int64Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Int64Value)
	return v ^ o
}

func (v Int64Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Int64Value)
	return v & o
}

func (v Int64Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Int64Value)
	return v << o
}

func (v Int64Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Int64Value)
	return v >> o
}

func (v Int64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Int64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Int64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Int64Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int64Type.Equal(numberType.StaticType)
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
func (v Int128Value) IsValue() {}

func (v Int128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt128Value(interpreter, v)
}

func (Int128Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Int128Type}
}

func (Int128Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt128
}

func (v Int128Value) Copy() Value {
	return Int128Value{BigInt: new(big.Int).Set(v.BigInt)}
}

func (Int128Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Int128Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Int128Value) IsModified() bool {
	return false
}

func (Int128Value) SetModified(_ bool) {
	// NO-OP
}

func (v Int128Value) ToInt() int {
	// TODO: handle overflow
	return int(v.BigInt.Int64())
}

func (v Int128Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v Int128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int128Value) KeyString() string {
	return v.BigInt.String()
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
	o := other.(Int128Value)
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
	o := other.(Int128Value)
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
	o := other.(Int128Value)
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
	o := other.(Int128Value)
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
	o := other.(Int128Value)
	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) Mul(other NumberValue) NumberValue {
	o := other.(Int128Value)
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
	o := other.(Int128Value)
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
	o := other.(Int128Value)
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
	o := other.(Int128Value)
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
	cmp := v.BigInt.Cmp(other.(Int128Value).BigInt)
	return cmp == -1
}

func (v Int128Value) LessEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(Int128Value).BigInt)
	return cmp <= 0
}

func (v Int128Value) Greater(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(Int128Value).BigInt)
	return cmp == 1
}

func (v Int128Value) GreaterEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(Int128Value).BigInt)
	return cmp >= 0
}

func (v Int128Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt, ok := other.(Int128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
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
	o := other.(Int128Value)
	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Int128Value)
	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Int128Value)
	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return Int128Value{res}
}

func (v Int128Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Int128Value)
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
	o := other.(Int128Value)
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
	return getNumberValueMember(v, name)
}

func (Int128Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Int128Value) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Int128Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int128Type.Equal(numberType.StaticType)
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

func (v Int256Value) IsValue() {}

func (v Int256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitInt256Value(interpreter, v)
}

func (Int256Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Int256Type}
}

func (Int256Value) StaticType() StaticType {
	return PrimitiveStaticTypeInt256
}

func (v Int256Value) Copy() Value {
	return Int256Value{new(big.Int).Set(v.BigInt)}
}

func (Int256Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Int256Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Int256Value) IsModified() bool {
	return false
}

func (Int256Value) SetModified(_ bool) {
	// NO-OP
}

func (v Int256Value) ToInt() int {
	// TODO: handle overflow
	return int(v.BigInt.Int64())
}

func (v Int256Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v Int256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v Int256Value) KeyString() string {
	return v.BigInt.String()
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
	o := other.(Int256Value)
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
	o := other.(Int256Value)
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
	o := other.(Int256Value)
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
	o := other.(Int256Value)
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
	o := other.(Int256Value)
	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) Mul(other NumberValue) NumberValue {
	o := other.(Int256Value)
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
	o := other.(Int256Value)
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
	o := other.(Int256Value)
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
	o := other.(Int256Value)
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
	cmp := v.BigInt.Cmp(other.(Int256Value).BigInt)
	return cmp == -1
}

func (v Int256Value) LessEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(Int256Value).BigInt)
	return cmp <= 0
}

func (v Int256Value) Greater(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(Int256Value).BigInt)
	return cmp == 1
}

func (v Int256Value) GreaterEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(Int256Value).BigInt)
	return cmp >= 0
}

func (v Int256Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt, ok := other.(Int256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
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
	o := other.(Int256Value)
	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Int256Value)
	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Int256Value)
	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return Int256Value{res}
}

func (v Int256Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Int256Value)
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
	o := other.(Int256Value)
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
	return getNumberValueMember(v, name)
}

func (Int256Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Int256Value) ToBigEndianBytes() []byte {
	return SignedBigIntToBigEndianBytes(v.BigInt)
}

func (v Int256Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Int256Type.Equal(numberType.StaticType)
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

func (v UIntValue) IsValue() {}

func (v UIntValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUIntValue(interpreter, v)
}

func (UIntValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UIntType}
}

func (UIntValue) StaticType() StaticType {
	return PrimitiveStaticTypeUInt
}

func (v UIntValue) Copy() Value {
	return UIntValue{new(big.Int).Set(v.BigInt)}
}

func (UIntValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UIntValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UIntValue) IsModified() bool {
	return false
}

func (UIntValue) SetModified(_ bool) {
	// NO-OP
}

func (v UIntValue) ToInt() int {
	// TODO: handle overflow
	return int(v.BigInt.Int64())
}

func (v UIntValue) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v UIntValue) String() string {
	return format.BigInt(v.BigInt)
}

func (v UIntValue) KeyString() string {
	return v.BigInt.String()
}

func (v UIntValue) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UIntValue) Plus(other NumberValue) NumberValue {
	o := other.(UIntValue)
	res := new(big.Int)
	res.Add(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) SaturatingPlus(other NumberValue) NumberValue {
	return v.Plus(other)
}

func (v UIntValue) Minus(other NumberValue) NumberValue {
	o := other.(UIntValue)
	res := new(big.Int)
	res.Sub(v.BigInt, o.BigInt)
	// INT30-C
	if res.Sign() < 0 {
		panic(UnderflowError{})
	}
	return UIntValue{res}
}

func (v UIntValue) SaturatingMinus(other NumberValue) NumberValue {
	o := other.(UIntValue)
	res := new(big.Int)
	res.Sub(v.BigInt, o.BigInt)
	// INT30-C
	if res.Sign() < 0 {
		return UIntValue{sema.UIntTypeMin}
	}
	return UIntValue{res}
}

func (v UIntValue) Mod(other NumberValue) NumberValue {
	o := other.(UIntValue)
	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) Mul(other NumberValue) NumberValue {
	o := other.(UIntValue)
	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) SaturatingMul(other NumberValue) NumberValue {
	return v.Mul(other)
}

func (v UIntValue) Div(other NumberValue) NumberValue {
	o := other.(UIntValue)
	res := new(big.Int)
	// INT33-C
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UIntValue) Less(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UIntValue).BigInt)
	return cmp == -1
}

func (v UIntValue) LessEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UIntValue).BigInt)
	return cmp <= 0
}

func (v UIntValue) Greater(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UIntValue).BigInt)
	return cmp == 1
}

func (v UIntValue) GreaterEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UIntValue).BigInt)
	return cmp >= 0
}

func (v UIntValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherUInt, ok := other.(UIntValue)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherUInt.BigInt)
	return cmp == 0
}

func (v UIntValue) BitwiseOr(other IntegerValue) IntegerValue {
	o := other.(UIntValue)
	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(UIntValue)
	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(UIntValue)
	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return UIntValue{res}
}

func (v UIntValue) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(UIntValue)
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
	o := other.(UIntValue)
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
	return getNumberValueMember(v, name)
}

func (UIntValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UIntValue) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UIntValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UIntType.Equal(numberType.StaticType)
}

// UInt8Value

type UInt8Value uint8

func (UInt8Value) IsValue() {}

func (v UInt8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt8Value(interpreter, v)
}

func (UInt8Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UInt8Type}
}

func (UInt8Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt8
}

func (v UInt8Value) Copy() Value {
	return v
}

func (UInt8Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UInt8Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UInt8Value) IsModified() bool {
	return false
}

func (UInt8Value) SetModified(_ bool) {
	// NO-OP
}

func (v UInt8Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt8Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v UInt8Value) ToInt() int {
	return int(v)
}

func (v UInt8Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt8Value) Plus(other NumberValue) NumberValue {
	sum := v + other.(UInt8Value)
	// INT30-C
	if sum < v {
		panic(OverflowError{})
	}
	return sum
}

func (v UInt8Value) SaturatingPlus(other NumberValue) NumberValue {
	sum := v + other.(UInt8Value)
	// INT30-C
	if sum < v {
		return UInt8Value(math.MaxUint8)
	}
	return sum
}

func (v UInt8Value) Minus(other NumberValue) NumberValue {
	diff := v - other.(UInt8Value)
	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt8Value) SaturatingMinus(other NumberValue) NumberValue {
	diff := v - other.(UInt8Value)
	// INT30-C
	if diff > v {
		return UInt8Value(0)
	}
	return diff
}

func (v UInt8Value) Mod(other NumberValue) NumberValue {
	o := other.(UInt8Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt8Value) Mul(other NumberValue) NumberValue {
	o := other.(UInt8Value)
	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint8 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt8Value) SaturatingMul(other NumberValue) NumberValue {
	o := other.(UInt8Value)
	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint8 / o)) {
		return UInt8Value(math.MaxUint8)
	}
	return v * o
}

func (v UInt8Value) Div(other NumberValue) NumberValue {
	o := other.(UInt8Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt8Value) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UInt8Value) Less(other NumberValue) BoolValue {
	return v < other.(UInt8Value)
}

func (v UInt8Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(UInt8Value)
}

func (v UInt8Value) Greater(other NumberValue) BoolValue {
	return v > other.(UInt8Value)
}

func (v UInt8Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(UInt8Value)
}

func (v UInt8Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherUInt8, ok := other.(UInt8Value)
	if !ok {
		return false
	}
	return v == otherUInt8
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
	o := other.(UInt8Value)
	return v | o
}

func (v UInt8Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(UInt8Value)
	return v ^ o
}

func (v UInt8Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(UInt8Value)
	return v & o
}

func (v UInt8Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(UInt8Value)
	return v << o
}

func (v UInt8Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(UInt8Value)
	return v >> o
}

func (v UInt8Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (UInt8Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UInt8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v UInt8Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt8Type.Equal(numberType.StaticType)
}

// UInt16Value

type UInt16Value uint16

func (UInt16Value) IsValue() {}

func (v UInt16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt16Value(interpreter, v)
}

func (UInt16Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UInt16Type}
}

func (UInt16Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt16
}

func (v UInt16Value) Copy() Value {
	return v
}
func (UInt16Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UInt16Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UInt16Value) IsModified() bool {
	return false
}

func (UInt16Value) SetModified(_ bool) {
	// NO-OP
}

func (v UInt16Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt16Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v UInt16Value) ToInt() int {
	return int(v)
}
func (v UInt16Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) Plus(other NumberValue) NumberValue {
	sum := v + other.(UInt16Value)
	// INT30-C
	if sum < v {
		panic(OverflowError{})
	}
	return sum
}

func (v UInt16Value) SaturatingPlus(other NumberValue) NumberValue {
	sum := v + other.(UInt16Value)
	// INT30-C
	if sum < v {
		return UInt16Value(math.MaxUint16)
	}
	return sum
}

func (v UInt16Value) Minus(other NumberValue) NumberValue {
	diff := v - other.(UInt16Value)
	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt16Value) SaturatingMinus(other NumberValue) NumberValue {
	diff := v - other.(UInt16Value)
	// INT30-C
	if diff > v {
		return UInt16Value(0)
	}
	return diff
}

func (v UInt16Value) Mod(other NumberValue) NumberValue {
	o := other.(UInt16Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt16Value) Mul(other NumberValue) NumberValue {
	o := other.(UInt16Value)
	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt16Value) SaturatingMul(other NumberValue) NumberValue {
	o := other.(UInt16Value)
	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint16 / o)) {
		return UInt16Value(math.MaxUint16)
	}
	return v * o
}

func (v UInt16Value) Div(other NumberValue) NumberValue {
	o := other.(UInt16Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt16Value) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UInt16Value) Less(other NumberValue) BoolValue {
	return v < other.(UInt16Value)
}

func (v UInt16Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(UInt16Value)
}

func (v UInt16Value) Greater(other NumberValue) BoolValue {
	return v > other.(UInt16Value)
}

func (v UInt16Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(UInt16Value)
}

func (v UInt16Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherUInt16, ok := other.(UInt16Value)
	if !ok {
		return false
	}
	return v == otherUInt16
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
	o := other.(UInt16Value)
	return v | o
}

func (v UInt16Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(UInt16Value)
	return v ^ o
}

func (v UInt16Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(UInt16Value)
	return v & o
}

func (v UInt16Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(UInt16Value)
	return v << o
}

func (v UInt16Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(UInt16Value)
	return v >> o
}

func (v UInt16Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (UInt16Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UInt16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v UInt16Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt16Type.Equal(numberType.StaticType)
}

// UInt32Value

type UInt32Value uint32

func (UInt32Value) IsValue() {}

func (v UInt32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt32Value(interpreter, v)
}

func (UInt32Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UInt32Type}
}

func (UInt32Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt32
}

func (v UInt32Value) Copy() Value {
	return v
}

func (UInt32Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UInt32Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UInt32Value) IsModified() bool {
	return false
}

func (UInt32Value) SetModified(_ bool) {
	// NO-OP
}

func (v UInt32Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt32Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v UInt32Value) ToInt() int {
	return int(v)
}

func (v UInt32Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) Plus(other NumberValue) NumberValue {
	sum := v + other.(UInt32Value)
	// INT30-C
	if sum < v {
		panic(OverflowError{})
	}
	return sum
}

func (v UInt32Value) SaturatingPlus(other NumberValue) NumberValue {
	sum := v + other.(UInt32Value)
	// INT30-C
	if sum < v {
		return UInt32Value(math.MaxUint32)
	}
	return sum
}

func (v UInt32Value) Minus(other NumberValue) NumberValue {
	diff := v - other.(UInt32Value)
	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt32Value) SaturatingMinus(other NumberValue) NumberValue {
	diff := v - other.(UInt32Value)
	// INT30-C
	if diff > v {
		return UInt32Value(0)
	}
	return diff
}

func (v UInt32Value) Mod(other NumberValue) NumberValue {
	o := other.(UInt32Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt32Value) Mul(other NumberValue) NumberValue {
	o := other.(UInt32Value)
	if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt32Value) SaturatingMul(other NumberValue) NumberValue {
	o := other.(UInt32Value)
	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint32 / o)) {
		return UInt32Value(math.MaxUint32)
	}
	return v * o
}

func (v UInt32Value) Div(other NumberValue) NumberValue {
	o := other.(UInt32Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt32Value) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UInt32Value) Less(other NumberValue) BoolValue {
	return v < other.(UInt32Value)
}

func (v UInt32Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(UInt32Value)
}

func (v UInt32Value) Greater(other NumberValue) BoolValue {
	return v > other.(UInt32Value)
}

func (v UInt32Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(UInt32Value)
}

func (v UInt32Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherUInt32, ok := other.(UInt32Value)
	if !ok {
		return false
	}
	return v == otherUInt32
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
	o := other.(UInt32Value)
	return v | o
}

func (v UInt32Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(UInt32Value)
	return v ^ o
}

func (v UInt32Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(UInt32Value)
	return v & o
}

func (v UInt32Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(UInt32Value)
	return v << o
}

func (v UInt32Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(UInt32Value)
	return v >> o
}

func (v UInt32Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (UInt32Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UInt32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v UInt32Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt32Type.Equal(numberType.StaticType)
}

// UInt64Value

type UInt64Value uint64

func (UInt64Value) IsValue() {}

func (v UInt64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt64Value(interpreter, v)
}

func (UInt64Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UInt64Type}
}

func (UInt64Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt64
}

func (v UInt64Value) Copy() Value {
	return v
}

func (UInt64Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UInt64Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UInt64Value) IsModified() bool {
	return false
}

func (UInt64Value) SetModified(_ bool) {
	// NO-OP
}

func (v UInt64Value) String() string {
	return format.Uint(uint64(v))
}

func (v UInt64Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v UInt64Value) ToInt() int {
	return int(v)
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
	o := other.(UInt64Value)
	return UInt64Value(safeAddUint64(uint64(v), uint64(o)))
}

func (v UInt64Value) SaturatingPlus(other NumberValue) NumberValue {
	sum := v + other.(UInt64Value)
	// INT30-C
	if sum < v {
		return UInt64Value(math.MaxUint64)
	}
	return sum
}

func (v UInt64Value) Minus(other NumberValue) NumberValue {
	diff := v - other.(UInt64Value)
	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UInt64Value) SaturatingMinus(other NumberValue) NumberValue {
	diff := v - other.(UInt64Value)
	// INT30-C
	if diff > v {
		return UInt64Value(0)
	}
	return diff
}

func (v UInt64Value) Mod(other NumberValue) NumberValue {
	o := other.(UInt64Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v UInt64Value) Mul(other NumberValue) NumberValue {
	o := other.(UInt64Value)
	if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
		panic(OverflowError{})
	}
	return v * o
}

func (v UInt64Value) SaturatingMul(other NumberValue) NumberValue {
	o := other.(UInt64Value)
	// INT30-C
	if (v > 0) && (o > 0) && (v > (math.MaxUint64 / o)) {
		return UInt64Value(math.MaxUint64)
	}
	return v * o
}

func (v UInt64Value) Div(other NumberValue) NumberValue {
	o := other.(UInt64Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v UInt64Value) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UInt64Value) Less(other NumberValue) BoolValue {
	return v < other.(UInt64Value)
}

func (v UInt64Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(UInt64Value)
}

func (v UInt64Value) Greater(other NumberValue) BoolValue {
	return v > other.(UInt64Value)
}

func (v UInt64Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(UInt64Value)
}

func (v UInt64Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherUInt64, ok := other.(UInt64Value)
	if !ok {
		return false
	}
	return v == otherUInt64
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
	o := other.(UInt64Value)
	return v | o
}

func (v UInt64Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(UInt64Value)
	return v ^ o
}

func (v UInt64Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(UInt64Value)
	return v & o
}

func (v UInt64Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(UInt64Value)
	return v << o
}

func (v UInt64Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(UInt64Value)
	return v >> o
}

func (v UInt64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (UInt64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UInt64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UInt64Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt64Type.Equal(numberType.StaticType)
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

func (v UInt128Value) IsValue() {}

func (v UInt128Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt128Value(interpreter, v)
}

func (UInt128Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UInt128Type}
}

func (UInt128Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt128
}

func (v UInt128Value) Copy() Value {
	return UInt128Value{new(big.Int).Set(v.BigInt)}
}

func (UInt128Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UInt128Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UInt128Value) IsModified() bool {
	return false
}

func (UInt128Value) SetModified(_ bool) {
	// NO-OP
}

func (v UInt128Value) ToInt() int {
	// TODO: handle overflow
	return int(v.BigInt.Int64())
}

func (v UInt128Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v UInt128Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v UInt128Value) KeyString() string {
	return v.BigInt.String()
}

func (v UInt128Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt128Value) Plus(other NumberValue) NumberValue {
	sum := new(big.Int)
	sum.Add(v.BigInt, other.(UInt128Value).BigInt)
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
	sum := new(big.Int)
	sum.Add(v.BigInt, other.(UInt128Value).BigInt)
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
	diff := new(big.Int)
	diff.Sub(v.BigInt, other.(UInt128Value).BigInt)
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
	diff := new(big.Int)
	diff.Sub(v.BigInt, other.(UInt128Value).BigInt)
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
	o := other.(UInt128Value)
	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) Mul(other NumberValue) NumberValue {
	o := other.(UInt128Value)
	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return UInt128Value{res}
}

func (v UInt128Value) SaturatingMul(other NumberValue) NumberValue {
	o := other.(UInt128Value)
	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
		return UInt128Value{sema.UInt128TypeMaxIntBig}
	}
	return UInt128Value{res}
}

func (v UInt128Value) Div(other NumberValue) NumberValue {
	o := other.(UInt128Value)
	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UInt128Value) Less(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt128Value).BigInt)
	return cmp == -1
}

func (v UInt128Value) LessEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt128Value).BigInt)
	return cmp <= 0
}

func (v UInt128Value) Greater(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt128Value).BigInt)
	return cmp == 1
}

func (v UInt128Value) GreaterEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt128Value).BigInt)
	return cmp >= 0
}

func (v UInt128Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt, ok := other.(UInt128Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
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
	o := other.(UInt128Value)
	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(UInt128Value)
	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(UInt128Value)
	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return UInt128Value{res}
}

func (v UInt128Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(UInt128Value)
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
	o := other.(UInt128Value)
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
	return getNumberValueMember(v, name)
}

func (UInt128Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UInt128Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UInt128Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt128Type.Equal(numberType.StaticType)
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

func (v UInt256Value) IsValue() {}

func (v UInt256Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUInt256Value(interpreter, v)
}

func (UInt256Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UInt256Type}
}

func (UInt256Value) StaticType() StaticType {
	return PrimitiveStaticTypeUInt256
}

func (v UInt256Value) Copy() Value {
	return UInt256Value{new(big.Int).Set(v.BigInt)}
}

func (UInt256Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UInt256Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UInt256Value) IsModified() bool {
	return false
}

func (UInt256Value) SetModified(_ bool) {
	// NO-OP
}

func (v UInt256Value) ToInt() int {
	// TODO: handle overflow
	return int(v.BigInt.Int64())
}

func (v UInt256Value) ToBigInt() *big.Int {
	return new(big.Int).Set(v.BigInt)
}

func (v UInt256Value) String() string {
	return format.BigInt(v.BigInt)
}

func (v UInt256Value) KeyString() string {
	return v.BigInt.String()
}

func (v UInt256Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) Plus(other NumberValue) NumberValue {
	sum := new(big.Int)
	sum.Add(v.BigInt, other.(UInt256Value).BigInt)
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
	sum := new(big.Int)
	sum.Add(v.BigInt, other.(UInt256Value).BigInt)
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
	diff := new(big.Int)
	diff.Sub(v.BigInt, other.(UInt256Value).BigInt)
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
	diff := new(big.Int)
	diff.Sub(v.BigInt, other.(UInt256Value).BigInt)
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
	o := other.(UInt256Value)
	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Rem(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) Mul(other NumberValue) NumberValue {
	o := other.(UInt256Value)
	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
		panic(OverflowError{})
	}
	return UInt256Value{res}
}

func (v UInt256Value) SaturatingMul(other NumberValue) NumberValue {
	o := other.(UInt256Value)
	res := new(big.Int)
	res.Mul(v.BigInt, o.BigInt)
	if res.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
		return UInt256Value{sema.UInt256TypeMaxIntBig}
	}
	return UInt256Value{res}
}

func (v UInt256Value) Div(other NumberValue) NumberValue {
	o := other.(UInt256Value)
	res := new(big.Int)
	if o.BigInt.Cmp(res) == 0 {
		panic(DivisionByZeroError{})
	}
	res.Div(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UInt256Value) Less(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt256Value).BigInt)
	return cmp == -1
}

func (v UInt256Value) LessEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt256Value).BigInt)
	return cmp <= 0
}

func (v UInt256Value) Greater(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt256Value).BigInt)
	return cmp == 1
}

func (v UInt256Value) GreaterEqual(other NumberValue) BoolValue {
	cmp := v.BigInt.Cmp(other.(UInt256Value).BigInt)
	return cmp >= 0
}

func (v UInt256Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherInt, ok := other.(UInt256Value)
	if !ok {
		return false
	}
	cmp := v.BigInt.Cmp(otherInt.BigInt)
	return cmp == 0
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
	o := other.(UInt256Value)
	res := new(big.Int)
	res.Or(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(UInt256Value)
	res := new(big.Int)
	res.Xor(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(UInt256Value)
	res := new(big.Int)
	res.And(v.BigInt, o.BigInt)
	return UInt256Value{res}
}

func (v UInt256Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(UInt256Value)
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
	o := other.(UInt256Value)
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
	return getNumberValueMember(v, name)
}

func (UInt256Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UInt256Value) ToBigEndianBytes() []byte {
	return UnsignedBigIntToBigEndianBytes(v.BigInt)
}

func (v UInt256Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UInt256Type.Equal(numberType.StaticType)
}

// Word8Value

type Word8Value uint8

func (Word8Value) IsValue() {}

func (v Word8Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord8Value(interpreter, v)
}

func (Word8Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Word8Type}
}

func (Word8Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord8
}

func (v Word8Value) Copy() Value {
	return v
}

func (Word8Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Word8Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Word8Value) IsModified() bool {
	return false
}

func (Word8Value) SetModified(_ bool) {
	// NO-OP
}

func (v Word8Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word8Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v Word8Value) ToInt() int {
	return int(v)
}

func (v Word8Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) Plus(other NumberValue) NumberValue {
	return v + other.(Word8Value)
}

func (v Word8Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Minus(other NumberValue) NumberValue {
	return v - other.(Word8Value)
}

func (v Word8Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Mod(other NumberValue) NumberValue {
	o := other.(Word8Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word8Value) Mul(other NumberValue) NumberValue {
	return v * other.(Word8Value)
}

func (v Word8Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Div(other NumberValue) NumberValue {
	o := other.(Word8Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word8Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word8Value) Less(other NumberValue) BoolValue {
	return v < other.(Word8Value)
}

func (v Word8Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Word8Value)
}

func (v Word8Value) Greater(other NumberValue) BoolValue {
	return v > other.(Word8Value)
}

func (v Word8Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Word8Value)
}

func (v Word8Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherWord8, ok := other.(Word8Value)
	if !ok {
		return false
	}
	return v == otherWord8
}

func ConvertWord8(value Value) Word8Value {
	return Word8Value(ConvertUInt8(value))
}

func (v Word8Value) BitwiseOr(other IntegerValue) IntegerValue {
	o := other.(Word8Value)
	return v | o
}

func (v Word8Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Word8Value)
	return v ^ o
}

func (v Word8Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Word8Value)
	return v & o
}

func (v Word8Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Word8Value)
	return v << o
}

func (v Word8Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Word8Value)
	return v >> o
}

func (v Word8Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Word8Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Word8Value) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Word8Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word8Type.Equal(numberType.StaticType)
}

// Word16Value

type Word16Value uint16

func (Word16Value) IsValue() {}

func (v Word16Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord16Value(interpreter, v)
}

func (Word16Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Word16Type}
}

func (Word16Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord16
}

func (v Word16Value) Copy() Value {
	return v
}
func (Word16Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Word16Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Word16Value) IsModified() bool {
	return false
}

func (Word16Value) SetModified(_ bool) {
	// NO-OP
}

func (v Word16Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word16Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v Word16Value) ToInt() int {
	return int(v)
}
func (v Word16Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) Plus(other NumberValue) NumberValue {
	return v + other.(Word16Value)
}

func (v Word16Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Minus(other NumberValue) NumberValue {
	return v - other.(Word16Value)
}

func (v Word16Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Mod(other NumberValue) NumberValue {
	o := other.(Word16Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word16Value) Mul(other NumberValue) NumberValue {
	return v * other.(Word16Value)
}

func (v Word16Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Div(other NumberValue) NumberValue {
	o := other.(Word16Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word16Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word16Value) Less(other NumberValue) BoolValue {
	return v < other.(Word16Value)
}

func (v Word16Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Word16Value)
}

func (v Word16Value) Greater(other NumberValue) BoolValue {
	return v > other.(Word16Value)
}

func (v Word16Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Word16Value)
}

func (v Word16Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherWord16, ok := other.(Word16Value)
	if !ok {
		return false
	}
	return v == otherWord16
}

func ConvertWord16(value Value) Word16Value {
	return Word16Value(ConvertUInt16(value))
}

func (v Word16Value) BitwiseOr(other IntegerValue) IntegerValue {
	o := other.(Word16Value)
	return v | o
}

func (v Word16Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Word16Value)
	return v ^ o
}

func (v Word16Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Word16Value)
	return v & o
}

func (v Word16Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Word16Value)
	return v << o
}

func (v Word16Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Word16Value)
	return v >> o
}

func (v Word16Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Word16Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Word16Value) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Word16Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word16Type.Equal(numberType.StaticType)
}

// Word32Value

type Word32Value uint32

func (Word32Value) IsValue() {}

func (v Word32Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord32Value(interpreter, v)
}

func (Word32Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Word32Type}
}

func (Word32Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord32
}

func (v Word32Value) Copy() Value {
	return v
}

func (Word32Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Word32Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Word32Value) IsModified() bool {
	return false
}

func (Word32Value) SetModified(_ bool) {
	// NO-OP
}

func (v Word32Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word32Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v Word32Value) ToInt() int {
	return int(v)
}

func (v Word32Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Plus(other NumberValue) NumberValue {
	return v + other.(Word32Value)
}

func (v Word32Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Minus(other NumberValue) NumberValue {
	return v - other.(Word32Value)
}

func (v Word32Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Mod(other NumberValue) NumberValue {
	o := other.(Word32Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word32Value) Mul(other NumberValue) NumberValue {
	return v * other.(Word32Value)
}

func (v Word32Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Div(other NumberValue) NumberValue {
	o := other.(Word32Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word32Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word32Value) Less(other NumberValue) BoolValue {
	return v < other.(Word32Value)
}

func (v Word32Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Word32Value)
}

func (v Word32Value) Greater(other NumberValue) BoolValue {
	return v > other.(Word32Value)
}

func (v Word32Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Word32Value)
}

func (v Word32Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherWord32, ok := other.(Word32Value)
	if !ok {
		return false
	}
	return v == otherWord32
}

func ConvertWord32(value Value) Word32Value {
	return Word32Value(ConvertUInt32(value))
}

func (v Word32Value) BitwiseOr(other IntegerValue) IntegerValue {
	o := other.(Word32Value)
	return v | o
}

func (v Word32Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Word32Value)
	return v ^ o
}

func (v Word32Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Word32Value)
	return v & o
}

func (v Word32Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Word32Value)
	return v << o
}

func (v Word32Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Word32Value)
	return v >> o
}

func (v Word32Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Word32Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Word32Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word32Type.Equal(numberType.StaticType)
}

// Word64Value

type Word64Value uint64

func (Word64Value) IsValue() {}

func (v Word64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitWord64Value(interpreter, v)
}

func (Word64Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Word64Type}
}

func (Word64Value) StaticType() StaticType {
	return PrimitiveStaticTypeWord64
}

func (v Word64Value) Copy() Value {
	return v
}

func (Word64Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Word64Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Word64Value) IsModified() bool {
	return false
}

func (Word64Value) SetModified(_ bool) {
	// NO-OP
}

func (v Word64Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word64Value) KeyString() string {
	return strconv.FormatUint(uint64(v), 10)
}

func (v Word64Value) ToInt() int {
	return int(v)
}

func (v Word64Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Plus(other NumberValue) NumberValue {
	return v + other.(Word64Value)
}

func (v Word64Value) SaturatingPlus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Minus(other NumberValue) NumberValue {
	return v - other.(Word64Value)
}

func (v Word64Value) SaturatingMinus(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Mod(other NumberValue) NumberValue {
	o := other.(Word64Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v % o
}

func (v Word64Value) Mul(other NumberValue) NumberValue {
	return v * other.(Word64Value)
}

func (v Word64Value) SaturatingMul(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Div(other NumberValue) NumberValue {
	o := other.(Word64Value)
	if o == 0 {
		panic(DivisionByZeroError{})
	}
	return v / o
}

func (v Word64Value) SaturatingDiv(_ NumberValue) NumberValue {
	panic(errors.UnreachableError{})
}

func (v Word64Value) Less(other NumberValue) BoolValue {
	return v < other.(Word64Value)
}

func (v Word64Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Word64Value)
}

func (v Word64Value) Greater(other NumberValue) BoolValue {
	return v > other.(Word64Value)
}

func (v Word64Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Word64Value)
}

func (v Word64Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherWord64, ok := other.(Word64Value)
	if !ok {
		return false
	}
	return v == otherWord64
}

func ConvertWord64(value Value) Word64Value {
	return Word64Value(ConvertUInt64(value))
}

func (v Word64Value) BitwiseOr(other IntegerValue) IntegerValue {
	o := other.(Word64Value)
	return v | o
}

func (v Word64Value) BitwiseXor(other IntegerValue) IntegerValue {
	o := other.(Word64Value)
	return v ^ o
}

func (v Word64Value) BitwiseAnd(other IntegerValue) IntegerValue {
	o := other.(Word64Value)
	return v & o
}

func (v Word64Value) BitwiseLeftShift(other IntegerValue) IntegerValue {
	o := other.(Word64Value)
	return v << o
}

func (v Word64Value) BitwiseRightShift(other IntegerValue) IntegerValue {
	o := other.(Word64Value)
	return v >> o
}

func (v Word64Value) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	return getNumberValueMember(v, name)
}

func (Word64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Word64Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Word64Type.Equal(numberType.StaticType)
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

func (Fix64Value) IsValue() {}

func (v Fix64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitFix64Value(interpreter, v)
}

func (Fix64Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.Fix64Type}
}

func (Fix64Value) StaticType() StaticType {
	return PrimitiveStaticTypeFix64
}

func (v Fix64Value) Copy() Value {
	return v
}

func (Fix64Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (Fix64Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (Fix64Value) IsModified() bool {
	return false
}

func (Fix64Value) SetModified(_ bool) {
	// NO-OP
}

func (v Fix64Value) String() string {
	return format.Fix64(int64(v))
}

func (v Fix64Value) KeyString() string {
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
	o := other.(Fix64Value)
	return Fix64Value(safeAddInt64(int64(v), int64(o)))
}

func (v Fix64Value) SaturatingPlus(other NumberValue) NumberValue {
	o := other.(Fix64Value)
	// INT32-C
	if (o > 0) && (v > (math.MaxInt64 - o)) {
		return Fix64Value(math.MaxInt64)
	} else if (o < 0) && (v < (math.MinInt64 - o)) {
		return Fix64Value(math.MinInt64)
	}
	return v + o
}

func (v Fix64Value) Minus(other NumberValue) NumberValue {
	o := other.(Fix64Value)
	// INT32-C
	if (o > 0) && (v < (math.MinInt64 + o)) {
		panic(OverflowError{})
	} else if (o < 0) && (v > (math.MaxInt64 + o)) {
		panic(UnderflowError{})
	}
	return v - o
}

func (v Fix64Value) SaturatingMinus(other NumberValue) NumberValue {
	o := other.(Fix64Value)
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
	o := other.(Fix64Value)

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
	o := other.(Fix64Value)

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
	o := other.(Fix64Value)

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
	o := other.(Fix64Value)

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
	o := other.(Fix64Value)
	// v - int(v/o) * o
	quotient := v.Div(o).(Fix64Value)
	truncatedQuotient := (int64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
	return v.Minus(Fix64Value(truncatedQuotient).Mul(o))
}

func (v Fix64Value) Less(other NumberValue) BoolValue {
	return v < other.(Fix64Value)
}

func (v Fix64Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(Fix64Value)
}

func (v Fix64Value) Greater(other NumberValue) BoolValue {
	return v > other.(Fix64Value)
}

func (v Fix64Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(Fix64Value)
}

func (v Fix64Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherFix64, ok := other.(Fix64Value)
	if !ok {
		return false
	}
	return v == otherFix64
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
	return getNumberValueMember(v, name)
}

func (Fix64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v Fix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Fix64Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.Fix64Type.Equal(numberType.StaticType)
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

func (UFix64Value) IsValue() {}

func (v UFix64Value) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitUFix64Value(interpreter, v)
}

func (UFix64Value) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NumberDynamicType{sema.UFix64Type}
}

func (UFix64Value) StaticType() StaticType {
	return PrimitiveStaticTypeUFix64
}

func (v UFix64Value) Copy() Value {
	return v
}

func (UFix64Value) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (UFix64Value) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (UFix64Value) IsModified() bool {
	return false
}

func (UFix64Value) SetModified(_ bool) {
	// NO-OP
}

func (v UFix64Value) String() string {
	return format.UFix64(uint64(v))
}

func (v UFix64Value) KeyString() string {
	return v.String()
}

func (v UFix64Value) ToInt() int {
	return int(v / sema.Fix64Factor)
}

func (v UFix64Value) Negate() NumberValue {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) Plus(other NumberValue) NumberValue {
	o := other.(UFix64Value)
	return UFix64Value(safeAddUint64(uint64(v), uint64(o)))
}

func (v UFix64Value) SaturatingPlus(other NumberValue) NumberValue {
	o := other.(UFix64Value)
	sum := v + o
	// INT30-C
	if sum < v {
		return UFix64Value(math.MaxUint64)
	}
	return sum
}

func (v UFix64Value) Minus(other NumberValue) NumberValue {
	diff := v - other.(UFix64Value)
	// INT30-C
	if diff > v {
		panic(UnderflowError{})
	}
	return diff
}

func (v UFix64Value) SaturatingMinus(other NumberValue) NumberValue {
	diff := v - other.(UFix64Value)
	// INT30-C
	if diff > v {
		return UFix64Value(0)
	}
	return diff
}

func (v UFix64Value) Mul(other NumberValue) NumberValue {
	o := other.(UFix64Value)

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
	o := other.(UFix64Value)

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
	o := other.(UFix64Value)

	a := new(big.Int).SetUint64(uint64(v))
	b := new(big.Int).SetUint64(uint64(o))

	result := new(big.Int).Mul(a, sema.Fix64FactorBig)
	result.Div(result, b)

	return UFix64Value(result.Uint64())
}

func (v UFix64Value) SaturatingDiv(other NumberValue) NumberValue {
	return v.Div(other)
}

func (v UFix64Value) Mod(other NumberValue) NumberValue {
	o := other.(UFix64Value)
	// v - int(v/o) * o
	quotient := v.Div(o).(UFix64Value)
	truncatedQuotient := (uint64(quotient) / sema.Fix64Factor) * sema.Fix64Factor
	return v.Minus(UFix64Value(truncatedQuotient).Mul(o))
}

func (v UFix64Value) Less(other NumberValue) BoolValue {
	return v < other.(UFix64Value)
}

func (v UFix64Value) LessEqual(other NumberValue) BoolValue {
	return v <= other.(UFix64Value)
}

func (v UFix64Value) Greater(other NumberValue) BoolValue {
	return v > other.(UFix64Value)
}

func (v UFix64Value) GreaterEqual(other NumberValue) BoolValue {
	return v >= other.(UFix64Value)
}

func (v UFix64Value) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherUFix64, ok := other.(UFix64Value)
	if !ok {
		return false
	}
	return v == otherUFix64
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
	return getNumberValueMember(v, name)
}

func (UFix64Value) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v UFix64Value) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UFix64Value) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	numberType, ok := dynamicType.(NumberDynamicType)
	return ok && sema.UFix64Type.Equal(numberType.StaticType)
}

// CompositeValue

type CompositeValue struct {
	Location            common.Location
	QualifiedIdentifier string
	Kind                common.CompositeKind
	Fields              *StringValueOrderedMap
	InjectedFields      *StringValueOrderedMap
	ComputedFields      *StringComputedFieldOrderedMap
	NestedVariables     *StringVariableOrderedMap
	Functions           map[string]FunctionValue
	Destructor          FunctionValue
	Owner               *common.Address
	destroyed           bool
	modified            bool
	stringer            func() string
}

type ComputedField func(*Interpreter) Value

func NewCompositeValue(
	location common.Location,
	qualifiedIdentifier string,
	kind common.CompositeKind,
	fields *StringValueOrderedMap,
	owner *common.Address,
) *CompositeValue {
	// TODO: only allocate when setting a field
	if fields == nil {
		fields = NewStringValueOrderedMap()
	}
	return &CompositeValue{
		Location:            location,
		QualifiedIdentifier: qualifiedIdentifier,
		Kind:                kind,
		Fields:              fields,
		Owner:               owner,
		modified:            true,
	}
}

func (v *CompositeValue) Destroy(interpreter *Interpreter, getLocationRange func() LocationRange) {

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

		destructor.Invoke(invocation)
	}

	v.destroyed = true
	v.modified = true
}

func (*CompositeValue) IsValue() {}

func (v *CompositeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitCompositeValue(interpreter, v)
	if !descend {
		return
	}

	v.Fields.Foreach(func(_ string, value Value) {
		value.Accept(interpreter, visitor)
	})
}

func (v *CompositeValue) DynamicType(interpreter *Interpreter, _ DynamicTypeResults) DynamicType {
	staticType := interpreter.getCompositeType(v.Location, v.QualifiedIdentifier)
	return CompositeDynamicType{
		StaticType: staticType,
	}
}

func (v *CompositeValue) StaticType() StaticType {
	return CompositeStaticType{
		Location:            v.Location,
		QualifiedIdentifier: v.QualifiedIdentifier,
	}
}

func (v *CompositeValue) Copy() Value {
	// Resources and contracts are not copied
	switch v.Kind {
	case common.CompositeKindResource, common.CompositeKindContract:
		return v

	default:
		break
	}

	newFields := NewStringValueOrderedMap()
	v.Fields.Foreach(func(fieldName string, value Value) {
		newFields.Set(fieldName, value.Copy())
	})

	// NOTE: not copying functions or destructor – they are linked in

	return &CompositeValue{
		Location:            v.Location,
		QualifiedIdentifier: v.QualifiedIdentifier,
		Kind:                v.Kind,
		Fields:              newFields,
		InjectedFields:      v.InjectedFields,
		ComputedFields:      v.ComputedFields,
		NestedVariables:     v.NestedVariables,
		Functions:           v.Functions,
		Destructor:          v.Destructor,
		destroyed:           v.destroyed,
		// NOTE: new value has no owner
		Owner:    nil,
		modified: true,
	}
}

func (v *CompositeValue) checkStatus(getLocationRange func() LocationRange) {
	if v.destroyed {
		panic(DestroyedCompositeError{
			CompositeKind: v.Kind,
			LocationRange: getLocationRange(),
		})
	}
}

func (v *CompositeValue) GetOwner() *common.Address {
	return v.Owner
}

func (v *CompositeValue) SetOwner(owner *common.Address) {
	if v.Owner == owner {
		return
	}

	v.Owner = owner

	v.Fields.Foreach(func(_ string, value Value) {
		value.SetOwner(owner)
	})
}

func (v *CompositeValue) IsModified() bool {
	if v.modified {
		return true
	}

	for pair := v.Fields.Oldest(); pair != nil; pair = pair.Next() {
		if pair.Value.IsModified() {
			return true
		}
	}

	if v.InjectedFields != nil {
		for pair := v.InjectedFields.Oldest(); pair != nil; pair = pair.Next() {
			if pair.Value.IsModified() {
				return true
			}
		}
	}

	if v.NestedVariables != nil {
		for pair := v.NestedVariables.Oldest(); pair != nil; pair = pair.Next() {
			if pair.Value.GetValue().IsModified() {
				return true
			}
		}
	}

	return false
}

func (v *CompositeValue) SetModified(modified bool) {
	v.modified = modified
}

func (v *CompositeValue) GetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value {
	v.checkStatus(getLocationRange)

	if v.Kind == common.CompositeKindResource &&
		name == sema.ResourceOwnerFieldName {

		return v.OwnerValue(interpreter)
	}

	value, ok := v.Fields.Get(name)
	if ok {
		return value
	}

	if v.NestedVariables != nil {
		variable, ok := v.NestedVariables.Get(name)
		if ok {
			return variable.GetValue()
		}
	}

	interpreter = v.getInterpreter(interpreter)

	if v.ComputedFields != nil {
		if computedField, ok := v.ComputedFields.Get(name); ok {
			return computedField(interpreter)
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
		value, ok = v.InjectedFields.Get(name)
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

	if v.Location == nil || common.LocationsMatch(interpreter.Location, v.Location) {
		return interpreter
	}

	return interpreter.EnsureLoadedWithLocationHandler(v.Location)
}

func (v *CompositeValue) InitializeFunctions(interpreter *Interpreter) {
	if v.Functions != nil {
		return
	}

	v.Functions = interpreter.typeCodes.CompositeCodes[v.TypeID()].CompositeFunctions
}

func (v *CompositeValue) OwnerValue(interpreter *Interpreter) OptionalValue {
	if v.Owner == nil {
		return NilValue{}
	}

	address := AddressValue(*v.Owner)
	ownerAccount := interpreter.accountHandler(address)

	// Owner must be of `PublicAccount` type.

	dynamicTypeResults := DynamicTypeResults{}
	dynamicType := ownerAccount.DynamicType(interpreter, dynamicTypeResults)

	compositeDynamicType, ok := dynamicType.(CompositeDynamicType)

	if !ok || !sema.PublicAccountType.Equal(compositeDynamicType.StaticType) {
		panic(&TypeMismatchError{
			ExpectedType: sema.PublicAccountType,
		})
	}

	return NewSomeValueOwningNonCopying(ownerAccount)
}

func (v *CompositeValue) SetMember(_ *Interpreter, getLocationRange func() LocationRange, name string, value Value) {
	v.checkStatus(getLocationRange)

	v.modified = true

	value.SetOwner(v.Owner)

	v.Fields.Set(name, value)
}

func (v *CompositeValue) String() string {
	if v.stringer != nil {
		return v.stringer()
	}

	return formatComposite(string(v.TypeID()), v.Fields)
}

func formatComposite(typeId string, fields *StringValueOrderedMap) string {
	preparedFields := make([]struct {
		Name  string
		Value string
	}, 0, fields.Len())

	fields.Foreach(func(fieldName string, value Value) {
		preparedFields = append(preparedFields,
			struct {
				Name  string
				Value string
			}{
				Name:  fieldName,
				Value: value.String(),
			},
		)
	})

	return format.Composite(typeId, preparedFields)
}

func (v *CompositeValue) GetField(name string) Value {
	value, _ := v.Fields.Get(name)
	return value
}

func (v *CompositeValue) Equal(other Value, interpreter *Interpreter, loadDeferred bool) bool {
	otherComposite, ok := other.(*CompositeValue)
	if !ok {
		return false
	}

	if !v.StaticType().Equal(otherComposite.StaticType()) ||
		v.Kind != otherComposite.Kind ||
		v.Fields.Len() != otherComposite.Fields.Len() {

		return false
	}

	for pair := v.Fields.Oldest(); pair != nil; pair = pair.Next() {
		key := pair.Key
		value := pair.Value

		otherValue, ok := otherComposite.Fields.Get(key)
		if !ok {
			return false
		}

		equatableValue, ok := value.(EquatableValue)
		if !ok || !equatableValue.Equal(otherValue, interpreter, loadDeferred) {
			return false
		}
	}

	return true
}

func (v *CompositeValue) KeyString() string {
	if v.Kind == common.CompositeKindEnum {
		rawValue, _ := v.Fields.Get(sema.EnumRawValueFieldName)
		return rawValue.String()
	}

	panic(errors.NewUnreachableError())
}

func (v *CompositeValue) TypeID() common.TypeID {
	if v.Location == nil {
		return common.TypeID(v.QualifiedIdentifier)
	}

	return v.Location.TypeID(v.QualifiedIdentifier)
}

func (v *CompositeValue) ConformsToDynamicType(
	interpreter *Interpreter,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {
	compositeDynamicType, ok := dynamicType.(CompositeDynamicType)
	if !ok {
		return false
	}

	compositeType, ok := compositeDynamicType.StaticType.(*sema.CompositeType)
	if !ok ||
		v.Kind != compositeType.Kind ||
		v.QualifiedIdentifier != compositeType.QualifiedIdentifier() ||
		v.Location.ID() != compositeType.Location.ID() {

		return false
	}

	// Here it is assumed that imported values can only have static fields values,
	// but not computed field values.
	if v.Fields.Len() != len(compositeType.Fields) {
		return false
	}

	for _, fieldName := range compositeType.Fields {
		field, ok := v.Fields.Get(fieldName)
		if !ok {
			return false
		}

		member, ok := compositeType.Members.Get(fieldName)
		if !ok {
			return false
		}

		dynamicTypeResults := DynamicTypeResults{}
		fieldDynamicType := field.DynamicType(interpreter, dynamicTypeResults)

		if !IsSubType(fieldDynamicType, member.TypeAnnotation.Type) {
			return false
		}

		if !field.ConformsToDynamicType(interpreter, fieldDynamicType, results) {
			return false
		}
	}

	return true
}

// DictionaryValue

type DictionaryValue struct {
	Keys     *ArrayValue
	Entries  *StringValueOrderedMap
	Owner    *common.Address
	modified bool
	// Deferral of values:
	//
	// Values in the dictionary might be deferred, i.e. are they encoded
	// separately and stored in separate storage keys.
	//
	// DeferredOwner is the account in which the deferred values are stored.
	// The account might differ from the owner: If the dictionary is moved
	// from one account to another, its owner changes, but its deferred values
	// stay stored in the deferred owner's account until the end of the transaction.
	DeferredOwner *common.Address
	// DeferredKeys are the keys which are deferred and have not been loaded from storage yet.
	DeferredKeys *orderedmap.StringStructOrderedMap
	// DeferredStorageKeyBase is the storage key prefix for all deferred keys
	DeferredStorageKeyBase string
	// prevDeferredKeys are the keys which are deferred and have been loaded from storage,
	// i.e. they are keys that were previously in DeferredKeys.
	prevDeferredKeys *orderedmap.StringStructOrderedMap
}

func NewDictionaryValueUnownedNonCopying(keysAndValues ...Value) *DictionaryValue {
	keysAndValuesCount := len(keysAndValues)
	if keysAndValuesCount%2 != 0 {
		panic("uneven number of keys and values")
	}

	result := &DictionaryValue{
		Keys:    NewArrayValueUnownedNonCopying(),
		Entries: NewStringValueOrderedMap(),
		// NOTE: new value has no owner
		Owner:                  nil,
		modified:               true,
		DeferredOwner:          nil,
		DeferredKeys:           nil,
		DeferredStorageKeyBase: "",
		prevDeferredKeys:       nil,
	}

	for i := 0; i < keysAndValuesCount; i += 2 {
		_ = result.Insert(nil, ReturnEmptyLocationRange, keysAndValues[i], keysAndValues[i+1])
	}

	return result
}

func (*DictionaryValue) IsValue() {}

func (v *DictionaryValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitDictionaryValue(interpreter, v)
	if !descend {
		return
	}
	for _, key := range v.Keys.Values {
		key.Accept(interpreter, visitor)

		// NOTE: Force unwrap. This is safe because we are iterating over the keys.
		value := v.Get(interpreter, ReturnEmptyLocationRange, key).(*SomeValue).Value
		value.Accept(interpreter, visitor)
	}
}

func (v *DictionaryValue) DynamicType(interpreter *Interpreter, results DynamicTypeResults) DynamicType {
	entryTypes := make([]struct{ KeyType, ValueType DynamicType }, len(v.Keys.Values))

	for i, key := range v.Keys.Values {
		// NOTE: Force unwrap, otherwise dynamic type check is for optional type.
		// This is safe because we are iterating over the keys.
		value := v.Get(interpreter, ReturnEmptyLocationRange, key).(*SomeValue).Value
		entryTypes[i] =
			struct{ KeyType, ValueType DynamicType }{
				KeyType:   key.DynamicType(interpreter, results),
				ValueType: value.DynamicType(interpreter, results),
			}
	}

	return DictionaryDynamicType{
		EntryTypes: entryTypes,
	}
}

func (v *DictionaryValue) StaticType() StaticType {
	// TODO: store static type in dictionary values
	return nil
}

func (v *DictionaryValue) Copy() Value {
	newKeys := v.Keys.Copy().(*ArrayValue)

	newEntries := NewStringValueOrderedMap()

	v.Entries.Foreach(func(key string, value Value) {
		newEntries.Set(key, value.Copy())
	})

	return &DictionaryValue{
		Keys:                   newKeys,
		Entries:                newEntries,
		DeferredOwner:          v.DeferredOwner,
		DeferredKeys:           v.DeferredKeys,
		DeferredStorageKeyBase: v.DeferredStorageKeyBase,
		prevDeferredKeys:       v.prevDeferredKeys,
		// NOTE: new value has no owner
		Owner:    nil,
		modified: true,
	}
}

func (v *DictionaryValue) GetOwner() *common.Address {
	return v.Owner
}

func (v *DictionaryValue) SetOwner(owner *common.Address) {
	if v.Owner == owner {
		return
	}

	v.Owner = owner

	v.Keys.SetOwner(owner)

	v.Entries.Foreach(func(_ string, value Value) {
		value.SetOwner(owner)
	})
}

func (v *DictionaryValue) IsModified() bool {
	if v.modified {
		return true
	}

	if v.Keys.IsModified() {
		return true
	}

	for pair := v.Entries.Oldest(); pair != nil; pair = pair.Next() {
		value := pair.Value
		if value.IsModified() {
			return true
		}
	}

	return false
}

func (v *DictionaryValue) SetModified(modified bool) {
	v.modified = modified
}

func maybeDestroy(inter *Interpreter, getLocationRange func() LocationRange, value Value) {
	destroyableValue, ok := value.(DestroyableValue)
	if !ok {
		return
	}

	destroyableValue.Destroy(inter, getLocationRange)
}

func (v *DictionaryValue) Destroy(inter *Interpreter, getLocationRange func() LocationRange) {

	for _, keyValue := range v.Keys.Values {
		// Don't use `Entries` here: the value might be deferred and needs to be loaded
		value := v.Get(inter, getLocationRange, keyValue)
		maybeDestroy(inter, getLocationRange, keyValue)
		maybeDestroy(inter, getLocationRange, value)
	}

	writeDeferredKeys(inter, v.DeferredOwner, v.DeferredStorageKeyBase, v.DeferredKeys)
	writeDeferredKeys(inter, v.DeferredOwner, v.DeferredStorageKeyBase, v.prevDeferredKeys)
}

func (v *DictionaryValue) ContainsKey(keyValue Value) BoolValue {
	key := dictionaryKey(keyValue)
	_, ok := v.Entries.Get(key)
	if ok {
		return true
	}
	if v.DeferredKeys != nil {
		_, ok := v.DeferredKeys.Get(key)
		if ok {
			return true
		}
	}
	return false
}

func (v *DictionaryValue) Get(inter *Interpreter, _ func() LocationRange, keyValue Value) Value {
	key := dictionaryKey(keyValue)
	value, ok := v.Entries.Get(key)
	if ok {
		return NewSomeValueOwningNonCopying(value)
	}

	// Is the key a deferred value? If so, load it from storage
	// and keep it as an entry in memory

	if v.DeferredKeys != nil {
		_, ok := v.DeferredKeys.Delete(key)
		if ok {
			storageKey := joinPathElements(v.DeferredStorageKeyBase, key)
			if v.prevDeferredKeys == nil {
				v.prevDeferredKeys = orderedmap.NewStringStructOrderedMap()
			}
			v.prevDeferredKeys.Set(key, struct{}{})

			storedValue := inter.ReadStored(*v.DeferredOwner, storageKey, true)
			v.Entries.Set(key, storedValue.(*SomeValue).Value)

			// NOTE: *not* writing nil to the storage key,
			// as this would result in a loss of the value:
			// the read value is not modified,
			// so it won't be written back

			return storedValue
		}
	}

	return NilValue{}

}

func dictionaryKey(keyValue Value) string {
	hasKeyString, ok := keyValue.(HasKeyString)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return hasKeyString.KeyString()
}

func (v *DictionaryValue) Set(inter *Interpreter, getLocationRange func() LocationRange, keyValue Value, value Value) {
	v.modified = true

	switch typedValue := value.(type) {
	case *SomeValue:
		_ = v.Insert(inter, getLocationRange, keyValue, typedValue.Value)

	case NilValue:
		_ = v.Remove(inter, getLocationRange, keyValue)
		return

	default:
		panic(errors.NewUnreachableError())
	}
}

func (v *DictionaryValue) String() string {
	pairs := make([]struct {
		Key   string
		Value string
	}, len(v.Keys.Values))

	for i, keyValue := range v.Keys.Values {
		key := dictionaryKey(keyValue)
		value, _ := v.Entries.Get(key)

		// Value is potentially deferred,
		// so might be nil

		var valueString string
		if value == nil {
			valueString = "..."
		} else {
			valueString = value.String()
		}

		pairs[i] = struct {
			Key   string
			Value string
		}{
			Key:   keyValue.String(),
			Value: valueString,
		}
	}

	return format.Dictionary(pairs)
}

func (v *DictionaryValue) GetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value {
	switch name {
	case "length":
		return NewIntValueFromInt64(int64(v.Count()))

	// TODO: is returning copies correct?
	case "keys":
		return v.Keys.Copy()

	// TODO: is returning copies correct?
	case "values":
		dictionaryValues := make([]Value, v.Count())
		i := 0
		for _, keyValue := range v.Keys.Values {
			value := v.Get(interpreter, getLocationRange, keyValue).(*SomeValue).Value
			dictionaryValues[i] = value.Copy()
			i++
		}
		return NewArrayValueUnownedNonCopying(dictionaryValues...)

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
		)

	case "containsKey":
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return v.ContainsKey(invocation.Arguments[0])
			},
		)

	}

	return nil
}

func (v *DictionaryValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	// Dictionaries have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v *DictionaryValue) Count() int {
	return v.Keys.Count()
}

// TODO: unset owner?
func (v *DictionaryValue) Remove(inter *Interpreter, getLocationRange func() LocationRange, keyValue Value) OptionalValue {
	v.modified = true

	// Don't use `Entries` here: the value might be deferred and needs to be loaded
	value := v.Get(inter, getLocationRange, keyValue)

	key := dictionaryKey(keyValue)

	// If a resource that was previously deferred is removed from the dictionary,
	// we delete its old key in storage, and then rely on resource semantics
	// to make sure it is stored or destroyed later

	if v.prevDeferredKeys != nil {
		if _, ok := v.prevDeferredKeys.Get(key); ok {
			storageKey := joinPathElements(v.DeferredStorageKeyBase, key)
			inter.writeStored(*v.DeferredOwner, storageKey, NilValue{})
		}
	}

	switch value := value.(type) {
	case *SomeValue:

		v.Entries.Delete(key)

		// TODO: optimize linear scan
		for i, keyValue := range v.Keys.Values {
			if dictionaryKey(keyValue) == key {
				v.Keys.Remove(i)
				return value
			}
		}

		// Should never occur, the key should have been found
		panic(errors.NewUnreachableError())

	case NilValue:
		return value

	default:
		panic(errors.NewUnreachableError())
	}
}

func (v *DictionaryValue) Insert(inter *Interpreter, locationRangeGetter func() LocationRange, keyValue, value Value) OptionalValue {
	v.modified = true

	// Don't use `Entries` here: the value might be deferred and needs to be loaded
	existingValue := v.Get(inter, locationRangeGetter, keyValue)

	key := dictionaryKey(keyValue)

	value.SetOwner(v.Owner)

	// Mark the inserted value itself modified.
	// It might have been stored as a deferred value and loaded,
	// so must be written (potentially as a deferred value again),
	// and would otherwise be ignored by the writeback optimization.

	value.SetModified(true)

	v.Entries.Set(key, value)

	switch existingValue := existingValue.(type) {
	case *SomeValue:
		return existingValue

	case NilValue:
		v.Keys.Append(keyValue)
		return existingValue

	default:
		panic(errors.NewUnreachableError())
	}
}

func writeDeferredKeys(
	inter *Interpreter,
	owner *common.Address,
	storageKeyBase string,
	keys *orderedmap.StringStructOrderedMap,
) {
	if keys == nil {
		return
	}

	for pair := keys.Oldest(); pair != nil; pair = pair.Next() {
		storageKey := joinPathElements(storageKeyBase, pair.Key)
		inter.writeStored(*owner, storageKey, NilValue{})
	}
}

type DictionaryEntryValues struct {
	Key   Value
	Value Value
}

func (v *DictionaryValue) ConformsToDynamicType(
	interpreter *Interpreter,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	dictionaryType, ok := dynamicType.(DictionaryDynamicType)
	if !ok || v.Count() != len(dictionaryType.EntryTypes) {
		return false
	}

	for index, entryKey := range v.Keys.Values {
		entryType := dictionaryType.EntryTypes[index]

		// Check the key
		if !entryKey.ConformsToDynamicType(interpreter, entryType.KeyType, results) {
			return false
		}

		// Check the value. Here it is assumed an imported value can only have
		// static entries, but not deferred keys/values.
		key := dictionaryKey(entryKey)
		entryValue, ok := v.Entries.Get(key)
		if !ok || !entryValue.ConformsToDynamicType(interpreter, entryType.ValueType, results) {
			return false
		}
	}

	return true
}

func (v *DictionaryValue) Equal(other Value, interpreter *Interpreter, loadDeferred bool) bool {
	otherDictionary, ok := other.(*DictionaryValue)
	if !ok {
		return false
	}

	if !v.Keys.Equal(otherDictionary.Keys, interpreter, loadDeferred) {
		return false
	}

	for i, keyValue := range v.Keys.Values {
		otherKeyValue := otherDictionary.Keys.Values[i]

		var value, otherValue Value
		var valueExists, otherValueExists bool

		if loadDeferred {
			value = v.Get(interpreter, nil, keyValue)
			valueExists = true

			otherValue = otherDictionary.Get(interpreter, nil, otherKeyValue)
			otherValueExists = true
		} else {
			value, valueExists = v.Entries.Get(dictionaryKey(keyValue))
			otherValue, otherValueExists = otherDictionary.Entries.Get(dictionaryKey(otherKeyValue))
		}

		if valueExists {
			equatableValue, ok := value.(EquatableValue)
			if !ok || !equatableValue.Equal(otherValue, interpreter, loadDeferred) {
				return false
			}
		} else if otherValueExists {
			return false
		}
	}

	return true
}

// OptionalValue

type OptionalValue interface {
	Value
	isOptionalValue()
}

// NilValue

type NilValue struct{}

func (NilValue) IsValue() {}

func (v NilValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitNilValue(interpreter, v)
}

func (NilValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return NilDynamicType{}
}

func (NilValue) StaticType() StaticType {
	return OptionalStaticType{
		Type: PrimitiveStaticTypeNever,
	}
}

func (NilValue) isOptionalValue() {}

func (v NilValue) Copy() Value {
	return v
}

func (NilValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (NilValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (NilValue) IsModified() bool {
	return false
}

func (NilValue) SetModified(_ bool) {
	// NO-OP
}

func (v NilValue) Destroy(_ *Interpreter, _ func() LocationRange) {
	// NO-OP
}

func (NilValue) String() string {
	return format.Nil
}

var nilValueMapFunction = NewHostFunctionValue(
	func(invocation Invocation) Value {
		return NilValue{}
	},
)

func (v NilValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "map":
		return nilValueMapFunction
	}

	return nil
}

func (NilValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v NilValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(NilDynamicType)
	return ok
}

func (v NilValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	_, ok := other.(NilValue)
	return ok
}

// SomeValue

type SomeValue struct {
	Value Value
	Owner *common.Address
}

func NewSomeValueOwningNonCopying(value Value) *SomeValue {
	return &SomeValue{
		Value: value,
		Owner: value.GetOwner(),
	}
}

func (*SomeValue) IsValue() {}

func (v *SomeValue) Accept(interpreter *Interpreter, visitor Visitor) {
	descend := visitor.VisitSomeValue(interpreter, v)
	if !descend {
		return
	}
	v.Value.Accept(interpreter, visitor)
}

func (v *SomeValue) DynamicType(interpreter *Interpreter, results DynamicTypeResults) DynamicType {
	innerType := v.Value.DynamicType(interpreter, results)
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

func (v *SomeValue) Copy() Value {
	return &SomeValue{
		Value: v.Value.Copy(),
		// NOTE: new value has no owner
		Owner: nil,
	}
}

func (v *SomeValue) GetOwner() *common.Address {
	return v.Owner
}

func (v *SomeValue) SetOwner(owner *common.Address) {
	if v.Owner == owner {
		return
	}

	v.Owner = owner

	v.Value.SetOwner(owner)
}

func (v *SomeValue) IsModified() bool {
	return v.Value.IsModified()
}

func (v *SomeValue) SetModified(modified bool) {
	v.Value.SetModified(modified)
}

func (v *SomeValue) Destroy(interpreter *Interpreter, getLocationRange func() LocationRange) {
	maybeDestroy(interpreter, getLocationRange, v.Value)
}

func (v *SomeValue) String() string {
	return v.Value.String()
}

func (v *SomeValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
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

				newValue := transformFunction.Invoke(transformInvocation)

				return NewSomeValueOwningNonCopying(newValue)
			},
		)
	}

	return nil
}

func (*SomeValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v SomeValue) ConformsToDynamicType(
	interpreter *Interpreter,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {
	someType, ok := dynamicType.(SomeDynamicType)
	return ok && v.Value.ConformsToDynamicType(interpreter, someType.InnerType, results)
}

func (v *SomeValue) Equal(other Value, interpreter *Interpreter, loadDeferred bool) bool {
	otherSome, ok := other.(*SomeValue)
	if !ok {
		return false
	}

	equatableValue, ok := v.Value.(EquatableValue)
	if !ok {
		return false
	}

	return equatableValue.Equal(otherSome.Value, interpreter, loadDeferred)
}

// StorageReferenceValue

type StorageReferenceValue struct {
	Authorized           bool
	TargetStorageAddress common.Address
	TargetKey            string
	BorrowedType         sema.Type
}

func (*StorageReferenceValue) IsValue() {}

func (v *StorageReferenceValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitStorageReferenceValue(interpreter, v)
}

func (v *StorageReferenceValue) String() string {
	return "StorageReference()"
}

func (v *StorageReferenceValue) DynamicType(interpreter *Interpreter, results DynamicTypeResults) DynamicType {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{})
	}

	innerType := (*referencedValue).DynamicType(interpreter, results)

	return StorageReferenceDynamicType{
		authorized: v.Authorized,
		innerType:  innerType,
	}
}

func (v *StorageReferenceValue) StaticType() StaticType {
	// TODO:
	return nil
}

func (v *StorageReferenceValue) Copy() Value {
	return &StorageReferenceValue{
		Authorized:           v.Authorized,
		TargetStorageAddress: v.TargetStorageAddress,
		TargetKey:            v.TargetKey,
		BorrowedType:         v.BorrowedType,
	}
}

func (v *StorageReferenceValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (v *StorageReferenceValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (*StorageReferenceValue) IsModified() bool {
	return false
}

func (*StorageReferenceValue) SetModified(_ bool) {
	// NO-OP
}

func (v *StorageReferenceValue) ReferencedValue(interpreter *Interpreter) *Value {
	switch referenced := interpreter.ReadStored(v.TargetStorageAddress, v.TargetKey, false).(type) {
	case *SomeValue:
		value := referenced.Value

		if v.BorrowedType != nil {
			dynamicTypeResults := DynamicTypeResults{}
			dynamicType := value.DynamicType(interpreter, dynamicTypeResults)
			if !IsSubType(dynamicType, v.BorrowedType) {
				return nil
			}
		}

		return &value

	case NilValue:
		return nil

	default:
		panic(errors.NewUnreachableError())
	}
}

func (v *StorageReferenceValue) GetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	return interpreter.getMember(*referencedValue, getLocationRange, name)
}

func (v *StorageReferenceValue) SetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string, value Value) {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	interpreter.setMember(*referencedValue, getLocationRange, name, value)
}

func (v *StorageReferenceValue) Get(interpreter *Interpreter, getLocationRange func() LocationRange, key Value) Value {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	return (*referencedValue).(ValueIndexableValue).
		Get(interpreter, getLocationRange, key)
}

func (v *StorageReferenceValue) Set(interpreter *Interpreter, getLocationRange func() LocationRange, key Value, value Value) {
	referencedValue := v.ReferencedValue(interpreter)
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	(*referencedValue).(ValueIndexableValue).
		Set(interpreter, getLocationRange, key, value)
}

func (v *StorageReferenceValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherReference, ok := other.(*StorageReferenceValue)
	if !ok {
		return false
	}

	return v.TargetStorageAddress == otherReference.TargetStorageAddress &&
		v.TargetKey == otherReference.TargetKey &&
		v.Authorized == otherReference.Authorized
}

func (v *StorageReferenceValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(StorageReferenceDynamicType)
	return ok
}

// EphemeralReferenceValue

type EphemeralReferenceValue struct {
	Authorized bool
	Value      Value
}

func (*EphemeralReferenceValue) IsValue() {}

func (v *EphemeralReferenceValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitEphemeralReferenceValue(interpreter, v)
}

func (v *EphemeralReferenceValue) String() string {
	return v.Value.String()
}

func (v *EphemeralReferenceValue) DynamicType(interpreter *Interpreter, results DynamicTypeResults) DynamicType {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{})
	}

	if result, ok := results[v]; ok {
		return result
	}

	results[v] = nil

	innerType := (*referencedValue).DynamicType(interpreter, results)

	result := EphemeralReferenceDynamicType{
		authorized: v.Authorized,
		innerType:  innerType,
	}

	results[v] = result

	return result
}

func (v *EphemeralReferenceValue) StaticType() StaticType {
	// TODO:
	return nil
}

func (v *EphemeralReferenceValue) Copy() Value {
	return v
}

func (v *EphemeralReferenceValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (v *EphemeralReferenceValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (*EphemeralReferenceValue) IsModified() bool {
	return false
}

func (*EphemeralReferenceValue) SetModified(_ bool) {
	// NO-OP
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

func (v *EphemeralReferenceValue) GetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string) Value {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	return interpreter.getMember(*referencedValue, getLocationRange, name)
}

func (v *EphemeralReferenceValue) SetMember(interpreter *Interpreter, getLocationRange func() LocationRange, name string, value Value) {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	interpreter.setMember(*referencedValue, getLocationRange, name, value)
}

func (v *EphemeralReferenceValue) Get(interpreter *Interpreter, getLocationRange func() LocationRange, key Value) Value {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	return (*referencedValue).(ValueIndexableValue).
		Get(interpreter, getLocationRange, key)
}

func (v *EphemeralReferenceValue) Set(interpreter *Interpreter, getLocationRange func() LocationRange, key Value, value Value) {
	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		panic(DereferenceError{
			LocationRange: getLocationRange(),
		})
	}

	(*referencedValue).(ValueIndexableValue).
		Set(interpreter, getLocationRange, key, value)
}

func (v *EphemeralReferenceValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherReference, ok := other.(*EphemeralReferenceValue)
	if !ok {
		return false
	}

	return v.Value == otherReference.Value &&
		v.Authorized == otherReference.Authorized
}

func (v *EphemeralReferenceValue) ConformsToDynamicType(
	interpreter *Interpreter,
	dynamicType DynamicType,
	results TypeConformanceResults,
) bool {

	refType, ok := dynamicType.(EphemeralReferenceDynamicType)
	if !ok {
		return false
	}

	referencedValue := v.ReferencedValue()
	if referencedValue == nil {
		return false
	}

	valueTypePair := valueDynamicTypePair{
		value:       v,
		dynamicType: dynamicType,
	}

	if result, contains := results[valueTypePair]; contains {
		return result
	}

	// It is safe to set 'true' here even this is not checked yet, because the final result
	// doesn't depend on this. It depends on the rest of values of the object tree.
	results[valueTypePair] = true

	result := (*referencedValue).ConformsToDynamicType(interpreter, refType.InnerType(), results)

	results[valueTypePair] = result

	return result
}

// AddressValue

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
	// TODO: https://github.com/dapperlabs/flow-go/issues/2141
	result := AddressValue{}
	if intValue, ok := value.(IntValue); ok {
		bigEndianBytes := intValue.BigInt.Bytes()
		copy(
			result[common.AddressLength-len(bigEndianBytes):common.AddressLength],
			bigEndianBytes,
		)
	} else {
		binary.BigEndian.PutUint64(
			result[common.AddressLength-8:common.AddressLength],
			uint64(value.(NumberValue).ToInt()),
		)
	}
	return result
}

func (AddressValue) IsValue() {}

func (v AddressValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitAddressValue(interpreter, v)
}

func (AddressValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return AddressDynamicType{}
}

func (AddressValue) StaticType() StaticType {
	return PrimitiveStaticTypeAddress
}

func (v AddressValue) Copy() Value {
	return v
}

func (v AddressValue) KeyString() string {
	return common.Address(v).ShortHexWithPrefix()
}

func (v AddressValue) String() string {
	return format.Address(common.Address(v))
}

func (AddressValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (AddressValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (AddressValue) IsModified() bool {
	return false
}

func (AddressValue) SetModified(_ bool) {
	// NO-OP
}

func (v AddressValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherAddress, ok := other.(AddressValue)
	if !ok {
		return false
	}
	return v == otherAddress
}

func (v AddressValue) Hex() string {
	return v.ToAddress().Hex()
}

func (v AddressValue) ToAddress() common.Address {
	return common.Address(v)
}

func (v AddressValue) GetMember(_ *Interpreter, _ func() LocationRange, name string) Value {
	switch name {

	case sema.ToStringFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				return NewStringValue(v.String())
			},
		)

	case sema.AddressTypeToBytesFunctionName:
		return NewHostFunctionValue(
			func(invocation Invocation) Value {
				bytes := common.Address(v)
				return ByteSliceToByteArrayValue(bytes[:])
			},
		)
	}

	return nil
}

func (AddressValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v AddressValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(AddressDynamicType)
	return ok
}

// NewAuthAccountValue constructs an auth account value.
func NewAuthAccountValue(
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func() UInt64Value,
	addPublicKeyFunction FunctionValue,
	removePublicKeyFunction FunctionValue,
	contracts *CompositeValue,
	keys *CompositeValue,
) *CompositeValue {

	fields := NewStringValueOrderedMap()
	fields.Set(sema.AuthAccountAddressField, address)
	fields.Set(sema.AuthAccountAddPublicKeyField, addPublicKeyFunction)
	fields.Set(sema.AuthAccountRemovePublicKeyField, removePublicKeyFunction)
	fields.Set(sema.AuthAccountGetCapabilityField, accountGetCapabilityFunction(address))
	fields.Set(sema.AuthAccountContractsField, contracts)
	fields.Set(sema.AuthAccountKeysField, keys)

	// Computed fields
	computedFields := NewStringComputedFieldOrderedMap()

	computedFields.Set(sema.AuthAccountBalanceField, func(*Interpreter) Value {
		return accountBalanceGet()
	})

	computedFields.Set(sema.AuthAccountAvailableBalanceField, func(*Interpreter) Value {
		return accountAvailableBalanceGet()
	})

	computedFields.Set(sema.AuthAccountStorageUsedField, func(inter *Interpreter) Value {
		return storageUsedGet(inter)
	})

	computedFields.Set(sema.AuthAccountStorageCapacityField, func(*Interpreter) Value {
		return storageCapacityGet()
	})

	computedFields.Set(sema.AuthAccountLoadField, func(inter *Interpreter) Value {
		return inter.authAccountLoadFunction(address)
	})

	computedFields.Set(sema.AuthAccountCopyField, func(inter *Interpreter) Value {
		return inter.authAccountCopyFunction(address)
	})

	computedFields.Set(sema.AuthAccountSaveField, func(inter *Interpreter) Value {
		return inter.authAccountSaveFunction(address)
	})

	computedFields.Set(sema.AuthAccountBorrowField, func(inter *Interpreter) Value {
		return inter.authAccountBorrowFunction(address)
	})

	computedFields.Set(sema.AuthAccountLinkField, func(inter *Interpreter) Value {
		return inter.authAccountLinkFunction(address)
	})

	computedFields.Set(sema.AuthAccountUnlinkField, func(inter *Interpreter) Value {
		return inter.authAccountUnlinkFunction(address)
	})

	computedFields.Set(sema.AuthAccountGetLinkTargetField, func(inter *Interpreter) Value {
		return inter.accountGetLinkTargetFunction(address)
	})

	stringer := func() string {
		return fmt.Sprintf("AuthAccount(%s)", address)
	}

	return &CompositeValue{
		QualifiedIdentifier: sema.AuthAccountType.QualifiedIdentifier(),
		Kind:                sema.AuthAccountType.Kind,
		Fields:              fields,
		ComputedFields:      computedFields,
		stringer:            stringer,
	}
}

func accountGetCapabilityFunction(
	addressValue AddressValue,
) HostFunctionValue {

	return NewHostFunctionValue(
		func(invocation Invocation) Value {

			path := invocation.Arguments[0].(PathValue)

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

			return CapabilityValue{
				Address:    addressValue,
				Path:       path,
				BorrowType: borrowStaticType,
			}
		},
	)
}

// NewPublicAccountValue constructs a public account value.
func NewPublicAccountValue(
	address AddressValue,
	accountBalanceGet func() UFix64Value,
	accountAvailableBalanceGet func() UFix64Value,
	storageUsedGet func(interpreter *Interpreter) UInt64Value,
	storageCapacityGet func() UInt64Value,
	keys *CompositeValue,
) *CompositeValue {

	fields := NewStringValueOrderedMap()
	fields.Set(sema.PublicAccountAddressField, address)
	fields.Set(sema.PublicAccountGetCapabilityField, accountGetCapabilityFunction(address))
	fields.Set(sema.PublicAccountKeysField, keys)

	// Computed fields
	computedFields := NewStringComputedFieldOrderedMap()

	computedFields.Set(sema.PublicAccountBalanceField, func(*Interpreter) Value {
		return accountBalanceGet()
	})

	computedFields.Set(sema.PublicAccountAvailableBalanceField, func(*Interpreter) Value {
		return accountAvailableBalanceGet()
	})

	computedFields.Set(sema.PublicAccountStorageUsedField, func(inter *Interpreter) Value {
		return storageUsedGet(inter)
	})

	computedFields.Set(sema.PublicAccountStorageCapacityField, func(*Interpreter) Value {
		return storageCapacityGet()
	})

	computedFields.Set(sema.PublicAccountGetTargetLinkField, func(inter *Interpreter) Value {
		return inter.accountGetLinkTargetFunction(address)
	})

	// Stringer function
	stringer := func() string {
		return fmt.Sprintf("PublicAccount(%s)", address)
	}

	return &CompositeValue{
		QualifiedIdentifier: sema.PublicAccountType.QualifiedIdentifier(),
		Kind:                sema.PublicAccountType.Kind,
		Fields:              fields,
		ComputedFields:      computedFields,
		stringer:            stringer,
	}
}

// PathValue

type PathValue struct {
	Domain     common.PathDomain
	Identifier string
}

func (PathValue) IsValue() {}

func (v PathValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitPathValue(interpreter, v)
}

func (v PathValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	switch v.Domain {
	case common.PathDomainStorage:
		return StoragePathDynamicType{}
	case common.PathDomainPublic:
		return PublicPathDynamicType{}
	case common.PathDomainPrivate:
		return PrivatePathDynamicType{}
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

func (v PathValue) Copy() Value {
	return v
}

func (PathValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (PathValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (PathValue) IsModified() bool {
	return false
}

func (PathValue) SetModified(_ bool) {
	// NO-OP
}

func (v PathValue) Destroy(_ *Interpreter, _ func() LocationRange) {
	// NO-OP
}

func (v PathValue) String() string {
	return format.Path(
		v.Domain.Identifier(),
		v.Identifier,
	)
}

func (v PathValue) KeyString() string {
	return fmt.Sprintf(
		"/%s/%s",
		v.Domain,
		v.Identifier,
	)
}

func (v PathValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
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

func (v PathValue) Equal(other Value, _ *Interpreter, _ bool) bool {
	otherPath, ok := other.(PathValue)
	if !ok {
		return false
	}

	return otherPath.Identifier == v.Identifier &&
		otherPath.Domain == v.Domain
}

// CapabilityValue

type CapabilityValue struct {
	Address    AddressValue
	Path       PathValue
	BorrowType StaticType
}

func (CapabilityValue) IsValue() {}

func (v CapabilityValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitCapabilityValue(interpreter, v)
}

func (v CapabilityValue) DynamicType(inter *Interpreter, _ DynamicTypeResults) DynamicType {
	var borrowType *sema.ReferenceType
	if v.BorrowType != nil {
		borrowType = inter.ConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
	}

	return CapabilityDynamicType{
		BorrowType: borrowType,
	}
}

func (v CapabilityValue) StaticType() StaticType {
	return CapabilityStaticType{
		BorrowType: v.BorrowType,
	}
}

func (v CapabilityValue) Copy() Value {
	return v
}

func (CapabilityValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (CapabilityValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (CapabilityValue) IsModified() bool {
	return false
}

func (CapabilityValue) SetModified(_ bool) {
	// NO-OP
}

func (v CapabilityValue) Destroy(_ *Interpreter, _ func() LocationRange) {
	// NO-OP
}

func (v CapabilityValue) String() string {
	var borrowType string
	if v.BorrowType != nil {
		borrowType = v.BorrowType.String()
	}
	return format.Capability(
		borrowType,
		v.Address.String(),
		v.Path.String(),
	)
}

func (v CapabilityValue) GetMember(inter *Interpreter, _ func() LocationRange, name string) Value {
	switch name {
	case "borrow":
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			borrowType = inter.ConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return inter.capabilityBorrowFunction(v.Address, v.Path, borrowType)

	case "check":
		var borrowType *sema.ReferenceType
		if v.BorrowType != nil {
			borrowType = inter.ConvertStaticToSemaType(v.BorrowType).(*sema.ReferenceType)
		}
		return inter.capabilityCheckFunction(v.Address, v.Path, borrowType)

	case "address":
		return v.Address
	}

	return nil
}

func (CapabilityValue) SetMember(_ *Interpreter, _ func() LocationRange, _ string, _ Value) {
	panic(errors.NewUnreachableError())
}

func (v CapabilityValue) ConformsToDynamicType(_ *Interpreter, dynamicType DynamicType, _ TypeConformanceResults) bool {
	_, ok := dynamicType.(CapabilityDynamicType)
	return ok
}

func (v CapabilityValue) Equal(other Value, interpreter *Interpreter, loadDeferred bool) bool {
	otherCapability, ok := other.(CapabilityValue)
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

	return otherCapability.Address.Equal(v.Address, interpreter, loadDeferred) &&
		otherCapability.Path.Equal(v.Path, interpreter, loadDeferred)
}

// LinkValue

type LinkValue struct {
	TargetPath PathValue
	Type       StaticType
}

func (LinkValue) IsValue() {}

func (v LinkValue) Accept(interpreter *Interpreter, visitor Visitor) {
	visitor.VisitLinkValue(interpreter, v)
}

func (LinkValue) DynamicType(_ *Interpreter, _ DynamicTypeResults) DynamicType {
	return nil
}

func (LinkValue) StaticType() StaticType {
	return nil
}

func (v LinkValue) Copy() Value {
	return v
}

func (LinkValue) GetOwner() *common.Address {
	// value is never owned
	return nil
}

func (LinkValue) SetOwner(_ *common.Address) {
	// NO-OP: value cannot be owned
}

func (LinkValue) IsModified() bool {
	return false
}

func (LinkValue) SetModified(_ bool) {
	// NO-OP
}

func (v LinkValue) Destroy(_ *Interpreter, _ func() LocationRange) {
	// NO-OP
}

func (v LinkValue) String() string {
	return format.Link(
		v.Type.String(),
		v.TargetPath.String(),
	)
}

func (v LinkValue) ConformsToDynamicType(_ *Interpreter, _ DynamicType, _ TypeConformanceResults) bool {
	// There is no dynamic type for links,
	// as they are not first-class values in programs,
	// but only stored
	return false
}

func (v LinkValue) Equal(other Value, interpreter *Interpreter, loadDeferred bool) bool {
	otherLink, ok := other.(LinkValue)
	if !ok {
		return false
	}

	return otherLink.TargetPath.Equal(v.TargetPath, interpreter, loadDeferred) &&
		otherLink.Type.Equal(v.Type)
}

// NewAccountKeyValue constructs an AccountKey value.
func NewAccountKeyValue(
	keyIndex IntValue,
	publicKey *CompositeValue,
	hashAlgo *CompositeValue,
	weight UFix64Value,
	isRevoked BoolValue,
) *CompositeValue {
	fields := NewStringValueOrderedMap()
	fields.Set(sema.AccountKeyKeyIndexField, keyIndex)
	fields.Set(sema.AccountKeyPublicKeyField, publicKey)
	fields.Set(sema.AccountKeyHashAlgoField, hashAlgo)
	fields.Set(sema.AccountKeyWeightField, weight)
	fields.Set(sema.AccountKeyIsRevokedField, isRevoked)

	return &CompositeValue{
		QualifiedIdentifier: sema.AccountKeyType.QualifiedIdentifier(),
		Kind:                sema.AccountKeyType.Kind,
		Fields:              fields,
	}
}

// NewPublicKeyValue constructs a PublicKey value.
func NewPublicKeyValue(publicKey *ArrayValue, signAlgo *CompositeValue) *CompositeValue {

	fields := NewStringValueOrderedMap()
	fields.Set(sema.PublicKeyPublicKeyField, publicKey)
	fields.Set(sema.PublicKeySignAlgoField, signAlgo)

	return &CompositeValue{
		QualifiedIdentifier: sema.PublicKeyType.QualifiedIdentifier(),
		Kind:                sema.PublicKeyType.Kind,
		Fields:              fields,
	}
}

// NewAuthAccountKeysValue constructs a AuthAccount.Keys value.
func NewAuthAccountKeysValue(addFunction FunctionValue, getFunction FunctionValue, revokeFunction FunctionValue) *CompositeValue {
	fields := NewStringValueOrderedMap()
	fields.Set(sema.AccountKeysAddFunctionName, addFunction)
	fields.Set(sema.AccountKeysGetFunctionName, getFunction)
	fields.Set(sema.AccountKeysRevokeFunctionName, revokeFunction)

	return &CompositeValue{
		QualifiedIdentifier: sema.AuthAccountKeysType.QualifiedIdentifier(),
		Kind:                sema.AuthAccountKeysType.Kind,
		Fields:              fields,
	}
}

// NewPublicAccountKeysValue constructs a PublicAccount.Keys value.
func NewPublicAccountKeysValue(getFunction FunctionValue) *CompositeValue {
	fields := NewStringValueOrderedMap()
	fields.Set(sema.AccountKeysGetFunctionName, getFunction)

	return &CompositeValue{
		QualifiedIdentifier: sema.PublicAccountKeysType.QualifiedIdentifier(),
		Kind:                sema.PublicAccountKeysType.Kind,
		Fields:              fields,
	}
}

func NewCryptoAlgorithmEnumCaseValue(enumType *sema.CompositeType, rawValue uint8) *CompositeValue {
	fields := NewStringValueOrderedMap()
	fields.Set(sema.EnumRawValueFieldName, UInt8Value(rawValue))

	return &CompositeValue{
		QualifiedIdentifier: enumType.QualifiedIdentifier(),
		Kind:                enumType.Kind,
		Fields:              fields,
	}
}
