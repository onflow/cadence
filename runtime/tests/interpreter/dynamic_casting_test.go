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
		"AnyStruct{I}",
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
		"AnyResource{I}",
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

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				testResourceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"R{I1, I2}",
					"R{I2}",
					operation,
				)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
                    `,
					"R{I1}",
					"R{I1, I2}",
					operation,
				)
			})

			t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                `,
					"R",
					"R{I}",
					operation,
				)
			})

			t.Run("restricted AnyResource -> conforming restricted type", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"AnyResource{RI}",
					"R{RI}",
					operation,
				)
			})

			// TODO: should statically fail?
			t.Run("restricted AnyResource -> non-conforming restricted type", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T {}
	                `,
					"AnyResource{RI}",
					"T{}",
					operation,
				)
			})

			t.Run("AnyResource -> conforming restricted type", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"AnyResource",
					"R{RI}",
					operation,
				)
			})

			t.Run("AnyResource -> non-conforming restricted type", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
                      resource R {}

	                  resource interface TI {}

	                  resource T: TI {}
	                `,
					"AnyResource",
					"T{TI}",
					operation,
				)
			})

			// Supertype: Resource (unrestricted)

			t.Run("restricted type -> unrestricted type: same resource", func(t *testing.T) {

				testResourceCastValid(t,
					`
                      resource interface I {}

	                  resource R: I {}
                    `,
					"R{I}",
					"R",
					operation,
				)
			})

			t.Run("restricted AnyResource -> conforming resource", func(t *testing.T) {

				testResourceCastValid(t,
					`
					  resource interface RI {}

	                  resource R: RI {}
	                `,
					"AnyResource{RI}",
					"R",
					operation,
				)
			})

			t.Run("restricted AnyResource -> non-conforming resource", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T: RI {}
	                `,
					"AnyResource{RI}",
					"T",
					operation,
				)
			})

			t.Run("AnyResource -> unrestricted type: same type", func(t *testing.T) {

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

			t.Run("AnyResource -> unrestricted type: different type", func(t *testing.T) {

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

			// Supertype: restricted AnyResource

			t.Run("resource -> restricted AnyResource with conformance restriction", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"R",
					"AnyResource{RI}",
					operation,
				)
			})

			t.Run("restricted type -> restricted AnyResource with conformance in restriction", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                `,
					"R{I}",
					"AnyResource{I}",
					operation,
				)
			})

			t.Run("restricted type -> restricted AnyResource with conformance not in restriction", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"R{I1}",
					"AnyResource{I2}",
					operation,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: fewer restrictions", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"AnyResource{I1, I2}",
					"AnyResource{I2}",
					operation,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: more restrictions", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"AnyResource{I1}",
					"AnyResource{I1, I2}",
					operation,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: different restrictions, conforming", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"AnyResource{I1}",
					"AnyResource{I2}",
					operation,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: different restrictions, non-conforming", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}

	                `,
					"AnyResource{I1}",
					"AnyResource{I2}",
					operation,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

				testResourceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}
	                `,
					"AnyResource{I1}",
					"AnyResource{I1, I2}",
					operation,
				)
			})

			t.Run("AnyResource -> restricted AnyResource", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                `,
					"AnyResource",
					"AnyResource{I}",
					operation,
				)
			})

			// Supertype: AnyResource

			t.Run("restricted type -> AnyResource", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"R{I1}",
					"AnyResource",
					operation,
				)
			})

			t.Run("restricted AnyResource -> AnyResource", func(t *testing.T) {

				testResourceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"AnyResource{I1}",
					"AnyResource",
					operation,
				)
			})

			t.Run("unrestricted type -> AnyResource", func(t *testing.T) {

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

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				testStructCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"S{I1, I2}",
					"S{I2}",
					operation,
				)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
                    `,
					"S{I1}",
					"S{I1, I2}",
					operation,
				)
			})

			t.Run("unrestricted type -> restricted type: same struct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"S",
					"S{I}",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> conforming restricted type", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"AnyStruct{SI}",
					"S{SI}",
					operation,
				)
			})

			// TODO: should statically fail?
			t.Run("restricted AnyStruct -> non-conforming restricted type", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T {}
	                `,
					"AnyStruct{SI}",
					"T{}",
					operation,
				)
			})

			t.Run("AnyStruct -> conforming restricted type", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"AnyStruct",
					"S{SI}",
					operation,
				)
			})

			t.Run("AnyStruct -> non-conforming restricted type", func(t *testing.T) {

				testStructCastInvalid(t,
					`
                      struct S {}

	                  struct interface TI {}

	                  struct T: TI {}
	                `,
					"AnyStruct",
					"T{TI}",
					operation,
				)
			})

			// Supertype: Struct (unrestricted)

			t.Run("restricted type -> unrestricted type: same struct", func(t *testing.T) {

				testStructCastValid(t,
					`
                      struct interface I {}

	                  struct S: I {}
                    `,
					"S{I}",
					"S",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> conforming struct", func(t *testing.T) {

				testStructCastValid(t,
					`
					  struct interface SI {}

	                  struct S: SI {}
	                `,
					"AnyStruct{SI}",
					"S",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> non-conforming struct", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T: SI {}
	                `,
					"AnyStruct{SI}",
					"T",
					operation,
				)
			})

			t.Run("AnyStruct -> unrestricted type: same type", func(t *testing.T) {

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

			t.Run("AnyStruct -> unrestricted type: different type", func(t *testing.T) {

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

			// Supertype: restricted AnyStruct

			t.Run("struct -> restricted AnyStruct with conformance restriction", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"S",
					"AnyStruct{SI}",
					operation,
				)
			})

			t.Run("restricted type -> restricted AnyStruct with conformance in restriction", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"S{I}",
					"AnyStruct{I}",
					operation,
				)
			})

			t.Run("restricted type -> restricted AnyStruct with conformance not in restriction", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"S{I1}",
					"AnyStruct{I2}",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: fewer restrictions", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"AnyStruct{I1, I2}",
					"AnyStruct{I2}",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: more restrictions", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"AnyStruct{I1}",
					"AnyStruct{I1, I2}",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: different restrictions, conforming", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"AnyStruct{I1}",
					"AnyStruct{I2}",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: different restrictions, non-conforming", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}

	                `,
					"AnyStruct{I1}",
					"AnyStruct{I2}",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

				testStructCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}
	                `,
					"AnyStruct{I1}",
					"AnyStruct{I1, I2}",
					operation,
				)
			})

			t.Run("AnyStruct -> restricted AnyStruct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"AnyStruct",
					"AnyStruct{I}",
					operation,
				)
			})

			// Supertype: AnyStruct

			t.Run("restricted type -> AnyStruct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"S{I1}",
					"AnyStruct",
					operation,
				)
			})

			t.Run("restricted AnyStruct -> AnyStruct", func(t *testing.T) {

				testStructCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"AnyStruct{I1}",
					"AnyStruct",
					operation,
				)
			})

			t.Run("unrestricted type -> AnyStruct", func(t *testing.T) {

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
                  fun test(): %[2]s? {
                      let x <- create R()
                      let r = &x as %[1]s
                      let r2 = r as? %[2]s
                      destroy x
                      return r2
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
                  fun test(): %[2]s {
                      let x <- create R()
                      let r = &x as %[1]s
                      let r2 = r as! %[2]s
                      destroy x
                      return r2
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

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"auth &R{I1, I2}",
					"&R{I2}",
					operation,
					true,
				)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"auth &R{I1}",
					"&R{I1, I2}",
					operation,
					true,
				)
			})

			t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                `,
					"auth &R",
					"&R{I}",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> conforming restricted type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"auth &AnyResource{RI}",
					"&R{RI}",
					operation,
					true,
				)
			})

			// TODO: should statically fail?
			t.Run("restricted AnyResource -> non-conforming restricted type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T {}
	                `,
					"auth &AnyResource{RI}",
					"&T{}",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> conforming restricted type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"auth &AnyResource",
					"&R{RI}",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> non-conforming restricted type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
                      resource R {}

	                  resource interface TI {}

	                  resource T: TI {}
	                `,
					"auth &AnyResource",
					"&T{TI}",
					operation,
					true,
				)
			})

			// Supertype: Resource (unrestricted)

			t.Run("restricted type -> unrestricted type: same resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                `,
					"auth &R{I}",
					"&R",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> conforming resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"auth &AnyResource{RI}",
					"&R",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> non-conforming resource", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T: RI {}
	                `,
					"auth &AnyResource{RI}",
					"&T",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> unrestricted type: same type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"auth &AnyResource",
					"&R",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> unrestricted type: different type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface RI {}

	                  resource R: RI {}

	                  resource T: RI {}
	                `,
					"auth &AnyResource",
					"&T",
					operation,
					true,
				)
			})

			// Supertype: restricted AnyResource

			t.Run("resource -> restricted AnyResource with conformance restriction", func(t *testing.T) {

				testReferenceCastValid(t, `
	                  resource interface RI {}

	                  resource R: RI {}
	                `,
					"auth &R",
					"&AnyResource{RI}",
					operation,
					true,
				)
			})

			t.Run("restricted type -> restricted AnyResource with conformance in restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                   resource interface I {}

	                   resource R: I {}
	                `,
					"auth &R{I}",
					"&AnyResource{I}",
					operation,
					true,
				)
			})

			t.Run("restricted type -> restricted AnyResource with conformance not in restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
                    `,
					"auth &R{I1}",
					"&AnyResource{I2}",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"auth &AnyResource{I1, I2}",
					"&AnyResource{I2}",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: more restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"auth &AnyResource{I1}",
					"&AnyResource{I1, I2}",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: different restrictions, conforming", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"auth &AnyResource{I1}",
					"&AnyResource{I2}",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: different restrictions, non-conforming", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}
	                `,
					"auth &AnyResource{I1}",
					"&AnyResource{I2}",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1 {}
	                `,
					"auth &AnyResource{I1}",
					"&AnyResource{I1, I2}",
					operation,
					true,
				)
			})

			t.Run("AnyResource -> restricted AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I {}

	                  resource R: I {}
	                `,
					"auth &AnyResource",
					"&AnyResource{I}",
					operation,
					true,
				)
			})

			// Supertype: AnyResource

			t.Run("restricted type -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"auth &R{I1}",
					"&AnyResource",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
	                `,
					"auth &AnyResource{I1}",
					"&AnyResource",
					operation,
					true,
				)
			})

			t.Run("unrestricted type -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  resource interface I1 {}

	                  resource interface I2 {}

	                  resource R: I1, I2 {}
                    `,
					"auth &R",
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

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"auth &S{I1, I2}",
					"&S{I2}",
					operation,
					false,
				)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"auth &S{I1}",
					"&S{I1, I2}",
					operation,
					false,
				)
			})

			t.Run("unrestricted type -> restricted type: same struct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"auth &S",
					"&S{I}",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> conforming restricted type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"auth &AnyStruct{SI}",
					"&S{SI}",
					operation,
					false,
				)
			})

			// TODO: should statically fail?
			t.Run("restricted AnyStruct -> non-conforming restricted type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T {}
	                `,
					"auth &AnyStruct{SI}",
					"&T{}",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> conforming restricted type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"auth &AnyStruct",
					"&S{SI}",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> non-conforming restricted type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
                      struct S {}

	                  struct interface TI {}

	                  struct T: TI {}
	                `,
					"auth &AnyStruct",
					"&T{TI}",
					operation,
					false,
				)
			})

			// Supertype: Struct (unrestricted)

			t.Run("restricted type -> unrestricted type: same struct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"auth &S{I}",
					"&S",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> conforming struct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"auth &AnyStruct{SI}",
					"&S",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> non-conforming struct", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T: SI {}
	                `,
					"auth &AnyStruct{SI}",
					"&T",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> unrestricted type: same type", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"auth &AnyStruct",
					"&S",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> unrestricted type: different type", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface SI {}

	                  struct S: SI {}

	                  struct T: SI {}
	                `,
					"auth &AnyStruct",
					"&T",
					operation,
					false,
				)
			})

			// Supertype: restricted AnyStruct

			t.Run("struct -> restricted AnyStruct with conformance restriction", func(t *testing.T) {

				testReferenceCastValid(t, `
	                  struct interface SI {}

	                  struct S: SI {}
	                `,
					"auth &S",
					"&AnyStruct{SI}",
					operation,
					false,
				)
			})

			t.Run("restricted type -> restricted AnyStruct with conformance in restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                   struct interface I {}

	                   struct S: I {}
	                `,
					"auth &S{I}",
					"&AnyStruct{I}",
					operation,
					false,
				)
			})

			t.Run("restricted type -> restricted AnyStruct with conformance not in restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
                    `,
					"auth &S{I1}",
					"&AnyStruct{I2}",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"auth &AnyStruct{I1, I2}",
					"&AnyStruct{I2}",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: more restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"auth &AnyStruct{I1}",
					"&AnyStruct{I1, I2}",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: different restrictions, conforming", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"auth &AnyStruct{I1}",
					"&AnyStruct{I2}",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: different restrictions, non-conforming", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}
	                `,
					"auth &AnyStruct{I1}",
					"&AnyStruct{I2}",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

				testReferenceCastInvalid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1 {}
	                `,
					"auth &AnyStruct{I1}",
					"&AnyStruct{I1, I2}",
					operation,
					false,
				)
			})

			t.Run("AnyStruct -> restricted AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I {}

	                  struct S: I {}
	                `,
					"auth &AnyStruct",
					"&AnyStruct{I}",
					operation,
					false,
				)
			})

			// Supertype: AnyStruct

			t.Run("restricted type -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"auth &S{I1}",
					"&AnyStruct",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
	                `,
					"auth &AnyStruct{I1}",
					"&AnyStruct",
					operation,
					false,
				)
			})

			t.Run("unrestricted type -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
	                  struct interface I1 {}

	                  struct interface I2 {}

	                  struct S: I1, I2 {}
                    `,
					"auth &S",
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
			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&R{I1, I2}",
					"&R{I2}",
					operation,
					true,
				)
			})

			t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I {}

                      resource R: I {}
                    `,
					"&R",
					"&R{I}",
					operation,
					true,
				)
			})

			// Supertype: restricted AnyResource

			t.Run("resource -> restricted AnyResource with conformance restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface RI {}

                      resource R: RI {}
                    `,
					"&R",
					"&AnyResource{RI}",
					operation,
					true,
				)
			})

			t.Run("restricted type -> restricted AnyResource with conformance in restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I {}

                      resource R: I {}
                    `,
					"&R{I}",
					"&AnyResource{I}",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> restricted AnyResource: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&AnyResource{I1, I2}",
					"&AnyResource{I2}",
					operation,
					true,
				)
			})

			// Supertype: AnyResource

			t.Run("restricted type -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&R{I1}",
					"&AnyResource",
					operation,
					true,
				)
			})

			t.Run("restricted AnyResource -> AnyResource", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}
                    `,
					"&AnyResource{I1}",
					"&AnyResource",
					operation,
					true,
				)
			})

			t.Run("unrestricted type -> AnyResource", func(t *testing.T) {

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
			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&S{I1, I2}",
					"&S{I2}",
					operation,
					false,
				)
			})

			t.Run("unrestricted type -> restricted type: same struct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I {}

                      struct S: I {}
                    `,
					"&S",
					"&S{I}",
					operation,
					false,
				)
			})

			// Supertype: restricted AnyStruct

			t.Run("struct -> restricted AnyStruct with conformance restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface RI {}

                      struct S: RI {}
                    `,
					"&S",
					"&AnyStruct{RI}",
					operation,
					false,
				)
			})

			t.Run("restricted type -> restricted AnyStruct with conformance in restriction", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I {}

                      struct S: I {}
                    `,
					"&S{I}",
					"&AnyStruct{I}",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> restricted AnyStruct: fewer restrictions", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&AnyStruct{I1, I2}",
					"&AnyStruct{I2}",
					operation,
					false,
				)
			})

			// Supertype: AnyStruct

			t.Run("restricted type -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&S{I1}",
					"&AnyStruct",
					operation,
					false,
				)
			})

			t.Run("restricted AnyStruct -> AnyStruct", func(t *testing.T) {

				testReferenceCastValid(t,
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}
                    `,
					"&AnyStruct{I1}",
					"&AnyStruct",
					operation,
					false,
				)
			})

			t.Run("unrestricted type -> AnyStruct", func(t *testing.T) {

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

	structType := &sema.CompositeType{
		Location:   TestLocation,
		Identifier: "S",
		Kind:       common.CompositeKindStructure,
	}

	types := []sema.Type{
		&sema.CapabilityType{
			BorrowType: &sema.ReferenceType{
				Type: structType,
			},
		},
		&sema.CapabilityType{
			BorrowType: &sema.ReferenceType{
				Type: sema.AnyStructType,
			},
		},
		&sema.CapabilityType{},
		sema.AnyStructType,
	}

	capabilityValue := &interpreter.StorageCapabilityValue{
		Address: interpreter.AddressValue{},
		Path:    interpreter.EmptyPathValue,
		BorrowType: interpreter.ConvertSemaToStaticType(
			nil,
			&sema.ReferenceType{
				Type: structType,
			},
		),
	}

	capabilityValueDeclaration := stdlib.StandardLibraryValue{
		Name: "cap",
		Type: &sema.CapabilityType{
			BorrowType: &sema.ReferenceType{
				Type: structType,
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
}

func TestInterpretResourceConstructorCast(t *testing.T) {

	t.Parallel()

	for operation, returnsOptional := range dynamicCastingOperations {
		inter := parseCheckAndInterpret(t,
			fmt.Sprintf(`
                  resource R {}

                  fun test(): AnyStruct {
                      return R %s ((): @R)
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
                let y = x as! ((String):String)

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
                let x = foo as ((String):String)
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
                 let x = foo as! ((AnyStruct):String)
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
                let x = foo as! ((String):AnyStruct)
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
                let x = foo as! ((String):String)
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
                let z = y as! ((String):String)
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
                let z = y as! [&bar{foo}]
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
                let z = y as! {String: &bar{foo}}
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
