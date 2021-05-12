/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
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
		resourceType.Members.Set(fieldName, NewPublicConstantFieldMember(
			ty.Type,
			fieldName,
			IntType,
			"",
		))

		actualMembers := ty.GetMembers()

		require.Contains(t, actualMembers, fieldName)

		var reportedError error
		actualMember := actualMembers[fieldName].Resolve(fieldName, ast.Range{}, func(err error) {
			reportedError = err
		})

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

		resourceType.Members.Set(fieldName, NewPublicConstantFieldMember(
			restrictedType.Type,
			fieldName,
			IntType,
			"",
		))

		interfaceMember := NewPublicConstantFieldMember(
			restrictedType.Type,
			fieldName,
			IntType,
			"",
		)
		interfaceType.Members.Set(fieldName, interfaceMember)

		actualMembers := restrictedType.GetMembers()

		require.Contains(t, actualMembers, fieldName)

		actualMember := actualMembers[fieldName].Resolve(fieldName, ast.Range{}, nil)

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
