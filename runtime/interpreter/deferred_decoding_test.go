/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCompositeDeferredDecoding(t *testing.T) {

	t.Parallel()

	t.Run("Simple composite", func(t *testing.T) {

		members := NewStringValueOrderedMap()
		members.Set("a", NewStringValue("hello"))
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)

		// Value must not be loaded. i.e: the content is available
		assert.NotNil(t, compositeValue.content)

		// The meta-info and fields raw content are not loaded yet
		assert.Nil(t, compositeValue.fieldsContent)
		assert.Empty(t, compositeValue.location)
		assert.Empty(t, compositeValue.qualifiedIdentifier)
		assert.Equal(t, common.CompositeKindUnknown, compositeValue.kind)

		// Use the Getters and see whether the meta-info are loaded
		assert.Equal(t, value.Location(), compositeValue.Location())
		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
		assert.Equal(t, value.Kind(), compositeValue.Kind())

		// Now the content must be cleared
		assert.Nil(t, compositeValue.content)

		// And the fields raw content must be available
		assert.NotNil(t, compositeValue.fieldsContent)

		// Check all the fields using getters

		decodedFields := compositeValue.Fields()
		require.Equal(t, 2, decodedFields.Len())

		decodeFieldValue, contains := decodedFields.Get("a")
		assert.True(t, contains)
		assert.Equal(t, NewStringValue("hello"), decodeFieldValue)

		decodeFieldValue, contains = decodedFields.Get("b")
		assert.True(t, contains)
		assert.Equal(t, BoolValue(true), decodeFieldValue)

		// Once all the fields are loaded, the fields raw content must be cleared
		assert.Nil(t, compositeValue.fieldsContent)
	})

	t.Run("Nested composite", func(t *testing.T) {
		value := newTestLargeCompositeValue(0)

		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)

		address, ok := compositeValue.Fields().Get("address")
		assert.True(t, ok)

		require.IsType(t, &CompositeValue{}, address)
		nestedCompositeValue := address.(*CompositeValue)

		// Inner composite value must not be loaded
		assert.NotNil(t, nestedCompositeValue.content)
	})

	t.Run("Field update", func(t *testing.T) {
		value := newTestLargeCompositeValue(0)

		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)

		newValue := NewStringValue("green")
		compositeValue.SetMember(nil, nil, "status", newValue)

		// Composite value must be loaded
		assert.Nil(t, compositeValue.content)

		// check updated value
		fieldValue, contains := compositeValue.Fields().Get("status")
		assert.True(t, contains)
		assert.Equal(t, newValue, fieldValue)
	})

	t.Run("Round trip - without loading", func(t *testing.T) {

		members := NewStringValueOrderedMap()
		members.Set("a", NewStringValue("hello"))
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		// Encode
		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		// Value must not be loaded. i.e: the content is available
		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)
		assert.NotNil(t, compositeValue.content)

		// Re encode the decoded value
		reEncoded, _, err := EncodeValue(decoded, nil, true, nil)
		require.NoError(t, err)

		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, reDecoded)
		compositeValue = reDecoded.(*CompositeValue)

		compositeValue.ensureFieldsLoaded()

		// Check the meta info
		assert.Equal(t, value.Location(), compositeValue.Location())
		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
		assert.Equal(t, value.Kind(), compositeValue.Kind())

		// Check the fields

		decodedFields := compositeValue.Fields()
		require.Equal(t, 2, decodedFields.Len())

		decodeFieldValue, contains := decodedFields.Get("a")
		assert.True(t, contains)
		assert.Equal(t, NewStringValue("hello"), decodeFieldValue)

		decodeFieldValue, contains = decodedFields.Get("b")
		assert.True(t, contains)
		assert.Equal(t, BoolValue(true), decodeFieldValue)
	})

	t.Run("Round trip - partially loaded", func(t *testing.T) {

		members := NewStringValueOrderedMap()
		members.Set("a", NewStringValue("hello"))
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		// Encode
		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		// Partially loaded the value.

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)
		// This will only load the meta info, but not the fields
		compositeValue.QualifiedIdentifier()

		assert.Nil(t, compositeValue.content)
		assert.NotNil(t, compositeValue.fieldsContent)

		// Re encode the decoded value
		reEncoded, _, err := EncodeValue(decoded, nil, true, nil)
		require.NoError(t, err)

		// Decode back the value
		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, reDecoded)
		compositeValue = reDecoded.(*CompositeValue)

		compositeValue.ensureFieldsLoaded()

		// Check the meta info
		assert.Equal(t, value.Location(), compositeValue.Location())
		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
		assert.Equal(t, value.Kind(), compositeValue.Kind())

		// Check the fields

		decodedFields := compositeValue.Fields()
		require.Equal(t, 2, decodedFields.Len())

		decodeFieldValue, contains := decodedFields.Get("a")
		assert.True(t, contains)
		assert.Equal(t, NewStringValue("hello"), decodeFieldValue)

		decodeFieldValue, contains = decodedFields.Get("b")
		assert.True(t, contains)
		assert.Equal(t, BoolValue(true), decodeFieldValue)
	})

	t.Run("callback", func(t *testing.T) {

		stringValue := NewStringValue("hello")

		members := NewStringValueOrderedMap()
		members.Set("a", stringValue)
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		// Encode
		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		// Decode

		type decodeCallback struct {
			value interface{}
			path  []string
		}

		var decodeCallbacks []decodeCallback
		callback := func(value interface{}, path []string) {
			valuePath := make([]string, len(path))
			copy(valuePath, path)

			decodeCallbacks = append(decodeCallbacks, decodeCallback{
				value: value,
				path:  valuePath,
			})
		}

		decoded, err := DecodeValue(encoded, &testOwner, []string{}, CurrentEncodingVersion, callback)
		require.NoError(t, err)

		// Callback must be only called once
		require.Len(t, decodeCallbacks, 1)
		assert.Equal(t, decoded, decodeCallbacks[0].value)
		assert.Equal(t, []string{}, decodeCallbacks[0].path)

		// Load the meta info, but not the fields
		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)
		compositeValue.QualifiedIdentifier()

		require.Len(t, decodeCallbacks, 1)
		assert.Equal(t, decoded, decodeCallbacks[0].value)
		assert.Equal(t, []string{}, decodeCallbacks[0].path)

		// Load fields
		compositeValue.Fields()

		// Callback must now have all three values
		require.Len(t, decodeCallbacks, 3)

		assert.Equal(t, decoded, decodeCallbacks[0].value)
		assert.Equal(t, []string{}, decodeCallbacks[0].path)

		assert.Equal(t, stringValue, decodeCallbacks[1].value)
		assert.Equal(t, []string{"a"}, decodeCallbacks[1].path)

		assert.Equal(t, BoolValue(true), decodeCallbacks[2].value)
		assert.Equal(t, []string{"b"}, decodeCallbacks[2].path)
	})

	t.Run("re-encoding", func(t *testing.T) {

		members := NewStringValueOrderedMap()
		members.Set("a", NewStringValue("hello"))
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)

		// Encode
		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		// Partially loaded the value.

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)
		// This will only load the meta info, but not the fields
		compositeValue.QualifiedIdentifier()

		assert.Nil(t, compositeValue.content)
		assert.NotNil(t, compositeValue.fieldsContent)

		// Re encode the decoded value
		type encodeCallback struct {
			value Value
			path  []string
		}

		var encodeCallbacks []encodeCallback
		callback := func(value Value, path []string) {
			valuePath := make([]string, len(path))
			copy(valuePath, path)

			encodeCallbacks = append(encodeCallbacks, encodeCallback{
				value: value,
				path:  valuePath,
			})
		}

		_, _, err = EncodeValue(decoded, nil, true, callback)
		require.NoError(t, err)

		// Elements are not loaded, so they must not be encoded again.
		// i.e: Callback must be only called once.
		require.Len(t, encodeCallbacks, 1)
		assert.Equal(t, decoded, encodeCallbacks[0].value)
		assert.Equal(t, []string{}, encodeCallbacks[0].path)
	})

	t.Run("storable and modified", func(t *testing.T) {
		members := NewStringValueOrderedMap()
		members.Set("a", NewStringValue("hello"))
		members.Set("b", BoolValue(true))

		value := NewCompositeValue(
			utils.TestLocation,
			"TestStruct",
			common.CompositeKindStructure,
			members,
			nil,
		)

		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &CompositeValue{}, decoded)
		compositeValue := decoded.(*CompositeValue)

		assert.False(t, compositeValue.IsModified())
		assert.True(t, compositeValue.IsStorable())

		// fields must not be loaded
		assert.Nil(t, compositeValue.fields)
		assert.Nil(t, compositeValue.content)
		assert.NotNil(t, compositeValue.fieldsContent)
	})
}

