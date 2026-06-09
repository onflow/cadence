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

package stdlib_test

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

var compile = flag.Bool("compile", false, "Run tests using the compiler")

// newInvokable parses, checks, and prepares the given code with the given
// standard library values available as built-ins.
func newInvokable(t *testing.T, code string, valueDeclarations ...stdlib.StandardLibraryValue) Invokable {
	semaBaseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	interpreterBaseActivation := activations.NewActivation(nil, interpreter.BaseActivation)

	for _, valueDeclaration := range valueDeclarations {
		semaBaseValueActivation.DeclareValue(valueDeclaration)
		interpreter.Declare(interpreterBaseActivation, valueDeclaration)
	}

	invokable, err := test_utils.ParseCheckAndPrepareWithOptions(
		t,
		code,
		test_utils.ParseCheckAndInterpretOptions{
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return semaBaseValueActivation
					},
					AccessCheckMode: sema.AccessCheckModeStrict,
				},
			},
			InterpreterConfig: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return interpreterBaseActivation
				},
			},
		},
		*compile,
		true,
	)
	require.NoError(t, err)

	return invokable
}

func TestCheckAssert(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterAssertFunction)

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

	inter := newInvokable(t,
		`access(all) let test = assert`,
		stdlib.InterpreterAssertFunction,
	)

	// Failing condition, with message
	_, err := inter.Invoke(
		"test",
		interpreter.FalseValue,
		interpreter.NewUnmeteredStringValue("oops"),
	)
	var assertionErr *stdlib.AssertionError
	require.ErrorAs(t, err, &assertionErr)
	assert.Equal(t, "oops", assertionErr.Message)

	// Failing condition, without message
	_, err = inter.Invoke("test", interpreter.FalseValue)
	require.ErrorAs(t, err, &assertionErr)
	assert.Equal(t, "", assertionErr.Message)

	// Passing condition, with message
	_, err = inter.Invoke(
		"test",
		interpreter.TrueValue,
		interpreter.NewUnmeteredStringValue("oops"),
	)
	assert.NoError(t, err)

	// Passing condition, with message
	_, err = inter.Invoke("test", interpreter.TrueValue)
	assert.NoError(t, err)
}

func TestCheckPanic(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.InterpreterPanicFunction)

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

	inter := newInvokable(t,
		`access(all) fun test(_ message: String): String {
            return panic(message)
        }`,
		stdlib.InterpreterPanicFunction,
	)

	_, err := inter.Invoke("test", interpreter.NewUnmeteredStringValue("oops"))
	var panicErr *stdlib.PanicError
	require.ErrorAs(t, err, &panicErr)
	assert.Equal(t, "oops", panicErr.Message)
}
