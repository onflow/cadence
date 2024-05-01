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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common/orderedmap"
)

func TestEntitlementSet_Add(t *testing.T) {
	t.Parallel()

	t.Run("no existing disjunctions", func(t *testing.T) {

		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		set.Add(e1)

		assert.False(t, set.minimized)
		assert.Equal(t, 1, set.Entitlements.Len())
		assert.Nil(t, set.Disjunctions)

		e2 := &EntitlementType{
			Identifier: "E2",
		}
		set.Add(e2)

		assert.False(t, set.minimized)
		assert.Equal(t, 2, set.Entitlements.Len())
		assert.Nil(t, set.Disjunctions)
	})

	t.Run("with existing disjunctions", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		set.AddDisjunction(e1e2)

		assert.False(t, set.minimized)
		assert.Nil(t, set.Entitlements)
		assert.Equal(t, 1, set.Disjunctions.Len())

		// Add

		set.Add(e2)

		assert.False(t, set.minimized)
		assert.Equal(t, 1, set.Entitlements.Len())
		// NOTE: the set is not minimal,
		// the disjunction is not discarded
		assert.Equal(t, 1, set.Disjunctions.Len())

	})
}

func TestEntitlementSet_AddDisjunction(t *testing.T) {
	t.Parallel()

	t.Run("no existing entitlements", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		// Add

		set.AddDisjunction(e1e2)

		assert.False(t, set.minimized)
		assert.Nil(t, set.Entitlements)
		assert.Equal(t, 1, set.Disjunctions.Len())

		// Re-add same

		set.AddDisjunction(e1e2)

		assert.False(t, set.minimized)
		assert.Nil(t, set.Entitlements)
		assert.Equal(t, 1, set.Disjunctions.Len())

		// Re-add equal with different order

		e2e1 := orderedmap.New[EntitlementOrderedSet](2)
		e2e1.Set(e2, struct{}{})
		e2e1.Set(e1, struct{}{})

		set.AddDisjunction(e2e1)

		assert.False(t, set.minimized)
		assert.Nil(t, set.Entitlements)
		assert.Equal(t, 1, set.Disjunctions.Len())

		// Re-add different, with partial overlap

		e3 := &EntitlementType{
			Identifier: "E3",
		}

		e2e3 := orderedmap.New[EntitlementOrderedSet](2)
		e2e3.Set(e2, struct{}{})
		e2e3.Set(e3, struct{}{})

		set.AddDisjunction(e2e3)

		assert.False(t, set.minimized)
		assert.Nil(t, set.Entitlements)
		assert.Equal(t, 2, set.Disjunctions.Len())
	})

	t.Run("with existing entitlements", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}

		set.Add(e1)

		assert.False(t, set.minimized)
		assert.Equal(t, 1, set.Entitlements.Len())
		assert.Nil(t, set.Disjunctions)

		// Add disjunction with overlap

		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		set.AddDisjunction(e1e2)

		assert.False(t, set.minimized)
		assert.Equal(t, 1, set.Entitlements.Len())
		assert.Nil(t, set.Disjunctions)
	})
}

func TestEntitlementSet_Merge(t *testing.T) {
	t.Parallel()

	e1 := &EntitlementType{
		Identifier: "E1",
	}
	e2 := &EntitlementType{
		Identifier: "E2",
	}
	e3 := &EntitlementType{
		Identifier: "E3",
	}
	e4 := &EntitlementType{
		Identifier: "E4",
	}

	e2e3 := orderedmap.New[EntitlementOrderedSet](2)
	e2e3.Set(e2, struct{}{})
	e2e3.Set(e3, struct{}{})

	e3e4 := orderedmap.New[EntitlementOrderedSet](2)
	e3e4.Set(e3, struct{}{})
	e3e4.Set(e4, struct{}{})

	// Prepare set 1

	set1 := &EntitlementSet{}
	set1.Add(e1)
	set1.AddDisjunction(e2e3)

	assert.False(t, set1.minimized)
	assert.Equal(t, 1, set1.Entitlements.Len())
	assert.Equal(t, 1, set1.Disjunctions.Len())

	// Prepare set 2

	set2 := &EntitlementSet{}
	set2.Add(e2)
	set2.AddDisjunction(e3e4)

	assert.False(t, set2.minimized)
	assert.Equal(t, 1, set2.Entitlements.Len())
	assert.Equal(t, 1, set2.Disjunctions.Len())

	// Merge

	set1.Merge(set2)

	assert.False(t, set1.minimized)
	assert.Equal(t, 2, set1.Entitlements.Len())
	assert.True(t, set1.Entitlements.Contains(e1))
	assert.True(t, set1.Entitlements.Contains(e2))

	// NOTE: the result is not minimal
	assert.Equal(t, 2, set1.Disjunctions.Len())
	assert.True(t, set1.Disjunctions.Contains(disjunctionKey(e2e3)))
	assert.True(t, set1.Disjunctions.Contains(disjunctionKey(e3e4)))
}

