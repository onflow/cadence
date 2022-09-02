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

type interpreterScriptExecutor struct {
	runtime *interpreterRuntime

	script  Script
	context Context

	preprocessOnce sync.Once
	preprocessErr  error

	storage            *Storage
	checkerOptions     []sema.Option
	interpreterOptions []interpreter.Option

	functions stdlib.StandardLibraryFunctions

	program                *interpreter.Program
	functionEntryPointType *sema.FunctionType

	executeOnce sync.Once
	executeErr  error
	result      cadence.Value
}

func newInterpreterScriptExecutor(
	runtime *interpreterRuntime,
	script Script,
	context Context,
) *interpreterScriptExecutor {

	return &interpreterScriptExecutor{
		runtime:        runtime,
		script:         script,
		context:        context,
		checkerOptions: context.CheckerOptions,
	}
}

func (executor *interpreterScriptExecutor) Preprocess() error {
	executor.preprocessOnce.Do(func() {
		executor.preprocessErr = executor.preprocess()
	})

	return executor.preprocessErr
}

func (executor *interpreterScriptExecutor) Execute() error {
	executor.executeOnce.Do(func() {
		executor.result, executor.executeErr = executor.execute()
	})

	return executor.executeErr
}

func (executor *interpreterScriptExecutor) Result() (cadence.Value, error) {
	// Note: Execute's error is saved into executor.executeErr and return in
	// the next line.
	_ = executor.Execute()
	return executor.result, executor.executeErr
}

func (executor *interpreterScriptExecutor) preprocess() (err error) {
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

	executor.functionEntryPointType, err = executor.program.Elaboration.FunctionEntryPointType()
	if err != nil {
		return newError(err, executor.context)
	}

	// Ensure the entry point's parameter types are importable
	if len(executor.functionEntryPointType.Parameters) > 0 {
		for _, param := range executor.functionEntryPointType.Parameters {
			if !param.TypeAnnotation.Type.IsImportable(map[*sema.Member]bool{}) {
				err = &ScriptParameterTypeNotImportableError{
					Type: param.TypeAnnotation.Type,
				}
				return newError(err, executor.context)
			}
		}
	}

	// Ensure the entry point's return type is valid
	if !executor.functionEntryPointType.ReturnTypeAnnotation.Type.IsExternallyReturnable(map[*sema.Member]bool{}) {
		err = &InvalidScriptReturnTypeError{
			Type: executor.functionEntryPointType.ReturnTypeAnnotation.Type,
		}
		return newError(err, executor.context)
	}

	return nil
}

func (executor *interpreterScriptExecutor) execute() (val cadence.Value, err error) {
	err = executor.Preprocess()
	if err != nil {
		return nil, err
	}

	defer executor.runtime.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		executor.context,
	)

	value, inter, err := executor.runtime.interpret(
		executor.program,
		executor.context,
		executor.storage,
		executor.functions,
		stdlib.BuiltinValues,
		executor.interpreterOptions,
		executor.checkerOptions,
		scriptExecutionFunction(
			executor.functionEntryPointType.Parameters,
			executor.script.Arguments,
			executor.context.Interface,
			interpreter.ReturnEmptyLocationRange,
		),
	)
	if err != nil {
		return nil, newError(err, executor.context)
	}

	// Export before committing storage

	result, err := exportValue(value, interpreter.ReturnEmptyLocationRange)
	if err != nil {
		return nil, newError(err, executor.context)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	err = executor.runtime.commitStorage(executor.storage, inter)
	if err != nil {
		return nil, newError(err, executor.context)
	}

	return result, nil
}

func scriptExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
	getLocationRange func() interpreter.LocationRange,
) interpretFunc {
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
		return inter.Invoke("main", values...)
	}
}
