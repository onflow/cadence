/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package stdlib

import (
	"testing"

	"github.com/onflow/cadence/runtime/activations"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/parser"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func newUnmeteredInMemoryStorage() interpreter.InMemoryStorage {
	return interpreter.NewInMemoryStorage(nil)
}

func newInterpreter(t *testing.T, code string, valueDeclarations ...StandardLibraryValue) *interpreter.Interpreter {
	program, err := parser.ParseProgram(
		nil,
		[]byte(code),
		parser.Config{},
	)
	require.NoError(t, err)

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	for _, valueDeclaration := range valueDeclarations {
		baseValueActivation.DeclareValue(valueDeclaration)
	}

	checker, err := sema.NewChecker(
		program,
		utils.TestLocation,
		nil,
		&sema.Config{
			BaseValueActivation: baseValueActivation,
			AccessCheckMode:     sema.AccessCheckModeStrict,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
	for _, valueDeclaration := range valueDeclarations {
		interpreter.Declare(baseActivation, valueDeclaration)
	}

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage:        storage,
			BaseActivation: baseActivation,
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	return inter
}

func TestAssert(t *testing.T) {

	t.Parallel()

	inter := newInterpreter(t,
		`access(all) let test = assert`,
		AssertFunction,
	)

	_, err := inter.Invoke(
		"test",
		interpreter.FalseValue,
		interpreter.NewUnmeteredStringValue("oops"),
	)
	assert.Equal(t,
		interpreter.Error{
			Err: AssertionError{
				Message: "oops",
			},
			Location: utils.TestLocation,
		},
		err,
	)

	_, err = inter.Invoke("test", interpreter.FalseValue)
	assert.Equal(t,
		interpreter.Error{
			Err: AssertionError{
				Message: "",
			},
			Location: utils.TestLocation,
		},
		err)

	_, err = inter.Invoke(
		"test",
		interpreter.TrueValue,
		interpreter.NewUnmeteredStringValue("oops"),
	)
	assert.NoError(t, err)

	_, err = inter.Invoke("test", interpreter.TrueValue)
	assert.NoError(t, err)
}

func TestPanic(t *testing.T) {

	t.Parallel()

	inter := newInterpreter(t,
		`access(all) let test = panic`,
		PanicFunction,
	)

	_, err := inter.Invoke("test", interpreter.NewUnmeteredStringValue("oops"))
	assert.Equal(t,
		interpreter.Error{
			Err: PanicError{
				Message: "oops",
			},
			Location: utils.TestLocation,
		},
		err,
	)
}
