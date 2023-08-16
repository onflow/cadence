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

            let r: @{} <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("struct: no types", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            let r: {} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("resource: one type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            let r: @{I1} <- create R()
        `)

		require.NoError(t, err)
	})

	t.Run("struct: one type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let r: {I1} = S()
        `)

		require.NoError(t, err)
	})

	t.Run("reference to resource type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            let r <- create R()
            let ref: &{} = &r as &R
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("reference to struct type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            let s = S()
            let ref: &{} = &s as &S
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("resource: non-conformance type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            // NOTE: R does not conform to I
            resource R {}

            let r: @{I} <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("struct: non-conformance type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            // NOTE: S does not conform to I
            struct S {}

            let s: {I} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("resource: duplicate type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            resource R: I {}

            // NOTE: I is duplicated
            let r: @{I, I} <- create R()
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
            let s: {I, I} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidIntersectionTypeDuplicateError{}, errs[0])
	})

	t.Run("intersection resource, with structure interface type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            resource R: I {}

            let r: {I} = create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("intersection struct, with resource interface type", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            struct S: I {}

            let s: @{I} <- S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.CompositeKindMismatchError{}, errs[0])
	})

	t.Run("intersection resource interface ", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I {}

            resource R: I {}

            let r: @{} <- create R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("intersection struct interface", func(t *testing.T) {

		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I {}

            struct S: I {}

            let s: {} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
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
                let s: {I} = S(n: 1)
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

		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
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

		assert.IsType(t, &sema.NotDeclaredMemberError{}, errs[0])
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
                let r: @{I1, I2} <- create R(n: 1)
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
                let s: {I1, I2} = S(n: 1)
                s.n
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.ConformanceError{}, errs[0])
		assert.IsType(t, &sema.IntersectionMemberClashError{}, errs[1])
	})
}

func TestCheckIntersectionTypeSubtyping(t *testing.T) {

	t.Parallel()

	t.Run("resource type to intersection type with same type, no type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let r: @{} <- create R()
                destroy r
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("struct type to intersection type with same type, no type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            let s: {} = S()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("resource type to intersection type with same type, one type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @{I1} <- create R()
                destroy r
            }
        `)

		require.NoError(t, err)
	})

	t.Run("struct type to intersection type with same type, one type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: {I1} = S()
        `)

		require.NoError(t, err)
	})

	t.Run("resource type to intersection type with different intersection type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            resource S {}

            fun test() {
                let s: @{} <- create R()
                destroy s
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("struct type to intersection type with different intersection type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct R {}

            struct S {}

            let s: {} = R()
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("intersection resource type to intersection type with same type, no types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource R {}

            fun test() {
                let r: @{} <- create R()
                let r2: @{} <- r
                destroy r2
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[1])
	})

	t.Run("intersection struct type to intersection type with same type, no types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct S {}

            fun test() {
                let s: {} = S()
                let s2: {} = s
            }
        `)

		errs := RequireCheckerErrors(t, err, 2)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[1])
	})

	t.Run("intersection resource type to intersection type with same type, 0 to 1 type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @{} <- create R()
                let r2: @{I1} <- r
                destroy r2
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("intersection struct type to intersection type with same type, 0 to 1 type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: {} = S()
            let s2: {I1} = s
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.AmbiguousIntersectionTypeError{}, errs[0])
	})

	t.Run("intersection resource type to intersection type with same type, 1 to 2 types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @{I2} <- create R()
                let r2: @{I1, I2} <- r
                destroy r2
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("intersection struct type to intersection type with same type, 1 to 2 types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `

            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: {I2} = S()
            let s2: {I1, I2} = s
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("intersection resource type to intersection type with same type, reordered types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @{I2, I1} <- create R()
                let r2: @{I1, I2} <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("intersection struct type to intersection type with same type, reordered types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: {I2, I1} = S()
            let s2: {I1, I2} = s
        `)

		require.NoError(t, err)
	})

	t.Run("intersection resource type to intersection type with same type, fewer types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @{I1, I2} <- create R()
                let r2: @{I2} <- r
                destroy r2
            }
        `)

		require.NoError(t, err)
	})

	t.Run("intersection struct type to intersection type with same type, fewer types", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: {I1, I2} = S()
            let s2: {I2} = s
        `)

		require.NoError(t, err)
	})

	t.Run("intersection resource type to resource type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            resource interface I1 {}

            resource interface I2 {}

            resource R: I1, I2 {}

            fun test() {
                let r: @{I1} <- create R()
                let r2: @R <- r
                destroy r2
            }
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})

	t.Run("intersection struct type to struct type", func(t *testing.T) {
		t.Parallel()

		_, err := ParseAndCheck(t, `
            struct interface I1 {}

            struct interface I2 {}

            struct S: I1, I2 {}

            let s: {I1} = S()
            let s2: S = s
        `)

		errs := RequireCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}

func TestCheckIntersectionTypeNoType(t *testing.T) {

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

func TestCheckIntersectionTypeConformanceOrder(t *testing.T) {

	t.Parallel()

	t.Run("valid", func(t *testing.T) {

		t.Parallel()

		// Test that the conformances for a composite are declared
		// before functions using them are checked

		_, err := ParseAndCheckWithPanic(t, `
          contract C {
              resource interface RI {}
              resource R: RI {}
              fun foo(): &{RI} { panic("") }
          }
        `)

		require.NoError(t, err)
	})

}

// https://github.com/onflow/cadence/issues/326
func TestCheckIntersectionConformance(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `

      contract C {

          resource interface RI {
              fun get(): &{RI}
          }

          resource R: RI {

              fun get(): &{RI} {
                  return &self as &{RI}
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
