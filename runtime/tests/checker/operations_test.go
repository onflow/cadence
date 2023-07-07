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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckInvalidUnaryBooleanNegationOfInteger(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let a = !1
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidUnaryOperandError{}, errs[0])
}

func TestCheckUnaryBooleanNegation(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let a = !true
	    `)

		require.NoError(t, err)
	})

	t.Run("invalid type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let a = !x
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

}

func TestCheckInvalidUnaryIntegerNegationOfBoolean(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let a = -true
	`)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidUnaryOperandError{}, errs[0])
}

func TestCheckUnaryIntegerNegation(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let a = -1
	    `)

		require.NoError(t, err)
	})

	t.Run("invalid type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let a = -x
	    `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

}

type operationTest struct {
	ty             sema.Type
	left, right    string
	expectedErrors []error
}

type operationTests struct {
	operations []ast.Operation
	tests      []operationTest
}

func TestCheckIntegerBinaryOperations(t *testing.T) {

	t.Parallel()

	allOperationTests := []operationTests{
		{
			operations: []ast.Operation{
				ast.OperationPlus,
				ast.OperationMinus,
				ast.OperationMod,
				ast.OperationMul,
				ast.OperationDiv,
			},
			tests: []operationTest{
				{sema.IntType, "1", "2", nil},
				{sema.UFix64Type, "1.2", "3.4", nil},
				{sema.Fix64Type, "-1.2", "-3.4", nil},
				{sema.UFix64Type, "1.2", "3", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.IntType, "1", "2.3", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.IntType, "true", "2", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
					&sema.TypeMismatchError{},
				}},
				{sema.Fix64Type, "true", "1.2", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
					&sema.TypeMismatchError{},
				}},
				{sema.IntType, "1", "true", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.UFix64Type, "1.2", "true", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.IntType, "true", "false", []error{
					&sema.InvalidBinaryOperandsError{},
					&sema.TypeMismatchError{},
				}},
			},
		},
		{
			operations: []ast.Operation{
				ast.OperationLess,
				ast.OperationLessEqual,
				ast.OperationGreater,
				ast.OperationGreaterEqual,
			},
			tests: []operationTest{
				{sema.BoolType, "1", "2", nil},
				{sema.BoolType, "1.2", "3.4", nil},
				{sema.BoolType, "true", "2", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1.2", "3", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1", "2.3", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "true", "1.2", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1", "true", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1.2", "true", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "true", "false", nil},
			},
		},
		{
			operations: []ast.Operation{
				ast.OperationOr,
				ast.OperationAnd,
			},
			tests: []operationTest{
				{sema.BoolType, "true", "false", nil},
				{sema.BoolType, "true", "2", []error{
					&sema.InvalidBinaryOperandError{},
				}},
				{sema.BoolType, "1", "true", []error{
					&sema.InvalidBinaryOperandError{},
				}},
				{sema.BoolType, "1", "2", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
			},
		},
		{
			operations: []ast.Operation{
				ast.OperationEqual,
				ast.OperationNotEqual,
			},
			tests: []operationTest{
				{sema.BoolType, "true", "false", nil},
				{sema.BoolType, "1", "2", nil},
				{sema.BoolType, "1.2", "3.4", nil},
				{sema.BoolType, "true", "2", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1.2", "3", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1", "2.3", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1", "true", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, `"test"`, `"test"`, nil},
			},
		},
		{
			operations: []ast.Operation{
				ast.OperationBitwiseOr,
				ast.OperationBitwiseXor,
				ast.OperationBitwiseAnd,
				ast.OperationBitwiseLeftShift,
				ast.OperationBitwiseRightShift,
			},
			tests: []operationTest{
				{sema.IntType, "1", "2", nil},
				{sema.UFix64Type, "1.2", "3.4", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.UFix64Type, "1.2", "3", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.IntType, "1", "2.3", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.IntType, "true", "2", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
					&sema.TypeMismatchError{},
				}},
				{sema.UFix64Type, "true", "1.2", []error{
					&sema.InvalidBinaryOperandsError{},
					&sema.TypeMismatchError{},
				}},
				{sema.IntType, "1", "true", []error{
					&sema.InvalidBinaryOperandError{},
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.UFix64Type, "1.2", "true", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.IntType, "true", "false", []error{
					&sema.InvalidBinaryOperandsError{},
					&sema.TypeMismatchError{},
				}},
			},
		},
	}

	for _, operationTests := range allOperationTests {
		for _, operation := range operationTests.operations {
			for _, test := range operationTests.tests {

				testName := fmt.Sprintf(
					"%s / %s %s %s",
					test.ty, test.left, operation.Symbol(), test.right,
				)

				t.Run(testName, func(t *testing.T) {

					_, err := ParseAndCheck(t,
						fmt.Sprintf(
							`fun test(): %s { return %s %s %s }`,
							test.ty, test.left, operation.Symbol(), test.right,
						),
					)

					errs := RequireCheckerErrors(t, err, len(test.expectedErrors))

					for i, expectedErr := range test.expectedErrors {
						assert.IsType(t, expectedErr, errs[i])
					}
				})
			}
		}
	}
}

