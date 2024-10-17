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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/common"
	"github.com/onflow/cadence/sema"
	"github.com/onflow/cadence/stdlib"
)

func TestCheckToString(t *testing.T) {

	t.Parallel()

	for _, numberOrAddressType := range common.Concat(
		sema.AllNumberTypes,
		[]sema.Type{
			sema.TheAddressType,
		},
	) {

		ty := numberOrAddressType

		t.Run(ty.String(), func(t *testing.T) {

			t.Parallel()

			checker, err := parseAndCheckWithTestValue(t,
				`
                  let res = test.toString()
                `,
				ty,
			)

			require.NoError(t, err)

			resType := RequireGlobalValue(t, checker.Elaboration, "res")

			assert.Equal(t,
				sema.StringType,
				resType,
			)
		})
	}
}

func TestCheckToBytes(t *testing.T) {

	t.Parallel()

	t.Run("Address", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let address: Address = 0x1
          let res = address.toBytes()
        `)

		require.NoError(t, err)

		resType := RequireGlobalValue(t, checker.Elaboration, "res")

		assert.Equal(t,
			sema.ByteArrayType,
			resType,
		)
	})
}

func TestCheckAddressFromBytes(t *testing.T) {
	t.Parallel()

	runValidCase := func(t *testing.T, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let address = Address.fromBytes(%s)", innerCode)

			checker, err := ParseAndCheck(t, code)

			require.NoError(t, err)

			resType := RequireGlobalValue(t, checker.Elaboration, "address")

			assert.Equal(t,
				sema.TheAddressType,
				resType,
			)
		})
	}

	runInvalidCase := func(t *testing.T, innerCode string, expectedErrorType sema.SemanticError) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let address = Address.fromBytes(%s)", innerCode)

			_, err := ParseAndCheck(t, code)

			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, expectedErrorType, errs[0])
		})
	}

	runValidCase(t, "[1]")
	runValidCase(t, "[12, 34, 56]")
	runValidCase(t, "[67, 97, 100, 101, 110, 99, 101, 33]")

	runInvalidCase(t, "[\"abc\"]", &sema.TypeMismatchError{})
	runInvalidCase(t, "1", &sema.TypeMismatchError{})
	runInvalidCase(t, "[1], [2, 3, 4]", &sema.ExcessiveArgumentsError{})
	runInvalidCase(t, "", &sema.InsufficientArgumentsError{})
	runInvalidCase(t, "typo: [1]", &sema.IncorrectArgumentLabelError{})
}

func TestCheckAddressFromString(t *testing.T) {
	t.Parallel()

	runValidCase := func(t *testing.T, innerCode string) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let address = Address.fromString(%s)", innerCode)

			checker, err := ParseAndCheck(t, code)

			require.NoError(t, err)

			resType := RequireGlobalValue(t, checker.Elaboration, "address")
			require.Equal(t,
				&sema.OptionalType{
					Type: sema.TheAddressType,
				},
				resType,
			)
		})
	}

	runInvalidCase := func(t *testing.T, innerCode string, expectedErrorType sema.SemanticError) {
		t.Run(innerCode, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let address = Address.fromString(%s)", innerCode)

			_, err := ParseAndCheck(t, code)

			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, expectedErrorType, errs[0])
		})
	}

	runValidCase(t, "\"0x1\"")
	runValidCase(t, "\"0x436164656E636521\"")

	// While these inputs will return Nil, for the checker these are valid inputs.
	runValidCase(t, "\"1\"")
	runValidCase(t, "\"ab\"")

	runInvalidCase(t, "[1232]", &sema.TypeMismatchError{})
	runInvalidCase(t, "1", &sema.TypeMismatchError{})
	runInvalidCase(t, "\"0x1\", \"0x2\"", &sema.ExcessiveArgumentsError{})
	runInvalidCase(t, "", &sema.InsufficientArgumentsError{})
	runInvalidCase(t, "typo: \"0x1\"", &sema.IncorrectArgumentLabelError{})
}

func TestCheckToBigEndianBytes(t *testing.T) {

	t.Parallel()

	for _, ty := range sema.AllNumberTypes {

		t.Run(ty.String(), func(t *testing.T) {

			checker, err := parseAndCheckWithTestValue(t,
				`
                  let res = test.toBigEndianBytes()
                `,
				ty,
			)

			require.NoError(t, err)

			resType := RequireGlobalValue(t, checker.Elaboration, "res")

			assert.Equal(t,
				sema.ByteArrayType,
				resType,
			)
		})
	}
}

func TestCheckFromBigEndianBytes(t *testing.T) {

	t.Parallel()

	runValidCase := func(t *testing.T, ty sema.Type, bytesString string) {
		t.Run(bytesString, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let res = %s.fromBigEndianBytes(%s)", ty, bytesString)

			checker, err := ParseAndCheck(t, code)

			require.NoError(t, err)

			resType := RequireGlobalValue(t, checker.Elaboration, "res")
			require.Equal(t,
				&sema.OptionalType{
					Type: ty,
				},
				resType,
			)
		})
	}

	runInvalidCase := func(t *testing.T, ty sema.Type, bytesString string, expectedErrorType sema.SemanticError) {
		t.Run(bytesString, func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let address = %s.fromBigEndianBytes(%s)", ty, bytesString)

			_, err := ParseAndCheck(t, code)

			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, expectedErrorType, errs[0])
		})
	}

	for _, ty := range sema.AllNumberTypes {
		switch ty {
		case sema.NumberType, sema.SignedNumberType,
			sema.IntegerType, sema.SignedIntegerType, sema.FixedSizeUnsignedIntegerType,
			sema.FixedPointType, sema.SignedFixedPointType:
			continue

		default:
			runValidCase(t, ty, "[]")
			runValidCase(t, ty, "[1]")
			runValidCase(t, ty, "[1, 2, 100, 4, 45, 12]")

			runInvalidCase(t, ty, "\"abcd\"", &sema.TypeMismatchError{})
			runInvalidCase(t, ty, "", &sema.InsufficientArgumentsError{})
			runInvalidCase(t, ty, "[1], [2, 4]", &sema.ExcessiveArgumentsError{})
			runInvalidCase(t, ty, "typo: [1]", &sema.IncorrectArgumentLabelError{})
		}
	}
}

type testRandomGenerator struct{}

func (*testRandomGenerator) ReadRandom([]byte) error {
	return nil
}

func TestCheckRevertibleRandom(t *testing.T) {

	t.Parallel()

	newOptions := func() ParseAndCheckOptions {
		baseValueActivation := sema.NewVariableActivation(sema.BaseValueActivation)
		baseValueActivation.DeclareValue(stdlib.NewRevertibleRandomFunction(&testRandomGenerator{}))
		return ParseAndCheckOptions{
			Config: &sema.Config{
				BaseValueActivationHandler: func(_ common.Location) *sema.VariableActivation {
					return baseValueActivation
				},
			},
		}
	}

	runValidCase := func(t *testing.T, ty sema.Type, code string) {

		checker, err := ParseAndCheckWithOptions(t, code, newOptions())

		require.NoError(t, err)

		resType := RequireGlobalValue(t, checker.Elaboration, "rand")
		require.Equal(t, ty, resType)
	}

	runValidCaseWithoutModulo := func(t *testing.T, ty sema.Type) {
		t.Run(fmt.Sprintf("revertibleRandom<%s>, no modulo", ty), func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let rand = revertibleRandom<%s>()", ty)
			runValidCase(t, ty, code)
		})
	}

	runValidCaseWithModulo := func(t *testing.T, ty sema.Type) {
		t.Run(fmt.Sprintf("revertibleRandom<%s>, modulo", ty), func(t *testing.T) {
			t.Parallel()

			code := fmt.Sprintf("let rand = revertibleRandom<%[1]s>(modulo: %[1]s(1))", ty)
			runValidCase(t, ty, code)
		})
	}

	runInvalidCase := func(t *testing.T, testName string, code string, expectedErrors []error) {
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheckWithOptions(t, code, newOptions())

			errs := RequireCheckerErrors(t, err, len(expectedErrors))
			for i := range expectedErrors {
				assert.IsType(t, expectedErrors[i], errs[i])
			}
		})
	}

	for _, ty := range sema.AllFixedSizeUnsignedIntegerTypes {
		switch ty {
		case sema.FixedSizeUnsignedIntegerType:
			continue

		default:
			runValidCaseWithoutModulo(t, ty)
			runValidCaseWithModulo(t, ty)
		}
	}

	runInvalidCase(
		t,
		"revertibleRandom<Int>",
		"let rand = revertibleRandom<Int>()",
		[]error{
			&sema.TypeMismatchError{},
		},
	)

	runInvalidCase(
		t,
		"revertibleRandom<String>",
		`let rand = revertibleRandom<String>(modulo: "abcd")`,
		[]error{
			&sema.TypeMismatchError{},
		},
	)

	runInvalidCase(
		t,
		"missing_argument_label",
		"let rand = revertibleRandom<UInt256>(UInt256(1))",
		[]error{
			&sema.MissingArgumentLabelError{},
		},
	)

	runInvalidCase(
		t,
		"incorrect_argument_label",
		"let rand = revertibleRandom<UInt256>(typo: UInt256(1))",
		[]error{
			&sema.IncorrectArgumentLabelError{},
		},
	)

	runInvalidCase(
		t,
		"too_many_args",
		"let rand = revertibleRandom<UInt256>(modulo: UInt256(1), 2, 3)",
		[]error{
			&sema.ExcessiveArgumentsError{},
		},
	)

	runInvalidCase(
		t,
		"modulo type mismatch",
		"let rand = revertibleRandom<UInt256>(modulo: UInt128(1))",
		[]error{
			&sema.TypeMismatchError{},
		},
	)

	runInvalidCase(
		t,
		"string modulo",
		`let rand = revertibleRandom<UInt256>(modulo: "abcd")`,
		[]error{
			&sema.TypeMismatchError{},
		},
	)

	runInvalidCase(
		t,
		"invalid type argument Never",
		`let rand = revertibleRandom<Never>(modulo: 1)`,
		[]error{
			&sema.TypeMismatchError{},
			&sema.InvalidTypeArgumentError{},
		},
	)
	runInvalidCase(
		t,
		"invalid type argument FixedSizeUnsignedInteger",
		`let rand = revertibleRandom<FixedSizeUnsignedInteger>(modulo: 1)`,
		[]error{
			&sema.InvalidTypeArgumentError{},
		},
	)

	runInvalidCase(
		t,
		"missing type argument",
		`let rand = revertibleRandom()`,
		[]error{
			&sema.TypeParameterTypeInferenceError{},
		},
	)

	t.Run("type parameter used for argument", func(t *testing.T) {
		t.Parallel()

		runValidCase(
			t,
			sema.UInt256Type,
			"let rand = revertibleRandom<UInt256>(modulo: 1)",
		)
	})
}
