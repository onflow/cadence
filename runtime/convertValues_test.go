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
	_ "embed"
	"fmt"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/parser"
	. "github.com/onflow/cadence/runtime"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/runtime_utils"
)

func TestRuntimeExportValue(t *testing.T) {

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

			inter := NewTestInterpreter(t)

			value := tt.value
			if tt.valueFactory != nil {
				value = tt.valueFactory(inter)
			}
			actual, err := ExportValue(
				value,
				inter,
				interpreter.EmptyLocationRange,
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
		return cadence.NewEnumType(
			nil,
			"SignatureAlgorithm",
			cadence.UInt8Type,
			[]cadence.Field{
				{
					Identifier: "rawValue",
					Type:       cadence.UInt8Type,
				},
			},
			nil,
		)
	}

	newPublicKeyType := func(signatureAlgorithmType cadence.Type) *cadence.StructType {
		return cadence.NewStructType(
			nil,
			"PublicKey",
			[]cadence.Field{
				{
					Identifier: "publicKey",
					Type: &cadence.VariableSizedArrayType{
						ElementType: cadence.UInt8Type,
					},
				},
				{
					Identifier: "signatureAlgorithm",
					Type:       signatureAlgorithmType,
				},
			},
			nil,
		)
	}

	newHashAlgorithmType := func() *cadence.EnumType {
		return cadence.NewEnumType(
			nil,
			"HashAlgorithm",
			cadence.UInt8Type,
			[]cadence.Field{
				{
					Identifier: "rawValue",
					Type:       cadence.UInt8Type,
				},
			},
			nil,
		)
	}

	testCharacter, _ := cadence.NewCharacter("a")

	testFunction := &interpreter.InterpretedFunctionValue{
		Type: sema.NewSimpleFunctionType(
			sema.FunctionPurityImpure,
			nil,
			sema.VoidTypeAnnotation,
		),
	}

	testFunctionType := cadence.NewFunctionType(
		sema.FunctionPurityImpure,
		nil,
		nil,
		cadence.VoidType,
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
					&interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeAnyStruct,
					},
					common.ZeroAddress,
				)
			},
			expected: cadence.NewArray([]cadence.Value{}).
				WithType(&cadence.VariableSizedArrayType{
					ElementType: cadence.AnyStructType,
				}),
		},
		{
			label: "Array (non-empty)",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.VariableSizedStaticType{
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
				ElementType: cadence.AnyStructType,
			}),
		},
		{
			label: "Array (non-empty) with HashableStruct",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeHashableStruct,
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
				ElementType: cadence.HashableStructType,
			}),
		},
		{
			label: "Dictionary",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeString,
						ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
					},
				)
			},
			expected: cadence.NewDictionary([]cadence.KeyValuePair{}).
				WithType(&cadence.DictionaryType{
					KeyType:     cadence.StringType,
					ElementType: cadence.AnyStructType,
				}),
		},
		{
			label: "Dictionary (non-empty)",
			valueFactory: func(inter *interpreter.Interpreter) interpreter.Value {
				return interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.DictionaryStaticType{
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
					KeyType:     cadence.StringType,
					ElementType: cadence.AnyStructType,
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

				return cadence.NewStruct([]cadence.Value{
					cadence.NewInt(1),
					cadence.NewStruct([]cadence.Value{
						cadence.NewArray([]cadence.Value{
							cadence.NewUInt8(1),
							cadence.NewUInt8(2),
							cadence.NewUInt8(3),
						}).WithType(&cadence.VariableSizedArrayType{
							ElementType: cadence.UInt8Type,
						}),
						cadence.NewEnum([]cadence.Value{
							cadence.UInt8(2),
						}).WithType(signatureAlgorithmType),
					}).WithType(publicKeyType),
					cadence.NewEnum([]cadence.Value{
						cadence.UInt8(1),
					}).WithType(hashAlgorithmType),
					cadence.UFix64(10_00000000),
					cadence.Bool(false),
				}).WithType(cadence.NewStructType(
					nil,
					"AccountKey",
					[]cadence.Field{
						{
							Identifier: "keyIndex",
							Type:       cadence.IntType,
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
							Type:       cadence.UFix64Type,
						},
						{
							Identifier: "isRevoked",
							Type:       cadence.BoolType,
						},
					},
					nil,
				))
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
		{
			label: "path capability, typed",
			value: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				interpreter.PrimitiveStaticTypeAnyResource,
				interpreter.AddressValue{0x1},
				interpreter.PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
			),
			expected: cadence.NewDeprecatedPathCapability( //nolint:staticcheck
				cadence.Address{0x1},
				cadence.MustNewPath(common.PathDomainStorage, "foo"),
				cadence.AnyResourceType,
			),
		},
		{
			label: "path capability, untyped",
			value: interpreter.NewUnmeteredPathCapabilityValue( //nolint:staticcheck
				// NOTE: no borrow type
				nil,
				interpreter.AddressValue{0x1},
				interpreter.PathValue{
					Domain:     common.PathDomainStorage,
					Identifier: "foo",
				},
			),
			expected: cadence.NewDeprecatedPathCapability( //nolint:staticcheck
				cadence.Address{0x1},
				cadence.MustNewPath(common.PathDomainStorage, "foo"),
				nil,
			),
		},
	} {
		test(tt)
	}

}

