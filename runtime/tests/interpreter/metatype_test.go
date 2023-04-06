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

package interpreter_test

import (
	"testing"

	"github.com/onflow/cadence/runtime/activations"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretMetaTypeEquality(t *testing.T) {

	t.Parallel()

	t.Run("Int == Int", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<Int>() == Type<Int>()
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			inter.Globals.Get("result").GetValue(),
		)
	})

	t.Run("Int != String", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<Int>() == Type<String>()
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			inter.Globals.Get("result").GetValue(),
		)
	})

	t.Run("Int != Int?", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<Int>() == Type<Int?>()
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			inter.Globals.Get("result").GetValue(),
		)
	})

	t.Run("&Int == &Int", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<&Int>() == Type<&Int>()
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			inter.Globals.Get("result").GetValue(),
		)
	})

	t.Run("&Int != &String", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
           let result = Type<&Int>() == Type<&String>()
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			inter.Globals.Get("result").GetValue(),
		)
	})

	t.Run("Int != unknownType", func(t *testing.T) {

		t.Parallel()

		valueDeclaration := stdlib.StandardLibraryValue{
			Name: "unknownType",
			Type: sema.MetaType,
			Value: interpreter.TypeValue{
				Type: nil,
			},
			Kind: common.DeclarationKindConstant,
		}

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(valueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, valueDeclaration)

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              let result = Type<Int>() == unknownType
            `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			inter.Globals.Get("result").GetValue(),
		)
	})

	t.Run("unknownType1 != unknownType2", func(t *testing.T) {

		t.Parallel()

		valueDeclarations := []stdlib.StandardLibraryValue{
			{
				Name: "unknownType1",
				Type: sema.MetaType,
				Value: interpreter.TypeValue{
					Type: nil,
				},
				Kind: common.DeclarationKindConstant,
			},
			{
				Name: "unknownType2",
				Type: sema.MetaType,
				Value: interpreter.TypeValue{
					Type: nil,
				},
				Kind: common.DeclarationKindConstant,
			},
		}

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		for _, valueDeclaration := range valueDeclarations {
			baseValueActivation.DeclareValue(valueDeclaration)
		}

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		for _, valueDeclaration := range valueDeclarations {
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              let result = unknownType1 == unknownType2
            `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			inter.Globals.Get("result").GetValue(),
		)
	})
}

func TestInterpretMetaTypeIdentifier(t *testing.T) {

	t.Parallel()

	t.Run("identifier, Int", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let type = Type<[Int]>()
          let identifier = type.identifier
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("[Int]"),
			inter.Globals.Get("identifier").GetValue(),
		)
	})

	t.Run("identifier, struct", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          struct S {}

          let type = Type<S>()
          let identifier = type.identifier
        `)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue("S.test.S"),
			inter.Globals.Get("identifier").GetValue(),
		)
	})

	t.Run("unknown", func(t *testing.T) {

		t.Parallel()

		valueDeclarations := []stdlib.StandardLibraryValue{
			{
				Name: "unknownType",
				Type: sema.MetaType,
				Value: interpreter.TypeValue{
					Type: nil,
				},
				Kind: common.DeclarationKindConstant,
			},
		}

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		for _, valueDeclaration := range valueDeclarations {
			baseValueActivation.DeclareValue(valueDeclaration)
		}

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		for _, valueDeclaration := range valueDeclarations {
			interpreter.Declare(baseActivation, valueDeclaration)
		}

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              let identifier = unknownType.identifier
            `,
			ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.NewUnmeteredStringValue(""),
			inter.Globals.Get("identifier").GetValue(),
		)
	})
}

