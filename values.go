/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"encoding/binary"
	"fmt"
	"math/big"
	"unicode/utf8"
	"unsafe"

	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// Value

type Value interface {
	isValue()
	Type(gauge common.MemoryGauge) Type
	ToGoValue() interface{}
	fmt.Stringer
}

// NumberValue

type NumberValue interface {
	Value
	ToBigEndianBytes() []byte
}

// Void

type Void struct{}

func NewVoid() Void {
	return Void{}
}

func NewMeteredVoid(memoryGauge common.MemoryGauge) Void {
	common.UseConstantMemory(memoryGauge, common.MemoryKindCadenceVoid)
	return NewVoid()
}

func (Void) isValue() {}

func (Void) Type(gauge common.MemoryGauge) Type {
	return NewVoidType(gauge)
}

func (Void) ToGoValue() interface{} {
	return nil
}

func (Void) String() string {
	return format.Void
}

// Optional

type Optional struct {
	Value Value
}

func NewOptional(value Value) Optional {
	return Optional{Value: value}
}

func NewMeteredOptional(memoryGauge common.MemoryGauge, value Value) Optional {
	common.UseConstantMemory(memoryGauge, common.MemoryKindCadenceOptional)
	return NewOptional(value)
}

func (Optional) isValue() {}

func (o Optional) Type(gauge common.MemoryGauge) Type {
	var innerType Type
	if o.Value == nil {
		innerType = NewNeverType(gauge)
	} else {
		innerType = o.Value.Type(gauge)
	}

	return NewOptionalType(
		gauge,
		innerType,
	)
}

func (o Optional) ToGoValue() interface{} {
	if o.Value == nil {
		return nil
	}

	value := o.Value.ToGoValue()

	return value
}

func (o Optional) String() string {
	if o.Value == nil {
		return format.Nil
	}
	return o.Value.String()
}

// Bool

type Bool bool

func NewBool(b bool) Bool {
	return Bool(b)
}

func NewMeteredBool(memoryGauge common.MemoryGauge, b bool) Bool {
	common.UseConstantMemory(memoryGauge, common.MemoryKindCadenceBool)
	return NewBool(b)
}

func (Bool) isValue() {}

func (Bool) Type(gauge common.MemoryGauge) Type {
	return NewBoolType(gauge)
}

func (v Bool) ToGoValue() interface{} {
	return bool(v)
}

func (v Bool) String() string {
	return format.Bool(bool(v))
}

// String

type String string

func NewString(s string) (String, error) {
	if !utf8.ValidString(s) {
		return "", fmt.Errorf("invalid UTF-8 in string: %s", s)
	}

	return String(s), nil
}

func NewMeteredString(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	stringConstructor func() string,
) (String, error) {
	common.UseMemory(memoryGauge, memoryUsage)
	str := stringConstructor()
	return NewString(str)
}

func (String) isValue() {}

func (String) Type(gauge common.MemoryGauge) Type {
	return NewStringType(gauge)
}

func (v String) ToGoValue() interface{} {
	return string(v)
}

func (v String) String() string {
	return format.String(string(v))
}

// Bytes

type Bytes []byte

// Unmetered because this is only used by cadence in tests
func NewBytes(b []byte) Bytes {
	return b
}

func (Bytes) isValue() {}

func (Bytes) Type(gauge common.MemoryGauge) Type {
	return NewBytesType(gauge)
}

func (v Bytes) ToGoValue() interface{} {
	return []byte(v)
}

func (v Bytes) String() string {
	return format.Bytes(v)
}

// Character

// Character represents a Cadence character, which is a Unicode extended grapheme cluster.
// Hence, use a Go string to be able to hold multiple Unicode code points (Go runes).
// It should consist of exactly one grapheme cluster
//
type Character string

func NewCharacter(b string) (Character, error) {
	if !sema.IsValidCharacter(b) {
		return "\uFFFD", fmt.Errorf("invalid character: %s", b)
	}
	return Character(b), nil
}

func NewMeteredCharacter(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	stringConstructor func() string,
) (Character, error) {
	common.UseMemory(memoryGauge, memoryUsage)
	str := stringConstructor()
	return NewCharacter(str)
}

func (Character) isValue() {}

func (Character) Type(gauge common.MemoryGauge) Type {
	return NewCharacterType(gauge)
}

func (v Character) ToGoValue() interface{} {
	return string(v)
}

func (v Character) String() string {
	return format.String(string(v))
}

