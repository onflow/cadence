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
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

// TestFrameworkRuntime is the Runtime implementation used by the test framework.
// It's a wrapper around interpreterRuntime, exposing additional functionalities
// needed for running tests.
//
type TestFrameworkRuntime struct {
	interpreterRuntime
}

func NewTestFrameworkRuntime(options ...Option) *TestFrameworkRuntime {
	runtime := &TestFrameworkRuntime{}
	for _, option := range options {
		option(runtime)
	}
	return runtime
}

// ParseAndCheck is a modified version of 'ParseAndCheckProgram' of 'interpreterRuntime'.
// This method allows passing checker-options and interpreter-options.
//
func (r *TestFrameworkRuntime) ParseAndCheck(
	code []byte,
	context Context,
	checkerOptions []sema.Option,
	interpreterOptions []interpreter.Option,
) (
	program *interpreter.Program,
	err error,
) {
	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		context,
	)

	context.InitializeCodesAndPrograms()

	memoryGauge, _ := context.Interface.(common.MemoryGauge)

	storage := NewStorage(context.Interface, memoryGauge)

	functions := r.standardLibraryFunctions(
		context,
		storage,
		interpreterOptions,
		checkerOptions,
	)

	program, err = r.parseAndCheckProgram(
		code,
		context,
		functions,
		stdlib.BuiltinValues,
		checkerOptions,
		true,
		importResolutionResults{},
	)
	if err != nil {
		return nil, newError(err, context)
	}

	return program, nil
}

// Interpret interprets the given program.
//
func (r TestFrameworkRuntime) Interpret(
	program *interpreter.Program,
	storage *Storage,
	context Context,
	checkerOptions []sema.Option,
	interpreterOptions []interpreter.Option,
) (*interpreter.Interpreter, error) {
	context.InitializeCodesAndPrograms()

	functions := r.standardLibraryFunctions(
		context,
		storage,
		interpreterOptions,
		checkerOptions,
	)

	_, inter, err := r.interpret(
		program,
		context,
		storage,
		functions,
		stdlib.BuiltinValues,
		interpreterOptions,
		checkerOptions,
		nil,
	)
	if err != nil {
		return nil, newError(err, context)
	}

	return inter, nil
}
