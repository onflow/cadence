/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	"github.com/onflow/cadence/runtime/tests/checker"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

func TestInterpretFunctionPreCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int): Int {
          pre {
              x == 0
          }
          return x
      }
    `)

	_, err := inter.Invoke(
		"test",
		interpreter.NewIntValueFromInt64(42),
	)
	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	zero := interpreter.NewIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	assert.Equal(t, zero, value)
}

func TestInterpretFunctionPostCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int): Int {
          post {
              y == 0
          }
          let y = x
          return y
      }
    `)

	_, err := inter.Invoke(
		"test",
		interpreter.NewIntValueFromInt64(42),
	)
	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	zero := interpreter.NewIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	assert.Equal(t, zero, value)
}

func TestInterpretFunctionWithResultAndPostConditionWithResult(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int): Int {
          post {
              result == 0
          }
          return x
      }
    `)

	_, err := inter.Invoke(
		"test",
		interpreter.NewIntValueFromInt64(42),
	)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	zero := interpreter.NewIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	assert.Equal(t, zero, value)
}

func TestInterpretFunctionWithoutResultAndPostConditionWithResult(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test() {
          post {
              result == 0
          }
          let result = 0
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretFunctionPostConditionWithBefore(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = 0

      fun test() {
          pre {
              x == 0
          }
          post {
              x == before(x) + 1
          }
          x = x + 1
      }
    `)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.VoidValue{},
		value,
	)
}

func TestInterpretFunctionPostConditionWithBeforeFailingPreCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = 0

      fun test() {
          pre {
              x == 1
          }
          post {
              x == before(x) + 1
          }
          x = x + 1
      }
    `)

	_, err := inter.Invoke("test")

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		ast.ConditionKindPre,
		conditionErr.ConditionKind,
	)
}

func TestInterpretFunctionPostConditionWithBeforeFailingPostCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      var x = 0

      fun test() {
          pre {
              x == 0
          }
          post {
              x == before(x) + 2
          }
          x = x + 1
      }
    `)

	_, err := inter.Invoke("test")

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		ast.ConditionKindPost,
		conditionErr.ConditionKind,
	)
}

func TestInterpretFunctionPostConditionWithMessageUsingStringLiteral(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int): Int {
          post {
              y == 0: "y should be zero"
          }
          let y = x
          return y
      }
    `)

	_, err := inter.Invoke(
		"test",
		interpreter.NewIntValueFromInt64(42),
	)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		"y should be zero",
		conditionErr.Message,
	)

	zero := interpreter.NewIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	assert.Equal(t, zero, value)
}

func TestInterpretFunctionPostConditionWithMessageUsingResult(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: Int): String {
          post {
              y == 0: result
          }
          let y = x
          return "return value"
      }
    `)

	_, err := inter.Invoke(
		"test",
		interpreter.NewIntValueFromInt64(42),
	)
	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		"return value",
		conditionErr.Message,
	)

	zero := interpreter.NewIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	assert.Equal(t,
		interpreter.NewStringValue("return value"),
		value,
	)
}

func TestInterpretFunctionPostConditionWithMessageUsingBefore(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: String): String {
          post {
              1 == 2: before(x)
          }
          return "return value"
      }
    `)

	_, err := inter.Invoke("test", interpreter.NewStringValue("parameter value"))

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		"parameter value",
		conditionErr.Message,
	)
}

func TestInterpretFunctionPostConditionWithMessageUsingParameter(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      fun test(x: String): String {
          post {
              1 == 2: x
          }
          return "return value"
      }
    `)

	_, err := inter.Invoke("test", interpreter.NewStringValue("parameter value"))

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		"parameter value",
		conditionErr.Message,
	)
}

