package json_test

import (
	"math"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence"
	"github.com/dapperlabs/cadence/encoding/json"
	"github.com/dapperlabs/cadence/runtime"
	"github.com/dapperlabs/cadence/runtime/sema"
)

type encodeTest struct {
	name     string
	val      cadence.Value
	expected string
}

func TestEncodeVoid(t *testing.T) {
	testEncode(t, cadence.NewVoid(), `{"type":"Void"}`)
}

func TestEncodeOptional(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Nil",
			cadence.NewOptional(nil),
			`{"type":"Optional","value":null}`,
		},
		{
			"Non-nil",
			cadence.NewOptional(cadence.NewInt(42)),
			`{"type":"Optional","value":{"type":"Int","value":"42"}}`,
		},
	}...)
}

func TestEncodeBool(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"True",
			cadence.NewBool(true),
			`{"type":"Bool","value":true}`,
		},
		{
			"False",
			cadence.NewBool(false),
			`{"type":"Bool","value":false}`,
		},
	}...)
}

func TestEncodeString(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Empty",
			cadence.NewString(""),
			`{"type":"String","value":""}`,
		},
		{
			"Non-empty",
			cadence.NewString("foo"),
			`{"type":"String","value":"foo"}`,
		},
	}...)
}

func TestEncodeAddress(t *testing.T) {
	testEncode(
		t,
		cadence.NewAddressFromBytes([]byte{1, 2, 3, 4, 5}),
		`{"type":"Address","value":"0x0102030405000000000000000000000000000000"}`,
	)
}

func TestEncodeInt(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Negative",
			cadence.NewInt(-42),
			`{"type":"Int","value":"-42"}`,
		},
		{
			"Zero",
			cadence.NewInt(0),
			`{"type":"Int","value":"0"}`,
		},
		{
			"Positive",
			cadence.NewInt(42),
			`{"type":"Int","value":"42"}`,
		},
		{
			"SmallerThanMinInt256",
			cadence.NewIntFromBig(big.NewInt(0).Sub(sema.Int256TypeMinInt, big.NewInt(10))),
			`{"type":"Int","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819978"}`,
		},
		{
			"LargerThanMaxUInt256",
			cadence.NewIntFromBig(big.NewInt(0).Add(sema.UInt256TypeMaxInt, big.NewInt(10))),
			`{"type":"Int","value":"115792089237316195423570985008687907853269984665640564039457584007913129639945"}`,
		},
	}...)
}

func TestEncodeInt8(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt8(math.MinInt8),
			`{"type":"Int8","value":"-128"}`,
		},
		{
			"Zero",
			cadence.NewInt8(0),
			`{"type":"Int8","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt8(math.MaxInt8),
			`{"type":"Int8","value":"127"}`,
		},
	}...)
}

func TestEncodeInt16(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt16(math.MinInt16),
			`{"type":"Int16","value":"-32768"}`,
		},
		{
			"Zero",
			cadence.NewInt16(0),
			`{"type":"Int16","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt16(math.MaxInt16),
			`{"type":"Int16","value":"32767"}`,
		},
	}...)
}

func TestEncodeInt32(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt32(math.MinInt32),
			`{"type":"Int32","value":"-2147483648"}`,
		},
		{
			"Zero",
			cadence.NewInt32(0),
			`{"type":"Int32","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt32(math.MaxInt32),
			`{"type":"Int32","value":"2147483647"}`,
		},
	}...)
}

func TestEncodeInt64(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt64(math.MinInt64),
			`{"type":"Int64","value":"-9223372036854775808"}`,
		},
		{
			"Zero",
			cadence.NewInt64(0),
			`{"type":"Int64","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt64(math.MaxInt64),
			`{"type":"Int64","value":"9223372036854775807"}`,
		},
	}...)
}

func TestEncodeInt128(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt128FromBig(sema.Int128TypeMinInt),
			`{"type":"Int128","value":"-170141183460469231731687303715884105728"}`,
		},
		{
			"Zero",
			cadence.NewInt128(0),
			`{"type":"Int128","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt128FromBig(sema.Int128TypeMaxInt),
			`{"type":"Int128","value":"170141183460469231731687303715884105727"}`,
		},
	}...)
}

func TestEncodeInt256(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt256FromBig(sema.Int256TypeMinInt),
			`{"type":"Int256","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819968"}`,
		},
		{
			"Zero",
			cadence.NewInt256(0),
			`{"type":"Int256","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt256FromBig(sema.Int256TypeMaxInt),
			`{"type":"Int256","value":"57896044618658097711785492504343953926634992332820282019728792003956564819967"}`,
		},
	}...)
}