// Address

const AddressLength = 8

type Address [AddressLength]byte

func NewAddress(b [AddressLength]byte) Address {
	return b
}

func NewMeteredAddress(memoryGauge common.MemoryGauge, b [AddressLength]byte) Address {
	common.UseConstantMemory(memoryGauge, common.MemoryKindCadenceAddress)
	return NewAddress(b)
}

func BytesToUnmeteredAddress(b []byte) Address {
	var a Address
	copy(a[AddressLength-len(b):AddressLength], b)
	return a
}

func BytesToAddress(memoryGauge common.MemoryGauge, b []byte) Address {
	common.UseConstantMemory(memoryGauge, common.MemoryKindCadenceAddress)
	return BytesToUnmeteredAddress(b)
}

func (Address) isValue() {}

func (Address) Type(gauge common.MemoryGauge) Type {
	return NewAddressType(gauge)
}

func (v Address) ToGoValue() interface{} {
	return [AddressLength]byte(v)
}

func (v Address) Bytes() []byte {
	return v[:]
}

func (v Address) String() string {
	return format.Address(common.Address(v))
}

func (v Address) Hex() string {
	return fmt.Sprintf("%x", [AddressLength]byte(v))
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

func NewMeteredIntFromBig(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) Int {
	common.UseMemory(memoryGauge, memoryUsage)
	value := bigIntConstructor()
	return NewIntFromBig(value)
}

func (Int) isValue() {}

func (Int) Type(gauge common.MemoryGauge) Type {
	return NewIntType(gauge)
}

func (v Int) ToGoValue() interface{} {
	return v.Big()
}

func (v Int) Int() int {
	return int(v.Value.Int64())
}

func (v Int) Big() *big.Int {
	return v.Value
}

func (v Int) ToBigEndianBytes() []byte {
	return interpreter.SignedBigIntToBigEndianBytes(v.Value)
}

func (v Int) String() string {
	return format.BigInt(v.Value)
}

// Int8

type Int8 int8

var Int8MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int8(0))))

func NewInt8(v int8) Int8 {
	return Int8(v)
}

func NewMeteredInt8(memoryGauge common.MemoryGauge, v int8) Int8 {
	common.UseMemory(memoryGauge, Int8MemoryUsage)
	return Int8(v)
}

func (Int8) isValue() {}

func (v Int8) ToGoValue() interface{} {
	return int8(v)
}

func (Int8) Type(gauge common.MemoryGauge) Type {
	return NewInt8Type(gauge)
}

func (v Int8) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Int8) String() string {
	return format.Int(int64(v))
}

// Int16

type Int16 int16

var Int16MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int16(0))))

func NewInt16(v int16) Int16 {
	return Int16(v)
}

func NewMeteredInt16(memoryGauge common.MemoryGauge, v int16) Int16 {
	common.UseMemory(memoryGauge, Int16MemoryUsage)
	return Int16(v)
}

func (Int16) isValue() {}

func (Int16) Type(gauge common.MemoryGauge) Type {
	return NewInt16Type(gauge)
}

func (v Int16) ToGoValue() interface{} {
	return int16(v)
}

func (v Int16) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Int16) String() string {
	return format.Int(int64(v))
}

// Int32

type Int32 int32

var Int32MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int32(0))))

func NewInt32(v int32) Int32 {
	return Int32(v)
}

func NewMeteredInt32(memoryGauge common.MemoryGauge, v int32) Int32 {
	common.UseMemory(memoryGauge, Int32MemoryUsage)
	return Int32(v)
}

func (Int32) isValue() {}

func (Int32) Type(gauge common.MemoryGauge) Type {
	return NewInt32Type(gauge)
}

func (v Int32) ToGoValue() interface{} {
	return int32(v)
}

func (v Int32) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Int32) String() string {
	return format.Int(int64(v))
}

// Int64

type Int64 int64

var Int64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int64(0))))

func NewInt64(i int64) Int64 {
	return Int64(i)
}

func NewMeteredInt64(memoryGauge common.MemoryGauge, v int64) Int64 {
	common.UseMemory(memoryGauge, Int64MemoryUsage)
	return Int64(v)
}

func (Int64) isValue() {}

func (Int64) Type(gauge common.MemoryGauge) Type {
	return NewInt64Type(gauge)
}

func (v Int64) ToGoValue() interface{} {
	return int64(v)
}

func (v Int64) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Int64) String() string {
	return format.Int(int64(v))
}

