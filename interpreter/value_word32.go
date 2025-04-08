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
	"unsafe"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

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

func (Word32Value) IsValue() {}

func (v Word32Value) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitWord32Value(interpreter, v)
}

func (Word32Value) Walk(_ ValueWalkContext, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (Word32Value) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeWord32)
}

func (Word32Value) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return true
}

func (v Word32Value) String() string {
	return format.Uint(uint64(v))
}

func (v Word32Value) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v Word32Value) MeteredString(context ValueStringContext, _ SeenReferences, _ LocationRange) string {
	common.UseMemory(
		context,
		common.NewRawStringMemoryUsage(
			OverEstimateNumberStringLength(context, v),
		),
	)
	return v.String()
}

func (v Word32Value) ToInt(_ LocationRange) int {
	return int(v)
}

func (v Word32Value) Negate(NumberValueArithmeticContext, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Plus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationPlus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v + o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) SaturatingPlus(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Minus(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMinus,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v - o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) SaturatingMinus(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Mod(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
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

	valueGetter := func() uint32 {
		return uint32(v % o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) Mul(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationMul,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v * o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) SaturatingMul(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Div(context NumberValueArithmeticContext, other NumberValue, locationRange LocationRange) NumberValue {
	o, ok := other.(Word32Value)
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

	valueGetter := func() uint32 {
		return uint32(v / o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) SaturatingDiv(NumberValueArithmeticContext, NumberValue, LocationRange) NumberValue {
	panic(errors.NewUnreachableError())
}

func (v Word32Value) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v < o
}

func (v Word32Value) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v <= o
}

func (v Word32Value) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v > o
}

func (v Word32Value) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return v >= o
}

func (v Word32Value) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherWord32, ok := other.(Word32Value)
	if !ok {
		return false
	}
	return v == otherWord32
}

// HashInput returns a byte slice containing:
// - HashInputTypeWord32 (1 byte)
// - uint32 value encoded in big-endian (4 bytes)
func (v Word32Value) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
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

func (v Word32Value) BitwiseOr(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseOr,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v | o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) BitwiseXor(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseXor,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v ^ o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) BitwiseAnd(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseAnd,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v & o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) BitwiseLeftShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseLeftShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v << o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) BitwiseRightShift(context ValueStaticTypeContext, other IntegerValue, locationRange LocationRange) IntegerValue {
	o, ok := other.(Word32Value)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationBitwiseRightShift,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	valueGetter := func() uint32 {
		return uint32(v >> o)
	}

	return NewWord32Value(context, valueGetter)
}

func (v Word32Value) GetMember(context MemberAccessibleContext, locationRange LocationRange, name string) Value {
	return getNumberValueMember(context, v, name, sema.Word32Type, locationRange)
}

func (Word32Value) RemoveMember(_ *Interpreter, _ LocationRange, _ string) Value {
	// Numbers have no removable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (Word32Value) SetMember(_ MemberAccessibleContext, _ LocationRange, _ string, _ Value) bool {
	// Numbers have no settable members (fields / functions)
	panic(errors.NewUnreachableError())
}

func (v Word32Value) ToBigEndianBytes() []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(v))
	return b
}

func (v Word32Value) ConformsToStaticType(
	_ ValueStaticTypeConformanceContext,
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

func (Word32Value) IsResourceKinded(_ ValueStaticTypeContext) bool {
	return false
}

func (v Word32Value) Transfer(
	context ValueTransferContext,
	_ LocationRange,
	_ atree.Address,
	remove bool,
	storable atree.Storable,
	_ map[atree.ValueID]struct{},
	_ bool,
) Value {
	if remove {
		RemoveReferencedSlab(context, storable)
	}
	return v
}

func (v Word32Value) Clone(_ *Interpreter) Value {
	return v
}

func (Word32Value) DeepRemove(_ ValueRemoveContext, _ bool) {
	// NO-OP
}

func (v Word32Value) ByteSize() uint32 {
	return values.CBORTagSize + values.GetUintCBORSize(uint64(v))
}

func (v Word32Value) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return v, nil
}

func (Word32Value) ChildStorables() []atree.Storable {
	return nil
}
