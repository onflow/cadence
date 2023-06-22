/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package ccf_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/ccf"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/checker"
	"github.com/onflow/cadence/runtime/tests/utils"
)

var deterministicEncMode, _ = ccf.EncOptions{
	SortCompositeFields: ccf.SortBytewiseLexical,
	SortRestrictedTypes: ccf.SortBytewiseLexical,
}.EncMode()

var deterministicDecMode, _ = ccf.DecOptions{
	EnforceSortCompositeFields: ccf.EnforceSortBytewiseLexical,
	EnforceSortRestrictedTypes: ccf.EnforceSortBytewiseLexical,
}.DecMode()

type encodeTest struct {
	name        string
	val         cadence.Value
	expected    []byte
	expectedVal cadence.Value
}

func TestEncodeVoid(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(
		t,
		cadence.NewVoid(),
		[]byte{
			// language=json, format=json-cdc
			// {"type":"Void"}
			//
			// language=edn, format=ccf
			// 130([137(50), null])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// void type ID (50)
			0x18, 0x32,
			// nil
			0xf6,
		},
	)
}

func TestEncodeOptional(t *testing.T) {

	t.Parallel()

	// Factories instead of values to avoid data races,
	// as tests may run in parallel

	newStructType := func() *cadence.StructType {
		return &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.NewOptionalType(cadence.NewIntType()),
				},
				{
					Identifier: "b",
					Type:       cadence.NewOptionalType(cadence.NewOptionalType(cadence.NewIntType())),
				},
				{
					Identifier: "c",
					Type:       cadence.NewOptionalType(cadence.NewOptionalType(cadence.NewOptionalType(cadence.NewIntType()))),
				},
			},
		}
	}

	newStructTypeWithOptionalAbstractField := func() *cadence.StructType {
		return &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.NewOptionalType(cadence.NewAnyStructType()),
				},
				{
					Identifier: "b",
					Type:       cadence.NewOptionalType(cadence.NewOptionalType(cadence.NewAnyStructType())),
				},
				{
					Identifier: "c",
					Type:       cadence.NewOptionalType(cadence.NewOptionalType(cadence.NewOptionalType(cadence.NewAnyStructType()))),
				},
			},
		}
	}

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Optional(nil)",
			val:  cadence.NewOptional(nil),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Optional","value":null}
				//
				// language=edn, format=ccf
				// 130([138(137(42)), null])
				//
				// language=cbor, format=ccf, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// never type ID (42)
				0x18, 0x2a,
				// nil
				0xf6,
			},
		},
		{
			name: "Optional(Int)",
			val:  cadence.NewOptional(cadence.NewInt(42)),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Optional","value":{"type":"Int","value":"42"}}
				//
				// language=edn, format=ccf
				// 130([138(137(4)), 42])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 42
				0x2a,
			},
		},
		{
			name: "Optional(Optional(nil))",
			val:  cadence.NewOptional(cadence.NewOptional(nil)),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"value":null,"type":"Optional"},"type":"Optional"}
				//
				// language=edn, format=ccf
				// 130([138(138(137(42))), null])
				//
				// language=cbor, format=ccf, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// never type ID (42)
				0x18, 0x2a,
				// nil
				0xf6,
			},
		},
		{
			name: "Optional(Optional(Int))",
			val:  cadence.NewOptional(cadence.NewOptional(cadence.NewInt(42))),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"value":{"value":"42","type":"Int"},"type":"Optional"},"type":"Optional"}
				//
				// language=edn, format=ccf
				// 130([138(138(137(4))), 42])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 42
				0x2a,
			},
		},
		{
			name: "Optional(Optional(Optional(nil)))",
			val:  cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(nil))),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"value":{"value":null,"type":"Optional"},"type":"Optional"},"type":"Optional"}
				//
				// language=edn, format=ccf
				// 130([138(138(138(137(42)))), null])
				//
				// language=cbor, format=ccf, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// never type ID (42)
				0x18, 0x2a,
				// nil
				0xf6,
			},
		},
		{
			name: "Optional(Optional(Optional(int)))",
			val:  cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(cadence.NewInt(42)))),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"value":{"value":{"value":"42","type":"Int"},"type":"Optional"},"type":"Optional"},"type":"Optional"}
				//
				// language=edn, format=ccf
				// 130([138(138(138(137(4)))), 42])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 42
				0x2a,
			},
		},
		{
			name: "struct with nil optional fields",
			val: func() cadence.Value {
				structType := newStructType()
				return cadence.NewStruct([]cadence.Value{
					cadence.NewOptional(nil),
					cadence.NewOptional(cadence.NewOptional(nil)),
					cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(nil))),
				}).WithType(structType)
			}(),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"S.test.Foo","fields":[{"value":{"value":null,"type":"Optional"},"name":"a"},{"value":{"value":{"value":null,"type":"Optional"},"type":"Optional"},"name":"b"},{"value":{"value":{"value":{"value":null,"type":"Optional"},"type":"Optional"},"type":"Optional"},"name":"c"}]},"type":"Struct"}
				//
				// language=edn, format=ccf
				// 129([[160([h'', "S.test.Foo", [["a", 138(137(4))], ["b", 138(138(137(4)))], ["c", 138(138(138(137(4))))]]])], [136(h''), [null, null, null]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 1 items follow
				0x81,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.Foo"
				// fields: [["a", OptionalType(IntType)], ["b", OptionalType(OptionalType(IntType))], ["c", OptionalType(OptionalType(OptionalType(IntType)))]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// fields
				// array, 3 items follow
				0x83,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 2
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// c
				0x63,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 3 items follow
				0x83,
				// nil
				0xf6,
				// nil
				0xf6,
				// nil
				0xf6,
			},
		},
		{
			name: "struct with non-nil optional fields",
			val: func() cadence.Value {
				structType := newStructType()
				return cadence.NewStruct([]cadence.Value{
					cadence.NewOptional(cadence.NewInt(1)),
					cadence.NewOptional(cadence.NewOptional(cadence.NewInt(2))),
					cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(cadence.NewInt(3)))),
				}).WithType(structType)
			}(),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"S.test.Foo","fields":[{"value":{"value":{"value":"1","type":"Int"},"type":"Optional"},"name":"a"},{"value":{"value":{"value":{"value":"2","type":"Int"},"type":"Optional"},"type":"Optional"},"name":"b"},{"value":{"value":{"value":{"value":{"value":"3","type":"Int"},"type":"Optional"},"type":"Optional"},"type":"Optional"},"name":"c"}]},"type":"Struct"}
				//
				// language=edn, format=ccf
				// 129([[160([h'', "S.test.Foo", [["a", 138(137(4))], ["b", 138(138(137(4)))], ["c", 138(138(138(137(4))))]]])], [136(h''), [1, 2, 3]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 1 items follow
				0x81,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.Foo"
				// fields: [["a", OptionalType(IntType)], ["b", OptionalType(OptionalType(IntType))], ["c", OptionalType(OptionalType(OptionalType(IntType)))]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// fields
				// array, 3 items follow
				0x83,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 2
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// c
				0x63,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 3 items follow
				0x83,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 1
				0x01,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 2
				0x02,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 3
				0x03,
			},
		},
		{
			name: "struct with nil optional abstract fields",
			val: func() cadence.Value {
				typeWithOptionalAbstractField := newStructTypeWithOptionalAbstractField()
				return cadence.NewStruct([]cadence.Value{
					cadence.NewOptional(nil),
					cadence.NewOptional(cadence.NewOptional(nil)),
					cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(nil))),
				}).WithType(typeWithOptionalAbstractField)
			}(),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"S.test.Foo","fields":[{"value":{"value":null,"type":"Optional"},"name":"a"},{"value":{"value":{"value":null,"type":"Optional"},"type":"Optional"},"name":"b"},{"value":{"value":{"value":{"value":null,"type":"Optional"},"type":"Optional"},"type":"Optional"},"name":"c"}]},"type":"Struct"}
				//
				// language=edn, format=ccf
				// 129([[160([h'', "S.test.Foo", [["a", 138(137(39))], ["b", 138(138(137(39)))], ["c", 138(138(138(137(39))))]]])], [136(h''), [null, null, null]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 1 items follow
				0x81,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.Foo"
				// fields: [["a", OptionalType(IntType)], ["b", OptionalType(OptionalType(IntType))], ["c", OptionalType(OptionalType(OptionalType(IntType)))]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// fields
				// array, 3 items follow
				0x83,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// field 2
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// c
				0x63,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 3 items follow
				0x83,
				// nil
				0xf6,
				// nil
				0xf6,
				// nil
				0xf6,
			},
		},
		{
			name: "struct with optional Int for optional abstract fields",
			val: func() cadence.Value {
				structTypeWithOptionalAbstractField := newStructTypeWithOptionalAbstractField()
				return cadence.NewStruct([]cadence.Value{
					cadence.NewOptional(cadence.NewOptional(cadence.NewInt(1))),
					cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(cadence.NewInt(2)))),
					cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(cadence.NewInt(3))))),
				}).WithType(structTypeWithOptionalAbstractField)
			}(),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"S.test.Foo","fields":[{"value":{"value":{"value":{"value":"1","type":"Int"},"type":"Optional"},"type":"Optional"},"name":"a"},{"value":{"value":{"value":{"value":{"value":"2","type":"Int"},"type":"Optional"},"type":"Optional"},"type":"Optional"},"name":"b"},{"value":{"value":{"value":{"value":{"value":{"value":"3","type":"Int"},"type":"Optional"},"type":"Optional"},"type":"Optional"},"type":"Optional"},"name":"c"}]},"type":"Struct"}
				//
				// language=edn, format=ccf
				// 129([[160([h'', "S.test.Foo", [["a", 138(137(39))], ["b", 138(138(137(39)))], ["c", 138(138(138(137(39))))]]])], [136(h''), [130([138(137(4)), 1]), 130([138(137(4)), 2]), 130([138(137(4)), 3])]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 1 items follow
				0x81,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.Foo"
				// fields: [["a", OptionalType(AnyStructType)], ["b", OptionalType(OptionalType(AnyStructType))], ["c", OptionalType(OptionalType(OptionalType(AnyStructType)))]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// fields
				// array, 3 items follow
				0x83,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// field 2
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// c
				0x63,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 3 items follow
				0x83,
				// field 0
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// field 1
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 2
				0x02,
				// field 2
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 3
				0x03,
			},
		},
		{
			name: "struct with non-nil optional abstract fields",
			val: func() cadence.Value {
				structTypeWithOptionalAbstractField := newStructTypeWithOptionalAbstractField()
				simpleStructType := &cadence.StructType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "FooStruct",
					Fields: []cadence.Field{
						{
							Identifier: "bar",
							Type:       cadence.IntType{},
						},
					},
				}
				return cadence.NewStruct([]cadence.Value{
					cadence.NewOptional(cadence.NewInt(1)),
					cadence.NewOptional(cadence.NewOptional(cadence.NewInt(2))),
					cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(cadence.NewStruct([]cadence.Value{
						cadence.NewInt(3),
					}).WithType(simpleStructType)))),
				}).WithType(structTypeWithOptionalAbstractField)
			}(),
			expected: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"S.test.Foo","fields":[{"value":{"value":{"value":"1","type":"Int"},"type":"Optional"},"name":"a"},{"value":{"value":{"value":{"value":"2","type":"Int"},"type":"Optional"},"type":"Optional"},"name":"b"},{"value":{"value":{"value":{"value":{"value":{"id":"S.test.FooStruct","fields":[{"value":{"value":"3","type":"Int"},"name":"bar"}]},"type":"Struct"},"type":"Optional"},"type":"Optional"},"type":"Optional"},"name":"c"}]},"type":"Struct"}
				//
				// language=edn, format=ccf
				// 129([[160([h'', "S.test.Foo", [["a", 138(137(39))], ["b", 138(138(137(39)))], ["c", 138(138(138(137(39))))]]]), 160([h'01', "S.test.FooStruct", [["bar", 137(4)]]])], [136(h''), [130([137(4), 1]), 130([137(4), 2]), 130([136(h'01'), [3]])]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 2 items follow
				0x82,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.Foo"
				// fields: [["a", OptionalType(AnyStructType)], ["b", OptionalType(OptionalType(AnyStructType))], ["c", OptionalType(OptionalType(OptionalType(AnyStructType)))]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// fields
				// array, 3 items follow
				0x83,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// field 2
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// c
				0x63,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,

				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.FooStruct"
				// fields: [["bar", IntType]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// cadence-type-id
				// string, 16 bytes follow
				0x70,
				// S.test.FooStruct
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
				// fields
				// array, 1 items follow
				0x81,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 3 items follow
				0x83,
				// field 0
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// field 1
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 2
				0x02,
				// field 2
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// array, 1 items follow
				0x81,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 3
				0x03,
			},
		},
	}...)
}

func TestEncodeBool(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "True",
			val:  cadence.NewBool(true),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Bool","value":true}
				//
				// language=edn, format=ccf
				// 130([137(0), true])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Bool type ID (0)
				0x00,
				// true
				0xf5,
			},
		},
		{
			name: "False",
			val:  cadence.NewBool(false),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Bool","value":false}
				//
				// language=edn, format=ccf
				// 130([137(0), false])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Bool type ID (0)
				0x00,
				// false
				0xf4,
			},
		},
	}...)
}

func TestEncodeCharacter(t *testing.T) {

	t.Parallel()

	a, _ := cadence.NewCharacter("a")
	b, _ := cadence.NewCharacter("b")

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "a",
			val:  a,
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Character","value":"a"}
				//
				// language=edn, format=ccf
				// 130([137(2), "a"])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Character type ID (2)
				0x02,
				// UTF-8 string, 1 bytes follow
				0x61,
				// a
				0x61,
			},
		},
		{
			name: "b",
			val:  b,
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Character","value":"b"}
				//
				// language=edn, format=ccf
				// 130([137(2), "b"])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Character type ID (2)
				0x02,
				// UTF-8 string, 1 bytes follow
				0x61,
				// b
				0x62,
			},
		},
	}...)
}

func TestEncodeString(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Empty",
			val:  cadence.String(""),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"String","value":""}
				//
				// language=edn, format=ccf
				// 130([137(1), ""])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// UTF-8 string, 0 bytes follow
				0x60,
			},
		},
		{
			name: "Non-empty",
			val:  cadence.String("foo"),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"String","value":"foo"}
				//
				// language=edn, format=ccf
				// 130([137(1), "foo"])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// UTF-8 string, 3 bytes follow
				0x63,
				// f, o, o
				0x66, 0x6f, 0x6f,
			},
		},
	}...)
}

func TestEncodeAddress(t *testing.T) {

	t.Parallel()

	testEncodeAndDecode(
		t,
		cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
		[]byte{
			// language=json, format=json-cdc
			// {"type":"Address","value":"0x0000000102030405"}
			//
			// language=edn, format=ccf
			// 130([137(3), h'0000000102030405'])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Address type ID (3)
			0x03,
			// bytes, 8 bytes follow
			0x48,
			// 0, 0, 0, 1, 2, 3, 4, 5
			0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x5,
		},
	)
}

func TestEncodeInt(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Negative",
			val:  cadence.NewInt(-42),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int","value":"-42"}
				//
				// language=edn, format=ccf
				// 130([137(4), -42])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (negative big number)
				0xc3,
				// bytes, 1 byte follow
				0x41,
				// -42
				0x29,
			},
		},
		{
			name: "Zero",
			val:  cadence.NewInt(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(4), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (positive big number)
				0xc2,
				// bytes, 0 byte follow
				0x40,
			},
		},
		{
			name: "Positive",
			val:  cadence.NewInt(42),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int","value":"42"}
				//
				// language=edn, format=ccf
				// 130([137(4), 42])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (positive big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 42
				0x2a,
			},
		},
		{
			name: "SmallerThanMinInt256",
			val:  cadence.NewIntFromBig(new(big.Int).Sub(sema.Int256TypeMinIntBig, big.NewInt(10))),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819978"}
				//
				// language=edn, format=ccf
				// 130([137(4), -57896044618658097711785492504343953926634992332820282019728792003956564819978])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (negative big number)
				0xc3,
				// bytes, 32 bytes follow
				0x58, 0x20,
				// -57896044618658097711785492504343953926634992332820282019728792003956564819978
				0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09,
			},
		},
		{
			name: "LargerThanMaxUInt256",
			val:  cadence.NewIntFromBig(new(big.Int).Add(sema.UInt256TypeMaxIntBig, big.NewInt(10))),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int","value":"115792089237316195423570985008687907853269984665640564039457584007913129639945"}
				//
				// language=edn, format=ccf
				// 130([137(4), 115792089237316195423570985008687907853269984665640564039457584007913129639945])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// tag (positive big number)
				0xc2,
				// bytes, 33 bytes follow
				0x58, 0x21,
				// 115792089237316195423570985008687907853269984665640564039457584007913129639945
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x09,
			},
		},
	}...)
}

func TestEncodeInt8(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Min",
			val:  cadence.NewInt8(math.MinInt8),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int8","value":"-128"}
				//
				// language=edn, format=ccf
				// 130([137(5), -128])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int8 type ID (5)
				0x05,
				// -128
				0x38, 0x7f,
			},
		},
		{
			name: "Zero",
			val:  cadence.NewInt8(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int8","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(5), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int8 type ID (5)
				0x05,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewInt8(math.MaxInt8),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int8","value":"127"}
				//
				// language=edn, format=ccf
				// 130([137(5), 127])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int8 type ID (5)
				0x05,
				// 127
				0x18, 0x7f,
			},
		},
	}...)
}

func TestEncodeInt16(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Min",
			val:  cadence.NewInt16(math.MinInt16),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int16","value":"-32768"}
				//
				// language=edn, format=ccf
				// 130([137(6), -32768])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int16 type ID (6)
				0x06,
				// -32768
				0x39, 0x7F, 0xFF,
			},
		},
		{
			name: "Zero",
			val:  cadence.NewInt16(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int16","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(6), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int16 type ID (6)
				0x06,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewInt16(math.MaxInt16),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int16","value":"32767"}
				//
				// language=edn, format=ccf
				// 130([137(6), 32767])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int16 type ID (6)
				0x06,
				// 32767
				0x19, 0x7F, 0xFF,
			},
		},
	}...)
}

func TestEncodeInt32(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Min",
			val:  cadence.NewInt32(math.MinInt32),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int32","value":"-2147483648"}
				//
				// language=edn, format=ccf
				// 130([137(7), -2147483648])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int32 type ID (7)
				0x07,
				// -2147483648
				0x3a, 0x7f, 0xff, 0xff, 0xff,
			},
		},
		{
			name: "Zero",
			val:  cadence.NewInt32(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int32","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(7), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int32 type ID (7)
				0x07,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewInt32(math.MaxInt32),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int32","value":"2147483647"}
				//
				// language=edn, format=ccf
				// 130([137(7), 2147483647])
				//
				// language=cbor, format=ccf, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int32 type ID (7)
				0x07,
				// 2147483647
				0x1a, 0x7f, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeInt64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Min",
			val:  cadence.NewInt64(math.MinInt64),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int64","value":"-9223372036854775808"}
				//
				// language=edn, format=ccf
				// 130([137(8), -9223372036854775808])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int64 type ID (8)
				0x08,
				// -9223372036854775808
				0x3b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			name: "Zero",
			val:  cadence.NewInt64(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int64","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(8), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int64 type ID (8)
				0x08,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewInt64(math.MaxInt64),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int64","value":"9223372036854775807"}
				//
				// language=edn, format=ccf
				// 130([137(8), 9223372036854775807])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int64 type ID (8)
				0x08,
				// 9223372036854775807
				0x1b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeInt128(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Min",
			val:  cadence.Int128{Value: sema.Int128TypeMinIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int128","value":"-170141183460469231731687303715884105728"}
				//
				// language=edn, format=ccf
				// 130([137(9), -170141183460469231731687303715884105728])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int128 type ID (9)
				0x09,
				// tag big num
				0xc3,
				// bytes, 16 bytes follow
				0x50,
				// -170141183460469231731687303715884105728
				0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			name: "Zero",
			val:  cadence.NewInt128(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int128","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(9), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int128 type ID (9)
				0x09,
				// tag big num
				0xc2,
				// bytes, 0 bytes follow
				0x40,
			},
		},
		{
			name: "Max",
			val:  cadence.Int128{Value: sema.Int128TypeMaxIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int128","value":"170141183460469231731687303715884105727"}
				//
				// language=edn, format=ccf
				// 130([137(9), 170141183460469231731687303715884105727])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int128 type ID (9)
				0x09,
				// tag big num
				0xc2,
				// bytes, 16 bytes follow
				0x50,
				// 170141183460469231731687303715884105727
				0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeInt256(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Min",
			val:  cadence.Int256{Value: sema.Int256TypeMinIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int256","value":"-57896044618658097711785492504343953926634992332820282019728792003956564819968"}
				//
				// language=edn, format=ccf
				// 130([137(10), -57896044618658097711785492504343953926634992332820282019728792003956564819968])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int256 type ID (10)
				0x0a,
				// tag big num
				0xc3,
				// bytes, 32 bytes follow
				0x58, 0x20,
				// -57896044618658097711785492504343953926634992332820282019728792003956564819968
				0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			name: "Zero",
			val:  cadence.NewInt256(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int256","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(10), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int256 type ID (10)
				0x0a,
				// tag big num
				0xc2,
				// bytes, 0 bytes follow
				0x40,
			},
		},
		{
			name: "Max",
			val:  cadence.Int256{Value: sema.Int256TypeMaxIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Int256","value":"57896044618658097711785492504343953926634992332820282019728792003956564819967"}
				//
				// language=edn, format=ccf
				// 130([137(10), 57896044618658097711785492504343953926634992332820282019728792003956564819967])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int256 type ID (10)
				0x0a,
				// tag big num
				0xc2,
				// bytes, 32 bytes follow
				0x58, 0x20,
				// 57896044618658097711785492504343953926634992332820282019728792003956564819967
				0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeUInt(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewUInt(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(11), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt type ID (11)
				0x0b,
				// tag big num
				0xc2,
				// bytes, 0 bytes follow
				0x40,
			},
		},
		{
			name: "Positive",
			val:  cadence.NewUInt(42),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt","value":"42"}
				//
				// language=edn, format=ccf
				// 130([137(11), 42])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt type ID (11)
				0x0b,
				// tag big num
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 42
				0x2a,
			},
		},
		{
			name: "LargerThanMaxUInt256",
			val:  cadence.UInt{Value: new(big.Int).Add(sema.UInt256TypeMaxIntBig, big.NewInt(10))},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt","value":"115792089237316195423570985008687907853269984665640564039457584007913129639945"}
				//
				// language=edn, format=ccf
				// 130([137(11), 115792089237316195423570985008687907853269984665640564039457584007913129639945])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt type ID (11)
				0x0b,
				// tag big num
				0xc2,
				// bytes, 32 bytes follow
				0x58, 0x21,
				// 115792089237316195423570985008687907853269984665640564039457584007913129639945
				0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x09,
			},
		},
	}...)
}

func TestEncodeUInt8(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewUInt8(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt8","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(12), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt8 type ID (12)
				0x0c,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewUInt8(math.MaxUint8),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt8","value":"255"}
				//
				// language=edn, format=ccf
				// 130([137(12), 255])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt8 type ID (12)
				0x0c,
				// 255
				0x18, 0xff,
			},
		},
	}...)
}

func TestEncodeUInt16(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewUInt16(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt16","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(13), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt16 type ID (13)
				0x0d,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewUInt16(math.MaxUint16),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt16","value":"65535"}
				//
				// language=edn, format=ccf
				// 130([137(13), 65535])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt16 type ID (13)
				0x0d,
				// 65535
				0x19, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeUInt32(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewUInt32(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt32","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(14), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt32 type ID (14)
				0x0e,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewUInt32(math.MaxUint32),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt32","value":"4294967295"}
				//
				// language=edn, format=ccf
				// 130([137(14), 4294967295])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt32 type ID (14)
				0x0e,
				// 4294967295
				0x1a, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeUInt64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewUInt64(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt64","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(15), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt64 type ID (15)
				0x0f,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewUInt64(uint64(math.MaxUint64)),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt64","value":"18446744073709551615"}
				//
				// language=edn, format=ccf
				// 130([137(15), 18446744073709551615])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt64 type ID (15)
				0x0f,
				// 18446744073709551615
				0x1b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeUInt128(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewUInt128(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt128","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(16), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt128 type ID (16)
				0x10,
				// tag (big num)
				0xc2,
				// bytes, 0 bytes follow
				0x40,
			},
		},
		{
			name: "Max",
			val:  cadence.UInt128{Value: sema.UInt128TypeMaxIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt128","value":"340282366920938463463374607431768211455"}
				//
				// language=edn, format=ccf
				// 130([137(16), 340282366920938463463374607431768211455])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt128 type ID (16)
				0x10,
				// tag (big num)
				0xc2,
				// bytes, 16 bytes follow
				0x50,
				// 340282366920938463463374607431768211455
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeUInt256(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewUInt256(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt256","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(17), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt256 type ID (17)
				0x11,
				// tag (big num)
				0xc2,
				// bytes, 0 bytes follow
				0x40,
			},
		},
		{
			name: "Max",
			val:  cadence.UInt256{Value: sema.UInt256TypeMaxIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UInt256","value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}
				//
				// language=edn, format=ccf
				// 130([137(17), 115792089237316195423570985008687907853269984665640564039457584007913129639935])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt256 type ID (17)
				0x11,
				// tag (big num)
				0xc2,
				// bytes, 32 bytes follow
				0x58, 0x20,
				// 115792089237316195423570985008687907853269984665640564039457584007913129639935
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeWord8(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewWord8(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word8","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(18), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word8 type ID (18)
				0x12,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewWord8(math.MaxUint8),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word8","value":"255"}
				//
				// language=edn, format=ccf
				// 130([137(18), 255])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word8 type ID (18)
				0x12,
				// 255
				0x18, 0xff,
			},
		},
	}...)
}

func TestEncodeWord16(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewWord16(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word16","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(19), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word16 type ID (19)
				0x13,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewWord16(math.MaxUint16),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word16","value":"65535"}
				//
				// language=edn, format=ccf
				// 130([137(19), 65535])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word16 type ID (19)
				0x13,
				// 65535
				0x19, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeWord32(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewWord32(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word32","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(20), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word32 type ID (20)
				0x14,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewWord32(math.MaxUint32),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word32","value":"4294967295"}
				//
				// language=edn, format=ccf
				// 130([137(20), 4294967295])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word32 type ID (20)
				0x14,
				// 4294967295
				0x1a, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeWord64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewWord64(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word64","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(21), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word64 type ID (21)
				0x15,
				// 0
				0x00,
			},
		},
		{
			name: "Max",
			val:  cadence.NewWord64(math.MaxUint64),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word64","value":"18446744073709551615"}
				//
				// language=edn, format=ccf
				// 130([137(21), 18446744073709551615])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word64 type ID (21)
				0x15,
				// 18446744073709551615
				0x1b, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestEncodeWord128(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewWord128(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word128","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(52), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word128 type ID (52)
				0x18, 0x34,
				// tag (big num)
				0xc2,
				// bytes, 0 bytes follow
				0x40,
			},
		},
		{
			name: "Max",
			val:  cadence.Word128{Value: sema.Word128TypeMaxIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word128","value":"340282366920938463463374607431768211455"}
				//
				// language=edn, format=ccf
				// 130([137(52), 340282366920938463463374607431768211455])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word128 type ID (52)
				0x18, 0x34,
				// tag (big num)
				0xc2,
				// bytes, 16 bytes follow
				0x50,
				// 340282366920938463463374607431768211455
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestDecodeWord128Invalid(t *testing.T) {
	t.Parallel()

	decModes := []ccf.DecMode{ccf.EventsDecMode, deterministicDecMode}

	for _, dm := range decModes {
		_, err := dm.Decode(nil, []byte{
			// language=json, format=json-cdc
			// {"type":"Word128","value":"0"}
			//
			// language=edn, format=ccf
			// 130([137(52), 0])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Word128 type ID (52)
			0x18, 0x34,
			// Invalid type
			0xd7,
			// bytes, 16 bytes follow
			0x50,
			// 340282366920938463463374607431768211455
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		})
		require.Error(t, err)
		assert.Equal(t, "ccf: failed to decode: failed to decode Word128: cbor: cannot decode CBOR tag type to big.Int", err.Error())
	}
}

func TestEncodeWord256(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.NewWord256(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word256","value":"0"}
				//
				// language=edn, format=ccf
				// 130([137(53), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word256 type ID (53)
				0x18, 0x35,
				// tag (big num)
				0xc2,
				// bytes, 0 bytes follow
				0x40,
			},
		},
		{
			name: "Max",
			val:  cadence.Word256{Value: sema.Word256TypeMaxIntBig},
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Word256","value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}
				//
				// language=edn, format=ccf
				// 130([137(53), 115792089237316195423570985008687907853269984665640564039457584007913129639935])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Word256 type ID (53)
				0x18, 0x35,
				// tag (big num)
				0xc2,
				// bytes, 32 bytes follow
				0x58, 0x20,
				// 115792089237316195423570985008687907853269984665640564039457584007913129639935
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
	}...)
}

func TestDecodeWord256Invalid(t *testing.T) {
	t.Parallel()

	decModes := []ccf.DecMode{ccf.EventsDecMode, deterministicDecMode}

	for _, dm := range decModes {
		_, err := dm.Decode(nil, []byte{
			// language=json, format=json-cdc
			// {"type":"Word256","value":"115792089237316195423570985008687907853269984665640564039457584007913129639935"}
			//
			// language=edn, format=ccf
			// 130([137(53), 0])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Word256 type ID (53)
			0x18, 0x35,
			// Invalid type
			0xd7,
			// bytes, 32 bytes follow
			0x58, 0x20,
			// 115792089237316195423570985008687907853269984665640564039457584007913129639935
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		})
		require.Error(t, err)
		assert.Equal(t, "ccf: failed to decode: failed to decode Word256: cbor: cannot decode CBOR tag type to big.Int", err.Error())
	}
}

func TestEncodeFix64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.Fix64(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Fix64","value":"0.00000000"}
				//
				// language=edn, format=ccf
				// 130([137(22), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Fix64 type ID (22)
				0x16,
				// 0
				0x00,
			},
		},
		{
			name: "789.00123010",
			val:  cadence.Fix64(78_900_123_010),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Fix64","value":"789.00123010"}
				//
				// language=edn, format=ccf
				// 130([137(22), 78900123010])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Fix64 type ID (22)
				0x16,
				// 78900123010
				0x1b, 0x00, 0x00, 0x00, 0x12, 0x5e, 0xd0, 0x55, 0x82,
			},
		},
		{
			name: "1234.056",
			val:  cadence.Fix64(123_405_600_000),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Fix64","value":"1234.05600000"}
				//
				// language=edn, format=ccf
				// 130([137(22), 123405600000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Fix64 type ID (22)
				0x16,
				// 123405600000
				0x1b, 0x00, 0x00, 0x00, 0x1c, 0xbb, 0x8c, 0x05, 0x00,
			},
		},
		{
			name: "-12345.006789",
			val:  cadence.Fix64(-1_234_500_678_900),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Fix64","value":"-12345.00678900"}
				//
				// language=edn, format=ccf
				// 130([137(22), -1234500678900])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Fix64 type ID (22)
				0x16,
				// -1234500678900
				0x3b, 0x00, 0x00, 0x01, 0x1f, 0x6d, 0xf9, 0x74, 0xf3,
			},
		},
	}...)
}

func TestEncodeUFix64(t *testing.T) {

	t.Parallel()

	testAllEncodeAndDecode(t, []encodeTest{
		{
			name: "Zero",
			val:  cadence.UFix64(0),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UFix64","value":"0.00000000"}
				//
				// language=edn, format=ccf
				// 130([137(23), 0])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// 0
				0x00,
			},
		},
		{
			name: "789.00123010",
			val:  cadence.UFix64(78_900_123_010),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UFix64","value":"789.00123010"}
				//
				// language=edn, format=ccf
				// 130([137(23), 78900123010])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// 78900123010
				0x1b, 0x00, 0x00, 0x00, 0x12, 0x5e, 0xd0, 0x55, 0x82,
			},
		},
		{
			name: "1234.056",
			val:  cadence.UFix64(123_405_600_000),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"UFix64","value":"1234.05600000"}
				//
				// language=edn, format=ccf
				// 130([137(23), 123405600000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// 123405600000
				0x1b, 0x00, 0x00, 0x00, 0x1c, 0xbb, 0x8c, 0x05, 0x00,
			},
		},
	}...)
}

