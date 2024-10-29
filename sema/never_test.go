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

package sema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/tests/sema_utils"
)

func TestCheckNever(t *testing.T) {

	t.Parallel()

	t.Run("never return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t,
			`
            fun test(): Int {
                return panic("XXX")
            }
        `,
		)

		require.NoError(t, err)
	})

	t.Run("numeric compatibility", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                fun test() {
                    var x: Never = 5
                }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchErr)

		assert.Equal(t, sema.NeverType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.IntType, typeMismatchErr.ActualType)
	})

	t.Run("character compatibility", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                fun test() {
                    var x: Never = "c"
                }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchErr)

		assert.Equal(t, sema.NeverType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})

	t.Run("string compatibility", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                fun test() {
                    var x: Never = "hello"
                }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var typeMismatchErr *sema.TypeMismatchError
		require.ErrorAs(t, errors[0], &typeMismatchErr)

		assert.Equal(t, sema.NeverType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})

	t.Run("binary op", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                fun test(a: Never, b: Never) {
                    var x: Int = a + b
                }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var binaryOpErr *sema.InvalidBinaryOperandsError
		require.ErrorAs(t, errors[0], &binaryOpErr)

		assert.Equal(t, sema.NeverType, binaryOpErr.LeftType)
		assert.Equal(t, sema.NeverType, binaryOpErr.RightType)
	})

	t.Run("unary op", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                fun test(a: Never) {
                    var x: Bool = !a
                }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var unaryOpErr *sema.InvalidUnaryOperandError
		require.ErrorAs(t, errors[0], &unaryOpErr)

		assert.Equal(t, sema.BoolType, unaryOpErr.ExpectedType)
		assert.Equal(t, sema.NeverType, unaryOpErr.ActualType)
	})

	t.Run("nil-coalescing", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                fun test(a: Never?) {
                    var x: Int = a ?? 4
                }
            `,
		)

		assert.NoError(t, err)
	})

	t.Run("enum raw type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                enum Foo: Never {}
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		var invalidEnumRawTypeErr *sema.InvalidEnumRawTypeError
		require.ErrorAs(t, errors[0], &invalidEnumRawTypeErr)

		assert.Equal(t, sema.NeverType, invalidEnumRawTypeErr.Type)
	})

	t.Run("tx prepare arg", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                transaction {
                    prepare(acct: Never) {}
                }
            `,
		)
		// Useless, but not an error
		require.NoError(t, err)
	})
}
