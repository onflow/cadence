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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/ast"
	"github.com/onflow/cadence/errors"
	"github.com/onflow/cadence/sema"
)

func TestCheckEventNonTypeRequirementConformance(t *testing.T) {

	t.Parallel()
	_, err := ParseAndCheck(t, `
      access(all) contract interface CI {

          access(all) event E(a: Int)
      }

      access(all) contract C: CI {

          access(all) event E(b: String)
      }
    `)

	require.NoError(t, err)
}

func TestCheckConformanceWithFunctionSubtype(t *testing.T) {

	t.Parallel()

	t.Run("valid, return type is subtype", func(t *testing.T) {

		t.Parallel()

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

	t.Run("invalid, return type is supertype", func(t *testing.T) {

		t.Parallel()

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

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("valid, return type is the same", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun get(): @R
          }

          struct S: SI {
              fun get(): @R {
                  return <- create R()
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("valid, parameter type is the same", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface RI {}

          resource R: RI {}

          struct interface SI {
              fun set(r: @{RI})
          }

          struct S: SI {
              fun set(r: @{RI}) {
                  destroy r
              }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid, parameter type is subtype", func(t *testing.T) {

		t.Parallel()

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

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})

	t.Run("invalid, parameter type is supertype", func(t *testing.T) {

		t.Parallel()

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

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
	})
}

func TestCheckInitializerConformanceErrorMessages(t *testing.T) {

	t.Parallel()

	t.Run("initializer notes", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
      access(all) resource interface I {
          let x: Int 
          init(x: Int)
      }

      access(all) resource R: I {
        let x: Int 
        init() {
            self.x = 1
        }
      }
    `)

		errs := RequireCheckerErrors(t, err, 1)

		var conformanceErr *sema.ConformanceError
		require.ErrorAs(t, errs[0], &conformanceErr)

		require.NotNil(t, conformanceErr.InitializerMismatch)
		notes := conformanceErr.ErrorNotes()
		require.Len(t, notes, 1)

		require.Equal(t, &sema.MemberMismatchNote{
			Range: ast.Range{
				StartPos: ast.Position{Offset: 158, Line: 9, Column: 8},
				EndPos:   ast.Position{Offset: 161, Line: 9, Column: 11},
			},
		}, notes[0])
	})

	t.Run("1 missing member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          access(all) resource interface I {
              fun foo(): Int
          }

          access(all) resource R: I {
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var conformanceErr *sema.ConformanceError
		require.ErrorAs(t, errs[0], &conformanceErr)

		require.Equal(t,
			"`R` is missing definitions for members: `foo`",
			conformanceErr.SecondaryError(),
		)
	})

	t.Run("2 missing member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          access(all) resource interface I {
              fun foo(): Int
              fun bar(): Int
          }

          access(all) resource R: I {
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var conformanceErr *sema.ConformanceError
		require.ErrorAs(t, errs[0], &conformanceErr)

		require.Equal(t,
			"`R` is missing definitions for members: `foo`, `bar`",
			conformanceErr.SecondaryError(),
		)
	})
}

func TestCheckConformanceAccessModifierMatches(t *testing.T) {
	t.Parallel()

	e1 := &sema.EntitlementType{
		Identifier: "E1",
	}
	e2 := &sema.EntitlementType{
		Identifier: "E2",
	}

	accessModifiers := []sema.Access{
		sema.PrimitiveAccess(ast.AccessSelf),
		sema.PrimitiveAccess(ast.AccessAccount),
		sema.PrimitiveAccess(ast.AccessContract),
		sema.NewEntitlementSetAccess(
			[]*sema.EntitlementType{e1, e2},
			sema.Conjunction,
		),
		sema.NewEntitlementSetAccess(
			[]*sema.EntitlementType{e1, e2},
			sema.Disjunction,
		),
		sema.NewEntitlementSetAccess(
			[]*sema.EntitlementType{e1},
			sema.Conjunction,
		),
		sema.PrimitiveAccess(ast.AccessAll),
	}

	asASTAccess := func(access sema.Access) ast.Access {
		switch access := access.(type) {
		case sema.PrimitiveAccess:
			return ast.PrimitiveAccess(access)

		case sema.EntitlementSetAccess:

			entitlementTypes := make([]*ast.NominalType, 0, access.Entitlements.Len())

			access.Entitlements.Foreach(func(entitlementType *sema.EntitlementType, _ struct{}) {
				entitlementTypes = append(
					entitlementTypes,
					&ast.NominalType{
						Identifier: ast.Identifier{
							Identifier: entitlementType.QualifiedIdentifier(),
						},
					},
				)
			})

			var entitlementSet ast.EntitlementSet
			switch access.SetKind {
			case sema.Conjunction:
				entitlementSet = ast.NewConjunctiveEntitlementSet(entitlementTypes)

			case sema.Disjunction:
				entitlementSet = ast.NewDisjunctiveEntitlementSet(entitlementTypes)

			default:
				panic(errors.NewUnreachableError())
			}

			return ast.EntitlementAccess{
				EntitlementSet: entitlementSet,
			}

		default:
			panic(errors.NewUnreachableError())
		}
	}

	test := func(t *testing.T, interfaceAccess, implementationAccess sema.Access) {
		name := fmt.Sprintf("%s %s", interfaceAccess, implementationAccess)
		t.Run(name, func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      entitlement E1
                      entitlement E2

                      struct interface SI {
                          %s fun foo()
                      }

                      struct S: SI {
                          %s fun foo() {}
                      }
                    `,
					asASTAccess(interfaceAccess).Keyword(),
					asASTAccess(implementationAccess).Keyword(),
				),
			)

			if interfaceAccess == sema.PrimitiveAccess(ast.AccessSelf) {
				if implementationAccess == sema.PrimitiveAccess(ast.AccessSelf) {
					errs := RequireCheckerErrors(t, err, 1)

					require.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
				} else {
					errs := RequireCheckerErrors(t, err, 2)

					require.IsType(t, &sema.InvalidAccessModifierError{}, errs[0])
					require.IsType(t, &sema.ConformanceError{}, errs[1])
				}
			} else if !implementationAccess.Equal(interfaceAccess) {
				errs := RequireCheckerErrors(t, err, 1)

				var conformanceErr *sema.ConformanceError
				require.ErrorAs(t, errs[0], &conformanceErr)

				require.Len(t, conformanceErr.MemberMismatches, 1)

			} else {
				require.NoError(t, err)
			}
		})
	}

	for _, access1 := range accessModifiers {
		for _, access2 := range accessModifiers {
			test(t, access1, access2)
		}
	}
}
