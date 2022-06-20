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

type IntervalST struct {
	root *node
}

func (t *IntervalST) Get(interval Interval) any {
	return t.get(t.root, interval)
}

func (t *IntervalST) get(x *node, interval Interval) any {
	if x == nil {
		return nil
	}
	switch cmp := interval.Compare(x.interval); {
	case cmp < 0:
		return t.get(x.left, interval)
	case cmp > 0:
		return t.get(x.right, interval)
	default:
		return x.value
	}
}

func (t *IntervalST) Contains(interval Interval) bool {
	return t.Get(interval) != nil
}

// Put associates an interval with a value.
//
// NOTE: does *not* check if the interval already exists
//
func (t *IntervalST) Put(interval Interval, value any) {
	t.root = t.randomizedInsert(t.root, interval, value)
}

func (t *IntervalST) randomizedInsert(x *node, interval Interval, value any) *node {
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

func (t *IntervalST) rootInsert(x *node, interval Interval, value any) *node {
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

func (t *IntervalST) SearchInterval(interval Interval) (*Interval, any) {
	return t.searchInterval(t.root, interval)
}

func (t *IntervalST) searchInterval(x *node, interval Interval) (*Interval, any) {
	for x != nil {
		if x.interval.Intersects(interval) {
			return &x.interval, x.value
		} else if x.left == nil || x.left.max.Compare(interval.Min) < 0 {
			x = x.right
		} else {
			x = x.left
		}
	}
	return nil, nil
}

func (t *IntervalST) Search(p Position) (*Interval, any) {
	return t.search(t.root, p)
}

func (t *IntervalST) search(x *node, p Position) (*Interval, any) {
	for x != nil {
		if x.interval.Contains(p) {
			return &x.interval, x.value
		} else if x.left == nil || x.left.max.Compare(p) < 0 {
			x = x.right
		} else {
			x = x.left
		}
	}
	return nil, nil
}

type Entry struct {
	Interval Interval
	Value    any
}

func (t *IntervalST) SearchAll(p Position) []Entry {
	_, entries := t.searchAll(t.root, p, nil)
	return entries
}

func (t *IntervalST) searchAll(n *node, p Position, entries []Entry) (bool, []Entry) {
	found1 := false
	found2 := false
	found3 := false

	if n == nil {
		return false, entries
	}

	if n.interval.Contains(p) {
		found1 = true
		entries = append(entries,
			Entry{
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

func (t *IntervalST) Values() []any {
	return t.root.Values()
}

func (t *IntervalST) check() bool {
	return t.root.checkCount() && t.root.checkMax()
}