func TestInterpretInterfaceFunctionUseWithPreCondition(t *testing.T) {

	t.Parallel()

	for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		var setupCode, tearDownCode, identifier string

		if compositeKind == common.CompositeKindContract {
			identifier = "TestImpl"
		} else {
			interfaceType := AsInterfaceType("Test", compositeKind)

			setupCode = fmt.Sprintf(
				`let test: %[1]s%[2]s %[3]s %[4]s TestImpl%[5]s`,
				compositeKind.Annotation(),
				interfaceType,
				compositeKind.TransferOperator(),
				compositeKind.ConstructionKeyword(),
				constructorArguments(compositeKind, ""),
			)
			identifier = "test"
		}

		if compositeKind == common.CompositeKindResource {
			tearDownCode = `destroy test`
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			inter := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      pub %[1]s interface Test {
                          pub fun test(x: Int): Int {
                              pre {
                                  x > 0: "x must be positive"
                              }
                          }
                      }

                      pub %[1]s TestImpl: Test {
                          pub fun test(x: Int): Int {
                              pre {
                                  x < 2: "x must be smaller than 2"
                              }
                              return x
                          }
                      }

                      pub fun callTest(x: Int): Int {
                          %[2]s
                          let res = %[3]s.test(x: x)
                          %[4]s
                          return res
                      }
                    `,
					compositeKind.Keyword(),
					setupCode,
					identifier,
					tearDownCode,
				),
				ParseCheckAndInterpretOptions{
					Options: []interpreter.Option{
						makeContractValueHandler(nil, nil, nil),
					},
				},
			)

			_, err := inter.Invoke("callTest", interpreter.NewIntValueFromInt64(0))

			var conditionErr interpreter.ConditionError
			require.ErrorAs(t, err, &conditionErr)

			value, err := inter.Invoke("callTest", interpreter.NewIntValueFromInt64(1))
			require.NoError(t, err)

			assert.Equal(t,
				interpreter.NewIntValueFromInt64(1),
				value,
			)

			_, err = inter.Invoke("callTest", interpreter.NewIntValueFromInt64(2))

			require.ErrorAs(t, err, &conditionErr)
		})
	}
}

func TestInterpretInitializerWithInterfacePreCondition(t *testing.T) {

	t.Parallel()

	tests := map[int64]error{
		0: interpreter.ConditionError{},
		1: nil,
		2: interpreter.ConditionError{},
	}

	for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			for value, expectedError := range tests {

				t.Run(fmt.Sprint(value), func(t *testing.T) {

					var testFunction string
					if compositeKind == common.CompositeKindContract {
						// use the contract singleton, so it is loaded
						testFunction = `
					       pub fun test() {
					            TestImpl
					       }
                        `
					} else {
						interfaceType := AsInterfaceType("Test", compositeKind)

						testFunction =
							fmt.Sprintf(
								`
					               pub fun test(x: Int): %[1]s%[2]s {
					                   return %[3]s %[4]s TestImpl%[5]s
					               }
                                `,
								compositeKind.Annotation(),
								interfaceType,
								compositeKind.MoveOperator(),
								compositeKind.ConstructionKeyword(),
								constructorArguments(compositeKind, "x: x"),
							)
					}

					checker, err := checker.ParseAndCheck(t,
						fmt.Sprintf(
							`
					             pub %[1]s interface Test {
					                 init(x: Int) {
					                     pre {
					                         x > 0: "x must be positive"
					                     }
					                 }
					             }

					             pub %[1]s TestImpl: Test {
					                 init(x: Int) {
					                     pre {
					                         x < 2: "x must be smaller than 2"
					                     }
					                 }
					             }

					             %[2]s
					           `,
							compositeKind.Keyword(),
							testFunction,
						),
					)
					require.NoError(t, err)

					check := func(err error) {
						if expectedError == nil {
							require.NoError(t, err)
						} else {
							require.IsType(t,
								interpreter.Error{},
								err,
							)
							err = err.(interpreter.Error).Unwrap()

							require.IsType(t,
								expectedError,
								err,
							)
						}
					}

					uuidHandler := interpreter.WithUUIDHandler(func() (uint64, error) {
						return 0, nil
					})

					if compositeKind == common.CompositeKindContract {

						inter, err := interpreter.NewInterpreter(
							interpreter.ProgramFromChecker(checker),
							checker.Location,
							makeContractValueHandler(
								[]interpreter.Value{
									interpreter.NewIntValueFromInt64(value),
								},
								[]sema.Type{
									sema.IntType,
								},
								[]sema.Type{
									sema.IntType,
								},
							),
							uuidHandler,
						)
						require.NoError(t, err)

						err = inter.Interpret()
						require.NoError(t, err)

						_, err = inter.Invoke("test")
						check(err)
					} else {
						inter, err := interpreter.NewInterpreter(
							interpreter.ProgramFromChecker(checker),
							checker.Location,
							uuidHandler,
						)
						require.NoError(t, err)

						err = inter.Interpret()
						require.NoError(t, err)

						_, err = inter.Invoke("test", interpreter.NewIntValueFromInt64(value))
						check(err)
					}
				})
			}
		})
	}
}

func TestInterpretTypeRequirementWithPreCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpretWithOptions(t,
		`

          pub struct interface Also {
             pub fun test(x: Int) {
                 pre {
                     x >= 0: "x >= 0"
                 }
             }
          }

          pub contract interface Test {

              pub struct Nested {
                  pub fun test(x: Int) {
                      pre {
                          x >= 1: "x >= 1"
                      }
                  }
              }
          }

          pub contract TestImpl: Test {

              pub struct Nested: Also {
                  pub fun test(x: Int) {
                      pre {
                          x < 2: "x < 2"
                      }
                  }
              }
          }

          pub fun test(x: Int) {
              TestImpl.Nested().test(x: x)
          }
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
			},
		},
	)

	t.Run("-1", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(-1))

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		// NOTE: The type requirement condition (`Test.Nested`) is evaluated first,
		//  before the type's conformances (`Also`)

		assert.Equal(t, "x >= 1", conditionErr.Message)
	})

	t.Run("0", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(0))

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t, "x >= 1", conditionErr.Message)
	})

	t.Run("1", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(1))
		require.NoError(t, err)

		assert.IsType(t,
			interpreter.VoidValue{},
			value,
		)
	})

	t.Run("2", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(2))
		require.IsType(t,
			interpreter.Error{},
			err,
		)
		interpreterErr := err.(interpreter.Error)

		require.IsType(t,
			interpreter.ConditionError{},
			interpreterErr.Err,
		)
	})
}

