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

func TestCheckIncompleteDictionaryType(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithOptions(t,
		`
          let dict: {Int:} = {}
        `,
		ParseAndCheckOptions{
			IgnoreParseError: true,
		},
	)

	require.NoError(t, err)

	assert.Equal(t,
		&sema.DictionaryType{
			KeyType:   sema.IntType,
			ValueType: sema.InvalidType,
		},
		RequireGlobalValue(t, checker.Elaboration, "dict"),
	)
}

func TestCheckMetaKeyType(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t,
		`
		let dict = {Type<Int>(): "a"}
        `,
	)

	require.NoError(t, err)

	assert.Equal(t,
		&sema.DictionaryType{
			KeyType:   sema.MetaType,
			ValueType: sema.StringType,
		},
		RequireGlobalValue(t, checker.Elaboration, "dict"),
	)
}
