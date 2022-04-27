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

func TestResourceUses(t *testing.T) {

	t.Parallel()

	posA := ast.Position{Offset: 0}
	posB := ast.Position{Offset: 1}
	posC := ast.Position{Offset: 2}

	// Parent set with only A

	// ... Prepare

	resourceUses := &sema.ResourceUses{}

	resourceUses.Add(posA)

	// ... Assert state after

	assert.True(t, resourceUses.Contains(posA))
	assert.False(t, resourceUses.Contains(posB))
	assert.False(t, resourceUses.Contains(posC))

	assert.Equal(t, 1, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	type entry struct {
		pos ast.Position
		use sema.ResourceUse
	}

	var forEachResult []entry

	err := resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)

	// Child set with also B

	withB := resourceUses.Clone()
	resourceUses = &withB

	// ... Assert state before

	assert.True(t, resourceUses.Contains(posA))
	assert.False(t, resourceUses.Contains(posB))
	assert.False(t, resourceUses.Contains(posC))

	assert.Equal(t, 1, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	forEachResult = nil

	err = resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)

	// ... Add B

	resourceUses.Add(posB)

	// ... Assert state after

	assert.True(t, resourceUses.Contains(posA))
	assert.True(t, resourceUses.Contains(posB))
	assert.False(t, resourceUses.Contains(posC))

	assert.Equal(t, 2, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	forEachResult = nil

	err = resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posB,
				use: sema.ResourceUse{},
			},
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)

	// Mark B use after invalidation reported

	resourceUses.MarkUseAfterInvalidationReported(posB)

	// ... Assert state after

	assert.True(t, resourceUses.Contains(posA))
	assert.True(t, resourceUses.Contains(posB))
	assert.False(t, resourceUses.Contains(posC))

	assert.Equal(t, 2, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.True(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	forEachResult = nil

	err = resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posB,
				use: sema.ResourceUse{
					UseAfterInvalidationReported: true,
				},
			},
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)

	// Child set with also C

	withC := resourceUses.Clone()
	resourceUses = &withC

	// ... Assert state before

	assert.True(t, resourceUses.Contains(posA))
	assert.True(t, resourceUses.Contains(posB))
	assert.False(t, resourceUses.Contains(posC))

	assert.Equal(t, 2, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.True(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	forEachResult = nil

	err = resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posB,
				use: sema.ResourceUse{
					UseAfterInvalidationReported: true,
				},
			},
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)

	// ... Add C, re-add A

	resourceUses.Add(posC)
	resourceUses.Add(posA)

	// ... Assert state after

	assert.True(t, resourceUses.Contains(posA))
	assert.True(t, resourceUses.Contains(posB))
	assert.True(t, resourceUses.Contains(posC))

	assert.Equal(t, 3, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.True(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	forEachResult = nil

	err = resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posC,
				use: sema.ResourceUse{},
			},
			{
				pos: posB,
				use: sema.ResourceUse{
					UseAfterInvalidationReported: true,
				},
			},
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)

	// Pop

	resourceUses = resourceUses.Parent

	assert.True(t, resourceUses.Contains(posA))
	assert.True(t, resourceUses.Contains(posB))
	assert.False(t, resourceUses.Contains(posC))

	assert.Equal(t, 2, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.True(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	forEachResult = nil

	err = resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posB,
				use: sema.ResourceUse{
					UseAfterInvalidationReported: true,
				},
			},
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)

	// Pop

	resourceUses = resourceUses.Parent

	assert.True(t, resourceUses.Contains(posA))
	assert.False(t, resourceUses.Contains(posB))
	assert.False(t, resourceUses.Contains(posC))

	assert.Equal(t, 1, resourceUses.Size())

	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posA))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posB))
	assert.False(t, resourceUses.IsUseAfterInvalidationReported(posC))

	forEachResult = nil

	err = resourceUses.ForEach(func(pos ast.Position, use sema.ResourceUse) error {
		forEachResult = append(forEachResult, entry{pos: pos, use: use})
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]entry{
			{
				pos: posA,
				use: sema.ResourceUse{},
			},
		},
		forEachResult,
	)
}

func TestResourceUses_Merge(t *testing.T) {

	t.Parallel()

	posA := ast.Position{Offset: 0}
	posB := ast.Position{Offset: 1}
	posC := ast.Position{Offset: 2}
	posD := ast.Position{Offset: 3}

	A := &sema.ResourceUses{}
	A.Add(posA)

	AB := A.Clone()
	AB.Add(posB)

	ABC := AB.Clone()
	ABC.Add(posC)

	AD := A.Clone()
	AD.Add(posD)

	ADC := AD.Clone()
	ADC.Add(posC)

	result := AB.Clone()
	result.Merge(ADC)
	assert.True(t, result.Contains(posA))
	assert.True(t, result.Contains(posB))
	assert.True(t, result.Contains(posC))
	assert.True(t, result.Contains(posD))

	assert.True(t, A.Contains(posA))
	assert.False(t, A.Contains(posB))
	assert.False(t, A.Contains(posC))
	assert.False(t, A.Contains(posD))

	assert.True(t, AB.Contains(posA))
	assert.True(t, AB.Contains(posB))
	assert.False(t, AB.Contains(posC))
	assert.False(t, AB.Contains(posD))

	assert.True(t, ABC.Contains(posA))
	assert.True(t, ABC.Contains(posB))
	assert.True(t, ABC.Contains(posC))
	assert.False(t, ABC.Contains(posD))

	assert.True(t, AD.Contains(posA))
	assert.False(t, AD.Contains(posB))
	assert.False(t, AD.Contains(posC))
	assert.True(t, AD.Contains(posD))

	assert.True(t, ADC.Contains(posA))
	assert.False(t, ADC.Contains(posB))
	assert.True(t, ADC.Contains(posC))
	assert.True(t, ADC.Contains(posD))
}
