/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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

package sema_codec_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/encoding/custom/common_codec"

	"github.com/onflow/cadence/encoding/custom/sema_codec"
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestSemaCodecSimpleTypes(t *testing.T) {
	t.Parallel()

	type TestInfo struct {
		SimpleType *sema.SimpleType
		Type       sema_codec.EncodedSema
	}

	tests := []TestInfo{
		{sema.AnyType, sema_codec.EncodedSemaSimpleTypeAnyType},
		{sema.AnyResourceType, sema_codec.EncodedSemaSimpleTypeAnyResourceType},
		{sema.AnyStructType, sema_codec.EncodedSemaSimpleTypeAnyStructType},
		{sema.BlockType, sema_codec.EncodedSemaSimpleTypeBlockType},
		{sema.BoolType, sema_codec.EncodedSemaSimpleTypeBoolType},
		{sema.CharacterType, sema_codec.EncodedSemaSimpleTypeCharacterType},
		{sema.DeployedContractType, sema_codec.EncodedSemaSimpleTypeDeployedContractType},
		{sema.InvalidType, sema_codec.EncodedSemaSimpleTypeInvalidType},
		{sema.MetaType, sema_codec.EncodedSemaSimpleTypeMetaType},
		{sema.NeverType, sema_codec.EncodedSemaSimpleTypeNeverType},
		{sema.PathType, sema_codec.EncodedSemaSimpleTypePathType},
		{sema.StoragePathType, sema_codec.EncodedSemaSimpleTypeStoragePathType},
		{sema.CapabilityPathType, sema_codec.EncodedSemaSimpleTypeCapabilityPathType},
		{sema.PublicPathType, sema_codec.EncodedSemaSimpleTypePublicPathType},
		{sema.PrivatePathType, sema_codec.EncodedSemaSimpleTypePrivatePathType},
		{sema.StorableType, sema_codec.EncodedSemaSimpleTypeStorableType},
		{sema.StringType, sema_codec.EncodedSemaSimpleTypeStringType},
		{sema.VoidType, sema_codec.EncodedSemaSimpleTypeVoidType},
	}

	for _, test := range tests {
		func(typ TestInfo) {
			t.Run(typ.SimpleType.Name, func(t *testing.T) {
				t.Parallel()
				testRootEncodeDecode(t, typ.SimpleType,
					byte(typ.Type),
				)
			})
		}(test)
	}
}

func TestSemaCodecNumericTypes(t *testing.T) {
	t.Parallel()

	type TestInfo struct {
		SimpleType sema.Type
		Type       sema_codec.EncodedSema
	}

	tests := []TestInfo{
		{sema.NumberType, sema_codec.EncodedSemaNumericTypeNumberType},
		{sema.SignedNumberType, sema_codec.EncodedSemaNumericTypeSignedNumberType},
		{sema.IntegerType, sema_codec.EncodedSemaNumericTypeIntegerType},
		{sema.SignedIntegerType, sema_codec.EncodedSemaNumericTypeSignedIntegerType},
		{sema.IntType, sema_codec.EncodedSemaNumericTypeIntType},
		{sema.Int8Type, sema_codec.EncodedSemaNumericTypeInt8Type},
		{sema.Int16Type, sema_codec.EncodedSemaNumericTypeInt16Type},
		{sema.Int32Type, sema_codec.EncodedSemaNumericTypeInt32Type},
		{sema.Int64Type, sema_codec.EncodedSemaNumericTypeInt64Type},
		{sema.Int128Type, sema_codec.EncodedSemaNumericTypeInt128Type},
		{sema.Int256Type, sema_codec.EncodedSemaNumericTypeInt256Type},
		{sema.UIntType, sema_codec.EncodedSemaNumericTypeUIntType},
		{sema.UInt8Type, sema_codec.EncodedSemaNumericTypeUInt8Type},
		{sema.UInt16Type, sema_codec.EncodedSemaNumericTypeUInt16Type},
		{sema.UInt32Type, sema_codec.EncodedSemaNumericTypeUInt32Type},
		{sema.UInt64Type, sema_codec.EncodedSemaNumericTypeUInt64Type},
		{sema.UInt128Type, sema_codec.EncodedSemaNumericTypeUInt128Type},
		{sema.UInt256Type, sema_codec.EncodedSemaNumericTypeUInt256Type},
		{sema.Word8Type, sema_codec.EncodedSemaNumericTypeWord8Type},
		{sema.Word16Type, sema_codec.EncodedSemaNumericTypeWord16Type},
		{sema.Word32Type, sema_codec.EncodedSemaNumericTypeWord32Type},
		{sema.Word64Type, sema_codec.EncodedSemaNumericTypeWord64Type},
		{sema.FixedPointType, sema_codec.EncodedSemaNumericTypeFixedPointType},
		{sema.SignedFixedPointType, sema_codec.EncodedSemaNumericTypeSignedFixedPointType},
		{sema.Fix64Type, sema_codec.EncodedSemaFix64Type},
		{sema.UFix64Type, sema_codec.EncodedSemaUFix64Type},
	}

	for _, test := range tests {
		func(typ TestInfo) {
			t.Run(typ.SimpleType.String(), func(t *testing.T) {
				t.Parallel()
				testRootEncodeDecode(t, typ.SimpleType,
					byte(typ.Type),
				)
			})
		}(test)
	}
}