// Int128

type Int128 struct {
	Value *big.Int
}

var Int128MemoryUsage = common.NewCadenceBigIntMemoryUsage(16)

func NewInt128(i int) Int128 {
	return Int128{big.NewInt(int64(i))}
}

func NewInt128FromBig(i *big.Int) (Int128, error) {
	if i.Cmp(sema.Int128TypeMinIntBig) < 0 {
		return Int128{}, fmt.Errorf("value exceeds min of Int128: %s", i.String())
	}
	if i.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		return Int128{}, fmt.Errorf("value exceeds max of Int128: %s", i.String())
	}
	return Int128{i}, nil
}

func NewMeteredInt128FromBig(
	memoryGauge common.MemoryGauge,
	bigIntConstructor func() *big.Int,
) (Int128, error) {
	common.UseMemory(memoryGauge, Int128MemoryUsage)
	value := bigIntConstructor()
	return NewInt128FromBig(value)
}

func (Int128) isValue() {}

func (Int128) Type(gauge common.MemoryGauge) Type {
	return NewInt128Type(gauge)
}

func (v Int128) ToGoValue() interface{} {
	return v.Big()
}

func (v Int128) Int() int {
	return int(v.Value.Int64())
}

func (v Int128) Big() *big.Int {
	return v.Value
}

func (v Int128) ToBigEndianBytes() []byte {
	return interpreter.SignedBigIntToBigEndianBytes(v.Value)
}

func (v Int128) String() string {
	return format.BigInt(v.Value)
}

// Int256

type Int256 struct {
	Value *big.Int
}

var Int256MemoryUsage = common.NewCadenceBigIntMemoryUsage(32)

func NewInt256(i int) Int256 {
	return Int256{big.NewInt(int64(i))}
}

func NewInt256FromBig(i *big.Int) (Int256, error) {
	if i.Cmp(sema.Int256TypeMinIntBig) < 0 {
		return Int256{}, fmt.Errorf("value exceeds min of Int256: %s", i.String())
	}
	if i.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		return Int256{}, fmt.Errorf("value exceeds max of Int256: %s", i.String())
	}
	return Int256{i}, nil
}

func NewMeteredInt256FromBig(
	memoryGauge common.MemoryGauge,
	bigIntConstructor func() *big.Int,
) (Int256, error) {
	common.UseMemory(memoryGauge, Int256MemoryUsage)
	value := bigIntConstructor()
	return NewInt256FromBig(value)
}

func (Int256) isValue() {}

func (Int256) Type(gauge common.MemoryGauge) Type {
	return NewInt256Type(gauge)
}

func (v Int256) ToGoValue() interface{} {
	return v.Big()
}

func (v Int256) Int() int {
	return int(v.Value.Int64())
}

func (v Int256) Big() *big.Int {
	return v.Value
}

func (v Int256) ToBigEndianBytes() []byte {
	return interpreter.SignedBigIntToBigEndianBytes(v.Value)
}

func (v Int256) String() string {
	return format.BigInt(v.Value)
}

// UInt

type UInt struct {
	Value *big.Int
}

func NewUInt(i uint) UInt {
	return UInt{big.NewInt(int64(i))}
}

func NewUIntFromBig(i *big.Int) (UInt, error) {
	if i.Sign() < 0 {
		return UInt{}, fmt.Errorf("invalid negative value for UInt: %s", i.String())
	}
	return UInt{i}, nil
}

func NewMeteredUIntFromBig(
	memoryGauge common.MemoryGauge,
	memoryUsage common.MemoryUsage,
	bigIntConstructor func() *big.Int,
) (UInt, error) {
	common.UseMemory(memoryGauge, memoryUsage)
	value := bigIntConstructor()
	return NewUIntFromBig(value)
}

func (UInt) isValue() {}

func (UInt) Type(gauge common.MemoryGauge) Type {
	return NewUIntType(gauge)
}

func (v UInt) ToGoValue() interface{} {
	return v.Big()
}

func (v UInt) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt) Big() *big.Int {
	return v.Value
}

func (v UInt) ToBigEndianBytes() []byte {
	return interpreter.UnsignedBigIntToBigEndianBytes(v.Value)
}

func (v UInt) String() string {
	return format.BigInt(v.Value)
}

// UInt8

type UInt8 uint8

var UInt8MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt8(0))))

func NewUInt8(v uint8) UInt8 {
	return UInt8(v)
}

