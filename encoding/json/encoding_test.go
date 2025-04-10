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

package json

import (
	"fmt"
	"math"
	"math/big"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

type encodeTest struct {
	name     string
	val      cadence.Value
	expected string
}

func TestEncodeVoid(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(t,
		cadence.NewVoid(),
		// language=json
		`{"type": "Void"}`,
	)
}

func TestEncodeOptional(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Nil",
			cadence.NewOptional(nil),
			// language=json
			`
              {
                "type": "Optional",
                "value": null
              }
            `,
		},
		{
			"Non-nil",
			cadence.NewOptional(cadence.NewInt(42)),
			// language=json
			`
              {
                "type": "Optional",
                "value": {
                  "type": "Int",
                  "value": "42"
                }
              }
            `,
		},
	}...)
}

func TestEncodeBool(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"True",
			cadence.NewBool(true),
			// language=json
			`{"type":"Bool","value":true}`,
		},
		{
			"False",
			cadence.NewBool(false),
			// language=json
			`{"type":"Bool","value":false}`,
		},
	}...)
}

func TestBadCharacters(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {

		t.Parallel()
		_, err := cadence.NewCharacter("")
		require.Error(t, err)
	})

	t.Run("long", func(t *testing.T) {

		t.Parallel()
		_, err := cadence.NewCharacter("ab")
		require.Error(t, err)
	})

	t.Run("ok simple", func(t *testing.T) {

		t.Parallel()
		_, err := cadence.NewCharacter(`\a`)
		require.Error(t, err)
	})

	t.Run("ok complex", func(t *testing.T) {

		t.Parallel()
		_, err := cadence.NewCharacter(`\u{75}\u{308}`)
		require.Error(t, err)
	})
}

func TestEncodeCharacter(t *testing.T) {

	t.Parallel()

	a, _ := cadence.NewCharacter("a")
	b, _ := cadence.NewCharacter("b")

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"a",
			a,
			// language=json
			`{"type":"Character","value":"a"}`,
		},
		{
			"b",
			b,
			// language=json
			`{"type":"Character","value":"b"}`,
		},
	}...)
}

func TestEncodeString(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Empty",
			cadence.String(""),
			// language=json
			`{"type":"String","value":""}`,
		},
		{
			"Non-empty",
			cadence.String("foo"),
			// language=json
			`{"type":"String","value":"foo"}`,
		},
	}...)
}

func TestEncodeAddress(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(
		t,
		cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
		// language=json
		`{"type":"Address","value":"0x0000000102030405"}`,
	)
}

func TestDecodeInvalidAddress(t *testing.T) {

	t.Parallel()

	t.Run("valid UTF-8 prefix", func(t *testing.T) {
		t.Parallel()

		msg := `{"type":"Address","value":"000000000102030405"}`

		_, err := Decode(nil, []byte(msg))
		require.ErrorContains(t, err, "invalid address prefix: expected 0x, got 00")
	})

	t.Run("invalid UTF-8 prefix", func(t *testing.T) {
		t.Parallel()

		msg := `{"type":"Address","value":"\u1234"}`

		_, err := Decode(nil, []byte(msg))
		require.ErrorContains(t, err, "invalid address prefix: (shown as hex) expected 3078, got e188")
	})
}

func TestEncodeInt(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Negative",
			cadence.NewInt(-42),
			// language=json
			`{"type":"Int","value":"-42"}`,
		},
		{
			"Zero",
			cadence.NewInt(0),
			// language=json
			`{"type":"Int","value":"0"}`,
		},
		{
			"Positive",
			cadence.NewInt(42),
			// language=json
			`{"type":"Int","value":"42"}`,
		},
		{
			"SmallerThanMinInt256",
			cadence.NewIntFromBig(new(big.Int).Sub(sema.Int256TypeMinIntBig, big.NewInt(10))),
			// language=json
			`{"type":"Int","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819978"}`,
		},
		{
			"LargerThanMaxUInt256",
			cadence.NewIntFromBig(new(big.Int).Add(sema.UInt256TypeMaxIntBig, big.NewInt(10))),
			// language=json
			`{"type":"Int","value":"115792089237316195423570985008687907853269984665640564039457584007913129639945"}`,
		},
	}...)
}

func TestEncodeInt8(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt8(math.MinInt8),
			// language=json
			`{"type":"Int8","value":"-128"}`,
		},
		{
			"Zero",
			cadence.NewInt8(0),
			// language=json
			`{"type":"Int8","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt8(math.MaxInt8),
			// language=json
			`{"type":"Int8","value":"127"}`,
		},
	}...)
}

func TestEncodeInt16(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt16(math.MinInt16),
			// language=json
			`{"type":"Int16","value":"-32768"}`,
		},
		{
			"Zero",
			cadence.NewInt16(0),
			// language=json
			`{"type":"Int16","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt16(math.MaxInt16),
			// language=json
			`{"type":"Int16","value":"32767"}`,
		},
	}...)
}

func TestEncodeInt32(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt32(math.MinInt32),
			// language=json
			`{"type":"Int32","value":"-2147483648"}`,
		},
		{
			"Zero",
			cadence.NewInt32(0),
			// language=json
			`{"type":"Int32","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt32(math.MaxInt32),
			// language=json
			`{"type":"Int32","value":"2147483647"}`,
		},
	}...)
}

func TestEncodeInt64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt64(math.MinInt64),
			// language=json
			`{"type":"Int64","value":"-9223372036854775808"}`,
		},
		{
			"Zero",
			cadence.NewInt64(0),
			// language=json
			`{"type":"Int64","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt64(math.MaxInt64),
			// language=json
			`{"type":"Int64","value":"9223372036854775807"}`,
		},
	}...)
}

func TestEncodeInt128(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.Int128{Value: sema.Int128TypeMinIntBig},
			// language=json
			`{"type":"Int128","value":"-170141183460469231731687303715884105728"}`,
		},
		{
			"Zero",
			cadence.NewInt128(0),
			// language=json
			`{"type":"Int128","value":"0"}`,
		},
		{
			"Max",
			cadence.Int128{Value: sema.Int128TypeMaxIntBig},
			// language=json
			`{"type":"Int128","value":"170141183460469231731687303715884105727"}`,
		},
	}...)
}

func TestEncodeInt256(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.Int256{Value: sema.Int256TypeMinIntBig},
			// language=json
			`{"type":"Int256","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819968"}`,
		},
		{
			"Zero",
			cadence.NewInt256(0),
			// language=json
			`{"type":"Int256","value":"0"}`,
		},
		{
			"Max",
			cadence.Int256{Value: sema.Int256TypeMaxIntBig},
			// language=json
			`{"type":"Int256","value":"57896044618658097711785492504343953926634992332820282019728792003956564819967"}`,
		},
	}...)
}

func TestEncodeUInt(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt(0),
			// language=json
			`{"type":"UInt","value":"0"}`,
		},
		{
			"Positive",
			cadence.NewUInt(42),
			// language=json
			`{"type":"UInt","value":"42"}`,
		},
		{
			"LargerThanMaxUInt256",
			cadence.UInt{Value: new(big.Int).Add(sema.UInt256TypeMaxIntBig, big.NewInt(10))},
			// language=json
			`{"type":"UInt","value":"115792089237316195423570985008687907853269984665640564039457584007913129639945"}`,
		},
	}...)
}

func TestEncodeUInt8(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt8(0),
			// language=json
			`{"type":"UInt8","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt8(math.MaxUint8),
			// language=json
			`{"type":"UInt8","value":"255"}`,
		},
	}...)
}

func TestEncodeUInt16(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt16(0),
			// language=json
			`{"type":"UInt16","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt16(math.MaxUint16),
			// language=json
			`{"type":"UInt16","value":"65535"}`,
		},
	}...)
}

