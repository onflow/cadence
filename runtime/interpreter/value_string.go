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
	"encoding/hex"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/onflow/atree"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/format"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/norm"
)

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
		Str: str,
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
	return v.NormalForm() == otherString.NormalForm()
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

	return AsBoolValue(v.NormalForm() < otherString.NormalForm())
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

	return AsBoolValue(v.NormalForm() <= otherString.NormalForm())
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

	return AsBoolValue(v.NormalForm() > otherString.NormalForm())
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

	return AsBoolValue(v.NormalForm() >= otherString.NormalForm())
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

func (v *StringValue) NormalForm() string {
	return norm.NFC.String(v.Str)
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

func stringFunctionEncodeHex(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter
	memoryUsage := common.NewStringMemoryUsage(
		safeMul(argument.Count(), 2, invocation.LocationRange),
	)
	return NewStringValue(
		inter,
		memoryUsage,
		func() string {
			bytes, _ := ByteArrayValueToByteSlice(inter, argument, invocation.LocationRange)
			return hex.EncodeToString(bytes)
		},
	)
}

func stringFunctionFromUtf8(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter
	// naively read the entire byte array before validating
	buf, err := ByteArrayValueToByteSlice(inter, argument, invocation.LocationRange)

	if err != nil {
		panic(errors.NewExternalError(err))
	}

	if !utf8.Valid(buf) {
		return Nil
	}

	memoryUsage := common.NewStringMemoryUsage(len(buf))

	return NewSomeValueNonCopying(
		inter,
		NewStringValue(inter, memoryUsage, func() string {
			return string(buf)
		}),
	)
}

func stringFunctionFromCharacters(invocation Invocation) Value {
	argument, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter

	// NewStringMemoryUsage already accounts for empty string.
	common.UseMemory(inter, common.NewStringMemoryUsage(0))
	var builder strings.Builder

	argument.Iterate(inter, func(element Value) (resume bool) {
		character := element.(CharacterValue)
		// Construct directly instead of using NewStringMemoryUsage to avoid
		// having to decrement by 1 due to double counting of empty string.
		common.UseMemory(inter,
			common.MemoryUsage{
				Kind:   common.MemoryKindStringValue,
				Amount: uint64(len(character)),
			},
		)
		builder.WriteString(string(character))

		return true
	})

	return NewUnmeteredStringValue(builder.String())
}

func stringFunctionJoin(invocation Invocation) Value {
	stringArray, ok := invocation.Arguments[0].(*ArrayValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	inter := invocation.Interpreter

	switch stringArray.Count() {
	case 0:
		return EmptyString
	case 1:
		return stringArray.Get(inter, invocation.LocationRange, 0)
	}

	separator, ok := invocation.Arguments[1].(*StringValue)
	if !ok {
		panic(errors.NewUnreachableError())
	}

	// NewStringMemoryUsage already accounts for empty string.
	common.UseMemory(inter, common.NewStringMemoryUsage(0))
	var builder strings.Builder
	first := true

	stringArray.Iterate(inter, func(element Value) (resume bool) {

		// Meter computation for iterating the array.
		inter.ReportComputation(common.ComputationKindLoop, 1)

		// Add separator
		if !first {
			// Construct directly instead of using NewStringMemoryUsage to avoid
			// having to decrement by 1 due to double counting of empty string.
			common.UseMemory(inter,
				common.MemoryUsage{
					Kind:   common.MemoryKindStringValue,
					Amount: uint64(len(separator.Str)),
				},
			)
			builder.WriteString(separator.Str)
		}
		first = false

		str, ok := element.(*StringValue)
		if !ok {
			panic(errors.NewUnreachableError())
		}

		// Construct directly instead of using NewStringMemoryUsage to avoid
		// having to decrement by 1 due to double counting of empty string.
		common.UseMemory(inter,
			common.MemoryUsage{
				Kind:   common.MemoryKindStringValue,
				Amount: uint64(len(str.Str)),
			},
		)
		builder.WriteString(str.Str)

		return true
	})

	return NewUnmeteredStringValue(builder.String())
}

// stringFunction is the `String` function. It is stateless, hence it can be re-used across interpreters.
var stringFunction = func() Value {
	functionValue := NewUnmeteredHostFunctionValue(
		sema.StringFunctionType,
		func(invocation Invocation) Value {
			return EmptyString
		},
	)

	addMember := func(name string, value Value) {
		if functionValue.NestedVariables == nil {
			functionValue.NestedVariables = map[string]*Variable{}
		}
		// these variables are not needed to be metered as they are only ever declared once,
		// and can be considered base interpreter overhead
		functionValue.NestedVariables[name] = NewVariableWithValue(nil, value)
	}

	addMember(
		sema.StringTypeEncodeHexFunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeEncodeHexFunctionType,
			stringFunctionEncodeHex,
		),
	)

	addMember(
		sema.StringTypeFromUtf8FunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeFromUtf8FunctionType,
			stringFunctionFromUtf8,
		),
	)

	addMember(
		sema.StringTypeFromCharactersFunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeFromCharactersFunctionType,
			stringFunctionFromCharacters,
		),
	)

	addMember(
		sema.StringTypeJoinFunctionName,
		NewUnmeteredHostFunctionValue(
			sema.StringTypeJoinFunctionType,
			stringFunctionJoin,
		),
	)

	return functionValue
}()
