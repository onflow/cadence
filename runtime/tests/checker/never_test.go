/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckNever(t *testing.T) {

	t.Parallel()

	t.Run("never return", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t,
			`
            pub fun test(): Int {
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
                pub fun test() {
                    var x: Never = 5
                }
            `,
		)

		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		typeMismatchErr := errors[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.NeverType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.IntType, typeMismatchErr.ActualType)
	})

	t.Run("character compatibility", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                pub fun test() {
                    var x: Never = "c"
                }
            `,
		)

		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		typeMismatchErr := errors[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.NeverType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})

	t.Run("string compatibility", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                pub fun test() {
                    var x: Never = "hello"
                }
            `,
		)

		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errors[0])
		typeMismatchErr := errors[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.NeverType, typeMismatchErr.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchErr.ActualType)
	})

	t.Run("binary op", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                pub fun test(a: Never, b: Never) {
                    var x: Int = a + b
                }
            `,
		)

		errors := ExpectCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidBinaryOperandsError{}, errors[0])
		binaryOpErr := errors[0].(*sema.InvalidBinaryOperandsError)

		assert.Equal(t, sema.NeverType, binaryOpErr.LeftType)
		assert.Equal(t, sema.NeverType, binaryOpErr.RightType)
	})

	t.Run("unary op", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                pub fun test(a: Never) {
                    var x: Bool = !a
                }
            `,
		)

		errors := ExpectCheckerErrors(t, err, 1)
		require.IsType(t, &sema.InvalidUnaryOperandError{}, errors[0])
		unaryOpErr := errors[0].(*sema.InvalidUnaryOperandError)

		assert.Equal(t, sema.BoolType, unaryOpErr.ExpectedType)
		assert.Equal(t, sema.NeverType, unaryOpErr.ActualType)
	})

	t.Run("nil-coalescing", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t,
			`
                pub fun test(a: Never?) {
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
                enum Foo: Never {
                }
            `,
		)

		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidEnumRawTypeError{}, errors[0])
		typeMismatchErr := errors[0].(*sema.InvalidEnumRawTypeError)

		assert.Equal(t, sema.NeverType, typeMismatchErr.Type)
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

		errors := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.InvalidTransactionPrepareParameterTypeError{}, errors[0])
		typeMismatchErr := errors[0].(*sema.InvalidTransactionPrepareParameterTypeError)

		assert.Equal(t, sema.NeverType, typeMismatchErr.Type)
	})
}