func BenchmarkCompositeDeferredDecoding(b *testing.B) {

	encoded, _, err := EncodeValue(newTestLargeCompositeValue(0), nil, true, nil)
	require.NoError(b, err)

	b.Run("Simply decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)
		}
	})

	b.Run("Access identifier", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)

			composite := decoded.(*CompositeValue)
			composite.QualifiedIdentifier()
		}
	})

	b.Run("Access field", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)

			composite := decoded.(*CompositeValue)
			_, ok := composite.Fields().Get("fname")
			require.True(b, ok)
		}
	})

	b.Run("Re-encode decoded", func(b *testing.B) {
		b.ReportAllocs()

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, err = EncodeValue(decoded, nil, true, nil)
			require.NoError(b, err)
		}
	})
}

func newTestLargeCompositeValue(id int) *CompositeValue {
	addressFields := NewStringValueOrderedMap()
	addressFields.Set("street", NewStringValue(fmt.Sprintf("No: %d", id)))
	addressFields.Set("city", NewStringValue("Vancouver"))
	addressFields.Set("state", NewStringValue("BC"))
	addressFields.Set("country", NewStringValue("Canada"))

	address := NewCompositeValue(
		utils.TestLocation,
		"Address",
		common.CompositeKindStructure,
		addressFields,
		nil,
	)

	members := NewStringValueOrderedMap()
	members.Set("fname", NewStringValue(fmt.Sprintf("John%d", id)))
	members.Set("lname", NewStringValue("Doe"))
	members.Set("age", NewIntValueFromInt64(999))
	members.Set("status", NewStringValue("unknown"))
	members.Set("address", address)

	return NewCompositeValue(
		utils.TestLocation,
		"Person",
		common.CompositeKindStructure,
		members,
		nil,
	)
}