func TestEntitlementSet_Minimize(t *testing.T) {
	t.Parallel()

	e1 := &EntitlementType{
		Identifier: "E1",
	}
	e2 := &EntitlementType{
		Identifier: "E2",
	}

	e1e2 := orderedmap.New[EntitlementOrderedSet](2)
	e1e2.Set(e1, struct{}{})
	e1e2.Set(e2, struct{}{})

	set := &EntitlementSet{}
	set.AddDisjunction(e1e2)

	assert.Nil(t, set.Entitlements)
	assert.Equal(t, 1, set.Disjunctions.Len())

	// Add entitlement

	set.Add(e1)

	// NOTE: the set is not minimal
	assert.False(t, set.minimized)
	assert.Equal(t, 1, set.Entitlements.Len())
	assert.Equal(t, 1, set.Disjunctions.Len())

	// Minimize

	set.Minimize()

	assert.True(t, set.minimized)
	assert.Equal(t, 1, set.Entitlements.Len())
	assert.Equal(t, 0, set.Disjunctions.Len())
}

func TestEntitlementSet_MinimallyRepresentable(t *testing.T) {
	t.Parallel()

	t.Run("no entitlements, no disjunctions", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}
		assert.True(t, set.IsMinimallyRepresentable())
	})

	t.Run("one entitlement, no disjunctions", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		set.Add(e1)

		assert.True(t, set.IsMinimallyRepresentable())
	})

	t.Run("two entitlements, no disjunctions", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		set.Add(e1)

		e2 := &EntitlementType{
			Identifier: "E2",
		}
		set.Add(e2)

		assert.True(t, set.IsMinimallyRepresentable())
	})

	t.Run("one entitlement, redundant disjunction", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}

		e2 := &EntitlementType{
			Identifier: "E2",
		}

		set.Add(e1)

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		set.AddDisjunction(e1e2)

		assert.True(t, set.IsMinimallyRepresentable())
	})

	t.Run("two entitlements, two redundant disjunctions", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}

		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e3 := &EntitlementType{
			Identifier: "E1",
		}

		e4 := &EntitlementType{
			Identifier: "E2",
		}

		set.Add(e1)
		set.Add(e3)

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		set.AddDisjunction(e1e2)

		e3e4 := orderedmap.New[EntitlementOrderedSet](2)
		e3e4.Set(e3, struct{}{})
		e3e4.Set(e4, struct{}{})

		set.AddDisjunction(e3e4)

		assert.True(t, set.IsMinimallyRepresentable())
	})

	t.Run("one entitlement, non-redundant disjunction", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}

		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e3 := &EntitlementType{
			Identifier: "E3",
		}

		set.Add(e1)

		e3e2 := orderedmap.New[EntitlementOrderedSet](2)
		e3e2.Set(e3, struct{}{})
		e3e2.Set(e2, struct{}{})

		set.AddDisjunction(e3e2)

		assert.False(t, set.IsMinimallyRepresentable())
	})

	t.Run("two disjunctions", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}

		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e3 := &EntitlementType{
			Identifier: "E3",
		}

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		set.AddDisjunction(e1e2)

		e3e2 := orderedmap.New[EntitlementOrderedSet](2)
		e3e2.Set(e3, struct{}{})
		e3e2.Set(e2, struct{}{})

		set.AddDisjunction(e3e2)

		assert.False(t, set.IsMinimallyRepresentable())
	})
}

