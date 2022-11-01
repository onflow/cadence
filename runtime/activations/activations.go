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
	"github.com/onflow/cadence/runtime/common"
)

// Activation is a map of strings to values.
// It can be used to represent an active scope in a program,
// i.e. it can be used as a symbol table during semantic analysis,
// or as an activation record during interpretation or compilation.
type Activation[T any] struct {
	entries     map[string]T
	Depth       int
	Parent      *Activation[T]
	IsFunction  bool
	MemoryGauge common.MemoryGauge
}

func NewActivation[T any](memoryGauge common.MemoryGauge, parent *Activation[T]) *Activation[T] {
	var depth int
	if parent != nil {
		depth = parent.Depth + 1
	}

	common.UseMemory(memoryGauge, common.ActivationMemoryUsage)

	return &Activation[T]{
		Depth:       depth,
		Parent:      parent,
		MemoryGauge: memoryGauge,
	}
}

// Find returns the value for a given name in the activation.
// It returns nil if no value is found.
func (a *Activation[T]) Find(name string) (_ T) {

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

	return
}

// FunctionValues returns all values in the current function activation.
func (a *Activation[T]) FunctionValues() map[string]T {

	values := make(map[string]T)

	current := a

	for current != nil {

		if current.entries != nil {
			for name, value := range current.entries { //nolint:maprangecheck
				if _, ok := values[name]; !ok {
					values[name] = value
				}
			}
		}

		if current.IsFunction {
			break
		}

		current = current.Parent
	}

	return values
}

// Set sets the given name-value pair in the activation.
func (a *Activation[T]) Set(name string, value T) {
	if a.entries == nil {
		common.UseMemory(a.MemoryGauge, common.ActivationEntriesMemoryUsage)
		a.entries = make(map[string]T)
	}

	a.entries[name] = value
}

// Remove removes the given name from the activation.
func (a *Activation[T]) Remove(name string) {
	if a.entries == nil {
		return
	}

	delete(a.entries, name)
}

// Activations is a stack of activation records.
// Each entry represents a new activation record.
//
// The current / most nested activation record can be found
// at the top of the stack (see function `Current`).
type Activations[T any] struct {
	activations []*Activation[T]
	memoryGauge common.MemoryGauge
}

func NewActivations[T any](memoryGauge common.MemoryGauge) *Activations[T] {
	// No need to meter since activations list is created only once per execution.
	// However, memory gauge is needed here for caching, and using it
	// later to meter each activation and activation entries initialization.
	return &Activations[T]{
		memoryGauge: memoryGauge,
	}
}

// Current returns the current / most nested activation,
// which can be found at the top of the stack.
// It returns nil if there is no active activation.
func (a *Activations[T]) Current() *Activation[T] {
	count := len(a.activations)
	if count < 1 {
		return nil
	}
	return a.activations[count-1]
}

// Find returns the value for a given key in the current activation.
// It returns nil if no value is found
// or if there is no current activation.
func (a *Activations[T]) Find(name string) (_ T) {
	current := a.Current()
	if current == nil {
		return
	}
	return current.Find(name)
}

// Set sets the name-value pair in the current scope.
func (a *Activations[T]) Set(name string, value T) {
	current := a.Current()
	// create the first scope if there is no scope
	if current == nil {
		current = a.PushNewWithParent(nil)
	}

	current.Set(name, value)
}

// Remove removes the given name from the current scope.
func (a *Activations[T]) Remove(name string) {
	current := a.Current()
	if current == nil {
		return
	}

	current.Remove(name)
}

// PushNewWithParent pushes a new empty activation
// to the top of the activation stack.
// The new activation has the given parent as its parent.
func (a *Activations[T]) PushNewWithParent(parent *Activation[T]) *Activation[T] {
	activation := NewActivation(a.memoryGauge, parent)
	a.Push(activation)
	return activation
}

// PushNewWithCurrent pushes a new empty activation
// to the top of the activation stack.
// The new activation has the current activation as its parent.
func (a *Activations[T]) PushNewWithCurrent() {
	a.PushNewWithParent(a.Current())
}

// Push pushes the given activation
// onto the top of the activation stack.
func (a *Activations[T]) Push(activation *Activation[T]) {
	a.activations = append(
		a.activations,
		activation,
	)
}

// Pop pops the top-most (current) activation
// from the top of the activation stack.
func (a *Activations[T]) Pop() {
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
func (a *Activations[T]) CurrentOrNew() *Activation[T] {
	current := a.Current()
	if current == nil {
		return NewActivation[T](a.memoryGauge, nil)
	}

	return current
}

// Depth returns the depth (size) of the activation stack.
func (a *Activations[T]) Depth() int {
	return len(a.activations)
}
