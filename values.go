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

package cadence

import (
	"fmt"
	"math/big"

	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/interpreter"
)

// Value

type Value interface {
	isValue()
	Type() Type
	ToGoValue() interface{}
	ToRuntimeValue() runtime.Value
}

// Void

type Void struct{}

func NewVoid() Void {
	return Void{}
}

func (Void) isValue() {}

func (Void) Type() Type {
	return VoidType{}
}

func (Void) ToGoValue() interface{} {
	return nil
}

func (Void) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.VoidValue{}}
}

// Optional

type Optional struct {
	Value Value
}

func NewOptional(value Value) Optional {
	return Optional{Value: value}
}

func (Optional) isValue() {}

func (Optional) Type() Type {
	return nil
}

func (v Optional) ToGoValue() interface{} {
	if v.Value == nil {
		return nil
	}

	value := v.Value.ToGoValue()

	return value
}

func (v Optional) ToRuntimeValue() runtime.Value {
	value := &interpreter.SomeValue{}

	if v.Value != nil {
		value.Value = v.Value.ToRuntimeValue().Value
	}

	return runtime.Value{Value: value}
}

// Bool

type Bool bool

func NewBool(b bool) Bool {
	return Bool(b)
}

func (Bool) isValue() {}

func (Bool) Type() Type {
	return BoolType{}
}

func (v Bool) ToGoValue() interface{} {
	return bool(v)
}

func (v Bool) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.BoolValue(v)}
}

// String

type String string

func NewString(s string) String {
	return String(s)
}

func (String) isValue() {}

func (String) Type() Type {
	return StringType{}
}

func (v String) ToGoValue() interface{} {
	return string(v)
}

func (v String) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewStringValue(string(v))}
}

// Bytes

type Bytes []byte

func NewBytes(b []byte) Bytes {
	return b
}

func (Bytes) isValue() {}

func (Bytes) Type() Type {
	return BytesType{}
}

func (v Bytes) ToGoValue() interface{} {
	return []byte(v)
}

func (v Bytes) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.ByteSliceToByteArrayValue(v)}
}

// Address

const AddressLength = 20

type Address [AddressLength]byte

func NewAddress(b [AddressLength]byte) Address {
	return b
}

func NewAddressFromBytes(b []byte) Address {
	var a Address
	copy(a[:], b)
	return a
}

func (Address) isValue() {}

func (Address) Type() Type {
	return AddressType{}
}

func (v Address) ToGoValue() interface{} {
	return [AddressLength]byte(v)
}

func (v Address) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewAddressValueFromBytes(v.Bytes())}
}

func (v Address) Bytes() []byte {
	return v[:]
}

func (v Address) String() string {
	return v.Hex()
}

func (v Address) Hex() string {
	return fmt.Sprintf("%x", [AddressLength]byte(v))
}

func BytesToAddress(b []byte) Address {
	var a Address
	copy(a[AddressLength-len(b):AddressLength], b)
	return a
}

// Int

type Int struct {
	Value *big.Int
}

func NewInt(i int) Int {
	return Int{big.NewInt(int64(i))}
}

func NewIntFromBig(i *big.Int) Int {
	return Int{i}
}

func (Int) isValue() {}

func (Int) Type() Type {
	return IntType{}
}

func (v Int) ToGoValue() interface{} {
	return v.Big()
}

func (v Int) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewIntValueFromBigInt(v.Value)}
}

func (v Int) Int() int {
	return int(v.Value.Int64())
}

func (v Int) Big() *big.Int {
	return v.Value
}

// Int8

type Int8 int8

func NewInt8(v int8) Int8 {
	return Int8(v)
}

func (Int8) isValue() {}

func (v Int8) ToGoValue() interface{} {
	return int8(v)
}

func (v Int8) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Int8Value(v)}
}

func (Int8) Type() Type {
	return Int8Type{}
}

// Int16

type Int16 int16

func NewInt16(v int16) Int16 {
	return Int16(v)
}

func (Int16) isValue() {}

func (Int16) Type() Type {
	return Int16Type{}
}

