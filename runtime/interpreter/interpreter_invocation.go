/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

	var transferredArguments []Value

	argumentCount := len(arguments)
	if argumentCount > 0 {
		transferredArguments = make([]Value, argumentCount)

		for i, argument := range arguments {
			argumentType := argumentTypes[i]

			var locationPos ast.HasPosition
			if i < len(expressions) {
				locationPos = expressions[i]
			} else {
				locationPos = invocationPosition
			}

			locationRange := LocationRange{
				Location:    interpreter.Location,
				HasPosition: locationPos,
			}

			if i < parameterTypeCount {
				parameterType := parameterTypes[i]
				transferredArguments[i] = interpreter.transferAndConvert(
					argument,
					argumentType,
					parameterType,
					locationRange,
				)
			} else {
				transferredArguments[i] = argument.Transfer(
					interpreter,
					locationRange,
					atree.Address{},
					false,
					nil,
					nil,
				)
			}
		}
	}

	locationRange := LocationRange{
		Location:    interpreter.Location,
		HasPosition: invocationPosition,
	}

	invocation := NewInvocation(
		interpreter,
		nil,
		nil,
		nil,
		transferredArguments,
		argumentTypes,
		typeParameterTypes,
		locationRange,
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
	current := interpreter.activations.PushNewWithParent(function.Activation)
	current.IsFunction = true

	interpreter.SharedState.callStack.Push(invocation)

	// Make `self` available, if any
	if invocation.Self != nil {
		interpreter.declareVariable(sema.SelfIdentifier, *invocation.Self)
	}
	if invocation.Base != nil {
		interpreter.declareVariable(sema.BaseIdentifier, invocation.Base)
	}
	if invocation.BoundAuthorization != nil {
		oldInvocationValue := interpreter.SharedState.currentEntitlementMappedValue
		interpreter.SharedState.currentEntitlementMappedValue = invocation.BoundAuthorization
		defer func() {
			interpreter.SharedState.currentEntitlementMappedValue = oldInvocationValue
		}()
	}

	return interpreter.invokeInterpretedFunctionActivated(function, invocation.Arguments)
}

// NOTE: assumes the function's activation (or an extension of it) is pushed!
func (interpreter *Interpreter) invokeInterpretedFunctionActivated(
	function *InterpretedFunctionValue,
	arguments []Value,
) Value {
	defer func() {
		// Only unwind the call stack if there was no error
		if r := recover(); r != nil {
			panic(r)
		}
		interpreter.SharedState.callStack.Pop()
	}()
	defer interpreter.activations.Pop()

	if function.ParameterList != nil {
		interpreter.bindParameterArguments(function.ParameterList, arguments)
	}

	return interpreter.visitFunctionBody(
		function.BeforeStatements,
		function.PreConditions,
		func() StatementResult {
			return interpreter.visitStatements(function.Statements)
		},
		function.PostConditions,
		function.Type.ReturnTypeAnnotation.Type,
	)
}

// bindParameterArguments binds the argument values to the given parameters.
// the handling of default arguments makes a number of assumptions to simplify the implementation;
// namely that a) all default arguments are lazily evaluated at the site of the invocation,
// b) that either all the parameters or none of the parameters of a function have default arguments,
// and c) functions cannot currently be explicitly invoked if they have default arguments
// if we plan to generalize this further, we will need to relax those assumptions
func (interpreter *Interpreter) bindParameterArguments(
	parameterList *ast.ParameterList,
	arguments []Value,
) {
	parameters := parameterList.Parameters

	if len(parameters) < 1 {
		return
	}

	// if the first parameter has a default arg, all of them do, and the arguments list is empty
	if parameters[0].DefaultArgument != nil {
		// lazily evaluate the default argument expression in this context
		for _, parameter := range parameters {
			defaultArg := interpreter.evalExpression(parameter.DefaultArgument)
			arguments = append(arguments, defaultArg)
		}
	}

	for parameterIndex, parameter := range parameters {
		argument := arguments[parameterIndex]
		interpreter.declareVariable(parameter.Identifier.Identifier, argument)
	}
}