func TestEncodeUInt(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt(0),
			`{"type":"UInt","value":"0"}`,
		},
		{
			"Positive",
			cadence.NewUInt(42),
			`{"type":"UInt","value":"42"}`,
		},
		{
			"LargerThanMaxUInt256",
			cadence.NewUIntFromBig(big.NewInt(0).Add(sema.UInt256TypeMaxInt, big.NewInt(10))),
			`{"type":"UInt","value":"115792089237316195423570985008687907853269984665640564039457584007913129639945"}`,
		},
	}...)
}

func TestEncodeUInt8(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt8(0),
			`{"type":"UInt8","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt8(math.MaxUint8),
			`{"type":"UInt8","value":"255"}`,
		},
	}...)
}

func TestEncodeUInt16(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt16(0),
			`{"type":"UInt16","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt16(math.MaxUint16),
			`{"type":"UInt16","value":"65535"}`,
		},
	}...)
}

func TestEncodeUInt32(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt32(0),
			`{"type":"UInt32","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt32(math.MaxUint32),
			`{"type":"UInt32","value":"4294967295"}`,
		},
	}...)
}

func TestEncodeUInt64(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt64(0),
			`{"type":"UInt64","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt64(uint64(math.MaxUint64)),
			`{"type":"UInt64","value":"18446744073709551615"}`,
		},
	}...)
}

func TestEncodeUInt128(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt128(0),
			`{"type":"UInt128","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt128FromBig(sema.UInt128TypeMaxInt),
			`{"type":"UInt128","value":"340282366920938463463374607431768211455"}`,
		},
	}...)
}

func TestEncodeUInt256(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt256(0),
			`{"type":"UInt256","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt256FromBig(sema.UInt256TypeMaxInt),
			`{"type":"UInt256","value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`,
		},
	}...)
}

func TestEncodeWord8(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord8(0),
			`{"type":"Word8","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord8(math.MaxUint8),
			`{"type":"Word8","value":"255"}`,
		},
	}...)
}

func TestEncodeWord16(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord16(0),
			`{"type":"Word16","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord16(math.MaxUint16),
			`{"type":"Word16","value":"65535"}`,
		},
	}...)
}