func TestSemaCodecMiscTypes(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		testRootEncodeDecode(t, nil, byte(sema_codec.EncodedSemaNilType))
	})

	t.Run("AddressType", func(t *testing.T) {
		t.Parallel()
		testRootEncodeDecode(t, &sema.AddressType{}, byte(sema_codec.EncodedSemaAddressType))
	})

	t.Run("OptionalType", func(t *testing.T) {
		t.Parallel()

		testRootEncodeDecode(
			t,
			&sema.OptionalType{Type: sema.BoolType},
			byte(sema_codec.EncodedSemaOptionalType),
			byte(sema_codec.EncodedSemaSimpleTypeBoolType),
		)
	})

	t.Run("ReferenceType", func(t *testing.T) {
		t.Parallel()

		testRootEncodeDecode(
			t,
			&sema.ReferenceType{
				Authorized: false,
				Type:       sema.AnyType,
			},
			byte(sema_codec.EncodedSemaReferenceType),
			byte(common_codec.EncodedBoolFalse),
			byte(sema_codec.EncodedSemaSimpleTypeAnyType),
		)
	})

	t.Run("CapabilityType", func(t *testing.T) {
		t.Parallel()

		testRootEncodeDecode(
			t,
			&sema.CapabilityType{BorrowType: sema.VoidType},
			byte(sema_codec.EncodedSemaCapabilityType),
			byte(sema_codec.EncodedSemaSimpleTypeVoidType),
		)
	})

	t.Run("GenericType", func(t *testing.T) {
		t.Parallel()

		name := "could be anything"

		testRootEncodeDecode(
			t,
			&sema.GenericType{TypeParameter: &sema.TypeParameter{
				Name:      name,
				TypeBound: sema.Int32Type,
				Optional:  true,
			}},
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaGenericType)},
				[]byte{0, 0, 0, byte(len(name))},
				[]byte(name),
				[]byte{byte(sema_codec.EncodedSemaNumericTypeInt32Type)},
				[]byte{byte(common_codec.EncodedBoolTrue)},
			)...,
		)
	})

	t.Run("GenericType (no TypeBound)", func(t *testing.T) {
		t.Parallel()

		name := "could be anything"

		testRootEncodeDecode(
			t,
			&sema.GenericType{TypeParameter: &sema.TypeParameter{
				Name:      name,
				TypeBound: nil,
				Optional:  true,
			}},
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaGenericType)},
				[]byte{0, 0, 0, byte(len(name))},
				[]byte(name),
				[]byte{byte(sema_codec.EncodedSemaNilType)},
				[]byte{byte(common_codec.EncodedBoolTrue)},
			)...,
		)
	})

	t.Run("FunctionType", func(t *testing.T) {
		t.Parallel()

		const isConstructor = true
		typeParameters := []*sema.TypeParameter{
			{
				Name:      "myriad",
				TypeBound: sema.VoidType,
				Optional:  false,
			},
		}
		parameters := []*sema.Parameter{
			{
				Label:          "juno",
				Identifier:     "fake0",
				TypeAnnotation: sema.NewTypeAnnotation(sema.AnyResourceType),
			},
			{
				Label:          "calipso",
				Identifier:     "fake1",
				TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
			},
		}
		returnTypeAnnotation := sema.NewTypeAnnotation(sema.PathType)
		requiredArgumentCount := 1

		members := &sema.StringMemberOrderedMap{}
		memberIdentifer := "someID"
		memberDocString := "\"doctored\" string"
		members.Set("yolo", sema.NewPublicConstantFieldMember(
			nil,
			sema.PrivatePathType,
			memberIdentifer,
			sema.Int8Type,
			memberDocString,
		))

		functionType := &sema.FunctionType{
			IsConstructor:            isConstructor,
			TypeParameters:           typeParameters,
			Parameters:               parameters,
			ReturnTypeAnnotation:     returnTypeAnnotation,
			RequiredArgumentCount:    &requiredArgumentCount,
			ArgumentExpressionsCheck: nil,
			Members:                  members,
		}

		encoder, decoder, buffer := NewTestCodec()

		err := encoder.EncodeType(functionType)
		require.NoError(t, err, "encoding error")

		expected := common_codec.Concat(
			[]byte{byte(sema_codec.EncodedSemaFunctionType)},

			[]byte{byte(common_codec.EncodedBoolTrue)}, // isConstructor

			[]byte{byte(common_codec.EncodedBoolFalse)}, // TypeParameters array is non-nil
			[]byte{0, 0, 0, byte(len(typeParameters))},
			[]byte{0, 0, 0, byte(len(typeParameters[0].Name))},
			[]byte(typeParameters[0].Name),
			[]byte{byte(sema_codec.EncodedSemaSimpleTypeVoidType)},
			[]byte{byte(common_codec.EncodedBoolFalse)},

			[]byte{byte(common_codec.EncodedBoolFalse)}, // Parameters array is non-nil
			[]byte{0, 0, 0, byte(len(parameters))},
			[]byte{0, 0, 0, byte(len(parameters[0].Label))},
			[]byte(parameters[0].Label),
			[]byte{0, 0, 0, byte(len(parameters[0].Identifier))},
			[]byte(parameters[0].Identifier),
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(common_codec.EncodedBoolTrue)},
			[]byte{byte(sema_codec.EncodedSemaSimpleTypeAnyResourceType)},
			[]byte{0, 0, 0, byte(len(parameters[1].Label))},
			[]byte(parameters[1].Label),
			[]byte{0, 0, 0, byte(len(parameters[1].Identifier))},
			[]byte(parameters[1].Identifier),
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(sema_codec.EncodedSemaSimpleTypeStringType)},

			[]byte{byte(common_codec.EncodedBoolFalse)}, // TypeAnnotation is not nil
			[]byte{byte(common_codec.EncodedBoolFalse)}, // TypeAnnotation: it is not a Resource
			[]byte{byte(sema_codec.EncodedSemaSimpleTypePathType)},

			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(requiredArgumentCount)},

			[]byte{byte(common_codec.EncodedBoolFalse)},      // Members is not nil
			[]byte{0, 0, 0, byte(members.Len())},             // Members length
			[]byte{0, 0, 0, byte(len(members.Newest().Key))}, // Member key
			[]byte(members.Newest().Key),
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(ast.AccessPublic)}, // Member value
			[]byte{0, 0, 0, byte(len(memberIdentifer))},         // Member AST identifier
			[]byte(memberIdentifer),
			[]byte{0, 0, 0, 0, 0, 0, 0, 0}, // Member AST identifier position
			[]byte{0, 0, 0, 0, 0, 0, 0, 0},
			[]byte{0, 0, 0, 0, 0, 0, 0, 0},
			[]byte{byte(common_codec.EncodedBoolFalse)}, // Member type annotation
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(sema_codec.EncodedSemaNumericTypeInt8Type)},
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(common.DeclarationKindField)}, // Member declaration kind
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(ast.VariableKindConstant)},    // member variable kind
			[]byte{byte(common_codec.EncodedBoolTrue)},                     // Member has no argument labels
			[]byte{byte(common_codec.EncodedBoolFalse)},                    // Member is not predeclared
			[]byte{0, 0, 0, byte(len(memberDocString))},                    // Member doc string
			[]byte(memberDocString),
		)

		assert.Equal(t, expected, buffer.Bytes(), "encoded bytes differ")

		decoded, err := decoder.DecodeType()
		require.NoError(t, err, "decoding error")

		// Cannot simply check equality between original and decoded types because they are not shallowly equal.
		// Specifically, RequiredArgumentCount and Members are not shallowly equal.
		switch f := decoded.(type) {
		case *sema.FunctionType:
			assert.Equal(t, isConstructor, f.IsConstructor)

			require.NotNil(t, f.TypeParameters, "TypeParameters")
			require.Len(t, f.TypeParameters, 1, "TypeParameters")
			assert.Equal(t, typeParameters[0], f.TypeParameters[0], "TypeParameters[0]")

			require.NotNil(t, f.Parameters, "Parameters")
			require.Len(t, f.Parameters, 2, "Parameters")
			assert.Equal(t, parameters[0], f.Parameters[0], "Parameters[0]")
			assert.Equal(t, parameters[1], f.Parameters[1], "Parameters[1]")

			assert.Equal(t, returnTypeAnnotation, f.ReturnTypeAnnotation, "ReturnTypeAnnotation")

			assert.Equal(t, requiredArgumentCount, *f.RequiredArgumentCount, "RequiredArgumentCount")

			assert.Nil(t, f.ArgumentExpressionsCheck, "ArgumentExpressionsCheck")

			// verify member equality
			require.Equal(t, members.Len(), f.Members.Len(), "members length")
			f.Members.Foreach(func(key string, actual *sema.Member) {
				expected, present := f.Members.Get(key)
				require.True(t, present, "extra member: %s", key)

				assert.Equal(t, expected.ContainerType.ID(), actual.ContainerType.ID(), "container type for %s", key)
				assert.Equal(t, expected.TypeAnnotation.QualifiedString(), actual.TypeAnnotation.QualifiedString(), "type annotation for %s", key)
			})
		default:
			assert.Fail(t, "Decoded type is not *sema.FunctionTypre")
		}
	})

	t.Run("DictionaryType", func(t *testing.T) {
		t.Parallel()

		testRootEncodeDecode(
			t,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.AnyStructType,
			},
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaDictionaryType)},
				[]byte{byte(sema_codec.EncodedSemaSimpleTypeStringType)},
				[]byte{byte(sema_codec.EncodedSemaSimpleTypeAnyStructType)},
			)...,
		)
	})

	t.Run("TransactionType", func(t *testing.T) {
		t.Parallel()

		members := &sema.StringMemberOrderedMap{}
		memberIdentifer := "someID"
		memberDocString := "\"doctored\" string"
		members.Set("yol2", sema.NewPublicConstantFieldMember(
			nil,
			sema.PrivatePathType,
			memberIdentifer,
			sema.Int8Type,
			memberDocString,
		))

		fields := []string{
			"twelve",
			"twenty four",
			"forty eight",
			"ninety six",
		}

		prepareParameters := []*sema.Parameter{
			{
				Label:          "replay",
				Identifier:     "fake6",
				TypeAnnotation: sema.NewTypeAnnotation(sema.UInt16Type),
			},
		}

		parameters := []*sema.Parameter{
			{
				Label:          "hadron",
				Identifier:     "collision",
				TypeAnnotation: sema.NewTypeAnnotation(sema.SignedFixedPointType),
			},
		}

		transactionType := &sema.TransactionType{
			Members:           members,
			Fields:            fields,
			PrepareParameters: prepareParameters,
			Parameters:        parameters,
		}

		encoder, decoder, buffer := NewTestCodec()

		err := encoder.EncodeType(transactionType)
		require.NoError(t, err, "encoding error")

		expected := common_codec.Concat(
			[]byte{byte(sema_codec.EncodedSemaTransactionType)},
			// members
			[]byte{byte(common_codec.EncodedBoolFalse)},      // Members is not nil
			[]byte{0, 0, 0, byte(members.Len())},             // Members length
			[]byte{0, 0, 0, byte(len(members.Newest().Key))}, // Member key
			[]byte(members.Newest().Key),
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(ast.AccessPublic)}, // Member value
			[]byte{0, 0, 0, byte(len(memberIdentifer))},         // Member AST identifier
			[]byte(memberIdentifer),
			[]byte{0, 0, 0, 0, 0, 0, 0, 0}, // Member AST identifier position
			[]byte{0, 0, 0, 0, 0, 0, 0, 0},
			[]byte{0, 0, 0, 0, 0, 0, 0, 0},
			[]byte{byte(common_codec.EncodedBoolFalse)}, // Member type annotation
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(sema_codec.EncodedSemaNumericTypeInt8Type)},
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(common.DeclarationKindField)}, // Member declaration kind
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(ast.VariableKindConstant)},    // member variable kind
			[]byte{byte(common_codec.EncodedBoolTrue)},                     // Member has no argument labels
			[]byte{byte(common_codec.EncodedBoolFalse)},                    // Member is not predeclared
			[]byte{0, 0, 0, byte(len(memberDocString))},                    // Member doc string
			[]byte(memberDocString),

			// array of strings for fields
			[]byte{byte(common_codec.EncodedBoolFalse)}, // array is not nil
			[]byte{0, 0, 0, byte(len(fields))},
			[]byte{0, 0, 0, byte(len(fields[0]))},
			[]byte(fields[0]),
			[]byte{0, 0, 0, byte(len(fields[1]))},
			[]byte(fields[1]),
			[]byte{0, 0, 0, byte(len(fields[2]))},
			[]byte(fields[2]),
			[]byte{0, 0, 0, byte(len(fields[3]))},
			[]byte(fields[3]),

			// array of parameters for prepareParameters
			[]byte{byte(common_codec.EncodedBoolFalse)}, // array is not nil
			[]byte{0, 0, 0, byte(len(prepareParameters))},
			[]byte{0, 0, 0, byte(len(prepareParameters[0].Label))},
			[]byte(prepareParameters[0].Label),
			[]byte{0, 0, 0, byte(len(prepareParameters[0].Identifier))},
			[]byte(prepareParameters[0].Identifier),
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(sema_codec.EncodedSemaNumericTypeUInt16Type)},

			// array of parameters for parameters
			[]byte{byte(common_codec.EncodedBoolFalse)}, // array is not nil
			[]byte{0, 0, 0, byte(len(parameters))},
			[]byte{0, 0, 0, byte(len(parameters[0].Label))},
			[]byte(parameters[0].Label),
			[]byte{0, 0, 0, byte(len(parameters[0].Identifier))},
			[]byte(parameters[0].Identifier),
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(sema_codec.EncodedSemaNumericTypeSignedFixedPointType)},
		)

		assert.Equal(t, expected, buffer.Bytes(), "encoded bytes differ")

		decoded, err := decoder.DecodeType()
		require.NoError(t, err, "decoding error")

		// Cannot simply check equality between original and decoded types because they are not shallowly equal.
		// Specifically, Members is not shallowly equal.
		switch tx := decoded.(type) {
		case *sema.TransactionType:
			// verify member equality
			require.Equal(t, members.Len(), tx.Members.Len(), "members length")
			tx.Members.Foreach(func(key string, actual *sema.Member) {
				expected, present := tx.Members.Get(key)
				require.True(t, present, "extra member: %s", key)

				assert.Equal(t, expected.ContainerType.ID(), actual.ContainerType.ID(), "container type for %s", key)
				assert.Equal(t, expected.TypeAnnotation.QualifiedString(), actual.TypeAnnotation.QualifiedString(), "type annotation for %s", key)
			})

			assert.Equal(t, fields, tx.Fields, "fields")
			assert.Equal(t, tx.Parameters, parameters, "parameters")
			assert.Equal(t, tx.PrepareParameters, prepareParameters, "prepareParameters")
		default:
			assert.Fail(t, "Decoded type is not *sema.TransactionType")
		}
	})

	t.Run("RestrictedType", func(t *testing.T) {
		t.Parallel()

		location := common.ScriptLocation{12, 24, 48, 96}
		restrictedType := &sema.RestrictedType{
			Type: sema.IntType,
			Restrictions: []*sema.InterfaceType{{
				Location:              location,
				Identifier:            "peaked",
				CompositeKind:         common.CompositeKindContract,
				Members:               nil,
				Fields:                nil,
				InitializerParameters: nil,
			}},
		}
		restrictedType.Restrictions[0].SetContainerType(restrictedType)

		encoder, decoder, buffer := NewTestCodec()

		err := encoder.EncodeType(restrictedType)
		require.NoError(t, err, "encoding error")

		expected := common_codec.Concat(
			[]byte{byte(sema_codec.EncodedSemaRestrictedType)},
			[]byte{byte(sema_codec.EncodedSemaNumericTypeIntType)},
			[]byte{byte(common_codec.EncodedBoolFalse)}, // array is not nil
			[]byte{0, 0, 0, 1}, // array length
			common_codec.Concat([]byte{'s'}, location[:]),
			[]byte{0, 0, 0, 6}, []byte("peaked"), // identifier
			[]byte{byte(common.CompositeKindContract)},
			[]byte{byte(common_codec.EncodedBoolTrue)},                  // members is nil
			[]byte{byte(common_codec.EncodedBoolTrue)},                  // fields is nil
			[]byte{byte(common_codec.EncodedBoolTrue)},                  // initializer parameters is nil
			[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // container type is root type
			[]byte{byte(common_codec.EncodedBoolTrue)},                  // nested types is nil
		)

		assert.Equal(t, expected, buffer.Bytes(), "encoded bytes differ")

		decoded, err := decoder.DecodeType()
		require.NoError(t, err, "decoding error")

		// Cannot simply check equality between original and decoded types because they are not shallowly equal.
		// Specifically, the elements of Restrictions are not shallowly equal.
		switch r := decoded.(type) {
		case *sema.RestrictedType:
			assert.Equal(t, sema.IntType, r.Type, "Type")

			require.Len(t, r.Restrictions, 1, "restrictions length")

			// minimal verification
			assert.Equal(t, restrictedType.Restrictions[0].Identifier, r.Restrictions[0].Identifier, "restriction identifier")
			assert.Equal(t, restrictedType.Restrictions[0].Location, r.Restrictions[0].Location, "restriction location")
		default:
			assert.Fail(t, "Decoded type is not *sema.RestrictionType")
		}
	})
}