func TestInterpretIsInstance(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name   string
		code   string
		result bool
	}{
		{
			name: "string is an instance of String",
			code: `
              let stringType = Type<String>()
              let result = "abc".isInstance(stringType)
            `,
			result: true,
		},
		{
			name: "int is an instance of Int",
			code: `
              let intType = Type<Int>()
              let result = (1).isInstance(intType)
            `,
			result: true,
		},
		{
			name: "resource is an instance of resource",
			code: `
              resource R {}

              let r <- create R()
              let rType = Type<@R>()
              let result = r.isInstance(rType)
            `,
			result: true,
		},
		{
			name: "int is not an instance of String",
			code: `
              let stringType = Type<String>()
              let result = (1).isInstance(stringType)
            `,
			result: false,
		},
		{
			name: "int is not an instance of resource",
			code: `
              resource R {}

              let rType = Type<@R>()
              let result = (1).isInstance(rType)
            `,
			result: false,
		},
		{
			name: "resource is not an instance of String",
			code: `
              resource R {}

              let r <- create R()
              let stringType = Type<String>()
              let result = r.isInstance(stringType)
            `,
			result: false,
		},
		{
			name: "resource R is not an instance of resource S",
			code: `
              resource R {}
              resource S {}

              let r <- create R()
              let sType = Type<@S>()
              let result = r.isInstance(sType)
            `,
			result: false,
		},
		{
			name: "struct S is not an instance of an unknown type",
			code: `
              struct S {}

              let s = S()
              let result = s.isInstance(unknownType)
            `,
			result: false,
		},
	}

	valueDeclaration := stdlib.StandardLibraryValue{
		Name: "unknownType",
		Type: sema.MetaType,
		Value: interpreter.TypeValue{
			Type: nil,
		},
		Kind: common.DeclarationKindConstant,
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(valueDeclaration)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, valueDeclaration)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			inter, err := parseCheckAndInterpretWithOptions(t, testCase.code, ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			})
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(testCase.result),
				inter.Globals.Get("result").GetValue(),
			)
		})
	}
}

func TestInterpretIsSubtype(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name   string
		code   string
		result bool
	}{
		{
			name: "String is a subtype of String",
			code: `
              let result = Type<String>().isSubtype(of: Type<String>())
            `,
			result: true,
		},
		{
			name: "Int is a subtype of Int",
			code: `
              let result = Type<Int>().isSubtype(of: Type<Int>())
            `,
			result: true,
		},
		{
			name: "Int is a subtype of Int?",
			code: `
              let result = Type<Int>().isSubtype(of: Type<Int?>())
            `,
			result: true,
		},
		{
			name: "Int? is a subtype of Int",
			code: `
              let result = Type<Int?>().isSubtype(of: Type<Int>())
            `,
			result: false,
		},
		{
			name: "resource is a subtype of AnyResource",
			code: `
              resource R {}
              let result = Type<@R>().isSubtype(of: Type<@AnyResource>())
            `,
			result: true,
		},
		{
			name: "struct is a subtype of AnyStruct",
			code: `
              struct S {}
              let result = Type<S>().isSubtype(of: Type<AnyStruct>())
            `,
			result: true,
		},
		{
			name: "Int is not a subtype of resource",
			code: `
              resource R {}
              let result = Type<Int>().isSubtype(of: Type<@R>())
            `,
			result: false,
		},
		{
			name: "resource is not a subtype of String",
			code: `
			  resource R {}
			  let result = Type<@R>().isSubtype(of: Type<String>())
            `,
			result: false,
		},
		{
			name: "resource R is not a subtype of resource S",
			code: `
              resource R {}
              resource S {}
              let result = Type<@R>().isSubtype(of: Type<@S>())
            `,
			result: false,
		},
		{
			name: "resource R is not a subtype of resource S",
			code: `
              resource R {}
              resource S {}
              let result = Type<@R>().isSubtype(of: Type<@S>())
            `,
			result: false,
		},
		{
			name: "Int is not a subtype of an unknown type",
			code: `
              let result = Type<Int>().isSubtype(of: unknownType)
            `,
			result: false,
		},
		{
			name: "unknown type is not a subtype of Int",
			code: `
              let result = unknownType.isSubtype(of: Type<Int>())
            `,
			result: false,
		},
	}

	valueDeclaration := stdlib.StandardLibraryValue{
		Name: "unknownType",
		Type: sema.MetaType,
		Value: interpreter.TypeValue{
			Type: nil,
		},
		Kind: common.DeclarationKindConstant,
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(valueDeclaration)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, valueDeclaration)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			inter, err := parseCheckAndInterpretWithOptions(t, testCase.code, ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			})
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.BoolValue(testCase.result),
				inter.Globals.Get("result").GetValue(),
			)
		})
	}
}

