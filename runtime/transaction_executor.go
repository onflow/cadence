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
	"github.com/onflow/cadence/runtime/stdlib"
)

type interpreterTransactionExecutor struct {
	runtime *interpreterRuntime

	script  Script
	context Context

	preprocessOnce sync.Once
	preprocessErr  error

	storage            *Storage
	interpreterOptions []interpreter.Option
	checkerOptions     []sema.Option

	functions stdlib.StandardLibraryFunctions

	program         *interpreter.Program
	transactionType *sema.TransactionType

	authorizers []Address

	executeOnce sync.Once
	executeErr  error
}

func newInterpreterTransactionExecutor(
	runtime *interpreterRuntime,
	script Script,
	context Context,
) Executor {

	return &interpreterTransactionExecutor{
		runtime:        runtime,
		script:         script,
		context:        context,
		checkerOptions: context.CheckerOptions,
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
	defer executor.runtime.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		executor.context,
	)

	executor.context.InitializeCodesAndPrograms()

	memoryGauge, _ := executor.context.Interface.(common.MemoryGauge)

	executor.storage = NewStorage(executor.context.Interface, memoryGauge)

	executor.functions = executor.runtime.standardLibraryFunctions(
		executor.context,
		executor.storage,
		executor.interpreterOptions,
		executor.checkerOptions,
	)

	executor.program, err = executor.runtime.parseAndCheckProgram(
		executor.script.Source,
		executor.context,
		executor.functions,
		stdlib.BuiltinValues,
		executor.checkerOptions,
		true,
		importResolutionResults{},
	)
	if err != nil {
		return newError(err, executor.context)
	}

	transactions := executor.program.Elaboration.TransactionTypes
	transactionCount := len(transactions)
	if transactionCount != 1 {
		err = InvalidTransactionCountError{
			Count: transactionCount,
		}
		return newError(err, executor.context)
	}

	executor.transactionType = transactions[0]

	wrapPanic(func() {
		executor.authorizers, err = executor.context.Interface.GetSigningAccounts()
	})
	if err != nil {
		return newError(err, executor.context)
	}
	// check parameter count

	argumentCount := len(executor.script.Arguments)
	authorizerCount := len(executor.authorizers)

	transactionParameterCount := len(executor.transactionType.Parameters)
	if argumentCount != transactionParameterCount {
		err = InvalidEntryPointParameterCountError{
			Expected: transactionParameterCount,
			Actual:   argumentCount,
		}
		return newError(err, executor.context)
	}

	transactionAuthorizerCount := len(executor.transactionType.PrepareParameters)
	if authorizerCount != transactionAuthorizerCount {
		err = InvalidTransactionAuthorizerCountError{
			Expected: transactionAuthorizerCount,
			Actual:   authorizerCount,
		}
		return newError(err, executor.context)
	}

	return nil
}

func (executor *interpreterTransactionExecutor) authorizerValues(inter *interpreter.Interpreter) []interpreter.Value {

	// gather authorizers

	authorizerValues := make([]interpreter.Value, len(executor.authorizers))

	for i, address := range executor.authorizers {
		authorizerValues[i] = executor.runtime.newAuthAccountValue(
			inter,
			interpreter.NewAddressValue(
				inter,
				address,
			),
			executor.context,
			executor.storage,
			executor.interpreterOptions,
			executor.checkerOptions,
		)
	}

	return authorizerValues
}

func (executor *interpreterTransactionExecutor) execute() (err error) {
	err = executor.Preprocess()
	if err != nil {
		return err
	}

	defer executor.runtime.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		executor.context,
	)

	_, inter, err := executor.runtime.interpret(
		executor.program,
		executor.context,
		executor.storage,
		executor.functions,
		stdlib.BuiltinValues,
		executor.interpreterOptions,
		executor.checkerOptions,
		executor.runtime.transactionExecutionFunction(
			executor.transactionType.Parameters,
			executor.script.Arguments,
			executor.context.Interface,
			executor.authorizerValues,
		),
	)
	if err != nil {
		return newError(err, executor.context)
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = executor.runtime.commitStorage(executor.storage, inter)
	if err != nil {
		return newError(err, executor.context)
	}

	return nil
}
