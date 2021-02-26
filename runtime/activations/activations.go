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
	"github.com/onflow/cadence/runtime/common/orderedmap"
)

// An Activation is a map of strings to arbitrary values.
// It can be used to represent an active scope in a program,
// i.e. it can be used as a symbol table during semantic analysis,
// or as an activation record during interpretation.
//
type Activation struct {
	entries *orderedmap.StringInterfaceOrderedMap
	Depth   int
	Parent  *Activation
}

func NewActivation(parent *Activation) *Activation {
	var depth int
	if parent != nil {
		depth = parent.Depth + 1
	}
	return &Activation{
		Depth:  depth,
		Parent: parent,
	}
}

// Find returns the value for a given key in the activation.
// It returns nil if no value is found.
//
func (a *Activation) Find(name string) interface{} {
	if a.entries != nil {
		result, ok := a.entries.Get(name)
		if ok {
			return result
		}
	}

	if a.Parent != nil {
		return a.Parent.Find(name)
	}

	return nil
}

// Set sets the given key-value pair in the activation.
//
func (a *Activation) Set(name string, value interface{}) {
	if a.entries == nil {
		a.entries = orderedmap.NewStringInterfaceOrderedMap()
	}

	a.entries.Set(name, value)
}

// ForEach calls the given function for each entry (key-value pair) in the activation.
// It can be used to iterate over all entries of the activation.
//
func (a *Activation) ForEach(cb func(string, interface{}) error) error {

	activation := a

	for activation != nil {

		if activation.entries != nil {
			for pair := activation.entries.Oldest(); pair != nil; pair = pair.Next() {
				err := cb(pair.Key, pair.Value)
				if err != nil {
					return err
				}
			}
		}

		activation = activation.Parent
	}

	return nil
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
	activations []*Activation
}

// Current returns the current / most nested activation,
// which can be found at the top of the stack.
// It returns nil if there is no active activation.
//
func (a *Activations) Current() *Activation {
	count := len(a.activations)
	if count < 1 {
		return nil
	}
	return a.activations[count-1]
}

// Find returns the value for a given key in the current activation.
// It returns nil if no value is found
// or if there is no current activation.
//
func (a *Activations) Find(key string) interface{} {
	current := a.Current()
	if current == nil {
		return nil
	}
	return current.Find(key)
}

// Set sets the key value pair int the current scope.
//
func (a *Activations) Set(name string, value interface{}) {
	current := a.Current()
	// create the first scope if there is no scope
	if current == nil {
		current = a.PushNewWithParent(nil)
	}

	current.Set(name, value)
}

// PushNewWithParent pushes a new empty activation
// to the top of the activation stack.
// The new activation has the given parent as its parent.
//
func (a *Activations) PushNewWithParent(parent *Activation) *Activation {
	activation := NewActivation(parent)
	a.Push(activation)
	return activation
}

// PushNewWithCurrent pushes a new empty activation
// to the top of the activation stack.
// The new activation has the current activation as its parent.
//
func (a *Activations) PushNewWithCurrent() {
	a.PushNewWithParent(a.Current())
}

// Push pushes the given activation
// onto the top of the activation stack.
//
func (a *Activations) Push(activation *Activation) {
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
func (a *Activations) CurrentOrNew() *Activation {
	current := a.Current()
	if current == nil {
		return NewActivation(nil)
	}

	return current
}

// Depth returns the depth (size) of the activation stack.
//
func (a *Activations) Depth() int {
	return len(a.activations)
}