func TestSemaCodecFailures(t *testing.T) {
	t.Parallel()

	t.Run("DecodeSema return error", func(t *testing.T) {
		t.Parallel()

		_, err := sema_codec.DecodeSema(nil, []byte{0xff})
		assert.ErrorContains(t, err, "unknown type", "encoding unknown type succeeded when it shouldn't have")
	})

	t.Run("Go error panic when encoding", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			var nilCompositeType *sema.CompositeType
			_, _ = sema_codec.EncodeSema(nilCompositeType)
		})
	})
}

func TestSemaCodecBadTypes(t *testing.T) {
	t.Parallel()

	t.Run("unknown type", func(t *testing.T) {
		t.Parallel()

		_, decoder, buffer := NewTestCodec()

		fakeType := byte(0xff)
		buffer.Write([]byte{fakeType})

		_, err := decoder.DecodeType()
		assert.ErrorContains(t, err, "unknown type", "encoding unknown type succeeded when it shouldn't have")
	})

	t.Run("unknown simple type", func(t *testing.T) {
		t.Parallel()

		encoder, _, _ := NewTestCodec()

		fakeSimpleType := &sema.SimpleType{}

		err := encoder.EncodeType(fakeSimpleType)
		assert.ErrorContains(t, err, "unknown simple type")
	})

	t.Run("unexpected numeric type", func(t *testing.T) {
		t.Parallel()

		encoder, _, _ := NewTestCodec()

		fakeNumericType := &sema.NumericType{}

		err := encoder.EncodeType(fakeNumericType)
		assert.ErrorContains(t, err, "unexpected numeric type")
	})

	t.Run("unexpected fixed point numeric type", func(t *testing.T) {
		t.Parallel()

		encoder, _, _ := NewTestCodec()

		fakeFixedPointNumericType := &sema.FixedPointNumericType{}

		err := encoder.EncodeType(fakeFixedPointNumericType)
		assert.ErrorContains(t, err, "unexpected fixed point numeric type")
	})

	t.Run("unexpected type", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO try to encode a fake sema.Type")
	})

	t.Run("unexpected location type", func(t *testing.T) {
		t.Parallel()
		t.Skip("TODO try to encode a fake common.Location")
	})
}

