package xdr_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence"
	"github.com/dapperlabs/cadence/encoding/xdr"
)

type encodeTest struct {
	name string
	typ  cadence.Type
	val  cadence.Value
}

func TestEncodeVoid(t *testing.T) {
	testEncode(t, cadence.VoidType{}, cadence.Void{})
}

func TestEncodeString(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"EmptyString",
			cadence.StringType{},
			cadence.NewString(""),
		},
		{
			"SimpleString",
			cadence.StringType{},
			cadence.NewString("abcdefg"),
		},
	}...)
}

func TestEncodeOptional(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Nil",
			cadence.OptionalType{Type: nil},
			cadence.NewOptional(nil),
		},
		{
			"SomeString",
			cadence.OptionalType{Type: cadence.StringType{}},
			cadence.NewOptional(cadence.NewString("abcdefg")),
		},
	}...)
}

func TestEncodeBool(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"True",
			cadence.BoolType{},
			cadence.NewBool(true),
		},
		{
			"False",
			cadence.BoolType{},
			cadence.NewBool(false),
		},
	}...)
}

func TestEncodeBytes(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"EmptyBytes",
			cadence.BytesType{},
			cadence.NewBytes([]byte{}),
		},
		{
			"SimpleBytes",
			cadence.BytesType{},
			cadence.NewBytes([]byte{1, 2, 3, 4, 5}),
		},
	}...)
}

func TestEncodeAddress(t *testing.T) {
	testEncode(t, cadence.AddressType{}, cadence.NewAddress([20]byte{1, 2, 3, 4, 5}))
}

func TestEncodeInt(t *testing.T) {
	x := big.NewInt(0).SetUint64(math.MaxUint64)
	x = x.Mul(x, big.NewInt(2))

	largerThanMaxUint64 := encodeTest{
		"LargerThanMaxUint64",
		cadence.IntType{},
		cadence.NewIntFromBig(x),
	}

	testAllEncode(t, []encodeTest{
		{
			"Negative",
			cadence.IntType{},
			cadence.NewInt(-42),
		},
		{
			"Zero",
			cadence.IntType{},
			cadence.NewInt(0),
		},
		{
			"Positive",
			cadence.IntType{},
			cadence.NewInt(42),
		},
		largerThanMaxUint64,
	}...)
}

func TestEncodeInt8(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.Int8Type{},
			cadence.NewInt8(math.MinInt8),
		},
		{
			"Zero",
			cadence.Int8Type{},
			cadence.NewInt8(0),
		},
		{
			"Max",
			cadence.Int8Type{},
			cadence.NewInt8(math.MaxInt8),
		},
	}...)
}

func TestEncodeInt16(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.Int16Type{},
			cadence.NewInt16(math.MinInt16),
		},
		{
			"Zero",
			cadence.Int16Type{},
			cadence.NewInt16(0),
		},
		{
			"Max",
			cadence.Int16Type{},
			cadence.NewInt16(math.MaxInt16),
		},
	}...)
}

func TestEncodeInt32(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.Int32Type{},
			cadence.NewInt32(math.MinInt32),
		},
		{
			"Zero",
			cadence.Int32Type{},
			cadence.NewInt32(0),
		},
		{
			"Max",
			cadence.Int32Type{},
			cadence.NewInt32(math.MaxInt32),
		},
	}...)
}

func TestEncodeInt64(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.Int64Type{},
			cadence.NewInt64(math.MinInt64),
		},
		{
			"Zero",
			cadence.Int64Type{},
			cadence.NewInt64(0),
		},
		{
			"Max",
			cadence.Int64Type{},
			cadence.NewInt64(math.MaxInt64),
		},
	}...)
}

func TestEncodeUint8(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.UInt8Type{},
			cadence.NewUInt8(0),
		},
		{
			"Max",
			cadence.UInt8Type{},
			cadence.NewUInt8(math.MaxUint8),
		},
	}...)
}

func TestEncodeUint16(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.UInt16Type{},
			cadence.NewUInt16(0),
		},
		{
			"Max",
			cadence.UInt16Type{},
			cadence.NewUInt16(math.MaxUint8),
		},
	}...)
}

