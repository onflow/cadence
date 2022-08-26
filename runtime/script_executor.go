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

	// preprocess
	preprocessOnce         sync.Once
	preprocessErr          error
	codesAndPrograms       codesAndPrograms
	storage                *Storage
	program                *interpreter.Program
	environment            Environment
	functionEntryPointType *sema.FunctionType
	interpret              InterpretFunc

	// execute
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
		environment = NewScriptInterpreterEnvironment(interpreterRuntime.DefaultConfig)
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

	functionEntryPointType, err := program.Elaboration.FunctionEntryPointType()
	if err != nil {
		return newError(err, location, codesAndPrograms)
	}
	executor.functionEntryPointType = functionEntryPointType

	// Ensure the entry point's parameter types are importable
	parameters := functionEntryPointType.Parameters
	if len(parameters) > 0 {
		for _, param := range parameters {
			if !param.TypeAnnotation.Type.IsImportable(map[*sema.Member]bool{}) {
				err = &ScriptParameterTypeNotImportableError{
					Type: param.TypeAnnotation.Type,
				}
				return newError(err, location, codesAndPrograms)
			}
		}
	}

	// Ensure the entry point's return type is valid
	returnType := functionEntryPointType.ReturnTypeAnnotation.Type
	if !returnType.IsExternallyReturnable(map[*sema.Member]bool{}) {
		err = &InvalidScriptReturnTypeError{
			Type: returnType,
		}
		return newError(err, location, codesAndPrograms)
	}

	executor.interpret = scriptExecutionFunction(
		parameters,
		script.Arguments,
		runtimeInterface,
	)

	return nil
}

func (executor *interpreterScriptExecutor) execute() (val cadence.Value, err error) {
	err = executor.Preprocess()
	if err != nil {
		return nil, err
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

	value, inter, err := environment.Interpret(
		location,
		executor.program,
		executor.interpret,
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

	err = environment.CommitStorage(inter)
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
