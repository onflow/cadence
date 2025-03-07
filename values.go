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

package cadence

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"unicode/utf8"
	"unsafe"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/fixedpoint"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// Value

type Value interface {
	isValue()
	Type() Type
	MeteredType(gauge common.MemoryGauge) Type
	fmt.Stringer
}

// NumberValue

type NumberValue interface {
	Value
	ToBigEndianBytes() []byte
}

// Void

type Void struct{}

var _ Value = Void{}

func NewVoid() Void {
	return Void{}
}

func NewMeteredVoid(memoryGauge common.MemoryGauge) Void {
	common.UseMemory(memoryGauge, common.CadenceVoidValueMemoryUsage)
	return NewVoid()
}

func (Void) isValue() {}

func (Void) Type() Type {
	return VoidType
}

func (v Void) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (Void) String() string {
	return format.Void
}

// Optional

type Optional struct {
	Value Value
}

var _ Value = Optional{}

func NewOptional(value Value) Optional {
	return Optional{Value: value}
}

func NewMeteredOptional(memoryGauge common.MemoryGauge, value Value) Optional {
	common.UseMemory(memoryGauge, common.CadenceOptionalValueMemoryUsage)
	return NewOptional(value)
}

func (Optional) isValue() {}

func (o Optional) Type() Type {
	var innerType Type
	if o.Value == nil {
		innerType = NeverType
	} else {
		innerType = o.Value.Type()
	}

	return NewOptionalType(
		innerType,
	)
}

func (o Optional) MeteredType(gauge common.MemoryGauge) Type {
	var innerType Type
	if o.Value == nil {
		innerType = NeverType
	} else {
		innerType = o.Value.MeteredType(gauge)
	}

	return NewMeteredOptionalType(
		gauge,
		innerType,
	)
}

func (o Optional) String() string {
	if o.Value == nil {
		return format.Nil
	}
	return o.Value.String()
}

// Bool

type Bool bool

var _ Value = Bool(false)

func NewBool(b bool) Bool {
	return Bool(b)
}

func NewMeteredBool(memoryGauge common.MemoryGauge, b bool) Bool {
	common.UseMemory(memoryGauge, common.CadenceBoolValueMemoryUsage)
	return NewBool(b)
}

func (Bool) isValue() {}

func (Bool) Type() Type {
	return BoolType
}

func (v Bool) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Bool) String() string {
	return format.Bool(bool(v))
}

// String

type String string

var _ Value = String("")

func NewString(s string) (String, error) {
	if !utf8.ValidString(s) {
		return "", errors.NewDefaultUserError("invalid UTF-8 in string: %s", s)
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

func (String) Type() Type {
	return StringType
}

func (v String) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v String) String() string {
	return format.String(string(v))
}

// Bytes

type Bytes []byte

var _ Value = Bytes(nil)

// Unmetered because this is only used by cadence in tests
func NewBytes(b []byte) Bytes {
	return b
}

func (Bytes) isValue() {}

func (Bytes) Type() Type {
	return TheBytesType
}

func (v Bytes) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Bytes) String() string {
	return format.Bytes(v)
}

// Character

// Character represents a Cadence character, which is a Unicode extended grapheme cluster.
// Hence, use a Go string to be able to hold multiple Unicode code points (Go runes).
// It should consist of exactly one grapheme cluster
type Character string

var _ Value = Character("")

func NewCharacter(b string) (Character, error) {
	if !sema.IsValidCharacter(b) {
		return "\uFFFD", errors.NewDefaultUserError("invalid character: %s", b)
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

func (Character) Type() Type {
	return CharacterType
}

func (v Character) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Character) String() string {
	return format.String(string(v))
}

// Address

const AddressLength = 8

type Address [AddressLength]byte

var _ Value = Address([8]byte{})

func NewAddress(b [AddressLength]byte) Address {
	return b
}

func NewMeteredAddress(memoryGauge common.MemoryGauge, b [AddressLength]byte) Address {
	common.UseMemory(memoryGauge, common.CadenceAddressValueMemoryUsage)
	return NewAddress(b)
}

func BytesToAddress(b []byte) Address {
	var a Address
	copy(a[AddressLength-len(b):AddressLength], b)
	return a
}

func BytesToMeteredAddress(memoryGauge common.MemoryGauge, b []byte) Address {
	common.UseMemory(memoryGauge, common.CadenceAddressValueMemoryUsage)
	return BytesToAddress(b)
}

func (Address) isValue() {}

func (Address) Type() Type {
	return AddressType
}

func (Address) MeteredType(common.MemoryGauge) Type {
	return AddressType
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

var _ Value = Int{}

func NewInt(i int) Int {
	return Int{
		Value: big.NewInt(int64(i)),
	}
}

func NewIntFromBig(i *big.Int) Int {
	return Int{
		Value: i,
	}
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

func (Int) Type() Type {
	return IntType
}

func (v Int) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Int) Int() int {
	return int(v.Value.Int64())
}

func (v Int) Big() *big.Int {
	return v.Value
}

func (v Int) ToBigEndianBytes() []byte {
	return values.SignedBigIntToBigEndianBytes(v.Value)
}

func (v Int) String() string {
	return format.BigInt(v.Value)
}

// Int8

type Int8 int8

var _ Value = Int8(0)

var Int8MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int8(0))))

