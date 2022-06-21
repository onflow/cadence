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

package deps

import "container/list"

// NodeStructOrderedMap
//
type NodeStructOrderedMap struct {
	pairs map[*Node]*NodeStructPair
	list  *list.List
}

// NewNodeStructOrderedMap creates a new NodeStructOrderedMap.
func NewNodeStructOrderedMap() *NodeStructOrderedMap {
	return &NodeStructOrderedMap{
		pairs: make(map[*Node]*NodeStructPair),
		list:  list.New(),
	}
}

// Clear removes all entries from this ordered map.
func (om *NodeStructOrderedMap) Clear() {
	om.list.Init()
	// NOTE: Range over map is safe, as it is only used to delete entries
	for key := range om.pairs { //nolint:maprangecheck
		delete(om.pairs, key)
	}
}

// Get returns the value associated with the given key.
// Returns nil if not found.
// The second return value indicates if the key is present in the map.
func (om *NodeStructOrderedMap) Get(key *Node) (result struct{}, present bool) {
	var pair *NodeStructPair
	if pair, present = om.pairs[key]; present {
		return pair.Value, present
	}
	return
}

// GetPair returns the key-value pair associated with the given key.
// Returns nil if not found.
func (om *NodeStructOrderedMap) GetPair(key *Node) *NodeStructPair {
	return om.pairs[key]
}

// Set sets the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Set`.
func (om *NodeStructOrderedMap) Set(key *Node, value struct{}) (oldValue struct{}, present bool) {
	var pair *NodeStructPair
	if pair, present = om.pairs[key]; present {
		oldValue = pair.Value
		pair.Value = value
		return
	}

	pair = &NodeStructPair{
		Key:   key,
		Value: value,
	}
	pair.element = om.list.PushBack(pair)
	om.pairs[key] = pair

	return
}

// Delete removes the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Delete`.
func (om *NodeStructOrderedMap) Delete(key *Node) (oldValue struct{}, present bool) {
	var pair *NodeStructPair
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
func (om *NodeStructOrderedMap) Len() int {
	return len(om.pairs)
}

// Oldest returns a pointer to the oldest pair.
func (om *NodeStructOrderedMap) Oldest() *NodeStructPair {
	return listElementToNodeStructPair(om.list.Front())
}

// Newest returns a pointer to the newest pair.
func (om *NodeStructOrderedMap) Newest() *NodeStructPair {
	return listElementToNodeStructPair(om.list.Back())
}

// Foreach iterates over the entries of the map in the insertion order, and invokes
// the provided function for each key-value pair.
func (om *NodeStructOrderedMap) Foreach(f func(key *Node, value struct{})) {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		f(pair.Key, pair.Value)
	}
}

// ForeachWithError iterates over the entries of the map in the insertion order,
// and invokes the provided function for each key-value pair.
// If the passed function returns an error, iteration breaks and the error is returned.
func (om *NodeStructOrderedMap) ForeachWithError(f func(key *Node, value struct{}) error) error {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		err := f(pair.Key, pair.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// NodeStructPair
//
type NodeStructPair struct {
	Key   *Node
	Value struct{}

	element *list.Element
}

// Next returns a pointer to the next pair.
func (p *NodeStructPair) Next() *NodeStructPair {
	return listElementToNodeStructPair(p.element.Next())
}

// Prev returns a pointer to the previous pair.
func (p *NodeStructPair) Prev() *NodeStructPair {
	return listElementToNodeStructPair(p.element.Prev())
}

func listElementToNodeStructPair(element *list.Element) *NodeStructPair {
	if element == nil {
		return nil
	}
	return element.Value.(*NodeStructPair)
}
