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
	"github.com/onflow/cadence/runtime/errors"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
	. "github.com/onflow/cadence/runtime/tests/utils"
)

// dynamic casting operation -> returns optional
var dynamicCastingOperations = map[ast.Operation]bool{
	ast.OperationFailableCast: true,
	ast.OperationForceCast:    false,
}

func TestInterpretDynamicCastingNumber(t *testing.T) {

	t.Parallel()

	type test struct {
		ty       sema.Type
		value    string
		expected interpreter.Value
	}

	tests := []test{
		{sema.IntType, "42", interpreter.NewUnmeteredIntValueFromInt64(42)},
		{sema.UIntType, "42", interpreter.NewUnmeteredUIntValueFromUint64(42)},
		{sema.Int8Type, "42", interpreter.NewUnmeteredInt8Value(42)},
		{sema.Int16Type, "42", interpreter.NewUnmeteredInt16Value(42)},
		{sema.Int32Type, "42", interpreter.NewUnmeteredInt32Value(42)},
		{sema.Int64Type, "42", interpreter.NewUnmeteredInt64Value(42)},
		{sema.Int128Type, "42", interpreter.NewUnmeteredInt128ValueFromInt64(42)},
		{sema.Int256Type, "42", interpreter.NewUnmeteredInt256ValueFromInt64(42)},
		{sema.UInt8Type, "42", interpreter.NewUnmeteredUInt8Value(42)},
		{sema.UInt16Type, "42", interpreter.NewUnmeteredUInt16Value(42)},
		{sema.UInt32Type, "42", interpreter.NewUnmeteredUInt32Value(42)},
		{sema.UInt64Type, "42", interpreter.NewUnmeteredUInt64Value(42)},
		{sema.UInt128Type, "42", interpreter.NewUnmeteredUInt128ValueFromUint64(42)},
		{sema.UInt256Type, "42", interpreter.NewUnmeteredUInt256ValueFromUint64(42)},
		{sema.Word8Type, "42", interpreter.NewUnmeteredWord8Value(42)},
		{sema.Word16Type, "42", interpreter.NewUnmeteredWord16Value(42)},
		{sema.Word32Type, "42", interpreter.NewUnmeteredWord32Value(42)},
		{sema.Word64Type, "42", interpreter.NewUnmeteredWord64Value(42)},
		{sema.Word128Type, "42", interpreter.NewUnmeteredWord128ValueFromUint64(42)},
		{sema.Fix64Type, "1.23", interpreter.NewUnmeteredFix64Value(123000000)},
		{sema.UFix64Type, "1.23", interpreter.NewUnmeteredUFix64Value(123000000)},
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, test := range tests {

				t.Run(test.ty.String(), func(t *testing.T) {

					types := []sema.Type{
						sema.AnyStructType,
						test.ty,
					}
					for _, fromType := range types {
						for _, targetType := range types {

							t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

								inter := parseCheckAndInterpret(t,
									fmt.Sprintf(
										`
                                          let x: %[1]s = %[2]s
                                          let y: %[3]s = x
                                          let z: %[4]s? = y %[5]s %[4]s
                                        `,
										test.ty,
										test.value,
										fromType,
										targetType,
										operation.Symbol(),
									),
								)

								AssertValuesEqual(
									t,
									inter,
									test.expected,
									inter.Globals.Get("x").GetValue(),
								)

								AssertValuesEqual(
									t,
									inter,
									test.expected,
									inter.Globals.Get("y").GetValue(),
								)

								AssertValuesEqual(
									t,
									inter,
									interpreter.NewUnmeteredSomeValueNonCopying(
										test.expected,
									),
									inter.Globals.Get("z").GetValue(),
								)
							})
						}

						for _, otherType := range []sema.Type{
							sema.BoolType,
							sema.StringType,
							sema.VoidType,
						} {

							t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

								inter := parseCheckAndInterpret(t,
									fmt.Sprintf(
										`
                                          fun test(): %[4]s? {
                                              let x: %[1]s = %[2]s
                                              let y: %[3]s = x
                                              return y %[5]s %[4]s
                                          }
                                        `,
										test.ty,
										test.value,
										fromType,
										otherType,
										operation.Symbol(),
									),
								)

								result, err := inter.Invoke("test")

								if returnsOptional {
									require.NoError(t, err)
									AssertValuesEqual(
										t,
										inter,
										interpreter.Nil,
										result,
									)
								} else {
									RequireError(t, err)

									require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
								}
							})
						}
					}
				})
			}
		})
	}
}

func TestInterpretDynamicCastingVoid(t *testing.T) {

	t.Parallel()

	types := []sema.Type{
		sema.AnyStructType,
		sema.VoidType,
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  fun f() {}

                                  let x: %[1]s = f()
                                  let y: %[2]s? = x %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.Void,
							inter.Globals.Get("x").GetValue(),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.NewUnmeteredSomeValueNonCopying(
								interpreter.Void,
							),
							inter.Globals.Get("y").GetValue(),
						)
					})
				}

				for _, otherType := range []sema.Type{
					sema.BoolType,
					sema.StringType,
					sema.IntType,
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  fun f() {}

                                  fun test(): %[2]s? {
                                      let x: %[1]s = f()
                                      return x %[3]s %[2]s
                                  }
                                `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}
		})
	}
}

func TestInterpretDynamicCastingString(t *testing.T) {

	t.Parallel()

	types := []sema.Type{
		sema.AnyStructType,
		sema.StringType,
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  let x: %[1]s = "test"
                                  let y: %[2]s? = x %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.NewUnmeteredStringValue("test"),
							inter.Globals.Get("x").GetValue(),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.NewUnmeteredSomeValueNonCopying(
								interpreter.NewUnmeteredStringValue("test"),
							),
							inter.Globals.Get("y").GetValue(),
						)
					})
				}

				for _, otherType := range []sema.Type{
					sema.BoolType,
					sema.VoidType,
					sema.IntType,
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  fun test(): %[2]s? {
                                      let x: String = "test"
                                      let y: %[1]s = x
                                      return y %[3]s %[2]s
                                  }
                                `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}
		})
	}
}