func TestEncodeUInt32(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt32(0),
			// language=json
			`{"type":"UInt32","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt32(math.MaxUint32),
			// language=json
			`{"type":"UInt32","value":"4294967295"}`,
		},
	}...)
}

func TestEncodeUInt64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt64(0),
			// language=json
			`{"type":"UInt64","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt64(uint64(math.MaxUint64)),
			// language=json
			`{"type":"UInt64","value":"18446744073709551615"}`,
		},
	}...)
}

func TestEncodeUInt128(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt128(0),
			// language=json
			`{"type":"UInt128","value":"0"}`,
		},
		{
			"Max",
			cadence.UInt128{Value: sema.UInt128TypeMaxIntBig},
			// language=json
			`{"type":"UInt128","value":"340282366920938463463374607431768211455"}`,
		},
	}...)
}

func TestEncodeUInt256(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt256(0),
			// language=json
			`{"type":"UInt256","value":"0"}`,
		},
		{
			"Max",
			cadence.UInt256{Value: sema.UInt256TypeMaxIntBig},
			// language=json
			`{"type":"UInt256","value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`,
		},
	}...)
}

func TestEncodeWord8(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord8(0),
			// language=json
			`{"type":"Word8","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord8(math.MaxUint8),
			// language=json
			`{"type":"Word8","value":"255"}`,
		},
	}...)
}

func TestEncodeWord16(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord16(0),
			// language=json
			`{"type":"Word16","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord16(math.MaxUint16),
			// language=json
			`{"type":"Word16","value":"65535"}`,
		},
	}...)
}

func TestEncodeWord32(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord32(0),
			// language=json
			`{"type":"Word32","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord32(math.MaxUint32),
			// language=json
			`{"type":"Word32","value":"4294967295"}`,
		},
	}...)
}

func TestEncodeWord64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord64(0),
			// language=json
			`{"type":"Word64","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord64(math.MaxUint64),
			// language=json
			`{"type":"Word64","value":"18446744073709551615"}`,
		},
	}...)
}

func TestEncodeWord128(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord128(0),
			// language=json
			`{"type":"Word128","value":"0"}`,
		},
		{
			"Max",
			cadence.Word128{Value: sema.Word128TypeMaxIntBig},
			// language=json
			`{"type":"Word128","value":"340282366920938463463374607431768211455"}`,
		},
	}...)
}

func TestEncodeWord256(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord256(0),
			// language=json
			`{"type":"Word256","value":"0"}`,
		},
		{
			"Max",
			cadence.Word256{Value: sema.Word256TypeMaxIntBig},
			// language=json
			`{"type":"Word256","value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}`,
		},
	}...)
}

func TestEncodeFix64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.Fix64(0),
			// language=json
			`{"type":"Fix64","value":"0.00000000"}`,
		},
		{
			"789.00123010",
			cadence.Fix64(78_900_123_010),
			// language=json
			`{"type":"Fix64","value":"789.00123010"}`,
		},
		{
			"1234.056",
			cadence.Fix64(123_405_600_000),
			// language=json
			`{"type":"Fix64","value":"1234.05600000"}`,
		},
		{
			"-12345.006789",
			cadence.Fix64(-1_234_500_678_900),
			// language=json
			`{"type":"Fix64","value":"-12345.00678900"}`,
		},
	}...)
}

func TestEncodeUFix64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.UFix64(0),
			// language=json
			`{"type":"UFix64","value":"0.00000000"}`,
		},
		{
			"789.00123010",
			cadence.UFix64(78_900_123_010),
			// language=json
			`{"type":"UFix64","value":"789.00123010"}`,
		},
		{
			"1234.056",
			cadence.UFix64(123_405_600_000),
			// language=json
			`{"type":"UFix64","value":"1234.05600000"}`,
		},
	}...)
}

func TestEncodeArray(t *testing.T) {

	t.Parallel()

	emptyArray := encodeTest{
		"Empty",
		cadence.NewArray([]cadence.Value{}),
		// language=json
		`{"type":"Array","value":[]}`,
	}

	intArray := encodeTest{
		"Integers",
		cadence.NewArray([]cadence.Value{
			cadence.NewInt(1),
			cadence.NewInt(2),
			cadence.NewInt(3),
		}),
		// language=json
		`
          {
            "type": "Array",
            "value": [
              {
                "type": "Int",
                "value": "1"
              },
              {
                "type": "Int",
                "value": "2"
              },
              {
                "type": "Int",
                "value": "3"
              }
            ]
          }
        `,
	}

	fooResourceType := newFooResourceType()

	resourceArray := encodeTest{
		"Resources",
		cadence.NewArray([]cadence.Value{
			cadence.NewResource([]cadence.Value{
				cadence.NewInt(1),
			}).WithType(fooResourceType),
			cadence.NewResource([]cadence.Value{
				cadence.NewInt(2),
			}).WithType(fooResourceType),
			cadence.NewResource([]cadence.Value{
				cadence.NewInt(3),
			}).WithType(fooResourceType),
		}),
		// language=json
		`
          {
            "type": "Array",
            "value": [
              {
                "type": "Resource",
                "value": {
                  "id": "S.test.Foo",
                  "fields": [
                    {
                      "name": "bar",
                      "value": {
                        "type": "Int",
                        "value": "1"
                      }
                    }
                  ]
                }
              },
              {
                "type": "Resource",
                "value": {
                  "id": "S.test.Foo",
                  "fields": [
                    {
                      "name": "bar",
                      "value": {
                        "type": "Int",
                        "value": "2"
                      }
                    }
                  ]
                }
              },
              {
                "type": "Resource",
                "value": {
                  "id": "S.test.Foo",
                  "fields": [
                    {
                      "name": "bar",
                      "value": {
                        "type": "Int",
                        "value": "3"
                      }
                    }
                  ]
                }
              }
            ]
          }
        `,
	}

	testAllEncodeAndDecode(t,
		emptyArray,
		intArray,
		resourceArray,
	)
}

