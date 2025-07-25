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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/activations"
	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/bbq/vm"
	compilerUtils "github.com/onflow/cadence/bbq/vm/test"
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	"github.com/onflow/cadence/test_utils"
	. "github.com/onflow/cadence/test_utils/common_utils"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretFunctionPreTestCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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

	assertConditionError(
		t,
		err,
		ast.ConditionKindPre,
	)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, zero, value)
}

func TestInterpretFunctionPreEmitCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndPrepareWithEvents(t,
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

	assert.Equal(t,
		[]interpreter.Value{
			answer,
		},
		event.EventFields,
	)
}

func TestInterpretFunctionPostTestCondition(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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

	assertConditionError(
		t,
		err,
		ast.ConditionKindPost,
	)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, zero, value)
}

func TestInterpretFunctionPostEmitCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndPrepareWithEvents(t,
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

	assert.Equal(t,
		[]interpreter.Value{
			answer,
		},
		event.EventFields,
	)
}

func TestInterpretFunctionWithResultAndPostTestConditionWithResult(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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

	assertConditionError(
		t,
		err,
		ast.ConditionKindPost,
	)

	zero := interpreter.NewUnmeteredIntValueFromInt64(0)
	value, err := inter.Invoke("test", zero)
	require.NoError(t, err)

	AssertValuesEqual(t, inter, zero, value)
}

func assertConditionError(
	t *testing.T,
	err error,
	conditionKind ast.ConditionKind,
) {
	RequireError(t, err)

	var conditionErr *interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(t,
		conditionKind,
		conditionErr.ConditionKind,
	)
}

func assertConditionErrorWithMessage(
	t *testing.T,
	err error,
	conditionKind ast.ConditionKind,
	message string,
) {
	RequireError(t, err)

	var conditionErr *interpreter.ConditionError
	require.ErrorAs(t, err, &conditionErr)

	assert.Equal(
		t,
		conditionKind,
		conditionErr.ConditionKind,
	)

	assert.Equal(
		t,
		message,
		conditionErr.Message,
	)
}

