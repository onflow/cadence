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
	_ "embed"
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestExportValue(t *testing.T) {

	t.Parallel()

	type exportTest struct {
		value    interpreter.Value
		expected cadence.Value
		// Some values need an interpreter to be created (e.g. stored values like arrays, dictionaries, and composites),
		// so provide an optional helper function to construct the value
		valueFactory func(*interpreter.Interpreter) interpreter.Value
		label        string
		invalid      bool
	}

	test := func(tt exportTest) {

		t.Run(tt.label, func(t *testing.T) {

			t.Parallel()

			inter := newTestInterpreter(t)

			value := tt.value
			if tt.valueFactory != nil {
				value = tt.valueFactory(inter)
			}
			actual, err := exportValueWithInterpreter(
				value,
				inter,
				interpreter.EmptyLocationRange,
				seenReferences{},
			)

			if tt.invalid {
				RequireError(t, err)
				assertUserError(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			}
		})
	}

	newSignatureAlgorithmType := func() *cadence.EnumType {
		return &cadence.EnumType{
			QualifiedIdentifier: "SignatureAlgorithm",
			RawType:             cadence.UInt8Type{},
			Fields: []cadence.Field{
				{
					Identifier: "rawValue",
					Type:       cadence.UInt8Type{},
				},
			},
		}
	}

	newPublicKeyType := func(signatureAlgorithmType cadence.Type) *cadence.StructType {
		return &cadence.StructType{
			QualifiedIdentifier: "PublicKey",
			Fields: []cadence.Field{
				{
					Identifier: "publicKey",
					Type: &cadence.VariableSizedArrayType{
						ElementType: cadence.UInt8Type{},
					},
				},
				{
					Identifier: "signatureAlgorithm",
					Type:       signatureAlgorithmType,
				},
			},
		}
	}

	newHashAlgorithmType := func() *cadence.EnumType {
		return &cadence.EnumType{
			QualifiedIdentifier: "HashAlgorithm",
			RawType:             cadence.UInt8Type{},
			Fields: []cadence.Field{
				{
					Identifier: "rawValue",
					Type:       cadence.UInt8Type{},
				},
			},
		}
	}

	testCharacter, _ := cadence.NewCharacter("a")

	testFunction := &interpreter.InterpretedFunctionValue{
		Type: &sema.FunctionType{
			ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.VoidType),
		},
	}

	testFunctionType := cadence.NewFunctionType(
		nil,
		nil,
		cadence.VoidType{},
	)

	for _, tt := range []exportTest{
		{
			label:    "Void",
			value:    interpreter.Void,
			expected: cadence.NewVoid(),
		},
		{
			label:    "Nil",
			value:    interpreter.Nil,
			expected: cadence.NewOptional(nil),
		},
		{
			label: "SomeValue",
			value: interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(42),
			),
			expected: cadence.NewOptional(cadence.NewInt(42)),
		},
		{
			label:    "Bool true",
			value:    interpreter.TrueValue,
			expected: cadence.NewBool(true),
		},

		{
			label:    "Bool false",
			value:    interpreter.FalseValue,
			expected: cadence.NewBool(false),
		},

		{
			label:    "String empty",
			value:    interpreter.NewUnmeteredStringValue(""),
			expected: cadence.String(""),
		},
		{
			label:    "String non-empty",
			value:    interpreter.NewUnmeteredStringValue("foo"),
			expected: cadence.String("foo"),
		},
		{
			label: "Array empty",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					common.ZeroAddress,
				)
			},
			expected: cadence.NewArray([]cadence.Value{}).
				WithType(&cadence.VariableSizedArrayType{
					ElementType: cadence.AnyStructType{},
				}),
		},
		{
			label: "Array (non-empty)",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					common.ZeroAddress,
					interpreter.NewUnmeteredIntValueFromInt64(42),
					interpreter.NewUnmeteredStringValue("foo"),
				)
			},
			expected: cadence.NewArray([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: cadence.AnyStructType{},
			}),
		},
		{
			label: "Dictionary",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeString,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
				)
			},
			expected: cadence.NewDictionary([]cadence.KeyValuePair{}).
				WithType(&cadence.DictionaryType{
					KeyType:     cadence.StringType{},
					ElementType: cadence.AnyStructType{},
				}),
		},
		{
			label: "Dictionary (non-empty)",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeString,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					interpreter.NewUnmeteredStringValue("a"),
					interpreter.NewUnmeteredIntValueFromInt64(1),
					interpreter.NewUnmeteredStringValue("b"),
					interpreter.NewUnmeteredIntValueFromInt64(2),
				)
			},
			expected: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("b"),
					Value: cadence.NewInt(2),
				},
				{
					Key:   cadence.String("a"),
					Value: cadence.NewInt(1),
				},
			}).
				WithType(&cadence.DictionaryType{
					KeyType:     cadence.StringType{},
					ElementType: cadence.AnyStructType{},
				}),
		},
		{
			label:    "Address",
			value:    interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
			expected: cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
		},
		{
			label:    "Int",
			value:    interpreter.NewUnmeteredIntValueFromInt64(42),
			expected: cadence.NewInt(42),
		},
		{
			label:    "Character",
			value:    interpreter.NewUnmeteredCharacterValue("a"),
			expected: testCharacter,
		},
		{
			label:    "Int8",
			value:    interpreter.NewUnmeteredInt8Value(42),
			expected: cadence.NewInt8(42),
		},
		{
			label:    "Int16",
			value:    interpreter.NewUnmeteredInt16Value(42),
			expected: cadence.NewInt16(42),
		},
		{
			label:    "Int32",
			value:    interpreter.NewUnmeteredInt32Value(42),
			expected: cadence.NewInt32(42),
		},
		{
			label:    "Int64",
			value:    interpreter.NewUnmeteredInt64Value(42),
			expected: cadence.NewInt64(42),
		},
		{
			label:    "Int128",
			value:    interpreter.NewUnmeteredInt128ValueFromInt64(42),
			expected: cadence.NewInt128(42),
		},
		{
			label:    "Int256",
			value:    interpreter.NewUnmeteredInt256ValueFromInt64(42),
			expected: cadence.NewInt256(42),
		},
		{
			label:    "UInt",
			value:    interpreter.NewUnmeteredUIntValueFromUint64(42),
			expected: cadence.NewUInt(42),
		},
		{
			label:    "UInt8",
			value:    interpreter.NewUnmeteredUInt8Value(42),
			expected: cadence.NewUInt8(42),
		},
		{
			label:    "UInt16",
			value:    interpreter.NewUnmeteredUInt16Value(42),
			expected: cadence.NewUInt16(42),
		},
		{
			label:    "UInt32",
			value:    interpreter.NewUnmeteredUInt32Value(42),
			expected: cadence.NewUInt32(42),
		},
		{
			label:    "UInt64",
			value:    interpreter.NewUnmeteredUInt64Value(42),
			expected: cadence.NewUInt64(42),
		},
		{
			label:    "UInt128",
			value:    interpreter.NewUnmeteredUInt128ValueFromUint64(42),
			expected: cadence.NewUInt128(42),
		},
		{
			label:    "UInt256",
			value:    interpreter.NewUnmeteredUInt256ValueFromUint64(42),
			expected: cadence.NewUInt256(42),
		},
		{
			label:    "Word8",
			value:    interpreter.NewUnmeteredWord8Value(42),
			expected: cadence.NewWord8(42),
		},
		{
			label:    "Word16",
			value:    interpreter.NewUnmeteredWord16Value(42),
			expected: cadence.NewWord16(42),
		},
		{
			label:    "Word32",
			value:    interpreter.NewUnmeteredWord32Value(42),
			expected: cadence.NewWord32(42),
		},
		{
			label:    "Word64",
			value:    interpreter.NewUnmeteredWord64Value(42),
			expected: cadence.NewWord64(42),
		},
		{
			label:    "Word128",
			value:    interpreter.NewUnmeteredWord128ValueFromUint64(42),
			expected: cadence.NewWord128(42),
		},
		{
			label:    "Word256",
			value:    interpreter.NewUnmeteredWord256ValueFromUint64(42),
			expected: cadence.NewWord256(42),
		},
		{
			label:    "Fix64",
			value:    interpreter.NewUnmeteredFix64Value(-123000000),
			expected: cadence.Fix64(-123000000),
		},
		{
			label:    "UFix64",
			value:    interpreter.NewUnmeteredUFix64Value(123000000),
			expected: cadence.UFix64(123000000),
		},
		{
			label: "Path",
			value: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			expected: cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
		},
		{
			label: "Interpreted Function",
			value: testFunction,
			expected: cadence.Function{
				FunctionType: testFunctionType,
			},
		},
		{
			label: "Host Function",
			value: &interpreter.HostFunctionValue{
				Type: testFunction.Type,
			},
			expected: cadence.Function{
				FunctionType: testFunctionType,
			},
		},
		{
			label: "Bound Function",
			value: interpreter.BoundFunctionValue{
				Function: testFunction,
			},
			expected: cadence.Function{
				FunctionType: testFunctionType,
			},
		},
		{
			label: "Account key",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				hashAlgorithm, _ := stdlib.NewHashAlgorithmCase(1, nil)

				return interpreter.NewAccountKeyValue(
					inter,
					interpreter.NewUnmeteredIntValueFromInt64(1),
					stdlib.NewPublicKeyValue(
						inter,
						interpreter.EmptyLocationRange,
						&stdlib.PublicKey{
							PublicKey: []byte{1, 2, 3},
							SignAlgo:  2,
						},
						nil,
						nil,
					),
					hashAlgorithm,
					interpreter.NewUnmeteredUFix64ValueWithInteger(10, interpreter.EmptyLocationRange),
					false,
				)
			},
			expected: func() cadence.Value {

				signatureAlgorithmType := newSignatureAlgorithmType()
				publicKeyType := newPublicKeyType(signatureAlgorithmType)
				hashAlgorithmType := newHashAlgorithmType()

				return cadence.Struct{
					StructType: &cadence.StructType{
						QualifiedIdentifier: "AccountKey",
						Fields: []cadence.Field{
							{
								Identifier: "keyIndex",
								Type:       cadence.IntType{},
							},
							{
								Identifier: "publicKey",
								Type:       publicKeyType,
							},
							{
								Identifier: "hashAlgorithm",
								Type:       hashAlgorithmType,
							},
							{
								Identifier: "weight",
								Type:       cadence.UFix64Type{},
							},
							{
								Identifier: "isRevoked",
								Type:       cadence.BoolType{},
							},
						},
					},
					Fields: []cadence.Value{
						cadence.NewInt(1),
						cadence.Struct{
							StructType: publicKeyType,
							Fields: []cadence.Value{
								cadence.NewArray([]cadence.Value{
									cadence.NewUInt8(1),
									cadence.NewUInt8(2),
									cadence.NewUInt8(3),
								}).WithType(&cadence.VariableSizedArrayType{
									ElementType: cadence.UInt8Type{},
								}),
								cadence.Enum{
									EnumType: signatureAlgorithmType,
									Fields: []cadence.Value{
										cadence.UInt8(2),
									},
								},
							},
						},
						cadence.Enum{
							EnumType: hashAlgorithmType,
							Fields: []cadence.Value{
								cadence.UInt8(1),
							},
						},
						cadence.UFix64(10_00000000),
						cadence.Bool(false),
					},
				}
			}(),
		},
		{
			label: "Deployed contract (invalid)",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewDeployedContractValue(
					inter,
					interpreter.AddressValue{},
					interpreter.NewUnmeteredStringValue("C"),
					interpreter.NewArrayValue(
						inter,
						interpreter.EmptyLocationRange,
						interpreter.ByteArrayStaticType,
						common.ZeroAddress,
					),
				)
			},
			invalid: true,
		},
		{
			label: "Block (invalid)",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				blockIDStaticType :=
					interpreter.ConvertSemaToStaticType(nil, sema.BlockTypeIdFieldType).(interpreter.ArrayStaticType)

				return interpreter.NewBlockValue(
					inter,
					interpreter.NewUnmeteredUInt64Value(1),
					interpreter.NewUnmeteredUInt64Value(2),
					interpreter.NewArrayValue(
						inter,
						interpreter.EmptyLocationRange,
						blockIDStaticType,
						common.ZeroAddress,
					),
					interpreter.NewUnmeteredUFix64ValueWithInteger(1, interpreter.EmptyLocationRange),
				)
			},
			invalid: true,
		},
	} {
		test(tt)
	}

}

