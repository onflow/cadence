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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckInvalidFunctionCallWithTooFewArguments(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          return f()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InsufficientArgumentsError{}, errs[0])
}

func TestCheckFunctionCallWithArgumentLabel(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(x: 1)
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionCallWithoutArgumentLabel(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(_ x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(1)
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFunctionCallWithNotRequiredArgumentLabel(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(_ x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(x: 1)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
}

func TestCheckIndirectFunctionCallWithoutArgumentLabel(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          let g = f
          return g(1)
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionCallMissingArgumentLabel(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(1)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[0])
}

func TestCheckFunctionCallIncorrectArgumentLabel(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(y: 1)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
}

func TestCheckInvalidFunctionCallWithTooManyArguments(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(2, 3)
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ExcessiveArgumentsError{}, errs[0])
	assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
}

func TestCheckInvalidFunctionCallOfBool(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int {
          return true()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotCallableError{}, errs[0])
}

func TestCheckInvalidFunctionCallOfInteger(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int {
          return 2()
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotCallableError{}, errs[0])
}

func TestCheckInvalidFunctionCallWithWrongType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(x: true)
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidFunctionCallWithWrongTypeAndMissingArgumentLabel(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int): Int {
          return x
      }

      fun test(): Int {
          return f(true)
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

	assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
}

func TestCheckInvocationOfFunctionFromStructFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(x: Int) {}

      struct Y {
        fun x() {
          f(x: 1)
        }
      }
    `)
	require.NoError(t, err)
}

func TestCheckInvalidStructFunctionInvocation(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      struct Y {
        fun x() {
          x()
        }
      }
    `)
	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvocationOfFunctionFromStructFunctionWithSameName(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun x(y: Int) {}

      struct Y {
        // struct function and global function have same name
        fun x() {
          x(y: 1)
        }
      }
    `)
	require.NoError(t, err)
}

func TestCheckIntricateIntegerBinaryExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x: Int8 = 100
      let y = (Int8(90) + Int8(10)) == x
    `)
	require.NoError(t, err)
}

func TestCheckInvocationWithOnlyVarargs(t *testing.T) {

	t.Parallel()

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.NewInterpreterStandardLibraryStaticFunction(
		"foo",
		&sema.FunctionType{
			ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			Arity:                &sema.Arity{Max: -1},
		},
		"",
		nil,
	))

	_, err := ParseAndCheckWithOptions(t,
		`
            access(all) fun test() {
                foo(1)
            }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckArgumentLabels(t *testing.T) {

	t.Parallel()

	t.Run("function", func(t *testing.T) {

		t.Run("", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              fun test(foo bar: Int, baz: String) {}

              let t = test(x: 1, "2")
            `)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})

		t.Run("imported", func(t *testing.T) {

			t.Parallel()

			importedChecker, err := ParseAndCheckWithOptions(t,
				`
                  fun test(foo bar: Int, baz: String) {}
                `,
				ParseAndCheckOptions{
					Location: ImportedLocation,
				},
			)

			require.NoError(t, err)

			_, err = ParseAndCheckWithOptions(t,
				`
                  import "imported"

                  let t = test(x: 1, "2")
                `,
				ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
							return sema.ElaborationImport{
								Elaboration: importedChecker.Elaboration,
							}, nil
						},
					},
				},
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})
	})

	t.Run("composite function", func(t *testing.T) {

		t.Run("", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              struct Test {
                  fun test(foo bar: Int, baz: String) {}
              }

              let t = Test().test(x: 1, "2")
            `)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})

		t.Run("imported", func(t *testing.T) {

			t.Parallel()

			importedChecker, err := ParseAndCheckWithOptions(t,
				`
                  struct Test {
                      fun test(foo bar: Int, baz: String) {}
                  }
                `,
				ParseAndCheckOptions{
					Location: ImportedLocation,
				},
			)

			require.NoError(t, err)

			_, err = ParseAndCheckWithOptions(t,
				`
                  import "imported"

                  let t = Test().test(x: 1, "2")
                `,
				ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
							return sema.ElaborationImport{
								Elaboration: importedChecker.Elaboration,
							}, nil
						},
					},
				},
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})
	})

	t.Run("constructor", func(t *testing.T) {

		t.Run("", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              struct Test {
                  init(foo bar: Int, baz: String) {}
              }

              let t = Test(x: 1, "2")
            `)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})

		t.Run("imported", func(t *testing.T) {

			t.Parallel()

			importedChecker, err := ParseAndCheckWithOptions(t,
				`
                  struct Test {
                      init(foo bar: Int, baz: String) {}
                  }
                `,
				ParseAndCheckOptions{
					Location: ImportedLocation,
				},
			)

			require.NoError(t, err)

			_, err = ParseAndCheckWithOptions(t,
				`
                  import "imported"

                  let t = Test(x: 1, "2")
                `,
				ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
							return sema.ElaborationImport{
								Elaboration: importedChecker.Elaboration,
							}, nil
						},
					},
				},
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})
	})

	t.Run("nested constructor", func(t *testing.T) {

		t.Run("", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              contract C {
                  struct S {
                      init(foo bar: Int, baz: String) {}
                  }
              }

              let t = C.S(x: 1, "2")
            `)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})

		t.Run("imported", func(t *testing.T) {

			t.Parallel()

			importedChecker, err := ParseAndCheckWithOptions(t,
				`
                  contract C {
                      struct S {
                          init(foo bar: Int, baz: String) {}
                      }
                  }
                `,
				ParseAndCheckOptions{
					Location: ImportedLocation,
				},
			)

			require.NoError(t, err)

			_, err = ParseAndCheckWithOptions(t,
				`
                  import "imported"

                  let t = C.S(x: 1, "2")
                `,
				ParseAndCheckOptions{
					CheckerConfig: &sema.Config{
						ImportHandler: func(_ *sema.Checker, _ common.Location, _ ast.Range) (sema.Import, error) {
							return sema.ElaborationImport{
								Elaboration: importedChecker.Elaboration,
							}, nil
						},
					},
				},
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.IncorrectArgumentLabelError{}, errs[0])
			assert.IsType(t, &sema.MissingArgumentLabelError{}, errs[1])
		})

	})
}

func TestCheckInvocationWithIncorrectTypeParameter(t *testing.T) {

	t.Parallel()

	// function type has incorrect type-arguments:
	// 	`fun Foo<T: AnyStruct>(_ a: R)`
	//
	funcType := &sema.FunctionType{
		ReturnTypeAnnotation: sema.VoidTypeAnnotation,
		TypeParameters: []*sema.TypeParameter{
			{
				Name:      "T",
				TypeBound: sema.AnyStructType,
			},
		},
		Parameters: []sema.Parameter{
			{
				Label:      sema.ArgumentLabelNotRequired,
				Identifier: "a",
				TypeAnnotation: sema.NewTypeAnnotation(
					&sema.GenericType{
						TypeParameter: &sema.TypeParameter{
							Name:      "R", // This is an incorrect/undefined type-parameter
							TypeBound: sema.AnyStructType,
						},
					},
				),
			},
		},
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(stdlib.NewInterpreterStandardLibraryStaticFunction(
		"foo",
		funcType,
		"",
		nil, // no need, we only type-check
	))

	_, err := ParseAndCheckWithOptions(t,
		`
            access(all) fun test() {
                foo<String>("hello")
            }
        `,
		ParseAndCheckOptions{
			CheckerConfig: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvocationTypeInferenceError{}, errs[0])
}