func TestSemaCodecArrayTypes(t *testing.T) {
	t.Parallel()

	t.Run("variable", func(t *testing.T) {
		t.Parallel()

		testRootEncodeDecode(
			t,
			&sema.VariableSizedType{Type: sema.CharacterType},
			byte(sema_codec.EncodedSemaVariableSizedType),
			byte(sema_codec.EncodedSemaSimpleTypeCharacterType),
		)
	})

	t.Run("constant", func(t *testing.T) {
		t.Parallel()

		testRootEncodeDecode(
			t,
			&sema.ConstantSizedType{
				Type: sema.CharacterType,
				Size: 90,
			},
			byte(sema_codec.EncodedSemaConstantSizedType),
			byte(sema_codec.EncodedSemaSimpleTypeCharacterType),
			0, 0, 0, 0, 0, 0, 0, byte(90),
		)
	})
}

func TestSemaCodecInterfaceType(t *testing.T) {
	t.Parallel()

	t.Run("custom InterfaceType", func(t *testing.T) {
		t.Parallel()

		location := common.TransactionLocation{1, 3, 9, 27, 81}

		identifier := "murakami"

		members := &sema.StringMemberOrderedMap{}
		memberIdentifer := "someID"
		memberDocString := "\"doctored\" string"
		members.Set("yolo", sema.NewPublicConstantFieldMember(
			nil,
			sema.PrivatePathType,
			memberIdentifer,
			sema.Int8Type,
			memberDocString,
		))

		fields := []string{"dance"}

		parameters := []*sema.Parameter{
			{
				Label:          "lol",
				Identifier:     "haha",
				TypeAnnotation: nil,
			},
		}

		interfaceType := &sema.InterfaceType{
			Location:              location,
			Identifier:            identifier,
			CompositeKind:         common.CompositeKindEnum,
			Members:               members,
			Fields:                fields,
			InitializerParameters: parameters,
		}

		empty := &sema.InterfaceType{
			Members: &sema.StringMemberOrderedMap{},
		}

		interfaceType.SetContainerType(empty)

		nestedTypes := &sema.StringTypeOrderedMap{}
		nestedTypes.Set("none", empty)
		interfaceType.SetNestedTypes(nestedTypes)

		encoder, decoder, buffer := NewTestCodec()

		err := encoder.EncodeType(interfaceType)
		require.NoError(t, err, "encoding error")

		expected := common_codec.Concat(
			[]byte{byte(sema_codec.EncodedSemaInterfaceType)},

			[]byte{common.TransactionLocationPrefix[0]},
			location[:],

			[]byte{0, 0, 0, byte(len(identifier))},
			[]byte(identifier),

			[]byte{byte(common.CompositeKindEnum)},

			[]byte{byte(common_codec.EncodedBoolFalse)},      // Members is not nil
			[]byte{0, 0, 0, byte(members.Len())},             // Members length
			[]byte{0, 0, 0, byte(len(members.Newest().Key))}, // Member key
			[]byte(members.Newest().Key),
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(ast.AccessPublic)}, // Member value
			[]byte{0, 0, 0, byte(len(memberIdentifer))},         // Member AST identifier
			[]byte(memberIdentifer),
			[]byte{0, 0, 0, 0, 0, 0, 0, 0}, // Member AST identifier position
			[]byte{0, 0, 0, 0, 0, 0, 0, 0},
			[]byte{0, 0, 0, 0, 0, 0, 0, 0},
			[]byte{byte(common_codec.EncodedBoolFalse)}, // Member type annotation
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{byte(sema_codec.EncodedSemaNumericTypeInt8Type)},
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(common.DeclarationKindField)}, // Member declaration kind
			[]byte{0, 0, 0, 0, 0, 0, 0, byte(ast.VariableKindConstant)},    // member variable kind
			[]byte{byte(common_codec.EncodedBoolTrue)},                     // Member has no argument labels
			[]byte{byte(common_codec.EncodedBoolFalse)},                    // Member is not predeclared
			[]byte{0, 0, 0, byte(len(memberDocString))},                    // Member doc string
			[]byte(memberDocString),

			[]byte{byte(common_codec.EncodedBoolFalse)}, // Fields array is not nil
			[]byte{0, 0, 0, byte(len(fields))},
			[]byte{0, 0, 0, byte(len(fields[0]))},
			[]byte(fields[0]),

			[]byte{byte(common_codec.EncodedBoolFalse)}, // InitializerParameters array is not nil
			[]byte{0, 0, 0, byte(len(parameters))},
			[]byte{0, 0, 0, byte(len(parameters[0].Label))},
			[]byte(parameters[0].Label),
			[]byte{0, 0, 0, byte(len(parameters[0].Identifier))},
			[]byte(parameters[0].Identifier),
			[]byte{byte(common_codec.EncodedBoolTrue)},

			[]byte{byte(sema_codec.EncodedSemaInterfaceType)}, // container type is empty interface
			[]byte{common_codec.NilLocationPrefix[0]},
			[]byte{0, 0, 0, 0},
			[]byte{byte(common.CompositeKindUnknown)},
			[]byte{byte(common_codec.EncodedBoolFalse)},
			[]byte{0, 0, 0, 0},
			[]byte{byte(common_codec.EncodedBoolTrue)},
			[]byte{byte(common_codec.EncodedBoolTrue)},
			[]byte{byte(sema_codec.EncodedSemaNilType)},
			[]byte{byte(common_codec.EncodedBoolTrue)},

			[]byte{byte(common_codec.EncodedBoolFalse)}, // nested type
			[]byte{0, 0, 0, 1},
			[]byte{0, 0, 0, 4},
			[]byte("none"),
			[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 0xb4}, // nested type is also container type
		)

		assert.Equal(t, expected, buffer.Bytes(), "encoded bytes differ")

		decoded, err := decoder.DecodeType()
		require.NoError(t, err, "decoding error")

		// Cannot simply check equality between original and decoded types because they are not shallowly equal.
		// Specifically, RequiredArgumentCount and Members are not shallowly equal.
		switch i := decoded.(type) {
		case *sema.InterfaceType:
			assert.Equal(t, location, i.Location, "location")

			assert.Equal(t, identifier, i.Identifier, "identifier")

			assert.Equal(t, common.CompositeKindEnum, i.CompositeKind, "composite kind")

			// verify member equality
			require.Equal(t, members.Len(), i.Members.Len(), "members length")
			i.Members.Foreach(func(key string, actual *sema.Member) {
				expected, present := i.Members.Get(key)
				require.True(t, present, "extra member: %s", key)

				assert.Equal(t, expected.ContainerType.ID(), actual.ContainerType.ID(), "container type for %s", key)
				assert.Equal(t, expected.TypeAnnotation.QualifiedString(), actual.TypeAnnotation.QualifiedString(), "type annotation for %s", key)
			})

			assert.Equal(t, fields, i.Fields, "fields")

			assert.Equal(t, parameters, i.InitializerParameters, "parameters")

			assert.Equal(t, i.GetContainerType(), empty, "container type")
			assert.Equal(t, i.GetNestedTypes(), nestedTypes, "nested types")
		default:
			assert.Fail(t, "Decoded type is not *sema.InterfaceType")
		}
	})
}