func TestInterpretFunctionWithResultAndPostEmitConditionWithResult(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndPrepareWithEvents(t, `
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

	assert.Equal(t,
		[]interpreter.Value{
			answer,
		},
		event.EventFields,
	)
}

func TestInterpretFunctionWithoutResultAndPostTestConditionWithResult(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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

	inter, getEvents, err := parseCheckAndPrepareWithEvents(t, `
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
	assert.Equal(t,
		[]interpreter.Value{
			answer,
		},
		event.EventFields,
	)
}

func TestInterpretFunctionPostTestConditionWithBefore(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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

	inter, getEvents, err := parseCheckAndPrepareWithEvents(t, `
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

	assert.Equal(t,
		[]interpreter.Value{
			interpreter.NewUnmeteredIntValueFromInt64(1),
			interpreter.NewUnmeteredIntValueFromInt64(0),
		},
		event.EventFields,
	)
}

func TestInterpretFunctionPostConditionWithBeforeFailingPreTestCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndPrepareWithEvents(t, `
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

	assertConditionError(
		t,
		err,
		ast.ConditionKindPre,
	)

	events := getEvents()
	require.Len(t, events, 0)
}

func TestInterpretFunctionPostConditionWithBeforeFailingPostTestCondition(t *testing.T) {

	t.Parallel()

	inter, getEvents, err := parseCheckAndPrepareWithEvents(t, `
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

	assertConditionError(
		t,
		err,
		ast.ConditionKindPost,
	)

	events := getEvents()
	require.Len(t, events, 0)
}

func TestInterpretFunctionPostConditionWithMessageUsingStringLiteral(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
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

	assertConditionErrorWithMessage(
		t,
		err,
		ast.ConditionKindPost,
		"y should be zero",
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

	inter := parseCheckAndPrepare(t, `
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

	assertConditionErrorWithMessage(
		t,
		err,
		ast.ConditionKindPost,
		"return value",
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

	inter := parseCheckAndPrepare(t, `
      fun test(x: String): String {
          post {
              1 == 2: before(x)
          }
          return "return value"
      }
    `)

	_, err := inter.Invoke("test", interpreter.NewUnmeteredStringValue("parameter value"))

	assertConditionErrorWithMessage(
		t,
		err,
		ast.ConditionKindPost,
		"parameter value",
	)
}

func TestInterpretFunctionPostConditionWithMessageUsingParameter(t *testing.T) {

	t.Parallel()

	inter := parseCheckAndPrepare(t, `
      fun test(x: String): String {
          post {
              1 == 2: x
          }
          return "return value"
      }
    `)

	_, err := inter.Invoke("test", interpreter.NewUnmeteredStringValue("parameter value"))

	assertConditionErrorWithMessage(
		t,
		err,
		ast.ConditionKindPost,
		"parameter value",
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
			interfaceType := "{Test}"

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

			inter, err := parseCheckAndPrepareWithOptions(t,
				fmt.Sprintf(
					`
                      event InterX(x: Int)
                      event ImplX(x: Int)

                      %[1]s interface Test {
                          fun test(x: Int): Int {
                              pre {
                                  x > 0: "x must be positive"
                                  emit InterX(x: x)
                              }
                          }
                      }

                      %[1]s TestImpl: Test {
                          fun test(x: Int): Int {
                              pre {
                                  x < 2: "x must be smaller than 2"
                                  emit ImplX(x: x)
                              }
                              return x
                          }
                      }

                      fun callTest(x: Int): Int {
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
					InterpreterConfig: &interpreter.Config{
						ContractValueHandler: makeContractValueHandler(nil, nil, nil),
						OnEventEmitted: func(
							_ interpreter.ValueExportContext,
							_ interpreter.LocationRange,
							eventType *sema.CompositeType,
							eventFields []interpreter.Value,
						) error {
							events = append(events, testEvent{
								EventType:   eventType,
								EventFields: eventFields,
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

				assertConditionErrorWithMessage(
					t,
					err,
					ast.ConditionKindPre,
					"x must be positive",
				)

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

				assert.Equal(t,
					TestLocation.TypeID(nil, "InterX"),
					events[0].EventType.ID(),
				)

				assert.Equal(t,
					[]interpreter.Value{
						value,
					},
					events[0].EventFields,
				)

				assert.Equal(t,
					TestLocation.TypeID(nil, "ImplX"),
					events[1].EventType.ID(),
				)

				assert.Equal(t,
					[]interpreter.Value{
						value,
					},
					events[1].EventFields,
				)
			})

			t.Run("callTest(2)", func(t *testing.T) {

				events = nil

				value := interpreter.NewUnmeteredIntValueFromInt64(2)

				_, err = inter.Invoke("callTest", value)

				assertConditionErrorWithMessage(
					t,
					err,
					ast.ConditionKindPre,
					"x must be smaller than 2",
				)

				require.Len(t, events, 1)

				assert.Equal(t,
					TestLocation.TypeID(nil, "InterX"),
					events[0].EventType.ID(),
				)

				assert.Equal(t,
					[]interpreter.Value{
						value,
					},
					events[0].EventFields,
				)
			})
		})
	}
}

func TestInterpretInitializerWithInterfacePreCondition(t *testing.T) {

	t.Parallel()

	tests := map[int64]struct {
		err    error
		events int
	}{
		0: {&interpreter.ConditionError{}, 0},
		1: {nil, 2},
		2: {&interpreter.ConditionError{}, 1},
	}

	for _, compositeKind := range common.CompositeKindsWithFieldsAndFunctions {

		if !compositeKind.SupportsInterfaces() {
			continue
		}

		t.Run(compositeKind.Keyword(), func(t *testing.T) {

			for value, expectedResult := range tests {

				t.Run(fmt.Sprint(value), func(t *testing.T) {

					var testFunction string
					if compositeKind == common.CompositeKindContract {
						// use the contract singleton, so it is loaded
						testFunction = `
					       fun test() {
					            TestImpl.NoOpFunc()
					       }
                        `
					} else {
						interfaceType := "{Test}"

						testFunction =
							fmt.Sprintf(
								`
					               fun test(x: Int): %[1]s%[2]s {
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

					code := fmt.Sprintf(
						`
                                 access(all)
                                 event Foo(x: Int)

					             %[1]s interface Test {
					                 init(x: Int) {
					                     pre {
					                         x > 0: "x must be positive"
                                             emit Foo(x: x)
					                     }
					                 }
					             }

					             %[1]s TestImpl: Test {
					                 init(x: Int) {
					                     pre {
					                         x < 2: "x must be smaller than 2"
                                             emit Foo(x: x)
					                     }
					                 }

					                 fun NoOpFunc() {}
					             }

					             %[2]s
					           `,
						compositeKind.Keyword(),
						testFunction,
					)

					var events []testEvent
					onEmitEvents := func(
						_ interpreter.ValueExportContext,
						_ interpreter.LocationRange,
						eventType *sema.CompositeType,
						eventFields []interpreter.Value,
					) error {
						events = append(events, testEvent{
							EventType:   eventType,
							EventFields: eventFields,
						})
						return nil
					}

					var invokable Invokable
					var err error

					if *compile {
						vmConfig := vm.NewConfig(NewUnmeteredInMemoryStorage())
						vmConfig.ContractValueHandler = compilerUtils.ContractValueHandler(
							"TestImpl",
							interpreter.NewUnmeteredIntValueFromInt64(value),
						)
						vmConfig.OnEventEmitted = onEmitEvents

						var vmInstance *vm.VM
						vmInstance, err = compilerUtils.CompileAndPrepareToInvoke(
							t,
							code,
							compilerUtils.CompilerAndVMOptions{
								VMConfig: vmConfig,
							},
						)

						invokable = test_utils.NewVMInvokable(vmInstance, nil)
					} else {
						invokable, err = parseCheckAndInterpretWithOptions(t,
							code,
							ParseCheckAndInterpretOptions{
								InterpreterConfig: &interpreter.Config{
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
									OnEventEmitted: onEmitEvents,
									UUIDHandler: func() (uint64, error) {
										return 0, nil
									},
								},
							},
						)
					}

					if compositeKind == common.CompositeKindContract {
						if *compile {
							require.NoError(t, err)
							_, err = invokable.Invoke("test")
						}

						// parseCheckAndInterpretWithOptions already loads the contract value.
						// So no need to explicitly call the `test` method.
					} else {
						require.NoError(t, err)
						_, err = invokable.Invoke(
							"test",
							interpreter.NewUnmeteredIntValueFromInt64(value),
						)
					}

					if expectedResult.err == nil {
						require.NoError(t, err)
					} else {
						RequireError(t, err)

						expectedError := expectedResult.err
						require.ErrorAs(t, err, &expectedError)
					}

					require.Len(t, events, expectedResult.events)
				})
			}
		})
	}
}

func TestInterpretResourceInterfaceInitializerPreConditions(t *testing.T) {

	t.Parallel()

	newInterpreter := func(t *testing.T) (invokable Invokable, getEvents func() []testEvent) {
		var err error
		invokable, getEvents, err = parseCheckAndPrepareWithEvents(t, `

          event InitPre(x: Int)
          event DestroyPre(x: Int)

          resource interface RI {

              x: Int

              init(_ x: Int) {
                  pre {
                      x > 1: "invalid init"
                      emit InitPre(x: x)
                  }
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
		require.NoError(t, err)
		return
	}

	t.Run("1", func(t *testing.T) {
		t.Parallel()

		inter, getEvents := newInterpreter(t)
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(1))

		assertConditionErrorWithMessage(
			t,
			err,
			ast.ConditionKindPre,
			"invalid init",
		)

		require.Len(t, getEvents(), 0)
	})

	t.Run("2", func(t *testing.T) {
		t.Parallel()

		inter, getEvents := newInterpreter(t)
		_, err := inter.Invoke("test", interpreter.NewUnmeteredIntValueFromInt64(2))
		require.NoError(t, err)

		require.Len(t, getEvents(), 1)
	})
}

func TestInterpretFunctionPostConditionInInterface(t *testing.T) {

	t.Parallel()

	newInterpreter := func(t *testing.T) (inter Invokable, getEvents func() []testEvent) {
		var err error
		inter, getEvents, err = parseCheckAndPrepareWithEvents(t, `

          event Status(on: Bool)

          struct interface SI {
              on: Bool

              fun turnOn() {
                  post {
                      self.on
                      emit Status(on: self.on)
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
		require.NoError(t, err)
		return
	}

	t.Run("test", func(t *testing.T) {
		t.Parallel()

		inter, getEvents := newInterpreter(t)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, getEvents(), 1)
	})

	t.Run("test2", func(t *testing.T) {
		t.Parallel()

		inter, getEvents := newInterpreter(t)

		_, err := inter.Invoke("test2")

		assertConditionError(
			t,
			err,
			ast.ConditionKindPost,
		)

		require.Len(t, getEvents(), 0)
	})
}

func TestInterpretFunctionPostConditionWithBeforeInInterface(t *testing.T) {

	t.Parallel()

	newInterpreter := func(t *testing.T) (inter Invokable, getEvents func() []testEvent) {
		var err error
		inter, getEvents, err = parseCheckAndPrepareWithEvents(t, `

          event Status(on: Bool)

          struct interface SI {
              on: Bool

              fun toggle() {
                  post {
                      self.on != before(self.on)
                      emit Status(on: self.on)
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

		require.NoError(t, err)
		return
	}

	t.Run("test", func(t *testing.T) {
		t.Parallel()

		inter, getEvents := newInterpreter(t)

		_, err := inter.Invoke("test")
		require.NoError(t, err)

		require.Len(t, getEvents(), 1)
	})

	t.Run("test2", func(t *testing.T) {
		t.Parallel()

		inter, getEvents := newInterpreter(t)

		_, err := inter.Invoke("test2")

		assertConditionError(
			t,
			err,
			ast.ConditionKindPost,
		)

		require.Len(t, getEvents(), 0)
	})
}

func TestInterpretIsInstanceCheckInPreCondition(t *testing.T) {

	t.Parallel()

	test := func(condition string) {

		inter, err := parseCheckAndPrepareWithOptions(t,
			fmt.Sprintf(
				`
                   contract interface CI {
                       struct interface X {
                            fun use(_ x: {X}) {
                                pre {
                                    %s
                                }
                            }
                       }
                   }

                   contract C1: CI {
                       struct X: CI.X {
                           fun use(_ x: {CI.X}) {}
                       }
                   }

                   contract C2: CI {
                       struct X: CI.X {
                           fun use(_ x: {CI.X}) {}
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
				InterpreterConfig: &interpreter.Config{
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
		Value: interpreter.NewStaticHostFunctionValue(
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

	inter, err := parseCheckAndPrepareWithOptions(t,
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
			ParseAndCheckOptions: &ParseAndCheckOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
						return baseValueActivation
					},
				},
			},
			InterpreterConfig: &interpreter.Config{
				BaseActivationHandler: func(_ common.Location) *interpreter.VariableActivation {
					return baseActivation
				},
			},
		},
	)
	require.NoError(t, err)

	_, err = inter.Invoke("test")
	require.NoError(t, err)
	require.True(t, checkCalled)
}

func TestInterpretInnerFunctionPreConditions(t *testing.T) {

	t.Parallel()

	t.Run("fail", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  fun foo(_ y: Int): Int {
                      pre {
                          y == 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		_, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		assertConditionError(t, err, ast.ConditionKindPre)
	})

	t.Run("in nested local function, fail", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  if true {
                      if true {
                          fun foo(_ y: Int): Int {
                              pre {
                                  y == 0
                              }
                              return y
                          }
                          return foo(x)
                      }
                  }

                  return 0
              }
            `,
		)

		_, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		assertConditionError(t, err, ast.ConditionKindPre)
	})

	t.Run("in local function, pass", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  fun foo(_ y: Int): Int {
                      pre {
                          y != 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		result, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			result,
		)
	})
}

func TestInterpretInnerFunctionPostConditions(t *testing.T) {

	t.Parallel()

	t.Run("fail", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  fun foo(_ y: Int): Int {
                      post {
                          y == 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		_, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		assertConditionError(t, err, ast.ConditionKindPost)
	})

	t.Run("in nested local function, fail", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  if true {
                      if true {
                          fun foo(_ y: Int): Int {
                              post {
                                  y == 0
                              }
                              return y
                          }
                          return foo(x)
                      }
                  }

                  return 0
              }
            `,
		)

		_, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		assertConditionError(t, err, ast.ConditionKindPost)
	})

	t.Run("in local function, pass", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  fun foo(_ y: Int): Int {
                      post {
                          y != 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		result, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			result,
		)
	})
}

func TestInterpretFunctionExpressionPreConditions(t *testing.T) {

	t.Parallel()

	t.Run("fail", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  var foo = fun(_ y: Int): Int {
                      pre {
                          y == 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		_, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		assertConditionError(t, err, ast.ConditionKindPre)
	})

	t.Run("pass", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  var foo = fun(_ y: Int): Int {
                      pre {
                          y != 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		result, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			result,
		)
	})
}

func TestInterpretFunctionExpressionPostConditions(t *testing.T) {

	t.Parallel()

	t.Run("fail", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  var foo = fun(_ y: Int): Int {
                      post {
                          y == 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		_, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		assertConditionError(t, err, ast.ConditionKindPost)
	})

	t.Run("pass", func(t *testing.T) {

		t.Parallel()

		invokable := parseCheckAndPrepare(t,
			`
              fun main(x: Int): Int {

                  var foo = fun(_ y: Int): Int {
                      post {
                          y != 0
                      }
                      return y
                  }

                  return foo(x)
              }
            `,
		)

		result, err := invokable.Invoke("main", interpreter.NewUnmeteredIntValueFromInt64(3))
		require.NoError(t, err)
		assert.Equal(
			t,
			interpreter.NewUnmeteredIntValueFromInt64(3),
			result,
		)
	})
}
