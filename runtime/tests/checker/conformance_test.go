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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckInvalidEventTypeRequirementConformance(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      pub contract interface CI {

          pub event E(a: Int)
      }

      pub contract C: CI {

          pub event E(b: String)
      }
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	require.IsType(t, &sema.ConformanceError{}, errs[0])
}

func TestCheckTypeRequirementConformance(t *testing.T) {

	t.Parallel()

	test := func(preparationCode string, interfaceCode string, conformanceCode string, valid bool) {
		_, err := ParseAndCheck(t,
			fmt.Sprintf(
				`
                  %s

                  pub contract interface CI {
                      %s
                  }

                  pub contract C: CI {
                      %s
                  }
                `,
				preparationCode,
				interfaceCode,
				conformanceCode,
			),
		)

		if valid {
			require.NoError(t, err)
		} else {
			errs := ExpectCheckerErrors(t, err, 1)

			require.IsType(t, &sema.ConformanceError{}, errs[0])
		}
	}

	t.Run("Both empty", func(t *testing.T) {

		t.Parallel()

		test(
			``,
			`pub struct S {}`,
			`pub struct S {}`,
			true,
		)
	})

	t.Run("Conformance with additional function", func(t *testing.T) {

		t.Parallel()

		test(
			``,
			`
              pub struct S {}
            `,
			`
              pub struct S {
                  fun foo() {}
              }
            `,
			true,
		)
	})

	t.Run("Conformance with missing function", func(t *testing.T) {

		t.Parallel()

		test(
			``,
			`
              pub struct S {
                  fun foo()
              }
            `,
			`
              pub struct S {}
            `,
			false,
		)
	})

	t.Run("Conformance with same name, same parameter type, but different argument label", func(t *testing.T) {

		t.Parallel()

		test(
			``,
			`
              pub struct S {
                  fun foo(x: Int)
              }
            `,
			`
              pub struct S {
                  fun foo(y: Int) {}
              }
            `,
			false,
		)
	})

	t.Run("Conformance with same name, same argument label, but different parameter type", func(t *testing.T) {

		t.Parallel()

		test(
			``,
			`
              pub struct S {
                  fun foo(x: Int)
              }
            `,
			`
              pub struct S {
                  fun foo(x: String) {}
              }
            `,
			false,
		)
	})

	t.Run("Conformance with same name, same argument label, same parameter type, different parameter name", func(t *testing.T) {

		t.Parallel()

		test(
			``,
			`
              pub struct S {
                  fun foo(x y: String)
              }
            `,
			`
              pub struct S {
                  fun foo(x z: String) {}
              }
            `,
			true,
		)
	})

	t.Run("Conformance with more specific parameter type", func(t *testing.T) {

		t.Parallel()

		test(
			`
                pub struct interface I {}
                pub struct T: I {}
            `,
			`
              pub struct S {
                  fun foo(bar: {I})
              }
            `,
			`
              pub struct S {
                  fun foo(bar: T) {}
              }
            `,
			false,
		)
	})

	t.Run("Conformance with same nested parameter type", func(t *testing.T) {

		t.Parallel()

		test(
			`
                pub contract X {
                    struct Bar {}
                }
            `,
			`
              pub struct S {
                  fun foo(bar: X.Bar)
              }
            `,
			`
              pub struct S {
                  fun foo(bar: X.Bar) {}
              }
            `,
			true,
		)
	})

	t.Run("Conformance with different nested parameter type", func(t *testing.T) {

		t.Parallel()

		test(
			`
              pub contract X {
                  struct Bar {}
              }

              pub contract Y {
                  struct Bar {}
              }
            `,
			`
              pub struct S {
                  fun foo(bar: X.Bar)
              }
            `,
			`
              pub struct S {
                  fun foo(bar: Y.Bar) {}
              }
            `,
			false,
		)

	})
}

func TestCheckConformanceWithFunctionSubtype(t *testing.T) {

	t.Parallel()

	t.Run("valid, return type is subtype", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun get(): @{RI}
          }

          struct S: SI {
              fun get(): @R {
                  return <- create R()
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("valid, parameter type is supertype", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun set(r: @R) 
          }

          struct S: SI {
              fun set(r: @{RI}) {
                  destroy r
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid, return type is supertype", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun get(): @R
          }

          struct S: SI {
              fun get(): @{RI} {
                  return <- create R()
              }
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid, parameter type is subtype", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun set(r: @{RI})
          }

          struct S: SI {
              fun set(r: @R) {
                  destroy r
              }
          }
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})
}
