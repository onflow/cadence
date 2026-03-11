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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretReturnType(t *testing.T) {

	t.Parallel()

	xValue := stdlib.StandardLibraryValue{
		Name: "x",
		Type: sema.IntType,
		// NOTE: value with different type than declared type
		Value: interpreter.TrueValue,
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(xValue)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, xValue)

	inter, err := parseCheckAndPrepareWithOptions(
		t,
		`
            fun test(): Int {
                return x
            }
        `,
		ParseCheckAndInterpretOptions{
			InterpreterConfig: &interpreter.Config{
				Storage: NewUnmeteredInMemoryStorage(),
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
					AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
				},
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	RequireError(t, err)

	var transferTypeError *interpreter.ValueTransferTypeError
	require.ErrorAs(t, err, &transferTypeError)
}

func TestInterpretSelfDeclaration(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, code string, expectSelf bool) {

		checkFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"check",
			&sema.FunctionType{
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			``,
			func(
				context interpreter.NativeFunctionContext,
				_ interpreter.TypeArgumentsIterator,
				_ interpreter.ArgumentTypesIterator,
				_ interpreter.Value,
				args []interpreter.Value,
			) interpreter.Value {
				// Check that the *caller's* self

				// This is an interpreter-only test.
				// So the `InvocationContext` is an interpreter instance.
				inter := context.(*interpreter.Interpreter)

				callStack := inter.CallStack()
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

		// NOTE: test only applies to the interpreter,
		// the VM does not provide a way to check the caller's self
		inter, err := parseCheckAndInterpretWithOptions(
			t,
			code,
			ParseCheckAndInterpretOptions{
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
						AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
					},
				},
				InterpreterConfig: &interpreter.Config{
					Storage: NewUnmeteredInMemoryStorage(),
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
			},
		)
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

	inter := parseCheckAndPrepare(t, `
      fun test(n: Int?): Int? {
		  return n.map(fun(n: Int): Int {
			  return n + 1
		  })
      }
    `)

	value := interpreter.NewUnmeteredUIntValueFromUint64(42)

	test := inter.GetGlobal("test").(interpreter.FunctionValue)

	invocation := interpreter.NewInvocation(
		inter,
		nil,
		nil,
		[]interpreter.Value{value},
		[]sema.Type{sema.IntType},
		nil,
		sema.IntType,
		interpreter.LocationRange{},
	)

	_, err := interpreter.InvokeFunction(
		inter,
		test,
		invocation,
	)
	RequireError(t, err)

	if *compile {
		var internalErr errors.InternalError
		require.ErrorAs(t, err, &internalErr)
	} else {
		var memberAccessTypeError *interpreter.MemberAccessTypeError
		require.ErrorAs(t, err, &memberAccessTypeError)
	}
}

func TestInterpretInvocationReturnTypeValidation(t *testing.T) {

	t.Parallel()

	t.Run("native function", func(t *testing.T) {

		fooFunction := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"foo",
			sema.NewSimpleFunctionType(
				sema.FunctionPurityImpure,
				nil,
				sema.TypeAnnotation{
					Type: sema.IntType,
				},
			),
			"",
			func(
				_ interpreter.NativeFunctionContext,
				_ interpreter.TypeArgumentsIterator,
				_ interpreter.ArgumentTypesIterator,
				_ interpreter.Value,
				_ []interpreter.Value,
			) interpreter.Value {
				return interpreter.NewUnmeteredStringValue("hello")
			},
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(fooFunction)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, fooFunction)

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			`
            fun test() {
                foo()
            }
        `,
			ParseCheckAndInterpretOptions{
				InterpreterConfig: &interpreter.Config{
					Storage: NewUnmeteredInMemoryStorage(),
					BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
						return baseActivation
					},
				},
				ParseAndCheckOptions: &ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
							return baseValueActivation
						},
						AccessCheckMode: sema.AccessCheckModeNotSpecifiedUnrestricted,
					},
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		var transferTypeError *interpreter.ValueTransferTypeError
		require.ErrorAs(t, err, &transferTypeError)
	})

	t.Run("user function", func(t *testing.T) {
		t.Parallel()

		inter, err := parseCheckAndPrepareWithOptions(
			t,
			`
            struct interface I {
                fun foo(): String
            }

            struct S: I {
                fun foo(): Int {
                    return 2
                }
            }

            fun test() {
                let s: {I} = S()
                s.foo()
            }
        `,
			ParseCheckAndInterpretOptions{
				HandleCheckerError: func(err error) {
					errs := RequireCheckerErrors(t, err, 1)
					require.IsType(t, &sema.ConformanceError{}, errs[0])
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test")
		RequireError(t, err)

		var transferTypeError *interpreter.ValueTransferTypeError
		require.ErrorAs(t, err, &transferTypeError)
	})
}

func TestInterpretInvocationOnTypeConfusedValue(t *testing.T) {
	t.Parallel()

	inter := parseCheckAndPrepare(t, `
        struct X {
            fun foo(): Int { return 3 }
        }

        struct Y {
            fun foo(): Int { return 4 }
        }

        fun test(x: X): Int {
            return x.foo()
        }
    `)

	yValue := interpreter.NewCompositeValue(
		inter,
		TestLocation,
		"Y",
		common.CompositeKindStructure,
		nil,
		common.ZeroAddress,
	)

	// Intentionally passing wrong type of value
	_, err := inter.InvokeUncheckedForTestingOnly("test", yValue) //nolint:staticcheck
	RequireError(t, err)

	var memberAccessTypeError *interpreter.MemberAccessTypeError
	require.ErrorAs(t, err, &memberAccessTypeError)
}

func TestInterpretInvocationReferenceAuthorizationReturnValueTypeCheck(t *testing.T) {
	t.Parallel()

	address := interpreter.NewUnmeteredAddressValueFromBytes([]byte{42})

	inter, _, _ := testAccount(
		t,
		address,
		true,
		nil,
		`
            struct interface SI {
                fun foo(): &[Int] {
                    pre { true }
                }
            }

            struct S: SI {

                fun foo(): auth(Mutate) &[Int] {
                    return &[]
                }
            }

            fun testS() {
                let s = S()
                let ref = s.foo()
            }

            fun testSI() {
                let si: {SI} = S()
                let ref = si.foo()
            }

            fun testSIDowncast() {
                let si: {SI} = S()
                si.foo() as! auth(Mutate) &[Int]
            }
        `,
		sema.Config{},
	)

	_, err := inter.Invoke("testS")
	require.NoError(t, err)

	_, err = inter.Invoke("testSI")
	require.NoError(t, err)

	_, err = inter.Invoke("testSIDowncast")
	RequireError(t, err)

	var forceCastTypeMismatchErr *interpreter.ForceCastTypeMismatchError
	require.ErrorAs(t, err, &forceCastTypeMismatchErr)

	assert.Equal(t,
		common.TypeID("auth(Mutate)&[Int]"),
		forceCastTypeMismatchErr.ExpectedType.ID(),
	)
	assert.Equal(t,
		common.TypeID("&[Int]"),
		forceCastTypeMismatchErr.ActualType.ID(),
	)
}

func TestInterpretFunctionParameterContravariance(t *testing.T) {

	t.Parallel()

	t.Run("optional parameter via function variable", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): UInt8? {
                let f: fun(Int): UInt8? = funcWithOptionalParam
                return f(4)
            }

            fun funcWithOptionalParam(_ a: Int?): UInt8? {
                return a.map(fun (_ n: Int): UInt8 {
                    return UInt8(n)
                })
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		require.IsType(t, &interpreter.SomeValue{}, result)
		innerValue := result.(*interpreter.SomeValue).InnerValue()
		assert.Equal(t, interpreter.UInt8Value(4), innerValue)
	})

	t.Run("multiple parameters", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): [Type] {
                let f: fun(Int, String): [Type] = actualFunc
                return f(1, "hello")
            }

            fun actualFunc(_ a: Int?, _ b: String?): [Type] {
                return [
                    a.getType(),
                    b.getType()
                ]
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		AssertValuesEqual(t,
			inter,
			interpreter.NewArrayValue(
				inter,
				interpreter.NewVariableSizedStaticType(
					inter,
					interpreter.PrimitiveStaticTypeMetaType,
				),
				common.ZeroAddress,
				interpreter.NewTypeValue(
					inter,
					interpreter.NewOptionalStaticType(
						inter,
						interpreter.PrimitiveStaticTypeInt,
					),
				),
				interpreter.NewTypeValue(
					inter,
					interpreter.NewOptionalStaticType(
						inter,
						interpreter.PrimitiveStaticTypeString,
					),
				),
			),
			result,
		)
	})

	t.Run("nested optional", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): Type {
                let f: fun(Int): Type = actualFunc
                return f(4)
            }

            fun actualFunc(_ a: Int??): Type {
                return a.getType()
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t,
			interpreter.NewTypeValue(
				inter,
				interpreter.NewOptionalStaticType(
					inter,
					interpreter.NewOptionalStaticType(
						inter,
						interpreter.PrimitiveStaticTypeInt,
					),
				),
			),
			result,
		)
	})

	t.Run("same types, no boxing needed", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(t, `
            fun test(): Int {
                let f: fun(Int): Int = identity
                return f(42)
            }

            fun identity(_ a: Int): Int {
                return a
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		assert.Equal(t, interpreter.NewUnmeteredIntValueFromInt64(42), result)
	})

	t.Run("reference type parameter", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndPrepare(
			t,
			`
            fun test() {
                let funcWithAuthReferenceParam: fun(auth(Mutate) &Int) = funcWithNonAuthReferenceParam
                funcWithAuthReferenceParam(&4 as auth(Mutate) &Int)
            }

            fun funcWithNonAuthReferenceParam(_ ref: &Int) {
                let any: AnyStruct = ref
                let authRef = any as! auth(Mutate) &Int
            }
        `,
		)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		var forceCastTypeMismatchErr *interpreter.ForceCastTypeMismatchError
		require.ErrorAs(t, err, &forceCastTypeMismatchErr)

		assert.Equal(t,
			common.TypeID("auth(Mutate)&Int"),
			forceCastTypeMismatchErr.ExpectedType.ID(),
		)
		assert.Equal(t,
			common.TypeID("&Int"),
			forceCastTypeMismatchErr.ActualType.ID(),
		)
	})
}
