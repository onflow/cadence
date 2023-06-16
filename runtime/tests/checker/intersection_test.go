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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckIntersectionType(t *testing.T) {

	t.Parallel()

	t.Run("resource: no types", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
            resource R {}

            let r: @R{} <- create R()
        `)

		require.NoError(t, err)
	})

	t.Run("struct: no types", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            let r: S{} = S()
        `)

		require.NoError(t, err)
	})

	t.Run("resource: one type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            let r: @R{I1} <- create R()
        `)

		require.NoError(t, err)
	})

	t.Run("struct: one type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let r: S{I1} = S()
        `)

		require.NoError(t, err)
	})

	t.Run("reference to resource type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            let r <- create R()
            let ref: &R{} = &r as &R
        `)

		require.NoError(t, err)
	})

	t.Run("reference to struct type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            let s = S()
            let ref: &S{} = &s as &S
        `)

		require.NoError(t, err)
	})

	t.Run("resource: non-conformance type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            // NOTE: R does not conform to I
            resource R {}

            let r: @R{I} <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNonConformanceIntersectionError{}, errs[0])
	})

	t.Run("struct: non-conformance type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            // NOTE: S does not conform to I
            struct S {}

            let s: S{I} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNonConformanceIntersectionError{}, errs[0])
	})

	t.Run("resource: duplicate type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            resource R: I {}

            // NOTE: I is duplicated
            let r: @R{I, I} <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidIntersectionTypeDuplicateError{}, errs[0])
	})

	t.Run("struct: duplicate type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            struct S: I {}

            // NOTE: I is duplicated
            let s: S{I, I} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidIntersectionTypeDuplicateError{}, errs[0])
	})

	t.Run("restricted resource, with structure interface type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            resource R: I {}

            let r: @R{I} <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("restricted struct, with resource interface type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            struct S: I {}

            let s: S{I} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("resource: non-concrete restricted type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            resource R: I {}

            let r: @[R]{I} <- [<-create R()]
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidIntersectionTypeError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("struct: non-concrete restricted type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            struct S: I {}

            let s: [S]{I} = [S()]
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.InvalidIntersectionTypeError{}, errs[0])
		assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
	})

	t.Run("restricted resource interface ", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            resource R: I {}

            let r: @I{} <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidIntersectionTypeError{}, errs[0])
	})

	t.Run("restricted struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            struct S: I {}

            let s: I{} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidIntersectionTypeError{}, errs[0])
	})

	t.Run("restricted type requirement", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
	      contract interface CI {
	          resource interface RI {}

	          resource R: RI {}

	          fun createR(): @R
	      }

          contract C: CI {
	          resource R: CI.RI {}

	          fun createR(): @R {
	              return <- create R()
	          }
	      }

          fun test() {
              let r <- C.createR()
              let r2: @CI.R{CI.RI} <- r
              destroy r2
          }
        `)
		require.NoError(t, err)
	})
}

func TestCheckIntersectionTypeMemberAccess(t *testing.T) {

	t.Parallel()

	t.Run("type with member: resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `

            resource interface I {
                let n: Int
            }

            resource R: I {
                let n: Int

                init(n: Int) {
                    self.n = n
                }
            }

            fun test() {
                let r: @{I} <- create R(n: 1)
                r.n
                destroy r
            }
        `)

		require.NoError(t, err)
	})

	t.Run("type with member: struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `

            struct interface I {
                let n: Int
            }

            struct S: I {
                let n: Int

                init(n: Int) {
                    self.n = n
                }
            }

            fun test() {
                let s: S{I} = S(n: 1)
                s.n
            }
        `)

		require.NoError(t, err)
	})

	t.Run("type without member: resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `

            resource interface I {
                // NOTE: no declaration for 'n'
            }

            resource R: I {
                let n: Int

                init(n: Int) {
                    self.n = n
                }
            }

            fun test() {
                let r: @{I} <- create R(n: 1)
                r.n
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessError{}, errs[0])
	})

	t.Run("type without member: struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `

            struct interface I {
                // NOTE: no declaration for 'n'
            }

            struct S: I {
                let n: Int

                init(n: Int) {
                    self.n = n
                }
            }

            fun test() {
                let s: {I} = S(n: 1)
                s.n
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidAccessError{}, errs[0])
	})

	t.Run("types with clashing members: resource", func(t *testing.T) {

		_, err := ParseAndCheck(t, `

            resource interface I1 {
                let n: Int
            }

            resource interface I2 {
                let n: Bool
            }

            resource R: I1, I2 {
                let n: Int

                init(n: Int) {
                    self.n = n
                }
            }

            fun test() {
                let r: @R{I1, I2} <- create R(n: 1)
                r.n
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
		assert.IsType(t, &sema.IntersectionMemberClashError{}, errs[1])
	})

	t.Run("types with clashing members: struct", func(t *testing.T) {

		_, err := ParseAndCheck(t, `

            struct interface I1 {
                let n: Int
            }

            struct interface I2 {
                let n: Bool
            }

            struct S: I1, I2 {
                let n: Int

                init(n: Int) {
                    self.n = n
                }
            }

            fun test() {
                let s: S{I1, I2} = S(n: 1)
                s.n
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
		assert.IsType(t, &sema.IntersectionMemberClashError{}, errs[1])
	})
}

func TestCheckRestrictedTypeSubtyping(t *testing.T) {

	t.Parallel()

	t.Run("resource type to restricted type with same type, no type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let r: @R{} <- create R()
                destroy r
            }
        `)

		require.NoError(t, err)
	})

	t.Run("struct type to restricted type with same type, no type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            let s: S{} = S()
        `)

		require.NoError(t, err)
	})

	t.Run("resource type to restricted type with same type, one type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @R{I1} <- create R()
                destroy r
            }
        `)

		require.NoError(t, err)
	})

	t.Run("struct type to restricted type with same type, one type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: S{I1} = S()
        `)

		require.NoError(t, err)
	})

	t.Run("resource type to restricted type with different restricted type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            resource S {}

            fun test() {
                let s: @S{} <- create R()
                destroy s
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct type to restricted type with different restricted type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct R {}

            struct S {}

            let s: S{} = R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("restricted resource type to restricted type with same type, no types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let r: @R{} <- create R()
                let r2: @R{} <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("restricted struct type to restricted type with same type, no types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            fun test() {
                let s: S{} = S()
                let s2: S{} = s
            }
        `)

		require.NoError(t, err)
	})

	t.Run("restricted resource type to restricted type with same type, 0 to 1 type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @R{} <- create R()
                let r2: @R{I1} <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("restricted struct type to restricted type with same type, 0 to 1 type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: S{} = S()
            let s2: S{I1} = s
        `)

		require.NoError(t, err)
	})

	t.Run("restricted resource type to restricted type with same type, 1 to 2 types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @R{I2} <- create R()
                let r2: @R{I1, I2} <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("restricted struct type to restricted type with same type, 1 to 2 types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `

            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: S{I2} = S()
            let s2: S{I1, I2} = s
        `)

		require.NoError(t, err)
	})

	t.Run("restricted resource type to restricted type with same type, reordered types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @R{I2, I1} <- create R()
                let r2: @R{I1, I2} <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("restricted struct type to restricted type with same type, reordered types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: S{I2, I1} = S()
            let s2: S{I1, I2} = s
        `)

		require.NoError(t, err)
	})

	t.Run("restricted resource type to restricted type with same type, fewer types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @R{I1, I2} <- create R()
                let r2: @R{I2} <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("restricted struct type to restricted type with same type, fewer types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: S{I1, I2} = S()
            let s2: S{I2} = s
        `)

		require.NoError(t, err)
	})

	t.Run("restricted resource type to resource type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @R{I1} <- create R()
                let r2: @R <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("restricted struct type to struct type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: S{I1} = S()
            let s2: S = s
        `)

		require.NoError(t, err)
	})
}

