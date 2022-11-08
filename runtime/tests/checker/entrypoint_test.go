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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestEntryPointParameters(t *testing.T) {

	t.Parallel()

	t.Run("script, no parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            pub fun main() {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})

	t.Run("script, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            pub fun main(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Equal(t,
			[]sema.Parameter{
				{
					Label:          "",
					Identifier:     "a",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			parameters,
		)
	})

	t.Run("transaction, no parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            transaction {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})

	t.Run("transaction, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            transaction(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Equal(t,
			[]sema.Parameter{
				{
					Label:          "",
					Identifier:     "a",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			parameters,
		)
	})

	t.Run("struct, script, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            pub struct SomeStruct {}

            pub fun main(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Equal(t,
			[]sema.Parameter{
				{
					Label:          "",
					Identifier:     "a",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			parameters,
		)
	})

	t.Run("interface, script, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            pub struct interface SomeInterface {}

            pub fun main(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Equal(t,
			[]sema.Parameter{
				{
					Label:          "",
					Identifier:     "a",
					TypeAnnotation: sema.NewTypeAnnotation(sema.IntType),
				},
			},
			parameters,
		)
	})

	t.Run("struct, transaction, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            pub struct SomeStruct {}

            transaction(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})

	t.Run("transaction and script", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            pub fun main(a: Int) {}

            transaction(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})
}
