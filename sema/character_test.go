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

func TestCheckCharacterLiteral(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
        let a: Character = "a"
    `)

	require.NoError(t, err)

	aType := RequireGlobalValue(t, checker.Elaboration, "a")

	assert.Equal(t,
		sema.CharacterType,
		aType,
	)
}

func TestCheckInvalidCharacterLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        let a: Character = "abc"
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidCharacterLiteralError{}, errs[0])
}

func TestCheckCharacterUtf8Field(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
		let a: Character = "a"
        let x = a.utf8
	`)

	require.NoError(t, err)

	assert.Equal(t,
		sema.ByteArrayType,
		RequireGlobalValue(t, checker.Elaboration, "x"),
	)
}