func TestSemaCodecCompositeType(t *testing.T) {
	t.Parallel()

	t.Run("AuthAccountType (IsContainerType=true)", func(t *testing.T) {
		t.Parallel()

		ty := sema.AuthAccountType

		encoder, decoder, _ := NewTestCodec()

		err := encoder.EncodeType(ty)
		require.NoError(t, err, "encoding error")

		decoded, err := decoder.DecodeType()
		require.NoError(t, err, "decoding error")

		switch d := decoded.(type) {
		case *sema.CompositeType:
			assert.Equal(t, true, d.IsContainerType(), "IsContainerType")
		default:
			assert.Fail(t, "decoded type is not *sema.CompositeType")
		}

	})

	t.Run("AccountKeyType", func(t *testing.T) {
		t.Parallel()

		theCompositeType := sema.AccountKeyType

		encoder, buffer := NewTestEncoder()
		err := encoder.EncodeType(theCompositeType)
		require.NoError(t, err, "encoding error")

		// verify the first few encoded bytes
		expected := []byte{
			// type of encoded sema type
			byte(sema_codec.EncodedSemaCompositeType),

			// location
			common_codec.NilLocationPrefix[0],

			// length of identifier
			0, 0, 0,
			byte(len(sema.AccountKeyTypeName)),

			// identifier
			sema.AccountKeyTypeName[0],
			sema.AccountKeyTypeName[1],
			sema.AccountKeyTypeName[2],
			sema.AccountKeyTypeName[3],
			sema.AccountKeyTypeName[4],
			sema.AccountKeyTypeName[5],
			sema.AccountKeyTypeName[6],
			sema.AccountKeyTypeName[7],
			sema.AccountKeyTypeName[8],
			sema.AccountKeyTypeName[9],

			// composite kind
			byte(common.CompositeKindStructure),

			// ExplicitInterfaceConformances array is nil
			byte(common_codec.EncodedBoolTrue),

			// ImplicitTypeRequirementConformances array is nil
			byte(common_codec.EncodedBoolTrue),
		}
		assert.Equal(t, expected, buffer.Bytes()[:len(expected)], "encoded bytes")

		decoder := sema_codec.NewSemaDecoder(nil, buffer)
		output, err := decoder.DecodeType()
		require.NoError(t, err)

		// populates `cachedIdentifiers` for top-level and its members
		output.QualifiedString()
		switch c := output.(type) {
		case *sema.CompositeType:
			c.Members.Foreach(func(key string, value *sema.Member) {
				value.TypeAnnotation.QualifiedString()
			})
		}

		// verify Equal(...) method equality... basically a smoke test
		assert.True(t, output.Equal(theCompositeType), ".Equal(...) is false")

		switch c := output.(type) {
		case *sema.CompositeType:
			// verify the easily verified
			assert.Equal(t, theCompositeType.Fields, c.Fields)
			assert.Equal(t, theCompositeType.Kind, c.Kind)
			assert.Equal(t, theCompositeType.Location, c.Location)
			assert.Equal(t, theCompositeType.EnumRawType, c.EnumRawType)
			assert.Equal(t, theCompositeType.Identifier, c.Identifier)
			assert.Equal(t, theCompositeType.ImportableWithoutLocation, c.ImportableWithoutLocation)
			assert.Equal(t, theCompositeType.ConstructorParameters, c.ConstructorParameters)
			assert.Equal(t, theCompositeType.ExplicitInterfaceConformances, c.ExplicitInterfaceConformances)
			assert.Equal(t, theCompositeType.ImplicitTypeRequirementConformances, c.ImplicitTypeRequirementConformances)
			assert.Equal(t, theCompositeType.GetContainerType(), c.GetContainerType())
			assert.Equal(t, theCompositeType.GetNestedTypes(), c.GetNestedTypes())

			// verify member equality
			// note that only 3/5 of members are serializable so the encoded type has only 3 members
			require.Equal(t, 3, c.Members.Len(), "members length")
			c.Members.Foreach(func(key string, actual *sema.Member) {
				expected, present := theCompositeType.Members.Get(key)
				require.True(t, present, "extra member: %s", key)

				assert.Equal(t, expected.ContainerType.ID(), actual.ContainerType.ID(), "container type for %s", key)
				assert.Equal(t, expected.TypeAnnotation.QualifiedString(), actual.TypeAnnotation.QualifiedString(), "type annotation for %s", key)
			})
		default:
			require.Fail(t, "decoded type is not CompositeType")
		}
	})
}

