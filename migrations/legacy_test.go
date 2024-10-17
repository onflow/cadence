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

package migrations

import (
	"bytes"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/tests/utils"
)

func TestLegacyEquality(t *testing.T) {

	t.Parallel()

	t.Run("Character value", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyCharacterValue{
				CharacterValue: interpreter.NewUnmeteredCharacterValue("foo"),
			}).Equal(nil, emptyLocationRange, &LegacyCharacterValue{
				CharacterValue: interpreter.NewUnmeteredCharacterValue("foo"),
			}),
		)
	})

	t.Run("String value", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyStringValue{
				StringValue: interpreter.NewUnmeteredStringValue("foo"),
			}).Equal(nil, emptyLocationRange, &LegacyStringValue{
				StringValue: interpreter.NewUnmeteredStringValue("foo"),
			}),
		)
	})

	t.Run("Intersection type", func(t *testing.T) {
		t.Parallel()

		fooType := interpreter.NewInterfaceStaticTypeComputeTypeID(
			nil,
			utils.TestLocation,
			"Test.Foo",
		)

		require.True(t,
			(&LegacyIntersectionType{
				IntersectionStaticType: interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooType,
					},
				),
			}).Equal(&LegacyIntersectionType{
				IntersectionStaticType: interpreter.NewIntersectionStaticType(
					nil,
					[]*interpreter.InterfaceStaticType{
						fooType,
					},
				),
			}),
		)
	})

	t.Run("Primitive type", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(LegacyPrimitiveStaticType{
				PrimitiveStaticType: interpreter.PrimitiveStaticTypeInt,
			}).Equal(LegacyPrimitiveStaticType{
				PrimitiveStaticType: interpreter.PrimitiveStaticTypeInt,
			}),
		)
	})

	t.Run("Reference type", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyReferenceType{
				ReferenceStaticType: &interpreter.ReferenceStaticType{
					Authorization:  interpreter.UnauthorizedAccess,
					ReferencedType: interpreter.PrimitiveStaticTypeInt,
				},
			}).Equal(&LegacyReferenceType{
				ReferenceStaticType: &interpreter.ReferenceStaticType{
					Authorization:  interpreter.UnauthorizedAccess,
					ReferencedType: interpreter.PrimitiveStaticTypeInt,
				},
			}),
		)
	})

	t.Run("Optional type", func(t *testing.T) {
		t.Parallel()

		require.True(t,
			(&LegacyOptionalType{
				OptionalStaticType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			}).Equal(&LegacyOptionalType{
				OptionalStaticType: &interpreter.OptionalStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			}),
		)
	})
}

func TestLegacyOptionalType(t *testing.T) {
	t.Parallel()

	test := func(
		t *testing.T,
		optionalType *interpreter.OptionalStaticType,
		expectedTypeID common.TypeID,
		expectedEncoding []byte,
	) {

		legacyRefType := &LegacyOptionalType{
			OptionalStaticType: optionalType,
		}

		assert.Equal(t,
			expectedTypeID,
			legacyRefType.ID(),
		)

		var buf bytes.Buffer

		encoder := cbor.NewStreamEncoder(&buf)
		err := legacyRefType.Encode(encoder)
		require.NoError(t, err)

		err = encoder.Flush()
		require.NoError(t, err)

		assert.Equal(t, expectedEncoding, buf.Bytes())
	}

	t.Run("reference to optional", func(t *testing.T) {

		t.Parallel()

		optionalType := interpreter.NewOptionalStaticType(
			nil,
			interpreter.NewReferenceStaticType(
				nil,
				interpreter.UnauthorizedAccess,
				interpreter.PrimitiveStaticTypeAnyStruct,
			),
		)

		test(t,
			optionalType,
			"&AnyStruct?",
			[]byte{
				// tag
				0xd8, interpreter.CBORTagOptionalStaticType,
				// tag
				0xd8, interpreter.CBORTagReferenceStaticType,
				// array, 2 items follow
				0x82,
				// Unauthorized
				0xd8, interpreter.CBORTagUnauthorizedStaticAuthorization,
				// nil
				0xf6,
				// tag
				0xd8, interpreter.CBORTagPrimitiveStaticType,
				// AnyStruct,
				byte(interpreter.PrimitiveStaticTypeAnyStruct),
			},
		)
	})
}

