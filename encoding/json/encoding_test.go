/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package json_test

import (
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

type encodeTest struct {
	name     string
	val      cadence.Value
	expected string
}

func TestEncodeVoid(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(t, cadence.NewVoid(), `{"type":"Void"}`)
}

func TestEncodeOptional(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testEncodeAndDecode(
		t,
		cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
		`{"type":"Address","value":"0x0000000102030405"}`,
	)
}

func TestEncodeInt(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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
			cadence.NewIntFromBig(new(big.Int).Sub(sema.Int256TypeMinIntBig, big.NewInt(10))),
			`{"type":"Int","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819978"}`,
		},
		{
			"LargerThanMaxUInt256",
			cadence.NewIntFromBig(new(big.Int).Add(sema.UInt256TypeMaxIntBig, big.NewInt(10))),
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt128FromBig(sema.Int128TypeMinIntBig),
			`{"type":"Int128","value":"-170141183460469231731687303715884105728"}`,
		},
		{
			"Zero",
			cadence.NewInt128(0),
			`{"type":"Int128","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt128FromBig(sema.Int128TypeMaxIntBig),
			`{"type":"Int128","value":"170141183460469231731687303715884105727"}`,
		},
	}...)
}

func TestEncodeInt256(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Min",
			cadence.NewInt256FromBig(sema.Int256TypeMinIntBig),
			`{"type":"Int256","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819968"}`,
		},
		{
			"Zero",
			cadence.NewInt256(0),
			`{"type":"Int256","value":"0"}`,
		},
		{
			"Max",
			cadence.NewInt256FromBig(sema.Int256TypeMaxIntBig),
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
			`{"type":"UInt","value":"0"}`,
		},
		{
			"Positive",
			cadence.NewUInt(42),
			`{"type":"UInt","value":"42"}`,
		},
		{
			"LargerThanMaxUInt256",
			cadence.NewUIntFromBig(new(big.Int).Add(sema.UInt256TypeMaxIntBig, big.NewInt(10))),
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewUInt128(0),
			`{"type":"UInt128","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt128FromBig(sema.UInt128TypeMaxIntBig),
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
			`{"type":"UInt256","value":"0"}`,
		},
		{
			"Max",
			cadence.NewUInt256FromBig(sema.UInt256TypeMaxIntBig),
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
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

func TestEncodeWord64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.NewWord64(0),
			`{"type":"Word64","value":"0"}`,
		},
		{
			"Max",
			cadence.NewWord64(math.MaxUint64),
			`{"type":"Word64","value":"18446744073709551615"}`,
		},
	}...)
}

func TestEncodeFix64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			"Zero",
			cadence.Fix64(0),
			`{"type":"Fix64","value":"0.00000000"}`,
		},
		{
			"789.00123010",
			cadence.Fix64(78_900_123_010),
			`{"type":"Fix64","value":"789.00123010"}`,
		},
		{
			"1234.056",
			cadence.Fix64(123_405_600_000),
			`{"type":"Fix64","value":"1234.05600000"}`,
		},
		{
			"-12345.006789",
			cadence.Fix64(-1_234_500_678_900),
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
			`{"type":"UFix64","value":"0.00000000"}`,
		},
		{
			"789.00123010",
			cadence.UFix64(78_900_123_010),
			`{"type":"UFix64","value":"789.00123010"}`,
		},
		{
			"1234.056",
			cadence.UFix64(123_405_600_000),
			`{"type":"UFix64","value":"1234.05600000"}`,
		},
	}...)
}

func TestEncodeArray(t *testing.T) {

	t.Parallel()

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
		`{"type":"Array","value":[{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"1"}}]}},{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"2"}}]}},{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"3"}}]}}]}`,
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
				Value: cadence.NewResource([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(fooResourceType),
			},
			{
				Key: cadence.NewString("b"),
				Value: cadence.NewResource([]cadence.Value{
					cadence.NewInt(2),
				}).WithType(fooResourceType),
			},
			{
				Key: cadence.NewString("c"),
				Value: cadence.NewResource([]cadence.Value{
					cadence.NewInt(3),
				}).WithType(fooResourceType),
			},
		}),
		`{"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"1"}}]}}},{"key":{"type":"String","value":"b"},"value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"2"}}]}}},{"key":{"type":"String","value":"c"},"value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"3"}}]}}}]}`,
	}

	testAllEncodeAndDecode(t,
		simpleDict,
		nestedDict,
		resourceDict,
	)
}

