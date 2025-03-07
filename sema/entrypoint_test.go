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

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/sema"
	. "github.com/onflow/cadence/test_utils/sema_utils"
)

func TestCheckEntryPointParameters(t *testing.T) {

	t.Parallel()

	t.Run("script, no parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            access(all) fun main() {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})

	t.Run("script, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            access(all) fun main(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Equal(t,
			[]sema.Parameter{
				{
					Label:          "",
					Identifier:     "a",
					TypeAnnotation: sema.IntTypeAnnotation,
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
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			parameters,
		)
	})

	t.Run("struct, script, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            access(all) struct SomeStruct {}

            access(all) fun main(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Equal(t,
			[]sema.Parameter{
				{
					Label:          "",
					Identifier:     "a",
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			parameters,
		)
	})

	t.Run("interface, script, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            access(all) struct interface SomeInterface {}

            access(all) fun main(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Equal(t,
			[]sema.Parameter{
				{
					Label:          "",
					Identifier:     "a",
					TypeAnnotation: sema.IntTypeAnnotation,
				},
			},
			parameters,
		)
	})

	t.Run("struct, transaction, one parameters", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            access(all) struct SomeStruct {}

            transaction(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})

	t.Run("transaction and script", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
            access(all) fun main(a: Int) {}

            transaction(a: Int) {}
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})

	t.Run("contract with init params", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
			access(all) contract SimpleContract {
				access(all) let v: Int
				init(a: Int) {
					self.v = a
				}
			}		
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

	t.Run("contract init empty", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
			access(all) contract SimpleContract {
				init() {}
			}		
        `)

		require.NoError(t, err)

		parameters := checker.EntryPointParameters()

		require.Empty(t, parameters)
	})
}