func TestLegacyReferenceType(t *testing.T) {

	t.Parallel()

	test := func(
		t *testing.T,
		refType *interpreter.ReferenceStaticType,
		expectedTypeID common.TypeID,
		expectedEncoding []byte,
	) {

		legacyRefType := &LegacyReferenceType{
			ReferenceStaticType: refType,
		}

		assert.Equal(t,
			expectedTypeID,
			legacyRefType.ID(),
		)

		var buf bytes.Buffer

		encoder := cbor.NewStreamEncoder(&buf)
		err := legacyRefType.Encode(encoder)
		require.NoError(t, err)

		err = encoder.Flush()
		require.NoError(t, err)

		assert.Equal(t, expectedEncoding, buf.Bytes())
	}

	t.Run("has legacy authorized, unauthorized", func(t *testing.T) {

		t.Parallel()

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			interpreter.PrimitiveStaticTypeAnyStruct,
		)
		refType.HasLegacyIsAuthorized = true
		refType.LegacyIsAuthorized = false

		test(t,
			refType,
			"&AnyStruct",
			[]byte{
				// tag
				0xd8, interpreter.CBORTagReferenceStaticType,
				// array, 2 items follow
				0x82,
				// authorized = false
				0xf4,
				// tag
				0xd8, interpreter.CBORTagPrimitiveStaticType,
				// AnyStruct,
				byte(interpreter.PrimitiveStaticTypeAnyStruct),
			},
		)
	})

	t.Run("has legacy authorized, authorized", func(t *testing.T) {

		t.Parallel()

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			interpreter.PrimitiveStaticTypeAnyStruct,
		)
		refType.HasLegacyIsAuthorized = true
		refType.LegacyIsAuthorized = true

		test(t,
			refType,
			"auth&AnyStruct",
			[]byte{
				// tag
				0xd8, interpreter.CBORTagReferenceStaticType,
				// array, 2 items follow
				0x82,
				// authorized = true
				0xf5,
				// tag
				0xd8, interpreter.CBORTagPrimitiveStaticType,
				// AnyStruct,
				byte(interpreter.PrimitiveStaticTypeAnyStruct),
			},
		)
	})

	t.Run("new authorization, unauthorized", func(t *testing.T) {
		t.Parallel()

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.UnauthorizedAccess,
			interpreter.PrimitiveStaticTypeAnyStruct,
		)

		test(t,
			refType,
			"&AnyStruct",
			[]byte{
				// tag
				0xd8, interpreter.CBORTagReferenceStaticType,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, interpreter.CBORTagUnauthorizedStaticAuthorization,
				// nil
				0xf6,
				// tag
				0xd8, interpreter.CBORTagPrimitiveStaticType,
				// AnyStruct,
				byte(interpreter.PrimitiveStaticTypeAnyStruct),
			},
		)

	})

	t.Run("new authorization, authorized", func(t *testing.T) {
		t.Parallel()

		refType := interpreter.NewReferenceStaticType(
			nil,
			interpreter.NewEntitlementSetAuthorization(
				nil,
				func() []common.TypeID {
					return []common.TypeID{"Foo"}
				},
				1,
				sema.Conjunction,
			),
			interpreter.PrimitiveStaticTypeAnyStruct,
		)

		test(t,
			refType,
			"auth(Foo)&AnyStruct",
			[]byte{
				// tag
				0xd8, interpreter.CBORTagReferenceStaticType,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, interpreter.CBORTagEntitlementSetStaticAuthorization,
				// array, 2 items follow
				0x82,
				0x0,
				// array, 1 items follow
				0x81,
				// UTF-8 string, 3 bytes follow
				0x63,
				// F, o, o
				0x46, 0x6f, 0x6f,
				// tag
				0xd8, interpreter.CBORTagPrimitiveStaticType,
				// AnyStruct,
				byte(interpreter.PrimitiveStaticTypeAnyStruct),
			},
		)
	})
}

func TestLegacyIntersectionType(t *testing.T) {
	t.Parallel()

	test := func(
		t *testing.T,
		intersectionType *interpreter.IntersectionStaticType,
		expectedTypeID common.TypeID,
		expectedEncoding []byte,
	) {

		legacyIntersectionType := &LegacyIntersectionType{
			IntersectionStaticType: intersectionType,
		}

		assert.Equal(t,
			expectedTypeID,
			legacyIntersectionType.ID(),
		)

		var buf bytes.Buffer

		encoder := cbor.NewStreamEncoder(&buf)
		err := legacyIntersectionType.Encode(encoder)
		require.NoError(t, err)

		err = encoder.Flush()
		require.NoError(t, err)

		assert.Equal(t, expectedEncoding, buf.Bytes())
	}

	t.Run("unsorted", func(t *testing.T) {

		t.Parallel()

		fooType := interpreter.NewInterfaceStaticTypeComputeTypeID(
			nil,
			utils.TestLocation,
			"Test.Foo",
		)

		barType := interpreter.NewInterfaceStaticTypeComputeTypeID(
			nil,
			utils.TestLocation,
			"Test.Bar",
		)

		intersectionType := interpreter.NewIntersectionStaticType(
			nil,
			[]*interpreter.InterfaceStaticType{
				fooType,
				barType,
			},
		)

		test(t,
			intersectionType,
			"{S.test.Test.Foo,S.test.Test.Bar}",
			[]byte{
				// tag
				0xd8, interpreter.CBORTagIntersectionStaticType,
				// array, length 2
				0x82,
				// nil
				0xf6,
				// array, length 2
				0x82,
				// tag
				0xd8, interpreter.CBORTagInterfaceStaticType,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, interpreter.CBORTagStringLocation,
				// UTF-8 string, length 4
				0x64,
				// t, e, s, t
				0x74, 0x65, 0x73, 0x74,
				// UTF-8 string, length 8
				0x68,
				// T, e, s, t, ., F, o, o
				0x54, 0x65, 0x73, 0x74, 0x2e, 0x46, 0x6f, 0x6f,
				// tag
				0xd8, interpreter.CBORTagInterfaceStaticType,
				// array, 2 items follow
				0x82,
				// tag
				0xd8, interpreter.CBORTagStringLocation,
				// UTF-8 string, length 4
				0x64,
				// t, e, s, t
				0x74, 0x65, 0x73, 0x74,
				// UTF-8 string, length 8
				0x68,
				// T, e, s, t, ., B, a, r
				0x54, 0x65, 0x73, 0x74, 0x2e, 0x42, 0x61, 0x72,
			},
		)
	})
}
