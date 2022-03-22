/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

	. "github.com/onflow/cadence/runtime/tests/utils"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/stdlib"
)

func TestInterpretEquality(t *testing.T) {

	t.Parallel()

	t.Run("capability", func(t *testing.T) {

		t.Parallel()

		capabilityValueDeclaration := stdlib.StandardLibraryValue{
			Name: "cap",
			Type: &sema.CapabilityType{},
			ValueFactory: func(_ *interpreter.Interpreter) interpreter.Value {
				return &interpreter.CapabilityValue{
					Address: interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
					Path: interpreter.PathValue{
						Domain:     common.PathDomainStorage,
						Identifier: "something",
					},
				}
			},
			Kind: common.DeclarationKindConstant,
		}

		inter, err := parseCheckAndInterpretWithOptions(t,
			`
              let maybeCapNonNil: Capability? = cap
              let maybeCapNil: Capability? = nil
              let res1 = maybeCapNonNil != nil
              let res2 = maybeCapNil == nil
		    `,
			ParseCheckAndInterpretOptions{
				Options: []interpreter.Option{
					interpreter.WithPredeclaredValues([]interpreter.ValueDeclaration{
						capabilityValueDeclaration,
					}),
				},
				CheckerOptions: []sema.Option{
					sema.WithPredeclaredValues([]sema.ValueDeclaration{
						capabilityValueDeclaration,
					}),
				},
			},
		)
		require.NoError(t, err)

		AssertValuesEqual(
			t,
			inter,
			interpreter.BoolValue(true),
			inter.Globals["res1"].GetValue(),
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.BoolValue(true),
			inter.Globals["res2"].GetValue(),
		)
	})

	t.Run("function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
		  fun func() {}

          let maybeFuncNonNil: ((): Void)? = func
          let maybeFuncNil: ((): Void)? = nil
          let res1 = maybeFuncNonNil != nil
          let res2 = maybeFuncNil == nil
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.BoolValue(true),
			inter.Globals["res1"].GetValue(),
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.BoolValue(true),
			inter.Globals["res2"].GetValue(),
		)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndInterpret(t, `
          let n: Int? = 1
          let res = nil == n
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.BoolValue(false),
			inter.Globals["res"].GetValue(),
		)
	})
}

func TestInterpretEqualityOnNumericSuperTypes(t *testing.T) {

	t.Parallel()

	operations := []ast.Operation{
		ast.OperationEqual,
		ast.OperationNotEqual,
	}

	t.Run("Integer subtypes", func(t *testing.T) {
		t.Parallel()

		intSubtypes := []interpreter.StaticType{
			interpreter.PrimitiveStaticTypeInt,
			interpreter.PrimitiveStaticTypeInt8,
			interpreter.PrimitiveStaticTypeInt16,
			interpreter.PrimitiveStaticTypeInt32,
			interpreter.PrimitiveStaticTypeInt64,
			interpreter.PrimitiveStaticTypeInt128,
			interpreter.PrimitiveStaticTypeInt256,
			interpreter.PrimitiveStaticTypeUInt,
			interpreter.PrimitiveStaticTypeUInt8,
			interpreter.PrimitiveStaticTypeUInt16,
			interpreter.PrimitiveStaticTypeUInt32,
			interpreter.PrimitiveStaticTypeUInt64,
			interpreter.PrimitiveStaticTypeUInt128,
			interpreter.PrimitiveStaticTypeUInt256,
			interpreter.PrimitiveStaticTypeWord8,
			interpreter.PrimitiveStaticTypeWord16,
			interpreter.PrimitiveStaticTypeWord32,
			interpreter.PrimitiveStaticTypeWord64,
		}

		for _, subtype := range intSubtypes {
			rhsType := interpreter.PrimitiveStaticTypeInt
			if subtype == rhsType {
				rhsType = interpreter.PrimitiveStaticTypeUInt
			}

			for _, op := range operations {
				t.Run(fmt.Sprintf("%s,%s", op.String(), subtype.String()), func(t *testing.T) {

					code := fmt.Sprintf(`
                        fun test(): Bool {
                            let x: Integer = 5 as %s
                            let y: Integer = 2 as %s
                            return x %s y
                        }`,
						subtype.String(),
						rhsType.String(),
						op.Symbol(),
					)

					inter := parseCheckAndInterpret(t, code)

					result, err := inter.Invoke("test")

					switch op {
					case ast.OperationEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.BoolValue(false), result)
					case ast.OperationNotEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.BoolValue(true), result)
					default:
						require.Error(t, err)

						operandError := &interpreter.InvalidOperandsError{}
						require.ErrorAs(t, err, operandError)

						assert.Equal(t, op, operandError.Operation)
						assert.Equal(t, subtype, operandError.LeftType)
						assert.Equal(t, rhsType, operandError.RightType)
					}
				})
			}
		}
	})

	t.Run("Fixed point subtypes", func(t *testing.T) {
		t.Parallel()

		fixedPointSubtypes := []interpreter.StaticType{
			interpreter.PrimitiveStaticTypeFix64,
		}

		rhsType := interpreter.PrimitiveStaticTypeUFix64

		for _, subtype := range fixedPointSubtypes {
			for _, op := range operations {
				t.Run(fmt.Sprintf("%s,%s", op.String(), subtype.String()), func(t *testing.T) {

					code := fmt.Sprintf(`
                        fun test(): Bool {
                            let x: FixedPoint = 5.2 as %s
                            let y: FixedPoint = 2.3 as %s
                            return x %s y
                        }`,
						subtype.String(),
						rhsType.String(),
						op.Symbol(),
					)

					inter := parseCheckAndInterpret(t, code)

					result, err := inter.Invoke("test")

					switch op {
					case ast.OperationEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.BoolValue(false), result)
					case ast.OperationNotEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.BoolValue(true), result)
					default:
						require.Error(t, err)

						operandError := &interpreter.InvalidOperandsError{}
						require.ErrorAs(t, err, operandError)

						assert.Equal(t, op, operandError.Operation)
						assert.Equal(t, subtype, operandError.LeftType)
						assert.Equal(t, rhsType, operandError.RightType)
					}
				})
			}
		}
	})

	t.Run("Unsigned fixed point subtypes", func(t *testing.T) {
		t.Parallel()

		fixedPointSubtypes := []interpreter.StaticType{
			interpreter.PrimitiveStaticTypeUFix64,
		}

		rhsType := interpreter.PrimitiveStaticTypeFix64

		for _, subtype := range fixedPointSubtypes {
			for _, op := range operations {
				t.Run(fmt.Sprintf("%s,%s", op.String(), subtype.String()), func(t *testing.T) {

					code := fmt.Sprintf(`
                        fun test(): Bool {
                            let x: FixedPoint = 5.2 as %s
                            let y: FixedPoint = 2.3 as %s
                            return x %s y
                        }`,
						subtype.String(),
						rhsType.String(),
						op.Symbol(),
					)

					inter := parseCheckAndInterpret(t, code)

					result, err := inter.Invoke("test")

					switch op {
					case ast.OperationEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.BoolValue(false), result)
					case ast.OperationNotEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.BoolValue(true), result)
					default:
						require.Error(t, err)

						operandError := &interpreter.InvalidOperandsError{}
						require.ErrorAs(t, err, operandError)

						assert.Equal(t, op, operandError.Operation)
						assert.Equal(t, subtype, operandError.LeftType)
						assert.Equal(t, rhsType, operandError.RightType)
					}
				})
			}
		}
	})
}
