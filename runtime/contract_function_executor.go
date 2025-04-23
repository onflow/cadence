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

	storage := NewStorage(
		runtimeInterface,
		runtimeInterface,
		StorageConfig{},
	)
	executor.storage = storage

	environment := context.Environment
	if environment == nil {
		if context.UseVM {
			environment = NewBaseVMEnvironment(executor.runtime.Config())
		} else {
			environment = NewBaseInterpreterEnvironment(executor.runtime.Config())
		}
	}
	environment.Configure(
		runtimeInterface,
		codesAndPrograms,
		storage,
		context.CoverageReport,
	)
	executor.environment = environment

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
	case *interpreterEnvironment:
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
	environment *interpreterEnvironment,
) (val cadence.Value, err error) {

	location := executor.context.Location

	// create interpreter
	_, inter, err := environment.interpret(
		location,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// ensure the contract is loaded
	inter = inter.EnsureLoaded(executor.contractLocation)

	arguments := make([]interpreter.Value, len(executor.arguments))

	arguments, err = executor.appendArguments(inter, arguments)
	if err != nil {
		return nil, err
	}

	contractValue, err := inter.GetContractComposite(executor.contractLocation)
	if err != nil {
		return nil, err
	}

	var self interpreter.Value = contractValue

	// prepare invocation
	invocation := interpreter.NewInvocation(
		inter,
		&self,
		nil,
		arguments,
		executor.argumentTypes,
		nil,
		interpreter.LocationRange{
			Location:    location,
			HasPosition: ast.EmptyRange,
		},
	)

	contractMember := contractValue.GetMember(
		inter,
		invocation.LocationRange,
		executor.functionName,
	)

	contractFunction, ok := contractMember.(interpreter.FunctionValue)
	if !ok {
		err := interpreter.NotInvokableError{
			Value: contractFunction,
		}
		return nil, err
	}

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
	exportedValue, err = ExportValue(value, inter, interpreter.EmptyLocationRange)
	if err != nil {
		return nil, err
	}

	return exportedValue, nil
}

func (executor *contractFunctionExecutor) executeWithVM(
	environment *vmEnvironment,
) (val cadence.Value, err error) {

	contractLocation := executor.contractLocation

	contractProgram, err := environment.loadProgram(contractLocation)
	if err != nil {
		return nil, err
	}

	compiledProgram := environment.compileProgram(contractProgram, contractLocation)

	vm := environment.newVM(contractLocation, compiledProgram)

	context := vm.Context()

	contractValue := loadContractValue(
		context,
		contractLocation,
		environment.storage,
	)

	// receiver + arguments
	invocationArguments := make([]interpreter.Value, 1+len(executor.arguments))
	invocationArguments[0] = contractValue

	invocationArguments, err = executor.appendArguments(context, invocationArguments)
	if err != nil {
		return nil, err
	}

	staticType := contractValue.StaticType(context)
	semaType := interpreter.MustConvertStaticToSemaType(staticType, context)
	typeQualifier := commons.TypeQualifier(semaType)
	qualifiedFuncName := commons.TypeQualifiedName(typeQualifier, executor.functionName)

	value, err := vm.Invoke(qualifiedFuncName, invocationArguments...)
	if err != nil {
		return nil, err
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.commitStorage(context)
	if err != nil {
		return nil, err
	}

	var exportedValue cadence.Value
	exportedValue, err = ExportValue(value, context, interpreter.EmptyLocationRange)
	if err != nil {
		return nil, err
	}

	return exportedValue, nil
}

type ArgumentConversionContext interface {
	interpreter.AccountCreationContext
}

func (executor *contractFunctionExecutor) convertArgument(
	context ArgumentConversionContext,
	argument cadence.Value,
	argumentType sema.Type,
	locationRange interpreter.LocationRange,
) (interpreter.Value, error) {
	environment := executor.environment

	// Convert `Address` arguments to account reference values (`&Account`)
	// if it is the expected argument type,
	// so there is no need for the caller to construct the value

	if address, ok := argument.(cadence.Address); ok {

		if referenceType, ok := argumentType.(*sema.ReferenceType); ok &&
			referenceType.Type == sema.AccountType {

			accountReferenceValue := newAccountReferenceValueFromAddress(
				context,
				common.Address(address),
				environment,
				referenceType.Authorization,
				locationRange,
			)

			return accountReferenceValue, nil
		}
	}

	return ImportValue(
		context,
		locationRange,
		environment,
		environment.ResolveLocation,
		argument,
		argumentType,
	)
}

func (executor *contractFunctionExecutor) appendArguments(
	context ArgumentConversionContext,
	arguments []interpreter.Value,
) (
	[]interpreter.Value,
	error,
) {
	locationRange := interpreter.LocationRange{
		Location:    executor.context.Location,
		HasPosition: ast.EmptyRange,
	}

	for i, argumentType := range executor.argumentTypes {
		argument, err := executor.convertArgument(
			context,
			executor.arguments[i],
			argumentType,
			locationRange,
		)
		if err != nil {
			return nil, err
		}
		arguments = append(arguments, argument)
	}

	return arguments, nil
}