func TestEncodeUint32(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.UInt32Type{},
			cadence.NewUInt32(0),
		},
		{
			"Max",
			cadence.UInt32Type{},
			cadence.NewUInt32(math.MaxUint32),
		},
	}...)
}

func TestEncodeUint64(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.UInt64Type{},
			cadence.NewUInt64(0),
		},
		{
			"Max",
			cadence.UInt64Type{},
			cadence.NewUInt64(math.MaxUint64),
		},
	}...)
}

func TestEncodeVariableSizedArray(t *testing.T) {
	emptyArray := encodeTest{
		"EmptyArray",
		cadence.VariableSizedArrayType{
			ElementType: cadence.IntType{},
		},
		cadence.NewArray([]cadence.Value{}),
	}

	intArray := encodeTest{
		"IntArray",
		cadence.VariableSizedArrayType{
			ElementType: cadence.IntType{},
		},
		cadence.NewArray([]cadence.Value{
			cadence.NewInt(1),
			cadence.NewInt(2),
			cadence.NewInt(3),
		}),
	}

	compositeArray := encodeTest{
		"CompositeArray",
		cadence.VariableSizedArrayType{
			ElementType: cadence.CompositeType{
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.StringType{},
					},
					{
						Identifier: "b",
						Type:       cadence.IntType{},
					},
				},
			},
		},
		cadence.NewArray([]cadence.Value{
			cadence.NewComposite([]cadence.Value{
				cadence.NewString("a"),
				cadence.NewInt(1),
			}),
			cadence.NewComposite([]cadence.Value{
				cadence.NewString("b"),
				cadence.NewInt(1),
			}),
			cadence.NewComposite([]cadence.Value{
				cadence.NewString("c"),
				cadence.NewInt(1),
			}),
		}),
	}

	testAllEncode(t,
		emptyArray,
		intArray,
		compositeArray,
	)
}

func TestEncodeConstantSizedArray(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"EmptyArray",
			cadence.ConstantSizedArrayType{
				Size:        0,
				ElementType: cadence.IntType{},
			},
			cadence.NewArray([]cadence.Value{}),
		},
		{
			"IntArray",
			cadence.ConstantSizedArrayType{
				Size:        3,
				ElementType: cadence.IntType{},
			},
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(1),
				cadence.NewInt(2),
				cadence.NewInt(3),
			}),
		},
	}...)
}

func TestEncodeDictionary(t *testing.T) {
	simpleDict := encodeTest{
		"SimpleDict",
		cadence.DictionaryType{
			KeyType:     cadence.StringType{},
			ElementType: cadence.IntType{},
		},
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.NewString("a"),
				Value: cadence.NewInt(1),
			},
			{
				Key:   cadence.NewString("b"),
				Value: cadence.NewInt(2),
			},
			{
				Key:   cadence.NewString("c"),
				Value: cadence.NewInt(3),
			},
		}),
	}

	nestedDict := encodeTest{
		"NestedDict",
		cadence.DictionaryType{
			KeyType: cadence.StringType{},
			ElementType: cadence.DictionaryType{
				KeyType:     cadence.StringType{},
				ElementType: cadence.IntType{},
			},
		},
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.NewString("a"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.NewString("1"),
						Value: cadence.NewInt(1),
					},
				}),
			},
			{
				Key: cadence.NewString("b"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.NewString("2"),
						Value: cadence.NewInt(2),
					},
				}),
			},
			{
				Key: cadence.NewString("c"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.NewString("3"),
						Value: cadence.NewInt(3),
					},
				}),
			},
		}),
	}

	compositeDict := encodeTest{
		"CompositeDict",
		cadence.DictionaryType{
			KeyType: cadence.StringType{},
			ElementType: cadence.CompositeType{
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.StringType{},
					},
					{
						Identifier: "b",
						Type:       cadence.IntType{},
					},
				},
			},
		},
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.NewString("a"),
				Value: cadence.NewComposite([]cadence.Value{
					cadence.NewString("a"),
					cadence.NewInt(1),
				}),
			},
			{
				Key: cadence.NewString("b"),
				Value: cadence.NewComposite([]cadence.Value{
					cadence.NewString("b"),
					cadence.NewInt(2),
				}),
			},
			{
				Key: cadence.NewString("c"),
				Value: cadence.NewComposite([]cadence.Value{
					cadence.NewString("c"),
					cadence.NewInt(3),
				}),
			},
		}),
	}

	testAllEncode(t,
		simpleDict,
		nestedDict,
		compositeDict,
	)
}

