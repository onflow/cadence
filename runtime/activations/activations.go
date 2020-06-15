/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

package activations

import (
	"github.com/raviqqe/hamt"

	"github.com/onflow/cadence/runtime/common"
)

// Activations is a stack of activation records.
// Each entry represents a new scope.
//
type Activations struct {
	activations []hamt.Map
}

func (a *Activations) current() *hamt.Map {
	count := len(a.activations)
	if count < 1 {
		return nil
	}
	current := a.activations[count-1]
	return &current
}

func (a *Activations) Find(key string) interface{} {
	current := a.current()
	if current == nil {
		return nil
	}
	return current.Find(common.StringEntry(key))
}

// Set adds the new key value pair to the current scope.
// The current scope is updated in an immutable way.
func (a *Activations) Set(name string, value interface{}) {
	current := a.current()
	// create the first scope if there is no scope
	if current == nil {
		a.PushCurrent()
		current = &a.activations[0]
	}

	count := len(a.activations)
	// update the current scope in an immutable way,
	// which builds on top of the old "current" activation value
	// without mutating it.
	a.activations[count-1] = current.
		Insert(common.StringEntry(name), value)
}

// PushCurrent makes a copy of the current activation, and pushes it to
// the top of the activation stack, so that the `Find` method only needs to
// look up a certain record by name from the current activation record
// without having to go through each activation in the stack.
func (a *Activations) PushCurrent() {
	current := a.current()
	if current == nil {
		first := hamt.NewMap()
		current = &first
	}
	a.Push(*current)
}

func (a *Activations) Push(activation hamt.Map) {
	a.activations = append(
		a.activations,
		activation,
	)
}

func (a *Activations) Pop() {
	count := len(a.activations)
	if count < 1 {
		return
	}
	a.activations = a.activations[:count-1]
}

func (a *Activations) CurrentOrNew() hamt.Map {
	current := a.current()
	if current == nil {
		return hamt.NewMap()
	}

	return *current
}

func (a *Activations) Depth() int {
	return len(a.activations)
}
