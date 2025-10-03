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

type scriptExecutorPreparation struct {
	environment            Environment
	preprocessErr          error
	codesAndPrograms       CodesAndPrograms
	functionEntryPointType *sema.FunctionType
	program                *interpreter.Program
	storage                *Storage
	interpret              interpretFunc
	preprocessOnce         sync.Once
}

type scriptExecutorExecution struct {
	executeErr  error
	result      cadence.Value
	executeOnce sync.Once
}

type scriptExecutor struct {
	context Context
	scriptExecutorExecution
	runtime Runtime
	scriptExecutorPreparation
	script Script
	vm     *vm.VM
}

func newScriptExecutor(
	runtime Runtime,
	script Script,
	context Context,
) *scriptExecutor {

	return &scriptExecutor{
		runtime: runtime,
		script:  script,
		context: context,
	}
}

func (executor *scriptExecutor) Preprocess() error {
	executor.preprocessOnce.Do(func() {
		executor.preprocessErr = executor.preprocess()
	})

	return executor.preprocessErr
}

func (executor *scriptExecutor) Execute() error {
	executor.executeOnce.Do(func() {
		executor.result, executor.executeErr = executor.execute()
	})

	return executor.executeErr
}

func (executor *scriptExecutor) Result() (cadence.Value, error) {
	// Note: Execute's error is saved into executor.executeErr and return in
	// the next line.
	_ = executor.Execute()
	return executor.result, executor.executeErr
}

func (executor *scriptExecutor) preprocess() (err error) {
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
		StorageConfig{},
	)
	executor.storage = storage

	environment := context.Environment
	if environment == nil {
		if context.UseVM {
			environment = NewScriptVMEnvironment(config)
		} else {
			environment = NewScriptInterpreterEnvironment(config)
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
	if !returnType.IsExportable(map[*sema.Member]bool{}) {
		err = &InvalidScriptReturnTypeError{
			Type: returnType,
		}
		return newError(err, location, codesAndPrograms)
	}

	switch environment := environment.(type) {
	case *InterpreterEnvironment:
		if context.UseVM {
			panic(errors.NewUnexpectedError(
				"expected to run with the VM, but found an incompatible environment: %T",
				environment,
			))
		}

		executor.interpret = executor.scriptExecutionFunction()

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
		return errors.NewUnexpectedError("scripts can only be executed with the interpreter")
	}

	return nil
}

func (executor *scriptExecutor) execute() (val cadence.Value, err error) {
	err = executor.Preprocess()
	if err != nil {
		return nil, err
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

	var result cadence.Value

	switch environment := environment.(type) {
	case *InterpreterEnvironment:
		result, err = executor.executeWithInterpreter(environment)

	case *vmEnvironment:
		result, err = executor.executeWithVM(environment)

	default:
		panic(errors.NewUnexpectedError("unsupported environment: %T", environment))
	}

	if err != nil {
		return nil, newError(err, executor.context.Location, codesAndPrograms)
	}
	return result, nil
}

func (executor *scriptExecutor) executeWithInterpreter(
	environment *InterpreterEnvironment,
) (val cadence.Value, err error) {

	value, inter, err := environment.Interpret(
		executor.context.Location,
		executor.program,
		executor.interpret,
	)
	if err != nil {
		return nil, err
	}

	return ExportValue(
		value,
		inter,
		interpreter.EmptyLocationRange,
	)
}

func (executor *scriptExecutor) executeWithVM(
	environment *vmEnvironment,
) (val cadence.Value, err error) {

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

	values, err := importValidatedArguments(
		context,
		executor.environment,
		interpreter.EmptyLocationRange,
		executor.script.Arguments,
		executor.functionEntryPointType.Parameters,
	)
	if err != nil {
		return nil, err
	}

	value, err := executor.vm.InvokeExternally(
		sema.FunctionEntryPointName,
		values...,
	)
	if err != nil {
		return nil, err
	}

	return ExportValue(value, context, interpreter.EmptyLocationRange)
}

func (executor *scriptExecutor) scriptExecutionFunction() interpretFunc {
	return func(inter *interpreter.Interpreter) (value interpreter.Value, err error) {

		// Recover internal panics and return them as an error.
		// For example, the argument validation might attempt to
		// load contract code for non-existing types

		defer inter.RecoverErrors(func(internalErr error) {
			err = internalErr
		})

		values, err := importValidatedArguments(
			inter,
			executor.environment,
			interpreter.EmptyLocationRange,
			executor.script.Arguments,
			executor.functionEntryPointType.Parameters,
		)
		if err != nil {
			return nil, err
		}

		return inter.Invoke(sema.FunctionEntryPointName, values...)
	}
}