func NewMeteredUInt8(gauge common.MemoryGauge, v uint8) UInt8 {
	common.UseMemory(gauge, UInt8MemoryUsage)
	return UInt8(v)
}

func (UInt8) isValue() {}

func (UInt8) Type(gauge common.MemoryGauge) Type {
	return NewUInt8Type(gauge)
}

func (v UInt8) ToGoValue() interface{} {
	return uint8(v)
}

func (v UInt8) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v UInt8) String() string {
	return format.Uint(uint64(v))
}

// UInt16

type UInt16 uint16

var UInt16MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt16(0))))

func NewUInt16(v uint16) UInt16 {
	return UInt16(v)
}

func NewMeteredUInt16(gauge common.MemoryGauge, v uint16) UInt16 {
	common.UseMemory(gauge, UInt16MemoryUsage)
	return UInt16(v)
}

func (UInt16) isValue() {}

func (UInt16) Type(gauge common.MemoryGauge) Type {
	return NewUInt16Type(gauge)
}

func (v UInt16) ToGoValue() interface{} {
	return uint16(v)
}

func (v UInt16) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v UInt16) String() string {
	return format.Uint(uint64(v))
}

// UInt32

type UInt32 uint32

var UInt32MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt32(0))))

func NewUInt32(v uint32) UInt32 {
	return UInt32(v)
}

func NewMeteredUInt32(gauge common.MemoryGauge, v uint32) UInt32 {
	common.UseMemory(gauge, UInt32MemoryUsage)
	return UInt32(v)
}

func (UInt32) isValue() {}

func (UInt32) Type(gauge common.MemoryGauge) Type {
	return NewUInt32Type(gauge)
}

func (v UInt32) ToGoValue() interface{} {
	return uint32(v)
}

func (v UInt32) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v UInt32) String() string {
	return format.Uint(uint64(v))
}

// UInt64

type UInt64 uint64

var UInt64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt64(0))))

func NewUInt64(v uint64) UInt64 {
	return UInt64(v)
}

func NewMeteredUInt64(gauge common.MemoryGauge, v uint64) UInt64 {
	common.UseMemory(gauge, UInt64MemoryUsage)
	return UInt64(v)
}

func (UInt64) isValue() {}

func (UInt64) Type(gauge common.MemoryGauge) Type {
	return NewUInt64Type(gauge)
}

func (v UInt64) ToGoValue() interface{} {
	return uint64(v)
}

func (v UInt64) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UInt64) String() string {
	return format.Uint(uint64(v))
}

// UInt128

type UInt128 struct {
	Value *big.Int
}

var UInt128MemoryUsage = common.NewCadenceBigIntMemoryUsage(16)

func NewUInt128(i uint) UInt128 {
	return UInt128{big.NewInt(int64(i))}
}

func NewUInt128FromBig(i *big.Int) (UInt128, error) {
	if i.Sign() < 0 {
		return UInt128{}, fmt.Errorf("invalid negative value for UInt: %s", i.String())
	}
	if i.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
		return UInt128{}, fmt.Errorf("value exceeds max of UInt128: %s", i.String())
	}
	return UInt128{i}, nil
}

func NewMeteredUInt128FromBig(
	memoryGauge common.MemoryGauge,
	bigIntConstructor func() *big.Int,
) (UInt128, error) {
	common.UseMemory(memoryGauge, UInt128MemoryUsage)
	value := bigIntConstructor()
	return NewUInt128FromBig(value)
}

func (UInt128) isValue() {}

func (UInt128) Type(gauge common.MemoryGauge) Type {
	return NewUInt128Type(gauge)
}

func (v UInt128) ToGoValue() interface{} {
	return v.Big()
}

func (v UInt128) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt128) Big() *big.Int {
	return v.Value
}

func (v UInt128) ToBigEndianBytes() []byte {
	return interpreter.UnsignedBigIntToBigEndianBytes(v.Value)
}

func (v UInt128) String() string {
	return format.BigInt(v.Value)
}

// UInt256

type UInt256 struct {
	Value *big.Int
}

var UInt256MemoryUsage = common.NewCadenceBigIntMemoryUsage(32)

func NewUInt256(i uint) UInt256 {
	return UInt256{big.NewInt(int64(i))}
}