func TestRuntimeImportValue(t *testing.T) {

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

			inter := NewTestInterpreter(t)

			actual, err := ImportValue(
				inter,
				interpreter.EmptyLocationRange,
				nil,
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
				NewTestInterpreter(t),
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
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
				NewTestInterpreter(t),
				interpreter.EmptyLocationRange,
				&interpreter.VariableSizedStaticType{
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
				NewTestInterpreter(t),
				interpreter.EmptyLocationRange,
				&interpreter.DictionaryStaticType{
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
				NewTestInterpreter(t),
				interpreter.EmptyLocationRange,
				&interpreter.DictionaryStaticType{
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
			label: "ID Capability (invalid)",
			value: cadence.NewCapability(
				4,
				cadence.Address{0x1},
				cadence.IntType,
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
			value:    cadence.NewTypeValue(cadence.IntType),
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

func TestRuntimeImportRuntimeType(t *testing.T) {
	t.Parallel()

	type importTest struct {
		label    string
		expected interpreter.StaticType
		input    cadence.Type
	}

	test := func(tt importTest) {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()
			actual := ImportType(nil, tt.input)
			assert.Equal(t, tt.expected, actual)

		})
	}

	tests := []importTest{
		{
			label: "AccountKey",
			input: &cadence.StructType{
				QualifiedIdentifier: "AccountKey",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, nil, "AccountKey"),
		},
		{
			label: "PublicKey",
			input: &cadence.StructType{
				QualifiedIdentifier: "PublicKey",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, nil, "PublicKey"),
		},
		{
			label: "HashAlgorithm",
			input: &cadence.StructType{
				QualifiedIdentifier: "HashAlgorithm",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, nil, "HashAlgorithm"),
		},
		{
			label: "SignatureAlgorithm",
			input: &cadence.StructType{
				QualifiedIdentifier: "SignatureAlgorithm",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, nil, "SignatureAlgorithm"),
		},
		{
			label: "Optional",
			input: &cadence.OptionalType{
				Type: cadence.IntType,
			},
			expected: &interpreter.OptionalStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "VariableSizedArray",
			input: &cadence.VariableSizedArrayType{
				ElementType: cadence.IntType,
			},
			expected: &interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "ConstantSizedArray",
			input: &cadence.ConstantSizedArrayType{
				ElementType: cadence.IntType,
				Size:        3,
			},
			expected: &interpreter.ConstantSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeInt,
				Size: 3,
			},
		},
		{
			label: "Dictionary",
			input: &cadence.DictionaryType{
				ElementType: cadence.IntType,
				KeyType:     cadence.StringType,
			},
			expected: &interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Unauthorized Reference",
			input: &cadence.ReferenceType{
				Authorization: cadence.UnauthorizedAccess,
				Type:          cadence.IntType,
			},
			expected: &interpreter.ReferenceStaticType{
				Authorization:  interpreter.UnauthorizedAccess,
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Entitlement Set Reference",
			input: &cadence.ReferenceType{
				Authorization: &cadence.EntitlementSetAuthorization{
					Kind:         cadence.Conjunction,
					Entitlements: []common.TypeID{"E", "F"},
				},
				Type: cadence.IntType,
			},
			expected: &interpreter.ReferenceStaticType{
				Authorization: interpreter.NewEntitlementSetAuthorization(
					nil,
					func() []common.TypeID { return []common.TypeID{"E", "F"} },
					2,
					sema.Conjunction,
				),
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Reference",
			input: &cadence.ReferenceType{
				Authorization: &cadence.EntitlementSetAuthorization{
					Kind:         cadence.Disjunction,
					Entitlements: []common.TypeID{"E", "F"},
				},
				Type: cadence.IntType,
			},
			expected: &interpreter.ReferenceStaticType{
				Authorization: interpreter.NewEntitlementSetAuthorization(
					nil,
					func() []common.TypeID { return []common.TypeID{"E", "F"} },
					2,
					sema.Disjunction),
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Entitlement Map Reference",
			input: &cadence.ReferenceType{
				Authorization: cadence.EntitlementMapAuthorization{
					TypeID: "M",
				},
				Type: cadence.IntType,
			},
			expected: &interpreter.ReferenceStaticType{
				Authorization: interpreter.EntitlementMapAuthorization{
					TypeID: "M",
				},
				ReferencedType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Capability",
			input: &cadence.CapabilityType{
				BorrowType: cadence.IntType,
			},
			expected: &interpreter.CapabilityStaticType{
				BorrowType: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			label: "Struct",
			input: &cadence.StructType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "Resource",
			input: &cadence.ResourceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "Contract",
			input: &cadence.ContractType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "Event",
			input: &cadence.EventType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "Enum",
			input: &cadence.EnumType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "StructInterface",
			input: &cadence.StructInterfaceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "ResourceInterface",
			input: &cadence.ResourceInterfaceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "ContractInterface",
			input: &cadence.ContractInterfaceType{
				Location:            TestLocation,
				QualifiedIdentifier: "S",
			},
			expected: interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "S"),
		},
		{
			label: "IntersectionType",
			input: &cadence.IntersectionType{
				Types: []cadence.Type{
					&cadence.StructInterfaceType{
						Location:            TestLocation,
						QualifiedIdentifier: "T",
					},
				},
			},
			expected: &interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "T"),
				},
			},
		},
		{
			label: "InclusiveRange",
			input: &cadence.InclusiveRangeType{
				ElementType: cadence.IntType,
			},
			expected: interpreter.InclusiveRangeStaticType{
				ElementType: interpreter.PrimitiveStaticTypeInt,
			},
		},
	}

	for ty := interpreter.PrimitiveStaticTypeUnknown + 1; ty < interpreter.PrimitiveStaticType_Count; ty++ {
		if !ty.IsDefined() {
			continue
		}

		tests = append(tests, importTest{
			label:    fmt.Sprintf("%s (primitive)", ty),
			input:    cadence.PrimitiveType(ty),
			expected: ty,
		})

		typeID := ty.ID()
		qualifiedIdentifier := string(typeID)

		var expectedForComposite interpreter.StaticType
		if ty.IsDeprecated() { //nolint:staticcheck
			expectedForComposite = interpreter.NewCompositeStaticType(
				nil,
				nil,
				qualifiedIdentifier,
				typeID,
			)
		} else {
			expectedForComposite = ty
		}

		tests = append(tests, importTest{
			label: fmt.Sprintf("%s (composite)", ty),
			input: &cadence.StructType{
				Location:            nil,
				QualifiedIdentifier: qualifiedIdentifier,
			},
			expected: expectedForComposite,
		})
	}

	for _, tt := range tests {
		test(tt)
	}
}

func TestRuntimeExportIntegerValuesFromScript(t *testing.T) {

	t.Parallel()

	test := func(integerType sema.Type) {

		t.Run(integerType.String(), func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`
                  access(all) fun main(): %s {
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

func TestRuntimeExportFixedPointValuesFromScript(t *testing.T) {

	t.Parallel()

	test := func(fixedPointType sema.Type, literal string) {

		t.Run(fixedPointType.String(), func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`
                  access(all) fun main(): %s {
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

func TestRuntimeExportAddressValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) fun main(): Address {
            return 0x42
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.BytesToAddress(
		[]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x42},
	)

	assert.Equal(t, expected, actual)
}

func TestExportInclusiveRangeValue(t *testing.T) {

	t.Parallel()

	t.Run("with step", func(t *testing.T) {

		t.Parallel()

		script := `
			access(all) fun main(): InclusiveRange<Int> {
				return InclusiveRange(10, 20, step: 2)
			}
		`

		inclusiveRangeType := cadence.NewInclusiveRangeType(cadence.IntType)

		actual := exportValueFromScript(t, script)
		expected := cadence.NewInclusiveRange(
			cadence.NewInt(10),
			cadence.NewInt(20),
			cadence.NewInt(2),
		).WithType(inclusiveRangeType)

		assert.Equal(t, expected, actual)
	})

	t.Run("without step", func(t *testing.T) {

		t.Parallel()

		script := `
			access(all) fun main(): InclusiveRange<Int> {
				return InclusiveRange(10, 20)
			}
		`

		inclusiveRangeType := cadence.NewInclusiveRangeType(cadence.IntType)

		actual := exportValueFromScript(t, script)
		expected := cadence.NewInclusiveRange(
			cadence.NewInt(10),
			cadence.NewInt(20),
			cadence.NewInt(1),
		).WithType(inclusiveRangeType)

		assert.Equal(t, expected, actual)
	})
}

func TestImportInclusiveRangeValue(t *testing.T) {

	t.Parallel()

	t.Run("simple - InclusiveRange<Int>", func(t *testing.T) {
		t.Parallel()

		value := cadence.NewInclusiveRange(cadence.NewInt(10), cadence.NewInt(-10), cadence.NewInt(-2))

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			nil,
			value,
			sema.NewInclusiveRangeType(inter, sema.IntType),
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewInclusiveRangeValueWithStep(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewIntValueFromInt64(inter, 10),
				interpreter.NewIntValueFromInt64(inter, -10),
				interpreter.NewIntValueFromInt64(inter, -2),
				interpreter.InclusiveRangeStaticType{
					ElementType: interpreter.PrimitiveStaticTypeInt,
				},
				sema.NewInclusiveRangeType(nil, sema.IntType),
			),
			actual,
		)
	})

	t.Run("import with broader type - AnyStruct", func(t *testing.T) {
		t.Parallel()

		value := cadence.NewInclusiveRange(cadence.NewInt(10), cadence.NewInt(-10), cadence.NewInt(-2))

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			nil,
			value,
			sema.AnyStructType,
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewInclusiveRangeValueWithStep(
				inter,
				interpreter.EmptyLocationRange,
				interpreter.NewIntValueFromInt64(inter, 10),
				interpreter.NewIntValueFromInt64(inter, -10),
				interpreter.NewIntValueFromInt64(inter, -2),
				interpreter.InclusiveRangeStaticType{
					ElementType: interpreter.PrimitiveStaticTypeInt,
				},
				sema.NewInclusiveRangeType(nil, sema.IntType),
			),
			actual,
		)
	})

	t.Run("invalid - mixed types", func(t *testing.T) {
		t.Parallel()

		value := cadence.NewInclusiveRange(cadence.NewInt(10), cadence.NewUInt(100), cadence.NewUInt64(1))

		inter := NewTestInterpreter(t)

		_, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			nil,
			value,
			sema.AnyStructType,
		)

		RequireError(t, err)
		assertUserError(t, err)

		var userError errors.DefaultUserError
		require.ErrorAs(t, err, &userError)
		require.Contains(
			t,
			userError.Error(),
			"cannot import InclusiveRange: start, end and step must be of the same type",
		)
	})

	t.Run("invalid - InclusiveRange<String>", func(t *testing.T) {
		t.Parallel()

		strValue, err := cadence.NewString("anything")
		require.NoError(t, err)

		value := cadence.NewInclusiveRange(strValue, strValue, strValue)

		inter := NewTestInterpreter(t)

		_, err = ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			nil,
			value,
			sema.StringType,
		)

		RequireError(t, err)
		assertUserError(t, err)

		var userError errors.DefaultUserError
		require.ErrorAs(t, err, &userError)
		require.Contains(
			t,
			userError.Error(),
			"cannot import InclusiveRange: start, end and step must be integers",
		)
	})
}

func TestRuntimeExportStructValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) struct Foo {
            access(all) let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        access(all) fun main(): Foo {
            return Foo(bar: 42)
        }
    `

	fooStructType := cadence.NewStructType(
		common.ScriptLocation{},
		"Foo",
		fooFields,
		nil,
	)

	actual := exportValueFromScript(t, script)
	expected := cadence.NewStruct([]cadence.Value{
		cadence.NewInt(42),
	}).WithType(fooStructType)

	assert.Equal(t, expected, actual)
}

func TestRuntimeExportResourceValue(t *testing.T) {

	t.Parallel()

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

	actual := exportValueFromScript(t, script)
	expected := cadence.NewResource([]cadence.Value{
		cadence.NewUInt64(1),
		cadence.NewInt(42),
	}).WithType(newFooResourceType())

	assert.Equal(t, expected, actual)
}

func TestRuntimeExportResourceArrayValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) resource Foo {
            access(all) let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        access(all) fun main(): @[Foo] {
            return <- [<- create Foo(bar: 3), <- create Foo(bar: 4)]
        }
    `

	fooResourceType := newFooResourceType()

	actual := exportValueFromScript(t, script)

	expected := cadence.NewArray([]cadence.Value{
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(1),
			cadence.NewInt(3),
		}).WithType(fooResourceType),
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(2),
			cadence.NewInt(4),
		}).WithType(fooResourceType),
	}).WithType(cadence.NewVariableSizedArrayType(
		cadence.NewResourceType(
			common.ScriptLocation{},
			"Foo",
			[]cadence.Field{
				{
					Identifier: "uuid",
					Type:       cadence.UInt64Type,
				},
				{
					Identifier: "bar",
					Type:       cadence.IntType,
				},
			},
			nil,
		),
	))

	assert.Equal(t, expected, actual)
}

