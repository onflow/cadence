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
			Purity: FunctionPurityImpure,
			Parameters: []Parameter{
				{
					TypeAnnotation: Int8TypeAnnotation,
				},
			},
			ReturnTypeAnnotation: Int16TypeAnnotation,
		},
		Size: 2,
	}

	assert.Equal(t,
		"[fun(Int8): Int16; 2]",
		ty.String(),
	)
}

func TestConstantSizedType_String_OfViewFunctionType(t *testing.T) {

	t.Parallel()

	ty := &ConstantSizedType{
		Type: &FunctionType{
			Purity: FunctionPurityView,
			Parameters: []Parameter{
				{
					TypeAnnotation: Int8TypeAnnotation,
				},
			},
			ReturnTypeAnnotation: Int16TypeAnnotation,
		},
		Size: 2,
	}

	assert.Equal(t,
		"[view fun(Int8): Int16; 2]",
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
			Parameters: []Parameter{
				{
					TypeAnnotation: Int8TypeAnnotation,
				},
			},
			ReturnTypeAnnotation: Int16TypeAnnotation,
		},
	}

	assert.Equal(t,
		"[fun(Int8): Int16]",
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

func TestIntersectionType_StringAndID(t *testing.T) {

	t.Parallel()

	t.Run("intersected types", func(t *testing.T) {

		t.Parallel()

		interfaceType := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I",
			Location:      common.StringLocation("b"),
		}

		ty := &IntersectionType{
			Types: []*InterfaceType{interfaceType},
		}

		assert.Equal(t,
			"{I}",
			ty.String(),
		)

		assert.Equal(t,
			TypeID("{S.b.I}"),
			ty.ID(),
		)
	})

	t.Run("intersected types", func(t *testing.T) {

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

		ty := &IntersectionType{
			Types: []*InterfaceType{i1, i2},
		}

		assert.Equal(t,
			ty.String(),
			"{I1, I2}",
		)

		assert.Equal(t,
			TypeID("{S.b.I1,S.c.I2}"),
			ty.ID(),
		)
	})
}

func TestIntersectionType_Equals(t *testing.T) {

	t.Parallel()

	t.Run("more intersected types", func(t *testing.T) {

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

		a := &IntersectionType{
			Types: []*InterfaceType{i1},
		}

		b := &IntersectionType{
			Types: []*InterfaceType{i1, i2},
		}

		assert.False(t, a.Equal(b))
	})

	t.Run("fewer intersected types", func(t *testing.T) {

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

		a := &IntersectionType{
			Types: []*InterfaceType{i1, i2},
		}

		b := &IntersectionType{
			Types: []*InterfaceType{i1},
		}

		assert.False(t, a.Equal(b))
	})

	t.Run("same intersected types", func(t *testing.T) {

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

		a := &IntersectionType{
			Types: []*InterfaceType{i1, i2},
		}

		b := &IntersectionType{
			Types: []*InterfaceType{i1, i2},
		}

		assert.True(t, a.Equal(b))
	})
}

func TestIntersectionType_GetMember(t *testing.T) {

	t.Parallel()

	t.Run("allow declared members", func(t *testing.T) {

		t.Parallel()

		interfaceType := &InterfaceType{
			CompositeKind: common.CompositeKindResource,
			Identifier:    "I",
			Members:       &StringMemberOrderedMap{},
		}

		resourceType := &CompositeType{
			Kind:       common.CompositeKindResource,
			Identifier: "R",
			Location:   common.StringLocation("a"),
			Fields:     []string{},
			Members:    &StringMemberOrderedMap{},
		}
		intersectionType := &IntersectionType{
			Types: []*InterfaceType{
				interfaceType,
			},
		}

		fieldName := "s"

		interfaceMember := NewUnmeteredPublicConstantFieldMember(
			resourceType,
			fieldName,
			IntType,
			"",
		)
		interfaceType.Members.Set(fieldName, interfaceMember)

		actualMembers := intersectionType.GetMembers()

		require.Contains(t, actualMembers, fieldName)

		actualMember := actualMembers[fieldName].Resolve(nil, fieldName, ast.EmptyRange, nil)

		assert.Same(t, interfaceMember, actualMember)
	})
}

func TestBeforeType_Strings(t *testing.T) {

	t.Parallel()

	expected := "view fun<T: AnyStruct>(_ value: T): T"

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

	t.Parallel()

	t.Run("with containers", func(t *testing.T) {

		t.Parallel()

		a := &CompositeType{
			Kind:       common.CompositeKindStructure,
			Identifier: "A",
			Location:   common.StringLocation("a"),
			Fields:     []string{},
			Members:    &StringMemberOrderedMap{},
		}

		b := &CompositeType{
			Kind:          common.CompositeKindStructure,
			Identifier:    "B",
			Location:      common.StringLocation("a"),
			Fields:        []string{},
			Members:       &StringMemberOrderedMap{},
			containerType: a,
		}

		c := &CompositeType{
			Kind:          common.CompositeKindStructure,
			Identifier:    "C",
			Location:      common.StringLocation("a"),
			Fields:        []string{},
			Members:       &StringMemberOrderedMap{},
			containerType: b,
		}

		identifier := qualifiedIdentifier("foo", c)
		assert.Equal(t, "A.B.C.foo", identifier)
	})

	t.Run("without containers", func(t *testing.T) {
		t.Parallel()

		identifier := qualifiedIdentifier("foo", nil)
		assert.Equal(t, "foo", identifier)
	})

	t.Run("account container", func(t *testing.T) {
		t.Parallel()

		identifier := qualifiedIdentifier("foo", AccountType)
		assert.Equal(t, "Account.foo", identifier)
	})
}