func (v Int16) ToGoValue() interface{} {
	return int16(v)
}

func (v Int16) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Int16Value(v)}
}

// Int32

type Int32 int32

func NewInt32(v int32) Int32 {
	return Int32(v)
}

func (Int32) isValue() {}

func (Int32) Type() Type {
	return Int32Type{}
}

func (v Int32) ToGoValue() interface{} {
	return int32(v)
}

func (v Int32) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Int32Value(v)}
}

// Int64

type Int64 int64

func NewInt64(i int64) Int64 {
	return Int64(i)
}

func (Int64) isValue() {}

func (Int64) Type() Type {
	return Int64Type{}
}

func (v Int64) ToGoValue() interface{} {
	return int64(v)
}

func (v Int64) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Int64Value(v)}
}

// Int128

type Int128 struct {
	Value *big.Int
}

func NewInt128(i int) Int128 {
	return Int128{big.NewInt(int64(i))}
}

func NewInt128FromBig(i *big.Int) Int128 {
	// TODO: check range?
	return Int128{i}
}

func (Int128) isValue() {}

func (Int128) Type() Type {
	return Int128Type{}
}

func (v Int128) ToGoValue() interface{} {
	return v.Big()
}

func (v Int128) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewInt128ValueFromBigInt(v.Value)}
}

func (v Int128) Int() int {
	return int(v.Value.Int64())
}

func (v Int128) Big() *big.Int {
	return v.Value
}

// Int256

type Int256 struct {
	Value *big.Int
}

func NewInt256(i int) Int256 {
	return Int256{big.NewInt(int64(i))}
}

func NewInt256FromBig(i *big.Int) Int256 {
	// TODO: check range?
	return Int256{i}
}

func (Int256) isValue() {}

func (Int256) Type() Type {
	return Int256Type{}
}

func (v Int256) ToGoValue() interface{} {
	return v.Big()
}

func (v Int256) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewInt256ValueFromBigInt(v.Value)}
}

func (v Int256) Int() int {
	return int(v.Value.Int64())
}

func (v Int256) Big() *big.Int {
	return v.Value
}

// UInt

type UInt struct {
	Value *big.Int
}

func NewUInt(i uint) UInt {
	return UInt{big.NewInt(int64(i))}
}

func NewUIntFromBig(i *big.Int) UInt {
	if i.Sign() < 0 {
		panic("negative input")
	}
	return UInt{i}
}

func (UInt) isValue() {}

func (UInt) Type() Type {
	return UIntType{}
}

func (v UInt) ToGoValue() interface{} {
	return v.Big()
}

func (v UInt) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewUIntValueFromBigInt(v.Value)}
}

func (v UInt) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt) Big() *big.Int {
	return v.Value
}

// UInt8

type UInt8 uint8

func NewUInt8(v uint8) UInt8 {
	return UInt8(v)
}

func (UInt8) isValue() {}

func (UInt8) Type() Type {
	return UInt8Type{}
}

func (v UInt8) ToGoValue() interface{} {
	return uint8(v)
}

func (v UInt8) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.UInt8Value(v)}
}

// UInt16

type UInt16 uint16

func NewUInt16(v uint16) UInt16 {
	return UInt16(v)
}

func (UInt16) isValue() {}

func (UInt16) Type() Type {
	return UInt16Type{}
}

func (v UInt16) ToGoValue() interface{} {
	return uint16(v)
}

func (v UInt16) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.UInt16Value(v)}
}

// UInt32

type UInt32 uint32

func NewUInt32(v uint32) UInt32 {
	return UInt32(v)
}

func (UInt32) isValue() {}

func (UInt32) Type() Type {
	return UInt32Type{}
}

func (v UInt32) ToGoValue() interface{} {
	return uint32(v)
}

func (v UInt32) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.UInt32Value(v)}
}

// UInt64

type UInt64 uint64

func NewUInt64(v uint64) UInt64 {
	return UInt64(v)
}

func (UInt64) isValue() {}

func (UInt64) Type() Type {
	return UInt64Type{}
}

func (v UInt64) ToGoValue() interface{} {
	return uint64(v)
}

