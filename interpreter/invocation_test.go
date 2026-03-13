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

	"github.com/onflow/cadence/bbq/commons"
	"github.com/onflow/cadence/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"

	. "github.com/onflow/cadence/bbq/test_utils"
	"github.com/onflow/cadence/bbq/vm"
	"github.com/onflow/cadence/bbq/vm/test"
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

	// Intentionally passing wrong type of value
	_, err := inter.InvokeUncheckedForTestingOnly("test", value) //nolint:staticcheck
	RequireError(t, err)

	var memberAccessTypeError *interpreter.MemberAccessTypeError
	require.ErrorAs(t, err, &memberAccessTypeError)
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

	t.Run("optional parameter in generic function", func(t *testing.T) {
		t.Parallel()

		// Scenario:
		//
		// struct interface {
		//    fun(_ a: T)
		// }
		//
		// struct {
		//    fun(_ a: T?) {}
		// }
		//
		// Usage:
		//   var si: {SI} = S()
		//   si.genericFunction<Int>(4)
		//
		// This should invoke the `genericFunction` implementation from struct `S`,
		// which has a generic-optional parameter type at runtime.

		const methodName = "genericFunction"

		// Use `nil`/built-in location, since the vm currently
		// only allows injecting builtin values.
		var location common.Location = nil

		typeParameter := &sema.TypeParameter{
			Name: "T",
		}

		// Interface type `SI`

		// Interface method type `fun(_ a: T)`.
		// Note the parameter type is `T`.
		interfaceMethodType := &sema.FunctionType{
			TypeParameters: []*sema.TypeParameter{
				typeParameter,
			},
			Parameters: []sema.Parameter{
				{
					Identifier: "a",
					Label:      sema.ArgumentLabelNotRequired,
					TypeAnnotation: sema.NewTypeAnnotation(
						&sema.GenericType{
							TypeParameter: typeParameter,
						},
					),
				},
			},
			ReturnTypeAnnotation: sema.VoidTypeAnnotation,
		}

		structInterfaceType := &sema.InterfaceType{
			Location:      location,
			Identifier:    "SI",
			CompositeKind: common.CompositeKindStructure,
			Members:       &sema.StringMemberOrderedMap{},
		}

		structInterfaceType.Members.Set(methodName, sema.NewUnmeteredPublicFunctionMember(
			structInterfaceType,
			methodName,
			interfaceMethodType,
			"",
		))

		// Concrete type `S`

		// Concrete method type `fun(_ a: T?)`
		// Note the parameter type is optional `T?`.
		concreteMethodType := &sema.FunctionType{
			TypeParameters: []*sema.TypeParameter{
				typeParameter,
			},
			Parameters: []sema.Parameter{
				{
					Identifier: "a",
					Label:      sema.ArgumentLabelNotRequired,
					TypeAnnotation: sema.NewTypeAnnotation(
						&sema.OptionalType{
							Type: &sema.GenericType{
								TypeParameter: typeParameter,
							},
						},
					),
				},
			},
			ReturnTypeAnnotation: sema.VoidTypeAnnotation,
		}

		structType := &sema.CompositeType{
			Location:   location,
			Identifier: "S",
			Kind:       common.CompositeKindStructure,
			Members:    &sema.StringMemberOrderedMap{},
			ExplicitInterfaceConformances: []*sema.InterfaceType{
				structInterfaceType,
			},
		}

		structType.Members.Set(methodName, sema.NewUnmeteredPublicFunctionMember(
			structType,
			methodName,
			concreteMethodType,
			"",
		))

		structStaticType := interpreter.NewCompositeStaticTypeComputeTypeID(
			nil,
			location,
			structType.Identifier,
		)

		// Declare Types
		baseTypeActivation := sema.NewVariableActivation(sema.BaseTypeActivation)
		baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
			Name: structInterfaceType.Identifier,
			Type: structInterfaceType,
			Kind: common.DeclarationKindStructureInterface,
		})
		baseTypeActivation.DeclareType(stdlib.StandardLibraryType{
			Name: structType.Identifier,
			Type: structType,
			Kind: common.DeclarationKindStructure,
		})

		// Concrete function value

		methodInvoked := false

		function := func(
			_ interpreter.NativeFunctionContext,
			_ interpreter.TypeArgumentsIterator,
			_ interpreter.ArgumentTypesIterator,
			_ interpreter.Value,
			args []interpreter.Value,
		) interpreter.Value {
			methodInvoked = true
			assert.Len(t, args, 1)

			if *compile {
				// TODO: Update the assert once the type-parameter resolving is
				// supported for invocations in the VM.
				assert.Equal(
					t,
					interpreter.NewUnmeteredIntValueFromInt64(4),
					args[0],
				)
			} else {
				assert.Equal(
					t,
					interpreter.NewUnmeteredSomeValueNonCopying(
						interpreter.NewUnmeteredIntValueFromInt64(4),
					),
					args[0],
				)
			}
			return interpreter.Void
		}

		var functionValue interpreter.FunctionValue
		if *compile {
			functionValue = vm.NewNativeFunctionValue(methodName, concreteMethodType, function)
		} else {
			functionValue = interpreter.NewUnmeteredStaticHostFunctionValueFromNativeFunction(concreteMethodType, function)
		}

		compositeTypeHandler := func(location common.Location, typeID interpreter.TypeID) *sema.CompositeType {
			if typeID == "S" {
				return structType
			}

			return nil
		}

		interfaceTypeHandler := func(location common.Location, typeID interpreter.TypeID) *sema.InterfaceType {
			if typeID == "SI" {
				return structInterfaceType
			}

			return nil
		}

		const code = `
            fun test(si: {SI}) {
			    // The interface function has an 'Int' parameter,
			    // whereas the actual runtime value (concrete method) has an optional 'Int?' parameter.
			    si.genericFunction<Int>(4)
		    }
        `

		var invokable Invokable

		if *compile {
			programs := CompiledPrograms{}
			program := ParseCheckAndCompileCodeWithOptions(t,
				code,
				location,
				ParseCheckAndCompileOptions{
					ParseAndCheckOptions: &ParseAndCheckOptions{
						CheckerConfig: &sema.Config{
							BaseTypeActivationHandler: func(_ common.Location) *sema.VariableActivation {
								return baseTypeActivation
							},
						},
					},
				},
				programs,
			)

			vmConfig := test.PrepareVMConfig(t, nil, nil)

			vmConfig.BuiltinGlobalsProvider = func(location common.Location) *activations.Activation[vm.Variable] {
				activation := activations.NewActivation(nil, vm.DefaultBuiltinGlobals())

				functionVariable := &interpreter.SimpleVariable{}
				functionVariable.InitializeWithValue(functionValue)
				activation.Set(
					commons.TypeQualifiedName(structType, methodName),
					functionVariable,
				)

				return activation
			}

			vmConfig.CompositeTypeHandler = compositeTypeHandler
			vmConfig.InterfaceTypeHandler = interfaceTypeHandler

			programVM := vm.NewVM(
				location,
				program,
				vmConfig,
			)

			invokable = test_utils.NewVMInvokable(programVM, programs[location].DesugaredElaboration)

		} else {
			var err error
			invokable, err = parseCheckAndPrepareWithOptions(t,
				code,
				ParseCheckAndInterpretOptions{
					InterpreterConfig: &interpreter.Config{
						Storage:              NewUnmeteredInMemoryStorage(),
						CompositeTypeHandler: compositeTypeHandler,
						InterfaceTypeHandler: interfaceTypeHandler,
					},
					ParseAndCheckOptions: &ParseAndCheckOptions{
						CheckerConfig: &sema.Config{
							BaseTypeActivationHandler: func(_ common.Location) *sema.VariableActivation {
								return baseTypeActivation
							},
						},
					},
				},
			)
			require.NoError(t, err)
		}

		// Construct an instance from struct `S`, and pass it as an argument.

		structValue := interpreter.NewSimpleCompositeValue(
			nil,
			structType.ID(),
			structStaticType,
			nil,
			nil,
			nil,
			func(name string, context interpreter.MemberAccessibleContext) interpreter.FunctionValue {
				if name == methodName {
					return functionValue
				}

				return nil
			},
			nil,
			nil,
		)

		_, err := invokable.Invoke("test", structValue)
		require.NoError(t, err)
		require.True(t, methodInvoked)
	})
}