func BenchmarkQualifiedIdentifierCreation(b *testing.B) {

	foo := &CompositeType{
		Kind:       common.CompositeKindStructure,
		Identifier: "foo",
		Location:   common.StringLocation("a"),
		Fields:     []string{},
		Members:    &StringMemberOrderedMap{},
	}

	bar := &CompositeType{
		Kind:          common.CompositeKindStructure,
		Identifier:    "bar",
		Location:      common.StringLocation("a"),
		Fields:        []string{},
		Members:       &StringMemberOrderedMap{},
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

	t.Parallel()

	code := `

      contract interface Test {

          struct interface NestedInterface {
              fun test(): Bool
          }

          struct interface Nested: NestedInterface {}
      }

      contract TestImpl {

          struct Nested: Test.Nested {
              fun test(): Bool {
                  return true
              }
          }
      }
	`

	program, err := parser.ParseProgram(nil, []byte(code), parser.Config{})
	require.NoError(t, err)

	checker, err := NewChecker(
		program,
		common.StringLocation("test"),
		nil,
		&Config{
			AccessCheckMode: AccessCheckModeNotSpecifiedUnrestricted,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	var typeIDs []common.TypeID

	checker.typeActivations.ForEachVariableDeclaredInAndBelow(
		checker.valueActivations.Depth(),
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
					semaType.clearCachedIdentifiers()

					recalculatedQualifiedID := semaType.QualifiedIdentifier()
					recalculatedID := semaType.ID()

					assert.Equal(t, recalculatedQualifiedID, cachedQualifiedID)
					assert.Equal(t, recalculatedID, cachedID)

					typeIDs = append(typeIDs, recalculatedID)

					// Recursively check for nested types
					checkNestedTypes(semaType.NestedTypes)

				case *InterfaceType:
					cachedQualifiedID := semaType.QualifiedIdentifier()
					cachedID := semaType.ID()

					// clear cached identifiers for one level
					semaType.clearCachedIdentifiers()

					recalculatedQualifiedID := semaType.QualifiedIdentifier()
					recalculatedID := semaType.ID()

					assert.Equal(t, recalculatedQualifiedID, cachedQualifiedID)
					assert.Equal(t, recalculatedID, cachedID)

					typeIDs = append(typeIDs, recalculatedID)

					// Recursively check for nested types
					checkNestedTypes(semaType.NestedTypes)
				}
			}

			checkIdentifiers(t, typ)
		},
	)

	assert.Equal(t,
		[]common.TypeID{
			"S.test.Test",
			"S.test.Test.NestedInterface",
			"S.test.Test.Nested",
			"S.test.TestImpl",
			"S.test.TestImpl.Nested",
		},
		typeIDs,
	)

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

	nilType := &OptionalType{
		Type: NeverType,
	}

	resourceType := &CompositeType{
		Location:   nil,
		Identifier: "Foo",
		Kind:       common.CompositeKindResource,
	}

	type testCase struct {
		expectedSuperType Type
		name              string
		types             []Type
	}

	testLeastCommonSuperType := func(t *testing.T, tests []testCase) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.True(
					t,
					test.expectedSuperType.Equal(LeastCommonSuperType(test.types...)),
				)
			})
		}
	}

	t.Run("All types", func(t *testing.T) {
		t.Parallel()

		// super type of similar types should be the type itself.
		// i.e: super type of collection of T's should be T.
		// Make sure it's true for all known types.

		var tests []testCase

		err := BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
			// Entitlements are not typical types. So skip.
			if _, ok := BuiltinEntitlements[name]; ok {
				return nil
			}
			if _, ok := BuiltinEntitlementMappings[name]; ok {
				return nil
			}

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
					Word128Type,
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
					UIntType,
				},
				expectedSuperType: IntegerType,
			},
			{
				name: "fixed size unsigned integers",
				types: []Type{
					UInt8Type,
					UInt128Type,
				},
				expectedSuperType: FixedSizeUnsignedIntegerType,
			},
			{
				name: "heterogeneous simple types",
				types: []Type{
					StringType,
					Int8Type,
				},
				expectedSuperType: HashableStructType,
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
			Members:       &StringMemberOrderedMap{},
		}

		interfaceType2 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I2",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		interfaceType3 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I3",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		superInterfaceType := &InterfaceType{
			Location:      testLocation,
			Identifier:    "SI",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		inheritedInterfaceType1 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "II1",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
			ExplicitInterfaceConformances: []*InterfaceType{
				superInterfaceType,
			},
		}

		inheritedInterfaceType2 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "II2",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
			ExplicitInterfaceConformances: []*InterfaceType{
				superInterfaceType,
			},
		}

		newCompositeWithInterfaces := func(name string, interfaces ...*InterfaceType) *CompositeType {
			return &CompositeType{
				Location:                      testLocation,
				Identifier:                    name,
				Kind:                          common.CompositeKindStructure,
				ExplicitInterfaceConformances: interfaces,
				Members:                       &StringMemberOrderedMap{},
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
					AccountType,
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
				expectedSuperType: func() Type {
					typ := &IntersectionType{
						Types: []*InterfaceType{interfaceType2},
					}
					// just initialize for equality
					typ.initializeEffectiveIntersectionSet()
					return typ
				}(),
			},
			{
				name: "multiple common interfaces",
				types: []Type{
					newCompositeWithInterfaces("Foo", interfaceType1, interfaceType2),
					newCompositeWithInterfaces("Baz", interfaceType1, interfaceType2, interfaceType3),
				},
				expectedSuperType: func() Type {
					typ := &IntersectionType{
						Types: []*InterfaceType{interfaceType1, interfaceType2},
					}
					// just initialize for equality
					typ.initializeEffectiveIntersectionSet()
					return typ
				}(),
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
				name: "inherited common interface",
				types: []Type{
					newCompositeWithInterfaces("Foo", inheritedInterfaceType1),
					newCompositeWithInterfaces("Bar", inheritedInterfaceType2),
				},
				expectedSuperType: func() Type {
					typ := &IntersectionType{
						Types: []*InterfaceType{superInterfaceType},
					}

					// just initialize for equality
					typ.initializeEffectiveIntersectionSet()
					return typ
				}(),
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
				expectedSuperType: &VariableSizedType{Type: HashableStructType},
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
					ValueType: HashableStructType,
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

	t.Run("Reference types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		entitlementE := NewEntitlementType(nil, testLocation, "E")
		entitlementF := NewEntitlementType(nil, testLocation, "F")
		entitlementG := NewEntitlementType(nil, testLocation, "G")
		entitlementM := NewEntitlementMapType(nil, testLocation, "E")

		entitlementsEOnly := NewEntitlementSetAccess([]*EntitlementType{entitlementE}, Conjunction)
		entitlementsEAndF := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementF}, Conjunction)
		entitlementsEAndG := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementG}, Conjunction)
		entitlementsFAndG := NewEntitlementSetAccess([]*EntitlementType{entitlementF, entitlementG}, Conjunction)
		entitlementsEAndFAndG := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementG, entitlementF}, Conjunction)
		entitlementsEOrFOrG := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementF, entitlementG}, Disjunction)
		entitlementsEOrG := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementG}, Disjunction)
		entitlementsEOrF := NewEntitlementSetAccess([]*EntitlementType{entitlementE, entitlementF}, Disjunction)
		entitlementsFOrG := NewEntitlementSetAccess([]*EntitlementType{entitlementG, entitlementF}, Disjunction)
		entitlementsM := NewEntitlementMapAccess(entitlementM)

		tests := []testCase{
			{
				name: "homogenous references",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: UnauthorizedAccess,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: UnauthorizedAccess,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: UnauthorizedAccess,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: UnauthorizedAccess,
				},
			},
			{
				name: "heterogeneous references",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: UnauthorizedAccess,
					},
					&ReferenceType{
						Type:          StringType,
						Authorization: UnauthorizedAccess,
					},
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "references & non-references",
				types: []Type{
					Int8Type,
					&ReferenceType{
						Type:          Int8Type,
						Authorization: UnauthorizedAccess,
					},
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "struct references & resource reference",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: UnauthorizedAccess,
					},
					&ReferenceType{
						Type:          resourceType,
						Authorization: UnauthorizedAccess,
					},
				},
				expectedSuperType: AnyStructType,
			},
			{
				name: "auth and non-auth references",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: UnauthorizedAccess,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: EntitlementSetAccess{},
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: UnauthorizedAccess,
				},
			},
			{
				name: "E and (E, F)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOnly,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEAndF,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOnly,
				},
			},
			{
				name: "E and (F, G)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOnly,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsFAndG,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOrFOrG,
				},
			},
			{
				name: "E and (E | G)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOnly,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOrG,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOrG,
				},
			},
			{
				name: "E and (F | G)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOnly,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsFOrG,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOrFOrG,
				},
			},
			{
				name: "(F | G) and E",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsFOrG,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOnly,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOrFOrG,
				},
			},
			{
				name: "(E, F) and (E | G)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEAndF,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOrG,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOrG,
				},
			},
			{
				name: "(E, F) and (E, G)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEAndF,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEAndG,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOnly,
				},
			},
			{
				name: "(E, F) and (E, F, G)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEAndF,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEAndFAndG,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEAndF,
				},
			},
			{
				name: "(E | G) and (E | F)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOrG,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOrF,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOrFOrG,
				},
			},
			{
				name: "(E | G) and (E | F | G)",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOrG,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOrFOrG,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: entitlementsEOrFOrG,
				},
			},
			{
				name: "M and E",
				types: []Type{
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsEOnly,
					},
					&ReferenceType{
						Type:          Int8Type,
						Authorization: entitlementsM,
					},
				},
				expectedSuperType: &ReferenceType{
					Type:          Int8Type,
					Authorization: UnauthorizedAccess,
				},
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
				expectedSuperType: HashableStructType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Intersection types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		interfaceType1 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I1",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		interfaceType2 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I2",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		intersectionType1 := &IntersectionType{
			Types: []*InterfaceType{interfaceType1},
		}

		intersectionType2 := &IntersectionType{
			Types: []*InterfaceType{interfaceType2},
		}

		tests := []testCase{
			{
				name: "homogenous",
				types: []Type{
					intersectionType1,
					intersectionType1,
				},
				expectedSuperType: intersectionType1,
			},
			{
				name: "heterogeneous",
				types: []Type{
					intersectionType1,
					intersectionType2,
				},
				expectedSuperType: AnyStructType,
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
			Members:       &StringMemberOrderedMap{},
		}

		interfaceType2 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I1",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		capType1 := &CapabilityType{
			BorrowType: &IntersectionType{
				Types: []*InterfaceType{interfaceType1},
			},
		}

		capType2 := &CapabilityType{
			BorrowType: &IntersectionType{
				Types: []*InterfaceType{interfaceType2},
			},
		}

		tests := []testCase{
			{
				name: "homogenous",
				types: []Type{
					capType1,
					capType1,
				},
				expectedSuperType: capType1,
			},
			{
				name: "heterogeneous",
				types: []Type{
					capType1,
					capType2,
				},
				expectedSuperType: AnyStructType,
			},
		}

		testLeastCommonSuperType(t, tests)
	})

	t.Run("Function types", func(t *testing.T) {
		t.Parallel()

		funcType1 := &FunctionType{
			Purity: FunctionPurityImpure,
			Parameters: []Parameter{
				{
					TypeAnnotation: StringTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: Int8TypeAnnotation,
			Members:              &StringMemberOrderedMap{},
		}

		funcType2 := &FunctionType{
			Purity: FunctionPurityImpure,
			Parameters: []Parameter{
				{
					TypeAnnotation: IntTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: Int8TypeAnnotation,
			Members:              &StringMemberOrderedMap{},
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
		t.Parallel()

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
		t.Parallel()

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

	t.Run("Upper and lower mask types", func(t *testing.T) {
		t.Parallel()

		lowerMaskTypes := []Type{
			NilType,
			Int64Type,
			AnyStructType,
			AnyResourceType,
		}

		upperMaskTypes := []Type{
			&CapabilityType{
				BorrowType: AnyStructType,
			},
			&IntersectionType{
				Types: []*InterfaceType{
					{
						Location:   common.StringLocation("test"),
						Identifier: "Foo",
					},
				},
			},
		}

		for _, firstType := range lowerMaskTypes {
			for _, secondType := range upperMaskTypes {
				superType := leastCommonSuperType(firstType, secondType)

				switch firstType {
				case AnyResourceType:
					assert.Equal(t, InvalidType, superType)
				case NilType:
					assert.Equal(t, &OptionalType{Type: secondType}, superType)
				default:
					assert.Equal(t, AnyStructType, superType)
				}
			}
		}
	})

	t.Run("Optional types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		structType := &CompositeType{
			Location:   testLocation,
			Identifier: "T",
			Kind:       common.CompositeKindStructure,
			Members:    &StringMemberOrderedMap{},
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
				expectedSuperType: &OptionalType{
					Type: HashableStructType,
				},
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
				expectedSuperType: &OptionalType{
					Type: HashableStructType,
				},
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

func TestIsPrimitive(t *testing.T) {
	t.Parallel()

	resourceType := &CompositeType{
		Location:   nil,
		Identifier: "Foo",
		Kind:       common.CompositeKindResource,
	}

	type testCase struct {
		expectedIsPrimitive bool
		name                string
		ty                  Type
	}

	testIsPrimitive := func(t *testing.T, tests []testCase) {
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				assert.Equal(t, test.expectedIsPrimitive, test.ty.IsPrimitiveType())
			})
		}
	}

	t.Run("number types", func(t *testing.T) {
		t.Parallel()

		var tests []testCase
		for _, ty := range AllNumberTypes {
			tests = append(tests, testCase{
				expectedIsPrimitive: true,
				name:                string(ty.ID()),
				ty:                  ty,
			})
		}

		testIsPrimitive(t, tests)
	})

	t.Run("simple types", func(t *testing.T) {
		t.Parallel()

		var tests []testCase
		for _, ty := range []Type{
			CharacterType,
			BoolType,
			StringType,
			TheAddressType,
			PrivatePathType,
			PublicPathType,
			StoragePathType,
			VoidType,
		} {
			tests = append(tests, testCase{
				expectedIsPrimitive: true,
				name:                string(ty.ID()),
				ty:                  ty,
			})
		}

		for _, ty := range []Type{
			&GenericType{TypeParameter: &TypeParameter{Name: "T"}},
			&TransactionType{},
		} {
			tests = append(tests, testCase{
				expectedIsPrimitive: false,
				name:                string(ty.ID()),
				ty:                  ty,
			})
		}

		testIsPrimitive(t, tests)
	})

	t.Run("Optional types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		structType := &CompositeType{
			Location:   testLocation,
			Identifier: "T",
			Kind:       common.CompositeKindStructure,
			Members:    &StringMemberOrderedMap{},
		}

		optionalStructType := &OptionalType{
			Type: structType,
		}

		doubleOptionalStructType := &OptionalType{
			Type: &OptionalType{
				Type: structType,
			},
		}

		var tests []testCase
		for _, ty := range []Type{
			CharacterType,
			BoolType,
			StringType,
			TheAddressType,
			PrivatePathType,
			PublicPathType,
			StoragePathType,
			VoidType,
		} {
			tests = append(tests, testCase{
				expectedIsPrimitive: true,
				name:                fmt.Sprintf("Optional<%s>", string(ty.ID())),
				ty:                  &OptionalType{Type: ty},
			})

			tests = append(tests, testCase{
				expectedIsPrimitive: true,
				name:                fmt.Sprintf("Optional<Optional<%s>>", string(ty.ID())),
				ty:                  &OptionalType{Type: &OptionalType{Type: ty}},
			})
		}

		tests = append(tests, testCase{
			expectedIsPrimitive: false,
			name:                "Optional<Struct>",
			ty:                  optionalStructType,
		})

		tests = append(tests, testCase{
			expectedIsPrimitive: false,
			name:                "Optional<Optional<Struct>>",
			ty:                  doubleOptionalStructType,
		})

		testIsPrimitive(t, tests)
	})

	t.Run("Arrays", func(t *testing.T) {
		t.Parallel()

		var tests []testCase
		err := BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
			// Entitlements are not typical types. So skip.
			if _, ok := BuiltinEntitlements[name]; ok {
				return nil
			}
			if _, ok := BuiltinEntitlementMappings[name]; ok {
				return nil
			}

			typ := variable.Type

			tests = append(tests, testCase{
				name:                fmt.Sprintf("VariableSizedType<%s>", name),
				ty:                  &VariableSizedType{Type: typ},
				expectedIsPrimitive: false,
			})

			tests = append(tests, testCase{
				name:                fmt.Sprintf("ConstantSizedType<%s>", name),
				ty:                  &ConstantSizedType{Type: typ, Size: 1},
				expectedIsPrimitive: false,
			})

			return nil
		})

		require.NoError(t, err)
		testIsPrimitive(t, tests)
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
				name:                "Dictionary<String,String>",
				ty:                  stringStringDictionary,
				expectedIsPrimitive: false,
			},
			{
				name:                "Dictionary<String,Bool>",
				ty:                  stringBoolDictionary,
				expectedIsPrimitive: false,
			},
			{
				name:                "Dictionary<String,Resource>",
				ty:                  stringResourceDictionary,
				expectedIsPrimitive: false,
			},
			{
				name:                "Dictionary<String,Dictionary<String,Resource>",
				ty:                  nestedResourceDictionary,
				expectedIsPrimitive: false,
			},
			{
				name:                "Dictionary<String,Dictionary<String,String>",
				ty:                  nestedStringDictionary,
				expectedIsPrimitive: false,
			},
		}

		testIsPrimitive(t, tests)
	})

	t.Run("References types", func(t *testing.T) {
		t.Parallel()

		var tests []testCase
		err := BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
			// Entitlements are not typical types. So skip.
			if _, ok := BuiltinEntitlements[name]; ok {
				return nil
			}
			if _, ok := BuiltinEntitlementMappings[name]; ok {
				return nil
			}

			typ := variable.Type

			tests = append(tests, testCase{
				name:                fmt.Sprintf("ReferenceType<%s>", name),
				ty:                  &ReferenceType{Type: typ},
				expectedIsPrimitive: false,
			})

			return nil
		})

		require.NoError(t, err)
		testIsPrimitive(t, tests)
	})

	t.Run("Capability types", func(t *testing.T) {
		t.Parallel()

		testLocation := common.StringLocation("test")

		interfaceType1 := &InterfaceType{
			Location:      testLocation,
			Identifier:    "I1",
			CompositeKind: common.CompositeKindStructure,
			Members:       &StringMemberOrderedMap{},
		}

		capType := &CapabilityType{
			BorrowType: &IntersectionType{
				Types: []*InterfaceType{interfaceType1},
			},
		}

		tests := []testCase{
			{
				name:                "CapabilityType",
				ty:                  capType,
				expectedIsPrimitive: false,
			},
		}

		testIsPrimitive(t, tests)
	})

	t.Run("Function types", func(t *testing.T) {
		t.Parallel()

		funcType1 := &FunctionType{
			Purity: FunctionPurityImpure,
			Parameters: []Parameter{
				{
					TypeAnnotation: StringTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: Int8TypeAnnotation,
			Members:              &StringMemberOrderedMap{},
		}

		funcType2 := &FunctionType{
			Purity: FunctionPurityImpure,
			Parameters: []Parameter{
				{
					TypeAnnotation: IntTypeAnnotation,
				},
			},
			ReturnTypeAnnotation: PublicPathTypeAnnotation,
			Members:              &StringMemberOrderedMap{},
		}

		tests := []testCase{
			{
				name:                "Function(String): Int8",
				ty:                  funcType1,
				expectedIsPrimitive: false,
			},
			{
				name:                "Function(Int): PublicPath",
				ty:                  funcType2,
				expectedIsPrimitive: false,
			},
		}

		testIsPrimitive(t, tests)
	})
}

func TestTypeInclusions(t *testing.T) {

	t.Parallel()

	// Test whether Number type-tag includes all numeric types.
	t.Run("Number", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllNumberTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, NumberTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("Integer", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllIntegerTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, IntegerTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("SignedInteger", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllSignedIntegerTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, SignedIntegerTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("UnsignedInteger", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllUnsignedIntegerTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, UnsignedIntegerTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("FixedSizeUnsignedInteger", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllFixedSizeUnsignedIntegerTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, FixedSizeUnsignedIntegerTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("FixedPoint", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllFixedPointTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, FixedPointTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("SignedFixedPoint", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllSignedFixedPointTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, SignedFixedPointTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	t.Run("UnsignedFixedPoint", func(t *testing.T) {
		t.Parallel()

		for _, typ := range AllUnsignedFixedPointTypes {
			t.Run(typ.String(), func(t *testing.T) {
				assert.True(t, UnsignedFixedPointTypeTag.ContainsAny(typ.Tag()))
			})
		}
	})

	// Test whether Any type-tag includes all the types.
	t.Run("Any", func(t *testing.T) {
		t.Parallel()

		err := BaseTypeActivation.ForEach(func(name string, variable *Variable) error {
			// Entitlements are not typical types. So skip.
			if _, ok := BuiltinEntitlements[name]; ok {
				return nil
			}
			if _, ok := BuiltinEntitlementMappings[name]; ok {
				return nil
			}

			t.Run(name, func(t *testing.T) {

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
			// Entitlements are not typical types. So skip.
			if _, ok := BuiltinEntitlements[name]; ok {
				return nil
			}
			if _, ok := BuiltinEntitlementMappings[name]; ok {
				return nil
			}

			t.Run(name, func(t *testing.T) {

				typ := variable.Type

				if _, ok := typ.(*CompositeType); ok {
					return
				}

				if _, ok := typ.(*InterfaceType); ok {
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

func BenchmarkSuperTypeInference(b *testing.B) {

	b.Run("integers", func(b *testing.B) {
		types := []Type{
			UInt8Type,
			UInt256Type,
			IntegerType,
			Word64Type,
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			LeastCommonSuperType(types...)
		}
	})

	b.Run("arrays", func(b *testing.B) {
		types := []Type{
			&VariableSizedType{
				Type: IntType,
			},
			&VariableSizedType{
				Type: Int8Type,
			},
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			LeastCommonSuperType(types...)
		}
	})

	b.Run("composites", func(b *testing.B) {
		types := []Type{
			PublicKeyType,
			AccountType,
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			LeastCommonSuperType(types...)
		}
	})
}

func TestMapType(t *testing.T) {

	t.Parallel()

	mapFn := func(ty Type) Type {
		switch typ := ty.(type) {
		case *SimpleType:
			return BoolType
		case *NumericType:
			return StringType
		case *CompositeType:
			return &InterfaceType{Identifier: typ.Identifier}
		case *IntersectionType:
			var interfaces []*InterfaceType
			for _, i := range typ.Types {
				interfaces = append(interfaces, &InterfaceType{Identifier: i.Identifier + "f"})
			}
			return NewIntersectionType(nil, nil, interfaces)
		}
		return ty
	}

	t.Run("map optional", func(t *testing.T) {
		t.Parallel()
		original := NewOptionalType(nil, StringType)
		mapped := NewOptionalType(nil, BoolType)

		require.Equal(t, mapped, original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn))
	})

	t.Run("map variable array", func(t *testing.T) {
		t.Parallel()
		original := NewVariableSizedType(nil, StringType)
		mapped := NewVariableSizedType(nil, BoolType)

		require.Equal(t, mapped, original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn))
	})

	t.Run("map constant sized array", func(t *testing.T) {
		t.Parallel()
		original := NewConstantSizedType(nil, StringType, 7)
		mapped := NewConstantSizedType(nil, BoolType, 7)

		require.Equal(t, mapped, original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn))
	})

	t.Run("map reference type", func(t *testing.T) {
		t.Parallel()
		mapType := NewEntitlementMapAccess(&EntitlementMapType{Identifier: "X"})
		original := NewReferenceType(nil, mapType, StringType)
		mapped := NewReferenceType(nil, mapType, BoolType)

		require.Equal(t, mapped, original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn))
	})

	t.Run("map dictionary type", func(t *testing.T) {
		t.Parallel()
		original := NewDictionaryType(nil, StringType, Int128Type)
		mapped := NewDictionaryType(nil, BoolType, StringType)

		require.Equal(t, mapped, original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn))
	})

	t.Run("map capability type", func(t *testing.T) {
		t.Parallel()
		original := NewCapabilityType(nil, StringType)
		mapped := NewCapabilityType(nil, BoolType)

		require.Equal(t, mapped, original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn))
	})

	t.Run("map intersection type", func(t *testing.T) {
		t.Parallel()

		original := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				{Identifier: "foo"},
				{Identifier: "bar"},
			},
		)
		mapped := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				{Identifier: "foof"},
				{Identifier: "barf"},
			},
		)

		require.Equal(t, mapped, original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn))
	})

	t.Run("map function type", func(t *testing.T) {
		t.Parallel()
		originalTypeParam := &TypeParameter{
			TypeBound: Int64Type,
			Name:      "X",
			Optional:  true,
		}
		original := NewSimpleFunctionType(
			FunctionPurityView,
			[]Parameter{
				{
					TypeAnnotation: NewTypeAnnotation(
						&GenericType{
							TypeParameter: originalTypeParam,
						},
					),
					Label:      "X",
					Identifier: "Y",
				},
				{
					TypeAnnotation: NewTypeAnnotation(&CompositeType{Identifier: "foo"}),
					Label:          "A",
					Identifier:     "B",
				},
			},
			NewTypeAnnotation(Int128Type),
		)
		original.TypeParameters = []*TypeParameter{originalTypeParam}

		mappedTypeParam := &TypeParameter{
			TypeBound: StringType,
			Name:      "X",
			Optional:  true,
		}
		mapped := NewSimpleFunctionType(
			FunctionPurityView,
			[]Parameter{
				{
					TypeAnnotation: NewTypeAnnotation(
						&GenericType{
							TypeParameter: mappedTypeParam,
						},
					),
					Label:      "X",
					Identifier: "Y",
				},
				{
					TypeAnnotation: NewTypeAnnotation(&InterfaceType{Identifier: "foo"}),
					Label:          "A",
					Identifier:     "B",
				},
			},
			NewTypeAnnotation(StringType),
		)
		mapped.TypeParameters = []*TypeParameter{mappedTypeParam}

		output := original.Map(nil, make(map[*TypeParameter]*TypeParameter), mapFn)

		require.IsType(t, &FunctionType{}, output)

		outputFunction := output.(*FunctionType)

		require.Equal(t, mapped, outputFunction)
		require.IsType(t, &GenericType{}, outputFunction.Parameters[0].TypeAnnotation.Type)
		require.True(t, outputFunction.Parameters[0].TypeAnnotation.Type.(*GenericType).TypeParameter == outputFunction.TypeParameters[0])
	})
}

func TestReferenceType_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	containerType := &CompositeType{
		Location:   testLocation,
		Identifier: "C",
	}

	t.Run("top-level, unauthorized", func(t *testing.T) {
		t.Parallel()

		referenceType := NewReferenceType(nil, UnauthorizedAccess, IntType)
		assert.Equal(t,
			TypeID("&Int"),
			referenceType.ID(),
		)
	})

	t.Run("top-level, authorized, map", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementMapAccess(NewEntitlementMapType(nil, testLocation, "M"))

		referenceType := NewReferenceType(nil, access, IntType)
		assert.Equal(t,
			TypeID("auth(S.test.M)&Int"),
			referenceType.ID(),
		)
	})

	t.Run("top-level, authorized, set", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Conjunction,
		)

		referenceType := NewReferenceType(nil, access, IntType)

		// NOTE: sorted
		assert.Equal(t,
			TypeID("auth(S.test.E1,S.test.E2)&Int"),
			referenceType.ID(),
		)
	})

	t.Run("nested, authorized, map", func(t *testing.T) {
		t.Parallel()

		mapType := NewEntitlementMapType(nil, testLocation, "M")
		mapType.SetContainerType(containerType)

		access := NewEntitlementMapAccess(mapType)

		referenceType := NewReferenceType(nil, access, IntType)
		assert.Equal(t,
			TypeID("auth(S.test.C.M)&Int"),
			referenceType.ID(),
		)
	})

	t.Run("nested, authorized, set", func(t *testing.T) {
		t.Parallel()

		entitlementType1 := NewEntitlementType(nil, testLocation, "E1")
		entitlementType1.SetContainerType(containerType)

		entitlementType2 := NewEntitlementType(nil, testLocation, "E2")
		entitlementType2.SetContainerType(containerType)

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				entitlementType2,
				entitlementType1,
			},
			Conjunction,
		)

		referenceType := NewReferenceType(nil, access, IntType)

		// NOTE: sorted
		assert.Equal(t,
			TypeID("auth(S.test.C.E1,S.test.C.E2)&Int"),
			referenceType.ID(),
		)
	})
}

func TestReferenceType_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("unauthorized", func(t *testing.T) {
		t.Parallel()

		referenceType := NewReferenceType(nil, UnauthorizedAccess, IntType)
		assert.Equal(t, "&Int", referenceType.String())
	})

	t.Run("top-level, authorized, map", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementMapAccess(NewEntitlementMapType(nil, testLocation, "M"))

		referenceType := NewReferenceType(nil, access, IntType)
		assert.Equal(t,
			"auth(mapping M) &Int",
			referenceType.String(),
		)
	})

	t.Run("top-level, authorized, set", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Conjunction,
		)

		referenceType := NewReferenceType(nil, access, IntType)

		// NOTE: order
		assert.Equal(t,
			"auth(E2, E1) &Int",
			referenceType.String(),
		)
	})
}