func (v UInt64) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.UInt64Value(v)}
}

// UInt128

type UInt128 struct {
	Value *big.Int
}

func NewUInt128(i uint) UInt128 {
	return UInt128{big.NewInt(int64(i))}
}

func NewUInt128FromBig(i *big.Int) UInt128 {
	// TODO: check range?
	if i.Sign() < 0 {
		panic("negative input")
	}
	return UInt128{i}
}

func (UInt128) isValue() {}

func (UInt128) Type() Type {
	return UInt128Type{}
}

func (v UInt128) ToGoValue() interface{} {
	return v.Big()
}

func (v UInt128) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewUInt128ValueFromBigInt(v.Value)}
}

func (v UInt128) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt128) Big() *big.Int {
	return v.Value
}

// UInt256

type UInt256 struct {
	Value *big.Int
}

func NewUInt256(i uint) UInt256 {
	return UInt256{big.NewInt(int64(i))}
}

func NewUInt256FromBig(i *big.Int) UInt256 {
	// TODO: check range?
	if i.Sign() < 0 {
		panic("negative input")
	}
	return UInt256{i}
}

func (UInt256) isValue() {}

func (UInt256) Type() Type {
	return UInt256Type{}
}

func (v UInt256) ToGoValue() interface{} {
	return v.Big()
}

func (v UInt256) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.NewUInt256ValueFromBigInt(v.Value)}
}

func (v UInt256) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt256) Big() *big.Int {
	return v.Value
}

// Word8

type Word8 uint8

func NewWord8(v uint8) Word8 {
	return Word8(v)
}

func (Word8) isValue() {}

func (Word8) Type() Type {
	return Word8Type{}
}

func (v Word8) ToGoValue() interface{} {
	return uint8(v)
}

func (v Word8) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Word8Value(v)}
}

// Word16

type Word16 uint16

func NewWord16(v uint16) Word16 {
	return Word16(v)
}

func (Word16) isValue() {}

func (Word16) Type() Type {
	return Word16Type{}
}

func (v Word16) ToGoValue() interface{} {
	return uint16(v)
}

func (v Word16) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Word16Value(v)}
}

// Word32

type Word32 uint32

func NewWord32(v uint32) Word32 {
	return Word32(v)
}

func (Word32) isValue() {}

func (Word32) Type() Type {
	return Word32Type{}
}

func (v Word32) ToGoValue() interface{} {
	return uint32(v)
}

func (v Word32) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Word32Value(v)}
}

// Word64

type Word64 uint64

func NewWord64(v uint64) Word64 {
	return Word64(v)
}

func (Word64) isValue() {}

func (Word64) Type() Type {
	return Word64Type{}
}

func (v Word64) ToGoValue() interface{} {
	return uint64(v)
}

func (v Word64) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Word64Value(v)}
}

// Fix64

type Fix64 int64

func NewFix64(i int64) Fix64 {
	return Fix64(i)
}

func (Fix64) isValue() {}

func (Fix64) Type() Type {
	return Fix64Type{}
}

func (v Fix64) ToGoValue() interface{} {
	return int64(v)
}

func (v Fix64) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.Fix64Value(v)}
}

// UFix64

type UFix64 uint64

func NewUFix64(i uint64) UFix64 {
	return UFix64(i)
}

func (UFix64) isValue() {}

func (UFix64) Type() Type {
	return UFix64Type{}
}

func (v UFix64) ToGoValue() interface{} {
	return uint64(v)
}

func (v UFix64) ToRuntimeValue() runtime.Value {
	return runtime.Value{Value: interpreter.UFix64Value(v)}
}

// Array

type Array struct {
	typ    Type
	Values []Value
}

func NewArray(values []Value) Array {
	return Array{Values: values}
}

func (Array) isValue() {}

func (v Array) Type() Type {
	return v.typ
}

func (v Array) ToGoValue() interface{} {
	ret := make([]interface{}, len(v.Values))

	for i, e := range v.Values {
		ret[i] = e.ToGoValue()
	}

	return ret
}