func TestEncodeDictionary(t *testing.T) {

	t.Parallel()

	simpleDict := encodeTest{
		"Simple",
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("a"),
				Value: cadence.NewInt(1),
			},
			{
				Key:   cadence.String("b"),
				Value: cadence.NewInt(2),
			},
			{
				Key:   cadence.String("c"),
				Value: cadence.NewInt(3),
			},
		}),
		// language=json
		`
          {
            "type": "Dictionary",
            "value": [
              {
                "key": {
                  "type": "String",
                  "value": "a"
                },
                "value": {
                  "type": "Int",
                  "value": "1"
                }
              },
              {
                "key": {
                  "type": "String",
                  "value": "b"
                },
                "value": {
                  "type": "Int",
                  "value": "2"
                }
              },
              {
                "key": {
                  "type": "String",
                  "value": "c"
                },
                "value": {
                  "type": "Int",
                  "value": "3"
                }
              }
            ]
          }
        `,
	}

	nestedDict := encodeTest{
		"Nested",
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.String("a"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("1"),
						Value: cadence.NewInt(1),
					},
				}),
			},
			{
				Key: cadence.String("b"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("2"),
						Value: cadence.NewInt(2),
					},
				}),
			},
			{
				Key: cadence.String("c"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("3"),
						Value: cadence.NewInt(3),
					},
				}),
			},
		}),
		// language=json
		`
          {
            "type": "Dictionary",
            "value": [
              {
                "key": {
                  "type": "String",
                  "value": "a"
                },
                "value": {
                  "type": "Dictionary",
                  "value": [
                    {
                      "key": {
                        "type": "String",
                        "value": "1"
                      },
                      "value": {
                        "type": "Int",
                        "value": "1"
                      }
                    }
                  ]
                }
              },
              {
                "key": {
                  "type": "String",
                  "value": "b"
                },
                "value": {
                  "type": "Dictionary",
                  "value": [
                    {
                      "key": {
                        "type": "String",
                        "value": "2"
                      },
                      "value": {
                        "type": "Int",
                        "value": "2"
                      }
                    }
                  ]
                }
              },
              {
                "key": {
                  "type": "String",
                  "value": "c"
                },
                "value": {
                  "type": "Dictionary",
                  "value": [
                    {
                      "key": {
                        "type": "String",
                        "value": "3"
                      },
                      "value": {
                        "type": "Int",
                        "value": "3"
                      }
                    }
                  ]
                }
              }
            ]
          }
        `,
	}

	fooResourceType := newFooResourceType()

	resourceDict := encodeTest{
		"Resources",
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.String("a"),
				Value: cadence.NewResource([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(fooResourceType),
			},
			{
				Key: cadence.String("b"),
				Value: cadence.NewResource([]cadence.Value{
					cadence.NewInt(2),
				}).WithType(fooResourceType),
			},
			{
				Key: cadence.String("c"),
				Value: cadence.NewResource([]cadence.Value{
					cadence.NewInt(3),
				}).WithType(fooResourceType),
			},
		}),
		// language=json
		`
          {
            "type": "Dictionary",
            "value": [
              {
                "key": {
                  "type": "String",
                  "value": "a"
                },
                "value": {
                  "type": "Resource",
                  "value": {
                    "id": "S.test.Foo",
                    "fields": [
                      {
                        "name": "bar",
                        "value": {
                          "type": "Int",
                          "value": "1"
                        }
                      }
                    ]
                  }
                }
              },
              {
                "key": {
                  "type": "String",
                  "value": "b"
                },
                "value": {
                  "type": "Resource",
                  "value": {
                    "id": "S.test.Foo",
                    "fields": [
                      {
                        "name": "bar",
                        "value": {
                          "type": "Int",
                          "value": "2"
                        }
                      }
                    ]
                  }
                }
              },
              {
                "key": {
                  "type": "String",
                  "value": "c"
                },
                "value": {
                  "type": "Resource",
                  "value": {
                    "id": "S.test.Foo",
                    "fields": [
                      {
                        "name": "bar",
                        "value": {
                          "type": "Int",
                          "value": "3"
                        }
                      }
                    ]
                  }
                }
              }
            ]
          }
        `,
	}

	testAllEncodeAndDecode(t,
		simpleDict,
		nestedDict,
		resourceDict,
	)
}

func exportFromScript(t *testing.T, code string) cadence.Value {
	checker, err := ParseAndCheck(t, code)
	require.NoError(t, err)

	var uuid uint64 = 0

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			UUIDHandler: func() (uint64, error) {
				uuid++
				return uuid, nil
			},
			AtreeStorageValidationEnabled: true,
			AtreeValueValidationEnabled:   true,
			Storage:                       interpreter.NewInMemoryStorage(nil),
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	result, err := inter.Invoke("main")
	require.NoError(t, err)

	exported, err := runtime.ExportValue(result, inter, interpreter.EmptyLocationRange)
	require.NoError(t, err)

	return exported
}

func TestEncodeResource(t *testing.T) {

	t.Parallel()

	t.Run("Simple", func(t *testing.T) {

		t.Parallel()

		actual := exportFromScript(t, `
            resource Foo {
                let bar: Int

                init(bar: Int) {
                    self.bar = bar
                }
            }

            fun main(): @Foo {
                return <- create Foo(bar: 42)
            }
        `)

		// language=json
		expectedJSON := `
          {
            "type": "Resource",
            "value": {
              "id": "S.test.Foo",
              "fields": [
                {
                  "name": "uuid",
                  "value": {
                    "type": "UInt64",
                    "value": "1"
                  }
                },
                {
                  "name": "bar",
                  "value": {
                    "type": "Int",
                    "value": "42"
                  }
                }
              ]
            }
          }
        `

		testEncodeAndDecode(t, actual, expectedJSON)
	})

	t.Run("With function member", func(t *testing.T) {

		t.Parallel()

		actual := exportFromScript(t, `
            resource Foo {
                let bar: Int

                fun foo(): String {
                    return "foo"
                }

                init(bar: Int) {
                    self.bar = bar
                }
            }

            fun main(): @Foo {
                return <- create Foo(bar: 42)
            }
        `)

		// function "foo" should be omitted from resulting JSON
		// language=json
		expectedJSON := `
          {
            "type": "Resource",
            "value": {
              "id": "S.test.Foo",
              "fields": [
                {
                  "name": "uuid",
                  "value": {
                    "type": "UInt64",
                    "value": "1"
                  }
                },
                {
                  "name": "bar",
                  "value": {
                    "type": "Int",
                    "value": "42"
                  }
                }
              ]
            }
          }
        `

		testEncodeAndDecode(t, actual, expectedJSON)
	})

	t.Run("Nested resource", func(t *testing.T) {

		t.Parallel()

		actual := exportFromScript(t, `
            resource Bar {
                let x: Int

                init(x: Int) {
                    self.x = x
                }
            }

            resource Foo {
                let bar: @Bar

                init(bar: @Bar) {
                    self.bar <- bar
                }
            }

            fun main(): @Foo {
                return <- create Foo(bar: <- create Bar(x: 42))
            }
        `)

		// language=json
		expectedJSON := `
          {
            "type": "Resource",
            "value": {
              "id": "S.test.Foo",
              "fields": [
                {
                  "name": "uuid",
                  "value": {
                    "type": "UInt64",
                    "value": "2"
                  }
                },
                {
                  "name": "bar",
                  "value": {
                    "type": "Resource",
                    "value": {
                      "id": "S.test.Bar",
                      "fields": [
                        {
                          "name": "uuid",
                          "value": {
                            "type": "UInt64",
                            "value": "1"
                          }
                        },
                        {
                          "name": "x",
                          "value": {
                            "type": "Int",
                            "value": "42"
                          }
                        }
                      ]
                    }
                  }
                }
              ]
            }
          }
        `

		testEncodeAndDecode(t, actual, expectedJSON)
	})
}

func TestEncodeStruct(t *testing.T) {

	t.Parallel()

	simpleStructType := cadence.NewStructType(
		TestLocation,
		"FooStruct",
		[]cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType,
			},
			{
				Identifier: "b",
				Type:       cadence.StringType,
			},
		},
		nil,
	)

	simpleStruct := encodeTest{
		"Simple",
		cadence.NewStruct(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.String("foo"),
			},
		).WithType(simpleStructType),
		// language=json
		`
          {
            "type": "Struct",
            "value": {
              "id": "S.test.FooStruct",
              "fields": [
                {
                  "name": "a",
                  "value": {
                    "type": "Int",
                    "value": "1"
                  }
                },
                {
                  "name": "b",
                  "value": {
                    "type": "String",
                    "value": "foo"
                  }
                }
              ]
            }
          }
        `,
	}

	fooResourceType := newFooResourceType()

	resourceStructType := cadence.NewStructType(
		TestLocation,
		"FooStruct",
		[]cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.StringType,
			},
			{
				Identifier: "b",
				Type:       fooResourceType,
			},
		},
		nil,
	)

	resourceStruct := encodeTest{
		"Resources",
		cadence.NewStruct(
			[]cadence.Value{
				cadence.String("foo"),
				cadence.NewResource(
					[]cadence.Value{
						cadence.NewInt(42),
					},
				).WithType(fooResourceType),
			},
		).WithType(resourceStructType),
		// language=json
		`
          {
            "type": "Struct",
            "value": {
              "id": "S.test.FooStruct",
              "fields": [
                {
                  "name": "a",
                  "value": {
                    "type": "String",
                    "value": "foo"
                  }
                },
                {
                  "name": "b",
                  "value": {
                    "type": "Resource",
                    "value": {
                      "id": "S.test.Foo",
                      "fields": [
                        {
                          "name": "bar",
                          "value": {
                            "type": "Int",
                            "value": "42"
                          }
                        }
                      ]
                    }
                  }
                }
              ]
            }
          }
        `,
	}

	testAllEncodeAndDecode(t, simpleStruct, resourceStruct)
}

