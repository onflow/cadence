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

		stringValue := NewStringValue("hello")
		stringValue.modified = false

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
		assert.Equal(t, stringValue, decodeFieldValue)

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
}

var newTestLargeCompositeValue = func(id int) *CompositeValue {
	addressFields := NewStringValueOrderedMap()
	addressFields.Set("street", NewStringValue(fmt.Sprintf("No: %d", id)))
	addressFields.Set("city", NewStringValue("Vancouver"))
	addressFields.Set("state", NewStringValue("BC"))
	addressFields.Set("country", NewStringValue("Canada"))

	address := NewCompositeValue(
		utils.TestLocation,
		"Address",
		common.CompositeKindResource,
		addressFields,
		nil,
	)

	members := NewStringValueOrderedMap()
	members.Set("fname", NewStringValue("John"))
	members.Set("lname", NewStringValue("Doe"))
	members.Set("age", NewIntValueFromInt64(999))
	members.Set("status", NewStringValue("unknown"))
	members.Set("address", address)

	return NewCompositeValue(
		utils.TestLocation,
		"TestResource",
		common.CompositeKindResource,
		members,
		nil,
	)
}
