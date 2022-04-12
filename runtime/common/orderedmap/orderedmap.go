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
	"container/list"

	"github.com/cheekybits/genny/generic"
)

type KeyType generic.Type
type ValueType generic.Type

// KeyTypeValueTypeOrderedMap
//
type KeyTypeValueTypeOrderedMap struct {
	pairs map[KeyType]*KeyTypeValueTypePair
	list  *list.List
}

// NewKeyTypeValueTypeOrderedMap creates a new KeyTypeValueTypeOrderedMap.
func NewKeyTypeValueTypeOrderedMap() *KeyTypeValueTypeOrderedMap {
	return &KeyTypeValueTypeOrderedMap{
		pairs: make(map[KeyType]*KeyTypeValueTypePair),
		list:  list.New(),
	}
}

// Clear removes all entries from this ordered map.
func (om *KeyTypeValueTypeOrderedMap) Clear() {
	om.list.Init()
	// NOTE: Range over map is safe, as it is only used to delete entries
	for key := range om.pairs { //nolint:maprangecheck
		delete(om.pairs, key)
	}
}

// Get returns the value associated with the given key.
// Returns nil if not found.
// The second return value indicates if the key is present in the map.
func (om *KeyTypeValueTypeOrderedMap) Get(key KeyType) (result ValueType, present bool) {
	var pair *KeyTypeValueTypePair
	if pair, present = om.pairs[key]; present {
		return pair.Value, present
	}
	return
}

// GetPair returns the key-value pair associated with the given key.
// Returns nil if not found.
func (om *KeyTypeValueTypeOrderedMap) GetPair(key KeyType) *KeyTypeValueTypePair {
	return om.pairs[key]
}

// Set sets the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Set`.
func (om *KeyTypeValueTypeOrderedMap) Set(key KeyType, value ValueType) (oldValue ValueType, present bool) {
	var pair *KeyTypeValueTypePair
	if pair, present = om.pairs[key]; present {
		oldValue = pair.Value
		pair.Value = value
		return
	}

	pair = &KeyTypeValueTypePair{
		Key:   key,
		Value: value,
	}
	pair.element = om.list.PushBack(pair)
	om.pairs[key] = pair

	return
}

// Delete removes the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Delete`.
func (om *KeyTypeValueTypeOrderedMap) Delete(key KeyType) (oldValue ValueType, present bool) {
	var pair *KeyTypeValueTypePair
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
func (om *KeyTypeValueTypeOrderedMap) Len() int {
	return len(om.pairs)
}

// Oldest returns a pointer to the oldest pair.
func (om *KeyTypeValueTypeOrderedMap) Oldest() *KeyTypeValueTypePair {
	return listElementToKeyTypeValueTypePair(om.list.Front())
}

// Newest returns a pointer to the newest pair.
func (om *KeyTypeValueTypeOrderedMap) Newest() *KeyTypeValueTypePair {
	return listElementToKeyTypeValueTypePair(om.list.Back())
}

// Foreach iterates over the entries of the map in the insertion order, and invokes
// the provided function for each key-value pair.
func (om *KeyTypeValueTypeOrderedMap) Foreach(f func(key KeyType, value ValueType)) {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		f(pair.Key, pair.Value)
	}
}

// ForeachWithError iterates over the entries of the map in the insertion order,
// and invokes the provided function for each key-value pair.
// If the passed function returns an error, iteration breaks and the error is returned.
func (om *KeyTypeValueTypeOrderedMap) ForeachWithError(f func(key KeyType, value ValueType) error) error {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		err := f(pair.Key, pair.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// KeyTypeValueTypePair
//
type KeyTypeValueTypePair struct {
	Key   KeyType
	Value ValueType

	element *list.Element
}

// Next returns a pointer to the next pair.
func (p *KeyTypeValueTypePair) Next() *KeyTypeValueTypePair {
	return listElementToKeyTypeValueTypePair(p.element.Next())
}

// Prev returns a pointer to the previous pair.
func (p *KeyTypeValueTypePair) Prev() *KeyTypeValueTypePair {
	return listElementToKeyTypeValueTypePair(p.element.Prev())
}

func listElementToKeyTypeValueTypePair(element *list.Element) *KeyTypeValueTypePair {
	if element == nil {
		return nil
	}
	return element.Value.(*KeyTypeValueTypePair)
}
