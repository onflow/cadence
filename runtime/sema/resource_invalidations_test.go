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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

func TestResourceInvalidations(t *testing.T) {

	t.Parallel()

	invalidationA := sema.ResourceInvalidation{
		Kind:     sema.ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Offset: 1},
		EndPos:   ast.Position{Offset: 2},
	}
	invalidationB := sema.ResourceInvalidation{
		Kind:     sema.ResourceInvalidationKindMoveTemporary,
		StartPos: ast.Position{Offset: 3},
		EndPos:   ast.Position{Offset: 4},
	}
	invalidationC := sema.ResourceInvalidation{
		Kind:     sema.ResourceInvalidationKindDestroy,
		StartPos: ast.Position{Offset: 5},
		EndPos:   ast.Position{Offset: 6},
	}

	// Parent set with only A

	// ... Prepare

	resourceInvalidations := &sema.ResourceInvalidations{}

	resourceInvalidations.Add(invalidationA)

	// ... Assert state after

	assert.True(t, resourceInvalidations.Contains(invalidationA))
	assert.False(t, resourceInvalidations.Contains(invalidationB))
	assert.False(t, resourceInvalidations.Contains(invalidationC))

	assert.Equal(t, 1, resourceInvalidations.Size())

	var forEachResult []sema.ResourceInvalidation

	err := resourceInvalidations.ForEach(func(invalidation sema.ResourceInvalidation) error {
		forEachResult = append(forEachResult, invalidation)
		return nil
	})
	assert.NoError(t, err)
	expected := []sema.ResourceInvalidation{
		invalidationA,
	}
	assert.Equal(t,
		expected,
		forEachResult,
	)
	assert.Equal(t,
		expected,
		resourceInvalidations.All(),
	)

	// Child set with also B

	withB := resourceInvalidations.Clone()
	resourceInvalidations = &withB

	// ... Assert state before

	assert.True(t, resourceInvalidations.Contains(invalidationA))
	assert.False(t, resourceInvalidations.Contains(invalidationB))
	assert.False(t, resourceInvalidations.Contains(invalidationC))

	assert.Equal(t, 1, resourceInvalidations.Size())

	forEachResult = nil

	err = resourceInvalidations.ForEach(func(invalidation sema.ResourceInvalidation) error {
		forEachResult = append(forEachResult, invalidation)
		return nil
	})
	assert.NoError(t, err)
	expected = []sema.ResourceInvalidation{
		invalidationA,
	}
	assert.Equal(t,
		expected,
		forEachResult,
	)
	assert.Equal(t,
		expected,
		resourceInvalidations.All(),
	)

	// ... Add B

	resourceInvalidations.Add(invalidationB)

	// ... Assert state after

	assert.True(t, resourceInvalidations.Contains(invalidationA))
	assert.True(t, resourceInvalidations.Contains(invalidationB))
	assert.False(t, resourceInvalidations.Contains(invalidationC))

	assert.Equal(t, 2, resourceInvalidations.Size())

	forEachResult = nil

	err = resourceInvalidations.ForEach(func(invalidation sema.ResourceInvalidation) error {
		forEachResult = append(forEachResult, invalidation)
		return nil
	})
	assert.NoError(t, err)
	expected = []sema.ResourceInvalidation{
		invalidationB,
		invalidationA,
	}
	assert.Equal(t,
		expected,
		forEachResult,
	)
	assert.Equal(t,
		expected,
		resourceInvalidations.All(),
	)

	// Child set with also C

	withC := resourceInvalidations.Clone()
	resourceInvalidations = &withC

	// ... Assert state before

	assert.True(t, resourceInvalidations.Contains(invalidationA))
	assert.True(t, resourceInvalidations.Contains(invalidationB))
	assert.False(t, resourceInvalidations.Contains(invalidationC))

	assert.Equal(t, 2, resourceInvalidations.Size())

	forEachResult = nil

	err = resourceInvalidations.ForEach(func(invalidation sema.ResourceInvalidation) error {
		forEachResult = append(forEachResult, invalidation)
		return nil
	})
	assert.NoError(t, err)
	expected = []sema.ResourceInvalidation{
		invalidationB,
		invalidationA,
	}
	assert.Equal(t,
		expected,
		forEachResult,
	)
	assert.Equal(t,
		expected,
		resourceInvalidations.All(),
	)

	// ... Add C, re-add A

	resourceInvalidations.Add(invalidationC)
	resourceInvalidations.Add(invalidationA)

	// ... Assert state after

	assert.True(t, resourceInvalidations.Contains(invalidationA))
	assert.True(t, resourceInvalidations.Contains(invalidationB))
	assert.True(t, resourceInvalidations.Contains(invalidationC))

	assert.Equal(t, 3, resourceInvalidations.Size())

	forEachResult = nil

	err = resourceInvalidations.ForEach(func(invalidation sema.ResourceInvalidation) error {
		forEachResult = append(forEachResult, invalidation)
		return nil
	})
	assert.NoError(t, err)
	expected = []sema.ResourceInvalidation{
		invalidationC,
		invalidationB,
		invalidationA,
	}
	assert.Equal(t,
		expected,
		forEachResult,
	)
	assert.Equal(t,
		expected,
		resourceInvalidations.All(),
	)

	// Pop

	resourceInvalidations = resourceInvalidations.Parent

	assert.True(t, resourceInvalidations.Contains(invalidationA))
	assert.True(t, resourceInvalidations.Contains(invalidationB))
	assert.False(t, resourceInvalidations.Contains(invalidationC))

	assert.Equal(t, 2, resourceInvalidations.Size())

	forEachResult = nil

	err = resourceInvalidations.ForEach(func(invalidation sema.ResourceInvalidation) error {
		forEachResult = append(forEachResult, invalidation)
		return nil
	})
	assert.NoError(t, err)
	expected = []sema.ResourceInvalidation{
		invalidationB,
		invalidationA,
	}
	assert.Equal(t,
		expected,
		forEachResult,
	)
	assert.Equal(t,
		expected,
		resourceInvalidations.All(),
	)

	// Pop

	resourceInvalidations = resourceInvalidations.Parent

	assert.True(t, resourceInvalidations.Contains(invalidationA))
	assert.False(t, resourceInvalidations.Contains(invalidationB))
	assert.False(t, resourceInvalidations.Contains(invalidationC))

	assert.Equal(t, 1, resourceInvalidations.Size())

	forEachResult = nil

	err = resourceInvalidations.ForEach(func(invalidation sema.ResourceInvalidation) error {
		forEachResult = append(forEachResult, invalidation)
		return nil
	})
	assert.NoError(t, err)
	expected = []sema.ResourceInvalidation{
		invalidationA,
	}
	assert.Equal(t,
		expected,
		forEachResult,
	)
	assert.Equal(t,
		expected,
		resourceInvalidations.All(),
	)
}

