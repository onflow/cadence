package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/sema"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckReferenceTypeOuter(t *testing.T) {

	t.Run("resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: &[R]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          struct S {}

          fun test(s: &[S]) {}
        `)

		require.NoError(t, err)
	})
}

func TestCheckReferenceTypeInner(t *testing.T) {

	t.Run("resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: [&R]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          struct S {}

          fun test(s: [&S]) {}
        `)

		require.NoError(t, err)
	})

}

func TestCheckNestedReferenceType(t *testing.T) {

	t.Run("resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource R {}

          fun test(r: &[&R]) {}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          struct S {}

          fun test(s: &[&S]) {}
        `)

		require.NoError(t, err)
	})
}

func TestCheckInvalidReferenceType(t *testing.T) {

	_, err := ParseAndCheck(t, `
      fun test(t: &T) {}
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])

}

func TestCheckReferenceExpressionWithCompositeResultType(t *testing.T) {

	t.Run("resource", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
          resource R {}

          let r <- create R()
          let ref = &r as &R
        `)

		require.NoError(t, err)

		refValueType := checker.GlobalValues["ref"].Type

		assert.IsType(t,
			&sema.ReferenceType{},
			refValueType,
		)

		assert.IsType(t,
			&sema.CompositeType{},
			refValueType.(*sema.ReferenceType).Type,
		)
	})

	t.Run("struct", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
          struct S {}

          let s = S()
          let ref = &s as &S
        `)

		require.NoError(t, err)

		refValueType := checker.GlobalValues["ref"].Type

		assert.IsType(t,
			&sema.ReferenceType{},
			refValueType,
		)

		assert.IsType(t,
			&sema.CompositeType{},
			refValueType.(*sema.ReferenceType).Type,
		)
	})
}

func TestCheckReferenceExpressionWithInterfaceResultType(t *testing.T) {

	t.Run("resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource interface I {}
          resource R: I {}

          let r <- create R()
          let ref = &r as &I
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          struct interface I {}
          struct S: I {}

          let s = S()
          let ref = &s as &I
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckReferenceExpressionWithRestrictedAnyResultType(t *testing.T) {

	t.Run("resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource interface I {}
          resource R: I {}

          let r <- create R()
          let ref = &r as &AnyResource{I}
        `)

		require.NoError(t, err)
	})

	t.Run("struct", func(t *testing.T) {

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

	t.Run("resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource R {}

          let r <- create R() 
          let ref = &r as &X
        `,
		)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})

	t.Run("struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          struct S {}

          let s = S() 
          let ref = &s as &X
        `,
		)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	})
}

func TestCheckInvalidReferenceExpressionStorageIndexType(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource R {}

          let ref = &storage[X] as &R
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
}

func TestCheckInvalidReferenceExpressionTypeMismatchStructResource(t *testing.T) {

	t.Run("struct / resource", func(t *testing.T) {

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

	_, err := ParseAndCheckStorage(t, `
          struct S {}
          struct T {}

          let ref = &storage[S] as &T
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckInvalidReferenceExpressionTypeMismatchDifferentResources(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource R {}
          resource T {}

          let ref = &storage[R] as &T
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckReferenceToNonStorage(t *testing.T) {

	t.Run("struct variable", func(t *testing.T) {

		_, err := ParseAndCheckStorage(t, `
          struct S {}

          let s = S()
          let ref = &s as &S
        `,
		)

		require.NoError(t, err)
	})

	t.Run("resource variable", func(t *testing.T) {

		_, err := ParseAndCheckStorage(t, `
          resource R {}

          let r <- create R()
          let ref = &r as &R
        `,
		)

		require.NoError(t, err)
	})

	t.Run("resource array indexing", func(t *testing.T) {

		_, err := ParseAndCheckStorage(t, `
          resource R {}

          let rs <- [<-create R()]
          let ref = &rs[0] as &R
        `,
		)

		require.NoError(t, err)
	})
}

func TestCheckReferenceUse(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
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
              var r: @R? <- create R()
              storage[R] <-> r
              // there was no old value, but it must be discarded
              destroy r

              let ref = &storage[R] as &R
              ref.x = 1
              let x1 = ref.x
              ref.setX(2)
              let x2 = ref.x
              return [x1, x2]
          }
        `,
	)

	require.NoError(t, err)
}

func TestCheckReferenceUseArray(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
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
              var rs: @[R]? <- [<-create R()]
              storage[[R]] <-> rs
              // there was no old value, but it must be discarded
              destroy rs

              let ref = &storage[[R]] as &[R]
              ref[0].x = 1
              let x1 = ref[0].x
              ref[0].setX(2)
              let x2 = ref[0].x
              return [x1, x2]
          }
        `,
	)

	require.NoError(t, err)
}

func TestCheckReferenceIndexingIfReferencedIndexable(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource R {}

          fun test() {
              var rs: @[R]? <- [<-create R()]
              storage[[R]] <-> rs
              // there was no old value, but it must be discarded
              destroy rs

              let ref = &storage[[R]] as &[R]
              var other <- create R()
              ref[0] <-> other
              destroy other
          }
        `,
	)

	require.NoError(t, err)
}

func TestCheckInvalidReferenceResourceLoss(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource R {}

          fun test() {
              var rs: @[R]? <- [<-create R()]
              storage[[R]] <-> rs
              // there was no old value, but it must be discarded
              destroy rs

              let ref = &storage[[R]] as &[R]
              ref[0]
          }
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.ResourceLossError{}, errs[0])
}

func TestCheckInvalidReferenceIndexingIfReferencedNotIndexable(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource R {}

          fun test() {
              var r: @R? <- create R()
              storage[R] <-> r
              // there was no old value, but it must be discarded
              destroy r

              let ref = &storage[R] as &R
              ref[0]
          }
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotIndexableTypeError{}, errs[0])
}

func TestCheckResourceInterfaceReferenceFunctionCall(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource interface I {
              fun foo()
          }

          resource R: I {
              fun foo() {}
          }

          fun test() {
              var r: @R? <- create R()
              storage[R] <-> r
              // there was no old value, but it must be discarded
              destroy r

              let ref = &storage[R] as &AnyResource{I}
              ref.foo()
          }
        `,
	)

	require.NoError(t, err)
}

