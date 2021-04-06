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

	"github.com/onflow/cadence/runtime/sema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckCapability(t *testing.T) {

	t.Parallel()

	t.Run("type annotation, untyped", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t, `
          let cap: Capability = panic("")
        `)

		require.NoError(t, err)

		capType := RequireGlobalValue(t, checker.Elaboration, "cap")

		assert.IsType(t,
			&sema.CapabilityType{},
			capType,
		)
	})

	t.Run("type annotation, typed", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t, `
          let cap: Capability<&Int> = panic("")
        `)

		require.NoError(t, err)

		capType := RequireGlobalValue(t, checker.Elaboration, "cap")

		assert.IsType(t,
			&sema.CapabilityType{
				BorrowType: sema.IntType,
			},
			capType,
		)
	})
}

func TestCheckCapability_borrow(t *testing.T) {

	t.Parallel()

	t.Run("missing type argument", func(t *testing.T) {

		_, err := ParseAndCheckWithPanic(t, `

          let capability: Capability = panic("")

          let r = capability.borrow()
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	for _, auth := range []bool{false, true} {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf(
			"explicit type argument, %s reference",
			authKeyword,
		)

		t.Run(testName, func(t *testing.T) {

			t.Run("untyped, resource", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let capability: Capability = panic("")

                          let r = capability.borrow<%s &R>()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				rType := RequireGlobalType(t, checker.Elaboration, "R")
				rValueType := RequireGlobalValue(t, checker.Elaboration, "r")

				require.Equal(t,
					&sema.OptionalType{
						Type: &sema.ReferenceType{
							Authorized: auth,
							Type:       rType,
						},
					},
					rValueType,
				)
			})

			t.Run("typed, resource", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let capability: Capability<%s &R> = panic("")

                          let r = capability.borrow()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				rType := RequireGlobalType(t, checker.Elaboration, "R")
				rValueType := RequireGlobalValue(t, checker.Elaboration, "r")

				require.Equal(t,
					&sema.OptionalType{
						Type: &sema.ReferenceType{
							Authorized: auth,
							Type:       rType,
						},
					},
					rValueType,
				)
			})

			t.Run("untyped, struct", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let capability: Capability = panic("")

                          let s = capability.borrow<%s &S>()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				sType := RequireGlobalType(t, checker.Elaboration, "S")
				sValueType := RequireGlobalValue(t, checker.Elaboration, "s")

				require.Equal(t,
					&sema.OptionalType{
						Type: &sema.ReferenceType{
							Authorized: auth,
							Type:       sType,
						},
					},
					sValueType,
				)
			})

			t.Run("typed, struct", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let capability: Capability<%s &S> = panic("")

                          let s = capability.borrow()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				sType := RequireGlobalType(t, checker.Elaboration, "S")
				sValueType := RequireGlobalValue(t, checker.Elaboration, "s")

				require.Equal(t,
					&sema.OptionalType{
						Type: &sema.ReferenceType{
							Authorized: auth,
							Type:       sType,
						},
					},
					sValueType,
				)
			})
		})
	}

	t.Run("explicit type argument, non-reference type", func(t *testing.T) {

		t.Run("resource", func(t *testing.T) {

			_, err := ParseAndCheckWithPanic(t, `

              resource R {}

              let capability: Capability = panic("")

              let r <- capability.borrow<@R>()
            `)

			errs := ExpectCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("struct", func(t *testing.T) {

			_, err := ParseAndCheckWithPanic(t, `

              struct S {}

              let capability: Capability = panic("")

              let s = capability.borrow<S>()
            `)

			errs := ExpectCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})
}

func TestCheckCapability_check(t *testing.T) {

	t.Parallel()

	t.Run("untyped, missing type argument", func(t *testing.T) {

		_, err := ParseAndCheckWithPanic(t, `

          let capability: Capability = panic("")

          let ok = capability.check()
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	for _, auth := range []bool{false, true} {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf(
			"explicit type argument, %s reference",
			authKeyword,
		)

		t.Run(testName, func(t *testing.T) {

			t.Run("untyped, resource", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let capability: Capability = panic("")

                          let ok = capability.check<%s &R>()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				okType := RequireGlobalValue(t, checker.Elaboration, "ok")

				require.Equal(t,
					sema.BoolType,
					okType,
				)
			})

			t.Run("typed, resource", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          resource R {}

                          let capability: Capability<%s &R> = panic("")

                          let ok = capability.check()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				okType := RequireGlobalValue(t, checker.Elaboration, "ok")

				require.Equal(t,
					sema.BoolType,
					okType,
				)
			})

			t.Run("untyped, struct", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let capability: Capability = panic("")

                          let ok = capability.check<%s &S>()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				okType := RequireGlobalValue(t, checker.Elaboration, "ok")

				require.Equal(t,
					sema.BoolType,
					okType,
				)
			})

			t.Run("typed, struct", func(t *testing.T) {

				checker, err := ParseAndCheckWithPanic(t,
					fmt.Sprintf(
						`
                          struct S {}

                          let capability: Capability<%s &S> = panic("")

                          let ok = capability.check()
                        `,
						authKeyword,
					),
				)

				require.NoError(t, err)

				okType := RequireGlobalValue(t, checker.Elaboration, "ok")

				require.Equal(t,
					sema.BoolType,
					okType,
				)
			})
		})
	}

	t.Run("explicit type argument, non-reference type", func(t *testing.T) {

		t.Run("resource", func(t *testing.T) {

			_, err := ParseAndCheckWithPanic(t, `

              resource R {}

              let capability: Capability = panic("")

              let ok = capability.check<@R>()
            `)

			errs := ExpectCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("struct", func(t *testing.T) {

			_, err := ParseAndCheckWithPanic(t, `

              struct S {}

              let capability: Capability = panic("")

              let ok = capability.check<S>()
            `)

			errs := ExpectCheckerErrors(t, err, 1)

			require.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})
}