func TestEncodeArray(t *testing.T) {

	t.Parallel()

	// []
	emptyArray := encodeTest{
		name: "Empty",
		val: cadence.NewArray(
			[]cadence.Value{},
		).WithType(cadence.NewVariableSizedArrayType(cadence.NewIntType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Array","value":[]}
			//
			// language=edn, format=ccf
			// 130([139(137(4)), []])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type []int
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array data without inlined type
			// array, 0 items follow
			0x80,
		},
	}

	// constant sized array [1, 2, 3]
	constantSizedIntArray := encodeTest{
		name: "Constant-sized Integers",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewInt(1),
			cadence.NewInt(2),
			cadence.NewInt(3),
		}).WithType(cadence.NewConstantSizedArrayType(3, cadence.NewIntType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Array","value":[{"type":"Int","value":"1"},{"type":"Int","value":"2"},{"type":"Int","value":"3"}]}
			//
			// language=edn, format=ccf
			// 130([140[3, (137(4))], [1, 2, 3]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type constant-sized [3]int
			// tag
			0xd8, ccf.CBORTagConstsizedArrayType,
			// array, 2 items follow
			0x82,
			// number of elements
			0x03,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array data without inlined type definition
			// array, 3 items follow
			0x83,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
		},
	}

	// [1, 2, 3]
	intArray := encodeTest{
		name: "Integers",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewInt(1),
			cadence.NewInt(2),
			cadence.NewInt(3),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.NewIntType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Array","value":[{"type":"Int","value":"1"},{"type":"Int","value":"2"},{"type":"Int","value":"3"}]}
			//
			// language=edn, format=ccf
			// 130([139(137(4)), [1, 2, 3]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type []int
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array data without inlined type definition
			// array, 3 items follow
			0x83,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
		},
	}

	// [[1], [2], [3]]
	nestedArray := encodeTest{
		name: "Nested",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(1),
			}).WithType(cadence.NewVariableSizedArrayType(cadence.NewIntType())),
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(2),
			}).WithType(cadence.NewVariableSizedArrayType(cadence.NewIntType())),
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(3),
			}).WithType(cadence.NewVariableSizedArrayType(cadence.NewIntType())),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.NewVariableSizedArrayType(cadence.NewIntType()))),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":[{"value":"1","type":"Int"}],"type":"Array"},{"value":[{"value":"2","type":"Int"}],"type":"Array"},{"value":[{"value":"3","type":"Int"}],"type":"Array"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(139(137(4))), [[1], [2], [3]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type [[]int]
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array data without inlined type definition
			// array, 3 items follow
			0x83,
			// array, 1 item follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 1 item follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// array, 1 item follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
		},
	}

	// [S.test.Foo{1}, S.test.Foo{2}, S.test.Foo{3}]
	resourceArray := encodeTest{
		name: "Resources",
		val: func() cadence.Value {
			fooResourceType := newFooResourceType()
			return cadence.NewArray([]cadence.Value{
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(fooResourceType),
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(2),
				}).WithType(fooResourceType),
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(3),
				}).WithType(fooResourceType),
			}).WithType(cadence.NewVariableSizedArrayType(fooResourceType))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Array","value":[{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"1"}}]}},{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"2"}}]}},{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"3"}}]}}]}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)]]])], [139(136(h'')), [[1], [2], [3]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["bar", int type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// [S.test.Foo{1}, S.test.Foo{2}, S.test.Foo{3}]
			// array, 3 items follow
			0x83,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
		},
	}

	s, err := cadence.NewString("a")
	require.NoError(t, err)

	resourceWithAbstractFieldArray := encodeTest{
		name: "Resources with abstract field",
		val: func() cadence.Value {
			foooResourceTypeWithAbstractField := newFoooResourceTypeWithAbstractField()
			return cadence.NewArray([]cadence.Value{
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(1),
					cadence.NewInt(1), // field is AnyStruct type
				}).WithType(foooResourceTypeWithAbstractField),
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(2),
					s, // field is AnyStruct type
				}).WithType(foooResourceTypeWithAbstractField),
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(3),
					cadence.NewBool(true), // field is AnyStruct type
				}).WithType(foooResourceTypeWithAbstractField),
			}).WithType(cadence.NewVariableSizedArrayType(foooResourceTypeWithAbstractField))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Array","value":[{"type":"Resource","value":{"id":"S.test.Fooo","fields":[{"name":"bar","value":{"type":"Int","value":"1"}},{"name":"baz","value":{"type":"Int","value":"1"}}]}},{"type":"Resource","value":{"id":"S.test.Fooo","fields":[{"name":"bar","value":{"type":"Int","value":"2"}},{"name":"baz","value":{"type":"String","value":"a"}}]}},{"type":"Resource","value":{"id":"S.test.Fooo","fields":[{"name":"bar","value":{"type":"Int","value":"3"}},{"name":"baz","value":{"type":"Bool","value":true}}]}}]}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Fooo", [["bar", 137(4)], ["baz", 137(39)]]])], [139(136(h'')), [[1, 130([137(4), 1])], [2, 130([137(1), "a"])], [3, 130([137(0), true])]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Fooo"
			// fields: [["bar", int type], ["baz", any type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 11 bytes follow
			0x6b,
			// S.test.Fooo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x6f,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// baz
			0x62, 0x61, 0x7a,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// [S.test.Foo{1, 1}, S.test.Foo{2, "a"}, S.test.Foo{3, true}]
			// array, 3 items follow
			0x83,
			// element 0
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// element 1
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// text string, 1 byte
			0x61,
			// "a"
			0x61,
			// element 2
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Bool type ID (0)
			0x00,
			// true
			0xf5,
		},
	}

	// [1, "a", true]
	heterogeneousSimpleTypeArray := encodeTest{
		name: "Heterogenous AnyStruct Array with Simple Values",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewInt(1),
			s,
			cadence.NewBool(true),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.NewAnyStructType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Array","value":[{"type":"Int","value":"1"},{"type":"String","value":"a"},{"type":"Bool","value":true}]}
			//
			// language=edn, format=ccf
			// 130([139(137(39)), [130([137(4), 1]), 130([137(1), "a"]), 130([137(0), true])]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type ([]AnyStruct)
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array data with inlined type because static array element type is abstract (AnyStruct)
			// array, 3 items follow
			0x83,
			// element 0 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// element 1 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// text string, length 1
			0x61,
			// "a"
			0x61,
			// element 2 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Bool type ID (0)
			0x00,
			// true
			0xf5,
		},
	}

	// [Int8(1), Int16(2), Int32(3)]
	heterogeneousNumberTypeArray := encodeTest{
		name: "Heterogeous Number Array",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewInt8(1),
			cadence.NewInt16(2),
			cadence.NewInt32(3),
		}).WithType(cadence.NewVariableSizedArrayType(cadence.NewNumberType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":"1","type":"Int8"},{"value":"2","type":"Int16"},{"value":"3","type":"Int32"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(137(43)), [130([137(5), 1]), 130([137(6), 2]), 130([137(7), 3])]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type ([]Integer)
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Number type ID (43)
			0x18, 0x2b,
			// array data with inlined type because static array element type is abstract (Number)
			// array, 3 items follow
			0x83,
			// element 0 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int8 type ID (5)
			0x05,
			// 1
			0x01,
			// element 1 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int16 type ID (6)
			0x06,
			// 2
			0x02,
			// element 2 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int32 type ID (7)
			0x07,
			// 3
			0x03,
		},
	}

	// [1, S.test.Foo{1}]
	heterogeneousCompositeTypeArray := encodeTest{
		name: "Heterogenous AnyStruct Array with Composite Value",
		val: func() cadence.Value {
			fooResourceType := newFooResourceType()
			return cadence.NewArray([]cadence.Value{
				cadence.NewInt(1),
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(fooResourceType),
			}).WithType(cadence.NewVariableSizedArrayType(cadence.NewAnyStructType()))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":"1","type":"Int"},{"value":{"id":"S.test.Foo","fields":[{"value":{"value":"1","type":"Int"},"name":"bar"}]},"type":"Resource"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)]]])], [139(137(39)), [130([137(4), 1]), 130([136(h''), [1]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definition
			// array, 1 items follow
			0x81,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["bar", int type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// type ([]AnyStruct)
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array data with inlined type because static array element type is abstract (AnyStruct)
			// array, 2 items follow
			0x82,
			// element 0 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// element 1 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// S.test.Foo{1}
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
		},
	}

	// [S.test.Foo{1}, S.test.Fooo{2, "a"}]
	resourceInterfaceTypeArray := encodeTest{
		name: "Resource Interface Array",
		val: func() cadence.Value {
			resourceInterfaceType := &cadence.ResourceInterfaceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "Bar",
			}
			fooResourceType := newFooResourceType()
			foooResourceTypeWithAbstractField := newFoooResourceTypeWithAbstractField()
			return cadence.NewArray([]cadence.Value{
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(fooResourceType),
				cadence.NewResource([]cadence.Value{
					cadence.NewInt(2),
					s,
				}).WithType(foooResourceTypeWithAbstractField),
			}).WithType(cadence.NewVariableSizedArrayType(resourceInterfaceType))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.Foo","fields":[{"value":{"value":"1","type":"Int"},"name":"bar"}]},"type":"Resource"},{"value":{"id":"S.test.Fooo","fields":[{"value":{"value":"2","type":"Int"},"name":"bar"},{"value":{"value":"a","type":"String"},"name":"baz"}]},"type":"Resource"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[177([h'', "S.test.Bar"]), 161([h'01', "S.test.Foo", [["bar", 137(4)]]]), 161([h'02', "S.test.Fooo", [["bar", 137(4)], ["baz", 137(39)]]])], [139(136(h'')), [130([136(h'01'), [1]]), 130([136(h'02'), [2, 130([137(1), "a"])]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 3 items follow
			0x83,
			// type definition 0
			// resource interface type:
			// id: []byte{}
			// cadence-type-id: "S.test.Bar"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Boo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x42, 0x61, 0x72,
			// type definition 1
			// resource type:
			// id: []byte{1}
			// cadence-type-id: "S.test.Foo"
			// fields: [["bar", int type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// type definition 2:
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Fooo"
			// fields: [["bar", int type], ["baz", any type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// cadence-type-id
			// string, 11 bytes follow
			0x6b,
			// S.test.Fooo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x6f,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// baz
			0x62, 0x61, 0x7a,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 2 item follow
			0x82,
			// element 0 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// S.test.Foo{1}
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// element 1 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// S.test.Fooo{2, "a"}
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// text string, 1 byte
			0x61,
			// "a"
			0x61,
		},
	}

	// [S.test.FooStruct{"a", S.test.Foo{0}}, S.test.FooStruct{"b", S.test.Foo{1}}]
	resourceStructArray := encodeTest{
		name: "Resource Struct Array",
		val: func() cadence.Value {
			fooResourceType := newFooResourceType()
			resourceStructType := newResourceStructType()
			return cadence.NewArray([]cadence.Value{
				cadence.NewStruct([]cadence.Value{
					cadence.String("a"),
					cadence.NewResource([]cadence.Value{
						cadence.NewInt(0),
					}).WithType(fooResourceType),
				}).WithType(resourceStructType),
				cadence.NewStruct([]cadence.Value{
					cadence.String("b"),
					cadence.NewResource([]cadence.Value{
						cadence.NewInt(1),
					}).WithType(fooResourceType),
				}).WithType(resourceStructType),
			}).WithType(cadence.NewVariableSizedArrayType(resourceStructType))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.FooStruct","fields":[{"value":{"value":"a","type":"String"},"name":"a"},{"value":{"value":{"id":"S.test.Foo","fields":[{"value":{"value":"0","type":"Int"},"name":"bar"}]},"type":"Resource"},"name":"b"}]},"type":"Struct"},{"value":{"id":"S.test.FooStruct","fields":[{"value":{"value":"b","type":"String"},"name":"a"},{"value":{"value":{"id":"S.test.Foo","fields":[{"value":{"value":"1","type":"Int"},"name":"bar"}]},"type":"Resource"},"name":"b"}]},"type":"Struct"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)]]]), 160([h'01', "S.test.FooStruct", [["a", 137(1)], ["b", 136(h'')]]])], [139(136(h'01')), [["a", [0]], ["b", [1]]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,
			// type definition 0
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["bar", int type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// type definition 1
			// struct type:
			// id: []byte{1}
			// cadence-type-id: "S.test.FooStruct"
			// fields: [["a", string type], ["b", foo resource type]]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 16 bytes follow
			0x70,
			// S.test.FooStruct
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// [S.test.FooStruct{"a", S.test.Foo{0}}, S.test.FooStruct{"b", S.test.Foo{1}}]
			// array, 2 item follow
			0x82,
			// element 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 0 bytes follow
			0x40,
			// element 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
		},
	}

	// [S.test.FooStruct{1}, S.test.FooStruct{2}]
	structInterfaceTypeArray := encodeTest{
		name: "Struct Interface Array",
		val: func() cadence.Value {
			structInterfaceType := &cadence.StructInterfaceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooStructInterface",
			}
			structType := &cadence.StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooStruct",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.IntType{},
					},
				},
			}
			return cadence.NewArray([]cadence.Value{
				cadence.NewStruct([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(structType),
				cadence.NewStruct([]cadence.Value{
					cadence.NewInt(2),
				}).WithType(structType),
			}).WithType(cadence.NewVariableSizedArrayType(structInterfaceType))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.FooStruct","fields":[{"value":{"value":"1","type":"Int"},"name":"a"}]},"type":"Struct"},{"value":{"id":"S.test.FooStruct","fields":[{"value":{"value":"2","type":"Int"},"name":"a"}]},"type":"Struct"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[160([h'', "S.test.FooStruct", [["a", 137(4)]]]), 176([h'01', "S.test.FooStructInterface"])], [139(136(h'01')), [130([136(h''), [1]]), 130([136(h''), [2]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,
			// type definition 0
			// struct type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooStruct"
			// fields: [["a", int type]]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 16 bytes follow
			0x70,
			// S.test.FooStruct
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// type definition 1
			// struct interface type:
			// id: []byte{1}
			// cadence-type-id: "S.test.FooStructInterface"
			// tag
			0xd8, ccf.CBORTagStructInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 25 bytes follow
			0x78, 0x19,
			// S.test.FooStructInterface
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 2 item follow
			0x82,
			// element 0 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// element 1 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
		},
	}

	// [S.test.FooContract{1}, S.test.FooContract{2}]
	contractInterfaceTypeArray := encodeTest{
		name: "Contract Interface Array",
		val: func() cadence.Value {
			contractInterfaceType := &cadence.ContractInterfaceType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooContractInterface",
			}
			contractType := &cadence.ContractType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooContract",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.IntType{},
					},
				},
			}
			return cadence.NewArray([]cadence.Value{
				cadence.NewContract([]cadence.Value{
					cadence.NewInt(1),
				}).WithType(contractType),
				cadence.NewContract([]cadence.Value{
					cadence.NewInt(2),
				}).WithType(contractType),
			}).WithType(cadence.NewVariableSizedArrayType(contractInterfaceType))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.FooContract","fields":[{"value":{"value":"1","type":"Int"},"name":"a"}]},"type":"Contract"},{"value":{"id":"S.test.FooContract","fields":[{"value":{"value":"2","type":"Int"},"name":"a"}]},"type":"Contract"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[163([h'', "S.test.FooContract", [["a", 137(4)]]]), 178([h'01', "S.test.FooContractInterface"])], [139(136(h'01')), [130([136(h''), [1]]), 130([136(h''), [2]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,
			// type definition 0
			// contract type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooContract"
			// fields: [["a", int type]]
			// tag
			0xd8, ccf.CBORTagContractType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 12 bytes follow
			0x72,
			// S.test.FooContract
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// type definition 1
			// constract interface type:
			// id: []byte{1}
			// cadence-type-id: "S.test.FooContractInterface"
			// tag
			0xd8, ccf.CBORTagContractInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 27 bytes follow
			0x78, 0x1b,
			// S.test.FooContractInterface
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,
			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 2 item follow
			0x82,
			// element 0 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// element 1 (inline type and value)
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 1 items follow
			0x81,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
		},
	}

	testAllEncodeAndDecode(t,
		emptyArray,
		constantSizedIntArray,
		intArray,
		nestedArray,
		resourceStructArray,
		resourceArray,
		resourceWithAbstractFieldArray,
		heterogeneousSimpleTypeArray,
		heterogeneousNumberTypeArray,
		heterogeneousCompositeTypeArray,
		resourceInterfaceTypeArray,
		structInterfaceTypeArray,
		contractInterfaceTypeArray,
	)
}

func TestEncodeDictionary(t *testing.T) {

	t.Parallel()

	// {}
	emptyDict := encodeTest{
		name: "empty",
		val: cadence.NewDictionary(
			[]cadence.KeyValuePair{},
		).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[],"type":"Dictionary"}
			//
			// language=edn, format=ccf
			// 130([141([137(1), 137(4)]), []])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type (map[string]int)
			// tag
			0xd8, ccf.CBORTagDictType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array, 6 items follow
			0x80,
		},
	}

	// {"c":3, "b":2, "a":1}
	simpleDict := encodeTest{
		name: "Simple",
		val: cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("c"),
				Value: cadence.NewInt(3),
			},
			{
				Key:   cadence.String("b"),
				Value: cadence.NewInt(2),
			},
			{
				Key:   cadence.String("a"),
				Value: cadence.NewInt(1),
			},
		}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
		expectedVal: cadence.NewDictionary([]cadence.KeyValuePair{
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
		}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Int","value":"1"}},{"key":{"type":"String","value":"b"},"value":{"type":"Int","value":"2"}},{"key":{"type":"String","value":"c"},"value":{"type":"Int","value":"3"}}]}
			//
			// language=edn, format=ccf
			// 130([141([137(1), 137(4)]), ["a", 1, "b", 2, "c", 3]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type (map[string]int)
			// tag
			0xd8, ccf.CBORTagDictType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array data without inlined type definition
			// array, 6 items follow
			0x86,
			// string, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// string, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// string, 1 bytes follow
			0x61,
			// c
			0x63,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
		},
	}

	// {"c":{"3:3"}, "b":{"2":2}, "a":{"1":1}}
	nestedDict := encodeTest{
		name: "Nested",
		val: cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.String("c"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("3"),
						Value: cadence.NewInt(3),
					},
				}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
			},
			{
				Key: cadence.String("b"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("2"),
						Value: cadence.NewInt(2),
					},
				}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
			},
			{
				Key: cadence.String("a"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("1"),
						Value: cadence.NewInt(1),
					},
				}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
			},
		}).WithType(cadence.NewDictionaryType(
			cadence.NewStringType(),
			cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
		),
		expectedVal: cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.String("a"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("1"),
						Value: cadence.NewInt(1),
					},
				}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
			},
			{
				Key: cadence.String("b"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("2"),
						Value: cadence.NewInt(2),
					},
				}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
			},
			{
				Key: cadence.String("c"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("3"),
						Value: cadence.NewInt(3),
					},
				}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
			},
		}).WithType(cadence.NewDictionaryType(
			cadence.NewStringType(),
			cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType())),
		),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Dictionary","value":[{"key":{"type":"String","value":"1"},"value":{"type":"Int","value":"1"}}]}},{"key":{"type":"String","value":"b"},"value":{"type":"Dictionary","value":[{"key":{"type":"String","value":"2"},"value":{"type":"Int","value":"2"}}]}},{"key":{"type":"String","value":"c"},"value":{"type":"Dictionary","value":[{"key":{"type":"String","value":"3"},"value":{"type":"Int","value":"3"}}]}}]}
			//
			// language=edn, format=ccf
			// 130([141([137(1), 141([137(1), 137(4)])]), ["a", ["1", 1], "b", ["2", 2], "c", ["3", 3]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type (map[string]map[string, int])
			// tag
			0xd8, ccf.CBORTagDictType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// tag
			0xd8, ccf.CBORTagDictType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array data without inlined type definition
			// array, 6 items follow
			0x86,
			// string, 1 bytes follow
			0x61,
			// a
			0x61,
			// nested dictionary
			// array, 2 items follow
			0x82,
			// string, 1 bytes follow
			0x61,
			// "1"
			0x31,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// string, 1 bytes follow
			0x61,
			// b
			0x62,
			// nested dictionary
			// array, 2 items follow
			0x82,
			// string, 1 bytes follow
			0x61,
			// "2"
			0x32,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// string, 1 bytes follow
			0x61,
			// c
			0x63,
			// nested dictionary
			// array, 2 items follow
			0x82,
			// string, 1 bytes follow
			0x61,
			// "3"
			0x33,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
		},
	}

	// {"c":foo{3}, "b":foo{2}, "a":foo{1}}
	resourceDict := func() encodeTest {
		fooResourceType := newFooResourceType()

		return encodeTest{
			name: "Resources",
			val: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key: cadence.String("c"),
					Value: cadence.NewResource([]cadence.Value{
						cadence.NewInt(3),
					}).WithType(fooResourceType),
				},
				{
					Key: cadence.String("b"),
					Value: cadence.NewResource([]cadence.Value{
						cadence.NewInt(2),
					}).WithType(fooResourceType),
				},
				{
					Key: cadence.String("a"),
					Value: cadence.NewResource([]cadence.Value{
						cadence.NewInt(1),
					}).WithType(fooResourceType),
				},
			}).WithType(cadence.NewDictionaryType(
				cadence.NewStringType(),
				fooResourceType,
			)),
			expectedVal: cadence.NewDictionary([]cadence.KeyValuePair{
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
			}).WithType(cadence.NewDictionaryType(
				cadence.NewStringType(),
				fooResourceType,
			)),
			expected: []byte{
				// language=json, format=json-cdc
				// {"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"1"}}]}}},{"key":{"type":"String","value":"b"},"value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"2"}}]}}},{"key":{"type":"String","value":"c"},"value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"3"}}]}}}]}
				//
				// language=edn, format=ccf
				// 129([[161([h'', "S.test.Foo", [["bar", 137(4)]]])], [141([137(1), 136(h'')]), ["a", [1], "b", [2], "c", [3]]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definition
				// array, 1 items follow
				0x81,
				// resource type:
				// id: []byte{}
				// cadence-type-id: "S.test.Foo"
				// fields: [["bar", int type]]
				// tag
				0xd8, ccf.CBORTagResourceType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// fields
				// array, 1 items follow
				0x81,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagDictType,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 6 items follow
				0x86,
				// string, 1 bytes follow
				0x61,
				// a
				0x61,
				// array, 1 items follow
				0x81,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// string, 1 bytes follow
				0x61,
				// b
				0x62,
				// array, 1 items follow
				0x81,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 2
				0x02,
				// string, 1 bytes follow
				0x61,
				// c
				0x63,
				// array, 1 items follow
				0x81,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 3
				0x03,
			},
		}
	}()

	// {"c":3, 0:1, true:3}
	heterogeneousSimpleTypeDict := encodeTest{
		name: "heterogeneous simple type",
		val: cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("c"),
				Value: cadence.NewInt(2),
			},
			{
				Key:   cadence.NewInt(0),
				Value: cadence.NewInt(1),
			},
			{
				Key:   cadence.NewBool(true),
				Value: cadence.NewInt(3),
			},
		}).WithType(cadence.NewDictionaryType(cadence.NewAnyStructType(), cadence.NewAnyStructType())),
		expectedVal: cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.NewBool(true),
				Value: cadence.NewInt(3),
			},
			{
				Key:   cadence.String("c"),
				Value: cadence.NewInt(2),
			},
			{
				Key:   cadence.NewInt(0),
				Value: cadence.NewInt(1),
			},
		}).WithType(cadence.NewDictionaryType(cadence.NewAnyStructType(), cadence.NewAnyStructType())),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"key":{"value":"c","type":"String"},"value":{"value":"2","type":"Int"}},{"key":{"value":"0","type":"Int"},"value":{"value":"1","type":"Int"}},{"key":{"value":true,"type":"Bool"},"value":{"value":"3","type":"Int"}}],"type":"Dictionary"}
			//
			// language=edn, format=ccf
			// 130([141([137(39), 137(39)]), [130([137(0), true]), 130([137(4), 3]), 130([137(1), "c"]), 130([137(4), 2]), 130([137(4), 0]), 130([137(4), 1])]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type (map[AnyStruct]AnyStruct)
			// tag
			0xd8, ccf.CBORTagDictType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array data without inlined type definition
			// array, 6 items follow
			0x86,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Bool type ID (0)
			0x00,
			// true
			0xf5,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// text, 1 byte follows
			0x61,
			// c
			0x63,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// tag (big num)
			0xc2,
			// bytes, 0 bytes follow
			0x40,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
		},
	}

	testAllEncodeAndDecode(t,
		emptyDict,
		simpleDict,
		nestedDict,
		resourceDict,
		heterogeneousSimpleTypeDict,
		//heterogeneousNumberTypeDict,
		//heterogeneousCompositeTypeDict,
	)
}