func TestInterpretResourceInterfaceInitializerAndDestructorPreConditions(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `

      resource interface RI {

          x: Int

          init(_ x: Int) {
              pre { x > 1: "invalid init" }
          }

          destroy() {
              pre { self.x < 3: "invalid destroy" }
          }
      }

      resource R: RI {

          let x: Int

          init(_ x: Int) {
              self.x = x
          }
      }

      fun test(_ x: Int) {
          let r <- create R(x)
          destroy r
      }
    `)

	t.Run("1", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(1))
		require.Error(t, err)

		require.IsType(t,
			interpreter.Error{},
			err,
		)
		interpreterErr := err.(interpreter.Error)

		require.IsType(t,
			interpreter.ConditionError{},
			interpreterErr.Err,
		)
		conditionError := interpreterErr.Err.(interpreter.ConditionError)

		assert.Equal(t, "invalid init", conditionError.Message)
	})

	t.Run("2", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(2))
		require.NoError(t, err)
	})

	t.Run("3", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(3))
		require.Error(t, err)

		require.IsType(t,
			interpreter.Error{},
			err,
		)
		interpreterErr := err.(interpreter.Error)

		require.IsType(t,
			interpreter.ConditionError{},
			interpreterErr.Err,
		)
		conditionError := interpreterErr.Err.(interpreter.ConditionError)

		assert.Equal(t, "invalid destroy", conditionError.Message)
	})
}

