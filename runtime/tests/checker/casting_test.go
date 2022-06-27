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

	"github.com/onflow/cadence/runtime/sema"
)

func TestCheckCastingIntLiteralToIntegerType(t *testing.T) {

	t.Parallel()

	test := func(t *testing.T, integerType sema.Type) {

		t.Run(integerType.String(), func(t *testing.T) {

			t.Parallel()

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x = 1 as %s
                    `,
					integerType,
				),
			)

			require.NoError(t, err)

			xType := RequireGlobalValue(t, checker.Elaboration, "x")

			assert.Equal(t,
				integerType,
				xType,
			)

			assert.NotEmpty(t, checker.Elaboration.CastingTargetTypes)
		})
	}

	for _, integerType := range sema.AllIntegerTypes {
		test(t, integerType)
	}
}

func TestCheckInvalidCastingIntLiteralToString(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      let x = 1 as String
    `)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckCastingIntLiteralToAnyStruct(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      let x = 1 as AnyStruct
    `)

	require.NoError(t, err)

	xType := RequireGlobalValue(t, checker.Elaboration, "x")

	assert.Equal(t,
		sema.AnyStructType,
		xType,
	)

	assert.NotEmpty(t, checker.Elaboration.CastingTargetTypes)
}

func TestCheckCastingResourceToAnyResource(t *testing.T) {

	t.Parallel()

	checker, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let r <- create R()
          let x <- r as @AnyResource
          destroy x
      }
    `)

	require.NoError(t, err)

	assert.NotEmpty(t, checker.Elaboration.CastingTargetTypes)
}

func TestCheckCastingArrayLiteral(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      fun zipOf3(a: [AnyStruct; 3], b: [Int; 3]): [[AnyStruct; 2]; 3] {
          return [
              [a[0], b[0]] as [AnyStruct; 2],
              [a[1], b[1]] as [AnyStruct; 2],
              [a[2], b[2]] as [AnyStruct; 2]
          ]
      }
    `)

	require.NoError(t, err)
}

func TestCheckCastResourceType(t *testing.T) {

	t.Parallel()

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1, I2} <- create R()
                  let r2 <- r as @R{I2}
                `,
			)

			require.NoError(t, err)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.RestrictedType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{I2}? {
                      let r: @R{I1, I2} <- create R()
                      if let r2 <- r as? @R{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @R{I1, I2}
                `,
			)

			require.NoError(t, err)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.RestrictedType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{I1, I2}? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @R{I1, I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R1{I} <- create R1()
                  let r2 <- r as @R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R2{I}? {
                      let r: @R1{I} <- create R1()
                      if let r2 <- r as? @R2{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @R{I}
                `,
			)

			require.NoError(t, err)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.RestrictedType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{I}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @R{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R1 <- create R1()
                  let r2 <- r as @R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R2{I}? {
                      let r: @R1 <- create R1()
                      if let r2 <- r as? @R2{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyResource -> conforming restricted type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @R{RI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{RI}? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @R{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> conforming restricted type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R{RI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{RI}? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> non-conforming restricted type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R{RI}? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[2])

		})
	})

	// Supertype: Resource (unrestricted)

	t.Run("restricted type -> unrestricted type: same resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I} <- create R()
                  let r2 <- r as @R
                `,
			)

			require.NoError(t, err)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.CompositeType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @R{I} <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}

          resource T: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R{I} <- create R()
                  let t <- r as @T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @T? {
                      let r: @R{I} <- create R()
                      if let t <- r as? @T {
                          return <-t
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyResource -> conforming resource", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> non-conforming resource", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{RI} <- create R()
                  let r2 <- r as @R
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @AnyResource{RI} <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyResource -> unrestricted type", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @R
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: restricted AnyResource

	t.Run("resource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const types = `
          resource interface RI {}

          // NOTE: R does not conform to RI
          resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @AnyResource{RI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{RI}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @AnyResource{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

	})

	t.Run("resource -> restricted AnyResource with conformance restriction", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @AnyResource{RI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{RI}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @AnyResource{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with conformance in restriction", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I} <- create R()
                  let r2 <- r as @AnyResource{I}
                `,
			)

			require.NoError(t, err)

			iType := RequireGlobalType(t, checker.Elaboration, "I")

			require.IsType(t, &sema.InterfaceType{}, iType)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.RestrictedType{
					Type: sema.AnyResourceType,
					Restrictions: []*sema.InterfaceType{
						iType.(*sema.InterfaceType),
					},
				},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I}? {
                      let r: @R{I} <- create R()
                      if let r2 <- r as? @AnyResource{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with conformance not in restriction", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @AnyResource{I2}
                `,
			)

			require.NoError(t, err)

			i2Type := RequireGlobalType(t, checker.Elaboration, "I2")

			require.IsType(t, &sema.InterfaceType{}, i2Type)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.RestrictedType{
					Type: sema.AnyResourceType,
					Restrictions: []*sema.InterfaceType{
						i2Type.(*sema.InterfaceType),
					},
				},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I2}? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @AnyResource{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I2}? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource: fewer restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1, I2} <- create R()
                  let r2 <- r as @AnyResource{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I2}? {
                      let r: @AnyResource{I1, I2} <- create R()
                      if let r2 <- r as? @AnyResource{I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource: more restrictions", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1} <- create R()
                  let r2 <- r as @AnyResource{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I1, I2}? {
                      let r: @AnyResource{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I1, I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> restricted AnyResource with non-conformance restriction", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1} <- create R()
                  let r2 <- r as @AnyResource{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I1, I2}? {
                      let r: @AnyResource{I1} <- create R()
                      if let r2 <- r as? @AnyResource{I1, I2} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyResource -> restricted AnyResource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @AnyResource{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource{I}? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @AnyResource{I} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: AnyResource

	t.Run("restricted type -> AnyResource", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R{I1} <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r: @R{I1} <- create R()
                      if let r2 <- r as? @AnyResource {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyResource -> AnyResource", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource{I1} <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r: @AnyResource{I1} <- create R()
                      if let r2 <- r as? @AnyResource {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> AnyResource", func(t *testing.T) {

		const types = `
           resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {
			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r <- create R()
                      if let r2 <- r as? @AnyResource {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			require.NoError(t, err)
		})
	})
}

func TestCheckCastStructType(t *testing.T) {

	t.Parallel()

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1, I2} = S()
                  let s2 = s as S{I2}
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.RestrictedType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1, I2} = S()
                  let s2 = s as? S{I2}
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.OptionalType{
					Type: &sema.RestrictedType{},
				},
				s2Type,
			)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as S{I1, I2}
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.RestrictedType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? S{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S1{I} = S1()
                  let s2 = s as S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S1{I} = S1()
                  let s2 = s as? S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as S{I}
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.RestrictedType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? S{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S1 = S1()
                  let s2 = s as S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                   let s: S1 = S1()
                   let s2 = s as? S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyStruct -> conforming restricted type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as S{SI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? S{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> conforming restricted type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S{SI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> non-conforming restricted type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
		})
	})

	// Supertype: Struct (unrestricted)

	t.Run("restricted type -> unrestricted type: same struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as S
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.CompositeType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as? S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different struct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}

          struct T: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: T{I} = S()
                  let t = s as T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: T{I} = S()
                  let t = s as? T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyStruct -> conforming struct", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> non-conforming struct", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{SI} = S()
                  let s2 = s as? S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyStruct -> unrestricted type", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as S
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? S
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: restricted AnyStruct

	t.Run("struct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const types = `
          struct interface SI {}

          // NOTE: S does not conform to SI
          struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as AnyStruct{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? AnyStruct{SI}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

	})

	t.Run("struct -> restricted AnyStruct with conformance restriction", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as AnyStruct{SI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? AnyStruct{SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with conformance in restriction", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as AnyStruct{I}
                `,
			)

			require.NoError(t, err)

			iType := RequireGlobalType(t, checker.Elaboration, "I")

			require.IsType(t, &sema.InterfaceType{}, iType)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.RestrictedType{
					Type: sema.AnyStructType,
					Restrictions: []*sema.InterfaceType{
						iType.(*sema.InterfaceType),
					},
				},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I} = S()
                  let s2 = s as? AnyStruct{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with conformance not in restriction", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as AnyStruct{I2}
                `,
			)

			require.NoError(t, err)

			i2Type := RequireGlobalType(t, checker.Elaboration, "I2")

			require.IsType(t, &sema.InterfaceType{}, i2Type)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.RestrictedType{
					Type: sema.AnyStructType,
					Restrictions: []*sema.InterfaceType{
						i2Type.(*sema.InterfaceType),
					},
				},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as AnyStruct{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? AnyStruct{I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct: fewer restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1, I2} = S()
                  let s2 = s as AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1, I2} = S()
                  let s2 = s as? AnyStruct{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct: more restrictions", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as AnyStruct{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as? AnyStruct{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> restricted AnyStruct with non-conformance restriction", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as AnyStruct{I1, I2}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as? AnyStruct{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyStruct -> restricted AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as AnyStruct{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? AnyStruct{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: AnyStruct

	t.Run("restricted type -> AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S{I1} = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted AnyStruct -> AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct{I1} = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> AnyStruct", func(t *testing.T) {

		const types = `
           struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {
			_, err := ParseAndCheck(t,
				types+`
                  let s = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})
}

func TestCheckReferenceTypeSubTyping(t *testing.T) {

	t.Parallel()

	t.Run("resource", func(t *testing.T) {

		test := func(ty string) {

			t.Run(fmt.Sprintf("auth to non-auth: %s", ty), func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(`
                          resource interface I {}

                          resource R: I {}

                          let r <- create R()
                          let ref = &r as auth &%[1]s
                          let ref2 = ref as &%[1]s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run(fmt.Sprintf("non-auth to auth: %s", ty), func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(`
                          resource interface I {}

                          resource R: I {}

                          let r <- create R()
                          let ref = &r as &%[1]s
                          let ref2 = ref as auth &%[1]s
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		}

		for _, ty := range []string{
			"R",
			"R{I}",
			"AnyResource",
			"AnyResource{I}",
			"Any",
			"Any{I}",
		} {
			test(ty)
		}
	})

	t.Run("struct", func(t *testing.T) {

		test := func(ty string) {

			t.Run(fmt.Sprintf("auth to non-auth: %s", ty), func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(`
                          struct interface I {}

                          struct S: I {}

                          let s = S()
                          let ref = &s as auth &%[1]s
                          let ref2 = ref as &%[1]s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run(fmt.Sprintf("non-auth to auth: %s", ty), func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let s = S()
                          let ref = &s as &%[1]s
                          let ref2 = ref as auth &%[1]s
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		}

		for _, ty := range []string{
			"S",
			"S{I}",
			"AnyStruct",
			"AnyStruct{I}",
			"Any",
			"Any{I}",
		} {
			test(ty)
		}
	})

	t.Run("non-composite", func(t *testing.T) {

		test := func(ty string) {

			t.Run(fmt.Sprintf("auth to non-auth: %s", ty), func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(`
                          let i = 1
                          let ref = &i as auth &%[1]s
                          let ref2 = ref as &%[1]s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run(fmt.Sprintf("non-auth to auth: %s", ty), func(t *testing.T) {

				t.Parallel()

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
                          let i = 1
                          let ref = &i as &%[1]s
                          let ref2 = ref as auth &%[1]s
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		}

		for _, ty := range []string{
			"Int",
			"AnyStruct",
			"Any",
		} {
			test(ty)
		}
	})
}

func TestCheckCastAuthorizedResourceReferenceType(t *testing.T) {

	t.Parallel()

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &R{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}

          let x <- create R()
          let r = &x as auth &R{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{I1, I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}

          let x <- create R1()
          let r = &x as auth &R1{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          let x <- create R()
          let r = &x as auth &R
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}

          let x <- create R1()
          let r = &x as auth &R1
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	for _, ty := range []sema.Type{
		sema.AnyResourceType,
		sema.AnyType,
	} {

		t.Run(fmt.Sprintf("restricted %s -> conforming restricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(`
                  resource interface RI {}

                  resource R: RI {}

                  let x <- create R()
                  let r = &x as auth &%s{RI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as &R{RI}
                    `,
				)

				// NOTE: static cast not allowed, only dynamic

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as? &R{RI}
                    `,
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("%s -> conforming restricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(`
                  resource interface RI {}

                  resource R: RI {}

                  let x <- create R()
                  let r = &x as auth &%s
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as &R{RI}
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as? &R{RI}
                    `,
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted %s -> non-conforming restricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(`
                  resource interface RI {}

                  resource R {}

                  let x <- create R()
                  let r = &x as auth &%s{RI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as &R{RI}
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as? &R{RI}
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			})
		})
	}

	// Supertype: Resource (unrestricted)

	t.Run("restricted type -> unrestricted type: same resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          let x <- create R()
          let r = &x as auth &R{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &R
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}

          resource T: I {}

          let x <- create R()
          let r = &x as auth &R{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = r as &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = r as? &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	for _, ty := range []sema.Type{
		sema.AnyResourceType,
		sema.AnyType,
	} {

		t.Run(fmt.Sprintf("restricted %s -> conforming resource", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  resource interface RI {}

                  resource R: RI {}

                  let x <- create R()
                  let r = &x as auth &%s{RI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as &R
                    `,
				)

				// NOTE: static cast not allowed, only dynamic

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as? &R
                    `,
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted %s -> non-conforming resource", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  resource interface RI {}

                  resource R {}

                  let x <- create R()
                  let r = &x as auth &%s{RI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as &R
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as? &R
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		})

		t.Run(fmt.Sprintf("%s -> unrestricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  resource interface RI {}

                  resource R: RI {}

                  let x <- create R()
                  let r = &x as auth &%s
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as &R
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as? &R
                    `,
				)

				require.NoError(t, err)
			})
		})

		// Supertype: restricted AnyResource / Any

		t.Run(fmt.Sprintf("resource -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

			const setup = `
              resource interface RI {}

              // NOTE: R does not conform to RI
              resource R {}

              let x <- create R()
              let r = &x as auth &R
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as &%s{RI}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as? &%s{RI}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		})

		t.Run(fmt.Sprintf("resource -> restricted %s with conformance restriction", ty), func(t *testing.T) {

			const setup = `
              resource interface RI {}

              resource R: RI {}

              let x <- create R()
              let r = &x as auth &R
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as &%s{RI}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as? &%s{RI}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance in restriction", ty), func(t *testing.T) {

			const setup = `
              resource interface I {}

              resource R: I {}

              let x <- create R()
              let r = &x as auth &R{I}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as &%s{I}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as? &%s{I}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance not in restriction", ty), func(t *testing.T) {

			const setup = `
              resource interface I1 {}

              resource interface I2 {}

              resource R: I1, I2 {}

              let x <- create R()
              let r = &x as auth &R{I1}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as &%s{I2}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as? &%s{I2}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted type -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

			const setup = `
              resource interface I1 {}

              resource interface I2 {}

              resource R: I1 {}

              let x <- create R()
              let r = &x as auth &R{I1}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as &%s{I2}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as? &%s{I2}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		})

		for _, otherType := range []sema.Type{
			sema.AnyResourceType,
			sema.AnyType,
		} {

			t.Run(fmt.Sprintf("restricted %s -> restricted %s: fewer restrictions", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}

                      let x <- create R()
                      let r = &x as auth &%s{I1, I2}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as &%s{I2}
                            `,
							otherType,
						),
					)

					if ty == sema.AnyType && otherType == sema.AnyResourceType {

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

						return
					}

					require.NoError(t, err)
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as? &%s{I2}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

			t.Run(fmt.Sprintf("restricted %s -> restricted %s: more restrictions", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}

                      let x <- create R()
                      let r = &x as auth &%s{I1}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as &%s{I1, I2}
                            `,
							otherType,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as? &%s{I1, I2}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

			t.Run(fmt.Sprintf("restricted %s -> restricted %s with non-conformance restriction", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1 {}

                      let x <- create R()
                      let r = &x as auth &%s{I1}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as &%s{I1, I2}
                            `,
							otherType,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as? &%s{I1, I2}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

			t.Run(fmt.Sprintf("%s -> restricted %s", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      resource interface I {}

                      resource R: I {}

                      let x <- create R()
                      let r = &x as auth &%s
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as &%s{I}
                            `,
							otherType,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as? &%s{I}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})
		}

		// Supertype: AnyResource / Any

		t.Run(fmt.Sprintf("restricted type -> %s", ty), func(t *testing.T) {

			const setup = `
              resource interface I1 {}

              resource interface I2 {}

              resource R: I1, I2 {}

              let x <- create R()
              let r = &x as auth &R{I1}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as? &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		for _, otherType := range []sema.Type{
			sema.AnyResourceType,
			sema.AnyType,
		} {
			t.Run(fmt.Sprintf("restricted %s -> %s", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      resource interface I1 {}

                      resource interface I2 {}

                      resource R: I1, I2 {}

                      let x <- create R()
                      let r = &x as auth &%s{I1}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as &%s
                            `,
							otherType,
						),
					)

					if ty == sema.AnyType && otherType == sema.AnyResourceType {

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

						return
					}

					require.NoError(t, err)
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let r2 = r as? &%s
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

		}

		t.Run(fmt.Sprintf("unrestricted type -> %s", ty), func(t *testing.T) {

			const setup = `
              resource interface I1 {}

              resource interface I2 {}

              resource R: I1, I2 {}

              let x <- create R()
              let r = &x as auth &R
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let r2 = r as? &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})
	}
}

func TestCheckCastAuthorizedStructReferenceType(t *testing.T) {

	t.Parallel()

	// Supertype: Restricted type

	t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &S{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}

          let x = S()
          let s = &x as auth &S{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{I1, I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> restricted type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}

          let x = S1()
          let s = &x as auth &S1{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("unrestricted type -> restricted type: same struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          let x = S()
          let s = &x as auth &S

        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("unrestricted type -> restricted type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}

          let x = S1()
          let s = &x as auth &S1
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S2{I}
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	for _, ty := range []sema.Type{
		sema.AnyStructType,
		sema.AnyType,
	} {
		t.Run(fmt.Sprintf("restricted %s -> conforming restricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface SI {}

                  struct S: SI {}

                  let x = S()
                  let s = &x as auth &%s{SI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &S{SI}
                    `,
				)

				// NOTE: static cast not allowed, only dynamic

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as? &S{SI}
                    `,
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("%s -> conforming restricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface SI {}

                  struct S: SI {}

                  let x = S()
                  let s = &x as auth &%s
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &S{SI}
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as? &S{SI}
                    `,
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted %s -> non-conforming restricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface SI {}

                  struct S {}

                  let x = S()
                  let s = &x as auth &%s{SI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &S{SI}
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 3)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as? &S{SI}
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
			})
		})
	}

	// Supertype: Struct (unrestricted)

	t.Run("restricted type -> unrestricted type: same struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          let x = S()
          let s = &x as auth &S{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("restricted type -> unrestricted type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          struct T: I {}

          let x = S()
          let s = &x as auth &S{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = s as &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = s as? &T
                `,
			)

			errs := ExpectCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	for _, ty := range []sema.Type{
		sema.AnyStructType,
		sema.AnyType,
	} {

		t.Run(fmt.Sprintf("restricted %s -> conforming struct", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface RI {}

                  struct S: RI {}

                  let x = S()
                  let s = &x as auth &%s{RI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &S
                    `,
				)

				// NOTE: static cast not allowed, only dynamic

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as? &S
                    `,
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted %s -> non-conforming struct", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface RI {}

                  struct S {}

                  let x = S()
                  let s = &x as auth &%s{RI}
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &S
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 2)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as? &S
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		})

		t.Run(fmt.Sprintf("%s -> unrestricted type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface SI {}

                  struct S: SI {}

                  let x = S()
                  let s = &x as auth &%s
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &S
                    `,
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as? &S
                    `,
				)

				require.NoError(t, err)
			})
		})

		// Supertype: restricted AnyStruct / Any

		t.Run(fmt.Sprintf("struct -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

			const setup = `
              struct interface SI {}

              // NOTE: S does not conform to SI
              struct S {}

              let x = S()
              let s = &x as auth &S
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as &%s{SI}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as? &%s{SI}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		})

		t.Run(fmt.Sprintf("struct -> restricted %s with conformance restriction", ty), func(t *testing.T) {

			const setup = `
              struct interface SI {}

              struct S: SI {}

              let x = S()
              let s = &x as auth &S
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as &%s{SI}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as? &%s{SI}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance in restriction", ty), func(t *testing.T) {

			const setup = `
              struct interface I {}

              struct S: I {}

              let x = S()
              let s = &x as auth &S{I}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as &%s{I}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as? &%s{I}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance not in restriction", ty), func(t *testing.T) {

			const setup = `
              struct interface I1 {}

              struct interface I2 {}

              struct S: I1, I2 {}

              let x = S()
              let s = &x as auth &S{I1}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as &%s{I2}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as? &%s{I2}
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("restricted type -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

			const setup = `
              struct interface I1 {}

              struct interface I2 {}

              struct S: I1 {}

              let x = S()
              let s = &x as auth &S{I1}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as &%s{I2}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as? &%s{I2}
                        `,
						ty,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		})

		for _, otherType := range []sema.Type{
			sema.AnyStructType,
			sema.AnyType,
		} {

			t.Run(fmt.Sprintf("restricted %s -> restricted %s: fewer restrictions", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}

                      let x = S()
                      let s = &x as auth &%s{I1, I2}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as &%s{I2}
                            `,
							otherType,
						),
					)

					if ty == sema.AnyType && otherType == sema.AnyStructType {

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

						return
					}

					require.NoError(t, err)
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as? &%s{I2}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

			t.Run(fmt.Sprintf("restricted %s -> restricted %s: more restrictions", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}

                      let x = S()
                      let s = &x as auth &%s{I1}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
							  let s2 = s as &%s{I1, I2}
                            `,
							otherType,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as? &%s{I1, I2}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

			t.Run(fmt.Sprintf("restricted %s -> restricted %s with non-conformance restriction", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1 {}

                      let x = S()
                      let s = &x as auth &%s{I1}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as &%s{I1, I2}
                            `,
							otherType,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as? &%s{I1, I2}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

			t.Run(fmt.Sprintf("%s -> restricted %s", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      struct interface I {}

                      struct S: I {}

                      let x = S()
                      let s = &x as auth &%s
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as &%s{I}
                            `,
							otherType,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as? &%s{I}
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})

			// Supertype: AnyStruct / Any

			t.Run(fmt.Sprintf("restricted %s -> %s", ty, otherType), func(t *testing.T) {

				setup := fmt.Sprintf(
					`
                      struct interface I1 {}

                      struct interface I2 {}

                      struct S: I1, I2 {}

                      let x = S()
                      let s = &x as auth &%s{I1}
                    `,
					ty,
				)

				t.Run("static", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as &%s
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})

				t.Run("dynamic", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						setup+fmt.Sprintf(
							`
                              let s2 = s as? &%s
                            `,
							otherType,
						),
					)

					require.NoError(t, err)
				})
			})
		}

		t.Run(fmt.Sprintf("restricted type -> %s", ty), func(t *testing.T) {

			const setup = `
              struct interface I1 {}

              struct interface I2 {}

              struct S: I1, I2 {}

              let x = S()
              let s = &x as auth &S{I1}
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as? &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})

		t.Run(fmt.Sprintf("unrestricted type -> %s", ty), func(t *testing.T) {

			const setup = `
              struct interface I1 {}

              struct interface I2 {}

              struct S: I1, I2 {}

              let x = S()
              let s = &x as auth &S
            `

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+fmt.Sprintf(
						`
                          let s2 = s as? &%s
                        `,
						ty,
					),
				)

				require.NoError(t, err)
			})
		})
	}
}

func TestCheckCastUnauthorizedResourceReferenceType(t *testing.T) {

	t.Parallel()

	for name, op := range map[string]string{
		"static":  "as",
		"dynamic": "as?",
	} {

		t.Run(name, func(t *testing.T) {

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &R{I1, I2}
                          let r2 = r %s &R{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &R{I1}
                          let r2 = r %s &R{I1, I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R1: I {}

                          resource R2: I {}

                          let x <- create R1()
                          let r = &x as &R1{I}
                          let r2 = r %s &R2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &R
                          let r2 = r %s &R{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R1: I {}

                          resource R2: I {}

                          let x <- create R1()
                          let r = &x as &R1
                          let r2 = r %s &R2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			for _, ty := range []sema.Type{
				sema.AnyResourceType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("restricted %s -> conforming restricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R: RI {}

                              let x <- create R()
                              let r = &x as &%s{RI}
                              let r2 = r %s &R{RI}
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("%s -> conforming restricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R: RI {}

                              let x <- create R()
                              let r = &x as &%s
                              let r2 = r %s &R{RI}
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("restricted %s -> non-conforming restricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R {}

                              let x <- create R()
                              let r = &x as &%s{RI}
                              let r2 = r %s &R{RI}
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 3)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
					assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
				})
			}

			// Supertype: Resource (unrestricted)

			t.Run("restricted type -> unrestricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &R{I}
                          let r2 = r %s &R
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          resource T: I {}

                          let x <- create R()
                          let r = &x as &R{I}
                          let t = r %s &T
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			for _, ty := range []sema.Type{
				sema.AnyResourceType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("restricted %s -> conforming resource", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R: RI {}

                              let x <- create R()
                              let r = &x as &%s{RI}
                              let r2 = r %s &R
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("restricted %s -> non-conforming resource", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R {}

                              let x <- create R()
                              let r = &x as &%s{RI}
                              let r2 = r %s &R
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 2)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
				})

				t.Run(fmt.Sprintf("%s -> unrestricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R: RI {}

                              let x <- create R()
                              let r = &x as &%s
                              let r2 = r %s &R
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				// Supertype: restricted AnyResource / Any

				t.Run(fmt.Sprintf("resource -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              // NOTE: R does not conform to RI
                              resource R {}

                              let x <- create R()
                              let r = &x as &R
                              let r2 = r %s &%s{RI}
                            `,
							op,
							ty,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("resource -> restricted %s with conformance restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R: RI {}

                              let x <- create R()
                              let r = &x as &R
                              let r2 = r %s &%s{RI}
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance in restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface I {}

                              resource R: I {}

                              let x <- create R()
                              let r = &x as &R{I}
                              let r2 = r %s &%s{I}
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance not in restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface I1 {}

                              resource interface I2 {}

                              resource R: I1, I2 {}

                              let x <- create R()
                              let r = &x as &R{I1}
                              let r2 = r %s &%s{I2}
                            `,
							op,
							ty,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("restricted type -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface I1 {}

                              resource interface I2 {}

                              resource R: I1 {}

                              let x <- create R()
                              let r = &x as &R{I1}
                              let r2 = r %s &%s{I2}
                            `,
							op,
							ty,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

				})

				for _, otherType := range []sema.Type{
					sema.AnyResourceType,
					sema.AnyType,
				} {

					t.Run(fmt.Sprintf("restricted %s -> restricted %s: fewer restrictions", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  resource interface I1 {}

                                  resource interface I2 {}

                                  resource R: I1, I2 {}

                                  let x <- create R()
                                  let r = &x as &%s{I1, I2}
                                  let r2 = r %s &%s{I2}
                                `,
								ty,
								op,
								otherType,
							),
						)

						if ty == sema.AnyType && otherType == sema.AnyResourceType {

							errs := ExpectCheckerErrors(t, err, 1)

							assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

							return
						}

						require.NoError(t, err)
					})

					t.Run(fmt.Sprintf("restricted %s -> restricted %s: more restrictions", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  resource interface I1 {}

                                  resource interface I2 {}

                                  resource R: I1, I2 {}

                                  let x <- create R()
                                  let r = &x as &%s{I1}
                                  let r2 = r %s &%s{I1, I2}
                                `,
								ty,
								op,
								otherType,
							),
						)

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					})

					t.Run(fmt.Sprintf("restricted %s -> restricted %s with non-conformance restriction", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  resource interface I1 {}

                                  resource interface I2 {}

                                  resource R: I1 {}

                                  let x <- create R()
                                  let r = &x as &%s{I1}
                                  let r2 = r %s &%s{I1, I2}
		                        `,
								ty,
								op,
								otherType,
							),
						)

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					})

					t.Run(fmt.Sprintf("%s -> restricted %s", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  resource interface I {}

                                  resource R: I {}

                                  let x <- create R()
                                  let r = &x as &%s
                                  let r2 = r %s &%s{I}
                                `,
								ty,
								op,
								otherType,
							),
						)

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					})

					// Supertype: AnyResource / Any

					t.Run(fmt.Sprintf("restricted %s -> %s", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  resource interface I1 {}

                                  resource interface I2 {}

                                  resource R: I1, I2 {}

                                  let x <- create R()
                                  let r = &x as &%s{I1}
                                  let r2 = r %s &%s
                                `,
								ty,
								op,
								otherType,
							),
						)

						if ty == sema.AnyType && otherType == sema.AnyResourceType {

							errs := ExpectCheckerErrors(t, err, 1)

							assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

							return
						}

						require.NoError(t, err)
					})

				}

				t.Run(fmt.Sprintf("restricted type -> %s", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface I1 {}

                              resource interface I2 {}

                              resource R: I1, I2 {}

                              let x <- create R()
                              let r = &x as &R{I1}
                              let r2 = r %s &%s
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("unrestricted type -> %s", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface I1 {}

                              resource interface I2 {}

                              resource R: I1, I2 {}

                              let x <- create R()
                              let r = &x as &R
                              let r2 = r %s &%s
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})
			}
		})
	}
}

func TestCheckCastUnauthorizedStructReferenceType(t *testing.T) {

	t.Parallel()

	for name, op := range map[string]string{
		"static":  "as",
		"dynamic": "as?",
	} {

		t.Run(name, func(t *testing.T) {

			// Supertype: Restricted type

			t.Run("restricted type -> restricted type: fewer restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &S{I1, I2}
                          let s2 = s %s &S{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("restricted type -> restricted type: more restrictions", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &S{I1}
                          let s2 = s %s &S{I1, I2}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S1: I {}

                          struct S2: I {}

                          let x = S1()
                          let s = &x as &S1{I}
                          let s2 = s %s &S2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("unrestricted type -> restricted type: same resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &S
                          let s2 = s %s &S{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("unrestricted type -> restricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S1: I {}

                          struct S2: I {}

                          let x = S1()
                          let s = &x as &S1
                          let s2 = s %s &S2{I}
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			for _, ty := range []sema.Type{
				sema.AnyStructType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("restricted %s -> conforming restricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S: RI {}

                              let x = S()
                              let s = &x as &%s{RI}
                              let s2 = s %s &S{RI}
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("%s -> conforming restricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S: RI {}

                              let x = S()
                              let s = &x as &%s
                              let s2 = s %s &S{RI}
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("restricted %s -> non-conforming restricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S {}

                              let x = S()
                              let s = &x as &%s{RI}
                              let s2 = s %s &S{RI}
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 3)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					assert.IsType(t, &sema.InvalidNonConformanceRestrictionError{}, errs[1])
					assert.IsType(t, &sema.TypeMismatchError{}, errs[2])
				})
			}

			// Supertype: Resource (unrestricted)

			t.Run("restricted type -> unrestricted type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &S{I}
                          let s2 = s %s &S
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("restricted type -> unrestricted type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          struct T: I {}

                          let x = S()
                          let s = &x as &S{I}
                          let t = s %s &T
                        `,
						op,
					),
				)

				errs := ExpectCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			for _, ty := range []sema.Type{
				sema.AnyStructType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("restricted %s -> conforming resource", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S: RI {}

                              let x = S()
                              let s = &x as &%s{RI}
                              let s2 = s %s &S
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("restricted %s -> non-conforming resource", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S {}

                              let x = S()
                              let s = &x as &%s{RI}
                              let s2 = s %s &S
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 2)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
				})

				t.Run(fmt.Sprintf("%s -> unrestricted type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S: RI {}

                              let x = S()
                              let s = &x as &%s
                              let s2 = s %s &S
                            `,
							ty,
							op,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				// Supertype: restricted AnyStruct / Any

				t.Run(fmt.Sprintf("resource -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              // NOTE: R does not conform to RI
                              struct S {}

                              let x = S()
                              let s = &x as &S
                              let s2 = s %s &%s{RI}
                            `,
							op,
							ty,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("resource -> restricted %s with conformance restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S: RI {}

                              let x = S()
                              let s = &x as &S
                              let s2 = s %s &%s{RI}
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance in restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface I {}

                              struct S: I {}

                              let x = S()
                              let s = &x as &S{I}
                              let s2 = s %s &%s{I}
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("restricted type -> restricted %s with conformance not in restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface I1 {}

                              struct interface I2 {}

                              struct S: I1, I2 {}

                              let x = S()
                              let s = &x as &S{I1}
                              let s2 = s %s &%s{I2}
                            `,
							op,
							ty,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				t.Run(fmt.Sprintf("restricted type -> restricted %s with non-conformance restriction", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface I1 {}

                              struct interface I2 {}

                              struct S: I1 {}

                              let x = S()
                              let s = &x as &S{I1}
                              let s2 = s %s &%s{I2}
                            `,
							op,
							ty,
						),
					)

					errs := ExpectCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				})

				for _, otherType := range []sema.Type{
					sema.AnyStructType,
					sema.AnyType,
				} {

					t.Run(fmt.Sprintf("restricted %s -> restricted %s: fewer restrictions", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  struct interface I1 {}

                                  struct interface I2 {}

                                  struct S: I1, I2 {}

                                  let x = S()
                                  let s = &x as &%s{I1, I2}
                                  let s2 = s %s &%s{I2}
                                `,
								ty,
								op,
								otherType,
							),
						)

						if ty == sema.AnyType && otherType == sema.AnyStructType {

							errs := ExpectCheckerErrors(t, err, 1)

							assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

							return
						}

						require.NoError(t, err)
					})

					t.Run(fmt.Sprintf("restricted %s -> restricted %s: more restrictions", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  struct interface I1 {}

                                  struct interface I2 {}

                                  struct S: I1, I2 {}

                                  let x = S()
                                  let s = &x as &%s{I1}
                                  let s2 = s %s &%s{I1, I2}
                                `,
								ty,
								op,
								otherType,
							),
						)

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					})

					t.Run(fmt.Sprintf("restricted %s -> restricted %s with non-conformance restriction", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  struct interface I1 {}

                                  struct interface I2 {}

                                  struct S: I1 {}

                                  let x = S()
                                  let s = &x as &%s{I1}
                                  let s2 = s %s &%s{I1, I2}
		                        `,
								ty,
								op,
								otherType,
							),
						)

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					})

					t.Run(fmt.Sprintf("%s -> restricted %s", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                  struct interface I {}

                                  struct S: I {}

                                  let x = S()
                                  let s = &x as &%s
                                  let s2 = s %s &%s{I}
                                `,
								ty,
								op,
								otherType,
							),
						)

						errs := ExpectCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					})

					// Supertype: AnyStruct / Any

					t.Run(fmt.Sprintf("restricted %s -> %s", ty, otherType), func(t *testing.T) {

						_, err := ParseAndCheckWithAny(t,
							fmt.Sprintf(
								`
                                 struct interface I1 {}

                                 struct interface I2 {}

                                 struct S: I1, I2 {}

                                 let x = S()
                                 let s = &x as &%s{I1}
                                 let s2 = s %s &%s
                               `,
								ty,
								op,
								otherType,
							),
						)

						require.NoError(t, err)
					})
				}

				t.Run(fmt.Sprintf("restricted type -> %s", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface I1 {}

                              struct interface I2 {}

                              struct S: I1, I2 {}

                              let x = S()
                              let s = &x as &S{I1}
                              let s2 = s %s &%s
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("unrestricted type -> %s", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface I1 {}

                              struct interface I2 {}

                              struct S: I1, I2 {}

                              let x = S()
                              let s = &x as &S
                              let s2 = s %s &%s
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})
			}
		})
	}
}

func TestCheckCastAuthorizedNonCompositeReferenceType(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheckWithAny(t, `
      let x = 1
      let xRef = &x as &Int
      let anyRef: &AnyStruct = xRef
    `)

	require.NoError(t, err)
}

func TestCheckResourceConstructorCast(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          resource R {}

          let c = R as ((): @R)
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckResourceConstructorReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          resource R {}

          fun test(): ((): @R) {
              return R
          }
        `,
	)

	errs := ExpectCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckStaticCastElaboration(t *testing.T) {

	t.Parallel()

	t.Run("Same type as expected type", func(t *testing.T) {
		t.Parallel()

		t.Run("Var decl", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int8 = 1 as Int8
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Int8Type, cast.TargetType)
			}
		})

		t.Run("Binary exp", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int8 = (1 as Int8) + (1 as Int8)
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Int8Type, cast.TargetType)
			}
		})

		t.Run("Nested casts", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = (1 as Int8) as Int8
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Int8Type, cast.TargetType)
			}
		})

		t.Run("Arrays", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: [Character] = ["c" as Character]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.CharacterType, cast.TargetType)
			}
		})

		t.Run("Dictionaries", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: {String: UInt8} = {"foo": 4 as UInt8}
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.UInt8Type, cast.TargetType)
			}
		})

		t.Run("Undefined types", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: T = 5 as R
            `)

			require.Error(t, err)

			errors := ExpectCheckerErrors(t, err, 2)
			assert.IsType(t, &sema.NotDeclaredError{}, errors[0])
			assert.IsType(t, &sema.NotDeclaredError{}, errors[1])

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
		})

		t.Run("with generics", func(t *testing.T) {
			t.Parallel()

			typeParameter := &sema.TypeParameter{
				Name:      "T",
				TypeBound: nil,
			}

			checker, err := parseAndCheckWithTestValue(t, `
                let res = test<[Int8]>([1, 2, 3] as [Int8])
                `,
				&sema.FunctionType{
					TypeParameters: []*sema.TypeParameter{
						typeParameter,
					},
					Parameters: []*sema.Parameter{
						{
							Label:      sema.ArgumentLabelNotRequired,
							Identifier: "value",
							TypeAnnotation: sema.NewTypeAnnotation(
								&sema.GenericType{
									TypeParameter: typeParameter,
								},
							),
						},
					},
					ReturnTypeAnnotation:  sema.NewTypeAnnotation(sema.VoidType),
					RequiredArgumentCount: nil,
				},
			)

			require.NoError(t, err)
			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
		})
	})

	t.Run("Same type as expression", func(t *testing.T) {
		t.Parallel()

		t.Run("String", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = "hello" as String
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.StringType, cast.TargetType)
			}
		})

		t.Run("Bool", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = true as Bool
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.BoolType, cast.TargetType)
			}
		})

		t.Run("Nil", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = nil as Never?
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.OptionalType{
					Type: sema.NeverType,
				}, cast.TargetType)
			}
		})

		t.Run("Without expected type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int8 = 5
                let y = x as Int8      // Not OK
                let z = x as Integer   // OK - 'Integer' is used as the variable type
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Int8Type, cast.ExprActualType)
			}
		})

		t.Run("With expected type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int8 = 5
                let y: AnyStruct = x as Int8      // Not OK
                let z: AnyStruct = x as Integer   // OK
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Int8Type, cast.ExprActualType)
			}
		})

		t.Run("With invalid expected type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int8 = 5
                let y: String = x as Int8
            `)

			require.Error(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Int8Type, cast.TargetType)
			}
		})

		t.Run("Int literal with expected type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: AnyStruct = 4 as Int8      // OK
                let y: AnyStruct = 4 as Integer   // OK
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.AnyStructType, cast.ExpectedType)
			}
		})

		t.Run("Fixed point literal", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = 4.5 as UFix64
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.UFix64Type, cast.TargetType)
			}

			checker, err = ParseAndCheckWithAny(t, `
				let y = -4.5 as Fix64
			`)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Fix64Type, cast.TargetType)
			}
		})

		t.Run("Array, with literals", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = [5, 6, 7] as [Int]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.VariableSizedType{
					Type: sema.IntType,
				}, cast.TargetType)
			}
		})

		t.Run("Array, with literals, inferred", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = [5, 6, 7] as [UInt8]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.VariableSizedType{
					Type: sema.UInt8Type,
				}, cast.TargetType)
			}
		})

		t.Run("Array, all elements self typed", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let a: Int8 = 5
                let b: Int8 = 6
                let c: Int8 = 7
                let x = [a, b, c] as [Int8]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.VariableSizedType{
					Type: sema.Int8Type,
				}, cast.TargetType)
			}
		})

		t.Run("Array, invalid typed elements", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let a: Int8 = 5
                let b: Int8 = 6
                let c: Int8 = 7
                let x = [a, b, c] as [String]
            `)

			require.Error(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 0)
		})

		t.Run("Nested array, all elements self typed", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let a: Int8 = 5
                let b: Int8 = 6
                let c: Int8 = 7
                let x = [[a, b], [a, c], [b, c]] as [[Int8]]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.VariableSizedType{
					Type: &sema.VariableSizedType{
						Type: sema.Int8Type,
					},
				}, cast.TargetType)
			}
		})

		t.Run("Nested array, one element inferred", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let a: Int8 = 5
                let b: Int8 = 6
                let x = [[a, b], [a, 7]] as [[Int8]]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.VariableSizedType{
					Type: &sema.VariableSizedType{
						Type: sema.Int8Type,
					},
				}, cast.TargetType)
			}
		})

		t.Run("Dictionary, invalid typed entries", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let a: Int8 = 5
                let b: Int8 = 6
                let c: Int8 = 7
                let x = {a: b, b: c, c: a} as {Int8: String}
            `)

			require.Error(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 0)
		})

		t.Run("Nested dictionary, all entries self typed", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let a: Int8 = 5
                let b: Int8 = 6
                let c: Int8 = 7
                let x = {a: {a: b}, b: {b: a}, c: {c: b}} as {Int8:  {Int8: Int8}}
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.DictionaryType{
					KeyType: sema.Int8Type,
					ValueType: &sema.DictionaryType{
						KeyType:   sema.Int8Type,
						ValueType: sema.Int8Type,
					},
				}, cast.TargetType)
			}
		})

		t.Run("Nested dictionary, one element inferred", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let a: Int8 = 5
                let b: Int8 = 6
                let x = {a: {a: b}, b: {b: 7}} as {Int8:  {Int8: Int8}}
                let y = {a: {a: b}, b: {7: a}} as {Int8:  {Int8: Int8}}
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.DictionaryType{
					KeyType: sema.Int8Type,
					ValueType: &sema.DictionaryType{
						KeyType:   sema.Int8Type,
						ValueType: sema.Int8Type,
					},
				}, cast.TargetType)
			}
		})

		t.Run("Reference, without type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Bool = false
                let y = &x as &Bool
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 0)
		})

		t.Run("Reference, with type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Bool = false
                let y: &Bool = &x as &Bool 
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 0)
		})

		t.Run("Conditional expr valid", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
		       let x = (true ? 5.4 : nil) as UFix64?
		   `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.OptionalType{
					Type: sema.UFix64Type,
				}, cast.TargetType)
			}
		})

		t.Run("Conditional expr invalid", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
		       let x = (true ? 5.4 : nil) as Fix64?
		   `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, &sema.OptionalType{
					Type: sema.Fix64Type,
				}, cast.TargetType)
			}
		})

		t.Run("Conditional expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
		       let x = (true ? 5.4 : 3.4) as UFix64
		   `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.UFix64Type, cast.TargetType)
			}
		})

		t.Run("Invocation", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = foo() as UInt

                fun foo(): UInt {
                    return 3
                }
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.UIntType, cast.TargetType)
			}
		})

		t.Run("Member access", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = Foo()
                let y = x.bar as String

                struct Foo {
                    pub var bar: String

                    init() {
                        self.bar = "hello"
                    }
                }
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.StringType, cast.TargetType)
			}
		})

		t.Run("Index access", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: [Int] = [1, 4, 6]
                let y = x[0] as Int
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.IntType, cast.TargetType)
			}
		})

		t.Run("Create expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x <- create Foo() as @Foo

                resource Foo {}
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.IsType(t, &sema.CompositeType{}, cast.TargetType)
				compositeType := cast.TargetType.(*sema.CompositeType)
				assert.Equal(t, "Foo", compositeType.Identifier)
			}
		})

		t.Run("Force expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int? = 5
                let y = x! as Int
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.IntType, cast.TargetType)
			}
		})

		t.Run("Path expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: CapabilityPath = /public/foo as PublicPath
                let y = /public/foo as PublicPath
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.PublicPathType, cast.TargetType)
			}
		})

		t.Run("Unary expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = !true as Bool
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.BoolType, cast.TargetType)
			}

			checker, err = ParseAndCheckWithAny(t, `
                let y: Fix64 = 5.0
                let z = -y as Fix64
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.Equal(t, sema.Fix64Type, cast.TargetType)
			}
		})

		t.Run("Binary expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = (1 + 2) as Int     // supposed to be redundant
                let y = (1 + 2) as Int8    // ok
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 2)
		})

		t.Run("Function expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x =
                    fun (_ x: Int): Int {
                        return x * 2
                    } as ((Int): Int)
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.StaticCastTypes, 1)
			for _, cast := range checker.Elaboration.StaticCastTypes { // nolint:maprangecheck
				assert.IsType(t, &sema.FunctionType{}, cast.TargetType)
			}
		})
	})
}

func TestCastResourceAsEnumAsEmptyDict(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "resource as { enum x : as { } }")

	errs := ExpectCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	assert.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[1])
}

//

func TestCastNumbersManyTimesThenGetType(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "let a = 0x0 as UInt64!as?UInt64!as?UInt64?!?.getType()")

	assert.Nil(t, err)
}
