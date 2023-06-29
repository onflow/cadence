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

package interpreter_test

import (
	"fmt"
	"testing"

	"github.com/onflow/cadence/runtime/activations"

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

func TestInterpretFunctionPreTestCondition(t *testing.T) {

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
		interpreter.NewUnmeteredIntValueFromInt64(42),
	)
	RequireError(t, err)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, zero, value)
}

func TestInterpretFunctionPreEmitCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndInterpretWithEvents(t,
		`
          event Foo(x: Int)

          fun test(x: Int): Int {
              pre {
                  emit Foo(x: x)
              }
              return x
          }
        `,
	)
	require.NoError(t, err)

	answer := interpreter.NewUnmeteredIntValueFromInt64(42)
	result, err := inter.Invoke("test", answer)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, answer, result)

	events := getEvents()
	require.Len(t, events, 1)
	event := events[0]

	expectedEvent := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		inter.Location,
		"Foo",
		common.CompositeKindEvent,
		[]interpreter.CompositeField{
			{
				Name:  "x",
				Value: answer,
			},
		},
		common.ZeroAddress,
	)
	AssertValuesEqual(t, inter, expectedEvent, event.event)
}

func TestInterpretFunctionPostTestCondition(t *testing.T) {

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
		interpreter.NewUnmeteredIntValueFromInt64(42),
	)
	RequireError(t, err)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, zero, value)
}

func TestInterpretFunctionPostEmitCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndInterpretWithEvents(t,
		`
          event Foo(y: Int)

          fun test(x: Int): Int {
              post {
                  emit Foo(y: y)
              }
              let y = x
              return y
          }
        `,
	)
	require.NoError(t, err)

	answer := interpreter.NewUnmeteredIntValueFromInt64(42)
	result, err := inter.Invoke("test", answer)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, answer, result)

	events := getEvents()
	require.Len(t, events, 1)
	event := events[0]

	expectedEvent := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		inter.Location,
		"Foo",
		common.CompositeKindEvent,
		[]interpreter.CompositeField{
			{
				Name:  "y",
				Value: answer,
			},
		},
		common.ZeroAddress,
	)
	AssertValuesEqual(t, inter, expectedEvent, event.event)
}

func TestInterpretFunctionWithResultAndPostTestConditionWithResult(t *testing.T) {

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
		interpreter.NewUnmeteredIntValueFromInt64(42),
	)
	RequireError(t, err)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, zero, value)
}

func TestInterpretFunctionWithResultAndPostEmitConditionWithResult(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndInterpretWithEvents(t, `
          event Foo(x: Int)

          fun test(x: Int): Int {
              post {
                  emit Foo(x: result)
              }
              return x
          }
        `)
	require.NoError(t, err)

	answer := interpreter.NewUnmeteredIntValueFromInt64(42)
	result, err := inter.Invoke("test", answer)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, answer, result)

	events := getEvents()
	require.Len(t, events, 1)
	event := events[0]

	expectedEvent := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		inter.Location,
		"Foo",
		common.CompositeKindEvent,
		[]interpreter.CompositeField{
			{
				Name:  "x",
				Value: answer,
			},
		},
		common.ZeroAddress,
	)
	AssertValuesEqual(t, inter, expectedEvent, event.event)
}

