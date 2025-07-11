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
	"github.com/onflow/cadence/parser"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckReferenceInFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          test
      }
    `)

	require.NoError(t, err)
}

func TestCheckParameterNameWithFunctionName(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(test: Int) {
          test
      }
    `)

	require.NoError(t, err)
}

func TestCheckMutuallyRecursiveFunctions(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun isEven(_ n: Int): Bool {
          if n == 0 {
              return true
          }
          return isOdd(n - 1)
      }

      fun isOdd(_ n: Int): Bool {
          if n == 0 {
              return false
          }
          return isEven(n - 1)
      }
    `)

	require.NoError(t, err)
}

func TestCheckMutuallyRecursiveScoping(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f(): Int {
         return g()
      }

      let x = f()
      let y = 0

      fun g(): Int {
          return y
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFunctionDeclarations(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          fun foo() {}
          fun foo() {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckInvalidFunctionRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo() {
          fun foo() {}
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckFunctionAccess(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       access(all) fun test() {}
    `)

	require.NoError(t, err)
}

func TestCheckReturnWithoutExpression(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
       fun returnNothing() {
           return
       }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionUseInsideFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo() {
          foo()
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionReturnFunction(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun foo(): fun(Int): Void {
          return bar
      }

      fun bar(_ n: Int) {}
    `)

	require.NoError(t, err)
}

func TestCheckInvalidParameterTypes(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: X, y: Y) {}
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])

}

func TestCheckInvalidParameterNameRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(a: Int, a: Int) {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckParameterRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(a: Int) {
          let a = 1
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidParameterAssignment(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(a: Int) {
          a = 1
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.AssignmentToConstantError{}, errs[0])
}

func TestCheckInvalidArgumentLabelRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x a: Int, x b: Int) {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.RedeclarationError{}, errs[0])
}

func TestCheckArgumentLabelRedeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(_ a: Int, _ b: Int) {}
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFunctionDeclarationReturnValue(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): Int {
          return true
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidResourceCapturingThroughVariable(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Kitty {}

      fun makeKittyCloner(): fun(): @Kitty {
          let kitty <- create Kitty()
          return fun (): @Kitty {
              return <-kitty
          }
      }

      let test = makeKittyCloner()
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
}

func TestCheckInvalidResourceCapturingThroughParameter(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Kitty {}

      fun makeKittyCloner(kitty: @Kitty): fun(): @Kitty {
          return fun (): @Kitty {
              return <-kitty
          }
      }

      let test = makeKittyCloner(kitty: <-create Kitty())
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
}

func TestCheckInvalidSelfResourceCapturing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource Kitty {
          fun makeCloner(): fun(): @Kitty {
              return fun (): @Kitty {
                  return <-self
              }
          }
      }

      let kitty <- create Kitty()
      let test = kitty.makeCloner()
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
	assert.IsType(t, &sema.InvalidSelfInvalidationError{}, errs[1])
}

func TestCheckInvalidResourceCapturingJustMemberAccess(t *testing.T) {

	t.Parallel()

	// Resource capturing even just for read access (e.g. reading a member) is invalid

	_, err := ParseAndCheck(t, `
      resource Kitty {
          let id: Int

          init(id: Int) {
              self.id = id
          }
      }

      fun makeKittyIdGetter(): fun(): Int {
          let kitty <- create Kitty(id: 1)
          let getId = fun (): Int {
              return kitty.id
          }
          destroy kitty
          return getId
      }

      let test = makeKittyIdGetter()
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceCapturingError{}, errs[0])
}

func TestCheckFunctionWithResult(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     fun test(): Int {
         let result = 0
         return result
     }
   `)
	require.NoError(t, err)
}

func TestCheckInvalidFunctionWithResultAndPostCondition(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
     fun test(): Int {
         post {
             result == 0
         }
         let result = 0
         return result
     }
   `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResultVariableConflictError{}, errs[0])
}

func TestCheckFunctionNonExistingField(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun f() {}

      let x = f.y
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckStaticFunctionDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithOptions(t,
		`
          static fun test() {}
        `,
		ParseAndCheckOptions{
			ParseOptions: parser.Config{
				StaticModifierEnabled: true,
			},
		},
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidStaticModifierError{}, errs[0])
}

