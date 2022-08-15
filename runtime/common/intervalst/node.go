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

type node[T any] struct {
	interval    Interval
	value       T
	left, right *node[T]
	// size of subtree rooted at this node
	n int
	// max endpoint in subtree rooted at this node
	max Position
}

func newNode[T any](interval Interval, value T) *node[T] {
	return &node[T]{
		interval: interval,
		value:    value,
		n:        1,
		max:      interval.Max,
	}
}

func (n *node[T]) size() int {
	if n == nil {
		return 0
	}
	return n.n
}

type MinPosition struct{}

func (MinPosition) Compare(other Position) int {
	_, ok := other.(MinPosition)
	if ok {
		return 0
	}
	return -1
}

var minPosition = MinPosition{}

func (n *node[T]) Max() Position {
	if n == nil {
		return minPosition
	}

	return n.max
}

func (n *node[T]) fix() {
	if n == nil {
		return
	}

	n.n = 1 + n.left.size() + n.right.size()
	n.max = max3(n.interval.Max, n.left.Max(), n.right.Max())
}

func max3(a, b, c Position) Position {
	if b.Compare(a) >= 0 && b.Compare(c) >= 0 {
		return b
	}
	if c.Compare(a) >= 0 && c.Compare(b) >= 0 {
		return c
	}
	return a
}

func (n *node[T]) rotR() *node[T] {
	x := n.left
	n.left = x.right
	x.right = n
	n.fix()
	x.fix()
	return x
}

func (n *node[T]) rotL() *node[T] {
	x := n.right
	n.right = x.left
	x.left = n
	n.fix()
	x.fix()
	return x
}

func (n *node[T]) Values() []T {
	if n == nil {
		return nil
	}

	return append(
		append(n.left.Values(), n.right.Values()...),
		n.value,
	)
}

func (n *node[T]) checkCount() bool {
	return n == nil ||
		(n.left.checkCount() && n.right.checkCount() &&
			(n.n == 1+n.left.size()+n.right.size()))
}

func (n *node[T]) checkMax() bool {
	if n == nil {
		return true
	}
	actual := max3(n.interval.Max, n.left.Max(), n.right.Max())
	return n.max.Compare(actual) == 0
}
