/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
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

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ParseAndCheckWithLocation(t *testing.T, code string, location common.Location) (*sema.Checker, error) {
	return ParseAndCheckWithOptions(t, code, ParseAndCheckOptions{Location: location})
}

func RequireTypeErrors(t *testing.T, err error, targets ...interface{}) {
	require.Error(t, err)
	errs := ExpectCheckerErrors(t, err, len(targets))

	for i, e := range targets {
		require.ErrorAs(t, errs[i], &e)
	}
}

func TestScriptEnv(t *testing.T) {
	t.Parallel()

	t.Run("getAuthAccount valid", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithLocation(t,
			`
			let a = getAuthAccount(0x1234567)
			`,
			common.ScriptLocation([]byte{1, 2, 3, 4, 5, 6, 7}),
		)
		require.NoError(t, err)

		assert.Equal(t,
			sema.AuthAccountType,
			RequireGlobalValue(t, checker.Elaboration, "a"),
		)
	})

	t.Run("getAuthAccount invalid argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithLocation(t,
			`
			let a = getAuthAccount("")
			`,
			common.ScriptLocation([]byte{1, 2, 3, 4, 5, 6, 7}),
		)
		var typeMismatch *sema.TypeMismatchError
		RequireTypeErrors(t, err, typeMismatch)
	})

	t.Run("getAuthAccount missing argument", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithLocation(t,
			`
			let a = getAuthAccount()
			`,
			common.ScriptLocation([]byte{1, 2, 3, 4, 5, 6, 7}),
		)
		var argCount *sema.ArgumentCountError
		RequireTypeErrors(t, err, argCount)
	})

	t.Run("getAuthAccount too many args", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithLocation(t,
			`
			let a = getAuthAccount(0x1, 0x2)
			`,
			common.ScriptLocation([]byte{1, 2, 3, 4, 5, 6, 7}),
		)
		var argCount *sema.ArgumentCountError
		RequireTypeErrors(t, err, argCount)
	})
}

func TestTransactionEnv(t *testing.T) {
	t.Parallel()

	t.Run("getAuthAccount", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithLocation(t,
			`
			let a = getAuthAccount(0x1234567)
			`,
			common.TransactionLocation([]byte{1, 2, 3, 4, 5, 6, 7}),
		)
		var notDeclared *sema.NotDeclaredError
		RequireTypeErrors(t, err, notDeclared)
	})
}