func TestSemaCodecRecursiveType(t *testing.T) {
	t.Parallel()

	t.Run("CompositeType", func(t *testing.T) {
		t.Parallel()

		c := &sema.CompositeType{}
		c.SetContainerType(c)

		encoder, decoder, buffer := NewTestCodec()

		err := encoder.EncodeType(c)
		require.NoError(t, err, "encoding error")

		expected := []byte{
			byte(sema_codec.EncodedSemaCompositeType),
			common_codec.NilLocationPrefix[0],
			0, 0, 0, 0, // identifier length
			0,                                                   // composite kind
			byte(common_codec.EncodedBoolTrue),                  // ExplicitInterfaceConformances array is nil
			byte(common_codec.EncodedBoolTrue),                  // ImplicitTypeRequirementConformances array is nil
			byte(common_codec.EncodedBoolTrue),                  // no members
			byte(common_codec.EncodedBoolTrue),                  // Fields array is nil
			byte(common_codec.EncodedBoolTrue),                  // ConstructorParameters array is nil
			byte(common_codec.EncodedBoolTrue),                  // no nested types
			byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1, // container type
			byte(sema_codec.EncodedSemaNilType), // EnumRawType
			byte(common_codec.EncodedBoolFalse), // hasComputedMembers
			byte(common_codec.EncodedBoolFalse), // ImportableWithoutLocation
		}

		assert.Equal(t, expected, buffer.Bytes(), "encoded bytes differ")

		decoded, err := decoder.DecodeType()
		require.NoError(t, err, "decoding error")

		// Cannot simply check equality between original and decoded types because they are not shallowly equal.
		switch cc := decoded.(type) {
		case *sema.CompositeType:
			assert.Equal(t, cc, cc.GetContainerType(), "container is self")
		default:
			assert.Fail(t, "Decoded type is not *sema.CompositeType")
		}
	})

	t.Run("InterfaceType", func(t *testing.T) {
		t.Parallel()

		c := &sema.InterfaceType{}
		c.SetContainerType(c)

		encoder, decoder, buffer := NewTestCodec()

		err := encoder.EncodeType(c)
		require.NoError(t, err, "encoding error")

		expected := []byte{
			byte(sema_codec.EncodedSemaInterfaceType),
			common_codec.NilLocationPrefix[0],
			0, 0, 0, 0, // identifier length
			0,                                                   // composite kind
			byte(common_codec.EncodedBoolTrue),                  // Members array is nil
			byte(common_codec.EncodedBoolTrue),                  // Fields array is nil
			byte(common_codec.EncodedBoolTrue),                  // InitializerParameters array is nil
			byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1, // container type
			byte(common_codec.EncodedBoolTrue), // nestedTypes
		}

		assert.Equal(t, expected, buffer.Bytes(), "encoded bytes differ")

		decoded, err := decoder.DecodeType()
		require.NoError(t, err, "decoding error")

		// Cannot simply check equality between original and decoded types because they are not shallowly equal.
		switch cc := decoded.(type) {
		case *sema.InterfaceType:
			assert.Equal(t, cc, cc.GetContainerType(), "container is self")
		default:
			assert.Fail(t, "Decoded type is not *sema.InterfaceType")
		}
	})

	t.Run("GenericType", func(t *testing.T) {
		t.Parallel()

		parent := &sema.GenericType{} // extra layer to test non-zero pointer

		g := &sema.GenericType{}
		g.TypeParameter = &sema.TypeParameter{
			Name:      "nomen",
			TypeBound: g,
			Optional:  true,
		}

		parent.TypeParameter = &sema.TypeParameter{
			Name:      "parent",
			TypeBound: g,
			Optional:  false,
		}

		testRootEncodeDecode(
			t,
			parent,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaGenericType)},
				[]byte{0, 0, 0, byte(len(parent.TypeParameter.Name))},
				[]byte(parent.TypeParameter.Name),
				[]byte{byte(sema_codec.EncodedSemaGenericType)},
				[]byte{0, 0, 0, byte(len(g.TypeParameter.Name))},
				[]byte(g.TypeParameter.Name),
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 12},
				[]byte{byte(common_codec.EncodedBoolTrue)},
				[]byte{byte(common_codec.EncodedBoolFalse)},
			)...,
		)
	})

	t.Run("FunctionType", func(t *testing.T) {
		t.Parallel()

		f := &sema.FunctionType{
			IsConstructor:            false,
			TypeParameters:           nil,
			Parameters:               nil,
			ReturnTypeAnnotation:     nil,
			RequiredArgumentCount:    nil,
			ArgumentExpressionsCheck: nil,
			Members:                  nil,
		}
		f.TypeParameters = []*sema.TypeParameter{{
			Name:      "nome",
			TypeBound: f,
			Optional:  false,
		}}

		testRootEncodeDecode(
			t,
			f,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaFunctionType)},
				[]byte{byte(common_codec.EncodedBoolFalse)}, // isConstructor
				[]byte{byte(common_codec.EncodedBoolFalse)}, // TypeParameters is not nil
				[]byte{0, 0, 0, 1},
				[]byte{0, 0, 0, byte(len(f.TypeParameters[0].Name))},
				[]byte(f.TypeParameters[0].Name),
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // container type
				[]byte{byte(common_codec.EncodedBoolFalse)},
				[]byte{byte(common_codec.EncodedBoolTrue)}, // Parameters is nil
				[]byte{byte(common_codec.EncodedBoolTrue)}, // ReturnTypeAnnotation is nil
				[]byte{byte(common_codec.EncodedBoolTrue)}, // RequiredArgumentCount is nil
				[]byte{byte(common_codec.EncodedBoolTrue)}, // Members is nil
			)...,
		)
	})

	t.Run("DictionaryType", func(t *testing.T) {
		t.Parallel()

		d := &sema.DictionaryType{}
		d.KeyType = d
		d.ValueType = d

		testRootEncodeDecode(
			t,
			d,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaDictionaryType)},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1},
			)...,
		)
	})

	t.Run("TransactionType", func(t *testing.T) {
		t.Parallel()

		tx := &sema.TransactionType{
			Members:           nil,
			Fields:            nil,
			PrepareParameters: nil,
			Parameters:        nil,
		}
		tx.Parameters = []*sema.Parameter{{
			Label:      "momentary",
			Identifier: "fade",
		}}
		tx.Parameters[0].TypeAnnotation = sema.NewTypeAnnotation(tx)

		testRootEncodeDecode(
			t,
			tx,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaTransactionType)},
				[]byte{byte(common_codec.EncodedBoolTrue)},  // Members is nil
				[]byte{byte(common_codec.EncodedBoolTrue)},  // Fields is nil
				[]byte{byte(common_codec.EncodedBoolTrue)},  // PrepareParameters is nil
				[]byte{byte(common_codec.EncodedBoolFalse)}, // Parameters is not nil
				[]byte{0, 0, 0, 1},                          // 1 Parameter
				[]byte{0, 0, 0, byte(len(tx.Parameters[0].Label))},
				[]byte(tx.Parameters[0].Label),
				[]byte{0, 0, 0, byte(len(tx.Parameters[0].Identifier))},
				[]byte(tx.Parameters[0].Identifier),
				[]byte{byte(common_codec.EncodedBoolFalse)},                 // TypeAnnotation is not nil
				[]byte{byte(common_codec.EncodedBoolFalse)},                 // TypeAnnotation is not a resource
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // type is recursive
			)...,
		)
	})

	t.Run("RestrictedType", func(t *testing.T) {
		t.Parallel()

		r := &sema.RestrictedType{}
		r.Type = r

		testRootEncodeDecode(
			t,
			r,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaRestrictedType)},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // type is recursive
				[]byte{byte(common_codec.EncodedBoolTrue)},                  // Restrictions is nil
			)...,
		)
	})

	t.Run("ConstantSizedType", func(t *testing.T) {
		t.Parallel()

		c := &sema.ConstantSizedType{}
		c.Type = c

		testRootEncodeDecode(
			t,
			c,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaConstantSizedType)},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // type is recursive
				[]byte{0, 0, 0, 0, 0, 0, 0, 0},
			)...,
		)
	})

	t.Run("VariableSizedType", func(t *testing.T) {
		t.Parallel()

		v := &sema.VariableSizedType{}
		v.Type = v

		testRootEncodeDecode(
			t,
			v,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaVariableSizedType)},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // type is recursive
			)...,
		)
	})

	t.Run("OptionalType", func(t *testing.T) {
		t.Parallel()

		o := &sema.OptionalType{}
		o.Type = o

		testRootEncodeDecode(
			t,
			o,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaOptionalType)},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // type is recursive
			)...,
		)
	})

	t.Run("ReferenceType", func(t *testing.T) {
		t.Parallel()

		r := &sema.ReferenceType{}
		r.Type = r

		testRootEncodeDecode(
			t,
			r,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaReferenceType)},
				[]byte{byte(common_codec.EncodedBoolFalse)},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // type is recursive
			)...,
		)
	})

	t.Run("CapabilityType", func(t *testing.T) {
		t.Parallel()

		v := &sema.CapabilityType{}
		v.BorrowType = v

		testRootEncodeDecode(
			t,
			v,
			common_codec.Concat(
				[]byte{byte(sema_codec.EncodedSemaCapabilityType)},
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 1}, // type is recursive
			)...,
		)
	})
}