func TestInterpretDynamicCastingBool(t *testing.T) {

	t.Parallel()

	types := []sema.Type{
		sema.AnyStructType,
		sema.BoolType,
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  let x: %[1]s = true
                                  let y: %[2]s? = x %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.TrueValue,
							inter.Globals.Get("x").GetValue(),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.NewUnmeteredSomeValueNonCopying(
								interpreter.TrueValue,
							),
							inter.Globals.Get("y").GetValue(),
						)
					})
				}

				for _, otherType := range []sema.Type{
					sema.StringType,
					sema.VoidType,
					sema.IntType,
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  fun test(): %[2]s? {
                                      let x: Bool = true
                                      let y: %[1]s = x
                                      return y %[3]s %[2]s
                                  }
                                `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}
		})
	}
}

func TestInterpretDynamicCastingAddress(t *testing.T) {

	t.Parallel()

	types := []sema.Type{
		sema.AnyStructType,
		sema.TheAddressType,
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  let x: Address = 0x1
                                  let y: %[1]s = x
                                  let z: %[2]s? = y %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						addressValue := interpreter.AddressValue{
							0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
						}
						AssertValuesEqual(
							t,
							inter,
							addressValue,
							inter.Globals.Get("y").GetValue(),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.NewUnmeteredSomeValueNonCopying(
								addressValue,
							),
							inter.Globals.Get("z").GetValue(),
						)
					})
				}

				for _, otherType := range []sema.Type{
					sema.StringType,
					sema.VoidType,
					sema.IntType,
					sema.BoolType,
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  fun test(): %[2]s? {
                                      let x: Address = 0x1
                                      let y: %[1]s = x
                                      return y %[3]s %[2]s
                                  }
                                `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}
		})
	}
}