func NewUInt256FromBig(i *big.Int) (UInt256, error) {
	if i.Sign() < 0 {
		return UInt256{}, fmt.Errorf("invalid negative value for UInt256: %s", i.String())
	}
	if i.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
		return UInt256{}, fmt.Errorf("value exceeds max of UInt256: %s", i.String())
	}
	return UInt256{i}, nil
}

func NewMeteredUInt256FromBig(
	memoryGauge common.MemoryGauge,
	bigIntConstructor func() *big.Int,
) (UInt256, error) {
	common.UseMemory(memoryGauge, UInt256MemoryUsage)
	value := bigIntConstructor()
	return NewUInt256FromBig(value)
}

func (UInt256) isValue() {}

func (UInt256) Type(gauge common.MemoryGauge) Type {
	return NewUInt256Type(gauge)
}

func (v UInt256) ToGoValue() interface{} {
	return v.Big()
}

func (v UInt256) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt256) Big() *big.Int {
	return v.Value
}

func (v UInt256) ToBigEndianBytes() []byte {
	return interpreter.UnsignedBigIntToBigEndianBytes(v.Value)
}

func (v UInt256) String() string {
	return format.BigInt(v.Value)
}

// Word8

type Word8 uint8

var word8MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word8(0))))

func NewWord8(v uint8) Word8 {
	return Word8(v)
}

func NewMeteredWord8(gauge common.MemoryGauge, v uint8) Word8 {
	common.UseMemory(gauge, word8MemoryUsage)
	return Word8(v)
}

func (Word8) isValue() {}

func (Word8) Type(gauge common.MemoryGauge) Type {
	return NewWord8Type(gauge)
}

func (v Word8) ToGoValue() interface{} {
	return uint8(v)
}

func (v Word8) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Word8) String() string {
	return format.Uint(uint64(v))
}

// Word16

type Word16 uint16

var word16MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word16(0))))

func NewWord16(v uint16) Word16 {
	return Word16(v)
}

func NewMeteredWord16(gauge common.MemoryGauge, v uint16) Word16 {
	common.UseMemory(gauge, word16MemoryUsage)
	return Word16(v)
}

func (Word16) isValue() {}

func (Word16) Type(gauge common.MemoryGauge) Type {
	return NewWord16Type(gauge)
}

func (v Word16) ToGoValue() interface{} {
	return uint16(v)
}

func (v Word16) ToBigEndianBytes() []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(v))
	return b
}

func (v Word16) String() string {
	return format.Uint(uint64(v))
}

// Word32

type Word32 uint32

var word32MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word32(0))))

func NewWord32(v uint32) Word32 {
	return Word32(v)
}

func NewMeteredWord32(gauge common.MemoryGauge, v uint32) Word32 {
	common.UseMemory(gauge, word32MemoryUsage)
	return Word32(v)
}

func (Word32) isValue() {}

func (Word32) Type(gauge common.MemoryGauge) Type {
	return NewWord32Type(gauge)
}

func (v Word32) ToGoValue() interface{} {
	return uint32(v)
}

func (v Word32) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Word32) String() string {
	return format.Uint(uint64(v))
}

// Word64

type Word64 uint64

var word64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word64(0))))

func NewWord64(v uint64) Word64 {
	return Word64(v)
}

func NewMeteredWord64(gauge common.MemoryGauge, v uint64) Word64 {
	common.UseMemory(gauge, word64MemoryUsage)
	return Word64(v)
}

func (Word64) isValue() {}

func (Word64) Type(gauge common.MemoryGauge) Type {
	return NewWord64Type(gauge)
}

func (v Word64) ToGoValue() interface{} {
	return uint64(v)
}

func (v Word64) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Word64) String() string {
	return format.Uint(uint64(v))
}

// Fix64

type Fix64 int64

var fix64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Fix64(0))))

func NewFix64(value int64) (Fix64, error) {
	return Fix64(value), nil
}

func NewFix64FromParts(negative bool, integer int, fraction uint) (Fix64, error) {
	v, err := fixedpoint.NewFix64(
		negative,
		new(big.Int).SetInt64(int64(integer)),
		new(big.Int).SetInt64(int64(fraction)),
		fixedpoint.Fix64Scale,
	)
	if err != nil {
		return 0, err
	}
	return Fix64(v.Int64()), nil
}

func NewMeteredFix64(gauge common.MemoryGauge, constructor func() (int64, error)) (Fix64, error) {
	common.UseMemory(gauge, fix64MemoryUsage)
	value, err := constructor()
	if err != nil {
		return 0, err
	}
	return NewFix64(value)
}

