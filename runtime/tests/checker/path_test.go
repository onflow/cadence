package checker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/common"
	"github.com/dapperlabs/cadence/runtime/sema"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckPath(t *testing.T) {

	for _, domain := range common.AllPathDomainsByIdentifier {

		t.Run(fmt.Sprintf("valid: %s", domain.Name()), func(t *testing.T) {

			checker, err := ParseAndCheck(t,
				fmt.Sprintf(
					`
                      let x: Path = /%[1]s/foo
                      let y = /%[1]s/bar
                    `,
					domain.Identifier(),
				),
			)

			require.NoError(t, err)

			assert.IsType(t,
				&sema.PathType{},
				checker.GlobalValues["x"].Type,
			)
		})
	}

	t.Run("invalid: unsupported domain", func(t *testing.T) {

		_, err := ParseAndCheck(t, `
          let x = /wrong/random
        `)

		errs := ExpectCheckerErrors(t, err, 1)

		assert.IsType(t, &sema.InvalidPathDomainError{}, errs[0])
	})
}
