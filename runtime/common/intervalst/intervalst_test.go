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
			lineAndColumn{Line: 2, Column: 2},
			lineAndColumn{Line: 2, Column: 4},
		),
		100,
	)

	interval, value, present := st.Search(lineAndColumn{Line: 1, Column: 3})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 2, Column: 1})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 2, Column: 2})
	assert.Equal(t, interval, &Interval{
		Min: lineAndColumn{Line: 2, Column: 2},
		Max: lineAndColumn{Line: 2, Column: 4},
	})
	assert.Equal(t, value, 100)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 2, Column: 3})
	assert.Equal(t, interval, &Interval{
		Min: lineAndColumn{Line: 2, Column: 2},
		Max: lineAndColumn{Line: 2, Column: 4},
	})
	assert.Equal(t, value, 100)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 2, Column: 4})
	assert.Equal(t, interval, &Interval{
		Min: lineAndColumn{Line: 2, Column: 2},
		Max: lineAndColumn{Line: 2, Column: 4},
	})
	assert.Equal(t, value, 100)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 2, Column: 5})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	st.Put(
		NewInterval(
			lineAndColumn{Line: 3, Column: 8},
			lineAndColumn{Line: 3, Column: 8},
		),
		200,
	)

	interval, value, present = st.Search(lineAndColumn{Line: 2, Column: 8})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 4, Column: 8})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 3, Column: 7})
	assert.Nil(t, interval)
	assert.Zero(t, value)
	assert.False(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 3, Column: 8})
	assert.Equal(t, interval, &Interval{
		Min: lineAndColumn{Line: 3, Column: 8},
		Max: lineAndColumn{Line: 3, Column: 8},
	})
	assert.Equal(t, value, 200)
	assert.True(t, present)

	interval, value, present = st.Search(lineAndColumn{Line: 3, Column: 9})
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
			Min: lineAndColumn{Line: 2, Column: 12},
			Max: lineAndColumn{Line: 2, Column: 12},
		},
		{
			Min: lineAndColumn{Line: 3, Column: 12},
			Max: lineAndColumn{Line: 3, Column: 12},
		},
		{
			Min: lineAndColumn{Line: 5, Column: 12},
			Max: lineAndColumn{Line: 5, Column: 13},
		},
		{
			Min: lineAndColumn{Line: 5, Column: 15},
			Max: lineAndColumn{Line: 5, Column: 20},
		},
		{
			Min: lineAndColumn{Line: 5, Column: 28},
			Max: lineAndColumn{Line: 5, Column: 33},
		},
		{
			Min: lineAndColumn{Line: 6, Column: 15},
			Max: lineAndColumn{Line: 6, Column: 15},
		},
		{
			Min: lineAndColumn{Line: 7, Column: 15},
			Max: lineAndColumn{Line: 7, Column: 15},
		},
		{
			Min: lineAndColumn{Line: 7, Column: 25},
			Max: lineAndColumn{Line: 7, Column: 25},
		},
		{
			Min: lineAndColumn{Line: 8, Column: 15},
			Max: lineAndColumn{Line: 8, Column: 16},
		},
		{
			Min: lineAndColumn{Line: 9, Column: 21},
			Max: lineAndColumn{Line: 9, Column: 21},
		},
		{
			Min: lineAndColumn{Line: 9, Column: 25},
			Max: lineAndColumn{Line: 9, Column: 25},
		},
		{
			Min: lineAndColumn{Line: 14, Column: 15},
			Max: lineAndColumn{Line: 14, Column: 16},
		},
		{
			Min: lineAndColumn{Line: 15, Column: 16},
			Max: lineAndColumn{Line: 15, Column: 19},
		},
		{
			Min: lineAndColumn{Line: 18, Column: 18},
			Max: lineAndColumn{Line: 18, Column: 19},
		},
		{
			Min: lineAndColumn{Line: 20, Column: 12},
			Max: lineAndColumn{Line: 20, Column: 13},
		},
		{
			Min: lineAndColumn{Line: 21, Column: 11},
			Max: lineAndColumn{Line: 21, Column: 12},
		},
		{
			Min: lineAndColumn{Line: 22, Column: 18},
			Max: lineAndColumn{Line: 22, Column: 19},
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
			lineAndColumn{Line: 1, Column: 1},
			lineAndColumn{Line: 1, Column: 2},
		),
		100,
	)

	st.Put(
		NewInterval(
			lineAndColumn{Line: 2, Column: 4},
			lineAndColumn{Line: 2, Column: 5},
		),
		200,
	)

	st.Put(
		NewInterval(
			lineAndColumn{Line: 3, Column: 7},
			lineAndColumn{Line: 3, Column: 10},
		),
		300,
	)

	st.Put(
		NewInterval(
			lineAndColumn{Line: 3, Column: 8},
			lineAndColumn{Line: 3, Column: 9},
		),
		400,
	)

	// Check line 2 (one interval)

	for i := 0; i <= 3; i++ {
		entries := st.SearchAll(lineAndColumn{Line: 2, Column: i})
		assert.Empty(t, entries)
	}

	for i := 4; i <= 5; i++ {
		entries := st.SearchAll(lineAndColumn{Line: 2, Column: i})
		assert.Equal(t,
			[]Entry[int]{
				{
					Interval: NewInterval(
						lineAndColumn{Line: 2, Column: 4},
						lineAndColumn{Line: 2, Column: 5},
					),
					Value: 200,
				},
			},
			entries,
		)
	}

	for i := 6; i <= 10; i++ {
		entries := st.SearchAll(lineAndColumn{Line: 2, Column: i})
		assert.Empty(t, entries)
	}

	// Check line 3 (two overlapping intervals)

	for i := 0; i <= 6; i++ {
		entries := st.SearchAll(lineAndColumn{Line: 3, Column: i})
		assert.Empty(t, entries)
	}

	entries := st.SearchAll(lineAndColumn{Line: 3, Column: 7})
	assert.Equal(t,
		[]Entry[int]{
			{
				Interval: NewInterval(
					lineAndColumn{Line: 3, Column: 7},
					lineAndColumn{Line: 3, Column: 10},
				),
				Value: 300,
			},
		},
		entries,
	)

	for i := 8; i <= 9; i++ {
		entries = st.SearchAll(lineAndColumn{Line: 3, Column: i})
		assert.ElementsMatch(t,
			[]Entry[int]{
				{
					Interval: NewInterval(
						lineAndColumn{Line: 3, Column: 8},
						lineAndColumn{Line: 3, Column: 9},
					),
					Value: 400,
				},
				{
					Interval: NewInterval(
						lineAndColumn{Line: 3, Column: 7},
						lineAndColumn{Line: 3, Column: 10},
					),
					Value: 300,
				},
			},
			entries,
		)
	}

	entries = st.SearchAll(lineAndColumn{Line: 3, Column: 10})
	assert.Equal(t,
		[]Entry[int]{
			{
				Interval: NewInterval(
					lineAndColumn{Line: 3, Column: 7},
					lineAndColumn{Line: 3, Column: 10},
				),
				Value: 300,
			},
		},
		entries,
	)

	for i := 11; i <= 20; i++ {
		entries = st.SearchAll(lineAndColumn{Line: 3, Column: i})
		assert.Empty(t, entries)
	}
}