func TestEncodeSortedDictionary(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name         string
		val          cadence.Value
		expectedVal  cadence.Value
		expectedCBOR []byte
	}

	dict := cadence.NewDictionary([]cadence.KeyValuePair{
		{
			Key:   cadence.String("c"),
			Value: cadence.NewInt(3),
		},
		{
			Key:   cadence.String("a"),
			Value: cadence.NewInt(1),
		},
		{
			Key:   cadence.String("b"),
			Value: cadence.NewInt(2),
		},
	}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType()))

	expectedDict := cadence.NewDictionary([]cadence.KeyValuePair{
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
	}).WithType(cadence.NewDictionaryType(cadence.NewStringType(), cadence.NewIntType()))

	simpleDict := testCase{
		"Simple",
		dict,
		expectedDict,
		[]byte{
			// language=json, format=json-cdc
			// {"type":"Dictionary","value":[{"key":{"type":"String","value":"a"},"value":{"type":"Int","value":"1"}},{"key":{"type":"String","value":"b"},"value":{"type":"Int","value":"2"}},{"key":{"type":"String","value":"c"},"value":{"type":"Int","value":"3"}}]}
			//
			// language=edn, format=ccf
			// 130([141([137(1), 137(4)]), ["a", 1, "b", 2, "c", 3]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type (map[string]int)
			// tag
			0xd8, ccf.CBORTagDictType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// array data without inlined type definition
			// array, 6 items follow
			0x86,
			// string, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// string, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// string, 1 bytes follow
			0x61,
			// c
			0x63,
			// tag (big num)
			0xc2,
			// bytes, 1 bytes follow
			0x41,
			// 3
			0x03,
		},
	}

	testCases := []testCase{
		simpleDict,
	}

	test := func(tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actualCBOR := testEncode(t, tc.val, tc.expectedCBOR)
			testDecode(t, actualCBOR, tc.expectedVal)
		})
	}

	for _, tc := range testCases {
		test(tc)
	}
}

