package interpreter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestDeferredDecoding(t *testing.T) {

	t.Parallel()

	stringValue := NewStringValue("")
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

	// Check the content is available
	assert.NotNil(t, compositeValue.content)

	// And the meta-info and fields raw content are not loaded yet
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
}

func BenchmarkName(b *testing.B) {

	b.ReportAllocs()

	encodedValues := make([][]byte, len(valueArray))
	for i, value := range valueArray {
		encoded, _, err := EncodeValue(value, nil, true, nil)
		require.NoError(b, err)

		encodedValues[i] = encoded
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, encoded := range encodedValues {
			_, err := DecodeValue(encoded, &testOwner, nil, CurrentEncodingVersion, nil)
			require.NoError(b, err)
		}
	}
}

const SIZE = 100_000

var valueArray = func() []Value {

	values := make([]Value, SIZE)

	for i := 0; i < SIZE; i++ {

		addressFields := NewStringValueOrderedMap()
		addressFields.Set("street", NewStringValue(fmt.Sprintf("No: %d", i)))
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

		values[i] = NewCompositeValue(
			utils.TestLocation,
			"TestResource",
			common.CompositeKindResource,
			members,
			nil,
		)
	}

	return values
}()