func TestReferenceType_QualifiedString(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	containerType := &CompositeType{
		Location:   testLocation,
		Identifier: "C",
	}

	t.Run("top-level, unauthorized", func(t *testing.T) {
		t.Parallel()

		referenceType := NewReferenceType(nil, UnauthorizedAccess, IntType)
		assert.Equal(t,
			"&Int",
			referenceType.QualifiedString(),
		)
	})

	t.Run("top-level, authorized, map", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementMapAccess(NewEntitlementMapType(nil, testLocation, "M"))

		referenceType := NewReferenceType(nil, access, IntType)
		assert.Equal(t,
			"auth(mapping M) &Int",
			referenceType.QualifiedString(),
		)
	})

	t.Run("top-level, authorized, set", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Conjunction,
		)

		referenceType := NewReferenceType(nil, access, IntType)

		// NOTE: order
		assert.Equal(t,
			"auth(E2, E1) &Int",
			referenceType.QualifiedString(),
		)
	})

	t.Run("nested, authorized, map", func(t *testing.T) {
		t.Parallel()

		mapType := NewEntitlementMapType(nil, testLocation, "M")
		mapType.SetContainerType(containerType)

		access := NewEntitlementMapAccess(mapType)

		referenceType := NewReferenceType(nil, access, IntType)
		assert.Equal(t,
			"auth(mapping C.M) &Int",
			referenceType.QualifiedString(),
		)
	})

	t.Run("nested, authorized, set", func(t *testing.T) {
		t.Parallel()

		entitlementType1 := NewEntitlementType(nil, testLocation, "E1")
		entitlementType1.SetContainerType(containerType)

		entitlementType2 := NewEntitlementType(nil, testLocation, "E2")
		entitlementType2.SetContainerType(containerType)

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				entitlementType2,
				entitlementType1,
			},
			Conjunction,
		)

		referenceType := NewReferenceType(nil, access, IntType)
		assert.Equal(t,
			"auth(C.E2, C.E1) &Int",
			referenceType.QualifiedString(),
		)
	})
}