func TestInterpretDynamicCastingStruct(t *testing.T) {

	t.Parallel()

	types := []string{
		"AnyStruct",
		"S",
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  struct S {}

                                  let x: %[1]s = S()
                                  let y: %[2]s? = x %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						assert.IsType(t,
							&interpreter.CompositeValue{},
							inter.Globals.Get("x").GetValue(),
						)

						require.IsType(t,
							&interpreter.SomeValue{},
							inter.Globals.Get("y").GetValue(),
						)

						require.IsType(t,
							&interpreter.CompositeValue{},
							inter.Globals.Get("y").GetValue().(*interpreter.SomeValue).
								InnerValue(inter, interpreter.EmptyLocationRange),
						)
					})
				}

				t.Run(fmt.Sprintf("invalid: from %s to T", fromType), func(t *testing.T) {

					inter := parseCheckAndInterpret(t,
						fmt.Sprintf(
							`
                              struct S {}

                              struct T {}

                              fun test(): T? {
                                  let x: S = S()
                                  let y: %[1]s = x
                                  return y %[2]s T
                              }
                            `,
							fromType,
							operation.Symbol(),
						),
					)

					result, err := inter.Invoke("test")

					if returnsOptional {
						require.NoError(t, err)
						AssertValuesEqual(
							t,
							inter,
							interpreter.Nil,
							result,
						)
					} else {
						RequireError(t, err)

						require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
					}
				})

				for _, otherType := range []sema.Type{
					sema.StringType,
					sema.VoidType,
					sema.IntType,
					sema.BoolType,
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  struct S {}

                                  fun test(): %[2]s? {
                                      let x: S = S()
                                      let y: %[1]s = x
                                      return y %[3]s %[2]s
                                  }
                                `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}
		})
	}
}

func returnResourceCasted(fromType, targetType string, operation ast.Operation) string {
	switch operation {
	case ast.OperationFailableCast:
		return fmt.Sprintf(
			`
              fun test(): @%[2]s? {
                  let r: @%[1]s <- create R()
                  if let r2 <- r as? @%[2]s {
                      return <-r2
                  } else {
                      destroy r
                      return nil
                  }
              }
            `,
			fromType,
			targetType,
		)

	case ast.OperationForceCast:
		return fmt.Sprintf(
			`
              fun test(): @%[2]s {
                  let r: @%[1]s <- create R()
                  let r2 <- r as! @%[2]s
                  return <-r2
              }
            `,
			fromType,
			targetType,
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func testResourceCastValid(t *testing.T, types, fromType string, targetType string, operation ast.Operation) {
	inter := parseCheckAndInterpret(t,
		types+
			returnResourceCasted(
				fromType,
				targetType,
				operation,
			),
	)

	value, err := inter.Invoke("test")

	require.NoError(t, err)

	switch operation {
	case ast.OperationFailableCast:
		require.IsType(t,
			&interpreter.SomeValue{},
			value,
		)

		require.IsType(t,
			&interpreter.CompositeValue{},
			value.(*interpreter.SomeValue).
				InnerValue(inter, interpreter.EmptyLocationRange),
		)

	case ast.OperationForceCast:

		require.IsType(t,
			&interpreter.CompositeValue{},
			value,
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func testResourceCastInvalid(t *testing.T, types, fromType, targetType string, operation ast.Operation) {
	inter := parseCheckAndInterpret(t,
		fmt.Sprintf(
			types+returnResourceCasted(
				fromType,
				targetType,
				operation,
			),
		),
	)

	value, err := inter.Invoke("test")

	switch operation {
	case ast.OperationFailableCast:
		require.NoError(t, err)

		require.IsType(t,
			interpreter.Nil,
			value,
		)

	case ast.OperationForceCast:
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

	default:
		panic(errors.NewUnreachableError())
	}
}

func TestInterpretDynamicCastingResource(t *testing.T) {

	t.Parallel()

	types := []string{
		"AnyResource",
		"R",
	}

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						testResourceCastValid(t,
							`
                              resource R {}
                            `,
							fromType,
							targetType,
							operation,
						)
					})
				}

				t.Run(fmt.Sprintf("invalid: from %s to T", fromType), func(t *testing.T) {

					testResourceCastInvalid(t,
						`
                          resource R {}

                          resource T {}
                        `,
						fromType,
						"T",
						operation,
					)
				})
			}
		})
	}
}

func returnStructCasted(fromType, targetType string, operation ast.Operation) string {
	switch operation {
	case ast.OperationFailableCast:
		return fmt.Sprintf(
			`
              fun test(): %[2]s? {
                  let s: %[1]s = S()
                  return s as? %[2]s
              }
            `,
			fromType,
			targetType,
		)

	case ast.OperationForceCast:
		return fmt.Sprintf(
			`
              fun test(): %[2]s {
                  let s: %[1]s = S()
                  return s as! %[2]s
              }
            `,
			fromType,
			targetType,
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func testStructCastValid(t *testing.T, types, fromType string, targetType string, operation ast.Operation) {
	inter := parseCheckAndInterpret(t,
		types+
			returnStructCasted(
				fromType,
				targetType,
				operation,
			),
	)

	value, err := inter.Invoke("test")

	require.NoError(t, err)

	switch operation {
	case ast.OperationFailableCast:
		require.IsType(t,
			&interpreter.SomeValue{},
			value,
		)

		require.IsType(t,
			&interpreter.CompositeValue{},
			value.(*interpreter.SomeValue).
				InnerValue(inter, interpreter.EmptyLocationRange),
		)

	case ast.OperationForceCast:

		require.IsType(t,
			&interpreter.CompositeValue{},
			value,
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func testStructCastInvalid(t *testing.T, types, fromType, targetType string, operation ast.Operation) {
	inter := parseCheckAndInterpret(t,
		fmt.Sprintf(
			types+
				returnStructCasted(
					fromType,
					targetType,
					operation,
				),
		),
	)

	value, err := inter.Invoke("test")

	switch operation {
	case ast.OperationFailableCast:
		require.NoError(t, err)

		require.IsType(t,
			interpreter.Nil,
			value,
		)

	case ast.OperationForceCast:
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

	default:
		panic(errors.NewUnreachableError())
	}
}

func TestInterpretDynamicCastingStructInterface(t *testing.T) {

	t.Parallel()

	types := []string{
		"AnyStruct",
		"S",
		"{I}",
	}

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						testStructCastValid(t,
							`
                              struct interface I {}

                              struct S: I {}
                            `,
							fromType,
							targetType,
							operation,
						)
					})
				}

				t.Run(fmt.Sprintf("invalid: from %s to other struct", fromType), func(t *testing.T) {

					testStructCastInvalid(t,
						`
                          struct interface I {}

                          struct S: I {}

                          struct T: I {}
                        `,
						fromType,
						"T",
						operation,
					)
				})
			}
		})
	}
}

func TestInterpretDynamicCastingResourceInterface(t *testing.T) {

	t.Parallel()

	types := []string{
		"AnyResource",
		"R",
		"{I}",
	}

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						testResourceCastValid(t,
							`
                              resource interface I {}

                              resource R: I {}
                            `,
							fromType,
							targetType,
							operation,
						)
					})
				}

				t.Run(fmt.Sprintf("invalid: from %s to other resource", fromType), func(t *testing.T) {

					testResourceCastInvalid(t,
						`
                          resource interface I {}

                          resource R: I {}

                          resource T: I {}
                        `,
						fromType,
						"T",
						operation,
					)
				})
			}
		})
	}
}

func TestInterpretDynamicCastingSome(t *testing.T) {

	t.Parallel()

	types := []sema.Type{
		&sema.OptionalType{Type: sema.IntType},
		&sema.OptionalType{Type: sema.AnyStructType},
		sema.AnyStructType,
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  let x: Int? = 42
                                  let y: %[1]s = x
                                  let z: %[2]s? = y %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						expectedValue := interpreter.NewUnmeteredSomeValueNonCopying(
							interpreter.NewUnmeteredIntValueFromInt64(42),
						)

						AssertValuesEqual(
							t,
							inter,
							expectedValue,
							inter.Globals.Get("y").GetValue(),
						)

						if targetType == sema.AnyStructType && !returnsOptional {

							AssertValuesEqual(
								t,
								inter,
								expectedValue,
								inter.Globals.Get("z").GetValue(),
							)

						} else {
							AssertValuesEqual(
								t,
								inter,
								interpreter.NewUnmeteredSomeValueNonCopying(
									expectedValue,
								),
								inter.Globals.Get("z").GetValue(),
							)
						}

					})
				}

				for _, otherType := range []sema.Type{
					&sema.OptionalType{Type: sema.StringType},
					&sema.OptionalType{Type: sema.VoidType},
					&sema.OptionalType{Type: sema.BoolType},
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  fun test(): %[2]s? {
	                                  let x: %[1]s = 42
	                                  return x %[3]s %[2]s
	                              }
	                            `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}
		})
	}
}

