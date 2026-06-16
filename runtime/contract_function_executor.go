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

package runtime

import (
	"sync"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type contractFunctionExecutor struct {
	context          Context
	environment      Environment
	result           cadence.Value
	executeErr       error
	preprocessErr    error
	codesAndPrograms CodesAndPrograms
	runtime          Runtime
	storage          *Storage
	contractLocation common.AddressLocation
	functionName     string
	arguments        []cadence.Value
	argumentTypes    []sema.Type
	executeOnce      sync.Once
	preprocessOnce   sync.Once
	vm               *vm.VM
}

func newContractFunctionExecutor(
	runtime Runtime,
	contractLocation common.AddressLocation,
	functionName string,
	arguments []cadence.Value,
	argumentTypes []sema.Type,
	context Context,
) *contractFunctionExecutor {
	return &contractFunctionExecutor{
		runtime:          runtime,
		contractLocation: contractLocation,
		functionName:     functionName,
		arguments:        arguments,
		argumentTypes:    argumentTypes,
		context:          context,
	}
}

func (executor *contractFunctionExecutor) Preprocess() error {
	executor.preprocessOnce.Do(func() {
		executor.preprocessErr = executor.preprocess()
	})

	return executor.preprocessErr
}

func (executor *contractFunctionExecutor) Execute() error {
	executor.executeOnce.Do(func() {
		executor.result, executor.executeErr = executor.execute()
	})

	return executor.executeErr
}

func (executor *contractFunctionExecutor) Result() (cadence.Value, error) {
	// Note: Execute's error is saved into executor.executeErr and return in
	// the next line.
	_ = executor.Execute()
	return executor.result, executor.executeErr
}

func (executor *contractFunctionExecutor) preprocess() (err error) {
	context := executor.context
	location := context.Location

	codesAndPrograms := NewCodesAndPrograms()
	executor.codesAndPrograms = codesAndPrograms

	defer Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	runtimeInterface := context.Interface

	config := executor.runtime.Config()

	storage := NewStorage(
		runtimeInterface,
		context.MemoryGauge,
		context.ComputationGauge,
		StorageConfig{},
	)
	executor.storage = storage

	environment := context.Environment
	if environment == nil {
		if context.UseVM {
			environment = NewBaseVMEnvironment(config)
		} else {
			environment = NewBaseInterpreterEnvironment(config)
		}
	}

	environment.Configure(
		runtimeInterface,
		codesAndPrograms,
		storage,
		context.MemoryGauge,
		context.ComputationGauge,
	)
	executor.environment = environment

	switch environment := environment.(type) {
	case *InterpreterEnvironment:
		if context.UseVM {
			panic(errors.NewUnexpectedError(
				"expected to run with the VM, but found an incompatible environment: %T",
				environment,
			))
		}

		// NO-OP

	case *vmEnvironment:
		if !context.UseVM {
			panic(errors.NewUnexpectedError(
				"expected to run with the interpreter, but found an incompatible environment: %T",
				environment,
			))
		}
		contractLocation := executor.contractLocation
		program := environment.importProgram(contractLocation)
		executor.vm = environment.newVM(contractLocation, program)

	default:
		panic(errors.NewUnexpectedError("unsupported environment: %T", environment))
	}

	return nil
}

func (executor *contractFunctionExecutor) execute() (val cadence.Value, err error) {
	err = executor.Preprocess()
	if err != nil {
		return nil, err
	}

	environment := executor.environment
	codesAndPrograms := executor.codesAndPrograms

	defer Recover(
		func(internalErr Error) {
			err = internalErr
		},
		executor.context.Location,
		codesAndPrograms,
	)

	switch environment := environment.(type) {
	case *InterpreterEnvironment:
		value, err := executor.executeWithInterpreter(environment)
		if err != nil {
			return nil, newError(err, executor.context.Location, codesAndPrograms)
		}
		return value, nil

	case *vmEnvironment:
		value, err := executor.executeWithVM(environment)
		if err != nil {
			return nil, newError(err, executor.context.Location, codesAndPrograms)
		}
		return value, nil

	default:
		panic(errors.NewUnexpectedError("unsupported environment: %T", environment))
	}
}

func (executor *contractFunctionExecutor) executeWithInterpreter(
	environment *InterpreterEnvironment,
) (val cadence.Value, err error) {

	location := executor.context.Location

	// create interpreter
	_, inter, err := environment.Interpret(
		location,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// ensure the contract is loaded
	inter = inter.EnsureLoaded(executor.contractLocation)

	arguments, err := convertArguments(
		executor.environment,
		inter,
		executor.arguments,
		executor.argumentTypes,
	)
	if err != nil {
		return nil, err
	}

	contractValue := inter.GetContractComposite(executor.contractLocation)
	if contractValue == nil {
		return nil, interpreter.NotDeclaredError{
			ExpectedKind: common.DeclarationKindContract,
			Name:         executor.contractLocation.Name,
		}
	}

	var self interpreter.Value = contractValue

	contractMember := contractValue.GetMember(
		inter,
		executor.functionName,
		common.DeclarationKindFunction,

		// Calling `GetMember` on the composite value, not on a reference value.
		// Also, used internally, and the function-value is not moved around.
		// Therefore, "accessedReference" is `nil`.
		nil,
	)

	contractFunction, ok := contractMember.(interpreter.FunctionValue)
	if !ok {
		err := interpreter.NotInvokableError{
			Value: contractFunction,
		}
		return nil, err
	}

	returnType := contractFunction.FunctionType(inter).ReturnTypeAnnotation.Type

	// prepare invocation
	invocation := interpreter.NewInvocation(
		inter,
		&self,
		nil,
		arguments,
		executor.argumentTypes,
		nil,
		returnType,
		interpreter.LocationRange{
			Location:    location,
			HasPosition: ast.EmptyRange,
		},
	)

	value, err := interpreter.InvokeFunction(inter, contractFunction, invocation)
	if err != nil {
		return nil, err
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.commitStorage(inter)
	if err != nil {
		return nil, err
	}

	var exportedValue cadence.Value
	exportedValue, err = ExportValue(value, inter)
	if err != nil {
		return nil, err
	}

	return exportedValue, nil
}

func (executor *contractFunctionExecutor) executeWithVM(
	environment *vmEnvironment,
) (val cadence.Value, err error) {

	contractLocation := executor.contractLocation

	context := executor.vm.Context()

	contractValue := loadContractValue(
		context,
		contractLocation,
		environment.storage,
	)
	if contractValue == nil {
		return nil, interpreter.NotDeclaredError{
			ExpectedKind: common.DeclarationKindContract,
			Name:         executor.contractLocation.Name,
		}
	}

	// receiver + arguments
	arguments, err := convertArguments(
		executor.environment,
		context,
		executor.arguments,
		executor.argumentTypes,
	)
	if err != nil {
		return nil, err
	}

	staticType := contractValue.StaticType(context)
	semaType := context.SemaTypeFromStaticType(staticType)
	qualifiedFuncName := commons.TypeQualifiedName(semaType, executor.functionName)

	value, err := executor.vm.InvokeMethodExternally(
		qualifiedFuncName,
		contractValue,
		arguments...,
	)
	if err != nil {
		return nil, err
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.commitStorage(context)
	if err != nil {
		return nil, err
	}

	var exportedValue cadence.Value
	exportedValue, err = ExportValue(value, context)
	if err != nil {
		return nil, err
	}

	return exportedValue, nil
}

type ArgumentConversionContext interface {
	interpreter.AccountCreationContext
}

// convertArguments converts the given arguments to interpreter values,
// using `context` for value construction and `environment` for importing values.
//
// `environment` may be nil. In that case, only arguments whose conversion
// does not require an environment are supported; any other argument type returns an error.
func convertArguments(
	environment Environment,
	context ArgumentConversionContext,
	arguments []cadence.Value,
	argumentTypes []sema.Type,
) ([]interpreter.Value, error) {
	convertedArguments := make([]interpreter.Value, 0, len(arguments))

	for i, argumentType := range argumentTypes {
		convertedArgument, err := convertArgument(
			environment,
			context,
			arguments[i],
			argumentType,
		)
		if err != nil {
			return nil, err
		}
		convertedArguments = append(convertedArguments, convertedArgument)
	}

	return convertedArguments, nil
}

// convertArgument converts a single argument to an interpreter value.
//
// `environment` may be nil. In that case, only arguments whose conversion
// does not require an environment are supported; any other argument type returns an error.
func convertArgument(
	environment Environment,
	context ArgumentConversionContext,
	argument cadence.Value,
	argumentType sema.Type,
) (interpreter.Value, error) {

	// Convert `Address` arguments to account reference values (`&Account`) if that is the expected argument type,
	// so there is no need for the caller to construct the value.
	accountReferenceValue := convertAccountReferenceArgument(context, argument, argumentType)
	if accountReferenceValue != nil {
		return accountReferenceValue, nil
	}

	// Importing any other argument type requires an environment.
	if environment == nil {
		return nil, errors.NewDefaultUserError(
			"cannot convert argument of type %s without an environment",
			argumentType.QualifiedString(),
		)
	}

	return ImportValue(
		context,
		environment,
		environment.ResolveLocation,
		argument,
		argumentType,
	)
}

// convertAccountReferenceArgument converts an `Address` argument to an account reference value (`&Account`)
// when the expected argument type is an `&Account` reference, and returns it.
//
// Returns nil if the argument is not an `Address`, or the expected type is not an `&Account` reference,
// in which case no conversion is performed.
func convertAccountReferenceArgument(
	context ArgumentConversionContext,
	argument cadence.Value,
	argumentType sema.Type,
) interpreter.Value {
	address, ok := argument.(cadence.Address)
	if !ok {
		return nil
	}

	referenceType, ok := argumentType.(*sema.ReferenceType)
	if !ok || referenceType.Type != sema.AccountType {
		return nil
	}

	return newAccountReferenceValueFromAddress(
		context,
		common.Address(address),
		referenceType.Authorization,
	)
}

// InvokeContractFunctionOnContext invokes a function of a contract using the supplied,
// already-executing invocation `context`, so that the call SHARES the same storage as that context.
//
// Unlike Runtime.InvokeContractFunction, this helper does NOT create a new Storage
// and does NOT commit storage: the outer program that owns `context` is responsible for committing.
// This is what lets a host (e.g. FVM) run a system-contract function as part of account creation
// against the same atree storage as the transaction that triggered it,
// so the writes are not lost to a separate, independently-committed storage instance.
//
// Arguments are converted using the same conversion as a regular contract-function invocation,
// but WITHOUT an environment. Therefore, only arguments whose conversion does not require an environment are supported:
// Any argument that requires an environment for ImportValue results in an error.
func InvokeContractFunctionOnContext(
	context interpreter.InvocationContext,
	contractLocation common.AddressLocation,
	functionName string,
	arguments []cadence.Value,
	argumentTypes []sema.Type,
) (val cadence.Value, err error) {

	defer context.RecoverErrors(func(internalErr error) {
		err = internalErr
	})

	// Reuse the regular argument conversion, without an environment (see the doc comment above for the
	// resulting limitation on supported argument types).
	convertedArguments, err := convertArguments(nil, context, arguments, argumentTypes)
	if err != nil {
		return nil, err
	}

	contractValue := context.GetContractValue(contractLocation)
	if contractValue == nil {
		return nil, interpreter.NotDeclaredError{
			ExpectedKind: common.DeclarationKindContract,
			Name:         contractLocation.Name,
		}
	}

	function := context.GetMethod(contractValue, functionName, nil)
	if function == nil {
		return nil, interpreter.NotDeclaredError{
			ExpectedKind: common.DeclarationKindFunction,
			Name:         functionName,
		}
	}

	functionType := function.FunctionType(context)

	result, err := interpreter.InvokeFunctionValue(
		context,
		function,
		convertedArguments,
		argumentTypes,
		functionType.ParameterTypes(),
		functionType.ReturnTypeAnnotation.Type,
	)
	if err != nil {
		return nil, err
	}

	return ExportValue(result, context)
}
