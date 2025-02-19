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
	"encoding/hex"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
	"golang.org/x/text/unicode/norm"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/format"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/values"
)

// StringValue

type StringValue struct {
	// graphemes is a grapheme cluster segmentation iterator,
	// which is initialized lazily and reused/reset in functions
	// that are based on grapheme clusters
	graphemes       *uniseg.Graphemes
	Str             string
	UnnormalizedStr string
	// length is the cached length of the string, based on grapheme clusters.
	// a negative value indicates the length has not been initialized, see Length()
	length int
}

func NewUnmeteredStringValue(str string) *StringValue {
	return &StringValue{
		Str:             norm.NFC.String(str),
		UnnormalizedStr: str,
		// a negative value indicates the length has not been initialized, see Length()
		length: -1,
	}
}

// Deprecated: NewStringValue_Unsafe creates a new string value
// from the given normalized and unnormalized string.
// NOTE: this function is unsafe, as it does not normalize the string.
// It should only be used for e.g. migration purposes.
func NewStringValue_Unsafe(normalizedStr, unnormalizedStr string) *StringValue {
	return &StringValue{
		Str:             normalizedStr,
		UnnormalizedStr: unnormalizedStr,
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
	// If the string is empty, methods of StringValue should never call prepareGraphemes,
	// as it is not only unnecessary, but also means that the value is the empty string singleton EmptyString,
	// which should not be mutated because it may be used from different goroutines,
	// so should not get mutated by preparing the graphemes iterator.
	if len(v.Str) == 0 {
		panic(errors.NewUnreachableError())
	}

	if v.graphemes == nil {
		v.graphemes = uniseg.NewGraphemes(v.Str)
	} else {
		v.graphemes.Reset()
	}
}

func (*StringValue) isValue() {}

func (v *StringValue) Accept(interpreter *Interpreter, visitor Visitor, _ LocationRange) {
	visitor.VisitStringValue(interpreter, v)
}

func (*StringValue) Walk(_ *Interpreter, _ func(Value), _ LocationRange) {
	// NO-OP
}

func (*StringValue) StaticType(context ValueStaticTypeContext) StaticType {
	return NewPrimitiveStaticType(context, PrimitiveStaticTypeString)
}

func (*StringValue) IsImportable(_ *Interpreter, _ LocationRange) bool {
	return sema.StringType.Importable
}

func (v *StringValue) String() string {
	return format.String(v.Str)
}

func (v *StringValue) RecursiveString(_ SeenReferences) string {
	return v.String()
}

func (v *StringValue) MeteredString(interpreter *Interpreter, _ SeenReferences, _ LocationRange) string {
	l := format.FormattedStringLength(v.Str)
	common.UseMemory(interpreter, common.NewRawStringMemoryUsage(l))
	return v.String()
}

func (v *StringValue) Equal(_ ValueComparisonContext, _ LocationRange, other Value) bool {
	otherString, ok := other.(*StringValue)
	if !ok {
		return false
	}
	return v.Str == otherString.Str
}

func (v *StringValue) Less(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLess,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.Str < otherString.Str)
}

func (v *StringValue) LessEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationLessEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.Str <= otherString.Str)
}

func (v *StringValue) Greater(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreater,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.Str > otherString.Str)
}

func (v *StringValue) GreaterEqual(context ValueComparisonContext, other ComparableValue, locationRange LocationRange) BoolValue {
	otherString, ok := other.(*StringValue)
	if !ok {
		panic(InvalidOperandsError{
			Operation:     ast.OperationGreaterEqual,
			LeftType:      v.StaticType(context),
			RightType:     other.StaticType(context),
			LocationRange: locationRange,
		})
	}

	return BoolValue(v.Str >= otherString.Str)
}

// HashInput returns a byte slice containing:
// - HashInputTypeString (1 byte)
// - string value (n bytes)
func (v *StringValue) HashInput(_ common.MemoryGauge, _ LocationRange, scratch []byte) []byte {
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

	// Meter computation as if the two strings were iterated.
	interpreter.ReportComputation(common.ComputationKindLoop, uint(newLength))

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
	return v.slice(fromIndex, toIndex, locationRange)
}

