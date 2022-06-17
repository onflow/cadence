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

import (
	"fmt"
)

// https://www.electricmonk.nl/docs/dependency_resolving_algorithm/dependency_resolving_algorithm.html

type CircularDependencyError struct {
	Dependent  *Node
	Dependency *Node
}

func (e CircularDependencyError) Error() string {
	return fmt.Sprintf(
		"circular dependency: %s -> %s",
		e.Dependent.Value,
		e.Dependency.Value,
	)
}

type Node struct {
	Value        interface{}
	dependents   NodeSet
	dependencies NodeSet
}

func NewNode(value interface{}, newNodeSet func() NodeSet) *Node {
	return &Node{
		Value:        value,
		dependents:   newNodeSet(),
		dependencies: newNodeSet(),
	}
}

func (n *Node) SetDependencies(dependencies ...*Node) {
	_ = n.dependencies.ForEach(func(dependency *Node) error {
		dependency.dependents.Remove(n)
		n.dependencies.Remove(dependency)
		return nil
	})

	for _, dependency := range dependencies {
		dependency.dependents.Add(n)
		n.dependencies.Add(dependency)
	}
}

func (n *Node) AllDependents() ([]*Node, error) {
	return n.solve(nil, MapNodeSet{}, MapNodeSet{})
}

func (n *Node) solve(solution []*Node, resolved, unresolved MapNodeSet) ([]*Node, error) {
	unresolved.Add(n)
	defer unresolved.Remove(n)

	err := n.dependents.ForEach(func(dependent *Node) error {
		if resolved.Contains(dependent) {
			return nil
		}

		if unresolved.Contains(dependent) {
			return CircularDependencyError{
				Dependent:  dependent,
				Dependency: n,
			}
		}

		var err error
		solution, err = dependent.solve(solution, resolved, unresolved)
		return err
	})
	if err != nil {
		return nil, err
	}

	resolved.Add(n)
	return append(solution, n), nil
}
