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

// NodeSet is a set of Node
//
type NodeSet interface {
	Add(*Node)
	Remove(*Node)
	Contains(*Node) bool
	ForEach(func(*Node) error) error
}

// MapNodeSet is a Node set backed by a Go map (unordered)
//
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
	for node := range m {
		err := f(node)
		if err != nil {
			return err
		}
	}
	return nil
}

// OrderedNodeSet is a Node set backed by an ordered map
//
type OrderedNodeSet struct {
	m *NodeStructOrderedMap
}

func NewOrderedNodeSet() NodeSet {
	return OrderedNodeSet{
		m: NewNodeStructOrderedMap(),
	}
}

var _ NodeSet = OrderedNodeSet{}

func (o OrderedNodeSet) Add(node *Node) {
	o.m.Set(node, struct{}{})
}

func (o OrderedNodeSet) Remove(node *Node) {
	o.m.Delete(node)
}

func (o OrderedNodeSet) Contains(node *Node) bool {
	_, ok := o.m.Get(node)
	return ok
}

func (o OrderedNodeSet) ForEach(f func(*Node) error) error {
	return o.m.ForeachWithError(func(node *Node, _ struct{}) error {
		return f(node)
	})
}