func TestIntersectionType_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	containerType := &CompositeType{
		Location:   testLocation,
		Identifier: "C",
	}

	t.Run("top-level, single", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				{
					Location:   testLocation,
					Identifier: "I",
				},
			},
		)
		assert.Equal(t,
			TypeID("{S.test.I}"),
			intersectionType.ID(),
		)
	})

	t.Run("top-level, two", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				// NOTE: order
				{
					Location:   testLocation,
					Identifier: "I2",
				},
				{
					Location:   testLocation,
					Identifier: "I1",
				},
			},
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("{S.test.I1,S.test.I2}"),
			intersectionType.ID(),
		)
	})

	t.Run("nested, two", func(t *testing.T) {
		t.Parallel()

		interfaceType1 := &InterfaceType{
			Location:   testLocation,
			Identifier: "I1",
		}
		interfaceType1.SetContainerType(containerType)

		interfaceType2 := &InterfaceType{
			Location:   testLocation,
			Identifier: "I2",
		}
		interfaceType2.SetContainerType(containerType)

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				// NOTE: order
				interfaceType2,
				interfaceType1,
			},
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("{S.test.C.I1,S.test.C.I2}"),
			intersectionType.ID(),
		)
	})
}

func TestIntersectionType_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level, single", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				{
					Location:   testLocation,
					Identifier: "I",
				},
			},
		)
		assert.Equal(t,
			"{I}",
			intersectionType.String(),
		)
	})

	t.Run("top-level, two", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				// NOTE: order
				{
					Location:   testLocation,
					Identifier: "I2",
				},
				{
					Location:   testLocation,
					Identifier: "I1",
				},
			},
		)
		// NOTE: order
		assert.Equal(t,
			"{I2, I1}",
			intersectionType.String(),
		)
	})
}

