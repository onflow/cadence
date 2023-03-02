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
              let x: auth &Int = &1
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

		t.Run("non-auth", func(t *testing.T) {

			t.Parallel()

			_, err := ParseAndCheck(t, `
              let x = &1 as auth &Int
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
          let x = &1 as &Int as auth &Int
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
			Type: sema.IntType,
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
				Type: rType,
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
				Type: sType,
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

func TestCheckReferenceExpressionWithRestrictedAnyResultType(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource interface I {}
          resource R: I {}

          let r <- create R()
          let ref = &r as &AnyResource{I}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct interface I {}
          struct S: I {}

          let s = S()
          let ref = &s as &AnyStruct{I}
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
              var other <- create R()
              ref[0] <-> other
              destroy rs
              destroy other
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
              var other = S()
              ref[0] <-> other
          }
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidReferenceResourceLoss(t *testing.T) {

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
              let ref = &r as &AnyResource{I}
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
              let ref = &s as &AnyStruct{I}
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
              let ref = &r as &AnyResource{I}
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
              let ref = &s as &AnyStruct{I}
              ref.foo()
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
	})
}

func TestCheckReferenceExpressionReferenceType(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, auth bool, kind common.CompositeKind) {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf("%s, auth: %v", kind.Name(), auth)

		t.Run(testName, func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      %[1]s T {}

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

			require.Equal(t,
				&sema.ReferenceType{
					Authorized: auth,
					Type:       tType,
				},
				refValueType,
			)
		})
	}

	for _, kind := range []common.CompositeKind{
		common.CompositeKindResource,
		common.CompositeKindStructure,
	} {
		for _, auth := range []bool{true, false} {
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
					Type: sema.IntType,
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

	t.Run("upcast to optional", func(t *testing.T) {

		t.Parallel()

		checker, err := ParseAndCheck(t, `
          let i: Int = 1
          let ref = &i as &Int?
        `)

		require.NoError(t, err)
		refValueType := RequireGlobalValue(t, checker.Elaboration, "ref")

		assert.Equal(t,
			&sema.OptionalType{
				Type: &sema.ReferenceType{
					Type: sema.IntType,
				},
			},
			refValueType,
		)
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
			Type: sema.IntType,
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

	assert.IsType(t, &sema.AmbiguousRestrictedTypeError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidReferenceExpressionNonReferenceAnyResource(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let y = &x as AnyResource{}
    `)

	errs := RequireCheckerErrors(t, err, 4)

	assert.IsType(t, &sema.MissingResourceAnnotationError{}, errs[0])
	assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[1])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[2])
	assert.IsType(t, &sema.IncorrectTransferOperationError{}, errs[3])
}