func TestRuntimeExportResourceDictionaryValue(t *testing.T) {

	t.Parallel()

	script := `
        access(all) resource Foo {
            access(all) let bar: Int

            init(bar: Int) {
                self.bar = bar
            }
        }

        access(all) fun main(): @{String: Foo} {
            return <- {
                "a": <- create Foo(bar: 3),
                "b": <- create Foo(bar: 4)
            }
        }
    `

	fooResourceType := newFooResourceType()

	actual := exportValueFromScript(t, script)

	expected := cadence.NewDictionary([]cadence.KeyValuePair{
		{
			Key: cadence.String("b"),
			Value: cadence.NewResource([]cadence.Value{
				cadence.NewUInt64(2),
				cadence.NewInt(4),
			}).WithType(fooResourceType),
		},
		{
			Key: cadence.String("a"),
			Value: cadence.NewResource([]cadence.Value{
				cadence.NewUInt64(1),
				cadence.NewInt(3),
			}).WithType(fooResourceType),
		},
	}).WithType(&cadence.DictionaryType{
		KeyType: cadence.StringType,
		ElementType: cadence.NewResourceType(
			common.ScriptLocation{},
			"Foo",
			[]cadence.Field{
				{
					Identifier: "uuid",
					Type:       cadence.UInt64Type,
				},
				{
					Identifier: "bar",
					Type:       cadence.IntType,
				},
			},
			nil,
		),
	})

	assert.Equal(t, expected, actual)
}

func TestRuntimeExportNestedResourceValueFromScript(t *testing.T) {

	t.Parallel()

	barResourceType := cadence.NewResourceType(
		common.ScriptLocation{},
		"Bar",
		[]cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "x",
				Type:       cadence.IntType,
			},
		},
		nil,
	)

	fooResourceType := cadence.NewResourceType(
		common.ScriptLocation{},
		"Foo",
		[]cadence.Field{
			{
				Identifier: "uuid",
				Type:       cadence.UInt64Type,
			},
			{
				Identifier: "bar",
				Type:       barResourceType,
			},
		},
		nil,
	)

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
        }

        access(all) fun main(): @Foo {
            return <- create Foo(bar: <- create Bar(x: 42))
        }
    `

	actual := exportValueFromScript(t, script)
	expected := cadence.NewResource([]cadence.Value{
		cadence.NewUInt64(2),
		cadence.NewResource([]cadence.Value{
			cadence.NewUInt64(1),
			cadence.NewInt(42),
		}).WithType(barResourceType),
	}).WithType(fooResourceType)

	assert.Equal(t, expected, actual)
}

func TestRuntimeExportEventValue(t *testing.T) {

	t.Parallel()

	t.Run("primitive", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          event Foo(bar: Int)

          access(all)
          fun main() {
              emit Foo(bar: 42)
          }
        `

		fooEventType := cadence.NewEventType(
			common.ScriptLocation{},
			"Foo",
			fooFields,
			nil,
		)

		actual := exportEventFromScript(t, script)
		expected := cadence.NewEvent([]cadence.Value{
			cadence.NewInt(42),
		}).WithType(fooEventType)

		assert.Equal(t, expected, actual)
	})

	t.Run("reference", func(t *testing.T) {
		t.Parallel()

		script := `
          access(all)
          event Foo(bar: &Int)

          access(all)
          fun main() {
              emit Foo(bar: &42 as &Int)
          }
        `

		fooEventType := cadence.NewEventType(
			common.ScriptLocation{},
			"Foo",
			[]cadence.Field{
				{
					Identifier: "bar",
					Type: cadence.NewReferenceType(
						cadence.UnauthorizedAccess,
						cadence.IntType,
					),
				},
			},
			nil,
		)

		actual := exportEventFromScript(t, script)
		expected := cadence.NewEvent([]cadence.Value{
			cadence.NewInt(42),
		}).WithType(fooEventType)

		assert.Equal(t, expected, actual)
	})
}

func exportEventFromScript(t *testing.T, script string) cadence.Event {
	rt := NewTestInterpreterRuntime()

	var events []cadence.Event

	inter := &TestRuntimeInterface{
		OnEmitEvent: func(event cadence.Event) error {
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
	rt := NewTestInterpreterRuntime()

	value, err := rt.ExecuteScript(
		Script{
			Source: []byte(script),
		},
		Context{
			Interface: &TestRuntimeInterface{},
			Location:  common.ScriptLocation{},
		},
	)

	require.NoError(t, err)

	return value
}

func TestRuntimeExportReferenceValue(t *testing.T) {

	t.Parallel()

	t.Run("ephemeral, Int", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(): &Int {
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
            access(all) fun main(): [&AnyStruct] {
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
					Type:          cadence.AnyStructType,
					Authorization: cadence.UnauthorizedAccess,
				},
			}),
		}).WithType(&cadence.VariableSizedArrayType{
			ElementType: &cadence.ReferenceType{
				Type:          cadence.AnyStructType,
				Authorization: cadence.UnauthorizedAccess,
			},
		})

		assert.Equal(t, expected, actual)
	})

	t.Run("ephemeral, invalidated", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) resource R {}

            access(all) struct S {
                access(all) let ref: &R

                init(ref: &R) {
                    self.ref = ref
                }
            }

            access(all) fun main(): S {
                let r <- create R()
                let s = S(ref: &r as &R)
                destroy r
                return s
            }
        `

		actual := exportValueFromScript(t, script)

		expected := cadence.NewStruct([]cadence.Value{nil}).
			WithType(cadence.NewStructType(
				common.ScriptLocation{},
				"S",
				[]cadence.Field{
					{
						Type: cadence.NewReferenceType(
							cadence.UnauthorizedAccess,
							cadence.NewResourceType(
								common.ScriptLocation{},
								"R",
								[]cadence.Field{
									{
										Identifier: sema.ResourceUUIDFieldName,
										Type:       cadence.UInt64Type,
									},
								},
								nil,
							),
						),
						Identifier: "ref",
					},
				},
				nil,
			))
		assert.Equal(t, expected, actual)
	})

	t.Run("storage", func(t *testing.T) {

		t.Parallel()

		// Arrange

		rt := NewTestInterpreterRuntime()

		transaction := `
            transaction {
                prepare(signer: auth(Storage, Capabilities) &Account) {
                    signer.storage.save(1, to: /storage/test)
                    let cap = signer.capabilities.storage.issue<&Int>(/storage/test)
                    signer.capabilities.publish(cap, at: /public/test)

                }
            }
        `

		address, err := common.HexToAddress("0x1")
		require.NoError(t, err)

		var events []cadence.Event

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
			OnGetSigningAccounts: func() ([]Address, error) {
				return []Address{
					address,
				}, nil
			},
			OnEmitEvent: func(event cadence.Event) error {
				events = append(events, event)
				return nil
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
            access(all) fun main(): &AnyStruct {
                return getAccount(0x1).capabilities.borrow<&AnyStruct>(/public/test)!
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
            access(all) fun main(): &AnyStruct {
                var acct = getAuthAccount<auth(Storage) &Account>(0x01)
	            var v:[AnyStruct] = []
	            acct.storage.save(v, to: /storage/x)

                var ref = acct.storage.borrow<auth(Insert) &[AnyStruct]>(from: /storage/x)!
	            ref.append(ref)
	            return ref
            }
        `

		rt := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
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
            access(all) fun main(): &AnyStruct {
                let acct = getAuthAccount<auth(Storage) &Account>(0x01)
	            let v: [AnyStruct] = []
	            acct.storage.save(v, to: /storage/x)

                let ref1 = acct.storage.borrow<auth(Insert) &[AnyStruct]>(from: /storage/x)!
                let ref2 = acct.storage.borrow<&[AnyStruct]>(from: /storage/x)!

	            ref1.append(ref2)
	            return ref1
            }
        `

		rt := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			Storage: NewTestLedger(nil, nil),
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

func TestRuntimeExportTypeValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(): Type {
                return Type<Int>()
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.TypeValue{
			StaticType: cadence.IntType,
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) struct S {}

            access(all) fun main(): Type {
                return Type<S>()
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.TypeValue{
			StaticType: cadence.NewStructType(
				common.ScriptLocation{},
				"S",
				[]cadence.Field{},
				nil,
			),
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("builtin struct", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(): Type {
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
		actual, err := ExportValue(
			value,
			NewTestInterpreter(t),
			interpreter.EmptyLocationRange,
		)
		require.NoError(t, err)

		expected := cadence.TypeValue{
			StaticType: nil,
		}

		assert.Equal(t, expected, actual)
	})

	t.Run("with intersection static type", func(t *testing.T) {

		t.Parallel()

		const code = `
          access(all) struct interface SI {}

          access(all) struct S: SI {}

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

		inter := NewTestInterpreter(t)
		inter.Program = interpreter.ProgramFromChecker(checker)

		ty := interpreter.TypeValue{
			Type: &interpreter.IntersectionStaticType{
				Types: []*interpreter.InterfaceStaticType{
					interpreter.NewInterfaceStaticTypeComputeTypeID(nil, TestLocation, "SI"),
				},
			},
		}

		actual, err := ExportValue(ty, inter, interpreter.EmptyLocationRange)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.TypeValue{
				StaticType: &cadence.IntersectionType{
					Types: []cadence.Type{
						cadence.NewStructInterfaceType(
							TestLocation,
							"SI",
							[]cadence.Field{},
							nil,
						),
					},
				},
			},
			actual,
		)
	})

}

