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

	errs := RequireCheckerErrors(t, err, 1)

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
}

func TestCheckCastingResourceToAnyResource(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t, `
      resource R {}

      fun test() {
          let r <- create R()
          let x <- r as @AnyResource
          destroy x
      }
    `)

	require.NoError(t, err)
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

	// Supertype: Intersection type

	t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @{I1, I2} <- create R()
                  let r2 <- r as @{I2}
                `,
			)

			require.NoError(t, err)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.IntersectionType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I2}? {
                      let r: @{I1, I2} <- create R()
                      if let r2 <- r as? @{I2} {
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

	t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1} <- create R()
                  let r2 <- r as @{I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I1, I2}? {
                      let r: @{I1} <- create R()
                      if let r2 <- r as? @{I1, I2} {
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

	t.Run("intersection type -> intersection type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I} <- create R1()
                  let r2 <- r as @{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I}? {
                      let r: @{I} <- create R1()
                      if let r2 <- r as? @{I} {
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

	t.Run("type -> intersection type: same resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @{I}
                `,
			)

			require.NoError(t, err)

			r2Type := RequireGlobalValue(t, checker.Elaboration, "r2")

			require.IsType(t,
				&sema.IntersectionType{},
				r2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @{I} {
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

	t.Run("type -> intersection type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R1: I {}

          resource R2: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R1 <- create R1()
                  let r2 <- r as @{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I}? {
                      let r: @R1 <- create R1()
                      if let r2 <- r as? @{I} {
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

	t.Run("AnyResource -> conforming intersection type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @{RI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{RI}? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @{RI} {
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

	t.Run("intersection -> conforming intersection type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{RI} <- create R()
                  let r2 <- r as @{RI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{RI}? {
                      let r: @{RI} <- create R()
                      if let r2 <- r as? @{RI} {
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

	t.Run("intersection -> non-conforming intersection type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{RI} <- create R()
                  let r2 <- r as @{RI}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{RI}? {
                      let r: @{RI} <- create R()
                      if let r2 <- r as? @{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])

		})
	})

	// Supertype: Resource

	t.Run("intersection type -> type: same resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I} <- create R()
                  let r2 <- r as @R
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @{I} <- create R()
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

	t.Run("intersection type -> type: different resource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}

          resource T: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I} <- create R()
                  let t <- r as @T
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @T? {
                      let r: @{I} <- create R()
                      if let t <- r as? @T {
                          return <-t
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

	t.Run("intersection AnyResource -> conforming resource", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{RI} <- create R()
                  let r2 <- r as @R
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @{RI} <- create R()
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

	t.Run("intersection AnyResource -> non-conforming resource", func(t *testing.T) {

		const types = `
           resource interface RI {}

           resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{RI} <- create R()
                  let r2 <- r as @R
                `,
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @R? {
                      let r: @{RI} <- create R()
                      if let r2 <- r as? @R {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyResource -> type", func(t *testing.T) {

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

			errs := RequireCheckerErrors(t, err, 1)

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

	// Supertype: intersection AnyResource

	t.Run("resource -> intersection AnyResource with non-conformance type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          // NOTE: R does not conform to RI
          resource R {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @{RI}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{RI}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @{RI} {
                          return <-r2
                      } else {
                          destroy r
                          return nil
                      }
                  }
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

	})

	t.Run("resource -> intersection AnyResource with conformance type", func(t *testing.T) {

		const types = `
          resource interface RI {}

          resource R: RI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @R <- create R()
                  let r2 <- r as @{RI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{RI}? {
                      let r: @R <- create R()
                      if let r2 <- r as? @{RI} {
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

	t.Run("intersection type -> intersection AnyResource with conformance in type", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I} <- create R()
                  let r2 <- r as @{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I}? {
                      let r: @{I} <- create R()
                      if let r2 <- r as? @{I} {
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

	t.Run("intersection type -> intersection with conformance not in type", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1} <- create R()
                  let r2 <- r as @{I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I2}? {
                      let r: @{I1} <- create R()
                      if let r2 <- r as? @{I2} {
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

	t.Run("intersection type -> intersection AnyResource with non-conformance type", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1} <- create R()
                  let r2 <- r as @{I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I2}? {
                      let r: @{I1} <- create R()
                      if let r2 <- r as? @{I2} {
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

	t.Run("intersection AnyResource -> intersection AnyResource: fewer types", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1, I2} <- create R()
                  let r2 <- r as @{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I2}? {
                      let r: @{I1, I2} <- create R()
                      if let r2 <- r as? @{I2} {
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

	t.Run("intersection AnyResource -> intersection AnyResource: more types", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1} <- create R()
                  let r2 <- r as @{I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I1, I2}? {
                      let r: @{I1} <- create R()
                      if let r2 <- r as? @{I1, I2} {
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

	t.Run("intersection AnyResource -> intersection AnyResource with non-conformance type", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1} <- create R()
                  let r2 <- r as @{I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I1, I2}? {
                      let r: @{I1} <- create R()
                      if let r2 <- r as? @{I1, I2} {
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

	t.Run("AnyResource -> intersection AnyResource", func(t *testing.T) {

		const types = `
          resource interface I {}

          resource R: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @AnyResource <- create R()
                  let r2 <- r as @{I}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @{I}? {
                      let r: @AnyResource <- create R()
                      if let r2 <- r as? @{I} {
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

	t.Run("intersection type -> AnyResource", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1} <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r: @{I1} <- create R()
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

	t.Run("intersection AnyResource -> AnyResource", func(t *testing.T) {

		const types = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let r: @{I1} <- create R()
                  let r2 <- r as @AnyResource
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  fun test(): @AnyResource? {
                      let r: @{I1} <- create R()
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

	t.Run("type -> AnyResource", func(t *testing.T) {

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

	// Supertype: Intersection type

	t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: {I1, I2} = S()
                  let s2 = s as {I2}
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.IntersectionType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: {I1, I2} = S()
                  let s2 = s as? {I2}
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.OptionalType{
					Type: &sema.IntersectionType{},
				},
				s2Type,
			)
		})
	})

	t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as {I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as? {I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("type -> intersection type", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as {I}
                `,
			)

			require.NoError(t, err)

			s2Type := RequireGlobalValue(t, checker.Elaboration, "s2")

			require.IsType(t,
				&sema.IntersectionType{},
				s2Type,
			)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? {I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyStruct -> conforming intersection type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as {SI}
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? {SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection -> conforming intersection type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {SI} = S()
                  let s2 = s as {SI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {SI} = S()
                  let s2 = s as? {SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: Struct

	t.Run("intersection -> conforming struct", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {SI} = S()
                  let s2 = s as S
                `,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {SI} = S()
                  let s2 = s as? S
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection AnyStruct -> non-conforming struct", func(t *testing.T) {

		const types = `
           struct interface SI {}

           struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {SI} = S()
                  let s2 = s as S
                `,
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {SI} = S()
                  let s2 = s as? S
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("AnyStruct -> type", func(t *testing.T) {

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

			errs := RequireCheckerErrors(t, err, 1)

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

	// Supertype: intersection AnyStruct

	t.Run("struct -> intersection AnyStruct with non-conformance type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          // NOTE: S does not conform to SI
          struct S {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as {SI}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? {SI}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

	})

	t.Run("struct -> intersection AnyStruct with conformance type", func(t *testing.T) {

		const types = `
          struct interface SI {}

          struct S: SI {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as {SI}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: S = S()
                  let s2 = s as? {SI}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection with non-conformance type", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as {I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as? {I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection -> intersection: fewer types", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1, I2} = S()
                  let s2 = s as {I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1, I2} = S()
                  let s2 = s as? {I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection AnyStruct -> intersection AnyStruct: more types", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as {I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as? {I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection AnyStruct -> intersection AnyStruct with non-conformance type", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as {I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as? {I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("AnyStruct -> intersection AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I {}

          struct S: I {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as {I}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: AnyStruct = S()
                  let s2 = s as? {I}
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: AnyStruct

	t.Run("intersection type -> AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection AnyStruct -> AnyStruct", func(t *testing.T) {

		const types = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as AnyStruct
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				types+`
                  let s: {I1} = S()
                  let s2 = s as? AnyStruct
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("type -> AnyStruct", func(t *testing.T) {

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
						  entitlement X
                          resource interface I {}

                          resource R: I {}

                          let r <- create R()
                          let ref = &r as auth(X) &%[1]s
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
						  entitlement X

                          let r <- create R()
                          let ref = &r as &%[1]s
                          let ref2 = ref as auth(X) &%[1]s
                        `,
						ty,
					),
				)

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		}

		for _, ty := range []string{
			"R",
			"AnyResource",
			"{I}",
			"Any",
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
						  entitlement X

                          let s = S()
                          let ref = &s as auth(X) &%[1]s
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
						  entitlement X

                          let s = S()
                          let ref = &s as &%[1]s
                          let ref2 = ref as auth(X) &%[1]s
                        `,
						ty,
					),
				)

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})
		}

		for _, ty := range []string{
			"S",
			"AnyStruct",
			"{I}",
			"Any",
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
						  entitlement X
                          let i = 1
                          let ref = &i as auth(X) &%[1]s
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
						  entitlement X
                          let i = 1
                          let ref = &i as &%[1]s
                          let ref2 = ref as auth(X) &%[1]s
                        `,
						ty,
					),
				)

				errs := RequireCheckerErrors(t, err, 1)

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

	// Supertype: Intersection type

	t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
		  entitlement X

          let x <- create R()
          let r = &x as auth(X) &{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

		const setup = `
          resource interface I1 {}

          resource interface I2 {}

          resource R: I1, I2 {}
		  entitlement X

          let x <- create R()
          let r = &x as auth(X) &{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &{I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("type -> intersection type: same resource", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}
		  entitlement X

          let x <- create R()
          let r = &x as auth(X) &R
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as? &{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	// Supertype: Resource

	t.Run("intersection type -> type", func(t *testing.T) {

		const setup = `
          resource interface I {}

          resource R: I {}
		  entitlement X

          let x <- create R()
          let r = &x as auth(X) &{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let r2 = r as &R
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
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

	t.Run("intersection -> conforming resource", func(t *testing.T) {

		setup :=
			`
			  resource interface RI {}

			  resource R: RI {}
			  entitlement X

			  let x <- create R()
			  let r = &x as auth(X) &{RI}
			`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+`
				  let r2 = r as &R
				`,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := RequireCheckerErrors(t, err, 1)

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

	t.Run("intersection -> non-conforming resource", func(t *testing.T) {

		setup :=
			`
			  resource interface RI {}

			  resource R {}
			  entitlement X

			  let x <- create R()
			  let r = &x as auth(X) &{RI}
			`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+`
				  let r2 = r as &R
				`,
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+`
				  let r2 = r as? &R
				`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("resource -> intersection with non-conformance type", func(t *testing.T) {

		const setup = `
		  resource interface RI {}

		  // NOTE: R does not conform to RI
		  resource R {}
		  entitlement X

		  let x <- create R()
		  let r = &x as auth(X) &R
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as &{RI}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as? &{RI}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("resource -> intersection with conformance type", func(t *testing.T) {

		const setup = `
		  resource interface RI {}

		  resource R: RI {}
		  entitlement X

		  let x <- create R()
		  let r = &x as auth(X) &R
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as &{RI}
					`,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as? &{RI}
					`,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection with conformance in type", func(t *testing.T) {

		const setup = `
		  resource interface I {}

		  resource R: I {}
		  entitlement X

		  let x <- create R()
		  let r = &x as auth(X) &{I}
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as &{I}
					`,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as? &{I}
					`,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection with conformance not in type", func(t *testing.T) {

		const setup = `
		  resource interface I1 {}

		  resource interface I2 {}

		  resource R: I1, I2 {}
		  entitlement X

		  let x <- create R()
		  let r = &x as auth(X) &{I1}
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as &{I2}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as? &{I2}
					`,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection with non-conformance type", func(t *testing.T) {

		const setup = `
		  resource interface I1 {}

		  resource interface I2 {}

		  resource R: I1 {}
		  entitlement X

		  let x <- create R()
		  let r = &x as auth(X) &{I1}
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as &{I2}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let r2 = r as? &{I2}
					`,
			)

			require.NoError(t, err)
		})
	})

	for _, ty := range []sema.Type{
		sema.AnyResourceType,
		sema.AnyType,
	} {

		t.Run(fmt.Sprintf("%s -> type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  resource interface RI {}

                  resource R: RI {}
				  entitlement X

                  let x <- create R()
                  let r = &x as auth(X) &%s
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let r2 = r as &R
                    `,
				)

				errs := RequireCheckerErrors(t, err, 1)

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

		t.Run(fmt.Sprintf("intersection type -> %s", ty), func(t *testing.T) {

			const setup = `
              resource interface I1 {}

              resource interface I2 {}

              resource R: I1, I2 {}
			  entitlement X

              let x <- create R()
              let r = &x as auth(X) &{I1}
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

		t.Run(fmt.Sprintf("type -> %s", ty), func(t *testing.T) {

			const setup = `
              resource interface I1 {}

              resource interface I2 {}

              resource R: I1, I2 {}
			  entitlement X

              let x <- create R()
              let r = &x as auth(X) &R
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

	// Supertype: Intersection type

	t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
		  entitlement X

          let x = S()
          let s = &x as auth(X) &{I1, I2}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &{I2}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &{I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

		const setup = `
          struct interface I1 {}

          struct interface I2 {}

          struct S: I1, I2 {}
		  entitlement X

          let x = S()
          let s = &x as auth(X) &{I1}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &{I1, I2}
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &{I1, I2}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S1: I {}

          struct S2: I {}
		  entitlement X

          let x = S1()
          let s = &x as auth(X) &{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("type -> intersection type: same struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}
		  entitlement X

          let x = S()
          let s = &x as auth(X) &S

        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &{I}
                `,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as? &{I}
                `,
			)

			require.NoError(t, err)
		})
	})

	for _, ty := range []sema.Type{
		sema.AnyStructType,
		sema.AnyType,
	} {

		t.Run(fmt.Sprintf("%s -> conforming intersection type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface SI {}

                  struct S: SI {}
				  entitlement X

                  let x = S()
                  let s = &x as auth(X) &%s
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &{SI}
                    `,
				)

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("dynamic", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as? &{SI}
                    `,
				)

				require.NoError(t, err)
			})
		})

	}

	// Supertype: Struct

	t.Run("intersection type -> type: same struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}
		  entitlement X

          let x = S()
          let s = &x as auth(X) &{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let s2 = s as &S
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
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

	t.Run("intersection type -> type: different struct", func(t *testing.T) {

		const setup = `
          struct interface I {}

          struct S: I {}

          struct T: I {}
		  entitlement X

          let x = S()
          let s = &x as auth(X) &{I}
        `

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = s as &T
                `,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheck(t,
				setup+`
                  let t = s as? &T
                `,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection -> conforming struct", func(t *testing.T) {

		setup :=
			`
			  struct interface RI {}

			  struct S: RI {}
			  entitlement X

			  let x = S()
			  let s = &x as auth(X) &{RI}
			`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+`
				  let s2 = s as &S
				`,
			)

			// NOTE: static cast not allowed, only dynamic

			errs := RequireCheckerErrors(t, err, 1)

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

	t.Run("intersection -> non-conforming struct", func(t *testing.T) {

		setup :=
			`
			  struct interface RI {}

			  struct S {}
			  entitlement X

			  let x = S()
			  let s = &x as auth(X) &{RI}
			`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+`
				  let s2 = s as &S
				`,
			)

			errs := RequireCheckerErrors(t, err, 2)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+`
				  let s2 = s as? &S
				`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("struct -> intersection with non-conformance type", func(t *testing.T) {

		const setup = `
		  struct interface SI {}

		  // NOTE: S does not conform to SI
		  struct S {}
		  entitlement X

		  let x = S()
		  let s = &x as auth(X) &S
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as &{SI}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as? &{SI}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})
	})

	t.Run("struct -> intersection with conformance type", func(t *testing.T) {

		const setup = `
		  struct interface SI {}

		  struct S: SI {}
		  entitlement X

		  let x = S()
		  let s = &x as auth(X) &S
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as &{SI}
					`,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as? &{SI}
					`,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection with conformance in type", func(t *testing.T) {

		const setup = `
		  struct interface I {}

		  struct S: I {}

		  entitlement X

		  let x = S()
		  let s = &x as auth(X) &{I}
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as &{I}
					`,
			)

			require.NoError(t, err)
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as? &{I}
					`,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection with conformance not in type", func(t *testing.T) {

		const setup = `
		  struct interface I1 {}

		  struct interface I2 {}

		  struct S: I1, I2 {}

		  entitlement X

		  let x = S()
		  let s = &x as auth(X) &{I1}
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as &{I2}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as? &{I2}
					`,
			)

			require.NoError(t, err)
		})
	})

	t.Run("intersection type -> intersection with non-conformance type", func(t *testing.T) {

		const setup = `
		  struct interface I1 {}

		  struct interface I2 {}

		  struct S: I1 {}

		  entitlement X

		  let x = S()
		  let s = &x as auth(X) &{I1}
		`

		t.Run("static", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as &{I2}
					`,
			)

			errs := RequireCheckerErrors(t, err, 1)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
		})

		t.Run("dynamic", func(t *testing.T) {

			_, err := ParseAndCheckWithAny(t,
				setup+
					`
					  let s2 = s as? &{I2}
					`,
			)

			require.NoError(t, err)
		})
	})

	for _, ty := range []sema.Type{
		sema.AnyStructType,
		sema.AnyType,
	} {

		t.Run(fmt.Sprintf("%s -> type", ty), func(t *testing.T) {

			setup := fmt.Sprintf(
				`
                  struct interface SI {}

                  struct S: SI {}
				  entitlement X

                  let x = S()
                  let s = &x as auth(X) &%s
                `,
				ty,
			)

			t.Run("static", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					setup+`
                      let s2 = s as &S
                    `,
				)

				errs := RequireCheckerErrors(t, err, 1)

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

	}
}

func TestCheckCastUnauthorizedResourceReferenceType(t *testing.T) {

	t.Parallel()

	for name, op := range map[string]string{
		"static":  "as",
		"dynamic": "as?",
	} {

		t.Run(name, func(t *testing.T) {

			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &{I1, I2}
                          let r2 = r %s &{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I1 {}

                          resource interface I2 {}

                          resource R: I1, I2 {}

                          let x <- create R()
                          let r = &x as &{I1}
                          let r2 = r %s &{I1, I2}
                        `,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("type -> intersection type: same resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &R
                          let r2 = r %s &{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("intersection -> conforming intersection type", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  resource interface RI {}

						  resource R: RI {}

						  let x <- create R()
						  let r = &x as &{RI}
						  let r2 = r %s &{RI}
						`,
						op,
					),
				)

				require.NoError(t, err)
			})

			for _, ty := range []sema.Type{
				sema.AnyResourceType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("%s -> conforming intersection type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface RI {}

                              resource R: RI {}

                              let x <- create R()
                              let r = &x as &%s
                              let r2 = r %s &{RI}
                            `,
							ty,
							op,
						),
					)

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})
			}

			// Supertype: Resource

			t.Run("intersection type -> type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          let x <- create R()
                          let r = &x as &{I}
                          let r2 = r %s &R
                        `,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("intersection type -> type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          resource interface I {}

                          resource R: I {}

                          resource T: I {}

                          let x <- create R()
                          let r = &x as &{I}
                          let t = r %s &T
                        `,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("intersection -> conforming resource", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  resource interface RI {}

						  resource R: RI {}

						  let x <- create R()
						  let r = &x as &{RI}
						  let r2 = r %s &R
						`,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("intersection -> non-conforming resource", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  resource interface RI {}

						  resource R {}

						  let x <- create R()
						  let r = &x as &{RI}
						  let r2 = r %s &R
						`,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 2)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
				} else {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})

			for _, ty := range []sema.Type{
				sema.AnyResourceType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("%s -> type", ty), func(t *testing.T) {

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

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})

				t.Run("intersection type -> intersection with conformance not in type", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface I1 {}

                              resource interface I2 {}

                              resource R: I1, I2 {}

                              let x <- create R()
                              let r = &x as &{I1}
                              let r2 = r %s &{I2}
                            `,
							op,
						),
					)

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})

				t.Run("intersection -> intersection with non-conformance type", func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
							  resource interface I1 {}

							  resource interface I2 {}

							  resource R: I1 {}

							  let x <- create R()
							  let r = &x as &{I1}
							  let r2 = r %s &{I1, I2}
							`,
							op,
						),
					)

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})

				t.Run(fmt.Sprintf("%s -> intersection", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
							  resource interface I {}

							  resource R: I {}

							  let x <- create R()
							  let r = &x as &%s
							  let r2 = r %s &{I}
							`,
							ty,
							op,
						),
					)

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})

				t.Run(fmt.Sprintf("intersection -> %s", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
							  resource interface I1 {}

							  resource interface I2 {}

							  resource R: I1, I2 {}

							  let x <- create R()
							  let r = &x as &{I1}
							  let r2 = r %s &%s
							`,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("intersection type -> %s", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              resource interface I1 {}

                              resource interface I2 {}

                              resource R: I1, I2 {}

                              let x <- create R()
                              let r = &x as &{I1}
                              let r2 = r %s &%s
                            `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("type -> %s", ty), func(t *testing.T) {

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

			// Supertype: Intersection type

			t.Run("intersection type -> intersection type: fewer types", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &{I1, I2}
                          let s2 = s %s &{I2}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("intersection type -> intersection type: more types", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I1 {}

                          struct interface I2 {}

                          struct S: I1, I2 {}

                          let x = S()
                          let s = &x as &{I1}
                          let s2 = s %s &{I1, I2}
                        `,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("type -> intersection type: same resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &S
                          let s2 = s %s &{I}
                        `,
						op,
					),
				)

				require.NoError(t, err)
			})

			for _, ty := range []sema.Type{
				sema.AnyStructType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("%s -> conforming intersection type", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
                              struct interface RI {}

                              struct S: RI {}

                              let x = S()
                              let s = &x as &%s
                              let s2 = s %s &{RI}
                            `,
							ty,
							op,
						),
					)

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})
			}

			// Supertype: Resource

			t.Run("intersection type -> type", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          let x = S()
                          let s = &x as &{I}
                          let s2 = s %s &S
                        `,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("intersection type -> type: different resource", func(t *testing.T) {

				_, err := ParseAndCheck(t,
					fmt.Sprintf(
						`
                          struct interface I {}

                          struct S: I {}

                          struct T: I {}

                          let x = S()
                          let s = &x as &{I}
                          let t = s %s &T
                        `,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("intersection -> conforming resource", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  struct interface RI {}

						  struct S: RI {}

						  let x = S()
						  let s = &x as &{RI}
						  let s2 = s %s &S
						`,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("intersection -> non-conforming resource", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  struct interface RI {}

						  struct S {}

						  let x = S()
						  let s = &x as &{RI}
						  let s2 = s %s &S
						`,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 2)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
				} else {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				}
			})

			// Supertype: intersection AnyStruct / Any

			t.Run("resource -> intersection with non-conformance type", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
                              struct interface RI {}

                              // NOTE: R does not conform to RI
                              struct S {}

                              let x = S()
                              let s = &x as &S
                              let s2 = s %s &{RI}
                            `,
						op,
					),
				)

				errs := RequireCheckerErrors(t, err, 1)

				assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			})

			t.Run("resource -> intersection with conformance type", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  struct interface RI {}

						  struct S: RI {}

						  let x = S()
						  let s = &x as &S
						  let s2 = s %s &{RI}
						`,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("intersection -> intersection: fewer types", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  struct interface I1 {}

						  struct interface I2 {}

						  struct S: I1, I2 {}

						  let x = S()
						  let s = &x as &{I1, I2}
						  let s2 = s %s &{I2}
						`,
						op,
					),
				)

				require.NoError(t, err)
			})

			t.Run("intersection -> intersection: more types", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  struct interface I1 {}

						  struct interface I2 {}

						  struct S: I1, I2 {}

						  let x = S()
						  let s = &x as &{I1}
						  let s2 = s %s &{I1, I2}
						`,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			t.Run("intersection -> intersection %s with non-conformance type", func(t *testing.T) {

				_, err := ParseAndCheckWithAny(t,
					fmt.Sprintf(
						`
						  struct interface I1 {}

						  struct interface I2 {}

						  struct S: I1 {}

						  let x = S()
						  let s = &x as &{I1}
						  let s2 = s %s &{I1, I2}
						`,
						op,
					),
				)

				if name == "static" {
					errs := RequireCheckerErrors(t, err, 1)

					assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
				} else {
					require.NoError(t, err)
				}
			})

			for _, ty := range []sema.Type{
				sema.AnyStructType,
				sema.AnyType,
			} {

				t.Run(fmt.Sprintf("%s -> type", ty), func(t *testing.T) {

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

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})

				t.Run(fmt.Sprintf("%s -> intersection", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
							  struct interface I {}

							  struct S: I {}

							  let x = S()
							  let s = &x as &%s
							  let s2 = s %s &{I}
							`,
							ty,
							op,
						),
					)

					if name == "static" {
						errs := RequireCheckerErrors(t, err, 1)

						assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
					} else {
						require.NoError(t, err)
					}
				})

				// Supertype: AnyStruct / Any

				t.Run(fmt.Sprintf("intersection -> %s", ty), func(t *testing.T) {

					_, err := ParseAndCheckWithAny(t,
						fmt.Sprintf(
							`
							 struct interface I1 {}

							 struct interface I2 {}

							 struct S: I1, I2 {}

							 let x = S()
							 let s = &x as &{I1}
							 let s2 = s %s &%s
						   `,
							op,
							ty,
						),
					)

					require.NoError(t, err)
				})

				t.Run(fmt.Sprintf("type -> %s", ty), func(t *testing.T) {

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

          let c = R as fun(): @R
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

	assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
}

func TestCheckResourceConstructorReturn(t *testing.T) {

	t.Parallel()

	_, err := ParseAndCheck(t,
		`
          resource R {}

          fun test(): fun(): @R {
              return R
          }
        `,
	)

	errs := RequireCheckerErrors(t, err, 1)

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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.Int8Type, cast.TargetType)
			}
		})

		t.Run("Binary exp", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int8 = (1 as Int8) + (1 as Int8)
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.Int8Type, cast.TargetType)
			}
		})

		t.Run("Nested casts", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = (1 as Int8) as Int8
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.Int8Type, cast.TargetType)
			}
		})

		t.Run("Arrays", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: [Character] = ["c" as Character]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.CharacterType, cast.TargetType)
			}
		})

		t.Run("Dictionaries", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: {String: UInt8} = {"foo": 4 as UInt8}
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.UInt8Type, cast.TargetType)
			}
		})

		t.Run("Undefined types", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: T = 5 as R
            `)

			errors := RequireCheckerErrors(t, err, 2)
			assert.IsType(t, &sema.NotDeclaredError{}, errors[0])
			assert.IsType(t, &sema.NotDeclaredError{}, errors[1])

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
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
					Parameters: []sema.Parameter{
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
					ReturnTypeAnnotation: sema.VoidTypeAnnotation,
				},
			)

			require.NoError(t, err)
			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.StringType, cast.TargetType)
			}
		})

		t.Run("Bool", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = true as Bool
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.BoolType, cast.TargetType)
			}
		})

		t.Run("Nil", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = nil as Never?
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.Int8Type, cast.ExprActualType)
			}
		})

		t.Run("With invalid expected type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Int8 = 5
                let y: String = x as Int8
            `)

			RequireCheckerErrors(t, err, 1)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.AnyStructType, cast.ExpectedType)
			}
		})

		t.Run("Fixed point literal", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = 4.5 as UFix64
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.UFix64Type, cast.TargetType)
			}

			checker, err = ParseAndCheckWithAny(t, `
				let y = -4.5 as Fix64
			`)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.Fix64Type, cast.TargetType)
			}
		})

		t.Run("Array, with literals", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = [5, 6, 7] as [Int]
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			errs := RequireCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 0)
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			errs := RequireCheckerErrors(t, err, 3)

			assert.IsType(t, &sema.TypeMismatchError{}, errs[0])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[1])
			assert.IsType(t, &sema.TypeMismatchError{}, errs[2])

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 0)
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, &sema.DictionaryType{
					KeyType: sema.Int8Type,
					ValueType: &sema.DictionaryType{
						KeyType:   sema.Int8Type,
						ValueType: sema.Int8Type,
					},
				}, cast.TargetType)
			}
		})

		t.Run("Reference, with cast", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Bool = false
                let y = &x as &Bool
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
		})

		t.Run("Reference, with type", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x: Bool = false
                let y: &Bool = &x as &Bool 
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
		})

		t.Run("Conditional expr valid", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
		       let x = (true ? 5.4 : nil) as UFix64?
		   `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.UIntType, cast.TargetType)
			}
		})

		t.Run("Member access", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = Foo()
                let y = x.bar as String

                struct Foo {
                    access(all) var bar: String

                    init() {
                        self.bar = "hello"
                    }
                }
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.PublicPathType, cast.TargetType)
			}
		})

		t.Run("Unary expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x = !true as Bool
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.Equal(t, sema.BoolType, cast.TargetType)
			}

			checker, err = ParseAndCheckWithAny(t, `
                let y: Fix64 = 5.0
                let z = -y as Fix64
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
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

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 2)
		})

		t.Run("Function expr", func(t *testing.T) {
			t.Parallel()

			checker, err := ParseAndCheckWithAny(t, `
                let x =
                    fun (_ x: Int): Int {
                        return x * 2
                    } as fun(Int): Int
            `)

			require.NoError(t, err)

			require.Len(t, checker.Elaboration.AllStaticCastTypes(), 1)
			for _, cast := range checker.Elaboration.AllStaticCastTypes() { // nolint:maprange
				assert.IsType(t, &sema.FunctionType{}, cast.TargetType)
			}
		})
	})
}

func TestCastResourceAsEnumAsEmptyDict(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "resource foo { enum x : foo { } }")

	errs := RequireCheckerErrors(t, err, 2)

	assert.IsType(t, &sema.InvalidNestedDeclarationError{}, errs[0])
	assert.IsType(t, &sema.InvalidEnumRawTypeError{}, errs[1])
}

//

func TestCastNumbersManyTimesThenGetType(t *testing.T) {
	t.Parallel()

	_, err := ParseAndCheck(t, "let a = 0x0 as UInt64!as?UInt64!as?UInt64?!?.getType()")

	assert.Nil(t, err)
}
