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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
)

func TestPrimitiveAccess_QualifiedKeyword(t *testing.T) {

	t.Parallel()

	expected := map[ast.PrimitiveAccess]string{
		ast.AccessNotSpecified:      "",
		ast.AccessSelf:              "access(self)",
		ast.AccessAll:               "access(all)",
		ast.AccessAccount:           "access(account)",
		ast.AccessContract:          "access(contract)",
		ast.AccessPubSettableLegacy: "pub(set)",
		ast.AccessNone:              "inaccessible",
	}

	for access := 0; access < ast.PrimitiveAccessCount(); access++ {
		assert.Equal(t,
			expected[ast.PrimitiveAccess(access)],
			PrimitiveAccess(access).QualifiedKeyword(),
		)
	}
}

func TestNewEntitlementAccess(t *testing.T) {

	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"neither map entitlement nor set entitlements given",
			func() {
				newEntitlementAccess(nil, Conjunction)
			},
		)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"invalid entitlement type: Void",
			func() {
				newEntitlementAccess([]Type{VoidType}, Conjunction)
			},
		)
	})

	t.Run("map + entitlement", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"mixed entitlement types",
			func() {
				newEntitlementAccess(
					[]Type{
						IdentityType,
						MutateType,
					},
					Conjunction,
				)
			},
		)
	})

	t.Run("entitlement + map", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"mixed entitlement types",
			func() {
				newEntitlementAccess(
					[]Type{
						MutateType,
						IdentityType,
					},
					Conjunction,
				)
			},
		)
	})

	t.Run("single entitlement", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			NewEntitlementSetAccess(
				[]*EntitlementType{
					MutateType,
				},
				Conjunction,
			),
			newEntitlementAccess(
				[]Type{
					MutateType,
				},
				Conjunction,
			),
		)
	})

	t.Run("two entitlements, conjunction", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			NewEntitlementSetAccess(
				[]*EntitlementType{
					MutateType,
					InsertType,
				},
				Conjunction,
			),
			newEntitlementAccess(
				[]Type{
					MutateType,
					InsertType,
				},
				Conjunction,
			),
		)
	})

	t.Run("two entitlements, disjunction", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			NewEntitlementSetAccess(
				[]*EntitlementType{
					MutateType,
					InsertType,
				},
				Disjunction,
			),
			newEntitlementAccess(
				[]Type{
					MutateType,
					InsertType,
				},
				Disjunction,
			),
		)
	})

	t.Run("single map", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t,
			NewEntitlementMapAccess(
				IdentityType,
			),
			newEntitlementAccess(
				[]Type{
					IdentityType,
				},
				Conjunction,
			),
		)
	})

	t.Run("two maps", func(t *testing.T) {
		t.Parallel()

		assert.PanicsWithError(t,
			"extra entitlement map type",
			func() {
				newEntitlementAccess(
					[]Type{
						IdentityType,
						AccountMappingType,
					},
					Conjunction,
				)
			},
		)
	})
}

func TestEntitlementSetAccess_QualifiedKeyword(t *testing.T) {

	t.Parallel()

	location := common.NewAddressLocation(nil, common.MustBytesToAddress([]byte{0x1}), "Foo")

	fooType := &CompositeType{
		Location:   location,
		Identifier: "Foo",
	}

	barType := NewEntitlementType(
		nil,
		location,
		"Bar",
	)
	barType.SetContainerType(fooType)

	bazType := NewEntitlementType(
		nil,
		location,
		"Baz",
	)
	bazType.SetContainerType(fooType)

	assert.Equal(t,
		"access(Foo.Bar | Foo.Baz)",
		newEntitlementAccess(
			[]Type{
				barType,
				bazType,
			},
			Disjunction,
		).QualifiedKeyword(),
	)
}

func TestEntitlementMapAccess_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementMapAccess(NewEntitlementMapType(nil, testLocation, "M"))
		assert.Equal(t, TypeID("S.test.M"), access.ID())
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		mapType := NewEntitlementMapType(nil, testLocation, "M")

		mapType.SetContainerType(&CompositeType{
			Location:   testLocation,
			Identifier: "C",
		})

		access := NewEntitlementMapAccess(mapType)
		assert.Equal(t, TypeID("S.test.C.M"), access.ID())
	})

}

func TestEntitlementMapAccess_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	access := NewEntitlementMapAccess(NewEntitlementMapType(nil, testLocation, "M"))
	assert.Equal(t, "M", access.String())
}

