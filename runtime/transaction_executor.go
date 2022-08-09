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
	"github.com/onflow/cadence/runtime/interpreter"
)

type interpreterTransactionExecutor struct {
	runtime interpreterRuntime

	script  Script
	context Context

	preprocessOnce sync.Once
	preprocessErr  error

	program       *interpreter.Program
	interpretFunc InterpretFunc

	authorizers []Address

	executeOnce sync.Once
	executeErr  error
}

func newInterpreterTransactionExecutor(
	runtime interpreterRuntime,
	script Script,
	context Context,
) Executor {

	return &interpreterTransactionExecutor{
		runtime: runtime,
		script:  script,
		context: context,
	}
}

// Transaction's preprocessing which could be done in parallel with other
// transactions / scripts.
func (executor *interpreterTransactionExecutor) Preprocess() error {
	executor.preprocessOnce.Do(func() {
		executor.preprocessErr = executor.preprocess()
	})

	return executor.preprocessErr
}

func (executor *interpreterTransactionExecutor) Execute() error {
	executor.executeOnce.Do(func() {
		executor.executeErr = executor.execute()
	})

	return executor.executeErr
}

func (executor *interpreterTransactionExecutor) Result() (cadence.Value, error) {
	return nil, executor.Execute()
}

func (executor *interpreterTransactionExecutor) preprocess() (err error) {
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

	executor.program, err = executor.context.Environment.ParseAndCheckProgram(
		executor.script.Source,
		location,
		true,
	)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	transactions := executor.program.Elaboration.TransactionTypes
	transactionCount := len(transactions)
	if transactionCount != 1 {
		err = InvalidTransactionCountError{
			Count: transactionCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	transactionType := transactions[0]

	var authorizers []Address
	wrapPanic(func() {
		authorizers, err = executor.context.Interface.GetSigningAccounts()
	})
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}
	// check parameter count

	argumentCount := len(executor.script.Arguments)
	authorizerCount := len(authorizers)

	transactionParameterCount := len(transactionType.Parameters)
	if argumentCount != transactionParameterCount {
		err = InvalidEntryPointParameterCountError{
			Expected: transactionParameterCount,
			Actual:   argumentCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	transactionAuthorizerCount := len(transactionType.PrepareParameters)
	if authorizerCount != transactionAuthorizerCount {
		err = InvalidTransactionAuthorizerCountError{
			Expected: transactionAuthorizerCount,
			Actual:   authorizerCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	// gather authorizers

	authorizerValues := func(inter *interpreter.Interpreter) []interpreter.Value {

		authorizerValues := make([]interpreter.Value, 0, authorizerCount)

		for _, address := range authorizers {
			addressValue := interpreter.NewAddressValue(inter, address)
			authorizerValues = append(
				authorizerValues,
				executor.context.Environment.NewAuthAccountValue(addressValue),
			)
		}

		return authorizerValues
	}

	executor.interpretFunc = executor.runtime.transactionExecutionFunction(
		transactionType.Parameters,
		executor.script.Arguments,
		executor.context.Interface,
		authorizerValues,
	)

	return nil
}

func (executor *interpreterTransactionExecutor) execute() (err error) {
	err = executor.Preprocess()
	if err != nil {
		return err
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

	_, inter, err := executor.context.Environment.Interpret(
		location,
		executor.program,
		executor.interpretFunc,
	)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = executor.context.Environment.CommitStorage(inter)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}
	return nil
}