func exportFromScript(t *testing.T, code string) cadence.Value {
	checker, err := checker.ParseAndCheck(t, code)
	require.NoError(t, err)

	var uuid uint64

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

		// expectedVal is different from actual because "bar" field is
		// encoded before "uuid" field due to deterministic encoding.
		expectedVal := cadence.NewResource([]cadence.Value{
			cadence.NewInt(42),
			cadence.NewUInt64(1),
		}).WithType(cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Foo",
			[]cadence.Field{
				{Type: cadence.NewIntType(), Identifier: "bar"},
				{Type: cadence.NewUInt64Type(), Identifier: "uuid"},
			},
			nil,
		))

		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"uuid","value":{"type":"UInt64","value":"1"}},{"name":"bar","value":{"type":"Int","value":"42"}}]}}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)], ["uuid", 137(15)]]])], [136(h''), [42, 1]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// 2 fields: [["bar", type(int)], ["uuid", type(uint64)]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 4 bytes follow
			0x64,
			// uuid
			0x75, 0x75, 0x69, 0x64,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Uint type ID (15)
			0x0f,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 42
			0x2a,
			// 1
			0x01,
		}

		testEncodeAndDecodeEx(t, actual, expectedCBOR, expectedVal)
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

		// expectedVal is different from actual because "bar" field is
		// encoded before "uuid" field due to deterministic encoding.
		expectedVal := cadence.NewResource([]cadence.Value{
			cadence.NewInt(42),
			cadence.NewUInt64(1),
		}).WithType(cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Foo",
			[]cadence.Field{
				{Type: cadence.NewIntType(), Identifier: "bar"},
				{Type: cadence.NewUInt64Type(), Identifier: "uuid"},
			},
			nil,
		))

		// function "foo" should be omitted from resulting CBOR
		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"uuid","value":{"type":"UInt64","value":"1"}},{"name":"bar","value":{"type":"Int","value":"42"}}]}}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)], ["uuid", 137(15)]]])], [136(h''), [42, 1]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// 2 fields: [["bar", type(int)], ["uuid", type(uint64)]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 4 bytes follow
			0x64,
			// uuid
			0x75, 0x75, 0x69, 0x64,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Uint type ID (15)
			0x0f,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 42
			0x2a,
			// 1
			0x01,
		}

		testEncodeAndDecodeEx(t, actual, expectedCBOR, expectedVal)
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

					destroy() {
						destroy self.bar
					}
				}

				fun main(): @Foo {
					return <- create Foo(bar: <- create Bar(x: 42))
				}
			`)

		// S.test.Foo(uuid: 2, bar: S.test.Bar(uuid: 1, x: 42)) (cadence.Resource)

		// expectedVal is different from actual because "bar" field is
		// encoded before "uuid" field due to deterministic encoding.
		expectedBarResourceType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Bar",
			[]cadence.Field{
				{Type: cadence.NewIntType(), Identifier: "x"},
				{Type: cadence.NewUInt64Type(), Identifier: "uuid"},
			},
			nil,
		)
		expectedBarResource := cadence.NewResource(
			[]cadence.Value{
				cadence.NewInt(42),
				cadence.NewUInt64(1),
			},
		).WithType(expectedBarResourceType)

		expectedVal := cadence.NewResource(
			[]cadence.Value{
				expectedBarResource,
				cadence.NewUInt64(2),
			}).WithType(cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Foo",
			[]cadence.Field{
				{Type: expectedBarResourceType, Identifier: "bar"},
				{Type: cadence.NewUInt64Type(), Identifier: "uuid"},
			},
			nil,
		))

		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"uuid","value":{"type":"UInt64","value":"2"}},{"name":"bar","value":{"type":"Resource","value":{"id":"S.test.Bar","fields":[{"name":"uuid","value":{"type":"UInt64","value":"1"}},{"name":"x","value":{"type":"Int","value":"42"}}]}}}]}}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Bar", [["x", 137(4)], ["uuid", 137(15)]]]), 161([h'01', "S.test.Foo", [["bar", 136(h'')], ["uuid", 137(15)]]])], [136(h'01'), [[42, 1], 2]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,

			// resource type:
			// id: []byte{01}
			// cadence-type-id: "S.test.Bar"
			// 2 fields: [["x", type(int)], ["uuid", type(uint64)], ]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Bar
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x42, 0x61, 0x72,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x61,
			// x
			0x78,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 4 bytes follow
			0x64,
			// uuid
			0x75, 0x75, 0x69, 0x64,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Uint64 type ID (15)
			0x0f,

			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// 2 fields: [["bar", type ref(1)], ["uuid", type(uint64)]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// type definition ID (0)
			// bytes, 0 bytes follow
			0x40,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 4 bytes follow
			0x64,
			// uuid
			0x75, 0x75, 0x69, 0x64,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Uint64 type ID (15)
			0x0f,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 2 items follow
			0x82,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 42
			0x2a,
			// 1
			0x01,
			// 2
			0x02,
		}

		testEncodeAndDecodeEx(t, actual, expectedCBOR, expectedVal)
	})
}

func TestEncodeStruct(t *testing.T) {

	t.Parallel()

	noFieldStruct := encodeTest{
		name: "no field",
		val: func() cadence.Value {
			noFieldStructType := &cadence.StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooStruct",
				Fields:              []cadence.Field{},
			}
			return cadence.NewStruct(
				[]cadence.Value{},
			).WithType(noFieldStructType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":{"id":"S.test.FooStruct","fields":[]},"type":"Struct"}
			//
			// language=edn, format=ccf
			// 129([[160([h'', "S.test.FooStruct", []])], [136(h''), []]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// struct type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooStruct"
			// 0 fields: []
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 16 bytes follow
			0x70,
			// S.test.FooStruct
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// array, 0 item follows
			0x80,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 0 items follow
			0x80,
		},
	}

	simpleStruct := encodeTest{
		name: "Simple",
		val: func() cadence.Value {
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
			return cadence.NewStruct(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.String("foo"),
				},
			).WithType(simpleStructType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Struct","value":{"id":"S.test.FooStruct","fields":[{"name":"a","value":{"type":"Int","value":"1"}},{"name":"b","value":{"type":"String","value":"foo"}}]}}
			//
			// language=edn, format=ccf
			// 129([[160([h'', "S.test.FooStruct", [["a", 137(4)], ["b", 137(1)]]])], [136(h''), [1, "foo"]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// struct type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooStruct"
			// 2 fields: [["a", type(int)], ["b", type(string)]]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 16 bytes follow
			0x70,
			// S.test.FooStruct
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 1
			0x01,
			// String, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
		},
	}

	resourceStruct := encodeTest{
		name: "Resources",
		val: func() cadence.Value {
			fooResourceType := newFooResourceType()
			resourceStructType := newResourceStructType()
			return cadence.NewStruct(
				[]cadence.Value{
					cadence.String("foo"),
					cadence.NewResource(
						[]cadence.Value{
							cadence.NewInt(42),
						},
					).WithType(fooResourceType),
				},
			).WithType(resourceStructType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Struct","value":{"id":"S.test.FooStruct","fields":[{"name":"a","value":{"type":"String","value":"foo"}},{"name":"b","value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"42"}}]}}}]}}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)]]]), 160([h'01', "S.test.FooStruct", [["a", 137(1)], ["b", 136(h'')]]])], [136(h'01'), ["foo", [42]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,

			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["bar", int type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,

			// struct type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooStruct"
			// 2 fields: [["a", type(int)], ["b", type(string)]]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 16 bytes follow
			0x70,
			// S.test.FooStruct
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// type reference ID (1)
			// bytes, 0 bytes follow
			0x40,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 2 items follow
			0x82,
			// String, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
			// array, 1 items follow
			0x81,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 42
			0x2a,
		},
	}

	testAllEncodeAndDecode(t,
		noFieldStruct,
		simpleStruct,
		resourceStruct,
	)
}

func TestEncodeEvent(t *testing.T) {

	t.Parallel()

	simpleEvent := encodeTest{
		name: "Simple",
		val: func() cadence.Value {
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
			return cadence.NewEvent(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.String("foo"),
				},
			).WithType(simpleEventType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Event","value":{"id":"S.test.FooEvent","fields":[{"name":"a","value":{"type":"Int","value":"1"}},{"name":"b","value":{"type":"String","value":"foo"}}]}}
			//
			// language=edn, format=ccf
			// 129([[162([h'', "S.test.FooEvent", [["a", 137(4)], ["b", 137(1)]]])], [136(h''), [1, "foo"]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// event type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooEvent"
			// 2 fields: [["a", type(int)], ["b", type(string)]]
			// tag
			0xd8, ccf.CBORTagEventType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 15 bytes follow
			0x6f,
			// S.test.FooEvent
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x45, 0x76, 0x65, 0x6e, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 1
			0x01,
			// String, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
		},
	}

	abstractEvent := encodeTest{
		name: "abstract event",
		val: func() cadence.Value {
			abstractEventType := &cadence.EventType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooEvent",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.IntType{},
					},
					{
						Identifier: "b",
						Type:       cadence.AnyStructType{},
					},
				},
			}
			simpleStructType := &cadence.StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooStruct",
				Fields: []cadence.Field{
					{
						Identifier: "c",
						Type:       cadence.StringType{},
					},
				},
			}
			return cadence.NewEvent(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.NewStruct([]cadence.Value{
						cadence.String("b"),
					}).WithType(simpleStructType),
				},
			).WithType(abstractEventType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":{"id":"S.test.FooEvent","fields":[{"value":{"value":"1","type":"Int"},"name":"a"},{"value":{"value":{"id":"S.test.FooStruct","fields":[{"value":{"value":"b","type":"String"},"name":"c"}]},"type":"Struct"},"name":"b"}]},"type":"Event"}
			//
			// language=edn, format=ccf
			// 129([[162([h'', "S.test.FooEvent", [["a", 137(4)], ["b", 137(39)]]]), 160([h'01', "S.test.FooStruct", [["c", 137(1)]]])], [136(h''), [1, 130([136(h'01'), ["b"]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,
			// event type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooEvent"
			// 2 fields: [["a", type(int)], ["b", type(anystruct)]]
			// tag
			0xd8, ccf.CBORTagEventType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 15 bytes follow
			0x6f,
			// S.test.FooEvent
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x45, 0x76, 0x65, 0x6e, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// struct type:
			// id: []byte{0x01}
			// cadence-type-id: "S.test.FooStruct"
			// 1 fields: [["c", type(string)]]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 16 bytes follow
			0x70,
			// S.test.FooStruct
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// c
			0x63,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 1
			0x01,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 1 items follow
			0x81,
			// string, 1 byte follows
			0x61,
			// "b"
			0x62,
		},
	}

	resourceEvent := encodeTest{
		name: "Resources",
		val: func() cadence.Value {
			fooResourceType := newFooResourceType()
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
			return cadence.NewEvent(
				[]cadence.Value{
					cadence.String("foo"),
					cadence.NewResource(
						[]cadence.Value{
							cadence.NewInt(42),
						},
					).WithType(fooResourceType),
				},
			).WithType(resourceEventType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Event","value":{"id":"S.test.FooEvent","fields":[{"name":"a","value":{"type":"String","value":"foo"}},{"name":"b","value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"42"}}]}}}]}}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)]]]), 162([h'01', "S.test.FooEvent", [["a", 137(1)], ["b", 136(h'')]]])], [136(h'01'), ["foo", [42]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,

			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["bar", int type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,

			// event type:
			// id: []byte{0x01}
			// cadence-type-id: "S.test.FooEvent"
			// 2 fields: [["a", type(int)], ["b", type(string)]]
			// tag
			0xd8, ccf.CBORTagEventType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 15 bytes follow
			0x6f,
			// S.test.FooEvent
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x45, 0x76, 0x65, 0x6e, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 2 items follow
			0x82,
			// String, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
			// array, 1 items follow
			0x81,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 42
			0x2a,
		},
	}

	testCases := []encodeTest{simpleEvent, resourceEvent, abstractEvent}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualCBOR, err := ccf.EventsEncMode.Encode(tc.val)
			require.NoError(t, err)
			utils.AssertEqualWithDiff(t, tc.expected, actualCBOR)

			decodedVal, err := ccf.EventsDecMode.Decode(nil, actualCBOR)
			require.NoError(t, err)
			assert.Equal(
				t,
				cadence.ValueWithCachedTypeID(tc.val),
				cadence.ValueWithCachedTypeID(decodedVal),
			)
		})
	}
}

func TestEncodeContract(t *testing.T) {

	t.Parallel()

	simpleContract := encodeTest{
		name: "Simple",
		val: func() cadence.Value {
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
			return cadence.NewContract(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.String("foo"),
				},
			).WithType(simpleContractType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Contract","value":{"id":"S.test.FooContract","fields":[{"name":"a","value":{"type":"Int","value":"1"}},{"name":"b","value":{"type":"String","value":"foo"}}]}}
			//
			// language=edn, format=ccf
			// 129([[163([h'', "S.test.FooContract", [["a", 137(4)], ["b", 137(1)]]])], [136(h''), [1, "foo"]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// contract type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooContract"
			// 2 fields: [["a", type(int)], ["b", type(string)]]
			// tag
			0xd8, ccf.CBORTagContractType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 18 bytes follow
			0x72,
			// S.test.FooContract
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 1
			0x01,
			// String, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
		},
	}

	abstractContract := encodeTest{
		name: "abstract contract",
		val: func() cadence.Value {
			simpleStructType := &cadence.StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooStruct",
				Fields: []cadence.Field{
					{
						Identifier: "c",
						Type:       cadence.StringType{},
					},
				},
			}
			abstractContractType := &cadence.ContractType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooContract",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.IntType{},
					},
					{
						Identifier: "b",
						Type:       cadence.AnyStructType{},
					},
				},
			}
			return cadence.NewContract(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.NewStruct([]cadence.Value{
						cadence.String("b"),
					}).WithType(simpleStructType),
				},
			).WithType(abstractContractType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":{"id":"S.test.FooContract","fields":[{"value":{"value":"1","type":"Int"},"name":"a"},{"value":{"value":{"id":"S.test.FooStruct","fields":[{"value":{"value":"b","type":"String"},"name":"c"}]},"type":"Struct"},"name":"b"}]},"type":"Contract"}
			//
			// language=edn, format=ccf
			// 129([[160([h'', "S.test.FooStruct", [["c", 137(1)]]]), 163([h'01', "S.test.FooContract", [["a", 137(4)], ["b", 137(39)]]])], [136(h'01'), [1, 130([136(h''), ["b"]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,
			// struct type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooStruct"
			// 1 fields: [["c", type(string)]]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 16 bytes follow
			0x70,
			// S.test.FooStruct
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// c
			0x63,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// contract type:
			// id: []byte{0x01}
			// cadence-type-id: "S.test.FooContract"
			// 2 fields: [["a", type(int)], ["b", type(string)]]
			// tag
			0xd8, ccf.CBORTagContractType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 18 bytes follow
			0x72,
			// S.test.FooContract
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 2 items follow
			0x82,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 1
			0x01,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follow
			0x40,
			// array, 1 item follows
			0x81,
			// String, 1 bytes follow
			0x61,
			// "b"
			0x62,
		},
	}

	resourceContract := encodeTest{
		name: "Resources",
		val: func() cadence.Value {
			fooResourceType := newFooResourceType()
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
			return cadence.NewContract(
				[]cadence.Value{
					cadence.String("foo"),
					cadence.NewResource(
						[]cadence.Value{
							cadence.NewInt(42),
						},
					).WithType(fooResourceType),
				},
			).WithType(resourceContractType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"type":"Contract","value":{"id":"S.test.FooContract","fields":[{"name":"a","value":{"type":"String","value":"foo"}},{"name":"b","value":{"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"bar","value":{"type":"Int","value":"42"}}]}}}]}}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["bar", 137(4)]]]), 163([h'01', "S.test.FooContract", [["a", 137(1)], ["b", 136(h'')]]])], [136(h'01'), ["foo", [42]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 2 items follow
			0x82,

			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["bar", int type]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,

			// contract type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooContract"
			// 2 fields: [["a", type(int)], ["b", type(string)]]
			// tag
			0xd8, ccf.CBORTagContractType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 18 bytes follow
			0x72,
			// S.test.FooContract
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x61, 0x63, 0x74,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// b
			0x62,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// array, 2 items follow
			0x82,
			// String, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
			// array, 1 items follow
			0x81,
			// tag (big number)
			0xc2,
			// bytes, 1 byte follow
			0x41,
			// 42
			0x2a,
		},
	}

	testAllEncodeAndDecode(t, simpleContract, abstractContract, resourceContract)
}

func TestEncodeEnum(t *testing.T) {
	t.Parallel()

	simpleEnum := encodeTest{
		name: "Simple",
		val: func() cadence.Value {
			simpleEnumType := &cadence.EnumType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "FooEnum",
				Fields: []cadence.Field{
					{
						Identifier: "raw",
						Type:       cadence.UInt8Type{},
					},
				},
			}
			return cadence.NewEnum(
				[]cadence.Value{
					cadence.NewUInt8(1),
				},
			).WithType(simpleEnumType)
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":{"id":"S.test.FooEnum","fields":[{"value":{"value":"1","type":"UInt8"},"name":"raw"}]},"type":"Enum"}
			//
			// language=edn, format=ccf
			// 129([[164([h'', "S.test.FooEnum", [["raw", 137(12)]]])], [136(h''), [1]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// enum type:
			// id: []byte{}
			// cadence-type-id: "S.test.FooEnum"
			// 1 fields: [["raw", type(uint8)]]
			// tag
			0xd8, ccf.CBORTagEnumType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 14 bytes follow
			0x6e,
			// S.test.FooEnum
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x45, 0x6e, 0x75, 0x6d,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// raw
			0x72, 0x61, 0x77,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// UInt8 type ID (12)
			0x0c,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 1 items follow
			0x81,
			// 1
			0x01,
		},
	}

	testAllEncodeAndDecode(t, simpleEnum)
}

func TestEncodeValueOfRestrictedType(t *testing.T) {

	t.Parallel()

	t.Run("nil restricted type", func(t *testing.T) {
		hasCountInterfaceType := cadence.NewResourceInterfaceType(
			common.NewStringLocation(nil, "test"),
			"HasCount",
			nil,
			nil,
		)

		hasSumInterfaceType := cadence.NewResourceInterfaceType(
			common.NewStringLocation(nil, "test"),
			"HasSum",
			nil,
			nil,
		)

		statsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("count", cadence.NewIntType()),
				cadence.NewField("sum", cadence.NewIntType()),
			},
			nil,
		)

		countSumRestrictedType := cadence.NewRestrictedType(
			nil,
			[]cadence.Type{
				hasCountInterfaceType,
				hasSumInterfaceType,
			},
		)

		val := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.NewInt(2),
				},
			).WithType(statsType),
		}).WithType(cadence.NewVariableSizedArrayType(countSumRestrictedType))

		expectedStatsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("sum", cadence.NewIntType()),
				cadence.NewField("count", cadence.NewIntType()),
			},
			nil,
		)

		expectedCountSumRestrictedType := cadence.NewRestrictedType(
			nil,
			[]cadence.Type{
				hasSumInterfaceType,
				hasCountInterfaceType,
			},
		)

		expectedVal := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(2),
					cadence.NewInt(1),
				},
			).WithType(expectedStatsType),
		}).WithType(cadence.NewVariableSizedArrayType(expectedCountSumRestrictedType))

		testEncodeAndDecodeEx(
			t,
			val,
			[]byte{
				// language=json, format=json-cdc
				// {"value":[{"value":{"id":"S.test.Stats","fields":[{"value":{"value":"1","type":"Int"},"name":"count"},{"value":{"value":"2","type":"Int"},"name":"sum"}]},"type":"Resource"}],"type":"Array"}
				//
				// language=edn, format=ccf
				// 129([[161([h'', "S.test.Stats", [["sum", 137(4)], ["count", 137(4)]]]), 177([h'01', "S.test.HasSum"]), 177([h'02', "S.test.HasCount"])], [139(143([null, [136(h'01'), 136(h'02')]])), [130([136(h''), [2, 1]])]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 3 items follow
				0x83,
				// resource type:
				// id: []byte{}
				// cadence-type-id: "S.test.Stats"
				// 2 fields: [["sum", type(int)], ["count", type(int)]]
				// tag
				0xd8, ccf.CBORTagResourceType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 12 bytes follow
				0x6c,
				// S.test.Stats
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x73,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 3 bytes follow
				0x63,
				// sum
				0x73, 0x75, 0x6d,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 5 bytes follow
				0x65,
				// count
				0x63, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// resource interface type:
				// id: []byte{1}
				// cadence-type-id: "S.test.HasSum"
				// tag
				0xd8, ccf.CBORTagResourceInterfaceType,
				// array, 2 items follow
				0x82,
				// id
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// cadence-type-id
				// string, 13 bytes follow
				0x6d,
				// S.test.HasSum
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x53, 0x75, 0x6d,
				// resource interface type:
				// id: []byte{2}
				// cadence-type-id: "S.test.HasCount"
				// tag
				0xd8, ccf.CBORTagResourceInterfaceType,
				// array, 2 items follow
				0x82,
				// id
				// bytes, 1 bytes follow
				0x41,
				// 2
				0x02,
				// cadence-type-id
				// string, 15 bytes follow
				0x6f,
				// S.test.HasCount
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagRestrictedType,
				// array, 2 items follow
				0x82,
				// type
				// null
				0xf6,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 1 byte follows
				0x41,
				// 1
				0x01,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 1 byte follows
				0x41,
				// 2
				0x02,

				// array, 1 item follows
				0x81,
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 byte follows
				0x40,
				// array, 2 items follow
				0x82,
				// tag (big num)
				0xc2,
				// bytes, 1 byte follows
				0x41,
				// 2
				0x02,
				// tag (big num)
				0xc2,
				// bytes, 1 byte follows
				0x41,
				// 1
				0x01,
			},
			expectedVal,
		)
	})

	t.Run("resource restricted type", func(t *testing.T) {
		t.Parallel()

		hasCountInterfaceType := cadence.NewResourceInterfaceType(
			common.NewStringLocation(nil, "test"),
			"HasCount",
			nil,
			nil,
		)

		hasSumInterfaceType := cadence.NewResourceInterfaceType(
			common.NewStringLocation(nil, "test"),
			"HasSum",
			nil,
			nil,
		)

		statsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("count", cadence.NewIntType()),
				cadence.NewField("sum", cadence.NewIntType()),
			},
			nil,
		)

		countSumRestrictedType := cadence.NewRestrictedType(
			statsType,
			[]cadence.Type{
				hasCountInterfaceType,
				hasSumInterfaceType,
			},
		)

		val := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.NewInt(2),
				},
			).WithType(statsType),
		}).WithType(cadence.NewVariableSizedArrayType(countSumRestrictedType))

		expectedStatsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("sum", cadence.NewIntType()),
				cadence.NewField("count", cadence.NewIntType()),
			},
			nil,
		)

		expectedCountSumRestrictedType := cadence.NewRestrictedType(
			expectedStatsType,
			[]cadence.Type{
				hasSumInterfaceType,
				hasCountInterfaceType,
			},
		)

		expectedVal := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(2),
					cadence.NewInt(1),
				},
			).WithType(expectedStatsType),
		}).WithType(cadence.NewVariableSizedArrayType(expectedCountSumRestrictedType))

		testEncodeAndDecodeEx(
			t,
			val,
			[]byte{
				// language=json, format=json-cdc
				// {"value":[{"value":{"id":"S.test.Stats","fields":[{"value":{"value":"1","type":"Int"},"name":"sum"},{"value":{"value":"2","type":"Int"},"name":"count"}]},"type":"Resource"}],"type":"Array"}
				//
				// language=edn, format=ccf
				// 129([[161([h'', "S.test.Stats", [["sum", 137(4)], ["count", 137(4)]]]), 177([h'01', "S.test.HasSum"]), 177([h'02', "S.test.HasCount"])], [139(143([136(h''), [136(h'01'), 136(h'02')]])), [130([136(h''), [2, 1]])]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 3 items follow
				0x83,
				// resource type:
				// id: []byte{}
				// cadence-type-id: "S.test.Stats"
				// 2 fields: [["sum", type(int)], ["count", type(int)]]
				// tag
				0xd8, ccf.CBORTagResourceType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 12 bytes follow
				0x6c,
				// S.test.Stats
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x73,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 3 bytes follow
				0x63,
				// sum
				0x73, 0x75, 0x6d,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 5 bytes follow
				0x65,
				// count
				0x63, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// resource interface type:
				// id: []byte{1}
				// cadence-type-id: "S.test.HasSum"
				// tag
				0xd8, ccf.CBORTagResourceInterfaceType,
				// array, 2 items follow
				0x82,
				// id
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// cadence-type-id
				// string, 13 bytes follow
				0x6d,
				// S.test.HasSum
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x53, 0x75, 0x6d,
				// resource interface type:
				// id: []byte{2}
				// cadence-type-id: "S.test.HasCount"
				// tag
				0xd8, ccf.CBORTagResourceInterfaceType,
				// array, 2 items follow
				0x82,
				// id
				// bytes, 1 bytes follow
				0x41,
				// 2
				0x02,
				// cadence-type-id
				// string, 15 bytes follow
				0x6f,
				// S.test.HasCount
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagRestrictedType,
				// array, 2 items follow
				0x82,
				// type
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 byte follows
				0x40,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 1 byte follows
				0x41,
				// 1
				0x01,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 1 byte follows
				0x41,
				// 2
				0x02,

				// array, 1 item follows
				0x81,
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 byte follows
				0x40,
				// array, 2 items follow
				0x82,
				// tag (big num)
				0xc2,
				// bytes, 1 byte follows
				0x41,
				// 2
				0x02,
				// tag (big num)
				0xc2,
				// bytes, 1 byte follows
				0x41,
				// 1
				0x01,
			},
			expectedVal,
		)
	})
}

func TestEncodeValueOfReferenceType(t *testing.T) {
	t.Parallel()

	// Factory instead of values to avoid data races,
	// as tests may run in parallel

	newSimpleStructType := func() *cadence.StructType {
		return &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.StringType{},
				},
			},
		}
	}

	// ["a", "b"] with static type []&String
	referenceToSimpleType := encodeTest{
		name: "array of reference to string",
		val: cadence.NewArray([]cadence.Value{
			cadence.String("a"),
			cadence.String("b"),
		}).WithType(cadence.NewVariableSizedArrayType(
			cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.NewStringType()),
		)),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":"a","type":"String"},{"value":"b","type":"String"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(142([false, 137(1)])), ["a", "b"]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type []&String
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// array data without inlined type
			// array, 2 items follow
			0x82,
			// text, 1 byte follow
			0x61,
			// "a"
			0x61,
			// text, 1 byte follow
			0x61,
			// "b"
			0x62,
		},
	}

	// ["a", nil] with static type []&String?
	referenceToOptionalSimpleType := encodeTest{
		name: "array of reference to optional string",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewOptional(cadence.String("a")),
			cadence.NewOptional(nil),
		}).WithType(cadence.NewVariableSizedArrayType(
			cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.NewOptionalType(cadence.NewStringType())),
		)),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"value":"a","type":"String"},"type":"Optional"},{"value":null,"type":"Optional"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(142([false, 138(137(1))])), ["a", null]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type []&String?
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// array data without inlined type
			// array, 2 items follow
			0x82,
			// text, 1 byte follow
			0x61,
			// "a"
			0x61,
			// nil
			0xf6,
		},
	}

	// {"one": 7456}
	optionalReferenceToSimpleType := encodeTest{
		name: "dictionary of optional reference to Int",
		val: func() cadence.Value {
			dictionaryType := &cadence.DictionaryType{
				KeyType: cadence.TheStringType,
				ElementType: &cadence.OptionalType{
					Type: &cadence.ReferenceType{
						Type:          cadence.TheInt128Type,
						Authorization: cadence.UnauthorizedAccess,
					},
				},
			}

			// dictionary is generated by fuzzer.
			return cadence.Dictionary{
				DictionaryType: dictionaryType,
				Pairs: []cadence.KeyValuePair{
					{
						Key: cadence.String("one"),
						Value: cadence.Optional{
							Value: cadence.Int128{
								Value: big.NewInt(7456),
							},
						},
					},
				},
			}
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"key":{"value":"one","type":"String"},"value":{"value":{"value":"7456","type":"Int128"},"type":"Optional"}}],"type":"Dictionary"}
			//
			// language=edn, format=ccf
			// 130([141([137(1), 138(142([false, 137(9)]))]), ["one", 7456]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type {string: &Int128?}
			// tag
			0xd8, ccf.CBORTagDictType,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// string type ID (1)
			0x01,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int128 type ID (9)
			0x09,
			// array data without inlined type
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// "one"
			0x6f, 0x6e, 0x65,
			// tag (big num)
			0xc2,
			// bytes, 2 bytes follow
			0x42,
			// 7456
			0x1d, 0x20,
		},
	}

	// ["a", 1] with static type []&AnyStruct
	referenceToAnyStructWithSimpleTypes := encodeTest{
		name: "array of reference to any",
		val: cadence.NewArray([]cadence.Value{
			cadence.String("a"),
			cadence.NewUInt8(1),
		}).WithType(cadence.NewVariableSizedArrayType(
			cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.NewAnyStructType()),
		)),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":"a","type":"String"},{"value":"1","type":"UInt8"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(142([false, 137(39)])), [130([137(1), "a"]), 130([137(12), 1])]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// type []&AnyStruct
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array data without inlined type
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// string type ID (1)
			0x01,
			// text, 1 byte follow
			0x61,
			// "a"
			0x61,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// UInt8 type ID (12)
			0x0c,
			// 1
			0x01,
		},
	}

	// [FooStruct(1), FooStruct(2)] with static type []&FooStruct
	referenceToStructType := encodeTest{
		name: "array of reference to struct",
		val: func() cadence.Value {
			simpleStructType := newSimpleStructType()
			return cadence.NewArray([]cadence.Value{
				cadence.NewStruct([]cadence.Value{
					cadence.String("a"),
				}).WithType(simpleStructType),
				cadence.NewStruct([]cadence.Value{
					cadence.String("b"),
				}).WithType(simpleStructType),
			}).WithType(cadence.NewVariableSizedArrayType(
				cadence.NewReferenceType(cadence.UnauthorizedAccess, simpleStructType),
			))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.Foo","fields":[{"value":{"value":"a","type":"String"},"name":"a"}]},"type":"Struct"},{"value":{"id":"S.test.Foo","fields":[{"value":{"value":"b","type":"String"},"name":"a"}]},"type":"Struct"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[160([h'', "S.test.Foo", [["a", 137(1)]]])], [139(142([false, 136(h'')])), [["a"], ["b"]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// struct type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["a", StringType]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,

			// array, 2 items follow
			0x82,
			// type []&S.test.Foo
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follows
			0x40,
			// array data without inlined type
			// array, 2 items follow
			0x82,
			// array, 1 items follow
			0x81,
			// text, 1 byte follow
			0x61,
			// "a"
			0x61,
			// array, 1 items follow
			0x81,
			// text, 1 byte follow
			0x61,
			// "b"
			0x62,
		},
	}

	// ["a", FooStruct("b")] with static type []&AnyStruct
	referenceToAnyStructWithStructType := encodeTest{
		name: "array of reference to any with struct",
		val: func() cadence.Value {
			simpleStructType := newSimpleStructType()
			return cadence.NewArray([]cadence.Value{
				cadence.String("a"),
				cadence.NewStruct([]cadence.Value{
					cadence.String("b"),
				}).WithType(simpleStructType),
			}).WithType(cadence.NewVariableSizedArrayType(
				cadence.NewReferenceType(cadence.UnauthorizedAccess, cadence.NewAnyStructType()),
			))
		}(),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":"a","type":"String"},{"value":{"id":"S.test.Foo","fields":[{"value":{"value":"1","type":"Int"},"name":"a"}]},"type":"Struct"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[160([h'', "S.test.Foo", [["a", 137(1)]]])], [139(142([false, 137(39)])), [130([137(1), "a"]), 130([136(h''), ["b"]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// struct type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// fields: [["a", StringType]
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 1 bytes follow
			0x61,
			// a
			0x61,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,

			// array, 2 items follow
			0x82,
			// type []&AnyStruct
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,

			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// string type ID (1)
			0x01,
			// text, 1 byte follow
			0x61,
			// "a"
			0x61,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follows
			0x40,
			// array, 1 item follows
			0x81,
			// text, 1 byte follow
			0x61,
			// "b"
			0x62,
		},
	}

	// ["a", "b", nil] with type array of reference to optional AnyStruct
	referenceToOptionalAnyStructType := encodeTest{
		name: "array of reference to optional AnyStruct",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewOptional(cadence.String("a")),
			cadence.NewOptional(cadence.NewOptional(cadence.String("b"))),
			cadence.NewOptional(nil),
		}).WithType(cadence.NewVariableSizedArrayType(
			cadence.NewReferenceType(
				cadence.UnauthorizedAccess,
				cadence.NewOptionalType(cadence.NewAnyStructType()),
			))),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"value":"a","type":"String"},"type":"Optional"},{"value":{"value":{"value":"b","type":"String"},"type":"Optional"},"type":"Optional"},{"value":null,"type":"Optional"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(142([false, 138(137(39))])), [130([137(1), "a"]), 130([138(137(1)), "b"]), null]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array data
			// array, 3 items follow
			0x83,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// string, 1 byte follows
			0x61,
			// "a"
			0x61,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// string, 1 byte follows
			0x61,
			// "b"
			0x62,
			// null
			0xf6,
		},
	}

	optionalReferenceToAnyStructType := encodeTest{
		name: "array of optional reference to AnyStruct",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewOptional(cadence.String("a")),
			cadence.NewOptional(cadence.NewOptional(cadence.String("b"))),
			cadence.NewOptional(nil),
		}).WithType(cadence.NewVariableSizedArrayType(
			cadence.NewOptionalType(
				cadence.NewReferenceType(
					cadence.UnauthorizedAccess,
					cadence.NewAnyStructType(),
				)))),
		expected: []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"value":"a","type":"String"},"type":"Optional"},{"value":{"value":{"value":"b","type":"String"},"type":"Optional"},"type":"Optional"},{"value":null,"type":"Optional"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(138(142([false, 137(39)]))), [130([137(1), "a"]), 130([138(137(1)), "b"]), null]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array data
			// array, 3 items follow
			0x83,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// string, 1 byte follows
			0x61,
			// "a"
			0x61,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// string, 1 byte follows
			0x61,
			// "b"
			0x62,
			// null
			0xf6,
		},
	}

	optionalReferenceToOptionalAnyStructType := encodeTest{
		name: "array of optional reference to optional AnyStruct",
		val: cadence.NewArray([]cadence.Value{
			cadence.NewOptional(cadence.NewOptional(cadence.String("a"))),
			cadence.NewOptional(cadence.NewOptional(cadence.NewOptional(cadence.String("b")))),
			cadence.NewOptional(nil),
		}).WithType(cadence.NewVariableSizedArrayType(
			cadence.NewOptionalType(
				cadence.NewReferenceType(
					cadence.UnauthorizedAccess,
					cadence.NewOptionalType(
						cadence.NewAnyStructType(),
					))))),
		expected: []byte{
			// language=json, format=json-cdc
			//  {"value":[{"value":{"value":{"value":"a","type":"String"},"type":"Optional"},"type":"Optional"},{"value":{"value":{"value":{"value":"b","type":"String"},"type":"Optional"},"type":"Optional"},"type":"Optional"},{"value":null,"type":"Optional"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 130([139(138(142([false, 138(137(39))]))), [130([137(1), "a"]), 130([138(137(1)), "b"]), null]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagReferenceType,
			// array, 2 items follow
			0x82,
			// nil
			0xf6,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array data
			// array, 3 items follow
			0x83,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// string, 1 byte follows
			0x61,
			// "a"
			0x61,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// String type ID (1)
			0x01,
			// string, 1 byte follows
			0x61,
			// "b"
			0x62,
			// null
			0xf6,
		},
	}

	testAllEncodeAndDecode(t,
		referenceToSimpleType,
		referenceToOptionalSimpleType,
		referenceToStructType,
		referenceToAnyStructWithSimpleTypes,
		referenceToAnyStructWithStructType,
		referenceToOptionalAnyStructType,
		optionalReferenceToSimpleType,
		optionalReferenceToAnyStructType,
		optionalReferenceToOptionalAnyStructType,
	)
}

func TestEncodeSimpleTypes(t *testing.T) {

	t.Parallel()

	type simpleTypes struct {
		typ              cadence.Type
		cborSimpleTypeID int
	}

	var tests []encodeTest

	for _, ty := range []simpleTypes{
		{cadence.AnyType{}, ccf.TypeAny},
		{cadence.AnyResourceType{}, ccf.TypeAnyResource},
		{cadence.AnyStructAttachmentType{}, ccf.TypeAnyStructAttachmentType},
		{cadence.AnyResourceAttachmentType{}, ccf.TypeAnyResourceAttachmentType},
		{cadence.MetaType{}, ccf.TypeMetaType},
		{cadence.VoidType{}, ccf.TypeVoid},
		{cadence.NeverType{}, ccf.TypeNever},
		{cadence.BoolType{}, ccf.TypeBool},
		{cadence.StringType{}, ccf.TypeString},
		{cadence.CharacterType{}, ccf.TypeCharacter},
		{cadence.BytesType{}, ccf.TypeBytes},
		{cadence.AddressType{}, ccf.TypeAddress},
		{cadence.SignedNumberType{}, ccf.TypeSignedNumber},
		{cadence.IntegerType{}, ccf.TypeInteger},
		{cadence.SignedIntegerType{}, ccf.TypeSignedInteger},
		{cadence.FixedPointType{}, ccf.TypeFixedPoint},
		{cadence.SignedFixedPointType{}, ccf.TypeSignedFixedPoint},
		{cadence.IntType{}, ccf.TypeInt},
		{cadence.Int8Type{}, ccf.TypeInt8},
		{cadence.Int16Type{}, ccf.TypeInt16},
		{cadence.Int32Type{}, ccf.TypeInt32},
		{cadence.Int64Type{}, ccf.TypeInt64},
		{cadence.Int128Type{}, ccf.TypeInt128},
		{cadence.Int256Type{}, ccf.TypeInt256},
		{cadence.UIntType{}, ccf.TypeUInt},
		{cadence.UInt8Type{}, ccf.TypeUInt8},
		{cadence.UInt16Type{}, ccf.TypeUInt16},
		{cadence.UInt32Type{}, ccf.TypeUInt32},
		{cadence.UInt64Type{}, ccf.TypeUInt64},
		{cadence.UInt128Type{}, ccf.TypeUInt128},
		{cadence.UInt256Type{}, ccf.TypeUInt256},
		{cadence.Word8Type{}, ccf.TypeWord8},
		{cadence.Word16Type{}, ccf.TypeWord16},
		{cadence.Word32Type{}, ccf.TypeWord32},
		{cadence.Word64Type{}, ccf.TypeWord64},
		{cadence.Word128Type{}, ccf.TypeWord128},
		{cadence.Word256Type{}, ccf.TypeWord256},
		{cadence.Fix64Type{}, ccf.TypeFix64},
		{cadence.UFix64Type{}, ccf.TypeUFix64},
		{cadence.BlockType{}, ccf.TypeBlock},
		{cadence.PathType{}, ccf.TypePath},
		{cadence.CapabilityPathType{}, ccf.TypeCapabilityPath},
		{cadence.StoragePathType{}, ccf.TypeStoragePath},
		{cadence.PublicPathType{}, ccf.TypePublicPath},
		{cadence.PrivatePathType{}, ccf.TypePrivatePath},
		{cadence.AccountKeyType{}, ccf.TypeAccountKey},
		{cadence.AuthAccountContractsType{}, ccf.TypeAuthAccountContracts},
		{cadence.AuthAccountKeysType{}, ccf.TypeAuthAccountKeys},
		{cadence.AuthAccountType{}, ccf.TypeAuthAccount},
		{cadence.PublicAccountContractsType{}, ccf.TypePublicAccountContracts},
		{cadence.PublicAccountKeysType{}, ccf.TypePublicAccountKeys},
		{cadence.PublicAccountType{}, ccf.TypePublicAccount},
		{cadence.DeployedContractType{}, ccf.TypeDeployedContract},
	} {
		var w bytes.Buffer

		cborEncMode := func() cbor.EncMode {
			options := cbor.CoreDetEncOptions()
			options.BigIntConvert = cbor.BigIntConvertNone
			encMode, err := options.EncMode()
			if err != nil {
				panic(err)
			}
			return encMode
		}()

		encoder := cborEncMode.NewStreamEncoder(&w)

		err := encoder.EncodeRawBytes([]byte{
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 elements follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Meta type ID (41)
			0x18, 0x29,
			// tag
			0xd8, ccf.CBORTagSimpleTypeValue,
		})
		require.NoError(t, err)

		err = encoder.EncodeInt(ty.cborSimpleTypeID)
		require.NoError(t, err)

		encoder.Flush()

		tests = append(tests, encodeTest{
			name: fmt.Sprintf("with static %s", ty.typ.ID()),
			val: cadence.TypeValue{
				StaticType: ty.typ,
			},
			expected: w.Bytes(),
			// language=json, format=json-cdc
			// {"type":"Type","value":{"staticType":{"kind":"[ty.ID()]"}}}
			//
			// language=edn, format=ccf
			// 130([137(41), 185(simple_type_id)])
		})
	}

	testAllEncodeAndDecode(t, tests...)
}

func TestEncodeType(t *testing.T) {

	t.Parallel()

	t.Run("with static int?", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.OptionalType{Type: cadence.IntType{}},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Optional", "type" : {"kind" : "Int"}}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 186(185(4))])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
			},
		)

	})

	t.Run("with static int??", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.OptionalType{
					Type: &cadence.OptionalType{
						Type: cadence.IntType{},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Optional", "type" : {"kind" : "Int"}}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 186(186(185(4)))])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
			},
		)

	})
	t.Run("with static [int]", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.VariableSizedArrayType{
					ElementType: cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"VariableSizedArray", "type" : {"kind" : "Int"}}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 187(185(4))])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayTypeValue,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
			},
		)

	})

	t.Run("with static [int; 3]", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ConstantSizedArrayType{
					ElementType: cadence.IntType{},
					Size:        3,
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"ConstantSizedArray", "type" : {"kind" : "Int"}, "size" : 3}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 188([3, 185(4)])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagConstsizedArrayTypeValue,
				// array, 2 elements follow
				0x82,
				// 3
				0x03,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
			},
		)

	})

	t.Run("with static {int:string}", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.DictionaryType{
					ElementType: cadence.StringType{},
					KeyType:     cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Dictionary", "key" : {"kind" : "Int"}, "value" : {"kind" : "String"}}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 189([185(4), 185(1)])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagDictTypeValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)

	})

	t.Run("with static struct with no field", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.StructType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "S",
					Fields:              []cadence.Field{},
					Initializers:        [][]cadence.Parameter{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Struct","typeID":"S.test.S","fields":[],"initializers":[]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 208([h'', "S.test.S", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.So
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		)
	})

	t.Run("with static struct no sort", func(t *testing.T) {
		t.Parallel()

		val := cadence.TypeValue{
			StaticType: &cadence.StructType{
				Location:            utils.TestLocation,
				QualifiedIdentifier: "S",
				Fields: []cadence.Field{
					{Identifier: "foo", Type: cadence.IntType{}},
					{Identifier: "bar", Type: cadence.IntType{}},
				},
				Initializers: [][]cadence.Parameter{
					{
						{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
						{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
					},
				},
			},
		}

		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"value":{"staticType":{"type":"","kind":"Struct","typeID":"S.test.S","fields":[{"type":{"kind":"Int"},"id":"foo"},{"type":{"kind":"Int"},"id":"bar"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"},{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
			//
			// language=edn, format=ccf
			// 130([137(41), 208([h'', "S.test.S", null, [["foo", 185(4)], ["bar", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 elements follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Meta type ID (41)
			0x18, 0x29,
			// tag
			0xd8, ccf.CBORTagStructTypeValue,
			// array, 5 elements follow
			0x85,
			// bytes, 0 bytes follow
			0x40,
			// string, 8 bytes follow
			0x68,
			// S.test.So
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
			// type (nil for struct)
			0xf6,
			// fields
			// array, 2 element follows
			0x82,
			// array, 2 elements follow
			0x82,
			// string, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
			// tag
			0xd8, ccf.CBORTagSimpleTypeValue,
			// Int type (4)
			0x04,
			// array, 2 elements follow
			0x82,
			// string, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleTypeValue,
			// Int type (4)
			0x04,
			// initializers
			// array, 1 elements follow
			0x81,
			// array, 2 element follows
			0x82,
			// array, 3 elements follow
			0x83,
			// string, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
			// string, 3 bytes follow
			0x63,
			// bar
			0x62, 0x61, 0x72,
			// tag
			0xd8, ccf.CBORTagSimpleTypeValue,
			// Int type (4)
			0x04,
			// array, 3 elements follow
			0x83,
			// string, 3 bytes follow
			0x63,
			// qux
			0x71, 0x75, 0x78,
			// string, 3 bytes follow
			0x63,
			// bax
			0x62, 0x61, 0x7a,
			// tag
			0xd8, ccf.CBORTagSimpleTypeValue,
			// String type (1)
			0x01,
		}

		// Encode value without sorting of composite fields.
		actualCBOR, err := ccf.Encode(val)
		require.NoError(t, err)
		utils.AssertEqualWithDiff(t, expectedCBOR, actualCBOR)

		// Decode value without enforcing sorting of composite fields.
		decodedVal, err := ccf.Decode(nil, actualCBOR)
		require.NoError(t, err)
		assert.Equal(
			t,
			cadence.ValueWithCachedTypeID(val),
			cadence.ValueWithCachedTypeID(decodedVal),
		)
	})

	t.Run("with static struct", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecodeEx(
			t,
			cadence.TypeValue{
				StaticType: &cadence.StructType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "S",
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
						{Identifier: "bar", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Struct","typeID":"S.test.S","fields":[{"type":{"kind":"Int"},"id":"foo"},{"type":{"kind":"Int"},"id":"bar"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"},{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 208([h'', "S.test.S", null, [["bar", 185(4)], ["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.So
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 2 element follows
				0x82,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
			cadence.TypeValue{
				StaticType: &cadence.StructType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "S",
					Fields: []cadence.Field{
						{Identifier: "bar", Type: cadence.IntType{}},
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
		)
	})

	t.Run("with static resource of composite fields and initializers", func(t *testing.T) {
		t.Parallel()

		fooTy := &cadence.ResourceType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields:              []cadence.Field{},
			Initializers:        [][]cadence.Parameter{},
		}

		fooTy2 := &cadence.ResourceType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo2",
			Fields:              []cadence.Field{},
			Initializers:        [][]cadence.Parameter{},
		}

		barTy := &cadence.ResourceType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Bar",
			Fields: []cadence.Field{
				{
					Identifier: "foo1",
					Type:       fooTy,
				},
			},
			Initializers: [][]cadence.Parameter{
				{
					cadence.Parameter{
						Type:       fooTy2,
						Label:      "aaa",
						Identifier: "aaa",
					},
				},
			},
		}

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: barTy,
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Resource","typeID":"S.test.Bar","fields":[{"type":{"type":"","kind":"Resource","typeID":"S.test.Foo","fields":[],"initializers":[]},"id":"foo1"},{"type":"S.test.Foo","id":"foo2"}],"initializers":[[{"type":{"type":"S.test.Foo","kind":"Optional"},"label":"aaa","id":"aaa"}],[{"type":{"type":"","kind":"Resource","typeID":"S.test.Foo2","fields":[],"initializers":[]},"label":"bbb","id":"bbb"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 209([h'', "S.test.Bar", null, [["foo1", 209([h'01', "S.test.Foo", null, [], []])]], [[["aaa", "aaa", 209([h'02', "S.test.Foo2", null, [], []])]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 10 bytes follow
				0x6a,
				// S.test.Bar
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x42, 0x61, 0x72,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 4 bytes follow
				0x64,
				// foo1
				0x66, 0x6f, 0x6f, 0x31,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 0 elements follow
				0x80,
				// initializer
				// array, 0 elements follow
				0x80,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 1 elements follow
				0x81,
				// array, 3 elements follow
				0x83,
				// text, 3 bytes follow
				0x63,
				// aaa
				0x61, 0x61, 0x61,
				// text, 3 bytes follow
				0x63,
				// aaa
				0x61, 0x61, 0x61,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// CCF type ID
				// bytes, 1 byte follows
				0x41,
				// 2
				0x02,
				// text, 11 bytes follow
				0x6b,
				// S.test.Foo2
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x32,
				// null
				0xf6,
				// array, 0 element follows
				0x80,
				// array, 0 element follows
				0x80,
			},
		)
	})

	t.Run("with static resource", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ResourceType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "R",
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Resource","typeID":"S.test.R","fields":[{"type":{"kind":"Int"},"id":"foo"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"},{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 209([h'', "S.test.R", null, [["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.R
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x52,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)
	})

	t.Run("with static contract", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ContractType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "C",
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Contract","typeID":"S.test.C","fields":[{"type":{"kind":"Int"},"id":"foo"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"}],[{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 211([h'', "S.test.C", null, [["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (42)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagContractTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.C
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x43,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)
	})

	t.Run("with static struct interface", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.StructInterfaceType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "S",
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"StructInterface","typeID":"S.test.S","fields":[{"type":{"kind":"Int"},"id":"foo"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"}],[{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 224([h'', "S.test.S", null, [["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.S
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)
	})

	t.Run("with static resource interface", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ResourceInterfaceType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "R",
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"ResourceInterface","typeID":"S.test.R","fields":[{"type":{"kind":"Int"},"id":"foo"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"}],[{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 225([h'', "S.test.R", null, [["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagResourceInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.R
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x52,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)
	})

	t.Run("with static contract interface", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ContractInterfaceType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "C",
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"ContractInterface","typeID":"S.test.C","fields":[{"type":{"kind":"Int"},"id":"foo"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"}],[{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 226([h'', "S.test.C", null, [["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagContractInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.C
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x43,
				// type (nil for contract interface)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)
	})

	t.Run("with static event", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.EventType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "E",
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializer: []cadence.Parameter{
						{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
						{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type", "value": {"staticType": {"kind": "Event", "type" : "", "typeID" : "S.test.E", "fields" : [ {"id" : "foo", "type": {"kind" : "Int"} } ], "initializers" : [[{"label" : "foo", "id" : "bar", "type": {"kind" : "Int"}}, {"label" : "qux", "id" : "baz", "type": {"kind" : "String"}}]] } } }
				//
				// language=edn, format=ccf
				// 130([137(41), 210([h'', "S.test.E", null, [["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagEventTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.E
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x45,
				// type (nil for event)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// baz
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)
	})

	t.Run("with static enum", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.EnumType{
					Location:            utils.TestLocation,
					QualifiedIdentifier: "E",
					RawType:             cadence.StringType{},
					Fields: []cadence.Field{
						{Identifier: "foo", Type: cadence.IntType{}},
					},
					Initializers: [][]cadence.Parameter{
						{
							{Label: "foo", Identifier: "bar", Type: cadence.IntType{}},
							{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
						},
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":{"kind":"String"},"kind":"Enum","typeID":"S.test.E","fields":[{"type":{"kind":"Int"},"id":"foo"}],"initializers":[[{"type":{"kind":"Int"},"label":"foo","id":"bar"}],[{"type":{"kind":"String"},"label":"qux","id":"baz"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 212([h'', "S.test.E", 185(1), [["foo", 185(4)]], [[["foo", "bar", 185(4)], ["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagEnumTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.E
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x45,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type ID (1)
				0x01,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 2 element follows
				0x82,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		)
	})

	t.Run("with static &int", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.ReferenceType{
					Authorization: cadence.UnauthorizedAccess,
					Type:          cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Reference", "type" : {"kind" : "Int"}, "authorized" : false}}}`
				//
				// language=edn, format=ccf
				// 130([137(41), 190([false, 185(4)])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagReferenceTypeValue,
				// array, 2 elements follow
				0x82,
				// nil
				0xf6,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type ID (4)
				0x04,
			},
		)

	})

	t.Run("with static function", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.FunctionType{
					TypeParameters: []cadence.TypeParameter{
						{Name: "T", TypeBound: cadence.AnyStructType{}},
					},
					Parameters: []cadence.Parameter{
						{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
					},
					ReturnType: cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"kind":"Function","typeParameters":[{"name":"T","typeBound":{"kind":"AnyStruct"}}],"parameters":[{"type":{"kind":"String"},"label":"qux","id":"baz"}],"return":{"kind":"Int"}}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 193([[["T", 185(39)]], [["qux", "baz", 185(1)]], 185(4)])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagFunctionTypeValue,
				// array, 3 elements follow
				0x83,
				// array, 1 elements follow
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 1 byte follows
				0x61,
				// "T"
				0x54,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// AnyStruct type (39)
				0x18, 0x27,
				// array, 1 elements follow
				0x81,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type ID (4)
				0x04,
			},
		)
	})

	t.Run("with static function nil type bound", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.FunctionType{
					TypeParameters: []cadence.TypeParameter{
						{Name: "T"},
					},
					Parameters: []cadence.Parameter{
						{Label: "qux", Identifier: "baz", Type: cadence.StringType{}},
					},
					ReturnType: cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"kind":"Function","typeParameters":[{"name":"T","typeBound":null}],"parameters":[{"type":{"kind":"String"},"label":"qux","id":"baz"}],"return":{"kind":"Int"}}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 193([[["T", null]], [["qux", "baz", 185(1)]], 185(4)])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagFunctionTypeValue,
				// array, 3 elements follow
				0x83,
				// array, 1 elements follow
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 1 byte follows
				0x61,
				// "T"
				0x54,
				// null
				0xf6,
				// array, 1 elements follow
				0x81,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type ID (4)
				0x04,
			},
		)
	})

	t.Run("with static unparameterized Capability", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.CapabilityType{},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Capability"}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 192([null])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagCapabilityTypeValue,
				// array, 1 element follows
				0x81,
				// null
				0xf6,
			},
		)
	})

	t.Run("with static Capability<Int>", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.CapabilityType{
					BorrowType: cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Capability", "type" : {"kind" : "Int"}}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 192([185(4)])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagCapabilityTypeValue,
				// array, 1 element follows
				0x81,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type ID (4)
				0x04,
			},
		)
	})

	t.Run("with static nil restricted type", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Restrictions: []cadence.Type{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"kind":"Restriction","type":"","restrictions":[]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 191([null, []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagRestrictedTypeValue,
				// array, 2 elements follow
				0x82,
				// null
				0xf6,
				// array, 0 element follows
				0x80,
			},
		)
	})

	t.Run("with static no restricted type", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecodeEx(
			t,
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Restrictions: []cadence.Type{},
					Type:         cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"kind":"Restriction","typeID":"Int{String}","type":{"kind":"Int"},"restrictions":[]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 191([185(4), []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagRestrictedTypeValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type ID (4)
				0x04,
				// array, 0 element follows
				0x80,
			},
			// Expected decoded RestrictedType doesn't have type ID.
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Restrictions: []cadence.Type{},
					Type:         cadence.IntType{},
				},
			},
		)
	})

	t.Run("with static restricted type", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecodeEx(
			t,
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Restrictions: []cadence.Type{
						cadence.StringType{},
					},
					Type: cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType": { "kind": "Restriction", "typeID":"Int{String}", "type" : {"kind" : "Int"}, "restrictions" : [ {"kind" : "String"} ]} } }
				//
				// language=edn, format=ccf
				// 130([137(41), 191([185(4), [185(1)]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagRestrictedTypeValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type ID (4)
				0x04,
				// array, 1 element follows
				0x81,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type ID (1)
				0x01,
			},
			// Expected decoded RestrictedType doesn't have type ID.
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Restrictions: []cadence.Type{
						cadence.StringType{},
					},
					Type: cadence.IntType{},
				},
			},
		)

	})

	t.Run("with static 2 restricted types", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecodeEx(
			t,
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Restrictions: []cadence.Type{
						cadence.NewAnyStructType(),
						cadence.StringType{},
					},
					Type: cadence.IntType{},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"kind":"Restriction","typeID":"Int{AnyStruct, String}","type":{"kind":"Int"},"restrictions":[{"kind":"AnyStruct"},{"kind":"String"}]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 191([185(4), [185(1), 185(39)]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagRestrictedTypeValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type ID (4)
				0x04,
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type ID (1)
				0x01,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// AnyStruct type ID (39)
				0x18, 0x27,
			},
			// Expected decoded RestrictedType has sorted restrictions and no type ID.
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Restrictions: []cadence.Type{
						cadence.StringType{},
						cadence.NewAnyStructType(),
					},
					Type: cadence.IntType{},
				},
			},
		)
	})

	t.Run("with static 3 restricted types", func(t *testing.T) {
		t.Parallel()

		// restrictedType is generated by fuzzer.
		testEncodeAndDecodeEx(
			t,
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Type: cadence.TheAnyStructType,
					Restrictions: []cadence.Type{
						cadence.NewStructInterfaceType(
							common.NewAddressLocation(nil, common.Address{0x01}, "TypeA"),
							"TypeA",
							[]cadence.Field{},
							[][]cadence.Parameter{},
						),
						cadence.NewStructInterfaceType(
							common.NewAddressLocation(nil, common.Address{0x01}, "TypeB"),
							"TypeB",
							[]cadence.Field{},
							[][]cadence.Parameter{},
						),
						cadence.NewStructInterfaceType(
							common.IdentifierLocation("LocationC"),
							"TypeC",
							[]cadence.Field{},
							[][]cadence.Parameter{},
						),
					},
				},
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"kind":"Restriction","typeID":"","type":{"kind":"AnyStruct"},"restrictions":[{"type":"","kind":"StructInterface","typeID":"A.0100000000000000.TypeA","fields":[],"initializers":[]},{"type":"","kind":"StructInterface","typeID":"A.0100000000000000.TypeB","fields":[],"initializers":[]},{"type":"","kind":"StructInterface","typeID":"I.LocationC.TypeC","fields":[],"initializers":[]}]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 191([185(39), [224([h'', "I.LocationC.TypeC", null, [], []]), 224([h'01', "A.0100000000000000.TypeA", null, [], []]), 224([h'02', "A.0100000000000000.TypeB", null, [], []])]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagRestrictedTypeValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// 3 sorted restrictions
				// array, 3 element follows
				0x83,
				// tag
				0xd8, ccf.CBORTagStructInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// CCF type ID
				// bytes, 0 byte follows
				0x40,
				// cadence type ID
				// text, 17 bytes follow
				0x71,
				// "I.LocationC.TypeC"
				0x49, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x43,
				// type
				// null
				0xf6,
				// array, 0 element follows
				0x80,
				// array, 0 element follows
				0x80,
				// tag
				0xd8, ccf.CBORTagStructInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// CCF type ID
				// bytes, 1 byte follows
				0x41,
				// 1
				0x01,
				// cadence type ID
				// text, 24 bytes follow
				0x78, 0x18,
				// "A.0100000000000000.TypeA"
				0x41, 0x2e, 0x30, 0x31, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x41,
				// type
				// null
				0xf6,
				// array, 0 element follows
				0x80,
				// array, 0 element follows
				0x80,
				// tag
				0xd8, ccf.CBORTagStructInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// CCF type ID
				// bytes, 1 byte follows
				0x41,
				// 2
				0x02,
				// cadence type ID
				// text, 24 bytes follow
				0x78, 0x18,
				// "A.0100000000000000.TypeB"
				0x41, 0x2e, 0x30, 0x31, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x42,
				// type
				// null
				0xf6,
				// array, 0 element follows
				0x80,
				// array, 0 element follows
				0x80,
			},
			// Expected decoded RestrictedType has sorted restrictions and no type ID.
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Type: cadence.TheAnyStructType,
					Restrictions: []cadence.Type{
						cadence.NewStructInterfaceType(
							common.IdentifierLocation("LocationC"),
							"TypeC",
							[]cadence.Field{},
							[][]cadence.Parameter{},
						),
						cadence.NewStructInterfaceType(
							common.NewAddressLocation(nil, common.Address{0x01}, "TypeA"),
							"TypeA",
							[]cadence.Field{},
							[][]cadence.Parameter{},
						),
						cadence.NewStructInterfaceType(
							common.NewAddressLocation(nil, common.Address{0x01}, "TypeB"),
							"TypeB",
							[]cadence.Field{},
							[][]cadence.Parameter{},
						),
					},
				},
			},
		)
	})

	t.Run("without static type", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.TypeValue{},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":""}}
				//
				// language=edn, format=ccf
				// 130([137(41), null])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// nil
				0xf6,
			},
		)
	})
}