type operationWithTypeTest struct {
	ty                  sema.Type
	left, right         string
	leftType, rightType string
	expectedErrors      []error
}

type operationWithTypeTests struct {
	operations []ast.Operation
	tests      []operationWithTypeTest
}

func TestCheckNonIntegerComparisonOperations(t *testing.T) {
	t.Parallel()

	allOperationTests := []operationWithTypeTests{
		{
			operations: []ast.Operation{
				ast.OperationLess,
				ast.OperationLessEqual,
				ast.OperationGreater,
				ast.OperationGreaterEqual,
			},
			tests: []operationWithTypeTest{
				{sema.BoolType, "false", "true", "Bool", "Bool", nil},
				{sema.BoolType, "false", "1", "Bool", "Int", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "\"a\"", "\"b\"", "Character", "Character", nil},
				{sema.BoolType, "1.2", "\"b\"", "Fix64", "Character", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "\"\"", "\"\"", "String", "String", nil},
				{sema.BoolType, "\"\"", "\"b\"", "String", "String", nil},
				{sema.BoolType, "\"xyz\"", "\"b\"", "String", "String", nil},
				{sema.BoolType, "\"x\"", "\"b\"", "Character", "String", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
				{sema.BoolType, "1.2", "\"bcd\"", "Fix64", "String", []error{
					&sema.InvalidBinaryOperandsError{},
				}},
			},
		},
	}

	for _, operationTests := range allOperationTests {
		for _, operation := range operationTests.operations {
			for _, test := range operationTests.tests {

				testName := fmt.Sprintf(
					"%s / %s %s %s",
					test.ty, test.left, operation.Symbol(), test.right,
				)

				t.Run(testName, func(t *testing.T) {

					_, err := ParseAndCheck(t,
						fmt.Sprintf(
							`fun test(): %s { 
								let a: %s = %s
								let b: %s = %s
								return a %s b 
							}`,
							test.ty, test.leftType, test.left, test.rightType, test.right, operation.Symbol(),
						),
					)

					errs := RequireCheckerErrors(t, err, len(test.expectedErrors))

					for i, expectedErr := range test.expectedErrors {
						assert.IsType(t, expectedErr, errs[i])
					}
				})
			}
		}
	}
}

func TestCheckSaturatedArithmeticFunctions(t *testing.T) {

	t.Parallel()

	type testCase struct {
		ty                              sema.Type
		add, subtract, multiply, divide bool
	}

	testCases := []testCase{
		{
			ty:       sema.IntType,
			add:      false,
			subtract: false,
			multiply: false,
			divide:   false,
		},
		{
			ty:       sema.UIntType,
			add:      false,
			subtract: true,
			multiply: false,
			divide:   false,
		},
	}

	for _, ty := range common.Concat(
		sema.AllSignedIntegerTypes,
		sema.AllSignedFixedPointTypes,
	) {

		if ty == sema.IntType {
			continue
		}

		testCases = append(testCases, testCase{
			ty:       ty,
			add:      true,
			subtract: true,
			multiply: true,
			divide:   true,
		})
	}

	for _, ty := range common.Concat(
		sema.AllUnsignedIntegerTypes,
		sema.AllUnsignedFixedPointTypes,
	) {

		if ty == sema.UIntType || strings.HasPrefix(ty.String(), "Word") {
			continue
		}

		testCases = append(testCases, testCase{
			ty:       ty,
			add:      true,
			subtract: true,
			multiply: true,
			divide:   false,
		})
	}

	test := func(ty sema.Type, method string, expected bool) {

		method = fmt.Sprintf("saturating%s", method)

		t.Run(fmt.Sprintf("%s %s", ty, method), func(t *testing.T) {

			_, err := ParseAndCheckWithPanic(t,
				fmt.Sprintf(
					`
                      fun test(a: %[1]s, b: %[1]s): %[1]s {
                          return a.%[2]s(b)
                      }
                    `,
					ty,
					method,
				),
			)

			if expected {
				require.NoError(t, err)
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
			}
		})
	}

	for _, testCase := range testCases {
		test(testCase.ty, "Add", testCase.add)
		test(testCase.ty, "Subtract", testCase.subtract)
		test(testCase.ty, "Multiply", testCase.multiply)
		test(testCase.ty, "Divide", testCase.divide)
	}
}

func TestCheckInvalidCompositeEquality(t *testing.T) {

	t.Parallel()

	test := func(compositeKind common.CompositeKind) {

		t.Run(compositeKind.Name(), func(t *testing.T) {

			t.Parallel()

			var preparationCode, firstIdentifier, secondIdentifier string
			if compositeKind == common.CompositeKindContract {
				firstIdentifier = "X"
				secondIdentifier = "X"
			} else {
				preparationCode = fmt.Sprintf(`
		              let x1: %[1]sX %[2]s %[3]s X%[4]s
                      let x2: %[1]sX %[2]s %[3]s X%[4]s
		            `,
					compositeKind.Annotation(),
					compositeKind.TransferOperator(),
					compositeKind.ConstructionKeyword(),
					constructorArguments(compositeKind),
				)
				firstIdentifier = "x1"
				secondIdentifier = "x2"
			}

			body := "{}"
			switch compositeKind {
			case common.CompositeKindEvent:
				body = "()"
			case common.CompositeKindEnum:
				body = "{ case a }"
			}

			conformances := ""
			if compositeKind == common.CompositeKindEnum {
				conformances = ": Int"
			}

			var baseType string
			if compositeKind == common.CompositeKindAttachment {
				baseType = "for AnyStruct"
			}

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s X %[7]s %[2]s %[3]s

                      %[4]s

                      let a = %[5]s == %[6]s
                    `,
					compositeKind.Keyword(),
					conformances,
					body,
					preparationCode,
					firstIdentifier,
					secondIdentifier,
					baseType,
				),
			)

			if compositeKind == common.CompositeKindEnum {
				require.NoError(t, err)
			} else if compositeKind == common.CompositeKindAttachment {
				errs := RequireCheckerErrors(t, err, 5)

				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[0])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[1])
				assert.IsType(t, &sema.InvalidAttachmentAnnotationError{}, errs[2])
				assert.IsType(t, &sema.InvalidAttachmentUsageError{}, errs[3])
				assert.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[4])
			} else {
				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
			}
		})
	}

	for _, compositeKind := range common.AllCompositeKinds {

		if compositeKind == common.CompositeKindEvent {
			continue
		}

		test(compositeKind)
	}
}

func TestCheckNumericSuperTypeBinaryOperations(t *testing.T) {

	t.Parallel()

	supertypes := []sema.Type{
		sema.NumberType,
		sema.SignedNumberType,
		sema.IntegerType,
		sema.SignedIntegerType,
		sema.FixedPointType,
		sema.SignedFixedPointType,
	}

	t.Run("non saturating operations", func(t *testing.T) {

		t.Parallel()

		operations := []ast.Operation{
			ast.OperationPlus,
			ast.OperationMinus,
			ast.OperationMul,
			ast.OperationDiv,
			ast.OperationMod,
			ast.OperationBitwiseAnd,
			ast.OperationBitwiseOr,
			ast.OperationBitwiseXor,
			ast.OperationBitwiseRightShift,
			ast.OperationBitwiseLeftShift,
			ast.OperationLess,
			ast.OperationLessEqual,
			ast.OperationGreater,
			ast.OperationGreaterEqual,
		}

		for _, supertype := range supertypes {
			for _, op := range operations {
				t.Run(fmt.Sprintf("%s,%s", supertype.String(), op.String()), func(t *testing.T) {
					code := fmt.Sprintf(`
                        fun test(a: %[1]s, b: %[1]s): AnyStruct {
                            return a %[2]s b
                        }`,
						supertype.String(),
						op.Symbol(),
					)

					_, err := ParseAndCheck(t, code)
					errs := RequireCheckerErrors(t, err, 1)
					assert.IsType(t, &sema.InvalidBinaryOperandsError{}, errs[0])
				})
			}
		}
	})

	t.Run("saturating operations", func(t *testing.T) {
		t.Parallel()

		saturatingFunctions := []string{
			sema.NumericTypeSaturatingAddFunctionName,
			sema.NumericTypeSaturatingSubtractFunctionName,
			sema.NumericTypeSaturatingMultiplyFunctionName,
			sema.NumericTypeSaturatingMultiplyFunctionName,
			sema.NumericTypeSaturatingDivideFunctionName,
		}

		for _, supertype := range supertypes {
			for _, saturatingFunc := range saturatingFunctions {
				t.Run(fmt.Sprintf("%s,%s", supertype.String(), saturatingFunc), func(t *testing.T) {
					code := fmt.Sprintf(`
                        fun test(a: %[1]s, b: %[1]s): AnyStruct {
                            return a.%[2]s(b)
                        }`,
						supertype.String(),
						saturatingFunc,
					)

					_, err := ParseAndCheck(t, code)
					errs := RequireCheckerErrors(t, err, 1)
					assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
				})
			}
		}
	})
}
