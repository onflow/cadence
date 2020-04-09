package interpreter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/common"
)

func testEncodeDecode(t *testing.T, tests map[string]Value) {
	owner := common.BytesToAddress([]byte{0x42})

	for name, value := range tests {
		t.Run(name, func(t *testing.T) {

			value.SetOwner(&owner)

			encoded, err := EncodeValue(value)
			require.NoError(t, err)

			decoded, err := DecodeValue(encoded, &owner)
			require.NoError(t, err)

			require.Equal(t, value, decoded)
		})
	}
}

func TestEncodeDecodeBool(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"true":  BoolValue(true),
			"false": BoolValue(false),
		},
	)
}

func TestEncodeDecodeString(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     NewStringValue(""),
			"non-empty": NewStringValue("foo"),
		},
	)
}

func TestEncodeDecodeArray(t *testing.T) {

	testEncodeDecode(t,
		map[string]Value{
			"empty": NewArrayValueUnownedNonCopying(),
			"string and bool": NewArrayValueUnownedNonCopying(
				NewStringValue("test"),
				BoolValue(true),
			),
		},
	)
}

func TestEncodeDecodeDictionary(t *testing.T) {

	testEncodeDecode(t,
		map[string]Value{
			"empty": NewDictionaryValueUnownedNonCopying(),
			"non-empty": NewDictionaryValueUnownedNonCopying(
				NewStringValue("test"), NewArrayValueUnownedNonCopying(),
				BoolValue(true), BoolValue(false),
				NewStringValue("foo"), NewStringValue("bar"),
			),
		},
	)
}

func TestEncodeDecodeComposite(t *testing.T) {

	// TODO: location

	testEncodeDecode(t,
		map[string]Value{
			"empty structure": &CompositeValue{
				TypeID: "TestStruct",
				Kind:   common.CompositeKindStructure,
				Fields: map[string]Value{},
			},
			"non-empty resource": &CompositeValue{
				TypeID: "TestResource",
				Kind:   common.CompositeKindResource,
				Fields: map[string]Value{
					"true":   BoolValue(true),
					"string": NewStringValue("test"),
				},
			},
		},
	)
}