func TestInterpretGetType(t *testing.T) {

	t.Parallel()

	cases := []struct {
		name   string
		code   string
		result interpreter.Value
	}{
		{
			name: "String",
			code: `
              fun test(): Type {
                  return "abc".getType()
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.PrimitiveStaticTypeString,
			},
		},
		{
			name: "Int",
			code: `
              fun test(): Type {
                  return (1).getType()
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.PrimitiveStaticTypeInt,
			},
		},
		{
			name: "resource",
			code: `
              resource R {}

              fun test(): Type {
                  let r <- create R()
                  let res = r.getType()
                  destroy r
                  return res
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.NewCompositeStaticTypeComputeTypeID(nil, TestLocation, "R"),
			},
		},
		{
			// wrapping the ephemeral reference in an optional
			// ensures getType doesn't dereference the value,
			// i.e. EphemeralReferenceValue.StaticType is tested
			name: "optional ephemeral reference, auth to unauth",
			code: `
			  entitlement X
              fun test(): Type {
                  let value = 1
                  let ref = &value as auth(X) &Int
                  let optRef: &Int? = ref
                  return optRef.getType()
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.OptionalStaticType{
					Type: interpreter.ReferenceStaticType{
						// Reference was not converted
						Authorization:  interpreter.NewEntitlementSetAuthorization(nil, []common.TypeID{"S.test.X"}),
						ReferencedType: interpreter.PrimitiveStaticTypeInt,
					},
				},
			},
		},
		{
			// wrapping the ephemeral reference in an optional
			// ensures getType doesn't dereference the value,
			// i.e. EphemeralReferenceValue.StaticType is tested
			name: "optional ephemeral reference, auth to auth",
			code: `
			  entitlement X
              fun test(): Type {
                  let value = 1
                  let ref = &value as auth(X) &Int
                  let optRef: auth(X) &Int? = ref
                  return optRef.getType()
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.OptionalStaticType{
					Type: interpreter.ReferenceStaticType{
						Authorization:  interpreter.NewEntitlementSetAuthorization(nil, []common.TypeID{"S.test.X"}),
						ReferencedType: interpreter.PrimitiveStaticTypeInt,
					},
				},
			},
		},
		{
			// wrapping the storage reference in an optional
			// ensures getType doesn't dereference the value,
			// i.e. StorageReferenceValue.StaticType is tested
			name: "optional storage reference, auth to unauth",
			code: `
			  entitlement X
			  fun getStorageReference(): auth(X) &Int {
				account.save(1, to: /storage/foo)
				return account.borrow<auth(X) &Int>(from: /storage/foo)!
			}
              fun test(): Type {
                  let ref = getStorageReference()
                  let optRef: &Int? = ref
                  return optRef.getType()
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.OptionalStaticType{
					Type: interpreter.ReferenceStaticType{
						// Reference was not converted
						Authorization:  interpreter.NewEntitlementSetAuthorization(nil, []common.TypeID{"S.test.X"}),
						ReferencedType: interpreter.PrimitiveStaticTypeInt,
					},
				},
			},
		},
		{
			// wrapping the storage reference in an optional
			// ensures getType doesn't dereference the value,
			// i.e. StorageReferenceValue.StaticType is tested
			name: "optional storage reference, auth to auth",
			code: `
			  entitlement X
			  fun getStorageReference(): auth(X) &Int {
			      account.save(1, to: /storage/foo)
				  return account.borrow<auth(X) &Int>(from: /storage/foo)!
			  }
              fun test(): Type {
                  let ref = getStorageReference()
                  let optRef: auth(X) &Int? = ref
                  return optRef.getType()
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.OptionalStaticType{
					Type: interpreter.ReferenceStaticType{
						Authorization:  interpreter.NewEntitlementSetAuthorization(nil, []common.TypeID{"S.test.X"}),
						ReferencedType: interpreter.PrimitiveStaticTypeInt,
					},
				},
			},
		},
		{
			name: "array",
			code: `
              fun test(): Type {
                  return [1, 3].getType()
              }
            `,
			result: interpreter.TypeValue{
				Type: interpreter.VariableSizedStaticType{
					Type: interpreter.PrimitiveStaticTypeInt,
				},
			},
		},
	}

	for _, testCase := range cases {
		address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})
		t.Run(testCase.name, func(t *testing.T) {
			inter, _ := testAccount(t,
				address,
				true,
				testCase.code,
				sema.Config{},
			)

			result, err := inter.Invoke("test")
			require.NoError(t, err)

			AssertValuesEqual(
				t,
				inter,
				testCase.result,
				result,
			)
		})
	}
}