func NewInt8(v int8) Int8 {
	return Int8(v)
}

func NewMeteredInt8(memoryGauge common.MemoryGauge, v int8) Int8 {
	common.UseMemory(memoryGauge, Int8MemoryUsage)
	return Int8(v)
}

func (Int8) isValue() {}

func (Int8) Type() Type {
	return Int8Type
}

func (v Int8) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Int8) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Int8) String() string {
	return format.Int(int64(v))
}

// Int16

type Int16 int16

var _ Value = Int16(0)

var Int16MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int16(0))))

func NewInt16(v int16) Int16 {
	return Int16(v)
}

func NewMeteredInt16(memoryGauge common.MemoryGauge, v int16) Int16 {
	common.UseMemory(memoryGauge, Int16MemoryUsage)
	return Int16(v)
}

func (Int16) isValue() {}

func (Int16) Type() Type {
	return Int16Type
}

func (v Int16) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = Int32(0)

var Int32MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int32(0))))

func NewInt32(v int32) Int32 {
	return Int32(v)
}

func NewMeteredInt32(memoryGauge common.MemoryGauge, v int32) Int32 {
	common.UseMemory(memoryGauge, Int32MemoryUsage)
	return Int32(v)
}

func (Int32) isValue() {}

func (Int32) Type() Type {
	return Int32Type
}

func (v Int32) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = Int64(0)

var Int64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Int64(0))))

func NewInt64(i int64) Int64 {
	return Int64(i)
}

func NewMeteredInt64(memoryGauge common.MemoryGauge, v int64) Int64 {
	common.UseMemory(memoryGauge, Int64MemoryUsage)
	return Int64(v)
}

func (Int64) isValue() {}

func (Int64) Type() Type {
	return Int64Type
}

func (v Int64) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = Int128{}

var Int128MemoryUsage = common.NewCadenceBigIntMemoryUsage(16)

func NewInt128(i int) Int128 {
	return Int128{
		Value: big.NewInt(int64(i)),
	}
}

var int128MinExceededError = errors.NewDefaultUserError("value exceeds min of Int128")
var int128MaxExceededError = errors.NewDefaultUserError("value exceeds max of Int128")

func NewInt128FromBig(i *big.Int) (Int128, error) {
	if i.Cmp(sema.Int128TypeMinIntBig) < 0 {
		return Int128{}, int128MinExceededError
	}
	if i.Cmp(sema.Int128TypeMaxIntBig) > 0 {
		return Int128{}, int128MaxExceededError
	}
	return Int128{Value: i}, nil
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

func (Int128) Type() Type {
	return Int128Type
}

func (v Int128) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Int128) Int() int {
	return int(v.Value.Int64())
}

func (v Int128) Big() *big.Int {
	return v.Value
}

func (v Int128) ToBigEndianBytes() []byte {
	return values.SignedBigIntToSizedBigEndianBytes(v.Value, sema.Int128TypeSize)
}

func (v Int128) String() string {
	return format.BigInt(v.Value)
}

// Int256

type Int256 struct {
	Value *big.Int
}

var _ Value = Int256{}

var Int256MemoryUsage = common.NewCadenceBigIntMemoryUsage(32)

func NewInt256(i int) Int256 {
	return Int256{
		Value: big.NewInt(int64(i)),
	}
}

var int256MinExceededError = errors.NewDefaultUserError("value exceeds min of Int256")
var int256MaxExceededError = errors.NewDefaultUserError("value exceeds max of Int256")

func NewInt256FromBig(i *big.Int) (Int256, error) {
	if i.Cmp(sema.Int256TypeMinIntBig) < 0 {
		return Int256{}, int256MinExceededError
	}
	if i.Cmp(sema.Int256TypeMaxIntBig) > 0 {
		return Int256{}, int256MaxExceededError
	}
	return Int256{Value: i}, nil
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

func (Int256) Type() Type {
	return Int256Type
}

func (v Int256) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Int256) Int() int {
	return int(v.Value.Int64())
}

func (v Int256) Big() *big.Int {
	return v.Value
}

func (v Int256) ToBigEndianBytes() []byte {
	return values.SignedBigIntToSizedBigEndianBytes(v.Value, sema.Int256TypeSize)
}

func (v Int256) String() string {
	return format.BigInt(v.Value)
}

// UInt

type UInt struct {
	Value *big.Int
}

var _ Value = UInt{}

func NewUInt(i uint) UInt {
	return UInt{
		Value: (&big.Int{}).SetUint64(uint64(i)),
	}
}

var uintNegativeError = errors.NewDefaultUserError("invalid negative value for UInt")

