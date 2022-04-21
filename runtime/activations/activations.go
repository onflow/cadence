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

package activations

import (
	"github.com/cheekybits/genny/generic"
)

type ValueType generic.Type

// A ValueTypeActivation is a map of strings to values.
// It can be used to represent an active scope in a program,
// i.e. it can be used as a symbol table during semantic analysis,
// or as an activation record during interpretation or compilation.
//
type ValueTypeActivation struct {
	entries    map[string]ValueType
	Depth      int
	Parent     *ValueTypeActivation
	isFunction bool
}

func NewValueTypeActivation(parent *ValueTypeActivation) *ValueTypeActivation {
	var depth int
	if parent != nil {
		depth = parent.Depth + 1
	}
	return &ValueTypeActivation{
		Depth:  depth,
		Parent: parent,
	}
}

// Find returns the value for a given name in the activation.
// It returns nil if no value is found.
//
func (a *ValueTypeActivation) Find(name string) ValueType {

	current := a

	for current != nil {

		if current.entries != nil {
			result, ok := current.entries[name]
			if ok {
				return result
			}
		}

		current = current.Parent
	}

	return nil
}

// FunctionValues returns all values in the current function activation.
//
func (a *ValueTypeActivation) FunctionValues() map[string]ValueType {

	values := make(map[string]ValueType)

	current := a

	for current != nil {

		if current.entries != nil {
			for name, value := range current.entries { //nolint:maprangecheck
				if _, ok := values[name]; !ok {
					values[name] = value
				}
			}
		}

		if current.isFunction {
			break
		}

		current = current.Parent
	}

	return values
}

// Set sets the given name-value pair in the activation.
//
func (a *ValueTypeActivation) Set(name string, value ValueType) {
	if a.entries == nil {
		a.entries = make(map[string]ValueType)
	}

	a.entries[name] = value
}

// ValueTypeActivations is a stack of activation records.
// Each entry represents a new activation record.
//
// The current / most nested activation record can be found
// at the top of the stack (see function `Current`).
//
type ValueTypeActivations struct {
	activations []*ValueTypeActivation
}

// Current returns the current / most nested activation,
// which can be found at the top of the stack.
// It returns nil if there is no active activation.
//
func (a *ValueTypeActivations) Current() *ValueTypeActivation {
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
func (a *ValueTypeActivations) Find(name string) ValueType {
	current := a.Current()
	if current == nil {
		return nil
	}
	return current.Find(name)
}

// Set sets the name-value pair in the current scope.
//
func (a *ValueTypeActivations) Set(name string, value ValueType) {
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
func (a *ValueTypeActivations) PushNewWithParent(parent *ValueTypeActivation) *ValueTypeActivation {
	activation := NewValueTypeActivation(parent)
	a.Push(activation)
	return activation
}

// PushNewWithCurrent pushes a new empty activation
// to the top of the activation stack.
// The new activation has the current activation as its parent.
//
func (a *ValueTypeActivations) PushNewWithCurrent() {
	a.PushNewWithParent(a.Current())
}

// Push pushes the given activation
// onto the top of the activation stack.
//
func (a *ValueTypeActivations) Push(activation *ValueTypeActivation) {
	a.activations = append(
		a.activations,
		activation,
	)
}

// Pop pops the top-most (current) activation
// from the top of the activation stack.
//
func (a *ValueTypeActivations) Pop() {
	count := len(a.activations)
	if count < 1 {
		return
	}
	lastIndex := count - 1
	a.activations[lastIndex] = nil
	a.activations = a.activations[:lastIndex]
}

// CurrentOrNew returns the current activation,
// or if it does not exist, a new activation
//
func (a *ValueTypeActivations) CurrentOrNew() *ValueTypeActivation {
	current := a.Current()
	if current == nil {
		return NewValueTypeActivation(nil)
	}

	return current
}

// Depth returns the depth (size) of the activation stack.
//
func (a *ValueTypeActivations) Depth() int {
	return len(a.activations)
}