func TestRuntimeExportCapabilityValue(t *testing.T) {

	t.Parallel()

	t.Run("Int", func(t *testing.T) {

		capability := interpreter.NewUnmeteredCapabilityValue(
			3,
			interpreter.AddressValue{0x1},
			interpreter.PrimitiveStaticTypeInt,
		)

		actual, err := ExportValue(
			capability,
			NewTestInterpreter(t),
			interpreter.EmptyLocationRange,
		)
		require.NoError(t, err)

		expected := cadence.NewCapability(
			3,
			cadence.Address{0x1},
			cadence.IntType,
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

		inter := NewTestInterpreter(t)
		inter.Program = interpreter.ProgramFromChecker(checker)

		capability := interpreter.NewUnmeteredCapabilityValue(
			3,
			interpreter.AddressValue{0x1},
			interpreter.NewCompositeStaticTypeComputeTypeID(inter, TestLocation, "S"),
		)

		actual, err := ExportValue(
			capability,
			inter,
			interpreter.EmptyLocationRange,
		)
		require.NoError(t, err)

		expected := cadence.NewCapability(
			3,
			cadence.Address{0x1},
			cadence.NewStructType(
				TestLocation,
				"S",
				[]cadence.Field{},
				nil,
			),
		)

		assert.Equal(t, expected, actual)
	})
}

func TestRuntimeExportCompositeValueWithFunctionValueField(t *testing.T) {

	t.Parallel()

	script := `
        access(all) struct Foo {
            access(all) let answer: Int
            access(all) let f: fun(): Void

            init() {
                self.answer = 42
                self.f = fun () {}
            }
        }

        access(all) fun main(): Foo {
            return Foo()
        }
    `

	fooStructType := cadence.NewStructType(
		common.ScriptLocation{},
		"Foo",
		[]cadence.Field{
			{
				Identifier: "answer",
				Type:       cadence.IntType,
			},
			{
				Identifier: "f",
				Type: &cadence.FunctionType{
					ReturnType: cadence.VoidType,
				},
			},
		},
		nil,
	)

	actual := exportValueFromScript(t, script)

	expected := cadence.NewStruct([]cadence.Value{
		cadence.NewInt(42),
		cadence.Function{
			FunctionType: &cadence.FunctionType{
				ReturnType: cadence.VoidType,
			},
		},
	}).WithType(fooStructType)

	assert.Equal(t, expected, actual)
}

//go:embed test-export-json-deterministic.json
var exportJsonDeterministicExpected string

func TestRuntimeExportJsonDeterministic(t *testing.T) {
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
		Type:       cadence.IntType,
	},
}
var fooResourceFields = []cadence.Field{
	{
		Identifier: "uuid",
		Type:       cadence.UInt64Type,
	},
	{
		Identifier: "bar",
		Type:       cadence.IntType,
	},
}

func newFooResourceType() *cadence.ResourceType {
	return cadence.NewResourceType(
		common.ScriptLocation{},
		"Foo",
		fooResourceFields,
		nil,
	)
}

