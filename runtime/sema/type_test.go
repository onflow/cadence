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

package sema

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/parser"
)

func TestConstantSizedType_String(t *testing.T) {

	t.Parallel()

	ty := &ConstantSizedType{
		Type: &VariableSizedType{Type: IntType},
		Size: 2,
	}

	assert.Equal(t,
		"[[Int]; 2]",
		ty.String(),
	)
}

func TestConstantSizedType_String_OfFunctionType(t *testing.T) {

	t.Parallel()

	ty := &ConstantSizedType{
		Type: &FunctionType{
			Parameters: []*Parameter{
				{
					TypeAnnotation: NewTypeAnnotation(Int8Type),
				},
			},
			ReturnTypeAnnotation: NewTypeAnnotation(
				Int16Type,
			),
		},
		Size: 2,
	}

	assert.Equal(t,
		"[((Int8): Int16); 2]",
		ty.String(),
	)
}

func TestVariableSizedType_String(t *testing.T) {

	t.Parallel()

	ty := &VariableSizedType{
		Type: &ConstantSizedType{
			Type: IntType,
			Size: 2,
		},
	}

	assert.Equal(t,
		"[[Int; 2]]",
		ty.String(),
	)
}

func TestVariableSizedType_String_OfFunctionType(t *testing.T) {

	t.Parallel()

	ty := &VariableSizedType{
		Type: &FunctionType{
			Parameters: []*Parameter{
				{
					TypeAnnotation: NewTypeAnnotation(Int8Type),
				},
			},
			ReturnTypeAnnotation: NewTypeAnnotation(
				Int16Type,
			),
		},
	}

	assert.Equal(t,
		"[((Int8): Int16)]",
		ty.String(),
	)
}

func TestIsResourceType_AnyStructNestedInArray(t *testing.T) {

	t.Parallel()

	ty := &VariableSizedType{
		Type: AnyStructType,
	}

	assert.False(t, ty.IsResourceType())
}

func TestIsResourceType_AnyResourceNestedInArray(t *testing.T) {

	t.Parallel()

	ty := &VariableSizedType{
		Type: AnyResourceType,
	}

	assert.True(t, ty.IsResourceType())
}

func TestIsResourceType_ResourceNestedInArray(t *testing.T) {

	t.Parallel()

	ty := &VariableSizedType{
		Type: &CompositeType{
			Kind: common.CompositeKindResource,
		},
	}

	assert.True(t, ty.IsResourceType())
}

func TestIsResourceType_ResourceNestedInDictionary(t *testing.T) {

	t.Parallel()

	ty := &DictionaryType{
		KeyType: StringType,
		ValueType: &VariableSizedType{
			Type: &CompositeType{
				Kind: common.CompositeKindResource,
			},
		},
	}

	assert.True(t, ty.IsResourceType())
}

func TestIsResourceType_StructNestedInDictionary(t *testing.T) {

	t.Parallel()

	ty := &DictionaryType{
		KeyType: StringType,
		ValueType: &VariableSizedType{
			Type: &CompositeType{
				Kind: common.CompositeKindStructure,
			},
		},
	}

	assert.False(t, ty.IsResourceType())
}

func TestRestrictedType_StringAndID(t *testing.T) {

	t.Parallel()

	t.Run("base type and restriction", func(t *testing.T) {

		t.Parallel()

		interfaceType := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I",
			Location:      common.StringLocation("b"),
		}

		ty := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{interfaceType},
		}

		assert.Equal(t,
			"R{I}",
			ty.String(),
		)

		assert.Equal(t,
			TypeID("S.a.R{S.b.I}"),
			ty.ID(),
		)
	})

	t.Run("base type and restrictions", func(t *testing.T) {

		t.Parallel()

		i1 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I1",
			Location:      common.StringLocation("b"),
		}

		i2 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I2",
			Location:      common.StringLocation("c"),
		}

		ty := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1, i2},
		}

		assert.Equal(t,
			ty.String(),
			"R{I1, I2}",
		)

		assert.Equal(t,
			TypeID("S.a.R{S.b.I1,S.c.I2}"),
			ty.ID(),
		)
	})

	t.Run("no restrictions", func(t *testing.T) {

		t.Parallel()

		ty := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
		}

		assert.Equal(t,
			"R{}",
			ty.String(),
		)

		assert.Equal(t,
			TypeID("S.a.R{}"),
			ty.ID(),
		)
	})
}

