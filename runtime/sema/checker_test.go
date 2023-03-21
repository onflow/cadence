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

package sema

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common"
)

func TestOptionalSubtyping(t *testing.T) {

	t.Parallel()

	t.Run("Int? <: Int?", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&OptionalType{Type: IntType},
				&OptionalType{Type: IntType},
			),
		)
	})

	t.Run("Int? <: Bool?", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&OptionalType{Type: IntType},
				&OptionalType{Type: BoolType},
			),
		)
	})

	t.Run("Int8? <: Integer?", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&OptionalType{Type: Int8Type},
				&OptionalType{Type: IntegerType},
			),
		)
	})
}

func TestCompositeType_ID(t *testing.T) {

	t.Parallel()

	location := common.StringLocation("x")

	t.Run("composite in composite", func(t *testing.T) {

		compositeInComposite :=
			&CompositeType{
				Location:   location,
				Identifier: "C",
				containerType: &CompositeType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID("S.x.A.B.C"),
			compositeInComposite.ID(),
		)
	})

	t.Run("composite in interface", func(t *testing.T) {

		compositeInInterface :=
			&CompositeType{
				Location:   location,
				Identifier: "C",
				containerType: &InterfaceType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID("S.x.A.B.C"),
			compositeInInterface.ID(),
		)
	})
}

func TestInterfaceType_ID(t *testing.T) {

	t.Parallel()

	location := common.StringLocation("x")

	t.Run("interface in composite", func(t *testing.T) {

		interfaceInComposite :=
			&InterfaceType{
				Location:   location,
				Identifier: "C",
				containerType: &CompositeType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID("S.x.A.B.C"),
			interfaceInComposite.ID(),
		)
	})

	t.Run("interface in interface", func(t *testing.T) {

		interfaceInInterface :=
			&InterfaceType{
				Location:   location,
				Identifier: "C",
				containerType: &InterfaceType{
					Location:   location,
					Identifier: "B",
					containerType: &CompositeType{
						Location:   location,
						Identifier: "A",
					},
				},
			}

		assert.Equal(t,
			TypeID("S.x.A.B.C"),
			interfaceInInterface.ID(),
		)
	})
}

