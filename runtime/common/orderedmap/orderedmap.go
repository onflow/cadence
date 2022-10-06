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
 *
 * Based on https://github.com/wk8/go-ordered-map, Copyright Jean Rougé
 *
 */

package orderedmap

import (
	"github.com/onflow/cadence/runtime/common/list"
)

// OrderedMap
//
type OrderedMap[K comparable, V any] struct {
	pairs map[K]*Pair[K, V]
	list  *list.List[*Pair[K, V]]
}

// New returns a new OrderedMap of the given size
func New[T OrderedMap[K, V], K comparable, V any](size int) *T {
	return &T{
		pairs: make(map[K]*Pair[K, V], size),
		list:  list.New[*Pair[K, V]](),
	}
}

func (om *OrderedMap[K, V]) ensureInitialized() {
	if om.pairs != nil {
		return
	}
	om.pairs = make(map[K]*Pair[K, V])
	om.list = list.New[*Pair[K, V]]()
}

// Clear removes all entries from this ordered map.
func (om *OrderedMap[K, V]) Clear() {
	if om.list == nil {
		return
	}

	om.list.Init()
	// NOTE: Range over map is safe, as it is only used to delete entries
	for key := range om.pairs { //nolint:maprangecheck
		delete(om.pairs, key)
	}
}

// Get returns the value associated with the given key.
// Returns nil if not found.
// The second return value indicates if the key is present in the map.
func (om OrderedMap[K, V]) Get(key K) (result V, present bool) {
	if om.pairs == nil {
		return
	}

	var pair *Pair[K, V]
	if pair, present = om.pairs[key]; present {
		return pair.Value, present
	}
	return
}

// Contains returns true if the key is present in the map
// and false otherwise.
func (om OrderedMap[K, V]) Contains(key K) (present bool) {
	if om.pairs == nil {
		return
	}

	_, present = om.pairs[key]
	return
}

// GetPair returns the key-value pair associated with the given key.
// Returns nil if not found.
func (om OrderedMap[K, V]) GetPair(key K) *Pair[K, V] {
	if om.pairs == nil {
		return nil
	}

	return om.pairs[key]
}

// Set sets the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Set`.
func (om *OrderedMap[K, V]) Set(key K, value V) (oldValue V, present bool) {
	om.ensureInitialized()

	var pair *Pair[K, V]
	if pair, present = om.pairs[key]; present {
		oldValue = pair.Value
		pair.Value = value
		return
	}

	pair = &Pair[K, V]{
		Key:   key,
		Value: value,
	}
	pair.element = om.list.PushBack(pair)
	om.pairs[key] = pair

	return
}

// Delete removes the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Delete`.
func (om *OrderedMap[K, V]) Delete(key K) (oldValue V, present bool) {
	if om.pairs == nil {
		return
	}

	var pair *Pair[K, V]
	pair, present = om.pairs[key]
	if !present {
		return
	}

	om.list.Remove(pair.element)
	delete(om.pairs, key)
	oldValue = pair.Value

	return
}

// Len returns the length of the ordered map.
func (om OrderedMap[K, V]) Len() int {
	return len(om.pairs)
}

// Oldest returns a pointer to the oldest pair.
func (om OrderedMap[K, V]) Oldest() *Pair[K, V] {
	if om.pairs == nil {
		return nil
	}

	return elementToPair[K, V](om.list.Front())
}

// Newest returns a pointer to the newest pair.
func (om OrderedMap[K, V]) Newest() *Pair[K, V] {
	if om.pairs == nil {
		return nil
	}

	return elementToPair[K, V](om.list.Back())
}

// Foreach iterates over the entries of the map in the insertion order, and invokes
// the provided function for each key-value pair.
func (om OrderedMap[K, V]) Foreach(f func(key K, value V)) {
	if om.pairs == nil {
		return
	}

	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		f(pair.Key, pair.Value)
	}
}

// ForeachWithError iterates over the entries of the map in the insertion order,
// and invokes the provided function for each key-value pair.
// If the passed function returns an error, iteration breaks and the error is returned.
func (om OrderedMap[K, V]) ForeachWithError(f func(key K, value V) error) error {
	if om.pairs == nil {
		return nil
	}

	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		err := f(pair.Key, pair.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// Pair is an entry in an OrderedMap
//
type Pair[K any, V any] struct {
	Key   K
	Value V

	element *list.Element[*Pair[K, V]]
}

// Next returns a pointer to the next pair.
func (p Pair[K, V]) Next() *Pair[K, V] {
	return elementToPair[K, V](p.element.Next())
}

// Prev returns a pointer to the previous pair.
func (p Pair[K, V]) Prev() *Pair[K, V] {
	return elementToPair[K, V](p.element.Prev())
}

func elementToPair[K any, V any](element *list.Element[*Pair[K, V]]) *Pair[K, V] {
	if element == nil {
		return nil
	}
	return element.Value
}