func TestEntitlementMapAccess_QualifiedString(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("top-level", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementMapAccess(NewEntitlementMapType(nil, testLocation, "M"))
		assert.Equal(t, "M", access.QualifiedString())
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		mapType := NewEntitlementMapType(nil, testLocation, "M")

		mapType.SetContainerType(&CompositeType{
			Location:   testLocation,
			Identifier: "C",
		})

		access := NewEntitlementMapAccess(mapType)
		assert.Equal(t, "C.M", access.QualifiedString())
	})
}

func TestEntitlementSetAccess_ID(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				NewEntitlementType(nil, testLocation, "E"),
			},
			Conjunction,
		)
		assert.Equal(t,
			TypeID("S.test.E"),
			access.ID(),
		)
	})

	t.Run("two, conjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Conjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.E1,S.test.E2"),
			access.ID(),
		)
	})

	t.Run("two, disjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Disjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.E1|S.test.E2"),
			access.ID(),
		)
	})

	t.Run("three, conjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E3"),
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Conjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.E1,S.test.E2,S.test.E3"),
			access.ID(),
		)
	})

	t.Run("three, disjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E3"),
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Disjunction,
		)
		// NOTE: sorted
		assert.Equal(t,
			TypeID("S.test.E1|S.test.E2|S.test.E3"),
			access.ID(),
		)
	})

}

func TestEntitlementSetAccess_String(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				NewEntitlementType(nil, testLocation, "E"),
			},
			Conjunction,
		)
		assert.Equal(t, "E", access.String())
	})

	t.Run("two, conjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Conjunction,
		)
		// NOTE: order
		assert.Equal(t, "E2, E1", access.String())
	})

	t.Run("two, disjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Disjunction,
		)
		// NOTE: order
		assert.Equal(t, "E2 | E1", access.String())
	})

	t.Run("three, conjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E3"),
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Conjunction,
		)
		// NOTE: order
		assert.Equal(t, "E3, E2, E1", access.String())
	})

	t.Run("three, disjunction", func(t *testing.T) {
		t.Parallel()

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				NewEntitlementType(nil, testLocation, "E3"),
				NewEntitlementType(nil, testLocation, "E2"),
				NewEntitlementType(nil, testLocation, "E1"),
			},
			Disjunction,
		)
		// NOTE: order
		assert.Equal(t, "E3 | E2 | E1", access.String())
	})
}

func TestEntitlementSetAccess_QualifiedString(t *testing.T) {
	t.Parallel()

	testLocation := common.StringLocation("test")

	containerType := &CompositeType{
		Location:   testLocation,
		Identifier: "C",
	}

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		entitlementType := NewEntitlementType(nil, testLocation, "E")
		entitlementType.SetContainerType(containerType)

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				entitlementType,
			},
			Conjunction,
		)
		assert.Equal(t, "C.E", access.QualifiedString())
	})

	t.Run("two, conjunction", func(t *testing.T) {
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
		// NOTE: order
		assert.Equal(t,
			"C.E2, C.E1",
			access.QualifiedString(),
		)
	})

	t.Run("two, disjunction", func(t *testing.T) {
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
			Disjunction,
		)
		// NOTE: order
		assert.Equal(t,
			"C.E2 | C.E1",
			access.QualifiedString(),
		)
	})

	t.Run("three, conjunction", func(t *testing.T) {
		t.Parallel()

		entitlementType1 := NewEntitlementType(nil, testLocation, "E1")
		entitlementType1.SetContainerType(containerType)

		entitlementType2 := NewEntitlementType(nil, testLocation, "E2")
		entitlementType2.SetContainerType(containerType)

		entitlementType3 := NewEntitlementType(nil, testLocation, "E3")
		entitlementType3.SetContainerType(containerType)

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				entitlementType3,
				entitlementType2,
				entitlementType1,
			},
			Conjunction,
		)
		// NOTE: order
		assert.Equal(t,
			"C.E3, C.E2, C.E1",
			access.QualifiedString(),
		)
	})

	t.Run("three, disjunction", func(t *testing.T) {
		t.Parallel()

		entitlementType1 := NewEntitlementType(nil, testLocation, "E1")
		entitlementType1.SetContainerType(containerType)

		entitlementType2 := NewEntitlementType(nil, testLocation, "E2")
		entitlementType2.SetContainerType(containerType)

		entitlementType3 := NewEntitlementType(nil, testLocation, "E3")
		entitlementType3.SetContainerType(containerType)

		access := NewEntitlementSetAccess(
			[]*EntitlementType{
				// NOTE: order
				entitlementType3,
				entitlementType2,
				entitlementType1,
			},
			Disjunction,
		)
		// NOTE: order
		assert.Equal(t,
			"C.E3 | C.E2 | C.E1",
			access.QualifiedString(),
		)
	})
}