func TestEntitlementSet_Access(t *testing.T) {
	t.Parallel()

	t.Run("no entitlements, no disjunctions: {} = unauthorized", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		access := set.Access()

		assert.True(t, set.minimized)
		assert.Equal(t, UnauthorizedAccess, access)
	})

	t.Run("entitlements, no disjunctions: {E1, E2} = auth(E1, E2)", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		set.Add(e1)

		e2 := &EntitlementType{
			Identifier: "E2",
		}
		set.Add(e2)

		assert.False(t, set.minimized)
		access := set.Access()
		assert.True(t, set.minimized)

		expectedEntitlements := orderedmap.New[EntitlementOrderedSet](2)
		expectedEntitlements.Set(e1, struct{}{})
		expectedEntitlements.Set(e2, struct{}{})

		assert.Equal(t,
			EntitlementSetAccess{
				Entitlements: expectedEntitlements,
				SetKind:      Conjunction,
			},
			access,
		)
	})

	t.Run("no entitlements, one disjunction: {(E1 | E2)} = auth(E1 | E2)", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		set.AddDisjunction(e1e2)

		assert.False(t, set.minimized)
		access := set.Access()
		assert.True(t, set.minimized)

		assert.Equal(t,
			EntitlementSetAccess{
				Entitlements: e1e2,
				SetKind:      Disjunction,
			},
			access,
		)
	})

	t.Run("no entitlements, two disjunctions: {(E1 | E2), (E2 | E3)} = auth(E1, E2, E3)", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		e2 := &EntitlementType{
			Identifier: "E2",
		}
		e3 := &EntitlementType{
			Identifier: "E3",
		}

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		e2e3 := orderedmap.New[EntitlementOrderedSet](2)
		e2e3.Set(e2, struct{}{})
		e2e3.Set(e3, struct{}{})

		set.AddDisjunction(e1e2)

		assert.False(t, set.minimized)
		set.Minimize()
		assert.True(t, set.minimized)

		set.AddDisjunction(e2e3)
		assert.False(t, set.minimized)

		access := set.Access()
		assert.True(t, set.minimized)

		// Cannot express (E1 | E2), (E2 | E3) in an access/auth,
		// so the result is the conjunction of all entitlements
		assert.False(t, set.IsMinimallyRepresentable())

		expectedEntitlements := orderedmap.New[EntitlementOrderedSet](3)
		expectedEntitlements.Set(e1, struct{}{})
		expectedEntitlements.Set(e2, struct{}{})
		expectedEntitlements.Set(e3, struct{}{})

		assert.Equal(t,
			EntitlementSetAccess{
				Entitlements: expectedEntitlements,
				SetKind:      Conjunction,
			},
			access,
		)
	})

	t.Run("entitlement, one disjunction, not minimal: {E1, (E2 | E3)} = auth(E1, E2, E3)", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		e2 := &EntitlementType{
			Identifier: "E2",
		}
		e3 := &EntitlementType{
			Identifier: "E3",
		}

		set.Add(e1)

		e2e3 := orderedmap.New[EntitlementOrderedSet](2)
		e2e3.Set(e2, struct{}{})
		e2e3.Set(e3, struct{}{})

		set.AddDisjunction(e2e3)
		assert.False(t, set.minimized)

		access := set.Access()
		assert.True(t, set.minimized)

		// Cannot express E1, (E2 | E3) in an access/auth,
		// so the result is the conjunction of all entitlements
		assert.False(t, set.IsMinimallyRepresentable())

		expectedEntitlements := orderedmap.New[EntitlementOrderedSet](3)
		expectedEntitlements.Set(e1, struct{}{})
		expectedEntitlements.Set(e2, struct{}{})
		expectedEntitlements.Set(e3, struct{}{})

		assert.Equal(t,
			EntitlementSetAccess{
				Entitlements: expectedEntitlements,
				SetKind:      Conjunction,
			},
			access,
		)
	})

	t.Run("entitlement, one disjunction, minimal: {(E1 | E2), E1} = auth(E1)", func(t *testing.T) {
		t.Parallel()

		set := &EntitlementSet{}

		e1 := &EntitlementType{
			Identifier: "E1",
		}
		e2 := &EntitlementType{
			Identifier: "E2",
		}

		e1e2 := orderedmap.New[EntitlementOrderedSet](2)
		e1e2.Set(e1, struct{}{})
		e1e2.Set(e2, struct{}{})

		set.AddDisjunction(e1e2)
		assert.False(t, set.minimized)

		set.Minimize()
		assert.True(t, set.minimized)

		set.Add(e1)
		assert.False(t, set.minimized)

		access := set.Access()
		assert.True(t, set.minimized)

		// NOTE: disjunction got removed during minimization
		assert.True(t, set.IsMinimallyRepresentable())

		expectedEntitlements := orderedmap.New[EntitlementOrderedSet](1)
		expectedEntitlements.Set(e1, struct{}{})

		assert.Equal(t,
			EntitlementSetAccess{
				Entitlements: expectedEntitlements,
				SetKind:      Conjunction,
			},
			access,
		)
	})
}