func TestInterpretDynamicCastingArray(t *testing.T) {

	t.Parallel()

	types := []sema.Type{
		&sema.VariableSizedType{Type: sema.IntType},
		&sema.VariableSizedType{Type: sema.AnyStructType},
		sema.AnyStructType,
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  let x: [Int] = [42]
                                  let y: %[1]s = x
                                  let z: %[2]s? = y %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						expectedElements := []interpreter.Value{
							interpreter.NewUnmeteredIntValueFromInt64(42),
						}

						yValue := inter.Globals.Get("y").GetValue()
						require.IsType(t, yValue, &interpreter.ArrayValue{})
						yArray := yValue.(*interpreter.ArrayValue)

						AssertValueSlicesEqual(
							t,
							inter,
							expectedElements,
							arrayElements(inter, yArray),
						)

						zValue := inter.Globals.Get("z").GetValue()
						require.IsType(t, zValue, &interpreter.SomeValue{})
						zSome := zValue.(*interpreter.SomeValue)

						innerValue := zSome.InnerValue(inter, interpreter.EmptyLocationRange)
						require.IsType(t, innerValue, &interpreter.ArrayValue{})
						innerArray := innerValue.(*interpreter.ArrayValue)

						AssertValueSlicesEqual(
							t,
							inter,
							expectedElements,
							arrayElements(inter, innerArray),
						)
					})
				}

				for _, otherType := range []sema.Type{
					sema.StringType,
					sema.VoidType,
					sema.BoolType,
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
		                          fun test(): [%[2]s]? {
		                              let x: %[1]s = [42]
		                              return x %[3]s [%[2]s]
		                          }
		                        `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}

			t.Run("invalid upcast", func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
		                  fun test(): [Int]? {
		                      let x: [AnyStruct] = []
		                      return x %s [Int]
		                  }
		                `,
						operation.Symbol(),
					),
				)

				result, err := inter.Invoke("test")

				if returnsOptional {
					require.NoError(t, err)
					AssertValuesEqual(
						t,
						inter,

						interpreter.Nil,
						result,
					)
				} else {
					RequireError(t, err)

					require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
				}
			})
		})
	}

	t.Run("[AnyStruct] to [Int]", func(t *testing.T) {
		inter := parseCheckAndInterpret(t, `
		    fun test(): [Int] {
		        let x: [AnyStruct] = [1, 2, 3]
		        return x as! [Int]
		    }
		`)

		_, err := inter.Invoke("test")

		RequireError(t, err)

		assert.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
	})
}

func TestInterpretDynamicCastingDictionary(t *testing.T) {

	t.Parallel()

	types := []sema.Type{
		&sema.DictionaryType{
			KeyType:   sema.StringType,
			ValueType: sema.IntType,
		},
		&sema.DictionaryType{
			KeyType:   sema.StringType,
			ValueType: sema.AnyStructType,
		},
	}

	for operation, returnsOptional := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			for _, fromType := range types {
				for _, targetType := range types {

					t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
                                  let x: {String: Int} = {"test": 42}
                                  let y: %[1]s = x
                                  let z: %[2]s? = y %[3]s %[2]s
                                `,
								fromType,
								targetType,
								operation.Symbol(),
							),
						)

						expectedDictionary := interpreter.NewDictionaryValue(
							inter,
							interpreter.EmptyLocationRange,
							interpreter.DictionaryStaticType{
								KeyType:   interpreter.PrimitiveStaticTypeString,
								ValueType: interpreter.PrimitiveStaticTypeInt,
							},
							interpreter.NewUnmeteredStringValue("test"),
							interpreter.NewUnmeteredIntValueFromInt64(42),
						)

						AssertValuesEqual(
							t,
							inter,
							expectedDictionary,
							inter.Globals.Get("y").GetValue(),
						)

						AssertValuesEqual(
							t,
							inter,
							interpreter.NewUnmeteredSomeValueNonCopying(
								expectedDictionary,
							),
							inter.Globals.Get("z").GetValue(),
						)
					})
				}

				for _, otherType := range []sema.Type{
					sema.StringType,
					sema.VoidType,
					sema.BoolType,
				} {

					t.Run(fmt.Sprintf("invalid: from %s to %s", fromType, otherType), func(t *testing.T) {

						inter := parseCheckAndInterpret(t,
							fmt.Sprintf(
								`
	                              fun test(): {String: %[2]s}? {
	                                  let x: {String: Int} = {"test": 42}
	                                  let y: %[1]s = x
	                                  return y %[3]s {String: %[2]s}
	                              }
	                            `,
								fromType,
								otherType,
								operation.Symbol(),
							),
						)

						result, err := inter.Invoke("test")

						if returnsOptional {
							require.NoError(t, err)
							AssertValuesEqual(
								t,
								inter,
								interpreter.Nil,
								result,
							)
						} else {
							RequireError(t, err)

							require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
						}
					})
				}
			}

			t.Run("invalid upcast", func(t *testing.T) {

				inter := parseCheckAndInterpret(t,
					fmt.Sprintf(
						`
		                  fun test(): {Int: String}? {
		                      let x: {Int: AnyStruct} = {}
		                      return x %s {Int: String}
		                  }
		                `,
						operation.Symbol(),
					),
				)

				result, err := inter.Invoke("test")

				if returnsOptional {
					require.NoError(t, err)
					AssertValuesEqual(
						t,
						inter,
						interpreter.Nil,
						result,
					)
				} else {
					RequireError(t, err)

					require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
				}
			})
		})
	}
}

func TestInterpretDynamicCastingResourceType(t *testing.T) {

	t.Parallel()

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				testResourceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"{I1, I2}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
                    `,
					"{I1}",
					"{I1, I2}",
					operation,
				)
			})

			t.Run("AnyResource -> conforming intersection type", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"AnyResource",
					"{RI}",
					operation,
				)
			})

			t.Run("AnyResource -> non-conforming intersection type", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
                      resource R {}

	                  resource interface TI {}

	                  resource T: TI {}
	                `,
					"AnyResource",
					"{TI}",
					operation,
				)
			})

			// Supertype: Resource

			t.Run("intersection type -> type: same resource", func(t *testing.T) {

				testResourceCastValid(t,
					`
                      resource interface I {}

	                  resource R: I {}
                    `,
					"{I}",
					"R",
					operation,
				)
			})

			t.Run("intersection -> conforming resource", func(t *testing.T) {

				testResourceCastValid(t,
					`
					  resource interface RI {}

	                  resource R: RI {}
	                `,
					"{RI}",
					"R",
					operation,
				)
			})

			t.Run("intersection -> non-conforming resource", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T: RI {}
	                `,
					"{RI}",
					"T",
					operation,
				)
			})

			t.Run("AnyResource -> type: same type", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                `,
					"AnyResource",
					"R",
					operation,
				)
			})

			t.Run("AnyResource -> type: different type", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T: RI {}

	                `,
					"AnyResource",
					"T",
					operation,
				)
			})

			// Supertype: intersection AnyResource

			t.Run("resource -> intersection AnyResource with conformance type", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"R",
					"{RI}",
					operation,
				)
			})

			t.Run("intersection type -> intersection with conformance not in type", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"{I1}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: fewer types", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"{I1, I2}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: more types", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"{I1}",
					"{I1, I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: different types, conforming", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"{I1}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: different types, non-conforming", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}

	                `,
					"{I1}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection with non-conformance type", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}
	                `,
					"{I1}",
					"{I1, I2}",
					operation,
				)
			})

			t.Run("AnyResource -> intersection", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                `,
					"AnyResource",
					"{I}",
					operation,
				)
			})

			// Supertype: AnyResource

			t.Run("intersection -> AnyResource", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"{I1}",
					"AnyResource",
					operation,
				)
			})

			t.Run("type -> AnyResource", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
                    `,
					"R",
					"AnyResource",
					operation,
				)
			})
		})
	}
}

