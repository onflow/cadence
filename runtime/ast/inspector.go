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

package ast

// This file's code is heavily inspired by Go tools' go/ast/inspector/inspector.go

// Inspector provides methods for inspecting (traversing) an AST element.
//
// The inspecting methods allow element filtering by type,
// and materialization of the traversal stack.
//
// During construction, the inspector does a complete traversal
// and builds a list of push/pop events and their element type.
// Subsequent method calls that request a traversal scan this list,
// rather than walk the AST, and perform type filtering using efficient bit sets.
//
// Inspector's traversals are faster than using the Inspect function,
// but it may take multiple traversals for this benefit
// to amortize the inspector's construction cost.
// If efficiency is the primary concern,
// do not use Inspector for one-off traversals.
//
// There are four orthogonal features in a traversal:
//  1 type filtering
//  2 pruning
//  3 postorder calls to f
//  4 stack
//
// Rather than offer all of them in the API,
// only a few combinations are exposed:
// - Preorder is the fastest and has the fewest features,
//   but is the most commonly needed traversal.
// - Elements and WithStack both provide pruning and postorder calls,
//   even though few clients need it, because supporting two versions
//   is not justified.
//
// More combinations could be supported  by expressing them as wrappers
// around a more generic traversal, but likely has worse performance.
//
type Inspector struct {
	events []event
}

// NewInspector returns an Inspector for the specified AST element.
func NewInspector(element Element) *Inspector {
	return &Inspector{traverse(element)}
}

// An event represents a push or a pop
// of an Element during a traversal.
type event struct {
	element Element
	typ     uint64 // 1 << element.ElementType()
	index   int    // 1 + index of corresponding pop event, or 0 if this is a pop
}

// Preorder visits all elements in depth-first order.
// It calls f(e) for each element e before it visits e's children.
//
// The types argument, if non-empty, enables type-based filtering of events.
// The function f if is called only for elements whose type matches an element of the types slice.
//
// Preorder is almost twice as fast as Elements,
// because it avoids postorder calls to f, and the pruning check.
//
func (in *Inspector) Preorder(types []Element, f func(Element)) {
	mask := maskOf(types)
	for _, ev := range in.events {
		if ev.typ&mask != 0 && ev.index > 0 {
			f(ev.element)
		}
	}
}

// Elements visits the elements in depth-first order.
// It calls f(e, true) for each element e before it visits e's children.
// If f returns true, Elements invokes f recursively
// for each of the non-nil children of the element,
// followed by a call of f(n, false).
//
// The types argument, if non-empty, enables type-based filtering of events.
// The function f if is called only for elements whose type matches an element of the types slice.
//
func (in *Inspector) Elements(types []Element, f func(element Element, push bool) (proceed bool)) {
	mask := maskOf(types)
	for i := 0; i < len(in.events); {
		ev := in.events[i]
		if ev.typ&mask != 0 {
			if ev.index > 0 {
				// push
				if !f(ev.element, true) {
					i = ev.index // jump to corresponding pop + 1
					continue
				}
			} else {
				// pop
				f(ev.element, false)
			}
		}
		i++
	}
}

// WithStack visits elements in a similar manner to Elements,
// but it supplies each call to f an additional argument,
// the current traversal stack.
//
// The stack's first element is the outermost element, its last is the innermost.
//
func (in *Inspector) WithStack(types []Element, f func(element Element, push bool, stack []Element) (proceed bool)) {
	mask := maskOf(types)
	var stack []Element
	for i := 0; i < len(in.events); {
		ev := in.events[i]
		if ev.index > 0 {
			// push
			stack = append(stack, ev.element)
			if ev.typ&mask != 0 {
				if !f(ev.element, true, stack) {
					i = ev.index
					stack = stack[:len(stack)-1]
					continue
				}
			}
		} else {
			// pop
			if ev.typ&mask != 0 {
				f(ev.element, false, stack)
			}
			stack = stack[:len(stack)-1]
		}
		i++
	}
}

// traverse builds the table of events representing a traversal.
func traverse(element Element) []event {

	// TODO: estimate capacity
	events := make([]event, 0)

	var stack []event

	Inspect(element, func(element Element) bool {
		if element != nil {
			// push
			ev := event{
				element: element,
				typ:     1 << element.ElementType(),
				index:   len(events), // push event temporarily holds own index
			}
			stack = append(stack, ev)
			events = append(events, ev)
		} else {
			// pop
			ev := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			events[ev.index].index = len(events) + 1 // make push refer to pop

			ev.index = 0 // turn ev into a pop event
			events = append(events, ev)
		}
		return true
	})

	return events
}

func maskOf(elements []Element) uint64 {
	if elements == nil {
		return 1<<64 - 1 // match all node types
	}
	var mask uint64
	for _, element := range elements {
		mask |= 1 << element.ElementType()
	}
	return mask
}
