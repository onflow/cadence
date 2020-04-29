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

package trampoline

import (
	"github.com/onflow/cadence/runtime/errors"
)

// Based on "Stackless Scala With Free" by Rúnar Óli Bjarnason:
// http://blog.higher-order.com/assets/trampolines.pdf
///
/// Trampolines allow computations to be executed in constant stack space,
/// by trading it for heap space. They can be used for computations which
/// would otherwise use a large amount of stack space, potentially crashing
/// when the limited amount is exhausted (stack overflow).
///
/// A Trampoline represents a computation which consists of steps.
/// Each step is either more work which should be executed (`More`),
/// in the form of a function which returns the next step,
/// or a final value (`Done`), which indicates the end of the computation.
///
/// In trampolined programs, instead of each computation invoking
/// the next computation, (i.e., calling functions, possibly recursing directly),
/// they yield the next computation.
///
/// A trampoline consists of a current computation and next computation.
/// trampolines can be chained together using the `FlatMap` method and can be executed
/// through a control loop using the `Run` method.
///

type Trampoline interface {
	Resume() interface{}
	FlatMap(f func(interface{}) Trampoline) Trampoline
	Map(f func(interface{}) interface{}) Trampoline
	Then(f func(interface{})) Trampoline
}

// Run runs one Trampoline at a time, until there is no more continuation.
func Run(t Trampoline) interface{} {
	for {
		result := t.Resume()

		if continuation, ok := result.(func() Trampoline); ok {
			t = continuation()
			continue
		}

		return result
	}
}

func MapTrampoline(t Trampoline, f func(interface{}) interface{}) Trampoline {
	return t.FlatMap(func(value interface{}) Trampoline {
		return Done{Result: f(value)}
	})
}

func ThenTrampoline(t Trampoline, f func(interface{})) Trampoline {
	return t.Map(func(value interface{}) interface{} {
		f(value)
		return value
	})
}

// Done is a Trampoline, which has an executed result.

type Done struct {
	Result interface{}
}

func (d Done) Resume() interface{} {
	return d.Result
}

func (d Done) FlatMap(f func(interface{}) Trampoline) Trampoline {
	return FlatMap{Subroutine: d, Continuation: f}
}

func (d Done) Map(f func(interface{}) interface{}) Trampoline {
	return MapTrampoline(d, f)
}

func (d Done) Then(f func(interface{})) Trampoline {
	return ThenTrampoline(d, f)
}

type Continuation interface {
	Continue() Trampoline
}

// More is a Trampoline that returns a Trampoline as more work.

type More func() Trampoline

func (m More) Resume() interface{} {
	return (func() Trampoline)(m)
}

func (m More) FlatMap(f func(interface{}) Trampoline) Trampoline {
	return FlatMap{Subroutine: m, Continuation: f}
}

func (m More) Map(f func(interface{}) interface{}) Trampoline {
	return MapTrampoline(m, f)
}

func (m More) Then(f func(interface{})) Trampoline {
	return ThenTrampoline(m, f)
}

func (m More) Continue() Trampoline {
	return m()
}

// FlatMap is a struct that contains the current computation and the continuation computation
type FlatMap struct {
	Subroutine   Trampoline
	Continuation func(interface{}) Trampoline
}

func (m FlatMap) FlatMap(f func(interface{}) Trampoline) Trampoline {
	continuation := m.Continuation
	return FlatMap{
		Subroutine: m.Subroutine,
		Continuation: func(value interface{}) Trampoline {
			return continuation(value).FlatMap(f)
		},
	}
}

func (m FlatMap) Resume() interface{} {
	continuation := m.Continuation

	switch sub := m.Subroutine.(type) {
	case Done:
		// if the subroutine is done, then the result is ready to be used as input for the continuation
		return func() Trampoline {
			return continuation(sub.Result)
		}
	case Continuation:
		// if the subroutine is a continuation, then the result is not available yet, it has to call
		// sub.Continue() and use FlatMap to wait until the result is ready and be given the the
		// current continuation.
		return func() Trampoline {
			return sub.Continue().FlatMap(continuation)
		}
	case FlatMap:
		panic("FlatMap is not a valid subroutine. Use the FlatMap function to construct proper FlatMap structures.")
	}

	panic(errors.NewUnreachableError())
}

func (m FlatMap) Map(f func(interface{}) interface{}) Trampoline {
	return MapTrampoline(m, f)
}

func (m FlatMap) Then(f func(interface{})) Trampoline {
	return ThenTrampoline(m, f)
}
