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
	"github.com/onflow/cadence/runtime/sema"
)

type interpreterTransactionExecutor struct {
	runtime interpreterRuntime
	script  Script
	context Context

	// prepare
	preprocessOnce   sync.Once
	preprocessErr    error
	codesAndPrograms codesAndPrograms
	storage          *Storage
	program          *interpreter.Program
	environment      Environment
	transactionType  *sema.TransactionType
	authorizers      []Address

	// execute
	executeOnce sync.Once
	executeErr  error
	interpret   InterpretFunc
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
	context := executor.context
	location := context.Location
	script := executor.script

	codesAndPrograms := newCodesAndPrograms()
	executor.codesAndPrograms = codesAndPrograms

	interpreterRuntime := executor.runtime

	defer interpreterRuntime.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	runtimeInterface := context.Interface

	storage := NewStorage(runtimeInterface, runtimeInterface)
	executor.storage = storage

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(interpreterRuntime.defaultConfig)
	}
	environment.Configure(
		runtimeInterface,
		codesAndPrograms,
		storage,
		context.CoverageReport,
	)
	executor.environment = environment

	program, err := environment.ParseAndCheckProgram(
		script.Source,
		location,
		true,
	)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}
	executor.program = program

	transactions := program.Elaboration.TransactionTypes
	transactionCount := len(transactions)
	if transactionCount != 1 {
		err = InvalidTransactionCountError{
			Count: transactionCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	transactionType := transactions[0]
	executor.transactionType = transactionType

	var authorizers []Address
	wrapPanic(func() {
		authorizers, err = runtimeInterface.GetSigningAccounts()
	})
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}
	executor.authorizers = authorizers

	// check parameter count

	argumentCount := len(script.Arguments)
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

	executor.interpret = transactionExecutionFunction(
		transactionType.Parameters,
		script.Arguments,
		context.Interface,
		executor.authorizerValues,
	)

	return nil
}

func (executor *interpreterTransactionExecutor) authorizerValues(inter *interpreter.Interpreter) []interpreter.Value {

	// gather authorizers

	authorizerValues := make([]interpreter.Value, 0, len(executor.authorizers))

	for _, address := range executor.authorizers {
		addressValue := interpreter.NewAddressValue(inter, address)
		authorizerValues = append(
			authorizerValues,
			executor.environment.NewAuthAccountValue(addressValue),
		)
	}

	return authorizerValues
}

func (executor *interpreterTransactionExecutor) execute() (err error) {
	err = executor.Preprocess()
	if err != nil {
		return err
	}

	environment := executor.environment
	context := executor.context
	location := context.Location
	codesAndPrograms := executor.codesAndPrograms
	interpreterRuntime := executor.runtime

	defer interpreterRuntime.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	_, inter, err := environment.Interpret(
		location,
		executor.program,
		executor.interpret,
	)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.CommitStorage(inter)
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	return nil
}

func transactionExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
	authorizerValues func(*interpreter.Interpreter) []interpreter.Value,
) InterpretFunc {
	return func(inter *interpreter.Interpreter) (value interpreter.Value, err error) {

		// Recover internal panics and return them as an error.
		// For example, the argument validation might attempt to
		// load contract code for non-existing types

		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		values, err := validateArgumentParams(
			inter,
			runtimeInterface,
			interpreter.ReturnEmptyLocationRange,
			arguments,
			parameters,
		)
		if err != nil {
			return nil, err
		}

		values = append(values, authorizerValues(inter)...)
		err = inter.InvokeTransaction(0, values...)
		return nil, err
	}
}