func NewUIntFromBig(i *big.Int) (UInt, error) {
	if i.Sign() < 0 {
		return UInt{}, uintNegativeError
	}
	return UInt{Value: i}, nil
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

func (UInt) Type() Type {
	return UIntType
}

func (v UInt) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v UInt) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt) Big() *big.Int {
	return v.Value
}

func (v UInt) ToBigEndianBytes() []byte {
	return values.UnsignedBigIntToBigEndianBytes(v.Value)
}

func (v UInt) String() string {
	return format.BigInt(v.Value)
}

// UInt8

type UInt8 uint8

var _ Value = UInt8(0)

var UInt8MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt8(0))))

func NewUInt8(v uint8) UInt8 {
	return UInt8(v)
}

func NewMeteredUInt8(gauge common.MemoryGauge, v uint8) UInt8 {
	common.UseMemory(gauge, UInt8MemoryUsage)
	return UInt8(v)
}

func (UInt8) isValue() {}

func (UInt8) Type() Type {
	return UInt8Type
}

func (v UInt8) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v UInt8) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v UInt8) String() string {
	return format.Uint(uint64(v))
}

// UInt16

type UInt16 uint16

var _ Value = UInt16(0)

var UInt16MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt16(0))))

func NewUInt16(v uint16) UInt16 {
	return UInt16(v)
}

func NewMeteredUInt16(gauge common.MemoryGauge, v uint16) UInt16 {
	common.UseMemory(gauge, UInt16MemoryUsage)
	return UInt16(v)
}

func (UInt16) isValue() {}

func (UInt16) Type() Type {
	return UInt16Type
}

func (v UInt16) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = UInt32(0)

var UInt32MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt32(0))))

func NewUInt32(v uint32) UInt32 {
	return UInt32(v)
}

func NewMeteredUInt32(gauge common.MemoryGauge, v uint32) UInt32 {
	common.UseMemory(gauge, UInt32MemoryUsage)
	return UInt32(v)
}

func (UInt32) isValue() {}

func (UInt32) Type() Type {
	return UInt32Type
}

func (v UInt32) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = UInt64(0)

var UInt64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UInt64(0))))

func NewUInt64(v uint64) UInt64 {
	return UInt64(v)
}

func NewMeteredUInt64(gauge common.MemoryGauge, v uint64) UInt64 {
	common.UseMemory(gauge, UInt64MemoryUsage)
	return UInt64(v)
}

func (UInt64) isValue() {}

func (UInt64) Type() Type {
	return UInt64Type
}

func (v UInt64) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = UInt128{}

var UInt128MemoryUsage = common.NewCadenceBigIntMemoryUsage(16)

func NewUInt128(i uint) UInt128 {
	return UInt128{
		Value: (&big.Int{}).SetUint64(uint64(i)),
	}
}

var uint128NegativeError = errors.NewDefaultUserError("invalid negative value for UInt128")
var uint128MaxExceededError = errors.NewDefaultUserError("value exceeds max of UInt128")

func NewUInt128FromBig(i *big.Int) (UInt128, error) {
	if i.Sign() < 0 {
		return UInt128{}, uint128NegativeError
	}
	if i.Cmp(sema.UInt128TypeMaxIntBig) > 0 {
		return UInt128{}, uint128MaxExceededError
	}
	return UInt128{Value: i}, nil
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

func (UInt128) Type() Type {
	return UInt128Type
}

func (v UInt128) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v UInt128) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt128) Big() *big.Int {
	return v.Value
}

func (v UInt128) ToBigEndianBytes() []byte {
	return values.UnsignedBigIntToSizedBigEndianBytes(v.Value, sema.UInt128TypeSize)
}

func (v UInt128) String() string {
	return format.BigInt(v.Value)
}

// UInt256

type UInt256 struct {
	Value *big.Int
}

var _ Value = UInt256{}

var UInt256MemoryUsage = common.NewCadenceBigIntMemoryUsage(32)

func NewUInt256(i uint) UInt256 {
	return UInt256{
		Value: (&big.Int{}).SetUint64(uint64(i)),
	}
}

var uint256NegativeError = errors.NewDefaultUserError("invalid negative value for UInt256")
var uint256MaxExceededError = errors.NewDefaultUserError("value exceeds max of UInt256")

func NewUInt256FromBig(i *big.Int) (UInt256, error) {
	if i.Sign() < 0 {
		return UInt256{}, uint256NegativeError
	}
	if i.Cmp(sema.UInt256TypeMaxIntBig) > 0 {
		return UInt256{}, uint256MaxExceededError
	}
	return UInt256{Value: i}, nil
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

func (UInt256) Type() Type {
	return UInt256Type
}

func (v UInt256) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v UInt256) Int() int {
	return int(v.Value.Uint64())
}

func (v UInt256) Big() *big.Int {
	return v.Value
}

func (v UInt256) ToBigEndianBytes() []byte {
	return values.UnsignedBigIntToSizedBigEndianBytes(v.Value, sema.UInt256TypeSize)
}