func ParseFix64(s string) (int64, error) {
	v, err := fixedpoint.ParseFix64(s)
	if err != nil {
		return 0, err
	}
	return v.Int64(), nil
}

func (Fix64) isValue() {}

func (Fix64) Type(gauge common.MemoryGauge) Type {
	return NewFix64Type(gauge)
}

func (v Fix64) ToGoValue() interface{} {
	return int64(v)
}

func (v Fix64) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Fix64) String() string {
	return format.Fix64(int64(v))
}

// UFix64

type UFix64 uint64

var ufix64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UFix64(0))))

func NewUFix64(value uint64) (UFix64, error) {
	return UFix64(value), nil
}

func NewUFix64FromParts(integer int, fraction uint) (UFix64, error) {
	v, err := fixedpoint.NewUFix64(
		new(big.Int).SetInt64(int64(integer)),
		new(big.Int).SetInt64(int64(fraction)),
		fixedpoint.Fix64Scale,
	)
	if err != nil {
		return 0, err
	}
	return UFix64(v.Uint64()), nil
}

func NewMeteredUFix64(gauge common.MemoryGauge, constructor func() (uint64, error)) (UFix64, error) {
	common.UseMemory(gauge, ufix64MemoryUsage)
	value, err := constructor()
	if err != nil {
		return 0, err
	}
	return NewUFix64(value)
}

func ParseUFix64(s string) (uint64, error) {
	v, err := fixedpoint.ParseUFix64(s)
	if err != nil {
		return 0, err
	}
	return v.Uint64(), nil
}

func (UFix64) isValue() {}

func (UFix64) Type(gauge common.MemoryGauge) Type {
	return NewUFix64Type(gauge)
}

func (v UFix64) ToGoValue() interface{} {
	return uint64(v)
}

func (v UFix64) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v UFix64) String() string {
	return format.UFix64(uint64(v))
}

// Array

type Array struct {
	ArrayType ArrayType
	Values    []Value
}

func NewArray(values []Value) Array {
	return Array{Values: values}
}

func NewMeteredArray(
	gauge common.MemoryGauge,
	length int,
	constructor func() ([]Value, error),
) (Array, error) {
	baseUse, lengthUse := common.NewCadenceArrayMemoryUsages(length)
	common.UseMemory(gauge, baseUse)
	common.UseMemory(gauge, lengthUse)

	values, err := constructor()
	if err != nil {
		return Array{}, err
	}

	return NewArray(values), nil
}

func (Array) isValue() {}

func (v Array) Type(_ common.MemoryGauge) Type {
	return v.ArrayType
}

func (v Array) WithType(arrayType ArrayType) Array {
	v.ArrayType = arrayType
	return v
}

func (v Array) ToGoValue() interface{} {
	ret := make([]interface{}, len(v.Values))

	for i, e := range v.Values {
		ret[i] = e.ToGoValue()
	}

	return ret
}

func (v Array) String() string {
	values := make([]string, len(v.Values))
	for i, value := range v.Values {
		values[i] = value.String()
	}
	return format.Array(values)
}

// Dictionary

type Dictionary struct {
	DictionaryType Type
	Pairs          []KeyValuePair
}

func NewDictionary(pairs []KeyValuePair) Dictionary {
	return Dictionary{Pairs: pairs}
}

func NewMeteredDictionary(
	gauge common.MemoryGauge,
	size int,
	constructor func() ([]KeyValuePair, error),
) (Dictionary, error) {
	baseUse, lengthUse := common.NewCadenceDictionaryMemoryUsages(size)
	common.UseMemory(gauge, baseUse)
	common.UseMemory(gauge, lengthUse)

	pairs, err := constructor()
	if err != nil {
		return Dictionary{}, err
	}
	return NewDictionary(pairs), err
}

func (Dictionary) isValue() {}

func (v Dictionary) Type(_ common.MemoryGauge) Type {
	return v.DictionaryType
}

func (v Dictionary) WithType(dictionaryType DictionaryType) Dictionary {
	v.DictionaryType = dictionaryType
	return v
}

func (v Dictionary) ToGoValue() interface{} {
	ret := map[interface{}]interface{}{}

	for _, p := range v.Pairs {
		ret[p.Key.ToGoValue()] = p.Value.ToGoValue()
	}

	return ret
}