func TestInterpretFunctionWithoutResultAndPostTestConditionWithResult(t *testing.T) {

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

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretFunctionWithoutResultAndPostEmitConditionWithResult(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndInterpretWithEvents(t, `
      event Foo(x: Int)

      fun test() {
          post {
              emit Foo(x: result)
          }
          let result = 42
      }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	events := getEvents()
	require.Len(t, events, 1)
	event := events[0]

	answer := interpreter.NewUnmeteredIntValueFromInt64(42)
	expectedEvent := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		inter.Location,
		"Foo",
		common.CompositeKindEvent,
		[]interpreter.CompositeField{
			{
				Name:  "x",
				Value: answer,
			},
		},
		common.ZeroAddress,
	)
	AssertValuesEqual(t, inter, expectedEvent, event.event)
}

func TestInterpretFunctionPostTestConditionWithBefore(t *testing.T) {

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

	AssertValuesEqual(
		t,
		inter,
		interpreter.Void,
		value,
	)
}

func TestInterpretFunctionPostEmitConditionWithBefore(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndInterpretWithEvents(t, `
      event Foo(x: Int, beforeX: Int)

      var x = 0

      fun test() {
          pre {
              x == 0
          }
          post {
              emit Foo(x: x, beforeX: before(x))
          }
          x = x + 1
      }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)

	events := getEvents()
	require.Len(t, events, 1)
	event := events[0]

	expectedEvent := interpreter.NewCompositeValue(
		inter,
		interpreter.EmptyLocationRange,
		inter.Location,
		"Foo",
		common.CompositeKindEvent,
		[]interpreter.CompositeField{
			{
				Name:  "x",
				Value: interpreter.NewUnmeteredIntValueFromInt64(1),
			},
			{
				Name:  "beforeX",
				Value: interpreter.NewUnmeteredIntValueFromInt64(0),
			},
		},
		common.ZeroAddress,
	)
	AssertValuesEqual(t, inter, expectedEvent, event.event)
}

func TestInterpretFunctionPostConditionWithBeforeFailingPreTestCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndInterpretWithEvents(t, `
      event Foo(x: Int)

      var x = 0

      fun test() {
          pre {
              x == 1
              emit Foo(x: x)
          }
          post {
              x == before(x) + 1
          }
          x = x + 1
      }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	RequireError(t, err)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		ast.ConditionKindPre,
		conditionErr.ConditionKind,
	)

	events := getEvents()
	require.Len(t, events, 0)
}

func TestInterpretFunctionPostConditionWithBeforeFailingPostTestCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndInterpretWithEvents(t, `
      event Foo(x: Int)

      var x = 0

      fun test() {
          pre {
              x == 0
          }
          post {
              x == before(x) + 2
              emit Foo(x: x)
          }
          x = x + 1
      }
    `)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	RequireError(t, err)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		ast.ConditionKindPost,
		conditionErr.ConditionKind,
	)

	events := getEvents()
	require.Len(t, events, 0)
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
		interpreter.NewUnmeteredIntValueFromInt64(42),
	)
	RequireError(t, err)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		"y should be zero",
		conditionErr.Message,
	)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		zero,
		value,
	)
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
		interpreter.NewUnmeteredIntValueFromInt64(42),
	)
	RequireError(t, err)

	var conditionErr interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		"return value",
		conditionErr.Message,
	)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(
		t,
		inter,
		interpreter.NewUnmeteredStringValue("return value"),
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

	_, err := inter.Invoke("test", interpreter.NewUnmeteredStringValue("parameter value"))
	RequireError(t, err)

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

	_, err := inter.Invoke("test", interpreter.NewUnmeteredStringValue("parameter value"))
	RequireError(t, err)

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

			var events []testEvent

			inter, err := parseCheckAndInterpretWithOptions(t,
				fmt.Sprintf(
					`
                      event InterX(x: Int)
                      event ImplX(x: Int)

                      access(all) %[1]s interface Test {
                          access(all) fun test(x: Int): Int {
                              pre {
                                  x > 0: "x must be positive"
                                  emit InterX(x: x)
                              }
                          }
                      }

                      access(all) %[1]s TestImpl: Test {
                          access(all) fun test(x: Int): Int {
                              pre {
                                  x < 2: "x must be smaller than 2"
                                  emit ImplX(x: x)
                              }
                              return x
                          }
                      }

                      access(all) fun callTest(x: Int): Int {
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
					Config: &interpreter.Config{
						ContractValueHandler: makeContractValueHandler(nil, nil, nil),
						OnEventEmitted: func(
							_ *interpreter.Interpreter,
							_ interpreter.LocationRange,
							event *interpreter.CompositeValue,
							eventType *sema.CompositeType,
						) error {
							events = append(events, testEvent{
								event:     event,
								eventType: eventType,
							})
							return nil
						},
					},
				},
			)
			require.NoError(t, err)

			t.Run("callTest(0)", func(t *testing.T) {

				events = nil

				_, err = inter.Invoke("callTest", interpreter.NewUnmeteredIntValueFromInt64(0))
				RequireError(t, err)

				var conditionErr interpreter.ConditionError
				require.ErrorAs(t, err, &conditionErr)

				require.Len(t, events, 0)
			})

			t.Run("callTest(1)", func(t *testing.T) {

				events = nil

				value := interpreter.NewUnmeteredIntValueFromInt64(1)

				result, err := inter.Invoke("callTest", value)
				require.NoError(t, err)

				AssertValuesEqual(
					t,
					inter,
					value,
					result,
				)

				require.Len(t, events, 2)

				AssertValuesEqual(t,
					inter,
					interpreter.NewCompositeValue(
						inter,
						interpreter.EmptyLocationRange,
						inter.Location,
						"InterX",
						common.CompositeKindEvent,
						[]interpreter.CompositeField{
							{
								Name:  "x",
								Value: value,
							},
						},
						common.ZeroAddress,
					),
					events[0].event,
				)

				AssertValuesEqual(t,
					inter,
					interpreter.NewCompositeValue(
						inter,
						interpreter.EmptyLocationRange,
						inter.Location,
						"ImplX",
						common.CompositeKindEvent,
						[]interpreter.CompositeField{
							{
								Name:  "x",
								Value: value,
							},
						},
						common.ZeroAddress,
					),
					events[1].event,
				)
			})

			t.Run("callTest(2)", func(t *testing.T) {

				events = nil

				value := interpreter.NewUnmeteredIntValueFromInt64(2)

				_, err = inter.Invoke("callTest", value)
				RequireError(t, err)

				var conditionErr interpreter.ConditionError
				require.ErrorAs(t, err, &conditionErr)

				require.Len(t, events, 1)

				AssertValuesEqual(t,
					inter,
					interpreter.NewCompositeValue(
						inter,
						interpreter.EmptyLocationRange,
						inter.Location,
						"InterX",
						common.CompositeKindEvent,
						[]interpreter.CompositeField{
							{
								Name:  "x",
								Value: value,
							},
						},
						common.ZeroAddress,
					),
					events[0].event,
				)
			})
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
					       access(all) fun test() {
					            TestImpl
					       }
                        `
					} else {
						interfaceType := AsInterfaceType("Test", compositeKind)

						testFunction =
							fmt.Sprintf(
								`
					               access(all) fun test(x: Int): %[1]s%[2]s {
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
					             access(all) %[1]s interface Test {
					                 init(x: Int) {
					                     pre {
					                         x > 0: "x must be positive"
					                     }
					                 }
					             }

					             access(all) %[1]s TestImpl: Test {
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

					uuidHandler := func() (uint64, error) {
						return 0, nil
					}

					if compositeKind == common.CompositeKindContract {

						storage := newUnmeteredInMemoryStorage()

						inter, err := interpreter.NewInterpreter(
							interpreter.ProgramFromChecker(checker),
							checker.Location,
							&interpreter.Config{
								Storage: storage,
								ContractValueHandler: makeContractValueHandler(
									[]interpreter.Value{
										interpreter.NewUnmeteredIntValueFromInt64(value),
									},
									[]sema.Type{
										sema.IntType,
									},
									[]sema.Type{
										sema.IntType,
									},
								),
								UUIDHandler: uuidHandler,
							},
						)
						require.NoError(t, err)

						err = inter.Interpret()
						require.NoError(t, err)

						_, err = inter.Invoke("test")
						check(err)
					} else {
						storage := newUnmeteredInMemoryStorage()

						inter, err := interpreter.NewInterpreter(
							interpreter.ProgramFromChecker(checker),
							checker.Location,
							&interpreter.Config{
								Storage:     storage,
								UUIDHandler: uuidHandler,
							},
						)
						require.NoError(t, err)

						err = inter.Interpret()
						require.NoError(t, err)

						_, err = inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(value))
						check(err)
					}
				})
			}
		})
	}
}