func TestEncodeInclusiveRange(t *testing.T) {

	t.Parallel()

	simpleInclusiveRange := encodeTest{
		"Simple",
		cadence.NewInclusiveRange(
			cadence.NewInt256(10),
			cadence.NewInt256(20),
			cadence.NewInt256(5),
		).WithType(cadence.NewInclusiveRangeType(cadence.Int256Type)),
		// language=json
		`
			{
				"type": "InclusiveRange",
				"value": {
					"start": {
						"type": "Int256",
						"value": "10"
					},
					"end": {
						"type": "Int256",
						"value": "20"
					},
					"step": {
						"type": "Int256",
						"value": "5"
					}
				}
			}
		`,
	}

	testAllEncodeAndDecode(t, simpleInclusiveRange)
}

func TestEncodeEvent(t *testing.T) {

	t.Parallel()

	simpleEventType := cadence.NewEventType(
		TestLocation,
		"FooEvent",
		[]cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType,
			},
			{
				Identifier: "b",
				Type:       cadence.StringType,
			},
		},
		nil,
	)

	simpleEvent := encodeTest{
		"Simple",
		cadence.NewEvent(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.String("foo"),
			},
		).WithType(simpleEventType),
		// language=json
		`
          {
            "type": "Event",
            "value": {
              "id": "S.test.FooEvent",
              "fields": [
                {
                  "name": "a",
                  "value": {
                    "type": "Int",
                    "value": "1"
                  }
                },
                {
                  "name": "b",
                  "value": {
                    "type": "String",
                    "value": "foo"
                  }
                }
              ]
            }
          }
        `,
	}

	fooResourceType := newFooResourceType()

	resourceEventType := cadence.NewEventType(
		TestLocation,
		"FooEvent",
		[]cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.StringType,
			},
			{
				Identifier: "b",
				Type:       fooResourceType,
			},
		},
		nil,
	)

	resourceEvent := encodeTest{
		"Resources",
		cadence.NewEvent(
			[]cadence.Value{
				cadence.String("foo"),
				cadence.NewResource(
					[]cadence.Value{
						cadence.NewInt(42),
					},
				).WithType(fooResourceType),
			},
		).WithType(resourceEventType),
		// language=json
		`
          {
            "type": "Event",
            "value": {
              "id": "S.test.FooEvent",
              "fields": [
                {
                  "name": "a",
                  "value": {
                    "type": "String",
                    "value": "foo"
                  }
                },
                {
                  "name": "b",
                  "value": {
                    "type": "Resource",
                    "value": {
                      "id": "S.test.Foo",
                      "fields": [
                        {
                          "name": "bar",
                          "value": {
                            "type": "Int",
                            "value": "42"
                          }
                        }
                      ]
                    }
                  }
                }
              ]
            }
          }
        `,
	}

	testAllEncodeAndDecode(t, simpleEvent, resourceEvent)
}

func TestEncodeContract(t *testing.T) {

	t.Parallel()

	simpleContractType := cadence.NewContractType(
		TestLocation,
		"FooContract",
		[]cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType,
			},
			{
				Identifier: "b",
				Type:       cadence.StringType,
			},
		},
		nil,
	)

	simpleContract := encodeTest{
		"Simple",
		cadence.NewContract(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.String("foo"),
			},
		).WithType(simpleContractType),
		// language=json
		`
          {
            "type": "Contract",
            "value": {
              "id": "S.test.FooContract",
              "fields": [
                {
                  "name": "a",
                  "value": {
                    "type": "Int",
                    "value": "1"
                  }
                },
                {
                  "name": "b",
                  "value": {
                    "type": "String",
                    "value": "foo"
                  }
                }
              ]
            }
          }
        `,
	}

	fooResourceType := newFooResourceType()

	resourceContractType := cadence.NewContractType(
		TestLocation,
		"FooContract",
		[]cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.StringType,
			},
			{
				Identifier: "b",
				Type:       fooResourceType,
			},
		},
		nil,
	)

	resourceContract := encodeTest{
		"Resources",
		cadence.NewContract(
			[]cadence.Value{
				cadence.String("foo"),
				cadence.NewResource(
					[]cadence.Value{
						cadence.NewInt(42),
					},
				).WithType(fooResourceType),
			},
		).WithType(resourceContractType),
		// language=json
		`
          {
            "type": "Contract",
            "value": {
              "id": "S.test.FooContract",
              "fields": [
                {
                  "name": "a",
                  "value": {
                    "type": "String",
                    "value": "foo"
                  }
                },
                {
                  "name": "b",
                  "value": {
                    "type": "Resource",
                    "value": {
                      "id": "S.test.Foo",
                      "fields": [
                        {
                          "name": "bar",
                          "value": {
                            "type": "Int",
                            "value": "42"
                          }
                        }
                      ]
                    }
                  }
                }
              ]
            }
          }
        `,
	}

	testAllEncodeAndDecode(t, simpleContract, resourceContract)
}

func TestEncodeSimpleTypes(t *testing.T) {

	t.Parallel()

	var tests []encodeTest

	for _, ty := range []cadence.Type{
		cadence.AnyType,
		cadence.TheBytesType,
	} {
		tests = append(tests, encodeTest{
			name: fmt.Sprintf("with static %s", ty.ID()),
			val: cadence.TypeValue{
				StaticType: ty,
			},
			// language=json
			expected: fmt.Sprintf(`{"type":"Type","value":{"staticType":{"kind":"%s"}}}`, ty.ID()),
		})
	}

	testAllEncodeAndDecode(t, tests...)
}

