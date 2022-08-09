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

type interpreterScriptExecutor struct {
	runtime interpreterRuntime

	script  Script
	context Context

	preprocessOnce sync.Once
	preprocessErr  error

	program                *interpreter.Program
	functionEntryPointType *sema.FunctionType

	executeOnce sync.Once
	executeErr  error
	result      cadence.Value
}

func newInterpreterScriptExecutor(
	runtime interpreterRuntime,
	script Script,
	context Context,
) *interpreterScriptExecutor {

	return &interpreterScriptExecutor{
		runtime: runtime,
		script:  script,
		context: context,
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
		executor.context.Environment = NewScriptInterpreterEnvironment(executor.runtime.defaultConfig)
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

	executor.functionEntryPointType, err = executor.program.Elaboration.FunctionEntryPointType()
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}

	// Ensure the entry point's parameter types are importable
	if len(executor.functionEntryPointType.Parameters) > 0 {
		for _, param := range executor.functionEntryPointType.Parameters {
			if !param.TypeAnnotation.Type.IsImportable(map[*sema.Member]bool{}) {
				err = &ScriptParameterTypeNotImportableError{
					Type: param.TypeAnnotation.Type,
				}
				return newError(err, location, codesAndPrograms)
			}
		}
	}

	// Ensure the entry point's return type is valid
	if !executor.functionEntryPointType.ReturnTypeAnnotation.Type.IsExternallyReturnable(map[*sema.Member]bool{}) {
		err = &InvalidScriptReturnTypeError{
			Type: executor.functionEntryPointType.ReturnTypeAnnotation.Type,
		}
		return newError(err, location, codesAndPrograms)
	}

	return nil
}

func (executor *interpreterScriptExecutor) execute() (val cadence.Value, err error) {
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

	interpret := scriptExecutionFunction(
		executor.functionEntryPointType.Parameters,
		executor.script.Arguments,
		executor.context.Interface,
	)

	value, inter, err := executor.context.Environment.Interpret(
		location,
		executor.program,
		interpret,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// Export before committing storage
	exportableValue := newExportableValue(value, inter)
	result, err := exportValue(
		exportableValue,
		interpreter.ReturnEmptyLocationRange,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	// Write back all stored values, which were actually just cached, back into storage.

	// Even though this function is `ExecuteScript`, that doesn't imply the changes
	// to storage will be actually persisted

	err = executor.context.Environment.CommitStorage(inter)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	return result, nil
}

func scriptExecutionFunction(
	parameters []*sema.Parameter,
	arguments [][]byte,
	runtimeInterface Interface,
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
		return inter.Invoke("main", values...)
	}
}