func TestEncodeIDCapability(t *testing.T) {

	t.Parallel()

	t.Run("unparameterized Capability", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.IDCapability{
				ID:      42,
				Address: cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
			},
			[]byte{
				// language=edn, format=ccf
				// 130([144([null]), [h'0000000102030405', 42]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagCapabilityType,
				// array, 1 element follows
				0x81,
				// null
				0xf6,
				// array, 2 elements follow
				0x82,
				// address
				// bytes, 8 bytes follow
				0x48,
				// {1,2,3,4,5}
				0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
				// 42
				0x18, 0x2a,
			},
		)
	})

	t.Run("array of unparameterized Capability", func(t *testing.T) {
		t.Parallel()

		simpleStructType := &cadence.StructType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "FooStruct",
			Fields: []cadence.Field{
				{
					Identifier: "bar",
					Type:       cadence.IntType{},
				},
			},
		}

		capability1 := cadence.IDCapability{
			ID:         42,
			Address:    cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
			BorrowType: cadence.IntType{},
		}

		capability2 := cadence.IDCapability{
			ID:         43,
			Address:    cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
			BorrowType: simpleStructType,
		}

		testEncodeAndDecode(
			t,
			cadence.NewArray([]cadence.Value{
				capability1,
				capability2,
			}).WithType(cadence.NewVariableSizedArrayType(cadence.NewCapabilityType(nil))),
			[]byte{
				// language=edn, format=ccf
				// 129([[160([h'', "S.test.FooStruct", [["bar", 137(4)]]])], [139(144([null])), [130([144([137(4)]), [h'0000000102030405', 42]]), 130([144([136(h'')]), [h'0000000102030405', 43]])]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 1 items follow
				0x81,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.FooStruct"
				// fields: [["bar", IntType]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 16 bytes follow
				0x70,
				// S.test.FooStruct
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
				// fields
				// array, 1 items follow
				0x81,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,

				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagCapabilityType,
				// array, 1 element follows
				0x81,
				// null
				0xf6,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagCapabilityType,
				// array, 1 elements follow
				0x81,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// array, 2 elements follow
				0x82,
				// address
				// bytes, 8 bytes follow
				0x48,
				// {1,2,3,4,5}
				0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
				// 42
				0x18, 0x2a,

				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagCapabilityType,
				// array, 1 elements follow
				0x81,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 byte follows
				0x40,
				// array, 2 elements follow
				0x82,
				// address
				// bytes, 8 bytes follow
				0x48,
				// {1,2,3,4,5}
				0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
				// 43
				0x18, 0x2b,
			},
		)
	})

	t.Run("Capability<Int>", func(t *testing.T) {
		t.Parallel()

		testEncodeAndDecode(
			t,
			cadence.IDCapability{
				ID:         42,
				Address:    cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
				BorrowType: cadence.IntType{},
			},
			[]byte{
				// language=edn, format=ccf
				// 130([144([137(4)]), [h'0000000102030405', 42]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagCapabilityType,
				// array, 1 element follows
				0x81,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// array, 2 elements follow
				0x82,
				// address
				// bytes, 8 bytes follow
				0x48,
				// {1,2,3,4,5}
				0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
				// 42
				0x18, 0x2a,
			},
		)
	})

	t.Run("array of Capability<Int>", func(t *testing.T) {
		t.Parallel()

		capability1 := cadence.IDCapability{
			ID:         42,
			Address:    cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
			BorrowType: cadence.IntType{},
		}
		capability2 := cadence.IDCapability{
			ID:         43,
			Address:    cadence.BytesToAddress([]byte{1, 2, 3, 4, 5}),
			BorrowType: cadence.IntType{},
		}

		testEncodeAndDecode(
			t,
			cadence.NewArray([]cadence.Value{
				capability1,
				capability2,
			}).WithType(cadence.NewVariableSizedArrayType(cadence.NewCapabilityType(cadence.NewIntType()))),
			[]byte{
				// language=edn, format=ccf
				// 130([139(144([137(4)])), [[h'0000000102030405', 42], [h'0000000102030405', 43]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagCapabilityType,
				// array, 1 element follows
				0x81,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// array, 2 elements follow
				0x82,
				// array, 2 elements follow
				0x82,
				// address
				// bytes, 8 bytes follow
				0x48,
				// {1,2,3,4,5}
				0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
				// 42
				0x18, 0x2a,
				// array, 2 elements follow
				0x82,
				// address
				// bytes, 8 bytes follow
				0x48,
				// {1,2,3,4,5}
				0x00, 0x00, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
				// 43
				0x18, 0x2b,
			},
		)
	})
}

func TestDecodeFix64(t *testing.T) {

	t.Parallel()

	var maxInt int64 = sema.Fix64TypeMaxInt
	var minInt int64 = sema.Fix64TypeMinInt
	var maxFrac int64 = sema.Fix64TypeMaxFractional
	var minFrac int64 = sema.Fix64TypeMinFractional
	var factor int64 = sema.Fix64Factor

	type testCase struct {
		name        string
		expected    cadence.Fix64
		encodedData []byte
		check       func(t *testing.T, actual cadence.Value, err error)
	}

	var testCases = []testCase{
		{
			name:     "12.3",
			expected: cadence.Fix64(12_30000000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "12.3"}
				//
				// language=edn, format=ccf
				// 130([137(22), 1230000000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 1230000000
				0x1a, 0x49, 0x50, 0x4f, 0x80,
			},
		},
		{
			name:     "12.03",
			expected: cadence.Fix64(12_03000000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "12.03"}
				//
				// language=edn, format=ccf
				// 130([137(22), 1203000000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 1203000000
				0x1a, 0x47, 0xb4, 0x52, 0xc0,
			},
		},
		{
			name:     "12.003",
			expected: cadence.Fix64(12_00300000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "12.003"}
				//
				// language=edn, format=ccf
				// 130([137(22), 1200300000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 1200300000
				0x1a, 0x47, 0x8b, 0x1f, 0xe0,
			},
		},
		{
			name:     "12.0003",
			expected: cadence.Fix64(12_00030000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "12.0003"}
				//
				// language=edn, format=ccf
				// 130([137(22), 1200030000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 1200030000
				0x1a, 0x47, 0x87, 0x01, 0x30,
			},
		},
		{
			name:     "12.00003",
			expected: cadence.Fix64(12_00003000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "12.00003"}
				//
				// language=edn, format=ccf
				// 130([137(22), 1200003000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 1200003000
				0x1a, 0x47, 0x86, 0x97, 0xb8,
			},
		},
		{
			name:     "12.000003",
			expected: cadence.Fix64(12_00000300),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "12.000003"}
				//
				// language=edn, format=ccf
				// 130([137(22), 1200000300])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 1200000300
				0x1a, 0x47, 0x86, 0x8d, 0x2c,
			},
		},
		{
			name:     "12.0000003",
			expected: cadence.Fix64(12_00000030),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "12.0000003"}
				//
				// language=edn, format=ccf
				// 130([137(22), 1200000030])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 1200000030
				0x1a, 0x47, 0x86, 0x8c, 0x1e,
			},
		},
		{
			name:     "120.3",
			expected: cadence.Fix64(120_30000000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "120.3"}
				//
				// language=edn, format=ccf
				// 130([137(22), 12030000000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 12030000000
				0x1b, 0x00, 0x00, 0x00, 0x02, 0xcd, 0x0b, 0x3b, 0x80,
			},
		},
		{
			// 92233720368.1
			name:     fmt.Sprintf("%d.1", maxInt),
			expected: cadence.Fix64(9223372036810000000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "92233720368.1"}
				//
				// language=edn, format=ccf
				// 130([137(22), 9223372036810000000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 9223372036810000000
				0x1b, 0x7f, 0xff, 0xff, 0xff, 0xfd, 0x54, 0xc6, 0x80,
			},
		},
		{
			// 92233720369.1
			name: fmt.Sprintf("%d.1", maxInt+1),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "92233720369.1"}
				//
				// language=edn, format=ccf
				// 130([137(22), 9223372036910000000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 9223372036910000000
				0x1b, 0x80, 0x00, 0x00, 0x00, 0x03, 0x4a, 0xa7, 0x80,
			},
			check: func(t *testing.T, actual cadence.Value, err error) {
				assert.Error(t, err)
			},
		},
		{
			// -92233720368.1
			name:     fmt.Sprintf("%d.1", minInt),
			expected: cadence.Fix64(-9223372036810000000),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "-92233720368.1"}
				//
				// language=edn, format=ccf
				// 130([137(22), -9223372036810000000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// -9223372036810000000
				0x3b, 0x7f, 0xff, 0xff, 0xff, 0xfd, 0x54, 0xc6, 0x7f,
			},
		},
		{
			// -92233720369.1
			name: fmt.Sprintf("%d.1", minInt-1),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "-92233720369.1"}
				//
				// language=edn, format=ccf
				// 130([137(22), -9223372036910000000])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// -9223372036910000000
				0x3b, 0x80, 0x00, 0x00, 0x00, 0x03, 0x4a, 0xa7, 0x7f,
			},
			check: func(t *testing.T, actual cadence.Value, err error) {
				assert.Error(t, err)
			},
		},
		{
			// 92233720368.54775807
			name:     fmt.Sprintf("%d.%d", maxInt, maxFrac),
			expected: cadence.Fix64(maxInt*factor + maxFrac),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "92233720368.54775807"}
				//
				// language=edn, format=ccf
				// 130([137(22), 9223372036854775807])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 9223372036854775807
				0x1b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			// 92233720368.54775808
			name: fmt.Sprintf("%d.%d", maxInt, maxFrac+1),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "92233720368.54775808"}
				//
				// language=edn, format=ccf
				// 130([137(22), 9223372036854775808])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// 9223372036854775808
				0x1b, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			check: func(t *testing.T, actual cadence.Value, err error) {
				assert.Error(t, err)
			},
		},
		{
			// -92233720368.54775808
			name:     fmt.Sprintf("%d.%d", minInt, -(minFrac)),
			expected: cadence.Fix64(-9223372036854775808),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "-92233720368.54775808"}
				//
				// language=edn, format=ccf
				// 130([137(22), -9223372036854775808])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// -9223372036854775808
				0x3b, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		{
			// -92233720368.54775809
			name: fmt.Sprintf("%d.%d", minInt, -(minFrac - 1)),
			encodedData: []byte{
				// language=json, format=json-cdc
				// {"type": "Fix64", "value": "-92233720368.54775809"}
				//
				// language=edn, format=ccf
				// 130([137(22), -9223372036854775809])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// fix64 type ID (22)
				0x16,
				// -9223372036854775809
				0x3b, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			check: func(t *testing.T, actual cadence.Value, err error) {
				assert.Error(t, err)
			},
		},
	}

	test := func(tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decModes := []ccf.DecMode{ccf.EventsDecMode, deterministicDecMode}

			for _, dm := range decModes {
				actual, err := dm.Decode(nil, tc.encodedData)
				if tc.check != nil {
					tc.check(t, actual, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expected, actual)
				}
			}
		})
	}

	for _, tc := range testCases {
		test(tc)
	}
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

	ty.Fields[0].Type = &cadence.OptionalType{
		Type: ty,
	}

	testEncode(
		t,
		cadence.Resource{
			Fields: []cadence.Value{
				cadence.Optional{},
			},
		}.WithType(ty),
		[]byte{
			// language=json, format=json-cdc
			// {"type":"Resource","value":{"id":"S.test.Foo","fields":[{"name":"foo","value":{"type": "Optional","value":null}}]}}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Foo", [["foo", 138(136(h''))]]])], [136(h''), [null]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definition
			// array, 1 items follow
			0x81,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Foo"
			// 1 fields: [["foo", optional(type ref id(0))]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 10 bytes follow
			0x6a,
			// S.test.Foo
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
			// fields
			// array, 1 items follow
			0x81,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// foo
			0x66, 0x6f, 0x6f,
			// tag
			0xd8, ccf.CBORTagOptionalType,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 1 items follow
			0x81,
			// nil
			0xf6,
		},
	)

}

func TestExportTypeValueRecursiveType(t *testing.T) {

	t.Parallel()

	t.Run("recursive field", func(t *testing.T) {

		t.Parallel()

		ty := &cadence.ResourceType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "foo",
				},
			},
			Initializers: [][]cadence.Parameter{},
		}

		ty.Fields[0].Type = &cadence.OptionalType{
			Type: ty,
		}

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: ty,
			},
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Resource","typeID":"S.test.Foo","fields":[{"id":"foo","type":{"kind":"Optional","type":"S.test.Foo"}}],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 209([h'', "S.test.Foo", null, [["foo", 186(184(h''))]], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 4 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6F, 0x6F,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagTypeValueRef,
				// bytes, 0 bytes follow
				0x40,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		)

	})

	t.Run("recursive initializer", func(t *testing.T) {
		t.Parallel()

		// structType is generated by fuzzer.
		structType := cadence.NewStructType(
			common.StringLocation("foo"),
			"Foo",
			[]cadence.Field{},
			[][]cadence.Parameter{{{}}},
		)

		structType.Initializers[0][0] = cadence.Parameter{
			Type:       &cadence.OptionalType{Type: structType},
			Label:      "aaa",
			Identifier: "aaa",
		}

		typeValue := cadence.NewTypeValue(structType)

		testEncodeAndDecode(
			t,
			typeValue,
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Struct","typeID":"S.foo.Foo","fields":[],"initializers":[[{"type":{"type":"S.foo.Foo","kind":"Optional"},"label":"aaa","id":"aaa"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 208([h'', "S.foo.Foo", null, [], [[["aaa", "aaa", 186(184(h''))]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 4 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 9 bytes follow
				0x69,
				// "S.foo.Foo"
				0x53, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x46, 0x6f, 0x6f,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 1 element follows
				0x81,
				// array, 1 element follows
				0x81,
				// array, 3 elements follow
				0x83,
				// text, 3 bytes follow
				0x63,
				// "aaa"
				0x61, 0x61, 0x61,
				// text, 3 bytes follow
				0x63,
				// "aaa"
				0x61, 0x61, 0x61,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagTypeValueRef,
				// bytes, 0 byte follows,
				0x40,
			},
		)
	})

	t.Run("recursive field and initializer", func(t *testing.T) {
		t.Parallel()

		// structType is generated by fuzzer.
		structType := cadence.NewStructType(
			common.StringLocation("foo"),
			"Foo",
			[]cadence.Field{{}},
			[][]cadence.Parameter{{{}}},
		)

		structType.Fields[0] = cadence.Field{
			Type:       &cadence.OptionalType{Type: structType},
			Identifier: "aa",
		}

		structType.Initializers[0][0] = cadence.Parameter{
			Type:       &cadence.OptionalType{Type: structType},
			Label:      "aaa",
			Identifier: "aaa",
		}

		typeValue := cadence.NewTypeValue(structType)

		testEncodeAndDecode(
			t,
			typeValue,
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Struct","typeID":"S.foo.Foo","fields":[{"type":{"type":"S.foo.Foo","kind":"Optional"},"id":"aa"}],"initializers":[[{"type":{"type":"S.foo.Foo","kind":"Optional"},"label":"aaa","id":"aaa"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 208([h'', "S.foo.Foo", null, [["aa", 186(184(h''))]], [[["aaa", "aaa", 186(184(h''))]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 4 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 9 bytes follow
				0x69,
				// "S.foo.Foo"
				0x53, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x46, 0x6f, 0x6f,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// text, 2 bytes follow
				0x62,
				// "aa"
				0x61, 0x61,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagTypeValueRef,
				// bytes, 0 byte follows
				0x40,
				// initializers
				// array, 1 element follows
				0x81,
				// array, 1 element follows
				0x81,
				// array, 3 elements follow
				0x83,
				// text, 3 bytes follow
				0x63,
				// "aaa"
				0x61, 0x61, 0x61,
				// text, 3 bytes follow
				0x63,
				// "aaa"
				0x61, 0x61, 0x61,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagTypeValueRef,
				// bytes, 0 byte follows,
				0x40,
			},
		)
	})

	t.Run("non-recursive, repeated", func(t *testing.T) {

		t.Parallel()

		fooTy := &cadence.ResourceType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo",
			Fields:              []cadence.Field{},
			Initializers:        [][]cadence.Parameter{},
		}

		fooTy2 := &cadence.ResourceType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Foo2",
			Fields:              []cadence.Field{},
			Initializers:        [][]cadence.Parameter{},
		}

		barTy := &cadence.ResourceType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: "Bar",
			Fields: []cadence.Field{
				{
					Identifier: "foo1",
					Type:       fooTy,
				},
				{
					Identifier: "foo2",
					Type:       fooTy,
				},
			},
			Initializers: [][]cadence.Parameter{
				{
					cadence.Parameter{
						Type:       &cadence.OptionalType{Type: fooTy},
						Label:      "bbb",
						Identifier: "bbb",
					},
					cadence.Parameter{
						Type:       fooTy2,
						Label:      "aaa",
						Identifier: "aaa",
					},
				},
			},
		}

		testEncodeAndDecode(
			t,
			cadence.TypeValue{
				StaticType: barTy,
			},
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"staticType":{"type":"","kind":"Resource","typeID":"S.test.Bar","fields":[{"type":{"type":"","kind":"Resource","typeID":"S.test.Foo","fields":[],"initializers":[]},"id":"foo1"},{"type":"S.test.Foo","id":"foo2"}],"initializers":[[{"type":{"type":"S.test.Foo","kind":"Optional"},"label":"bbb","id":"bbb"},{"type":{"type":"","kind":"Resource","typeID":"S.test.Foo2","fields":[],"initializers":[]},"label":"aaa","id":"aaa"}]]}},"type":"Type"}
				//
				// language=edn, format=ccf
				// 130([137(41), 209([h'', "S.test.Bar", null, [["foo1", 209([h'01', "S.test.Foo", null, [], []])], ["foo2", 184(h'01')]], [[["bbb", "bbb", 186(184(h'01'))], ["aaa", "aaa", 209([h'02', "S.test.Foo2", null, [], []])]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 10 bytes follow
				0x6a,
				// S.test.Bar
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x42, 0x61, 0x72,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 2 element follows
				0x82,
				// array, 2 elements follow
				0x82,
				// string, 4 bytes follow
				0x64,
				// foo1
				0x66, 0x6f, 0x6f, 0x31,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// string, 10 bytes follow
				0x6a,
				// S.test.Foo
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 0 elements follow
				0x80,
				// initializer
				// array, 0 elements follow
				0x80,
				// array, 2 elements follow
				0x82,
				// string, 4 bytes follow
				0x64,
				// foo2
				0x66, 0x6f, 0x6f, 0x32,
				// tag
				0xd8, ccf.CBORTagTypeValueRef,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// initializers
				// array, 1 element follow
				0x81,
				// array, 2 elements follow
				0x82,
				// array, 3 elements follow
				0x83,
				// text, 3 bytes follow
				0x63,
				// bbb
				0x62, 0x62, 0x62,
				// text, 3 bytes follow
				0x63,
				// bbb
				0x62, 0x62, 0x62,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// tag
				0xd8, ccf.CBORTagTypeValueRef,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
				// array, 3 elements follow
				0x83,
				// text, 3 bytes follow
				0x63,
				// aaa
				0x61, 0x61, 0x61,
				// text, 3 bytes follow
				0x63,
				// aaa
				0x61, 0x61, 0x61,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// CCF type ID
				// bytes, 1 byte follows
				0x41,
				// 2
				0x02,
				// text, 11 bytes follow
				0x6b,
				// S.test.Foo2
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x32,
				// null
				0xf6,
				// array, 0 element follows
				0x80,
				// array, 0 element follows
				0x80,
			},
		)
	})
}