func TestFunctionSubtyping(t *testing.T) {

	t.Parallel()

	t.Run("fun(Int): Void <: fun(AnyStruct): Void", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&FunctionType{
					Parameters: []Parameter{
						{
							TypeAnnotation: IntTypeAnnotation,
						},
					},
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
				&FunctionType{
					Parameters: []Parameter{
						{
							TypeAnnotation: AnyStructTypeAnnotation,
						},
					},
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
			),
		)
	})

	t.Run("fun(AnyStruct): Void <: fun(Int): Void", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&FunctionType{
					Parameters: []Parameter{
						{
							TypeAnnotation: AnyStructTypeAnnotation,
						},
					},
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
				&FunctionType{
					Parameters: []Parameter{
						{
							TypeAnnotation: IntTypeAnnotation,
						},
					},
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
			),
		)
	})

	t.Run("fun(): Int <: fun(): AnyStruct", func(t *testing.T) {
		assert.True(t,
			IsSubType(
				&FunctionType{
					ReturnTypeAnnotation: IntTypeAnnotation,
				},
				&FunctionType{
					ReturnTypeAnnotation: AnyStructTypeAnnotation,
				},
			),
		)
	})

	t.Run("fun(): Any <: fun(): Int", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&FunctionType{
					ReturnTypeAnnotation: AnyStructTypeAnnotation,
				},
				&FunctionType{
					ReturnTypeAnnotation: IntTypeAnnotation,
				},
			),
		)
	})

	t.Run("constructor != non-constructor", func(t *testing.T) {
		assert.False(t,
			IsSubType(
				&FunctionType{
					IsConstructor:        false,
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
				&FunctionType{
					IsConstructor:        true,
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
			),
		)
	})

	t.Run("different receiver types", func(t *testing.T) {
		// Receiver shouldn't matter
		assert.True(t,
			IsSubType(
				&FunctionType{
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
				&FunctionType{
					ReturnTypeAnnotation: VoidTypeAnnotation,
				},
			),
		)
	})
}

func TestReferenceSubtyping(t *testing.T) {

	t.Parallel()

	testLocation := common.StringLocation("test")

	intRef := func(access Access) *ReferenceType {
		return &ReferenceType{
			Authorization: access,
			Type:          IntType,
		}
	}

	anyStructRef := func(access Access) *ReferenceType {
		return &ReferenceType{
			Authorization: access,
			Type:          AnyStructType,
		}
	}

	anyResourceRef := func(access Access) *ReferenceType {
		return &ReferenceType{
			Authorization: access,
			Type:          AnyResourceType,
		}
	}

	mapAccess := EntitlementMapAccess{
		Type: &EntitlementMapType{
			Location:   testLocation,
			Identifier: "M",
		},
	}

	containedMapAccess := EntitlementMapAccess{
		Type: &EntitlementMapType{
			Location: testLocation,
			containerType: &InterfaceType{
				Location:   testLocation,
				Identifier: "C",
			},
			Identifier: "M",
		},
	}

	x := &EntitlementType{
		Location:   testLocation,
		Identifier: "X",
	}

	y := &EntitlementType{
		Location:   testLocation,
		Identifier: "Y",
	}

	z := &EntitlementType{
		Location:   testLocation,
		Identifier: "Z",
	}

	cx := &EntitlementType{
		Location: testLocation,
		containerType: &InterfaceType{
			Location:   testLocation,
			Identifier: "C",
		},
		Identifier: "X",
	}

	xyzConjunction := NewEntitlementSetAccess([]*EntitlementType{x, y, z}, Conjunction)

	xyConjunction := NewEntitlementSetAccess([]*EntitlementType{x, y}, Conjunction)
	xzConjunction := NewEntitlementSetAccess([]*EntitlementType{x, z}, Conjunction)
	yzConjunction := NewEntitlementSetAccess([]*EntitlementType{y, z}, Conjunction)

	xConjunction := NewEntitlementSetAccess([]*EntitlementType{x}, Conjunction)
	yConjunction := NewEntitlementSetAccess([]*EntitlementType{y}, Conjunction)
	zConjunction := NewEntitlementSetAccess([]*EntitlementType{z}, Conjunction)
	cxConjunction := NewEntitlementSetAccess([]*EntitlementType{cx}, Conjunction)

	xyzDisjunction := NewEntitlementSetAccess([]*EntitlementType{x, y, z}, Disjunction)
	xyDisjunction := NewEntitlementSetAccess([]*EntitlementType{x, y}, Disjunction)
	xzDisjunction := NewEntitlementSetAccess([]*EntitlementType{x, z}, Disjunction)
	yzDisjunction := NewEntitlementSetAccess([]*EntitlementType{y, z}, Disjunction)

	test := func(result bool, subType, superType Type) {
		t.Run(fmt.Sprintf("%s <: %s", subType.QualifiedString(), superType.QualifiedString()), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, result, IsSubType(subType, superType))
		})
	}

	tests := []struct {
		subType   Type
		superType Type
		result    bool
	}{
		{intRef(UnauthorizedAccess), AnyStructType, true},
		{intRef(UnauthorizedAccess), AnyResourceType, false},
		{anyStructRef(UnauthorizedAccess), AnyStructType, true},
		{anyStructRef(UnauthorizedAccess), AnyResourceType, false},
		{anyResourceRef(UnauthorizedAccess), AnyStructType, true},
		{anyResourceRef(UnauthorizedAccess), AnyResourceType, false},

		{AnyStructType, intRef(UnauthorizedAccess), false},
		{AnyResourceType, intRef(UnauthorizedAccess), false},

		{intRef(UnauthorizedAccess), intRef(UnauthorizedAccess), true},
		{anyStructRef(UnauthorizedAccess), anyStructRef(UnauthorizedAccess), true},
		{anyResourceRef(UnauthorizedAccess), anyResourceRef(UnauthorizedAccess), true},
		{intRef(UnauthorizedAccess), anyStructRef(UnauthorizedAccess), true},
		{anyStructRef(UnauthorizedAccess), intRef(UnauthorizedAccess), false},
		{intRef(UnauthorizedAccess), anyResourceRef(UnauthorizedAccess), false},
		{anyResourceRef(UnauthorizedAccess), anyStructRef(UnauthorizedAccess), false},

		{intRef(UnauthorizedAccess), intRef(mapAccess), false},
		{intRef(UnauthorizedAccess), intRef(containedMapAccess), false},
		{intRef(UnauthorizedAccess), intRef(xyzConjunction), false},
		{intRef(UnauthorizedAccess), intRef(xyzDisjunction), false},

		{intRef(UnauthorizedAccess), anyStructRef(mapAccess), false},
		{intRef(UnauthorizedAccess), anyStructRef(containedMapAccess), false},
		{intRef(UnauthorizedAccess), anyStructRef(xyzConjunction), false},
		{intRef(UnauthorizedAccess), anyStructRef(xyzDisjunction), false},

		{intRef(UnauthorizedAccess), anyResourceRef(mapAccess), false},
		{intRef(UnauthorizedAccess), anyResourceRef(containedMapAccess), false},
		{intRef(UnauthorizedAccess), anyResourceRef(xyzConjunction), false},
		{intRef(UnauthorizedAccess), anyResourceRef(xyzDisjunction), false},

		{intRef(mapAccess), intRef(UnauthorizedAccess), true},
		{intRef(mapAccess), intRef(mapAccess), true},
		{intRef(mapAccess), intRef(containedMapAccess), false},
		{intRef(mapAccess), intRef(xyzConjunction), false},
		{intRef(mapAccess), intRef(xyzDisjunction), false},

		{intRef(mapAccess), anyStructRef(UnauthorizedAccess), true},
		{intRef(mapAccess), anyStructRef(mapAccess), true},
		{intRef(mapAccess), anyStructRef(containedMapAccess), false},
		{intRef(mapAccess), anyStructRef(xyzConjunction), false},
		{intRef(mapAccess), anyStructRef(xyzDisjunction), false},

		{intRef(containedMapAccess), intRef(UnauthorizedAccess), true},
		{intRef(containedMapAccess), intRef(containedMapAccess), true},
		{intRef(containedMapAccess), intRef(mapAccess), false},
		{intRef(containedMapAccess), intRef(xyzConjunction), false},
		{intRef(containedMapAccess), intRef(xyzDisjunction), false},

		{intRef(containedMapAccess), anyStructRef(UnauthorizedAccess), true},
		{intRef(containedMapAccess), anyStructRef(containedMapAccess), true},
		{intRef(containedMapAccess), anyStructRef(mapAccess), false},
		{intRef(containedMapAccess), anyStructRef(xyzConjunction), false},
		{intRef(containedMapAccess), anyStructRef(xyzDisjunction), false},

		{intRef(xyzConjunction), intRef(UnauthorizedAccess), true},
		{intRef(xyzConjunction), intRef(containedMapAccess), false},
		{intRef(xyzConjunction), intRef(mapAccess), false},
		{intRef(xyzConjunction), intRef(xyzConjunction), true},
		{intRef(xyzConjunction), intRef(xyzDisjunction), true},
		{intRef(xyzConjunction), intRef(xConjunction), true},
		{intRef(xyzConjunction), intRef(xyConjunction), true},
		{intRef(xyzConjunction), intRef(xyDisjunction), true},

		{intRef(xyzConjunction), anyStructRef(UnauthorizedAccess), true},
		{intRef(xyzConjunction), anyStructRef(containedMapAccess), false},
		{intRef(xyzConjunction), anyStructRef(mapAccess), false},
		{intRef(xyzConjunction), anyStructRef(xyzConjunction), true},
		{intRef(xyzConjunction), anyStructRef(xyzDisjunction), true},
		{intRef(xyzConjunction), anyStructRef(xConjunction), true},
		{intRef(xyzConjunction), anyStructRef(xyConjunction), true},
		{intRef(xyzConjunction), anyStructRef(xyDisjunction), true},

		{intRef(xyzDisjunction), intRef(UnauthorizedAccess), true},
		{intRef(xyzDisjunction), intRef(containedMapAccess), false},
		{intRef(xyzDisjunction), intRef(mapAccess), false},
		{intRef(xyzDisjunction), intRef(xyzConjunction), false},
		{intRef(xyzDisjunction), intRef(xyzDisjunction), true},
		{intRef(xyzDisjunction), intRef(xConjunction), false},
		{intRef(xyzDisjunction), intRef(xyConjunction), false},
		{intRef(xyzDisjunction), intRef(xyDisjunction), false},

		{intRef(xyzDisjunction), anyStructRef(UnauthorizedAccess), true},
		{intRef(xyzDisjunction), anyStructRef(containedMapAccess), false},
		{intRef(xyzDisjunction), anyStructRef(mapAccess), false},
		{intRef(xyzDisjunction), anyStructRef(xyzConjunction), false},
		{intRef(xyzDisjunction), anyStructRef(xyzDisjunction), true},
		{intRef(xyzDisjunction), anyStructRef(xConjunction), false},
		{intRef(xyzDisjunction), anyStructRef(xyConjunction), false},
		{intRef(xyzDisjunction), anyStructRef(xyDisjunction), false},

		{intRef(xConjunction), intRef(yConjunction), false},
		{intRef(xConjunction), intRef(zConjunction), false},
		{intRef(xConjunction), intRef(cxConjunction), false},
		{intRef(xConjunction), intRef(xzConjunction), false},
		{intRef(xConjunction), intRef(xyzConjunction), false},
		{intRef(xConjunction), intRef(xConjunction), true},
		{intRef(xConjunction), intRef(xyDisjunction), true},
		{intRef(xConjunction), intRef(xzDisjunction), true},
		{intRef(xConjunction), intRef(yzDisjunction), false},
		{intRef(xConjunction), intRef(xyzDisjunction), true},

		{intRef(xzConjunction), intRef(xConjunction), true},
		{intRef(xzConjunction), intRef(cxConjunction), false},
		{intRef(xzConjunction), intRef(zConjunction), true},
		{intRef(xzConjunction), intRef(yConjunction), false},
		{intRef(xzConjunction), intRef(xyConjunction), false},
		{intRef(xzConjunction), intRef(xzConjunction), true},
		{intRef(xzConjunction), intRef(yzConjunction), false},
		{intRef(xzConjunction), intRef(xyDisjunction), true},
		{intRef(xzConjunction), intRef(xzDisjunction), true},
		{intRef(xzConjunction), intRef(yzDisjunction), true},
		{intRef(xzConjunction), intRef(xyzDisjunction), true},
		{intRef(xzConjunction), intRef(xyzConjunction), false},

		{intRef(xzDisjunction), intRef(xConjunction), false},
		{intRef(xzDisjunction), intRef(cxConjunction), false},
		{intRef(xzDisjunction), intRef(zConjunction), false},
		{intRef(xzDisjunction), intRef(yConjunction), false},
		{intRef(xzDisjunction), intRef(xyConjunction), false},
		{intRef(xzDisjunction), intRef(xzConjunction), false},
		{intRef(xzDisjunction), intRef(yzConjunction), false},
		{intRef(xzDisjunction), intRef(xyDisjunction), false},
		{intRef(xzDisjunction), intRef(xzDisjunction), true},
		{intRef(xzDisjunction), intRef(yzDisjunction), false},
		{intRef(xzDisjunction), intRef(xyzDisjunction), true},
		{intRef(xzDisjunction), intRef(xyzConjunction), false},
	}

	for _, t := range tests {
		test(t.result, t.subType, t.superType)
	}
}
