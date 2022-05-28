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

package interpreter

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

func (interpreter *Interpreter) InvokeFunctionValue(
	function FunctionValue,
	arguments []Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	invocationPosition ast.HasPosition,
) (
	value Value,
	err error,
) {

	// recover internal panics and return them as an error
	defer interpreter.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	return interpreter.invokeFunctionValue(
		function,
		arguments,
		nil,
		argumentTypes,
		parameterTypes,
		nil,
		invocationPosition,
	), nil
}

func (interpreter *Interpreter) invokeFunctionValue(
	function FunctionValue,
	arguments []Value,
	expressions []ast.Expression,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	typeParameterTypes *sema.TypeParameterTypeOrderedMap,
	invocationPosition ast.HasPosition,
) Value {

	parameterTypeCount := len(parameterTypes)
	transferredArguments := make([]Value, len(arguments))

	for i, argument := range arguments {
		argumentType := argumentTypes[i]

		var locationPos ast.HasPosition
		if i < len(expressions) {
			locationPos = expressions[i]
		} else {
			locationPos = invocationPosition
		}

		getLocationRange := locationRangeGetter(interpreter, interpreter.Location, locationPos)

		if i < parameterTypeCount {
			parameterType := parameterTypes[i]
			transferredArguments[i] = interpreter.transferAndConvert(
				argument,
				argumentType,
				parameterType,
				getLocationRange,
			)
		} else {
			transferredArguments[i] = argument.Transfer(
				interpreter,
				getLocationRange,
				atree.Address{},
				false,
				nil,
			)
		}
	}

	getLocationRange := locationRangeGetter(interpreter, interpreter.Location, invocationPosition)

	invocation := NewInvocation(
		interpreter,
		nil,
		transferredArguments,
		argumentTypes,
		typeParameterTypes,
		getLocationRange,
	)

	return function.invoke(invocation)
}

func (interpreter *Interpreter) invokeInterpretedFunction(
	function *InterpretedFunctionValue,
	invocation Invocation,
) Value {

	// Start a new activation record.
	// Lexical scope: use the function declaration's activation record,
	// not the current one (which would be dynamic scope)
	interpreter.activations.PushNewWithParent(function.Activation)
	interpreter.activations.Current().isFunction = true

	// Make `self` available, if any
	if invocation.Self != nil {
		interpreter.declareVariable(sema.SelfIdentifier, invocation.Self)
	}

	return interpreter.invokeInterpretedFunctionActivated(function, invocation.Arguments)
}

// NOTE: assumes the function's activation (or an extension of it) is pushed!
//
func (interpreter *Interpreter) invokeInterpretedFunctionActivated(
	function *InterpretedFunctionValue,
	arguments []Value,
) Value {
	defer interpreter.activations.Pop()

	if function.ParameterList != nil {
		interpreter.bindParameterArguments(function.ParameterList, arguments)
	}

	return interpreter.visitFunctionBody(
		function.BeforeStatements,
		function.PreConditions,
		func() controlReturn {
			return interpreter.visitStatements(function.Statements)
		},
		function.PostConditions,
		function.Type.ReturnTypeAnnotation.Type,
	)
}

// bindParameterArguments binds the argument values to the given parameters
//
func (interpreter *Interpreter) bindParameterArguments(
	parameterList *ast.ParameterList,
	arguments []Value,
) {
	for parameterIndex, parameter := range parameterList.Parameters {
		argument := arguments[parameterIndex]
		interpreter.declareVariable(parameter.Identifier.Identifier, argument)
	}
}
