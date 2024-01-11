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

package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

func TestCheckReference(t *testing.T) {

	t.Parallel()

	t.Run("variable declaration type annotation", func(t *testing.T) {

		t.Parallel()

		t.Run("non-auth", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              let x: &Int = &1
            `)

			require.NoError(t, err)

		})

		t.Run("auth", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
            entitlement X
              let x: auth(X) &Int = &1
            `)

			require.NoError(t, err)
		})

		t.Run("non-reference type", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              let x: Int = &1
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[0])
		})
	})

	t.Run("variable declaration type annotation", func(t *testing.T) {

		t.Run("non-auth", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              let x = &1 as &Int
            `)

			require.NoError(t, err)
		})

		t.Run("auth", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              entitlement X
              let x = &1 as auth(X) &Int
            `)

			require.NoError(t, err)
		})

		t.Run("non-reference type", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              let x = &1 as Int
            `)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[0])
		})
	})

	t.Run("invalid non-auth to auth cast", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          entitlement X
          let x = &1 as &Int as auth(X) &Int
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("missing type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let x = &1
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeAnnotationRequiredError{}, errs[0])
	})

}

func TestCheckReferenceTypeOuter(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: &[R]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          fun test(s: &[S]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(i: &[Int]) {}
        `)

		require.NoError(t, err)
	})
}

func TestCheckReferenceTypeInner(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: [&R]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          fun test(s: [&S]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(i: [&Int]) {}
        `)

		require.NoError(t, err)
	})

}

func TestCheckNestedReferenceType(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: &[&R]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          fun test(s: &[&S]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test(s: &[&Int]) {}
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidReferenceType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun test(t: &T) {}
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

}

func TestCheckReferenceExpressionWithNonCompositeResultType(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `

      let i = 1
      let ref = &i as &Int
    `)

	require.NoError(t, err)

	refValueType := RequireGlobalValue(t, checker.Elaboration, "ref")

	assert.Equal(t,
		&sema.ReferenceType{
			Type:          sema.IntType,
			Authorization: sema.UnauthorizedAccess,
		},
		refValueType,
	)
}

func TestCheckReferenceExpressionWithCompositeResultType(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          resource R {}

          let r <- create R()
          let ref = &r as &R
        `)

		require.NoError(t, err)

		rType := RequireGlobalType(t, checker.Elaboration, "R")

		refValueType := RequireGlobalValue(t, checker.Elaboration, "ref")

		assert.Equal(t,
			&sema.ReferenceType{
				Type:          rType,
				Authorization: sema.UnauthorizedAccess,
			},
			refValueType,
		)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          struct S {}

          let s = S()
          let ref = &s as &S
        `)

		require.NoError(t, err)

		sType := RequireGlobalType(t, checker.Elaboration, "S")

		refValueType := RequireGlobalValue(t, checker.Elaboration, "ref")

		assert.Equal(t,
			&sema.ReferenceType{
				Type:          sType,
				Authorization: sema.UnauthorizedAccess,
			},
			refValueType,
		)
	})
}

func TestCheckReferenceExpressionWithInterfaceResultType(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface I {}
          resource R: I {}

          let r <- create R()
          let ref = &r as &I
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct interface I {}
          struct S: I {}

          let s = S()
          let ref = &s as &I
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidInterfaceTypeError{}, errs[0])
	})
}

func TestCheckReferenceExpressionWithRdAnyResultType(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          let r <- create R()
          let ref = &r as &AnyResource
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          let s = S()
          let ref = &s as &AnyStruct
        `)

		require.NoError(t, err)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i = 1
          let ref = &i as &AnyStruct
        `)

		require.NoError(t, err)
	})
}

func TestCheckReferenceExpressionWithIntersectionAnyResultType(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface I {}
          resource R: I {}

          let r <- create R()
          let ref = &r as &{I}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct interface I {}
          struct S: I {}

          let s = S()
          let ref = &s as &{I}
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidReferenceExpressionType(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          let r <- create R()
          let ref = &r as &X
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          let s = S()
          let ref = &s as &X
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i = 1
          let ref = &i as &X
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckInvalidReferenceExpressionTypeMismatchStructResource(t *testing.T) {

	t.Parallel()

	t.Run("struct / resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}
          resource R {}

          let s = S()
          let ref = &s as &R
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("resource / struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}
          resource R {}

          let r <- create R()
          let ref = &r as &S
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckInvalidReferenceExpressionDifferentStructs(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      struct S {}
      struct T {}

      let s = S()
      let ref = &s as &T
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidReferenceExpressionTypeMismatchDifferentResources(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}
      resource T {}

      let r <- create R()
      let ref = &r as &T
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckReferenceResourceArrayIndexing(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      let rs <- [<-create R()]
      let ref = &rs[0] as &R
    `)

	require.NoError(t, err)
}

