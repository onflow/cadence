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

package persistent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/common/persistent"
)

func TestOrderedSet(t *testing.T) {

	t.Parallel()

	itemA := "a"
	itemB := "b"
	itemC := "c"

	// Parent set with only A

	// ... Prepare

	set := persistent.NewOrderedSet[string](nil)

	set.Add(itemA)

	// ... Assert state after

	assert.True(t, set.Contains(itemA))
	assert.False(t, set.Contains(itemB))
	assert.False(t, set.Contains(itemC))

	var forEachResult []string

	err := set.ForEach(func(item string) error {
		forEachResult = append(forEachResult, item)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			itemA,
		},
		forEachResult,
	)

	// Child set with also B

	set = set.Clone()

	// ... Assert state before

	forEachResult = nil

	err = set.ForEach(func(item string) error {
		forEachResult = append(forEachResult, item)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			itemA,
		},
		forEachResult,
	)

	// ... Add B

	set.Add(itemB)

	// ... Assert state after

	assert.True(t, set.Contains(itemA))
	assert.True(t, set.Contains(itemB))
	assert.False(t, set.Contains(itemC))

	forEachResult = nil

	err = set.ForEach(func(item string) error {
		forEachResult = append(forEachResult, item)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			itemB,
			itemA,
		},
		forEachResult,
	)

	// Child set with also B

	set = set.Clone()

	// ... Assert state before

	forEachResult = nil

	err = set.ForEach(func(item string) error {
		forEachResult = append(forEachResult, item)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			itemB,
			itemA,
		},
		forEachResult,
	)

	// ... Add C, re-add A

	set.Add(itemC)
	set.Add(itemA)

	// ... Assert state after

	assert.True(t, set.Contains(itemA))
	assert.True(t, set.Contains(itemB))
	assert.True(t, set.Contains(itemC))

	forEachResult = nil

	err = set.ForEach(func(item string) error {
		forEachResult = append(forEachResult, item)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			itemC,
			itemB,
			itemA,
		},
		forEachResult,
	)

	// Pop

	set = set.Parent

	assert.True(t, set.Contains(itemA))
	assert.True(t, set.Contains(itemB))
	assert.False(t, set.Contains(itemC))

	forEachResult = nil

	err = set.ForEach(func(item string) error {
		forEachResult = append(forEachResult, item)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			itemB,
			itemA,
		},
		forEachResult,
	)

	// Pop

	set = set.Parent

	assert.True(t, set.Contains(itemA))
	assert.False(t, set.Contains(itemB))
	assert.False(t, set.Contains(itemC))

	forEachResult = nil

	err = set.ForEach(func(item string) error {
		forEachResult = append(forEachResult, item)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{
			itemA,
		},
		forEachResult,
	)
}

func TestOrderedSet_Intersection(t *testing.T) {

	t.Parallel()

	itemA := "a"
	itemB := "b"
	itemC := "c"
	itemD := "d"
	itemE := "e"

	A := persistent.NewOrderedSet[string](nil)
	A.Add(itemA)

	AB := A.Clone()
	AB.Add(itemB)

	ABC := AB.Clone()
	ABC.Add(itemC)

	AD := A.Clone()
	AD.Add(itemD)

	ADC := AD.Clone()
	ADC.Add(itemC)

	ACE := persistent.NewOrderedSet[string](nil)
	ACE.Add(itemE)
	ACE.AddIntersection(ABC, ADC)
	assert.True(t, ACE.Contains(itemA))
	assert.False(t, ACE.Contains(itemB))
	assert.True(t, ACE.Contains(itemC))
	assert.False(t, ACE.Contains(itemD))
	assert.True(t, ACE.Contains(itemE))

	assert.True(t, A.Contains(itemA))
	assert.False(t, A.Contains(itemB))
	assert.False(t, A.Contains(itemC))
	assert.False(t, A.Contains(itemD))

	assert.True(t, AB.Contains(itemA))
	assert.True(t, AB.Contains(itemB))
	assert.False(t, AB.Contains(itemC))
	assert.False(t, AB.Contains(itemD))

	assert.True(t, ABC.Contains(itemA))
	assert.True(t, ABC.Contains(itemB))
	assert.True(t, ABC.Contains(itemC))
	assert.False(t, ABC.Contains(itemD))

	assert.True(t, AD.Contains(itemA))
	assert.False(t, AD.Contains(itemB))
	assert.False(t, AD.Contains(itemC))
	assert.True(t, AD.Contains(itemD))

	assert.True(t, ADC.Contains(itemA))
	assert.False(t, ADC.Contains(itemB))
	assert.True(t, ADC.Contains(itemC))
	assert.True(t, ADC.Contains(itemD))
}
