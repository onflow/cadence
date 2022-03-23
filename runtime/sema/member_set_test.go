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

	"github.com/onflow/cadence/runtime/sema"
)

func TestMemberSet(t *testing.T) {

	t.Parallel()

	memberA := &sema.Member{}
	memberB := &sema.Member{}
	memberC := &sema.Member{}

	// Parent set with only A

	// ... Prepare

	memberSet := sema.NewMemberSet(nil)

	memberSet.Add(memberA)

	// ... Assert state after

	assert.True(t, memberSet.Contains(memberA))
	assert.False(t, memberSet.Contains(memberB))
	assert.False(t, memberSet.Contains(memberC))

	var forEachResult []*sema.Member

	err := memberSet.ForEach(func(member *sema.Member) error {
		forEachResult = append(forEachResult, member)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]*sema.Member{
			memberA,
		},
		forEachResult,
	)

	// Child set with also B

	memberSet = memberSet.Clone()

	// ... Assert state before

	forEachResult = nil

	err = memberSet.ForEach(func(member *sema.Member) error {
		forEachResult = append(forEachResult, member)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]*sema.Member{
			memberA,
		},
		forEachResult,
	)

	// ... Add B

	memberSet.Add(memberB)

	// ... Assert state after

	assert.True(t, memberSet.Contains(memberA))
	assert.True(t, memberSet.Contains(memberB))
	assert.False(t, memberSet.Contains(memberC))

	forEachResult = nil

	err = memberSet.ForEach(func(member *sema.Member) error {
		forEachResult = append(forEachResult, member)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]*sema.Member{
			memberA,
			memberB,
		},
		forEachResult,
	)

	// Child set with also B

	memberSet = memberSet.Clone()

	// ... Assert state before

	forEachResult = nil

	err = memberSet.ForEach(func(member *sema.Member) error {
		forEachResult = append(forEachResult, member)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]*sema.Member{
			memberA,
			memberB,
		},
		forEachResult,
	)

	// ... Add C, re-add A

	memberSet.Add(memberC)
	memberSet.Add(memberA)

	// ... Assert state after

	assert.True(t, memberSet.Contains(memberA))
	assert.True(t, memberSet.Contains(memberB))
	assert.True(t, memberSet.Contains(memberC))

	forEachResult = nil

	err = memberSet.ForEach(func(member *sema.Member) error {
		forEachResult = append(forEachResult, member)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]*sema.Member{
			memberA,
			memberB,
			memberC,
		},
		forEachResult,
	)

	// Pop

	memberSet = memberSet.Parent

	assert.True(t, memberSet.Contains(memberA))
	assert.True(t, memberSet.Contains(memberB))
	assert.False(t, memberSet.Contains(memberC))

	forEachResult = nil

	err = memberSet.ForEach(func(member *sema.Member) error {
		forEachResult = append(forEachResult, member)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]*sema.Member{
			memberA,
			memberB,
		},
		forEachResult,
	)

	// Pop

	memberSet = memberSet.Parent

	assert.True(t, memberSet.Contains(memberA))
	assert.False(t, memberSet.Contains(memberB))
	assert.False(t, memberSet.Contains(memberC))

	forEachResult = nil

	err = memberSet.ForEach(func(member *sema.Member) error {
		forEachResult = append(forEachResult, member)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t,
		[]*sema.Member{
			memberA,
		},
		forEachResult,
	)
}

func TestMemberSet_Intersection(t *testing.T) {

	t.Parallel()

	memberA := &sema.Member{}
	memberB := &sema.Member{}
	memberC := &sema.Member{}
	memberD := &sema.Member{}
	memberE := &sema.Member{}

	A := sema.NewMemberSet(nil)
	A.Add(memberA)

	AB := A.Clone()
	AB.Add(memberB)

	ABC := AB.Clone()
	ABC.Add(memberC)

	AD := A.Clone()
	AD.Add(memberD)

	ADC := AD.Clone()
	ADC.Add(memberC)

	ACE := sema.NewMemberSet(nil)
	ACE.Add(memberE)
	ACE.AddIntersection(ABC, ADC)
	assert.True(t, ACE.Contains(memberA))
	assert.False(t, ACE.Contains(memberB))
	assert.True(t, ACE.Contains(memberC))
	assert.False(t, ACE.Contains(memberD))
	assert.True(t, ACE.Contains(memberE))

	assert.True(t, A.Contains(memberA))
	assert.False(t, A.Contains(memberB))
	assert.False(t, A.Contains(memberC))
	assert.False(t, A.Contains(memberD))

	assert.True(t, AB.Contains(memberA))
	assert.True(t, AB.Contains(memberB))
	assert.False(t, AB.Contains(memberC))
	assert.False(t, AB.Contains(memberD))

	assert.True(t, ABC.Contains(memberA))
	assert.True(t, ABC.Contains(memberB))
	assert.True(t, ABC.Contains(memberC))
	assert.False(t, ABC.Contains(memberD))

	assert.True(t, AD.Contains(memberA))
	assert.False(t, AD.Contains(memberB))
	assert.False(t, AD.Contains(memberC))
	assert.True(t, AD.Contains(memberD))

	assert.True(t, ADC.Contains(memberA))
	assert.False(t, ADC.Contains(memberB))
	assert.True(t, ADC.Contains(memberC))
	assert.True(t, ADC.Contains(memberD))
}