func TestInterpretTypeRequirementWithPreCondition(t *testing.T) {

	t.Parallel()

	inter, err := parseCheckAndInterpretWithOptions(t,
		`

          access(all) struct interface Also {
             access(all) fun test(x: Int) {
                 pre {
                     x >= 0: "x >= 0"
                 }
             }
          }

          access(all) contract interface Test {

              access(all) struct Nested {
                  access(all) fun test(x: Int) {
                      pre {
                          x >= 1: "x >= 1"
                      }
                  }
              }
          }

          access(all) contract TestImpl: Test {

              access(all) struct Nested: Also {
                  access(all) fun test(x: Int) {
                      pre {
                          x < 2: "x < 2"
                      }
                  }
              }
          }

          access(all) fun test(x: Int) {
              TestImpl.Nested().test(x: x)
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	t.Run("-1", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(-1))
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		// NOTE: The type requirement condition (`Test.Nested`) is evaluated first,
		//  before the type's conformances (`Also`)

		assert.Equal(t, "x >= 1", conditionErr.Message)
	})

	t.Run("0", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(0))
		RequireError(t, err)

		var conditionErr interpreter.ConditionError
		require.ErrorAs(t, err, &conditionErr)

		assert.Equal(t, "x >= 1", conditionErr.Message)
	})

	t.Run("1", func(t *testing.T) {
		value, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(1))
		require.NoError(t, err)

		assert.IsType(t,
			interpreter.Void,
			value,
		)
	})

	t.Run("2", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(2))
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
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(1))
		RequireError(t, err)

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
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(2))
		require.NoError(t, err)
	})

	t.Run("3", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(3))
		RequireError(t, err)

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

	inter, err := parseCheckAndInterpretWithOptions(t,
		`
          access(all) contract interface CI {

              access(all) resource R {

                  access(all) x: Int

                  init(_ x: Int) {
                      pre { x > 1: "invalid init" }
                  }

                  destroy() {
                      pre { self.x < 3: "invalid destroy" }
                  }
              }
          }

          access(all) contract C: CI {

              access(all) resource R {

                  access(all) let x: Int

                  init(_ x: Int) {
                      self.x = x
                  }
              }

              access(all) fun test(_ x: Int) {
                  let r <- create C.R(x)
                  destroy r
              }
          }

          fun test(_ x: Int) {
              C.test(x)
          }
        `,
		ParseCheckAndInterpretOptions{
			Config: &interpreter.Config{
				ContractValueHandler: makeContractValueHandler(nil, nil, nil),
			},
		},
	)
	require.NoError(t, err)

	t.Run("1", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(1))
		RequireError(t, err)

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
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(2))
		require.NoError(t, err)
	})

	t.Run("3", func(t *testing.T) {
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(3))
		RequireError(t, err)

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

		inter, err := parseCheckAndInterpretWithOptions(t,
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
				Config: &interpreter.Config{
					ContractValueHandler: makeContractValueHandler(nil, nil, nil),
				},
			},
		)
		require.NoError(t, err)

		_, err = inter.Invoke("test1")
		require.NoError(t, err)

		_, err = inter.Invoke("test2")
		RequireError(t, err)
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

	checkFunctionType := &sema.FunctionType{
		Purity: sema.FunctionPurityView,
		Parameters: []sema.Parameter{
			{
				Label:          sema.ArgumentLabelNotRequired,
				Identifier:     "value",
				TypeAnnotation: sema.AnyStructTypeAnnotation,
			},
		},
		ReturnTypeAnnotation: sema.VoidTypeAnnotation,
	}

	valueDeclaration := stdlib.StandardLibraryValue{
		Name: "check",
		Type: checkFunctionType,
		Value: interpreter.NewHostFunctionValue(
			nil,
			checkFunctionType,
			func(invocation interpreter.Invocation) interpreter.Value {
				checkCalled = true

				argument := invocation.Arguments[0]
				require.IsType(t, &interpreter.EphemeralReferenceValue{}, argument)

				return interpreter.Void
			},
		),
		Kind: common.DeclarationKindConstant,
	}

	baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
	baseValueActivation.DeclareValue(valueDeclaration)

	baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
	interpreter.Declare(baseActivation, valueDeclaration)

	inter, err := parseCheckAndInterpretWithOptions(t,
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

              view fun use(_ r: &R): Bool {
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
			CheckerConfig: &sema.Config{
				BaseValueActivation: baseValueActivation,
			},
			Config: &interpreter.Config{
				BaseActivation: baseActivation,
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)
	require.True(t, checkCalled)
}
