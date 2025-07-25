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
	"github.com/onflow/atree"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/sema"
)

func InvokeFunctionValue(
	context InvocationContext,
	function FunctionValue,
	arguments []Value,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	returnType sema.Type,
	invocationPosition ast.HasPosition,
) (
	value Value,
	err error,
) {

	// recover internal panics and return them as an error
	defer context.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	return invokeFunctionValue(
		context,
		function,
		arguments,
		nil,
		argumentTypes,
		parameterTypes,
		returnType,
		nil,
		invocationPosition,
	), nil
}

func invokeFunctionValue(
	context InvocationContext,
	function FunctionValue,
	arguments []Value,
	expressions []ast.Expression,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	returnType sema.Type,
	typeParameterTypes *sema.TypeParameterTypeOrderedMap,
	invocationPosition ast.HasPosition,
) Value {
	return invokeFunctionValueWithEval(
		context,
		function,
		arguments,
		func(argument Value) Value {
			return argument
		},
		nil, // no implicit argument
		expressions,
		argumentTypes,
		parameterTypes,
		returnType,
		typeParameterTypes,
		invocationPosition,
	)
}

func invokeFunctionValueWithEval[T any](
	context InvocationContext,
	function FunctionValue,
	arguments []T,
	evaluate func(T) Value,
	implicitArgumentValue Value,
	expressions []ast.Expression,
	argumentTypes []sema.Type,
	parameterTypes []sema.Type,
	returnType sema.Type,
	typeParameterTypes *sema.TypeParameterTypeOrderedMap,
	invocationPosition ast.HasPosition,
) Value {

	parameterTypeCount := len(parameterTypes)

	var transferredArguments []Value

	location := context.GetLocation()

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
				Location:    location,
				HasPosition: locationPos,
			}

			argumentValue := evaluate(argument)

			if i < parameterTypeCount {
				parameterType := parameterTypes[i]
				transferredArguments[i] = TransferAndConvert(
					context,
					argumentValue,
					argumentType,
					parameterType,
					locationRange,
				)
			} else {
				transferredArguments[i] = argumentValue.Transfer(
					context,
					locationRange,
					atree.Address{},
					false,
					nil,
					nil,
					true, // argument is standalone.
				)
			}
		}
	}

	// add the implicit argument to the end of the argument list, if it exists
	if implicitArgumentValue != nil {
		transferredImplicitArgument := implicitArgumentValue.Transfer(
			context,
			LocationRange{
				Location:    location,
				HasPosition: invocationPosition,
			},
			atree.Address{},
			false,
			nil,
			nil,
			true, // argument is standalone.
		)
		transferredArguments = append(transferredArguments, transferredImplicitArgument)
		argumentType := MustSemaTypeOfValue(implicitArgumentValue, context)
		argumentTypes = append(argumentTypes, argumentType)
	}

	locationRange := LocationRange{
		Location:    location,
		HasPosition: invocationPosition,
	}

	invocation := NewInvocation(
		context,
		nil,
		nil,
		transferredArguments,
		argumentTypes,
		typeParameterTypes,
		locationRange,
	)

	resultValue := function.Invoke(invocation)

	functionReturnType := function.FunctionType(context).ReturnTypeAnnotation.Type

	// Only convert and box.
	// No need to transfer, since transfer would happen later, when the return value gets assigned.
	//
	// The conversion is needed because, the runtime function's return type could be a
	// subtype of the invocation's return type.
	// e.g:
	//   struct interface I {
	//     fun foo(): T?
	//   }
	//
	//   struct S: I {
	//     fun foo(): T {...}
	//   }
	//
	//   var i: {I} = S()
	//   return i.foo()?.bar
	//
	// Here runtime function's return type is `T`, but invocation's return type is `T?`.

	return ConvertAndBox(
		context,
		locationRange,
		resultValue,
		functionReturnType,
		returnType,
	)
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
		interpreter.declareSelfVariable(*invocation.Self, invocation.LocationRange)
	}
	if invocation.Base != nil {
		interpreter.declareVariable(sema.BaseIdentifier, invocation.Base)
	}

	return interpreter.invokeInterpretedFunctionActivated(function, invocation.Arguments, invocation.LocationRange)
}

// NOTE: assumes the function's activation (or an extension of it) is pushed!
func (interpreter *Interpreter) invokeInterpretedFunctionActivated(
	function *InterpretedFunctionValue,
	arguments []Value,
	declarationLocationRange LocationRange,
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
		declarationLocationRange,
	)
}

// bindParameterArguments binds the argument values to the given parameters
func (interpreter *Interpreter) bindParameterArguments(
	parameterList *ast.ParameterList,
	arguments []Value,
) {
	for parameterIndex, parameter := range parameterList.Parameters {
		argument := arguments[parameterIndex]
		interpreter.declareVariable(parameter.Identifier.Identifier, argument)
	}
}
