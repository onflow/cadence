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

package interpreter_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
)

func TestInterpretFunctionInvocationCheckArgumentTypes(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
       fun test(_ x: Int): Int {
           return x
       }
   `)

	_, err := inter.Invoke("test", interpreter.TrueValue)
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.ValueTransferTypeError{})
}

func TestInterpretSelfDeclaration(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string, expectSelf bool) {

		checkFunction := stdlib.NewStandardLibraryStaticFunction(
			"check",
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			``,
			func(invocation interpreter.Invocation) interpreter.Value {
				// Check that the *caller's* self

				callStack := invocation.InvocationContext.CallStack()
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

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, checkFunction)

		inter, err := parseCheckAndInterpretWithOptions(t, code, ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				Storage: newUnmeteredInMemoryStorage(),
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
				AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
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

func TestInterpretRejectUnboxedInvocation(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(n: Int?): Int? {
		  return n.map(fun(n: Int): Int {
			  return n + 1
		  })
      }
    `)

	value := interpreter.NewUnmeteredUIntValueFromUint64(42)

	test := inter.Globals.Get("test").GetValue(inter).(interpreter.FunctionValue)

	invocation := interpreter.NewInvocation(
		inter,
		nil,
		nil,
		[]interpreter.Value{value},
		[]sema.Type{sema.IntType},
		nil,
		interpreter.EmptyLocationRange,
	)

	_, err := interpreter.InvokeFunction(
		inter,
		test,
		invocation,
	)
	RequireError(t, err)

	require.ErrorAs(t, err, &interpreter.MemberAccessTypeError{})
}
