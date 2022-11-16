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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/activations"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretFunctionInvocationCheckArgumentTypes(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(_ x: Int): Int {
           return x
       }
   `)

	_, err := inter.Invoke("test", interpreter.BoolValue(true))
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.ValueTransferTypeError{})
}

func TestInterpretSelfDeclaration(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string, expectSelf bool) {

		checkFunction := stdlib.NewStandardLibraryFunction(
			"check",
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			``,
			func(invocation interpreter.Invocation) interpreter.Value {
				// Check that the *caller's* self

				callStack := invocation.Interpreter.CallStack()
				parentInvocation := callStack[len(callStack)-1]

				if expectSelf {
					require.NotNil(t, parentInvocation.Self)
				} else {
					require.Nil(t, parentInvocation.Self)
				}
				return interpreter.Void
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(checkFunction)

		baseActivation := activations.NewActivation[*interpreter.Variable](nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, checkFunction)

		inter, err := parseCheckAndInterpretWithOptions(t, code, ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				Storage:        newUnmeteredInMemoryStorage(),
				BaseActivation: baseActivation,
			},
			CheckerConfig: &sema.Config{
				BaseValueActivation: baseValueActivation,
				AccessCheckMode:     sema.AccessCheckModeNotSpecifiedUnrestricted,
			},
		})
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		require.NoError(t, err)
	}

	t.Run("plain function", func(t *testing.T) {

		t.Parallel()

		code := `
            fun foo() {
                check()
            }

            fun test() {
                foo()
            }
        `
		test(t, code, false)
	})

	t.Run("composite function", func(t *testing.T) {

		t.Parallel()

		code := `
            struct S {
                fun test() {
                     check()
                }
            }


            fun test() {
                S().test()
            }
        `
		test(t, code, true)
	})

}
