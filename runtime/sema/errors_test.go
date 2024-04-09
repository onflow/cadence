/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Dapper Labs, Inc.
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

package sema

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/onflow/cadence/runtime/parser"
)

func TestErrorMessageExpectedActualTypes(t *testing.T) {

	t.Parallel()

	t.Run("qualified strings are different", func(t *testing.T) {

		t.Parallel()

		expected, actual := ErrorMessageExpectedActualTypes(
			&SimpleType{
				QualifiedName: "Foo",
				TypeID:        "Foo",
			},
			&SimpleType{
				QualifiedName: "Bar",
				TypeID:        "Bar",
			},
		)

		assert.Equal(t, "Foo", expected)
		assert.Equal(t, "Bar", actual)

	})

	t.Run("qualified strings are the same", func(t *testing.T) {

		t.Parallel()

		expected, actual := ErrorMessageExpectedActualTypes(
			&SimpleType{
				QualifiedName: "Bar.Foo",
				TypeID:        "A.0000000000000001.Bar.Foo",
			},
			&SimpleType{
				QualifiedName: "Bar.Foo",
				TypeID:        "A.0000000000000002.Bar.Foo",
			})

		assert.Equal(t, "A.0000000000000001.Bar.Foo", expected)
		assert.Equal(t, "A.0000000000000002.Bar.Foo", actual)

	})
}

func TestUnwrappingCheckerError(t *testing.T) {
	t.Parallel()

	targetErr := fmt.Errorf("target error")

	t.Run("unwrap matches child", func(t *testing.T) {
		t.Parallel()

		err := CheckerError{
			Errors: []error{
				fmt.Errorf("first error"),
				targetErr,
			},
		}

		assert.True(t, errors.Is(err, targetErr))
	})

	t.Run("unwrap matches wrapped child", func(t *testing.T) {
		t.Parallel()

		err := CheckerError{
			Errors: []error{
				fmt.Errorf("first error"),
				fmt.Errorf("wrapped error: %w", &InvalidPragmaError{}),
			},
		}

		var pragmaErr *InvalidPragmaError
		assert.True(t, errors.As(err, &pragmaErr))
	})

	t.Run("unwrap finds wrapped grandchild", func(t *testing.T) {
		t.Parallel()

		err := CheckerError{
			Errors: []error{
				fmt.Errorf("first error"),
				fmt.Errorf("wrapped error: %w", &parser.Error{
					Errors: []error{
						fmt.Errorf("first parser error"),
						targetErr,
					},
				}),
			},
		}

		var parserErr *parser.Error
		assert.True(t, errors.As(err, &parserErr))
		assert.True(t, errors.Is(err, targetErr))

		var pragmaErr *InvalidPragmaError
		assert.False(t, errors.As(err, &pragmaErr))
	})
}
