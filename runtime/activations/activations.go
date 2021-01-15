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

// Package activations implements data structures that can be used
// when dealing with program scopes.
//
package activations

import (
	"github.com/raviqqe/hamt"

	"github.com/onflow/cadence/runtime/common"
)

// An Activation is an immutable map of strings to arbitrary values.
// It can be used to represent an active scope in a program,
// i.e. it can be used as a symbol table during semantic analysis,
// or as an activation record during interpretation.
//
type Activation hamt.Map

func NewActivation() Activation {
	return Activation(hamt.NewMap())
}

// FirstRest returns the first entry (key-value pair) in the activation,
// and the remaining entries in the activation.
// It can be used to iterate over all entries of the activation.
//
func (a Activation) FirstRest() (string, interface{}, Activation) {
	entry, value, rest := hamt.Map(a).FirstRest()
	if entry == nil {
		return "", nil, Activation{}
	}

	name := string(entry.(common.StringEntry))
	return name, value, Activation(rest)
}

// Find returns the value for a given key in the activation.
// It returns nil if no value is found.
//
func (a Activation) Find(name string) interface{} {
	return hamt.Map(a).Find(common.StringEntry(name))
}

// Insert inserts the given key-value pair into the activation.
//
func (a Activation) Insert(name string, value interface{}) Activation {
	return Activation(hamt.Map(a).Insert(common.StringEntry(name), value))
}

// Activations is a stack of activation records.
// Each entry represents a new ac.
//
// The current / most nested activation record can be found
// at the top of the stack (see function `current`).
//
// Each activation in the stack contains *all* active records –
// there is no need to traverse to parent records.
// This is implemented efficiently by using immutable maps
// that share data with their parents.
//
type Activations struct {
	activations []Activation
}

// current returns the current / most nested activation,
// which can be found at the top of the stack.
// It returns nil if there is no active activation.
//
func (a *Activations) current() *Activation {
	count := len(a.activations)
	if count < 1 {
		return nil
	}
	current := a.activations[count-1]
	return &current
}

// Find returns the value for a given key in the current activation.
// It returns nil if no value is found
// or if there is no current activation.
//
func (a *Activations) Find(key string) interface{} {
	current := a.current()
	if current == nil {
		return nil
	}
	return current.Find(key)
}

// Set adds the new key value pair to the current activation.
// The current activation is updated in an immutable way.
//
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
	a.activations[count-1] = current.Insert(name, value)
}

// PushCurrent makes a copy of the current activation,
// and pushes it to the top of the activation stack,
// so that the `Find` method only needs to look up a certain record by name
// from the current activation record,
// without having to go through each activation in the stack.
//
func (a *Activations) PushCurrent() {
	current := a.current()
	if current == nil {
		first := NewActivation()
		current = &first
	}
	a.Push(*current)
}

// Push pushes the given activation
// onto the top of the activation stack.
//
func (a *Activations) Push(activation Activation) {
	a.activations = append(
		a.activations,
		activation,
	)
}

// Pop pops the top-most (current) activation
// from the top of the activation stack.
//
func (a *Activations) Pop() {
	count := len(a.activations)
	if count < 1 {
		return
	}
	a.activations = a.activations[:count-1]
}

// CurrentOrNew returns the current activation,
// or if it does not exists, a new activation
//
func (a *Activations) CurrentOrNew() Activation {
	current := a.current()
	if current == nil {
		return NewActivation()
	}

	return *current
}

// Depth returns the depth (size) of the activation stack.
//
func (a *Activations) Depth() int {
	return len(a.activations)
}