func TestRestrictedType_Equals(t *testing.T) {

	t.Parallel()

	t.Run("same base type and more restrictions", func(t *testing.T) {

		t.Parallel()

		i1 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I1",
			Location:      common.StringLocation("b"),
		}

		i2 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I2",
			Location:      common.StringLocation("b"),
		}

		a := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1},
		}

		b := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1, i2},
		}

		assert.False(t, a.Equal(b))
	})

	t.Run("same base type and fewer restrictions", func(t *testing.T) {

		t.Parallel()

		i1 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I1",
			Location:      common.StringLocation("b"),
		}

		i2 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I2",
			Location:      common.StringLocation("b"),
		}

		a := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1, i2},
		}

		b := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1},
		}

		assert.False(t, a.Equal(b))
	})

	t.Run("same base type and same restrictions", func(t *testing.T) {

		t.Parallel()

		i1 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I1",
			Location:      common.StringLocation("b"),
		}

		i2 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I2",
			Location:      common.StringLocation("b"),
		}

		a := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1, i2},
		}

		b := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1, i2},
		}

		assert.True(t, a.Equal(b))
	})

	t.Run("different base type and same restrictions", func(t *testing.T) {

		t.Parallel()

		i1 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I1",
			Location:      common.StringLocation("b"),
		}

		i2 := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I2",
			Location:      common.StringLocation("b"),
		}

		a := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R1",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1, i2},
		}

		b := &RestrictedType{
			Type: &CompositeType{
				Kind:       common.CompositeKindResource,
				Identifier: "R2",
				Location:   common.StringLocation("a"),
			},
			Restrictions: []*InterfaceType{i1, i2},
		}

		assert.False(t, a.Equal(b))
	})
}

func TestRestrictedType_GetMember(t *testing.T) {

	t.Parallel()

	t.Run("forbid undeclared members", func(t *testing.T) {

		t.Parallel()

		resourceType := &CompositeType{
			Kind:       common.CompositeKindResource,
			Identifier: "R",
			Location:   common.StringLocation("a"),
			Fields:     []string{},
			Members:    NewStringMemberOrderedMap(),
		}
		ty := &RestrictedType{
			Type:         resourceType,
			Restrictions: []*InterfaceType{},
		}

		fieldName := "s"
		resourceType.Members.Set(fieldName, NewUnmeteredPublicConstantFieldMember(
			ty.Type,
			fieldName,
			IntType,
			"",
		))

		actualMembers := ty.GetMembers()

		require.Contains(t, actualMembers, fieldName)

		var reportedError error
		actualMember := actualMembers[fieldName].Resolve(
			nil,
			fieldName,
			ast.Range{},
			func(err error) {
				reportedError = err
			},
		)

		assert.IsType(t, &InvalidRestrictedTypeMemberAccessError{}, reportedError)
		assert.NotNil(t, actualMember)
	})

	t.Run("allow declared members", func(t *testing.T) {

		t.Parallel()

		interfaceType := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I",
			Members:       NewStringMemberOrderedMap(),
		}

		resourceType := &CompositeType{
			Kind:       common.CompositeKindResource,
			Identifier: "R",
			Location:   common.StringLocation("a"),
			Fields:     []string{},
			Members:    NewStringMemberOrderedMap(),
		}
		restrictedType := &RestrictedType{
			Type: resourceType,
			Restrictions: []*InterfaceType{
				interfaceType,
			},
		}

		fieldName := "s"

		resourceType.Members.Set(fieldName, NewUnmeteredPublicConstantFieldMember(
			restrictedType.Type,
			fieldName,
			IntType,
			"",
		))

		interfaceMember := NewUnmeteredPublicConstantFieldMember(
			restrictedType.Type,
			fieldName,
			IntType,
			"",
		)
		interfaceType.Members.Set(fieldName, interfaceMember)

		actualMembers := restrictedType.GetMembers()

		require.Contains(t, actualMembers, fieldName)

		actualMember := actualMembers[fieldName].Resolve(nil, fieldName, ast.Range{}, nil)

		assert.Same(t, interfaceMember, actualMember)
	})
}