func TestEncodePath(t *testing.T) {

	t.Parallel()

	t.Run("Storage", func(t *testing.T) {
		t.Parallel()

		path, err := cadence.NewPath(1, "foo")
		require.NoError(t, err)

		testEncodeAndDecode(
			t,
			path,
			[]byte{
				// language=json, format=json-cdc
				// {"value":{"domain":"storage","identifier":"foo"},"type":"Path"}
				//
				// language=edn, format=ccf
				// 130([137(26), [1, "foo"]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// StoragePath type ID (26)
				0x18, 0x1a,
				// array, 2 elements follow
				0x82,
				// 1
				0x01,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
			},
		)
	})

	t.Run("Private", func(t *testing.T) {
		t.Parallel()

		path, err := cadence.NewPath(2, "foo")
		require.NoError(t, err)

		testEncodeAndDecode(
			t,
			path,
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Path","value":{"domain":"private","identifier":"foo"}}
				//
				// language=edn, format=ccf
				// 130([137(28), [2, "foo"]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// PrivatePath type ID (28)
				0x18, 0x1c,
				// array, 2 elements follow
				0x82,
				// 2
				0x02,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
			},
		)
	})

	t.Run("Public", func(t *testing.T) {
		t.Parallel()

		path, err := cadence.NewPath(3, "foo")
		require.NoError(t, err)

		testEncodeAndDecode(
			t,
			path,
			[]byte{
				// language=json, format=json-cdc
				// {"type":"Path","value":{"domain":"public","identifier":"foo"}}
				//
				// language=edn, format=ccf
				// 130([137(27), [3, "foo"]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// PublicPath type ID (27)
				0x18, 0x1b,
				// array, 2 elements follow
				0x82,
				// 3
				0x03,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
			},
		)
	})

	t.Run("Array of StoragePath", func(t *testing.T) {
		t.Parallel()

		storagePath, err := cadence.NewPath(1, "foo")
		require.NoError(t, err)

		privatePath, err := cadence.NewPath(1, "bar")
		require.NoError(t, err)

		publicPath, err := cadence.NewPath(1, "baz")
		require.NoError(t, err)

		arrayOfPaths := cadence.NewArray([]cadence.Value{
			storagePath,
			privatePath,
			publicPath,
		}).WithType(cadence.NewVariableSizedArrayType(cadence.NewStoragePathType()))

		testEncodeAndDecode(
			t,
			arrayOfPaths,
			[]byte{
				// language=json, format=json-cdc
				// {"value":[{"value":{"domain":"storage","identifier":"foo"},"type":"Path"},{"value":{"domain":"private","identifier":"bar"},"type":"Path"},{"value":{"domain":"public","identifier":"baz"},"type":"Path"}],"type":"Array"}
				//
				// language=edn, format=ccf
				// 130([139(137(26)), [[1, "foo"], [1, "bar"], [1, "baz"]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// StoragePath type ID (26)
				0x18, 0x1a,
				// array, 3 elements follow
				0x83,
				// element 0: storage path
				// array, 2 elements follow
				0x82,
				// 1
				0x01,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// element 1: storage path
				// array, 2 elements follow
				0x82,
				// 1
				0x01,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// element 2: storage path
				// array, 2 elements follow
				0x82,
				// 1
				0x01,
				// string, 3 bytes follow
				0x63,
				// baz
				0x62, 0x61, 0x7a,
			},
		)
	})

	t.Run("Array of Path", func(t *testing.T) {
		t.Parallel()

		storagePath, err := cadence.NewPath(1, "foo")
		require.NoError(t, err)

		privatePath, err := cadence.NewPath(2, "bar")
		require.NoError(t, err)

		publicPath, err := cadence.NewPath(3, "baz")
		require.NoError(t, err)

		arrayOfPaths := cadence.NewArray([]cadence.Value{
			storagePath,
			privatePath,
			publicPath,
		}).WithType(cadence.NewVariableSizedArrayType(cadence.NewPathType()))

		testEncodeAndDecode(
			t,
			arrayOfPaths,
			[]byte{
				// language=json, format=json-cdc
				// {"value":[{"value":{"domain":"storage","identifier":"foo"},"type":"Path"},{"value":{"domain":"private","identifier":"bar"},"type":"Path"},{"value":{"domain":"public","identifier":"baz"},"type":"Path"}],"type":"Array"}
				//
				// language=edn, format=ccf
				// 130([139(137(24)), [130([137(26), [1, "foo"]]), 130([137(28), [2, "bar"]]), 130([137(27), [3, "baz"]])]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Path type ID (24)
				0x18, 0x18,
				// array, 3 elements follow
				0x83,
				// element 0: storage path
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// StoragePath type ID (26)
				0x18, 0x1a,
				// array, 2 elements follow
				0x82,
				// 1
				0x01,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// element 1: private path
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// PrivatePath type ID (28)
				0x18, 0x1c,
				// array, 2 elements follow
				0x82,
				// 2
				0x02,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// element 2: public path
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// PublicPath type ID (27)
				0x18, 0x1b,
				// array, 2 elements follow
				0x82,
				// 3
				0x03,
				// string, 3 bytes follow
				0x63,
				// baz
				0x62, 0x61, 0x7a,
			},
		)
	})
}

func testAllEncodeAndDecode(t *testing.T, testCases ...encodeTest) {

	test := func(tc encodeTest) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if tc.expectedVal == nil {
				testEncodeAndDecode(t, tc.val, tc.expected)
			} else {
				testEncodeAndDecodeEx(t, tc.val, tc.expected, tc.expectedVal)
			}
		})
	}

	for _, tc := range testCases {
		test(tc)
	}
}

func TestDecodeInvalidType(t *testing.T) {

	t.Parallel()

	t.Run("empty type", func(t *testing.T) {
		t.Parallel()

		encodedData := []byte{
			// language=json, format=json-cdc
			// { "type":"Struct", "value":{ "id":"", "fields":[] } }
			//
			// language=edn, format=ccf
			// 129([[160([h'', "", []])], [136(h''), []]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definition
			// array, 1 items follow
			0x81,
			// struct type:
			// id: []byte{}
			// cadence-type-id: ""
			// 0 field
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 0 bytes follow
			0x60,
			// fields
			// array, 0 items follow
			0x80,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 0 items follow
			0x80,
		}

		decModes := []ccf.DecMode{ccf.EventsDecMode, deterministicDecMode}

		for _, dm := range decModes {
			_, err := dm.Decode(nil, encodedData)
			require.Error(t, err)
			assert.Equal(t, "ccf: failed to decode: invalid type ID for built-in: ``", err.Error())
		}
	})

	t.Run("invalid type ID", func(t *testing.T) {
		t.Parallel()

		encodedData := []byte{
			// language=json, format=json-cdc
			// { "type":"Struct", "value":{ "id":"I", "fields":[] } }
			//
			// language=edn, format=ccf
			// 129([[160([h'', "I.Foo", []])], [136(h''), []]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definition
			// array, 1 items follow
			0x81,
			// struct type:
			// id: []byte{}
			// cadence-type-id: "I"
			// 0 field
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 1 bytes follow
			0x61,
			// I
			0x49,
			// fields
			// array, 0 items follow
			0x80,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 0 items follow
			0x80,
		}

		decModes := []ccf.DecMode{ccf.EventsDecMode, deterministicDecMode}

		for _, dm := range decModes {
			_, err := dm.Decode(nil, encodedData)
			require.Error(t, err)
			assert.Equal(t, "ccf: failed to decode: invalid type ID `I`: invalid identifier location type ID: missing location", err.Error())
		}
	})

	t.Run("unknown location prefix", func(t *testing.T) {
		t.Parallel()

		encodedData := []byte{
			// language=json, format=json-cdc
			// { "type":"Struct", "value":{ "id":"N.PublicKey", "fields":[] } }
			//
			// language=edn, format=ccf
			// 129([[160([h'', "N.PublicKey", []])], [136(h''), []]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 1 items follow
			0x81,
			// struct type:
			// id: []byte{}
			// cadence-type-id: "N.PublicKey"
			// 0 field
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 11 bytes follow
			0x6b,
			// N.PublicKey
			0x4e, 0x2e, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79,
			// fields
			// array, 0 items follow
			0x80,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 bytes follow
			0x40,
			// array, 0 items follow
			0x80,
		}

		decModes := []ccf.DecMode{ccf.EventsDecMode, deterministicDecMode}

		for _, dm := range decModes {
			_, err := dm.Decode(nil, encodedData)
			require.Error(t, err)
			assert.Equal(t, "ccf: failed to decode: invalid type ID for built-in: `N.PublicKey`", err.Error())
		}
	})
}

func testEncodeAndDecode(t *testing.T, val cadence.Value, expectedCBOR []byte) {
	actualCBOR := testEncode(t, val, expectedCBOR)
	testDecode(t, actualCBOR, val)
}

// testEncodeAndDecodeEx is used when val != expectedVal because of deterministic encoding.
func testEncodeAndDecodeEx(t *testing.T, val cadence.Value, expectedCBOR []byte, expectedVal cadence.Value) {
	actualCBOR := testEncode(t, val, expectedCBOR)
	testDecode(t, actualCBOR, expectedVal)
}

func testEncode(t *testing.T, val cadence.Value, expectedCBOR []byte) (actualCBOR []byte) {
	actualCBOR, err := deterministicEncMode.Encode(val)
	require.NoError(t, err)

	utils.AssertEqualWithDiff(t, expectedCBOR, actualCBOR)
	return actualCBOR
}

func testDecode(t *testing.T, actualCBOR []byte, expectedVal cadence.Value) {
	decodedVal, err := deterministicDecMode.Decode(nil, actualCBOR)
	require.NoError(t, err)
	assert.Equal(
		t,
		cadence.ValueWithCachedTypeID(expectedVal),
		cadence.ValueWithCachedTypeID(decodedVal),
	)
}

// TODO: make resource (illegal nesting)
func newResourceStructType() *cadence.StructType {
	return &cadence.StructType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "FooStruct",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.StringType{},
			},
			{
				Identifier: "b",
				Type:       newFooResourceType(),
			},
		},
	}
}

func newFooResourceType() *cadence.ResourceType {
	return &cadence.ResourceType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "bar",
				Type:       cadence.IntType{},
			},
		},
	}
}

func newFoooResourceTypeWithAbstractField() *cadence.ResourceType {
	return &cadence.ResourceType{
		Location:            utils.TestLocation,
		QualifiedIdentifier: "Fooo",
		Fields: []cadence.Field{
			{
				Identifier: "bar",
				Type:       cadence.IntType{},
			},
			{
				Identifier: "baz",
				Type:       cadence.AnyStructType{},
			},
		},
	}
}

func TestEncodeBuiltinComposites(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name    string
		typ     cadence.Type
		encoded []byte
	}

	testCases := []testCase{
		{
			name: "Struct",
			typ: &cadence.StructType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Struct","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 208([h'', "Foo", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		},
		{
			name: "StructInterface",
			typ: &cadence.StructInterfaceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"StructInterface","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 224([h'', "Foo", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		},
		{
			name: "Resource",
			typ: &cadence.ResourceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Resource","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 209([h'', "Foo", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagResourceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		},
		{
			name: "ResourceInterface",
			typ: &cadence.ResourceInterfaceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"ResourceInterface","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 225([h'', "Foo", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagResourceInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		},
		{
			name: "Contract",
			typ: &cadence.ContractType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Contract","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 211([h'', "Foo", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagContractTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		},
		{
			name: "ContractInterface",
			typ: &cadence.ContractInterfaceType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"ContractInterface","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 226([h'', "Foo", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagContractInterfaceTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		},
		{
			name: "Enum",
			typ: &cadence.EnumType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Enum","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 212([h'', "Foo", null, [], []])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagEnumTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 0 elements follow
				0x80,
			},
		},
		{
			name: "Event",
			typ: &cadence.EventType{
				Location:            nil,
				QualifiedIdentifier: "Foo",
			},
			encoded: []byte{
				// language=json, format=json-cdc
				// {"type":"Type","value":{"staticType":{"kind":"Event","typeID":"Foo","fields":[],"initializers":[],"type":""}}}
				//
				// language=edn, format=ccf
				// 130([137(41), 210([h'', "Foo", null, [], [[]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagEventTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 3 bytes follow
				0x63,
				// Foo
				0x46, 0x6f, 0x6f,
				// type (nil for event)
				0xf6,
				// fields
				// array, 0 element follows
				0x80,
				// initializers
				// array, 1 elements follow
				0x81,
				// array, 0 element follow
				0x80,
			},
		},
	}

	test := func(tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			typeValue := cadence.NewTypeValue(tc.typ)
			testEncode(t, typeValue, tc.encoded)
		})
	}

	for _, tc := range testCases {
		test(tc)
	}
}

func TestExportFunctionValue(t *testing.T) {

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

	ty.Fields[0].Type = &cadence.OptionalType{
		Type: ty,
	}

	testEncode(
		t,
		cadence.Function{
			FunctionType: &cadence.FunctionType{
				Parameters: []cadence.Parameter{},
				ReturnType: cadence.VoidType{},
			},
		},
		[]byte{
			// language=json, format=json-cdc
			// {"value":{"functionType":{"kind":"Function","typeParameters":[],"parameters":[],"return":{"kind":"Void"}}},"type":"Function"}
			//
			// language=edn, format=ccf
			// 130([137(51), [[], [], 185(50)]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 elements follow
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Function type ID (51)
			0x18, 0x33,
			// array, 3 elements follow
			0x83,
			// element 0: type parameters
			0x80,
			// element 1: parameters
			// array, 0 element
			0x80,
			// element 2: return type
			// tag
			0xd8, ccf.CBORTagSimpleTypeValue,
			// Void type ID (50)
			0x18, 0x32,
		},
	)
}

func TestDeployedEvents(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name         string
		event        cadence.Event
		expectedCBOR []byte
	}

	var testCases = []testCase{
		{
			name:  "FlowFees.FeesDeducted",
			event: createFlowFeesFeesDeductedEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.f919ee77447b7497.FlowFees.FeesDeducted","fields":[{"value":{"value":"0.01797293","type":"UFix64"},"name":"amount"},{"value":{"value":"1.00000000","type":"UFix64"},"name":"inclusionEffort"},{"value":{"value":"0.00360123","type":"UFix64"},"name":"executionEffort"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.f919ee77447b7497.FlowFees.FeesDeducted", [["amount", 137(23)], ["inclusionEffort", 137(23)], ["executionEffort", 137(23)]]])], [136(h''), [1797293, 100000000, 360123]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.f919ee77447b7497.FlowFees.FeesDeducted"
				// 3 fields: [["amount", type(ufix64)], ["executionEffort", type(ufix64)], ["inclusionEffort", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 elements follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 40 bytes follow
				0x78, 0x28,
				// A.f919ee77447b7497.FlowFees.FeesDeducted
				0x41, 0x2e, 0x66, 0x39, 0x31, 0x39, 0x65, 0x65, 0x37, 0x37, 0x34, 0x34, 0x37, 0x62, 0x37, 0x34, 0x39, 0x37, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x46, 0x65, 0x65, 0x73, 0x2e, 0x46, 0x65, 0x65, 0x73, 0x44, 0x65, 0x64, 0x75, 0x63, 0x74, 0x65, 0x64,
				// fields
				// array, 3 items follow
				0x83,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 6 bytes follow
				0x66,
				// amount
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 15 bytes follow
				0x6f,
				// inclusionEffort
				0x69, 0x6e, 0x63, 0x6c, 0x75, 0x73, 0x69, 0x6f, 0x6e, 0x45, 0x66, 0x66, 0x6f, 0x72, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 2
				// array, 2 items follow
				0x82,
				// text, 15 bytes follow
				0x6f,
				// executionEffort
				0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x45, 0x66, 0x66, 0x6f, 0x72, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 3 items follow
				0x83,
				// 1797293
				0x1a, 0x00, 0x1b, 0x6c, 0xad,
				// 100000000
				0x1a, 0x05, 0xf5, 0xe1, 0x00,
				// 360123
				0x1a, 0x00, 0x05, 0x7e, 0xbb,
			},
		},
		{
			name:  "FlowFees.TokensWithdrawn",
			event: createFlowFeesTokensWithdrawnEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.f919ee77447b7497.FlowFees.TokensWithdrawn","fields":[{"value":{"value":"53.04112895","type":"UFix64"},"name":"amount"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.f919ee77447b7497.FlowFees.TokensWithdrawn", [["amount", 137(23)]]])], [136(h''), [5304112895]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.f919ee77447b7497.FlowFees.TokensWithdrawn"
				// 1 field: [["amount", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 43 bytes follow
				0x78, 0x2b,
				// "A.f919ee77447b7497.FlowFees.TokensWithdrawn"
				0x41, 0x2e, 0x66, 0x39, 0x31, 0x39, 0x65, 0x65, 0x37, 0x37, 0x34, 0x34, 0x37, 0x62, 0x37, 0x34, 0x39, 0x37, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x46, 0x65, 0x65, 0x73, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x73, 0x57, 0x69, 0x74, 0x68, 0x64, 0x72, 0x61, 0x77, 0x6e,
				// fields
				// array, 1 items follow
				0x81,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "amount"
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 1 items follow
				0x81,
				// 5304112895
				0x1b, 0x00, 0x00, 0x00, 0x01, 0x3c, 0x26, 0x56, 0xff,
			},
		},
		{
			name:  "FlowIDTableStaking.DelegatorRewardsPaid",
			event: createFlowIDTableStakingDelegatorRewardsPaidEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.8624b52f9ddcd04a.FlowIDTableStaking.DelegatorRewardsPaid","fields":[{"value":{"value":"e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb","type":"String"},"name":"nodeID"},{"value":{"value":"92","type":"UInt32"},"name":"delegatorID"},{"value":{"value":"4.38760261","type":"UFix64"},"name":"amount"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.8624b52f9ddcd04a.FlowIDTableStaking.DelegatorRewardsPaid", [["nodeID", 137(1)], ["delegatorID", 137(14)], ["amount", 137(23)]]])], [136(h''), ["e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb", 92, 438760261]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,

				// event type:
				// id: []byte{}
				// cadence-type-id: "A.8624b52f9ddcd04a.FlowIDTableStaking.DelegatorRewardsPaid"
				// 3 field: [["nodeID", type(string)], ["delegatorID", type(uint32)], ["amount", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 58 bytes follow
				0x78, 0x3a,
				// "A.8624b52f9ddcd04a.FlowIDTableStaking.DelegatorRewardsPaid"
				0x41, 0x2e, 0x38, 0x36, 0x32, 0x34, 0x62, 0x35, 0x32, 0x66, 0x39, 0x64, 0x64, 0x63, 0x64, 0x30, 0x34, 0x61, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x49, 0x44, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x6b, 0x69, 0x6e, 0x67, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x67, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64, 0x73, 0x50, 0x61, 0x69, 0x64,
				// fields
				// array, 3 items follow
				0x83,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "nodeID"
				0x6e, 0x6f, 0x64, 0x65, 0x49, 0x44,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// field 1
				// array, 2 element follows
				0x82,
				// text, 11 bytes follow
				0x6b,
				// "delegatorID"
				0x64, 0x65, 0x6c, 0x65, 0x67, 0x61, 0x74, 0x6f, 0x72, 0x49, 0x44,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UInt32 type ID (14)
				0x0e,
				// field 2
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "amount"
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 3 items follow
				0x83,
				// text, 64 bytes follow
				0x78, 0x40,
				// "e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb"
				0x65, 0x35, 0x32, 0x63, 0x62, 0x63, 0x64, 0x38, 0x32, 0x35, 0x65, 0x33, 0x32, 0x38, 0x61, 0x63, 0x61, 0x63, 0x38, 0x64, 0x62, 0x36, 0x62, 0x63, 0x62, 0x64, 0x63, 0x62, 0x62, 0x36, 0x65, 0x37, 0x37, 0x32, 0x34, 0x38, 0x36, 0x32, 0x63, 0x38, 0x62, 0x38, 0x39, 0x62, 0x30, 0x39, 0x64, 0x38, 0x35, 0x65, 0x64, 0x63, 0x63, 0x66, 0x34, 0x31, 0x66, 0x66, 0x39, 0x39, 0x38, 0x31, 0x65, 0x62,
				// 92
				0x18, 0x5c,
				// 438760261
				0x1a, 0x1a, 0x26, 0xf3, 0x45,
			},
		},
		{
			name:  "FlowIDTableStaking.EpochTotalRewardsPaid",
			event: createFlowIDTableStakingEpochTotalRewardsPaidEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.8624b52f9ddcd04a.FlowIDTableStaking.EpochTotalRewardsPaid","fields":[{"value":{"value":"1316543.00000000","type":"UFix64"},"name":"total"},{"value":{"value":"53.04112895","type":"UFix64"},"name":"fromFees"},{"value":{"value":"1316489.95887105","type":"UFix64"},"name":"minted"},{"value":{"value":"6.04080767","type":"UFix64"},"name":"feesBurned"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.8624b52f9ddcd04a.FlowIDTableStaking.EpochTotalRewardsPaid", [["total", 137(23)], ["fromFees", 137(23)], ["minted", 137(23)], ["feesBurned", 137(23)]]])], [136(h''), [131654300000000, 5304112895, 131648995887105, 604080767]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.8624b52f9ddcd04a.FlowIDTableStaking.EpochTotalRewardsPaid"
				// 4 field: [["total", type(ufix64)], ["minted", type(ufix64)], ["fromFees", type(ufix64)], ["feesBurned", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 59 bytes follow
				0x78, 0x3b,
				// "A.8624b52f9ddcd04a.FlowIDTableStaking.EpochTotalRewardsPaid"
				0x41, 0x2e, 0x38, 0x36, 0x32, 0x34, 0x62, 0x35, 0x32, 0x66, 0x39, 0x64, 0x64, 0x63, 0x64, 0x30, 0x34, 0x61, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x49, 0x44, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x6b, 0x69, 0x6e, 0x67, 0x2e, 0x45, 0x70, 0x6f, 0x63, 0x68, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64, 0x73, 0x50, 0x61, 0x69, 0x64,
				// fields
				// array, 4 items follow
				0x84,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 5 bytes follow
				0x65,
				// "total"
				0x74, 0x6f, 0x74, 0x61, 0x6c,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 1
				// array, 2 element follows
				0x82,
				// text, 8 bytes follow
				0x68,
				// "fromFees"
				0x66, 0x72, 0x6f, 0x6d, 0x46, 0x65, 0x65, 0x73,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 2
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "minted"
				0x6d, 0x69, 0x6e, 0x74, 0x65, 0x64,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 3
				// array, 2 element follows
				0x82,
				// text, 10 bytes follow
				0x6a,
				// "feesBurned"
				0x66, 0x65, 0x65, 0x73, 0x42, 0x75, 0x72, 0x6e, 0x65, 0x64,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 4 items follow
				0x84,
				// 131654300000000
				0x1b, 0x00, 0x00, 0x77, 0xbd, 0x27, 0xc8, 0xdf, 0x00,
				// 5304112895
				0x1b, 0x00, 0x00, 0x00, 0x01, 0x3c, 0x26, 0x56, 0xff,
				// 131648995887105
				0x1b, 0x00, 0x00, 0x77, 0xbb, 0xeb, 0xa2, 0x88, 0x01,
				// 604080767
				0x1a, 0x24, 0x01, 0x8a, 0x7f,
			},
		},
		{
			name:  "FlowIDTableStaking.NewWeeklyPayout",
			event: createFlowIDTableStakingNewWeeklyPayoutEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.8624b52f9ddcd04a.FlowIDTableStaking.NewWeeklyPayout","fields":[{"value":{"value":"1317778.00000000","type":"UFix64"},"name":"newPayout"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.8624b52f9ddcd04a.FlowIDTableStaking.NewWeeklyPayout", [["newPayout", 137(23)]]])], [136(h''), [131777800000000]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.8624b52f9ddcd04a.FlowIDTableStaking.NewWeeklyPayout"
				// 1 field: [["newPayout", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 53 bytes follow
				0x78, 0x35,
				// "A.8624b52f9ddcd04a.FlowIDTableStaking.NewWeeklyPayout"
				0x41, 0x2e, 0x38, 0x36, 0x32, 0x34, 0x62, 0x35, 0x32, 0x66, 0x39, 0x64, 0x64, 0x63, 0x64, 0x30, 0x34, 0x61, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x49, 0x44, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x6b, 0x69, 0x6e, 0x67, 0x2e, 0x4e, 0x65, 0x77, 0x57, 0x65, 0x65, 0x6b, 0x6c, 0x79, 0x50, 0x61, 0x79, 0x6f, 0x75, 0x74,
				// fields
				// array, 1 items follow
				0x81,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 9 bytes follow
				0x69,
				// "newPayout"
				0x6e, 0x65, 0x77, 0x50, 0x61, 0x79, 0x6f, 0x75, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 1 items follow
				0x81,
				// 131777800000000
				0x1b, 0x00, 0x00, 0x77, 0xd9, 0xe8, 0xf5, 0x52, 0x00,
			},
		},
		{
			name:  "FlowIDTableStaking.RewardsPaid",
			event: createFlowIDTableStakingRewardsPaidEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.8624b52f9ddcd04a.FlowIDTableStaking.RewardsPaid","fields":[{"value":{"value":"e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb","type":"String"},"name":"nodeID"},{"value":{"value":"1745.49955740","type":"UFix64"},"name":"amount"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.8624b52f9ddcd04a.FlowIDTableStaking.RewardsPaid", [["nodeID", 137(1)], ["amount", 137(23)]]])], [136(h''), ["e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb", 174549955740]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.8624b52f9ddcd04a.FlowIDTableStaking.RewardsPaid"
				// 2 field: [["nodeID", type(string)], ["amount", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 49 bytes follow
				0x78, 0x31,
				// "A.8624b52f9ddcd04a.FlowIDTableStaking.RewardsPaid"
				0x41, 0x2e, 0x38, 0x36, 0x32, 0x34, 0x62, 0x35, 0x32, 0x66, 0x39, 0x64, 0x64, 0x63, 0x64, 0x30, 0x34, 0x61, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x49, 0x44, 0x54, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x74, 0x61, 0x6b, 0x69, 0x6e, 0x67, 0x2e, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64, 0x73, 0x50, 0x61, 0x69, 0x64,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "nodeID"
				0x6e, 0x6f, 0x64, 0x65, 0x49, 0x44,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// field 1
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "amount"
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 2 items follow
				0x82,
				// string, 64 bytes follow
				0x78, 0x40,
				// "e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb"
				0x65, 0x35, 0x32, 0x63, 0x62, 0x63, 0x64, 0x38, 0x32, 0x35, 0x65, 0x33, 0x32, 0x38, 0x61, 0x63, 0x61, 0x63, 0x38, 0x64, 0x62, 0x36, 0x62, 0x63, 0x62, 0x64, 0x63, 0x62, 0x62, 0x36, 0x65, 0x37, 0x37, 0x32, 0x34, 0x38, 0x36, 0x32, 0x63, 0x38, 0x62, 0x38, 0x39, 0x62, 0x30, 0x39, 0x64, 0x38, 0x35, 0x65, 0x64, 0x63, 0x63, 0x66, 0x34, 0x31, 0x66, 0x66, 0x39, 0x39, 0x38, 0x31, 0x65, 0x62,
				// 174549955740
				0x1b, 0x00, 0x00, 0x00, 0x28, 0xa3, 0xfc, 0xf4, 0x9c,
			},
		},
		{
			name:  "FlowToken.TokensDeposited with nil receiver",
			event: createFlowTokenTokensDepositedEventNoReceiver(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.1654653399040a61.FlowToken.TokensDeposited","fields":[{"value":{"value":"1316489.95887105","type":"UFix64"},"name":"amount"},{"value":{"value":null,"type":"Optional"},"name":"to"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.1654653399040a61.FlowToken.TokensDeposited", [["amount", 137(23)], ["to", 138(137(3))]]])], [136(h''), [131648995887105, null]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.1654653399040a61.FlowToken.TokensDeposited"
				// 2 field: [["to", type(optional(address))], ["amount", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 44 bytes follow
				0x78, 0x2c,
				// "A.1654653399040a61.FlowToken.TokensDeposited"
				0x41, 0x2e, 0x31, 0x36, 0x35, 0x34, 0x36, 0x35, 0x33, 0x33, 0x39, 0x39, 0x30, 0x34, 0x30, 0x61, 0x36, 0x31, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x73, 0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x65, 0x64,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "amount"
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 1
				// array, 2 element follows
				0x82,
				// text, 2 bytes follow
				0x62,
				// "to"
				0x74, 0x6f,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Address type ID (3)
				0x03,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 2 items follow
				0x82,
				// 131648995887105
				0x1b, 0x00, 0x00, 0x77, 0xbb, 0xeb, 0xa2, 0x88, 0x01,
				// null
				0xf6,
			},
		},
		{
			name:  "FlowToken.TokensDeposited",
			event: createFlowTokenTokensDepositedEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.1654653399040a61.FlowToken.TokensDeposited","fields":[{"value":{"value":"1745.49955740","type":"UFix64"},"name":"amount"},{"value":{"value":{"value":"0x8624b52f9ddcd04a","type":"Address"},"type":"Optional"},"name":"to"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.1654653399040a61.FlowToken.TokensDeposited", [["amount", 137(23)], ["to", 138(137(3))]]])], [136(h''), [174549955740, h'8624B52F9DDCD04A']]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.1654653399040a61.FlowToken.TokensDeposited"
				// 2 field: [["to", type(optional(address))], ["amount", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 44 bytes follow
				0x78, 0x2c,
				// "A.1654653399040a61.FlowToken.TokensDeposited"
				0x41, 0x2e, 0x31, 0x36, 0x35, 0x34, 0x36, 0x35, 0x33, 0x33, 0x39, 0x39, 0x30, 0x34, 0x30, 0x61, 0x36, 0x31, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x73, 0x44, 0x65, 0x70, 0x6f, 0x73, 0x69, 0x74, 0x65, 0x64,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "amount"
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 1
				// array, 2 element follows
				0x82,
				// text, 2 bytes follow
				0x62,
				// "to"
				0x74, 0x6f,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Address type ID (3)
				0x03,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 2 items follow
				0x82,
				// 174549955740
				0x1b, 0x00, 0x00, 0x00, 0x28, 0xa3, 0xfc, 0xf4, 0x9c,
				// bytes, 8 bytes follow
				0x48,
				// 0x8624b52f9ddcd04a
				0x86, 0x24, 0xb5, 0x2f, 0x9d, 0xdc, 0xd0, 0x4a,
			},
		},
		{
			name:  "FlowToken.TokensMinted",
			event: createFlowTokenTokensMintedEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.1654653399040a61.FlowToken.TokensMinted","fields":[{"value":{"value":"1316489.95887105","type":"UFix64"},"name":"amount"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.1654653399040a61.FlowToken.TokensMinted", [["amount", 137(23)]]])], [136(h''), [131648995887105]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.1654653399040a61.FlowToken.TokensMinted"
				// 1 field: [["amount", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 41 bytes follow
				0x78, 0x29,
				// "A.1654653399040a61.FlowToken.TokensMinted"
				0x41, 0x2e, 0x31, 0x36, 0x35, 0x34, 0x36, 0x35, 0x33, 0x33, 0x39, 0x39, 0x30, 0x34, 0x30, 0x61, 0x36, 0x31, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x73, 0x4d, 0x69, 0x6e, 0x74, 0x65, 0x64,
				// fields
				// array, 1 items follow
				0x81,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "amount"
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 1 items follow
				0x81,
				// 131648995887105
				0x1b, 0x00, 0x00, 0x77, 0xbb, 0xeb, 0xa2, 0x88, 0x01,
			},
		},
		{
			name:  "FlowToken.TokensWithdrawn",
			event: createFlowTokenTokensWithdrawnEvent(),
			expectedCBOR: []byte{
				// language=json, format=json-cdc
				// {"value":{"id":"A.1654653399040a61.FlowToken.TokensWithdrawn","fields":[{"value":{"value":"53.04112895","type":"UFix64"},"name":"amount"},{"value":{"value":{"value":"0xf919ee77447b7497","type":"Address"},"type":"Optional"},"name":"from"}]},"type":"Event"}
				//
				// language=edn, format=ccf
				// 129([[162([h'', "A.1654653399040a61.FlowToken.TokensWithdrawn", [["amount", 137(23)], ["from", 138(137(3))]]])], [136(h''), [5304112895, h'F919EE77447B7497']]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 element follows
				0x82,
				// element 0: type definitions
				// array, 1 element follows
				0x81,
				// event type:
				// id: []byte{}
				// cadence-type-id: "A.1654653399040a61.FlowToken.TokensWithdrawn"
				// 2 field: [["from", type(optional(address))], ["amount", type(ufix64)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 element follows
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 44 bytes follow
				0x78, 0x2c,
				// "A.1654653399040a61.FlowToken.TokensWithdrawn"
				0x41, 0x2e, 0x31, 0x36, 0x35, 0x34, 0x36, 0x35, 0x33, 0x33, 0x39, 0x39, 0x30, 0x34, 0x30, 0x61, 0x36, 0x31, 0x2e, 0x46, 0x6c, 0x6f, 0x77, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x2e, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x73, 0x57, 0x69, 0x74, 0x68, 0x64, 0x72, 0x61, 0x77, 0x6e,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 element follows
				0x82,
				// text, 6 bytes follow
				0x66,
				// "amount"
				0x61, 0x6d, 0x6f, 0x75, 0x6e, 0x74,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// UFix64 type ID (23)
				0x17,
				// field 1
				// array, 2 element follows
				0x82,
				// text, 4 bytes follow
				0x64,
				// "from"
				0x66, 0x72, 0x6f, 0x6d,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Address type ID (3)
				0x03,

				// element 1: type and value
				// array, 2 element follows
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 2 items follow
				0x82,
				// 5304112895
				0x1b, 0x00, 0x00, 0x00, 0x01, 0x3c, 0x26, 0x56, 0xff,
				// bytes, 8 bytes follow
				0x48,
				// 0xf919ee77447b7497
				0xf9, 0x19, 0xee, 0x77, 0x44, 0x7b, 0x74, 0x97,
			},
		},
	}

	test := func(tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Encode Cadence value to CCF
			actualCBOR, err := ccf.EventsEncMode.Encode(tc.event)
			require.NoError(t, err)
			utils.AssertEqualWithDiff(t, tc.expectedCBOR, actualCBOR)

			// Decode CCF to Cadence value
			decodedEvent, err := ccf.EventsDecMode.Decode(nil, actualCBOR)
			require.NoError(t, err)

			// Since event encoding doesn't sort fields, make sure that input event is identical to decoded event.
			require.Equal(
				t,
				cadence.ValueWithCachedTypeID(tc.event),
				cadence.ValueWithCachedTypeID(decodedEvent),
			)
		})
	}

	for _, tc := range testCases {
		test(tc)
	}
}

