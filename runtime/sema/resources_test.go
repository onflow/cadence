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

	"github.com/onflow/cadence/runtime/ast"
)

func TestResources_MaybeRecordInvalidation(t *testing.T) {

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

	assert.Nil(t, resources.Get(varX).Invalidation())
	assert.Nil(t, resources.Get(varY).Invalidation())
	assert.Nil(t, resources.Get(varZ).Invalidation())

	// record invalidation for X

	resources.MaybeRecordInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})

	assert.Equal(t,
		resources.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)
	assert.Nil(t, resources.Get(varY).Invalidation())
	assert.Nil(t, resources.Get(varZ).Invalidation())

	// record another invalidation for X

	resources.MaybeRecordInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 2, Column: 2},
		EndPos:   ast.Position{Line: 2, Column: 2},
	})

	assert.Equal(t,
		resources.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)
	assert.Nil(t, resources.Get(varY).Invalidation())
	assert.Nil(t, resources.Get(varZ).Invalidation())

	// record invalidation for Y

	resources.MaybeRecordInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})

	assert.Equal(t,
		resources.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)
	assert.Equal(t,
		resources.Get(varY).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 3, Column: 3},
			EndPos:   ast.Position{Line: 3, Column: 3},
		},
	)
	assert.Nil(t, resources.Get(varZ).Invalidation())
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

	// record invalidations for X and Y

	resources.MaybeRecordInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})

	resources.MaybeRecordInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})

	result := map[*Variable]*ResourceInvalidation{}

	resources.ForEach(func(resource Resource, info ResourceInfo) {
		variable := resource.Variable
		result[variable] = info.Invalidation()
	})

	assert.Len(t, result, 2)

	assert.Equal(t,
		result[varX.Variable],
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)

	assert.Equal(t,
		result[varY.Variable],
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 3, Column: 3},
			EndPos:   ast.Position{Line: 3, Column: 3},
		},
	)
}

func TestResources_MergeBranches(t *testing.T) {

	t.Parallel()

	resourcesThen := NewResources()
	resourcesElse := NewResources()

	varX := Resource{
		Variable: &Variable{
			Identifier: "x",
			Type:       IntType,
		},
	}

	varY := Resource{
		Variable: &Variable{
			Identifier: "y",
			Type:       IntType,
		},
	}

	varZ := Resource{
		Variable: &Variable{
			Identifier: "z",
			Type:       IntType,
		},
	}

	// invalidate X and Y in then branch

	resourcesThen.MaybeRecordInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})
	resourcesThen.MaybeRecordInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 2, Column: 2},
		EndPos:   ast.Position{Line: 2, Column: 2},
	})

	// invalidate Y and Z in else branch

	resourcesElse.MaybeRecordInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})
	resourcesElse.MaybeRecordInvalidation(varZ, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 4, Column: 4},
		EndPos:   ast.Position{Line: 4, Column: 4},
	})

	resources := NewResources()
	resources.MergeBranches(
		resourcesThen,
		NewReturnInfo(),
		resourcesElse,
		NewReturnInfo(),
	)

	varXInfo := resources.Get(varX)
	assert.False(t, varXInfo.DefinitivelyInvalidated())
	assert.Equal(t,
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMovePotential,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
		varXInfo.Invalidation(),
	)

	varYInfo := resources.Get(varY)
	assert.True(t, varYInfo.DefinitivelyInvalidated())
	assert.Equal(t,
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 2, Column: 2},
			EndPos:   ast.Position{Line: 2, Column: 2},
		},
		varYInfo.Invalidation(),
	)

	varZInfo := resources.Get(varZ)
	assert.False(t, varZInfo.DefinitivelyInvalidated())
	assert.Equal(t,
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMovePotential,
			StartPos: ast.Position{Line: 4, Column: 4},
			EndPos:   ast.Position{Line: 4, Column: 4},
		},
		varZInfo.Invalidation(),
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

	resources.MaybeRecordInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 1, Column: 1},
		EndPos:   ast.Position{Line: 1, Column: 1},
	})

	// ... Assert state after

	assert.Equal(t,
		resources.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)
	assert.Nil(t, resources.Get(varY).Invalidation())
	assert.Nil(t, resources.Get(varZ).Invalidation())

	// Child set with also invalidation for Y

	withXY := resources.Clone()

	// ... Assert state before

	assert.Equal(t,
		resources.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)
	assert.Nil(t, resources.Get(varY).Invalidation())
	assert.Nil(t, resources.Get(varZ).Invalidation())

	assert.Equal(t,
		withXY.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)
	assert.Nil(t, withXY.Get(varY).Invalidation())
	assert.Nil(t, withXY.Get(varZ).Invalidation())

	// ... Add invalidation for Y and another for X

	withXY.MaybeRecordInvalidation(varY, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 2, Column: 2},
		EndPos:   ast.Position{Line: 2, Column: 2},
	})

	withXY.MaybeRecordInvalidation(varX, ResourceInvalidation{
		Kind:     ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Line: 3, Column: 3},
		EndPos:   ast.Position{Line: 3, Column: 3},
	})

	// ... Assert state after

	assert.Equal(t,
		resources.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 1},
		},
	)
	assert.Nil(t, resources.Get(varY).Invalidation())
	assert.Nil(t, resources.Get(varZ).Invalidation())

	assert.Equal(t,
		withXY.Get(varX).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 3, Column: 3},
			EndPos:   ast.Position{Line: 3, Column: 3},
		},
	)
	assert.Equal(t,
		withXY.Get(varY).Invalidation(),
		&ResourceInvalidation{
			Kind:     ResourceInvalidationKindMoveDefinite,
			StartPos: ast.Position{Line: 2, Column: 2},
			EndPos:   ast.Position{Line: 2, Column: 2},
		},
	)
	assert.Nil(t, withXY.Get(varZ).Invalidation())
}