func TestBeforeType_Strings(t *testing.T) {

	t.Parallel()

	expected := "(<T: AnyStruct>(_ value: T): T)"

	assert.Equal(t,
		expected,
		beforeType.String(),
	)

	assert.Equal(t,
		expected,
		beforeType.QualifiedString(),
	)
}

func TestQualifiedIdentifierCreation(t *testing.T) {

	t.Run("with containers", func(t *testing.T) {

		a := &CompositeType{
			Kind:       common.CompositeKindStructure,
			Identifier: "A",
			Location:   common.StringLocation("a"),
			Fields:     []string{},
			Members:    NewStringMemberOrderedMap(),
		}

		b := &CompositeType{
			Kind:          common.CompositeKindStructure,
			Identifier:    "B",
			Location:      common.StringLocation("a"),
			Fields:        []string{},
			Members:       NewStringMemberOrderedMap(),
			containerType: a,
		}

		c := &CompositeType{
			Kind:          common.CompositeKindStructure,
			Identifier:    "C",
			Location:      common.StringLocation("a"),
			Fields:        []string{},
			Members:       NewStringMemberOrderedMap(),
			containerType: b,
		}

		identifier := qualifiedIdentifier("foo", c)
		assert.Equal(t, "A.B.C.foo", identifier)
	})

	t.Run("without containers", func(t *testing.T) {
		identifier := qualifiedIdentifier("foo", nil)
		assert.Equal(t, "foo", identifier)
	})

	t.Run("public account container", func(t *testing.T) {
		identifier := qualifiedIdentifier("foo", PublicAccountType)
		assert.Equal(t, "PublicAccount.foo", identifier)
	})
}

func BenchmarkQualifiedIdentifierCreation(b *testing.B) {

	foo := &CompositeType{
		Kind:       common.CompositeKindStructure,
		Identifier: "foo",
		Location:   common.StringLocation("a"),
		Fields:     []string{},
		Members:    NewStringMemberOrderedMap(),
	}

	bar := &CompositeType{
		Kind:          common.CompositeKindStructure,
		Identifier:    "bar",
		Location:      common.StringLocation("a"),
		Fields:        []string{},
		Members:       NewStringMemberOrderedMap(),
		containerType: foo,
	}

	b.Run("One level", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			qualifiedIdentifier("baz", nil)
		}
	})

	b.Run("Three levels", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			qualifiedIdentifier("baz", bar)
		}
	})
}

