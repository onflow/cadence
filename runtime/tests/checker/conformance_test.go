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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
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

	errs := RequireCheckerErrors(t, err, 1)

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
			errs := RequireCheckerErrors(t, err, 1)

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

func TestCheckTypeRequirementDuplicateDeclaration(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
	  contract interface CI {
          // Checking if CI_TR1 conforms to CI here,
          // requires checking the type requirement CI_TR2 of CI.
          //
          // Note that CI_TR1 here declares 2 (!)
          // nested composite declarations named CI_TR2.
          //
          // Checking should not just use the first declaration named CI_TR2,
          // but detect the second / duplicate, error,
          // and stop further conformance checking
          //
	      contract CI_TR1: CI {
	          contract CI_TR2 {}
	          contract CI_TR2: CI {
	              contract CI_TR2_TR {}
	          }
	      }

	      contract CI_TR2: CI {
	          contract CI_TR2_TR {}
	      }
	  }
	`)

	errs := RequireCheckerErrors(t, err, 13)

	require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[1])
	require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[2])
	require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[3])
	require.IsType(t, &sema.RedeclarationError{}, errs[4])
	require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[5])
	require.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[6])
	require.IsType(t, &sema.RedeclarationError{}, errs[7])
	require.IsType(t, &sema.RedeclarationError{}, errs[8])
	require.IsType(t, &sema.RedeclarationError{}, errs[9])
	require.IsType(t, &sema.ConformanceError{}, errs[10])
	require.IsType(t, &sema.ConformanceError{}, errs[11])
	require.IsType(t, &sema.ConformanceError{}, errs[12])
}

func TestCheckMultipleTypeRequirements(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      contract interface IA {

          struct X {
              let a: Int
          }
      }

      contract interface IB {

          struct X {
              let b: Int
          }
      }

      contract Test: IA, IB {

          struct X {
              let a: Int
              // missing b

              init() {
                  self.a = 0
              }
          }
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	require.IsType(t, &sema.ConformanceError{}, errs[0])
}

func TestCheckInitializerConformanceErrorMessages(t *testing.T) {

	t.Parallel()

	t.Run("initializer notes", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
      pub resource interface I {
          let x: Int 
          init(x: Int)
      }

      pub resource R: I {
        let x: Int 
        init() {
            self.x = 1
        }
      }
    `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])

		conformanceErr := errs[0].(*sema.ConformanceError)
		require.NotNil(t, conformanceErr.InitializerMismatch)
		notes := conformanceErr.ErrorNotes()
		require.Len(t, notes, 1)

		require.Equal(t, &sema.MemberMismatchNote{
			Range: ast.Range{
				StartPos: ast.Position{Offset: 142, Line: 9, Column: 8},
				EndPos:   ast.Position{Offset: 145, Line: 9, Column: 11},
			},
		}, notes[0])
	})

	t.Run("1 missing member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        pub resource interface I {
            fun foo(): Int
        }

        pub resource R: I {
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		conformanceErr := errs[0].(*sema.ConformanceError)
		require.Equal(t, "`R` is missing definitions for members: `foo`", conformanceErr.SecondaryError())
	})

	t.Run("2 missing member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        pub resource interface I {
            fun foo(): Int
            fun bar(): Int
        }

        pub resource R: I {
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		conformanceErr := errs[0].(*sema.ConformanceError)
		require.Equal(t, "`R` is missing definitions for members: `foo`, `bar`", conformanceErr.SecondaryError())
	})

	t.Run("1 missing type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        pub contract interface I {
            pub struct S {}
        }

        pub contract C: I {
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		conformanceErr := errs[0].(*sema.ConformanceError)
		require.Equal(t, "`C` is missing definitions for types: `I.S`", conformanceErr.SecondaryError())
	})

	t.Run("2 missing type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        pub contract interface I {
            pub struct S {}
            pub resource R {}
        }

        pub contract C: I {
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		conformanceErr := errs[0].(*sema.ConformanceError)
		require.Equal(t, "`C` is missing definitions for types: `I.S`, `I.R`", conformanceErr.SecondaryError())
	})

	t.Run("missing type and member", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
        pub contract interface I {
            pub struct S {}
            pub fun foo() 
        }

        pub contract C: I {
        }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.ConformanceError{}, errs[0])
		conformanceErr := errs[0].(*sema.ConformanceError)
		require.Equal(t, "`C` is missing definitions for members: `foo`. `C` is also missing definitions for types: `I.S`", conformanceErr.SecondaryError())
	})
}
