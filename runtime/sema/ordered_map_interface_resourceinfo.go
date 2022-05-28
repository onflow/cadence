// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

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

package sema

import "container/list"

// InterfaceResourceInfoOrderedMap
//
type InterfaceResourceInfoOrderedMap struct {
	pairs map[interface{}]*InterfaceResourceInfoPair
	list  *list.List
}

// NewInterfaceResourceInfoOrderedMap creates a new InterfaceResourceInfoOrderedMap.
func NewInterfaceResourceInfoOrderedMap() *InterfaceResourceInfoOrderedMap {
	return &InterfaceResourceInfoOrderedMap{
		pairs: make(map[interface{}]*InterfaceResourceInfoPair),
		list:  list.New(),
	}
}

// Clear removes all entries from this ordered map.
func (om *InterfaceResourceInfoOrderedMap) Clear() {
	om.list.Init()
	// NOTE: Range over map is safe, as it is only used to delete entries
	for key := range om.pairs { //nolint:maprangecheck
		delete(om.pairs, key)
	}
}

// Get returns the value associated with the given key.
// Returns nil if not found.
// The second return value indicates if the key is present in the map.
func (om *InterfaceResourceInfoOrderedMap) Get(key interface{}) (result ResourceInfo, present bool) {
	var pair *InterfaceResourceInfoPair
	if pair, present = om.pairs[key]; present {
		return pair.Value, present
	}
	return
}

// GetPair returns the key-value pair associated with the given key.
// Returns nil if not found.
func (om *InterfaceResourceInfoOrderedMap) GetPair(key interface{}) *InterfaceResourceInfoPair {
	return om.pairs[key]
}

// Set sets the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Set`.
func (om *InterfaceResourceInfoOrderedMap) Set(key interface{}, value ResourceInfo) (oldValue ResourceInfo, present bool) {
	var pair *InterfaceResourceInfoPair
	if pair, present = om.pairs[key]; present {
		oldValue = pair.Value
		pair.Value = value
		return
	}

	pair = &InterfaceResourceInfoPair{
		Key:   key,
		Value: value,
	}
	pair.element = om.list.PushBack(pair)
	om.pairs[key] = pair

	return
}

// Delete removes the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Delete`.
func (om *InterfaceResourceInfoOrderedMap) Delete(key interface{}) (oldValue ResourceInfo, present bool) {
	var pair *InterfaceResourceInfoPair
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
func (om *InterfaceResourceInfoOrderedMap) Len() int {
	return len(om.pairs)
}

// Oldest returns a pointer to the oldest pair.
func (om *InterfaceResourceInfoOrderedMap) Oldest() *InterfaceResourceInfoPair {
	return listElementToInterfaceResourceInfoPair(om.list.Front())
}

// Newest returns a pointer to the newest pair.
func (om *InterfaceResourceInfoOrderedMap) Newest() *InterfaceResourceInfoPair {
	return listElementToInterfaceResourceInfoPair(om.list.Back())
}

// Foreach iterates over the entries of the map in the insertion order, and invokes
// the provided function for each key-value pair.
func (om *InterfaceResourceInfoOrderedMap) Foreach(f func(key interface{}, value ResourceInfo)) {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		f(pair.Key, pair.Value)
	}
}

// ForeachWithError iterates over the entries of the map in the insertion order,
// and invokes the provided function for each key-value pair.
// If the passed function returns an error, iteration breaks and the error is returned.
func (om *InterfaceResourceInfoOrderedMap) ForeachWithError(f func(key interface{}, value ResourceInfo) error) error {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		err := f(pair.Key, pair.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// InterfaceResourceInfoPair
//
type InterfaceResourceInfoPair struct {
	Key   interface{}
	Value ResourceInfo

	element *list.Element
}

// Next returns a pointer to the next pair.
func (p *InterfaceResourceInfoPair) Next() *InterfaceResourceInfoPair {
	return listElementToInterfaceResourceInfoPair(p.element.Next())
}

// Prev returns a pointer to the previous pair.
func (p *InterfaceResourceInfoPair) Prev() *InterfaceResourceInfoPair {
	return listElementToInterfaceResourceInfoPair(p.element.Prev())
}

func listElementToInterfaceResourceInfoPair(element *list.Element) *InterfaceResourceInfoPair {
	if element == nil {
		return nil
	}
	return element.Value.(*InterfaceResourceInfoPair)
}