func TestCheckNativeFunctionDeclaration(t *testing.T) {

	t.Parallel()

	t.Run("disabled", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
              native fun test(): Int {}
            `,
			ParseAndCheckOptions{
				ParseOptions: parser.Config{
					NativeModifierEnabled: true,
				},
				CheckerConfig: &sema.Config{
					AllowNativeDeclarations: false,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNativeModifierError{}, errs[0])
	})

	t.Run("enabled, valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
              native fun test(): Int {}
            `,
			ParseAndCheckOptions{
				ParseOptions: parser.Config{
					NativeModifierEnabled: true,
				},
				CheckerConfig: &sema.Config{
					AllowNativeDeclarations: true,
				},
			},
		)

		require.NoError(t, err)
	})

	t.Run("enabled, invalid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithOptions(t,
			`
              native fun test(): Int {
                  return 1
              }
            `,
			ParseAndCheckOptions{
				ParseOptions: parser.Config{
					NativeModifierEnabled: true,
				},
				CheckerConfig: &sema.Config{
					AllowNativeDeclarations: true,
				},
			},
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NativeFunctionWithImplementationError{}, errs[0])
	})

	t.Run("enabled, composite", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithOptions(t,
			`
              struct S {
                  native fun test(foo: String): Int {}
              }
            `,
			ParseAndCheckOptions{
				ParseOptions: parser.Config{
					NativeModifierEnabled: true,
				},
				CheckerConfig: &sema.Config{
					AllowNativeDeclarations: true,
				},
			},
		)
		require.NoError(t, err)

		sType := RequireGlobalType(t, checker.Elaboration, "S")
		require.NotNil(t, sType)

		const testFunctionIdentifier = "test"
		testMemberResolver, ok := sType.GetMembers()[testFunctionIdentifier]
		require.True(t, ok)

		assert.Equal(t,
			common.DeclarationKindFunction,
			testMemberResolver.Kind,
		)

		member := testMemberResolver.Resolve(nil, testFunctionIdentifier, ast.EmptyRange, nil)

		assert.Equal(t,
			sema.NewTypeAnnotation(&sema.FunctionType{
				Parameters: []sema.Parameter{
					{
						Identifier:     "foo",
						TypeAnnotation: sema.NewTypeAnnotation(sema.StringType),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
			}),
			member.TypeAnnotation,
		)
	})
}

func TestCheckResultVariable(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            access(all) resource R {
                access(all) let id: UInt64
                init() {
                    self.id = 1
                }
            }

            access(all) fun main(): @R  {
                post {
                    result.id == 1234: "Invalid id"
                }
                return <- create R()
            }`,
		)

		require.NoError(t, err)
	})

	t.Run("optional resource", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            access(all) resource R {
                access(all) let id: UInt64
                init() {
                    self.id = 1
                }
            }

            access(all) fun main(): @R?  {
                post {
                    result!.id == 1234: "invalid id"
                }
                return nil
            }`,
		)

		require.NoError(t, err)
	})
}

func TestCheckViewFunctionWithErrors(t *testing.T) {

	t.Parallel()

	t.Run("index assignment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            view fun foo() {
                a[b] = 1
            }`,
		)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.PurityError{}, errs[1])
	})

	t.Run("member assignment", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            view fun foo() {
                a.b = 1
            }`,
		)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.IsType(t, &sema.PurityError{}, errs[1])
	})
}

