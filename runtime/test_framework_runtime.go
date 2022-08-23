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
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

var _ Runtime = &TestFrameworkRuntime{}

// TestFrameworkRuntime is the Runtime implementation used by the test framework.
// It's a wrapper around interpreterRuntime, exposing additional functionalities
// needed for running tests.
//
type TestFrameworkRuntime struct {
	interpreterRuntime
}

func NewTestFrameworkRuntime() *TestFrameworkRuntime {
	return &TestFrameworkRuntime{}
}

// ParseAndCheck is a modified version of 'ParseAndCheckProgram' of 'interpreterRuntime'.
// This method allows passing configs.
//
func (r *TestFrameworkRuntime) ParseAndCheck(
	code []byte,
	context Context,
	checkerConfig *sema.Config,
) (
	program *interpreter.Program,
	err error,
) {

	// TODO: use `checkerConfig`

	location := context.Location

	codesAndPrograms := newCodesAndPrograms()

	defer r.Recover(
		func(internalErr Error) {
			err = internalErr
		},
		location,
		codesAndPrograms,
	)

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(r.defaultConfig)
	}
	environment.Configure(
		context.Interface,
		codesAndPrograms,
		nil,
		context.CoverageReport,
	)

	program, err = environment.ParseAndCheckProgram(
		code,
		location,
		true,
	)
	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	return program, nil
}

// Interpret interprets the given program.
//
func (r TestFrameworkRuntime) Interpret(
	context Context,
	interpreterConfig *interpreter.Config,
) (*interpreter.Interpreter, error) {

	// TODO: use `interpreterConfig`

	codesAndPrograms := newCodesAndPrograms()

	environment := context.Environment
	if environment == nil {
		environment = NewBaseInterpreterEnvironment(r.defaultConfig)
	}
	environment.Configure(
		context.Interface,
		codesAndPrograms,
		nil,
		context.CoverageReport,
	)

	location := context.Location

	_, inter, err := environment.Interpret(
		location,
		nil,
		nil,
	)

	if err != nil {
		return nil, newError(err, location, codesAndPrograms)
	}

	return inter, nil
}