func (v UInt256) String() string {
	return format.BigInt(v.Value)
}

// Word8

type Word8 uint8

var _ Value = Word8(0)

var word8MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word8(0))))

func NewWord8(v uint8) Word8 {
	return Word8(v)
}

func NewMeteredWord8(gauge common.MemoryGauge, v uint8) Word8 {
	common.UseMemory(gauge, word8MemoryUsage)
	return Word8(v)
}

func (Word8) isValue() {}

func (Word8) Type() Type {
	return Word8Type
}

func (v Word8) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Word8) ToBigEndianBytes() []byte {
	return []byte{byte(v)}
}

func (v Word8) String() string {
	return format.Uint(uint64(v))
}

// Word16

type Word16 uint16

var _ Value = Word16(0)

var word16MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word16(0))))

func NewWord16(v uint16) Word16 {
	return Word16(v)
}

func NewMeteredWord16(gauge common.MemoryGauge, v uint16) Word16 {
	common.UseMemory(gauge, word16MemoryUsage)
	return Word16(v)
}

func (Word16) isValue() {}

func (Word16) Type() Type {
	return Word16Type
}

func (v Word16) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = Word32(0)

var word32MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word32(0))))

func NewWord32(v uint32) Word32 {
	return Word32(v)
}

func NewMeteredWord32(gauge common.MemoryGauge, v uint32) Word32 {
	common.UseMemory(gauge, word32MemoryUsage)
	return Word32(v)
}

func (Word32) isValue() {}

func (Word32) Type() Type {
	return Word32Type
}

func (v Word32) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = Word64(0)

var word64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Word64(0))))

func NewWord64(v uint64) Word64 {
	return Word64(v)
}

func NewMeteredWord64(gauge common.MemoryGauge, v uint64) Word64 {
	common.UseMemory(gauge, word64MemoryUsage)
	return Word64(v)
}

func (Word64) isValue() {}

func (Word64) Type() Type {
	return Word64Type
}

func (v Word64) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Word64) ToBigEndianBytes() []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func (v Word64) String() string {
	return format.Uint(uint64(v))
}

// Word128

type Word128 struct {
	Value *big.Int
}

var _ Value = Word128{}

var Word128MemoryUsage = common.NewCadenceBigIntMemoryUsage(16)

func NewWord128(i uint) Word128 {
	return Word128{
		Value: (&big.Int{}).SetUint64(uint64(i)),
	}
}

var word128NegativeError = errors.NewDefaultUserError("invalid negative value for Word128")
var word128MaxExceededError = errors.NewDefaultUserError("value exceeds max of Word128")

func NewWord128FromBig(i *big.Int) (Word128, error) {
	if i.Sign() < 0 {
		return Word128{}, word128NegativeError
	}
	if i.Cmp(sema.Word128TypeMaxIntBig) > 0 {
		return Word128{}, word128MaxExceededError
	}
	return Word128{Value: i}, nil
}

func NewMeteredWord128FromBig(
	memoryGauge common.MemoryGauge,
	bigIntConstructor func() *big.Int,
) (Word128, error) {
	common.UseMemory(memoryGauge, Word128MemoryUsage)
	value := bigIntConstructor()
	return NewWord128FromBig(value)
}

func (Word128) isValue() {}

func (Word128) Type() Type {
	return Word128Type
}

func (v Word128) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Word128) Int() int {
	return int(v.Value.Uint64())
}

func (v Word128) Big() *big.Int {
	return v.Value
}

func (v Word128) ToBigEndianBytes() []byte {
	return values.UnsignedBigIntToBigEndianBytes(v.Value)
}

func (v Word128) String() string {
	return format.BigInt(v.Value)
}

// Word256

type Word256 struct {
	Value *big.Int
}

var _ Value = Word256{}

var Word256MemoryUsage = common.NewCadenceBigIntMemoryUsage(32)

func NewWord256(i uint) Word256 {
	return Word256{
		Value: (&big.Int{}).SetUint64(uint64(i)),
	}
}

var word256NegativeError = errors.NewDefaultUserError("invalid negative value for Word256")
var word256MaxExceededError = errors.NewDefaultUserError("value exceeds max of Word256")

func NewWord256FromBig(i *big.Int) (Word256, error) {
	if i.Sign() < 0 {
		return Word256{}, word256NegativeError
	}
	if i.Cmp(sema.Word256TypeMaxIntBig) > 0 {
		return Word256{}, word256MaxExceededError
	}
	return Word256{Value: i}, nil
}

func NewMeteredWord256FromBig(
	memoryGauge common.MemoryGauge,
	bigIntConstructor func() *big.Int,
) (Word256, error) {
	common.UseMemory(memoryGauge, Word256MemoryUsage)
	value := bigIntConstructor()
	return NewWord256FromBig(value)
}

func (Word256) isValue() {}

func (Word256) Type() Type {
	return Word256Type
}

func (v Word256) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Word256) Int() int {
	return int(v.Value.Uint64())
}

