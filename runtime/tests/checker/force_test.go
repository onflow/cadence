package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/sema"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckForce(t *testing.T) {

	t.Run("valid", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
          let x: Int? = 1
          let y = x!
        `)

		require.NoError(t, err)

		assert.Equal(t,
			&sema.OptionalType{Type: &sema.IntType{}},
			checker.GlobalValues["x"].Type,
		)

		assert.Equal(t,
			&sema.IntType{},
			checker.GlobalValues["y"].Type,
		)

	})

	t.Run("invalid: non-optional", func(t *testing.T) {

		checker, err := ParseAndCheck(t, `
          let x: Int = 1
          let y = x!
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.NonOptionalForceError{}, errs[0])

		assert.Equal(t,
			&sema.IntType{},
			checker.GlobalValues["x"].Type,
		)

		assert.Equal(t,
			&sema.IntType{},
			checker.GlobalValues["y"].Type,
		)
	})

	t.Run("invalid: force resource multiple times", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          resource R {}

          let x: @R? <- create R()
          let x2 <- x!
          let x3 <- x!
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.ResourceUseAfterInvalidationError{}, errs[0])
	})
}