func TestEncodeResource(t *testing.T) {

	t.Parallel()

	t.Run("Simple", func(t *testing.T) {

		t.Parallel()

		script := `
			pub resource Foo {
				pub let bar: Int
	
				init(bar: Int) {
					self.bar = bar
				}
			}
	
			pub fun main(): @Foo {
				return <- create Foo(bar: 42)
			}
		`

		expectedJSON := `{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"uuid","value":{"type":"UInt64","value":"0"}},{"name":"bar","value":{"type":"Int","value":"42"}}]}}`

		v := convertValueFromScript(t, script)

		testEncodeAndDecode(t, v, expectedJSON)
	})

	t.Run("With function member", func(t *testing.T) {

		t.Parallel()

		script := `
			pub resource Foo {
				pub let bar: Int
	
				pub fun foo(): String {
					return "foo"
				}
	
				init(bar: Int) {
					self.bar = bar
				}
			}
	
			pub fun main(): @Foo {
				return <- create Foo(bar: 42)
			}
		`

		// function "foo" should be omitted from resulting JSON
		expectedJSON := `{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"uuid","value":{"type":"UInt64","value":"0"}},{"name":"bar","value":{"type":"Int","value":"42"}}]}}`

		v := convertValueFromScript(t, script)

		actualJSON, err := json.Encode(v)
		require.NoError(t, err)

		assert.JSONEq(t, expectedJSON, string(actualJSON))
	})

	t.Run("Nested resource", func(t *testing.T) {

		t.Parallel()

		script := `
			pub resource Bar {
				pub let x: Int
	
				init(x: Int) {
					self.x = x
				}
			}
	
			pub resource Foo {
				pub let bar: @Bar
	
				init(bar: @Bar) {
					self.bar <- bar
				}
	
				destroy() {
					destroy self.bar
				}
			}
	
			pub fun main(): @Foo {
				return <- create Foo(bar: <- create Bar(x: 42))
			}
		`

		expectedJSON := `{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"uuid","value":{"type":"UInt64","value":"0"}},{"name":"bar","value":{"type":"Resource","value":{"id":"S.test.Bar","fields":[{"name":"uuid","value":{"type":"UInt64","value":"0"}},{"name":"x","value":{"type":"Int","value":"42"}}]}}}]}}`

		v := convertValueFromScript(t, script)

		testEncodeAndDecode(t, v, expectedJSON)
	})
}

func TestEncodeStruct(t *testing.T) {

	t.Parallel()

	simpleStructType := &cadence.StructType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooStruct",
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
	}

	simpleStruct := encodeTest{
		"Simple",
		cadence.NewStruct(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.NewString("foo"),
			},
		).WithType(simpleStructType),
		`{"type":"Struct","value":{"id":"S.test.FooStruct","fields":[{"name":"a","value":{"type":"Int","value":"1"}},{"name":"b","value":{"type":"String","value":"foo"}}]}}`,
	}

	resourceStructType := &cadence.StructType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooStruct",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.StringType{},
			},
			{
				Identifier: "b",
				Type:       fooResourceType,
			},
		},
	}

	resourceStruct := encodeTest{
		"Resources",
		cadence.NewStruct(
			[]cadence.Value{
				cadence.NewString("foo"),
				cadence.NewResource(
					[]cadence.Value{
						cadence.NewInt(42),
					},
				).WithType(fooResourceType),
			},
		).WithType(resourceStructType),
		`{"type":"Struct","value":{"id":"S.test.FooStruct","fields":[{"name":"a","value":{"type":"String","value":"foo"}},{"name":"b","value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"42"}}]}}}]}}`,
	}

	testAllEncodeAndDecode(t, simpleStruct, resourceStruct)
}