func TestImportValue(t *testing.T) {

	t.Parallel()

	type importTest struct {
		expected     interpreter.Value
		value        cadence.Value
		expectedType sema.Type
		label        string
	}

	test := func(tt importTest) {

		t.Run(tt.label, func(t *testing.T) {

			t.Parallel()

			inter := newTestInterpreter(t)

			actual, err := ImportValue(
				inter,
				interpreter.EmptyLocationRange,
				nil,
				tt.value,
				tt.expectedType,
			)

			if tt.expected == nil {
				RequireError(t, err)
				assertUserError(t, err)
			} else {
				require.NoError(t, err)
				AssertValuesEqual(t, inter, tt.expected, actual)
			}
		})
	}

	a, _ := cadence.NewCharacter("a")

	for _, tt := range []importTest{
		{
			label:    "Void",
			expected: interpreter.Void,
			value:    cadence.NewVoid(),
		},
		{
			label:    "Nil",
			value:    cadence.NewOptional(nil),
			expected: interpreter.Nil,
		},
		{
			label: "SomeValue",
			value: cadence.NewOptional(cadence.NewInt(42)),
			expected: interpreter.NewUnmeteredSomeValueNonCopying(
				interpreter.NewUnmeteredIntValueFromInt64(42),
			),
		},
		{
			label:    "Bool true",
			value:    cadence.NewBool(true),
			expected: interpreter.TrueValue,
		},
		{
			label:    "Bool false",
			expected: interpreter.FalseValue,
			value:    cadence.NewBool(false),
		},
		{
			label:    "String empty",
			value:    cadence.String(""),
			expected: interpreter.NewUnmeteredStringValue(""),
		},
		{
			label:    "String non-empty",
			value:    cadence.String("foo"),
			expected: interpreter.NewUnmeteredStringValue("foo"),
		},
		{
			label: "Array empty",
			value: cadence.NewArray([]cadence.Value{}),
			expected: interpreter.NewArrayValue(
				newTestInterpreter(t),
				interpreter.EmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				common.ZeroAddress,
			),
			expectedType: &sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
		},
		{
			label: "Array non-empty",
			value: cadence.NewArray([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}),
			expected: interpreter.NewArrayValue(
				newTestInterpreter(t),
				interpreter.EmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				interpreter.NewUnmeteredStringValue("foo"),
			),
			expectedType: &sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
		},
		{
			label: "Dictionary",
			expected: interpreter.NewDictionaryValue(
				newTestInterpreter(t),
				interpreter.EmptyLocationRange,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
				},
			),
			value: cadence.NewDictionary([]cadence.KeyValuePair{}),
			expectedType: &sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.AnyStructType,
			},
		},
		{
			label: "Dictionary (non-empty)",
			expected: interpreter.NewDictionaryValue(
				newTestInterpreter(t),
				interpreter.EmptyLocationRange,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				interpreter.NewUnmeteredStringValue("a"),
				interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredStringValue("b"),
				interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			value: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("a"),
					Value: cadence.NewInt(1),
				},
				{
					Key:   cadence.String("b"),
					Value: cadence.NewInt(2),
				},
			}),
			expectedType: &sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.AnyStructType,
			},
		},
		{
			label:    "Address",
			expected: interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
			value:    cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 0, 0, 1}),
		},
		{
			label:    "Int",
			value:    cadence.NewInt(42),
			expected: interpreter.NewUnmeteredIntValueFromInt64(42),
		},
		{
			label:    "Character",
			value:    a,
			expected: interpreter.NewUnmeteredCharacterValue("a"),
		},
		{
			label:    "Int8",
			value:    cadence.NewInt8(42),
			expected: interpreter.NewUnmeteredInt8Value(42),
		},
		{
			label:    "Int16",
			value:    cadence.NewInt16(42),
			expected: interpreter.NewUnmeteredInt16Value(42),
		},
		{
			label:    "Int32",
			value:    cadence.NewInt32(42),
			expected: interpreter.NewUnmeteredInt32Value(42),
		},
		{
			label:    "Int64",
			value:    cadence.NewInt64(42),
			expected: interpreter.NewUnmeteredInt64Value(42),
		},
		{
			label:    "Int128",
			value:    cadence.NewInt128(42),
			expected: interpreter.NewUnmeteredInt128ValueFromInt64(42),
		},
		{
			label:    "Int256",
			value:    cadence.NewInt256(42),
			expected: interpreter.NewUnmeteredInt256ValueFromInt64(42),
		},
		{
			label:    "UInt",
			value:    cadence.NewUInt(42),
			expected: interpreter.NewUnmeteredUIntValueFromUint64(42),
		},
		{
			label:    "UInt8",
			value:    cadence.NewUInt8(42),
			expected: interpreter.NewUnmeteredUInt8Value(42),
		},
		{
			label:    "UInt16",
			value:    cadence.NewUInt16(42),
			expected: interpreter.NewUnmeteredUInt16Value(42),
		},
		{
			label:    "UInt32",
			value:    cadence.NewUInt32(42),
			expected: interpreter.NewUnmeteredUInt32Value(42),
		},
		{
			label:    "UInt64",
			value:    cadence.NewUInt64(42),
			expected: interpreter.NewUnmeteredUInt64Value(42),
		},
		{
			label:    "UInt128",
			value:    cadence.NewUInt128(42),
			expected: interpreter.NewUnmeteredUInt128ValueFromUint64(42),
		},
		{
			label:    "UInt256",
			value:    cadence.NewUInt256(42),
			expected: interpreter.NewUnmeteredUInt256ValueFromUint64(42),
		},
		{
			label:    "Word8",
			value:    cadence.NewWord8(42),
			expected: interpreter.NewUnmeteredWord8Value(42),
		},
		{
			label:    "Word16",
			value:    cadence.NewWord16(42),
			expected: interpreter.NewUnmeteredWord16Value(42),
		},
		{
			label:    "Word32",
			value:    cadence.NewWord32(42),
			expected: interpreter.NewUnmeteredWord32Value(42),
		},
		{
			label:    "Word64",
			value:    cadence.NewWord64(42),
			expected: interpreter.NewUnmeteredWord64Value(42),
		},
		{
			label:    "Word128",
			value:    cadence.NewWord128(42),
			expected: interpreter.NewUnmeteredWord128ValueFromUint64(42),
		},
		{
			label:    "Word256",
			value:    cadence.NewWord256(42),
			expected: interpreter.NewUnmeteredWord256ValueFromUint64(42),
		},
		{
			label:    "Fix64",
			value:    cadence.Fix64(-123000000),
			expected: interpreter.NewUnmeteredFix64Value(-123000000),
		},
		{
			label:    "UFix64",
			value:    cadence.UFix64(123000000),
			expected: interpreter.NewUnmeteredUFix64Value(123000000),
		},
		{
			label: "Path",
			value: cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			expected: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
		},
		{
			label: "Path Link (invalid)",
			value: cadence.PathLink{
				TargetPath: cadence.Path{
					Domain:     common.PathDomainStorage,
					Identifier: "test",
				},
				BorrowType: "Int",
			},
			expected: nil,
		},
		{
			label:    "Account Link (invalid)",
			value:    cadence.AccountLink{},
			expected: nil,
		},
		{
			label: "path Capability (invalid)",
			value: cadence.NewPathCapability(
				cadence.Address{0x1},
				cadence.Path{
					Domain:     common.PathDomainPublic,
					Identifier: "test",
				},
				cadence.IntType{},
			),
			expected: nil,
		},
		{
			label: "ID Capability (invalid)",
			value: cadence.NewIDCapability(
				4,
				cadence.Address{0x1},
				cadence.IntType{},
			),
			expected: nil,
		},
		{
			label:    "Function (invalid)",
			value:    cadence.Function{},
			expected: nil,
		},
		{
			label:    "Type<Int>()",
			value:    cadence.NewTypeValue(cadence.IntType{}),
			expected: interpreter.TypeValue{Type: interpreter.PrimitiveStaticTypeInt},
		},
	} {
		test(tt)
	}
}

func assertUserError(t *testing.T, err error) {
	require.True(t,
		errors.IsUserError(err),
		"Expected `UserError`, found `%T`",
		err,
	)
}