func TestCheckInvalidResourceInterfaceReferenceFunctionCall(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource interface I {}

          resource R: I {
              fun foo() {}
          }

          fun test() {
              var r: @R? <- create R()
              storage[R] <-> r
              // there was no old value, but it must be discarded
              destroy r

              let ref = &storage[R] as &AnyResource{I}
              ref.foo()
          }
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
}

func TestCheckReferenceExpressionReferenceType(t *testing.T) {

	t.Run("non-auth reference", func(t *testing.T) {

		checker, err := ParseAndCheckStorage(t, `
          resource R {}

          let ref = &storage[R] as &R
        `,
		)

		require.NoError(t, err)

		refValueType := checker.GlobalValues["ref"].Type

		require.IsType(t,
			&sema.ReferenceType{},
			refValueType,
		)

		referenceType := refValueType.(*sema.ReferenceType)

		assert.IsType(t,
			&sema.CompositeType{},
			referenceType.Type,
		)

		assert.False(t, referenceType.Authorized)
	})

	t.Run("auth reference", func(t *testing.T) {

		checker, err := ParseAndCheckStorage(t, `
          resource R {}

          let ref = &storage[R] as auth &R
        `,
		)

		require.NoError(t, err)

		refValueType := checker.GlobalValues["ref"].Type

		require.IsType(t,
			&sema.ReferenceType{},
			refValueType,
		)

		referenceType := refValueType.(*sema.ReferenceType)

		assert.IsType(t,
			&sema.CompositeType{},
			referenceType.Type,
		)

		assert.True(t, referenceType.Authorized)
	})
}

func TestCheckReferenceExpressionOfOptional(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          resource R {}

          let r: @R? <- create R()
          let ref = &r as &R
        `,
	)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.OptionalTypeReferenceError{}, errs[0])
	assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
}

func TestCheckInvalidReferenceExpressionNonReferenceAmbiguous(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          let y = &x as {}
        `,
	)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.AmbiguousRestrictedTypeError{}, errs[1])
}

func TestCheckInvalidReferenceExpressionNonReferenceAnyResource(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          let y = &x as AnyResource{}
        `,
	)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[1])
}

func TestCheckInvalidReferenceExpressionNonReferenceAnyStruct(t *testing.T) {

	_, err := ParseAndCheckStorage(t, `
          let y = &x as AnyStruct{}
        `,
	)

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.NotDeclaredError{}, errs[0])
	assert.IsType(t, &sema.NonReferenceTypeReferenceError{}, errs[1])
}