func TestEncodeEvent(t *testing.T) {

	t.Parallel()

	simpleEventType := &cadence.EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooEvent",
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
	}

	simpleEvent := encodeTest{
		"Simple",
		cadence.NewEvent(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.NewString("foo"),
			},
		).WithType(simpleEventType),
		`{"type":"Event","value":{"id":"S.test.FooEvent","fields":[{"name":"a","value":{"type":"Int","value":"1"}},{"name":"b","value":{"type":"String","value":"foo"}}]}}`,
	}

	resourceEventType := &cadence.EventType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooEvent",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.StringType{},
			},
			{
				Identifier: "b",
				Type:       fooResourceType,
			},
		},
	}

	resourceEvent := encodeTest{
		"Resources",
		cadence.NewEvent(
			[]cadence.Value{
				cadence.NewString("foo"),
				cadence.NewResource(
					[]cadence.Value{
						cadence.NewInt(42),
					},
				).WithType(fooResourceType),
			},
		).WithType(resourceEventType),
		`{"type":"Event","value":{"id":"S.test.FooEvent","fields":[{"name":"a","value":{"type":"String","value":"foo"}},{"name":"b","value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"42"}}]}}}]}}`,
	}

	testAllEncodeAndDecode(t, simpleEvent, resourceEvent)
}

func TestEncodeContract(t *testing.T) {

	t.Parallel()

	simpleContractType := &cadence.ContractType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooContract",
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
	}

	simpleContract := encodeTest{
		"Simple",
		cadence.NewContract(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.NewString("foo"),
			},
		).WithType(simpleContractType),
		`{"type":"Contract","value":{"id":"S.test.FooContract","fields":[{"name":"a","value":{"type":"Int","value":"1"}},{"name":"b","value":{"type":"String","value":"foo"}}]}}`,
	}

	resourceContractType := &cadence.ContractType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooContract",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.StringType{},
			},
			{
				Identifier: "b",
				Type:       fooResourceType,
			},
		},
	}

	resourceContract := encodeTest{
		"Resources",
		cadence.NewContract(
			[]cadence.Value{
				cadence.NewString("foo"),
				cadence.NewResource(
					[]cadence.Value{
						cadence.NewInt(42),
					},
				).WithType(fooResourceType),
			},
		).WithType(resourceContractType),
		`{"type":"Contract","value":{"id":"S.test.FooContract","fields":[{"name":"a","value":{"type":"String","value":"foo"}},{"name":"b","value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"42"}}]}}}]}}`,
	}

	testAllEncodeAndDecode(t, simpleContract, resourceContract)
}

func TestEncodeLink(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(
		t,
		cadence.NewLink(
			cadence.Path{Domain: "storage", Identifier: "foo"},
			"Bar",
		),
		`{"type":"Link","value":{"targetPath":{"type":"Path","value":{"domain":"storage","identifier":"foo"}},"borrowType":"Bar"}}`,
	)
}

func TestEncodeType(t *testing.T) {

	t.Parallel()

	t.Run("with static type", func(t *testing.T) {

		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: "Int",
			},
			`{"type":"Type","value":{"staticType":"Int"}}`,
		)

	})
	t.Run("without static type", func(t *testing.T) {

		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{},
			`{"type":"Type","value":{"staticType":""}}`,
		)
	})
}

func TestEncodeCapability(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(
		t,
		cadence.Capability{
			Path:       cadence.Path{Domain: "storage", Identifier: "foo"},
			Address:    cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
			BorrowType: "Int",
		},
		`{"type":"Capability","value":{"path":{"type":"Path","value":{"domain":"storage","identifier":"foo"}},"borrowType":"Int","address":"0x0000000102030405"}}`,
	)
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
		cadence.Fix64Type{}: {
			constructor: func(i int) cadence.Value { return cadence.Fix64(int64(i)) },
			maxInt:      sema.Fix64TypeMaxInt,
			minInt:      sema.Fix64TypeMinInt,
			maxFrac:     sema.Fix64TypeMaxFractional,
			minFrac:     sema.Fix64TypeMinFractional,
		},
		cadence.UFix64Type{}: {
			constructor: func(i int) cadence.Value { return cadence.UFix64(uint64(i)) },
			maxInt:      int64(sema.UFix64TypeMaxInt),
			minInt:      sema.UFix64TypeMinInt,
			maxFrac:     int64(sema.UFix64TypeMaxFractional),
			minFrac:     sema.UFix64TypeMinFractional,
		},
	}

	type test struct {
		input    string
		expected int
		check    func(t *testing.T, actual cadence.Value, err error)
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

					enc := fmt.Sprintf(`{ "type": "%s", "value": "%s"}`, ty.ID(), tt.input)

					actual, err := json.Decode([]byte(enc))

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

		_, err := json.Decode([]byte(`{"type": "Fix64", "value": "1.-1"}`))
		assert.Error(t, err)
	})

	t.Run("plus sign in fractional", func(t *testing.T) {

		t.Parallel()

		_, err := json.Decode([]byte(`{"type": "Fix64", "value": "1.+1"}`))
		assert.Error(t, err)
	})

	t.Run("missing integer", func(t *testing.T) {

		t.Parallel()

		_, err := json.Decode([]byte(`{"type": "Fix64", "value": ".1"}`))
		assert.Error(t, err)
	})

	t.Run("missing fractional", func(t *testing.T) {

		t.Parallel()

		_, err := json.Decode([]byte(`{"type": "Fix64", "value": "1."}`))
		assert.Error(t, err)
	})
}