func TestCheckReferenceUse(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              init() {
                  self.x = 0
              }

              fun setX(_ newX: Int) {
                  self.x = newX
              }
          }

          fun test(): [Int] {
              let r <- create R()
              let ref = &r as &R
              ref.x = 1
              let x1 = ref.x
              ref.setX(2)
              let x2 = ref.x
              destroy r
              return [x1, x2]
          }
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              var x: Int

              init() {
                  self.x = 0
              }

              fun setX(_ newX: Int) {
                  self.x = newX
              }
          }

          fun test(): [Int] {
              let s = S()
              let ref = &s as &S
              ref.x = 1
              let x1 = ref.x
              ref.setX(2)
              let x2 = ref.x
              return [x1, x2]
          }
        `)

		require.NoError(t, err)
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          fun test(): String {
              let i = 1
              let ref = &i as &Int
              return ref.toString()
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckReferenceUseArray(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var x: Int

              init() {
                  self.x = 0
              }

              fun setX(_ newX: Int) {
                  self.x = newX
              }
          }

          fun test(): [Int] {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
              ref[0].x = 1
              let x1 = ref[0].x
              ref[0].setX(2)
              let x2 = ref[0].x
              destroy rs
              return [x1, x2]
          }
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {
              var x: Int

              init() {
                  self.x = 0
              }

              fun setX(_ newX: Int) {
                  self.x = newX
              }
          }

          fun test(): [Int] {
              let s = [S()]
              let ref = &s as &[S]
              ref[0].x = 1
              let x1 = ref[0].x
              ref[0].setX(2)
              let x2 = ref[0].x
              return [x1, x2]
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckReferenceIndexingIfReferencedIndexable(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
              ref[0]
              destroy rs
          }
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          struct S {}

          fun test() {
              let s = [S()]
              let ref = &s as &[S]
              ref[0]
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckReferenceResourceLoss(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let rs <- [<-create R()]
          let ref = &rs as &[R]
          ref[0]  // This result in a reference, so no resource loss
          destroy rs
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidReferenceResourceLoss(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun f(): @R {
          return <- create R()
      }

      fun test() {
          let ref = &f() as &R
      }
    `)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidReferenceIndexingIfReferencedNotIndexable(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource R {}

          fun test() {
              let r <- create R()
              let ref = &r as &R
              ref[0]
              destroy r
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          struct S {}

          fun test() {
              let s = S()
              let ref = &s as &S
              ref[0]
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidTypeIndexingError{}, errs[0])
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          fun test() {
              let i = 1
              let ref = &i as &Int
              ref[0]
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
	})
}

func TestCheckResourceInterfaceReferenceFunctionCall(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource interface I {
              fun foo()
          }

          resource R: I {
              fun foo() {}
          }

          fun test() {
              let r <- create R()
              let ref = &r as &{I}
              ref.foo()
              destroy r
          }
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          struct interface I {
              fun foo()
          }

          struct S: I {
              fun foo() {}
          }

          fun test() {
              let s = S()
              let ref = &s as &{I}
              ref.foo()
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidResourceInterfaceReferenceFunctionCall(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          resource interface I {}

          resource R: I {
              fun foo() {}
          }

          fun test() {
              let r <- create R()
              let ref = &r as &{I}
              ref.foo()
              destroy r
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          struct interface I {}

          struct S: I {
              fun foo() {}
          }

          fun test() {
              let s = S()
              let ref = &s as &{I}
              ref.foo()
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})
}

func TestCheckReferenceExpressionReferenceType(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, auth sema.Access, kind common.CompositeKind) {

		var authKeyword string
		if auth != sema.UnauthorizedAccess {
			authKeyword = fmt.Sprintf("auth(%s)", auth.QualifiedString())
		}

		testName := fmt.Sprintf("%s, auth: %v", kind.Name(), auth)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T {}
                      entitlement X

                      let t %[2]s %[3]s T()
                      let ref = &t as %[4]s &T
                    `,
					kind.Keyword(),
					kind.TransferOperator(),
					kind.ConstructionKeyword(),
					authKeyword,
				),
			)

			require.NoError(t, err)

			tType := RequireGlobalType(t, checker.Elaboration, "T")

			refValueType := RequireGlobalValue(t, checker.Elaboration, "ref")
			xType := RequireGlobalType(t, checker.Elaboration, "X").(*sema.EntitlementType)
			var access sema.Access = sema.UnauthorizedAccess
			if !auth.Equal(sema.UnauthorizedAccess) {
				access = sema.NewEntitlementSetAccess([]*sema.EntitlementType{xType}, sema.Conjunction)
			}

			require.Equal(t,
				&sema.ReferenceType{
					Authorization: access,
					Type:          tType,
				},
				refValueType,
			)
		})
	}

	for _, kind := range []common.CompositeKind{
		common.CompositeKindResource,
		common.CompositeKindStructure,
	} {
		for _, auth := range []sema.Access{
			sema.UnauthorizedAccess,
			sema.NewEntitlementSetAccess([]*sema.EntitlementType{{
				Location:   utils.TestLocation,
				Identifier: "X",
			}}, sema.Conjunction),
		} {
			test(t, auth, kind)
		}
	}
}

func TestCheckReferenceExpressionOfOptional(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          let r: @R? <- create R()
          let ref = &r as &R
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          let s: S? = S()
          let ref = &s as &S
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i: Int? = 1
          let ref = &i as &Int
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("as optional", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let i: Int? = 1
          let ref = &i as &Int?
        `)

		require.NoError(t, err)
		refValueType := RequireGlobalValue(t, checker.Elaboration, "ref")

		assert.Equal(t,
			&sema.OptionalType{
				Type: &sema.ReferenceType{
					Type:          sema.IntType,
					Authorization: sema.UnauthorizedAccess,
				},
			},
			refValueType,
		)
	})

	t.Run("double optional", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i: Int? = 1
          let ref = &i as &Int??
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[0])
	})

	t.Run("mismatched type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i: String? = ""
          let ref = &i as &Int?
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("optional reference to non-optional value", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i: Int = 1
          let ref = &i as &Int?
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("non-optional reference to optional value", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let opt: Int? = 1
          let ref = &opt as &AnyStruct
        `)

		errs := RequireCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckNilCoalesceReference(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheckWithPanic(t, `
      let xs = {"a": 1}
      let ref = &xs["a"] as &Int? ?? panic("no a")
    `)
	require.NoError(t, err)

	refValueType := RequireGlobalValue(t, checker.Elaboration, "ref")

	assert.Equal(t,
		&sema.ReferenceType{
			Type:          sema.IntType,
			Authorization: sema.UnauthorizedAccess,
		},
		refValueType,
	)
}

func TestCheckInvalidReferenceExpressionNonReferenceAmbiguous(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let y = &x as {}
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidReferenceExpressionNonReferenceAnyStruct(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let y = &x as {}
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidDictionaryAccessReference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let xs: {Int: Int} = {}
      let ref = &xs[1] as &String
    `)

	errs := RequireCheckerErrors(t, err, 1)

	require.IsType(t, &sema.TypeMismatchError{}, errs[0])

	typeMismatchError := errs[0].(*sema.TypeMismatchError)
	assert.Equal(t, 17, typeMismatchError.StartPos.Column)
	assert.Equal(t, 21, typeMismatchError.EndPos.Column)
}

func TestCheckDictionaryAccessReferenceIsOptional(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let xs: {Int: Int} = {}
      let ref: Int = &xs[1] as &Int?
    `)

	errs := RequireCheckerErrors(t, err, 1)

	require.IsType(t, &sema.TypeMismatchError{}, errs[0])

	typeMismatchError := errs[0].(*sema.TypeMismatchError)
	assert.Equal(t, 21, typeMismatchError.StartPos.Column)
	assert.Equal(t, 35, typeMismatchError.EndPos.Column)
}

func TestCheckInvalidDictionaryAccessOptionalReference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        access(all) struct S {
            access(all) let foo: Number
            init() {
                self.foo = 0
            }
        }
        let dict: {String: S} = {}
        let s = &dict[""] as &S?
        let n = s.foo
    `)

	errs := RequireCheckerErrors(t, err, 1)

	require.IsType(t, &sema.NotDeclaredMemberError{}, errs[0]) // nil has no member foo
}

