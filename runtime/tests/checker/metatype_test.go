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

func TestCheckIsInstance_Use(t *testing.T) {

	t.Parallel()

	t.Run("String", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let stringType = Type<String>()
          let result = "abc".isInstance(stringType)
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.BoolType{},
			checker.GlobalValues["result"].Type,
		)
	})

	t.Run("Int", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let intType = Type<Int>()
          let result = (1).isInstance(intType)
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.BoolType{},
			checker.GlobalValues["result"].Type,
		)
	})

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          resource R {}

          let r <- create R()
          let rType = Type<@R>()
          let result = r.isInstance(rType)
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.BoolType{},
			checker.GlobalValues["result"].Type,
		)
	})
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