func TestInterpretDynamicCastingStructType(t *testing.T) {

	t.Parallel()

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				testStructCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"{I1, I2}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
                    `,
					"{I1}",
					"{I1, I2}",
					operation,
				)
			})

			t.Run("type -> intersection type: same struct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"S",
					"{I}",
					operation,
				)
			})

			t.Run("AnyStruct -> conforming intersection type", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"AnyStruct",
					"{SI}",
					operation,
				)
			})

			t.Run("AnyStruct -> non-conforming intersection type", func(t *testing.T) {

				testStructCastInvalid(t,
					`
                      struct S {}

	                  struct interface TI {}

	                  struct T: TI {}
	                `,
					"AnyStruct",
					"{TI}",
					operation,
				)
			})

			// Supertype: Struct

			t.Run("intersection type -> type: same struct", func(t *testing.T) {

				testStructCastValid(t,
					`
                      struct interface I {}

	                  struct S: I {}
                    `,
					"{I}",
					"S",
					operation,
				)
			})

			t.Run("intersection AnyStruct -> conforming struct", func(t *testing.T) {

				testStructCastValid(t,
					`
					  struct interface SI {}

	                  struct S: SI {}
	                `,
					"{SI}",
					"S",
					operation,
				)
			})

			t.Run("intersection AnyStruct -> non-conforming struct", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T: SI {}
	                `,
					"{SI}",
					"T",
					operation,
				)
			})

			t.Run("AnyStruct -> type: same type", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                `,
					"AnyStruct",
					"S",
					operation,
				)
			})

			t.Run("AnyStruct -> type: different type", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T: SI {}

	                `,
					"AnyStruct",
					"T",
					operation,
				)
			})

			// Supertype: intersection

			t.Run("struct -> intersection with conformance type", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"S",
					"{SI}",
					operation,
				)
			})

			t.Run("intersection type -> intersection with conformance not in type", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"{I1}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: fewer types", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"{I1, I2}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: more types", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"{I1}",
					"{I1, I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: different types, conforming", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"{I1}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection -> intersection: different types, non-conforming", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}

	                `,
					"{I1}",
					"{I2}",
					operation,
				)
			})

			t.Run("intersection AnyStruct -> intersection AnyStruct with non-conformance type", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}
	                `,
					"{I1}",
					"{I1, I2}",
					operation,
				)
			})

			t.Run("AnyStruct -> intersection AnyStruct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"AnyStruct",
					"{I}",
					operation,
				)
			})

			// Supertype: AnyStruct

			t.Run("intersection type -> AnyStruct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"{I1}",
					"AnyStruct",
					operation,
				)
			})

			t.Run("intersection -> AnyStruct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"{I1}",
					"AnyStruct",
					operation,
				)
			})

			t.Run("type -> AnyStruct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
                    `,
					"S",
					"AnyStruct",
					operation,
				)
			})
		})
	}
}

func returnReferenceCasted(fromType, targetType string, operation ast.Operation, isResource bool) string {
	switch operation {
	case ast.OperationFailableCast:
		if isResource {
			return fmt.Sprintf(
				`
                  fun test(): Bool {
                      let x <- create R()
                      let r = &x as %[1]s
                      let r2 = r as? %[2]s
                      let isSuccess = r2 != nil
                      destroy x
                      return isSuccess
                  }
                `,
				fromType,
				targetType,
			)
		} else {
			return fmt.Sprintf(
				`
                  fun test(): %[2]s? {
                      let x = S()
                      let r = &x as %[1]s
                      return r as? %[2]s
                  }
                `,
				fromType,
				targetType,
			)
		}

	case ast.OperationForceCast:
		if isResource {
			return fmt.Sprintf(
				`
                  fun test(): Bool {
                      let x <- create R()
                      let r = &x as %[1]s
                      let r2 = r as! %[2]s
                      let isSuccess = r2 != nil
                      destroy x
                      return isSuccess
                  }
                `,
				fromType,
				targetType,
			)
		} else {
			return fmt.Sprintf(
				`
                  fun test(): %[2]s {
                      let x = S()
                      let r = &x as %[1]s
                      return r as! %[2]s
                  }
                `,
				fromType,
				targetType,
			)
		}

	default:
		panic(errors.NewUnreachableError())
	}
}

func testReferenceCastValid(t *testing.T, types, fromType, targetType string, operation ast.Operation, isResource bool) {
	inter := parseCheckAndInterpret(t,
		types+
			returnReferenceCasted(fromType, targetType, operation, isResource),
	)

	value, err := inter.Invoke("test")
	require.NoError(t, err)

	switch operation {
	case ast.OperationFailableCast:
		if isResource {
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
			break
		}

		require.IsType(t,
			&interpreter.SomeValue{},
			value,
		)

		require.IsType(t,
			&interpreter.EphemeralReferenceValue{},
			value.(*interpreter.SomeValue).
				InnerValue(inter, interpreter.EmptyLocationRange),
		)

	case ast.OperationForceCast:
		if isResource {
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(true),
				value,
			)
			break
		}

		require.IsType(t,
			&interpreter.EphemeralReferenceValue{},
			value,
		)

	default:
		panic(errors.NewUnreachableError())
	}
}

func testReferenceCastInvalid(t *testing.T, types, fromType, targetType string, operation ast.Operation, isResource bool) {
	inter := parseCheckAndInterpret(t,
		fmt.Sprintf(
			types+
				returnReferenceCasted(fromType, targetType, operation, isResource),
		),
	)

	value, err := inter.Invoke("test")

	switch operation {
	case ast.OperationFailableCast:
		require.NoError(t, err)

		if isResource {
			AssertValuesEqual(
				t,
				inter,
				interpreter.BoolValue(false),
				value,
			)
			break
		}

		require.IsType(t,
			interpreter.Nil,
			value,
		)

	case ast.OperationForceCast:
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})

	default:
		panic(errors.NewUnreachableError())
	}
}

func TestInterpretDynamicCastingAuthorizedResourceReferenceType(t *testing.T) {

	t.Parallel()

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}

					  entitlement E
                    `,
					"auth(E) &{I1, I2}",
					"&{I2}",
					operation,
					true,
				)
			})

			t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}

					  entitlement E
                    `,
					"auth(E) &{I1}",
					"&{I1, I2}",
					operation,
					true,
				)
			})

			t.Run("type -> intersection type: same resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}

					  entitlement E
	                `,
					"auth(E) &R",
					"&{I}",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> conforming intersection type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

					  entitlement E
	                `,
					"auth(E) &{RI}",
					"&{RI}",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> conforming intersection type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

					  entitlement E
	                `,
					"auth(E) &AnyResource",
					"&{RI}",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> non-conforming intersection type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
                      resource R {}

	                  resource interface TI {}

	                  resource T: TI {}

					  entitlement E
	                `,
					"auth(E) &AnyResource",
					"&{TI}",
					operation,
					true,
				)
			})

			// Supertype: Resource

			t.Run("intersection type -> type: same resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}

					  entitlement E
	                `,
					"auth(E) &{I}",
					"&R",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> conforming resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

					  entitlement E
	                `,
					"auth(E) &{RI}",
					"&R",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> non-conforming resource", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T: RI {}

					  entitlement E
	                `,
					"auth(E) &{RI}",
					"&T",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> type: same type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

					  entitlement E
	                `,
					"auth(E) &AnyResource",
					"&R",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> type: different type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T: RI {}

					  entitlement E
	                `,
					"auth(E) &AnyResource",
					"&T",
					operation,
					true,
				)
			})

			// Supertype: intersection AnyResource

			t.Run("resource -> intersection AnyResource with conformance type", func(t *testing.T) {

				testReferenceCastValid(t, `
	                  resource interface RI {}

	                  resource R: RI {}

					  entitlement E
	                `,
					"auth(E) &R",
					"&{RI}",
					operation,
					true,
				)
			})

			t.Run("intersection type -> intersection AnyResource with conformance in type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                   resource interface I {}

	                   resource R: I {}

					   entitlement E
	                `,
					"auth(E) &{I}",
					"&{I}",
					operation,
					true,
				)
			})

			t.Run("intersection type -> intersection AnyResource with conformance not in type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}

					  entitlement E
                    `,
					"auth(E) &{I1}",
					"&{I2}",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> intersection AnyResource: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}

					  entitlement E
	                `,
					"auth(E) &{I1, I2}",
					"&{I2}",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> intersection AnyResource: more types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}

					  entitlement E
	                `,
					"auth(E) &{I1}",
					"&{I1, I2}",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> intersection AnyResource: different types, conforming", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}

					  entitlement E
	                `,
					"auth(E) &{I1}",
					"&{I2}",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> intersection AnyResource: different types, non-conforming", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}

					  entitlement E
	                `,
					"auth(E) &{I1}",
					"&{I2}",
					operation,
					true,
				)
			})

			t.Run("intersection AnyResource -> intersection AnyResource with non-conformance type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&{I1, I2}",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> intersection AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                entitlement E
`,
					"auth(E) &AnyResource",
					"&{I}",
					operation,
					true,
				)
			})

			// Supertype: AnyResource

			t.Run("intersection type -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&AnyResource",
					operation,
					true,
				)
			})

			t.Run("intersection -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&AnyResource",
					operation,
					true,
				)
			})

			t.Run("type -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
                    entitlement E
`,
					"auth(E) &R",
					"&AnyResource",
					operation,
					true,
				)
			})
		})
	}
}