func TestCheckInvalidDictionaryAccessNonOptionalReference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        access(all) struct S {
            access(all) let foo: Number
            init() {
                self.foo = 0
            }
        }
        let dict: {String: S} = {}
        let s = &dict[""] as &S
    `)

	errs := RequireCheckerErrors(t, err, 1)

	require.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckArrayAccessReference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        access(all) struct S {
            access(all) let foo: Number
            init() {
                self.foo = 0
            }
        }
        let dict: [S] = []
        let s = &dict[0] as &S
        let n = s.foo
    `)

	require.NoError(t, err)
}

func TestCheckInvalidatedReferenceUse(t *testing.T) {

	t.Parallel()

	t.Run("no errors", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let xRef = &x as &R
                xRef.a
                destroy x
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("after destroy", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let xRef = &x as &R
                destroy x
                xRef.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("after destroy - array", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- [<-create R()]
                let xRef = &x as &[R]
                destroy x
                xRef[0].a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("after destroy - dictionary", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- {1: <- create R()}
                let xRef = &x as &{Int: R}
                destroy x
                xRef[1]?.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("after move", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let xRef = &x as &R
                consume(<-x)
                xRef.a
            }

            access(all) fun consume(_ r: @AnyResource) {
                destroy r
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("after move - array", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- [<-create R()]
                let xRef = &x as &[R]
                consume(<-x)
                xRef[0].a
            }

            access(all) fun consume(_ r: @AnyResource) {
                destroy r
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("after move - dictionary", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- {1: <- create R()}
                let xRef = &x as &{Int: R}
                consume(<-x)
                xRef[1]?.a
            }

            access(all) fun consume(_ r: @AnyResource) {
                destroy r
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("after swap", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                var x <- create R()
                var y <- create R()
                let xRef = &x as &R
                x <-> y
                destroy x
                destroy y
                xRef.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let xRef = &x as &R
                if true {
                    destroy x
                } else {
                    destroy x
                }

                if true {
                    if true {
                    } else {
                        xRef.a
                    }
                }
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("storage reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test(storage: auth(Storage) &Account.Storage) {
                storage.save(<-[<-create R()], to: /storage/a)

                let collectionRef = storage.borrow<&[R]>(from: /storage/a)!
                let ref = collectionRef[0]

                let collection <- storage.load<@[R]>(from: /storage/a)!
                storage.save(<- collection, to: /storage/b)

                ref.setA(2)
            }

            access(all) resource R {
                access(all) var a: Int

                access(all) fun setA(_ a: Int) {
                    self.a = a
                }

                init() {
                    self.a = 5
                }
            }
            `,
		)

		// Cannot detect storage transfers
		require.NoError(t, err)
	})

	t.Run("inside func expr", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let f = fun() {
                    let x <- create R()
                    let xRef = &x as &R
                    destroy x
                    xRef.a
                }

                f()
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("self var", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) contract Test {
                access(self) var x: @R
                init() {
                    self.x <- create R()
                }

                access(all) fun test() {
                    let xRef = &self.x as &R
                    xRef.a
                }
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("self var using contract name", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) contract Test {
                access(self) var x: @R
                init() {
                    self.x <- create R()
                }

                access(all) fun test() {
                    let xRef = &Test.x as &R
                    xRef.a
                }
            }

            access(all) resource R {
                access(all) let a: Int
                init() {
                    self.a = 5
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("ref to ref", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                var r: @{UInt64: {UInt64: [R]}} <- {}
                let ref1 = (&r[0] as &{UInt64: [R]}?)!
                let ref2 = ref1[0]!
                let ref3 = ref2[0]
                ref3.a

                destroy r
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("ref to ref invalid, index expr", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                var r: @{UInt64: {UInt64: [R]}} <- {}
                let ref1 = (&r[0] as &{UInt64: [R]}?)!
                let ref2 = ref1[0]!
                let ref3 = ref2[0]
                destroy r
                ref3.a
            }

            access(all) resource R {
                access(all) let a: Int
                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("ref to ref invalid, member expr", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                var r: @R1 <- create R1()
                let ref1 = &r as &R1
                let ref2 = ref1.r2
                let ref3 = ref2.r3
                destroy r
                ref3.a
            }

            access(all) resource R1 {
                access(all) let r2: @R2
                init() {
                    self.r2 <- create R2()
                }
            }

            access(all) resource R2 {
                access(all) let r3: @R3
                init() {
                    self.r3 <- create R3()
                }
            }

            access(all) resource R3 {
                access(all) let a: Int
                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("create ref with force expr", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let xRef = (&x as &R?)!
                destroy x
                xRef.a
            }

            access(all) resource R {
                access(all) let a: Int
                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 2)

		typeMismatchError := &sema.TypeMismatchError{}
		assert.ErrorAs(t, errors[0], &typeMismatchError)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
	})

	t.Run("contract field ref", func(t *testing.T) {

		t.Parallel()

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
                    access(all) contract Foo {
                        access(all) let field: @AnyResource
                        init() {
                            self.field <- create R()
                        }
                    }

                    access(all) resource R {
                        access(all) let a: Int
                        init() {
                            self.a = 5
                        }
                    }
                `,
			ParseAndCheckOptions{
				Location: utils.ImportedLocation,
			},
		)

		require.NoError(t, err)

		_, err = ParseAndCheckWithOptions(
			t,
			`
            import Foo from "imported"

            access(all) fun test() {
                let xRef = &Foo.field as &AnyResource
                xRef
            }
        `,
			ParseAndCheckOptions{
				Config: &sema.Config{
					ImportHandler: func(*sema.Checker, common.Location, ast.Range) (sema.Import, error) {
						return sema.ElaborationImport{
							Elaboration: importedChecker.Elaboration,
						}, nil
					},
				},
			},
		)

		require.NoError(t, err)
	})

	t.Run("self as reference", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }

                access(all) fun test() {
                    let xRef = &self as &R
                    xRef.a
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("contract field nested ref", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) contract Test {
                access(all) let a: @{UInt64: {UInt64: Test.R}}

                init() {
                    self.a <- {}
                }

                access(all) resource R {
                    access(all) fun test() {
                        if let storage = &Test.a[0] as &{UInt64: Test.R}? {
                            let nftRef = storage[0]!
                            nftRef
                        }
                    }
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("non resource refs", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) contract Test {
                access(all) resource R {
                    access(all) fun test () {
                        let sourceRefNFTs: {UInt64: &Test.R} = {}
                        let sourceNFTs: @[Test.R] <- []

                        while true {
                            let nft <- create Test.R()
                            let nftRef = &nft as &Test.R
                            sourceRefNFTs[nftRef.uuid] = nftRef
                            sourceNFTs.append(<- nft)
                        }

                        let nftRef = sourceRefNFTs[0]!
                        nftRef

                        destroy sourceNFTs
                    }

                    access(all) fun bar(): Bool {
                        return true
                    }
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("non resource refs param", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) contract Test {
                access(all) resource R {
                    access(all) fun test(packList: &[Test.R]) {
                        var i = 0
                        while i < packList.length {
                            let pack = packList[i]
                            pack
                            i = i + 1
                        }

                        return
                    }
                }
            }

            `,
		)

		require.NoError(t, err)
	})

	t.Run("partial invalidation", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let xRef = &x as &R
                if true {
                    destroy x
                } else {
                    // nothing
                }
                xRef.a

                destroy x
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 3)

		var invalidatedRefError *sema.InvalidatedResourceReferenceError
		assert.ErrorAs(t, errors[0], &invalidatedRefError)

		var resourceUseAfterInvalidationErr *sema.ResourceUseAfterInvalidationError
		assert.ErrorAs(t, errors[1], &resourceUseAfterInvalidationErr)

		var resourceLossErr *sema.ResourceLossError
		assert.ErrorAs(t, errors[2], &resourceLossErr)
	})

	t.Run("nil coalescing lhs", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x: @R? <- create R()
                let ref = (&x as &R?) ?? nil
                destroy x
                ref!.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("nil coalescing rhs", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = nil ?? (&y as &R?)
                destroy y
                ref!.a
                destroy x
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 2)

		typeMismatchError := &sema.TypeMismatchError{}
		assert.ErrorAs(t, errors[0], &typeMismatchError)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
	})

	t.Run("nil coalescing both sides", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = (&x as &R?) ?? (&y as &R?)
                destroy y
                destroy x
                ref!.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 3)

		typeMismatchError := &sema.TypeMismatchError{}
		assert.ErrorAs(t, errors[0], &typeMismatchError)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
		assert.ErrorAs(t, errors[2], &invalidatedRefError)
	})

	t.Run("nil coalescing nested", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()
                let z: @R? <- create R()

                let ref1 = (&x as &R?) ?? ((&y as &R?) ?? (&z as &R?))
                let ref2 = ref1
                destroy y
                destroy x
                destroy z
                ref2!.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 4)

		typeMismatchError := &sema.TypeMismatchError{}
		assert.ErrorAs(t, errors[0], &typeMismatchError)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
		assert.ErrorAs(t, errors[2], &invalidatedRefError)
		assert.ErrorAs(t, errors[3], &invalidatedRefError)
	})

	t.Run("ref assignment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                var ref1: &R? = nil
                ref1 = &x as &R

                destroy x
                ref1!.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("ref assignment non resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x = S()
                var ref1: &S? = nil
                ref1 = &x as &S
                consume(x)
                ref1!.a
            }

            access(all) fun consume(_ s:S) {}

            access(all) struct S {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("ref assignment chain", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let ref1 = &x as &R
                let ref2 = ref1
                let ref3 = ref2
                destroy x
                ref3.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("ref target is field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let r <- create R()
                let s = S()

                s.setB(&r as &R)
                destroy r
                s.b!.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }

            access(all) struct S {
                access(all) var b: &R?

                access(all) fun setB(_ b: &R) {
                    self.b = b
                }

                init() {
                    self.b = nil
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("ref source is field", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let r <- create R()
                let s = S()
                s.setB(&r as &R)

                let x = s.b!
                destroy r
                x.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }

            access(all) struct S {
                access(all) var b: &R?

                access(all) fun setB(_ b: &R) {
                    self.b = b
                }

                init() {
                    self.b = nil
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("conditional expr lhs", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x: @R? <- create R()
                let ref = true ? (&x as &R?) : nil
                destroy x
                ref!.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("conditional expr rhs", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = true ? nil : (&y as &R?)
                destroy y
                ref!.a
                destroy x
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 2)

		typeMismatchError := &sema.TypeMismatchError{}
		assert.ErrorAs(t, errors[0], &typeMismatchError)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
	})

	t.Run("conditional expr both sides", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = true ? (&x as &R?) : (&y as &R?)
                destroy y
                destroy x
                ref!.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 3)

		typeMismatchError := &sema.TypeMismatchError{}
		assert.ErrorAs(t, errors[0], &typeMismatchError)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
		assert.ErrorAs(t, errors[2], &invalidatedRefError)
	})

	t.Run("error notes", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let x <- create R()
                let xRef = &x as &R
                destroy x
                xRef.a
            }

            access(all) resource R {
                access(all) let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)

		errorNotes := invalidatedRefError.ErrorNotes()
		require.Len(t, errorNotes, 1)

		require.IsType(t, errorNotes[0], sema.PreviousResourceInvalidationNote{})
		prevInvalidationNote := errorNotes[0].(sema.PreviousResourceInvalidationNote)

		assert.Equal(
			t,
			prevInvalidationNote.Range.StartPos,
			ast.Position{
				Offset: 134,
				Line:   5,
				Column: 24,
			})
		assert.Equal(
			t,
			prevInvalidationNote.Range.EndPos,
			ast.Position{
				Offset: 134,
				Line:   5,
				Column: 24,
			})
	})

	t.Run("create ref by field access", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let foo <- create Foo()
                var fooRef = &foo as &Foo

                let barRef = fooRef.bar
                destroy foo
                barRef.id
            }

            resource Foo {
                let bar: @Bar
                init() {
                    self.bar <-create Bar()
                }
            }

            resource Bar {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("create ref by index access", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let array <- [<- create Foo()]
                var arrayRef = &array as &[Foo]

                let fooRef = arrayRef[0]
                destroy array
                fooRef.id
            }

            resource Foo {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})

	t.Run("create ref by field and index access", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            access(all) fun test() {
                let array <- [<- create Foo()]
                var arrayRef = &array as &[Foo]

                let barRef = arrayRef[0].bar
                destroy array
                barRef.id
            }

            resource Foo {
                let bar: @Bar
                init() {
                    self.bar <-create Bar()
                }
            }

            resource Bar {
                let id: UInt8
                init() {
                    self.id = 1
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 1)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
	})
}

func TestCheckReferenceUseAfterCopy(t *testing.T) {

	t.Parallel()

	t.Run("resource, field write", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var name: String
              init(name: String) {
                  self.name = name
              }
          }

          fun test() {
              let r <- create R(name: "1")
              let ref = &r as &R
              let container <- [<-r]
              ref.name = "2"
              destroy container
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})

	t.Run("resource, field read", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {
              var name: String
              init(name: String) {
                  self.name = name
              }
          }

          fun test(): String {
              let r <- create R(name: "1")
              let ref = &r as &R
              let container <- [<-r]
              let name = ref.name
              destroy container
              return name
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})

	t.Run("resource array, insert", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as auth(Mutate) &[R]
              let container <- [<-rs]
              ref.insert(at: 1, <-create R())
              destroy container
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})

	t.Run("resource array, append", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as auth(Mutate) &[R]
              let container <- [<-rs]
              ref.append(<-create R())
              destroy container
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})

	t.Run("resource array, get/set", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
              let container <- [<-rs]
              var r <- create R()
              ref[0] <-> r
              destroy container
              destroy r
          }
        `)

		errs := RequireCheckerErrors(t, err, 4)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)

		unauthorizedReferenceAssignmentError := &sema.UnauthorizedReferenceAssignmentError{}
		assert.ErrorAs(t, errs[1], &unauthorizedReferenceAssignmentError)

		assert.ErrorAs(t, errs[2], &invalidatedRefError)

		typeMismatchError := &sema.TypeMismatchError{}
		assert.ErrorAs(t, errs[3], &typeMismatchError)

	})

	t.Run("resource array, remove", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as auth(Mutate) &[R]
              let container <- [<-rs]
              let r <- ref.remove(at: 0)
              destroy container
              destroy r
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})

	t.Run("resource dictionary, insert", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- {0: <-create R()}
              let ref = &rs as &{Int: R}
              let container <- [<-rs]
              ref[1] <-! create R()
              destroy container
          }
        `)

		errs := RequireCheckerErrors(t, err, 2)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)

		unauthorizedReferenceAssignmentError := &sema.UnauthorizedReferenceAssignmentError{}
		assert.ErrorAs(t, errs[1], &unauthorizedReferenceAssignmentError)
	})

	t.Run("resource dictionary, remove", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- {0: <-create R()}
              let ref = &rs as auth(Remove) &{Int: R}
              let container <- [<-rs]
              let r <- ref.remove(key: 0)
              destroy container
              destroy r
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})

	t.Run("attachments", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            attachment A for R {
                access(all) var id: UInt8
                init() {
                    self.id = 1
                }
            }

            fun test() {
                let r <- create R()
                let r2 <- attach A() to <-r

                let a = r2[A]!
                destroy r2

                // Access attachment ref, after destroying the resource
                a.id
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})
}

func TestCheckResourceReferenceMethodInvocationAfterMove(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
        resource Foo {

            let id: UInt8

            init() {
                self.id = 12
            }

            access(all) fun something() {}
        }

        fun main() {
            var foo <- create Foo()
            var fooRef = &foo as &Foo

            // Invocation should not un-track the reference
            fooRef.something()

            // Moving the resource should update the tracking
            var newFoo <- foo

            fooRef.id

            destroy newFoo
        }
    `)

	errs := RequireCheckerErrors(t, err, 1)
	invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
	assert.ErrorAs(t, errs[0], &invalidatedRefError)
}

func TestCheckReferenceCreationWithInvalidType(t *testing.T) {

	t.Parallel()

	t.Run("invalid reference type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            let foo: AnyStruct? = nil
            let x = &foo as &Foo
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var notDeclaredError *sema.NotDeclaredError
		require.ErrorAs(t, errs[0], &notDeclaredError)
	})

	t.Run("valid non-reference type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct Foo {}

            let foo: AnyStruct? = nil
            let x = &foo as Foo
        `)

		errs := RequireCheckerErrors(t, err, 1)

		var nonReferenceTypeReferenceError *sema.NonReferenceTypeReferenceError
		require.ErrorAs(t, errs[0], &nonReferenceTypeReferenceError)
	})
}

func TestCheckResourceReferenceFieldNilAssignment(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, `
        access(all) resource Outer {
            access(all) var inner : @Inner?

            init(_ v: @Inner){
                self.inner <- v
                var outerRef = &self as &Outer
                outerRef.inner = nil
            }
        }

        access(all) resource Inner {}

        fun main() {
            let inner <- create Inner()
            let outer <- create Outer(<- inner)
            destroy outer
        }
    `)

	errors := RequireCheckerErrors(t, err, 2)
	require.IsType(t, &sema.IncorrectTransferOperationError{}, errors[0])
	require.IsType(t, &sema.InvalidResourceAssignmentError{}, errors[1])
}

func TestCheckResourceReferenceIndexNilAssignment(t *testing.T) {
	t.Parallel()

	t.Run("one level", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            access(all) resource Foo {}

            fun main() {
                let array: @[Foo?] <- [<- create Foo()]
                let arrayRef = &array as auth(Mutate) &[Foo?]

                arrayRef[0] = nil

                destroy array
            }
        `)

		errors := RequireCheckerErrors(t, err, 2)
		require.IsType(t, &sema.IncorrectTransferOperationError{}, errors[0])
		require.IsType(t, &sema.InvalidResourceAssignmentError{}, errors[1])
	})

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            access(all) resource Foo {}

            fun main() {
                let array: @[[Foo?]] <- [<- [<- create Foo()]]
                let arrayRef = &array as auth(Mutate) &[[Foo?]]

                arrayRef[0][0] = nil

                destroy array
            }
        `)

		errors := RequireCheckerErrors(t, err, 3)
		require.IsType(t, &sema.UnauthorizedReferenceAssignmentError{}, errors[0])
		require.IsType(t, &sema.IncorrectTransferOperationError{}, errors[1])
		require.IsType(t, &sema.InvalidResourceAssignmentError{}, errors[2])
	})
}

func TestCheckNestedReference(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                let x = &1 as &Int
                let y = &x as & &Int
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.NestedReferenceError{}, errors[0])
	})

	t.Run("type of underlying value checked", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                let x = &1 as &Int
                let y = &x as &AnyStruct
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.NestedReferenceError{}, errors[0])
	})

	t.Run("optional", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                let x: &Int? = &1 as &Int
                let y = &x as &AnyStruct?
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.NestedReferenceError{}, errors[0])
	})

	t.Run("nested optional", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            fun main() {
                let x: &Int?? = &1 as &Int
                let y = &x as &AnyStruct?
            }
        `)

		errors := RequireCheckerErrors(t, err, 1)
		require.IsType(t, &sema.NestedReferenceError{}, errors[0])
	})
}

