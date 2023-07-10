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

package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckFunctionTestConditions(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: Int) {
          pre {
              x != 0
          }
          post {
              x == 0
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionEmitConditions(t *testing.T) {

	t.Parallel()

	t.Run("existing types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo()
          event Bar()

          fun test(x: Int) {
              pre {
                  emit Foo()
              }
              post {
                  emit Bar()
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("non-existing types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int) {
              pre {
                  emit Foo()
              }
              post {
                  emit Bar()
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		var notDeclaredErr *sema.NotDeclaredError

		require.ErrorAs(t, errs[0], &notDeclaredErr)
		assert.Equal(t, "Foo", notDeclaredErr.Name)

		require.ErrorAs(t, errs[1], &notDeclaredErr)
		assert.Equal(t, "Bar", notDeclaredErr.Name)
	})
}

func TestCheckInvalidFunctionConditionValueReference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      event Foo(y: Int)
      event Bar(z: Int)

      fun test(x: Int) {
          pre {
              y == 0
              emit Foo(y: a)
          }
          post {
              z == 0
              emit Bar(z: b)
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 4)

	var notDeclaredErr *sema.NotDeclaredError

	require.ErrorAs(t, errs[0], &notDeclaredErr)
	assert.Equal(t, "y", notDeclaredErr.Name)

	require.ErrorAs(t, errs[1], &notDeclaredErr)
	assert.Equal(t, "a", notDeclaredErr.Name)

	require.ErrorAs(t, errs[2], &notDeclaredErr)
	assert.Equal(t, "z", notDeclaredErr.Name)

	require.ErrorAs(t, errs[3], &notDeclaredErr)
	assert.Equal(t, "b", notDeclaredErr.Name)
}

func TestCheckInvalidFunctionPostEmitConditionBefore(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: Int) {
          post {
              emit before(x)
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	require.IsType(t, &sema.InvalidEmitConditionError{}, errs[0])
	require.IsType(t, &sema.EmitNonEventError{}, errs[1])
}

func TestCheckInvalidFunctionNonBoolCondition(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: Int) {
          pre {
              1
          }
          post {
              2
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
}

func TestCheckFunctionPostConditionWithBefore(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          fun test(x: Int) {
              post {
                  before(x) != 0
              }
          }
        `)

		require.NoError(t, err)

		assert.Equal(t, 1, checker.Elaboration.VariableDeclarationTypesCount())
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(x: Int) {
              post {
                  emit Foo(x: before(x))
              }
          }
        `)

		require.NoError(t, err)

		assert.Equal(t, 1, checker.Elaboration.VariableDeclarationTypesCount())
	})
}

func TestCheckFunctionPostConditionWithBeforeNotDeclaredUse(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              post {
                  before(x) != 0
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test() {
              post {
                  emit Foo(x: before(x))
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckInvalidFunctionPostConditionWithBeforeAndNoArgument(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int) {
              post {
                  before() != 0
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ArgumentCountError{}, errs[0])
		assert.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(x: Int) {
              post {
                  emit Foo(x: before())
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ArgumentCountError{}, errs[0])
		assert.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[1])
	})
}

func TestCheckInvalidFunctionPreConditionWithBefore(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int) {
              pre {
                  before(x) != 0
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.Equal(t,
			"before",
			errs[0].(*sema.NotDeclaredError).Name,
		)
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(x: Int) {
              pre {
                  emit Foo(x: before(x))
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.Equal(t,
			"before",
			errs[0].(*sema.NotDeclaredError).Name,
		)
	})
}

func TestCheckInvalidFunctionWithBeforeVariableAndPostConditionWithBefore(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int) {
              post {
                  before(x) == 0
              }
              let before = 0
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(x: Int) {
              post {
                  emit Foo(x: before(x))
              }
              let before = 0
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	})
}

func TestCheckFunctionWithBeforeVariable(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: Int) {
          let before = 0
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionPostCondition(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(x: Int): Int {
              post {
                  y == 0
              }
              let y = x
              return y
          }
        `)

		require.NoError(t, err)
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(x: Int): Int {
              post {
                  emit Foo(x: y)
              }
              let y = x
              return y
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidFunctionPreConditionWithResult(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              pre {
                  result == 0
              }
              return 0
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.Equal(t,
			"result",
			errs[0].(*sema.NotDeclaredError).Name,
		)
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(): Int {
              pre {
                  emit Foo(x: result)
              }
              return 0
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.Equal(t,
			"result",
			errs[0].(*sema.NotDeclaredError).Name,
		)
	})
}

func TestCheckInvalidFunctionPostConditionWithResultWrongType(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              post {
                  result == true
              }
              return 0
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Bool)

          fun test(): Int {
              post {
                  emit Foo(x: result)
              }
              return 0
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckFunctionPostConditionWithResult(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              post {
                  result == 0
              }
              return 0
          }
        `)

		require.NoError(t, err)
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(): Int {
              post {
                  emit Foo(x: result)
              }
              return 0
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidFunctionPostConditionWithResult(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              post {
                  result == 0
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.Equal(t,
			"result",
			errs[0].(*sema.NotDeclaredError).Name,
		)
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test() {
              post {
                  emit Foo(x: result)
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
		assert.Equal(t,
			"result",
			errs[0].(*sema.NotDeclaredError).Name,
		)
	})
}

func TestCheckFunctionWithoutReturnTypeAndLocalResultAndPostConditionWithResult(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              post {
                  result == 0
              }
              let result = 0
          }
        `)

		require.NoError(t, err)
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test() {
              post {
                  emit Foo(x: result)
              }
              let result = 0
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckFunctionWithoutReturnTypeAndResultParameterAndPostConditionWithResult(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(result: Int) {
              post {
                  result == 0
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(result: Int) {
              post {
                  emit Foo(x: result)
              }
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidFunctionWithReturnTypeAndLocalResultAndPostConditionWithResult(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(): Int {
              post {
                  result == 2
              }
              let result = 1
              return result * 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(): Int {
              post {
                  emit Foo(x: result)
              }
              let result = 1
              return result * 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	})
}

func TestCheckInvalidFunctionWithReturnTypeAndResultParameterAndPostConditionWithResult(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(result: Int): Int {
              post {
                  result == 2
              }
              return result * 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	})

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(result: Int): Int {
              post {
                  emit Foo(x: result)
              }
              return result * 2
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.RedeclarationError{}, errs[0])
	})
}

func TestCheckInvalidFunctionPostConditionWithFunction(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              post {
				  (view fun (): Int { return 2 })() == 2
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FunctionExpressionInConditionError{}, errs[0])
	})

	t.Run("emit condition", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test() {
              post {
                  emit Foo(x: (view fun (): Int { return 2 })())
              }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.FunctionExpressionInConditionError{}, errs[0])
	})
}

func TestCheckFunctionPostTestConditionWithMessageUsingStringLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          post {
             1 == 2: "nope"
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidFunctionPostTestConditionWithMessageUsingBooleanLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          post {
             1 == 2: true
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckFunctionPostTestConditionWithMessageUsingResult(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(): String {
          post {
             1 == 2: result
          }
          return ""
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionPostTestConditionWithMessageUsingBefore(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: String) {
          post {
             1 == 2: before(x)
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionPostTestConditionWithMessageUsingParameter(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(x: String) {
          post {
             1 == 2: x
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckFunctionWithPostTestConditionAndResourceResult(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

        resource R {}

        resource Container {

            let resources: @{String: R}

            init() {
                self.resources <- {"original": <-create R()}
            }

            fun withdraw(): @R {
                post {
                    self.add(<-result)
                }
                return <- self.resources.remove(key: "original")!
            }

            fun add(_ r: @R): Bool {
                self.resources["duplicate"] <-! r
                return true
            }

            destroy() {
                destroy self.resources
            }
        }
    `)

	errs := RequireCheckerErrors(t, err, 3)

	require.IsType(t, &sema.InvalidMoveOperationError{}, errs[1])
	require.IsType(t, &sema.TypeMismatchError{}, errs[2])
	require.IsType(t, &sema.PurityError{}, errs[0])
}

// TestCheckConditionCreateBefore tests if the AST expression extractor properly handles
// that the rewritten expression of a create expression may not be an invocation expression.
// For example, this is the case for the expression `create before(...)`,
// where the sema.BeforeExtractor returns an IdentifierExpression.
func TestCheckInvalidConditionCreateBefore(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(n: Int) {
          post {
              create before(n)
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	var notCallableErr *sema.NotCallableError
	require.ErrorAs(t, errs[0], &notCallableErr)
	require.Equal(t, sema.IntType, notCallableErr.Type)
}

func TestCheckRewrittenPostConditions(t *testing.T) {

	t.Parallel()

	t.Run("test condition", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
          fun test(x: Int) {
              post {
                  before(x) == 0
              }
          }
        `)
		require.NoError(t, err)

		declarations := checker.Program.Declarations()
		require.Len(t, declarations, 1)
		firstDeclaration := declarations[0]

		require.IsType(t, &ast.FunctionDeclaration{}, firstDeclaration)
		functionDeclaration := firstDeclaration.(*ast.FunctionDeclaration)

		postConditions := functionDeclaration.FunctionBlock.PostConditions
		postConditionsRewrite := checker.Elaboration.PostConditionsRewrite(postConditions)

		require.Len(t, postConditionsRewrite.RewrittenPostConditions, 1)
		require.Len(t, postConditionsRewrite.BeforeStatements, 1)

		beforeStatement := postConditionsRewrite.BeforeStatements[0]

		ast.Inspect(beforeStatement, func(element ast.Element) bool {
			if element != nil {
				assert.Positive(t, element.StartPosition().Line)
			}
			return true
		})

	})

	t.Run("emit condition", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
          event Foo(x: Int)

          fun test(x: Int) {
              post {
                  emit Foo(x: before(x))
              }
          }
        `)
		require.NoError(t, err)

		declarations := checker.Program.Declarations()
		require.Len(t, declarations, 2)
		secondDeclaration := declarations[1]

		require.IsType(t, &ast.FunctionDeclaration{}, secondDeclaration)
		functionDeclaration := secondDeclaration.(*ast.FunctionDeclaration)

		postConditions := functionDeclaration.FunctionBlock.PostConditions
		postConditionsRewrite := checker.Elaboration.PostConditionsRewrite(postConditions)

		require.Len(t, postConditionsRewrite.RewrittenPostConditions, 1)
		require.Len(t, postConditionsRewrite.BeforeStatements, 1)

		beforeStatement := postConditionsRewrite.BeforeStatements[0]

		ast.Inspect(beforeStatement, func(element ast.Element) bool {
			if element != nil {
				assert.Positive(t, element.StartPosition().Line)
			}
			return true
		})

	})
}