func (v *StringValue) slice(fromIndex int, toIndex int, locationRange LocationRange) *StringValue {

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

	// If the string is empty or the result is empty,
	// return the empty string singleton EmptyString,
	// as an optimization to avoid allocating a new value.
	//
	// It also ensures that if the sliced value is the empty string singleton EmptyString,
	// which should not be mutated because it may be used from different goroutines,
	// it does not get mutated by preparing the graphemes iterator.
	if len(v.Str) == 0 || fromIndex == toIndex {
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
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeConcatFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				interpreter := invocation.Interpreter
				otherArray, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}
				return v.Concat(interpreter, otherArray, locationRange)
			},
		)

	case sema.StringTypeSliceFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeSliceFunctionType,
			func(v *StringValue, invocation Invocation) Value {
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

	case sema.StringTypeContainsFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeContainsFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Contains(invocation.Interpreter, other)
			},
		)

	case sema.StringTypeIndexFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeIndexFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.IndexOf(invocation.Interpreter, other)
			},
		)

	case sema.StringTypeCountFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeIndexFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				other, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Count(
					invocation.Interpreter,
					invocation.LocationRange,
					other,
				)
			},
		)

	case sema.StringTypeDecodeHexFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeDecodeHexFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				return v.DecodeHex(
					invocation.Interpreter,
					invocation.LocationRange,
				)
			},
		)

	case sema.StringTypeToLowerFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeToLowerFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				return v.ToLower(invocation.Interpreter)
			},
		)

	case sema.StringTypeSplitFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeSplitFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				separator, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.Split(
					invocation.Interpreter,
					invocation.LocationRange,
					separator,
				)
			},
		)

	case sema.StringTypeReplaceAllFunctionName:
		return NewBoundHostFunctionValue(
			interpreter,
			v,
			sema.StringTypeReplaceAllFunctionType,
			func(v *StringValue, invocation Invocation) Value {
				original, ok := invocation.Arguments[0].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				replacement, ok := invocation.Arguments[1].(*StringValue)
				if !ok {
					panic(errors.NewUnreachableError())
				}

				return v.ReplaceAll(
					invocation.Interpreter,
					invocation.LocationRange,
					original,
					replacement,
				)
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
	// If the string is empty, the length is 0, and there are no graphemes.
	//
	// Do NOT store the length, as the value is the empty string singleton EmptyString,
	// which should not be mutated because it may be used from different goroutines.
	if len(v.Str) == 0 {
		return 0
	}

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

	// Meter computation as if the string was iterated.
	interpreter.ReportComputation(common.ComputationKindLoop, uint(len(v.Str)))

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

func (v *StringValue) Split(inter *Interpreter, locationRange LocationRange, separator *StringValue) *ArrayValue {

	if len(separator.Str) == 0 {
		return v.Explode(inter, locationRange)
	}

	count := v.count(inter, locationRange, separator) + 1

	partIndex := 0

	remaining := v

	return NewArrayValueWithIterator(
		inter,
		VarSizedArrayOfStringType,
		common.ZeroAddress,
		uint64(count),
		func() Value {

			inter.ReportComputation(common.ComputationKindLoop, 1)

			if partIndex >= count {
				return nil
			}

			// Set the remainder as the last part
			if partIndex == count-1 {
				partIndex++
				return remaining
			}

			separatorCharacterIndex, _ := remaining.indexOf(inter, separator)
			if separatorCharacterIndex < 0 {
				return nil
			}

			partIndex++

			part := remaining.slice(
				0,
				separatorCharacterIndex,
				locationRange,
			)

			remaining = remaining.slice(
				separatorCharacterIndex+separator.Length(),
				remaining.Length(),
				locationRange,
			)

			return part
		},
	)
}

// Explode returns a Cadence array of type [String], where each element is a single character of the string
func (v *StringValue) Explode(inter *Interpreter, locationRange LocationRange) *ArrayValue {

	iterator := v.Iterator()

	return NewArrayValueWithIterator(
		inter,
		VarSizedArrayOfStringType,
		common.ZeroAddress,
		uint64(v.Length()),
		func() Value {
			value := iterator.Next(inter, locationRange)
			if value == nil {
				return nil
			}

			character, ok := value.(CharacterValue)
			if !ok {
				panic(errors.NewUnreachableError())
			}

			str := character.Str

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

func (v *StringValue) ReplaceAll(
	inter *Interpreter,
	locationRange LocationRange,
	original *StringValue,
	replacement *StringValue,
) *StringValue {

	count := v.count(inter, locationRange, original)
	if count == 0 {
		return v
	}

	newByteLength := len(v.Str) + count*(len(replacement.Str)-len(original.Str))

	memoryUsage := common.NewStringMemoryUsage(newByteLength)

	// Meter computation as if the string was iterated.
	inter.ReportComputation(common.ComputationKindLoop, uint(len(v.Str)))

	remaining := v

	return NewStringValue(
		inter,
		memoryUsage,
		func() string {
			var b strings.Builder
			b.Grow(newByteLength)
			for i := 0; i < count; i++ {

				var originalCharacterIndex, originalByteOffset int
				if original.Length() == 0 {
					if i > 0 {
						originalCharacterIndex = 1

						remaining.prepareGraphemes()
						remaining.graphemes.Next()
						_, originalByteOffset = remaining.graphemes.Positions()
					}
				} else {
					originalCharacterIndex, originalByteOffset = remaining.indexOf(inter, original)
					if originalCharacterIndex < 0 {
						panic(errors.NewUnreachableError())
					}
				}

				b.WriteString(remaining.Str[:originalByteOffset])
				b.WriteString(replacement.Str)

				remaining = remaining.slice(
					originalCharacterIndex+original.Length(),
					remaining.Length(),
					locationRange,
				)
			}
			b.WriteString(remaining.Str)
			return b.String()
		},
	)
}

func (v *StringValue) Storable(storage atree.SlabStorage, address atree.Address, maxInlineSize uint64) (atree.Storable, error) {
	return values.MaybeLargeImmutableStorable(v, storage, address, maxInlineSize)
}

func (*StringValue) NeedsStoreTo(_ atree.Address) bool {
	return false
}

func (*StringValue) IsResourceKinded(context ValueStaticTypeContext) bool {
	return false
}

func (v *StringValue) Transfer(
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

func (v *StringValue) Clone(_ *Interpreter) Value {
	return NewUnmeteredStringValue(v.Str)
}

func (*StringValue) DeepRemove(_ *Interpreter, _ bool) {
	// NO-OP
}

func (v *StringValue) ByteSize() uint32 {
	return values.CBORTagSize + values.GetBytesCBORSize([]byte(v.Str))
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

func (v *StringValue) Iterator() StringValueIterator {
	return StringValueIterator{
		graphemes: uniseg.NewGraphemes(v.Str),
	}
}

func (v *StringValue) ForEach(
	interpreter *Interpreter,
	_ sema.Type,
	function func(value Value) (resume bool),
	transferElements bool,
	locationRange LocationRange,
) {
	iterator := v.Iterator()
	for {
		value := iterator.Next(interpreter, locationRange)
		if value == nil {
			return
		}

		if transferElements {
			value = value.Transfer(
				interpreter,
				locationRange,
				atree.Address{},
				false,
				nil,
				nil,
				false, // value has a parent container because it is from iterator.
			)
		}

		if !function(value) {
			return
		}
	}
}

func (v *StringValue) IsGraphemeBoundaryStart(startOffset int) bool {

	// Empty strings have no grapheme clusters, and therefore no boundaries.
	//
	// Exiting early also ensures that if the checked value is the empty string singleton EmptyString,
	// which should not be mutated because it may be used from different goroutines,
	// it does not get mutated by preparing the graphemes iterator.
	if len(v.Str) == 0 {
		return false
	}

	v.prepareGraphemes()

	var characterIndex int
	return v.seekGraphemeBoundaryStartPrepared(startOffset, &characterIndex)
}

func (v *StringValue) seekGraphemeBoundaryStartPrepared(startOffset int, characterIndex *int) bool {

	for ; v.graphemes.Next(); *characterIndex++ {

		boundaryStart, boundaryEnd := v.graphemes.Positions()
		if boundaryStart == boundaryEnd {
			// Graphemes.Positions() should never return a zero-length grapheme,
			// and only does so if the grapheme iterator
			// - is at the beginning of the string and has not been initialized (i.e. Next() has not been called); or
			// - is at the end of the string and has been exhausted (i.e. Next() has returned false)
			panic(errors.NewUnreachableError())
		}

		if startOffset == boundaryStart {
			return true
		} else if boundaryStart > startOffset {
			return false
		}
	}

	return false
}

func (v *StringValue) IsGraphemeBoundaryEnd(end int) bool {

	// Empty strings have no grapheme clusters, and therefore no boundaries.
	//
	// Exiting early also ensures that if the checked value is the empty string singleton EmptyString,
	// which should not be mutated because it may be used from different goroutines,
	// it does not get mutated by preparing the graphemes iterator.
	if len(v.Str) == 0 {
		return false
	}

	v.prepareGraphemes()
	v.graphemes.Next()

	return v.isGraphemeBoundaryEndPrepared(end)
}

func (v *StringValue) isGraphemeBoundaryEndPrepared(end int) bool {

	for {
		boundaryStart, boundaryEnd := v.graphemes.Positions()
		if boundaryStart == boundaryEnd {
			// Graphemes.Positions() should never return a zero-length grapheme,
			// and only does so if the grapheme iterator
			// - is at the beginning of the string and has not been initialized (i.e. Next() has not been called); or
			// - is at the end of the string and has been exhausted (i.e. Next() has returned false)
			panic(errors.NewUnreachableError())
		}

		if end == boundaryEnd {
			return true
		} else if boundaryEnd > end {
			return false
		}

		if !v.graphemes.Next() {
			return false
		}
	}
}

func (v *StringValue) IndexOf(inter *Interpreter, other *StringValue) IntValue {
	index, _ := v.indexOf(inter, other)
	return NewIntValueFromInt64(inter, int64(index))
}

func (v *StringValue) indexOf(inter *Interpreter, other *StringValue) (characterIndex int, byteOffset int) {

	if len(other.Str) == 0 {
		return 0, 0
	}

	// If the string is empty, exit early.
	//
	// That ensures that if the checked value is the empty string singleton EmptyString,
	// which should not be mutated because it may be used from different goroutines,
	// it does not get mutated by preparing the graphemes iterator.
	if len(v.Str) == 0 {
		return -1, -1
	}

	// Meter computation as if the string was iterated.
	// This is a conservative over-estimation.
	inter.ReportComputation(common.ComputationKindLoop, uint(len(v.Str)*len(other.Str)))

	v.prepareGraphemes()

	// We are dealing with two different positions / indices / measures:
	// - 'CharacterIndex' indicates Cadence characters (grapheme clusters)
	// - 'ByteOffset' indicates bytes

	// Find the position of the substring in the string,
	// by using strings.Index with an increasing start byte offset.
	//
	// The byte offset returned from strings.Index is the start of the substring in the string,
	// but it may not be at a grapheme boundary, so we need to check
	// that both the start and end byte offsets are grapheme boundaries.
	//
	// We do not have a way to translate a byte offset into a character index.
	// Instead, we iterate over the grapheme clusters until we reach the byte offset,
	// keeping track of the character index.
	//
	// We need to back up and restore the grapheme iterator and character index
	// when either the start or the end byte offset are not grapheme boundaries,
	// so the next iteration can start from the correct position.

	for searchStartByteOffset := 0; searchStartByteOffset < len(v.Str); searchStartByteOffset++ {

		relativeFoundByteOffset := strings.Index(v.Str[searchStartByteOffset:], other.Str)
		if relativeFoundByteOffset < 0 {
			break
		}

		// The resulting found byte offset is relative to the search start byte offset,
		// so we need to add the search start byte offset to get the absolute byte offset
		absoluteFoundByteOffset := searchStartByteOffset + relativeFoundByteOffset

		// Back up the grapheme iterator and character index,
		// so the iteration state can be restored
		// in case the byte offset is not at a grapheme boundary
		graphemesBackup := *v.graphemes
		characterIndexBackup := characterIndex

		if v.seekGraphemeBoundaryStartPrepared(absoluteFoundByteOffset, &characterIndex) &&
			v.isGraphemeBoundaryEndPrepared(absoluteFoundByteOffset+len(other.Str)) {

			return characterIndex, absoluteFoundByteOffset
		}

		// Restore the grapheme iterator and character index
		v.graphemes = &graphemesBackup
		characterIndex = characterIndexBackup
	}

	return -1, -1
}

func (v *StringValue) Contains(inter *Interpreter, other *StringValue) BoolValue {
	characterIndex, _ := v.indexOf(inter, other)
	return BoolValue(characterIndex >= 0)
}

func (v *StringValue) Count(inter *Interpreter, locationRange LocationRange, other *StringValue) IntValue {
	index := v.count(inter, locationRange, other)
	return NewIntValueFromInt64(inter, int64(index))
}

func (v *StringValue) count(inter *Interpreter, locationRange LocationRange, other *StringValue) int {
	if other.Length() == 0 {
		return 1 + v.Length()
	}

	// Meter computation as if the string was iterated.
	inter.ReportComputation(common.ComputationKindLoop, uint(len(v.Str)))

	remaining := v
	count := 0

	for {
		index, _ := remaining.indexOf(inter, other)
		if index == -1 {
			return count
		}

		count++

		remaining = remaining.slice(
			index+other.Length(),
			remaining.Length(),
			locationRange,
		)
	}
}

type StringValueIterator struct {
	graphemes *uniseg.Graphemes
}

var _ ValueIterator = StringValueIterator{}

func (i StringValueIterator) Next(_ ValueIteratorContext, _ LocationRange) Value {
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

	argument.Iterate(
		inter,
		func(element Value) (resume bool) {
			character := element.(CharacterValue)
			// Construct directly instead of using NewStringMemoryUsage to avoid
			// having to decrement by 1 due to double counting of empty string.
			common.UseMemory(inter,
				common.MemoryUsage{
					Kind:   common.MemoryKindStringValue,
					Amount: uint64(len(character.Str)),
				},
			)
			builder.WriteString(character.Str)

			return true
		},
		false,
		invocation.LocationRange,
	)

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

	stringArray.Iterate(
		inter,
		func(element Value) (resume bool) {

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
		},
		false,
		invocation.LocationRange,
	)

	return NewUnmeteredStringValue(builder.String())
}

// stringFunction is the `String` function. It is stateless, hence it can be re-used across interpreters.
// Type bound functions are static functions.
var stringFunction = func() Value {
	functionValue := NewUnmeteredStaticHostFunctionValue(
		sema.StringFunctionType,
		func(invocation Invocation) Value {
			return EmptyString
		},
	)

	addMember := func(name string, value Value) {
		if functionValue.NestedVariables == nil {
			functionValue.NestedVariables = map[string]Variable{}
		}
		// these variables are not needed to be metered as they are only ever declared once,
		// and can be considered base interpreter overhead
		functionValue.NestedVariables[name] = NewVariableWithValue(nil, value)
	}

	addMember(
		sema.StringTypeEncodeHexFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.StringTypeEncodeHexFunctionType,
			stringFunctionEncodeHex,
		),
	)

	addMember(
		sema.StringTypeFromUtf8FunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.StringTypeFromUtf8FunctionType,
			stringFunctionFromUtf8,
		),
	)

	addMember(
		sema.StringTypeFromCharactersFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.StringTypeFromCharactersFunctionType,
			stringFunctionFromCharacters,
		),
	)

	addMember(
		sema.StringTypeJoinFunctionName,
		NewUnmeteredStaticHostFunctionValue(
			sema.StringTypeJoinFunctionType,
			stringFunctionJoin,
		),
	)

	return functionValue
}()
