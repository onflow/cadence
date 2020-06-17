/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2020 Dapper Labs, Inc.
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

func TestCheckMetaType(t *testing.T) {

	t.Parallel()

	t.Run("constructor", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let type: Type = Type<[Int]>()
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.MetaType{},
			checker.GlobalValues["type"].Type,
		)
	})

	t.Run("identifier", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let type = Type<[Int]>()
          let identifier = type.identifier
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.MetaType{},
			checker.GlobalValues["type"].Type,
		)
	})
}

func TestCheckIsInstance(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		code  string
		valid bool
	}{
		"string is an instance of string": {
			`
          let stringType = Type<String>()
          let result = "abc".isInstance(stringType)
			`,
			true,
		},
		"int is an instance of int": {

			`
          let intType = Type<Int>()
          let result = (1).isInstance(intType)
			`,
			true,
		},
		"resource is an instance of resource": {
			`
          resource R {}

          let r <- create R()
          let rType = Type<@R>()
          let result = r.isInstance(rType)
			`,
			true,
		},
		"1 is an instance of Int?": {
			`
				let result = (1).isInstance(Type<Int?>())
			`,
			true,
		},
		"isInstance must take a type": {
			`
				let result = (1).isInstance(3)
			`,
			false,
		},
		"nil is not a type": {
			`
				let result = (1).isInstance(nil)
			`,
			false,
		},
	}

	for name, cases := range cases {
		t.Run(name, func(t *testing.T) {
			checker, err := ParseAndCheck(t, cases.code)
			if cases.valid {
				require.NoError(t, err)
				assert.Equal(t,
					&sema.BoolType{},
					checker.GlobalValues["result"].Type,
				)
			} else {
				require.Error(t, err)
			}
		})
	}

}

func TestCheckIsInstance_Redeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct R {
          fun isInstance() {}
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.InvalidDeclarationError{}, errs[0])
}