func TestRuntimeEnumValue(t *testing.T) {

	t.Parallel()

	newEnumValue := func() cadence.Enum {
		return cadence.NewEnum([]cadence.Value{
			cadence.NewInt(3),
		}).WithType(cadence.NewEnumType(
			common.ScriptLocation{},
			"Direction",
			cadence.IntType,
			[]cadence.Field{
				{
					Identifier: sema.EnumRawValueFieldName,
					Type:       cadence.IntType,
				},
			},
			nil,
		))
	}

	t.Run("test export", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(): Direction {
                return Direction.RIGHT
            }

            access(all) enum Direction: Int {
                access(all) case UP
                access(all) case DOWN
                access(all) case LEFT
                access(all) case RIGHT
            }
        `

		expected := newEnumValue()
		actual := exportValueFromScript(t, script)

		assert.Equal(t, expected, actual)
	})

	t.Run("test import", func(t *testing.T) {
		t.Parallel()

		script := `
            access(all) fun main(dir: Direction): Direction {
                if !dir.isInstance(Type<Direction>()) {
                    panic("Not a Direction value")
                }

                return dir
            }

            access(all) enum Direction: Int {
                access(all) case UP
                access(all) case DOWN
                access(all) case LEFT
                access(all) case RIGHT
            }
        `

		expected := newEnumValue()
		actual, err := executeTestScript(t, script, expected)
		require.NoError(t, err)

		assert.Equal(t, expected, actual)
	})
}

func executeTestScript(t *testing.T, script string, arg cadence.Value) (cadence.Value, error) {
	rt := NewTestInterpreterRuntime()

	runtimeInterface := &TestRuntimeInterface{
		Storage: NewTestLedger(nil, nil),
		OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
			return json.Decode(nil, b)
		},
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
					ElementType: cadence.StringType,
				}),
		},
		{
			label:         "Array non-empty",
			typeSignature: "[String]",
			exportedValue: cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: cadence.StringType,
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
				KeyType:     cadence.StringType,
				ElementType: cadence.StringType,
			}),
		},
		{
			label:         "InclusiveRange",
			typeSignature: "InclusiveRange<UInt128>",
			exportedValue: cadence.NewInclusiveRange(
				cadence.NewUInt128(1),
				cadence.NewUInt128(500),
				cadence.NewUInt128(25),
			).WithType(&cadence.InclusiveRangeType{
				ElementType: cadence.UInt128Type,
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
				`access(all) fun main(arg: %[1]s)%[2]s {

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
				expected := test.exportedValue
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
	structType := cadence.NewStructType(
		common.ScriptLocation{},
		"Foo",
		[]cadence.Field{
			{
				Identifier: "a",
				Type: &cadence.OptionalType{
					Type: cadence.StringType,
				},
			},
			{
				Identifier: "b",
				Type: &cadence.DictionaryType{
					KeyType:     cadence.StringType,
					ElementType: cadence.StringType,
				},
			},
			{
				Identifier: "c",
				Type: &cadence.VariableSizedArrayType{
					ElementType: cadence.StringType,
				},
			},
			{
				Identifier: "d",
				Type: &cadence.ConstantSizedArrayType{
					ElementType: cadence.StringType,
					Size:        2,
				},
			},
			{
				Identifier: "e",
				Type:       cadence.AddressType,
			},
			{
				Identifier: "f",
				Type:       cadence.BoolType,
			},
			{
				Identifier: "g",
				Type:       cadence.StoragePathType,
			},
			{
				Identifier: "h",
				Type:       cadence.PublicPathType,
			},
			{
				Identifier: "i",
				Type:       cadence.PrivatePathType,
			},
			{
				Identifier: "j",
				Type:       cadence.AnyStructType,
			},
			{
				Identifier: "k",
				Type:       cadence.HashableStructType,
			},
		},
		nil,
	)

	complexStructValue := cadence.NewStruct([]cadence.Value{
		cadence.NewOptional(
			cadence.String("John"),
		),
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("name"),
				Value: cadence.String("Doe"),
			},
		}).WithType(&cadence.DictionaryType{
			KeyType:     cadence.StringType,
			ElementType: cadence.StringType,
		}),
		cadence.NewArray([]cadence.Value{
			cadence.String("foo"),
			cadence.String("bar"),
		}).WithType(&cadence.VariableSizedArrayType{
			ElementType: cadence.StringType,
		}),
		cadence.NewArray([]cadence.Value{
			cadence.String("foo"),
			cadence.String("bar"),
		}).WithType(&cadence.ConstantSizedArrayType{
			ElementType: cadence.StringType,
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
		cadence.String("foo"),
	}).WithType(structType)

	script := fmt.Sprintf(
		`
          access(all) fun main(arg: %[1]s): %[1]s {

              if !arg.isInstance(Type<%[1]s>()) {
                  panic("Not a %[1]s value")
              }

              return arg
          }

          access(all) struct Foo {
              access(all) var a: String?
              access(all) var b: {String: String}
              access(all) var c: [String]
              access(all) var d: [String; 2]
              access(all) var e: Address
              access(all) var f: Bool
              access(all) var g: StoragePath
              access(all) var h: PublicPath
              access(all) var i: PrivatePath
              access(all) var j: AnyStruct
              access(all) var k: HashableStruct

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
                  self.k = "hashable_struct_value"
              }
          }
        `,
		"Foo",
	)

	actual, err := executeTestScript(t, script, complexStructValue)
	require.NoError(t, err)

	expected := complexStructValue
	assert.Equal(t, expected, actual)

}

func TestRuntimeComplexStructWithAnyStructFields(t *testing.T) {

	t.Parallel()

	// Complex struct value
	structType := cadence.NewStructType(
		common.ScriptLocation{},
		"Foo",
		[]cadence.Field{
			{
				Identifier: "a",
				Type: &cadence.OptionalType{
					Type: cadence.AnyStructType,
				},
			},
			{
				Identifier: "b",
				Type: &cadence.DictionaryType{
					KeyType:     cadence.StringType,
					ElementType: cadence.AnyStructType,
				},
			},
			{
				Identifier: "c",
				Type: &cadence.VariableSizedArrayType{
					ElementType: cadence.AnyStructType,
				},
			},
			{
				Identifier: "d",
				Type: &cadence.ConstantSizedArrayType{
					ElementType: cadence.AnyStructType,
					Size:        2,
				},
			},
			{
				Identifier: "e",
				Type:       cadence.AnyStructType,
			},
		},
		nil,
	)

	complexStructValue := cadence.NewStruct([]cadence.Value{
		cadence.NewOptional(cadence.String("John")),
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("name"),
				Value: cadence.String("Doe"),
			},
		}).WithType(&cadence.DictionaryType{
			KeyType:     cadence.StringType,
			ElementType: cadence.AnyStructType,
		}),
		cadence.NewArray([]cadence.Value{
			cadence.String("foo"),
			cadence.String("bar"),
		}).WithType(&cadence.VariableSizedArrayType{
			ElementType: cadence.AnyStructType,
		}),
		cadence.NewArray([]cadence.Value{
			cadence.String("foo"),
			cadence.String("bar"),
		}).WithType(&cadence.ConstantSizedArrayType{
			ElementType: cadence.AnyStructType,
			Size:        2,
		}),
		cadence.Path{
			Domain:     common.PathDomainStorage,
			Identifier: "foo",
		},
	}).WithType(structType)

	script := fmt.Sprintf(
		`
          access(all) fun main(arg: %[1]s): %[1]s {

              if !arg.isInstance(Type<%[1]s>()) {
                  panic("Not a %[1]s value")
              }

              return arg
          }

          access(all) struct Foo {
              access(all) var a: AnyStruct?
              access(all) var b: {String: AnyStruct}
              access(all) var c: [AnyStruct]
              access(all) var d: [AnyStruct; 2]
              access(all) var e: AnyStruct

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

	assert.Equal(t, complexStructValue, actual)
}

func TestRuntimeComplexStructWithHashableStructFields(t *testing.T) {

	t.Parallel()

	// Complex struct value
	structType := cadence.NewStructType(
		common.ScriptLocation{},
		"Foo",
		[]cadence.Field{
			{
				Identifier: "a",
				Type: &cadence.OptionalType{
					Type: cadence.HashableStructType,
				},
			},
			{
				Identifier: "b",
				Type: &cadence.DictionaryType{
					KeyType:     cadence.StringType,
					ElementType: cadence.HashableStructType,
				},
			},
			{
				Identifier: "c",
				Type: &cadence.VariableSizedArrayType{
					ElementType: cadence.HashableStructType,
				},
			},
			{
				Identifier: "d",
				Type: &cadence.ConstantSizedArrayType{
					ElementType: cadence.HashableStructType,
					Size:        2,
				},
			},
			{
				Identifier: "e",
				Type:       cadence.HashableStructType,
			},
		},
		nil,
	)

	complexStructValue := cadence.NewStruct([]cadence.Value{
		cadence.NewOptional(cadence.String("John")),
		cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("name"),
				Value: cadence.String("Doe"),
			},
		}).WithType(&cadence.DictionaryType{
			KeyType:     cadence.StringType,
			ElementType: cadence.HashableStructType,
		}),
		cadence.NewArray([]cadence.Value{
			cadence.String("foo"),
			cadence.String("bar"),
		}).WithType(&cadence.VariableSizedArrayType{
			ElementType: cadence.HashableStructType,
		}),
		cadence.NewArray([]cadence.Value{
			cadence.String("foo"),
			cadence.String("bar"),
		}).WithType(&cadence.ConstantSizedArrayType{
			ElementType: cadence.HashableStructType,
			Size:        2,
		}),
		cadence.Path{
			Domain:     common.PathDomainStorage,
			Identifier: "foo",
		},
	}).WithType(structType)

	script := fmt.Sprintf(
		`
          access(all) fun main(arg: %[1]s): %[1]s {
              if !arg.isInstance(Type<%[1]s>()) {
                  panic("Not a %[1]s value")
              }
              return arg
          }

          access(all) struct Foo {
            access(all) var a: HashableStruct?
            access(all) var b: {String: HashableStruct}
            access(all) var c: [HashableStruct]
            access(all) var d: [HashableStruct; 2]
            access(all) var e: HashableStruct

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

	expected := complexStructValue
	assert.Equal(t, expected, actual)
}

func TestRuntimeMalformedArgumentPassing(t *testing.T) {

	t.Parallel()

	// Struct with wrong field type

	newMalformedStructType1 := func() *cadence.StructType {
		return cadence.NewStructType(
			common.ScriptLocation{},
			"Foo",
			[]cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.IntType,
				},
			},
			nil,
		)
	}

	newMalformedStruct1 := func() cadence.Struct {
		return cadence.NewStruct([]cadence.Value{
			cadence.NewInt(3),
		}).WithType(newMalformedStructType1())
	}

	// Struct with wrong field name

	newMalformedStruct2 := func() cadence.Struct {
		return cadence.NewStruct([]cadence.Value{
			cadence.String("John"),
		}).WithType(cadence.NewStructType(
			common.ScriptLocation{},
			"Foo",
			[]cadence.Field{
				{
					Identifier: "nonExisting",
					Type:       cadence.StringType,
				},
			},
			nil,
		))
	}

	// Struct with nested malformed array value
	newMalformedStruct3 := func() cadence.Struct {
		return cadence.NewStruct([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				newMalformedStruct1(),
			}),
		}).WithType(cadence.NewStructType(
			common.ScriptLocation{},
			"Bar",
			[]cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.VariableSizedArrayType{
						ElementType: newMalformedStructType1(),
					},
				},
			},
			nil,
		))
	}

	// Struct with nested malformed dictionary value
	newMalformedStruct4 := func() cadence.Struct {
		return cadence.NewStruct([]cadence.Value{
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: newMalformedStruct1(),
				},
			}),
		}).WithType(cadence.NewStructType(
			common.ScriptLocation{},
			"Baz",
			[]cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.DictionaryType{
						KeyType:     cadence.StringType,
						ElementType: newMalformedStructType1(),
					},
				},
			},
			nil,
		))
	}

	// Struct with nested array with mismatching element type
	newMalformedStruct5 := func() cadence.Struct {
		return cadence.NewStruct([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				cadence.String("mismatching value"),
			}),
		}).WithType(cadence.NewStructType(
			common.ScriptLocation{},
			"Bar",
			[]cadence.Field{
				{
					Identifier: "a",
					Type: &cadence.VariableSizedArrayType{
						ElementType: newMalformedStructType1(),
					},
				},
			},
			nil,
		))
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
			exportedValue:                            newMalformedStruct1(),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed Struct field name",
			typeSignature:                            "Foo",
			exportedValue:                            newMalformedStruct2(),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed AnyStruct",
			typeSignature:                            "AnyStruct",
			exportedValue:                            newMalformedStruct1(),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed nested struct array",
			typeSignature:                            "Bar",
			exportedValue:                            newMalformedStruct3(),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed nested struct dictionary",
			typeSignature:                            "Baz",
			exportedValue:                            newMalformedStruct4(),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Variable-size array with malformed element",
			typeSignature: "[Foo]",
			exportedValue: cadence.NewArray([]cadence.Value{
				newMalformedStruct1(),
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Constant-size array with malformed element",
			typeSignature: "[Foo; 1]",
			exportedValue: cadence.NewArray([]cadence.Value{
				newMalformedStruct1(),
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
			exportedValue:                            newMalformedStruct5(),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:                                    "Malformed Optional",
			typeSignature:                            "Foo?",
			exportedValue:                            cadence.NewOptional(newMalformedStruct1()),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Malformed dictionary",
			typeSignature: "{String: Foo}",
			exportedValue: cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: newMalformedStruct1(),
				},
			}),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
		{
			label:         "Malformed InclusiveRange",
			typeSignature: "InclusiveRange<Int>",
			exportedValue: cadence.NewInclusiveRange(
				cadence.NewUInt(1),
				cadence.NewUInt(10),
				cadence.NewUInt(3),
			),
			expectedInvalidEntryPointArgumentErrType: &MalformedValueError{},
		},
	}

	testArgumentPassing := func(test argumentPassingTest) {

		t.Run(test.label, func(t *testing.T) {

			t.Parallel()

			script := fmt.Sprintf(
				`access(all) fun main(arg: %[1]s): %[1]s {

                    if !arg.isInstance(Type<%[1]s>()) {
                        panic("Not a %[1]s value")
                    }

                    return arg
                }

                access(all) struct Foo {
                    access(all) var a: String

                    init() {
                        self.a = "Hello"
                    }
                }

                access(all) struct Bar {
                    access(all) var a: [Foo]

                    init() {
                        self.a = []
                    }
                }

                access(all) struct Baz {
                    access(all) var a: {String: Foo}

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

				var containerMutationError *interpreter.ContainerMutationError
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

		inter := NewTestInterpreter(t)

		value := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
		)

		actual, err := ExportValue(
			value,
			inter,
			interpreter.EmptyLocationRange,
		)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewArray([]cadence.Value{}).
				WithType(&cadence.VariableSizedArrayType{
					ElementType: cadence.AnyStructType,
				}),
			actual,
		)
	})

	t.Run("import empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewArray([]cadence.Value{})

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
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
				&interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeUInt8,
				},
				common.ZeroAddress,
			),
			actual,
		)
	})

	t.Run("export non-empty", func(t *testing.T) {

		t.Parallel()

		inter := NewTestInterpreter(t)

		value := interpreter.NewArrayValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			common.ZeroAddress,
			interpreter.NewUnmeteredIntValueFromInt64(42),
			interpreter.NewUnmeteredStringValue("foo"),
		)

		actual, err := ExportValue(
			value,
			inter,
			interpreter.EmptyLocationRange,
		)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewArray([]cadence.Value{
				cadence.NewInt(42),
				cadence.String("foo"),
			}).WithType(&cadence.VariableSizedArrayType{
				ElementType: cadence.AnyStructType,
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

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
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
				&interpreter.VariableSizedStaticType{
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

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
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
				&interpreter.VariableSizedStaticType{
					Type: &interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeInt8,
					},
				},
				common.ZeroAddress,
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.VariableSizedStaticType{
						Type: interpreter.PrimitiveStaticTypeInt8,
					},
					common.ZeroAddress,
					interpreter.NewUnmeteredInt8Value(4),
					interpreter.NewUnmeteredInt8Value(3),
				),
				interpreter.NewArrayValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.VariableSizedStaticType{
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
			NewTestInterpreter(t),
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
		)

		actual, err := ExportValue(
			value,
			NewTestInterpreter(t),
			interpreter.EmptyLocationRange,
		)
		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewDictionary([]cadence.KeyValuePair{}).
				WithType(&cadence.DictionaryType{
					KeyType:     cadence.StringType,
					ElementType: cadence.IntType,
				}),
			actual,
		)
	})

	t.Run("import empty", func(t *testing.T) {

		t.Parallel()

		value := cadence.NewDictionary([]cadence.KeyValuePair{})

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
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
				&interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeString,
					ValueType: interpreter.PrimitiveStaticTypeUInt8,
				},
			),
			actual,
		)
	})

	t.Run("export non-empty", func(t *testing.T) {

		t.Parallel()

		inter := NewTestInterpreter(t)

		value := interpreter.NewDictionaryValue(
			inter,
			interpreter.EmptyLocationRange,
			&interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeString,
				ValueType: interpreter.PrimitiveStaticTypeInt,
			},
			interpreter.NewUnmeteredStringValue("a"), interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredStringValue("b"), interpreter.NewUnmeteredIntValueFromInt64(2),
		)

		actual, err := ExportValue(
			value,
			inter,
			interpreter.EmptyLocationRange,
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
				KeyType:     cadence.StringType,
				ElementType: cadence.IntType,
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

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
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
				&interpreter.DictionaryStaticType{
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

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
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
				&interpreter.DictionaryStaticType{
					KeyType: interpreter.PrimitiveStaticTypeString,
					ValueType: &interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeSignedInteger,
						ValueType: interpreter.PrimitiveStaticTypeHashableStruct,
					},
				},

				interpreter.NewUnmeteredStringValue("a"),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeInt8,
						ValueType: interpreter.PrimitiveStaticTypeHashableStruct,
					},
					interpreter.NewUnmeteredInt8Value(1), interpreter.NewUnmeteredIntValueFromInt64(100),
					interpreter.NewUnmeteredInt8Value(2), interpreter.NewUnmeteredStringValue("hello"),
				),

				interpreter.NewUnmeteredStringValue("b"),
				interpreter.NewDictionaryValue(
					inter,
					interpreter.EmptyLocationRange,
					&interpreter.DictionaryStaticType{
						KeyType:   interpreter.PrimitiveStaticTypeSignedInteger,
						ValueType: interpreter.PrimitiveStaticTypeHashableStruct,
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

		dictionaryWithHeterogenousKeys := cadence.NewDictionary([]cadence.KeyValuePair{
			{
				Key:   cadence.String("foo"),
				Value: cadence.String("value1"),
			},
			{
				Key:   cadence.NewInt(5),
				Value: cadence.String("value2"),
			},
		})

		inter := NewTestInterpreter(t)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
			nil,
			dictionaryWithHeterogenousKeys,
			sema.AnyStructType,
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewDictionaryValue(
				inter,
				interpreter.EmptyLocationRange,
				&interpreter.DictionaryStaticType{
					KeyType:   interpreter.PrimitiveStaticTypeHashableStruct,
					ValueType: interpreter.PrimitiveStaticTypeString,
				},

				interpreter.NewUnmeteredStringValue("foo"),
				interpreter.NewUnmeteredStringValue("value1"),

				interpreter.NewIntValueFromInt64(nil, 5),
				interpreter.NewUnmeteredStringValue("value2"),
			),
			actual,
		)
	})

	t.Run("nested dictionary with mismatching element", func(t *testing.T) {
		t.Parallel()

		script :=
			`access(all) fun main(arg: {String: {String: String}}) {
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

		var argErr *interpreter.ContainerMutationError
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
            access(all) fun main(s: String) {
                log(s)
            }
        `

		encodedArg, err := json.Encode(stringValue)
		require.NoError(t, err)

		rt := NewTestInterpreterRuntime()

		var validated bool

		runtimeInterface := &TestRuntimeInterface{
			OnProgramLog: func(s string) {
				assert.True(t, utf8.ValidString(s))
				validated = true
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

func TestRuntimeTypeValueImport(t *testing.T) {

	t.Parallel()

	t.Run("Type<Int>", func(t *testing.T) {

		t.Parallel()

		typeValue := cadence.NewTypeValue(cadence.IntType)

		script := `
            access(all) fun main(s: Type) {
                log(s.identifier)
            }
        `

		encodedArg, err := json.Encode(typeValue)
		require.NoError(t, err)

		rt := NewTestInterpreterRuntime()

		var ok bool

		runtimeInterface := &TestRuntimeInterface{
			OnProgramLog: func(s string) {
				assert.Equal(t, "\"Int\"", s)
				ok = true
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

		typeValue := cadence.NewTypeValue(cadence.NewStructType(
			TestLocation,
			"S",
			[]cadence.Field{},
			[][]cadence.Parameter{},
		))

		script := `
            access(all) fun main(s: Type) {
            }
        `

		encodedArg, err := json.Encode(typeValue)
		require.NoError(t, err)

		rt := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

func TestRuntimeCapabilityValueImport(t *testing.T) {

	t.Parallel()

	t.Run("Capability<&Int>", func(t *testing.T) {

		t.Parallel()

		capabilityValue := cadence.NewCapability(
			42,
			cadence.Address{0x1},
			&cadence.ReferenceType{Type: cadence.IntType},
		)

		script := `
            access(all) fun main(s: Capability<&Int>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

		capabilityValue := cadence.NewCapability(
			3,
			cadence.Address{0x1},
			cadence.IntType,
		)

		script := `
            access(all) fun main(s: Capability<Int>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

		borrowType := cadence.NewStructType(
			TestLocation,
			"S",
			[]cadence.Field{},
			[][]cadence.Parameter{},
		)

		capabilityValue := cadence.NewCapability(
			42,
			cadence.Address{0x1},
			borrowType,
		)

		script := `
            access(all) fun main(s: Capability<S>) {
            }
        `

		encodedArg, err := json.Encode(capabilityValue)
		require.NoError(t, err)

		rt := NewTestInterpreterRuntime()

		runtimeInterface := &TestRuntimeInterface{
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

		rt := NewTestInterpreterRuntime()

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
                        access(all) fun main(key: PublicKey) {
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

					storage := NewTestLedger(nil, nil)

					runtimeInterface := &TestRuntimeInterface{
						Storage: storage,
						OnValidatePublicKey: func(publicKey *stdlib.PublicKey) error {
							publicKeyValidated = true
							return publicKeyActualError
						},
						OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
							return json.Decode(nil, b)
						},
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

						var publicKeyError *interpreter.InvalidPublicKeyError
						assert.ErrorAs(t, err, &publicKeyError)

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
            access(all) fun main(key: PublicKey): Bool {
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

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnVerifySignature: func(
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
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}
		addPublicKeyValidation(runtimeInterface, nil)

		actual, err := executeScript(t, script, publicKey, runtimeInterface)
		require.NoError(t, err)

		assert.True(t, verifyInvoked)
		assert.Equal(t, cadence.NewBool(true), actual)
	})

	t.Run("Invalid raw public key", func(t *testing.T) {
		script := `
            access(all) fun main(key: PublicKey) {
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

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid content in public key", func(t *testing.T) {
		script := `
            access(all) fun main(key: PublicKey) {
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

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid sign algo", func(t *testing.T) {
		script := `
            access(all) fun main(key: PublicKey) {
            }
        `

		publicKey := cadence.NewStruct(
			[]cadence.Value{
				publicKeyBytes,

				// Invalid value for 'signatureAlgorithm' field
				cadence.NewBool(true),
			},
		).WithType(PublicKeyType)

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Invalid sign algo fields", func(t *testing.T) {
		script := `
            access(all) fun main(key: PublicKey) {
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

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
		}

		_, err := executeScript(t, script, publicKey, runtimeInterface)
		RequireError(t, err)
		assertUserError(t, err)

		var argErr *InvalidEntryPointArgumentError
		require.ErrorAs(t, err, &argErr)
	})

	t.Run("Extra field", func(t *testing.T) {
		script := `
            access(all) fun main(key: PublicKey) {
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

		rt := NewTestInterpreterRuntime()

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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
            access(all) fun main(key: PublicKey): PublicKey {
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

		rt := NewTestInterpreterRuntime()

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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
            access(all) fun main(key: PublicKey): [UInt8] {
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

		rt := NewTestInterpreterRuntime()

		var publicKeyValidated bool

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnValidatePublicKey: func(publicKey *stdlib.PublicKey) error {
				publicKeyValidated = true
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

		RequireError(t, err)
		assert.Contains(t, err.Error(),
			"invalid argument at index 0: cannot import value of type 'PublicKey'. missing field 'publicKey'")
		assert.False(t, publicKeyValidated)
		assert.Nil(t, value)
	})

	t.Run("Missing signatureAlgorithm", func(t *testing.T) {
		script := `
            access(all) fun main(key: PublicKey): SignatureAlgorithm {
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

		rt := NewTestInterpreterRuntime()

		var publicKeyValidated bool

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnValidatePublicKey: func(publicKey *stdlib.PublicKey) error {
				publicKeyValidated = true
				return nil
			},
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

		RequireError(t, err)
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

	inter := NewTestInterpreter(t)
	inter.Program = &program

	// Array

	semaArrayType := &sema.VariableSizedType{
		Type: sema.AnyStructType,
	}

	staticArrayType := &interpreter.VariableSizedStaticType{
		Type: interpreter.PrimitiveStaticTypeAnyStruct,
	}

	externalArrayType := &cadence.VariableSizedArrayType{
		ElementType: cadence.AnyStructType,
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
		ElementType: cadence.AnyStructType,
	})

	// Dictionary

	semaDictionaryType := &sema.DictionaryType{
		KeyType:   sema.StringType,
		ValueType: semaArrayType,
	}

	staticDictionaryType := &interpreter.DictionaryStaticType{
		KeyType:   interpreter.PrimitiveStaticTypeString,
		ValueType: staticArrayType,
	}

	externalDictionaryType := &cadence.DictionaryType{
		KeyType:     cadence.StringType,
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
		KeyType: cadence.StringType,
		ElementType: &cadence.VariableSizedArrayType{
			ElementType: cadence.AnyStructType,
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

	externalCompositeType := cadence.NewStructType(
		TestLocation,
		"Foo",
		[]cadence.Field{
			{
				Identifier: "dictionary",
				Type:       externalDictionaryType,
			},
		},
		nil,
	)

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

	externalCompositeValue := cadence.NewStruct([]cadence.Value{
		externalDictionaryValue,
	}).WithType(externalCompositeType)

	t.Run("export", func(t *testing.T) {

		// NOTE: cannot be parallel, due to type's ID being cached (potential data race)

		actual, err := ExportValue(
			internalCompositeValue,
			inter,
			interpreter.EmptyLocationRange,
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

		inter := NewTestInterpreter(t)
		inter.Program = &program

		program.Elaboration.SetCompositeType(
			semaCompositeType.ID(),
			semaCompositeType,
		)

		actual, err := ImportValue(
			inter,
			interpreter.EmptyLocationRange,
			nil,
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
            access(all) fun main(arg: Foo) {
            }

            access(all) struct Foo {
                access(all) var a: AnyStruct

                init() {
                    self.a = nil
                }
            }
        `

		structValue := cadence.NewStruct([]cadence.Value{
			cadence.NewArray([]cadence.Value{
				cadence.String("foo"),
				cadence.String("bar"),
			}),
		}).WithType(cadence.NewStructType(
			common.ScriptLocation{},
			"Foo",
			[]cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.AnyStructType,
				},
			},
			nil,
		))

		_, err := executeTestScript(t, script, structValue)
		require.NoError(t, err)
	})

	t.Run("inner dictionary", func(t *testing.T) {
		script := `
            access(all) fun main(arg: Foo) {
            }

            access(all) struct Foo {
                access(all) var a: AnyStruct

                init() {
                    self.a = nil
                }
            }
        `

		structValue := cadence.NewStruct([]cadence.Value{
			cadence.NewDictionary([]cadence.KeyValuePair{
				{
					Key:   cadence.String("foo"),
					Value: cadence.String("bar"),
				},
			}),
		}).WithType(cadence.NewStructType(
			common.ScriptLocation{},
			"Foo",
			[]cadence.Field{
				{
					Identifier: "a",
					Type:       cadence.AnyStructType,
				},
			},
			nil,
		))

		_, err := executeTestScript(t, script, structValue)
		require.NoError(t, err)
	})
}

func TestRuntimeNestedStructArgPassing(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(v: AnyStruct): UInt8 {
                return (v as! Foo).bytes[0]
            }

            access(all) struct Foo {
                access(all) let bytes: [UInt8]

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

		rt := NewTestInterpreterRuntime()

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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
            access(all) fun main(v: AnyStruct) {
            }

            access(all) struct interface Foo {
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

		rt := NewTestInterpreterRuntime()

		storage := NewTestLedger(nil, nil)

		runtimeInterface := &TestRuntimeInterface{
			Storage: storage,
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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

func TestRuntimeDestroyedResourceReferenceExport(t *testing.T) {
	t.Parallel()

	rt := NewTestInterpreterRuntime()

	script := []byte(`
        access(all) resource S {}

        access(all) fun main(): &S  {
            var s <- create S()
            var ref = &s as &S

            // Just to trick the checker,
            // and get pass the static referenced resource invalidation analysis.
            var ref2 = getRef(ref)

            destroy s
            return ref2!
        }

        access(all) fun getRef(_ ref: &S): &S  {
            return ref
        }
	 `)

	runtimeInterface := &TestRuntimeInterface{}

	nextScriptLocation := NewScriptLocationGenerator()
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
	var invalidatedResourceReferenceError *interpreter.InvalidatedResourceReferenceError
		require.ErrorAs(t, err, &invalidatedResourceReferenceError)
}

func TestRuntimeDeploymentResultValueImportExport(t *testing.T) {

	t.Parallel()

	t.Run("import", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(v: DeploymentResult) {}
        `

		rt := NewTestInterpreterRuntime()
		runtimeInterface := &TestRuntimeInterface{}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)

		var notImportableError *ScriptParameterTypeNotImportableError
		require.ErrorAs(t, err, &notImportableError)
	})

	t.Run("export", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(): DeploymentResult? {
                return nil
            }
        `

		rt := NewTestInterpreterRuntime()
		runtimeInterface := &TestRuntimeInterface{}

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		RequireError(t, err)

		var invalidReturnTypeError *InvalidScriptReturnTypeError
		require.ErrorAs(t, err, &invalidReturnTypeError)
	})
}

func TestRuntimeDeploymentResultTypeImportExport(t *testing.T) {

	t.Parallel()

	t.Run("import", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(v: Type) {
                assert(v == Type<DeploymentResult>())
            }
        `

		rt := NewTestInterpreterRuntime()

		typeValue := cadence.NewTypeValue(cadence.NewStructType(
			nil,
			"DeploymentResult",
			[]cadence.Field{
				{
					Type:       cadence.NewOptionalType(cadence.DeployedContractType),
					Identifier: "deployedContract",
				},
			},
			nil,
		))

		encodedArg, err := json.Encode(typeValue)
		require.NoError(t, err)

		runtimeInterface := &TestRuntimeInterface{
			OnDecodeArgument: func(b []byte, t cadence.Type) (value cadence.Value, err error) {
				return json.Decode(nil, b)
			},
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
	})

	t.Run("export", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) fun main(): Type {
                return Type<DeploymentResult>()
            }
        `

		rt := NewTestInterpreterRuntime()
		runtimeInterface := &TestRuntimeInterface{}

		result, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: runtimeInterface,
				Location:  common.ScriptLocation{},
			},
		)

		require.NoError(t, err)

		assert.Equal(t,
			cadence.NewTypeValue(cadence.NewStructType(
				nil,
				"DeploymentResult",
				[]cadence.Field{
					{
						Type:       cadence.NewOptionalType(cadence.DeployedContractType),
						Identifier: "deployedContract",
					},
				},
				nil,
			)),
			result,
		)
	})
}

func TestRuntimeExportInterfaceType(t *testing.T) {

	t.Parallel()

	t.Run("exportable interface, exportable implementation", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) struct interface I {}

            access(all) struct S: I {}

            access(all) fun main(): {I} {
                return S()
            }
        `

		actual := exportValueFromScript(t, script)
		expected := cadence.NewStruct([]cadence.Value{}).
			WithType(cadence.NewStructType(
				common.ScriptLocation{},
				"S",
				[]cadence.Field{},
				nil,
			))

		assert.Equal(t, expected, actual)
	})

	t.Run("exportable interface, non exportable implementation", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) struct interface I {}

            access(all) struct S: I {
                access(self) var a: Block?
                init() {
                    self.a = nil
                }
            }

            access(all) fun main(): {I} {
                return S()
            }
        `

		rt := NewTestInterpreterRuntime()

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: &TestRuntimeInterface{},
				Location:  common.ScriptLocation{},
			},
		)

		// Dynamically validated
		notExportableError := &ValueNotExportableError{}
		require.ErrorAs(t, err, &notExportableError)
	})

	t.Run("non exportable interface, non exportable implementation", func(t *testing.T) {

		t.Parallel()

		script := `
            access(all) struct interface I {
                access(all) var a: Block?
            }

            access(all) struct S: I {
                access(all) var a: Block?
                init() {
                    self.a = nil
                }
            }

            access(all) fun main(): {I} {
                return S()
            }
        `

		rt := NewTestInterpreterRuntime()

		_, err := rt.ExecuteScript(
			Script{
				Source: []byte(script),
			},
			Context{
				Interface: &TestRuntimeInterface{},
				Location:  common.ScriptLocation{},
			},
		)

		// Statically validated
		invalidReturnType := &InvalidScriptReturnTypeError{}
		require.ErrorAs(t, err, &invalidReturnType)
	})
}
func TestRuntimeImportResolvedLocation(t *testing.T) {

	t.Parallel()

	addressLocation := common.AddressLocation{
		Address: common.MustBytesToAddress([]byte{42}),
		Name:    "Test",
	}

	identifierLocation := common.IdentifierLocation("Test")

	storage := NewUnmeteredInMemoryStorage()

	program := &interpreter.Program{
		Elaboration: sema.NewElaboration(nil),
	}

	inter, err := interpreter.NewInterpreter(
		program,
		addressLocation,
		&interpreter.Config{
			Storage:                       storage,
			AtreeValueValidationEnabled:   true,
			AtreeStorageValidationEnabled: true,
		},
	)
	require.NoError(t, err)

	semaCompositeType := &sema.CompositeType{
		Location:   addressLocation,
		Identifier: "Foo",
		Kind:       common.CompositeKindStructure,
		Members:    &sema.StringMemberOrderedMap{},
	}

	program.Elaboration.SetCompositeType(
		semaCompositeType.ID(),
		semaCompositeType,
	)

	externalCompositeType := cadence.NewStructType(
		identifierLocation,
		"Foo",
		[]cadence.Field{},
		nil,
	)

	externalCompositeValue := cadence.NewStruct(nil).
		WithType(externalCompositeType)

	resolveLocation := func(
		identifiers []ast.Identifier,
		location common.Location,
	) ([]sema.ResolvedLocation, error) {
		require.Equal(t, identifierLocation, location)

		location = addressLocation

		return []sema.ResolvedLocation{
			{
				Location:    location,
				Identifiers: identifiers,
			},
		}, nil
	}

	actual, err := ImportValue(
		inter,
		interpreter.EmptyLocationRange,
		nil,
		resolveLocation,
		externalCompositeValue,
		semaCompositeType,
	)
	require.NoError(t, err)

	internalCompositeValue := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		addressLocation,
		"Foo",
		common.CompositeKindStructure,
		nil,
		common.ZeroAddress,
	)

	AssertValuesEqual(
		t,
		inter,
		internalCompositeValue,
		actual,
	)
}

func TestExportNil(t *testing.T) {
	t.Parallel()

	inter := NewTestInterpreter(t)
	actual, err := ExportValue(
		nil,
		inter,
		interpreter.EmptyLocationRange,
	)
	require.NoError(t, err)
	assert.Nil(t, actual)
}