func newTestArrayValue(size int) *ArrayValue {
	values := make([]Value, size)

	for i := 0; i < size; i++ {
		values[i] = newTestLargeCompositeValue(i)
	}

	return NewArrayValueUnownedNonCopying(
		VariableSizedStaticType{
			Type: PrimitiveStaticTypeAnyStruct,
		},
		values...,
	)
}

func TestArrayDeferredDecoding(t *testing.T) {

	t.Parallel()

	t.Run("Simple array", func(t *testing.T) {
		array := newTestArrayValue(10)

		encoded, _, err := EncodeValue(array, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &ArrayValue{}, decoded)
		decodedArray := decoded.(*ArrayValue)

		// fields must not be loaded
		assert.NotNil(t, decodedArray.content)

		// Get an element from the array
		element := decodedArray.Get(nil, nil, NewIntValueFromInt64(1))
		assert.NotNil(t, element)

		require.IsType(t, &CompositeValue{}, element)
		compositeValue := element.(*CompositeValue)

		// This loading must be shallow. The content of the element must not be loaded.
		assert.NotNil(t, compositeValue.content)

		// raw content cache must be empty
		assert.Nil(t, decodedArray.content)

		// all the fields must be shallow loaded
		for _, element := range decodedArray.Elements() {
			assert.NotNil(t, element)
			require.IsType(t, &CompositeValue{}, element)
			compositeValue := element.(*CompositeValue)
			assert.NotNil(t, compositeValue.content)
		}
	})

	t.Run("Round trip - without loading", func(t *testing.T) {

		array := newTestArrayValue(2)

		// Encode
		encoded, _, err := EncodeValue(array, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		// Value must not be loaded. i.e: the content is available
		require.IsType(t, &ArrayValue{}, decoded)
		decodedArray := decoded.(*ArrayValue)
		assert.NotNil(t, decodedArray.content)

		// Re-encode the decoded value
		reEncoded, _, err := EncodeValue(decodedArray, nil, true, nil)
		require.NoError(t, err)

		require.Equal(t, encoded, reEncoded)

		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &ArrayValue{}, reDecoded)
		reDecodedArray := reDecoded.(*ArrayValue)

		reDecodedArray.ensureElementsLoaded()

		// Check the elements

		elements := reDecodedArray.Elements()
		require.Len(t, elements, 2)

		for i, element := range elements {
			require.IsType(t, &CompositeValue{}, element)
			elementVal := element.(*CompositeValue)

			decodeFieldValue, contains := elementVal.Fields().Get("fname")
			assert.True(t, contains)

			expected := NewStringValue(fmt.Sprintf("John%d", i))

			assert.Equal(t, expected, decodeFieldValue)
		}

	})

	t.Run("re-encoding", func(t *testing.T) {

		array := newTestArrayValue(2)

		// Encode
		encoded, _, err := EncodeValue(array, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		// Value must not be loaded. i.e: the content is available
		require.IsType(t, &ArrayValue{}, decoded)
		decodedArray := decoded.(*ArrayValue)
		assert.NotNil(t, decodedArray.content)

		type prepareCallback struct {
			value Value
			path  []string
		}

		var prepareCallbacks []prepareCallback

		callback := func(value Value, path []string) {
			prepareCallbacks = append(prepareCallbacks, prepareCallback{
				value: value,
				path:  path,
			})
		}

		// Re encode the decoded value
		reEncoded, _, err := EncodeValue(decodedArray, []string{}, true, callback)
		require.NoError(t, err)

		require.Equal(t, encoded, reEncoded)

		// Elements are not loaded, so they must not be encoded again.
		// i.e: Callback must be only called once.
		require.Len(t, prepareCallbacks, 1)
		assert.Equal(t, decoded, prepareCallbacks[0].value)
		assert.Equal(t, []string{}, prepareCallbacks[0].path)
	})

	t.Run("storable and modified", func(t *testing.T) {
		array := newTestArrayValue(2)

		encoded, _, err := EncodeValue(array, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &ArrayValue{}, decoded)
		decodedArray := decoded.(*ArrayValue)

		assert.False(t, decodedArray.IsModified())
		assert.True(t, decodedArray.IsStorable())

		// elements must not be loaded
		assert.Nil(t, decodedArray.values)
		assert.NotNil(t, decodedArray.content)
	})

	t.Run("Decode with V4", func(t *testing.T) {

		array := newTestArrayValue(2)

		// Encode
		encoded, _, err := EncodeValueV4(array, nil, true, nil)
		require.NoError(t, err)

		// Decode
		const encodingVersion = 4
		decoded, err := DecodeValueV4(encoded, &testOwner, nil, encodingVersion, nil)
		require.NoError(t, err)

		// Value must not be loaded. i.e: the content is available
		require.IsType(t, &ArrayValue{}, decoded)
		decodedArray := decoded.(*ArrayValue)
		assert.NotNil(t, decodedArray.content)

		decodedArray.ensureElementsLoaded()

		// Check the elements

		elements := decodedArray.Elements()
		require.Len(t, elements, 2)

		for i, element := range elements {
			require.IsType(t, &CompositeValue{}, element)
			elementVal := element.(*CompositeValue)

			decodeFieldValue, contains := elementVal.Fields().Get("fname")
			assert.True(t, contains)

			expected := NewStringValue(fmt.Sprintf("John%d", i))

			assert.Equal(t, expected, decodeFieldValue)
		}
	})
}

func BenchmarkArrayDeferredDecoding(b *testing.B) {

	const size = 1000

	encoded, _, err := EncodeValue(newTestArrayValue(size), nil, true, nil)
	require.NoError(b, err)

	b.Run("Simply decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)
		}
	})

	b.Run("Get first element", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)

			decodedArray := decoded.(*ArrayValue)
			element := decodedArray.Get(nil, nil, NewIntValueFromInt64(0))
			assert.NotNil(b, element)
		}
	})

	b.Run("Get last element", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)

			decodedArray := decoded.(*ArrayValue)
			element := decodedArray.Get(nil, nil, NewIntValueFromInt64(size-1))
			assert.NotNil(b, element)
		}
	})

	b.Run("Re-encode decoded", func(b *testing.B) {
		b.ReportAllocs()

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, err = EncodeValue(decoded, nil, true, nil)
			require.NoError(b, err)
		}
	})
}