func TestImportRuntimeType(t *testing.T) {
	t.Parallel()

	type importTest struct {
		expected interpreter.StaticType
		actual   cadence.Type
		label    string
	}

	test := func(tt importTest) {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()
			actual := ImportType(nil, tt.actual)
			assert.Equal(t, tt.expected, actual)

		})
	}

	for _, tt := range []importTest{
		{
			label:    "Any",
			actual:   cadence.AnyType{},
			expected: interpreter.PrimitiveStaticTypeAny,
		},
		{
			label:    "AnyStruct",
			actual:   cadence.AnyStructType{},
			expected: interpreter.PrimitiveStaticTypeAnyStruct,
		},
		{
			label:    "AnyResource",
			actual:   cadence.AnyResourceType{},
			expected: interpreter.PrimitiveStaticTypeAnyResource,
		},
		{
			label:    "MetaType",
			actual:   cadence.MetaType{},
			expected: interpreter.PrimitiveStaticTypeMetaType,
		},
		{
			label:    "Void",
			actual:   cadence.VoidType{},
			expected: interpreter.PrimitiveStaticTypeVoid,
		},
		{
			label:    "Never",
			actual:   cadence.NeverType{},
			expected: interpreter.PrimitiveStaticTypeNever,
		},
		{
			label:    "Bool",
			actual:   cadence.BoolType{},
			expected: interpreter.PrimitiveStaticTypeBool,
		},
		{
			label:    "String",
			actual:   cadence.StringType{},
			expected: interpreter.PrimitiveStaticTypeString,
		},
		{
			label:    "Character",
			actual:   cadence.CharacterType{},
			expected: interpreter.PrimitiveStaticTypeCharacter,
		},
		{
			label:    "Address",
			actual:   cadence.AddressType{},
			expected: interpreter.PrimitiveStaticTypeAddress,
		},
		{
			label:    "Number",
			actual:   cadence.NumberType{},
			expected: interpreter.PrimitiveStaticTypeNumber,
		},
		{
			label:    "SignedNumber",
			actual:   cadence.SignedNumberType{},
			expected: interpreter.PrimitiveStaticTypeSignedNumber,
		},
		{
			label:    "Integer",
			actual:   cadence.IntegerType{},
			expected: interpreter.PrimitiveStaticTypeInteger,
		},
		{
			label:    "SignedInteger",
			actual:   cadence.SignedIntegerType{},
			expected: interpreter.PrimitiveStaticTypeSignedInteger,
		},
		{
			label:    "FixedPoint",
			actual:   cadence.FixedPointType{},
			expected: interpreter.PrimitiveStaticTypeFixedPoint,
		},
		{
			label:    "SignedFixedPoint",
			actual:   cadence.SignedFixedPointType{},
			expected: interpreter.PrimitiveStaticTypeSignedFixedPoint,
		},
		{
			label:    "Int",
			actual:   cadence.IntType{},
			expected: interpreter.PrimitiveStaticTypeInt,
		},
		{
			label:    "Int8",
			actual:   cadence.Int8Type{},
			expected: interpreter.PrimitiveStaticTypeInt8,
		},
		{
			label:    "Int16",
			actual:   cadence.Int16Type{},
			expected: interpreter.PrimitiveStaticTypeInt16,
		},
		{
			label:    "Int32",
			actual:   cadence.Int32Type{},
			expected: interpreter.PrimitiveStaticTypeInt32,
		},
		{
			label:    "Int64",
			actual:   cadence.Int64Type{},
			expected: interpreter.PrimitiveStaticTypeInt64,
		},
		{
			label:    "Int128",
			actual:   cadence.Int128Type{},
			expected: interpreter.PrimitiveStaticTypeInt128,
		},
		{
			label:    "Int256",
			actual:   cadence.Int256Type{},
			expected: interpreter.PrimitiveStaticTypeInt256,
		},
		{
			label:    "UInt",
			actual:   cadence.UIntType{},
			expected: interpreter.PrimitiveStaticTypeUInt,
		},
		{
			label:    "UInt8",
			actual:   cadence.UInt8Type{},
			expected: interpreter.PrimitiveStaticTypeUInt8,
		},
		{
			label:    "UInt16",
			actual:   cadence.UInt16Type{},
			expected: interpreter.PrimitiveStaticTypeUInt16,
		},
		{
			label:    "UInt32",
			actual:   cadence.UInt32Type{},
			expected: interpreter.PrimitiveStaticTypeUInt32,
		},
		{
			label:    "UInt64",
			actual:   cadence.UInt64Type{},
			expected: interpreter.PrimitiveStaticTypeUInt64,
		},
		{
			label:    "UInt128",
			actual:   cadence.UInt128Type{},
			expected: interpreter.PrimitiveStaticTypeUInt128,
		},
		{
			label:    "UInt256",
			actual:   cadence.UInt256Type{},
			expected: interpreter.PrimitiveStaticTypeUInt256,
		},
		{
			label:    "Word8",
			actual:   cadence.Word8Type{},
			expected: interpreter.PrimitiveStaticTypeWord8,
		},
		{
			label:    "Word16",
			actual:   cadence.Word16Type{},
			expected: interpreter.PrimitiveStaticTypeWord16,
		},
		{
			label:    "Word32",
			actual:   cadence.Word32Type{},
			expected: interpreter.PrimitiveStaticTypeWord32,
		},
		{
			label:    "Word64",
			actual:   cadence.Word64Type{},
			expected: interpreter.PrimitiveStaticTypeWord64,
		},
		{
			label:    "Word128",
			actual:   cadence.Word128Type{},
			expected: interpreter.PrimitiveStaticTypeWord128,
		},
		{
			label:    "Word256",
			actual:   cadence.Word256Type{},
			expected: interpreter.PrimitiveStaticTypeWord256,
		},
		{
			label:    "Fix64",
			actual:   cadence.Fix64Type{},
			expected: interpreter.PrimitiveStaticTypeFix64,
		},
		{
			label:    "UFix64",
			actual:   cadence.UFix64Type{},
			expected: interpreter.PrimitiveStaticTypeUFix64,
		},
		{
			label:    "Block",
			actual:   cadence.BlockType{},
			expected: interpreter.PrimitiveStaticTypeBlock,
		},
		{
			label:    "CapabilityPath",
			actual:   cadence.CapabilityPathType{},
			expected: interpreter.PrimitiveStaticTypeCapabilityPath,
		},
		{
			label:    "StoragePath",
			actual:   cadence.StoragePathType{},
			expected: interpreter.PrimitiveStaticTypeStoragePath,
		},
		{
			label:    "PublicPath",
			actual:   cadence.PublicPathType{},
			expected: interpreter.PrimitiveStaticTypePublicPath,
		},
		{
			label:    "PrivatePath",
			actual:   cadence.PrivatePathType{},
			expected: interpreter.PrimitiveStaticTypePrivatePath,
		},
		{
			label:    "AuthAccount",
			actual:   cadence.AuthAccountType{},
			expected: interpreter.PrimitiveStaticTypeAuthAccount,
		},
		{
			label:    "PublicAccount",
			actual:   cadence.PublicAccountType{},
			expected: interpreter.PrimitiveStaticTypePublicAccount,
		},
		{
			label:    "DeployedContract",
			actual:   cadence.DeployedContractType{},
			expected: interpreter.PrimitiveStaticTypeDeployedContract,
		},
		{
			label:    "AuthAccount.Keys",
			actual:   cadence.AuthAccountKeysType{},
			expected: interpreter.PrimitiveStaticTypeAuthAccountKeys,
		},
		{
			label:    "PublicAccount.Keys",
			actual:   cadence.PublicAccountKeysType{},
			expected: interpreter.PrimitiveStaticTypePublicAccountKeys,
		},
		{
			label:    "AuthAccount.Contracts",
			actual:   cadence.AuthAccountContractsType{},
			expected: interpreter.PrimitiveStaticTypeAuthAccountContracts,
		},
		{
			label:    "PublicAccount.Contracts",
			actual:   cadence.PublicAccountContractsType{},
			expected: interpreter.PrimitiveStaticTypePublicAccountContracts,
		},
		{
			label:    "AccountKey",
			actual:   cadence.AccountKeyType{},
			expected: interpreter.PrimitiveStaticTypeAccountKey,
		},
		{
			label: "Optional",
			actual: &cadence.OptionalType{
				Type: cadence.IntType{},
			},
			expected: interpreter.OptionalStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "VariableSizedArray",
			actual: &cadence.VariableSizedArrayType{
				ElementType: cadence.IntType{},
			},
			expected: interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "ConstantSizedArray",
			actual: &cadence.ConstantSizedArrayType{
				ElementType: cadence.IntType{},
				Size:        3,
			},
			expected: interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
				Size: 3,
			},
		},
		{
			label: "Dictionary",
			actual: &cadence.DictionaryType{
				ElementType: cadence.IntType{},
				KeyType:     cadence.StringType{},
			},
			expected: interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Reference",
			actual: &cadence.ReferenceType{
				Authorized: false,
				Type:       cadence.IntType{},
			},
			expected: interpreter.ReferenceStaticType{
				Authorized:   false,
				BorrowedType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Capability",
			actual: &cadence.CapabilityType{
				BorrowType: cadence.IntType{},
			},
			expected: interpreter.CapabilityStaticType{
				BorrowType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Struct",
			actual: &cadence.StructType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.CompositeStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "Resource",
			actual: &cadence.ResourceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.CompositeStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "Contract",
			actual: &cadence.ContractType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.CompositeStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "Event",
			actual: &cadence.EventType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.CompositeStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "Enum",
			actual: &cadence.EnumType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.CompositeStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "StructInterface",
			actual: &cadence.StructInterfaceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.InterfaceStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "ResourceInterface",
			actual: &cadence.ResourceInterfaceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.InterfaceStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "ContractInterface",
			actual: &cadence.ContractInterfaceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.InterfaceStaticType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
		},
		{
			label: "RestrictedType",
			actual: &cadence.RestrictedType{
				Type: &cadence.StructType{
					Location:            TestLocation,
					QualifiedIdentifier: "S",
				},
				Restrictions: []cadence.Type{
					&cadence.StructInterfaceType{
						Location:            TestLocation,
						QualifiedIdentifier: "T",
					}},
			},
			expected: &interpreter.RestrictedStaticType{
				Type: interpreter.CompositeStaticType{
					Location:            TestLocation,
					QualifiedIdentifier: "S",
				},
				Restrictions: []interpreter.InterfaceStaticType{
					{
						Location:            TestLocation,
						QualifiedIdentifier: "T",
					},
				},
			},
		},
	} {
		test(tt)
	}
}

func TestExportIntegerValuesFromScript(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type) {

		t.Run(integerType.String(), func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`
                  pub fun main(): %s {
                      return 42
                  }
                `,
				integerType,
			)

			assert.NotPanics(t, func() {
				exportValueFromScript(t, script)
			})
		})
	}

	for _, integerType := range sema.AllIntegerTypes {
		test(integerType)
	}
}

func TestExportFixedPointValuesFromScript(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type, literal string) {

		t.Run(fixedPointType.String(), func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`
                  pub fun main(): %s {
                      return %s
                  }
                `,
				fixedPointType,
				literal,
			)

			assert.NotPanics(t, func() {
				exportValueFromScript(t, script)
			})
		})
	}

	for _, fixedPointType := range sema.AllFixedPointTypes {

		var literal string
		if sema.IsSubType(fixedPointType, sema.SignedFixedPointType) {
			literal = "-1.23"
		} else {
			literal = "1.23"
		}

		test(fixedPointType, literal)
	}
}

