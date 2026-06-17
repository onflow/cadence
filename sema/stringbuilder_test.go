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
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckStringBuilder(t *testing.T) {

	t.Parallel()

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			let sb = StringBuilder()
		`)

		require.NoError(t, err)

		assert.Equal(t,
			sema.StringBuilderType,
			RequireGlobalValue(t, checker.Elaboration, "sb"),
		)
	})

	t.Run("constructor with invalid argument", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			let sb = StringBuilder("hello")
		`)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ExcessiveArgumentsError{}, errs[0])
	})

	t.Run("append valid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.append("hello")
			}
		`)

		require.NoError(t, err)
	})

	t.Run("append invalid argument type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.append(42)
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("append missing argument", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.append()
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InsufficientArgumentsError{}, errs[0])
	})

	t.Run("appendCharacter valid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				let c: Character = "a"
				sb.appendCharacter(c)
			}
		`)

		require.NoError(t, err)
	})

	t.Run("appendCharacter invalid argument type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.appendCharacter(42)
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("appendCharacter string instead of character", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.appendCharacter("hello")
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.InvalidCharacterLiteralError{}, errs[0])
	})

	t.Run("clear valid", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.clear()
			}
		`)

		require.NoError(t, err)
	})

	t.Run("clear unexpected argument", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.clear("hello")
			}
		`)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.ExcessiveArgumentsError{}, errs[0])
	})

	t.Run("toString valid", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			fun test(): String {
				let sb = StringBuilder()
				sb.append("hello")
				return sb.toString()
			}
		`)

		require.NoError(t, err)

		funcType := RequireGlobalValue(t, checker.Elaboration, "test")
		assert.Equal(t,
			sema.StringType,
			funcType.(*sema.FunctionType).ReturnTypeAnnotation.Type,
		)
	})

	t.Run("toString is view", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			view fun test(): String {
				let sb = StringBuilder()
				return sb.toString()
			}
		`)

		require.NoError(t, err)
	})

	t.Run("length valid", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheck(t, `
			let sb = StringBuilder()
			let l = sb.length
		`)

		require.NoError(t, err)

		assert.Equal(t,
			sema.IntType,
			RequireGlobalValue(t, checker.Elaboration, "l"),
		)
	})

	t.Run("length not assignable", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun test() {
				let sb = StringBuilder()
				sb.length = 5
			}
		`)

		errs := RequireCheckerErrors(t, err, 2)
		assert.IsType(t, &sema.InvalidAssignmentAccessError{}, errs[0])
		assert.IsType(t, &sema.AssignmentToConstantMemberError{}, errs[1])
	})

	t.Run("used in function", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
			fun build(_ parts: [String]): String {
				let sb = StringBuilder()
				for part in parts {
					sb.append(part)
				}
				return sb.toString()
			}
		`)

		require.NoError(t, err)
	})
}