func TestResourceInvalidations_Merge(t *testing.T) {

	t.Parallel()

	invalidationA := sema.ResourceInvalidation{
		Kind:     sema.ResourceInvalidationKindMoveDefinite,
		StartPos: ast.Position{Offset: 1},
		EndPos:   ast.Position{Offset: 2},
	}
	invalidationB := sema.ResourceInvalidation{
		Kind:     sema.ResourceInvalidationKindMoveTemporary,
		StartPos: ast.Position{Offset: 3},
		EndPos:   ast.Position{Offset: 4},
	}
	invalidationC := sema.ResourceInvalidation{
		Kind:     sema.ResourceInvalidationKindDestroy,
		StartPos: ast.Position{Offset: 5},
		EndPos:   ast.Position{Offset: 6},
	}
	invalidationD := sema.ResourceInvalidation{
		Kind:     sema.ResourceInvalidationKindUnknown,
		StartPos: ast.Position{Offset: 7},
		EndPos:   ast.Position{Offset: 8},
	}

	A := &sema.ResourceInvalidations{}
	A.Add(invalidationA)

	AB := A.Clone()
	AB.Add(invalidationB)

	ABC := AB.Clone()
	ABC.Add(invalidationC)

	AD := A.Clone()
	AD.Add(invalidationD)

	ADC := AD.Clone()
	ADC.Add(invalidationC)

	result := AB.Clone()
	result.Merge(ADC)
	assert.True(t, result.Contains(invalidationA))
	assert.True(t, result.Contains(invalidationB))
	assert.True(t, result.Contains(invalidationC))
	assert.True(t, result.Contains(invalidationD))

	assert.True(t, A.Contains(invalidationA))
	assert.False(t, A.Contains(invalidationB))
	assert.False(t, A.Contains(invalidationC))
	assert.False(t, A.Contains(invalidationD))

	assert.True(t, AB.Contains(invalidationA))
	assert.True(t, AB.Contains(invalidationB))
	assert.False(t, AB.Contains(invalidationC))
	assert.False(t, AB.Contains(invalidationD))

	assert.True(t, ABC.Contains(invalidationA))
	assert.True(t, ABC.Contains(invalidationB))
	assert.True(t, ABC.Contains(invalidationC))
	assert.False(t, ABC.Contains(invalidationD))

	assert.True(t, AD.Contains(invalidationA))
	assert.False(t, AD.Contains(invalidationB))
	assert.False(t, AD.Contains(invalidationC))
	assert.True(t, AD.Contains(invalidationD))

	assert.True(t, ADC.Contains(invalidationA))
	assert.False(t, ADC.Contains(invalidationB))
	assert.True(t, ADC.Contains(invalidationC))
	assert.True(t, ADC.Contains(invalidationD))
}