func TestIdentifierCacheUpdate(t *testing.T) {

	code := `
          pub contract interface Test {

              pub struct interface NestedInterface {
                  pub fun test(): Bool
              }

              pub struct Nested: NestedInterface {}
          }

          pub contract TestImpl {

              pub struct Nested {
                  pub fun test(): Bool {
                      return true
                  }
              }
          }
	`

	program, err := parser.ParseProgram(code, nil)
	require.NoError(t, err)

	checker, err := NewChecker(
		program,
		common.StringLocation("test"),
		nil,
		false,
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	checker.typeActivations.ForEachVariableDeclaredInAndBelow(
		0,
		func(_ string, value *Variable) {
			typ := value.Type

			var checkIdentifiers func(t *testing.T, typ Type)

			checkNestedTypes := func(nestedTypes *StringTypeOrderedMap) {
				if nestedTypes != nil {
					nestedTypes.Foreach(
						func(_ string, typ Type) {
							checkIdentifiers(t, typ)
						},
					)
				}
			}

			checkIdentifiers = func(t *testing.T, typ Type) {
				switch semaType := typ.(type) {
				case *CompositeType:
					cachedQualifiedID := semaType.QualifiedIdentifier()
					cachedID := semaType.ID()

					// clear cached identifiers for one level
					semaType.cachedIdentifiers = nil

					recalculatedQualifiedID := semaType.QualifiedIdentifier()
					recalculatedID := semaType.ID()

					assert.Equal(t, recalculatedQualifiedID, cachedQualifiedID)
					assert.Equal(t, recalculatedID, cachedID)

					// Recursively check for nested types
					checkNestedTypes(semaType.nestedTypes)

				case *InterfaceType:
					cachedQualifiedID := semaType.QualifiedIdentifier()
					cachedID := semaType.ID()

					// clear cached identifiers for one level
					semaType.cachedIdentifiers = nil

					recalculatedQualifiedID := semaType.QualifiedIdentifier()
					recalculatedID := semaType.ID()

					assert.Equal(t, recalculatedQualifiedID, cachedQualifiedID)
					assert.Equal(t, recalculatedID, cachedID)

					// Recursively check for nested types
					checkNestedTypes(semaType.nestedTypes)
				}
			}

			checkIdentifiers(t, typ)
		})
}

func TestCommonSuperType(t *testing.T) {
	t.Parallel()

	t.Run("Duplicate Mask", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				err, _ := r.(error)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "duplicate type tag: {32 0}")
			}
		}()

		_ = newTypeTagFromLowerMask(32)
	})

	nilType := &OptionalType{NeverType}

	resourceType := &CompositeType{
		Location:   nil,
		Identifier: "Foo",
		Kind:       common.CompositeKindResource,
	}

	type testCase struct {
		name              string
		types             []Type
		expectedSuperType Type
	}

	testLeastCommonSuperType := func(t *testing.T, tests []testCase) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(
					t,
					test.expectedSuperType,
					LeastCommonSuperType(test.types...),
				)
			})
		}
	}

	t.Run("All types", func(t *testing.T) {
		t.Parallel()

		// super type of similar types should be the type itself.
		// i.e: super type of collection of T's should be T.
		// Make sure it's true for all known types.

		tests := make([]testCase, 0)

		err := BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
			typ := variable.Type
			tests = append(tests, testCase{
				name: name,
				types: []Type{
					typ,
					typ,
				},
				expectedSuperType: typ,
			})

			return nil
		})

		require.NoError(t, err)
		testLeastCommonSuperType(t, tests)
	})

	t.Run("Simple types", func(t *testing.T) {
		t.Parallel()

		tests := []testCase{
			{
				name: "homogenous integer types",
				types: []Type{
					UInt8Type,
					UInt8Type,
					UInt8Type,
				},
				expectedSuperType: UInt8Type,
			},
			{
				name: "heterogeneous integer types",
				types: []Type{
					UInt8Type,
					UInt16Type,
					UInt256Type,
					IntegerType,
					Word64Type,
				},
				expectedSuperType: IntegerType,
			},
			{
				name: "heterogeneous fixed-point types",
				types: []Type{
					Fix64Type,
					UFix64Type,
					FixedPointType,
				},
				expectedSuperType: FixedPointType,
			},
			{
				name: "heterogeneous numeric types",
				types: []Type{
					Int8Type,
					UInt16Type,
					IntegerType,
					Word64Type,
					Fix64Type,
					UFix64Type,
					FixedPointType,
				},
				expectedSuperType: NumberType,
			},
			{
				name: "signed numbers",
				types: []Type{
					Int8Type,
					Int128Type,
					Fix64Type,
				},
				expectedSuperType: SignedNumberType,
			},
			{
				name: "signed integers",
				types: []Type{
					Int8Type,
					Int128Type,
				},
				expectedSuperType: SignedIntegerType,
			},
			{
				name: "unsigned numbers",
				types: []Type{
					UInt8Type,
					UInt128Type,
					UFix64Type,
				},
				expectedSuperType: NumberType,
			},
			{
				name: "unsigned integers",
				types: []Type{
					UInt8Type,
					UInt128Type,
				},
				expectedSuperType: IntegerType,
			},
			{
				name: "heterogeneous simple types",
				types: []Type{
					StringType,
					Int8Type,
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "all nil",
				types: []Type{
					nilType,
					nilType,
					nilType,
				},
				expectedSuperType: nilType,
			},
			{
				name: "never type",
				types: []Type{
					NeverType,
					NeverType,
				},
				expectedSuperType: NeverType,
			},
			{
				name: "never with numerics",
				types: []Type{
					IntType,
					Int8Type,
					NeverType,
				},
				expectedSuperType: SignedIntegerType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Structs & Resources", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		interfaceType1 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I1",
			CompositeKind: common.CompositeKindStructure,
			Members:       NewStringMemberOrderedMap(),
		}

		interfaceType2 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I2",
			CompositeKind: common.CompositeKindStructure,
			Members:       NewStringMemberOrderedMap(),
		}

		interfaceType3 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I3",
			CompositeKind: common.CompositeKindStructure,
			Members:       NewStringMemberOrderedMap(),
		}

		newCompositeWithInterfaces := func(name string, interfaces ...*InterfaceType) *CompositeType {
			return &CompositeType{
				Location:                      testLocation,
				Identifier:                    name,
				Kind:                          common.CompositeKindStructure,
				ExplicitInterfaceConformances: interfaces,
				Members:                       NewStringMemberOrderedMap(),
			}
		}

		tests := []testCase{
			{
				name: "all anyStructs",
				types: []Type{
					AnyStructType,
					AnyStructType,
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "all anyResources",
				types: []Type{
					AnyResourceType,
					AnyResourceType,
				},
				expectedSuperType: AnyResourceType,
			},
			{
				name: "structs and resources",
				types: []Type{
					AnyResourceType,
					AnyStructType,
				},
				expectedSuperType: InvalidType,
			},
			{
				name: "all structs",
				types: []Type{
					PublicKeyType,
					PublicKeyType,
				},
				expectedSuperType: PublicKeyType,
			},
			{
				name: "mixed type structs",
				types: []Type{
					PublicKeyType,
					AuthAccountType,
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "struct and anyStruct",
				types: []Type{
					AnyStructType,
					PublicKeyType,
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "common interface",
				types: []Type{
					newCompositeWithInterfaces("Foo", interfaceType1, interfaceType2),
					newCompositeWithInterfaces("Bar", interfaceType2, interfaceType3),
					newCompositeWithInterfaces("Baz", interfaceType1, interfaceType2, interfaceType3),
				},
				expectedSuperType: &RestrictedType{
					Type:         AnyStructType,
					Restrictions: []*InterfaceType{interfaceType2},
				},
			},
			{
				name: "multiple common interfaces",
				types: []Type{
					newCompositeWithInterfaces("Foo", interfaceType1, interfaceType2),
					newCompositeWithInterfaces("Baz", interfaceType1, interfaceType2, interfaceType3),
				},
				expectedSuperType: &RestrictedType{
					Type:         AnyStructType,
					Restrictions: []*InterfaceType{interfaceType1, interfaceType2},
				},
			},
			{
				name: "no common interfaces",
				types: []Type{
					newCompositeWithInterfaces("Foo", interfaceType1),
					newCompositeWithInterfaces("Baz", interfaceType2),
					newCompositeWithInterfaces("Baz", interfaceType3),
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "structs with never",
				types: []Type{
					NeverType,
					PublicKeyType,
					PublicKeyType,
					NeverType,
					PublicKeyType,
				},
				expectedSuperType: PublicKeyType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Arrays", func(t *testing.T) {
		t.Parallel()

		stringArray := &VariableSizedType{
			Type: StringType,
		}

		resourceArray := &VariableSizedType{
			Type: resourceType,
		}

		nestedResourceArray := &VariableSizedType{
			Type: resourceArray,
		}

		tests := []testCase{
			{
				name: "homogeneous arrays",
				types: []Type{
					stringArray,
					stringArray,
				},
				expectedSuperType: stringArray,
			},
			{
				name: "var-sized & constant-sized",
				types: []Type{
					stringArray,
					&ConstantSizedType{Type: StringType, Size: 2},
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "heterogeneous arrays",
				types: []Type{
					stringArray,
					&VariableSizedType{Type: BoolType},
				},
				expectedSuperType: &VariableSizedType{Type: AnyStructType},
			},
			{
				name: "simple-typed array & resource array",
				types: []Type{
					stringArray,
					resourceArray,
				},
				expectedSuperType: InvalidType,
			},
			{
				name: "array & non-array",
				types: []Type{
					stringArray,
					StringType,
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "resource array",
				types: []Type{
					resourceArray,
					resourceArray,
				},
				expectedSuperType: resourceArray,
			},
			{
				name: "resource array & resource",
				types: []Type{
					resourceArray,
					resourceType,
				},
				expectedSuperType: AnyResourceType,
			},
			{
				name: "nested resource arrays",
				types: []Type{
					nestedResourceArray,
					nestedResourceArray,
				},
				expectedSuperType: nestedResourceArray,
			},
			{
				name: "nested resource-array & struct-array",
				types: []Type{
					nestedResourceArray,
					&VariableSizedType{Type: stringArray},
				},
				expectedSuperType: InvalidType,
			},
			{
				name: "covariant arrays",
				types: []Type{
					&VariableSizedType{
						Type: IntType,
					},
					&VariableSizedType{
						Type: Int8Type,
					},
				},
				expectedSuperType: &VariableSizedType{
					Type: SignedIntegerType,
				},
			},
			{
				name: "arrays with never",
				types: []Type{
					NeverType,
					stringArray,
					NeverType,
					stringArray,
				},
				expectedSuperType: stringArray,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Dictionaries", func(t *testing.T) {
		t.Parallel()

		stringStringDictionary := &DictionaryType{
			KeyType:   StringType,
			ValueType: StringType,
		}

		stringBoolDictionary := &DictionaryType{
			KeyType:   StringType,
			ValueType: BoolType,
		}

		stringResourceDictionary := &DictionaryType{
			KeyType:   StringType,
			ValueType: resourceType,
		}

		nestedResourceDictionary := &DictionaryType{
			KeyType:   StringType,
			ValueType: stringResourceDictionary,
		}

		nestedStringDictionary := &DictionaryType{
			KeyType:   StringType,
			ValueType: stringStringDictionary,
		}

		tests := []testCase{
			{
				name: "homogeneous dictionaries",
				types: []Type{
					stringStringDictionary,
					stringStringDictionary,
				},
				expectedSuperType: stringStringDictionary,
			},
			{
				name: "heterogeneous dictionaries",
				types: []Type{
					stringStringDictionary,
					stringBoolDictionary,
				},
				expectedSuperType: &DictionaryType{
					KeyType:   StringType,
					ValueType: AnyStructType,
				},
			},
			{
				name: "dictionary & non-dictionary",
				types: []Type{
					stringStringDictionary,
					StringType,
				},
				expectedSuperType: AnyStructType,
			},

			{
				name: "struct dictionary & resource dictionary",
				types: []Type{
					stringStringDictionary,
					stringResourceDictionary,
				},
				expectedSuperType: InvalidType,
			},

			{
				name: "resource dictionaries",
				types: []Type{
					stringResourceDictionary,
					stringResourceDictionary,
				},
				expectedSuperType: stringResourceDictionary,
			},
			{
				name: "resource dictionary & resource",
				types: []Type{
					stringResourceDictionary,
					resourceType,
				},
				expectedSuperType: AnyResourceType,
			},
			{
				name: "nested resource dictionaries",
				types: []Type{
					nestedResourceDictionary,
					nestedResourceDictionary,
				},
				expectedSuperType: nestedResourceDictionary,
			},
			{
				name: "nested resource-dictionary & nested struct-dictionary",
				types: []Type{
					nestedResourceDictionary,
					nestedStringDictionary,
				},
				expectedSuperType: InvalidType,
			},
			{
				name: "dictionaries with never",
				types: []Type{
					NeverType,
					stringStringDictionary,
					NeverType,
					stringStringDictionary,
				},
				expectedSuperType: stringStringDictionary,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("References types", func(t *testing.T) {
		t.Parallel()

		tests := []testCase{
			{
				name: "homogenous references",
				types: []Type{
					&ReferenceType{
						Type: Int8Type,
					},
					&ReferenceType{
						Type: Int8Type,
					},
					&ReferenceType{
						Type: Int8Type,
					},
				},
				expectedSuperType: &ReferenceType{
					Type: Int8Type,
				},
			},
			{
				name: "heterogeneous references",
				types: []Type{
					&ReferenceType{
						Type: Int8Type,
					},
					&ReferenceType{
						Type: StringType,
					},
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "references & non-references",
				types: []Type{
					Int8Type,
					&ReferenceType{
						Type: Int8Type,
					},
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "struct references & resource reference",
				types: []Type{
					&ReferenceType{
						Type: Int8Type,
					},
					&ReferenceType{
						Type: resourceType,
					},
				},
				expectedSuperType: AnyStructType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Path types", func(t *testing.T) {
		t.Parallel()

		tests := []testCase{
			{
				name: "homogenous paths",
				types: []Type{
					PrivatePathType,
					PrivatePathType,
				},
				expectedSuperType: PrivatePathType,
			},
			{
				name: "capability paths",
				types: []Type{
					PrivatePathType,
					PublicPathType,
				},
				expectedSuperType: CapabilityPathType,
			},
			{
				name: "heterogeneous paths",
				types: []Type{
					PrivatePathType,
					PublicPathType,
					StoragePathType,
				},
				expectedSuperType: PathType,
			},
			{
				name: "paths & non-paths",
				types: []Type{
					PrivatePathType,
					PublicPathType,
					StoragePathType,
					StringType,
				},
				expectedSuperType: AnyStructType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Restricted types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		interfaceType1 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I1",
			CompositeKind: common.CompositeKindStructure,
			Members:       NewStringMemberOrderedMap(),
		}

		restrictedType1 := &RestrictedType{
			Type:         AnyStructType,
			Restrictions: []*InterfaceType{interfaceType1},
		}

		restrictedType2 := &RestrictedType{
			Type:         AnyResourceType,
			Restrictions: []*InterfaceType{interfaceType1},
		}

		tests := []testCase{
			{
				name: "homogenous",
				types: []Type{
					restrictedType1,
					restrictedType1,
				},
				expectedSuperType: restrictedType1,
			},
			{
				name: "heterogeneous",
				types: []Type{
					restrictedType1,
					restrictedType2,
				},
				expectedSuperType: InvalidType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Capability types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		interfaceType1 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I1",
			CompositeKind: common.CompositeKindStructure,
			Members:       NewStringMemberOrderedMap(),
		}

		restrictedType1 := &RestrictedType{
			Type:         AnyStructType,
			Restrictions: []*InterfaceType{interfaceType1},
		}

		restrictedType2 := &RestrictedType{
			Type:         AnyResourceType,
			Restrictions: []*InterfaceType{interfaceType1},
		}

		tests := []testCase{
			{
				name: "homogenous",
				types: []Type{
					restrictedType1,
					restrictedType1,
				},
				expectedSuperType: restrictedType1,
			},
			{
				name: "heterogeneous",
				types: []Type{
					restrictedType1,
					restrictedType2,
				},
				expectedSuperType: InvalidType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Function types", func(t *testing.T) {
		t.Parallel()

		funcType1 := &FunctionType{
			Parameters: []*Parameter{
				{
					TypeAnnotation: NewTypeAnnotation(StringType),
				},
			},
			ReturnTypeAnnotation: NewTypeAnnotation(Int8Type),
			Members:              NewStringMemberOrderedMap(),
		}

		funcType2 := &FunctionType{
			Parameters: []*Parameter{
				{
					TypeAnnotation: NewTypeAnnotation(IntType),
				},
			},
			ReturnTypeAnnotation: NewTypeAnnotation(Int8Type),
			Members:              NewStringMemberOrderedMap(),
		}

		tests := []testCase{
			{
				name: "homogenous",
				types: []Type{
					funcType1,
					funcType1,
				},
				expectedSuperType: funcType1,
			},
			{
				name: "heterogeneous",
				types: []Type{
					funcType1,
					funcType2,
				},
				expectedSuperType: AnyStructType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Lower mask types", func(t *testing.T) {
		for _, typeTag := range allLowerMaskedTypeTags {
			// Upper mask must be zero
			assert.Equal(t, uint64(0), typeTag.upperMask)

			switch typeTag.lowerMask {
			case
				// No such types available
				unsignedIntegerTypeMask,
				unsignedFixedPointTypeMask,

				compositeTypeMask,
				constantSizedTypeMask:
				continue
			}

			// findSuperTypeFromLowerMask must implement all lower-masked types
			t.Run(fmt.Sprintf("mask_%d", typeTag.lowerMask), func(t *testing.T) {
				typ := findSuperTypeFromLowerMask(typeTag, nil)
				assert.NotNil(t, typ, fmt.Sprintf("not implemented %v", typeTag))
			})
		}
	})

	t.Run("Upper mask types", func(t *testing.T) {
		for _, typeTag := range allUpperMaskedTypeTags {
			// Lower mask must be zero
			assert.Equal(t, uint64(0), typeTag.lowerMask)

			// findSuperTypeFromUpperMask must implement all upper-masked types
			t.Run(fmt.Sprintf("mask_%d", typeTag.upperMask), func(t *testing.T) {
				typ := findSuperTypeFromUpperMask(typeTag, nil)
				assert.NotNil(t, typ, fmt.Sprintf("not implemented %v", typeTag))
			})
		}
	})

	t.Run("Optional types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		structType := &CompositeType{
			Location:   testLocation,
			Identifier: "T",
			Kind:       common.CompositeKindStructure,
			Members:    NewStringMemberOrderedMap(),
		}

		optionalStructType := &OptionalType{
			Type: structType,
		}

		doubleOptionalStructType := &OptionalType{
			Type: &OptionalType{
				Type: structType,
			},
		}

		tests := []testCase{
			{
				name: "simple types",
				types: []Type{
					&OptionalType{
						Type: IntType,
					},
					Int8Type,
					StringType,
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "nil with simple type",
				types: []Type{
					nilType,
					Int8Type,
				},
				expectedSuperType: &OptionalType{
					Type: Int8Type,
				},
			},
			{
				name: "nil with heterogeneous types",
				types: []Type{
					nilType,
					Int8Type,
					StringType,
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "multi-level simple optional types",
				types: []Type{
					Int8Type,
					&OptionalType{
						Type: Int8Type,
					},
					&OptionalType{
						Type: &OptionalType{
							Type: Int8Type,
						},
					},
				},

				// supertype of `T`, `T?`, `T??` is `T??`
				expectedSuperType: &OptionalType{
					Type: &OptionalType{
						Type: Int8Type,
					},
				},
			},
			{
				name: "multi-level optional structs",
				types: []Type{
					structType,
					optionalStructType,
					doubleOptionalStructType,
				},

				// supertype of `T`, `T?`, `T??` is `T??`
				expectedSuperType: doubleOptionalStructType,
			},
			{
				name: "multi-level heterogeneous optional types",
				types: []Type{
					&OptionalType{
						Type: Int8Type,
					},
					optionalStructType,
					doubleOptionalStructType,
				},

				expectedSuperType: AnyStructType,
			},
			{
				name: "multi-level heterogeneous types",
				types: []Type{
					Int8Type,
					optionalStructType,
					doubleOptionalStructType,
				},

				expectedSuperType: AnyStructType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})
}

func TestTypeInclusions(t *testing.T) {

	t.Parallel()

	// Test whether Number type-tag includes all numeric types.
	t.Run("Number", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllNumberTypes {
			t.Run(typ.String(), func(t *testing.T) {
				t.Parallel()
				assert.True(t, NumberTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("Integer", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllIntegerTypes {
			t.Run(typ.String(), func(t *testing.T) {
				t.Parallel()
				assert.True(t, IntegerTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("SignedInteger", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllSignedIntegerTypes {
			t.Run(typ.String(), func(t *testing.T) {
				t.Parallel()
				assert.True(t, SignedIntegerTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("UnsignedInteger", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllUnsignedIntegerTypes {
			t.Run(typ.String(), func(t *testing.T) {
				t.Parallel()
				assert.True(t, UnsignedIntegerTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("FixedPoint", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllFixedPointTypes {
			t.Run(typ.String(), func(t *testing.T) {
				t.Parallel()
				assert.True(t, FixedPointTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("SignedFixedPoint", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllSignedFixedPointTypes {
			t.Run(typ.String(), func(t *testing.T) {
				t.Parallel()
				assert.True(t, SignedFixedPointTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("UnsignedFixedPoint", func(t *testing.T) {
		for _, typ := range AllUnsignedFixedPointTypes {
			t.Run(typ.String(), func(t *testing.T) {
				t.Parallel()
				assert.True(t, UnsignedFixedPointTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	// Test whether Any type-tag includes all the types.
	t.Run("Any", func(t *testing.T) {
		t.Parallel()

		err := BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				typ := variable.Type
				if _, ok := typ.(*CompositeType); ok {
					return
				}

				assert.True(t, AnyTypeTag.ContainsAny(typ.Tag()))
			})
			return nil
		})

		require.NoError(t, err)
	})

	// Test whether AnyStruct type-tag includes all the pre-known AnyStruct types.
	t.Run("AnyStruct", func(t *testing.T) {
		t.Parallel()

		err := BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				typ := variable.Type

				if _, ok := typ.(*CompositeType); ok {
					return
				}

				if typ.IsResourceType() {
					return
				}

				assert.True(t, AnyStructTypeTag.ContainsAny(typ.Tag()))
			})
			return nil
		})

		require.NoError(t, err)
	})
}