func TestIntersectAccess(t *testing.T) {

	t.Parallel()

	conjunctionAccess := func(entitlements ...*EntitlementType) Access {
		return NewEntitlementSetAccess(entitlements, Conjunction)
	}

	disjunctionAccess := func(entitlements ...*EntitlementType) Access {
		return NewEntitlementSetAccess(entitlements, Disjunction)
	}

	t.Run("both unauthorized", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(UnauthorizedAccess, UnauthorizedAccess)
		assert.Equal(t, UnauthorizedAccess, result)
	})

	t.Run("left unauthorized", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(UnauthorizedAccess, conjunctionAccess(MutateType))
		assert.Equal(t, UnauthorizedAccess, result)
	})

	t.Run("right unauthorized", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(conjunctionAccess(MutateType), UnauthorizedAccess)
		assert.Equal(t, UnauthorizedAccess, result)
	})

	t.Run("conjunction, identical", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(
			conjunctionAccess(MutateType, InsertType),
			conjunctionAccess(MutateType, InsertType),
		)
		assert.Equal(t, conjunctionAccess(MutateType, InsertType), result)
	})

	t.Run("conjunction, partial overlap", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(
			conjunctionAccess(MutateType, InsertType),
			conjunctionAccess(InsertType, RemoveType),
		)
		assert.Equal(t, conjunctionAccess(InsertType), result)
	})

	t.Run("conjunction, disjoint", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(
			conjunctionAccess(MutateType),
			conjunctionAccess(InsertType),
		)
		assert.Equal(t, UnauthorizedAccess, result)
	})

	t.Run("disjunction, identical", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(
			disjunctionAccess(MutateType, InsertType),
			disjunctionAccess(MutateType, InsertType),
		)
		assert.Equal(t, disjunctionAccess(MutateType, InsertType), result)
	})

	t.Run("disjunction, partial overlap", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(
			disjunctionAccess(MutateType, InsertType),
			disjunctionAccess(InsertType, RemoveType),
		)
		assert.Equal(t, disjunctionAccess(InsertType), result)
	})

	t.Run("disjunction, disjoint", func(t *testing.T) {
		t.Parallel()

		result := IntersectAccess(
			disjunctionAccess(MutateType),
			disjunctionAccess(InsertType),
		)
		assert.Equal(t, UnauthorizedAccess, result)
	})

	t.Run("mixed, conjunction left, disjunction right", func(t *testing.T) {
		t.Parallel()

		// Conjunction has all of {Mutate, Insert}.
		// Disjunction has one of {Insert, Remove}.
		// Intersection: {Insert}. Result must be disjunction
		// because the disjunction side only guarantees one.
		result := IntersectAccess(
			conjunctionAccess(MutateType, InsertType),
			disjunctionAccess(InsertType, RemoveType),
		)
		assert.Equal(t, disjunctionAccess(InsertType), result)
	})

	t.Run("mixed, disjunction left, conjunction right", func(t *testing.T) {
		t.Parallel()

		// Disjunction has one of {Mutate, Insert}.
		// Conjunction has all of {Insert, Remove}.
		// Intersection: {Insert}. Result must be disjunction.
		result := IntersectAccess(
			disjunctionAccess(MutateType, InsertType),
			conjunctionAccess(InsertType, RemoveType),
		)
		assert.Equal(t, disjunctionAccess(InsertType), result)
	})

	t.Run("mixed, full overlap, disjunction constrains result", func(t *testing.T) {
		t.Parallel()

		// Disjunction has one of {Mutate, Insert}.
		// Conjunction has all of {Mutate, Insert}.
		// Intersection: {Mutate, Insert}. Still disjunction —
		// can't upgrade to conjunction just because the sets match.
		result := IntersectAccess(
			disjunctionAccess(MutateType, InsertType),
			conjunctionAccess(MutateType, InsertType),
		)
		assert.Equal(t, disjunctionAccess(MutateType, InsertType), result)
	})

	t.Run("entitlement map access", func(t *testing.T) {
		t.Parallel()

		mapAccess := NewEntitlementMapAccess(IdentityType)
		result := IntersectAccess(mapAccess, conjunctionAccess(MutateType))
		assert.Equal(t, UnauthorizedAccess, result)
	})
}

func TestEntitlementMapAccess_QualifiedKeyword(t *testing.T) {

	t.Parallel()

	location := common.NewAddressLocation(nil, common.MustBytesToAddress([]byte{0x1}), "Foo")

	fooType := &CompositeType{
		Location:   location,
		Identifier: "Foo",
	}

	barType := NewEntitlementMapType(
		nil,
		location,
		"Bar",
	)
	barType.SetContainerType(fooType)

	assert.Equal(t,
		"access(mapping Foo.Bar)",
		NewEntitlementMapAccess(barType).QualifiedKeyword(),
	)
}
