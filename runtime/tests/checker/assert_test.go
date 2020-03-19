package checker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dapperlabs/cadence/runtime/sema"
	"github.com/dapperlabs/cadence/runtime/stdlib"
	. "github.com/dapperlabs/cadence/runtime/tests/utils"
)

func TestCheckAssertWithoutMessage(t *testing.T) {

	_, err := ParseAndCheckWithOptions(t,
		`
            pub fun test() {
                assert(1 == 2)
            }
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(
					stdlib.StandardLibraryFunctions{
						stdlib.AssertFunction,
					}.ToValueDeclarations(),
				),
			},
		},
	)

	require.NoError(t, err)
}

func TestCheckAssertWithMessage(t *testing.T) {

	_, err := ParseAndCheckWithOptions(t,
		`
            pub fun test() {
                assert(1 == 2, message: "test message")
            }
        `,
		ParseAndCheckOptions{
			Options: []sema.Option{
				sema.WithPredeclaredValues(
					stdlib.StandardLibraryFunctions{
						stdlib.AssertFunction,
					}.ToValueDeclarations(),
				),
			},
		},
	)

	require.NoError(t, err)
}