func TestCheckRestrictedTypeNoType(t *testing.T) {

	t.Parallel()

	const resourceTypes = `
      resource interface I1 {}

      resource interface I2 {}
    `

	const structTypes = `
      struct interface I1 {}

      struct interface I2 {}
    `

	t.Run("resource: empty", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t,
			resourceTypes+`
              let r: @{} <- panic("")
            `,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("struct: empty", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t,
			structTypes+`
              let s: {} = panic("")
            `,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("resource: one", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			resourceTypes+`
              let r: @{I1} <- panic("")
            `,
		)

		require.NoError(t, err)

		rType := RequireGlobalValue(t, checker.Elaboration, "r")
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 1)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
	})

	t.Run("struct: one", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			structTypes+`
              let s: {I1} = panic("")
            `,
		)

		require.NoError(t, err)

		rType := RequireGlobalValue(t, checker.Elaboration, "s")
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 1)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
	})

	t.Run("resource: two", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			resourceTypes+`
              let r: @{I1, I2} <- panic("")
            `,
		)

		require.NoError(t, err)

		rType := RequireGlobalValue(t, checker.Elaboration, "r")
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 2)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I2"),
			ty.Types[1],
		)
	})

	t.Run("struct: two", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			structTypes+`
              let s: {I1, I2} = panic("")
            `,
		)

		require.NoError(t, err)

		rType := RequireGlobalValue(t, checker.Elaboration, "s")
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 2)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I2"),
			ty.Types[1],
		)
	})

	t.Run("reference: empty", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheckWithPanic(t,
			resourceTypes+`
              let ref: &{} = panic("")
            `,
		)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("resource reference: one", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			resourceTypes+`
              let ref: &{I1} = panic("")
            `,
		)

		require.NoError(t, err)

		refType := RequireGlobalValue(t, checker.Elaboration, "ref")
		require.IsType(t, &sema.ReferenceType{}, refType)

		rType := refType.(*sema.ReferenceType).Type
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 1)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
	})

	t.Run("struct reference: one", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			structTypes+`
              let ref: &{I1} = panic("")
            `,
		)

		require.NoError(t, err)

		refType := RequireGlobalValue(t, checker.Elaboration, "ref")
		require.IsType(t, &sema.ReferenceType{}, refType)

		rType := refType.(*sema.ReferenceType).Type
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 1)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
	})

	t.Run("resource reference: two", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			resourceTypes+`
              let ref: &{I1, I2} = panic("")
            `,
		)

		require.NoError(t, err)

		refType := RequireGlobalValue(t, checker.Elaboration, "ref")
		require.IsType(t, &sema.ReferenceType{}, refType)

		rType := refType.(*sema.ReferenceType).Type
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 2)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I2"),
			ty.Types[1],
		)
	})

	t.Run("struct reference: two", func(t *testing.T) {
		t.Parallel()

		checker, err := ParseAndCheckWithPanic(t,
			structTypes+`
              let ref: &{I1, I2} = panic("")
            `,
		)

		require.NoError(t, err)

		refType := RequireGlobalValue(t, checker.Elaboration, "ref")
		require.IsType(t, &sema.ReferenceType{}, refType)

		rType := refType.(*sema.ReferenceType).Type
		require.IsType(t, &sema.IntersectionType{}, rType)

		ty := rType.(*sema.IntersectionType)

		require.Len(t, ty.Types, 2)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I1"),
			ty.Types[0],
		)
		assert.Same(t,
			RequireGlobalType(t, checker.Elaboration, "I2"),
			ty.Types[1],
		)
	})
}

func TestCheckRestrictedTypeConformanceOrder(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		// Test that the conformances for a composite are declared
		// before functions using them are checked

		_, err := ParseAndCheckWithPanic(t, `
          contract C {
              resource interface RI {}
              resource R: RI {}
              fun foo(): &R{RI} { panic("") }
          }
        `)

		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheckWithPanic(t, `
          contract C {
              resource interface RI {}
              resource R {}
              fun foo(): &R{RI} { panic("") }
          }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidNonConformanceIntersectionError{}, errs[0])
	})

}

// https://github.com/onflow/cadence/issues/326
func TestCheckRestrictedConformance(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      contract C {

          resource interface RI {
              fun get(): &R{RI}
          }

          resource R: RI {

              fun get(): &R{RI} {
                  return &self as &R{RI}
              }
          }
      }
    `)

	require.NoError(t, err)
}

func TestCheckInvalidIntersection(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x: {h} = nil
    `)

	errs := RequireCheckerErrors(t, err, 2)

	require.IsType(t, &sema.NotDeclaredError{}, errs[0])
	require.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[1])
}