func TestInterpretResourceTypeRequirementInitializerAndDestructorPreConditions(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          pub contract interface CI {

              pub resource R {

                  pub x: Int

                  init(_ x: Int) {
                      pre { x > 1: "invalid init" }
                  }

                  destroy() {
                      pre { self.x < 3: "invalid destroy" }
                  }
              }
          }

          pub contract C: CI {

              pub resource R {

                  pub let x: Int

                  init(_ x: Int) {
                      self.x = x
                  }
              }

              pub fun test(_ x: Int) {
                  let r <- create C.R(x)
                  destroy r
              }
          }

          fun test(_ x: Int) {
              C.test(x)
          }
        `,
		ParseCheckAndInterpretOptions{
			Options: []interpreter.Option{
				makeContractValueHandler(nil, nil, nil),
			},
		},
	)

	t.Run("1", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(1))
		require.Error(t, err)

		require.IsType(t,
			interpreter.Error{},
			err,
		)
		interpreterErr := err.(interpreter.Error)

		require.IsType(t,
			interpreter.ConditionError{},
			interpreterErr.Err,
		)
		conditionError := interpreterErr.Err.(interpreter.ConditionError)

		assert.Equal(t, "invalid init", conditionError.Message)
	})

	t.Run("2", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(2))
		require.NoError(t, err)
	})

	t.Run("3", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewIntValueFromInt64(3))
		require.Error(t, err)

		require.IsType(t,
			interpreter.Error{},
			err,
		)
		interpreterErr := err.(interpreter.Error)

		require.IsType(t,
			interpreter.ConditionError{},
			interpreterErr.Err,
		)
		conditionError := interpreterErr.Err.(interpreter.ConditionError)

		assert.Equal(t, "invalid destroy", conditionError.Message)
	})
}

func TestInterpretFunctionPostConditionInInterface(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct interface SI {
          on: Bool

          fun turnOn() {
              post {
                  self.on
              }
          }
      }

      struct S: SI {
          var on: Bool

          init() {
              self.on = false
          }

          fun turnOn() {
              self.on = true
          }
      }

      struct S2: SI {
          var on: Bool

          init() {
              self.on = false
          }

          fun turnOn() {
              // incorrect
          }
      }

      fun test() {
          S().turnOn()
      }

      fun test2() {
          S2().turnOn()
      }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	_, err = inter.Invoke("test2")
	require.IsType(t,
		interpreter.Error{},
		err,
	)
	interpreterErr := err.(interpreter.Error)

	require.IsType(t,
		interpreter.ConditionError{},
		interpreterErr.Err,
	)
}

func TestInterpretFunctionPostConditionWithBeforeInInterface(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndInterpret(t, `
      struct interface SI {
          on: Bool

          fun toggle() {
              post {
                  self.on != before(self.on)
              }
          }
      }

      struct S: SI {
          var on: Bool

          init() {
              self.on = false
          }

          fun toggle() {
              self.on = !self.on
          }
      }

      struct S2: SI {
          var on: Bool

          init() {
              self.on = false
          }

          fun toggle() {
              // incorrect
          }
      }

      fun test() {
          S().toggle()
      }

      fun test2() {
          S2().toggle()
      }
    `)

	_, err := inter.Invoke("test")
	require.NoError(t, err)

	_, err = inter.Invoke("test2")
	require.IsType(t,
		interpreter.Error{},
		err,
	)
	interpreterErr := err.(interpreter.Error)

	require.IsType(t,
		interpreter.ConditionError{},
		interpreterErr.Err,
	)
}

func TestInterpretIsInstanceCheckInPreCondition(t *testing.T) {

	t.Parallel()

	test := func(condition string) {

		inter := parseCheckAndInterpretWithOptions(t,
			fmt.Sprintf(
				`
                   contract interface CI {
                       struct X {
                            fun use(_ x: X) {
                                pre {
                                    %s
                                }
                            }
                       }
                   }

                   contract C1: CI {
                       struct X {
                           fun use(_ x: CI.X) {}
                       }
                   }

                   contract C2: CI {
                       struct X {
                           fun use(_ x: CI.X) {}
                       }
                   }

                   fun test1() {
                       C1.X().use(C1.X())
                   }

                   fun test2() {
                        C1.X().use(C2.X())
                   }
                `,
				condition,
			),
			ParseCheckAndInterpretOptions{
				Options: []interpreter.Option{
					makeContractValueHandler(nil, nil, nil),
				},
			},
		)

		_, err := inter.Invoke("test1")
		require.NoError(t, err)

		_, err = inter.Invoke("test2")
		require.Error(t, err)
	}

	t.Run("isInstance", func(t *testing.T) {
		test("x.isInstance(self.getType())")
	})

	t.Run("equality", func(t *testing.T) {
		test("x.getType() == self.getType()")
	})
}

func TestInterpretFunctionWithPostConditionAndResourceResult(t *testing.T) {

	t.Parallel()

	checkCalled := false

	// Inject a host function that is used to assert that the `result` value
	// in the post condition is in fact a reference (ephemeral reference value),
	// and not a resource (composite value)

	valueDeclarations := stdlib.StandardLibraryValues{
		{
			Name: "check",
			Type: &sema.FunctionType{
				Parameters: []*sema.Parameter{
					{
						Label:      sema.ArgumentLabelNotRequired,
						Identifier: "value",
						TypeAnnotation: sema.NewTypeAnnotation(
							sema.AnyStructType,
						),
					},
				},
				ReturnTypeAnnotation: sema.NewTypeAnnotation(
					sema.VoidType,
				),
			},
			Value: interpreter.NewHostFunctionValue(
				func(invocation interpreter.Invocation) interpreter.Value {
					checkCalled = true

					argument := invocation.Arguments[0]
					require.IsType(t, &interpreter.EphemeralReferenceValue{}, argument)

					return interpreter.VoidValue{}
				},
			),
			Kind: common.DeclarationKindConstant,
		},
	}

	semaValueDeclarations := valueDeclarations.ToSemaValueDeclarations()
	interpreterValueDeclarations := valueDeclarations.ToInterpreterValueDeclarations()

	inter := parseCheckAndInterpretWithOptions(t,
		`
          resource R {}

          resource Container {

              let resources: @{String: R}

              init() {
                  self.resources <- {"original": <-create R()}
              }

              fun withdraw(): @R {
                  post {
                      self.use(result)
                  }
                  return <- self.resources.remove(key: "original")!
              }

              fun use(_ r: &R): Bool {
                  check(r)
                  return true
              }

              destroy() {
                  destroy self.resources
              }
          }

          fun test(): Bool {
              let container <- create Container()

              let r <- container.withdraw()
              // show that while r is the resource,
              // it also still exists in the container
              let duplicated = container.resources["duplicate"] != nil

              // clean-up
              destroy r
              destroy container

              return duplicated
          }
        `,
		ParseCheckAndInterpretOptions{
			CheckerOptions: []sema.Option{
				sema.WithPredeclaredValues(semaValueDeclarations),
			},
			Options: []interpreter.Option{
				interpreter.WithPredeclaredValues(interpreterValueDeclarations),
			},
		})

	_, err := inter.Invoke("test")
	require.NoError(t, err)
	require.True(t, checkCalled)
}