func TestInterpretDynamicCastingAuthorizedStructReferenceType(t *testing.T) {

	t.Parallel()

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {

			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                      entitlement E
`,
					"auth(E) &{I1, I2}",
					"&{I2}",
					operation,
					false,
				)
			})

			t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    entitlement E
`,
					"auth(E) &{I1}",
					"&{I1, I2}",
					operation,
					false,
				)
			})

			t.Run("type -> intersection type: same struct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                entitlement E
`,
					"auth(E) &S",
					"&{I}",
					operation,
					false,
				)
			})

			t.Run("intersection -> conforming intersection type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                  entitlement E
`,
					"auth(E) &{SI}",
					"&{SI}",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> conforming intersection type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                entitlement E
`,
					"auth(E) &AnyStruct",
					"&{SI}",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> non-conforming intersection type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
                      struct S {}

	                  struct interface TI {}

	                  struct T: TI {}
	                entitlement E
`,
					"auth(E) &AnyStruct",
					"&{TI}",
					operation,
					false,
				)
			})

			t.Run("intersection -> conforming struct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                entitlement E
`,
					"auth(E) &{SI}",
					"&S",
					operation,
					false,
				)
			})

			t.Run("intersection AnyStruct -> non-conforming struct", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T: SI {}
	                entitlement E
`,
					"auth(E) &{SI}",
					"&T",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> type: same type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                entitlement E
`,
					"auth(E) &AnyStruct",
					"&S",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> type: different type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T: SI {}
	                entitlement E
`,
					"auth(E) &AnyStruct",
					"&T",
					operation,
					false,
				)
			})

			// Supertype: intersection

			t.Run("struct -> intersection with conformance type", func(t *testing.T) {

				testReferenceCastValid(t, `
	                  struct interface SI {}

	                  struct S: SI {}
	                entitlement E
`,
					"auth(E) &S",
					"&{SI}",
					operation,
					false,
				)
			})
			t.Run("intersection type -> intersection AnyStruct with conformance not in type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
                    entitlement E
`,
					"auth(E) &{I1}",
					"&{I2}",
					operation,
					false,
				)
			})

			t.Run("intersection -> intersection: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                entitlement E
`,
					"auth(E) &{I1, I2}",
					"&{I2}",
					operation,
					false,
				)
			})

			t.Run("intersection -> intersection: more types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&{I1, I2}",
					operation,
					false,
				)
			})

			t.Run("intersection -> intersection: different types, conforming", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&{I2}",
					operation,
					false,
				)
			})

			t.Run("intersection -> intersection: different types, non-conforming", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&{I2}",
					operation,
					false,
				)
			})

			t.Run("intersection -> intersection with non-conformance type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&{I1, I2}",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> intersection", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                entitlement E
`,
					"auth(E) &AnyStruct",
					"&{I}",
					operation,
					false,
				)
			})

			// Supertype: AnyStruct

			t.Run("intersection type -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                entitlement E
`,
					"auth(E) &{I1}",
					"&AnyStruct",
					operation,
					false,
				)
			})

			t.Run("type -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
                    entitlement E
`,
					"auth(E) &S",
					"&AnyStruct",
					operation,
					false,
				)
			})
		})
	}
}

func TestInterpretDynamicCastingUnauthorizedResourceReferenceType(t *testing.T) {

	t.Parallel()

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {
			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&{I1, I2}",
					"&{I2}",
					operation,
					true,
				)
			})

			t.Run("type -> intersection type: same resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I {}

                      resource R: I {}
                    `,
					"&R",
					"&{I}",
					operation,
					true,
				)
			})

			// Supertype: intersection AnyResource

			t.Run("resource -> intersection with conformance type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface RI {}

                      resource R: RI {}
                    `,
					"&R",
					"&{RI}",
					operation,
					true,
				)
			})

			t.Run("intersection type -> intersection with conformance in type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I {}

                      resource R: I {}
                    `,
					"&{I}",
					"&{I}",
					operation,
					true,
				)
			})

			t.Run("intersection -> intersection: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&{I1, I2}",
					"&{I2}",
					operation,
					true,
				)
			})

			// Supertype: AnyResource

			t.Run("intersection type -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&{I1}",
					"&AnyResource",
					operation,
					true,
				)
			})

			t.Run("type -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&R",
					"&AnyResource",
					operation,
					true,
				)
			})
		})
	}
}

