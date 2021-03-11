/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

func (interpreter *Interpreter) InvokeFunctionValue(
	function FunctionValue,
	arguments []Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	invocationRange ast.Range,
) (value Value, err error) {
	// recover internal panics and return them as an error
	defer recoverErrors(func(internalErr error) {
		err = internalErr
	})

	return interpreter.invokeFunctionValue(
		function,
		arguments,
		argumentTypes,
		parameterTypes,
		nil,
		invocationRange,
	), nil
}

func (interpreter *Interpreter) invokeFunctionValue(
	function FunctionValue,
	arguments []Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	typeParameterTypes *sema.TypeParameterTypeOrderedMap,
	invocationRange ast.Range,
) Value {

	parameterTypeCount := len(parameterTypes)
	argumentCopies := make([]Value, len(arguments))

	for i, argument := range arguments {
		argumentType := argumentTypes[i]
		if i < parameterTypeCount {
			parameterType := parameterTypes[i]
			argumentCopies[i] = interpreter.copyAndConvert(argument, argumentType, parameterType)
		} else {
			argumentCopies[i] = argument.Copy()
		}
	}

	// TODO: optimize: only potentially used by host-functions

	locationRange := LocationRange{
		Location: interpreter.Location,
		Range:    invocationRange,
	}

	invocation := Invocation{
		Arguments:          argumentCopies,
		ArgumentTypes:      argumentTypes,
		TypeParameterTypes: typeParameterTypes,
		LocationRange:      locationRange,
		Interpreter:        interpreter,
	}

	return function.Invoke(invocation)
}

func (interpreter *Interpreter) invokeInterpretedFunction(
	function InterpretedFunctionValue,
	invocation Invocation,
) Value {

	// Start a new activation record.
	// Lexical scope: use the function declaration's activation record,
	// not the current one (which would be dynamic scope)
	interpreter.activations.PushNewWithParent(function.Activation)

	// Make `self` available, if any
	if invocation.Self != nil {
		interpreter.declareVariable(sema.SelfIdentifier, invocation.Self)
	}

	return interpreter.invokeInterpretedFunctionActivated(function, invocation.Arguments)
}

// NOTE: assumes the function's activation (or an extension of it) is pushed!
//
func (interpreter *Interpreter) invokeInterpretedFunctionActivated(
	function InterpretedFunctionValue,
	arguments []Value,
) Value {
	defer interpreter.activations.Pop()

	if function.ParameterList != nil {
		interpreter.bindParameterArguments(function.ParameterList, arguments)
	}

	functionBlockTrampoline := interpreter.visitFunctionBody(
		function.BeforeStatements,
		function.PreConditions,
		func() interface{} {
			return interpreter.runAllStatements(interpreter.visitStatements(function.Statements))
		},
		function.PostConditions,
		function.Type.ReturnTypeAnnotation.Type,
	)

	return interpreter.runAllStatements(functionBlockTrampoline).(Value)
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