func TestDictionaryDeferredDecoding(t *testing.T) {
	t.Parallel()

	t.Run("Simple dictionary", func(t *testing.T) {
		dictionary := newTestDictionaryValue(10)

		encoded, _, err := EncodeValue(dictionary, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &DictionaryValue{}, decoded)
		decodedDictionary := decoded.(*DictionaryValue)

		// entries must not be loaded
		assert.Nil(t, decodedDictionary.keys)
		assert.Nil(t, decodedDictionary.entries)
		assert.NotNil(t, decodedDictionary.content)

		// Get a value from the dictionary
		element := decodedDictionary.Get(nil, nil, NewStringValue("key5"))

		assert.Equal(
			t,
			NewSomeValueOwningNonCopying(NewStringValue("value5")),
			element,
		)

		// entries must be now loaded
		assert.NotNil(t, decodedDictionary.keys)
		assert.NotNil(t, decodedDictionary.entries)
		assert.Nil(t, decodedDictionary.content)
	})

	t.Run("Round trip - without loading", func(t *testing.T) {

		dictionary := newTestDictionaryValue(2)

		// Encode
		encoded, _, err := EncodeValue(dictionary, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &DictionaryValue{}, decoded)
		decodedDictionary := decoded.(*DictionaryValue)

		// entries must not be loaded
		assert.Nil(t, decodedDictionary.keys)
		assert.Nil(t, decodedDictionary.entries)
		assert.NotNil(t, decodedDictionary.content)

		// Re encode the decoded value
		reEncoded, _, err := EncodeValue(decodedDictionary, nil, true, nil)
		require.NoError(t, err)

		require.Equal(t, encoded, reEncoded)

		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &DictionaryValue{}, reDecoded)
		reDecodedDictionary := decoded.(*DictionaryValue)

		reDecodedDictionary.ensureLoaded()

		// Check the elements
		require.Equal(t, 2, reDecodedDictionary.Count())

		i := 0
		reDecodedDictionary.Entries().Foreach(func(key string, value Value) {
			assert.Equal(t, fmt.Sprintf("key%d", i), key)
			assert.Equal(t, NewStringValue(fmt.Sprintf("value%d", i)), value)
			i++
		})
	})

	t.Run("re-encoding", func(t *testing.T) {

		dictionary := newTestDictionaryValue(2)

		// Encode
		encoded, _, err := EncodeValue(dictionary, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &DictionaryValue{}, decoded)
		decodedDictionary := decoded.(*DictionaryValue)

		// entries must not be loaded
		assert.Nil(t, decodedDictionary.keys)
		assert.Nil(t, decodedDictionary.entries)
		assert.NotNil(t, decodedDictionary.content)

		type prepareCallback struct {
			value Value
			path  []string
		}

		var prepareCallbacks []prepareCallback

		callback := func(value Value, path []string) {
			valuePath := make([]string, len(path))
			copy(valuePath, path)

			prepareCallbacks = append(prepareCallbacks, prepareCallback{
				value: value,
				path:  valuePath,
			})
		}

		// Re encode the decoded value
		reEncoded, _, err := EncodeValue(decodedDictionary, []string{}, true, callback)
		require.NoError(t, err)

		require.Equal(t, encoded, reEncoded)

		// Entries are not loaded, so they must not be encoded again.
		// i.e: Callback must be only called once.
		require.Len(t, prepareCallbacks, 1)
		assert.Equal(t, decoded, prepareCallbacks[0].value)
		assert.Equal(t, []string{}, prepareCallbacks[0].path)
	})

	t.Run("deferred dictionary", func(t *testing.T) {
		const size = 2
		values := make([]Value, size*2)

		testResource := NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			NewStringValueOrderedMap(),
			nil,
		)

		for i := 0; i < size; i++ {
			values[i*2] = NewStringValue(fmt.Sprintf("key%d", i))
			values[i*2+1] = testResource
		}

		dictionary := NewDictionaryValueUnownedNonCopying(
			DictionaryStaticType{
				KeyType: PrimitiveStaticTypeString,
				ValueType: CompositeStaticType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "TestResource",
				},
			},
			values...,
		)

		// Encode
		encoded, _, err := EncodeValue(dictionary, nil, true, nil)
		require.NoError(t, err)

		// Decode
		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &DictionaryValue{}, decoded)
		decodedDictionary := decoded.(*DictionaryValue)

		// entries must not be loaded
		assert.Nil(t, decodedDictionary.keys)
		assert.Nil(t, decodedDictionary.entries)
		assert.NotNil(t, decodedDictionary.content)

		// Create a dummy interpreter for 'get' function
		inter, err := NewInterpreter(nil, utils.TestLocation)
		require.NoError(t, err)
		inter.storageReadHandler = func(
			inter *Interpreter,
			storageAddress common.Address,
			key string,
			deferred bool,
		) OptionalValue {
			if key == joinPath([]string{dictionaryValuePathPrefix, "key0"}) {
				return NewSomeValueOwningNonCopying(testResource)
			}

			return NilValue{}
		}

		// Get a value from the dictionary
		element := decodedDictionary.Get(inter, nil, NewStringValue("key0"))

		assert.Equal(t, NewSomeValueOwningNonCopying(testResource), element)

		// entries must be now loaded
		assert.NotNil(t, decodedDictionary.keys)
		assert.NotNil(t, decodedDictionary.entries)
		assert.Nil(t, decodedDictionary.content)
	})

	t.Run("storable and modified", func(t *testing.T) {
		dictionary := newTestDictionaryValue(2)

		encoded, _, err := EncodeValue(dictionary, nil, true, nil)
		require.NoError(t, err)

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &DictionaryValue{}, decoded)
		decodedDictionary := decoded.(*DictionaryValue)

		assert.False(t, decodedDictionary.IsModified())
		assert.True(t, decodedDictionary.IsStorable())

		// entries must not be loaded
		assert.Nil(t, decodedDictionary.keys)
		assert.Nil(t, decodedDictionary.entries)
		assert.NotNil(t, decodedDictionary.content)
	})

	t.Run("Decode with V4", func(t *testing.T) {

		dictionary := newTestDictionaryValue(2)

		// Encode
		encoded, _, err := EncodeValueV4(dictionary, nil, true, nil)
		require.NoError(t, err)

		// Decode
		const encodingVersion = 4
		decoded, err := DecodeValueV4(encoded, &testOwner, nil, encodingVersion, nil)
		require.NoError(t, err)

		require.IsType(t, &DictionaryValue{}, decoded)
		decodedDictionary := decoded.(*DictionaryValue)

		decodedDictionary.ensureLoaded()

		// Check the elements
		require.Equal(t, 2, decodedDictionary.Count())

		i := 0
		decodedDictionary.Entries().Foreach(func(key string, value Value) {
			assert.Equal(t, fmt.Sprintf("key%d", i), key)
			assert.Equal(t, NewStringValue(fmt.Sprintf("value%d", i)), value)
			i++
		})
	})
}