func TestCheckInvalidFunctionSubtyping(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      entitlement E

      fun test() {
          var f: fun (&R) = fun(ref: &R) {}
          f = fun(ref: auth(E) &R) {}
      }
    `)
	errs := RequireCheckerErrors(t, err, 1)
	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

	var errTypeMismatch *sema.TypeMismatchError
	require.ErrorAs(t, err, &errTypeMismatch)
	assert.Equal(t, 8, errTypeMismatch.StartPos.Line)
	assert.Equal(t,
		common.TypeID("fun(&S.test.R):Void"),
		errTypeMismatch.ExpectedType.ID(),
	)
	assert.Equal(t,
		common.TypeID("fun(auth(S.test.E)&S.test.R):Void"),
		errTypeMismatch.ActualType.ID(),
	)
}

func TestCheckGenericFunctionSubtyping(t *testing.T) {

	t.Parallel()

	parseAndCheck := func(tt *testing.T, code string, boundType1, boundType2 sema.Type) (*sema.Checker, error) {
		typeParameter1 := &sema.TypeParameter{
			Name:      "T",
			TypeBound: boundType1,
		}

		function1 := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"foo",
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter1,
				},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			"",
			nil,
		)

		typeParameter2 := &sema.TypeParameter{
			Name:      "T",
			TypeBound: boundType2,
		}

		function2 := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"bar",
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter2,
				},
				ReturnTypeAnnotation: sema.VoidTypeAnnotation,
			},
			"",
			nil,
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(function1)
		baseValueActivation.DeclareValue(function2)

		return ParseAndCheckWithOptions(tt,
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

	t.Run("same bound type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func = foo  // fun<T AnyStruct>(): Void
                func = bar      // fun<T AnyStruct>(): Void
            }`,
			sema.AnyStructType,
			sema.AnyStructType,
		)

		require.NoError(t, err)
	})

	t.Run("different bound types", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func = foo  // fun<T Integer>(): Void
                func = bar      // fun<T Path>(): Void
            }`,
			sema.IntegerType,
			sema.PathType,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchError)
	})

	t.Run("second bound type is a subtype", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func = foo  // fun<T AnyStruct>(): Void
                func = bar      // fun<T Integer>(): Void
            }`,
			sema.AnyStructType,
			sema.IntegerType,
		)

		errors := RequireCheckerErrors(t, err, 1)
		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchError)
	})

	t.Run("second bound type is a super-type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func = foo  // fun<T Integer>(): Void
                func = bar      // fun<T AnyStruct>(): Void
            }`,
			sema.IntegerType,
			sema.AnyStructType,
		)

		errors := RequireCheckerErrors(t, err, 1)
		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchError)
	})

	t.Run("target has no bound type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func = foo  // fun<T>(): Void
                func = bar      // fun<T AnyStruct>(): Void
            }`,
			nil,
			sema.AnyStructType,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchError)
	})

	t.Run("value has no bound type", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func = foo  // fun<T AnyStruct>(): Void
                func = bar      // fun<T>(): Void
            }`,
			sema.AnyStructType,
			nil,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchError)
	})

	t.Run("generic function to a non-generic var", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func1: fun():Void = foo   // fun<T AnyStruct>(): Void
                var func2: fun():Void = bar   // fun<T>(): Void
            }
        `,
			sema.AnyStructType,
			nil,
		)

		errors := RequireCheckerErrors(t, err, 2)

		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchError)

		require.ErrorAs(t, errors[1], &typeMismatchError)
	})

	t.Run("no bound types, equal param and return types", func(t *testing.T) {
		t.Parallel()

		_, err := parseAndCheck(t, `
            fun test() {
                var func = foo  // fun<T>(): Void
                func = bar      // fun<T>(): Void
            }`,
			nil,
			nil,
		)

		require.NoError(t, err)
	})

	t.Run("no bound types, different return types", func(t *testing.T) {
		typeParameter1 := &sema.TypeParameter{
			Name: "T",
		}

		function1 := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"foo",
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter1,
				},
				ReturnTypeAnnotation: sema.IntTypeAnnotation,
			},
			"",
			nil,
		)

		typeParameter2 := &sema.TypeParameter{
			Name: "T",
		}

		function2 := stdlib.NewInterpreterStandardLibraryStaticFunction(
			"bar",
			&sema.FunctionType{
				TypeParameters: []*sema.TypeParameter{
					typeParameter2,
				},
				ReturnTypeAnnotation: sema.PathTypeAnnotation,
			},
			"",
			nil,
		)

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(function1)
		baseValueActivation.DeclareValue(function2)

		_, err := ParseAndCheckWithOptions(t,
			`fun test() {
                var func = foo  // fun<T>(): Int
                func = bar      // fun<T>(): Path
		    }`,
			ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
		)

		errors := RequireCheckerErrors(t, err, 1)
		var typeMismatchError *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchError)
	})
}