func TestEncodeType(t *testing.T) {

	t.Parallel()

	t.Run("with static int?", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.OptionalType{Type: cadence.IntType},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Optional",
                    "type": {
                      "kind": "Int"
                    }
                  }
                }
              }
            `,
		)

	})

	t.Run("with static [int]", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.VariableSizedArrayType{ElementType: cadence.IntType},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "VariableSizedArray",
                    "type": {
                      "kind": "Int"
                    }
                  }
                }
              }
            `,
		)

	})

	t.Run("with static [int; 3]", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ConstantSizedArrayType{
					ElementType: cadence.IntType,
					Size:        3,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "ConstantSizedArray",
                    "type": {
                      "kind": "Int"
                    },
                    "size": 3
                  }
                }
              }
            `,
		)

	})

	t.Run("with static {int:string}", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.DictionaryType{
					ElementType: cadence.StringType,
					KeyType:     cadence.IntType,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Dictionary",
                    "key": {
                      "kind": "Int"
                    },
                    "value": {
                      "kind": "String"
                    }
                  }
                }
              }
            `,
		)

	})

	t.Run("with static InclusiveRange<Int>", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.InclusiveRangeType{
					ElementType: cadence.IntType,
				},
			},
			// language=json
			`
				{
				"type": "Type",
				"value": {
					"staticType": {
					"kind": "InclusiveRange",
					"element": {
						"kind": "Int"
					}
					}
				}
				}
			`,
		)

	})

	t.Run("with static struct", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewStructType(
					TestLocation,
					"S",
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[][]cadence.Parameter{
						{{Label: "foo", Identifier: "bar", Type: cadence.IntType}},
						{{Label: "qux", Identifier: "baz", Type: cadence.StringType}},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Struct",
                    "type": "",
                    "typeID": "S.test.S",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        }
                      ],
                      [
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static resource", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewResourceType(
					TestLocation,
					"R",
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[][]cadence.Parameter{
						{{Label: "foo", Identifier: "bar", Type: cadence.IntType}},
						{{Label: "qux", Identifier: "baz", Type: cadence.StringType}},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Resource",
                    "type": "",
                    "typeID": "S.test.R",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        }
                      ],
                      [
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static contract", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewContractType(
					TestLocation,
					"C",
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[][]cadence.Parameter{
						{{Label: "foo", Identifier: "bar", Type: cadence.IntType}},
						{{Label: "qux", Identifier: "baz", Type: cadence.StringType}},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Contract",
                    "type": "",
                    "typeID": "S.test.C",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        }
                      ],
                      [
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static struct interface", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewStructInterfaceType(
					TestLocation,
					"S",
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[][]cadence.Parameter{
						{{Label: "foo", Identifier: "bar", Type: cadence.IntType}},
						{{Label: "qux", Identifier: "baz", Type: cadence.StringType}},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "StructInterface",
                    "type": "",
                    "typeID": "S.test.S",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        }
                      ],
                      [
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static resource interface", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewResourceInterfaceType(
					TestLocation,
					"R",
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[][]cadence.Parameter{
						{{Label: "foo", Identifier: "bar", Type: cadence.IntType}},
						{{Label: "qux", Identifier: "baz", Type: cadence.StringType}},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "ResourceInterface",
                    "type": "",
                    "typeID": "S.test.R",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        }
                      ],
                      [
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static contract interface", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewContractInterfaceType(
					TestLocation,
					"C",
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[][]cadence.Parameter{
						{{Label: "foo", Identifier: "bar", Type: cadence.IntType}},
						{{Label: "qux", Identifier: "baz", Type: cadence.StringType}},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "ContractInterface",
                    "type": "",
                    "typeID": "S.test.C",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        }
                      ],
                      [
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static event", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewEventType(
					TestLocation,
					"E",
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[]cadence.Parameter{
						{Label: "foo", Identifier: "bar", Type: cadence.IntType},
						{Label: "qux", Identifier: "baz", Type: cadence.StringType},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Event",
                    "type": "",
                    "typeID": "S.test.E",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        },
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static enum", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: cadence.NewEnumType(
					TestLocation,
					"E",
					cadence.StringType,
					[]cadence.Field{
						{Identifier: "foo", Type: cadence.IntType},
					},
					[][]cadence.Parameter{
						{{Label: "foo", Identifier: "bar", Type: cadence.IntType}},
						{{Label: "qux", Identifier: "baz", Type: cadence.StringType}},
					},
				),
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Enum",
                    "type": {
                      "kind": "String"
                    },
                    "typeID": "S.test.E",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Int"
                        }
                      }
                    ],
                    "initializers": [
                      [
                        {
                          "label": "foo",
                          "id": "bar",
                          "type": {
                            "kind": "Int"
                          }
                        }
                      ],
                      [
                        {
                          "label": "qux",
                          "id": "baz",
                          "type": {
                            "kind": "String"
                          }
                        }
                      ]
                    ]
                  }
                }
              }
            `,
		)
	})

	t.Run("with static &int", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ReferenceType{
					Authorization: cadence.UnauthorizedAccess,
					Type:          cadence.IntType,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Reference",
                    "type": {
                      "kind": "Int"
                    },
                    "authorization": {
						"kind": "Unauthorized",
						"entitlements": null
					}
                  }
                }
              }
            `,
		)
	})

	t.Run("with static auth(foo) &int", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ReferenceType{
					Authorization: cadence.EntitlementMapAuthorization{
						TypeID: "foo",
					},
					Type: cadence.IntType,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Reference",
                    "type": {
                      "kind": "Int"
                    },
                    "authorization": {
						"kind": "EntitlementMapAuthorization",
						"entitlements": [
							{
								"kind": "EntitlementMap",
								"typeID": "foo",
								"type": null,
								"fields": null, 
								"initializers": null
							}
						]
					}
                  }
                }
              }
            `,
		)
	})

	t.Run("with static auth(X, Y) &int", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ReferenceType{
					Authorization: &cadence.EntitlementSetAuthorization{
						Kind:         cadence.Conjunction,
						Entitlements: []common.TypeID{"X", "Y"},
					},
					Type: cadence.IntType,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Reference",
                    "type": {
                      "kind": "Int"
                    },
                    "authorization": {
						"kind": "EntitlementConjunctionSet",
						"entitlements": [
							{
								"kind": "Entitlement",
								"typeID": "X",
								"type": null,
								"fields": null, 
								"initializers": null
							},
							{
								"kind": "Entitlement",
								"typeID": "Y",
								"type": null,
								"fields": null, 
								"initializers": null
							}
						]
					}
                  }
                }
              }
            `,
		)
	})

	t.Run("with static auth(X | Y) &int", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ReferenceType{
					Authorization: &cadence.EntitlementSetAuthorization{
						Kind:         cadence.Disjunction,
						Entitlements: []common.TypeID{"X", "Y"},
					},
					Type: cadence.IntType,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Reference",
                    "type": {
                      "kind": "Int"
                    },
                    "authorization": {
						"kind": "EntitlementDisjunctionSet",
						"entitlements": [
							{
								"kind": "Entitlement",
								"typeID": "X",
								"type": null,
								"fields": null, 
								"initializers": null
							},
							{
								"kind": "Entitlement",
								"typeID": "Y",
								"type": null,
								"fields": null, 
								"initializers": null
							}
						]
					}
                  }
                }
              }
            `,
		)
	})

	t.Run("with static function, with type parameters", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.FunctionType{
					TypeParameters: []cadence.TypeParameter{
						{Name: "T", TypeBound: cadence.AnyStructType},
					},
					Parameters: []cadence.Parameter{
						{Label: "qux", Identifier: "baz", Type: cadence.StringType},
					},
					ReturnType: cadence.IntType,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Function",
					"purity": "",
                    "typeID": "fun<T:AnyStruct>(String):Int",
                    "return": {
                      "kind": "Int"
                    },
                    "typeParameters": [
                      {
                        "name": "T",
                        "typeBound": {
                          "kind": "AnyStruct"
                        }
                      }
                    ],
                    "parameters": [
                      {
                        "label": "qux",
                        "id": "baz",
                        "type": {
                          "kind": "String"
                        }
                      }
                    ]
                  }
                }
              }
            `,
		)

	})

	t.Run("with view static function", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.FunctionType{
					Purity: cadence.FunctionPurityView,
					Parameters: []cadence.Parameter{
						{Label: "qux", Identifier: "baz", Type: cadence.StringType},
					},
					ReturnType:     cadence.IntType,
					TypeParameters: []cadence.TypeParameter{},
				},
			},
			`{"type":"Type","value":{"staticType":
				{	
					"kind" : "Function",
					"purity": "view",
                    "typeID": "view fun(String):Int",
					"return" : {"kind" : "Int"},
					"typeParameters": [],
					"parameters" : [
						{"label" : "qux", "id" : "baz", "type": {"kind" : "String"}}
					]}
				}
			}`,
		)

	})

	t.Run("with static function, without type parameters (decode only)", func(t *testing.T) {

		testDecode(
			t,
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Function",
                    "typeID": "((String):Int)",
                    "return": {
                      "kind": "Int"
                    },
                    "parameters": [
                      {
                        "label": "qux",
                        "id": "baz",
                        "type": {
                          "kind": "String"
                        }
                      }
                    ]
                  }
                }
              }
            `,
			cadence.TypeValue{
				StaticType: &cadence.FunctionType{
					Parameters: []cadence.Parameter{
						{Label: "qux", Identifier: "baz", Type: cadence.StringType},
					},
					ReturnType: cadence.IntType,
				},
			},
		)
	})

	t.Run("with implicit purity", func(t *testing.T) {

		encodedValue := `{"type":"Type","value":{"staticType":
			{	
				"kind" : "Function",
				"return" : {"kind" : "Int"},
				"typeParameters": [],
				"parameters" : [
					{"label" : "qux", "id" : "baz", "type": {"kind" : "String"}}
				]}
			}
		}`

		value := cadence.TypeValue{
			StaticType: &cadence.FunctionType{
				Parameters: []cadence.Parameter{
					{Label: "qux", Identifier: "baz", Type: cadence.StringType},
				},
				ReturnType:     cadence.IntType,
				TypeParameters: []cadence.TypeParameter{},
			},
		}

		decodedValue, err := Decode(nil, []byte(encodedValue))
		require.NoError(t, err)
		require.Equal(t, value, decodedValue)
	})

	t.Run("with static Capability<Int>", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.CapabilityType{
					BorrowType: cadence.IntType,
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Capability",
                    "type": {
                      "kind": "Int"
                    }
                  }
                }
              }
            `,
		)

	})

	t.Run("with static intersection type", func(t *testing.T) {

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.IntersectionType{
					Types: []cadence.Type{
						cadence.StringType,
					},
				},
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Intersection",
                    "typeID": "{String}",
                    "types": [
                      {
                        "kind": "String"
                      }
                    ]
                  }
                }
              }
            `,
		)

	})

	t.Run("without static type", func(t *testing.T) {

		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{},
			// language=json
			`{"type":"Type","value":{"staticType":""}}`,
		)
	})
}

func TestEncodeCapability(t *testing.T) {

	t.Parallel()

	t.Run("valid capability", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.NewCapability(
				6,
				cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
				cadence.IntType,
			),
			// language=json
			`
          {
            "type": "Capability",
            "value": {
              "borrowType": {
                "kind": "Int"
              },
              "address": "0x0000000102030405",
              "id": "6"
            }
          }
        `,
		)
	})

	t.Run("deprecated path capability", func(t *testing.T) {
		t.Parallel()

		testEncode(
			t,
			cadence.NewDeprecatedPathCapability( //nolint:staticcheck
				cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
				cadence.MustNewPath(common.PathDomainPublic, "foo"),
				cadence.AnyResourceType,
			),
			// language=json
			`
          {
            "type": "Capability",
            "value": {
              "path": {
                "type": "Path",
                "value": {
                  "domain": "public",
                  "identifier": "foo"
                }
              },
              "borrowType": {
                "kind": "AnyResource"
              },
              "address": "0x0000000102030405",
              "id": "0"
            }
          }
        `,
		)
	})
}

func TestDecodeCapability(t *testing.T) {

	t.Run("with backwards compatibility", func(t *testing.T) {
		t.Parallel()

		testDecode(
			t,
			// language=json
			`
		  {
		    "type": "Capability",
		    "value": {
		      "borrowType": {
		        "kind": "Int"
		      },
		      "address": "0x0000000102030405",
		      "id": "6"
		    }
		  }
        `,
			cadence.NewCapability(
				6,
				cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
				cadence.IntType,
			),
			WithBackwardsCompatibility(),
		)
	})

	t.Run("with backwards compatibility on a deprecated Path Capability", func(t *testing.T) {
		t.Parallel()

		testDecode(
			t,
			// language=json
			`
			{
			  "type": "Capability",
			  "value": {
				"path": {
				  "type": "Path",
				  "value": {
					"domain": "public",
					"identifier": "foo"
				  }
				},
				"borrowType": {
				  "kind": "Int"
				},
				"address": "0x0000000102030405"
			  }
			}
		  `,
			cadence.NewDeprecatedPathCapability( //nolint:staticcheck
				cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
				cadence.Path{
					Domain:     common.PathDomainPublic,
					Identifier: "foo",
				},
				cadence.IntType,
			),
			WithBackwardsCompatibility(),
		)
	})

	t.Run("deprecated Path Capability without backwards compatibility", func(t *testing.T) {
		t.Parallel()

		_, err := Decode(nil, []byte(
			`
			{
			  "type": "Capability",
			  "value": {
				"path": {
				  "type": "Path",
				  "value": {
					"domain": "public",
					"identifier": "foo"
				  }
				},
				"borrowType": {
				  "kind": "Int"
				},
				"address": "0x0000000102030405"
			  }
			}
		  `,
		))
		require.Error(t, err)

	})
}

func TestDecodeFixedPoints(t *testing.T) {

	t.Parallel()

	allFixedPointTypes := map[cadence.Type]struct {
		constructor func(int) cadence.Value
		maxInt      int64
		minInt      int64
		maxFrac     int64
		minFrac     int64
	}{
		cadence.Fix64Type: {
			constructor: func(i int) cadence.Value { return cadence.Fix64(int64(i)) },
			maxInt:      sema.Fix64TypeMaxInt,
			minInt:      sema.Fix64TypeMinInt,
			maxFrac:     sema.Fix64TypeMaxFractional,
			minFrac:     sema.Fix64TypeMinFractional,
		},
		cadence.UFix64Type: {
			constructor: func(i int) cadence.Value { return cadence.UFix64(uint64(i)) },
			maxInt:      int64(sema.UFix64TypeMaxInt),
			minInt:      sema.UFix64TypeMinInt,
			maxFrac:     int64(sema.UFix64TypeMaxFractional),
			minFrac:     sema.UFix64TypeMinFractional,
		},
	}

	type test struct {
		check    func(t *testing.T, actual cadence.Value, err error)
		input    string
		expected int
	}

	for ty, params := range allFixedPointTypes {
		t.Run(ty.ID(), func(t *testing.T) {

			var tests = []test{
				{
					input: "12.300000000",
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.Error(t, err)
					},
				},
				{
					input:    "12.30000000",
					expected: 12_30000000,
				},
				{
					input:    "12.3000000",
					expected: 12_30000000,
				},
				{
					input:    "12.300000",
					expected: 12_30000000,
				},
				{
					input:    "12.30000",
					expected: 12_30000000,
				},
				{
					input:    "12.3000",
					expected: 12_30000000,
				},
				{
					input:    "12.300",
					expected: 12_30000000,
				},
				{
					input:    "12.30",
					expected: 12_30000000,
				},
				{
					input:    "12.3",
					expected: 12_30000000,
				},
				{
					input:    "12.03",
					expected: 12_03000000,
				},
				{
					input:    "12.003",
					expected: 12_00300000,
				},
				{
					input:    "12.0003",
					expected: 12_00030000,
				},
				{
					input:    "12.00003",
					expected: 12_00003000,
				},
				{
					input:    "12.000003",
					expected: 12_00000300,
				},
				{
					input:    "12.0000003",
					expected: 12_00000030,
				},
				{
					input:    "12.00000003",
					expected: 12_00000003,
				},
				{
					input: "12.000000003",
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.Error(t, err)
					},
				},
				{
					input:    "120.3",
					expected: 120_30000000,
				},
				{
					input:    "012.3",
					expected: 12_30000000,
				},
				{
					input: fmt.Sprintf("%d.1", params.maxInt),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.NoError(t, err)
					},
				},
				{
					input: fmt.Sprintf("%d.1", params.maxInt+1),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.Error(t, err)
					},
				},
				{
					input: fmt.Sprintf("%d.1", params.minInt),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.NoError(t, err)
					},
				},
				{
					input: fmt.Sprintf("%d.1", params.minInt-1),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.Error(t, err)
					},
				},
				{
					input: fmt.Sprintf("%d.%d", params.maxInt, params.maxFrac),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.NoError(t, err)
					},
				},
				{
					input: fmt.Sprintf("%d.%d", params.maxInt, params.maxFrac+1),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.Error(t, err)
					},
				},
				{
					input: fmt.Sprintf("%d.%d", params.minInt, -(params.minFrac)),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.NoError(t, err)
					},
				},
			}

			if params.minFrac != 0 {
				tests = append(tests, test{
					input: fmt.Sprintf("%d.%d", params.minInt, -(params.minFrac - 1)),
					check: func(t *testing.T, actual cadence.Value, err error) {
						assert.Error(t, err)
					},
				})
			}

			for _, tt := range tests {
				t.Run(tt.input, func(t *testing.T) {

					// language=json
					enc := fmt.Sprintf(`{"type": "%s", "value": "%s"}`, ty.ID(), tt.input)

					actual, err := Decode(nil, []byte(enc))

					if tt.check != nil {
						tt.check(t, actual, err)
					} else {
						require.NoError(t, err)
						assert.Equal(t, params.constructor(tt.expected), actual)
					}
				})
			}
		})
	}

	t.Run("minus sign in fractional", func(t *testing.T) {

		t.Parallel()

		// language=json
		_, err := Decode(nil, []byte(`{"type": "Fix64", "value": "1.-1"}`))
		assert.Error(t, err)
	})

	t.Run("plus sign in fractional", func(t *testing.T) {

		t.Parallel()

		// language=json
		_, err := Decode(nil, []byte(`{"type": "Fix64", "value": "1.+1"}`))
		assert.Error(t, err)
	})

	t.Run("missing integer", func(t *testing.T) {

		t.Parallel()

		// language=json
		_, err := Decode(nil, []byte(`{"type": "Fix64", "value": ".1"}`))
		assert.Error(t, err)
	})

	t.Run("missing fractional", func(t *testing.T) {

		t.Parallel()

		// language=json
		_, err := Decode(nil, []byte(`{"type": "Fix64", "value": "1."}`))
		assert.Error(t, err)
	})
}

func TestDecodeDeprecatedTypes(t *testing.T) {

	t.Parallel()

	t.Run("with static reference type", func(t *testing.T) {

		t.Parallel()

		testDecode(
			t,
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Reference",
                    "type": {
                      "kind": "Int"
                    },
                    "authorized": true 
                  }
                }
              }
            `,
			cadence.TypeValue{
				StaticType: &cadence.DeprecatedReferenceType{
					Authorized: true,
					Type:       cadence.IntType,
				},
			},
			WithBackwardsCompatibility(),
		)
	})

	t.Run("with static reference type without backwards compatibility", func(t *testing.T) {

		t.Parallel()

		// Decode with error if reference is not supported
		_, err := Decode(nil, []byte(`
	              {
	                "type": "Type",
	                "value": {
	                  "staticType": {
	                    "kind": "Reference",
	                    "type": {
	                      "kind": "Int"
	                    },
	                    "authorized": true 
	                  }
	                }
	              }
	            `))
		require.Error(t, err)
	})

	t.Run("with static restricted type", func(t *testing.T) {

		t.Parallel()

		testDecode(
			t,
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Restriction",
                    "typeID": "Int{String}",
                    "type": {
                      "kind": "Int"
                    },
                    "restrictions": [
                      {
                        "kind": "String"
                      }
                    ]
                  }
                }
              }
            `,
			cadence.TypeValue{
				StaticType: &cadence.DeprecatedRestrictedType{
					Restrictions: []cadence.Type{
						cadence.StringType,
					},
					Type: cadence.IntType,
				},
			},
			WithBackwardsCompatibility(),
		)
	})

	t.Run("with static restricted type without backwards compatibility", func(t *testing.T) {

		t.Parallel()

		testDecode(
			t,
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Restriction",
                    "typeID": "Int{String}",
                    "type": {
                      "kind": "Int"
                    },
                    "restrictions": [
                      {
                        "kind": "String"
                      }
                    ]
                  }
                }
              }
            `,
			cadence.TypeValue{
				StaticType: &cadence.DeprecatedRestrictedType{
					Restrictions: []cadence.Type{
						cadence.StringType,
					},
					Type: cadence.IntType,
				},
			},
			WithBackwardsCompatibility(),
		)
	})
}