func TestInterpretDynamicCastingUnauthorizedStructReferenceType(t *testing.T) {

	t.Parallel()

	for operation := range dynamicCastingOperations {

		t.Run(operation.Symbol(), func(t *testing.T) {
			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&{I1, I2}",
					"&{I2}",
					operation,
					false,
				)
			})

			t.Run("type -> intersection type: same struct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I {}

                      struct S: I {}
                    `,
					"&S",
					"&{I}",
					operation,
					false,
				)
			})

			// Supertype: intersection AnyStruct

			t.Run("struct -> intersection with conformance type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface RI {}

                      struct S: RI {}
                    `,
					"&S",
					"&{RI}",
					operation,
					false,
				)
			})

			t.Run("intersection -> intersection: fewer types", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&{I1, I2}",
					"&{I2}",
					operation,
					false,
				)
			})

			// Supertype: AnyStruct

			t.Run("intersection type -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&{I1}",
					"&AnyStruct",
					operation,
					false,
				)
			})

			t.Run("type -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&S",
					"&AnyStruct",
					operation,
					false,
				)
			})
		})
	}
}

func TestInterpretDynamicCastingCapability(t *testing.T) {

	t.Parallel()

	test := func(
		name string,
		newCapabilityValue func(borrowType interpreter.StaticType) interpreter.CapabilityValue,
	) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			structType := &sema.CompositeType{
				Location:   TestLocation,
				Identifier: "S",
				Kind:       common.CompositeKindStructure,
			}

			capabilityValue := newCapabilityValue(
				interpreter.ConvertSemaToStaticType(
					nil,
					&sema.ReferenceType{
						Type:          structType,
						Authorization: sema.UnauthorizedAccess,
					},
				),
			)

			types := []sema.Type{
				&sema.CapabilityType{
					BorrowType: &sema.ReferenceType{
						Type:          structType,
						Authorization: sema.UnauthorizedAccess,
					},
				},
				&sema.CapabilityType{
					BorrowType: &sema.ReferenceType{
						Type:          sema.AnyStructType,
						Authorization: sema.UnauthorizedAccess,
					},
				},
				&sema.CapabilityType{},
				sema.AnyStructType,
			}

			capabilityValueDeclaration := stdlib.StandardLibraryValue{
				Name: "cap",
				Type: &sema.CapabilityType{
					BorrowType: &sema.ReferenceType{
						Type:          structType,
						Authorization: sema.UnauthorizedAccess,
					},
				},
				Value: capabilityValue,
				Kind:  common.DeclarationKindConstant,
			}

			baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
			baseValueActivation.DeclareValue(capabilityValueDeclaration)

			baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
			interpreter.Declare(baseActivation, capabilityValueDeclaration)

			options := ParseCheckAndInterpretOptions{
				CheckerConfig: &sema.Config{
					BaseValueActivation: baseValueActivation,
				},
				Config: &interpreter.Config{
					BaseActivation: baseActivation,
				},
			}

			for operation, returnsOptional := range dynamicCastingOperations {

				t.Run(operation.Symbol(), func(t *testing.T) {

					for _, fromType := range types {
						for _, targetType := range types {

							t.Run(fmt.Sprintf("valid: from %s to %s", fromType, targetType), func(t *testing.T) {

								inter, err := parseCheckAndInterpretWithOptions(t,
									fmt.Sprintf(
										`
                                  struct S {}
                                  let x: %[1]s = cap
                                  let y: %[2]s? = x %[3]s %[2]s
                                `,
										fromType,
										targetType,
										operation.Symbol(),
									),
									options,
								)
								require.NoError(t, err)

								AssertValuesEqual(
									t,
									inter,
									capabilityValue,
									inter.Globals.Get("x").GetValue(),
								)

								AssertValuesEqual(
									t,
									inter,
									interpreter.NewUnmeteredSomeValueNonCopying(
										capabilityValue,
									),
									inter.Globals.Get("y").GetValue(),
								)
							})
						}

						for _, otherType := range []sema.Type{
							sema.StringType,
							sema.VoidType,
							sema.BoolType,
						} {

							t.Run(fmt.Sprintf("invalid: from %s to Capability<&%s>", fromType, otherType), func(t *testing.T) {

								inter, err := parseCheckAndInterpretWithOptions(t,
									fmt.Sprintf(
										`
                                  struct S {}

		                          fun test(): Capability<&%[2]s>? {
		                              let x: %[1]s = cap
		                              return x %[3]s Capability<&%[2]s>
		                          }
		                        `,
										fromType,
										otherType,
										operation.Symbol(),
									),
									options,
								)
								require.NoError(t, err)

								result, err := inter.Invoke("test")

								if returnsOptional {
									require.NoError(t, err)
									AssertValuesEqual(
										t,
										inter,
										interpreter.Nil,
										result,
									)
								} else {
									RequireError(t, err)

									require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
								}
							})
						}
					}
				})
			}
		})
	}

	test(
		"path capability",
		func(borrowType interpreter.StaticType) interpreter.CapabilityValue {
			return interpreter.NewUnmeteredPathCapabilityValue(
				interpreter.AddressValue{},
				interpreter.EmptyPathValue,
				borrowType,
			)
		},
	)
	test("path capability",
		func(borrowType interpreter.StaticType) interpreter.CapabilityValue {
			return interpreter.NewUnmeteredIDCapabilityValue(
				4,
				interpreter.AddressValue{},
				borrowType,
			)
		},
	)
}

