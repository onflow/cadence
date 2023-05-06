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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckToString(t *testing.T) {

	t.Parallel()

	for _, numberOrAddressType := range append(
		sema.AllNumberTypes[:],
		sema.TheAddressType,
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
	runInvalidCase(t, "[1], [2, 3, 4]", &sema.ArgumentCountError{})
	runInvalidCase(t, "", &sema.ArgumentCountError{})
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

			assert.Equal(t,
				sema.TheAddressType,
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

	runValidCase(t, "\"1\"")
	runValidCase(t, "\"ab\"")
	runValidCase(t, "\"0x1\"")
	runValidCase(t, "\"0x436164656E636521\"")

	runInvalidCase(t, "[1232]", &sema.TypeMismatchError{})
	runInvalidCase(t, "1", &sema.TypeMismatchError{})
	runInvalidCase(t, "\"0x1\", \"0x2\"", &sema.ArgumentCountError{})
	runInvalidCase(t, "", &sema.ArgumentCountError{})
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