func newTestDictionaryValue(size int) *DictionaryValue {
	values := make([]Value, size*2)

	for i := 0; i < size; i++ {
		values[i*2] = NewStringValue(fmt.Sprintf("key%d", i))
		values[i*2+1] = NewStringValue(fmt.Sprintf("value%d", i))
	}

	return NewDictionaryValueUnownedNonCopying(
		DictionaryStaticType{
			KeyType:   PrimitiveStaticTypeString,
			ValueType: PrimitiveStaticTypeString,
		},
		values...,
	)
}

func BenchmarkDictionaryDeferredDecoding(b *testing.B) {

	const size = 100

	encoded, _, err := EncodeValue(newTestDictionaryValue(size), nil, true, nil)
	require.NoError(b, err)

	b.Run("Simply decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)
		}
	})

	b.Run("Get value", func(b *testing.B) {
		b.ReportAllocs()

		key := NewStringValue(fmt.Sprintf("key%d", size/2))

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)

			decodedDictionary := decoded.(*DictionaryValue)
			value := decodedDictionary.Get(nil, nil, key)
			assert.NotNil(b, value)
		}
	})

	b.Run("Re-encode decoded", func(b *testing.B) {
		b.ReportAllocs()

		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
		require.NoError(b, err)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _, err = EncodeValue(decoded, nil, true, nil)
			require.NoError(b, err)
		}
	})
}
