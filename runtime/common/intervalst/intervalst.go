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

import "math/rand"

// IntervalST

type IntervalST[T any] struct {
	root *node[T]
}

func (t *IntervalST[T]) Get(interval Interval) (T, bool) {
	return t.get(t.root, interval)
}

func (t *IntervalST[T]) get(x *node[T], interval Interval) (result T, present bool) {
	if x == nil {
		return
	}
	switch cmp := interval.Compare(x.interval); {
	case cmp < 0:
		return t.get(x.left, interval)
	case cmp > 0:
		return t.get(x.right, interval)
	default:
		return x.value, true
	}
}

func (t *IntervalST[T]) Contains(interval Interval) bool {
	_, present := t.Get(interval)
	return present
}

// Put associates an interval with a value.
//
// NOTE: does *not* check if the interval already exists
//
func (t *IntervalST[T]) Put(interval Interval, value T) {
	t.root = t.randomizedInsert(t.root, interval, value)
}

func (t *IntervalST[T]) randomizedInsert(x *node[T], interval Interval, value T) *node[T] {
	if x == nil {
		return newNode(interval, value)
	}

	if rand.Float32()*float32(x.size()) < 1.0 {
		return t.rootInsert(x, interval, value)
	}

	cmp := interval.Compare(x.interval)
	if cmp < 0 {
		x.left = t.randomizedInsert(x.left, interval, value)
	} else {
		x.right = t.randomizedInsert(x.right, interval, value)
	}

	x.fix()

	return x
}

func (t *IntervalST[T]) rootInsert(x *node[T], interval Interval, value T) *node[T] {
	if x == nil {
		return newNode(interval, value)
	}

	cmp := interval.Compare(x.interval)
	if cmp < 0 {
		x.left = t.rootInsert(x.left, interval, value)
		x = x.rotR()
	} else {
		x.right = t.rootInsert(x.right, interval, value)
		x = x.rotL()
	}

	return x
}

func (t *IntervalST[T]) SearchInterval(interval Interval) (*Interval, T, bool) {
	return t.searchInterval(t.root, interval)
}

func (t *IntervalST[T]) searchInterval(x *node[T], interval Interval) (i *Interval, value T, present bool) {
	for x != nil {
		if x.interval.Intersects(interval) {
			return &x.interval, x.value, true
		} else if x.left == nil || x.left.max.Compare(interval.Min) < 0 {
			x = x.right
		} else {
			x = x.left
		}
	}
	return i, value, present
}

func (t *IntervalST[T]) Search(p Position) (*Interval, T, bool) {
	return t.search(t.root, p)
}

func (t *IntervalST[T]) search(x *node[T], p Position) (i *Interval, value T, present bool) {
	for x != nil {
		if x.interval.Contains(p) {
			return &x.interval, x.value, true
		} else if x.left == nil || x.left.max.Compare(p) < 0 {
			x = x.right
		} else {
			x = x.left
		}
	}
	return i, value, false
}

type Entry[T any] struct {
	Interval Interval
	Value    T
}

func (t *IntervalST[T]) SearchAll(p Position) []Entry[T] {
	_, entries := t.searchAll(t.root, p, nil)
	return entries
}

func (t *IntervalST[T]) searchAll(n *node[T], p Position, entries []Entry[T]) (bool, []Entry[T]) {
	found1 := false
	found2 := false
	found3 := false

	if n == nil {
		return false, entries
	}

	if n.interval.Contains(p) {
		found1 = true
		entries = append(entries,
			Entry[T]{
				Interval: n.interval,
				Value:    n.value,
			},
		)
	}

	if n.left != nil && n.left.max.Compare(p) >= 0 {
		found2, entries = t.searchAll(n.left, p, entries)
	}

	if found2 || n.left == nil || n.left.max.Compare(p) < 0 {
		found3, entries = t.searchAll(n.right, p, entries)
	}

	found := found1 || found2 || found3

	return found, entries
}

func (t *IntervalST[T]) Values() []T {
	return t.root.Values()
}

func (t *IntervalST[T]) check() bool {
	return t.root.checkCount() && t.root.checkMax()
}
