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

package runtime_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/runtime_utils"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestRuntimeParseLiteral(t *testing.T) {
	t.Parallel()

	t.Run("String, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`"hello"`, sema.StringType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.String("hello"),
			value,
		)
	})

	t.Run("String, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.StringType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Bool, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.BoolType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewBool(true),
			value,
		)
	})

	t.Run("Bool, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`"hello"`, sema.BoolType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Optional, nil", func(t *testing.T) {
		value, err := ParseLiteral(
			`nil`,
			&sema.OptionalType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewOptional(nil),
			value,
		)
	})

	t.Run("nested Optional, nil", func(t *testing.T) {
		value, err := ParseLiteral(
			`nil`,
			&sema.OptionalType{
				Type: &sema.OptionalType{
					Type: sema.BoolType,
				},
			},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewOptional(
				cadence.NewOptional(nil),
			),
			value,
		)
	})

	t.Run("Optional, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`true`,
			&sema.OptionalType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewOptional(cadence.NewBool(true)),
			value,
		)
	})

	t.Run("nested Optional, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`true`,
			&sema.OptionalType{
				Type: &sema.OptionalType{
					Type: sema.BoolType,
				},
			},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewOptional(
				cadence.NewOptional(
					cadence.NewBool(true),
				),
			),
			value,
		)
	})

	t.Run("Optional, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`"hello"`,
			&sema.OptionalType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("VariableSizedArray, empty", func(t *testing.T) {
		value, err := ParseLiteral(
			`[]`,
			&sema.VariableSizedType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray([]cadence.Value{}).WithType(cadence.NewVariableSizedArrayType(cadence.BoolType)),
			value,
		)
	})

	t.Run("VariableSizedArray, one element", func(t *testing.T) {
		value, err := ParseLiteral(
			`[true]`,
			&sema.VariableSizedType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray([]cadence.Value{
				cadence.NewBool(true),
			}).WithType(cadence.NewVariableSizedArrayType(cadence.BoolType)),
			value,
		)
	})

	t.Run("VariableSizedArray, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`"hello"`,
			&sema.VariableSizedType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("ConstantSizedArray, empty", func(t *testing.T) {
		value, err := ParseLiteral(
			`[]`,
			&sema.ConstantSizedType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray(
				[]cadence.Value{},
			).WithType(cadence.NewConstantSizedArrayType(0, cadence.BoolType)),
			value,
		)

	})

	t.Run("ConstantSizedArray, one element", func(t *testing.T) {
		value, err := ParseLiteral(
			`[true]`,
			&sema.ConstantSizedType{Type: sema.BoolType, Size: 1},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray([]cadence.Value{
				cadence.NewBool(true),
			}).WithType(cadence.NewConstantSizedArrayType(1, cadence.BoolType)),
			value,
		)
	})

	t.Run("ConstantSizedArray, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`"hello"`,
			&sema.ConstantSizedType{Type: sema.BoolType},
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Dictionary, empty", func(t *testing.T) {
		value, err := ParseLiteral(
			`{}`,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.BoolType,
			},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{}).WithType(cadence.NewDictionaryType(cadence.StringType, cadence.BoolType)),
			value,
		)
	})

	t.Run("Dictionary, one entry", func(t *testing.T) {
		value, err := ParseLiteral(
			`{"hello": true}`,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.BoolType,
			},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("hello"),
					Value: cadence.NewBool(true),
				},
			}).WithType(cadence.NewDictionaryType(cadence.StringType, cadence.BoolType)),
			value,
		)
	})

	t.Run("Dictionary, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`"hello"`,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.BoolType,
			},
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Path, valid literal (storage)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/storage/foo`,
			sema.PathType,
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("Path, valid literal (private)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/private/foo`,
			sema.PathType,
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("Path, valid literal (public)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/public/foo`,
			sema.PathType,
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("Path, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`true`,
			sema.PathType,
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("StoragePath, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`/storage/foo`,
			sema.StoragePathType,
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("StoragePath, invalid literal (private)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/private/foo`,
			sema.StoragePathType,
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("StoragePath, invalid literal (public)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/public/foo`,
			sema.StoragePathType,
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("StoragePath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`true`,
			sema.StoragePathType,
			NewTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("CapabilityPath, valid literal (private)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/private/foo`,
			sema.CapabilityPathType,
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("CapabilityPath, invalid literal (public)", func(t *testing.T) {
		value, err := ParseLiteral(`/public/foo`, sema.CapabilityPathType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("CapabilityPath, invalid literal (storage)", func(t *testing.T) {
		value, err := ParseLiteral(`/storage/foo`, sema.CapabilityPathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("CapabilityPath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.CapabilityPathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PublicPath, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`/public/foo`, sema.PublicPathType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("PublicPath, invalid literal (private)", func(t *testing.T) {
		value, err := ParseLiteral(`/private/foo`, sema.PublicPathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PublicPath, invalid literal (storage)", func(t *testing.T) {
		value, err := ParseLiteral(`/storage/foo`, sema.PublicPathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PublicPath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.PublicPathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PrivatePath, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`/private/foo`, sema.PrivatePathType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			value,
		)
	})

	t.Run("PrivatePath, invalid literal (public)", func(t *testing.T) {
		value, err := ParseLiteral(`/public/foo`, sema.PrivatePathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PrivatePath, invalid literal (storage)", func(t *testing.T) {
		value, err := ParseLiteral(`/storage/foo`, sema.PrivatePathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PrivatePath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.PrivatePathType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Address, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`0x1`, sema.TheAddressType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
			value,
		)
	})

	t.Run("Address, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.TheAddressType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Fix64, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(false, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.Fix64Type, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("Fix64, valid literal, negative", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(true, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`-1.0`, sema.Fix64Type, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("Fix64, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.Fix64Type, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("UFix64, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewUFix64FromParts(1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.UFix64Type, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("UFix64, invalid literal, negative", func(t *testing.T) {
		value, err := ParseLiteral(`-1.0`, sema.UFix64Type, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("UFix64, invalid literal, invalid expression", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.UFix64Type, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("FixedPoint, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(false, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.FixedPointType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("FixedPoint, valid literal, negative", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(true, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`-1.0`, sema.FixedPointType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("FixedPoint, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.FixedPointType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("SignedFixedPoint, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(false, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.SignedFixedPointType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("SignedFixedPoint, valid literal, negative", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(true, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`-1.0`, sema.SignedFixedPointType, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("SignedFixedPoint, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.SignedFixedPointType, NewTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	for _, unsignedIntegerType := range sema.AllUnsignedIntegerTypes {

		t.Run(
			fmt.Sprintf(
				"%s, valid literal, positive",
				unsignedIntegerType.String(),
			),
			func(t *testing.T) {
				value, err := ParseLiteral(`1`, unsignedIntegerType, NewTestInterpreter(t))
				require.NoError(t, err)
				require.NotNil(t, value)
			},
		)

		t.Run(
			fmt.Sprintf(
				"%s, invalid literal, negative",
				unsignedIntegerType.String(),
			),
			func(t *testing.T) {
				value, err := ParseLiteral(`-1`, unsignedIntegerType, NewTestInterpreter(t))
				RequireError(t, err)

				require.Nil(t, value)
			},
		)

		t.Run(
			fmt.Sprintf(
				"%s, invalid literal",
				unsignedIntegerType.String(),
			),
			func(t *testing.T) {
				value, err := ParseLiteral(`true`, unsignedIntegerType, NewTestInterpreter(t))
				RequireError(t, err)

				require.Nil(t, value)
			},
		)
	}

	for _, signedIntegerType := range common.Concat(
		sema.AllSignedIntegerTypes,
		[]sema.Type{
			sema.IntegerType,
			sema.SignedIntegerType,
		},
	) {

		t.Run(
			fmt.Sprintf(
				"%s, valid literal, positive",
				signedIntegerType.String(),
			),
			func(t *testing.T) {
				value, err := ParseLiteral(`1`, signedIntegerType, NewTestInterpreter(t))
				require.NoError(t, err)
				require.NotNil(t, value)
			},
		)

		t.Run(
			fmt.Sprintf(
				"%s, valid literal, negative",
				signedIntegerType.String(),
			),
			func(t *testing.T) {
				value, err := ParseLiteral(`-1`, signedIntegerType, NewTestInterpreter(t))
				require.NoError(t, err)
				require.NotNil(t, value)
			},
		)

		t.Run(
			fmt.Sprintf(
				"%s, invalid literal",
				signedIntegerType.String(),
			),
			func(t *testing.T) {
				value, err := ParseLiteral(`true`, signedIntegerType, NewTestInterpreter(t))
				RequireError(t, err)

				require.Nil(t, value)
			},
		)
	}
}

func TestRuntimeParseLiteralArgumentList(t *testing.T) {
	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseLiteralArgumentList("", nil, NewTestInterpreter(t))
		RequireError(t, err)

	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		arguments, err := ParseLiteralArgumentList(`()`, nil, NewTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, []cadence.Value{}, arguments)
	})

	t.Run("one argument", func(t *testing.T) {
		t.Parallel()

		arguments, err := ParseLiteralArgumentList(
			`(a: 1)`,
			[]sema.Type{
				sema.IntType,
			},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			[]cadence.Value{
				cadence.Int{Value: big.NewInt(1)},
			},
			arguments,
		)
	})

	t.Run("two arguments", func(t *testing.T) {
		t.Parallel()

		arguments, err := ParseLiteralArgumentList(
			`(a: 1, 2)`,
			[]sema.Type{
				sema.IntType,
				sema.IntType,
			},
			NewTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			[]cadence.Value{
				cadence.Int{Value: big.NewInt(1)},
				cadence.Int{Value: big.NewInt(2)},
			},
			arguments,
		)
	})

	t.Run("invalid second argument", func(t *testing.T) {
		t.Parallel()

		_, err := ParseLiteralArgumentList(
			`(a: 1, 2)`,
			[]sema.Type{
				sema.IntType,
				sema.BoolType,
			},
			NewTestInterpreter(t),
		)
		RequireError(t, err)

	})

	t.Run("too many arguments", func(t *testing.T) {
		t.Parallel()

		_, err := ParseLiteralArgumentList(
			`(a: 1, 2)`,
			[]sema.Type{
				sema.IntType,
			},
			NewTestInterpreter(t),
		)
		RequireError(t, err)

	})

	t.Run("too few arguments", func(t *testing.T) {
		t.Parallel()

		_, err := ParseLiteralArgumentList(
			`(a: 1)`,
			[]sema.Type{
				sema.IntType,
				sema.IntType,
			},
			NewTestInterpreter(t),
		)
		RequireError(t, err)

	})

	t.Run("non-literal argument", func(t *testing.T) {
		t.Parallel()

		_, err := ParseLiteralArgumentList(
			`(a: b)`,
			[]sema.Type{
				sema.IntType,
			},
			NewTestInterpreter(t),
		)
		RequireError(t, err)
	})
}