//
// Elaboration
//

func TestSemaCodecElaboration(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		el := sema.NewElaboration(nil, false)

		encoder, decoder, buffer := NewTestCodec()

		testEncodeDecode(
			t,
			el,
			buffer,
			encoder.EncodeElaboration,
			decoder.DecodeElaboration,
			[]byte{
				0, 0, 0, 0, // length of composite types
				0, 0, 0, 0, // length of interface types
			},
		)
	})

	t.Run("full", func(t *testing.T) {
		t.Parallel()

		typeId := common.TypeID("houses")
		location := common.ScriptLocation{9, 3, 1}
		identifier := "valence"
		kind := common.CompositeKindStructure

		compType := &sema.CompositeType{
			Location:   location,
			Identifier: identifier,
			Kind:       kind,
		}
		compType.SetContainerType(compType) // test recursive type

		el := sema.NewElaboration(nil, false)
		el.CompositeTypes[typeId] = compType
		el.InterfaceTypes[typeId] = compType.InterfaceType()

		encoder, decoder, buffer := NewTestCodec()

		testEncodeDecode(
			t,
			el,
			buffer,
			encoder.EncodeElaboration,
			decoder.DecodeElaboration,
			common_codec.Concat(
				[]byte{0, 0, 0, 1},                 // length of CompositeTypes map
				[]byte{0, 0, 0, byte(len(typeId))}, // TypeID aka map key
				[]byte(typeId),
				[]byte{common.ScriptLocationPrefix[0]}, // location
				location[:],
				[]byte{0, 0, 0, byte(len(identifier))}, // identifier
				[]byte(identifier),
				[]byte{byte(common.CompositeKindStructure)}, // composite kind
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil ExplicitInterfaceConformances
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil ImplicitTypeRequirementConformances
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil Members
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil Fields
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil ConstructorParameters
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil nestedTypes
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 14},
				[]byte{byte(sema_codec.EncodedSemaNilType)}, // nil EnumRawType
				[]byte{byte(common_codec.EncodedBoolFalse)}, // hasComputedMembers
				[]byte{byte(common_codec.EncodedBoolFalse)}, // ImportableWithoutLocation

				[]byte{0, 0, 0, 1},                 // length of InterfaceTypes map
				[]byte{0, 0, 0, byte(len(typeId))}, // TypeID aka map key
				[]byte(typeId),
				[]byte{common.ScriptLocationPrefix[0]}, // location
				location[:],
				[]byte{0, 0, 0, byte(len(identifier))}, // identifier
				[]byte(identifier),
				[]byte{byte(common.CompositeKindStructure)}, // composite kind
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil Members
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil Fields
				[]byte{byte(common_codec.EncodedBoolTrue)},  // nil InitializerParameters
				[]byte{byte(sema_codec.EncodedSemaPointerType), 0, 0, 0, 14},
				[]byte{byte(common_codec.EncodedBoolTrue)}, // nil nestedTypes

			),
		)
	})
}

