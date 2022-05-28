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

func TestCheckConditionalExpressionTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = true ? 1 : 2
      }
	`)

	require.NoError(t, err)
}

func TestCheckInvalidConditionalExpressionTest(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test() {
          let x = 1 ? 2 : 3
      }
	`)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidConditionalExpressionElse(t *testing.T) {

	t.Parallel()

	t.Run("undeclared variable", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? 2 : y
	    `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(t, sema.InvalidType, xType)
	})

	t.Run("mismatching type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            let x: Int8 = true ? 2 : "hello"
	    `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		typeMismatchError := errs[0].(*sema.TypeMismatchError)

		assert.Equal(t, sema.Int8Type, typeMismatchError.ExpectedType)
		assert.Equal(t, sema.StringType, typeMismatchError.ActualType)
	})
}

func TestCheckConditionalExpressionTypeInferring(t *testing.T) {

	t.Parallel()

	t.Run("different simple types", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? 2 : false
        `)

		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(t, sema.AnyStructType, xType)
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? 1 : nil
        `)

		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(
			t,
			&sema.OptionalType{
				Type: sema.IntType,
			},
			xType,
		)
	})

	t.Run("chained optional", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
            let x = true ? Int8(1) : (false ? Int(5) : nil)
        `)

		require.NoError(t, err)

		xType := RequireGlobalValue(t, checker.Elaboration, "x")
		assert.Equal(
			t,
			&sema.OptionalType{
				Type: sema.SignedIntegerType,
			},
			xType,
		)
	})
}
