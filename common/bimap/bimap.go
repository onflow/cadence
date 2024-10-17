/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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
 * Based on https://github.com/vishalkuo/bimap, Copyright Vishal Kuo
 *
 */

package bimap

type BiMap[K comparable, V comparable] struct {
	forward  map[K]V
	backward map[V]K
}

// NewBiMap returns a an empty, mutable, biMap
func NewBiMap[K comparable, V comparable]() *BiMap[K, V] {
	return &BiMap[K, V]{forward: make(map[K]V), backward: make(map[V]K)}
}

// Insert puts a key and value into the BiMap, and creates the reverse mapping from value to key.
func (b *BiMap[K, V]) Insert(k K, v V) {
	if _, ok := b.forward[k]; ok {
		delete(b.backward, b.forward[k])
	}
	b.forward[k] = v
	b.backward[v] = k
}

// Exists checks whether or not a key exists in the BiMap
func (b *BiMap[K, V]) Exists(k K) bool {
	_, ok := b.forward[k]
	return ok
}

// ExistsInverse checks whether or not a value exists in the BiMap
func (b *BiMap[K, V]) ExistsInverse(k V) bool {
	_, ok := b.backward[k]
	return ok
}

// Get returns the value for a given key in the BiMap and whether or not the element was present.
func (b *BiMap[K, V]) Get(k K) (V, bool) {
	if !b.Exists(k) {
		return *new(V), false
	}
	return b.forward[k], true
}

// GetInverse returns the key for a given value in the BiMap and whether or not the element was present.
func (b *BiMap[K, V]) GetInverse(v V) (K, bool) {
	if !b.ExistsInverse(v) {
		return *new(K), false
	}
	return b.backward[v], true
}

// Delete removes a key-value pair from the BiMap for a given key. Returns if the key doesn't exist
func (b *BiMap[K, V]) Delete(k K) {
	if !b.Exists(k) {
		return
	}
	val, _ := b.Get(k)
	delete(b.forward, k)
	delete(b.backward, val)
}

// DeleteInverse removes a key-value pair from the BiMap for a given value. Returns if the value doesn't exist
func (b *BiMap[K, V]) DeleteInverse(v V) {
	if !b.ExistsInverse(v) {
		return
	}

	key, _ := b.GetInverse(v)
	delete(b.backward, v)
	delete(b.forward, key)

}

// Size returns the number of elements in the bimap
func (b *BiMap[K, V]) Size() int {
	return len(b.forward)
}
