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

package intervalst

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

type lineAndColumn struct {
	Line   int
	Column int
}

func (l lineAndColumn) Compare(other Position) int {
	if _, ok := other.(MinPosition); ok {
		return 1
	}

	otherL, ok := other.(lineAndColumn)
	if !ok {
		panic(fmt.Sprintf("not a lineAndColumn: %#+v", other))
	}
	if l.Line < otherL.Line {
		return -1
	}
	if l.Line > otherL.Line {
		return 1
	}
	if l.Column < otherL.Column {
		return -1
	}
	if l.Column > otherL.Column {
		return 1
	}
	return 0
}

func TestIntervalST_Search(t *testing.T) {

	t.Parallel()

	st := &IntervalST[int]{}

	st.Put(
		NewInterval(
			lineAndColumn{2, 2},
			lineAndColumn{2, 4},
		),
		100,
	)

	interval, value, present := st.Search(lineAndColumn{1, 3})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{2, 1})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{2, 2})
	assert.Equal(t, interval, &Interval{
		lineAndColumn{2, 2},
		lineAndColumn{2, 4},
	})
	assert.Equal(t, value, 100)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{2, 3})
	assert.Equal(t, interval, &Interval{
		lineAndColumn{2, 2},
		lineAndColumn{2, 4},
	})
	assert.Equal(t, value, 100)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{2, 4})
	assert.Equal(t, interval, &Interval{
		lineAndColumn{2, 2},
		lineAndColumn{2, 4},
	})
	assert.Equal(t, value, 100)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{2, 5})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	st.Put(
		NewInterval(
			lineAndColumn{3, 8},
			lineAndColumn{3, 8},
		),
		200,
	)

	interval, value, present = st.Search(lineAndColumn{2, 8})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{4, 8})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{3, 7})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{3, 8})
	assert.Equal(t, interval, &Interval{
		lineAndColumn{3, 8},
		lineAndColumn{3, 8},
	})
	assert.Equal(t, value, 200)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{3, 9})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	if !st.check() {
		t.Fail()
	}
}

func TestIntervalST_check(t *testing.T) {

	t.Parallel()

	intervals := []Interval{
		{
			lineAndColumn{Line: 2, Column: 12},
			lineAndColumn{Line: 2, Column: 12},
		},
		{
			lineAndColumn{Line: 3, Column: 12},
			lineAndColumn{Line: 3, Column: 12},
		},
		{
			lineAndColumn{Line: 5, Column: 12},
			lineAndColumn{Line: 5, Column: 13},
		},
		{
			lineAndColumn{Line: 5, Column: 15},
			lineAndColumn{Line: 5, Column: 20},
		},
		{
			lineAndColumn{Line: 5, Column: 28},
			lineAndColumn{Line: 5, Column: 33},
		},
		{
			lineAndColumn{Line: 6, Column: 15},
			lineAndColumn{Line: 6, Column: 15},
		},
		{
			lineAndColumn{Line: 7, Column: 15},
			lineAndColumn{Line: 7, Column: 15},
		},
		{
			lineAndColumn{Line: 7, Column: 25},
			lineAndColumn{Line: 7, Column: 25},
		},
		{
			lineAndColumn{Line: 8, Column: 15},
			lineAndColumn{Line: 8, Column: 16},
		},
		{
			lineAndColumn{Line: 9, Column: 21},
			lineAndColumn{Line: 9, Column: 21},
		},
		{
			lineAndColumn{Line: 9, Column: 25},
			lineAndColumn{Line: 9, Column: 25},
		},
		{
			lineAndColumn{Line: 14, Column: 15},
			lineAndColumn{Line: 14, Column: 16},
		},
		{
			lineAndColumn{Line: 15, Column: 16},
			lineAndColumn{Line: 15, Column: 19},
		},
		{
			lineAndColumn{Line: 18, Column: 18},
			lineAndColumn{Line: 18, Column: 19},
		},
		{
			lineAndColumn{Line: 20, Column: 12},
			lineAndColumn{Line: 20, Column: 13},
		},
		{
			lineAndColumn{Line: 21, Column: 11},
			lineAndColumn{Line: 21, Column: 12},
		},
		{
			lineAndColumn{Line: 22, Column: 18},
			lineAndColumn{Line: 22, Column: 19},
		},
	}

	st := &IntervalST[Interval]{}

	rand.Shuffle(len(intervals), func(i, j int) {
		intervals[i], intervals[j] = intervals[j], intervals[i]
	})

	for _, interval := range intervals {
		st.Put(interval, interval)
	}

	if !st.check() {
		t.Fail()
	}

	for _, interval := range intervals {
		res, _, _ := st.Search(interval.Min)
		assert.NotNil(t, res)
		res, _, _ = st.Search(interval.Max)
		assert.NotNil(t, res)
	}

	for _, interval := range st.Values() {
		res, _, _ := st.Search(interval.Min)
		assert.NotNil(t, res)
		res, _, _ = st.Search(interval.Max)
		assert.NotNil(t, res)
	}
}

