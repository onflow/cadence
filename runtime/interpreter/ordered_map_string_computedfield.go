// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

package interpreter

import "container/list"

// StringComputedFieldOrderedMap
//
type StringComputedFieldOrderedMap struct {
	pairs map[string]*StringComputedFieldPair
	list  *list.List
}

// NewStringComputedFieldOrderedMap creates a new StringComputedFieldOrderedMap.
func NewStringComputedFieldOrderedMap() *StringComputedFieldOrderedMap {
	return &StringComputedFieldOrderedMap{
		pairs: make(map[string]*StringComputedFieldPair),
		list:  list.New(),
	}
}

// Get returns the value associated with the given key.
// Returns nil if not found.
// The second return value indicates if the key is present in the map.
func (om *StringComputedFieldOrderedMap) Get(key string) (result ComputedField, present bool) {
	var pair *StringComputedFieldPair
	if pair, present = om.pairs[key]; present {
		return pair.Value, present
	}
	return
}

// GetPair returns the key-value pair associated with the given key.
// Returns nil if not found.
func (om *StringComputedFieldOrderedMap) GetPair(key string) *StringComputedFieldPair {
	return om.pairs[key]
}

// Set sets the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Set`.
func (om *StringComputedFieldOrderedMap) Set(key string, value ComputedField) (oldValue ComputedField, present bool) {
	var pair *StringComputedFieldPair
	if pair, present = om.pairs[key]; present {
		oldValue = pair.Value
		pair.Value = value
		return
	}

	pair = &StringComputedFieldPair{
		Key:   key,
		Value: value,
	}
	pair.element = om.list.PushBack(pair)
	om.pairs[key] = pair

	return
}

// Delete removes the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Delete`.
func (om *StringComputedFieldOrderedMap) Delete(key string) (oldValue ComputedField, present bool) {
	var pair *StringComputedFieldPair
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
func (om *StringComputedFieldOrderedMap) Len() int {
	return len(om.pairs)
}

// Oldest returns a pointer to the oldest pair.
func (om *StringComputedFieldOrderedMap) Oldest() *StringComputedFieldPair {
	return listElementToStringComputedFieldPair(om.list.Front())
}

// Newest returns a pointer to the newest pair.
func (om *StringComputedFieldOrderedMap) Newest() *StringComputedFieldPair {
	return listElementToStringComputedFieldPair(om.list.Back())
}

// Foreach iterates over the entries of the map in the insertion order, and invokes
// the provided function for each key-value pair.
func (om *StringComputedFieldOrderedMap) Foreach(f func(key string, value ComputedField)) {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		f(pair.Key, pair.Value)
	}
}

// StringComputedFieldPair
//
type StringComputedFieldPair struct {
	Key   string
	Value ComputedField

	element *list.Element
}

// Next returns a pointer to the next pair.
func (p *StringComputedFieldPair) Next() *StringComputedFieldPair {
	return listElementToStringComputedFieldPair(p.element.Next())
}

// Prev returns a pointer to the previous pair.
func (p *StringComputedFieldPair) Prev() *StringComputedFieldPair {
	return listElementToStringComputedFieldPair(p.element.Prev())
}

func listElementToStringComputedFieldPair(element *list.Element) *StringComputedFieldPair {
	if element == nil {
		return nil
	}
	return element.Value.(*StringComputedFieldPair)
}
