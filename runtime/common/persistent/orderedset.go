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

package persistent

import (
	"github.com/onflow/cadence/runtime/common/orderedmap"
)

type OrderedSet[T comparable] struct {
	Parent *OrderedSet[T]
	items  *orderedmap.OrderedMap[T, struct{}]
}

// NewOrderedSet returns a set with the given parent.
// To create an empty set, pass nil.
func NewOrderedSet[T comparable](parent *OrderedSet[T]) *OrderedSet[T] {
	return &OrderedSet[T]{
		Parent: parent,
	}
}

// Add inserts an item into the set.
func (s *OrderedSet[T]) Add(item T) {

	if s.Contains(item) {
		return
	}

	if s.items == nil {
		s.items = &orderedmap.OrderedMap[T, struct{}]{}
	}

	s.items.Set(item, struct{}{})
}

// Contains returns true if the given item exists in the set.
func (s *OrderedSet[T]) Contains(item T) (present bool) {
	currentS := s

	for currentS != nil {
		if currentS.items != nil {
			present = currentS.items.Contains(item)
			if present {
				return
			}
		}

		currentS = currentS.Parent
	}

	return
}

// ForEach calls the given function for each item.
// It can be used to iterate over all items of the set.
func (s *OrderedSet[T]) ForEach(cb func(item T) error) error {
	currentS := s

	for currentS != nil {

		if currentS.items != nil {
			for pair := currentS.items.Oldest(); pair != nil; pair = pair.Next() {
				item := pair.Key

				err := cb(item)
				if err != nil {
					return err
				}

			}
		}

		currentS = currentS.Parent
	}

	return nil
}

// AddIntersection adds the members that exist in both given member sets.
func (s *OrderedSet[T]) AddIntersection(a, b *OrderedSet[T]) {

	_ = a.ForEach(func(item T) error {
		if b.Contains(item) {
			s.Add(item)
		}

		return nil
	})
}

// Clone returns a new child set that contains all items of this parent set.
// Changes to the returned set will only be applied in the returned set, not the parent.
func (s *OrderedSet[T]) Clone() *OrderedSet[T] {
	return NewOrderedSet(s)
}
