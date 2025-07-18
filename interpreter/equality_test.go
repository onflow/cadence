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
	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/interpreter"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
	. "github.com/onflow/cadence/test_utils/interpreter_utils"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestInterpretEquality(t *testing.T) {

	t.Parallel()

	t.Run("capability (ID)", func(t *testing.T) {

		t.Parallel()

		capabilityValueDeclaration := stdlib.StandardLibraryValue{
			Name: "cap",
			Type: &sema.CapabilityType{},
			Value: interpreter.NewUnmeteredCapabilityValue(
				4,
				interpreter.NewUnmeteredAddressValueFromBytes([]byte{0x1}),
				nil,
			),
			Kind: common.DeclarationKindConstant,
		}

		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(capabilityValueDeclaration)

		baseActivation := activations.NewActivation(nil, interpreter.BaseActivation)
		interpreter.Declare(baseActivation, capabilityValueDeclaration)

		inter, err := parseCheckAndPrepareWithOptions(t,
			`
              let maybeCapNonNil: Capability? = cap
              let maybeCapNil: Capability? = nil
              let res1 = maybeCapNonNil != nil
              let res2 = maybeCapNil == nil
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

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			inter.GetGlobal("res1"),
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			inter.GetGlobal("res2"),
		)
	})

	t.Run("function", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
		  fun func() {}

          let maybeFuncNonNil: (fun(): Void)? = func
          let maybeFuncNil: (fun(): Void)? = nil
          let res1 = maybeFuncNonNil != nil
          let res2 = maybeFuncNil == nil
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			inter.GetGlobal("res1"),
		)

		AssertValuesEqual(
			t,
			inter,
			interpreter.TrueValue,
			inter.GetGlobal("res2"),
		)
	})

	t.Run("nil", func(t *testing.T) {

		t.Parallel()

		inter := parseCheckAndPrepare(t, `
          let n: Int? = 1
          let res = nil == n
		`)

		AssertValuesEqual(
			t,
			inter,
			interpreter.FalseValue,
			inter.GetGlobal("res"),
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
			interpreter.PrimitiveStaticTypeWord128,
			interpreter.PrimitiveStaticTypeWord256,
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

					inter := parseCheckAndPrepare(t, code)

					result, err := inter.Invoke("test")

					switch op {
					case ast.OperationEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.FalseValue, result)
					case ast.OperationNotEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.TrueValue, result)
					default:
						assert.Failf(t, "unknown operation: %s", op.String())
					}
				})
			}
		}
	})

	t.Run("FixedSizeUnsignedInteger subtypes", func(t *testing.T) {
		t.Parallel()

		subtypes := []interpreter.StaticType{
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
			interpreter.PrimitiveStaticTypeWord128,
			interpreter.PrimitiveStaticTypeWord256,
		}

		for _, subtype := range subtypes {
			rhsType := interpreter.PrimitiveStaticTypeUInt8
			if subtype == rhsType {
				rhsType = interpreter.PrimitiveStaticTypeWord128
			}

			for _, op := range operations {
				t.Run(fmt.Sprintf("%s,%s", op.String(), subtype.String()), func(t *testing.T) {

					code := fmt.Sprintf(`
                        fun test(): Bool {
                            let x: FixedSizeUnsignedInteger = 5 as %s
                            let y: FixedSizeUnsignedInteger = 2 as %s
                            return x %s y
                        }`,
						subtype.String(),
						rhsType.String(),
						op.Symbol(),
					)

					inter := parseCheckAndPrepare(t, code)

					result, err := inter.Invoke("test")

					switch op {
					case ast.OperationEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.FalseValue, result)
					case ast.OperationNotEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.TrueValue, result)
					default:
						assert.Failf(t, "unknown operation: %s", op.String())
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

					inter := parseCheckAndPrepare(t, code)

					result, err := inter.Invoke("test")

					switch op {
					case ast.OperationEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.FalseValue, result)
					case ast.OperationNotEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.TrueValue, result)
					default:
						assert.Failf(t, "unknown operation: %s", op.String())
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

					inter := parseCheckAndPrepare(t, code)

					result, err := inter.Invoke("test")

					switch op {
					case ast.OperationEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.FalseValue, result)
					case ast.OperationNotEqual:
						require.NoError(t, err)
						assert.Equal(t, interpreter.TrueValue, result)
					default:
						assert.Failf(t, "unknown operation: %s", op.String())
					}
				})
			}
		}
	})
}
