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

package runtime

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestParseLiteral(t *testing.T) {
	t.Parallel()

	t.Run("String, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`"hello"`, sema.StringType, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.String("hello"),
			value,
		)
	})

	t.Run("String, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.StringType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Bool, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.BoolType, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewBool(true),
			value,
		)
	})

	t.Run("Bool, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`"hello"`, sema.BoolType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Optional, nil", func(t *testing.T) {
		value, err := ParseLiteral(
			`nil`,
			&sema.OptionalType{Type: sema.BoolType},
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("VariableSizedArray, empty", func(t *testing.T) {
		value, err := ParseLiteral(
			`[]`,
			&sema.VariableSizedType{Type: sema.BoolType},
			newTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray([]cadence.Value{}),
			value,
		)
	})

	t.Run("VariableSizedArray, one element", func(t *testing.T) {
		value, err := ParseLiteral(
			`[true]`,
			&sema.VariableSizedType{Type: sema.BoolType},
			newTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray([]cadence.Value{
				cadence.NewBool(true),
			}),
			value,
		)
	})

	t.Run("VariableSizedArray, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`"hello"`,
			&sema.VariableSizedType{Type: sema.BoolType},
			newTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("ConstantSizedArray, empty", func(t *testing.T) {
		value, err := ParseLiteral(
			`[]`,
			&sema.ConstantSizedType{Type: sema.BoolType},
			newTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray([]cadence.Value{}),
			value,
		)
	})

	t.Run("ConstantSizedArray, one element", func(t *testing.T) {
		value, err := ParseLiteral(
			`[true]`,
			&sema.ConstantSizedType{Type: sema.BoolType},
			newTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewArray([]cadence.Value{
				cadence.NewBool(true),
			}),
			value,
		)
	})

	t.Run("ConstantSizedArray, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`"hello"`,
			&sema.ConstantSizedType{Type: sema.BoolType},
			newTestInterpreter(t),
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
			newTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{}),
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
			newTestInterpreter(t),
		)
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("hello"),
					Value: cadence.NewBool(true),
				},
			}),
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
			newTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Path, valid literal (storage)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/storage/foo`,
			sema.PathType,
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("StoragePath, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`/storage/foo`,
			sema.StoragePathType,
			newTestInterpreter(t),
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
			newTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("StoragePath, invalid literal (public)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/public/foo`,
			sema.StoragePathType,
			newTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("StoragePath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(
			`true`,
			sema.StoragePathType,
			newTestInterpreter(t),
		)
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("CapabilityPath, valid literal (private)", func(t *testing.T) {
		value, err := ParseLiteral(
			`/private/foo`,
			sema.CapabilityPathType,
			newTestInterpreter(t),
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
		value, err := ParseLiteral(`/public/foo`, sema.CapabilityPathType, newTestInterpreter(t))
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
		value, err := ParseLiteral(`/storage/foo`, sema.CapabilityPathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("CapabilityPath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.CapabilityPathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PublicPath, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`/public/foo`, sema.PublicPathType, newTestInterpreter(t))
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
		value, err := ParseLiteral(`/private/foo`, sema.PublicPathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PublicPath, invalid literal (storage)", func(t *testing.T) {
		value, err := ParseLiteral(`/storage/foo`, sema.PublicPathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PublicPath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.PublicPathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PrivatePath, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`/private/foo`, sema.PrivatePathType, newTestInterpreter(t))
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
		value, err := ParseLiteral(`/public/foo`, sema.PrivatePathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PrivatePath, invalid literal (storage)", func(t *testing.T) {
		value, err := ParseLiteral(`/storage/foo`, sema.PrivatePathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("PrivatePath, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`true`, sema.PrivatePathType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Address, valid literal", func(t *testing.T) {
		value, err := ParseLiteral(`0x1`, sema.TheAddressType, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t,
			cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
			value,
		)
	})

	t.Run("Address, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.TheAddressType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("Fix64, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(false, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.Fix64Type, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("Fix64, valid literal, negative", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(true, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`-1.0`, sema.Fix64Type, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("Fix64, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.Fix64Type, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("UFix64, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewUFix64FromParts(1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.UFix64Type, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("UFix64, invalid literal, negative", func(t *testing.T) {
		value, err := ParseLiteral(`-1.0`, sema.UFix64Type, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("UFix64, invalid literal, invalid expression", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.UFix64Type, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("FixedPoint, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(false, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.FixedPointType, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("FixedPoint, valid literal, negative", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(true, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`-1.0`, sema.FixedPointType, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("FixedPoint, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.FixedPointType, newTestInterpreter(t))
		RequireError(t, err)

		require.Nil(t, value)
	})

	t.Run("SignedFixedPoint, valid literal, positive", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(false, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`1.0`, sema.SignedFixedPointType, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("SignedFixedPoint, valid literal, negative", func(t *testing.T) {
		expected, err := cadence.NewFix64FromParts(true, 1, 0)
		require.NoError(t, err)

		value, err := ParseLiteral(`-1.0`, sema.SignedFixedPointType, newTestInterpreter(t))
		require.NoError(t, err)
		require.Equal(t, expected, value)
	})

	t.Run("SignedFixedPoint, invalid literal", func(t *testing.T) {
		value, err := ParseLiteral(`1`, sema.SignedFixedPointType, newTestInterpreter(t))
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
				value, err := ParseLiteral(`1`, unsignedIntegerType, newTestInterpreter(t))
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
				value, err := ParseLiteral(`-1`, unsignedIntegerType, newTestInterpreter(t))
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
				value, err := ParseLiteral(`true`, unsignedIntegerType, newTestInterpreter(t))
				RequireError(t, err)

				require.Nil(t, value)
			},
		)
	}

	for _, signedIntegerType := range append(
		sema.AllSignedIntegerTypes[:],
		sema.IntegerType,
		sema.SignedIntegerType,
	) {

		t.Run(
			fmt.Sprintf(
				"%s, valid literal, positive",
				signedIntegerType.String(),
			),
			func(t *testing.T) {
				value, err := ParseLiteral(`1`, signedIntegerType, newTestInterpreter(t))
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
				value, err := ParseLiteral(`-1`, signedIntegerType, newTestInterpreter(t))
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
				value, err := ParseLiteral(`true`, signedIntegerType, newTestInterpreter(t))
				RequireError(t, err)

				require.Nil(t, value)
			},
		)
	}
}

func TestParseLiteralArgumentList(t *testing.T) {
	t.Parallel()

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseLiteralArgumentList("", nil, newTestInterpreter(t))
		RequireError(t, err)

	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		arguments, err := ParseLiteralArgumentList(`()`, nil, newTestInterpreter(t))
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
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
			newTestInterpreter(t),
		)
		RequireError(t, err)
	})
}
