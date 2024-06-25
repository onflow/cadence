/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
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

package interpreter

import (
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// Invocation
type Invocation struct {
	LocationRange      LocationRange
	Self               *Value
	Base               *EphemeralReferenceValue
	BoundAuthorization Authorization
	TypeParameterTypes *sema.TypeParameterTypeOrderedMap
	Interpreter        *Interpreter
	Arguments          []Value
	ArgumentTypes      []sema.Type
}

func NewInvocation(
	interpreter *Interpreter,
	self *Value,
	base *EphemeralReferenceValue,
	boundAuth Authorization,
	arguments []Value,
	argumentTypes []sema.Type,
	typeParameterTypes *sema.TypeParameterTypeOrderedMap,
	locationRange LocationRange,
) Invocation {
	common.UseMemory(interpreter, common.InvocationMemoryUsage)

	return Invocation{
		Self:               self,
		Base:               base,
		BoundAuthorization: boundAuth,
		Arguments:          arguments,
		ArgumentTypes:      argumentTypes,
		TypeParameterTypes: typeParameterTypes,
		LocationRange:      locationRange,
		Interpreter:        interpreter,
	}
}

// CallStack is the stack of invocations (call stack).
type CallStack struct {
	Invocations []Invocation
}

func (i *CallStack) Push(invocation Invocation) {
	i.Invocations = append(i.Invocations, invocation)
}

func (i *CallStack) Pop() {
	depth := len(i.Invocations)
	i.Invocations[depth-1] = Invocation{}
	i.Invocations = i.Invocations[:depth-1]
}