func TestExportAddressValue(t *testing.T) {

	t.Parallel()

	script := `
        pub fun main(): Address {
            return 0x42
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.BytesToAddress(
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42},
	)

	assert.Equal(t, expected, actual)
}

func TestExportStructValue(t *testing.T) {

	t.Parallel()

	script := `
        pub struct Foo {
            pub let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        pub fun main(): Foo {
            return Foo(bar: 42)
        }
    `

	fooStructType := &cadence.StructType{
		Location:            common.ScriptLocation{},
		QualifiedIdentifier: "Foo",
		Fields:              fooFields,
	}

	actual := exportValueFromScript(t, script)
	expected := cadence.NewStruct([]cadence.Value{cadence.NewInt(42)}).
		WithType(fooStructType)

	assert.Equal(t, expected, actual)
}

func TestExportResourceValue(t *testing.T) {

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

	actual := exportValueFromScript(t, script)
	expected :=
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(42),
		}).WithType(newFooResourceType())

	assert.Equal(t, expected, actual)
}

func TestExportResourceArrayValue(t *testing.T) {

	t.Parallel()

	script := `
        pub resource Foo {
            pub let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        pub fun main(): @[Foo] {
            return <- [<- create Foo(bar: 1), <- create Foo(bar: 2)]
        }
    `

	fooResourceType := newFooResourceType()

	actual := exportValueFromScript(t, script)
	expected := cadence.NewArray([]cadence.Value{
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(1),
		}).WithType(fooResourceType),
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(2),
		}).WithType(fooResourceType),
	}).WithType(&cadence.VariableSizedArrayType{
		ElementType: &cadence.ResourceType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "uuid",
					Type:       cadence.UInt64Type{},
				},
				{
					Identifier: "bar",
					Type:       cadence.IntType{},
				},
			},
		},
	})

	assert.Equal(t, expected, actual)
}

func TestExportResourceDictionaryValue(t *testing.T) {

	t.Parallel()

	script := `
        pub resource Foo {
            pub let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        pub fun main(): @{String: Foo} {
            return <- {
                "a": <- create Foo(bar: 1),
                "b": <- create Foo(bar: 2)
            }
        }
    `

	fooResourceType := newFooResourceType()

	actual := exportValueFromScript(t, script)
	expected := cadence.NewDictionary([]cadence.KeyValuePair{
		{
			Key: cadence.String("b"),
			Value: cadence.NewResource([]cadence.Value{
				cadence.NewUInt64(0),
				cadence.NewInt(2),
			}).WithType(fooResourceType),
		},
		{
			Key: cadence.String("a"),
			Value: cadence.NewResource([]cadence.Value{
				cadence.NewUInt64(0),
				cadence.NewInt(1),
			}).WithType(fooResourceType),
		},
	}).WithType(&cadence.DictionaryType{
		KeyType: cadence.StringType{},
		ElementType: &cadence.ResourceType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "uuid",
					Type:       cadence.UInt64Type{},
				},
				{
					Identifier: "bar",
					Type:       cadence.IntType{},
				},
			},
		},
	})

	assert.Equal(t, expected, actual)
}

func TestExportNestedResourceValueFromScript(t *testing.T) {

	t.Parallel()

	barResourceType := &cadence.ResourceType{
		Location:            common.ScriptLocation{},
		QualifiedIdentifier: "Bar",
		Fields: []cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "x",
				Type:       cadence.IntType{},
			},
		},
	}

	fooResourceType := &cadence.ResourceType{
		Location:            common.ScriptLocation{},
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type{},
			},
			{
				Identifier: "bar",
				Type:       barResourceType,
			},
		},
	}

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

	actual := exportValueFromScript(t, script)
	expected := cadence.NewResource([]cadence.Value{
		cadence.NewUInt64(0),
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(0),
			cadence.NewInt(42),
		}).WithType(barResourceType),
	}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestExportEventValue(t *testing.T) {

	t.Parallel()

	script := `
        pub event Foo(bar: Int)

        pub fun main() {
            emit Foo(bar: 42)
        }
    `

	fooEventType := &cadence.EventType{
		Location:            common.ScriptLocation{},
		QualifiedIdentifier: "Foo",
		Fields:              fooFields,
	}

	actual := exportEventFromScript(t, script)
	expected := cadence.NewEvent([]cadence.Value{cadence.NewInt(42)}).
		WithType(fooEventType)

	assert.Equal(t, expected, actual)
}

func exportEventFromScript(t *testing.T, script string) cadence.Event {
	rt := newTestInterpreterRuntime()

	var events []cadence.Event

	inter := &testRuntimeInterface{
		emitEvent: func(event cadence.Event) error {
			events = append(events, event)
			return nil
		},
	}

	_, err := rt.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: inter,
			Location:  common.ScriptLocation{},
		},
	)

	require.NoError(t, err)
	require.Len(t, events, 1)

	event := events[0]

	return event
}

func exportValueFromScript(t *testing.T, script string) cadence.Value {
	rt := newTestInterpreterRuntime()

	value, err := rt.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: &testRuntimeInterface{},
			Location:  common.ScriptLocation{},
		},
	)

	require.NoError(t, err)

	return value
}

func TestExportReferenceValue(t *testing.T) {

	t.Parallel()

	t.Run("ephemeral, Int", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): &Int {
                return &1 as &Int
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.NewInt(1)

		assert.Equal(t, expected, actual)
	})

	t.Run("ephemeral, recursive", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): [&AnyStruct] {
                let refs: [&AnyStruct] = []
                refs.append(&refs as &AnyStruct)
                return refs
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.NewArray([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				nil,
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: &cadence.ReferenceType{
					Type: cadence.AnyStructType{},
				},
			}),
		}).WithType(&cadence.VariableSizedArrayType{
			ElementType: &cadence.ReferenceType{
				Type: cadence.AnyStructType{},
			},
		})

		assert.Equal(t, expected, actual)
	})

	t.Run("storage", func(t *testing.T) {

		t.Parallel()

		// Arrange

		rt := newTestInterpreterRuntime()

		transaction := `
            transaction {
                prepare(signer: AuthAccount) {
                    signer.save(1, to: /storage/test)
                    signer.link<&Int>(
                        /public/test,
                        target: /storage/test
                    )
                }
            }
        `

		address, err := common.HexToAddress("0x1")
		require.NoError(t, err)

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
			getSigningAccounts: func() ([]Address, error) {
				return []Address{
					address,
				}, nil
			},
		}

		// Act

		err = rt.ExecuteTransaction(
			Script{
				Source: []byte(transaction),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.TransactionLocation{},
			},
		)
		require.NoError(t, err)

		script := `
            pub fun main(): &AnyStruct {
                return getAccount(0x1).getCapability(/public/test).borrow<&AnyStruct>()!
            }
        `

		actual, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.NoError(t, err)

		expected := cadence.NewInt(1)

		assert.Equal(t, expected, actual)
	})

	t.Run("storage, recursive, same reference", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): &AnyStruct {
                var acct = getAuthAccount(0x01)
	            var v:[AnyStruct] = []
	            acct.save(v, to: /storage/x)

                var ref = acct.borrow<&[AnyStruct]>(from: /storage/x)!
	            ref.append(ref)
	            return ref
            }
        `

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
		}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot store non-storable value")
	})

	t.Run("storage, recursive, two references", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): &AnyStruct {
                var acct = getAuthAccount(0x01)
	            var v:[AnyStruct] = []
	            acct.save(v, to: /storage/x)

                var ref1 = acct.borrow<&[AnyStruct]>(from: /storage/x)!
                var ref2 = acct.borrow<&[AnyStruct]>(from: /storage/x)!

	            ref1.append(ref2)
	            return ref1
            }
        `

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			storage: newTestLedger(nil, nil),
		}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot store non-storable value")
	})
}

func TestExportTypeValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): Type {
                return Type<Int>()
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.TypeValue{
			StaticType: cadence.IntType{},
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		script := `
            pub struct S {}

            pub fun main(): Type {
                return Type<S>()
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.TypeValue{
			StaticType: &cadence.StructType{
				QualifiedIdentifier: "S",
				Location:            common.ScriptLocation{},
				Fields:              []cadence.Field{},
			},
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("builtin struct", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(): Type {
                return CompositeType("PublicKey")!
            }
        `

		actual := exportValueFromScript(t, script)

		_, err := json.Encode(actual)
		require.NoError(t, err)
	})

	t.Run("without static type", func(t *testing.T) {

		t.Parallel()

		value := interpreter.TypeValue{
			Type: nil,
		}
		actual, err := exportValueWithInterpreter(
			value,
			newTestInterpreter(t),
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.TypeValue{
			StaticType: nil,
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("with restricted static type", func(t *testing.T) {

		t.Parallel()

		const code = `
          pub struct interface SI {}

          pub struct S: SI {}

        `
		program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})
		require.NoError(t, err)

		checker, err := sema.NewChecker(
			program,
			TestLocation,
			nil,
			&sema.Config{
				AccessCheckMode: sema.AccessCheckModeStrict,
			},
		)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		inter := newTestInterpreter(t)
		inter.Program = interpreter.ProgramFromChecker(checker)

		ty := interpreter.TypeValue{
			Type: &interpreter.RestrictedStaticType{
				Type: interpreter.NewCompositeStaticTypeComputeTypeID(inter, TestLocation, "S"),
				Restrictions: []interpreter.InterfaceStaticType{
					{
						Location:            TestLocation,
						QualifiedIdentifier: "SI",
					},
				},
			},
		}

		actual, err := ExportValue(ty, inter, interpreter.EmptyLocationRange)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.TypeValue{
				StaticType: &cadence.RestrictedType{
					Type: &cadence.StructType{
						QualifiedIdentifier: "S",
						Location:            TestLocation,
						Fields:              []cadence.Field{},
					},
					Restrictions: []cadence.Type{
						&cadence.StructInterfaceType{
							QualifiedIdentifier: "SI",
							Location:            TestLocation,
							Fields:              []cadence.Field{},
						},
					},
				},
			},
			actual,
		)
	})

}

func TestExportPathCapabilityValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		capability := interpreter.NewUnmeteredPathCapabilityValue(
			interpreter.AddressValue{0x1},
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			interpreter.PrimitiveStaticTypeInt,
		)

		actual, err := exportValueWithInterpreter(
			capability,
			newTestInterpreter(t),
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			cadence.IntType{},
		)

		assert.Equal(t, expected, actual)

	})

	t.Run("Struct", func(t *testing.T) {

		const code = `
          struct S {}
        `
		program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})
		require.NoError(t, err)

		checker, err := sema.NewChecker(
			program,
			TestLocation,
			nil,
			&sema.Config{
				AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
			},
		)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		inter := newTestInterpreter(t)
		inter.Program = interpreter.ProgramFromChecker(checker)

		capability := interpreter.NewUnmeteredPathCapabilityValue(
			interpreter.AddressValue{0x1},
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			interpreter.NewCompositeStaticTypeComputeTypeID(inter, TestLocation, "S"),
		)

		actual, err := exportValueWithInterpreter(
			capability,
			inter,
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			&cadence.StructType{
				QualifiedIdentifier: "S",
				Location:            TestLocation,
				Fields:              []cadence.Field{},
			},
		)

		assert.Equal(t, expected, actual)
	})

	t.Run("no borrow type", func(t *testing.T) {

		capability := interpreter.NewUnmeteredPathCapabilityValue(
			interpreter.AddressValue{0x1},
			interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			nil,
		)

		actual, err := exportValueWithInterpreter(
			capability,
			newTestInterpreter(t),
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			nil,
		)

		assert.Equal(t, expected, actual)
	})
}

func TestExportIDCapabilityValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		capability := interpreter.NewUnmeteredIDCapabilityValue(
			3,
			interpreter.AddressValue{0x1},
			interpreter.PrimitiveStaticTypeInt,
		)

		actual, err := exportValueWithInterpreter(
			capability,
			newTestInterpreter(t),
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.NewIDCapability(
			3,
			cadence.Address{0x1},
			cadence.IntType{},
		)

		assert.Equal(t, expected, actual)

	})

	t.Run("Struct", func(t *testing.T) {

		const code = `
          struct S {}
        `
		program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})
		require.NoError(t, err)

		checker, err := sema.NewChecker(
			program,
			TestLocation,
			nil,
			&sema.Config{
				AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
			},
		)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		inter := newTestInterpreter(t)
		inter.Program = interpreter.ProgramFromChecker(checker)

		capability := interpreter.NewUnmeteredIDCapabilityValue(
			3,
			interpreter.AddressValue{0x1},
			interpreter.NewCompositeStaticTypeComputeTypeID(inter, TestLocation, "S"),
		)

		actual, err := exportValueWithInterpreter(
			capability,
			inter,
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.NewIDCapability(
			3,
			cadence.Address{0x1},
			&cadence.StructType{
				QualifiedIdentifier: "S",
				Location:            TestLocation,
				Fields:              []cadence.Field{},
			},
		)

		assert.Equal(t, expected, actual)
	})
}

func TestExportPathLinkValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		link := interpreter.PathLinkValue{
			TargetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			Type: interpreter.PrimitiveStaticTypeInt,
		}

		actual, err := exportValueWithInterpreter(
			link,
			newTestInterpreter(t),
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.PathLink{
			TargetPath: cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			BorrowType: "Int",
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("Struct", func(t *testing.T) {

		const code = `
          struct S {}
        `
		program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})
		require.NoError(t, err)

		checker, err := sema.NewChecker(
			program,
			TestLocation,
			nil,
			&sema.Config{
				AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
			},
		)
		require.NoError(t, err)

		err = checker.Check()
		require.NoError(t, err)

		inter := newTestInterpreter(t)
		inter.Program = interpreter.ProgramFromChecker(checker)

		capability := interpreter.PathLinkValue{
			TargetPath: interpreter.PathValue{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			Type: interpreter.NewCompositeStaticTypeComputeTypeID(inter, TestLocation, "S"),
		}

		actual, err := exportValueWithInterpreter(
			capability,
			inter,
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		expected := cadence.PathLink{
			TargetPath: cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			BorrowType: "S.test.S",
		}

		assert.Equal(t, expected, actual)
	})
}

func TestExportAccountLinkValue(t *testing.T) {

	t.Parallel()

	link := interpreter.AccountLinkValue{}

	actual, err := exportValueWithInterpreter(
		link,
		newTestInterpreter(t),
		interpreter.EmptyLocationRange,
		seenReferences{},
	)
	require.NoError(t, err)

	expected := cadence.AccountLink{}

	assert.Equal(t, expected, actual)
}

func TestExportCompositeValueWithFunctionValueField(t *testing.T) {

	t.Parallel()

	script := `
        pub struct Foo {
            pub let answer: Int
            pub let f: ((): Void)

            init() {
                self.answer = 42
                self.f = fun () {}
            }
        }

        pub fun main(): Foo {
            return Foo()
        }
    `

	fooStructType := &cadence.StructType{
		Location:            common.ScriptLocation{},
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "answer",
				Type:       cadence.IntType{},
			},
			{
				Identifier: "f",
				Type: &cadence.FunctionType{
					ReturnType: cadence.VoidType{},
				},
			},
		},
	}

	actual := exportValueFromScript(t, script)
	expected := cadence.NewStruct([]cadence.Value{
		cadence.NewInt(42),
		cadence.Function{
			FunctionType: &cadence.FunctionType{
				ReturnType: cadence.VoidType{},
			},
		},
	}).WithType(fooStructType)

	assert.Equal(t, expected, actual)
}

//go:embed test-export-json-deterministic.txt
var exportJsonDeterministicExpected string

func TestExportJsonDeterministic(t *testing.T) {
	t.Parallel()

	// exported order of field in a dictionary depends on the execution ,
	// however the deterministic code should generate deterministic type

	script := `
        access(all) event Foo(bar: Int, aaa: {Int: {Int: String}})

        access(all) fun main() {

			let dict0 = {
				3: "c",
				2: "c",
				1: "a",
				0: "a"
			}

			let dict2 = {
				7: "d"
			}

			dict2[1] = "c"
			dict2[3] = "b"

            emit Foo(
				bar: 2,
				aaa: {
					2: dict2,
					1: {
						3: "a",
						7: "b",
						2: "a",
						1: ""
					},
					0: dict0
				}
			)
        }
    `

	event := exportEventFromScript(t, script)

	bytes, err := json.Encode(event)

	assert.NoError(t, err)
	// NOTE: NOT using JSONEq, as we want to check order (by string equality), instead of structural equality
	assert.Equal(t, exportJsonDeterministicExpected, string(bytes))
}

var fooFields = []cadence.Field{
	{
		Identifier: "bar",
		Type:       cadence.IntType{},
	},
}
var fooResourceFields = []cadence.Field{
	{
		Identifier: "uuid",
		Type:       cadence.UInt64Type{},
	},
	{
		Identifier: "bar",
		Type:       cadence.IntType{},
	},
}

func newFooResourceType() *cadence.ResourceType {
	return &cadence.ResourceType{
		Location:            common.ScriptLocation{},
		QualifiedIdentifier: "Foo",
		Fields:              fooResourceFields,
	}
}

func TestRuntimeEnumValue(t *testing.T) {

	t.Parallel()

	enumValue := cadence.Enum{
		EnumType: &cadence.EnumType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Direction",
			Fields: []cadence.Field{
				{
					Identifier: sema.EnumRawValueFieldName,
					Type:       cadence.IntType{},
				},
			},
			RawType: cadence.IntType{},
		},
		Fields: []cadence.Value{
			cadence.NewInt(3),
		},
	}

	t.Run("test export", func(t *testing.T) {
		script := `
            pub fun main(): Direction {
                return Direction.RIGHT
            }

            pub enum Direction: Int {
                pub case UP
                pub case DOWN
                pub case LEFT
                pub case RIGHT
            }
        `

		actual := exportValueFromScript(t, script)
		assert.Equal(t, enumValue, actual)
	})

	t.Run("test import", func(t *testing.T) {
		script := `
            pub fun main(dir: Direction): Direction {
                if !dir.isInstance(Type<Direction>()) {
                    panic("Not a Direction value")
                }

                return dir
            }

            pub enum Direction: Int {
                pub case UP
                pub case DOWN
                pub case LEFT
                pub case RIGHT
            }
        `

		actual, err := executeTestScript(t, script, enumValue)
		require.NoError(t, err)
		assert.Equal(t, enumValue, actual)
	})
}

func executeTestScript(t *testing.T, script string, arg cadence.Value) (cadence.Value, error) {
	rt := newTestInterpreterRuntime()

	runtimeInterface := &testRuntimeInterface{
		storage: newTestLedger(nil, nil),
		meterMemory: func(_ common.MemoryUsage) error {
			return nil
		},
	}
	runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
		return json.Decode(runtimeInterface, b)
	}

	scriptParam := Script{
		Source: []byte(script),
	}

	if arg != nil {
		encodedArg, err := json.Encode(arg)
		require.NoError(t, err)
		scriptParam.Arguments = [][]byte{encodedArg}
	}

	value, err := rt.ExecuteScript(
		scriptParam,
		Context{
			Interface: runtimeInterface,
			Location:  common.ScriptLocation{},
		},
	)

	if err == nil {
		value = cadence.ValueWithCachedTypeID(value)
	}

	return value, err
}

func TestRuntimeArgumentPassing(t *testing.T) {

	t.Parallel()

	type argumentPassingTest struct {
		exportedValue cadence.Value
		label         string
		typeSignature string
		skipExport    bool
	}

	var argumentPassingTests = []argumentPassingTest{
		{
			label:         "Nil",
			typeSignature: "String?",
			exportedValue: cadence.NewOptional(nil),
		},
		{
			label:         "Bool true",
			typeSignature: "Bool",
			exportedValue: cadence.NewBool(true),
		},
		{
			label:         "Bool false",
			typeSignature: "Bool",
			exportedValue: cadence.NewBool(false),
		},
		{
			label:         "String empty",
			typeSignature: "String",
			exportedValue: cadence.String(""),
		},
		{
			label:         "String non-empty",
			typeSignature: "String",
			exportedValue: cadence.String("foo"),
		},
		{
			label:         "Array empty",
			typeSignature: "[String]",
			exportedValue: cadence.NewArray([]cadence.Value{}).
				WithType(&cadence.VariableSizedArrayType{
					ElementType: cadence.StringType{},
				}),
		},
		{
			label:         "Array non-empty",
			typeSignature: "[String]",
			exportedValue: cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: cadence.StringType{},
			}),
		},
		{
			label:         "Dictionary non-empty",
			typeSignature: "{String: String}",
			exportedValue: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: cadence.String("bar"),
				},
			}).WithType(&cadence.DictionaryType{
				KeyType:     cadence.StringType{},
				ElementType: cadence.StringType{},
			}),
		},
		{
			label:         "Int",
			typeSignature: "Int",
			exportedValue: cadence.NewInt(42),
		},
		{
			label:         "Int8",
			typeSignature: "Int8",
			exportedValue: cadence.NewInt8(42),
		},
		{
			label:         "Int16",
			typeSignature: "Int16",
			exportedValue: cadence.NewInt16(42),
		},
		{
			label:         "Int32",
			typeSignature: "Int32",
			exportedValue: cadence.NewInt32(42),
		},
		{
			label:         "Int64",
			typeSignature: "Int64",
			exportedValue: cadence.NewInt64(42),
		},
		{
			label:         "Int128",
			typeSignature: "Int128",
			exportedValue: cadence.NewInt128(42),
		},
		{
			label:         "Int256",
			typeSignature: "Int256",
			exportedValue: cadence.NewInt256(42),
		},
		{
			label:         "UInt",
			typeSignature: "UInt",
			exportedValue: cadence.NewUInt(42),
		},
		{
			label:         "UInt8",
			typeSignature: "UInt8",
			exportedValue: cadence.NewUInt8(42),
		},
		{
			label:         "UInt16",
			typeSignature: "UInt16",
			exportedValue: cadence.NewUInt16(42),
		},
		{
			label:         "UInt32",
			typeSignature: "UInt32",
			exportedValue: cadence.NewUInt32(42),
		},
		{
			label:         "UInt64",
			typeSignature: "UInt64",
			exportedValue: cadence.NewUInt64(42),
		},
		{
			label:         "UInt128",
			typeSignature: "UInt128",
			exportedValue: cadence.NewUInt128(42),
		},
		{
			label:         "UInt256",
			typeSignature: "UInt256",
			exportedValue: cadence.NewUInt256(42),
		},
		{
			label:         "Word8",
			typeSignature: "Word8",
			exportedValue: cadence.NewWord8(42),
		},
		{
			label:         "Word16",
			typeSignature: "Word16",
			exportedValue: cadence.NewWord16(42),
		},
		{
			label:         "Word32",
			typeSignature: "Word32",
			exportedValue: cadence.NewWord32(42),
		},
		{
			label:         "Word64",
			typeSignature: "Word64",
			exportedValue: cadence.NewWord64(42),
		},
		{
			label:         "Word128",
			typeSignature: "Word128",
			exportedValue: cadence.NewWord128(42),
		},
		{
			label:         "Word256",
			typeSignature: "Word256",
			exportedValue: cadence.NewWord256(42),
		},
		{
			label:         "Fix64",
			typeSignature: "Fix64",
			exportedValue: cadence.Fix64(-123000000),
		},
		{
			label:         "UFix64",
			typeSignature: "UFix64",
			exportedValue: cadence.UFix64(123000000),
		},
		{
			label:         "StoragePath",
			typeSignature: "StoragePath",
			exportedValue: cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			skipExport: true,
		},
		{
			label:         "PrivatePath",
			typeSignature: "PrivatePath",
			exportedValue: cadence.Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			skipExport: true,
		},
		{
			label:         "PublicPath",
			typeSignature: "PublicPath",
			exportedValue: cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			skipExport: true,
		},
		{
			label:         "Address",
			typeSignature: "Address",
			exportedValue: cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 1, 0, 2}),
		},
	}

	testArgumentPassing := func(test argumentPassingTest) {

		t.Run(test.label, func(t *testing.T) {

			t.Parallel()

			returnSignature := ""
			returnStmt := ""

			if !test.skipExport {
				returnSignature = fmt.Sprintf(": %[1]s", test.typeSignature)
				returnStmt = "return arg"
			}

			script := fmt.Sprintf(
				`pub fun main(arg: %[1]s)%[2]s {

                    if !arg.isInstance(Type<%[1]s>()) {
                        panic("Not a %[1]s value")
                    }

                    %[3]s
                }`,
				test.typeSignature,
				returnSignature,
				returnStmt,
			)

			actual, err := executeTestScript(t, script, test.exportedValue)
			require.NoError(t, err)

			if !test.skipExport {
				expected := cadence.ValueWithCachedTypeID(test.exportedValue)
				assert.Equal(t, expected, actual)
			}
		})
	}

	for _, testCase := range argumentPassingTests {
		testArgumentPassing(testCase)
	}
}

func TestRuntimeComplexStructArgumentPassing(t *testing.T) {

	t.Parallel()

	// Complex struct value
	complexStructValue := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.OptionalType{
						Type: cadence.StringType{},
					},
				},
				{
					Identifier: "b",
					Type: &cadence.DictionaryType{
						KeyType:     cadence.StringType{},
						ElementType: cadence.StringType{},
					},
				},
				{
					Identifier: "c",
					Type: &cadence.VariableSizedArrayType{
						ElementType: cadence.StringType{},
					},
				},
				{
					Identifier: "d",
					Type: &cadence.ConstantSizedArrayType{
						ElementType: cadence.StringType{},
						Size:        2,
					},
				},
				{
					Identifier: "e",
					Type:       cadence.AddressType{},
				},
				{
					Identifier: "f",
					Type:       cadence.BoolType{},
				},
				{
					Identifier: "g",
					Type:       cadence.StoragePathType{},
				},
				{
					Identifier: "h",
					Type:       cadence.PublicPathType{},
				},
				{
					Identifier: "i",
					Type:       cadence.PrivatePathType{},
				},
				{
					Identifier: "j",
					Type:       cadence.AnyStructType{},
				},
			},
		},

		Fields: []cadence.Value{
			cadence.NewOptional(
				cadence.String("John"),
			),
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("name"),
					Value: cadence.String("Doe"),
				},
			}).WithType(&cadence.DictionaryType{
				KeyType:     cadence.StringType{},
				ElementType: cadence.StringType{},
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: cadence.StringType{},
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}).WithType(&cadence.ConstantSizedArrayType{
				ElementType: cadence.StringType{},
				Size:        2,
			}),
			cadence.NewAddress([8]byte{0, 0, 0, 0, 0, 1, 0, 2}),
			cadence.NewBool(true),
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			cadence.Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			cadence.String("foo"),
		},
	}

	script := fmt.Sprintf(
		`
          pub fun main(arg: %[1]s): %[1]s {

              if !arg.isInstance(Type<%[1]s>()) {
                  panic("Not a %[1]s value")
              }

              return arg
          }

          pub struct Foo {
              pub var a: String?
              pub var b: {String: String}
              pub var c: [String]
              pub var d: [String; 2]
              pub var e: Address
              pub var f: Bool
              pub var g: StoragePath
              pub var h: PublicPath
              pub var i: PrivatePath
              pub var j: AnyStruct

              init() {
                  self.a = "Hello"
                  self.b = {}
                  self.c = []
                  self.d = ["foo", "bar"]
                  self.e = 0x42
                  self.f = true
                  self.g = /storage/foo
                  self.h = /public/foo
                  self.i = /private/foo
                  self.j = nil
              }
          }
        `,
		"Foo",
	)

	actual, err := executeTestScript(t, script, complexStructValue)
	require.NoError(t, err)

	expected := cadence.ValueWithCachedTypeID(complexStructValue)
	assert.Equal(t, expected, actual)

}

func TestRuntimeComplexStructWithAnyStructFields(t *testing.T) {

	t.Parallel()

	// Complex struct value
	complexStructValue := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.OptionalType{
						Type: cadence.AnyStructType{},
					},
				},
				{
					Identifier: "b",
					Type: &cadence.DictionaryType{
						KeyType:     cadence.StringType{},
						ElementType: cadence.AnyStructType{},
					},
				},
				{
					Identifier: "c",
					Type: &cadence.VariableSizedArrayType{
						ElementType: cadence.AnyStructType{},
					},
				},
				{
					Identifier: "d",
					Type: &cadence.ConstantSizedArrayType{
						ElementType: cadence.AnyStructType{},
						Size:        2,
					},
				},
				{
					Identifier: "e",
					Type:       cadence.AnyStructType{},
				},
			},
		},

		Fields: []cadence.Value{
			cadence.NewOptional(cadence.String("John")),
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("name"),
					Value: cadence.String("Doe"),
				},
			}).WithType(&cadence.DictionaryType{
				KeyType:     cadence.StringType{},
				ElementType: cadence.AnyStructType{},
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: cadence.AnyStructType{},
			}),
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}).WithType(&cadence.ConstantSizedArrayType{
				ElementType: cadence.AnyStructType{},
				Size:        2,
			}),
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
		},
	}

	script := fmt.Sprintf(
		`
          pub fun main(arg: %[1]s): %[1]s {

              if !arg.isInstance(Type<%[1]s>()) {
                  panic("Not a %[1]s value")
              }

              return arg
          }

          pub struct Foo {
              pub var a: AnyStruct?
              pub var b: {String: AnyStruct}
              pub var c: [AnyStruct]
              pub var d: [AnyStruct; 2]
              pub var e: AnyStruct

              init() {
                  self.a = "Hello"
                  self.b = {}
                  self.c = []
                  self.d = ["foo", "bar"]
                  self.e = /storage/foo
              }
        }
        `,
		"Foo",
	)

	actual, err := executeTestScript(t, script, complexStructValue)
	require.NoError(t, err)

	expected := cadence.ValueWithCachedTypeID(complexStructValue)
	assert.Equal(t, expected, actual)
}

func TestRuntimeMalformedArgumentPassing(t *testing.T) {

	t.Parallel()

	// Struct with wrong field type

	malformedStructType1 := &cadence.StructType{
		Location:            common.ScriptLocation{},
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "a",
				Type:       cadence.IntType{},
			},
		},
	}

	malformedStruct1 := cadence.Struct{
		StructType: malformedStructType1,
		Fields: []cadence.Value{
			cadence.NewInt(3),
		},
	}

	// Struct with wrong field name

	malformedStruct2 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Foo",
			Fields: []cadence.Field{
				{
					Identifier: "nonExisting",
					Type:       cadence.StringType{},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.String("John"),
		},
	}

	// Struct with nested malformed array value
	malformedStruct3 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Bar",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.VariableSizedArrayType{
						ElementType: malformedStructType1,
					},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
		},
	}

	// Struct with nested malformed dictionary value
	malformedStruct4 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Baz",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.DictionaryType{
						KeyType:     cadence.StringType{},
						ElementType: malformedStructType1,
					},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: malformedStruct1,
				},
			}),
		},
	}

	// Struct with nested array with mismatching element type
	malformedStruct5 := cadence.Struct{
		StructType: &cadence.StructType{
			Location:            common.ScriptLocation{},
			QualifiedIdentifier: "Bar",
			Fields: []cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.VariableSizedArrayType{
						ElementType: malformedStructType1,
					},
				},
			},
		},
		Fields: []cadence.Value{
			cadence.NewArray([]cadence.Value{
				cadence.String("mismatching value"),
			}),
		},
	}

	type argumentPassingTest struct {
		exportedValue                            cadence.Value
		expectedInvalidEntryPointArgumentErrType error
		label                                    string
		typeSignature                            string
		expectedContainerMutationError           bool
	}

	var argumentPassingTests = []argumentPassingTest{
		{
			label:                                    "Malformed Struct field type",
			typeSignature:                            "Foo",
			exportedValue:                            malformedStruct1,
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed Struct field name",
			typeSignature:                            "Foo",
			exportedValue:                            malformedStruct2,
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed AnyStruct",
			typeSignature:                            "AnyStruct",
			exportedValue:                            malformedStruct1,
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed nested struct array",
			typeSignature:                            "Bar",
			exportedValue:                            malformedStruct3,
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed nested struct dictionary",
			typeSignature:                            "Baz",
			exportedValue:                            malformedStruct4,
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Variable-size array with malformed element",
			typeSignature: "[Foo]",
			exportedValue: cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Constant-size array with malformed element",
			typeSignature: "[Foo; 1]",
			exportedValue: cadence.NewArray([]cadence.Value{
				malformedStruct1,
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Constant-size array with too few elements",
			typeSignature: "[Int; 2]",
			exportedValue: cadence.NewArray([]cadence.Value{
				cadence.NewInt(1),
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Constant-size array with too many elements",
			typeSignature: "[Int; 2]",
			exportedValue: cadence.NewArray([]cadence.Value{
				cadence.NewInt(1),
				cadence.NewInt(2),
				cadence.NewInt(3),
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Nested array with mismatching element",
			typeSignature: "[[String]]",
			exportedValue: cadence.NewArray([]cadence.Value{
				cadence.NewArray([]cadence.Value{
					cadence.NewInt(5),
				}),
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Inner array with mismatching element",
			typeSignature:                            "Bar",
			exportedValue:                            malformedStruct5,
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed Optional",
			typeSignature:                            "Foo?",
			exportedValue:                            cadence.NewOptional(malformedStruct1),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Malformed dictionary",
			typeSignature: "{String: Foo}",
			exportedValue: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: malformedStruct1,
				},
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
	}

	testArgumentPassing := func(test argumentPassingTest) {

		t.Run(test.label, func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`pub fun main(arg: %[1]s): %[1]s {

                    if !arg.isInstance(Type<%[1]s>()) {
                        panic("Not a %[1]s value")
                    }

                    return arg
                }

                pub struct Foo {
                    pub var a: String

                    init() {
                        self.a = "Hello"
                    }
                }

                pub struct Bar {
                    pub var a: [Foo]

                    init() {
                        self.a = []
                    }
                }

                pub struct Baz {
                    pub var a: {String: Foo}

                    init() {
                        self.a = {}
                    }
                }`,
				test.typeSignature,
			)

			_, err := executeTestScript(t, script, test.exportedValue)

			if test.expectedInvalidEntryPointArgumentErrType != nil {
				RequireError(t, err)
				assertUserError(t, err)

				var invalidEntryPointArgumentError *InvalidEntryPointArgumentError
				require.ErrorAs(t, err, &invalidEntryPointArgumentError)

				require.IsType(t,
					test.expectedInvalidEntryPointArgumentErrType,
					invalidEntryPointArgumentError.Err,
				)
			} else if test.expectedContainerMutationError {
				RequireError(t, err)
				assertUserError(t, err)

				var containerMutationError interpreter.ContainerMutationError
				require.ErrorAs(t, err, &containerMutationError)
			} else {
				require.NoError(t, err)
			}
		})
	}

	for _, testCase := range argumentPassingTests {
		testArgumentPassing(testCase)
	}
}

func TestRuntimeImportExportArrayValue(t *testing.T) {

	t.Parallel()

	t.Run("export empty", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		value := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
		)

		actual, err := exportValueWithInterpreter(
			value,
			inter,
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewArray([]cadence.Value{}).
				WithType(&cadence.VariableSizedArrayType{
					ElementType: cadence.AnyStructType{},
				}),
			actual,
		)
	})

	t.Run("import empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewArray([]cadence.Value{})

		inter := newTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			value,
			sema.ByteArrayType,
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
				common.ZeroAddress,
			),
			actual,
		)
	})

	t.Run("export non-empty", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		value := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(42),
			interpreter.NewUnmeteredStringValue("foo"),
		)

		actual, err := exportValueWithInterpreter(
			value,
			inter,
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: cadence.AnyStructType{},
			}),
			actual,
		)
	})

	t.Run("import non-empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewArray([]cadence.Value{
			cadence.NewInt(42),
			cadence.String("foo"),
		})

		inter := newTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			value,
			&sema.VariableSizedType{
				Type: sema.AnyStructType,
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeAnyStruct,
				},
				common.ZeroAddress,
				interpreter.NewUnmeteredIntValueFromInt64(42),
				interpreter.NewUnmeteredStringValue("foo"),
			),
			actual,
		)
	})

	t.Run("import nested array with broader expected type", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewArray([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				cadence.NewInt8(4),
				cadence.NewInt8(3),
			}),
			cadence.NewArray([]cadence.Value{
				cadence.NewInt8(42),
				cadence.NewInt8(54),
			}),
		})

		inter := newTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			value,
			sema.AnyStructType,
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.VariableSizedStaticType{
					Type: interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeInt8,
					},
				},
				common.ZeroAddress,
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeInt8,
					},
					common.ZeroAddress,
					interpreter.NewUnmeteredInt8Value(4),
					interpreter.NewUnmeteredInt8Value(3),
				),
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeInt8,
					},
					common.ZeroAddress,
					interpreter.NewUnmeteredInt8Value(42),
					interpreter.NewUnmeteredInt8Value(54),
				),
			),
			actual,
		)
	})
}

func TestRuntimeImportExportDictionaryValue(t *testing.T) {

	t.Parallel()

	t.Run("export empty", func(t *testing.T) {

		t.Parallel()

		value := interpreter.NewDictionaryValue(
			newTestInterpreter(t),
			interpreter.EmptyLocationRange,
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
		)

		actual, err := exportValueWithInterpreter(
			value,
			newTestInterpreter(t),
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{}).
				WithType(&cadence.DictionaryType{
					KeyType:     cadence.StringType{},
					ElementType: cadence.IntType{},
				}),
			actual,
		)
	})

	t.Run("import empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewDictionary([]cadence.KeyValuePair{})

		inter := newTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			value,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.UInt8Type,
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeUInt8,
				},
			),
			actual,
		)
	})

	t.Run("export non-empty", func(t *testing.T) {

		t.Parallel()

		inter := newTestInterpreter(t)

		value := interpreter.NewDictionaryValue(
			inter,
			interpreter.EmptyLocationRange,
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.NewUnmeteredStringValue("a"), interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredStringValue("b"), interpreter.NewUnmeteredIntValueFromInt64(2),
		)

		actual, err := exportValueWithInterpreter(
			value,
			inter,
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("b"),
					Value: cadence.NewInt(2),
				},
				{
					Key:   cadence.String("a"),
					Value: cadence.NewInt(1),
				},
			}).WithType(&cadence.DictionaryType{
				KeyType:     cadence.StringType{},
				ElementType: cadence.IntType{},
			}),
			actual,
		)
	})

	t.Run("import non-empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("a"),
				Value: cadence.NewInt(1),
			},
			{
				Key:   cadence.String("b"),
				Value: cadence.NewInt(2),
			},
		})

		inter := newTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			value,
			&sema.DictionaryType{
				KeyType:   sema.StringType,
				ValueType: sema.IntType,
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeInt,
				},
				interpreter.NewUnmeteredStringValue("a"), interpreter.NewUnmeteredIntValueFromInt64(1),
				interpreter.NewUnmeteredStringValue("b"), interpreter.NewUnmeteredIntValueFromInt64(2),
			),
			actual,
		)
	})

	t.Run("import nested dictionary with broader expected type", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key: cadence.String("a"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.NewInt8(1),
						Value: cadence.NewInt(100),
					},
					{
						Key:   cadence.NewInt8(2),
						Value: cadence.String("hello"),
					},
				}),
			},
			{
				Key: cadence.String("b"),
				Value: cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.NewInt8(1),
						Value: cadence.String("foo"),
					},
					{
						Key:   cadence.NewInt(2),
						Value: cadence.NewInt(50),
					},
				}),
			},
		})

		inter := newTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			value,
			sema.AnyStructType,
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.DictionaryStaticType{
					KeyType: interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeSignedInteger,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
				},

				interpreter.NewUnmeteredStringValue("a"),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeInt8,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					interpreter.NewUnmeteredInt8Value(1), interpreter.NewUnmeteredIntValueFromInt64(100),
					interpreter.NewUnmeteredInt8Value(2), interpreter.NewUnmeteredStringValue("hello"),
				),

				interpreter.NewUnmeteredStringValue("b"),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeSignedInteger,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					interpreter.NewUnmeteredInt8Value(1), interpreter.NewUnmeteredStringValue("foo"),
					interpreter.NewUnmeteredIntValueFromInt64(2), interpreter.NewUnmeteredIntValueFromInt64(50),
				),
			),
			actual,
		)
	})

	t.Run("import dictionary with heterogeneous keys", func(t *testing.T) {
		t.Parallel()

		script :=
			`pub fun main(arg: Foo) {
            }

            pub struct Foo {
                pub var a: AnyStruct

                init() {
                    self.a = nil
                }
            }`

		// Struct with nested malformed dictionary value
		malformedStruct := cadence.Struct{
			StructType: &cadence.StructType{
				Location:            common.ScriptLocation{},
				QualifiedIdentifier: "Foo",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.AnyStructType{},
					},
				},
			},
			Fields: []cadence.Value{
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("foo"),
						Value: cadence.String("value1"),
					},
					{
						Key:   cadence.NewInt(5),
						Value: cadence.String("value2"),
					},
				}),
			},
		}

		_, err := executeTestScript(t, script, malformedStruct)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)

		assert.Contains(t, argErr.Error(), "cannot import dictionary: keys does not belong to the same type")
	})

	t.Run("nested dictionary with mismatching element", func(t *testing.T) {
		t.Parallel()

		script :=
			`pub fun main(arg: {String: {String: String}}) {
            }
            `

		dictionary := cadence.NewDictionary(
			[]cadence.KeyValuePair{
				{
					Key: cadence.String("hello"),
					Value: cadence.NewDictionary(
						[]cadence.KeyValuePair{
							{
								Key:   cadence.String("hello"),
								Value: cadence.NewInt(6),
							},
						},
					),
				},
			},
		)

		_, err := executeTestScript(t, script, dictionary)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr interpreter.ContainerMutationError
		require.ErrorAs(t, err, &argErr)
	})
}

func TestRuntimeStringValueImport(t *testing.T) {

	t.Parallel()

	t.Run("non-utf8", func(t *testing.T) {

		t.Parallel()

		nonUTF8String := "\xbd\xb2\x3d\xbc\x20\xe2"
		require.False(t, utf8.ValidString(nonUTF8String))

		// Avoid using the `NewMeteredString()` constructor to skip the validation
		stringValue := cadence.String(nonUTF8String)

		script := `
            pub fun main(s: String) {
                log(s)
            }
        `

		encodedArg, err := json.Encode(stringValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		var validated bool

		runtimeInterface := &testRuntimeInterface{
			log: func(s string) {
				assert.True(t, utf8.ValidString(s))
				validated = true
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)

		assert.True(t, validated)
	})
}

func TestTypeValueImport(t *testing.T) {

	t.Parallel()

	t.Run("Type<Int>", func(t *testing.T) {

		t.Parallel()

		typeValue := cadence.NewTypeValue(cadence.IntType{})

		script := `
            pub fun main(s: Type) {
                log(s.identifier)
            }
        `

		encodedArg, err := json.Encode(typeValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		var ok bool

		runtimeInterface := &testRuntimeInterface{
			log: func(s string) {
				assert.Equal(t, "\"Int\"", s)
				ok = true
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("missing struct", func(t *testing.T) {

		t.Parallel()

		typeValue := cadence.NewTypeValue(&cadence.StructType{
			QualifiedIdentifier: "S",
			Location:            TestLocation,
			Fields:              []cadence.Field{},
			Initializers:        [][]cadence.Parameter{},
		})

		script := `
            pub fun main(s: Type) {
            }
        `

		encodedArg, err := json.Encode(typeValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
		require.IsType(t, interpreter.TypeLoadingError{}, err.(Error).Err.(*InvalidEntryPointArgumentError).Err)
	})
}

func TestPathCapabilityValueImport(t *testing.T) {

	t.Parallel()

	t.Run("public Capability<&Int>", func(t *testing.T) {

		t.Parallel()

		capabilityValue := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			&cadence.ReferenceType{Type: cadence.IntType{}},
		)

		script := `
            pub fun main(s: Capability<&Int>) {
                log(s)
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		var ok bool

		runtimeInterface := &testRuntimeInterface{
			log: func(s string) {
				assert.Equal(t, "Capability<&Int>(address: 0x0100000000000000, path: /public/foo)", s)
				ok = true
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("Capability<Int>", func(t *testing.T) {

		t.Parallel()

		capabilityValue := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			cadence.IntType{},
		)

		script := `
            pub fun main(s: Capability<Int>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
	})

	t.Run("private Capability<&Int>", func(t *testing.T) {

		t.Parallel()

		capabilityValue := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainPrivate,
				Identifier: "foo",
			},
			&cadence.ReferenceType{
				Type: cadence.IntType{},
			},
		)

		script := `
            pub fun main(s: Capability<&Int>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
	})

	t.Run("storage Capability<&Int>", func(t *testing.T) {

		t.Parallel()

		capabilityValue := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainStorage,
				Identifier: "foo",
			},
			&cadence.ReferenceType{Type: cadence.IntType{}},
		)

		script := `
            pub fun main(s: Capability<&Int>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
	})

	t.Run("missing struct", func(t *testing.T) {

		t.Parallel()

		borrowType := &cadence.StructType{
			QualifiedIdentifier: "S",
			Location:            TestLocation,
			Fields:              []cadence.Field{},
			Initializers:        [][]cadence.Parameter{},
		}

		capabilityValue := cadence.NewPathCapability(
			cadence.Address{0x1},
			cadence.Path{
				Domain:     common.PathDomainPublic,
				Identifier: "foo",
			},
			borrowType,
		)

		script := `
            pub fun main(s: Capability<S>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
	})
}

func TestIDCapabilityValueImport(t *testing.T) {

	t.Parallel()

	t.Run("Capability<&Int>", func(t *testing.T) {

		t.Parallel()

		capabilityValue := cadence.NewIDCapability(
			42,
			cadence.Address{0x1},
			&cadence.ReferenceType{Type: cadence.IntType{}},
		)

		script := `
            pub fun main(s: Capability<&Int>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
	})

	t.Run("Capability<Int>", func(t *testing.T) {

		t.Parallel()

		capabilityValue := cadence.NewIDCapability(
			3,
			cadence.Address{0x1},
			cadence.IntType{},
		)

		script := `
            pub fun main(s: Capability<Int>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
	})

	t.Run("missing struct", func(t *testing.T) {

		t.Parallel()

		borrowType := &cadence.StructType{
			QualifiedIdentifier: "S",
			Location:            TestLocation,
			Fields:              []cadence.Field{},
			Initializers:        [][]cadence.Parameter{},
		}

		capabilityValue := cadence.NewIDCapability(
			42,
			cadence.Address{0x1},
			borrowType,
		)

		script := `
            pub fun main(s: Capability<S>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		runtimeInterface := &testRuntimeInterface{
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err = rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)
	})
}

func TestRuntimePublicKeyImport(t *testing.T) {

	t.Parallel()

	executeScript := func(
		t *testing.T,
		script string,
		arg cadence.Value,
		runtimeInterface Interface,
	) (cadence.Value, error) {

		encodedArg, err := json.Encode(arg)
		require.NoError(t, err)

		rt := newTestInterpreterRuntime()

		return rt.ExecuteScript(
			Script{
				Source:    []byte(script),
				Arguments: [][]byte{encodedArg},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
	}

	publicKeyBytes := cadence.NewArray([]cadence.Value{
		cadence.NewUInt8(1),
		cadence.NewUInt8(2),
	})

	t.Run("Test importing validates PublicKey", func(t *testing.T) {
		t.Parallel()

		testPublicKeyImport := func(publicKeyActualError error) {
			t.Run(
				fmt.Sprintf("Actual(%v)", publicKeyActualError),
				func(t *testing.T) {

					t.Parallel()

					script := `
                        pub fun main(key: PublicKey) {
                        }
                    `

					publicKey := cadence.NewStruct(
						[]cadence.Value{
							// PublicKey bytes
							publicKeyBytes,

							// Sign algorithm
							cadence.NewEnum(
								[]cadence.Value{
									cadence.NewUInt8(1),
								},
							).WithType(SignAlgoType),
						},
					).WithType(PublicKeyType)

					var publicKeyValidated bool

					storage := newTestLedger(nil, nil)

					runtimeInterface := &testRuntimeInterface{
						storage: storage,
						validatePublicKey: func(publicKey *stdlib.PublicKey) error {
							publicKeyValidated = true
							return publicKeyActualError
						},
						meterMemory: func(_ common.MemoryUsage) error {
							return nil
						},
					}
					runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
						return json.Decode(runtimeInterface, b)
					}

					_, err := executeScript(t, script, publicKey, runtimeInterface)

					// runtimeInterface.validatePublicKey() should be called
					assert.True(t, publicKeyValidated)

					// Invalid PublicKey errors but valid PublicKey does not.
					if publicKeyActualError == nil {
						require.NoError(t, err)
					} else {
						RequireError(t, err)
						assertUserError(t, err)

						var invalidEntryPointArgumentError *InvalidEntryPointArgumentError
						assert.ErrorAs(t, err, &invalidEntryPointArgumentError)
						assert.ErrorAs(t, err, &interpreter.InvalidPublicKeyError{})
						assert.ErrorAs(t, err, &publicKeyActualError)
					}
				},
			)
		}

		testPublicKeyImport(nil)
		testPublicKeyImport(&fakeError{})
	})

	t.Run("Test Verify", func(t *testing.T) {
		t.Parallel()

		script := `
            pub fun main(key: PublicKey): Bool {
                return key.verify(
                    signature: [],
                    signedData: [],
                    domainSeparationTag: "",
                    hashAlgorithm: HashAlgorithm.SHA2_256
                )
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				// PublicKey bytes
				publicKeyBytes,

				// Sign algorithm
				cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(1),
					},
				).WithType(SignAlgoType),
			},
		).WithType(PublicKeyType)

		var verifyInvoked bool

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			verifySignature: func(
				signature []byte,
				tag string,
				signedData []byte,
				publicKey []byte,
				signatureAlgorithm SignatureAlgorithm,
				hashAlgorithm HashAlgorithm,
			) (bool, error) {
				verifyInvoked = true
				return true, nil
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}
		addPublicKeyValidation(runtimeInterface, nil)

		actual, err := executeScript(t, script, publicKey, runtimeInterface)
		require.NoError(t, err)

		assert.True(t, verifyInvoked)
		assert.Equal(t, cadence.NewBool(true), actual)
	})

	t.Run("Invalid raw public key", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				// Invalid value for 'publicKey' field
				cadence.NewBool(true),

				cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(0),
					},
				).WithType(SignAlgoType),
			},
		).WithType(PublicKeyType)

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid content in public key", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				// Invalid content for 'publicKey' field
				cadence.NewArray([]cadence.Value{
					cadence.String("1"),
					cadence.String("2"),
				}),

				cadence.NewEnum(
					[]cadence.Value{
						cadence.NewUInt8(0),
					},
				).WithType(SignAlgoType),
			},
		).WithType(PublicKeyType)

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid sign algo", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				publicKeyBytes,

				// Invalid value for 'signatureAlgorithm' field
				cadence.NewBool(true),
			},
		).WithType(PublicKeyType)

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid sign algo fields", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				publicKeyBytes,

				// Invalid value for fields of 'signatureAlgorithm'
				cadence.NewEnum(
					[]cadence.Value{
						cadence.String("hello"),
					},
				).WithType(SignAlgoType),
			},
		).WithType(PublicKeyType)

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Extra field", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey) {
            }
        `

		jsonCdc := `
            {
                "type":"Struct",
                "value":{
                    "id":"PublicKey",
                    "fields":[
                        {
                            "name":"publicKey",
                            "value":{
                                "type":"Array",
                                "value":[
                                    {
                                        "type":"UInt8",
                                        "value":"1"
                                    },
                                    {
                                        "type":"UInt8",
                                        "value":"2"
                                    }
                                ]
                            }
                        },
                        {
                            "name":"signatureAlgorithm",
                            "value":{
                                "type":"Enum",
                                "value":{
                                    "id":"SignatureAlgorithm",
                                    "fields":[
                                        {
                                            "name":"rawValue",
                                            "value":{
                                                "type":"UInt8",
                                                "value":"0"
                                            }
                                        }
                                    ]
                                }
                            }
                        },
                        {
                            "name":"extraField",
                            "value":{
                            "type":"Bool",
                            "value":true
                            }
                        }
                    ]
                }
            }
        `

		rt := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Missing raw public key", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey): PublicKey {
                return key
            }
        `

		jsonCdc := `
            {
                "type":"Struct",
                "value":{
                    "id":"PublicKey",
                    "fields":[
                        {
                            "name":"signatureAlgorithm",
                            "value":{
                                "type":"Enum",
                                "value":{
                                    "id":"SignatureAlgorithm",
                                    "fields":[
                                        {
                                            "name":"rawValue",
                                            "value":{
                                                "type":"UInt8",
                                                "value":"0"
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

		rt := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Missing publicKey", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey): [UInt8] {
                return key.publicKey
            }
        `

		jsonCdc := `
            {
                "type":"Struct",
                "value":{
                    "id":"PublicKey",
                    "fields":[
                        {
                            "name":"signatureAlgorithm",
                            "value":{
                                "type":"Enum",
                                "value":{
                                    "id":"SignatureAlgorithm",
                                    "fields":[
                                        {
                                            "name":"rawValue",
                                            "value":{
                                                "type":"UInt8",
                                                "value":"1"
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

		rt := newTestInterpreterRuntime()

		var publicKeyValidated bool

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			validatePublicKey: func(publicKey *stdlib.PublicKey) error {
				publicKeyValidated = true
				return nil
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		value, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		assert.Contains(t, err.Error(),
			"invalid argument at index 0: cannot import value of type 'PublicKey'. missing field 'publicKey'")
		assert.False(t, publicKeyValidated)
		assert.Nil(t, value)
	})

	t.Run("Missing signatureAlgorithm", func(t *testing.T) {
		script := `
            pub fun main(key: PublicKey): SignatureAlgorithm {
                return key.signatureAlgorithm
            }
        `

		jsonCdc := `
            {
                "type":"Struct",
                "value":{
                    "id":"PublicKey",
                    "fields":[
                        {
                            "name":"publicKey",
                            "value":{
                                "type":"Array",
                                "value":[
                                    {
                                        "type":"UInt8",
                                        "value":"1"
                                    },
                                    {
                                        "type":"UInt8",
                                        "value":"2"
                                    }
                                ]
                            }
                        }
                    ]
                }
            }
        `

		rt := newTestInterpreterRuntime()

		var publicKeyValidated bool

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			validatePublicKey: func(publicKey *stdlib.PublicKey) error {
				publicKeyValidated = true
				return nil
			},
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		value, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		assert.Contains(t, err.Error(),
			"invalid argument at index 0: cannot import value of type 'PublicKey'. missing field 'signatureAlgorithm'")
		assert.False(t, publicKeyValidated)
		assert.Nil(t, value)
	})

}

func TestRuntimeImportExportComplex(t *testing.T) {

	t.Parallel()

	program := interpreter.Program{
		Elaboration: sema.NewElaboration(nil),
	}

	inter := newTestInterpreter(t)
	inter.Program = &program

	// Array

	semaArrayType := &sema.VariableSizedType{
		Type: sema.AnyStructType,
	}

	staticArrayType := interpreter.VariableSizedStaticType{
		Type: interpreter.PrimitiveStaticTypeAnyStruct,
	}

	externalArrayType := &cadence.VariableSizedArrayType{
		ElementType: cadence.AnyStructType{},
	}

	internalArrayValue := interpreter.NewArrayValue(
		inter,
		interpreter.EmptyLocationRange,
		staticArrayType,
		common.ZeroAddress,
		interpreter.NewUnmeteredIntValueFromInt64(42),
		interpreter.NewUnmeteredStringValue("foo"),
	)

	externalArrayValue := cadence.NewArray([]cadence.Value{
		cadence.NewInt(42),
		cadence.String("foo"),
	}).WithType(&cadence.VariableSizedArrayType{
		ElementType: cadence.AnyStructType{},
	})

	// Dictionary

	semaDictionaryType := &sema.DictionaryType{
		KeyType:   sema.StringType,
		ValueType: semaArrayType,
	}

	staticDictionaryType := interpreter.DictionaryStaticType{
		KeyType:   interpreter.PrimitiveStaticTypeString,
		ValueType: staticArrayType,
	}

	externalDictionaryType := &cadence.DictionaryType{
		KeyType:     cadence.StringType{},
		ElementType: externalArrayType,
	}

	internalDictionaryValue := interpreter.NewDictionaryValue(
		inter,
		interpreter.EmptyLocationRange,
		staticDictionaryType,
		interpreter.NewUnmeteredStringValue("a"), internalArrayValue,
	)

	externalDictionaryValue := cadence.NewDictionary([]cadence.KeyValuePair{
		{
			Key:   cadence.String("a"),
			Value: externalArrayValue,
		},
	}).WithType(&cadence.DictionaryType{
		KeyType: cadence.StringType{},
		ElementType: &cadence.VariableSizedArrayType{
			ElementType: cadence.AnyStructType{},
		},
	})

	// Composite

	semaCompositeType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "Foo",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
		Fields:     []string{"dictionary"},
	}

	program.Elaboration.SetCompositeType(
		semaCompositeType.ID(),
		semaCompositeType,
	)

	semaCompositeType.Members.Set(
		"dictionary",
		sema.NewUnmeteredPublicConstantFieldMember(
			semaCompositeType,
			"dictionary",
			semaDictionaryType,
			"",
		),
	)

	externalCompositeType := &cadence.StructType{
		Location:            TestLocation,
		QualifiedIdentifier: "Foo",
		Fields: []cadence.Field{
			{
				Identifier: "dictionary",
				Type:       externalDictionaryType,
			},
		},
	}

	internalCompositeValueFields := []interpreter.CompositeField{
		{
			Name:  "dictionary",
			Value: internalDictionaryValue,
		},
	}

	internalCompositeValue := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		TestLocation,
		"Foo",
		common.CompositeKindStructure,
		internalCompositeValueFields,
		common.ZeroAddress,
	)

	externalCompositeValue := cadence.Struct{
		StructType: externalCompositeType,
		Fields: []cadence.Value{
			externalDictionaryValue,
		},
	}

	t.Run("export", func(t *testing.T) {

		// NOTE: cannot be parallel, due to type's ID being cached (potential data race)

		actual, err := exportValueWithInterpreter(
			internalCompositeValue,
			inter,
			interpreter.EmptyLocationRange,
			seenReferences{},
		)
		require.NoError(t, err)

		assert.Equal(t,
			externalCompositeValue,
			actual,
		)
	})

	t.Run("import", func(t *testing.T) {

		// NOTE: cannot be parallel, due to type's ID being cached (potential data race)

		program := interpreter.Program{
			Elaboration: sema.NewElaboration(nil),
		}

		inter := newTestInterpreter(t)
		inter.Program = &program

		program.Elaboration.SetCompositeType(
			semaCompositeType.ID(),
			semaCompositeType,
		)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			externalCompositeValue,
			semaCompositeType,
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			internalCompositeValue,
			actual,
		)
	})
}

func TestRuntimeStaticTypeAvailability(t *testing.T) {

	t.Parallel()

	t.Run("inner array", func(t *testing.T) {
		script := `
            pub fun main(arg: Foo) {
            }

            pub struct Foo {
                pub var a: AnyStruct

                init() {
                    self.a = nil
                }
            }
        `

		structValue := cadence.Struct{
			StructType: &cadence.StructType{
				Location:            common.ScriptLocation{},
				QualifiedIdentifier: "Foo",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.AnyStructType{},
					},
				},
			},

			Fields: []cadence.Value{
				cadence.NewArray([]cadence.Value{
					cadence.String("foo"),
					cadence.String("bar"),
				}),
			},
		}

		_, err := executeTestScript(t, script, structValue)
		require.NoError(t, err)
	})

	t.Run("inner dictionary", func(t *testing.T) {
		script := `
            pub fun main(arg: Foo) {
            }

            pub struct Foo {
                pub var a: AnyStruct

                init() {
                    self.a = nil
                }
            }
        `

		structValue := cadence.Struct{
			StructType: &cadence.StructType{
				Location:            common.ScriptLocation{},
				QualifiedIdentifier: "Foo",
				Fields: []cadence.Field{
					{
						Identifier: "a",
						Type:       cadence.AnyStructType{},
					},
				},
			},

			Fields: []cadence.Value{
				cadence.NewDictionary([]cadence.KeyValuePair{
					{
						Key:   cadence.String("foo"),
						Value: cadence.String("bar"),
					},
				}),
			},
		}

		_, err := executeTestScript(t, script, structValue)
		require.NoError(t, err)
	})
}

func newTestInterpreter(tb testing.TB) *interpreter.Interpreter {
	storage := newUnmeteredInMemoryStorage()

	inter, err := interpreter.NewInterpreter(
		nil,
		TestLocation,
		&interpreter.Config{
			Storage:                       storage,
			AtreeValueValidationEnabled:   true,
			AtreeStorageValidationEnabled: true,
		},
	)
	require.NoError(tb, err)

	return inter
}

func newUnmeteredInMemoryStorage() interpreter.Storage {
	return interpreter.NewInMemoryStorage(nil)
}

func TestNestedStructArgPassing(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(v: AnyStruct): UInt8 {
                return (v as! Foo).bytes[0]
            }

            pub struct Foo {
                pub let bytes: [UInt8]

                init(_ bytes: [UInt8]) {
                    self.bytes = bytes
               }
            }
        `

		jsonCdc := `
          {
            "value": {
              "id": "s.0000000000000000000000000000000000000000000000000000000000000000.Foo",
              "fields": [
                {
                  "name": "bytes",
                  "value": {
                    "value": [
                      {
                        "value": "32",
                        "type": "UInt8"
                      }
                    ],
                    "type": "Array"
                  }
                }
              ]
            },
            "type": "Struct"
          }
        `

		rt := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}

		value, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, cadence.NewUInt8(32), value)
	})

	t.Run("invalid interface", func(t *testing.T) {

		t.Parallel()

		script := `
            pub fun main(v: AnyStruct) {
            }

            pub struct interface Foo {
            }
        `

		jsonCdc := `
          {
            "value": {
              "id": "s.0000000000000000000000000000000000000000000000000000000000000000.Foo",
              "fields": [
                {
                  "name": "bytes",
                  "value": {
                    "value": [
                      {
                        "value": "32",
                        "type": "UInt8"
                      }
                    ],
                    "type": "Array"
                  }
                }
              ]
            },
            "type": "Struct"
          }
        `

		rt := newTestInterpreterRuntime()

		storage := newTestLedger(nil, nil)

		runtimeInterface := &testRuntimeInterface{
			storage: storage,
			meterMemory: func(_ common.MemoryUsage) error {
				return nil
			},
		}
		runtimeInterface.decodeArgument = func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(runtimeInterface, b)
		}
		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
				Arguments: [][]byte{
					[]byte(jsonCdc),
				},
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})
}

func TestDestroyedResourceReferenceExport(t *testing.T) {
	t.Parallel()

	rt := newTestInterpreterRuntimeWithAttachments()

	script := []byte(`
        pub resource S {}

        pub fun main(): &S  {
            var s <- create S()
            var ref = &s as &S
            destroy s
            return ref
        }
	 `)

	runtimeInterface := &testRuntimeInterface{}

	nextScriptLocation := newScriptLocationGenerator()
	_, err := rt.ExecuteScript(
		Script{
			Source: script,
		},
		Context{
			Interface: runtimeInterface,
			Location:  nextScriptLocation(),
		},
	)
	require.Error(t, err)
	require.ErrorAs(t, err, &interpreter.DestroyedResourceError{})
}
