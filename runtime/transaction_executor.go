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
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
)

type transactionExecutorPreparation struct {
	codesAndPrograms CodesAndPrograms
	environment      Environment
	preprocessErr    error
	transactionType  *sema.TransactionType
	storage          *Storage
	program          *interpreter.Program
	preprocessOnce   sync.Once
}

type transactionExecutorExecution struct {
	executeErr  error
	interpret   interpretFunc
	executeOnce sync.Once
}

type transactionExecutor struct {
	context Context
	transactionExecutorExecution
	runtime Runtime
	script  Script
	transactionExecutorPreparation
	vm               *vm.VM
	authorizerValues func(context interpreter.AccountCreationContext) []interpreter.Value
}

func newTransactionExecutor(
	runtime Runtime,
	script Script,
	context Context,
) Executor {

	return &transactionExecutor{
		runtime: runtime,
		script:  script,
		context: context,
	}
}

// Transaction's preprocessing which could be done in parallel with other
// transactions / scripts.
func (executor *transactionExecutor) Preprocess() error {
	executor.preprocessOnce.Do(func() {
		executor.preprocessErr = executor.preprocess()
	})

	return executor.preprocessErr
}

func (executor *transactionExecutor) Execute() error {
	executor.executeOnce.Do(func() {
		executor.executeErr = executor.execute()
	})

	return executor.executeErr
}

func (executor *transactionExecutor) Result() (cadence.Value, error) {
	return nil, executor.Execute()
}

func (executor *transactionExecutor) preprocess() (err error) {
	context := executor.context
	location := context.Location
	script := executor.script

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

	authorizerAddresses, err := runtimeInterface.GetSigningAccounts()
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	// check parameter count

	argumentCount := len(script.Arguments)
	authorizerCount := len(authorizerAddresses)

	transactionParameterCount := len(transactionType.Parameters)
	if argumentCount != transactionParameterCount {
		err = InvalidEntryPointParameterCountError{
			Expected: transactionParameterCount,
			Actual:   argumentCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	prepareParameters := transactionType.PrepareParameters

	transactionAuthorizerCount := len(prepareParameters)
	if authorizerCount != transactionAuthorizerCount {
		err = InvalidTransactionAuthorizerCountError{
			Expected: transactionAuthorizerCount,
			Actual:   authorizerCount,
		}
		return newError(err, location, codesAndPrograms)
	}

	executor.authorizerValues = func(context interpreter.AccountCreationContext) []interpreter.Value {
		return authorizerValues(
			executor.environment,
			context,
			authorizerAddresses,
			prepareParameters,
		)
	}

	switch environment := environment.(type) {
	case *InterpreterEnvironment:
		if context.UseVM {
			panic(errors.NewUnexpectedError(
				"expected to run with the VM, but found an incompatible environment: %T",
				environment,
			))
		}

		executor.interpret = executor.transactionExecutionFunction()

	case *vmEnvironment:
		if !context.UseVM {
			panic(errors.NewUnexpectedError(
				"expected to run with the interpreter, but found an incompatible environment: %T",
				environment,
			))
		}

		var program *Program
		program, err = environment.loadProgram(location)
		if err != nil {
			return newError(err, location, codesAndPrograms)
		}
		executor.vm = environment.newVM(location, program.compiledProgram.program)

	default:
		return errors.NewUnexpectedError("transactions can only be executed with the interpreter")
	}

	return nil
}

func (executor *transactionExecutor) execute() (err error) {
	err = executor.Preprocess()
	if err != nil {
		return err
	}

	environment := executor.environment
	context := executor.context
	location := context.Location
	codesAndPrograms := executor.codesAndPrograms

	defer Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	switch environment := environment.(type) {
	case *InterpreterEnvironment:
		err = executor.executeWithInterpreter(environment)

	case *vmEnvironment:
		err = executor.executeWithVM()

	default:
		panic(errors.NewUnexpectedError("unsupported environment: %T", environment))
	}
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}
	return nil
}

func (executor *transactionExecutor) executeWithInterpreter(
	environment *InterpreterEnvironment,
) error {
	_, inter, err := environment.Interpret(
		executor.context.Location,
		executor.program,
		executor.interpret,
	)
	if err != nil {
		return err
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.commitStorage(inter)
	if err != nil {
		return err
	}

	return nil
}

func (executor *transactionExecutor) executeWithVM() (err error) {

	context := executor.vm.Context()
	codesAndPrograms := executor.codesAndPrograms

	// Recover internal panics and return them as an error.
	// For example, the argument validation might attempt to
	// load contract code for non-existing types

	defer Recover(
		func(internalErr Error) {
			err = internalErr
		},
		executor.context.Location,
		codesAndPrograms,
	)

	environment := executor.environment

	arguments, err := importValidatedArguments(
		context,
		environment,
		executor.script.Arguments,
		executor.transactionType.Parameters,
	)
	if err != nil {
		return err
	}

	signers := executor.authorizerValues(context)

	err = executor.vm.InvokeTransaction(arguments, signers...)
	if err != nil {
		return err
	}

	// Write back all stored values, which were actually just cached, back into storage
	err = environment.commitStorage(context)
	if err != nil {
		return err
	}

	return nil
}

func (executor *transactionExecutor) transactionExecutionFunction() interpretFunc {
	return func(inter *interpreter.Interpreter) (value interpreter.Value, err error) {

		// Recover internal panics and return them as an error.
		// For example, the argument validation might attempt to
		// load contract code for non-existing types

		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		arguments, err := importValidatedArguments(
			inter,
			executor.environment,
			executor.script.Arguments,
			executor.transactionType.Parameters,
		)
		if err != nil {
			return nil, err
		}

		signers := executor.authorizerValues(inter)

		err = inter.InvokeTransaction(arguments, signers...)

		return nil, err
	}
}