func TestExportRecursiveType(t *testing.T) {

	t.Parallel()

	ty := &cadence.ResourceType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "foo",
			},
		},
	}

	ty.Fields[0].Type = cadence.OptionalType{
		Type: ty,
	}

	testEncode(
		t,
		cadence.Resource{
			Fields: []cadence.Value{
				cadence.Optional{},
			},
		}.WithType(ty),
		`{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"foo","value":{"type": "Optional","value":null}}]}}`,
	)

}

func TestEncodePath(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(
		t,
		cadence.Path{Domain: "storage", Identifier: "foo"},
		`{"type":"Path","value":{"domain":"storage","identifier":"foo"}}`,
	)
}

func convertValueFromScript(t *testing.T, script string) cadence.Value {
	rt := runtime.NewInterpreterRuntime()

	value, err := rt.ExecuteScript(
		runtime.Script{
			Source:    []byte(script),
			Arguments: nil,
		},
		runtime.Context{
			Interface: runtime.NewEmptyRuntimeInterface(),
			Location:  utils.TestLocation,
		},
	)

	require.NoError(t, err)

	return value
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

		encodedValue := `
		{
			"type":"Struct",
			"value":{
				"id":"",
				"fields":[]
			}
		}
	`
		_, err := json.Decode([]byte(encodedValue))
		require.Error(t, err)
		assert.Equal(t, "failed to decode value: invalid JSON Cadence structure. invalid type ID: ``", err.Error())
	})

	t.Run("undefined type", func(t *testing.T) {
		t.Parallel()

		encodedValue := `
		{
			"type":"Struct",
			"value":{
				"id":"I.Foo",
				"fields":[]
			}
		}
	`
		_, err := json.Decode([]byte(encodedValue))
		require.Error(t, err)
		assert.Equal(t, "failed to decode value: invalid JSON Cadence structure. invalid type ID: `I.Foo`", err.Error())
	})

	t.Run("unknown location prefix", func(t *testing.T) {
		t.Parallel()

		encodedValue := `
		{
			"type":"Struct",
			"value":{
				"id":"N.PublicKey",
				"fields":[]
			}
		}
	`
		_, err := json.Decode([]byte(encodedValue))
		require.Error(t, err)
		assert.Equal(t, "failed to decode value: invalid JSON Cadence structure. invalid type ID: `N.PublicKey`", err.Error())
	})
}

func testEncodeAndDecode(t *testing.T, val cadence.Value, expectedJSON string) {
	actualJSON := testEncode(t, val, expectedJSON)
	testDecode(t, actualJSON, val)
}

func testEncode(t *testing.T, val cadence.Value, expectedJSON string) (actualJSON string) {
	actualJSONBytes, err := json.Encode(val)
	require.NoError(t, err)

	actualJSON = string(actualJSONBytes)

	assert.JSONEq(t, expectedJSON, actualJSON)

	return actualJSON
}

func testDecode(t *testing.T, actualJSON string, expectedVal cadence.Value) {
	decodedVal, err := json.Decode([]byte(actualJSON))
	require.NoError(t, err)

	assert.Equal(t, expectedVal, decodedVal)
}

var fooResourceType = &cadence.ResourceType{
	Location:            utils.TestLocation,
	QualifiedIdentifier: "Foo",
	Fields: []cadence.Field{
		{
			Identifier: "bar",
			Type:       cadence.IntType{},
		},
	},
}