func TestEncodeWord32(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord32(0),
			`{"type":"Word32","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord32(math.MaxUint32),
			`{"type":"Word32","value":"4294967295"}`,
		},
	}...)
}

func TestEncodeFix64(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewFix64(0),
			`{"type":"Fix64","value":"0.00000000"}`,
		},
		{
			"789.00123010",
			cadence.NewFix64(78_900_123_010),
			`{"type":"Fix64","value":"789.00123010"}`,
		},
		{
			"1234.056",
			cadence.NewFix64(123_405_600_000),
			`{"type":"Fix64","value":"1234.05600000"}`,
		},
		{
			"-12345.006789",
			cadence.NewFix64(-1_234_500_678_900),
			`{"type":"Fix64","value":"-12345.00678900"}`,
		},
	}...)
}

func TestEncodeUFix64(t *testing.T) {
	testAllEncode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUFix64(0),
			`{"type":"UFix64","value":"0.00000000"}`,
		},
		{
			"789.00123010",
			cadence.NewUFix64(78_900_123_010),
			`{"type":"UFix64","value":"789.00123010"}`,
		},
		{
			"1234.056",
			cadence.NewUFix64(123_405_600_000),
			`{"type":"UFix64","value":"1234.05600000"}`,
		},
	}...)
}

var resourceType = cadence.ResourceType{
	CompositeType: cadence.CompositeType{
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
}

func TestEncodeArray(t *testing.T) {
	emptyArray := encodeTest{
		"Empty",
		cadence.NewArray([]cadence.Value{}),
		`{"type":"Array","value":[]}`,
	}

	intArray := encodeTest{
		"Integers",
		cadence.NewArray([]cadence.Value{
			cadence.NewInt(1),
			cadence.NewInt(2),
			cadence.NewInt(3),
		}),
		`{"type":"Array","value":[{"type":"Int","value":"1"},{"type":"Int","value":"2"},{"type":"Int","value":"3"}]}`,
	}

	resourceArray := encodeTest{
		"Resources",
		cadence.NewArray([]cadence.Value{
			cadence.NewComposite([]cadence.Value{
				cadence.NewString("a"),
				cadence.NewInt(1),
			}).WithType(resourceType),
			cadence.NewComposite([]cadence.Value{
				cadence.NewString("b"),
				cadence.NewInt(1),
			}).WithType(resourceType),
			cadence.NewComposite([]cadence.Value{
				cadence.NewString("c"),
				cadence.NewInt(1),
			}).WithType(resourceType),
		}),
		`{"type":"Array","value":[{"type":"Resource","value":{"id":"","fields":[{"name":"a","value":{"type":"String","value":"a"}},{"name":"b","value":{"type":"Int","value":"1"}}]}},{"type":"Resource","value":{"id":"","fields":[{"name":"a","value":{"type":"String","value":"b"}},{"name":"b","value":{"type":"Int","value":"1"}}]}},{"type":"Resource","value":{"id":"","fields":[{"name":"a","value":{"type":"String","value":"c"}},{"name":"b","value":{"type":"Int","value":"1"}}]}}]}`,
	}

	testAllEncode(t,
		emptyArray,
		intArray,
		resourceArray,
	)
}

func TestEncodeDictionary(t *testing.T) {
	simpleDict := encodeTest{
		"Simple",
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
		`{"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Int","value":"1"}},{"key":{"type":"String","value":"b"},"value":{"type":"Int","value":"2"}},{"key":{"type":"String","value":"c"},"value":{"type":"Int","value":"3"}}]}`,
	}

	nestedDict := encodeTest{
		"Nested",
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
		`{"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Dictionary","value":[{"key":{"type":"String","value":"1"},"value":{"type":"Int","value":"1"}}]}},{"key":{"type":"String","value":"b"},"value":{"type":"Dictionary","value":[{"key":{"type":"String","value":"2"},"value":{"type":"Int","value":"2"}}]}},{"key":{"type":"String","value":"c"},"value":{"type":"Dictionary","value":[{"key":{"type":"String","value":"3"},"value":{"type":"Int","value":"3"}}]}}]}`,
	}

	resourceDict := encodeTest{
		"Resources",
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.NewString("a"),
				Value: cadence.NewComposite([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(fooResourceType),
			},
			{
				Key: cadence.NewString("b"),
				Value: cadence.NewComposite([]cadence.Value{
					cadence.NewInt(2),
				}).WithType(fooResourceType),
			},
			{
				Key: cadence.NewString("c"),
				Value: cadence.NewComposite([]cadence.Value{
					cadence.NewInt(3),
				}).WithType(fooResourceType),
			},
		}),
		`{"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Resource","value":{"id":"test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"1"}}]}}},{"key":{"type":"String","value":"b"},"value":{"type":"Resource","value":{"id":"test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"2"}}]}}},{"key":{"type":"String","value":"c"},"value":{"type":"Resource","value":{"id":"test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"3"}}]}}}]}`,
	}

	testAllEncode(t,
		simpleDict,
		nestedDict,
		resourceDict,
	)
}

func TestEncodeResource(t *testing.T) {
	script := `
        access(all) resource Foo {
            access(all) let bar: Int
        
            init(bar: Int) {
                self.bar = bar
            }
        }
    
        access(all) fun main(): @Foo {
            return <- create Foo(bar: 42)
        }
    `

	expectedJSON := `{"type":"Resource","value":{"id":"test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"42"}}]}}`

	v := convertValueFromScript(t, script)

	testEncode(t, v, expectedJSON)
}

func TestEncodeNestedResource(t *testing.T) {
	script := `
        access(all) resource Bar {
            access(all) let x: Int

            init(x: Int) {
                self.x = x
            }
        }

        access(all) resource Foo {
            access(all) let bar: @Bar
        
            init(bar: @Bar) {
                self.bar <- bar
            }

            destroy() {
                destroy self.bar
            }
        }
    
        access(all) fun main(): @Foo {
            return <- create Foo(bar: <- create Bar(x: 42))
        }
    `

	expectedJSON := `{"type":"Resource","value":{"id":"test.Foo","fields":[{"name":"bar","value":{"type":"Resource","value":{"id":"test.Bar","fields":[{"name":"x","value":{"type":"Int","value":"42"}}]}}}]}}`

	v := convertValueFromScript(t, script)

	testEncode(t, v, expectedJSON)
}

func TestEncodeEvent(t *testing.T) {
	// TODO: test event encoding
}

func trimJSON(b []byte) string {
	return strings.TrimSuffix(string(b), "\n")
}

func convertValueFromScript(t *testing.T, script string) cadence.Value {
	rt := runtime.NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		[]byte(script),
		nil,
		runtime.StringLocation("test"),
	)

	require.NoError(t, err)

	return cadence.ConvertValue(value)
}

func testAllEncode(t *testing.T, tests ...encodeTest) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testEncode(t, test.val, test.expected)
		})
	}
}

func testEncode(t *testing.T, val cadence.Value, expectedJSON string) {
	actualJSON, err := json.Encode(val)
	require.NoError(t, err)

	assert.Equal(t, expectedJSON, trimJSON(actualJSON))
}

var fooResourceType = cadence.ResourceType{
	CompositeType: cadence.CompositeType{
		Identifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "bar",
				Type:       cadence.IntType{},
			},
		},
	}.WithID("test.Foo"),
}
