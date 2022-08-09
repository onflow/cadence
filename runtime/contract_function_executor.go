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

package runtime

import (
	"sync"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

type interpreterContractFunctionExecutor struct {
	runtime interpreterRuntime

	contractLocation common.AddressLocation
	functionName     string
	arguments        []cadence.Value
	argumentTypes    []sema.Type
	context          Context

	preprocessOnce sync.Once
	preprocessErr  error

	executeOnce sync.Once
	executeErr  error
	result      cadence.Value
}

func newInterpreterContractFunctionExecutor(
	runtime interpreterRuntime,
	contractLocation common.AddressLocation,
	functionName string,
	arguments []cadence.Value,
	argumentTypes []sema.Type,
	context Context,
) *interpreterContractFunctionExecutor {
	return &interpreterContractFunctionExecutor{
		runtime:          runtime,
		contractLocation: contractLocation,
		functionName:     functionName,
		arguments:        arguments,
		argumentTypes:    argumentTypes,
		context:          context,
	}
}

func (executor *interpreterContractFunctionExecutor) Preprocess() error {
	executor.preprocessOnce.Do(func() {
		executor.preprocessErr = executor.preprocess()
	})

	return executor.preprocessErr
}

func (executor *interpreterContractFunctionExecutor) Execute() error {
	executor.executeOnce.Do(func() {
		executor.result, executor.executeErr = executor.execute()
	})

	return executor.executeErr
}

func (executor *interpreterContractFunctionExecutor) Result() (cadence.Value, error) {
	// Note: Execute's error is saved into executor.executeErr and return in
	// the next line.
	_ = executor.Execute()
	return executor.result, executor.executeErr
}

func (executor *interpreterContractFunctionExecutor) preprocess() (err error) {
	location := executor.context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer executor.runtime.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	storage := NewStorage(executor.context.Interface, executor.context.Interface)

	if executor.context.Environment == nil {
		executor.context.Environment = NewBaseInterpreterEnvironment(executor.runtime.defaultConfig)
	}
	executor.context.Environment.Configure(
		executor.context.Interface,
		codesAndPrograms,
		storage,
		executor.context.CoverageReport,
	)

	return nil
}

func (executor *interpreterContractFunctionExecutor) execute() (val cadence.Value, err error) {
	err = executor.Preprocess()
	if err != nil {
		return nil, err
	}

	location := executor.context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer executor.runtime.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	// create interpreter
	_, inter, err := executor.context.Environment.Interpret(
		location,
		nil,
		nil,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// ensure the contract is loaded
	inter = inter.EnsureLoaded(executor.contractLocation)

	interpreterArguments := make([]interpreter.Value, len(executor.arguments))

	for i, argumentType := range executor.argumentTypes {

		ia, err := executor.convertArgument(
			inter,
			executor.arguments[i],
			argumentType,
			executor.context.Environment,
			func() interpreter.LocationRange {
				return interpreter.LocationRange{
					Location: executor.context.Location,
				}
			},
		)
		if err != nil {
			return nil, newError(err, location, codesAndPrograms)
		}
		interpreterArguments[i] = ia
	}

	contractValue, err := inter.GetContractComposite(executor.contractLocation)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// prepare invocation
	invocation := interpreter.NewInvocation(
		inter,
		contractValue,
		interpreterArguments,
		executor.argumentTypes,
		nil,
		func() interpreter.LocationRange {
			return interpreter.LocationRange{
				Location: executor.context.Location,
			}
		},
	)

	contractMember := contractValue.GetMember(inter, invocation.GetLocationRange, executor.functionName)

	contractFunction, ok := contractMember.(interpreter.FunctionValue)
	if !ok {
		err := interpreter.NotInvokableError{
			Value: contractFunction,
		}
		return nil, newError(err, location, codesAndPrograms)
	}

	value, err := inter.InvokeFunction(contractFunction, invocation)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = executor.context.Environment.CommitStorage(inter)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	var exportedValue cadence.Value
	exportedValue, err = ExportValue(value, inter, interpreter.ReturnEmptyLocationRange)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	return exportedValue, nil
}

func (executor *interpreterContractFunctionExecutor) convertArgument(
	inter *interpreter.Interpreter,
	argument cadence.Value,
	argumentType sema.Type,
	environment Environment,
	getLocationRange func() interpreter.LocationRange,
) (interpreter.Value, error) {
	switch argumentType {
	case sema.AuthAccountType:
		// convert addresses to auth accounts so there is no need to construct an auth account value for the caller
		if addressValue, ok := argument.(cadence.Address); ok {
			address := interpreter.NewAddressValueFromConstructor(
				inter,
				func() common.Address {
					return common.Address(addressValue)
				})
			return environment.NewAuthAccountValue(address), nil
		}
	case sema.PublicAccountType:
		// convert addresses to public accounts so there is no need to construct a public account value for the caller
		if addressValue, ok := argument.(cadence.Address); ok {
			address := interpreter.NewAddressValueFromConstructor(
				inter,
				func() common.Address {
					return common.Address(addressValue)
				})
			return environment.NewPublicAccountValue(address), nil
		}
	}
	return importValue(
		inter,
		getLocationRange,
		argument,
		argumentType,
	)
}
