package interpreter

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/ast"
	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/sema"
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

func TestEncodeDecodeNilValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     NilValue{},
			"non-empty": NilValue{},
		},
	)
}

func TestEncodeDecodeVoidValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     VoidValue{},
			"non-empty": VoidValue{},
		},
	)
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
	testEncodeDecode(t,
		map[string]Value{
			"empty structure": &CompositeValue{
				TypeID:   "TestStruct",
				Kind:     common.CompositeKindStructure,
				Fields:   map[string]Value{},
				Location: ast.StringLocation(""),
			},
			"non-empty resource": &CompositeValue{
				TypeID: "TestResource",
				Kind:   common.CompositeKindResource,
				Fields: map[string]Value{
					"true":   BoolValue(true),
					"string": NewStringValue("test"),
				},
				Location: ast.StringLocation("0:1"),
			},
			"non-empty resource + address location": &CompositeValue{
				TypeID: "TestResource",
				Kind:   common.CompositeKindResource,
				Fields: map[string]Value{
					"true":   BoolValue(true),
					"string": NewStringValue("test"),
				},
				Location: ast.AddressLocation{0x40},
			},
		},
	)
}

func TestEncodeDecodeIntValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     IntValue{Int: big.NewInt(0)},
			"non-empty": IntValue{Int: big.NewInt(64)},
			"signed":    IntValue{Int: big.NewInt(-64)},
		},
	)
}

func TestEncodeDecodeInt8Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":        Int8Value(0),
			"non-empty":    Int8Value(64),
			"boundary-max": Int8Value(math.MaxInt8),
			"boundary-min": Int8Value(math.MinInt8),
			"signed":       Int8Value(math.MinInt8),
		},
	)
}

func TestEncodeDecodeInt16Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     Int16Value(0),
			"non-empty": Int16Value(128),
			"boundary":  Int16Value(math.MaxInt16),
			"signed":    Int16Value(math.MinInt16),
		},
	)
}

func TestEncodeDecodeInt32Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     Int32Value(0),
			"non-empty": Int32Value(128),
			"boundary":  Int32Value(math.MaxInt32),
			"signed":    Int32Value(math.MinInt32),
		},
	)
}

func TestEncodeDecodeInt64Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     Int64Value(0),
			"non-empty": Int64Value(128),
			"boundary":  Int64Value(math.MaxInt64),
			"signed":    Int64Value(math.MinInt64),
		},
	)
}

func TestEncodeDecodeUIntValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     UIntValue{Int: big.NewInt(0)},
			"non-empty": UIntValue{Int: big.NewInt(64)},
			"signed":    UIntValue{Int: big.NewInt(-64)},
		},
	)
}

func TestEncodeDecodeInt128Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     Int128Value{Int: big.NewInt(0)},
			"non-empty": Int128Value{Int: big.NewInt(64)},
			"signed":    Int128Value{Int: big.NewInt(-64)},
		},
	)
}

func TestEncodeDecodeInt256Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     Int256Value{Int: big.NewInt(0)},
			"non-empty": Int256Value{Int: big.NewInt(64)},
			"signed":    Int256Value{Int: big.NewInt(-64)},
		},
	)
}

func TestEncodeDecodeUInt8Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     UInt8Value(0),
			"non-empty": UInt8Value(64),
			"boundary":  UInt8Value(127),
		},
	)
}

func TestEncodeDecodeUInt16Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     UInt16Value(0),
			"non-empty": UInt16Value(128),
			"boundary":  UInt16Value(1000),
		},
	)
}

func TestEncodeDecodeUInt32Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     UInt32Value(0),
			"non-empty": UInt32Value(128),
		},
	)
}

func TestEncodeDecodeUInt64Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     UInt64Value(0),
			"non-empty": UInt64Value(128),
			"boundary":  UInt64Value(128),
		},
	)
}

func TestEncodeDecodeUInt128Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     UInt128Value{Int: big.NewInt(0)},
			"non-empty": UInt128Value{Int: big.NewInt(64)},
			"signed":    UInt128Value{Int: big.NewInt(-64)},
		},
	)
}

func TestEncodeDecodeUInt256Value(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     UInt256Value{Int: big.NewInt(0)},
			"non-empty": UInt256Value{Int: big.NewInt(64)},
			"signed":    UInt256Value{Int: big.NewInt(-64)},
		},
	)
}
func TestEncodeDecodeSomeValue(t *testing.T) {
	owner := common.BytesToAddress([]byte{0x42})
	testEncodeDecode(t,
		map[string]Value{
			"empty": &SomeValue{Value: NilValue{}, Owner: &owner},
			"non-empty": &SomeValue{
				Owner: &owner,
				Value: NewStringValue("test"),
			},
		},
	)
}

func TestEncodeDecodeStorageReferenceValue(t *testing.T) {
	owner := common.BytesToAddress([]byte{0x42})
	testEncodeDecode(t,
		map[string]Value{
			// "empty": &StorageReferenceValue{},
			"non-empty": &StorageReferenceValue{
				Authorized:           true,
				TargetKey:            "",
				TargetStorageAddress: common.BytesToAddress([]byte{0x042}),
				Owner:                &owner,
			},
		},
	)
}

func TestEncodeDecodeEphemeralReferenceValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty": &EphemeralReferenceValue{
				Authorized: true,
				Value:      NilValue{},
			},
			"non-empty": &EphemeralReferenceValue{
				Authorized: true,
				Value:      NewStringValue("test"),
			},
		},
	)
}

func TestEncodeDecodeAddressValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     AddressValue{},
			"non-empty": AddressValue{0x42},
		},
	)
}

var emptyPathValue = PathValue{Domain: common.PathDomainUnknown, Identifier: ""}
var nonEmptyPathValue = PathValue{Domain: common.PathDomainPublic, Identifier: "testing"}

func TestEncodeDecodePathValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty":     &emptyPathValue,
			"non-empty": &nonEmptyPathValue,
		},
	)
}

func TestEncodeDecodeCapabilityValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty": &CapabilityValue{
				Address: NewAddressValueFromBytes([]byte{0x00}),
				Path:    emptyPathValue,
			},
			"non-empty": &CapabilityValue{
				Address: NewAddressValueFromBytes([]byte{0x42}),
				Path:    nonEmptyPathValue,
			},
		},
	)
}

func TestEncodeDecodeLinkValue(t *testing.T) {
	testEncodeDecode(t,
		map[string]Value{
			"empty": &LinkValue{
				TargetPath: nonEmptyPathValue,
				Type:       CompositeStaticType{TypeID: sema.TypeID("test")},
			},
			"non-empty+composite": &LinkValue{
				TargetPath: nonEmptyPathValue,
				Type: CompositeStaticType{
					TypeID:   "SimpleStruct",
					Location: ast.StringLocation("0:0"),
				},
			},
			"non-empty+interface": &LinkValue{
				TargetPath: nonEmptyPathValue,
				Type: InterfaceStaticType{
					TypeID:   "SimpleInterface",
					Location: ast.StringLocation("0:0"),
				},
			},
			"non-empty+variable-sized": &LinkValue{
				TargetPath: nonEmptyPathValue,
				Type: VariableSizedStaticType{
					Type: TypeStaticType{},
				},
			},
		},
	)
}