func (v Dictionary) String() string {
	pairs := make([]struct {
		Key   string
		Value string
	}, len(v.Pairs))

	for i, pair := range v.Pairs {
		pairs[i] = struct {
			Key   string
			Value string
		}{
			Key:   pair.Key.String(),
			Value: pair.Value.String(),
		}
	}

	return format.Dictionary(pairs)
}

// KeyValuePair

type KeyValuePair struct {
	Key   Value
	Value Value
}

func NewMeteredKeyValuePair(gauge common.MemoryGauge, key, value Value) KeyValuePair {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceKeyValuePair)
	return KeyValuePair{
		Key:   key,
		Value: value,
	}
}

// Struct

type Struct struct {
	StructType *StructType
	Fields     []Value
}

func NewStruct(fields []Value) Struct {
	return Struct{Fields: fields}
}

func NewMeteredStruct(
	gauge common.MemoryGauge,
	numberOfFields int,
	constructor func() ([]Value, error),
) (Struct, error) {
	baseUsage, sizeUsage := common.NewCadenceStructMemoryUsages(numberOfFields)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, sizeUsage)

	fields, err := constructor()
	if err != nil {
		return Struct{}, err
	}
	return NewStruct(fields), nil
}

func (Struct) isValue() {}

func (v Struct) Type(_ common.MemoryGauge) Type {
	return v.StructType
}

func (v Struct) WithType(typ *StructType) Struct {
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

func (v Struct) String() string {
	return formatComposite(v.StructType.ID(), v.StructType.Fields, v.Fields)
}

func formatComposite(typeID string, fields []Field, values []Value) string {
	preparedFields := make([]struct {
		Name  string
		Value string
	}, 0, len(fields))
	for i, field := range fields {
		value := values[i]
		preparedFields = append(preparedFields,
			struct {
				Name  string
				Value string
			}{
				Name:  field.Identifier,
				Value: value.String(),
			},
		)
	}

	return format.Composite(typeID, preparedFields)
}

// Resource

type Resource struct {
	ResourceType *ResourceType
	Fields       []Value
}

func NewResource(fields []Value) Resource {
	return Resource{Fields: fields}
}

func NewMeteredResource(
	gauge common.MemoryGauge,
	numberOfFields int,
	constructor func() ([]Value, error),
) (Resource, error) {
	baseUsage, sizeUsage := common.NewCadenceResourceMemoryUsages(numberOfFields)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, sizeUsage)
	fields, err := constructor()
	if err != nil {
		return Resource{}, err
	}
	return NewResource(fields), nil
}

func (Resource) isValue() {}

func (v Resource) Type(_ common.MemoryGauge) Type {
	return v.ResourceType
}

func (v Resource) WithType(typ *ResourceType) Resource {
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

func (v Resource) String() string {
	return formatComposite(v.ResourceType.ID(), v.ResourceType.Fields, v.Fields)
}

// Event

type Event struct {
	EventType *EventType
	Fields    []Value
}

func NewEvent(fields []Value) Event {
	return Event{Fields: fields}
}

func NewMeteredEvent(
	gauge common.MemoryGauge,
	numberOfFields int,
	constructor func() ([]Value, error),
) (Event, error) {
	baseUsage, sizeUsage := common.NewCadenceEventMemoryUsages(numberOfFields)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, sizeUsage)
	fields, err := constructor()
	if err != nil {
		return Event{}, err
	}
	return NewEvent(fields), nil
}

func (Event) isValue() {}

func (v Event) Type(_ common.MemoryGauge) Type {
	return v.EventType
}

func (v Event) WithType(typ *EventType) Event {
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
func (v Event) String() string {
	return formatComposite(v.EventType.ID(), v.EventType.Fields, v.Fields)
}

// Contract

type Contract struct {
	ContractType *ContractType
	Fields       []Value
}

func NewContract(fields []Value) Contract {
	return Contract{Fields: fields}
}

func NewMeteredContract(
	gauge common.MemoryGauge,
	numberOfFields int,
	constructor func() ([]Value, error),
) (Contract, error) {
	baseUsage, sizeUsage := common.NewCadenceContractMemoryUsages(numberOfFields)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, sizeUsage)
	fields, err := constructor()
	if err != nil {
		return Contract{}, err
	}
	return NewContract(fields), nil
}

func (Contract) isValue() {}

func (v Contract) Type(_ common.MemoryGauge) Type {
	return v.ContractType
}

func (v Contract) WithType(typ *ContractType) Contract {
	v.ContractType = typ
	return v
}

