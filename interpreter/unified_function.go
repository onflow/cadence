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
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

// minimal interfaces needed by all native/host functions
type UnifiedFunctionContext interface {
	ValueStaticTypeContext
	ValueTransferContext
	StaticTypeConversionHandler
	InvocationContext
}

type UnifiedNativeFunction func(
	context UnifiedFunctionContext,
	locationRange LocationRange,
	typeArguments []StaticType,
	receiver Value,
	args ...Value,
) Value

// These are all the functions that need to exist to work with the interpreter
func AdaptUnifiedFunctionForInterpreter(fn UnifiedNativeFunction) HostFunction {
	return func(invocation Invocation) Value {
		context := invocation.InvocationContext

		var receiver Value
		if invocation.Self != nil {
			receiver = *invocation.Self
		}

		// convert TypeParameterTypes to []StaticType
		var typeArguments []StaticType
		if invocation.TypeParameterTypes != nil {
			typeArguments = make([]StaticType, 0, invocation.TypeParameterTypes.Len())
			invocation.TypeParameterTypes.Foreach(func(key *sema.TypeParameter, semaType sema.Type) {
				staticType := ConvertSemaToStaticType(context, semaType)
				typeArguments = append(typeArguments, staticType)
			})
		}

		result := fn(context, invocation.LocationRange, typeArguments, receiver, invocation.Arguments...)

		return result
	}
}

func NewUnifiedStaticHostFunctionValue(
	context InvocationContext,
	functionType *sema.FunctionType,
	fn UnifiedNativeFunction,
) *HostFunctionValue {
	return NewStaticHostFunctionValue(
		context,
		functionType,
		AdaptUnifiedFunctionForInterpreter(fn),
	)
}

func NewUnifiedBoundHostFunctionValue(
	context FunctionCreationContext,
	self Value,
	funcType *sema.FunctionType,
	function UnifiedNativeFunction,
) BoundFunctionValue {

	// wrap the unified function to work with the standard HostFunction signature
	// just like how we do it in the interpreter
	wrappedFunction := AdaptUnifiedFunctionForInterpreter(function)

	hostFunc := NewStaticHostFunctionValue(context, funcType, wrappedFunction)

	return NewBoundFunctionValue(
		context,
		hostFunc,
		&self,
		nil,
	)
}

// generic helper function to assert that the provided value is of a specific type
// useful for asserting receiver and argument types in unified functions
func assertValueOfType[T Value](val Value) T {
	value, ok := val.(T)
	if !ok {
		panic(errors.NewUnreachableError())
	}
	return value
}