func TestIntersectionType_QualifiedString(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	containerType := &CompositeType{
		Location:   testLocation,
		Identifier: "C",
	}

	t.Run("top-level, single", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				{
					Location:   testLocation,
					Identifier: "I",
				},
			},
		)
		assert.Equal(t,
			"{I}",
			intersectionType.QualifiedString(),
		)
	})

	t.Run("top-level, two", func(t *testing.T) {
		t.Parallel()

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				// NOTE: order
				{
					Location:   testLocation,
					Identifier: "I2",
				},
				{
					Location:   testLocation,
					Identifier: "I1",
				},
			},
		)
		// NOTE: order
		assert.Equal(t,
			"{I2, I1}",
			intersectionType.QualifiedString(),
		)
	})

	t.Run("nested, two", func(t *testing.T) {
		t.Parallel()

		interfaceType1 := &InterfaceType{
			Location:   testLocation,
			Identifier: "I1",
		}
		interfaceType1.SetContainerType(containerType)

		interfaceType2 := &InterfaceType{
			Location:   testLocation,
			Identifier: "I2",
		}
		interfaceType2.SetContainerType(containerType)

		intersectionType := NewIntersectionType(
			nil,
			nil,
			[]*InterfaceType{
				// NOTE: order
				interfaceType2,
				interfaceType1,
			},
		)
		// NOTE: sorted
		assert.Equal(t,
			"{C.I2, C.I1}",
			intersectionType.QualifiedString(),
		)
	})
}