func TestCheckDereference(t *testing.T) {

	t.Parallel()

	type testCase struct {
		ty          sema.Type
		initializer string
	}

	runValidTestCase := func(t *testing.T, name, code string, expectedTy sema.Type) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheck(t, code)

			require.NoError(t, err)

			yType := RequireGlobalValue(t, checker.Elaboration, "y")

			assert.Equal(t,
				expectedTy,
				yType,
			)
		})
	}

	runInvalidTestCase := func(t *testing.T, name, code string) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseAndCheck(t, code)

			errs := RequireCheckerErrors(t, err, 1)
			assert.IsType(t, &sema.InvalidUnaryOperandError{}, errs[0])
		})
	}

	t.Run("Numeric Types", func(t *testing.T) {
		t.Parallel()

		for _, typ := range sema.AllIntegerTypes {
			integerType := typ
			typString := typ.QualifiedString()

			runValidTestCase(
				t,
				typString,
				fmt.Sprintf(
					`
                      let x: &%[1]s = &1
                      let y: %[1]s = *x
                    `,
					integerType,
				),
				integerType,
			)
		}

		for _, typ := range sema.AllFixedPointTypes {
			fixedPointType := typ
			typString := typ.QualifiedString()

			runValidTestCase(
				t,
				typString,
				fmt.Sprintf(
					`
                      let x: &%[1]s = &1.0
                      let y: %[1]s = *x
                    `,
					fixedPointType,
				),
				fixedPointType,
			)
		}
	})

	t.Run("Simple types", func(t *testing.T) {
		t.Parallel()

		for _, testCase := range []testCase{
			{
				ty:          sema.CharacterType,
				initializer: "\"\\u{FC}\"",
			},
			{
				ty:          sema.StringType,
				initializer: "\"\\u{FC}\"",
			},
			{
				ty:          sema.BoolType,
				initializer: "false",
			},
			{
				ty:          sema.TheAddressType,
				initializer: "0x0000000000000001",
			},
			{
				ty:          sema.PrivatePathType,
				initializer: "/private/foo",
			},
			{
				ty:          sema.PublicPathType,
				initializer: "/public/foo",
			},
		} {
			runValidTestCase(
				t,
				testCase.ty.QualifiedString(),
				fmt.Sprintf(
					`
                      let value: %[1]s = %[2]s
                      let x: &%[1]s = &value
                      let y: %[1]s = *x
                    `,
					testCase.ty,
					testCase.initializer,
				),
				testCase.ty,
			)
		}
	})

	t.Run("Arrays", func(t *testing.T) {
		t.Parallel()

		for _, testCase := range []testCase{
			{
				ty:          &sema.VariableSizedType{Type: sema.IntType},
				initializer: "[1, 2, 3]",
			},
			{
				ty:          &sema.VariableSizedType{Type: sema.Fix64Type},
				initializer: "[1.0, 5.7]",
			},
			{
				ty:          &sema.VariableSizedType{Type: sema.StringType},
				initializer: "[\"abc\", \"def\"]",
			},
			{
				ty: &sema.VariableSizedType{
					Type: &sema.VariableSizedType{
						Type: sema.StringType,
					},
				},
				initializer: "[ [\"abc\", \"def\"], [\"xyz\"]]",
			},
			{
				ty: &sema.VariableSizedType{
					Type: &sema.DictionaryType{
						KeyType:   sema.IntType,
						ValueType: sema.StringType,
					}},
				initializer: "[{1: \"abc\", 2: \"def\"}, {3: \"xyz\"}]",
			},
			{
				ty:          &sema.ConstantSizedType{Type: sema.IntType, Size: 3},
				initializer: "[1, 2, 3]",
			},
			{
				ty:          &sema.ConstantSizedType{Type: sema.Fix64Type, Size: 2},
				initializer: "[1.0, 5.7]",
			},
			{
				ty:          &sema.ConstantSizedType{Type: sema.StringType, Size: 2},
				initializer: "[\"abc\", \"def\"]",
			},
			{
				ty: &sema.ConstantSizedType{
					Type: &sema.VariableSizedType{
						Type: sema.StringType,
					},
					Size: 2,
				},
				initializer: "[ [\"abc\", \"def\"], [\"xyz\"]]",
			},
			{
				ty: &sema.ConstantSizedType{
					Type: &sema.DictionaryType{
						KeyType:   sema.IntType,
						ValueType: sema.StringType,
					},
					Size: 1,
				},
				initializer: "[{1: \"abc\", 2: \"def\"}]",
			},
		} {
			runValidTestCase(
				t,
				testCase.ty.QualifiedString(),
				fmt.Sprintf(
					`
                      let value: %[1]s = %[2]s
                      let x: &%[1]s = &value
                      let y: %[1]s = *x
                    `,
					testCase.ty,
					testCase.initializer,
				),
				testCase.ty,
			)
		}

		// Arrays of non-primitives cannot be dereferenced.
		runInvalidTestCase(
			t,
			"[Struct]",
			`
              struct S{}

              fun test() {
                  let value: [S] = [S(), S()]
                  let x: &[S] = &value
                  let y: [S] = *x
              }
            `,
		)

		runInvalidTestCase(
			t,
			"[Struct; 3]",
			`
              struct S{}

              fun test() {
                  let value: [S; 3] = [S(),S(),S()]
                  let x: &[S; 3] = &value
                  let y: [S; 3] = *x
              }
            `,
		)
	})

	t.Run("Dictionary", func(t *testing.T) {
		t.Parallel()

		for _, testCase := range []testCase{
			{
				ty:          &sema.DictionaryType{KeyType: sema.IntType, ValueType: sema.IntType},
				initializer: "{1: 1, 2: 2, 3: 3}",
			},
			{
				ty:          &sema.DictionaryType{KeyType: sema.IntType, ValueType: sema.Fix64Type},
				initializer: "{1: 1.2, 2: 2.4, 3: 3.0}",
			},
			{
				ty:          &sema.DictionaryType{KeyType: sema.StringType, ValueType: sema.StringType},
				initializer: "{\"123\": \"abc\", \"456\": \"def\"}",
			},
			{
				ty: &sema.DictionaryType{
					KeyType: sema.StringType,
					ValueType: &sema.VariableSizedType{
						Type: sema.IntType,
					},
				},
				initializer: "{\"123\": [1, 2, 3], \"456\": [4, 5, 6]}",
			},
			{
				ty: &sema.DictionaryType{
					KeyType: sema.StringType,
					ValueType: &sema.ConstantSizedType{
						Type: sema.IntType,
						Size: 3,
					},
				},
				initializer: "{\"123\": [1, 2, 3], \"456\": [4, 5, 6]}",
			},
		} {
			runValidTestCase(
				t,
				testCase.ty.QualifiedString(),
				fmt.Sprintf(
					`
                      let value: %[1]s = %[2]s
                      let x: &%[1]s = &value
                      let y: %[1]s = *x
                    `,
					testCase.ty,
					testCase.initializer,
				),
				testCase.ty,
			)
		}

		// Dictionaries with value as non-primitive cannot be dereferenced.
		runInvalidTestCase(
			t,
			"{Int: Struct}",
			`
              struct S{}

              fun test() {
                  let value: {Int: S} = { 1: S(), 2: S() }
                  let x: &{Int: S} = &value
                  let y: {Int: S} = *x
              }
            `,
		)
	})

	runInvalidTestCase(
		t,
		"Resource",
		`
          resource interface I {
              fun foo()
          }

          resource R: I {
              fun foo() {}
          }

          fun test() {
              let r <- create R()
              let ref = &r as &{I}
              let deref <- *ref
              destroy r
              destroy deref
          }
        `,
	)

	runInvalidTestCase(
		t,
		"Struct",
		`
          struct S{}

          fun test() {
              let s = S()
              let ref = &s as &S
              let deref = *ref
          }
        `,
	)

	t.Run("built-in", func(t *testing.T) {

		t.Parallel()

		runInvalidTestCase(
			t,
			"Account",
			`
              fun test(ref: &Account): Account {
                  return *ref
              }
            `,
		)
	})

	t.Run("Optional", func(t *testing.T) {
		t.Parallel()

		runValidTestCase(
			t,
			"valid",
			`
              let ref: &Int? = &1 as &Int
              let y = *ref
            `,
			&sema.OptionalType{
				Type: sema.IntType,
			},
		)

		runInvalidTestCase(
			t,
			"invalid",
			`
              struct S {}

              fun test() {
                  let s = S()
                  let ref: &S? = &s as &S
                  let deref = *ref
              }
            `,
		)
	})
}