func (v Contract) ToGoValue() interface{} {
	ret := make([]interface{}, len(v.Fields))

	for i, field := range v.Fields {
		ret[i] = field.ToGoValue()
	}

	return ret
}

func (v Contract) String() string {
	return formatComposite(v.ContractType.ID(), v.ContractType.Fields, v.Fields)
}

// Link

type Link struct {
	TargetPath Path
	// TODO: a future version might want to export the whole type
	BorrowType string
}

func NewLink(targetPath Path, borrowType string) Link {
	return Link{
		TargetPath: targetPath,
		BorrowType: borrowType,
	}
}

func NewMeteredLink(gauge common.MemoryGauge, targetPath Path, borrowType string) Link {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceLink)
	return NewLink(targetPath, borrowType)
}

func (Link) isValue() {}

func (v Link) Type(_ common.MemoryGauge) Type {
	return nil
}

func (v Link) ToGoValue() interface{} {
	return nil
}

func (v Link) String() string {
	return format.Link(
		v.BorrowType,
		v.TargetPath.String(),
	)
}

// Path

type Path struct {
	Domain     string
	Identifier string
}

func NewPath(domain, identifier string) Path {
	return Path{
		Domain:     domain,
		Identifier: identifier,
	}
}

func NewMeteredPath(gauge common.MemoryGauge, domain, identifier string) Path {
	common.UseConstantMemory(gauge, common.MemoryKindCadencePath)
	return NewPath(domain, identifier)
}

func (Path) isValue() {}

func (Path) Type(gauge common.MemoryGauge) Type {
	return NewPathType(gauge)
}

func (Path) ToGoValue() interface{} {
	return nil
}

func (v Path) String() string {
	return format.Path(
		v.Domain,
		v.Identifier,
	)
}

// TypeValue

type TypeValue struct {
	StaticType Type
}

func NewTypeValue(staticType Type) TypeValue {
	return TypeValue{
		StaticType: staticType,
	}
}

func NewMeteredTypeValue(gauge common.MemoryGauge, staticType Type) TypeValue {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceTypeValue)
	return NewTypeValue(staticType)
}

func (TypeValue) isValue() {}

func (TypeValue) Type(gauge common.MemoryGauge) Type {
	return NewMetaType(gauge)
}

func (TypeValue) ToGoValue() interface{} {
	return nil
}

func (v TypeValue) String() string {
	return format.TypeValue(v.StaticType.ID())
}

// Capability

type Capability struct {
	Path       Path
	Address    Address
	BorrowType Type
}

func NewCapability(path Path, address Address, borrowType Type) Capability {
	return Capability{
		Path:       path,
		Address:    address,
		BorrowType: borrowType,
	}
}

func NewMeteredCapability(gauge common.MemoryGauge, path Path, address Address, borrowType Type) Capability {
	common.UseConstantMemory(gauge, common.MemoryKindCadenceCapability)
	return NewCapability(path, address, borrowType)
}

func (Capability) isValue() {}

func (v Capability) Type(gauge common.MemoryGauge) Type {
	return NewCapabilityType(gauge, v.BorrowType)
}

func (Capability) ToGoValue() interface{} {
	return nil
}

func (v Capability) String() string {
	return format.Capability(
		v.BorrowType.ID(),
		v.Address.String(),
		v.Path.String(),
	)
}

// Enum
type Enum struct {
	EnumType *EnumType
	Fields   []Value
}

func NewEnum(fields []Value) Enum {
	return Enum{Fields: fields}
}

func NewMeteredEnum(
	gauge common.MemoryGauge,
	numberOfFields int,
	constructor func() ([]Value, error),
) (Enum, error) {
	baseUsage, sizeUsage := common.NewCadenceEnumMemoryUsages(numberOfFields)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, sizeUsage)
	fields, err := constructor()
	if err != nil {
		return Enum{}, err
	}
	return NewEnum(fields), nil
}

func (Enum) isValue() {}

func (v Enum) Type(_ common.MemoryGauge) Type {
	return v.EnumType
}

func (v Enum) WithType(typ *EnumType) Enum {
	v.EnumType = typ
	return v
}

func (v Enum) ToGoValue() interface{} {
	ret := make([]interface{}, len(v.Fields))

	for i, field := range v.Fields {
		ret[i] = field.ToGoValue()
	}

	return ret
}

func (v Enum) String() string {
	return formatComposite(v.EnumType.ID(), v.EnumType.Fields, v.Fields)
}