func TestType_IsOrContainsReference(t *testing.T) {

	t.Parallel()

	type testCase struct {
		name     string
		ty       Type
		expected bool
		genTy    func(innerType Type) Type
	}

	someNonReferenceType := VoidType

	tests := []testCase{
		{
			name: "Capability, with type",
			genTy: func(innerType Type) Type {
				return &CapabilityType{
					BorrowType: innerType,
				}
			},
		},
		{
			name:     "Capability, without type",
			ty:       &CapabilityType{},
			expected: false,
		},
		{
			name: "Variable-sized array",
			genTy: func(innerType Type) Type {
				return &VariableSizedType{
					Type: innerType,
				}
			},
		},
		{
			name: "Constant-sized array",
			genTy: func(innerType Type) Type {
				return &ConstantSizedType{
					Type: innerType,
					Size: 42,
				}
			},
		},
		{
			name: "Optional",
			genTy: func(innerType Type) Type {
				return &OptionalType{
					Type: innerType,
				}
			},
		},
		{
			name: "Reference",
			genTy: func(innerType Type) Type {
				return &ReferenceType{
					Type: innerType,
				}
			},
		},
		{
			name: "Dictionary, key",
			genTy: func(innerType Type) Type {
				return &DictionaryType{
					KeyType:   innerType,
					ValueType: someNonReferenceType,
				}
			},
		},
		{
			name: "Dictionary, value",
			genTy: func(innerType Type) Type {
				return &DictionaryType{
					KeyType:   someNonReferenceType,
					ValueType: innerType,
				}
			},
		},
		{
			name:     "Function",
			ty:       &FunctionType{},
			expected: false,
		},
		{
			name:     "Interface",
			ty:       &InterfaceType{},
			expected: false,
		},
		{
			name:     "Composite",
			ty:       &CompositeType{},
			expected: false,
		},
		{
			name: "InclusiveRange",
			genTy: func(innerType Type) Type {
				return &InclusiveRangeType{
					MemberType: innerType,
				}
			},
		},
	}

	test := func(test testCase) {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()

			if test.genTy != nil {

				itself := test.genTy(someNonReferenceType)

				_, ok := itself.(*ReferenceType)
				assert.Equal(t, ok, itself.IsOrContainsReferenceType())

				assert.True(t,
					test.genTy(&ReferenceType{
						Type: someNonReferenceType,
					}).IsOrContainsReferenceType(),
				)
			} else {
				assert.Equal(t,
					test.expected,
					test.ty.IsOrContainsReferenceType(),
				)
			}
		})
	}

	for _, testCase := range tests {
		test(testCase)
	}
}