//
// Helpers
//

func testRootEncodeDecode(
	t *testing.T,
	input sema.Type,
	expectedEncoding ...byte,
) ([]byte, sema.Type) {
	blob, err := sema_codec.EncodeSema(input)
	require.NoError(t, err, "encoding error")

	if expectedEncoding != nil {
		assert.Equal(t, expectedEncoding, blob)
	}

	output, err := sema_codec.DecodeSema(nil, blob)
	require.NoError(t, err, "decoding error")

	assert.Equal(t, input, output, "decoded message differs from input")

	return blob, output
}

func testEncodeDecode[T any](
	t *testing.T,
	input T,
	buffer *bytes.Buffer,
	encode func(T) error,
	decode func() (T, error),
	expectedEncoding []byte,
) {
	err := encode(input)
	require.NoError(t, err, "encoding error")

	if expectedEncoding != nil {
		assert.Equal(t, expectedEncoding, buffer.Bytes(), "encoded bytes differ from expectation")
	}

	output, err := decode()
	require.NoError(t, err, "decoding error")

	assert.Equal(t, input, output, "decoded data structure differs from expectation")
}

func NewTestEncoder() (*sema_codec.SemaEncoder, *bytes.Buffer) {
	var w bytes.Buffer
	encoder := sema_codec.NewSemaEncoder(&w)
	return encoder, &w
}

func NewTestCodec() (encoder *sema_codec.SemaEncoder, decoder *sema_codec.SemaDecoder, buffer *bytes.Buffer) {
	var w bytes.Buffer
	buffer = &w
	encoder = sema_codec.NewSemaEncoder(buffer)
	decoder = sema_codec.NewSemaDecoder(nil, buffer)
	return
}

// TODO test via fuzzing
