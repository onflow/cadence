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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/ast"
)

func TestResources_Add(t *testing.T) {

	t.Parallel()

	resources := NewResources()

	varX := Resource{Variable: &Variable{
		Identifier: "x",
		Type:       IntType,
	}}

	varY := Resource{Variable: &Variable{
		Identifier: "y",
		Type:       IntType,
	}}

	varZ := Resource{Variable: &Variable{
		Identifier: "z",
		Type:       IntType,
	}}

	assert.Empty(t, resources.Get(varX).Invalidations.All())
	assert.Empty(t, resources.Get(varY).Invalidations.All())
	assert.Empty(t, resources.Get(varZ).Invalidations.All())

	// add invalidation for X

	resources.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})

	assert.ElementsMatch(t,
		resources.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
		},
	)
	assert.Empty(t, resources.Get(varY).Invalidations.All())
	assert.Empty(t, resources.Get(varZ).Invalidations.All())

	// add invalidation for X

	resources.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 2, Column: 2},
		EndPos:   ast.Position{Line: 2, Column: 2},
	})

	assert.ElementsMatch(t,
		resources.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 2, Column: 2},
				EndPos:   ast.Position{Line: 2, Column: 2},
			},
		},
	)
	assert.Empty(t, resources.Get(varY).Invalidations.All())
	assert.Empty(t, resources.Get(varZ).Invalidations.All())

	// add invalidation for Y

	resources.AddInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})

	assert.ElementsMatch(t,
		resources.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 2, Column: 2},
				EndPos:   ast.Position{Line: 2, Column: 2},
			},
		},
	)
	assert.ElementsMatch(t,
		resources.Get(varY).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 3, Column: 3},
				EndPos:   ast.Position{Line: 3, Column: 3},
			},
		},
	)
	assert.Empty(t, resources.Get(varZ).Invalidations.All())
}

func TestResourceResources_ForEach(t *testing.T) {

	t.Parallel()

	resources := NewResources()

	varX := Resource{Variable: &Variable{
		Identifier: "x",
		Type:       IntType,
	}}

	varY := Resource{Variable: &Variable{
		Identifier: "y",
		Type:       IntType,
	}}

	// add resources for X and Y

	resources.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})

	resources.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 2, Column: 2},
		EndPos:   ast.Position{Line: 2, Column: 2},
	})

	resources.AddInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})

	result := map[*Variable][]ResourceInvalidation{}

	resources.ForEach(func(resource Resource, info ResourceInfo) {
		variable := resource.Variable
		result[variable] = info.Invalidations.All()
	})

	assert.Len(t, result, 2)

	assert.ElementsMatch(t,
		result[varX.Variable],
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 2, Column: 2},
				EndPos:   ast.Position{Line: 2, Column: 2},
			},
		},
	)

	assert.ElementsMatch(t,
		result[varY.Variable],
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 3, Column: 3},
				EndPos:   ast.Position{Line: 3, Column: 3},
			},
		},
	)
}

func TestResources_MergeBranches(t *testing.T) {

	t.Parallel()

	resourcesThen := NewResources()
	resourcesElse := NewResources()

	varX := Resource{Variable: &Variable{
		Identifier: "x",
		Type:       IntType,
	}}

	varY := Resource{Variable: &Variable{
		Identifier: "y",
		Type:       IntType,
	}}

	varZ := Resource{Variable: &Variable{
		Identifier: "z",
		Type:       IntType,
	}}

	// invalidate X and Y in then branch

	resourcesThen.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})
	resourcesThen.AddInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 2, Column: 2},
		EndPos:   ast.Position{Line: 2, Column: 2},
	})

	// invalidate Y and Z in else branch

	resourcesElse.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})
	resourcesElse.AddInvalidation(varZ, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 4, Column: 4},
		EndPos:   ast.Position{Line: 4, Column: 4},
	})

	// treat var Y already invalidated in main
	resources := NewResources()
	resources.AddInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 0, Column: 0},
		EndPos:   ast.Position{Line: 0, Column: 0},
	})

	resources.MergeBranches(
		resourcesThen,
		resourcesElse,
	)

	varXInfo := resources.Get(varX)
	assert.True(t, varXInfo.DefinitivelyInvalidated)
	assert.ElementsMatch(t,
		varXInfo.Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 3, Column: 3},
				EndPos:   ast.Position{Line: 3, Column: 3},
			},
		},
	)

	varYInfo := resources.Get(varY)
	assert.True(t, varYInfo.DefinitivelyInvalidated)
	assert.ElementsMatch(t,
		varYInfo.Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 0, Column: 0},
				EndPos:   ast.Position{Line: 0, Column: 0},
			},
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 2, Column: 2},
				EndPos:   ast.Position{Line: 2, Column: 2},
			},
		},
	)

	varZInfo := resources.Get(varZ)
	assert.False(t, varZInfo.DefinitivelyInvalidated)
	assert.ElementsMatch(t,
		varZInfo.Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 4, Column: 4},
				EndPos:   ast.Position{Line: 4, Column: 4},
			},
		},
	)
}

func TestResources_Clone(t *testing.T) {

	t.Parallel()

	varX := Resource{Variable: &Variable{Identifier: "x"}}
	varY := Resource{Variable: &Variable{Identifier: "y"}}
	varZ := Resource{Variable: &Variable{Identifier: "z"}}

	// Parent set with only X

	// ... Prepare

	resources := NewResources()

	// add invalidation for X

	resources.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})

	// ... Assert state after

	assert.ElementsMatch(t,
		resources.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
		},
	)
	assert.Empty(t, resources.Get(varY).Invalidations.All())
	assert.Empty(t, resources.Get(varZ).Invalidations.All())

	// Child set with also invalidation for Y

	withXY := resources.Clone()

	// ... Assert state before

	assert.ElementsMatch(t,
		resources.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
		},
	)
	assert.Empty(t, resources.Get(varY).Invalidations.All())
	assert.Empty(t, resources.Get(varZ).Invalidations.All())

	assert.ElementsMatch(t,
		withXY.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
		},
	)
	assert.Empty(t, withXY.Get(varY).Invalidations.All())
	assert.Empty(t, withXY.Get(varZ).Invalidations.All())

	// ... Add invalidation for Y and another for X

	withXY.AddInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 2, Column: 2},
		EndPos:   ast.Position{Line: 2, Column: 2},
	})

	withXY.AddInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})

	// ... Assert state after

	assert.ElementsMatch(t,
		resources.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
		},
	)
	assert.Empty(t, resources.Get(varY).Invalidations.All())
	assert.Empty(t, resources.Get(varZ).Invalidations.All())

	assert.ElementsMatch(t,
		withXY.Get(varX).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 3, Column: 3},
				EndPos:   ast.Position{Line: 3, Column: 3},
			},
		},
	)
	assert.ElementsMatch(t,
		withXY.Get(varY).Invalidations.All(),
		[]ResourceInvalidation{
			{
				Kind:     ResourceInvalidationKindMoveDefinite,
				StartPos: ast.Position{Line: 2, Column: 2},
				EndPos:   ast.Position{Line: 2, Column: 2},
			},
		},
	)
	assert.Empty(t, withXY.Get(varZ).Invalidations.All())
}