func newFlowFeesFeesDeductedEventType() *cadence.EventType {
	// access(all) event FeesDeducted(amount: UFix64, inclusionEffort: UFix64, executionEffort: UFix64)

	address, _ := common.HexToAddress("f919ee77447b7497")
	location := common.NewAddressLocation(nil, address, "FlowFees")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowFees.FeesDeducted",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UFix64Type{},
			},
			{
				Identifier: "inclusionEffort",
				Type:       cadence.UFix64Type{},
			},
			{
				Identifier: "executionEffort",
				Type:       cadence.UFix64Type{},
			},
		},
	}
}

func createFlowFeesFeesDeductedEvent() cadence.Event {
	/*
		A.f919ee77447b7497.FlowFees.FeesDeducted
		{
			"amount": "0.01797293",
			"inclusionEffort": "1.00000000",
			"executionEffort": "0.00360123"
		}
	*/
	amount, _ := cadence.NewUFix64("0.01797293")
	inclusionEffort, _ := cadence.NewUFix64("1.00000000")
	executionEffort, _ := cadence.NewUFix64("0.00360123")

	return cadence.NewEvent(
		[]cadence.Value{amount, inclusionEffort, executionEffort},
	).WithType(newFlowFeesFeesDeductedEventType())
}

func newFlowFeesTokensWithdrawnEventType() *cadence.EventType {
	// access(all) event TokensWithdrawn(amount: UFix64)

	address, _ := common.HexToAddress("f919ee77447b7497")
	location := common.NewAddressLocation(nil, address, "FlowFees")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowFees.TokensWithdrawn",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UFix64Type{},
			},
		},
	}
}

func createFlowFeesTokensWithdrawnEvent() cadence.Event {
	/*
		A.f919ee77447b7497.FlowFees.TokensWithdrawn
		{
			"amount": "53.04112895"
		}
	*/
	amount, _ := cadence.NewUFix64("53.04112895")

	return cadence.NewEvent(
		[]cadence.Value{amount},
	).WithType(newFlowFeesTokensWithdrawnEventType())
}

func newFlowTokenTokensDepositedEventType() *cadence.EventType {
	// access(all) event TokensDeposited(amount: UFix64, to: Address?)

	address, _ := common.HexToAddress("1654653399040a61")
	location := common.NewAddressLocation(nil, address, "FlowToken")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowToken.TokensDeposited",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UFix64Type{},
			},
			{
				Identifier: "to",
				Type: &cadence.OptionalType{
					Type: cadence.NewAddressType(),
				},
			},
		},
	}
}

func createFlowTokenTokensDepositedEventNoReceiver() cadence.Event {
	/*
		A.1654653399040a61.FlowToken.TokensDeposited
		{
			"amount": "1316489.95887105",
			"to": null
		}
	*/
	amount, _ := cadence.NewUFix64("1316489.95887105")
	to := cadence.NewOptional(nil)

	return cadence.NewEvent(
		[]cadence.Value{amount, to},
	).WithType(newFlowTokenTokensDepositedEventType())
}

func createFlowTokenTokensDepositedEvent() cadence.Event {
	/*
		A.1654653399040a61.FlowToken.TokensDeposited
		{
			"amount": "1745.49955740",
			"to": "0x8624b52f9ddcd04a"
		}
	*/
	addressBytes, _ := hex.DecodeString("8624b52f9ddcd04a")

	amount, _ := cadence.NewUFix64("1745.49955740")
	to := cadence.NewOptional(cadence.BytesToAddress(addressBytes))

	return cadence.NewEvent(
		[]cadence.Value{amount, to},
	).WithType(newFlowTokenTokensDepositedEventType())
}

func newFlowTokenTokensMintedEventType() *cadence.EventType {
	// access(all) event TokensMinted(amount: UFix64)

	address, _ := common.HexToAddress("1654653399040a61")
	location := common.NewAddressLocation(nil, address, "FlowToken")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowToken.TokensMinted",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UFix64Type{},
			},
		},
	}
}

func createFlowTokenTokensMintedEvent() cadence.Event {
	/*
		A.1654653399040a61.FlowToken.TokensMinted
		{
			"amount": "1316489.95887105"
		}
	*/
	amount, _ := cadence.NewUFix64("1316489.95887105")

	return cadence.NewEvent(
		[]cadence.Value{amount},
	).WithType(newFlowTokenTokensMintedEventType())
}

func newFlowTokenTokensWithdrawnEventType() *cadence.EventType {
	// access(all) event TokensWithdrawn(amount: UFix64, from: Address?)

	address, _ := common.HexToAddress("1654653399040a61")
	location := common.NewAddressLocation(nil, address, "FlowToken")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowToken.TokensWithdrawn",
		Fields: []cadence.Field{
			{
				Identifier: "amount",
				Type:       cadence.UFix64Type{},
			},
			{
				Identifier: "from",
				Type: &cadence.OptionalType{
					Type: cadence.NewAddressType(),
				},
			},
		},
	}
}

func createFlowTokenTokensWithdrawnEvent() cadence.Event {
	/*
		A.1654653399040a61.FlowToken.TokensWithdrawn
		{
			"amount": "53.04112895",
			"from": "0xf919ee77447b7497"
		}
	*/
	addressBytes, _ := hex.DecodeString("f919ee77447b7497")

	amount, _ := cadence.NewUFix64("53.04112895")
	from := cadence.NewOptional(cadence.BytesToAddress(addressBytes))

	return cadence.NewEvent(
		[]cadence.Value{amount, from},
	).WithType(newFlowTokenTokensWithdrawnEventType())
}

func newFlowIDTableStakingDelegatorRewardsPaidEventType() *cadence.EventType {
	// access(all) event DelegatorRewardsPaid(nodeID: String, delegatorID: UInt32, amount: UFix64)

	address, _ := common.HexToAddress("8624b52f9ddcd04a")
	location := common.NewAddressLocation(nil, address, "FlowIDTableStaking")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowIDTableStaking.DelegatorRewardsPaid",
		Fields: []cadence.Field{
			{
				Identifier: "nodeID",
				Type:       cadence.StringType{},
			},
			{
				Identifier: "delegatorID",
				Type:       cadence.UInt32Type{},
			},
			{
				Identifier: "amount",
				Type:       cadence.UFix64Type{},
			},
		},
	}
}

func createFlowIDTableStakingDelegatorRewardsPaidEvent() cadence.Event {
	/*
		A.8624b52f9ddcd04a.FlowIDTableStaking.DelegatorRewardsPaid
		{
			"nodeID": "e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb",
			"delegatorID": 92,
			"amount": "4.38760261"
		}
	*/
	nodeID := cadence.String("e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb")
	delegatorID := cadence.UInt32(92)
	amount, _ := cadence.NewUFix64("4.38760261")

	return cadence.NewEvent(
		[]cadence.Value{nodeID, delegatorID, amount},
	).WithType(newFlowIDTableStakingDelegatorRewardsPaidEventType())
}

func newFlowIDTableStakingEpochTotalRewardsPaidEventType() *cadence.EventType {
	// access(all) event EpochTotalRewardsPaid(total: UFix64, fromFees: UFix64, minted: UFix64, feesBurned: UFix64)

	address, _ := common.HexToAddress("8624b52f9ddcd04a")
	location := common.NewAddressLocation(nil, address, "FlowIDTableStaking")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowIDTableStaking.EpochTotalRewardsPaid",
		Fields: []cadence.Field{
			{
				Identifier: "total",
				Type:       cadence.UFix64Type{},
			},
			{
				Identifier: "fromFees",
				Type:       cadence.UFix64Type{},
			},
			{
				Identifier: "minted",
				Type:       cadence.UFix64Type{},
			},
			{
				Identifier: "feesBurned",
				Type:       cadence.UFix64Type{},
			},
		},
	}
}

func createFlowIDTableStakingEpochTotalRewardsPaidEvent() cadence.Event {
	/*
		A.8624b52f9ddcd04a.FlowIDTableStaking.EpochTotalRewardsPaid
		{
			"total": "1316543.00000000",
			"fromFees": "53.04112895",
			"minted": "1316489.95887105",
			"feesBurned": "6.04080767"
		}
	*/
	total, _ := cadence.NewUFix64("1316543.00000000")
	fromFees, _ := cadence.NewUFix64("53.04112895")
	minted, _ := cadence.NewUFix64("1316489.95887105")
	feesBurned, _ := cadence.NewUFix64("6.04080767")

	return cadence.NewEvent(
		[]cadence.Value{total, fromFees, minted, feesBurned},
	).WithType(newFlowIDTableStakingEpochTotalRewardsPaidEventType())
}

func newFlowIDTableStakingNewWeeklyPayoutEventType() *cadence.EventType {
	// access(all) event NewWeeklyPayout(newPayout: UFix64)

	address, _ := common.HexToAddress("8624b52f9ddcd04a")
	location := common.NewAddressLocation(nil, address, "FlowIDTableStaking")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowIDTableStaking.NewWeeklyPayout",
		Fields: []cadence.Field{
			{
				Identifier: "newPayout",
				Type:       cadence.UFix64Type{},
			},
		},
	}
}

func createFlowIDTableStakingNewWeeklyPayoutEvent() cadence.Event {
	/*
		A.8624b52f9ddcd04a.FlowIDTableStaking.NewWeeklyPayout
		{
			"newPayout": "1317778.00000000"
		}
	*/
	newPayout, _ := cadence.NewUFix64("1317778.00000000")

	return cadence.NewEvent(
		[]cadence.Value{newPayout},
	).WithType(newFlowIDTableStakingNewWeeklyPayoutEventType())
}

func newFlowIDTableStakingRewardsPaidEventType() *cadence.EventType {
	// access(all) event RewardsPaid(nodeID: String, amount: UFix64)

	address, _ := common.HexToAddress("8624b52f9ddcd04a")
	location := common.NewAddressLocation(nil, address, "FlowIDTableStaking")

	return &cadence.EventType{
		Location:            location,
		QualifiedIdentifier: "FlowIDTableStaking.RewardsPaid",
		Fields: []cadence.Field{
			{
				Identifier: "nodeID",
				Type:       cadence.StringType{},
			},
			{
				Identifier: "amount",
				Type:       cadence.UFix64Type{},
			},
		},
	}
}

func createFlowIDTableStakingRewardsPaidEvent() cadence.Event {
	nodeID, _ := cadence.NewString("e52cbcd825e328acac8db6bcbdcbb6e7724862c8b89b09d85edccf41ff9981eb")
	amount, _ := cadence.NewUFix64("1745.49955740")

	return cadence.NewEvent(
		[]cadence.Value{nodeID, amount},
	).WithType(newFlowIDTableStakingRewardsPaidEventType())
}

func TestDecodeTruncatedData(t *testing.T) {
	t.Parallel()

	data, err := deterministicEncMode.Encode(createFlowTokenTokensWithdrawnEvent())
	require.NoError(t, err)

	_, err = deterministicDecMode.Decode(nil, data)
	require.NoError(t, err)

	for i := len(data) - 1; i >= 0; i-- {
		decodedVal, err := deterministicDecMode.Decode(nil, data[:i])
		require.Nil(t, decodedVal)
		require.Error(t, err)
	}
}

func TestDecodeInvalidData(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		data []byte
	}

	testCases := []testCase{
		{
			name: "nil",
			data: nil,
		},
		{
			name: "empty",
			data: []byte{},
		},
		{
			name: "malformed CBOR data for potential OOM",
			data: []byte{0x9b, 0x00, 0x00, 0x42, 0xfa, 0x42, 0xfa, 0x42, 0xfa, 0x42},
		},
		{
			name: "mismatched type and value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(1), true])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// true
				0xf5,
			},
		},
		{
			name: "not found type definition",
			data: []byte{
				// language=edn, format=ccf
				// 130([136(h''), [1]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 1 items follow
				0x81,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
			},
		},
		{
			name: "unreferenced type definition",
			data: []byte{
				// language=edn, format=ccf
				// 129([[162([h'', "S.test.FooEvent", [["a", 137(4)], ["b", 137(1)]]]), 160([h'1', "S.test.FooStruct", [["a", 137(4)], ["b", 137(1)]]])], [136(h''), [1, "foo"]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 2 items follow
				0x82,
				// event type:
				// id: []byte{}
				// cadence-type-id: "S.test.FooEvent"
				// 2 fields: [["a", type(int)], ["b", type(string)]]
				// tag
				0xd8, ccf.CBORTagEventType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 15 bytes follow
				0x6f,
				// S.test.FooEvent
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x45, 0x76, 0x65, 0x6e, 0x74,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.FooStruct"
				// 2 fields: [["a", type(int)], ["b", type(string)]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 1 byte follow
				0x41,
				// 1
				0x01,
				// cadence-type-id
				// string, 16 bytes follow
				0x70,
				// S.test.FooStruct
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 2 items follow
				0x82,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 1
				0x01,
				// String, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
			},
		},
		{
			name: "nil type",
			data: []byte{
				// language=edn, format=ccf
				// 130([null, true])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// nil
				0xf6,
				// true
				0xf5,
			},
		},
		{
			name: "nil type definitions",
			data: []byte{
				// language=edn, format=ccf
				// 129(null, [137(0), true])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// nil
				0xf6,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Bool type ID (0)
				0x00,
				// true
				0xf5,
			},
		},
		{
			name: "nil type definition",
			data: []byte{
				// language=edn, format=ccf
				// 129([null], [137(0), true])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// array, 1 items follow
				0x81,
				// nil
				0xf6,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Bool type ID (0)
				0x00,
				// true
				0xf5,
			},
		},
		{
			name: "nil optional inner type",
			data: []byte{
				// language=edn, format=ccf
				// 130([138(null), null])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagOptionalType,
				// nil
				0xf6,
				// nil
				0xf6,
			},
		},
		{
			name: "nil element type in constant sized array",
			data: []byte{
				// language=edn, format=ccf
				// 130([140[1, null], [1]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// type constant-sized [1]nil
				// tag
				0xd8, ccf.CBORTagConstsizedArrayType,
				// array, 2 items follow
				0x82,
				// number of elements
				0x01,
				// nil
				0xf6,
				// array data without inlined type definition
				// array, 1 items follow
				0x81,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
			},
		},
		{
			name: "nil element type in variable sized array",
			data: []byte{
				// language=edn, format=ccf
				// 130([139(null), [1]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// type []nil
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// null
				0xf6,
				// array data without inlined type definition
				// array, 1 items follow
				0x81,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
			},
		},
		{
			name: "nil key type in dictionary type",
			data: []byte{
				// language=edn, format=ccf
				// 130([141([nil, 137(4)]), ["a", 1]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// type (map[nil]int)
				// tag
				0xd8, ccf.CBORTagDictType,
				// array, 2 items follow
				0x82,
				// null
				0xf6,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Int type ID (4)
				0x04,
				// array data without inlined type definition
				// array, 2 items follow
				0x82,
				// string, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
			},
		},
		{
			name: "nil element type in dictionary type",
			data: []byte{
				// language=edn, format=ccf
				// 130([141([137(1), nil]), ["a", 1]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// type (map[int]nil)
				// tag
				0xd8, ccf.CBORTagDictType,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,
				// null
				0xf6,
				// array data without inlined type definition
				// array, 2 items follow
				0x82,
				// string, 1 bytes follow
				0x61,
				// a
				0x61,
				// tag (big num)
				0xc2,
				// bytes, 1 bytes follow
				0x41,
				// 1
				0x01,
			},
		},
		{
			name: "nil composite field type",
			data: []byte{
				// language=edn, format=ccf
				// 129([[160([h'', "S.test.FooStruct", [["a", nil], ["b", 137(1)]]])], [136(h''), [1, "foo"]]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeDefAndValue,
				// array, 2 items follow
				0x82,
				// element 0: type definitions
				// array, 1 items follow
				0x81,
				// struct type:
				// id: []byte{}
				// cadence-type-id: "S.test.FooStruct"
				// 2 fields: [["a", nil], ["b", type(string)]]
				// tag
				0xd8, ccf.CBORTagStructType,
				// array, 3 items follow
				0x83,
				// id
				// bytes, 0 bytes follow
				0x40,
				// cadence-type-id
				// string, 16 bytes follow
				0x70,
				// S.test.FooStruct
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
				// fields
				// array, 2 items follow
				0x82,
				// field 0
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// a
				0x61,
				// null
				0xf6,
				// field 1
				// array, 2 items follow
				0x82,
				// text, 1 bytes follow
				0x61,
				// b
				0x62,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// String type ID (1)
				0x01,

				// element 1: type and value
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagTypeRef,
				// bytes, 0 bytes follow
				0x40,
				// array, 2 items follow
				0x82,
				// tag (big number)
				0xc2,
				// bytes, 1 byte follow
				0x41,
				// 1
				0x01,
				// String, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
			},
		},
		{
			name: "nil inner type in optional type value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 186(null)])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagOptionalTypeValue,
				// null
				0xf6,
			},
		},
		{
			name: "nil element type in constant sized array type value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 188([3, null])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagConstsizedArrayTypeValue,
				// array, 2 elements follow
				0x82,
				// 3
				0x03,
				// null
				0xf6,
			},
		},
		{
			name: "nil element type in variable sized array type value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 187(null)])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayTypeValue,
				// null
				0xf6,
			},
		},
		{
			name: "nil key type in dictionary type value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 189([null, 185(1)])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagDictTypeValue,
				// array, 2 elements follow
				0x82,
				// null
				0xf6,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		},
		{
			name: "nil element type in dictionary type value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 189([185(4), null])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagDictTypeValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// null
				0xf6,
			},
		},
		{
			name: "nil field type in struct type value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 208([h'', "S.test.S", null, [["foo", null]], [[["foo", "bar", 185(4)]], [["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.So
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// null
				0xf6,
				// initializers
				// array, 2 elements follow
				0x82,
				// array, 1 element follows
				0x81,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// array, 1 element follows
				0x81,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		},
		{
			name: "nil initializer type in struct type value",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 208([h'', "S.test.S", null, [["foo", 185(4)]], [[["foo", "bar", null]], [["qux", "baz", 185(1)]]]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 elements follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 5 elements follow
				0x85,
				// bytes, 0 bytes follow
				0x40,
				// string, 8 bytes follow
				0x68,
				// S.test.So
				0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
				// type (nil for struct)
				0xf6,
				// fields
				// array, 1 element follows
				0x81,
				// array, 2 elements follow
				0x82,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// Int type (4)
				0x04,
				// initializers
				// array, 2 elements follow
				0x82,
				// array, 1 element follows
				0x81,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// foo
				0x66, 0x6f, 0x6f,
				// string, 3 bytes follow
				0x63,
				// bar
				0x62, 0x61, 0x72,
				// null
				0xf6,
				// array, 1 element follows
				0x81,
				// array, 3 elements follow
				0x83,
				// string, 3 bytes follow
				0x63,
				// qux
				0x71, 0x75, 0x78,
				// string, 3 bytes follow
				0x63,
				// bax
				0x62, 0x61, 0x7a,
				// tag
				0xd8, ccf.CBORTagSimpleTypeValue,
				// String type (1)
				0x01,
			},
		},
		{
			name: "null restriction in restricted type value",
			// Data is generated by fuzzer.
			data: []byte{
				// language=edn, format=ccf
				// 130([137(41), 191([208([h'', "S.\ufffd0000000.000000", null, [], []]), [null]])])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Meta type ID (41)
				0x18, 0x29,
				// tag
				0xd8, ccf.CBORTagRestrictedTypeValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagStructTypeValue,
				// array, 5 items follow
				0x85,
				// ccf type ID
				// bytes, 0 byte follows
				0x40,
				// cadence type ID
				// text, 19 bytes follow
				0x73,
				// "S.�0000000.000000"
				0x53, 0x2e, 0xef, 0xbf, 0xbd, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x2e, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30,
				// type
				// nil
				0xf6,
				// fields
				// array, 0 item follows
				0x80,
				// initializers
				// array, 0 item follows
				0x80,
				// restrictions
				// array, 1 item follows
				0x81,
				// nil
				0xf6,
			},
		},
		{
			name: "extraneous data",
			data: []byte{
				// language=edn, format=ccf
				// 130([137(0), true]), 0
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// Bool type ID (0)
				0x00,
				// true
				0xf5,
				// extraneous data
				0x00,
			},
		},
	}

	test := func(tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			decodedVal, err := deterministicDecMode.Decode(nil, tc.data)
			require.Nil(t, decodedVal)
			require.Error(t, err)
		})
	}
	for _, tc := range testCases {
		test(tc)
	}
}