func TestInterpretResourceConstructorCast(t *testing.T) {

	t.Parallel()

	for operation, returnsOptional := range dynamicCastingOperations {
		inter := parseCheckAndInterpret(t,
			fmt.Sprintf(`
                  resource R {}

                  fun test(): AnyStruct {
                      return R %s fun(): @R
                  }
                `,
				operation.Symbol(),
			),
		)

		result, err := inter.Invoke("test")
		if returnsOptional {
			require.NoError(t, err)
			require.Equal(t, interpreter.Nil, result)
		} else {
			RequireError(t, err)
		}
	}
}

func TestInterpretFunctionTypeCasting(t *testing.T) {

	t.Parallel()

	t.Run("function casting", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): String {
                let x: AnyStruct = foo
                let y = x as! fun(String):String

                return y("hello")
            }

            fun foo(a: String): String {
                return a.concat(" from foo")
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("hello from foo"), result)
	})

	t.Run("param contravariance", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): String {
                let x = foo as fun(String):String
                return x("hello")
            }

            fun foo(a: AnyStruct): String {
                return (a as! String).concat(" from foo")
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("hello from foo"), result)
	})

	t.Run("param contravariance negative", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): String {
                 let x = foo as! fun(AnyStruct):String
                 return x("hello")
            }

            fun foo(a: String): String {
                return a.concat(" from foo")
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
	})

	t.Run("return type covariance", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): AnyStruct {
                let x = foo as! fun(String):AnyStruct
                return x("hello")
            }

            fun foo(a: String): String {
                return a.concat(" from foo")
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("hello from foo"), result)
	})

	t.Run("return type covariance negative", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): String {
                let x = foo as! fun(String):String
                return x("hello")
            }

            fun foo(a: String): AnyStruct {
                return a.concat(" from foo")
            }
        `)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		require.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
	})

	t.Run("bound function casting", func(t *testing.T) {
		t.Parallel()

		inter := parseCheckAndInterpret(t, `
            fun test(): String {
                let x = foo()
                let y: AnyStruct = x.bar
                let z = y as! fun(String):String
                return z("hello")
            }

            struct foo {
                fun bar(a: String): String {
                    return a.concat(" from foo.bar")
                }
            }
        `)

		result, err := inter.Invoke("test")
		require.NoError(t, err)
		require.Equal(t, interpreter.NewUnmeteredStringValue("hello from foo.bar"), result)
	})
}

func TestInterpretReferenceCasting(t *testing.T) {

	t.Parallel()

	t.Run("array", func(t *testing.T) {
		t.Parallel()

		code := `
            fun test() {
                let x = bar()
                let y = [&x as &AnyStruct]
                let z = y as! [&{foo}]
            }

            struct interface foo {}

            struct bar: foo {}
        `

		inter := parseCheckAndInterpret(t, code)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		assert.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
	})

	t.Run("dictionary", func(t *testing.T) {
		t.Parallel()

		code := `
            fun test() {
                let x = bar()
                let y = {"a": &x as &AnyStruct}
                let z = y as! {String: &{foo}}
            }

            struct interface foo {}

            struct bar: foo {}
        `

		inter := parseCheckAndInterpret(t, code)

		_, err := inter.Invoke("test")
		RequireError(t, err)

		assert.ErrorAs(t, err, &interpreter.ForceCastTypeMismatchError{})
	})
}