func TestCheckInvalidReferenceExpressionNonReferenceAnyStruct(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let y = &x as AnyStruct{}
    `)

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[0])
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
		pub struct S {
			pub let foo: Number
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
		pub struct S {
			pub let foo: Number
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
		pub struct S {
			pub let foo: Number
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

func TestCheckReferenceTypeImplicitConformance(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          contract interface CI {
              struct S {}
          }

          contract C: CI {
              struct S {}
          }

          let s = C.S()

          let refS: &CI.S = &s as &C.S
        `)

		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `

          contract interface CI {
              struct S {}
          }

          contract C {
              struct S {}
          }

          let s = C.S()

          let refS: &CI.S = &s as &C.S
        `)

		errs := RequireCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckInvalidatedReferenceUse(t *testing.T) {

	t.Parallel()

	t.Run("no errors", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            pub fun test() {
                let x <- create R()
                let xRef = &x as &R
                xRef.a
                destroy x
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x <- create R()
                let xRef = &x as &R
                destroy x
                xRef.a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x <- [<-create R()]
                let xRef = &x as &[R]
                destroy x
                xRef[0].a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x <- {1: <- create R()}
                let xRef = &x as &{Int: R}
                destroy x
                xRef[1]?.a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x <- create R()
                let xRef = &x as &R
                consume(<-x)
                xRef.a
            }

            pub fun consume(_ r: @AnyResource) {
                destroy r
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x <- [<-create R()]
                let xRef = &x as &[R]
                consume(<-x)
                xRef[0].a
            }

            pub fun consume(_ r: @AnyResource) {
                destroy r
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x <- {1: <- create R()}
                let xRef = &x as &{Int: R}
                consume(<-x)
                xRef[1]?.a
            }

            pub fun consume(_ r: @AnyResource) {
                destroy r
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                var x <- create R()
                var y <- create R()
                let xRef = &x as &R
                x <-> y
                destroy x
                destroy y
                xRef.a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
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

            pub resource R {
                pub let a: Int

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

		_, err := ParseAndCheckAccount(t,
			`
            pub fun test() {
                authAccount.save(<-[<-create R()], to: /storage/a)

                let collectionRef = authAccount.borrow<&[R]>(from: /storage/a)!
                let ref = &collectionRef[0] as &R

                let collection <- authAccount.load<@[R]>(from: /storage/a)!
                authAccount.save(<- collection, to: /storage/b)

                ref.a = 2
            }

            pub resource R {
                pub(set) var a: Int

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
            pub fun test() {
                let f = fun() {
                    let x <- create R()
                    let xRef = &x as &R
                    destroy x
                    xRef.a
                }

                f()
            }

            pub resource R {
                pub let a: Int

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
            pub contract Test {
                priv var x: @R
                init() {
                    self.x <- create R()
                }

                pub fun test() {
                    let xRef = &self.x as &R
                    xRef.a
                }
            }

            pub resource R {
                pub let a: Int

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
            pub contract Test {
                priv var x: @R
                init() {
                    self.x <- create R()
                }

                pub fun test() {
                    let xRef = &Test.x as &R
                    xRef.a
                }
            }

            pub resource R {
                pub let a: Int
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
            pub fun test() {
                var r: @{UInt64: {UInt64: [R]}} <- {}
                let ref1 = (&r[0] as &{UInt64: [R]}?)!
                let ref2 = (&ref1[0] as &[R]?)!
                let ref3 = &ref2[0] as &R
                ref3.a

                destroy r
            }

            pub resource R {
                pub let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		require.NoError(t, err)
	})

	t.Run("ref to ref invalid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            pub fun test() {
                var r: @{UInt64: {UInt64: [R]}} <- {}
                let ref1 = (&r[0] as &{UInt64: [R]}?)!
                let ref2 = (&ref1[0] as &[R]?)!
                let ref3 = &ref2[0] as &R
                destroy r
                ref3.a
            }

            pub resource R {
                pub let a: Int
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
            pub fun test() {
                let x <- create R()
                let xRef = (&x as &R?)!
                destroy x
                xRef.a
            }

            pub resource R {
                pub let a: Int
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

	t.Run("contract field ref", func(t *testing.T) {

		t.Parallel()

		importedChecker, err := ParseAndCheckWithOptions(t,
			`
                    pub contract Foo {
                        pub let field: @AnyResource
                        init() {
                            self.field <- create R()
                        }
                    }

                    pub resource R {
                        pub let a: Int
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

            pub fun test() {
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
            pub resource R {
                pub let a: Int

                init() {
                    self.a = 5
                }

                pub fun test() {
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
            pub contract Test {
                pub let a: @{UInt64: {UInt64: Test.R}}

                init() {
                    self.a <- {}
                }

                pub resource R {
                    pub fun test() {
                        if let storage = &Test.a[0] as &{UInt64: Test.R}? {
                            let nftRef = (&storage[0] as &Test.R?)!
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
            pub contract Test {
                pub resource R {
                    pub fun test () {
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

                    pub fun bar(): Bool {
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
            pub contract Test {
                pub resource R {
                    pub fun test(packList: &[Test.R]) {
                        var i = 0;
                        while i < packList.length {
                            let pack = &packList[i] as &Test.R;
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
            pub fun test() {
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

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x: @R? <- create R()
                let ref = (&x as &R?) ?? nil
                destroy x
                ref!.a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = nil ?? (&y as &R?)
                destroy y
                ref!.a
                destroy x
            }

            pub resource R {
                pub let a: Int

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

	t.Run("nil coalescing both sides", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            pub fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = (&x as &R?) ?? (&y as &R?)
                destroy y
                destroy x
                ref!.a
            }

            pub resource R {
                pub let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 2)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
	})

	t.Run("nil coalescing nested", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            pub fun test() {
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

            pub resource R {
                pub let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 3)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
		assert.ErrorAs(t, errors[2], &invalidatedRefError)
	})

	t.Run("ref assignment", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            pub fun test() {
                let x <- create R()
                var ref1: &R? = nil
                ref1 = &x as &R

                destroy x
                ref1!.a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x = S()
                var ref1: &S? = nil
                ref1 = &x as &S
                consume(x)
                ref1!.a
            }

            pub fun consume(_ s:S) {}

            pub struct S {
                pub let a: Int

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
            pub fun test() {
                let x <- create R()
                let ref1 = &x as &R
                let ref2 = ref1
                let ref3 = ref2
                destroy x
                ref3.a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let r <- create R()
                let s = S()

                s.b = &r as &R
                destroy r
                s.b!.a
            }

            pub resource R {
                pub let a: Int

                init() {
                    self.a = 5
                }
            }

            pub struct S {
                pub(set) var b: &R?

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
            pub fun test() {
                let r <- create R()
                let s = S()
                s.b = &r as &R

                let x = s.b!
                destroy r
                x.a
            }

            pub resource R {
                pub let a: Int

                init() {
                    self.a = 5
                }
            }

            pub struct S {
                pub(set) var b: &R?

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
            pub fun test() {
                let x: @R? <- create R()
                let ref = true ? (&x as &R?) : nil
                destroy x
                ref!.a
            }

            pub resource R {
                pub let a: Int

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
            pub fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = true ? nil : (&y as &R?)
                destroy y
                ref!.a
                destroy x
            }

            pub resource R {
                pub let a: Int

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

	t.Run("conditional expr both sides", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            pub fun test() {
                let x: @R? <- create R()
                let y: @R <- create R()

                let ref = true ? (&x as &R?) : (&y as &R?)
                destroy y
                destroy x
                ref!.a
            }

            pub resource R {
                pub let a: Int

                init() {
                    self.a = 5
                }
            }
            `,
		)

		errors := RequireCheckerErrors(t, err, 2)

		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errors[0], &invalidatedRefError)
		assert.ErrorAs(t, errors[1], &invalidatedRefError)
	})

	t.Run("error notes", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t,
			`
            pub fun test() {
                let x <- create R()
                let xRef = &x as &R
                destroy x
                xRef.a
            }

            pub resource R {
                pub let a: Int

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
				Offset: 126,
				Line:   5,
				Column: 24,
			})
		assert.Equal(
			t,
			prevInvalidationNote.Range.EndPos,
			ast.Position{
				Offset: 126,
				Line:   5,
				Column: 24,
			})
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
              let ref = &rs as &[R]
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
              let ref = &rs as &[R]
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

		errs := RequireCheckerErrors(t, err, 2)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
		assert.ErrorAs(t, errs[1], &invalidatedRefError)
	})

	t.Run("resource array, remove", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- [<-create R()]
              let ref = &rs as &[R]
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

		errs := RequireCheckerErrors(t, err, 1)
		invalidatedRefError := &sema.InvalidatedResourceReferenceError{}
		assert.ErrorAs(t, errs[0], &invalidatedRefError)
	})

	t.Run("resource dictionary, remove", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test() {
              let rs <- {0: <-create R()}
              let ref = &rs as &{Int: R}
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
}