func TestEncodeComposite(t *testing.T) {
	simpleComp := encodeTest{
		"SimpleComposite",
		cadence.CompositeType{
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.StringType{},
				},
				{
					Identifier: "b",
					Type:       cadence.StringType{},
				},
			},
		},
		cadence.NewComposite([]cadence.Value{
			cadence.NewString("foo"),
			cadence.NewString("bar"),
		}),
	}

	multiTypeComp := encodeTest{
		"MultiTypeComposite",
		cadence.CompositeType{
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.StringType{},
				},
				{
					Identifier: "b",
					Type:       cadence.IntType{},
				},
				{
					Identifier: "c",
					Type:       cadence.BoolType{},
				},
			},
		},
		cadence.NewComposite([]cadence.Value{
			cadence.NewString("foo"),
			cadence.NewInt(42),
			cadence.NewBool(true),
		}),
	}

	arrayComp := encodeTest{
		"ArrayComposite",
		cadence.CompositeType{
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: cadence.VariableSizedArrayType{
						ElementType: cadence.IntType{},
					},
				},
			},
		},
		cadence.NewComposite([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(1),
				cadence.NewInt(2),
				cadence.NewInt(3),
				cadence.NewInt(4),
				cadence.NewInt(5),
			}),
		}),
	}

	nestedComp := encodeTest{
		"NestedComposite",
		cadence.CompositeType{
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.StringType{},
				},
				{
					Identifier: "b",
					Type: cadence.CompositeType{
						Fields: []cadence.Field{
							{
								Identifier: "c",
								Type:       cadence.IntType{},
							},
						},
					},
				},
			},
		},
		cadence.NewComposite([]cadence.Value{
			cadence.NewString("foo"),
			cadence.NewComposite([]cadence.Value{
				cadence.NewInt(42),
			}),
		}),
	}

	testAllEncode(t,
		simpleComp,
		multiTypeComp,
		arrayComp,
		nestedComp,
	)
}

func TestEncodeEvent(t *testing.T) {
	simpleEvent := encodeTest{
		"SimpleEvent",
		cadence.EventType{
			CompositeType: cadence.CompositeType{
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.IntType{},
					},
					{
						Identifier: "b",
						Type:       cadence.StringType{},
					},
				},
			},
		},
		cadence.NewComposite(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.NewString("foo"),
			},
		),
	}

	compositeEvent := encodeTest{
		"CompositeEvent",
		cadence.EventType{
			CompositeType: cadence.CompositeType{
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.StringType{},
					},
					{
						Identifier: "b",
						Type: cadence.CompositeType{
							Fields: []cadence.Field{
								{
									Identifier: "c",
									Type:       cadence.StringType{},
								},
								{
									Identifier: "d",
									Type:       cadence.IntType{},
								},
							},
						},
					},
				},
			},
		},
		cadence.NewComposite(
			[]cadence.Value{
				cadence.NewString("foo"),
				cadence.NewComposite(
					[]cadence.Value{
						cadence.NewString("bar"),
						cadence.NewInt(42),
					},
				),
			},
		),
	}

	testAllEncode(t, simpleEvent, compositeEvent)
}

func testAllEncode(t *testing.T, tests ...encodeTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEncode(t, test.typ, test.val)
		})
	}
}

const numTrials = 250

func testEncode(t *testing.T, typ cadence.Type, val cadence.Value) {
	b1, err := xdr.Encode(val)
	require.NoError(t, err)

	t.Logf("Encoded value: %x", b1)

	// encoding should be deterministic, repeat to confirm
	for i := 0; i < numTrials; i++ {
		b2, err := xdr.Encode(val)
		require.NoError(t, err)
		assert.Equal(t, b1, b2)
	}

	decodedVal, err := xdr.Decode(typ, b1)
	require.NoError(t, err)

	assert.Equal(t, val, decodedVal)
}