func TestIntervalST_SearchAll(t *testing.T) {

	t.Parallel()

	st := &IntervalST[int]{}

	st.Put(
		NewInterval(
			lineAndColumn{1, 1},
			lineAndColumn{1, 2},
		),
		100,
	)

	st.Put(
		NewInterval(
			lineAndColumn{2, 4},
			lineAndColumn{2, 5},
		),
		200,
	)

	st.Put(
		NewInterval(
			lineAndColumn{3, 7},
			lineAndColumn{3, 10},
		),
		300,
	)

	st.Put(
		NewInterval(
			lineAndColumn{3, 8},
			lineAndColumn{3, 9},
		),
		400,
	)

	// Check line 2 (one interval)

	for i := 0; i <= 3; i++ {
		entries := st.SearchAll(lineAndColumn{2, i})
		assert.Empty(t, entries)
	}

	for i := 4; i <= 5; i++ {
		entries := st.SearchAll(lineAndColumn{2, i})
		assert.Equal(t,
			[]Entry[int]{
				{
					Interval: NewInterval(
						lineAndColumn{2, 4},
						lineAndColumn{2, 5},
					),
					Value: 200,
				},
			},
			entries,
		)
	}

	for i := 6; i <= 10; i++ {
		entries := st.SearchAll(lineAndColumn{2, i})
		assert.Empty(t, entries)
	}

	// Check line 3 (two overlapping intervals)

	for i := 0; i <= 6; i++ {
		entries := st.SearchAll(lineAndColumn{3, i})
		assert.Empty(t, entries)
	}

	entries := st.SearchAll(lineAndColumn{3, 7})
	assert.Equal(t,
		[]Entry[int]{
			{
				Interval: NewInterval(
					lineAndColumn{3, 7},
					lineAndColumn{3, 10},
				),
				Value: 300,
			},
		},
		entries,
	)

	for i := 8; i <= 9; i++ {
		entries = st.SearchAll(lineAndColumn{3, i})
		assert.ElementsMatch(t,
			[]Entry[int]{
				{
					Interval: NewInterval(
						lineAndColumn{3, 8},
						lineAndColumn{3, 9},
					),
					Value: 400,
				},
				{
					Interval: NewInterval(
						lineAndColumn{3, 7},
						lineAndColumn{3, 10},
					),
					Value: 300,
				},
			},
			entries,
		)
	}

	entries = st.SearchAll(lineAndColumn{3, 10})
	assert.Equal(t,
		[]Entry[int]{
			{
				Interval: NewInterval(
					lineAndColumn{3, 7},
					lineAndColumn{3, 10},
				),
				Value: 300,
			},
		},
		entries,
	)

	for i := 11; i <= 20; i++ {
		entries = st.SearchAll(lineAndColumn{3, i})
		assert.Empty(t, entries)
	}
}