func (v Array) ToRuntimeValue() runtime.Value {
	values := make([]interpreter.Value, len(v.Values))

	for i, e := range v.Values {
		values[i] = e.ToRuntimeValue().Value
	}

	return runtime.Value{Value: interpreter.NewArrayValueUnownedNonCopying(values...)}
}

// Dictionary

type Dictionary struct {
	typ   Type
	Pairs []KeyValuePair
}

func NewDictionary(pairs []KeyValuePair) Dictionary {
	return Dictionary{Pairs: pairs}
}

func (Dictionary) isValue() {}

func (v Dictionary) Type() Type {
	return v.typ
}

func (v Dictionary) ToGoValue() interface{} {
	ret := map[interface{}]interface{}{}

	for _, p := range v.Pairs {
		ret[p.Key.ToGoValue()] = p.Value.ToGoValue()
	}

	return ret
}

func (v Dictionary) ToRuntimeValue() runtime.Value {
	keysAndValues := make([]interpreter.Value, len(v.Pairs)*2)

	for i, pair := range v.Pairs {
		keysAndValues[i*2] = pair.Key.ToRuntimeValue().Value
		keysAndValues[i*2+1] = pair.Value.ToRuntimeValue().Value
	}

	return runtime.Value{Value: interpreter.NewDictionaryValueUnownedNonCopying(keysAndValues...)}
}

// KeyValuePair

type KeyValuePair struct {
	Key   Value
	Value Value
}

// Struct

type Struct struct {
	StructType StructType
	Fields     []Value
}

func NewStruct(fields []Value) Struct {
	return Struct{Fields: fields}
}

func (Struct) isValue() {}

func (v Struct) Type() Type {
	return v.StructType
}

func (v Struct) WithType(typ StructType) Struct {
	v.StructType = typ
	return v
}

func (v Struct) ToGoValue() interface{} {
	ret := make([]interface{}, len(v.Fields))

	for i, field := range v.Fields {
		ret[i] = field.ToGoValue()
	}

	return ret
}

func (v Struct) ToRuntimeValue() runtime.Value {
	return compositeToRuntimeValue(v.StructType.Fields, v.Fields)
}

// Resource

type Resource struct {
	ResourceType ResourceType
	Fields       []Value
}

func NewResource(fields []Value) Resource {
	return Resource{Fields: fields}
}

func (Resource) isValue() {}

func (v Resource) Type() Type {
	return v.ResourceType
}

func (v Resource) WithType(typ ResourceType) Resource {
	v.ResourceType = typ
	return v
}

func (v Resource) ToGoValue() interface{} {
	ret := make([]interface{}, len(v.Fields))

	for i, field := range v.Fields {
		ret[i] = field.ToGoValue()
	}

	return ret
}

func (v Resource) ToRuntimeValue() runtime.Value {
	return compositeToRuntimeValue(v.ResourceType.Fields, v.Fields)
}

// Event

type Event struct {
	EventType EventType
	Fields    []Value
}

func NewEvent(fields []Value) Event {
	return Event{Fields: fields}
}

func (Event) isValue() {}

func (v Event) Type() Type {
	return v.EventType
}

func (v Event) WithType(typ EventType) Event {
	v.EventType = typ
	return v
}

func (v Event) ToGoValue() interface{} {
	ret := make([]interface{}, len(v.Fields))

	for i, field := range v.Fields {
		ret[i] = field.ToGoValue()
	}

	return ret
}

func (v Event) ToRuntimeValue() runtime.Value {
	return compositeToRuntimeValue(v.EventType.Fields, v.Fields)
}

func compositeToRuntimeValue(fieldTypes []Field, fieldValues []Value) runtime.Value {
	fields := make(map[string]interpreter.Value, len(fieldTypes))

	for i := 0; i < len(fieldTypes) && i < len(fieldValues); i++ {
		fieldType := fieldTypes[i]
		fieldValue := fieldValues[i]
		fields[fieldType.Identifier] = fieldValue.ToRuntimeValue().Value
	}

	return runtime.Value{Value: &interpreter.CompositeValue{Fields: fields}}
}
