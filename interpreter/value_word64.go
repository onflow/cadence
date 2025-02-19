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
	"encoding/binary"
	"math"
	"math/big"
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

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

func (v Word64Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitWord64Value(interpreter, v)
}

func (Word64Value) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Word64Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeWord64)
}

func (Word64Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v Word64Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word64Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word64Value) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		interpreter,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(interpreter, v),
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

func (v Word64Value) Negate(NumberValueArithmeticContext, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v + o)
	}

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) SaturatingPlus(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v - o)
	}

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) SaturatingMinus(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMod,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
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

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v * o)
	}

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) SaturatingMul(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationDiv,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
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

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) SaturatingDiv(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word64Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v < o)
}

func (v Word64Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v <= o)
}

func (v Word64Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v > o)
}

func (v Word64Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v >= o)
}

func (v Word64Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherWord64, ok := other.(Word64Value)
	if !ok {
		return false
	}
	return v == otherWord64
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord64 (1 byte)
// - uint64 value encoded in big-endian (8 bytes)
func (v Word64Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
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

func (v Word64Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v | o)
	}

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v ^ o)
	}

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v & o)
	}

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v << o)
	}

	return NewWord64Value(context, valueGetter)
}

func (v Word64Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word64Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint64 {
		return uint64(v >> o)
	}

	return NewWord64Value(context, valueGetter)
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

func (Word64Value) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v Word64Value) Transfer(
	interpreter *Interpreter,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
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
	return values.CBORTagSize + values.GetUintCBORSize(uint64(v))
}

func (Word64Value) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v Word64Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word64Value) ChildStorables() []atree.Storable {
	return nil
}
