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

package stdlib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
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
		TestLocation,
		nil,
		&sema.Config{
			BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
				return baseValueActivation
			},
			AccessCheckMode: sema.AccessCheckModeStrict,
		},
	)
	require.NoError(t, err)

	err = checker.Check()
	require.NoError(t, err)

	storage := newUnmeteredInMemoryStorage()

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	for _, valueDeclaration := range valueDeclarations {
		interpreter.Declare(baseActivation, valueDeclaration)
	}

	inter, err := interpreter.NewInterpreter(
		interpreter.ProgramFromChecker(checker),
		checker.Location,
		&interpreter.Config{
			Storage: storage,
			BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
				return baseActivation
			},
		},
	)
	require.NoError(t, err)

	err = inter.Interpret()
	require.NoError(t, err)

	return inter
}

func TestCheckAssert(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(InterpreterAssertFunction)

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
	}

	t.Run("too few arguments", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = assert()`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.InsufficientArgumentsError{})
	})

	t.Run("invalid first argument", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = assert(1)`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.TypeMismatchError{})
	})

	t.Run("no message", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = assert(true)`)

		require.NoError(t, err)
	})

	t.Run("with message", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = assert(true, message: "foo")`)

		require.NoError(t, err)
	})

	t.Run("invalid message", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = assert(true, message: 1)`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.TypeMismatchError{})
	})

	t.Run("missing argument label for message", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = assert(true, "")`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.MissingArgumentLabelError{})
	})

	t.Run("too many arguments", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = assert(true, message: "foo", true)`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.ExcessiveArgumentsError{})
	})
}

func TestInterpretAssert(t *testing.T) {

	inter := newInterpreter(t,
		`access(all) let test = assert`,
		InterpreterAssertFunction,
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
			Location: TestLocation,
		},
		err,
	)

	_, err = inter.Invoke("test", interpreter.FalseValue)
	assert.Equal(t,
		interpreter.Error{
			Err: AssertionError{
				Message: "",
			},
			Location: TestLocation,
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

func TestCheckPanic(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(InterpreterPanicFunction)

	parseAndCheck := func(t *testing.T, code string) (*sema.Checker, error) {
		return ParseAndCheckWithOptions(t,
			code,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)
	}

	t.Run("too few arguments", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = panic()`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.InsufficientArgumentsError{})
	})

	t.Run("message", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = panic("test")`)

		require.NoError(t, err)

	})

	t.Run("invalid message", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = panic(true)`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.TypeMismatchError{})
	})

	t.Run("too many arguments", func(t *testing.T) {

		_, err := parseAndCheck(t, `let _ = panic("test", 1)`)

		errs := RequireCheckerErrors(t, err, 1)
		require.IsType(t, errs[0], &sema.ExcessiveArgumentsError{})
	})
}

func TestInterpretPanic(t *testing.T) {

	t.Parallel()

	inter := newInterpreter(t,
		`access(all) let test = panic`,
		InterpreterPanicFunction,
	)

	_, err := inter.Invoke("test", interpreter.NewUnmeteredStringValue("oops"))
	assert.Equal(t,
		interpreter.Error{
			Err: &PanicError{
				Message: "oops",
			},
			Location: TestLocation,
		},
		err,
	)
}
