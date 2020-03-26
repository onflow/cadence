package checker

import (
	"fmt"
	"testing"

	"github.com/dapperlabs/cadence/runtime/sema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckCapability(t *testing.T) {

	t.Run("type annotation", func(t *testing.T) {

		checker, err := ParseAndCheckWithPanic(t, `
          let x: Capability = panic("")
        `)

		require.NoError(t, err)

		assert.IsType(t,
			&sema.CapabilityType{},
			checker.GlobalValues["x"].Type,
		)
	})

	t.Run("borrowing: missing type argument", func(t *testing.T) {

		_, err := ParseAndCheckWithPanic(t, `

          let capability: Capability = panic("")

          let r = capability.borrow()
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeParameterTypeInferenceError{}, errs[0])
	})

	for _, auth := range []bool{false, true} {

		authKeyword := ""
		if auth {
			authKeyword = "auth"
		}

		testName := fmt.Sprintf(
			"borrowing: explicit type argument, %s reference",
			authKeyword,
		)

		t.Run(testName, func(t *testing.T) {

			checker, err := ParseAndCheckWithPanic(t,
				fmt.Sprintf(
					`
                      resource R {}

                      let capability: Capability = panic("")

                      let r = capability.borrow<%s &R>()
                    `,
					authKeyword,
				),
			)

			require.NoError(t, err)

			rType := checker.GlobalTypes["R"].Type

			rValueType := checker.GlobalValues["r"].Type

			require.Equal(t,
				&sema.OptionalType{
					Type: &sema.ReferenceType{
						Authorized: auth,
						Type:       rType,
					},
				},
				rValueType,
			)
		})
	}

	t.Run("borrowing: explicit type argument, non-reference type", func(t *testing.T) {

		_, err := ParseAndCheckWithPanic(t, `

          resource R {}

          let capability: Capability = panic("")

          let r <- capability.borrow<@R>()
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		require.IsType(t, &sema.TypeMismatchError{}, errs[0])
	})
}