func TestExportRecursiveType(t *testing.T) {

	t.Parallel()

	fields := []cadence.Field{
		{
			Identifier: "foo",
		},
	}
	ty := cadence.NewResourceType(
		TestLocation,
		"Foo",
		fields,
		nil,
	)

	fields[0].Type = &cadence.OptionalType{
		Type: ty,
	}

	testEncode(
		t,
		cadence.NewResource([]cadence.Value{
			cadence.Optional{},
		}).WithType(ty),
		// language=json
		`
          {
            "type": "Resource",
            "value": {
              "id": "S.test.Foo",
              "fields": [
                {
                  "name": "foo",
                  "value": {
                    "type": "Optional",
                    "value": null
                  }
                }
              ]
            }
          }
        `,
	)

}

func TestExportTypeValueRecursiveType(t *testing.T) {

	t.Parallel()

	t.Run("recursive", func(t *testing.T) {

		t.Parallel()

		fields := []cadence.Field{
			{
				Identifier: "foo",
			},
		}
		ty := cadence.NewResourceType(
			TestLocation,
			"Foo",
			fields,
			[][]cadence.Parameter{},
		)

		fields[0].Type = &cadence.OptionalType{
			Type: ty,
		}

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: ty,
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Resource",
                    "typeID": "S.test.Foo",
                    "fields": [
                      {
                        "id": "foo",
                        "type": {
                          "kind": "Optional",
                          "type": "S.test.Foo"
                        }
                      }
                    ],
                    "initializers": [],
                    "type": ""
                  }
                }
              }
            `,
		)

	})

	t.Run("non-recursive, repeated", func(t *testing.T) {

		t.Parallel()

		fooTy := cadence.NewResourceType(
			TestLocation,
			"Foo",
			[]cadence.Field{},
			[][]cadence.Parameter{},
		)

		barTy := cadence.NewResourceType(
			TestLocation,
			"Bar",
			[]cadence.Field{
				{
					Identifier: "foo1",
					Type:       fooTy,
				},
				{
					Identifier: "foo2",
					Type:       fooTy,
				},
			},
			[][]cadence.Parameter{},
		)

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: barTy,
			},
			// language=json
			`
              {
                "type": "Type",
                "value": {
                  "staticType": {
                    "kind": "Resource",
                    "typeID": "S.test.Bar",
                    "fields": [
                      {
                        "id": "foo1",
                        "type": {
                          "kind": "Resource",
                          "typeID": "S.test.Foo",
                          "fields": [],
                          "initializers": [],
                          "type": ""
                        }
                      },
                      {
                        "id": "foo2",
                        "type": "S.test.Foo"
                      }
                    ],
                    "initializers": [],
                    "type": ""
                  }
                }
              }
            `,
		)
	})
}

func TestEncodePath(t *testing.T) {

	t.Parallel()

	t.Run("storage", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			// language=json
			`{"type":"Path","value":{"domain":"storage","identifier":"foo"}}`,
		)
	})

	t.Run("private", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			// language=json
			`{"type":"Path","value":{"domain":"private","identifier":"foo"}}`,
		)
	})

	t.Run("public", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			// language=json
			`{"type":"Path","value":{"domain":"public","identifier":"foo"}}`,
		)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		_, err := Decode(nil, []byte(
			// language=json
			`{"type":"Path","value":{"domain":"Storage","identifier":"foo"}}`,
		))
		require.ErrorContains(t, err, "unknown domain in path")
	})
}

func testAllEncodeAndDecode(t *testing.T, tests ...encodeTest) {

	test := func(testCase encodeTest) {

		t.Run(testCase.name, func(t *testing.T) {

			t.Parallel()

			testEncodeAndDecode(t, testCase.val, testCase.expected)
		})
	}

	for _, testCase := range tests {
		test(testCase)
	}
}

func TestDecodeInvalidType(t *testing.T) {

	t.Parallel()

	t.Run("empty type", func(t *testing.T) {
		t.Parallel()

		// language=json
		encodedValue := `
          {
            "type": "Struct",
            "value": {
              "id": "",
              "fields": []
            }
          }
        `
		_, err := Decode(nil, []byte(encodedValue))
		require.Error(t, err)
		assert.Equal(t, "failed to decode JSON-Cadence value: invalid type ID for built-in: ``", err.Error())
	})

	t.Run("invalid type ID", func(t *testing.T) {
		t.Parallel()

		// language=json
		encodedValue := `
          {
            "type": "Struct",
            "value": {
              "id": "I",
              "fields": []
            }
          }
        `
		_, err := Decode(nil, []byte(encodedValue))
		require.Error(t, err)
		assert.Equal(t, "failed to decode JSON-Cadence value: invalid type ID `I`: invalid identifier location type ID: missing location", err.Error())
	})

	t.Run("unknown location prefix", func(t *testing.T) {
		t.Parallel()

		// language=json
		encodedValue := `
          {
            "type": "Struct",
            "value":{
              "id": "N.PublicKey",
              "fields": []
            }
          }
        `
		_, err := Decode(nil, []byte(encodedValue))
		require.Error(t, err)
		assert.Equal(t, "failed to decode JSON-Cadence value: invalid type ID for built-in: `N.PublicKey`", err.Error())
	})
}

func testEncodeAndDecode(t *testing.T, val cadence.Value, expectedJSON string) {
	actualJSON := testEncode(t, val, expectedJSON)
	testDecode(t, actualJSON, val)
}

func testEncode(t *testing.T, val cadence.Value, expectedJSON string) (actualJSON string) {
	actualJSONBytes, err := Encode(val)
	require.NoError(t, err)

	actualJSON = string(actualJSONBytes)

	assert.JSONEq(t, expectedJSON, actualJSON, fmt.Sprintf("actual: %s", actualJSON))

	return actualJSON
}

func testDecode(t *testing.T, actualJSON string, expectedVal cadence.Value, options ...Option) {
	decodedVal, err := Decode(nil, []byte(actualJSON), options...)
	require.NoError(t, err)

	assert.Equal(
		t,
		expectedVal,
		decodedVal,
	)
}

func newFooResourceType() *cadence.ResourceType {
	return cadence.NewResourceType(
		TestLocation,
		"Foo",
		[]cadence.Field{
			{
				Identifier: "bar",
				Type:       cadence.IntType,
			},
		},
		nil,
	)
}

func TestNonUTF8StringEncoding(t *testing.T) {

	t.Parallel()

	nonUTF8String := "\xbd\xb2\x3d\xbc\x20\xe2"

	// Make sure it is an invalid utf8 string
	assert.False(t, utf8.ValidString(nonUTF8String))

	// Avoid using the `NewMeteredString()` constructor to skip the validation
	stringValue := cadence.String(nonUTF8String)

	encodedValue, err := Encode(stringValue)
	require.NoError(t, err)

	decodedValue, err := Decode(nil, encodedValue)
	require.NoError(t, err)

	// Decoded value must be a valid utf8 string
	assert.IsType(t, cadence.String(""), decodedValue)
	assert.True(t, utf8.ValidString(decodedValue.String()))
}

func TestDecodeBackwardsCompatibilityTypeID(t *testing.T) {

	t.Parallel()

	// language=json
	encoded := `{"type":"Type","value":{"staticType":"&Int"}}`

	t.Run("unstructured static types allowed", func(t *testing.T) {

		t.Parallel()

		testDecode(
			t,
			encoded,

			cadence.TypeValue{
				StaticType: cadence.TypeID("&Int"),
			},
			WithAllowUnstructuredStaticTypes(true),
		)
	})

	t.Run("unstructured static types disallowed", func(t *testing.T) {

		t.Parallel()

		_, err := Decode(nil, []byte(encoded))
		require.Error(t, err)
	})

}

func TestEncodeBuiltinComposites(t *testing.T) {
	t.Parallel()

	type staticType struct {
		typ  cadence.Type
		kind string
	}

	types := []staticType{
		{
			typ: &cadence.StructType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "Struct",
		},
		{
			typ: &cadence.StructInterfaceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "StructInterface",
		},
		{
			typ: &cadence.ResourceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "Resource",
		},
		{
			typ: &cadence.ResourceInterfaceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "ResourceInterface",
		},
		{
			typ: &cadence.ContractType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "Contract",
		},
		{
			typ: &cadence.ContractInterfaceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "ContractInterface",
		},
		{
			typ: &cadence.EnumType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "Enum",
		},
		{
			typ: &cadence.EventType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			kind: "Event",
		},
	}

	// language=json
	compositeJsonTemplate := `
      {
        "type": "Type",
        "value": {
          "staticType": {
            "kind": "%s",
            "typeID": "Foo",
            "fields": [],
            "initializers": [],
            "type": ""
          }
        }
      }
    `

	// language=json
	eventJson := `
      {
        "type": "Type",
        "value": {
          "staticType": {
            "kind": "Event",
            "typeID": "Foo",
            "fields": [],
            "initializers": [
              []
            ],
            "type": ""
          }
        }
      }
    `

	for _, typ := range types {
		typeValue := cadence.NewTypeValue(typ.typ)
		var expectedJson string

		switch typ.typ.(type) {
		case *cadence.EventType:
			expectedJson = eventJson
		default:
			expectedJson = fmt.Sprintf(compositeJsonTemplate, typ.kind)
		}

		testEncode(t, typeValue, expectedJson)
	}
}

func TestExportFunctionValue(t *testing.T) {

	t.Parallel()

	testEncode(
		t,
		cadence.Function{
			FunctionType: &cadence.FunctionType{
				Parameters: []cadence.Parameter{},
				ReturnType: cadence.VoidType,
			},
		},
		// language=json
		`
          {
            "type": "Function",
            "value": {
              "functionType": {
                "kind": "Function",
                "typeID": "fun():Void",
                "parameters": [],
                "typeParameters": [],
                "purity":"",
                "return": {
                  "kind": "Void"
                }
              }
            }
          }
        `,
	)
}

func TestImportFunctionValue(t *testing.T) {

	t.Parallel()

	t.Run("without type parameters", func(t *testing.T) {

		t.Parallel()

		testDecode(
			t,
			// language=json
			`
              {
                "type": "Function",
                "value": {
                  "functionType": {
                    "kind": "Function",
                    "typeID": "(():Void)",
                    "parameters": [],
                    "return": {
                      "kind": "Void"
                    }
                  }
                }
              }
            `,
			cadence.Function{
				FunctionType: &cadence.FunctionType{
					Parameters: []cadence.Parameter{},
					ReturnType: cadence.VoidType,
				},
			},
		)
	})

	t.Run("with type parameters", func(t *testing.T) {

		t.Parallel()

		testDecode(
			t,
			// language=json
			`
              {
                "type": "Function",
                "value": {
                  "functionType": {
                    "kind": "Function",
                    "typeID": "(<T>():Void)",
                    "typeParameters": [
                      {"name": "T"}
                    ],
                    "parameters": [],
                    "return": {
                      "kind": "Void"
                    }
                  }
                }
              }
            `,
			cadence.Function{
				FunctionType: &cadence.FunctionType{
					TypeParameters: []cadence.TypeParameter{
						{Name: "T"},
					},
					Parameters: []cadence.Parameter{},
					ReturnType: cadence.VoidType,
				},
			},
		)
	})

}

func TestSimpleTypes(t *testing.T) {
	t.Parallel()

	test := func(cadenceType cadence.PrimitiveType, semaType sema.Type) {

		t.Run(semaType.QualifiedString(), func(t *testing.T) {
			t.Parallel()

			prepared := PrepareType(cadenceType, TypePreparationResults{})
			require.IsType(t, jsonSimpleType{}, prepared)

			encoded, err := Encode(cadence.NewTypeValue(cadenceType))
			require.NoError(t, err)

			decoded, err := Decode(nil, encoded)
			require.NoError(t, err)

			require.IsType(t, cadence.TypeValue{}, decoded)
			typeValue := decoded.(cadence.TypeValue)
			require.Equal(t, cadenceType, typeValue.StaticType)
		})
	}

	for ty := interpreter.PrimitiveStaticType(1); ty < interpreter.PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() || ty.IsDeprecated() { //nolint:staticcheck
			continue
		}

		semaType := ty.SemaType()

		cadenceType := cadence.PrimitiveType(ty)
		if !canEncodeAsSimpleType(cadenceType) {
			continue
		}

		test(cadenceType, semaType)
	}
}
