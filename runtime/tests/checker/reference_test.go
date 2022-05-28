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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

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

	errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          let s = S()
          let ref = &s as &X
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i = 1
          let ref = &i as &X
        `)

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

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

	errs := ExpectCheckerErrors(t, err, 1)

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

	errs := ExpectCheckerErrors(t, err, 1)

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

	errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
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

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
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

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          struct S {}

          let s: S? = S()
          let ref = &s as &S
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("non-composite", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i: Int? = 1
          let ref = &i as &Int
        `)

		errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)
		assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[0])
	})

	t.Run("mismatched type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
          let i: String? = ""
          let ref = &i as &Int?
        `)

		errs := ExpectCheckerErrors(t, err, 1)
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

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.AmbiguousRestrictedTypeError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidReferenceExpressionNonReferenceAnyResource(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let y = &x as AnyResource{}
    `)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidReferenceExpressionNonReferenceAnyStruct(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let y = &x as AnyStruct{}
    `)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[0])
	assert.IsType(t, &sema.NotDeclaredError{}, errs[1])
}

func TestCheckInvalidDictionaryAccessReference(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let xs: {Int: Int} = {}
      let ref = &xs[1] as &String
    `)

	errs := ExpectCheckerErrors(t, err, 1)

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

	errs := ExpectCheckerErrors(t, err, 1)

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

	errs := ExpectCheckerErrors(t, err, 1)

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

	errs := ExpectCheckerErrors(t, err, 1)

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

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}
