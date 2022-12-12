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

package deps

import "github.com/onflow/cadence/runtime/common/orderedmap"

// NodeSet is a set of Node
type NodeSet interface {
	Add(*Node)
	Remove(*Node)
	Contains(*Node) bool
	ForEach(func(*Node) error) error
}

// MapNodeSet is a Node set backed by a Go map (unordered).
// NOTE: DO *NOT* USE this for on-chain operations, but OrderedNodeSet
type MapNodeSet map[*Node]struct{}

func NewMapNodeSet() NodeSet {
	return MapNodeSet{}
}

var _ NodeSet = MapNodeSet{}

func (m MapNodeSet) Add(node *Node) {
	m[node] = struct{}{}
}

func (m MapNodeSet) Remove(node *Node) {
	delete(m, node)
}

func (m MapNodeSet) Contains(node *Node) bool {
	_, ok := m[node]
	return ok
}

func (m MapNodeSet) ForEach(f func(*Node) error) error {
	for node := range m { // nolint:maprange
		err := f(node)
		if err != nil {
			return err
		}
	}
	return nil
}

// OrderedNodeSet is a Node set backed by an ordered map
type OrderedNodeSet orderedmap.OrderedMap[*Node, struct{}]

var _ NodeSet = &OrderedNodeSet{}

func (os *OrderedNodeSet) Add(node *Node) {
	om := (*orderedmap.OrderedMap[*Node, struct{}])(os)
	om.Set(node, struct{}{})
}

func (os *OrderedNodeSet) Remove(node *Node) {
	om := (*orderedmap.OrderedMap[*Node, struct{}])(os)
	om.Delete(node)
}

func (os *OrderedNodeSet) Contains(node *Node) bool {
	om := (*orderedmap.OrderedMap[*Node, struct{}])(os)
	return om.Contains(node)
}

func (os *OrderedNodeSet) ForEach(f func(*Node) error) error {
	om := (*orderedmap.OrderedMap[*Node, struct{}])(os)

	return om.ForeachWithError(func(node *Node, _ struct{}) error {
		return f(node)
	})
}