func (v Word256) Big() *big.Int {
	return v.Value
}

func (v Word256) ToBigEndianBytes() []byte {
	return values.UnsignedBigIntToBigEndianBytes(v.Value)
}

func (v Word256) String() string {
	return format.BigInt(v.Value)
}

// Fix64

type Fix64 int64

var _ Value = Fix64(0)

var fix64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(Fix64(0))))

func NewFix64(s string) (Fix64, error) {
	v, err := fixedpoint.ParseFix64(s)
	if err != nil {
		return 0, err
	}
	return Fix64(v.Int64()), nil
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

func NewMeteredFix64(gauge common.MemoryGauge, constructor func() (string, error)) (Fix64, error) {
	common.UseMemory(gauge, fix64MemoryUsage)
	value, err := constructor()
	if err != nil {
		return 0, err
	}
	return NewFix64(value)
}

func NewMeteredFix64FromRawFixedPointNumber(gauge common.MemoryGauge, n int64) (Fix64, error) {
	common.UseMemory(gauge, fix64MemoryUsage)
	return Fix64(n), nil
}

func (Fix64) isValue() {}

func (Fix64) Type() Type {
	return Fix64Type
}

func (v Fix64) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = UFix64(0)

var ufix64MemoryUsage = common.NewCadenceNumberMemoryUsage(int(unsafe.Sizeof(UFix64(0))))

func NewUFix64(s string) (UFix64, error) {
	v, err := fixedpoint.ParseUFix64(s)
	if err != nil {
		return 0, err
	}
	return UFix64(v.Uint64()), nil
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

func NewMeteredUFix64(gauge common.MemoryGauge, constructor func() (string, error)) (UFix64, error) {
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

func NewMeteredUFix64FromRawFixedPointNumber(gauge common.MemoryGauge, n uint64) (UFix64, error) {
	common.UseMemory(gauge, ufix64MemoryUsage)
	return UFix64(n), nil
}

func (UFix64) isValue() {}

func (UFix64) Type() Type {
	return UFix64Type
}

func (v UFix64) MeteredType(common.MemoryGauge) Type {
	return v.Type()
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

var _ Value = Array{}

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

func (v Array) Type() Type {
	return v.ArrayType
}

func (v Array) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Array) WithType(arrayType ArrayType) Array {
	v.ArrayType = arrayType
	return v
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
	DictionaryType *DictionaryType
	Pairs          []KeyValuePair
}

var _ Value = Dictionary{}

func NewDictionary(pairs []KeyValuePair) Dictionary {
	return Dictionary{Pairs: pairs}
}

func NewMeteredDictionary(
	gauge common.MemoryGauge,
	size int,
	constructor func() ([]KeyValuePair, error),
) (Dictionary, error) {
	common.UseMemory(gauge, common.CadenceDictionaryValueMemoryUsage)

	pairs, err := constructor()
	if err != nil {
		return Dictionary{}, err
	}
	return NewDictionary(pairs), err
}

func (Dictionary) isValue() {}

func (v Dictionary) Type() Type {
	if v.DictionaryType == nil {
		// Return nil Type instead of Type referencing nil *DictionaryType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.DictionaryType
}

func (v Dictionary) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Dictionary) WithType(dictionaryType *DictionaryType) Dictionary {
	v.DictionaryType = dictionaryType
	return v
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
	common.UseMemory(gauge, common.CadenceKeyValuePairMemoryUsage)
	return KeyValuePair{
		Key:   key,
		Value: value,
	}
}

// Composite

type Composite interface {
	Value

	isComposite()
	getFields() []Field
	getFieldValues() []Value

	SearchFieldByName(fieldName string) Value
	FieldsMappedByName() map[string]Value
}

// linked in by packages that need access to Composite.getFieldValues,
// e.g. JSON and CCF codecs
func getCompositeFieldValues(composite Composite) []Value { //nolint:unused
	return composite.getFieldValues()
}

// SearchFieldByName searches for the field with the given name in the composite type,
// and returns the value of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using FieldsMappedByName if you need to access multiple fields.
func SearchFieldByName(v Composite, fieldName string) Value {
	fieldValues := v.getFieldValues()
	fields := v.getFields()

	if fieldValues == nil || fields == nil {
		return nil
	}

	for i, field := range fields {
		if field.Identifier == fieldName {
			return fieldValues[i]
		}
	}
	return nil
}

func FieldsMappedByName(v Composite) map[string]Value {
	fieldValues := v.getFieldValues()
	fields := v.getFields()

	if fieldValues == nil || fields == nil {
		return nil
	}

	fieldsMap := make(map[string]Value, len(fields))
	for i, fieldValue := range fieldValues {
		var fieldName string
		if i < len(fields) {
			fieldName = fields[i].Identifier
		} else if attachment, ok := fieldValue.(Attachment); ok {
			fieldName = interpreter.AttachmentMemberName(attachment.Type().ID())
		} else {
			panic(errors.NewUnreachableError())
		}
		fieldsMap[fieldName] = fieldValue
	}

	return fieldsMap
}

// Struct

type Struct struct {
	StructType *StructType
	fields     []Value
}

var _ Value = Struct{}
var _ Composite = Struct{}

func NewStruct(fields []Value) Struct {
	return Struct{fields: fields}
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

func (Struct) isComposite() {}

func (v Struct) Type() Type {
	if v.StructType == nil {
		// Return nil Type instead of Type referencing nil *StructType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.StructType
}

func (v Struct) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Struct) WithType(typ *StructType) Struct {
	v.StructType = typ
	return v
}

func (v Struct) String() string {
	return formatComposite(
		v.StructType.ID(),
		v.StructType.fields,
		v.fields,
	)
}

func (v Struct) getFields() []Field {
	if v.StructType == nil {
		return nil
	}

	return v.StructType.fields
}

func (v Struct) getFieldValues() []Value {
	return v.fields
}

// SearchFieldByName searches for the field with the given name in the struct,
// and returns the value of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using FieldsMappedByName if you need to access multiple fields.
func (v Struct) SearchFieldByName(fieldName string) Value {
	return SearchFieldByName(v, fieldName)
}

func (v Struct) FieldsMappedByName() map[string]Value {
	return FieldsMappedByName(v)
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
	fields       []Value
}

var _ Value = Resource{}
var _ Composite = Resource{}

func NewResource(fields []Value) Resource {
	return Resource{fields: fields}
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

func (Resource) isComposite() {}

func (v Resource) Type() Type {
	if v.ResourceType == nil {
		// Return nil Type instead of Type referencing nil *ResourceType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.ResourceType
}

func (v Resource) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Resource) WithType(typ *ResourceType) Resource {
	v.ResourceType = typ
	return v
}

func (v Resource) String() string {
	return formatComposite(
		v.ResourceType.ID(),
		v.ResourceType.fields,
		v.fields,
	)
}

func (v Resource) getFields() []Field {
	if v.ResourceType == nil {
		return nil
	}

	return v.ResourceType.fields
}

func (v Resource) getFieldValues() []Value {
	return v.fields
}

// SearchFieldByName searches for the field with the given name in the resource,
// and returns the value of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using FieldsMappedByName if you need to access multiple fields.
func (v Resource) SearchFieldByName(fieldName string) Value {
	return SearchFieldByName(v, fieldName)
}

func (v Resource) FieldsMappedByName() map[string]Value {
	return FieldsMappedByName(v)
}

// Attachment

type Attachment struct {
	AttachmentType *AttachmentType
	fields         []Value
}

var _ Value = Attachment{}
var _ Composite = Attachment{}

func NewAttachment(fields []Value) Attachment {
	return Attachment{fields: fields}
}

func NewMeteredAttachment(
	gauge common.MemoryGauge,
	numberOfFields int,
	constructor func() ([]Value, error),
) (Attachment, error) {
	baseUsage, sizeUsage := common.NewCadenceAttachmentMemoryUsages(numberOfFields)
	common.UseMemory(gauge, baseUsage)
	common.UseMemory(gauge, sizeUsage)
	fields, err := constructor()
	if err != nil {
		return Attachment{}, err
	}
	return NewAttachment(fields), nil
}

func (Attachment) isValue() {}

func (Attachment) isComposite() {}

func (v Attachment) Type() Type {
	if v.AttachmentType == nil {
		// Return nil Type instead of Type referencing nil *AttachmentType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.AttachmentType
}

func (v Attachment) MeteredType(_ common.MemoryGauge) Type {
	return v.Type()
}

func (v Attachment) WithType(typ *AttachmentType) Attachment {
	v.AttachmentType = typ
	return v
}

func (v Attachment) String() string {
	return formatComposite(
		v.AttachmentType.ID(),
		v.AttachmentType.fields,
		v.fields,
	)
}

func (v Attachment) getFields() []Field {
	if v.AttachmentType == nil {
		return nil
	}

	return v.AttachmentType.fields
}

func (v Attachment) getFieldValues() []Value {
	return v.fields
}

// SearchFieldByName searches for the field with the given name in the attachment,
// and returns the value of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using FieldsMappedByName if you need to access multiple fields.
func (v Attachment) SearchFieldByName(fieldName string) Value {
	return SearchFieldByName(v, fieldName)
}

func (v Attachment) FieldsMappedByName() map[string]Value {
	return FieldsMappedByName(v)
}

// Event

type Event struct {
	EventType *EventType
	fields    []Value
}

var _ Value = Event{}
var _ Composite = Event{}

func NewEvent(fields []Value) Event {
	return Event{fields: fields}
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

func (Event) isComposite() {}

func (v Event) Type() Type {
	if v.EventType == nil {
		// Return nil Type instead of Type referencing nil *EventType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.EventType
}

func (v Event) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Event) WithType(typ *EventType) Event {
	v.EventType = typ
	return v
}

func (v Event) String() string {
	return formatComposite(
		v.EventType.ID(),
		v.EventType.fields,
		v.fields,
	)
}

func (v Event) getFields() []Field {
	if v.EventType == nil {
		return nil
	}

	return v.EventType.fields
}

func (v Event) getFieldValues() []Value {
	return v.fields
}

// SearchFieldByName searches for the field with the given name in the event,
// and returns the value of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using FieldsMappedByName if you need to access multiple fields.
func (v Event) SearchFieldByName(fieldName string) Value {
	return SearchFieldByName(v, fieldName)
}

func (v Event) FieldsMappedByName() map[string]Value {
	return FieldsMappedByName(v)
}

// Contract

type Contract struct {
	ContractType *ContractType
	fields       []Value
}

var _ Value = Contract{}
var _ Composite = Contract{}

func NewContract(fields []Value) Contract {
	return Contract{fields: fields}
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

func (Contract) isComposite() {}

func (v Contract) Type() Type {
	if v.ContractType == nil {
		// Return nil Type instead of Type referencing nil *ContractType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.ContractType
}

func (v Contract) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Contract) WithType(typ *ContractType) Contract {
	v.ContractType = typ
	return v
}

func (v Contract) String() string {
	return formatComposite(
		v.ContractType.ID(),
		v.ContractType.fields,
		v.fields,
	)
}

func (v Contract) getFields() []Field {
	if v.ContractType == nil {
		return nil
	}

	return v.ContractType.fields
}

func (v Contract) getFieldValues() []Value {
	return v.fields
}

// SearchFieldByName searches for the field with the given name in the contract,
// and returns the value of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using FieldsMappedByName if you need to access multiple fields.
func (v Contract) SearchFieldByName(fieldName string) Value {
	return SearchFieldByName(v, fieldName)
}

func (v Contract) FieldsMappedByName() map[string]Value {
	return FieldsMappedByName(v)
}

// InclusiveRange

type InclusiveRange struct {
	InclusiveRangeType *InclusiveRangeType
	Start              Value
	End                Value
	Step               Value
	fields             []Field
}

var _ Value = &InclusiveRange{}

func NewInclusiveRange(start, end, step Value) *InclusiveRange {
	return &InclusiveRange{
		Start: start,
		End:   end,
		Step:  step,
	}
}

func NewMeteredInclusiveRange(
	gauge common.MemoryGauge,
	start, end, step Value,
) *InclusiveRange {
	common.UseMemory(gauge, common.CadenceInclusiveRangeValueMemoryUsage)
	return NewInclusiveRange(start, end, step)
}

func (*InclusiveRange) isValue() {}

func (v *InclusiveRange) Type() Type {
	if v.InclusiveRangeType == nil {
		// Return nil Type instead of Type referencing nil *InclusiveRangeType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.InclusiveRangeType
}

func (v *InclusiveRange) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v *InclusiveRange) WithType(typ *InclusiveRangeType) *InclusiveRange {
	v.InclusiveRangeType = typ
	return v
}

func (v *InclusiveRange) String() string {
	if v.InclusiveRangeType == nil {
		return ""
	}

	if v.fields == nil {
		elementType := v.InclusiveRangeType.ElementType
		v.fields = []Field{
			{
				Identifier: sema.InclusiveRangeTypeStartFieldName,
				Type:       elementType,
			},
			{
				Identifier: sema.InclusiveRangeTypeEndFieldName,
				Type:       elementType,
			},
			{
				Identifier: sema.InclusiveRangeTypeStepFieldName,
				Type:       elementType,
			},
		}
	}

	return formatComposite(
		v.InclusiveRangeType.ID(),
		v.fields,
		[]Value{v.Start, v.End, v.Step},
	)
}

// Path

type Path struct {
	Domain     common.PathDomain
	Identifier string
}

var _ Value = Path{}

func NewPath(domain common.PathDomain, identifier string) (Path, error) {
	if domain == common.PathDomainUnknown {
		return Path{}, errors.NewDefaultUserError("unknown domain in path")
	}

	return Path{
		Domain:     domain,
		Identifier: identifier,
	}, nil
}

func MustNewPath(domain common.PathDomain, identifier string) Path {
	path, err := NewPath(domain, identifier)
	if err != nil {
		panic(err)
	}
	return path
}

func NewMeteredPath(gauge common.MemoryGauge, domain common.PathDomain, identifier string) (Path, error) {
	common.UseMemory(gauge, common.CadencePathValueMemoryUsage)
	return NewPath(domain, identifier)
}

func (Path) isValue() {}

func (v Path) Type() Type {
	switch v.Domain {
	case common.PathDomainStorage:
		return StoragePathType
	case common.PathDomainPrivate:
		return PrivatePathType
	case common.PathDomainPublic:
		return PublicPathType
	}

	panic(errors.NewUnreachableError())
}

func (v Path) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Path) String() string {
	return format.Path(
		v.Domain.Identifier(),
		v.Identifier,
	)
}

// TypeValue

type TypeValue struct {
	StaticType Type
}

var _ Value = TypeValue{}

func NewTypeValue(staticType Type) TypeValue {
	return TypeValue{
		StaticType: staticType,
	}
}

func NewMeteredTypeValue(gauge common.MemoryGauge, staticType Type) TypeValue {
	common.UseMemory(gauge, common.CadenceTypeValueMemoryUsage)
	return NewTypeValue(staticType)
}

func (TypeValue) isValue() {}

func (TypeValue) Type() Type {
	return MetaType
}

func (v TypeValue) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v TypeValue) String() string {
	return format.TypeValue(v.StaticType.ID())
}

// Capability

type Capability struct {
	BorrowType     Type
	Address        Address
	DeprecatedPath *Path // Deprecated: removed in v1.0.0
	ID             UInt64
}

var _ Value = Capability{}

func NewCapability(
	id UInt64,
	address Address,
	borrowType Type,
) Capability {
	return Capability{
		ID:         id,
		Address:    address,
		BorrowType: borrowType,
	}
}

func NewMeteredCapability(
	gauge common.MemoryGauge,
	id UInt64,
	address Address,
	borrowType Type,
) Capability {
	common.UseMemory(gauge, common.CadenceCapabilityValueMemoryUsage)
	return NewCapability(
		id,
		address,
		borrowType,
	)
}

func (Capability) isValue() {}

func (v Capability) Type() Type {
	return NewCapabilityType(v.BorrowType)
}

func (v Capability) MeteredType(gauge common.MemoryGauge) Type {
	return NewMeteredCapabilityType(gauge, v.BorrowType)
}

func (v Capability) String() string {
	if v.DeprecatedPath != nil && v.DeprecatedPath.String() != "" {
		var borrowType string
		if v.BorrowType != nil {
			borrowType = v.BorrowType.ID()
		}

		return format.DeprecatedPathCapability(
			borrowType,
			v.Address.String(),
			v.DeprecatedPath.String(),
		)
	} else {
		return format.Capability(
			v.BorrowType.ID(),
			v.Address.String(),
			v.ID.String(),
		)
	}
}

// Deprecated: removed in v1.0.0
func NewDeprecatedPathCapability(
	address Address,
	path Path,
	borrowType Type,
) Capability {
	return Capability{
		DeprecatedPath: &path,
		Address:        address,
		BorrowType:     borrowType,
	}
}

func NewDeprecatedMeteredPathCapability(
	gauge common.MemoryGauge,
	address Address,
	path Path,
	borrowType Type,
) Capability {
	common.UseMemory(gauge, common.CadenceDeprecatedPathCapabilityValueMemoryUsage)
	return NewDeprecatedPathCapability(
		address,
		path,
		borrowType,
	)
}

// Enum
type Enum struct {
	EnumType *EnumType
	fields   []Value
}

var _ Value = Enum{}
var _ Composite = Enum{}

func NewEnum(fields []Value) Enum {
	return Enum{fields: fields}
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

func (Enum) isComposite() {}

func (v Enum) Type() Type {
	if v.EnumType == nil {
		// Return nil Type instead of Type referencing nil *EnumType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.EnumType
}

func (v Enum) MeteredType(common.MemoryGauge) Type {
	return v.Type()
}

func (v Enum) WithType(typ *EnumType) Enum {
	v.EnumType = typ
	return v
}

func (v Enum) String() string {
	return formatComposite(
		v.EnumType.ID(),
		v.EnumType.fields,
		v.fields,
	)
}

func (v Enum) getFields() []Field {
	if v.EnumType == nil {
		return nil
	}

	return v.EnumType.fields
}

func (v Enum) getFieldValues() []Value {
	return v.fields
}

// SearchFieldByName searches for the field with the given name in the enum,
// and returns the value of the field, or nil if the field is not found.
//
// WARNING: This function performs a linear search, so is not efficient for accessing multiple fields.
// Prefer using FieldsMappedByName if you need to access multiple fields.
func (v Enum) SearchFieldByName(fieldName string) Value {
	return SearchFieldByName(v, fieldName)
}

func (v Enum) FieldsMappedByName() map[string]Value {
	return FieldsMappedByName(v)
}

// Function
type Function struct {
	FunctionType *FunctionType
}

var _ Value = Function{}

func NewFunction(functionType *FunctionType) Function {
	return Function{
		FunctionType: functionType,
	}
}

func NewMeteredFunction(gauge common.MemoryGauge, functionType *FunctionType) Function {
	common.UseMemory(gauge, common.CadenceFunctionValueMemoryUsage)
	return NewFunction(functionType)
}

func (Function) isValue() {}

func (v Function) Type() Type {
	if v.FunctionType == nil {
		// Return nil Type instead of Type referencing nil *FunctionType,
		// so caller can check if v's type is nil and also prevent nil pointer dereference.
		return nil
	}
	return v.FunctionType
}

func (v Function) MeteredType(common.MemoryGauge) Type {
	return v.FunctionType
}

func (v Function) String() string {
	// TODO: include function type
	return "fun ..."
}