func TestEncodeValueOfRestrictedInterface(t *testing.T) {

	t.Parallel()

	// Values and types are generated by fuzzer.
	/*
		// Type def

		struct OuterStruct {
		    var field: MiddleStruct
		}

		struct MiddleStruct {
		    var field: AnyStruct{Interface}
		}

		struct interface Interface {}

		struct InnerStruct: Interface {}     // 'InnerStruct' conforms to 'Interface'

		// Value

		OuterStruct {
		    field: MiddleStruct {
		        field: InnerStruct{}   // <-- here the value is the implementation, for the restricted type.
		    }
		}
	*/

	interfaceType := cadence.NewStructInterfaceType(
		common.StringLocation("LocationA"),
		"Interface",
		nil,
		nil,
	)

	middleStruct := cadence.NewStructType(
		common.StringLocation("LocationB"),
		"MiddleStruct",
		[]cadence.Field{
			{
				Type: cadence.NewRestrictedType(
					cadence.TheAnyStructType, []cadence.Type{interfaceType}),
				Identifier: "field",
			},
		},
		nil,
	)

	outerStructType := cadence.NewStructType(
		common.StringLocation("LocationC"),
		"OuterStruct",
		[]cadence.Field{
			{
				Type:       middleStruct,
				Identifier: "field",
			},
		},
		nil,
	)

	innerStructType := cadence.NewStructType(
		common.StringLocation("LocationD"),
		"InnerStruct",
		[]cadence.Field{},
		nil,
	)

	value := cadence.NewStruct([]cadence.Value{
		cadence.NewStruct([]cadence.Value{
			cadence.NewStruct([]cadence.Value{}).WithType(innerStructType),
		}).WithType(middleStruct),
	}).WithType(outerStructType)

	testEncodeAndDecode(
		t,
		value,
		[]byte{
			// language=json, format=json-cdc
			// {"value":{"id":"S.LocationC.OuterStruct","fields":[{"value":{"value":{"id":"S.LocationB.MiddleStruct","fields":[{"value":{"value":{"id":"S.LocationD.InnerStruct","fields":[]},"type":"Struct"},"name":"field"}]},"type":"Struct"},"name":"field"}]},"type":"Struct"}
			//
			// language=edn, format=ccf
			// 129([[176([h'', "S.LocationA.Interface"]), 160([h'01', "S.LocationC.OuterStruct", [["field", 136(h'03')]]]), 160([h'02', "S.LocationD.InnerStruct", []]), 160([h'03', "S.LocationB.MiddleStruct", [["field", 143([137(39), [136(h'')]])]]])], [136(h'01'), [[130([136(h'02'), []])]]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// array, 4 items follow
			0x84,
			// tag
			0xd8, ccf.CBORTagStructInterfaceType,
			// array, 2 items follow
			0x82,
			// CCF type ID
			// bytes, 0 byte follows
			0x40,
			// cadence type ID
			// text, 21 bytes follow
			0x75,
			// "S.LocationA.Interface"
			0x53, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x41, 0x2e, 0x49, 0x6e, 0x74, 0x65, 0x72, 0x66, 0x61, 0x63, 0x65,
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 items follow
			0x83,
			// CCF type ID
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
			// text, 23 bytes follow
			0x77,
			// "S.LocationC.OuterStruct"
			0x53, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x2e, 0x4f, 0x75, 0x74, 0x65, 0x72, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 1 item follows
			0x81,
			// array, 2 item follows
			0x82,
			// text, 5 bytes follow
			0x65,
			// "field"
			0x66, 0x69, 0x65, 0x6c, 0x64,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follow
			0x41,
			// 3
			0x03,
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 item follows
			0x83,
			// CCF type ID
			// bytes, 1 byte follow
			0x41,
			// 2
			0x02,
			// Cadence type ID
			// text, 23 bytes follow
			0x77,
			// "S.LocationD.InnerStruct"
			0x53, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x2e, 0x49, 0x6e, 0x6e, 0x65, 0x72, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 0 item follows
			0x80,
			// tag
			0xd8, ccf.CBORTagStructType,
			// array, 3 item follows
			0x83,
			// CCF type ID
			// bytes, 1 byte follow
			0x41,
			// 3
			0x03,
			// Cadence type ID
			// text, 24 bytes follow
			0x78, 0x18,
			// "S.LocationB.MiddleStruct"
			0x53, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x42, 0x2e, 0x4d, 0x69, 0x64, 0x64, 0x6c, 0x65, 0x53, 0x74, 0x72, 0x75, 0x63, 0x74,
			// fields
			// array, 1 item follows
			0x81,
			// array, 2 item follows
			0x82,
			// text, 5 bytes follow
			0x65,
			// "field"
			0x66, 0x69, 0x65, 0x6c, 0x64,
			// tag
			0xd8, ccf.CBORTagRestrictedType,
			// array, 2 item follows
			0x82,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// AnyStruct type ID (39)
			0x18, 0x27,
			// array, 1 item follows
			0x81,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follows
			0x40,
			// array, 2 item follows
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
			// array, 1 item follows
			0x81,
			// array, 1 item follows
			0x81,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 item follows
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,
			// array, 0 item follows
			0x80,
		},
	)
}

func TestCyclicReferenceValue(t *testing.T) {

	// Test data is from TestRuntimeScriptReturnSpecial in runtime_test.go
	t.Run("recursive reference", func(t *testing.T) {

		t.Parallel()

		script := `
			access(all) fun main(): AnyStruct {
				let refs: [&AnyStruct] = []
				refs.append(&refs as &AnyStruct)
				return refs
			}
        `

		actual := exportFromScript(t, script)

		expected := cadence.NewArray([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				nil,
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: &cadence.ReferenceType{
					Authorization: cadence.Unauthorized{},
					Type:          cadence.AnyStructType{},
				},
			}),
		}).WithType(&cadence.VariableSizedArrayType{
			ElementType: &cadence.ReferenceType{
				Authorization: cadence.Unauthorized{},
				Type:          cadence.AnyStructType{},
			},
		})

		assert.Equal(t, expected, actual)

		testEncodeAndDecode(
			t,
			expected,
			[]byte{
				// language=json, format=json-cdc
				// {"value":[{"value":[null],"type":"Array"}],"type":"Array"}
				//
				// language=edn, format=ccf
				// 130([139(142([false, 137(39)])), [130([139(142([false, 137(39)])), [null]])]])
				//
				// language=cbor, format=ccf
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// static type
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagReferenceType,
				// array, 2 items follow
				0x82,
				// nil
				0xf6,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,

				// data
				// array, 1 items follow
				0x81,
				// tag
				0xd8, ccf.CBORTagTypeAndValue,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, ccf.CBORTagVarsizedArrayType,
				// tag
				0xd8, ccf.CBORTagReferenceType,
				// array, 2 items follow
				0x82,
				// nil
				0xf6,
				// tag
				0xd8, ccf.CBORTagSimpleType,
				// AnyStruct type ID (39)
				0x18, 0x27,
				// array, 1 items follow
				0x81,
				// nil
				0xf6,
			},
		)
	})
}

func TestSortOptions(t *testing.T) {
	// Test sorting of:
	// - composite fields ("count", "sum")
	// - restricted types ("HasCount", "HasSum")

	sortFieldsEncMode, err := ccf.EncOptions{
		SortCompositeFields: ccf.SortBytewiseLexical,
	}.EncMode()
	require.NoError(t, err)

	sortRestrictedTypesEncMode, err := ccf.EncOptions{
		SortRestrictedTypes: ccf.SortBytewiseLexical,
	}.EncMode()
	require.NoError(t, err)

	enforceSortedFieldsDecMode, err := ccf.DecOptions{
		EnforceSortCompositeFields: ccf.EnforceSortBytewiseLexical,
	}.DecMode()
	require.NoError(t, err)

	enforceSortedRestrictedTypesDecMode, err := ccf.DecOptions{
		EnforceSortRestrictedTypes: ccf.EnforceSortBytewiseLexical,
	}.DecMode()
	require.NoError(t, err)

	hasCountInterfaceType := cadence.NewResourceInterfaceType(
		common.NewStringLocation(nil, "test"),
		"HasCount",
		nil,
		nil,
	)

	hasSumInterfaceType := cadence.NewResourceInterfaceType(
		common.NewStringLocation(nil, "test"),
		"HasSum",
		nil,
		nil,
	)

	statsType := cadence.NewResourceType(
		common.NewStringLocation(nil, "test"),
		"Stats",
		[]cadence.Field{
			cadence.NewField("count", cadence.NewIntType()),
			cadence.NewField("sum", cadence.NewIntType()),
		},
		nil,
	)

	countSumRestrictedType := cadence.NewRestrictedType(
		nil,
		[]cadence.Type{
			hasCountInterfaceType,
			hasSumInterfaceType,
		},
	)

	val := cadence.NewArray([]cadence.Value{
		cadence.NewResource(
			[]cadence.Value{
				cadence.NewInt(1),
				cadence.NewInt(2),
			},
		).WithType(statsType),
	}).WithType(cadence.NewVariableSizedArrayType(countSumRestrictedType))

	t.Run("no sort", func(t *testing.T) {
		expectedStatsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("count", cadence.NewIntType()),
				cadence.NewField("sum", cadence.NewIntType()),
			},
			nil,
		)

		expectedCountSumRestrictedType := cadence.NewRestrictedType(
			nil,
			[]cadence.Type{
				hasCountInterfaceType,
				hasSumInterfaceType,
			},
		)

		expectedVal := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.NewInt(2),
				},
			).WithType(expectedStatsType),
		}).WithType(cadence.NewVariableSizedArrayType(expectedCountSumRestrictedType))

		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.Stats","fields":[{"value":{"value":"1","type":"Int"},"name":"count"},{"value":{"value":"2","type":"Int"},"name":"sum"}]},"type":"Resource"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Stats", [["count", 137(4)], ["sum", 137(4)]]]), 177([h'01', "S.test.HasSum"]), 177([h'02', "S.test.HasCount"]), ], [139(143([null, [136(h'02'), 136(h'01')]])), [130([136(h''), [2, 1]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 3 items follow
			0x83,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Stats"
			// 2 fields: [["count", type(int)], ["sum", type(int)]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 12 bytes follow
			0x6c,
			// S.test.Stats
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x73,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 5 bytes follow
			0x65,
			// count
			0x63, 0x6f, 0x75, 0x6e, 0x74,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// sum
			0x73, 0x75, 0x6d,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// resource interface type:
			// id: []byte{1}
			// cadence-type-id: "S.test.HasSum"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 13 bytes follow
			0x6d,
			// S.test.HasSum
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x53, 0x75, 0x6d,
			// resource interface type:
			// id: []byte{2}
			// cadence-type-id: "S.test.HasCount"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// cadence-type-id
			// string, 15 bytes follow
			0x6f,
			// S.test.HasCount
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagRestrictedType,
			// array, 2 items follow
			0x82,
			// type
			// null
			0xf6,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,

			// array, 1 item follows
			0x81,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follows
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,
		}

		// Encode value without sorting.
		actualCBOR, err := ccf.Encode(val)
		require.NoError(t, err)
		utils.AssertEqualWithDiff(t, expectedCBOR, actualCBOR)

		// Decode value without enforcing sorting.
		decodedVal, err := ccf.Decode(nil, actualCBOR)
		require.NoError(t, err)
		assert.Equal(
			t,
			cadence.ValueWithCachedTypeID(expectedVal),
			cadence.ValueWithCachedTypeID(decodedVal),
		)

		// Decode value enforcing sorting of composite fields should return error.
		_, err = enforceSortedFieldsDecMode.Decode(nil, actualCBOR)
		require.Error(t, err)

		// Decode value enforcing sorting of restricted types should return error.
		_, err = enforceSortedRestrictedTypesDecMode.Decode(nil, actualCBOR)
		require.Error(t, err)
	})

	t.Run("sort composite fields only", func(t *testing.T) {
		expectedStatsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("sum", cadence.NewIntType()),
				cadence.NewField("count", cadence.NewIntType()),
			},
			nil,
		)

		expectedCountSumRestrictedType := cadence.NewRestrictedType(
			nil,
			[]cadence.Type{
				hasCountInterfaceType,
				hasSumInterfaceType,
			},
		)

		expectedVal := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(2),
					cadence.NewInt(1),
				},
			).WithType(expectedStatsType),
		}).WithType(cadence.NewVariableSizedArrayType(expectedCountSumRestrictedType))

		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.Stats","fields":[{"value":{"value":"1","type":"Int"},"name":"count"},{"value":{"value":"2","type":"Int"},"name":"sum"}]},"type":"Resource"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Stats", [["sum", 137(4)], ["count", 137(4)]]]), 177([h'01', "S.test.HasSum"]), 177([h'02', "S.test.HasCount"]), ], [139(143([null, [136(h'02'), 136(h'01')]])), [130([136(h''), [2, 1]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 3 items follow
			0x83,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Stats"
			// 2 fields: [["sum", type(int)], ["count", type(int)]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 12 bytes follow
			0x6c,
			// S.test.Stats
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x73,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// sum
			0x73, 0x75, 0x6d,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 5 bytes follow
			0x65,
			// count
			0x63, 0x6f, 0x75, 0x6e, 0x74,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// resource interface type:
			// id: []byte{1}
			// cadence-type-id: "S.test.HasSum"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 13 bytes follow
			0x6d,
			// S.test.HasSum
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x53, 0x75, 0x6d,
			// resource interface type:
			// id: []byte{2}
			// cadence-type-id: "S.test.HasCount"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// cadence-type-id
			// string, 15 bytes follow
			0x6f,
			// S.test.HasCount
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagRestrictedType,
			// array, 2 items follow
			0x82,
			// type
			// null
			0xf6,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,

			// array, 1 item follows
			0x81,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follows
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
		}

		// Encode value with sorted composite fields.
		actualCBOR, err := sortFieldsEncMode.Encode(val)
		require.NoError(t, err)
		utils.AssertEqualWithDiff(t, expectedCBOR, actualCBOR)

		// Decode value enforcing sorting of composite fields.
		decodedVal, err := enforceSortedFieldsDecMode.Decode(nil, actualCBOR)
		require.NoError(t, err)
		assert.Equal(
			t,
			cadence.ValueWithCachedTypeID(expectedVal),
			cadence.ValueWithCachedTypeID(decodedVal),
		)

		// Decode value without enforcing sorting should return no error.
		_, err = ccf.Decode(nil, actualCBOR)
		require.NoError(t, err)

		// Decode value enforcing sorting of restricted types should return error.
		_, err = enforceSortedRestrictedTypesDecMode.Decode(nil, actualCBOR)
		require.Error(t, err)
	})

	t.Run("sort restricted types only", func(t *testing.T) {
		expectedStatsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("count", cadence.NewIntType()),
				cadence.NewField("sum", cadence.NewIntType()),
			},
			nil,
		)

		expectedCountSumRestrictedType := cadence.NewRestrictedType(
			nil,
			[]cadence.Type{
				hasSumInterfaceType,
				hasCountInterfaceType,
			},
		)

		expectedVal := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(1),
					cadence.NewInt(2),
				},
			).WithType(expectedStatsType),
		}).WithType(cadence.NewVariableSizedArrayType(expectedCountSumRestrictedType))

		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.Stats","fields":[{"value":{"value":"1","type":"Int"},"name":"count"},{"value":{"value":"2","type":"Int"},"name":"sum"}]},"type":"Resource"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Stats", [["count", 137(4)], ["sum", 137(4)]]]), 177([h'01', "S.test.HasSum"]), 177([h'02', "S.test.HasCount"]), ], [139(143([null, [136(h'01'), 136(h'02')]])), [130([136(h''), [2, 1]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 3 items follow
			0x83,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Stats"
			// 2 fields: [["count", type(int)], ["sum", type(int)]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 12 bytes follow
			0x6c,
			// S.test.Stats
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x73,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 5 bytes follow
			0x65,
			// count
			0x63, 0x6f, 0x75, 0x6e, 0x74,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// sum
			0x73, 0x75, 0x6d,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// resource interface type:
			// id: []byte{1}
			// cadence-type-id: "S.test.HasSum"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 13 bytes follow
			0x6d,
			// S.test.HasSum
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x53, 0x75, 0x6d,
			// resource interface type:
			// id: []byte{2}
			// cadence-type-id: "S.test.HasCount"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// cadence-type-id
			// string, 15 bytes follow
			0x6f,
			// S.test.HasCount
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagRestrictedType,
			// array, 2 items follow
			0x82,
			// type
			// null
			0xf6,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,

			// array, 1 item follows
			0x81,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follows
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,
		}

		// Encode value with sorted restricted types.
		actualCBOR, err := sortRestrictedTypesEncMode.Encode(val)
		require.NoError(t, err)
		utils.AssertEqualWithDiff(t, expectedCBOR, actualCBOR)

		// Decode value enforcing sorting of restricted types.
		decodedVal, err := enforceSortedRestrictedTypesDecMode.Decode(nil, actualCBOR)
		require.NoError(t, err)
		assert.Equal(
			t,
			cadence.ValueWithCachedTypeID(expectedVal),
			cadence.ValueWithCachedTypeID(decodedVal),
		)

		// Decode value without enforcing sorting should return no error.
		_, err = ccf.Decode(nil, actualCBOR)
		require.NoError(t, err)

		// Decode value enforcing sorting of composite fields should return error.
		_, err = enforceSortedFieldsDecMode.Decode(nil, actualCBOR)
		require.Error(t, err)
	})

	t.Run("sort", func(t *testing.T) {
		expectedStatsType := cadence.NewResourceType(
			common.NewStringLocation(nil, "test"),
			"Stats",
			[]cadence.Field{
				cadence.NewField("sum", cadence.NewIntType()),
				cadence.NewField("count", cadence.NewIntType()),
			},
			nil,
		)

		expectedCountSumRestrictedType := cadence.NewRestrictedType(
			nil,
			[]cadence.Type{
				hasSumInterfaceType,
				hasCountInterfaceType,
			},
		)

		expectedVal := cadence.NewArray([]cadence.Value{
			cadence.NewResource(
				[]cadence.Value{
					cadence.NewInt(2),
					cadence.NewInt(1),
				},
			).WithType(expectedStatsType),
		}).WithType(cadence.NewVariableSizedArrayType(expectedCountSumRestrictedType))

		expectedCBOR := []byte{
			// language=json, format=json-cdc
			// {"value":[{"value":{"id":"S.test.Stats","fields":[{"value":{"value":"1","type":"Int"},"name":"count"},{"value":{"value":"2","type":"Int"},"name":"sum"}]},"type":"Resource"}],"type":"Array"}
			//
			// language=edn, format=ccf
			// 129([[161([h'', "S.test.Stats", [["sum", 137(4)], ["count", 137(4)]]]), 177([h'01', "S.test.HasSum"]), 177([h'02', "S.test.HasCount"])], [139(143([null, [136(h'01'), 136(h'02')]])), [130([136(h''), [2, 1]])]]])
			//
			// language=cbor, format=ccf
			// tag
			0xd8, ccf.CBORTagTypeDefAndValue,
			// array, 2 items follow
			0x82,
			// element 0: type definitions
			// array, 3 items follow
			0x83,
			// resource type:
			// id: []byte{}
			// cadence-type-id: "S.test.Stats"
			// 2 fields: [["sum", type(int)], ["count", type(int)]]
			// tag
			0xd8, ccf.CBORTagResourceType,
			// array, 3 items follow
			0x83,
			// id
			// bytes, 0 bytes follow
			0x40,
			// cadence-type-id
			// string, 12 bytes follow
			0x6c,
			// S.test.Stats
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x73,
			// fields
			// array, 2 items follow
			0x82,
			// field 0
			// array, 2 items follow
			0x82,
			// text, 3 bytes follow
			0x63,
			// sum
			0x73, 0x75, 0x6d,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// field 1
			// array, 2 items follow
			0x82,
			// text, 5 bytes follow
			0x65,
			// count
			0x63, 0x6f, 0x75, 0x6e, 0x74,
			// tag
			0xd8, ccf.CBORTagSimpleType,
			// Int type ID (4)
			0x04,
			// resource interface type:
			// id: []byte{1}
			// cadence-type-id: "S.test.HasSum"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 1
			0x01,
			// cadence-type-id
			// string, 13 bytes follow
			0x6d,
			// S.test.HasSum
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x53, 0x75, 0x6d,
			// resource interface type:
			// id: []byte{2}
			// cadence-type-id: "S.test.HasCount"
			// tag
			0xd8, ccf.CBORTagResourceInterfaceType,
			// array, 2 items follow
			0x82,
			// id
			// bytes, 1 bytes follow
			0x41,
			// 2
			0x02,
			// cadence-type-id
			// string, 15 bytes follow
			0x6f,
			// S.test.HasCount
			0x53, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x48, 0x61, 0x73, 0x43, 0x6f, 0x75, 0x6e, 0x74,

			// element 1: type and value
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagVarsizedArrayType,
			// tag
			0xd8, ccf.CBORTagRestrictedType,
			// array, 2 items follow
			0x82,
			// type
			// null
			0xf6,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,

			// array, 1 item follows
			0x81,
			// tag
			0xd8, ccf.CBORTagTypeAndValue,
			// array, 2 items follow
			0x82,
			// tag
			0xd8, ccf.CBORTagTypeRef,
			// bytes, 0 byte follows
			0x40,
			// array, 2 items follow
			0x82,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 2
			0x02,
			// tag (big num)
			0xc2,
			// bytes, 1 byte follows
			0x41,
			// 1
			0x01,
		}

		// Encode value with sorted composite fields and restricted types.
		actualCBOR, err := deterministicEncMode.Encode(val)
		require.NoError(t, err)
		utils.AssertEqualWithDiff(t, expectedCBOR, actualCBOR)

		// Decode value enforcing sorting of composite fields and restricted types.
		decodedVal, err := deterministicDecMode.Decode(nil, actualCBOR)
		require.NoError(t, err)
		assert.Equal(
			t,
			cadence.ValueWithCachedTypeID(expectedVal),
			cadence.ValueWithCachedTypeID(decodedVal),
		)

		// Decode value without enforcing sorting should return no error.
		_, err = ccf.Decode(nil, actualCBOR)
		require.NoError(t, err)

		// Decode value enforcing sorting of composite fields should return no error.
		_, err = enforceSortedFieldsDecMode.Decode(nil, actualCBOR)
		require.NoError(t, err)

		// Decode value enforcing sorting of restricted types should return no error.
		_, err = enforceSortedRestrictedTypesDecMode.Decode(nil, actualCBOR)
		require.NoError(t, err)
	})
}

func TestInvalidEncodingOptions(t *testing.T) {
	opts := ccf.EncOptions{
		SortCompositeFields: 100,
	}
	_, err := opts.EncMode()
	require.Error(t, err)

	opts = ccf.EncOptions{
		SortRestrictedTypes: 100,
	}
	_, err = opts.EncMode()
	require.Error(t, err)
}

func TestInvalidDecodingOptions(t *testing.T) {
	opts := ccf.DecOptions{
		EnforceSortCompositeFields: 100,
	}
	_, err := opts.DecMode()
	require.Error(t, err)

	opts = ccf.DecOptions{
		EnforceSortRestrictedTypes: 100,
	}
	_, err = opts.DecMode()
	require.Error(t, err)
}
