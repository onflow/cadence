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

package interpreter_test

// TODO:
//
//func TestCompositeDeferredDecoding(t *testing.T) {
//
//	t.Parallel()
//
//	t.Run("Simple composite", func(t *testing.T) {
//		t.Parallel()
//
//		members := NewStringValueOrderedMap()
//		members.Set("a", NewStringValue("hello"))
//		members.Set("b", BoolValue(true))
//
//		value := NewCompositeValue(
//			utils.TestLocation,
//			"TestResource",
//			common.CompositeKindResource,
//			members,
//			nil,
//		)
//
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//
//		// Value must not be loaded. i.e: the content is available
//		assert.NotNil(t, compositeValue.content)
//
//		// The meta-info and fields raw content are not loaded yet
//		assert.Nil(t, compositeValue.fieldsContent)
//		assert.Empty(t, compositeValue.location)
//		assert.Empty(t, compositeValue.QualifiedIdentifier)
//		assert.Equal(t, common.CompositeKindUnknown, compositeValue.kind)
//
//		// Use the Getters and see whether the meta-info are loaded
//		assert.Equal(t, value.Location(), compositeValue.Location())
//		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
//		assert.Equal(t, value.Kind(), compositeValue.Kind())
//
//		// Now the content must be cleared
//		assert.Nil(t, compositeValue.content)
//
//		// And the fields raw content must be available
//		assert.NotNil(t, compositeValue.fieldsContent)
//
//		// Check all the fields using getters
//
//		decodedFields := compositeValue.Fields
//		require.Equal(t, 2, decodedFields.Len())
//
//		decodeFieldValue, contains := decodedFields.Get("a")
//		assert.True(t, contains)
//		assert.Equal(t, NewStringValue("hello"), decodeFieldValue)
//
//		decodeFieldValue, contains = decodedFields.Get("b")
//		assert.True(t, contains)
//		assert.Equal(t, BoolValue(true), decodeFieldValue)
//
//		// Once all the fields are loaded, the fields raw content must be cleared
//		assert.Nil(t, compositeValue.fieldsContent)
//	})
//
//	t.Run("Nested composite", func(t *testing.T) {
//		t.Parallel()
//
//		value := newTestLargeCompositeValue(0)
//
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//
//		address, ok := compositeValue.Fields.Get("address")
//		assert.True(t, ok)
//
//		require.IsType(t, &CompositeValue{}, address)
//		nestedCompositeValue := address.(*CompositeValue)
//
//		// Inner composite value must not be loaded
//		assert.NotNil(t, nestedCompositeValue.content)
//	})
//
//	t.Run("Field update", func(t *testing.T) {
//		t.Parallel()
//
//		value := newTestLargeCompositeValue(0)
//
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//
//		newValue := NewStringValue("green")
//		compositeValue.SetMember(nil, nil, "status", newValue)
//
//		// Composite value must be loaded
//		assert.Nil(t, compositeValue.content)
//
//		// check updated value
//		fieldValue, contains := compositeValue.Fields.Get("status")
//		assert.True(t, contains)
//		assert.Equal(t, newValue, fieldValue)
//	})
//
//	t.Run("Round trip - without loading", func(t *testing.T) {
//		t.Parallel()
//
//		members := NewStringValueOrderedMap()
//		members.Set("a", NewStringValue("hello"))
//		members.Set("b", BoolValue(true))
//
//		value := NewCompositeValue(
//			utils.TestLocation,
//			"TestResource",
//			common.CompositeKindResource,
//			members,
//			nil,
//		)
//
//		// Encode
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		// Decode
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		// Value must not be loaded. i.e: the content is available
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//		assert.NotNil(t, compositeValue.content)
//
//		// Re encode the decoded value
//		reEncoded, _, err := EncodeValue(decoded, nil, true, nil)
//		require.NoError(t, err)
//
//		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		require.IsType(t, &CompositeValue{}, reDecoded)
//		compositeValue = reDecoded.(*CompositeValue)
//
//		compositeValue.ensureFieldsLoaded()
//
//		// Check the meta info
//		assert.Equal(t, value.Location(), compositeValue.Location())
//		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
//		assert.Equal(t, value.Kind(), compositeValue.Kind())
//
//		// Check the fields
//
//		decodedFields := compositeValue.Fields
//		require.Equal(t, 2, decodedFields.Len())
//
//		decodeFieldValue, contains := decodedFields.Get("a")
//		assert.True(t, contains)
//		assert.Equal(t, NewStringValue("hello"), decodeFieldValue)
//
//		decodeFieldValue, contains = decodedFields.Get("b")
//		assert.True(t, contains)
//		assert.Equal(t, BoolValue(true), decodeFieldValue)
//	})
//
//	t.Run("Round trip - partially loaded", func(t *testing.T) {
//		t.Parallel()
//
//		members := NewStringValueOrderedMap()
//		members.Set("a", NewStringValue("hello"))
//		members.Set("b", BoolValue(true))
//
//		value := NewCompositeValue(
//			utils.TestLocation,
//			"TestResource",
//			common.CompositeKindResource,
//			members,
//			nil,
//		)
//
//		// Encode
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		// Decode
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		// Partially loaded the value.
//
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//		// This will only load the meta info, but not the fields
//		compositeValue.QualifiedIdentifier()
//
//		assert.Nil(t, compositeValue.content)
//		assert.NotNil(t, compositeValue.fieldsContent)
//
//		// Re encode the decoded value
//		reEncoded, _, err := EncodeValue(decoded, nil, true, nil)
//		require.NoError(t, err)
//
//		// Decode back the value
//		reDecoded, err := DecodeValue(reEncoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		require.IsType(t, &CompositeValue{}, reDecoded)
//		compositeValue = reDecoded.(*CompositeValue)
//
//		compositeValue.ensureFieldsLoaded()
//
//		// Check the meta info
//		assert.Equal(t, value.Location(), compositeValue.Location())
//		assert.Equal(t, value.QualifiedIdentifier(), compositeValue.QualifiedIdentifier())
//		assert.Equal(t, value.Kind(), compositeValue.Kind())
//
//		// Check the fields
//
//		decodedFields := compositeValue.Fields
//		require.Equal(t, 2, decodedFields.Len())
//
//		decodeFieldValue, contains := decodedFields.Get("a")
//		assert.True(t, contains)
//		assert.Equal(t, NewStringValue("hello"), decodeFieldValue)
//
//		decodeFieldValue, contains = decodedFields.Get("b")
//		assert.True(t, contains)
//		assert.Equal(t, BoolValue(true), decodeFieldValue)
//	})
//
//	t.Run("callback", func(t *testing.T) {
//		t.Parallel()
//
//		stringValue := NewStringValue("hello")
//
//		members := NewStringValueOrderedMap()
//		members.Set("a", stringValue)
//		members.Set("b", BoolValue(true))
//
//		value := NewCompositeValue(
//			utils.TestLocation,
//			"TestResource",
//			common.CompositeKindResource,
//			members,
//			nil,
//		)
//
//		// Encode
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		// Decode
//
//		type decodeCallback struct {
//			value interface{}
//			path  []string
//		}
//
//		var decodeCallbacks []decodeCallback
//		callback := func(value interface{}, path []string) {
//			valuePath := make([]string, len(path))
//			copy(valuePath, path)
//
//			decodeCallbacks = append(decodeCallbacks, decodeCallback{
//				value: value,
//				path:  valuePath,
//			})
//		}
//
//		decoded, err := DecodeValue(encoded, &testOwner, []string{}, CurrentEncodingVersion, callback)
//		require.NoError(t, err)
//
//		// Callback must be only called once
//		require.Len(t, decodeCallbacks, 1)
//		assert.Equal(t, decoded, decodeCallbacks[0].value)
//		assert.Equal(t, []string{}, decodeCallbacks[0].path)
//
//		// Load the meta info, but not the fields
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//		compositeValue.QualifiedIdentifier()
//
//		require.Len(t, decodeCallbacks, 1)
//		assert.Equal(t, decoded, decodeCallbacks[0].value)
//		assert.Equal(t, []string{}, decodeCallbacks[0].path)
//
//		// Load fields
//		compositeValue.Fields
//
//		// Callback must now have all three values
//		require.Len(t, decodeCallbacks, 3)
//
//		assert.Equal(t, decoded, decodeCallbacks[0].value)
//		assert.Equal(t, []string{}, decodeCallbacks[0].path)
//
//		assert.Equal(t, stringValue, decodeCallbacks[1].value)
//		assert.Equal(t, []string{"a"}, decodeCallbacks[1].path)
//
//		assert.Equal(t, BoolValue(true), decodeCallbacks[2].value)
//		assert.Equal(t, []string{"b"}, decodeCallbacks[2].path)
//	})
//
//	t.Run("re-encoding", func(t *testing.T) {
//		t.Parallel()
//
//		members := NewStringValueOrderedMap()
//		members.Set("a", NewStringValue("hello"))
//		members.Set("b", BoolValue(true))
//
//		value := NewCompositeValue(
//			utils.TestLocation,
//			"TestResource",
//			common.CompositeKindResource,
//			members,
//			nil,
//		)
//
//		// Encode
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		// Decode
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		// Partially loaded the value.
//
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//		// This will only load the meta info, but not the fields
//		compositeValue.QualifiedIdentifier()
//
//		assert.Nil(t, compositeValue.content)
//		assert.NotNil(t, compositeValue.fieldsContent)
//
//		// Re encode the decoded value
//		type encodeCallback struct {
//			value Value
//			path  []string
//		}
//
//		var encodeCallbacks []encodeCallback
//		callback := func(value Value, path []string) {
//			valuePath := make([]string, len(path))
//			copy(valuePath, path)
//
//			encodeCallbacks = append(encodeCallbacks, encodeCallback{
//				value: value,
//				path:  valuePath,
//			})
//		}
//
//		_, _, err = EncodeValue(decoded, nil, true, callback)
//		require.NoError(t, err)
//
//		// Elements are not loaded, so they must not be encoded again.
//		// i.e: Callback must be only called once.
//		require.Len(t, encodeCallbacks, 1)
//		assert.Equal(t, decoded, encodeCallbacks[0].value)
//		assert.Equal(t, []string{}, encodeCallbacks[0].path)
//	})
//
//	t.Run("storable and modified", func(t *testing.T) {
//		t.Parallel()
//
//		members := NewStringValueOrderedMap()
//		members.Set("a", NewStringValue("hello"))
//		members.Set("b", BoolValue(true))
//
//		value := NewCompositeValue(
//			utils.TestLocation,
//			"TestStruct",
//			common.CompositeKindStructure,
//			members,
//			nil,
//		)
//
//		encoded, _, err := EncodeValue(value, nil, true, nil)
//		require.NoError(t, err)
//
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(t, err)
//
//		require.IsType(t, &CompositeValue{}, decoded)
//		compositeValue := decoded.(*CompositeValue)
//
//		assert.True(t, compositeValue.IsStorable())
//
//		// fields must not be loaded
//		assert.Nil(t, compositeValue.fields)
//		assert.Nil(t, compositeValue.content)
//		assert.NotNil(t, compositeValue.fieldsContent)
//	})
//}
//
//func BenchmarkCompositeDeferredDecoding(b *testing.B) {
//
//	encoded, _, err := EncodeValue(newTestLargeCompositeValue(0), nil, true, nil)
//	require.NoError(b, err)
//
//	b.Run("Simply decode", func(b *testing.B) {
//		b.ReportAllocs()
//		b.ResetTimer()
//
//		for i := 0; i < b.N; i++ {
//			_, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//			require.NoError(b, err)
//		}
//	})
//
//	b.Run("Access identifier", func(b *testing.B) {
//		b.ReportAllocs()
//		b.ResetTimer()
//
//		for i := 0; i < b.N; i++ {
//			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//			require.NoError(b, err)
//
//			composite := decoded.(*CompositeValue)
//			composite.QualifiedIdentifier()
//		}
//	})
//
//	b.Run("Access field", func(b *testing.B) {
//		b.ReportAllocs()
//		b.ResetTimer()
//
//		for i := 0; i < b.N; i++ {
//			decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//			require.NoError(b, err)
//
//			composite := decoded.(*CompositeValue)
//			_, ok := composite.Fields.Get("fname")
//			require.True(b, ok)
//		}
//	})
//
//	b.Run("Re-encode decoded", func(b *testing.B) {
//		b.ReportAllocs()
//
//		decoded, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
//		require.NoError(b, err)
//
//		b.ResetTimer()
//
//		for i := 0; i < b.N; i++ {
//			_, _, err = EncodeValue(decoded, nil, true, nil)
//			require.NoError(b, err)
//		}
//	})
//}
//
//func newTestLargeCompositeValue(id int) *CompositeValue {
//	addressFields := NewStringValueOrderedMap()
//	addressFields.Set("street", NewStringValue(fmt.Sprintf("No: %d", id)))
//	addressFields.Set("city", NewStringValue("Vancouver"))
//	addressFields.Set("state", NewStringValue("BC"))
//	addressFields.Set("country", NewStringValue("Canada"))
//
//	address := NewCompositeValue(
//		utils.TestLocation,
//		"Address",
//		common.CompositeKindStructure,
//		addressFields,
//		nil,
//	)
//
//	members := NewStringValueOrderedMap()
//	members.Set("fname", NewStringValue(fmt.Sprintf("John%d", id)))
//	members.Set("lname", NewStringValue("Doe"))
//	members.Set("age", NewIntValueFromInt64(999))
//	members.Set("status", NewStringValue("unknown"))
//	members.Set("address", address)
//
//	return NewCompositeValue(
//		utils.TestLocation,
//		"Person",
//		common.CompositeKindStructure,
//		members,
//		nil,
//	)
//}
